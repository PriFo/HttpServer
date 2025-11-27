package handlers

import (
	"net/http"

	"httpserver/server/middleware"
)

// BaseHandler базовый обработчик с общими методами
type BaseHandler struct {
	writeJSONResponse func(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int)
	writeJSONError    func(w http.ResponseWriter, r *http.Request, message string, statusCode int)
	handleHTTPError  func(w http.ResponseWriter, r *http.Request, err error)
}

// NewBaseHandler создает новый базовый обработчик
func NewBaseHandler(
	writeJSONResponse func(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int),
	writeJSONError func(w http.ResponseWriter, r *http.Request, message string, statusCode int),
	handleHTTPError func(w http.ResponseWriter, r *http.Request, err error),
) *BaseHandler {
	return &BaseHandler{
		writeJSONResponse: writeJSONResponse,
		writeJSONError:    writeJSONError,
		handleHTTPError:   handleHTTPError,
	}
}

// NewBaseHandlerFromMiddleware создает новый базовый обработчик используя middleware функции
func NewBaseHandlerFromMiddleware() *BaseHandler {
	return &BaseHandler{
		writeJSONResponse: middleware.WriteJSONResponse,
		writeJSONError:    middleware.WriteJSONError,
		handleHTTPError:   middleware.HandleHTTPError,
	}
}

// WriteJSONResponse записывает JSON ответ
func (h *BaseHandler) WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	h.writeJSONResponse(w, r, data, statusCode)
}

// WriteJSONError записывает JSON ошибку
func (h *BaseHandler) WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	h.writeJSONError(w, r, message, statusCode)
}

// HandleHTTPError обрабатывает HTTP ошибку
func (h *BaseHandler) HandleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	h.handleHTTPError(w, r, err)
}

// HandleValidationError обрабатывает ошибку валидации
func (h *BaseHandler) HandleValidationError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err != nil {
		h.handleHTTPError(w, r, err)
		return true
	}
	return false
}

// HandleMethodNotAllowed обрабатывает неразрешенные HTTP методы
func (h *BaseHandler) HandleMethodNotAllowed(w http.ResponseWriter, r *http.Request, allowedMethods ...string) {
	w.Header().Set("Allow", joinMethods(allowedMethods))
	h.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
}

func joinMethods(methods []string) string {
	if len(methods) == 0 {
		return "GET, POST, PUT, DELETE, PATCH"
	}
	result := ""
	for i, method := range methods {
		if i > 0 {
			result += ", "
		}
		result += method
	}
	return result
}
