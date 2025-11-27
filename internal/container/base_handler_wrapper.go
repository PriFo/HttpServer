package container

import (
	"net/http"

	"httpserver/server/middleware"
)

// baseHandlerWrapper обертка для BaseHandler, чтобы избежать циклического импорта
type baseHandlerWrapper struct {
	writeJSONResponse func(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int)
	writeJSONError    func(w http.ResponseWriter, r *http.Request, message string, statusCode int)
	handleHTTPError   func(w http.ResponseWriter, r *http.Request, err error)
}

func (h *baseHandlerWrapper) WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	h.writeJSONResponse(w, r, data, statusCode)
}

func (h *baseHandlerWrapper) WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	h.writeJSONError(w, r, message, statusCode)
}

func (h *baseHandlerWrapper) HandleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	h.handleHTTPError(w, r, err)
}

// newBaseHandlerWrapper создает новую обертку для BaseHandler
func newBaseHandlerWrapper() *baseHandlerWrapper {
	return &baseHandlerWrapper{
		writeJSONResponse: middleware.WriteJSONResponse,
		writeJSONError:    middleware.WriteJSONError,
		handleHTTPError:   middleware.HandleHTTPError,
	}
}

