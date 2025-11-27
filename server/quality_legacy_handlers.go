package server

import (
	"fmt"
	"net/http"
	"strings"

	"httpserver/database"
)

// Legacy quality handlers - перемещены из server.go для рефакторинга
// TODO: Заменить на новые handlers из internal/api/handlers/

// handleQualityUploadRoutes обрабатывает маршруты качества для выгрузок
func (s *Server) handleQualityUploadRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/upload/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	uploadUUID := parts[0]
	action := parts[1]

	switch action {
	case "quality-report":
		if r.Method == http.MethodGet {
			s.handleQualityReport(w, r, uploadUUID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "quality-analysis":
		if r.Method == http.MethodPost {
			s.handleQualityAnalysis(w, r, uploadUUID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		// Пропускаем другие маршруты
		return
	}
}

// handleQualityDatabaseRoutes обрабатывает маршруты качества для баз данных
func (s *Server) handleQualityDatabaseRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/databases/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		// Пропускаем другие маршруты баз данных - передаем в handleDatabaseV1Routes
		s.handleDatabaseV1Routes(w, r)
		return
	}

	databaseIDStr := parts[0]
	action := parts[1]

	// Проверяем, что это маршрут качества
	if action != "quality-dashboard" && action != "quality-issues" && action != "quality-trends" {
		// Пропускаем другие маршруты - передаем в handleDatabaseV1Routes
		s.handleDatabaseV1Routes(w, r)
		return
	}

	databaseID, err := ValidateIDPathParam(databaseIDStr, "database_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid database ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	switch action {
	case "quality-dashboard":
		if r.Method == http.MethodGet {
			s.handleQualityDashboard(w, r, databaseID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "quality-issues":
		if r.Method == http.MethodGet {
			s.handleQualityIssues(w, r, databaseID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "quality-trends":
		if r.Method == http.MethodGet {
			s.handleQualityTrends(w, r, databaseID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		// Пропускаем другие маршруты - передаем в handleDatabaseV1Routes
		s.handleDatabaseV1Routes(w, r)
	}
}

// handleQualityReport возвращает отчет о качестве выгрузки
func (s *Server) handleQualityReport(w http.ResponseWriter, r *http.Request, uploadUUID string) {
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		s.writeJSONError(w, r, "Upload not found", http.StatusNotFound)
		return
	}

	// Парсим параметры запроса
	summaryOnly := r.URL.Query().Get("summary_only") == "true"

	// Получаем метрики качества
	metrics, err := s.db.GetQualityMetrics(upload.ID)
	if err != nil {
		s.writeJSONError(w, r, "Failed to get quality metrics", http.StatusInternalServerError)
		return
	}

	// Определяем параметры пагинации с валидацией
	limit := 0
	offset := 0

	// Сначала проверяем max_issues, затем limit (limit имеет приоритет)
	if maxIssues, err := ValidateIntParam(r, "max_issues", 0, 1, 0); err == nil && maxIssues > 0 {
		limit = maxIssues
	}
	if limitVal, err := ValidateIntParam(r, "limit", 0, 1, 0); err == nil && limitVal > 0 {
		limit = limitVal
	}
	if offsetVal, err := ValidateIntParam(r, "offset", 0, 0, 0); err == nil && offsetVal >= 0 {
		offset = offsetVal
	}

	// Если summary_only=true, не загружаем issues
	var issues []database.DataQualityIssue
	var totalIssuesCount int
	if !summaryOnly {
		// Получаем проблемы качества с пагинацией
		issues, totalIssuesCount, err = s.db.GetQualityIssues(upload.ID, map[string]interface{}{}, limit, offset)
		if err != nil {
			s.writeJSONError(w, r, "Failed to get quality issues", http.StatusInternalServerError)
			return
		}
	} else {
		// Для сводки получаем только количество без деталей
		_, totalIssuesCount, err = s.db.GetQualityIssues(upload.ID, map[string]interface{}{}, 0, 0)
		if err != nil {
			s.writeJSONError(w, r, "Failed to count quality issues", http.StatusInternalServerError)
			return
		}
		issues = []database.DataQualityIssue{} // Пустой список
	}

	// Формируем сводку
	summary := QualitySummary{
		TotalIssues:       totalIssuesCount,
		MetricsByCategory: make(map[string]float64),
	}

	// Подсчитываем проблемы по уровням серьезности
	// Если summary_only, используем только загруженные issues для подсчета
	// В противном случае нужно получить статистику отдельным запросом
	if !summaryOnly {
		for _, issue := range issues {
			switch issue.IssueSeverity {
			case "CRITICAL":
				summary.CriticalIssues++
			case "HIGH":
				summary.HighIssues++
			case "MEDIUM":
				summary.MediumIssues++
			case "LOW":
				summary.LowIssues++
			}
		}
	} else {
		// Для summary_only получаем статистику по уровням отдельным запросом
		severityStats, err := s.getIssuesSeverityStats(upload.ID)
		if err == nil {
			summary.CriticalIssues = severityStats["CRITICAL"]
			summary.HighIssues = severityStats["HIGH"]
			summary.MediumIssues = severityStats["MEDIUM"]
			summary.LowIssues = severityStats["LOW"]
		}
	}

	// Группируем метрики по категориям
	for _, metric := range metrics {
		if _, exists := summary.MetricsByCategory[metric.MetricCategory]; !exists {
			summary.MetricsByCategory[metric.MetricCategory] = 0.0
		}
		summary.MetricsByCategory[metric.MetricCategory] += metric.MetricValue
	}

	// Рассчитываем средние значения
	for category := range summary.MetricsByCategory {
		count := 0
		for _, metric := range metrics {
			if metric.MetricCategory == category {
				count++
			}
		}
		if count > 0 {
			summary.MetricsByCategory[category] = summary.MetricsByCategory[category] / float64(count)
		}
	}

	databaseID := 0
	if upload.DatabaseID != nil {
		databaseID = *upload.DatabaseID
	}

	report := QualityReport{
		UploadUUID:   uploadUUID,
		DatabaseID:   databaseID,
		AnalyzedAt:   upload.CompletedAt,
		OverallScore: 0.0,
		Metrics:      metrics,
		Issues:       issues,
		Summary:      summary,
	}

	// Рассчитываем общий балл
	if len(metrics) > 0 {
		var totalScore float64
		for _, metric := range metrics {
			totalScore += metric.MetricValue
		}
		report.OverallScore = totalScore / float64(len(metrics))
	}

	// Добавляем метаданные пагинации, если используется пагинация
	response := map[string]interface{}{
		"upload_uuid":   report.UploadUUID,
		"database_id":   report.DatabaseID,
		"analyzed_at":   report.AnalyzedAt,
		"overall_score": report.OverallScore,
		"metrics":       report.Metrics,
		"issues":        report.Issues,
		"summary":       report.Summary,
	}

	if limit > 0 {
		response["pagination"] = map[string]interface{}{
			"limit":       limit,
			"offset":      offset,
			"total_count": totalIssuesCount,
			"returned":    len(issues),
			"has_more":    offset+len(issues) < totalIssuesCount,
		}
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// getIssuesSeverityStats получает статистику по уровням серьезности проблем
func (s *Server) getIssuesSeverityStats(uploadID int) (map[string]int, error) {
	query := `
		SELECT issue_severity, COUNT(*) as count
		FROM data_quality_issues
		WHERE upload_id = ?
		GROUP BY issue_severity
	`

	rows, err := s.db.Query(query, uploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to query severity stats: %w", err)
	}
	defer rows.Close()

	stats := map[string]int{
		"CRITICAL": 0,
		"HIGH":     0,
		"MEDIUM":   0,
		"LOW":      0,
	}

	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			continue
		}
		stats[severity] = count
	}

	return stats, nil
}

// handleQualityAnalysis запускает анализ качества для выгрузки
func (s *Server) handleQualityAnalysis(w http.ResponseWriter, r *http.Request, uploadUUID string) {
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		s.writeJSONError(w, r, "Upload not found", http.StatusNotFound)
		return
	}

	databaseID := 0
	if upload.DatabaseID != nil {
		databaseID = *upload.DatabaseID
	}

	if databaseID == 0 {
		s.writeJSONError(w, r, "Database ID not set for upload", http.StatusBadRequest)
		return
	}

	// Запускаем анализ в фоне
	go func() {
		s.logInfo(fmt.Sprintf("Starting quality analysis for upload %s (ID: %d, Database: %d)", uploadUUID, upload.ID, databaseID), r.URL.Path)
		if err := s.qualityAnalyzer.AnalyzeUpload(upload.ID, databaseID); err != nil {
			s.logError(fmt.Sprintf("Quality analysis failed for upload %s: %v", uploadUUID, err), r.URL.Path)
		} else {
			s.logInfo(fmt.Sprintf("Quality analysis completed for upload %s", uploadUUID), r.URL.Path)
		}
	}()

	response := map[string]interface{}{
		"status":  "analysis_started",
		"message": "Quality analysis started in background",
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleQualityDashboard возвращает дашборд качества для базы данных
func (s *Server) handleQualityDashboard(w http.ResponseWriter, r *http.Request, databaseID int) {
	// Получаем тренды качества
	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := ValidateIntParam(r, "days", 30, 1, 0); err == nil && d > 0 {
			days = d
		}
	}

	trends, err := s.db.GetQualityTrends(databaseID, days)
	if err != nil {
		s.writeJSONError(w, r, "Failed to get quality trends", http.StatusInternalServerError)
		return
	}

	// Текущие метрики
	currentMetrics, err := s.db.GetCurrentQualityMetrics(databaseID)
	if err != nil {
		s.writeJSONError(w, r, "Failed to get current metrics", http.StatusInternalServerError)
		return
	}

	// Топ проблем
	limit, err := ValidateIntParam(r, "limit", 10, 1, 0)
	if err != nil {
		limit = 10 // Используем значение по умолчанию при ошибке
	}

	topIssues, err := s.db.GetTopQualityIssues(databaseID, limit)
	if err != nil {
		s.writeJSONError(w, r, "Failed to get top issues", http.StatusInternalServerError)
		return
	}

	// Группируем метрики по сущностям
	metricsByEntity := make(map[string]EntityMetrics)

	for _, metric := range currentMetrics {
		// Определяем тип сущности из имени метрики
		entityType := "unknown"
		if strings.Contains(metric.MetricName, "nomenclature") {
			entityType = "nomenclature"
		} else if strings.Contains(metric.MetricName, "counterparty") {
			entityType = "counterparty"
		}

		if _, exists := metricsByEntity[entityType]; !exists {
			metricsByEntity[entityType] = EntityMetrics{}
		}

		entityMetrics := metricsByEntity[entityType]
		switch metric.MetricCategory {
		case "completeness":
			entityMetrics.Completeness = metric.MetricValue
		case "consistency":
			entityMetrics.Consistency = metric.MetricValue
		case "uniqueness":
			entityMetrics.Uniqueness = metric.MetricValue
		case "validity":
			entityMetrics.Validity = metric.MetricValue
		}

		// Рассчитываем общий балл
		count := 0
		total := 0.0
		if entityMetrics.Completeness > 0 {
			total += entityMetrics.Completeness
			count++
		}
		if entityMetrics.Consistency > 0 {
			total += entityMetrics.Consistency
			count++
		}
		if entityMetrics.Uniqueness > 0 {
			total += entityMetrics.Uniqueness
			count++
		}
		if entityMetrics.Validity > 0 {
			total += entityMetrics.Validity
			count++
		}
		if count > 0 {
			entityMetrics.OverallScore = total / float64(count)
		}

		metricsByEntity[entityType] = entityMetrics
	}

	// Рассчитываем текущий общий балл
	currentScore := 0.0
	if len(trends) > 0 {
		currentScore = trends[0].OverallScore
	} else if len(currentMetrics) > 0 {
		var total float64
		for _, metric := range currentMetrics {
			total += metric.MetricValue
		}
		currentScore = total / float64(len(currentMetrics))
	}

	dashboard := QualityDashboard{
		DatabaseID:      databaseID,
		CurrentScore:    currentScore,
		Trends:          trends,
		TopIssues:       topIssues,
		MetricsByEntity: metricsByEntity,
	}

	s.writeJSONResponse(w, r, dashboard, http.StatusOK)
}

// handleQualityIssues возвращает проблемы качества для базы данных
func (s *Server) handleQualityIssues(w http.ResponseWriter, r *http.Request, databaseID int) {
	// Получаем параметры фильтрации
	filters := make(map[string]interface{})

	if entityType := r.URL.Query().Get("entity_type"); entityType != "" {
		filters["entity_type"] = entityType
	}

	if severity := r.URL.Query().Get("severity"); severity != "" {
		filters["severity"] = severity
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	// Получаем все выгрузки для базы данных
	uploads, err := s.db.GetAllUploads()
	if err != nil {
		s.writeJSONError(w, r, "Failed to get uploads", http.StatusInternalServerError)
		return
	}

	var allIssues []database.DataQualityIssue
	for _, upload := range uploads {
		if upload.DatabaseID != nil && *upload.DatabaseID == databaseID {
			issues, _, err := s.db.GetQualityIssues(upload.ID, filters, 0, 0)
			if err != nil {
				continue
			}
			allIssues = append(allIssues, issues...)
		}
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"issues": allIssues,
		"total":  len(allIssues),
	}, http.StatusOK)
}

// handleQualityTrends возвращает тренды качества для базы данных
func (s *Server) handleQualityTrends(w http.ResponseWriter, r *http.Request, databaseID int) {
	days, err := ValidateIntParam(r, "days", 30, 1, 365)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	trends, err := s.db.GetQualityTrends(databaseID, days)
	if err != nil {
		s.writeJSONError(w, r, "Failed to get quality trends", http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"trends": trends,
		"total":  len(trends),
	}, http.StatusOK)
}

// handle1CProcessingXML генерирует актуальный XML файл обработки 1С
