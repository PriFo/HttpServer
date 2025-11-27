//go:build tool_check_db_content
// +build tool_check_db_content

package main

import (
	"fmt"
	"log"

	"httpserver/database"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("=== ПРОВЕРКА СОДЕРЖИМОГО service.db ===\n")

	serviceDB, err := database.NewServiceDB("data/service.db")
	if err != nil {
		log.Fatalf("Failed to open service.db: %v", err)
	}
	defer serviceDB.Close()

	// 1. Проверяем клиентов
	fmt.Println("1. КЛИЕНТЫ:")
	clientsQuery := "SELECT id, name FROM clients ORDER BY id LIMIT 10"
	rows, _ := serviceDB.Query(clientsQuery)
	defer rows.Close()
	
	clientCount := 0
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		fmt.Printf("   - ID: %d, Name: %s\n", id, name)
		clientCount++
	}
	fmt.Printf("   Всего: %d\n\n", clientCount)

	// 2. Проверяем проекты
	fmt.Println("2. ПРОЕКТЫ:")
	projectsQuery := "SELECT id, name, client_id FROM client_projects ORDER BY id LIMIT 10"
	rows2, _ := serviceDB.Query(projectsQuery)
	defer rows2.Close()
	
	projectCount := 0
	for rows2.Next() {
		var id, clientID int
		var name string
		rows2.Scan(&id, &name, &clientID)
		fmt.Printf("   - ID: %d, Name: %s, Client ID: %d\n", id, name, clientID)
		projectCount++
	}
	fmt.Printf("   Всего: %d\n\n", projectCount)

	// 3. Проверяем БД
	fmt.Println("3. БАЗЫ ДАННЫХ:")
	dbsQuery := "SELECT id, name, client_project_id, file_path FROM project_databases ORDER BY id LIMIT 10"
	rows3, _ := serviceDB.Query(dbsQuery)
	defer rows3.Close()
	
	dbCount := 0
	for rows3.Next() {
		var id, projectID int
		var name, filePath string
		rows3.Scan(&id, &name, &projectID, &filePath)
		fmt.Printf("   - ID: %d, Name: %s, Project ID: %d\n", id, name, projectID)
		fmt.Printf("     Path: %s\n", filePath)
		dbCount++
	}
	fmt.Printf("   Всего: %d\n\n", dbCount)

	// 4. Проверяем метаданные
	fmt.Println("4. МЕТАДАННЫЕ:")
	metaQuery := "SELECT id, database_id, table_name, detection_confidence FROM database_table_metadata LIMIT 10"
	rows4, _ := serviceDB.Query(metaQuery)
	defer rows4.Close()
	
	metaCount := 0
	for rows4.Next() {
		var id, dbID int
		var tableName string
		var confidence float64
		rows4.Scan(&id, &dbID, &tableName, &confidence)
		fmt.Printf("   - ID: %d, DB ID: %d, Table: %s, Confidence: %.2f\n", id, dbID, tableName, confidence)
		metaCount++
	}
	fmt.Printf("   Всего: %d\n", metaCount)
}

