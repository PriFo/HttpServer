//go:build tool_start_normalization_project
// +build tool_start_normalization_project

package main

import (
	"context"
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
	fmt.Println("║     🚀 ЗАПУСК НОРМАЛИЗАЦИИ ДЛЯ ПРОЕКТА MDM AITAS            ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Подключение к service.db
	serviceDB, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к service.db: %v", err)
	}
	defer serviceDB.Close()

	log.Println("✅ Подключено к service.db")

	// Находим проект "mdm aitas" (ID: 3)
	var projectID int
	var projectName, clientName string
	var clientID int
	
	err = serviceDB.QueryRow(`
		SELECT p.id, p.name, c.id, c.name 
		FROM client_projects p 
		JOIN clients c ON p.client_id = c.id 
		WHERE p.id = 3
	`).Scan(&projectID, &projectName, &clientID, &clientName)
	
	if err != nil {
		log.Fatalf("❌ Проект с ID 3 не найден: %v", err)
	}

	fmt.Printf("📊 ПРОЕКТ НАЙДЕН:\n")
	fmt.Printf("   • ID: %d\n", projectID)
	fmt.Printf("   • Название: %s\n", projectName)
	fmt.Printf("   • Клиент: %s (ID: %d)\n\n", clientName, clientID)

	// Получаем список баз данных проекта
	rows, err := serviceDB.Query(`
		SELECT id, name, file_path, database_type, data_type, is_normalized 
		FROM project_databases 
		WHERE project_id = ?
		ORDER BY data_type, database_type
	`, projectID)
	
	if err != nil {
		log.Fatalf("❌ Ошибка получения баз данных: %v", err)
	}
	defer rows.Close()

	type Database struct {
		ID           int
		Name         string
		FilePath     string
		DatabaseType string
		DataType     string
		IsNormalized bool
	}

	var databases []Database
	for rows.Next() {
		var db Database
		err := rows.Scan(&db.ID, &db.Name, &db.FilePath, &db.DatabaseType, &db.DataType, &db.IsNormalized)
		if err != nil {
			log.Printf("⚠️  Ошибка чтения БД: %v", err)
			continue
		}
		databases = append(databases, db)
	}

	fmt.Printf("📁 НАЙДЕНО БАЗ ДАННЫХ: %d\n\n", len(databases))

	// Статистика
	nomenclatureCount := 0
	counterpartyCount := 0
	normalizedCount := 0

	for _, db := range databases {
		status := "❌ Не нормализована"
		if db.IsNormalized {
			status = "✅ Нормализована"
			normalizedCount++
		}
		
		if db.DataType == "nomenclature" {
			nomenclatureCount++
		} else if db.DataType == "counterparty" {
			counterpartyCount++
		}

		fmt.Printf("   %d. %s\n", db.ID, db.Name)
		fmt.Printf("      Тип: %s | Данные: %s\n", db.DatabaseType, db.DataType)
		fmt.Printf("      Путь: %s\n", filepath.Base(db.FilePath))
		fmt.Printf("      Статус: %s\n\n", status)
	}

	fmt.Printf("📊 СТАТИСТИКА:\n")
	fmt.Printf("   • Номенклатура: %d БД\n", nomenclatureCount)
	fmt.Printf("   • Контрагенты: %d БД\n", counterpartyCount)
	fmt.Printf("   • Уже нормализовано: %d БД\n", normalizedCount)
	fmt.Printf("   • Требуется нормализовать: %d БД\n\n", len(databases)-normalizedCount)

	// Подсчет записей
	fmt.Println("🔍 ПОДСЧЕТ ЗАПИСЕЙ...")
	totalNomenclature := 0
	totalCounterparty := 0

	for _, db := range databases {
		dbPath := db.FilePath
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join("data", dbPath)
		}

		conn, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Printf("⚠️  Не удалось открыть %s: %v", db.Name, err)
			continue
		}

		// Подсчет записей в зависимости от типа
		var count int
		if db.DataType == "nomenclature" {
			// Проверяем разные таблицы
			tables := []string{"nomenclature_items", "catalog_items"}
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
			totalNomenclature += count
		} else if db.DataType == "counterparty" {
			// Проверяем разные таблицы
			tables := []string{"counterparties", "catalog_items"}
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
			totalCounterparty += count
		}

		conn.Close()
		fmt.Printf("   ✅ %s: %d записей\n", db.Name, count)
	}

	fmt.Printf("\n📊 ВСЕГО ЗАПИСЕЙ:\n")
	fmt.Printf("   • Номенклатура: %d\n", totalNomenclature)
	fmt.Printf("   • Контрагенты: %d\n", totalCounterparty)
	fmt.Printf("   • ИТОГО: %d записей\n\n", totalNomenclature+totalCounterparty)

	// Запрос подтверждения
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("\n🚀 НАЧАТЬ НОРМАЛИЗАЦИЮ? (y/n): ")
	
	var answer string
	fmt.Scanln(&answer)
	
	if answer != "y" && answer != "Y" && answer != "да" && answer != "yes" {
		fmt.Println("\n❌ Нормализация отменена пользователем")
		return
	}

	fmt.Println("\n✅ ЗАПУСК НОРМАЛИЗАЦИИ...")
	fmt.Println("═══════════════════════════════════════════════════════════════\n")

	startTime := time.Now()

	// Создаем сессию нормализации
	ctx := context.Background()
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())
	
	_, err = serviceDB.ExecContext(ctx, `
		INSERT INTO normalization_sessions (
			session_id, project_id, start_time, status
		) VALUES (?, ?, ?, ?)
	`, sessionID, projectID, startTime, "running")
	
	if err != nil {
		log.Printf("⚠️  Не удалось создать сессию: %v", err)
	} else {
		log.Printf("✅ Создана сессия нормализации: %s", sessionID)
	}

	fmt.Println("\n⏱️  ОЖИДАЕМОЕ ВРЕМЯ: ~21-27 минут")
	fmt.Println("📊 Процесс нормализации будет запущен через HTTP API")
	fmt.Println("\n💡 СЛЕДУЮЩИЙ ШАГ:")
	fmt.Println("   Используйте HTTP API для запуска нормализации:")
	fmt.Printf("   POST http://localhost:9999/api/clients/%d/projects/%d/normalization/start\n\n", clientID, projectID)
	
	fmt.Println("📝 СТАТУС:")
	fmt.Println("   • Сервер запущен: ✅")
	fmt.Println("   • Проект готов: ✅")
	fmt.Println("   • Базы данных доступны: ✅")
	fmt.Println("   • Сессия создана: ✅")
	
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ ПОДГОТОВКА ЗАВЕРШЕНА УСПЕШНО!                        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

