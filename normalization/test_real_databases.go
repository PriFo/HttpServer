//go:build ignore
// +build ignore

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/normalization"

	_ "github.com/mattn/go-sqlite3"
)

// TestResult результат тестирования одной базы
type TestResult struct {
	DatabasePath        string                                         `json:"database_path"`
	DatabaseName        string                                         `json:"database_name"`
	Success             bool                                           `json:"success"`
	Error               string                                         `json:"error,omitempty"`
	CounterpartiesCount int                                            `json:"counterparties_count"`
	ProcessingTime      string                                         `json:"processing_time"`
	Result              *normalization.CounterpartyNormalizationResult `json:"result,omitempty"`
	Statistics          *DatabaseStatistics                            `json:"statistics,omitempty"`
	Timestamp           time.Time                                      `json:"timestamp"`
}

// DatabaseStatistics статистика по базе данных
type DatabaseStatistics struct {
	TotalNormalized int `json:"total_normalized"`
	WithTaxID       int `json:"with_tax_id"`
	WithAddress     int `json:"with_address"`
	WithContacts    int `json:"with_contacts"`
	WithBankData    int `json:"with_bank_data"`
	WithLegalForm   int `json:"with_legal_form"`
	WithKPP         int `json:"with_kpp"`
}

// SummaryReport итоговый отчет
type SummaryReport struct {
	TotalDatabases      int          `json:"total_databases"`
	SuccessfulTests     int          `json:"successful_tests"`
	FailedTests         int          `json:"failed_tests"`
	TotalCounterparties int          `json:"total_counterparties"`
	TotalProcessed      int          `json:"total_processed"`
	TotalTime           string       `json:"total_time"`
	Results             []TestResult `json:"results"`
	Timestamp           time.Time    `json:"timestamp"`
}

func main() {
	var (
		allFlag       = flag.Bool("all", false, "Тестировать все найденные базы")
		outputJSON    = flag.String("json", "", "Экспортировать результаты в JSON файл")
		verbose       = flag.Bool("verbose", false, "Подробный вывод")
		limit         = flag.Int("limit", 0, "Ограничить количество обрабатываемых контрагентов (0 = без ограничений)")
		useBenchmarks = flag.Bool("benchmarks", false, "Использовать реальные эталоны из БД")
	)
	flag.Parse()

	var dbPaths []string

	if *allFlag {
		// Находим все базы данных
		searchDirs := []string{".", "data", "data/uploads"}
		for _, dir := range searchDirs {
			pattern := filepath.Join(dir, "*.db")
			matches, _ := filepath.Glob(pattern)
			for _, match := range matches {
				baseName := filepath.Base(match)
				if baseName != "service.db" && baseName != "test.db" && baseName != "normalized_data.db" {
					dbPaths = append(dbPaths, match)
				}
			}
		}
	} else if flag.NArg() > 0 {
		dbPaths = flag.Args()
	} else {
		fmt.Println("Использование: go run test_real_databases.go [опции] <путь_к_базе_данных> ...")
		fmt.Println("\nОпции:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(dbPaths) == 0 {
		fmt.Println("Базы данных не найдены!")
		os.Exit(1)
	}

	fmt.Printf("Найдено баз данных для тестирования: %d\n\n", len(dbPaths))

	// Создаем временную сервисную БД
	tmpDir := os.TempDir()
	serviceDBPath := filepath.Join(tmpDir, "test_service_"+fmt.Sprintf("%d", time.Now().Unix())+".db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		log.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()
	defer os.Remove(serviceDBPath)

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "user")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Project Description", "1C", 0.8)
	if err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}

	// Создаем отчет
	report := SummaryReport{
		TotalDatabases: len(dbPaths),
		Results:        make([]TestResult, 0, len(dbPaths)),
		Timestamp:      time.Now(),
	}
	startTime := time.Now()

	// Тестируем каждую базу
	for i, dbPath := range dbPaths {
		fmt.Printf("=== Тестирование базы %d/%d: %s ===\n", i+1, len(dbPaths), dbPath)

		testResult := TestResult{
			DatabasePath: dbPath,
			DatabaseName: filepath.Base(dbPath),
			Timestamp:    time.Now(),
		}

		// Проверяем существование файла
		if _, err := os.Stat(dbPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				testResult.Success = false
				testResult.Error = fmt.Sprintf("Файл не найден: %s", dbPath)
				fmt.Printf("  ✗ %s\n", testResult.Error)
			} else {
				testResult.Success = false
				testResult.Error = fmt.Sprintf("Ошибка проверки файла %s: %v", dbPath, err)
				fmt.Printf("  ✗ %s\n", testResult.Error)
			}
			report.Results = append(report.Results, testResult)
			report.FailedTests++
			continue
		}

		// Открываем базу данных
		sourceDB, err := database.NewDB(dbPath)
		if err != nil {
			testResult.Success = false
			testResult.Error = fmt.Sprintf("Ошибка открытия БД: %v", err)
			fmt.Printf("  ✗ %s\n", testResult.Error)
			report.Results = append(report.Results, testResult)
			report.FailedTests++
			continue
		}
		defer sourceDB.Close()

		// Получаем контрагентов
		counterparties, err := getCounterpartiesFromDB(sourceDB)
		if err != nil {
			testResult.Success = false
			testResult.Error = fmt.Sprintf("Ошибка получения контрагентов: %v", err)
			fmt.Printf("  ✗ %s\n", testResult.Error)
			report.Results = append(report.Results, testResult)
			report.FailedTests++
			continue
		}

		if len(counterparties) == 0 {
			if *verbose {
				fmt.Printf("  ⚠ Контрагенты не найдены, пропускаем\n")
			}
			testResult.Success = true
			testResult.CounterpartiesCount = 0
			report.Results = append(report.Results, testResult)
			continue
		}

		// Ограничиваем количество, если указан лимит
		if *limit > 0 && len(counterparties) > *limit {
			counterparties = counterparties[:*limit]
			if *verbose {
				fmt.Printf("  ⚠ Ограничено до %d контрагентов\n", *limit)
			}
		}

		testResult.CounterpartiesCount = len(counterparties)
		report.TotalCounterparties += len(counterparties)
		fmt.Printf("  Найдено контрагентов: %d\n", len(counterparties))

		// Создаем проект БД
		_, err = serviceDB.CreateProjectDatabase(project.ID, filepath.Base(dbPath), dbPath, "Test DB", 0)
		if err != nil {
			testResult.Success = false
			testResult.Error = fmt.Sprintf("Ошибка создания проекта БД: %v", err)
			fmt.Printf("  ✗ %s\n", testResult.Error)
			report.Results = append(report.Results, testResult)
			report.FailedTests++
			continue
		}

		// Создаем моковый нормализатор
		mockNormalizer := &MockAINameNormalizer{}

		// Используем реальный BenchmarkFinder, если указано
		var benchmarkFinder normalization.BenchmarkFinder
		if *useBenchmarks {
			// TODO: Реализовать подключение к реальному BenchmarkService
			// Пока используем моковый
			benchmarkFinder = &MockBenchmarkFinder{}
			if *verbose {
				fmt.Printf("  ℹ Используется моковый BenchmarkFinder (реальная интеграция TODO)\n")
			}
		} else {
			benchmarkFinder = &MockBenchmarkFinder{}
		}

		// Создаем канал для событий
		eventChannel := make(chan string, 100)

		// Создаем контекст
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Создаем нормализатор
		normalizer := normalization.NewCounterpartyNormalizer(
			serviceDB,
			client.ID,
			project.ID,
			eventChannel,
			ctx,
			mockNormalizer,
			benchmarkFinder,
		)

		// Запускаем нормализацию
		dbStartTime := time.Now()
		result, err := normalizer.ProcessNormalization(counterparties, false)
		duration := time.Since(dbStartTime)
		testResult.ProcessingTime = duration.String()

		if err != nil {
			testResult.Success = false
			testResult.Error = fmt.Sprintf("Ошибка нормализации: %v", err)
			fmt.Printf("  ✗ %s\n", testResult.Error)
			report.Results = append(report.Results, testResult)
			report.FailedTests++
			continue
		}

		if result == nil {
			testResult.Success = false
			testResult.Error = "Результат нормализации nil"
			fmt.Printf("  ✗ %s\n", testResult.Error)
			report.Results = append(report.Results, testResult)
			report.FailedTests++
			continue
		}

		testResult.Result = result
		report.TotalProcessed += result.TotalProcessed

		// Выводим результаты
		fmt.Printf("  ✓ Нормализация завершена за %v\n", duration)
		fmt.Printf("    Обработано: %d\n", result.TotalProcessed)
		fmt.Printf("    Совпадений с эталонами: %d\n", result.BenchmarkMatches)
		fmt.Printf("    Обогащено: %d\n", result.EnrichedCount)
		fmt.Printf("    Групп дублей: %d\n", result.DuplicateGroups)
		if len(result.Errors) > 0 {
			fmt.Printf("    Ошибок: %d\n", len(result.Errors))
			if *verbose {
				for _, errMsg := range result.Errors {
					fmt.Printf("      - %s\n", errMsg)
				}
			}
		}

		// Проверяем сохраненные данные
		normalized, totalCount, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, 10000, "", "", "")
		stats := &DatabaseStatistics{}
		if err == nil {
			// Если есть больше записей, чем мы получили, обновляем статистику
			if totalCount > len(normalized) {
				if *verbose {
					fmt.Printf("    Всего нормализованных: %d (показано: %d)\n", totalCount, len(normalized))
				}
			}
			stats.TotalNormalized = len(normalized)
			fmt.Printf("    Сохранено нормализованных: %d\n", len(normalized))

			// Статистика по извлеченным данным
			for _, cp := range normalized {
				if cp.TaxID != "" || cp.BIN != "" {
					stats.WithTaxID++
				}
				if cp.KPP != "" {
					stats.WithKPP++
				}
				if cp.LegalAddress != "" {
					stats.WithAddress++
				}
				if cp.ContactPhone != "" || cp.ContactEmail != "" {
					stats.WithContacts++
				}
				if cp.BankName != "" || cp.BankAccount != "" {
					stats.WithBankData++
				}
				if cp.LegalForm != "" {
					stats.WithLegalForm++
				}
			}

			fmt.Printf("    С ИНН/БИН: %d (%.1f%%)\n", stats.WithTaxID, float64(stats.WithTaxID)/float64(len(normalized))*100)
			fmt.Printf("    С КПП: %d (%.1f%%)\n", stats.WithKPP, float64(stats.WithKPP)/float64(len(normalized))*100)
			fmt.Printf("    С адресом: %d (%.1f%%)\n", stats.WithAddress, float64(stats.WithAddress)/float64(len(normalized))*100)
			fmt.Printf("    С контактами: %d (%.1f%%)\n", stats.WithContacts, float64(stats.WithContacts)/float64(len(normalized))*100)
			fmt.Printf("    С банковскими данными: %d (%.1f%%)\n", stats.WithBankData, float64(stats.WithBankData)/float64(len(normalized))*100)
			fmt.Printf("    С юридической формой: %d (%.1f%%)\n", stats.WithLegalForm, float64(stats.WithLegalForm)/float64(len(normalized))*100)
		}
		testResult.Statistics = stats
		testResult.Success = true
		report.Results = append(report.Results, testResult)
		report.SuccessfulTests++
		fmt.Println()
	}

	// Итоговая статистика
	report.TotalTime = time.Since(startTime).String()
	fmt.Printf("=== Итоги ===\n")
	fmt.Printf("Успешно: %d\n", report.SuccessfulTests)
	fmt.Printf("Провалено: %d\n", report.FailedTests)
	fmt.Printf("Всего: %d\n", report.TotalDatabases)
	fmt.Printf("Всего контрагентов: %d\n", report.TotalCounterparties)
	fmt.Printf("Всего обработано: %d\n", report.TotalProcessed)
	fmt.Printf("Общее время: %s\n", report.TotalTime)

	// Экспортируем в JSON, если указано
	if *outputJSON != "" {
		reportJSON, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Printf("Ошибка сериализации JSON: %v\n", err)
		} else {
			if err := os.WriteFile(*outputJSON, reportJSON, 0644); err != nil {
				fmt.Printf("Ошибка записи JSON файла: %v\n", err)
			} else {
				fmt.Printf("\n✓ Результаты экспортированы в: %s\n", *outputJSON)
			}
		}
	}

	if report.FailedTests == 0 {
		fmt.Println("\n✓ Все тесты пройдены успешно!")
		os.Exit(0)
	} else {
		fmt.Printf("\n✗ %d тестов провалились\n", report.FailedTests)
		os.Exit(1)
	}
}

func getCounterpartiesFromDB(db *database.DB) ([]*database.CatalogItem, error) {
	// Пробуем получить из catalog_items
	items, err := db.GetAllCatalogItems()
	if err == nil && len(items) > 0 {
		return items, nil
	}

	// Если не получилось, пробуем напрямую через SQL
	conn := db.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// Проверяем существование таблицы
	var tableExists int
	err = conn.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='table' AND name='catalog_items'
	`).Scan(&tableExists)
	if err != nil || tableExists == 0 {
		// Пробуем другие возможные таблицы
		possibleTables := []string{"counterparties", "Контрагенты", "СправочникКонтрагентов"}
		for _, tableName := range possibleTables {
			err = conn.QueryRow(`
				SELECT COUNT(*) FROM sqlite_master 
				WHERE type='table' AND name=?
			`, tableName).Scan(&tableExists)
			if err == nil && tableExists > 0 {
				// Пробуем получить данные из этой таблицы
				rows, err := conn.Query(fmt.Sprintf(`
					SELECT name, code, reference, attributes 
					FROM %s
					LIMIT 1000
				`, tableName))
				if err == nil {
					defer rows.Close()
					var items2 []*database.CatalogItem
					for rows.Next() {
						item := &database.CatalogItem{
							CatalogName: tableName,
						}
						err := rows.Scan(&item.Name, &item.Code, &item.Reference, &item.Attributes)
						if err != nil {
							continue
						}
						items2 = append(items2, item)
					}
					if len(items2) > 0 {
						return items2, rows.Err()
					}
				}
			}
		}
		return nil, fmt.Errorf("table catalog_items not found and no alternative tables found")
	}

	rows, err := conn.Query(`
		SELECT id, catalog_id, catalog_name, reference, code, name, attributes, table_parts, created_at
		FROM catalog_items
		LIMIT 1000
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query catalog_items: %w", err)
	}
	defer rows.Close()

	var items2 []*database.CatalogItem
	for rows.Next() {
		item := &database.CatalogItem{}
		err := rows.Scan(
			&item.ID,
			&item.CatalogID,
			&item.CatalogName,
			&item.Reference,
			&item.Code,
			&item.Name,
			&item.Attributes,
			&item.TableParts,
			&item.CreatedAt,
		)
		if err != nil {
			continue
		}
		items2 = append(items2, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return items2, nil
}

// MockAINameNormalizer моковый AI нормализатор
type MockAINameNormalizer struct{}

func (m *MockAINameNormalizer) NormalizeName(ctx context.Context, name string) (string, error) {
	return strings.TrimSpace(name), nil
}

func (m *MockAINameNormalizer) NormalizeCounterparty(ctx context.Context, name, inn, bin string) (string, error) {
	return strings.TrimSpace(name), nil
}

// MockBenchmarkFinder моковый BenchmarkFinder
type MockBenchmarkFinder struct{}

func (m *MockBenchmarkFinder) FindBestMatch(name string, entityType string) (normalizedName string, found bool, err error) {
	return "", false, nil
}
