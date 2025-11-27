package services

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"httpserver/database"
	"httpserver/importer"
	apperrors "httpserver/server/errors"
)

// GISPService сервис для работы с GISP (gisp.gov.ru)
type GISPService struct {
	serviceDB *database.ServiceDB
}

// NewGISPService создает новый сервис для работы с GISP
func NewGISPService(serviceDB *database.ServiceDB) *GISPService {
	return &GISPService{
		serviceDB: serviceDB,
	}
}

// ImportNomenclatures импортирует номенклатуры из Excel файла
func (s *GISPService) ImportNomenclatures(file io.Reader, filename string) (map[string]interface{}, error) {
	// Проверяем, что file не nil
	if file == nil {
		return nil, apperrors.NewValidationError("файл не может быть nil", nil)
	}

	// Проверяем расширение файла
	filenameLower := filepath.Ext(filename)
	if filenameLower != ".xlsx" && filenameLower != ".xls" {
		return nil, apperrors.NewValidationError("файл должен быть в формате Excel (.xlsx или .xls)", nil)
	}

	// Создаем временный файл
	tempDir := filepath.Join("data", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, apperrors.NewInternalError("не удалось создать временную директорию", err)
	}

	tempFile := filepath.Join(tempDir, fmt.Sprintf("gisp_import_%d_%s", time.Now().Unix(), filename))
	outFile, err := os.Create(tempFile)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось создать временный файл", err)
	}
	defer outFile.Close()
	defer os.Remove(tempFile) // Удаляем временный файл после обработки

	// Копируем содержимое файла
	_, err = io.Copy(outFile, file)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось сохранить файл", err)
	}
	outFile.Close()

	// Парсим Excel файл
	records, err := importer.ParseGISPExcelFile(tempFile)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось распарсить Excel файл", err)
	}

	// Получаем или создаем системный проект
	systemProject, err := s.serviceDB.GetOrCreateSystemProject()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить системный проект", err)
	}

	// Импортируем данные
	nomenclatureImporter := importer.NewNomenclatureImporter(s.serviceDB)
	result, err := nomenclatureImporter.ImportNomenclatures(records, systemProject.ID)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось импортировать номенклатуры", err)
	}

	// Преобразуем результат в map
	return map[string]interface{}{
		"success":    result.Success,
		"total":      result.Total,
		"updated":    result.Updated,
		"errors":     result.Errors,
		"started":    result.Started,
		"completed":  result.Completed,
		"duration":   result.Duration,
	}, nil
}

// GetNomenclatures возвращает список номенклатур с фильтрацией
func (s *GISPService) GetNomenclatures(limit, offset int, search, okpd2Code, tnvedCode string, manufacturerID *int) (map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	// Получаем системный проект
	systemProject, err := s.serviceDB.GetOrCreateSystemProject()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить системный проект", err)
	}

	// Строим запрос с фильтрами
	whereClause := "cb.client_project_id = ? AND cb.category = 'nomenclature' AND cb.source_database = 'gisp_gov_ru'"
	args := []interface{}{systemProject.ID}

	// Фильтр по поисковому запросу
	if search != "" {
		whereClause += " AND (cb.original_name LIKE ? OR cb.normalized_name LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Фильтр по производителю
	if manufacturerID != nil && *manufacturerID > 0 {
		whereClause += " AND cb.manufacturer_benchmark_id = ?"
		args = append(args, *manufacturerID)
	}

	// Фильтр по OKPD2 (через okpd2_reference_id)
	if okpd2Code != "" {
		// Нужно найти reference_id по коду OKPD2
		// Пока используем простой подход - ищем в attributes или создаем отдельный запрос
		whereClause += " AND EXISTS (SELECT 1 FROM client_benchmarks okpd2 WHERE okpd2.id = cb.okpd2_reference_id AND (okpd2.original_name LIKE ? OR okpd2.normalized_name LIKE ?))"
		okpd2Pattern := "%" + okpd2Code + "%"
		args = append(args, okpd2Pattern, okpd2Pattern)
	}

	// Фильтр по TNVED (через tnved_reference_id)
	if tnvedCode != "" {
		whereClause += " AND EXISTS (SELECT 1 FROM client_benchmarks tnved WHERE tnved.id = cb.tnved_reference_id AND (tnved.original_name LIKE ? OR tnved.normalized_name LIKE ?))"
		tnvedPattern := "%" + tnvedCode + "%"
		args = append(args, tnvedPattern, tnvedPattern)
	}

	// Получаем общее количество
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM client_benchmarks cb WHERE %s", whereClause)
	var total int
	err = s.serviceDB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить количество номенклатур", err)
	}

	// Получаем записи с пагинацией
	query := fmt.Sprintf(`
		SELECT cb.id, cb.client_project_id, cb.original_name, cb.normalized_name, cb.category,
		       COALESCE(cb.subcategory, '') as subcategory,
		       COALESCE(cb.attributes, '') as attributes, cb.quality_score, cb.is_approved,
		       COALESCE(cb.approved_by, '') as approved_by, cb.approved_at,
		       COALESCE(cb.source_database, '') as source_database, cb.usage_count,
		       cb.manufacturer_benchmark_id, cb.okpd2_reference_id, cb.tnved_reference_id, cb.tu_gost_reference_id,
		       cb.created_at, cb.updated_at
		FROM client_benchmarks cb
		WHERE %s
		ORDER BY cb.is_approved DESC, cb.quality_score DESC, cb.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	queryArgs := append(args, limit, offset)
	rows, err := s.serviceDB.Query(query, queryArgs...)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить номенклатуры", err)
	}
	defer rows.Close()

	var nomenclatures []map[string]interface{}
	for rows.Next() {
		var id, projectID, usageCount int
		var originalName, normalizedName, category, subcategory, attributes, approvedBy, sourceDatabase string
		var qualityScore float64
		var isApproved bool
		var approvedAt sql.NullTime
		var manufacturerID, okpd2RefID, tnvedRefID, tuGostRefID sql.NullInt64
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &projectID, &originalName, &normalizedName, &category,
			&subcategory, &attributes, &qualityScore, &isApproved,
			&approvedBy, &approvedAt, &sourceDatabase, &usageCount,
			&manufacturerID, &okpd2RefID, &tnvedRefID, &tuGostRefID,
			&createdAt, &updatedAt,
		)
		if err != nil {
			continue // Пропускаем ошибки сканирования
		}

		nomenclature := map[string]interface{}{
			"id":              id,
			"original_name":   originalName,
			"normalized_name": normalizedName,
			"category":        category,
			"subcategory":     subcategory,
			"quality_score":   qualityScore,
			"is_approved":     isApproved,
			"source_database": sourceDatabase,
			"usage_count":     usageCount,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
		}

		if approvedAt.Valid {
			nomenclature["approved_at"] = approvedAt.Time
		}
		if manufacturerID.Valid {
			nomenclature["manufacturer_id"] = int(manufacturerID.Int64)
		}
		if okpd2RefID.Valid {
			nomenclature["okpd2_reference_id"] = int(okpd2RefID.Int64)
		}
		if tnvedRefID.Valid {
			nomenclature["tnved_reference_id"] = int(tnvedRefID.Int64)
		}
		if tuGostRefID.Valid {
			nomenclature["tu_gost_reference_id"] = int(tuGostRefID.Int64)
		}

		nomenclatures = append(nomenclatures, nomenclature)
	}

	return map[string]interface{}{
		"total":         total,
		"limit":         limit,
		"offset":        offset,
		"nomenclatures": nomenclatures,
	}, nil
}

// GetNomenclatureDetail возвращает детальную информацию о номенклатуре
func (s *GISPService) GetNomenclatureDetail(id int) (map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	benchmark, err := s.serviceDB.GetClientBenchmark(id)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить бенчмарк", err)
	}

	if benchmark == nil {
		return nil, apperrors.NewNotFoundError("номенклатура не найдена", nil)
	}

	if benchmark.Category != "nomenclature" || benchmark.SourceDatabase != "gisp_gov_ru" {
		return nil, apperrors.NewValidationError("это не номенклатура GISP", nil)
	}

	result := map[string]interface{}{
		"id":              benchmark.ID,
		"original_name":   benchmark.OriginalName,
		"normalized_name": benchmark.NormalizedName,
		"category":        benchmark.Category,
		"subcategory":     benchmark.Subcategory,
		"quality_score":   benchmark.QualityScore,
		"is_approved":      benchmark.IsApproved,
		"source_database": benchmark.SourceDatabase,
		"usage_count":      benchmark.UsageCount,
		"created_at":       benchmark.CreatedAt,
		"updated_at":        benchmark.UpdatedAt,
	}

	// Получаем связанные данные - производитель
	if benchmark.ManufacturerBenchmarkID != nil {
		manufacturer, err := s.serviceDB.GetClientBenchmark(*benchmark.ManufacturerBenchmarkID)
		if err == nil && manufacturer != nil {
			result["manufacturer"] = map[string]interface{}{
				"id":              manufacturer.ID,
				"original_name":   manufacturer.OriginalName,
				"normalized_name": manufacturer.NormalizedName,
				"legal_form":       manufacturer.LegalForm,
				"tax_id":           manufacturer.TaxID,
			}
		}
	}

	// Получаем связанные данные - OKPD2
	if benchmark.OKPD2ReferenceID != nil {
		okpd2, err := s.serviceDB.GetClientBenchmark(*benchmark.OKPD2ReferenceID)
		if err == nil && okpd2 != nil {
			result["okpd2"] = map[string]interface{}{
				"id":              okpd2.ID,
				"original_name":   okpd2.OriginalName,
				"normalized_name": okpd2.NormalizedName,
			}
		}
	}

	// Получаем связанные данные - TNVED
	if benchmark.TNVEDReferenceID != nil {
		tnved, err := s.serviceDB.GetClientBenchmark(*benchmark.TNVEDReferenceID)
		if err == nil && tnved != nil {
			result["tnved"] = map[string]interface{}{
				"id":              tnved.ID,
				"original_name":   tnved.OriginalName,
				"normalized_name": tnved.NormalizedName,
			}
		}
	}

	// Получаем связанные данные - TU GOST
	if benchmark.TUGOSTReferenceID != nil {
		tuGost, err := s.serviceDB.GetClientBenchmark(*benchmark.TUGOSTReferenceID)
		if err == nil && tuGost != nil {
			result["tu_gost"] = map[string]interface{}{
				"id":              tuGost.ID,
				"original_name":   tuGost.OriginalName,
				"normalized_name": tuGost.NormalizedName,
			}
		}
	}

	return result, nil
}

// GetReferenceBooks возвращает статистику по справочникам
func (s *GISPService) GetReferenceBooks() (map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	// Получаем системный проект
	systemProject, err := s.serviceDB.GetOrCreateSystemProject()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить системный проект", err)
	}

	// Статистика по OKPD2
	var okpd2Total, okpd2Used int
	err = s.serviceDB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT cb.okpd2_reference_id) as used
		FROM client_benchmarks cb
		WHERE cb.client_project_id = ? 
		  AND cb.category = 'nomenclature'
		  AND cb.source_database = 'gisp_gov_ru'
		  AND cb.okpd2_reference_id IS NOT NULL
	`, systemProject.ID).Scan(&okpd2Used, &okpd2Total)
	if err != nil {
		okpd2Total = 0
		okpd2Used = 0
	}

	// Общее количество записей OKPD2 в справочниках
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'reference'
		  AND (normalized_name LIKE '%ОКПД%' OR normalized_name LIKE '%OKPD%' OR subcategory = 'okpd2')
	`, systemProject.ID).Scan(&okpd2Total)
	if err != nil {
		okpd2Total = 0
	}

	// Статистика по TNVED
	var tnvedTotal, tnvedUsed int
	err = s.serviceDB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT cb.tnved_reference_id) as used
		FROM client_benchmarks cb
		WHERE cb.client_project_id = ? 
		  AND cb.category = 'nomenclature'
		  AND cb.source_database = 'gisp_gov_ru'
		  AND cb.tnved_reference_id IS NOT NULL
	`, systemProject.ID).Scan(&tnvedUsed, &tnvedTotal)
	if err != nil {
		tnvedTotal = 0
		tnvedUsed = 0
	}

	// Общее количество записей TNVED в справочниках
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'reference'
		  AND (normalized_name LIKE '%ТНВЭД%' OR normalized_name LIKE '%TNVED%' OR subcategory = 'tnved')
	`, systemProject.ID).Scan(&tnvedTotal)
	if err != nil {
		tnvedTotal = 0
	}

	// Статистика по TU GOST
	var tuGostTotal, tuGostUsed int
	err = s.serviceDB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT cb.tu_gost_reference_id) as used
		FROM client_benchmarks cb
		WHERE cb.client_project_id = ? 
		  AND cb.category = 'nomenclature'
		  AND cb.source_database = 'gisp_gov_ru'
		  AND cb.tu_gost_reference_id IS NOT NULL
	`, systemProject.ID).Scan(&tuGostUsed, &tuGostTotal)
	if err != nil {
		tuGostTotal = 0
		tuGostUsed = 0
	}

	// Общее количество записей TU GOST в справочниках
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'reference'
		  AND (normalized_name LIKE '%ТУ%' OR normalized_name LIKE '%ГОСТ%' OR subcategory = 'tu_gost')
	`, systemProject.ID).Scan(&tuGostTotal)
	if err != nil {
		tuGostTotal = 0
	}

	return map[string]interface{}{
		"okpd2": map[string]int{
			"total_records": okpd2Total,
			"used_records":  okpd2Used,
		},
		"tnved": map[string]int{
			"total_records": tnvedTotal,
			"used_records":  tnvedUsed,
		},
		"tu_gost": map[string]int{
			"total_records": tuGostTotal,
			"used_records":  tuGostUsed,
		},
	}, nil
}

// SearchReferenceBook выполняет поиск в справочниках
func (s *GISPService) SearchReferenceBook(bookType, search string, limit int) (map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if search == "" {
		return nil, apperrors.NewValidationError("поисковый запрос не может быть пустым", nil)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	// Получаем системный проект
	systemProject, err := s.serviceDB.GetOrCreateSystemProject()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить системный проект", err)
	}

	// Определяем subcategory в зависимости от типа справочника
	var subcategoryFilter string
	switch bookType {
	case "okpd2":
		subcategoryFilter = "okpd2"
	case "tnved":
		subcategoryFilter = "tnved"
	case "tu_gost", "tu-gost":
		subcategoryFilter = "tu_gost"
	default:
		subcategoryFilter = ""
	}

	// Строим запрос
	whereClause := "client_project_id = ? AND category = 'reference'"
	args := []interface{}{systemProject.ID}

	if subcategoryFilter != "" {
		whereClause += " AND subcategory = ?"
		args = append(args, subcategoryFilter)
	}

	whereClause += " AND (original_name LIKE ? OR normalized_name LIKE ?)"
	searchPattern := "%" + search + "%"
	args = append(args, searchPattern, searchPattern)

	query := fmt.Sprintf(`
		SELECT id, original_name, normalized_name, subcategory, quality_score, is_approved, created_at
		FROM client_benchmarks
		WHERE %s
		ORDER BY is_approved DESC, quality_score DESC, normalized_name
		LIMIT ?
	`, whereClause)

	args = append(args, limit)
	rows, err := s.serviceDB.Query(query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось выполнить поиск в справочниках", err)
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id int
		var originalName, normalizedName, subcategory string
		var qualityScore float64
		var isApproved bool
		var createdAt time.Time

		err := rows.Scan(&id, &originalName, &normalizedName, &subcategory, &qualityScore, &isApproved, &createdAt)
		if err != nil {
			continue
		}

		items = append(items, map[string]interface{}{
			"id":              id,
			"original_name":   originalName,
			"normalized_name": normalizedName,
			"subcategory":     subcategory,
			"quality_score":   qualityScore,
			"is_approved":     isApproved,
			"created_at":      createdAt,
		})
	}

	return map[string]interface{}{
		"type":  bookType,
		"limit": limit,
		"items": items,
		"count": len(items),
	}, nil
}

// GetStatistics возвращает статистику по импортированным данным
func (s *GISPService) GetStatistics() (map[string]interface{}, error) {
	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	// Получаем системный проект
	systemProject, err := s.serviceDB.GetOrCreateSystemProject()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить системный проект", err)
	}

	// Общее количество номенклатур
	var totalNomenclatures int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'nomenclature' 
		  AND source_database = 'gisp_gov_ru'
	`, systemProject.ID).Scan(&totalNomenclatures)
	if err != nil {
		totalNomenclatures = 0
	}

	// Одобренные номенклатуры
	var approvedNomenclatures int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'nomenclature' 
		  AND source_database = 'gisp_gov_ru'
		  AND is_approved = TRUE
	`, systemProject.ID).Scan(&approvedNomenclatures)
	if err != nil {
		approvedNomenclatures = 0
	}

	// Номенклатуры с OKPD2
	var withOKPD2 int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'nomenclature' 
		  AND source_database = 'gisp_gov_ru'
		  AND okpd2_reference_id IS NOT NULL
	`, systemProject.ID).Scan(&withOKPD2)
	if err != nil {
		withOKPD2 = 0
	}

	// Номенклатуры с TNVED
	var withTNVED int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'nomenclature' 
		  AND source_database = 'gisp_gov_ru'
		  AND tnved_reference_id IS NOT NULL
	`, systemProject.ID).Scan(&withTNVED)
	if err != nil {
		withTNVED = 0
	}

	// Номенклатуры с TU GOST
	var withTUGOST int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'nomenclature' 
		  AND source_database = 'gisp_gov_ru'
		  AND tu_gost_reference_id IS NOT NULL
	`, systemProject.ID).Scan(&withTUGOST)
	if err != nil {
		withTUGOST = 0
	}

	// Номенклатуры с производителем
	var withManufacturer int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'nomenclature' 
		  AND source_database = 'gisp_gov_ru'
		  AND manufacturer_benchmark_id IS NOT NULL
	`, systemProject.ID).Scan(&withManufacturer)
	if err != nil {
		withManufacturer = 0
	}

	// Общее количество производителей
	var totalManufacturers int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(DISTINCT manufacturer_benchmark_id) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'nomenclature' 
		  AND source_database = 'gisp_gov_ru'
		  AND manufacturer_benchmark_id IS NOT NULL
	`, systemProject.ID).Scan(&totalManufacturers)
	if err != nil {
		totalManufacturers = 0
	}

	// Общее количество OKPD2 в справочниках
	var okpd2Total int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'reference'
		  AND (normalized_name LIKE '%ОКПД%' OR normalized_name LIKE '%OKPD%' OR subcategory = 'okpd2')
	`, systemProject.ID).Scan(&okpd2Total)
	if err != nil {
		okpd2Total = 0
	}

	// Общее количество TNVED в справочниках
	var tnvedTotal int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'reference'
		  AND (normalized_name LIKE '%ТНВЭД%' OR normalized_name LIKE '%TNVED%' OR subcategory = 'tnved')
	`, systemProject.ID).Scan(&tnvedTotal)
	if err != nil {
		tnvedTotal = 0
	}

	// Общее количество TU GOST в справочниках
	var tuGostTotal int
	err = s.serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'reference'
		  AND (normalized_name LIKE '%ТУ%' OR normalized_name LIKE '%ГОСТ%' OR subcategory = 'tu_gost')
	`, systemProject.ID).Scan(&tuGostTotal)
	if err != nil {
		tuGostTotal = 0
	}

	return map[string]interface{}{
		"total_nomenclatures":    totalNomenclatures,
		"approved_nomenclatures": approvedNomenclatures,
		"total_manufacturers":     totalManufacturers,
		"with_okpd2":             withOKPD2,
		"with_tnved":             withTNVED,
		"with_tu_gost":           withTUGOST,
		"with_manufacturer":      withManufacturer,
		"okpd2_total":            okpd2Total,
		"tnved_total":            tnvedTotal,
		"tu_gost_total":          tuGostTotal,
	}, nil
}

