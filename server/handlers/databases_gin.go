package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apperrors "httpserver/server/errors"
	"httpserver/server/services"
)

// DatabaseHandler обработчик для работы с базами данных
type DatabaseHandler struct {
	databaseService *services.DatabaseService
	baseHandler     *BaseHandler
}

// NewDatabaseHandler создает новый обработчик для работы с базами данных
func NewDatabaseHandler(
	databaseService *services.DatabaseService,
	baseHandler *BaseHandler,
) *DatabaseHandler {
	return &DatabaseHandler{
		databaseService: databaseService,
		baseHandler:     baseHandler,
	}
}

// DatabaseListResponse структура ответа для списка баз данных
type DatabaseListResponse struct {
	Databases         []interface{}            `json:"databases"`
	Total             int                      `json:"total"`
	AggregatedStats   map[string]interface{}   `json:"aggregated_stats,omitempty"`
}

// DatabaseInfoResponse структура ответа для информации о базе данных
type DatabaseInfoResponse struct {
	Info interface{} `json:"info"`
}

// ErrorResponse структура ошибки
type ErrorResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// HandleDatabasesListGin обработчик списка баз данных для Gin
// @Summary Получить список всех баз данных
// @Description Возвращает список всех доступных баз данных в системе
// @Tags databases
// @Accept json
// @Produce json
// @Success 200 {object} DatabaseListResponse "Список баз данных"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/list [get]
func (h *DatabaseHandler) HandleDatabasesListGin(c *gin.Context) {
	databases, err := h.databaseService.ListDatabases()
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить список баз данных")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	// Получаем текущий путь базы данных для определения is_current
	currentDBPath := h.databaseService.GetCurrentDBPath()

	// Получаем статистику для всех баз данных одним batch-запросом (оптимизация N+1)
	var statsMap map[int]map[string]interface{}
	if h.databaseService.GetDB() != nil && len(databases) > 0 {
		databaseIDs := make([]int, 0, len(databases))
		for _, db := range databases {
			databaseIDs = append(databaseIDs, db.ID)
		}
		
		var err error
		statsMap, err = h.databaseService.GetDB().GetUploadStatsByDatabaseIDs(databaseIDs)
		if err != nil {
			// Игнорируем ошибки получения статистики - это не критично для отображения списка БД
			statsMap = make(map[int]map[string]interface{})
		}
	}

	// Форматируем данные для фронтенда
	databasesInterface := make([]interface{}, 0, len(databases))
	for _, db := range databases {
		dbInfo := make(map[string]interface{})
		
		// Базовые поля
		dbInfo["name"] = db.Name
		dbInfo["path"] = db.FilePath
		dbInfo["size"] = db.FileSize
		
		// Дата последнего изменения
		if db.LastUsedAt != nil {
			dbInfo["modified_at"] = db.LastUsedAt.Format(time.RFC3339)
		} else {
			dbInfo["modified_at"] = db.UpdatedAt.Format(time.RFC3339)
		}
		
		// Проверяем, является ли это текущей базой данных
		if currentDBPath != "" && db.FilePath == currentDBPath {
			dbInfo["is_current"] = true
		}
		
		// Добавляем статистику из batch-запроса
		if statsMap != nil {
			if stats, exists := statsMap[db.ID]; exists && stats != nil {
				dbInfo["stats"] = stats
			}
		}
		
		// Получаем информацию о файле, если он существует
		if db.FilePath != "" {
			if info, err := os.Stat(db.FilePath); err == nil {
				dbInfo["size"] = info.Size()
				dbInfo["modified_at"] = info.ModTime().Format(time.RFC3339)
			}
		}
		
		databasesInterface = append(databasesInterface, dbInfo)
	}

	// Получаем агрегированную статистику по всем базам данных
	var aggregatedStats map[string]interface{}
	if h.databaseService.GetDB() != nil {
		stats, err := h.databaseService.GetAggregatedUploadStats()
		if err == nil && stats != nil {
			aggregatedStats = stats
		}
		// Игнорируем ошибки получения агрегированной статистики - это не критично
	}

	SendJSONResponse(c, http.StatusOK, DatabaseListResponse{
		Databases:       databasesInterface,
		Total:           len(databasesInterface),
		AggregatedStats: aggregatedStats,
	})
}

// HandleDatabaseInfoGin обработчик информации о базе данных для Gin
// @Summary Получить информацию о базе данных
// @Description Возвращает детальную информацию о базе данных
// @Tags databases
// @Accept json
// @Produce json
// @Success 200 {object} DatabaseInfoResponse "Информация о базе данных"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/database/info [get]
func (h *DatabaseHandler) HandleDatabaseInfoGin(c *gin.Context) {
	info, err := h.databaseService.GetDatabaseInfo()
	if err != nil {
		// Вместо возврата ошибки, возвращаем базовую информацию
		// Это позволяет фронтенду работать даже если БД недоступна
		info = map[string]interface{}{
			"current_db_path":            "",
			"current_normalized_db_path": "",
			"name":                       "",
			"path":                       "",
			"size":                       int64(0),
			"modified_at":                "",
			"status":                     "disconnected",
			"stats":                      make(map[string]interface{}),
		}
		// Логируем ошибку, но не прерываем выполнение
		// appErr := apperrors.WrapError(err, "не удалось получить информацию о базе данных")
		// SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		// return
	}

	SendJSONResponse(c, http.StatusOK, DatabaseInfoResponse{
		Info: info,
	})
}

// HandleFindDatabaseGin обработчик поиска базы данных для Gin
// @Summary Найти базы данных
// @Description Ищет базы данных по запросу
// @Tags databases
// @Accept json
// @Produce json
// @Param q query string true "Поисковый запрос"
// @Success 200 {object} DatabaseListResponse "Найденные базы данных"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/find [get]
func (h *DatabaseHandler) HandleFindDatabaseGin(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		SendJSONError(c, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	databases, err := h.databaseService.FindDatabase(query)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось найти базы данных")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	// Получаем статистику для всех найденных баз данных одним batch-запросом (оптимизация N+1)
	var statsMap map[int]map[string]interface{}
	if h.databaseService.GetDB() != nil && len(databases) > 0 {
		databaseIDs := make([]int, 0, len(databases))
		for _, db := range databases {
			databaseIDs = append(databaseIDs, db.ID)
		}

		var statsErr error
		statsMap, statsErr = h.databaseService.GetDB().GetUploadStatsByDatabaseIDs(databaseIDs)
		if statsErr != nil {
			// Игнорируем ошибки получения статистики - это не критично для отображения списка БД
			statsMap = make(map[int]map[string]interface{})
		}
	}

	// Форматируем данные с добавлением статистики
	databasesInterface := make([]interface{}, 0, len(databases))
	for _, db := range databases {
		dbInfo := map[string]interface{}{
			"id":                db.ID,
			"client_project_id": db.ClientProjectID,
			"name":              db.Name,
			"file_path":         db.FilePath,
			"description":       db.Description,
			"is_active":         db.IsActive,
			"file_size":         db.FileSize,
			"created_at":        db.CreatedAt.Format(time.RFC3339),
			"updated_at":        db.UpdatedAt.Format(time.RFC3339),
		}

		if db.LastUsedAt != nil {
			dbInfo["last_used_at"] = db.LastUsedAt.Format(time.RFC3339)
		}

		// Добавляем статистику из batch-запроса
		if statsMap != nil {
			if stats, exists := statsMap[db.ID]; exists && stats != nil {
				dbInfo["stats"] = stats
			}
		}

		databasesInterface = append(databasesInterface, dbInfo)
	}

	// Получаем агрегированную статистику по всем найденным базам данных
	var aggregatedStats map[string]interface{}
	if h.databaseService.GetDB() != nil && len(databases) > 0 {
		// Для поиска используем статистику только по найденным БД
		// Можно было бы использовать GetAggregatedUploadStats, но это даст статистику по всем БД
		// Для поиска это не критично, поэтому оставляем пустым или используем общую статистику
		stats, err := h.databaseService.GetAggregatedUploadStats()
		if err == nil && stats != nil {
			aggregatedStats = stats
		}
	}

	SendJSONResponse(c, http.StatusOK, DatabaseListResponse{
		Databases:       databasesInterface,
		Total:           len(databasesInterface),
		AggregatedStats: aggregatedStats,
	})
}

// HandlePendingDatabasesGin обработчик ожидающих баз данных для Gin
// @Summary Получить список ожидающих баз данных
// @Description Возвращает список баз данных со статусом pending или другим указанным статусом
// @Tags databases
// @Accept json
// @Produce json
// @Param status query string false "Фильтр по статусу (по умолчанию 'pending')"
// @Success 200 {array} object "Список ожидающих баз данных"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/pending [get]
func (h *DatabaseHandler) HandlePendingDatabasesGin(c *gin.Context) {
	// Получаем параметр status из query string
	statusFilter := c.Query("status")
	if statusFilter == "" {
		statusFilter = "pending" // По умолчанию фильтруем по статусу "pending"
	}

	databases, err := h.databaseService.GetPendingDatabases(statusFilter)
	if err != nil {
		appErr := apperrors.WrapError(err, "не удалось получить список ожидающих баз данных")
		SendJSONError(c, appErr.StatusCode(), appErr.UserMessage())
		return
	}

	// Преобразуем в формат для JSON ответа
	databasesResponse := make([]map[string]interface{}, 0, len(databases))
	for _, db := range databases {
		dbInfo := map[string]interface{}{
			"id":               db.ID,
			"file_path":        db.FilePath,
			"file_name":        db.FileName,
			"file_size":        db.FileSize,
			"detected_at":      db.DetectedAt.Format(time.RFC3339),
			"indexing_status":  db.IndexingStatus,
			"moved_to_uploads": db.MovedToUploads,
			"original_path":    db.OriginalPath,
		}

		if db.IndexingStartedAt != nil {
			dbInfo["indexing_started_at"] = db.IndexingStartedAt.Format(time.RFC3339)
		}
		if db.IndexingCompletedAt != nil {
			dbInfo["indexing_completed_at"] = db.IndexingCompletedAt.Format(time.RFC3339)
		}
		if db.ErrorMessage != "" {
			dbInfo["error_message"] = db.ErrorMessage
		}
		if db.ClientID != nil {
			dbInfo["client_id"] = *db.ClientID
		}
		if db.ProjectID != nil {
			dbInfo["project_id"] = *db.ProjectID
		}

		databasesResponse = append(databasesResponse, dbInfo)
	}

	SendJSONResponse(c, http.StatusOK, databasesResponse)
}

// HandleListBackups обрабатывает запросы к /api/backups
func (h *DatabaseHandler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	backupDir := "data/backups"
	backups, err := h.databaseService.ListBackups(backupDir)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, "Failed to list backups", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"backups": backups,
		"total":   len(backups),
	}
	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleDownloadBackup обрабатывает запросы к /api/databases/backups/{filename}
func (h *DatabaseHandler) HandleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	// Извлекаем имя файла из URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/databases/backups/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		h.baseHandler.WriteJSONError(w, r, "Backup filename is required", http.StatusBadRequest)
		return
	}

	filename := pathParts[0]
	
	// Безопасность: проверяем, что имя файла не содержит переходов
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		h.baseHandler.WriteJSONError(w, r, "Invalid backup filename", http.StatusBadRequest)
		return
	}

	backupDir := "data/backups"
	backupPath := filepath.Join(backupDir, filename)

	// Проверяем, что файл существует
	if _, err := os.Stat(backupPath); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Backup file not found", http.StatusNotFound)
		return
	}

	// Отправляем файл
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	http.ServeFile(w, r, backupPath)
}

// HandleRestoreBackup обрабатывает запросы к /api/backups/restore
func (h *DatabaseHandler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.WriteJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BackupFile string `json:"backup_file"`
		TargetPath string `json:"target_path,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BackupFile == "" {
		h.baseHandler.WriteJSONError(w, r, "backup_file is required", http.StatusBadRequest)
		return
	}

	// Безопасность: проверяем, что имя файла не содержит переходов
	if strings.Contains(req.BackupFile, "..") || strings.Contains(req.BackupFile, "/") || strings.Contains(req.BackupFile, "\\") {
		h.baseHandler.WriteJSONError(w, r, "Invalid backup filename", http.StatusBadRequest)
		return
	}

	backupDir := "data/backups"
	backupPath := filepath.Join(backupDir, req.BackupFile)

	// Определяем целевой путь
	targetPath := req.TargetPath
	if targetPath == "" {
		// Если целевой путь не указан, используем текущую БД
		targetPath = h.databaseService.GetCurrentDBPath()
		if targetPath == "" {
			h.baseHandler.WriteJSONError(w, r, "Target path is required", http.StatusBadRequest)
			return
		}
	}

	// Восстанавливаем резервную копию
	err := h.databaseService.RestoreBackup(backupPath, targetPath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, "Failed to restore backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"backup_path": backupPath,
		"target_path": targetPath,
	}
	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}