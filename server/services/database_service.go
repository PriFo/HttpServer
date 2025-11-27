package services

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"httpserver/database"
	apperrors "httpserver/server/errors"
)

// DatabaseService сервис для работы с базами данных
type DatabaseService struct {
	serviceDB    *database.ServiceDB
	db           *database.DB
	normalizedDB *database.DB
	currentDBPath           string
	currentNormalizedDBPath string
	dbInfoCache  interface{} // DatabaseInfoCache - будет определен позже
	logger       interface{} // Logger - для логирования (опционально)
	// Callback для обновления БД в Server (опционально)
	onDBUpdate   func(newDB *database.DB, newPath string) error
}

// NewDatabaseService создает новый сервис для работы с базами данных
func NewDatabaseService(
	serviceDB *database.ServiceDB,
	db *database.DB,
	normalizedDB *database.DB,
	currentDBPath string,
	currentNormalizedDBPath string,
	dbInfoCache interface{},
) *DatabaseService {
	return &DatabaseService{
		serviceDB:              serviceDB,
		db:                     db,
		normalizedDB:           normalizedDB,
		currentDBPath:          currentDBPath,
		currentNormalizedDBPath: currentNormalizedDBPath,
		dbInfoCache:            dbInfoCache,
	}
}

// GetDatabaseInfo возвращает информацию о текущей базе данных
func (s *DatabaseService) GetDatabaseInfo() (map[string]interface{}, error) {
	if s.db == nil {
		return nil, apperrors.NewInternalError("база данных недоступна", nil)
	}

	info := map[string]interface{}{
		"current_db_path":            s.currentDBPath,
		"current_normalized_db_path": s.currentNormalizedDBPath,
	}

	// Получаем информацию о файле, если путь доступен
	if s.currentDBPath != "" {
		if fileInfo, err := os.Stat(s.currentDBPath); err == nil {
			// Извлекаем имя файла из пути
			info["name"] = filepath.Base(s.currentDBPath)
			info["path"] = s.currentDBPath
			info["size"] = fileInfo.Size()
			info["modified_at"] = fileInfo.ModTime().Format(time.RFC3339)
			info["status"] = "connected"
		} else {
			// Файл не существует, но путь указан
			info["name"] = filepath.Base(s.currentDBPath)
			info["path"] = s.currentDBPath
			info["size"] = int64(0)
			info["modified_at"] = ""
			info["status"] = "disconnected"
		}
	} else {
		// Путь не указан
		info["name"] = ""
		info["path"] = ""
		info["size"] = int64(0)
		info["modified_at"] = ""
		info["status"] = "disconnected"
	}

	// Получаем общую статистику из БД
	// Делаем это безопасно, чтобы не ломать основной функционал при ошибках
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Игнорируем панику - это не критично для получения общей информации
			}
		}()
		
		stats, err := s.db.GetStats()
		if err == nil && stats != nil {
			info["stats"] = stats
		}
		// Игнорируем ошибки получения статистики - это не критично
	}()

	// Получаем статистику из uploads для текущей БД, если можем определить database_id
	// Делаем это безопасно, чтобы не ломать основной функционал, если БД не найдена в serviceDB
	if s.serviceDB != nil && s.currentDBPath != "" && s.db != nil {
		// Находим database_id по пути к файлу
		// Используем recover для защиты от паники
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Игнорируем панику - это не критично для получения общей информации
				}
			}()
			
			_, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(s.currentDBPath)
			if err == nil && projectID > 0 {
				// Получаем базу данных проекта по пути
				dbInfo, err := s.serviceDB.GetProjectDatabaseByPath(projectID, s.currentDBPath)
				if err == nil && dbInfo != nil {
					// Получаем статистику из uploads для этой БД
					uploadStats, err := s.db.GetUploadStatsByDatabaseID(dbInfo.ID)
					if err == nil && uploadStats != nil {
						info["upload_stats"] = uploadStats
					}
					// Игнорируем ошибки получения статистики - это не критично
				}
			}
			// Игнорируем ошибки поиска БД - это не критично для получения общей информации
		}()
	}

	return info, nil
}

// GetCurrentDBPath возвращает путь к текущей базе данных
func (s *DatabaseService) GetCurrentDBPath() string {
	return s.currentDBPath
}

// GetDB возвращает указатель на текущую базу данных
func (s *DatabaseService) GetDB() *database.DB {
	return s.db
}

// GetServiceDB возвращает указатель на сервисную базу данных
func (s *DatabaseService) GetServiceDB() *database.ServiceDB {
	return s.serviceDB
}

// GetAggregatedUploadStats возвращает агрегированную статистику по выгрузкам для всех баз данных
func (s *DatabaseService) GetAggregatedUploadStats() (map[string]interface{}, error) {
	if s.db == nil {
		return nil, apperrors.NewInternalError("база данных недоступна", nil)
	}
	
	return s.db.GetAggregatedUploadStats()
}

// ListDatabases возвращает список всех баз данных с информацией из таблицы uploads
func (s *DatabaseService) ListDatabases() ([]*database.ProjectDatabase, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	// Получаем все базы данных из serviceDB
	projectDatabases, err := s.serviceDB.GetAllProjectDatabases()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить список баз данных проекта", err)
	}

	// Если нет основной БД для получения статистики, возвращаем базы данных без статистики
	if s.db == nil {
		return projectDatabases, nil
	}

	// Для каждой базы данных получаем статистику из таблицы uploads
	// Примечание: статистика будет добавлена через расширение структуры ProjectDatabase
	// или через отдельный метод, который форматирует данные для фронтенда
	// Сейчас возвращаем базы данных как есть, статистика будет добавлена в обработчике
	return projectDatabases, nil
}

// SearchDatabases ищет базы данных по имени, пути или описанию
func (s *DatabaseService) SearchDatabases(searchQuery string) ([]*database.ProjectDatabase, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	// Если поисковый запрос пустой, возвращаем все базы данных
	if searchQuery == "" {
		return s.ListDatabases()
	}

	// Получаем все базы данных
	allDatabases, err := s.serviceDB.GetAllProjectDatabases()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить список баз данных", err)
	}

	// Фильтруем по поисковому запросу (поиск в имени и пути)
	searchLower := strings.ToLower(searchQuery)
	var filteredDatabases []*database.ProjectDatabase

	for _, db := range allDatabases {
		// Проверяем имя
		if strings.Contains(strings.ToLower(db.Name), searchLower) {
			filteredDatabases = append(filteredDatabases, db)
			continue
		}

		// Проверяем путь к файлу
		if db.FilePath != "" && strings.Contains(strings.ToLower(db.FilePath), searchLower) {
			filteredDatabases = append(filteredDatabases, db)
			continue
		}

		// Проверяем описание
		if db.Description != "" && strings.Contains(strings.ToLower(db.Description), searchLower) {
			filteredDatabases = append(filteredDatabases, db)
		}
	}

	return filteredDatabases, nil
}

// FindDatabase ищет базу данных по имени или пути
// Использует SearchDatabases для поиска
func (s *DatabaseService) FindDatabase(query string) ([]*database.ProjectDatabase, error) {
	return s.SearchDatabases(query)
}

// FindProjectByDatabase находит проект по базе данных
func (s *DatabaseService) FindProjectByDatabase(dbPath string) (*database.ProjectDatabaseWithProject, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}
	// Используем существующий метод FindClientAndProjectByDatabasePath
	clientID, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(dbPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("проект не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось найти проект", err)
	}
	
	// Получаем БД проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("проект клиента не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить проект", err)
	}
	
	// Получаем БД по пути
	db, err := s.serviceDB.GetProjectDatabaseByPath(projectID, dbPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("база данных не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить базу данных", err)
	}
	
	// Получаем клиента
	client, err := s.serviceDB.GetClient(clientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("клиент не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить клиента", err)
	}
	
	// Создаем структуру с проектом
	return &database.ProjectDatabaseWithProject{
		ProjectDatabase: *db,
		Project: &database.ClientProjectWithClient{
			ClientProject: *project,
			Client:        client,
		},
	}, nil
}

// GetDatabaseAnalytics возвращает аналитику по базе данных
func (s *DatabaseService) GetDatabaseAnalytics(databaseID int) (map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	dbInfo, err := s.serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("база данных не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить базу данных", err)
	}

	if dbInfo == nil {
		return nil, apperrors.NewNotFoundError("база данных не найдена", nil)
	}

	analytics := map[string]interface{}{
		"database_id":       dbInfo.ID,
		"database_name":    dbInfo.Name,
		"file_path":         dbInfo.FilePath,
		"client_project_id": dbInfo.ClientProjectID,
		"created_at":        dbInfo.CreatedAt,
		"updated_at":        dbInfo.UpdatedAt,
	}

	// Если указан путь к файлу, получаем полную аналитику через database.GetDatabaseAnalytics
	if dbInfo.FilePath != "" {
		// Проверяем существование файла
		if _, err := os.Stat(dbInfo.FilePath); err == nil {
			// Получаем полную аналитику из файла БД
			fullAnalytics, err := database.GetDatabaseAnalytics(dbInfo.FilePath)
			if err == nil && fullAnalytics != nil {
				// Объединяем данные
				analytics["database_type"] = fullAnalytics.DatabaseType
				analytics["total_size"] = fullAnalytics.TotalSize
				analytics["total_size_mb"] = fullAnalytics.TotalSizeMB
				analytics["table_count"] = fullAnalytics.TableCount
				analytics["total_rows"] = fullAnalytics.TotalRows
				analytics["table_stats"] = fullAnalytics.TableStats
				analytics["top_tables"] = fullAnalytics.TopTables
				analytics["analyzed_at"] = fullAnalytics.AnalyzedAt

				// Обновляем историю изменений
				if s.serviceDB != nil {
					if err := database.UpdateDatabaseHistory(s.serviceDB, dbInfo.FilePath, fullAnalytics.TotalSize, fullAnalytics.TotalRows); err != nil {
						// Логируем ошибку, но не прерываем выполнение
						slog.Warn("[GetDatabaseAnalytics] Failed to update database history",
							"database_id", databaseID,
							"file_path", dbInfo.FilePath,
							"error", err,
						)
					} else {
						slog.Info("[GetDatabaseAnalytics] Database history updated",
							"database_id", databaseID,
							"file_path", dbInfo.FilePath,
							"total_size", fullAnalytics.TotalSize,
							"total_rows", fullAnalytics.TotalRows,
						)
					}
				}
			}
			// Игнорируем ошибки получения аналитики из файла - это не критично
		}
	}

	// Получаем статистику из uploads, если доступна основная БД
	if s.db != nil {
		stats, err := s.db.GetUploadStatsByDatabaseID(databaseID)
		if err == nil && stats != nil {
			// Если статистика нулевая, пытаемся получить из файла БД проекта
			totalUploads, _ := stats["total_uploads"].(int)
			totalCatalogs, _ := stats["total_catalogs"].(int)
			totalItems, _ := stats["total_items"].(int)
			if totalUploads == 0 && totalCatalogs == 0 && totalItems == 0 && dbInfo.FilePath != "" {
				fileStats, fileErr := database.GetUploadStatsFromDatabaseFile(dbInfo.FilePath)
				if fileErr == nil && fileStats != nil {
					// Используем статистику из файла БД, если она больше нуля
					fileUploads, _ := fileStats["total_uploads"].(int)
					fileCatalogs, _ := fileStats["total_catalogs"].(int)
					fileItems, _ := fileStats["total_items"].(int)
					if fileUploads > 0 || fileCatalogs > 0 || fileItems > 0 {
						slog.Info("[GetDatabaseAnalytics] Using stats from database file",
							"database_id", databaseID,
							"file_path", dbInfo.FilePath,
						)
						stats = fileStats
					}
				}
			}
			analytics["upload_stats"] = stats
			analytics["stats"] = stats // Для совместимости с frontend
		} else if err != nil {
			// Логируем предупреждение, но не прерываем выполнение
			slog.Warn("[GetDatabaseAnalytics] Failed to get upload stats",
				"database_id", databaseID,
				"error", err,
			)
		}
	}

	return analytics, nil
}

// GetDatabaseHistory возвращает историю изменений базы данных
func (s *DatabaseService) GetDatabaseHistory(databaseID int) ([]map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	// Получаем информацию о базе данных
	dbInfo, err := s.serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("база данных не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить базу данных", err)
	}

	if dbInfo == nil {
		return nil, apperrors.NewNotFoundError("база данных не найдена", nil)
	}

	// Если путь к файлу не указан, возвращаем пустую историю
	if dbInfo.FilePath == "" {
		return []map[string]interface{}{}, nil
	}

	// Получаем историю из метаданных через database.GetDatabaseHistory
	historyEntries, err := database.GetDatabaseHistory(s.serviceDB, dbInfo.FilePath)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить историю базы данных", err)
	}

	// Преобразуем HistoryEntry в map[string]interface{}
	history := make([]map[string]interface{}, len(historyEntries))
	for i, entry := range historyEntries {
		history[i] = map[string]interface{}{
			"database_id": databaseID,
			"timestamp":  entry.Timestamp,
			"size":       entry.Size,
			"size_mb":    entry.SizeMB,
			"row_count":  entry.RowCount,
		}
	}

	return history, nil
}

// GetPendingDatabases возвращает список ожидающих баз данных
func (s *DatabaseService) GetPendingDatabases(statusFilter string) ([]*database.PendingDatabase, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}
	return s.serviceDB.GetPendingDatabases(statusFilter)
}

// GetPendingDatabase возвращает ожидающую базу данных по ID
func (s *DatabaseService) GetPendingDatabase(id int) (*database.PendingDatabase, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}
	return s.serviceDB.GetPendingDatabase(id)
}

// CleanupPendingDatabases очищает старые ожидающие базы данных.
// Удаляет pending databases, которые старше указанного количества дней и не были привязаны к клиентам/проектам.
//
// Параметры:
//   - olderThanDays: количество дней, после которых базы данных считаются старыми
//
// Возвращает:
//   - количество удаленных записей
//   - ошибку при неудаче
func (s *DatabaseService) CleanupPendingDatabases(olderThanDays int) (int, error) {
	if s.serviceDB == nil {
		return 0, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}
	
	if olderThanDays < 0 {
		return 0, apperrors.NewValidationError("количество дней не может быть отрицательным", nil)
	}
	
	deleted, err := s.serviceDB.CleanupOldPendingDatabases(olderThanDays)
	if err != nil {
		return 0, apperrors.NewInternalError("не удалось очистить старые pending databases", err)
	}
	
	return deleted, nil
}

// ScanForDatabaseFiles сканирует файловую систему на наличие .db файлов
func (s *DatabaseService) ScanForDatabaseFiles(paths []string) ([]map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	// Используем функцию из database_scanner.go через рефлексию или создаем обертку
	// Для простоты, реализуем логику напрямую
	var foundFiles []map[string]interface{}
	patterns := []string{"Выгрузка_Номенклатура_", "Выгрузка_Контрагенты_"}

	for _, scanPath := range paths {
		if _, err := os.Stat(scanPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				slog.Warn("[ScanForDatabaseFiles] Path does not exist, skipping",
					"path", scanPath,
				)
				continue
			}
			// Другие ошибки тоже пропускаем с предупреждением
			slog.Warn("[ScanForDatabaseFiles] Failed to check path, skipping",
				"path", scanPath,
				"error", err,
			)
			continue
		}

		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Пропускаем ошибки доступа
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(path), ".db") {
				return nil
			}

			fileName := filepath.Base(path)
			matchesPattern := false
			for _, pattern := range patterns {
				if strings.HasPrefix(fileName, pattern) {
					matchesPattern = true
					break
				}
			}

			if matchesPattern {
				absPath, err := filepath.Abs(path)
				if err != nil {
					slog.Warn("[ScanForDatabaseFiles] Failed to get absolute path",
						"path", path,
						"error", err,
					)
					return nil
				}

				foundFiles = append(foundFiles, map[string]interface{}{
					"path": absPath,
					"name": fileName,
					"size": info.Size(),
				})

				// Добавляем в pending_databases, если еще нет
				_, err = s.serviceDB.GetPendingDatabaseByPath(absPath)
				if err != nil {
					// Файл еще не в базе, добавляем
					_, createErr := s.serviceDB.CreatePendingDatabase(absPath, fileName, info.Size())
					if createErr != nil {
						slog.Warn("[ScanForDatabaseFiles] Failed to add file to pending databases",
							"path", absPath,
							"error", createErr,
						)
					} else {
						slog.Info("[ScanForDatabaseFiles] Added file to pending databases",
							"path", absPath,
						)
					}
				}
			}

			return nil
		})

		if err != nil {
			slog.Error("[ScanForDatabaseFiles] Error scanning path",
				"path", scanPath,
				"error", err,
			)
		}
	}

	return foundFiles, nil
}

// GetDatabaseFiles возвращает список всех .db файлов
func (s *DatabaseService) GetDatabaseFiles() ([]map[string]interface{}, error) {
	// Защищенные файлы, которые нельзя удалять
	protectedFiles := map[string]bool{
		"service.db":         true,
		"1c_data.db":         true,
		"data.db":            true,
		"normalized_data.db": true,
	}

	var allFiles []map[string]interface{}

	// Сканируем основные директории
	scanPaths := []string{
		".",
		"data",
		"data/uploads",
		"/app",
		"/app/data",
		"/app/data/uploads",
	}

	fileMap := make(map[string]bool) // Для дедупликации

	for _, scanPath := range scanPaths {
		if _, err := os.Stat(scanPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			// Другие ошибки тоже пропускаем
			continue
		}

		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Пропускаем ошибки доступа
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(path), ".db") {
				return nil
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil
			}

			// Дедупликация
			if fileMap[absPath] {
				return nil
			}
			fileMap[absPath] = true

			fileName := filepath.Base(absPath)
			isProtected := protectedFiles[fileName]

			// Определяем тип файла
			fileType := "other"
			if isProtected {
				if fileName == "service.db" {
					fileType = "service"
				} else {
					fileType = "main"
				}
			} else if strings.Contains(absPath, "uploads") || strings.Contains(absPath, "data/uploads") {
				fileType = "uploaded"
			} else if strings.Contains(absPath, "data") {
				fileType = "main"
			}

			fileInfo := map[string]interface{}{
				"path":        absPath,
				"name":        fileName,
				"size":        info.Size(),
				"modified_at": info.ModTime(),
				"type":        fileType,
				"is_protected": isProtected,
				"linked_to_project": false,
			}

			// Обновляем метаданные с информацией о конфигурации 1С, если их еще нет
			if s.serviceDB != nil {
				metadata, err := s.serviceDB.GetDatabaseMetadata(absPath)
				if err == nil && (metadata == nil || metadata.MetadataJSON == "" || !strings.Contains(metadata.MetadataJSON, "config_name")) {
					// Определяем тип базы данных
					dbType := "unknown"
					if detectedType, err := database.DetectDatabaseType(absPath); err == nil {
						dbType = detectedType
					}
					// Обновляем метаданные с конфигурацией
					if err := s.updateDatabaseMetadataWithConfig(absPath, dbType); err != nil {
						slog.Warn("[GetDatabaseFiles] Failed to update metadata", "path", absPath, "error", err)
					}
				}

				// Проверяем, связан ли файл с проектом
				clientID, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(absPath)
				if err == nil && clientID > 0 && projectID > 0 {
					fileInfo["linked_to_project"] = true
					fileInfo["client_id"] = clientID
					fileInfo["project_id"] = projectID

					// Получаем информацию о проекте
					project, err := s.serviceDB.GetClientProject(projectID)
					if err == nil {
						fileInfo["project_name"] = project.Name
					}

					// Получаем ID базы данных в проекте
					projectDB, err := s.serviceDB.GetProjectDatabaseByPath(projectID, absPath)
					if err == nil && projectDB != nil {
						fileInfo["database_id"] = projectDB.ID
					}
				} else {
					// Добавляем информацию о конфигурации из метаданных
					metadata, err := s.serviceDB.GetDatabaseMetadata(absPath)
					if err == nil && metadata != nil && metadata.MetadataJSON != "" {
						var metadataMap map[string]interface{}
						if err := json.Unmarshal([]byte(metadata.MetadataJSON), &metadataMap); err == nil {
							if configName, ok := metadataMap["config_name"].(string); ok && configName != "" {
								fileInfo["config_name"] = configName
							}
							if displayName, ok := metadataMap["display_name"].(string); ok && displayName != "" {
								fileInfo["display_name"] = displayName
							}
						}
					}
				}
			}

			allFiles = append(allFiles, fileInfo)
			return nil
		})

		if err != nil {
			slog.Error("[GetDatabaseFiles] Error scanning path",
				"path", scanPath,
				"error", err,
			)
		}
	}

	return allFiles, nil
}

// SetOnDBUpdate устанавливает callback для обновления БД в Server
func (s *DatabaseService) SetOnDBUpdate(callback func(newDB *database.DB, newPath string) error) {
	s.onDBUpdate = callback
}

// BackupDatabase создает резервную копию базы данных
func (s *DatabaseService) BackupDatabase(dbPath string, backupDir string) (string, error) {
	// Проверяем, что файл существует
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", apperrors.NewNotFoundError("файл базы данных не найден", err)
		}
		return "", apperrors.NewInternalError("не удалось проверить файл базы данных", err)
	}

	// Создаем директорию для резервных копий, если её нет
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", apperrors.NewInternalError("не удалось создать директорию для резервных копий", err)
	}

	// Генерируем имя файла резервной копии
	baseName := filepath.Base(dbPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("%s_backup_%s%s", nameWithoutExt, timestamp, ext)
	backupPath := filepath.Join(backupDir, backupFileName)

	// Копируем файл
	sourceFile, err := os.Open(dbPath)
	if err != nil {
		return "", apperrors.NewInternalError("не удалось открыть исходный файл", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(backupPath)
	if err != nil {
		return "", apperrors.NewInternalError("не удалось создать файл резервной копии", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		os.Remove(backupPath) // Удаляем частично созданный файл
		return "", apperrors.NewInternalError("не удалось скопировать файл", err)
	}

	slog.Info("[BackupDatabase] Created backup",
		"source", dbPath,
		"backup", backupPath,
	)

	return backupPath, nil
}

// ListBackups возвращает список резервных копий
func (s *DatabaseService) ListBackups(backupDir string) ([]map[string]interface{}, error) {
	if _, err := os.Stat(backupDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Директория не существует, возвращаем пустой список
			return []map[string]interface{}{}, nil
		}
		return nil, apperrors.NewInternalError("не удалось проверить директорию резервных копий", err)
	}

	var backups []map[string]interface{}

	err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Пропускаем ошибки доступа
		}

		if info.IsDir() {
			return nil
		}

		// Проверяем, что это файл резервной копии (.db файл)
		if !strings.HasSuffix(strings.ToLower(path), ".db") {
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}

		backupInfo := map[string]interface{}{
			"filename":   info.Name(),
			"path":       absPath,
			"size":       info.Size(),
			"created_at": info.ModTime().Format(time.RFC3339),
		}

		backups = append(backups, backupInfo)
		return nil
	})

	if err != nil {
		return nil, apperrors.NewInternalError("ошибка при сканировании директории резервных копий", err)
	}

	return backups, nil
}

// RestoreBackup восстанавливает базу данных из резервной копии
func (s *DatabaseService) RestoreBackup(backupPath string, targetPath string) error {
	// Проверяем, что файл резервной копии существует
	if _, err := os.Stat(backupPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return apperrors.NewNotFoundError("файл резервной копии не найден", err)
		}
		return apperrors.NewInternalError("не удалось проверить файл резервной копии", err)
	}

	// Создаем директорию для целевого файла, если её нет
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return apperrors.NewInternalError("не удалось создать директорию для целевого файла", err)
	}

	// Если целевой файл существует, создаем резервную копию перед восстановлением
	if _, err := os.Stat(targetPath); err == nil {
		backupBeforeRestore := targetPath + ".before_restore_" + time.Now().Format("20060102_150405")
		sourceFile, err := os.Open(targetPath)
		if err == nil {
			destFile, err := os.Create(backupBeforeRestore)
			if err == nil {
				io.Copy(destFile, sourceFile)
				destFile.Close()
				slog.Info("[RestoreBackup] Created backup before restore",
					"backup", backupBeforeRestore,
				)
			}
			sourceFile.Close()
		}
	}

	// Копируем файл резервной копии в целевой путь
	sourceFile, err := os.Open(backupPath)
	if err != nil {
		return apperrors.NewInternalError("не удалось открыть файл резервной копии", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(targetPath)
	if err != nil {
		return apperrors.NewInternalError("не удалось создать целевой файл", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return apperrors.NewInternalError("не удалось скопировать файл резервной копии", err)
	}

	slog.Info("[RestoreBackup] Successfully restored backup",
		"backup_path", backupPath,
		"target_path", targetPath,
	)

	return nil
}

// SwitchDatabase переключает текущую базу данных
func (s *DatabaseService) SwitchDatabase(dbPath string) error {
	// Проверяем, что файл существует
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return apperrors.NewNotFoundError("файл базы данных не найден", err)
		}
		return apperrors.NewInternalError("не удалось проверить файл базы данных", err)
	}

	// Создаем новое подключение к БД
	newDB, err := database.NewDB(dbPath)
	if err != nil {
		return apperrors.NewInternalError("не удалось открыть новую базу данных", err)
	}

	// Закрываем старое подключение, если оно существует
	if s.db != nil {
		if closeErr := s.db.Close(); closeErr != nil {
			slog.Warn("[SwitchDatabase] Failed to close old database connection",
				"error", closeErr,
			)
			// Не возвращаем ошибку, так как новое подключение уже создано
		}
	}

	// Обновляем внутренние поля
	oldPath := s.currentDBPath
	s.db = newDB
	s.currentDBPath = dbPath

	// Вызываем callback для обновления БД в Server, если он установлен
	if s.onDBUpdate != nil {
		if err := s.onDBUpdate(newDB, dbPath); err != nil {
			// Если callback вернул ошибку, пытаемся восстановить старое подключение
			slog.Error("[SwitchDatabase] Failed to update database in Server, attempting to restore",
				"error", err,
			)
			if oldPath != "" {
				oldDB, restoreErr := database.NewDB(oldPath)
				if restoreErr == nil {
					s.db = oldDB
					s.currentDBPath = oldPath
					newDB.Close() // Закрываем новое подключение
				}
			}
			return apperrors.NewInternalError("не удалось обновить базу данных в Server", err)
		}
	}

	slog.Info("[SwitchDatabase] Successfully switched database",
		"old_path", oldPath,
		"new_path", dbPath,
	)

	return nil
}

// BulkDeleteDatabases удаляет несколько баз данных
func (s *DatabaseService) BulkDeleteDatabases(paths []string, ids []int) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"deleted_files": 0,
		"deleted_records": 0,
		"failed_deletions": []string{},
		"errors": []string{},
	}

	deletedFiles := 0
	deletedRecords := 0
	failedDeletions := []string{}
	errorsList := []string{}

	// Удаляем файлы по путям
	for _, path := range paths {
		if path == "" {
			continue
		}

		// Проверяем, что это не защищенный файл
		fileName := filepath.Base(path)
		protectedFiles := map[string]bool{
			"service.db":         true,
			"1c_data.db":         true,
			"data.db":            true,
			"normalized_data.db": true,
		}
		if protectedFiles[fileName] {
			errorsList = append(errorsList, fmt.Sprintf("нельзя удалить защищенный файл: %s", fileName))
			continue
		}

		// Проверяем, что файл существует
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				errorsList = append(errorsList, fmt.Sprintf("файл не найден: %s", path))
			} else {
				errorsList = append(errorsList, fmt.Sprintf("ошибка проверки файла %s: %v", path, err))
			}
			continue
		}

		// Удаляем файл
		if err := os.Remove(path); err != nil {
			failedDeletions = append(failedDeletions, path)
			errorsList = append(errorsList, fmt.Sprintf("не удалось удалить файл %s: %v", path, err))
		} else {
			deletedFiles++
			slog.Info("[BulkDeleteDatabases] Deleted file",
				"path", path,
			)
		}
	}

	// Удаляем записи из serviceDB по ID
	if s.serviceDB != nil {
		for _, id := range ids {
			if id <= 0 {
				continue
			}

			// Получаем информацию о базе данных перед удалением
			db, err := s.serviceDB.GetProjectDatabase(id)
			if err != nil {
				errorsList = append(errorsList, fmt.Sprintf("не удалось получить информацию о БД с ID %d: %v", id, err))
				continue
			}
			if db == nil {
				errorsList = append(errorsList, fmt.Sprintf("база данных с ID %d не найдена", id))
				continue
			}

			// Удаляем запись из serviceDB
			if err := s.serviceDB.DeleteProjectDatabase(id); err != nil {
				errorsList = append(errorsList, fmt.Sprintf("не удалось удалить запись БД с ID %d: %v", id, err))
			} else {
				deletedRecords++
				slog.Info("[BulkDeleteDatabases] Deleted database record",
					"id", id,
					"path", db.FilePath,
				)
			}
		}
	}

	result["deleted_files"] = deletedFiles
	result["deleted_records"] = deletedRecords
	result["failed_deletions"] = failedDeletions
	result["errors"] = errorsList

	if len(errorsList) > 0 && deletedFiles == 0 && deletedRecords == 0 {
		return result, apperrors.NewInternalError("не удалось удалить ни одного файла или записи", nil)
	}

	return result, nil
}

// CreateBackup создает резервную копию баз данных
func (s *DatabaseService) CreateBackup(includeMain, includeUploads, includeService bool, selectedFiles []string, format string) (map[string]interface{}, error) {
	// Нормализуем формат
	if format != "zip" && format != "copy" && format != "both" {
		format = "both"
	}

	// Создаем директорию для бэкапов, если не существует
	backupDir := "data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, apperrors.NewInternalError("не удалось создать директорию для резервных копий", err)
	}

	// Генерируем timestamp для имени бэкапа
	timestamp := time.Now().Format("20060102_150405")

	var zipFile *os.File
	var zipWriter *zip.Writer
	var backupFileName string
	var backupPath string
	var filesCopyDir string

	// Создаем ZIP архив, если нужно
	if format == "zip" || format == "both" {
		backupFileName = fmt.Sprintf("backup_%s.zip", timestamp)
		backupPath = filepath.Join(backupDir, backupFileName)

		var err error
		zipFile, err = os.Create(backupPath)
		if err != nil {
			return nil, apperrors.NewInternalError("не удалось создать файл резервной копии", err)
		}
		defer zipFile.Close()

		zipWriter = zip.NewWriter(zipFile)
		defer zipWriter.Close()
	}

	// Создаем директорию для копий файлов, если нужно
	if format == "copy" || format == "both" {
		filesCopyDir = filepath.Join(backupDir, "files", timestamp)
		if err := os.MkdirAll(filesCopyDir, 0755); err != nil {
			return nil, apperrors.NewInternalError("не удалось создать директорию для копий файлов", err)
		}
	}

	// Собираем файлы для бэкапа
	filesToBackup := []string{}

	if len(selectedFiles) > 0 {
		// Используем выбранные файлы
		for _, filePath := range selectedFiles {
			if _, err := os.Stat(filePath); err == nil {
				filesToBackup = append(filesToBackup, filePath)
			}
		}
	} else {
		// Собираем файлы на основе флагов
		if includeMain && s.currentDBPath != "" {
			if _, err := os.Stat(s.currentDBPath); err == nil {
				filesToBackup = append(filesToBackup, s.currentDBPath)
			}
		}

		if includeUploads {
			// Сканируем директорию uploads
			uploadsDir := "data/uploads"
			if err := filepath.Walk(uploadsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".db") {
					filesToBackup = append(filesToBackup, path)
				}
				return nil
			}); err != nil {
				slog.Warn("[CreateBackup] Failed to scan uploads directory",
					"error", err,
				)
			}
		}

		if includeService {
			// Используем стандартный путь к service DB
			serviceDBPaths := []string{"service.db", "data/service.db", "./service.db"}
			for _, serviceDBPath := range serviceDBPaths {
				if _, err := os.Stat(serviceDBPath); err == nil {
					filesToBackup = append(filesToBackup, serviceDBPath)
					break // Используем первый найденный файл
				}
			}
		}
	}

	// Добавляем файлы в архив и/или копируем их
	addedFiles := 0
	totalSize := int64(0)

	for _, filePath := range filesToBackup {
		// Определяем путь в архиве
		var archivePath string
		fileName := filepath.Base(filePath)

		if strings.Contains(filePath, "uploads") {
			archivePath = filepath.Join("uploads", fileName)
		} else if fileName == "service.db" {
			archivePath = filepath.Join("service", fileName)
		} else {
			archivePath = filepath.Join("main", fileName)
		}

		// Открываем файл для чтения
		sourceFile, err := os.Open(filePath)
		if err != nil {
			slog.Warn("[CreateBackup] Failed to open file",
				"path", filePath,
				"error", err,
			)
			continue
		}

		fileInfo, err := sourceFile.Stat()
		if err != nil {
			sourceFile.Close()
			continue
		}

		// Если нужно добавить в ZIP архив
		if zipWriter != nil {
			// Создаем запись в архиве
			archiveFile, err := zipWriter.Create(archivePath)
			if err != nil {
				slog.Warn("[CreateBackup] Failed to create archive entry",
					"path", filePath,
					"error", err,
				)
				sourceFile.Close()
				continue
			}

			// Копируем содержимое файла в архив
			if _, err := io.Copy(archiveFile, sourceFile); err != nil {
				slog.Warn("[CreateBackup] Failed to copy file to archive",
					"path", filePath,
					"error", err,
				)
				sourceFile.Close()
				continue
			}
		}

		// Если нужно скопировать файлы
		if filesCopyDir != "" {
			destPath := filepath.Join(filesCopyDir, archivePath)
			destDir := filepath.Dir(destPath)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				slog.Warn("[CreateBackup] Failed to create directory",
					"dir", destDir,
					"error", err,
				)
				sourceFile.Close()
				continue
			}

			// Сбрасываем позицию файла для копирования
			if _, err := sourceFile.Seek(0, 0); err != nil {
				slog.Warn("[CreateBackup] Failed to seek file",
					"path", filePath,
					"error", err,
				)
				sourceFile.Close()
				continue
			}

			destFile, err := os.Create(destPath)
			if err != nil {
				slog.Warn("[CreateBackup] Failed to create destination file",
					"path", destPath,
					"error", err,
				)
				sourceFile.Close()
				continue
			}

			if _, err := io.Copy(destFile, sourceFile); err != nil {
				slog.Warn("[CreateBackup] Failed to copy file",
					"source", filePath,
					"dest", destPath,
					"error", err,
				)
				sourceFile.Close()
				destFile.Close()
				continue
			}
			destFile.Close()
		}

		sourceFile.Close()
		addedFiles++
		totalSize += fileInfo.Size()
	}

	// Закрываем архив, если он был создан
	if zipWriter != nil {
		if err := zipWriter.Close(); err != nil {
			return nil, apperrors.NewInternalError("не удалось завершить создание архива", err)
		}
	}

	// Проверяем, что были добавлены файлы
	if addedFiles == 0 {
		return nil, apperrors.NewValidationError("не найдено файлов для резервного копирования", nil)
	}

	backupInfo := map[string]interface{}{
		"files_count": addedFiles,
		"total_size":  totalSize,
		"created_at":  time.Now().Format(time.RFC3339),
		"format":      format,
	}

	// Добавляем информацию о ZIP архиве, если он был создан
	if backupFileName != "" {
		backupInfo["backup_file"] = backupFileName
		backupInfo["backup_path"] = backupPath
	}

	// Добавляем информацию о директории с копиями файлов, если она была создана
	if filesCopyDir != "" {
		backupInfo["files_copy_dir"] = filesCopyDir
	}

	slog.Info("[CreateBackup] Successfully created backup",
		"files_count", addedFiles,
		"total_size", totalSize,
		"format", format,
	)

	return backupInfo, nil
}

// DownloadBackup возвращает путь к файлу резервной копии для скачивания
func (s *DatabaseService) DownloadBackup(backupDir, filename string) (string, error) {
	// Безопасность: проверяем, что имя файла не содержит переходов
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", apperrors.NewValidationError("недопустимое имя файла", nil)
	}

	backupPath := filepath.Join(backupDir, filename)

	// Проверяем, что файл существует
	if _, err := os.Stat(backupPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", apperrors.NewNotFoundError("файл резервной копии не найден", err)
		}
		return "", apperrors.NewInternalError("не удалось проверить файл резервной копии", err)
	}

	// Проверяем, что путь находится в backupDir (защита от path traversal)
	absBackupDir, err := filepath.Abs(backupDir)
	if err != nil {
		return "", apperrors.NewInternalError("не удалось получить абсолютный путь к директории", err)
	}

	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return "", apperrors.NewInternalError("не удалось получить абсолютный путь к файлу", err)
	}

	if !strings.HasPrefix(absBackupPath, absBackupDir) {
		return "", apperrors.NewValidationError("недопустимый путь к файлу", nil)
	}

	return absBackupPath, nil
}

// updateDatabaseMetadataWithConfig обновляет метаданные базы данных с информацией о конфигурации 1С
func (s *DatabaseService) updateDatabaseMetadataWithConfig(filePath string, dbType string) error {
	if s.serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	fileName := filepath.Base(filePath)
	
	// Используем унифицированную функцию ParseDatabaseFileInfo из пакета server
	// Импортируем через алиас для избежания циклических зависимостей
	// Временно используем локальную реализацию, так как функция находится в пакете server
	// TODO: Вынести ParseDatabaseFileInfo в общий пакет utils
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	parts := strings.Split(nameWithoutExt, "_")
	
	var configName, databaseType, dataType, displayName string
	
	if len(parts) >= 3 {
		databaseType = parts[1] // Номенклатура или Контрагенты
		configName = parts[2]   // Название конфигурации
		
		// Если конфигурация "Unknown", пробуем взять следующую часть
		if configName == "Unknown" && len(parts) > 3 {
			configName = parts[3]
		}
		
		// Определяем тип данных для project_type
		if databaseType == "Номенклатура" {
			dataType = "nomenclature"
		} else if databaseType == "Контрагенты" {
			dataType = "counterparties"
		}
		
		// Формируем читаемое название
		if configName != "Unknown" && configName != "" {
			// Для латинских букв: разделяем по заглавным
			if strings.ContainsAny(configName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
				var formattedConfig strings.Builder
				for i, r := range configName {
					if i > 0 && r >= 'A' && r <= 'Z' {
						formattedConfig.WriteRune(' ')
					}
					formattedConfig.WriteRune(r)
				}
				displayName = strings.TrimSpace(formattedConfig.String() + " " + databaseType)
			} else {
				displayName = strings.TrimSpace(configName + " " + databaseType)
			}
		} else {
			displayName = databaseType
		}
	} else {
		displayName = nameWithoutExt
	}

	// Получаем существующие метаданные
	existingMetadata, err := s.serviceDB.GetDatabaseMetadata(filePath)
	if err != nil {
		return fmt.Errorf("failed to get existing metadata: %w", err)
	}

	// Создаем структуру для метаданных
	metadataMap := make(map[string]interface{})
	if existingMetadata != nil && existingMetadata.MetadataJSON != "" {
		// Парсим существующие метаданные
		if err := json.Unmarshal([]byte(existingMetadata.MetadataJSON), &metadataMap); err != nil {
			// Если не удалось распарсить, начинаем с пустой карты
			metadataMap = make(map[string]interface{})
		}
	}

	// Обновляем информацию о конфигурации 1С
	metadataMap["config_name"] = configName
	metadataMap["database_type"] = databaseType
	metadataMap["data_type"] = dataType
	metadataMap["display_name"] = displayName

	// Сериализуем обратно в JSON
	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Формируем описание
	description := fmt.Sprintf("База данных типа %s", dbType)
	if configName != "" && configName != "Unknown" {
		description = fmt.Sprintf("%s, конфигурация: %s", description, displayName)
	}

	// Обновляем метаданные
	if err := s.serviceDB.UpsertDatabaseMetadata(filePath, dbType, description, string(metadataJSON)); err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}

	return nil
}

