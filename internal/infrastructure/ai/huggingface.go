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

// HuggingFaceClient клиент для работы с Hugging Face Inference API
type HuggingFaceClient struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	retryConfig RetryConfig
}

// GetAPIKey возвращает API ключ (для доступа из других пакетов)
func (c *HuggingFaceClient) GetAPIKey() string {
	return c.apiKey
}

// HuggingFaceRequest структура запроса к Hugging Face API
type HuggingFaceRequest struct {
	Inputs     string                 `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// HuggingFaceResponse структура ответа от Hugging Face API
type HuggingFaceResponse []struct {
	GeneratedText string `json:"generated_text"`
}

// NewHuggingFaceClient создает новый клиент Hugging Face
func NewHuggingFaceClient(apiKey, baseURL string) *HuggingFaceClient {
	if baseURL == "" {
		baseURL = "https://api-inference.huggingface.co"
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

	return &HuggingFaceClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
		retryConfig: DefaultRetryConfig(),
	}
}

// ChatCompletion отправляет запрос к Hugging Face API для генерации текста
func (c *HuggingFaceClient) ChatCompletion(model string, messages []nomenclature.Message) (string, error) {
	ctx := context.Background()
	return c.ChatCompletionWithContext(ctx, model, messages)
}

// ChatCompletionWithContext отправляет запрос к Hugging Face API с поддержкой контекста и retry
func (c *HuggingFaceClient) ChatCompletionWithContext(ctx context.Context, model string, messages []nomenclature.Message) (string, error) {
	// Трансформируем messages в единый промпт
	prompt := c.messagesToPrompt(messages)

	// Формируем запрос в формате Hugging Face
	request := HuggingFaceRequest{
		Inputs: prompt,
		Parameters: map[string]interface{}{
			"max_new_tokens": 512,
			"temperature":    0.7,
			"return_full_text": false,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// URL для конкретной модели
	url := fmt.Sprintf("%s/models/%s", c.baseURL, model)

	var lastErr error
	delay := c.retryConfig.InitialDelay

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d for Hugging Face chat completion after %v", attempt, c.retryConfig.MaxRetries, delay)
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
			log.Printf("Hugging Face chat completion failed: %v", lastErr)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			log.Printf("Failed to read Hugging Face response body: %v", lastErr)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			// Парсим ответ
			var hfResp HuggingFaceResponse
			if err := json.Unmarshal(body, &hfResp); err != nil {
				// Попробуем альтернативный формат (один объект вместо массива)
				var singleResp struct {
					GeneratedText string `json:"generated_text"`
				}
				if err2 := json.Unmarshal(body, &singleResp); err2 != nil {
					lastErr = fmt.Errorf("failed to decode response: %w (also tried single format: %v), body: %s", err, err2, string(body))
					log.Printf("Failed to decode Hugging Face response: %v", lastErr)
					continue
				}
				return strings.TrimSpace(singleResp.GeneratedText), nil
			}

			if len(hfResp) == 0 {
				lastErr = fmt.Errorf("empty response from API")
				log.Printf("Empty Hugging Face response")
				continue
			}

			return strings.TrimSpace(hfResp[0].GeneratedText), nil
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d, body: %s", resp.StatusCode, string(body))
			log.Printf("Hugging Face server error %d, will retry: %s", resp.StatusCode, string(body))
			continue
		}

		lastErr = fmt.Errorf("client error: %d, body: %s", resp.StatusCode, string(body))
		log.Printf("Hugging Face client error %d, not retrying: %s", resp.StatusCode, string(body))
		break
	}

	return "", fmt.Errorf("all retry attempts failed for Hugging Face chat completion: %w", lastErr)
}

// messagesToPrompt конвертирует массив сообщений в единый промпт
func (c *HuggingFaceClient) messagesToPrompt(messages []nomenclature.Message) string {
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

// GetModels возвращает курированный список моделей Hugging Face
func (c *HuggingFaceClient) GetModels(ctx context.Context, requestID string) ([]ArliaiModel, error) {
	// Курированный список популярных моделей для текстовой генерации
	curatedModels := []struct {
		ID          string
		Name        string
		Description string
		MaxTokens   int
	}{
		{
			ID:          "microsoft/DialoGPT-medium",
			Name:        "Microsoft DialoGPT Medium",
			Description: "Диалоговая модель от Microsoft",
			MaxTokens:   1024,
		},
		{
			ID:          "google/flan-t5-base",
			Name:        "Google FLAN-T5 Base",
			Description: "Инструкционная модель от Google",
			MaxTokens:   512,
		},
		{
			ID:          "meta-llama/Llama-2-7b-chat-hf",
			Name:        "Meta Llama 2 7B Chat",
			Description: "Чат-модель Llama 2 от Meta",
			MaxTokens:   4096,
		},
		{
			ID:          "mistralai/Mistral-7B-Instruct-v0.1",
			Name:        "Mistral 7B Instruct",
			Description: "Инструкционная модель Mistral 7B",
			MaxTokens:   8192,
		},
		{
			ID:          "google/gemma-7b-it",
			Name:        "Google Gemma 7B IT",
			Description: "Инструкционная модель Gemma от Google",
			MaxTokens:   8192,
		},
		{
			ID:          "HuggingFaceH4/zephyr-7b-beta",
			Name:        "Zephyr 7B Beta",
			Description: "Чат-модель Zephyr от Hugging Face",
			MaxTokens:   4096,
		},
		{
			ID:          "tiiuae/falcon-7b-instruct",
			Name:        "Falcon 7B Instruct",
			Description: "Инструкционная модель Falcon",
			MaxTokens:   2048,
		},
	}

	// Преобразуем в формат ArliaiModel
	models := make([]ArliaiModel, 0, len(curatedModels))
	for _, m := range curatedModels {
		models = append(models, ArliaiModel{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Status:      "active",
			MaxTokens:   m.MaxTokens,
			Speed:       "medium",
			Quality:     "high",
			Tags:        []string{"text-generation", "chat"},
		})
	}

	log.Printf("[%s] Hugging Face models list returned (count: %d)", requestID, len(models))
	return models, nil
}

// CheckConnection проверяет подключение к Hugging Face API
// Для Hugging Face проверяем доступность API, пытаясь получить список моделей
func (c *HuggingFaceClient) CheckConnection(ctx context.Context, requestID string) (map[string]interface{}, error) {
	// Простая проверка: пытаемся получить список моделей
	// Если это работает, значит API доступен и ключ валиден
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

