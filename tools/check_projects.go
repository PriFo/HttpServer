//go:build tool_check_projects
// +build tool_check_projects

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

	fmt.Println("🔍 СПИСОК ПРОЕКТОВ:")
	fmt.Println("═══════════════════════════════════════")

	rows, err := db.Query(`
		SELECT p.id, p.name, c.id, c.name, p.description
		FROM client_projects p
		JOIN clients c ON p.client_id = c.id
		ORDER BY p.id
	`)
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var projectID, clientID int
		var projectName, clientName, description string
		rows.Scan(&projectID, &projectName, &clientID, &clientName, &description)
		fmt.Printf("\nID: %d\n", projectID)
		fmt.Printf("  Название: %s\n", projectName)
		fmt.Printf("  Клиент: %s (ID: %d)\n", clientName, clientID)
		fmt.Printf("  Описание: %s\n", description)
	}

	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("\n🔍 СТРУКТУРА ТАБЛИЦЫ project_databases:")
	fmt.Println("═══════════════════════════════════════")

	rows2, err := db.Query("PRAGMA table_info(project_databases)")
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var cid int
		var name, type_, notnull, dfltValue, pk string
		rows2.Scan(&cid, &name, &type_, &notnull, &dfltValue, &pk)
		fmt.Printf("  • %s (%s)\n", name, type_)
	}
}

