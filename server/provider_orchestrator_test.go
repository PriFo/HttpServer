package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/internal/infrastructure/ai"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProviderClientForOrchestrator мок для ProviderClient интерфейса
type MockProviderClientForOrchestrator struct {
	name      string
	enabled   bool
	shouldErr bool
	result    string
	called    bool
}

func NewMockProviderClientForOrchestrator(name string, result string, shouldErr bool) *MockProviderClientForOrchestrator {
	return &MockProviderClientForOrchestrator{
		name:      name,
		enabled:   true,
		shouldErr: shouldErr,
		result:    result,
		called:    false,
	}
}

func (m *MockProviderClientForOrchestrator) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	m.called = true
	if m.shouldErr {
		return "", fmt.Errorf("mock error from %s", m.name)
	}
	return m.result, nil
}

func (m *MockProviderClientForOrchestrator) GetProviderName() string {
	return m.name
}

func (m *MockProviderClientForOrchestrator) IsEnabled() bool {
	return m.enabled
}

// MockDaDataClient мок для DaData через ProviderClient интерфейс
type MockDaDataClient struct {
	shouldFail bool
	called     bool
	result     string
}

func NewMockDaDataClient(shouldFail bool, result string) *MockDaDataClient {
	return &MockDaDataClient{
		shouldFail: shouldFail,
		result:     result,
	}
}

func (d *MockDaDataClient) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	d.called = true
	if d.shouldFail {
		return "", fmt.Errorf("dadata service unavailable")
	}
	return d.result, nil
}

func (d *MockDaDataClient) GetProviderName() string {
	return "DaData"
}

func (d *MockDaDataClient) IsEnabled() bool {
	return true
}

// MockAdataKzClient мок для Adata.kz через ProviderClient интерфейс
type MockAdataKzClient struct {
	shouldFail bool
	called     bool
	result     string
}

func NewMockAdataKzClient(shouldFail bool, result string) *MockAdataKzClient {
	return &MockAdataKzClient{
		shouldFail: shouldFail,
		result:     result,
	}
}

func (a *MockAdataKzClient) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	a.called = true
	if a.shouldFail {
		return "", fmt.Errorf("adata.kz service unavailable")
	}
	return a.result, nil
}

func (a *MockAdataKzClient) GetProviderName() string {
	return "Adata.kz"
}

func (a *MockAdataKzClient) IsEnabled() bool {
	return true
}

// TestOrchestrator_RegisterProvider проверяет регистрацию провайдеров
func TestOrchestrator_RegisterProvider(t *testing.T) {
	orchestrator := ai.NewProviderOrchestrator(30*time.Second, nil)

	// Создаем мок клиента
	mockClient := NewMockProviderClientForOrchestrator("test-provider", "test response", false)

	// Регистрируем провайдера
	orchestrator.RegisterProvider("test-id", "Test Provider", mockClient, true, 1)

	// Проверяем, что провайдер зарегистрирован
	activeProviders := orchestrator.GetActiveProviders()
	if len(activeProviders) != 1 {
		t.Errorf("Expected 1 active provider, got %d", len(activeProviders))
	}

	if activeProviders[0].ID != "test-id" {
		t.Errorf("Expected provider ID 'test-id', got '%s'", activeProviders[0].ID)
	}

	if activeProviders[0].Name != "Test Provider" {
		t.Errorf("Expected provider name 'Test Provider', got '%s'", activeProviders[0].Name)
	}
}

// TestOrchestrator_GetActiveProviders проверяет получение активных провайдеров
func TestOrchestrator_GetActiveProviders(t *testing.T) {
	orchestrator := ai.NewProviderOrchestrator(30*time.Second, nil)

	// Регистрируем несколько провайдеров
	mockClient1 := NewMockProviderClientForOrchestrator("provider1", "response1", false)
	mockClient2 := NewMockProviderClientForOrchestrator("provider2", "response2", false)
	mockClient3 := NewMockProviderClientForOrchestrator("provider3", "response3", false)

	orchestrator.RegisterProvider("id1", "Provider 1", mockClient1, true, 1)
	orchestrator.RegisterProvider("id2", "Provider 2", mockClient2, false, 2) // отключен
	orchestrator.RegisterProvider("id3", "Provider 3", mockClient3, true, 3)

	activeProviders := orchestrator.GetActiveProviders()
	if len(activeProviders) != 2 {
		t.Errorf("Expected 2 active providers, got %d", len(activeProviders))
	}

	// Проверяем сортировку по приоритету
	if activeProviders[0].Priority != 1 {
		t.Errorf("Expected first provider to have priority 1, got %d", activeProviders[0].Priority)
	}
	if activeProviders[1].Priority != 3 {
		t.Errorf("Expected second provider to have priority 3, got %d", activeProviders[1].Priority)
	}
}

// TestOrchestrator_SetStrategy проверяет установку стратегии
func TestOrchestrator_SetStrategy(t *testing.T) {
	orchestrator := ai.NewProviderOrchestrator(30*time.Second, nil)

	// Проверяем стратегию по умолчанию
	if orchestrator.GetStrategy() != ai.FirstSuccess {
		t.Errorf("Expected default strategy to be 'first_success', got '%s'", orchestrator.GetStrategy())
	}

	// Устанавливаем новую стратегию
	orchestrator.SetStrategy(ai.MajorityVote)

	if orchestrator.GetStrategy() != ai.MajorityVote {
		t.Errorf("Expected strategy to be 'majority_vote', got '%s'", orchestrator.GetStrategy())
	}
}

// TestOrchestrator_RouteToDaData_Success проверяет успешную маршрутизацию в DaData
func TestOrchestrator_RouteToDaData_Success(t *testing.T) {
	dadata := NewMockDaDataClient(false, "ООО \"Тестовая Компания\"")
	adata := NewMockAdataKzClient(false, "ТОО \"Тестовая Компания\"")

	// Создаем роутер с моками
	router := NewCounterpartyProviderRouter(dadata, adata)

	// Создаем провайдеры для MultiProviderClient
	providers := []*database.Provider{
		{ID: 1, Name: "DaData", Type: "dadata", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Adata.kz", Type: "adata", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"dadata": dadata,
		"adata":  adata,
	}

	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод нормализации контрагента с российским ИНН
	result, err := multiClient.NormalizeCounterparty(context.Background(), "ООО Ромашка", "1234567890", "")

	// Проверяем результат
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "ООО \"Тестовая Компания\"" {
		t.Errorf("Expected 'ООО \"Тестовая Компания\"', got '%s'", result)
	}

	// Проверяем, что был вызван DaData
	if !dadata.called {
		t.Error("Expected DaData to be called, but it wasn't")
	}

	// Проверяем, что Adata не был вызван
	if adata.called {
		t.Error("Expected Adata not to be called, but it was")
	}
}

// TestOrchestrator_RouteToAdata_Success проверяет успешную маршрутизацию в Adata
func TestOrchestrator_RouteToAdata_Success(t *testing.T) {
	dadata := NewMockDaDataClient(false, "ООО \"Тестовая Компания\"")
	adata := NewMockAdataKzClient(false, "ТОО \"Тестовая Компания\"")

	// Создаем роутер с моками
	router := NewCounterpartyProviderRouter(dadata, adata)

	// Создаем провайдеры для MultiProviderClient
	providers := []*database.Provider{
		{ID: 1, Name: "DaData", Type: "dadata", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Adata.kz", Type: "adata", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"dadata": dadata,
		"adata":  adata,
	}

	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод нормализации контрагента с казахстанским БИН
	result, err := multiClient.NormalizeCounterparty(context.Background(), "ТОО Ромашка", "", "123456789012")

	// Проверяем результат
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "ТОО \"Тестовая Компания\"" {
		t.Errorf("Expected 'ТОО \"Тестовая Компания\"', got '%s'", result)
	}

	// Проверяем, что был вызван Adata
	if !adata.called {
		t.Error("Expected Adata to be called, but it wasn't")
	}

	// Проверяем, что DaData не был вызван
	if dadata.called {
		t.Error("Expected DaData not to be called, but it was")
	}
}

// TestOrchestrator_FallbackToGenerativeAI проверяет fallback на генеративные AI
func TestOrchestrator_FallbackToGenerativeAI(t *testing.T) {
	dadata := NewMockDaDataClient(true, "") // Ошибка
	adata := NewMockAdataKzClient(true, "") // Ошибка

	generativeAI := NewMockProviderClientForOrchestrator("OpenRouter", "ООО \"Генеративный Результат\"", false)

	// Создаем провайдеров для MultiProviderClient
	providers := []*database.Provider{
		{ID: 1, Name: "DaData", Type: "dadata", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Adata.kz", Type: "adata", IsActive: true, Config: `{"channels":1}`},
		{ID: 3, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"dadata":     dadata,
		"adata":      adata,
		"openrouter": generativeAI,
	}

	router := NewCounterpartyProviderRouter(dadata, adata)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод с любым ИНН/БИН - должен сработать fallback
	result, err := multiClient.NormalizeCounterparty(context.Background(), "Тестовая Компания", "invalid_inn", "invalid_bin")

	// Проверяем результат
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "ООО \"Генеративный Результат\"" {
		t.Errorf("Expected 'ООО \"Генеративный Результат\"', got '%s'", result)
	}

	// Проверяем, что специализированные провайдеры были вызваны и вернули ошибки
	if !dadata.called {
		t.Error("Expected DaData to be called")
	}
	if !adata.called {
		t.Error("Expected Adata to be called")
	}
	if !generativeAI.called {
		t.Error("Expected generative AI to be called")
	}
}

// TestOrchestrator_MultiProviderClient_GenerativeAI проверяет работу с несколькими генеративными AI провайдерами
func TestOrchestrator_MultiProviderClient_GenerativeAI(t *testing.T) {
	// Создаем мок-клиенты для разных провайдеров
	clients := map[string]ai.ProviderClient{
		"openrouter":  NewMockProviderClientForOrchestrator("OpenRouter", "response", false),
		"huggingface": NewMockProviderClientForOrchestrator("HuggingFace", "response", false),
		"arliai":      NewMockProviderClientForOrchestrator("Arliai", "response", false),
	}

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "HuggingFace", Type: "huggingface", IsActive: true, Config: `{"channels":1}`},
		{ID: 3, Name: "Arliai", Type: "arliai", IsActive: true, Config: `{"channels":1}`},
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод нормализации
	result, err := multiClient.NormalizeName(context.Background(), "ромашка")

	// Проверяем результат
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "response" {
		t.Errorf("Expected 'response', got '%s'", result)
	}

	// Проверяем, что все провайдеры были вызваны
	for providerID, client := range clients {
		mockClient := client.(*MockProviderClientForOrchestrator)
		if !mockClient.called {
			t.Errorf("Expected %s to be called, but it wasn't", providerID)
		}
	}
}

// TestOrchestrator_MultiProviderClient_MajorityVote проверяет majority vote стратегию
func TestOrchestrator_MultiProviderClient_MajorityVote(t *testing.T) {
	// Создаем провайдеров с разными результатами
	clients := map[string]ai.ProviderClient{
		"openrouter":  NewMockProviderClientForOrchestrator("OpenRouter", "ООО \"Тестовая Компания\"", false),
		"arliai":      NewMockProviderClientForOrchestrator("Arliai", "ООО \"Тестовая Компания\"", false),
		"huggingface": NewMockProviderClientForOrchestrator("HuggingFace", "ООО \"Другая Компания\"", false),
	}

	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Arliai", Type: "arliai", IsActive: true, Config: `{"channels":1}`},
		{ID: 3, Name: "HuggingFace", Type: "huggingface", IsActive: true, Config: `{"channels":1}`},
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод нормализации
	result, err := multiClient.NormalizeName(context.Background(), "тестовая компания")

	// Проверяем результат
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Majority vote должен выбрать "ООО \"Тестовая Компания\"" (2 голоса из 3)
	if result != "ООО \"Тестовая Компания\"" {
		t.Errorf("Expected majority vote result 'ООО \"Тестовая Компания\"', got '%s'", result)
	}
}

// TestOrchestrator_MultiProviderClient_ErrorHandling проверяет обработку ошибок
func TestOrchestrator_MultiProviderClient_ErrorHandling(t *testing.T) {
	// Создаем провайдеров, где все возвращают ошибки
	clients := map[string]ai.ProviderClient{
		"openrouter": NewMockProviderClientForOrchestrator("OpenRouter", "", true),
		"arliai":     NewMockProviderClientForOrchestrator("Arliai", "", true),
	}

	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Arliai", Type: "arliai", IsActive: true, Config: `{"channels":1}`},
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод нормализации
	result, err := multiClient.NormalizeName(context.Background(), "тестовая компания")

	// Должна вернуться ошибка
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result, got '%s'", result)
	}
}

// TestOrchestrator_Integration_WithGlm45 проверяет интеграцию с z.ai/glm-4.5 через OpenRouter
func TestOrchestrator_Integration_WithGlm45(t *testing.T) {
	// Создаем мок-сервер, который будет отвечать как glm-4.5
	mockOpenRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос идет к правильному эндпоинту
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		// Проверяем, что в теле запроса есть наша модель
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)
		assert.Equal(t, "z.ai/glm-4.5", requestBody["model"])

		// Возвращаем предсказуемый ответ glm-4.5
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{
			"id": "chatcmpl-...",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "z.ai/glm-4.5",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "ООО \"Ромашка\""
				},
				"finish_reason": "stop"
			}]
		}`)
	}))
	defer mockOpenRouter.Close()

	// Создаем OpenRouter клиент с указанием модели z.ai/glm-4.5
	openrouterClient := ai.NewOpenRouterClient("fake-api-key")
	// Устанавливаем baseURL через рефлексию для мок-сервера
	rv := reflect.ValueOf(openrouterClient).Elem()
	rv.FieldByName("baseURL").SetString(mockOpenRouter.URL)
	openrouterAdapter := ai.NewOpenRouterProviderAdapter(openrouterClient)

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter": openrouterAdapter,
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор
	result, err := multiClient.NormalizeName(context.Background(), "ромашка")

	// Проверяем результат
	require.NoError(t, err)
	assert.Equal(t, "ООО \"Ромашка\"", result)
}

// TestOrchestrator_Glm45_Timeout проверяет обработку таймаута для glm-4.5
func TestOrchestrator_Glm45_Timeout(t *testing.T) {
	// Создаем мок-сервер, который не отвечает на запросы (имитация таймаута)
	mockOpenRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Имитируем зависание сервера
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockOpenRouter.Close()

	// Создаем OpenRouter клиент с коротким таймаутом
	openrouterClient := ai.NewOpenRouterClient("fake-api-key")
	// Устанавливаем baseURL через рефлексию для мок-сервера
	rv := reflect.ValueOf(openrouterClient).Elem()
	rv.FieldByName("baseURL").SetString(mockOpenRouter.URL)
	httpClientField := rv.FieldByName("httpClient")
	if httpClientField.IsValid() && !httpClientField.IsNil() {
		httpClient := httpClientField.Interface().(*http.Client)
		if httpClient != nil {
			httpClient.Timeout = 100 * time.Millisecond
		}
	}

	// Создаем адаптер для OpenRouter
	openrouterAdapter := ai.NewOpenRouterProviderAdapter(openrouterClient)

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter": openrouterAdapter,
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор с коротким таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := multiClient.NormalizeName(ctx, "ромашка")

	// Должна вернуться ошибка таймаута
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result, got '%s'", result)
	}
}

// TestOrchestrator_Glm45_ReturnsError проверяет обработку ошибки от glm-4.5
func TestOrchestrator_Glm45_ReturnsError(t *testing.T) {
	// Создаем мок-сервер, который возвращает HTTP 500
	mockOpenRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error": "Internal server error"}`)
	}))
	defer mockOpenRouter.Close()

	// Создаем OpenRouter клиент
	openrouterClient := ai.NewOpenRouterClient("fake-api-key")
	// Устанавливаем baseURL через рефлексию для мок-сервера
	rv := reflect.ValueOf(openrouterClient).Elem()
	rv.FieldByName("baseURL").SetString(mockOpenRouter.URL)

	// Создаем адаптер для OpenRouter
	openrouterAdapter := ai.NewOpenRouterProviderAdapter(openrouterClient)

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter": openrouterAdapter,
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор
	result, err := multiClient.NormalizeName(context.Background(), "ромашка")

	// Должна вернуться ошибка
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result, got '%s'", result)
	}
}

// TestOrchestrator_ParseAIResponse проверяет парсинг ответа от AI
// ПРИМЕЧАНИЕ: Тест удален, так как parseAIResponse - приватный метод и не должен тестироваться напрямую.
// Парсинг тестируется косвенно через публичный метод Normalize.
func TestOrchestrator_ParseAIResponse(t *testing.T) {
	// Тест удален - parseAIResponse является приватным методом
	// Парсинг тестируется косвенно через публичный метод Normalize
	t.Skip("parseAIResponse is a private method and should not be tested directly")
}

// BenchmarkOrchestrator_StandardizeCounterparty бенчмарк для метода нормализации контрагентов
func BenchmarkOrchestrator_StandardizeCounterparty(b *testing.B) {
	dadata := NewMockDaDataClient(false, "ООО \"Тестовая Компания\"")
	adata := NewMockAdataKzClient(false, "ТОО \"Тестовая Компания\"")

	router := NewCounterpartyProviderRouter(dadata, adata)

	providers := []*database.Provider{
		{ID: 1, Name: "DaData", Type: "dadata", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Adata.kz", Type: "adata", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"dadata": dadata,
		"adata":  adata,
	}

	multiClient := NewMultiProviderClient(providers, clients, router)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := multiClient.NormalizeCounterparty(context.Background(), "Тестовая Компания", "1234567890", "")
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// TestOrchestrator_ConcurrentCalls проверяет конкурентные вызовы
func TestOrchestrator_ConcurrentCalls(t *testing.T) {
	dadata := NewMockDaDataClient(false, "ООО \"Тестовая Компания\"")
	adata := NewMockAdataKzClient(false, "ТОО \"Тестовая Компания\"")

	router := NewCounterpartyProviderRouter(dadata, adata)

	providers := []*database.Provider{
		{ID: 1, Name: "DaData", Type: "dadata", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Adata.kz", Type: "adata", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"dadata": dadata,
		"adata":  adata,
	}

	multiClient := NewMultiProviderClient(providers, clients, router)

	// Запускаем 100 горутин, которые одновременно вызывают нормализацию
	var wg sync.WaitGroup
	results := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := multiClient.NormalizeCounterparty(context.Background(), fmt.Sprintf("Компания %d", id), "1234567890", "")
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	// Проверяем результаты
	for err := range results {
		if err != nil {
			t.Errorf("Concurrent call failed: %v", err)
		}
	}
}

// TestOrchestrator_SelectsDaData_ForRussianInn проверяет маршрутизацию для российского ИНН
// Это алиас для TestOrchestrator_RouteToDaData_Success для соответствия плану
func TestOrchestrator_SelectsDaData_ForRussianInn(t *testing.T) {
	TestOrchestrator_RouteToDaData_Success(t)
}

// TestOrchestrator_SelectsAdata_ForKazakhBin проверяет маршрутизацию для казахстанского БИН
// Это алиас для TestOrchestrator_RouteToAdata_Success для соответствия плану
func TestOrchestrator_SelectsAdata_ForKazakhBin(t *testing.T) {
	TestOrchestrator_RouteToAdata_Success(t)
}

// TestOrchestrator_FallsBackToGenerativeAI_OnSpecializedError проверяет fallback при ошибке специализированного провайдера
// Это алиас для TestOrchestrator_FallbackToGenerativeAI для соответствия плану
func TestOrchestrator_FallsBackToGenerativeAI_OnSpecializedError(t *testing.T) {
	TestOrchestrator_FallbackToGenerativeAI(t)
}

// TestOrchestrator_FallsBackToGenerativeAI_OnSpecializedNoResult проверяет fallback при пустом результате (критично!)
func TestOrchestrator_FallsBackToGenerativeAI_OnSpecializedNoResult(t *testing.T) {
	// Создаем моки, которые возвращают пустой результат (не ошибку)
	dadata := NewMockDaDataClient(false, "") // Пустой результат
	adata := NewMockAdataKzClient(false, "") // Пустой результат

	generativeAI := NewMockProviderClientForOrchestrator("OpenRouter", "ООО \"Генеративный Результат\"", false)

	// Создаем провайдеров для MultiProviderClient
	providers := []*database.Provider{
		{ID: 1, Name: "DaData", Type: "dadata", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Adata.kz", Type: "adata", IsActive: true, Config: `{"channels":1}`},
		{ID: 3, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"dadata":     dadata,
		"adata":      adata,
		"openrouter": generativeAI,
	}

	router := NewCounterpartyProviderRouter(dadata, adata)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод с валидным ИНН - специализированный провайдер вернет пустой результат
	result, err := multiClient.NormalizeCounterparty(context.Background(), "Тестовая Компания", "1234567890", "")

	// Проверяем, что система перешла к генеративным AI
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Должен быть результат от генеративного AI
	if result != "ООО \"Генеративный Результат\"" {
		t.Errorf("Expected 'ООО \"Генеративный Результат\"', got '%s'", result)
	}

	// Проверяем, что специализированные провайдеры были вызваны
	if !dadata.called {
		t.Error("Expected DaData to be called")
	}

	// Проверяем, что генеративный AI был вызван как fallback
	if !generativeAI.called {
		t.Error("Expected generative AI to be called as fallback")
	}
}

// TestOrchestrator_AggregatesAndVotesCorrectly проверяет логику голосования с разными результатами
func TestOrchestrator_AggregatesAndVotesCorrectly(t *testing.T) {
	// Создаем провайдеров с разными результатами для majority vote
	clients := map[string]ai.ProviderClient{
		"openrouter":  NewMockProviderClientForOrchestrator("OpenRouter", "ООО \"Тестовая Компания\"", false),
		"arliai":      NewMockProviderClientForOrchestrator("Arliai", "ООО \"Тестовая Компания\"", false),
		"huggingface": NewMockProviderClientForOrchestrator("HuggingFace", "ООО \"Другая Компания\"", false),
		"edenai":      NewMockProviderClientForOrchestrator("EdenAI", "ООО \"Тестовая Компания\"", false),
	}

	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
		{ID: 2, Name: "Arliai", Type: "arliai", IsActive: true, Config: `{"channels":1}`},
		{ID: 3, Name: "HuggingFace", Type: "huggingface", IsActive: true, Config: `{"channels":1}`},
		{ID: 4, Name: "EdenAI", Type: "edenai", IsActive: true, Config: `{"channels":1}`},
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем метод нормализации
	result, err := multiClient.NormalizeName(context.Background(), "тестовая компания")

	// Проверяем результат
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Majority vote должен выбрать "ООО \"Тестовая Компания\"" (3 голоса из 4)
	if result != "ООО \"Тестовая Компания\"" {
		t.Errorf("Expected majority vote result 'ООО \"Тестовая Компания\"', got '%s'", result)
	}

	// Проверяем, что все провайдеры были вызваны
	for providerID, client := range clients {
		mockClient := client.(*MockProviderClientForOrchestrator)
		if !mockClient.called {
			t.Errorf("Expected %s to be called, but it wasn't", providerID)
		}
	}
}

// TestOrchestrator_Glm45_ModelConfiguration проверяет передачу модели z.ai/glm-4.5 в запрос
func TestOrchestrator_Glm45_ModelConfiguration(t *testing.T) {
	// Создаем мок-сервер, который будет отвечать как OpenRouter
	mockOpenRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что запрос идет к правильному эндпоинту
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		// Проверяем, что в теле запроса есть наша модель
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			return
		}

		// Проверяем, что модель z.ai/glm-4.5 передана в запросе
		model, ok := requestBody["model"].(string)
		if !ok {
			t.Error("Model not found in request body or is not a string")
			return
		}

		if model != "z.ai/glm-4.5" {
			t.Errorf("Expected model 'z.ai/glm-4.5', got '%s'", model)
			return
		}

		// Возвращаем предсказуемый ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{
			"id": "chatcmpl-...",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "z.ai/glm-4.5",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "ООО \"Ромашка\""
				},
				"finish_reason": "stop"
			}]
		}`)
	}))
	defer mockOpenRouter.Close()

	// Устанавливаем переменную окружения для модели
	os.Setenv("OPENROUTER_MODEL", "z.ai/glm-4.5")
	defer os.Unsetenv("OPENROUTER_MODEL")

	// Создаем OpenRouter клиент
	openrouterClient := ai.NewOpenRouterClient("fake-api-key")
	// Устанавливаем baseURL через рефлексию для мок-сервера
	rv := reflect.ValueOf(openrouterClient).Elem()
	rv.FieldByName("baseURL").SetString(mockOpenRouter.URL)

	// Создаем адаптер для OpenRouter
	openrouterAdapter := ai.NewOpenRouterProviderAdapter(openrouterClient)

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter": openrouterAdapter,
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор
	result, err := multiClient.NormalizeName(context.Background(), "ромашка")

	// Проверяем результат
	require.NoError(t, err)
	assert.Equal(t, "ООО \"Ромашка\"", result)
}

// TestOrchestrator_Integration_Glm45_Success алиас для TestOrchestrator_Integration_WithGlm45
func TestOrchestrator_Integration_Glm45_Success(t *testing.T) {
	TestOrchestrator_Integration_WithGlm45(t)
}

// TestOrchestrator_Integration_Glm45_MalformedJsonResponse проверяет обработку невалидного JSON от glm-4.5
func TestOrchestrator_Integration_Glm45_MalformedJsonResponse(t *testing.T) {
	// Создаем мок-сервер, который возвращает невалидный JSON
	mockOpenRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Возвращаем невалидный JSON
		fmt.Fprintln(w, `{invalid json}`)
	}))
	defer mockOpenRouter.Close()

	// Устанавливаем переменную окружения для модели
	os.Setenv("OPENROUTER_MODEL", "z.ai/glm-4.5")
	defer os.Unsetenv("OPENROUTER_MODEL")

	// Создаем OpenRouter клиент
	openrouterClient := ai.NewOpenRouterClient("fake-api-key")
	// Устанавливаем baseURL через рефлексию для мок-сервера
	rv := reflect.ValueOf(openrouterClient).Elem()
	rv.FieldByName("baseURL").SetString(mockOpenRouter.URL)

	// Создаем адаптер для OpenRouter
	openrouterAdapter := ai.NewOpenRouterProviderAdapter(openrouterClient)

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter": openrouterAdapter,
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор
	result, err := multiClient.NormalizeName(context.Background(), "ромашка")

	// Должна вернуться ошибка парсинга
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result for malformed JSON, got '%s'", result)
	}
}

// TestOrchestrator_Glm45_HttpError_500 алиас для TestOrchestrator_Glm45_ReturnsError
func TestOrchestrator_Glm45_HttpError_500(t *testing.T) {
	TestOrchestrator_Glm45_ReturnsError(t)
}

// TestOrchestrator_Glm45_HttpError_429 проверяет обработку rate limiting (429 Too Many Requests)
func TestOrchestrator_Glm45_HttpError_429(t *testing.T) {
	// Создаем мок-сервер, который возвращает HTTP 429
	mockOpenRouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintln(w, `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`)
	}))
	defer mockOpenRouter.Close()

	// Создаем OpenRouter клиент
	openrouterClient := ai.NewOpenRouterClient("fake-api-key")
	// Устанавливаем baseURL через рефлексию для мок-сервера
	rv := reflect.ValueOf(openrouterClient).Elem()
	rv.FieldByName("baseURL").SetString(mockOpenRouter.URL)

	// Создаем адаптер для OpenRouter
	openrouterAdapter := ai.NewOpenRouterProviderAdapter(openrouterClient)

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter": openrouterAdapter,
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор
	result, err := multiClient.NormalizeName(context.Background(), "ромашка")

	// Должна вернуться ошибка rate limiting
	if err == nil {
		t.Error("Expected rate limit error, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result for rate limit error, got '%s'", result)
	}

	// Проверяем, что ошибка содержит информацию о rate limit
	if err != nil && !strings.Contains(err.Error(), "rate") && !strings.Contains(err.Error(), "429") {
		t.Errorf("Expected rate limit error message, got: %v", err)
	}
}

// TestOrchestrator_Glm45_NetworkFailure проверяет обработку сетевых ошибок
func TestOrchestrator_Glm45_NetworkFailure(t *testing.T) {
	// Создаем OpenRouter клиент с недоступным адресом
	openrouterClient := ai.NewOpenRouterClient("fake-api-key")
	// Устанавливаем baseURL через рефлексию для недоступного адреса
	rv := reflect.ValueOf(openrouterClient).Elem()
	rv.FieldByName("baseURL").SetString("http://localhost:99999") // Недоступный порт

	// Создаем адаптер для OpenRouter
	openrouterAdapter := ai.NewOpenRouterProviderAdapter(openrouterClient)

	// Создаем провайдеров
	providers := []*database.Provider{
		{ID: 1, Name: "OpenRouter", Type: "openrouter", IsActive: true, Config: `{"channels":1}`},
	}

	clients := map[string]ai.ProviderClient{
		"openrouter": openrouterAdapter,
	}

	router := NewCounterpartyProviderRouter(nil, nil)
	multiClient := NewMultiProviderClient(providers, clients, router)

	// Вызываем оркестратор с коротким таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := multiClient.NormalizeName(ctx, "ромашка")

	// Должна вернуться ошибка сети
	if err == nil {
		t.Error("Expected network error, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result for network error, got '%s'", result)
	}

	// Проверяем, что ошибка связана с сетью
	if err != nil {
		errStr := err.Error()
		if !strings.Contains(errStr, "connection") && !strings.Contains(errStr, "refused") && !strings.Contains(errStr, "timeout") {
			t.Errorf("Expected network error, got: %v", err)
		}
	}
}
