package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"

	"httpserver/websearch/types"
)

// BingProvider провайдер для Bing Search API
type BingProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	rateLimit  time.Duration
	available  bool
}

// NewBingProvider создает новый провайдер Bing
func NewBingProvider(apiKey string, timeout time.Duration, rateLimit time.Duration) *BingProvider {
	limiter := rate.NewLimiter(rate.Every(rateLimit), 1)

	return &BingProvider{
		apiKey:  apiKey,
		baseURL: "https://api.bing.microsoft.com/v7.0/search",
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter:   limiter,
		rateLimit: rateLimit,
		available: apiKey != "",
	}
}

// GetName возвращает имя провайдера
func (b *BingProvider) GetName() string {
	return "bing"
}

// IsAvailable проверяет доступность провайдера
func (b *BingProvider) IsAvailable() bool {
	return b.available && b.apiKey != ""
}

// ValidateCredentials проверяет валидность учетных данных
func (b *BingProvider) ValidateCredentials(ctx context.Context) error {
	if b.apiKey == "" {
		return fmt.Errorf("API key is required for Bing")
	}
	// Можно добавить тестовый запрос для проверки
	return nil
}

// Search выполняет поиск через Bing Search API
func (b *BingProvider) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	if !b.IsAvailable() {
		return nil, fmt.Errorf("provider is not available")
	}

	if err := b.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	params := url.Values{}
	params.Add("q", query)
	params.Add("count", "10")

	fullURL := fmt.Sprintf("%s?%s", b.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Ocp-Apim-Subscription-Key", b.apiKey)
	req.Header.Set("User-Agent", "HttpServer/1.0")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		b.available = false
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		b.available = false
		return nil, fmt.Errorf("authentication failed: invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		b.available = false
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var bingResponse BingResponse
	if err := json.NewDecoder(resp.Body).Decode(&bingResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	b.available = true
	return b.transformResults(query, &bingResponse), nil
}

// BingResponse структура ответа Bing Search API
type BingResponse struct {
	WebPages struct {
		Value []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Snippet     string `json:"snippet"`
			DateLastCrawled string `json:"dateLastCrawled"`
		} `json:"value"`
		TotalEstimatedMatches int `json:"totalEstimatedMatches"`
	} `json:"webPages"`
}

// transformResults преобразует ответ Bing в унифицированный формат
func (b *BingProvider) transformResults(query string, resp *BingResponse) *types.SearchResult {
	results := make([]types.SearchItem, 0, len(resp.WebPages.Value))

	for i, item := range resp.WebPages.Value {
		relevance := 1.0 - (float64(i) * 0.1) // Уменьшаем релевантность для последующих результатов
		if relevance < 0.3 {
			relevance = 0.3
		}

		results = append(results, types.SearchItem{
			Title:     item.Name,
			URL:       item.URL,
			Snippet:   item.Snippet,
			Relevance: relevance,
		})
	}

	confidence := 0.0
	if len(results) > 0 {
		confidence = 0.85 // Высокая уверенность для Bing
		if len(results) >= 5 {
			confidence = 0.95 // Очень высокая, если много результатов
		}
	}

	return &types.SearchResult{
		Query:      query,
		Found:      len(results) > 0,
		Results:    results,
		Confidence: confidence,
		Source:     b.GetName(),
		Timestamp:  time.Now(),
	}
}

// GetRateLimit возвращает лимит запросов
func (b *BingProvider) GetRateLimit() time.Duration {
	return b.rateLimit
}

