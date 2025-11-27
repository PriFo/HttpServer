package routes

import (
	"net/http"

	"httpserver/server/handlers"
)

// SnapshotHandlers содержит handlers для работы со срезами данных
type SnapshotHandlers struct {
	// Handler из server/handlers
	Handler *handlers.SnapshotHandler
	// Legacy handlers для fallback (если нужны)
	HandleSnapshotsRoutes      http.HandlerFunc
	HandleSnapshotRoutes        http.HandlerFunc
	HandleCreateAutoSnapshot    http.HandlerFunc
	HandleProjectSnapshotsRoutes http.HandlerFunc
}

// RegisterSnapshotRoutes регистрирует маршруты для работы со срезами данных
func RegisterSnapshotRoutes(mux *http.ServeMux, h *SnapshotHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/snapshots", h.Handler.HandleSnapshotsRoutes)
		mux.HandleFunc("/api/snapshots/", h.Handler.HandleSnapshotRoutes)
		mux.HandleFunc("/api/snapshots/auto", h.Handler.HandleCreateAutoSnapshot)
		mux.HandleFunc("/api/projects/", h.Handler.HandleProjectSnapshotsRoutes)
	} else {
		// Fallback к старым handlers
		if h.HandleSnapshotsRoutes != nil {
			mux.HandleFunc("/api/snapshots", h.HandleSnapshotsRoutes)
		}
		if h.HandleSnapshotRoutes != nil {
			mux.HandleFunc("/api/snapshots/", h.HandleSnapshotRoutes)
		}
		if h.HandleCreateAutoSnapshot != nil {
			mux.HandleFunc("/api/snapshots/auto", h.HandleCreateAutoSnapshot)
		}
		if h.HandleProjectSnapshotsRoutes != nil {
			mux.HandleFunc("/api/projects/", h.HandleProjectSnapshotsRoutes)
		}
	}
}

