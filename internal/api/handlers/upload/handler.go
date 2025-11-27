package upload

import (
	"encoding/json"
	"net/http"

	uploadapp "httpserver/internal/application/upload"
	uploaddomain "httpserver/internal/domain/upload"
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

// Handler HTTP обработчик для работы с выгрузками
type Handler struct {
	baseHandler BaseHandlerInterface
	useCase     *uploadapp.UseCase
}

// NewHandler создает новый HTTP обработчик для выгрузок
func NewHandler(
	baseHandler BaseHandlerInterface,
	useCase *uploadapp.UseCase,
) *Handler {
	return &Handler{
		baseHandler: baseHandler,
		useCase:     useCase,
	}
}

// HandleHandshake обрабатывает handshake запрос
// POST /handshake или POST /api/v1/upload/handshake
func (h *Handler) HandleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req uploaddomain.HandshakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.ProcessHandshake(r.Context(), req)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleMetadata обрабатывает метаданные выгрузки
// POST /metadata или POST /api/v1/upload/metadata
func (h *Handler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadUUID := r.URL.Query().Get("upload_uuid")
	if uploadUUID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
		return
	}

	var req uploaddomain.MetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := h.useCase.ProcessMetadata(r.Context(), uploadUUID, req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "ok"}, http.StatusOK)
}

// HandleConstant обрабатывает константу
// POST /constant
func (h *Handler) HandleConstant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadUUID := r.URL.Query().Get("upload_uuid")
	if uploadUUID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
		return
	}

	var req uploaddomain.ConstantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := h.useCase.ProcessConstant(r.Context(), uploadUUID, req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "ok"}, http.StatusOK)
}

// HandleCatalogMeta обрабатывает метаданные каталога
// POST /catalog/meta
func (h *Handler) HandleCatalogMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadUUID := r.URL.Query().Get("upload_uuid")
	if uploadUUID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
		return
	}

	var req uploaddomain.CatalogMetaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := h.useCase.ProcessCatalogMeta(r.Context(), uploadUUID, req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "ok"}, http.StatusOK)
}

// HandleCatalogItem обрабатывает элемент каталога
// POST /catalog/item
func (h *Handler) HandleCatalogItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadUUID := r.URL.Query().Get("upload_uuid")
	if uploadUUID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
		return
	}

	var req uploaddomain.CatalogItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := h.useCase.ProcessCatalogItem(r.Context(), uploadUUID, req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "ok"}, http.StatusOK)
}

// HandleCatalogItems обрабатывает пакет элементов каталога
// POST /catalog/items
func (h *Handler) HandleCatalogItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadUUID := r.URL.Query().Get("upload_uuid")
	if uploadUUID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
		return
	}

	var items []uploaddomain.CatalogItemRequest
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := h.useCase.ProcessCatalogItems(r.Context(), uploadUUID, items); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "ok"}, http.StatusOK)
}

// HandleNomenclatureBatch обрабатывает пакет номенклатуры
// POST /api/v1/upload/nomenclature/batch
func (h *Handler) HandleNomenclatureBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadUUID := r.URL.Query().Get("upload_uuid")
	if uploadUUID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
		return
	}

	var req uploaddomain.NomenclatureBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return
	}

	if err := h.useCase.ProcessNomenclatureBatch(r.Context(), uploadUUID, req); err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "ok"}, http.StatusOK)
}

// HandleComplete обрабатывает завершение выгрузки
// POST /complete
func (h *Handler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	uploadUUID := r.URL.Query().Get("upload_uuid")
	if uploadUUID == "" {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "upload_uuid is required"}, http.StatusBadRequest)
		return
	}

	result, err := h.useCase.CompleteUpload(r.Context(), uploadUUID)
	if err != nil {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleListUploads возвращает список выгрузок
// GET /api/uploads
func (h *Handler) HandleListUploads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	// TODO: Парсинг фильтров из query параметров
	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "not implemented"}, http.StatusOK)
}

// HandleGetUpload возвращает выгрузку по UUID
// GET /api/uploads/{uuid}
func (h *Handler) HandleGetUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.WriteJSONResponse(w, r, map[string]string{"error": "method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	// TODO: Извлечение UUID из пути
	h.baseHandler.WriteJSONResponse(w, r, map[string]string{"status": "not implemented"}, http.StatusOK)
}

