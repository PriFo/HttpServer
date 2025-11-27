package container

import (
	"fmt"

	classificationapp "httpserver/internal/application/classification"
	classificationdomain "httpserver/internal/domain/classification"
	classificationhandler "httpserver/internal/api/handlers/classification"
	"httpserver/internal/infrastructure/persistence"
)

// initClassificationComponents инициализирует компоненты classification domain
func (c *Container) initClassificationComponents() error {
	// 1. Создаем репозиторий (infrastructure layer)
	classificationRepo := persistence.NewClassificationRepository(c.DB, c.ServiceDB)

	// 2. Создаем domain service
	classificationDomainService := classificationdomain.NewService(classificationRepo)

	// 3. Создаем application use case
	classificationUseCase := classificationapp.NewUseCase(classificationRepo, classificationDomainService)

	// 4. Создаем base handler через wrapper, чтобы избежать циклического импорта
	baseHandler := newBaseHandlerWrapper()

	// 5. Создаем HTTP handler
	// Используем type assertion для преобразования нашего wrapper в нужный тип
	// Но сначала нужно проверить, какой тип ожидает NewHandler
	classificationHandler := classificationhandler.NewHandler(baseHandler, classificationUseCase)

	// Сохраняем в контейнере
	c.ClassificationHandlerV2 = classificationHandler
	c.ClassificationUseCase = classificationUseCase
	c.ClassificationDomainService = classificationDomainService

	return nil
}

// GetClassificationHandler возвращает classification handler из контейнера
func (c *Container) GetClassificationHandler() (*classificationhandler.Handler, error) {
	if c.ClassificationHandlerV2 == nil {
		return nil, fmt.Errorf("classification handler not initialized")
	}

	handler, ok := c.ClassificationHandlerV2.(*classificationhandler.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid classification handler type")
	}

	return handler, nil
}

