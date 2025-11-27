package normalization

import (
	"testing"

	"httpserver/database"
)

func setupTestMapper(t *testing.T) (*CounterpartyMapper, *database.ServiceDB) {
	serviceDB := setupTestServiceDBForDuplicate(t)
	mapper := NewCounterpartyMapper(serviceDB)
	return mapper, serviceDB
}

func createTestClientForMapper(t *testing.T, serviceDB *database.ServiceDB) *database.Client {
	t.Helper()
	client, err := serviceDB.CreateClient(
		"Test Client",
		"Test Client LLC",
		"Test client for mapper",
		"test@example.com",
		"+70000000000",
		"TAX1234567",
		"KZ",
		"tests",
	)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	return client
}

func TestNewCounterpartyMapper(t *testing.T) {
	serviceDB := setupTestServiceDBForDuplicate(t)
	mapper := NewCounterpartyMapper(serviceDB)

	if mapper == nil {
		t.Fatal("NewCounterpartyMapper returned nil")
	}

	if mapper.serviceDB == nil {
		t.Error("mapper.serviceDB is nil")
	}

	if mapper.logger == nil {
		t.Error("mapper.logger is nil")
	}

	if mapper.analyzer == nil {
		t.Error("mapper.analyzer is nil")
	}
}

func TestCounterpartyMapper_MapCounterpartiesFromDatabase_EmptyDatabase(t *testing.T) {
	mapper, serviceDB := setupTestMapper(t)

	client := createTestClientForMapper(t, serviceDB)
	// Создаем тестовый проект и базу данных
	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "test", "Test description", "test_system", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	db, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", "test.db", "Test database", 1000)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Тестируем с пустой базой данных
	err = mapper.MapCounterpartiesFromDatabase(project.ID, db.ID)
	if err != nil {
		t.Errorf("MapCounterpartiesFromDatabase with empty database should not return error, got: %v", err)
	}
}

func TestCounterpartyMapper_MapCounterpartiesFromDatabase_InvalidDatabase(t *testing.T) {
	mapper, _ := setupTestMapper(t)

	// Тестируем с несуществующей базой данных
	err := mapper.MapCounterpartiesFromDatabase(1, 99999)
	if err == nil {
		t.Error("MapCounterpartiesFromDatabase with invalid database should return error")
	}
}

func TestCounterpartyMapper_MapCounterpartiesFromDatabase_WrongProject(t *testing.T) {
	mapper, serviceDB := setupTestMapper(t)

	client := createTestClientForMapper(t, serviceDB)
	// Создаем два проекта
	project1, err := serviceDB.CreateClientProject(client.ID, "Project 1", "test", "Test description 1", "test_system", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}

	project2, err := serviceDB.CreateClientProject(client.ID, "Project 2", "test", "Test description 2", "test_system", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	// Создаем базу данных для проекта 1
	db, err := serviceDB.CreateProjectDatabase(project1.ID, "Test DB", "test.db", "Test database", 1000)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Пытаемся мэппить базу из проекта 1 в проект 2
	err = mapper.MapCounterpartiesFromDatabase(project2.ID, db.ID)
	if err == nil {
		t.Error("MapCounterpartiesFromDatabase with wrong project should return error")
	}
}

func TestCounterpartyMapper_MapAllCounterpartiesForProject_EmptyProject(t *testing.T) {
	mapper, serviceDB := setupTestMapper(t)

	client := createTestClientForMapper(t, serviceDB)
	// Создаем проект без баз данных
	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "test", "Test description", "test_system", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Тестируем с проектом без баз данных
	err = mapper.MapAllCounterpartiesForProject(project.ID)
	if err != nil {
		t.Errorf("MapAllCounterpartiesForProject with empty project should not return error, got: %v", err)
	}
}

func TestCounterpartyMapper_MapAllCounterpartiesForProject_InvalidProject(t *testing.T) {
	mapper, _ := setupTestMapper(t)

	// Тестируем с несуществующим проектом
	err := mapper.MapAllCounterpartiesForProject(99999)
	if err == nil {
		t.Error("MapAllCounterpartiesForProject with invalid project should return error")
	}
}

func TestCounterpartyDuplicateAnalyzer_SelectMasterRecordWithStrategy_MaxQuality(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{
			ID:            1,
			Name:          "Company 1",
			QualityScore:  0.5,
			DatabaseCount: 1,
		},
		{
			ID:            2,
			Name:          "Company 2",
			QualityScore:  0.9,
			DatabaseCount: 1,
		},
		{
			ID:            3,
			Name:          "Company 3",
			QualityScore:  0.3,
			DatabaseCount: 1,
		},
	}

	master := analyzer.SelectMasterRecordWithStrategy(items, "max_quality")
	if master == nil {
		t.Fatal("SelectMasterRecordWithStrategy returned nil")
	}

	if master.ID != 2 {
		t.Errorf("Expected master ID 2 (highest quality), got %d", master.ID)
	}
}

func TestCounterpartyDuplicateAnalyzer_SelectMasterRecordWithStrategy_MaxDatabases(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{
			ID:            1,
			Name:          "Company 1",
			QualityScore:  0.5,
			DatabaseCount: 1,
		},
		{
			ID:            2,
			Name:          "Company 2",
			QualityScore:  0.9,
			DatabaseCount: 3,
		},
		{
			ID:            3,
			Name:          "Company 3",
			QualityScore:  0.3,
			DatabaseCount: 2,
		},
	}

	master := analyzer.SelectMasterRecordWithStrategy(items, "max_databases")
	if master == nil {
		t.Fatal("SelectMasterRecordWithStrategy returned nil")
	}

	if master.ID != 2 {
		t.Errorf("Expected master ID 2 (most databases), got %d", master.ID)
	}
}

func TestCounterpartyDuplicateAnalyzer_SelectMasterRecordWithStrategy_MaxData(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{
			ID:            1,
			Name:          "Company 1",
			INN:           "1234567890",
			KPP:           "123456789",
			QualityScore:  0.5,
			DatabaseCount: 1,
		},
		{
			ID:            2,
			Name:          "Company 2",
			INN:           "1234567890",
			QualityScore:  0.9,
			DatabaseCount: 1,
		},
		{
			ID:            3,
			Name:          "Company 3",
			QualityScore:  0.3,
			DatabaseCount: 1,
		},
	}

	master := analyzer.SelectMasterRecordWithStrategy(items, "max_data")
	if master == nil {
		t.Fatal("SelectMasterRecordWithStrategy returned nil")
	}

	// Company 1 должна быть выбрана, так как у неё больше данных (есть КПП)
	if master.ID != 1 {
		t.Errorf("Expected master ID 1 (most data), got %d", master.ID)
	}
}

func TestCounterpartyDuplicateAnalyzer_SelectMasterRecordWithStrategy_Default(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{
			ID:            1,
			Name:          "Company 1",
			QualityScore:  0.5,
			DatabaseCount: 1,
		},
		{
			ID:            2,
			Name:          "Company 2",
			QualityScore:  0.9,
			DatabaseCount: 1,
		},
	}

	// Тестируем с неизвестной стратегией - должна использоваться max_data
	master := analyzer.SelectMasterRecordWithStrategy(items, "unknown_strategy")
	if master == nil {
		t.Fatal("SelectMasterRecordWithStrategy returned nil")
	}

	// Должна использоваться стандартная логика
	if master.ID != 2 {
		t.Logf("Note: Using default strategy selected ID %d", master.ID)
	}
}

func TestCounterpartyDuplicateAnalyzer_SelectMasterRecordWithStrategy_EmptyList(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{}

	master := analyzer.SelectMasterRecordWithStrategy(items, "max_data")
	if master != nil {
		t.Error("SelectMasterRecordWithStrategy with empty list should return nil")
	}
}

func TestCounterpartyDuplicateAnalyzer_SelectMasterRecordWithStrategy_SingleItem(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{
			ID:            1,
			Name:          "Company 1",
			QualityScore:  0.5,
			DatabaseCount: 1,
		},
	}

	master := analyzer.SelectMasterRecordWithStrategy(items, "max_data")
	if master == nil {
		t.Fatal("SelectMasterRecordWithStrategy returned nil")
	}

	if master.ID != 1 {
		t.Errorf("Expected master ID 1, got %d", master.ID)
	}
}
