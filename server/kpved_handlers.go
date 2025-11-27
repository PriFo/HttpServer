package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл содержит KPVED handlers, извлеченные из server.go
// для сокращения размера server.go

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"httpserver/database"
	"httpserver/nomenclature"
	"httpserver/normalization"
)

func (s *Server) handleKpvedHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры
	parentCode := r.URL.Query().Get("parent")
	level := r.URL.Query().Get("level")
	// Используем сервисную БД для классификатора КПВЭД
	db := s.serviceDB.GetDB()

	// Строим запрос
	query := "SELECT code, name, parent_code, level FROM kpved_classifier WHERE 1=1"
	args := []interface{}{}

	if parentCode != "" {
		query += " AND parent_code = ?"
		args = append(args, parentCode)
	} else if level != "" {
		// Если указан уровень, но нет родителя - показываем этот уровень
		query += " AND level = ?"
		levelInt, err := ValidateIntPathParam(level, "level")
		if err == nil {
			args = append(args, levelInt)
		}
		// Если уровень невалидный, игнорируем его (fallback на уровень по умолчанию)
	} else {
		// По умолчанию показываем верхний уровень (секции A-Z, level = 0)
		// Секции имеют parent_code = NULL или parent_code = ''
		query += " AND level = 0 AND (parent_code IS NULL OR parent_code = '')"
	}

	query += " ORDER BY code"

	// Сначала проверяем, существует ли таблица
	var tableCount int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='kpved_classifier'").Scan(&tableCount)
	if err != nil {
		log.Printf("[KPVED] Error checking table existence: %v", err)
		s.writeJSONError(w, r, "Failed to check KPVED table", http.StatusInternalServerError)
		return
	}

	if tableCount == 0 {
		log.Printf("[KPVED] Table kpved_classifier does not exist")
		// Возвращаем пустой массив, а не ошибку
		response := map[string]interface{}{
			"nodes": []map[string]interface{}{},
			"total": 0,
		}
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
	}

	// Проверяем, есть ли данные в таблице
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&totalCount)
	if err != nil {
		log.Printf("[KPVED] Error counting rows: %v", err)
	} else {
		log.Printf("[KPVED] Total rows in kpved_classifier: %d", totalCount)
	}

	if totalCount == 0 {
		log.Printf("[KPVED] Table kpved_classifier is empty")
		// Возвращаем пустой массив, а не ошибку
		response := map[string]interface{}{
			"nodes": []map[string]interface{}{},
			"total": 0,
		}
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
	}

	log.Printf("[KPVED] Querying hierarchy with query: %s, args: %v", query, args)
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("[KPVED] Error querying kpved hierarchy: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to fetch KPVED hierarchy: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	nodes := []map[string]interface{}{}
	for rows.Next() {
		var code, name string
		var parentCode sql.NullString
		var level int

		if err := rows.Scan(&code, &name, &parentCode, &level); err != nil {
			log.Printf("Error scanning kpved row: %v", err)
			continue
		}

		// Проверяем, есть ли дочерние узлы
		var hasChildren bool
		childQuery := "SELECT COUNT(*) FROM kpved_classifier WHERE parent_code = ?"
		var childCount int
		if err := db.QueryRow(childQuery, code).Scan(&childCount); err == nil {
			hasChildren = childCount > 0
		}

		node := map[string]interface{}{
			"code":         code,
			"name":         name,
			"level":        level,
			"has_children": hasChildren,
		}
		if parentCode.Valid {
			node["parent_code"] = parentCode.String
		}

		nodes = append(nodes, node)
	}

	// Формируем ответ в формате, ожидаемом фронтендом
	log.Printf("[KPVED] Returning %d nodes for hierarchy query", len(nodes))
	response := map[string]interface{}{
		"nodes": nodes,
		"total": len(nodes),
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleKpvedSearch выполняет поиск по КПВЭД классификатору
func (s *Server) handleKpvedSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		s.writeJSONError(w, r, "Search query is required", http.StatusBadRequest)
		return
	}

	limit, err := ValidateIntParam(r, "limit", 50, 1, 100)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	// Используем сервисную БД для классификатора КПВЭД
	db := s.serviceDB.GetDB()

	query := `
		SELECT code, name, parent_code, level
		FROM kpved_classifier
		WHERE name LIKE ? OR code LIKE ?
		ORDER BY level, code
		LIMIT ?
	`

	searchParam := "%" + searchQuery + "%"
	rows, err := db.Query(query, searchParam, searchParam, limit)
	if err != nil {
		log.Printf("Error searching kpved: %v", err)
		s.writeJSONError(w, r, "Failed to search KPVED", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var code, name string
		var parentCode sql.NullString
		var level int

		if err := rows.Scan(&code, &name, &parentCode, &level); err != nil {
			log.Printf("Error scanning kpved row: %v", err)
			continue
		}

		item := map[string]interface{}{
			"code":  code,
			"name":  name,
			"level": level,
		}
		if parentCode.Valid {
			item["parent_code"] = parentCode.String
		}

		items = append(items, item)
	}

	s.writeJSONResponse(w, r, items, http.StatusOK)
}

// handleKpvedStats возвращает статистику по использованию КПВЭД кодов
func (s *Server) handleKpvedStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Используем сервисную БД для классификатора КПВЭД
	db := s.serviceDB.GetDB()

	// Получаем общее количество записей в классификаторе
	var totalCodes int
	err := db.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&totalCodes)
	if err != nil {
		log.Printf("Error counting kpved codes: %v", err)
		totalCodes = 0
	}

	// Получаем максимальный уровень в классификаторе
	// Используем COALESCE для обработки NULL, когда таблица пуста
	var maxLevel int
	err = db.QueryRow("SELECT COALESCE(MAX(level), 0) FROM kpved_classifier").Scan(&maxLevel)
	if err != nil {
		log.Printf("Error getting max level: %v", err)
		maxLevel = 0
	}

	// Получаем распределение по уровням
	levelsQuery := `
		SELECT level, COUNT(*) as count
		FROM kpved_classifier
		GROUP BY level
		ORDER BY level
	`
	levelsRows, err := db.Query(levelsQuery)
	if err != nil {
		log.Printf("Error querying kpved levels: %v", err)
	}
	defer levelsRows.Close()

	levels := []map[string]interface{}{}
	if levelsRows != nil {
		for levelsRows.Next() {
			var level, count int
			if err := levelsRows.Scan(&level, &count); err == nil {
				levels = append(levels, map[string]interface{}{
					"level": level,
					"count": count,
				})
			}
		}
	}

	// Формируем упрощенную статистику для фронтенда
	stats := map[string]interface{}{
		"total":               totalCodes,
		"levels":              maxLevel + 1, // +1 потому что уровни начинаются с 0
		"levels_distribution": levels,
	}

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleKpvedLoad загружает классификатор КПВЭД из файла в базу данных
func (s *Server) handleKpvedLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	var req struct {
		FilePath string `json:"file_path"`
		Database string `json:"database,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		s.writeJSONError(w, r, "file_path is required", http.StatusBadRequest)
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(req.FilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeJSONError(w, r, fmt.Sprintf("File not found: %s", req.FilePath), http.StatusNotFound)
			return
		}
		s.writeJSONError(w, r, fmt.Sprintf("Error checking file: %v", err), http.StatusInternalServerError)
		return
	}

	// Используем сервисную БД для классификатора КПВЭД
	log.Printf("Loading KPVED classifier from file: %s to service database", req.FilePath)
	if err := database.LoadKpvedFromFile(s.serviceDB, req.FilePath); err != nil {
		log.Printf("Error loading KPVED: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to load KPVED: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем статистику после загрузки
	var totalCodes int
	err := s.serviceDB.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&totalCodes)
	if err != nil {
		log.Printf("Error counting kpved codes: %v", err)
		totalCodes = 0
	}

	response := map[string]interface{}{
		"success":     true,
		"message":     "KPVED classifier loaded successfully",
		"file_path":   req.FilePath,
		"total_codes": totalCodes,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleKpvedClassifyTest тестирует КПВЭД классификацию для одного товара
func (s *Server) handleKpvedClassifyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	var req struct {
		NormalizedName string `json:"normalized_name"`
		Model          string `json:"model"` // Опциональный параметр для указания модели
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NormalizedName == "" {
		http.Error(w, "normalized_name is required", http.StatusBadRequest)
		return
	}

	// Проверяем, что нормализатор существует и AI включен
	if s.normalizer == nil {
		http.Error(w, "Normalizer not initialized", http.StatusInternalServerError)
		return
	}

	// Получаем API ключ из конфигурации воркеров или переменной окружения
	var apiKey string
	if s.workerConfigManager != nil {
		key, _, err := s.workerConfigManager.GetModelAndAPIKey()
		if err == nil && key != "" {
			apiKey = key
		}
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey == "" {
		http.Error(w, "AI API key not configured. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", http.StatusServiceUnavailable)
		return
	}

	// Получаем модель: из запроса или из конфигурации
	model := req.Model
	if model == "" {
		model = s.getModelFromConfig()
	}

	// Создаем временный классификатор для теста
	classifier := normalization.NewKpvedClassifier(s.normalizedDB, apiKey, "КПВЭД.txt", model)
	result, err := classifier.ClassifyWithKpved(req.NormalizedName)
	if err != nil {
		log.Printf("Error classifying: %v", err)
		http.Error(w, fmt.Sprintf("Classification failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, result, http.StatusOK)
}

// handleKpvedReclassify переклассифицирует существующие группы
func (s *Server) handleKpvedReclassify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	var req struct {
		Limit int `json:"limit"` // Количество групп для переклассификации (0 = все)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Limit = 10 // По умолчанию 10 групп
	}

	// Получаем API ключ из конфигурации воркеров или переменной окружения
	var apiKey string
	if s.workerConfigManager != nil {
		key, _, err := s.workerConfigManager.GetModelAndAPIKey()
		if err == nil && key != "" {
			apiKey = key
		}
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey == "" {
		http.Error(w, "AI API key not configured. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", http.StatusServiceUnavailable)
		return
	}

	// Получаем группы без КПВЭД классификации
	query := `
		SELECT DISTINCT normalized_name, category
		FROM normalized_data
		WHERE (kpved_code IS NULL OR kpved_code = '' OR TRIM(kpved_code) = '')
		LIMIT ?
	`

	limitValue := req.Limit
	if limitValue == 0 {
		limitValue = 1000000 // Большое число для "все"
	}

	rows, err := s.db.Query(query, limitValue)
	if err != nil {
		log.Printf("Error querying groups: %v", err)
		http.Error(w, "Failed to query groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Получаем модель из WorkerConfigManager
	model := s.getModelFromConfig()

	// Создаем классификатор
	classifier := normalization.NewKpvedClassifier(s.normalizedDB, apiKey, "КПВЭД.txt", model)

	classified := 0
	failed := 0
	results := []map[string]interface{}{}

	for rows.Next() {
		var normalizedName, category string
		if err := rows.Scan(&normalizedName, &category); err != nil {
			continue
		}

		// Классифицируем
		result, err := classifier.ClassifyWithKpved(normalizedName)
		if err != nil {
			log.Printf("Failed to classify '%s': %v", normalizedName, err)
			failed++
			continue
		}

		// Обновляем все записи в этой группе
		updateQuery := `
			UPDATE normalized_data
			SET kpved_code = ?, kpved_name = ?, kpved_confidence = ?
			WHERE normalized_name = ? AND category = ?
		`
		_, err = s.db.Exec(updateQuery, result.KpvedCode, result.KpvedName, result.KpvedConfidence, normalizedName, category)
		if err != nil {
			log.Printf("Failed to update group '%s': %v", normalizedName, err)
			failed++
			continue
		}

		classified++
		results = append(results, map[string]interface{}{
			"normalized_name":  normalizedName,
			"category":         category,
			"kpved_code":       result.KpvedCode,
			"kpved_name":       result.KpvedName,
			"kpved_confidence": result.KpvedConfidence,
		})

		// Логируем прогресс
		if classified%10 == 0 {
			log.Printf("Reclassified %d groups...", classified)
		}
	}

	response := map[string]interface{}{
		"classified": classified,
		"failed":     failed,
		"results":    results,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleKpvedClassifyHierarchical выполняет иерархическую классификацию для тестирования
func (s *Server) handleKpvedClassifyHierarchical(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	var req struct {
		NormalizedName string `json:"normalized_name"`
		Category       string `json:"category"`
		Model          string `json:"model"` // Опциональный параметр для указания модели
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NormalizedName == "" {
		http.Error(w, "normalized_name is required", http.StatusBadRequest)
		return
	}

	// Используем "общее" как категорию по умолчанию
	if req.Category == "" {
		req.Category = "общее"
	}

	// Получаем API ключ
	// Получаем API ключ из конфигурации воркеров или переменной окружения
	var apiKey string
	if s.workerConfigManager != nil {
		key, _, err := s.workerConfigManager.GetModelAndAPIKey()
		if err == nil && key != "" {
			apiKey = key
		}
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey == "" {
		http.Error(w, "AI API key not configured. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", http.StatusServiceUnavailable)
		return
	}

	// Получаем модель: из запроса или из WorkerConfigManager
	model := req.Model
	if model == "" {
		var err error
		_, model, err = s.workerConfigManager.GetModelAndAPIKey()
		if err != nil {
			log.Printf("[KPVED Test] Error getting model from config: %v, using default", err)
			model = "GLM-4.5-Air" // Дефолтная модель
		}
	}
	log.Printf("[KPVED Test] Using model: %s", model)

	// Создаем AI клиент
	aiClient := nomenclature.NewAIClient(apiKey, model)

	// Создаем иерархический классификатор (используем serviceDB где находится kpved_classifier)
	hierarchicalClassifier, err := normalization.NewHierarchicalClassifier(s.serviceDB, aiClient)
	if err != nil {
		log.Printf("Error creating hierarchical classifier: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create classifier: %v", err), http.StatusInternalServerError)
		return
	}

	// Классифицируем
	startTime := time.Now()
	result, err := hierarchicalClassifier.Classify(req.NormalizedName, req.Category)
	if err != nil {
		log.Printf("Error classifying: %v", err)
		http.Error(w, fmt.Sprintf("Classification failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Добавляем общее время выполнения
	result.TotalDuration = time.Since(startTime).Milliseconds()

	log.Printf("Hierarchical classification completed: %s -> %s (%s) in %dms with %d steps",
		req.NormalizedName, result.FinalCode, result.FinalName, result.TotalDuration, len(result.Steps))

	s.writeJSONResponse(w, r, result, http.StatusOK)
}

// ClassificationTask представляет задачу для классификации группы
// Экспортирован для использования в handlers
type ClassificationTask struct {
	NormalizedName string
	Category       string
	MergedCount    int // Количество дублей в группе
	Index          int
}

// classificationTask представляет задачу для классификации группы (приватный алиас для обратной совместимости)
type classificationTask = ClassificationTask

// classificationResult представляет результат классификации
type classificationResult struct {
	task         ClassificationTask
	result       *normalization.HierarchicalResult
	err          error
	rowsAffected int64
}

// handleKpvedReclassifyHierarchical переклассифицирует существующие группы с иерархическим подходом
// Реализация вынесена в server_kpved_reclassify.go для улучшения читаемости и поддержки

// handleKpvedCurrentTasks возвращает текущие обрабатываемые задачи
func (s *Server) handleKpvedCurrentTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.kpvedCurrentTasksMutex.RLock()
	defer s.kpvedCurrentTasksMutex.RUnlock()

	// Преобразуем map в массив для JSON
	currentTasks := []map[string]interface{}{}
	for workerID, task := range s.kpvedCurrentTasks {
		if task != nil {
			currentTasks = append(currentTasks, map[string]interface{}{
				"worker_id":       workerID,
				"normalized_name": task.NormalizedName,
				"category":        task.Category,
				"merged_count":    task.MergedCount,
				"index":           task.Index,
			})
		}
	}

	response := map[string]interface{}{
		"current_tasks": currentTasks,
		"count":         len(currentTasks),
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// ============================================================================
// Quality Endpoints Handlers
// ============================================================================

// handleQualityUploadRoutes обрабатывает маршруты качества для выгрузок

// Quality handlers перемещены в server/quality_legacy_handlers.go
