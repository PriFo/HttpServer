package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"httpserver/database"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var dbPath string
	var projectID int
	var clientID int

	flag.StringVar(&dbPath, "db", "", "Путь к базе данных проекта")
	flag.IntVar(&projectID, "project", 0, "ID проекта")
	flag.IntVar(&clientID, "client", 0, "ID клиента")
	flag.Parse()

	if dbPath == "" {
		fmt.Println("Использование: check_data_chain -db <путь_к_бд> -project <project_id> -client <client_id>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Проверяем существование файла
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("Файл базы данных не найден: %s", dbPath)
	}

	fmt.Printf("Проверка цепочки данных для БД: %s\n", dbPath)
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	// Открываем исходную базу данных
	sourceDB, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Не удалось открыть БД: %v", err)
	}
	defer sourceDB.Close()

	// Шаг 1: Проверка наличия таблиц
	fmt.Println("\n1. Проверка наличия таблиц:")
	checkTables(sourceDB)

	// Шаг 2: Проверка upload записей
	fmt.Println("\n2. Проверка upload записей:")
	uploads, err := checkUploads(sourceDB, clientID, projectID)
	if err != nil {
		log.Printf("Ошибка при проверке upload записей: %v", err)
	}

	// Шаг 3: Проверка catalog_items
	fmt.Println("\n3. Проверка catalog_items:")
	checkCatalogItems(sourceDB, uploads)

	// Шаг 4: Проверка nomenclature_items
	fmt.Println("\n4. Проверка nomenclature_items:")
	checkNomenclatureItems(sourceDB, uploads)

	// Шаг 5: Итоговая статистика
	fmt.Println("\n5. Итоговая статистика:")
	printSummary(sourceDB, uploads)
}

func checkTables(db *database.DB) {
	tables := []string{"uploads", "catalogs", "catalog_items", "nomenclature_items"}

	for _, table := range tables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name=?
			)
		`, table).Scan(&exists)

		if err != nil {
			fmt.Printf("  ❌ %s: ошибка проверки - %v\n", table, err)
			continue
		}

		if exists {
			var count int
			db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
			fmt.Printf("  ✅ %s: существует, записей: %d\n", table, count)
		} else {
			fmt.Printf("  ❌ %s: не существует\n", table)
		}
	}
}

func checkUploads(db *database.DB, clientID, projectID int) ([]*database.Upload, error) {
	uploads, err := db.GetAllUploads()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить upload записи: %w", err)
	}

	fmt.Printf("  Всего upload записей: %d\n", len(uploads))

	if len(uploads) == 0 {
		fmt.Println("  ⚠️  Нет upload записей - данные не будут найдены через getNomenclatureFromMainDB")
		return uploads, nil
	}

	correctCount := 0
	for _, upload := range uploads {
		hasClientID := upload.ClientID != nil && *upload.ClientID == clientID
		hasProjectID := upload.ProjectID != nil && *upload.ProjectID == projectID

		if hasClientID && hasProjectID {
			correctCount++
			fmt.Printf("  ✅ Upload %d: client_id=%d, project_id=%d (правильно)\n", upload.ID, *upload.ClientID, *upload.ProjectID)
		} else {
			fmt.Printf("  ⚠️  Upload %d: client_id=%v, project_id=%v (требует обновления)\n", upload.ID, upload.ClientID, upload.ProjectID)
		}
	}

	if correctCount == 0 {
		fmt.Println("  ⚠️  Нет upload записей с правильными client_id и project_id")
	}

	return uploads, nil
}

func checkCatalogItems(db *database.DB, uploads []*database.Upload) {
	if len(uploads) == 0 {
		fmt.Println("  ⚠️  Нет upload записей для проверки catalog_items")
		return
	}

	totalItems := 0
	for _, upload := range uploads {
		items, _, err := db.GetCatalogItemsByUpload(upload.ID, nil, 0, 0)
		if err != nil {
			fmt.Printf("  ⚠️  Upload %d: ошибка получения items - %v\n", upload.ID, err)
			continue
		}

		itemCount := len(items)
		totalItems += itemCount
		fmt.Printf("  Upload %d: %d catalog_items\n", upload.ID, itemCount)

		// Проверяем каталоги
		catalogs, err := db.GetCatalogsByUpload(upload.ID)
		if err == nil {
			for _, catalog := range catalogs {
				fmt.Printf("    - Каталог: %s (synonym: %s)\n", catalog.Name, catalog.Synonym)
			}
		}
	}

	fmt.Printf("  Всего catalog_items: %d\n", totalItems)
}

func checkNomenclatureItems(db *database.DB, uploads []*database.Upload) {
	if len(uploads) == 0 {
		fmt.Println("  ⚠️  Нет upload записей для проверки nomenclature_items")
		return
	}

	totalItems := 0
	for _, upload := range uploads {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM nomenclature_items 
			WHERE upload_id = ?
		`, upload.ID).Scan(&count)

		if err != nil {
			// Таблица может не существовать
			continue
		}

		totalItems += count
		if count > 0 {
			fmt.Printf("  Upload %d: %d nomenclature_items\n", upload.ID, count)
		}
	}

	if totalItems > 0 {
		fmt.Printf("  Всего nomenclature_items: %d\n", totalItems)
	} else {
		fmt.Println("  Нет nomenclature_items")
	}
}

func printSummary(db *database.DB, uploads []*database.Upload) {
	// Подсчитываем общее количество данных
	var totalCatalogItems int
	var totalNomenclatureItems int

	for _, upload := range uploads {
		items, _, err := db.GetCatalogItemsByUpload(upload.ID, nil, 0, 0)
		if err == nil {
			totalCatalogItems += len(items)
		}

		var count int
		db.QueryRow(`SELECT COUNT(*) FROM nomenclature_items WHERE upload_id = ?`, upload.ID).Scan(&count)
		totalNomenclatureItems += count
	}

	fmt.Printf("  Upload записей: %d\n", len(uploads))
	fmt.Printf("  Catalog items: %d\n", totalCatalogItems)
	fmt.Printf("  Nomenclature items: %d\n", totalNomenclatureItems)
	fmt.Printf("  Всего записей: %d\n", totalCatalogItems+totalNomenclatureItems)

	if len(uploads) == 0 {
		fmt.Println("\n⚠️  ПРОБЛЕМА: Нет upload записей!")
		fmt.Println("   Решение: Запустите ensureUploadRecordsForDatabase при добавлении БД")
	} else if totalCatalogItems == 0 && totalNomenclatureItems == 0 {
		fmt.Println("\n⚠️  ПРОБЛЕМА: Нет данных в catalog_items и nomenclature_items!")
		fmt.Println("   Решение: Проверьте, что данные были импортированы в БД")
	} else {
		fmt.Println("\n✅ Данные найдены и готовы к извлечению")
	}
}

