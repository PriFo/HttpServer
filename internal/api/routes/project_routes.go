package routes

import (
	"net/http"

	"httpserver/internal/api/handlers/project"
)

// ProjectHandlers содержит handlers для работы с проектами
type ProjectHandlers struct {
	// Новый handler из internal/api/handlers/project
	NewHandler *project.Handler
	// Старый handler из server/handlers (через интерфейс)
	OldHandler interface {
		HandleProjects(http.ResponseWriter, *http.Request)
		HandleProjectRoutes(http.ResponseWriter, *http.Request)
	}
	// Legacy handlers для fallback
	HandleProjects      http.HandlerFunc
	HandleProjectRoutes http.HandlerFunc
}

// RegisterProjectRoutes регистрирует маршруты для работы с проектами
func RegisterProjectRoutes(mux *http.ServeMux, h *ProjectHandlers) {
	// Используем новый handler если доступен
	if h.NewHandler != nil {
		mux.HandleFunc("/api/v2/projects", h.NewHandler.HandleProjects)
		mux.HandleFunc("/api/v2/projects/", h.NewHandler.HandleProjectRoutes)
		return
	}

	// Используем старый handler если доступен
	if h.OldHandler != nil {
		mux.HandleFunc("/api/v2/projects", h.OldHandler.HandleProjects)
		mux.HandleFunc("/api/v2/projects/", h.OldHandler.HandleProjectRoutes)
		return
	}

	// Fallback к legacy handlers
	if h.HandleProjects != nil {
		mux.HandleFunc("/api/v2/projects", h.HandleProjects)
	}
	if h.HandleProjectRoutes != nil {
		mux.HandleFunc("/api/v2/projects/", h.HandleProjectRoutes)
	}
}
