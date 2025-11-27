package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл содержит Upload и Normalized handlers, извлеченные из server.go
// для сокращения размера server.go

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"httpserver/database"
)
// handleGetUpload обрабатывает запрос детальной информации о выгрузке
func (s *Server) handleGetUpload(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем детали выгрузки
	_, catalogs, constants, err := s.db.GetUploadDetails(upload.UploadUUID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get upload details: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем количество элементов для каждого справочника
	itemCounts, err := s.db.GetCatalogItemCountByCatalog(upload.ID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get catalog item counts: %v", err), http.StatusInternalServerError)
		return
	}

	catalogInfos := make([]CatalogInfo, len(catalogs))
	for i, catalog := range catalogs {
		catalogInfos[i] = CatalogInfo{
			ID:        catalog.ID,
			Name:      catalog.Name,
			Synonym:   catalog.Synonym,
			ItemCount: itemCounts[catalog.ID],
			CreatedAt: catalog.CreatedAt,
		}
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

	details := UploadDetails{
		UploadUUID:     upload.UploadUUID,
		StartedAt:      upload.StartedAt,
		CompletedAt:    upload.CompletedAt,
		Status:         upload.Status,
		Version1C:      upload.Version1C,
		ConfigName:     upload.ConfigName,
		TotalConstants: upload.TotalConstants,
		TotalCatalogs:  upload.TotalCatalogs,
		TotalItems:     upload.TotalItems,
		Catalogs:       catalogInfos,
		Constants:      constantData,
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Upload details requested for %s", upload.UploadUUID),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/uploads/{uuid}",
	})

	s.writeJSONResponse(w, r, details, http.StatusOK)
}

// handleGetUploadData обрабатывает запрос данных выгрузки с фильтрацией и пагинацией
func (s *Server) handleGetUploadData(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	if r.Method != http.MethodGet {
		s.handleHTTPError(w, r, NewValidationError("Метод не разрешен", nil))
		return
	}

	// Парсим query параметры
	dataType := r.URL.Query().Get("type")
	if dataType == "" {
		dataType = "all"
	}

	catalogNamesStr := r.URL.Query().Get("catalog_names")
	var catalogNames []string
	if catalogNamesStr != "" {
		catalogNames = strings.Split(catalogNamesStr, ",")
		for i := range catalogNames {
			catalogNames[i] = strings.TrimSpace(catalogNames[i])
		}
	}

	// Проверяем поддержку Flusher ДО установки заголовков
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeJSONError(w, r, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Stream started for upload %s, type=%s", upload.UploadUUID, dataType),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/uploads/{uuid}/stream",
	})

	// Функция для экранирования XML
	escapeXML := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&apos;")
		return s
	}

	// Отправляем константы
	if dataType == "constants" || dataType == "all" {
		constants, err := s.db.GetConstantsByUpload(upload.ID)
		if err == nil {
			for _, constant := range constants {
				// Формируем XML для константы - включаем все поля из БД
				dataXML := fmt.Sprintf(`<constant><id>%d</id><upload_id>%d</upload_id><name>%s</name><synonym>%s</synonym><type>%s</type><value>%s</value><created_at>%s</created_at></constant>`,
					constant.ID, constant.UploadID, escapeXML(constant.Name), escapeXML(constant.Synonym),
					escapeXML(constant.Type), escapeXML(constant.Value), constant.CreatedAt.Format(time.RFC3339))

				item := DataItem{
					Type:      "constant",
					ID:        constant.ID,
					Data:      dataXML,
					CreatedAt: constant.CreatedAt,
				}

				// Отправляем как XML
				xmlData, _ := xml.Marshal(item)
				fmt.Fprintf(w, "data: %s\n\n", string(xmlData))
				flusher.Flush()
			}
		}
	}

	// Отправляем элементы справочников
	if dataType == "catalogs" || dataType == "all" {
		offset := 0
		limit := 100

		for {
			items, _, err := s.db.GetCatalogItemsByUpload(upload.ID, catalogNames, offset, limit)
			if err != nil || len(items) == 0 {
				break
			}

			for _, itemData := range items {
				// Формируем XML для элемента справочника
				// Включаем все поля из БД: id, catalog_id, catalog_name, reference, code, name, attributes_xml, table_parts_xml, created_at
				// attributes_xml и table_parts_xml уже содержат XML, вставляем их как есть (innerXML)
				dataXML := fmt.Sprintf(`<catalog_item><id>%d</id><catalog_id>%d</catalog_id><catalog_name>%s</catalog_name><reference>%s</reference><code>%s</code><name>%s</name><attributes_xml>%s</attributes_xml><table_parts_xml>%s</table_parts_xml><created_at>%s</created_at></catalog_item>`,
					itemData.ID, itemData.CatalogID, escapeXML(itemData.CatalogName),
					escapeXML(itemData.Reference), escapeXML(itemData.Code), escapeXML(itemData.Name),
					itemData.Attributes, itemData.TableParts, itemData.CreatedAt.Format(time.RFC3339))

				dataItem := DataItem{
					Type:      "catalog_item",
					ID:        itemData.ID,
					Data:      dataXML,
					CreatedAt: itemData.CreatedAt,
				}

				// Отправляем как XML
				xmlData, _ := xml.Marshal(dataItem)
				fmt.Fprintf(w, "data: %s\n\n", string(xmlData))
				flusher.Flush()
			}

			if len(items) < limit {
				break
			}

			offset += limit
		}
	}

	// Отправляем завершающее сообщение
	fmt.Fprintf(w, "data: {\"type\":\"complete\"}\n\n")
	flusher.Flush()
}

// handleStreamUploadData - алиас для handleGetUploadData (оба делают стриминг)
func (s *Server) handleStreamUploadData(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	s.handleGetUploadData(w, r, upload)
}

// handleVerifyUpload обрабатывает проверку успешной передачи
func (s *Server) handleVerifyUpload(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	if r.Method != http.MethodPost {
		s.handleHTTPError(w, r, NewValidationError("Метод не разрешен", nil))
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogError(r.Context(), err, "Failed to parse verify request body", "upload_uuid", upload.UploadUUID)
		s.handleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	// Получаем все ID элементов выгрузки
	receivedSet := make(map[int]bool)
	for _, id := range req.ReceivedIDs {
		receivedSet[id] = true
	}

	// Получаем все константы
	constants, err := s.db.GetConstantsByUpload(upload.ID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get constants for verify", "upload_id", upload.ID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить константы", err))
		return
	}

	// Получаем все элементы справочников
	catalogItems, _, err := s.db.GetCatalogItemsByUpload(upload.ID, nil, 0, 0)
	if err != nil {
		LogError(r.Context(), err, "Failed to get catalog items for verify", "upload_id", upload.ID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить элементы справочника", err))
		return
	}

	// Собираем все ожидаемые ID
	expectedSet := make(map[int]bool)
	for _, constant := range constants {
		expectedSet[constant.ID] = true
	}
	for _, item := range catalogItems {
		expectedSet[item.ID] = true
	}

	// Находим отсутствующие ID
	var missingIDs []int
	for id := range expectedSet {
		if !receivedSet[id] {
			missingIDs = append(missingIDs, id)
		}
	}

	expectedTotal := len(expectedSet)
	receivedCount := len(req.ReceivedIDs)
	isComplete := len(missingIDs) == 0

	message := fmt.Sprintf("Received %d of %d items", receivedCount, expectedTotal)
	if !isComplete {
		message += fmt.Sprintf(", %d items missing", len(missingIDs))
	} else {
		message += ", all items received"
	}

	response := VerifyResponse{
		UploadUUID:    upload.UploadUUID,
		ExpectedTotal: expectedTotal,
		ReceivedCount: receivedCount,
		MissingIDs:    missingIDs,
		IsComplete:    isComplete,
		Message:       message,
	}

	LogInfo(r.Context(), "Verify requested", "upload_uuid", upload.UploadUUID, "message", message)

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetUploadNormalized обрабатывает запрос детальной информации о выгрузке из нормализованной БД
func (s *Server) handleGetUploadNormalized(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	if r.Method != http.MethodGet {
		s.handleHTTPError(w, r, NewValidationError("Метод не разрешен", nil))
		return
	}

	// Получаем детали выгрузки из нормализованной БД
	_, catalogs, constants, err := s.normalizedDB.GetUploadDetails(upload.UploadUUID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get normalized upload details", "upload_uuid", upload.UploadUUID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить детали нормализованной выгрузки", err))
		return
	}

	// Получаем количество элементов для каждого справочника
	itemCounts, err := s.normalizedDB.GetCatalogItemCountByCatalog(upload.ID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get catalog item counts for normalized upload", "upload_id", upload.ID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить количество элементов справочника", err))
		return
	}

	catalogInfos := make([]CatalogInfo, len(catalogs))
	for i, catalog := range catalogs {
		catalogInfos[i] = CatalogInfo{
			ID:        catalog.ID,
			Name:      catalog.Name,
			Synonym:   catalog.Synonym,
			ItemCount: itemCounts[catalog.ID],
			CreatedAt: catalog.CreatedAt,
		}
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

	details := UploadDetails{
		UploadUUID:     upload.UploadUUID,
		StartedAt:      upload.StartedAt,
		CompletedAt:    upload.CompletedAt,
		Status:         upload.Status,
		Version1C:      upload.Version1C,
		ConfigName:     upload.ConfigName,
		TotalConstants: upload.TotalConstants,
		TotalCatalogs:  upload.TotalCatalogs,
		TotalItems:     upload.TotalItems,
		Catalogs:       catalogInfos,
		Constants:      constantData,
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized upload details requested for %s", upload.UploadUUID),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/normalized/uploads/{uuid}",
	})

	s.writeJSONResponse(w, r, details, http.StatusOK)
}

// handleGetUploadDataNormalized обрабатывает запрос данных выгрузки из нормализованной БД
func (s *Server) handleGetUploadDataNormalized(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим query параметры
	dataType := r.URL.Query().Get("type")
	if dataType == "" {
		dataType = "all"
	}

	catalogNamesStr := r.URL.Query().Get("catalog_names")
	var catalogNames []string
	if catalogNamesStr != "" {
		catalogNames = strings.Split(catalogNamesStr, ",")
		for i := range catalogNames {
			catalogNames[i] = strings.TrimSpace(catalogNames[i])
		}
	}

	// Валидация параметров пагинации
	page, err := ValidateIntParam(r, "page", 1, 1, 0)
	if err != nil {
		page = 1 // Используем значение по умолчанию при ошибке
	}

	limit, err := ValidateIntParam(r, "limit", 100, 1, 1000)
	if err != nil {
		limit = 100 // Используем значение по умолчанию при ошибке
	}

	offset := (page - 1) * limit

	var responseItems []DataItem
	var total int

	// Функция для экранирования XML
	escapeXML := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&apos;")
		return s
	}

	// Получаем данные в зависимости от типа из нормализованной БД
	if dataType == "constants" {
		constants, err := s.normalizedDB.GetConstantsByUpload(upload.ID)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get constants: %v", err), http.StatusInternalServerError)
			return
		}

		total = len(constants)

		// Применяем пагинацию для констант
		start := offset
		end := offset + limit
		if start > len(constants) {
			start = len(constants)
		}
		if end > len(constants) {
			end = len(constants)
		}

		for i := start; i < end; i++ {
			constData := constants[i]
			// Формируем XML для константы - включаем все поля из БД
			dataXML := fmt.Sprintf(`<constant><id>%d</id><upload_id>%d</upload_id><name>%s</name><synonym>%s</synonym><type>%s</type><value>%s</value><created_at>%s</created_at></constant>`,
				constData.ID, constData.UploadID, escapeXML(constData.Name), escapeXML(constData.Synonym),
				escapeXML(constData.Type), escapeXML(constData.Value), constData.CreatedAt.Format(time.RFC3339))

			responseItems = append(responseItems, DataItem{
				Type:      "constant",
				ID:        constData.ID,
				Data:      dataXML,
				CreatedAt: constData.CreatedAt,
			})
		}
	} else if dataType == "catalogs" {
		catalogItems, itemTotal, err := s.normalizedDB.GetCatalogItemsByUpload(upload.ID, catalogNames, offset, limit)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get catalog items: %v", err), http.StatusInternalServerError)
			return
		}

		total = itemTotal

		for _, itemData := range catalogItems {
			// Формируем XML для элемента справочника
			// Включаем все поля из БД: id, catalog_id, catalog_name, reference, code, name, attributes_xml, table_parts_xml, created_at
			// attributes_xml и table_parts_xml уже содержат XML, вставляем их как есть (innerXML)
			dataXML := fmt.Sprintf(`<catalog_item><id>%d</id><catalog_id>%d</catalog_id><catalog_name>%s</catalog_name><reference>%s</reference><code>%s</code><name>%s</name><attributes_xml>%s</attributes_xml><table_parts_xml>%s</table_parts_xml><created_at>%s</created_at></catalog_item>`,
				itemData.ID, itemData.CatalogID, escapeXML(itemData.CatalogName),
				escapeXML(itemData.Reference), escapeXML(itemData.Code), escapeXML(itemData.Name),
				itemData.Attributes, itemData.TableParts, itemData.CreatedAt.Format(time.RFC3339))

			responseItems = append(responseItems, DataItem{
				Type:      "catalog_item",
				ID:        itemData.ID,
				Data:      dataXML,
				CreatedAt: itemData.CreatedAt,
			})
		}
	} else { // dataType == "all"
		// Для "all" сначала получаем все константы и элементы
		constants, err := s.normalizedDB.GetConstantsByUpload(upload.ID)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get constants: %v", err), http.StatusInternalServerError)
			return
		}

		catalogItems, itemTotal, err := s.normalizedDB.GetCatalogItemsByUpload(upload.ID, catalogNames, 0, 0)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get catalog items: %v", err), http.StatusInternalServerError)
			return
		}

		total = len(constants) + itemTotal

		// Объединяем все элементы и применяем пагинацию
		allItems := make([]DataItem, 0, total)

		// Добавляем константы - включаем все поля из БД
		for _, constant := range constants {
			dataXML := fmt.Sprintf(`<constant><id>%d</id><upload_id>%d</upload_id><name>%s</name><synonym>%s</synonym><type>%s</type><value>%s</value><created_at>%s</created_at></constant>`,
				constant.ID, constant.UploadID, escapeXML(constant.Name), escapeXML(constant.Synonym),
				escapeXML(constant.Type), escapeXML(constant.Value), constant.CreatedAt.Format(time.RFC3339))

			allItems = append(allItems, DataItem{
				Type:      "constant",
				ID:        constant.ID,
				Data:      dataXML,
				CreatedAt: constant.CreatedAt,
			})
		}

		// Добавляем элементы справочников
		for _, itemData := range catalogItems {
			// Включаем все поля из БД: id, catalog_id, catalog_name, reference, code, name, attributes_xml, table_parts_xml, created_at
			// attributes_xml и table_parts_xml уже содержат XML, вставляем их как есть (innerXML)
			dataXML := fmt.Sprintf(`<catalog_item><id>%d</id><catalog_id>%d</catalog_id><catalog_name>%s</catalog_name><reference>%s</reference><code>%s</code><name>%s</name><attributes_xml>%s</attributes_xml><table_parts_xml>%s</table_parts_xml><created_at>%s</created_at></catalog_item>`,
				itemData.ID, itemData.CatalogID, escapeXML(itemData.CatalogName),
				escapeXML(itemData.Reference), escapeXML(itemData.Code), escapeXML(itemData.Name),
				itemData.Attributes, itemData.TableParts, itemData.CreatedAt.Format(time.RFC3339))

			allItems = append(allItems, DataItem{
				Type:      "catalog_item",
				ID:        itemData.ID,
				Data:      dataXML,
				CreatedAt: itemData.CreatedAt,
			})
		}

		// Применяем пагинацию
		start := offset
		end := offset + limit
		if start > len(allItems) {
			start = len(allItems)
		}
		if end > len(allItems) {
			end = len(allItems)
		}

		responseItems = allItems[start:end]
	}

	// Формируем XML ответ
	response := DataResponse{
		UploadUUID: upload.UploadUUID,
		Type:       dataType,
		Page:       page,
		Limit:      limit,
		Total:      total,
		Items:      responseItems,
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized upload data requested for %s, type=%s, returned %d items", upload.UploadUUID, dataType, len(responseItems)),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/normalized/uploads/{uuid}/data",
	})

	s.writeXMLResponse(w, response)
}

// handleStreamUploadDataNormalized обрабатывает потоковую отправку данных из нормализованной БД через SSE
func (s *Server) handleStreamUploadDataNormalized(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим query параметры
	dataType := r.URL.Query().Get("type")
	if dataType == "" {
		dataType = "all"
	}

	catalogNamesStr := r.URL.Query().Get("catalog_names")
	var catalogNames []string
	if catalogNamesStr != "" {
		catalogNames = strings.Split(catalogNamesStr, ",")
		for i := range catalogNames {
			catalogNames[i] = strings.TrimSpace(catalogNames[i])
		}
	}

	// Проверяем поддержку Flusher ДО установки заголовков
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeJSONError(w, r, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized stream started for upload %s, type=%s", upload.UploadUUID, dataType),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/normalized/uploads/{uuid}/stream",
	})

	// Функция для экранирования XML
	escapeXML := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&apos;")
		return s
	}

	// Отправляем константы из нормализованной БД
	if dataType == "constants" || dataType == "all" {
		constants, err := s.normalizedDB.GetConstantsByUpload(upload.ID)
		if err == nil {
			for _, constant := range constants {
				// Формируем XML для константы - включаем все поля из БД
				dataXML := fmt.Sprintf(`<constant><id>%d</id><upload_id>%d</upload_id><name>%s</name><synonym>%s</synonym><type>%s</type><value>%s</value><created_at>%s</created_at></constant>`,
					constant.ID, constant.UploadID, escapeXML(constant.Name), escapeXML(constant.Synonym),
					escapeXML(constant.Type), escapeXML(constant.Value), constant.CreatedAt.Format(time.RFC3339))

				item := DataItem{
					Type:      "constant",
					ID:        constant.ID,
					Data:      dataXML,
					CreatedAt: constant.CreatedAt,
				}

				// Отправляем как XML
				xmlData, _ := xml.Marshal(item)
				fmt.Fprintf(w, "data: %s\n\n", string(xmlData))
				flusher.Flush()
			}
		}
	}

	// Отправляем элементы справочников из нормализованной БД
	if dataType == "catalogs" || dataType == "all" {
		offset := 0
		limit := 100

		for {
			items, _, err := s.normalizedDB.GetCatalogItemsByUpload(upload.ID, catalogNames, offset, limit)
			if err != nil || len(items) == 0 {
				break
			}

			for _, itemData := range items {
				// Формируем XML для элемента справочника
				// Включаем все поля из БД: id, catalog_id, catalog_name, reference, code, name, attributes_xml, table_parts_xml, created_at
				// attributes_xml и table_parts_xml уже содержат XML, вставляем их как есть (innerXML)
				dataXML := fmt.Sprintf(`<catalog_item><id>%d</id><catalog_id>%d</catalog_id><catalog_name>%s</catalog_name><reference>%s</reference><code>%s</code><name>%s</name><attributes_xml>%s</attributes_xml><table_parts_xml>%s</table_parts_xml><created_at>%s</created_at></catalog_item>`,
					itemData.ID, itemData.CatalogID, escapeXML(itemData.CatalogName),
					escapeXML(itemData.Reference), escapeXML(itemData.Code), escapeXML(itemData.Name),
					itemData.Attributes, itemData.TableParts, itemData.CreatedAt.Format(time.RFC3339))

				dataItem := DataItem{
					Type:      "catalog_item",
					ID:        itemData.ID,
					Data:      dataXML,
					CreatedAt: itemData.CreatedAt,
				}

				// Отправляем как XML
				xmlData, err := xml.Marshal(dataItem)
				if err != nil {
					log.Printf("[StreamUploadNormalized] Error marshaling catalog item: %v", err)
					continue
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(xmlData)); err != nil {
					log.Printf("[StreamUploadNormalized] Error sending catalog item data: %v", err)
					return
				}
				flusher.Flush()
			}

			if len(items) < limit {
				break
			}

			offset += limit
		}
	}

	// Отправляем завершающее сообщение
	if _, err := fmt.Fprintf(w, "data: {\"type\":\"complete\"}\n\n"); err != nil {
		log.Printf("[StreamUploadNormalized] Error sending complete message: %v", err)
		return
	}
	flusher.Flush()
}

// handleVerifyUploadNormalized обрабатывает проверку успешной передачи для нормализованной БД
func (s *Server) handleVerifyUploadNormalized(w http.ResponseWriter, r *http.Request, upload *database.Upload) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	// Получаем все ID элементов выгрузки из нормализованной БД
	receivedSet := make(map[int]bool)
	for _, id := range req.ReceivedIDs {
		receivedSet[id] = true
	}

	// Получаем все константы
	constants, err := s.normalizedDB.GetConstantsByUpload(upload.ID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get constants: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем все элементы справочников
	catalogItems, _, err := s.normalizedDB.GetCatalogItemsByUpload(upload.ID, nil, 0, 0)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get catalog items: %v", err), http.StatusInternalServerError)
		return
	}

	// Собираем все ожидаемые ID
	expectedSet := make(map[int]bool)
	for _, constant := range constants {
		expectedSet[constant.ID] = true
	}
	for _, item := range catalogItems {
		expectedSet[item.ID] = true
	}

	// Находим отсутствующие ID
	var missingIDs []int
	for id := range expectedSet {
		if !receivedSet[id] {
			missingIDs = append(missingIDs, id)
		}
	}

	expectedTotal := len(expectedSet)
	receivedCount := len(req.ReceivedIDs)
	isComplete := len(missingIDs) == 0

	message := fmt.Sprintf("Received %d of %d items", receivedCount, expectedTotal)
	if !isComplete {
		message += fmt.Sprintf(", %d items missing", len(missingIDs))
	} else {
		message += ", all items received"
	}

	response := VerifyResponse{
		UploadUUID:    upload.UploadUUID,
		ExpectedTotal: expectedTotal,
		ReceivedCount: receivedCount,
		MissingIDs:    missingIDs,
		IsComplete:    isComplete,
		Message:       message,
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized verify requested for upload %s: %s", upload.UploadUUID, message),
		UploadUUID: upload.UploadUUID,
		Endpoint:   "/api/normalized/uploads/{uuid}/verify",
	})

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleNormalizedHandshake обрабатывает рукопожатие для нормализованных данных
func (s *Server) handleNormalizedHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req HandshakeRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Создаем новую выгрузку в нормализованной БД
	uploadUUID := uuid.New().String()
	_, err = s.normalizedDB.CreateUpload(uploadUUID, req.Version1C, req.ConfigName)
	if err != nil {
		s.writeErrorResponse(w, "Failed to create normalized upload", err)
		return
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized handshake successful for upload %s", uploadUUID),
		UploadUUID: uploadUUID,
		Endpoint:   "/api/normalized/upload/handshake",
	})

	response := HandshakeResponse{
		Success:    true,
		UploadUUID: uploadUUID,
		Message:    "Normalized handshake successful",
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}

// handleNormalizedMetadata обрабатывает метаинформацию для нормализованных данных
func (s *Server) handleNormalizedMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req MetadataRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Проверяем существование выгрузки в нормализованной БД
	_, err = s.normalizedDB.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		s.writeErrorResponse(w, "Normalized upload not found", err)
		return
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    "Normalized metadata received successfully",
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/normalized/upload/metadata",
	})

	response := MetadataResponse{
		Success:   true,
		Message:   "Normalized metadata received successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}

// handleNormalizedConstant обрабатывает константу для нормализованных данных
func (s *Server) handleNormalizedConstant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req ConstantRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку из нормализованной БД
	upload, err := s.normalizedDB.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		s.writeErrorResponse(w, "Normalized upload not found", err)
		return
	}

	// Добавляем константу в нормализованную БД
	// req.Value теперь структура ConstantValue, используем Content для получения XML строки
	valueContent := req.Value.Content
	if err := s.normalizedDB.AddConstant(upload.ID, req.Name, req.Synonym, req.Type, valueContent); err != nil {
		s.writeErrorResponse(w, "Failed to add normalized constant", err)
		return
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized constant '%s' added successfully", req.Name),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/normalized/upload/constant",
	})

	response := ConstantResponse{
		Success:   true,
		Message:   "Normalized constant added successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}

// handleNormalizedCatalogMeta обрабатывает метаданные справочника для нормализованных данных
func (s *Server) handleNormalizedCatalogMeta(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req CatalogMetaRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку из нормализованной БД
	upload, err := s.normalizedDB.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		s.writeErrorResponse(w, "Normalized upload not found", err)
		return
	}

	// Добавляем справочник в нормализованную БД
	catalog, err := s.normalizedDB.AddCatalog(upload.ID, req.Name, req.Synonym)
	if err != nil {
		s.writeErrorResponse(w, "Failed to add normalized catalog", err)
		return
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized catalog '%s' metadata added successfully", req.Name),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/normalized/upload/catalog/meta",
	})

	response := CatalogMetaResponse{
		Success:   true,
		CatalogID: catalog.ID,
		Message:   "Normalized catalog metadata added successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}

// handleNormalizedCatalogItem обрабатывает элемент справочника для нормализованных данных
func (s *Server) handleNormalizedCatalogItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req CatalogItemRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку из нормализованной БД
	upload, err := s.normalizedDB.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		s.writeErrorResponse(w, "Normalized upload not found", err)
		return
	}

	// Находим справочник по имени в нормализованной БД
	var catalogID int
	err = s.normalizedDB.QueryRow("SELECT id FROM catalogs WHERE upload_id = ? AND name = ?", upload.ID, req.CatalogName).Scan(&catalogID)
	if err != nil {
		s.writeErrorResponse(w, "Normalized catalog not found", err)
		return
	}

	// Attributes и TableParts уже приходят как XML строки
	// Передаем их напрямую как строки
	if err := s.normalizedDB.AddCatalogItem(catalogID, req.Reference, req.Code, req.Name, req.Attributes, req.TableParts); err != nil {
		s.writeErrorResponse(w, "Failed to add normalized catalog item", err)
		return
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized catalog item '%s' added successfully", req.Name),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/normalized/upload/catalog/item",
	})

	response := CatalogItemResponse{
		Success:   true,
		Message:   "Normalized catalog item added successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}

// handleNormalizedComplete обрабатывает завершение выгрузки нормализованных данных
func (s *Server) handleNormalizedComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req CompleteRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Получаем выгрузку из нормализованной БД
	upload, err := s.normalizedDB.GetUploadByUUID(req.UploadUUID)
	if err != nil {
		s.writeErrorResponse(w, "Normalized upload not found", err)
		return
	}

	// Завершаем выгрузку в нормализованной БД
	if err := s.normalizedDB.CompleteUpload(upload.ID); err != nil {
		s.writeErrorResponse(w, "Failed to complete normalized upload", err)
		return
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized upload %s completed successfully", req.UploadUUID),
		UploadUUID: req.UploadUUID,
		Endpoint:   "/api/normalized/upload/complete",
	})

	response := CompleteResponse{
		Success:   true,
		Message:   "Normalized upload completed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}
