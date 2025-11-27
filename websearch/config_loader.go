package websearch

import (
	"fmt"

	"httpserver/internal/infrastructure/persistence"
)

// ConfigLoader загружает конфигурацию провайдеров из базы данных
type ConfigLoader struct {
	repo *persistence.WebSearchRepository
}

// NewConfigLoader создает новый загрузчик конфигурации
func NewConfigLoader(repo *persistence.WebSearchRepository) *ConfigLoader {
	return &ConfigLoader{
		repo: repo,
	}
}

// LoadEnabledProviders загружает включенные провайдеры из БД
func (cl *ConfigLoader) LoadEnabledProviders() ([]ProviderConfigDB, error) {
	if cl.repo == nil {
		return nil, fmt.Errorf("repository is nil")
	}

	providers, err := cl.repo.GetEnabledProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to load enabled providers: %w", err)
	}

	configs := make([]ProviderConfigDB, 0, len(providers))
	for _, p := range providers {
		configs = append(configs, ProviderConfigDB{
			Name:             p.Name,
			Enabled:          p.Enabled,
			APIKey:           p.APIKey,
			SearchID:         p.SearchID,
			User:             p.User,
			BaseURL:          p.BaseURL,
			RateLimitSeconds: p.RateLimitSeconds,
			Priority:         p.Priority,
			Region:           p.Region,
		})
	}

	return configs, nil
}

// LoadAllProviders загружает всех провайдеров из БД
func (cl *ConfigLoader) LoadAllProviders() ([]ProviderConfigDB, error) {
	if cl.repo == nil {
		return nil, fmt.Errorf("repository is nil")
	}

	providers, err := cl.repo.GetAllProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to load providers: %w", err)
	}

	configs := make([]ProviderConfigDB, 0, len(providers))
	for _, p := range providers {
		configs = append(configs, ProviderConfigDB{
			Name:             p.Name,
			Enabled:          p.Enabled,
			APIKey:           p.APIKey,
			SearchID:         p.SearchID,
			User:             p.User,
			BaseURL:          p.BaseURL,
			RateLimitSeconds: p.RateLimitSeconds,
			Priority:         p.Priority,
			Region:           p.Region,
		})
	}

	return configs, nil
}
