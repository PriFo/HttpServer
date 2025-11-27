package nomenclature

import (
	"testing"
	"time"
)

// TestNewAIClient проверяет создание нового AI клиента
func TestNewAIClient(t *testing.T) {
	apiKey := "test-api-key"
	model := "test-model"
	
	client := NewAIClient(apiKey, model)
	
	if client == nil {
		t.Fatal("NewAIClient() returned nil")
	}
	
	if client.apiKey != apiKey {
		t.Errorf("AIClient.apiKey = %v, want %v", client.apiKey, apiKey)
	}
	
	if client.model != model {
		t.Errorf("AIClient.model = %v, want %v", client.model, model)
	}
	
	if client.baseURL == "" {
		t.Error("AIClient.baseURL is empty")
	}
	
	if client.httpClient == nil {
		t.Error("AIClient.httpClient is nil")
	}
	
	if client.rateLimiter == nil {
		t.Error("AIClient.rateLimiter is nil")
	}
	
	if client.circuitBreaker == nil {
		t.Error("AIClient.circuitBreaker is nil")
	}
}

// TestNewAIClientWithBaseURL проверяет создание клиента с кастомным URL
func TestNewAIClientWithBaseURL(t *testing.T) {
	apiKey := "test-api-key"
	model := "test-model"
	baseURL := "https://custom-api.example.com"
	
	client := NewAIClientWithBaseURL(apiKey, model, baseURL)
	
	if client == nil {
		t.Fatal("NewAIClientWithBaseURL() returned nil")
	}
	
	if client.baseURL != baseURL {
		t.Errorf("AIClient.baseURL = %v, want %v", client.baseURL, baseURL)
	}
}

// TestCircuitBreaker проверяет работу Circuit Breaker
func TestCircuitBreaker(t *testing.T) {
	breaker := &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: 3,
		successThreshold: 2,
		timeout:          1 * time.Second,
	}
	
	// Проверяем начальное состояние
	if breaker.getState() != "closed" {
		t.Errorf("Initial state = %v, want closed", breaker.getState())
	}
	
	// Проверяем, что можно выполнять запросы
	if !breaker.canProceed() {
		t.Error("Circuit breaker should allow requests in closed state")
	}
	
	// Симулируем несколько ошибок
	for i := 0; i < 3; i++ {
		breaker.recordFailure()
	}
	
	// После 3 ошибок breaker должен открыться
	if breaker.getState() != "open" {
		t.Errorf("State after failures = %v, want open", breaker.getState())
	}
	
	// В открытом состоянии запросы должны блокироваться
	if breaker.canProceed() {
		t.Error("Circuit breaker should block requests in open state")
	}
	
	// Ждем таймаут
	time.Sleep(2 * time.Second)
	
	// Вызываем canProceed() чтобы перевести в half-open (это происходит автоматически в canProceed)
	if !breaker.canProceed() {
		t.Error("Circuit breaker should allow requests after timeout")
	}
	
	// После вызова canProceed() должен перейти в half-open
	if breaker.getState() != "half-open" {
		t.Errorf("State after timeout and canProceed() = %v, want half-open", breaker.getState())
	}
	
	// Симулируем успешные запросы
	for i := 0; i < 2; i++ {
		breaker.recordSuccess()
	}
	
	// После успешных запросов должен закрыться
	if breaker.getState() != "closed" {
		t.Errorf("State after successes = %v, want closed", breaker.getState())
	}
}

// TestCircuitBreaker_StateTransitions проверяет переходы состояний
func TestCircuitBreaker_StateTransitions(t *testing.T) {
	breaker := &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: 2,
		successThreshold: 1,
		timeout:          100 * time.Millisecond,
	}
	
	tests := []struct {
		name           string
		action         func()
		expectedState  string
		canProceed     bool
	}{
		{
			name: "initial closed state",
			action: func() {
				// Ничего не делаем
			},
			expectedState: "closed",
			canProceed:    true,
		},
		{
			name: "open after failures",
			action: func() {
				breaker.recordFailure()
				breaker.recordFailure()
			},
			expectedState: "open",
			canProceed:    false,
		},
		{
			name: "half-open after timeout",
			action: func() {
				time.Sleep(150 * time.Millisecond)
				// Вызываем canProceed() чтобы перевести в half-open
				breaker.canProceed()
			},
			expectedState: "half-open",
			canProceed:    true,
		},
		{
			name: "closed after success",
			action: func() {
				breaker.recordSuccess()
			},
			expectedState: "closed",
			canProceed:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.action()
			
			if breaker.getState() != tt.expectedState {
				t.Errorf("State = %v, want %v", breaker.getState(), tt.expectedState)
			}
			
			if breaker.canProceed() != tt.canProceed {
				t.Errorf("canProceed() = %v, want %v", breaker.canProceed(), tt.canProceed)
			}
		})
	}
}

// TestAIRequest проверяет структуру AI запроса
func TestAIRequest(t *testing.T) {
	request := AIRequest{
		Model:       "test-model",
		Messages:    []Message{{Role: "user", Content: "test"}},
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      false,
	}
	
	if request.Model == "" {
		t.Error("AIRequest.Model is empty")
	}
	
	if len(request.Messages) == 0 {
		t.Error("AIRequest.Messages is empty")
	}
	
	if request.Temperature < 0 || request.Temperature > 2 {
		t.Errorf("AIRequest.Temperature = %f, should be between 0 and 2", request.Temperature)
	}
}

// TestMessage проверяет структуру сообщения
func TestMessage(t *testing.T) {
	message := Message{
		Role:    "user",
		Content: "test content",
	}
	
	if message.Role == "" {
		t.Error("Message.Role is empty")
	}
	
	if message.Content == "" {
		t.Error("Message.Content is empty")
	}
}

// TestAIProcessingResult проверяет структуру результата обработки
func TestAIProcessingResult(t *testing.T) {
	result := &AIProcessingResult{
		NormalizedName: "test name",
		KpvedCode:      "01.11",
		KpvedName:      "test category",
		Confidence:     0.95,
		Reasoning:      "test reasoning",
	}
	
	if result.NormalizedName == "" {
		t.Error("AIProcessingResult.NormalizedName is empty")
	}
	
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("AIProcessingResult.Confidence = %f, should be between 0 and 1", result.Confidence)
	}
}

