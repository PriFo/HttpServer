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

// DuckDuckGoProvider провайдер для DuckDuckGo API
type DuckDuckGoProvider struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	rateLimit  time.Duration
	available  bool
}

// NewDuckDuckGoProvider создает новый провайдер DuckDuckGo
func NewDuckDuckGoProvider(timeout time.Duration, rateLimit time.Duration) *DuckDuckGoProvider {
	limiter := rate.NewLimiter(rate.Every(rateLimit), 1)

	return &DuckDuckGoProvider{
		baseURL: "https://api.duckduckgo.com",
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter:   limiter,
		rateLimit: rateLimit,
		available: true,
	}
}

// GetName возвращает имя провайдера
func (d *DuckDuckGoProvider) GetName() string {
	return "duckduckgo"
}

// IsAvailable проверяет доступность провайдера
func (d *DuckDuckGoProvider) IsAvailable() bool {
	return d.available
}

// ValidateCredentials проверяет валидность учетных данных (для DuckDuckGo не требуется)
func (d *DuckDuckGoProvider) ValidateCredentials(ctx context.Context) error {
	// DuckDuckGo не требует API ключа
	return nil
}

// Search выполняет поиск через DuckDuckGo API
func (d *DuckDuckGoProvider) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	if err := d.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	params := url.Values{}
	params.Add("q", query)
	params.Add("format", "json")
	params.Add("no_html", "1")
	params.Add("skip_disambig", "1")

	fullURL := fmt.Sprintf("%s/?%s", d.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "HttpServer/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		d.available = false
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Обработка различных HTTP статусов
	if resp.StatusCode == http.StatusTooManyRequests {
		d.available = false
		return nil, fmt.Errorf("rate limit exceeded: too many requests")
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		d.available = false
		return nil, fmt.Errorf("service unavailable: DuckDuckGo API is temporarily down")
	}

	if resp.StatusCode != http.StatusOK {
		d.available = false
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, resp.Status)
	}

	var ddgResponse DuckDuckGoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddgResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	d.available = true
	return d.transformResults(query, &ddgResponse), nil
}

// DuckDuckGoResponse структура ответа DuckDuckGo API
type DuckDuckGoResponse struct {
	Abstract     string `json:"Abstract"`
	AbstractText string `json:"AbstractText"`
	AbstractURL  string `json:"AbstractURL"`
	Answer       string `json:"Answer"`
	AnswerType   string `json:"AnswerType"`
	Definition   string `json:"Definition"`
	DefinitionURL string `json:"DefinitionURL"`
	Heading      string `json:"Heading"`
	Image        string `json:"Image"`
	ImageIsLogo  int    `json:"ImageIsLogo"`
	ImageURL     string `json:"ImageURL"`
	Infobox      string `json:"Infobox"`
	Redirect     string `json:"Redirect"`
	RelatedTopics []struct {
		FirstURL string `json:"FirstURL"`
		Icon     struct {
			Height string `json:"Height"`
			URL    string `json:"URL"`
			Width  string `json:"Width"`
		} `json:"Icon"`
		Result string `json:"Result"`
		Text   string `json:"Text"`
	} `json:"RelatedTopics"`
	Results []struct {
		FirstURL string `json:"FirstURL"`
		Icon     struct {
			Height string `json:"Height"`
			URL    string `json:"URL"`
			Width  string `json:"Width"`
		} `json:"Icon"`
		Result string `json:"Result"`
		Text   string `json:"Text"`
	} `json:"Results"`
}

// transformResults преобразует ответ DuckDuckGo в унифицированный формат
func (d *DuckDuckGoProvider) transformResults(query string, resp *DuckDuckGoResponse) *types.SearchResult {
	results := make([]types.SearchItem, 0)

	// Добавляем основной результат, если есть
	if resp.AbstractText != "" {
		results = append(results, types.SearchItem{
			Title:     resp.Heading,
			URL:       resp.AbstractURL,
			Snippet:   resp.AbstractText,
			Relevance: 1.0,
		})
	}

	// Добавляем связанные темы
	for _, topic := range resp.RelatedTopics {
		if topic.FirstURL != "" && topic.Text != "" {
			results = append(results, types.SearchItem{
				Title:     topic.Text,
				URL:       topic.FirstURL,
				Snippet:   topic.Text,
				Relevance: 0.8,
			})
		}
	}

	// Добавляем результаты
	for _, result := range resp.Results {
		if result.FirstURL != "" && result.Text != "" {
			results = append(results, types.SearchItem{
				Title:     result.Text,
				URL:       result.FirstURL,
				Snippet:   result.Text,
				Relevance: 0.7,
			})
		}
	}

	confidence := 0.0
	if len(results) > 0 {
		confidence = 0.7 // Базовая уверенность для DuckDuckGo
		if resp.AbstractText != "" {
			confidence = 0.9 // Выше, если есть основной результат
		}
	}

	return &types.SearchResult{
		Query:      query,
		Found:      len(results) > 0,
		Results:    results,
		Confidence: confidence,
		Source:     d.GetName(),
		Timestamp:  time.Now(),
	}
}

// GetRateLimit возвращает лимит запросов
func (d *DuckDuckGoProvider) GetRateLimit() time.Duration {
	return d.rateLimit
}

