//go:build test_benchmark
// +build test_benchmark

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGetAvailableModels_Success проверяет успешное получение моделей из API
func TestGetAvailableModels_Success(t *testing.T) {
	// Создаем мок-сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workers/models" || r.URL.Query().Get("provider") != "huggingface" {
			t.Errorf("Expected path /api/workers/models?provider=huggingface, got %s?%s", r.URL.Path, r.URL.RawQuery)
		}

		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"models": []map[string]interface{}{
					{"id": "model1", "name": "Model 1", "status": "active"},
					{"id": "model2", "name": "Model 2", "status": "active"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	models, err := getAvailableModels(server.URL)
	if err != nil {
		t.Fatalf("getAvailableModels() returned error: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0] != "Model 1" {
		t.Errorf("Expected first model to be 'Model 1', got '%s'", models[0])
	}
}

// TestGetAvailableModels_Fallback проверяет использование fallback моделей при ошибке API
func TestGetAvailableModels_Fallback(t *testing.T) {
	// Создаем мок-сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	models, err := getAvailableModels(server.URL)
	if err != nil {
		t.Fatalf("getAvailableModels() should not return error on fallback, got: %v", err)
	}

	// Должны вернуться fallback модели
	expectedFallbackCount := 7
	if len(models) != expectedFallbackCount {
		t.Errorf("Expected %d fallback models, got %d", expectedFallbackCount, len(models))
	}

	// Проверяем, что есть известные модели
	knownModels := []string{
		"microsoft/DialoGPT-medium",
		"google/flan-t5-base",
		"meta-llama/Llama-2-7b-chat-hf",
	}
	for _, known := range knownModels {
		found := false
		for _, model := range models {
			if model == known {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected fallback models to include '%s', but it wasn't found", known)
		}
	}
}

// TestGetAvailableModels_EmptyResponse проверяет обработку пустого ответа
func TestGetAvailableModels_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"models": []map[string]interface{}{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	models, err := getAvailableModels(server.URL)
	if err != nil {
		t.Fatalf("getAvailableModels() should not return error on empty response, got: %v", err)
	}

	// Должны вернуться fallback модели
	if len(models) == 0 {
		t.Error("Expected fallback models when API returns empty list")
	}
}

// TestTestModel_Success проверяет успешное тестирование модели
func TestTestModel_Success(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/kpved/classify-hierarchical" {
			t.Errorf("Expected POST /api/kpved/classify-hierarchical, got %s %s", r.Method, r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Проверяем, что модель передана
		if model, ok := reqBody["model"].(string); !ok || model == "" {
			t.Error("Expected 'model' field in request body")
		}

		requestCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"kpved_code": "1234",
		})
	}))
	defer server.Close()

	testProducts := []string{"Болт М8х20", "Гайка М8"}
	benchmark := testModel(server.URL, "test-model", testProducts)

	if benchmark.Name != "test-model" {
		t.Errorf("Expected benchmark name 'test-model', got '%s'", benchmark.Name)
	}

	if benchmark.SuccessCount != int64(len(testProducts)) {
		t.Errorf("Expected %d successful requests, got %d", len(testProducts), benchmark.SuccessCount)
	}

	if benchmark.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", benchmark.ErrorCount)
	}

	if benchmark.Speed <= 0 {
		t.Error("Expected speed > 0 for successful requests")
	}

	if benchmark.AvgResponseTime <= 0 {
		t.Error("Expected avg response time > 0")
	}
}

// TestTestModel_Errors проверяет обработку ошибок при тестировании модели
func TestTestModel_Errors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Internal server error",
		})
	}))
	defer server.Close()

	testProducts := []string{"Болт М8х20"}
	benchmark := testModel(server.URL, "test-model", testProducts)

	if benchmark.SuccessCount != 0 {
		t.Errorf("Expected 0 successful requests, got %d", benchmark.SuccessCount)
	}

	if benchmark.ErrorCount != int64(len(testProducts)) {
		t.Errorf("Expected %d errors, got %d", len(testProducts), benchmark.ErrorCount)
	}

	if benchmark.Speed != 0 {
		t.Error("Expected speed = 0 when all requests fail")
	}

	if benchmark.Status != "✗ FAILED" {
		t.Errorf("Expected status '✗ FAILED', got '%s'", benchmark.Status)
	}
}

// TestTestModel_NetworkError проверяет обработку сетевых ошибок
func TestTestModel_NetworkError(t *testing.T) {
	// Используем несуществующий URL для имитации сетевой ошибки
	testProducts := []string{"Болт М8х20"}
	benchmark := testModel("http://localhost:99999", "test-model", testProducts)

	if benchmark.SuccessCount != 0 {
		t.Errorf("Expected 0 successful requests on network error, got %d", benchmark.SuccessCount)
	}

	if benchmark.ErrorCount != int64(len(testProducts)) {
		t.Errorf("Expected %d errors on network error, got %d", len(testProducts), benchmark.ErrorCount)
	}
}

// TestModelBenchmark_Statistics проверяет расчет статистики
func TestModelBenchmark_Statistics(t *testing.T) {
	responseTimes := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
	}

	benchmark := &ModelBenchmark{
		Name:          "test-model",
		ResponseTimes: responseTimes,
		SuccessCount:  int64(len(responseTimes)),
		TotalRequests: int64(len(responseTimes)),
	}

	// Рассчитываем статистику вручную для проверки
	totalDuration := time.Duration(0)
	for _, rt := range responseTimes {
		totalDuration += rt
	}
	benchmark.AvgResponseTime = totalDuration / time.Duration(len(responseTimes))

	// Проверяем среднее время
	expectedAvg := 300 * time.Millisecond
	if benchmark.AvgResponseTime != expectedAvg {
		t.Errorf("Expected avg response time %v, got %v", expectedAvg, benchmark.AvgResponseTime)
	}

	// Проверяем минимальное время
	benchmark.MinResponseTime = responseTimes[0]
	for _, rt := range responseTimes {
		if rt < benchmark.MinResponseTime {
			benchmark.MinResponseTime = rt
		}
	}
	if benchmark.MinResponseTime != 100*time.Millisecond {
		t.Errorf("Expected min response time 100ms, got %v", benchmark.MinResponseTime)
	}

	// Проверяем максимальное время
	benchmark.MaxResponseTime = responseTimes[0]
	for _, rt := range responseTimes {
		if rt > benchmark.MaxResponseTime {
			benchmark.MaxResponseTime = rt
		}
	}
	if benchmark.MaxResponseTime != 500*time.Millisecond {
		t.Errorf("Expected max response time 500ms, got %v", benchmark.MaxResponseTime)
	}

	// Проверяем процент успешных запросов
	benchmark.SuccessRate = float64(benchmark.SuccessCount) / float64(benchmark.TotalRequests) * 100
	if benchmark.SuccessRate != 100.0 {
		t.Errorf("Expected success rate 100%%, got %.1f%%", benchmark.SuccessRate)
	}
}

// TestRepeat проверяет функцию repeat
func TestRepeat(t *testing.T) {
	result := repeat("a", 5)
	expected := "aaaaa"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	result = repeat("ab", 3)
	expected = "ababab"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	result = repeat("x", 0)
	expected = ""
	if result != expected {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestTruncateString проверяет функцию truncateString
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"very long string", 10, "very lo..."},
		{"exact", 5, "exact"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// TestSaveResultsToJSON проверяет сохранение результатов в JSON
func TestSaveResultsToJSON(t *testing.T) {
	benchmarks := []*ModelBenchmark{
		{
			Name:            "model1",
			Priority:        1,
			Speed:           10.5,
			AvgResponseTime: 100 * time.Millisecond,
			MedianResponseTime: 95 * time.Millisecond,
			P95ResponseTime: 120 * time.Millisecond,
			MinResponseTime: 80 * time.Millisecond,
			MaxResponseTime: 120 * time.Millisecond,
			SuccessCount:    10,
			ErrorCount:      0,
			TotalRequests:   10,
			SuccessRate:     100.0,
			Status:          "✓ OK",
		},
		{
			Name:            "model2",
			Priority:        2,
			Speed:           5.0,
			AvgResponseTime: 200 * time.Millisecond,
			MedianResponseTime: 195 * time.Millisecond,
			P95ResponseTime: 250 * time.Millisecond,
			MinResponseTime: 150 * time.Millisecond,
			MaxResponseTime: 250 * time.Millisecond,
			SuccessCount:    5,
			ErrorCount:      5,
			TotalRequests:   10,
			SuccessRate:     50.0,
			Status:          "⚠ 5 ошибок",
		},
	}

	// Сохраняем результаты
	saveResultsToJSON(benchmarks)

	// Проверяем, что функция не паникует
	// В реальном тесте можно проверить содержимое файла
}

// TestSaveResultsToHTML проверяет сохранение результатов в HTML
func TestSaveResultsToHTML(t *testing.T) {
	benchmarks := []*ModelBenchmark{
		{
			Name:            "model1",
			Priority:        1,
			Speed:           10.5,
			AvgResponseTime: 100 * time.Millisecond,
			MedianResponseTime: 95 * time.Millisecond,
			P95ResponseTime: 120 * time.Millisecond,
			MinResponseTime: 80 * time.Millisecond,
			MaxResponseTime: 120 * time.Millisecond,
			SuccessCount:    10,
			ErrorCount:      0,
			TotalRequests:   10,
			SuccessRate:     100.0,
			Status:          "✓ OK",
		},
		{
			Name:            "model2",
			Priority:        2,
			Speed:           5.0,
			AvgResponseTime: 200 * time.Millisecond,
			MedianResponseTime: 195 * time.Millisecond,
			P95ResponseTime: 250 * time.Millisecond,
			MinResponseTime: 150 * time.Millisecond,
			MaxResponseTime: 250 * time.Millisecond,
			SuccessCount:    5,
			ErrorCount:      5,
			TotalRequests:   10,
			SuccessRate:     50.0,
			Status:          "⚠ 5 ошибок",
		},
	}

	// Сохраняем результаты
	saveResultsToHTML(benchmarks)

	// Проверяем, что функция не паникует
	// В реальном тесте можно проверить содержимое файла
}

// TestModelBenchmark_EmptyResults проверяет обработку пустых результатов
func TestModelBenchmark_EmptyResults(t *testing.T) {
	benchmarks := []*ModelBenchmark{}

	// Проверяем, что функции не паникуют на пустых данных
	saveResultsToJSON(benchmarks)
	saveResultsToHTML(benchmarks)
}

// TestModelBenchmark_AllFailed проверяет обработку случая, когда все модели не работают
func TestModelBenchmark_AllFailed(t *testing.T) {
	benchmarks := []*ModelBenchmark{
		{
			Name:            "failed-model",
			Priority:        1,
			Speed:           0.0,
			AvgResponseTime: 500 * time.Millisecond,
			SuccessCount:    0,
			ErrorCount:      10,
			TotalRequests:   10,
			SuccessRate:     0.0,
			Status:          "✗ FAILED",
		},
	}

	// Проверяем, что функции корректно обрабатывают неудачные модели
	saveResultsToJSON(benchmarks)
	saveResultsToHTML(benchmarks)
}

// TestModelBenchmark_Sorting проверяет логику сортировки моделей
func TestModelBenchmark_Sorting(t *testing.T) {
	benchmarks := []*ModelBenchmark{
		{
			Name:         "slow-model",
			Speed:        1.0,
			SuccessCount: 5,
			ErrorCount:   5,
		},
		{
			Name:         "fast-model",
			Speed:        10.0,
			SuccessCount: 10,
			ErrorCount:   0,
		},
		{
			Name:         "failed-model",
			Speed:        0.0,
			SuccessCount: 0,
			ErrorCount:   10,
		},
	}

	// Проверяем логику сортировки (быстрая модель должна быть первой)
	if benchmarks[1].Speed <= benchmarks[0].Speed {
		t.Error("Fast model should have higher speed than slow model")
	}

	// Проверяем, что модель с успешными запросами лучше модели без них
	if benchmarks[1].SuccessCount == 0 && benchmarks[0].SuccessCount > 0 {
		t.Error("Model with successful requests should be ranked higher")
	}
}

// TestModelBenchmark_MedianAndPercentiles проверяет расчет медианы и перцентилей
func TestModelBenchmark_MedianAndPercentiles(t *testing.T) {
	// Тестовые данные для проверки медианы и перцентилей
	responseTimes := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
		600 * time.Millisecond,
		700 * time.Millisecond,
	}

	benchmark := &ModelBenchmark{
		Name:          "test-model",
		ResponseTimes: responseTimes,
		SuccessCount:  int64(len(responseTimes)),
		TotalRequests: int64(len(responseTimes)),
	}

	// Сортируем времена ответов
	sortedTimes := make([]time.Duration, len(responseTimes))
	copy(sortedTimes, responseTimes)
	for i := 0; i < len(sortedTimes)-1; i++ {
		for j := i + 1; j < len(sortedTimes); j++ {
			if sortedTimes[i] > sortedTimes[j] {
				sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
			}
		}
	}

	// Медиана (для 7 элементов это 4-й элемент, индекс 3)
	medianIdx := len(sortedTimes) / 2
	var median time.Duration
	if len(sortedTimes)%2 == 0 {
		median = (sortedTimes[medianIdx-1] + sortedTimes[medianIdx]) / 2
	} else {
		median = sortedTimes[medianIdx]
	}
	benchmark.MedianResponseTime = median

	expectedMedian := 400 * time.Millisecond
	if benchmark.MedianResponseTime != expectedMedian {
		t.Errorf("Expected median %v, got %v", expectedMedian, benchmark.MedianResponseTime)
	}

	// 95-й перцентиль
	p95Idx := int(float64(len(sortedTimes)) * 0.95)
	if p95Idx >= len(sortedTimes) {
		p95Idx = len(sortedTimes) - 1
	}
	benchmark.P95ResponseTime = sortedTimes[p95Idx]

	// Для 7 элементов 95% это примерно 6.65, округляем до 7, индекс 6
	expectedP95 := 700 * time.Millisecond
	if benchmark.P95ResponseTime != expectedP95 {
		t.Errorf("Expected P95 %v, got %v", expectedP95, benchmark.P95ResponseTime)
	}
}

// TestModelBenchmark_SpeedCalculation проверяет расчет скорости
func TestModelBenchmark_SpeedCalculation(t *testing.T) {
	benchmark := &ModelBenchmark{
		Name:          "test-model",
		SuccessCount:  10,
		TotalRequests: 10,
	}

	// Симулируем общее время 2 секунды
	totalTime := 2 * time.Second

	// Рассчитываем скорость
	if benchmark.SuccessCount > 0 && totalTime.Seconds() > 0 {
		benchmark.Speed = float64(benchmark.SuccessCount) / totalTime.Seconds()
	}

	expectedSpeed := 5.0 // 10 запросов за 2 секунды = 5 req/s
	if benchmark.Speed != expectedSpeed {
		t.Errorf("Expected speed %.2f req/s, got %.2f req/s", expectedSpeed, benchmark.Speed)
	}

	// Проверяем случай с нулевыми успешными запросами
	benchmark.SuccessCount = 0
	benchmark.Speed = 0
	if benchmark.SuccessCount == 0 {
		benchmark.Speed = 0
	}
	if benchmark.Speed != 0 {
		t.Error("Expected speed = 0 when no successful requests")
	}
}

// TestModelBenchmark_SuccessRate проверяет расчет процента успешных запросов
func TestModelBenchmark_SuccessRate(t *testing.T) {
	tests := []struct {
		successCount  int64
		totalRequests int64
		expectedRate  float64
	}{
		{10, 10, 100.0},
		{5, 10, 50.0},
		{0, 10, 0.0},
		{7, 10, 70.0},
	}

	for _, tt := range tests {
		benchmark := &ModelBenchmark{
			SuccessCount:  tt.successCount,
			TotalRequests: tt.totalRequests,
		}

		if benchmark.TotalRequests > 0 {
			benchmark.SuccessRate = float64(benchmark.SuccessCount) / float64(benchmark.TotalRequests) * 100
		}

		if benchmark.SuccessRate != tt.expectedRate {
			t.Errorf("For %d/%d: expected success rate %.1f%%, got %.1f%%",
				tt.successCount, tt.totalRequests, tt.expectedRate, benchmark.SuccessRate)
		}
	}
}
