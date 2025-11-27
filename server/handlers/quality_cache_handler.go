package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"httpserver/server/types"

	"github.com/gin-gonic/gin"
)

// HandleQualityCacheStats обрабатывает запрос статистики кэша качества проектов
func (h *QualityHandler) HandleQualityCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.projectStatsCache == nil {
		h.WriteJSONResponse(w, r, map[string]interface{}{
			"enabled": false,
			"message": "Cache is not initialized",
		}, http.StatusOK)
		return
	}

	stats := h.projectStatsCache.GetStats()
	h.WriteJSONResponse(w, r, map[string]interface{}{
		"enabled": true,
		"stats":   stats,
	}, http.StatusOK)
}

// HandleQualityCacheInvalidate обрабатывает запрос инвалидации кэша для проекта
func (h *QualityHandler) HandleQualityCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.projectStatsCache == nil {
		h.WriteJSONError(w, r, "Cache is not initialized", http.StatusInternalServerError)
		return
	}

	projectID, projectIDProvided, err := parseProjectIDFromRequest(r)
	if err != nil {
		h.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	if !projectIDProvided {
		h.projectStatsCache.Clear()
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Quality cache cleared",
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONResponse(w, r, map[string]interface{}{
			"message": "Cache cleared",
		}, http.StatusOK)
		return
	}

	// Инвалидируем кэш для конкретного проекта
	h.projectStatsCache.InvalidateProject(projectID)
	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Quality cache invalidated for project %d", projectID),
		Endpoint:  r.URL.Path,
	})
	h.WriteJSONResponse(w, r, map[string]interface{}{
		"message":    "Cache invalidated",
		"project_id": projectID,
	}, http.StatusOK)
}

// HandleQualityCacheStatsGin обрабатывает запрос статистики кэша через Gin
func (h *QualityHandler) HandleQualityCacheStatsGin(c *gin.Context) {
	if h.projectStatsCache == nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   "Quality cache stats requested but cache is not initialized",
			Endpoint:  c.Request.URL.Path,
		})
		c.JSON(http.StatusOK, map[string]interface{}{
			"enabled": false,
			"message": "Cache is not initialized",
		})
		return
	}

	stats := h.projectStatsCache.GetStats()
	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Quality cache stats retrieved",
		Endpoint:  c.Request.URL.Path,
	})
	SendJSONResponse(c, http.StatusOK, map[string]interface{}{
		"enabled": true,
		"stats":   stats,
	})
}

// HandleQualityCacheInvalidateGin обрабатывает запрос инвалидации кэша через Gin
func (h *QualityHandler) HandleQualityCacheInvalidateGin(c *gin.Context) {
	if h.projectStatsCache == nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Quality cache invalidation requested but cache is not initialized",
			Endpoint:  c.Request.URL.Path,
		})
		SendJSONError(c, http.StatusInternalServerError, "Cache is not initialized")
		return
	}

	projectID, projectIDProvided, err := parseProjectIDFromGin(c)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   err.Error(),
			Endpoint:  c.Request.URL.Path,
		})
		SendJSONError(c, http.StatusBadRequest, err.Error())
		return
	}

	if !projectIDProvided {
		h.projectStatsCache.Clear()
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Quality cache cleared",
			Endpoint:  c.Request.URL.Path,
		})
		SendJSONResponse(c, http.StatusOK, map[string]interface{}{
			"message": "Cache cleared",
		})
		return
	}

	// Инвалидируем кэш для конкретного проекта
	h.projectStatsCache.InvalidateProject(projectID)
	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Quality cache invalidated for project %d", projectID),
		Endpoint:  c.Request.URL.Path,
	})
	SendJSONResponse(c, http.StatusOK, map[string]interface{}{
		"message":    "Cache invalidated",
		"project_id": projectID,
	})
}

type cacheInvalidatePayload struct {
	ProjectID *int `json:"project_id"`
}

func parseProjectIDFromRequest(r *http.Request) (id int, provided bool, err error) {
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr != "" {
		projectID, convErr := strconv.Atoi(projectIDStr)
		if convErr != nil {
			return 0, false, fmt.Errorf("Invalid project_id")
		}
		return projectID, true, nil
	}

	// Попытка прочитать JSON body
	if r.Body == nil {
		return 0, false, nil
	}
	bodyBytes, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return 0, false, fmt.Errorf("Failed to read request body")
	}
	if len(bodyBytes) == 0 {
		return 0, false, nil
	}

	var payload cacheInvalidatePayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return 0, false, fmt.Errorf("Invalid JSON payload")
	}

	if payload.ProjectID == nil {
		return 0, false, nil
	}
	if *payload.ProjectID <= 0 {
		return 0, false, fmt.Errorf("Invalid project_id")
	}

	return *payload.ProjectID, true, nil
}

func parseProjectIDFromGin(c *gin.Context) (id int, provided bool, err error) {
	projectIDStr := c.Query("project_id")
	if projectIDStr != "" {
		projectID, convErr := strconv.Atoi(projectIDStr)
		if convErr != nil {
			return 0, false, fmt.Errorf("Invalid project_id format: %s", projectIDStr)
		}
		return projectID, true, nil
	}

	var payload cacheInvalidatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		if err == io.EOF {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("Invalid JSON payload: %w", err)
	}

	if payload.ProjectID == nil {
		return 0, false, nil
	}
	if *payload.ProjectID <= 0 {
		return 0, false, fmt.Errorf("Invalid project_id: must be positive")
	}

	return *payload.ProjectID, true, nil
}
