package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"httpserver/server/services"
)

// NormalizationBenchmarkHandler обработчик для бенчмарков нормализации
type NormalizationBenchmarkHandler struct {
	normalizationBenchmarkService *services.NormalizationBenchmarkService
	baseHandler                   *BaseHandler
}

// NewNormalizationBenchmarkHandler создает новый обработчик для бенчмарков нормализации
func NewNormalizationBenchmarkHandler(
	normalizationBenchmarkService *services.NormalizationBenchmarkService,
	baseHandler *BaseHandler,
) *NormalizationBenchmarkHandler {
	return &NormalizationBenchmarkHandler{
		normalizationBenchmarkService: normalizationBenchmarkService,
		baseHandler:                   baseHandler,
	}
}

// HandleUploadBenchmark обрабатывает запросы к /api/normalization/benchmark/upload
func (h *NormalizationBenchmarkHandler) HandleUploadBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var report services.NormalizationBenchmarkReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.normalizationBenchmarkService.UploadBenchmark(report)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to upload benchmark: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleListBenchmarks обрабатывает запросы к /api/normalization/benchmark/list
func (h *NormalizationBenchmarkHandler) HandleListBenchmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	result, err := h.normalizationBenchmarkService.ListBenchmarks()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to list benchmarks: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetBenchmark обрабатывает запросы к /api/normalization/benchmark/{id}
func (h *NormalizationBenchmarkHandler) HandleGetBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/normalization/benchmark/")
	pathParts := strings.Split(path, "/")
	var id string
	for i, part := range pathParts {
		if part == "benchmark" && i+1 < len(pathParts) {
			id = pathParts[i+1]
			break
		}
	}

	// Если не нашли в пути, пробуем из query параметра
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	if id == "" {
		h.baseHandler.WriteJSONError(w, r, "Benchmark ID not provided", http.StatusBadRequest)
		return
	}

	report, err := h.normalizationBenchmarkService.GetBenchmark(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.baseHandler.WriteJSONError(w, r, "Benchmark not found", http.StatusNotFound)
		} else {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get benchmark: %v", err), http.StatusInternalServerError)
		}
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, report, http.StatusOK)
}

