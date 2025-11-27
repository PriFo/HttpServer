package routes

import (
	"net/http"

	"httpserver/server/handlers"
)

// NomenclatureHandlers содержит handlers для работы с номенклатурой
type NomenclatureHandlers struct {
	// Handler из server/handlers
	Handler *handlers.NomenclatureHandler
	// Legacy handlers для fallback
	HandleStartProcessing    http.HandlerFunc
	HandleGetStatus          http.HandlerFunc
	HandleGetRecentRecords   http.HandlerFunc
	HandleGetPendingRecords  http.HandlerFunc
	ServeNomenclatureStatusPage http.HandlerFunc
}

// RegisterNomenclatureRoutes регистрирует маршруты для обработки номенклатуры
func RegisterNomenclatureRoutes(mux *http.ServeMux, h *NomenclatureHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/nomenclature/process", h.Handler.HandleStartProcessing)
		mux.HandleFunc("/api/nomenclature/status", h.Handler.HandleGetStatus)
		mux.HandleFunc("/api/nomenclature/recent", h.Handler.HandleGetRecentRecords)
		mux.HandleFunc("/api/nomenclature/pending", h.Handler.HandleGetPendingRecords)
	} else {
		// Fallback к старым handlers
		mux.HandleFunc("/api/nomenclature/process", h.HandleStartProcessing)
		mux.HandleFunc("/api/nomenclature/status", h.HandleGetStatus)
		mux.HandleFunc("/api/nomenclature/recent", h.HandleGetRecentRecords)
		mux.HandleFunc("/api/nomenclature/pending", h.HandleGetPendingRecords)
	}

	// HTML страница остается в server.go, но регистрируем здесь для консистентности
	if h.ServeNomenclatureStatusPage != nil {
		mux.HandleFunc("/nomenclature/status", h.ServeNomenclatureStatusPage)
	}
}

