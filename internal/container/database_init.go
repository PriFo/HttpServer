package container

import (
	"fmt"

	databasehandler "httpserver/internal/api/handlers/database"
	databaseapp "httpserver/internal/application/database"
	databasedomain "httpserver/internal/domain/database"
	"httpserver/internal/infrastructure/persistence"
)

// initDatabaseComponents инициализирует компоненты database domain
func (c *Container) initDatabaseComponents() error {
	// 1. Создаем репозитории (infrastructure layer)
	databaseRepo := persistence.NewDatabaseRepository(c.ServiceDB)
	projectRepo := persistence.NewProjectRepository(c.ServiceDB)

	// 2. Создаем domain service
	databaseDomainService := databasedomain.NewService(
		databaseRepo,
		projectRepo,
	)

	// 3. Создаем application use case
	databaseUseCase := databaseapp.NewUseCase(
		databaseRepo,
		projectRepo,
		databaseDomainService,
	)

	// 4. Создаем base handler через wrapper, чтобы избежать циклического импорта
	baseHandler := newBaseHandlerWrapper()

	// 5. Создаем HTTP handler
	databaseHandler := databasehandler.NewHandler(baseHandler, databaseUseCase)

	// Сохраняем в контейнере
	c.DatabaseHandlerV2 = databaseHandler
	c.DatabaseUseCase = databaseUseCase
	c.DatabaseDomainService = databaseDomainService

	return nil
}

// GetDatabaseHandler возвращает database handler из контейнера
func (c *Container) GetDatabaseHandler() (*databasehandler.Handler, error) {
	if c.DatabaseHandlerV2 == nil {
		return nil, fmt.Errorf("database handler not initialized")
	}

	handler, ok := c.DatabaseHandlerV2.(*databasehandler.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid database handler type")
	}

	return handler, nil
}

