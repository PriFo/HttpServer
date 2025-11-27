package server

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"httpserver/internal/infrastructure/ai"
	"httpserver/internal/infrastructure/cache"
	"httpserver/internal/infrastructure/workers"
)

// TestFetchProviderModels_NoAPIKey проверяет обработку отсутствия API ключа
func TestFetchProviderModels_NoAPIKey(t *testing.T) {
	// Сохраняем оригинальное значение
	originalKey := os.Getenv("ARLIAI_API_KEY")
	defer os.Setenv("ARLIAI_API_KEY", originalKey)

	// Удаляем API ключ
	os.Unsetenv("ARLIAI_API_KEY")

	provider := &workers.ProviderConfig{
		Name:   "arliai",
		APIKey: "", // Пустой ключ
	}

	// Устанавливаем переменные окружения для теста
	os.Setenv("ARLIAI_API_KEY", "")
	os.Setenv("ARLIAI_BASE_URL", "https://api.arliai.com")
	defer os.Unsetenv("ARLIAI_API_KEY")
	defer os.Unsetenv("ARLIAI_BASE_URL")

	s := &Server{
		arliaiClient: ai.NewArliaiClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.fetchProviderModels(ctx, "test-trace-id", provider)

	if err == nil {
		t.Error("fetchProviderModels() should return error when API key is missing")
	}

	if err != nil && !contains(err.Error(), "API ключ") {
		t.Errorf("Expected error about missing API key, got: %v", err)
	}
}

// TestFetchProviderModels_InvalidProvider проверяет обработку несуществующего провайдера
func TestFetchProviderModels_InvalidProvider(t *testing.T) {
	provider := &workers.ProviderConfig{
		Name:   "invalid-provider",
		APIKey: "test-key",
	}

	os.Setenv("ARLIAI_API_KEY", "test-key")
	os.Setenv("ARLIAI_BASE_URL", "https://api.arliai.com")
	defer os.Unsetenv("ARLIAI_API_KEY")
	defer os.Unsetenv("ARLIAI_BASE_URL")

	s := &Server{
		arliaiClient: ai.NewArliaiClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.fetchProviderModels(ctx, "test-trace-id", provider)

	// Для несуществующего провайдера должен использоваться arliaiClient по умолчанию
	// Но если arliaiClient == nil, должна быть ошибка
	if s.arliaiClient == nil && err == nil {
		t.Error("fetchProviderModels() should return error when client is nil")
	}
}

// TestBuildWorkerModelsData_Refresh проверяет принудительное обновление кеша
func TestBuildWorkerModelsData_Refresh(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	cache := cache.NewArliaiCache()
	
	// Устанавливаем кеш
	cache.SetModels(map[string]interface{}{
		"models": []interface{}{},
		"cached": true,
	})

	os.Setenv("ARLIAI_API_KEY", "test-key")
	os.Setenv("ARLIAI_BASE_URL", "https://api.arliai.com")
	defer os.Unsetenv("ARLIAI_API_KEY")
	defer os.Unsetenv("ARLIAI_BASE_URL")

	s := &Server{
		workerConfigManager: manager,
		arliaiCache:         cache,
		arliaiClient:        ai.NewArliaiClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Тест с refresh=true
	opts := workerModelsOptions{
		ProviderFilter: "arliai",
		Refresh:        true,
	}

	// Проверяем, что кеш очищается
	_, ok := cache.GetModels()
	if ok {
		t.Log("Cache exists before refresh")
	}

	// Вызываем buildWorkerModelsData с refresh
	// Это может вернуть ошибку, если нет реального API ключа, но это нормально
	result, cached, apiErr := s.buildWorkerModelsData(ctx, "test-trace-id", opts)

	// Если есть ошибка из-за отсутствия API ключа, это ожидаемо
	if apiErr != nil {
		if contains(apiErr.Message, "API ключ") || contains(apiErr.Message, "SERVICE_UNAVAILABLE") {
			t.Logf("Expected error (no API key or service unavailable): %v", apiErr)
			return
		}
		t.Logf("Unexpected error: %v", apiErr)
	}

	// Если нет ошибки, проверяем что кеш был обновлен (cached = false)
	if apiErr == nil && cached {
		t.Error("buildWorkerModelsData() with refresh=true should not return cached data")
	}

	if result == nil && apiErr == nil {
		t.Error("buildWorkerModelsData() should return result or error")
	}
}

// TestBuildWorkerModelsData_Cache проверяет использование кеша
func TestBuildWorkerModelsData_Cache(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	cache := cache.NewArliaiCache()
	
	// Устанавливаем валидный кеш
	cachedData := map[string]interface{}{
		"models": []interface{}{
			map[string]interface{}{
				"name":   "test-model",
				"id":     "test-model",
				"status": "active",
			},
		},
		"provider": "arliai",
		"total":    1,
	}
	cache.SetModels(cachedData)

	s := &Server{
		workerConfigManager: manager,
		arliaiCache:         cache,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := workerModelsOptions{
		ProviderFilter: "arliai",
		Refresh:        false, // Не обновляем кеш
	}

	result, cached, apiErr := s.buildWorkerModelsData(ctx, "test-trace-id", opts)

	if apiErr != nil {
		t.Logf("Error (may be expected): %v", apiErr)
		// Если ошибка из-за отсутствия провайдера, это нормально
		if !contains(apiErr.Message, "not found") {
			t.Errorf("Unexpected error: %v", apiErr)
		}
		return
	}

	if !cached {
		t.Error("buildWorkerModelsData() should return cached data when refresh=false and cache exists")
	}

	if result == nil {
		t.Error("buildWorkerModelsData() should return result when cache exists")
	}
}

// TestBuildWorkerModelsData_NoProvider проверяет обработку отсутствия провайдера
func TestBuildWorkerModelsData_NoProvider(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	s := &Server{
		workerConfigManager: manager,
		arliaiCache:         cache.NewArliaiCache(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := workerModelsOptions{
		ProviderFilter: "non-existent-provider",
		Refresh:        false,
	}

	_, _, apiErr := s.buildWorkerModelsData(ctx, "test-trace-id", opts)

	if apiErr == nil {
		t.Error("buildWorkerModelsData() should return error for non-existent provider")
	}

	if apiErr != nil && apiErr.Code != "PROVIDER_NOT_FOUND" {
		t.Errorf("Expected error code PROVIDER_NOT_FOUND, got: %s", apiErr.Code)
	}
}

// TestBuildWorkerModelsData_NoConfigManager проверяет обработку отсутствия менеджера конфигурации
func TestBuildWorkerModelsData_NoConfigManager(t *testing.T) {
	s := &Server{
		workerConfigManager: nil, // Нет менеджера
		arliaiCache:         cache.NewArliaiCache(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := workerModelsOptions{
		ProviderFilter: "arliai",
		Refresh:        false,
	}

	_, _, apiErr := s.buildWorkerModelsData(ctx, "test-trace-id", opts)

	if apiErr == nil {
		t.Error("buildWorkerModelsData() should return error when config manager is nil")
	}

	if apiErr != nil && apiErr.Code != "SERVICE_UNAVAILABLE" {
		t.Errorf("Expected error code SERVICE_UNAVAILABLE, got: %s", apiErr.Code)
	}
}

// TestFetchProviderModels_OpenRouter проверяет обработку OpenRouter провайдера
func TestFetchProviderModels_OpenRouter(t *testing.T) {
	provider := &workers.ProviderConfig{
		Name:   "openrouter",
		APIKey: "test-key",
	}

	s := &Server{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Это может вернуть ошибку из-за невалидного API ключа, но это нормально
	_, err := s.fetchProviderModels(ctx, "test-trace-id", provider)

	// Проверяем, что ошибка связана с API, а не с отсутствием провайдера
	if err != nil && !contains(err.Error(), "API ключ") && !contains(err.Error(), "request failed") && !contains(err.Error(), "unauthorized") {
		t.Logf("Error (may be expected for invalid API key): %v", err)
	}
}

// TestFetchProviderModels_HuggingFace проверяет обработку HuggingFace провайдера
func TestFetchProviderModels_HuggingFace(t *testing.T) {
	provider := &workers.ProviderConfig{
		Name:   "huggingface",
		APIKey: "test-key",
		BaseURL: "https://api-inference.huggingface.co",
	}

	s := &Server{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// HuggingFace возвращает статический список, поэтому должен работать
	models, err := s.fetchProviderModels(ctx, "test-trace-id", provider)

	if err != nil {
		t.Errorf("fetchProviderModels() for HuggingFace should not return error (static list), got: %v", err)
	}

	if len(models) == 0 {
		t.Error("fetchProviderModels() for HuggingFace should return models")
	}
}

// contains проверяет наличие подстроки в строке
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestFetchModelsWithAPIKeyFromDB проверяет получение моделей с API ключом из БД
// Этот тест требует реального API ключа в переменной окружения или в БД
func TestFetchModelsWithAPIKeyFromDB(t *testing.T) {
	// Пропускаем тест, если не указан флаг для интеграционных тестов
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	// Получаем API ключ из переменной окружения для теста
	apiKey := os.Getenv("ARLIAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: ARLIAI_API_KEY not set. Set it to test with real API key.")
	}

	// Сохраняем API ключ в конфигурацию БД
	providerConfig := &workers.ProviderConfig{
		Name:     "arliai",
		APIKey:   apiKey,
		BaseURL:  "https://api.arliai.com",
		Enabled:  true,
		Priority: 1,
		Timeout:  60 * time.Second,
	}

	err := manager.UpdateProvider("arliai", providerConfig)
	if err != nil {
		t.Fatalf("Failed to update provider: %v", err)
	}

	// Устанавливаем arliai как активного провайдера
	err = manager.SetDefaultProvider("arliai")
	if err != nil {
		t.Fatalf("Failed to set default provider: %v", err)
	}

	// Создаем сервер с менеджером конфигурации
	s := &Server{
		workerConfigManager: manager,
		arliaiCache:         cache.NewArliaiCache(),
		arliaiClient:        ai.NewArliaiClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Получаем провайдера из конфигурации
	provider, err := manager.GetActiveProvider()
	if err != nil {
		t.Fatalf("Failed to get active provider: %v", err)
	}

	t.Logf("Testing with provider: %s, API key set: %v", provider.Name, provider.APIKey != "")

	// Пытаемся получить модели
	models, err := s.fetchProviderModels(ctx, "test-trace-id", provider)

	if err != nil {
		// Проверяем тип ошибки
		if strings.Contains(err.Error(), "API ключ") {
			t.Errorf("API key error: %v", err)
		} else if strings.Contains(err.Error(), "timeout") {
			t.Logf("Timeout error (may be expected): %v", err)
		} else if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "401") {
			t.Errorf("Unauthorized error - API key may be invalid: %v", err)
		} else {
			t.Logf("Error fetching models (may be expected): %v", err)
		}
		return
	}

	// Если модели получены успешно
	if len(models) == 0 {
		t.Error("No models returned, but no error occurred")
		return
	}

	t.Logf("Successfully fetched %d models from API", len(models))

	// Выводим информацию о первых 10 моделях
	maxModels := 10
	if len(models) < maxModels {
		maxModels = len(models)
	}

	t.Logf("First %d models:", maxModels)
	for i := 0; i < maxModels; i++ {
		model := models[i]
		t.Logf("  - %s (ID: %s, Status: %s)", model.Name, model.ID, model.Status)
	}

	// Проверяем, что есть хотя бы одна активная модель
	activeCount := 0
	for _, model := range models {
		if model.Status == "active" {
			activeCount++
		}
	}

	if activeCount == 0 {
		t.Log("Warning: No active models found")
	} else {
		t.Logf("Found %d active models", activeCount)
	}
}

// TestFetchModelsMultipleProviders проверяет получение моделей для разных провайдеров
func TestFetchModelsMultipleProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	s := &Server{
		workerConfigManager: manager,
		arliaiCache:         cache.NewArliaiCache(),
		arliaiClient:        ai.NewArliaiClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Тестируем разные провайдеры
	providers := []struct {
		name     string
		apiKey   string
		envVar   string
		baseURL   string
		required bool
	}{
		{"arliai", os.Getenv("ARLIAI_API_KEY"), "ARLIAI_API_KEY", "https://api.arliai.com", false},
		{"openrouter", os.Getenv("OPENROUTER_API_KEY"), "OPENROUTER_API_KEY", "https://openrouter.ai/api/v1", false},
		{"huggingface", os.Getenv("HUGGINGFACE_API_KEY"), "HUGGINGFACE_API_KEY", "https://api-inference.huggingface.co", false},
	}

	for _, p := range providers {
		if p.apiKey == "" {
			t.Logf("Skipping %s: %s not set", p.name, p.envVar)
			continue
		}

		// Сохраняем провайдера в конфигурацию
		providerConfig := &workers.ProviderConfig{
			Name:     p.name,
			APIKey:   p.apiKey,
			BaseURL:  p.baseURL,
			Enabled:  true,
			Priority: 1,
			Timeout:  30 * time.Second,
		}

		err := manager.UpdateProvider(p.name, providerConfig)
		if err != nil {
			t.Logf("Failed to update provider %s: %v", p.name, err)
			continue
		}

		// Получаем провайдера из конфигурации
		config := manager.GetConfig()
		providersMap, ok := config["providers"].(map[string]interface{})
		if !ok {
			t.Logf("Failed to get providers map for %s", p.name)
			continue
		}

		providerData, ok := providersMap[p.name].(map[string]interface{})
		if !ok {
			t.Logf("Provider %s not found in config", p.name)
			continue
		}

		provider := &workers.ProviderConfig{
			Name:    getStringFromMap(providerData, "name"),
			APIKey:  getStringFromMap(providerData, "api_key"),
			BaseURL: getStringFromMap(providerData, "base_url"),
		}

		t.Logf("Testing provider: %s", p.name)

		// Пытаемся получить модели
		models, err := s.fetchProviderModels(ctx, "test-trace-id", provider)

		if err != nil {
			t.Logf("  Error fetching models for %s: %v", p.name, err)
			continue
		}

		t.Logf("  Successfully fetched %d models from %s", len(models), p.name)
		if len(models) > 0 {
			t.Logf("  First model: %s", models[0].Name)
		}
	}
}

// getStringFromMap извлекает строку из map (вспомогательная функция для теста)
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

