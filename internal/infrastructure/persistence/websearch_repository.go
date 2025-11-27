package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"httpserver/database"
)

// WebSearchProvider модель провайдера веб-поиска
type WebSearchProvider struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Enabled          bool      `json:"enabled"`
	APIKey           string    `json:"api_key,omitempty"`
	SearchID         string    `json:"search_id,omitempty"`
	User             string    `json:"user,omitempty"`
	BaseURL          string    `json:"base_url"`
	RateLimitSeconds int       `json:"rate_limit_seconds"`
	Priority         int       `json:"priority"`
	Region           string    `json:"region"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ProviderStats статистика провайдера
type ProviderStats struct {
	ProviderName      string    `json:"provider_name"`
	RequestsTotal     int64     `json:"requests_total"`
	RequestsSuccess   int64     `json:"requests_success"`
	RequestsFailed    int64     `json:"requests_failed"`
	FailureRate       float64   `json:"failure_rate"`
	AvgResponseTimeMs int64     `json:"avg_response_time_ms"`
	LastSuccess       *time.Time `json:"last_success,omitempty"`
	LastFailure       *time.Time `json:"last_failure,omitempty"`
	LastError         string    `json:"last_error"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WebSearchRepository репозиторий для работы с веб-поиском
type WebSearchRepository struct {
	serviceDB *database.ServiceDB
}

// NewWebSearchRepository создает новый репозиторий веб-поиска
func NewWebSearchRepository(serviceDB *database.ServiceDB) *WebSearchRepository {
	return &WebSearchRepository{
		serviceDB: serviceDB,
	}
}

// GetAllProviders возвращает всех провайдеров
func (r *WebSearchRepository) GetAllProviders() ([]WebSearchProvider, error) {
	query := `SELECT id, name, enabled, api_key, search_id, user, base_url, 
	                 rate_limit_seconds, priority, region, created_at, updated_at 
	          FROM websearch_providers 
	          ORDER BY priority DESC, name`

	rows, err := r.serviceDB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query providers: %w", err)
	}
	defer rows.Close()

	var providers []WebSearchProvider
	for rows.Next() {
		var p WebSearchProvider
		err := rows.Scan(
			&p.ID, &p.Name, &p.Enabled, &p.APIKey, &p.SearchID, &p.User,
			&p.BaseURL, &p.RateLimitSeconds, &p.Priority, &p.Region,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, p)
	}

	return providers, nil
}

// GetEnabledProviders возвращает только включенные провайдеры
func (r *WebSearchRepository) GetEnabledProviders() ([]WebSearchProvider, error) {
	query := `SELECT id, name, enabled, api_key, search_id, user, base_url, 
	                 rate_limit_seconds, priority, region, created_at, updated_at 
	          FROM websearch_providers 
	          WHERE enabled = TRUE 
	          ORDER BY priority DESC, name`

	rows, err := r.serviceDB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled providers: %w", err)
	}
	defer rows.Close()

	var providers []WebSearchProvider
	for rows.Next() {
		var p WebSearchProvider
		err := rows.Scan(
			&p.ID, &p.Name, &p.Enabled, &p.APIKey, &p.SearchID, &p.User,
			&p.BaseURL, &p.RateLimitSeconds, &p.Priority, &p.Region,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, p)
	}

	return providers, nil
}

// GetProviderByName возвращает провайдера по имени
func (r *WebSearchRepository) GetProviderByName(name string) (*WebSearchProvider, error) {
	query := `SELECT id, name, enabled, api_key, search_id, user, base_url, 
	                 rate_limit_seconds, priority, region, created_at, updated_at 
	          FROM websearch_providers 
	          WHERE name = ?`

	var p WebSearchProvider
	err := r.serviceDB.QueryRow(query, name).Scan(
		&p.ID, &p.Name, &p.Enabled, &p.APIKey, &p.SearchID, &p.User,
		&p.BaseURL, &p.RateLimitSeconds, &p.Priority, &p.Region,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return &p, nil
}

// UpdateProvider обновляет провайдера
func (r *WebSearchRepository) UpdateProvider(provider *WebSearchProvider) error {
	query := `UPDATE websearch_providers 
	          SET enabled = ?, api_key = ?, search_id = ?, user = ?, base_url = ?,
	              rate_limit_seconds = ?, priority = ?, region = ?, updated_at = CURRENT_TIMESTAMP
	          WHERE name = ?`

	result, err := r.serviceDB.Exec(
		query,
		provider.Enabled, provider.APIKey, provider.SearchID, provider.User,
		provider.BaseURL, provider.RateLimitSeconds, provider.Priority,
		provider.Region, provider.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("provider not found: %s", provider.Name)
	}

	return nil
}

// CreateProvider создает нового провайдера
func (r *WebSearchRepository) CreateProvider(provider *WebSearchProvider) error {
	query := `INSERT INTO websearch_providers 
	          (name, enabled, api_key, search_id, user, base_url, rate_limit_seconds, priority, region)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.serviceDB.Exec(
		query,
		provider.Name, provider.Enabled, provider.APIKey, provider.SearchID,
		provider.User, provider.BaseURL, provider.RateLimitSeconds,
		provider.Priority, provider.Region,
	)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	return nil
}

// GetProviderStats возвращает статистику провайдера
func (r *WebSearchRepository) GetProviderStats(providerName string) (*ProviderStats, error) {
	query := `SELECT provider_name, requests_total, requests_success, requests_failed,
	                 failure_rate, avg_response_time_ms, last_success, last_failure,
	                 last_error, updated_at
	          FROM websearch_provider_stats 
	          WHERE provider_name = ?`

	var stats ProviderStats
	var lastSuccess, lastFailure sql.NullTime

	err := r.serviceDB.QueryRow(query, providerName).Scan(
		&stats.ProviderName, &stats.RequestsTotal, &stats.RequestsSuccess,
		&stats.RequestsFailed, &stats.FailureRate, &stats.AvgResponseTimeMs,
		&lastSuccess, &lastFailure, &stats.LastError, &stats.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return &ProviderStats{
			ProviderName: providerName,
			UpdatedAt:    time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider stats: %w", err)
	}

	if lastSuccess.Valid {
		stats.LastSuccess = &lastSuccess.Time
	}
	if lastFailure.Valid {
		stats.LastFailure = &lastFailure.Time
	}

	return &stats, nil
}

// UpdateProviderStats обновляет статистику провайдера
func (r *WebSearchRepository) UpdateProviderStats(stats *ProviderStats) error {
	query := `INSERT OR REPLACE INTO websearch_provider_stats 
	          (provider_name, requests_total, requests_success, requests_failed,
	           failure_rate, avg_response_time_ms, last_success, last_failure,
	           last_error, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	var lastSuccess, lastFailure interface{}
	if stats.LastSuccess != nil {
		lastSuccess = stats.LastSuccess
	}
	if stats.LastFailure != nil {
		lastFailure = stats.LastFailure
	}

	_, err := r.serviceDB.Exec(
		query,
		stats.ProviderName, stats.RequestsTotal, stats.RequestsSuccess,
		stats.RequestsFailed, stats.FailureRate, stats.AvgResponseTimeMs,
		lastSuccess, lastFailure, stats.LastError,
	)
	if err != nil {
		return fmt.Errorf("failed to update provider stats: %w", err)
	}

	return nil
}

// GetWebSearchRules возвращает правила веб-поиска из конфигурации нормализации
func (r *WebSearchRepository) GetWebSearchRules() (map[string]interface{}, error) {
	query := `SELECT websearch_rules FROM normalization_config LIMIT 1`

	var rulesJSON sql.NullString
	err := r.serviceDB.QueryRow(query).Scan(&rulesJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to get websearch rules: %w", err)
	}

	if !rulesJSON.Valid || rulesJSON.String == "" {
		return make(map[string]interface{}), nil
	}

	var rules map[string]interface{}
	if err := json.Unmarshal([]byte(rulesJSON.String), &rules); err != nil {
		return make(map[string]interface{}), nil
	}

	return rules, nil
}

// UpdateWebSearchRules обновляет правила веб-поиска
func (r *WebSearchRepository) UpdateWebSearchRules(rules map[string]interface{}) error {
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	query := `UPDATE normalization_config SET websearch_rules = ?`
	_, err = r.serviceDB.Exec(query, string(rulesJSON))
	if err != nil {
		return fmt.Errorf("failed to update websearch rules: %w", err)
	}

	return nil
}

