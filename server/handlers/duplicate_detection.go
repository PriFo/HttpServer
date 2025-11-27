package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"httpserver/normalization/algorithms"
	"httpserver/server/services"
)

// DuplicateDetectionHandler обработчик для обнаружения дубликатов
type DuplicateDetectionHandler struct {
	duplicateDetectionService *services.DuplicateDetectionService
	baseHandler              *BaseHandler
}

// NewDuplicateDetectionHandler создает новый обработчик для обнаружения дубликатов
func NewDuplicateDetectionHandler(
	duplicateDetectionService *services.DuplicateDetectionService,
	baseHandler *BaseHandler,
) *DuplicateDetectionHandler {
	return &DuplicateDetectionHandler{
		duplicateDetectionService: duplicateDetectionService,
		baseHandler:               baseHandler,
	}
}

// HandleStartDetection обрабатывает запросы к /api/duplicates/detect
func (h *DuplicateDetectionHandler) HandleStartDetection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		ProjectID   int                          `json:"project_id"`
		Threshold   float64                      `json:"threshold"`
		BatchSize   int                          `json:"batch_size"`
		UseAdvanced bool                         `json:"use_advanced"`
		Weights     *algorithms.SimilarityWeights `json:"weights,omitempty"`
		MaxItems    int                          `json:"max_items,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	taskID, err := h.duplicateDetectionService.StartDetection(
		req.ProjectID,
		req.Threshold,
		req.BatchSize,
		req.UseAdvanced,
		req.Weights,
		req.MaxItems,
	)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to start detection: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"task_id": taskID,
		"status":  "started",
		"message": "Duplicate detection started",
	}, http.StatusAccepted)
}

// HandleGetStatus обрабатывает запросы к /api/duplicates/detect/{taskID}
func (h *DuplicateDetectionHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Извлекаем taskID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/duplicates/detect/")
	if path == "" {
		h.baseHandler.WriteJSONError(w, r, "task_id is required", http.StatusBadRequest)
		return
	}

	task, err := h.duplicateDetectionService.GetTaskStatus(path)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get task status: %v", err), http.StatusNotFound)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, task, http.StatusOK)
}

