package handlers

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"httpserver/database"
	"httpserver/server/types"
	"httpserver/server/services"
)

// UploadHandler обработчик для работы с выгрузками данных из 1С
type UploadHandler struct {
	uploadService       *services.UploadService
	notificationService *services.NotificationService
	baseHandler         *BaseHandler
	logFunc             func(entry interface{}) // server.LogEntry, но без прямого импорта для избежания циклических зависимостей
	normalizedDB        *database.DB            // Нормализованная БД для работы с нормализованными выгрузками
	currentNormalizedDBPath string
}

// NewUploadHandler создает новый обработчик для работы с выгрузками
func NewUploadHandler(
	uploadService *services.UploadService,
	baseHandler *BaseHandler,
	logFunc func(entry interface{}), // server.LogEntry, но без прямого импорта
) *UploadHandler {
	return &UploadHandler{
		uploadService: uploadService,
		baseHandler:   baseHandler,
		logFunc:       logFunc,
	}
}

// NewUploadHandlerWithNotifications создает новый обработчик с поддержкой уведомлений
func NewUploadHandlerWithNotifications(
	uploadService *services.UploadService,
	notificationService *services.NotificationService,
	baseHandler *BaseHandler,
	logFunc func(entry interface{}), // server.LogEntry, но без прямого импорта
) *UploadHandler {
	return &UploadHandler{
		uploadService:       uploadService,
		notificationService: notificationService,
		baseHandler:         baseHandler,
		logFunc:             logFunc,
	}
}

// SetNormalizedDB устанавливает normalizedDB и путь для работы с нормализованными выгрузками
func (h *UploadHandler) SetNormalizedDB(normalizedDB *database.DB, currentNormalizedDBPath string) {
	h.normalizedDB = normalizedDB
	h.currentNormalizedDBPath = currentNormalizedDBPath
}

// HandleHandshake обрабатывает рукопожатие
func (h *UploadHandler) HandleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.HandshakeRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Логирование всех полей итераций для отладки
	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Message: fmt.Sprintf("Handshake request received - Version1C: %s, ConfigName: %s, DatabaseID: %s, IterationNumber: %d, IterationLabel: %s, ProgrammerName: %s, UploadPurpose: %s, ParentUploadID: %s",
			req.Version1C, req.ConfigName, req.DatabaseID, req.IterationNumber, req.IterationLabel, req.ProgrammerName, req.UploadPurpose, req.ParentUploadID),
		Endpoint: "/handshake",
	})

	// Обрабатываем handshake через сервис
	result, err := h.uploadService.ProcessHandshake(req)
	if err != nil {
		WriteXMLError(w, r, "Failed to process handshake", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("Handshake successful for upload %s (database_id: %v, identified_by: %s, iteration_number: %d, iteration_label: %s, programmer: %s, purpose: %s, parent_upload_id: %v)",
			result.UploadUUID, result.DatabaseID, result.IdentifiedBy, result.Upload.IterationNumber, result.Upload.IterationLabel, result.Upload.ProgrammerName, result.Upload.UploadPurpose, result.Upload.ParentUploadID),
		UploadUUID: result.UploadUUID,
		Endpoint:   "/handshake",
	})

	response := types.HandshakeResponse{
		Success:      true,
		UploadUUID:   result.UploadUUID,
		ClientName:   result.ClientName,
		ProjectName:  result.ProjectName,
		DatabaseName: result.DatabaseName,
		Message:      "Handshake successful",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/handshake",
		})
	}
}

// HandleMetadata обрабатывает метаинформацию
func (h *UploadHandler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.MetadataRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Обрабатываем метаинформацию через сервис
	if err := h.uploadService.ProcessMetadata(req.UploadUUID); err != nil {
		WriteXMLError(w, r, "Failed to process metadata", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    "Metadata received successfully",
		UploadUUID: req.UploadUUID,
		Endpoint:   "/metadata",
	})

	response := types.MetadataResponse{
		Success:   true,
		Message:   "Metadata received successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/metadata",
		})
	}
}

// HandleConstant обрабатывает константу
func (h *UploadHandler) HandleConstant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	// Логирование сырого XML для отладки
	bodyStr := string(body)
	bodyPreview := bodyStr
	if len(bodyPreview) > 500 {
		bodyPreview = bodyPreview[:500] + "..."
	}
	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Message:   fmt.Sprintf("Received constant XML (length: %d): %s", len(bodyStr), bodyPreview),
		Endpoint:  "/constant",
	})

	var req types.ConstantRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to parse XML: %v, body preview: %s", err, bodyPreview),
			Endpoint:  "/constant",
		})
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Логирование распарсенных данных
	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Message:   fmt.Sprintf("Parsed constant - Name: %s, Type: %s, Value.Content length: %d, Value.Content: %s", req.Name, req.Type, len(req.Value.Content), req.Value.Content),
		Endpoint:  "/constant",
	})

	// Обрабатываем константу через сервис
	valueContent := req.Value.Content
	if err := h.uploadService.ProcessConstant(req.UploadUUID, req.Name, req.Synonym, req.Type, valueContent); err != nil {
		WriteXMLError(w, r, "Failed to process constant", err)
		return
	}

	// Логирование для отладки
	valuePreview := valueContent
	if len(valuePreview) > 100 {
		valuePreview = valuePreview[:100] + "..."
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Constant '%s' (type: %s) added successfully, value preview: %s", req.Name, req.Type, valuePreview),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/constant",
	})

	response := types.ConstantResponse{
		Success:   true,
		Message:   "Constant added successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/constant",
		})
	}
}

// HandleCatalogMeta обрабатывает метаданные справочника
func (h *UploadHandler) HandleCatalogMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.CatalogMetaRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Обрабатываем метаданные справочника через сервис
	catalog, err := h.uploadService.ProcessCatalogMeta(req.UploadUUID, req.Name, req.Synonym)
	if err != nil {
		WriteXMLError(w, r, "Failed to process catalog metadata", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Catalog '%s' metadata added successfully", req.Name),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/catalog/meta",
	})

	response := types.CatalogMetaResponse{
		Success:   true,
		CatalogID: catalog.ID,
		Message:   "Catalog metadata added successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/catalog/meta",
		})
	}
}

// HandleCatalogItem обрабатывает элемент справочника
func (h *UploadHandler) HandleCatalogItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.CatalogItemRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Обрабатываем элемент справочника через сервис
	if err := h.uploadService.ProcessCatalogItem(req.UploadUUID, req.CatalogName, req.Reference, req.Code, req.Name, req.Attributes, req.TableParts); err != nil {
		WriteXMLError(w, r, "Failed to process catalog item", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Catalog item '%s' added successfully", req.Name),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/catalog/item",
	})

	response := types.CatalogItemResponse{
		Success:   true,
		Message:   "Catalog item added successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/catalog/item",
		})
	}
}

// HandleCatalogItems обрабатывает пакетную загрузку элементов справочника
func (h *UploadHandler) HandleCatalogItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.CatalogItemsRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Обрабатываем пакет элементов справочника через сервис
	processedCount, failedCount, err := h.uploadService.ProcessCatalogItemsBatch(req.UploadUUID, req.CatalogName, req.Items)
	if err != nil {
		WriteXMLError(w, r, "Failed to process catalog items", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Batch catalog items processed: %d successful, %d failed", processedCount, failedCount),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/catalog/items",
	})

	response := types.CatalogItemsResponse{
		Success:        true,
		ProcessedCount: processedCount,
		FailedCount:    failedCount,
		Message:        fmt.Sprintf("Processed %d items, %d failed", processedCount, failedCount),
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/catalog/items",
		})
	}
}

// HandleNomenclatureBatch обрабатывает пакетную загрузку номенклатуры
func (h *UploadHandler) HandleNomenclatureBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.NomenclatureBatchRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Обрабатываем пакет номенклатуры через сервис
	processedCount, err := h.uploadService.ProcessNomenclatureBatch(req.UploadUUID, req.Items)
	if err != nil {
		WriteXMLError(w, r, "Failed to process nomenclature batch", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Batch nomenclature items processed: %d items", processedCount),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/v1/upload/nomenclature/batch",
	})

	response := types.NomenclatureBatchResponse{
		Success:        true,
		ProcessedCount: processedCount,
		FailedCount:    0,
		Message:        fmt.Sprintf("Processed %d nomenclature items", processedCount),
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/api/v1/upload/nomenclature/batch",
		})
	}
}

// HandleComplete обрабатывает завершение выгрузки
// qualityAnalyzerFunc - опциональная функция для запуска анализа качества в фоне
func (h *UploadHandler) HandleComplete(w http.ResponseWriter, r *http.Request, qualityAnalyzerFunc func(uploadID, databaseID int) error) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.CompleteRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Обрабатываем завершение выгрузки через сервис
	upload, err := h.uploadService.ProcessCompleteWithUpload(req.UploadUUID)
	if err != nil {
		WriteXMLError(w, r, "Failed to complete upload", err)
		// Отправляем уведомление об ошибке
		if h.notificationService != nil {
			ctx := r.Context()
			_, _ = h.notificationService.AddNotification(ctx, services.NotificationTypeError, "Ошибка завершения загрузки", fmt.Sprintf("Не удалось завершить загрузку %s: %v", req.UploadUUID, err), nil, nil, map[string]interface{}{"upload_uuid": req.UploadUUID})
		}
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Upload %s completed successfully", req.UploadUUID),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/complete",
	})

	// Отправляем уведомление об успешном завершении загрузки
	if h.notificationService != nil {
		ctx := r.Context()
		var clientID, projectID *int
		// Пытаемся получить clientID и projectID из метаданных upload, если они есть
		if upload.DatabaseID != nil {
			if cID, pID, err := h.uploadService.GetClientProjectIDs(*upload.DatabaseID); err == nil {
				clientID = &cID
				projectID = &pID
			}
		}
		_, _ = h.notificationService.AddNotification(ctx, services.NotificationTypeSuccess, "Загрузка завершена", fmt.Sprintf("Загрузка %s успешно завершена", req.UploadUUID), clientID, projectID, map[string]interface{}{"upload_uuid": req.UploadUUID, "upload_id": upload.ID})
	}

	// Запускаем анализ качества в фоне, если функция предоставлена
	if qualityAnalyzerFunc != nil {
		databaseID := 0
		if upload.DatabaseID != nil {
			databaseID = *upload.DatabaseID
		}

		if databaseID > 0 {
			go func() {
				if err := qualityAnalyzerFunc(upload.ID, databaseID); err != nil {
					h.logFunc(types.LogEntry{
						Timestamp:  time.Now(),
						Level:      "ERROR",
						Message:    fmt.Sprintf("Quality analysis failed for upload %s: %v", req.UploadUUID, err),
						UploadUUID: req.UploadUUID,
						Endpoint:   "/complete",
					})
					// Отправляем уведомление об ошибке анализа качества
					if h.notificationService != nil {
						ctx := context.Background()
						var clientID, projectID *int
						if cID, pID, err := h.uploadService.GetClientProjectIDs(databaseID); err == nil {
							clientID = &cID
							projectID = &pID
						}
						_, _ = h.notificationService.AddNotification(ctx, services.NotificationTypeError, "Ошибка анализа качества", fmt.Sprintf("Ошибка анализа качества для загрузки %s: %v", req.UploadUUID, err), clientID, projectID, map[string]interface{}{"upload_uuid": req.UploadUUID, "upload_id": upload.ID, "database_id": databaseID})
					}
				} else {
					// Отправляем уведомление об успешном завершении анализа качества
					if h.notificationService != nil {
						ctx := context.Background()
						var clientID, projectID *int
						if cID, pID, err := h.uploadService.GetClientProjectIDs(databaseID); err == nil {
							clientID = &cID
							projectID = &pID
						}
						_, _ = h.notificationService.AddNotification(ctx, services.NotificationTypeSuccess, "Анализ качества завершен", fmt.Sprintf("Анализ качества для загрузки %s успешно завершен", req.UploadUUID), clientID, projectID, map[string]interface{}{"upload_uuid": req.UploadUUID, "upload_id": upload.ID, "database_id": databaseID})
					}
				}
			}()
		}
	}

	response := types.CompleteResponse{
		Success:   true,
		Message:   "Upload completed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:   "/complete",
		})
	}
}

// HandleListUploads обрабатывает запросы к /api/uploads
func (h *UploadHandler) HandleListUploads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем все выгрузки
	uploads, err := h.uploadService.GetAllUploads()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get uploads: %v", err), http.StatusInternalServerError)
		return
	}

	// Применяем фильтры
	filteredUploads := h.filterUploads(uploads, r)

	// Применяем пагинацию
	limit, _ := ValidateIntParam(r, "limit", 50, 1, 1000)
	offset, _ := ValidateIntParam(r, "offset", 0, 0, 0)

	total := len(filteredUploads)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	var paginatedUploads []*database.Upload
	if start < end {
		paginatedUploads = filteredUploads[start:end]
	} else {
		paginatedUploads = []*database.Upload{}
	}

	// Формируем ответ
	response := map[string]interface{}{
		"uploads": paginatedUploads,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"count":   len(paginatedUploads),
		"has_more": offset+len(paginatedUploads) < total,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// filterUploads применяет фильтры к списку выгрузок
func (h *UploadHandler) filterUploads(uploads []*database.Upload, r *http.Request) []*database.Upload {
	var filtered []*database.Upload

	databaseID := r.URL.Query().Get("database_id")
	clientID := r.URL.Query().Get("client_id")
	projectID := r.URL.Query().Get("project_id")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	for _, upload := range uploads {
		// Фильтр по database_id
		if databaseID != "" {
			if upload.DatabaseID == nil {
				continue
			}
			if fmt.Sprintf("%d", *upload.DatabaseID) != databaseID {
				continue
			}
		}

		// Фильтр по client_id
		if clientID != "" {
			if upload.ClientID == nil {
				continue
			}
			if fmt.Sprintf("%d", *upload.ClientID) != clientID {
				continue
			}
		}

		// Фильтр по project_id
		if projectID != "" {
			if upload.ProjectID == nil {
				continue
			}
			if fmt.Sprintf("%d", *upload.ProjectID) != projectID {
				continue
			}
		}

		// Фильтр по status
		if status != "" && upload.Status != status {
			continue
		}

		// Поиск по тексту (в UUID, ConfigName, Version1C)
		if search != "" {
			searchLower := strings.ToLower(search)
			matched := strings.Contains(strings.ToLower(upload.UploadUUID), searchLower) ||
				strings.Contains(strings.ToLower(upload.ConfigName), searchLower) ||
				strings.Contains(strings.ToLower(upload.Version1C), searchLower) ||
				strings.Contains(strings.ToLower(upload.ComputerName), searchLower) ||
				strings.Contains(strings.ToLower(upload.UserName), searchLower)
			if !matched {
				continue
			}
		}

		filtered = append(filtered, upload)
	}

	return filtered
}

// HandleUploadRoutes обрабатывает маршруты для конкретной выгрузки
// Поддерживает: GET /api/uploads/{uuid} - детали выгрузки
func (h *UploadHandler) HandleUploadRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.baseHandler.WriteJSONError(w, r, "Upload UUID required", http.StatusBadRequest)
		return
	}

	uuid := parts[0]

	// Получаем выгрузку
	upload, err := h.uploadService.GetUploadByUUID(uuid)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, "Upload not found", http.StatusNotFound)
		return
	}

	// Обрабатываем подмаршруты
	if len(parts) == 1 {
		// GET /api/uploads/{uuid} - детали выгрузки
		if r.Method != http.MethodGet {
			h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
			return
		}
		h.handleGetUpload(w, r, upload)
	} else {
		// Другие подмаршруты пока не реализованы
		h.baseHandler.WriteJSONError(w, r, "Sub-route not implemented", http.StatusNotImplemented)
	}
}

// handleGetUpload обрабатывает запрос детальной информации о выгрузке
func (h *UploadHandler) handleGetUpload(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	// Получаем детали выгрузки
	_, catalogs, constants, err := h.uploadService.GetUploadDetails(upload.UploadUUID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get upload details: %v", err), http.StatusInternalServerError)
		return
	}

	// Преобразуем константы в интерфейсы
	constantData := make([]interface{}, len(constants))
	for i, constant := range constants {
		constantData[i] = map[string]interface{}{
			"id":         constant.ID,
			"name":       constant.Name,
			"synonym":    constant.Synonym,
			"type":       constant.Type,
			"value":      constant.Value,
			"created_at": constant.CreatedAt,
		}
	}

	// Преобразуем справочники
	catalogData := make([]interface{}, len(catalogs))
	for i, catalog := range catalogs {
		catalogData[i] = map[string]interface{}{
			"id":         catalog.ID,
			"name":       catalog.Name,
			"synonym":    catalog.Synonym,
			"created_at": catalog.CreatedAt,
		}
	}

	details := map[string]interface{}{
		"upload_uuid":      upload.UploadUUID,
		"started_at":       upload.StartedAt,
		"completed_at":     upload.CompletedAt,
		"status":           upload.Status,
		"version_1c":       upload.Version1C,
		"config_name":      upload.ConfigName,
		"total_constants":  upload.TotalConstants,
		"total_catalogs":   upload.TotalCatalogs,
		"total_items":      upload.TotalItems,
		"database_id":      upload.DatabaseID,
		"client_id":        upload.ClientID,
		"project_id":       upload.ProjectID,
		"computer_name":    upload.ComputerName,
		"user_name":        upload.UserName,
		"config_version":   upload.ConfigVersion,
		"iteration_number": upload.IterationNumber,
		"iteration_label":  upload.IterationLabel,
		"programmer_name":   upload.ProgrammerName,
		"upload_purpose":    upload.UploadPurpose,
		"parent_upload_id":  upload.ParentUploadID,
		"catalogs":          catalogData,
		"constants":         constantData,
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Upload details requested for %s", upload.UploadUUID),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/uploads/{uuid}",
	})

	h.baseHandler.WriteJSONResponse(w, r, details, http.StatusOK)
}

// HandleNormalizedListUploads обрабатывает запросы к /api/normalized/uploads
func (h *UploadHandler) HandleNormalizedListUploads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.normalizedDB == nil {
		h.baseHandler.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
		return
	}

	// Получаем все выгрузки из нормализованной БД
	uploads, err := h.normalizedDB.GetAllUploads()
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get normalized uploads: %v", err), http.StatusInternalServerError)
		return
	}

	// Применяем фильтры (используем ту же логику, что и для обычных выгрузок)
	filteredUploads := h.filterUploads(uploads, r)

	// Применяем пагинацию
	limit, _ := ValidateIntParam(r, "limit", 50, 1, 1000)
	offset, _ := ValidateIntParam(r, "offset", 0, 0, 0)

	total := len(filteredUploads)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	var paginatedUploads []*database.Upload
	if start < end {
		paginatedUploads = filteredUploads[start:end]
	} else {
		paginatedUploads = []*database.Upload{}
	}

	// Формируем ответ
	response := map[string]interface{}{
		"uploads":  paginatedUploads,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"count":    len(paginatedUploads),
		"has_more": offset+len(paginatedUploads) < total,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleNormalizedUploadRoutes обрабатывает маршруты для нормализованной выгрузки
// Поддерживает: GET /api/normalized/uploads/{uuid} - детали нормализованной выгрузки
func (h *UploadHandler) HandleNormalizedUploadRoutes(w http.ResponseWriter, r *http.Request) {
	if h.normalizedDB == nil {
		h.baseHandler.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/normalized/uploads/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.baseHandler.WriteJSONError(w, r, "Upload UUID required", http.StatusBadRequest)
		return
	}

	uuid := parts[0]

	// Получаем выгрузку из нормализованной БД
	upload, err := h.normalizedDB.GetUploadByUUID(uuid)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, "Normalized upload not found", http.StatusNotFound)
		return
	}

	// Обрабатываем подмаршруты
	if len(parts) == 1 {
		// GET /api/normalized/uploads/{uuid} - детали нормализованной выгрузки
		if r.Method != http.MethodGet {
			h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
			return
		}
		h.handleGetUploadNormalized(w, r, upload)
	} else {
		// Другие подмаршруты пока не реализованы
		h.baseHandler.WriteJSONError(w, r, "Sub-route not implemented", http.StatusNotImplemented)
	}
}

// handleGetUploadNormalized обрабатывает запрос детальной информации о нормализованной выгрузке
func (h *UploadHandler) handleGetUploadNormalized(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	// Получаем детали выгрузки из нормализованной БД
	_, catalogs, constants, err := h.normalizedDB.GetUploadDetails(upload.UploadUUID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get normalized upload details: %v", err), http.StatusInternalServerError)
		return
	}

	// Преобразуем константы в интерфейсы
	constantData := make([]interface{}, len(constants))
	for i, constant := range constants {
		constantData[i] = map[string]interface{}{
			"id":         constant.ID,
			"name":       constant.Name,
			"synonym":    constant.Synonym,
			"type":       constant.Type,
			"value":      constant.Value,
			"created_at": constant.CreatedAt,
		}
	}

	// Преобразуем справочники
	catalogData := make([]interface{}, len(catalogs))
	for i, catalog := range catalogs {
		catalogData[i] = map[string]interface{}{
			"id":         catalog.ID,
			"name":       catalog.Name,
			"synonym":    catalog.Synonym,
			"created_at": catalog.CreatedAt,
		}
	}

	details := map[string]interface{}{
		"upload_uuid":      upload.UploadUUID,
		"started_at":       upload.StartedAt,
		"completed_at":     upload.CompletedAt,
		"status":           upload.Status,
		"version_1c":       upload.Version1C,
		"config_name":      upload.ConfigName,
		"total_constants":  upload.TotalConstants,
		"total_catalogs":   upload.TotalCatalogs,
		"total_items":      upload.TotalItems,
		"database_id":      upload.DatabaseID,
		"client_id":        upload.ClientID,
		"project_id":       upload.ProjectID,
		"computer_name":    upload.ComputerName,
		"user_name":        upload.UserName,
		"config_version":   upload.ConfigVersion,
		"iteration_number": upload.IterationNumber,
		"iteration_label":  upload.IterationLabel,
		"programmer_name":   upload.ProgrammerName,
		"upload_purpose":    upload.UploadPurpose,
		"parent_upload_id":  upload.ParentUploadID,
		"catalogs":          catalogData,
		"constants":         constantData,
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized upload details requested for %s", upload.UploadUUID),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/normalized/uploads/{uuid}",
	})

	h.baseHandler.WriteJSONResponse(w, r, details, http.StatusOK)
}

// HandleNormalizedHandshake обрабатывает рукопожатие для нормализованных данных
func (h *UploadHandler) HandleNormalizedHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.normalizedDB == nil {
		WriteXMLError(w, r, "Normalized database not available", nil)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.HandshakeRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Создаем новую выгрузку в нормализованной БД
	uploadUUID := uuid.New().String()
	_, err = h.normalizedDB.CreateUpload(uploadUUID, req.Version1C, req.ConfigName)
	if err != nil {
		WriteXMLError(w, r, "Failed to create normalized upload", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized handshake successful for upload %s", uploadUUID),
		UploadUUID: uploadUUID,
		Endpoint:   "/api/normalized/upload/handshake",
	})

	response := types.HandshakeResponse{
		Success:    true,
		UploadUUID: uploadUUID,
		Message:    "Normalized handshake successful",
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:  "/api/normalized/upload/handshake",
		})
	}
}

// HandleNormalizedMetadata обрабатывает метаинформацию для нормализованных данных
func (h *UploadHandler) HandleNormalizedMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.normalizedDB == nil {
		WriteXMLError(w, r, "Normalized database not available", nil)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.MetadataRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Проверяем существование выгрузки в нормализованной БД
	_, err = h.normalizedDB.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		WriteXMLError(w, r, "Normalized upload not found", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    "Normalized metadata received successfully",
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/normalized/upload/metadata",
	})

	response := types.MetadataResponse{
		Success:   true,
		Message:   "Normalized metadata received successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:  "/api/normalized/upload/metadata",
		})
	}
}

// HandleNormalizedConstant обрабатывает константу для нормализованных данных
func (h *UploadHandler) HandleNormalizedConstant(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать обработку константы для нормализованных данных
	h.baseHandler.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleNormalizedCatalogMeta обрабатывает метаданные справочника для нормализованных данных
func (h *UploadHandler) HandleNormalizedCatalogMeta(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать обработку метаданных справочника для нормализованных данных
	h.baseHandler.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleNormalizedCatalogItem обрабатывает элемент справочника для нормализованных данных
func (h *UploadHandler) HandleNormalizedCatalogItem(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать обработку элемента справочника для нормализованных данных
	h.baseHandler.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleNormalizedComplete обрабатывает завершение выгрузки для нормализованных данных
func (h *UploadHandler) HandleNormalizedComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if h.normalizedDB == nil {
		WriteXMLError(w, r, "Normalized database not available", nil)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteXMLError(w, r, "Failed to read request body", err)
		return
	}

	var req types.CompleteRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		WriteXMLError(w, r, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку из нормализованной БД
	upload, err := h.normalizedDB.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		WriteXMLError(w, r, "Normalized upload not found", err)
		return
	}

	// Завершаем выгрузку в нормализованной БД
	if err := h.normalizedDB.CompleteUpload(upload.ID); err != nil {
		WriteXMLError(w, r, "Failed to complete normalized upload", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized upload %s completed successfully", req.UploadUUID),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/normalized/upload/complete",
	})

	response := types.CompleteResponse{
		Success:   true,
		Message:   "Normalized upload completed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := WriteXMLResponse(w, response); err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:  "/api/normalized/upload/complete",
		})
	}
}