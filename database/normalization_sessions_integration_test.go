package database

import (
	"sync"
	"testing"
	"time"
)

// TestNormalizationSessions_ConcurrentProcessing проверяет, что несколько воркеров не могут одновременно обрабатывать одну БД
func TestNormalizationSessions_ConcurrentProcessing(t *testing.T) {
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

	// Симулируем работу нескольких воркеров, которые пытаются обработать одну БД
	const numWorkers = 5
	var wg sync.WaitGroup
	successfulWorkers := make([]int, 0)
	var mu sync.Mutex

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Пытаемся создать сессию
			sessionID, created, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
			if err != nil {
				t.Errorf("Worker %d: Failed to try create session: %v", workerID, err)
				return
			}

			if created {
				mu.Lock()
				successfulWorkers = append(successfulWorkers, workerID)
				mu.Unlock()

				// Симулируем обработку БД
				time.Sleep(10 * time.Millisecond)

				// Завершаем сессию
				finishedAt := time.Now()
				err = db.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
				if err != nil {
					t.Errorf("Worker %d: Failed to update session: %v", workerID, err)
				}
			} else {
				// Воркер не смог создать сессию - это нормально, значит другой воркер уже обрабатывает
				t.Logf("Worker %d: Session not created (another worker is processing)", workerID)
			}
		}(i)
	}

	// Ждем завершения всех воркеров
	wg.Wait()

	// Только один воркер должен был успешно создать сессию
	if len(successfulWorkers) != 1 {
		t.Errorf("Expected exactly 1 successful worker, got %d: %v", len(successfulWorkers), successfulWorkers)
	}

	// Проверяем, что в БД нет активных сессий
	session, err := db.GetLastNormalizationSession(projectDB.ID)
	if err != nil {
		t.Fatalf("Failed to get last session: %v", err)
	}
	if session != nil && session.Status == "running" {
		t.Error("There should be no active sessions after all workers completed")
	}
}

// TestNormalizationSessions_SequentialProcessing проверяет последовательную обработку БД разными воркерами
func TestNormalizationSessions_SequentialProcessing(t *testing.T) {
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

	// Симулируем последовательную обработку БД несколькими воркерами
	const numWorkers = 3
	sessionIDs := make([]int, 0, numWorkers)

	for i := 0; i < numWorkers; i++ {
		// Пытаемся создать сессию
		sessionID, created, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
		if err != nil {
			t.Fatalf("Worker %d: Failed to try create session: %v", i, err)
		}

		if !created {
			t.Fatalf("Worker %d: Expected to create session, but it was not created", i)
		}

		sessionIDs = append(sessionIDs, sessionID)

		// Симулируем обработку БД
		time.Sleep(5 * time.Millisecond)

		// Завершаем сессию
		finishedAt := time.Now()
		err = db.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
		if err != nil {
			t.Fatalf("Worker %d: Failed to update session: %v", i, err)
		}
	}

	// Проверяем, что все сессии созданы с разными ID
	uniqueIDs := make(map[int]bool)
	for _, id := range sessionIDs {
		if uniqueIDs[id] {
			t.Errorf("Duplicate session ID: %d", id)
		}
		uniqueIDs[id] = true
	}

	// Проверяем, что в БД нет активных сессий
	session, err := db.GetLastNormalizationSession(projectDB.ID)
	if err != nil {
		t.Fatalf("Failed to get last session: %v", err)
	}
	if session != nil && session.Status == "running" {
		t.Error("There should be no active sessions after all workers completed")
	}
}

// TestNormalizationSessions_MultipleDatabases проверяет параллельную обработку разных БД
func TestNormalizationSessions_MultipleDatabases(t *testing.T) {
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

	// Создаем несколько баз данных проекта
	const numDatabases = 5
	projectDBs := make([]*ProjectDatabase, 0, numDatabases)
	for i := 0; i < numDatabases; i++ {
		projectDB, err := db.CreateProjectDatabase(
			project.ID,
			"Test DB",
			"/test/path.db",
			"Test Database",
			1024,
		)
		if err != nil {
			t.Fatalf("Failed to create project database %d: %v", i, err)
		}
		projectDBs = append(projectDBs, projectDB)
	}

	// Пытаемся создать сессии для всех БД одновременно
	var wg sync.WaitGroup
	createdSessions := make([]int, 0, numDatabases)
	var mu sync.Mutex

		for i, projectDB := range projectDBs {
		wg.Add(1)
		go func(dbID int, projectDB *ProjectDatabase) {
			defer wg.Done()

			sessionID, created, err := db.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
			if err != nil {
				t.Errorf("Database %d: Failed to try create session: %v", dbID, err)
				return
			}

			if created {
				mu.Lock()
				createdSessions = append(createdSessions, sessionID)
				mu.Unlock()
			} else {
				t.Errorf("Database %d: Expected to create session, but it was not created", dbID)
			}
		}(i, projectDB)
	}

	wg.Wait()

	// Все БД должны иметь активные сессии
	if len(createdSessions) != numDatabases {
		t.Errorf("Expected %d sessions to be created, got %d", numDatabases, len(createdSessions))
	}

	// Проверяем, что каждая БД имеет активную сессию
	for _, projectDB := range projectDBs {
		session, err := db.GetLastNormalizationSession(projectDB.ID)
		if err != nil {
			t.Fatalf("Failed to get last session for DB %d: %v", projectDB.ID, err)
		}
		if session == nil {
			t.Errorf("Database %d: Expected active session, got nil", projectDB.ID)
		} else if session.Status != "running" {
			t.Errorf("Database %d: Expected session status 'running', got '%s'", projectDB.ID, session.Status)
		}
	}
}

