//go:build tool_analyze_database_structure
// +build tool_analyze_database_structure

package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// DatabaseInfo информация о БД
type DatabaseInfo struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	FilePath     string `json:"file_path"`
	ProjectID    int    `json:"project_id"`
	ProjectName  string `json:"project_name"`
	TableName    string `json:"table_name,omitempty"`
	Columns      []ColumnInfo `json:"columns,omitempty"`
}

// ColumnInfo информация о колонке
type ColumnInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"not_null"`
}

func main() {
	// Подключаемся к service.db
	serviceDB, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("Failed to open service.db: %v", err)
	}
	defer serviceDB.Close()

	// Получаем все БД из project_databases
	query := `
		SELECT pd.id, pd.name, pd.file_path, pd.client_project_id, cp.name as project_name
		FROM project_databases pd
		JOIN client_projects cp ON pd.client_project_id = cp.id
		WHERE cp.client_id = 1 AND pd.is_active = 1
		ORDER BY cp.id, pd.id
	`

	rows, err := serviceDB.Query(query)
	if err != nil {
		log.Fatalf("Failed to query databases: %v", err)
	}
	defer rows.Close()

	var databases []DatabaseInfo
	for rows.Next() {
		var db DatabaseInfo
		if err := rows.Scan(&db.ID, &db.Name, &db.FilePath, &db.ProjectID, &db.ProjectName); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		log.Printf("Found DB: ID=%d, Name=%s, Path=%s", db.ID, db.Name, db.FilePath)
		databases = append(databases, db)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	fmt.Printf("\nНайдено БД: %d\n\n", len(databases))

	// Анализируем каждую БД
	results := make(map[string][]DatabaseInfo)
	
	for _, db := range databases {
		fmt.Printf("Анализ БД: %s (ID: %d)\n", db.Name, db.ID)
		fmt.Printf("  Путь: %s\n", db.FilePath)
		
		// Проверяем существование файла
		if _, err := os.Stat(db.FilePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Printf("  ❌ Файл не существует\n\n")
			} else {
				fmt.Printf("  ❌ Ошибка проверки файла: %v\n\n", err)
			}
			continue
		}

		// Открываем БД
		targetDB, err := sql.Open("sqlite3", db.FilePath+"?mode=ro")
		if err != nil {
			fmt.Printf("  ❌ Ошибка открытия: %v\n\n", err)
			continue
		}

		// Ищем таблицы с контрагентами
		tablesQuery := `
			SELECT name FROM sqlite_master 
			WHERE type='table' 
			AND (
				LOWER(name) LIKE '%контрагент%' 
				OR LOWER(name) LIKE '%counterpart%'
				OR LOWER(name) = 'catalog_items'
				OR LOWER(name) = 'catalogs'
			)
			AND name NOT LIKE 'sqlite_%'
		`

		tableRows, err := targetDB.Query(tablesQuery)
		if err != nil {
			targetDB.Close()
			fmt.Printf("  ❌ Ошибка получения таблиц: %v\n\n", err)
			continue
		}

		var tables []string
		for tableRows.Next() {
			var tableName string
			if err := tableRows.Scan(&tableName); err == nil {
				tables = append(tables, tableName)
			}
		}
		tableRows.Close()

		fmt.Printf("  Найдено таблиц: %d\n", len(tables))

		// Анализируем каждую таблицу
		for _, tableName := range tables {
			fmt.Printf("  📋 Таблица: %s\n", tableName)
			
			// Получаем структуру таблицы
			columnQuery := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
			colRows, err := targetDB.Query(columnQuery)
			if err != nil {
				continue
			}

			var columns []ColumnInfo
			for colRows.Next() {
				var cid int
				var name, colType string
				var notNull, pk int
				var dfltValue sql.NullString

				if err := colRows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err == nil {
					columns = append(columns, ColumnInfo{
						Name:    name,
						Type:    colType,
						NotNull: notNull == 1,
					})
				}
			}
			colRows.Close()

			fmt.Printf("    Колонок: %d\n", len(columns))
			
			dbInfo := DatabaseInfo{
				ID:          db.ID,
				Name:        db.Name,
				FilePath:    db.FilePath,
				ProjectID:   db.ProjectID,
				ProjectName: db.ProjectName,
				TableName:   tableName,
				Columns:     columns,
			}

			results[tableName] = append(results[tableName], dbInfo)

			// Выводим первые 10 колонок
			for i, col := range columns {
				if i >= 10 {
					fmt.Printf("    ... и еще %d колонок\n", len(columns)-10)
					break
				}
				notNullStr := ""
				if col.NotNull {
					notNullStr = " NOT NULL"
				}
				fmt.Printf("    - %s (%s%s)\n", col.Name, col.Type, notNullStr)
			}
		}

		targetDB.Close()
		fmt.Println()
	}

	// Сохраняем результаты в JSON
	jsonData, _ := json.MarshalIndent(results, "", "  ")
	if err := os.WriteFile("database_structure_analysis.json", jsonData, 0644); err != nil {
		log.Printf("Failed to save JSON: %v", err)
	} else {
		fmt.Println("✅ Результаты сохранены в database_structure_analysis.json")
	}

	// Создаем отчет о совместимости
	fmt.Println("\n=== АНАЛИЗ СОВМЕСТИМОСТИ ===\n")
	
	for tableName, dbs := range results {
		if len(dbs) <= 1 {
			continue
		}

		fmt.Printf("📊 Таблица: %s (найдена в %d БД)\n", tableName, len(dbs))
		
		// Собираем все уникальные колонки
		allColumns := make(map[string]int) // column name -> count
		for _, db := range dbs {
			for _, col := range db.Columns {
				allColumns[col.Name]++
			}
		}

		// Находим общие и уникальные колонки
		commonCols := []string{}
		uniqueCols := []string{}
		
		for colName, count := range allColumns {
			if count == len(dbs) {
				commonCols = append(commonCols, colName)
			} else {
				uniqueCols = append(uniqueCols, colName)
			}
		}

		fmt.Printf("  ✅ Общих колонок: %d\n", len(commonCols))
		fmt.Printf("  ⚠️  Уникальных колонок: %d\n", len(uniqueCols))
		
		if len(uniqueCols) > 0 && len(uniqueCols) <= 10 {
			fmt.Println("  Уникальные колонки:")
			for _, col := range uniqueCols {
				count := allColumns[col]
				fmt.Printf("    - %s (в %d из %d БД)\n", col, count, len(dbs))
			}
		}
		
		fmt.Println()
	}
}

