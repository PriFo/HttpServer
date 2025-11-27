package common

import (
	"net/http"

	"httpserver/server/middleware"
)

// BaseHandlerInterface интерфейс для базового обработчика
// Используется для разрыва циклических зависимостей
type BaseHandlerInterface interface {
	WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int)
	WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int)
	HandleHTTPError(w http.ResponseWriter, r *http.Request, err error)
}

// BaseHandlerImpl реализация BaseHandlerInterface через middleware
// Может использоваться всеми handlers для единообразия
type BaseHandlerImpl struct{}

// NewBaseHandlerImpl создает новую реализацию BaseHandlerInterface
func NewBaseHandlerImpl() *BaseHandlerImpl {
	return &BaseHandlerImpl{}
}

func (h *BaseHandlerImpl) WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	middleware.WriteJSONResponse(w, r, data, statusCode)
}

func (h *BaseHandlerImpl) WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	middleware.WriteJSONError(w, r, message, statusCode)
}

func (h *BaseHandlerImpl) HandleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	middleware.HandleHTTPError(w, r, err)
}
