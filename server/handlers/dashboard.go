package handlers

import (
	"fmt"
	"net/http"
	"time"

	"httpserver/server/services"
	"httpserver/server/types"
)

// DashboardHandler обработчик для работы с дашбордом
type DashboardHandler struct {
	dashboardService     *services.DashboardService
	clientService        *services.ClientService
	normalizationService *services.NormalizationService
	qualityService       services.QualityServiceInterface // Используем интерфейс для улучшения тестируемости
	baseHandler          *BaseHandler
	getMonitoringMetrics func() MonitoringData // Функция для получения метрик провайдеров
}

// NewDashboardHandler создает новый обработчик для работы с дашбордом
func NewDashboardHandler(
	dashboardService *services.DashboardService,
	baseHandler *BaseHandler,
) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		baseHandler:      baseHandler,
	}
}

// NewDashboardHandlerWithServices создает новый обработчик с полным набором сервисов
// Принимает интерфейс QualityServiceInterface для улучшения тестируемости
func NewDashboardHandlerWithServices(
	dashboardService *services.DashboardService,
	clientService *services.ClientService,
	normalizationService *services.NormalizationService,
	qualityService services.QualityServiceInterface, // Используем интерфейс вместо конкретного типа
	baseHandler *BaseHandler,
	getMonitoringMetrics func() MonitoringData,
) *DashboardHandler {
	return &DashboardHandler{
		dashboardService:     dashboardService,
		clientService:        clientService,
		normalizationService: normalizationService,
		qualityService:       qualityService,
		baseHandler:          baseHandler,
		getMonitoringMetrics: getMonitoringMetrics,
	}
}

// HandleGetStats обрабатывает запросы к /api/dashboard/stats
func (h *DashboardHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	stats, err := h.dashboardService.GetStats()
	if err != nil || stats == nil {
		h.baseHandler.WriteJSONResponse(w, r, buildDashboardStatsFallback(err), http.StatusOK)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleGetNormalizationStatus обрабатывает запросы к /api/dashboard/normalization-status
func (h *DashboardHandler) HandleGetNormalizationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	status, err := h.dashboardService.GetNormalizationStatus()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить статус нормализации: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, status, http.StatusOK)
}

// HandleGetQualityMetrics обрабатывает запросы к /api/quality/metrics
func (h *DashboardHandler) HandleGetQualityMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	metrics, err := h.dashboardService.GetQualityMetrics()
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, buildQualityMetricsFallback(err), http.StatusOK)
		return
	}

	if metrics == nil {
		metrics = &services.QualityMetrics{}
	}

	h.baseHandler.WriteJSONResponse(w, r, metrics, http.StatusOK)
}

// HandleDashboardOverview обрабатывает запросы к /api/dashboard/overview
// Возвращает агрегированные данные для главной страницы дашборда
func (h *DashboardHandler) HandleDashboardOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	response := types.DashboardOverviewResponse{
		RecentActivity: []types.ActivityLog{},
		SystemHealth:   "ok",
	}

	// Получаем общее количество клиентов
	if h.clientService != nil {
		clients, err := h.clientService.GetAllClients(r.Context())
		if err == nil {
			response.TotalClients = len(clients)
		}
	}

	// Получаем статистику через dashboardService
	stats, err := h.dashboardService.GetStats()
	if err == nil {
		if totalDBs, ok := stats["total_databases"].(int); ok {
			response.TotalDatabases = totalDBs
		}
		if totalProjects, ok := stats["total_projects"].(int); ok {
			response.TotalProjects = totalProjects
		}
	}

	// Получаем статус нормализации
	if h.normalizationService != nil {
		status := h.normalizationService.GetStatus()
		// Рассчитываем прогресс
		progress := 0.0
		total := status.Processed + status.Success + status.Errors
		if total > 0 {
			progress = float64(status.Processed) / float64(total) * 100
		}

		response.NormalizationStatus = types.NormalizationStatus{
			IsRunning:   status.IsRunning,
			Progress:    progress,
			Processed:   status.Processed,
			Total:       total,
			Success:     status.Success,
			Errors:      status.Errors,
			CurrentStep: "Не запущено",
			Logs:        []string{},
		}
		if status.IsRunning {
			response.NormalizationStatus.CurrentStep = "Выполняется"
			if status.StartTime != "" {
				response.NormalizationStatus.StartTime = status.StartTime
				response.NormalizationStatus.ElapsedTime = status.ElapsedTime
				// Парсим ElapsedTime для расчета rate
				if elapsed, err := time.ParseDuration(status.ElapsedTime); err == nil && elapsed.Seconds() > 0 {
					response.NormalizationStatus.Rate = float64(status.Processed) / elapsed.Seconds()
				}
			}
		}
	} else {
		// Fallback через dashboardService
		status, err := h.dashboardService.GetNormalizationStatus()
		if err == nil {
			if isRunning, ok := status["is_running"].(bool); ok && isRunning {
				response.NormalizationStatus.IsRunning = true
				response.NormalizationStatus.CurrentStep = "Выполняется"
			}
		}
	}

	// Получаем общий quality score
	qualityMetrics, err := h.dashboardService.GetQualityMetrics()
	if err == nil && qualityMetrics != nil {
		response.OverallQualityScore = qualityMetrics.OverallQuality
	}

	// Получаем метрики провайдеров
	if h.getMonitoringMetrics != nil {
		providerMetrics := h.getMonitoringMetrics()
		response.ProviderMetrics = providerMetrics

		// Определяем SystemHealth на основе метрик провайдеров
		hasErrors := false
		hasWarnings := false
		for _, provider := range providerMetrics.Providers {
			if provider.Status == "error" {
				hasErrors = true
				break
			}
			if provider.AverageLatencyMs > 2000 || provider.FailedRequests > 0 {
				hasWarnings = true
			}
		}
		if hasErrors {
			response.SystemHealth = "error"
		} else if hasWarnings {
			response.SystemHealth = "warning"
		}
	}

	// Получаем RecentActivity из разных источников (uploads, databases, projects)
	if h.dashboardService != nil {
		activities, err := h.dashboardService.GetRecentActivity(20)
		if err == nil {
			// Преобразуем services.ActivityLog в types.ActivityLog
			response.RecentActivity = make([]types.ActivityLog, len(activities))
			for i, activity := range activities {
				response.RecentActivity[i] = types.ActivityLog{
					ID:        activity.ID,
					Type:      activity.Type,
					Message:   activity.Message,
					Timestamp: activity.Timestamp,
					ClientID:  activity.ClientID,
					ProjectID: activity.ProjectID,
				}
			}
		}
		// Игнорируем ошибки получения активности - это не критично для дашборда
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

func buildDashboardStatsFallback(err error) map[string]interface{} {
	reason := "dashboard stats are temporarily unavailable"
	if err != nil {
		reason = fmt.Sprintf("dashboard stats unavailable: %v", err)
	}

	return map[string]interface{}{
		"totalRecords":     0,
		"totalDatabases":   0,
		"processedRecords": 0,
		"createdGroups":    0,
		"mergedRecords":    0,
		"systemVersion":    "unknown",
		"currentDatabase":  nil,
		"isFallback":       true,
		"fallbackReason":   reason,
		"timestamp":        time.Now().Format(time.RFC3339),
		"normalizationStatus": map[string]interface{}{
			"status":       "idle",
			"progress":     0,
			"currentStage": "Недоступно",
			"startTime":    nil,
			"endTime":      nil,
		},
		"qualityMetrics": map[string]interface{}{
			"overallQuality":   0.0,
			"highConfidence":   0.0,
			"mediumConfidence": 0.0,
			"lowConfidence":    0.0,
			"totalRecords":     0,
		},
	}
}

type qualityMetricsFallback struct {
	OverallQuality   float64 `json:"overallQuality"`
	HighConfidence   float64 `json:"highConfidence"`
	MediumConfidence float64 `json:"mediumConfidence"`
	LowConfidence    float64 `json:"lowConfidence"`
	TotalRecords     int     `json:"totalRecords"`
	IsFallback       bool    `json:"isFallback"`
	FallbackReason   string  `json:"fallbackReason"`
	Timestamp        string  `json:"timestamp"`
}

func buildQualityMetricsFallback(err error) qualityMetricsFallback {
	reason := "quality metrics are temporarily unavailable"
	if err != nil {
		reason = fmt.Sprintf("quality metrics unavailable: %v", err)
	}

	return qualityMetricsFallback{
		OverallQuality:   0,
		HighConfidence:   0,
		MediumConfidence: 0,
		LowConfidence:    0,
		TotalRecords:     0,
		IsFallback:       true,
		FallbackReason:   reason,
		Timestamp:        time.Now().Format(time.RFC3339),
	}
}
