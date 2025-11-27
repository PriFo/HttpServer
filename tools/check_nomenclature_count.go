package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		dbPath       = flag.String("db", "", "Path to database file")
		clientID     = flag.Int("client", 0, "Client ID")
		projectID    = flag.Int("project", 0, "Project ID (optional)")
		showDetails  = flag.Bool("details", false, "Show detailed breakdown")
	)
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("Database path is required. Use -db flag")
	}

	if *clientID == 0 {
		log.Fatal("Client ID is required. Use -client flag")
	}

	// Открываем базу данных
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("Checking nomenclature in database: %s\n", *dbPath)
	fmt.Printf("Client ID: %d\n", *clientID)
	if *projectID > 0 {
		fmt.Printf("Project ID: %d\n", *projectID)
	}
	fmt.Println(strings.Repeat("=", 80))

	// Проверяем наличие таблиц
	var hasNormalizedData, hasCatalogItems, hasNomenclatureItems bool
	var normalizedCount, catalogItemsCount, nomenclatureItemsCount int

	// Проверка normalized_data
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='normalized_data'").Scan(&hasNormalizedData)
	if err == nil && hasNormalizedData {
		query := "SELECT COUNT(*) FROM normalized_data WHERE project_id = ?"
		args := []interface{}{*projectID}
		if *projectID == 0 {
			query = "SELECT COUNT(*) FROM normalized_data"
			args = []interface{}{}
		}
		err = db.QueryRow(query, args...).Scan(&normalizedCount)
		if err != nil {
			log.Printf("Error counting normalized_data: %v", err)
		}
	}

	// Проверка catalog_items
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='catalog_items'").Scan(&hasCatalogItems)
	if err == nil && hasCatalogItems {
		// Нужно проверить через uploads
		query := `
			SELECT COUNT(*) FROM catalog_items ci
			INNER JOIN catalogs c ON ci.catalog_id = c.id
			INNER JOIN uploads u ON c.upload_id = u.id
			WHERE u.client_id = ?
		`
		args := []interface{}{*clientID}
		if *projectID > 0 {
			query += " AND u.project_id = ?"
			args = append(args, *projectID)
		}
		err = db.QueryRow(query, args...).Scan(&catalogItemsCount)
		if err != nil {
			log.Printf("Error counting catalog_items: %v", err)
		}
	}

	// Проверка nomenclature_items
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='nomenclature_items'").Scan(&hasNomenclatureItems)
	if err == nil && hasNomenclatureItems {
		query := `
			SELECT COUNT(*) FROM nomenclature_items ni
			INNER JOIN uploads u ON ni.upload_id = u.id
			WHERE u.client_id = ?
		`
		args := []interface{}{*clientID}
		if *projectID > 0 {
			query += " AND u.project_id = ?"
			args = append(args, *projectID)
		}
		err = db.QueryRow(query, args...).Scan(&nomenclatureItemsCount)
		if err != nil {
			log.Printf("Error counting nomenclature_items: %v", err)
		}
	}

	// Выводим результаты
	fmt.Println("\nTable Status:")
	fmt.Printf("  normalized_data:     %v (count: %d)\n", hasNormalizedData, normalizedCount)
	fmt.Printf("  catalog_items:        %v (count: %d)\n", hasCatalogItems, catalogItemsCount)
	fmt.Printf("  nomenclature_items:   %v (count: %d)\n", hasNomenclatureItems, nomenclatureItemsCount)

	totalFromMain := catalogItemsCount + nomenclatureItemsCount
	totalAll := normalizedCount + totalFromMain

	fmt.Println("\nSummary:")
	fmt.Printf("  Normalized items:     %d\n", normalizedCount)
	fmt.Printf("  Main DB items:       %d\n", totalFromMain)
	fmt.Printf("    - catalog_items:    %d\n", catalogItemsCount)
	fmt.Printf("    - nomenclature_items: %d\n", nomenclatureItemsCount)
	fmt.Printf("  TOTAL:                %d\n", totalAll)

	// Детальная информация
	if *showDetails {
		fmt.Println("\nDetailed Breakdown:")

		if hasNormalizedData && normalizedCount > 0 {
			fmt.Println("\nNormalized items:")
			query := "SELECT id, code, normalized_name, category FROM normalized_data LIMIT 10"
			rows, err := db.Query(query)
			if err == nil {
				for rows.Next() {
					var id int
					var code, name, category string
					if err := rows.Scan(&id, &code, &name, &category); err == nil {
						fmt.Printf("  [%d] %s: %s (%s)\n", id, code, name, category)
					}
				}
				rows.Close()
				if normalizedCount > 10 {
					fmt.Printf("  ... and %d more\n", normalizedCount-10)
				}
			}
		}

		if hasCatalogItems && catalogItemsCount > 0 {
			fmt.Println("\nCatalog items:")
			query := `
				SELECT ci.id, ci.code, ci.name
				FROM catalog_items ci
				INNER JOIN catalogs c ON ci.catalog_id = c.id
				INNER JOIN uploads u ON c.upload_id = u.id
				WHERE u.client_id = ?
				LIMIT 10
			`
			rows, err := db.Query(query, *clientID)
			if err == nil {
				for rows.Next() {
					var id int
					var code, name string
					if err := rows.Scan(&id, &code, &name); err == nil {
						fmt.Printf("  [%d] %s: %s\n", id, code, name)
					}
				}
				rows.Close()
				if catalogItemsCount > 10 {
					fmt.Printf("  ... and %d more\n", catalogItemsCount-10)
				}
			}
		}

		if hasNomenclatureItems && nomenclatureItemsCount > 0 {
			fmt.Println("\nNomenclature items:")
			query := `
				SELECT id, nomenclature_code, nomenclature_name
				FROM nomenclature_items ni
				INNER JOIN uploads u ON ni.upload_id = u.id
				WHERE u.client_id = ?
				LIMIT 10
			`
			rows, err := db.Query(query, *clientID)
			if err == nil {
				for rows.Next() {
					var id int
					var code, name string
					if err := rows.Scan(&id, &code, &name); err == nil {
						fmt.Printf("  [%d] %s: %s\n", id, code, name)
					}
				}
				rows.Close()
				if nomenclatureItemsCount > 10 {
					fmt.Printf("  ... and %d more\n", nomenclatureItemsCount-10)
				}
			}
		}
	}

	// Важное замечание
	if hasNormalizedData && totalFromMain > 0 {
		fmt.Println("\n⚠️  IMPORTANT:")
		fmt.Println("  This database contains BOTH normalized_data AND source tables.")
		fmt.Println("  After the fix, ALL items should be shown:")
		fmt.Printf("    - %d normalized items (from normalized_data table)\n", normalizedCount)
		fmt.Printf("    - %d source items (from catalog_items/nomenclature_items)\n", totalFromMain)
		fmt.Printf("    - Total: %d items\n", totalAll)
	}
}

