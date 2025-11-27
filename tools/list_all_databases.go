//go:build tool_list_all_databases
// +build tool_list_all_databases

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

	fmt.Println("🔍 ВСЕ БАЗЫ ДАННЫХ В СИСТЕМЕ:")
	fmt.Println("═══════════════════════════════════════")

	rows, err := db.Query(`
		SELECT pd.id, pd.name, pd.file_path, cp.name as project_name, c.name as client_name
		FROM project_databases pd
		JOIN client_projects cp ON pd.client_project_id = cp.id
		JOIN clients c ON cp.client_id = c.id
		ORDER BY pd.id
	`)
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name, filePath, projectName, clientName string
		rows.Scan(&id, &name, &filePath, &projectName, &clientName)
		
		count++
		fmt.Printf("\n%d. %s\n", id, name)
		fmt.Printf("   Проект: %s\n", projectName)
		fmt.Printf("   Клиент: %s\n", clientName)
		fmt.Printf("   Путь: %s\n", filePath)
	}

	fmt.Printf("\n═══════════════════════════════════════\n")
	fmt.Printf("Всего БД: %d\n", count)
}

