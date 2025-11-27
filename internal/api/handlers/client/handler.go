package client

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	clientapp "httpserver/internal/application/client"
	"httpserver/internal/domain/repositories"
	"httpserver/internal/api/handlers/common"
)

// Handler HTTP обработчик для работы с клиентами и проектами
type Handler struct {
	baseHandler common.BaseHandlerInterface
	useCase     *clientapp.UseCase
}

// NewHandler создает новый HTTP обработчик для клиентов
func NewHandler(
	baseHandler common.BaseHandlerInterface,
	useCase *clientapp.UseCase,
) *Handler {
	return &Handler{
		baseHandler: baseHandler,
		useCase:     useCase,
	}
}

// NewHandlerWithDefaults создает handler с дефолтным baseHandler
func NewHandlerWithDefaults(useCase *clientapp.UseCase) *Handler {
	return &Handler{
		baseHandler: common.NewBaseHandlerImpl(),
		useCase:     useCase,
	}
}

// HandleClients обрабатывает запросы к /api/v2/clients
func (h *Handler) HandleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetClients(w, r)
	case http.MethodPost:
		h.CreateClient(w, r)
	default:
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// GetClients возвращает список клиентов
// GET /api/v2/clients
func (h *Handler) GetClients(w http.ResponseWriter, r *http.Request) {
	filter := repositories.ClientFilter{
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

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []string{status}
	}

	clients, total, err := h.useCase.ListClients(r.Context(), filter)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"clients": clients,
		"total":   total,
		"limit":   filter.Limit,
		"offset":  filter.Offset,
	}, http.StatusOK)
}

// CreateClient создает нового клиента
// POST /api/v2/clients
func (h *Handler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req clientapp.CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	client, err := h.useCase.CreateClient(r.Context(), req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, client, http.StatusCreated)
}

// HandleClientRoutes обрабатывает запросы к /api/v2/clients/{id}
func (h *Handler) HandleClientRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/clients/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "client ID required"}, http.StatusBadRequest)
		return
	}

	clientID := parts[0]

	// Обработка вложенных маршрутов
	if len(parts) > 1 {
		switch parts[1] {
		case "statistics":
			if r.Method == http.MethodGet {
				h.GetClientStatistics(w, r, clientID)
				return
			}
		case "projects":
			if len(parts) == 2 {
				if r.Method == http.MethodGet {
					h.GetClientProjects(w, r, clientID)
				} else if r.Method == http.MethodPost {
					h.CreateClientProject(w, r, clientID)
				} else {
					h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
				}
				return
			} else if len(parts) == 3 {
				projectID := parts[2]
				if r.Method == http.MethodGet {
					h.GetProject(w, r, clientID, projectID)
				} else if r.Method == http.MethodPut {
					h.UpdateProject(w, r, clientID, projectID)
				} else if r.Method == http.MethodDelete {
					h.DeleteProject(w, r, clientID, projectID)
				} else {
					h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
				}
				return
			}
		case "databases":
			if r.Method == http.MethodGet {
				h.GetClientDatabases(w, r, clientID)
				return
			}
		}
	}

	// Обработка основных операций с клиентом
	switch r.Method {
	case http.MethodGet:
		h.GetClient(w, r, clientID)
	case http.MethodPut:
		h.UpdateClient(w, r, clientID)
	case http.MethodDelete:
		h.DeleteClient(w, r, clientID)
	default:
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// GetClient возвращает клиента по ID
// GET /api/v2/clients/{id}
func (h *Handler) GetClient(w http.ResponseWriter, r *http.Request, clientID string) {
	client, err := h.useCase.GetClient(r.Context(), clientID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusNotFound)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, client, http.StatusOK)
}

// UpdateClient обновляет клиента
// PUT /api/v2/clients/{id}
func (h *Handler) UpdateClient(w http.ResponseWriter, r *http.Request, clientID string) {
	var req clientapp.UpdateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	client, err := h.useCase.UpdateClient(r.Context(), clientID, req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, client, http.StatusOK)
}

// DeleteClient удаляет клиента
// DELETE /api/v2/clients/{id}
func (h *Handler) DeleteClient(w http.ResponseWriter, r *http.Request, clientID string) {
	if err := h.useCase.DeleteClient(r.Context(), clientID); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"message": "client deleted"}, http.StatusOK)
}

// GetClientStatistics возвращает статистику клиента
// GET /api/v2/clients/{id}/statistics
func (h *Handler) GetClientStatistics(w http.ResponseWriter, r *http.Request, clientID string) {
	stats, err := h.useCase.GetClientStatistics(r.Context(), clientID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// GetClientProjects возвращает список проектов клиента
// GET /api/v2/clients/{id}/projects
func (h *Handler) GetClientProjects(w http.ResponseWriter, r *http.Request, clientID string) {
	filter := repositories.ProjectFilter{
		Limit:  100,
		Offset: 0,
	}

	projects, total, err := h.useCase.ListProjects(r.Context(), clientID, filter)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"projects": projects,
		"total":    total,
	}, http.StatusOK)
}

// CreateClientProject создает новый проект для клиента
// POST /api/v2/clients/{id}/projects
func (h *Handler) CreateClientProject(w http.ResponseWriter, r *http.Request, clientID string) {
	var req clientapp.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	project, err := h.useCase.CreateProject(r.Context(), clientID, req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusCreated)
}

// GetProject возвращает проект по ID
// GET /api/v2/clients/{id}/projects/{project_id}
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request, clientID string, projectID string) {
	project, err := h.useCase.GetProject(r.Context(), clientID, projectID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusNotFound)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusOK)
}

// UpdateProject обновляет проект
// PUT /api/v2/clients/{id}/projects/{project_id}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request, clientID string, projectID string) {
	var req clientapp.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	project, err := h.useCase.UpdateProject(r.Context(), clientID, projectID, req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusOK)
}

// DeleteProject удаляет проект
// DELETE /api/v2/clients/{id}/projects/{project_id}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request, clientID string, projectID string) {
	if err := h.useCase.DeleteProject(r.Context(), clientID, projectID); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"message": "project deleted"}, http.StatusOK)
}

// GetClientDatabases возвращает базы данных клиента
// GET /api/v2/clients/{id}/databases
func (h *Handler) GetClientDatabases(w http.ResponseWriter, r *http.Request, clientID string) {
	databases, err := h.useCase.GetClientDatabases(r.Context(), clientID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"databases": databases,
		"total":     len(databases),
	}, http.StatusOK)
}

