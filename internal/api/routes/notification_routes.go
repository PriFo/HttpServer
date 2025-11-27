package routes

import (
	"net/http"

	"httpserver/server/handlers"
)

// NotificationHandlers содержит handlers для работы с уведомлениями
type NotificationHandlers struct {
	// Handler из server/handlers
	Handler *handlers.NotificationHandler
	// Legacy handlers для fallback (если нужны)
	HandleNotificationRoutes http.HandlerFunc
}

// RegisterNotificationRoutes регистрирует маршруты для уведомлений
func RegisterNotificationRoutes(mux *http.ServeMux, h *NotificationHandlers) {
	if h.Handler != nil {
		mux.HandleFunc("/api/notifications", h.Handler.HandleNotificationRoutes)
		mux.HandleFunc("/api/notifications/", h.Handler.HandleNotificationRoutes)
	} else {
		// Fallback к старым handlers
		if h.HandleNotificationRoutes != nil {
			mux.HandleFunc("/api/notifications", h.HandleNotificationRoutes)
			mux.HandleFunc("/api/notifications/", h.HandleNotificationRoutes)
		}
	}
}

