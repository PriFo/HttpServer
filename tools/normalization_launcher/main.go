package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     🚀 ЗАПУСК И МОНИТОРИНГ НОРМАЛИЗАЦИИ                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	projectID := 1 // AITAS-MDM-2025-001
	clientID := 1  // AITAS KZ

	fmt.Println("🔍 ШАГ 1: ПРОВЕРКА ГОТОВНОСТИ СИСТЕМЫ")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	serviceDB, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к service.db: %v", err)
	}
	defer serviceDB.Close()

	var activeDBs int
	serviceDB.QueryRow(`
		SELECT COUNT(*) FROM project_databases 
		WHERE client_project_id = ? AND is_active = 1
	`, projectID).Scan(&activeDBs)

	if activeDBs == 0 {
		log.Fatalf("❌ Нет активных баз данных для проекта %d", projectID)
	}

	fmt.Printf("✅ Активных баз данных: %d\n\n", activeDBs)

	fmt.Println("🔍 ШАГ 2: ПРОВЕРКА API")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	apiURL := "http://localhost:9999"
	client := &http.Client{Timeout: 5 * time.Second}

	req, _ := http.NewRequest("GET", apiURL+"/api/clients", nil)
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("❌ API недоступен: %v\n\n", err)
		fmt.Println("💡 РЕКОМЕНДАЦИИ:")
		fmt.Println("   1. Запустите сервер:")
		fmt.Println("      cd E:\\HttpServer")
		fmt.Println("      go run ./cmd/server")
		fmt.Println()
		fmt.Println("   2. Или откройте веб-интерфейс:")
		fmt.Println("      http://localhost:3000")
		fmt.Println()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("⚠️  API отвечает с кодом: %d\n", resp.StatusCode)
		fmt.Println("   Попробуйте использовать веб-интерфейс\n")
	} else {
		fmt.Printf("✅ API доступен: %s\n\n", apiURL)
	}

	fmt.Println("🔍 ШАГ 3: ПРОВЕРКА СТАТУСА НОРМАЛИЗАЦИИ")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	statusURL := fmt.Sprintf("%s/api/clients/%d/projects/%d/normalization/status", apiURL, clientID, projectID)
	req2, _ := http.NewRequest("GET", statusURL, nil)
	resp2, err := client.Do(req2)

	if err == nil && resp2.StatusCode == 200 {
		defer resp2.Body.Close()
		body, _ := io.ReadAll(resp2.Body)

		var status map[string]interface{}
		if json.Unmarshal(body, &status) == nil {
			fmt.Println("   Текущий статус:")
			for key, value := range status {
				fmt.Printf("     %s: %v\n", key, value)
			}
		}
	} else {
		fmt.Println("   ⚠️  Не удалось получить статус (нормализация может быть не запущена)")
	}
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("🚀 ИНСТРУКЦИИ ПО ЗАПУСКУ НОРМАЛИЗАЦИИ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Вариант 1: Через веб-интерфейс (рекомендуется)")
	fmt.Println("   1. Откройте: http://localhost:3000")
	fmt.Println("   2. Проекты → AITAS-MDM-2025-001")
	fmt.Println("   3. Вкладка «Нормализация» → «Запустить нормализацию»")
	fmt.Println("   4. Выберите «Все активные базы данных»")
	fmt.Println()
	fmt.Println("Вариант 2: Через HTTP API")
	fmt.Printf("   POST %s/api/clients/%d/projects/%d/normalization/start\n", apiURL, clientID, projectID)
	fmt.Println("   Body: {\"all_active\": true}")
	fmt.Println()

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📊 МОНИТОРИНГ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Для мониторинга прогресса используйте:")
	fmt.Println("   go run tools/monitor_normalization.go")
	fmt.Println()
	fmt.Printf("   Или API: GET %s/api/clients/%d/projects/%d/normalization/status\n", apiURL, clientID, projectID)
	fmt.Println()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ СИСТЕМА ГОТОВА К ЗАПУСКУ НОРМАЛИЗАЦИИ!              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
