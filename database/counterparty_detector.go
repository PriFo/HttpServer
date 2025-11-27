package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// DatabaseStructureMetadata содержит метаданные о структуре базы данных
type DatabaseStructureMetadata struct {
	ID                   int
	DatabaseID           int
	TableName            string
	EntityType           string  // 'counterparty', 'nomenclature', 'document'
	ColumnMappings       string  // JSON: {"name": "Наименование", "inn": "ИНН"}
	DetectionConfidence  float64 // 0.0-1.0
	LastUpdated          string
}

// CounterpartyStructure содержит информацию о структуре таблицы контрагентов
type CounterpartyStructure struct {
	TableName       string
	NameColumn      string
	INNColumn       string
	OGRNColumn      string
	KPPColumn       string
	BINColumn       string
	LegalNameColumn string
	AddressColumn   string
	PhoneColumn     string
	EmailColumn     string
	Confidence      float64
}

// CounterpartyDetector автоматически обнаруживает структуру контрагентов в БД
type CounterpartyDetector struct {
	serviceDB *ServiceDB
}

// NewCounterpartyDetector создает новый детектор контрагентов
func NewCounterpartyDetector(serviceDB *ServiceDB) *CounterpartyDetector {
	return &CounterpartyDetector{
		serviceDB: serviceDB,
	}
}

// DetectStructure автоматически определяет структуру таблицы контрагентов
func (d *CounterpartyDetector) DetectStructure(databaseID int, dbPath string) (*CounterpartyStructure, error) {
	// Открываем базу данных
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 1. Находим потенциальные таблицы с контрагентами
	tables, err := d.findCounterpartyTables(db)
	if err != nil {
		return nil, fmt.Errorf("failed to find counterparty tables: %w", err)
	}

	if len(tables) == 0 {
		return nil, fmt.Errorf("no counterparty tables found")
	}

	// 2. Анализируем каждую таблицу и выбираем лучшую
	var bestStructure *CounterpartyStructure
	for _, tableName := range tables {
		structure, err := d.analyzeTable(db, tableName)
		if err != nil {
			log.Printf("Failed to analyze table %s: %v", tableName, err)
			continue
		}

		if bestStructure == nil || structure.Confidence > bestStructure.Confidence {
			bestStructure = structure
		}
	}

	if bestStructure == nil {
		return nil, fmt.Errorf("failed to detect counterparty structure")
	}

	// 3. Сохраняем метаданные в service DB
	if err := d.saveMetadata(databaseID, bestStructure); err != nil {
		log.Printf("Warning: failed to save metadata: %v", err)
	}

	return bestStructure, nil
}

// findCounterpartyTables находит таблицы, которые могут содержать контрагентов
func (d *CounterpartyDetector) findCounterpartyTables(db *sql.DB) ([]string, error) {
	query := `
		SELECT name FROM sqlite_master 
		WHERE type='table' 
		AND (
			LOWER(name) LIKE '%контрагент%' 
			OR LOWER(name) LIKE '%counterpart%'
			OR LOWER(name) LIKE '%контр_агент%'
			OR LOWER(name) = 'клиенты'
			OR LOWER(name) = 'clients'
			OR LOWER(name) = 'catalog_items'
			OR LOWER(name) = 'catalogs'
		)
		AND name NOT LIKE 'sqlite_%'
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// analyzeTable анализирует структуру таблицы и определяет колонки
func (d *CounterpartyDetector) analyzeTable(db *sql.DB, tableName string) (*CounterpartyStructure, error) {
	// Получаем список колонок
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]string) // columnName -> type
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		columns[name] = colType
	}

	// Создаем структуру и определяем колонки
	structure := &CounterpartyStructure{
		TableName:  tableName,
		Confidence: 0.0,
	}

	// Определяем колонки и вычисляем confidence
	for colName := range columns {
		colLower := strings.ToLower(colName)

		// Название/Имя
		if structure.NameColumn == "" {
			if colLower == "наименование" || colLower == "name" || 
			   colLower == "название" || colLower == "full_name" || 
			   colLower == "title" {
				structure.NameColumn = colName
				structure.Confidence += 0.3
			}
		}

		// ИНН
		if structure.INNColumn == "" {
			if colLower == "инн" || colLower == "inn" || colLower == "tax_id" {
				structure.INNColumn = colName
				structure.Confidence += 0.25
			}
		}

		// БИН (для Казахстана)
		if structure.BINColumn == "" {
			if colLower == "бин" || colLower == "bin" {
				structure.BINColumn = colName
				structure.Confidence += 0.15
			}
		}

		// ОГРН
		if structure.OGRNColumn == "" {
			if colLower == "огрн" || colLower == "ogrn" {
				structure.OGRNColumn = colName
				structure.Confidence += 0.1
			}
		}

		// КПП
		if structure.KPPColumn == "" {
			if colLower == "кпп" || colLower == "kpp" {
				structure.KPPColumn = colName
				structure.Confidence += 0.05
			}
		}

		// Юридическое наименование
		if structure.LegalNameColumn == "" {
			if strings.Contains(colLower, "legal") || colLower == "полное_наименование" {
				structure.LegalNameColumn = colName
				structure.Confidence += 0.05
			}
		}

		// Адрес
		if structure.AddressColumn == "" {
			if colLower == "адрес" || colLower == "address" || 
			   colLower == "legal_address" || colLower == "юридический_адрес" {
				structure.AddressColumn = colName
				structure.Confidence += 0.03
			}
		}

		// Телефон
		if structure.PhoneColumn == "" {
			if colLower == "телефон" || colLower == "phone" || 
			   colLower == "contact_phone" {
				structure.PhoneColumn = colName
				structure.Confidence += 0.02
			}
		}

		// Email
		if structure.EmailColumn == "" {
			if colLower == "email" || colLower == "почта" || 
			   colLower == "contact_email" {
				structure.EmailColumn = colName
				structure.Confidence += 0.02
			}
		}
	}

	// Бонус за название таблицы
	tableLower := strings.ToLower(tableName)
	if strings.Contains(tableLower, "контрагент") || strings.Contains(tableLower, "counterpart") {
		structure.Confidence += 0.3
	} else if tableLower == "клиенты" || tableLower == "clients" {
		structure.Confidence += 0.2
	}

	// Проверяем, что хотя бы имя или ИНН/БИН найдены
	if structure.NameColumn == "" && structure.INNColumn == "" && structure.BINColumn == "" {
		return nil, fmt.Errorf("table %s does not have required columns (name, inn or bin)", tableName)
	}

	return structure, nil
}

// saveMetadata сохраняет метаданные в service DB
func (d *CounterpartyDetector) saveMetadata(databaseID int, structure *CounterpartyStructure) error {
	if d.serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	// Формируем JSON с маппингом колонок
	mappings := fmt.Sprintf(`{
		"table_name": "%s",
		"name": "%s",
		"inn": "%s",
		"bin": "%s",
		"ogrn": "%s",
		"kpp": "%s",
		"legal_name": "%s",
		"address": "%s",
		"phone": "%s",
		"email": "%s"
	}`, structure.TableName, structure.NameColumn, structure.INNColumn,
		structure.BINColumn, structure.OGRNColumn, structure.KPPColumn,
		structure.LegalNameColumn, structure.AddressColumn,
		structure.PhoneColumn, structure.EmailColumn)

	query := `
		INSERT OR REPLACE INTO database_table_metadata 
		(database_id, table_name, entity_type, column_mappings, detection_confidence, last_updated)
		VALUES (?, ?, 'counterparty', ?, ?, datetime('now'))
	`

	_, err := d.serviceDB.conn.Exec(query, databaseID, structure.TableName, mappings, structure.Confidence)
	return err
}

// GetCachedMetadata получает кэшированные метаданные
func (d *CounterpartyDetector) GetCachedMetadata(databaseID int) (*CounterpartyStructure, error) {
	if d.serviceDB == nil {
		return nil, fmt.Errorf("serviceDB is nil")
	}

	query := `
		SELECT table_name, column_mappings, detection_confidence
		FROM database_table_metadata
		WHERE database_id = ? AND entity_type = 'counterparty'
		ORDER BY detection_confidence DESC
		LIMIT 1
	`

	var tableName, mappings string
	var confidence float64

	err := d.serviceDB.conn.QueryRow(query, databaseID).Scan(&tableName, &mappings, &confidence)
	if err == sql.ErrNoRows {
		return nil, nil // Метаданных нет
	}
	if err != nil {
		return nil, err
	}

	// TODO: Парсинг JSON mappings в структуру
	// Для простоты пока возвращаем базовую структуру
	structure := &CounterpartyStructure{
		TableName:  tableName,
		Confidence: confidence,
	}

	return structure, nil
}

// GetCounterparties получает контрагентов из БД используя обнаруженную структуру
func (d *CounterpartyDetector) GetCounterparties(dbPath string, structure *CounterpartyStructure, limit, offset int) ([]map[string]interface{}, error) {
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Формируем SELECT с обнаруженными колонками
	selectCols := []string{}
	if structure.NameColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as name", structure.NameColumn))
	}
	if structure.INNColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as inn", structure.INNColumn))
	}
	if structure.BINColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as bin", structure.BINColumn))
	}
	if structure.OGRNColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as ogrn", structure.OGRNColumn))
	}
	if structure.KPPColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as kpp", structure.KPPColumn))
	}
	if structure.LegalNameColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as legal_name", structure.LegalNameColumn))
	}
	if structure.AddressColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as address", structure.AddressColumn))
	}
	if structure.PhoneColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as phone", structure.PhoneColumn))
	}
	if structure.EmailColumn != "" {
		selectCols = append(selectCols, fmt.Sprintf("%s as email", structure.EmailColumn))
	}

	if len(selectCols) == 0 {
		return nil, fmt.Errorf("no columns found")
	}

	// Формируем запрос
	whereClause := ""
	if structure.NameColumn != "" {
		whereClause = fmt.Sprintf("WHERE %s IS NOT NULL AND %s != ''", structure.NameColumn, structure.NameColumn)
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM %s
		%s
		LIMIT ? OFFSET ?
	`, strings.Join(selectCols, ", "), structure.TableName, whereClause)

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query counterparties: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := []map[string]interface{}{}
	for rows.Next() {
		// Создаем слайс интерфейсов для сканирования
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		// Формируем map
		entry := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				entry[col] = string(b)
			} else {
				entry[col] = val
			}
		}

		results = append(results, entry)
	}

	return results, nil
}

