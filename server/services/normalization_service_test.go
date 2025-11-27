package services

import (
	"path/filepath"
	"testing"

	"httpserver/database"
	"httpserver/normalization"
)

// setupTestDBForNormalization создает тестовую БД для нормализации
func setupTestDBForNormalization(t *testing.T) *database.DB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

// setupTestServiceDBForNormalization создает тестовую ServiceDB для нормализации
func setupTestServiceDBForNormalization(t *testing.T) *database.ServiceDB {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "service_test.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test service database: %v", err)
	}
	return serviceDB
}

// TestNewNormalizationService проверяет создание нового сервиса нормализации
func TestNewNormalizationService(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)
	benchmarkService := NewBenchmarkService(nil, db, serviceDB)

	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)
	if service == nil {
		t.Error("NewNormalizationService() should not return nil")
	}
}

// TestNormalizationService_IsRunning проверяет проверку статуса запуска
func TestNormalizationService_IsRunning(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)
	benchmarkService := NewBenchmarkService(nil, db, serviceDB)

	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	// Изначально не запущена
	if service.IsRunning() {
		t.Error("Expected normalization to not be running initially")
	}

	// Запускаем
	err := service.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Теперь должна быть запущена
	if !service.IsRunning() {
		t.Error("Expected normalization to be running after Start()")
	}
}

// TestNormalizationService_Start_Success проверяет успешный запуск нормализации
func TestNormalizationService_Start_Success(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	err := service.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !service.IsRunning() {
		t.Error("Expected normalization to be running")
	}
}

// TestNormalizationService_Start_AlreadyRunning проверяет обработку повторного запуска
func TestNormalizationService_Start_AlreadyRunning(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	// Первый запуск
	err := service.Start()
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}

	// Второй запуск должен вернуть ошибку
	err = service.Start()
	if err == nil {
		t.Error("Expected error when starting already running normalization")
	}
}

// TestNormalizationService_Stop_Success проверяет успешную остановку нормализации
func TestNormalizationService_Stop_Success(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	// Запускаем
	err := service.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Останавливаем
	wasRunning := service.Stop()
	if !wasRunning {
		t.Error("Expected Stop() to return true when normalization was running")
	}

	if service.IsRunning() {
		t.Error("Expected normalization to not be running after Stop()")
	}
}

// TestNormalizationService_Stop_NotRunning проверяет остановку когда не запущена
func TestNormalizationService_Stop_NotRunning(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	// Останавливаем без запуска
	wasRunning := service.Stop()
	if wasRunning {
		t.Error("Expected Stop() to return false when normalization was not running")
	}
}

// TestNormalizationService_GetStatus проверяет получение статуса нормализации
func TestNormalizationService_GetStatus(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	// Статус до запуска
	status := service.GetStatus()
	if status.IsRunning {
		t.Error("Expected status.IsRunning to be false initially")
	}

	// Запускаем
	err := service.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Статус после запуска
	status = service.GetStatus()
	if !status.IsRunning {
		t.Error("Expected status.IsRunning to be true after Start()")
	}
	if status.Processed != 0 {
		t.Errorf("Expected status.Processed to be 0, got %d", status.Processed)
	}
	if status.Success != 0 {
		t.Errorf("Expected status.Success to be 0, got %d", status.Success)
	}
	if status.Errors != 0 {
		t.Errorf("Expected status.Errors to be 0, got %d", status.Errors)
	}
}

// TestNormalizationService_GetSessionHistory_NilDB проверяет обработку nil БД
func TestNormalizationService_GetSessionHistory_NilDB(t *testing.T) {
	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(nil, events, nil)
	benchmarkService := NewBenchmarkService(nil, nil, serviceDB)

	service := NewNormalizationService(nil, serviceDB, normalizer, benchmarkService, events)

	_, err := service.GetSessionHistory(1)
	if err == nil {
		t.Error("Expected error when db is nil")
	}
}

// TestNormalizationService_GetSessionHistory_NotFound проверяет обработку несуществующей сессии
func TestNormalizationService_GetSessionHistory_NotFound(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	_, err := service.GetSessionHistory(99999)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

// TestNormalizationService_StartVersionedNormalization проверяет создание сессии версионированной нормализации
func TestNormalizationService_StartVersionedNormalization(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	// Создаем тестовый элемент каталога
	catalog, err := db.AddCatalog(1, "TestCatalog", "test_catalog")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	err = db.AddCatalogItem(catalog.ID, "ref1", "code1", "Test Item Name", "", "")
	if err != nil {
		t.Fatalf("Failed to create catalog item: %v", err)
	}

	var itemID int
	err = db.QueryRow("SELECT id FROM catalog_items WHERE reference = ?", "ref1").Scan(&itemID)
	if err != nil {
		t.Fatalf("Failed to get catalog item ID: %v", err)
	}

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	getAPIKey := func() string { return "test-key" }
	result, err := service.StartVersionedNormalization(itemID, "Test Item Name", getAPIKey)
	if err != nil {
		t.Fatalf("StartVersionedNormalization failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}

	sessionID, ok := result["session_id"].(int)
	if !ok || sessionID == 0 {
		t.Error("Expected valid session_id in result")
	}
}

// TestNormalizationService_ApplyPatterns проверяет применение паттернов к сессии
func TestNormalizationService_ApplyPatterns(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	// Создаем тестовый элемент каталога и сессию
	catalog, err := db.AddCatalog(1, "TestCatalog", "test_catalog")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	err = db.AddCatalogItem(catalog.ID, "ref1", "code1", "Test Item ER-00013004", "", "")
	if err != nil {
		t.Fatalf("Failed to create catalog item: %v", err)
	}

	var itemID int
	err = db.QueryRow("SELECT id FROM catalog_items WHERE reference = ?", "ref1").Scan(&itemID)
	if err != nil {
		t.Fatalf("Failed to get catalog item ID: %v", err)
	}

	// Создаем сессию нормализации
	sessionID, err := db.CreateNormalizationSession(itemID, "Test Item ER-00013004")
	if err != nil {
		t.Fatalf("Failed to create normalization session: %v", err)
	}

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	getAPIKey := func() string { return "test-key" }
	result, err := service.ApplyPatterns(sessionID, getAPIKey)
	if err != nil {
		t.Fatalf("ApplyPatterns failed: %v", err)
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}

	currentName, ok := result["current_name"].(string)
	if !ok || currentName == "" {
		t.Error("Expected valid current_name in result")
	}
}

// TestNormalizationService_ApplyAI проверяет применение AI коррекции к сессии
func TestNormalizationService_ApplyAI(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	getAPIKey := func() string { return "test-key" }
	_, err := service.ApplyAI(1, false, []string{}, getAPIKey)
	if err == nil {
		t.Error("Expected error for not implemented method")
	}
}

// TestNormalizationService_RevertStage проверяет обработку нереализованного метода
func TestNormalizationService_RevertStage(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	getAPIKey := func() string { return "test-key" }
	_, err := service.RevertStage(1, 0, getAPIKey)
	if err == nil {
		t.Error("Expected error for not implemented method")
	}
}

// TestNormalizationService_ApplyCategorization проверяет обработку нереализованного метода
func TestNormalizationService_ApplyCategorization(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	getAPIKey := func() string { return "test-key" }
	_, err := service.ApplyCategorization(1, "category", getAPIKey)
	if err == nil {
		t.Error("Expected error for not implemented method")
	}
}

// TestNormalizationService_CreateStopCheck проверяет создание функции проверки остановки
func TestNormalizationService_CreateStopCheck(t *testing.T) {
	db := setupTestDBForNormalization(t)
	defer db.Close()

	serviceDB := setupTestServiceDBForNormalization(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	normalizer := normalization.NewNormalizer(db, events, nil)

	benchmarkService := NewBenchmarkService(nil, db, serviceDB)
	service := NewNormalizationService(db, serviceDB, normalizer, benchmarkService, events)

	stopCheck := service.CreateStopCheck()

	// Изначально должна возвращать true (не запущена)
	if !stopCheck() {
		t.Error("Expected stopCheck() to return true when not running")
	}

	// Запускаем
	err := service.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Теперь должна возвращать false (запущена)
	if stopCheck() {
		t.Error("Expected stopCheck() to return false when running")
	}

	// Останавливаем
	service.Stop()

	// Теперь должна возвращать true (остановлена)
	if !stopCheck() {
		t.Error("Expected stopCheck() to return true when stopped")
	}
}
