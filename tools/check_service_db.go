//go:build tool_check_service_db
// +build tool_check_service_db

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

	fmt.Println("🔍 Таблицы в service.db:")
	fmt.Println("═══════════════════════════════════════")

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
		fmt.Printf("  • %s\n", name)
	}

	fmt.Printf("\nВсего таблиц: %d\n", len(tables))
}

