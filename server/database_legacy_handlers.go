package server

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/server/middleware"
)

// Legacy database handlers - перемещены из server.go для рефакторинга
// TODO: Заменить на новые handlers из internal/api/handlers/

// handleDatabaseV1Routes обрабатывает маршруты /api/v1/databases/{id}
func (s *Server) handleDatabaseV1Routes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/databases/")

	if path == "" {
		s.handleHTTPError(w, r, NewValidationError("ID базы данных обязателен", nil))
		return
	}

	// Парсим database_id (может быть строкой или числом)
	databaseID, err := ValidateIDPathParam(path, "database_id")
	if err != nil {
		LogError(r.Context(), err, "Invalid database ID", "path", path)
		s.handleHTTPError(w, r, NewValidationError("неверный ID базы данных", err))
		return
	}

	if r.Method != http.MethodGet {
		s.handleHTTPError(w, r, NewValidationError("Метод не разрешен", nil))
		return
	}

	// Получаем информацию о базе данных
	if s.serviceDB == nil {
		LogError(r.Context(), nil, "Service database not available")
		s.handleHTTPError(w, r, NewInternalError("Service database not available", fmt.Errorf("serviceDB is nil")))
		return
	}

	dbInfo, err := s.serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get database", "database_id", databaseID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить информацию о базе данных", err))
		return
	}

	if dbInfo == nil {
		LogWarn(r.Context(), "Database not found", "database_id", databaseID)
		s.handleHTTPError(w, r, NewNotFoundError("База данных не найдена", fmt.Errorf("database with ID %d not found", databaseID)))
		return
	}

	// Получаем информацию о базе данных с проектом и клиентом одним запросом
	dbWithProject, err := s.serviceDB.GetProjectDatabaseWithClient(databaseID)
	if err != nil {
		s.writeErrorResponse(w, "Failed to get database info", err)
		return
	}

	project := dbWithProject.Project
	client := project.Client
	dbInfo = &dbWithProject.ProjectDatabase

	// Формируем XML ответ
	response := DatabaseInfoResponse{
		XMLName:      xml.Name{Local: "database_info"},
		DatabaseID:   strconv.Itoa(databaseID),
		DatabaseName: dbInfo.Name,
		ProjectID:    strconv.Itoa(project.ID),
		ProjectName:  project.Name,
		ClientID:     strconv.Itoa(client.ID),
		ClientName:   client.Name,
		Status:       "success",
		Message:      "Database information retrieved successfully",
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}

// DatabaseInfoResponse структура для ответа с информацией о базе данных
type DatabaseInfoResponse struct {
	XMLName      xml.Name `xml:"database_info"`
	DatabaseID   string   `xml:"database_id"`
	DatabaseName string   `xml:"database_name"`
	ProjectID    string   `xml:"project_id,omitempty"`
	ProjectName  string   `xml:"project_name"`
	ClientID     string   `xml:"client_id,omitempty"`
	ClientName   string   `xml:"client_name"`
	Status       string   `xml:"status"`
	Message      string   `xml:"message,omitempty"`
	Timestamp    string   `xml:"timestamp"`
}

// GetLogChannel возвращает канал для получения логов
func (s *Server) GetLogChannel() <-chan LogEntry {
	return s.logChan
}

// GetCircuitBreakerState возвращает состояние Circuit Breaker
func (s *Server) GetCircuitBreakerState() map[string]interface{} {
	if s.normalizer == nil || s.normalizer.GetAINormalizer() == nil {
		return map[string]interface{}{
			"enabled":       false,
			"state":         "unknown",
			"can_proceed":   false,
			"failure_count": 0,
		}
	}

	aiNormalizer := s.normalizer.GetAINormalizer()
	if aiNormalizer == nil {
		return map[string]interface{}{
			"enabled":       false,
			"state":         "unknown",
			"can_proceed":   false,
			"failure_count": 0,
		}
	}

	// Получаем реальное состояние Circuit Breaker через AINormalizer
	cbState := aiNormalizer.GetCircuitBreakerState()
	return cbState
}

// GetBatchProcessorStats возвращает статистику батчевой обработки
func (s *Server) GetBatchProcessorStats() map[string]interface{} {
	if s.normalizer == nil || s.normalizer.GetAINormalizer() == nil {
		return map[string]interface{}{
			"enabled":             false,
			"queue_size":          0,
			"total_batches":       0,
			"avg_items_per_batch": 0.0,
			"api_calls_saved":     0,
		}
	}

	aiNormalizer := s.normalizer.GetAINormalizer()
	if aiNormalizer == nil {
		return map[string]interface{}{
			"enabled":             false,
			"queue_size":          0,
			"total_batches":       0,
			"avg_items_per_batch": 0.0,
			"api_calls_saved":     0,
		}
	}

	// Получаем реальную статистику от BatchProcessor
	return aiNormalizer.GetBatchProcessorStats()
}

// GetCheckpointStatus возвращает статус checkpoint system
func (s *Server) GetCheckpointStatus() map[string]interface{} {
	if s.normalizer == nil {
		return map[string]interface{}{
			"enabled":          false,
			"active":           false,
			"processed_count":  0,
			"total_count":      0,
			"progress_percent": 0.0,
		}
	}

	// Получаем реальный статус от Normalizer
	return s.normalizer.GetCheckpointStatus()
}

// CollectMetricsSnapshot собирает текущий снимок метрик производительности
func (s *Server) CollectMetricsSnapshot() *database.PerformanceMetricsSnapshot {
	// Рассчитываем uptime
	uptime := time.Since(s.startTime).Seconds()

	// Получаем метрики от компонентов
	cbState := s.GetCircuitBreakerState()
	batchStats := s.GetBatchProcessorStats()
	checkpointStatus := s.GetCheckpointStatus()

	// Собираем AI и cache метрики
	aiSuccessRate := 0.0
	cacheHitRate := 0.0
	throughput := 0.0

	if s.normalizer != nil && s.normalizer.GetAINormalizer() != nil {
		statsCollector := s.normalizer.GetAINormalizer().GetStatsCollector()
		if statsCollector != nil {
			perfMetrics := statsCollector.GetMetrics()
			if perfMetrics.TotalAIRequests > 0 {
				aiSuccessRate = float64(perfMetrics.SuccessfulAIRequest) / float64(perfMetrics.TotalAIRequests)
			}
			if perfMetrics.TotalNormalized > 0 && uptime > 0 {
				throughput = float64(perfMetrics.TotalNormalized) / uptime
			}
		}

		cacheStats := s.normalizer.GetAINormalizer().GetCacheStats()
		cacheHitRate = cacheStats.HitRate
	}

	// Получаем checkpoint progress из checkpointStatus
	checkpointProgress := 0.0
	if progress, ok := checkpointStatus["progress_percent"].(float64); ok {
		checkpointProgress = progress
	}

	// Формируем детальные метрики в JSON
	detailedMetrics := map[string]interface{}{
		"uptime_seconds":        int(uptime),
		"throughput":            throughput,
		"ai_success_rate":       aiSuccessRate,
		"cache_hit_rate":        cacheHitRate,
		"circuit_breaker_state": cbState["state"].(string),
		"checkpoint_progress":   checkpointProgress,
	}

	// Добавляем batch queue size если доступен
	var batchQueueSize int
	if queueSize, ok := batchStats["queue_size"].(int); ok {
		batchQueueSize = queueSize
		detailedMetrics["batch_queue_size"] = queueSize
	}

	// Добавляем детальные метрики из statsCollector если доступны
	if s.normalizer != nil && s.normalizer.GetAINormalizer() != nil {
		statsCollector := s.normalizer.GetAINormalizer().GetStatsCollector()
		if statsCollector != nil {
			perfMetrics := statsCollector.GetMetrics()
			detailedMetrics["ai_requests"] = map[string]interface{}{
				"total":      perfMetrics.TotalAIRequests,
				"successful": perfMetrics.SuccessfulAIRequest,
				"failed":     perfMetrics.TotalAIRequests - perfMetrics.SuccessfulAIRequest,
			}
			detailedMetrics["normalized_count"] = perfMetrics.TotalNormalized
		}

		cacheStats := s.normalizer.GetAINormalizer().GetCacheStats()
		detailedMetrics["cache"] = map[string]interface{}{
			"hits":     cacheStats.Hits,
			"misses":   cacheStats.Misses,
			"hit_rate": cacheStats.HitRate,
		}
	}

	// Сериализуем детальные метрики в JSON
	metricDataJSON, err := json.Marshal(detailedMetrics)
	if err != nil {
		log.Printf("Failed to marshal detailed metrics: %v", err)
		metricDataJSON = []byte("{}")
	}

	// Создаем snapshot с детальными метриками
	snapshot := &database.PerformanceMetricsSnapshot{
		Timestamp:           time.Now(),
		MetricType:          "all", // Общие метрики
		MetricData:          string(metricDataJSON),
		UptimeSeconds:       int(uptime),
		Throughput:          throughput,
		AISuccessRate:       aiSuccessRate,
		CacheHitRate:        cacheHitRate,
		CircuitBreakerState: cbState["state"].(string),
		CheckpointProgress:  checkpointProgress,
		BatchQueueSize:      batchQueueSize,
	}

	return snapshot
}

// writeJSONResponse записывает JSON ответ
func (s *Server) writeJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	middleware.WriteJSONResponse(w, r, data, statusCode)
}

// writeJSONError записывает JSON ошибку
func (s *Server) writeJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	middleware.WriteJSONError(w, r, message, statusCode)
}

// handleHTTPError обрабатывает ошибку используя новую систему AppError
func (s *Server) handleListUploads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.handleHTTPError(w, r, NewValidationError("Метод не разрешен", nil))
		return
	}

	uploads, err := s.db.GetAllUploads()
	if err != nil {
		LogError(r.Context(), err, "Failed to get uploads")
		s.handleHTTPError(w, r, NewInternalError("не удалось получить список выгрузок", err))
		return
	}

	items := make([]UploadListItem, len(uploads))
	for i, upload := range uploads {
		items[i] = UploadListItem{
			UploadUUID:     upload.UploadUUID,
			StartedAt:      upload.StartedAt,
			CompletedAt:    upload.CompletedAt,
			Status:         upload.Status,
			Version1C:      upload.Version1C,
			ConfigName:     upload.ConfigName,
			TotalConstants: upload.TotalConstants,
			TotalCatalogs:  upload.TotalCatalogs,
			TotalItems:     upload.TotalItems,
		}
	}

	LogInfo(r.Context(), "List uploads requested", "count", len(items))

	s.writeJSONResponse(w, r, map[string]interface{}{
		"uploads": items,
		"total":   len(items),
	}, http.StatusOK)
}

// handleUploadRoutes обрабатывает маршруты с UUID выгрузки
func (s *Server) handleUploadRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	uuid := parts[0]

	// Проверяем существование выгрузки
	upload, err := s.db.GetUploadByUUID(uuid)
	if err != nil {
		LogError(r.Context(), err, "Upload not found", "uuid", uuid)
		s.handleHTTPError(w, r, NewNotFoundError("Выгрузка не найдена", err))
		return
	}

	// Обрабатываем подмаршруты
	if len(parts) == 1 {
		// GET /api/uploads/{uuid} - детали выгрузки
		s.handleGetUpload(w, r, upload)
	} else if len(parts) == 2 {
		switch parts[1] {
		case "data":
			// GET /api/uploads/{uuid}/data - получение данных
			s.handleGetUploadData(w, r, upload)
		case "stream":
			// GET /api/uploads/{uuid}/stream - потоковая отправка
			s.handleStreamUploadData(w, r, upload)
		case "verify":
			// POST /api/uploads/{uuid}/verify - проверка передачи
			s.handleVerifyUpload(w, r, upload)
		default:
			http.NotFound(w, r)
		}
	} else {
		http.NotFound(w, r)
	}
}

// handleNormalizedListUploads обрабатывает запрос списка выгрузок из нормализованной БД
func (s *Server) handleNormalizedListUploads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploads, err := s.normalizedDB.GetAllUploads()
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get normalized uploads: %v", err), http.StatusInternalServerError)
		return
	}

	items := make([]UploadListItem, len(uploads))
	for i, upload := range uploads {
		items[i] = UploadListItem{
			UploadUUID:     upload.UploadUUID,
			StartedAt:      upload.StartedAt,
			CompletedAt:    upload.CompletedAt,
			Status:         upload.Status,
			Version1C:      upload.Version1C,
			ConfigName:     upload.ConfigName,
			TotalConstants: upload.TotalConstants,
			TotalCatalogs:  upload.TotalCatalogs,
			TotalItems:     upload.TotalItems,
		}
	}

	LogInfo(r.Context(), "List normalized uploads requested", "count", len(items))

	s.writeJSONResponse(w, r, map[string]interface{}{
		"uploads": items,
		"total":   len(items),
	}, http.StatusOK)
}

// handleNormalizedUploadRoutes обрабатывает маршруты с UUID выгрузки из нормализованной БД
func (s *Server) handleNormalizedUploadRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/normalized/uploads/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	uuid := parts[0]

	// Проверяем существование выгрузки в нормализованной БД
	upload, err := s.normalizedDB.GetUploadByUUID(uuid)
	if err != nil {
		s.writeJSONError(w, r, "Normalized upload not found", http.StatusNotFound)
		return
	}

	// Обрабатываем подмаршруты
	if len(parts) == 1 {
		// GET /api/normalized/uploads/{uuid} - детали выгрузки
		s.handleGetUploadNormalized(w, r, upload)
	} else if len(parts) == 2 {
		switch parts[1] {
		case "data":
			// GET /api/normalized/uploads/{uuid}/data - получение данных
			s.handleGetUploadDataNormalized(w, r, upload)
		case "stream":
			// GET /api/normalized/uploads/{uuid}/stream - потоковая отправка
			s.handleStreamUploadDataNormalized(w, r, upload)
		case "verify":
			// POST /api/normalized/uploads/{uuid}/verify - проверка передачи
			s.handleVerifyUploadNormalized(w, r, upload)
		default:
			http.NotFound(w, r)
		}
	} else {
		http.NotFound(w, r)
	}
}

// handleGetUpload обрабатывает запрос детальной информации о выгрузке
func (s *Server) handlePendingDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	databases, err := s.serviceDB.GetPendingDatabases(statusFilter)
	if err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"databases": databases,
		"total":     len(databases),
	}, http.StatusOK)
}

// handlePendingDatabaseRoutes обрабатывает запросы к /api/databases/pending/{id}
func (s *Server) handlePendingDatabaseRoutes(w http.ResponseWriter, r *http.Request) {
	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/databases/pending/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		s.writeJSONError(w, r, "Invalid request path", http.StatusBadRequest)
		return
	}

	id, err := ValidateIDPathParam(parts[0], "database_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid database ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		pendingDB, err := s.serviceDB.GetPendingDatabase(id)
		if err != nil {
			s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if pendingDB == nil {
			s.writeJSONError(w, r, "Pending database not found", http.StatusNotFound)
			return
		}
		s.writeJSONResponse(w, r, pendingDB, http.StatusOK)

	case http.MethodDelete:
		if err := s.serviceDB.DeletePendingDatabase(id); err != nil {
			s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSONResponse(w, r, map[string]interface{}{"success": true}, http.StatusOK)

	case http.MethodPost:
		// Обработка действий: index, bind, cleanup
		if len(parts) < 2 {
			s.writeJSONError(w, r, "Action required", http.StatusBadRequest)
			return
		}

		action := parts[1]
		switch action {
		case "index":
			s.handleStartIndexing(w, r, id)
		case "bind":
			s.handleBindPendingDatabase(w, r, id)
		default:
			s.writeJSONError(w, r, "Unknown action", http.StatusBadRequest)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStartIndexing запускает индексацию pending database
func (s *Server) handleStartIndexing(w http.ResponseWriter, r *http.Request, id int) {
	pendingDB, err := s.serviceDB.GetPendingDatabase(id)
	if err != nil || pendingDB == nil {
		s.writeJSONError(w, r, "Pending database not found", http.StatusNotFound)
		return
	}

	if pendingDB.IndexingStatus == "indexing" {
		s.writeJSONError(w, r, "Indexing already in progress", http.StatusBadRequest)
		return
	}

	// Обновляем статус на indexing
	if err := s.serviceDB.UpdatePendingDatabaseStatus(id, "indexing", ""); err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Запускаем индексацию в горутине
	go func() {
		// Здесь можно добавить логику индексации БД
		// Пока просто помечаем как completed
		time.Sleep(1 * time.Second) // Симуляция индексации
		s.serviceDB.UpdatePendingDatabaseStatus(id, "completed", "")
	}()

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Indexing started",
	}, http.StatusOK)
}

// handleBindPendingDatabase привязывает pending database к проекту
func (s *Server) handleBindPendingDatabase(w http.ResponseWriter, r *http.Request, id int) {
	var req struct {
		ClientID   int    `json:"client_id"`
		ProjectID  int    `json:"project_id"`
		CustomPath string `json:"custom_path"` // Опциональный путь, если не указан - перемещаем в uploads
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClientID == 0 || req.ProjectID == 0 {
		s.writeJSONError(w, r, "client_id and project_id are required", http.StatusBadRequest)
		return
	}

	pendingDB, err := s.serviceDB.GetPendingDatabase(id)
	if err != nil || pendingDB == nil {
		s.writeJSONError(w, r, "Pending database not found", http.StatusNotFound)
		return
	}

	// Проверяем проект
	project, err := s.serviceDB.GetClientProject(req.ProjectID)
	if err != nil {
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != req.ClientID {
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Определяем новый путь к файлу
	var newFilePath string
	var movedToUploads bool

	if req.CustomPath != "" {
		// Используем указанный путь
		newFilePath = req.CustomPath
		movedToUploads = false
	} else {
		// Перемещаем в data/uploads/
		uploadsDir, err := EnsureUploadsDirectory(".")
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to create uploads directory: %v", err), http.StatusInternalServerError)
			return
		}

		newFilePath, err = MoveDatabaseToUploads(pendingDB.FilePath, uploadsDir)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to move file: %v", err), http.StatusInternalServerError)
			return
		}
		movedToUploads = true
	}

	// Обновляем pending database
	if err := s.serviceDB.BindPendingDatabaseToProject(id, req.ClientID, req.ProjectID, newFilePath, movedToUploads); err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаем запись в project_databases
	fileInfo, _ := os.Stat(newFilePath)
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	projectDB, err := s.serviceDB.CreateProjectDatabase(req.ProjectID, pendingDB.FileName, newFilePath, "Автоматически добавлена из pending databases", fileSize)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to create project database: %v", err), http.StatusInternalServerError)
		return
	}

	// Удаляем из pending databases
	s.serviceDB.DeletePendingDatabase(id)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":          true,
		"message":          "Database bound to project",
		"database":         projectDB,
		"moved_to_uploads": movedToUploads,
	}, http.StatusOK)
}

// handleCleanupPendingDatabases очищает старые pending databases
func (s *Server) handleCleanupPendingDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	// Получаем количество дней из параметра запроса (по умолчанию 30)
	daysOld, err := ValidateIntParam(r, "days", 30, 1, 365)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	deletedCount, err := s.serviceDB.CleanupOldPendingDatabases(daysOld)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to cleanup pending databases: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":       true,
		"message":       fmt.Sprintf("Cleaned up %d old pending databases (older than %d days)", deletedCount, daysOld),
		"deleted_count": deletedCount,
		"days_old":      daysOld,
	}, http.StatusOK)
}

// handleScanDatabases запускает сканирование файлов
func (s *Server) handleScanDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Paths []string `json:"paths"` // Опционально: пути для сканирования
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Если тело пустое, используем дефолтные пути
		req.Paths = []string{".", "data/uploads"}
	}

	if len(req.Paths) == 0 {
		req.Paths = []string{".", "data/uploads"}
	}

	foundFiles, err := ScanForDatabaseFiles(req.Paths, s.serviceDB)
	if err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":     true,
		"found_files": len(foundFiles),
		"files":       foundFiles,
	}, http.StatusOK)
}

// DatabaseFileInfo информация о файле базы данных
type DatabaseFileInfo struct {
	Path            string    `json:"path"`
	Name            string    `json:"name"`
	Size            int64     `json:"size"`
	ModifiedAt      time.Time `json:"modified_at"`
	Type            string    `json:"type"` // "main", "service", "uploaded", "other"
	IsProtected     bool      `json:"is_protected"`
	LinkedToProject bool      `json:"linked_to_project"`
	ClientID        *int      `json:"client_id,omitempty"`
	ProjectID       *int      `json:"project_id,omitempty"`
	ProjectName     *string   `json:"project_name,omitempty"`
	DatabaseID      *int      `json:"database_id,omitempty"`
}

// handleDatabasesFiles возвращает список всех найденных .db файлов с группировкой по типам
func (s *Server) handleDatabasesFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Защищенные файлы, которые нельзя удалять
	protectedFiles := map[string]bool{
		"service.db":         true,
		"1c_data.db":         true,
		"data.db":            true,
		"normalized_data.db": true,
	}

	var allFiles []DatabaseFileInfo

	// 1. Сканируем основные директории
	scanPaths := []string{
		".",
		"data",
		"data/uploads",
		"/app",
		"/app/data",
		"/app/data/uploads",
	}

	fileMap := make(map[string]bool) // Для дедупликации

	for _, scanPath := range scanPaths {
		if _, err := os.Stat(scanPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			// Другие ошибки тоже пропускаем
			continue
		}

		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Пропускаем ошибки доступа
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(path), ".db") {
				return nil
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil
			}

			// Дедупликация
			if fileMap[absPath] {
				return nil
			}
			fileMap[absPath] = true

			fileName := filepath.Base(absPath)
			isProtected := protectedFiles[fileName]

			// Определяем тип файла
			fileType := "other"
			if isProtected {
				if fileName == "service.db" {
					fileType = "service"
				} else {
					fileType = "main"
				}
			} else if strings.Contains(absPath, "uploads") || strings.Contains(absPath, "data/uploads") {
				fileType = "uploaded"
			} else if strings.Contains(absPath, "data") {
				fileType = "main"
			}

			fileInfo := DatabaseFileInfo{
				Path:            absPath,
				Name:            fileName,
				Size:            info.Size(),
				ModifiedAt:      info.ModTime(),
				Type:            fileType,
				IsProtected:     isProtected,
				LinkedToProject: false,
			}

			// Проверяем, связан ли файл с проектом
			if s.serviceDB != nil {
				clientID, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(absPath)
				if err == nil && clientID > 0 && projectID > 0 {
					fileInfo.LinkedToProject = true
					fileInfo.ClientID = &clientID
					fileInfo.ProjectID = &projectID

					// Получаем информацию о проекте
					project, err := s.serviceDB.GetClientProject(projectID)
					if err == nil {
						fileInfo.ProjectName = &project.Name
					}

					// Получаем ID базы данных в проекте
					projectDB, err := s.serviceDB.GetProjectDatabaseByPath(projectID, absPath)
					if err == nil && projectDB != nil {
						fileInfo.DatabaseID = &projectDB.ID
					}
				}
			}

			allFiles = append(allFiles, fileInfo)
			return nil
		})

		if err != nil {
			log.Printf("Error scanning path %s: %v", scanPath, err)
		}
	}

	// Группируем по типам
	grouped := map[string][]DatabaseFileInfo{
		"main":     []DatabaseFileInfo{},
		"service":  []DatabaseFileInfo{},
		"uploaded": []DatabaseFileInfo{},
		"other":    []DatabaseFileInfo{},
	}

	for _, file := range allFiles {
		grouped[file.Type] = append(grouped[file.Type], file)
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"total":   len(allFiles),
		"files":   allFiles,
		"grouped": grouped,
		"summary": map[string]int{
			"main":     len(grouped["main"]),
			"service":  len(grouped["service"]),
			"uploaded": len(grouped["uploaded"]),
			"other":    len(grouped["other"]),
		},
	}, http.StatusOK)
}

// handleCounterpartyDuplicates обрабатывает запросы на получение дублей контрагентов
func (s *Server) handleBulkDeleteDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Paths   []string `json:"paths"`   // Пути к файлам
		IDs     []int    `json:"ids"`     // ID баз данных в проектах
		Confirm bool     `json:"confirm"` // Подтверждение удаления
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Confirm {
		s.writeJSONError(w, r, "confirm=true is required for deletion", http.StatusBadRequest)
		return
	}

	// Защищенные файлы
	protectedFiles := map[string]bool{
		"service.db":         true,
		"1c_data.db":         true,
		"data.db":            true,
		"normalized_data.db": true,
	}

	type DeleteResult struct {
		Path    string `json:"path"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	results := []DeleteResult{}

	// Обрабатываем пути
	for _, path := range req.Paths {
		result := DeleteResult{Path: path}

		absPath, err := filepath.Abs(path)
		if err != nil {
			result.Error = fmt.Sprintf("Invalid path: %v", err)
			results = append(results, result)
			continue
		}

		fileName := filepath.Base(absPath)
		if protectedFiles[fileName] {
			result.Error = "File is protected and cannot be deleted"
			results = append(results, result)
			continue
		}

		// Проверяем, что файл существует
		if _, err := os.Stat(absPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				result.Error = "File does not exist"
			} else {
				result.Error = fmt.Sprintf("Error checking file: %v", err)
			}
			results = append(results, result)
			continue
		}

		// Удаляем записи из project_databases, если они существуют
		if s.serviceDB != nil {
			_, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(absPath)
			if err == nil && projectID > 0 {
				// Находим и удаляем запись в project_databases
				projectDB, err := s.serviceDB.GetProjectDatabaseByPath(projectID, absPath)
				if err == nil && projectDB != nil {
					if err := s.serviceDB.DeleteProjectDatabase(projectDB.ID); err != nil {
						log.Printf("Warning: Failed to delete database record for %s: %v", absPath, err)
					} else {
						log.Printf("Deleted database record (ID: %d) for file: %s", projectDB.ID, absPath)
					}
				}
			}
		}

		// Удаляем физический файл
		if err := os.Remove(absPath); err != nil {
			result.Error = fmt.Sprintf("Failed to delete file: %v", err)
			results = append(results, result)
			continue
		}

		result.Success = true
		log.Printf("Deleted database file: %s", absPath)
		results = append(results, result)
	}

	// Обрабатываем ID (получаем путь к файлу и удаляем только файл)
	for _, id := range req.IDs {
		result := DeleteResult{Path: fmt.Sprintf("ID:%d", id)}

		if s.serviceDB == nil {
			result.Error = "Service database not available"
			results = append(results, result)
			continue
		}

		projectDB, err := s.serviceDB.GetProjectDatabase(id)
		if err != nil || projectDB == nil {
			result.Error = "Database not found"
			results = append(results, result)
			continue
		}

		absPath, err := filepath.Abs(projectDB.FilePath)
		if err != nil {
			result.Error = fmt.Sprintf("Invalid path: %v", err)
			results = append(results, result)
			continue
		}

		result.Path = absPath

		fileName := filepath.Base(absPath)
		if protectedFiles[fileName] {
			result.Error = "File is protected and cannot be deleted"
			results = append(results, result)
			continue
		}

		// Проверяем, что файл существует
		if _, err := os.Stat(absPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				result.Error = "File does not exist"
			} else {
				result.Error = fmt.Sprintf("Error checking file: %v", err)
			}
			results = append(results, result)
			continue
		}

		// Удаляем запись из project_databases
		if err := s.serviceDB.DeleteProjectDatabase(id); err != nil {
			result.Error = fmt.Sprintf("Failed to delete database record: %v", err)
			results = append(results, result)
			continue
		}
		log.Printf("Deleted database record (ID: %d)", id)

		// Удаляем физический файл
		if err := os.Remove(absPath); err != nil {
			result.Error = fmt.Sprintf("Failed to delete file: %v", err)
			results = append(results, result)
			continue
		}

		result.Success = true
		log.Printf("Deleted database file and record: %s (ID: %d)", absPath, id)
		results = append(results, result)
	}

	successCount := 0
	failedCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":       failedCount == 0,
		"total":         len(results),
		"success_count": successCount,
		"failed_count":  failedCount,
		"results":       results,
	}, http.StatusOK)
}

// handleBackupDatabases создает бэкап баз данных
func (s *Server) handleBackupDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IncludeMain    bool     `json:"include_main"`
		IncludeUploads bool     `json:"include_uploads"`
		IncludeService bool     `json:"include_service"`
		SelectedFiles  []string `json:"selected_files"`
		Format         string   `json:"format"` // "zip", "copy", "both"
	}

	// Значения по умолчанию
	req.IncludeMain = true
	req.IncludeUploads = true
	req.IncludeService = false
	req.Format = "both" // По умолчанию создаем и ZIP, и копии

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Если тело пустое, используем значения по умолчанию
	}

	// Нормализуем формат
	if req.Format != "zip" && req.Format != "copy" && req.Format != "both" {
		req.Format = "both"
	}

	// Создаем директорию для бэкапов, если не существует
	backupDir := "data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to create backup directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Генерируем timestamp для имени бэкапа
	timestamp := time.Now().Format("20060102_150405")

	var zipFile *os.File
	var zipWriter *zip.Writer
	var backupFileName string
	var backupPath string
	var filesCopyDir string

	// Создаем ZIP архив, если нужно
	if req.Format == "zip" || req.Format == "both" {
		backupFileName = fmt.Sprintf("backup_%s.zip", timestamp)
		backupPath = filepath.Join(backupDir, backupFileName)

		var err error
		zipFile, err = os.Create(backupPath)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to create backup file: %v", err), http.StatusInternalServerError)
			return
		}
		defer zipFile.Close()

		zipWriter = zip.NewWriter(zipFile)
		defer zipWriter.Close()
	}

	// Создаем директорию для копий файлов, если нужно
	if req.Format == "copy" || req.Format == "both" {
		filesCopyDir = filepath.Join(backupDir, "files", timestamp)
		if err := os.MkdirAll(filesCopyDir, 0755); err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to create files backup directory: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Собираем файлы для бэкапа
	filesToBackup := []string{}

	if len(req.SelectedFiles) > 0 {
		// Выборочный бэкап
		for _, path := range req.SelectedFiles {
			absPath, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			if _, err := os.Stat(absPath); err == nil {
				filesToBackup = append(filesToBackup, absPath)
			}
		}
	} else {
		// Полный бэкап согласно параметрам
		if req.IncludeMain {
			// Основные БД из корня и data/
			mainPatterns := []string{"*.db", "data/*.db"}
			for _, pattern := range mainPatterns {
				if matches, err := filepath.Glob(pattern); err == nil {
					for _, match := range matches {
						fileName := filepath.Base(match)
						if fileName != "service.db" {
							filesToBackup = append(filesToBackup, match)
						}
					}
				}
			}
		}

		if req.IncludeUploads {
			// Загруженные БД
			uploadPatterns := []string{"data/uploads/*.db"}
			for _, pattern := range uploadPatterns {
				if matches, err := filepath.Glob(pattern); err == nil {
					filesToBackup = append(filesToBackup, matches...)
				}
			}
		}

		if req.IncludeService {
			// Сервисная БД
			servicePaths := []string{"data/service.db", "service.db"}
			for _, path := range servicePaths {
				if _, err := os.Stat(path); err == nil {
					filesToBackup = append(filesToBackup, path)
					break
				}
			}
		}
	}

	// Создаем структуру папок в архиве
	addedFiles := 0
	totalSize := int64(0)

	for _, filePath := range filesToBackup {
		// Определяем путь в архиве
		var archivePath string
		fileName := filepath.Base(filePath)

		if strings.Contains(filePath, "uploads") {
			archivePath = filepath.Join("uploads", fileName)
		} else if fileName == "service.db" {
			archivePath = filepath.Join("service", fileName)
		} else {
			archivePath = filepath.Join("main", fileName)
		}

		// Открываем файл для чтения
		sourceFile, err := os.Open(filePath)
		if err != nil {
			s.logError(fmt.Sprintf("Failed to open file %s for backup: %v", filePath, err), r.URL.Path)
			continue
		}

		fileInfo, err := sourceFile.Stat()
		if err != nil {
			sourceFile.Close()
			continue
		}

		// Если нужно добавить в ZIP архив
		if zipWriter != nil {
			// Создаем запись в архиве
			archiveFile, err := zipWriter.Create(archivePath)
			if err != nil {
				s.logError(fmt.Sprintf("Failed to create archive entry for %s: %v", filePath, err), r.URL.Path)
				sourceFile.Close()
				continue
			}

			// Копируем содержимое файла в архив
			if _, err := io.Copy(archiveFile, sourceFile); err != nil {
				s.logError(fmt.Sprintf("Failed to copy file %s to archive: %v", filePath, err), r.URL.Path)
				sourceFile.Close()
				continue
			}
		}

		// Если нужно скопировать файлы
		if filesCopyDir != "" {
			destPath := filepath.Join(filesCopyDir, archivePath)
			destDir := filepath.Dir(destPath)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				s.logError(fmt.Sprintf("Failed to create directory %s: %v", destDir, err), r.URL.Path)
				sourceFile.Close()
				continue
			}

			// Сбрасываем позицию файла для копирования
			if _, err := sourceFile.Seek(0, 0); err != nil {
				log.Printf("Failed to seek file %s: %v", filePath, err)
				sourceFile.Close()
				continue
			}

			destFile, err := os.Create(destPath)
			if err != nil {
				log.Printf("Failed to create destination file %s: %v", destPath, err)
				sourceFile.Close()
				continue
			}

			if _, err := io.Copy(destFile, sourceFile); err != nil {
				log.Printf("Failed to copy file %s to %s: %v", filePath, destPath, err)
				sourceFile.Close()
				destFile.Close()
				continue
			}
			destFile.Close()
		}

		sourceFile.Close()
		addedFiles++
		totalSize += fileInfo.Size()
	}

	// Закрываем архив, если он был создан
	if zipWriter != nil {
		if err := zipWriter.Close(); err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to finalize backup: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Проверяем, что были добавлены файлы
	if addedFiles == 0 {
		s.writeJSONError(w, r, "No files were found to backup", http.StatusBadRequest)
		return
	}

	backupInfo := map[string]interface{}{
		"files_count": addedFiles,
		"total_size":  totalSize,
		"created_at":  time.Now().Format(time.RFC3339),
		"format":      req.Format,
	}

	// Добавляем информацию о ZIP архиве, если он был создан
	if backupFileName != "" {
		backupInfo["backup_file"] = backupFileName
		backupInfo["backup_path"] = backupPath
	}

	// Добавляем информацию о директории с копиями файлов, если она была создана
	if filesCopyDir != "" {
		backupInfo["files_copy_dir"] = filesCopyDir
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Backup created successfully with %d files", addedFiles),
		"backup":  backupInfo,
	}, http.StatusOK)
}
func (s *Server) handleFindProjectByDatabase(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Panic in handleFindProjectByDatabase: %v", rec)
			s.writeJSONError(w, r, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePathRaw := r.URL.Query().Get("file_path")
	if filePathRaw == "" {
		log.Printf("Error: file_path parameter is missing in find-project request")
		s.writeJSONError(w, r, "file_path parameter is required", http.StatusBadRequest)
		return
	}

	// Декодируем URL-кодированный путь
	filePath, err := url.QueryUnescape(filePathRaw)
	if err != nil {
		log.Printf("Warning: Failed to decode file_path %s, using as-is: %v", filePathRaw, err)
		filePath = filePathRaw
	}

	// Нормализуем путь (приводим к единому формату)
	filePath = filepath.Clean(filePath)
	// Приводим к формату с прямыми слешами для Windows
	filePath = filepath.ToSlash(filePath)

	log.Printf("Received find-project request for file_path: %s (decoded from: %s)", filePath, filePathRaw)

	if s.serviceDB == nil {
		log.Printf("Error: Service database not available for find-project request with file_path: %s", filePath)
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	// Пробуем найти с нормализованным путем (метод уже поддерживает разные форматы)
	clientID, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(filePath)
	if err != nil {
		log.Printf("Error finding client and project for file_path %s: %v", filePath, err)
		s.writeJSONError(w, r, err.Error(), http.StatusNotFound)
		return
	}

	// Получаем db_id по пути базы данных
	var dbID *int
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		log.Printf("Warning: Failed to get project databases for project_id %d: %v", projectID, err)
		// Продолжаем выполнение, даже если не удалось получить список БД
	} else {
		for _, db := range databases {
			if db.FilePath == filePath {
				dbID = &db.ID
				break
			}
		}
	}

	log.Printf("Successfully found project for file_path %s: client_id=%d, project_id=%d, db_id=%v",
		filePath, clientID, projectID, dbID)

	response := map[string]interface{}{
		"client_id":  clientID,
		"project_id": projectID,
	}
	if dbID != nil {
		response["db_id"] = *dbID
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetProjectPipelineStats получает статистику этапов обработки для проекта
// Версия без clientID и projectID - использует параметры из запроса
// ПРИМЕЧАНИЕ: Реализация с параметрами находится в server.go как handleGetProjectPipelineStatsWithParams
func (s *Server) handleGetProjectPipelineStats(w http.ResponseWriter, r *http.Request) {
	// Извлекаем project_id из query параметров
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

	// Извлекаем client_id из query параметров (опционально)
	clientID := 0
	if clientIDStr := r.URL.Query().Get("client_id"); clientIDStr != "" {
		clientID, err = ValidateIDParam(r, "client_id")
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid client_id: %s", err.Error()), http.StatusBadRequest)
			return
		}
	}

	// Вызываем версию с параметрами из server.go
	// Используем прямое обращение к методу через рефлексию или создаем вспомогательный метод
	// Для простоты используем прямую реализацию здесь
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if clientID > 0 && project.ClientID != clientID {
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Проверяем тип проекта
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

// handleListBackups обрабатывает запросы к /api/backups
func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	// Используем DatabaseHandler, если доступен
	if s.databaseHandler != nil {
		s.databaseHandler.HandleListBackups(w, r)
		return
	}

	// Fallback: простая реализация
	backupDir := "data/backups"
	if _, err := os.Stat(backupDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeJSONResponse(w, r, map[string]interface{}{
				"backups": []interface{}{},
				"total":   0,
			}, http.StatusOK)
			return
		}
		// Другие ошибки - возвращаем пустой список
		s.writeJSONResponse(w, r, map[string]interface{}{
			"backups": []interface{}{},
			"total":   0,
		}, http.StatusOK)
		return
	}

	var backups []map[string]interface{}
	err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".db") {
			return nil
		}
		absPath, _ := filepath.Abs(path)
		backups = append(backups, map[string]interface{}{
			"filename":   info.Name(),
			"path":       absPath,
			"size":       info.Size(),
			"created_at": info.ModTime().Format(time.RFC3339),
		})
		return nil
	})

	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to list backups: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"backups": backups,
		"total":   len(backups),
	}, http.StatusOK)
}

// handleDownloadBackup обрабатывает запросы к /api/databases/backups/{filename}
func (s *Server) handleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	// Используем DatabaseHandler, если доступен
	if s.databaseHandler != nil {
		s.databaseHandler.HandleDownloadBackup(w, r)
		return
	}

	// Fallback: простая реализация
	s.writeJSONError(w, r, "Database handler not initialized", http.StatusServiceUnavailable)
}

// handleRestoreBackup обрабатывает запросы к /api/backups/restore
func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	// Используем DatabaseHandler, если доступен
	if s.databaseHandler != nil {
		s.databaseHandler.HandleRestoreBackup(w, r)
		return
	}

	// Fallback: простая реализация
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BackupFile string `json:"backup_file"`
		TargetPath string `json:"target_path,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BackupFile == "" {
		s.writeJSONError(w, r, "backup_file is required", http.StatusBadRequest)
		return
	}

	// Безопасность: проверяем, что путь не содержит переходов
	if strings.Contains(req.BackupFile, "..") || strings.Contains(req.BackupFile, "/") || strings.Contains(req.BackupFile, "\\") {
		s.writeJSONError(w, r, "Invalid backup filename", http.StatusBadRequest)
		return
	}

	backupDir := "data/backups"
	backupPath := filepath.Join(backupDir, req.BackupFile)

	if _, err := os.Stat(backupPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeJSONError(w, r, "Backup file not found", http.StatusNotFound)
			return
		}
		s.writeJSONError(w, r, fmt.Sprintf("Error checking backup file: %v", err), http.StatusInternalServerError)
		return
	}

	// Простая реализация восстановления (копирование файла)
	targetPath := req.TargetPath
	if targetPath == "" {
		// Используем путь основной БД
		targetPath = s.currentDBPath
	}

	if targetPath == "" {
		s.writeJSONError(w, r, "Target path not specified", http.StatusBadRequest)
		return
	}

	// Копируем файл
	sourceFile, err := os.Open(backupPath)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to open backup file: %v", err), http.StatusInternalServerError)
		return
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to create target file: %v", err), http.StatusInternalServerError)
		return
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to copy backup file: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":     true,
		"backup_file": req.BackupFile,
	}, http.StatusOK)
}
