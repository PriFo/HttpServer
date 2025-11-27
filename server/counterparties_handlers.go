package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл содержит Counterparties и прочие handlers, извлеченные из server.go
// для сокращения размера server.go

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"httpserver/database"
	"httpserver/normalization"
	"httpserver/server/handlers"
)

// handle1CProcessingXML обрабатывает XML от 1C
func (s *Server) handle1CProcessingXML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем рабочую директорию (директорию, откуда запущен сервер)
	workDir, err := os.Getwd()
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get working directory: %v", err),
			Endpoint:  "/api/1c/processing/xml",
		})
		http.Error(w, fmt.Sprintf("Failed to get working directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Читаем файлы модулей с абсолютными путями
	modulePath := filepath.Join(workDir, "1c_processing", "Module", "Module.bsl")
	moduleCode, err := os.ReadFile(modulePath)
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to read Module.bsl from %s: %v", modulePath, err),
			Endpoint:  "/api/1c/processing/xml",
		})
		http.Error(w, fmt.Sprintf("Failed to read module file: %v", err), http.StatusInternalServerError)
		return
	}

	extensionsPath := filepath.Join(workDir, "1c_module_extensions.bsl")
	extensionsCode, err := os.ReadFile(extensionsPath)
	if err != nil {
		// Расширения могут отсутствовать, используем пустую строку
		extensionsCode = []byte("")
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Extensions file not found at %s, using empty: %v", extensionsPath, err),
			Endpoint:  "/api/1c/processing/xml",
		})
	}

	exportFunctionsPath := filepath.Join(workDir, "1c_export_functions.txt")
	exportFunctionsCode, err := os.ReadFile(exportFunctionsPath)
	if err != nil {
		// Файл может отсутствовать, используем пустую строку
		exportFunctionsCode = []byte("")
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Export functions file not found, using empty: %v", err),
			Endpoint:  "/api/1c/processing/xml",
		})
	}

	// Объединяем код модуля
	fullModuleCode := string(moduleCode)

	// Добавляем код из export_functions, только область ПрограммныйИнтерфейс
	if len(exportFunctionsCode) > 0 {
		exportCodeStr := string(exportFunctionsCode)
		startMarker := "#Область ПрограммныйИнтерфейс"
		endMarker := "#КонецОбласти"

		startPos := strings.Index(exportCodeStr, startMarker)
		if startPos >= 0 {
			endPos := strings.Index(exportCodeStr[startPos+len(startMarker):], endMarker)
			if endPos >= 0 {
				endPos += startPos + len(startMarker)
				programInterfaceCode := exportCodeStr[startPos : endPos+len(endMarker)]
				fullModuleCode += "\n\n" + programInterfaceCode
			}
		}
	}

	// Добавляем расширения
	if len(extensionsCode) > 0 {
		fullModuleCode += "\n\n" + string(extensionsCode)
	}

	// Генерируем UUID для обработки
	processingUUID := strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", ""))

	// Код формы (из Python скрипта)
	formModuleCode := `&НаКлиенте
Процедура ПриСозданииНаСервере(Отказ, СтандартнаяОбработка)
	
	// Устанавливаем значения по умолчанию
	Если Объект.АдресСервера = "" Тогда
		Объект.АдресСервера = "http://localhost:9999";
	КонецЕсли;
	
	Если Объект.РазмерПакета = 0 Тогда
		Объект.РазмерПакета = 50;
	КонецЕсли;
	
	Если Объект.ИспользоватьПакетнуюВыгрузку = Неопределено Тогда
		Объект.ИспользоватьПакетнуюВыгрузку = Истина;
	КонецЕсли;
	
КонецПроцедуры

&НаКлиенте
Процедура ПриОткрытии(Отказ)
	// Код инициализации формы
КонецПроцедуры

&НаКлиенте
Процедура ПередЗакрытием(Отказ, СтандартнаяОбработка)
	// Обработка перед закрытием формы
КонецПроцедуры`

	// Создаем единый XML файл с правильной структурой для внешней обработки 1С
	// Используем корневой элемент Configuration с ExternalDataProcessor внутри
	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Configuration xmlns="http://v8.1c.ru/8.1/data/enterprise/current-config" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <Properties>
    <SyncMode>Independent</SyncMode>
    <DataLockControlMode>Managed</DataLockControlMode>
  </Properties>
  <MetaDataObject xmlns="http://v8.1c.ru/8.1/data/enterprise" xmlns:v8="http://v8.1c.ru/8.1/data/core" xsi:type="ExternalDataProcessor">
    <Properties>
      <Name>ВыгрузкаДанныхВСервис</Name>
      <Synonym>
        <v8:item>
          <v8:lang>ru</v8:lang>
          <v8:content>Выгрузка данных в сервис нормализации</v8:content>
        </v8:item>
      </Synonym>
      <Comment>Обработка для выгрузки данных из 1С в сервис нормализации и анализа через HTTP</Comment>
      <DefaultForm>Форма</DefaultForm>
      <Help>
        <v8:item>
          <v8:lang>ru</v8:lang>
          <v8:content>Обработка для выгрузки данных</v8:content>
        </v8:item>
      </Help>
    </Properties>
    <uuid>%s</uuid>
    <module>
      <text><![CDATA[%s]]></text>
    </module>
    <forms>
      <form xsi:type="ManagedForm">
        <Properties>
          <Name>Форма</Name>
          <Synonym>
            <v8:item>
              <v8:lang>ru</v8:lang>
              <v8:content>Форма</v8:content>
            </v8:item>
          </Synonym>
        </Properties>
        <module>
          <text><![CDATA[%s]]></text>
        </module>
      </form>
    </forms>
  </MetaDataObject>
</Configuration>`, processingUUID, fullModuleCode, formModuleCode)

	// Устанавливаем заголовки для скачивания файла
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"1c_processing_%s.xml\"", time.Now().Format("20060102_150405")))
	w.WriteHeader(http.StatusOK)

	// Отправляем XML
	if _, err := w.Write([]byte(xmlContent)); err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to write XML response: %v", err),
			Endpoint:  "/api/1c/processing/xml",
		})
		return
	}

	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Generated 1C processing XML (UUID: %s, module size: %d chars)", processingUUID, len(fullModuleCode)),
		Endpoint:  "/api/1c/processing/xml",
	})
}

// ============================================================================
// Snapshot Handlers
// ============================================================================

// handleSnapshotsRoutes обрабатывает запросы к /api/snapshots

// getModelFromConfig получает модель из WorkerConfigManager с fallback на переменные окружения
func (s *Server) getModelFromConfig() string {
	var model string
	if s.workerConfigManager != nil {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err == nil {
			activeModel, err := s.workerConfigManager.GetActiveModel(provider.Name)
			if err == nil {
				model = activeModel.Name
			} else {
				// Используем дефолтную модель из конфигурации
				config := s.workerConfigManager.GetConfig()
				if defaultModel, ok := config["default_model"].(string); ok {
					model = defaultModel
				}
			}
		}
	}

	// Fallback на переменные окружения, если WorkerConfigManager не доступен
	if model == "" {
		model = os.Getenv("ARLIAI_MODEL")
		if model == "" {
			model = "GLM-4.5-Air" // Последний fallback
		}
	}

	return model
}

// Pending/backup database handlers перемещены в server/database_legacy_handlers.go
// handlePendingDatabases, handlePendingDatabaseRoutes, handleStartIndexing,
// handleBindPendingDatabase, handleCleanupPendingDatabases, handleScanDatabases,
// handleDatabasesFiles, handleFindProjectByDatabase - все эти методы уже определены в database_legacy_handlers.go

// handleNormalizedCounterparties обрабатывает запросы на получение нормализованных контрагентов
func (s *Server) handleNormalizedCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Поддерживаем два режима: по проекту или по клиенту
	projectIDStr := r.URL.Query().Get("project_id")
	clientIDStr := r.URL.Query().Get("client_id")

	var counterparties []*database.NormalizedCounterparty
	var projects []*database.ClientProject
	var totalCount int

	// Получаем параметры пагинации
	page, limit, err := ValidatePaginationParams(r, 1, 100, 1000)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	// Поддержка offset для обратной совместимости
	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		offset, err = ValidateIntParam(r, "offset", 0, 0, 0)
		if err != nil {
			if s.HandleValidationError(w, r, err) {
				return
			}
		}
	} else {
		// Вычисляем offset из page
		offset = (page - 1) * limit
	}

	// Получаем параметры фильтрации
	search := r.URL.Query().Get("search")
	enrichment := r.URL.Query().Get("enrichment")
	subcategory := r.URL.Query().Get("subcategory")

	if clientIDStr != "" {
		// Режим получения по клиенту (все проекты)
		clientID, err := ValidateIDParam(r, "client_id")
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid client_id: %s", err.Error()), http.StatusBadRequest)
			return
		}

		var projectID *int
		if projectIDStr != "" {
			pID, err := ValidateIDParam(r, "project_id")
			if err == nil {
				projectID = &pID
			}
		}

		counterparties, projects, totalCount, err = s.serviceDB.GetNormalizedCounterpartiesByClient(clientID, projectID, offset, limit, search, enrichment, subcategory)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
			return
		}
	} else if projectIDStr != "" {
		// Режим получения по проекту
		projectID, err := ValidateIDParam(r, "project_id")
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid project_id: %s", err.Error()), http.StatusBadRequest)
			return
		}

		// Проверяем существование проекта
		project, err := s.serviceDB.GetClientProject(projectID)
		if err != nil {
			s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
			return
		}

		// Получаем нормализованных контрагентов
		counterparties, totalCount, err = s.serviceDB.GetNormalizedCounterparties(projectID, offset, limit, search, enrichment, subcategory)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get normalized counterparties: %v", err), http.StatusInternalServerError)
			return
		}
		projects = []*database.ClientProject{project}
	} else {
		s.writeJSONError(w, r, "Either project_id or client_id is required", http.StatusBadRequest)
		return
	}

	// Формируем ответ с информацией о проектах
	projectsInfo := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		projectsInfo[i] = map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		}
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"counterparties": counterparties,
		"projects":       projectsInfo,
		"total":          totalCount,
		"offset":         offset,
		"limit":          limit,
		"page":           page,
	}, http.StatusOK)
}

// handleGetAllCounterparties обрабатывает запросы на получение всех контрагентов (из баз и нормализованных)
func (s *Server) handleGetAllCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Валидация обязательного параметра client_id
	clientID, err := ValidateIntParam(r, "client_id", 0, 1, 0)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
		s.writeJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
		return
	}
	if clientID <= 0 {
		s.writeJSONError(w, r, "client_id is required and must be positive", http.StatusBadRequest)
		return
	}

	// Валидация опционального параметра project_id
	var projectID *int
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr != "" {
		pID, err := ValidateIntParam(r, "project_id", 0, 1, 0)
		if err != nil {
			if s.HandleValidationError(w, r, err) {
				return
			}
			s.writeJSONError(w, r, fmt.Sprintf("Invalid project_id: %v", err), http.StatusBadRequest)
			return
		}
		if pID > 0 {
			projectID = &pID
		}
	}

	// Валидация параметров пагинации
	offset, err := ValidateIntParam(r, "offset", 0, 0, 0)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
		offset = 0
	}
	if offset < 0 {
		offset = 0
	}

	limit, err := ValidateIntParam(r, "limit", 100, 1, 100000)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
		limit = 100
	}

	// Валидация параметра поиска
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err := ValidateSearchQuery(search, 500); err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
		search = ""
	}

	// Валидация параметра source
	source := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
	if err := handlers.ValidateEnumParam(source, "source", []string{"database", "normalized"}, false); err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	// Валидация параметров сортировки
	sortBy := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_by")))
	order := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if err := handlers.ValidateSortParams(sortBy, order, []string{"name", "quality", "source", "id", ""}); err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	// Получаем параметры фильтрации по качеству
	var minQuality, maxQuality *float64
	if minQualityStr := r.URL.Query().Get("min_quality"); minQualityStr != "" {
		if q, err := strconv.ParseFloat(minQualityStr, 64); err == nil {
			minQuality = &q
		}
	}
	if maxQualityStr := r.URL.Query().Get("max_quality"); maxQualityStr != "" {
		if q, err := strconv.ParseFloat(maxQualityStr, 64); err == nil {
			maxQuality = &q
		}
	}

	// Логирование запроса
	minQStr := "nil"
	maxQStr := "nil"
	if minQuality != nil {
		minQStr = fmt.Sprintf("%.2f", *minQuality)
	}
	if maxQuality != nil {
		maxQStr = fmt.Sprintf("%.2f", *maxQuality)
	}
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("GetAllCounterparties request - client_id: %d, project_id: %v, offset: %d, limit: %d, search: %q, source: %q, sort_by: %q, order: %q, min_quality: %s, max_quality: %s",
			clientID, projectID, offset, limit, search, source, sortBy, order, minQStr, maxQStr),
		Endpoint: "/api/counterparties/all",
	})

	// Получаем всех контрагентов
	result, err := s.serviceDB.GetAllCounterpartiesByClient(clientID, projectID, offset, limit, search, source, sortBy, order, minQuality, maxQuality)
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get counterparties for client_id %d: %v", clientID, err),
			Endpoint:  "/api/counterparties/all",
		})
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
		return
	}

	// Формируем ответ с информацией о проектах
	projectsInfo := make([]map[string]interface{}, len(result.Projects))
	for i, p := range result.Projects {
		projectsInfo[i] = map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		}
	}

	// Логирование успешного ответа
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("GetAllCounterparties success - client_id: %d, total: %d, returned: %d, processing_time: %dms",
			clientID, result.TotalCount, len(result.Counterparties), result.Stats.ProcessingTimeMs),
		Endpoint: "/api/counterparties/all",
	})

	s.writeJSONResponse(w, r, map[string]interface{}{
		"counterparties": result.Counterparties,
		"projects":       projectsInfo,
		"total":          result.TotalCount,
		"offset":         offset,
		"limit":          limit,
		"stats":          result.Stats,
	}, http.StatusOK)
}

// handleExportAllCounterparties экспортирует все контрагенты клиента в CSV или JSON формате
func (s *Server) handleExportAllCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Валидация обязательного параметра client_id
	clientID, err := ValidateIntParam(r, "client_id", 0, 1, 0)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
		s.writeJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
		return
	}
	if clientID <= 0 {
		s.writeJSONError(w, r, "client_id is required and must be positive", http.StatusBadRequest)
		return
	}

	// Валидация опционального параметра project_id
	var projectID *int
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr != "" {
		pID, err := ValidateIntParam(r, "project_id", 0, 1, 0)
		if err != nil {
			if s.HandleValidationError(w, r, err) {
				return
			}
			s.writeJSONError(w, r, fmt.Sprintf("Invalid project_id: %v", err), http.StatusBadRequest)
			return
		}
		if pID > 0 {
			projectID = &pID
		}
	}

	// Получаем параметры фильтрации (те же, что и в handleGetAllCounterparties)
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err := ValidateSearchQuery(search, 500); err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
		search = ""
	}

	source := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
	if source != "" && source != "database" && source != "normalized" {
		s.writeJSONError(w, r, "Invalid source parameter. Must be 'database', 'normalized', or empty", http.StatusBadRequest)
		return
	}

	// Параметры сортировки
	sortBy := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_by")))
	validSortFields := map[string]bool{
		"name": true, "quality": true, "source": true, "id": true, "": true,
	}
	if !validSortFields[sortBy] {
		s.writeJSONError(w, r, "Invalid sort_by parameter. Must be 'name', 'quality', 'source', 'id', or empty", http.StatusBadRequest)
		return
	}

	order := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if order != "" && order != "asc" && order != "desc" {
		s.writeJSONError(w, r, "Invalid order parameter. Must be 'asc', 'desc', or empty", http.StatusBadRequest)
		return
	}

	// Валидация фильтров по качеству
	var minQuality, maxQuality *float64
	if minQualityStr := r.URL.Query().Get("min_quality"); minQualityStr != "" {
		if q, err := strconv.ParseFloat(minQualityStr, 64); err == nil && q >= 0 && q <= 1 {
			minQuality = &q
		}
	}
	if maxQualityStr := r.URL.Query().Get("max_quality"); maxQualityStr != "" {
		if q, err := strconv.ParseFloat(maxQualityStr, 64); err == nil && q >= 0 && q <= 1 {
			maxQuality = &q
		}
	}

	// Проверяем логику фильтров
	if minQuality != nil && maxQuality != nil && *minQuality > *maxQuality {
		s.writeJSONError(w, r, "min_quality must be less than or equal to max_quality", http.StatusBadRequest)
		return
	}

	// Определяем формат экспорта (из query параметра или Accept header)
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "text/csv") || strings.Contains(acceptHeader, "application/csv") {
			format = "csv"
		} else if strings.Contains(acceptHeader, "application/json") {
			format = "json"
		} else {
			format = "json" // по умолчанию JSON
		}
	}

	if format != "csv" && format != "json" {
		s.writeJSONError(w, r, "Invalid format parameter. Must be 'csv' or 'json'", http.StatusBadRequest)
		return
	}

	// Получаем ВСЕ контрагенты (без пагинации для экспорта)
	result, err := s.serviceDB.GetAllCounterpartiesByClient(clientID, projectID, 0, 1000000, search, source, sortBy, order, minQuality, maxQuality)
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get counterparties for export (client_id %d): %v", clientID, err),
			Endpoint:  "/api/counterparties/all/export",
		})
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
		return
	}

	// Логирование экспорта
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("ExportAllCounterparties - client_id: %d, project_id: %v, format: %s, total: %d",
			clientID, projectID, format, result.TotalCount),
		Endpoint: "/api/counterparties/all/export",
	})

	// Генерируем имя файла
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("counterparties_client_%d_%s", clientID, timestamp)
	if projectID != nil {
		filename = fmt.Sprintf("counterparties_client_%d_project_%d_%s", clientID, *projectID, timestamp)
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		// Заголовки CSV
		headers := []string{
			"ID", "Name", "Source", "Project ID", "Project Name",
			"Database ID", "Database Name", "Tax ID (INN)", "KPP", "BIN",
			"Legal Address", "Postal Address", "Contact Phone", "Contact Email", "Contact Person",
			"Quality Score", "Reference", "Code", "Normalized Name", "Source Name", "Source Reference",
		}
		if err := csvWriter.Write(headers); err != nil {
			s.log(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to write CSV headers: %v", err),
				Endpoint:  "/api/counterparties/all/export",
			})
			return
		}

		// Данные
		for _, cp := range result.Counterparties {
			qualityScore := ""
			if cp.QualityScore != nil {
				qualityScore = fmt.Sprintf("%.2f", *cp.QualityScore)
			}
			databaseID := ""
			if cp.DatabaseID != nil {
				databaseID = fmt.Sprintf("%d", *cp.DatabaseID)
			}
			row := []string{
				fmt.Sprintf("%d", cp.ID),
				cp.Name,
				cp.Source,
				fmt.Sprintf("%d", cp.ProjectID),
				cp.ProjectName,
				databaseID,
				cp.DatabaseName,
				cp.TaxID,
				cp.KPP,
				cp.BIN,
				cp.LegalAddress,
				cp.PostalAddress,
				cp.ContactPhone,
				cp.ContactEmail,
				cp.ContactPerson,
				qualityScore,
				cp.Reference,
				cp.Code,
				cp.NormalizedName,
				cp.SourceName,
				cp.SourceReference,
			}
			if err := csvWriter.Write(row); err != nil {
				s.log(LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Failed to write CSV row: %v", err),
					Endpoint:  "/api/counterparties/all/export",
				})
				return
			}
		}

	default: // json
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))

		exportData := map[string]interface{}{
			"client_id":      clientID,
			"project_id":     projectID,
			"export_date":    time.Now().Format(time.RFC3339),
			"format_version": "1.0",
			"total":          result.TotalCount,
			"stats":          result.Stats,
			"counterparties": result.Counterparties,
			"projects":       result.Projects,
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(exportData); err != nil {
			s.log(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to encode JSON: %v", err),
				Endpoint:  "/api/counterparties/all/export",
			})
			return
		}
	}
}

// handleNormalizedCounterpartyRoutes обрабатывает вложенные маршруты для нормализованных контрагентов
func (s *Server) handleNormalizedCounterpartyRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/counterparties/normalized/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		s.writeJSONError(w, r, "Invalid request path", http.StatusBadRequest)
		return
	}

	// Обработка stats
	if len(parts) == 1 && parts[0] == "stats" {
		s.handleNormalizedCounterpartyStats(w, r)
		return
	}

	// Обработка enrich - ручное обогащение
	if len(parts) == 1 && parts[0] == "enrich" {
		if r.Method == http.MethodPost {
			s.handleEnrichCounterparty(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка duplicates - получение групп дубликатов
	if len(parts) == 1 && parts[0] == "duplicates" {
		if r.Method == http.MethodGet {
			s.handleGetCounterpartyDuplicates(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка merge дубликатов: /api/counterparties/normalized/duplicates/{groupId}/merge
	if len(parts) == 3 && parts[0] == "duplicates" && parts[2] == "merge" {
		if r.Method == http.MethodPost {
			groupId, err := ValidateIDPathParam(parts[1], "group_id")
			if err != nil {
				s.writeJSONError(w, r, fmt.Sprintf("Invalid duplicate group ID: %s", err.Error()), http.StatusBadRequest)
				return
			}
			s.handleMergeCounterpartyDuplicates(w, r, groupId)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка export - экспорт контрагентов
	if len(parts) == 1 && parts[0] == "export" {
		if r.Method == http.MethodPost {
			s.handleExportCounterparties(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка конкретного контрагента по ID: /api/counterparties/normalized/{id}
	if len(parts) == 1 {
		id, err := ValidateIDPathParam(parts[0], "counterparty_id")
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid counterparty ID: %s", err.Error()), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			s.handleGetNormalizedCounterparty(w, r, id)
		case http.MethodPut, http.MethodPatch:
			s.handleUpdateNormalizedCounterparty(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	s.writeJSONError(w, r, "Not found", http.StatusNotFound)
}

// handleNormalizedCounterpartyStats получает статистику по нормализованным контрагентам
func (s *Server) handleNormalizedCounterpartyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		s.writeJSONError(w, r, "project_id is required", http.StatusBadRequest)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid project_id: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Проверяем существование проекта
	_, err = s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	// Получаем статистику
	stats, err := s.serviceDB.GetNormalizedCounterpartyStats(projectID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleGetNormalizedCounterparty получает контрагента по ID
func (s *Server) handleGetNormalizedCounterparty(w http.ResponseWriter, r *http.Request, id int) {
	counterparty, err := s.serviceDB.GetNormalizedCounterparty(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeJSONError(w, r, "Counterparty not found", http.StatusNotFound)
		} else {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get counterparty: %v", err), http.StatusInternalServerError)
		}
		return
	}

	s.writeJSONResponse(w, r, counterparty, http.StatusOK)
}

// handleUpdateNormalizedCounterparty обновляет контрагента
func (s *Server) handleUpdateNormalizedCounterparty(w http.ResponseWriter, r *http.Request, id int) {
	var req struct {
		NormalizedName       string  `json:"normalized_name"`
		TaxID                string  `json:"tax_id"`
		KPP                  string  `json:"kpp"`
		BIN                  string  `json:"bin"`
		LegalAddress         string  `json:"legal_address"`
		PostalAddress        string  `json:"postal_address"`
		ContactPhone         string  `json:"contact_phone"`
		ContactEmail         string  `json:"contact_email"`
		ContactPerson        string  `json:"contact_person"`
		LegalForm            string  `json:"legal_form"`
		BankName             string  `json:"bank_name"`
		BankAccount          string  `json:"bank_account"`
		CorrespondentAccount string  `json:"correspondent_account"`
		BIK                  string  `json:"bik"`
		QualityScore         float64 `json:"quality_score"`
		SourceEnrichment     string  `json:"source_enrichment"`
		Subcategory          string  `json:"subcategory"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем существование контрагента
	_, err := s.serviceDB.GetNormalizedCounterparty(id)
	if err != nil {
		s.writeJSONError(w, r, "Counterparty not found", http.StatusNotFound)
		return
	}

	// Обновляем контрагента
	err = s.serviceDB.UpdateNormalizedCounterparty(
		id,
		req.NormalizedName,
		req.TaxID, req.KPP, req.BIN,
		req.LegalAddress, req.PostalAddress,
		req.ContactPhone, req.ContactEmail,
		req.ContactPerson, req.LegalForm,
		req.BankName, req.BankAccount,
		req.CorrespondentAccount, req.BIK,
		req.QualityScore,
		req.SourceEnrichment,
		req.Subcategory,
	)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to update counterparty: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем обновленного контрагента
	updated, err := s.serviceDB.GetNormalizedCounterparty(id)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get updated counterparty: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":      true,
		"message":      "Counterparty updated successfully",
		"counterparty": updated,
	}, http.StatusOK)
}

// handleEnrichCounterparty выполняет ручное обогащение контрагента
func (s *Server) handleEnrichCounterparty(w http.ResponseWriter, r *http.Request) {
	if s.enrichmentFactory == nil {
		s.writeJSONError(w, r, "Enrichment is not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		CounterpartyID int    `json:"counterparty_id"`
		INN            string `json:"inn"`
		BIN            string `json:"bin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Если указан ID контрагента, получаем его данные
	if req.CounterpartyID > 0 {
		cp, err := s.serviceDB.GetNormalizedCounterparty(req.CounterpartyID)
		if err != nil {
			s.writeJSONError(w, r, "Counterparty not found", http.StatusNotFound)
			return
		}
		if req.INN == "" {
			req.INN = cp.TaxID
		}
		if req.BIN == "" {
			req.BIN = cp.BIN
		}
	}

	if req.INN == "" && req.BIN == "" {
		s.writeJSONError(w, r, "INN or BIN is required", http.StatusBadRequest)
		return
	}

	// Выполняем обогащение
	response := s.enrichmentFactory.Enrich(req.INN, req.BIN)
	if !response.Success {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"success": false,
			"errors":  response.Errors,
		}, http.StatusOK)
		return
	}

	// Берем лучший результат
	bestResult := s.enrichmentFactory.GetBestResult(response.Results)
	if bestResult == nil {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"success": false,
			"message": "No enrichment results available",
		}, http.StatusOK)
		return
	}

	// Если указан ID контрагента, обновляем его
	if req.CounterpartyID > 0 {
		cp, _ := s.serviceDB.GetNormalizedCounterparty(req.CounterpartyID)
		if cp != nil {
			// Объединяем данные из обогащения
			normalizedName := cp.NormalizedName
			if bestResult.FullName != "" {
				normalizedName = bestResult.FullName
			}

			inn := cp.TaxID
			if bestResult.INN != "" {
				inn = bestResult.INN
			}
			bin := cp.BIN
			if bestResult.BIN != "" {
				bin = bestResult.BIN
			}

			legalAddress := cp.LegalAddress
			if bestResult.LegalAddress != "" {
				legalAddress = bestResult.LegalAddress
			}

			contactPhone := cp.ContactPhone
			if bestResult.Phone != "" {
				contactPhone = bestResult.Phone
			}

			contactEmail := cp.ContactEmail
			if bestResult.Email != "" {
				contactEmail = bestResult.Email
			}

			// Обновляем контрагента
			err := s.serviceDB.UpdateNormalizedCounterparty(
				req.CounterpartyID,
				normalizedName,
				inn, cp.KPP, bin,
				legalAddress, cp.PostalAddress,
				contactPhone, contactEmail,
				cp.ContactPerson, cp.LegalForm,
				cp.BankName, cp.BankAccount,
				cp.CorrespondentAccount, cp.BIK,
				cp.QualityScore,
				bestResult.Source,
				cp.Subcategory,
			)
			if err != nil {
				log.Printf("Failed to update counterparty after enrichment: %v", err)
			}
		}
	}

	// Преобразуем результат в JSON для ответа
	resultJSON, _ := bestResult.ToJSON()

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"result":  bestResult,
		"raw":     resultJSON,
	}, http.StatusOK)
}

// handleGetCounterpartyDuplicates получает группы дубликатов контрагентов
func (s *Server) handleGetCounterpartyDuplicates(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		s.writeJSONError(w, r, "project_id is required", http.StatusBadRequest)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid project_id: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем всех контрагентов проекта
	counterparties, _, err := s.serviceDB.GetNormalizedCounterparties(projectID, 0, 10000, "", "", "")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
		return
	}

	// Группируем по ИНН/БИН
	groups := make(map[string][]*database.NormalizedCounterparty)
	for _, cp := range counterparties {
		key := cp.TaxID
		if key == "" {
			key = cp.BIN
		}
		if key != "" {
			groups[key] = append(groups[key], cp)
		}
	}

	// Фильтруем только группы с дубликатами
	duplicateGroups := []map[string]interface{}{}
	for key, items := range groups {
		if len(items) > 1 {
			duplicateGroups = append(duplicateGroups, map[string]interface{}{
				"tax_id": key,
				"count":  len(items),
				"items":  items,
			})
		}
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"total_groups": len(duplicateGroups),
		"groups":       duplicateGroups,
	}, http.StatusOK)
}

// handleMergeCounterpartyDuplicates выполняет слияние дубликатов
func (s *Server) handleMergeCounterpartyDuplicates(w http.ResponseWriter, r *http.Request, groupID int) {
	var req struct {
		MasterID int   `json:"master_id"`
		MergeIDs []int `json:"merge_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MasterID == 0 {
		s.writeJSONError(w, r, "master_id is required", http.StatusBadRequest)
		return
	}

	// Получаем мастер-контрагента
	master, err := s.serviceDB.GetNormalizedCounterparty(req.MasterID)
	if err != nil {
		s.writeJSONError(w, r, "Master counterparty not found", http.StatusNotFound)
		return
	}

	// Объединяем данные из дубликатов в мастер
	for _, mergeID := range req.MergeIDs {
		if mergeID == req.MasterID {
			continue
		}

		duplicate, err := s.serviceDB.GetNormalizedCounterparty(mergeID)
		if err != nil {
			continue
		}

		// Объединяем данные (приоритет у мастер-записи)
		if master.LegalAddress == "" && duplicate.LegalAddress != "" {
			master.LegalAddress = duplicate.LegalAddress
		}
		if master.PostalAddress == "" && duplicate.PostalAddress != "" {
			master.PostalAddress = duplicate.PostalAddress
		}
		if master.ContactPhone == "" && duplicate.ContactPhone != "" {
			master.ContactPhone = duplicate.ContactPhone
		}
		if master.ContactEmail == "" && duplicate.ContactEmail != "" {
			master.ContactEmail = duplicate.ContactEmail
		}
		if master.ContactPerson == "" && duplicate.ContactPerson != "" {
			master.ContactPerson = duplicate.ContactPerson
		}

		// Обновляем мастер-запись
		err = s.serviceDB.UpdateNormalizedCounterparty(
			req.MasterID,
			master.NormalizedName,
			master.TaxID, master.KPP, master.BIN,
			master.LegalAddress, master.PostalAddress,
			master.ContactPhone, master.ContactEmail,
			master.ContactPerson, master.LegalForm,
			master.BankName, master.BankAccount,
			master.CorrespondentAccount, master.BIK,
			master.QualityScore,
			master.SourceEnrichment,
			master.Subcategory,
		)
		if err != nil {
			log.Printf("Failed to update master counterparty: %v", err)
		}

		// Помечаем дубликат как объединенный - удаляем запись
		// Все данные уже перенесены в мастер-запись
		err = s.serviceDB.DeleteNormalizedCounterparty(mergeID)
		if err != nil {
			log.Printf("Warning: Failed to delete merged counterparty %d: %v", mergeID, err)
			// Не критично - продолжаем работу
		} else {
			log.Printf("Merged and deleted counterparty %d into master %d", mergeID, req.MasterID)
		}
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Merged %d counterparties into master %d", len(req.MergeIDs), req.MasterID),
	}, http.StatusOK)
}

// handleExportCounterparties экспортирует контрагентов
func (s *Server) handleExportCounterparties(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID int    `json:"project_id"`
		Format    string `json:"format"` // csv, json, xml
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProjectID == 0 {
		s.writeJSONError(w, r, "project_id is required", http.StatusBadRequest)
		return
	}

	if req.Format == "" {
		req.Format = "json"
	}

	// Получаем всех контрагентов проекта
	counterparties, _, err := s.serviceDB.GetNormalizedCounterparties(req.ProjectID, 0, 100000, "", "", "")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
		return
	}

	switch req.Format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=counterparties_%d.csv", req.ProjectID))
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		// Заголовки
		csvWriter.Write([]string{
			"ID", "Name", "Normalized Name", "INN", "KPP", "BIN",
			"Legal Address", "Postal Address", "Phone", "Email",
			"Contact Person", "Quality Score", "Source Enrichment",
		})

		// Данные
		for _, cp := range counterparties {
			csvWriter.Write([]string{
				fmt.Sprintf("%d", cp.ID),
				cp.SourceName,
				cp.NormalizedName,
				cp.TaxID,
				cp.KPP,
				cp.BIN,
				cp.LegalAddress,
				cp.PostalAddress,
				cp.ContactPhone,
				cp.ContactEmail,
				cp.ContactPerson,
				fmt.Sprintf("%.2f", cp.QualityScore),
				cp.SourceEnrichment,
			})
		}

	case "xml":
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=counterparties_%d.xml", req.ProjectID))
		w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<counterparties>\n"))
		for _, cp := range counterparties {
			w.Write([]byte(fmt.Sprintf(
				"  <counterparty id=\"%d\">\n    <name>%s</name>\n    <normalized_name>%s</normalized_name>\n    <inn>%s</inn>\n    <kpp>%s</kpp>\n    <bin>%s</bin>\n  </counterparty>\n",
				cp.ID, cp.SourceName, cp.NormalizedName, cp.TaxID, cp.KPP, cp.BIN,
			)))
		}
		w.Write([]byte("</counterparties>\n"))

	default: // json
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=counterparties_%d.json", req.ProjectID))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"project_id":     req.ProjectID,
			"total":          len(counterparties),
			"counterparties": counterparties,
		})
	}
}

// handleBulkUpdateCounterparties выполняет массовое обновление контрагентов
func (s *Server) handleBulkUpdateCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDs     []int `json:"ids"`
		Updates struct {
			NormalizedName       *string  `json:"normalized_name"`
			TaxID                *string  `json:"tax_id"`
			KPP                  *string  `json:"kpp"`
			BIN                  *string  `json:"bin"`
			LegalAddress         *string  `json:"legal_address"`
			PostalAddress        *string  `json:"postal_address"`
			ContactPhone         *string  `json:"contact_phone"`
			ContactEmail         *string  `json:"contact_email"`
			ContactPerson        *string  `json:"contact_person"`
			LegalForm            *string  `json:"legal_form"`
			BankName             *string  `json:"bank_name"`
			BankAccount          *string  `json:"bank_account"`
			CorrespondentAccount *string  `json:"correspondent_account"`
			BIK                  *string  `json:"bik"`
			QualityScore         *float64 `json:"quality_score"`
			SourceEnrichment     *string  `json:"source_enrichment"`
			Subcategory          *string  `json:"subcategory"`
		} `json:"updates"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		s.writeJSONError(w, r, "ids array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	successCount := 0
	failedCount := 0
	errors := []string{}

	for _, id := range req.IDs {
		// Получаем текущего контрагента
		cp, err := s.serviceDB.GetNormalizedCounterparty(id)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: %v", id, err))
			continue
		}

		// Применяем обновления
		normalizedName := cp.NormalizedName
		if req.Updates.NormalizedName != nil {
			normalizedName = *req.Updates.NormalizedName
		}
		taxID := cp.TaxID
		if req.Updates.TaxID != nil {
			taxID = *req.Updates.TaxID
		}
		kpp := cp.KPP
		if req.Updates.KPP != nil {
			kpp = *req.Updates.KPP
		}
		bin := cp.BIN
		if req.Updates.BIN != nil {
			bin = *req.Updates.BIN
		}
		legalAddress := cp.LegalAddress
		if req.Updates.LegalAddress != nil {
			legalAddress = *req.Updates.LegalAddress
		}
		postalAddress := cp.PostalAddress
		if req.Updates.PostalAddress != nil {
			postalAddress = *req.Updates.PostalAddress
		}
		contactPhone := cp.ContactPhone
		if req.Updates.ContactPhone != nil {
			contactPhone = *req.Updates.ContactPhone
		}
		contactEmail := cp.ContactEmail
		if req.Updates.ContactEmail != nil {
			contactEmail = *req.Updates.ContactEmail
		}
		contactPerson := cp.ContactPerson
		if req.Updates.ContactPerson != nil {
			contactPerson = *req.Updates.ContactPerson
		}
		legalForm := cp.LegalForm
		if req.Updates.LegalForm != nil {
			legalForm = *req.Updates.LegalForm
		}
		bankName := cp.BankName
		if req.Updates.BankName != nil {
			bankName = *req.Updates.BankName
		}
		bankAccount := cp.BankAccount
		if req.Updates.BankAccount != nil {
			bankAccount = *req.Updates.BankAccount
		}
		correspondentAccount := cp.CorrespondentAccount
		if req.Updates.CorrespondentAccount != nil {
			correspondentAccount = *req.Updates.CorrespondentAccount
		}
		bik := cp.BIK
		if req.Updates.BIK != nil {
			bik = *req.Updates.BIK
		}
		qualityScore := cp.QualityScore
		if req.Updates.QualityScore != nil {
			qualityScore = *req.Updates.QualityScore
		}
		sourceEnrichment := cp.SourceEnrichment
		if req.Updates.SourceEnrichment != nil {
			sourceEnrichment = *req.Updates.SourceEnrichment
		}
		subcategory := cp.Subcategory
		if req.Updates.Subcategory != nil {
			subcategory = *req.Updates.Subcategory
		}

		// Обновляем контрагента
		err = s.serviceDB.UpdateNormalizedCounterparty(
			id,
			normalizedName,
			taxID, kpp, bin,
			legalAddress, postalAddress,
			contactPhone, contactEmail,
			contactPerson, legalForm,
			bankName, bankAccount,
			correspondentAccount, bik,
			qualityScore,
			sourceEnrichment,
			subcategory,
		)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: %v", id, err))
			continue
		}

		successCount++
	}

	response := map[string]interface{}{
		"success":       failedCount == 0,
		"total":         len(req.IDs),
		"success_count": successCount,
		"failed_count":  failedCount,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := http.StatusOK
	if failedCount > 0 && successCount == 0 {
		statusCode = http.StatusInternalServerError
	}

	s.writeJSONResponse(w, r, response, statusCode)
}

// handleBulkDeleteCounterparties выполняет массовое удаление контрагентов
func (s *Server) handleBulkDeleteCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDs []int `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		s.writeJSONError(w, r, "ids array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	successCount := 0
	failedCount := 0
	errors := []string{}

	for _, id := range req.IDs {
		err := s.serviceDB.DeleteNormalizedCounterparty(id)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: %v", id, err))
			continue
		}
		successCount++
	}

	response := map[string]interface{}{
		"success":       failedCount == 0,
		"total":         len(req.IDs),
		"success_count": successCount,
		"failed_count":  failedCount,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := http.StatusOK
	if failedCount > 0 && successCount == 0 {
		statusCode = http.StatusInternalServerError
	}

	s.writeJSONResponse(w, r, response, statusCode)
}

// handleBulkEnrichCounterparties выполняет массовое обогащение контрагентов
func (s *Server) handleBulkEnrichCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.enrichmentFactory == nil {
		s.writeJSONError(w, r, "Enrichment is not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		IDs []int `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		s.writeJSONError(w, r, "ids array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	successCount := 0
	failedCount := 0
	errors := []string{}

	for _, id := range req.IDs {
		// Получаем контрагента
		cp, err := s.serviceDB.GetNormalizedCounterparty(id)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: not found", id))
			continue
		}

		// Определяем ИНН/БИН для обогащения
		inn := cp.TaxID
		bin := cp.BIN
		if inn == "" && bin == "" {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: INN or BIN is required", id))
			continue
		}

		// Выполняем обогащение
		response := s.enrichmentFactory.Enrich(inn, bin)
		if !response.Success {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: enrichment failed: %v", id, response.Errors))
			continue
		}

		// Берем лучший результат
		bestResult := s.enrichmentFactory.GetBestResult(response.Results)
		if bestResult == nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: no enrichment results available", id))
			continue
		}

		// Объединяем данные из обогащения
		normalizedName := cp.NormalizedName
		if bestResult.FullName != "" {
			normalizedName = bestResult.FullName
		}

		if bestResult.INN != "" {
			inn = bestResult.INN
		}
		if bestResult.BIN != "" {
			bin = bestResult.BIN
		}

		legalAddress := cp.LegalAddress
		if bestResult.LegalAddress != "" {
			legalAddress = bestResult.LegalAddress
		}

		contactPhone := cp.ContactPhone
		if bestResult.Phone != "" {
			contactPhone = bestResult.Phone
		}

		contactEmail := cp.ContactEmail
		if bestResult.Email != "" {
			contactEmail = bestResult.Email
		}

		// Обновляем контрагента
		err = s.serviceDB.UpdateNormalizedCounterparty(
			id,
			normalizedName,
			inn, cp.KPP, bin,
			legalAddress, cp.PostalAddress,
			contactPhone, contactEmail,
			cp.ContactPerson, cp.LegalForm,
			cp.BankName, cp.BankAccount,
			cp.CorrespondentAccount, cp.BIK,
			cp.QualityScore,
			bestResult.Source,
			cp.Subcategory,
		)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: failed to update: %v", id, err))
			continue
		}

		successCount++
	}

	response := map[string]interface{}{
		"success":       failedCount == 0,
		"total":         len(req.IDs),
		"success_count": successCount,
		"failed_count":  failedCount,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := http.StatusOK
	if failedCount > 0 && successCount == 0 {
		statusCode = http.StatusInternalServerError
	}

	s.writeJSONResponse(w, r, response, statusCode)
}

// normalizePathForComparison нормализует путь для сравнения, возвращая все варианты
func normalizePathForComparison(path string) []string {
	normalized := filepath.Clean(path)
	normalizedSlash := filepath.ToSlash(normalized)
	normalizedBackslash := filepath.FromSlash(normalized)

	// Возвращаем все варианты для сравнения
	return []string{path, normalized, normalizedSlash, normalizedBackslash}
}

// pathsMatch проверяет, совпадают ли два пути (с учетом разных форматов)
func pathsMatch(path1, path2 string) bool {
	variants1 := normalizePathForComparison(path1)
	variants2 := normalizePathForComparison(path2)

	for _, v1 := range variants1 {
		for _, v2 := range variants2 {
			if v1 == v2 {
				return true
			}
		}
	}
	return false
}

// handleFindProjectByDatabase перемещен в server/database_legacy_handlers.go

// handleGetProjectPipelineStatsWithParams получает статистику этапов обработки для проекта с параметрами clientID и projectID
// Версия с параметрами для использования из client routes
func (s *Server) handleGetProjectPipelineStatsWithParams(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Проверяем тип проекта - статистика этапов для номенклатуры и нормализации
	// Также поддерживаем nomenclature_counterparties для совместимости
	if project.ProjectType != "nomenclature" &&
		project.ProjectType != "normalization" &&
		project.ProjectType != "nomenclature_counterparties" {
		s.writeJSONError(w, r, "Pipeline stats are only available for nomenclature and normalization projects", http.StatusBadRequest)
		return
	}

	// Получаем активные базы данных проекта
	databases, err := s.serviceDB.GetProjectDatabases(projectID, true)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get project databases: %v", err), http.StatusInternalServerError)
		return
	}

	if len(databases) == 0 {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"total_records":       0,
			"overall_progress":    0,
			"stage_stats":         []interface{}{},
			"quality_metrics":     map[string]interface{}{},
			"processing_duration": "N/A",
			"last_updated":        "",
			"message":             "No active databases found for this project",
		}, http.StatusOK)
		return
	}

	// Агрегируем статистику по всем активным БД проекта
	var allStats []map[string]interface{}
	for _, dbInfo := range databases {
		stats, err := database.GetProjectPipelineStats(dbInfo.FilePath)
		if err != nil {
			log.Printf("Failed to get pipeline stats from database %s: %v", dbInfo.FilePath, err)
			continue
		}
		allStats = append(allStats, stats)
	}

	// Агрегируем статистику из всех БД
	if len(allStats) == 0 {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"total_records":       0,
			"overall_progress":    0,
			"stage_stats":         []interface{}{},
			"quality_metrics":     map[string]interface{}{},
			"processing_duration": "N/A",
			"last_updated":        "",
			"message":             "No statistics available",
		}, http.StatusOK)
		return
	}

	// Объединяем статистику из всех БД
	aggregatedStats := database.AggregatePipelineStats(allStats)
	s.writeJSONResponse(w, r, aggregatedStats, http.StatusOK)
}

// extractKeywords извлекает ключевые слова из нормализованного имени
func extractKeywords(normalizedName string) []string {
	// Удаляем служебные слова и символы
	stopWords := map[string]bool{
		"и": true, "в": true, "на": true, "с": true, "по": true, "для": true,
		"из": true, "от": true, "к": true, "о": true, "об": true, "со": true,
		"the": true, "a": true, "an": true, "and": true, "or": true, "of": true,
		"to": true, "in": true, "for": true, "with": true, "on": true,
	}

	// Разбиваем на слова
	words := regexp.MustCompile(`\s+`).Split(strings.ToLower(normalizedName), -1)
	var keywords []string

	for _, word := range words {
		// Удаляем знаки препинания
		word = regexp.MustCompile(`[^\p{L}\p{N}]+`).ReplaceAllString(word, "")
		// Пропускаем короткие слова и стоп-слова
		if len(word) >= 3 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// calculateOkpd2Confidence вычисляет уверенность классификации ОКПД2
func calculateOkpd2Confidence(searchTerm, okpd2Name string, level int) float64 {
	searchTerm = strings.ToLower(searchTerm)
	okpd2NameLower := strings.ToLower(okpd2Name)

	// Базовая уверенность зависит от уровня (более глубокие уровни более специфичны)
	baseConfidence := 0.3 + float64(level)*0.1
	if baseConfidence > 0.9 {
		baseConfidence = 0.9
	}

	// Точное совпадение
	if okpd2NameLower == searchTerm {
		return 0.95
	}

	// Начинается с поискового термина
	if strings.HasPrefix(okpd2NameLower, searchTerm) {
		return baseConfidence + 0.3
	}

	// Содержит поисковый термин
	if strings.Contains(okpd2NameLower, searchTerm) {
		// Проверяем, сколько раз встречается
		count := strings.Count(okpd2NameLower, searchTerm)
		confidence := baseConfidence + float64(count)*0.1
		if confidence > 0.85 {
			confidence = 0.85
		}
		return confidence
	}

	// Частичное совпадение (по словам)
	okpd2Words := regexp.MustCompile(`\s+`).Split(okpd2NameLower, -1)
	matchedWords := 0
	for _, word := range okpd2Words {
		if strings.Contains(word, searchTerm) || strings.Contains(searchTerm, word) {
			matchedWords++
		}
	}

	if matchedWords > 0 {
		wordMatchConfidence := float64(matchedWords) / float64(len(okpd2Words)) * 0.4
		return baseConfidence + wordMatchConfidence
	}

	return baseConfidence
}

// classifyKpvedForDatabase выполняет КПВЭД классификацию для базы данных
func (s *Server) classifyKpvedForDatabase(db *database.DB, dbName string) {
	log.Println("Начинаем КПВЭД классификацию...")
	s.normalizerEvents <- "Начало КПВЭД классификации"

	s.kpvedClassifierMutex.RLock()
	classifier := s.hierarchicalClassifier
	s.kpvedClassifierMutex.RUnlock()

	if classifier == nil {
		log.Println("КПВЭД классификатор недоступен")
		s.normalizerEvents <- "КПВЭД классификатор недоступен"
		return
	}

	// Получаем записи без КПВЭД классификации
	rows, err := db.Query(`
		SELECT id, normalized_name, category
		FROM normalized_data
		WHERE (kpved_code IS NULL OR kpved_code = '' OR TRIM(kpved_code) = '')
	`)
	if err != nil {
		log.Printf("Ошибка получения записей для КПВЭД классификации: %v", err)
		s.normalizerEvents <- fmt.Sprintf("Ошибка КПВЭД: %v", err)
		return
	}
	defer rows.Close()

	var recordsToClassify []struct {
		ID             int
		NormalizedName string
		Category       string
	}

	for rows.Next() {
		var record struct {
			ID             int
			NormalizedName string
			Category       string
		}
		if err := rows.Scan(&record.ID, &record.NormalizedName, &record.Category); err != nil {
			log.Printf("Ошибка сканирования записи: %v", err)
			continue
		}
		recordsToClassify = append(recordsToClassify, record)
	}

	totalToClassify := len(recordsToClassify)
	if totalToClassify == 0 {
		log.Println("Нет записей для КПВЭД классификации")
		s.normalizerEvents <- "Все записи уже классифицированы по КПВЭД"
		return
	}

	log.Printf("Найдено записей для КПВЭД классификации: %d", totalToClassify)
	s.normalizerEvents <- fmt.Sprintf("Классификация %d записей по КПВЭД", totalToClassify)

	classified := 0
	failed := 0
	for i, record := range recordsToClassify {
		result, err := classifier.Classify(record.NormalizedName, record.Category)
		if err != nil {
			log.Printf("Ошибка классификации записи %d: %v", record.ID, err)
			failed++
			continue
		}

		_, err = db.Exec(`
			UPDATE normalized_data
			SET kpved_code = ?, kpved_name = ?, kpved_confidence = ?,
			    stage11_kpved_code = ?, stage11_kpved_name = ?, stage11_kpved_confidence = ?,
			    stage11_kpved_completed = 1, stage11_kpved_completed_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, result.FinalCode, result.FinalName, result.FinalConfidence,
			result.FinalCode, result.FinalName, result.FinalConfidence, record.ID)

		if err != nil {
			log.Printf("Ошибка обновления КПВЭД для записи %d: %v", record.ID, err)
			failed++
			continue
		}

		classified++

		if (i+1)%10 == 0 || i+1 == totalToClassify {
			progress := float64(i+1) / float64(totalToClassify) * 100
			log.Printf("КПВЭД классификация: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
			s.normalizerEvents <- fmt.Sprintf("КПВЭД: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
		}
	}

	log.Printf("КПВЭД классификация завершена: классифицировано %d из %d записей (ошибок: %d)", classified, totalToClassify, failed)
	s.normalizerEvents <- fmt.Sprintf("КПВЭД классификация завершена: %d/%d (ошибок: %d)", classified, totalToClassify, failed)
}

// classifyOkpd2ForDatabase выполняет ОКПД2 классификацию для базы данных
func (s *Server) classifyOkpd2ForDatabase(db *database.DB, dbName string) {
	log.Println("Начинаем ОКПД2 классификацию...")
	s.normalizerEvents <- "Начало ОКПД2 классификации"

	serviceDB := s.serviceDB.GetDB()

	// Получаем записи без ОКПД2 классификации
	rows, err := db.Query(`
		SELECT id, normalized_name, category
		FROM normalized_data
		WHERE (stage12_okpd2_code IS NULL OR stage12_okpd2_code = '' OR TRIM(stage12_okpd2_code) = '')
	`)
	if err != nil {
		log.Printf("Ошибка получения записей для ОКПД2 классификации: %v", err)
		s.normalizerEvents <- fmt.Sprintf("Ошибка ОКПД2: %v", err)
		return
	}
	defer rows.Close()

	var recordsToClassify []struct {
		ID             int
		NormalizedName string
		Category       string
	}

	for rows.Next() {
		var record struct {
			ID             int
			NormalizedName string
			Category       string
		}
		if err := rows.Scan(&record.ID, &record.NormalizedName, &record.Category); err != nil {
			log.Printf("Ошибка сканирования записи: %v", err)
			continue
		}
		recordsToClassify = append(recordsToClassify, record)
	}

	totalToClassify := len(recordsToClassify)
	if totalToClassify == 0 {
		log.Println("Нет записей для ОКПД2 классификации")
		s.normalizerEvents <- "Все записи уже классифицированы по ОКПД2"
		return
	}

	log.Printf("Найдено записей для ОКПД2 классификации: %d", totalToClassify)
	s.normalizerEvents <- fmt.Sprintf("Классификация %d записей по ОКПД2", totalToClassify)

	classified := 0
	failed := 0

	for i, record := range recordsToClassify {
		// Простой поиск по ключевым словам в ОКПД2
		searchTerms := extractKeywords(record.NormalizedName)
		if len(searchTerms) == 0 {
			searchTerms = []string{record.NormalizedName}
		}

		var bestMatch struct {
			Code       string
			Name       string
			Confidence float64
		}
		bestMatch.Confidence = 0.0

		// Ищем совпадения по каждому ключевому слову
		for _, term := range searchTerms {
			if len(term) < 3 {
				continue
			}

			searchPattern := "%" + term + "%"
			query := `
				SELECT code, name, level
				FROM okpd2_classifier
				WHERE name LIKE ?
				ORDER BY 
					CASE 
						WHEN name LIKE ? THEN 1
						WHEN name LIKE ? THEN 2
						ELSE 3
					END,
					level DESC
				LIMIT 5
			`
			exactPattern := term
			startPattern := term + "%"

			okpd2Rows, err := serviceDB.Query(query, searchPattern, exactPattern, startPattern)
			if err != nil {
				log.Printf("Ошибка поиска ОКПД2 для '%s': %v", term, err)
				continue
			}

			for okpd2Rows.Next() {
				var code, name string
				var level int
				if err := okpd2Rows.Scan(&code, &name, &level); err != nil {
					continue
				}

				confidence := calculateOkpd2Confidence(term, name, level)
				if confidence > bestMatch.Confidence {
					bestMatch.Code = code
					bestMatch.Name = name
					bestMatch.Confidence = confidence
				}
			}
			okpd2Rows.Close()
		}

		// Если нашли совпадение с достаточной уверенностью, сохраняем
		if bestMatch.Confidence >= 0.3 && bestMatch.Code != "" {
			_, err = db.Exec(`
				UPDATE normalized_data
				SET stage12_okpd2_code = ?, stage12_okpd2_name = ?, stage12_okpd2_confidence = ?,
				    stage12_okpd2_completed = 1, stage12_okpd2_completed_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, bestMatch.Code, bestMatch.Name, bestMatch.Confidence, record.ID)

			if err != nil {
				log.Printf("Ошибка обновления ОКПД2 для записи %d: %v", record.ID, err)
				failed++
				continue
			}

			classified++
		} else {
			// Отмечаем как обработанное, но без результата
			_, err = db.Exec(`
				UPDATE normalized_data
				SET stage12_okpd2_completed = 1, stage12_okpd2_completed_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, record.ID)
			if err != nil {
				log.Printf("Ошибка обновления статуса ОКПД2 для записи %d: %v", record.ID, err)
			}
			failed++
		}

		// Логируем прогресс каждые 10 записей или на последней записи
		if (i+1)%10 == 0 || i+1 == totalToClassify {
			progress := float64(i+1) / float64(totalToClassify) * 100
			log.Printf("ОКПД2 классификация: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
			s.normalizerEvents <- fmt.Sprintf("ОКПД2: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
		}
	}

	log.Printf("ОКПД2 классификация завершена: классифицировано %d из %d записей (не найдено: %d)", classified, totalToClassify, failed)
	s.normalizerEvents <- fmt.Sprintf("ОКПД2 классификация завершена: %d/%d (не найдено: %d)", classified, totalToClassify, failed)
}

// convertClientGroupsToNormalizedItems преобразует группы из ClientNormalizer в NormalizedItem для сохранения
func (s *Server) convertClientGroupsToNormalizedItems(
	groups map[string]*normalization.ClientNormalizationGroup,
	projectID int,
	sessionID int,
) ([]*database.NormalizedItem, map[string][]*database.ItemAttribute) {
	normalizedItems := make([]*database.NormalizedItem, 0)
	itemAttributes := make(map[string][]*database.ItemAttribute)

	for _, group := range groups {
		normalizedReference := group.NormalizedName
		mergedCount := len(group.Items)

		// Для каждой записи в группе создаем нормализованную запись
		for _, item := range group.Items {
			normalizedItem := &database.NormalizedItem{
				SourceReference:     item.Reference,
				SourceName:          item.Name,
				Code:                item.Code,
				NormalizedName:      group.NormalizedName,
				NormalizedReference: normalizedReference,
				Category:            group.Category,
				MergedCount:         mergedCount,
				AIConfidence:        group.AIConfidence,
				AIReasoning:         group.AIReasoning,
				ProcessingLevel:     group.ProcessingLevel,
				KpvedCode:           group.KpvedCode,
				KpvedName:           group.KpvedName,
				KpvedConfidence:     group.KpvedConfidence,
			}

			normalizedItems = append(normalizedItems, normalizedItem)

			// Сохраняем атрибуты для этого элемента
			if item.Code != "" {
				if attrs, ok := group.Attributes[item.Code]; ok && len(attrs) > 0 {
					itemAttributes[item.Code] = attrs
				}
			}
		}
	}

	return normalizedItems, itemAttributes
}
