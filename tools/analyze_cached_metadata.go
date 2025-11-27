//go:build tool_analyze_cached_metadata
// +build tool_analyze_cached_metadata

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// CachedMetadata метаданные из кэша
type CachedMetadata struct {
	ID                  int     `json:"id"`
	DatabaseID          int     `json:"database_id"`
	TableName           string  `json:"table_name"`
	EntityType          string  `json:"entity_type"`
	ColumnMappings      string  `json:"column_mappings"`
	DetectionConfidence float64 `json:"detection_confidence"`
	LastUpdated         string  `json:"last_updated"`
	DatabaseName        string  `json:"database_name"`
	DatabasePath        string  `json:"database_path"`
	ProjectName         string  `json:"project_name"`
}

// ColumnMapping маппинг колонок из JSON
type ColumnMapping struct {
	TableName string `json:"table_name"`
	Name      string `json:"name"`
	INN       string `json:"inn"`
	BIN       string `json:"bin"`
	OGRN      string `json:"ogrn"`
	KPP       string `json:"kpp"`
	LegalName string `json:"legal_name"`
	Address   string `json:"address"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
}

func main() {
	fmt.Println("=== АНАЛИЗ КЭШИРОВАННЫХ МЕТАДАННЫХ ===\n")

	// Подключаемся к service.db
	db, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("Failed to open service.db: %v", err)
	}
	defer db.Close()

	// Получаем все кэшированные метаданные
	query := `
		SELECT 
			dtm.id, 
			dtm.database_id, 
			dtm.table_name, 
			dtm.entity_type, 
			dtm.column_mappings, 
			dtm.detection_confidence,
			dtm.last_updated,
			pd.name as database_name,
			pd.file_path as database_path,
			cp.name as project_name
		FROM database_table_metadata dtm
		LEFT JOIN project_databases pd ON dtm.database_id = pd.id
		LEFT JOIN client_projects cp ON pd.client_project_id = cp.id
		WHERE dtm.entity_type = 'counterparty'
		ORDER BY dtm.detection_confidence DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query metadata: %v", err)
	}
	defer rows.Close()

	var metadata []CachedMetadata
	for rows.Next() {
		var m CachedMetadata
		var dbName, dbPath, projName sql.NullString
		
		if err := rows.Scan(
			&m.ID,
			&m.DatabaseID,
			&m.TableName,
			&m.EntityType,
			&m.ColumnMappings,
			&m.DetectionConfidence,
			&m.LastUpdated,
			&dbName,
			&dbPath,
			&projName,
		); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		
		m.DatabaseName = dbName.String
		m.DatabasePath = dbPath.String
		m.ProjectName = projName.String
		
		metadata = append(metadata, m)
	}

	fmt.Printf("Найдено кэшированных записей: %d\n\n", len(metadata))

	if len(metadata) == 0 {
		fmt.Println("❌ Нет кэшированных метаданных. Запустите детектор сначала.")
		return
	}

	// Анализируем каждую запись
	allColumns := make(map[string]map[string]int) // field -> value -> count

	for i, m := range metadata {
		fmt.Printf("%d. БД: %s (ID: %d)\n", i+1, m.DatabaseName, m.DatabaseID)
		fmt.Printf("   Проект: %s\n", m.ProjectName)
		fmt.Printf("   Путь: %s\n", m.DatabasePath)
		fmt.Printf("   Таблица: %s\n", m.TableName)
		fmt.Printf("   Confidence: %.2f\n", m.DetectionConfidence)
		
		// Парсим JSON маппинг
		var mapping ColumnMapping
		if err := json.Unmarshal([]byte(m.ColumnMappings), &mapping); err != nil {
			fmt.Printf("   ⚠️  Ошибка парсинга JSON: %v\n\n", err)
			continue
		}

		fmt.Printf("   Колонки:\n")
		if mapping.Name != "" {
			fmt.Printf("     - Наименование: %s\n", mapping.Name)
			if allColumns["name"] == nil {
				allColumns["name"] = make(map[string]int)
			}
			allColumns["name"][mapping.Name]++
		}
		if mapping.INN != "" {
			fmt.Printf("     - ИНН: %s\n", mapping.INN)
			if allColumns["inn"] == nil {
				allColumns["inn"] = make(map[string]int)
			}
			allColumns["inn"][mapping.INN]++
		}
		if mapping.BIN != "" {
			fmt.Printf("     - БИН: %s\n", mapping.BIN)
			if allColumns["bin"] == nil {
				allColumns["bin"] = make(map[string]int)
			}
			allColumns["bin"][mapping.BIN]++
		}
		if mapping.OGRN != "" {
			fmt.Printf("     - ОГРН: %s\n", mapping.OGRN)
			if allColumns["ogrn"] == nil {
				allColumns["ogrn"] = make(map[string]int)
			}
			allColumns["ogrn"][mapping.OGRN]++
		}
		if mapping.KPP != "" {
			fmt.Printf("     - КПП: %s\n", mapping.KPP)
			if allColumns["kpp"] == nil {
				allColumns["kpp"] = make(map[string]int)
			}
			allColumns["kpp"][mapping.KPP]++
		}
		if mapping.Address != "" {
			fmt.Printf("     - Адрес: %s\n", mapping.Address)
			if allColumns["address"] == nil {
				allColumns["address"] = make(map[string]int)
			}
			allColumns["address"][mapping.Address]++
		}
		if mapping.Phone != "" {
			fmt.Printf("     - Телефон: %s\n", mapping.Phone)
			if allColumns["phone"] == nil {
				allColumns["phone"] = make(map[string]int)
			}
			allColumns["phone"][mapping.Phone]++
		}
		if mapping.Email != "" {
			fmt.Printf("     - Email: %s\n", mapping.Email)
			if allColumns["email"] == nil {
				allColumns["email"] = make(map[string]int)
			}
			allColumns["email"][mapping.Email]++
		}
		
		fmt.Println()
	}

	// Анализ совместимости
	fmt.Println("\n=== АНАЛИЗ СОВМЕСТИМОСТИ РЕКВИЗИТОВ ===\n")

	totalDBs := len(metadata)
	
	for field, values := range allColumns {
		fmt.Printf("📊 Реквизит: %s\n", field)
		fmt.Printf("   Уникальных названий колонок: %d\n", len(values))
		
		if len(values) == 1 {
			for colName := range values {
				fmt.Printf("   ✅ Единообразное название: '%s' (во всех %d БД)\n", colName, totalDBs)
			}
		} else {
			fmt.Printf("   ⚠️  Разные названия:\n")
			for colName, count := range values {
				percentage := float64(count) / float64(totalDBs) * 100
				fmt.Printf("     - '%s': %d БД (%.1f%%)\n", colName, count, percentage)
			}
		}
		fmt.Println()
	}

	// Сводная таблица
	fmt.Println("\n=== СВОДНАЯ ТАБЛИЦА РЕКВИЗИТОВ ===\n")
	fmt.Printf("| Реквизит      | Присутствует | Единообразие |\n")
	fmt.Printf("|---------------|--------------|-------------|\n")
	
	requiredFields := []string{"name", "inn", "bin", "ogrn", "kpp", "address", "phone", "email"}
	for _, field := range requiredFields {
		values, exists := allColumns[field]
		if !exists {
			fmt.Printf("| %-13s | %3d/%d (%.0f%%) | ❌ Отсутствует |\n", 
				field, 0, totalDBs, 0.0)
			continue
		}
		
		count := 0
		for _, c := range values {
			count += c
		}
		
		uniform := "✅"
		if len(values) > 1 {
			uniform = "⚠️  Различается"
		}
		
		fmt.Printf("| %-13s | %3d/%d (%.0f%%) | %-15s |\n", 
			field, count, totalDBs, float64(count)/float64(totalDBs)*100, uniform)
	}
	
	fmt.Println()

	// Сохраняем результаты
	output, _ := json.MarshalIndent(metadata, "", "  ")
	if err := os.WriteFile("cached_metadata_analysis.json", output, 0644); err != nil {
		log.Printf("Failed to save JSON: %v", err)
	} else {
		fmt.Println("✅ Результаты сохранены в cached_metadata_analysis.json")
	}
}

