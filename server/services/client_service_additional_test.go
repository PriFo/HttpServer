package services

import (
	"context"
	"path/filepath"
	"testing"

	"httpserver/database"
)

// setupTestServiceDBAdditional создает тестовую ServiceDB для дополнительных тестов
func setupTestServiceDBAdditional(t *testing.T) *database.ServiceDB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_service.db")
	serviceDB, err := database.NewServiceDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test ServiceDB: %v", err)
	}
	return serviceDB
}

// TestClientService_GetServiceDB проверяет получение ServiceDB
func TestClientService_GetServiceDB(t *testing.T) {
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	result := service.GetServiceDB()
	if result != serviceDB {
		t.Error("GetServiceDB() should return the same ServiceDB instance")
	}
}

// TestClientService_GetAllClients_EmptyResult проверяет обработку пустого результата
func TestClientService_GetAllClients_EmptyResult(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	clients, err := service.GetAllClients(ctx)
	if err != nil {
		t.Fatalf("GetAllClients() error = %v", err)
	}

	if clients == nil {
		t.Error("Expected non-nil empty slice, got nil")
	}
	if len(clients) != 0 {
		t.Errorf("Expected empty slice, got %d clients", len(clients))
	}
}

// TestClientService_UpdateClient_NotFound проверяет обработку несуществующего клиента
func TestClientService_UpdateClient_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.UpdateClient(ctx, 99999, "Updated Name", "Legal", "Desc", "email@test.com", "+1234567890", "TAX123", "US", "active")
	if err == nil {
		t.Error("Expected error for non-existent client")
	}
}

// TestClientService_DeleteClient_NotFound проверяет обработку несуществующего клиента
func TestClientService_DeleteClient_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	err = service.DeleteClient(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent client")
	}
}

// TestClientService_GetClientProjects_NotFound проверяет обработку несуществующего клиента
func TestClientService_GetClientProjects_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	projects, err := service.GetClientProjects(ctx, 99999)
	if err != nil {
		// Может быть ошибка или пустой список, зависит от реализации
		return
	}

	if projects == nil {
		t.Error("Expected non-nil empty slice, got nil")
	}
}

// TestClientService_CreateClientProject_InvalidClientID проверяет обработку невалидного clientID
func TestClientService_CreateClientProject_InvalidClientID(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = service.CreateClientProject(ctx, 0, "Project", "type", "Desc", "source", 85.0)
	if err == nil {
		t.Error("Expected error for invalid clientID")
	}
}

// TestClientService_UpdateClientProject_NotFound проверяет обработку несуществующего проекта
func TestClientService_UpdateClientProject_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента
	client, err := serviceDB.CreateClient("Test Client", "Legal", "Desc", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	_, err = service.UpdateClientProject(ctx, client.ID, 99999, "Updated", "type", "Desc", "source", "active", 90.0)
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

// TestClientService_DeleteClientProject_NotFound проверяет обработку несуществующего проекта
func TestClientService_DeleteClientProject_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента
	client, err := serviceDB.CreateClient("Test Client", "Legal", "Desc", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	err = service.DeleteClientProject(ctx, client.ID, 99999)
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

// TestClientService_GetProjectDatabase_NotFound проверяет обработку несуществующей БД
func TestClientService_GetProjectDatabase_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal", "Desc", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Desc", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	_, err = service.GetProjectDatabase(ctx, client.ID, project.ID, 99999)
	if err == nil {
		t.Error("Expected error for non-existent database")
	}
}

// TestClientService_CreateProjectDatabase_InvalidProjectID проверяет обработку невалидного projectID
func TestClientService_CreateProjectDatabase_InvalidProjectID(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента
	client, err := serviceDB.CreateClient("Test Client", "Legal", "Desc", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	_, err = service.CreateProjectDatabase(ctx, client.ID, 0, "DB", "/path", "Desc", 1024)
	if err == nil {
		t.Error("Expected error for invalid projectID")
	}
}

// TestClientService_UpdateProjectDatabase_NotFound проверяет обработку несуществующей БД
func TestClientService_UpdateProjectDatabase_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal", "Desc", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Desc", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	_, err = service.UpdateProjectDatabase(ctx, client.ID, project.ID, 99999, "Updated", "/path", "Desc", true)
	if err == nil {
		t.Error("Expected error for non-existent database")
	}
}

// TestClientService_DeleteProjectDatabase_NotFound проверяет обработку несуществующей БД
func TestClientService_DeleteProjectDatabase_NotFound(t *testing.T) {
	ctx := context.Background()
	serviceDB := setupTestServiceDBAdditional(t)
	defer serviceDB.Close()

	service, err := NewClientService(serviceDB, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Legal", "Desc", "email@test.com", "+1234567890", "TAX123", "US", "test")
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "type1", "Desc", "source1", 85.0)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	err = service.DeleteProjectDatabase(ctx, client.ID, project.ID, 99999)
	if err == nil {
		t.Error("Expected error for non-existent database")
	}
}

