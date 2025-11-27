package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"httpserver/database"
)

// BenchmarkHandleStartClientNormalization бенчмарк обработки запроса запуска
func BenchmarkHandleStartClientNormalization(b *testing.B) {
	srv, cleanup := setupTestServerForBenchmark(b)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterpartiesForBenchmark(b, counterparties)
	createTestProjectDatabaseForBenchmark(b, srv.serviceDB, project.ID, dbPath)

	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Убеждаемся, что нормализация не запущена перед каждым запросом
		srv.normalizerMutex.Lock()
		srv.normalizerRunning = false
		srv.normalizerMutex.Unlock()

		srv.handleStartClientNormalization(w, req, client.ID, project.ID)

		// Останавливаем нормализацию после каждого запроса
		srv.normalizerMutex.Lock()
		srv.normalizerRunning = false
		srv.normalizerMutex.Unlock()
	}
}

// BenchmarkHandleGetClientNormalizationStatus бенчмарк получения статуса
func BenchmarkHandleGetClientNormalizationStatus(b *testing.B) {
	srv, cleanup := setupTestServerForBenchmark(b)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
		w := httptest.NewRecorder()

		srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)
	}
}

// BenchmarkProcessCounterpartyDatabase бенчмарк обработки одной БД
func BenchmarkProcessCounterpartyDatabase(b *testing.B) {
	srv, cleanup := setupTestServerForBenchmark(b)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	// Создаем БД с контрагентами
	counterparties := make([]map[string]string, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = map[string]string{
			"name": "ООО Тест " + string(rune('A'+(i%26))),
			"inn":  "123456789" + string(rune('0'+(i%10))),
		}
	}
	dbPath := createTestDatabaseWithCounterpartiesForBenchmark(b, counterparties)
	projectDB := createTestProjectDatabaseForBenchmark(b, srv.serviceDB, project.ID, dbPath)

	// Устанавливаем флаг нормализации
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Сбрасываем флаг перед каждой итерацией
		srv.normalizerMutex.Lock()
		srv.normalizerRunning = true
		srv.normalizerMutex.Unlock()

		srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

		// Останавливаем после обработки
		srv.normalizerMutex.Lock()
		srv.normalizerRunning = false
		srv.normalizerMutex.Unlock()
	}
}

// BenchmarkProcessCounterpartyDatabasesParallel бенчмарк параллельной обработки нескольких БД
func BenchmarkProcessCounterpartyDatabasesParallel(b *testing.B) {
	srv, cleanup := setupTestServerForBenchmark(b)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	// Создаем несколько БД
	var databases []*database.ProjectDatabase
	for i := 0; i < 5; i++ {
		counterparties := []map[string]string{
			{"name": "ООО Тест " + string(rune('A'+i)), "inn": "123456789" + string(rune('0'+i))},
		}
		dbPath := createTestDatabaseWithCounterpartiesForBenchmark(b, counterparties)
		projectDB := createTestProjectDatabaseForBenchmark(b, srv.serviceDB, project.ID, dbPath)
		databases = append(databases, projectDB)
	}

	// Устанавливаем флаг нормализации
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Сбрасываем флаг перед каждой итерацией
		srv.normalizerMutex.Lock()
		srv.normalizerRunning = true
		srv.normalizerMutex.Unlock()

		srv.processCounterpartyDatabasesParallel(databases, client.ID, project.ID)

		// Останавливаем после обработки
		srv.normalizerMutex.Lock()
		srv.normalizerRunning = false
		srv.normalizerMutex.Unlock()
	}
}

// BenchmarkHandleStopClientNormalization бенчмарк обработки запроса остановки
func BenchmarkHandleStopClientNormalization(b *testing.B) {
	srv, cleanup := setupTestServerForBenchmark(b)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Устанавливаем флаг нормализации перед каждым запросом
		srv.normalizerMutex.Lock()
		srv.normalizerRunning = true
		srv.normalizerMutex.Unlock()

		req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)
		w := httptest.NewRecorder()

		srv.handleStopClientNormalization(w, req, client.ID, project.ID)
	}
}

// BenchmarkHandleGetClientNormalizationStats бенчмарк получения статистики
func BenchmarkHandleGetClientNormalizationStats(b *testing.B) {
	srv, cleanup := setupTestServerForBenchmark(b)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/stats", nil)
		w := httptest.NewRecorder()

		srv.handleGetClientNormalizationStats(w, req, client.ID, project.ID)
	}
}

// BenchmarkHandleGetClientNormalizationGroups бенчмарк получения групп
func BenchmarkHandleGetClientNormalizationGroups(b *testing.B) {
	srv, cleanup := setupTestServerForBenchmark(b)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/groups", nil)
		w := httptest.NewRecorder()

		srv.handleGetClientNormalizationGroups(w, req, client.ID, project.ID)
	}
}

// setupTestServerForBenchmark создает тестовый сервер для benchmark тестов
func setupTestServerForBenchmark(b *testing.B) (*Server, func()) {
	tempDir, err := os.MkdirTemp("", "bench_test_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	serviceDBPath := filepath.Join(tempDir, "test_service.db")

	// Создаем временное подключение для создания минимальной схемы
	conn, err := sql.Open("sqlite3", serviceDBPath)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to open test service DB: %v", err)
	}

	// Создаем минимальную таблицу catalog_items для миграций
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS catalog_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			catalog_id INTEGER NOT NULL,
			reference TEXT NOT NULL,
			code TEXT,
			name TEXT,
			attributes_xml TEXT,
			table_parts_xml TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		conn.Close()
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to create catalog_items table: %v", err)
	}
	conn.Close()

	// Теперь создаем serviceDB - миграции должны пройти успешно
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		os.RemoveAll(tempDir)
		if strings.Contains(err.Error(), "catalog_items") || strings.Contains(err.Error(), "migration") {
			b.Skip("Skipping benchmark due to schema migration dependencies")
		}
		b.Fatalf("Failed to create test service DB: %v", err)
	}

	// Создаем конфигурацию
	config := &Config{
		Port:                    "9999",
		DatabasePath:            ":memory:",
		NormalizedDatabasePath:  ":memory:",
		ServiceDatabasePath:     serviceDBPath,
		MaxOpenConns:            25,
		MaxIdleConns:            5,
		ConnMaxLifetime:         5 * 60 * 1000000000, // 5 минут в наносекундах
		LogBufferSize:           100,
		NormalizerEventsBufferSize: 100,
	}

	// Создаем сервер с nil для db и normalizedDB (не нужны для этих тестов)
	srv := NewServerWithConfig(nil, nil, serviceDB, ":memory:", ":memory:", config)

	// Функция очистки
	cleanup := func() {
		serviceDB.Close()
		os.RemoveAll(tempDir)
	}

	return srv, cleanup
}

// createTestDatabaseWithCounterpartiesForBenchmark создает тестовую БД с контрагентами для benchmark
func createTestDatabaseWithCounterpartiesForBenchmark(b *testing.B, counterparties []map[string]string) string {
	tempDir, err := os.MkdirTemp("", "bench_db_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDB(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close()

	// Создаем выгрузку
	upload, err := db.CreateUpload("test-uuid", "8.3", "test-config")
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to create upload: %v", err)
	}

	// Создаем каталог "Контрагенты"
	catalog, err := db.AddCatalog(upload.ID, "Контрагенты", "counterparties")
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем контрагентов
	for i, cp := range counterparties {
		attributes := ""
		if inn, ok := cp["inn"]; ok {
			attributes += `<ИНН>` + inn + `</ИНН>`
		}
		if kpp, ok := cp["kpp"]; ok {
			attributes += `<КПП>` + kpp + `</КПП>`
		}
		if bin, ok := cp["bin"]; ok {
			attributes += `<БИН>` + bin + `</БИН>`
		}

		name := cp["name"]
		if name == "" {
			name = "ООО Тест " + string(rune('A'+i))
		}

		err := db.AddCatalogItem(catalog.ID, "ref_"+name, "code_"+name, name, attributes, "")
		if err != nil {
			os.RemoveAll(tempDir)
			b.Fatalf("Failed to add catalog item: %v", err)
		}
	}

	return dbPath
}

// createTestProjectDatabaseForBenchmark создает тестовую проектную БД для benchmark
func createTestProjectDatabaseForBenchmark(b *testing.B, serviceDB *database.ServiceDB, projectID int, filePath string) *database.ProjectDatabase {
	db, err := serviceDB.CreateProjectDatabase(projectID, "Test DB", filePath, "", 0)
	if err != nil {
		b.Fatalf("Failed to create project database: %v", err)
	}
	return db
}
