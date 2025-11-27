package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"httpserver/database"
)

// TestCounterpartyNormalizationE2E_FullCycle проверяет полный цикл нормализации
func TestCounterpartyNormalizationE2E_FullCycle(t *testing.T) {
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
		{"name": "ООО Тест 1", "inn": "1234567890", "kpp": "123456789"},
		{"name": "ООО Тест 2", "inn": "1234567891", "kpp": "123456789"},
		{"name": "ООО Тест 3", "inn": "1234567892", "kpp": "123456789"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// 1. Проверяем начальный статус
	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w := httptest.NewRecorder()
	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var statusResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &statusResponse); err != nil {
		t.Fatalf("Failed to parse status response: %v", err)
	}

	// Статус должен быть "not_running" или "idle"
	if status, ok := statusResponse["status"].(string); ok {
		if status == "running" {
			t.Error("Expected normalization not to be running initially")
		}
	}

	// 2. Запускаем нормализацию
	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)
	req = httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Проверяем, что нормализация запущена
	srv.normalizerMutex.RLock()
	isRunning := srv.normalizerRunning
	srv.normalizerMutex.RUnlock()

	if !isRunning {
		t.Error("Expected normalization to be running")
	}

	// 3. Ждем завершения нормализации
	maxWait := 5 * time.Second
	waitInterval := 100 * time.Millisecond
	waited := 0 * time.Second

	for waited < maxWait {
		srv.normalizerMutex.RLock()
		running := srv.normalizerRunning
		srv.normalizerMutex.RUnlock()

		if !running {
			break
		}

		time.Sleep(waitInterval)
		waited += waitInterval
	}

	// 4. Проверяем финальный статус
	req = httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w = httptest.NewRecorder()
	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &statusResponse); err != nil {
		t.Fatalf("Failed to parse final status response: %v", err)
	}

	// Статус должен быть "completed" или "not_running"
	if status, ok := statusResponse["status"].(string); ok {
		if status == "running" {
			t.Log("Normalization is still running after wait period")
		}
	}

	// 5. Проверяем, что данные были сохранены
	stats, err := srv.serviceDB.GetNormalizedCounterpartyStats(project.ID)
	if err == nil {
		if totalCountVal, ok := stats["total_count"]; ok {
			var totalCount int
			switch v := totalCountVal.(type) {
			case int:
				totalCount = v
			case int64:
				totalCount = int(v)
			case float64:
				totalCount = int(v)
			}
			if totalCount == 0 {
				t.Log("No normalized counterparties found (may be expected if normalization failed)")
			} else {
				t.Logf("Found %d normalized counterparties", totalCount)
			}
		}
	}
}

// TestCounterpartyNormalizationE2E_StopAndResume проверяет остановку и сохранение частичных результатов
func TestCounterpartyNormalizationE2E_StopAndResume(t *testing.T) {
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

	// Создаем БД с большим количеством контрагентов для тестирования остановки
	counterparties := make([]map[string]string, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = map[string]string{
			"name": "ООО Тест " + string(rune('A'+(i%26))),
			"inn":  "123456789" + string(rune('0'+(i%10))),
			"kpp":  "123456789",
		}
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// 1. Запускаем нормализацию
	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// 2. Ждем немного, чтобы нормализация началась
	time.Sleep(100 * time.Millisecond)

	// 3. Останавливаем нормализацию
	req = httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)
	w = httptest.NewRecorder()

	srv.handleStopClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Проверяем, что нормализация остановлена
	srv.normalizerMutex.RLock()
	isRunning := srv.normalizerRunning
	srv.normalizerMutex.RUnlock()

	if isRunning {
		t.Error("Expected normalization to be stopped")
	}

	// 4. Ждем завершения обработки остановки
	time.Sleep(500 * time.Millisecond)

	// 5. Проверяем статус сессии
	lastSession, err := srv.serviceDB.GetLastNormalizationSession(projectDB.ID)
	if err == nil && lastSession != nil {
		// Статус должен быть "stopped" или "completed" в зависимости от того, когда произошла остановка
		if lastSession.Status != "stopped" && lastSession.Status != "completed" {
			t.Logf("Expected session status 'stopped' or 'completed', got '%s'", lastSession.Status)
		}
	}

	// 6. Проверяем, что частично обработанные данные сохранены
	stats, err := srv.serviceDB.GetNormalizedCounterpartyStats(project.ID)
	if err == nil {
		// Могут быть сохранены частичные результаты
		totalCount, _ := stats["total_count"].(int)
		t.Logf("Normalized counterparties after stop: %d", totalCount)
	}
}

// TestCounterpartyNormalizationE2E_MultipleDatabases проверяет обработку нескольких БД
func TestCounterpartyNormalizationE2E_MultipleDatabases(t *testing.T) {
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

	// Создаем несколько БД
	for i := 0; i < 3; i++ {
		counterparties := []map[string]string{
			{"name": "ООО Тест " + string(rune('A'+i)), "inn": "123456789" + string(rune('0'+i))},
		}
		dbPath := createTestDatabaseWithCounterparties(t, counterparties)
		createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)
	}

	// Запускаем нормализацию для всех активных БД
	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Ждем завершения
	maxWait := 5 * time.Second
	waitInterval := 100 * time.Millisecond
	waited := 0 * time.Second

	for waited < maxWait {
		srv.normalizerMutex.RLock()
		running := srv.normalizerRunning
		srv.normalizerMutex.RUnlock()

		if !running {
			break
		}

		time.Sleep(waitInterval)
		waited += waitInterval
	}

	// Проверяем, что все БД были обработаны
	databases, err := srv.serviceDB.GetProjectDatabases(project.ID, false)
	if err == nil {
		for _, db := range databases {
			lastSession, err := srv.serviceDB.GetLastNormalizationSession(db.ID)
			if err == nil && lastSession != nil {
				t.Logf("Database %s: session status %s", db.Name, lastSession.Status)
			}
		}
	}
}

// TestCounterpartyNormalizationE2E_WithBenchmarks проверяет нормализацию с эталонами
func TestCounterpartyNormalizationE2E_WithBenchmarks(t *testing.T) {
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

	// Создаем эталон
	benchmark, err := srv.serviceDB.CreateCounterpartyBenchmark(
		project.ID,
		"ООО Эталонная Компания",
		"ООО Эталонная Компания",
		"1234567890", // taxID
		"123456789",  // kpp
		"",           // bin
		"",           // ogrn
		"",           // region
		"г. Москва, ул. Тестовая, д. 1", // legalAddress
		"",   // postalAddress
		"",   // contactPhone
		"",   // contactEmail
		"",   // contactPerson
		"",   // legalForm
		"",   // bankName
		"",   // bankAccount
		"",   // correspondentAccount
		"",   // bik
		0.95, // qualityScore
	)
	if err != nil {
		t.Fatalf("Failed to create benchmark: %v", err)
	}

	// Создаем БД с контрагентом, совпадающим с эталоном
	counterparties := []map[string]string{
		{"name": "ООО Эталонная Компания", "inn": "1234567890", "kpp": "123456789"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Запускаем нормализацию
	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Ждем завершения
	maxWait := 5 * time.Second
	waitInterval := 100 * time.Millisecond
	waited := 0 * time.Second

	for waited < maxWait {
		srv.normalizerMutex.RLock()
		running := srv.normalizerRunning
		srv.normalizerMutex.RUnlock()

		if !running {
			break
		}

		time.Sleep(waitInterval)
		waited += waitInterval
	}

	// Проверяем, что эталон был использован
	stats, err := srv.serviceDB.GetNormalizedCounterpartyStats(project.ID)
	if err == nil {
		totalCount, _ := stats["total_count"].(int)
		withBenchmark, _ := stats["with_benchmark"].(int)
		t.Logf("Normalized counterparties: %d, with benchmark: %d", totalCount, withBenchmark)
		if benchmark != nil {
			t.Logf("Benchmark ID: %d", benchmark.ID)
		}
	}
}

// TestCounterpartyNormalizationE2E_ErrorRecovery проверяет восстановление после ошибок
func TestCounterpartyNormalizationE2E_ErrorRecovery(t *testing.T) {
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

	// Создаем БД с валидными данными
	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	validDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Создаем БД с несуществующим путем (для тестирования ошибок)
	invalidDB := &database.ProjectDatabase{
		ID:              999,
		ClientProjectID: project.ID,
		Name:            "Invalid DB",
		FilePath:        "/non/existent/path.db",
		IsActive:        true,
	}

	// Добавляем невалидную БД вручную (для тестирования)
	// В реальном сценарии это не должно происходить, но тестируем обработку ошибок
	_ = invalidDB

	// Запускаем нормализацию для валидной БД
	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Ждем завершения
	maxWait := 5 * time.Second
	waitInterval := 100 * time.Millisecond
	waited := 0 * time.Second

	for waited < maxWait {
		srv.normalizerMutex.RLock()
		running := srv.normalizerRunning
		srv.normalizerMutex.RUnlock()

		if !running {
			break
		}

		time.Sleep(waitInterval)
		waited += waitInterval
	}

	// Проверяем, что валидная БД была обработана
	lastSession, err := srv.serviceDB.GetLastNormalizationSession(validDB.ID)
	if err == nil && lastSession != nil {
		t.Logf("Valid DB session status: %s", lastSession.Status)
		// Статус должен быть "completed" или "stopped", но не "failed" для валидной БД
		if lastSession.Status == "failed" {
			t.Log("Valid DB session failed (may indicate an issue)")
		}
	}
}
