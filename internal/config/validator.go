package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"httpserver/enrichment"
)

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	var errors []string

	// Валидация порта
	if c.Port == "" {
		errors = append(errors, "port is required")
	} else {
		port, err := strconv.Atoi(c.Port)
		if err != nil {
			errors = append(errors, fmt.Sprintf("invalid port: %s", c.Port))
		} else if port < 1 || port > 65535 {
			errors = append(errors, fmt.Sprintf("port must be between 1 and 65535, got %d", port))
		}
	}

	// Валидация путей к базам данных
	if c.DatabasePath == "" {
		errors = append(errors, "database path is required")
	}
	if c.NormalizedDatabasePath == "" {
		errors = append(errors, "normalized database path is required")
	}
	if c.ServiceDatabasePath == "" {
		errors = append(errors, "service database path is required")
	}

	// Валидация connection pooling
	if c.MaxOpenConns < 1 {
		errors = append(errors, "max open connections must be at least 1")
	}
	if c.MaxIdleConns < 1 {
		errors = append(errors, "max idle connections must be at least 1")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		errors = append(errors, "max idle connections cannot be greater than max open connections")
	}
	if c.ConnMaxLifetime < time.Second {
		errors = append(errors, "connection max lifetime must be at least 1 second")
	}

	// Валидация буферов
	if c.LogBufferSize < 1 {
		errors = append(errors, "log buffer size must be at least 1")
	}
	if c.NormalizerEventsBufferSize < 1 {
		errors = append(errors, "normalizer events buffer size must be at least 1")
	}

	// Валидация уровня логирования
	validLogLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	if c.LogLevel != "" {
		valid := false
		logLevelUpper := strings.ToUpper(c.LogLevel)
		for _, level := range validLogLevels {
			if logLevelUpper == level {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("invalid log level: %s (valid: %s)", 
				c.LogLevel, strings.Join(validLogLevels, ", ")))
		}
	}

	// Валидация AI конфигурации
	if c.ArliaiModel == "" {
		errors = append(errors, "arliai model is required")
	}

	// Валидация таймаутов
	if c.AITimeout < time.Second {
		errors = append(errors, "AI timeout must be at least 1 second")
	}

	// Валидация стратегии агрегации
	validStrategies := []string{"first_success", "best_confidence", "majority", "weighted_average"}
	if c.AggregationStrategy != "" {
		valid := false
		for _, strategy := range validStrategies {
			if c.AggregationStrategy == strategy {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("invalid aggregation strategy: %s (valid: %s)", 
				c.AggregationStrategy, strings.Join(validStrategies, ", ")))
		}
	}

	// Валидация конфигурации обогащения
	if c.Enrichment != nil {
		if err := c.Enrichment.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("enrichment config: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate проверяет корректность конфигурации обогащения
func (ec *EnrichmentConfig) Validate() error {
	var errors []string

	// Валидация минимального качества
	if ec.MinQualityScore < 0 || ec.MinQualityScore > 1 {
		errors = append(errors, "min quality score must be between 0 and 1")
	}

	// Валидация сервисов
	for name, service := range ec.Services {
		if service == nil {
			errors = append(errors, fmt.Sprintf("service %s is nil", name))
			continue
		}

		if service.Timeout < time.Second {
			errors = append(errors, fmt.Sprintf("service %s timeout must be at least 1 second", name))
		}

		if service.MaxRequests < 1 {
			errors = append(errors, fmt.Sprintf("service %s max requests must be at least 1", name))
		}

		if service.Priority < 1 {
			errors = append(errors, fmt.Sprintf("service %s priority must be at least 1", name))
		}
	}

	// Валидация кэша
	if ec.Cache != nil {
		if ec.Cache.TTL < time.Minute {
			errors = append(errors, "cache TTL must be at least 1 minute")
		}
		if ec.Cache.CleanupInterval < time.Minute {
			errors = append(errors, "cache cleanup interval must be at least 1 minute")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("enrichment validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetDefaults возвращает конфигурацию со значениями по умолчанию
func GetDefaults() *Config {
	return &Config{
		Port:                       "9999",
		DatabasePath:               "data.db",
		NormalizedDatabasePath:     "normalized_data.db",
		ServiceDatabasePath:        "service.db",
		ArliaiModel:                "GLM-4.5-Air",
		MaxOpenConns:               25,
		MaxIdleConns:               5,
		ConnMaxLifetime:            5 * time.Minute,
		LogBufferSize:              100,
		NormalizerEventsBufferSize: 100,
		MultiProviderEnabled:       false,
		AggregationStrategy:        "first_success",
		AITimeout:                  30 * time.Second,
		Enrichment:                 GetDefaultEnrichmentConfig(),
	}
}

// GetDefaultEnrichmentConfig возвращает конфигурацию обогащения со значениями по умолчанию
func GetDefaultEnrichmentConfig() *EnrichmentConfig {
	return &EnrichmentConfig{
		Enabled:         true,
		AutoEnrich:      true,
		MinQualityScore: 0.3,
		Services:        make(map[string]*enrichment.EnricherConfig),
		Cache: &enrichment.CacheConfig{
			Enabled:         true,
			TTL:             24 * time.Hour,
			CleanupInterval: 1 * time.Hour,
		},
	}
}

