package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"httpserver/database"
)

// handleDatabaseInfo возвращает информацию о текущей базе данных
func (s *Server) handleDatabaseInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.dbMutex.RLock()
	defer s.dbMutex.RUnlock()

	// Получаем статистику из БД
	// Делаем это безопасно, чтобы не ломать основной функционал при ошибках
	var stats map[string]interface{}
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Игнорируем панику - это не критично для получения общей информации
				log.Printf("Panic при получении статистики БД: %v", r)
			}
		}()
		stats, err = s.db.GetStats()
	}()

	if err != nil {
		log.Printf("Ошибка получения статистики БД: %v", err)
		// Не возвращаем ошибку, а продолжаем с пустой статистикой
		stats = make(map[string]interface{})
	}

	// Получаем информацию о файле
	fileInfo, err := os.Stat(s.currentDBPath)
	var fileSize int64
	var modTime time.Time
	if err == nil {
		fileSize = fileInfo.Size()
		modTime = fileInfo.ModTime()
	}

	response := map[string]interface{}{
		"name":        filepath.Base(s.currentDBPath),
		"path":        s.currentDBPath,
		"size":        fileSize,
		"modified_at": modTime,
		"stats":       stats,
		"status":      "connected",
	}

	// Получаем статистику из uploads для текущей БД, если можем определить database_id
	// Делаем это безопасно, чтобы не ломать основной функционал
	if s.serviceDB != nil && s.currentDBPath != "" {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Игнорируем панику - это не критично
					log.Printf("Panic при получении статистики uploads: %v", r)
				}
			}()

			_, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(s.currentDBPath)
			if err == nil && projectID > 0 {
				dbInfo, err := s.serviceDB.GetProjectDatabaseByPath(projectID, s.currentDBPath)
				if err == nil && dbInfo != nil {
					uploadStats, err := s.db.GetUploadStatsByDatabaseID(dbInfo.ID)
					if err == nil && uploadStats != nil {
						response["upload_stats"] = uploadStats
					}
				}
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDatabasesList возвращает список доступных баз данных
func (s *Server) handleDatabasesList(w http.ResponseWriter, r *http.Request) {
	// Обработка паники для предотвращения краша сервера
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic in handleDatabasesList: %v", err)
			s.writeJSONError(w, r, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Сканируем несколько директорий на наличие .db файлов
	var allFiles []string

	// 1. Текущая директория
	if files, err := filepath.Glob("*.db"); err == nil {
		allFiles = append(allFiles, files...)
	}

	// 2. Директория /app/data (для Docker)
	if dataFiles, err := filepath.Glob("data/*.db"); err == nil {
		allFiles = append(allFiles, dataFiles...)
	}

	// 3. Директория /app/data (абсолютный путь для Docker)
	if absDataFiles, err := filepath.Glob("/app/data/*.db"); err == nil {
		allFiles = append(allFiles, absDataFiles...)
	}

	// 4. Директория /app (абсолютный путь для Docker)
	absAppFiles, err := filepath.Glob("/app/*.db")
	if err == nil {
		// Фильтруем файлы и добавляем их
		filteredFiles := make([]string, 0, len(absAppFiles))
		for _, file := range absAppFiles {
			// Пропускаем service.db из корня, так как он должен быть в data/
			if filepath.Base(file) != "service.db" || filepath.Dir(file) == "/app/data" {
				filteredFiles = append(filteredFiles, file)
			}
		}
		allFiles = append(allFiles, filteredFiles...)
	}

	// Убираем дубликаты по абсолютному пути
	fileMap := make(map[string]string) // absPath -> original path
	uniqueFiles := []string{}
	for _, file := range allFiles {
		absPath, err := filepath.Abs(file)
		if err != nil {
			absPath = file
		}
		// Нормализуем путь (убираем лишние слеши и т.д.)
		absPath = filepath.Clean(absPath)

		// Если файл уже есть, выбираем более короткий путь
		if existingPath, exists := fileMap[absPath]; exists {
			// Предпочитаем путь из /app/data/ или более короткий
			if len(file) < len(existingPath) || strings.Contains(file, "data/") {
				fileMap[absPath] = file
			}
		} else {
			fileMap[absPath] = file
		}
	}

	// Обновляем список уникальных файлов с выбранными путями
	uniqueFiles = []string{}
	for _, path := range fileMap {
		uniqueFiles = append(uniqueFiles, path)
	}

	databases := []map[string]interface{}{}
	s.dbMutex.RLock()
	currentDB := s.currentDBPath
	s.dbMutex.RUnlock()

	for _, file := range uniqueFiles {
		// Обрабатываем каждый файл с обработкой ошибок
		func() {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Error processing database file %s: %v", file, err)
				}
			}()

			fileInfo, err := os.Stat(file)
			if err != nil {
				log.Printf("Error stat file %s: %v", file, err)
				return
			}

			isCurrent := file == currentDB

			// Определяем тип базы данных
			dbType := "unknown"
			if detectedType, err := database.DetectDatabaseType(file); err == nil {
				dbType = detectedType
			} else {
				log.Printf("Ошибка определения типа БД %s: %v", file, err)
			}

			// Получаем метаданные из serviceDB
			var metadata *database.DatabaseMetadata
			if s.serviceDB != nil {
				metadata, _ = s.serviceDB.GetDatabaseMetadata(file)
			}

			// Если метаданных нет, создаем их с информацией о конфигурации 1С
			if metadata == nil && s.serviceDB != nil {
				if err := UpdateDatabaseMetadataWithConfig(s.serviceDB, file, dbType); err != nil {
					log.Printf("Error upserting metadata for %s: %v", file, err)
				} else {
					metadata, _ = s.serviceDB.GetDatabaseMetadata(file)
				}
			}

			dbInfo := map[string]interface{}{
				"name":       filepath.Base(file),
				"path":       file,
				"size":       fileInfo.Size(),
				"modifiedAt": fileInfo.ModTime(),
				"isCurrent":  isCurrent, // Используем camelCase для фронтенда
				"type":       dbType,
			}

			// Добавляем информацию из метаданных
			if metadata != nil {
				dbInfo["firstSeenAt"] = metadata.FirstSeenAt
				if metadata.LastAnalyzedAt != nil {
					dbInfo["lastAnalyzedAt"] = metadata.LastAnalyzedAt
				}
				dbInfo["description"] = metadata.Description
			}

			// Получаем базовую статистику (количество таблиц)
			if tableStats, err := database.GetTableStats(file); err == nil {
				var totalRows int64
				for _, stat := range tableStats {
					totalRows += stat.RowCount
				}
				dbInfo["tableCount"] = len(tableStats)
				dbInfo["totalRows"] = totalRows
			} else {
				log.Printf("Error getting table stats for %s: %v", file, err)
			}

			databases = append(databases, dbInfo)
		}()
	}

	response := map[string]interface{}{
		"databases": databases,
		"current":   currentDB,
		"total":     len(databases),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logErrorf("Error encoding JSON response: %v", err)
		s.writeJSONError(w, r, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleDatabaseAnalytics возвращает детальную аналитику базы данных
func (s *Server) handleDatabaseAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Приоритет 1: путь из query параметра
	dbPath := r.URL.Query().Get("path")

	// Приоритет 2: имя из пути URL (для обратной совместимости)
	if dbPath == "" {
		path := r.URL.Path
		dbPath = strings.TrimPrefix(path, "/api/databases/analytics/")
		// Убираем завершающий слеш, если есть
		dbPath = strings.TrimSuffix(dbPath, "/")
	}

	if dbPath == "" {
		s.writeJSONError(w, r, "Database path is required", http.StatusBadRequest)
		return
	}

	// Декодируем путь, если он был закодирован
	if decodedPath, err := url.QueryUnescape(dbPath); err == nil {
		dbPath = decodedPath
	}

	// Нормализуем путь используя filepath для правильной обработки на всех ОС
	// Сначала пробуем получить абсолютный путь
	if absPath, err := filepath.Abs(dbPath); err == nil {
		dbPath = filepath.Clean(absPath)
	} else {
		// Если не удалось получить абсолютный путь, просто очищаем
		dbPath = filepath.Clean(dbPath)
	}

	// Логируем запрос для отладки
	log.Printf("Запрос аналитики для БД: %s", dbPath)

	// Проверяем существование файла
	_, err := os.Stat(dbPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			wd, _ := os.Getwd()
			log.Printf("Файл БД не найден: %s (текущая директория: %s)", dbPath, wd)
			s.writeJSONError(w, r, fmt.Sprintf("Database file not found: %s", dbPath), http.StatusNotFound)
			return
		}
		s.writeJSONError(w, r, fmt.Sprintf("Error checking database file: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем аналитику
	analytics, err := database.GetDatabaseAnalytics(dbPath)
	if err != nil {
		log.Printf("Ошибка получения аналитики БД %s: %v", dbPath, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get analytics: %v", err), http.StatusInternalServerError)
		return
	}

	// Обновляем историю изменений
	if s.serviceDB != nil {
		var totalRows int64
		for _, stat := range analytics.TableStats {
			totalRows += stat.RowCount
		}
		database.UpdateDatabaseHistory(s.serviceDB, dbPath, analytics.TotalSize, totalRows)
	}

	// Получаем статистику из uploads, если доступна основная БД и serviceDB
	if s.db != nil && s.serviceDB != nil {
		// Находим databaseID по пути к файлу
		_, projectID, err := s.serviceDB.FindClientAndProjectByDatabasePath(dbPath)
		if err == nil {
			// Получаем базу данных проекта по пути
			dbInfo, err := s.serviceDB.GetProjectDatabaseByPath(projectID, dbPath)
			if err == nil && dbInfo != nil {
				// Получаем статистику из uploads
				stats, err := s.db.GetUploadStatsByDatabaseID(dbInfo.ID)
				if err == nil && stats != nil {
					// Добавляем статистику в ответ
					analyticsMap := make(map[string]interface{})
					analyticsMap["file_path"] = analytics.FilePath
					analyticsMap["database_type"] = analytics.DatabaseType
					analyticsMap["total_size"] = analytics.TotalSize
					analyticsMap["total_size_mb"] = analytics.TotalSizeMB
					analyticsMap["table_count"] = analytics.TableCount
					analyticsMap["total_rows"] = analytics.TotalRows
					analyticsMap["table_stats"] = analytics.TableStats
					analyticsMap["top_tables"] = analytics.TopTables
					analyticsMap["analyzed_at"] = analytics.AnalyzedAt.Format(time.RFC3339)
					analyticsMap["upload_stats"] = stats

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(analyticsMap)
					return
				}
			}
		}
		// Игнорируем ошибки получения статистики - это не критично для аналитики
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// handleDatabaseHistory возвращает историю изменений базы данных
func (s *Server) handleDatabaseHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	// Извлекаем имя базы данных из пути
	path := r.URL.Path
	dbName := strings.TrimPrefix(path, "/api/databases/history/")
	if dbName == "" {
		s.writeJSONError(w, r, "Database name is required", http.StatusBadRequest)
		return
	}

	// Получаем историю
	history, err := database.GetDatabaseHistory(s.serviceDB, dbName)
	if err != nil {
		log.Printf("Ошибка получения истории БД %s: %v", dbName, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"database": dbName,
		"history":  history,
		"count":    len(history),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFindDatabase ищет database_id по client_id и project_id
func (s *Server) handleFindDatabase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Валидация обязательных параметров
	clientID, err := ValidateIDParam(r, "client_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("client_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	// Проверяем существование проекта и принадлежность клиенту
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Получаем список баз данных проекта (сначала активные)
	databases, err := s.serviceDB.GetProjectDatabases(projectID, true)
	if err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Если активных нет, получаем все
	if len(databases) == 0 {
		databases, err = s.serviceDB.GetProjectDatabases(projectID, false)
		if err != nil {
			s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if len(databases) == 0 {
		s.writeJSONError(w, r, "No databases found for this project", http.StatusNotFound)
		return
	}

	// Возвращаем первую базу данных
	db := databases[0]
	response := map[string]interface{}{
		"database_id": db.ID,
		"name":        db.Name,
		"exists":      true,
		"is_active":   db.IsActive,
		"file_path":   db.FilePath,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleDatabaseSwitch переключает текущую базу данных
func (s *Server) handleDatabaseSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем, что нормализация не запущена
	s.normalizerMutex.RLock()
	isRunning := s.normalizerRunning
	s.normalizerMutex.RUnlock()

	if isRunning {
		http.Error(w, "Cannot switch database while normalization is running", http.StatusBadRequest)
		return
	}

	// Читаем запрос
	var request struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Path == "" {
		http.Error(w, "Database path is required", http.StatusBadRequest)
		return
	}

	// Проверяем, что файл существует
	if _, err := os.Stat(request.Path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "Database file not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Error checking database file: %v", err), http.StatusInternalServerError)
		return
	}

	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	// Закрываем текущую БД
	if err := s.db.Close(); err != nil {
		log.Printf("Ошибка закрытия текущей БД: %v", err)
		http.Error(w, "Failed to close current database", http.StatusInternalServerError)
		return
	}

	// Открываем новую БД
	newDB, err := database.NewDB(request.Path)
	if err != nil {
		log.Printf("Ошибка открытия новой БД: %v", err)
		// Пытаемся восстановить старую БД
		oldDB, restoreErr := database.NewDB(s.currentDBPath)
		if restoreErr != nil {
			log.Printf("КРИТИЧЕСКАЯ ОШИБКА: не удалось восстановить старую БД: %v", restoreErr)
			http.Error(w, "Failed to open new database and restore failed", http.StatusInternalServerError)
			return
		}
		s.db = oldDB
		http.Error(w, "Failed to open new database", http.StatusInternalServerError)
		return
	}
	// Гарантируем закрытие новой БД в случае ошибки после открытия
	defer func() {
		// Закрываем newDB только если не удалось успешно переключиться
		if s.currentDBPath != request.Path && newDB != nil {
			if err := newDB.Close(); err != nil {
				log.Printf("Ошибка закрытия новой БД при откате: %v", err)
			}
		}
	}()

	// Успешно переключились
	s.db = newDB
	s.currentDBPath = request.Path
	// Отменяем defer закрытия, так как БД успешно переключена
	newDB = nil

	log.Printf("База данных переключена на: %s", request.Path)

	response := map[string]interface{}{
		"status":  "success",
		"message": "Database switched successfully",
		"path":    request.Path,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
