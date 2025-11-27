package providers

import (
	"context"
	"fmt"
	"time"

	"httpserver/websearch/types"
)

// YandexProvider провайдер для Yandex XML Search API
type YandexProvider struct {
	apiKey    string
	user      string
	baseURL   string
	rateLimit time.Duration
	available bool
}

// NewYandexProvider создает новый провайдер Yandex
func NewYandexProvider(apiKey, user string, timeout time.Duration, rateLimit time.Duration) *YandexProvider {
	return &YandexProvider{
		apiKey:    apiKey,
		user:      user,
		baseURL:   "https://yandex.com/search/xml",
		rateLimit: rateLimit,
		available: apiKey != "" && user != "",
	}
}

// GetName возвращает имя провайдера
func (y *YandexProvider) GetName() string {
	return "yandex"
}

// IsAvailable проверяет доступность провайдера
func (y *YandexProvider) IsAvailable() bool {
	return y.available && y.apiKey != "" && y.user != ""
}

// ValidateCredentials проверяет валидность учетных данных
func (y *YandexProvider) ValidateCredentials(ctx context.Context) error {
	if y.apiKey == "" {
		return fmt.Errorf("API key is required for Yandex")
	}
	if y.user == "" {
		return fmt.Errorf("User is required for Yandex")
	}
	// Можно добавить тестовый запрос для проверки
	return nil
}

// Search выполняет поиск через Yandex XML Search API
// Примечание: Yandex XML Search API требует платной подписки и API ключа
// Для использования необходимо:
// 1. Получить API ключ на https://yandex.ru/dev/xml/
// 2. Указать ключ и пользователя в конфигурации
// 3. Реализовать XML-парсинг ответа от API
func (y *YandexProvider) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	if !y.IsAvailable() {
		return nil, fmt.Errorf("Yandex provider is not available: API key or user not configured")
	}

	// Yandex XML Search API требует реализации XML-парсинга
	// Временная заглушка: возвращаем пустой результат с информативным сообщением
	// Это позволяет системе работать, но Yandex не будет использоваться для поиска
	return &types.SearchResult{
		Query:      query,
		Found:      false,
		Results:    []types.SearchItem{},
		Confidence: 0.0,
		Source:     y.GetName(),
		Timestamp:  time.Now(),
	}, fmt.Errorf("Yandex XML Search API integration not implemented. To use Yandex, implement XML parsing for https://yandex.com/search/xml API")
}

// GetRateLimit возвращает лимит запросов
func (y *YandexProvider) GetRateLimit() time.Duration {
	return y.rateLimit
}
