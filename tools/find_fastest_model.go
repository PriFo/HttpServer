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
}

type ModelResult struct {
	Model                string  `json:"model"`
	AvgResponseTimeMs    float64 `json:"avg_response_time_ms"`
	Speed                float64 `json:"speed"`
	SuccessRate          float64 `json:"success_rate"`
	Priority             int     `json:"priority"`
	Status               string  `json:"status"`
	MedianResponseTimeMs float64 `json:"median_response_time_ms"`
	P95ResponseTimeMs    float64 `json:"p95_response_time_ms"`
	MinResponseTimeMs    float64 `json:"min_response_time_ms"`
	MaxResponseTimeMs    float64 `json:"max_response_time_ms"`
}

func main() {
	url := "http://localhost:9999/api/models/benchmark"
	
	var data BenchmarkResponse
	var err error
	
	// Пробуем получить с сервера
	resp, err := http.Get(url)
	if err != nil {
		// Если сервер недоступен, пробуем прочитать из файла
		files := []string{"benchmark_results.json", "benchmark_data.json"}
		for _, filename := range files {
			file, fileErr := os.Open(filename)
			if fileErr == nil {
				defer file.Close()
				if jsonErr := json.NewDecoder(file).Decode(&data); jsonErr == nil && len(data.Models) > 0 {
					fmt.Printf("Данные загружены из %s\n\n", filename)
					goto process
				}
			}
		}
		log.Fatalf("Ошибка запроса к серверу: %v\nУбедитесь, что сервер запущен на localhost:9999 или есть файл с данными", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Ошибка сервера (%d): %s", resp.StatusCode, string(body))
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Fatalf("Ошибка парсинга JSON: %v", err)
	}
	
process:
	
	if len(data.Models) == 0 {
		log.Fatal("Нет данных о моделях в ответе")
	}
	
	fmt.Printf("=== АНАЛИЗ ВСЕХ %d МОДЕЛЕЙ ===\n\n", len(data.Models))
	
	// Фильтруем модели с валидным временем ответа
	validModels := make([]ModelResult, 0)
	for _, m := range data.Models {
		if m.AvgResponseTimeMs > 0 {
			validModels = append(validModels, m)
		}
	}
	
	if len(validModels) == 0 {
		log.Fatal("Нет моделей с валидным временем ответа")
	}
	
	// Сортируем по времени ответа
	sort.Slice(validModels, func(i, j int) bool {
		return validModels[i].AvgResponseTimeMs < validModels[j].AvgResponseTimeMs
	})
	
	// Самая быстрая модель
	fastest := validModels[0]
	
	fmt.Println("=== САМАЯ БЫСТРАЯ МОДЕЛЬ (из всех протестированных) ===")
	fmt.Printf("Модель: %s\n", fastest.Model)
	fmt.Printf("Среднее время ответа: %.0fms\n", fastest.AvgResponseTimeMs)
	fmt.Printf("Медианное время: %.0fms\n", fastest.MedianResponseTimeMs)
	fmt.Printf("P95 время: %.0fms\n", fastest.P95ResponseTimeMs)
	fmt.Printf("Мин/Макс: %.0fms / %.0fms\n", fastest.MinResponseTimeMs, fastest.MaxResponseTimeMs)
	fmt.Printf("Скорость (speed): %.4f\n", fastest.Speed)
	fmt.Printf("Успешность: %.1f%%\n", fastest.SuccessRate)
	fmt.Printf("Приоритет: %d\n", fastest.Priority)
	fmt.Printf("Статус: %s\n", fastest.Status)
	fmt.Println()
	
	// ТОП-10 самых быстрых
	fmt.Println("=== ТОП-10 САМЫХ БЫСТРЫХ МОДЕЛЕЙ ===")
	for i := 0; i < 10 && i < len(validModels); i++ {
		m := validModels[i]
		fmt.Printf("%2d. %s\n", i+1, m.Model)
		fmt.Printf("    Время: %.0fms | Успешность: %.1f%% | Speed: %.4f | Приоритет: %d\n",
			m.AvgResponseTimeMs, m.SuccessRate, m.Speed, m.Priority)
	}
	fmt.Println()
	
	// Статистика
	fmt.Println("=== СТАТИСТИКА ===")
	fmt.Printf("Всего моделей проанализировано: %d\n", len(validModels))
	fmt.Printf("Самое быстрое время: %.0fms (%s)\n", fastest.AvgResponseTimeMs, fastest.Model)
	
	slowest := validModels[len(validModels)-1]
	fmt.Printf("Самое медленное время: %.0fms (%s)\n", slowest.AvgResponseTimeMs, slowest.Model)
	
	var sum float64
	for _, m := range validModels {
		sum += m.AvgResponseTimeMs
	}
	avg := sum / float64(len(validModels))
	fmt.Printf("Среднее время ответа: %.0fms\n", avg)
}

