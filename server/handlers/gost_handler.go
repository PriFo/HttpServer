package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "httpserver/server/errors"
	"httpserver/server/services"
)

// GostHandler обработчик для работы с ГОСТами
type GostHandler struct {
	gostService *services.GostService
}

// NewGostHandler создает новый обработчик для ГОСТов
func NewGostHandler(gostService *services.GostService) *GostHandler {
	return &GostHandler{
		gostService: gostService,
	}
}

// HandleGetGosts обработчик получения списка ГОСТов
// @Summary Получить список ГОСТов
// @Description Возвращает список ГОСТов с фильтрацией и пагинацией
// @Tags gosts
// @Accept json
// @Produce json
// @Param limit query int false "Количество записей на странице" default(50)
// @Param offset query int false "Смещение для пагинации" default(0)
// @Param status query string false "Фильтр по статусу"
// @Param source_type query string false "Фильтр по типу источника"
// @Param search query string false "Поисковый запрос"
// @Param adoption_from query string false "Дата принятия с (ГГГГ-ММ-ДД)"
// @Param adoption_to query string false "Дата принятия по (ГГГГ-ММ-ДД)"
// @Param effective_from query string false "Дата вступления с (ГГГГ-ММ-ДД)"
// @Param effective_to query string false "Дата вступления по (ГГГГ-ММ-ДД)"
// @Success 200 {object} map[string]interface{} "Список ГОСТов"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts [get]
func (h *GostHandler) HandleGetGosts(c *gin.Context) {
	// Парсим параметры запроса
	limit := 50
	offset := 0
	status := c.Query("status")
	sourceType := c.Query("source_type")
	search := c.Query("search")
	adoptionFrom := c.Query("adoption_from")
	adoptionTo := c.Query("adoption_to")
	effectiveFrom := c.Query("effective_from")
	effectiveTo := c.Query("effective_to")

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	dateParams := []struct {
		value string
		name  string
	}{
		{adoptionFrom, "adoption_from"},
		{adoptionTo, "adoption_to"},
		{effectiveFrom, "effective_from"},
		{effectiveTo, "effective_to"},
	}

	for _, param := range dateParams {
		if param.value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", param.value); err != nil {
			SendJSONError(c, http.StatusBadRequest, fmt.Sprintf("Неверный формат даты для %s. Используйте формат ГГГГ-ММ-ДД", param.name))
			return
		}
	}

	result, err := h.gostService.GetGosts(
		limit,
		offset,
		status,
		sourceType,
		search,
		adoptionFrom,
		adoptionTo,
		effectiveFrom,
		effectiveTo,
	)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить список ГОСТов")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	SendJSONResponse(c, http.StatusOK, result)
}

// HandleGetGostDetail обработчик получения детальной информации о ГОСТе
// @Summary Получить детальную информацию о ГОСТе
// @Description Возвращает детальную информацию о ГОСТе по ID
// @Tags gosts
// @Accept json
// @Produce json
// @Param id path int true "ID ГОСТа"
// @Success 200 {object} map[string]interface{} "Детальная информация о ГОСТе"
// @Failure 404 {object} ErrorResponse "ГОСТ не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/:id [get]
func (h *GostHandler) HandleGetGostDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Неверный ID ГОСТа")
		return
	}

	result, err := h.gostService.GetGostDetail(id)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить ГОСТ")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	SendJSONResponse(c, http.StatusOK, result)
}

// HandleSearchGosts обработчик поиска ГОСТов
// @Summary Поиск ГОСТов
// @Description Выполняет поиск ГОСТов по номеру, названию или ключевым словам
// @Tags gosts
// @Accept json
// @Produce json
// @Param q query string true "Поисковый запрос"
// @Param limit query int false "Количество записей на странице" default(50)
// @Param offset query int false "Смещение для пагинации" default(0)
// @Success 200 {object} map[string]interface{} "Результаты поиска"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/search [get]
func (h *GostHandler) HandleSearchGosts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		SendJSONError(c, http.StatusBadRequest, "Параметр 'q' обязателен для поиска")
		return
	}

	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	result, err := h.gostService.GetGosts(
		limit,
		offset,
		c.Query("status"),
		c.Query("source_type"),
		query,
		c.Query("adoption_from"),
		c.Query("adoption_to"),
		c.Query("effective_from"),
		c.Query("effective_to"),
	)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось выполнить поиск")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	SendJSONResponse(c, http.StatusOK, result)
}

// HandleGetGostByNumber обработчик получения ГОСТа по номеру
// @Summary Получить ГОСТ по номеру
// @Description Возвращает информацию о ГОСТе по его номеру
// @Tags gosts
// @Accept json
// @Produce json
// @Param number path string true "Номер ГОСТа"
// @Success 200 {object} map[string]interface{} "Информация о ГОСТе"
// @Failure 404 {object} ErrorResponse "ГОСТ не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/number/:number [get]
func (h *GostHandler) HandleGetGostByNumber(c *gin.Context) {
	gostNumber := c.Param("number")
	if gostNumber == "" {
		SendJSONError(c, http.StatusBadRequest, "Номер ГОСТа обязателен")
		return
	}

	result, err := h.gostService.GetGostByNumber(gostNumber)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить ГОСТ")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	SendJSONResponse(c, http.StatusOK, result)
}

// HandleImportGosts обработчик импорта ГОСТов из CSV
// @Summary Импортировать ГОСТы из CSV
// @Description Загружает и импортирует ГОСТы из CSV файла
// @Tags gosts
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV файл с ГОСТами"
// @Param source_type formData string true "Тип источника данных"
// @Param source_url formData string false "URL источника данных"
// @Success 200 {object} map[string]interface{} "Результат импорта"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/import [post]
func (h *GostHandler) HandleImportGosts(c *gin.Context) {
	// Парсим multipart/form-data
	err := c.Request.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Не удалось распарсить форму")
		return
	}

	// Получаем файл
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Файл не найден в запросе")
		return
	}
	defer file.Close()

	// Проверяем расширение файла
	filename := header.Filename
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		SendJSONError(c, http.StatusBadRequest, "Файл должен быть в формате CSV")
		return
	}

	// Получаем параметры
	sourceType := c.PostForm("source_type")
	if sourceType == "" {
		SendJSONError(c, http.StatusBadRequest, "Параметр 'source_type' обязателен")
		return
	}

	sourceURL := c.PostForm("source_url")

	// Импортируем данные
	result, err := h.gostService.ImportGosts(file, filename, sourceType, sourceURL)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось импортировать ГОСТы")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	SendJSONResponse(c, http.StatusOK, result)
}

// HandleGetStatistics обработчик получения статистики
// @Summary Получить статистику по базе ГОСТов
// @Description Возвращает статистику по базе ГОСТов
// @Tags gosts
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Статистика"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/statistics [get]
func (h *GostHandler) HandleGetStatistics(c *gin.Context) {
	stats, err := h.gostService.GetStatistics()
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить статистику")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	SendJSONResponse(c, http.StatusOK, stats)
}

// HandleUploadDocument обработчик загрузки документа ГОСТа
// @Summary Загрузить документ для ГОСТа
// @Description Загружает полный текст ГОСТа (PDF/Word)
// @Tags gosts
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID ГОСТа"
// @Param file formData file true "Файл документа (PDF/Word)"
// @Success 200 {object} map[string]interface{} "Информация о загруженном документе"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 404 {object} ErrorResponse "ГОСТ не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/:id/document [post]
func (h *GostHandler) HandleUploadDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Неверный ID ГОСТа")
		return
	}

	// Парсим multipart/form-data
	err = c.Request.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Не удалось распарсить форму")
		return
	}

	// Получаем файл
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Файл не найден в запросе")
		return
	}
	defer file.Close()

	// Проверяем расширение файла
	filename := header.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := []string{".pdf", ".doc", ".docx"}
	isAllowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		SendJSONError(c, http.StatusBadRequest, "Файл должен быть в формате PDF, DOC или DOCX")
		return
	}

	// Создаем директорию для документов
	docDir := filepath.Join("data", "gosts", "documents")
	if err := os.MkdirAll(docDir, 0755); err != nil {
		SendJSONError(c, http.StatusInternalServerError, "Не удалось создать директорию для документов")
		return
	}

	// Сохраняем файл
	timestamp := time.Now().Unix()
	savedFilename := fmt.Sprintf("%d_%d%s", id, timestamp, ext)
	filePath := filepath.Join(docDir, savedFilename)

	outFile, err := os.Create(filePath)
	if err != nil {
		SendJSONError(c, http.StatusInternalServerError, "Не удалось создать файл")
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		os.Remove(filePath) // Удаляем файл при ошибке
		SendJSONError(c, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}

	// Получаем размер файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		os.Remove(filePath)
		SendJSONError(c, http.StatusInternalServerError, "Не удалось получить информацию о файле")
		return
	}

	// Определяем тип файла
	fileType := strings.TrimPrefix(ext, ".")

	// Сохраняем информацию о документе в БД
	result, err := h.gostService.UploadDocument(id, filePath, fileType, fileInfo.Size())
	if err != nil {
		os.Remove(filePath) // Удаляем файл при ошибке
		appErr := apperrors.WrapError(err, "не удалось сохранить информацию о документе")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	SendJSONResponse(c, http.StatusOK, result)
}

// HandleGetDocument обработчик получения документа ГОСТа
// @Summary Получить документ ГОСТа
// @Description Возвращает файл документа ГОСТа
// @Tags gosts
// @Accept json
// @Produce application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document
// @Param id path int true "ID ГОСТа"
// @Param doc_id query int false "ID документа (если не указан, возвращается последний)"
// @Success 200 "Файл документа"
// @Failure 404 {object} ErrorResponse "Документ не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/:id/document [get]
func (h *GostHandler) HandleGetDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		SendJSONError(c, http.StatusBadRequest, "Неверный ID ГОСТа")
		return
	}

	// Получаем детальную информацию о ГОСТе (включая документы)
	gostDetail, err := h.gostService.GetGostDetail(id)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить ГОСТ")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	// Получаем список документов
	documents, ok := gostDetail["documents"].([]interface{})
	if !ok || len(documents) == 0 {
		SendJSONError(c, http.StatusNotFound, "Документы для данного ГОСТа не найдены")
		return
	}

	// Определяем, какой документ вернуть
	var selectedDoc map[string]interface{}
	docIDStr := c.Query("doc_id")
	if docIDStr != "" {
		docID, err := strconv.Atoi(docIDStr)
		if err == nil {
			// Ищем документ по ID
			for _, doc := range documents {
				if docMap, ok := doc.(map[string]interface{}); ok {
					if docIDFloat, ok := docMap["id"].(float64); ok && int(docIDFloat) == docID {
						selectedDoc = docMap
						break
					}
				}
			}
		}
	}

	// Если документ не найден по ID, берем последний
	if selectedDoc == nil {
		if lastDoc, ok := documents[len(documents)-1].(map[string]interface{}); ok {
			selectedDoc = lastDoc
		}
	}

	if selectedDoc == nil {
		SendJSONError(c, http.StatusNotFound, "Документ не найден")
		return
	}

	// Получаем путь к файлу
	filePath, ok := selectedDoc["file_path"].(string)
	if !ok || filePath == "" {
		SendJSONError(c, http.StatusNotFound, "Путь к файлу не найден")
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			SendJSONError(c, http.StatusNotFound, "Файл не найден на диске")
			return
		}
		SendJSONError(c, http.StatusInternalServerError, fmt.Sprintf("Ошибка проверки файла: %v", err))
		return
	}

	// Определяем Content-Type
	fileType, ok := selectedDoc["file_type"].(string)
	if !ok {
		fileType = filepath.Ext(filePath)
	}

	contentType := "application/octet-stream"
	switch strings.ToLower(fileType) {
	case "pdf":
		contentType = "application/pdf"
	case "doc":
		contentType = "application/msword"
	case "docx":
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	// Отправляем файл
	c.Header("Content-Type", contentType)
	c.File(filePath)
}

// HandleExportGosts обработчик экспорта ГОСТов в CSV
// @Summary Экспортировать ГОСТы в CSV
// @Description Экспортирует все ГОСТы с учетом фильтров в CSV формат
// @Tags gosts
// @Accept json
// @Produce text/csv
// @Param status query string false "Фильтр по статусу"
// @Param source_type query string false "Фильтр по типу источника"
// @Param search query string false "Поисковый запрос"
// @Param adoption_from query string false "Дата принятия с (ГГГГ-ММ-ДД)"
// @Param adoption_to query string false "Дата принятия по (ГГГГ-ММ-ДД)"
// @Param effective_from query string false "Дата вступления с (ГГГГ-ММ-ДД)"
// @Param effective_to query string false "Дата вступления по (ГГГГ-ММ-ДД)"
// @Success 200 "CSV файл с ГОСТами"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/gosts/export [get]
func (h *GostHandler) HandleExportGosts(c *gin.Context) {
	// Парсим параметры запроса (те же, что и для списка)
	status := c.Query("status")
	sourceType := c.Query("source_type")
	search := c.Query("search")
	adoptionFrom := c.Query("adoption_from")
	adoptionTo := c.Query("adoption_to")
	effectiveFrom := c.Query("effective_from")
	effectiveTo := c.Query("effective_to")

	// Валидация дат
	dateParams := []struct {
		value string
		name  string
	}{
		{adoptionFrom, "adoption_from"},
		{adoptionTo, "adoption_to"},
		{effectiveFrom, "effective_from"},
		{effectiveTo, "effective_to"},
	}

	for _, param := range dateParams {
		if param.value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", param.value); err != nil {
			SendJSONError(c, http.StatusBadRequest, fmt.Sprintf("Неверный формат даты для %s. Используйте формат ГГГГ-ММ-ДД", param.name))
			return
		}
	}

	// Получаем все ГОСТы для экспорта
	gosts, err := h.gostService.GetAllGostsForExport(
		status,
		sourceType,
		search,
		adoptionFrom,
		adoptionTo,
		effectiveFrom,
		effectiveTo,
	)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить ГОСТы для экспорта")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	// Генерируем имя файла
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("gosts_export_%s.csv", timestamp)

	// Устанавливаем заголовки для скачивания файла
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	// Записываем BOM для корректного отображения кириллицы в Excel
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	// Записываем заголовки CSV
	headers := []string{"Номер ГОСТа", "Название", "Дата принятия", "Дата вступления", "Статус", "Тип источника", "URL источника", "Описание", "Ключевые слова"}
	csvLine := strings.Join(headers, ",") + "\n"
	c.Writer.WriteString(csvLine)

	// Записываем данные
	for _, gost := range gosts {
		// Экранируем кавычки в полях
		escapeCSV := func(s string) string {
			if strings.Contains(s, ",") || strings.Contains(s, "\"") || strings.Contains(s, "\n") {
				s = strings.ReplaceAll(s, "\"", "\"\"")
				return "\"" + s + "\""
			}
			return s
		}

		adoptionDate := ""
		if gost.AdoptionDate != nil {
			adoptionDate = gost.AdoptionDate.Format("2006-01-02")
		}

		effectiveDate := ""
		if gost.EffectiveDate != nil {
			effectiveDate = gost.EffectiveDate.Format("2006-01-02")
		}

		row := []string{
			escapeCSV(gost.GostNumber),
			escapeCSV(gost.Title),
			adoptionDate,
			effectiveDate,
			escapeCSV(gost.Status),
			escapeCSV(gost.SourceType),
			escapeCSV(gost.SourceURL),
			escapeCSV(gost.Description),
			escapeCSV(gost.Keywords),
		}
		csvLine := strings.Join(row, ",") + "\n"
		c.Writer.WriteString(csvLine)
	}

	c.Writer.Flush()
}
