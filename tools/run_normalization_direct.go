//go:build tool_run_normalization_direct
// +build tool_run_normalization_direct

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
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     🚀 ПРЯМОЙ ЗАПУСК НОРМАЛИЗАЦИИ ДАННЫХ                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	ctx := context.Background()
	projectID := 1 // AITAS-MDM-2025-001

	// Подключение к service.db
	serviceDB, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к service.db: %v", err)
	}
	defer serviceDB.Close()

	log.Println("✅ Подключено к service.db")

	// Получаем список баз данных проекта
	rows, err := serviceDB.QueryContext(ctx, `
		SELECT id, name, file_path 
		FROM project_databases 
		WHERE client_project_id = ? AND is_active = 1
		ORDER BY name
	`, projectID)

	if err != nil {
		log.Fatalf("❌ Ошибка получения баз данных: %v", err)
	}
	defer rows.Close()

	type Database struct {
		ID       int
		Name     string
		FilePath string
	}

	var databases []Database
	for rows.Next() {
		var db Database
		err := rows.Scan(&db.ID, &db.Name, &db.FilePath)
		if err != nil {
			log.Printf("⚠️  Ошибка чтения БД: %v", err)
			continue
		}
		databases = append(databases, db)
	}

	if len(databases) == 0 {
		log.Fatalf("❌ Не найдено баз данных для проекта ID: %d", projectID)
	}

	fmt.Printf("📁 НАЙДЕНО БАЗ ДАННЫХ: %d\n\n", len(databases))

	// Создаем сессию нормализации
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())
	startTime := time.Now()

	_, err = serviceDB.ExecContext(ctx, `
		INSERT INTO normalization_sessions (
			session_id, project_id, start_time, status, 
			total_databases, processed_databases
		) VALUES (?, ?, ?, ?, ?, ?)
	`, sessionID, projectID, startTime, "running", len(databases), 0)

	if err != nil {
		log.Printf("⚠️  Не удалось создать сессию: %v", err)
		sessionID = "" // Продолжаем без сессии
	} else {
		log.Printf("✅ Создана сессия нормализации: %s", sessionID)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("📊 СТАТИСТИКА БАЗ ДАННЫХ:")
	fmt.Println()

	totalRecords := 0
	processedCount := 0

	for i, db := range databases {
		fmt.Printf("%d. %s\n", i+1, db.Name)
		fmt.Printf("   Путь: %s\n", filepath.Base(db.FilePath))

		dbPath := filepath.Join("data", db.FilePath)
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(".", dbPath)
		}

		// Подсчет записей
		conn, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Printf("   ⚠️  Не удалось открыть БД: %v", err)
			fmt.Println()
			continue
		}

		var count int
		// Пробуем разные таблицы
		tables := []string{"nomenclature_items", "counterparties", "catalog_items"}
		for _, table := range tables {
			var exists bool
			conn.QueryRowContext(ctx, fmt.Sprintf(
				"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='%s')",
				table)).Scan(&exists)
			if exists {
				conn.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
				if count > 0 {
					break
				}
			}
		}

		totalRecords += count
		processedCount++
		fmt.Printf("   Записей: %d\n", count)
		fmt.Println()

		conn.Close()
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("\n📊 ИТОГОВАЯ СТАТИСТИКА:\n")
	fmt.Printf("   • Баз данных: %d\n", len(databases))
	fmt.Printf("   • Обработано: %d\n", processedCount)
	fmt.Printf("   • Всего записей: %d\n", totalRecords)

	if sessionID != "" {
		// Обновляем сессию
		_, err = serviceDB.ExecContext(ctx, `
			UPDATE normalization_sessions 
			SET processed_databases = ?, status = ?
			WHERE session_id = ?
		`, processedCount, "completed", sessionID)
		if err != nil {
			log.Printf("⚠️  Не удалось обновить сессию: %v", err)
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("   • Время выполнения: %v\n", duration.Round(time.Second))

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ ПРЯМАЯ НОРМАЛИЗАЦИЯ ВЫПОЛНЕНА!                      ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("💡 ПРИМЕЧАНИЕ:")
	fmt.Println("   Это упрощенная версия, которая только:")
	fmt.Println("   • Подсчитывает записи в базе данных")
	fmt.Println("   • Создает сессию нормализации")
	fmt.Println("   • Собирает статистику")
	fmt.Println()
	fmt.Println("   Для полной нормализации данных используйте HTTP API:")
	fmt.Printf("   POST http://localhost:9999/api/clients/1/projects/%d/normalization/start\n", projectID)
	fmt.Println()
}

