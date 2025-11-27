package database

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// allowedTableNames whitelist допустимых имен таблиц
	// Добавьте сюда все таблицы, которые могут использоваться в динамических SQL запросах
	allowedTableNames = map[string]bool{
		"normalized_data":        true,
		"catalog_items":          true,
		"uploads":                true,
		"clients":                true,
		"client_projects":        true,
		"project_databases":      true,
		"normalized_counterparties": true,
		"data_quality_issues":    true,
		"data_quality_metrics":   true,
		"duplicate_groups":       true,
		"sqlite_master":          true, // для системных запросов
	}

	// allowedColumnNames whitelist допустимых имен колонок
	// Используется для валидации динамических имен колонок
	allowedColumnNames = map[string]bool{
		"id":                     true,
		"code":                   true,
		"name":                   true,
		"normalized_name":        true,
		"category":               true,
		"kpved_code":             true,
		"kpved_confidence":       true,
		"processing_level":       true,
		"ai_confidence":          true,
		"ai_reasoning":           true,
		"merged_count":           true,
		"quality_score":          true,
		"source_reference":       true,
		"source_name":            true,
		"category_level1":        true,
		"category_level2":        true,
		"category_level3":        true,
		"category_level4":        true,
		"category_level5":        true,
	}

	// tableNamePattern паттерн для валидации имен таблиц (alphanumeric + underscore)
	tableNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// columnNamePattern паттерн для валидации имен колонок (alphanumeric + underscore)
	columnNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// ValidateTableName проверяет, что имя таблицы безопасно для использования в SQL запросах
// Проверяет:
// 1. Соответствие паттерну (alphanumeric + underscore)
// 2. Наличие в whitelist (если strict=true)
//
// Если strict=false, проверяет только паттерн (для случаев, когда таблица создается динамически)
func ValidateTableName(name string, strict bool) error {
	if name == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Проверяем паттерн
	if !tableNamePattern.MatchString(name) {
		return fmt.Errorf("invalid table name format: %s (must contain only alphanumeric characters and underscores, start with letter or underscore)", name)
	}

	// Если strict=true, проверяем whitelist
	if strict {
		if !allowedTableNames[name] {
			return fmt.Errorf("table name '%s' is not in allowed list. Allowed tables: %v", name, getKeys(allowedTableNames))
		}
	}

	return nil
}

// ValidateColumnName проверяет, что имя колонки безопасно для использования в SQL запросах
// Проверяет:
// 1. Соответствие паттерну (alphanumeric + underscore)
// 2. Наличие в whitelist (если strict=true)
//
// Если strict=false, проверяет только паттерн (для случаев, когда колонка создается динамически)
func ValidateColumnName(name string, strict bool) error {
	if name == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	// Проверяем паттерн
	if !columnNamePattern.MatchString(name) {
		return fmt.Errorf("invalid column name format: %s (must contain only alphanumeric characters and underscores, start with letter or underscore)", name)
	}

	// Если strict=true, проверяем whitelist
	if strict {
		// Проверяем базовое имя (без суффикса уровня для category_levelN)
		baseName := name
		if strings.HasPrefix(name, "category_level") {
			baseName = "category_level"
		}
		
		if !allowedColumnNames[baseName] && !allowedColumnNames[name] {
			return fmt.Errorf("column name '%s' is not in allowed list. Allowed columns: %v", name, getKeys(allowedColumnNames))
		}
	}

	return nil
}

// ValidateTableAndColumnNames проверяет имя таблицы и список имен колонок
func ValidateTableAndColumnNames(tableName string, columnNames []string, strict bool) error {
	if err := ValidateTableName(tableName, strict); err != nil {
		return fmt.Errorf("table validation failed: %w", err)
	}

	for _, colName := range columnNames {
		if err := ValidateColumnName(colName, strict); err != nil {
			return fmt.Errorf("column '%s' validation failed: %w", colName, err)
		}
	}

	return nil
}

// getKeys возвращает ключи из map (для сообщений об ошибках)
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// AddAllowedTableName добавляет имя таблицы в whitelist (для расширяемости)
func AddAllowedTableName(name string) {
	allowedTableNames[name] = true
}

// AddAllowedColumnName добавляет имя колонки в whitelist (для расширяемости)
func AddAllowedColumnName(name string) {
	allowedColumnNames[name] = true
}


