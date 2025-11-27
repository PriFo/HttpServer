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

// EdenAIClient клиент для работы с Eden AI API
type EdenAIClient struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	retryConfig RetryConfig
}

// EdenAITextGenerationRequest структура запроса для генерации текста
type EdenAITextGenerationRequest struct {
	Providers string                 `json:"providers"`
	Text      string                 `json:"text"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
}

// EdenAITextGenerationResponse структура ответа от Eden AI
type EdenAITextGenerationResponse map[string]interface{}

// EdenAIBrick структура кирпичика (модели) Eden AI
type EdenAIBrick struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// EdenAIBricksCatalogResponse ответ со списком кирпичиков
type EdenAIBricksCatalogResponse struct {
	Bricks []EdenAIBrick `json:"bricks,omitempty"`
	Data   []EdenAIBrick `json:"data,omitempty"` // Альтернативный формат
}

// NewEdenAIClient создает новый клиент Eden AI
func NewEdenAIClient(apiKey, baseURL string) *EdenAIClient {
	if baseURL == "" {
		baseURL = "https://api.edenai.run/v2"
	}

	// Оптимизированный HTTP Transport с connection pooling
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     5,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		MaxIdleConnsPerHost: 5,
	}

	return &EdenAIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
		retryConfig: DefaultRetryConfig(),
	}
}

// ChatCompletion выполняет запрос к Eden AI API для генерации текста
func (c *EdenAIClient) ChatCompletion(model string, messages []nomenclature.Message) (string, error) {
	ctx := context.Background()
	return c.ChatCompletionWithContext(ctx, model, messages)
}

// ChatCompletionWithContext выполняет запрос к Eden AI API с поддержкой контекста и retry
func (c *EdenAIClient) ChatCompletionWithContext(ctx context.Context, model string, messages []nomenclature.Message) (string, error) {
	// Преобразуем messages в единый промпт
	prompt := c.messagesToPrompt(messages)

	// Определяем провайдеров из модели (формат: "provider/model" или просто модель)
	providers := "openai,google,cohere" // По умолчанию используем несколько провайдеров
	if model != "" {
		// Если модель содержит провайдера, извлекаем его
		parts := strings.Split(model, "/")
		if len(parts) > 0 {
			// Маппинг названий провайдеров
			providerMap := map[string]string{
				"openai":  "openai",
				"google":  "google",
				"cohere":  "cohere",
				"anthropic": "anthropic",
				"meta":    "meta",
			}
			if mapped, ok := providerMap[strings.ToLower(parts[0])]; ok {
				providers = mapped
			}
		}
	}

	// Формируем запрос в формате Eden AI
	request := EdenAITextGenerationRequest{
		Providers: providers,
		Text:      prompt,
		Settings: map[string]interface{}{
			"temperature": 0.2,
			"max_tokens":  512,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// URL для генерации текста
	url := fmt.Sprintf("%s/text/generation", c.baseURL)

	var lastErr error
	delay := c.retryConfig.InitialDelay

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d for Eden AI chat completion after %v", attempt, c.retryConfig.MaxRetries, delay)
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * c.retryConfig.BackoffMultiplier)
			if delay > c.retryConfig.MaxDelay {
				delay = c.retryConfig.MaxDelay
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			log.Printf("Eden AI chat completion failed: %v", lastErr)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			log.Printf("Failed to read Eden AI response body: %v", lastErr)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			// Парсим ответ Eden AI
			var edenaiResp EdenAITextGenerationResponse
			if err := json.Unmarshal(body, &edenaiResp); err != nil {
				lastErr = fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
				log.Printf("Failed to decode Eden AI response: %v", lastErr)
				continue
			}

			// Извлекаем сгенерированный текст из ответа
			// Eden AI возвращает результаты от каждого провайдера в формате:
			// { "provider_name": { "generated_text": "...", "status": "success" }, ... }
			generatedText, err := c.extractGeneratedText(edenaiResp)
			if err != nil {
				lastErr = fmt.Errorf("failed to extract generated text: %w, response: %s", err, string(body))
				log.Printf("Failed to extract generated text: %v", lastErr)
				continue
			}

			return strings.TrimSpace(generatedText), nil
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d, body: %s", resp.StatusCode, string(body))
			log.Printf("Eden AI server error %d, will retry: %s", resp.StatusCode, string(body))
			continue
		}

		lastErr = fmt.Errorf("client error: %d, body: %s", resp.StatusCode, string(body))
		log.Printf("Eden AI client error %d, not retrying: %s", resp.StatusCode, string(body))
		break
	}

	return "", fmt.Errorf("all retry attempts failed for Eden AI chat completion: %w", lastErr)
}

// extractGeneratedText извлекает сгенерированный текст из ответа Eden AI
func (c *EdenAIClient) extractGeneratedText(response EdenAITextGenerationResponse) (string, error) {
	// Итерируем по всем провайдерам в ответе
	for _, providerData := range response {
		if providerMap, ok := providerData.(map[string]interface{}); ok {
			// Проверяем статус
			if status, ok := providerMap["status"].(string); ok && status == "success" {
				// Извлекаем generated_text
				if generatedText, ok := providerMap["generated_text"].(string); ok && generatedText != "" {
					return generatedText, nil
				}
				// Альтернативное поле
				if text, ok := providerMap["text"].(string); ok && text != "" {
					return text, nil
				}
			}
		}
	}

	// Если не нашли успешный ответ, возвращаем первый доступный текст
	for _, providerData := range response {
		if providerMap, ok := providerData.(map[string]interface{}); ok {
			if generatedText, ok := providerMap["generated_text"].(string); ok && generatedText != "" {
				return generatedText, nil
			}
			if text, ok := providerMap["text"].(string); ok && text != "" {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("no generated text found in response")
}

// messagesToPrompt конвертирует массив сообщений в единый промпт
func (c *EdenAIClient) messagesToPrompt(messages []nomenclature.Message) string {
	var parts []string
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			parts = append(parts, fmt.Sprintf("System: %s", msg.Content))
		case "user":
			parts = append(parts, fmt.Sprintf("User: %s", msg.Content))
		case "assistant":
			parts = append(parts, fmt.Sprintf("Assistant: %s", msg.Content))
		default:
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

// GetModels получает список доступных моделей (bricks) из Eden AI
func (c *EdenAIClient) GetModels(ctx context.Context, requestID string) ([]ArliaiModel, error) {
	startTime := time.Now()

	// Используем каталог bricks
	url := "https://app.edenai.run/bricks/catalog"

	var lastErr error
	delay := c.retryConfig.InitialDelay

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[%s] Retry attempt %d/%d for Eden AI models after %v", requestID, attempt, c.retryConfig.MaxRetries, delay)
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
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", requestID)

		log.Printf("[%s] Fetching Eden AI models (attempt %d/%d)", requestID, attempt+1, c.retryConfig.MaxRetries+1)

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

			var catalogResp EdenAIBricksCatalogResponse
			if err := json.Unmarshal(body, &catalogResp); err != nil {
				// Попробуем альтернативный формат
				var altResp struct {
					Bricks []EdenAIBrick `json:"bricks"`
				}
				if err2 := json.Unmarshal(body, &altResp); err2 != nil {
					lastErr = fmt.Errorf("failed to decode response: %w (also tried alt format: %v), body: %s", err, err2, string(body))
					log.Printf("[%s] Failed to decode models response: %v, body: %s", requestID, lastErr, string(body))
					continue
				}
				catalogResp.Bricks = altResp.Bricks
			}

			// Объединяем bricks из разных полей
			bricks := catalogResp.Bricks
			if len(bricks) == 0 {
				bricks = catalogResp.Data
			}

			// Фильтруем только текстовые модели
			textModels := make([]ArliaiModel, 0)
			for _, brick := range bricks {
				// Фильтруем по категории или тегам
				isTextModel := false
				if brick.Category != "" {
					categoryLower := strings.ToLower(brick.Category)
					isTextModel = strings.Contains(categoryLower, "text") ||
						strings.Contains(categoryLower, "chat") ||
						strings.Contains(categoryLower, "generation")
				}
				if !isTextModel && len(brick.Tags) > 0 {
					for _, tag := range brick.Tags {
						tagLower := strings.ToLower(tag)
						if strings.Contains(tagLower, "text") ||
							strings.Contains(tagLower, "chat") ||
							strings.Contains(tagLower, "generation") {
							isTextModel = true
							break
						}
					}
				}
				// Если категория и теги пустые, включаем все (на случай если API не возвращает эти поля)
				if !isTextModel && brick.Category == "" && len(brick.Tags) == 0 {
					isTextModel = true
				}

				if isTextModel {
					textModels = append(textModels, ArliaiModel{
						ID:          brick.ID,
						Name:        brick.Name,
						Description: brick.Description,
						Status:      "active",
						MaxTokens:   4096, // Значение по умолчанию
						Speed:       "medium",
						Quality:     "high",
						Tags:        brick.Tags,
					})
				}
			}

			log.Printf("[%s] Eden AI models fetched successfully (duration: %v, count: %d)", requestID, duration, len(textModels))
			return textModels, nil
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

// CheckConnection проверяет подключение к Eden AI API
func (c *EdenAIClient) CheckConnection(ctx context.Context, requestID string) (map[string]interface{}, error) {
	// Простая проверка: пытаемся получить список моделей
	models, err := c.GetModels(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to check connection: %w", err)
	}

	status := map[string]interface{}{
		"status":        "ok",
		"connected":     true,
		"models_count":  len(models),
		"api_available": true,
		"timestamp":     time.Now(),
	}

	return status, nil
}

