//go:build tool_check_project_databases
// +build tool_check_project_databases

package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 БАЗЫ ДАННЫХ ПРОЕКТА AITAS (ID: 1):")
	fmt.Println("═══════════════════════════════════════")

	rows, err := db.Query(`
		SELECT id, name, file_path, description, file_size, is_active
		FROM project_databases
		WHERE client_project_id = 1
		ORDER BY name
	`)
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}
	defer rows.Close()

	type Database struct {
		ID          int
		Name        string
		FilePath    string
		Description sql.NullString
		FileSize    sql.NullInt64
		IsActive    bool
	}

	var databases []Database
	for rows.Next() {
		var db Database
		err := rows.Scan(&db.ID, &db.Name, &db.FilePath, &db.Description, &db.FileSize, &db.IsActive)
		if err != nil {
			log.Printf("Ошибка: %v", err)
			continue
		}
		databases = append(databases, db)
	}

	fmt.Printf("\nВсего БД: %d\n\n", len(databases))

	totalNomenclature := 0
	totalCounterparty := 0

	for _, database := range databases {
		status := "✅"
		if !database.IsActive {
			status = "❌"
		}

		fmt.Printf("%s %d. %s\n", status, database.ID, database.Name)
		fmt.Printf("   Путь: %s\n", filepath.Base(database.FilePath))
		if database.FileSize.Valid {
			fmt.Printf("   Размер: %d байт\n", database.FileSize.Int64)
		}

		// Проверяем, какой тип данных в БД
		dbPath := database.FilePath
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join("data", dbPath)
		}

		conn, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			fmt.Printf("   ⚠️  Ошибка открытия: %v\n", err)
			fmt.Println()
			continue
		}

		// Определяем тип данных
		dataType := ""
		var count int

		// Проверяем nomenclature_items
		var hasNomenclature bool
		conn.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='nomenclature_items')").Scan(&hasNomenclature)
		if hasNomenclature {
			conn.QueryRow("SELECT COUNT(*) FROM nomenclature_items").Scan(&count)
			if count > 0 {
				dataType = "nomenclature"
				totalNomenclature += count
			}
		}

		// Проверяем counterparties
		var hasCounterparties bool
		conn.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='counterparties')").Scan(&hasCounterparties)
		if hasCounterparties {
			conn.QueryRow("SELECT COUNT(*) FROM counterparties").Scan(&count)
			if count > 0 {
				dataType = "counterparty"
				totalCounterparty += count
			}
		}

		// Проверяем catalog_items
		if dataType == "" {
			var hasCatalogItems bool
			conn.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='catalog_items')").Scan(&hasCatalogItems)
			if hasCatalogItems {
				conn.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&count)
				if count > 0 {
					// Определяем тип по названию файла
					if filepath.Base(database.FilePath) == "" ||
						len(filepath.Base(database.FilePath)) < 10 {
						dataType = "unknown"
					} else {
						fileName := filepath.Base(database.FilePath)
						if len(fileName) > 15 && fileName[8:19] == "Номенклатура" {
							dataType = "nomenclature"
							totalNomenclature += count
						} else if len(fileName) > 15 && fileName[8:19] == "Контрагенты" {
							dataType = "counterparty"
							totalCounterparty += count
						} else {
							dataType = "unknown"
						}
					}
				}
			}
		}

		conn.Close()

		if dataType != "" {
			fmt.Printf("   Тип: %s\n", dataType)
			fmt.Printf("   Записей: %d\n", count)
		} else {
			fmt.Printf("   Тип: неизвестно\n")
		}
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("\n📊 ИТОГОВАЯ СТАТИСТИКА:\n")
	fmt.Printf("   • Номенклатура: %d записей\n", totalNomenclature)
	fmt.Printf("   • Контрагенты: %d записей\n", totalCounterparty)
	fmt.Printf("   • ВСЕГО: %d записей\n", totalNomenclature+totalCounterparty)
	fmt.Println()
}

