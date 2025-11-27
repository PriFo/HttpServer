package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"httpserver/classification"
	"httpserver/database"
	"httpserver/nomenclature"
	"httpserver/normalization"
	apperrors "httpserver/server/errors"
)

// ClassificationService сервис для работы с классификацией
type ClassificationService struct {
	db                  *database.DB
	normalizedDB        *database.DB
	serviceDB           *database.ServiceDB
	getModelFromConfig  func() string
	getAPIKeyFromConfig func() string // Функция получения API ключа из конфигурации воркеров
	kpvedWorkersStopped bool
	kpvedWorkersMutex   sync.RWMutex
}

// KpvedStats описывает агрегированные метрики классификации КПВЭД
type KpvedStats struct {
	TotalRecords    int            `json:"total_records"`
	Classified      int            `json:"classified"`
	NotClassified   int            `json:"not_classified"`
	LowConfidence   int            `json:"low_confidence"`
	MarkedIncorrect int            `json:"marked_incorrect"`
	ByConfidence    map[string]int `json:"by_confidence"`
}

// CategoryStats описывает статистику по отдельной категории
type CategoryStats struct {
	Total           int `json:"total"`
	Classified      int `json:"classified"`
	NotClassified   int `json:"not_classified"`
	LowConfidence   int `json:"low_confidence"`
	MarkedIncorrect int `json:"marked_incorrect"`
}

// IncorrectClassificationItem описывает запись с неправильной классификацией
type IncorrectClassificationItem struct {
	NormalizedName string  `json:"normalized_name"`
	Category       string  `json:"category"`
	KpvedCode      string  `json:"kpved_code"`
	KpvedName      string  `json:"kpved_name"`
	Confidence     float64 `json:"confidence"`
	Reason         string  `json:"reason,omitempty"`
}

// NewClassificationService создает новый сервис классификации
// getAPIKeyFromConfig может быть nil - в этом случае будет использоваться только переменная окружения
func NewClassificationService(db *database.DB, normalizedDB *database.DB, serviceDB *database.ServiceDB, getModelFromConfig func() string, getAPIKeyFromConfig func() string) *ClassificationService {
	return &ClassificationService{
		db:                  db,
		normalizedDB:        normalizedDB,
		serviceDB:           serviceDB,
		getModelFromConfig:  getModelFromConfig,
		getAPIKeyFromConfig: getAPIKeyFromConfig,
		kpvedWorkersStopped: false,
	}
}

// GetClassifiers возвращает классификаторы с фильтрацией по клиенту, проекту и признаку активности
func (cs *ClassificationService) GetClassifiers(clientID *int, projectID *int, activeOnly bool) ([]*database.CategoryClassifier, error) {
	if cs == nil || cs.db == nil {
		return nil, apperrors.NewInternalError("classification database not available", nil)
	}
	return cs.db.GetCategoryClassifiersByFilter(clientID, projectID, activeOnly)
}

// GetKpvedHierarchy получает иерархию КПВЭД классификатора
func (cs *ClassificationService) GetKpvedHierarchy(parentCode string, level string) ([]map[string]interface{}, error) {
	db := cs.serviceDB.GetDB()

	// Проверяем, существует ли таблица
	var tableCount int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='kpved_classifier'").Scan(&tableCount)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось проверить наличие таблицы КПВЭД", err)
	}

	if tableCount == 0 {
		// Таблица не существует - возвращаем пустой массив, а не ошибку
		return []map[string]interface{}{}, nil
	}

	// Проверяем, есть ли данные в таблице
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&totalCount)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось подсчитать количество записей в таблице КПВЭД", err)
	}

	if totalCount == 0 {
		// Таблица пуста - возвращаем пустой массив, а не ошибку
		return []map[string]interface{}{}, nil
	}

	// Строим запрос
	query := "SELECT code, name, parent_code, level FROM kpved_classifier WHERE 1=1"
	args := []interface{}{}

	if parentCode != "" {
		query += " AND parent_code = ?"
		args = append(args, parentCode)
	} else if level != "" {
		// Если указан уровень, но нет родителя - показываем этот уровень
		query += " AND level = ?"
		// Валидация уровня будет в handler
		args = append(args, level)
	} else {
		// По умолчанию показываем верхний уровень (секции A-Z, level = 0)
		query += " AND level = 0 AND (parent_code IS NULL OR parent_code = '')"
	}

	query += " ORDER BY code"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError(fmt.Sprintf("не удалось выполнить запрос иерархии КПВЭД: %v", err), err)
	}
	defer rows.Close()

	nodes := []map[string]interface{}{}
	for rows.Next() {
		var code, name string
		var parentCode sql.NullString
		var level int

		if err := rows.Scan(&code, &name, &parentCode, &level); err != nil {
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

	return nodes, nil
}

// SearchKpved выполняет поиск по КПВЭД классификатору
func (cs *ClassificationService) SearchKpved(searchQuery string, limit int) ([]map[string]interface{}, error) {
	// Валидация лимита
	if limit <= 0 {
		return nil, apperrors.NewValidationError("limit должен быть положительным числом", nil)
	}

	db := cs.serviceDB.GetDB()

	// Проверяем, существует ли таблица
	var tableCount int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='kpved_classifier'").Scan(&tableCount)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось проверить наличие таблицы КПВЭД", err)
	}

	if tableCount == 0 {
		// Таблица не существует - возвращаем пустой массив, а не ошибку
		return []map[string]interface{}{}, nil
	}

	// Проверяем, есть ли данные в таблице
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&totalCount)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось подсчитать количество записей в таблице КПВЭД", err)
	}

	if totalCount == 0 {
		// Таблица пуста - возвращаем пустой массив, а не ошибку
		return []map[string]interface{}{}, nil
	}

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
		return nil, apperrors.NewInternalError(fmt.Sprintf("не удалось выполнить поиск КПВЭД: %v", err), err)
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var code, name string
		var parentCode sql.NullString
		var level int

		if err := rows.Scan(&code, &name, &parentCode, &level); err != nil {
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

	return items, nil
}

// GetKpvedStats получает статистику по использованию КПВЭД кодов
func (cs *ClassificationService) GetKpvedStats() (map[string]interface{}, error) {
	db := cs.serviceDB.GetDB()

	// Получаем общее количество записей в классификаторе
	var totalCodes int
	err := db.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&totalCodes)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось подсчитать коды КПВЭД", err)
	}

	// Получаем максимальный уровень в классификаторе
	// Используем COALESCE для обработки NULL, когда таблица пуста
	var maxLevel int
	err = db.QueryRow("SELECT COALESCE(MAX(level), 0) FROM kpved_classifier").Scan(&maxLevel)
	if err != nil {
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
		return nil, apperrors.NewInternalError("не удалось выполнить запрос уровней КПВЭД", err)
	}
	defer levelsRows.Close()

	levels := []map[string]interface{}{}
	for levelsRows.Next() {
		var level, count int
		if err := levelsRows.Scan(&level, &count); err == nil {
			levels = append(levels, map[string]interface{}{
				"level": level,
				"count": count,
			})
		}
	}

	stats := map[string]interface{}{
		"total":               totalCodes,
		"levels":              maxLevel + 1, // +1 потому что уровни начинаются с 0
		"levels_distribution": levels,
	}

	return stats, nil
}

// LoadKpvedFromFile загружает классификатор КПВЭД из файла
func (cs *ClassificationService) LoadKpvedFromFile(filePath string) (int, error) {
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, apperrors.NewNotFoundError("файл не найден", err)
		}
		return 0, apperrors.NewInternalError("не удалось проверить файл", err)
	}

	db := cs.serviceDB
	if err := database.LoadKpvedFromFile(db, filePath); err != nil {
		return 0, apperrors.NewInternalError("не удалось загрузить КПВЭД из файла", err)
	}

	// Получаем статистику после загрузки
	var totalCodes int
	err := db.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&totalCodes)
	if err != nil {
		return 0, apperrors.NewInternalError("не удалось подсчитать коды КПВЭД", err)
	}

	return totalCodes, nil
}

// GetNormalizedItemName возвращает имя нормализованного элемента по ID
func (cs *ClassificationService) GetNormalizedItemName(itemID int) (string, error) {
	if cs == nil || cs.db == nil {
		return "", apperrors.NewInternalError("database not available", nil)
	}

	query := `
		SELECT normalized_name, source_name
		FROM normalized_data
		WHERE id = ?
	`

	var normalizedName, sourceName sql.NullString
	err := cs.db.GetDB().QueryRow(query, itemID).Scan(&normalizedName, &sourceName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", apperrors.NewNotFoundError("normalized item not found", err)
		}
		return "", apperrors.NewInternalError("failed to get normalized item", err)
	}

	name := strings.TrimSpace(normalizedName.String)
	if name == "" {
		name = strings.TrimSpace(sourceName.String)
	}
	if name == "" {
		return "", apperrors.NewValidationError("item_name is required", nil)
	}

	return name, nil
}

// ClassifyTest тестирует КПВЭД классификацию для одного товара
func (cs *ClassificationService) ClassifyTest(normalizedName string, model string) (interface{}, error) {
	var apiKey string
	// Сначала пытаемся получить из функции, если она предоставлена
	if cs.getAPIKeyFromConfig != nil {
		apiKey = cs.getAPIKeyFromConfig()
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey == "" {
		return nil, apperrors.NewInternalError("API ключ AI не настроен. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", nil)
	}

	// Получаем модель: из запроса или из конфигурации
	if model == "" {
		model = cs.getModelFromConfig()
	}

	// Создаем временный классификатор для теста
	classifier := normalization.NewKpvedClassifier(cs.normalizedDB, apiKey, "КПВЭД.txt", model)
	result, err := classifier.ClassifyWithKpved(normalizedName)
	if err != nil {
		return nil, apperrors.NewInternalError("классификация не удалась", err)
	}

	return result, nil
}

// ClassifyHierarchical выполняет иерархическую классификацию для тестирования
func (cs *ClassificationService) ClassifyHierarchical(normalizedName string, category string, model string) (interface{}, error) {
	var apiKey string
	// Сначала пытаемся получить из функции, если она предоставлена
	if cs.getAPIKeyFromConfig != nil {
		apiKey = cs.getAPIKeyFromConfig()
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey == "" {
		return nil, apperrors.NewInternalError("API ключ AI не настроен. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", nil)
	}

	// Используем "общее" как категорию по умолчанию
	if category == "" {
		category = "общее"
	}

	// Получаем модель: из запроса или из конфигурации
	if model == "" {
		model = cs.getModelFromConfig()
	}

	// Создаем AI клиент
	aiClient := nomenclature.NewAIClient(apiKey, model)

	// Создаем иерархический классификатор
	hierarchicalClassifier, err := normalization.NewHierarchicalClassifier(cs.serviceDB, aiClient)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось создать классификатор", err)
	}

	// Классифицируем
	result, err := hierarchicalClassifier.Classify(normalizedName, category)
	if err != nil {
		return nil, apperrors.NewInternalError("классификация не удалась", err)
	}

	return result, nil
}

// ResetClassification сбрасывает классификацию для конкретных записей
func (cs *ClassificationService) ResetClassification(normalizedName string, category string, kpvedCode string, minConfidence float64, resetAll bool) (int64, error) {
	if cs == nil || cs.db == nil {
		return 0, apperrors.NewInternalError("database not available", nil)
	}

	// Строим SQL запрос для сброса
	var conditions []string
	var args []interface{}

	if resetAll {
		query := `UPDATE normalized_data 
			SET kpved_code = NULL, kpved_name = NULL, kpved_confidence = 0.0,
			    validation_status = ''
			WHERE kpved_code IS NOT NULL AND kpved_code != ''`
		result, err := cs.db.Exec(query)
		if err != nil {
			return 0, apperrors.NewInternalError("не удалось сбросить все классификации", err)
		}
		return result.RowsAffected()
	}

	if normalizedName != "" {
		conditions = append(conditions, "normalized_name = ?")
		args = append(args, normalizedName)
	}

	if category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, category)
	}

	if kpvedCode != "" {
		conditions = append(conditions, "kpved_code = ?")
		args = append(args, kpvedCode)
	}

	if minConfidence > 0 {
		conditions = append(conditions, "kpved_confidence < ?")
		args = append(args, minConfidence)
	}

	if len(conditions) == 0 {
		return 0, apperrors.NewValidationError("не указаны критерии для сброса", nil)
	}

	// Безопасная конкатенация условий через strings.Join
	// Условия формируются из контролируемых данных (параметры функции),
	// поэтому безопасны, но используем strings.Join для читаемости
	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`UPDATE normalized_data 
		SET kpved_code = NULL, kpved_name = NULL, kpved_confidence = 0.0,
		    validation_status = ''
		WHERE kpved_code IS NOT NULL AND kpved_code != '' AND %s`, whereClause)

	result, err := cs.db.Exec(query, args...)
	if err != nil {
		return 0, apperrors.NewInternalError("не удалось сбросить классификацию", err)
	}

	return result.RowsAffected()
}

// MarkIncorrect помечает классификацию как неправильную
func (cs *ClassificationService) MarkIncorrect(normalizedName string, category string, reason string) (int64, error) {
	query := `UPDATE normalized_data 
		SET validation_status = 'incorrect',
		    validation_reason = ?,
		    kpved_code = NULL, kpved_name = NULL, kpved_confidence = 0.0
		WHERE normalized_name = ? AND category = ?`

	result, err := cs.db.Exec(query, reason, normalizedName, category)
	if err != nil {
		return 0, apperrors.NewInternalError("не удалось пометить как неправильное", err)
	}

	return result.RowsAffected()
}

// MarkCorrect снимает пометку неправильной классификации
func (cs *ClassificationService) MarkCorrect(normalizedName string, category string) (int64, error) {
	query := `UPDATE normalized_data 
		SET validation_status = 'correct',
		    validation_reason = NULL
		WHERE normalized_name = ? AND category = ?`

	result, err := cs.db.Exec(query, normalizedName, category)
	if err != nil {
		return 0, apperrors.NewInternalError("не удалось пометить как правильное", err)
	}

	return result.RowsAffected()
}

// GetClassifiersByProjectType получает классификаторы по типу проекта
func (cs *ClassificationService) GetClassifiersByProjectType(projectType string) ([]map[string]interface{}, error) {
	if cs.serviceDB == nil {
		return nil, apperrors.NewInternalError("service database not available", nil)
	}
	return cs.serviceDB.GetClassifiersByProjectType(projectType)
}

// GetFoldingStrategies получает стратегии свертки для клиента
func (cs *ClassificationService) GetFoldingStrategies(clientID *int) ([]*database.FoldingStrategy, error) {
	if cs.db == nil {
		return nil, apperrors.NewInternalError("database not available", nil)
	}
	if clientID == nil {
		return []*database.FoldingStrategy{}, nil
	}
	return cs.db.GetFoldingStrategiesByClient(*clientID)
}

// CreateClientStrategy создает стратегию свертки для конкретного клиента
func (cs *ClassificationService) CreateClientStrategy(clientID int, config classification.FoldingStrategyConfig) (*database.FoldingStrategy, error) {
	if cs == nil || cs.db == nil {
		return nil, apperrors.NewInternalError("database not available", nil)
	}
	if clientID <= 0 {
		return nil, apperrors.NewValidationError("client_id must be greater than zero", nil)
	}
	if strings.TrimSpace(config.Name) == "" {
		return nil, apperrors.NewValidationError("strategy name is required", nil)
	}
	if config.MaxDepth <= 0 {
		config.MaxDepth = 2
	}
	if config.ID == "" {
		config.ID = fmt.Sprintf("client_%d_%d", clientID, time.Now().UnixNano())
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to serialize strategy config", err)
	}

	clientIDCopy := clientID
	dbStrategy := &database.FoldingStrategy{
		Name:           config.Name,
		Description:    config.Description,
		StrategyConfig: string(configJSON),
		ClientID:       &clientIDCopy,
		IsDefault:      false,
	}

	createdStrategy, err := cs.db.CreateFoldingStrategy(dbStrategy)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to create strategy", err)
	}

	return createdStrategy, nil
}

// ClassifyItemAI выполняет прямую AI-классификацию товара без создания сессии
func (cs *ClassificationService) ClassifyItemAI(itemName string, itemCode string, category string, model string, context map[string]interface{}) (*classification.AIClassificationResponse, string, error) {
	if strings.TrimSpace(itemName) == "" {
		return nil, "", apperrors.NewValidationError("item_name is required", nil)
	}

	var apiKey string
	// Сначала пытаемся получить из функции, если она предоставлена
	if cs.getAPIKeyFromConfig != nil {
		apiKey = cs.getAPIKeyFromConfig()
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey == "" {
		return nil, "", apperrors.NewInternalError("AI API key not configured. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", nil)
	}

	actualModel := model
	if actualModel == "" && cs.getModelFromConfig != nil {
		actualModel = cs.getModelFromConfig()
	}
	if actualModel == "" {
		actualModel = "GLM-4.5-Air"
	}

	aiClassifier := classification.NewAIClassifier(apiKey, actualModel)
	aiRequest := classification.AIClassificationRequest{
		ItemName:    itemName,
		Description: itemCode,
		Context:     context,
		MaxLevels:   5,
	}

	result, err := aiClassifier.ClassifyWithAI(aiRequest)
	if err != nil {
		return nil, "", apperrors.NewInternalError("classification failed", err)
	}

	return result, actualModel, nil
}

// GetOptimizationStats возвращает информацию об оптимизациях классификатора
func (cs *ClassificationService) GetOptimizationStats() map[string]interface{} {
	return map[string]interface{}{
		"optimizations": map[string]interface{}{
			"category_format": map[string]interface{}{
				"enabled":     true,
				"description": "Компактный список категорий вместо дерева",
				"reduction":   "90-95%",
			},
			"category_cache": map[string]interface{}{
				"enabled":     true,
				"description": "Кэширование списка категорий",
				"benefit":     "Исключены повторные вычисления",
			},
			"prompt_simplification": map[string]interface{}{
				"enabled":     true,
				"description": "Упрощенный промпт",
				"reduction":   "~95% (с ~2000+ до ~50-100 токенов)",
			},
			"system_prompt_simplification": map[string]interface{}{
				"enabled":     true,
				"description": "Упрощенный системный промпт",
				"reduction":   "~85% (с 7 строк до 1 строки)",
			},
			"name_truncation": map[string]interface{}{
				"enabled":     true,
				"description": "Обрезка длинных названий категорий",
				"max_length":  50,
			},
			"compact_output": map[string]interface{}{
				"enabled":     true,
				"description": "Компактный формат вывода без ID и форматирования",
			},
		},
		"expected_results": map[string]interface{}{
			"context_size_reduction": "50-100x (с ~105000 до ~1000-2000 токенов)",
			"performance":            "Кэширование ускоряет последующие запросы",
			"reliability":            "Ошибки 503 должны исчезнуть",
			"quality":                "Классификация остается точной",
		},
		"monitoring": map[string]interface{}{
			"prompt_size_logging": true,
			"token_estimation":    true,
			"cache_statistics":    true,
			"performance_metrics": true,
			"log_prefix":          "[AIClassifier]",
		},
		"configuration": map[string]interface{}{
			"max_categories":        15,
			"max_category_name_len": 50,
			"enable_logging":        true,
			"env_variables": map[string]string{
				"AI_CLASSIFIER_MAX_CATEGORIES": "Максимальное количество категорий (по умолчанию 15)",
				"AI_CLASSIFIER_MAX_NAME_LEN":   "Максимальная длина названия категории (по умолчанию 50)",
				"AI_CLASSIFIER_ENABLE_LOGGING": "Включить логирование (true/false, по умолчанию true)",
			},
		},
	}
}

func (cs *ClassificationService) getServiceDatabase() (*sql.DB, error) {
	if cs == nil || cs.serviceDB == nil {
		return nil, apperrors.NewInternalError("service database not available", nil)
	}
	return cs.serviceDB.GetDB(), nil
}

func (cs *ClassificationService) okpd2TableExists(db *sql.DB) (bool, error) {
	if db == nil {
		return false, apperrors.NewInternalError("service database not available", nil)
	}

	var exists int
	query := `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='okpd2_classifier'
		)
	`
	if err := db.QueryRow(query).Scan(&exists); err != nil {
		return false, apperrors.NewInternalError("failed to check OKPD2 table", err)
	}

	return exists == 1, nil
}

// GetOkpd2Hierarchy возвращает иерархию ОКПД2
func (cs *ClassificationService) GetOkpd2Hierarchy(parentCode string, level *int) (map[string]interface{}, error) {
	db, err := cs.getServiceDatabase()
	if err != nil {
		return nil, err
	}

	exists, err := cs.okpd2TableExists(db)
	if err != nil {
		return nil, err
	}

	if !exists {
		return map[string]interface{}{
			"nodes": []map[string]interface{}{},
			"total": 0,
		}, nil
	}

	query := "SELECT code, name, parent_code, level FROM okpd2_classifier WHERE 1=1"
	args := []interface{}{}

	if strings.TrimSpace(parentCode) != "" {
		query += " AND parent_code = ?"
		args = append(args, parentCode)
	} else if level != nil {
		query += " AND level = ?"
		args = append(args, *level)
	} else {
		query += " AND (parent_code IS NULL OR parent_code = '')"
	}

	query += " ORDER BY code"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить иерархию OKPD2", err)
	}
	defer rows.Close()

	nodes := make([]map[string]interface{}, 0)
	for rows.Next() {
		var code, name string
		var parent sql.NullString
		var levelValue int

		if err := rows.Scan(&code, &name, &parent, &levelValue); err != nil {
			continue
		}

		var childCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM okpd2_classifier WHERE parent_code = ?", code).Scan(&childCount); err != nil {
			childCount = 0
		}

		node := map[string]interface{}{
			"code":         code,
			"name":         name,
			"level":        levelValue,
			"has_children": childCount > 0,
		}
		if parent.Valid {
			node["parent_code"] = parent.String
		}

		nodes = append(nodes, node)
	}

	total := len(nodes)
	if err := db.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&total); err != nil {
		total = len(nodes)
	}

	return map[string]interface{}{
		"nodes": nodes,
		"total": total,
	}, nil
}

// SearchOkpd2 выполняет поиск по классификатору ОКПД2
func (cs *ClassificationService) SearchOkpd2(query string, limit int) ([]map[string]interface{}, error) {
	db, err := cs.getServiceDatabase()
	if err != nil {
		return nil, err
	}

	exists, err := cs.okpd2TableExists(db)
	if err != nil {
		return nil, err
	}

	if !exists {
		return []map[string]interface{}{}, nil
	}

	searchPattern := "%" + query + "%"
	sqlQuery := `
		SELECT code, name, parent_code, level
		FROM okpd2_classifier
		WHERE code LIKE ? OR name LIKE ?
		ORDER BY code
		LIMIT ?
	`

	rows, err := db.Query(sqlQuery, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось выполнить поиск OKPD2", err)
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var code, name string
		var parent sql.NullString
		var levelValue int

		if err := rows.Scan(&code, &name, &parent, &levelValue); err != nil {
			continue
		}

		entry := map[string]interface{}{
			"code":  code,
			"name":  name,
			"level": levelValue,
		}
		if parent.Valid {
			entry["parent_code"] = parent.String
		}

		results = append(results, entry)
	}

	return results, nil
}

// GetOkpd2Stats возвращает агрегированную статистику по классификатору
func (cs *ClassificationService) GetOkpd2Stats() (map[string]interface{}, error) {
	db, err := cs.getServiceDatabase()
	if err != nil {
		return nil, err
	}

	exists, err := cs.okpd2TableExists(db)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{}
	if !exists {
		stats["total_codes"] = 0
		stats["max_level"] = 0
		stats["levels"] = []map[string]interface{}{}
		return stats, nil
	}

	var totalCodes int
	if err := db.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&totalCodes); err != nil {
		return nil, apperrors.NewInternalError("не удалось подсчитать OKPD2 коды", err)
	}
	stats["total_codes"] = totalCodes

	// Используем COALESCE для обработки NULL, когда таблица пуста
	var maxLevel int
	if err := db.QueryRow("SELECT COALESCE(MAX(level), 0) FROM okpd2_classifier").Scan(&maxLevel); err != nil {
		maxLevel = 0
	}
	stats["max_level"] = maxLevel

	levelRows, err := db.Query(`
		SELECT level, COUNT(*) as count
		FROM okpd2_classifier
		GROUP BY level
		ORDER BY level
	`)
	if err != nil {
		stats["levels"] = []map[string]interface{}{}
		return stats, nil
	}
	defer levelRows.Close()

	distribution := make([]map[string]interface{}, 0)
	for levelRows.Next() {
		var levelValue, count int
		if err := levelRows.Scan(&levelValue, &count); err != nil {
			continue
		}
		distribution = append(distribution, map[string]interface{}{
			"level": levelValue,
			"count": count,
		})
	}
	stats["levels"] = distribution

	return stats, nil
}

// LoadOkpd2FromFile загружает классификатор из файла
func (cs *ClassificationService) LoadOkpd2FromFile(filePath string) (int, error) {
	if strings.TrimSpace(filePath) == "" {
		return 0, apperrors.NewValidationError("file_path is required", nil)
	}

	if cs == nil || cs.serviceDB == nil {
		return 0, apperrors.NewInternalError("service database not available", nil)
	}

	if err := database.LoadOkpd2FromFile(cs.serviceDB, filePath); err != nil {
		return 0, apperrors.NewInternalError("failed to load OKPD2 from file", err)
	}

	var totalCodes int
	if err := cs.serviceDB.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&totalCodes); err != nil {
		return 0, apperrors.NewInternalError("failed to count OKPD2 codes", err)
	}

	return totalCodes, nil
}

// ClearOkpd2 очищает классификатор
func (cs *ClassificationService) ClearOkpd2() (int, bool, error) {
	db, err := cs.getServiceDatabase()
	if err != nil {
		return 0, false, err
	}

	exists, err := cs.okpd2TableExists(db)
	if err != nil {
		return 0, false, err
	}

	if !exists {
		return 0, false, nil
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM okpd2_classifier").Scan(&total); err != nil {
		return 0, true, apperrors.NewInternalError("failed to count OKPD2 records", err)
	}

	if _, err := db.Exec("DELETE FROM okpd2_classifier"); err != nil {
		return 0, true, apperrors.NewInternalError("failed to clear OKPD2 classifier", err)
	}

	return total, true, nil
}

// ReclassifyKpvedGroups выполняет массовую переклассификацию незаполненных групп
func (cs *ClassificationService) ReclassifyKpvedGroups(limit int) (int, int, []map[string]interface{}, error) {
	if cs == nil || cs.db == nil || cs.normalizedDB == nil {
		return 0, 0, nil, apperrors.NewInternalError("classification databases not available", nil)
	}

	var apiKey string
	// Сначала пытаемся получить из функции, если она предоставлена
	if cs.getAPIKeyFromConfig != nil {
		apiKey = cs.getAPIKeyFromConfig()
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if strings.TrimSpace(apiKey) == "" {
		return 0, 0, nil, apperrors.NewInternalError("AI API key not configured. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", nil)
	}

	model := "GLM-4.5-Air"
	if cs.getModelFromConfig != nil {
		if configured := strings.TrimSpace(cs.getModelFromConfig()); configured != "" {
			model = configured
		}
	}

	if limit <= 0 {
		limit = 1000000
	}

	query := `
		SELECT DISTINCT normalized_name, category
		FROM normalized_data
		WHERE (kpved_code IS NULL OR kpved_code = '' OR TRIM(kpved_code) = '')
		LIMIT ?
	`

	rows, err := cs.db.Query(query, limit)
	if err != nil {
		return 0, 0, nil, apperrors.NewInternalError("failed to query groups for reclassification", err)
	}
	defer rows.Close()

	classifier := normalization.NewKpvedClassifier(cs.normalizedDB, apiKey, "КПВЭД.txt", model)

	classified := 0
	failed := 0
	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		var normalizedName, category string
		if err := rows.Scan(&normalizedName, &category); err != nil {
			continue
		}

		result, err := classifier.ClassifyWithKpved(normalizedName)
		if err != nil {
			failed++
			continue
		}

		updateQuery := `
			UPDATE normalized_data
			SET kpved_code = ?, kpved_name = ?, kpved_confidence = ?
			WHERE normalized_name = ? AND category = ?
		`
		if _, err := cs.db.Exec(updateQuery, result.KpvedCode, result.KpvedName, result.KpvedConfidence, normalizedName, category); err != nil {
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
	}

	return classified, failed, results, nil
}

// ReclassifyKpvedHierarchical выполняет иерархическую классификацию для одной группы
func (cs *ClassificationService) ReclassifyKpvedHierarchical(normalizedName string, category string, modelOverride string) (map[string]interface{}, error) {
	if cs == nil || cs.normalizedDB == nil || cs.serviceDB == nil {
		return nil, apperrors.NewInternalError("classification databases not available", nil)
	}

	name := strings.TrimSpace(normalizedName)
	if name == "" {
		return nil, apperrors.NewValidationError("normalized_name is required", nil)
	}
	if strings.TrimSpace(category) == "" {
		category = "общее"
	}

	var apiKey string
	// Сначала пытаемся получить из функции, если она предоставлена
	if cs.getAPIKeyFromConfig != nil {
		apiKey = cs.getAPIKeyFromConfig()
	}
	// Fallback на переменную окружения
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, apperrors.NewInternalError("AI API key not configured. Установите API ключ в разделе 'Воркеры' или через переменную окружения ARLIAI_API_KEY", nil)
	}

	model := strings.TrimSpace(modelOverride)
	if model == "" && cs.getModelFromConfig != nil {
		model = cs.getModelFromConfig()
	}
	if model == "" {
		model = "GLM-4.5-Air"
	}

	aiClient := nomenclature.NewAIClient(apiKey, model)
	hierarchicalClassifier, err := normalization.NewHierarchicalClassifier(cs.serviceDB, aiClient)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to create hierarchical classifier", err)
	}

	start := time.Now()
	result, err := hierarchicalClassifier.Classify(name, category)
	if err != nil {
		return nil, apperrors.NewInternalError("hierarchical classification failed", err)
	}
	result.TotalDuration = time.Since(start).Milliseconds()

	response := map[string]interface{}{
		"final_code":      result.FinalCode,
		"final_name":      result.FinalName,
		"total_duration":  result.TotalDuration,
		"steps":           result.Steps,
		"model":           model,
		"normalized_name": name,
		"category":        category,
	}

	if result.FinalCode != "" {
		updateQuery := `
			UPDATE normalized_data
			SET kpved_code = ?, kpved_name = ?, kpved_confidence = ?, validation_status = ''
			WHERE normalized_name = ? AND category = ?
		`
		if _, err := cs.db.Exec(updateQuery, result.FinalCode, result.FinalName, result.FinalConfidence, name, category); err != nil {
			return nil, apperrors.NewInternalError("failed to persist classification result", err)
		}
	}

	return response, nil
}

// GetKpvedCurrentTasks возвращает список групп без классификации
func (cs *ClassificationService) GetKpvedCurrentTasks(limit int) ([]map[string]interface{}, error) {
	if cs == nil || cs.db == nil {
		return nil, apperrors.NewInternalError("database not available", nil)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT normalized_name, category, COUNT(*) as merged_count
		FROM normalized_data
		WHERE kpved_code IS NULL OR kpved_code = '' OR TRIM(kpved_code) = ''
		GROUP BY normalized_name, category
		ORDER BY merged_count DESC, normalized_name
		LIMIT ?
	`

	rows, err := cs.db.Query(query, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to query current tasks", err)
	}
	defer rows.Close()

	tasks := make([]map[string]interface{}, 0)
	index := 1
	for rows.Next() {
		var normalizedName, category string
		var mergedCount int
		if err := rows.Scan(&normalizedName, &category, &mergedCount); err != nil {
			continue
		}

		tasks = append(tasks, map[string]interface{}{
			"normalized_name": normalizedName,
			"category":        category,
			"merged_count":    mergedCount,
			"index":           index,
		})
		index++
	}

	return tasks, nil
}

// GetKpvedWorkersStatus возвращает состояние воркеров КПВЭД
func (cs *ClassificationService) GetKpvedWorkersStatus(limit int) (map[string]interface{}, error) {
	if cs == nil {
		return nil, apperrors.NewInternalError("classification service not available", nil)
	}

	if limit <= 0 {
		limit = 20
	}

	currentTasks, err := cs.GetKpvedCurrentTasks(limit)
	if err != nil {
		return nil, err
	}

	cs.kpvedWorkersMutex.RLock()
	stopped := cs.kpvedWorkersStopped
	cs.kpvedWorkersMutex.RUnlock()

	return map[string]interface{}{
		"is_running":    !stopped,
		"stopped":       stopped,
		"workers_count": len(currentTasks),
		"current_tasks": currentTasks,
	}, nil
}

// StopKpvedWorkers активирует флаг остановки воркеров
func (cs *ClassificationService) StopKpvedWorkers() (map[string]interface{}, error) {
	if cs == nil {
		return nil, apperrors.NewInternalError("classification service not available", nil)
	}

	cs.kpvedWorkersMutex.Lock()
	cs.kpvedWorkersStopped = true
	cs.kpvedWorkersMutex.Unlock()

	return map[string]interface{}{
		"success": true,
		"message": "Воркеры остановлены. Текущие задачи будут завершены, новые задачи не будут обрабатываться.",
		"stopped": true,
	}, nil
}

// ResumeKpvedWorkers снимает флаг остановки воркеров
func (cs *ClassificationService) ResumeKpvedWorkers() (map[string]interface{}, error) {
	if cs == nil {
		return nil, apperrors.NewInternalError("classification service not available", nil)
	}

	cs.kpvedWorkersMutex.Lock()
	cs.kpvedWorkersStopped = false
	cs.kpvedWorkersMutex.Unlock()

	return map[string]interface{}{
		"success": true,
		"message": "Воркеры возобновлены",
		"stopped": false,
	}, nil
}

// GetModelsBenchmarkHistory возвращает историю или последний результат бенчмарков моделей
func (cs *ClassificationService) GetModelsBenchmarkHistory(limit int, model string, includeHistory bool) (map[string]interface{}, error) {
	if cs == nil || cs.serviceDB == nil {
		return nil, apperrors.NewInternalError("service database not available", nil)
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	records, err := cs.serviceDB.GetBenchmarkHistory(limit, model)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get benchmark history", err)
	}

	if includeHistory {
		return map[string]interface{}{
			"history": records,
			"total":   len(records),
		}, nil
	}

	if len(records) == 0 {
		return map[string]interface{}{
			"models":     []map[string]interface{}{},
			"total":      0,
			"test_count": 0,
			"timestamp":  "",
			"message":    "No benchmark results found. Use POST to run benchmark or ?history=true to get history",
		}, nil
	}

	lastTimestamp := fmt.Sprintf("%v", records[0]["timestamp"])
	models := make([]map[string]interface{}, 0)
	testCount := 0

	for _, record := range records {
		if fmt.Sprintf("%v", record["timestamp"]) != lastTimestamp {
			break
		}

		if testCount == 0 {
			if val, ok := record["test_count"].(int); ok {
				testCount = val
			} else if converted, ok := convertToInt(record["test_count"]); ok {
				testCount = converted
			}
		}

		modelEntry := map[string]interface{}{
			"model":                   record["model"],
			"priority":                record["priority"],
			"speed":                   record["speed"],
			"avg_response_time_ms":    record["avg_response_time_ms"],
			"median_response_time_ms": record["median_response_time_ms"],
			"p95_response_time_ms":    record["p95_response_time_ms"],
			"min_response_time_ms":    record["min_response_time_ms"],
			"max_response_time_ms":    record["max_response_time_ms"],
			"success_count":           record["success_count"],
			"error_count":             record["error_count"],
			"total_requests":          record["total_requests"],
			"success_rate":            record["success_rate"],
			"status":                  record["status"],
		}
		models = append(models, modelEntry)
	}

	sort.Slice(models, func(i, j int) bool {
		pi, _ := convertToInt(models[i]["priority"])
		pj, _ := convertToInt(models[j]["priority"])
		return pi < pj
	})

	return map[string]interface{}{
		"models":     models,
		"total":      len(models),
		"test_count": testCount,
		"timestamp":  lastTimestamp,
		"message":    "Last benchmark result from history. Use POST to run benchmark or ?history=true to get full history",
	}, nil
}

func convertToInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

// GetKpvedStatsGeneral возвращает агрегированные метрики по классификации КПВЭД
func (cs *ClassificationService) GetKpvedStatsGeneral() (*KpvedStats, error) {
	if cs == nil || cs.db == nil {
		return nil, apperrors.NewInternalError("database not available", nil)
	}

	db := cs.db.GetDB()
	stats := &KpvedStats{
		ByConfidence: map[string]int{
			"high":   0,
			"medium": 0,
			"low":    0,
		},
	}

	if err := db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&stats.TotalRecords); err != nil {
		return nil, apperrors.NewInternalError("failed to count normalized records", err)
	}

	if err := db.QueryRow(`
		SELECT COUNT(*) FROM normalized_data 
		WHERE kpved_code IS NOT NULL AND kpved_code != '' AND TRIM(kpved_code) != ''
	`).Scan(&stats.Classified); err != nil {
		return nil, apperrors.NewInternalError("failed to count classified records", err)
	}
	stats.NotClassified = stats.TotalRecords - stats.Classified

	if err := db.QueryRow(`
		SELECT COUNT(*) FROM normalized_data 
		WHERE kpved_code IS NOT NULL AND kpved_code != '' 
		  AND kpved_confidence < 0.7
	`).Scan(&stats.LowConfidence); err != nil {
		return nil, apperrors.NewInternalError("failed to count low confidence records", err)
	}

	if err := db.QueryRow(`
		SELECT COUNT(*) FROM normalized_data 
		WHERE validation_status = 'incorrect'
	`).Scan(&stats.MarkedIncorrect); err != nil {
		return nil, apperrors.NewInternalError("failed to count incorrect records", err)
	}

	var highConfidence int
	db.QueryRow(`
		SELECT COUNT(*) FROM normalized_data 
		WHERE kpved_code IS NOT NULL AND kpved_code != '' 
		  AND kpved_confidence >= 0.9
	`).Scan(&highConfidence)
	stats.ByConfidence["high"] = highConfidence

	var mediumConfidence int
	db.QueryRow(`
		SELECT COUNT(*) FROM normalized_data 
		WHERE kpved_code IS NOT NULL AND kpved_code != '' 
		  AND kpved_confidence >= 0.7 AND kpved_confidence < 0.9
	`).Scan(&mediumConfidence)
	stats.ByConfidence["medium"] = mediumConfidence

	var lowConfidence int
	db.QueryRow(`
		SELECT COUNT(*) FROM normalized_data 
		WHERE kpved_code IS NOT NULL AND kpved_code != '' 
		  AND kpved_confidence < 0.7
	`).Scan(&lowConfidence)
	stats.ByConfidence["low"] = lowConfidence

	return stats, nil
}

// GetKpvedStatsByCategory возвращает статистику классификации по категориям
func (cs *ClassificationService) GetKpvedStatsByCategory() (map[string]CategoryStats, error) {
	if cs == nil || cs.db == nil {
		return nil, apperrors.NewInternalError("database not available", nil)
	}

	db := cs.db.GetDB()
	query := `
		SELECT 
			category,
			COUNT(*) as total,
			COUNT(CASE WHEN kpved_code IS NOT NULL AND kpved_code != '' THEN 1 END) as classified,
			COUNT(CASE WHEN kpved_code IS NULL OR kpved_code = '' THEN 1 END) as not_classified,
			COUNT(CASE WHEN kpved_code IS NOT NULL AND kpved_code != '' AND kpved_confidence < 0.7 THEN 1 END) as low_confidence,
			COUNT(CASE WHEN validation_status = 'incorrect' THEN 1 END) as marked_incorrect
		FROM normalized_data
		GROUP BY category
		ORDER BY total DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to query category stats", err)
	}
	defer rows.Close()

	stats := make(map[string]CategoryStats)
	for rows.Next() {
		var category string
		var value CategoryStats
		if err := rows.Scan(&category, &value.Total, &value.Classified, &value.NotClassified, &value.LowConfidence, &value.MarkedIncorrect); err != nil {
			continue
		}
		stats[category] = value
	}

	return stats, nil
}

// GetIncorrectKpvedClassifications возвращает записи, помеченные как неправильные
func (cs *ClassificationService) GetIncorrectKpvedClassifications(limit int) ([]IncorrectClassificationItem, int, error) {
	if cs == nil || cs.db == nil {
		return nil, 0, apperrors.NewInternalError("database not available", nil)
	}
	if limit <= 0 {
		limit = 100
	}

	db := cs.db.GetDB()
	query := `
		SELECT DISTINCT normalized_name, category, kpved_code, kpved_name, 
		       kpved_confidence, validation_reason
		FROM normalized_data
		WHERE validation_status = 'incorrect'
		ORDER BY normalized_name, category
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, 0, apperrors.NewInternalError("failed to query incorrect classifications", err)
	}
	defer rows.Close()

	items := make([]IncorrectClassificationItem, 0)
	for rows.Next() {
		var item IncorrectClassificationItem
		if err := rows.Scan(&item.NormalizedName, &item.Category, &item.KpvedCode, &item.KpvedName, &item.Confidence, &item.Reason); err != nil {
			continue
		}
		items = append(items, item)
	}

	var total int
	db.QueryRow(`
		SELECT COUNT(DISTINCT normalized_name || '|' || category) 
		FROM normalized_data WHERE validation_status = 'incorrect'
	`).Scan(&total)

	return items, total, nil
}
