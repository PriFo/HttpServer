package container

import (
	"fmt"

	clienthandler "httpserver/internal/api/handlers/client"
	clientapp "httpserver/internal/application/client"
	clientdomain "httpserver/internal/domain/client"
	"httpserver/internal/infrastructure/persistence"
)

// initClientComponents инициализирует компоненты client domain
func (c *Container) initClientComponents() error {
	// 1. Создаем репозитории (infrastructure layer)
	clientRepo := persistence.NewClientRepository(c.ServiceDB)
	databaseRepo := persistence.NewDatabaseRepository(c.ServiceDB)

	// 2. Создаем domain service
	clientDomainService := clientdomain.NewService(
		clientRepo,
		databaseRepo,
	)

	// 3. Создаем application use case
	clientUseCase := clientapp.NewUseCase(
		clientRepo,
		databaseRepo,
		clientDomainService,
	)

	// 4. Создаем base handler через wrapper, чтобы избежать циклического импорта
	baseHandler := newBaseHandlerWrapper()

	// 5. Создаем HTTP handler
	clientHandler := clienthandler.NewHandler(baseHandler, clientUseCase)

	// Сохраняем в контейнере
	c.ClientHandlerV2 = clientHandler
	c.ClientUseCase = clientUseCase
	c.ClientDomainService = clientDomainService

	return nil
}

// GetClientHandler возвращает client handler из контейнера
func (c *Container) GetClientHandler() (*clienthandler.Handler, error) {
	if c.ClientHandlerV2 == nil {
		return nil, fmt.Errorf("client handler not initialized")
	}

	handler, ok := c.ClientHandlerV2.(*clienthandler.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid client handler type")
	}

	return handler, nil
}
