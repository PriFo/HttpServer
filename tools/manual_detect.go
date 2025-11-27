//go:build tool_manual_detect
// +build tool_manual_detect

package main

import (
	"fmt"
	"log"

	"httpserver/database"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("=== РУЧНОЙ ЗАПУСК ДЕТЕКТОРА ===\n")

	// Открываем service.db
	serviceDB, err := database.NewServiceDB("data/service.db")
	if err != nil {
		log.Fatalf("Failed to open service.db: %v", err)
	}
	defer serviceDB.Close()

	// Получаем список БД из project_databases
	query := `
		SELECT pd.id, pd.name, pd.file_path, cp.name as project_name
		FROM project_databases pd
		JOIN client_projects cp ON pd.client_project_id = cp.id
		WHERE cp.client_id = 1
		ORDER BY pd.id
	`

	rows, err := serviceDB.Query(query)
	if err != nil {
		log.Fatalf("Failed to query databases: %v", err)
	}
	defer rows.Close()

	type DatabaseInfo struct {
		ID          int
		Name        string
		FilePath    string
		ProjectName string
	}

	var databases []DatabaseInfo
	for rows.Next() {
		var db DatabaseInfo
		if err := rows.Scan(&db.ID, &db.Name, &db.FilePath, &db.ProjectName); err != nil {
			log.Printf("Failed to scan: %v", err)
			continue
		}
		databases = append(databases, db)
	}

	fmt.Printf("Найдено БД: %d\n\n", len(databases))

	if len(databases) == 0 {
		fmt.Println("❌ Нет БД для анализа")
		return
	}

	// Создаем детектор
	detector := database.NewCounterpartyDetector(serviceDB)

	// Анализируем каждую БД
	for i, db := range databases {
		fmt.Printf("%d. БД: %s (ID: %d)\n", i+1, db.Name, db.ID)
		fmt.Printf("   Проект: %s\n", db.ProjectName)
		fmt.Printf("   Путь: %s\n", db.FilePath)

		// Проверяем кэш
		cached, err := detector.GetCachedMetadata(db.ID)
		if err == nil && cached != nil {
			fmt.Printf("   ✅ КЭШИРОВАНО (confidence: %.2f)\n", cached.Confidence)
			fmt.Printf("      Таблица: %s\n", cached.TableName)
			continue
		}

		// Детектим структуру
		structure, err := detector.DetectStructure(db.ID, db.FilePath)
		if err != nil {
			fmt.Printf("   ❌ ОШИБКА: %v\n\n", err)
			continue
		}

		fmt.Printf("   ✅ ОБНАРУЖЕНО (confidence: %.2f)\n", structure.Confidence)
		fmt.Printf("      Таблица: %s\n", structure.TableName)
		
		if structure.NameColumn != "" {
			fmt.Printf("      Наименование: %s\n", structure.NameColumn)
		}
		if structure.INNColumn != "" {
			fmt.Printf("      ИНН: %s\n", structure.INNColumn)
		}
		if structure.BINColumn != "" {
			fmt.Printf("      БИН: %s\n", structure.BINColumn)
		}
		if structure.OGRNColumn != "" {
			fmt.Printf("      ОГРН: %s\n", structure.OGRNColumn)
		}
		if structure.KPPColumn != "" {
			fmt.Printf("      КПП: %s\n", structure.KPPColumn)
		}
		if structure.AddressColumn != "" {
			fmt.Printf("      Адрес: %s\n", structure.AddressColumn)
		}
		
		fmt.Println()
	}

	// Проверяем, что сохранилось в БД
	fmt.Println("\n=== ПРОВЕРКА СОХРАНЕННЫХ МЕТАДАННЫХ ===\n")
	
	checkQuery := `
		SELECT database_id, table_name, detection_confidence
		FROM database_table_metadata
		WHERE entity_type = 'counterparty'
		ORDER BY database_id
	`
	
	checkRows, err := serviceDB.Query(checkQuery)
	if err != nil {
		log.Fatalf("Failed to check metadata: %v", err)
	}
	defer checkRows.Close()
	
	count := 0
	for checkRows.Next() {
		var dbID int
		var tableName string
		var confidence float64
		
		if err := checkRows.Scan(&dbID, &tableName, &confidence); err != nil {
			continue
		}
		
		count++
		fmt.Printf("%d. DB ID: %d, Table: %s, Confidence: %.2f\n", count, dbID, tableName, confidence)
	}
	
	if count == 0 {
		fmt.Println("❌ Нет сохраненных метаданных")
	} else {
		fmt.Printf("\n✅ Всего сохранено: %d записей\n", count)
	}
}

