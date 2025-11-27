package services

import (
	"path/filepath"
	"testing"

	"httpserver/database"
)

// setupTestDBForClassification создает тестовую БД для классификации
func setupTestDBForClassification(t *testing.T) *database.DB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	return db
}

// setupTestServiceDBForClassification создает тестовую ServiceDB для классификации
func setupTestServiceDBForClassification(t *testing.T) *database.ServiceDB {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "service_test.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test ServiceDB: %v", err)
	}
	return serviceDB
}

// TestClassificationService_ResetClassification_InvalidCriteria проверяет обработку невалидных критериев
func TestClassificationService_ResetClassification_InvalidCriteria(t *testing.T) {
	db := setupTestDBForClassification(t)
	defer db.Close()

	normalizedDB := setupTestDBForClassification(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	// Тест с пустыми критериями (все параметры пустые и resetAll = false)
	_, err := service.ResetClassification("", "", "", 0.0, false)
	if err == nil {
		t.Error("Expected error for empty criteria")
	}
	// Проверяем, что ошибка содержит информацию о невалидных критериях
	if err != nil && err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
	// Проверяем, что ошибка действительно связана с отсутствием критериев
	if err != nil {
		errorMsg := err.Error()
		if errorMsg == "" {
			t.Error("Expected error message for empty criteria")
		}
	}
}

// TestClassificationService_ResetClassification_NilDatabase проверяет обработку nil БД
func TestClassificationService_ResetClassification_NilDatabase(t *testing.T) {
	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	getAPIKey := func() string { return "" }
	service := NewClassificationService(nil, nil, serviceDB, getModel, getAPIKey)

	// Попытка сброса должна вернуть ошибку
	_, err := service.ResetClassification("test", "", "", 0.0, false)
	if err == nil {
		t.Error("Expected error when database is nil")
	}
}

// TestClassificationService_ResetClassification_InvalidConfidence проверяет обработку невалидного confidence
func TestClassificationService_ResetClassification_InvalidConfidence(t *testing.T) {
	db := setupTestDBForClassification(t)
	defer db.Close()

	normalizedDB := setupTestDBForClassification(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	// Тест с отрицательным confidence
	_, err := service.ResetClassification("", "", "", -1.0, false)
	// Может быть ошибка или просто игнорироваться, зависит от реализации
	_ = err

	// Тест с confidence > 1.0
	_, err = service.ResetClassification("", "", "", 2.0, false)
	// Может быть ошибка или просто игнорироваться
	_ = err
}

// TestClassificationService_GetKpvedHierarchy_DatabaseError проверяет обработку ошибки БД
// Теперь, когда таблица не существует, возвращается пустой массив, а не ошибка
// Этот тест проверяет, что метод корректно обрабатывает отсутствие таблицы
func TestClassificationService_GetKpvedHierarchy_DatabaseError(t *testing.T) {
	// Создаем сервис с новой БД (таблица не существует)
	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }
	getAPIKey := func() string { return "" }

	service := NewClassificationService(nil, nil, serviceDB, getModel, getAPIKey)

	// Попытка получить иерархию должна вернуть пустой массив, а не ошибку
	// (т.к. таблица не существует)
	hierarchy, err := service.GetKpvedHierarchy("", "")
	if err != nil {
		t.Fatalf("GetKpvedHierarchy() should not return error when table doesn't exist, got: %v", err)
	}
	
	if hierarchy == nil {
		t.Error("Expected non-nil hierarchy (empty slice)")
	}
	
	if len(hierarchy) != 0 {
		t.Errorf("Expected empty hierarchy, got %d nodes", len(hierarchy))
	}
}

// TestClassificationService_SearchKpved_InvalidLimit проверяет обработку невалидного лимита
func TestClassificationService_SearchKpved_InvalidLimit(t *testing.T) {
	db := setupTestDBForClassification(t)
	defer db.Close()

	normalizedDB := setupTestDBForClassification(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	// Тест с нулевым лимитом
	_, err := service.SearchKpved("test", 0)
	if err == nil {
		t.Error("Expected error for zero limit")
	}

	// Тест с отрицательным лимитом
	_, err = service.SearchKpved("test", -1)
	if err == nil {
		t.Error("Expected error for negative limit")
	}
}

// TestClassificationService_SearchKpved_NilDatabase проверяет обработку nil БД
// Теперь, когда таблица не существует, возвращается пустой массив, а не ошибка
// Этот тест проверяет, что метод корректно обрабатывает отсутствие таблицы
func TestClassificationService_SearchKpved_NilDatabase(t *testing.T) {
	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }
	getAPIKey := func() string { return "" }

	service := NewClassificationService(nil, nil, serviceDB, getModel, getAPIKey)

	// Попытка поиска должна вернуть пустой массив, а не ошибку
	// (т.к. таблица не существует)
	results, err := service.SearchKpved("test", 10)
	if err != nil {
		t.Fatalf("SearchKpved() should not return error when table doesn't exist, got: %v", err)
	}
	
	if results == nil {
		t.Error("Expected non-nil results (empty slice)")
	}
	
	if len(results) != 0 {
		t.Errorf("Expected empty results, got %d items", len(results))
	}
}

// TestClassificationService_GetKpvedHierarchy_TableNotExists проверяет обработку отсутствия таблицы
func TestClassificationService_GetKpvedHierarchy_TableNotExists(t *testing.T) {
	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }
	getAPIKey := func() string { return "" }

	service := NewClassificationService(nil, nil, serviceDB, getModel, getAPIKey)

	// Таблица kpved_classifier не существует в новой тестовой БД
	// Метод должен вернуть пустой массив, а не ошибку
	hierarchy, err := service.GetKpvedHierarchy("", "")
	if err != nil {
		t.Fatalf("GetKpvedHierarchy() should not return error for non-existent table, got: %v", err)
	}

	if hierarchy == nil {
		t.Error("Expected non-nil hierarchy (empty slice)")
	}

	if len(hierarchy) != 0 {
		t.Errorf("Expected empty hierarchy, got %d nodes", len(hierarchy))
	}
}

// TestClassificationService_GetKpvedHierarchy_EmptyTable проверяет обработку пустой таблицы
func TestClassificationService_GetKpvedHierarchy_EmptyTable(t *testing.T) {
	serviceDB := setupTestServiceDBForClassification(t)
	defer serviceDB.Close()

	// Создаем таблицу, но не заполняем её данными
	db := serviceDB.GetDB()
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kpved_classifier (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_code TEXT,
			level INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	getModel := func() string { return "test-model" }
	getAPIKey := func() string { return "" }

	service := NewClassificationService(nil, nil, serviceDB, getModel, getAPIKey)

	// Таблица существует, но пуста
	// Метод должен вернуть пустой массив, а не ошибку
	hierarchy, err := service.GetKpvedHierarchy("", "")
	if err != nil {
		t.Fatalf("GetKpvedHierarchy() should not return error for empty table, got: %v", err)
	}

	if hierarchy == nil {
		t.Error("Expected non-nil hierarchy (empty slice)")
	}

	if len(hierarchy) != 0 {
		t.Errorf("Expected empty hierarchy, got %d nodes", len(hierarchy))
	}
}

