package routes

import (
	"net/http"

	"httpserver/internal/api/handlers/client"
)

// ClientHandlers содержит handlers для работы с клиентами
type ClientHandlers struct {
	// Новый handler из internal/api/handlers/client
	NewHandler *client.Handler
	// Старый handler из server/handlers (через интерфейс)
	OldHandler interface {
		HandleClients(http.ResponseWriter, *http.Request)
		HandleClientRoutes(http.ResponseWriter, *http.Request)
	}
	// Legacy handlers для fallback
	HandleClients      http.HandlerFunc
	HandleClientRoutes http.HandlerFunc
}

// RegisterClientRoutes регистрирует маршруты для работы с клиентами
func RegisterClientRoutes(mux *http.ServeMux, h *ClientHandlers) {
	// Используем новый handler если доступен
	if h.NewHandler != nil {
		mux.HandleFunc("/api/clients", h.NewHandler.HandleClients)
		mux.HandleFunc("/api/clients/", h.NewHandler.HandleClientRoutes)
		return
	}

	// Используем старый handler если доступен
	if h.OldHandler != nil {
		mux.HandleFunc("/api/clients", h.OldHandler.HandleClients)
		mux.HandleFunc("/api/clients/", h.OldHandler.HandleClientRoutes)
		return
	}

	// Fallback к legacy handlers
	if h.HandleClients != nil {
		mux.HandleFunc("/api/clients", h.HandleClients)
	}
	if h.HandleClientRoutes != nil {
		mux.HandleFunc("/api/clients/", h.HandleClientRoutes)
	}
}
