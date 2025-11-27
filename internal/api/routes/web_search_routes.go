package routes

import (
	"net/http"

	"httpserver/server/handlers"
)

// RegisterWebSearchRoutes регистрирует маршруты для веб-поиска валидации
func RegisterWebSearchRoutes(mux *http.ServeMux, validationHandler *handlers.WebSearchValidationHandler, adminHandler *handlers.WebSearchAdminHandler) {
	// API endpoints для валидации
	if validationHandler != nil {
		mux.HandleFunc("/api/validation/websearch", validationHandler.HandleValidateProduct)
		mux.HandleFunc("/api/validation/websearch/search", validationHandler.HandleSearch)
	}
	
	// Основной endpoint для веб-поиска (если есть WebSearchHandler)
	// WebSearchHandler регистрируется отдельно в server.go

	// Административные API endpoints
	if adminHandler != nil {
		mux.HandleFunc("/api/admin/websearch/providers", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				adminHandler.HandleListProviders(w, r)
			} else if r.Method == http.MethodPost {
				adminHandler.HandleCreateProvider(w, r)
			}
		})
		
		mux.HandleFunc("/api/admin/websearch/providers/reload", adminHandler.HandleReloadProviders)
		mux.HandleFunc("/api/admin/websearch/stats", adminHandler.HandleGetStats)
		
		// Маршрут для обновления/удаления конкретного провайдера
		mux.HandleFunc("/api/admin/websearch/providers/", func(w http.ResponseWriter, r *http.Request) {
			// Извлекаем имя провайдера из пути
			name := r.URL.Path[len("/api/admin/websearch/providers/"):]
			if name == "" {
				http.Error(w, "Provider name is required", http.StatusBadRequest)
				return
			}
			
			if r.Method == http.MethodPut {
				adminHandler.HandleUpdateProvider(w, r, name)
			} else if r.Method == http.MethodDelete {
				adminHandler.HandleDeleteProvider(w, r, name)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}
}

