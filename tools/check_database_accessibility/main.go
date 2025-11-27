package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     🔍 ДЕТАЛЬНАЯ ПРОВЕРКА ДОСТУПНОСТИ БАЗ ДАННЫХ          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Подключение к service.db
	serviceDB, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к service.db: %v", err)
	}
	defer serviceDB.Close()

	log.Println("✅ Подключено к service.db")

	projectID := 1 // AITAS-MDM-2025-001

	// Получаем список баз данных
	rows, err := serviceDB.Query(`
		SELECT id, name, file_path, is_active
		FROM project_databases 
		WHERE client_project_id = ? AND is_active = 1
		ORDER BY name
	`, projectID)

	if err != nil {
		log.Fatalf("❌ Ошибка получения баз данных: %v", err)
	}
	defer rows.Close()

	type Database struct {
		ID       int
		Name     string
		FilePath string
		IsActive bool
	}

	var databases []Database
	for rows.Next() {
		var db Database
		err := rows.Scan(&db.ID, &db.Name, &db.FilePath, &db.IsActive)
		if err != nil {
			log.Printf("⚠️  Ошибка чтения БД: %v", err)
			continue
		}
		databases = append(databases, db)
	}

	fmt.Printf("📁 ПРОВЕРКА %d БАЗ ДАННЫХ:\n\n", len(databases))

	accessibleCount := 0
	inaccessibleCount := 0
	totalRecords := 0

	for i, db := range databases {
		fmt.Printf("%d. %s [ID: %d]\n", i+1, db.Name, db.ID)
		fmt.Printf("   Путь: %s\n", filepath.Base(db.FilePath))

		// Формируем полный путь
		dbPath := filepath.Join("data", db.FilePath)
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(".", dbPath)
		}

		// Проверка 1: Существование файла
		fmt.Printf("   [1] Файл существует: ")
		if _, err := sql.Open("sqlite3", dbPath); err != nil {
			fmt.Printf("❌ Ошибка пути: %v\n", err)
			inaccessibleCount++
			fmt.Println()
			continue
		}

		// Проверка 2: Открытие базы данных
		fmt.Printf("✅\n")
		fmt.Printf("   [2] Открытие БД: ")

		conn, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			inaccessibleCount++
			fmt.Println()
			continue
		}

		// Проверка 3: Подключение работает
		if err := conn.Ping(); err != nil {
			fmt.Printf("❌ Не удалось подключиться: %v\n", err)
			conn.Close()
			inaccessibleCount++
			fmt.Println()
			continue
		}
		fmt.Printf("✅\n")

		// Проверка 4: Проверка таблиц
		fmt.Printf("   [3] Таблицы: ")

		var tableNames []string
		tableRows, err := conn.Query(`
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name NOT LIKE 'sqlite_%'
			ORDER BY name
		`)
		if err != nil {
			fmt.Printf("❌ Ошибка запроса таблиц: %v\n", err)
			conn.Close()
			inaccessibleCount++
			fmt.Println()
			continue
		}

		for tableRows.Next() {
			var name string
			tableRows.Scan(&name)
			tableNames = append(tableNames, name)
		}
		tableRows.Close()

		if len(tableNames) == 0 {
			fmt.Printf("⚠️  Нет таблиц\n")
		} else {
			fmt.Printf("✅ %d таблиц: %v\n", len(tableNames), tableNames)
		}

		// Проверка 5: Подсчет записей
		fmt.Printf("   [4] Записи: ")

		var count int
		hasData := false

		// Пробуем разные таблицы
		tables := []string{"nomenclature_items", "counterparties", "catalog_items"}
		for _, table := range tables {
			var exists bool
			conn.QueryRow(fmt.Sprintf(
				"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='%s')",
				table)).Scan(&exists)
			if exists {
				conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
				if count > 0 {
					hasData = true
					break
				}
			}
		}

		if hasData {
			fmt.Printf("✅ %d записей\n", count)
			totalRecords += count
		} else {
			fmt.Printf("⚠️  Нет данных\n")
		}

		// Проверка 6: Чтение образца данных
		fmt.Printf("   [5] Чтение данных: ")

		var sampleData []string
		hasSample := false

		for _, table := range tables {
			var exists bool
			conn.QueryRow(fmt.Sprintf(
				"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='%s')",
				table)).Scan(&exists)
			if exists {
				var testCount int
				conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s LIMIT 1", table)).Scan(&testCount)
				if testCount > 0 {
					var name, code string
					err := conn.QueryRow(fmt.Sprintf("SELECT name, code FROM %s LIMIT 1", table)).Scan(&name, &code)
					if err == nil {
						if name != "" && len(name) > 50 {
							name = name[:50] + "..."
						}
						if name != "" {
							sampleData = append(sampleData, fmt.Sprintf("name='%s'", name))
						}
						if code != "" {
							sampleData = append(sampleData, fmt.Sprintf("code='%s'", code))
						}
						hasSample = true
						break
					}
				}
			}
		}

		if hasSample {
			fmt.Printf("✅ %s\n", sampleData[0])
		} else {
			fmt.Printf("⚠️  Не удалось прочитать\n")
		}

		conn.Close()

		fmt.Printf("   📊 ИТОГ: ✅ ДОСТУПНА\n")
		accessibleCount++
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📊 ИТОГОВАЯ СТАТИСТИКА:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("✅ Доступных БД: %d\n", accessibleCount)
	if inaccessibleCount > 0 {
		fmt.Printf("❌ Недоступных БД: %d\n", inaccessibleCount)
	}
	fmt.Printf("📊 Всего записей: %d\n", totalRecords)
	fmt.Println()

	if accessibleCount == len(databases) {
		fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
		fmt.Println("║     ✅ ВСЕ БАЗЫ ДАННЫХ ДОСТУПНЫ!                           ║")
		fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	} else {
		fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
		fmt.Println("║     ⚠️  НЕКОТОРЫЕ БД НЕДОСТУПНЫ                             ║")
		fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	}
	fmt.Println()
}
