package services

import (
	"errors"
	"testing"
)

// mockWorkerConfigManager мок для workerConfigManager
type mockWorkerConfigManager struct {
	config          map[string]interface{}
	activeProvider  interface{}
	activeModel     interface{}
	updateProviderErr error
	updateModelErr    error
}

func (m *mockWorkerConfigManager) GetConfig() map[string]interface{} {
	return m.config
}

func (m *mockWorkerConfigManager) GetActiveProvider() (interface{}, error) {
	if m.activeProvider == nil {
		return nil, errors.New("no active provider")
	}
	return m.activeProvider, nil
}

func (m *mockWorkerConfigManager) GetActiveModel(providerName string) (interface{}, error) {
	if m.activeModel == nil {
		return nil, errors.New("no active model")
	}
	return m.activeModel, nil
}

func (m *mockWorkerConfigManager) UpdateProvider(providerName string, config interface{}) error {
	return m.updateProviderErr
}

func (m *mockWorkerConfigManager) UpdateModel(providerName string, modelName string, config interface{}) error {
	return m.updateModelErr
}

func (m *mockWorkerConfigManager) SetDefaultProvider(providerName string) error {
	return nil
}

func (m *mockWorkerConfigManager) SetDefaultModel(providerName string, modelName string) error {
	return nil
}

func (m *mockWorkerConfigManager) SetGlobalMaxWorkers(maxWorkers int) error {
	return nil
}

// TestNewWorkerService проверяет создание нового сервиса воркеров
func TestNewWorkerService(t *testing.T) {
	mockManager := &mockWorkerConfigManager{
		config: map[string]interface{}{"test": "config"},
	}

	service := NewWorkerService(mockManager)
	if service == nil {
		t.Error("NewWorkerService() should not return nil")
	}
}

// TestWorkerService_GetConfig проверяет получение конфигурации
func TestWorkerService_GetConfig(t *testing.T) {
	mockManager := &mockWorkerConfigManager{
		config: map[string]interface{}{
			"max_workers": 10,
			"providers":   []string{"provider1"},
		},
	}

	service := NewWorkerService(mockManager)

	config, err := service.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if config == nil {
		t.Error("Expected non-nil config")
	}
}

// TestWorkerService_GetConfig_NilManager проверяет обработку nil менеджера
func TestWorkerService_GetConfig_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	_, err := service.GetConfig()
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}

// TestWorkerService_GetActiveProvider проверяет получение активного провайдера
func TestWorkerService_GetActiveProvider(t *testing.T) {
	mockManager := &mockWorkerConfigManager{
		activeProvider: map[string]interface{}{
			"name": "test-provider",
		},
	}

	service := NewWorkerService(mockManager)

	provider, err := service.GetActiveProvider()
	if err != nil {
		t.Fatalf("GetActiveProvider() error = %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}
}

// TestWorkerService_GetActiveProvider_NilManager проверяет обработку nil менеджера
func TestWorkerService_GetActiveProvider_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	_, err := service.GetActiveProvider()
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}

// TestWorkerService_GetActiveModel проверяет получение активной модели
func TestWorkerService_GetActiveModel(t *testing.T) {
	mockManager := &mockWorkerConfigManager{
		activeModel: map[string]interface{}{
			"name": "test-model",
		},
	}

	service := NewWorkerService(mockManager)

	model, err := service.GetActiveModel("test-provider")
	if err != nil {
		t.Fatalf("GetActiveModel() error = %v", err)
	}

	if model == nil {
		t.Error("Expected non-nil model")
	}
}

// TestWorkerService_GetActiveModel_NilManager проверяет обработку nil менеджера
func TestWorkerService_GetActiveModel_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	_, err := service.GetActiveModel("test-provider")
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}

// TestWorkerService_UpdateProvider проверяет обновление провайдера
func TestWorkerService_UpdateProvider(t *testing.T) {
	mockManager := &mockWorkerConfigManager{}

	service := NewWorkerService(mockManager)

	err := service.UpdateProvider("test-provider", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("UpdateProvider() error = %v", err)
	}
}

// TestWorkerService_UpdateProvider_NilManager проверяет обработку nil менеджера
func TestWorkerService_UpdateProvider_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	err := service.UpdateProvider("test-provider", map[string]interface{}{"key": "value"})
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}

// TestWorkerService_UpdateModel проверяет обновление модели
func TestWorkerService_UpdateModel(t *testing.T) {
	mockManager := &mockWorkerConfigManager{}

	service := NewWorkerService(mockManager)

	err := service.UpdateModel("test-provider", "test-model", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("UpdateModel() error = %v", err)
	}
}

// TestWorkerService_UpdateModel_NilManager проверяет обработку nil менеджера
func TestWorkerService_UpdateModel_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	err := service.UpdateModel("test-provider", "test-model", map[string]interface{}{"key": "value"})
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}

// TestWorkerService_SetDefaultProvider проверяет установку провайдера по умолчанию
func TestWorkerService_SetDefaultProvider(t *testing.T) {
	mockManager := &mockWorkerConfigManager{}

	service := NewWorkerService(mockManager)

	err := service.SetDefaultProvider("test-provider")
	if err != nil {
		t.Fatalf("SetDefaultProvider() error = %v", err)
	}
}

// TestWorkerService_SetDefaultProvider_NilManager проверяет обработку nil менеджера
func TestWorkerService_SetDefaultProvider_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	err := service.SetDefaultProvider("test-provider")
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}

// TestWorkerService_SetDefaultModel проверяет установку модели по умолчанию
func TestWorkerService_SetDefaultModel(t *testing.T) {
	mockManager := &mockWorkerConfigManager{}

	service := NewWorkerService(mockManager)

	err := service.SetDefaultModel("test-provider", "test-model")
	if err != nil {
		t.Fatalf("SetDefaultModel() error = %v", err)
	}
}

// TestWorkerService_SetDefaultModel_NilManager проверяет обработку nil менеджера
func TestWorkerService_SetDefaultModel_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	err := service.SetDefaultModel("test-provider", "test-model")
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}

// TestWorkerService_SetGlobalMaxWorkers проверяет установку максимального количества воркеров
func TestWorkerService_SetGlobalMaxWorkers(t *testing.T) {
	mockManager := &mockWorkerConfigManager{}

	service := NewWorkerService(mockManager)

	err := service.SetGlobalMaxWorkers(10)
	if err != nil {
		t.Fatalf("SetGlobalMaxWorkers() error = %v", err)
	}
}

// TestWorkerService_SetGlobalMaxWorkers_NilManager проверяет обработку nil менеджера
func TestWorkerService_SetGlobalMaxWorkers_NilManager(t *testing.T) {
	service := NewWorkerService(nil)

	err := service.SetGlobalMaxWorkers(10)
	if err == nil {
		t.Error("Expected error when workerConfigManager is nil")
	}
}


