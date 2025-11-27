package websearch

import (
	"httpserver/websearch/types"
)

// Реэкспорт типов из websearch/types для обратной совместимости
type SearchResult = types.SearchResult
type SearchItem = types.SearchItem
type ValidationResult = types.ValidationResult
type SearchProviderInterface = types.SearchProviderInterface

// DuckDuckGoResponse ответ от DuckDuckGo API
type DuckDuckGoResponse struct {
	Abstract       string        `json:"Abstract"`
	AbstractText   string        `json:"AbstractText"`
	AbstractSource string        `json:"AbstractSource"`
	AbstractURL    string        `json:"AbstractURL"`
	RelatedTopics  []RelatedTopic `json:"RelatedTopics"`
	Results        []Result      `json:"Results"`
}

// RelatedTopic связанная тема
type RelatedTopic struct {
	Text     string `json:"Text"`
	FirstURL string `json:"FirstURL"`
}

// Result результат поиска
type Result struct {
	FirstURL string `json:"FirstURL"`
	Text     string `json:"Text"`
}
