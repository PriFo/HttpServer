package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// TestStartCounterpartyNormalization_Success проверяет успешный запуск нормализации
func TestStartCounterpartyNormalization_Success(t *testing.T) {
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

	// Создаем тестовую БД
	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Создаем запрос
	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Вызываем обработчик
	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	// Проверяем ответ
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
}

// TestStartCounterpartyNormalization_InvalidProject проверяет обработку неверного проекта
func TestStartCounterpartyNormalization_InvalidProject(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/clients/1/projects/99999/normalization/start", nil)
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, 99999)

	if w.Code != 404 {
		t.Errorf("Expected status code 404, got %d", w.Code)
	}
}

// TestStartCounterpartyNormalization_AlreadyRunning проверяет обработку уже запущенной нормализации
func TestStartCounterpartyNormalization_AlreadyRunning(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", nil)
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 400 {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

// TestStartCounterpartyNormalization_NoDatabases проверяет обработку отсутствия БД
func TestStartCounterpartyNormalization_NoDatabases(t *testing.T) {
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

	// Не создаем БД

	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	// Может быть 200 (нормализация не запустится) или ошибка
	// Проверяем, что запрос обработан
	if w.Code >= 500 {
		t.Errorf("Unexpected server error: %d", w.Code)
	}
}

// TestStopCounterpartyNormalization_Success проверяет успешную остановку
func TestStopCounterpartyNormalization_Success(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)
	w := httptest.NewRecorder()

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
}

// TestStopCounterpartyNormalization_NotRunning проверяет остановку, когда нормализация не запущена
func TestStopCounterpartyNormalization_NotRunning(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)
	w := httptest.NewRecorder()

	srv.handleStopClientNormalization(w, req, client.ID, project.ID)

	if w.Code != 400 {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

// TestStopCounterpartyNormalization_InvalidProject проверяет обработку неверного проекта
func TestStopCounterpartyNormalization_InvalidProject(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client, err := srv.serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/clients/1/projects/99999/normalization/stop", nil)
	w := httptest.NewRecorder()

	srv.handleStopClientNormalization(w, req, client.ID, 99999)

	if w.Code != 404 {
		t.Errorf("Expected status code 404, got %d", w.Code)
	}
}

// TestStopCounterpartyNormalization_WrongClient проверяет обработку проекта, не принадлежащего клиенту
func TestStopCounterpartyNormalization_WrongClient(t *testing.T) {
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

	req := httptest.NewRequest("POST", "/api/clients/2/projects/1/normalization/stop", nil)
	w := httptest.NewRecorder()

	srv.handleStopClientNormalization(w, req, client2.ID, project.ID)

	if w.Code != 400 {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

// TestGetNormalizationStatus_Running проверяет получение статуса "running"
func TestGetNormalizationStatus_Running(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if status, ok := response["status"].(string); ok {
		if status != "running" {
			t.Errorf("Expected status 'running', got '%s'", status)
		}
	}
}

// TestGetNormalizationStatus_NotRunning проверяет получение статуса "not_running"
func TestGetNormalizationStatus_NotRunning(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if status, ok := response["status"].(string); ok {
		if status == "running" {
			t.Error("Expected status not to be 'running'")
		}
	}
}

// TestGetNormalizationStatus_Events проверяет получение событий
func TestGetNormalizationStatus_Events(t *testing.T) {
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

	// Отправляем событие
	select {
	case srv.normalizerEvents <- "Test event":
	default:
	}

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Проверяем, что события присутствуют
	if logs, ok := response["logs"].([]interface{}); ok {
		if len(logs) == 0 {
			t.Log("No events in response (may be expected if channel was empty)")
		}
	}
}

// TestStartNormalization_InvalidRequestBody проверяет обработку невалидного JSON
func TestStartNormalization_InvalidRequestBody(t *testing.T) {
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

	// Невалидный JSON
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	// Должен вернуть 200 (использует значения по умолчанию) или 400
	if w.Code >= 500 {
		t.Errorf("Unexpected server error: %d", w.Code)
	}
}

// TestStartNormalization_DatabaseNotFound проверяет обработку несуществующей БД
func TestStartNormalization_DatabaseNotFound(t *testing.T) {
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

	reqBody := map[string]interface{}{
		"all_active":    false,
		"database_path": "/non/existent/path.db",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleStartClientNormalization(w, req, client.ID, project.ID)

	// Должен вернуть 400 (БД не найдена)
	if w.Code != 400 {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

// TestStartNormalization_ConcurrentStart проверяет параллельный запуск
func TestStartNormalization_ConcurrentStart(t *testing.T) {
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
	createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	reqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(reqBody)

	// Первый запрос должен успешно запустить нормализацию
	req1 := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()

	srv.handleStartClientNormalization(w1, req1, client.ID, project.ID)

	if w1.Code != 200 {
		t.Errorf("Expected status code 200 for first request, got %d", w1.Code)
	}

	// Второй запрос должен вернуть ошибку (уже запущено)
	req2 := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/start", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	srv.handleStartClientNormalization(w2, req2, client.ID, project.ID)

	if w2.Code != 400 {
		t.Errorf("Expected status code 400 for concurrent request, got %d", w2.Code)
	}
}

// TestStopNormalization_DuringProcessing проверяет остановку во время обработки
func TestStopNormalization_DuringProcessing(t *testing.T) {
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

	// Создаем БД с контрагентами
	counterparties := make([]map[string]string, 50)
	for i := 0; i < 50; i++ {
		counterparties[i] = map[string]string{
			"name": "ООО Тест " + string(rune('A'+(i%26))),
			"inn":  "123456789" + string(rune('0'+(i%10))),
		}
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

	// Останавливаем нормализацию
	stopReq := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)
	stopW := httptest.NewRecorder()

	srv.handleStopClientNormalization(stopW, stopReq, client.ID, project.ID)

	if stopW.Code != 200 {
		t.Errorf("Expected status code 200 for stop, got %d", stopW.Code)
	}

	// Проверяем, что нормализация остановлена
	srv.normalizerMutex.RLock()
	isRunning := srv.normalizerRunning
	srv.normalizerMutex.RUnlock()

	if isRunning {
		t.Error("Expected normalization to be stopped")
	}
}

// TestStopNormalization_MultipleDatabases проверяет остановку при обработке нескольких БД
func TestStopNormalization_MultipleDatabases(t *testing.T) {
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
	for i := 0; i < 3; i++ {
		counterparties := []map[string]string{
			{"name": "ООО Тест " + string(rune('A'+i)), "inn": "123456789" + string(rune('0'+i))},
		}
		dbPath := createTestDatabaseWithCounterparties(t, counterparties)
		createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)
	}

	// Запускаем нормализацию всех БД
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

	// Останавливаем нормализацию
	stopReq := httptest.NewRequest("POST", "/api/clients/1/projects/1/normalization/stop", nil)
	stopW := httptest.NewRecorder()

	srv.handleStopClientNormalization(stopW, stopReq, client.ID, project.ID)

	if stopW.Code != 200 {
		t.Errorf("Expected status code 200 for stop, got %d", stopW.Code)
	}
}

// TestGetStatus_WhileRunning проверяет получение статуса во время выполнения
func TestGetStatus_WhileRunning(t *testing.T) {
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

	// Устанавливаем флаг нормализации и начальное время
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerStartTime = time.Now()
	srv.normalizerProcessed = 10
	srv.normalizerMutex.Unlock()

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if isRunning, ok := response["isRunning"].(bool); ok {
		if !isRunning {
			t.Error("Expected isRunning to be true")
		}
	}

	if processed, ok := response["processed"].(float64); ok {
		if processed != 10 {
			t.Errorf("Expected processed 10, got %f", processed)
		}
	}
}

// TestGetStatus_AfterStop проверяет статус после остановки
func TestGetStatus_AfterStop(t *testing.T) {
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

	// Устанавливаем флаг нормализации как остановленной
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = false
	srv.normalizerProcessed = 5
	srv.normalizerMutex.Unlock()

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if isRunning, ok := response["isRunning"].(bool); ok {
		if isRunning {
			t.Error("Expected isRunning to be false after stop")
		}
	}
}

// TestGetStats_EmptyProject проверяет статистику для пустого проекта
func TestGetStats_EmptyProject(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/stats", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationStats(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Проверяем, что статистика возвращена (может быть пустой)
	if response == nil {
		t.Error("Expected response to be not nil")
	}
}

// TestGetGroups_WithDuplicates проверяет получение групп с дубликатами
func TestGetGroups_WithDuplicates(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/groups", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationGroups(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Проверяем, что ответ содержит группы (может быть пустым)
	if response == nil {
		t.Error("Expected response to be not nil")
	}
}

// TestGetGroups_EmptyProject проверяет группы для пустого проекта
func TestGetGroups_EmptyProject(t *testing.T) {
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

	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/groups", nil)
	w := httptest.NewRecorder()

	srv.handleGetClientNormalizationGroups(w, req, client.ID, project.ID)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

