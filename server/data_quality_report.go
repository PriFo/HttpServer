package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/extractors"
)

// DataQualityReport основной отчет о качестве данных
type DataQualityReport struct {
	ReportMetadata    DataQualityReportMetadata `json:"metadata"`
	OverallScore      OverallQualityScore       `json:"overall_score"`
	CounterpartyStats CounterpartyQualityStats  `json:"counterparty_stats"`
	NomenclatureStats NomenclatureQualityStats  `json:"nomenclature_stats"`
	DatabaseBreakdown []DatabaseQualityStats    `json:"database_breakdown"`
	Recommendations   []string                  `json:"recommendations"`
}

// DataQualityReportMetadata метаданные отчета о качестве данных
type DataQualityReportMetadata struct {
	GeneratedAt    time.Time `json:"generated_at"`
	ReportVersion  string    `json:"report_version"`
	TotalDatabases int       `json:"total_databases"`
	TotalProjects  int       `json:"total_projects"`
}

// OverallQualityScore общая оценка качества
type OverallQualityScore struct {
	Score        float64 `json:"score"`        // 0-100
	Completeness float64 `json:"completeness"` // % полноты данных
	Uniqueness   float64 `json:"uniqueness"`   // % уникальности (100 - % дубликатов)
	Consistency  float64 `json:"consistency"`  // % консистентности
	DataQuality  string  `json:"data_quality"` // "excellent", "good", "fair", "poor"
}

// CounterpartyQualityStats статистика по контрагентам
type CounterpartyQualityStats struct {
	TotalRecords        int                 `json:"total_records"`
	CompletenessScore   float64             `json:"completeness_score"`       // % записей с именем или ИНН/БИН
	PotentialDupRate    float64             `json:"potential_duplicate_rate"` // % записей с неуникальным ИНН/БИН или именем
	TopInconsistencies  []InconsistencyStat `json:"top_inconsistencies"`
	NameLengthStats     NameLengthStats     `json:"name_length_stats"`
	RecordsWithName     int                 `json:"records_with_name"`
	RecordsWithINN      int                 `json:"records_with_inn"`
	RecordsWithBIN      int                 `json:"records_with_bin"`
	RecordsWithoutName  int                 `json:"records_without_name"`
	RecordsWithoutTaxID int                 `json:"records_without_tax_id"`
	InvalidINNFormat    int                 `json:"invalid_inn_format"`
	InvalidBINFormat    int                 `json:"invalid_bin_format"`
}

// NomenclatureQualityStats статистика по номенклатуре
type NomenclatureQualityStats struct {
	TotalRecords          int                   `json:"total_records"`
	CompletenessScore     float64               `json:"completeness_score"`       // % записей с названием или артикулом/SKU
	PotentialDupRate      float64               `json:"potential_duplicate_rate"` // % записей с неуникальным названием или артикулом
	TopInconsistencies    []InconsistencyStat   `json:"top_inconsistencies"`
	NameLengthStats       NameLengthStats       `json:"name_length_stats"`
	RecordsWithName       int                   `json:"records_with_name"`
	RecordsWithoutName    int                   `json:"records_without_name"`
	RecordsWithArticle    int                   `json:"records_with_article"`    // Записи с артикулом
	RecordsWithoutArticle int                   `json:"records_without_article"` // Записи без артикула
	RecordsWithSKU        int                   `json:"records_with_sku"`        // Записи с SKU
	RecordsWithoutSKU     int                   `json:"records_without_sku"`     // Записи без SKU
	UnitOfMeasureStats    UnitOfMeasureStats    `json:"unit_of_measure_stats"`   // Статистика по единицам измерения
	AttributeCompleteness AttributeCompleteness `json:"attribute_completeness"`  // Заполненность атрибутов
}

// UnitOfMeasureStats статистика по единицам измерения
type UnitOfMeasureStats struct {
	UniqueCount     int             `json:"unique_count"`    // Количество уникальных ЕИ
	TopVariations   []UnitVariation `json:"top_variations"`  // Топ-10 вариаций
	Inconsistencies []string        `json:"inconsistencies"` // Список неконсистентных вариаций (например, "шт", "штука", "штук")
}

// UnitVariation вариация единицы измерения
type UnitVariation struct {
	Unit  string `json:"unit"`
	Count int    `json:"count"`
}

// AttributeCompleteness заполненность атрибутов
type AttributeCompleteness struct {
	BrandPercent            float64         `json:"brand_percent"`        // % записей с брендом
	ManufacturerPercent     float64         `json:"manufacturer_percent"` // % записей с производителем
	CountryPercent          float64         `json:"country_percent"`      // % записей со страной происхождения
	RecordsWithBrand        int             `json:"records_with_brand"`
	RecordsWithManufacturer int             `json:"records_with_manufacturer"`
	RecordsWithCountry      int             `json:"records_with_country"`
	TopBrands               []UnitVariation `json:"top_brands"` // Топ-10 брендов
}

// DatabaseQualityStats статистика по базе данных
type DatabaseQualityStats struct {
	DatabaseID           int     `json:"database_id"`
	DatabaseName         string  `json:"database_name"`
	FilePath             string  `json:"file_path"`
	ProjectID            int     `json:"project_id"`
	ProjectName          string  `json:"project_name"`
	ClientID             int     `json:"client_id"`
	Counterparties       int     `json:"counterparties"`
	Nomenclature         int     `json:"nomenclature"`
	CompletenessScore    float64 `json:"completeness_score"`
	PotentialDupRate     float64 `json:"potential_duplicate_rate"`
	InconsistenciesCount int     `json:"inconsistencies_count"`
	Status               string  `json:"status"` // "success", "error", "skipped"
	ErrorMessage         string  `json:"error_message,omitempty"`
}

// InconsistencyStat тип несоответствия с примером
type InconsistencyStat struct {
	Type        string `json:"type"`
	Count       int    `json:"count"`
	Example     string `json:"example"`
	Description string `json:"description"`
}

// NameLengthStats статистика длины имен
type NameLengthStats struct {
	Min     int     `json:"min"`
	Max     int     `json:"max"`
	Average float64 `json:"average"`
}

// handleGenerateDataQualityReport обработчик для генерации отчета о качестве данных
func (s *Server) handleGenerateDataQualityReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим тело запроса (опционально - project_id)
	type Request struct {
		ProjectID *int `json:"project_id,omitempty"`
	}

	var req Request
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request body: %v, using defaults", err)
			// Продолжаем с пустым запросом (все проекты)
		}
	}

	// Валидация project_id, если указан
	if req.ProjectID != nil && *req.ProjectID <= 0 {
		s.writeJSONError(w, r, "Invalid project_id: must be positive integer", http.StatusBadRequest)
		return
	}

	// Генерируем отчет
	report, err := s.generateDataQualityReport(req.ProjectID)
	if err != nil {
		log.Printf("Error generating data quality report: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to generate data quality report: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, report, http.StatusOK)
}

// generateDataQualityReport генерирует отчет о качестве данных
func (s *Server) generateDataQualityReport(projectID *int) (*DataQualityReport, error) {
	report := &DataQualityReport{
		ReportMetadata: DataQualityReportMetadata{
			GeneratedAt:   time.Now(),
			ReportVersion: "1.0",
		},
		CounterpartyStats: CounterpartyQualityStats{},
		NomenclatureStats: NomenclatureQualityStats{},
		DatabaseBreakdown: []DatabaseQualityStats{},
		Recommendations:   []string{},
	}

	// Получаем все проекты
	var projects []*database.ClientProject
	if projectID != nil {
		project, err := s.serviceDB.GetClientProject(*projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get project: %w", err)
		}
		projects = []*database.ClientProject{project}
	} else {
		// Получаем все активные проекты через прямой SQL запрос
		serviceDB := s.serviceDB.GetDB()
		rows, err := serviceDB.Query(`
			SELECT id, client_id, name, project_type, description, source_system, 
			       status, target_quality_score, created_at, updated_at
			FROM client_projects
			WHERE status = 'active'
		`)
		if err != nil {
			return nil, fmt.Errorf("failed to get projects: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			project := &database.ClientProject{}
			err := rows.Scan(
				&project.ID, &project.ClientID, &project.Name, &project.ProjectType,
				&project.Description, &project.SourceSystem, &project.Status,
				&project.TargetQualityScore, &project.CreatedAt, &project.UpdatedAt,
			)
			if err != nil {
				log.Printf("Error scanning project: %v", err)
				continue
			}
			projects = append(projects, project)
		}
		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating projects: %w", err)
		}
	}

	report.ReportMetadata.TotalProjects = len(projects)

	// Агрегаторы для общей статистики контрагентов
	var totalCounterparties int
	var totalNomenclature int
	var totalWithName int
	var totalWithINN int
	var totalWithBIN int
	var totalWithoutName int
	var totalWithoutTaxID int
	var totalInvalidINN int
	var totalInvalidBIN int
	var totalDuplicateRecords int
	var totalInconsistencies int

	// Агрегаторы для статистики номенклатуры
	var totalNomWithName int
	var totalNomWithoutName int
	var totalNomWithArticle int
	var totalNomWithoutArticle int
	var totalNomWithSKU int
	var totalNomWithoutSKU int
	var totalNomDuplicateRecords int
	var allNomStats []NomenclatureQualityStats

	// Анализируем каждую БД каждого проекта
	for _, project := range projects {
		databases, err := s.serviceDB.GetProjectDatabases(project.ID, false)
		if err != nil {
			log.Printf("Error getting databases for project %d: %v", project.ID, err)
			continue
		}

		for _, projectDB := range databases {
			if !projectDB.IsActive {
				continue
			}

			dbStats := DatabaseQualityStats{
				DatabaseID:   projectDB.ID,
				DatabaseName: projectDB.Name,
				FilePath:     projectDB.FilePath,
				ProjectID:    project.ID,
				ProjectName:  project.Name,
				ClientID:     project.ClientID,
				Status:       "success",
			}

			// Анализируем БД и получаем детальную статистику
			cpStats, nomStats, err := s.analyzeDatabase(projectDB, &dbStats)
			if err != nil {
				log.Printf("Error analyzing database %s: %v", projectDB.FilePath, err)
				dbStats.Status = "error"
				dbStats.ErrorMessage = err.Error()
				report.DatabaseBreakdown = append(report.DatabaseBreakdown, dbStats)
				continue
			}

			// Агрегируем статистику контрагентов
			totalCounterparties += dbStats.Counterparties
			totalInconsistencies += dbStats.InconsistenciesCount
			if cpStats != nil {
				totalWithName += cpStats.RecordsWithName
				totalWithINN += cpStats.RecordsWithINN
				totalWithBIN += cpStats.RecordsWithBIN
				totalWithoutName += cpStats.RecordsWithoutName
				totalWithoutTaxID += cpStats.RecordsWithoutTaxID
				totalInvalidINN += cpStats.InvalidINNFormat
				totalInvalidBIN += cpStats.InvalidBINFormat
				// Подсчитываем дубликаты из этой БД
				if cpStats.TotalRecords > 0 {
					dbDuplicateCount := int(float64(cpStats.TotalRecords) * cpStats.PotentialDupRate / 100)
					totalDuplicateRecords += dbDuplicateCount
				}
			}

			// Агрегируем статистику номенклатуры
			totalNomenclature += dbStats.Nomenclature
			if nomStats != nil {
				allNomStats = append(allNomStats, *nomStats)
				totalNomWithName += nomStats.RecordsWithName
				totalNomWithoutName += nomStats.RecordsWithoutName
				totalNomWithArticle += nomStats.RecordsWithArticle
				totalNomWithoutArticle += nomStats.RecordsWithoutArticle
				totalNomWithSKU += nomStats.RecordsWithSKU
				totalNomWithoutSKU += nomStats.RecordsWithoutSKU
				if nomStats.TotalRecords > 0 {
					nomDuplicateCount := int(float64(nomStats.TotalRecords) * nomStats.PotentialDupRate / 100)
					totalNomDuplicateRecords += nomDuplicateCount
				}
			}

			report.DatabaseBreakdown = append(report.DatabaseBreakdown, dbStats)
		}
	}

	report.ReportMetadata.TotalDatabases = len(report.DatabaseBreakdown)

	// Вычисляем общую статистику по контрагентам
	if totalCounterparties > 0 {
		report.CounterpartyStats.TotalRecords = totalCounterparties
		report.CounterpartyStats.RecordsWithName = totalWithName
		report.CounterpartyStats.RecordsWithINN = totalWithINN
		report.CounterpartyStats.RecordsWithBIN = totalWithBIN
		report.CounterpartyStats.RecordsWithoutName = totalWithoutName
		report.CounterpartyStats.RecordsWithoutTaxID = totalWithoutTaxID
		report.CounterpartyStats.InvalidINNFormat = totalInvalidINN
		report.CounterpartyStats.InvalidBINFormat = totalInvalidBIN

		// Completeness: % записей с именем или ИНН/БИН
		recordsWithData := totalWithName + totalWithINN + totalWithBIN
		report.CounterpartyStats.CompletenessScore = float64(recordsWithData) / float64(totalCounterparties) * 100

		// Potential duplicate rate
		report.CounterpartyStats.PotentialDupRate = float64(totalDuplicateRecords) / float64(totalCounterparties) * 100
	}

	// Вычисляем общую статистику по номенклатуре
	if totalNomenclature > 0 {
		// Инициализируем вложенные структуры
		report.NomenclatureStats.TopInconsistencies = []InconsistencyStat{}
		report.NomenclatureStats.UnitOfMeasureStats = UnitOfMeasureStats{
			TopVariations:   []UnitVariation{},
			Inconsistencies: []string{},
		}
		report.NomenclatureStats.AttributeCompleteness = AttributeCompleteness{
			TopBrands: []UnitVariation{},
		}

		report.NomenclatureStats.TotalRecords = totalNomenclature
		report.NomenclatureStats.RecordsWithName = totalNomWithName
		report.NomenclatureStats.RecordsWithoutName = totalNomWithoutName
		report.NomenclatureStats.RecordsWithArticle = totalNomWithArticle
		report.NomenclatureStats.RecordsWithoutArticle = totalNomWithoutArticle
		report.NomenclatureStats.RecordsWithSKU = totalNomWithSKU
		report.NomenclatureStats.RecordsWithoutSKU = totalNomWithoutSKU

		// Completeness: % записей с названием или артикулом/SKU
		recordsWithData := totalNomWithName + totalNomWithArticle + totalNomWithSKU
		report.NomenclatureStats.CompletenessScore = float64(recordsWithData) / float64(totalNomenclature) * 100

		// Potential duplicate rate
		report.NomenclatureStats.PotentialDupRate = float64(totalNomDuplicateRecords) / float64(totalNomenclature) * 100

		// Агрегируем статистику единиц измерения и атрибутов из всех БД
		unitMap := make(map[string]int)
		brandMap := make(map[string]int)
		var allInconsistencies []InconsistencyStat
		var nameLengths []int

		for _, nomStat := range allNomStats {
			// Собираем единицы измерения
			for _, unitVar := range nomStat.UnitOfMeasureStats.TopVariations {
				unitMap[unitVar.Unit] += unitVar.Count
			}
			// Собираем бренды (если они есть в статистике)
			for _, brandVar := range nomStat.AttributeCompleteness.TopBrands {
				brandMap[brandVar.Unit] += brandVar.Count
			}
			// Собираем несоответствия
			allInconsistencies = append(allInconsistencies, nomStat.TopInconsistencies...)
		}

		// Формируем топ-10 единиц измерения
		type unitCount struct {
			unit  string
			count int
		}
		units := make([]unitCount, 0, len(unitMap))
		for u, c := range unitMap {
			units = append(units, unitCount{unit: u, count: c})
		}
		sort.Slice(units, func(i, j int) bool {
			return units[i].count > units[j].count
		})
		for i, uc := range units {
			if i >= 10 {
				break
			}
			report.NomenclatureStats.UnitOfMeasureStats.TopVariations = append(
				report.NomenclatureStats.UnitOfMeasureStats.TopVariations,
				UnitVariation{Unit: uc.unit, Count: uc.count},
			)
		}
		report.NomenclatureStats.UnitOfMeasureStats.UniqueCount = len(unitMap)

		// Формируем топ-10 брендов
		if len(brandMap) > 0 {
			type brandCount struct {
				brand string
				count int
			}
			brands := make([]brandCount, 0, len(brandMap))
			for b, c := range brandMap {
				brands = append(brands, brandCount{brand: b, count: c})
			}
			sort.Slice(brands, func(i, j int) bool {
				return brands[i].count > brands[j].count
			})
			for i, bc := range brands {
				if i >= 10 {
					break
				}
				report.NomenclatureStats.AttributeCompleteness.TopBrands = append(
					report.NomenclatureStats.AttributeCompleteness.TopBrands,
					UnitVariation{Unit: bc.brand, Count: bc.count},
				)
			}
		}

		// Агрегируем заполненность атрибутов
		var totalWithBrand, totalWithManufacturer, totalWithCountry int
		for _, nomStat := range allNomStats {
			totalWithBrand += nomStat.AttributeCompleteness.RecordsWithBrand
			totalWithManufacturer += nomStat.AttributeCompleteness.RecordsWithManufacturer
			totalWithCountry += nomStat.AttributeCompleteness.RecordsWithCountry
		}
		report.NomenclatureStats.AttributeCompleteness.RecordsWithBrand = totalWithBrand
		report.NomenclatureStats.AttributeCompleteness.RecordsWithManufacturer = totalWithManufacturer
		report.NomenclatureStats.AttributeCompleteness.RecordsWithCountry = totalWithCountry
		if totalNomenclature > 0 {
			report.NomenclatureStats.AttributeCompleteness.BrandPercent = float64(totalWithBrand) / float64(totalNomenclature) * 100
			report.NomenclatureStats.AttributeCompleteness.ManufacturerPercent = float64(totalWithManufacturer) / float64(totalNomenclature) * 100
			report.NomenclatureStats.AttributeCompleteness.CountryPercent = float64(totalWithCountry) / float64(totalNomenclature) * 100
		}

		// Собираем топ несоответствий (топ-5 по количеству)
		inconsistencyMap := make(map[string]InconsistencyStat)
		for _, inc := range allInconsistencies {
			if existing, exists := inconsistencyMap[inc.Type]; exists {
				existing.Count += inc.Count
				inconsistencyMap[inc.Type] = existing
			} else {
				inconsistencyMap[inc.Type] = inc
			}
		}
		inconsistencies := make([]InconsistencyStat, 0, len(inconsistencyMap))
		for _, inc := range inconsistencyMap {
			inconsistencies = append(inconsistencies, inc)
		}
		sort.Slice(inconsistencies, func(i, j int) bool {
			return inconsistencies[i].Count > inconsistencies[j].Count
		})
		for i, inc := range inconsistencies {
			if i >= 5 {
				break
			}
			report.NomenclatureStats.TopInconsistencies = append(report.NomenclatureStats.TopInconsistencies, inc)
		}

		// Агрегируем статистику длины имен
		for _, nomStat := range allNomStats {
			if nomStat.NameLengthStats.Max > 0 {
				nameLengths = append(nameLengths, nomStat.NameLengthStats.Max)
			}
		}
		if len(nameLengths) > 0 {
			min := nameLengths[0]
			max := nameLengths[0]
			sum := 0
			for _, length := range nameLengths {
				if length < min {
					min = length
				}
				if length > max {
					max = length
				}
				sum += length
			}
			report.NomenclatureStats.NameLengthStats = NameLengthStats{
				Min:     min,
				Max:     max,
				Average: float64(sum) / float64(len(nameLengths)),
			}
		}
	}

	// Вычисляем общую оценку качества
	report.OverallScore = s.calculateDataQualityScore(report)

	// Генерируем рекомендации
	report.Recommendations = s.generateDataQualityRecommendations(report)

	return report, nil
}

// analyzeCounterparties анализирует качество данных контрагентов
func (s *Server) analyzeCounterparties(items []*database.CatalogItem) CounterpartyQualityStats {
	stats := CounterpartyQualityStats{
		TotalRecords:       len(items),
		TopInconsistencies: []InconsistencyStat{},
	}

	// Обработка пустого списка
	if len(items) == 0 {
		return stats
	}

	// Логирование для больших объемов данных
	if len(items) > 10000 {
		log.Printf("Analyzing large counterparty dataset: %d items", len(items))
	}

	// Мапы для подсчета дубликатов
	innMap := make(map[string][]string) // ИНН -> список имен
	binMap := make(map[string][]string) // БИН -> список имен
	nameMap := make(map[string]int)     // нормализованное имя -> количество

	// Статистика
	var withName int
	var withINN int
	var withBIN int
	var withoutName int
	var withoutTaxID int
	var invalidINN int
	var invalidBIN int
	var nameLengths []int

	// Примеры для несоответствий
	var innWithoutNameExample string
	var nameWithoutTaxIDExample string
	var invalidINNExample string
	var invalidBINExample string

	for _, item := range items {
		hasName := item.Name != "" && strings.TrimSpace(item.Name) != ""
		hasINN := false
		hasBIN := false
		var inn string
		var bin string

		// Извлекаем ИНН
		if extractedINN, err := extractors.ExtractINNFromAttributes(item.Attributes); err == nil {
			inn = extractedINN
			hasINN = true
			// Проверяем формат ИНН (должен быть 10 или 12 цифр)
			if len(inn) != 10 && len(inn) != 12 {
				invalidINN++
				if invalidINNExample == "" {
					invalidINNExample = fmt.Sprintf("ИНН: %s (длина: %d)", inn, len(inn))
				}
			}
		}

		// Извлекаем БИН
		if extractedBIN, err := extractors.ExtractBINFromAttributes(item.Attributes); err == nil {
			bin = extractedBIN
			hasBIN = true
			// Проверяем формат БИН (должен быть 12 цифр)
			if len(bin) != 12 {
				invalidBIN++
				if invalidBINExample == "" {
					invalidBINExample = fmt.Sprintf("БИН: %s (длина: %d)", bin, len(bin))
				}
			}
		}

		// Подсчитываем статистику
		if hasName {
			withName++
			nameLengths = append(nameLengths, len(item.Name))
		} else {
			withoutName++
			if hasINN && innWithoutNameExample == "" {
				innWithoutNameExample = fmt.Sprintf("ИНН: %s, имя отсутствует", inn)
			}
			if hasBIN && innWithoutNameExample == "" {
				innWithoutNameExample = fmt.Sprintf("БИН: %s, имя отсутствует", bin)
			}
		}

		if hasINN {
			withINN++
		}
		if hasBIN {
			withBIN++
		}

		if !hasINN && !hasBIN {
			withoutTaxID++
			if hasName && nameWithoutTaxIDExample == "" {
				nameWithoutTaxIDExample = fmt.Sprintf("Имя: %s, ИНН/БИН отсутствует", item.Name)
			}
		}

		// Группируем для поиска дубликатов
		if hasINN && inn != "" {
			innMap[inn] = append(innMap[inn], item.Name)
		}
		if hasBIN && bin != "" {
			binMap[bin] = append(binMap[bin], item.Name)
		}
		if hasName {
			normalizedName := strings.ToLower(strings.TrimSpace(item.Name))
			nameMap[normalizedName]++
		}
	}

	// Вычисляем статистику
	stats.RecordsWithName = withName
	stats.RecordsWithINN = withINN
	stats.RecordsWithBIN = withBIN
	stats.RecordsWithoutName = withoutName
	stats.RecordsWithoutTaxID = withoutTaxID
	stats.InvalidINNFormat = invalidINN
	stats.InvalidBINFormat = invalidBIN

	// Completeness: % записей с именем или ИНН/БИН
	recordsWithData := withName + withINN + withBIN
	if stats.TotalRecords > 0 {
		stats.CompletenessScore = float64(recordsWithData) / float64(stats.TotalRecords) * 100
	}

	// Подсчитываем дубликаты (только записи, которые входят в группы дубликатов)
	duplicateSet := make(map[int]bool)

	// Проверяем дубликаты по ИНН/БИН и именам
	for i, item := range items {
		hasDuplicate := false
		var inn string
		var bin string

		if extractedINN, err := extractors.ExtractINNFromAttributes(item.Attributes); err == nil {
			inn = extractedINN
		}
		if extractedBIN, err := extractors.ExtractBINFromAttributes(item.Attributes); err == nil {
			bin = extractedBIN
		}

		if inn != "" && len(innMap[inn]) > 1 {
			hasDuplicate = true
		}
		if bin != "" && len(binMap[bin]) > 1 {
			hasDuplicate = true
		}

		if item.Name != "" {
			normalizedName := strings.ToLower(strings.TrimSpace(item.Name))
			if nameMap[normalizedName] > 1 {
				hasDuplicate = true
			}
		}

		if hasDuplicate {
			duplicateSet[i] = true
		}
	}

	duplicateCount := len(duplicateSet)

	if stats.TotalRecords > 0 {
		stats.PotentialDupRate = float64(duplicateCount) / float64(stats.TotalRecords) * 100
	}

	// Статистика длины имен
	if len(nameLengths) > 0 {
		min := nameLengths[0]
		max := nameLengths[0]
		sum := 0
		for _, length := range nameLengths {
			if length < min {
				min = length
			}
			if length > max {
				max = length
			}
			sum += length
		}
		stats.NameLengthStats = NameLengthStats{
			Min:     min,
			Max:     max,
			Average: float64(sum) / float64(len(nameLengths)),
		}
	}

	// Формируем список несоответствий
	if withoutName > 0 && innWithoutNameExample != "" {
		stats.TopInconsistencies = append(stats.TopInconsistencies, InconsistencyStat{
			Type:        "inn_without_name",
			Count:       withoutName,
			Example:     innWithoutNameExample,
			Description: "Записи с ИНН/БИН, но без имени",
		})
	}
	if withoutTaxID > 0 && nameWithoutTaxIDExample != "" {
		stats.TopInconsistencies = append(stats.TopInconsistencies, InconsistencyStat{
			Type:        "name_without_tax_id",
			Count:       withoutTaxID,
			Example:     nameWithoutTaxIDExample,
			Description: "Записи с именем, но без ИНН/БИН",
		})
	}
	if invalidINN > 0 && invalidINNExample != "" {
		stats.TopInconsistencies = append(stats.TopInconsistencies, InconsistencyStat{
			Type:        "invalid_inn_format",
			Count:       invalidINN,
			Example:     invalidINNExample,
			Description: "Некорректный формат ИНН (не 10 или 12 цифр)",
		})
	}
	if invalidBIN > 0 && invalidBINExample != "" {
		stats.TopInconsistencies = append(stats.TopInconsistencies, InconsistencyStat{
			Type:        "invalid_bin_format",
			Count:       invalidBIN,
			Example:     invalidBINExample,
			Description: "Некорректный формат БИН (не 12 цифр)",
		})
	}

	return stats
}

// analyzeNomenclature анализирует качество данных номенклатуры
func (s *Server) analyzeNomenclature(items []*database.CatalogItem) NomenclatureQualityStats {
	stats := NomenclatureQualityStats{
		TotalRecords:       len(items),
		TopInconsistencies: []InconsistencyStat{},
		UnitOfMeasureStats: UnitOfMeasureStats{
			TopVariations:   []UnitVariation{},
			Inconsistencies: []string{},
		},
		AttributeCompleteness: AttributeCompleteness{
			TopBrands: []UnitVariation{},
		},
	}

	// Обработка пустого списка
	if len(items) == 0 {
		return stats
	}

	// Логирование для больших объемов данных
	if len(items) > 10000 {
		log.Printf("Analyzing large nomenclature dataset: %d items", len(items))
	}

	// Мапы для подсчета дубликатов и статистики
	articleMap := make(map[string][]string) // артикул -> список имен
	skuMap := make(map[string][]string)     // SKU -> список имен
	nameMap := make(map[string]int)         // нормализованное имя -> количество
	unitMap := make(map[string]int)         // единица измерения -> количество
	brandMap := make(map[string]int)        // бренд -> количество
	manufacturerMap := make(map[string]int) // производитель -> количество
	countryMap := make(map[string]int)      // страна -> количество

	// Статистика
	var withName int
	var withoutName int
	var withArticle int
	var withoutArticle int
	var withSKU int
	var withoutSKU int
	var withBrand int
	var withManufacturer int
	var withCountry int
	var nameLengths []int

	// Примеры для несоответствий
	var nameWithoutArticleExample string
	var articleWithoutNameExample string
	var inconsistentUnitExample string

	// Вариации единиц измерения для поиска неконсистентности
	unitVariations := map[string][]string{
		"штука":     {"шт", "штука", "штук", "шт."},
		"килограмм": {"кг", "килограмм", "килограммов", "кг."},
		"метр":      {"м", "метр", "метров", "м."},
		"литр":      {"л", "литр", "литров", "л."},
		"грамм":     {"г", "грамм", "граммов", "г."},
		"тонна":     {"т", "тонна", "тонн", "т."},
	}

	for _, item := range items {
		hasName := item.Name != "" && strings.TrimSpace(item.Name) != ""
		hasArticle := false
		hasSKU := false
		var article string
		var sku string
		var unit string
		var brand string
		var manufacturer string
		var country string

		// Извлекаем артикул из атрибутов
		article = extractArticleFromAttributes(item.Attributes)
		if article != "" {
			hasArticle = true
		}

		// Извлекаем SKU из атрибутов
		sku = extractSKUFromAttributes(item.Attributes)
		if sku != "" {
			hasSKU = true
		}

		// Извлекаем единицу измерения
		unit = extractUnitFromAttributes(item.Attributes)
		if unit != "" {
			normalizedUnit := normalizeUnit(unit)
			unitMap[normalizedUnit]++
		}

		// Извлекаем бренд, производителя, страну
		brand = extractBrandFromAttributes(item.Attributes)
		if brand != "" {
			withBrand++
			brandMap[brand]++
		}

		manufacturer = extractManufacturerFromAttributes(item.Attributes)
		if manufacturer != "" {
			withManufacturer++
			manufacturerMap[manufacturer]++
		}

		country = extractCountryFromAttributes(item.Attributes)
		if country != "" {
			withCountry++
			countryMap[country]++
		}

		// Подсчитываем статистику
		if hasName {
			withName++
			nameLengths = append(nameLengths, len(item.Name))
		} else {
			withoutName++
			if hasArticle && articleWithoutNameExample == "" {
				articleWithoutNameExample = fmt.Sprintf("Артикул: %s, название отсутствует", article)
			}
		}

		if hasArticle {
			withArticle++
		} else {
			withoutArticle++
			if hasName && nameWithoutArticleExample == "" {
				nameWithoutArticleExample = fmt.Sprintf("Название: %s, артикул отсутствует", item.Name)
			}
		}

		if hasSKU {
			withSKU++
		} else {
			withoutSKU++
		}

		// Группируем для поиска дубликатов
		if hasArticle && article != "" {
			articleMap[article] = append(articleMap[article], item.Name)
		}
		if hasSKU && sku != "" {
			skuMap[sku] = append(skuMap[sku], item.Name)
		}
		if hasName {
			normalizedName := strings.ToLower(strings.TrimSpace(item.Name))
			nameMap[normalizedName]++
		}
	}

	// Вычисляем статистику
	stats.RecordsWithName = withName
	stats.RecordsWithoutName = withoutName
	stats.RecordsWithArticle = withArticle
	stats.RecordsWithoutArticle = withoutArticle
	stats.RecordsWithSKU = withSKU
	stats.RecordsWithoutSKU = withoutSKU

	// Completeness: % записей с названием или артикулом/SKU
	recordsWithData := withName + withArticle + withSKU
	if stats.TotalRecords > 0 {
		stats.CompletenessScore = float64(recordsWithData) / float64(stats.TotalRecords) * 100
	}

	// Подсчитываем дубликаты (только записи, которые входят в группы дубликатов)
	duplicateSet := make(map[int]bool)

	for i, item := range items {
		hasDuplicate := false
		var article string
		var sku string

		article = extractArticleFromAttributes(item.Attributes)
		sku = extractSKUFromAttributes(item.Attributes)

		if article != "" && len(articleMap[article]) > 1 {
			hasDuplicate = true
		}
		if sku != "" && len(skuMap[sku]) > 1 {
			hasDuplicate = true
		}

		if item.Name != "" {
			normalizedName := strings.ToLower(strings.TrimSpace(item.Name))
			if nameMap[normalizedName] > 1 {
				hasDuplicate = true
			}
		}

		if hasDuplicate {
			duplicateSet[i] = true
		}
	}

	duplicateCount := len(duplicateSet)

	if stats.TotalRecords > 0 {
		stats.PotentialDupRate = float64(duplicateCount) / float64(stats.TotalRecords) * 100
	}

	// Статистика длины имен
	if len(nameLengths) > 0 {
		min := nameLengths[0]
		max := nameLengths[0]
		sum := 0
		for _, length := range nameLengths {
			if length < min {
				min = length
			}
			if length > max {
				max = length
			}
			sum += length
		}
		stats.NameLengthStats = NameLengthStats{
			Min:     min,
			Max:     max,
			Average: float64(sum) / float64(len(nameLengths)),
		}
	}

	// Статистика единиц измерения
	stats.UnitOfMeasureStats.UniqueCount = len(unitMap)

	// Топ-10 единиц измерения
	type unitCount struct {
		unit  string
		count int
	}
	units := make([]unitCount, 0, len(unitMap))
	for u, c := range unitMap {
		units = append(units, unitCount{unit: u, count: c})
	}
	sort.Slice(units, func(i, j int) bool {
		return units[i].count > units[j].count
	})
	for i, uc := range units {
		if i >= 10 {
			break
		}
		stats.UnitOfMeasureStats.TopVariations = append(stats.UnitOfMeasureStats.TopVariations, UnitVariation{
			Unit:  uc.unit,
			Count: uc.count,
		})
	}

	// Поиск неконсистентных вариаций единиц измерения
	for baseUnit, variations := range unitVariations {
		foundVariations := []string{}
		for _, variation := range variations {
			normalizedVariation := normalizeUnit(variation)
			if _, exists := unitMap[normalizedVariation]; exists {
				foundVariations = append(foundVariations, variation)
			}
		}
		if len(foundVariations) > 1 {
			stats.UnitOfMeasureStats.Inconsistencies = append(stats.UnitOfMeasureStats.Inconsistencies,
				fmt.Sprintf("Обнаружены вариации '%s': %v", baseUnit, foundVariations))
			if inconsistentUnitExample == "" {
				inconsistentUnitExample = fmt.Sprintf("Вариации единицы '%s': %v", baseUnit, foundVariations)
			}
		}
	}

	// Заполненность атрибутов
	if stats.TotalRecords > 0 {
		stats.AttributeCompleteness.RecordsWithBrand = withBrand
		stats.AttributeCompleteness.RecordsWithManufacturer = withManufacturer
		stats.AttributeCompleteness.RecordsWithCountry = withCountry
		stats.AttributeCompleteness.BrandPercent = float64(withBrand) / float64(stats.TotalRecords) * 100
		stats.AttributeCompleteness.ManufacturerPercent = float64(withManufacturer) / float64(stats.TotalRecords) * 100
		stats.AttributeCompleteness.CountryPercent = float64(withCountry) / float64(stats.TotalRecords) * 100
	}

	// Формируем топ-10 брендов для этой БД
	if len(brandMap) > 0 {
		type brandCount struct {
			brand string
			count int
		}
		brands := make([]brandCount, 0, len(brandMap))
		for b, c := range brandMap {
			brands = append(brands, brandCount{brand: b, count: c})
		}
		sort.Slice(brands, func(i, j int) bool {
			return brands[i].count > brands[j].count
		})
		for i, bc := range brands {
			if i >= 10 {
				break
			}
			stats.AttributeCompleteness.TopBrands = append(
				stats.AttributeCompleteness.TopBrands,
				UnitVariation{Unit: bc.brand, Count: bc.count},
			)
		}
	}

	// Формируем список несоответствий
	if withoutArticle > 0 && nameWithoutArticleExample != "" {
		stats.TopInconsistencies = append(stats.TopInconsistencies, InconsistencyStat{
			Type:        "name_without_article",
			Count:       withoutArticle,
			Example:     nameWithoutArticleExample,
			Description: "Записи с названием, но без артикула",
		})
	}
	if withoutName > 0 && articleWithoutNameExample != "" {
		stats.TopInconsistencies = append(stats.TopInconsistencies, InconsistencyStat{
			Type:        "article_without_name",
			Count:       withoutName,
			Example:     articleWithoutNameExample,
			Description: "Записи с артикулом, но без названия",
		})
	}
	if len(stats.UnitOfMeasureStats.Inconsistencies) > 0 && inconsistentUnitExample != "" {
		stats.TopInconsistencies = append(stats.TopInconsistencies, InconsistencyStat{
			Type:        "inconsistent_units",
			Count:       len(stats.UnitOfMeasureStats.Inconsistencies),
			Example:     inconsistentUnitExample,
			Description: "Неконсистентные единицы измерения",
		})
	}

	return stats
}

// extractArticleFromAttributes извлекает артикул из XML атрибутов
func extractArticleFromAttributes(attributesXML string) string {
	if attributesXML == "" {
		return ""
	}

	// Сначала пробуем найти артикул в тексте
	re := regexp.MustCompile(`(?i)(?:артикул|article|артикулноменклатуры)[\s:]*([^\s<,;]+)`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value
		}
	}

	// Пробуем парсить как XML с разными вариантами названий полей
	possibleFields := []string{"Артикул", "АртикулНоменклатуры", "Article", "ArticleNumber", "АртикулТовара", "АртикулИзделия"}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			// Пробуем найти поле в XML (как тег или как атрибут)
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}

			// Пробуем найти как атрибут Name="Артикул" Value="..."
			re = regexp.MustCompile(fmt.Sprintf(`(?i)Name="%s"[^>]*Value="([^"]+)"`, field))
			matches = re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// extractSKUFromAttributes извлекает SKU из XML атрибутов
func extractSKUFromAttributes(attributesXML string) string {
	if attributesXML == "" {
		return ""
	}

	// Сначала пробуем найти SKU в тексте
	re := regexp.MustCompile(`(?i)(?:sku|код|кодтовара)[\s:]*([^\s<,;]+)`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value
		}
	}

	// Пробуем парсить как XML
	possibleFields := []string{"SKU", "Код", "КодТовара", "КодНоменклатуры", "Code", "ProductCode"}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}

			re = regexp.MustCompile(fmt.Sprintf(`(?i)Name="%s"[^>]*Value="([^"]+)"`, field))
			matches = re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// extractUnitFromAttributes извлекает единицу измерения из XML атрибутов
func extractUnitFromAttributes(attributesXML string) string {
	if attributesXML == "" {
		return ""
	}

	// Сначала пробуем найти единицу измерения в тексте
	re := regexp.MustCompile(`(?i)(?:единицаизмерения|единица|unit|еи)[\s:]*([^\s<,;]+)`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value
		}
	}

	// Пробуем парсить как XML
	possibleFields := []string{"ЕдиницаИзмерения", "Единица", "Unit", "ЕИ", "ЕдиницаХранения", "BaseUnit"}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}

			re = regexp.MustCompile(fmt.Sprintf(`(?i)Name="%s"[^>]*Value="([^"]+)"`, field))
			matches = re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// extractBrandFromAttributes извлекает бренд из XML атрибутов
func extractBrandFromAttributes(attributesXML string) string {
	if attributesXML == "" {
		return ""
	}

	// Сначала пробуем найти бренд в тексте
	re := regexp.MustCompile(`(?i)(?:бренд|brand|торговаямарка)[\s:]*([^\s<,;]+)`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value
		}
	}

	// Пробуем парсить как XML
	possibleFields := []string{"Бренд", "Brand", "ТорговаяМарка", "ТМ", "TradeMark"}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}

			re = regexp.MustCompile(fmt.Sprintf(`(?i)Name="%s"[^>]*Value="([^"]+)"`, field))
			matches = re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// extractManufacturerFromAttributes извлекает производителя из XML атрибутов
func extractManufacturerFromAttributes(attributesXML string) string {
	if attributesXML == "" {
		return ""
	}

	// Сначала пробуем найти производителя в тексте
	re := regexp.MustCompile(`(?i)(?:производитель|manufacturer|изготовитель)[\s:]*([^\s<,;]+)`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value
		}
	}

	// Пробуем парсить как XML
	possibleFields := []string{"Производитель", "Manufacturer", "Изготовитель", "ПроизводительТовара"}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}

			re = regexp.MustCompile(fmt.Sprintf(`(?i)Name="%s"[^>]*Value="([^"]+)"`, field))
			matches = re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// extractCountryFromAttributes извлекает страну происхождения из XML атрибутов
func extractCountryFromAttributes(attributesXML string) string {
	if attributesXML == "" {
		return ""
	}

	// Сначала пробуем найти страну в тексте
	re := regexp.MustCompile(`(?i)(?:странапроисхождения|страна|country)[\s:]*([^\s<,;]+)`)
	matches := re.FindStringSubmatch(attributesXML)
	if len(matches) > 1 {
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value
		}
	}

	// Пробуем парсить как XML
	possibleFields := []string{"СтранаПроисхождения", "Страна", "Country", "СтранаИзготовителя"}

	if strings.Contains(attributesXML, "<") || strings.Contains(attributesXML, ">") {
		for _, field := range possibleFields {
			re := regexp.MustCompile(fmt.Sprintf(`(?i)<%s[^>]*>([^<]+)</%s>`, field, field))
			matches := re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}

			re = regexp.MustCompile(fmt.Sprintf(`(?i)Name="%s"[^>]*Value="([^"]+)"`, field))
			matches = re.FindStringSubmatch(attributesXML)
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// normalizeUnit нормализует единицу измерения (приводит к нижнему регистру, убирает точки)
func normalizeUnit(unit string) string {
	normalized := strings.ToLower(strings.TrimSpace(unit))
	normalized = strings.TrimSuffix(normalized, ".")
	return normalized
}

// calculateDataQualityScore вычисляет общую оценку качества данных
func (s *Server) calculateDataQualityScore(report *DataQualityReport) OverallQualityScore {
	score := OverallQualityScore{}

	// Используем статистику контрагентов и номенклатуры для расчета
	cpStats := report.CounterpartyStats
	nomStats := report.NomenclatureStats

	// Усредняем показатели или берем худшие
	var completenessSum, completenessCount float64
	var uniquenessSum, uniquenessCount float64
	var inconsistencyCount int

	if cpStats.TotalRecords > 0 {
		completenessSum += cpStats.CompletenessScore
		completenessCount++
		uniquenessSum += (100 - cpStats.PotentialDupRate)
		uniquenessCount++
		inconsistencyCount += len(cpStats.TopInconsistencies)
	}

	if nomStats.TotalRecords > 0 {
		completenessSum += nomStats.CompletenessScore
		completenessCount++
		uniquenessSum += (100 - nomStats.PotentialDupRate)
		uniquenessCount++
		inconsistencyCount += len(nomStats.TopInconsistencies)
	}

	if completenessCount > 0 {
		score.Completeness = completenessSum / completenessCount
	}
	if uniquenessCount > 0 {
		score.Uniqueness = uniquenessSum / uniquenessCount
	}
	score.Consistency = 100 - float64(inconsistencyCount)*10 // Чем меньше несоответствий, тем выше

	// Общая оценка - среднее арифметическое
	if completenessCount > 0 || uniquenessCount > 0 {
		score.Score = (score.Completeness + score.Uniqueness + score.Consistency) / 3
	}

	// Определяем качество данных
	if score.Score >= 90 {
		score.DataQuality = "excellent"
	} else if score.Score >= 75 {
		score.DataQuality = "good"
	} else if score.Score >= 60 {
		score.DataQuality = "fair"
	} else {
		score.DataQuality = "poor"
	}

	return score
}

// generateDataQualityRecommendations генерирует рекомендации на основе анализа качества данных
func (s *Server) generateDataQualityRecommendations(report *DataQualityReport) []string {
	recommendations := []string{}

	// Анализ полноты данных контрагентов
	if report.CounterpartyStats.TotalRecords > 0 && report.CounterpartyStats.CompletenessScore < 80 {
		recommendations = append(recommendations,
			fmt.Sprintf("Низкая полнота данных контрагентов: %.1f%% записей имеют имя или ИНН/БИН. Рекомендуется проверить источники данных.",
				report.CounterpartyStats.CompletenessScore))
	}

	// Анализ дубликатов контрагентов
	if report.CounterpartyStats.TotalRecords > 0 && report.CounterpartyStats.PotentialDupRate > 20 {
		recommendations = append(recommendations,
			fmt.Sprintf("Высокий процент потенциальных дубликатов контрагентов: %.1f%%. Нормализация поможет объединить дубликаты.",
				report.CounterpartyStats.PotentialDupRate))
	}

	// Анализ несоответствий контрагентов
	if len(report.CounterpartyStats.TopInconsistencies) > 0 {
		for _, inconsistency := range report.CounterpartyStats.TopInconsistencies {
			if inconsistency.Count > 10 {
				recommendations = append(recommendations,
					fmt.Sprintf("Обнаружено %d случаев: %s. Пример: %s",
						inconsistency.Count, inconsistency.Description, inconsistency.Example))
			}
		}
	}

	// Анализ полноты данных номенклатуры
	if report.NomenclatureStats.TotalRecords > 0 && report.NomenclatureStats.CompletenessScore < 80 {
		recommendations = append(recommendations,
			fmt.Sprintf("Низкая полнота данных номенклатуры: %.1f%% записей имеют название или артикул/SKU. Рекомендуется проверить источники данных.",
				report.NomenclatureStats.CompletenessScore))
	}

	// Анализ дубликатов номенклатуры
	if report.NomenclatureStats.TotalRecords > 0 && report.NomenclatureStats.PotentialDupRate > 20 {
		recommendations = append(recommendations,
			fmt.Sprintf("Высокий процент потенциальных дубликатов номенклатуры: %.1f%%. Нормализация поможет объединить дубликаты.",
				report.NomenclatureStats.PotentialDupRate))
	}

	// Анализ несоответствий номенклатуры
	if len(report.NomenclatureStats.TopInconsistencies) > 0 {
		for _, inconsistency := range report.NomenclatureStats.TopInconsistencies {
			if inconsistency.Count > 10 {
				recommendations = append(recommendations,
					fmt.Sprintf("Обнаружено %d случаев: %s. Пример: %s",
						inconsistency.Count, inconsistency.Description, inconsistency.Example))
			}
		}
	}

	// Рекомендации по единицам измерения
	if len(report.NomenclatureStats.UnitOfMeasureStats.Inconsistencies) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Обнаружено %d типов неконсистентных единиц измерения. Рекомендуется нормализовать вариации (например, 'шт', 'штука', 'штук').",
				len(report.NomenclatureStats.UnitOfMeasureStats.Inconsistencies)))
	}

	// Рекомендации по заполненности атрибутов номенклатуры
	if report.NomenclatureStats.TotalRecords > 0 {
		if report.NomenclatureStats.AttributeCompleteness.BrandPercent < 50 {
			recommendations = append(recommendations,
				fmt.Sprintf("Низкая заполненность брендов: %.1f%% записей имеют бренд. Рекомендуется дополнить данные.",
					report.NomenclatureStats.AttributeCompleteness.BrandPercent))
		}
		if report.NomenclatureStats.AttributeCompleteness.ManufacturerPercent < 50 {
			recommendations = append(recommendations,
				fmt.Sprintf("Низкая заполненность производителей: %.1f%% записей имеют производителя. Рекомендуется дополнить данные.",
					report.NomenclatureStats.AttributeCompleteness.ManufacturerPercent))
		}
		if report.NomenclatureStats.RecordsWithoutArticle > 0 {
			articlePercent := float64(report.NomenclatureStats.RecordsWithoutArticle) / float64(report.NomenclatureStats.TotalRecords) * 100
			if articlePercent > 30 {
				recommendations = append(recommendations,
					fmt.Sprintf("%.1f%% товаров не имеют артикулов, что затрудняет их точную идентификацию. Рекомендуется добавить артикулы.",
						articlePercent))
			}
		}
	}

	// Анализ по БД
	if len(report.DatabaseBreakdown) > 0 {
		var maxDupDB *DatabaseQualityStats
		var minCompletenessDB *DatabaseQualityStats

		for i := range report.DatabaseBreakdown {
			db := &report.DatabaseBreakdown[i]
			if db.Status != "success" {
				continue
			}

			if maxDupDB == nil || db.PotentialDupRate > maxDupDB.PotentialDupRate {
				maxDupDB = db
			}
			if minCompletenessDB == nil || db.CompletenessScore < minCompletenessDB.CompletenessScore {
				minCompletenessDB = db
			}
		}

		if maxDupDB != nil && maxDupDB.PotentialDupRate > 15 {
			recommendations = append(recommendations,
				fmt.Sprintf("База данных '%s' содержит наибольшее количество потенциальных дубликатов (%.1f%%). Рекомендуется приоритетная обработка.",
					maxDupDB.DatabaseName, maxDupDB.PotentialDupRate))
		}

		if minCompletenessDB != nil && minCompletenessDB.CompletenessScore < 70 {
			recommendations = append(recommendations,
				fmt.Sprintf("База данных '%s' имеет низкую полноту данных (%.1f%%). Рекомендуется проверить качество исходных данных.",
					minCompletenessDB.DatabaseName, minCompletenessDB.CompletenessScore))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Качество данных в целом хорошее. Можно приступать к нормализации.")
	}

	return recommendations
}

// analyzeDatabase анализирует качество данных в одной БД
func (s *Server) analyzeDatabase(projectDB *database.ProjectDatabase, dbStats *DatabaseQualityStats) (*CounterpartyQualityStats, *NomenclatureQualityStats, error) {
	// Открываем БД
	sourceDB, err := database.NewDB(projectDB.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sourceDB.Close()

	// Получаем все выгрузки
	uploads, err := sourceDB.GetAllUploads()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get uploads: %w", err)
	}

	// Собираем контрагентов
	var counterparties []*database.CatalogItem
	for _, upload := range uploads {
		items, _, err := sourceDB.GetCatalogItemsByUpload(upload.ID, []string{"Контрагенты"}, 0, 0)
		if err != nil {
			log.Printf("Failed to get counterparties from upload %d: %v", upload.ID, err)
			continue
		}
		counterparties = append(counterparties, items...)
	}

	dbStats.Counterparties = len(counterparties)

	// Собираем номенклатуру
	var nomenclature []*database.CatalogItem
	for _, upload := range uploads {
		items, _, err := sourceDB.GetCatalogItemsByUpload(upload.ID, []string{"Номенклатура"}, 0, 0)
		if err != nil {
			log.Printf("Failed to get nomenclature from upload %d: %v", upload.ID, err)
			continue
		}
		nomenclature = append(nomenclature, items...)
	}

	dbStats.Nomenclature = len(nomenclature)

	// Анализируем контрагентов
	var cpStats *CounterpartyQualityStats
	if len(counterparties) > 0 {
		stats := s.analyzeCounterparties(counterparties)
		cpStats = &stats
		dbStats.CompletenessScore = stats.CompletenessScore
		dbStats.PotentialDupRate = stats.PotentialDupRate
		dbStats.InconsistenciesCount = len(stats.TopInconsistencies)
	} else {
		cpStats = &CounterpartyQualityStats{}
	}

	// Анализируем номенклатуру
	var nomStats *NomenclatureQualityStats
	if len(nomenclature) > 0 {
		stats := s.analyzeNomenclature(nomenclature)
		nomStats = &stats
		// Обновляем dbStats с учетом номенклатуры (усредняем показатели)
		if dbStats.CompletenessScore == 0 {
			dbStats.CompletenessScore = nomStats.CompletenessScore
		} else {
			dbStats.CompletenessScore = (dbStats.CompletenessScore + nomStats.CompletenessScore) / 2
		}
		if dbStats.PotentialDupRate == 0 {
			dbStats.PotentialDupRate = nomStats.PotentialDupRate
		} else {
			dbStats.PotentialDupRate = (dbStats.PotentialDupRate + nomStats.PotentialDupRate) / 2
		}
		dbStats.InconsistenciesCount += len(nomStats.TopInconsistencies)
	} else {
		nomStats = &NomenclatureQualityStats{}
	}

	return cpStats, nomStats, nil
}
