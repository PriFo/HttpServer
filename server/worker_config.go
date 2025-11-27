package server

import (
	"httpserver/database"
	"httpserver/internal/infrastructure/workers"
)

// WorkerConfigManager обертка для workers.WorkerConfigManager
// Используется для обратной совместимости с cmd файлами
type WorkerConfigManager = workers.WorkerConfigManager
type ProviderConfig = workers.ProviderConfig
type ModelConfig = workers.ModelConfig

// NewWorkerConfigManager создает новый менеджер конфигурации воркеров
// Это обертка над workers.NewWorkerConfigManager для обратной совместимости
func NewWorkerConfigManager(serviceDB *database.ServiceDB) *WorkerConfigManager {
	return workers.NewWorkerConfigManager(serviceDB)
}
