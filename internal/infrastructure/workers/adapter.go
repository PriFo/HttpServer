package workers

import "fmt"

// Adapter адаптер для WorkerConfigManager, чтобы соответствовать интерфейсу WorkerService
// Используется для интеграции WorkerConfigManager с сервисами, требующими интерфейс WorkerService
type Adapter struct {
	Wcm *WorkerConfigManager
}

// GetConfig получает конфигурацию
func (a *Adapter) GetConfig() map[string]interface{} {
	if a.Wcm == nil {
		return nil
	}
	return a.Wcm.GetConfig()
}

// GetActiveProvider получает активного провайдера
func (a *Adapter) GetActiveProvider() (interface{}, error) {
	if a.Wcm == nil {
		return nil, fmt.Errorf("worker config manager not initialized")
	}
	return a.Wcm.GetActiveProvider()
}

// GetActiveModel получает активную модель
func (a *Adapter) GetActiveModel(providerName string) (interface{}, error) {
	if a.Wcm == nil {
		return nil, fmt.Errorf("worker config manager not initialized")
	}
	return a.Wcm.GetActiveModel(providerName)
}

// UpdateProvider обновляет провайдера
func (a *Adapter) UpdateProvider(providerName string, config interface{}) error {
	if a.Wcm == nil {
		return fmt.Errorf("worker config manager not initialized")
	}
	if providerConfig, ok := config.(*ProviderConfig); ok {
		return a.Wcm.UpdateProvider(providerName, providerConfig)
	}
	return fmt.Errorf("invalid provider config type")
}

// UpdateModel обновляет модель
func (a *Adapter) UpdateModel(providerName string, modelName string, config interface{}) error {
	if a.Wcm == nil {
		return fmt.Errorf("worker config manager not initialized")
	}
	if modelConfig, ok := config.(*ModelConfig); ok {
		return a.Wcm.UpdateModel(providerName, modelName, modelConfig)
	}
	return fmt.Errorf("invalid model config type")
}

// SetDefaultProvider устанавливает провайдера по умолчанию
func (a *Adapter) SetDefaultProvider(providerName string) error {
	if a.Wcm == nil {
		return fmt.Errorf("worker config manager not initialized")
	}
	return a.Wcm.SetDefaultProvider(providerName)
}

// SetDefaultModel устанавливает модель по умолчанию
func (a *Adapter) SetDefaultModel(providerName string, modelName string) error {
	if a.Wcm == nil {
		return fmt.Errorf("worker config manager not initialized")
	}
	return a.Wcm.SetDefaultModel(providerName, modelName)
}

// SetGlobalMaxWorkers устанавливает максимальное количество воркеров
func (a *Adapter) SetGlobalMaxWorkers(maxWorkers int) error {
	if a.Wcm == nil {
		return fmt.Errorf("worker config manager not initialized")
	}
	return a.Wcm.SetGlobalMaxWorkers(maxWorkers)
}
