package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"httpserver/websearch"
)

// WebSearchValidationHandler обработчик для валидации через веб-поиск
// Поддерживает как *websearch.Client, так и *websearch.MultiProviderClient
type WebSearchValidationHandler struct {
	*BaseHandler
	client              websearch.SearchClientInterface
	existenceValidator  *websearch.ProductExistenceValidator
	accuracyValidator   *websearch.ProductAccuracyValidator
}

// NewWebSearchValidationHandler создает новый обработчик валидации
// Принимает как *websearch.Client, так и *websearch.MultiProviderClient
func NewWebSearchValidationHandler(baseHandler *BaseHandler, client websearch.SearchClientInterface) *WebSearchValidationHandler {
	var existenceValidator *websearch.ProductExistenceValidator
	var accuracyValidator *websearch.ProductAccuracyValidator

	if client != nil {
		existenceValidator = websearch.NewProductExistenceValidator(client)
		accuracyValidator = websearch.NewProductAccuracyValidator(client)
	}

	return &WebSearchValidationHandler{
		BaseHandler:        baseHandler,
		client:             client,
		existenceValidator: existenceValidator,
		accuracyValidator:  accuracyValidator,
	}
}

// HandleValidateProduct обрабатывает валидацию товара
// POST /api/validation/websearch
func (h *WebSearchValidationHandler) HandleValidateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.client == nil || h.existenceValidator == nil {
		http.Error(w, "Web search is not configured", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		Name     string `json:"name"`
		Code     string `json:"code,omitempty"`
		Type     string `json:"type,omitempty"` // "existence" or "accuracy"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if request.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var validation *websearch.ValidationResult
	var err error

	if request.Type == "accuracy" && request.Code != "" {
		// Проверка точности данных
		validation, err = h.accuracyValidator.Validate(ctx, request.Name, request.Code)
	} else {
		// Проверка существования
		validation, err = h.existenceValidator.Validate(ctx, request.Name)
	}

	if err != nil {
		h.BaseHandler.WriteJSONError(w, r, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"validation": validation,
	}

	h.BaseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleSearch выполняет простой поиск
// GET /api/validation/websearch/search?query=...
func (h *WebSearchValidationHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.client == nil {
		http.Error(w, "Web search is not configured", http.StatusServiceUnavailable)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.client.Search(ctx, query)
	if err != nil {
		h.BaseHandler.WriteJSONError(w, r, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"query":   result.Query,
		"found":   result.Found,
		"results": result.Results,
		"source":  result.Source,
		"confidence": result.Confidence,
	}

	h.BaseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

