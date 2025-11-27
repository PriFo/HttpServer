package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/internal/domain/models"

	_ "github.com/mattn/go-sqlite3"
)

// Алиасы для обратной совместимости
type SystemSummary = models.SystemSummary
type UploadSummary = models.UploadSummary
type ScanAlert = models.ScanAlert

// ScanAndSummarizeAllDatabases сканирует все базы данных и формирует сводный отчет
func ScanAndSummarizeAllDatabases(ctx context.Context, serviceDBPath, mainDBPath string) (*SystemSummary, error) {
	startTime := time.Now()

	// Создаем контекст с таймаутом если не передан
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
	}

	log.Printf("[SystemScanner] Начало сканирования системы. serviceDB=%s, mainDB=%s", serviceDBPath, mainDBPath)

	summary := &models.SystemSummary{
		UploadDetails: []models.UploadSummary{},
		LastActivity:  time.Time{},
	}

	// Подключаемся к сервисной БД
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к service.db: %w", err)
	}
	defer serviceDB.Close()

	// Подключаемся к основной БД
	mainDB, err := database.NewDB(mainDBPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к основной БД: %w", err)
	}
	defer mainDB.Close()

	// Получаем список всех загрузок из основной БД
	uploads, err := mainDB.GetAllUploads()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список загрузок: %w", err)
	}

	summary.TotalUploads = int64(len(uploads))
	if summary.TotalUploads == 0 {
		return summary, nil
	}

	// Подсчитываем статистику по статусам и находим последнюю активность
	uniqueDatabaseIDs := make(map[int]bool)
	var lastActivity time.Time

	for _, upload := range uploads {
		// Подсчет по статусам
		switch upload.Status {
		case "completed":
			summary.CompletedUploads++
		case "failed":
			summary.FailedUploads++
		case "in_progress":
			summary.InProgressUploads++
		}

		// Отслеживаем уникальные БД
		if upload.DatabaseID != nil {
			uniqueDatabaseIDs[*upload.DatabaseID] = true
		}

		// Находим последнюю активность
		if !upload.StartedAt.IsZero() && upload.StartedAt.After(lastActivity) {
			lastActivity = upload.StartedAt
		}
		if upload.CompletedAt != nil && !upload.CompletedAt.IsZero() && upload.CompletedAt.After(lastActivity) {
			lastActivity = *upload.CompletedAt
		}
	}

	summary.TotalDatabases = len(uniqueDatabaseIDs)
	summary.LastActivity = lastActivity

	// Предзагружаем пути к БД для оптимизации (кешируем пути, чтобы не запрашивать их повторно)
	dbPathCache := make(map[int]string)
	if len(uniqueDatabaseIDs) > 0 {
		for dbID := range uniqueDatabaseIDs {
			projectDB, err := serviceDB.GetProjectDatabase(dbID)
			if err != nil {
				log.Printf("Предупреждение: не удалось получить информацию о БД %d: %v", dbID, err)
			} else if projectDB != nil {
				dbPathCache[dbID] = projectDB.FilePath
			}
		}
	}

	// Обрабатываем загрузки параллельно с ограничением на количество одновременных подключений
	const maxConcurrentDBs = 5
	semaphore := make(chan struct{}, maxConcurrentDBs)
	var wg sync.WaitGroup
	var mu sync.Mutex // Для синхронизации доступа к summary

	// Создаем контекст с таймаутом для каждой БД (5 секунд на БД)
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Обрабатываем каждую загрузку
	for _, upload := range uploads {
		uploadSummary := models.UploadSummary{
			ID:         fmt.Sprintf("%d", upload.ID),
			UploadUUID: upload.UploadUUID,
			Name:       upload.ConfigName,
			Status:     upload.Status,
			CreatedAt:  upload.StartedAt,
			DatabaseID: upload.DatabaseID,
			ClientID:   upload.ClientID,
			ProjectID:  upload.ProjectID,
		}

		if upload.CompletedAt != nil {
			uploadSummary.CompletedAt = upload.CompletedAt
		}

		// Если у загрузки есть database_id, получаем путь к файлу БД из кэша
		var dbFilePath string
		if upload.DatabaseID != nil {
			if path, ok := dbPathCache[*upload.DatabaseID]; ok {
				dbFilePath = path
				uploadSummary.DatabaseFile = dbFilePath
			} else {
				mu.Lock()
				summary.DatabasesSkipped++
				mu.Unlock()
			}
		}

		// Если файл БД найден, обрабатываем параллельно
		if dbFilePath != "" {
			wg.Add(1)
			semaphore <- struct{}{} // Занимаем слот

			// Сохраняем указатель на uploadSummary для доступа в горутине
			uploadSummaryPtr := &uploadSummary

			go func(upload *database.Upload, uploadSummary *UploadSummary, dbPath string, systemSummary *SystemSummary) {
				defer wg.Done()
				defer func() { <-semaphore }() // Освобождаем слот

				// Обработка паники в горутине
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[SystemScanner] Panic при обработке БД %s для загрузки %s: %v", dbPath, uploadSummary.UploadUUID, r)
						mu.Lock()
						systemSummary.DatabasesSkipped++
						mu.Unlock()
					}
				}()

				nomenclatureCount, counterpartyCount, dbSize, err := countDatabaseRecords(dbCtx, dbPath)
				if err != nil {
					// Классифицируем тип ошибки для лучшего логирования
					errStr := err.Error()
					errorType := "unknown"
					if strings.Contains(errStr, "не найден") || strings.Contains(errStr, "not found") {
						errorType = "file_not_found"
					} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
						errorType = "timeout"
					} else if strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
						errorType = "connection"
					}

					log.Printf("[SystemScanner] Предупреждение: не удалось просканировать БД %s для загрузки %s [error_type: %s]: %v",
						dbPath, uploadSummary.UploadUUID, errorType, err)

					mu.Lock()
					systemSummary.DatabasesSkipped++
					mu.Unlock()
					// Продолжаем обработку, даже если не удалось подсчитать записи
				} else {
					uploadSummary.NomenclatureCount = nomenclatureCount
					uploadSummary.CounterpartyCount = counterpartyCount
					if dbSize > 0 {
						uploadSummary.DatabaseSize = &dbSize
					}

					// Безопасно обновляем общие счетчики
					mu.Lock()
					systemSummary.TotalNomenclature += nomenclatureCount
					systemSummary.TotalCounterparties += counterpartyCount
					systemSummary.DatabasesProcessed++
					mu.Unlock()
				}
			}(upload, uploadSummaryPtr, dbFilePath, summary)
		}

		summary.UploadDetails = append(summary.UploadDetails, uploadSummary)
	}

	// Ждем завершения всех горутин
	wg.Wait()

	// Вычисляем метрики производительности
	duration := time.Since(startTime)
	durationStr := duration.Round(time.Millisecond).String()
	summary.ScanDuration = &durationStr

	log.Printf("[SystemScanner] Сканирование завершено за %s. Обработано БД: %d, пропущено: %d, всего загрузок: %d, номенклатуры: %d, контрагентов: %d",
		durationStr, summary.DatabasesProcessed, summary.DatabasesSkipped, summary.TotalUploads, summary.TotalNomenclature, summary.TotalCounterparties)

	return summary, nil
}

// countDatabaseRecords подсчитывает количество записей в таблицах nomenclature_items и counterparties
// Возвращает также размер файла БД в байтах
func countDatabaseRecords(ctx context.Context, dbFilePath string) (nomenclatureCount, counterpartyCount int64, dbSize int64, err error) {
	// Проверяем существование файла и получаем размер
	fileInfo, err := os.Stat(dbFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, 0, 0, fmt.Errorf("файл БД не найден: %s", dbFilePath)
		}
		return 0, 0, 0, fmt.Errorf("не удалось получить информацию о файле БД: %w", err)
	}
	dbSize = fileInfo.Size()

	// Открываем базу данных
	conn, err := sql.Open("sqlite3", dbFilePath+"?_timeout=5000")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("не удалось открыть БД: %w", err)
	}
	defer conn.Close()

	// Настраиваем таймаут для запросов
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(30 * time.Second)

	// Проверяем подключение
	if err := conn.PingContext(ctx); err != nil {
		return 0, 0, 0, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	// Подсчитываем номенклатуру
	// Проверяем наличие таблицы nomenclature_items
	var tableExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='nomenclature_items'
		)
	`).Scan(&tableExists)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("не удалось проверить наличие таблицы nomenclature_items: %w", err)
	}

	if tableExists {
		err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM nomenclature_items").Scan(&nomenclatureCount)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("не удалось подсчитать номенклатуру: %w", err)
		}
	}

	// Подсчитываем контрагентов
	// Проверяем наличие таблицы counterparties
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='counterparties'
		)
	`).Scan(&tableExists)
	if err != nil {
		// Если таблицы нет, это не ошибка - просто будет 0
		log.Printf("Предупреждение: не удалось проверить наличие таблицы counterparties в %s: %v", dbFilePath, err)
	} else if tableExists {
		err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM counterparties").Scan(&counterpartyCount)
		if err != nil {
			log.Printf("Предупреждение: не удалось подсчитать контрагентов в %s: %v", dbFilePath, err)
			// Продолжаем, даже если не удалось подсчитать
		}
	} else {
		// Если таблицы counterparties нет, проверяем normalized_data или catalog_items
		// Проверяем normalized_data
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name='normalized_data'
			)
		`).Scan(&tableExists)
		if err == nil && tableExists {
			// В normalized_data могут быть контрагенты, если они нормализованы
			// Проверяем наличие колонки, которая указывает на тип данных
			var hasTypeColumn bool
			err = conn.QueryRowContext(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM pragma_table_info('normalized_data')
					WHERE name='data_type' OR name='type'
				)
			`).Scan(&hasTypeColumn)
			if err == nil && hasTypeColumn {
				// Если есть колонка типа, считаем только контрагентов
				err = conn.QueryRowContext(ctx, `
					SELECT COUNT(*) FROM normalized_data 
					WHERE (data_type = 'counterparty' OR type = 'counterparty')
				`).Scan(&counterpartyCount)
				if err != nil {
					// Если не получилось с фильтром, считаем все записи
					conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM normalized_data").Scan(&counterpartyCount)
				}
			} else {
				// Если нет колонки типа, считаем все записи как потенциальные контрагенты
				conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM normalized_data").Scan(&counterpartyCount)
			}
		} else {
			// Проверяем catalog_items (может содержать как номенклатуру, так и контрагентов)
			err = conn.QueryRowContext(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM sqlite_master 
					WHERE type='table' AND name='catalog_items'
				)
			`).Scan(&tableExists)
			if err == nil && tableExists {
				// Определяем тип данных по каталогу или имени файла
				dataType := ""
				
				// Сначала пробуем определить по таблице catalogs
				var catalogsTableExists bool
				err = conn.QueryRowContext(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM sqlite_master 
						WHERE type='table' AND name='catalogs'
					)
				`).Scan(&catalogsTableExists)
				
				if err == nil && catalogsTableExists {
					// Проверяем, есть ли каталог "Номенклатура"
					var nomenclatureCatalogExists bool
					err = conn.QueryRowContext(ctx, `
						SELECT EXISTS (
							SELECT 1 FROM catalogs 
							WHERE name = 'Номенклатура' OR name LIKE '%оменклатур%'
						)
					`).Scan(&nomenclatureCatalogExists)
					if err == nil && nomenclatureCatalogExists {
						dataType = "nomenclature"
					} else {
						// Проверяем, есть ли каталог "Контрагенты"
						var counterpartyCatalogExists bool
						err = conn.QueryRowContext(ctx, `
							SELECT EXISTS (
								SELECT 1 FROM catalogs 
								WHERE name = 'Контрагенты' OR name LIKE '%онтрагент%'
							)
						`).Scan(&counterpartyCatalogExists)
						if err == nil && counterpartyCatalogExists {
							dataType = "counterparties"
						}
					}
				}
				
				// Если не удалось определить по каталогу, используем имя файла
				if dataType == "" {
					fileName := filepath.Base(dbFilePath)
					fileInfo := database.ParseDatabaseFileInfo(fileName)
					dataType = fileInfo.DataType
				}
				
				// Подсчитываем записи в зависимости от типа данных
				var catalogItemsCount int64
				err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_items").Scan(&catalogItemsCount)
				if err != nil {
					log.Printf("Предупреждение: не удалось подсчитать catalog_items в %s: %v", dbFilePath, err)
				} else {
					if dataType == "nomenclature" {
						// Считаем как номенклатуру
						nomenclatureCount = catalogItemsCount
					} else {
						// По умолчанию считаем как контрагентов
						counterpartyCount = catalogItemsCount
					}
				}
			}
		}
	}

	return nomenclatureCount, counterpartyCount, dbSize, nil
}
