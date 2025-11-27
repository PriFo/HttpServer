package handlers

import (
	"net/http"
	"strconv"

	"httpserver/server/middleware"
)

// ErrorMetricsHandler обработчик для получения метрик ошибок
type ErrorMetricsHandler struct {
	baseHandler *BaseHandler
}

// NewErrorMetricsHandler создает новый обработчик метрик ошибок
func NewErrorMetricsHandler(baseHandler *BaseHandler) *ErrorMetricsHandler {
	return &ErrorMetricsHandler{
		baseHandler: baseHandler,
	}
}

// GetErrorMetrics возвращает метрики ошибок
func (h *ErrorMetricsHandler) GetErrorMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	metrics := middleware.GetErrorMetrics()
	metricsData := metrics.GetMetrics()

	h.baseHandler.WriteJSONResponse(w, r, metricsData, http.StatusOK)
}

// GetErrorsByType возвращает ошибки по типу
func (h *ErrorMetricsHandler) GetErrorsByType(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	metrics := middleware.GetErrorMetrics()
	errorsByType := metrics.GetErrorsByType()

	h.baseHandler.WriteJSONResponse(w, r, errorsByType, http.StatusOK)
}

// GetErrorsByCode возвращает ошибки по HTTP коду
func (h *ErrorMetricsHandler) GetErrorsByCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	metrics := middleware.GetErrorMetrics()
	errorsByCode := metrics.GetErrorsByCode()

	h.baseHandler.WriteJSONResponse(w, r, errorsByCode, http.StatusOK)
}

// GetErrorsByEndpoint возвращает ошибки по эндпоинту
func (h *ErrorMetricsHandler) GetErrorsByEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	metrics := middleware.GetErrorMetrics()
	errorsByEndpoint := metrics.GetErrorsByEndpoint()

	h.baseHandler.WriteJSONResponse(w, r, errorsByEndpoint, http.StatusOK)
}

// GetLastErrors возвращает последние ошибки
func (h *ErrorMetricsHandler) GetLastErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	limit := 50 // По умолчанию
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := parseIntQueryParam(limitStr, "limit"); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	metrics := middleware.GetErrorMetrics()
	lastErrors := metrics.GetLastErrors(limit)

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"errors": lastErrors,
		"count":  len(lastErrors),
		"limit":  limit,
	}, http.StatusOK)
}

// ResetErrorMetrics сбрасывает метрики ошибок
func (h *ErrorMetricsHandler) ResetErrorMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	metrics := middleware.GetErrorMetrics()
	metrics.Reset()

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Метрики ошибок сброшены",
	}, http.StatusOK)
}

// parseIntQueryParam парсит целочисленный параметр запроса
func parseIntQueryParam(value, paramName string) (int, error) {
	// Простая реализация парсинга int
	// Используем strconv для парсинга
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

