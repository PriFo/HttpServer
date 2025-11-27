package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: check_counterparties <database_path>")
	}

	dbPath := os.Args[1]

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Проверяем список таблиц
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         АНАЛИЗ БАЗЫ ДАННЫХ КОНТРАГЕНТОВ                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	tables, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		log.Fatal(err)
	}
	defer tables.Close()

	fmt.Println("📋 Таблицы в базе данных:")
	var tableList []string
	for tables.Next() {
		var name string
		tables.Scan(&name)
		fmt.Printf("   - %s\n", name)
		tableList = append(tableList, name)
	}

	// Ищем таблицу с данными
	var dataTable string
	for _, table := range tableList {
		if table == "catalog_items" || table == "counterparties" {
			dataTable = table
			break
		}
	}

	if dataTable == "" {
		fmt.Println()
		fmt.Println("⚠ Не найдена таблица с данными")
		return
	}

	fmt.Println()
	fmt.Printf("📊 Используем таблицу: %s\n", dataTable)

	// Получаем структуру таблицы
	fmt.Println()
	fmt.Println("🔧 Структура таблицы:")
	pragma, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", dataTable))
	if err != nil {
		log.Fatal(err)
	}
	defer pragma.Close()

	type Column struct {
		ID      int
		Name    string
		Type    string
		NotNull int
		Default *string
		PK      int
	}

	var columns []Column
	for pragma.Next() {
		var col Column
		pragma.Scan(&col.ID, &col.Name, &col.Type, &col.NotNull, &col.Default, &col.PK)
		columns = append(columns, col)
		fmt.Printf("   - %-20s %s\n", col.Name, col.Type)
	}

	// Получаем примеры записей
	fmt.Println()
	fmt.Println("📝 Примеры записей (первые 3):")
	fmt.Println()

	query := fmt.Sprintf("SELECT * FROM %s LIMIT 3", dataTable)
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	colNames, _ := rows.Columns()

	recordNum := 1
	for rows.Next() {
		fmt.Printf("─────────────────────────────────────────────────────────────\n")
		fmt.Printf("Запись #%d:\n", recordNum)
		fmt.Printf("─────────────────────────────────────────────────────────────\n")

		// Создаем срез для сканирования
		values := make([]interface{}, len(colNames))
		valuePtrs := make([]interface{}, len(colNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		rows.Scan(valuePtrs...)

		// Выводим значения полей
		for i, colName := range colNames {
			val := values[i]
			if val == nil {
				fmt.Printf("   %-20s: NULL\n", colName)
			} else {
				switch v := val.(type) {
				case []byte:
					fmt.Printf("   %-20s: %s\n", colName, string(v))
				case string:
					if len(v) > 100 {
						fmt.Printf("   %-20s: %s...\n", colName, v[:100])
					} else {
						fmt.Printf("   %-20s: %s\n", colName, v)
					}
				default:
					fmt.Printf("   %-20s: %v\n", colName, v)
				}
			}
		}
		fmt.Println()
		recordNum++
	}

	// Подсчитываем общее количество
	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", dataTable)).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n✅ Всего записей в таблице: %d\n\n", count)
}
