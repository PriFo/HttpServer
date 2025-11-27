package nomenclature

import (
	"os"
	"testing"
	"time"
)

// TestConfig проверяет структуру конфигурации
func TestConfig(t *testing.T) {
	config := Config{
		DatabasePath:   "./test.db",
		KpvedFilePath:  "./kpved.txt",
		ArliaiAPIKey:   "test-key",
		AIModel:        "test-model",
		MaxWorkers:     2,
		BatchSize:      50,
		MaxRetries:     3,
		RequestTimeout: 30 * time.Second,
	}
	
	if config.DatabasePath == "" {
		t.Error("Config.DatabasePath should not be empty")
	}
	
	if config.KpvedFilePath == "" {
		t.Error("Config.KpvedFilePath should not be empty")
	}
	
	if config.MaxWorkers <= 0 {
		t.Error("Config.MaxWorkers should be positive")
	}
	
	if config.BatchSize <= 0 {
		t.Error("Config.BatchSize should be positive")
	}
	
	if config.MaxRetries < 0 {
		t.Error("Config.MaxRetries should be non-negative")
	}
	
	if config.RequestTimeout <= 0 {
		t.Error("Config.RequestTimeout should be positive")
	}
}

// TestDefaultConfig проверяет конфигурацию по умолчанию
func TestDefaultConfig(t *testing.T) {
	// Сохраняем текущее значение переменной окружения
	originalModel := os.Getenv("ARLIAI_MODEL")
	defer os.Setenv("ARLIAI_MODEL", originalModel)
	
	// Тест с установленной переменной окружения
	os.Setenv("ARLIAI_MODEL", "custom-model")
	config := DefaultConfig()
	
	if config.AIModel != "custom-model" {
		t.Errorf("Config.AIModel = %s, want 'custom-model'", config.AIModel)
	}
	
	if config.DatabasePath == "" {
		t.Error("Config.DatabasePath should not be empty")
	}
	
	if config.KpvedFilePath == "" {
		t.Error("Config.KpvedFilePath should not be empty")
	}
	
	if config.MaxWorkers != 2 {
		t.Errorf("Config.MaxWorkers = %d, want 2", config.MaxWorkers)
	}
	
	if config.BatchSize <= 0 {
		t.Error("Config.BatchSize should be positive")
	}
	
	if config.MaxRetries <= 0 {
		t.Error("Config.MaxRetries should be positive")
	}
	
	if config.RequestTimeout <= 0 {
		t.Error("Config.RequestTimeout should be positive")
	}
}

// TestDefaultConfig_NoEnvVar проверяет конфигурацию без переменной окружения
func TestDefaultConfig_NoEnvVar(t *testing.T) {
	// Сохраняем и очищаем переменную окружения
	originalModel := os.Getenv("ARLIAI_MODEL")
	defer os.Setenv("ARLIAI_MODEL", originalModel)
	
	os.Unsetenv("ARLIAI_MODEL")
	
	config := DefaultConfig()
	
	// Должна использоваться модель по умолчанию
	if config.AIModel != "GLM-4.5-Air" {
		t.Errorf("Config.AIModel = %s, want 'GLM-4.5-Air'", config.AIModel)
	}
}

// TestConfig_Validation проверяет валидацию конфигурации
func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				DatabasePath:   "./test.db",
				KpvedFilePath:  "./kpved.txt",
				ArliaiAPIKey:   "test-key",
				AIModel:        "test-model",
				MaxWorkers:     2,
				BatchSize:      50,
				MaxRetries:     3,
				RequestTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty database path",
			config: Config{
				DatabasePath:   "",
				KpvedFilePath:  "./kpved.txt",
				MaxWorkers:     2,
			},
			wantErr: true,
		},
		{
			name: "zero max workers",
			config: Config{
				DatabasePath:  "./test.db",
				KpvedFilePath: "./kpved.txt",
				MaxWorkers:    0,
			},
			wantErr: true,
		},
		{
			name: "zero batch size",
			config: Config{
				DatabasePath:  "./test.db",
				KpvedFilePath: "./kpved.txt",
				MaxWorkers:    2,
				BatchSize:     0,
			},
			wantErr: true,
		},
		{
			name: "negative max retries",
			config: Config{
				DatabasePath:  "./test.db",
				KpvedFilePath: "./kpved.txt",
				MaxWorkers:    2,
				MaxRetries:    -1,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.config.DatabasePath == "" ||
				tt.config.MaxWorkers <= 0 ||
				tt.config.BatchSize <= 0 ||
				tt.config.MaxRetries < 0)
			
			if hasError != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

