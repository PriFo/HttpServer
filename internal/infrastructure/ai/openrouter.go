package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"httpserver/nomenclature"
)

// OpenRouterClient клиент для работы с OpenRouter API
type OpenRouterClient struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	retryConfig RetryConfig
}

// GetAPIKey возвращает API ключ (для доступа из других пакетов)
func (c *OpenRouterClient) GetAPIKey() string {
	return c.apiKey
}

// OpenRouterModel модель OpenRouter
type OpenRouterModel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Context     int                    `json:"context_length,omitempty"`
	Architecture map[string]interface{} `json:"architecture,omitempty"`
	TopProvider struct {
		MaxCompletions int     `json:"max_completions,omitempty"`
		IsModerated    bool    `json:"is_moderated,omitempty"`
		IsVirtual      bool    `json:"is_virtual,omitempty"`
	} `json:"top_provider,omitempty"`
	Pricing struct {
		Prompt  string `json:"prompt,omitempty"`
		Completion string `json:"completion,omitempty"`
	} `json:"pricing,omitempty"`
}

// OpenRouterModelsResponse ответ со списком моделей от OpenRouter
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// NewOpenRouterClient создает новый клиент OpenRouter
func NewOpenRouterClient(apiKey string) *OpenRouterClient {
	baseURL := "https://openrouter.ai/api/v1"

	// Оптимизированный HTTP Transport с connection pooling
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     5,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		MaxIdleConnsPerHost: 5,
	}

	return &OpenRouterClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		retryConfig: DefaultRetryConfig(),
	}
}

// GetModels получает список моделей с повторными попытками
func (c *OpenRouterClient) GetModels(ctx context.Context, requestID string) ([]OpenRouterModel, error) {
	startTime := time.Now()

	url := fmt.Sprintf("%s/models", c.baseURL)

	var lastErr error
	delay := c.retryConfig.InitialDelay

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[%s] Retry attempt %d/%d for models after %v", requestID, attempt, c.retryConfig.MaxRetries, delay)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * c.retryConfig.BackoffMultiplier)
			if delay > c.retryConfig.MaxDelay {
				delay = c.retryConfig.MaxDelay
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		if c.apiKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}
		req.Header.Set("HTTP-Referer", "https://github.com/your-repo") // OpenRouter требует HTTP-Referer
		req.Header.Set("X-Request-ID", requestID)
		req.Header.Set("Content-Type", "application/json")

		log.Printf("[%s] Fetching OpenRouter models (attempt %d/%d)", requestID, attempt+1, c.retryConfig.MaxRetries+1)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			log.Printf("[%s] Models fetch failed: %v", requestID, lastErr)
			continue
		}

		duration := time.Since(startTime)

		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("failed to read response: %w", err)
				log.Printf("[%s] Failed to read response: %v", requestID, lastErr)
				continue
			}

			var modelsResp OpenRouterModelsResponse
			if err := json.Unmarshal(body, &modelsResp); err != nil {
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				log.Printf("[%s] Failed to decode models response: %v, body: %s", requestID, lastErr, string(body))
				continue
			}

			log.Printf("[%s] OpenRouter models fetched successfully (duration: %v, count: %d)", requestID, duration, len(modelsResp.Data))
			return modelsResp.Data, nil
		}

		resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			log.Printf("[%s] Server error %d, will retry", requestID, resp.StatusCode)
			continue
		}

		lastErr = fmt.Errorf("client error: %d", resp.StatusCode)
		log.Printf("[%s] Client error %d, not retrying", requestID, resp.StatusCode)
		break
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// ChatCompletion выполняет запрос к OpenRouter API для получения ответа от модели
// Поддерживает retry с экспоненциальной задержкой для ошибок rate limit и quota exceeded
func (c *OpenRouterClient) ChatCompletion(model string, messages []nomenclature.Message) (string, error) {
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)

	// Формируем запрос в формате OpenRouter
	requestBody := map[string]interface{}{
		"model":    model,
		"messages":  messages,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	delay := c.retryConfig.InitialDelay

	// Retry логика для обработки rate limit и quota ошибок
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[OpenRouter] Retry attempt %d/%d for ChatCompletion after %v", attempt, c.retryConfig.MaxRetries, delay)
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * c.retryConfig.BackoffMultiplier)
			if delay > c.retryConfig.MaxDelay {
				delay = c.retryConfig.MaxDelay
			}
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		if c.apiKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}
		req.Header.Set("HTTP-Referer", "https://github.com/your-repo")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			log.Printf("[OpenRouter] Request failed (attempt %d/%d): %v", attempt+1, c.retryConfig.MaxRetries+1, lastErr)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Обработка HTTP 429 (Too Many Requests) и quota exceeded
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := c.parseRetryAfter(resp)
			if retryAfter > 0 {
				delay = retryAfter
			}
			lastErr = fmt.Errorf("rate limit exceeded (429): %s", string(body))
			log.Printf("[OpenRouter] Rate limit exceeded (attempt %d/%d), retry after %v: %s", 
				attempt+1, c.retryConfig.MaxRetries+1, delay, string(body))
			continue
		}

		// Обработка других ошибок статуса
		if resp.StatusCode != http.StatusOK {
			// Парсим ошибку из ответа
			var errorResp struct {
				Error *struct {
					Message string `json:"message"`
					Type    string `json:"type"`
					Code    string `json:"code,omitempty"`
				} `json:"error,omitempty"`
			}
			
			// Пытаемся распарсить ошибку
			json.Unmarshal(body, &errorResp)
			
			errorMsg := string(body)
			if errorResp.Error != nil {
				errorMsg = errorResp.Error.Message
				// Проверяем на quota exceeded
				if strings.Contains(strings.ToLower(errorMsg), "quota") || 
				   strings.Contains(strings.ToLower(errorMsg), "exceeded") ||
				   strings.Contains(strings.ToLower(errorResp.Error.Type), "quota") {
					lastErr = fmt.Errorf("quota exceeded: %s (type: %s)", errorMsg, errorResp.Error.Type)
					log.Printf("[OpenRouter] Quota exceeded (attempt %d/%d): %s", 
						attempt+1, c.retryConfig.MaxRetries+1, errorMsg)
					// Для quota exceeded не делаем retry, так как это не временная ошибка
					return "", lastErr
				}
			}
			
			lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, errorMsg)
			
			// Для 5xx ошибок делаем retry
			if resp.StatusCode >= 500 && attempt < c.retryConfig.MaxRetries {
				log.Printf("[OpenRouter] Server error %d (attempt %d/%d), will retry: %s", 
					resp.StatusCode, attempt+1, c.retryConfig.MaxRetries+1, errorMsg)
				continue
			}
			
			// Для других ошибок не делаем retry
			return "", lastErr
		}

		// Успешный ответ - парсим
		var response struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error,omitempty"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			lastErr = fmt.Errorf("failed to decode response: %w", err)
			log.Printf("[OpenRouter] Failed to decode response (attempt %d/%d): %v", 
				attempt+1, c.retryConfig.MaxRetries+1, lastErr)
			continue
		}

		if response.Error != nil {
			errorMsg := response.Error.Message
			// Проверяем на quota/rate limit в сообщении об ошибке
			if strings.Contains(strings.ToLower(errorMsg), "quota") || 
			   strings.Contains(strings.ToLower(errorMsg), "rate limit") {
				lastErr = fmt.Errorf("quota/rate limit error: %s (type: %s)", errorMsg, response.Error.Type)
				log.Printf("[OpenRouter] Quota/rate limit error in response (attempt %d/%d): %s", 
					attempt+1, c.retryConfig.MaxRetries+1, errorMsg)
				if attempt < c.retryConfig.MaxRetries {
					continue
				}
			}
			return "", fmt.Errorf("API error: %s (type: %s)", errorMsg, response.Error.Type)
		}

		if len(response.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// parseRetryAfter парсит заголовок Retry-After из ответа
func (c *OpenRouterClient) parseRetryAfter(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}
	
	// Пытаемся распарсить как число секунд
	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}
	
	// Пытаемся распарсить как число
	if seconds, err := time.ParseDuration(fmt.Sprintf("%ss", retryAfter)); err == nil {
		return seconds
	}
	
	return 0
}


