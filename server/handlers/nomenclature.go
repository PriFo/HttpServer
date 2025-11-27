package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"httpserver/server/services"
)

// NomenclatureHandler обработчик для работы с номенклатурой
type NomenclatureHandler struct {
	nomenclatureService *services.NomenclatureService
	baseHandler         *BaseHandler
}

// NewNomenclatureHandler создает новый обработчик для работы с номенклатурой
func NewNomenclatureHandler(
	nomenclatureService *services.NomenclatureService,
	baseHandler *BaseHandler,
) *NomenclatureHandler {
	return &NomenclatureHandler{
		nomenclatureService: nomenclatureService,
		baseHandler:         baseHandler,
	}
}

// HandleStartProcessing обрабатывает запросы к /api/nomenclature/process
func (h *NomenclatureHandler) HandleStartProcessing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	err := h.nomenclatureService.StartProcessing()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось запустить обработку номенклатуры: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"status":  "processing_started",
		"message": "Обработка номенклатуры запущена",
	}, http.StatusOK)
}

// HandleGetStatus обрабатывает запросы к /api/nomenclature/status
func (h *NomenclatureHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	status, err := h.nomenclatureService.GetStatus()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить статус: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, status, http.StatusOK)
}

// HandleGetRecentRecords обрабатывает запросы к /api/nomenclature/recent
func (h *NomenclatureHandler) HandleGetRecentRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Значение по умолчанию
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 50
		}
	}

	records, err := h.nomenclatureService.GetRecentRecords(limit)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить недавние записи: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"records": records,
		"total":   len(records),
	}, http.StatusOK)
}

// HandleGetPendingRecords обрабатывает запросы к /api/nomenclature/pending
func (h *NomenclatureHandler) HandleGetPendingRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Значение по умолчанию
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 50
		}
	}

	records, err := h.nomenclatureService.GetPendingRecords(limit)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить ожидающие записи: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"records": records,
		"total":   len(records),
	}, http.StatusOK)
}

