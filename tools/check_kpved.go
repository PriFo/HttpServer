package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := "service.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Проверяем наличие таблицы
	var tableExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='kpved_classifier'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Fatalf("Failed to check table existence: %v", err)
	}

	if !tableExists {
		fmt.Println("❌ Таблица kpved_classifier не существует")
		return
	}

	// Общее количество записей
	var total int
	err = db.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&total)
	if err != nil {
		log.Fatalf("Failed to count records: %v", err)
	}

	fmt.Printf("✓ Таблица kpved_classifier существует\n")
	fmt.Printf("✓ Всего записей: %d\n\n", total)

	// Статистика по уровням
	fmt.Println("Распределение по уровням:")
	rows, err := db.Query(`
		SELECT level, COUNT(*) as count
		FROM kpved_classifier
		GROUP BY level
		ORDER BY level
	`)
	if err != nil {
		log.Fatalf("Failed to query levels: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var level, count int
		if err := rows.Scan(&level, &count); err != nil {
			continue
		}
		fmt.Printf("  Уровень %d: %d записей\n", level, count)
	}

	// Максимальный уровень
	var maxLevel int
	err = db.QueryRow("SELECT MAX(level) FROM kpved_classifier").Scan(&maxLevel)
	if err != nil {
		log.Fatalf("Failed to get max level: %v", err)
	}
	fmt.Printf("\nМаксимальный уровень: %d\n\n", maxLevel)

	// Примеры записей
	fmt.Println("Примеры записей (первые 10):")
	exampleRows, err := db.Query(`
		SELECT code, name, level, parent_code
		FROM kpved_classifier
		ORDER BY level, code
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Failed to query examples: %v", err)
	}
	defer exampleRows.Close()

	fmt.Printf("%-10s %-8s %-6s %-10s\n", "Код", "Уровень", "Родитель", "Название")
	fmt.Println(strings.Repeat("-", 80))
	for exampleRows.Next() {
		var code, name, parentCode sql.NullString
		var level int
		if err := exampleRows.Scan(&code, &name, &level, &parentCode); err != nil {
			continue
		}
		parent := "-"
		if parentCode.Valid {
			parent = parentCode.String
		}
		nameStr := name.String
		if len(nameStr) > 40 {
			nameStr = nameStr[:40] + "..."
		}
		fmt.Printf("%-10s %-8d %-10s %s\n", code.String, level, parent, nameStr)
	}

	// Проверяем наличие секций (уровень 1)
	var sectionsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM kpved_classifier WHERE level = 1").Scan(&sectionsCount)
	if err == nil {
		fmt.Printf("\n✓ Секций (уровень 1): %d\n", sectionsCount)
	}
}

