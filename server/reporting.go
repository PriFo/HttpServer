package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ProviderMetricsReport метрики для одного провайдера (для отчета)
type ProviderMetricsReport struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	ActiveChannels     int     `json:"active_channels"`
	CurrentRequests    int     `json:"current_requests"`
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	AverageLatencyMs   float64 `json:"average_latency_ms"`
	LastRequestTime    string  `json:"last_request_time"`
	Status             string  `json:"status"`
	RequestsPerSecond  float64 `json:"requests_per_second"`
}

// NormalizationReport структура для комплексного отчета по нормализации
type NormalizationReport struct {
	ReportMetadata       ReportMetadata          `json:"metadata"`
	OverallStats         OverallStats            `json:"overall_stats"`
	CounterpartyAnalysis DataAnalysis            `json:"counterparty_analysis"`
	NomenclatureAnalysis DataAnalysis            `json:"nomenclature_analysis"`
	ProviderPerformance  []ProviderMetricsReport `json:"provider_performance"`
	DatabaseBreakdown    []DatabaseStats         `json:"database_breakdown"`
	Recommendations      []string                `json:"recommendations"`
}

// ReportMetadata метаданные отчета
type ReportMetadata struct {
	GeneratedAt    time.Time `json:"generated_at"`
	ReportVersion  string    `json:"report_version"`
	TotalDatabases int       `json:"total_databases"`
	TotalProjects  int       `json:"total_projects"`
}

// OverallStats общая статистика проекта
type OverallStats struct {
	TotalDatabasesProcessed int     `json:"total_databases_processed"`
	TotalCounterparties     int     `json:"total_counterparties"`
	TotalNomenclature       int     `json:"total_nomenclature"`
	TotalDuplicateGroups    int     `json:"total_duplicate_groups"`
	TotalErrors             int     `json:"total_errors"`
	AverageQualityScore     float64 `json:"average_quality_score"`
}

// DataAnalysis детальный анализ данных
type DataAnalysis struct {
	TotalRecordsBefore   int             `json:"total_records_before"`
	TotalRecordsAfter    int             `json:"total_records_after"`
	ReductionPercentage  float64         `json:"reduction_percentage"`
	DuplicateGroupsFound int             `json:"duplicate_groups_found"`
	TopNormalizedNames   []NameFrequency `json:"top_normalized_names"`
	ValidationErrors     int             `json:"validation_errors"`
	NormalizationErrors  int             `json:"normalization_errors"`
	AverageQualityScore  float64         `json:"average_quality_score"`
	EnrichmentStats      EnrichmentStats `json:"enrichment_stats"`
}

// NameFrequency частота использования имени
type NameFrequency struct {
	Name       string  `json:"name"`
	Frequency  int     `json:"frequency"`
	Percentage float64 `json:"percentage"`
}

// EnrichmentStats статистика обогащения
type EnrichmentStats struct {
	TotalEnriched      int     `json:"total_enriched"`
	EnrichmentRate     float64 `json:"enrichment_rate"`
	BenchmarkMatches   int     `json:"benchmark_matches"`
	ExternalEnrichment int     `json:"external_enrichment"`
}

// DatabaseStats статистика по базе данных
type DatabaseStats struct {
	DatabaseID      int        `json:"database_id"`
	DatabaseName    string     `json:"database_name"`
	FilePath        string     `json:"file_path"`
	Counterparties  int        `json:"counterparties"`
	Nomenclature    int        `json:"nomenclature"`
	DuplicateGroups int        `json:"duplicate_groups"`
	Errors          int        `json:"errors"`
	QualityScore    float64    `json:"quality_score"`
	LastProcessed   *time.Time `json:"last_processed,omitempty"`
}

// GenerateNormalizationReport генерирует комплексный отчет по нормализации
func (s *Server) GenerateNormalizationReport() (*NormalizationReport, error) {
	report := &NormalizationReport{
		ReportMetadata: ReportMetadata{
			GeneratedAt:   time.Now(),
			ReportVersion: "1.0",
		},
		OverallStats:         OverallStats{},
		CounterpartyAnalysis: DataAnalysis{},
		NomenclatureAnalysis: DataAnalysis{},
		DatabaseBreakdown:    []DatabaseStats{},
		Recommendations:      []string{},
	}

	// Получаем все проекты через прямой SQL запрос
	serviceDB := s.serviceDB.GetDB()
	var projectCount int
	err := serviceDB.QueryRow(`SELECT COUNT(*) FROM client_projects WHERE status = 'active'`).Scan(&projectCount)
	if err == nil {
		report.ReportMetadata.TotalProjects = projectCount
	}

	// Собираем данные по контрагентам
	if err := s.collectCounterpartyData(report); err != nil {
		log.Printf("Error collecting counterparty data: %v", err)
		// Продолжаем, даже если есть ошибки
	}

	// Собираем данные по номенклатуре
	if err := s.collectNomenclatureData(report); err != nil {
		log.Printf("Error collecting nomenclature data: %v", err)
		// Продолжаем, даже если есть ошибки
	}

	// Собираем данные по базам данных
	if err := s.collectDatabaseData(report); err != nil {
		log.Printf("Error collecting database data: %v", err)
		// Продолжаем, даже если есть ошибки
	}

	// Получаем метрики провайдеров
	if s.monitoringManager != nil {
		monitoringData := s.monitoringManager.GetAllMetrics()
		// Преобразуем monitoring.ProviderMetrics в handlers.ProviderMetrics
		report.ProviderPerformance = make([]ProviderMetricsReport, len(monitoringData.Providers))
		for i, pm := range monitoringData.Providers {
			lastRequestTime := ""
			if !pm.LastRequestTime.IsZero() {
				lastRequestTime = pm.LastRequestTime.Format(time.RFC3339)
			}
			report.ProviderPerformance[i] = ProviderMetricsReport{
				ID:                 pm.ID,
				Name:               pm.Name,
				ActiveChannels:     pm.ActiveChannels,
				CurrentRequests:    pm.CurrentRequests,
				TotalRequests:      pm.TotalRequests,
				SuccessfulRequests: pm.SuccessfulRequests,
				FailedRequests:     pm.FailedRequests,
				AverageLatencyMs:   pm.AverageLatencyMs,
				LastRequestTime:    lastRequestTime,
				Status:             pm.Status,
				RequestsPerSecond:  pm.RequestsPerSecond,
			}
		}
	}

	// Генерируем рекомендации
	report.generateRecommendations()

	return report, nil
}

// collectCounterpartyData собирает данные по контрагентам
func (s *Server) collectCounterpartyData(report *NormalizationReport) error {
	serviceDB := s.serviceDB.GetDB()

	// Общая статистика по контрагентам
	var totalCounterparties int
	err := serviceDB.QueryRow(`
		SELECT COUNT(*) 
		FROM normalized_counterparties
	`).Scan(&totalCounterparties)
	if err != nil {
		return fmt.Errorf("failed to count counterparties: %w", err)
	}

	report.CounterpartyAnalysis.TotalRecordsAfter = totalCounterparties

	// Подсчет групп дубликатов (по normalized_name)
	var duplicateGroups int
	err = serviceDB.QueryRow(`
		SELECT COUNT(DISTINCT normalized_name)
		FROM normalized_counterparties
		WHERE normalized_name IS NOT NULL AND normalized_name != ''
	`).Scan(&duplicateGroups)
	if err != nil {
		log.Printf("Error counting duplicate groups: %v", err)
	} else {
		report.CounterpartyAnalysis.DuplicateGroupsFound = duplicateGroups
		report.OverallStats.TotalDuplicateGroups += duplicateGroups
	}

	// Топ-10 нормализованных имен
	rows, err := serviceDB.Query(`
		SELECT normalized_name, COUNT(*) as freq
		FROM normalized_counterparties
		WHERE normalized_name IS NOT NULL AND normalized_name != ''
		GROUP BY normalized_name
		ORDER BY freq DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		totalFreq := 0
		topNames := []NameFrequency{}

		for rows.Next() {
			var name string
			var freq int
			if err := rows.Scan(&name, &freq); err == nil {
				topNames = append(topNames, NameFrequency{
					Name:      name,
					Frequency: freq,
				})
				totalFreq += freq
			}
		}

		// Вычисляем проценты
		if totalFreq > 0 {
			for i := range topNames {
				topNames[i].Percentage = float64(topNames[i].Frequency) / float64(totalCounterparties) * 100
			}
		}

		report.CounterpartyAnalysis.TopNormalizedNames = topNames
	}

	// Средний quality_score
	var avgQuality sql.NullFloat64
	err = serviceDB.QueryRow(`
		SELECT AVG(quality_score)
		FROM normalized_counterparties
		WHERE quality_score > 0
	`).Scan(&avgQuality)
	if err == nil && avgQuality.Valid {
		report.CounterpartyAnalysis.AverageQualityScore = avgQuality.Float64
		report.OverallStats.AverageQualityScore = avgQuality.Float64
	}

	// Статистика обогащения
	var enrichedCount int
	err = serviceDB.QueryRow(`
		SELECT COUNT(*)
		FROM normalized_counterparties
		WHERE enrichment_applied = 1
	`).Scan(&enrichedCount)
	if err == nil {
		report.CounterpartyAnalysis.EnrichmentStats.TotalEnriched = enrichedCount
		if totalCounterparties > 0 {
			report.CounterpartyAnalysis.EnrichmentStats.EnrichmentRate = float64(enrichedCount) / float64(totalCounterparties) * 100
		}
	}

	// Benchmark matches
	var benchmarkMatches int
	err = serviceDB.QueryRow(`
		SELECT COUNT(*)
		FROM normalized_counterparties
		WHERE benchmark_id IS NOT NULL
	`).Scan(&benchmarkMatches)
	if err == nil {
		report.CounterpartyAnalysis.EnrichmentStats.BenchmarkMatches = benchmarkMatches
	}

	// External enrichment (Adata, Dadata, gisp)
	var externalEnrichment int
	err = serviceDB.QueryRow(`
		SELECT COUNT(*)
		FROM normalized_counterparties
		WHERE source_enrichment IS NOT NULL AND source_enrichment != ''
	`).Scan(&externalEnrichment)
	if err == nil {
		report.CounterpartyAnalysis.EnrichmentStats.ExternalEnrichment = externalEnrichment
	}

	report.OverallStats.TotalCounterparties = totalCounterparties
	report.CounterpartyAnalysis.TotalRecordsBefore = totalCounterparties // Предполагаем, что до нормализации было столько же или больше

	return nil
}

// collectNomenclatureData собирает данные по номенклатуре
func (s *Server) collectNomenclatureData(report *NormalizationReport) error {
	if s.normalizedDB == nil {
		return fmt.Errorf("normalized database not available")
	}

	db := s.normalizedDB.GetDB()

	// Общая статистика по номенклатуре
	var totalNomenclature int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM normalized_data
	`).Scan(&totalNomenclature)
	if err != nil {
		return fmt.Errorf("failed to count nomenclature: %w", err)
	}

	report.NomenclatureAnalysis.TotalRecordsAfter = totalNomenclature

	// Подсчет групп дубликатов (по normalized_name)
	var duplicateGroups int
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT normalized_name)
		FROM normalized_data
		WHERE normalized_name IS NOT NULL AND normalized_name != ''
	`).Scan(&duplicateGroups)
	if err == nil {
		report.NomenclatureAnalysis.DuplicateGroupsFound = duplicateGroups
		report.OverallStats.TotalDuplicateGroups += duplicateGroups
	}

	// Топ-10 нормализованных имен
	rows, err := db.Query(`
		SELECT normalized_name, COUNT(*) as freq
		FROM normalized_data
		WHERE normalized_name IS NOT NULL AND normalized_name != ''
		GROUP BY normalized_name
		ORDER BY freq DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		totalFreq := 0
		topNames := []NameFrequency{}

		for rows.Next() {
			var name string
			var freq int
			if err := rows.Scan(&name, &freq); err == nil {
				topNames = append(topNames, NameFrequency{
					Name:      name,
					Frequency: freq,
				})
				totalFreq += freq
			}
		}

		// Вычисляем проценты
		if totalFreq > 0 {
			for i := range topNames {
				topNames[i].Percentage = float64(topNames[i].Frequency) / float64(totalNomenclature) * 100
			}
		}

		report.NomenclatureAnalysis.TopNormalizedNames = topNames
	}

	// Средний quality_score (если есть)
	var avgQuality sql.NullFloat64
	err = db.QueryRow(`
		SELECT AVG(ai_confidence)
		FROM normalized_data
		WHERE ai_confidence > 0
	`).Scan(&avgQuality)
	if err == nil && avgQuality.Valid {
		report.NomenclatureAnalysis.AverageQualityScore = avgQuality.Float64
	}

	// Статистика по merged_count (объединенные записи)
	var totalMerged int
	err = db.QueryRow(`
		SELECT SUM(merged_count)
		FROM normalized_data
		WHERE merged_count > 1
	`).Scan(&totalMerged)
	if err == nil {
		report.NomenclatureAnalysis.TotalRecordsBefore = totalMerged + (totalNomenclature - totalMerged)
		if report.NomenclatureAnalysis.TotalRecordsBefore > 0 {
			report.NomenclatureAnalysis.ReductionPercentage =
				float64(report.NomenclatureAnalysis.TotalRecordsBefore-totalNomenclature) /
					float64(report.NomenclatureAnalysis.TotalRecordsBefore) * 100
		}
	} else {
		report.NomenclatureAnalysis.TotalRecordsBefore = totalNomenclature
	}

	report.OverallStats.TotalNomenclature = totalNomenclature

	return nil
}

// collectDatabaseData собирает данные по базам данных
func (s *Server) collectDatabaseData(report *NormalizationReport) error {
	serviceDB := s.serviceDB.GetDB()
	databaseStats := []DatabaseStats{}

	// Получаем все активные проекты
	rows, err := serviceDB.Query(`
		SELECT id, name
		FROM client_projects
		WHERE status = 'active'
	`)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}
	defer rows.Close()

	var projects []struct {
		ID   int
		Name string
	}
	for rows.Next() {
		var project struct {
			ID   int
			Name string
		}
		if err := rows.Scan(&project.ID, &project.Name); err == nil {
			projects = append(projects, project)
		}
	}

	for _, project := range projects {
		// Получаем базы данных проекта
		projectDBs, err := s.serviceDB.GetProjectDatabases(project.ID, false)
		if err != nil {
			log.Printf("Error getting databases for project %d: %v", project.ID, err)
			continue
		}

		for _, projectDB := range projectDBs {
			stats := DatabaseStats{
				DatabaseID:   projectDB.ID,
				DatabaseName: projectDB.Name,
				FilePath:     projectDB.FilePath,
			}

			// Подсчет контрагентов по source_database
			var counterparties int
			err = serviceDB.QueryRow(`
				SELECT COUNT(*)
				FROM normalized_counterparties
				WHERE source_database = ? OR source_database LIKE ?
			`, projectDB.Name, "%"+filepath.Base(projectDB.FilePath)+"%").Scan(&counterparties)
			if err == nil {
				stats.Counterparties = counterparties
			}

			// Подсчет групп дубликатов для этой БД
			var duplicateGroups int
			err = serviceDB.QueryRow(`
				SELECT COUNT(DISTINCT normalized_name)
				FROM normalized_counterparties
				WHERE (source_database = ? OR source_database LIKE ?)
				AND normalized_name IS NOT NULL AND normalized_name != ''
			`, projectDB.Name, "%"+filepath.Base(projectDB.FilePath)+"%").Scan(&duplicateGroups)
			if err == nil {
				stats.DuplicateGroups = duplicateGroups
			}

			// Средний quality_score для этой БД
			var avgQuality sql.NullFloat64
			err = serviceDB.QueryRow(`
				SELECT AVG(quality_score)
				FROM normalized_counterparties
				WHERE (source_database = ? OR source_database LIKE ?)
				AND quality_score > 0
			`, projectDB.Name, "%"+filepath.Base(projectDB.FilePath)+"%").Scan(&avgQuality)
			if err == nil && avgQuality.Valid {
				stats.QualityScore = avgQuality.Float64
			}

			// Последнее время обработки
			if projectDB.LastUsedAt != nil {
				stats.LastProcessed = projectDB.LastUsedAt
			}

			databaseStats = append(databaseStats, stats)
		}
	}

	report.DatabaseBreakdown = databaseStats
	report.ReportMetadata.TotalDatabases = len(databaseStats)
	report.OverallStats.TotalDatabasesProcessed = len(databaseStats)

	return nil
}

// generateRecommendations генерирует рекомендации на основе собранных данных
func (r *NormalizationReport) generateRecommendations() {
	r.Recommendations = []string{}

	// Анализ провайдеров
	if len(r.ProviderPerformance) > 0 {
		var slowestProvider *ProviderMetricsReport
		var fastestProvider *ProviderMetricsReport
		var mostErrors *ProviderMetricsReport

		for i := range r.ProviderPerformance {
			provider := &r.ProviderPerformance[i]
			if slowestProvider == nil || provider.AverageLatencyMs > slowestProvider.AverageLatencyMs {
				slowestProvider = provider
			}
			if fastestProvider == nil || provider.AverageLatencyMs < fastestProvider.AverageLatencyMs {
				fastestProvider = provider
			}
			if mostErrors == nil || provider.FailedRequests > mostErrors.FailedRequests {
				mostErrors = provider
			}
		}

		if slowestProvider != nil && slowestProvider.AverageLatencyMs > 2000 {
			r.Recommendations = append(r.Recommendations,
				fmt.Sprintf("Провайдер %s показывает высокую задержку (%.0f мс). Рекомендуется проверить подключение или рассмотреть альтернативы.",
					slowestProvider.Name, slowestProvider.AverageLatencyMs))
		}

		if mostErrors != nil && mostErrors.FailedRequests > 0 {
			errorRate := float64(mostErrors.FailedRequests) / float64(mostErrors.TotalRequests) * 100
			if errorRate > 10 {
				r.Recommendations = append(r.Recommendations,
					fmt.Sprintf("Провайдер %s имеет высокий процент ошибок (%.1f%%). Рекомендуется проверить API ключи и лимиты.",
						mostErrors.Name, errorRate))
			}
		}
	}

	// Анализ качества данных
	if r.CounterpartyAnalysis.AverageQualityScore < 0.7 {
		r.Recommendations = append(r.Recommendations,
			"Средний показатель качества контрагентов ниже 0.7. Рекомендуется провести дополнительную нормализацию и обогащение данных.")
	}

	// Анализ дубликатов
	if r.OverallStats.TotalDuplicateGroups > 0 {
		duplicateRate := float64(r.OverallStats.TotalDuplicateGroups) / float64(r.OverallStats.TotalCounterparties) * 100
		if duplicateRate > 50 {
			r.Recommendations = append(r.Recommendations,
				"Обнаружено большое количество групп дубликатов. Рекомендуется провести ручную проверку и объединение записей.")
		}
	}

	// Анализ обогащения
	if r.CounterpartyAnalysis.EnrichmentStats.EnrichmentRate < 30 {
		r.Recommendations = append(r.Recommendations,
			"Низкий процент обогащения данных. Рекомендуется проверить настройки обогащения и доступность внешних сервисов.")
	}

	// Анализ баз данных
	if len(r.DatabaseBreakdown) > 0 {
		var maxDuplicatesDB *DatabaseStats
		for i := range r.DatabaseBreakdown {
			db := &r.DatabaseBreakdown[i]
			if maxDuplicatesDB == nil || db.DuplicateGroups > maxDuplicatesDB.DuplicateGroups {
				maxDuplicatesDB = db
			}
		}

		if maxDuplicatesDB != nil && maxDuplicatesDB.DuplicateGroups > 100 {
			r.Recommendations = append(r.Recommendations,
				fmt.Sprintf("База данных '%s' содержит наибольшее количество дубликатов (%d). Рекомендуется приоритетная обработка.",
					maxDuplicatesDB.DatabaseName, maxDuplicatesDB.DuplicateGroups))
		}
	}

	if len(r.Recommendations) == 0 {
		r.Recommendations = append(r.Recommendations,
			"Система работает в штатном режиме. Все показатели в норме.")
	}
}

// handleGenerateNormalizationReport обработчик для генерации отчета
func (s *Server) handleGenerateNormalizationReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Генерируем отчет
	report, err := s.GenerateNormalizationReport()
	if err != nil {
		log.Printf("Error generating normalization report: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	// Отправляем JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(report); err != nil {
		log.Printf("Error encoding report: %v", err)
		http.Error(w, "Failed to encode report", http.StatusInternalServerError)
		return
	}
}
