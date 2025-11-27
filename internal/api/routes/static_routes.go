package routes

import (
	"net/http"
)

// RegisterStaticRoutes регистрирует маршруты для статического контента
func RegisterStaticRoutes(mux *http.ServeMux, staticDir string) {
	if staticDir == "" {
		staticDir = "./static/"
	}
	
	// Статический контент для GUI (регистрируем последним)
	// Используем префикс, чтобы не перехватывать API запросы
	staticFS := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFS))
}

