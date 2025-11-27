package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

	// Проверяем структуру таблицы
	fmt.Println("Структура таблицы kpved_classifier:")
	rows, err := db.Query("PRAGMA table_info(kpved_classifier)")
	if err != nil {
		log.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		pkStr := ""
		if pk == 1 {
			pkStr = " (PK)"
		}
		fmt.Printf("  %s: %s%s\n", name, dataType, pkStr)
	}

	// Проверяем индексы
	fmt.Println("\nИндексы:")
	indexRows, err := db.Query(`
		SELECT name, sql 
		FROM sqlite_master 
		WHERE type='index' AND tbl_name='kpved_classifier'
	`)
	if err == nil {
		defer indexRows.Close()
		for indexRows.Next() {
			var name, sql sql.NullString
			if err := indexRows.Scan(&name, &sql); err == nil {
				fmt.Printf("  %s\n", name.String)
			}
		}
	}

	// Проверяем несколько записей с разными уровнями
	fmt.Println("\nПримеры записей по уровням:")

	// Уровень 0 (секции)
	fmt.Println("\nУровень 0 (секции):")
	rows0, err := db.Query(`
		SELECT code, name 
		FROM kpved_classifier 
		WHERE level = 0 
		ORDER BY code 
		LIMIT 5
	`)
	if err == nil {
		defer rows0.Close()
		for rows0.Next() {
			var code, name string
			if err := rows0.Scan(&code, &name); err == nil {
				if len(name) > 50 {
					name = name[:50] + "..."
				}
				fmt.Printf("  %s: %s\n", code, name)
			}
		}
	}

	// Уровень 1
	fmt.Println("\nУровень 1 (примеры):")
	rows1, err := db.Query(`
		SELECT code, name, parent_code 
		FROM kpved_classifier 
		WHERE level = 1 
		ORDER BY code 
		LIMIT 5
	`)
	if err == nil {
		defer rows1.Close()
		for rows1.Next() {
			var code, name, parentCode sql.NullString
			if err := rows1.Scan(&code, &name, &parentCode); err == nil {
				parent := "-"
				if parentCode.Valid {
					parent = parentCode.String
				}
				nameStr := name.String
				if len(nameStr) > 40 {
					nameStr = nameStr[:40] + "..."
				}
				fmt.Printf("  %s (родитель: %s): %s\n", code.String, parent, nameStr)
			}
		}
	}

	// Уровень 3 (детальные коды)
	fmt.Println("\nУровень 3 (детальные коды, примеры):")
	rows3, err := db.Query(`
		SELECT code, name, parent_code 
		FROM kpved_classifier 
		WHERE level = 3 
		ORDER BY code 
		LIMIT 5
	`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var code, name, parentCode sql.NullString
			if err := rows3.Scan(&code, &name, &parentCode); err == nil {
				parent := "-"
				if parentCode.Valid {
					parent = parentCode.String
				}
				nameStr := name.String
				if len(nameStr) > 50 {
					nameStr = nameStr[:50] + "..."
				}
				fmt.Printf("  %s (родитель: %s): %s\n", code.String, parent, nameStr)
			}
		}
	}

	// Проверяем целостность данных
	fmt.Println("\nПроверка целостности данных:")

	// Записи без родителя на уровнях > 0
	var orphanCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM kpved_classifier 
		WHERE level > 0 AND (parent_code IS NULL OR parent_code = '')
	`).Scan(&orphanCount)
	if err == nil {
		if orphanCount == 0 {
			fmt.Printf("  ✓ Все записи уровней > 0 имеют родителя\n")
		} else {
			fmt.Printf("  ⚠ Найдено %d записей без родителя на уровнях > 0\n", orphanCount)
		}
	}

	// Дубликаты кодов
	var duplicateCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM (
			SELECT code, COUNT(*) as cnt 
			FROM kpved_classifier 
			GROUP BY code 
			HAVING cnt > 1
		)
	`).Scan(&duplicateCount)
	if err == nil {
		if duplicateCount == 0 {
			fmt.Printf("  ✓ Дубликатов кодов нет\n")
		} else {
			fmt.Printf("  ⚠ Найдено %d дубликатов кодов\n", duplicateCount)
		}
	}

	fmt.Println("\n✓ Проверка завершена")
}

