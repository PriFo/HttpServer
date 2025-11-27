package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"httpserver/internal/infrastructure/cache"
)

// Legacy system handlers - перемещены из server.go для рефакторинга
// TODO: Заменить на новые handlers из internal/api/handlers/

// handleStats возвращает общую статистику сервера
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleHealth обрабатывает проверку здоровья сервера
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.healthChecker != nil {
		s.healthChecker.HTTPHandler()(w, r)
	} else {
		// Fallback для обратной совместимости
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	}
}

// handlePerformanceMetrics возвращает метрики производительности из MetricsCollector
func (s *Server) handlePerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.metricsCollector == nil {
		http.Error(w, "Metrics collector not initialized", http.StatusServiceUnavailable)
		return
	}

	metrics := s.metricsCollector.GetMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleSystemSummary обрабатывает запрос сводной информации по всем базам данных системы
func (s *Server) handleSystemSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем пути к БД из конфигурации
	serviceDBPath := "service.db"
	mainDBPath := s.currentDBPath

	if s.config != nil {
		serviceDBPath = s.config.ServiceDatabasePath
		if s.config.DatabasePath != "" {
			mainDBPath = s.config.DatabasePath
		}
	}

	// Если путь к основной БД не установлен, используем путь по умолчанию
	if mainDBPath == "" {
		mainDBPath = "data.db"
	}

	// Проверяем кэш
	if s.systemSummaryCache != nil {
		if cached, ok := s.systemSummaryCache.Get(); ok {
			if cachedSummary, ok := cached.(*SystemSummary); ok {
				LogInfo(r.Context(), "Возвращена кэшированная системная сводка",
					"total_uploads", cachedSummary.TotalUploads,
					"total_databases", cachedSummary.TotalDatabases)
				s.writeJSONResponse(w, r, cachedSummary, http.StatusOK)
				return
			}
		}
	}

	// Создаем контекст с таймаутом для операции сканирования
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Вызываем функцию сканирования
	summary, err := ScanAndSummarizeAllDatabases(ctx, serviceDBPath, mainDBPath)
	if err != nil {
		LogError(r.Context(), err, "Failed to scan and summarize databases", "service_db", serviceDBPath, "main_db", mainDBPath)

		// В случае ошибки пытаемся вернуть кэшированные данные (если есть)
		if s.systemSummaryCache != nil {
			if cached, ok := s.systemSummaryCache.Get(); ok {
				if cachedSummary, ok := cached.(*SystemSummary); ok {
					LogWarn(r.Context(), "Ошибка сканирования, возвращена устаревшая кэшированная сводка")
					s.writeJSONResponse(w, r, cachedSummary, http.StatusOK)
					return
				}
			}
		}

		s.handleHTTPError(w, r, NewInternalError("не удалось просканировать системы", err))
		return
	}

	// Сохраняем в кэш
	if s.systemSummaryCache != nil {
		s.systemSummaryCache.Set(summary)
	}

	// Сохраняем в историю
	if s.scanHistoryManager != nil {
		scanDuration := ""
		if summary.ScanDuration != nil {
			scanDuration = *summary.ScanDuration
		}
		if err := s.scanHistoryManager.SaveScan(ctx, summary, scanDuration, nil); err != nil {
			LogWarn(r.Context(), "Не удалось сохранить сканирование в историю", "error", err)
		}
	}

	// Проверяем формат экспорта
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "text/csv") || strings.Contains(acceptHeader, "application/csv") {
			format = "csv"
		}
	}

	// Применяем фильтры из query параметров
	filter := ParseSystemSummaryFilterFromRequest(r)
	if len(filter.Status) > 0 || filter.Search != "" || filter.CreatedAfter != nil || filter.CreatedBefore != nil || filter.SortBy != "" || filter.Limit > 0 {
		summary = ApplyFilters(summary, filter)
		LogInfo(r.Context(), "Применены фильтры к системной сводке",
			"status", filter.Status,
			"search", filter.Search,
			"sort_by", filter.SortBy,
			"limit", filter.Limit)
	}

	LogInfo(r.Context(), "Системная сводка успешно сформирована",
		"total_uploads", summary.TotalUploads,
		"total_databases", summary.TotalDatabases,
		"scan_duration", summary.ScanDuration)

	// Экспорт в CSV или JSON
	if format == "csv" {
		if err := ExportSystemSummaryToCSV(w, summary); err != nil {
			LogError(r.Context(), err, "Failed to export system summary to CSV")
			s.handleHTTPError(w, r, NewInternalError("не удалось экспортировать данные в CSV", err))
			return
		}
		return
	}

	s.writeJSONResponse(w, r, summary, http.StatusOK)
}

// handleSystemSummaryCacheStats возвращает статистику кеша системной сводки
// GET /api/system/summary/cache/stats
func (s *Server) handleSystemSummaryCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.systemSummaryCache == nil {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"error": "Кеш не инициализирован",
		}, http.StatusServiceUnavailable)
		return
	}

	stats := s.systemSummaryCache.GetStats()
	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleSystemSummaryCacheInvalidate инвалидирует кеш системной сводки
// POST /api/system/summary/cache/invalidate
func (s *Server) handleSystemSummaryCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.systemSummaryCache == nil {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"error": "Кеш не инициализирован",
		}, http.StatusServiceUnavailable)
		return
	}

	s.systemSummaryCache.Invalidate()
	LogInfo(r.Context(), "Кеш системной сводки инвалидирован")

	s.writeJSONResponse(w, r, map[string]interface{}{
		"message": "Кеш успешно инвалидирован",
		"stats":   s.systemSummaryCache.GetStats(),
	}, http.StatusOK)
}

// handleSystemSummaryCacheClear полностью очищает кеш системной сводки
// POST /api/system/summary/cache/clear
func (s *Server) handleSystemSummaryCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.systemSummaryCache == nil {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"error": "Кеш не инициализирован",
		}, http.StatusServiceUnavailable)
		return
	}

	s.systemSummaryCache.Clear()
	LogInfo(r.Context(), "Кеш системной сводки очищен")

	s.writeJSONResponse(w, r, map[string]interface{}{
		"message": "Кеш успешно очищен",
	}, http.StatusOK)
}

// handleSystemSummaryExport экспортирует системную сводку в различных форматах
// GET /api/system/summary/export?format=csv|json
func (s *Server) handleSystemSummaryExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем пути к БД из конфигурации
	serviceDBPath := "service.db"
	mainDBPath := s.currentDBPath

	if s.config != nil {
		serviceDBPath = s.config.ServiceDatabasePath
		if s.config.DatabasePath != "" {
			mainDBPath = s.config.DatabasePath
		}
	}

	if mainDBPath == "" {
		mainDBPath = "data.db"
	}

	// Определяем формат экспорта
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "csv" // По умолчанию CSV для экспорта
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Вызываем функцию сканирования
	summary, err := ScanAndSummarizeAllDatabases(ctx, serviceDBPath, mainDBPath)
	if err != nil {
		LogError(r.Context(), err, "Failed to scan databases for export")
		s.handleHTTPError(w, r, NewInternalError("не удалось просканировать системы для экспорта", err))
		return
	}

	// Применяем фильтры
	filter := ParseSystemSummaryFilterFromRequest(r)
	if len(filter.Status) > 0 || filter.Search != "" || filter.CreatedAfter != nil || filter.CreatedBefore != nil || filter.SortBy != "" || filter.Limit > 0 {
		summary = ApplyFilters(summary, filter)
	}

	// Экспорт
	switch format {
	case "csv":
		if err := ExportSystemSummaryToCSV(w, summary); err != nil {
			LogError(r.Context(), err, "Failed to export to CSV")
			s.handleHTTPError(w, r, NewInternalError("не удалось экспортировать в CSV", err))
			return
		}
	case "json":
		if err := ExportSystemSummaryToJSON(w, summary); err != nil {
			LogError(r.Context(), err, "Failed to export to JSON")
			s.handleHTTPError(w, r, NewInternalError("не удалось экспортировать в JSON", err))
			return
		}
	default:
		s.handleHTTPError(w, r, NewValidationError("неподдерживаемый формат экспорта. Используйте 'csv' или 'json'", nil))
		return
	}

	LogInfo(r.Context(), "Системная сводка экспортирована", "format", format)
}

// handleSystemSummaryHealth проверяет здоровье системы сканирования
// GET /api/system/summary/health
func (s *Server) handleSystemSummaryHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":            "healthy",
		"cache_initialized": s.systemSummaryCache != nil,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	if s.systemSummaryCache != nil {
		stats := s.systemSummaryCache.GetStats()
		health["cache_stats"] = map[string]interface{}{
			"hits":     stats.Hits,
			"misses":   stats.Misses,
			"hit_rate": stats.HitRate,
			"has_data": stats.HasData,
			"is_stale": stats.IsStale,
		}
	}

	s.writeJSONResponse(w, r, health, http.StatusOK)
}

// handleSystemSummaryHistory возвращает историю сканирований
// GET /api/system/summary/history?limit=50
func (s *Server) handleSystemSummaryHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.scanHistoryManager == nil {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"error": "История сканирований не инициализирована",
		}, http.StatusServiceUnavailable)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	history, err := s.scanHistoryManager.GetHistory(ctx, limit)
	if err != nil {
		LogError(r.Context(), err, "Failed to get scan history")
		s.handleHTTPError(w, r, NewInternalError("не удалось получить историю сканирований", err))
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"history": history,
		"count":   len(history),
	}, http.StatusOK)
}

// handleSystemSummaryCompare сравнивает два сканирования
// GET /api/system/summary/compare?old_id=1&new_id=2
func (s *Server) handleSystemSummaryCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.scanHistoryManager == nil {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"error": "История сканирований не инициализирована",
		}, http.StatusServiceUnavailable)
		return
	}

	// Получаем ID сканирований
	oldIDStr := r.URL.Query().Get("old_id")
	newIDStr := r.URL.Query().Get("new_id")

	if oldIDStr == "" || newIDStr == "" {
		// Если ID не указаны, сравниваем последние два сканирования
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		history, err := s.scanHistoryManager.GetHistory(ctx, 2)
		if err != nil || len(history) < 2 {
			s.handleHTTPError(w, r, NewValidationError("недостаточно сканирований для сравнения (нужно минимум 2)", nil))
			return
		}

		oldSummary := history[1].Summary
		newSummary := history[0].Summary

		if oldSummary == nil || newSummary == nil {
			s.handleHTTPError(w, r, NewInternalError("не удалось получить данные сканирований", nil))
			return
		}

		diff := cache.CompareScans(oldSummary, newSummary)
		s.writeJSONResponse(w, r, map[string]interface{}{
			"old_scan": history[1],
			"new_scan": history[0],
			"diff":     diff,
		}, http.StatusOK)
		return
	}

	// TODO: Реализовать сравнение по ID (требует добавления метода GetScanByID)
	s.handleHTTPError(w, r, NewValidationError("сравнение по ID пока не поддерживается, используйте сравнение последних двух сканирований", nil))
}
