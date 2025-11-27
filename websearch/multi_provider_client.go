package websearch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"httpserver/websearch/types"
)

// MultiProviderClient клиент для работы с несколькими провайдерами веб-поиска
// Поддерживает роутинг между провайдерами, fallback и агрегацию результатов
type MultiProviderClient struct {
	providers map[string]types.SearchProviderInterface
	router    *ProviderRouter
	cache     *Cache
	timeout   time.Duration
	mu        sync.RWMutex
}

// MultiProviderClientConfig конфигурация для MultiProviderClient
type MultiProviderClientConfig struct {
	Providers map[string]types.SearchProviderInterface
	Router    *ProviderRouter
	Cache     *Cache
	Timeout   time.Duration
}

// NewMultiProviderClient создает новый мульти-провайдерный клиент
func NewMultiProviderClient(config MultiProviderClientConfig) *MultiProviderClient {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &MultiProviderClient{
		providers: config.Providers,
		router:    config.Router,
		cache:     config.Cache,
		timeout:   config.Timeout,
	}
}

// Search выполняет поиск через активные провайдеры с fallback
func (mpc *MultiProviderClient) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	// Проверка кэша
	if mpc.cache != nil {
		cacheKey := generateCacheKey(query)
		if cached, found := mpc.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Используем роутер для выбора провайдера
	if mpc.router != nil {
		result, err := mpc.router.SearchWithFallback(ctx, query, 3) // Максимум 3 попытки
		if err != nil {
			return nil, err
		}

		// Сохраняем в кэш
		if mpc.cache != nil && result != nil {
			cacheKey := generateCacheKey(query)
			mpc.cache.Set(cacheKey, result)
		}

		return result, nil
	}

	// Если роутера нет, используем первый доступный провайдер
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	for _, provider := range mpc.providers {
		if provider.IsAvailable() {
			result, err := provider.Search(ctx, query)
			if err == nil && result != nil {
				// Сохраняем в кэш
				if mpc.cache != nil {
					cacheKey := generateCacheKey(query)
					mpc.cache.Set(cacheKey, result)
				}
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("no available providers")
}

// ReloadProviders обновляет список провайдеров
func (mpc *MultiProviderClient) ReloadProviders(providers map[string]types.SearchProviderInterface) error {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	mpc.providers = providers

	// Обновляем роутер, если он есть
	if mpc.router != nil {
		mpc.router.UpdateProviders(providers)
	}

	return nil
}

// GetStats возвращает статистику по провайдерам
func (mpc *MultiProviderClient) GetStats() map[string]interface{} {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, provider := range mpc.providers {
		stats[name] = map[string]interface{}{
			"available": provider.IsAvailable(),
			"rate_limit": provider.GetRateLimit().String(),
		}
	}

	return stats
}

// GetCacheStats возвращает статистику кэша
func (mpc *MultiProviderClient) GetCacheStats() map[string]interface{} {
	if mpc.cache == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	stats := mpc.cache.GetStats()
	return map[string]interface{}{
		"enabled": true,
		"hits":    stats.Hits,
		"misses":  stats.Misses,
		"size":    stats.Size,
	}
}

// GetActiveProvidersCount возвращает количество активных провайдеров
func (mpc *MultiProviderClient) GetActiveProvidersCount() int {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	count := 0
	for _, provider := range mpc.providers {
		if provider.IsAvailable() {
			count++
		}
	}

	return count
}

