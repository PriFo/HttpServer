package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DiagnosticsServer интерфейс для методов диагностики
type DiagnosticsServer interface {
	CheckAllProjectDatabases(projectID, clientID int) ([]interface{}, error)
	CheckUploadRecords(projectID, clientID int) ([]interface{}, error)
	CreateMissingUploads(projectID, clientID int) (int, error)
	CheckExtractionStatus(projectID, clientID int) ([]interface{}, error)
	CheckNormalizationStatus(projectID int) (interface{}, error)
}

// DiagnosticsHandler обработчик для диагностики цепочки данных
type DiagnosticsHandler struct {
	server DiagnosticsServer
}

// NewDiagnosticsHandler создает новый обработчик диагностики
func NewDiagnosticsHandler(server DiagnosticsServer) *DiagnosticsHandler {
	return &DiagnosticsHandler{
		server: server,
	}
}

// HandleCheckDatabases проверяет все базы данных проекта
// @Summary Проверить базы данных проекта
// @Description Возвращает диагностику всех баз данных проекта
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Success 200 {array} DatabaseDiagnostic
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/clients/:clientId/projects/:projectId/diagnostics/databases [get]
func (h *DiagnosticsHandler) HandleCheckDatabases(c *gin.Context) {
	clientIDStr := c.Param("clientId")
	projectIDStr := c.Param("projectId")

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid client ID")
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid project ID")
		return
	}

	diagnostics, err := h.server.CheckAllProjectDatabases(projectID, clientID)
	if err != nil {
		SendJSONError(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSONResponse(c, http.StatusOK, diagnostics)
}

// HandleCheckUploads проверяет upload записи
// @Summary Проверить upload записи
// @Description Возвращает статус upload записей для всех баз данных проекта
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Success 200 {array} UploadStatus
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/clients/:clientId/projects/:projectId/diagnostics/uploads [get]
func (h *DiagnosticsHandler) HandleCheckUploads(c *gin.Context) {
	clientIDStr := c.Param("clientId")
	projectIDStr := c.Param("projectId")

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid client ID")
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid project ID")
		return
	}

	statuses, err := h.server.CheckUploadRecords(projectID, clientID)
	if err != nil {
		SendJSONError(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSONResponse(c, http.StatusOK, statuses)
}

// HandleFixUploads создает недостающие upload записи
// @Summary Создать недостающие upload записи
// @Description Создает upload записи для баз данных, у которых их нет
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/clients/:clientId/projects/:projectId/diagnostics/uploads/fix [post]
func (h *DiagnosticsHandler) HandleFixUploads(c *gin.Context) {
	clientIDStr := c.Param("clientId")
	projectIDStr := c.Param("projectId")

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid client ID")
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid project ID")
		return
	}

	fixedCount, err := h.server.CreateMissingUploads(projectID, clientID)
	if err != nil {
		SendJSONError(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSONResponse(c, http.StatusOK, map[string]interface{}{
		"fixed_count": fixedCount,
		"message":     "Upload записи созданы",
	})
}

// HandleCheckExtraction проверяет статус извлечения данных
// @Summary Проверить извлечение данных
// @Description Возвращает статус извлечения данных для всех баз данных проекта
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Success 200 {array} ExtractionStatus
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/clients/:clientId/projects/:projectId/diagnostics/extraction [get]
func (h *DiagnosticsHandler) HandleCheckExtraction(c *gin.Context) {
	clientIDStr := c.Param("clientId")
	projectIDStr := c.Param("projectId")

	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid client ID")
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid project ID")
		return
	}

	statuses, err := h.server.CheckExtractionStatus(projectID, clientID)
	if err != nil {
		SendJSONError(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSONResponse(c, http.StatusOK, statuses)
}

// HandleCheckNormalization проверяет статус нормализации
// @Summary Проверить нормализацию
// @Description Возвращает статус нормализации для проекта
// @Tags diagnostics
// @Accept json
// @Produce json
// @Param projectId path int true "ID проекта"
// @Success 200 {object} DiagnosticNormalizationStatus
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/clients/:clientId/projects/:projectId/diagnostics/normalization [get]
func (h *DiagnosticsHandler) HandleCheckNormalization(c *gin.Context) {
	projectIDStr := c.Param("projectId")

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Invalid project ID")
		return
	}

	status, err := h.server.CheckNormalizationStatus(projectID)
	if err != nil {
		SendJSONError(c, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSONResponse(c, http.StatusOK, status)
}

