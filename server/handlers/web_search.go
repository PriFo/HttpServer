package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"httpserver/websearch"
)

// WebSearchHandler обработчик для веб-поиска валидации
type WebSearchHandler struct {
	*BaseHandler
	client              websearch.SearchClientInterface
	existenceValidator  *websearch.ProductExistenceValidator
	accuracyValidator   *websearch.ProductAccuracyValidator
}

// NewWebSearchHandler создает новый обработчик веб-поиска
// Принимает как *websearch.Client, так и *websearch.MultiProviderClient
func NewWebSearchHandler(baseHandler *BaseHandler, client websearch.SearchClientInterface) *WebSearchHandler {
	var existenceValidator *websearch.ProductExistenceValidator
	var accuracyValidator *websearch.ProductAccuracyValidator
	
	if client != nil {
		existenceValidator = websearch.NewProductExistenceValidator(client)
		accuracyValidator = websearch.NewProductAccuracyValidator(client)
	}

	return &WebSearchHandler{
		BaseHandler:        baseHandler,
		client:            client,
		existenceValidator: existenceValidator,
		accuracyValidator:  accuracyValidator,
	}
}

// HandleWebSearch обрабатывает запрос веб-поиска для валидации
// GET /api/validation/web-search?query=...&item_id=...&item_name=...
func (h *WebSearchHandler) HandleWebSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.client == nil {
		http.Error(w, "Web search is not configured", http.StatusServiceUnavailable)
		return
	}

	// Получаем параметры
	query := r.URL.Query().Get("query")
	itemID := r.URL.Query().Get("item_id")
	itemName := r.URL.Query().Get("item_name")

	// Валидация параметров
	if query == "" && itemName == "" {
		http.Error(w, "query or item_name parameter is required", http.StatusBadRequest)
		return
	}

	// Формируем поисковый запрос
	searchQuery := query
	if searchQuery == "" {
		searchQuery = itemName
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Выполняем поиск
	result, err := h.client.Search(ctx, searchQuery)
	if err != nil {
		h.BaseHandler.WriteJSONError(w, r, fmt.Sprintf("Web search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Если указан item_id или item_name, выполняем валидацию
	var validation *websearch.ValidationResult
	if itemName != "" || itemID != "" {
		if h.existenceValidator != nil {
			validation, _ = h.existenceValidator.Validate(ctx, itemName)
		}
	}

	// Формируем ответ
	response := map[string]interface{}{
		"success": true,
		"found":   result.Found,
		"results": result.Results,
		"query":   result.Query,
		"source":  result.Source,
	}

	if validation != nil {
		response["validation"] = map[string]interface{}{
			"status":    validation.Status,
			"message":   validation.Message,
			"score":     validation.Score,
			"found":     validation.Found,
			"results":   validation.Results,
			"provider":  validation.Provider,
			"details":   validation.Details,
		}
	}

	h.BaseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleWebSearchBatch обрабатывает батч-валидацию через веб-поиск
// POST /api/validation/web-search/batch
func (h *WebSearchHandler) HandleWebSearchBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.client == nil || h.existenceValidator == nil || h.accuracyValidator == nil {
		http.Error(w, "Web search is not configured", http.StatusServiceUnavailable)
		return
	}

	// Парсим тело запроса
	var request struct {
		Items []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Code     string `json:"code"`
			Category string `json:"category,omitempty"`
		} `json:"items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(request.Items) == 0 {
		http.Error(w, "items array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	// Ограничиваем размер батча
	maxBatchSize := 10
	if len(request.Items) > maxBatchSize {
		http.Error(w, fmt.Sprintf("batch size exceeds maximum of %d", maxBatchSize), http.StatusBadRequest)
		return
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Обрабатываем каждый элемент
	results := make([]map[string]interface{}, 0, len(request.Items))
	for _, item := range request.Items {
		// Выполняем валидацию
		var validation *websearch.ValidationResult
		var err error

		if item.Code != "" {
			// Используем валидатор точности, если есть код
			validation, err = h.accuracyValidator.Validate(ctx, item.Name, item.Code)
		} else {
			// Используем валидатор существования, если есть только название
			validation, err = h.existenceValidator.Validate(ctx, item.Name)
		}

		result := map[string]interface{}{
			"id":      item.ID,
			"name":    item.Name,
			"code":    item.Code,
			"success": err == nil,
		}

		if err != nil {
			result["error"] = err.Error()
		}

		if validation != nil {
			result["validation"] = map[string]interface{}{
				"status":    validation.Status,
				"message":   validation.Message,
				"score":     validation.Score,
				"found":     validation.Found,
				"provider":  validation.Provider,
				"details":   validation.Details,
				"results":   validation.Results,
				"timestamp": validation.Timestamp,
			}
		}

		results = append(results, result)
	}

	// Формируем ответ
	response := map[string]interface{}{
		"success": true,
		"count":   len(results),
		"results": results,
	}

	h.BaseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}
