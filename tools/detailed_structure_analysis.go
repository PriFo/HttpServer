//go:build tool_detailed_structure_analysis
// +build tool_detailed_structure_analysis

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("=== ДЕТАЛЬНЫЙ АНАЛИЗ СТРУКТУРЫ БД ===\n")

	// Берем первую БД
	dbPath := "data/uploads/Выгрузка_Контрагенты_БухгалтерияДляКазахстана_Unknown_Unknown_2025.db"

	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("База данных: %s\n\n", dbPath)

	// 1. Получаем структуру catalog_items
	fmt.Println("1. СТРУКТУРА ТАБЛИЦЫ catalog_items:")
	fmt.Println("   ┌──────────────────────────────────┬─────────────────┬──────────┐")
	fmt.Println("   │ Колонка                          │ Тип             │ NOT NULL │")
	fmt.Println("   ├──────────────────────────────────┼─────────────────┼──────────┤")

	rows, err := db.Query("PRAGMA table_info(catalog_items)")
	if err != nil {
		log.Fatalf("Failed to get table info: %v", err)
	}

	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err == nil {
			columns = append(columns, name)
			notNullStr := ""
			if notNull == 1 {
				notNullStr = "ДА"
			} else {
				notNullStr = "НЕТ"
			}
			fmt.Printf("   │ %-32s │ %-15s │ %-8s │\n", name, colType, notNullStr)
		}
	}
	rows.Close()
	fmt.Println("   └──────────────────────────────────┴─────────────────┴──────────┘")

	// 2. Показываем первые 3 записи
	fmt.Println("\n2. ПЕРВЫЕ 3 ЗАПИСИ (в JSON):")

	query := `SELECT * FROM catalog_items LIMIT 3`
	rows2, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}
	defer rows2.Close()

	colNames, _ := rows2.Columns()
	
	recordNum := 0
	for rows2.Next() {
		recordNum++
		
		// Создаем массив для сканирования
		values := make([]interface{}, len(colNames))
		valuePtrs := make([]interface{}, len(colNames))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows2.Scan(valuePtrs...); err != nil {
			continue
		}
		
		// Создаем map для JSON
		record := make(map[string]interface{})
		for i, col := range colNames {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			record[col] = v
		}
		
		jsonData, _ := json.MarshalIndent(record, "   ", "  ")
		fmt.Printf("\n   Запись %d:\n   %s\n", recordNum, string(jsonData))
	}

	// 3. Проверяем, есть ли JSON в attributes
	fmt.Println("\n3. АНАЛИЗ ПОЛЯ 'attributes' (если есть):")
	
	var hasAttributes bool
	for _, col := range colNames {
		if col == "attributes" {
			hasAttributes = true
			break
		}
	}
	
	if hasAttributes {
		var attrSample string
		err = db.QueryRow("SELECT attributes FROM catalog_items WHERE attributes IS NOT NULL AND attributes != '' LIMIT 1").Scan(&attrSample)
		if err == nil && attrSample != "" {
			fmt.Println("   Пример значения attributes:")
			
			// Пытаемся распарсить JSON
			var attrJSON map[string]interface{}
			if err := json.Unmarshal([]byte(attrSample), &attrJSON); err == nil {
				fmt.Println("   ✅ Это JSON! Содержит:")
				for key := range attrJSON {
					fmt.Printf("      - %s\n", key)
				}
			} else {
				fmt.Println("   ⚠️  Не является JSON")
				if len(attrSample) > 200 {
					fmt.Printf("   %s...\n", attrSample[:200])
				} else {
					fmt.Printf("   %s\n", attrSample)
				}
			}
		} else {
			fmt.Println("   ❌ Поле attributes пусто")
		}
	} else {
		fmt.Println("   ❌ Поле attributes не найдено")
	}

	// 4. Статистика по заполненности
	fmt.Println("\n4. СТАТИСТИКА ПО ЗАПОЛНЕННОСТИ:")
	fmt.Println("   ┌──────────────────────────────────┬──────────┬────────────┐")
	fmt.Println("   │ Колонка                          │ Заполнено│ Процент    │")
	fmt.Println("   ├──────────────────────────────────┼──────────┼────────────┤")
	
	var total int
	db.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&total)
	
	for _, col := range colNames {
		var filled int
		query := fmt.Sprintf("SELECT COUNT(*) FROM catalog_items WHERE %s IS NOT NULL AND %s != ''", col, col)
		db.QueryRow(query).Scan(&filled)
		percentage := float64(filled) / float64(total) * 100
		fmt.Printf("   │ %-32s │ %8d │ %9.1f%% │\n", col, filled, percentage)
	}
	
	fmt.Println("   └──────────────────────────────────┴──────────┴────────────┘")
	
	fmt.Printf("\n   Всего записей: %d\n", total)
}

