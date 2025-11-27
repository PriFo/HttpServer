package services

import (
	"testing"
)

// TestNewClassificationService проверяет создание нового сервиса классификации
func TestNewClassificationService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)
	if service == nil {
		t.Error("NewClassificationService() should not return nil")
	}
}

// TestClassificationService_GetKpvedHierarchy проверяет получение иерархии КПВЭД
func TestClassificationService_GetKpvedHierarchy(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	// Тест с пустыми параметрами (верхний уровень)
	hierarchy, err := service.GetKpvedHierarchy("", "")
	if err != nil {
		t.Fatalf("GetKpvedHierarchy() error = %v", err)
	}

	if hierarchy == nil {
		t.Error("Expected non-nil hierarchy")
	}
}

// TestClassificationService_GetKpvedHierarchy_WithParent проверяет получение иерархии с родителем
func TestClassificationService_GetKpvedHierarchy_WithParent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	hierarchy, err := service.GetKpvedHierarchy("A", "")
	if err != nil {
		t.Fatalf("GetKpvedHierarchy() error = %v", err)
	}

	if hierarchy == nil {
		t.Error("Expected non-nil hierarchy")
	}
}

// TestClassificationService_GetKpvedHierarchy_WithLevel проверяет получение иерархии с уровнем
func TestClassificationService_GetKpvedHierarchy_WithLevel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	hierarchy, err := service.GetKpvedHierarchy("", "1")
	if err != nil {
		t.Fatalf("GetKpvedHierarchy() error = %v", err)
	}

	if hierarchy == nil {
		t.Error("Expected non-nil hierarchy")
	}
}

// TestClassificationService_SearchKpved проверяет поиск по КПВЭД
func TestClassificationService_SearchKpved(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	results, err := service.SearchKpved("test", 10)
	if err != nil {
		t.Fatalf("SearchKpved() error = %v", err)
	}

	if results == nil {
		t.Error("Expected non-nil results")
	}
}

// TestClassificationService_SearchKpved_EmptyQuery проверяет поиск с пустым запросом
func TestClassificationService_SearchKpved_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	getModel := func() string { return "test-model" }

	service := NewClassificationService(db, normalizedDB, serviceDB, getModel, nil)

	results, err := service.SearchKpved("", 10)
	if err != nil {
		t.Fatalf("SearchKpved() error = %v", err)
	}

	if results == nil {
		t.Error("Expected non-nil results")
	}
}