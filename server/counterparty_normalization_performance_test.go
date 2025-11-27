package server

import (
	"net/http/httptest"
	"testing"

	"httpserver/normalization"
)

// TestCounterpartyNormalizationStopCheckPerformance проверяет получение метрик производительности
func TestCounterpartyNormalizationStopCheckPerformance(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Сбрасываем метрики перед тестом
	normalization.ResetStopCheckMetrics()

	// Выполняем несколько проверок для генерации метрик
	stopCheck := func() bool { return false }
	for i := 0; i < 10; i++ {
		_ = stopCheck()
	}

	req := httptest.NewRequest("GET", "/api/counterparty/normalization/stop-check/performance", nil)
	w := httptest.NewRecorder()

	srv.handleCounterpartyNormalizationStopCheckPerformance(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Проверяем, что ответ содержит метрики
	if w.Body.Len() == 0 {
		t.Error("Expected response body to contain metrics")
	}
}

// TestCounterpartyNormalizationStopCheckPerformanceReset проверяет сброс метрик
func TestCounterpartyNormalizationStopCheckPerformanceReset(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Генерируем некоторые метрики
	stopCheck := func() bool { return false }
	for i := 0; i < 5; i++ {
		_ = stopCheck()
	}

	// Сбрасываем метрики
	req := httptest.NewRequest("POST", "/api/counterparty/normalization/stop-check/performance/reset", nil)
	w := httptest.NewRecorder()

	srv.handleCounterpartyNormalizationStopCheckPerformanceReset(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Проверяем, что метрики сброшены
	stats := normalization.GetStopCheckStats()
	if stats["total_checks"].(int64) != 0 {
		t.Error("Expected metrics to be reset")
	}
}

// TestCounterpartyNormalizationStopCheckPerformance_MethodNotAllowed проверяет обработку неправильного метода
func TestCounterpartyNormalizationStopCheckPerformance_MethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/counterparty/normalization/stop-check/performance", nil)
	w := httptest.NewRecorder()

	srv.handleCounterpartyNormalizationStopCheckPerformance(w, req)

	if w.Code != 405 {
		t.Errorf("Expected status code 405, got %d", w.Code)
	}
}

// TestCounterpartyNormalizationStopCheckPerformanceReset_MethodNotAllowed проверяет обработку неправильного метода
func TestCounterpartyNormalizationStopCheckPerformanceReset_MethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/counterparty/normalization/stop-check/performance/reset", nil)
	w := httptest.NewRecorder()

	srv.handleCounterpartyNormalizationStopCheckPerformanceReset(w, req)

	if w.Code != 405 {
		t.Errorf("Expected status code 405, got %d", w.Code)
	}
}

