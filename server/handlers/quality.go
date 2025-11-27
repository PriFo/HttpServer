package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/quality"
	apperrors "httpserver/server/errors"
	"httpserver/server/services"
	"httpserver/server/types"
)

// QualityAnalysisStatus статус анализа качества
type QualityAnalysisStatus struct {
	IsRunning        bool    `json:"is_running"`
	Progress         float64 `json:"progress"`
	Processed        int     `json:"processed"`
	Total            int     `json:"total"`
	CurrentStep      string  `json:"current_step"`
	DuplicatesFound  int     `json:"duplicates_found"`
	ViolationsFound  int     `json:"violations_found"`
	SuggestionsFound int     `json:"suggestions_found"`
	Error            string  `json:"error,omitempty"`
}

// QualityHandler обработчик для качества данных
type QualityHandler struct {
	*BaseHandler
	qualityService          services.QualityServiceInterface
	logFunc                 func(entry interface{}) // types.LogEntry, но без прямого импорта
	normalizedDB            *database.DB
	currentNormalizedDBPath string
	generateQualityReport   func(string) (interface{}, error)                                         // Функция для генерации отчета
	getProjectDatabases     func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error) // Функция для получения баз проекта
	projectStatsCache       *ProjectQualityStatsCache                                                 // Кэш для статистики проектов
	// Поля для отслеживания статуса анализа
	qualityAnalysisRunning bool
	qualityAnalysisMutex   sync.RWMutex
	qualityAnalysisStatus  QualityAnalysisStatus
}

// NewQualityHandler создает новый обработчик качества
// Принимает интерфейс QualityServiceInterface для улучшения тестируемости
func NewQualityHandler(baseHandler *BaseHandler, qualityService services.QualityServiceInterface, logFunc func(entry interface{}), normalizedDB *database.DB, currentNormalizedDBPath string) *QualityHandler {
	return &QualityHandler{
		BaseHandler:             baseHandler,
		qualityService:          qualityService,
		logFunc:                 logFunc,
		normalizedDB:            normalizedDB,
		currentNormalizedDBPath: currentNormalizedDBPath,
		qualityAnalysisRunning:  false,
		qualityAnalysisStatus:   QualityAnalysisStatus{},
	}
}

// SetNormalizedDB устанавливает normalizedDB и путь для работы с качеством
func (h *QualityHandler) SetNormalizedDB(normalizedDB *database.DB, currentNormalizedDBPath string) {
	h.normalizedDB = normalizedDB
	h.currentNormalizedDBPath = currentNormalizedDBPath
}

// SetGenerateQualityReport устанавливает функцию для генерации отчета о качестве
func (h *QualityHandler) SetGenerateQualityReport(fn func(string) (interface{}, error)) {
	h.generateQualityReport = fn
}

// SetGetProjectDatabases устанавливает функцию для получения баз данных проекта
func (h *QualityHandler) SetGetProjectDatabases(fn func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error)) {
	h.getProjectDatabases = fn
}

// SetProjectStatsCache устанавливает кэш для статистики проектов
func (h *QualityHandler) SetProjectStatsCache(cache *ProjectQualityStatsCache) {
	h.projectStatsCache = cache
}

// getDB получает БД по пути, используя normalizedDB по умолчанию
func (h *QualityHandler) getDB(databasePath string) (*database.DB, error) {
	if databasePath == "" {
		databasePath = h.currentNormalizedDBPath
	}

	if databasePath != "" && databasePath != h.currentNormalizedDBPath {
		db, err := database.NewDB(databasePath)
		if err != nil {
			return nil, apperrors.NewInternalError(fmt.Sprintf("failed to open database %s", databasePath), err)
		}
		return db, nil
	}

	return h.normalizedDB, nil
}

// HandleQualityStats обрабатывает запрос статистики качества
func (h *QualityHandler) HandleQualityStats(w http.ResponseWriter, r *http.Request, currentDB *database.DB, currentNormalizedDBPath string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры из query
	databasePath := r.URL.Query().Get("database")
	projectParam := r.URL.Query().Get("project")

	// Если указан проект, получаем агрегированную статистику по всем базам проекта
	if projectParam != "" && h.getProjectDatabases != nil {
		// Парсим project (формат: "clientId:projectId")
		parts := strings.Split(projectParam, ":")
		if len(parts) == 2 {
			projectID, err := strconv.Atoi(parts[1])
			if err == nil {
				// Проверяем кэш
				cacheKey := fmt.Sprintf("project:%d", projectID)
				if h.projectStatsCache != nil {
					if cachedStats, found := h.projectStatsCache.Get(cacheKey); found {
						h.logFunc(types.LogEntry{
							Timestamp: time.Now(),
							Level:     "INFO",
							Message:   fmt.Sprintf("Returning cached stats for project %d", projectID),
							Endpoint:  r.URL.Path,
						})
						h.WriteJSONResponse(w, r, cachedStats, http.StatusOK)
						return
					}
				}

				// Получаем все базы данных проекта
				projectDatabases, err := h.getProjectDatabases(projectID, true) // activeOnly = true
				if err != nil {
					h.logFunc(types.LogEntry{
						Timestamp: time.Now(),
						Level:     "ERROR",
						Message:   fmt.Sprintf("Error getting project databases for project_id %d: %v", projectID, err),
						Endpoint:  r.URL.Path,
					})
					// Проверяем тип ошибки для более точного HTTP статуса
					if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
						h.WriteJSONError(w, r, fmt.Sprintf("Project not found: %v", err), http.StatusNotFound)
					} else {
						h.WriteJSONError(w, r, fmt.Sprintf("Failed to get project databases: %v", err), http.StatusInternalServerError)
					}
					return
				}

				// Если нет баз данных, возвращаем пустую статистику
				if len(projectDatabases) == 0 {
					h.logFunc(types.LogEntry{
						Timestamp: time.Now(),
						Level:     "INFO",
						Message:   fmt.Sprintf("No active databases found for project_id %d", projectID),
						Endpoint:  r.URL.Path,
					})
					emptyStats := map[string]interface{}{
						"total_items":          0,
						"by_level":             make(map[string]interface{}),
						"average_quality":      0.0,
						"benchmark_count":      0,
						"benchmark_percentage": 0.0,
						"databases":            []interface{}{},
						"databases_count":      0,
					}
					h.WriteJSONResponse(w, r, emptyStats, http.StatusOK)
					return
				}

				// Агрегируем статистику по всем базам проекта
				startTime := time.Now()
				aggregatedStats, err := h.aggregateProjectStats(r.Context(), projectDatabases, currentDB)
				duration := time.Since(startTime)

				if err != nil {
					h.logFunc(types.LogEntry{
						Timestamp: time.Now(),
						Level:     "ERROR",
						Message:   fmt.Sprintf("Error aggregating project stats: %v", err),
						Endpoint:  r.URL.Path,
					})
					h.WriteJSONError(w, r, fmt.Sprintf("Failed to aggregate project stats: %v", err), http.StatusInternalServerError)
					return
				}

				// Сохраняем в кэш
				if h.projectStatsCache != nil {
					h.projectStatsCache.Set(cacheKey, aggregatedStats)
				}

				// Логируем производительность
				h.logFunc(types.LogEntry{
					Timestamp: time.Now(),
					Level:     "INFO",
					Message:   fmt.Sprintf("Aggregated stats for project %d (%d databases) in %v", projectID, len(projectDatabases), duration),
					Endpoint:  r.URL.Path,
				})

				h.WriteJSONResponse(w, r, aggregatedStats, http.StatusOK)
				return
			}
		}
	}

	// Если указана конкретная база данных или нет проекта, работаем как раньше
	if databasePath == "" {
		// Если не указан, используем normalizedDB по умолчанию
		databasePath = currentNormalizedDBPath
	}

	// Проверяем, что databasePath указан
	if databasePath == "" {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   "Database path is not specified and no default normalized database available",
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, "Database path is required", http.StatusBadRequest)
		return
	}

	// Проверяем, что currentDB доступен, если не указан databasePath
	if currentDB == nil {
		// Пытаемся использовать normalizedDB из handler
		if h.normalizedDB == nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   "Database connection is not available",
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, "Database connection is not available", http.StatusInternalServerError)
			return
		}
		currentDB = h.normalizedDB
	}

	// Создаем адаптер для *database.DB в DatabaseInterface
	dbAdapter := services.NewDatabaseAdapter(currentDB)
	if dbAdapter == nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Failed to create database adapter",
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, "Database connection is not available", http.StatusInternalServerError)
		return
	}
	
	stats, err := h.qualityService.GetQualityStats(r.Context(), databasePath, dbAdapter)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting quality stats for database %s: %v", databasePath, err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality stats: %v", err), http.StatusInternalServerError)
		return
	}

	if stats == nil {
		// Возвращаем пустую статистику вместо ошибки
		stats = map[string]interface{}{
			"total_items":          0,
			"by_level":             make(map[string]interface{}),
			"average_quality":      0.0,
			"benchmark_count":      0,
			"benchmark_percentage": 0.0,
		}
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// aggregateProjectStats агрегирует статистику качества по всем базам данных проекта
func (h *QualityHandler) aggregateProjectStats(ctx context.Context, projectDatabases []*database.ProjectDatabase, currentDB *database.DB) (interface{}, error) {
	if len(projectDatabases) == 0 {
		// Возвращаем пустую статистику, если нет баз данных
		return map[string]interface{}{
			"total_items":          0,
			"by_level":             make(map[string]interface{}),
			"average_quality":      0.0,
			"benchmark_count":      0,
			"benchmark_percentage": 0.0,
			"databases":            []interface{}{},
			"databases_count":      0,
		}, nil
	}

	// Создаем адаптер для currentDB
	// Проверяем, что currentDB не nil
	if currentDB == nil {
		// Если currentDB nil, пытаемся использовать normalizedDB из handler
		if h.normalizedDB == nil {
			return nil, apperrors.NewInternalError("database connection is not available", nil)
		}
		currentDB = h.normalizedDB
	}
	
	dbAdapter := services.NewDatabaseAdapter(currentDB)
	if dbAdapter == nil {
		return nil, apperrors.NewInternalError("database connection is not available", nil)
	}

	var totalItems int
	var totalQualitySum float64
	var totalQualityWeight int
	var benchmarkCount int
	var byLevel = make(map[string]struct {
		count         int
		qualitySum    float64
		qualityWeight int
	})
	var databasesStats []map[string]interface{}
	processedDBs := 0
	var overallLastActivity *time.Time

	// Собираем статистику для каждой базы данных параллельно
	type dbStatsResult struct {
		db    *database.ProjectDatabase
		stats interface{}
		err   error
	}

	statsChan := make(chan dbStatsResult, len(projectDatabases))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Ограничиваем количество одновременных запросов до 5

	// Подсчитываем активные БД для таймаута
	activeDBsCount := 0
	for _, projectDB := range projectDatabases {
		if projectDB.IsActive {
			activeDBsCount++
		}
	}

	for _, projectDB := range projectDatabases {
		if !projectDB.IsActive {
			continue
		}

		wg.Add(1)
		go func(db *database.ProjectDatabase) {
			defer wg.Done()
			semaphore <- struct{}{}        // Захватываем слот
			defer func() { <-semaphore }() // Освобождаем слот

			// Получаем статистику для базы данных
			stats, err := h.qualityService.GetQualityStats(ctx, db.FilePath, dbAdapter)
			statsChan <- dbStatsResult{db: db, stats: stats, err: err}
		}(projectDB)
	}

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(statsChan)
	}()

	// Обрабатываем результаты с таймаутом
	// Таймаут зависит от количества активных баз данных (минимум 30 секунд, +5 секунд на каждую БД)
	timeoutDuration := 30*time.Second + time.Duration(activeDBsCount)*5*time.Second
	if timeoutDuration > 2*time.Minute {
		timeoutDuration = 2 * time.Minute // Максимум 2 минуты
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	errorCount := 0
	maxErrors := 3 // Логируем только первые 3 ошибки, чтобы не засорять логи

	// Обрабатываем результаты
resultLoop:
	for result := range statsChan {
		// Проверяем контекст на отмену
		select {
		case <-ctxWithTimeout.Done():
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "WARN",
				Message:   fmt.Sprintf("Timeout while aggregating project stats: %v", ctxWithTimeout.Err()),
				Endpoint:  "/api/quality/stats",
			})
			break resultLoop
		default:
		}

		if result.err != nil {
			errorCount++
			// Пропускаем базы с ошибками, но логируем (ограничиваем количество логов)
			if errorCount <= maxErrors {
				h.logFunc(types.LogEntry{
					Timestamp: time.Now(),
					Level:     "WARN",
					Message:   fmt.Sprintf("Failed to get stats for database %s (ID: %d): %v", result.db.FilePath, result.db.ID, result.err),
					Endpoint:  "/api/quality/stats",
				})
			}
			continue
		}

		stats := result.stats

		// Преобразуем статистику в map для обработки
		statsMap, ok := stats.(map[string]interface{})
		if !ok {
			// Пытаемся преобразовать через JSON
			statsJSON, err := json.Marshal(stats)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(statsJSON, &statsMap); err != nil {
				continue
			}
		}

		// Добавляем статистику базы в список
		dbStat := map[string]interface{}{
			"database_id":   result.db.ID,
			"database_name": result.db.Name,
			"database_path": result.db.FilePath,
			"stats":         statsMap,
		}

		// Добавляем информацию об активности базы данных
		lastUsedAt := cloneTimePtr(result.db.LastUsedAt)
		lastUploadAt := h.getDatabaseLastUploadAt(result.db.FilePath, result.db.ID)

		if lastUsedAt != nil {
			dbStat["last_used_at"] = formatTimeRFC3339(lastUsedAt)
		}
		if lastUploadAt != nil {
			dbStat["last_upload_at"] = formatTimeRFC3339(lastUploadAt)
		}

		if latest := latestTime(lastUploadAt, lastUsedAt); latest != nil {
			dbStat["last_activity"] = formatTimeRFC3339(latest)
			if overallLastActivity == nil || latest.After(*overallLastActivity) {
				copy := *latest
				overallLastActivity = &copy
			}
		}
		databasesStats = append(databasesStats, dbStat)

		// Агрегируем общие метрики
		var dbItems int
		if items, ok := statsMap["total_items"].(float64); ok {
			dbItems = int(items)
			totalItems += dbItems
		} else if items, ok := statsMap["total_items"].(int); ok {
			dbItems = items
			totalItems += items
		}

		if quality, ok := statsMap["average_quality"].(float64); ok && dbItems > 0 {
			totalQualitySum += quality * float64(dbItems)
			totalQualityWeight += dbItems
		}

		if benchmark, ok := statsMap["benchmark_count"].(float64); ok {
			benchmarkCount += int(benchmark)
		} else if benchmark, ok := statsMap["benchmark_count"].(int); ok {
			benchmarkCount += benchmark
		}

		// Агрегируем статистику по уровням
		if byLevelData, ok := statsMap["by_level"].(map[string]interface{}); ok {
			for level, levelData := range byLevelData {
				if levelMap, ok := levelData.(map[string]interface{}); ok {
					levelStat := byLevel[level]
					if levelStat.count == 0 && levelStat.qualityWeight == 0 {
						// Инициализируем структуру, если её еще нет
						levelStat = struct {
							count         int
							qualitySum    float64
							qualityWeight int
						}{}
					}

					var levelCount int
					if count, ok := levelMap["count"].(float64); ok {
						levelCount = int(count)
					} else if count, ok := levelMap["count"].(int); ok {
						levelCount = count
					}
					levelStat.count += levelCount

					if avgQuality, ok := levelMap["avg_quality"].(float64); ok && levelCount > 0 {
						levelStat.qualitySum += avgQuality * float64(levelCount)
						levelStat.qualityWeight += levelCount
					}

					byLevel[level] = levelStat
				}
			}
		}
		processedDBs++
	}

	// Вычисляем среднее качество (взвешенное по количеству элементов)
	averageQuality := 0.0
	if totalQualityWeight > 0 {
		averageQuality = totalQualitySum / float64(totalQualityWeight)
	}

	// Формируем результат для by_level
	byLevelResult := make(map[string]interface{})
	for level, levelStat := range byLevel {
		avgQuality := 0.0
		if levelStat.qualityWeight > 0 {
			avgQuality = levelStat.qualitySum / float64(levelStat.qualityWeight)
		}
		percentage := 0.0
		if totalItems > 0 {
			percentage = float64(levelStat.count) / float64(totalItems) * 100.0
		}
		byLevelResult[level] = map[string]interface{}{
			"count":       levelStat.count,
			"avg_quality": avgQuality,
			"percentage":  percentage,
		}
	}

	// Вычисляем процент эталонов
	benchmarkPercentage := 0.0
	if totalItems > 0 {
		benchmarkPercentage = float64(benchmarkCount) / float64(totalItems) * 100.0
	}

	// Формируем результат
	result := map[string]interface{}{
		"total_items":          totalItems,
		"by_level":             byLevelResult,
		"average_quality":      averageQuality,
		"benchmark_count":      benchmarkCount,
		"benchmark_percentage": benchmarkPercentage,
		"databases":            databasesStats,
		"databases_count":      len(databasesStats),
		"databases_processed":  processedDBs,
	}

	if overallLastActivity != nil {
		result["last_activity"] = formatTimeRFC3339(overallLastActivity)
	}

	return result, nil
}

func (h *QualityHandler) getDatabaseLastUploadAt(dbPath string, databaseID int) *time.Time {
	if dbPath == "" || databaseID <= 0 {
		return nil
	}

	db, err := database.NewDB(dbPath)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Failed to open database for last upload check (%s): %v", dbPath, err),
			Endpoint:  "/api/quality/stats",
		})
		return nil
	}
	defer db.Close()

	uploads, err := db.GetLatestUploads(databaseID, 1)
	if err != nil || len(uploads) == 0 {
		if err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "WARN",
				Message:   fmt.Sprintf("Failed to fetch latest uploads for database %d: %v", databaseID, err),
				Endpoint:  "/api/quality/stats",
			})
		}
		return nil
	}

	latest := uploads[0]
	if latest.CompletedAt != nil {
		copy := latest.CompletedAt.UTC()
		return &copy
	}
	copy := latest.StartedAt.UTC()
	return &copy
}

func formatTimeRFC3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	copy := t.UTC()
	return &copy
}

func latestTime(times ...*time.Time) *time.Time {
	var latest *time.Time
	for _, t := range times {
		if t == nil {
			continue
		}
		if latest == nil || t.After(*latest) {
			copy := *t
			latest = &copy
		}
	}
	return latest
}

// HandleQualityUploadRoutes обрабатывает маршруты качества для выгрузок
func (h *QualityHandler) HandleQualityUploadRoutes(w http.ResponseWriter, r *http.Request) {
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
			h.HandleQualityReport(w, r, uploadUUID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "quality-analysis":
		if r.Method == http.MethodPost {
			h.HandleQualityAnalysis(w, r, uploadUUID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		// Пропускаем другие маршруты
		return
	}
}

// HandleQualityDatabaseRoutes обрабатывает маршруты качества для баз данных
func (h *QualityHandler) HandleQualityDatabaseRoutes(w http.ResponseWriter, r *http.Request, handleDatabaseV1Routes func(http.ResponseWriter, *http.Request)) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/databases/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		// Пропускаем другие маршруты баз данных - передаем в handleDatabaseV1Routes
		handleDatabaseV1Routes(w, r)
		return
	}

	databaseIDStr := parts[0]
	action := parts[1]

	// Проверяем, что это маршрут качества
	if action != "quality-dashboard" && action != "quality-issues" && action != "quality-trends" {
		// Пропускаем другие маршруты - передаем в handleDatabaseV1Routes
		handleDatabaseV1Routes(w, r)
		return
	}

	databaseID, err := ValidateIDPathParam(databaseIDStr, "database_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid database ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	switch action {
	case "quality-dashboard":
		if r.Method == http.MethodGet {
			h.HandleQualityDashboard(w, r, databaseID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "quality-issues":
		if r.Method == http.MethodGet {
			h.HandleQualityIssues(w, r, databaseID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "quality-trends":
		if r.Method == http.MethodGet {
			h.HandleQualityTrends(w, r, databaseID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		// Пропускаем другие маршруты - передаем в handleDatabaseV1Routes
		handleDatabaseV1Routes(w, r)
	}
}

// HandleQualityReport возвращает отчет о качестве выгрузки
func (h *QualityHandler) HandleQualityReport(w http.ResponseWriter, r *http.Request, uploadUUID string) {
	// Парсим параметры запроса
	summaryOnly := r.URL.Query().Get("summary_only") == "true"

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

	report, err := h.qualityService.GetQualityReport(r.Context(), uploadUUID, summaryOnly, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.WriteJSONError(w, r, "Upload not found", http.StatusNotFound)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality report: %v", err), http.StatusInternalServerError)
		}
		return
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
		totalIssuesCount := report.Summary.TotalIssues
		response["pagination"] = map[string]interface{}{
			"limit":       limit,
			"offset":      offset,
			"total_count": totalIssuesCount,
			"returned":    len(report.Issues),
			"has_more":    offset+len(report.Issues) < totalIssuesCount,
		}
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleQualityAnalysis запускает анализ качества для выгрузки
func (h *QualityHandler) HandleQualityAnalysis(w http.ResponseWriter, r *http.Request, uploadUUID string) {
	// Запускаем анализ в фоне
	go func() {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Starting quality analysis for upload %s", uploadUUID),
			Endpoint:  r.URL.Path,
		})
		if err := h.qualityService.AnalyzeQuality(r.Context(), uploadUUID); err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Quality analysis failed for upload %s: %v", uploadUUID, err),
				Endpoint:  r.URL.Path,
			})
		} else {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   fmt.Sprintf("Quality analysis completed for upload %s", uploadUUID),
				Endpoint:  r.URL.Path,
			})
		}
	}()

	response := map[string]interface{}{
		"status":  "analysis_started",
		"message": fmt.Sprintf("Quality analysis started for upload %s", uploadUUID),
	}

	h.WriteJSONResponse(w, r, response, http.StatusAccepted)
}

// HandleQualityDashboard возвращает дашборд качества для базы данных
func (h *QualityHandler) HandleQualityDashboard(w http.ResponseWriter, r *http.Request, databaseID int) {
	// Получаем тренды качества
	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := ValidateIntParam(r, "days", 30, 1, 0); err == nil && d > 0 {
			days = d
		}
	}

	limit, err := ValidateIntParam(r, "limit", 10, 1, 0)
	if err != nil {
		limit = 10 // Используем значение по умолчанию при ошибке
	}

	dashboard, err := h.qualityService.GetQualityDashboard(r.Context(), databaseID, days, limit)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality dashboard: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, dashboard, http.StatusOK)
}

// HandleQualityIssues возвращает проблемы качества для базы данных
func (h *QualityHandler) HandleQualityIssues(w http.ResponseWriter, r *http.Request, databaseID int) {
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

	issues, err := h.qualityService.GetQualityIssues(r.Context(), databaseID, filters)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality issues: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"issues": issues,
		"total":  len(issues),
	}, http.StatusOK)
}

// HandleQualityTrends возвращает тренды качества для базы данных
func (h *QualityHandler) HandleQualityTrends(w http.ResponseWriter, r *http.Request, databaseID int) {
	days, err := ValidateIntParam(r, "days", 30, 1, 365)
	if err != nil {
		if h.HandleValidationError(w, r, err) {
			return
		}
	}

	trends, err := h.qualityService.GetQualityTrends(r.Context(), databaseID, days)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality trends: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"trends": trends,
		"total":  len(trends),
	}, http.StatusOK)
}

// HandleGetQualityReport обрабатывает запросы к /api/quality/report
func (h *QualityHandler) HandleGetQualityReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметр database из query
	databasePath := r.URL.Query().Get("database")
	if databasePath == "" {
		databasePath = h.currentNormalizedDBPath
	}

	// Открываем нужную БД
	var db *database.DB
	var err error
	if databasePath != "" && databasePath != h.currentNormalizedDBPath {
		db, err = database.NewDB(databasePath)
		if err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error opening database %s: %v", databasePath, err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
			return
		}
		defer db.Close()
	} else {
		if h.normalizedDB == nil {
			h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
			return
		}
		db = h.normalizedDB
	}

	// Получаем статистику качества
	stats, err := db.GetQualityStats()
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting quality stats: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем топ дубликатов
	duplicateGroups, _, err := db.GetDuplicateGroups(false, 10, 0)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Error getting duplicate groups: %v", err),
			Endpoint:  r.URL.Path,
		})
		duplicateGroups = []database.DuplicateGroup{}
	}

	// Получаем топ нарушений
	violations, _, err := db.GetViolations(map[string]interface{}{}, 10, 0)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Error getting violations: %v", err),
			Endpoint:  r.URL.Path,
		})
		violations = []database.QualityViolation{}
	}

	// Получаем топ предложений
	suggestions, _, err := db.GetSuggestions(map[string]interface{}{"applied": false}, 10, 0)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Error getting suggestions: %v", err),
			Endpoint:  r.URL.Path,
		})
		suggestions = []database.QualitySuggestion{}
	}

	response := map[string]interface{}{
		"stats":        stats,
		"duplicates":   duplicateGroups,
		"violations":   violations,
		"suggestions":  suggestions,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleQualityItemDetail обрабатывает запросы к /api/quality/item/{id}
func (h *QualityHandler) HandleQualityItemDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/quality/item/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		h.WriteJSONError(w, r, "Item ID is required", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(pathParts[0])
	if err != nil {
		h.WriteJSONError(w, r, "Invalid item ID", http.StatusBadRequest)
		return
	}

	if h.normalizedDB == nil {
		h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
		return
	}

	// Получаем последнюю оценку качества
	assessment, err := h.normalizedDB.GetQualityAssessment(itemID)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting quality assessment for item %d: %v", itemID, err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality assessment: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем violations для этой записи
	violations, _, err := h.normalizedDB.GetViolations(map[string]interface{}{
		"normalized_item_id": itemID,
	}, 100, 0)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Error getting violations for item %d: %v", itemID, err),
			Endpoint:  r.URL.Path,
		})
		violations = []database.QualityViolation{}
	}

	// Получаем suggestions для этой записи
	suggestions, _, err := h.normalizedDB.GetSuggestions(map[string]interface{}{
		"normalized_item_id": itemID,
		"applied":            false,
	}, 100, 0)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Error getting suggestions for item %d: %v", itemID, err),
			Endpoint:  r.URL.Path,
		})
		suggestions = []database.QualitySuggestion{}
	}

	response := map[string]interface{}{
		"assessment":  assessment,
		"violations":  violations,
		"suggestions": suggestions,
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleQualityViolations обрабатывает запросы к /api/quality/violations
func (h *QualityHandler) HandleQualityViolations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры из query
	databasePath := r.URL.Query().Get("database")
	projectParam := r.URL.Query().Get("project")
	searchParam := strings.TrimSpace(r.URL.Query().Get("search"))
	showResolvedParam := strings.ToLower(r.URL.Query().Get("show_resolved"))
	includeResolved := showResolvedParam == "true"

	// Если указан проект, собираем данные из всех баз проекта
	if projectParam != "" && h.getProjectDatabases != nil {
		parts := strings.Split(projectParam, ":")
		if len(parts) == 2 {
			projectID, err := strconv.Atoi(parts[1])
			if err == nil {
				projectDatabases, err := h.getProjectDatabases(projectID, true)
				if err == nil {
					// Собираем нарушения из всех баз проекта
					allViolations := []interface{}{}
					totalCount := 0

					for _, projectDB := range projectDatabases {
						if !projectDB.IsActive {
							continue
						}

						db, err := database.NewDB(projectDB.FilePath)
						if err != nil {
							continue
						}

						filters := make(map[string]interface{})
						if severity := r.URL.Query().Get("severity"); severity != "" {
							filters["severity"] = severity
						}
						if category := r.URL.Query().Get("category"); category != "" {
							filters["category"] = category
						}
						if searchParam != "" {
							filters["search"] = searchParam
						}
						if !includeResolved {
							filters["resolved"] = false
						}

						limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
						if limit <= 0 {
							limit = 1000
						}
						offset := 0

						violations, count, err := db.GetViolations(filters, limit, offset)
						db.Close()

						if err == nil {
							// Добавляем информацию о базе данных к каждому нарушению
							for _, violation := range violations {
								// Преобразуем структуру в map через JSON
								violationJSON, err := json.Marshal(violation)
								if err != nil {
									continue
								}
								var violationMap map[string]interface{}
								if err := json.Unmarshal(violationJSON, &violationMap); err != nil {
									continue
								}
								violationMap["database_id"] = projectDB.ID
								violationMap["database_name"] = projectDB.Name
								violationMap["database_path"] = projectDB.FilePath
								allViolations = append(allViolations, violationMap)
							}
							totalCount += count
						}
					}

					// Применяем пагинацию
					limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
					if limit <= 0 {
						limit = 50
					}
					offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
					if offset < 0 {
						offset = 0
					}

					start := offset
					end := offset + limit
					if start > len(allViolations) {
						start = len(allViolations)
					}
					if end > len(allViolations) {
						end = len(allViolations)
					}

					response := map[string]interface{}{
						"violations": allViolations[start:end],
						"total":      totalCount,
						"limit":      limit,
						"offset":     offset,
					}

					h.WriteJSONResponse(w, r, response, http.StatusOK)
					return
				}
			}
		}
	}

	// Если указана конкретная база данных или нет проекта, работаем как раньше
	if databasePath == "" {
		databasePath = h.currentNormalizedDBPath
	}

	// Открываем нужную БД
	var db *database.DB
	var err error
	if databasePath != "" && databasePath != h.currentNormalizedDBPath {
		db, err = database.NewDB(databasePath)
		if err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error opening database %s: %v", databasePath, err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
			return
		}
		defer db.Close()
	} else {
		if h.normalizedDB == nil {
			h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
			return
		}
		db = h.normalizedDB
	}

	// Параметры фильтрации
	filters := make(map[string]interface{})

	if severity := r.URL.Query().Get("severity"); severity != "" {
		filters["severity"] = severity
	}

	if category := r.URL.Query().Get("category"); category != "" {
		filters["category"] = category
	}

	if searchParam != "" {
		filters["search"] = searchParam
	}

	if !includeResolved {
		filters["resolved"] = false
	}

	// Pagination
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	violations, total, err := db.GetViolations(filters, limit, offset)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting violations: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get violations: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"violations": violations,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleQualityViolationDetail обрабатывает запросы к /api/quality/violations/{id}
func (h *QualityHandler) HandleQualityViolationDetail(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/quality/violations/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		h.WriteJSONError(w, r, "Violation ID is required", http.StatusBadRequest)
		return
	}

	violationID, err := strconv.Atoi(pathParts[0])
	if err != nil {
		h.WriteJSONError(w, r, "Invalid violation ID", http.StatusBadRequest)
		return
	}

	// GET - получить детали нарушения
	if r.Method == http.MethodGet {
		if h.normalizedDB == nil {
			h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
			return
		}

		violations, _, err := h.normalizedDB.GetViolations(map[string]interface{}{
			"id": violationID,
		}, 1, 0)
		if err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error getting violation %d: %v", violationID, err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to get violation: %v", err), http.StatusInternalServerError)
			return
		}

		if len(violations) == 0 {
			h.WriteJSONError(w, r, "Violation not found", http.StatusNotFound)
			return
		}

		h.WriteJSONResponse(w, r, violations[0], http.StatusOK)
		return
	}

	// POST - разрешить нарушение
	if r.Method == http.MethodPost {
		var reqBody struct {
			ResolvedBy string `json:"resolved_by"`
		}

		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}

		if h.normalizedDB == nil {
			h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
			return
		}

		if err := h.normalizedDB.ResolveViolation(violationID, reqBody.ResolvedBy); err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error resolving violation %d: %v", violationID, err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to resolve violation: %v", err), http.StatusInternalServerError)
			return
		}

		h.WriteJSONResponse(w, r, map[string]interface{}{
			"success": true,
			"message": "Violation resolved successfully",
		}, http.StatusOK)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleQualitySuggestions обрабатывает запросы к /api/quality/suggestions
func (h *QualityHandler) HandleQualitySuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры из query
	databasePath := r.URL.Query().Get("database")
	projectParam := r.URL.Query().Get("project")
	searchParam := strings.TrimSpace(r.URL.Query().Get("search"))
	suggestionType := r.URL.Query().Get("type")

	// Если указан проект, собираем данные из всех баз проекта
	if projectParam != "" && h.getProjectDatabases != nil {
		parts := strings.Split(projectParam, ":")
		if len(parts) == 2 {
			projectID, err := strconv.Atoi(parts[1])
			if err == nil {
				projectDatabases, err := h.getProjectDatabases(projectID, true)
				if err == nil {
					// Собираем предложения из всех баз проекта
					allSuggestions := []interface{}{}
					totalCount := 0

					for _, projectDB := range projectDatabases {
						if !projectDB.IsActive {
							continue
						}

						db, err := database.NewDB(projectDB.FilePath)
						if err != nil {
							continue
						}

						filters := make(map[string]interface{})
						if priority := r.URL.Query().Get("priority"); priority != "" {
							filters["priority"] = priority
						}
						if autoApplyable := r.URL.Query().Get("auto_applyable"); autoApplyable == "true" {
							filters["auto_applyable"] = true
						}
						if applied := r.URL.Query().Get("applied"); applied == "false" {
							filters["applied"] = false
						}
						if suggestionType != "" && suggestionType != "all" {
							filters["suggestion_type"] = suggestionType
						}
						if searchParam != "" {
							filters["suggestion_search"] = searchParam
						}

						limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
						if limit <= 0 {
							limit = 1000
						}
						offset := 0

						suggestions, count, err := db.GetSuggestions(filters, limit, offset)
						db.Close()

						if err == nil {
							// Добавляем информацию о базе данных к каждому предложению
							for _, suggestion := range suggestions {
								// Преобразуем структуру в map через JSON
								suggestionJSON, err := json.Marshal(suggestion)
								if err != nil {
									continue
								}
								var suggestionMap map[string]interface{}
								if err := json.Unmarshal(suggestionJSON, &suggestionMap); err != nil {
									continue
								}
								suggestionMap["database_id"] = projectDB.ID
								suggestionMap["database_name"] = projectDB.Name
								suggestionMap["database_path"] = projectDB.FilePath
								allSuggestions = append(allSuggestions, suggestionMap)
							}
							totalCount += count
						}
					}

					// Применяем пагинацию
					limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
					if limit <= 0 {
						limit = 50
					}
					offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
					if offset < 0 {
						offset = 0
					}

					start := offset
					end := offset + limit
					if start > len(allSuggestions) {
						start = len(allSuggestions)
					}
					if end > len(allSuggestions) {
						end = len(allSuggestions)
					}

					response := map[string]interface{}{
						"suggestions": allSuggestions[start:end],
						"total":       totalCount,
						"limit":       limit,
						"offset":      offset,
					}

					h.WriteJSONResponse(w, r, response, http.StatusOK)
					return
				}
			}
		}
	}

	// Если указана конкретная база данных или нет проекта, работаем как раньше
	if databasePath == "" {
		databasePath = h.currentNormalizedDBPath
	}

	// Открываем нужную БД
	var db *database.DB
	var err error
	if databasePath != "" && databasePath != h.currentNormalizedDBPath {
		db, err = database.NewDB(databasePath)
		if err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error opening database %s: %v", databasePath, err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
			return
		}
		defer db.Close()
	} else {
		if h.normalizedDB == nil {
			h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
			return
		}
		db = h.normalizedDB
	}

	// Параметры фильтрации
	filters := make(map[string]interface{})

	if priority := r.URL.Query().Get("priority"); priority != "" {
		filters["priority"] = priority
	}

	if autoApplyable := r.URL.Query().Get("auto_applyable"); autoApplyable == "true" {
		filters["auto_applyable"] = true
	}

	if applied := r.URL.Query().Get("applied"); applied == "false" {
		filters["applied"] = false
	}

	if suggestionType != "" && suggestionType != "all" {
		filters["suggestion_type"] = suggestionType
	}

	if searchParam != "" {
		filters["suggestion_search"] = searchParam
	}

	// Pagination
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	suggestions, total, err := db.GetSuggestions(filters, limit, offset)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting suggestions: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get suggestions: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"suggestions": suggestions,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleQualitySuggestionAction обрабатывает запросы к /api/quality/suggestions/{id}
func (h *QualityHandler) HandleQualitySuggestionAction(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/quality/suggestions/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		h.WriteJSONError(w, r, "Suggestion ID is required", http.StatusBadRequest)
		return
	}

	suggestionID, err := strconv.Atoi(pathParts[0])
	if err != nil {
		h.WriteJSONError(w, r, "Invalid suggestion ID", http.StatusBadRequest)
		return
	}

	// POST - применить предложение
	if r.Method == http.MethodPost {
		action := ""
		if len(pathParts) > 1 {
			action = pathParts[1]
		}

		if action == "apply" {
			if h.normalizedDB == nil {
				h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
				return
			}

			if err := h.normalizedDB.ApplySuggestion(suggestionID); err != nil {
				h.logFunc(types.LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Error applying suggestion %d: %v", suggestionID, err),
					Endpoint:  r.URL.Path,
				})
				h.WriteJSONError(w, r, fmt.Sprintf("Failed to apply suggestion: %v", err), http.StatusInternalServerError)
				return
			}

			h.WriteJSONResponse(w, r, map[string]interface{}{
				"success": true,
				"message": "Suggestion applied successfully",
			}, http.StatusOK)
			return
		}

		h.WriteJSONError(w, r, "Invalid action", http.StatusBadRequest)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleQualityDuplicates обрабатывает запросы к /api/quality/duplicates
func (h *QualityHandler) HandleQualityDuplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры из query
	databasePath := r.URL.Query().Get("database")
	projectParam := r.URL.Query().Get("project")

	// Если указан проект, собираем данные из всех баз проекта
	if projectParam != "" && h.getProjectDatabases != nil {
		parts := strings.Split(projectParam, ":")
		if len(parts) == 2 {
			projectID, err := strconv.Atoi(parts[1])
			if err == nil {
				projectDatabases, err := h.getProjectDatabases(projectID, true)
				if err == nil {
					// Собираем дубликаты из всех баз проекта
					allGroups := []map[string]interface{}{}
					totalCount := 0

					for _, projectDB := range projectDatabases {
						if !projectDB.IsActive {
							continue
						}

						db, err := database.NewDB(projectDB.FilePath)
						if err != nil {
							continue
						}

						onlyUnmerged := r.URL.Query().Get("unmerged") == "true"
						limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
						if limit <= 0 {
							limit = 1000 // Большой лимит для сбора всех данных
						}
						offset := 0

						groups, count, err := db.GetDuplicateGroups(onlyUnmerged, limit, offset)
						db.Close()

						if err == nil {
							// Обогащаем группы данными
							for _, group := range groups {
								enrichedGroup := map[string]interface{}{
									"id":                  group.ID,
									"group_hash":          group.GroupHash,
									"duplicate_type":      group.DuplicateType,
									"similarity_score":    group.SimilarityScore,
									"item_ids":            group.ItemIDs,
									"suggested_master_id": group.SuggestedMasterID,
									"confidence":          group.Confidence,
									"reason":              group.Reason,
									"merged":              group.Merged,
									"merged_at":           group.MergedAt,
									"created_at":          group.CreatedAt,
									"updated_at":          group.UpdatedAt,
									"item_count":          len(group.ItemIDs),
									"database_id":         projectDB.ID,
									"database_name":       projectDB.Name,
									"database_path":       projectDB.FilePath,
								}
								allGroups = append(allGroups, enrichedGroup)
							}
							totalCount += count
						}
					}

					// Применяем пагинацию к объединенному результату
					limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
					if limit <= 0 {
						limit = 50
					}
					offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
					if offset < 0 {
						offset = 0
					}

					start := offset
					end := offset + limit
					if start > len(allGroups) {
						start = len(allGroups)
					}
					if end > len(allGroups) {
						end = len(allGroups)
					}

					response := map[string]interface{}{
						"groups": allGroups[start:end],
						"total":  totalCount,
						"limit":  limit,
						"offset": offset,
					}

					h.WriteJSONResponse(w, r, response, http.StatusOK)
					return
				}
			}
		}
	}

	// Если указана конкретная база данных или нет проекта, работаем как раньше
	if databasePath == "" {
		databasePath = h.currentNormalizedDBPath
	}

	// Открываем нужную БД
	var db *database.DB
	var err error
	if databasePath != "" && databasePath != h.currentNormalizedDBPath {
		db, err = database.NewDB(databasePath)
		if err != nil {
			h.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error opening database %s: %v", databasePath, err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
			return
		}
		defer db.Close()
	} else {
		if h.normalizedDB == nil {
			h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
			return
		}
		db = h.normalizedDB
	}

	// Параметры фильтрации
	onlyUnmerged := r.URL.Query().Get("unmerged") == "true"

	// Pagination
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	groups, total, err := db.GetDuplicateGroups(onlyUnmerged, limit, offset)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting duplicate groups: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get duplicate groups: %v", err), http.StatusInternalServerError)
		return
	}

	// Обогащаем группы полными данными элементов
	enrichedGroups := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		enrichedGroup := map[string]interface{}{
			"id":                  group.ID,
			"group_hash":          group.GroupHash,
			"duplicate_type":      group.DuplicateType,
			"similarity_score":    group.SimilarityScore,
			"item_ids":            group.ItemIDs,
			"suggested_master_id": group.SuggestedMasterID,
			"confidence":          group.Confidence,
			"reason":              group.Reason,
			"merged":              group.Merged,
			"merged_at":           group.MergedAt,
			"created_at":          group.CreatedAt,
			"updated_at":          group.UpdatedAt,
			"item_count":          len(group.ItemIDs),
		}

		// Загружаем полные данные элементов
		if len(group.ItemIDs) > 0 {
			items := make([]map[string]interface{}, 0)
			// Формируем IN запрос для получения всех элементов за раз
			placeholders := make([]string, len(group.ItemIDs))
			args := make([]interface{}, len(group.ItemIDs))
			for j, id := range group.ItemIDs {
				placeholders[j] = "?"
				args[j] = id
			}

			// Пытаемся найти элементы в normalized_data
			query := fmt.Sprintf(`
				SELECT id, 
					COALESCE(code, '') as code, 
					COALESCE(normalized_name, '') as normalized_name, 
					COALESCE(category, '') as category, 
					COALESCE(kpved_code, '') as kpved_code, 
					COALESCE(processing_level, 'basic') as processing_level, 
					COALESCE(merged_count, 0) as merged_count,
					COALESCE(quality_score, 0.0) as quality_score
				FROM normalized_data
				WHERE id IN (%s)
			`, strings.Join(placeholders, ","))

			rows, err := db.GetDB().Query(query, args...)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var id, mergedCount int
					var code, normalizedName, category, kpvedCode, processingLevel sql.NullString
					var qualityScore sql.NullFloat64

					if err := rows.Scan(&id, &code, &normalizedName, &category, &kpvedCode, &processingLevel, &mergedCount, &qualityScore); err == nil {
						items = append(items, map[string]interface{}{
							"id":              id,
							"code":            getStringValue(code),
							"normalized_name": getStringValue(normalizedName),
							"category":        getStringValue(category),
							"kpved_code":      getStringValue(kpvedCode),
							"quality_score": func() float64 {
								if qualityScore.Valid {
									return qualityScore.Float64
								}
								return 0.0
							}(),
							"processing_level": getStringValue(processingLevel),
							"merged_count":     mergedCount,
						})
					}
				}
			} else {
				h.logFunc(types.LogEntry{
					Timestamp: time.Now(),
					Level:     "WARN",
					Message:   fmt.Sprintf("Could not find items for group %d: %v", group.ID, err),
					Endpoint:  r.URL.Path,
				})
			}
			enrichedGroup["items"] = items
		} else {
			enrichedGroup["items"] = []interface{}{}
		}

		enrichedGroups[i] = enrichedGroup
	}

	response := map[string]interface{}{
		"groups": enrichedGroups,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// getStringValue извлекает строковое значение из sql.NullString
func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// HandleQualityDuplicateAction обрабатывает запросы к /api/quality/duplicates/{id}
func (h *QualityHandler) HandleQualityDuplicateAction(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/quality/duplicates/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		h.WriteJSONError(w, r, "Duplicate group ID is required", http.StatusBadRequest)
		return
	}

	groupID, err := strconv.Atoi(pathParts[0])
	if err != nil {
		h.WriteJSONError(w, r, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// POST - действия с группой
	if r.Method == http.MethodPost {
		action := ""
		if len(pathParts) > 1 {
			action = pathParts[1]
		}

		if action == "merge" {
			if h.normalizedDB == nil {
				h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
				return
			}

			if err := h.normalizedDB.MarkDuplicateGroupMerged(groupID); err != nil {
				h.logFunc(types.LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Error merging duplicate group %d: %v", groupID, err),
					Endpoint:  r.URL.Path,
				})
				h.WriteJSONError(w, r, fmt.Sprintf("Failed to merge duplicate group: %v", err), http.StatusInternalServerError)
				return
			}

			h.WriteJSONResponse(w, r, map[string]interface{}{
				"success": true,
				"message": "Duplicate group merged successfully",
			}, http.StatusOK)
			return
		}

		h.WriteJSONError(w, r, "Invalid action", http.StatusBadRequest)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleQualityAssess обрабатывает запросы к /api/quality/assess
func (h *QualityHandler) HandleQualityAssess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		ItemID int `json:"item_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.ItemID <= 0 {
		h.WriteJSONError(w, r, "Item ID is required and must be positive", http.StatusBadRequest)
		return
	}

	if h.normalizedDB == nil {
		h.WriteJSONError(w, r, "Normalized database is not available", http.StatusInternalServerError)
		return
	}

	// Получаем оценку качества
	assessment, err := h.normalizedDB.GetQualityAssessment(reqBody.ItemID)
	if err != nil {
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting quality assessment for item %d: %v", reqBody.ItemID, err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get quality assessment: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, assessment, http.StatusOK)
}

// HandleQualityAnalyze обрабатывает запросы к /api/quality/analyze
func (h *QualityHandler) HandleQualityAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var reqBody struct {
		Database   string `json:"database"`
		Table      string `json:"table"`
		CodeColumn string `json:"code_column"`
		NameColumn string `json:"name_column"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем, не выполняется ли уже анализ
	h.qualityAnalysisMutex.Lock()
	if h.qualityAnalysisRunning {
		h.qualityAnalysisMutex.Unlock()
		h.WriteJSONError(w, r, "Analysis is already running", http.StatusConflict)
		return
	}
	h.qualityAnalysisRunning = true
	h.qualityAnalysisStatus = QualityAnalysisStatus{
		IsRunning:        true,
		Progress:         0,
		Processed:        0,
		Total:            0,
		CurrentStep:      "initializing",
		DuplicatesFound:  0,
		ViolationsFound:  0,
		SuggestionsFound: 0,
	}
	h.qualityAnalysisMutex.Unlock()

	// Определяем колонки по умолчанию если не указаны
	codeColumn := reqBody.CodeColumn
	nameColumn := reqBody.NameColumn

	if codeColumn == "" {
		switch reqBody.Table {
		case "normalized_data":
			codeColumn = "code"
		case "nomenclature_items":
			codeColumn = "nomenclature_code"
		case "catalog_items":
			codeColumn = "code"
		default:
			codeColumn = "code"
		}
	}

	if nameColumn == "" {
		switch reqBody.Table {
		case "normalized_data":
			nameColumn = "normalized_name"
		case "nomenclature_items":
			nameColumn = "nomenclature_name"
		case "catalog_items":
			nameColumn = "name"
		default:
			nameColumn = "name"
		}
	}

	// Открываем базу данных
	db, err := database.NewDB(reqBody.Database)
	if err != nil {
		h.qualityAnalysisMutex.Lock()
		h.qualityAnalysisRunning = false
		h.qualityAnalysisStatus.Error = err.Error()
		h.qualityAnalysisMutex.Unlock()
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error opening database: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}

	// Запускаем анализ в фоновой горутине
	go h.runQualityAnalysis(db, reqBody.Table, codeColumn, nameColumn)

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Quality analysis started",
		"table":   reqBody.Table,
	}, http.StatusOK)
}

// HandleQualityAnalyzeStatus обрабатывает запросы к /api/quality/analyze/status
func (h *QualityHandler) HandleQualityAnalyzeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.qualityAnalysisMutex.RLock()
	status := h.qualityAnalysisStatus
	h.qualityAnalysisMutex.RUnlock()

	h.WriteJSONResponse(w, r, status, http.StatusOK)
}

// runQualityAnalysis выполняет анализ качества в фоновом режиме
func (h *QualityHandler) runQualityAnalysis(db *database.DB, tableName, codeColumn, nameColumn string) {
	defer db.Close()
	defer func() {
		h.qualityAnalysisMutex.Lock()
		h.qualityAnalysisRunning = false
		if h.qualityAnalysisStatus.Error == "" {
			h.qualityAnalysisStatus.CurrentStep = "completed"
			h.qualityAnalysisStatus.Progress = 100
		}
		h.qualityAnalysisMutex.Unlock()
	}()

	analyzer := quality.NewTableAnalyzer(db)
	batchSize := 1000

	// 1. Анализ дубликатов
	h.qualityAnalysisMutex.Lock()
	h.qualityAnalysisStatus.CurrentStep = "duplicates"
	h.qualityAnalysisMutex.Unlock()

	duplicatesCount, err := analyzer.AnalyzeTableForDuplicates(
		tableName, codeColumn, nameColumn, batchSize,
		func(processed, total int) {
			h.qualityAnalysisMutex.Lock()
			h.qualityAnalysisStatus.Processed = processed
			h.qualityAnalysisStatus.Total = total
			if total > 0 {
				h.qualityAnalysisStatus.Progress = float64(processed) / float64(total) * 33.33
			}
			h.qualityAnalysisMutex.Unlock()
		},
	)

	if err != nil {
		h.qualityAnalysisMutex.Lock()
		h.qualityAnalysisStatus.Error = fmt.Sprintf("Duplicate analysis failed: %v", err)
		h.qualityAnalysisMutex.Unlock()
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Duplicate analysis failed: %v", err),
		})
		return
	}

	h.qualityAnalysisMutex.Lock()
	h.qualityAnalysisStatus.DuplicatesFound = duplicatesCount
	h.qualityAnalysisMutex.Unlock()

	// 2. Анализ нарушений
	h.qualityAnalysisMutex.Lock()
	h.qualityAnalysisStatus.CurrentStep = "violations"
	h.qualityAnalysisStatus.Processed = 0
	h.qualityAnalysisStatus.Total = 0
	h.qualityAnalysisMutex.Unlock()

	violationsCount, err := analyzer.AnalyzeTableForViolations(
		tableName, codeColumn, nameColumn, batchSize,
		func(processed, total int) {
			h.qualityAnalysisMutex.Lock()
			h.qualityAnalysisStatus.Processed = processed
			h.qualityAnalysisStatus.Total = total
			if total > 0 {
				h.qualityAnalysisStatus.Progress = 33.33 + float64(processed)/float64(total)*33.33
			}
			h.qualityAnalysisMutex.Unlock()
		},
	)

	if err != nil {
		h.qualityAnalysisMutex.Lock()
		h.qualityAnalysisStatus.Error = fmt.Sprintf("Violations analysis failed: %v", err)
		h.qualityAnalysisMutex.Unlock()
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Violations analysis failed: %v", err),
		})
		return
	}

	h.qualityAnalysisMutex.Lock()
	h.qualityAnalysisStatus.ViolationsFound = violationsCount
	h.qualityAnalysisMutex.Unlock()

	// 3. Анализ предложений
	h.qualityAnalysisMutex.Lock()
	h.qualityAnalysisStatus.CurrentStep = "suggestions"
	h.qualityAnalysisStatus.Processed = 0
	h.qualityAnalysisStatus.Total = 0
	h.qualityAnalysisMutex.Unlock()

	suggestionsCount, err := analyzer.AnalyzeTableForSuggestions(
		tableName, codeColumn, nameColumn, batchSize,
		func(processed, total int) {
			h.qualityAnalysisMutex.Lock()
			h.qualityAnalysisStatus.Processed = processed
			h.qualityAnalysisStatus.Total = total
			if total > 0 {
				h.qualityAnalysisStatus.Progress = 66.66 + float64(processed)/float64(total)*33.34
			}
			h.qualityAnalysisMutex.Unlock()
		},
	)

	if err != nil {
		h.qualityAnalysisMutex.Lock()
		h.qualityAnalysisStatus.Error = fmt.Sprintf("Suggestions analysis failed: %v", err)
		h.qualityAnalysisMutex.Unlock()
		h.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Suggestions analysis failed: %v", err),
		})
		return
	}

	h.qualityAnalysisMutex.Lock()
	h.qualityAnalysisStatus.SuggestionsFound = suggestionsCount
	h.qualityAnalysisMutex.Unlock()

	h.logFunc(types.LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Quality analysis completed: duplicates=%d, violations=%d, suggestions=%d", duplicatesCount, violationsCount, suggestionsCount),
	})
}
