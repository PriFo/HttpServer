package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/internal/infrastructure/ai"
)

// MultiProviderClient клиент для работы с несколькими AI провайдерами параллельно.
// Поддерживает настраиваемое количество каналов (параллельных запросов) для каждого провайдера.
// Реализует интерфейс AINameNormalizer для использования в нормализации контрагентов.
//
// Особенности:
//   - Параллельные запросы к нескольким провайдерам через каналы
//   - Агрегация результатов методом majority vote
//   - Fallback на генеративные AI при недоступности специализированных провайдеров
//   - Поддержка контекста для отмены операций
//
// Пример использования:
//
//	providers := []*database.Provider{
//	    {ID: "openrouter", Name: "OpenRouter", Enabled: true, Channels: 2, Priority: 1},
//	}
//	clients := map[string]ProviderClient{"openrouter": openrouterAdapter}
//	mpc := NewMultiProviderClient(providers, clients, router)
//	result, err := mpc.NormalizeName(ctx, "ООО Ромашка")
type MultiProviderClient struct {
	providers          map[string]*MultiProviderConfig // Ключ: ID провайдера (openrouter, huggingface, arliai)
	counterpartyRouter *CounterpartyProviderRouter     // Роутер для стандартизации контрагентов (DaData/Adata)
	timeout            time.Duration
	logger             *slog.Logger // Структурированный логгер
	mu                 sync.RWMutex
}

// MultiProviderConfig конфигурация провайдера с количеством каналов для MultiProviderClient
type MultiProviderConfig struct {
	ID       string
	Name     string
	Client   ai.ProviderClient // Адаптер провайдера
	Channels int               // Количество параллельных каналов
	Enabled  bool
	Priority int
}

// MultiProviderResult результат от одного канала провайдера
type MultiProviderResult struct {
	ProviderID   string
	ProviderName string
	ChannelID    int // Номер канала (для отладки)
	Result       string
	Error        error
	Duration     time.Duration
	Success      bool
}

// MultiProviderAggregatedResult агрегированный результат от всех провайдеров
type MultiProviderAggregatedResult struct {
	FinalResult   string
	AllResults    []MultiProviderResult
	TotalChannels int
	SuccessCount  int
	ErrorCount    int
	TotalDuration time.Duration
	Strategy      string
}

// NewMultiProviderClient создает новый мульти-провайдерный клиент.
//
// Параметры:
//   - providersFromDB: список провайдеров из базы данных с их конфигурацией
//   - clients: карта клиентов провайдеров (ключ - ID провайдера)
//   - counterpartyRouter: роутер для маршрутизации к специализированным провайдерам (DaData/Adata)
//
// Возвращает новый экземпляр MultiProviderClient с настроенными провайдерами.
func NewMultiProviderClient(providersFromDB []*database.Provider, clients map[string]ai.ProviderClient, counterpartyRouter *CounterpartyProviderRouter) *MultiProviderClient {
	logger := slog.Default().With("component", "multi_provider_client")
	mpc := &MultiProviderClient{
		providers:          make(map[string]*MultiProviderConfig),
		counterpartyRouter: counterpartyRouter,
		timeout:            30 * time.Second,
		logger:             logger,
	}

	for _, p := range providersFromDB {
		if !p.IsActive {
			continue
		}

		// Используем Type как идентификатор провайдера (для обратной совместимости)
		providerID := p.Type
		if providerID == "" {
			// Если type не установлен, используем name в нижнем регистре
			providerID = strings.ToLower(strings.ReplaceAll(p.Name, " ", "_"))
		}

		client, ok := clients[providerID]
		if !ok || client == nil {
			mpc.logger.Warn("Provider has no client, skipping",
				"provider", p.Name,
				"provider_type", providerID)
			continue
		}

		// Убеждаемся, что клиент активен
		if !client.IsEnabled() {
			mpc.logger.Warn("Provider is not enabled, skipping",
				"provider", p.Name,
				"provider_type", providerID)
			continue
		}

		// Извлекаем количество каналов из config JSON (по умолчанию 1)
		channels := 1
		if p.Config != "" {
			// Парсим JSON config для получения channels
			var configMap map[string]interface{}
			if err := json.Unmarshal([]byte(p.Config), &configMap); err == nil {
				if ch, ok := configMap["channels"].(float64); ok {
					channels = int(ch)
				}
			}
		}
		if channels < 1 {
			channels = 1
		}

		mpc.providers[providerID] = &MultiProviderConfig{
			ID:       providerID,
			Name:     p.Name,
			Client:   client,
			Channels: channels,
			Enabled:  p.IsActive,
			Priority: 0, // Priority больше не хранится в новой структуре
		}

		mpc.logger.Info("Provider registered",
			"provider", p.Name,
			"provider_id", p.ID,
			"channels", channels)
	}

	return mpc
}

// NormalizeName нормализует название контрагента, используя все активные провайдеры параллельно
// Каждый провайдер отправляет запросы через указанное количество каналов
// Результаты агрегируются методом majority vote
func (mpc *MultiProviderClient) NormalizeName(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	// Генерируем request_id для трассировки
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	logger := mpc.logger.With("request_id", requestID, "name", name)

	mpc.mu.RLock()
	activeProviders := make([]*MultiProviderConfig, 0, len(mpc.providers))
	for _, p := range mpc.providers {
		if p.Enabled && p.Client != nil && p.Client.IsEnabled() {
			activeProviders = append(activeProviders, p)
		}
	}
	mpc.mu.RUnlock()

	if len(activeProviders) == 0 {
		logger.Warn("No active providers available")
		return "", fmt.Errorf("no active providers available")
	}

	logger.Info("Starting name normalization",
		"providers_count", len(activeProviders))

	startTime := time.Now()

	// Формируем промпт для нормализации
	systemPrompt := "Ты эксперт по нормализации названий компаний. Нормализуй название, приведя его к каноничному виду с правильными регистрами."
	userPrompt := fmt.Sprintf("Нормализуй название компании: \"%s\". Верни только каноничное название без объяснений и дополнительного текста.", name)

	// Каналы для сбора результатов
	resultChan := make(chan MultiProviderResult, 100) // Буферизованный канал
	var wg sync.WaitGroup

	// Запускаем горутины для каждого провайдера и каждого его канала
	totalChannels := 0
	for _, provider := range activeProviders {
		for channelID := 0; channelID < provider.Channels; channelID++ {
			totalChannels++
			wg.Add(1)
			go func(p *MultiProviderConfig, chID int) {
				defer wg.Done()

				result := MultiProviderResult{
					ProviderID:   p.ID,
					ProviderName: p.Name,
					ChannelID:    chID,
					Success:      false,
				}

				// Устанавливаем таймаут для каждого запроса
				reqCtx, cancel := context.WithTimeout(ctx, mpc.timeout)
				defer cancel()

				reqStart := time.Now()

				// Выполняем запрос к провайдеру
				response, err := mpc.executeProviderRequest(reqCtx, p.Client, systemPrompt, userPrompt)
				result.Duration = time.Since(reqStart)

				if err != nil {
					result.Error = fmt.Errorf("provider %s channel %d: %w", p.Name, chID, err)
					logger.Warn("Provider request failed",
						"provider", p.Name,
						"provider_id", p.ID,
						"channel", chID,
						"error", err.Error(),
						"duration_ms", result.Duration.Milliseconds())
				} else {
					result.Result = strings.TrimSpace(response)
					result.Success = true
					logger.Info("Provider request succeeded",
						"provider", p.Name,
						"provider_id", p.ID,
						"channel", chID,
						"result", result.Result,
						"duration_ms", result.Duration.Milliseconds())
				}

				resultChan <- result
			}(provider, channelID)
		}
	}

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Собираем результаты
	var allResults []MultiProviderResult
	for result := range resultChan {
		allResults = append(allResults, result)
	}

	if len(allResults) == 0 {
		return "", fmt.Errorf("no responses received from any provider")
	}

	// Агрегируем результаты методом majority vote
	aggregated := mpc.aggregateResults(allResults, totalChannels, time.Since(startTime))

	if aggregated.FinalResult == "" {
		// Собираем информацию об ошибках для детального сообщения
		errorDetails := make([]string, 0)
		for _, result := range allResults {
			if !result.Success && result.Error != nil {
				errorDetails = append(errorDetails, fmt.Sprintf("%s: %v", result.ProviderName, result.Error))
			}
		}
		if len(errorDetails) > 0 {
			return "", fmt.Errorf("failed to aggregate results: no successful responses. Errors: %v", errorDetails)
		}
		return "", fmt.Errorf("failed to aggregate results: no successful responses")
	}

	logger.Info("Name normalization completed",
		"success_count", aggregated.SuccessCount,
		"total_channels", totalChannels,
		"error_count", aggregated.ErrorCount,
		"result", aggregated.FinalResult,
		"total_duration_ms", aggregated.TotalDuration.Milliseconds())

	return aggregated.FinalResult, nil
}

// executeProviderRequest выполняет запрос к провайдеру с поддержкой контекста
func (mpc *MultiProviderClient) executeProviderRequest(ctx context.Context, client ai.ProviderClient, systemPrompt, userPrompt string) (string, error) {
	// Создаем каналы для результата
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

// aggregateResults агрегирует результаты методом majority vote
func (mpc *MultiProviderClient) aggregateResults(results []MultiProviderResult, totalChannels int, totalDuration time.Duration) *MultiProviderAggregatedResult {
	aggregated := &MultiProviderAggregatedResult{
		AllResults:    results,
		TotalChannels: totalChannels,
		Strategy:      "majority_vote",
		TotalDuration: totalDuration,
	}

	// Подсчитываем успешные и неуспешные запросы
	for _, result := range results {
		if result.Success {
			aggregated.SuccessCount++
		} else {
			aggregated.ErrorCount++
		}
	}

	if aggregated.SuccessCount == 0 {
		return aggregated
	}

	// Группируем результаты по значению (majority vote)
	votes := make(map[string]int)
	resultProviders := make(map[string][]string) // Для отслеживания, какие провайдеры дали какой результат
	for _, result := range results {
		if result.Success && result.Result != "" {
			// Нормализуем результат для сравнения (приводим к нижнему регистру и убираем пробелы)
			normalized := strings.ToLower(strings.TrimSpace(result.Result))
			votes[normalized]++
			if resultProviders[normalized] == nil {
				resultProviders[normalized] = make([]string, 0)
			}
			resultProviders[normalized] = append(resultProviders[normalized], result.ProviderName)
		}
	}

	if len(votes) == 0 {
		return aggregated
	}

	// Находим результат с максимальным количеством голосов
	maxVotes := 0
	var bestResult string
	for result, count := range votes {
		if count > maxVotes {
			maxVotes = count
			bestResult = result
		}
	}

	// Проверяем, есть ли большинство (больше половины успешных результатов)
	if maxVotes > aggregated.SuccessCount/2 {
		// Восстанавливаем оригинальный регистр из первого успешного результата
		for _, result := range results {
			if result.Success && strings.ToLower(strings.TrimSpace(result.Result)) == bestResult {
				aggregated.FinalResult = strings.TrimSpace(result.Result)
				// Логирование будет добавлено в вызывающем методе
				return aggregated
			}
		}
	}

	// Если нет большинства, используем результат с наибольшим количеством голосов
	for _, result := range results {
		if result.Success && strings.ToLower(strings.TrimSpace(result.Result)) == bestResult {
			aggregated.FinalResult = strings.TrimSpace(result.Result)
			// Логирование будет добавлено в вызывающем методе
			return aggregated
		}
	}

	return aggregated
}

// GetActiveProvidersCount возвращает количество активных провайдеров
func (mpc *MultiProviderClient) GetActiveProvidersCount() int {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()
	return len(mpc.providers)
}

// GetTotalChannels возвращает общее количество каналов всех активных провайдеров
func (mpc *MultiProviderClient) GetTotalChannels() int {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	total := 0
	for _, p := range mpc.providers {
		if p.Enabled {
			total += p.Channels
		}
	}
	return total
}

// NormalizeCounterparty нормализует контрагента, используя специализированные провайдеры (DaData/Adata)
// Если специализированные провайдеры недоступны или не могут определить страну, используется fallback на генеративные AI
func (mpc *MultiProviderClient) NormalizeCounterparty(ctx context.Context, name, inn, bin string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	// Генерируем request_id для трассировки
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	logger := mpc.logger.With("request_id", requestID, "name", name, "inn", inn, "bin", bin)

	// Сначала пытаемся использовать специализированные провайдеры через роутер
	if mpc.counterpartyRouter != nil {
		result, err := mpc.counterpartyRouter.StandardizeCounterparty(name, inn, bin)
		if err == nil {
			logger.Info("Counterparty normalized via specialized provider", "result", result)
			return result, nil
		}
		logger.Info("Router failed, falling back to generative AI",
			"error", err.Error())
	}

	// Fallback: используем генеративные AI провайдеры
	return mpc.NormalizeName(ctx, name)
}
