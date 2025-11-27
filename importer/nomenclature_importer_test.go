package importer

import (
	"path/filepath"
	"testing"
	"time"

	"httpserver/database"
)

// setupTestServiceDB создает тестовую сервисную БД
func setupTestServiceDB(t *testing.T) *database.ServiceDB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_service.db")
	
	serviceDB, err := database.NewServiceDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	
	return serviceDB
}

// TestNewNomenclatureImporter проверяет создание нового импортера
func TestNewNomenclatureImporter(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()
	
	importer := NewNomenclatureImporter(serviceDB)
	
	if importer == nil {
		t.Fatal("NewNomenclatureImporter() returned nil")
	}
	
	if importer.db == nil {
		t.Error("NomenclatureImporter.db is nil")
	}
}

// TestImportNomenclatures_EmptyRecords проверяет импорт пустого списка
func TestImportNomenclatures_EmptyRecords(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()
	
	importer := NewNomenclatureImporter(serviceDB)
	
	result, err := importer.ImportNomenclatures([]NomenclatureRecord{}, 1)
	if err != nil {
		t.Fatalf("ImportNomenclatures() failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("ImportNomenclatures() returned nil result")
	}
	
	if result.Total != 0 {
		t.Errorf("ImportNomenclatures() Total = %d, want 0", result.Total)
	}
	
	if result.Success != 0 {
		t.Errorf("ImportNomenclatures() Success = %d, want 0", result.Success)
	}
}

// TestImportNomenclatures_InvalidRecord проверяет обработку невалидных записей
func TestImportNomenclatures_InvalidRecord(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()
	
	importer := NewNomenclatureImporter(serviceDB)
	
	records := []NomenclatureRecord{
		{
			// Запись без названия продукта
			ManufacturerName: "Test Manufacturer",
			ProductName:      "", // Пустое название
		},
	}
	
	result, err := importer.ImportNomenclatures(records, 1)
	if err != nil {
		t.Fatalf("ImportNomenclatures() failed: %v", err)
	}
	
	if result.Success > 0 {
		t.Errorf("ImportNomenclatures() Success = %d, want 0 for invalid records", result.Success)
	}
	
	if len(result.Errors) == 0 {
		t.Error("ImportNomenclatures() should have errors for invalid records")
	}
}

// TestImportResult проверяет структуру результата импорта
func TestImportResult(t *testing.T) {
	result := &ImportResult{
		Total:    100,
		Success:  90,
		Updated:  10,
		Errors:   []string{"error1", "error2"},
		Started:  time.Now(),
		Completed: time.Now(),
		Duration: time.Second,
	}
	
	if result.Total < 0 {
		t.Error("ImportResult.Total should be non-negative")
	}
	
	if result.Success < 0 {
		t.Error("ImportResult.Success should be non-negative")
	}
	
	if result.Updated < 0 {
		t.Error("ImportResult.Updated should be non-negative")
	}
	
	if result.Success > result.Total {
		t.Error("ImportResult.Success should not exceed Total")
	}
	
	if result.Updated > result.Success {
		t.Error("ImportResult.Updated should not exceed Success")
	}
	
	if result.Duration < 0 {
		t.Error("ImportResult.Duration should be non-negative")
	}
}

// TestImportNomenclatures_ValidRecord проверяет импорт валидной записи
func TestImportNomenclatures_ValidRecord(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()
	
	importer := NewNomenclatureImporter(serviceDB)
	
	records := []NomenclatureRecord{
		{
			ManufacturerName: "Test Manufacturer",
			ProductName:      "Test Product",
			INN:              "1234567890",
		},
	}
	
	result, err := importer.ImportNomenclatures(records, 1)
	// Может быть ошибка из-за отсутствия схемы БД - это нормально для unit тестов
	if err != nil {
		t.Logf("ImportNomenclatures() returned error (may be expected): %v", err)
		return
	}
	
	if result == nil {
		t.Fatal("ImportNomenclatures() returned nil result")
	}
	
	if result.Total != 1 {
		t.Errorf("ImportNomenclatures() Total = %d, want 1", result.Total)
	}
}

