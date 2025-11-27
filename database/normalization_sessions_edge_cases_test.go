package database

import (
	"testing"
	"time"
)

// TestTryCreateNormalizationSession_InvalidDatabaseID проверяет обработку невалидного ID БД
func TestTryCreateNormalizationSession_InvalidDatabaseID(t *testing.T) {
	db, err := NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer db.Close()

	// Пытаемся создать сессию для несуществующей БД
	sessionID, created, err := db.TryCreateNormalizationSession(0, 0, 3600)
	if err == nil {
		t.Error("Expected error for invalid database ID, got nil")
	}
	if created {
		t.Error("Expected session not to be created for invalid database ID")
	}
	if sessionID != 0 {
		t.Errorf("Expected session ID 0 for invalid database ID, got %d", sessionID)
	}

	// Пытаемся создать сессию для отрицательного ID
	sessionID, created, err = db.TryCreateNormalizationSession(-1, 0, 3600)
	if err == nil {
		t.Error("Expected error for negative database ID, got nil")
	}
	if created {
		t.Error("Expected session not to be created for negative database ID")
	}
	if sessionID != 0 {
		t.Errorf("Expected session ID 0 for negative database ID, got %d", sessionID)
	}
}

// TestTryCreateNormalizationSession_Timeout проверяет обработку таймаута
func TestTryCreateNormalizationSession_Timeout(t *testing.T) {
	db, err := NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient(
		"Test Client",
		"Test Client Legal Name",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"US",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.8,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем базу данных проекта
	projectDB, err := db.CreateProjectDatabase(
		project.ID,
		"Test DB",
		"/test/path.db",
		"Test Database",
		1024,
	)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем сессию с таймаутом
	sessionID, created, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 60)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if !created {
		t.Fatal("Session should be created")
	}

	// Проверяем, что таймаут установлен
	session, err := db.GetNormalizationSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if session.TimeoutSeconds != 60 {
		t.Errorf("Expected timeout 60 seconds, got %d", session.TimeoutSeconds)
	}
}

// TestTryCreateNormalizationSession_DefaultTimeout проверяет использование дефолтного таймаута
func TestTryCreateNormalizationSession_DefaultTimeout(t *testing.T) {
	db, err := NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient(
		"Test Client",
		"Test Client Legal Name",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"US",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.8,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем базу данных проекта
	projectDB, err := db.CreateProjectDatabase(
		project.ID,
		"Test DB",
		"/test/path.db",
		"Test Database",
		1024,
	)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем сессию с нулевым таймаутом (должен использоваться дефолтный)
	sessionID, created, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if !created {
		t.Fatal("Session should be created")
	}

	// Проверяем, что использован дефолтный таймаут (3600 секунд)
	session, err := db.GetNormalizationSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if session.TimeoutSeconds != 3600 {
		t.Errorf("Expected default timeout 3600 seconds, got %d", session.TimeoutSeconds)
	}
}

// TestTryCreateNormalizationSession_Priority проверяет установку приоритета
func TestTryCreateNormalizationSession_Priority(t *testing.T) {
	db, err := NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient(
		"Test Client",
		"Test Client Legal Name",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"US",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.8,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем базу данных проекта
	projectDB, err := db.CreateProjectDatabase(
		project.ID,
		"Test DB",
		"/test/path.db",
		"Test Database",
		1024,
	)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем сессию с приоритетом
	sessionID, created, err := db.TryCreateNormalizationSession(projectDB.ID, 10, 3600)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if !created {
		t.Fatal("Session should be created")
	}

	// Проверяем, что приоритет установлен
	session, err := db.GetNormalizationSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if session.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", session.Priority)
	}
}

// TestTryCreateNormalizationSession_FailedSession проверяет, что можно создать новую сессию после failed
func TestTryCreateNormalizationSession_FailedSession(t *testing.T) {
	db, err := NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient(
		"Test Client",
		"Test Client Legal Name",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"US",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.8,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем базу данных проекта
	projectDB, err := db.CreateProjectDatabase(
		project.ID,
		"Test DB",
		"/test/path.db",
		"Test Database",
		1024,
	)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем первую сессию
	sessionID1, created1, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create first session: %v", err)
	}
	if !created1 {
		t.Fatal("First session should be created")
	}

	// Помечаем сессию как failed
	err = db.UpdateNormalizationSession(sessionID1, "failed", nil)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Теперь можно создать новую сессию
	sessionID2, created2, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}
	if !created2 {
		t.Error("Second session should be created after first session failed")
	}
	if sessionID2 == sessionID1 {
		t.Error("Second session should have different ID")
	}
}

// TestTryCreateNormalizationSession_StoppedSession проверяет, что можно создать новую сессию после stopped
func TestTryCreateNormalizationSession_StoppedSession(t *testing.T) {
	db, err := NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer db.Close()

	// Создаем клиента и проект
	client, err := db.CreateClient(
		"Test Client",
		"Test Client Legal Name",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"US",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := db.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.8,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем базу данных проекта
	projectDB, err := db.CreateProjectDatabase(
		project.ID,
		"Test DB",
		"/test/path.db",
		"Test Database",
		1024,
	)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем первую сессию
	sessionID1, created1, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create first session: %v", err)
	}
	if !created1 {
		t.Fatal("First session should be created")
	}

	// Останавливаем сессию
	finishedAt := time.Now()
	err = db.UpdateNormalizationSession(sessionID1, "stopped", &finishedAt)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Теперь можно создать новую сессию
	sessionID2, created2, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}
	if !created2 {
		t.Error("Second session should be created after first session stopped")
	}
	if sessionID2 == sessionID1 {
		t.Error("Second session should have different ID")
	}
}

