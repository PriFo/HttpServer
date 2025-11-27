//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SpeedResult результат измерения скорости для одной модели
type SpeedResult struct {
	ModelID    string        `json:"model_id"`
	ModelName  string        `json:"model_name"`
	AvgLatency time.Duration `json:"avg_latency_ms"`
	MinLatency time.Duration `json:"min_latency_ms"`
	MaxLatency time.Duration `json:"max_latency_ms"`
	Runs       int           `json:"runs"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
}

// OpenRouterModel модель OpenRouter
type OpenRouterModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OpenRouterModelsResponse ответ со списком моделей
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// OpenRouterChatRequest запрос к OpenRouter API
type OpenRouterChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

// Message сообщение в чате
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterChatResponse ответ от OpenRouter API
type OpenRouterChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func main() {
	var (
		apiKeyFlag = flag.String("api-key", "", "OpenRouter API ключ (или используйте переменную окружения OPENROUTER_API_KEY)")
		runsFlag   = flag.Int("runs", 5, "Количество запросов для каждой модели")
		timeoutFlag = flag.Duration("timeout", 30*time.Second, "Таймаут для каждого запроса")
	)
	flag.Parse()

	// Получаем API ключ
	apiKey := *apiKeyFlag
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY не установлен. Используйте флаг -api-key или переменную окружения OPENROUTER_API_KEY")
	}

	runs := *runsFlag
	if runs < 1 {
		runs = 5
	}

	fmt.Println("=== Бенчмарк скорости OpenRouter API ===")
	fmt.Printf("Количество запросов на модель: %d\n", runs)
	fmt.Printf("Промпт: \"Hi\" (максимально простой для измерения скорости)\n")
	fmt.Printf("Max tokens: 10 (минимальная генерация)\n")
	fmt.Println()

	// Получаем список бесплатных моделей
	fmt.Println("Получение списка бесплатных моделей OpenRouter...")
	models, err := getFreeModels(apiKey)
	if err != nil {
		log.Fatalf("Ошибка получения списка моделей: %v", err)
	}

	if len(models) == 0 {
		log.Fatal("Не найдено бесплатных моделей")
	}

	fmt.Printf("Найдено бесплатных моделей: %d\n", len(models))
	for i, model := range models {
		fmt.Printf("  %d. %s (%s)\n", i+1, model.Name, model.ID)
	}
	fmt.Println()

	// Тестируем каждую модель
	fmt.Println("=== Запуск бенчмарка скорости ===")
	fmt.Println()

	results := make([]*SpeedResult, 0, len(models))
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup

	totalModels := len(models)
	var completedModels int64

	for i, model := range models {
		wg.Add(1)
		go func(modelID, modelName string, index int) {
			defer wg.Done()
			fmt.Printf("[%d/%d] Тестирование: %s...\n", index+1, totalModels, modelName)

			result := testModelSpeed(apiKey, modelID, modelName, runs, *timeoutFlag)

			resultsMutex.Lock()
			results = append(results, result)
			resultsMutex.Unlock()

			completed := atomic.AddInt64(&completedModels, 1)
			if result.Success {
				fmt.Printf("  ✓ [%d/%d] %s: среднее время %v (мин: %v, макс: %v)\n",
					completed, totalModels, modelName,
					result.AvgLatency.Round(time.Millisecond),
					result.MinLatency.Round(time.Millisecond),
					result.MaxLatency.Round(time.Millisecond))
			} else {
				fmt.Printf("  ✗ [%d/%d] %s: ошибка - %s\n",
					completed, totalModels, modelName, result.Error)
			}
		}(model.ID, model.Name, i)
	}

	wg.Wait()
	fmt.Println()

	// Сортируем результаты по среднему времени ответа (быстрее = лучше)
	sort.Slice(results, func(i, j int) bool {
		// Сначала успешные запросы
		if results[i].Success && !results[j].Success {
			return true
		}
		if !results[i].Success && results[j].Success {
			return false
		}
		// Если оба успешны или оба неуспешны, сортируем по времени
		return results[i].AvgLatency < results[j].AvgLatency
	})

	// Выводим результаты
	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Println("РЕЗУЛЬТАТЫ БЕНЧМАРКА СКОРОСТИ OPENROUTER API")
	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Println()

	// Заголовок таблицы
	fmt.Printf("%-40s | %-12s | %-12s | %-12s | %-8s | %-10s\n",
		"Модель", "Среднее", "Минимум", "Максимум", "Запросов", "Статус")
	fmt.Println(strings.Repeat("-", 120))

	// Данные таблицы
	for _, result := range results {
		status := "✓ OK"
		if !result.Success {
			status = "✗ FAILED"
		}

		modelName := result.ModelName
		if modelName == "" {
			modelName = result.ModelID
		}
		if len(modelName) > 38 {
			modelName = modelName[:35] + "..."
		}

		fmt.Printf("%-40s | %-12v | %-12v | %-12v | %-8d | %-10s\n",
			modelName,
			result.AvgLatency.Round(time.Millisecond),
			result.MinLatency.Round(time.Millisecond),
			result.MaxLatency.Round(time.Millisecond),
			result.Runs,
			status)
	}

	fmt.Println(strings.Repeat("-", 120))
	fmt.Println()

	// Топ-5 самых быстрых моделей
	fmt.Println("=== ТОП-5 САМЫХ БЫСТРЫХ МОДЕЛЕЙ ===")
	fmt.Println()
	topCount := 0
	for _, result := range results {
		if result.Success && topCount < 5 {
			topCount++
			modelName := result.ModelName
			if modelName == "" {
				modelName = result.ModelID
			}
			fmt.Printf("%d. %s\n", topCount, modelName)
			fmt.Printf("   Среднее время: %v\n", result.AvgLatency.Round(time.Millisecond))
			fmt.Printf("   Диапазон: %v - %v\n", result.MinLatency.Round(time.Millisecond), result.MaxLatency.Round(time.Millisecond))
			fmt.Println()
		}
		if topCount >= 5 {
			break
		}
	}

	if topCount == 0 {
		fmt.Println("Нет успешных результатов")
		fmt.Println()
	}

	// Сохраняем результаты в JSON
	saveResultsToJSON(results)
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 100))
}

// getFreeModels получает список бесплатных моделей из OpenRouter API
func getFreeModels(apiKey string) ([]OpenRouterModel, error) {
	url := "https://openrouter.ai/api/v1/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("HTTP-Referer", "https://github.com/your-repo")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var modelsResp OpenRouterModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Фильтруем только бесплатные модели (окончание ":free" или "free")
	freeModels := make([]OpenRouterModel, 0)
	for _, model := range modelsResp.Data {
		modelIDLower := strings.ToLower(model.ID)
		modelNameLower := strings.ToLower(model.Name)
		
		// Проверяем окончание ":free" (стандартный формат) или просто "free"
		if strings.HasSuffix(modelIDLower, ":free") || strings.HasSuffix(modelNameLower, ":free") ||
			strings.HasSuffix(modelIDLower, "free") || strings.HasSuffix(modelNameLower, "free") {
			freeModels = append(freeModels, model)
		}
	}

	// Если не найдено бесплатных моделей, используем fallback список
	if len(freeModels) == 0 {
		log.Printf("Не найдено бесплатных моделей в API ответе, используем fallback список")
		knownFreeModels := []OpenRouterModel{
			{ID: "meta-llama/llama-3.2-3b-instruct:free", Name: "Llama 3.2 3B Instruct (Free)"},
			{ID: "mistralai/mistral-7b-instruct:free", Name: "Mistral 7B Instruct (Free)"},
			{ID: "google/gemma-3-4b-it:free", Name: "Gemma 3 4B IT (Free)"},
			{ID: "qwen/qwen-2.5-72b-instruct:free", Name: "Qwen 2.5 72B Instruct (Free)"},
			{ID: "deepseek/deepseek-r1:free", Name: "DeepSeek R1 (Free)"},
			{ID: "x-ai/grok-4.1-fast:free", Name: "Grok 4.1 Fast (Free)"},
			{ID: "z-ai/glm-4.5-air:free", Name: "GLM 4.5 Air (Free)"},
			{ID: "google/gemini-2.0-flash-exp:free", Name: "Gemini 2.0 Flash Exp (Free)"},
		}
		return knownFreeModels, nil
	}

	return freeModels, nil
}

// testModelSpeed тестирует скорость одной модели
func testModelSpeed(apiKey, modelID, modelName string, runs int, timeout time.Duration) *SpeedResult {
	result := &SpeedResult{
		ModelID:   modelID,
		ModelName: modelName,
		Runs:      runs,
		Success:   false,
	}

	durations := make([]time.Duration, 0, runs)

	for i := 0; i < runs; i++ {
		duration, err := makeOpenRouterRequest(apiKey, modelID, timeout)
		if err == nil {
			durations = append(durations, duration)
		} else {
			log.Printf("Ошибка при запросе к %s (попытка %d/%d): %v", modelID, i+1, runs, err)
			// Продолжаем тестирование, даже если один запрос не удался
		}

		// Небольшая пауза между запросами, чтобы не нагружать API
		if i < runs-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	if len(durations) == 0 {
		result.Error = "все запросы завершились ошибкой"
		return result
	}

	// Вычисляем статистику
	var total time.Duration
	min := durations[0]
	max := durations[0]
	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	result.AvgLatency = total / time.Duration(len(durations))
	result.MinLatency = min
	result.MaxLatency = max
	result.Success = true

	return result
}

// makeOpenRouterRequest выполняет один запрос к OpenRouter API
func makeOpenRouterRequest(apiKey, modelID string, timeout time.Duration) (time.Duration, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	request := OpenRouterChatRequest{
		Model: modelID,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Hi",
			},
		},
		MaxTokens: 10, // Минимальное количество токенов для быстрого ответа
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("HTTP-Referer", "https://github.com/your-repo")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Читаем весь ответ, чтобы замерить полное время
	body, err := io.ReadAll(resp.Body)
	duration := time.Since(startTime)

	if err != nil {
		return duration, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp OpenRouterChatResponse
		json.Unmarshal(body, &errorResp)
		errorMsg := fmt.Sprintf("API returned status %d", resp.StatusCode)
		if errorResp.Error != nil {
			errorMsg = errorResp.Error.Message
		}
		return duration, fmt.Errorf("%s: %s", errorMsg, string(body))
	}

	var chatResp OpenRouterChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return duration, fmt.Errorf("failed to decode response: %w", err)
	}

	if chatResp.Error != nil {
		return duration, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return duration, fmt.Errorf("no choices in response")
	}

	return duration, nil
}

// saveResultsToJSON сохраняет результаты в JSON файл
func saveResultsToJSON(results []*SpeedResult) {
	// Преобразуем результаты в формат для JSON
	jsonResults := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		jsonResults = append(jsonResults, map[string]interface{}{
			"model_id":       result.ModelID,
			"model_name":     result.ModelName,
			"avg_latency_ms": result.AvgLatency.Milliseconds(),
			"min_latency_ms": result.MinLatency.Milliseconds(),
			"max_latency_ms": result.MaxLatency.Milliseconds(),
			"runs":           result.Runs,
			"success":        result.Success,
			"error":          result.Error,
		})
	}

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"results":   jsonResults,
	}, "", "  ")
	if err != nil {
		log.Printf("Ошибка при сериализации JSON: %v", err)
		return
	}

	filename := fmt.Sprintf("openrouter_speed_results_%s.json", time.Now().Format("20060102_150405"))
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		log.Printf("Ошибка при сохранении файла: %v", err)
		return
	}

	fmt.Printf("✓ Результаты сохранены в: %s\n", filename)
}

