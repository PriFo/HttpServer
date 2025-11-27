//go:build tool_register_aitas_databases
// +build tool_register_aitas_databases

package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     📁 РЕГИСТРАЦИЯ БАЗ ДАННЫХ AITAS В СИСТЕМЕ              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Подключение к service.db
	db, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к service.db: %v", err)
	}
	defer db.Close()

	log.Println("✅ Подключено к service.db")

	// Проверяем проект AITAS
	var projectID int
	var projectName string
	err = db.QueryRow(`
		SELECT id, name FROM client_projects WHERE id = 1
	`).Scan(&projectID, &projectName)

	if err != nil {
		log.Fatalf("❌ Проект AITAS (ID: 1) не найден: %v", err)
	}

	fmt.Printf("📊 ПРОЕКТ: %s (ID: %d)\n\n", projectName, projectID)

	// Список баз данных для регистрации
	databases := []struct {
		Name        string
		FilePath    string
		Description string
	}{
		{
			Name:        "Контрагенты ERPWE",
			FilePath:    "uploads/Выгрузка_Контрагенты_ERPWE_Unknown_Unknown_2025_11_20_13_27_39.db",
			Description: "База контрагентов из конфигурации ERPWE",
		},
		{
			Name:        "Контрагенты Бухгалтерия (1)",
			FilePath:    "uploads/Выгрузка_Контрагенты_БухгалтерияДляКазахстана_Unknown_Unknown_2025.db",
			Description: "База контрагентов из конфигурации Бухгалтерия для Казахстана",
		},
		{
			Name:        "Контрагенты Бухгалтерия (2)",
			FilePath:    "uploads/Выгрузка_Контрагенты_БухгалтерияДляКазахстана_Unknown_Unknown_2025_20251121_125915.db",
			Description: "База контрагентов из конфигурации Бухгалтерия для Казахстана (обновленная)",
		},
		{
			Name:        "Контрагенты Управление Предприятием",
			FilePath:    "uploads/Выгрузка_Контрагенты_УправлениеПредприятиемДляКазахстана_Unknown.db",
			Description: "База контрагентов из конфигурации Управление Предприятием",
		},
		{
			Name:        "Номенклатура ERPWE",
			FilePath:    "uploads/Выгрузка_Номенклатура_ERPWE_Unknown_Unknown_2025_11_20_10_18_55.db",
			Description: "База номенклатуры из конфигурации ERPWE",
		},
		{
			Name:        "Номенклатура Бухгалтерия (1)",
			FilePath:    "uploads/Выгрузка_Номенклатура_БухгалтерияДляКазахстана_Unknown_Unknown_2025.db",
			Description: "База номенклатуры из конфигурации Бухгалтерия для Казахстана",
		},
		{
			Name:        "Номенклатура Бухгалтерия (2)",
			FilePath:    "uploads/Выгрузка_Номенклатура_БухгалтерияДляКазахстана_Unknown_Unknown_2025_20251121_125914.db",
			Description: "База номенклатуры из конфигурации Бухгалтерия для Казахстана (обновленная)",
		},
		{
			Name:        "Номенклатура Управление Предприятием",
			FilePath:    "uploads/Выгрузка_Номенклатура_УправлениеПредприятиемДляКазахстана_Unknown.db",
			Description: "База номенклатуры из конфигурации Управление Предприятием",
		},
	}

	fmt.Printf("📁 РЕГИСТРАЦИЯ %d БАЗ ДАННЫХ...\n\n", len(databases))

	// Начинаем транзакцию
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("❌ Ошибка начала транзакции: %v", err)
	}

	registeredCount := 0
	now := time.Now()

	for i, dbInfo := range databases {
		// Проверяем, не зарегистрирована ли уже эта БД
		var exists bool
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM project_databases 
				WHERE client_project_id = ? AND file_path = ?
			)
		`, projectID, dbInfo.FilePath).Scan(&exists)

		if err != nil {
			log.Printf("⚠️  Ошибка проверки БД %s: %v", dbInfo.Name, err)
			continue
		}

		if exists {
			fmt.Printf("   %d. %s - УЖЕ ЗАРЕГИСТРИРОВАНА ⏭️\n", i+1, dbInfo.Name)
			continue
		}

		// Регистрируем БД
		_, err = tx.Exec(`
			INSERT INTO project_databases (
				client_project_id, name, file_path, description,
				is_active, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, projectID, dbInfo.Name, dbInfo.FilePath, dbInfo.Description,
			true, now, now)

		if err != nil {
			log.Printf("❌ Ошибка регистрации %s: %v", dbInfo.Name, err)
			tx.Rollback()
			log.Fatalf("Транзакция отменена")
		}

		fmt.Printf("   %d. %s - ✅ ЗАРЕГИСТРИРОВАНА\n", i+1, dbInfo.Name)
		registeredCount++
	}

	// Коммитим транзакцию
	err = tx.Commit()
	if err != nil {
		log.Fatalf("❌ Ошибка коммита: %v", err)
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("\n✅ УСПЕШНО ЗАРЕГИСТРИРОВАНО: %d БД\n", registeredCount)
	
	// Проверяем результат
	var totalDBs int
	db.QueryRow(`
		SELECT COUNT(*) FROM project_databases WHERE client_project_id = ?
	`, projectID).Scan(&totalDBs)

	fmt.Printf("📊 ВСЕГО БД В ПРОЕКТЕ: %d\n", totalDBs)

	// Подсчет записей
	fmt.Println("\n🔍 ПОДСЧЕТ ЗАПИСЕЙ...")
	totalRecords := 0

	rows, err := db.Query(`
		SELECT file_path FROM project_databases WHERE client_project_id = ?
	`, projectID)
	if err != nil {
		log.Printf("⚠️  Ошибка получения списка БД: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var filePath string
			rows.Scan(&filePath)

			dbPath := filepath.Join("data", filePath)
			conn, err := sql.Open("sqlite3", dbPath)
			if err != nil {
				continue
			}

			var count int
			// Пробуем разные таблицы
			tables := []string{"nomenclature_items", "counterparties", "catalog_items"}
			for _, table := range tables {
				var exists bool
				conn.QueryRow(fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='%s')", table)).Scan(&exists)
				if exists {
					conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
					if count > 0 {
						break
					}
				}
			}

			totalRecords += count
			conn.Close()
		}
	}

	fmt.Printf("   Всего записей: %d\n", totalRecords)

	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ РЕГИСТРАЦИЯ ЗАВЕРШЕНА УСПЕШНО!                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🚀 СЛЕДУЮЩИЙ ШАГ:")
	fmt.Println("   Запустить нормализацию через HTTP API:")
	fmt.Printf("   POST http://localhost:9999/api/clients/1/projects/%d/normalization/start\n\n", projectID)
}

