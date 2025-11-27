package ai

// ProviderClient интерфейс для всех AI провайдеров
type ProviderClient interface {
	// GetCompletion выполняет запрос к AI и возвращает результат нормализации
	GetCompletion(systemPrompt, userPrompt string) (string, error)
	// GetProviderName возвращает имя провайдера
	GetProviderName() string
	// IsEnabled проверяет, активен ли провайдер
	IsEnabled() bool
}
