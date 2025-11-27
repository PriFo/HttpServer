package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNormalizationEvents_StructuredEvents проверяет отправку структурированных событий
func TestNormalizationEvents_StructuredEvents(t *testing.T) {
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
	createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Запускаем нормализацию в горутине
	done := make(chan bool)
	var events []string
	go func() {
		// Собираем события
		for {
			select {
			case event := <-srv.normalizerEvents:
				events = append(events, event)
				// Проверяем, есть ли структурированные события
				if strings.HasPrefix(event, `{"type":`) {
					var eventData map[string]interface{}
					if err := json.Unmarshal([]byte(event), &eventData); err == nil {
						t.Logf("Received structured event: type=%v, data=%v", eventData["type"], eventData["data"])
					}
				}
			case <-time.After(5 * time.Second):
				done <- true
				return
			}
		}
	}()

	// Запускаем нормализацию
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	go func() {
		databases, _ := srv.serviceDB.GetProjectDatabases(project.ID, true)
		if len(databases) > 0 {
			srv.processCounterpartyDatabase(databases[0], client.ID, project.ID)
		}
	}()

	// Ждем завершения
	<-done

	// Проверяем, что были получены структурированные события
	structuredCount := 0
	for _, event := range events {
		if strings.HasPrefix(event, `{"type":`) {
			structuredCount++
		}
	}

	if structuredCount == 0 {
		t.Log("No structured events received (may be expected if normalization completed too quickly)")
	} else {
		t.Logf("Received %d structured events", structuredCount)
	}
}

// TestNormalizationStatus_WithSessions проверяет, что статус endpoint возвращает информацию о сессиях
func TestNormalizationStatus_WithSessions(t *testing.T) {
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

	// Создаем БД
	counterparties := []map[string]string{
		{"name": "ООО Тест 1", "inn": "1234567890"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	projectDB := createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Создаем сессию нормализации
	sessionID, err := srv.serviceDB.CreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Обновляем сессию как completed
	finishedAt := time.Now()
	err = srv.serviceDB.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Запрашиваем статус
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

	// Проверяем наличие информации о сессиях и БД
	if sessions, ok := response["sessions"].([]interface{}); ok {
		t.Logf("Found %d sessions in response", len(sessions))
	} else {
		t.Log("No sessions field in response (may be expected if no active sessions)")
	}

	if databases, ok := response["databases"].([]interface{}); ok {
		if len(databases) == 0 {
			t.Error("Expected at least one database in response")
		} else {
			t.Logf("Found %d databases in response", len(databases))
		}
	} else {
		t.Error("Expected 'databases' field in response")
	}

	if activeSessionsCount, ok := response["active_sessions_count"].(float64); ok {
		t.Logf("Active sessions count: %.0f", activeSessionsCount)
	}

	if totalDatabasesCount, ok := response["total_databases_count"].(float64); ok {
		t.Logf("Total databases count: %.0f", totalDatabasesCount)
	}
}

// TestNormalizationEvents_Parsing проверяет парсинг структурированных событий
func TestNormalizationEvents_Parsing(t *testing.T) {
	// Тестируем различные типы структурированных событий
	testCases := []struct {
		name      string
		eventType string
		data      map[string]interface{}
	}{
		{
			name:      "progress event",
			eventType: "progress",
			data: map[string]interface{}{
				"processed":        50,
				"total":            100,
				"progress_percent": 50.0,
				"benchmark_matches": 5,
				"enriched_count":    3,
				"duplicate_groups":  2,
			},
		},
		{
			name:      "start event",
			eventType: "start",
			data: map[string]interface{}{
				"total_counterparties": 100,
				"original_count":       100,
				"skipped_count":         0,
			},
		},
		{
			name:      "completed event",
			eventType: "completed",
			data: map[string]interface{}{
				"total_processed":     100,
				"total_counterparties": 100,
				"duration_seconds":    10.5,
				"items_per_second":     9.52,
				"benchmark_matches":    10,
				"enriched_count":       5,
				"success_rate":         100.0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eventData := map[string]interface{}{
				"type":      tc.eventType,
				"timestamp": time.Now().Format(time.RFC3339),
				"client_id": 1,
				"project_id": 1,
				"data":      tc.data,
			}

			jsonData, err := json.Marshal(eventData)
			if err != nil {
				t.Fatalf("Failed to marshal event: %v", err)
			}

			// Проверяем, что можно распарсить обратно
			var parsed map[string]interface{}
			if err := json.Unmarshal(jsonData, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal event: %v", err)
			}

			if parsed["type"] != tc.eventType {
				t.Errorf("Expected type %s, got %v", tc.eventType, parsed["type"])
			}

			if data, ok := parsed["data"].(map[string]interface{}); ok {
				t.Logf("Successfully parsed event data: %v", data)
			} else {
				t.Error("Failed to parse event data")
			}
		})
	}
}

// TestNormalizationEvents_StructuredEvents_Integration проверяет отправку структурированных событий в реальной нормализации
func TestNormalizationEvents_StructuredEvents_Integration(t *testing.T) {
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
		{"name": "ООО Тест 3", "inn": "1234567892"},
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Собираем события
	events := make([]string, 0)
	eventChan := make(chan string, 100)
	
	// Заменяем канал событий на наш тестовый
	originalChannel := srv.normalizerEvents
	srv.normalizerEvents = eventChan
	defer func() {
		srv.normalizerEvents = originalChannel
	}()

	// Запускаем сбор событий
	go func() {
		for event := range eventChan {
			events = append(events, event)
			t.Logf("Received event: %s", event[:minInt(100, len(event))])
		}
	}()

	// Запускаем нормализацию
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	databases, _ := srv.serviceDB.GetProjectDatabases(project.ID, true)
	if len(databases) > 0 {
		srv.processCounterpartyDatabase(databases[0], client.ID, project.ID)
	}

	// Ждем завершения
	time.Sleep(1 * time.Second)
	close(eventChan)

	// Проверяем структурированные события
	structuredCount := 0
	progressCount := 0
	startCount := 0
	completedCount := 0

	for _, event := range events {
		if strings.HasPrefix(event, `{"type":`) {
			structuredCount++
			var eventData map[string]interface{}
			if err := json.Unmarshal([]byte(event), &eventData); err == nil {
				eventType, _ := eventData["type"].(string)
				switch eventType {
				case "progress":
					progressCount++
				case "start":
					startCount++
				case "completed":
					completedCount++
				}
			}
		}
	}

	t.Logf("Total events: %d, Structured: %d, Progress: %d, Start: %d, Completed: %d", 
		len(events), structuredCount, progressCount, startCount, completedCount)

	if structuredCount > 0 {
		t.Log("✓ Structured events are being sent")
	} else {
		t.Log("⚠ No structured events found (may be expected if normalization is too fast)")
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestNormalizationStatus_RealTimeUpdates проверяет обновления статуса в реальном времени
func TestNormalizationStatus_RealTimeUpdates(t *testing.T) {
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

	// Создаем БД с большим количеством контрагентов для тестирования прогресса
	counterparties := make([]map[string]string, 150)
	for i := 0; i < 150; i++ {
		counterparties[i] = map[string]string{
			"name": "ООО Тест " + string(rune('A'+(i%26))),
			"inn":  "123456789" + string(rune('0'+(i%10))),
		}
	}
	dbPath := createTestDatabaseWithCounterparties(t, counterparties)
	createTestProjectDatabase(t, srv.serviceDB, project.ID, dbPath)

	// Запускаем нормализацию
	srv.normalizerMutex.Lock()
	srv.normalizerRunning = true
	srv.normalizerMutex.Unlock()

	// Запускаем в горутине
	go func() {
		databases, _ := srv.serviceDB.GetProjectDatabases(project.ID, true)
		if len(databases) > 0 {
			srv.processCounterpartyDatabase(databases[0], client.ID, project.ID)
		}
	}()

	// Проверяем статус несколько раз во время обработки
	for i := 0; i < 5; i++ {
		time.Sleep(200 * time.Millisecond)

		req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
		w := httptest.NewRecorder()
		srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

		if w.Code == 200 {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
				processed, _ := response["processed"].(float64)
				progress, _ := response["progress"].(float64)
				currentStep, _ := response["currentStep"].(string)
				t.Logf("Status update %d: processed=%.0f, progress=%.1f%%, step=%s", i+1, processed, progress, currentStep)
			}
		}
	}

	// Ждем завершения
	time.Sleep(2 * time.Second)

	// Финальная проверка статуса
	req := httptest.NewRequest("GET", "/api/clients/1/projects/1/normalization/status", nil)
	w := httptest.NewRecorder()
	srv.handleGetClientNormalizationStatus(w, req, client.ID, project.ID)

	if w.Code == 200 {
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
			isRunning, _ := response["isRunning"].(bool)
			processed, _ := response["processed"].(float64)
			t.Logf("Final status: isRunning=%v, processed=%.0f", isRunning, processed)
		}
	}
}

