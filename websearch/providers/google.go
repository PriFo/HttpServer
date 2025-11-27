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

// GoogleProvider провайдер для Google Custom Search API
type GoogleProvider struct {
	apiKey     string
	searchID   string
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	rateLimit  time.Duration
	available  bool
}

// NewGoogleProvider создает новый провайдер Google
func NewGoogleProvider(apiKey, searchID string, timeout time.Duration, rateLimit time.Duration) *GoogleProvider {
	limiter := rate.NewLimiter(rate.Every(rateLimit), 1)

	return &GoogleProvider{
		apiKey:    apiKey,
		searchID:  searchID,
		baseURL:   "https://www.googleapis.com/customsearch/v1",
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter:   limiter,
		rateLimit: rateLimit,
		available: apiKey != "" && searchID != "",
	}
}

// GetName возвращает имя провайдера
func (g *GoogleProvider) GetName() string {
	return "google"
}

// IsAvailable проверяет доступность провайдера
func (g *GoogleProvider) IsAvailable() bool {
	return g.available && g.apiKey != "" && g.searchID != ""
}

// ValidateCredentials проверяет валидность учетных данных
func (g *GoogleProvider) ValidateCredentials(ctx context.Context) error {
	if g.apiKey == "" {
		return fmt.Errorf("API key is required for Google")
	}
	if g.searchID == "" {
		return fmt.Errorf("Search ID is required for Google")
	}
	return nil
}

// Search выполняет поиск через Google Custom Search API
func (g *GoogleProvider) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	if !g.IsAvailable() {
		return nil, fmt.Errorf("provider is not available")
	}

	if err := g.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	params := url.Values{}
	params.Add("key", g.apiKey)
	params.Add("cx", g.searchID)
	params.Add("q", query)
	params.Add("num", "10")

	fullURL := fmt.Sprintf("%s?%s", g.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "HttpServer/1.0")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		g.available = false
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Обработка различных HTTP статусов
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		g.available = false
		return nil, fmt.Errorf("authentication failed: invalid API key or search ID")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		g.available = false
		return nil, fmt.Errorf("rate limit exceeded: too many requests to Google API")
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		g.available = false
		return nil, fmt.Errorf("service unavailable: Google API is temporarily down")
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp GoogleErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error.Message != "" {
			g.available = false
			return nil, fmt.Errorf("Google API error (code %d): %s", errorResp.Error.Code, errorResp.Error.Message)
		}
		g.available = false
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, resp.Status)
	}

	var googleResponse GoogleResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	g.available = true
	return g.transformResults(query, &googleResponse), nil
}

// GoogleResponse структура ответа Google Custom Search API
type GoogleResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
	SearchInformation struct {
		TotalResults string `json:"totalResults"`
	} `json:"searchInformation"`
}

// GoogleErrorResponse структура ошибки Google API
type GoogleErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// transformResults преобразует ответ Google в унифицированный формат
func (g *GoogleProvider) transformResults(query string, resp *GoogleResponse) *types.SearchResult {
	results := make([]types.SearchItem, 0, len(resp.Items))

	for i, item := range resp.Items {
		relevance := 1.0 - (float64(i) * 0.1) // Уменьшаем релевантность для последующих результатов
		if relevance < 0.3 {
			relevance = 0.3
		}

		results = append(results, types.SearchItem{
			Title:     item.Title,
			URL:       item.Link,
			Snippet:   item.Snippet,
			Relevance: relevance,
		})
	}

	confidence := 0.0
	if len(results) > 0 {
		confidence = 0.9 // Очень высокая уверенность для Google
		if len(results) >= 5 {
			confidence = 0.95
		}
	}

	return &types.SearchResult{
		Query:      query,
		Found:      len(results) > 0,
		Results:    results,
		Confidence: confidence,
		Source:     g.GetName(),
		Timestamp:  time.Now(),
	}
}

// GetRateLimit возвращает лимит запросов
func (g *GoogleProvider) GetRateLimit() time.Duration {
	return g.rateLimit
}

