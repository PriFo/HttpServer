package database

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	databaseapp "httpserver/internal/application/database"
	"httpserver/internal/domain/repositories"
	"httpserver/internal/api/handlers/common"
)

// Handler HTTP обработчик для работы с базами данных
type Handler struct {
	baseHandler common.BaseHandlerInterface
	useCase     *databaseapp.UseCase
}

// NewHandler создает новый HTTP обработчик для баз данных
func NewHandler(
	baseHandler common.BaseHandlerInterface,
	useCase *databaseapp.UseCase,
) *Handler {
	return &Handler{
		baseHandler: baseHandler,
		useCase:     useCase,
	}
}

// HandleDatabases обрабатывает запросы к /api/v2/databases
func (h *Handler) HandleDatabases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetDatabases(w, r)
	case http.MethodPost:
		h.CreateDatabase(w, r)
	default:
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// GetDatabases возвращает список баз данных
// GET /api/v2/databases
func (h *Handler) GetDatabases(w http.ResponseWriter, r *http.Request) {
	filter := repositories.DatabaseFilter{
		Limit:  100,
		Offset: 0,
	}

	// Парсим query параметры
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = name
	}

	if dbType := r.URL.Query().Get("type"); dbType != "" {
		filter.Type = dbType
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []string{status}
	}

	if path := r.URL.Query().Get("path"); path != "" {
		filter.Path = path
	}

	databases, total, err := h.useCase.ListDatabases(r.Context(), filter)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"databases": databases,
		"total":     total,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	}, http.StatusOK)
}

// CreateDatabase создает новую базу данных
// POST /api/v2/databases
func (h *Handler) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	var req databaseapp.CreateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	database, err := h.useCase.CreateDatabase(r.Context(), req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, database, http.StatusCreated)
}

// HandleDatabaseRoutes обрабатывает запросы к /api/v2/databases/{id}
func (h *Handler) HandleDatabaseRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/databases/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "database ID required"}, http.StatusBadRequest)
		return
	}

	databaseID := parts[0]

	// Обработка вложенных маршрутов
	if len(parts) > 1 {
		switch parts[1] {
		case "test-connection":
			if r.Method == http.MethodPost {
				h.TestConnection(w, r, databaseID)
				return
			}
		case "status":
			if r.Method == http.MethodGet {
				h.GetConnectionStatus(w, r, databaseID)
				return
			}
		case "statistics":
			if r.Method == http.MethodGet {
				h.GetDatabaseStatistics(w, r, databaseID)
				return
			}
		}
	}

	// Обработка основных операций с базой данных
	switch r.Method {
	case http.MethodGet:
		h.GetDatabase(w, r, databaseID)
	case http.MethodPut:
		h.UpdateDatabase(w, r, databaseID)
	case http.MethodDelete:
		h.DeleteDatabase(w, r, databaseID)
	default:
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// GetDatabase возвращает базу данных по ID
// GET /api/v2/databases/{id}
func (h *Handler) GetDatabase(w http.ResponseWriter, r *http.Request, databaseID string) {
	database, err := h.useCase.GetDatabase(r.Context(), databaseID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusNotFound)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, database, http.StatusOK)
}

// UpdateDatabase обновляет базу данных
// PUT /api/v2/databases/{id}
func (h *Handler) UpdateDatabase(w http.ResponseWriter, r *http.Request, databaseID string) {
	var req databaseapp.UpdateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	database, err := h.useCase.UpdateDatabase(r.Context(), databaseID, req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, database, http.StatusOK)
}

// DeleteDatabase удаляет базу данных
// DELETE /api/v2/databases/{id}
func (h *Handler) DeleteDatabase(w http.ResponseWriter, r *http.Request, databaseID string) {
	if err := h.useCase.DeleteDatabase(r.Context(), databaseID); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"message": "database deleted"}, http.StatusOK)
}

// TestConnection проверяет подключение к базе данных
// POST /api/v2/databases/{id}/test-connection
func (h *Handler) TestConnection(w http.ResponseWriter, r *http.Request, databaseID string) {
	if err := h.useCase.TestConnection(r.Context(), databaseID); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"message": "connection successful"}, http.StatusOK)
}

// GetConnectionStatus возвращает статус подключения
// GET /api/v2/databases/{id}/status
func (h *Handler) GetConnectionStatus(w http.ResponseWriter, r *http.Request, databaseID string) {
	status, err := h.useCase.GetConnectionStatus(r.Context(), databaseID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": status}, http.StatusOK)
}

// GetDatabaseStatistics возвращает статистику базы данных
// GET /api/v2/databases/{id}/statistics
func (h *Handler) GetDatabaseStatistics(w http.ResponseWriter, r *http.Request, databaseID string) {
	stats, err := h.useCase.GetDatabaseStatistics(r.Context(), databaseID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, stats, http.StatusOK)
}

