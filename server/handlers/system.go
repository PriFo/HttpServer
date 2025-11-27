package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"httpserver/database"
	"httpserver/internal/infrastructure/cache"
	"httpserver/server/monitoring"
)

// SystemHandler обработчик для системных endpoints (health, stats, metrics)
type SystemHandler struct {
	*BaseHandler
	db              *database.DB
	healthChecker   *monitoring.HealthChecker
	metricsCollector *monitoring.MetricsCollector
}

// NewSystemHandler создает новый системный обработчик
func NewSystemHandler(
	baseHandler *BaseHandler,
	db *database.DB,
	healthChecker *monitoring.HealthChecker,
	metricsCollector *monitoring.MetricsCollector,
) *SystemHandler {
	return &SystemHandler{
		BaseHandler:     baseHandler,
		db:              db,
		healthChecker:   healthChecker,
		metricsCollector: metricsCollector,
	}
}

// HandleStats обрабатывает запрос статистики
// GET /api/stats
func (h *SystemHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.db == nil {
		http.Error(w, "Database not initialized", http.StatusServiceUnavailable)
		return
	}

	stats, err := h.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleHealth обрабатывает проверку здоровья сервера
// GET /api/v1/health
func (h *SystemHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if h.healthChecker != nil {
		h.healthChecker.HTTPHandler()(w, r)
	} else {
		// Fallback для обратной совместимости
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	}
}

// HandlePerformanceMetrics возвращает метрики производительности
// GET /api/performance/metrics
func (h *SystemHandler) HandlePerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.metricsCollector == nil {
		http.Error(w, "Metrics collector not initialized", http.StatusServiceUnavailable)
		return
	}

	metrics := h.metricsCollector.GetMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// SystemSummaryHandler обработчик для системных сводок
type SystemSummaryHandler struct {
	*BaseHandler
	systemSummaryCache *cache.SystemSummaryCache
	scanHistoryManager interface{} // *ScanHistoryManager (используем interface{} для избежания циклических зависимостей)
	config             interface{} // *config.Config
	currentDBPath      string
	// Функции для работы с системными сводками (передаются из server для избежания циклических зависимостей)
	scanAndSummarizeFunc func(ctx context.Context, serviceDBPath, mainDBPath string) (interface{}, error)
	parseFilterFunc       func(r *http.Request) interface{}
	applyFiltersFunc      func(summary interface{}, filter interface{}) interface{}
	exportCSVFunc         func(w http.ResponseWriter, summary interface{}) error
	exportJSONFunc        func(w http.ResponseWriter, summary interface{}) error
	compareScansFunc      func(old, new interface{}) map[string]interface{}
}

// NewSystemSummaryHandler создает новый обработчик системных сводок
func NewSystemSummaryHandler(
	baseHandler *BaseHandler,
	systemSummaryCache *cache.SystemSummaryCache,
	scanHistoryManager interface{},
	config interface{},
	currentDBPath string,
) *SystemSummaryHandler {
	return &SystemSummaryHandler{
		BaseHandler:        baseHandler,
		systemSummaryCache: systemSummaryCache,
		scanHistoryManager: scanHistoryManager,
		config:             config,
		currentDBPath:      currentDBPath,
		// Функции для работы с системными сводками (опциональные, nil по умолчанию)
		// Они могут быть установлены позже или переданы через отдельный метод
		scanAndSummarizeFunc: nil,
		parseFilterFunc:      nil,
		applyFiltersFunc:     nil,
		exportCSVFunc:        nil,
		exportJSONFunc:       nil,
		compareScansFunc:     nil,
	}
}

// HandleSystemSummaryCacheStats возвращает статистику кэша
// GET /api/system/summary/cache/stats
func (h *SystemSummaryHandler) HandleSystemSummaryCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.systemSummaryCache == nil {
		h.WriteJSONResponse(w, r, map[string]interface{}{
			"error": "Кеш не инициализирован",
		}, http.StatusServiceUnavailable)
		return
	}

	stats := h.systemSummaryCache.GetStats()
	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleSystemSummaryCacheInvalidate инвалидирует кэш
// POST /api/system/summary/cache/invalidate
func (h *SystemSummaryHandler) HandleSystemSummaryCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.systemSummaryCache == nil {
		h.WriteJSONResponse(w, r, map[string]interface{}{
			"error": "Кеш не инициализирован",
		}, http.StatusServiceUnavailable)
		return
	}

	h.systemSummaryCache.Invalidate()

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"message": "Кеш успешно инвалидирован",
		"stats":   h.systemSummaryCache.GetStats(),
	}, http.StatusOK)
}

// HandleSystemSummaryCacheClear очищает кэш
// POST /api/system/summary/cache/clear
func (h *SystemSummaryHandler) HandleSystemSummaryCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.systemSummaryCache == nil {
		h.WriteJSONResponse(w, r, map[string]interface{}{
			"error": "Кеш не инициализирован",
		}, http.StatusServiceUnavailable)
		return
	}

	h.systemSummaryCache.Clear()

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"message": "Кеш успешно очищен",
	}, http.StatusOK)
}

// HandleSystemSummaryHealth проверяет здоровье системы сканирования
// GET /api/system/summary/health
func (h *SystemSummaryHandler) HandleSystemSummaryHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":            "healthy",
		"cache_initialized": h.systemSummaryCache != nil,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	if h.systemSummaryCache != nil {
		stats := h.systemSummaryCache.GetStats()
		health["cache_stats"] = map[string]interface{}{
			"hits":     stats.Hits,
			"misses":   stats.Misses,
			"hit_rate": stats.HitRate,
			"has_data": stats.HasData,
			"is_stale": stats.IsStale,
		}
	}

	h.WriteJSONResponse(w, r, health, http.StatusOK)
}

// HandleSystemSummary обрабатывает запрос сводной информации по всем базам данных системы
// GET /api/system/summary
// ПРИМЕЧАНИЕ: Этот метод требует доступа к функциям из server пакета
// Пока что оставляем его в server.go для избежания циклических зависимостей
func (h *SystemSummaryHandler) HandleSystemSummary(w http.ResponseWriter, r *http.Request) {
	// Этот метод требует доступа к Server для полной функциональности
	// Пока что оставляем его в server.go
	http.Error(w, "Summary endpoint requires Server instance", http.StatusNotImplemented)
}


// HandleSystemSummaryExport экспортирует системную сводку в различных форматах
// GET /api/system/summary/export?format=csv|json
// ПРИМЕЧАНИЕ: Этот метод требует доступа к функциям из server пакета
// Пока что оставляем его в server.go для избежания циклических зависимостей
func (h *SystemSummaryHandler) HandleSystemSummaryExport(w http.ResponseWriter, r *http.Request) {
	// Этот метод требует доступа к Server для полной функциональности
	// Пока что оставляем его в server.go
	http.Error(w, "Export endpoint requires Server instance", http.StatusNotImplemented)
}


// HandleSystemSummaryHistory возвращает историю сканирований
// GET /api/system/summary/history?limit=50
// ПРИМЕЧАНИЕ: Этот метод требует доступа к функциям из server пакета
// Пока что оставляем его в server.go для избежания циклических зависимостей
func (h *SystemSummaryHandler) HandleSystemSummaryHistory(w http.ResponseWriter, r *http.Request) {
	// Этот метод требует доступа к Server для полной функциональности
	// Пока что оставляем его в server.go
	http.Error(w, "History endpoint requires Server instance", http.StatusNotImplemented)
}


// HandleSystemSummaryCompare сравнивает два сканирования
// GET /api/system/summary/compare?old_id=1&new_id=2
// ПРИМЕЧАНИЕ: Этот метод требует доступа к функциям из server пакета
// Пока что оставляем его в server.go для избежания циклических зависимостей
func (h *SystemSummaryHandler) HandleSystemSummaryCompare(w http.ResponseWriter, r *http.Request) {
	// Этот метод требует доступа к Server для полной функциональности
	// Пока что оставляем его в server.go
	http.Error(w, "Compare endpoint requires Server instance", http.StatusNotImplemented)
}


// HandleSystemSummaryStream обрабатывает SSE соединение для real-time обновлений сканирования
// GET /api/system/summary/stream
// Примечание: Этот метод требует доступа к Server для SSE функциональности
// Пока что оставляем его в server.go, так как он требует специальной обработки SSE
func (h *SystemSummaryHandler) HandleSystemSummaryStream(w http.ResponseWriter, r *http.Request) {
	// Этот метод требует доступа к Server для SSE функциональности
	// Пока что оставляем его в server.go
	http.Error(w, "Stream endpoint requires Server instance", http.StatusNotImplemented)
}

