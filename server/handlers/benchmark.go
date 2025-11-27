package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"httpserver/server/models"
	"httpserver/server/services"
)

// BenchmarkHandler обработчик для работы с эталонами
type BenchmarkHandler struct {
	benchmarkService *services.BenchmarkService
	baseHandler      *BaseHandler
}

// NewBenchmarkHandler создает новый обработчик для работы с эталонами
func NewBenchmarkHandler(
	benchmarkService *services.BenchmarkService,
	baseHandler *BaseHandler,
) *BenchmarkHandler {
	return &BenchmarkHandler{
		benchmarkService: benchmarkService,
		baseHandler:      baseHandler,
	}
}

// HandleImportManufacturers обрабатывает запросы к /api/benchmarks/import-manufacturers
// Это устаревший метод, оставлен для обратной совместимости
// Используется старая реализация из server.handleImportManufacturers
func (h *BenchmarkHandler) HandleImportManufacturers(w http.ResponseWriter, r *http.Request) {
	// Этот метод не реализован в новом BenchmarkService
	// Используется fallback к старой реализации в server.go
	h.baseHandler.WriteJSONError(w, r, "This endpoint is handled by legacy implementation", http.StatusNotImplemented)
}

// CreateFromUpload обрабатывает POST /api/benchmarks/from-upload
func (h *BenchmarkHandler) CreateFromUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateBenchmarkFromUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if req.UploadID == "" {
		h.baseHandler.WriteJSONError(w, r, "upload_id is required", http.StatusBadRequest)
		return
	}
	if len(req.ItemIDs) == 0 {
		h.baseHandler.WriteJSONError(w, r, "item_ids is required and must not be empty", http.StatusBadRequest)
		return
	}
	if req.EntityType != "counterparty" && req.EntityType != "nomenclature" {
		h.baseHandler.WriteJSONError(w, r, "entity_type must be 'counterparty' or 'nomenclature'", http.StatusBadRequest)
		return
	}

	benchmark, err := h.benchmarkService.CreateFromUpload(req.UploadID, req.ItemIDs, req.EntityType)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, benchmark, http.StatusCreated)
}

// Search обрабатывает GET /api/benchmarks/search
func (h *BenchmarkHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	entityType := r.URL.Query().Get("type")

	if name == "" {
		h.baseHandler.WriteJSONError(w, r, "name parameter is required", http.StatusBadRequest)
		return
	}
	if entityType == "" {
		h.baseHandler.WriteJSONError(w, r, "type parameter is required", http.StatusBadRequest)
		return
	}
	if entityType != "counterparty" && entityType != "nomenclature" {
		h.baseHandler.WriteJSONError(w, r, "type must be 'counterparty' or 'nomenclature'", http.StatusBadRequest)
		return
	}

	benchmark, err := h.benchmarkService.FindBestMatch(name, entityType)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	if benchmark == nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{"found": false}, http.StatusOK)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{"found": true, "benchmark": benchmark}, http.StatusOK)
}

// List обрабатывает GET /api/benchmarks
func (h *BenchmarkHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entityType := r.URL.Query().Get("type")
	activeOnlyStr := r.URL.Query().Get("active")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	activeOnly := true
	if activeOnlyStr != "" {
		activeOnly = strings.ToLower(activeOnlyStr) == "true" || activeOnlyStr == "1"
	}

	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	response, err := h.benchmarkService.List(entityType, activeOnly, limit, offset)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetByID обрабатывает GET /api/benchmarks/:id
func (h *BenchmarkHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	if path == "" {
		h.baseHandler.WriteJSONError(w, r, "benchmark ID is required", http.StatusBadRequest)
		return
	}

	benchmark, err := h.benchmarkService.GetByID(path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.baseHandler.WriteJSONError(w, r, "Benchmark not found", http.StatusNotFound)
			return
		}
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, benchmark, http.StatusOK)
}

// Update обрабатывает PUT /api/benchmarks/:id
func (h *BenchmarkHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	if path == "" {
		h.baseHandler.WriteJSONError(w, r, "benchmark ID is required", http.StatusBadRequest)
		return
	}

	var req models.UpdateBenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем существующий эталон
	benchmark, err := h.benchmarkService.GetByID(path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.baseHandler.WriteJSONError(w, r, "Benchmark not found", http.StatusNotFound)
			return
		}
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	// Обновляем поля из запроса
	if req.EntityType != "" {
		benchmark.EntityType = req.EntityType
	}
	if req.Name != "" {
		benchmark.Name = req.Name
	}
	if req.Data != nil {
		benchmark.Data = req.Data
	}
	if req.IsActive != nil {
		benchmark.IsActive = *req.IsActive
	}
	if req.Variations != nil {
		benchmark.Variations = req.Variations
	}

	// Сохраняем обновленный эталон
	if err := h.benchmarkService.Update(benchmark); err != nil {
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, benchmark, http.StatusOK)
}

// Delete обрабатывает DELETE /api/benchmarks/:id
func (h *BenchmarkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	if path == "" {
		h.baseHandler.WriteJSONError(w, r, "benchmark ID is required", http.StatusBadRequest)
		return
	}

	if err := h.benchmarkService.Delete(path); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.baseHandler.WriteJSONError(w, r, "Benchmark not found", http.StatusNotFound)
			return
		}
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{"message": "Benchmark deleted successfully"}, http.StatusOK)
}

// Create обрабатывает POST /api/benchmarks
func (h *BenchmarkHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateBenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if req.EntityType == "" {
		h.baseHandler.WriteJSONError(w, r, "entity_type is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		h.baseHandler.WriteJSONError(w, r, "name is required", http.StatusBadRequest)
		return
	}
	if req.EntityType != "counterparty" && req.EntityType != "nomenclature" {
		h.baseHandler.WriteJSONError(w, r, "entity_type must be 'counterparty' or 'nomenclature'", http.StatusBadRequest)
		return
	}

	benchmark, err := h.benchmarkService.Create(&req)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, err)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, benchmark, http.StatusCreated)
}
