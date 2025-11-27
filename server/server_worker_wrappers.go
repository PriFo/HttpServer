package server

import (
	"context"
	"fmt"
	"os"
	"time"

	"httpserver/internal/infrastructure/ai"
	inframonitoring "httpserver/internal/infrastructure/monitoring"
)

// checkArliaiConnectionWrapper обертка для checkArliaiConnection
func (s *Server) checkArliaiConnectionWrapper(ctx context.Context, traceID string) (interface{}, error) {
	startTime := time.Now()

	// Проверяем кеш
	if cached, ok := s.arliaiCache.GetStatus(); ok {
		cacheAge := s.arliaiCache.GetStatusAge()
		return map[string]interface{}{
			"success":   true,
			"data":      cached,
			"timestamp": time.Now(),
			"duration":  time.Since(startTime),
			"metadata": map[string]interface{}{
				"cached":      true,
				"cache_age_s": cacheAge.Seconds(),
			},
		}, nil
	}

	// Проверяем WorkerConfigManager для локальной информации
	var localStatus map[string]interface{}
	if s.workerConfigManager != nil {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err == nil && provider != nil && provider.Name == "arliai" {
			apiKey := provider.APIKey
			if apiKey == "" {
				apiKey = os.Getenv("ARLIAI_API_KEY")
			}

			model, modelErr := s.workerConfigManager.GetActiveModel(provider.Name)
			modelName := ""
			if modelErr == nil && model != nil {
				modelName = model.Name
			}

			localStatus = map[string]interface{}{
				"provider":    provider.Name,
				"has_api_key": apiKey != "",
				"model":       modelName,
				"enabled":     provider.Enabled,
			}
		}
	}

	// Пытаемся проверить подключение через Arliai API
	statusResp, err := s.arliaiClient.CheckConnection(ctx, traceID)
	if err != nil {
		// Если API недоступен, возвращаем локальный статус
		connected := false
		if localStatus != nil {
			if hasKey, ok := localStatus["has_api_key"].(bool); ok {
				if enabled, ok2 := localStatus["enabled"].(bool); ok2 {
					connected = hasKey && enabled
				}
			}
		}

		responseData := map[string]interface{}{
			"connected":     connected,
			"api_available": false,
			"last_check":    time.Now(),
		}
		if localStatus != nil {
			responseData["provider"] = localStatus["provider"]
			responseData["has_api_key"] = localStatus["has_api_key"]
			responseData["model"] = localStatus["model"]
			responseData["enabled"] = localStatus["enabled"]
		}

		response := map[string]interface{}{
			"success":   true,
			"data":      responseData,
			"timestamp": time.Now(),
			"duration":  time.Since(startTime),
			"metadata": map[string]interface{}{
				"cached":    false,
				"api_error": err.Error(),
			},
		}

		// Кешируем результат даже при ошибке
		s.arliaiCache.SetStatus(responseData)

		return response, nil
	}

	// Успешная проверка через API
	connected := statusResp.Status == "ok" || statusResp.Status == "healthy"

	responseData := map[string]interface{}{
		"connected":        connected,
		"status":           statusResp.Status,
		"model":            statusResp.Model,
		"version":          statusResp.Version,
		"api_available":    true,
		"last_check":       statusResp.Timestamp,
		"response_time_ms": time.Since(startTime).Milliseconds(),
	}

	// Объединяем с локальной информацией
	if localStatus != nil {
		responseData["provider"] = localStatus["provider"]
		responseData["enabled"] = localStatus["enabled"]
		if responseData["model"] == "" {
			responseData["model"] = localStatus["model"]
		}
	}

	response := map[string]interface{}{
		"success":   true,
		"data":      responseData,
		"timestamp": time.Now(),
		"duration":  time.Since(startTime),
		"metadata": map[string]interface{}{
			"cached": false,
		},
	}

	// Кешируем успешный результат
	s.arliaiCache.SetStatus(responseData)

	return response, nil
}

// checkOpenRouterConnectionWrapper обертка для checkOpenRouterConnection
func (s *Server) checkOpenRouterConnectionWrapper(ctx context.Context, traceID string, apiKey string) (interface{}, error) {
	startTime := time.Now()

	// Используем GetModels для проверки подключения
	// Создаем новый клиент для проверки
	openrouterClient := NewOpenRouterClient(apiKey)

	models, err := openrouterClient.GetModels(ctx, traceID)
	if err != nil {
		connected := apiKey != ""
		responseData := map[string]interface{}{
			"connected":     connected,
			"api_available": false,
			"last_check":    time.Now(),
		}

		return map[string]interface{}{
			"success":   true,
			"data":      responseData,
			"timestamp": time.Now(),
			"duration":  time.Since(startTime),
			"metadata": map[string]interface{}{
				"cached":    false,
				"api_error": err.Error(),
			},
		}, nil
	}

	// Успешная проверка через API
	connected := len(models) > 0

	responseData := map[string]interface{}{
		"connected":        connected,
		"status":           "ok",
		"api_available":    true,
		"last_check":       time.Now(),
		"models_count":     len(models),
		"response_time_ms": time.Since(startTime).Milliseconds(),
	}

	return map[string]interface{}{
		"success":   true,
		"data":      responseData,
		"timestamp": time.Now(),
		"duration":  time.Since(startTime),
		"metadata": map[string]interface{}{
			"cached": false,
		},
	}, nil
}

// checkHuggingFaceConnectionWrapper обертка для checkHuggingFaceConnection
func (s *Server) checkHuggingFaceConnectionWrapper(ctx context.Context, traceID string, apiKey string, baseURL string) (interface{}, error) {
	startTime := time.Now()

	// Создаем новый клиент для проверки
	huggingfaceClient := NewHuggingFaceClient(apiKey, baseURL)

	statusResp, err := huggingfaceClient.CheckConnection(ctx, traceID)
	if err != nil {
		connected := apiKey != ""
		responseData := map[string]interface{}{
			"connected":     connected,
			"api_available": false,
			"last_check":    time.Now(),
		}

		return map[string]interface{}{
			"success":   true,
			"data":      responseData,
			"timestamp": time.Now(),
			"duration":  time.Since(startTime),
			"metadata": map[string]interface{}{
				"cached":    false,
				"api_error": err.Error(),
			},
		}, nil
	}

	// Успешная проверка через API
	connected := false
	if statusResp != nil {
		if c, ok := statusResp["connected"].(bool); ok {
			connected = c
		}
	}

	responseData := map[string]interface{}{
		"connected":        connected,
		"status":           statusResp["status"],
		"api_available":    true,
		"last_check":       statusResp["timestamp"],
		"models_count":     statusResp["models_count"],
		"response_time_ms": time.Since(startTime).Milliseconds(),
	}

	return map[string]interface{}{
		"success":   true,
		"data":      responseData,
		"timestamp": time.Now(),
		"duration":  time.Since(startTime),
		"metadata": map[string]interface{}{
			"cached": false,
		},
	}, nil
}

// getModelsWrapper обертка для getModels
func (s *Server) getModelsWrapper(ctx context.Context, traceID string, providerFilter string, filterStatus string, filterEnabled string, searchQuery string) (interface{}, error) {
	// Проверяем, есть ли параметр refresh в контексте (передается через query параметры)
	refresh := false
	if ctx.Value("refresh") != nil {
		if refreshVal, ok := ctx.Value("refresh").(bool); ok {
			refresh = refreshVal
		}
	}
	
	result, _, apiErr := s.buildWorkerModelsData(ctx, traceID, workerModelsOptions{
		ProviderFilter: providerFilter,
		FilterStatus:   filterStatus,
		FilterEnabled:  filterEnabled,
		SearchQuery:    searchQuery,
		Refresh:        refresh,
	})
	if apiErr != nil {
		return nil, fmt.Errorf("%s", apiErr.Message)
	}

	return map[string]interface{}{
		"success":   true,
		"data":      result,
		"timestamp": time.Now(),
	}, nil
}

func (s *Server) updateHuggingFaceClientWrapper(apiKey string, baseURL string) error {
	if apiKey == "" {
		apiKey = os.Getenv("HUGGINGFACE_API_KEY")
	}
	if baseURL == "" {
		baseURL = "https://api-inference.huggingface.co"
	}
	s.huggingfaceClient = ai.NewHuggingFaceClient(apiKey, baseURL)
	if s.providerOrchestrator != nil {
		adapter := ai.NewHuggingFaceProviderAdapter(s.huggingfaceClient)
		s.providerOrchestrator.RegisterProvider("huggingface", "Hugging Face", adapter, apiKey != "", 3)
	}
	return nil
}

func (s *Server) updateProviderOrchestratorWrapper(providerName string, adapter interface{}, enabled bool, priority int) error {
	if s.providerOrchestrator == nil {
		return fmt.Errorf("provider orchestrator not initialized")
	}
	if priority <= 0 {
		priority = 3
	}

	switch providerName {
	case "huggingface":
		if s.huggingfaceClient == nil {
			return fmt.Errorf("huggingface client not initialized")
		}
		hfAdapter := ai.NewHuggingFaceProviderAdapter(s.huggingfaceClient)
		s.providerOrchestrator.RegisterProvider("huggingface", "Hugging Face", hfAdapter, enabled, priority)
		return nil
	case "openrouter":
		if s.openrouterClient == nil {
			return fmt.Errorf("openrouter client not initialized")
		}
		orAdapter := ai.NewOpenRouterProviderAdapter(s.openrouterClient)
		s.providerOrchestrator.RegisterProvider("openrouter", "OpenRouter", orAdapter, enabled, priority)
		return nil
	default:
		return fmt.Errorf("provider %s update not supported", providerName)
	}
}

// getOrchestratorStrategyWrapper обертка для getOrchestratorStrategy
func (s *Server) getOrchestratorStrategyWrapper() (string, []string, []interface{}) {
	if s.providerOrchestrator == nil {
		return "", nil, nil
	}

	activeProviders := s.providerOrchestrator.GetActiveProviders()
	providersInfo := make([]interface{}, 0, len(activeProviders))
	for _, p := range activeProviders {
		providersInfo = append(providersInfo, map[string]interface{}{
			"id":       p.ID,
			"name":     p.Name,
			"priority": p.Priority,
			"enabled":  p.Enabled,
		})
	}

	availableStrategies := []string{
		string(ai.FirstSuccess),
		string(ai.MajorityVote),
		string(ai.AllResults),
		string(ai.HighestConfidence),
	}

	return string(s.providerOrchestrator.GetStrategy()), availableStrategies, providersInfo
}

// setOrchestratorStrategyWrapper обертка для setOrchestratorStrategy
func (s *Server) setOrchestratorStrategyWrapper(strategy string) error {
	if s.providerOrchestrator == nil {
		return fmt.Errorf("provider orchestrator not initialized")
	}

	validStrategies := map[string]bool{
		string(ai.FirstSuccess):      true,
		string(ai.MajorityVote):      true,
		string(ai.AllResults):        true,
		string(ai.HighestConfidence): true,
	}

	if !validStrategies[strategy] {
		return fmt.Errorf("invalid strategy: %s", strategy)
	}

	s.providerOrchestrator.SetStrategy(ai.AggregationStrategy(strategy))
	return nil
}

// getOrchestratorStatsWrapper обертка для getOrchestratorStats
func (s *Server) getOrchestratorStatsWrapper() (interface{}, error) {
	if s.providerOrchestrator == nil {
		return nil, fmt.Errorf("provider orchestrator not initialized")
	}

	stats := make(map[string]interface{})
	if s.monitoringManager != nil {
		activeProviders := s.providerOrchestrator.GetActiveProviders()
		providersStats := make([]map[string]interface{}, 0, len(activeProviders))

		allMetrics := s.monitoringManager.GetAllMetrics()
		metricsMap := make(map[string]*inframonitoring.ProviderMetrics)
		for i := range allMetrics.Providers {
			metricsMap[allMetrics.Providers[i].ID] = &allMetrics.Providers[i]
		}

		for _, p := range activeProviders {
			providerStats := map[string]interface{}{
				"id":       p.ID,
				"name":     p.Name,
				"priority": p.Priority,
				"enabled":  p.Enabled,
			}

			if metrics, exists := metricsMap[p.ID]; exists {
				providerStats["total_requests"] = metrics.TotalRequests
				providerStats["successful"] = metrics.SuccessfulRequests
				providerStats["failed"] = metrics.FailedRequests
				providerStats["current_requests"] = metrics.CurrentRequests
				providerStats["status"] = metrics.Status
				providerStats["rps"] = metrics.RequestsPerSecond

				if metrics.TotalRequests > 0 {
					providerStats["success_rate"] = float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests)
				} else {
					providerStats["success_rate"] = 0.0
				}

				providerStats["avg_latency_ms"] = metrics.AverageLatencyMs
			} else {
				providerStats["total_requests"] = 0
				providerStats["successful"] = 0
				providerStats["failed"] = 0
				providerStats["current_requests"] = 0
				providerStats["status"] = "unknown"
				providerStats["rps"] = 0.0
				providerStats["success_rate"] = 0.0
				providerStats["avg_latency_ms"] = 0.0
			}

			providersStats = append(providersStats, providerStats)
		}

		stats["providers"] = providersStats
		stats["total_providers"] = len(activeProviders)
		stats["system"] = map[string]interface{}{
			"total_requests":   allMetrics.System.TotalRequests,
			"total_successful": allMetrics.System.TotalSuccessful,
			"total_failed":     allMetrics.System.TotalFailed,
			"system_rps":       allMetrics.System.SystemRequestsPerSecond,
		}
	}

	stats["strategy"] = s.providerOrchestrator.GetStrategy()
	stats["timeout"] = s.providerOrchestrator.GetTimeout().String()

	return stats, nil
}
