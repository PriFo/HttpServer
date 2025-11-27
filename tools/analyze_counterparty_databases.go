//go:build tool_analyze_counterparty_databases
// +build tool_analyze_counterparty_databases

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"httpserver/database"

	_ "github.com/mattn/go-sqlite3"
)

type RequisiteMapping struct {
	DBName      string
	DBPath      string
	TableName   string
	Confidence  float64
	Requisites  map[string]string // requisite name -> column name
}

func main() {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("║    АНАЛИЗ РЕКВИЗИТНОГО СОСТАВА БАЗ КОНТРАГЕНТОВ    ║")
	fmt.Println("═══════════════════════════════════════════════════════════\n")

	// Открываем service.db
	serviceDB, err := database.NewServiceDB("data/service.db")
	if err != nil {
		log.Fatalf("Failed to open service.db: %v", err)
	}
	defer serviceDB.Close()

	// Ищем все БД с контрагентами
	uploadsDir := "data/uploads"
	files, err := filepath.Glob(filepath.Join(uploadsDir, "*Контрагент*.db"))
	if err != nil {
		log.Fatalf("Failed to search files: %v", err)
	}

	if len(files) == 0 {
		fmt.Println("❌ Не найдено БД с контрагентами в data/uploads")
		return
	}

	fmt.Printf("Найдено БД с контрагентами: %d\n\n", len(files))

	// Анализируем каждую БД
	var mappings []RequisiteMapping

	for i, filePath := range files {
		dbName := filepath.Base(filePath)
		fmt.Printf("%d. БД: %s\n", i+1, dbName)

		// Открываем БД
		db, err := sql.Open("sqlite3", filePath+"?mode=ro")
		if err != nil {
			fmt.Printf("   ❌ Ошибка открытия: %v\n\n", err)
			continue
		}

		// Ищем таблицу catalog_items
		var tableName string
		err = db.QueryRow(`
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name = 'catalog_items'
			LIMIT 1
		`).Scan(&tableName)

		if err != nil {
			fmt.Printf("   ⚠️  Таблица catalog_items не найдена\n\n", err)
			db.Close()
			continue
		}

		// Получаем структуру таблицы
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			fmt.Printf("   ❌ Ошибка получения структуры: %v\n\n", err)
			db.Close()
			continue
		}

		requisites := make(map[string]string)
		var allColumns []string

		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue sql.NullString
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err == nil {
				allColumns = append(allColumns, name)
				
				nameLower := strings.ToLower(name)
				// Определяем типы реквизитов
				if strings.Contains(nameLower, "наименование") || nameLower == "name" {
					requisites["Наименование"] = name
				}
				if strings.Contains(nameLower, "инн") || nameLower == "inn" || strings.Contains(nameLower, "taxpayerid") {
					requisites["ИНН"] = name
				}
				if strings.Contains(nameLower, "бин") || nameLower == "bin" {
					requisites["БИН"] = name
				}
				if strings.Contains(nameLower, "огрн") || nameLower == "ogrn" {
					requisites["ОГРН"] = name
				}
				if strings.Contains(nameLower, "кпп") || nameLower == "kpp" {
					requisites["КПП"] = name
				}
				if strings.Contains(nameLower, "юридический") && strings.Contains(nameLower, "адрес") {
					requisites["Юридический адрес"] = name
				}
				if strings.Contains(nameLower, "фактический") && strings.Contains(nameLower, "адрес") {
					requisites["Фактический адрес"] = name
				}
				if (strings.Contains(nameLower, "телефон") || nameLower == "phone") && !strings.Contains(nameLower, "факс") {
					requisites["Телефон"] = name
				}
				if (strings.Contains(nameLower, "email") || strings.Contains(nameLower, "почта")) && !strings.Contains(nameLower, "индекс") {
					requisites["Email"] = name
				}
			}
		}
		rows.Close()

		// Подсчет записей
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)

		confidence := float64(len(requisites)) / 9.0 // 9 основных реквизитов

		mapping := RequisiteMapping{
			DBName:     dbName,
			DBPath:     filePath,
			TableName:  tableName,
			Confidence: confidence,
			Requisites: requisites,
		}
		mappings = append(mappings, mapping)

		fmt.Printf("   ✅ Таблица: %s\n", tableName)
		fmt.Printf("   📊 Записей: %d\n", count)
		fmt.Printf("   📋 Всего колонок: %d\n", len(allColumns))
		fmt.Printf("   🎯 Confidence: %.2f\n", confidence)
		fmt.Printf("   📑 Найдено реквизитов: %d\n", len(requisites))
		
		for req, col := range requisites {
			fmt.Printf("      • %s → %s\n", req, col)
		}

		db.Close()
		fmt.Println()
	}

	// Анализ совместимости
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("║           АНАЛИЗ СОВМЕСТИМОСТИ РЕКВИЗИТОВ           ║")
	fmt.Println("═══════════════════════════════════════════════════════════\n")

	// Собираем все уникальные реквизиты
	allRequisites := make(map[string]map[string]int) // requisite -> column_name -> count
	
	for _, m := range mappings {
		for req, col := range m.Requisites {
			if allRequisites[req] == nil {
				allRequisites[req] = make(map[string]int)
			}
			allRequisites[req][col]++
		}
	}

	totalDBs := len(mappings)
	requisitesList := []string{"Наименование", "ИНН", "БИН", "ОГРН", "КПП", "Юридический адрес", "Фактический адрес", "Телефон", "Email"}

	fmt.Printf("┌─────────────────────────┬──────────────┬─────────────────┐\n")
	fmt.Printf("│ Реквизит                │ Присутствует │ Единообразие    │\n")
	fmt.Printf("├─────────────────────────┼──────────────┼─────────────────┤\n")

	for _, req := range requisitesList {
		columns, exists := allRequisites[req]
		if !exists {
			fmt.Printf("│ %-23s │ %2d/%d (%.0f%%) │ ❌ Отсутствует  │\n", 
				req, 0, totalDBs, 0.0)
			continue
		}

		totalCount := 0
		for _, count := range columns {
			totalCount += count
		}

		status := "✅ Единообразно"
		if len(columns) > 1 {
			status = "⚠️  Различается"
		}

		fmt.Printf("│ %-23s │ %2d/%d (%.0f%%) │ %-15s │\n",
			req, totalCount, totalDBs, 
			float64(totalCount)/float64(totalDBs)*100, status)
	}

	fmt.Printf("└─────────────────────────┴──────────────┴─────────────────┘\n\n")

	// Детальная информация о различиях
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("║        ДЕТАЛЬНАЯ ИНФОРМАЦИЯ О РАЗЛИЧИЯХ          ║")
	fmt.Println("═══════════════════════════════════════════════════════════\n")

	for _, req := range requisitesList {
		columns, exists := allRequisites[req]
		if !exists || len(columns) <= 1 {
			continue
		}

		fmt.Printf("📊 %s:\n", req)
		for col, count := range columns {
			percentage := float64(count) / float64(totalDBs) * 100
			fmt.Printf("   • '%s': %d БД (%.0f%%)\n", col, count, percentage)
		}
		fmt.Println()
	}

	// Сохраняем результаты в JSON
	output, _ := json.MarshalIndent(mappings, "", "  ")
	if err := os.WriteFile("requisite_analysis.json", output, 0644); err != nil {
		log.Printf("Failed to save JSON: %v", err)
	} else {
		fmt.Println("\n✅ Результаты сохранены в requisite_analysis.json")
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("║              АНАЛИЗ ЗАВЕРШЕН УСПЕШНО!             ║")
	fmt.Println("═══════════════════════════════════════════════════════════")
}

