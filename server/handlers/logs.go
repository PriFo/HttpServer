package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// LogsHandler обработчик для логирования ошибок клиента
type LogsHandler struct {
	baseHandler *BaseHandler
}

// NewLogsHandler создает новый обработчик для логов
func NewLogsHandler(baseHandler *BaseHandler) *LogsHandler {
	return &LogsHandler{
		baseHandler: baseHandler,
	}
}

// ClientErrorRequest описывает структуру запроса на логирование ошибки клиента
type ClientErrorRequest struct {
	Error     interface{} `json:"error"`
	Stack     string      `json:"stack,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	URL       string      `json:"url,omitempty"`
	Context   interface{} `json:"context,omitempty"`
	UserAgent string      `json:"user_agent,omitempty"`
}

// HandleClientError обрабатывает POST запросы для логирования ошибок клиента
// @Summary Логирование ошибок клиента
// @Description Принимает ошибки от frontend и логирует их на сервере
// @Tags logs
// @Accept json
// @Produce json
// @Param error body ClientErrorRequest true "Данные об ошибке"
// @Success 200 {object} map[string]interface{} "Успешное логирование"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/logs/client-error [post]
func (h *LogsHandler) HandleClientError(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req ClientErrorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("[ClientError] Failed to decode request body", "error", err)
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Логируем ошибку клиента
	slog.Error("[ClientError] Frontend error",
		"error", req.Error,
		"stack", req.Stack,
		"timestamp", req.Timestamp,
		"url", req.URL,
		"context", req.Context,
		"user_agent", req.UserAgent,
		"remote_addr", r.RemoteAddr,
		"request_id", r.Header.Get("X-Request-ID"),
	)

	// Возвращаем успешный ответ
	response := map[string]interface{}{
		"success":   true,
		"message":   "Error logged successfully",
		"logged_at": time.Now().Format(time.RFC3339),
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

