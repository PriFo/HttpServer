package quality

import (
	"net/http"
	"strconv"
	"time"

	qualityapp "httpserver/internal/application/quality"
	qualitydomain "httpserver/internal/domain/quality"
	"httpserver/internal/api/handlers/common"
)

// Handler HTTP обработчик для работы с качеством данных
type Handler struct {
	baseHandler common.BaseHandlerInterface
	useCase     *qualityapp.UseCase
}

// NewHandler создает новый HTTP обработчик для качества
func NewHandler(
	baseHandler common.BaseHandlerInterface,
	useCase *qualityapp.UseCase,
) *Handler {
	return &Handler{
		baseHandler: baseHandler,
		useCase:     useCase,
	}
}

// HandleAnalyzeQuality запускает анализ качества для выгрузки
// POST /api/v1/upload/{upload_uuid}/quality/analyze
func (h *Handler) HandleAnalyzeQuality(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadID := r.URL.Query().Get("upload_uuid")
	if uploadID == "" {
		// Попытаемся извлечь из пути
		uploadID = extractUUIDFromPath(r.URL.Path)
		if uploadID == "" {
			h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
			return
		}
	}

	if err := h.useCase.AnalyzeQuality(r.Context(), uploadID); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "analysis started"}, http.StatusOK)
}

// HandleGetQualityReport возвращает отчет о качестве
// GET /api/v1/upload/{upload_uuid}/quality/report
func (h *Handler) HandleGetQualityReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadID := r.URL.Query().Get("upload_uuid")
	if uploadID == "" {
		uploadID = extractUUIDFromPath(r.URL.Path)
		if uploadID == "" {
			h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
			return
		}
	}

	summaryOnly := r.URL.Query().Get("summary_only") == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	result, err := h.useCase.GetQualityReport(r.Context(), uploadID, summaryOnly, limit, offset)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusNotFound)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetQualityDashboard возвращает дашборд качества
// GET /api/v1/databases/{database_id}/quality/dashboard
func (h *Handler) HandleGetQualityDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	databaseIDStr := r.URL.Query().Get("database_id")
	if databaseIDStr == "" {
		databaseIDStr = extractIDFromPath(r.URL.Path)
	}

	databaseID, err := strconv.Atoi(databaseIDStr)
	if err != nil || databaseID <= 0 {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid database_id"}, http.StatusBadRequest)
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 30
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	result, err := h.useCase.GetQualityDashboard(r.Context(), databaseID, days, limit)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetQualityIssues возвращает проблемы качества
// GET /api/v1/quality/issues
func (h *Handler) HandleGetQualityIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	filter := qualitydomain.QualityIssueFilter{
		Severity:   parseStringSlice(r.URL.Query().Get("severity")),
		EntityType: r.URL.Query().Get("entity_type"),
		EntityID:   r.URL.Query().Get("entity_id"),
		Field:      r.URL.Query().Get("field"),
		Limit:      100,
		Offset:     0,
	}

	if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && limit > 0 {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && offset >= 0 {
		filter.Offset = offset
	}

	if resolvedStr := r.URL.Query().Get("resolved"); resolvedStr != "" {
		resolved := resolvedStr == "true"
		filter.Resolved = &resolved
	}

	result, err := h.useCase.GetQualityIssues(r.Context(), filter)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetQualityStatistics возвращает статистику качества
// GET /api/v1/databases/{database_id}/quality/statistics
func (h *Handler) HandleGetQualityStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	databaseIDStr := r.URL.Query().Get("database_id")
	if databaseIDStr == "" {
		databaseIDStr = extractIDFromPath(r.URL.Path)
	}

	databaseID, err := strconv.Atoi(databaseIDStr)
	if err != nil || databaseID <= 0 {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid database_id"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.GetQualityStatistics(r.Context(), databaseID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetQualityTrends возвращает тренды качества
// GET /api/v1/databases/{database_id}/quality/trends
func (h *Handler) HandleGetQualityTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	databaseIDStr := r.URL.Query().Get("database_id")
	if databaseIDStr == "" {
		databaseIDStr = extractIDFromPath(r.URL.Path)
	}

	databaseID, err := strconv.Atoi(databaseIDStr)
	if err != nil || databaseID <= 0 {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid database_id"}, http.StatusBadRequest)
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 30
	}

	period := time.Duration(days) * 24 * time.Hour

	result, err := h.useCase.GetQualityTrends(r.Context(), databaseID, period)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"trends": result,
		"count":  len(result),
	}, http.StatusOK)
}

// Вспомогательные функции

func extractUUIDFromPath(path string) string {
	// Простая реализация извлечения UUID из пути
	// TODO: Улучшить парсинг пути
	return ""
}

func extractIDFromPath(path string) string {
	// Простая реализация извлечения ID из пути
	// TODO: Улучшить парсинг пути
	return ""
}

func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

