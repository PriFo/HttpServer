package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"httpserver/classification"
	"httpserver/database"
	apperrors "httpserver/server/errors"
	"httpserver/server/services"
)

// LogEntry представляет запись лога
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Endpoint  string
}

// ClassificationHandler обработчик для классификации
type ClassificationHandler struct {
	*BaseHandler
	classificationService *services.ClassificationService
	logFunc               func(entry interface{}) // server.LogEntry, но без прямого импорта
}

// ClassifyRequest описывает запрос на классификацию
type ClassifyRequest struct {
	ItemID     int                    `json:"item_id"`
	ItemName   string                 `json:"item_name"`
	ItemCode   string                 `json:"item_code,omitempty"`
	Classifier string                 `json:"classifier,omitempty"`
	Model      string                 `json:"model,omitempty"`
	StrategyID string                 `json:"strategy_id,omitempty"`
	Category   string                 `json:"category,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// ClassifyResponse описывает ответ классификации
type ClassifyResponse struct {
	ItemID      int                    `json:"item_id"`
	ItemName    string                 `json:"item_name"`
	ItemCode    string                 `json:"item_code,omitempty"`
	Classifier  string                 `json:"classifier"`
	Code        string                 `json:"code,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Subcategory string                 `json:"subcategory,omitempty"`
	Confidence  float64                `json:"confidence"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
}

type configureStrategyRequest struct {
	ClientID    *int                         `json:"client_id,omitempty"`
	StrategyID  string                       `json:"strategy_id,omitempty"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	MaxDepth    int                          `json:"max_depth"`
	Priority    []string                     `json:"priority"`
	Rules       []classification.FoldingRule `json:"rules"`
	IsDefault   bool                         `json:"is_default"`
}

// ClassificationStatsResponse описывает ответ статистики классификации
type ClassificationStatsResponse struct {
	TotalItems        int     `json:"total_items"`
	ClassifiedItems   int     `json:"classified_items"`
	PendingItems      int     `json://pending_items"`
	AverageConfidence float64 `json:"average_confidence"`
}

func (h *ClassificationHandler) resolveClassifyItemName(req *ClassifyRequest) (string, error) {
	name := strings.TrimSpace(req.ItemName)
	if name != "" {
		return name, nil
	}

	if req.ItemID > 0 {
		if h.classificationService == nil {
			return "", apperrors.NewInternalError("classification service not available", nil)
		}

		resolvedName, err := h.classificationService.GetNormalizedItemName(req.ItemID)
		if err != nil {
			return "", err
		}
		return resolvedName, nil
	}

	return "", apperrors.NewValidationError("item_name or item_id is required", nil)
}

func (h *ClassificationHandler) handleClassificationError(w http.ResponseWriter, r *http.Request, err error, contextMessage string) {
	if err == nil {
		return
	}

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("%s: %v", contextMessage, err),
			Endpoint:  r.URL.Path,
		})
	}

	if strings.Contains(err.Error(), "API key not configured") || strings.Contains(err.Error(), "API ключ AI не настроен") {
		h.WriteJSONError(w, r, "AI API key not configured", http.StatusServiceUnavailable)
		return
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		h.WriteJSONError(w, r, appErr.Message, appErr.StatusCode())
		return
	}

	h.WriteJSONError(w, r, fmt.Sprintf("%s: %v", contextMessage, err), http.StatusInternalServerError)
}

// NewClassificationHandler создает новый обработчик классификации
func NewClassificationHandler(baseHandler *BaseHandler, classificationService *services.ClassificationService, logFunc func(entry interface{})) *ClassificationHandler {
	return &ClassificationHandler{
		BaseHandler:           baseHandler,
		classificationService: classificationService,
		logFunc:               logFunc,
	}
}

func sortedStrategyIDs(strategies map[string]classification.FoldingStrategyConfig) []string {
	ids := make([]string, 0, len(strategies))
	for id := range strategies {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// HandleKpvedHierarchy обрабатывает запрос иерархии КПВЭД
func (h *ClassificationHandler) HandleKpvedHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры
	parentCode := r.URL.Query().Get("parent")
	level := r.URL.Query().Get("level")

	// Валидация уровня
	if level != "" {
		if _, err := ValidateIntPathParam(level, "level"); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid level parameter: %s", err.Error()), http.StatusBadRequest)
			return
		}
	}

	nodes, err := h.classificationService.GetKpvedHierarchy(parentCode, level)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting KPVED hierarchy: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to fetch KPVED hierarchy: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"nodes": nodes,
		"total": len(nodes),
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleKpvedSearch обрабатывает запрос поиска по КПВЭД
func (h *ClassificationHandler) HandleKpvedSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		h.WriteJSONError(w, r, "Search query is required", http.StatusBadRequest)
		return
	}

	limit, err := ValidateIntParam(r, "limit", 50, 1, 100)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid limit parameter: %s", err.Error()), http.StatusBadRequest)
		return
	}

	items, err := h.classificationService.SearchKpved(searchQuery, limit)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error searching KPVED: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to search KPVED: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, items, http.StatusOK)
}

// HandleKpvedStats обрабатывает запрос статистики КПВЭД
func (h *ClassificationHandler) HandleKpvedStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.classificationService.GetKpvedStats()
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting KPVED stats: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get KPVED stats: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleKpvedLoad обрабатывает запрос загрузки КПВЭД из файла
func (h *ClassificationHandler) HandleKpvedLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		h.WriteJSONError(w, r, "file_path is required", http.StatusBadRequest)
		return
	}

	totalCodes, err := h.classificationService.LoadKpvedFromFile(req.FilePath)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error loading KPVED: %v", err),
			Endpoint:  r.URL.Path,
		})
		if strings.Contains(err.Error(), "not found") {
			h.WriteJSONError(w, r, fmt.Sprintf("File not found: %s", req.FilePath), http.StatusNotFound)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to load KPVED: %v", err), http.StatusInternalServerError)
		}
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":     true,
		"message":     "KPVED classifier loaded successfully",
		"total_codes": totalCodes,
	}, http.StatusOK)
}

// HandleKpvedClassifyTest обрабатывает запрос тестирования классификации
func (h *ClassificationHandler) HandleKpvedClassifyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NormalizedName string `json:"normalized_name"`
		Model          string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NormalizedName == "" {
		h.WriteJSONError(w, r, "normalized_name is required", http.StatusBadRequest)
		return
	}

	result, err := h.classificationService.ClassifyTest(req.NormalizedName, req.Model)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error classifying: %v", err),
			Endpoint:  r.URL.Path,
		})
		if strings.Contains(err.Error(), "not configured") {
			h.WriteJSONError(w, r, "AI API key not configured", http.StatusServiceUnavailable)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Classification failed: %v", err), http.StatusInternalServerError)
		}
		return
	}

	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleKpvedClassifyHierarchical обрабатывает запрос иерархической классификации
func (h *ClassificationHandler) HandleKpvedClassifyHierarchical(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NormalizedName string `json:"normalized_name"`
		Category       string `json:"category"`
		Model          string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NormalizedName == "" {
		h.WriteJSONError(w, r, "normalized_name is required", http.StatusBadRequest)
		return
	}

	result, err := h.classificationService.ClassifyHierarchical(req.NormalizedName, req.Category, req.Model)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error classifying: %v", err),
			Endpoint:  r.URL.Path,
		})
		if strings.Contains(err.Error(), "not configured") {
			h.WriteJSONError(w, r, "AI API key not configured", http.StatusServiceUnavailable)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Classification failed: %v", err), http.StatusInternalServerError)
		}
		return
	}

	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleResetClassification обрабатывает запрос сброса классификации
func (h *ClassificationHandler) HandleResetClassification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NormalizedName string  `json:"normalized_name"`
		Category       string  `json:"category"`
		KpvedCode      string  `json:"kpved_code"`
		MinConfidence  float64 `json:"min_confidence"`
		ResetAll       bool    `json:"reset_all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if !req.ResetAll && req.NormalizedName == "" && req.Category == "" && req.KpvedCode == "" && req.MinConfidence == 0 {
		h.WriteJSONError(w, r, "Не указаны критерии для сброса. Укажите normalized_name, category, kpved_code или установите reset_all=true", http.StatusBadRequest)
		return
	}

	rowsAffected, err := h.classificationService.ResetClassification(req.NormalizedName, req.Category, req.KpvedCode, req.MinConfidence, req.ResetAll)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error resetting classification: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to reset classification: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":       true,
		"message":       "Классификация сброшена",
		"rows_affected": rowsAffected,
	}, http.StatusOK)
}

// HandleMarkIncorrect обрабатывает запрос отметки классификации как неправильной
func (h *ClassificationHandler) HandleMarkIncorrect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NormalizedName string `json:"normalized_name"`
		Category       string `json:"category"`
		Reason         string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.NormalizedName == "" || req.Category == "" {
		h.WriteJSONError(w, r, "normalized_name and category are required", http.StatusBadRequest)
		return
	}

	rowsAffected, err := h.classificationService.MarkIncorrect(req.NormalizedName, req.Category, req.Reason)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error marking as incorrect: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to mark as incorrect: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":         true,
		"message":         "Классификация помечена как неправильная и сброшена",
		"rows_affected":   rowsAffected,
		"normalized_name": req.NormalizedName,
		"category":        req.Category,
		"reason":          req.Reason,
	}, http.StatusOK)
}

// HandleMarkCorrect обрабатывает запрос снятия пометки неправильной классификации
func (h *ClassificationHandler) HandleMarkCorrect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NormalizedName string `json:"normalized_name"`
		Category       string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.NormalizedName == "" || req.Category == "" {
		h.WriteJSONError(w, r, "normalized_name and category are required", http.StatusBadRequest)
		return
	}

	rowsAffected, err := h.classificationService.MarkCorrect(req.NormalizedName, req.Category)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error marking as correct: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to mark as correct: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":         true,
		"message":         "Пометка неправильной классификации снята",
		"rows_affected":   rowsAffected,
		"normalized_name": req.NormalizedName,
		"category":        req.Category,
	}, http.StatusOK)
}

// HandleClassifyItem обрабатывает запросы к /api/classification/classify
func (h *ClassificationHandler) HandleClassifyItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req ClassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	itemName, err := h.resolveClassifyItemName(&req)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to resolve item name")
		return
	}

	model := req.Model
	if model == "" {
		model = req.Classifier
	}

	strategyID := resolveStrategyID(req.StrategyID)

	aiResponse, classifierName, err := h.classificationService.ClassifyItemAI(itemName, req.ItemCode, req.Category, model, req.Context)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to classify item")
		return
	}

	categoryPath := normalizeCategoryPath(aiResponse.CategoryPath, req.Category)
	classificationPayload := map[string]interface{}{
		"category_path": categoryPath,
		"confidence":    aiResponse.Confidence,
		"reasoning":     aiResponse.Reasoning,
		"alternatives":  normalizeAlternatives(aiResponse.Alternatives),
		"strategy":      strategyID,
	}

	response := map[string]interface{}{
		"item_id":        req.ItemID,
		"item_name":      itemName,
		"item_code":      req.ItemCode,
		"original_name":  itemName,
		"classifier":     classifierName,
		"strategy":       strategyID,
		"strategy_id":    strategyID,
		"category":       categoryPath,
		"category_path":  categoryPath,
		"confidence":     aiResponse.Confidence,
		"reasoning":      aiResponse.Reasoning,
		"context":        req.Context,
		"options":        req.Options,
		"classification": classificationPayload,
	}

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Classified item '%s'", itemName),
			Endpoint:  r.URL.Path,
		})
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleClassifyItemDirect обрабатывает запросы к /api/classification/classify-item
func (h *ClassificationHandler) HandleClassifyItemDirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req ClassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	itemName, err := h.resolveClassifyItemName(&req)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to resolve item name")
		return
	}

	model := req.Model
	if model == "" {
		model = req.Classifier
	}

	strategyID := resolveStrategyID(req.StrategyID)

	aiResponse, classifierName, err := h.classificationService.ClassifyItemAI(itemName, req.ItemCode, req.Category, model, req.Context)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to classify item")
		return
	}

	categoryPath := normalizeCategoryPath(aiResponse.CategoryPath, req.Category)
	classificationPayload := map[string]interface{}{
		"category_path": categoryPath,
		"confidence":    aiResponse.Confidence,
		"reasoning":     aiResponse.Reasoning,
		"alternatives":  normalizeAlternatives(aiResponse.Alternatives),
		"strategy":      strategyID,
	}

	response := map[string]interface{}{
		"item_name":      itemName,
		"item_code":      req.ItemCode,
		"original_name":  itemName,
		"strategy":       strategyID,
		"strategy_id":    strategyID,
		"classifier":     classifierName,
		"category":       categoryPath,
		"category_path":  categoryPath,
		"confidence":     aiResponse.Confidence,
		"reasoning":      aiResponse.Reasoning,
		"context":        req.Context,
		"classification": classificationPayload,
	}

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Direct classification completed for '%s'", itemName),
			Endpoint:  r.URL.Path,
		})
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

func resolveStrategyID(strategy string) string {
	id := strings.TrimSpace(strategy)
	if id == "" {
		return "top_priority"
	}
	return id
}

func normalizeCategoryPath(path []string, fallback string) []string {
	if len(path) > 0 {
		return path
	}

	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return []string{}
	}

	// Документация ожидает массив, даже если указан только один уровень
	return []string{fallback}
}

func normalizeAlternatives(alternatives [][]string) [][]string {
	if len(alternatives) == 0 {
		return [][]string{}
	}

	normalized := make([][]string, 0, len(alternatives))
	for _, row := range alternatives {
		if len(row) == 0 {
			continue
		}
		normalized = append(normalized, row)
	}

	return normalized
}

// HandleGetStrategies обрабатывает запросы к /api/classification/strategies
func (h *ClassificationHandler) HandleGetStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	strategyManager := classification.NewStrategyManager()
	allStrategies := strategyManager.GetAllStrategies()

	result := make([]classification.FoldingStrategyConfig, 0, len(allStrategies))
	for _, id := range sortedStrategyIDs(allStrategies) {
		result = append(result, allStrategies[id])
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"strategies": result,
		"total":      len(result),
	}, http.StatusOK)
}

// HandleConfigureStrategy обрабатывает запросы к /api/classification/strategies/configure
func (h *ClassificationHandler) HandleConfigureStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req configureStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		h.WriteJSONError(w, r, "name is required", http.StatusBadRequest)
		return
	}

	if req.MaxDepth <= 0 {
		req.MaxDepth = 2
	}

	if strings.TrimSpace(req.StrategyID) == "" {
		req.StrategyID = fmt.Sprintf("custom_%d", time.Now().UnixNano())
	}

	config := classification.FoldingStrategyConfig{
		ID:          req.StrategyID,
		Name:        req.Name,
		Description: req.Description,
		MaxDepth:    req.MaxDepth,
		Priority:    req.Priority,
		Rules:       req.Rules,
	}

	if req.ClientID != nil && *req.ClientID > 0 {
		if h.classificationService == nil {
			h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
			return
		}

		createdStrategy, err := h.classificationService.CreateClientStrategy(*req.ClientID, config)
		if err != nil {
			if h.logFunc != nil {
				h.logFunc(LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Failed to save client strategy: %v", err),
					Endpoint:  r.URL.Path,
				})
			}
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to configure strategy: %v", err), http.StatusInternalServerError)
			return
		}

		h.WriteJSONResponse(w, r, map[string]interface{}{
			"success":      true,
			"message":      "Strategy configured and saved",
			"client_id":    *req.ClientID,
			"strategy_id":  createdStrategy.ID,
			"is_default":   req.IsDefault,
			"strategy":     createdStrategy,
			"strategy_raw": createdStrategy.StrategyConfig,
		}, http.StatusCreated)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":       true,
		"message":       "Strategy configured",
		"strategy_id":   config.ID,
		"strategy":      config,
		"is_persistent": false,
	}, http.StatusOK)
}

// HandleGetClientStrategies обрабатывает запросы к /api/classification/strategies/client
func (h *ClassificationHandler) HandleGetClientStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	clientIDParam := r.URL.Query().Get("client_id")
	if clientIDParam == "" {
		h.WriteJSONError(w, r, "client_id parameter is required", http.StatusBadRequest)
		return
	}

	clientID, err := ValidateIntPathParam(clientIDParam, "client_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
		return
	}

	strategies, err := h.classificationService.GetFoldingStrategies(&clientID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get client strategies: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, "Failed to get client strategies", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"client_id":   clientID,
		"strategies":  strategies,
		"total_count": len(strategies),
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleCreateOrUpdateClientStrategy обрабатывает запросы к /api/classification/strategies/create
func (h *ClassificationHandler) HandleCreateOrUpdateClientStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		ClientID    int                          `json:"client_id"`
		Name        string                       `json:"name"`
		Description string                       `json:"description"`
		MaxDepth    int                          `json:"max_depth"`
		Priority    []string                     `json:"priority"`
		Rules       []classification.FoldingRule `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.ClientID <= 0 {
		h.WriteJSONError(w, r, "client_id is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.WriteJSONError(w, r, "name is required", http.StatusBadRequest)
		return
	}
	if req.MaxDepth <= 0 {
		req.MaxDepth = 2
	}

	config := classification.FoldingStrategyConfig{
		Name:        req.Name,
		Description: req.Description,
		MaxDepth:    req.MaxDepth,
		Priority:    req.Priority,
		Rules:       req.Rules,
	}

	createdStrategy, err := h.classificationService.CreateClientStrategy(req.ClientID, config)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to create client strategy: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to create strategy: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"strategy": map[string]interface{}{
			"id":              createdStrategy.ID,
			"name":            createdStrategy.Name,
			"description":     createdStrategy.Description,
			"strategy_config": createdStrategy.StrategyConfig,
			"client_id":       createdStrategy.ClientID,
			"is_default":      createdStrategy.IsDefault,
			"created_at":      createdStrategy.CreatedAt,
			"updated_at":      createdStrategy.UpdatedAt,
		},
		"message":     "Strategy created successfully",
		"strategy_id": createdStrategy.ID,
	}, http.StatusCreated)
}

// HandleGetAvailableStrategies обрабатывает запросы к /api/classification/available
func (h *ClassificationHandler) HandleGetAvailableStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	strategyManager := classification.NewStrategyManager()
	allStrategies := strategyManager.GetAllStrategies()

	clientIDParam := strings.TrimSpace(r.URL.Query().Get("client_id"))
	var clientID *int
	if clientIDParam != "" {
		id, err := ValidateIntPathParam(clientIDParam, "client_id")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
			return
		}
		clientID = &id
	}

	available := make([]map[string]interface{}, 0, len(allStrategies))
	for _, id := range sortedStrategyIDs(allStrategies) {
		strategy := allStrategies[id]
		available = append(available, map[string]interface{}{
			"id":          strategy.ID,
			"name":        strategy.Name,
			"description": strategy.Description,
			"max_depth":   strategy.MaxDepth,
			"priority":    strategy.Priority,
			"rules":       strategy.Rules,
			"source":      "global",
		})
	}

	if clientID != nil && h.classificationService != nil {
		clientStrategies, err := h.classificationService.GetFoldingStrategies(clientID)
		if err != nil {
			if h.logFunc != nil {
				h.logFunc(LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Failed to get client strategies: %v", err),
					Endpoint:  r.URL.Path,
				})
			}
		} else {
			for _, clientStrategy := range clientStrategies {
				cfg, err := convertDBStrategyToConfig(clientStrategy)
				if err != nil {
					if h.logFunc != nil {
						h.logFunc(LogEntry{
							Timestamp: time.Now(),
							Level:     "ERROR",
							Message:   fmt.Sprintf("Failed to parse strategy config (ID %d): %v", clientStrategy.ID, err),
							Endpoint:  r.URL.Path,
						})
					}
					continue
				}

				entry := map[string]interface{}{
					"id":                 cfg.ID,
					"name":               cfg.Name,
					"description":        cfg.Description,
					"max_depth":          cfg.MaxDepth,
					"priority":           cfg.Priority,
					"rules":              cfg.Rules,
					"source":             "client",
					"client_id":          clientID,
					"is_default":         clientStrategy.IsDefault,
					"strategy_record_id": clientStrategy.ID,
				}
				available = append(available, entry)
			}
		}
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"strategies":    available,
		"total_count":   len(available),
		"client_filter": clientIDParam,
	}, http.StatusOK)
}

// HandleGetClassifiers обрабатывает запросы к /api/classification/classifiers
func (h *ClassificationHandler) HandleGetClassifiers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	clientIDParam := strings.TrimSpace(r.URL.Query().Get("client_id"))
	projectIDParam := strings.TrimSpace(r.URL.Query().Get("project_id"))
	activeOnlyParam := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("active_only")))

	var clientID *int
	if clientIDParam != "" {
		id, err := ValidateIntPathParam(clientIDParam, "client_id")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
			return
		}
		clientID = &id
	}

	var projectID *int
	if projectIDParam != "" {
		id, err := ValidateIntPathParam(projectIDParam, "project_id")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid project_id: %v", err), http.StatusBadRequest)
			return
		}
		projectID = &id
	}

	activeOnly := false
	if activeOnlyParam == "true" || activeOnlyParam == "1" || activeOnlyParam == "yes" {
		activeOnly = true
	}

	classifiers, err := h.classificationService.GetClassifiers(clientID, projectID, activeOnly)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get classifiers: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, "Failed to get classifiers", http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, classifiers, http.StatusOK)
}

// HandleGetClassifiersByProjectType обрабатывает запросы к /api/classification/classifiers/by-project-type
func (h *ClassificationHandler) HandleGetClassifiersByProjectType(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	projectType := r.URL.Query().Get("project_type")
	if projectType == "" {
		h.WriteJSONError(w, r, "project_type parameter is required", http.StatusBadRequest)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	classifiers, err := h.classificationService.GetClassifiersByProjectType(projectType)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get classifiers by project type: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, "Failed to get classifiers by project type", http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, classifiers, http.StatusOK)
}

// HandleClassificationOptimizationStats обрабатывает запросы к /api/classification/optimization-stats
func (h *ClassificationHandler) HandleClassificationOptimizationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	stats := h.classificationService.GetOptimizationStats()
	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleKpvedLoadFromFile обрабатывает запросы к /api/kpved/load-from-file
func (h *ClassificationHandler) HandleKpvedLoadFromFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	req.FilePath = strings.TrimSpace(req.FilePath)
	if req.FilePath == "" {
		h.WriteJSONError(w, r, "file_path is required", http.StatusBadRequest)
		return
	}

	totalCodes, err := h.classificationService.LoadKpvedFromFile(req.FilePath)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to load KPVED classifier")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":     true,
		"message":     "Классификатор КПВЭД успешно загружен",
		"file_path":   req.FilePath,
		"total_codes": totalCodes,
	}, http.StatusOK)
}

// HandleOkpd2Hierarchy обрабатывает запросы к /api/okpd2/hierarchy
func (h *ClassificationHandler) HandleOkpd2Hierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	parentCode := strings.TrimSpace(r.URL.Query().Get("parent"))
	levelParam := strings.TrimSpace(r.URL.Query().Get("level"))

	var level *int
	if levelParam != "" {
		parsedLevel, err := strconv.Atoi(levelParam)
		if err != nil {
			h.WriteJSONError(w, r, "Invalid level parameter", http.StatusBadRequest)
			return
		}
		level = &parsedLevel
	}

	result, err := h.classificationService.GetOkpd2Hierarchy(parentCode, level)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to fetch OKPD2 hierarchy")
		return
	}

	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleOkpd2Search обрабатывает запросы к /api/okpd2/search
func (h *ClassificationHandler) HandleOkpd2Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		h.WriteJSONError(w, r, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limit, err := ValidateIntParam(r, "limit", 50, 1, 200)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid limit parameter: %v", err), http.StatusBadRequest)
		return
	}

	results, err := h.classificationService.SearchOkpd2(query, limit)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to search OKPD2 classifier")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"results": results,
		"total":   len(results),
	}, http.StatusOK)
}

// HandleOkpd2Stats обрабатывает запросы к /api/okpd2/stats
func (h *ClassificationHandler) HandleOkpd2Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	stats, err := h.classificationService.GetOkpd2Stats()
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to get OKPD2 stats")
		return
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleOkpd2LoadFromFile обрабатывает запросы к /api/okpd2/load-from-file
func (h *ClassificationHandler) HandleOkpd2LoadFromFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var filePath string
	var originalName string
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))

	if strings.HasPrefix(contentType, "application/json") || strings.HasPrefix(contentType, "text/json") {
		var req struct {
			FilePath string `json:"file_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.FilePath) == "" {
			h.WriteJSONError(w, r, "file_path is required", http.StatusBadRequest)
			return
		}

		filePath = req.FilePath
		originalName = filepath.Base(req.FilePath)
	} else {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to parse form data: %v", err), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to read uploaded file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		tempFile, err := os.CreateTemp("", "okpd2-*")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to create temp file: %v", err), http.StatusInternalServerError)
			return
		}
		tempFilePath := tempFile.Name()
		defer func() {
			tempFile.Close()
			_ = os.Remove(tempFilePath)
		}()

		if _, err := io.Copy(tempFile, file); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to store uploaded file: %v", err), http.StatusInternalServerError)
			return
		}

		filePath = tempFilePath
		originalName = header.Filename
	}

	totalCodes, err := h.classificationService.LoadOkpd2FromFile(filePath)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to load OKPD2 from file")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":     true,
		"message":     "Классификатор ОКПД2 успешно загружен",
		"filename":    originalName,
		"total_codes": totalCodes,
	}, http.StatusOK)
}

// HandleOkpd2Clear обрабатывает запросы к /api/okpd2/clear
func (h *ClassificationHandler) HandleOkpd2Clear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	deletedCount, tableExists, err := h.classificationService.ClearOkpd2()
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to clear OKPD2 classifier")
		return
	}

	message := "Классификатор ОКПД2 успешно очищен"
	if !tableExists {
		message = "Классификатор ОКПД2 уже пуст"
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":       true,
		"message":       message,
		"deleted_count": deletedCount,
	}, http.StatusOK)
}

// HandleKpvedReclassify обрабатывает запросы к /api/kpved/reclassify
func (h *ClassificationHandler) HandleKpvedReclassify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		Limit int `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	classified, failed, results, err := h.classificationService.ReclassifyKpvedGroups(req.Limit)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to reclassify groups")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":    true,
		"classified": classified,
		"failed":     failed,
		"results":    results,
	}, http.StatusOK)
}

// HandleKpvedReclassifyHierarchical обрабатывает запросы к /api/kpved/reclassify-hierarchical
func (h *ClassificationHandler) HandleKpvedReclassifyHierarchical(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		NormalizedName string `json:"normalized_name"`
		Category       string `json:"category"`
		Model          string `json:"model,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.NormalizedName) == "" {
		h.WriteJSONError(w, r, "normalized_name is required", http.StatusBadRequest)
		return
	}

	result, err := h.classificationService.ReclassifyKpvedHierarchical(req.NormalizedName, req.Category, req.Model)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to perform hierarchical reclassification")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"result":  result,
	}, http.StatusOK)
}

// HandleKpvedCurrentTasks обрабатывает запросы к /api/kpved/current-tasks
func (h *ClassificationHandler) HandleKpvedCurrentTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	limit, err := ValidateIntParam(r, "limit", 20, 1, 1000)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid limit parameter: %v", err), http.StatusBadRequest)
		return
	}

	tasks, err := h.classificationService.GetKpvedCurrentTasks(limit)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to fetch current tasks")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"current_tasks": tasks,
		"count":         len(tasks),
		"limit":         limit,
	}, http.StatusOK)
}

// HandleResetAllClassification обрабатывает запросы к /api/kpved/reset-all
func (h *ClassificationHandler) HandleResetAllClassification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	rows, err := h.classificationService.ResetClassification("", "", "", 0, true)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to reset all classifications")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":       true,
		"message":       "Вся классификация сброшена",
		"rows_affected": rows,
	}, http.StatusOK)
}

// HandleResetByCode обрабатывает запросы к /api/kpved/reset-by-code
func (h *ClassificationHandler) HandleResetByCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		KpvedCode string `json:"kpved_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	req.KpvedCode = strings.TrimSpace(req.KpvedCode)
	if req.KpvedCode == "" {
		h.WriteJSONError(w, r, "kpved_code is required", http.StatusBadRequest)
		return
	}

	rows, err := h.classificationService.ResetClassification("", "", req.KpvedCode, 0, false)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to reset classification by code")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":       true,
		"message":       fmt.Sprintf("Классификация с кодом %s сброшена", req.KpvedCode),
		"rows_affected": rows,
		"kpved_code":    req.KpvedCode,
	}, http.StatusOK)
}

// HandleResetLowConfidence обрабатывает запросы к /api/kpved/reset-low-confidence
func (h *ClassificationHandler) HandleResetLowConfidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		MaxConfidence float64 `json:"max_confidence,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	maxConfidence := req.MaxConfidence
	if maxConfidence <= 0 {
		maxConfidence = 0.7
	}

	rows, err := h.classificationService.ResetClassification("", "", "", maxConfidence, false)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to reset low-confidence classifications")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":         true,
		"message":         fmt.Sprintf("Классификация с уверенностью ниже %.2f сброшена", maxConfidence),
		"rows_affected":   rows,
		"max_confidence":  maxConfidence,
		"default_applied": req.MaxConfidence <= 0,
	}, http.StatusOK)
}

// HandleKpvedWorkersStatus обрабатывает запросы к /api/kpved/workers/status
func (h *ClassificationHandler) HandleKpvedWorkersStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	limit, err := ValidateIntParam(r, "limit", 20, 1, 1000)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid limit parameter: %v", err), http.StatusBadRequest)
		return
	}

	status, err := h.classificationService.GetKpvedWorkersStatus(limit)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to get workers status")
		return
	}

	h.WriteJSONResponse(w, r, status, http.StatusOK)
}

// HandleKpvedWorkersStop обрабатывает запросы к /api/kpved/workers/stop
func (h *ClassificationHandler) HandleKpvedWorkersStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	status, err := h.classificationService.StopKpvedWorkers()
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to stop workers")
		return
	}

	h.WriteJSONResponse(w, r, status, http.StatusOK)
}

// HandleKpvedWorkersResume обрабатывает запросы к /api/kpved/workers/resume
func (h *ClassificationHandler) HandleKpvedWorkersResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	status, err := h.classificationService.ResumeKpvedWorkers()
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to resume workers")
		return
	}

	h.WriteJSONResponse(w, r, status, http.StatusOK)
}

// HandleKpvedStatsGeneral обрабатывает запросы к /api/kpved/stats/classification
func (h *ClassificationHandler) HandleKpvedStatsGeneral(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	stats, err := h.classificationService.GetKpvedStatsGeneral()
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to get KPVED statistics")
		return
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleKpvedStatsByCategory обрабатывает запросы к /api/kpved/stats/by-category
func (h *ClassificationHandler) HandleKpvedStatsByCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	stats, err := h.classificationService.GetKpvedStatsByCategory()
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to get category statistics")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"by_category": stats,
	}, http.StatusOK)
}

// HandleKpvedStatsIncorrect обрабатывает запросы к /api/kpved/stats/incorrect
func (h *ClassificationHandler) HandleKpvedStatsIncorrect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	limit, err := ValidateIntParam(r, "limit", 100, 1, 1000)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid limit parameter: %v", err), http.StatusBadRequest)
		return
	}

	items, total, err := h.classificationService.GetIncorrectKpvedClassifications(limit)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to get incorrect classifications")
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"total":           total,
		"shown":           len(items),
		"incorrect_items": items,
	}, http.StatusOK)
}

// HandleModelsBenchmark обрабатывает запросы к /api/models/benchmark
func (h *ClassificationHandler) HandleModelsBenchmark(w http.ResponseWriter, r *http.Request) {
	if h.classificationService == nil {
		h.WriteJSONError(w, r, "Classification service not available", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleModelsBenchmarkGet(w, r)
	case http.MethodPost:
		h.WriteJSONError(w, r, "Benchmark execution endpoint is not implemented yet", http.StatusNotImplemented)
	default:
		h.BaseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

func (h *ClassificationHandler) handleModelsBenchmarkGet(w http.ResponseWriter, r *http.Request) {
	limitParam := strings.TrimSpace(r.URL.Query().Get("limit"))
	model := strings.TrimSpace(r.URL.Query().Get("model"))
	historyFlag := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("history")), "true")

	limit := 100
	if limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			limit = parsed
		} else {
			h.WriteJSONError(w, r, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
	}

	data, err := h.classificationService.GetModelsBenchmarkHistory(limit, model, historyFlag)
	if err != nil {
		h.handleClassificationError(w, r, err, "Failed to get benchmark history")
		return
	}

	h.WriteJSONResponse(w, r, data, http.StatusOK)
}
func convertDBStrategyToConfig(strategy *database.FoldingStrategy) (classification.FoldingStrategyConfig, error) {
	if strategy == nil {
		return classification.FoldingStrategyConfig{}, fmt.Errorf("strategy is nil")
	}

	var cfg classification.FoldingStrategyConfig
	if strings.TrimSpace(strategy.StrategyConfig) != "" {
		if err := json.Unmarshal([]byte(strategy.StrategyConfig), &cfg); err != nil {
			return cfg, err
		}
	}

	if cfg.ID == "" {
		cfg.ID = fmt.Sprintf("client_strategy_%d", strategy.ID)
	}
	if cfg.Name == "" {
		cfg.Name = strategy.Name
	}
	if cfg.Description == "" {
		cfg.Description = strategy.Description
	}
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = 2
	}

	return cfg, nil
}
