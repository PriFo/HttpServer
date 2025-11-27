package database

import (
	"database/sql"
	"fmt"
)

// InitWebSearchSchema создает таблицы для веб-поиска в service.db
func InitWebSearchSchema(db *sql.DB) error {
	// Таблица конфигурации провайдеров
	createProvidersTable := `
	CREATE TABLE IF NOT EXISTS websearch_providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		enabled BOOLEAN NOT NULL DEFAULT FALSE,
		api_key TEXT,
		search_id TEXT,
		user TEXT,
		base_url TEXT,
		rate_limit_seconds INTEGER DEFAULT 1,
		priority INTEGER DEFAULT 1,
		region TEXT DEFAULT 'global',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	// Таблица статистики надежности
	createStatsTable := `
	CREATE TABLE IF NOT EXISTS websearch_provider_stats (
		provider_name TEXT PRIMARY KEY,
		requests_total INTEGER DEFAULT 0,
		requests_success INTEGER DEFAULT 0,
		requests_failed INTEGER DEFAULT 0,
		failure_rate REAL DEFAULT 0.0,
		avg_response_time_ms INTEGER DEFAULT 0,
		last_success DATETIME,
		last_failure DATETIME,
		last_error TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	// Попытка добавить поле websearch_rules в normalization_config
	alterNormalizationConfig := `
	ALTER TABLE normalization_config 
	ADD COLUMN websearch_rules TEXT DEFAULT '{}'`

	// Создаем таблицы
	if _, err := db.Exec(createProvidersTable); err != nil {
		return fmt.Errorf("failed to create websearch_providers table: %w", err)
	}

	if _, err := db.Exec(createStatsTable); err != nil {
		return fmt.Errorf("failed to create websearch_provider_stats table: %w", err)
	}

	// Пытаемся добавить поле в normalization_config (игнорируем ошибку если уже существует)
	_, _ = db.Exec(alterNormalizationConfig)

	// Вставляем начальные данные для провайдеров
	initialProviders := []struct {
		name    string
		enabled bool
		priority int
	}{
		{"duckduckgo", true, 1},  // DuckDuckGo включен по умолчанию (бесплатный)
		{"bing", false, 2},
		{"google", false, 3},
		{"yandex", false, 4},
	}

	for _, provider := range initialProviders {
		_, err := db.Exec(`
			INSERT OR IGNORE INTO websearch_providers (name, enabled, priority, rate_limit_seconds)
			VALUES (?, ?, ?, ?)
		`, provider.name, provider.enabled, provider.priority, 1)
		if err != nil {
			return fmt.Errorf("failed to insert initial provider %s: %w", provider.name, err)
		}
	}

	// Создаем индексы
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_websearch_providers_enabled ON websearch_providers(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_websearch_providers_priority ON websearch_providers(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_websearch_provider_stats_name ON websearch_provider_stats(provider_name)`,
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			// Игнорируем ошибки создания индекса, если он уже существует
			continue
		}
	}

	return nil
}
