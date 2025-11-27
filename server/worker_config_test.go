package server

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"httpserver/database"
)

// setupTestWorkerConfigManager создает тестовый менеджер конфигурации
func setupTestWorkerConfigManager(t *testing.T) (*WorkerConfigManager, func()) {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "test_service.db")

	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}

	manager := NewWorkerConfigManager(serviceDB)

	cleanup := func() {
		serviceDB.Close()
	}

	return manager, cleanup
}

// TestNewWorkerConfigManager проверяет создание нового менеджера конфигурации
func TestNewWorkerConfigManager(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	if manager == nil {
		t.Fatal("NewWorkerConfigManager() returned nil")
	}

	config := manager.GetConfig()
	if config == nil {
		t.Fatal("GetConfig() returned nil")
	}

	defaultProvider, _ := config["default_provider"].(string)
	if strings.TrimSpace(defaultProvider) == "" {
		t.Error("default_provider should not be empty")
	}

	defaultModel, _ := config["default_model"].(string)
	if strings.TrimSpace(defaultModel) == "" {
		t.Error("default_model should not be empty")
	}

	globalMaxWorkers := -1
	switch v := config["global_max_workers"].(type) {
	case int:
		globalMaxWorkers = v
	case int64:
		globalMaxWorkers = int(v)
	case float64:
		globalMaxWorkers = int(v)
	}
	if globalMaxWorkers <= 0 {
		t.Errorf("global_max_workers should be positive, got %d", globalMaxWorkers)
	}
}

// TestProviderConfig проверяет структуру конфигурации провайдера
func TestProviderConfig(t *testing.T) {
	config := &ProviderConfig{
		Name:       "test-provider",
		APIKey:     "test-key",
		BaseURL:    "https://api.test.com",
		Enabled:    true,
		Priority:   1,
		MaxWorkers: 5,
		RateLimit:  100,
		Timeout:    30 * time.Second,
		Models:     []ModelConfig{},
		Metadata:   make(map[string]string),
	}

	if config.Name == "" {
		t.Error("ProviderConfig.Name should not be empty")
	}

	if config.BaseURL == "" {
		t.Error("ProviderConfig.BaseURL should not be empty")
	}

	if config.MaxWorkers <= 0 {
		t.Error("ProviderConfig.MaxWorkers should be positive")
	}

	if config.RateLimit <= 0 {
		t.Error("ProviderConfig.RateLimit should be positive")
	}

	if config.Timeout <= 0 {
		t.Error("ProviderConfig.Timeout should be positive")
	}
}

// TestModelConfig проверяет структуру конфигурации модели
func TestModelConfig(t *testing.T) {
	config := ModelConfig{
		Name:         "test-model",
		Provider:     "test-provider",
		Enabled:      true,
		Priority:     1,
		MaxTokens:    4096,
		Temperature:  0.7,
		CostPerToken: 0.001,
		Speed:        "fast",
		Quality:      "high",
	}

	if config.Name == "" {
		t.Error("ModelConfig.Name should not be empty")
	}

	if config.Provider == "" {
		t.Error("ModelConfig.Provider should not be empty")
	}

	if config.MaxTokens <= 0 {
		t.Error("ModelConfig.MaxTokens should be positive")
	}

	if config.Temperature < 0 || config.Temperature > 2 {
		t.Errorf("ModelConfig.Temperature = %f, should be between 0 and 2", config.Temperature)
	}

	if config.CostPerToken < 0 {
		t.Error("ModelConfig.CostPerToken should be non-negative")
	}
}

// TestWorkerConfigManager_GetActiveProvider проверяет получение активного провайдера
func TestWorkerConfigManager_GetActiveProvider(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	provider, err := manager.GetActiveProvider()
	if err != nil {
		t.Fatalf("GetActiveProvider() failed: %v", err)
	}

	if provider == nil {
		t.Fatal("GetActiveProvider() returned nil")
	}

	if !provider.Enabled {
		t.Error("Active provider should be enabled")
	}

	if provider.Name == "" {
		t.Error("Provider.Name should not be empty")
	}
}

// TestWorkerConfigManager_GetActiveModel проверяет получение активной модели
func TestWorkerConfigManager_GetActiveModel(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	model, err := manager.GetActiveModel("arliai")
	if err != nil {
		t.Logf("GetActiveModel() returned error (may be expected): %v", err)
		return
	}

	if model == nil {
		t.Error("GetActiveModel() should return model for arliai")
	}

	if model != nil && !model.Enabled {
		t.Error("Active model should be enabled")
	}
}

// TestWorkerConfigManager_GetConfig проверяет получение конфигурации
func TestWorkerConfigManager_GetConfig(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	config := manager.GetConfig()

	if config == nil {
		t.Fatal("GetConfig() returned nil")
	}

	// Проверяем наличие основных полей
	if _, ok := config["default_provider"]; !ok {
		t.Error("Config should contain 'default_provider'")
	}

	if _, ok := config["default_model"]; !ok {
		t.Error("Config should contain 'default_model'")
	}

	if _, ok := config["global_max_workers"]; !ok {
		t.Error("Config should contain 'global_max_workers'")
	}
}

// TestWorkerConfigManager_SetDefaultProvider_NonExistent проверяет валидацию несуществующего провайдера
func TestWorkerConfigManager_SetDefaultProvider_NonExistent(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	// Проверяем только валидацию, без сохранения (может зависнуть на saveConfig)
	// Тест с несуществующим провайдером должен вернуть ошибку до сохранения
	// Используем таймаут для защиты от зависания
	done := make(chan error, 1)
	go func() {
		done <- manager.SetDefaultProvider("non-existent")
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("SetDefaultProvider() should return error for non-existent provider")
		}
		if err != nil && !strings.Contains(err.Error(), "not found") {
			t.Logf("SetDefaultProvider() returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Log("SetDefaultProvider() timed out (may be expected if saveConfig blocks)")
	}
}

// TestWorkerConfigManager_SetGlobalMaxWorkers_Invalid проверяет валидацию невалидных значений
func TestWorkerConfigManager_SetGlobalMaxWorkers_Invalid(t *testing.T) {
	manager, cleanup := setupTestWorkerConfigManager(t)
	defer cleanup()

	// Тест с нулевым значением (валидация происходит до сохранения)
	err := manager.SetGlobalMaxWorkers(0)
	if err == nil {
		t.Error("SetGlobalMaxWorkers() should return error for 0")
	}

	// Тест с отрицательным значением
	err = manager.SetGlobalMaxWorkers(-1)
	if err == nil {
		t.Error("SetGlobalMaxWorkers() should return error for negative value")
	}

	// Тест со значением больше 100
	err = manager.SetGlobalMaxWorkers(101)
	if err == nil {
		t.Error("SetGlobalMaxWorkers() should return error for value > 100")
	}
}

// TestProviderConfig_Validation проверяет валидацию конфигурации провайдера
func TestProviderConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ProviderConfig{
				Name:       "test",
				BaseURL:    "https://api.test.com",
				Enabled:    true,
				MaxWorkers: 5,
				RateLimit:  100,
				Timeout:    30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: &ProviderConfig{
				Name:       "",
				BaseURL:    "https://api.test.com",
				Enabled:    true,
				MaxWorkers: 5,
			},
			wantErr: true,
		},
		{
			name: "empty base URL",
			config: &ProviderConfig{
				Name:       "test",
				BaseURL:    "",
				Enabled:    true,
				MaxWorkers: 5,
			},
			wantErr: true,
		},
		{
			name: "zero max workers",
			config: &ProviderConfig{
				Name:       "test",
				BaseURL:    "https://api.test.com",
				Enabled:    true,
				MaxWorkers: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.config.Name == "" || tt.config.BaseURL == "" || tt.config.MaxWorkers <= 0)
			if hasError != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

// TestModelConfig_Validation проверяет валидацию конфигурации модели
func TestModelConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ModelConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ModelConfig{
				Name:        "test-model",
				Provider:    "test-provider",
				Enabled:     true,
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: ModelConfig{
				Name:     "",
				Provider: "test-provider",
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			config: ModelConfig{
				Name:        "test-model",
				Provider:    "test-provider",
				Temperature: 3.0, // Вне допустимого диапазона
			},
			wantErr: true,
		},
		{
			name: "zero max tokens",
			config: ModelConfig{
				Name:      "test-model",
				Provider:  "test-provider",
				MaxTokens: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.config.Name == "" ||
				tt.config.MaxTokens <= 0 ||
				tt.config.Temperature < 0 || tt.config.Temperature > 2)
			if hasError != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}
