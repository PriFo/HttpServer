package project

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	projectapp "httpserver/internal/application/project"
	"httpserver/internal/domain/repositories"
	"httpserver/server/middleware"
)

// BaseHandlerInterface интерфейс для базового обработчика (разрывает циклический импорт)
type BaseHandlerInterface interface {
	WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int)
	WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int)
	HandleHTTPError(w http.ResponseWriter, r *http.Request, err error)
}

// baseHandlerImpl реализация BaseHandlerInterface через middleware
type baseHandlerImpl struct{}

func (h *baseHandlerImpl) WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	middleware.WriteJSONResponse(w, r, data, statusCode)
}

func (h *baseHandlerImpl) WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	middleware.WriteJSONError(w, r, message, statusCode)
}

func (h *baseHandlerImpl) HandleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	middleware.HandleHTTPError(w, r, err)
}

// Handler HTTP обработчик для работы с проектами
type Handler struct {
	baseHandler BaseHandlerInterface
	useCase     *projectapp.UseCase
}

// NewHandler создает новый HTTP обработчик для проектов
func NewHandler(
	baseHandler BaseHandlerInterface,
	useCase *projectapp.UseCase,
) *Handler {
	return &Handler{
		baseHandler: baseHandler,
		useCase:     useCase,
	}
}

// HandleProjects обрабатывает запросы к /api/v2/projects
func (h *Handler) HandleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetProjects(w, r)
	case http.MethodPost:
		h.CreateProject(w, r)
	default:
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// GetProjects возвращает список проектов
// GET /api/v2/projects
func (h *Handler) GetProjects(w http.ResponseWriter, r *http.Request) {
	filter := repositories.ProjectFilter{
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

	if projectType := r.URL.Query().Get("type"); projectType != "" {
		filter.Type = projectType
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = []string{status}
	}

	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		filter.ClientID = clientID
	}

	projects, total, err := h.useCase.ListProjects(r.Context(), filter)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"projects": projects,
		"total":    total,
		"limit":    filter.Limit,
		"offset":   filter.Offset,
	}, http.StatusOK)
}

// CreateProject создает новый проект
// POST /api/v2/projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req projectapp.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	project, err := h.useCase.CreateProject(r.Context(), req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusCreated)
}

// HandleProjectRoutes обрабатывает запросы к /api/v2/projects/{id}
func (h *Handler) HandleProjectRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "project ID required"}, http.StatusBadRequest)
		return
	}

	projectID := parts[0]

	// Обработка вложенных маршрутов
	if len(parts) > 1 {
		switch parts[1] {
		case "statistics":
			if r.Method == http.MethodGet {
				h.GetProjectStatistics(w, r, projectID)
				return
			}
		case "databases":
			if r.Method == http.MethodGet {
				h.GetProjectDatabases(w, r, projectID)
				return
			}
		}
	}

	// Обработка основных операций с проектом
	switch r.Method {
	case http.MethodGet:
		h.GetProject(w, r, projectID)
	case http.MethodPut:
		h.UpdateProject(w, r, projectID)
	case http.MethodDelete:
		h.DeleteProject(w, r, projectID)
	default:
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
	}
}

// GetProject возвращает проект по ID
// GET /api/v2/projects/{id}
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request, projectID string) {
	project, err := h.useCase.GetProject(r.Context(), projectID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusNotFound)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusOK)
}

// UpdateProject обновляет проект
// PUT /api/v2/projects/{id}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request, projectID string) {
	var req projectapp.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	project, err := h.useCase.UpdateProject(r.Context(), projectID, req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusOK)
}

// DeleteProject удаляет проект
// DELETE /api/v2/projects/{id}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	if err := h.useCase.DeleteProject(r.Context(), projectID); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"message": "project deleted"}, http.StatusOK)
}

// GetProjectStatistics возвращает статистику проекта
// GET /api/v2/projects/{id}/statistics
func (h *Handler) GetProjectStatistics(w http.ResponseWriter, r *http.Request, projectID string) {
	stats, err := h.useCase.GetProjectStatistics(r.Context(), projectID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// GetProjectDatabases возвращает базы данных проекта
// GET /api/v2/projects/{id}/databases
func (h *Handler) GetProjectDatabases(w http.ResponseWriter, r *http.Request, projectID string) {
	databases, err := h.useCase.GetProjectDatabases(r.Context(), projectID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"databases": databases,
		"total":     len(databases),
	}, http.StatusOK)
}

