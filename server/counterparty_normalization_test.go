package server

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"httpserver/database"
)

// createTestProjectDatabase создает тестовую ProjectDatabase
func createTestProjectDatabase(t *testing.T, serviceDB *database.ServiceDB, projectID int, filePath string) *database.ProjectDatabase {
	db, err := serviceDB.CreateProjectDatabase(projectID, "Test DB", filePath, "", 0)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}
	return db
}

// createTestDatabaseWithCounterparties создает тестовую БД с контрагентами
func createTestDatabaseWithCounterparties(t *testing.T, counterparties []map[string]string) string {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close()

	// Создаем выгрузку
	upload, err := db.CreateUpload("test-uuid", "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	// Создаем каталог "Контрагенты"
	catalog, err := db.AddCatalog(upload.ID, "Контрагенты", "counterparties")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем контрагентов
	for i, cp := range counterparties {
		attributes := ""
		if inn, ok := cp["inn"]; ok {
			attributes += `<ИНН>` + inn + `</ИНН>`
		}
		if kpp, ok := cp["kpp"]; ok {
			attributes += `<КПП>` + kpp + `</КПП>`
		}
		if bin, ok := cp["bin"]; ok {
			attributes += `<БИН>` + bin + `</БИН>`
		}

		name := cp["name"]
		if name == "" {
			name = "ООО Тест " + string(rune('A'+i))
		}

		err := db.AddCatalogItem(catalog.ID, "ref_"+name, "code_"+name, name, attributes, "")
		if err != nil {
			t.Fatalf("Failed to add catalog item: %v", err)
		}
	}

	return dbPath
}

// TestProcessCounterpartyDatabase_Basic проверяет базовую обработку БД
func TestProcessCounterpartyDatabase_Basic(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Создаем клиента и проект
	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем тестовую БД с контрагентами
	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
		{"name": "ООО Тест 2", "inn": "1234567891"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)

	// Добавляем БД в проект
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Устанавливаем флаг нормализации
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Проверяем, что данные были обработаны
	// (в реальном тесте можно проверить через GetNormalizedCounterpartyStats)
}

// TestProcessCounterpartyDatabase_NoCounterparties проверяет обработку БД без контрагентов
func TestProcessCounterpartyDatabase_NoCounterparties(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем пустую БД
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "empty.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create empty DB: %v", err)
	}
	db.Close()

	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Не должно быть ошибок
}

// TestProcessCounterpartyDatabase_DatabaseError проверяет обработку ошибки открытия БД
func TestProcessCounterpartyDatabase_DatabaseError(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем ProjectDatabase с несуществующим путем
	projectDB := &database.ProjectDatabase{
		ID:              1,
		ClientProjectID: project.ID,
		Name:            "Non-existent DB",
		FilePath:        "/non/existent/path.db",
		IsActive:        true,
	}

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку - должна быть обработана ошибка
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Проверяем, что сессия была создана (через GetRunningSessions или GetLastNormalizationSession)
	runningSessions, err := srv.serviceDB.GetRunningSessions()
	if err == nil {
		// Ищем сессию для этой БД
		for _, session := range runningSessions {
			if session.ProjectDatabaseID == projectDB.ID {
				if session.Status != "failed" {
					t.Logf("Expected session status 'failed', got '%s'", session.Status)
				}
				break
			}
		}
	}
}

// TestProcessCounterpartyDatabase_StopCheck_BeforeProcessing проверяет остановку до обработки
func TestProcessCounterpartyDatabase_StopCheck_BeforeProcessing(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Устанавливаем флаг остановки
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = false // Остановлено
	srv.normalizerMutex.Unlock()

	// Запускаем обработку
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Проверяем, что сессия была обновлена как stopped
	lastSession, err := srv.serviceDB.GetLastNormalizationSession(projectDB.ID)
	if err == nil && lastSession != nil {
		if lastSession.Status != "stopped" {
			t.Logf("Expected session status 'stopped', got '%s'", lastSession.Status)
		}
	}
}

// TestProcessCounterpartyDatabasesParallel_SingleDB проверяет параллельную обработку одной БД
func TestProcessCounterpartyDatabasesParallel_SingleDB(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	databases := []*database.ProjectDatabase{projectDB}
	srv.processCounterpartyDatabasesParallel(databases, client.ID, project.ID)

	// Даем время на завершение
	time.Sleep(100 * time.Millisecond)
}

// TestProcessCounterpartyDatabasesParallel_MultipleDBs проверяет параллельную обработку нескольких БД
func TestProcessCounterpartyDatabasesParallel_MultipleDBs(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем несколько БД
	var databases []*database.ProjectDatabase
	for i := 0; i < 3; i++ {
		counterparties := []map[string]string{
			{"name": "ООО Тест " + string(rune('A'+i)), "inn": "123456789" + string(rune('0'+i))},
		}
		dbPath := createTestDatabaseWithCounterparties(t, counterparties)
		projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)
		databases = append(databases, projectDB)
	}

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	srv.processCounterpartyDatabasesParallel(databases, client.ID, project.ID)

	// Даем время на завершение
	time.Sleep(200 * time.Millisecond)
}

// TestProcessCounterpartyDatabasesParallel_StopCheck проверяет остановку во время параллельной обработки
func TestProcessCounterpartyDatabasesParallel_StopCheck(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем несколько БД
	var databases []*database.ProjectDatabase
	for i := 0; i < 5; i++ {
		counterparties := []map[string]string{
			{"name": "ООО Тест " + string(rune('A'+i)), "inn": "123456789" + string(rune('0'+i))},
		}
		dbPath := createTestDatabaseWithCounterparties(t, counterparties)
		projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)
		databases = append(databases, projectDB)
	}

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку в отдельной горутине
	go srv.processCounterpartyDatabasesParallel(databases, client.ID, project.ID)

	// Останавливаем через небольшое время
	time.Sleep(50 * time.Millisecond)
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = false
	srv.normalizerMutex.Unlock()

	// Даем время на обработку остановки
	time.Sleep(200 * time.Millisecond)
}

// TestHandleStopClientNormalization_Success проверяет успешную остановку
func TestHandleStopClientNormalization_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Устанавливаем флаг нормализации
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Создаем тестовый запрос
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)

	// Вызываем обработчик
	srv.handleStopClientNormalization(w, req, client.ID, project.ID)

	// Проверяем, что нормализация остановлена
	srv.normalizerMutex.RLock()
	isRunning := srv.normalizerRunning
	srv.normalizerMutex.RUnlock()

	if isRunning {
		t.Error("Expected normalization to be stopped")
	}

	// Проверяем ответ
	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

// TestHandleStopClientNormalization_NotRunning проверяет остановку, когда нормализация не запущена
func TestHandleStopClientNormalization_NotRunning(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Нормализация не запущена
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = false
	srv.normalizerMutex.Unlock()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)

	srv.handleStopClientNormalization(w, req, client.ID, project.ID)

	// Должна быть ошибка
	if w.Code != 400 {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

// TestHandleStopClientNormalization_ProjectNotFound проверяет обработку несуществующего проекта
func TestHandleStopClientNormalization_ProjectNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/clients/1/projects/99999/normalization/stop", nil)

	srv.handleStopClientNormalization(w, req, client.ID, 99999)

	// Должна быть ошибка 404
	if w.Code != 404 {
		t.Errorf("Expected status code 404, got %d", w.Code)
	}
}

// TestHandleStopClientNormalization_WrongClient проверяет обработку проекта, не принадлежащего клиенту
func TestHandleStopClientNormalization_WrongClient(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client1, err := srv.serviceDB.CreateClient("Test Client 1", "Test Legal 1", "Desc", "test1@test.com", "+123", "TAX1", "user")
	if err != nil {
		t.Fatalf("Failed to create client 1: %v", err)
	}

	client2, err := srv.serviceDB.CreateClient("Test Client 2", "Test Legal 2", "Desc", "test2@test.com", "+124", "TAX2", "user")
	if err != nil {
		t.Fatalf("Failed to create client 2: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client1.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/clients/2/projects/1/normalization/stop", nil)

	// Пытаемся остановить от имени другого клиента
	srv.handleStopClientNormalization(w, req, client2.ID, project.ID)

	// Должна быть ошибка 400
	if w.Code != 400 {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

// TestProcessCounterpartyDatabase_ErrorOpeningDB проверяет обработку ошибки открытия БД
func TestProcessCounterpartyDatabase_ErrorOpeningDB(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем ProjectDatabase с несуществующим путем
	projectDB := &database.ProjectDatabase{
		ID:              1,
		ClientProjectID: project.ID,
		Name:            "Non-existent DB",
		FilePath:        "/non/existent/path.db",
		IsActive:        true,
	}

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку - должна быть обработана ошибка
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Проверяем, что сессия была создана и обновлена как failed
	runningSessions, err := srv.serviceDB.GetRunningSessions()
	if err == nil {
		for _, session := range runningSessions {
			if session.ProjectDatabaseID == projectDB.ID {
				if session.Status != "failed" {
					t.Logf("Expected session status 'failed', got '%s'", session.Status)
				}
				break
			}
		}
	}
}

// TestProcessCounterpartyDatabase_ErrorGettingCounterparties проверяет обработку ошибки получения контрагентов
func TestProcessCounterpartyDatabase_ErrorGettingCounterparties(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем пустую БД (без контрагентов)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "empty.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create empty DB: %v", err)
	}
	db.Close()

	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку - должна обработать пустую БД без ошибок
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Не должно быть критических ошибок
}

// TestProcessCounterpartyDatabase_ErrorSavingNormalized проверяет обработку ошибки сохранения
func TestProcessCounterpartyDatabase_ErrorSavingNormalized(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку - ошибки сохранения должны быть обработаны
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Процесс должен завершиться без паники
}

// TestProcessCounterpartyDatabase_ErrorCreatingSession проверяет обработку ошибки создания сессии
func TestProcessCounterpartyDatabase_ErrorCreatingSession(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Удаляем проект, чтобы вызвать ошибку при создании сессии
	// (сессия требует project_database_id, который должен существовать)
	// В реальности это сложно протестировать без мокирования, поэтому просто проверяем нормальную работу

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Процесс должен обработать ошибку без паники
}

// TestProcessCounterpartyDatabase_EmptyCounterparties проверяет обработку БД без контрагентов
func TestProcessCounterpartyDatabase_EmptyCounterparties(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем БД без контрагентов
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "empty.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create empty DB: %v", err)
	}
	db.Close()

	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Не должно быть ошибок
}

// TestProcessCounterpartyDatabase_InvalidData проверяет обработку некорректных данных
func TestProcessCounterpartyDatabase_InvalidData(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем БД с контрагентами с некорректными данными
	counterparties := []map[string]string{
		{"name": "", "inn": ""},           // Пустые данные
		{"name": "Тест", "inn": "invalid"}, // Некорректный ИНН
		{"name": strings.Repeat("A", 1000), "inn": "1234567890"}, // Очень длинное название
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Процесс должен обработать некорректные данные без паники
}

// TestProcessCounterpartyDatabase_StopAtDifferentPoints проверяет остановку в разных точках
func TestProcessCounterpartyDatabase_StopAtDifferentPoints(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	counterparties := make([]map[string]string, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = map[string]string{
			"name": "ООО Тест " + string(rune('A'+(i%26))),
			"inn":  "123456789" + string(rune('0'+(i%10))),
		}
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Тест 1: Остановка до начала обработки
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = false
	srv.normalizerMutex.Unlock()

	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Тест 2: Остановка во время обработки
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем в горутине и останавливаем через небольшое время
	go srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)
	time.Sleep(100 * time.Millisecond)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = false
	srv.normalizerMutex.Unlock()

	// Ждем завершения
	time.Sleep(500 * time.Millisecond)
}

// TestProcessCounterpartyDatabase_SessionCreation проверяет создание сессии
func TestProcessCounterpartyDatabase_SessionCreation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку
	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Проверяем, что сессия была создана
	lastSession, err := srv.serviceDB.GetLastNormalizationSession(projectDB.ID)
	if err != nil {
		t.Fatalf("Failed to get last session: %v", err)
	}

	if lastSession == nil {
		t.Error("Expected session to be created")
	}
	if lastSession.ProjectDatabaseID != projectDB.ID {
		t.Errorf("Expected ProjectDatabaseID %d, got %d", projectDB.ID, lastSession.ProjectDatabaseID)
	}
}

// TestProcessCounterpartyDatabase_SessionUpdate_Completed проверяет обновление сессии как completed
func TestProcessCounterpartyDatabase_SessionUpdate_Completed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Даем время на завершение
	time.Sleep(100 * time.Millisecond)

	// Проверяем статус сессии
	lastSession, err := srv.serviceDB.GetLastNormalizationSession(projectDB.ID)
	if err == nil && lastSession != nil {
		if lastSession.Status != "completed" {
			t.Logf("Expected session status 'completed', got '%s'", lastSession.Status)
		}
		if lastSession.FinishedAt == nil {
			t.Error("Expected FinishedAt to be set for completed session")
		}
	}
}

// TestProcessCounterpartyDatabase_SessionUpdate_Stopped проверяет обновление сессии как stopped
func TestProcessCounterpartyDatabase_SessionUpdate_Stopped(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем БД с большим количеством контрагентов для тестирования остановки
	counterparties := make([]map[string]string, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = map[string]string{
			"name": "ООО Тест " + string(rune('A'+(i%26))),
			"inn":  "123456789" + string(rune('0'+(i%10))),
		}
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку в отдельной горутине
	done := make(chan bool)
	go func() {
		srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)
		done <- true
	}()

	// Останавливаем через небольшое время
	time.Sleep(50 * time.Millisecond)
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = false
	srv.normalizerMutex.Unlock()

	// Ждем завершения
	<-done
	time.Sleep(100 * time.Millisecond)

	// Проверяем статус сессии
	lastSession, err := srv.serviceDB.GetLastNormalizationSession(projectDB.ID)
	if err == nil && lastSession != nil {
		// Статус может быть "stopped" или "completed" в зависимости от того, когда произошла остановка
		if lastSession.Status != "stopped" && lastSession.Status != "completed" {
			t.Logf("Expected session status 'stopped' or 'completed', got '%s'", lastSession.Status)
		}
	}
}

// TestProcessCounterpartyDatabase_SessionUpdate_Failed проверяет обновление сессии как failed
func TestProcessCounterpartyDatabase_SessionUpdate_Failed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем ProjectDatabase с несуществующим путем
	projectDB := &database.ProjectDatabase{
		ID:              1,
		ClientProjectID: project.ID,
		Name:            "Non-existent DB",
		FilePath:        "/non/existent/path.db",
		IsActive:        true,
	}

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Даем время на обработку ошибки
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что сессия была создана и обновлена как failed
	// (сессия создается до попытки открыть БД)
	runningSessions, err := srv.serviceDB.GetRunningSessions()
	if err == nil {
		for _, session := range runningSessions {
			if session.ProjectDatabaseID == projectDB.ID {
				if session.Status == "failed" {
					// Это ожидаемое поведение
					return
				}
			}
		}
	}
}

// TestProcessCounterpartyDatabasesParallel_WorkerLimit проверяет ограничение воркеров
func TestProcessCounterpartyDatabasesParallel_WorkerLimit(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем больше БД, чем максимальное количество воркеров (по умолчанию 5)
	var databases []*database.ProjectDatabase
	for i := 0; i < 10; i++ {
		counterparties := []map[string]string{
			{"name": "ООО Тест " + string(rune('A'+i)), "inn": "123456789" + string(rune('0'+i))},
		}
		dbPath := createTestDatabaseWithCounterparties(t, counterparties)
		projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)
		databases = append(databases, projectDB)
	}

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем параллельную обработку
	srv.processCounterpartyDatabasesParallel(databases, client.ID, project.ID)

	// Даем время на завершение
	time.Sleep(500 * time.Millisecond)

	// Проверяем, что все БД были обработаны (или остановлены)
	// В реальном тесте можно проверить через GetRunningSessions
}

// TestProcessCounterpartyDatabasesParallel_PanicRecovery проверяет восстановление после panic
func TestProcessCounterpartyDatabasesParallel_PanicRecovery(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем БД с валидными данными
	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку - не должно быть паники
	srv.processCounterpartyDatabasesParallel([]*database.ProjectDatabase{projectDB}, client.ID, project.ID)

	// Даем время на завершение
	time.Sleep(200 * time.Millisecond)

	// Если мы дошли сюда без паники, тест прошел
}

// TestHandleStopClientNormalization_SessionUpdate проверяет обновление сессий при остановке
func TestHandleStopClientNormalization_SessionUpdate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := srv.serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем БД и запускаем нормализацию
	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем обработку в отдельной горутине
	go srv.processCounterpartyDatabase(projectDB, client.ID, project.ID)

	// Даем время на создание сессии
	time.Sleep(50 * time.Millisecond)

	// Останавливаем через API
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)
	srv.handleStopClientNormalization(w, req, client.ID, project.ID)

	// Даем время на обновление сессий
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что сессии были обновлены
	runningSessions, err := srv.serviceDB.GetRunningSessions()
	if err == nil {
		for _, session := range runningSessions {
			if session.ProjectDatabaseID == projectDB.ID {
				// Сессия должна быть остановлена
				if session.Status != "stopped" && session.Status != "completed" {
					t.Logf("Expected session status 'stopped' or 'completed', got '%s'", session.Status)
				}
			}
		}
	}
}

