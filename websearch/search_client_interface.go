package websearch

import (
	"context"

	"httpserver/websearch/types"
)

// SearchClientInterface интерфейс для клиентов веб-поиска
// Поддерживает как простой Client, так и MultiProviderClient
type SearchClientInterface interface {
	// Search выполняет поиск по запросу
	Search(ctx context.Context, query string) (*types.SearchResult, error)
}

// Проверка, что типы реализуют интерфейс
var _ SearchClientInterface = (*Client)(nil)
var _ SearchClientInterface = (*MultiProviderClient)(nil)
