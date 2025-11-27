package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestServiceDB создает временную тестовую БД
func setupTestServiceDB(t *testing.T) (*ServiceDB, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_service.db")
	
	db, err := NewServiceDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	
	// Схема инициализируется автоматически в NewServiceDBWithConfig
	return db, dbPath
}

// TestServiceDB_CRUD_Operations проверяет базовые CRUD операции
func TestServiceDB_CRUD_Operations(t *testing.T) {
	db, _ := setupTestServiceDB(t)
	defer db.Close()

	// Создаем клиента
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Читаем клиента
	readClient, err := db.GetClient(client.ID)
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if readClient.Name != "Test Client" {
		t.Errorf("Expected client name 'Test Client', got '%s'", readClient.Name)
	}

	// Обновляем клиента
	err = db.UpdateClient(client.ID, "Updated Client", "Updated Legal", "Updated Desc", "updated@test.com", "+9876543210", "TAX456", "US", "active")
	if err != nil {
		t.Fatalf("Failed to update client: %v", err)
	}

	// Проверяем обновление
	updatedClient, err := db.GetClient(client.ID)
	if err != nil {
		t.Fatalf("Failed to get updated client: %v", err)
	}

	if updatedClient.Name != "Updated Client" {
		t.Errorf("Expected updated client name 'Updated Client', got '%s'", updatedClient.Name)
	}

	// Удаляем клиента
	err = db.DeleteClient(client.ID)
	if err != nil {
		t.Fatalf("Failed to delete client: %v", err)
	}

	// Проверяем, что клиент удален
	_, err = db.GetClient(client.ID)
	if err == nil {
		t.Error("Expected error when getting deleted client")
	}
}

// TestServiceDB_ProjectOperations проверяет операции с проектами
func TestServiceDB_ProjectOperations(t *testing.T) {
	db, _ := setupTestServiceDB(t)
	defer db.Close()

	// Создаем клиента
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Создаем проект
	project, err := db.CreateClientProject(client.ID, "Test Project", "counterparty", "Project Description", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Читаем проект
	readProject, err := db.GetClientProject(project.ID)
	if err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}

	if readProject.Name != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", readProject.Name)
	}

	// Получаем все проекты клиента
	projects, err := db.GetClientProjects(client.ID)
	if err != nil {
		t.Fatalf("Failed to get client projects: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}
}

// TestServiceDB_NormalizedCounterparty_CRUD проверяет CRUD операции с нормализованными контрагентами
func TestServiceDB_NormalizedCounterparty_CRUD(t *testing.T) {
	db, _ := setupTestServiceDB(t)
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(client.ID, "Test Project", "counterparty", "Project Description", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем нормализованного контрагента
	err = db.SaveNormalizedCounterparty(
		project.ID,
		"REF001",
		"ООО Тест",
		"ООО Тест",
		"1234567890",
		"123456789",
		"",
		"Москва, ул. Тестовая, 1",
		"Москва, ул. Тестовая, 1",
		"+71234567890",
		"test@test.com",
		"Иванов Иван",
		"ООО",
		"Банк Тест",
		"40702810100000000001",
		"30101810100000000001",
		"044525225",
		0,
		0.9,
		false,
		"attributes",
		"Контрагенты",
		"",
	)
	if err != nil {
		t.Fatalf("Failed to save normalized counterparty: %v", err)
	}

	// Получаем нормализованных контрагентов по source_reference
	normalizedMap, err := db.GetNormalizedCounterpartiesBySourceReferences(project.ID, []string{"REF001"})
	if err != nil {
		t.Fatalf("Failed to get normalized counterparties: %v", err)
	}

	if !normalizedMap["REF001"] {
		t.Error("Expected normalized counterparty with REF001 to exist")
	}

	// Получаем все нормализованные контрагенты проекта для проверки
	allNormalized, _, err := db.GetNormalizedCounterparties(project.ID, 0, 100, "", "", "")
	if err != nil {
		t.Fatalf("Failed to get all normalized counterparties: %v", err)
	}

	if len(allNormalized) == 0 {
		t.Fatal("Expected at least one normalized counterparty")
	}

	normalized := allNormalized[0]

	if normalized.SourceName != "ООО Тест" {
		t.Errorf("Expected source name 'ООО Тест', got '%s'", normalized.SourceName)
	}

	if normalized.TaxID != "1234567890" {
		t.Errorf("Expected TaxID '1234567890', got '%s'", normalized.TaxID)
	}

	// Обновляем нормализованного контрагента
	err = db.UpdateNormalizedCounterparty(
		normalized.ID,
		"ООО Обновленный Тест",
		"1234567890",
		"123456789",
		"",
		"Москва, ул. Обновленная, 2",
		"Москва, ул. Обновленная, 2",
		"+79876543210",
		"updated@test.com",
		"Петров Петр",
		"ООО",
		"Банк Обновленный",
		"40702810200000000002",
		"30101810200000000002",
		"044525226",
		0.95,
		"api",
		"",
	)
	if err != nil {
		t.Fatalf("Failed to update normalized counterparty: %v", err)
	}

	// Проверяем обновление
	updated, err := db.GetNormalizedCounterparty(normalized.ID)
	if err != nil {
		t.Fatalf("Failed to get updated normalized counterparty: %v", err)
	}

	if updated.NormalizedName != "ООО Обновленный Тест" {
		t.Errorf("Expected updated name 'ООО Обновленный Тест', got '%s'", updated.NormalizedName)
	}
}

// TestServiceDB_TransactionRollback проверяет откат транзакций
func TestServiceDB_TransactionRollback(t *testing.T) {
	db, _ := setupTestServiceDB(t)
	defer db.Close()

	// Создаем клиента
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Начинаем транзакцию
	tx, err := db.conn.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Создаем проект в транзакции
	query := `INSERT INTO client_projects (client_id, name, project_type, description, source_system, status) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	result, err := tx.Exec(query, client.ID, "Test Project", "counterparty", "Description", "1C", "active")
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create project in transaction: %v", err)
	}

	projectID, _ := result.LastInsertId()

	// Откатываем транзакцию
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Проверяем, что проект не создан
	_, err = db.GetClientProject(int(projectID))
	if err == nil {
		t.Error("Expected error when getting project after rollback")
	}
}

// TestServiceDB_NormalizationSession проверяет операции с сессиями нормализации
func TestServiceDB_NormalizationSession(t *testing.T) {
	db, _ := setupTestServiceDB(t)
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(client.ID, "Test Project", "counterparty", "Project Description", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем проект БД для сессии
	projectDB, err := db.CreateProjectDatabase(project.ID, "Test DB", "/test/path.db", "Test Description", 0)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем сессию нормализации
	sessionID, err := db.CreateNormalizationSession(projectDB.ID, 1, 3600)
	if err != nil {
		t.Fatalf("Failed to create normalization session: %v", err)
	}

	// Получаем сессию
	session, err := db.GetNormalizationSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get normalization session: %v", err)
	}

	if session.Status != "running" {
		t.Errorf("Expected session status 'running', got '%s'", session.Status)
	}

	// Обновляем статус сессии
	finishedAt := time.Now()
	err = db.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
	if err != nil {
		t.Fatalf("Failed to update normalization session: %v", err)
	}

	// Проверяем обновление
	updatedSession, err := db.GetNormalizationSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	if updatedSession.Status != "completed" {
		t.Errorf("Expected updated session status 'completed', got '%s'", updatedSession.Status)
	}
}

// TestServiceDB_ConcurrentAccess проверяет конкурентный доступ к БД
func TestServiceDB_ConcurrentAccess(t *testing.T) {
	db, _ := setupTestServiceDB(t)
	defer db.Close()

	// Создаем клиента
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Создаем несколько проектов параллельно
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			_, err := db.CreateClientProject(client.ID, 
				"Test Project "+string(rune('A'+index)), 
				"counterparty", 
				"Description", 
				"1C", 
				0.8)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Успешно
		case err := <-errors:
			t.Errorf("Error creating project: %v", err)
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for project creation")
		}
	}

	// Проверяем, что все проекты созданы
	projects, err := db.GetClientProjects(client.ID)
	if err != nil {
		t.Fatalf("Failed to get client projects: %v", err)
	}

	if len(projects) != 10 {
		t.Errorf("Expected 10 projects, got %d", len(projects))
	}
}

// TestServiceDB_StopMechanism проверяет механизм остановки с транзакциями
func TestServiceDB_StopMechanism(t *testing.T) {
	db, _ := setupTestServiceDB(t)
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(client.ID, "Test Project", "counterparty", "Project Description", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем проект БД для сессии
	projectDB, err := db.CreateProjectDatabase(project.ID, "Test DB", "/test/path.db", "Test Description", 0)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем сессию нормализации
	sessionID, err := db.CreateNormalizationSession(projectDB.ID, 1, 3600)
	if err != nil {
		t.Fatalf("Failed to create normalization session: %v", err)
	}

	// Начинаем транзакцию для создания нормализованных контрагентов
	tx, err := db.conn.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Создаем несколько нормализованных контрагентов в транзакции
	for i := 0; i < 5; i++ {
		query := `INSERT INTO normalized_counterparties 
		          (client_project_id, source_reference, source_name, normalized_name, tax_id, quality_score) 
		          VALUES (?, ?, ?, ?, ?, ?)`
		_, err := tx.Exec(query, project.ID, "REF"+string(rune('0'+i)), "ООО Тест "+string(rune('0'+i)), 
			"ООО Тест "+string(rune('0'+i)), "123456789"+string(rune('0'+i)), 0.8)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert normalized counterparty: %v", err)
		}
	}

	// Откатываем транзакцию (имитируем откат при остановке)
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction on stop: %v", err)
	}

	// Останавливаем сессию ПОСЛЕ отката транзакции (имитируем остановку нормализации)
	finishedAt := time.Now()
	err = db.UpdateNormalizationSession(sessionID, "stopped", &finishedAt)
	if err != nil {
		t.Fatalf("Failed to stop normalization session: %v", err)
	}

	// Проверяем, что нормализованные контрагенты не созданы
	normalized, err := db.GetNormalizedCounterpartiesBySourceReferences(project.ID, []string{"REF0", "REF1", "REF2", "REF3", "REF4"})
	if err != nil {
		t.Fatalf("Failed to get normalized counterparties: %v", err)
	}

	if len(normalized) > 0 {
		t.Errorf("Expected no normalized counterparties after rollback, got %d", len(normalized))
	}
}

// TestServiceDB_InitializeSchema проверяет инициализацию схемы
func TestServiceDB_InitializeSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_schema.db")
	
	// Удаляем файл, если существует
	os.Remove(dbPath)
	
	db, err := NewServiceDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer db.Close()
	
	// Схема инициализируется автоматически в NewServiceDBWithConfig
	// Проверяем, что таблицы созданы, пытаясь создать клиента
	client, err := db.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "US", "user")
	if err != nil {
		t.Fatalf("Failed to create client after schema init: %v", err)
	}
	
	if client.ID == 0 {
		t.Error("Expected client ID to be non-zero")
	}
}

