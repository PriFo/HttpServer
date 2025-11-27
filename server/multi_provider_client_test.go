package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/internal/infrastructure/ai"
)

// mockProviderClient мок-клиент для тестирования
type mockProviderClient struct {
	name            string
	enabled         bool
	shouldError     bool
	response        string
	responseDelay   time.Duration
	callCount       int
}

func (m *mockProviderClient) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	m.callCount++
	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}
	if m.shouldError {
		return "", errors.New("mock provider error")
	}
	return m.response, nil
}

func (m *mockProviderClient) GetProviderName() string {
	return m.name
}

func (m *mockProviderClient) IsEnabled() bool {
	return m.enabled
}

// TestNewMultiProviderClient проверяет создание мульти-провайдерного клиента
func TestNewMultiProviderClient(t *testing.T) {
	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Test Provider 1",
			Type:     "test1",
			IsActive: true,
			Config:   `{"channels":2}`,
		},
		{
			ID:       2,
			Name:     "Test Provider 2",
			Type:     "test2",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	clients := map[string]ai.ProviderClient{
		"test1": &mockProviderClient{name: "Test Provider 1", enabled: true, response: "Result 1"},
		"test2": &mockProviderClient{name: "Test Provider 2", enabled: true, response: "Result 2"},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)

	if mpc == nil {
		t.Fatal("NewMultiProviderClient returned nil")
	}

	if mpc.GetActiveProvidersCount() != 2 {
		t.Errorf("Expected 2 active providers, got %d", mpc.GetActiveProvidersCount())
	}

	if mpc.GetTotalChannels() != 3 {
		t.Errorf("Expected 3 total channels (2+1), got %d", mpc.GetTotalChannels())
	}
}

// TestMultiProviderClient_NormalizeName_MajorityVote проверяет агрегацию результатов методом majority vote
func TestMultiProviderClient_NormalizeName_MajorityVote(t *testing.T) {
	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Provider 1",
			Type:     "provider1",
			IsActive: true,
			Config:   `{"channels":2}`, // 2 канала, оба вернут "ООО Тест"
		},
		{
			ID:       2,
			Name:     "Provider 2",
			Type:     "provider2",
			IsActive: true,
			Config:   `{"channels":1}`, // 1 канал, вернет "ООО Тест"
		},
		{
			ID:       3,
			Name:     "Provider 3",
			Type:     "provider3",
			IsActive: true,
			Config:   `{"channels":1}`, // 1 канал, вернет "ООО Тест Другой" (меньшинство)
		},
	}

	clients := map[string]ai.ProviderClient{
		"provider1": &mockProviderClient{name: "Provider 1", enabled: true, response: "ООО Тест"},
		"provider2": &mockProviderClient{name: "Provider 2", enabled: true, response: "ООО Тест"},
		"provider3": &mockProviderClient{name: "Provider 3", enabled: true, response: "ООО Тест Другой"},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	result, err := mpc.NormalizeName(ctx, "ооо тест")

	if err != nil {
		t.Fatalf("NormalizeName failed: %v", err)
	}

	// Majority vote должен выбрать "ООО Тест" (3 голоса из 4)
	if result != "ООО Тест" {
		t.Errorf("Expected 'ООО Тест', got '%s'", result)
	}
}

// TestMultiProviderClient_NormalizeName_WithErrors проверяет обработку ошибок провайдеров
func TestMultiProviderClient_NormalizeName_WithErrors(t *testing.T) {
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

	clients := map[string]ai.ProviderClient{
		"provider1": &mockProviderClient{name: "Provider 1", enabled: true, shouldError: true},
		"provider2": &mockProviderClient{name: "Provider 2", enabled: true, response: "ООО Тест"},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	result, err := mpc.NormalizeName(ctx, "ооо тест")

	if err != nil {
		t.Fatalf("NormalizeName failed even with one successful provider: %v", err)
	}

	// Должен вернуть результат от успешного провайдера
	if result != "ООО Тест" {
		t.Errorf("Expected 'ООО Тест', got '%s'", result)
	}
}

// TestMultiProviderClient_NormalizeName_AllErrors проверяет обработку ситуации, когда все провайдеры вернули ошибки
func TestMultiProviderClient_NormalizeName_AllErrors(t *testing.T) {
	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Provider 1",
			Type:     "provider1",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	clients := map[string]ai.ProviderClient{
		"provider1": &mockProviderClient{name: "Provider 1", enabled: true, shouldError: true},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	_, err := mpc.NormalizeName(ctx, "ооо тест")

	if err == nil {
		t.Error("Expected error when all providers fail, got nil")
	}
}

// TestMultiProviderClient_NormalizeName_Timeout проверяет обработку таймаутов
func TestMultiProviderClient_NormalizeName_Timeout(t *testing.T) {
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

	clients := map[string]ai.ProviderClient{
		"provider1": &mockProviderClient{name: "Provider 1", enabled: true, responseDelay: 2 * time.Second, response: "ООО Тест"},
		"provider2": &mockProviderClient{name: "Provider 2", enabled: true, response: "ООО Тест"},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)
	mpc.timeout = 500 * time.Millisecond // Короткий таймаут

	ctx := context.Background()
	result, err := mpc.NormalizeName(ctx, "ооо тест")

	// Должен вернуть результат от быстрого провайдера, даже если другой таймаутит
	if err != nil {
		t.Fatalf("NormalizeName failed: %v", err)
	}

	if result != "ООО Тест" {
		t.Errorf("Expected 'ООО Тест', got '%s'", result)
	}
}

// TestMultiProviderClient_EmptyName проверяет обработку пустого имени
func TestMultiProviderClient_EmptyName(t *testing.T) {
	providers := []*database.Provider{}
	clients := map[string]ai.ProviderClient{}

	mpc := NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	_, err := mpc.NormalizeName(ctx, "")

	if err == nil {
		t.Error("Expected error for empty name, got nil")
	}
}

// TestMultiProviderClient_NoActiveProviders проверяет обработку отсутствия активных провайдеров
func TestMultiProviderClient_NoActiveProviders(t *testing.T) {
	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Provider 1",
			Type:     "provider1",
			IsActive: false, // Отключен
			Config:   `{"channels":1}`,
		},
	}

	clients := map[string]ai.ProviderClient{
		"provider1": &mockProviderClient{name: "Provider 1", enabled: false},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	_, err := mpc.NormalizeName(ctx, "ооо тест")

	if err == nil {
		t.Error("Expected error when no active providers, got nil")
	}
}

// TestMultiProviderClient_NormalizeName_MajorityVote_Wins проверяет, что majority vote правильно выбирает результат
func TestMultiProviderClient_NormalizeName_MajorityVote_Wins(t *testing.T) {
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

	clients := map[string]ai.ProviderClient{
		"provider1": &mockProviderClient{name: "Provider 1", enabled: true, response: "Result A"},
		"provider2": &mockProviderClient{name: "Provider 2", enabled: true, response: "Result A"},
		"provider3": &mockProviderClient{name: "Provider 3", enabled: true, response: "Result B"},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	result, err := mpc.NormalizeName(ctx, "test")

	if err != nil {
		t.Fatalf("NormalizeName failed: %v", err)
	}

	// Majority vote должен выбрать "Result A" (2 голоса из 3)
	if result != "Result A" {
		t.Errorf("Expected 'Result A' (majority), got '%s'", result)
	}
}

// TestMultiProviderClient_NormalizeName_ConcurrentExecution_AllCalled проверяет, что все провайдеры вызываются параллельно
func TestMultiProviderClient_NormalizeName_ConcurrentExecution_AllCalled(t *testing.T) {
	// Используем каналы для отслеживания вызовов
	callChannels := make(map[string]chan struct{})
	
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

	// Создаем моки с каналами для отслеживания вызовов
	callChannels["provider1"] = make(chan struct{}, 1)
	callChannels["provider2"] = make(chan struct{}, 1)
	callChannels["provider3"] = make(chan struct{}, 1)

	clients := map[string]ai.ProviderClient{
		"provider1": &concurrentMockProvider{
			mockProviderClient: mockProviderClient{name: "Provider 1", enabled: true, response: "Result A"},
			callChan: callChannels["provider1"],
		},
		"provider2": &concurrentMockProvider{
			mockProviderClient: mockProviderClient{name: "Provider 2", enabled: true, response: "Result A"},
			callChan: callChannels["provider2"],
		},
		"provider3": &concurrentMockProvider{
			mockProviderClient: mockProviderClient{name: "Provider 3", enabled: true, response: "Result A"},
			callChan: callChannels["provider3"],
		},
	}

	mpc := NewMultiProviderClient(providers, clients, nil)

	ctx := context.Background()
	
	// Запускаем нормализацию
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)
	
	go func() {
		result, err := mpc.NormalizeName(ctx, "test")
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// Ждем, чтобы все провайдеры были вызваны (или таймаут)
	timeout := time.After(2 * time.Second)
	callCount := 0
	
	for callCount < 3 {
		select {
		case <-callChannels["provider1"]:
			callCount++
		case <-callChannels["provider2"]:
			callCount++
		case <-callChannels["provider3"]:
			callCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for all providers to be called. Only %d/%d called", callCount, 3)
		}
	}

	// Проверяем результат
	select {
	case result := <-resultChan:
		if result == "" {
			t.Error("Expected non-empty result")
		}
	case err := <-errChan:
		t.Fatalf("NormalizeName failed: %v", err)
	case <-timeout:
		t.Error("Timeout waiting for result")
	}
}

// concurrentMockProvider мок провайдера с поддержкой отслеживания вызовов
type concurrentMockProvider struct {
	mockProviderClient
	callChan chan struct{}
}

func (m *concurrentMockProvider) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	// Отправляем сигнал о вызове
	select {
	case m.callChan <- struct{}{}:
	default:
	}
	return m.mockProviderClient.GetCompletion(systemPrompt, userPrompt)
}

