package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"httpserver/server/handlers"
	"httpserver/server/middleware"
)

// WorkerTraceHandler обработчик для трассировки воркеров
type WorkerTraceHandler struct {
	*handlers.BaseHandler
	logFunc func(entry interface{}) // server.LogEntry
}

// NewWorkerTraceHandler создает новый обработчик трассировки воркеров
func NewWorkerTraceHandler(
	baseHandler *handlers.BaseHandler,
	logFunc func(entry interface{}),
) *WorkerTraceHandler {
	return &WorkerTraceHandler{
		BaseHandler: baseHandler,
		logFunc:     logFunc,
	}
}

// SetLogFunc устанавливает функцию логирования
func (h *WorkerTraceHandler) SetLogFunc(logFunc func(entry interface{})) {
	h.logFunc = logFunc
}

// WorkerTraceStep представляет шаг выполнения воркера
type WorkerTraceStep struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"trace_id"`
	Step      string                 `json:"step"`
	StartTime int64                  `json:"start_time"` // Unix timestamp в миллисекундах
	EndTime   *int64                 `json:"end_time,omitempty"`
	Duration  *int64                 `json:"duration,omitempty"` // в миллисекундах
	Level     string                 `json:"level"`              // INFO, WARNING, ERROR
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HandleWorkerTraceStream обрабатывает SSE соединение для стриминга логов по trace_id
func (h *WorkerTraceHandler) HandleWorkerTraceStream(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[WorkerTrace] Panic in HandleWorkerTraceStream",
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

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем trace_id из query параметра или заголовка
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		traceID = r.Header.Get("X-Request-ID")
	}
	if traceID == "" {
		traceID = middleware.GetRequestID(r.Context())
	}

	if traceID == "" {
		h.WriteJSONError(w, r, "trace_id is required", http.StatusBadRequest)
		return
	}

	// Проверяем поддержку Flusher ДО установки заголовков
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.WriteJSONError(w, r, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	w.WriteHeader(http.StatusOK)

	// Отправляем начальное событие с обработкой ошибок
	connectedEvent := map[string]interface{}{
		"type":      "connected",
		"message":   fmt.Sprintf("Connected to worker trace stream for trace_id: %s", traceID),
		"trace_id":  traceID,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	eventJSON, err := json.Marshal(connectedEvent)
	if err != nil {
		slog.Error("[WorkerTrace] Error marshaling connected event",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", eventJSON); err != nil {
		slog.Error("[WorkerTrace] Error sending initial connection message",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	flusher.Flush()

	// Создаем канал для событий
	events := make(chan WorkerTraceStep, 100)
	defer close(events)

	// Запускаем горутину для симуляции получения логов
	// В реальном приложении здесь должна быть интеграция с системой логирования
	go h.simulateLogStream(r.Context(), traceID, events)

	// Heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case step, ok := <-events:
			if !ok {
				// Канал закрыт, отправляем завершающее событие
				finishedEvent := map[string]interface{}{
					"type":      "finished",
					"message":   "Trace stream finished",
					"trace_id":  traceID,
					"timestamp": time.Now().Format(time.RFC3339),
				}
				eventJSON, err := json.Marshal(finishedEvent)
				if err != nil {
					slog.Error("[WorkerTrace] Error marshaling finished event",
						"error", err,
						"path", r.URL.Path,
					)
					return
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", eventJSON); err != nil {
					slog.Error("[WorkerTrace] Error sending finished event",
						"error", err,
						"path", r.URL.Path,
					)
					return
				}
				flusher.Flush()
				return
			}

			// Отправляем шаг
			stepJSON, err := json.Marshal(step)
			if err != nil {
				slog.Error("[WorkerTrace] Error marshaling step",
					"error", err,
					"path", r.URL.Path,
				)
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", stepJSON); err != nil {
				slog.Error("[WorkerTrace] Error sending SSE event",
					"error", err,
					"path", r.URL.Path,
				)
				return
			}
			flusher.Flush()

		case <-ticker.C:
			// Heartbeat для поддержания соединения
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				slog.Error("[WorkerTrace] Error sending heartbeat",
					"error", err,
					"path", r.URL.Path,
				)
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			// Клиент отключился
			slog.Info("[WorkerTrace] Client disconnected",
				"error", r.Context().Err(),
				"path", r.URL.Path,
			)
			return
		}
	}
}

// simulateLogStream симулирует поток логов для заданного trace_id
// В реальном приложении здесь должна быть интеграция с реальной системой логирования
func (h *WorkerTraceHandler) simulateLogStream(ctx context.Context, traceID string, events chan<- WorkerTraceStep) {
	defer close(events)

	// Симулируем несколько шагов выполнения
	steps := []struct {
		step     string
		level    string
		message  string
		duration int64 // в миллисекундах
	}{
		{"initialization", "INFO", "Worker initialization started", 50},
		{"config_load", "INFO", "Loading worker configuration", 100},
		{"provider_check", "INFO", "Checking provider availability", 200},
		{"model_selection", "INFO", "Selecting model for processing", 150},
		{"processing", "INFO", "Processing started", 500},
		{"validation", "INFO", "Validating results", 100},
		{"completion", "INFO", "Worker completed successfully", 50},
	}

	baseTime := time.Now().UnixMilli()
	stepCounter := 0

	for i, stepInfo := range steps {
		select {
		case <-ctx.Done():
			return
		default:
		}

		startTime := baseTime + int64(stepCounter*100)
		endTime := startTime + stepInfo.duration

		step := WorkerTraceStep{
			ID:        fmt.Sprintf("%s-step-%d", traceID, i+1),
			TraceID:   traceID,
			Step:      stepInfo.step,
			StartTime: startTime,
			EndTime:   &endTime,
			Duration:  &stepInfo.duration,
			Level:     stepInfo.level,
			Message:   stepInfo.message,
			Metadata: map[string]interface{}{
				"step_number": i + 1,
				"total_steps": len(steps),
			},
		}

		// Небольшая задержка для симуляции реального времени
		time.Sleep(time.Duration(stepInfo.duration) * time.Millisecond)

		select {
		case events <- step:
		case <-ctx.Done():
			return
		}

		stepCounter++
	}

	// Небольшая задержка перед завершением
	time.Sleep(500 * time.Millisecond)
}
