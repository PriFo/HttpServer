package server

import (
	"log"

	appcontainer "httpserver/internal/container"
)

// initNewUploadArchitecture инициализирует новую архитектуру доменов (Clean Architecture)
// Использует DI Container для создания компонентов Upload, Normalization и Quality доменов
func (s *Server) initNewUploadArchitecture() {
	// Проверяем, что базы данных инициализированы
	if s.db == nil || s.serviceDB == nil {
		log.Printf("Warning: Cannot initialize new architecture: databases not initialized")
		return
	}

	// Создаем контейнер с текущей конфигурацией
	container, err := appcontainer.NewContainer(s.config)
	if err != nil {
		log.Printf("Warning: Failed to create container for new architecture: %v", err)
		return
	}

	// Инициализируем контейнер (включает все домены: Upload, Normalization, Quality)
	if err := container.Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize container for new architecture: %v", err)
		return
	}

	// Проверяем, что все handlers инициализированы
	uploadHandler, err := container.GetUploadHandler()
	if err != nil {
		log.Printf("Warning: Failed to get upload handler: %v", err)
	} else {
		log.Printf("Upload Domain handler initialized successfully")
		s.uploadHandlerV2 = uploadHandler
	}

	normalizationHandler, err := container.GetNormalizationHandler()
	if err != nil {
		log.Printf("Warning: Failed to get normalization handler: %v", err)
	} else {
		log.Printf("Normalization Domain handler initialized successfully")
	}

	qualityHandler, err := container.GetQualityHandler()
	if err != nil {
		log.Printf("Warning: Failed to get quality handler: %v", err)
	} else {
		log.Printf("Quality Domain handler initialized successfully")
	}

	// Сохраняем контейнер новой архитектуры отдельно для использования в setupRouter()
	s.cleanContainer = container

	// Сохраняем handlers (опционально, для прямого доступа)
	_ = normalizationHandler
	_ = qualityHandler

	log.Printf("New Clean Architecture initialized successfully (Upload, Normalization, Quality domains)")
}
