package classification

import (
	"encoding/json"
	"net/http"
	"strings"

	classificationapp "httpserver/internal/application/classification"
	"httpserver/internal/api/handlers/common"
)

// Handler HTTP обработчик для работы с классификацией
type Handler struct {
	baseHandler common.BaseHandlerInterface
	useCase     *classificationapp.UseCase
}

// NewHandler создает новый HTTP обработчик для классификации
func NewHandler(
	baseHandler common.BaseHandlerInterface,
	useCase *classificationapp.UseCase,
) *Handler {
	return &Handler{
		baseHandler: baseHandler,
		useCase:     useCase,
	}
}

// NewHandlerWithDefaults создает handler с дефолтным baseHandler
func NewHandlerWithDefaults(useCase *classificationapp.UseCase) *Handler {
	return &Handler{
		baseHandler: common.NewBaseHandlerImpl(),
		useCase:     useCase,
	}
}

// HandleClassifyEntity классифицирует сущность
// POST /api/classification/classify
func (h *Handler) HandleClassifyEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EntityID   string `json:"entity_id"`
		EntityType string `json:"entity_type"`
		Category   string `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.ClassifyEntity(r.Context(), req.EntityID, req.EntityType, req.Category)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetClassification возвращает классификацию по ID
// GET /api/classification/:id
func (h *Handler) HandleGetClassification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	classificationID := r.URL.Query().Get("id")
	if classificationID == "" {
		classificationID = extractIDFromPath(r.URL.Path)
	}

	if classificationID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "classification id is required"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.GetClassification(r.Context(), classificationID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetClassificationByEntity возвращает классификацию по ID сущности
// GET /api/classification/entity/:entity_id
func (h *Handler) HandleGetClassificationByEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	entityID := r.URL.Query().Get("entity_id")
	if entityID == "" {
		entityID = extractEntityIDFromPath(r.URL.Path)
	}

	if entityID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "entity id is required"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.GetClassificationByEntity(r.Context(), entityID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetClassificationHistory возвращает историю классификаций
// GET /api/classification/history?entity_id=xxx
func (h *Handler) HandleGetClassificationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	entityID := r.URL.Query().Get("entity_id")
	if entityID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "entity_id is required"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.GetClassificationHistory(r.Context(), entityID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetClassificationStatistics возвращает статистику классификации
// GET /api/classification/stats
func (h *Handler) HandleGetClassificationStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	result, err := h.useCase.GetClassificationStatistics(r.Context())
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleClassifyHierarchical выполняет иерархическую классификацию
// POST /api/classification/hierarchical
func (h *Handler) HandleClassifyHierarchical(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EntityID   string `json:"entity_id"`
		EntityType string `json:"entity_type"`
		Category   string `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.ClassifyHierarchical(r.Context(), req.EntityID, req.EntityType, req.Category)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// Вспомогательные функции для извлечения параметров из пути

func extractIDFromPath(path string) string {
	// Извлекает ID из пути типа /api/classification/123
	parts := strings.Split(path, "/")
	if len(parts) >= 4 && parts[len(parts)-2] == "classification" {
		return parts[len(parts)-1]
	}
	return ""
}

func extractEntityIDFromPath(path string) string {
	// Извлекает entity_id из пути типа /api/classification/entity/123
	parts := strings.Split(path, "/")
	if len(parts) >= 5 && parts[len(parts)-3] == "classification" && parts[len(parts)-2] == "entity" {
		return parts[len(parts)-1]
	}
	return ""
}

