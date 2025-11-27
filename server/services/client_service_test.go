package services

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"httpserver/database"
)

// mockServiceDB мок для database.ServiceDB
type mockServiceDB struct {
	getClientsWithStatsFunc    func() ([]map[string]interface{}, error)
	getClientFunc               func(id int) (*database.Client, error)
	getClientsByIDsFunc         func(ids []int) ([]*database.Client, error)
	createClientFunc            func(name, legalName, description, contactEmail, contactPhone, taxID, country, createdBy string) (*database.Client, error)
	updateClientFunc            func(id int, name, legalName, description, contactEmail, contactPhone, taxID, country, status string) error
	deleteClientFunc            func(id int) error
	getClientProjectsFunc       func(clientID int) ([]*database.ClientProject, error)
	getClientProjectFunc        func(id int) (*database.ClientProject, error)
	createClientProjectFunc     func(clientID int, name, projectType, description, sourceSystem string, targetQualityScore float64) (*database.ClientProject, error)
	updateClientProjectFunc     func(id int, name, projectType, description, sourceSystem, status string, targetQualityScore float64) error
	deleteClientProjectFunc     func(id int) error
	getProjectDatabasesFunc     func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error)
	getProjectDatabaseFunc      func(id int) (*database.ProjectDatabase, error)
	createProjectDatabaseFunc   func(projectID int, name, filePath, description string, fileSize int64) (*database.ProjectDatabase, error)
	updateProjectDatabaseFunc   func(id int, name, filePath, description string, isActive bool) error
	deleteProjectDatabaseFunc   func(id int) error
}

func (m *mockServiceDB) GetClientsWithStats() ([]map[string]interface{}, error) {
	if m.getClientsWithStatsFunc != nil {
		return m.getClientsWithStatsFunc()
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) GetClient(id int) (*database.Client, error) {
	if m.getClientFunc != nil {
		return m.getClientFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) GetClientsByIDs(ids []int) ([]*database.Client, error) {
	if m.getClientsByIDsFunc != nil {
		return m.getClientsByIDsFunc(ids)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) CreateClient(name, legalName, description, contactEmail, contactPhone, taxID, country, createdBy string) (*database.Client, error) {
	if m.createClientFunc != nil {
		return m.createClientFunc(name, legalName, description, contactEmail, contactPhone, taxID, country, createdBy)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) UpdateClient(id int, name, legalName, description, contactEmail, contactPhone, taxID, country, status string) error {
	if m.updateClientFunc != nil {
		return m.updateClientFunc(id, name, legalName, description, contactEmail, contactPhone, taxID, country, status)
	}
	return errors.New("not implemented")
}

func (m *mockServiceDB) DeleteClient(id int) error {
	if m.deleteClientFunc != nil {
		return m.deleteClientFunc(id)
	}
	return errors.New("not implemented")
}

func (m *mockServiceDB) GetClientProjects(clientID int) ([]*database.ClientProject, error) {
	if m.getClientProjectsFunc != nil {
		return m.getClientProjectsFunc(clientID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) GetClientProject(id int) (*database.ClientProject, error) {
	if m.getClientProjectFunc != nil {
		return m.getClientProjectFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) CreateClientProject(clientID int, name, projectType, description, sourceSystem string, targetQualityScore float64) (*database.ClientProject, error) {
	if m.createClientProjectFunc != nil {
		return m.createClientProjectFunc(clientID, name, projectType, description, sourceSystem, targetQualityScore)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) UpdateClientProject(id int, name, projectType, description, sourceSystem, status string, targetQualityScore float64) error {
	if m.updateClientProjectFunc != nil {
		return m.updateClientProjectFunc(id, name, projectType, description, sourceSystem, status, targetQualityScore)
	}
	return errors.New("not implemented")
}

func (m *mockServiceDB) DeleteClientProject(id int) error {
	if m.deleteClientProjectFunc != nil {
		return m.deleteClientProjectFunc(id)
	}
	return errors.New("not implemented")
}

func (m *mockServiceDB) GetProjectDatabases(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error) {
	if m.getProjectDatabasesFunc != nil {
		return m.getProjectDatabasesFunc(projectID, activeOnly)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) GetProjectDatabase(id int) (*database.ProjectDatabase, error) {
	if m.getProjectDatabaseFunc != nil {
		return m.getProjectDatabaseFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) CreateProjectDatabase(projectID int, name, filePath, description string, fileSize int64) (*database.ProjectDatabase, error) {
	if m.createProjectDatabaseFunc != nil {
		return m.createProjectDatabaseFunc(projectID, name, filePath, description, fileSize)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDB) UpdateProjectDatabase(id int, name, filePath, description string, isActive bool) error {
	if m.updateProjectDatabaseFunc != nil {
		return m.updateProjectDatabaseFunc(id, name, filePath, description, isActive)
	}
	return errors.New("not implemented")
}

func (m *mockServiceDB) DeleteProjectDatabase(id int) error {
	if m.deleteProjectDatabaseFunc != nil {
		return m.deleteProjectDatabaseFunc(id)
	}
	return errors.New("not implemented")
}

// setupTestServiceDB создает тестовую ServiceDB
func setupTestServiceDB(t *testing.T) *database.ServiceDB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_service.db")
	serviceDB, err := database.NewServiceDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test ServiceDB: %v", err)
	}
	return serviceDB
}

// TestNewClientService_Success проверяет успешное создание сервиса
func TestNewClientService_Success(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("NewClientService() returned error: %v", err)
	}
	if service == nil {
		t.Fatal("NewClientService() returned nil")
	}
	if service.serviceDB != serviceDB {
		t.Error("Service.serviceDB is not set correctly")
	}
}

// TestNewClientService_NilServiceDB проверяет обработку nil ServiceDB
func TestNewClientService_NilServiceDB(t *testing.T) {
	service, err := NewClientService(nil, nil, nil)
	if err == nil {
		t.Fatal("NewClientService() should return error for nil ServiceDB")
	}
	if service != nil {
		t.Error("NewClientService() should return nil service on error")
	}
}

// TestNewClientServiceWithLogger_Success проверяет создание сервиса с логгером
func TestNewClientServiceWithLogger_Success(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	logger := slog.Default()
	service, err := NewClientServiceWithLogger(serviceDB, nil, nil, logger)
	if err != nil {
		t.Fatalf("NewClientServiceWithLogger() returned error: %v", err)
	}
	if service == nil {
		t.Fatal("NewClientServiceWithLogger() returned nil")
	}
}

// TestNewClientServiceWithLogger_NilLogger проверяет создание сервиса с nil логгером
func TestNewClientServiceWithLogger_NilLogger(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientServiceWithLogger(serviceDB, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewClientServiceWithLogger() returned error: %v", err)
	}
	if service == nil {
		t.Fatal("NewClientServiceWithLogger() returned nil")
	}
	if service.logger == nil {
		t.Error("Logger should be set to default when nil is passed")
	}
}


// TestClientService_GetAllClients_Success проверяет успешное получение всех клиентов
func TestClientService_GetAllClients_Success(t *testing.T) {
	ctx := context.Background()

	// Создаем сервис с моком через рефлексию или используем реальную ServiceDB
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Используем реальную БД для теста
	// Создаем тестового клиента
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	clients, err := service.GetAllClients(ctx)
	if err != nil {
		t.Fatalf("GetAllClients() error = %v", err)
	}

	if len(clients) == 0 {
		t.Error("Expected at least one client")
	}
	found := false
	for _, c := range clients {
		if c.ID == client.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created client not found in results")
	}
}

// TestClientService_GetAllClients_NilContext проверяет обработку nil context
func TestClientService_GetAllClients_NilContext(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	var nilCtx context.Context
	_, err = service.GetAllClients(nilCtx)
	if err == nil {
		t.Error("Expected error for nil context")
	}
	if err != nil && !errors.Is(err, errors.New("context cannot be nil")) {
		t.Errorf("Expected 'context cannot be nil' error, got: %v", err)
	}
}

// TestClientService_GetAllClients_ContextCancelled проверяет обработку отмены контекста
func TestClientService_GetAllClients_ContextCancelled(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = service.GetAllClients(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// Проверяем, что ошибка содержит "context cancelled"
		if err.Error() == "" || !strings.Contains(err.Error(), "context cancelled") {
			t.Errorf("Expected context cancellation error, got: %v", err)
		}
	}
}

// TestClientService_GetClient_Success проверяет успешное получение клиента
func TestClientService_GetClient_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем тестового клиента
	createdClient, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	client, err := service.GetClient(ctx, createdClient.ID)
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if client.ID != createdClient.ID {
		t.Errorf("Client ID = %d, want %d", client.ID, createdClient.ID)
	}
	if client.Name != "Test Client" {
		t.Errorf("Client Name = %s, want 'Test Client'", client.Name)
	}
}

// TestClientService_GetClient_NotFound проверяет обработку несуществующего клиента
func TestClientService_GetClient_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.GetClient(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent client")
	}
}

// TestClientService_GetClient_InvalidID проверяет обработку невалидного ID
func TestClientService_GetClient_InvalidID(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.GetClient(ctx, 0)
	if err == nil {
		t.Error("Expected error for invalid client ID")
	}
	if err != nil && !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("Expected 'must be positive' error, got: %v", err)
	}
}

// TestClientService_CreateClient_Success проверяет успешное создание клиента
func TestClientService_CreateClient_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	client, err := service.CreateClient(ctx, "New Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US")
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	if client.Name != "New Client" {
		t.Errorf("Client Name = %s, want 'New Client'", client.Name)
	}
	if client.ID == 0 {
		t.Error("Client ID should be set")
	}
}

// TestClientService_CreateClient_EmptyName проверяет обработку пустого имени
func TestClientService_CreateClient_EmptyName(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.CreateClient(ctx, "", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US")
	if err == nil {
		t.Error("Expected error for empty name")
	}
	if err != nil && !strings.Contains(err.Error(), "name is required") {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}
}

// TestClientService_UpdateClient_Success проверяет успешное обновление клиента
func TestClientService_UpdateClient_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента
	createdClient, err := serviceDB.CreateClient("Original Name", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Обновляем клиента
	updatedClient, err := service.UpdateClient(ctx, createdClient.ID, "Updated Name", "Updated Legal", "New Description", "newemail@test.com", "+9876543210", "TAX456", "US", "active")
	if err != nil {
		t.Fatalf("UpdateClient() error = %v", err)
	}

	if updatedClient.Name != "Updated Name" {
		t.Errorf("Client Name = %s, want 'Updated Name'", updatedClient.Name)
	}
}

// TestClientService_DeleteClient_Success проверяет успешное удаление клиента
func TestClientService_DeleteClient_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента
	createdClient, err := serviceDB.CreateClient("To Delete", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Удаляем клиента
	err = service.DeleteClient(ctx, createdClient.ID)
	if err != nil {
		t.Fatalf("DeleteClient() error = %v", err)
	}

	// Проверяем, что клиент удален
	_, err = service.GetClient(ctx, createdClient.ID)
	if err == nil {
		t.Error("Client should be deleted")
	}
}

// TestClientService_GetClientProjects_Success проверяет успешное получение проектов клиента
func TestClientService_GetClientProjects_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	projects, err := service.GetClientProjects(ctx, client.ID)
	if err != nil {
		t.Fatalf("GetClientProjects() error = %v", err)
	}

	if len(projects) == 0 {
		t.Error("Expected at least one project")
	}
	found := false
	for _, p := range projects {
		if p.ID == project.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created project not found in results")
	}
}

// TestClientService_CreateClientProject_Success проверяет успешное создание проекта
func TestClientService_CreateClientProject_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := service.CreateClientProject(ctx, client.ID, "New Project", "type1", "Description", "source1", 90.0)
	if err != nil {
		t.Fatalf("CreateClientProject() error = %v", err)
	}

	if project.Name != "New Project" {
		t.Errorf("Project Name = %s, want 'New Project'", project.Name)
	}
	if project.ID == 0 {
		t.Error("Project ID should be set")
	}
}

// TestClientService_GetClientDatabases_Success проверяет успешное получение баз данных клиента
func TestClientService_GetClientDatabases_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента, проект и базу данных
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	db, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", "/path/to/db", "Description", 1024)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	databases, err := service.GetClientDatabases(ctx, client.ID)
	if err != nil {
		t.Fatalf("GetClientDatabases() error = %v", err)
	}

	if len(databases) == 0 {
		t.Error("Expected at least one database")
	}
	found := false
	for _, d := range databases {
		if d.ID == db.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created database not found in results")
	}
}

// TestClientService_CreateProjectDatabase_Success проверяет успешное создание базы данных проекта
func TestClientService_CreateProjectDatabase_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	database, err := service.CreateProjectDatabase(ctx, client.ID, project.ID, "New DB", "/path/to/newdb", "Description", 2048)
	if err != nil {
		t.Fatalf("CreateProjectDatabase() error = %v", err)
	}

	if database.Name != "New DB" {
		t.Errorf("Database Name = %s, want 'New DB'", database.Name)
	}
	if database.ID == 0 {
		t.Error("Database ID should be set")
	}
}

// TestClientService_GetAllClients_ServiceDBNil проверяет обработку nil serviceDB
func TestClientService_GetAllClients_ServiceDBNil(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Устанавливаем serviceDB в nil для теста
	service.serviceDB = nil
	_, err = service.GetAllClients(ctx)
	if err == nil {
		t.Error("Expected error for nil serviceDB")
	}
	if !strings.Contains(err.Error(), "service database not available") {
		t.Errorf("Expected 'service database not available' error, got: %v", err)
	}
}

// TestClientService_GetClientProject_Success проверяет успешное получение проекта клиента
func TestClientService_GetClientProject_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	result, err := service.GetClientProject(ctx, client.ID, project.ID)
	if err != nil {
		t.Fatalf("GetClientProject() error = %v", err)
	}

	if result.ID != project.ID {
		t.Errorf("Project ID = %d, want %d", result.ID, project.ID)
	}
	if result.Name != "Test Project" {
		t.Errorf("Project Name = %s, want 'Test Project'", result.Name)
	}
}

// TestClientService_GetClientProject_InvalidClientID проверяет обработку невалидного clientID
func TestClientService_GetClientProject_InvalidClientID(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.GetClientProject(ctx, 0, 1)
	if err == nil {
		t.Error("Expected error for invalid clientID")
	}
	if !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("Expected 'must be positive' error, got: %v", err)
	}
}

// TestClientService_GetClientProject_InvalidProjectID проверяет обработку невалидного projectID
func TestClientService_GetClientProject_InvalidProjectID(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.GetClientProject(ctx, 1, 0)
	if err == nil {
		t.Error("Expected error for invalid projectID")
	}
	if !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("Expected 'must be positive' error, got: %v", err)
	}
}

// TestClientService_UpdateClientProject_Success проверяет успешное обновление проекта клиента
func TestClientService_UpdateClientProject_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	result, err := service.UpdateClientProject(ctx, client.ID, project.ID, "Updated Project", "type2", "New Description", "source2", "active", 90.0)
	if err != nil {
		t.Fatalf("UpdateClientProject() error = %v", err)
	}

	if result.Name != "Updated Project" {
		t.Errorf("Project Name = %s, want 'Updated Project'", result.Name)
	}
}

// TestClientService_DeleteClientProject_Success проверяет успешное удаление проекта клиента
func TestClientService_DeleteClientProject_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	err = service.DeleteClientProject(ctx, client.ID, project.ID)
	if err != nil {
		t.Fatalf("DeleteClientProject() error = %v", err)
	}
}


// TestClientService_GetProjectDatabases_Success проверяет успешное получение баз данных проекта
func TestClientService_GetProjectDatabases_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента, проект и базу данных
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	_, err = serviceDB.CreateProjectDatabase(project.ID, "Test DB", "/path/to/db", "Description", 2048)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	databases, err := service.GetProjectDatabases(ctx, client.ID, project.ID)
	if err != nil {
		t.Fatalf("GetProjectDatabases() error = %v", err)
	}

	if len(databases) == 0 {
		t.Error("Expected at least one database")
	}
}

// TestClientService_GetProjectDatabase_Success проверяет успешное получение базы данных проекта
func TestClientService_GetProjectDatabase_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента, проект и базу данных
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	db, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", "/path/to/db", "Description", 2048)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	result, err := service.GetProjectDatabase(ctx, client.ID, project.ID, db.ID)
	if err != nil {
		t.Fatalf("GetProjectDatabase() error = %v", err)
	}

	if result.ID != db.ID {
		t.Errorf("Database ID = %d, want %d", result.ID, db.ID)
	}
	if result.Name != "Test DB" {
		t.Errorf("Database Name = %s, want 'Test DB'", result.Name)
	}
}

// TestClientService_UpdateProjectDatabase_Success проверяет успешное обновление базы данных проекта
func TestClientService_UpdateProjectDatabase_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента, проект и базу данных
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	db, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", "/path/to/db", "Description", 2048)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	result, err := service.UpdateProjectDatabase(ctx, client.ID, project.ID, db.ID, "Updated DB", "/path/to/updatedb", "New Description", true)
	if err != nil {
		t.Fatalf("UpdateProjectDatabase() error = %v", err)
	}

	if result.Name != "Updated DB" {
		t.Errorf("Database Name = %s, want 'Updated DB'", result.Name)
	}
}

// TestClientService_DeleteProjectDatabase_Success проверяет успешное удаление базы данных проекта
func TestClientService_DeleteProjectDatabase_Success(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента, проект и базу данных
	client, err := serviceDB.CreateClient("Test Client", "Legal Name", "Description", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Description", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	db, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", "/path/to/db", "Description", 2048)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = service.DeleteProjectDatabase(ctx, client.ID, project.ID, db.ID)
	if err != nil {
		t.Fatalf("DeleteProjectDatabase() error = %v", err)
	}
}


