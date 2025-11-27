package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// handleSystemSummaryStream обрабатывает SSE соединение для real-time обновлений сканирования
// GET /api/system/summary/stream
func (s *Server) handleSystemSummaryStream(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[SystemScanner] Panic in handleSystemSummaryStream",
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

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	w.WriteHeader(http.StatusOK)

	// Отправляем начальное событие с обработкой ошибок
	if _, err := fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","message":"Connected to system summary stream"}`); err != nil {
		slog.Error("[SystemScanner] Error sending initial connection message",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	flusher.Flush()

	// Создаем тикер для периодической отправки обновлений
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Heartbeat тикер
	// Heartbeat тикер (каждые 10 секунд для предотвращения таймаута)
	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()

	// Отправляем начальное состояние
	s.sendSummaryUpdate(w, flusher)

	for {
		select {
		case <-ticker.C:
			// Отправляем обновление сводки
			s.sendSummaryUpdate(w, flusher)

		case <-heartbeatTicker.C:
			// Heartbeat для поддержания соединения
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				slog.Error("[SystemScanner] Error sending heartbeat",
					"error", err,
					"path", r.URL.Path,
				)
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			// Клиент отключился
			slog.Info("[SystemScanner] Client disconnected from system summary stream",
				"error", r.Context().Err(),
				"path", r.URL.Path,
			)
			return
		}
	}
}

// sendSummaryUpdate отправляет обновление сводки через SSE
func (s *Server) sendSummaryUpdate(w http.ResponseWriter, flusher http.Flusher) {
	// Получаем кэшированную сводку или создаем новую
	var summary *SystemSummary
	var fromCache bool

	if s.systemSummaryCache != nil {
		if cached, ok := s.systemSummaryCache.Get(); ok {
			if cachedSummary, ok := cached.(*SystemSummary); ok {
				summary = cachedSummary
				fromCache = true
			}
		}
	}

	// Если нет в кеше, получаем базовую информацию
	if summary == nil {
		summary = &SystemSummary{
			TotalDatabases: 0,
			TotalUploads:   0,
		}
	}

	// Формируем обновление
	update := map[string]interface{}{
		"type":        "summary_update",
		"timestamp":   time.Now().Format(time.RFC3339),
		"from_cache":  fromCache,
		"summary": map[string]interface{}{
			"total_databases":      summary.TotalDatabases,
			"total_uploads":       summary.TotalUploads,
			"completed_uploads":   summary.CompletedUploads,
			"failed_uploads":      summary.FailedUploads,
			"in_progress_uploads": summary.InProgressUploads,
			"last_activity":       summary.LastActivity.Format(time.RFC3339),
			"total_nomenclature":  summary.TotalNomenclature,
			"total_counterparties": summary.TotalCounterparties,
		},
	}

	if s.systemSummaryCache != nil {
		stats := s.systemSummaryCache.GetStats()
		update["cache_stats"] = map[string]interface{}{
			"hits":     stats.Hits,
			"misses":   stats.Misses,
			"hit_rate": stats.HitRate,
			"has_data": stats.HasData,
			"is_stale": stats.IsStale,
		}
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		slog.Error("[SystemScanner] Error marshaling summary update",
			"error", err,
		)
		return
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", string(updateJSON)); err != nil {
		slog.Error("[SystemScanner] Error sending summary update",
			"error", err,
		)
		return
	}
	flusher.Flush()
}

