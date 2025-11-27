package container

import (
	"fmt"

	qualityapp "httpserver/internal/application/quality"
	qualitydomain "httpserver/internal/domain/quality"
	qualityhandler "httpserver/internal/api/handlers/quality"
	infraservices "httpserver/internal/infrastructure/services"
	"httpserver/internal/infrastructure/persistence"
	"httpserver/server/middleware"
)

// initQualityComponents инициализирует компоненты quality domain
func (c *Container) initQualityComponents() error {
	// 1. Создаем репозиторий (infrastructure layer)
	qualityRepo := persistence.NewQualityRepository(c.DB)

	// 2. Создаем адаптер для QualityAnalyzer
	var qualityAnalyzerAdapter qualitydomain.QualityAnalyzerInterface
	if c.QualityService != nil {
		// QualityService уже реализует QualityServiceInterface
		qualityAnalyzerAdapter = infraservices.NewQualityAnalyzerAdapter(c.QualityService)
	}

	// 3. Создаем domain service
	qualityDomainService := qualitydomain.NewService(
		qualityRepo,
		qualityAnalyzerAdapter,
	)

	// 4. Создаем application use case
	qualityUseCase := qualityapp.NewUseCase(qualityRepo, qualityDomainService)

	// 5. Создаем base handler через wrapper, чтобы избежать циклического импорта
	baseHandler := &baseHandlerWrapper{
		writeJSONResponse: middleware.WriteJSONResponse,
		writeJSONError:    middleware.WriteJSONError,
		handleHTTPError:   middleware.HandleHTTPError,
	}

	// 6. Создаем HTTP handler
	qualityHandler := qualityhandler.NewHandler(baseHandler, qualityUseCase)

	// Сохраняем в контейнере
	c.QualityHandlerV2 = qualityHandler
	c.QualityUseCase = qualityUseCase
	c.QualityDomainService = qualityDomainService

	return nil
}

// GetQualityHandler возвращает quality handler из контейнера
func (c *Container) GetQualityHandler() (*qualityhandler.Handler, error) {
	if c.QualityHandlerV2 == nil {
		return nil, fmt.Errorf("quality handler not initialized")
	}

	handler, ok := c.QualityHandlerV2.(*qualityhandler.Handler)
	if !ok {
		return nil, fmt.Errorf("invalid quality handler type")
	}

	return handler, nil
}

