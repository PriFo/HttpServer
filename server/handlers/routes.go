package handlers

import (
	"net/http"
)

// RouteGroup представляет группу маршрутов
type RouteGroup struct {
	Prefix     string
	Routes     []Route
	Middleware []func(http.Handler) http.Handler
}

// Route представляет один маршрут
type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []func(http.Handler) http.Handler
}

// RouteRegistry регистрирует маршруты
type RouteRegistry struct {
	groups []RouteGroup
}

// NewRouteRegistry создает новый реестр маршрутов
func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{
		groups: make([]RouteGroup, 0),
	}
}

// RegisterGroup регистрирует группу маршрутов
func (r *RouteRegistry) RegisterGroup(group RouteGroup) {
	r.groups = append(r.groups, group)
}

// RegisterRoutes регистрирует все маршруты в мультиплексор
func (r *RouteRegistry) RegisterRoutes(mux *http.ServeMux) {
	for _, group := range r.groups {
		for _, route := range group.Routes {
			var handler http.Handler = http.HandlerFunc(route.Handler)

			// Применяем middleware маршрута
			for _, mw := range route.Middlewares {
				handler = mw(handler)
			}

			// Применяем middleware группы
			for _, mw := range group.Middleware {
				handler = mw(handler)
			}

			// Регистрируем маршрут
			fullPath := group.Prefix + route.Path
			if route.Method != "" {
				handler = methodMiddleware(route.Method, route.Handler)
			}
			mux.Handle(fullPath, handler)
		}
	}
}

// methodMiddleware проверяет HTTP метод
func methodMiddleware(method string, handler http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	})
}
