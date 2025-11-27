package container

import (
	"fmt"

	normalizationapp "httpserver/internal/application/normalization"
	normalizationdomain "httpserver/internal/domain/normalization"
	normalizationhandler "httpserver/internal/api/handlers/normalization"
	infraservices "httpserver/internal/infrastructure/services"
	"httpserver/internal/infrastructure/persistence"
)

// initNormalizationComponents инициализирует компоненты normalization domain
func (c *Container) initNormalizationComponents() error {
	// 1. Создаем репозиторий (infrastructure layer)
	normalizationRepo := persistence.NewNormalizationRepository(c.DB, c.ServiceDB)

	// 2. Создаем адаптер для Normalizer
	var normalizerAdapter normalizationdomain.NormalizerInterface
	if c.Normalizer != nil {
		normalizerAdapter = infraservices.NewNormalizerAdapter(c.Normalizer)
	}

	// 3. Создаем адаптер для BenchmarkService
	var benchmarkAdapter normalizationdomain.BenchmarkServiceInterface
	if c.BenchmarkService != nil {
		benchmarkAdapter = infraservices.NewBenchmarkAdapter(c.BenchmarkService)
	}

	// 4. Создаем domain service
	normalizationDomainService := normalizationdomain.NewService(
		normalizationRepo,
		normalizerAdapter,
		benchmarkAdapter,
	)

	// 5. Создаем application use case
	normalizationUseCase := normalizationapp.NewUseCase(normalizationRepo, normalizationDomainService)

	// 6. Создаем base handler через wrapper, чтобы избежать циклического импорта
	baseHandler := newBaseHandlerWrapper()

	// 7. Создаем HTTP handler
	normalizationHandler := normalizationhandler.NewHandler(baseHandler, normalizationUseCase)

	// Сохраняем в контейнере
	c.NormalizationHandlerV2 = normalizationHandler
	c.NormalizationUseCase = normalizationUseCase
	c.NormalizationDomainService = normalizationDomainService

	return nil
}

// GetNormalizationHandler возвращает normalization handler из контейнера
func (c *Container) GetNormalizationHandler() (*normalizationhandler.Handler, error) {
	if c.NormalizationHandlerV2 == nil {
		return nil, fmt.Errorf("normalization handler not initialized")
	}

	handler, ok := c.NormalizationHandlerV2.(*normalizationhandler.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid normalization handler type")
	}

	return handler, nil
}

