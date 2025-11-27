//go:build tool_check_normalization_readiness
// +build tool_check_normalization_readiness

package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     📊 ПРОВЕРКА ГОТОВНОСТИ К НОРМАЛИЗАЦИИ                 ║")
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

	// Проверяем проект
	var projectName string
	var clientID int
	var clientName string
	err = serviceDB.QueryRow(`
		SELECT p.name, c.id, c.name 
		FROM client_projects p 
		JOIN clients c ON p.client_id = c.id 
		WHERE p.id = ?
	`, projectID).Scan(&projectName, &clientID, &clientName)

	if err != nil {
		log.Fatalf("❌ Проект не найден: %v", err)
	}

	fmt.Printf("📊 ПРОЕКТ: %s (ID: %d)\n", projectName, projectID)
	fmt.Printf("👤 КЛИЕНТ: %s (ID: %d)\n\n", clientName, clientID)

	// Получаем список баз данных
	rows, err := serviceDB.Query(`
		SELECT id, name, file_path, is_active, last_used_at
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
		LastUsedAt sql.NullTime
	}

	var databases []Database
	var activeCount int
	var inactiveCount int

	for rows.Next() {
		var db Database
		err := rows.Scan(&db.ID, &db.Name, &db.FilePath, &db.IsActive, &db.LastUsedAt)
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

	// Статистика по типам
	nomenclatureCount := 0
	counterpartyCount := 0
	totalRecords := 0

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📋 ДЕТАЛЬНАЯ ИНФОРМАЦИЯ О БАЗАХ ДАННЫХ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for i, db := range databases {
		status := "✅ Активна"
		if !db.IsActive {
			status = "❌ Неактивна"
		}

		fmt.Printf("%d. %s [ID: %d]\n", i+1, db.Name, db.ID)
		fmt.Printf("   Статус: %s\n", status)
		fmt.Printf("   Файл: %s\n", filepath.Base(db.FilePath))

		// Определяем тип данных
		dbPath := filepath.Join("data", db.FilePath)
		conn, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			fmt.Printf("   ⚠️  Ошибка открытия: %v\n", err)
			fmt.Println()
			continue
		}

		dataType := ""
		var count int

		// Проверяем таблицы
		var hasNomenclature bool
		conn.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='nomenclature_items')").Scan(&hasNomenclature)
		if hasNomenclature {
			conn.QueryRow("SELECT COUNT(*) FROM nomenclature_items").Scan(&count)
			if count > 0 {
				dataType = "Номенклатура"
				nomenclatureCount++
			}
		}

		var hasCounterparties bool
		conn.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='counterparties')").Scan(&hasCounterparties)
		if hasCounterparties {
			conn.QueryRow("SELECT COUNT(*) FROM counterparties").Scan(&count)
			if count > 0 {
				dataType = "Контрагенты"
				counterpartyCount++
			}
		}

		// Проверяем catalog_items
		if dataType == "" {
			var hasCatalogItems bool
			conn.QueryRow("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='catalog_items')").Scan(&hasCatalogItems)
			if hasCatalogItems {
				conn.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&count)
				if count > 0 {
					// Определяем тип по имени файла
					fileName := filepath.Base(db.FilePath)
					if len(fileName) > 10 {
						if strings.Contains(fileName, "Номенклатура") {
							dataType = "Номенклатура"
							nomenclatureCount++
						} else if strings.Contains(fileName, "Контрагенты") {
							dataType = "Контрагенты"
							counterpartyCount++
						} else {
							dataType = "Неизвестно"
						}
					}
				}
			}
		}

		if dataType != "" {
			fmt.Printf("   Тип: %s\n", dataType)
			fmt.Printf("   Записей: %d\n", count)
			if db.IsActive {
				totalRecords += count
			}
		}

		if db.LastUsedAt.Valid {
			fmt.Printf("   Использована: %s\n", db.LastUsedAt.Time.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   Использована: никогда\n")
		}

		fmt.Println()
		conn.Close()
	}

	// Проверяем сессии нормализации
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("🔄 СЕССИИ НОРМАЛИЗАЦИИ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	rows2, err := serviceDB.Query(`
		SELECT id, database_id, start_time, end_time, status, processed_count
		FROM normalization_sessions
		WHERE project_id = ?
		ORDER BY start_time DESC
		LIMIT 10
	`, projectID)

	if err != nil {
		log.Printf("⚠️  Ошибка получения сессий: %v", err)
	} else {
		defer rows2.Close()

		sessionCount := 0
		runningCount := 0
		completedCount := 0
		failedCount := 0

		for rows2.Next() {
			sessionCount++
			var id, dbID, processedCount sql.NullInt64
			var startTime, endTime sql.NullTime
			var status sql.NullString

			rows2.Scan(&id, &dbID, &startTime, &endTime, &status, &processedCount)

			if status.Valid {
				switch status.String {
				case "running":
					runningCount++
				case "completed":
					completedCount++
				case "failed":
					failedCount++
				}
			}
		}

		fmt.Printf("Всего сессий: %d\n", sessionCount)
		if sessionCount > 0 {
			fmt.Printf("  • Запущено: %d\n", runningCount)
			fmt.Printf("  • Завершено: %d\n", completedCount)
			fmt.Printf("  • Ошибок: %d\n", failedCount)
		} else {
			fmt.Printf("  ⚠️  Нет сессий нормализации\n")
		}
		fmt.Println()
	}

	// Итоговая статистика
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📊 ИТОГОВАЯ СТАТИСТИКА:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("✅ Готовность:\n")
	fmt.Printf("   • Активных БД: %d\n", activeCount)
	fmt.Printf("   • Номенклатура: %d БД\n", nomenclatureCount)
	fmt.Printf("   • Контрагенты: %d БД\n", counterpartyCount)
	fmt.Printf("   • Всего записей: %d\n", totalRecords)
	fmt.Println()

	fmt.Printf("🎯 Статус нормализации:\n")
	if activeCount > 0 {
		fmt.Printf("   ✅ Готово к запуску нормализации\n")
		fmt.Printf("   📊 Записей к обработке: %d\n", totalRecords)
		estimatedTime := totalRecords / 2500 // ~2500 записей в минуту
		fmt.Printf("   ⏱️  Ожидаемое время: ~%d минут\n", estimatedTime)
	} else {
		fmt.Printf("   ❌ Нет активных баз данных\n")
	}
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("🚀 ИНСТРУКЦИИ ПО ЗАПУСКУ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Println("Вариант 1: Через веб-интерфейс (рекомендуется)")
	fmt.Printf("   1. Откройте: http://localhost:3000\n")
	fmt.Printf("   2. Проекты → %s\n", projectName)
	fmt.Printf("   3. Вкладка 'Нормализация'\n")
	fmt.Printf("   4. Нажмите 'Запустить нормализацию'\n")
	fmt.Println()

	fmt.Println("Вариант 2: Через HTTP API (когда исправлен)")
	fmt.Printf("   POST http://localhost:9999/api/clients/%d/projects/%d/normalization/start\n", clientID, projectID)
	fmt.Println("   Body: {\"all_active\": true}")
	fmt.Println()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ ПРОВЕРКА ЗАВЕРШЕНА                                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

