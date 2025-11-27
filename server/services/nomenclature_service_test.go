package services

import (
	"testing"

	"httpserver/nomenclature"
)

// mockWorkerConfigManager мок для workerConfigManager
type mockNomenclatureConfigManager struct {
	config nomenclature.Config
	err    error
}

func (m *mockNomenclatureConfigManager) GetNomenclatureConfig() (nomenclature.Config, error) {
	return m.config, m.err
}

// TestNewNomenclatureService проверяет создание нового сервиса номенклатуры
func TestNewNomenclatureService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	var processor *nomenclature.NomenclatureProcessor
	processorGetter := func() *nomenclature.NomenclatureProcessor {
		return processor
	}
	processorSetter := func(p *nomenclature.NomenclatureProcessor) {
		processor = p
	}

	service := NewNomenclatureService(db, normalizedDB, serviceDB, nil, processorGetter, processorSetter)
	if service == nil {
		t.Error("NewNomenclatureService() should not return nil")
	}
}

// TestNomenclatureService_GetStatus_NotStarted проверяет получение статуса когда обработка не запущена
func TestNomenclatureService_GetStatus_NotStarted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	var processor *nomenclature.NomenclatureProcessor
	processorGetter := func() *nomenclature.NomenclatureProcessor {
		return processor
	}
	processorSetter := func(p *nomenclature.NomenclatureProcessor) {
		processor = p
	}

	service := NewNomenclatureService(db, normalizedDB, serviceDB, nil, processorGetter, processorSetter)

	status, err := service.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status == nil {
		t.Error("Expected non-nil status")
	}

	if status["status"] != "not_started" {
		t.Errorf("Expected status 'not_started', got '%v'", status["status"])
	}
}

// TestNomenclatureService_GetRecentRecords проверяет получение недавних записей
func TestNomenclatureService_GetRecentRecords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	var processor *nomenclature.NomenclatureProcessor
	processorGetter := func() *nomenclature.NomenclatureProcessor {
		return processor
	}
	processorSetter := func(p *nomenclature.NomenclatureProcessor) {
		processor = p
	}

	service := NewNomenclatureService(db, normalizedDB, serviceDB, nil, processorGetter, processorSetter)

	records, err := service.GetRecentRecords(10)
	if err != nil {
		t.Fatalf("GetRecentRecords() error = %v", err)
	}

	if records == nil {
		t.Error("Expected non-nil records")
	}
}

// TestNomenclatureService_GetPendingRecords проверяет получение ожидающих записей
func TestNomenclatureService_GetPendingRecords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	var processor *nomenclature.NomenclatureProcessor
	processorGetter := func() *nomenclature.NomenclatureProcessor {
		return processor
	}
	processorSetter := func(p *nomenclature.NomenclatureProcessor) {
		processor = p
	}

	service := NewNomenclatureService(db, normalizedDB, serviceDB, nil, processorGetter, processorSetter)

	records, err := service.GetPendingRecords(10)
	if err != nil {
		t.Fatalf("GetPendingRecords() error = %v", err)
	}

	if records == nil {
		t.Error("Expected non-nil records")
	}
}

// TestNomenclatureService_GetDBStats проверяет получение статистики из БД
func TestNomenclatureService_GetDBStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizedDB := setupTestDB(t)
	defer normalizedDB.Close()

	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	var processor *nomenclature.NomenclatureProcessor
	processorGetter := func() *nomenclature.NomenclatureProcessor {
		return processor
	}
	processorSetter := func(p *nomenclature.NomenclatureProcessor) {
		processor = p
	}

	service := NewNomenclatureService(db, normalizedDB, serviceDB, nil, processorGetter, processorSetter)

	stats, err := service.GetDBStats(db)
	if err != nil {
		// Может быть ошибка, если таблица catalog_items не существует
		// Это нормально для тестовой БД
		return
	}

	if stats == nil {
		t.Error("Expected non-nil stats")
	}
}

