package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/ для организации,
// но остается в пакете server для доступа к методам Server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"httpserver/database"
)

// handleOkpd2Hierarchy возвращает иерархию ОКПД2 классификатора
func (s *Server) handleOkpd2Hierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры
	parentCode := r.URL.Query().Get("parent")
	level := r.URL.Query().Get("level")
	// Используем сервисную БД для классификатора ОКПД2
	db := s.serviceDB.GetDB()

	// Строим запрос
	query := "SELECT code, name, parent_code, level FROM okpd2_classifier WHERE 1=1"
	args := []interface{}{}

	if parentCode != "" {
		query += " AND parent_code = ?"
		args = append(args, parentCode)
	} else if level != "" {
		// Если указан уровень, но нет родителя - показываем этот уровень
		query += " AND level = ?"
		levelInt, _ := strconv.Atoi(level)
		args = append(args, levelInt)
	} else {
		// По умолчанию показываем верхний уровень (level = 0 или минимальный)
		query += " AND (parent_code IS NULL OR parent_code = '')"
	}

	query += " ORDER BY code"

	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='okpd2_classifier'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Printf("Error checking okpd2_classifier table: %v", err)
		s.writeJSONError(w, r, "Failed to check OKPD2 table", http.StatusInternalServerError)
		return
	}

	if !tableExists {
		log.Printf("okpd2_classifier table does not exist")
		// Возвращаем пустой результат вместо ошибки
		s.writeJSONResponse(w, r, map[string]interface{}{
			"nodes": []map[string]interface{}{},
			"total": 0,
		}, http.StatusOK)
		return
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying okpd2 hierarchy: %v", err)
		s.writeJSONError(w, r, "Failed to fetch OKPD2 hierarchy", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	nodes := []map[string]interface{}{}
	for rows.Next() {
		var code, name string
		var parentCode sql.NullString
		var level int

		if err := rows.Scan(&code, &name, &parentCode, &level); err != nil {
			log.Printf("Error scanning okpd2 row: %v", err)
			continue
		}

		// Проверяем, есть ли дочерние узлы
		var hasChildren bool
		childQuery := "SELECT COUNT(*) FROM okpd2_classifier WHERE parent_code = ?"
		err = db.QueryRow(childQuery, code).Scan(&hasChildren)
		if err != nil {
			log.Printf("Error checking children for %s: %v", code, err)
		}

		node := map[string]interface{}{
			"code":        code,
			"name":        name,
			"level":       level,
			"has_children": hasChildren,
		}

		if parentCode.Valid {
			node["parent_code"] = parentCode.String
		}

		nodes = append(nodes, node)
	}

	// Получаем общее количество
	var total int
	err = db.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&total)
	if err != nil {
		log.Printf("Error counting okpd2 nodes: %v", err)
		total = len(nodes)
	}

	response := map[string]interface{}{
		"nodes": nodes,
		"total": total,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleOkpd2Clear очищает классификатор ОКПД2
func (s *Server) handleOkpd2Clear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := s.serviceDB.GetDB()

	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='okpd2_classifier'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Printf("Error checking okpd2_classifier table: %v", err)
		s.writeJSONError(w, r, "Failed to check OKPD2 table", http.StatusInternalServerError)
		return
	}

	if !tableExists {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"success": true,
			"message": "Классификатор ОКПД2 уже пуст",
			"deleted_count": 0,
		}, http.StatusOK)
		return
	}

	// Получаем количество записей перед удалением
	var countBefore int
	err = db.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&countBefore)
	if err != nil {
		log.Printf("Error counting okpd2 records: %v", err)
		countBefore = 0
	}

	// Очищаем таблицу
	_, err = db.Exec("DELETE FROM okpd2_classifier")
	if err != nil {
		log.Printf("Error clearing okpd2_classifier table: %v", err)
		s.writeJSONError(w, r, "Failed to clear OKPD2 classifier", http.StatusInternalServerError)
		return
	}

	log.Printf("[OKPD2] Cleared %d records from okpd2_classifier", countBefore)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":      true,
		"message":      "Классификатор ОКПД2 успешно очищен",
		"deleted_count": countBefore,
	}, http.StatusOK)
}

// handleOkpd2Search выполняет поиск по классификатору ОКПД2
func (s *Server) handleOkpd2Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeJSONError(w, r, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	db := s.serviceDB.GetDB()

	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='okpd2_classifier'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Printf("Error checking okpd2_classifier table: %v", err)
		s.writeJSONError(w, r, "Failed to check OKPD2 table", http.StatusInternalServerError)
		return
	}

	if !tableExists {
		log.Printf("okpd2_classifier table does not exist")
		s.writeJSONResponse(w, r, map[string]interface{}{
			"results": []map[string]interface{}{},
			"total":   0,
		}, http.StatusOK)
		return
	}

	// Поиск по коду или названию
	searchQuery := `
		SELECT code, name, parent_code, level 
		FROM okpd2_classifier 
		WHERE code LIKE ? OR name LIKE ?
		ORDER BY code
		LIMIT 50
	`

	searchPattern := "%" + query + "%"
	rows, err := db.Query(searchQuery, searchPattern, searchPattern)
	if err != nil {
		log.Printf("Error searching okpd2: %v", err)
		s.writeJSONError(w, r, "Failed to search OKPD2", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var code, name string
		var parentCode sql.NullString
		var level int

		if err := rows.Scan(&code, &name, &parentCode, &level); err != nil {
			continue
		}

		result := map[string]interface{}{
			"code":  code,
			"name":  name,
			"level": level,
		}

		if parentCode.Valid {
			result["parent_code"] = parentCode.String
		}

		results = append(results, result)
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"results": results,
		"total":   len(results),
	}, http.StatusOK)
}

// handleOkpd2Stats возвращает статистику по классификатору ОКПД2
func (s *Server) handleOkpd2Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db := s.serviceDB.GetDB()

	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='okpd2_classifier'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Printf("Error checking okpd2_classifier table: %v", err)
		s.writeJSONError(w, r, "Failed to check OKPD2 table", http.StatusInternalServerError)
		return
	}

	stats := map[string]interface{}{}

	if !tableExists {
		stats["total_codes"] = 0
		stats["max_level"] = 0
		stats["levels"] = []map[string]interface{}{}
		s.writeJSONResponse(w, r, stats, http.StatusOK)
		return
	}

	// Общее количество кодов
	var totalCodes int
	err = db.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&totalCodes)
	if err != nil {
		log.Printf("Error counting okpd2 codes: %v", err)
		s.writeJSONError(w, r, "Failed to get OKPD2 stats", http.StatusInternalServerError)
		return
	}
	stats["total_codes"] = totalCodes

	// Максимальный уровень
	// Используем COALESCE для обработки NULL, когда таблица пуста
	var maxLevel int
	err = db.QueryRow("SELECT COALESCE(MAX(level), 0) FROM okpd2_classifier").Scan(&maxLevel)
	if err != nil {
		log.Printf("Error getting max level: %v", err)
		maxLevel = 0
	}
	stats["max_level"] = maxLevel

	// Распределение по уровням
	levelStats := []map[string]interface{}{}
	levelQuery := `
		SELECT level, COUNT(*) as count
		FROM okpd2_classifier
		GROUP BY level
		ORDER BY level
	`
	levelRows, err := db.Query(levelQuery)
	if err == nil {
		defer levelRows.Close()
		for levelRows.Next() {
			var level, count int
			if err := levelRows.Scan(&level, &count); err == nil {
				levelStats = append(levelStats, map[string]interface{}{
					"level": level,
					"count": count,
				})
			}
		}
	}
	stats["levels"] = levelStats

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleOkpd2LoadFromFile обрабатывает загрузку файла с классификатором ОКПД2
// Поддерживает два формата:
// 1. JSON с полем file_path - путь к файлу на сервере
// 2. multipart/form-data с полем file - загруженный файл
func (s *Server) handleOkpd2LoadFromFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var filePath string
	var fileName string

	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		// JSON формат - получаем путь к файлу
		var req struct {
			FilePath string `json:"file_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[OKPD2] Error decoding JSON: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка парсинга JSON: %v", err), http.StatusBadRequest)
			return
		}
		filePath = req.FilePath
		fileName = filePath
	} else {
		// multipart/form-data - получаем файл
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			log.Printf("[OKPD2] Error parsing multipart form: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка парсинга формы: %v", err), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			log.Printf("[OKPD2] Error getting file from form: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка получения файла: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileName = header.Filename
		log.Printf("[OKPD2] Received file: %s (size: %d bytes)", fileName, header.Size)

		// Создаем временный файл
		tempDir := "/tmp"
		if runtime.GOOS == "windows" {
			tempDir = os.TempDir()
		}
		tempFile, err := os.CreateTemp(tempDir, "okpd2-*.txt")
		if err != nil {
			log.Printf("[OKPD2] Error creating temp file: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка создания временного файла: %v", err), http.StatusInternalServerError)
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Копируем содержимое
		_, err = io.Copy(tempFile, file)
		if err != nil {
			log.Printf("[OKPD2] Error copying file content: %v", err)
			http.Error(w, fmt.Sprintf("Ошибка сохранения файла: %v", err), http.StatusInternalServerError)
			return
		}
		tempFile.Close()
		filePath = tempFile.Name()
	}

	if filePath == "" {
		http.Error(w, "Не указан путь к файлу или файл не загружен", http.StatusBadRequest)
		return
	}

	log.Printf("[OKPD2] Loading from file: %s", filePath)

	// Загружаем данные из файла в сервисную БД
	err := database.LoadOkpd2FromFile(s.serviceDB, filePath)
	if err != nil {
		log.Printf("[OKPD2] Error loading OKPD2 from file: %v", err)
		http.Error(w, fmt.Sprintf("Ошибка загрузки данных: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем количество загруженных записей
	var totalCodes int
	err = s.serviceDB.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&totalCodes)
	if err != nil {
		log.Printf("[OKPD2] Error counting loaded codes: %v", err)
		totalCodes = 0
	}

	log.Printf("[OKPD2] Successfully loaded %d OKPD2 records from file %s", totalCodes, fileName)

	// Возвращаем успешный ответ
	response := map[string]interface{}{
		"success":     true,
		"message":     "Классификатор ОКПД2 успешно загружен",
		"filename":    fileName,
		"total_codes": totalCodes,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

