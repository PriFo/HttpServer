package services

import (
	"fmt"
	"io"
	"time"

	"httpserver/database"
	"httpserver/importer"
	apperrors "httpserver/server/errors"
)

// GostService сервис для работы с ГОСТами
type GostService struct {
	gostsDB *database.GostsDB
}

// NewGostService создает новый сервис для работы с ГОСТами
func NewGostService(gostsDB *database.GostsDB) *GostService {
	return &GostService{
		gostsDB: gostsDB,
	}
}

// ImportGosts импортирует ГОСТы из CSV файла
func (s *GostService) ImportGosts(file io.Reader, filename string, sourceType, sourceURL string) (map[string]interface{}, error) {
	// Валидация параметров
	if sourceType == "" {
		return nil, apperrors.NewValidationError("тип источника данных обязателен", nil)
	}

	// Парсим CSV файл напрямую из io.Reader
	records, err := importer.ParseGostCSVFromReader(file)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось распарсить CSV файл", err)
	}

	if len(records) == 0 {
		return nil, apperrors.NewValidationError("CSV файл не содержит записей", nil)
	}

	// Создаем или обновляем источник данных
	source := &database.GostSource{
		SourceName:   sourceType,
		SourceURL:    sourceURL,
		LastSyncDate: timePtr(time.Now()),
		RecordsCount: len(records),
	}

	sourceRecord, err := s.gostsDB.CreateOrUpdateSource(source)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось создать источник данных", err)
	}

	// Импортируем данные
	successCount := 0
	errorCount := 0
	updatedCount := 0
	errors := []string{}
	const maxErrorsToReport = 100 // Ограничиваем количество ошибок в ответе

	for i, record := range records {
		// Валидация обязательных полей
		if record.GostNumber == "" {
			errorCount++
			if len(errors) < maxErrorsToReport {
				errors = append(errors, fmt.Sprintf("Строка %d: отсутствует номер ГОСТа", i+1))
			}
			continue
		}

		if record.Title == "" {
			errorCount++
			if len(errors) < maxErrorsToReport {
				errors = append(errors, fmt.Sprintf("ГОСТ %s: отсутствует название", record.GostNumber))
			}
			continue
		}

		// Проверяем, существует ли уже ГОСТ с таким номером
		existingGost, err := s.gostsDB.GetGostByNumber(record.GostNumber)
		isUpdate := err == nil && existingGost != nil

		gost := &database.Gost{
			GostNumber:    record.GostNumber,
			Title:         record.Title,
			AdoptionDate:  record.AdoptionDate,
			EffectiveDate: record.EffectiveDate,
			Status:        record.Status,
			SourceType:    sourceType,
			SourceID:      &sourceRecord.ID,
			SourceURL:     sourceURL,
			Description:   record.Description,
			Keywords:      record.Keywords,
		}

		_, err = s.gostsDB.CreateOrUpdateGost(gost)
		if err != nil {
			errorCount++
			if len(errors) < maxErrorsToReport {
				errors = append(errors, fmt.Sprintf("ГОСТ %s: %v", record.GostNumber, err))
			}
			continue
		}

		if isUpdate {
			updatedCount++
		}
		successCount++
	}

	result := map[string]interface{}{
		"success":   successCount,
		"updated":   updatedCount,
		"created":   successCount - updatedCount,
		"total":     len(records),
		"errors":    errorCount,
		"source_id": sourceRecord.ID,
	}

	// Добавляем список ошибок только если их не слишком много
	if len(errors) > 0 {
		result["error_list"] = errors
		if errorCount > maxErrorsToReport {
			result["error_list_truncated"] = true
			result["total_errors"] = errorCount
		}
	}

	return result, nil
}

// GetGosts возвращает список ГОСТов с фильтрацией и пагинацией
func (s *GostService) GetGosts(
	limit, offset int,
	status, sourceType, search string,
	adoptionFrom, adoptionTo, effectiveFrom, effectiveTo string,
) (map[string]interface{}, error) {
	var gosts []*database.Gost
	var total int
	var err error

	if search != "" {
		gosts, total, err = s.gostsDB.SearchGosts(
			search,
			limit,
			offset,
			status,
			sourceType,
			adoptionFrom,
			adoptionTo,
			effectiveFrom,
			effectiveTo,
		)
	} else {
		gosts, total, err = s.gostsDB.ListGosts(
			limit,
			offset,
			status,
			sourceType,
			adoptionFrom,
			adoptionTo,
			effectiveFrom,
			effectiveTo,
		)
	}

	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить список ГОСТов", err)
	}

	// Преобразуем в интерфейсы для JSON
	gostsInterface := make([]interface{}, 0, len(gosts))
	for _, gost := range gosts {
		gostsInterface = append(gostsInterface, map[string]interface{}{
			"id":             gost.ID,
			"gost_number":    gost.GostNumber,
			"title":          gost.Title,
			"adoption_date":  formatDate(gost.AdoptionDate),
			"effective_date": formatDate(gost.EffectiveDate),
			"status":         gost.Status,
			"source_type":    gost.SourceType,
			"source_url":     gost.SourceURL,
			"description":    gost.Description,
			"keywords":       gost.Keywords,
			"created_at":     gost.CreatedAt.Format(time.RFC3339),
			"updated_at":     gost.UpdatedAt.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"gosts":  gostsInterface,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, nil
}

// GetAllGostsForExport возвращает все ГОСТы с фильтрацией для экспорта (без пагинации)
func (s *GostService) GetAllGostsForExport(
	status, sourceType, search string,
	adoptionFrom, adoptionTo, effectiveFrom, effectiveTo string,
) ([]*database.Gost, error) {
	var gosts []*database.Gost
	var err error

	// Используем большой лимит для получения всех записей
	// В реальности лучше использовать отдельный метод без лимита
	const maxLimit = 1000000 // Практически без ограничений

	if search != "" {
		gosts, _, err = s.gostsDB.SearchGosts(
			search,
			maxLimit,
			0, // offset = 0
			status,
			sourceType,
			adoptionFrom,
			adoptionTo,
			effectiveFrom,
			effectiveTo,
		)
	} else {
		gosts, _, err = s.gostsDB.ListGosts(
			maxLimit,
			0, // offset = 0
			status,
			sourceType,
			adoptionFrom,
			adoptionTo,
			effectiveFrom,
			effectiveTo,
		)
	}

	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить ГОСТы для экспорта", err)
	}

	return gosts, nil
}

// GetGostDetail возвращает детальную информацию о ГОСТе
func (s *GostService) GetGostDetail(id int) (map[string]interface{}, error) {
	gost, err := s.gostsDB.GetGost(id)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ГОСТ не найден", err)
	}

	// Получаем документы
	documents, err := s.gostsDB.GetDocumentsByGostID(id)
	if err != nil {
		// Не критично, если документы не найдены
		documents = []*database.GostDocument{}
	}

	documentsInterface := make([]interface{}, 0, len(documents))
	for _, doc := range documents {
		documentsInterface = append(documentsInterface, map[string]interface{}{
			"id":          doc.ID,
			"file_path":   doc.FilePath,
			"file_type":   doc.FileType,
			"file_size":   doc.FileSize,
			"uploaded_at": doc.UploadedAt.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"id":             gost.ID,
		"gost_number":    gost.GostNumber,
		"title":          gost.Title,
		"adoption_date":  formatDate(gost.AdoptionDate),
		"effective_date": formatDate(gost.EffectiveDate),
		"status":         gost.Status,
		"source_type":    gost.SourceType,
		"source_url":     gost.SourceURL,
		"description":    gost.Description,
		"keywords":       gost.Keywords,
		"documents":      documentsInterface,
		"created_at":     gost.CreatedAt.Format(time.RFC3339),
		"updated_at":     gost.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetGostByNumber возвращает ГОСТ по номеру
func (s *GostService) GetGostByNumber(gostNumber string) (map[string]interface{}, error) {
	gost, err := s.gostsDB.GetGostByNumber(gostNumber)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ГОСТ не найден", err)
	}

	return map[string]interface{}{
		"id":             gost.ID,
		"gost_number":    gost.GostNumber,
		"title":          gost.Title,
		"adoption_date":  formatDate(gost.AdoptionDate),
		"effective_date": formatDate(gost.EffectiveDate),
		"status":         gost.Status,
		"source_type":    gost.SourceType,
		"source_url":     gost.SourceURL,
		"description":    gost.Description,
		"keywords":       gost.Keywords,
		"created_at":     gost.CreatedAt.Format(time.RFC3339),
		"updated_at":     gost.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetStatistics возвращает статистику по базе ГОСТов
func (s *GostService) GetStatistics() (map[string]interface{}, error) {
	stats, err := s.gostsDB.GetStatistics()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить статистику", err)
	}

	return stats, nil
}

// UploadDocument загружает документ для ГОСТа
func (s *GostService) UploadDocument(gostID int, filePath, fileType string, fileSize int64) (map[string]interface{}, error) {
	// Проверяем существование ГОСТа
	_, err := s.gostsDB.GetGost(gostID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ГОСТ не найден", err)
	}

	doc := &database.GostDocument{
		GostID:   gostID,
		FilePath: filePath,
		FileType: fileType,
		FileSize: fileSize,
	}

	docRecord, err := s.gostsDB.AddDocument(doc)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось добавить документ", err)
	}

	return map[string]interface{}{
		"id":          docRecord.ID,
		"gost_id":     docRecord.GostID,
		"file_path":   docRecord.FilePath,
		"file_type":   docRecord.FileType,
		"file_size":   docRecord.FileSize,
		"uploaded_at": docRecord.UploadedAt.Format(time.RFC3339),
	}, nil
}

// Вспомогательные функции

func timePtr(t time.Time) *time.Time {
	return &t
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
