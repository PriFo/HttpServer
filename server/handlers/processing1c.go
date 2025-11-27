package handlers

import (
	"fmt"
	"net/http"

	"httpserver/server/services"
)

// Processing1CHandler обработчик для работы с обработками 1С
type Processing1CHandler struct {
	processing1CService *services.Processing1CService
	baseHandler         *BaseHandler
}

// NewProcessing1CHandler создает новый обработчик для работы с обработками 1С
func NewProcessing1CHandler(
	processing1CService *services.Processing1CService,
	baseHandler *BaseHandler,
) *Processing1CHandler {
	return &Processing1CHandler{
		processing1CService: processing1CService,
		baseHandler:         baseHandler,
	}
}

// HandleGenerateProcessingXML обрабатывает запросы к /api/1c/processing/xml
func (h *Processing1CHandler) HandleGenerateProcessingXML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	xmlContent, err := h.processing1CService.GenerateProcessingXML()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось сгенерировать XML: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=processing.xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xmlContent))
}

