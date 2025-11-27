package server

import (
	"os"
	"testing"
	"time"

	"httpserver/internal/config"
)

// TestLoadConfig_DefaultValues проверяет загрузку конфигурации с дефолтными значениями
func TestLoadConfig_DefaultValues(t *testing.T) {
	// Очищаем переменные окружения для чистого теста
	os.Clearenv()
	
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	
	if cfg.Port != "9999" {
		t.Errorf("Expected Port=9999, got %s", cfg.Port)
	}
	
	if cfg.DatabasePath != "data.db" {
		t.Errorf("Expected DatabasePath=data.db, got %s", cfg.DatabasePath)
	}
	
	if cfg.NormalizedDatabasePath != "normalized_data.db" {
		t.Errorf("Expected NormalizedDatabasePath=normalized_data.db, got %s", cfg.NormalizedDatabasePath)
	}
	
	if cfg.ServiceDatabasePath != "service.db" {
		t.Errorf("Expected ServiceDatabasePath=service.db, got %s", cfg.ServiceDatabasePath)
	}
	
	if cfg.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns=25, got %d", cfg.MaxOpenConns)
	}
	
	if cfg.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns=5, got %d", cfg.MaxIdleConns)
	}
	
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("Expected ConnMaxLifetime=5m, got %v", cfg.ConnMaxLifetime)
	}
}

// TestLoadConfig_EnvironmentVariables проверяет загрузку конфигурации из переменных окружения
func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	os.Clearenv()
	
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("DATABASE_PATH", "/custom/data.db")
	os.Setenv("NORMALIZED_DATABASE_PATH", "/custom/normalized.db")
	os.Setenv("SERVICE_DATABASE_PATH", "/custom/service.db")
	os.Setenv("DB_MAX_OPEN_CONNS", "50")
	os.Setenv("DB_MAX_IDLE_CONNS", "10")
	os.Setenv("DB_CONN_MAX_LIFETIME", "10m")
	os.Setenv("ARLIAI_API_KEY", "test-api-key")
	os.Setenv("ARLIAI_MODEL", "test-model")
	
	defer os.Clearenv()
	
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	
	if cfg.Port != "8080" {
		t.Errorf("Expected Port=8080, got %s", cfg.Port)
	}
	
	if cfg.DatabasePath != "/custom/data.db" {
		t.Errorf("Expected DatabasePath=/custom/data.db, got %s", cfg.DatabasePath)
	}
	
	if cfg.MaxOpenConns != 50 {
		t.Errorf("Expected MaxOpenConns=50, got %d", cfg.MaxOpenConns)
	}
	
	if cfg.ArliaiAPIKey != "test-api-key" {
		t.Errorf("Expected ArliaiAPIKey=test-api-key, got %s", cfg.ArliaiAPIKey)
	}
}

// TestConfig_Validate проверяет валидацию конфигурации
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Port:                  "8080",
				DatabasePath:          "data.db",
				NormalizedDatabasePath: "normalized.db",
				ServiceDatabasePath:   "service.db",
				MaxOpenConns:          25,
				MaxIdleConns:          5,
			},
			wantErr: false,
		},
		{
			name: "empty port",
			config: &Config{
				Port:                  "",
				DatabasePath:          "data.db",
				NormalizedDatabasePath: "normalized.db",
				ServiceDatabasePath:   "service.db",
				MaxOpenConns:          25,
				MaxIdleConns:          5,
			},
			wantErr: true,
		},
		{
			name: "empty database path",
			config: &Config{
				Port:                  "8080",
				DatabasePath:          "",
				NormalizedDatabasePath: "normalized.db",
				ServiceDatabasePath:   "service.db",
				MaxOpenConns:          25,
				MaxIdleConns:          5,
			},
			wantErr: true,
		},
		{
			name: "zero max open conns",
			config: &Config{
				Port:                  "8080",
				DatabasePath:          "data.db",
				NormalizedDatabasePath: "normalized.db",
				ServiceDatabasePath:   "service.db",
				MaxOpenConns:          0,
				MaxIdleConns:          5,
			},
			wantErr: true,
		},
		{
			name: "zero max idle conns",
			config: &Config{
				Port:                  "8080",
				DatabasePath:          "data.db",
				NormalizedDatabasePath: "normalized.db",
				ServiceDatabasePath:   "service.db",
				MaxOpenConns:          25,
				MaxIdleConns:          0,
			},
			wantErr: true,
		},
		{
			name: "max idle greater than max open",
			config: &Config{
				Port:                  "8080",
				DatabasePath:          "data.db",
				NormalizedDatabasePath: "normalized.db",
				ServiceDatabasePath:   "service.db",
				MaxOpenConns:          10,
				MaxIdleConns:          20,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLoadEnrichmentConfig проверяет загрузку конфигурации обогащения
func TestLoadEnrichmentConfig(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()
	
	enrichmentConfig := config.LoadEnrichmentConfig()
	
	if enrichmentConfig == nil {
		t.Fatal("LoadEnrichmentConfig() returned nil")
	}
	
	// Проверяем дефолтные значения
	if enrichmentConfig.MinQualityScore != 0.3 {
		t.Errorf("Expected MinQualityScore=0.3, got %f", enrichmentConfig.MinQualityScore)
	}
	
	// Проверяем, что кэш включен по умолчанию
	if enrichmentConfig.Cache == nil {
		t.Fatal("Cache config is nil")
	}
	
	if !enrichmentConfig.Cache.Enabled {
		t.Error("Cache should be enabled by default")
	}
}

// TestLoadEnrichmentConfig_EnvironmentVariables проверяет загрузку конфигурации обогащения из переменных окружения
func TestLoadEnrichmentConfig_EnvironmentVariables(t *testing.T) {
	os.Clearenv()
	
	os.Setenv("ENRICHMENT_ENABLED", "false")
	os.Setenv("ENRICHMENT_AUTO_ENRICH", "false")
	os.Setenv("ENRICHMENT_MIN_QUALITY_SCORE", "0.5")
	os.Setenv("DADATA_ENABLED", "true")
	os.Setenv("DADATA_API_KEY", "test-dadata-key")
	os.Setenv("DADATA_SECRET_KEY", "test-dadata-secret")
	
	defer os.Clearenv()
	
	enrichmentConfig := config.LoadEnrichmentConfig()
	
	if enrichmentConfig.Enabled {
		t.Error("Expected Enabled=false")
	}
	
	if enrichmentConfig.AutoEnrich {
		t.Error("Expected AutoEnrich=false")
	}
	
	if enrichmentConfig.MinQualityScore != 0.5 {
		t.Errorf("Expected MinQualityScore=0.5, got %f", enrichmentConfig.MinQualityScore)
	}
	
	if enrichmentConfig.Services["dadata"] == nil {
		t.Fatal("Dadata service config is nil")
	}
	
	if enrichmentConfig.Services["dadata"].APIKey != "test-dadata-key" {
		t.Errorf("Expected Dadata APIKey=test-dadata-key, got %s", enrichmentConfig.Services["dadata"].APIKey)
	}
}

