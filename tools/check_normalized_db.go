//go:build tool_check_normalized_db
// +build tool_check_normalized_db

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := "data/normalized_data.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("❌ Файл %s не найден", dbPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения: %v", err)
	}
	defer db.Close()

	fmt.Println("Проверка normalized_data.db:")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Получаем список таблиц
	rows, err := db.Query(`
		SELECT name 
		FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nТаблицы в базе:")
	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
		fmt.Printf("  - %s\n", name)
	}

	if len(tables) == 0 {
		fmt.Println("  (нет таблиц)")
		return
	}

	// Проверяем normalized_data
	if contains(tables, "normalized_data") {
		var total int
		db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&total)
		fmt.Printf("\n📊 Всего записей в normalized_data: %d\n", total)

		if total > 0 {
			// Проверяем структуру
			colRows, _ := db.Query("PRAGMA table_info(normalized_data)")
			if colRows != nil {
				defer colRows.Close()
				fmt.Println("\nСтруктура таблицы:")
				for colRows.Next() {
					var cid int
					var name, dataType string
					var notNull, pk int
					var defaultValue sql.NullString
					colRows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
					fmt.Printf("  - %s (%s)\n", name, dataType)
				}
			}

			// Статистика по project_id если есть
			var hasProjectID bool
			db.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM pragma_table_info('normalized_data') 
					WHERE name='project_id'
				)
			`).Scan(&hasProjectID)

			if hasProjectID {
				fmt.Println("\nРаспределение по проектам:")
				projRows, _ := db.Query(`
					SELECT project_id, COUNT(*) as cnt
					FROM normalized_data
					GROUP BY project_id
					ORDER BY project_id
				`)
				if projRows != nil {
					defer projRows.Close()
					for projRows.Next() {
						var projID sql.NullInt64
						var cnt int
						projRows.Scan(&projID, &cnt)
						if projID.Valid {
							fmt.Printf("  - Проект %d: %d записей\n", projID.Int64, cnt)
						} else {
							fmt.Printf("  - (без проекта): %d записей\n", cnt)
						}
					}
				}
			}

			// Статистика по категориям
			fmt.Println("\nТоп-10 категорий:")
			catRows, _ := db.Query(`
				SELECT category, COUNT(*) as cnt
				FROM normalized_data
				GROUP BY category
				ORDER BY cnt DESC
				LIMIT 10
			`)
			if catRows != nil {
				defer catRows.Close()
				for catRows.Next() {
					var cat sql.NullString
					var cnt int
					catRows.Scan(&cat, &cnt)
					catStr := "(без категории)"
					if cat.Valid && cat.String != "" {
						catStr = cat.String
					}
					fmt.Printf("  - %s: %d записей\n", catStr, cnt)
				}
			}
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

