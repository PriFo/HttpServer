package services

import (
	apperrors "httpserver/server/errors"
)

// WorkerService сервис для работы с конфигурацией воркеров и провайдеров
type WorkerService struct {
	workerConfigManager interface {
		GetConfig() map[string]interface{}
		GetActiveProvider() (interface{}, error)
		GetActiveModel(providerName string) (interface{}, error)
		UpdateProvider(providerName string, config interface{}) error
		UpdateModel(providerName string, modelName string, config interface{}) error
		SetDefaultProvider(providerName string) error
		SetDefaultModel(providerName string, modelName string) error
		SetGlobalMaxWorkers(maxWorkers int) error
	}
}

// NewWorkerService создает новый сервис воркеров
func NewWorkerService(workerConfigManager interface {
	GetConfig() map[string]interface{}
	GetActiveProvider() (interface{}, error)
	GetActiveModel(providerName string) (interface{}, error)
	UpdateProvider(providerName string, config interface{}) error
	UpdateModel(providerName string, modelName string, config interface{}) error
	SetDefaultProvider(providerName string) error
	SetDefaultModel(providerName string, modelName string) error
	SetGlobalMaxWorkers(maxWorkers int) error
}) *WorkerService {
	return &WorkerService{
		workerConfigManager: workerConfigManager,
	}
}

// GetConfig получает конфигурацию воркеров
func (ws *WorkerService) GetConfig() (map[string]interface{}, error) {
	if ws.workerConfigManager == nil {
		return nil, apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.GetConfig(), nil
}

// GetActiveProvider получает активного провайдера
func (ws *WorkerService) GetActiveProvider() (interface{}, error) {
	if ws.workerConfigManager == nil {
		return nil, apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.GetActiveProvider()
}

// GetActiveModel получает активную модель провайдера
func (ws *WorkerService) GetActiveModel(providerName string) (interface{}, error) {
	if ws.workerConfigManager == nil {
		return nil, apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.GetActiveModel(providerName)
}

// UpdateProvider обновляет конфигурацию провайдера
func (ws *WorkerService) UpdateProvider(providerName string, config interface{}) error {
	if ws.workerConfigManager == nil {
		return apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.UpdateProvider(providerName, config)
}

// UpdateModel обновляет конфигурацию модели
func (ws *WorkerService) UpdateModel(providerName string, modelName string, config interface{}) error {
	if ws.workerConfigManager == nil {
		return apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.UpdateModel(providerName, modelName, config)
}

// SetDefaultProvider устанавливает провайдера по умолчанию
func (ws *WorkerService) SetDefaultProvider(providerName string) error {
	if ws.workerConfigManager == nil {
		return apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.SetDefaultProvider(providerName)
}

// SetDefaultModel устанавливает модель по умолчанию
func (ws *WorkerService) SetDefaultModel(providerName string, modelName string) error {
	if ws.workerConfigManager == nil {
		return apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.SetDefaultModel(providerName, modelName)
}

// SetGlobalMaxWorkers устанавливает максимальное количество воркеров
func (ws *WorkerService) SetGlobalMaxWorkers(maxWorkers int) error {
	if ws.workerConfigManager == nil {
		return apperrors.NewInternalError("менеджер конфигурации воркеров не инициализирован", nil)
	}
	return ws.workerConfigManager.SetGlobalMaxWorkers(maxWorkers)
}

