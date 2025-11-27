package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"httpserver/database"
	apperrors "httpserver/server/errors"
	"httpserver/server/services"
)

// MonitoringData представляет данные мониторинга провайдеров
// Это алиас для типа из server пакета, чтобы избежать циклических зависимостей
type MonitoringData struct {
	Providers []ProviderMetrics `json:"providers"`
	System    SystemStats       `json:"system"`
}

// ProviderMetrics метрики для одного провайдера
type ProviderMetrics struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	ActiveChannels    int       `json:"active_channels"`
	CurrentRequests   int       `json:"current_requests"`
	TotalRequests     int64     `json:"total_requests"`
	SuccessfulRequests int64    `json:"successful_requests"`
	FailedRequests    int64     `json:"failed_requests"`
	AverageLatencyMs  float64   `json:"average_latency_ms"`
	LastRequestTime   string    `json:"last_request_time"`
	Status            string    `json:"status"`
	RequestsPerSecond float64   `json:"requests_per_second"`
}

// SystemStats общая статистика системы
type SystemStats struct {
	TotalProviders          int       `json:"total_providers"`
	ActiveProviders         int       `json:"active_providers"`
	TotalRequests           int64     `json:"total_requests"`
	TotalSuccessful         int64     `json:"total_successful"`
	TotalFailed             int64     `json:"total_failed"`
	SystemRequestsPerSecond float64   `json:"system_requests_per_second"`
	Timestamp               string    `json:"timestamp"`
}

// MonitoringHandler обработчик для мониторинга
type MonitoringHandler struct {
	*BaseHandler
	monitoringService *services.MonitoringService
	logFunc           func(entry interface{}) // server.LogEntry, но без прямого импорта
	// Функции для получения метрик от Server
	getCircuitBreakerState func() map[string]interface{}
	getBatchProcessorStats  func() map[string]interface{}
	getCheckpointStatus     func() map[string]interface{}
	collectMetricsSnapshot  func() *database.PerformanceMetricsSnapshot
	getMonitoringMetrics    func() MonitoringData // Функция для получения метрик провайдеров
}

// NewMonitoringHandler создает новый обработчик мониторинга
func NewMonitoringHandler(
	baseHandler *BaseHandler,
	monitoringService *services.MonitoringService,
	logFunc func(entry interface{}), // server.LogEntry, но без прямого импорта
	getCircuitBreakerState func() map[string]interface{},
	getBatchProcessorStats func() map[string]interface{},
	getCheckpointStatus func() map[string]interface{},
	collectMetricsSnapshot func() *database.PerformanceMetricsSnapshot,
	getMonitoringMetrics func() MonitoringData, // Функция для получения метрик провайдеров
) *MonitoringHandler {
	return &MonitoringHandler{
		BaseHandler:           baseHandler,
		monitoringService:      monitoringService,
		logFunc:                logFunc,
		getCircuitBreakerState: getCircuitBreakerState,
		getBatchProcessorStats: getBatchProcessorStats,
		getCheckpointStatus:    getCheckpointStatus,
		collectMetricsSnapshot: collectMetricsSnapshot,
		getMonitoringMetrics:   getMonitoringMetrics,
	}
}

// HandleMonitoringMetrics обрабатывает запрос общих метрик мониторинга
func (h *MonitoringHandler) HandleMonitoringMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics, err := h.monitoringService.GetMetrics(
		h.getCircuitBreakerState,
		h.getBatchProcessorStats,
		h.getCheckpointStatus,
	)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting monitoring metrics: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get metrics: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, metrics, http.StatusOK)
}

// HandleMonitoringCache обрабатывает запрос статистики кеша
func (h *MonitoringHandler) HandleMonitoringCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.monitoringService.GetCacheStats()
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting cache stats: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get cache stats: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleMonitoringAI обрабатывает запрос статистики AI обработки
func (h *MonitoringHandler) HandleMonitoringAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.monitoringService.GetAIStats()
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting AI stats: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get AI stats: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleMonitoringHistory обрабатывает запрос истории метрик
func (h *MonitoringHandler) HandleMonitoringHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим параметры запроса
	query := r.URL.Query()

	// Параметр from (начало периода)
	var fromTime *time.Time
	if fromStr := query.Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromTime = &t
		} else {
			// Попробуем другой формат
			if t, err := time.Parse("2006-01-02 15:04:05", fromStr); err == nil {
				fromTime = &t
			}
		}
	}

	// Параметр to (конец периода)
	var toTime *time.Time
	if toStr := query.Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toTime = &t
		} else {
			if t, err := time.Parse("2006-01-02 15:04:05", toStr); err == nil {
				toTime = &t
			}
		}
	}

	// Параметр metricType (фильтр по типу)
	metricType := query.Get("metric_type")

	// Параметр limit (максимальное количество записей)
	limit, err := ValidateIntParam(r, "limit", 100, 1, 1000)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid limit parameter: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем историю метрик
	snapshots, err := h.monitoringService.GetMetricsHistory(fromTime, toTime, metricType, limit)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting metrics history: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get metrics history: %v", err), http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := map[string]interface{}{
		"count":     len(snapshots),
		"snapshots": snapshots,
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleMonitoringEvents обрабатывает SSE соединение для real-time метрик мониторинга
func (h *MonitoringHandler) HandleMonitoringEvents(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[Monitoring] Panic in HandleMonitoringEvents",
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
	if _, err := fmt.Fprintf(w, "data: %s\n\n", "{\"type\":\"connected\",\"message\":\"Connected to monitoring events\"}"); err != nil {
		slog.Error("[Monitoring] Error sending initial connection message",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	flusher.Flush()

	// Создаем тикер для периодической отправки метрик
	metricsTicker := time.NewTicker(5 * time.Second)
	defer metricsTicker.Stop()

	// Heartbeat тикер (каждые 10 секунд для предотвращения таймаута)
	// WriteTimeout на сервере 60 секунд, heartbeat каждые 10 секунд гарантирует активность
	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-metricsTicker.C:
			// Собираем текущие метрики с обработкой паники
			func() {
				defer func() {
					if panicVal := recover(); panicVal != nil {
						slog.Error("[Monitoring] Panic in HandleMonitoringEvents",
							"panic", panicVal,
							"stack", string(debug.Stack()),
							"path", r.URL.Path,
						)
						errorMsg := fmt.Sprintf(`{"type":"error","error":"internal error retrieving metrics","details":"%v"}`, panicVal)
						if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", errorMsg); writeErr != nil {
							slog.Error("[Monitoring] Error sending panic error message",
								"error", writeErr,
								"path", r.URL.Path,
							)
							return
						}
						flusher.Flush()
					}
				}()

				var metricsJSON []byte
				var err error
				
				if h.collectMetricsSnapshot != nil {
					snapshot := h.collectMetricsSnapshot()
					if snapshot != nil {
						// Преобразуем snapshot в map для JSON
						snapshotJSON, err := json.Marshal(snapshot)
						var snapshotMap map[string]interface{}
						if err == nil {
							json.Unmarshal(snapshotJSON, &snapshotMap)
						}

						// Формируем JSON с метриками
						metricsData := map[string]interface{}{
							"type":      "metrics",
							"timestamp": time.Now().Format(time.RFC3339),
						}
						
						// Добавляем данные из snapshot
						if snapshotMap != nil {
							if uptime, ok := snapshotMap["uptime_seconds"].(int); ok {
								metricsData["uptime_seconds"] = uptime
							}
							if throughput, ok := snapshotMap["throughput"].(float64); ok {
								metricsData["throughput"] = throughput
							}
							if aiSuccessRate, ok := snapshotMap["ai_success_rate"].(float64); ok {
								metricsData["ai_success_rate"] = aiSuccessRate
							}
							if cacheHitRate, ok := snapshotMap["cache_hit_rate"].(float64); ok {
								metricsData["cache_hit_rate"] = cacheHitRate
							}
							if batchQueueSize, ok := snapshotMap["batch_queue_size"].(int); ok {
								metricsData["batch_queue_size"] = batchQueueSize
							}
							if circuitBreakerState, ok := snapshotMap["circuit_breaker_state"].(string); ok {
								metricsData["circuit_breaker_state"] = circuitBreakerState
							}
							if checkpointProgress, ok := snapshotMap["checkpoint_progress"].(float64); ok {
								metricsData["checkpoint_progress"] = checkpointProgress
							}
						}

						metricsJSON, err = json.Marshal(metricsData)
						if err != nil {
							slog.Error("[Monitoring] Error marshaling metrics",
								"error", err,
								"data", metricsData,
								"path", r.URL.Path,
							)
							if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", `{"type":"error","error":"failed to marshal metrics"}`); writeErr != nil {
								slog.Error("[Monitoring] Error sending marshal error message",
									"error", writeErr,
									"path", r.URL.Path,
								)
								return
							}
							flusher.Flush()
							return
						}
					}
				}

				if metricsJSON != nil {
					if _, err = fmt.Fprintf(w, "data: %s\n\n", string(metricsJSON)); err != nil {
						slog.Error("[Monitoring] Error sending SSE metrics",
							"error", err,
							"path", r.URL.Path,
						)
						return
					}
					flusher.Flush()
				}
			}()

		case <-heartbeatTicker.C:
			// Отправляем heartbeat для поддержания соединения
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				slog.Error("[Monitoring] Error sending heartbeat",
					"error", err,
					"path", r.URL.Path,
				)
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			// Клиент отключился
			slog.Info("[Monitoring] SSE client disconnected",
				"error", r.Context().Err(),
				"path", r.URL.Path,
			)
			return
		}
	}
}

// HandleMonitoringProvidersStream обрабатывает SSE соединение для провайдеров
func (h *MonitoringHandler) HandleMonitoringProvidersStream(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[Monitoring] Panic in HandleMonitoringProvidersStream",
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

	// Проверяем, что функция получения метрик доступна ДО установки заголовков
	if h.getMonitoringMetrics == nil {
		slog.Error("[Monitoring] getMonitoringMetrics function is nil",
			"path", r.URL.Path,
		)
		http.Error(w, "Monitoring metrics function not available", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	w.WriteHeader(http.StatusOK)

	// Отправляем начальное событие подключения с обработкой ошибок
	if _, err := fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","message":"Connected to providers monitoring stream"}`); err != nil {
		slog.Error("[Monitoring] Error sending initial connection message",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	flusher.Flush()

	// Создаем тикер для периодической отправки метрик (каждую секунду)
	metricsTicker := time.NewTicker(1 * time.Second)
	defer metricsTicker.Stop()

	// Heartbeat тикер (каждые 10 секунд для предотвращения таймаута)
	// WriteTimeout на сервере 60 секунд, heartbeat каждые 10 секунд гарантирует активность
	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()

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

		case <-metricsTicker.C:
			// Получаем метрики провайдеров с обработкой паники
			func() {
				defer func() {
					if panicVal := recover(); panicVal != nil {
						slog.Error("[Monitoring] Panic in getMonitoringMetrics",
							"panic", panicVal,
							"stack", string(debug.Stack()),
							"path", r.URL.Path,
						)
						errorMsg := fmt.Sprintf(`{"type":"error","error":"internal error retrieving metrics","details":"%v"}`, panicVal)
						if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", errorMsg); writeErr != nil {
							slog.Error("[Monitoring] Error sending panic error message",
								"error", writeErr,
								"path", r.URL.Path,
							)
							return
						}
						flusher.Flush()
					}
				}()

				// Функция уже проверена в начале, но проверяем еще раз для безопасности
				if h.getMonitoringMetrics == nil {
					slog.Warn("[Monitoring] getMonitoringMetrics function is nil during stream",
						"path", r.URL.Path,
					)
					if _, err := fmt.Fprintf(w, "data: %s\n\n", `{"type":"error","error":"monitoring metrics function not available"}`); err != nil {
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
				var monitoringData MonitoringData
				var metricsError error
				func() {
					defer func() {
						if panicVal := recover(); panicVal != nil {
							slog.Error("[Monitoring] Panic in getMonitoringMetrics call",
								"panic", panicVal,
								"stack", string(debug.Stack()),
								"path", r.URL.Path,
							)
							// Устанавливаем пустые метрики при панике
							monitoringData = MonitoringData{
								Providers: []ProviderMetrics{},
								System:    SystemStats{
									Timestamp: time.Now().Format(time.RFC3339),
								},
							}
							metricsError = apperrors.NewInternalError("паника при получении метрик", fmt.Errorf("panic: %v", panicVal))
						}
					}()
					monitoringData = h.getMonitoringMetrics()
				}()
				
				// Если была ошибка при получении метрик, отправляем сообщение об ошибке
				if metricsError != nil {
					errorMsg := fmt.Sprintf(`{"type":"error","error":"%s"}`, metricsError.Error())
					if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", errorMsg); writeErr != nil {
						slog.Error("[Monitoring] Error sending error message",
							"error", writeErr,
							"path", r.URL.Path,
						)
						return
					}
					flusher.Flush()
					return
				}
				
				// Сериализуем в JSON
				jsonData, err := json.Marshal(monitoringData)
				if err != nil {
					slog.Error("[Monitoring] Error marshaling provider metrics",
						"error", err,
						"data", monitoringData,
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

		case <-heartbeatTicker.C:
			// Отправляем heartbeat для поддержания соединения
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				slog.Error("[Monitoring] Error sending heartbeat",
					"error", err,
					"path", r.URL.Path,
				)
				return
			}
			flusher.Flush()
		}
	}
}

// HandleMonitoringProviders обрабатывает запросы к /api/monitoring/providers
func (h *MonitoringHandler) HandleMonitoringProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем метрики провайдеров
	if h.getMonitoringMetrics == nil {
		h.WriteJSONResponse(w, r, MonitoringData{
			Providers: []ProviderMetrics{},
			System:    SystemStats{},
		}, http.StatusOK)
		return
	}

	// Безопасно получаем метрики с обработкой паники
	var monitoringData MonitoringData
	func() {
		defer func() {
			if panicVal := recover(); panicVal != nil {
				slog.Error("[Monitoring] Panic in getMonitoringMetrics",
					"panic", panicVal,
					"path", r.URL.Path,
				)
				monitoringData = MonitoringData{
					Providers: []ProviderMetrics{},
					System:    SystemStats{},
				}
			}
		}()
		monitoringData = h.getMonitoringMetrics()
	}()

	h.WriteJSONResponse(w, r, monitoringData, http.StatusOK)
}

