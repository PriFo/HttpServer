package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/server"
	
	"github.com/google/uuid"
)

// TestCounterpartyNormalization_E2E_FullCycle проверяет полный E2E цикл нормализации с остановкой
func TestCounterpartyNormalization_E2E_FullCycle(t *testing.T) {
	// Создаем тестовые БД
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close()

	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Создаем сервер
	config := &server.Config{
		Port:                    "8080",
		DatabasePath:            ":memory:",
		NormalizedDatabasePath:  ":memory:",
		ServiceDatabasePath:     ":memory:",
		LogBufferSize:           100,
		NormalizerEventsBufferSize: 100,
	}

	srv := server.NewServerWithConfig(db, db, serviceDB, ":memory:", ":memory:", config)

	// Создаем тестового клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем тестовую БД проекта
	projectDB, err := serviceDB.CreateProjectDatabase(project.ID, "test_db", ":memory:", "Test database", 0)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем тестовых контрагентов в БД
	testDB, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

	uploadUUID := uuid.New().String()
	upload, err := testDB.CreateUpload(uploadUUID, "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	catalog, err := testDB.AddCatalog(upload.ID, "TestCatalog", "test_catalog")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем контрагентов (достаточно для длительной обработки)
	for i := 0; i < 50; i++ {
		err = testDB.AddCatalogItem(catalog.ID, fmt.Sprintf("ref%d", i), fmt.Sprintf("code%d", i), 
			fmt.Sprintf("ООО Тест %d", i), fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10), "")
		if err != nil {
			t.Fatalf("Failed to add catalog item: %v", err)
		}
	}

	// Шаг 1: Запускаем нормализацию через API
	startReqBody := map[string]interface{}{
		"all_active": false,
		"database_path": projectDB.FilePath,
	}
	body, _ := json.Marshal(startReqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/clients/%d/projects/%d/normalization/start", client.ID, project.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var startResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &startResponse); err != nil {
		t.Fatalf("Failed to parse start response: %v", err)
	}

	// API возвращает "success": true вместо "status": "started"
	if success, ok := startResponse["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", startResponse)
	}

	// Шаг 2: Ждем немного, чтобы нормализация началась
	time.Sleep(200 * time.Millisecond)

	// Шаг 3: Останавливаем нормализацию через API (используем глобальный endpoint)
	stopReq := httptest.NewRequest("POST", "/api/normalization/stop", nil)
	stopW := httptest.NewRecorder()

	srv.ServeHTTP(stopW, stopReq)

	if stopW.Code != http.StatusOK {
		t.Errorf("Expected status 200 for stop, got %d. Body: %s", stopW.Code, stopW.Body.String())
	}

	var stopResponse map[string]interface{}
	if err := json.Unmarshal(stopW.Body.Bytes(), &stopResponse); err != nil {
		t.Fatalf("Failed to parse stop response: %v", err)
	}

	// API возвращает "success": true и "was_running": bool вместо "status": "stopped"
	if success, ok := stopResponse["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true in stop response, got %v", stopResponse)
	}

	// Шаг 4: Проверяем статус через API
	statusReq := httptest.NewRequest("GET", "/api/normalization/status", nil)
	statusW := httptest.NewRecorder()

	srv.ServeHTTP(statusW, statusReq)

	if statusW.Code != http.StatusOK {
		t.Errorf("Expected status 200 for status, got %d", statusW.Code)
	}

	var statusResponse map[string]interface{}
	if err := json.Unmarshal(statusW.Body.Bytes(), &statusResponse); err != nil {
		t.Fatalf("Failed to parse status response: %v", err)
	}

	// Проверяем, что нормализация не запущена
	if isRunning, ok := statusResponse["is_running"].(bool); ok && isRunning {
		t.Error("Expected normalization to be stopped, but is_running is true")
	}

	// Шаг 5: Проверяем сессии нормализации в БД
	runningSessions, err := serviceDB.GetRunningSessions()
	if err != nil {
		t.Fatalf("Failed to get running sessions: %v", err)
	}

	// Не должно быть запущенных сессий
	if len(runningSessions) > 0 {
		t.Errorf("Expected no running sessions after stop, got %d", len(runningSessions))
	}

	// Шаг 6: Проверяем, что частичные данные сохранены
	// (если были обработаны записи до остановки)
	stats, err := serviceDB.GetNormalizedCounterpartyStats(project.ID)
	if err == nil {
		// Проверяем, что есть статистика (даже если 0)
		if stats == nil {
			t.Error("Expected stats to be returned, got nil")
		}
	}

	// Ждем завершения всех горутин
	time.Sleep(300 * time.Millisecond)
}

// TestCounterpartyNormalization_E2E_API_Contract проверяет API контракт
func TestCounterpartyNormalization_E2E_API_Contract(t *testing.T) {
	// Создаем тестовые БД
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close()

	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Создаем сервер
	config := &server.Config{
		Port:                    "8080",
		DatabasePath:            ":memory:",
		NormalizedDatabasePath:  ":memory:",
		ServiceDatabasePath:     ":memory:",
		LogBufferSize:           100,
		NormalizerEventsBufferSize: 100,
	}

	srv := server.NewServerWithConfig(db, db, serviceDB, ":memory:", ":memory:", config)

	// Создаем тестового клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем тестовую БД проекта (необходимо для запуска нормализации)
	_, err = serviceDB.CreateProjectDatabase(project.ID, "test_db", ":memory:", "Test database", 0)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Тест 1: Проверка заголовков ответа
	startReqBody := map[string]interface{}{
		"all_active": true,
	}
	body, _ := json.Marshal(startReqBody)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/clients/%d/projects/%d/normalization/start", client.ID, project.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	// Проверяем заголовки
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", w.Header().Get("Content-Type"))
	}

	// Тест 2: Проверка структуры JSON ответа
	var startResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &startResponse); err != nil {
		t.Fatalf("Failed to parse start response: %v", err)
	}

	// Проверяем обязательные поля (API возвращает success, message, client_id, project_id)
	requiredFields := []string{"success", "message", "client_id", "project_id"}
	for _, field := range requiredFields {
		if _, ok := startResponse[field]; !ok {
			t.Errorf("Expected field '%s' in response", field)
		}
	}

	// Тест 3: Проверка статуса ответа
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Тест 4: Проверка остановки (используем глобальный endpoint)
	stopReq := httptest.NewRequest("POST", "/api/normalization/stop", nil)
	stopW := httptest.NewRecorder()

	srv.ServeHTTP(stopW, stopReq)

	// Проверяем заголовки
	if stopW.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json' for stop, got %s", stopW.Header().Get("Content-Type"))
	}

	// Проверяем структуру JSON
	var stopResponse map[string]interface{}
	if err := json.Unmarshal(stopW.Body.Bytes(), &stopResponse); err != nil {
		t.Fatalf("Failed to parse stop response: %v", err)
	}

	// Проверяем обязательные поля (API возвращает success, message, was_running)
	requiredStopFields := []string{"success", "message", "was_running"}
	for _, field := range requiredStopFields {
		if _, ok := stopResponse[field]; !ok {
			t.Errorf("Expected field '%s' in stop response", field)
		}
	}

	// Тест 5: Проверка статуса ответа для остановки
	if stopW.Code != http.StatusOK {
		t.Errorf("Expected status 200 for stop, got %d", stopW.Code)
	}

	// Ждем завершения
	time.Sleep(200 * time.Millisecond)
}
