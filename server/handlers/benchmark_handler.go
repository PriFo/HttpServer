package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"httpserver/server/models"
	"httpserver/server/services"
)

// BenchmarkAPIHandler обработчик для работы с эталонами
type BenchmarkAPIHandler struct {
	service *services.BenchmarkService
}

// NewBenchmarkAPIHandler создает новый обработчик для работы с эталонами
func NewBenchmarkAPIHandler(service *services.BenchmarkService) *BenchmarkAPIHandler {
	return &BenchmarkAPIHandler{
		service: service,
	}
}

// @Summary Create benchmark from upload
// @Description Creates a new benchmark from selected items in an upload
// @Tags benchmarks
// @Accept json
// @Produce json
// @Param request body models.CreateBenchmarkFromUploadRequest true "Benchmark creation request"
// @Success 201 {object} models.Benchmark "Benchmark created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 404 {object} ErrorResponse "Upload not found"
// @Failure 500 {object} ErrorResponse "Failed to create benchmark"
// @Router /api/benchmarks/from-upload [post]
// CreateFromUpload обрабатывает POST /api/benchmarks/from-upload
func (h *BenchmarkAPIHandler) CreateFromUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateBenchmarkFromUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if req.UploadID == "" {
		h.WriteJSONError(w, r, "upload_id is required", http.StatusBadRequest)
		return
	}
	if len(req.ItemIDs) == 0 {
		h.WriteJSONError(w, r, "item_ids is required and must not be empty", http.StatusBadRequest)
		return
	}
	if req.EntityType == "" {
		h.WriteJSONError(w, r, "entity_type is required", http.StatusBadRequest)
		return
	}

	benchmark, err := h.service.CreateFromUpload(req.UploadID, req.ItemIDs, req.EntityType)
	if err != nil {
		h.HandleHTTPError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, benchmark, http.StatusCreated)
}

// @Summary Search for best match benchmark
// @Description Finds the best matching benchmark for a given name and entity type
// @Tags benchmarks
// @Accept json
// @Produce json
// @Param name query string true "Name to search for"
// @Param type query string true "Entity type (e.g., counterparty, nomenclature)"
// @Success 200 {object} models.Benchmark "Best matching benchmark"
// @Failure 400 {object} ErrorResponse "Missing required parameter"
// @Failure 404 {object} ErrorResponse "Benchmark not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/benchmarks/search [get]
// Search обрабатывает GET /api/benchmarks/search
func (h *BenchmarkAPIHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	entityType := r.URL.Query().Get("type")

	if name == "" {
		h.WriteJSONError(w, r, "name parameter is required", http.StatusBadRequest)
		return
	}
	if entityType == "" {
		h.WriteJSONError(w, r, "type parameter is required", http.StatusBadRequest)
		return
	}

	benchmark, err := h.service.FindBestMatch(name, entityType)
	if err != nil {
		h.HandleHTTPError(w, r, err)
		return
	}

	if benchmark == nil {
		h.WriteJSONError(w, r, "Benchmark not found", http.StatusNotFound)
		return
	}

	h.WriteJSONResponse(w, r, benchmark, http.StatusOK)
}

// @Summary List all benchmarks
// @Description Returns a list of all active benchmarks or filtered by type
// @Tags benchmarks
// @Accept json
// @Produce json
// @Param type query string false "Entity type to filter by"
// @Success 200 {array} models.Benchmark "List of benchmarks"
// @Failure 500 {object} ErrorResponse "Failed to get benchmarks"
// @Router /api/benchmarks [get]
// List обрабатывает GET /api/benchmarks
func (h *BenchmarkAPIHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entityType := r.URL.Query().Get("type")

	// Get all active benchmarks if no type provided
	if entityType == "" {
		// Call a method to get all active benchmarks
		// Since there's no direct method for this, we'll use List with empty type
		response, err := h.service.List("", true, 1000, 0)
		if err != nil {
			h.HandleHTTPError(w, r, err)
			return
		}
		h.WriteJSONResponse(w, r, response.Benchmarks, http.StatusOK)
		return
	}

	// Get benchmarks by type
	benchmarks, err := h.service.GetByType(entityType)
	if err != nil {
		h.HandleHTTPError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, benchmarks, http.StatusOK)
}

// @Summary Get benchmark by ID
// @Description Retrieves a specific benchmark by its ID
// @Tags benchmarks
// @Accept json
// @Produce json
// @Param id path string true "Benchmark ID"
// @Success 200 {object} models.Benchmark "Benchmark details"
// @Failure 400 {object} ErrorResponse "Invalid benchmark ID"
// @Failure 404 {object} ErrorResponse "Benchmark not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/benchmarks/{id} [get]
// GetByID обрабатывает GET /api/benchmarks/{id}
func (h *BenchmarkAPIHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	if path == "" {
		h.WriteJSONError(w, r, "benchmark ID is required", http.StatusBadRequest)
		return
	}

	benchmark, err := h.service.GetByID(path)
	if err != nil {
		h.HandleHTTPError(w, r, err)
		return
	}

	if benchmark == nil {
		h.WriteJSONError(w, r, "Benchmark not found", http.StatusNotFound)
		return
	}

	h.WriteJSONResponse(w, r, benchmark, http.StatusOK)
}

// @Summary Update benchmark
// @Description Updates an existing benchmark with new data
// @Tags benchmarks
// @Accept json
// @Produce json
// @Param id path string true "Benchmark ID"
// @Param request body models.UpdateBenchmarkRequest true "Benchmark update request"
// @Success 200 {object} models.Benchmark "Updated benchmark"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 404 {object} ErrorResponse "Benchmark not found"
// @Failure 500 {object} ErrorResponse "Failed to update benchmark"
// @Router /api/benchmarks/{id} [put]
// Update обрабатывает PUT /api/benchmarks/{id}
func (h *BenchmarkAPIHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	if path == "" {
		h.WriteJSONError(w, r, "benchmark ID is required", http.StatusBadRequest)
		return
	}

	var req models.UpdateBenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем существующий эталон
	benchmark, err := h.service.GetByID(path)
	if err != nil {
		h.HandleHTTPError(w, r, err)
		return
	}

	if benchmark == nil {
		h.WriteJSONError(w, r, "Benchmark not found", http.StatusNotFound)
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
	if err := h.service.Update(benchmark); err != nil {
		h.HandleHTTPError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, benchmark, http.StatusOK)
}

// @Summary Delete benchmark
// @Description Marks a benchmark as deleted (soft delete)
// @Tags benchmarks
// @Accept json
// @Produce json
// @Param id path string true "Benchmark ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} ErrorResponse "Invalid benchmark ID"
// @Failure 500 {object} ErrorResponse "Failed to delete benchmark"
// @Router /api/benchmarks/{id} [delete]
// Delete обрабатывает DELETE /api/benchmarks/{id}
func (h *BenchmarkAPIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmarks/")
	if path == "" {
		h.WriteJSONError(w, r, "benchmark ID is required", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(path); err != nil {
		h.HandleHTTPError(w, r, err)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{"message": "Benchmark deleted successfully"}, http.StatusOK)
}

// Helper methods for JSON responses and error handling
func (h *BenchmarkAPIHandler) WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *BenchmarkAPIHandler) WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *BenchmarkAPIHandler) HandleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	// You can implement more sophisticated error handling here
	// For now, just return a generic 500 error with the error message
	h.WriteJSONError(w, r, "Internal server error: "+err.Error(), http.StatusInternalServerError)
}
