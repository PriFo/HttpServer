package container

import (
	"fmt"

	"httpserver/internal/api/handlers/upload"
	uploadapp "httpserver/internal/application/upload"
	uploaddomain "httpserver/internal/domain/upload"
	infraservices "httpserver/internal/infrastructure/services"
	"httpserver/internal/infrastructure/persistence"
)

// initUploadComponents инициализирует компоненты upload domain
func (c *Container) initUploadComponents() error {
	// 1. Создаем репозиторий (infrastructure layer)
	uploadRepo := persistence.NewUploadRepository(c.DB)

	// 2. Создаем адаптер для DatabaseInfoService
	databaseInfoService := infraservices.NewDatabaseInfoAdapter(
		c.DB,
		c.ServiceDB,
		c.DBInfoCache,
	)

	// 3. Создаем domain service
	uploadDomainService := uploaddomain.NewService(uploadRepo, databaseInfoService)

	// 4. Создаем application use case
	uploadUseCase := uploadapp.NewUseCase(uploadRepo, uploadDomainService)

	// 5. Создаем base handler через wrapper, чтобы избежать циклического импорта
	baseHandler := newBaseHandlerWrapper()

	// 6. Создаем HTTP handler
	uploadHandler := upload.NewHandler(baseHandler, uploadUseCase)

	// Сохраняем в контейнере
	c.UploadHandlerV2 = uploadHandler
	c.UploadUseCase = uploadUseCase
	c.UploadDomainService = uploadDomainService

	return nil
}

// GetUploadHandler возвращает upload handler из контейнера
// Создает новый экземпляр с текущими зависимостями контейнера
func (c *Container) GetUploadHandler() (*upload.Handler, error) {
	if c.DB == nil || c.ServiceDB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	uploadRepo := persistence.NewUploadRepository(c.DB)
	databaseInfoService := infraservices.NewDatabaseInfoAdapter(
		c.DB,
		c.ServiceDB,
		c.DBInfoCache,
	)
	uploadDomainService := uploaddomain.NewService(uploadRepo, databaseInfoService)
	uploadUseCase := uploadapp.NewUseCase(uploadRepo, uploadDomainService)
	baseHandler := newBaseHandlerWrapper()

	return upload.NewHandler(baseHandler, uploadUseCase), nil
}

