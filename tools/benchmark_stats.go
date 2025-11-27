package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
)

type BenchmarkResponse struct {
	Models    []ModelResult `json:"models"`
	TestCount int           `json:"test_count"`
	Total     int           `json:"total"`
	Timestamp string        `json:"timestamp"`
	Message   string        `json:"message,omitempty"`
}

type ModelResult struct {
	Model                 string  `json:"model"`
	Priority              int     `json:"priority"`
	Speed                 float64 `json:"speed"`
	AvgResponseTimeMs     float64 `json:"avg_response_time_ms"`
	MedianResponseTimeMs  float64 `json:"median_response_time_ms"`
	P95ResponseTimeMs     float64 `json:"p95_response_time_ms"`
	MinResponseTimeMs     float64 `json:"min_response_time_ms"`
	MaxResponseTimeMs     float64 `json:"max_response_time_ms"`
	SuccessCount          int     `json:"success_count"`
	ErrorCount            int     `json:"error_count"`
	TotalRequests         int     `json:"total_requests"`
	SuccessRate           float64 `json:"success_rate"`
	Status                string  `json:"status"`
}

func main() {
	url := "http://localhost:9999/api/models/benchmark"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	fmt.Println("=== СБОР СТАТИСТИКИ БЕНЧМАРКА ===")
	fmt.Println()

	// Получаем данные
	var data BenchmarkResponse
	if _, err := os.Stat("benchmark_results.json"); err == nil {
		// Читаем из файла, если он существует
		file, err := os.Open("benchmark_results.json")
		if err != nil {
			log.Fatalf("Ошибка открытия файла: %v", err)
		}
		defer file.Close()
		json.NewDecoder(file).Decode(&data)
		fmt.Println("Данные загружены из benchmark_results.json")
	} else {
		// Получаем с сервера
		resp, err := http.Get(url)
		if err != nil {
			log.Fatalf("Ошибка запроса: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Fatalf("Ошибка сервера (%d): %s", resp.StatusCode, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			log.Fatalf("Ошибка парсинга JSON: %v", err)
		}

		// Сохраняем в файл
		file, _ := os.Create("benchmark_results.json")
		json.NewEncoder(file).Encode(data)
		file.Close()
		fmt.Println("Данные получены с сервера и сохранены в benchmark_results.json")
	}

	if len(data.Models) == 0 {
		fmt.Println("Нет данных о моделях")
		if data.Message != "" {
			fmt.Printf("Сообщение: %s\n", data.Message)
		}
		return
	}

	// Общая информация
	fmt.Println("=== ОБЩАЯ ИНФОРМАЦИЯ ===")
	fmt.Printf("Тестов выполнено: %d\n", data.TestCount)
	fmt.Printf("Всего моделей: %d\n", data.Total)
	if data.Timestamp != "" {
		fmt.Printf("Время: %s\n", data.Timestamp)
	}
	fmt.Println()

	models := data.Models

	// ТОП-10 по скорости
	fmt.Println("=== ТОП-10 МОДЕЛЕЙ ПО СКОРОСТИ (speed) ===")
	sort.Slice(models, func(i, j int) bool {
		return models[i].Speed > models[j].Speed
	})
	for i := 0; i < 10 && i < len(models); i++ {
		m := models[i]
		fmt.Printf("%2d. %s\n", i+1, m.Model)
		fmt.Printf("    Speed: %.4f | Priority: %d | Success: %.1f%% | Avg Time: %.0fms\n",
			m.Speed, m.Priority, m.SuccessRate, m.AvgResponseTimeMs)
	}
	fmt.Println()

	// ТОП-10 по успешности
	fmt.Println("=== ТОП-10 МОДЕЛЕЙ ПО УСПЕШНОСТИ (success_rate) ===")
	sort.Slice(models, func(i, j int) bool {
		return models[i].SuccessRate > models[j].SuccessRate
	})
	for i := 0; i < 10 && i < len(models); i++ {
		m := models[i]
		fmt.Printf("%2d. %s\n", i+1, m.Model)
		fmt.Printf("    Success: %.1f%% | Speed: %.4f | Avg Time: %.0fms | Status: %s\n",
			m.SuccessRate, m.Speed, m.AvgResponseTimeMs, m.Status)
	}
	fmt.Println()

	// ТОП-10 по скорости ответа
	fmt.Println("=== ТОП-10 МОДЕЛЕЙ ПО СКОРОСТИ ОТВЕТА (avg_response_time_ms) ===")
	fastModels := make([]ModelResult, 0)
	for _, m := range models {
		if m.AvgResponseTimeMs > 0 {
			fastModels = append(fastModels, m)
		}
	}
	sort.Slice(fastModels, func(i, j int) bool {
		return fastModels[i].AvgResponseTimeMs < fastModels[j].AvgResponseTimeMs
	})
	for i := 0; i < 10 && i < len(fastModels); i++ {
		m := fastModels[i]
		fmt.Printf("%2d. %s\n", i+1, m.Model)
		fmt.Printf("    Avg Time: %.0fms | Success: %.1f%% | Speed: %.4f\n",
			m.AvgResponseTimeMs, m.SuccessRate, m.Speed)
	}
	fmt.Println()

	// САМАЯ БЫСТРАЯ МОДЕЛЬ
	if len(fastModels) > 0 {
		fastest := fastModels[0]
		fmt.Println("=== САМАЯ БЫСТРАЯ МОДЕЛЬ (из всех протестированных) ===")
		fmt.Printf("Модель: %s\n", fastest.Model)
		fmt.Printf("Среднее время ответа: %.0fms\n", fastest.AvgResponseTimeMs)
		fmt.Printf("Скорость (speed): %.4f\n", fastest.Speed)
		fmt.Printf("Успешность: %.1f%%\n", fastest.SuccessRate)
		fmt.Printf("Приоритет: %d\n", fastest.Priority)
		fmt.Printf("Статус: %s\n", fastest.Status)
		fmt.Printf("Медианное время: %.0fms\n", fastest.MedianResponseTimeMs)
		fmt.Printf("P95 время: %.0fms\n", fastest.P95ResponseTimeMs)
		fmt.Printf("Мин/Макс: %.0fms / %.0fms\n", fastest.MinResponseTimeMs, fastest.MaxResponseTimeMs)
		fmt.Println()
	}

	// Статистика по статусам
	fmt.Println("=== СТАТИСТИКА ПО СТАТУСАМ ===")
	statusCount := make(map[string]int)
	for _, m := range models {
		statusCount[m.Status]++
	}
	for status, count := range statusCount {
		fmt.Printf("  %s: %d\n", status, count)
	}
	fmt.Println()

	// Общая статистика
	fmt.Println("=== ОБЩАЯ СТАТИСТИКА ===")
	var totalRequests, totalSuccess, totalErrors int
	var totalSpeed, totalResponseTime float64
	var responseTimeCount int

	for _, m := range models {
		totalRequests += m.TotalRequests
		totalSuccess += m.SuccessCount
		totalErrors += m.ErrorCount
		totalSpeed += m.Speed
		if m.AvgResponseTimeMs > 0 {
			totalResponseTime += m.AvgResponseTimeMs
			responseTimeCount++
		}
	}

	avgSpeed := totalSpeed / float64(len(models))
	avgResponseTime := totalResponseTime / float64(responseTimeCount)
	successPercent := float64(totalSuccess) / float64(totalRequests) * 100
	errorPercent := float64(totalErrors) / float64(totalRequests) * 100

	fmt.Printf("  Всего запросов: %d\n", totalRequests)
	fmt.Printf("  Успешных: %d (%.1f%%)\n", totalSuccess, successPercent)
	fmt.Printf("  Ошибок: %d (%.1f%%)\n", totalErrors, errorPercent)
	fmt.Printf("  Средняя скорость: %.4f\n", avgSpeed)
	fmt.Printf("  Среднее время ответа: %.0fms\n", avgResponseTime)
	fmt.Println()

	// Статистика по приоритетам
	fmt.Println("=== СТАТИСТИКА ПО ПРИОРИТЕТАМ ===")
	var priorities []int
	for _, m := range models {
		priorities = append(priorities, m.Priority)
	}
	sort.Ints(priorities)
	if len(priorities) > 0 {
		fmt.Printf("  Диапазон: %d - %d\n", priorities[0], priorities[len(priorities)-1])
	}

	priorityRanges := map[string]int{
		"1-10":   0,
		"11-50":  0,
		"51-100": 0,
		">100":   0,
	}

	for _, m := range models {
		if m.Priority >= 1 && m.Priority <= 10 {
			priorityRanges["1-10"]++
		} else if m.Priority >= 11 && m.Priority <= 50 {
			priorityRanges["11-50"]++
		} else if m.Priority >= 51 && m.Priority <= 100 {
			priorityRanges["51-100"]++
		} else {
			priorityRanges[">100"]++
		}
	}

	for rangeName, count := range priorityRanges {
		fmt.Printf("  Приоритет %s: %d моделей\n", rangeName, count)
	}
	fmt.Println()

	fmt.Println("Статистика собрана успешно!")
}

