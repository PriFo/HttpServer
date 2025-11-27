package handlers

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/internal/domain/models"
	"httpserver/internal/infrastructure/cache"
	"httpserver/quality"
	"httpserver/server/types"

	"github.com/google/uuid"
)

// UploadLegacyHandler обработчик для legacy upload endpoints (handshake, metadata, catalog)
type UploadLegacyHandler struct {
	db              *database.DB
	serviceDB       *database.ServiceDB
	dbInfoCache     *cache.DatabaseInfoCache
	qualityAnalyzer *quality.QualityAnalyzer
	logFunc         func(entry types.LogEntry)
}

// NewUploadLegacyHandler создает новый обработчик legacy upload
func NewUploadLegacyHandler(
	db *database.DB,
	serviceDB *database.ServiceDB,
	dbInfoCache *cache.DatabaseInfoCache,
	qualityAnalyzer *quality.QualityAnalyzer,
	logFunc func(entry types.LogEntry),
) *UploadLegacyHandler {
	return &UploadLegacyHandler{
		db:              db,
		serviceDB:       serviceDB,
		dbInfoCache:     dbInfoCache,
		qualityAnalyzer: qualityAnalyzer,
		logFunc:         logFunc,
	}
}

// writeXMLResponse записывает XML ответ
func (h *UploadLegacyHandler) writeXMLResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	xmlData, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		h.writeErrorResponse(w, "Failed to marshal XML", err)
		return
	}

	w.Write([]byte(xml.Header))
	w.Write(xmlData)
}

// writeErrorResponse записывает ошибку в XML формате
func (h *UploadLegacyHandler) writeErrorResponse(w http.ResponseWriter, message string, err error) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	response := models.ErrorResponse{
		Success:   false,
		Error:     "",
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response.Error = err.Error()
	}

	xmlData, _ := xml.MarshalIndent(response, "", "  ")
	w.Write([]byte(xml.Header))
	w.Write(xmlData)
}

// HandleHandshake обрабатывает рукопожатие
// POST /handshake
func (h *UploadLegacyHandler) HandleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req types.HandshakeRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Валидация обязательных полей
	if err := ValidateHandshakeRequest(req.Version1C, req.ConfigName); err != nil {
		h.writeErrorResponse(w, fmt.Sprintf("Missing required field: %s", err.Error()), err)
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

	// Создаем новую выгрузку
	uploadUUID := uuid.New().String()

	// Определяем database_id с приоритетами:
	// 1. Прямой database_id из запроса (если указан)
	// 2. Автоматический поиск по косвенным параметрам (computer_name, user_name, config_name, version_1c)
	databaseID, identifiedBy, similarUpload, err := ResolveDatabaseID(
		req.DatabaseID,
		req.ComputerName,
		req.UserName,
		req.ConfigName,
		req.Version1C,
		req.ConfigVersion,
		h.db,
	)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp:  time.Now(),
			Level:      "WARN",
			Message:    fmt.Sprintf("Error resolving database ID: %v", err),
			UploadUUID: uploadUUID,
			Endpoint:   "/handshake",
		})
	}

	// Получаем информацию о базе данных, проекте и клиенте
	var clientName, projectName, databaseName string
	if databaseID != nil {
		clientName, projectName, databaseName, err = GetDatabaseInfo(h.serviceDB, *databaseID, h.dbInfoCache)
		if err != nil {
			// Логируем ошибку, но продолжаем работу
			h.logFunc(types.LogEntry{
				Timestamp:  time.Now(),
				Level:      "WARN",
				Message:    fmt.Sprintf("Failed to get database info: %v", err),
				UploadUUID: uploadUUID,
				Endpoint:   "/handshake",
			})
		}

		// Логируем успешную автоматическую идентификацию
		if identifiedBy != "" && strings.HasPrefix(identifiedBy, "similar_upload_") {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message: fmt.Sprintf("Auto-identified database_id=%d from similar upload (computer=%s, config=%s, version=%s)",
					*databaseID, req.ComputerName, req.ConfigName, req.Version1C),
				UploadUUID: uploadUUID,
				Endpoint:   "/handshake",
			})
		}
	} else {
		// Логируем, что автоматическая идентификация не удалась
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message: fmt.Sprintf("Could not auto-identify database (computer=%s, config=%s, version=%s)",
				req.ComputerName, req.ConfigName, req.Version1C),
			UploadUUID: uploadUUID,
			Endpoint:   "/handshake",
		})
	}

	// Определяем parent_upload_id если указан ParentUploadID (UUID)
	var parentUploadID *int
	if req.ParentUploadID != "" {
		parentUpload, err := h.db.GetUploadByUUID(req.ParentUploadID)
		if err == nil {
			parentUploadID = &parentUpload.ID
		}
	}

	// Устанавливаем значения по умолчанию для полей итераций
	iterationNumber := NormalizeIterationNumber(req.IterationNumber)

	// Создаем выгрузку с привязкой к базе данных и полями итераций
	upload, err := h.db.CreateUploadWithDatabase(
		uploadUUID, req.Version1C, req.ConfigName, databaseID,
		req.ComputerName, req.UserName, req.ConfigVersion,
		iterationNumber, req.IterationLabel, req.ProgrammerName, req.UploadPurpose, parentUploadID,
	)
	if err != nil {
		h.writeErrorResponse(w, "Failed to create upload", err)
		return
	}

	// Обновляем кэшированные значения client_id и project_id
	if databaseID != nil {
		var clientID, projectID int

		// Если идентификация была по похожей выгрузке, используем её значения
		if identifiedBy != "" && strings.HasPrefix(identifiedBy, "similar_upload_") && similarUpload != nil {
			if similarUpload.ClientID != nil {
				clientID = *similarUpload.ClientID
			}
			if similarUpload.ProjectID != nil {
				projectID = *similarUpload.ProjectID
			}
		}

		// Если не получили из похожей выгрузки, получаем из serviceDB
		if clientID == 0 || projectID == 0 {
			var err error
			clientID, projectID, err = GetClientProjectIDs(h.serviceDB, *databaseID, h.dbInfoCache)
			if err != nil {
				h.logFunc(types.LogEntry{
					Timestamp:  time.Now(),
					Level:      "WARN",
					Message:    fmt.Sprintf("Failed to get client/project IDs: %v", err),
					UploadUUID: uploadUUID,
					Endpoint:   "/handshake",
				})
			}
		}

		// Обновляем upload с кэшированными значениями
		if clientID > 0 && projectID > 0 {
			_, err = h.db.Exec(`
				UPDATE uploads 
				SET client_id = ?, project_id = ? 
				WHERE id = ?
			`, clientID, projectID, upload.ID)
			if err != nil {
				// Логируем ошибку, но не прерываем процесс
				h.logFunc(types.LogEntry{
					Timestamp:  time.Now(),
					Level:      "WARNING",
					Message:    fmt.Sprintf("Failed to update cached client_id and project_id: %v", err),
					UploadUUID: uploadUUID,
					Endpoint:   "/handshake",
				})
			}
		}
	}

	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("Handshake successful for upload %s (database_id: %v, identified_by: %s, iteration_number: %d, iteration_label: %s, programmer: %s, purpose: %s, parent_upload_id: %v)",
			uploadUUID, databaseID, identifiedBy, upload.IterationNumber, upload.IterationLabel, upload.ProgrammerName, upload.UploadPurpose, upload.ParentUploadID),
		UploadUUID: uploadUUID,
		Endpoint:   "/handshake",
	})

	response := types.HandshakeResponse{
		Success:      true,
		UploadUUID:   uploadUUID,
		ClientName:   clientName,
		ProjectName:  projectName,
		DatabaseName: databaseName,
		Message:      "Handshake successful",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	h.writeXMLResponse(w, response)
}

// HandleMetadata обрабатывает метаинформацию
// POST /metadata
func (h *UploadLegacyHandler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req types.MetadataRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Проверяем существование выгрузки
	_, err = h.db.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		h.writeErrorResponse(w, "Upload not found", err)
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

	h.writeXMLResponse(w, response)
}

// HandleConstant обрабатывает константу
// POST /constant
func (h *UploadLegacyHandler) HandleConstant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
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
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Логирование распарсенных данных
	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "DEBUG",
		Message:   fmt.Sprintf("Parsed constant - Name: %s, Type: %s, Value.Content length: %d, Value.Content: %s", req.Name, req.Type, len(req.Value.Content), req.Value.Content),
		Endpoint:  "/constant",
	})

	// Получаем выгрузку
	upload, err := h.db.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		h.writeErrorResponse(w, "Upload not found", err)
		return
	}

	// Добавляем константу
	// req.Value теперь структура ConstantValue, используем Content для получения XML строки
	valueContent := req.Value.Content
	if err := h.db.AddConstant(upload.ID, req.Name, req.Synonym, req.Type, valueContent); err != nil {
		h.writeErrorResponse(w, "Failed to add constant", err)
		return
	}

	// Логирование для отладки (логируем первые 100 символов значения)
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

	h.writeXMLResponse(w, response)
}

// HandleCatalogMeta обрабатывает метаданные справочника
// POST /catalog/meta
func (h *UploadLegacyHandler) HandleCatalogMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req types.CatalogMetaRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку
	upload, err := h.db.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		h.writeErrorResponse(w, "Upload not found", err)
		return
	}

	// Добавляем справочник
	catalog, err := h.db.AddCatalog(upload.ID, req.Name, req.Synonym)
	if err != nil {
		h.writeErrorResponse(w, "Failed to add catalog", err)
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

	h.writeXMLResponse(w, response)
}

// HandleCatalogItem обрабатывает элемент справочника
// POST /catalog/item
func (h *UploadLegacyHandler) HandleCatalogItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req types.CatalogItemRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку
	upload, err := h.db.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		h.writeErrorResponse(w, "Upload not found", err)
		return
	}

	// Находим справочник по имени
	var catalogID int
	err = h.db.QueryRow("SELECT id FROM catalogs WHERE upload_id = ? AND name = ?", upload.ID, req.CatalogName).Scan(&catalogID)
	if err != nil {
		h.writeErrorResponse(w, "Catalog not found", err)
		return
	}

	// Attributes и TableParts уже приходят как XML строки из 1С
	// Передаем их напрямую как строки
	if err := h.db.AddCatalogItem(catalogID, req.Reference, req.Code, req.Name, req.Attributes, req.TableParts); err != nil {
		h.writeErrorResponse(w, "Failed to add catalog item", err)
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

	h.writeXMLResponse(w, response)
}

// HandleCatalogItems обрабатывает пакетную загрузку элементов справочника
// POST /catalog/items
func (h *UploadLegacyHandler) HandleCatalogItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req types.CatalogItemsRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку
	upload, err := h.db.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		h.writeErrorResponse(w, "Upload not found", err)
		return
	}

	// Находим справочник по имени
	var catalogID int
	err = h.db.QueryRow("SELECT id FROM catalogs WHERE upload_id = ? AND name = ?", upload.ID, req.CatalogName).Scan(&catalogID)
	if err != nil {
		h.writeErrorResponse(w, "Catalog not found", err)
		return
	}

	// Обрабатываем каждый элемент пакета
	processedCount := 0
	failedCount := 0

	for _, item := range req.Items {
		if err := h.db.AddCatalogItem(catalogID, item.Reference, item.Code, item.Name, item.Attributes, item.TableParts); err != nil {
			failedCount++
			h.logFunc(types.LogEntry{
				Timestamp:  time.Now(),
				Level:      "ERROR",
				Message:    fmt.Sprintf("Failed to add catalog item '%s': %v", item.Name, err),
				UploadUUID: req.UploadUUID,
				Endpoint:   "/catalog/items",
			})
		} else {
			processedCount++
		}
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

	h.writeXMLResponse(w, response)
}

// HandleNomenclatureBatch обрабатывает пакетную загрузку номенклатуры с характеристиками
// POST /api/v1/upload/nomenclature/batch
func (h *UploadLegacyHandler) HandleNomenclatureBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req types.NomenclatureBatchRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку
	upload, err := h.db.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		h.writeErrorResponse(w, "Upload not found", err)
		return
	}

	// Преобразуем элементы в формат для базы данных
	nomenclatureItems := make([]database.NomenclatureItem, 0, len(req.Items))
	for _, item := range req.Items {
		nomenclatureItems = append(nomenclatureItems, database.NomenclatureItem{
			NomenclatureReference:   item.NomenclatureReference,
			NomenclatureCode:        item.NomenclatureCode,
			NomenclatureName:        item.NomenclatureName,
			CharacteristicReference: item.CharacteristicReference,
			CharacteristicName:      item.CharacteristicName,
			AttributesXML:           item.Attributes,
			TablePartsXML:           item.TableParts,
		})
	}

	// Добавляем пакет элементов номенклатуры
	if err := h.db.AddNomenclatureItemsBatch(upload.ID, nomenclatureItems); err != nil {
		h.writeErrorResponse(w, "Failed to add nomenclature items", err)
		return
	}

	processedCount := len(req.Items)

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

	h.writeXMLResponse(w, response)
}

// HandleComplete обрабатывает завершение выгрузки
// POST /complete
func (h *UploadLegacyHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req types.CompleteRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		h.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку
	upload, err := h.db.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		h.writeErrorResponse(w, "Upload not found", err)
		return
	}

	// Завершаем выгрузку
	if err := h.db.CompleteUpload(upload.ID); err != nil {
		h.writeErrorResponse(w, "Failed to complete upload", err)
		return
	}

	h.logFunc(types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Upload %s completed successfully", req.UploadUUID),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/complete",
	})

	// Запускаем анализ качества в фоне
	go func() {
		databaseID := 0
		if upload.DatabaseID != nil {
			databaseID = *upload.DatabaseID
		}

		if databaseID > 0 {
			log.Printf("Starting quality analysis for upload %s (ID: %d, Database: %d)", req.UploadUUID, upload.ID, databaseID)
			if err := h.qualityAnalyzer.AnalyzeUpload(upload.ID, databaseID); err != nil {
				log.Printf("Quality analysis failed for upload %s: %v", req.UploadUUID, err)
			} else {
				log.Printf("Quality analysis completed for upload %s", req.UploadUUID)
			}
		} else {
			log.Printf("Skipping quality analysis for upload %s: database_id not set", req.UploadUUID)
		}
	}()

	response := types.CompleteResponse{
		Success:   true,
		Message:   "Upload completed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	h.writeXMLResponse(w, response)
}

