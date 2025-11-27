package container

import (
	"fmt"

	projecthandler "httpserver/internal/api/handlers/project"
	projectapp "httpserver/internal/application/project"
	projectdomain "httpserver/internal/domain/project"
	"httpserver/internal/infrastructure/persistence"
)

// initProjectComponents инициализирует компоненты project domain
func (c *Container) initProjectComponents() error {
	// 1. Создаем репозитории (infrastructure layer)
	projectRepo := persistence.NewProjectRepository(c.ServiceDB)
	clientRepo := persistence.NewClientRepository(c.ServiceDB)
	databaseRepo := persistence.NewDatabaseRepository(c.ServiceDB)

	// 2. Создаем domain service
	projectDomainService := projectdomain.NewService(
		projectRepo,
		clientRepo,
		databaseRepo,
	)

	// 3. Создаем application use case
	projectUseCase := projectapp.NewUseCase(
		projectRepo,
		clientRepo,
		databaseRepo,
		projectDomainService,
	)

	// 4. Создаем base handler через wrapper, чтобы избежать циклического импорта
	baseHandler := newBaseHandlerWrapper()

	// 5. Создаем HTTP handler
	projectHandler := projecthandler.NewHandler(baseHandler, projectUseCase)

	// Сохраняем в контейнере
	c.ProjectHandlerV2 = projectHandler
	c.ProjectUseCase = projectUseCase
	c.ProjectDomainService = projectDomainService

	return nil
}

// GetProjectHandler возвращает project handler из контейнера
func (c *Container) GetProjectHandler() (*projecthandler.Handler, error) {
	if c.ProjectHandlerV2 == nil {
		return nil, fmt.Errorf("project handler not initialized")
	}

	handler, ok := c.ProjectHandlerV2.(*projecthandler.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid project handler type")
	}

	return handler, nil
}
