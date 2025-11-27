package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"httpserver/server/services"
)

// PatternDetectionHandler обработчик для обнаружения паттернов
type PatternDetectionHandler struct {
	patternDetectionService *services.PatternDetectionService
	baseHandler            *BaseHandler
	getNamesFunc           func(limit int, table, column string) ([]string, error)
}

// NewPatternDetectionHandler создает новый обработчик для обнаружения паттернов
func NewPatternDetectionHandler(
	patternDetectionService *services.PatternDetectionService,
	baseHandler *BaseHandler,
	getNamesFunc func(limit int, table, column string) ([]string, error),
) *PatternDetectionHandler {
	return &PatternDetectionHandler{
		patternDetectionService: patternDetectionService,
		baseHandler:            baseHandler,
		getNamesFunc:           getNamesFunc,
	}
}

// HandleDetectPatterns обрабатывает запросы к /api/patterns/detect
func (h *PatternDetectionHandler) HandleDetectPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.patternDetectionService.DetectPatterns(req.Name)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to detect patterns: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleSuggestPatterns обрабатывает запросы к /api/patterns/suggest
func (h *PatternDetectionHandler) HandleSuggestPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		Name  string `json:"name"`
		UseAI bool   `json:"use_ai,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.patternDetectionService.SuggestPatternCorrection(req.Name, req.UseAI)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to suggest patterns: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleTestBatch обрабатывает запросы к /api/patterns/test-batch
func (h *PatternDetectionHandler) HandleTestBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		Limit  int    `json:"limit,omitempty"`
		UseAI  bool   `json:"use_ai,omitempty"`
		Table  string `json:"table,omitempty"`
		Column string `json:"column,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Если тело пустое, используем дефолтные значения
		req.Limit = 50
		req.UseAI = false
		req.Table = "catalog_items"
		req.Column = "name"
	}

	if h.getNamesFunc == nil {
		h.baseHandler.WriteJSONError(w, r, "getNamesFunc not configured", http.StatusInternalServerError)
		return
	}

	result, err := h.patternDetectionService.TestPatternsBatch(
		req.Limit,
		req.UseAI,
		req.Table,
		req.Column,
		h.getNamesFunc,
	)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to test patterns: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

