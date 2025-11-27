package routes

import (
	"net/http"

	"httpserver/database"
	"httpserver/internal/api/handlers/quality"
)

// QualityHandlers содержит handlers для качества данных
type QualityHandlers struct {
	// Новый handler из internal/api/handlers/quality
	NewHandler *quality.Handler
	// Старый handler из server/handlers (через интерфейс)
	OldHandler interface {
		HandleQualityUploadRoutes(http.ResponseWriter, *http.Request)
		HandleQualityDatabaseRoutes(http.ResponseWriter, *http.Request, func(http.ResponseWriter, *http.Request))
		HandleQualityStats(http.ResponseWriter, *http.Request, *database.DB, string)
		HandleGetQualityReport(http.ResponseWriter, *http.Request)
		HandleQualityItemDetail(http.ResponseWriter, *http.Request)
		HandleQualityViolations(http.ResponseWriter, *http.Request)
		HandleQualityViolationDetail(http.ResponseWriter, *http.Request)
		HandleQualitySuggestions(http.ResponseWriter, *http.Request)
		HandleQualitySuggestionAction(http.ResponseWriter, *http.Request)
		HandleQualityDuplicates(http.ResponseWriter, *http.Request)
		HandleQualityDuplicateAction(http.ResponseWriter, *http.Request)
		HandleQualityAssess(http.ResponseWriter, *http.Request)
		HandleQualityAnalyze(http.ResponseWriter, *http.Request)
		HandleQualityAnalyzeStatus(http.ResponseWriter, *http.Request)
	}
	// Legacy handlers для fallback
	DB                      interface{}
	CurrentNormalizedDBPath string
	HandleDatabaseV1Routes  func(http.ResponseWriter, *http.Request)
	HandleQualityUploadRoutes http.HandlerFunc
	HandleQualityDatabaseRoutes http.HandlerFunc
	HandleQualityStats      http.HandlerFunc
	HandleGetQualityReport  http.HandlerFunc
	HandleQualityItemDetail http.HandlerFunc
	HandleQualityViolations http.HandlerFunc
	HandleQualityViolationDetail http.HandlerFunc
	HandleQualitySuggestions http.HandlerFunc
	HandleQualitySuggestionAction http.HandlerFunc
	HandleQualityDuplicates http.HandlerFunc
	HandleQualityDuplicateAction http.HandlerFunc
	HandleQualityAssess     http.HandlerFunc
	HandleQualityAnalyze    http.HandlerFunc
	HandleQualityAnalyzeStatus http.HandlerFunc
}

// RegisterQualityRoutes регистрирует маршруты для quality
func RegisterQualityRoutes(mux *http.ServeMux, h *QualityHandlers) {
	// Используем новый handler если доступен
	if h.NewHandler != nil {
		mux.HandleFunc("/api/v1/upload/quality/analyze", h.NewHandler.HandleAnalyzeQuality)
		mux.HandleFunc("/api/v1/upload/quality/report", h.NewHandler.HandleGetQualityReport)
		mux.HandleFunc("/api/v1/databases/quality/dashboard", h.NewHandler.HandleGetQualityDashboard)
		mux.HandleFunc("/api/v1/quality/issues", h.NewHandler.HandleGetQualityIssues)
		mux.HandleFunc("/api/v1/databases/quality/statistics", h.NewHandler.HandleGetQualityStatistics)
		mux.HandleFunc("/api/v1/databases/quality/trends", h.NewHandler.HandleGetQualityTrends)
	}

	// Используем старый handler если доступен (через legacy handlers)
	if h.OldHandler != nil {
		// Регистрируем эндпоинты качества данных (до общих маршрутов для приоритета)
		mux.HandleFunc("/api/v1/upload/", h.OldHandler.HandleQualityUploadRoutes)
		if h.HandleDatabaseV1Routes != nil {
			mux.HandleFunc("/api/v1/databases/", func(w http.ResponseWriter, r *http.Request) {
				h.OldHandler.HandleQualityDatabaseRoutes(w, r, h.HandleDatabaseV1Routes)
			})
		}
		// Регистрируем эндпоинты для качества нормализации
		if h.DB != nil {
			// Делаем type assertion для h.DB
			if db, ok := h.DB.(*database.DB); ok {
				mux.HandleFunc("/api/quality/stats", func(w http.ResponseWriter, r *http.Request) {
					h.OldHandler.HandleQualityStats(w, r, db, h.CurrentNormalizedDBPath)
				})
			}
		}
		mux.HandleFunc("/api/quality/report", h.OldHandler.HandleGetQualityReport)
		mux.HandleFunc("/api/quality/item/", h.OldHandler.HandleQualityItemDetail)
		mux.HandleFunc("/api/quality/violations", h.OldHandler.HandleQualityViolations)
		mux.HandleFunc("/api/quality/violations/", h.OldHandler.HandleQualityViolationDetail)
		mux.HandleFunc("/api/quality/suggestions", h.OldHandler.HandleQualitySuggestions)
		mux.HandleFunc("/api/quality/suggestions/", h.OldHandler.HandleQualitySuggestionAction)
		mux.HandleFunc("/api/quality/duplicates", h.OldHandler.HandleQualityDuplicates)
		mux.HandleFunc("/api/quality/duplicates/", h.OldHandler.HandleQualityDuplicateAction)
		mux.HandleFunc("/api/quality/assess", h.OldHandler.HandleQualityAssess)
		mux.HandleFunc("/api/quality/analyze", h.OldHandler.HandleQualityAnalyze)
		mux.HandleFunc("/api/quality/analyze/status", h.OldHandler.HandleQualityAnalyzeStatus)
		
		// Регистрируем endpoints для управления кэшем качества проектов
		// Проверяем, что OldHandler поддерживает методы кэша
		if cacheHandler, ok := h.OldHandler.(interface {
			HandleQualityCacheStats(http.ResponseWriter, *http.Request)
			HandleQualityCacheInvalidate(http.ResponseWriter, *http.Request)
		}); ok {
			mux.HandleFunc("/api/quality/cache/stats", cacheHandler.HandleQualityCacheStats)
			mux.HandleFunc("/api/quality/cache/invalidate", cacheHandler.HandleQualityCacheInvalidate)
		}
		return
	}

	// Fallback к legacy handlers
	if h.HandleQualityUploadRoutes != nil || h.HandleQualityDatabaseRoutes != nil {
		// Fallback к legacy handlers
		if h.HandleQualityUploadRoutes != nil {
			mux.HandleFunc("/api/v1/upload/", h.HandleQualityUploadRoutes)
		}
		if h.HandleQualityDatabaseRoutes != nil {
			mux.HandleFunc("/api/v1/databases/", h.HandleQualityDatabaseRoutes)
		}
		if h.HandleQualityStats != nil {
			mux.HandleFunc("/api/quality/stats", h.HandleQualityStats)
		}
		if h.HandleGetQualityReport != nil {
			mux.HandleFunc("/api/quality/report", h.HandleGetQualityReport)
		}
		if h.HandleQualityItemDetail != nil {
			mux.HandleFunc("/api/quality/item/", h.HandleQualityItemDetail)
		}
		if h.HandleQualityViolations != nil {
			mux.HandleFunc("/api/quality/violations", h.HandleQualityViolations)
		}
		if h.HandleQualityViolationDetail != nil {
			mux.HandleFunc("/api/quality/violations/", h.HandleQualityViolationDetail)
		}
		if h.HandleQualitySuggestions != nil {
			mux.HandleFunc("/api/quality/suggestions", h.HandleQualitySuggestions)
		}
		if h.HandleQualitySuggestionAction != nil {
			mux.HandleFunc("/api/quality/suggestions/", h.HandleQualitySuggestionAction)
		}
		if h.HandleQualityDuplicates != nil {
			mux.HandleFunc("/api/quality/duplicates", h.HandleQualityDuplicates)
		}
		if h.HandleQualityDuplicateAction != nil {
			mux.HandleFunc("/api/quality/duplicates/", h.HandleQualityDuplicateAction)
		}
		if h.HandleQualityAssess != nil {
			mux.HandleFunc("/api/quality/assess", h.HandleQualityAssess)
		}
		if h.HandleQualityAnalyze != nil {
			mux.HandleFunc("/api/quality/analyze", h.HandleQualityAnalyze)
		}
		if h.HandleQualityAnalyzeStatus != nil {
			mux.HandleFunc("/api/quality/analyze/status", h.HandleQualityAnalyzeStatus)
		}
	}
}

