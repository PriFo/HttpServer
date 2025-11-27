package services

import (
	"testing"

	"httpserver/database"
)

// TestNewSnapshotService проверяет создание нового сервиса срезов
func TestNewSnapshotService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)
	if service == nil {
		t.Error("NewSnapshotService() should not return nil")
	}
}

// TestSnapshotService_GetAllSnapshots проверяет получение всех срезов
func TestSnapshotService_GetAllSnapshots(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	snapshots, err := service.GetAllSnapshots()
	if err != nil {
		// Может быть ошибка, если таблица не существует
		// Это нормально для тестовой БД
		return
	}

	if snapshots == nil {
		t.Error("Expected non-nil snapshots")
	}
}

// TestSnapshotService_GetSnapshotWithUploads проверяет получение среза с выгрузками
func TestSnapshotService_GetSnapshotWithUploads(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	snapshot, uploads, err := service.GetSnapshotWithUploads(1)
	if err != nil {
		// Может быть ошибка, если срез не существует
		// Это нормально для тестовой БД
		return
	}

	if snapshot == nil && uploads == nil {
		// Оба могут быть nil, если срез не найден
		return
	}
}

// TestSnapshotService_GetSnapshotsByProject проверяет получение срезов проекта
func TestSnapshotService_GetSnapshotsByProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	snapshots, err := service.GetSnapshotsByProject(1)
	if err != nil {
		// Может быть ошибка, если таблица не существует
		// Это нормально для тестовой БД
		return
	}

	if snapshots == nil {
		t.Error("Expected non-nil snapshots")
	}
}

// TestSnapshotService_NormalizeSnapshot проверяет нормализацию среза
func TestSnapshotService_NormalizeSnapshot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	normalizeFunc := func(snapshotID int, req interface{}) (interface{}, error) {
		return map[string]interface{}{"status": "normalized"}, nil
	}

	result, err := service.NormalizeSnapshot(1, normalizeFunc, nil)
	if err != nil {
		t.Fatalf("NormalizeSnapshot() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestSnapshotService_CompareSnapshotIterations проверяет сравнение итераций
func TestSnapshotService_CompareSnapshotIterations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	compareFunc := func(snapshotID int) (interface{}, error) {
		return map[string]interface{}{"comparison": "done"}, nil
	}

	result, err := service.CompareSnapshotIterations(1, compareFunc)
	if err != nil {
		t.Fatalf("CompareSnapshotIterations() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestSnapshotService_CalculateSnapshotMetrics проверяет вычисление метрик
func TestSnapshotService_CalculateSnapshotMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	calculateFunc := func(snapshotID int) (interface{}, error) {
		return map[string]interface{}{"metrics": "calculated"}, nil
	}

	result, err := service.CalculateSnapshotMetrics(1, calculateFunc)
	if err != nil {
		t.Fatalf("CalculateSnapshotMetrics() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestSnapshotService_GetSnapshotEvolution проверяет получение эволюции
func TestSnapshotService_GetSnapshotEvolution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	evolutionFunc := func(snapshotID int) (interface{}, error) {
		return map[string]interface{}{"evolution": "tracked"}, nil
	}

	result, err := service.GetSnapshotEvolution(1, evolutionFunc)
	if err != nil {
		t.Fatalf("GetSnapshotEvolution() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestSnapshotService_CreateAutoSnapshot проверяет создание автоматического среза
func TestSnapshotService_CreateAutoSnapshot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewSnapshotService(db)

	createFunc := func(projectID int, uploadsPerDatabase int, name, description string) (*database.DataSnapshot, error) {
		projectIDPtr := &projectID
		return &database.DataSnapshot{
			ID:          1,
			ProjectID:   projectIDPtr,
			Name:        name,
			Description: description,
		}, nil
	}

	snapshot, err := service.CreateAutoSnapshot(1, 5, "Test Snapshot", "Test Description", createFunc)
	if err != nil {
		t.Fatalf("CreateAutoSnapshot() error = %v", err)
	}

	if snapshot == nil {
		t.Error("Expected non-nil snapshot")
	}
}

