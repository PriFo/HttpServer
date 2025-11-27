package server

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"httpserver/database"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDatabases создает тестовые базы данных для тестирования функции сканирования
func setupTestDatabases(t *testing.T) (serviceDBPath, mainDBPath string, cleanup func()) {
	tempDir := t.TempDir()

	// Создаем service.db
	serviceDBPath = filepath.Join(tempDir, "service.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient(
		"Test Client",
		"Test Client Legal Name",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.9,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем несколько тестовых БД для проекта
	uploadsDir := filepath.Join(tempDir, "data", "uploads")
	os.MkdirAll(uploadsDir, 0755)

	db1Path := filepath.Join(uploadsDir, "test_db1.db")
	db2Path := filepath.Join(uploadsDir, "test_db2.db")

	// Создаем первую БД с номенклатурой и контрагентами
	createTestDatabaseWithData(t, db1Path, 10, 5)
	db1, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB 1", db1Path, "Test database 1", 1024)
	if err != nil {
		t.Fatalf("Failed to create project database 1: %v", err)
	}

	// Создаем вторую БД с номенклатурой и контрагентами
	createTestDatabaseWithData(t, db2Path, 15, 8)
	db2, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB 2", db2Path, "Test database 2", 2048)
	if err != nil {
		t.Fatalf("Failed to create project database 2: %v", err)
	}

	// Создаем основную БД (data.db)
	mainDBPath = filepath.Join(tempDir, "data.db")
	mainDB, err := database.NewDB(mainDBPath)
	if err != nil {
		t.Fatalf("Failed to create main DB: %v", err)
	}
	defer mainDB.Close()

	// Создаем загрузки, связанные с БД
	upload1, err := mainDB.CreateUploadWithDatabase(
		"uuid-1",
		"8.3.25",
		"TestConfig1",
		&db1.ID,
		"computer1",
		"user1",
		"v1.0",
		1,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create upload 1: %v", err)
	}

	upload2, err := mainDB.CreateUploadWithDatabase(
		"uuid-2",
		"8.3.25",
		"TestConfig2",
		&db2.ID,
		"computer2",
		"user2",
		"v1.0",
		1,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create upload 2: %v", err)
	}

	// Завершаем загрузки
	if err := mainDB.CompleteUpload(upload1.ID); err != nil {
		t.Fatalf("Failed to complete upload 1: %v", err)
	}
	if err := mainDB.CompleteUpload(upload2.ID); err != nil {
		t.Fatalf("Failed to complete upload 2: %v", err)
	}

	cleanup = func() {
		os.RemoveAll(tempDir)
	}

	return serviceDBPath, mainDBPath, cleanup
}

// createTestDatabaseWithData создает тестовую БД с номенклатурой и контрагентами
func createTestDatabaseWithData(t *testing.T, dbPath string, nomenclatureCount, counterpartyCount int) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer conn.Close()

	// Создаем таблицу номенклатуры
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS nomenclature_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			upload_id INTEGER,
			nomenclature_reference TEXT,
			nomenclature_code TEXT,
			nomenclature_name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create nomenclature_items table: %v", err)
	}

	// Создаем таблицу контрагентов
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS counterparties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			upload_id INTEGER,
			reference TEXT,
			name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create counterparties table: %v", err)
	}

	// Заполняем номенклатуру
	for i := 0; i < nomenclatureCount; i++ {
		_, err = conn.Exec(
			`INSERT INTO nomenclature_items (upload_id, nomenclature_reference, nomenclature_code, nomenclature_name) 
			 VALUES (?, ?, ?, ?)`,
			1,
			"ref-"+strconv.Itoa(i),
			"code-"+strconv.Itoa(i),
			"Item "+strconv.Itoa(i),
		)
		if err != nil {
			t.Fatalf("Failed to insert nomenclature item: %v", err)
		}
	}

	// Заполняем контрагентов
	for i := 0; i < counterpartyCount; i++ {
		_, err = conn.Exec(
			`INSERT INTO counterparties (upload_id, reference, name) 
			 VALUES (?, ?, ?)`,
			1,
			"ref-counterparty-"+strconv.Itoa(i),
			"Counterparty "+strconv.Itoa(i),
		)
		if err != nil {
			t.Fatalf("Failed to insert counterparty: %v", err)
		}
	}
}

// TestScanAndSummarizeAllDatabases проверяет основную функциональность сканирования
func TestScanAndSummarizeAllDatabases(t *testing.T) {
	serviceDBPath, mainDBPath, cleanup := setupTestDatabases(t)
	defer cleanup()

	ctx := context.Background()
	summary, err := ScanAndSummarizeAllDatabases(ctx, serviceDBPath, mainDBPath)
	if err != nil {
		t.Fatalf("ScanAndSummarizeAllDatabases() failed: %v", err)
	}

	// Проверяем общую статистику
	if summary.TotalUploads != 2 {
		t.Errorf("TotalUploads = %d, want 2", summary.TotalUploads)
	}

	if summary.TotalDatabases != 2 {
		t.Errorf("TotalDatabases = %d, want 2", summary.TotalDatabases)
	}

	if summary.CompletedUploads != 2 {
		t.Errorf("CompletedUploads = %d, want 2", summary.CompletedUploads)
	}

	// Проверяем, что общее количество номенклатуры = 10 + 15 = 25
	if summary.TotalNomenclature != 25 {
		t.Errorf("TotalNomenclature = %d, want 25", summary.TotalNomenclature)
	}

	// Проверяем, что общее количество контрагентов = 5 + 8 = 13
	if summary.TotalCounterparties != 13 {
		t.Errorf("TotalCounterparties = %d, want 13", summary.TotalCounterparties)
	}

	// Проверяем детали загрузок
	if len(summary.UploadDetails) != 2 {
		t.Errorf("UploadDetails length = %d, want 2", len(summary.UploadDetails))
	}

	// Проверяем первую загрузку
	upload1 := summary.UploadDetails[0]
	if upload1.NomenclatureCount != 10 && upload1.NomenclatureCount != 15 {
		t.Errorf("Upload1 NomenclatureCount = %d, want 10 or 15", upload1.NomenclatureCount)
	}

	if upload1.CounterpartyCount != 5 && upload1.CounterpartyCount != 8 {
		t.Errorf("Upload1 CounterpartyCount = %d, want 5 or 8", upload1.CounterpartyCount)
	}
}

// TestScanAndSummarizeAllDatabases_Empty проверяет обработку пустой системы
func TestScanAndSummarizeAllDatabases_Empty(t *testing.T) {
	tempDir := t.TempDir()

	// Создаем пустую service.db
	serviceDBPath := filepath.Join(tempDir, "service.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	serviceDB.Close()

	// Создаем пустую основную БД
	mainDBPath := filepath.Join(tempDir, "data.db")
	mainDB, err := database.NewDB(mainDBPath)
	if err != nil {
		t.Fatalf("Failed to create main DB: %v", err)
	}
	mainDB.Close()

	ctx := context.Background()
	summary, err := ScanAndSummarizeAllDatabases(ctx, serviceDBPath, mainDBPath)
	if err != nil {
		t.Fatalf("ScanAndSummarizeAllDatabases() failed: %v", err)
	}

	// Проверяем, что все счетчики равны 0
	if summary.TotalUploads != 0 {
		t.Errorf("TotalUploads = %d, want 0", summary.TotalUploads)
	}

	if summary.TotalDatabases != 0 {
		t.Errorf("TotalDatabases = %d, want 0", summary.TotalDatabases)
	}

	if summary.CompletedUploads != 0 {
		t.Errorf("CompletedUploads = %d, want 0", summary.CompletedUploads)
	}

	if summary.TotalNomenclature != 0 {
		t.Errorf("TotalNomenclature = %d, want 0", summary.TotalNomenclature)
	}

	if summary.TotalCounterparties != 0 {
		t.Errorf("TotalCounterparties = %d, want 0", summary.TotalCounterparties)
	}

	if len(summary.UploadDetails) != 0 {
		t.Errorf("UploadDetails length = %d, want 0", len(summary.UploadDetails))
	}
}

// TestCountDatabaseRecords проверяет подсчет записей в БД
func TestCountDatabaseRecords(t *testing.T) {
	// Обновляем сигнатуру функции для поддержки размера БД
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Создаем тестовую БД
	createTestDatabaseWithData(t, dbPath, 10, 5)

	ctx := context.Background()
	nomenclatureCount, counterpartyCount, dbSize, err := countDatabaseRecords(ctx, dbPath)
	if err != nil {
		t.Fatalf("countDatabaseRecords() failed: %v", err)
	}
	
	// Проверяем, что размер БД получен
	if dbSize <= 0 {
		t.Error("Expected database size > 0")
	}

	if nomenclatureCount != 10 {
		t.Errorf("NomenclatureCount = %d, want 10", nomenclatureCount)
	}

	if counterpartyCount != 5 {
		t.Errorf("CounterpartyCount = %d, want 5", counterpartyCount)
	}
}

// TestCountDatabaseRecords_NoTables проверяет обработку БД без таблиц
func TestCountDatabaseRecords_NoTables(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "empty.db")

	// Создаем пустую БД
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	conn.Close()

	ctx := context.Background()
	nomenclatureCount, counterpartyCount, dbSize, err := countDatabaseRecords(ctx, dbPath)
	if err != nil {
		t.Fatalf("countDatabaseRecords() failed: %v", err)
	}
	
	// Проверяем, что размер БД получен
	if dbSize <= 0 {
		t.Error("Expected database size > 0")
	}

	if nomenclatureCount != 0 {
		t.Errorf("NomenclatureCount = %d, want 0", nomenclatureCount)
	}

	if counterpartyCount != 0 {
		t.Errorf("CounterpartyCount = %d, want 0", counterpartyCount)
	}
}

// TestCountDatabaseRecords_NotFound проверяет обработку несуществующего файла
func TestCountDatabaseRecords_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "nonexistent.db")

	ctx := context.Background()
	_, _, _, err := countDatabaseRecords(ctx, dbPath)
	if err == nil {
		t.Error("countDatabaseRecords() should return error for non-existent file")
	}
}
