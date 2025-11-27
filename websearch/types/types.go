package types

import (
	"context"
	"time"
)

// SearchResult унифицированный результат поиска
type SearchResult struct {
	Query      string       `json:"query"`
	Found      bool         `json:"found"`
	Results    []SearchItem `json:"results"`
	Confidence float64      `json:"confidence"`
	Source     string       `json:"source"` // имя провайдера
	Timestamp  time.Time    `json:"timestamp"`
}

// SearchItem элемент результата поиска
type SearchItem struct {
	Title     string  `json:"title"`
	URL       string  `json:"url"`
	Snippet   string  `json:"snippet"`
	Relevance float64 `json:"relevance"` // релевантность от 0.0 до 1.0
}

// ValidationResult результат валидации через веб-поиск
type ValidationResult struct {
	Status    string                 `json:"status"` // "success", "error", "not_found"
	Message   string                 `json:"message"`
	Score     float64                `json:"score"` // уверенность от 0.0 до 1.0
	Details   map[string]interface{} `json:"details,omitempty"`
	Found     bool                   `json:"found"`
	Results   []SearchItem           `json:"results,omitempty"`
	Provider  string                 `json:"provider,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// SearchProviderInterface интерфейс для провайдеров веб-поиска
// Определен здесь, чтобы избежать циклических импортов
type SearchProviderInterface interface {
	// Search выполняет поиск по запросу
	Search(ctx context.Context, query string) (*SearchResult, error)

	// GetName возвращает имя провайдера
	GetName() string

	// IsAvailable проверяет доступность провайдера
	IsAvailable() bool

	// ValidateCredentials проверяет валидность учетных данных
	ValidateCredentials(ctx context.Context) error

	// GetRateLimit возвращает лимит запросов
	GetRateLimit() time.Duration
}

