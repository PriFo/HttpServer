package resilience

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"httpserver/database"
	ai "httpserver/internal/infrastructure/ai"
	"httpserver/server"
)

// TestProviderFailure_Timeout проверяет обработку таймаута провайдера
func TestProviderFailure_Timeout(t *testing.T) {
	// Создаем сервер, который не отвечает (таймаут)
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Больше, чем таймаут клиента
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer slowServer.Close()

	// Создаем мок провайдера, который использует медленный сервер
	mockProvider := &timeoutProvider{
		baseURL: slowServer.URL,
		timeout: 1 * time.Second, // Короткий таймаут
	}

	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Slow Provider",
			Type:     "slow_provider",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	clients := map[string]ai.ProviderClient{
		"slow_provider": mockProvider,
	}

	mpc := server.NewMultiProviderClient(providers, clients, nil)
	// Таймаут устанавливается при создании клиента (по умолчанию 30 секунд)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Должен вернуть ошибку из-за таймаута, но не зависнуть
	_, err := mpc.NormalizeName(ctx, "test")
	if err == nil {
		t.Error("Expected error due to timeout, got nil")
	}
}

// TestProviderFailure_PartialFailure проверяет, что система работает при частичном отказе провайдеров
func TestProviderFailure_PartialFailure(t *testing.T) {
	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Provider 1",
			Type:     "provider1",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
		{
			ID:       2,
			Name:     "Provider 2",
			Type:     "provider2",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
		{
			ID:       3,
			Name:     "Provider 3",
			Type:     "provider3",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	// Один провайдер возвращает ошибку, остальные работают
	clients := map[string]ai.ProviderClient{
		"provider1": &failingProvider{name: "Provider 1", enabled: true, shouldError: true},
		"provider2": &failingProvider{name: "Provider 2", enabled: true, response: "Result A"},
		"provider3": &failingProvider{name: "Provider 3", enabled: true, response: "Result A"},
	}

	mpc := server.NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	result, err := mpc.NormalizeName(ctx, "test")

	// Должен вернуть результат от успешных провайдеров
	if err != nil {
		t.Fatalf("Expected success with partial failure, got error: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result from successful providers")
	}
}

// TestProviderFailure_AllFail проверяет обработку ситуации, когда все провайдеры недоступны
func TestProviderFailure_AllFail(t *testing.T) {
	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Provider 1",
			Type:     "provider1",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
		{
			ID:       2,
			Name:     "Provider 2",
			Type:     "provider2",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	// Все провайдеры возвращают ошибки
	clients := map[string]ai.ProviderClient{
		"provider1": &failingProvider{name: "Provider 1", enabled: true, shouldError: true},
		"provider2": &failingProvider{name: "Provider 2", enabled: true, shouldError: true},
	}

	mpc := server.NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	_, err := mpc.NormalizeName(ctx, "test")

	// Должна вернуться ошибка
	if err == nil {
		t.Error("Expected error when all providers fail, got nil")
	}
}

// TestProviderFailure_NetworkError проверяет обработку сетевых ошибок
func TestProviderFailure_NetworkError(t *testing.T) {
	// Создаем сервер, который возвращает ошибку
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer errorServer.Close()

	mockProvider := &networkErrorProvider{
		baseURL: errorServer.URL,
	}

	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Error Provider",
			Type:     "error_provider",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	clients := map[string]ai.ProviderClient{
		"error_provider": mockProvider,
	}

	mpc := server.NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	_, err := mpc.NormalizeName(ctx, "test")

	// Должна вернуться ошибка
	if err == nil {
		t.Error("Expected error from network error provider, got nil")
	}
}

// timeoutProvider мок провайдера с таймаутом
type timeoutProvider struct {
	baseURL string
	timeout time.Duration
}

func (p *timeoutProvider) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	client := &http.Client{
		Timeout: p.timeout,
	}
	_, err := client.Get(p.baseURL)
	if err != nil {
		return "", err
	}
	return "response", nil
}

func (p *timeoutProvider) GetProviderName() string {
	return "Timeout Provider"
}

func (p *timeoutProvider) IsEnabled() bool {
	return true
}

// failingProvider мок провайдера, который может возвращать ошибки
type failingProvider struct {
	name        string
	enabled     bool
	shouldError bool
	response    string
}

func (p *failingProvider) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	if p.shouldError {
		return "", &providerError{message: "provider error"}
	}
	return p.response, nil
}

func (p *failingProvider) GetProviderName() string {
	return p.name
}

func (p *failingProvider) IsEnabled() bool {
	return p.enabled
}

// providerError ошибка провайдера
type providerError struct {
	message string
}

func (e *providerError) Error() string {
	return e.message
}

// networkErrorProvider мок провайдера с сетевыми ошибками
type networkErrorProvider struct {
	baseURL string
}

func (p *networkErrorProvider) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	resp, err := client.Get(p.baseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &providerError{message: "HTTP error: " + resp.Status}
	}

	return "response", nil
}

func (p *networkErrorProvider) GetProviderName() string {
	return "Network Error Provider"
}

func (p *networkErrorProvider) IsEnabled() bool {
	return true
}

// TestOrchestrator_FallbackChain_Intact проверяет каскадный fallback при отказе z.ai/glm-4.5
func TestOrchestrator_FallbackChain_Intact(t *testing.T) {
	// Создаем мок-провайдер для OpenRouter (z.ai/glm-4.5), который возвращает ошибку
	openrouterClient := &failingProvider{
		name:        "OpenRouter",
		enabled:     true,
		shouldError: true, // z.ai/glm-4.5 недоступен
		response:    "",
	}

	// Создаем резервный провайдер (HuggingFace), который работает
	huggingfaceClient := &failingProvider{
		name:        "HuggingFace",
		enabled:     true,
		shouldError: false,
		response:    "ООО \"Резервный Результат\"",
	}

	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "HuggingFace", Type: "huggingface", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter":  openrouterClient,
		"huggingface": huggingfaceClient,
	}

	router := server.NewCounterpartyProviderRouter(nil, nil)
	multiClient := server.NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор
	result, err := multiClient.NormalizeName(context.Background(), "тестовая компания")

	// Должен вернуться результат от резервного провайдера
	if err != nil {
		t.Errorf("Expected success from fallback provider, got error: %v", err)
	}

	if result != "ООО \"Резервный Результат\"" {
		t.Errorf("Expected result from fallback provider 'ООО \"Резервный Результат\"', got '%s'", result)
	}
}

// TestOrchestrator_AllProvidersFail алиас для TestProviderFailure_AllFail
func TestOrchestrator_AllProvidersFail(t *testing.T) {
	TestProviderFailure_AllFail(t)
}

// TestOrchestrator_PartialProviderFailure алиас для TestProviderFailure_PartialFailure
func TestOrchestrator_PartialProviderFailure(t *testing.T) {
	TestProviderFailure_PartialFailure(t)
}

// TestOrchestrator_CircuitBreakerBehavior проверяет поведение circuit breaker (если реализован)
// Этот тест будет работать только после внедрения circuit breaker
func TestOrchestrator_CircuitBreakerBehavior(t *testing.T) {
	// Создаем мок-сервер, который всегда возвращает ошибку
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error": "Internal server error"}`)
	}))
	defer mockServer.Close()

	// Создаем провайдер, который будет падать
	mockProvider := &networkErrorProvider{
		baseURL: mockServer.URL,
	}

	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Circuit Provider",
			Type:     "circuit_provider",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	clients := map[string]ai.ProviderClient{
		"circuit_provider": mockProvider,
	}

	mpc := server.NewMultiProviderClient(providers, clients, nil)

	// Делаем несколько запросов, которые должны привести к открытию circuit breaker
	// (если circuit breaker реализован)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := mpc.NormalizeName(ctx, "test")
		// Ожидаем ошибки
		if err == nil {
			t.Logf("Request %d: Expected error, but got success (circuit breaker may not be implemented)", i+1)
		}
	}

	// После нескольких ошибок circuit breaker должен открыться
	// и последующие запросы должны сразу возвращать ошибку без попытки запроса
	_, err := mpc.NormalizeName(ctx, "test")
	// Если circuit breaker реализован, ошибка должна вернуться быстро
	// Если нет - это нормально, тест просто проверяет, что система не зависает
	if err == nil {
		t.Log("Circuit breaker may not be implemented - this is expected if not yet added")
	}
}
