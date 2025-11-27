package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен из server/server_kpved_reclassify.go для организации,
// но остается в пакете server для доступа к методам Server
// TODO:legacy-migration revisit dependencies after handler extraction

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"httpserver/nomenclature"
	"httpserver/normalization"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// kpvedReclassifyRequest представляет запрос на переклассификацию
type kpvedReclassifyRequest struct {
	Limit int `json:"limit"` // Количество групп для переклассификации (0 = все)
}

// kpvedReclassifyValidationResult представляет результат валидации
type kpvedReclassifyValidationResult struct {
	IsValid                 bool
	TotalGroups             int
	TotalGroupsWithoutKpved int
	GroupsWithKpved         int
	Error                   error
	ErrorMessage            string
	HTTPStatus              int
}

// validateKpvedReclassifyRequest валидирует запрос на переклассификацию
func (s *Server) validateKpvedReclassifyRequest(r *http.Request) (kpvedReclassifyRequest, error) {
	var req kpvedReclassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Limit = 10 // По умолчанию 10 групп
	}
	return req, nil
}

// validateKpvedDatabaseState проверяет состояние БД перед переклассификацией
func (s *Server) validateKpvedDatabaseState() kpvedReclassifyValidationResult {
	result := kpvedReclassifyValidationResult{IsValid: true}

	// Проверяем наличие данных в normalized_data
	var totalGroups int
	err := s.db.QueryRow("SELECT COUNT(DISTINCT normalized_name || '|' || category) FROM normalized_data").Scan(&totalGroups)
	if err != nil {
		log.Printf("[KPVED] Error counting total groups: %v", err)
	} else {
		log.Printf("[KPVED] Total groups in normalized_data: %d", totalGroups)
	}
	result.TotalGroups = totalGroups

	// Проверяем наличие таблицы kpved_classifier в сервисной БД
	var kpvedTableExists bool
	err = s.serviceDB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='kpved_classifier'
		)
	`).Scan(&kpvedTableExists)
	if err != nil {
		log.Printf("[KPVED] Error checking kpved_classifier table: %v", err)
		result.IsValid = false
		result.Error = err
		result.ErrorMessage = "Failed to check kpved_classifier table"
		result.HTTPStatus = http.StatusInternalServerError
		return result
	}

	if !kpvedTableExists {
		log.Printf("[KPVED] ERROR: Table kpved_classifier does not exist in service DB!")
		result.IsValid = false
		result.ErrorMessage = "KPVED classifier table not found"
		result.HTTPStatus = http.StatusInternalServerError
		return result
	}

	// Проверяем количество записей в kpved_classifier
	var kpvedNodesCount int
	err = s.serviceDB.QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&kpvedNodesCount)
	if err != nil {
		log.Printf("[KPVED] Error counting kpved_classifier nodes: %v", err)
		result.IsValid = false
		result.Error = err
		result.ErrorMessage = "Failed to count kpved_classifier nodes"
		result.HTTPStatus = http.StatusInternalServerError
		return result
	}

	log.Printf("[KPVED] KPVED classifier nodes in database: %d", kpvedNodesCount)
	if kpvedNodesCount == 0 {
		log.Printf("[KPVED] ERROR: kpved_classifier table is empty!")
		result.IsValid = false
		result.ErrorMessage = "Таблица kpved_classifier пуста. Загрузите классификатор КПВЭД через эндпоинт /api/kpved/load-from-file или используйте файл КПВЭД.txt"
		result.HTTPStatus = http.StatusInternalServerError
		return result
	}

	// Подсчитываем общее количество групп без КПВЭД
	countQuery := `
		SELECT COUNT(DISTINCT normalized_name || '|' || category)
		FROM normalized_data
		WHERE (kpved_code IS NULL OR kpved_code = '' OR TRIM(kpved_code) = '')
	`
	err = s.db.QueryRow(countQuery).Scan(&result.TotalGroupsWithoutKpved)
	if err != nil {
		log.Printf("[KPVED] Error counting groups without KPVED: %v", err)
		result.TotalGroupsWithoutKpved = 0
	}

	// Подсчитываем группы с KPVED
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT normalized_name || '|' || category)
		FROM normalized_data
		WHERE kpved_code IS NOT NULL AND kpved_code != '' AND TRIM(kpved_code) != ''
	`).Scan(&result.GroupsWithKpved)
	if err != nil {
		log.Printf("[KPVED] Error counting groups with KPVED: %v", err)
		result.GroupsWithKpved = 0
	}

	return result
}

// createKpvedClassifier создает AI клиент и иерархический классификатор
func (s *Server) createKpvedClassifier(apiKey, model string) (*nomenclature.AIClient, *normalization.HierarchicalClassifier, error) {
	log.Printf("[KPVED] Creating hierarchical classifier with API key (length: %d) and model: %s", len(apiKey), model)
	if apiKey == "" {
		log.Printf("[KPVED] ERROR: API key is empty!")
		return nil, nil, fmt.Errorf("API ключ не настроен. Настройте API ключ в конфигурации воркеров или переменной окружения ARLIAI_API_KEY")
	}

	log.Printf("[KPVED] Initializing AI client and hierarchical classifier...")

	// Проверяем, не работает ли нормализация одновременно
	s.normalizerMutex.RLock()
	isNormalizerRunning := s.normalizerRunning
	s.normalizerMutex.RUnlock()

	if isNormalizerRunning {
		log.Printf("[KPVED] WARNING: Normalizer is running! This may cause exceeding Arliai API limit (2 parallel calls for ADVANCED plan)")
		log.Printf("[KPVED] Consider stopping normalization before starting KPVED classification")
	}

	// Создаем один AI клиент и один hierarchical classifier для всех воркеров
	aiClient := nomenclature.NewAIClient(apiKey, model)
	log.Printf("[KPVED] Created AI client instance (rate limiter: 1 req/sec, burst: 5)")

	hierarchicalClassifier, err := normalization.NewHierarchicalClassifier(s.serviceDB, aiClient)
	if err != nil {
		log.Printf("[KPVED] ERROR creating hierarchical classifier: %v", err)
		return nil, nil, fmt.Errorf("Не удалось создать классификатор: %w. Проверьте, что таблица kpved_classifier загружена и содержит данные", err)
	}
	log.Printf("[KPVED] Hierarchical classifier created successfully (will be shared by all workers)")

	// Получаем статистику дерева КПВЭД
	cacheStats := hierarchicalClassifier.GetCacheStats()
	log.Printf("[KPVED] Hierarchical classifier created successfully. Cache stats: %+v", cacheStats)

	return aiClient, hierarchicalClassifier, nil
}

// getKpvedClassificationTasks получает задачи для классификации из БД
func (s *Server) getKpvedClassificationTasks(limit int) ([]ClassificationTask, error) {
	query := `
		SELECT normalized_name, category, MAX(merged_count) as merged_count
		FROM normalized_data
		WHERE (kpved_code IS NULL OR kpved_code = '' OR TRIM(kpved_code) = '')
		GROUP BY normalized_name, category
		ORDER BY merged_count DESC
		LIMIT ?
	`

	limitValue := limit
	if limitValue == 0 {
		limitValue = 1000000 // Большое число для "все"
	}

	log.Printf("[KPVED] Querying groups without KPVED classification (limit: %d, sorted by merged_count DESC)...", limitValue)
	rows, err := s.db.Query(query, limitValue)
	if err != nil {
		log.Printf("[KPVED] Error querying groups: %v", err)
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var tasks []ClassificationTask
	index := 0
	for rows.Next() {
		var normalizedName, category string
		var mergedCount int
		if err := rows.Scan(&normalizedName, &category, &mergedCount); err != nil {
			log.Printf("[KPVED] Error scanning row: %v", err)
			continue
		}
		tasks = append(tasks, ClassificationTask{
			NormalizedName: normalizedName,
			Category:       category,
			MergedCount:    mergedCount,
			Index:          index,
		})
		index++
	}

	if err := rows.Err(); err != nil {
		log.Printf("[KPVED] Error iterating rows: %v", err)
		return nil, fmt.Errorf("failed to read groups from database: %w", err)
	}

	return tasks, nil
}

// setupKpvedWorkers определяет количество воркеров для классификации
func (s *Server) setupKpvedWorkers() int {
	maxWorkers := 2 // Arliai API ограничение: максимум 2 параллельных вызова (НЕ ограничение на количество моделей)

	// Проверяем, не работает ли нормализация одновременно
	s.normalizerMutex.RLock()
	isNormalizerRunning := s.normalizerRunning
	s.normalizerMutex.RUnlock()

	if isNormalizerRunning {
		log.Printf("[KPVED] WARNING: Normalizer is running simultaneously. Reducing workers to 1 to avoid exceeding Arliai API limit (2 parallel calls for ADVANCED plan)")
		maxWorkers = 1
	}

	// Проверяем настройки из WorkerConfigManager
	if s.workerConfigManager != nil {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err == nil && provider != nil {
			if provider.MaxWorkers > 0 && provider.MaxWorkers < maxWorkers {
				log.Printf("[KPVED] Using provider MaxWorkers=%d (requested %d)", provider.MaxWorkers, maxWorkers)
				maxWorkers = provider.MaxWorkers
			}
		}
		// Также проверяем глобальное значение
		globalMaxWorkers := s.workerConfigManager.GetGlobalMaxWorkers()
		if globalMaxWorkers > 0 && globalMaxWorkers < maxWorkers {
			log.Printf("[KPVED] Using global MaxWorkers=%d (requested %d)", globalMaxWorkers, maxWorkers)
			maxWorkers = globalMaxWorkers
		}
	}

	// Не превышаем лимит, но используем максимум доступных воркеров
	if maxWorkers > 2 {
		log.Printf("[KPVED] WARNING: Requested %d workers, but Arliai API ADVANCED plan supports only 2 parallel calls TOTAL. Limiting to 2 workers.", maxWorkers)
		maxWorkers = 2
	}
	// Обеспечиваем минимум 1 воркер
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	log.Printf("[KPVED] Using %d workers for classification (normalizer running: %v, Arliai API limit: 2 parallel calls)", maxWorkers, isNormalizerRunning)
	log.Printf("[KPVED] IMPORTANT: All %d workers will share the same AI client instance to respect rate limits", maxWorkers)

	if isNormalizerRunning && maxWorkers >= 2 {
		log.Printf("[KPVED] CRITICAL: Normalizer is running AND using %d workers! This will likely exceed Arliai API limit (2 parallel calls)", maxWorkers)
		log.Printf("[KPVED] Total parallel requests: %d (normalizer) + %d (KPVED workers) = %d (limit: 2)", 1, maxWorkers, maxWorkers+1)
	}

	return maxWorkers
}

// retryUpdate выполняет UPDATE запрос с повторными попытками при ошибках блокировки
func (s *Server) retryUpdate(query string, args ...interface{}) (sql.Result, error) {
	maxRetries := 5
	baseDelay := 10 * time.Millisecond
	queryTimeout := 10 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		result, err := s.db.ExecContext(ctx, query, args...)
		cancel()

		if err == nil {
			return result, nil
		}

		errStr := err.Error()
		isLockError := strings.Contains(errStr, "database is locked") || strings.Contains(errStr, "locked")
		isTimeoutError := strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context deadline exceeded")

		if !isLockError && !isTimeoutError {
			return nil, err
		}

		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			if isTimeoutError {
				log.Printf("[KPVED] Query timeout, retrying in %v (attempt %d/%d)...", delay, attempt+1, maxRetries)
			} else {
				log.Printf("[KPVED] Database locked, retrying in %v (attempt %d/%d)...", delay, attempt+1, maxRetries)
			}
			time.Sleep(delay)
		} else {
			log.Printf("[KPVED] Max retries reached for database update")
			return nil, err
		}
	}

	return nil, fmt.Errorf("failed after %d retries", maxRetries)
}

// processKpvedClassificationTasks обрабатывает задачи классификации с использованием worker pool
func (s *Server) processKpvedClassificationTasks(
	tasks []ClassificationTask,
	hierarchicalClassifier *normalization.HierarchicalClassifier,
	maxWorkers int,
) []classificationResult {
	taskChan := make(chan ClassificationTask, maxWorkers*2)
	resultChan := make(chan classificationResult, maxWorkers*2)
	var wg sync.WaitGroup

	// Очищаем map для отслеживания текущих задач перед началом новой классификации
	s.kpvedCurrentTasksMutex.Lock()
	s.kpvedCurrentTasks = make(map[int]*ClassificationTask)
	s.kpvedCurrentTasksMutex.Unlock()

	// Сбрасываем флаг остановки при начале новой классификации
	s.kpvedWorkersStopMutex.Lock()
	s.kpvedWorkersStopped = false
	s.kpvedWorkersStopMutex.Unlock()

	// Запускаем воркеры
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go s.kpvedWorker(i, taskChan, resultChan, hierarchicalClassifier, &wg)
	}

	// Отправляем задачи в канал
	go func() {
		for _, task := range tasks {
			taskChan <- task
		}
		close(taskChan)
	}()

	// Закрываем канал результатов после завершения всех воркеров
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Собираем результаты
	var results []classificationResult
	for res := range resultChan {
		results = append(results, res)
	}

	return results
}

// kpvedWorker обрабатывает задачи классификации в отдельной горутине
func (s *Server) kpvedWorker(
	workerID int,
	taskChan <-chan ClassificationTask,
	resultChan chan<- classificationResult,
	hierarchicalClassifier *normalization.HierarchicalClassifier,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for task := range taskChan {
		// Проверяем флаг остановки
		s.kpvedWorkersStopMutex.RLock()
		stopped := s.kpvedWorkersStopped
		s.kpvedWorkersStopMutex.RUnlock()

		if stopped {
			log.Printf("[KPVED Worker %d] Stopped by user, skipping task %d: '%s'", workerID, task.Index, task.NormalizedName)
			s.kpvedCurrentTasksMutex.Lock()
			delete(s.kpvedCurrentTasks, workerID)
			s.kpvedCurrentTasksMutex.Unlock()
			resultChan <- classificationResult{
				task: task,
				err:  fmt.Errorf("worker stopped by user"),
			}
			continue
		}

		// Обновляем текущую задачу для этого воркера
		s.kpvedCurrentTasksMutex.Lock()
		s.kpvedCurrentTasks[workerID] = &task
		s.kpvedCurrentTasksMutex.Unlock()

		// Логируем каждую задачу для отслеживания параллелизма
		if task.Index%100 == 0 {
			log.Printf("[KPVED Worker %d] Processing task %d: '%s' in category '%s' (merged_count: %d)",
				workerID, task.Index, task.NormalizedName, task.Category, task.MergedCount)
		}

		// Классифицируем
		result, err := s.classifyWithRetry(workerID, task, hierarchicalClassifier)
		if err != nil {
			s.handleClassificationError(workerID, task, err, resultChan)
			continue
		}

		// Обновляем БД
		rowsAffected, err := s.updateNormalizedData(workerID, task, result)
		if err != nil {
			s.handleClassificationError(workerID, task, err, resultChan)
			continue
		}

		// Удаляем задачу из отслеживания после успешного завершения
		s.kpvedCurrentTasksMutex.Lock()
		delete(s.kpvedCurrentTasks, workerID)
		s.kpvedCurrentTasksMutex.Unlock()

		resultChan <- classificationResult{
			task:         task,
			result:       result,
			rowsAffected: rowsAffected,
		}
	}
}

// classifyWithRetry выполняет классификацию с обработкой circuit breaker
func (s *Server) classifyWithRetry(
	workerID int,
	task ClassificationTask,
	hierarchicalClassifier *normalization.HierarchicalClassifier,
) (*normalization.HierarchicalResult, error) {
	if task.Index%10 == 0 {
		log.Printf("[KPVED Worker %d] Starting classification for '%s' (category: '%s')", workerID, task.NormalizedName, task.Category)
	}

	result, err := hierarchicalClassifier.Classify(task.NormalizedName, task.Category)
	if err != nil {
		errStr := err.Error()
		isCircuitBreakerOpen := strings.Contains(errStr, "circuit breaker is open")

		if isCircuitBreakerOpen {
			log.Printf("[KPVED Worker %d] Circuit breaker is open (task: '%s'), waiting for recovery...", workerID, task.NormalizedName)
			recovered := hierarchicalClassifier.WaitForCircuitBreakerRecovery(30 * time.Second)
			if recovered {
				log.Printf("[KPVED Worker %d] Circuit breaker recovered, retrying classification for '%s'", workerID, task.NormalizedName)
				result, err = hierarchicalClassifier.Classify(task.NormalizedName, task.Category)
			} else {
				log.Printf("[KPVED Worker %d] Circuit breaker recovery timeout, skipping task '%s'", workerID, task.NormalizedName)
			}
		}
	}

	return result, err
}

// handleClassificationError обрабатывает ошибки классификации
func (s *Server) handleClassificationError(
	workerID int,
	task ClassificationTask,
	err error,
	resultChan chan<- classificationResult,
) {
	s.kpvedCurrentTasksMutex.Lock()
	delete(s.kpvedCurrentTasks, workerID)
	s.kpvedCurrentTasksMutex.Unlock()

	errStr := err.Error()
	isRateLimit := strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "exceeded the maximum number of parallel requests")

	if isRateLimit {
		log.Printf("[KPVED Worker %d] Rate limit detected, rate limiter will handle throttling automatically", workerID)
	}

	log.Printf("[KPVED Worker %d] ERROR classifying '%s' (category: '%s', merged_count: %d): %v",
		workerID, task.NormalizedName, task.Category, task.MergedCount, err)

	// Детальное логирование типа ошибки
	if isRateLimit {
		log.Printf("[KPVED Worker %d]   -> Rate limit error - Arliai API limit reached, paused and will retry", workerID)
	} else if strings.Contains(errStr, "ai call failed") {
		log.Printf("[KPVED Worker %d]   -> AI call failed - check API key, network connection, or rate limits", workerID)
	} else if strings.Contains(errStr, "no candidates found") {
		log.Printf("[KPVED Worker %d]   -> No candidates found in KPVED tree - check classifier data", workerID)
	} else if strings.Contains(errStr, "json unmarshal") {
		log.Printf("[KPVED Worker %d]   -> JSON parsing error - AI response format issue", workerID)
	} else if strings.Contains(errStr, "timeout") {
		log.Printf("[KPVED Worker %d]   -> Timeout error - AI service may be slow", workerID)
	}

	resultChan <- classificationResult{
		task: task,
		err:  err,
	}
}

// updateNormalizedData обновляет данные в normalized_data с retry логикой
func (s *Server) updateNormalizedData(
	workerID int,
	task ClassificationTask,
	result *normalization.HierarchicalResult,
) (int64, error) {
	updateQuery := `
		UPDATE normalized_data
		SET kpved_code = ?, kpved_name = ?, kpved_confidence = ?
		WHERE normalized_name = ? AND category = ?
	`
	updateResult, err := s.retryUpdate(updateQuery, result.FinalCode, result.FinalName, result.FinalConfidence, task.NormalizedName, task.Category)
	if err != nil {
		log.Printf("[KPVED Worker %d] Failed to update group '%s' (category: '%s') after retries: %v", workerID, task.NormalizedName, task.Category, err)
		return 0, err
	}

	rowsAffected, _ := updateResult.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("[KPVED Worker %d] WARNING: Update query affected 0 rows for group '%s' (category: '%s')", workerID, task.NormalizedName, task.Category)
	} else {
		log.Printf("[KPVED Worker %d] Updated %d rows for group '%s' (category: '%s') -> KPVED: %s (%s, confidence: %.2f)",
			workerID, rowsAffected, task.NormalizedName, task.Category, result.FinalCode, result.FinalName, result.FinalConfidence)
	}

	return rowsAffected, nil
}

// collectKpvedResults собирает и обрабатывает результаты классификации
func (s *Server) collectKpvedResults(results []classificationResult) map[string]interface{} {
	classified := 0
	failed := 0
	totalDuration := int64(0)
	totalSteps := 0
	totalAICalls := 0
	responseResults := []map[string]interface{}{}
	errorSamples := []string{}

	for _, res := range results {
		if res.err != nil {
			failed++
			errorMsg := res.err.Error()

			if len(errorSamples) < 10 {
				errorSamples = append(errorSamples, fmt.Sprintf("'%s' (category: '%s'): %s",
					res.task.NormalizedName, res.task.Category, errorMsg))
			}

			if failed <= 10 {
				log.Printf("[KPVED] Error sample %d: '%s' (category: '%s', merged_count: %d) -> %v",
					failed, res.task.NormalizedName, res.task.Category, res.task.MergedCount, res.err)
				errStr := errorMsg
				if strings.Contains(errStr, "ai call failed") {
					log.Printf("[KPVED]   -> AI call failed - check API key, network, or rate limits")
				} else if strings.Contains(errStr, "no candidates found") {
					log.Printf("[KPVED]   -> No candidates in KPVED tree - check classifier data")
				} else if strings.Contains(errStr, "json unmarshal") {
					log.Printf("[KPVED]   -> JSON parsing error - AI response format issue")
				} else if strings.Contains(errStr, "timeout") {
					log.Printf("[KPVED]   -> Timeout error - AI service may be slow or unavailable")
				}
			}

			responseResults = append(responseResults, map[string]interface{}{
				"normalized_name": res.task.NormalizedName,
				"category":        res.task.Category,
				"error":           errorMsg,
				"success":         false,
			})
		} else {
			classified++
			totalDuration += res.result.TotalDuration
			totalSteps += len(res.result.Steps)
			totalAICalls += res.result.AICallsCount

			responseResults = append(responseResults, map[string]interface{}{
				"normalized_name":  res.task.NormalizedName,
				"category":         res.task.Category,
				"kpved_code":       res.result.FinalCode,
				"kpved_name":       res.result.FinalName,
				"kpved_confidence": res.result.FinalConfidence,
				"steps":            len(res.result.Steps),
				"duration_ms":      res.result.TotalDuration,
				"ai_calls":         res.result.AICallsCount,
			})

			if classified%20 == 0 {
				avgDuration := totalDuration / int64(classified)
				log.Printf("[KPVED] Progress: %d/%d classified (avg: %dms, %d AI calls, %d failed)...",
					classified+failed, len(results), avgDuration, totalAICalls, failed)
			}
		}
	}

	avgDuration := int64(0)
	if classified > 0 {
		avgDuration = totalDuration / int64(classified)
	}

	avgSteps := 0.0
	if classified > 0 {
		avgSteps = float64(totalSteps) / float64(classified)
	}

	avgAICalls := 0.0
	if classified > 0 {
		avgAICalls = float64(totalAICalls) / float64(classified)
	}

	message := fmt.Sprintf("Классификация завершена: %d успешно, %d ошибок", classified, failed)
	if failed > 0 && classified == 0 {
		message = fmt.Sprintf("Все группы (%d) завершились с ошибкой", failed)
		if len(errorSamples) > 0 {
			sampleCount := min(len(errorSamples), 3)
			message = fmt.Sprintf("Все группы (%d) завершились с ошибкой. Примеры ошибок:\n%s\n\nПроверьте логи сервера для деталей. Возможные причины: неверный API ключ, отсутствие данных КПВЭД классификатора, проблемы с сетью.",
				failed, strings.Join(errorSamples[:sampleCount], "\n"))
		}
	}

	response := map[string]interface{}{
		"classified":     classified,
		"failed":         failed,
		"total_duration": totalDuration,
		"avg_duration":   avgDuration,
		"avg_steps":      avgSteps,
		"avg_ai_calls":   avgAICalls,
		"total_ai_calls": totalAICalls,
		"results":        responseResults,
		"message":        message,
	}

	if failed > 0 && classified == 0 && len(errorSamples) > 0 {
		sampleCount := min(len(errorSamples), 3)
		response["error_samples"] = errorSamples[:sampleCount]
	}

	return response
}

// buildKpvedEmptyResponse создает ответ для случая, когда нет групп без классификации
func (s *Server) buildKpvedEmptyResponse(totalGroups, groupsWithKpved int) map[string]interface{} {
	return map[string]interface{}{
		"classified":        0,
		"failed":            0,
		"total_duration":    0,
		"avg_duration":      0,
		"avg_steps":         0.0,
		"avg_ai_calls":      0.0,
		"total_ai_calls":    0,
		"results":           []map[string]interface{}{},
		"message":           fmt.Sprintf("Не найдено групп без классификации КПВЭД. Всего групп: %d, с классификацией: %d", totalGroups, groupsWithKpved),
		"total_groups":      totalGroups,
		"groups_with_kpved": groupsWithKpved,
	}
}

// handleKpvedReclassifyHierarchical переклассифицирует существующие группы с иерархическим подходом
// Рефакторенная версия с разбиением на меньшие функции
func (s *Server) handleKpvedReclassifyHierarchical(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Валидация запроса
	req, err := s.validateKpvedReclassifyRequest(r)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Получаем API ключ и модель
	apiKey, model, err := s.workerConfigManager.GetModelAndAPIKey()
	if err != nil {
		log.Printf("[KPVED] Error getting API key and model: %v", err)
		http.Error(w, fmt.Sprintf("AI API key not configured: %v", err), http.StatusServiceUnavailable)
		return
	}
	log.Printf("[KPVED] Using API key and model: %s", model)

	// Валидация состояния БД
	validationResult := s.validateKpvedDatabaseState()
	if !validationResult.IsValid {
		http.Error(w, validationResult.ErrorMessage, validationResult.HTTPStatus)
		return
	}

	// Проверяем на пустой результат
	if validationResult.TotalGroupsWithoutKpved == 0 {
		log.Printf("[KPVED] WARNING: No groups found without KPVED classification!")
		response := s.buildKpvedEmptyResponse(validationResult.TotalGroups, validationResult.GroupsWithKpved)
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
	}

	// Получаем задачи
	tasks, err := s.getKpvedClassificationTasks(req.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(tasks) == 0 {
		log.Printf("[KPVED] No tasks to process")
		response := map[string]interface{}{
			"classified":     0,
			"failed":         0,
			"total_duration": 0,
			"avg_duration":   0,
			"avg_steps":      0.0,
			"avg_ai_calls":   0.0,
			"total_ai_calls": 0,
			"results":        []map[string]interface{}{},
		}
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
	}

	// Создаем классификатор
	_, hierarchicalClassifier, err := s.createKpvedClassifier(apiKey, model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Настраиваем воркеры
	maxWorkers := s.setupKpvedWorkers()
	log.Printf("[KPVED] Starting classification with %d workers for %d groups (sorted by merged_count DESC)", maxWorkers, len(tasks))

	// Обрабатываем задачи
	results := s.processKpvedClassificationTasks(tasks, hierarchicalClassifier, maxWorkers)

	// Собираем результаты
	response := s.collectKpvedResults(results)

	s.writeJSONResponse(w, r, response, http.StatusOK)
}
