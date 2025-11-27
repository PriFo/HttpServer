package database

import (
	"sync"
	"testing"
	"time"
)

// TestTryCreateNormalizationSession_Atomicity проверяет атомарность создания сессий
// Несколько горутин одновременно пытаются создать сессию для одной БД
// Только одна должна успешно создать сессию
func TestTryCreateNormalizationSession_Atomicity(t *testing.T) {
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

	// Количество горутин, которые будут пытаться создать сессию одновременно
	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make([]struct {
		sessionID int
		created   bool
		err       error
	}, numGoroutines)

	// Запускаем горутины одновременно
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sessionID, created, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
			results[index] = struct {
				sessionID int
				created   bool
				err       error
			}{sessionID, created, err}
		}(i)
	}

	// Ждем завершения всех горутин
	wg.Wait()

	// Проверяем результаты
	createdCount := 0
	var createdSessionID int
	for i, result := range results {
		if result.err != nil {
			t.Errorf("Goroutine %d returned error: %v", i, result.err)
		}
		if result.created {
			createdCount++
			if createdSessionID == 0 {
				createdSessionID = result.sessionID
			} else if createdSessionID != result.sessionID {
				t.Errorf("Multiple sessions created with different IDs: %d and %d", createdSessionID, result.sessionID)
			}
		}
	}

	// Только одна сессия должна быть создана
	if createdCount != 1 {
		t.Errorf("Expected exactly 1 session to be created, got %d", createdCount)
	}

	// Проверяем, что сессия действительно создана в БД
	session, err := db.GetNormalizationSession(createdSessionID)
	if err != nil {
		t.Fatalf("Failed to get created session: %v", err)
	}
	if session == nil {
		t.Fatal("Created session not found in database")
	}
	if session.Status != "running" {
		t.Errorf("Expected session status 'running', got '%s'", session.Status)
	}
	if session.ProjectDatabaseID != projectDB.ID {
		t.Errorf("Expected project_database_id %d, got %d", projectDB.ID, session.ProjectDatabaseID)
	}
}

// TestTryCreateNormalizationSession_WithActiveSession проверяет, что сессия не создается, если уже есть активная
func TestTryCreateNormalizationSession_WithActiveSession(t *testing.T) {
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
	if sessionID1 == 0 {
		t.Fatal("First session ID should not be zero")
	}

	// Пытаемся создать вторую сессию для той же БД
	sessionID2, created2, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to try create second session: %v", err)
	}
	if created2 {
		t.Error("Second session should not be created when active session exists")
	}
	if sessionID2 != 0 {
		t.Errorf("Second session ID should be 0 when not created, got %d", sessionID2)
	}

	// Проверяем, что в БД только одна активная сессия
	session, err := db.GetLastNormalizationSession(projectDB.ID)
	if err != nil {
		t.Fatalf("Failed to get last session: %v", err)
	}
	if session == nil {
		t.Fatal("Session not found")
	}
	if session.ID != sessionID1 {
		t.Errorf("Expected session ID %d, got %d", sessionID1, session.ID)
	}
	if session.Status != "running" {
		t.Errorf("Expected session status 'running', got '%s'", session.Status)
	}
}

// TestTryCreateNormalizationSession_AfterSessionCompleted проверяет, что можно создать новую сессию после завершения предыдущей
func TestTryCreateNormalizationSession_AfterSessionCompleted(t *testing.T) {
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

	// Завершаем первую сессию
	finishedAt := time.Now()
	err = db.UpdateNormalizationSession(sessionID1, "completed", &finishedAt)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Теперь можно создать новую сессию
	sessionID2, created2, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}
	if !created2 {
		t.Error("Second session should be created after first session completed")
	}
	if sessionID2 == 0 {
		t.Fatal("Second session ID should not be zero")
	}
	if sessionID2 == sessionID1 {
		t.Error("Second session should have different ID")
	}

	// Проверяем, что вторая сессия создана и активна
	session, err := db.GetNormalizationSession(sessionID2)
	if err != nil {
		t.Fatalf("Failed to get second session: %v", err)
	}
	if session == nil {
		t.Fatal("Second session not found")
	}
	if session.ID != sessionID2 {
		t.Errorf("Expected session ID %d, got %d", sessionID2, session.ID)
	}
	if session.Status != "running" {
		t.Errorf("Expected session status 'running', got '%s'", session.Status)
	}

	// Проверяем, что первая сессия завершена
	session1, err := db.GetNormalizationSession(sessionID1)
	if err != nil {
		t.Fatalf("Failed to get first session: %v", err)
	}
	if session1 == nil {
		t.Fatal("First session not found")
	}
	if session1.Status != "completed" {
		t.Errorf("Expected first session status 'completed', got '%s'", session1.Status)
	}
}

// TestTryCreateNormalizationSession_DifferentDatabases проверяет, что можно создать сессии для разных БД одновременно
func TestTryCreateNormalizationSession_DifferentDatabases(t *testing.T) {
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

	// Создаем две базы данных проекта
	projectDB1, err := db.CreateProjectDatabase(
		project.ID,
		"Test DB 1",
		"/test/path1.db",
		"Test Database 1",
		1024,
	)
	if err != nil {
		t.Fatalf("Failed to create project database 1: %v", err)
	}

	projectDB2, err := db.CreateProjectDatabase(
		project.ID,
		"Test DB 2",
		"/test/path2.db",
		"Test Database 2",
		1024,
	)
	if err != nil {
		t.Fatalf("Failed to create project database 2: %v", err)
	}

	// Создаем сессии для разных БД одновременно
	sessionID1, created1, err := db.TryCreateNormalizationSession(projectDB1.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create session for DB 1: %v", err)
	}
	if !created1 {
		t.Error("Session for DB 1 should be created")
	}

	sessionID2, created2, err := db.TryCreateNormalizationSession(projectDB2.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create session for DB 2: %v", err)
	}
	if !created2 {
		t.Error("Session for DB 2 should be created")
	}

	// Проверяем, что обе сессии созданы и активны
	session1, err := db.GetNormalizationSession(sessionID1)
	if err != nil {
		t.Fatalf("Failed to get session 1: %v", err)
	}
	if session1 == nil || session1.Status != "running" {
		t.Error("Session 1 should be running")
	}

	session2, err := db.GetNormalizationSession(sessionID2)
	if err != nil {
		t.Fatalf("Failed to get session 2: %v", err)
	}
	if session2 == nil || session2.Status != "running" {
		t.Error("Session 2 should be running")
	}

	// Проверяем, что сессии принадлежат разным БД
	if session1.ProjectDatabaseID == session2.ProjectDatabaseID {
		t.Error("Sessions should belong to different databases")
	}
}

