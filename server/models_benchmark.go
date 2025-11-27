package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/ для организации,
// но остается в пакете server для доступа к методам Server
// TODO:legacy-migration revisit dependencies after handler extraction

import (
	"context"
	"encoding/json"
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

	"httpserver/internal/infrastructure/ai"
	"httpserver/nomenclature"
	"httpserver/normalization"
)

// handleModelsBenchmark запускает бенчмарк всех доступных моделей
func (s *Server) handleModelsBenchmark(w http.ResponseWriter, r *http.Request) {
	// Поддерживаем GET для получения последних результатов и POST для запуска нового бенчмарка
	if r.Method == http.MethodGet {
		// Получаем параметры запроса
		limitStr := r.URL.Query().Get("limit")
		modelName := r.URL.Query().Get("model")
		history := r.URL.Query().Get("history") == "true"

		if history && s.serviceDB != nil {
			// Возвращаем историю бенчмарков
			limit := 100
			if limitStr != "" {
				if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && parsedLimit == 1 {
					// limit уже установлен
				}
			}

			historyData, err := s.serviceDB.GetBenchmarkHistory(limit, modelName)
			if err != nil {
				s.writeJSONError(w, r, fmt.Sprintf("Failed to get benchmark history: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"history": historyData,
				"total":   len(historyData),
			}
			s.writeJSONResponse(w, r, response, http.StatusOK)
			return
		}

		// Возвращаем последние результаты из истории (если есть)
		if s.serviceDB != nil {
			// Получаем достаточно записей, чтобы собрать все модели из последнего бенчмарка
			historyData, err := s.serviceDB.GetBenchmarkHistory(100, modelName)
			if err != nil {
				log.Printf("[Benchmark] Failed to get benchmark history: %v", err)
			} else if len(historyData) > 0 {
				// Группируем результаты по timestamp (все модели из одного бенчмарка имеют одинаковый timestamp)
				// Берем самый последний timestamp
				if len(historyData) > 0 {
					lastTimestamp, _ := historyData[0]["timestamp"].(string)
					var models []map[string]interface{}
					var testCount int
					
					// Собираем все модели с последним timestamp
					for _, record := range historyData {
						recordTimestamp, _ := record["timestamp"].(string)
						if recordTimestamp == lastTimestamp {
							// Безопасное извлечение значений
							testCountVal, _ := getIntValue(record["test_count"])
							if testCount == 0 {
								testCount = testCountVal
							}
							
							model := map[string]interface{}{
								"model":                 record["model"],
								"priority":              record["priority"],
								"speed":                 record["speed"],
								"avg_response_time_ms":   record["avg_response_time_ms"],
								"median_response_time_ms": record["median_response_time_ms"],
								"p95_response_time_ms":   record["p95_response_time_ms"],
								"min_response_time_ms":   record["min_response_time_ms"],
								"max_response_time_ms":   record["max_response_time_ms"],
								"success_count":          record["success_count"],
								"error_count":            record["error_count"],
								"total_requests":          record["total_requests"],
								"success_rate":           record["success_rate"],
								"status":                 record["status"],
							}
							models = append(models, model)
						} else {
							// Дошли до следующего timestamp, останавливаемся
							break
						}
					}
					
					if len(models) > 0 {
						// Сортируем по приоритету
						sort.Slice(models, func(i, j int) bool {
							priI, _ := getIntValue(models[i]["priority"])
							priJ, _ := getIntValue(models[j]["priority"])
							return priI < priJ
						})
						
						response := map[string]interface{}{
							"models":     models,
							"total":      len(models),
							"test_count": testCount,
							"timestamp":  lastTimestamp,
							"message":    "Last benchmark result from history. Use POST to run new benchmark or ?history=true to get full history",
						}
						s.writeJSONResponse(w, r, response, http.StatusOK)
						return
					}
				}
			}
		}

		// Если нет истории, возвращаем пустой ответ
		response := map[string]interface{}{
			"models":     []map[string]interface{}{},
			"total":      0,
			"test_count": 0,
			"timestamp":  time.Now(),
			"message":    "No benchmark results found. Use POST to run benchmark or ?history=true to get history",
		}
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем опции из запроса
	type BenchmarkRequest struct {
		AutoUpdatePriorities    bool     `json:"auto_update_priorities"`
		TestProducts            []string `json:"test_products"`            // Кастомные тестовые данные
		MaxRetries              int      `json:"max_retries"`              // Максимум попыток для каждого запроса
		RetryDelayMS            int      `json:"retry_delay_ms"`            // Задержка между попытками в миллисекундах
		Models                  []string `json:"models"`                    // Список моделей для тестирования (если пусто - все)
		Provider                string   `json:"provider"`                  // Провайдер для тестирования (если указан - только его модели)
		UseNormalizationDataset bool     `json:"use_normalization_dataset"` // Использовать датасет из нормализации
		DatasetLimit            int      `json:"dataset_limit"`            // Лимит записей из датасета (по умолчанию 50)
		Database                string   `json:"database"`                  // Путь к базе данных для датасета
	}
	
	var reqOptions BenchmarkRequest
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &reqOptions); err != nil {
				log.Printf("[Benchmark] Failed to parse request body: %v, using defaults", err)
			}
		}
	}
	
	// Устанавливаем значения по умолчанию
	autoUpdatePriorities := reqOptions.AutoUpdatePriorities
	maxRetries := reqOptions.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5 // По умолчанию 5 попыток
	}
	retryDelayMS := reqOptions.RetryDelayMS
	if retryDelayMS <= 0 {
		retryDelayMS = 200 // По умолчанию 200ms
	}

	// Получаем список моделей
	allModels, err := s.getAvailableModels()
	if err != nil {
		log.Printf("[Benchmark] Error getting models: %v", err)
		// Формируем более информативное сообщение об ошибке
		errorMsg := fmt.Sprintf("Failed to get models: %v", err)
		if strings.Contains(strings.ToLower(err.Error()), "api key") || strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
			errorMsg = "Failed to get models: API key may be invalid or expired. Please check ARLIAI_API_KEY configuration."
		} else if strings.Contains(strings.ToLower(err.Error()), "network") || strings.Contains(strings.ToLower(err.Error()), "connection") {
			errorMsg = "Failed to get models: Network error. Please check your internet connection and API endpoint availability."
		} else if strings.Contains(strings.ToLower(err.Error()), "timeout") {
			errorMsg = "Failed to get models: Request timeout. The API may be slow or unavailable. Please try again later."
		}
		s.writeJSONError(w, r, errorMsg, http.StatusInternalServerError)
		return
	}

	if len(allModels) == 0 {
		log.Printf("[Benchmark] No models available for benchmarking")
		s.writeJSONError(w, r, "No models available. Please check ARLIAI_API_KEY and ensure models are configured. If you have a subscription, make sure all models are enabled in the provider configuration.", http.StatusNotFound)
		return
	}
	
	// Валидация: предупреждение, если моделей слишком мало
	if len(allModels) <= 2 {
		log.Printf("[Benchmark] WARNING: Only %d models available. Expected more models from Arliai API.", len(allModels))
		log.Printf("[Benchmark] This might indicate filtering or API limitations. Check logs for details.")
	}

	// Фильтруем модели по провайдеру, если указан
	if reqOptions.Provider != "" {
		if s.workerConfigManager != nil {
			config := s.workerConfigManager.GetConfig()
			if providersMap, ok := config["providers"].(map[string]interface{}); ok {
				if providerData, ok := providersMap[reqOptions.Provider].(map[string]interface{}); ok {
					if modelsData, ok := providerData["models"].([]interface{}); ok {
						providerModels := make([]string, 0)
						for _, m := range modelsData {
							if modelMap, ok := m.(map[string]interface{}); ok {
								if modelName, ok := modelMap["name"].(string); ok && modelName != "" {
									providerModels = append(providerModels, modelName)
								}
							}
						}
						if len(providerModels) > 0 {
							// Фильтруем allModels, оставляя только модели провайдера
							providerModelMap := make(map[string]bool)
							for _, pm := range providerModels {
								providerModelMap[pm] = true
							}
							filteredByProvider := make([]string, 0)
							for _, m := range allModels {
								if providerModelMap[m] {
									filteredByProvider = append(filteredByProvider, m)
								}
							}
							if len(filteredByProvider) > 0 {
								allModels = filteredByProvider
								log.Printf("[Benchmark] Filtered by provider %s: %d models", reqOptions.Provider, len(allModels))
							} else {
								log.Printf("[Benchmark] WARNING: Provider %s specified but no matching models found in available models", reqOptions.Provider)
							}
						}
					}
				}
			}
		}
	}

	// Фильтруем модели, если указаны конкретные
	models := allModels
	if len(reqOptions.Models) > 0 {
		models = make([]string, 0)
		modelMap := make(map[string]bool)
		for _, m := range reqOptions.Models {
			modelMap[m] = true
		}
		for _, m := range allModels {
			if modelMap[m] {
				models = append(models, m)
			}
		}
		if len(models) == 0 {
			s.writeJSONError(w, r, fmt.Sprintf("None of the specified models (%v) are available. Please check model names and ensure they are enabled in the provider configuration. Available models can be retrieved from /api/workers/models endpoint.", reqOptions.Models), http.StatusBadRequest)
			return
		}
		log.Printf("[Benchmark] Filtered models: %v (from %v)", models, allModels)
	}

	// Тестовые данные - используем датасет из нормализации, кастомные или дефолтные
	testProducts := reqOptions.TestProducts
	if len(testProducts) == 0 && reqOptions.UseNormalizationDataset {
		// Получаем датасет из нормализации
		datasetLimit := reqOptions.DatasetLimit
		if datasetLimit <= 0 {
			datasetLimit = 50
		}
		if datasetLimit > 500 {
			datasetLimit = 500
			log.Printf("[Benchmark] Dataset limit exceeded maximum, capped at 500")
		}

		// Получаем порт из конфигурации или используем дефолтный
		port := "9999"
		if s.config != nil && s.config.Port != "" {
			port = s.config.Port
		}

		// Формируем URL для получения датасета
		datasetURL := fmt.Sprintf("http://localhost:%s/api/normalization/benchmark-dataset?limit=%d", port, datasetLimit)
		if reqOptions.Database != "" {
			// Валидация database path для предотвращения path traversal
			if strings.Contains(reqOptions.Database, "..") || strings.Contains(reqOptions.Database, "~") {
				log.Printf("[Benchmark] WARNING: Invalid database path detected, ignoring: %s", reqOptions.Database)
			} else {
				datasetURL += "&database=" + reqOptions.Database
			}
		}

		// Создаем HTTP клиент с таймаутом
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		// Создаем контекст с таймаутом для запроса
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", datasetURL, nil)
		if err != nil {
			log.Printf("[Benchmark] Failed to create request for dataset: %v", err)
		} else {
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("[Benchmark] Failed to fetch normalization dataset: %v", err)
			} else {
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					var datasetResp map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&datasetResp); err != nil {
						log.Printf("[Benchmark] Failed to decode dataset response: %v", err)
					} else {
						if data, ok := datasetResp["data"].([]interface{}); ok {
							testProducts = make([]string, 0, len(data))
							for _, item := range data {
								if name, ok := item.(string); ok && name != "" {
									testProducts = append(testProducts, name)
								}
							}
							if len(testProducts) > 0 {
								log.Printf("[Benchmark] Loaded %d items from normalization dataset", len(testProducts))
							} else {
								log.Printf("[Benchmark] Dataset returned empty array")
							}
						} else {
							log.Printf("[Benchmark] Dataset response missing 'data' field or invalid format")
						}
					}
				} else {
					log.Printf("[Benchmark] Dataset endpoint returned status %d", resp.StatusCode)
				}
			}
		}
	}

	// Если все еще нет данных, используем дефолтные
	if len(testProducts) == 0 {
		// Дефолтные тестовые данные
		testProducts = []string{
			"Болт М8х20",
			"Гайка М8",
			"Шайба плоская М8",
			"Винт саморез 4.2х16",
			"Гвоздь строительный 100мм",
			"Саморез по дереву 4.5х50",
			"Дюбель распорный 8х50",
			"Анкерный болт М10х100",
			"Шуруп по металлу 4.2х19",
			"Заклепка вытяжная 4х8",
			"Болт с гайкой М10",
			"Шпилька резьбовая М12",
			"Винт с потайной головкой",
			"Гайка самоконтрящаяся",
			"Шайба пружинная",
		}
	}
	
	log.Printf("[Benchmark] Starting benchmark with %d models, %d test products, max_retries=%d, retry_delay=%dms", 
		len(models), len(testProducts), maxRetries, retryDelayMS)

	// Получаем API ключ из конфигурации воркеров (из БД) или из переменной окружения
	var apiKey string
	if s.workerConfigManager != nil {
		var err error
		apiKey, _, err = s.workerConfigManager.GetModelAndAPIKey()
		if err != nil {
			log.Printf("Failed to get API key from worker config: %v, trying environment variable", err)
			apiKey = os.Getenv("ARLIAI_API_KEY")
		}
	} else {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	
	if apiKey == "" {
		log.Printf("[Benchmark] ERROR: ARLIAI_API_KEY not configured. Worker config manager: %v", s.workerConfigManager != nil)
		s.writeJSONError(w, r, "ARLIAI_API_KEY not configured. Please set it in worker configuration or environment variable. The API key is required to access all available models.", http.StatusServiceUnavailable)
		return
	}

	// Получаем или создаем кэшированное KpvedTree для всех моделей
	// Это критически важно для предотвращения блокировок SQLite при параллельных бенчмарках
	sharedTree := s.getOrCreateKpvedTree()
	if sharedTree == nil {
		log.Printf("[Benchmark] ERROR: Failed to get or create KPVED tree")
		s.writeJSONError(w, r, "Failed to build KPVED tree. Please ensure the database contains KPVED classifier data.", http.StatusInternalServerError)
		return
	}
	nodeCount := len(sharedTree.NodeMap)
	sectionCount := len(sharedTree.Root.Children)
	log.Printf("[Benchmark] Using shared KPVED tree: %d nodes, %d sections (root children)", nodeCount, sectionCount)

	// Бенчмарк для каждой модели - обрабатываем параллельно
	results := make([]map[string]interface{}, 0, len(models))
	var resultsMutex sync.Mutex
	var modelsWg sync.WaitGroup

	// Ограничиваем параллелизм моделей для предотвращения перегрузки системы и блокировок SQLite
	// SQLite плохо справляется с большим количеством одновременных соединений
	// Уменьшаем до 2 для еще большей стабильности при работе с БД
	const maxModelWorkers = 2 // Максимум 2 модели одновременно (уменьшено для предотвращения блокировок SQLite)
	modelSem := make(chan struct{}, maxModelWorkers)

	for _, modelName := range models {
		modelsWg.Add(1)
		modelSem <- struct{}{} // Захватываем слот
		go func(name string) {
			defer func() {
				<-modelSem // Освобождаем слот
				modelsWg.Done()
			}()
			log.Printf("[Benchmark] Starting benchmark for model: %s", name)
			benchmark := s.testModelBenchmark(apiKey, name, testProducts, maxRetries, time.Duration(retryDelayMS)*time.Millisecond, sharedTree)
			resultsMutex.Lock()
			results = append(results, benchmark)
			resultsMutex.Unlock()
			log.Printf("[Benchmark] Completed benchmark for model: %s (success: %v, speed: %.2f req/s)", 
				name, benchmark["success_count"], benchmark["speed"])
		}(modelName)
	}

	// Ждем завершения всех бенчмарков моделей
	modelsWg.Wait()
	log.Printf("[Benchmark] All model benchmarks completed. Total models: %d", len(results))
	
	// Подсчитываем статистику
	var totalSuccess int64
	var totalErrors int64
	var successfulModels int
	var failedModels int
	for _, result := range results {
		if successCount, ok := result["success_count"].(int64); ok {
			totalSuccess += successCount
		} else if successCount, ok := result["success_count"].(float64); ok {
			totalSuccess += int64(successCount)
		}
		if errorCount, ok := result["error_count"].(int64); ok {
			totalErrors += errorCount
		} else if errorCount, ok := result["error_count"].(float64); ok {
			totalErrors += int64(errorCount)
		}
		if status, ok := result["status"].(string); ok {
			if status == "ok" || status == "partial" {
				successfulModels++
			} else {
				failedModels++
			}
		}
	}
	log.Printf("[Benchmark] Statistics: %d successful models, %d failed models, %d total successes, %d total errors", 
		successfulModels, failedModels, totalSuccess, totalErrors)
	
	// Детальная статистика по типам ошибок
	var quotaErrors int
	var rateLimitErrors int
	var timeoutErrors int
	var networkErrors int
	var authErrors int
	var otherErrors int
	
	for _, result := range results {
		if errorCount, ok := result["error_count"].(int64); ok && errorCount > 0 {
			// Анализируем ошибки модели (если есть детали)
			if status, ok := result["status"].(string); ok && status == "failed" {
				// Можно добавить более детальный анализ, если в result есть информация об ошибках
				otherErrors++
			}
		}
	}
	
	log.Printf("[Benchmark] Error breakdown: quota=%d, rate_limit=%d, timeout=%d, network=%d, auth=%d, other=%d", 
		quotaErrors, rateLimitErrors, timeoutErrors, networkErrors, authErrors, otherErrors)

	// Сортируем модели по комплексной оценке качества
	// Приоритет: success_rate > avg_confidence > speed > stability
	sort.Slice(results, func(i, j int) bool {
		// Получаем метрики для модели i
		successRateI, _ := getFloatValue(results[i]["success_rate"])
		avgConfidenceI, _ := getFloatValue(results[i]["avg_confidence"])
		speedI, _ := getFloatValue(results[i]["speed"])
		coeffVarI, _ := getFloatValue(results[i]["coefficient_of_variation"])
		
		// Получаем метрики для модели j
		successRateJ, _ := getFloatValue(results[j]["success_rate"])
		avgConfidenceJ, _ := getFloatValue(results[j]["avg_confidence"])
		speedJ, _ := getFloatValue(results[j]["speed"])
		coeffVarJ, _ := getFloatValue(results[j]["coefficient_of_variation"])
		
		// 1. Приоритет: success_rate (чем выше, тем лучше)
		if successRateI != successRateJ {
			return successRateI > successRateJ
		}
		
		// 2. При одинаковом success_rate: avg_confidence (чем выше, тем лучше)
		if avgConfidenceI != avgConfidenceJ {
			return avgConfidenceI > avgConfidenceJ
		}
		
		// 3. При одинаковой уверенности: speed (чем выше, тем лучше)
		if speedI != speedJ {
			return speedI > speedJ
		}
		
		// 4. При одинаковой скорости: стабильность (чем меньше коэффициент вариации, тем лучше)
		return coeffVarI < coeffVarJ
	})

	// Устанавливаем приоритеты на основе скорости
	for i := range results {
		results[i]["priority"] = i + 1
	}

	// Обновляем приоритеты в конфигурации, если запрошено
	updatedPriorities := false
	if autoUpdatePriorities && s.workerConfigManager != nil {
		updatedPriorities = s.updateModelPrioritiesFromBenchmark(results)
		if !updatedPriorities {
			log.Printf("[Benchmark] WARNING: Failed to update model priorities from benchmark results")
		} else {
			log.Printf("[Benchmark] Successfully updated model priorities from benchmark results")
		}
	}

	// Сохраняем результаты в историю
	if s.serviceDB != nil {
		// Добавляем timestamp к каждому результату
		timestamp := time.Now().Format(time.RFC3339)
		for i := range results {
			results[i]["timestamp"] = timestamp
		}
		if err := s.serviceDB.SaveBenchmarkHistory(results, len(testProducts)); err != nil {
			log.Printf("Failed to save benchmark history: %v", err)
		}
	}

	// Вычисляем общую статистику для ответа
	totalRequests := len(testProducts) * len(models)
	overallSuccessRate := 0.0
	if totalRequests > 0 {
		overallSuccessRate = float64(totalSuccess) / float64(totalRequests) * 100.0
	}
	
	// Формируем информативное сообщение для пользователя
	message := fmt.Sprintf("Benchmark completed: %d models tested, %d successful, %d failed", 
		len(results), successfulModels, failedModels)
	if len(allModels) > len(models) {
		message += fmt.Sprintf(" (filtered from %d available models)", len(allModels))
	}
	if overallSuccessRate < 50.0 {
		message += ". WARNING: Low success rate - check API keys, rate limits, and quota. Some models may have exceeded their quota or rate limits."
	}
	if len(allModels) <= 2 {
		message += ". NOTE: Only 2 models available - check if API returns all models. MaxWorkers=2 is a limit on parallel requests, not on the number of models."
	}
	if totalErrors > 0 && float64(totalErrors)/float64(totalRequests) > 0.3 {
		message += ". WARNING: High error rate detected - check API keys, network connectivity, and provider status."
	}
	
	response := map[string]interface{}{
		"models":               results,
		"total":                len(results),
		"test_count":           len(testProducts),
		"timestamp":            time.Now(),
		"priorities_updated":   updatedPriorities,
		"message":              message,
		"statistics": map[string]interface{}{
			"successful_models":    successfulModels,
			"failed_models":        failedModels,
			"total_successes":      totalSuccess,
			"total_errors":         totalErrors,
			"total_requests":       totalRequests,
			"overall_success_rate": overallSuccessRate,
			"models_tested":        len(models),
			"models_available":     len(allModels),
		},
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// getAvailableModels получает список доступных моделей
// Получает все модели напрямую из Arliai API, включая все статусы (active, deprecated, beta)
// и все модели независимо от подписки (бесплатные и платные)
func (s *Server) getAvailableModels() ([]string, error) {
	// Очищаем кеш перед получением моделей для бенчмарка, чтобы получить свежие данные
	if s.arliaiCache != nil {
		s.arliaiCache.Clear()
		log.Printf("[Benchmark] Cleared Arliai cache to get fresh models")
	}
	
	// Пытаемся получить все модели напрямую из Arliai API
	if s.arliaiClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		traceID := fmt.Sprintf("benchmark-%d-%d", time.Now().UnixNano(), time.Now().Unix())
		
		// Пробуем разные варианты query параметров для получения всех моделей
		var aiModels []ai.ArliaiModel
		var err error
		
		// Вариант 1: Пробуем с параметром status=all
		log.Printf("[Benchmark] Attempting to get models with status=all")
		aiModels, err = s.arliaiClient.GetModels(ctx, traceID, "status=all")
		if err == nil && len(aiModels) > 2 {
			log.Printf("[Benchmark] Successfully got %d models with status=all", len(aiModels))
		} else {
			log.Printf("[Benchmark] status=all returned %d models or error: %v, trying include=all", len(aiModels), err)
			// Вариант 2: Пробуем с параметром include=all
			aiModels, err = s.arliaiClient.GetModels(ctx, traceID, "include=all")
			if err == nil && len(aiModels) > 2 {
				log.Printf("[Benchmark] Successfully got %d models with include=all", len(aiModels))
			} else {
				log.Printf("[Benchmark] include=all returned %d models or error: %v, trying all=true", len(aiModels), err)
				// Вариант 3: Пробуем с параметром all=true
				aiModels, err = s.arliaiClient.GetModels(ctx, traceID, "all=true")
				if err == nil && len(aiModels) > 2 {
					log.Printf("[Benchmark] Successfully got %d models with all=true", len(aiModels))
				} else {
					log.Printf("[Benchmark] all=true returned %d models or error: %v, trying without params", len(aiModels), err)
					// Вариант 4: Пробуем без параметров (по умолчанию)
					aiModels, err = s.arliaiClient.GetModels(ctx, traceID)
					if err == nil {
						log.Printf("[Benchmark] Got %d models without params", len(aiModels))
					}
				}
			}
		}
		if err == nil && len(aiModels) > 0 {
			// Конвертируем ai.ArliaiModel в server.ArliaiModel
			apiModels := make([]ArliaiModel, 0, len(aiModels))
			for _, aiModel := range aiModels {
				apiModels = append(apiModels, ArliaiModel{
					ID:          aiModel.ID,
					Name:        aiModel.Name,
					Speed:       aiModel.Speed,
					Quality:     aiModel.Quality,
					Description: aiModel.Description,
					Status:      aiModel.Status,
					MaxTokens:   aiModel.MaxTokens,
					Tags:        aiModel.Tags,
				})
			}
			
			models := make([]string, 0, len(apiModels))
			modelSet := make(map[string]bool) // Для исключения дубликатов
			
			for _, model := range apiModels {
				// Используем ID или Name модели
				modelName := model.ID
				if model.Name != "" {
					modelName = model.Name
				}
				
				// Пропускаем пустые имена
				if modelName == "" {
					continue
				}
				
				// Добавляем все модели, независимо от статуса (active, deprecated, beta)
				// и независимо от подписки (бесплатные и платные)
				if !modelSet[modelName] {
					models = append(models, modelName)
					modelSet[modelName] = true
				}
			}
			
			if len(models) > 0 {
				log.Printf("[Benchmark] Got %d models from Arliai API (all statuses, all subscription types)", len(models))
				// Логируем первые несколько моделей для отладки
				if len(models) <= 10 {
					log.Printf("[Benchmark] Models: %v", models)
				} else {
					log.Printf("[Benchmark] First 10 models: %v ... (total: %d)", models[:10], len(models))
				}
				
				// Предупреждение, если получили мало моделей
				if len(models) <= 2 {
					log.Printf("[Benchmark] WARNING: Only %d models retrieved. Expected more models from Arliai API.", len(models))
					log.Printf("[Benchmark] This might indicate that API is filtering models or returning only active/enabled models.")
					log.Printf("[Benchmark] Check API response and consider using different query parameters.")
				}
				
				return models, nil
			} else {
				log.Printf("[Benchmark] Arliai API returned %d models but all were empty or duplicates", len(apiModels))
			}
		} else {
			log.Printf("[Benchmark] Failed to get models from Arliai API: %v, trying internal API endpoint", err)
			
			// Альтернативный способ: используем внутренний API сервера
			// Это может помочь, если прямой вызов API имеет ограничения
			if s.arliaiClient != nil && s.arliaiClient.GetAPIKey() != "" {
				// Получаем порт из конфигурации или используем дефолтный
				port := "9999"
				if s.config != nil && s.config.Port != "" {
					port = s.config.Port
				}
				internalURL := fmt.Sprintf("http://localhost:%s/api/workers/models?enabled=all&status=all", port)
				log.Printf("[Benchmark] Attempting to get models via internal API: %s", internalURL)
				
				ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel2()
				
				req, reqErr := http.NewRequestWithContext(ctx2, "GET", internalURL, nil)
				if reqErr != nil {
					log.Printf("[Benchmark] Failed to create request for internal API: %v", reqErr)
				} else {
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("X-Request-ID", traceID)
					
					client := &http.Client{Timeout: 10 * time.Second}
					resp, respErr := client.Do(req)
					if respErr != nil {
						log.Printf("[Benchmark] Failed to execute request to internal API: %v", respErr)
					} else if resp.StatusCode != http.StatusOK {
						log.Printf("[Benchmark] Internal API returned status %d", resp.StatusCode)
						resp.Body.Close()
					} else {
						var apiResp struct {
							Success bool `json:"success"`
							Data    struct {
								Models []struct {
									Name   string `json:"name"`
									ID     string `json:"id"`
									Status string `json:"status"`
								} `json:"models"`
							} `json:"data"`
						}
						decodeErr := json.NewDecoder(resp.Body).Decode(&apiResp)
						resp.Body.Close()
						
						if decodeErr != nil {
							log.Printf("[Benchmark] Failed to decode internal API response: %v", decodeErr)
						} else if apiResp.Success && len(apiResp.Data.Models) > 0 {
							models := make([]string, 0, len(apiResp.Data.Models))
							for _, m := range apiResp.Data.Models {
								modelName := m.Name
								if modelName == "" {
									modelName = m.ID
								}
								if modelName != "" {
									models = append(models, modelName)
								}
							}
							if len(models) > 0 {
								log.Printf("[Benchmark] Got %d models via internal API endpoint", len(models))
								return models, nil
							} else {
								log.Printf("[Benchmark] Internal API returned models but all were empty")
							}
						} else {
							log.Printf("[Benchmark] Internal API response unsuccessful or empty: success=%v, models=%d", apiResp.Success, len(apiResp.Data.Models))
						}
					}
				}
			}
			
			log.Printf("[Benchmark] Internal API also failed, falling back to config: %v", err)
		}
	}

	// Fallback: получаем модели из конфигурации (включая не включенные для бенчмарка)
	if s.workerConfigManager != nil {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err != nil {
			log.Printf("[Benchmark] Failed to get active provider from config: %v", err)
		} else {
			models := make([]string, 0)
			for _, model := range provider.Models {
				// Для бенчмарка включаем все модели, не только enabled
				if model.Name != "" {
					models = append(models, model.Name)
				}
			}
			if len(models) > 0 {
				log.Printf("[Benchmark] Got %d models from config (including disabled)", len(models))
				return models, nil
			} else {
				log.Printf("[Benchmark] Config provider has no models")
			}
		}
	}

	// Последний fallback: известные модели
	knownModels := []string{
		"GLM-4.5-Air",
		"GLM-4.5",
		"GLM-4",
		"GLM-3-Turbo",
		"GLM-3-6B",
		"Gemma-3-27B-ArliAI-RPMax-v3",
	}
	log.Printf("[Benchmark] Using fallback models: %v", knownModels)
	return knownModels, nil
}

// testModelBenchmark тестирует одну модель и возвращает результаты
// Все запросы обрабатываются параллельно для максимальной производительности
// sharedTree - переиспользуемое дерево KPVED для всех моделей (избегает множественных запросов к БД)
func (s *Server) testModelBenchmark(apiKey, modelName string, testProducts []string, maxRetries int, retryDelay time.Duration, sharedTree *normalization.KpvedTree) map[string]interface{} {
	// Создаем контекст с отменой и таймаутом для всего бенчмарка
	// Таймаут: 5 минут на модель (достаточно для большого количества запросов)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel() // Отменяем контекст при завершении функции для очистки всех горутин

	startTime := time.Now()
	var successCount int64
	var errorCount int64
	var totalDuration int64
	var minTime time.Duration = time.Hour
	var maxTime time.Duration

	// Создаем AI клиент для этой модели
	aiClient := nomenclature.NewAIClient(apiKey, modelName)

	// Создаем иерархический классификатор с переиспользуемым деревом
	// Это избегает множественных запросов к БД для каждой модели
	hierarchicalClassifier := normalization.NewHierarchicalClassifierWithTree(sharedTree, s.serviceDB, aiClient)

	// Мьютекс для защиты minTime и maxTime
	var timeMutex sync.Mutex
	
	// Канал для сбора времен ответов (потокобезопасный)
	responseTimesChan := make(chan time.Duration, len(testProducts))
	
	// Каналы для сбора метрик качества
	confidencesChan := make(chan float64, len(testProducts))
	aiCallsCountChan := make(chan int, len(testProducts))
	retriesChan := make(chan int, len(testProducts))
	
	// Счетчики по типам ошибок
	var quotaErrorsCount int64
	var rateLimitErrorsCount int64
	var timeoutErrorsCount int64
	var networkErrorsCount int64
	var authErrorsCount int64
	var otherErrorsCount int64
	
	// Используем WaitGroup для ожидания завершения всех goroutines
	var wg sync.WaitGroup
	
	// Не ограничиваем параллелизм для AI API запросов - rate limiter сам контролирует поток
	// Rate limiter настроен на 5 запросов/сек, что достаточно для контроля
	// Ограничение параллелизма моделей (maxModelWorkers) предотвращает перегрузку SQLite
	
	// Запускаем все запросы параллельно с retry механизмом
	// Параметры передаются из запроса для гибкой настройки
	if maxRetries <= 0 {
		maxRetries = 5 // Fallback на дефолтное значение
	}
	if retryDelay <= 0 {
		retryDelay = 200 * time.Millisecond // Fallback на дефолтное значение
	}
	
	log.Printf("[Benchmark] Model %s: Starting parallel benchmark for %d products with %d max retries, retry_delay=%v (rate limiter controls concurrency)", 
		modelName, len(testProducts), maxRetries, retryDelay)
	
	for i, product := range testProducts {
		wg.Add(1)
		go func(productName string, productIndex int) {
			// Обработка паники для гарантии завершения горутины
			defer func() {
				if panicVal := recover(); panicVal != nil {
					log.Printf("[Benchmark] Model %s: PANIC in goroutine for '%s' (product %d/%d): %v", 
						modelName, productName, productIndex+1, len(testProducts), panicVal)
					atomic.AddInt64(&errorCount, 1)
					atomic.AddInt64(&otherErrorsCount, 1)
				}
				wg.Done()   // Всегда вызываем Done, даже при панике
			}()
			
			// Проверяем отмену контекста перед началом работы
			if ctx.Err() != nil {
				log.Printf("[Benchmark] Model %s: Context cancelled for '%s' (product %d/%d)", 
					modelName, productName, productIndex+1, len(testProducts))
				atomic.AddInt64(&errorCount, 1)
				return
			}
			
			var reqDuration time.Duration
			var err error
			var success bool
			var classificationResult *normalization.HierarchicalResult
			var totalRetries int
			requestStartTime := time.Now() // Общее время всех попыток
			
			// Повторяем запрос до maxRetries раз
			for attempt := 0; attempt < maxRetries; attempt++ {
				// Проверяем отмену контекста перед каждой попыткой
				if ctx.Err() != nil {
					log.Printf("[Benchmark] Model %s: Context cancelled during retry for '%s' (attempt %d/%d)", 
						modelName, productName, attempt+1, maxRetries)
					break
				}
				
				reqStart := time.Now()
				// Выполняем вызов API с защитой от паники
				func() {
					defer func() {
						if panicVal := recover(); panicVal != nil {
							log.Printf("[Benchmark] Model %s: PANIC during API call for '%s' (attempt %d/%d): %v", 
								modelName, productName, attempt+1, maxRetries, panicVal)
							err = fmt.Errorf("panic during API call: %v", panicVal)
						}
					}()
					classificationResult, err = hierarchicalClassifier.ClassifyWithContext(ctx, productName, "общее")
				}()
				reqDuration = time.Since(reqStart)
				totalRetries = attempt + 1 // Считаем попытки (1-based)

				if err == nil {
					success = true
					// Используем время успешной попытки
					reqDuration = time.Since(requestStartTime)
					break // Успешный запрос - выходим из цикла
				}
				
				// Если это не последняя попытка, ждем перед повтором
				if attempt < maxRetries-1 {
					// Экспоненциальная задержка: 200ms, 400ms, 800ms, 1600ms
					delay := retryDelay * time.Duration(1<<uint(attempt))
					
					// Используем context-aware sleep для возможности отмены
					select {
					case <-ctx.Done():
						log.Printf("[Benchmark] Model %s: Context cancelled during retry delay for '%s'", 
							modelName, productName)
						// Выходим из цикла retry при отмене контекста
						attempt = maxRetries // Устанавливаем attempt в maxRetries для выхода из цикла
					case <-time.After(delay):
						log.Printf("[Benchmark] Model %s: Retry %d/%d for '%s' after %v (error: %v)", 
							modelName, attempt+1, maxRetries, productName, delay, err)
					}
				} else {
					// Последняя попытка - логируем финальную ошибку
					reqDuration = time.Since(requestStartTime) // Общее время всех попыток
					log.Printf("[Benchmark] Model %s: Final attempt failed for '%s' after %d attempts (total time: %v, error: %v)", 
						modelName, productName, maxRetries, reqDuration, err)
				}
			}

			// Атомарно обновляем счетчики
			atomic.AddInt64(&totalDuration, int64(reqDuration))
			
			// Защищаем minTime и maxTime мьютексом
			timeMutex.Lock()
			if reqDuration < minTime {
				minTime = reqDuration
			}
			if reqDuration > maxTime {
				maxTime = reqDuration
			}
			timeMutex.Unlock()

			if !success {
				atomic.AddInt64(&errorCount, 1)
				
				// Классифицируем тип ошибки для лучшего логирования и подсчета
				// Безопасно получаем строку ошибки (err может быть nil в редких случаях)
				errStr := "unknown error"
				if err != nil {
					errStr = err.Error()
				}
				errorType := "unknown"
				if strings.Contains(strings.ToLower(errStr), "quota") || 
				   strings.Contains(strings.ToLower(errStr), "quota exceeded") {
					errorType = "quota_exceeded"
					atomic.AddInt64(&quotaErrorsCount, 1)
				} else if strings.Contains(strings.ToLower(errStr), "rate limit") || 
				          strings.Contains(strings.ToLower(errStr), "429") ||
				          strings.Contains(strings.ToLower(errStr), "too many requests") {
					errorType = "rate_limit"
					atomic.AddInt64(&rateLimitErrorsCount, 1)
				} else if strings.Contains(strings.ToLower(errStr), "timeout") ||
				          strings.Contains(strings.ToLower(errStr), "deadline exceeded") {
					errorType = "timeout"
					atomic.AddInt64(&timeoutErrorsCount, 1)
				} else if strings.Contains(strings.ToLower(errStr), "network") ||
				          strings.Contains(strings.ToLower(errStr), "connection") {
					errorType = "network"
					atomic.AddInt64(&networkErrorsCount, 1)
				} else if strings.Contains(strings.ToLower(errStr), "api key") ||
				          strings.Contains(strings.ToLower(errStr), "unauthorized") ||
				          strings.Contains(strings.ToLower(errStr), "401") {
					errorType = "auth"
					atomic.AddInt64(&authErrorsCount, 1)
				} else {
					atomic.AddInt64(&otherErrorsCount, 1)
				}
				
				// Отправляем количество retry попыток даже при ошибке
				retriesChan <- totalRetries
				
				log.Printf("[Benchmark] Model %s: Failed to classify '%s' (product %d/%d) after %d attempts [error_type: %s]: %v", 
					modelName, productName, productIndex+1, len(testProducts), maxRetries, errorType, err)
			} else {
				atomic.AddInt64(&successCount, 1)
				// Отправляем время ответа в канал
				responseTimesChan <- reqDuration
				
				// Собираем метрики качества из результата классификации
				if classificationResult != nil {
					confidencesChan <- classificationResult.FinalConfidence
					aiCallsCountChan <- classificationResult.AICallsCount
				} else {
					// Если результат nil (не должно быть, но на всякий случай)
					confidencesChan <- 0.0
					aiCallsCountChan <- 0
				}
				retriesChan <- totalRetries
				
				log.Printf("[Benchmark] Model %s: Successfully classified '%s' (product %d/%d) in %v (confidence: %.2f, AI calls: %d)", 
					modelName, productName, productIndex+1, len(testProducts), reqDuration, 
					classificationResult.FinalConfidence, classificationResult.AICallsCount)
			}
		}(product, i)
	}

	// Ждем завершения всех goroutines
	wg.Wait()
	
	// Явно отменяем контекст для гарантии завершения всех операций
	cancel()
	
	// Закрываем каналы после завершения всех горутин
	close(responseTimesChan)
	close(confidencesChan)
	close(aiCallsCountChan)
	close(retriesChan)

	// Собираем времена ответов из канала
	responseTimes := make([]time.Duration, 0, len(testProducts))
	for duration := range responseTimesChan {
		responseTimes = append(responseTimes, duration)
	}
	
	// Собираем метрики качества
	confidences := make([]float64, 0, len(testProducts))
	for confidence := range confidencesChan {
		confidences = append(confidences, confidence)
	}
	
	aiCallsCounts := make([]int, 0, len(testProducts))
	for aiCalls := range aiCallsCountChan {
		aiCallsCounts = append(aiCallsCounts, aiCalls)
	}
	
	retries := make([]int, 0, len(testProducts))
	for retryCount := range retriesChan {
		retries = append(retries, retryCount)
	}
	
	totalTime := time.Since(startTime)
	
	// Получаем финальные значения из атомарных счетчиков
	finalSuccessCount := atomic.LoadInt64(&successCount)
	finalErrorCount := atomic.LoadInt64(&errorCount)
	finalTotalDuration := atomic.LoadInt64(&totalDuration)
	
	log.Printf("[Benchmark] Model %s: Parallel benchmark completed - Success: %d, Errors: %d, Total: %d, Duration: %v", 
		modelName, finalSuccessCount, finalErrorCount, len(testProducts), totalTime)
	
	speed := 0.0
	avgTime := time.Duration(0)
	if finalSuccessCount > 0 && totalTime.Seconds() > 0 {
		speed = float64(finalSuccessCount) / totalTime.Seconds()
		avgTime = time.Duration(finalTotalDuration) / time.Duration(finalSuccessCount)
	}

	// Рассчитываем перцентили времени ответа (P50, P75, P90, P95, P99)
	medianTime := time.Duration(0)
	p75Time := time.Duration(0)
	p90Time := time.Duration(0)
	p95Time := time.Duration(0)
	p99Time := time.Duration(0)
	coefficientOfVariation := 0.0 // Коэффициент вариации для оценки стабильности
	
	if len(responseTimes) > 0 {
		// Используем sort.Slice для эффективной сортировки
		sortedTimes := make([]time.Duration, len(responseTimes))
		copy(sortedTimes, responseTimes)
		sort.Slice(sortedTimes, func(i, j int) bool {
			return sortedTimes[i] < sortedTimes[j]
		})
		
		// Функция для получения перцентиля
		getPercentile := func(sorted []time.Duration, percentile float64) time.Duration {
			if len(sorted) == 0 {
				return 0
			}
			idx := int(float64(len(sorted)) * percentile)
			if idx >= len(sorted) {
				idx = len(sorted) - 1
			}
			if idx < 0 {
				idx = 0
			}
			return sorted[idx]
		}
		
		// P50 (медиана)
		medianTime = getPercentile(sortedTimes, 0.50)
		// P75
		p75Time = getPercentile(sortedTimes, 0.75)
		// P90
		p90Time = getPercentile(sortedTimes, 0.90)
		// P95
		p95Time = getPercentile(sortedTimes, 0.95)
		// P99
		p99Time = getPercentile(sortedTimes, 0.99)
		
		// Коэффициент вариации (стандартное отклонение / среднее значение)
		if avgTime > 0 {
			var sumSquaredDiff int64
			for _, t := range responseTimes {
				diff := int64(t) - int64(avgTime)
				sumSquaredDiff += diff * diff
			}
			variance := float64(sumSquaredDiff) / float64(len(responseTimes))
			stdDev := time.Duration(int64(variance))
			if avgTime > 0 {
				coefficientOfVariation = float64(stdDev) / float64(avgTime)
			}
		}
	}
	
	// Рассчитываем метрики качества (уверенность)
	avgConfidence := 0.0
	minConfidence := 1.0
	maxConfidence := 0.0
	if len(confidences) > 0 {
		var sumConfidence float64
		for _, conf := range confidences {
			sumConfidence += conf
			if conf < minConfidence {
				minConfidence = conf
			}
			if conf > maxConfidence {
				maxConfidence = conf
			}
		}
		avgConfidence = sumConfidence / float64(len(confidences))
	}
	
	// Среднее количество AI вызовов на запрос
	avgAICalls := 0.0
	if len(aiCallsCounts) > 0 {
		var sumAICalls int
		for _, calls := range aiCallsCounts {
			sumAICalls += calls
		}
		avgAICalls = float64(sumAICalls) / float64(len(aiCallsCounts))
	}
	
	// Среднее количество retry попыток
	avgRetries := 0.0
	if len(retries) > 0 {
		var sumRetries int
		for _, retry := range retries {
			sumRetries += retry
		}
		avgRetries = float64(sumRetries) / float64(len(retries))
	}
	
	// Получаем финальные значения счетчиков ошибок
	finalQuotaErrors := atomic.LoadInt64(&quotaErrorsCount)
	finalRateLimitErrors := atomic.LoadInt64(&rateLimitErrorsCount)
	finalTimeoutErrors := atomic.LoadInt64(&timeoutErrorsCount)
	finalNetworkErrors := atomic.LoadInt64(&networkErrorsCount)
	finalAuthErrors := atomic.LoadInt64(&authErrorsCount)
	finalOtherErrors := atomic.LoadInt64(&otherErrorsCount)

	successRate := 0.0
	if len(testProducts) > 0 {
		successRate = float64(finalSuccessCount) / float64(len(testProducts)) * 100
	}

	status := "ok"
	if finalErrorCount > 0 && finalSuccessCount == 0 {
		status = "failed"
	} else if finalErrorCount > 0 {
		status = "partial"
	}

	return map[string]interface{}{
		"model":                modelName,
		"status":               status,
		"success_count":        finalSuccessCount,
		"error_count":          finalErrorCount,
		"total_requests":       len(testProducts),
		"success_rate":         successRate,
		
		// Метрики производительности
		"speed":                    speed, // requests per second
		"throughput_items_per_sec": speed, // alias для ясности
		"avg_response_time_ms":     avgTime.Milliseconds(),
		"median_response_time_ms":  medianTime.Milliseconds(),
		"p75_response_time_ms":     p75Time.Milliseconds(),
		"p90_response_time_ms":     p90Time.Milliseconds(),
		"p95_response_time_ms":     p95Time.Milliseconds(),
		"p99_response_time_ms":     p99Time.Milliseconds(),
		"min_response_time_ms":     minTime.Milliseconds(),
		"max_response_time_ms":     maxTime.Milliseconds(),
		"total_time_ms":            totalTime.Milliseconds(),
		"coefficient_of_variation":  coefficientOfVariation, // Стабильность (меньше = стабильнее)
		
		// Метрики качества классификации
		"avg_confidence":     avgConfidence,
		"min_confidence":     minConfidence,
		"max_confidence":     maxConfidence,
		"avg_ai_calls_count": avgAICalls, // Среднее количество AI вызовов на запрос
		
		// Метрики надежности
		"avg_retries": avgRetries, // Среднее количество попыток до успеха
		
		// Детальная статистика по типам ошибок
		"error_breakdown": map[string]interface{}{
			"quota_exceeded": finalQuotaErrors,
			"rate_limit":     finalRateLimitErrors,
			"timeout":        finalTimeoutErrors,
			"network":        finalNetworkErrors,
			"auth":           finalAuthErrors,
			"other":          finalOtherErrors,
		},
	}
}

// updateModelPrioritiesFromBenchmark обновляет приоритеты моделей в конфигурации на основе результатов бенчмарка
func (s *Server) updateModelPrioritiesFromBenchmark(benchmarks []map[string]interface{}) bool {
	if s.workerConfigManager == nil {
		return false
	}

	provider, err := s.workerConfigManager.GetActiveProvider()
	if err != nil {
		log.Printf("Failed to get active provider: %v", err)
		return false
	}

	updated := false
	for _, benchmark := range benchmarks {
		modelName, ok := benchmark["model"].(string)
		if !ok {
			continue
		}

		priority, ok := benchmark["priority"].(int)
		if !ok {
			// Пробуем float64
			if priorityFloat, ok := benchmark["priority"].(float64); ok {
				priority = int(priorityFloat)
			} else {
				continue
			}
		}

		// Ищем модель в провайдере
		for i := range provider.Models {
			if provider.Models[i].Name == modelName {
				oldPriority := provider.Models[i].Priority
				provider.Models[i].Priority = priority
				
				// Обновляем модель через менеджер
				if err := s.workerConfigManager.UpdateModel(provider.Name, modelName, &provider.Models[i]); err != nil {
					log.Printf("Failed to update model %s priority: %v", modelName, err)
					provider.Models[i].Priority = oldPriority // Откатываем изменение
				} else {
					log.Printf("Updated model %s priority from %d to %d", modelName, oldPriority, priority)
					updated = true
				}
				break
			}
		}
	}

	return updated
}

// getIntValue извлекает int значение из interface{}
func getIntValue(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

// getFloatValue извлекает float64 значение из interface{}
func getFloatValue(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0.0, false
	}
}


