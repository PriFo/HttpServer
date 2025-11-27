package routes

import (
	"net/http"
	"strings"

	"httpserver/server/handlers"
)

// CounterpartyHandlers содержит handlers для работы с контрагентами
type CounterpartyHandlers struct {
	// Handler из server/handlers
	Handler *handlers.CounterpartyHandler
	// Legacy handlers для fallback
	HandleCounterpartyNormalizationStopCheckPerformance     http.HandlerFunc
	HandleCounterpartyNormalizationStopCheckPerformanceReset http.HandlerFunc
	HandleNormalizedCounterparties                          http.HandlerFunc
	HandleNormalizedCounterpartyRoutes                      http.HandlerFunc
	HandleGetAllCounterparties                              http.HandlerFunc
	HandleExportAllCounterparties                           http.HandlerFunc
	HandleBulkUpdateCounterparties                          http.HandlerFunc
	HandleBulkDeleteCounterparties                          http.HandlerFunc
	HandleBulkEnrichCounterparties                          http.HandlerFunc
	HandleCounterpartyDuplicates                            http.HandlerFunc
	HandleCounterpartyDuplicateRoutes                       http.HandlerFunc
}

// RegisterCounterpartyRoutes регистрирует маршруты для работы с контрагентами
func RegisterCounterpartyRoutes(mux *http.ServeMux, h *CounterpartyHandlers) {
	// Используем handler если доступен
	if h.Handler != nil {
		// Метрики производительности проверок остановки нормализации
		mux.HandleFunc("/api/counterparty/normalization/stop-check/performance", h.Handler.HandleCounterpartyNormalizationStopCheckPerformance)
		mux.HandleFunc("/api/counterparty/normalization/stop-check/performance/reset", h.Handler.HandleCounterpartyNormalizationStopCheckPerformanceReset)
		// Нормализованные контрагенты
		mux.HandleFunc("/api/counterparties/normalized", h.Handler.HandleNormalizedCounterparties)
		mux.HandleFunc("/api/counterparties/normalized/", h.Handler.HandleNormalizedCounterpartyRoutes)
		// Автоматический мэппинг контрагентов
		mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
			// Обрабатываем /api/projects/{projectId}/counterparties/auto-map
			path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
			parts := strings.Split(path, "/")
			if len(parts) >= 3 && parts[1] == "counterparties" && parts[2] == "auto-map" {
				h.Handler.HandleAutoMapCounterparties(w, r)
				return
			}
			if len(parts) >= 3 && parts[1] == "counterparties" && parts[2] == "mapping-status" {
				h.Handler.HandleMappingStatus(w, r)
				return
			}
			if len(parts) >= 3 && parts[1] == "counterparties" && parts[2] == "normalization-config" {
				if r.Method == http.MethodGet {
					h.Handler.HandleMappingStatus(w, r)
				} else if r.Method == http.MethodPut || r.Method == http.MethodPost {
					h.Handler.HandleUpdateNormalizationConfig(w, r)
				} else {
					http.NotFound(w, r)
				}
				return
			}
			if len(parts) >= 3 && parts[1] == "counterparties" && parts[2] == "merge-duplicates" {
				h.Handler.HandleMergeCounterpartyDuplicates(w, r, 0)
				return
			}
			http.NotFound(w, r)
		})
		mux.HandleFunc("/api/counterparties/all", h.Handler.HandleGetAllCounterparties)
		mux.HandleFunc("/api/counterparties/all/export", h.Handler.HandleExportAllCounterparties)
		mux.HandleFunc("/api/counterparties/bulk/update", h.Handler.HandleBulkUpdateCounterparties)
		mux.HandleFunc("/api/counterparties/bulk/delete", h.Handler.HandleBulkDeleteCounterparties)
		mux.HandleFunc("/api/counterparties/bulk/enrich", h.Handler.HandleBulkEnrichCounterparties)
		mux.HandleFunc("/api/counterparties/duplicates", h.Handler.HandleCounterpartyDuplicates)
		mux.HandleFunc("/api/counterparties/duplicates/", h.Handler.HandleCounterpartyDuplicateRoutes)
		// Автоматический мэппинг контрагентов
		mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
			// Обрабатываем /api/projects/{projectId}/counterparties/auto-map
			path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
			parts := strings.Split(path, "/")
			if len(parts) >= 3 && parts[1] == "counterparties" && parts[2] == "auto-map" {
				h.Handler.HandleAutoMapCounterparties(w, r)
				return
			}
			if len(parts) >= 3 && parts[1] == "counterparties" && parts[2] == "mapping-status" {
				h.Handler.HandleMappingStatus(w, r)
				return
			}
			if len(parts) >= 3 && parts[1] == "counterparties" && parts[2] == "merge-duplicates" {
				h.Handler.HandleMergeCounterpartyDuplicates(w, r, 0)
				return
			}
			http.NotFound(w, r)
		})
		return
	}

	// Fallback к legacy handlers
	if h.HandleCounterpartyNormalizationStopCheckPerformance != nil {
		mux.HandleFunc("/api/counterparty/normalization/stop-check/performance", h.HandleCounterpartyNormalizationStopCheckPerformance)
	}
	if h.HandleCounterpartyNormalizationStopCheckPerformanceReset != nil {
		mux.HandleFunc("/api/counterparty/normalization/stop-check/performance/reset", h.HandleCounterpartyNormalizationStopCheckPerformanceReset)
	}
	if h.HandleNormalizedCounterparties != nil {
		mux.HandleFunc("/api/counterparties/normalized", h.HandleNormalizedCounterparties)
	}
	if h.HandleNormalizedCounterpartyRoutes != nil {
		mux.HandleFunc("/api/counterparties/normalized/", h.HandleNormalizedCounterpartyRoutes)
	}
	if h.HandleGetAllCounterparties != nil {
		mux.HandleFunc("/api/counterparties/all", h.HandleGetAllCounterparties)
	}
	if h.HandleExportAllCounterparties != nil {
		mux.HandleFunc("/api/counterparties/all/export", h.HandleExportAllCounterparties)
	}
	if h.HandleBulkUpdateCounterparties != nil {
		mux.HandleFunc("/api/counterparties/bulk/update", h.HandleBulkUpdateCounterparties)
	}
	if h.HandleBulkDeleteCounterparties != nil {
		mux.HandleFunc("/api/counterparties/bulk/delete", h.HandleBulkDeleteCounterparties)
	}
	if h.HandleBulkEnrichCounterparties != nil {
		mux.HandleFunc("/api/counterparties/bulk/enrich", h.HandleBulkEnrichCounterparties)
	}
	if h.HandleCounterpartyDuplicates != nil {
		mux.HandleFunc("/api/counterparties/duplicates", h.HandleCounterpartyDuplicates)
	}
	if h.HandleCounterpartyDuplicateRoutes != nil {
		mux.HandleFunc("/api/counterparties/duplicates/", h.HandleCounterpartyDuplicateRoutes)
	}
}

