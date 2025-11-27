package config

import (
	"testing"
	"time"
)

func TestConfigLogLevelValidation(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		wantError bool
	}{
		{"Valid DEBUG", "DEBUG", false},
		{"Valid INFO", "INFO", false},
		{"Valid WARN", "WARN", false},
		{"Valid ERROR", "ERROR", false},
		{"Valid lowercase debug", "debug", false},
		{"Valid lowercase info", "info", false},
		{"Invalid value", "INVALID", true},
		{"Empty string", "", false}, // Пустая строка допустима (будет использовано значение по умолчанию)
		{"Mixed case", "DeBuG", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:                       "9999",
				DatabasePath:               "data.db",
				NormalizedDatabasePath:     "normalized_data.db",
				ServiceDatabasePath:        "service.db",
				ArliaiModel:                "GLM-4.5-Air",
				MaxOpenConns:               25,
				MaxIdleConns:               5,
				ConnMaxLifetime:            5 * time.Minute,
				LogBufferSize:              100,
				LogLevel:                   tt.logLevel,
				NormalizerEventsBufferSize: 100,
				AggregationStrategy:        "first_success",
				AITimeout:                   30 * time.Second,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestConfigLogLevelDefault(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.LogLevel == "" {
		t.Error("LogLevel should have a default value")
	}

	// Проверяем, что значение по умолчанию валидно
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Default LogLevel should be valid, got error: %v", err)
	}
}

