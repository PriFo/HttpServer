//go:build tool_list_all_tables
// +build tool_list_all_tables

package main

import (
	"fmt"
	"log"

	"httpserver/database"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("=== СПИСОК ВСЕХ ТАБЛИЦ В service.db ===\n")

	serviceDB, err := database.NewServiceDB("data/service.db")
	if err != nil {
		log.Fatalf("Failed to open service.db: %v", err)
	}
	defer serviceDB.Close()

	// Получаем все таблицы
	query := `
		SELECT name, sql FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := serviceDB.Query(query)
	if err != nil {
		log.Fatalf("Failed to query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err != nil {
			continue
		}
		
		fmt.Printf("📋 Таблица: %s\n", name)
		
		// Подсчитываем записи
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", name)
		var count int
		serviceDB.QueryRow(countQuery).Scan(&count)
		fmt.Printf("   Записей: %d\n\n", count)
	}
}

