package routes

import (
	"net/http"

	"httpserver/database"
	"httpserver/server/handlers"
)

// QualityLegacyHandlers содержит legacy handlers для качества данных
type QualityLegacyHandlers struct {
	// Handler из server/handlers
	Handler *handlers.QualityHandler
	// Legacy handlers для fallback
	HandleQualityStats              http.HandlerFunc
	HandleQualityCacheStats         http.HandlerFunc
	HandleQualityCacheInvalidate    http.HandlerFunc
	HandleGetQualityReport          http.HandlerFunc
	HandleQualityItemDetail         http.HandlerFunc
	HandleQualityViolations         http.HandlerFunc
	HandleQualityViolationDetail    http.HandlerFunc
	HandleQualitySuggestions        http.HandlerFunc
	HandleQualitySuggestionAction   http.HandlerFunc
	HandleQualityDuplicates         http.HandlerFunc
	HandleQualityDuplicateAction    http.HandlerFunc
	HandleQualityAssess             http.HandlerFunc
	HandleQualityAnalyze            http.HandlerFunc
	HandleQualityAnalyzeStatus      http.HandlerFunc
	HandleQualityUploadRoutes       http.HandlerFunc
	HandleQualityDatabaseRoutes     http.HandlerFunc
	// Дополнительные параметры для HandleQualityStats
	DB                      *database.DB
	CurrentNormalizedDBPath string
	HandleDatabaseV1Routes  http.HandlerFunc
}

// RegisterQualityLegacyRoutes регистрирует legacy маршруты для качества данных
func RegisterQualityLegacyRoutes(mux *http.ServeMux, h *QualityLegacyHandlers) {
	// Используем handler если доступен
	if h.Handler != nil {
		// HandleQualityStats требует дополнительные параметры
		if h.DB != nil && h.CurrentNormalizedDBPath != "" {
			mux.HandleFunc("/api/quality/stats", func(w http.ResponseWriter, r *http.Request) {
				h.Handler.HandleQualityStats(w, r, h.DB, h.CurrentNormalizedDBPath)
			})
		}
		mux.HandleFunc("/api/quality/cache/stats", h.Handler.HandleQualityCacheStats)
		mux.HandleFunc("/api/quality/cache/invalidate", h.Handler.HandleQualityCacheInvalidate)
		mux.HandleFunc("/api/quality/report", h.Handler.HandleGetQualityReport)
		mux.HandleFunc("/api/quality/item/", h.Handler.HandleQualityItemDetail)
		mux.HandleFunc("/api/quality/violations", h.Handler.HandleQualityViolations)
		mux.HandleFunc("/api/quality/violations/", h.Handler.HandleQualityViolationDetail)
		mux.HandleFunc("/api/quality/suggestions", h.Handler.HandleQualitySuggestions)
		mux.HandleFunc("/api/quality/suggestions/", h.Handler.HandleQualitySuggestionAction)
		mux.HandleFunc("/api/quality/duplicates", h.Handler.HandleQualityDuplicates)
		mux.HandleFunc("/api/quality/duplicates/", h.Handler.HandleQualityDuplicateAction)
		mux.HandleFunc("/api/quality/assess", h.Handler.HandleQualityAssess)
		mux.HandleFunc("/api/quality/analyze", h.Handler.HandleQualityAnalyze)
		mux.HandleFunc("/api/quality/analyze/status", h.Handler.HandleQualityAnalyzeStatus)
		
		// HandleQualityUploadRoutes и HandleQualityDatabaseRoutes
		if h.HandleQualityUploadRoutes != nil {
			mux.HandleFunc("/api/v1/upload/", h.Handler.HandleQualityUploadRoutes)
		}
		if h.HandleQualityDatabaseRoutes != nil && h.HandleDatabaseV1Routes != nil {
			mux.HandleFunc("/api/v1/databases/", func(w http.ResponseWriter, r *http.Request) {
				h.Handler.HandleQualityDatabaseRoutes(w, r, h.HandleDatabaseV1Routes)
			})
		}
		return
	}

	// Fallback к legacy handlers
	if h.HandleQualityStats != nil {
		mux.HandleFunc("/api/quality/stats", h.HandleQualityStats)
	}
	if h.HandleQualityCacheStats != nil {
		mux.HandleFunc("/api/quality/cache/stats", h.HandleQualityCacheStats)
	}
	if h.HandleQualityCacheInvalidate != nil {
		mux.HandleFunc("/api/quality/cache/invalidate", h.HandleQualityCacheInvalidate)
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
	if h.HandleQualityUploadRoutes != nil {
		mux.HandleFunc("/api/v1/upload/", h.HandleQualityUploadRoutes)
	}
	if h.HandleQualityDatabaseRoutes != nil {
		mux.HandleFunc("/api/v1/databases/", h.HandleQualityDatabaseRoutes)
	}
}

