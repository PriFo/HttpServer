package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	monitoringinfra "httpserver/internal/infrastructure/monitoring"
)

// handleMonitoringProvidersStream обрабатывает SSE поток метрик провайдеров
func (s *Server) handleMonitoringProvidersStream(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[Monitoring] Panic in handleMonitoringProvidersStream",
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

	// Проверяем поддержку Flusher ДО установки заголовков
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Проверяем, что monitoringManager доступен ДО установки заголовков
	if s.monitoringManager == nil {
		slog.Error("[Monitoring] monitoringManager is nil",
			"path", r.URL.Path,
		)
		http.Error(w, "Monitoring manager not initialized", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	w.WriteHeader(http.StatusOK)

	// Отправляем начальное сообщение с обработкой ошибок
	if _, err := fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","message":"Connected to providers monitoring stream"}`); err != nil {
		slog.Error("[Monitoring] Error sending initial connection message",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	flusher.Flush()

	// Создаем тикер для периодической отправки метрик
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Канал для отслеживания закрытия соединения
	clientGone := r.Context().Done()

	for {
		select {
		case <-clientGone:
			// Клиент отключился
			slog.Info("[Monitoring] Client disconnected from providers stream",
				"error", r.Context().Err(),
				"path", r.URL.Path,
			)
			return
		case <-ticker.C:
			// Собираем метрики с обработкой паники
			func() {
				defer func() {
					if panicVal := recover(); panicVal != nil {
						slog.Error("[Monitoring] Panic in GetAllMetrics",
							"panic", panicVal,
							"stack", string(debug.Stack()),
							"path", r.URL.Path,
						)
						errorMsg := fmt.Sprintf(`{"type":"error","error":"internal error retrieving metrics","details":"%v"}`, panicVal)
						if _, err := fmt.Fprintf(w, "data: %s\n\n", errorMsg); err != nil {
							slog.Error("[Monitoring] Error sending panic error message",
								"error", err,
								"path", r.URL.Path,
							)
							return
						}
						flusher.Flush()
					}
				}()

				// Проверяем еще раз (на случай, если manager стал nil после установки заголовков)
				if s.monitoringManager == nil {
					slog.Error("[Monitoring] monitoringManager is nil during stream",
						"path", r.URL.Path,
					)
					if _, err := fmt.Fprintf(w, "data: %s\n\n", `{"type":"error","error":"monitoring manager not initialized"}`); err != nil {
						slog.Error("[Monitoring] Error sending error message",
							"error", err,
							"path", r.URL.Path,
						)
						return
					}
					flusher.Flush()
					return
				}

				// Безопасно получаем метрики
				var monitoringData monitoringinfra.MonitoringData
				func() {
					defer func() {
						if panicVal := recover(); panicVal != nil {
							slog.Error("[Monitoring] Panic in GetAllMetrics call",
								"panic", panicVal,
								"path", r.URL.Path,
							)
							monitoringData = monitoringinfra.MonitoringData{
								Providers: []monitoringinfra.ProviderMetrics{},
								System: monitoringinfra.SystemStats{
									Timestamp: time.Now(),
								},
							}
						}
					}()
					monitoringData = s.monitoringManager.GetAllMetrics()
				}()

				// Сериализуем в JSON
				jsonData, err := json.Marshal(monitoringData)
				if err != nil {
					slog.Error("[Monitoring] Error marshaling metrics",
						"error", err,
						"path", r.URL.Path,
					)
					if _, err := fmt.Fprintf(w, "data: %s\n\n", `{"type":"error","error":"failed to marshal metrics"}`); err != nil {
						slog.Error("[Monitoring] Error sending marshal error message",
							"error", err,
							"path", r.URL.Path,
						)
						return
					}
					flusher.Flush()
					return
				}

				// Отправляем данные в формате SSE
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(jsonData)); err != nil {
					slog.Error("[Monitoring] Error sending SSE data",
						"error", err,
						"path", r.URL.Path,
					)
					return
				}
				flusher.Flush()
			}()
		}
	}
}

// handleMonitoringProviders возвращает текущие метрики провайдеров (одноразовый запрос)
func (s *Server) handleMonitoringProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.monitoringManager == nil {
		s.writeJSONError(w, r, "Monitoring manager not initialized", http.StatusInternalServerError)
		return
	}

	monitoringData := s.monitoringManager.GetAllMetrics()
	s.writeJSONResponse(w, r, monitoringData, http.StatusOK)
}
