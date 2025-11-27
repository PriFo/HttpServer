//go:build tool_check_api_status
// +build tool_check_api_status

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     🔍 ПРОВЕРКА СТАТУСА API                                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	baseURL := "http://localhost:9999"
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	endpoints := []struct {
		Name string
		Path string
		Method string
	}{
		{"Health Check", "/api/health", "GET"},
		{"Список клиентов", "/api/clients", "GET"},
		{"Проект AITAS", "/api/clients/1/projects/1", "GET"},
		{"Статус нормализации", "/api/clients/1/projects/1/normalization/status", "GET"},
		{"Swagger UI", "/swagger/index.html", "GET"},
	}

	workingCount := 0
	totalCount := len(endpoints)

	for i, endpoint := range endpoints {
		fmt.Printf("%d. %s\n", i+1, endpoint.Name)
		fmt.Printf("   URL: %s%s\n", baseURL, endpoint.Path)

		req, err := http.NewRequest(endpoint.Method, baseURL+endpoint.Path, nil)
		if err != nil {
			fmt.Printf("   ❌ Ошибка создания запроса: %v\n\n", err)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("   ❌ Ошибка подключения: %v\n\n", err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			fmt.Printf("   ✅ Статус: %d OK\n", resp.StatusCode)
			if len(body) < 200 {
				fmt.Printf("   Ответ: %s\n", string(body))
			} else {
				fmt.Printf("   Ответ: %s... (обрезан)\n", string(body[:200]))
			}
			workingCount++
		} else {
			fmt.Printf("   ❌ Статус: %d %s\n", resp.StatusCode, resp.Status)
			if len(body) < 100 {
				fmt.Printf("   Ошибка: %s\n", string(body))
			}
		}
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("\n📊 ИТОГО: %d/%d endpoints работают\n\n", workingCount, totalCount)

	if workingCount == 0 {
		fmt.Println("❌ API недоступен")
		fmt.Println()
		fmt.Println("🔧 РЕКОМЕНДАЦИИ:")
		fmt.Println("   1. Проверьте, что сервер запущен на порту 9999")
		fmt.Println("   2. Проверьте логи сервера на наличие ошибок")
		fmt.Println("   3. Попробуйте перезапустить сервер:")
		fmt.Println("      cd E:\\HttpServer")
		fmt.Println("      go run ./cmd/server")
		fmt.Println()
		fmt.Println("💡 АЛЬТЕРНАТИВА:")
		fmt.Println("   Используйте веб-интерфейс: http://localhost:3000")
	} else if workingCount < totalCount {
		fmt.Println("⚠️  Некоторые endpoints недоступны")
		fmt.Println("   Попробуйте использовать веб-интерфейс для запуска нормализации")
	} else {
		fmt.Println("✅ API полностью работает!")
		fmt.Println("   Можно запускать нормализацию через HTTP API")
	}
	fmt.Println()

	// Проверяем статус нормализации через API (если доступен)
	if workingCount > 0 {
		fmt.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println("📊 ПРОВЕРКА СТАТУСА НОРМАЛИЗАЦИИ:")
		fmt.Println("═══════════════════════════════════════════════════════════════")
		fmt.Println()

		req, _ := http.NewRequest("GET", baseURL+"/api/clients/1/projects/1/normalization/status", nil)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			var status map[string]interface{}
			if json.Unmarshal(body, &status) == nil {
				fmt.Println("   Статус нормализации:")
				for key, value := range status {
					fmt.Printf("     %s: %v\n", key, value)
				}
			} else {
				fmt.Printf("   Ответ: %s\n", string(body))
			}
		} else {
			fmt.Println("   ⚠️  Не удалось получить статус нормализации")
		}
		fmt.Println()
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ ПРОВЕРКА ЗАВЕРШЕНА                                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

