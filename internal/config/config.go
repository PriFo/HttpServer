package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"httpserver/database"
	"httpserver/enrichment"
)

// Config конфигурация сервера
type Config struct {
	// Сервер
	Port string `json:"port"`

	// Базы данных
	DatabasePath           string `json:"database_path"`
	NormalizedDatabasePath string `json:"normalized_database_path"`
	ServiceDatabasePath    string `json:"service_database_path"`

	// AI конфигурация
	ArliaiAPIKey string `json:"arliai_api_key"`
	ArliaiModel  string `json:"arliai_model"`

	// Connection pooling
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// Логирование
	LogBufferSize int    `json:"log_buffer_size"`
	LogLevel       string `json:"log_level"`

	// Нормализация
	NormalizerEventsBufferSize int `json:"normalizer_events_buffer_size"`

	// Мульти-провайдерная нормализация
	MultiProviderEnabled bool          `json:"multi_provider_enabled"`
	AggregationStrategy  string        `json:"aggregation_strategy"`
	AITimeout            time.Duration `json:"ai_timeout"`

	// Обогащение контрагентов
	Enrichment *EnrichmentConfig `json:"enrichment"`

	// Веб-поиск для валидации
	WebSearch *WebSearchConfig `json:"web_search"`
}

// EnrichmentConfig конфигурация обогащения
type EnrichmentConfig struct {
	Enabled         bool                                  `json:"enabled"`
	AutoEnrich      bool                                  `json:"auto_enrich"`
	MinQualityScore float64                               `json:"min_quality_score"`
	Services        map[string]*enrichment.EnricherConfig `json:"services"`
	Cache           *enrichment.CacheConfig               `json:"cache"`
}

// LoadConfig загружает конфигурацию из сервисной БД (если serviceDB передан) или из переменных окружения
func LoadConfig(serviceDB ...*database.ServiceDB) (*Config, error) {
	var config *Config

	// Пытаемся загрузить из БД, если передан serviceDB
	if len(serviceDB) > 0 && serviceDB[0] != nil {
		configJSONStr, err := serviceDB[0].GetAppConfig()
		if err == nil && configJSONStr != "" {
			// Парсим JSON из БД
			var cfgJSON configJSON
			if err := json.Unmarshal([]byte(configJSONStr), &cfgJSON); err == nil {
				// Преобразуем configJSON в Config
				connMaxLifetime, err := time.ParseDuration(cfgJSON.ConnMaxLifetime)
				if err != nil {
					connMaxLifetime = 5 * time.Minute // fallback
				}
				aiTimeout, err := time.ParseDuration(cfgJSON.AITimeout)
				if err != nil {
					aiTimeout = 30 * time.Second // fallback
				}

				config = &Config{
					Port:                       cfgJSON.Port,
					DatabasePath:               cfgJSON.DatabasePath,
					NormalizedDatabasePath:     cfgJSON.NormalizedDatabasePath,
					ServiceDatabasePath:        cfgJSON.ServiceDatabasePath,
					ArliaiAPIKey:               cfgJSON.ArliaiAPIKey,
					ArliaiModel:                cfgJSON.ArliaiModel,
					MaxOpenConns:               cfgJSON.MaxOpenConns,
					MaxIdleConns:               cfgJSON.MaxIdleConns,
					ConnMaxLifetime:            connMaxLifetime,
					LogBufferSize:              cfgJSON.LogBufferSize,
					LogLevel:                   cfgJSON.LogLevel,
					NormalizerEventsBufferSize: cfgJSON.NormalizerEventsBufferSize,
					MultiProviderEnabled:       cfgJSON.MultiProviderEnabled,
					AggregationStrategy:        cfgJSON.AggregationStrategy,
					AITimeout:                  aiTimeout,
					Enrichment:                 cfgJSON.Enrichment,
					WebSearch:                  cfgJSON.WebSearch,
				}

				log.Printf("Config loaded from service database")
				// Валидация
				if err := config.Validate(); err != nil {
					log.Printf("Invalid config from DB, falling back to env: %v", err)
					config = nil // Сбрасываем, чтобы загрузить из env
				} else {
					return config, nil
				}
			} else {
				log.Printf("Failed to parse config from DB, falling back to env: %v", err)
			}
		}
	}

	// Fallback на переменные окружения
	config = &Config{
		// Сервер
		Port: getEnv("SERVER_PORT", "9999"),

		// Базы данных
		DatabasePath:           getEnv("DATABASE_PATH", "data.db"),
		NormalizedDatabasePath: getEnv("NORMALIZED_DATABASE_PATH", "normalized_data.db"),
		ServiceDatabasePath:    getEnv("SERVICE_DATABASE_PATH", "service.db"),

		// AI конфигурация
		ArliaiAPIKey: os.Getenv("ARLIAI_API_KEY"),
		ArliaiModel:  getEnv("ARLIAI_MODEL", "GLM-4.5-Air"),

		// Connection pooling
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),

		// Логирование
		LogBufferSize: getEnvInt("LOG_BUFFER_SIZE", 100),
		LogLevel:      getEnv("LOG_LEVEL", "INFO"),

		// Нормализация
		NormalizerEventsBufferSize: getEnvInt("NORMALIZER_EVENTS_BUFFER_SIZE", 100),

		// Мульти-провайдерная нормализация
		MultiProviderEnabled: getEnv("MULTI_PROVIDER_ENABLED", "false") == "true",
		AggregationStrategy:  getEnv("AGGREGATION_STRATEGY", "first_success"),
		AITimeout:            getEnvDuration("AI_TIMEOUT", 30*time.Second),

		// Обогащение
		Enrichment: LoadEnrichmentConfig(),

		// Веб-поиск
		WebSearch: LoadWebSearchConfig(),
	}

	// Валидация
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// LoadEnrichmentConfig загружает конфигурацию обогащения
func LoadEnrichmentConfig() *EnrichmentConfig {
	enabled := getEnv("ENRICHMENT_ENABLED", "true") == "true"
	autoEnrich := getEnv("ENRICHMENT_AUTO_ENRICH", "true") == "true"
	minQualityScore := 0.3
	if scoreStr := os.Getenv("ENRICHMENT_MIN_QUALITY_SCORE"); scoreStr != "" {
		if score, err := strconv.ParseFloat(scoreStr, 64); err == nil {
			minQualityScore = score
		}
	}

	services := make(map[string]*enrichment.EnricherConfig)

	// Dadata конфигурация
	dadataEnabled := getEnv("DADATA_ENABLED", "true") == "true"
	if dadataEnabled {
		services["dadata"] = &enrichment.EnricherConfig{
			APIKey:      os.Getenv("DADATA_API_KEY"),
			SecretKey:   os.Getenv("DADATA_SECRET_KEY"),
			BaseURL:     getEnv("DADATA_BASE_URL", "https://suggestions.dadata.ru"),
			Timeout:     getEnvDuration("DADATA_TIMEOUT", 30*time.Second),
			MaxRequests: getEnvInt("DADATA_MAX_REQUESTS", 100),
			Enabled:     dadataEnabled,
			Priority:    getEnvInt("DADATA_PRIORITY", 1),
		}
	}

	// Adata конфигурация
	adataEnabled := getEnv("ADATA_ENABLED", "true") == "true"
	if adataEnabled {
		services["adata"] = &enrichment.EnricherConfig{
			APIKey:      os.Getenv("ADATA_API_KEY"),
			BaseURL:     getEnv("ADATA_BASE_URL", "https://adata.kz"),
			Timeout:     getEnvDuration("ADATA_TIMEOUT", 30*time.Second),
			MaxRequests: getEnvInt("ADATA_MAX_REQUESTS", 50),
			Enabled:     adataEnabled,
			Priority:    getEnvInt("ADATA_PRIORITY", 2),
		}
	}

	// Gisp конфигурация
	gispEnabled := getEnv("GISP_ENABLED", "false") == "true"
	if gispEnabled {
		services["gisp"] = &enrichment.EnricherConfig{
			APIKey:      os.Getenv("GISP_API_KEY"),
			BaseURL:     getEnv("GISP_BASE_URL", "https://gisp.gov.ru"),
			Timeout:     getEnvDuration("GISP_TIMEOUT", 30*time.Second),
			MaxRequests: getEnvInt("GISP_MAX_REQUESTS", 50),
			Enabled:     gispEnabled,
			Priority:    getEnvInt("GISP_PRIORITY", 3),
		}
	}

	return &EnrichmentConfig{
		Enabled:         enabled,
		AutoEnrich:      autoEnrich,
		MinQualityScore: minQualityScore,
		Services:        services,
		Cache: &enrichment.CacheConfig{
			Enabled:         true,
			TTL:             getEnvDuration("ENRICHMENT_CACHE_TTL", 24*time.Hour),
			CleanupInterval: getEnvDuration("ENRICHMENT_CACHE_CLEANUP", 1*time.Hour),
		},
	}
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt получает переменную окружения как int или возвращает значение по умолчанию
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration получает переменную окружения как Duration или возвращает значение по умолчанию
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// WebSearchConfig конфигурация веб-поиска
type WebSearchConfig struct {
	Enabled         bool          `json:"enabled"`
	Timeout         time.Duration `json:"timeout"`
	CacheTTL        time.Duration `json:"cache_ttl"`
	CacheEnabled    bool          `json:"cache_enabled"`
	RateLimitPerSec int           `json:"rate_limit_per_sec"`
	BaseURL         string        `json:"base_url"`
}

// LoadWebSearchConfig загружает конфигурацию веб-поиска
func LoadWebSearchConfig() *WebSearchConfig {
	enabled := getEnv("WEB_SEARCH_ENABLED", "true") == "true"
	timeout := getEnvDuration("WEB_SEARCH_TIMEOUT", 5*time.Second)
	cacheTTL := getEnvDuration("WEB_SEARCH_CACHE_TTL", 24*time.Hour)
	cacheEnabled := getEnv("WEB_SEARCH_CACHE_ENABLED", "true") == "true"
	rateLimit := getEnvInt("WEB_SEARCH_RATE_LIMIT_PER_SEC", 1)
	baseURL := getEnv("WEB_SEARCH_BASE_URL", "https://api.duckduckgo.com")

	return &WebSearchConfig{
		Enabled:         enabled,
		Timeout:         timeout,
		CacheTTL:        cacheTTL,
		CacheEnabled:    cacheEnabled,
		RateLimitPerSec: rateLimit,
		BaseURL:         baseURL,
	}
}

// configJSON структура для сериализации конфигурации в JSON
type configJSON struct {
	Port                       string                     `json:"port"`
	DatabasePath               string                     `json:"database_path"`
	NormalizedDatabasePath     string                     `json:"normalized_database_path"`
	ServiceDatabasePath        string                     `json:"service_database_path"`
	ArliaiAPIKey               string                     `json:"arliai_api_key"`
	ArliaiModel                string                     `json:"arliai_model"`
	MaxOpenConns               int                        `json:"max_open_conns"`
	MaxIdleConns               int                        `json:"max_idle_conns"`
	ConnMaxLifetime            string                     `json:"conn_max_lifetime"` // time.Duration как строка
	LogBufferSize              int                        `json:"log_buffer_size"`
	LogLevel                   string                     `json:"log_level"`
	NormalizerEventsBufferSize int                        `json:"normalizer_events_buffer_size"`
	MultiProviderEnabled       bool                       `json:"multi_provider_enabled"`
	AggregationStrategy        string                     `json:"aggregation_strategy"`
	AITimeout                  string                     `json:"ai_timeout"` // time.Duration как строка
	Enrichment                 *EnrichmentConfig          `json:"enrichment"`
	WebSearch                  *WebSearchConfig           `json:"web_search"`
}

// SaveConfig сохраняет конфигурацию в сервисную БД
func SaveConfig(cfg *Config, serviceDB *database.ServiceDB) error {
	return SaveConfigWithHistory(cfg, serviceDB, "", "")
}

// SaveConfigWithHistory сохраняет конфигурацию в сервисную БД с историей изменений
func SaveConfigWithHistory(cfg *Config, serviceDB *database.ServiceDB, changedBy, changeReason string) error {
	if serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	// Преобразуем Config в configJSON для сериализации
	cfgJSON := &configJSON{
		Port:                       cfg.Port,
		DatabasePath:               cfg.DatabasePath,
		NormalizedDatabasePath:     cfg.NormalizedDatabasePath,
		ServiceDatabasePath:        cfg.ServiceDatabasePath,
		ArliaiAPIKey:               cfg.ArliaiAPIKey,
		ArliaiModel:                cfg.ArliaiModel,
		MaxOpenConns:               cfg.MaxOpenConns,
		MaxIdleConns:               cfg.MaxIdleConns,
		ConnMaxLifetime:            cfg.ConnMaxLifetime.String(),
		LogBufferSize:              cfg.LogBufferSize,
		LogLevel:                   cfg.LogLevel,
		NormalizerEventsBufferSize: cfg.NormalizerEventsBufferSize,
		MultiProviderEnabled:       cfg.MultiProviderEnabled,
		AggregationStrategy:        cfg.AggregationStrategy,
		AITimeout:                  cfg.AITimeout.String(),
		Enrichment:                 cfg.Enrichment,
		WebSearch:                  cfg.WebSearch,
	}

	configJSONBytes, err := json.Marshal(cfgJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := serviceDB.SaveAppConfigWithHistory(string(configJSONBytes), changedBy, changeReason); err != nil {
		return fmt.Errorf("failed to save config to database: %w", err)
	}

	log.Printf("Config saved to service database")
	return nil
}

