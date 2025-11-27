package normalization

import (
	"encoding/json"
	"net/http"

	normalizationapp "httpserver/internal/application/normalization"
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

// Handler HTTP обработчик для работы с нормализацией
type Handler struct {
	baseHandler BaseHandlerInterface
	useCase     *normalizationapp.UseCase
}

// NewHandler создает новый HTTP обработчик для нормализации
func NewHandler(
	baseHandler BaseHandlerInterface,
	useCase *normalizationapp.UseCase,
) *Handler {
	return &Handler{
		baseHandler: baseHandler,
		useCase:     useCase,
	}
}

// HandleStartProcess запускает процесс нормализации
// POST /api/normalize/start
func (h *Handler) HandleStartProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UploadID string `json:"upload_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.StartProcess(r.Context(), req.UploadID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetProcessStatus возвращает статус процесса нормализации
// GET /api/normalization/status?process_id={id}
func (h *Handler) HandleGetProcessStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	processID := r.URL.Query().Get("process_id")
	if processID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "process_id is required"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.GetProcessStatus(r.Context(), processID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusNotFound)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleStopProcess останавливает процесс нормализации
// POST /api/normalization/stop
func (h *Handler) HandleStopProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProcessID string `json:"process_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := h.useCase.StopProcess(r.Context(), req.ProcessID); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "stopped"}, http.StatusOK)
}

// HandleGetActiveProcesses возвращает активные процессы нормализации
// GET /api/normalization/processes/active
func (h *Handler) HandleGetActiveProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	result, err := h.useCase.GetActiveProcesses(r.Context())
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"processes": result,
		"count":     len(result),
	}, http.StatusOK)
}

// HandleNormalizeName нормализует название
// POST /api/normalization/normalize-name
func (h *Handler) HandleNormalizeName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name       string `json:"name"`
		EntityType string `json:"entity_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.NormalizeName(r.Context(), req.Name, req.EntityType)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{
		"original_name":  req.Name,
		"normalized_name": result,
	}, http.StatusOK)
}

// HandleGetStatistics возвращает статистику нормализации
// GET /api/normalization/stats
func (h *Handler) HandleGetStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	result, err := h.useCase.GetStatistics(r.Context())
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetProcessHistory возвращает историю процессов нормализации
// GET /api/normalization/history?upload_id={id}
func (h *Handler) HandleGetProcessHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadID := r.URL.Query().Get("upload_id")
	if uploadID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_id is required"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.GetProcessHistory(r.Context(), uploadID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"processes": result,
		"count":     len(result),
	}, http.StatusOK)
}

