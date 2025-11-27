package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"httpserver/normalization"
)

// handleMonitoringMetrics возвращает общие метрики мониторинга
func (s *Server) handleMonitoringMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем реальную статистику от нормализатора
	var statsCollector *normalization.StatsCollector
	var cacheStats normalization.CacheStats
	hasCacheStats := false

	if s.normalizer != nil && s.normalizer.GetAINormalizer() != nil {
		statsCollector = s.normalizer.GetAINormalizer().GetStatsCollector()
		cacheStats = s.normalizer.GetAINormalizer().GetCacheStats()
		hasCacheStats = true
	}

	// Получаем статистику качества из БД (используем db, так как записи сохраняются туда)
	qualityStatsMap, err := s.db.GetQualityStats()
	if err != nil {
		slog.Error("Error getting quality stats",
			"error", err,
		)
		qualityStatsMap = make(map[string]interface{})
	}

	// Извлекаем значения из карты БД (используем как fallback)
	dbTotalNormalized := int64(0)
	dbBasicNormalized := int64(0)
	dbAIEnhanced := int64(0)
	dbBenchmarkQuality := int64(0)
	dbAverageQualityScore := 0.0

	if total, ok := qualityStatsMap["total_items"].(int); ok {
		dbTotalNormalized = int64(total)
	}
	if avg, ok := qualityStatsMap["average_quality"].(float64); ok {
		dbAverageQualityScore = avg
	}
	if benchmark, ok := qualityStatsMap["benchmark_count"].(int); ok {
		dbBenchmarkQuality = int64(benchmark)
	}
	if byLevel, ok := qualityStatsMap["by_level"].(map[string]map[string]interface{}); ok {
		if basicStats, ok := byLevel["basic"]; ok {
			if count, ok := basicStats["count"].(int); ok {
				dbBasicNormalized = int64(count)
			}
		}
		if aiStats, ok := byLevel["ai_enhanced"]; ok {
			if count, ok := aiStats["count"].(int); ok {
				dbAIEnhanced = int64(count)
			}
		}
	}

	// Используем метрики из StatsCollector как основной источник, БД как fallback
	totalNormalized := dbTotalNormalized
	basicNormalized := dbBasicNormalized
	aiEnhanced := dbAIEnhanced
	benchmarkQuality := dbBenchmarkQuality
	averageQualityScore := dbAverageQualityScore

	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		// Используем метрики из StatsCollector если они доступны
		if perfMetrics.TotalNormalized > 0 {
			totalNormalized = perfMetrics.TotalNormalized
			basicNormalized = perfMetrics.BasicNormalized
			aiEnhanced = perfMetrics.AIEnhanced
			benchmarkQuality = perfMetrics.BenchmarkQuality
			if perfMetrics.AverageQualityScore > 0 {
				averageQualityScore = perfMetrics.AverageQualityScore
			} else if dbAverageQualityScore > 0 {
				averageQualityScore = dbAverageQualityScore
			}
		}
	}

	// Рассчитываем uptime
	uptime := time.Since(s.startTime).Seconds()

	// Рассчитываем throughput (за всё время работы)
	throughput := 0.0
	if uptime > 0 && totalNormalized > 0 {
		throughput = float64(totalNormalized) / uptime
	}

	// Формируем ответ
	summary := map[string]interface{}{
		"uptime_seconds":              uptime,
		"throughput_items_per_second": throughput,
		"ai": map[string]interface{}{
			"total_requests":     0,
			"successful":         0,
			"failed":             0,
			"success_rate":       0.0,
			"average_latency_ms": 0.0,
		},
		"cache": map[string]interface{}{
			"hits":            0,
			"misses":          0,
			"hit_rate":        0.0,
			"size":            0,
			"memory_usage_kb": 0.0,
		},
		"quality": map[string]interface{}{
			"total_normalized":      totalNormalized,
			"basic":                 basicNormalized,
			"ai_enhanced":           aiEnhanced,
			"benchmark":             benchmarkQuality,
			"average_quality_score": averageQualityScore,
		},
	}

	// Добавляем реальные AI метрики если доступны
	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		successRate := 0.0
		if perfMetrics.TotalAIRequests > 0 {
			successRate = float64(perfMetrics.SuccessfulAIRequest) / float64(perfMetrics.TotalAIRequests)
		}
		avgLatencyMs := float64(perfMetrics.AverageAILatency.Milliseconds())

		summary["ai"] = map[string]interface{}{
			"total_requests":     perfMetrics.TotalAIRequests,
			"successful":         perfMetrics.SuccessfulAIRequest,
			"failed":             perfMetrics.FailedAIRequests,
			"success_rate":       successRate,
			"average_latency_ms": avgLatencyMs,
		}
	}

	// Добавляем реальные cache метрики если доступны
	if hasCacheStats {
		summary["cache"] = map[string]interface{}{
			"hits":            cacheStats.Hits,
			"misses":          cacheStats.Misses,
			"hit_rate":        cacheStats.HitRate,
			"size":            cacheStats.Entries,
			"memory_usage_kb": float64(cacheStats.MemoryUsageB) / 1024.0,
		}
	}

	// Добавляем метрики Circuit Breaker
	summary["circuit_breaker"] = s.GetCircuitBreakerState()

	// Добавляем метрики Batch Processor
	summary["batch_processor"] = s.GetBatchProcessorStats()

	// Добавляем статус Checkpoint
	summary["checkpoint"] = s.GetCheckpointStatus()

	s.writeJSONResponse(w, r, summary, http.StatusOK)
}

// handleMonitoringCache возвращает статистику кеша
func (s *Server) handleMonitoringCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем реальную статистику кеша
	var cacheStats normalization.CacheStats
	if s.normalizer != nil && s.normalizer.GetAINormalizer() != nil {
		cacheStats = s.normalizer.GetAINormalizer().GetCacheStats()
	}

	response := map[string]interface{}{
		"hits":            cacheStats.Hits,
		"misses":          cacheStats.Misses,
		"hit_rate_pct":    cacheStats.HitRate * 100.0,
		"size":            cacheStats.Entries,
		"memory_usage_kb": float64(cacheStats.MemoryUsageB) / 1024.0,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleMonitoringAI возвращает статистику AI обработки
func (s *Server) handleMonitoringAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем реальную статистику AI
	var statsCollector *normalization.StatsCollector
	var cacheStats normalization.CacheStats

	if s.normalizer != nil && s.normalizer.GetAINormalizer() != nil {
		statsCollector = s.normalizer.GetAINormalizer().GetStatsCollector()
		cacheStats = s.normalizer.GetAINormalizer().GetCacheStats()
	}

	totalCalls := int64(0)
	errors := int64(0)
	avgLatencyMs := 0.0

	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		totalCalls = perfMetrics.TotalAIRequests
		errors = perfMetrics.FailedAIRequests
		avgLatencyMs = float64(perfMetrics.AverageAILatency.Milliseconds())
	}

	cacheHitRate := 0.0
	if cacheStats.Hits+cacheStats.Misses > 0 {
		cacheHitRate = float64(cacheStats.Hits) / float64(cacheStats.Hits+cacheStats.Misses) * 100.0
	}

	stats := map[string]interface{}{
		"total_calls":    totalCalls,
		"cache_hits":     cacheStats.Hits,
		"cache_misses":   cacheStats.Misses,
		"errors":         errors,
		"avg_latency_ms": avgLatencyMs,
		"cache_hit_rate": cacheHitRate,
	}

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleMonitoringHistory возвращает историю метрик за указанный период
func (s *Server) handleMonitoringHistory(w http.ResponseWriter, r *http.Request) {
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
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	// Получаем историю метрик из БД
	snapshots, err := s.db.GetMetricsHistory(fromTime, toTime, metricType, limit)
	if err != nil {
		slog.Error("Error getting metrics history",
			"error", err,
		)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get metrics history: %v", err), http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := map[string]interface{}{
		"count":     len(snapshots),
		"snapshots": snapshots,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleMonitoringEvents обрабатывает SSE соединение для real-time метрик мониторинга
func (s *Server) handleMonitoringEvents(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[Monitoring] Panic in handleMonitoringEvents",
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
	w.WriteHeader(http.StatusOK)

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

	// Heartbeat тикер
	// Heartbeat тикер (каждые 10 секунд для предотвращения таймаута)
	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-metricsTicker.C:
			// Собираем текущие метрики с обработкой ошибок
			func() {
				defer func() {
					if panicVal := recover(); panicVal != nil {
						slog.Error("[Monitoring] Panic in CollectMetricsSnapshot",
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

				snapshot := s.CollectMetricsSnapshot()
				if snapshot != nil {
					// Формируем JSON с метриками
					metricsJSON, err := json.Marshal(map[string]interface{}{
						"type":                  "metrics",
						"timestamp":             time.Now().Format(time.RFC3339),
						"uptime_seconds":        snapshot.UptimeSeconds,
						"throughput":            snapshot.Throughput,
						"ai_success_rate":       snapshot.AISuccessRate,
						"cache_hit_rate":        snapshot.CacheHitRate,
						"batch_queue_size":      snapshot.BatchQueueSize,
						"circuit_breaker_state": snapshot.CircuitBreakerState,
						"checkpoint_progress":   snapshot.CheckpointProgress,
					})

					if err != nil {
						slog.Error("[Monitoring] Error marshaling metrics",
							"error", err,
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

					if _, err := fmt.Fprintf(w, "data: %s\n\n", string(metricsJSON)); err != nil {
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
			slog.Info("[Monitoring] Client disconnected",
				"error", r.Context().Err(),
				"path", r.URL.Path,
			)
			return
		}
	}
}
