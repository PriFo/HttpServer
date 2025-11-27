//go:build tool_monitor_normalization
// +build tool_monitor_normalization

package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	projectID := 1 // AITAS-MDM-2025-001

	// Подключение к service.db
	serviceDB, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к service.db: %v", err)
	}
	defer serviceDB.Close()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     📊 МОНИТОРИНГ НОРМАЛИЗАЦИИ                              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Проверяем активные сессии
	rows, err := serviceDB.Query(`
		SELECT id, database_id, start_time, status, processed_count, total_count
		FROM normalization_sessions
		WHERE project_id = ? AND status = 'running'
		ORDER BY start_time DESC
	`, projectID)

	if err != nil {
		// Возможно, таблица имеет другую структуру
		fmt.Printf("⚠️  Ошибка проверки сессий: %v\n", err)
		fmt.Println("   Возможно, нормализация еще не запущена")
		fmt.Println()
		return
	}
	defer rows.Close()

	var sessions []struct {
		ID            int
		DatabaseID    sql.NullInt64
		StartTime     time.Time
		Status        string
		ProcessedCount sql.NullInt64
		TotalCount    sql.NullInt64
	}

	for rows.Next() {
		var s struct {
			ID            int
			DatabaseID    sql.NullInt64
			StartTime     time.Time
			Status        string
			ProcessedCount sql.NullInt64
			TotalCount    sql.NullInt64
		}
		err := rows.Scan(&s.ID, &s.DatabaseID, &s.StartTime, &s.Status, &s.ProcessedCount, &s.TotalCount)
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	if len(sessions) == 0 {
		fmt.Println("📊 Текущий статус: Нормализация не запущена")
		fmt.Println()
		fmt.Println("💡 Для запуска нормализации:")
		fmt.Println("   1. Откройте веб-интерфейс: http://localhost:3000")
		fmt.Println("   2. Перейдите в проект AITAS-MDM-2025-001")
		fmt.Println("   3. Вкладка 'Нормализация' → 'Запустить нормализацию'")
		fmt.Println()
		return
	}

	fmt.Printf("🔄 АКТИВНЫХ СЕССИЙ: %d\n\n", len(sessions))

	for i, session := range sessions {
		fmt.Printf("Сессия #%d (ID: %d):\n", i+1, session.ID)
		fmt.Printf("  Статус: %s\n", session.Status)
		fmt.Printf("  Запущена: %s\n", session.StartTime.Format("2006-01-02 15:04:05"))

		if session.ProcessedCount.Valid && session.TotalCount.Valid {
			processed := session.ProcessedCount.Int64
			total := session.TotalCount.Int64
			if total > 0 {
				percent := float64(processed) / float64(total) * 100
				fmt.Printf("  Прогресс: %d / %d (%.1f%%)\n", processed, total, percent)

				// Оценка оставшегося времени
				elapsed := time.Since(session.StartTime)
				if processed > 0 {
					rate := float64(processed) / elapsed.Seconds() // записей в секунду
					remaining := float64(total-processed) / rate
					fmt.Printf("  Скорость: %.1f записей/сек\n", rate)
					fmt.Printf("  Осталось: ~%s\n", time.Duration(remaining)*time.Second)
				}
			}
		}
		fmt.Println()
	}

	// Проверяем результаты в normalized_data.db
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📊 РЕЗУЛЬТАТЫ В normalized_data.db:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	normalizedDB, err := sql.Open("sqlite3", "data/normalized_data.db")
	if err != nil {
		fmt.Printf("⚠️  Не удалось открыть normalized_data.db: %v\n", err)
		fmt.Println()
	} else {
		defer normalizedDB.Close()

		// Проверяем существование таблицы
		var tableExists bool
		normalizedDB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name='normalized_data'
			)
		`).Scan(&tableExists)

		if tableExists {
			var totalCount int
			var normalizedCount int
			var projectCount int

			normalizedDB.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&totalCount)
			normalizedDB.QueryRow(`
				SELECT COUNT(*) FROM normalized_data 
				WHERE normalized_name IS NOT NULL AND normalized_name != ''
			`).Scan(&normalizedCount)
			normalizedDB.QueryRow(`
				SELECT COUNT(*) FROM normalized_data 
				WHERE project_id = ?
			`, projectID).Scan(&projectCount)

			fmt.Printf("  Всего записей: %d\n", totalCount)
			fmt.Printf("  Нормализовано: %d\n", normalizedCount)
			fmt.Printf("  Для проекта %d: %d\n", projectID, projectCount)

			if normalizedCount > 0 && totalCount > 0 {
				percent := float64(normalizedCount) / float64(totalCount) * 100
				fmt.Printf("  Процент: %.1f%%\n", percent)
			}
		} else {
			fmt.Println("  ⚠️  Таблица normalized_data еще не создана")
		}
		fmt.Println()
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ МОНИТОРИНГ ЗАВЕРШЕН                                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("💡 Запустите этот скрипт периодически для отслеживания прогресса:")
	fmt.Println("   go run tools/monitor_normalization.go")
	fmt.Println()
}

