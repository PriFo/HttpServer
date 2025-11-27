//go:build tool_check_database_metadata
// +build tool_check_database_metadata

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}
	defer db.Close()

	// Проверяем структуру database_metadata
	fmt.Println("🔍 СТРУКТУРА ТАБЛИЦЫ database_metadata:")
	fmt.Println("═══════════════════════════════════════")

	rows, err := db.Query("PRAGMA table_info(database_metadata)")
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}
	
	for rows.Next() {
		var cid int
		var name, type_, notnull, dfltValue, pk string
		rows.Scan(&cid, &name, &type_, &notnull, &dfltValue, &pk)
		fmt.Printf("  • %s (%s)\n", name, type_)
	}
	rows.Close()

	fmt.Println("\n🔍 ЗАПИСИ В database_metadata:")
	fmt.Println("═══════════════════════════════════════")

	rows2, err := db.Query("SELECT * FROM database_metadata LIMIT 10")
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}

	// Получаем имена колонок
	cols, _ := rows2.Columns()
	fmt.Printf("Колонок: %d\n\n", len(cols))

	count := 0
	for rows2.Next() {
		count++
		// Создаем срез для сканирования
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		rows2.Scan(valuePtrs...)

		fmt.Printf("Запись %d:\n", count)
		for i, col := range cols {
			val := values[i]
			if val != nil {
				fmt.Printf("  %s: %v\n", col, val)
			}
		}
		fmt.Println()
	}
	rows2.Close()

	fmt.Printf("Всего записей: %d\n", count)
}
