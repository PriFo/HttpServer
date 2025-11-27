package routes

import (
	"net/http"

	"httpserver/internal/api/handlers/database"
)

// DatabaseHandlers содержит handlers для работы с базами данных
type DatabaseHandlers struct {
	// Новый handler из internal/api/handlers/database
	NewHandler *database.Handler
	// Старый handler из server/handlers (через интерфейс)
	OldHandler interface {
		HandleDatabases(http.ResponseWriter, *http.Request)
		HandleDatabaseRoutes(http.ResponseWriter, *http.Request)
	}
	// Legacy handlers для fallback
	HandleDatabases      http.HandlerFunc
	HandleDatabaseRoutes http.HandlerFunc
}

// RegisterDatabaseRoutes регистрирует маршруты для работы с базами данных
func RegisterDatabaseRoutes(mux *http.ServeMux, h *DatabaseHandlers) {
	// Используем новый handler если доступен
	if h.NewHandler != nil {
		mux.HandleFunc("/api/v2/databases", h.NewHandler.HandleDatabases)
		mux.HandleFunc("/api/v2/databases/", h.NewHandler.HandleDatabaseRoutes)
		return
	}

	// Используем старый handler если доступен
	if h.OldHandler != nil {
		mux.HandleFunc("/api/v2/databases", h.OldHandler.HandleDatabases)
		mux.HandleFunc("/api/v2/databases/", h.OldHandler.HandleDatabaseRoutes)
		return
	}

	// Fallback к legacy handlers
	if h.HandleDatabases != nil {
		mux.HandleFunc("/api/v2/databases", h.HandleDatabases)
	}
	if h.HandleDatabaseRoutes != nil {
		mux.HandleFunc("/api/v2/databases/", h.HandleDatabaseRoutes)
	}
}
