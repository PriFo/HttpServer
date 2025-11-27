package websearch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"httpserver/websearch/types"
)

// RouterStrategy стратегия выбора провайдера
type RouterStrategy string

const (
	// StrategyRoundRobin поочередный выбор провайдеров
	StrategyRoundRobin RouterStrategy = "round_robin"
	// StrategyWeighted выбор на основе весов/надежности
	StrategyWeighted RouterStrategy = "weighted"
	// StrategyRandom случайный выбор
	StrategyRandom RouterStrategy = "random"
)

// RouterConfig конфигурация роутера
type RouterConfig struct {
	Strategy RouterStrategy
}

// ProviderRouter роутер для выбора и переключения между провайдерами веб-поиска
// Поддерживает fallback при ошибках и обновление списка провайдеров
type ProviderRouter struct {
	providers          map[string]types.SearchProviderInterface
	reliabilityManager ReliabilityManagerInterface
	config             RouterConfig
	currentIndex       int
	mu                 sync.RWMutex
}

// NewProviderRouter создает новый роутер провайдеров
func NewProviderRouter(
	providers map[string]types.SearchProviderInterface,
	reliabilityManager ReliabilityManagerInterface,
	config RouterConfig,
) *ProviderRouter {
	if config.Strategy == "" {
		config.Strategy = StrategyRoundRobin
	}

	return &ProviderRouter{
		providers:          providers,
		reliabilityManager: reliabilityManager,
		config:             config,
	}
}

// SearchWithFallback выполняет поиск с fallback на другие провайдеры при ошибках
// maxAttempts - максимальное количество попыток с разными провайдерами
func (pr *ProviderRouter) SearchWithFallback(ctx context.Context, query string, maxAttempts int) (*types.SearchResult, error) {
	pr.mu.RLock()
	providers := make([]types.SearchProviderInterface, 0, len(pr.providers))
	for _, p := range pr.providers {
		if p.IsAvailable() {
			providers = append(providers, p)
		}
	}
	pr.mu.RUnlock()

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Выбираем провайдеров на основе стратегии
	selectedProviders := pr.selectProviders(providers, maxAttempts)

	attempts := 0
	var lastErr error

	// Пробуем каждый выбранный провайдер по очереди
	for _, provider := range selectedProviders {
		if attempts >= maxAttempts {
			break
		}

		startTime := time.Now()
		result, err := provider.Search(ctx, query)
		responseTime := time.Since(startTime)

		if err == nil && result != nil {
			// Регистрируем успех
			if pr.reliabilityManager != nil {
				_ = pr.reliabilityManager.RecordSuccessWithTime(provider.GetName(), responseTime)
			}
			return result, nil
		}

		// Регистрируем ошибку
		if pr.reliabilityManager != nil {
			if err != nil {
				_ = pr.reliabilityManager.RecordFailureWithError(provider.GetName(), err)
			} else {
				_ = pr.reliabilityManager.RecordFailure(provider.GetName())
			}
		}

		attempts++
		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("provider %s returned nil result", provider.GetName())
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
	}

	return nil, fmt.Errorf("no available providers")
}

// selectProviders выбирает провайдеров на основе стратегии
func (pr *ProviderRouter) selectProviders(providers []types.SearchProviderInterface, count int) []types.SearchProviderInterface {
	if len(providers) == 0 {
		return nil
	}

	if count > len(providers) {
		count = len(providers)
	}

	switch pr.config.Strategy {
	case StrategyRoundRobin:
		return pr.selectRoundRobin(providers, count)
	case StrategyWeighted:
		return pr.selectWeighted(providers, count)
	case StrategyRandom:
		return pr.selectRandom(providers, count)
	default:
		// Fallback на round-robin
		return pr.selectRoundRobin(providers, count)
	}
}

// selectRoundRobin выбирает провайдеров поочередно
func (pr *ProviderRouter) selectRoundRobin(providers []types.SearchProviderInterface, count int) []types.SearchProviderInterface {
	selected := make([]types.SearchProviderInterface, 0, count)
	pr.mu.Lock()
	startIndex := pr.currentIndex
	pr.currentIndex = (pr.currentIndex + count) % len(providers)
	pr.mu.Unlock()

	for i := 0; i < count; i++ {
		index := (startIndex + i) % len(providers)
		selected = append(selected, providers[index])
	}

	return selected
}

// selectWeighted выбирает провайдеров на основе весов/надежности
func (pr *ProviderRouter) selectWeighted(providers []types.SearchProviderInterface, count int) []types.SearchProviderInterface {
	// Если есть ReliabilityManager, используем его для выбора лучших провайдеров
	if pr.reliabilityManager != nil {
		// Упрощенная версия - сортируем по failure rate
		selected := make([]types.SearchProviderInterface, 0, count)

		// Берем первые count провайдеров с наименьшим failure rate
		for i := 0; i < count && i < len(providers); i++ {
			stats := pr.reliabilityManager.GetStats(providers[i].GetName())
			if stats != nil && stats.FailureRate >= 0.9 {
				// Пропускаем провайдеры с высоким failure rate
				continue
			}
			selected = append(selected, providers[i])
		}

		// Если не набрали достаточно, добавляем остальных
		for len(selected) < count && len(selected) < len(providers) {
			for _, p := range providers {
				found := false
				for _, sp := range selected {
					if sp.GetName() == p.GetName() {
						found = true
						break
					}
				}
				if !found {
					selected = append(selected, p)
					break
				}
			}
		}

		return selected
	}

	// Если нет ReliabilityManager, используем round-robin
	return pr.selectRoundRobin(providers, count)
}

// selectRandom выбирает провайдеров случайно (упрощенная версия - первые count)
func (pr *ProviderRouter) selectRandom(providers []types.SearchProviderInterface, count int) []types.SearchProviderInterface {
	selected := make([]types.SearchProviderInterface, 0, count)
	for i := 0; i < count && i < len(providers); i++ {
		selected = append(selected, providers[i])
	}
	return selected
}

// UpdateProviders обновляет список провайдеров
func (pr *ProviderRouter) UpdateProviders(providers map[string]types.SearchProviderInterface) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.providers = providers
}

// GetProviders возвращает текущий список провайдеров
func (pr *ProviderRouter) GetProviders() map[string]types.SearchProviderInterface {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// Возвращаем копию для безопасности
	result := make(map[string]types.SearchProviderInterface)
	for k, v := range pr.providers {
		result[k] = v
	}
	return result
}
