package websearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"httpserver/websearch/types"
)

// Client клиент для веб-поиска через DuckDuckGo
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	limiter    *rate.Limiter
	cache      *Cache
}

// ClientConfig конфигурация клиента
type ClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	RateLimit  rate.Limit
	Cache      *Cache
}

// NewClient создает новый клиент для веб-поиска
func NewClient(config ClientConfig) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.duckduckgo.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.RateLimit == 0 {
		config.RateLimit = rate.Every(time.Second) // 1 запрос в секунду
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		timeout: config.Timeout,
		limiter: rate.NewLimiter(config.RateLimit, 1),
		cache:   config.Cache,
	}
}

// Search выполняет поиск по запросу
// Сначала пытается использовать Instant Answer API, если результатов нет - использует HTML-поиск
func (c *Client) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	// Валидация и санитизация запроса
	query = sanitizeQuery(query)
	if query == "" {
		return nil, fmt.Errorf("empty query after sanitization")
	}

	// Проверка кэша
	cacheKey := generateCacheKey(query)
	if c.cache != nil {
		if cached, found := c.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Сначала пробуем Instant Answer API
	result, err := c.searchInstantAnswer(ctx, query)
	if err == nil && result != nil && result.Found && len(result.Results) > 0 {
		// Сохранение в кэш
		if c.cache != nil {
			c.cache.Set(cacheKey, result)
		}
		return result, nil
	}

	// Если Instant Answer не дал результатов, используем HTML-поиск
	return c.SearchHTML(ctx, query)
}

// searchInstantAnswer выполняет поиск через Instant Answer API
func (c *Client) searchInstantAnswer(ctx context.Context, query string) (*types.SearchResult, error) {
	// Проверка лимита запросов
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Формирование URL
	params := url.Values{}
	params.Add("q", query)
	params.Add("format", "json")
	params.Add("no_html", "1")

	fullURL := fmt.Sprintf("%s/?%s", c.baseURL, params.Encode())

	// Создание запроса с контекстом
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "HttpServer/1.0")

	// Выполнение запроса
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Декодирование ответа
	var ddgResponse DuckDuckGoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddgResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Преобразование в наш формат
	return c.convertResponse(&ddgResponse, query), nil
}

// convertResponse преобразует ответ DuckDuckGo в SearchResult
func (c *Client) convertResponse(ddgResp *DuckDuckGoResponse, query string) *types.SearchResult {
	result := &types.SearchResult{
		Query:     query,
		Source:    "duckduckgo",
		Timestamp: time.Now(),
		Results:   make([]types.SearchItem, 0),
	}

	// Обрабатываем Abstract (Instant Answer)
	if ddgResp.AbstractText != "" {
		result.Found = true
		result.Results = append(result.Results, types.SearchItem{
			Title:     ddgResp.Abstract,
			URL:       ddgResp.AbstractURL,
			Snippet:   ddgResp.AbstractText,
			Relevance: 1.0, // Instant Answer имеет максимальную релевантность
		})
		result.Confidence = 0.9
	}

	// Обрабатываем RelatedTopics
	for _, topic := range ddgResp.RelatedTopics {
		if topic.Text != "" && topic.FirstURL != "" {
			result.Found = true
			result.Results = append(result.Results, types.SearchItem{
				Title:     extractTitle(topic.Text),
				URL:       topic.FirstURL,
				Snippet:   topic.Text,
				Relevance: 0.7,
			})
			if result.Confidence < 0.7 {
				result.Confidence = 0.7
			}
		}
	}

	// Обрабатываем Results
	for _, res := range ddgResp.Results {
		if res.Text != "" && res.FirstURL != "" {
			result.Found = true
			result.Results = append(result.Results, types.SearchItem{
				Title:     extractTitle(res.Text),
				URL:       res.FirstURL,
				Snippet:   res.Text,
				Relevance: 0.6,
			})
			if result.Confidence < 0.6 {
				result.Confidence = 0.6
			}
		}
	}

	// Если результатов нет, устанавливаем низкую уверенность
	if !result.Found {
		result.Confidence = 0.0
	}

	return result
}

// sanitizeQuery очищает и валидирует поисковый запрос
func sanitizeQuery(query string) string {
	// Убираем лишние пробелы
	query = strings.TrimSpace(query)

	// Ограничиваем длину
	maxLength := 200
	if len(query) > maxLength {
		query = query[:maxLength]
	}

	return query
}

// extractTitle извлекает заголовок из текста
func extractTitle(text string) string {
	// Берем первые 100 символов как заголовок
	if len(text) > 100 {
		return text[:100] + "..."
	}
	return text
}

// generateCacheKey генерирует ключ кэша из запроса
func generateCacheKey(query string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(query)))
	return hex.EncodeToString(hash[:])
}

