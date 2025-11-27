//go:build tool_list_clients
// +build tool_list_clients

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	serviceDBPath := "data/service.db"
	if _, err := os.Stat(serviceDBPath); os.IsNotExist(err) {
		log.Fatalf("❌ Файл service.db не найден: %s", serviceDBPath)
	}

	serviceDB, err := sql.Open("sqlite3", serviceDBPath)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения: %v", err)
	}
	defer serviceDB.Close()

	fmt.Println("Список клиентов:")
	fmt.Println("═══════════════════════════════════════════════════════════")

	rows, err := serviceDB.Query(`
		SELECT id, name, legal_name 
		FROM clients 
		ORDER BY id
	`)
	if err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name, legalName sql.NullString
		rows.Scan(&id, &name, &legalName)
		
		count++
		fmt.Printf("%d. ID: %d\n", count, id)
		if name.Valid {
			fmt.Printf("   Имя: %s\n", name.String)
		}
		if legalName.Valid {
			fmt.Printf("   Юр. имя: %s\n", legalName.String)
		}
		
		// Проверяем проекты этого клиента
		projRows, _ := serviceDB.Query(`
			SELECT id, name 
			FROM client_projects 
			WHERE client_id = ?
			ORDER BY id
		`, id)
		
		projCount := 0
		for projRows.Next() {
			var projID int
			var projName string
			if err := projRows.Scan(&projID, &projName); err == nil {
				if projCount == 0 {
					fmt.Printf("   Проекты:\n")
				}
				fmt.Printf("      - [%d] %s\n", projID, projName)
				projCount++
			}
		}
		projRows.Close()
		
		fmt.Println()
	}

	if count == 0 {
		fmt.Println("Нет клиентов в базе")
	}
}

