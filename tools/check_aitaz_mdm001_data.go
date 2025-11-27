//go:build tool_check_aitaz_mdm001
// +build tool_check_aitaz_mdm001

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     📊 ПРОВЕРКА ДАННЫХ: AITAS / MDM                          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Подключение к service.db
	serviceDBPath := "data/service.db"
	if _, err := os.Stat(serviceDBPath); os.IsNotExist(err) {
		log.Fatalf("❌ Файл service.db не найден: %s", serviceDBPath)
	}

	serviceDB, err := sql.Open("sqlite3", serviceDBPath)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к service.db: %v", err)
	}
	defer serviceDB.Close()

	log.Println("✅ Подключено к service.db")

	// Ищем клиента "AITAS" или "aitaz"
	var clientID int
	var clientName string
	err = serviceDB.QueryRow(`
		SELECT id, name 
		FROM clients 
		WHERE LOWER(name) LIKE '%aitas%' OR LOWER(legal_name) LIKE '%aitas%' OR LOWER(name) LIKE '%aitaz%'
		LIMIT 1
	`).Scan(&clientID, &clientName)

	if err == sql.ErrNoRows {
		log.Fatalf("❌ Клиент 'AITAS' не найден")
	}
	if err != nil {
		log.Fatalf("❌ Ошибка поиска клиента: %v", err)
	}

	fmt.Printf("👤 КЛИЕНТ: %s (ID: %d)\n\n", clientName, clientID)

	// Ищем проект с MDM в названии
	var projectID int
	var projectName string
	err = serviceDB.QueryRow(`
		SELECT id, name 
		FROM client_projects 
		WHERE client_id = ? AND (name LIKE '%MDM%' OR name LIKE '%MDM_001%')
		ORDER BY id
		LIMIT 1
	`, clientID).Scan(&projectID, &projectName)

	if err == sql.ErrNoRows {
		log.Fatalf("❌ Проект с MDM не найден для клиента %s", clientName)
	}
	if err != nil {
		log.Fatalf("❌ Ошибка поиска проекта: %v", err)
	}

	fmt.Printf("📊 ПРОЕКТ: %s (ID: %d)\n\n", projectName, projectID)

	// Получаем список баз данных
	rows, err := serviceDB.Query(`
		SELECT id, name, file_path, is_active, created_at, updated_at, last_used_at
		FROM project_databases 
		WHERE client_project_id = ?
		ORDER BY name
	`, projectID)

	if err != nil {
		log.Fatalf("❌ Ошибка получения баз данных: %v", err)
	}
	defer rows.Close()

	type Database struct {
		ID         int
		Name       string
		FilePath   string
		IsActive   bool
		CreatedAt  string
		UpdatedAt  string
		LastUsedAt sql.NullTime
	}

	var databases []Database
	var activeCount int
	var inactiveCount int

	for rows.Next() {
		var db Database
		err := rows.Scan(&db.ID, &db.Name, &db.FilePath, &db.IsActive, &db.CreatedAt, &db.UpdatedAt, &db.LastUsedAt)
		if err != nil {
			log.Printf("⚠️  Ошибка чтения БД: %v", err)
			continue
		}
		databases = append(databases, db)
		if db.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	fmt.Printf("📁 БАЗЫ ДАННЫХ: %d всего (%d активных, %d неактивных)\n\n", len(databases), activeCount, inactiveCount)

	if len(databases) == 0 {
		fmt.Println("⚠️  В проекте нет баз данных")
		return
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📋 ДЕТАЛЬНАЯ ИНФОРМАЦИЯ О БАЗАХ ДАННЫХ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	totalTables := 0
	totalRecords := 0
	nomenclatureCount := 0
	counterpartyCount := 0
	catalogCount := 0
	totalNomenclatureRecords := 0
	totalCounterpartyRecords := 0
	totalNormalizedRecords := 0

	for i, db := range databases {
		status := "✅ Активна"
		if !db.IsActive {
			status = "❌ Неактивна"
		}

		fmt.Printf("%d. %s [ID: %d]\n", i+1, db.Name, db.ID)
		fmt.Printf("   Статус: %s\n", status)
		fmt.Printf("   Файл: %s\n", filepath.Base(db.FilePath))
		fmt.Printf("   Путь: %s\n", db.FilePath)
		fmt.Printf("   Создана: %s\n", db.CreatedAt)
		fmt.Printf("   Обновлена: %s\n", db.UpdatedAt)

		// Проверяем существование файла
		dbFullPath := db.FilePath
		if !filepath.IsAbs(dbFullPath) {
			dbFullPath = filepath.Join("data", db.FilePath)
		}

		if _, err := os.Stat(dbFullPath); os.IsNotExist(err) {
			fmt.Printf("   ⚠️  Файл не найден: %s\n\n", dbFullPath)
			continue
		}

		// Открываем базу данных
		conn, err := sql.Open("sqlite3", dbFullPath)
		if err != nil {
			fmt.Printf("   ⚠️  Ошибка открытия: %v\n\n", err)
			continue
		}

		// Получаем список всех таблиц
		tableRows, err := conn.Query(`
			SELECT name 
			FROM sqlite_master 
			WHERE type='table' AND name NOT LIKE 'sqlite_%'
			ORDER BY name
		`)
		if err != nil {
			fmt.Printf("   ⚠️  Ошибка получения таблиц: %v\n\n", err)
			conn.Close()
			continue
		}

		var tables []string
		for tableRows.Next() {
			var tableName string
			if err := tableRows.Scan(&tableName); err == nil {
				tables = append(tables, tableName)
			}
		}
		tableRows.Close()

		if len(tables) == 0 {
			fmt.Printf("   ⚠️  Нет таблиц в базе данных\n\n")
			conn.Close()
			continue
		}

		fmt.Printf("   📊 Таблиц: %d\n", len(tables))
		totalTables += len(tables)

		// Проверяем основные таблицы
		dbRecordCount := 0
		dbNomenclatureRecords := 0
		dbCounterpartyRecords := 0

		// Номенклатура
		if contains(tables, "nomenclature_items") {
			var count int
			conn.QueryRow("SELECT COUNT(*) FROM nomenclature_items").Scan(&count)
			if count > 0 {
				fmt.Printf("   📦 Номенклатура (nomenclature_items): %d записей\n", count)
				nomenclatureCount++
				dbNomenclatureRecords = count
				dbRecordCount += count
			}
		}

		// Контрагенты
		if contains(tables, "counterparties") {
			var count int
			conn.QueryRow("SELECT COUNT(*) FROM counterparties").Scan(&count)
			if count > 0 {
				fmt.Printf("   👥 Контрагенты (counterparties): %d записей\n", count)
				counterpartyCount++
				dbCounterpartyRecords = count
				dbRecordCount += count
			}
		}

		// Каталог товаров
		if contains(tables, "catalog_items") {
			var count int
			conn.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&count)
			if count > 0 {
				fmt.Printf("   📚 Каталог товаров (catalog_items): %d записей\n", count)
				catalogCount++
				dbRecordCount += count
			}
		}

		// Проверяем, есть ли номенклатура или контрагенты в catalog_items
		// Определяем по названию базы данных
		if contains(tables, "catalog_items") {
			var catalogCount int
			conn.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&catalogCount)
			
			// Определяем тип по названию базы
			dbNameLower := strings.ToLower(db.Name)
			if strings.Contains(dbNameLower, "номенклатура") || strings.Contains(dbNameLower, "nomenclature") {
				// Это база с номенклатурой
				dbNomenclatureRecords = catalogCount
				fmt.Printf("   📦 Номенклатура (определено по названию БД): %d записей\n", catalogCount)
			} else if strings.Contains(dbNameLower, "контрагент") || strings.Contains(dbNameLower, "counterparty") {
				// Это база с контрагентами
				dbCounterpartyRecords = catalogCount
				fmt.Printf("   👥 Контрагенты (определено по названию БД): %d записей\n", catalogCount)
			} else {
				// Пытаемся определить по категориям в catalog_items
				var nomCount, contCount int
				conn.QueryRow(`
					SELECT 
						SUM(CASE WHEN category LIKE '%Номенклатура%' OR category LIKE '%номенклатура%' OR category LIKE '%Nomenclature%' THEN 1 ELSE 0 END),
						SUM(CASE WHEN category LIKE '%Контрагент%' OR category LIKE '%контрагент%' OR category LIKE '%Counterparty%' THEN 1 ELSE 0 END)
					FROM catalog_items
				`).Scan(&nomCount, &contCount)
				
				if nomCount > 0 {
					fmt.Printf("   📦 Номенклатура (в catalog_items по категории): %d записей\n", nomCount)
					dbNomenclatureRecords = nomCount
				}
				if contCount > 0 {
					fmt.Printf("   👥 Контрагенты (в catalog_items по категории): %d записей\n", contCount)
					dbCounterpartyRecords = contCount
				}
			}
		}

		// Суммируем общие значения
		totalNomenclatureRecords += dbNomenclatureRecords
		totalCounterpartyRecords += dbCounterpartyRecords

		// Проверяем наличие normalized_data в этой базе
		if contains(tables, "normalized_data") {
			var normalizedCount int
			conn.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&normalizedCount)
			if normalizedCount > 0 {
				fmt.Printf("   🔄 Нормализованных записей: %d\n", normalizedCount)
				totalNormalizedRecords += normalizedCount
				
				// Статистика по нормализации
				var withKpved, withAI int
				var avgConf sql.NullFloat64
				conn.QueryRow(`
					SELECT 
						SUM(CASE WHEN kpved_code IS NOT NULL AND kpved_code != '' THEN 1 ELSE 0 END),
						SUM(CASE WHEN ai_confidence > 0 THEN 1 ELSE 0 END),
						AVG(ai_confidence)
					FROM normalized_data
				`).Scan(&withKpved, &withAI, &avgConf)
				
				if withKpved > 0 {
					fmt.Printf("      • С КПВЭД: %d (%.1f%%)\n", withKpved, float64(withKpved)*100/float64(normalizedCount))
				}
				if withAI > 0 {
					fmt.Printf("      • С AI: %d (%.1f%%)\n", withAI, float64(withAI)*100/float64(normalizedCount))
				}
				if avgConf.Valid {
					fmt.Printf("      • Средняя AI уверенность: %.2f%%\n", avgConf.Float64*100)
				}
			}
		}

		// Другие таблицы (первые 10)
		otherTables := 0
		importantTables := []string{"nomenclature_items", "counterparties", "catalog_items", "normalized_data"}
		for _, table := range tables {
			if !contains(importantTables, table) {
				var count int
				if err := conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err == nil {
					otherTables++
					if otherTables <= 5 {
						fmt.Printf("   📋 %s: %d записей\n", table, count)
					}
				}
			}
		}
		if otherTables > 5 {
			fmt.Printf("   ... и еще %d таблиц\n", otherTables-5)
		}

		totalRecords += dbRecordCount
		if dbRecordCount > 0 {
			fmt.Printf("   ✅ Всего записей: %d\n", dbRecordCount)
		}

		// Размер файла
		if info, err := os.Stat(dbFullPath); err == nil {
			sizeMB := float64(info.Size()) / (1024 * 1024)
			fmt.Printf("   💾 Размер: %.2f MB\n", sizeMB)
		}

		if db.LastUsedAt.Valid {
			fmt.Printf("   🕐 Использована: %s\n", db.LastUsedAt.Time.Format("2006-01-02 15:04:05"))
		}

		fmt.Println()
		conn.Close()
	}

	// Проверяем нормализованные данные в основной БД
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("🔄 НОРМАЛИЗОВАННЫЕ ДАННЫЕ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	// Подключаемся к основной БД для проверки normalized_data
	// Проверяем несколько возможных путей
	possiblePaths := []string{
		"data/normalized_data.db",
		"data/normalized.db",
		"normalized_data.db",
		"normalized.db",
	}
	
	var mainDBPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			mainDBPath = path
			break
		}
	}
	
	if mainDBPath != "" {
		mainDB, err := sql.Open("sqlite3", mainDBPath)
		if err == nil {
			defer mainDB.Close()

			// Проверяем наличие таблицы normalized_data
			var tableExists bool
			mainDB.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM sqlite_master 
					WHERE type='table' AND name='normalized_data'
				)
			`).Scan(&tableExists)

			if tableExists {
				// Проверяем, есть ли поле project_id
				var hasProjectID bool
				mainDB.QueryRow(`
					SELECT EXISTS (
						SELECT 1 FROM pragma_table_info('normalized_data') 
						WHERE name='project_id'
					)
				`).Scan(&hasProjectID)

				// Общая статистика
				var totalNormalized, totalWithKpved, totalWithAI int
				var avgConfidence, avgKpvedConfidence sql.NullFloat64
				
				var query string
				if hasProjectID {
					query = `
						SELECT 
							COUNT(*),
							SUM(CASE WHEN kpved_code IS NOT NULL AND kpved_code != '' THEN 1 ELSE 0 END),
							SUM(CASE WHEN ai_confidence > 0 THEN 1 ELSE 0 END),
							AVG(ai_confidence),
							AVG(kpved_confidence)
						FROM normalized_data
						WHERE project_id = ?
					`
					mainDB.QueryRow(query, projectID).Scan(&totalNormalized, &totalWithKpved, &totalWithAI, &avgConfidence, &avgKpvedConfidence)
				} else {
					// Если нет project_id, считаем все записи
					query = `
						SELECT 
							COUNT(*),
							SUM(CASE WHEN kpved_code IS NOT NULL AND kpved_code != '' THEN 1 ELSE 0 END),
							SUM(CASE WHEN ai_confidence > 0 THEN 1 ELSE 0 END),
							AVG(ai_confidence),
							AVG(kpved_confidence)
						FROM normalized_data
					`
					mainDB.QueryRow(query).Scan(&totalNormalized, &totalWithKpved, &totalWithAI, &avgConfidence, &avgKpvedConfidence)
				}

				fmt.Printf("📊 Всего нормализованных записей: %d\n", totalNormalized)
				if totalNormalized > 0 {
					fmt.Printf("   • С КПВЭД классификацией: %d (%.1f%%)\n", totalWithKpved, float64(totalWithKpved)*100/float64(totalNormalized))
					fmt.Printf("   • С AI обработкой: %d (%.1f%%)\n", totalWithAI, float64(totalWithAI)*100/float64(totalNormalized))
					if avgConfidence.Valid {
						fmt.Printf("   • Средняя AI уверенность: %.2f%%\n", avgConfidence.Float64*100)
					}
					if avgKpvedConfidence.Valid {
						fmt.Printf("   • Средняя КПВЭД уверенность: %.2f%%\n", avgKpvedConfidence.Float64*100)
					}

					// Статистика по категориям
					var catQuery string
					if hasProjectID {
						catQuery = `
							SELECT category, COUNT(*) as cnt
							FROM normalized_data
							WHERE project_id = ?
							GROUP BY category
							ORDER BY cnt DESC
							LIMIT 10
						`
					} else {
						catQuery = `
							SELECT category, COUNT(*) as cnt
							FROM normalized_data
							GROUP BY category
							ORDER BY cnt DESC
							LIMIT 10
						`
					}
					
					catRows, _ := mainDB.Query(catQuery, projectID)
					if catRows != nil {
						defer catRows.Close()
						fmt.Println()
						fmt.Println("   📋 Топ категорий:")
						catCount := 0
						for catRows.Next() {
							var cat sql.NullString
							var cnt int
							if err := catRows.Scan(&cat, &cnt); err == nil {
								catCount++
								catStr := "(без категории)"
								if cat.Valid && cat.String != "" {
									catStr = cat.String
								}
								fmt.Printf("      • %s: %d записей\n", catStr, cnt)
							}
						}
						if catCount == 0 {
							fmt.Println("      (нет данных)")
						}
					}

					// Статистика по уровням обработки
					var levelQuery string
					if hasProjectID {
						levelQuery = `
							SELECT processing_level, COUNT(*) as cnt
							FROM normalized_data
							WHERE project_id = ?
							GROUP BY processing_level
							ORDER BY cnt DESC
						`
					} else {
						levelQuery = `
							SELECT processing_level, COUNT(*) as cnt
							FROM normalized_data
							GROUP BY processing_level
							ORDER BY cnt DESC
						`
					}
					
					levelRows, _ := mainDB.Query(levelQuery, projectID)
					if levelRows != nil {
						defer levelRows.Close()
						fmt.Println()
						fmt.Println("   🔧 Уровни обработки:")
						levelCount := 0
						for levelRows.Next() {
							var level sql.NullString
							var cnt int
							if err := levelRows.Scan(&level, &cnt); err == nil {
								levelCount++
								levelStr := "basic"
								if level.Valid && level.String != "" {
									levelStr = level.String
								}
								fmt.Printf("      • %s: %d записей\n", levelStr, cnt)
							}
						}
						if levelCount == 0 {
							fmt.Println("      (нет данных)")
						}
					}
					
					totalNormalizedRecords = totalNormalized
				} else {
					fmt.Println("   ⚠️  Нет нормализованных данных для этого проекта")
				}
			} else {
				fmt.Println("   ⚠️  Таблица normalized_data не найдена")
			}
		} else {
			fmt.Printf("   ⚠️  Ошибка подключения к %s: %v\n", mainDBPath, err)
		}
	} else {
		fmt.Println("   ⚠️  Файл normalized_data.db не найден")
	}

	fmt.Println()

	// Итоговая статистика
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📊 ИТОГОВАЯ СТАТИСТИКА:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("📁 Всего баз данных: %d\n", len(databases))
	fmt.Printf("   • Активных: %d\n", activeCount)
	fmt.Printf("   • Неактивных: %d\n", inactiveCount)
	fmt.Println()
	fmt.Printf("📊 Всего таблиц: %d\n", totalTables)
	fmt.Printf("📦 Баз с номенклатурой: %d\n", nomenclatureCount)
	fmt.Printf("👥 Баз с контрагентами: %d\n", counterpartyCount)
	fmt.Printf("📚 Баз с каталогом: %d\n", catalogCount)
	fmt.Println()
	fmt.Printf("📦 ВСЕГО ЗАПИСЕЙ НОМЕНКЛАТУРЫ: %d\n", totalNomenclatureRecords)
	fmt.Printf("👥 ВСЕГО ЗАПИСЕЙ КОНТРАГЕНТОВ: %d\n", totalCounterpartyRecords)
	fmt.Printf("✅ Всего записей в исходных БД: %d\n", totalRecords)
	if totalNormalizedRecords > 0 {
		fmt.Printf("🔄 Всего нормализованных записей: %d\n", totalNormalizedRecords)
		fmt.Printf("   📊 Процент нормализации: %.1f%%\n", float64(totalNormalizedRecords)*100/float64(totalRecords))
	} else {
		fmt.Printf("⚠️  Нормализованных записей: 0 (нормализация не выполнялась)\n")
	}
	fmt.Println()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ ПРОВЕРКА ЗАВЕРШЕНА                                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

