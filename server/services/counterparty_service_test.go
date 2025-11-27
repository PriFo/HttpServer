package services

import (
	"testing"

	"httpserver/database"
)

// setupTestBenchmarksDB создает тестовую БД для эталонов
func setupTestBenchmarksDB(t *testing.T) *database.BenchmarksDB {
	tempDir := t.TempDir()
	dbPath := tempDir + "/benchmarks.db"
	db, err := database.NewBenchmarksDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test BenchmarksDB: %v", err)
	}
	return db
}

// TestNewCounterpartyService проверяет создание нового сервиса контрагентов
func TestNewCounterpartyService(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	db := setupTestDB(t)
	defer db.Close()
	benchmarksDB := setupTestBenchmarksDB(t)
	defer benchmarksDB.Close()
	benchmarkService := NewBenchmarkService(benchmarksDB, db, serviceDB)
	service := NewCounterpartyService(serviceDB, events, benchmarkService)
	if service == nil {
		t.Error("NewCounterpartyService() should not return nil")
	}
}

// TestCounterpartyService_IsRunning проверяет проверку статуса запуска
func TestCounterpartyService_IsRunning(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	db := setupTestDB(t)
	defer db.Close()
	benchmarksDB := setupTestBenchmarksDB(t)
	defer benchmarksDB.Close()
	benchmarkService := NewBenchmarkService(benchmarksDB, db, serviceDB)
	service := NewCounterpartyService(serviceDB, events, benchmarkService)

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

// TestCounterpartyService_Start_Success проверяет успешный запуск
func TestCounterpartyService_Start_Success(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	db := setupTestDB(t)
	defer db.Close()
	benchmarksDB := setupTestBenchmarksDB(t)
	defer benchmarksDB.Close()
	benchmarkService := NewBenchmarkService(benchmarksDB, db, serviceDB)
	service := NewCounterpartyService(serviceDB, events, benchmarkService)

	err := service.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !service.IsRunning() {
		t.Error("Expected normalization to be running")
	}
}

// TestCounterpartyService_Start_AlreadyRunning проверяет обработку повторного запуска
func TestCounterpartyService_Start_AlreadyRunning(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	db := setupTestDB(t)
	defer db.Close()
	benchmarksDB := setupTestBenchmarksDB(t)
	defer benchmarksDB.Close()
	benchmarkService := NewBenchmarkService(benchmarksDB, db, serviceDB)
	service := NewCounterpartyService(serviceDB, events, benchmarkService)

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

// TestCounterpartyService_Stop_Success проверяет успешную остановку
func TestCounterpartyService_Stop_Success(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	db := setupTestDB(t)
	defer db.Close()
	benchmarksDB := setupTestBenchmarksDB(t)
	defer benchmarksDB.Close()
	benchmarkService := NewBenchmarkService(benchmarksDB, db, serviceDB)
	service := NewCounterpartyService(serviceDB, events, benchmarkService)

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

// TestCounterpartyService_Stop_NotRunning проверяет остановку когда не запущена
func TestCounterpartyService_Stop_NotRunning(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	db := setupTestDB(t)
	defer db.Close()
	benchmarksDB := setupTestBenchmarksDB(t)
	defer benchmarksDB.Close()
	benchmarkService := NewBenchmarkService(benchmarksDB, db, serviceDB)
	service := NewCounterpartyService(serviceDB, events, benchmarkService)

	// Останавливаем без запуска
	wasRunning := service.Stop()
	if wasRunning {
		t.Error("Expected Stop() to return false when normalization was not running")
	}
}

// TestCounterpartyService_GetNormalizedCounterpartyStats проверяет получение статистики
func TestCounterpartyService_GetNormalizedCounterpartyStats(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	events := make(chan string, 10)
	db := setupTestDB(t)
	defer db.Close()
	benchmarksDB := setupTestBenchmarksDB(t)
	defer benchmarksDB.Close()
	benchmarkService := NewBenchmarkService(benchmarksDB, db, serviceDB)
	service := NewCounterpartyService(serviceDB, events, benchmarkService)

	stats, err := service.GetNormalizedCounterpartyStats(1)
	if err != nil {
		t.Fatalf("GetNormalizedCounterpartyStats() error = %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil stats")
	}
}


