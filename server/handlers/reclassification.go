package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"httpserver/server/services"
)

// ReclassificationHandler обработчик для переклассификации
type ReclassificationHandler struct {
	reclassificationService *services.ReclassificationService
	baseHandler            *BaseHandler
}

// NewReclassificationHandler создает новый обработчик для переклассификации
func NewReclassificationHandler(
	reclassificationService *services.ReclassificationService,
	baseHandler *BaseHandler,
) *ReclassificationHandler {
	return &ReclassificationHandler{
		reclassificationService: reclassificationService,
		baseHandler:             baseHandler,
	}
}

// HandleStart обрабатывает запросы к /api/reclassification/start
func (h *ReclassificationHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req services.ReclassificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Ошибка парсинга запроса: %v", err), http.StatusBadRequest)
		return
	}

	err := h.reclassificationService.Start(req)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, err.Error(), http.StatusConflict)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success":       true,
		"message":       "Переклассификация запущена",
		"classifier_id": req.ClassifierID,
		"strategy_id":   req.StrategyID,
		"limit":         req.Limit,
	}, http.StatusOK)
}

// HandleEvents обрабатывает SSE соединение для событий переклассификации
func (h *ReclassificationHandler) HandleEvents(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[Reclassification] Panic in HandleEvents",
				"panic", panicVal,
				"stack", string(debug.Stack()),
				"path", r.URL.Path,
			)
			// Если заголовки еще не установлены, отправляем обычный HTTP ответ
			if w.Header().Get("Content-Type") != "text/event-stream" {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}
	}()

	// Проверяем поддержку Flusher ДО установки заголовков
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Отправляем начальное событие с обработкой ошибок
	if _, err := fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","message":"Connected to reclassification events"}`); err != nil {
		slog.Error("[Reclassification] Error sending initial connection message",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	flusher.Flush()

	events := h.reclassificationService.GetEvents()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-events:
			// Обработка события с защитой от паники
			func() {
				defer func() {
					if panicVal := recover(); panicVal != nil {
						slog.Error("[Reclassification] Panic in HandleEvents",
							"panic", panicVal,
							"stack", string(debug.Stack()),
							"path", r.URL.Path,
						)
						errorMsg := fmt.Sprintf(`{"error":"internal error processing event","details":"%v"}`, panicVal)
						fmt.Fprintf(w, "data: %s\n\n", errorMsg)
						flusher.Flush()
					}
				}()

				eventJSON := fmt.Sprintf(`{"type":"log","message":%q,"timestamp":%q}`,
					event, time.Now().Format(time.RFC3339))
				if _, err := fmt.Fprintf(w, "data: %s\n\n", eventJSON); err != nil {
					slog.Error("[Reclassification] Error sending SSE event",
						"error", err,
						"path", r.URL.Path,
					)
					return
				}
				flusher.Flush()
			}()
		case <-ticker.C:
			// Heartbeat
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				slog.Error("[Reclassification] Error sending heartbeat",
					"error", err,
					"path", r.URL.Path,
				)
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			slog.Info("[Reclassification] Client disconnected",
				"error", r.Context().Err(),
				"path", r.URL.Path,
			)
			return
		}
	}
}

// HandleStatus обрабатывает запросы к /api/reclassification/status
func (h *ReclassificationHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	status := h.reclassificationService.GetStatus()
	h.baseHandler.WriteJSONResponse(w, r, status, http.StatusOK)
}

// HandleStop обрабатывает запросы к /api/reclassification/stop
func (h *ReclassificationHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	wasRunning := h.reclassificationService.Stop()
	if !wasRunning {
		h.baseHandler.WriteJSONError(w, r, "Переклассификация не выполняется", http.StatusBadRequest)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Переклассификация остановлена",
	}, http.StatusOK)
}

