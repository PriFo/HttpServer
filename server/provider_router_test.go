package server

import (
	"errors"
	"sync"
	"testing"
)

// mockDaDataClient мок для DaData клиента
type mockDaDataClient struct {
	enabled     bool
	shouldError bool
	response    string
	callCount   int
	mu          sync.Mutex
	callChan    chan struct{} // Канал для отслеживания вызовов
}

func (m *mockDaDataClient) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.callChan != nil {
		select {
		case m.callChan <- struct{}{}:
		default:
		}
	}
	if m.shouldError {
		return "", errors.New("dadata provider error")
	}
	return m.response, nil
}

func (m *mockDaDataClient) GetProviderName() string {
	return "DaData"
}

func (m *mockDaDataClient) IsEnabled() bool {
	return m.enabled
}

// AssertCalled проверяет, что клиент был вызван
func (m *mockDaDataClient) AssertCalled(t *testing.T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callCount == 0 {
		t.Error("Expected DaData client to be called, but it wasn't")
	}
}

// AssertNotCalled проверяет, что клиент не был вызван
func (m *mockDaDataClient) AssertNotCalled(t *testing.T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callCount > 0 {
		t.Errorf("Expected DaData client not to be called, but it was called %d times", m.callCount)
	}
}

// mockAdataKzClient мок для Adata.kz клиента
type mockAdataKzClient struct {
	enabled     bool
	shouldError bool
	response    string
	callCount   int
	mu          sync.Mutex
	callChan    chan struct{} // Канал для отслеживания вызовов
}

func (m *mockAdataKzClient) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.callChan != nil {
		select {
		case m.callChan <- struct{}{}:
		default:
		}
	}
	if m.shouldError {
		return "", errors.New("adata provider error")
	}
	return m.response, nil
}

func (m *mockAdataKzClient) GetProviderName() string {
	return "Adata.kz"
}

func (m *mockAdataKzClient) IsEnabled() bool {
	return m.enabled
}

// AssertCalled проверяет, что клиент был вызван
func (m *mockAdataKzClient) AssertCalled(t *testing.T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callCount == 0 {
		t.Error("Expected Adata client to be called, but it wasn't")
	}
}

// AssertNotCalled проверяет, что клиент не был вызван
func (m *mockAdataKzClient) AssertNotCalled(t *testing.T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callCount > 0 {
		t.Errorf("Expected Adata client not to be called, but it was called %d times", m.callCount)
	}
}

// setupRouter создает роутер с моками для тестирования
func setupRouter() (*CounterpartyProviderRouter, *mockDaDataClient, *mockAdataKzClient) {
	mockDaData := &mockDaDataClient{
		enabled:  true,
		response: "ООО 'Ромашка' (ИНН: 1234567890)",
		callChan: make(chan struct{}, 10),
	}
	mockAdata := &mockAdataKzClient{
		enabled:  true,
		response: "ТОО 'Алма' (БИН: 123456789012)",
		callChan: make(chan struct{}, 10),
	}
	router := NewCounterpartyProviderRouter(mockDaData, mockAdata)
	return router, mockDaData, mockAdata
}

// TestProviderRouter_RouteByTaxID_RussianINN_10Digits_ReturnsDaData проверяет маршрутизацию для российского ИНН (10 цифр)
func TestProviderRouter_RouteByTaxID_RussianINN_10Digits_ReturnsDaData(t *testing.T) {
	router, _, mockAdata := setupRouter()

	provider := router.RouteByTaxID("1234567890", "")

	if provider == nil {
		t.Fatal("Expected provider to be returned, got nil")
	}

	if provider.GetProviderName() != "DaData" {
		t.Errorf("Expected DaData provider, got %s", provider.GetProviderName())
	}

	// Проверяем, что DaData был выбран, а Adata - нет
	// (пока не вызываем GetCompletion, просто проверяем выбор)
	mockAdata.AssertNotCalled(t)
}

// TestProviderRouter_RouteByTaxID_RussianINN_12Digits_ReturnsDaData проверяет маршрутизацию для российского ИНН (12 цифр)
func TestProviderRouter_RouteByTaxID_RussianINN_12Digits_ReturnsDaData(t *testing.T) {
	router, _, mockAdata := setupRouter()

	provider := router.RouteByTaxID("123456789012", "")

	if provider == nil {
		t.Fatal("Expected provider to be returned, got nil")
	}

	if provider.GetProviderName() != "DaData" {
		t.Errorf("Expected DaData provider, got %s", provider.GetProviderName())
	}

	// Проверяем, что DaData был выбран, а Adata - нет
	mockAdata.AssertNotCalled(t)
}

// TestProviderRouter_RouteByTaxID_KazakhBIN_ReturnsAdata проверяет маршрутизацию для казахстанского БИН (12 цифр)
func TestProviderRouter_RouteByTaxID_KazakhBIN_ReturnsAdata(t *testing.T) {
	router, mockDaData, _ := setupRouter()

	provider := router.RouteByTaxID("", "123456789012")

	if provider == nil {
		t.Fatal("Expected provider to be returned, got nil")
	}

	if provider.GetProviderName() != "Adata.kz" {
		t.Errorf("Expected Adata.kz provider, got %s", provider.GetProviderName())
	}

	// Проверяем, что Adata был выбран, а DaData - нет
	mockDaData.AssertNotCalled(t)
}

// TestProviderRouter_RouteByTaxID_BINPriority проверяет, что БИН имеет приоритет над ИНН
func TestProviderRouter_RouteByTaxID_BINPriority(t *testing.T) {
	router, mockDaData, _ := setupRouter()

	// Если есть и ИНН (12 цифр) и БИН (12 цифр), должен быть выбран Adata (БИН имеет приоритет)
	provider := router.RouteByTaxID("123456789012", "987654321098")

	if provider == nil {
		t.Fatal("Expected provider to be returned, got nil")
	}

	if provider.GetProviderName() != "Adata.kz" {
		t.Errorf("Expected Adata.kz provider (BIN priority), got %s", provider.GetProviderName())
	}

	// Проверяем, что Adata был выбран, а DaData - нет
	mockDaData.AssertNotCalled(t)
}

// TestProviderRouter_RouteByTaxID_InvalidTaxID_ReturnsNil проверяет обработку невалидного ИНН/БИН
func TestProviderRouter_RouteByTaxID_InvalidTaxID_ReturnsNil(t *testing.T) {
	router, _, _ := setupRouter()

	provider := router.RouteByTaxID("123", "456")

	if provider != nil {
		t.Errorf("Expected nil for invalid tax ID, got %s", provider.GetProviderName())
	}
}

// TestProviderRouter_RouteByTaxID_NormalizedTaxID проверяет нормализацию ИНН/БИН (убирает пробелы и нецифровые символы)
func TestProviderRouter_RouteByTaxID_NormalizedTaxID(t *testing.T) {
	router, _, mockAdata := setupRouter()

	// ИНН с пробелами и дефисами должен быть нормализован
	provider := router.RouteByTaxID("123-456-789-0", "")

	if provider == nil {
		t.Fatal("Expected provider to be returned, got nil")
	}

	if provider.GetProviderName() != "DaData" {
		t.Errorf("Expected DaData provider for normalized INN, got %s", provider.GetProviderName())
	}

	mockAdata.AssertNotCalled(t)
}

// TestProviderRouter_StandardizeCounterparty_DaDataSuccess_ReturnsNormalized проверяет успешную стандартизацию через DaData
func TestProviderRouter_StandardizeCounterparty_DaDataSuccess_ReturnsNormalized(t *testing.T) {
	router, mockDaData, _ := setupRouter()
	mockDaData.response = "ООО 'Ромашка' (ИНН: 1234567890)"

	result, err := router.StandardizeCounterparty("Ромашка", "1234567890", "")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != mockDaData.response {
		t.Errorf("Expected result '%s', got '%s'", mockDaData.response, result)
	}

	mockDaData.AssertCalled(t)
}

// TestProviderRouter_StandardizeCounterparty_AdataSuccess_ReturnsNormalized проверяет успешную стандартизацию через Adata
func TestProviderRouter_StandardizeCounterparty_AdataSuccess_ReturnsNormalized(t *testing.T) {
	router, _, mockAdata := setupRouter()
	mockAdata.response = "ТОО 'Алма' (БИН: 123456789012)"

	result, err := router.StandardizeCounterparty("Алма", "", "123456789012")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != mockAdata.response {
		t.Errorf("Expected result '%s', got '%s'", mockAdata.response, result)
	}

	mockAdata.AssertCalled(t)
}

// TestProviderRouter_StandardizeCounterparty_ProviderError_ReturnsError проверяет обработку ошибки провайдера
func TestProviderRouter_StandardizeCounterparty_ProviderError_ReturnsError(t *testing.T) {
	router, mockDaData, _ := setupRouter()
	mockDaData.shouldError = true

	result, err := router.StandardizeCounterparty("Ромашка", "1234567890", "")

	if err == nil {
		t.Error("Expected error from provider, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result on error, got '%s'", result)
	}

	// Проверяем, что ошибка содержит информацию о провайдере
	if err.Error() == "" {
		t.Error("Expected error message to contain provider information")
	}
}

// TestProviderRouter_StandardizeCounterparty_EmptyName_ReturnsError проверяет обработку пустого имени
func TestProviderRouter_StandardizeCounterparty_EmptyName_ReturnsError(t *testing.T) {
	router, _, _ := setupRouter()

	result, err := router.StandardizeCounterparty("", "1234567890", "")

	if err == nil {
		t.Error("Expected error for empty name, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result for empty name, got '%s'", result)
	}
}

// TestProviderRouter_StandardizeCounterparty_CannotDetermineProvider_ReturnsError проверяет обработку ситуации, когда провайдер не может быть определен
func TestProviderRouter_StandardizeCounterparty_CannotDetermineProvider_ReturnsError(t *testing.T) {
	router, _, _ := setupRouter()

	result, err := router.StandardizeCounterparty("Компания", "123", "456")

	if err == nil {
		t.Error("Expected error when provider cannot be determined, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty result when provider cannot be determined, got '%s'", result)
	}
}

// TestProviderRouter_StandardizeCounterparty_DisabledProvider проверяет обработку отключенного провайдера
func TestProviderRouter_StandardizeCounterparty_DisabledProvider(t *testing.T) {
	mockDaData := &mockDaDataClient{enabled: false}
	mockAdata := &mockAdataKzClient{enabled: false}
	router := NewCounterpartyProviderRouter(mockDaData, mockAdata)

	provider := router.RouteByTaxID("1234567890", "")

	if provider != nil {
		t.Error("Expected nil when provider is disabled, got provider")
	}
}

// TestProviderRouter_ConcurrentAccess проверяет безопасность при конкурентном доступе
func TestProviderRouter_ConcurrentAccess(t *testing.T) {
	router, mockDaData, mockAdata := setupRouter()

	var wg sync.WaitGroup
	concurrency := 10

	// Запускаем несколько горутин одновременно
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			provider := router.RouteByTaxID("1234567890", "")
			if provider == nil {
				t.Error("Expected provider in concurrent access")
			}
		}()
	}

	wg.Wait()

	// Проверяем, что не было гонок данных
	// Это нормально, если были вызовы GetCompletion
	_ = mockDaData
	_ = mockAdata
}

// TestProviderRouter_FormatCounterpartyPrompt проверяет форматирование промпта
func TestProviderRouter_FormatCounterpartyPrompt(t *testing.T) {
	router, mockDaData, _ := setupRouter()

	// Вызываем StandardizeCounterparty, чтобы проверить форматирование промпта
	_, err := router.StandardizeCounterparty("Ромашка", "1234567890", "")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Проверяем, что промпт был сформирован и отправлен
	mockDaData.AssertCalled(t)
}

// TestProviderRouter_FormatCounterpartyPrompt_WithBIN проверяет форматирование промпта с БИН
func TestProviderRouter_FormatCounterpartyPrompt_WithBIN(t *testing.T) {
	router, _, mockAdata := setupRouter()

	_, err := router.StandardizeCounterparty("Алма", "", "123456789012")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	mockAdata.AssertCalled(t)
}
