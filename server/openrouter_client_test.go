package server

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestNewOpenRouterClient проверяет создание нового клиента OpenRouter
func TestNewOpenRouterClient(t *testing.T) {
	apiKey := "test-api-key"
	client := NewOpenRouterClient(apiKey)
	
	if client == nil {
		t.Fatal("NewOpenRouterClient() returned nil")
	}
	
	if client.apiKey != apiKey {
		t.Errorf("OpenRouterClient.apiKey = %v, want %v", client.apiKey, apiKey)
	}
	
	if client.baseURL == "" {
		t.Error("OpenRouterClient.baseURL should not be empty")
	}
	
	if client.httpClient == nil {
		t.Error("OpenRouterClient.httpClient is nil")
	}
	
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("HTTP client timeout = %v, want 30s", client.httpClient.Timeout)
	}
}

// TestOpenRouterModel проверяет структуру модели OpenRouter
func TestOpenRouterModel(t *testing.T) {
	model := OpenRouterModel{
		ID:          "test-model-id",
		Name:        "Test Model",
		Description: "Test description",
		Context:     4096,
	}
	
	if model.ID == "" {
		t.Error("OpenRouterModel.ID should not be empty")
	}
	
	if model.Name == "" {
		t.Error("OpenRouterModel.Name should not be empty")
	}
	
	if model.Context <= 0 {
		t.Error("OpenRouterModel.Context should be positive")
	}
}

// TestOpenRouterModelsResponse проверяет структуру ответа со списком моделей
func TestOpenRouterModelsResponse(t *testing.T) {
	response := OpenRouterModelsResponse{
		Data: []OpenRouterModel{
			{ID: "model1", Name: "Model 1"},
			{ID: "model2", Name: "Model 2"},
		},
	}
	
	if len(response.Data) == 0 {
		t.Error("OpenRouterModelsResponse.Data should not be empty")
	}
	
	for i, model := range response.Data {
		if model.ID == "" {
			t.Errorf("Model %d ID should not be empty", i)
		}
		if model.Name == "" {
			t.Errorf("Model %d Name should not be empty", i)
		}
	}
}

// TestOpenRouterClient_RetryConfig проверяет конфигурацию повторов
func TestOpenRouterClient_RetryConfig(t *testing.T) {
	client := NewOpenRouterClient("test-key")
	
	if client.retryConfig.MaxRetries <= 0 {
		t.Error("RetryConfig.MaxRetries should be positive")
	}
	
	if client.retryConfig.InitialDelay <= 0 {
		t.Error("RetryConfig.InitialDelay should be positive")
	}
	
	if client.retryConfig.MaxDelay <= 0 {
		t.Error("RetryConfig.MaxDelay should be positive")
	}
	
	if client.retryConfig.BackoffMultiplier <= 0 {
		t.Error("RetryConfig.BackoffMultiplier should be positive")
	}
	
	if client.retryConfig.InitialDelay > client.retryConfig.MaxDelay {
		t.Error("InitialDelay should not exceed MaxDelay")
	}
}

// TestOpenRouterClient_GetModels_Context проверяет обработку контекста
func TestOpenRouterClient_GetModels_Context(t *testing.T) {
	client := NewOpenRouterClient("test-key")
	
	// Тест с отмененным контекстом
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу
	
	_, err := client.GetModels(ctx, "test-request-id")
	if err == nil {
		t.Error("GetModels() should return error for cancelled context")
	}
	
	if err != nil && !strings.Contains(err.Error(), "cancelled") {
		t.Logf("GetModels() returned error (may be expected): %v", err)
	}
}

// TestOpenRouterClient_HTTPTransport проверяет настройки HTTP транспорта
func TestOpenRouterClient_HTTPTransport(t *testing.T) {
	client := NewOpenRouterClient("test-key")
	
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("HTTP Transport is not *http.Transport")
	}
	
	if transport.MaxIdleConns <= 0 {
		t.Error("Transport.MaxIdleConns should be positive")
	}
	
	if transport.MaxConnsPerHost <= 0 {
		t.Error("Transport.MaxConnsPerHost should be positive")
	}
	
	if transport.IdleConnTimeout <= 0 {
		t.Error("Transport.IdleConnTimeout should be positive")
	}
}

