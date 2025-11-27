package routes

import (
	"log"
	"net/http"

	"httpserver/server/handlers"
	"httpserver/server/monitoring"
)

// ServerHandlers содержит все handlers сервера для регистрации маршрутов
type ServerHandlers struct {
	// System handlers
	SystemHandler        *handlers.SystemHandler
	SystemSummaryHandler *handlers.SystemSummaryHandler
	HealthChecker        *monitoring.HealthChecker
	MetricsCollector     *monitoring.MetricsCollector

	// Monitoring handlers
	MonitoringHandler *handlers.MonitoringHandler
	ErrorMetricsHandler *handlers.ErrorMetricsHandler

	// Legacy handlers (для fallback)
	HandleStats                http.HandlerFunc
	HandleHealth               http.HandlerFunc
	HandlePerformanceMetrics  http.HandlerFunc
	HandleSystemSummary        http.HandlerFunc
	HandleSystemSummaryExport  http.HandlerFunc
	HandleSystemSummaryHistory http.HandlerFunc
	HandleSystemSummaryCompare http.HandlerFunc
	HandleSystemSummaryStream  http.HandlerFunc
	HandleSystemSummaryCacheStats      http.HandlerFunc
	HandleSystemSummaryCacheInvalidate http.HandlerFunc
	HandleSystemSummaryCacheClear      http.HandlerFunc
	HandleSystemSummaryHealth          http.HandlerFunc
	HandleMonitoringMetrics   http.HandlerFunc
	HandleMonitoringCache     http.HandlerFunc
	HandleMonitoringAI        http.HandlerFunc
	HandleMonitoringHistory   http.HandlerFunc
	HandleMonitoringEvents    http.HandlerFunc
	HandleMonitoringProvidersStream http.HandlerFunc
	HandleMonitoringProviders http.HandlerFunc
}

// RegisterSystemRoutes регистрирует системные маршруты
func RegisterSystemRoutes(mux *http.ServeMux, h *ServerHandlers) {
	// Регистрируем системные endpoints
	if h.SystemHandler != nil {
		mux.HandleFunc("/stats", h.SystemHandler.HandleStats)
		mux.HandleFunc("/health", h.SystemHandler.HandleHealth)
		mux.HandleFunc("/api/v1/health", h.SystemHandler.HandleHealth)
		if h.MetricsCollector != nil {
			mux.HandleFunc("/api/monitoring/performance", h.SystemHandler.HandlePerformanceMetrics)
		}
	} else {
		// Fallback к старым handlers
		if h.HandleStats != nil {
			mux.HandleFunc("/stats", h.HandleStats)
		}
		if h.HandleHealth != nil {
			mux.HandleFunc("/health", h.HandleHealth)
			mux.HandleFunc("/api/v1/health", h.HandleHealth)
		}
		if h.MetricsCollector != nil && h.HandlePerformanceMetrics != nil {
			mux.HandleFunc("/api/monitoring/performance", h.HandlePerformanceMetrics)
		}
	}

	// Health check endpoints для Kubernetes
	if h.HealthChecker != nil {
		mux.HandleFunc("/health/live", h.HealthChecker.LivenessHandler())
		mux.HandleFunc("/health/ready", h.HealthChecker.ReadinessHandler())
	}

	// Регистрируем endpoints для системных сводок
	if h.SystemSummaryHandler != nil {
		mux.HandleFunc("/api/system/summary/cache/stats", h.SystemSummaryHandler.HandleSystemSummaryCacheStats)
		mux.HandleFunc("/api/system/summary/cache/invalidate", h.SystemSummaryHandler.HandleSystemSummaryCacheInvalidate)
		mux.HandleFunc("/api/system/summary/cache/clear", h.SystemSummaryHandler.HandleSystemSummaryCacheClear)
		mux.HandleFunc("/api/system/summary/health", h.SystemSummaryHandler.HandleSystemSummaryHealth)
	} else {
		// Fallback к старым handlers
		if h.HandleSystemSummaryCacheStats != nil {
			mux.HandleFunc("/api/system/summary/cache/stats", h.HandleSystemSummaryCacheStats)
		}
		if h.HandleSystemSummaryCacheInvalidate != nil {
			mux.HandleFunc("/api/system/summary/cache/invalidate", h.HandleSystemSummaryCacheInvalidate)
		}
		if h.HandleSystemSummaryCacheClear != nil {
			mux.HandleFunc("/api/system/summary/cache/clear", h.HandleSystemSummaryCacheClear)
		}
		if h.HandleSystemSummaryHealth != nil {
			mux.HandleFunc("/api/system/summary/health", h.HandleSystemSummaryHealth)
		}
	}

	// Остальные системные endpoints пока остаются в server.go (handleSystemSummary, handleSystemSummaryExport, etc.)
	if h.HandleSystemSummary != nil {
		mux.HandleFunc("/api/system/summary", h.HandleSystemSummary)
	}
	if h.HandleSystemSummaryExport != nil {
		mux.HandleFunc("/api/system/summary/export", h.HandleSystemSummaryExport)
	}
	if h.HandleSystemSummaryHistory != nil {
		mux.HandleFunc("/api/system/summary/history", h.HandleSystemSummaryHistory)
	}
	if h.HandleSystemSummaryCompare != nil {
		mux.HandleFunc("/api/system/summary/compare", h.HandleSystemSummaryCompare)
	}
	if h.HandleSystemSummaryStream != nil {
		mux.HandleFunc("/api/system/summary/stream", h.HandleSystemSummaryStream)
	}

	// Регистрируем эндпоинты для мониторинга производительности
	if h.MonitoringHandler != nil {
		mux.HandleFunc("/api/monitoring/metrics", h.MonitoringHandler.HandleMonitoringMetrics)
		mux.HandleFunc("/api/monitoring/cache", h.MonitoringHandler.HandleMonitoringCache)
		mux.HandleFunc("/api/monitoring/ai", h.MonitoringHandler.HandleMonitoringAI)
		mux.HandleFunc("/api/monitoring/history", h.MonitoringHandler.HandleMonitoringHistory)
		mux.HandleFunc("/api/monitoring/events", h.MonitoringHandler.HandleMonitoringEvents)
		mux.HandleFunc("/api/monitoring/providers/stream", h.MonitoringHandler.HandleMonitoringProvidersStream)
		mux.HandleFunc("/api/monitoring/providers", h.MonitoringHandler.HandleMonitoringProviders)
	} else {
		// Fallback к старым handlers
		if h.HandleMonitoringMetrics != nil {
			mux.HandleFunc("/api/monitoring/metrics", h.HandleMonitoringMetrics)
		}
		if h.HandleMonitoringCache != nil {
			mux.HandleFunc("/api/monitoring/cache", h.HandleMonitoringCache)
		}
		if h.HandleMonitoringAI != nil {
			mux.HandleFunc("/api/monitoring/ai", h.HandleMonitoringAI)
		}
		if h.HandleMonitoringHistory != nil {
			mux.HandleFunc("/api/monitoring/history", h.HandleMonitoringHistory)
		}
		if h.HandleMonitoringEvents != nil {
			mux.HandleFunc("/api/monitoring/events", h.HandleMonitoringEvents)
		}
		if h.HandleMonitoringProvidersStream != nil {
			mux.HandleFunc("/api/monitoring/providers/stream", h.HandleMonitoringProvidersStream)
			log.Printf("Registered /api/monitoring/providers/stream via fallback handler")
		} else {
			log.Printf("Warning: HandleMonitoringProvidersStream is nil, /api/monitoring/providers/stream will not be available")
		}
		if h.HandleMonitoringProviders != nil {
			mux.HandleFunc("/api/monitoring/providers", h.HandleMonitoringProviders)
			log.Printf("Registered /api/monitoring/providers via fallback handler")
		} else {
			log.Printf("Warning: HandleMonitoringProviders is nil, /api/monitoring/providers will not be available")
		}
	}

	// Регистрируем эндпоинты для метрик ошибок
	if h.ErrorMetricsHandler != nil {
		mux.HandleFunc("/api/errors/metrics", h.ErrorMetricsHandler.GetErrorMetrics)
		mux.HandleFunc("/api/errors/by-type", h.ErrorMetricsHandler.GetErrorsByType)
		mux.HandleFunc("/api/errors/by-code", h.ErrorMetricsHandler.GetErrorsByCode)
		mux.HandleFunc("/api/errors/by-endpoint", h.ErrorMetricsHandler.GetErrorsByEndpoint)
		mux.HandleFunc("/api/errors/last", h.ErrorMetricsHandler.GetLastErrors)
		mux.HandleFunc("/api/errors/reset", h.ErrorMetricsHandler.ResetErrorMetrics)
	}
}

