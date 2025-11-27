package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"httpserver/database"
	"httpserver/internal/config"
)

// ConfigHandler обработчик для управления конфигурацией приложения
type ConfigHandler struct {
	serviceDB *database.ServiceDB
}

// NewConfigHandler создает новый обработчик конфигурации
func NewConfigHandler(serviceDB *database.ServiceDB) *ConfigHandler {
	return &ConfigHandler{
		serviceDB: serviceDB,
	}
}

// HandleGetConfig возвращает текущую конфигурацию приложения
func (h *ConfigHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	if h.serviceDB == nil {
		log.Printf("[Config] Service database not available")
		http.Error(w, "Service database not available", http.StatusServiceUnavailable)
		return
	}

	// Загружаем конфигурацию из БД
	cfg, err := config.LoadConfig(h.serviceDB)
	if err != nil {
		log.Printf("[Config] Error loading config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[Config] Configuration retrieved (full)")

	// Возвращаем конфигурацию (без чувствительных данных в ответе, если нужно)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		log.Printf("[Config] Error encoding config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleUpdateConfig обновляет конфигурацию приложения
func (h *ConfigHandler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if h.serviceDB == nil {
		http.Error(w, "Service database not available", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Загружаем текущую конфигурацию для сравнения
	oldCfg, _ := config.LoadConfig(h.serviceDB)

	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		log.Printf("[Config] Error decoding config: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		log.Printf("[Config] Validation failed: %v", err)
		http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Логируем изменения конфигурации
	if oldCfg != nil {
		h.logConfigChanges(oldCfg, &cfg)
	}

	// Получаем информацию о пользователе из заголовков (если есть авторизация)
	changedBy := r.Header.Get("X-User-Id")
	if changedBy == "" {
		changedBy = r.RemoteAddr // Fallback на IP адрес
	}

	// Получаем причину изменения из запроса (опционально)
	changeReason := r.URL.Query().Get("reason")

	// Сохраняем конфигурацию в БД с историей
	if err := config.SaveConfigWithHistory(&cfg, h.serviceDB, changedBy, changeReason); err != nil {
		log.Printf("[Config] Error saving config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[Config] Configuration updated successfully")

	// Возвращаем обновленную конфигурацию
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		log.Printf("[Config] Error encoding config: %v", err)
	}
}

// logConfigChanges логирует изменения в конфигурации
func (h *ConfigHandler) logConfigChanges(oldCfg, newCfg *config.Config) {
	var changes []string

	// Сравниваем основные поля
	if oldCfg.Port != newCfg.Port {
		changes = append(changes, fmt.Sprintf("port: %s -> %s", oldCfg.Port, newCfg.Port))
	}
	if oldCfg.DatabasePath != newCfg.DatabasePath {
		changes = append(changes, fmt.Sprintf("database_path: %s -> %s", oldCfg.DatabasePath, newCfg.DatabasePath))
	}
	if oldCfg.NormalizedDatabasePath != newCfg.NormalizedDatabasePath {
		changes = append(changes, fmt.Sprintf("normalized_database_path: %s -> %s", 
			oldCfg.NormalizedDatabasePath, newCfg.NormalizedDatabasePath))
	}
	if oldCfg.ServiceDatabasePath != newCfg.ServiceDatabasePath {
		changes = append(changes, fmt.Sprintf("service_database_path: %s -> %s", 
			oldCfg.ServiceDatabasePath, newCfg.ServiceDatabasePath))
	}
	if oldCfg.ArliaiModel != newCfg.ArliaiModel {
		changes = append(changes, fmt.Sprintf("arliai_model: %s -> %s", oldCfg.ArliaiModel, newCfg.ArliaiModel))
	}
	if oldCfg.MaxOpenConns != newCfg.MaxOpenConns {
		changes = append(changes, fmt.Sprintf("max_open_conns: %d -> %d", oldCfg.MaxOpenConns, newCfg.MaxOpenConns))
	}
	if oldCfg.MaxIdleConns != newCfg.MaxIdleConns {
		changes = append(changes, fmt.Sprintf("max_idle_conns: %d -> %d", oldCfg.MaxIdleConns, newCfg.MaxIdleConns))
	}
	if oldCfg.ConnMaxLifetime != newCfg.ConnMaxLifetime {
		changes = append(changes, fmt.Sprintf("conn_max_lifetime: %s -> %s", 
			oldCfg.ConnMaxLifetime, newCfg.ConnMaxLifetime))
	}
	if oldCfg.LogBufferSize != newCfg.LogBufferSize {
		changes = append(changes, fmt.Sprintf("log_buffer_size: %d -> %d", oldCfg.LogBufferSize, newCfg.LogBufferSize))
	}
	if oldCfg.LogLevel != newCfg.LogLevel {
		changes = append(changes, fmt.Sprintf("log_level: %s -> %s", oldCfg.LogLevel, newCfg.LogLevel))
	}
	if oldCfg.NormalizerEventsBufferSize != newCfg.NormalizerEventsBufferSize {
		changes = append(changes, fmt.Sprintf("normalizer_events_buffer_size: %d -> %d", 
			oldCfg.NormalizerEventsBufferSize, newCfg.NormalizerEventsBufferSize))
	}
	if oldCfg.MultiProviderEnabled != newCfg.MultiProviderEnabled {
		changes = append(changes, fmt.Sprintf("multi_provider_enabled: %v -> %v", 
			oldCfg.MultiProviderEnabled, newCfg.MultiProviderEnabled))
	}
	if oldCfg.AggregationStrategy != newCfg.AggregationStrategy {
		changes = append(changes, fmt.Sprintf("aggregation_strategy: %s -> %s", 
			oldCfg.AggregationStrategy, newCfg.AggregationStrategy))
	}
	if oldCfg.AITimeout != newCfg.AITimeout {
		changes = append(changes, fmt.Sprintf("ai_timeout: %s -> %s", oldCfg.AITimeout, newCfg.AITimeout))
	}

	// Проверяем изменения API ключа (только факт изменения, не значение)
	if oldCfg.ArliaiAPIKey != newCfg.ArliaiAPIKey {
		oldHasKey := oldCfg.ArliaiAPIKey != ""
		newHasKey := newCfg.ArliaiAPIKey != ""
		if oldHasKey != newHasKey {
			changes = append(changes, fmt.Sprintf("arliai_api_key: %v -> %v", oldHasKey, newHasKey))
		} else {
			changes = append(changes, "arliai_api_key: [changed]")
		}
	}

	// Проверяем изменения в Enrichment
	if oldCfg.Enrichment != nil && newCfg.Enrichment != nil {
		if oldCfg.Enrichment.Enabled != newCfg.Enrichment.Enabled {
			changes = append(changes, fmt.Sprintf("enrichment.enabled: %v -> %v", 
				oldCfg.Enrichment.Enabled, newCfg.Enrichment.Enabled))
		}
		if oldCfg.Enrichment.AutoEnrich != newCfg.Enrichment.AutoEnrich {
			changes = append(changes, fmt.Sprintf("enrichment.auto_enrich: %v -> %v", 
				oldCfg.Enrichment.AutoEnrich, newCfg.Enrichment.AutoEnrich))
		}
		if oldCfg.Enrichment.MinQualityScore != newCfg.Enrichment.MinQualityScore {
			changes = append(changes, fmt.Sprintf("enrichment.min_quality_score: %.2f -> %.2f", 
				oldCfg.Enrichment.MinQualityScore, newCfg.Enrichment.MinQualityScore))
		}
	}

	// Проверяем изменения в WebSearch
	if oldCfg.WebSearch != nil && newCfg.WebSearch != nil {
		if oldCfg.WebSearch.Enabled != newCfg.WebSearch.Enabled {
			changes = append(changes, fmt.Sprintf("web_search.enabled: %v -> %v", 
				oldCfg.WebSearch.Enabled, newCfg.WebSearch.Enabled))
		}
		if oldCfg.WebSearch.Timeout != newCfg.WebSearch.Timeout {
			changes = append(changes, fmt.Sprintf("web_search.timeout: %s -> %s", 
				oldCfg.WebSearch.Timeout, newCfg.WebSearch.Timeout))
		}
	}

	if len(changes) > 0 {
		log.Printf("[Config] Configuration changes detected: %s", strings.Join(changes, ", "))
	} else {
		log.Printf("[Config] Configuration update requested but no changes detected")
	}
}

// HandleGetConfigSafe возвращает конфигурацию без чувствительных данных (API ключи)
func (h *ConfigHandler) HandleGetConfigSafe(w http.ResponseWriter, r *http.Request) {
	if h.serviceDB == nil {
		log.Printf("[Config] Service database not available")
		http.Error(w, "Service database not available", http.StatusServiceUnavailable)
		return
	}

	// Загружаем конфигурацию из БД
	cfg, err := config.LoadConfig(h.serviceDB)
	if err != nil {
		log.Printf("[Config] Error loading config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Создаем безопасную копию без API ключей
	safeConfig := struct {
		Port                       string                     `json:"port"`
		DatabasePath               string                     `json:"database_path"`
		NormalizedDatabasePath     string                     `json:"normalized_database_path"`
		ServiceDatabasePath        string                     `json:"service_database_path"`
		ArliaiModel                string                     `json:"arliai_model"`
		MaxOpenConns               int                        `json:"max_open_conns"`
		MaxIdleConns               int                        `json:"max_idle_conns"`
		ConnMaxLifetime            string                     `json:"conn_max_lifetime"`
		LogBufferSize               int                        `json:"log_buffer_size"`
		LogLevel                    string                     `json:"log_level"`
		NormalizerEventsBufferSize int                        `json:"normalizer_events_buffer_size"`
		MultiProviderEnabled       bool                       `json:"multi_provider_enabled"`
		AggregationStrategy        string                     `json:"aggregation_strategy"`
		AITimeout                  string                     `json:"ai_timeout"`
		Enrichment                 *config.EnrichmentConfig   `json:"enrichment"`
		WebSearch                  *config.WebSearchConfig    `json:"web_search"`
		HasArliaiAPIKey            bool                       `json:"has_arliai_api_key"`
	}{
		Port:                       cfg.Port,
		DatabasePath:               cfg.DatabasePath,
		NormalizedDatabasePath:     cfg.NormalizedDatabasePath,
		ServiceDatabasePath:        cfg.ServiceDatabasePath,
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
		HasArliaiAPIKey:            cfg.ArliaiAPIKey != "",
	}

	// Скрываем API ключи в Enrichment
	if safeConfig.Enrichment != nil && safeConfig.Enrichment.Services != nil {
		for _, service := range safeConfig.Enrichment.Services {
			if service != nil {
				hasKey := service.APIKey != ""
				service.APIKey = ""
				// Можно добавить поле has_api_key если нужно
				_ = hasKey
			}
		}
	}

	log.Printf("[Config] Configuration retrieved (safe)")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(safeConfig); err != nil {
		log.Printf("[Config] Error encoding safe config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleGetConfigHistory возвращает историю изменений конфигурации
func (h *ConfigHandler) HandleGetConfigHistory(w http.ResponseWriter, r *http.Request) {
	if h.serviceDB == nil {
		log.Printf("[Config] Service database not available")
		http.Error(w, "Service database not available", http.StatusServiceUnavailable)
		return
	}

	// Получаем лимит из query параметра
	limit := 10 // По умолчанию
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || parsedLimit != 1 {
			limit = 10
		}
	}

	history, err := h.serviceDB.GetAppConfigHistory(limit)
	if err != nil {
		log.Printf("[Config] Error loading config history: %v", err)
		http.Error(w, fmt.Sprintf("Failed to load config history: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем текущую версию
	currentVersion, err := h.serviceDB.GetAppConfigVersion()
	if err != nil {
		log.Printf("[Config] Warning: failed to get current version: %v", err)
	}

	response := map[string]interface{}{
		"current_version": currentVersion,
		"history":         history,
		"count":           len(history),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[Config] Error encoding config history: %v", err)
		http.Error(w, fmt.Sprintf("Failed to encode config history: %v", err), http.StatusInternalServerError)
		return
	}
}

