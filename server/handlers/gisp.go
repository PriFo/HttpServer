package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"httpserver/server/services"
)

// GISPHandler обработчик для работы с GISP
type GISPHandler struct {
	gispService *services.GISPService
	baseHandler *BaseHandler
}

// NewGISPHandler создает новый обработчик для работы с GISP
func NewGISPHandler(
	gispService *services.GISPService,
	baseHandler *BaseHandler,
) *GISPHandler {
	return &GISPHandler{
		gispService: gispService,
		baseHandler: baseHandler,
	}
}

// HandleImportNomenclatures обрабатывает запросы к /api/gisp/nomenclatures/import
func (h *GISPHandler) HandleImportNomenclatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Парсим multipart/form-data
	err := r.ParseMultipartForm(100 << 20) // 100 MB max
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось распарсить форму: %v", err), http.StatusBadRequest)
		return
	}

	// Получаем файл
	file, header, err := r.FormFile("file")
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить файл: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := h.gispService.ImportNomenclatures(file, header.Filename)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось импортировать номенклатуры: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetNomenclatures обрабатывает запросы к /api/gisp/nomenclatures
func (h *GISPHandler) HandleGetNomenclatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	search := query.Get("search")
	okpd2Code := query.Get("okpd2")
	tnvedCode := query.Get("tnved")
	manufacturerIDStr := query.Get("manufacturer_id")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var manufacturerID *int
	if manufacturerIDStr != "" {
		if id, err := strconv.Atoi(manufacturerIDStr); err == nil {
			manufacturerID = &id
		}
	}

	result, err := h.gispService.GetNomenclatures(limit, offset, search, okpd2Code, tnvedCode, manufacturerID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить номенклатуры: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetNomenclatureDetail обрабатывает запросы к /api/gisp/nomenclatures/{id}
func (h *GISPHandler) HandleGetNomenclatureDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Извлекаем ID из пути
	path := strings.TrimPrefix(r.URL.Path, "/api/gisp/nomenclatures/")
	id, err := ValidateIntPathParam(path, "nomenclature_id")
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("неверный ID номенклатуры: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.gispService.GetNomenclatureDetail(id)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить детали номенклатуры: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetReferenceBooks обрабатывает запросы к /api/gisp/reference-books
func (h *GISPHandler) HandleGetReferenceBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	result, err := h.gispService.GetReferenceBooks()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить справочники: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleSearchReferenceBook обрабатывает запросы к /api/gisp/reference-books/search
func (h *GISPHandler) HandleSearchReferenceBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	query := r.URL.Query()
	bookType := query.Get("type")
	search := query.Get("search")
	limitStr := query.Get("limit")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	if bookType == "" {
		h.baseHandler.WriteJSONError(w, r, "параметр 'type' обязателен (okpd2, tnved, tu_gost)", http.StatusBadRequest)
		return
	}

	result, err := h.gispService.SearchReferenceBook(bookType, search, limit)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось выполнить поиск: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetStatistics обрабатывает запросы к /api/gisp/statistics
func (h *GISPHandler) HandleGetStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	result, err := h.gispService.GetStatistics()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("не удалось получить статистику: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

