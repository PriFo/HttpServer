package routes

import (
	"net/http"

	"httpserver/internal/api/handlers/classification"
	"httpserver/internal/api/handlers/client"
	"httpserver/internal/api/handlers/database"
	"httpserver/internal/api/handlers/normalization"
	"httpserver/internal/api/handlers/project"
	"httpserver/internal/api/handlers/quality"
	"httpserver/internal/api/handlers/upload"
	"httpserver/internal/container"
	"httpserver/server/handlers"
)

// Router управляет маршрутизацией приложения
// Централизует регистрацию всех маршрутов
type Router struct {
	mux                        *http.ServeMux
	uploadHandler              *upload.Handler
	normalizationHandler       *normalization.Handler
	legacyNormalizationHandler *handlers.NormalizationHandler
	qualityHandler             *quality.Handler
	classificationHandler      *classification.Handler
	clientHandler              *client.Handler
	projectHandler             *project.Handler
	databaseHandler            *database.Handler
	workerHandler              *handlers.WorkerHandler
	webSearchValidationHandler *handlers.WebSearchValidationHandler
	webSearchAdminHandler      *handlers.WebSearchAdminHandler
	baseHandler                *handlers.BaseHandler
}

// RegisterOptions задают опции регистрации маршрутов
type RegisterOptions struct {
	SkipUploadRoutes         bool
	SkipNormalizationRoutes  bool
	SkipQualityRoutes        bool
	SkipClassificationRoutes bool
	SkipClientRoutes         bool
	SkipProjectRoutes        bool
	SkipDatabaseRoutes       bool
	SkipWebSearchRoutes      bool
}

// NewRouter создает новый роутер
func NewRouter(mux *http.ServeMux, container *container.Container) (*Router, error) {
	router := &Router{
		mux:         mux,
		baseHandler: handlers.NewBaseHandlerFromMiddleware(),
	}

	// Получаем upload handler из контейнера (новая архитектура)
	uploadHandler, err := container.GetUploadHandler()
	if err == nil {
		router.uploadHandler = uploadHandler
	}

	// Получаем normalization handler из контейнера (новая архитектура)
	normalizationHandler, err := container.GetNormalizationHandler()
	if err == nil {
		router.normalizationHandler = normalizationHandler
	}
	// Сохраняем legacy normalization handler для маршрутов, которые пока не перенесены
	if container.NormalizationHandler != nil {
		router.legacyNormalizationHandler = container.NormalizationHandler
	}

	// Получаем quality handler из контейнера (новая архитектура)
	qualityHandler, err := container.GetQualityHandler()
	if err == nil {
		router.qualityHandler = qualityHandler
	}

	// Получаем classification handler из контейнера (новая архитектура)
	classificationHandler, err := container.GetClassificationHandler()
	if err == nil {
		router.classificationHandler = classificationHandler
	}

	// Получаем client handler из контейнера (новая архитектура)
	clientHandler, err := container.GetClientHandler()
	if err == nil {
		router.clientHandler = clientHandler
	}

	// Получаем project handler из контейнера (новая архитектура)
	projectHandler, err := container.GetProjectHandler()
	if err == nil {
		router.projectHandler = projectHandler
	}

	// Получаем database handler из контейнера (новая архитектура)
	databaseHandler, err := container.GetDatabaseHandler()
	if err == nil {
		router.databaseHandler = databaseHandler
	}

	// Получаем worker handler из контейнера
	if container.WorkerHandler != nil {
		router.workerHandler = container.WorkerHandler
	}

	// Получаем web search handlers из контейнера
	if container.WebSearchValidationHandler != nil {
		router.webSearchValidationHandler = container.WebSearchValidationHandler
	}
	if container.WebSearchAdminHandler != nil {
		router.webSearchAdminHandler = container.WebSearchAdminHandler
	}

	return router, nil
}

// RegisterAllRoutes регистрирует все маршруты приложения
func (r *Router) RegisterAllRoutes(opts ...RegisterOptions) {
	var options RegisterOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	// Регистрируем upload routes (новая архитектура)
	if !options.SkipUploadRoutes && r.uploadHandler != nil {
		RegisterUploadRoutes(r.mux, &UploadHandlers{
			Handler: r.uploadHandler,
		})
	}

	// Регистрируем normalization routes (новая архитектура + legacy fallback)
	if !options.SkipNormalizationRoutes && (r.normalizationHandler != nil || r.legacyNormalizationHandler != nil) {
		handlers := &NormalizationHandlers{
			NewHandler: r.normalizationHandler,
		}
		if r.legacyNormalizationHandler != nil {
			handlers.OldHandler = r.legacyNormalizationHandler
		}
		RegisterNormalizationRoutes(r.mux, handlers)
	}

	// Регистрируем quality routes (новая архитектура)
	if !options.SkipQualityRoutes && r.qualityHandler != nil {
		RegisterQualityRoutes(r.mux, &QualityHandlers{
			NewHandler: r.qualityHandler,
		})
	}

	// Регистрируем classification routes (новая архитектура)
	if !options.SkipClassificationRoutes && r.classificationHandler != nil {
		RegisterClassificationRoutes(r.mux, &ClassificationHandlers{
			NewHandler: r.classificationHandler,
		})
	}

	// Регистрируем client routes (новая архитектура)
	if !options.SkipClientRoutes && r.clientHandler != nil {
		RegisterClientRoutes(r.mux, &ClientHandlers{
			NewHandler: r.clientHandler,
		})
	}

	// Регистрируем project routes (новая архитектура)
	if !options.SkipProjectRoutes && r.projectHandler != nil {
		RegisterProjectRoutes(r.mux, &ProjectHandlers{
			NewHandler: r.projectHandler,
		})
	}

	// Регистрируем database routes (новая архитектура)
	if !options.SkipDatabaseRoutes && r.databaseHandler != nil {
		RegisterDatabaseRoutes(r.mux, &DatabaseHandlers{
			NewHandler: r.databaseHandler,
		})
	}

	// Регистрируем worker routes (новая архитектура)
	if r.workerHandler != nil {
		RegisterWorkerRoutes(r.mux, &WorkerHandlers{
			Handler: r.workerHandler,
		})
	}

	// Регистрируем web search routes
	if !options.SkipWebSearchRoutes && (r.webSearchValidationHandler != nil || r.webSearchAdminHandler != nil) {
		RegisterWebSearchRoutes(r.mux, r.webSearchValidationHandler, r.webSearchAdminHandler)
	}

	// Регистрируем snapshot routes (если handler есть в контейнере)
	// SnapshotHandler регистрируется отдельно в server.go, так как требует много параметров

	// Регистрируем notification routes (если handler есть в контейнере)
	// NotificationHandler регистрируется отдельно в server.go
}

// SetWebSearchHandlers устанавливает обработчики веб-поиска
func (r *Router) SetWebSearchHandlers(validationHandler *handlers.WebSearchValidationHandler, adminHandler *handlers.WebSearchAdminHandler) {
	r.webSearchValidationHandler = validationHandler
	r.webSearchAdminHandler = adminHandler
}

// RegisterLegacyRoutes регистрирует legacy маршруты для обратной совместимости
// Эти маршруты будут постепенно заменяться новыми
func (r *Router) RegisterLegacyRoutes() {
	// Legacy routes будут регистрироваться в server.go
	// Этот метод можно использовать для дополнительных legacy маршрутов
}

// GetUploadHandler возвращает handler загрузок новой архитектуры
func (r *Router) GetUploadHandler() *upload.Handler {
	return r.uploadHandler
}
