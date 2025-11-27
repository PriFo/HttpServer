package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"httpserver/internal/infrastructure/monitoring"
	monitoringinfra "httpserver/internal/infrastructure/monitoring"
	"httpserver/nomenclature"
	servermonitoring "httpserver/server/monitoring"
)

// AggregationStrategy стратегия агрегации результатов от нескольких провайдеров
type AggregationStrategy string

const (
	FirstSuccess      AggregationStrategy = "first_success"      // Первый успешный ответ
	MajorityVote      AggregationStrategy = "majority_vote"      // Голосование большинством
	AllResults        AggregationStrategy = "all_results"        // Все результаты
	HighestConfidence AggregationStrategy = "highest_confidence" // Наивысшая уверенность
)

// ProviderResult результат от одного провайдера
type ProviderResult struct {
	ProviderID   string
	ProviderName string
	Result       *nomenclature.AIProcessingResult
	Error        error
	Duration     time.Duration
	Success      bool
}

// AggregatedResult агрегированный результат от всех провайдеров
type AggregatedResult struct {
	FinalResult    *nomenclature.AIProcessingResult
	AllResults     []ProviderResult
	Strategy       AggregationStrategy
	TotalProviders int
	SuccessCount   int
	ErrorCount     int
	TotalDuration  time.Duration
}

// ArliaiProviderAdapter адаптер для ArliaiClient
type ArliaiProviderAdapter struct {
	client *nomenclature.AIClient
	name   string
}

func NewArliaiProviderAdapter(client *nomenclature.AIClient) *ArliaiProviderAdapter {
	return &ArliaiProviderAdapter{
		client: client,
		name:   "Arliai",
	}
}

func (a *ArliaiProviderAdapter) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	return a.client.GetCompletion(systemPrompt, userPrompt)
}

func (a *ArliaiProviderAdapter) GetProviderName() string {
	return a.name
}

func (a *ArliaiProviderAdapter) IsEnabled() bool {
	return a.client != nil
}

// OpenRouterProviderAdapter адаптер для OpenRouterClient
type OpenRouterProviderAdapter struct {
	client *OpenRouterClient
	name   string
}

func NewOpenRouterProviderAdapter(client *OpenRouterClient) *OpenRouterProviderAdapter {
	return &OpenRouterProviderAdapter{
		client: client,
		name:   "OpenRouter",
	}
}

func (o *OpenRouterProviderAdapter) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	// OpenRouter использует ChatCompletion, нужно преобразовать
	messages := []nomenclature.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Получаем модель из конфигурации с приоритетом: Config > Env > Default
	model := ""

	// Если модель не найдена в конфиге, используем переменную окружения
	if model == "" {
		model = os.Getenv("OPENROUTER_MODEL")
	}

	// Последний fallback на дефолтную модель
	if model == "" {
		model = "z.ai/glm-4.5" // z.ai/glm-4.5 как приоритетная модель по умолчанию
	}

	return o.client.ChatCompletion(model, messages)
}

func (o *OpenRouterProviderAdapter) GetProviderName() string {
	return o.name
}

func (o *OpenRouterProviderAdapter) IsEnabled() bool {
	return o.client != nil && o.client.apiKey != ""
}

// HuggingFaceProviderAdapter адаптер для HuggingFaceClient
type HuggingFaceProviderAdapter struct {
	client *HuggingFaceClient
	name   string
}

func NewHuggingFaceProviderAdapter(client *HuggingFaceClient) *HuggingFaceProviderAdapter {
	return &HuggingFaceProviderAdapter{
		client: client,
		name:   "HuggingFace",
	}
}

func (h *HuggingFaceProviderAdapter) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	messages := []nomenclature.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Получаем модель из WorkerConfigManager, если он доступен
	model := ""

	// Если модель не найдена в конфиге, используем переменную окружения или дефолтную
	if model == "" {
		model = os.Getenv("HUGGINGFACE_MODEL")
		if model == "" {
			// Используем одну из популярных моделей Hugging Face
			model = "mistralai/Mistral-7B-Instruct-v0.1"
		}
	}

	// Используем контекст с таймаутом для предотвращения зависания
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return h.client.ChatCompletionWithContext(ctx, model, messages)
}

func (h *HuggingFaceProviderAdapter) GetProviderName() string {
	return h.name
}

func (h *HuggingFaceProviderAdapter) IsEnabled() bool {
	return h.client != nil && h.client.apiKey != ""
}

// EdenAIProviderAdapter адаптер для EdenAIClient
type EdenAIProviderAdapter struct {
	client *EdenAIClient
	name   string
}

func NewEdenAIProviderAdapter(client *EdenAIClient) *EdenAIProviderAdapter {
	return &EdenAIProviderAdapter{
		client: client,
		name:   "Eden AI",
	}
}

func (e *EdenAIProviderAdapter) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	messages := []nomenclature.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Получаем модель из переменной окружения или используем дефолтную
	model := os.Getenv("EDENAI_MODEL")
	if model == "" {
		// Используем дефолтную модель Eden AI
		model = "openai/gpt-3.5-turbo"
	}

	// Используем контекст с таймаутом для предотвращения зависания
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return e.client.ChatCompletionWithContext(ctx, model, messages)
}

func (e *EdenAIProviderAdapter) GetProviderName() string {
	return e.name
}

func (e *EdenAIProviderAdapter) IsEnabled() bool {
	return e.client != nil && e.client.apiKey != ""
}

// ProviderWrapper обертка для разных типов клиентов
type ProviderWrapper struct {
	ID       string
	Name     string
	Client   ProviderClient
	Enabled  bool
	Priority int
}

// ProviderOrchestrator оркестратор для управления несколькими провайдерами AI.
// Координирует параллельные запросы к различным провайдерам и агрегирует результаты
// согласно выбранной стратегии (FirstSuccess, MajorityVote, AllResults, HighestConfidence).
//
// Пример использования:
//
//	orchestrator := NewProviderOrchestrator(30*time.Second, monitoringManager)
//	orchestrator.RegisterProvider("openrouter", "OpenRouter", openrouterAdapter, true, 1)
//	result, err := orchestrator.Normalize("system prompt", "user prompt")
type ProviderOrchestrator struct {
	monitoringManager *monitoringinfra.Manager           // Опциональный менеджер мониторинга
	metricsCollector  *servermonitoring.MetricsCollector // Сборщик метрик
	providers         map[string]*ProviderWrapper
	strategy          AggregationStrategy
	timeout           time.Duration
	logger            *slog.Logger // Структурированный логгер
	mu                sync.RWMutex
}

// NewProviderOrchestrator создает новый оркестратор провайдеров.
//
// Параметры:
//   - timeout: максимальное время ожидания ответа от провайдеров
//   - monitoringManager: опциональный менеджер мониторинга для отслеживания метрик
//
// Возвращает новый экземпляр ProviderOrchestrator с дефолтной стратегией FirstSuccess.
func NewProviderOrchestrator(timeout time.Duration, monitoringManager *monitoring.Manager) *ProviderOrchestrator {
	strategy := AggregationStrategy(os.Getenv("AGGREGATION_STRATEGY"))
	if strategy == "" {
		strategy = FirstSuccess // Стратегия по умолчанию
	}

	logger := slog.Default().With("component", "provider_orchestrator")

	return &ProviderOrchestrator{
		monitoringManager: monitoringManager,
		metricsCollector:  servermonitoring.NewMetricsCollector(),
		providers:         make(map[string]*ProviderWrapper),
		strategy:          strategy,
		timeout:           timeout,
		logger:            logger,
	}
}

// RegisterProvider регистрирует провайдер в оркестраторе
func (po *ProviderOrchestrator) RegisterProvider(id, name string, client ProviderClient, enabled bool, priority int) {
	po.mu.Lock()
	defer po.mu.Unlock()

	po.providers[id] = &ProviderWrapper{
		ID:       id,
		Name:     name,
		Client:   client,
		Enabled:  enabled,
		Priority: priority,
	}
}

// GetActiveProviders возвращает список активных провайдеров, отсортированных по приоритету
func (po *ProviderOrchestrator) GetActiveProviders() []*ProviderWrapper {
	po.mu.RLock()
	defer po.mu.RUnlock()

	active := make([]*ProviderWrapper, 0)
	for _, provider := range po.providers {
		if provider.Enabled && provider.Client != nil && provider.Client.IsEnabled() {
			active = append(active, provider)
		}
	}

	// Сортируем по приоритету (меньше = выше приоритет)
	sort.Slice(active, func(i, j int) bool {
		return active[i].Priority < active[j].Priority
	})

	return active
}

// SetStrategy устанавливает стратегию агрегации
func (po *ProviderOrchestrator) SetStrategy(strategy AggregationStrategy) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.strategy = strategy
}

// GetStrategy возвращает текущую стратегию агрегации
func (po *ProviderOrchestrator) GetStrategy() AggregationStrategy {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.strategy
}

// GetTimeout возвращает таймаут для запросов
func (po *ProviderOrchestrator) GetTimeout() time.Duration {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.timeout
}

// Normalize выполняет нормализацию с использованием всех активных провайдеров.
// Запросы к провайдерам выполняются параллельно, результаты агрегируются
// согласно текущей стратегии (FirstSuccess, MajorityVote, AllResults, HighestConfidence).
//
// Параметры:
//   - systemPrompt: системный промпт для AI-модели
//   - userPrompt: пользовательский промпт с данными для нормализации
//
// Возвращает агрегированный результат или ошибку, если все провайдеры недоступны.
func (po *ProviderOrchestrator) Normalize(systemPrompt, userPrompt string) (*AggregatedResult, error) {
	activeProviders := po.GetActiveProviders()
	if len(activeProviders) == 0 {
		return nil, fmt.Errorf("no active providers available")
	}

	// Генерируем request_id для трассировки
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	logger := po.logger.With("request_id", requestID)

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), po.timeout)
	defer cancel()

	logger.Info("Starting normalization", "providers_count", len(activeProviders), "strategy", string(po.strategy))

	// Канал для сбора результатов
	resultsChan := make(chan ProviderResult, len(activeProviders))
	var wg sync.WaitGroup

	// Запускаем запросы ко всем провайдерам параллельно
	for _, provider := range activeProviders {
		wg.Add(1)
		go func(p *ProviderWrapper) {
			defer wg.Done()

			result := ProviderResult{
				ProviderID:   p.ID,
				ProviderName: p.Name,
				Success:      false,
			}

			reqStart := time.Now()

			// Записываем начало запроса в мониторинг
			if po.monitoringManager != nil {
				po.monitoringManager.IncrementRequest(p.ID)
			}

			// Выполняем запрос с контекстом для возможности отмены
			response, err := po.executeWithContext(ctx, p.Client, systemPrompt, userPrompt)
			result.Duration = time.Since(reqStart)

			// Записываем завершение запроса в мониторинг
			if po.monitoringManager != nil {
				latencyMs := float64(result.Duration.Milliseconds())
				po.monitoringManager.RecordResponse(p.ID, latencyMs, err)
			}
			if err != nil {
				errorType := "unknown"
				errorMsg := err.Error()

				// Определяем тип ошибки для метрик
				if err == context.DeadlineExceeded {
					errorType = "timeout"
				} else if strings.Contains(strings.ToLower(errorMsg), "quota") ||
					strings.Contains(strings.ToLower(errorMsg), "quota exceeded") {
					errorType = "quota_exceeded"
				} else if strings.Contains(strings.ToLower(errorMsg), "rate limit") ||
					strings.Contains(strings.ToLower(errorMsg), "429") ||
					strings.Contains(strings.ToLower(errorMsg), "too many requests") {
					errorType = "rate_limit"
				} else if strings.Contains(strings.ToLower(errorMsg), "timeout") ||
					strings.Contains(strings.ToLower(errorMsg), "deadline exceeded") {
					errorType = "timeout"
				} else if strings.Contains(strings.ToLower(errorMsg), "network") ||
					strings.Contains(strings.ToLower(errorMsg), "connection") {
					errorType = "network"
				}

				// Детальное логирование ошибок
				if errorType == "quota_exceeded" {
					logger.Warn("Provider quota exceeded - will not retry, fallback to other providers",
						"provider", p.Name,
						"provider_id", p.ID,
						"error", errorMsg,
						"duration_ms", result.Duration.Milliseconds(),
						"error_type", errorType)
				} else if errorType == "rate_limit" {
					logger.Warn("Provider rate limit exceeded - may retry with other providers",
						"provider", p.Name,
						"provider_id", p.ID,
						"error", errorMsg,
						"duration_ms", result.Duration.Milliseconds(),
						"error_type", errorType)
				} else {
					logger.Warn("Provider request failed",
						"provider", p.Name,
						"provider_id", p.ID,
						"error", errorMsg,
						"duration_ms", result.Duration.Milliseconds(),
						"error_type", errorType)
				}

				result.Error = err
				resultsChan <- result
				return
			}

			// Парсим результат
			aiResult, err := po.parseAIResponse(response)
			if err != nil {
				result.Error = fmt.Errorf("failed to parse response from %s: %v", p.Name, err)
				resultsChan <- result
				return
			}

			result.Result = aiResult
			result.Success = true
			logger.Info("Provider request succeeded",
				"provider", p.Name,
				"provider_id", p.ID,
				"duration_ms", result.Duration.Milliseconds(),
				"confidence", aiResult.Confidence)
			resultsChan <- result
		}(provider)
	}

	// Ждем завершения всех запросов или таймаута
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Собираем результаты
	var allResults []ProviderResult
	select {
	case <-done:
		// Все запросы завершены
		close(resultsChan)
		for result := range resultsChan {
			allResults = append(allResults, result)
		}
	case <-ctx.Done():
		// Таймаут - собираем то, что успело прийти
		close(resultsChan)
		for result := range resultsChan {
			allResults = append(allResults, result)
		}
	}

	// Агрегируем результаты согласно стратегии
	aggregated := po.aggregateResults(allResults, activeProviders)
	aggregated.TotalDuration = time.Since(startTime)

	// Записываем метрики нормализации

	if aggregated.FinalResult != nil {
		logger.Info("Normalization completed successfully",
			"total_duration_ms", aggregated.TotalDuration.Milliseconds(),
			"success_count", aggregated.SuccessCount,
			"error_count", aggregated.ErrorCount,
			"final_confidence", aggregated.FinalResult.Confidence)
	} else {
		logger.Warn("Normalization completed with no result",
			"total_duration_ms", aggregated.TotalDuration.Milliseconds(),
			"success_count", aggregated.SuccessCount,
			"error_count", aggregated.ErrorCount)
	}

	return aggregated, nil
}

// GetMetrics возвращает метрики оркестратора
func (po *ProviderOrchestrator) GetMetrics() map[string]interface{} {
	return map[string]interface{}{}
}

// executeWithContext выполняет запрос с поддержкой контекста
func (po *ProviderOrchestrator) executeWithContext(ctx context.Context, client ProviderClient, systemPrompt, userPrompt string) (string, error) {
	// Создаем канал для результата
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		response, err := client.GetCompletion(systemPrompt, userPrompt)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- response
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errChan:
		return "", err
	case result := <-resultChan:
		return result, nil
	}
}

// parseAIResponse парсит JSON ответ от AI для нормализации
func (po *ProviderOrchestrator) parseAIResponse(response string) (*nomenclature.AIProcessingResult, error) {
	// Очищаем от возможных markdown обрамлений
	cleaned := strings.TrimSpace(response)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)

	// Парсим ответ нормализации (формат отличается от KPVED)
	var normalizationResult struct {
		NormalizedName string  `json:"normalized_name"`
		Category       string  `json:"category"`
		Confidence     float64 `json:"confidence"`
		Reasoning      string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(cleaned), &normalizationResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Валидация
	if normalizationResult.NormalizedName == "" {
		return nil, fmt.Errorf("empty normalized_name in response")
	}

	// Преобразуем в AIProcessingResult
	result := &nomenclature.AIProcessingResult{
		NormalizedName: normalizationResult.NormalizedName,
		KpvedCode:      "",                           // Для нормализации КПВЭД код не используется
		KpvedName:      normalizationResult.Category, // Используем category как KpvedName
		Confidence:     normalizationResult.Confidence,
		Reasoning:      normalizationResult.Reasoning,
	}

	return result, nil
}

// aggregateResults агрегирует результаты согласно выбранной стратегии
func (po *ProviderOrchestrator) aggregateResults(results []ProviderResult, providers []*ProviderWrapper) *AggregatedResult {
	aggregated := &AggregatedResult{
		AllResults:     results,
		Strategy:       po.strategy,
		TotalProviders: len(providers),
		FinalResult:    nil,
	}

	// Подсчитываем успешные и неуспешные запросы
	for _, result := range results {
		if result.Success {
			aggregated.SuccessCount++
		} else {
			aggregated.ErrorCount++
		}
	}

	// Применяем стратегию агрегации
	switch po.strategy {
	case FirstSuccess:
		aggregated.FinalResult = po.firstSuccessStrategy(results)
	case MajorityVote:
		aggregated.FinalResult = po.majorityVoteStrategy(results)
	case HighestConfidence:
		aggregated.FinalResult = po.highestConfidenceStrategy(results)
	case AllResults:
		// Для all_results возвращаем первый успешный, но сохраняем все
		aggregated.FinalResult = po.firstSuccessStrategy(results)
	default:
		// Fallback на first_success
		aggregated.FinalResult = po.firstSuccessStrategy(results)
	}

	return aggregated
}

// firstSuccessStrategy возвращает первый успешный результат
func (po *ProviderOrchestrator) firstSuccessStrategy(results []ProviderResult) *nomenclature.AIProcessingResult {
	for _, result := range results {
		if result.Success && result.Result != nil {
			po.logger.Info("Using first success result",
				"provider", result.ProviderName,
				"provider_id", result.ProviderID,
				"duration_ms", result.Duration.Milliseconds())
			return result.Result
		}
	}
	return nil
}

// majorityVoteStrategy выбирает результат, который вернуло большинство провайдеров
func (po *ProviderOrchestrator) majorityVoteStrategy(results []ProviderResult) *nomenclature.AIProcessingResult {
	// Группируем результаты по нормализованному имени и категории
	votes := make(map[string]int)
	resultMap := make(map[string]*nomenclature.AIProcessingResult)

	for _, result := range results {
		if !result.Success || result.Result == nil {
			continue
		}

		// Создаем ключ из нормализованного имени и категории
		key := fmt.Sprintf("%s|%s", result.Result.NormalizedName, result.Result.KpvedCode)

		votes[key]++
		if votes[key] == 1 {
			resultMap[key] = result.Result
		}
	}

	if len(votes) == 0 {
		return nil
	}

	// Находим ключ с максимальным количеством голосов
	maxVotes := 0
	var bestKey string
	for key, count := range votes {
		if count > maxVotes {
			maxVotes = count
			bestKey = key
		}
	}

	// Проверяем, есть ли большинство (больше половины успешных результатов)
	totalSuccess := 0
	for _, result := range results {
		if result.Success {
			totalSuccess++
		}
	}
	if totalSuccess == 0 {
		return nil
	}

	if maxVotes > totalSuccess/2 {
		po.logger.Info("Majority vote result selected",
			"votes", maxVotes,
			"total_success", totalSuccess,
			"result_key", bestKey)
		return resultMap[bestKey]
	}

	// Если нет большинства, возвращаем результат с наибольшим количеством голосов
	po.logger.Info("No majority, using result with most votes",
		"votes", maxVotes,
		"total_success", totalSuccess,
		"result_key", bestKey)
	return resultMap[bestKey]
}

// highestConfidenceStrategy выбирает результат с наивысшей уверенностью
func (po *ProviderOrchestrator) highestConfidenceStrategy(results []ProviderResult) *nomenclature.AIProcessingResult {
	var bestResult *nomenclature.AIProcessingResult
	maxConfidence := 0.0

	for _, result := range results {
		if !result.Success || result.Result == nil {
			continue
		}

		if result.Result.Confidence > maxConfidence {
			maxConfidence = result.Result.Confidence
			bestResult = result.Result
		}
	}

	if bestResult != nil {
		po.logger.Info("Using highest confidence result",
			"confidence", maxConfidence)
	}

	return bestResult
}
