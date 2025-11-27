package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"httpserver/internal/infrastructure/ai"
	"httpserver/internal/infrastructure/workers"
)

// Вспомогательные функции для безопасного извлечения значений из map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		case int64:
			return int(val)
		}
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.0
}

// handleGetWorkerConfig возвращает текущую конфигурацию воркеров и моделей
func (s *Server) handleGetWorkerConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Добавляем таймаут для предотвращения зависания
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Проверяем контекст на отмену
	select {
	case <-ctx.Done():
		s.writeJSONError(w, r, "Request timeout", http.StatusRequestTimeout)
		return
	default:
	}

	if s.workerConfigManager == nil {
		s.writeJSONError(w, r, "Worker config manager not initialized", http.StatusInternalServerError)
		return
	}

	// Получаем конфигурацию с таймаутом
	configChan := make(chan map[string]interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("Panic in GetConfig: %v", rec)
				errChan <- fmt.Errorf("panic: %v", rec)
			}
		}()
		config := s.workerConfigManager.GetConfig()
		configChan <- config
	}()

	select {
	case config := <-configChan:
		log.Printf("GetWorkerConfig completed in %v", time.Since(startTime))
		s.writeJSONResponse(w, r, config, http.StatusOK)
	case err := <-errChan:
		log.Printf("Error in GetWorkerConfig: %v", err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
	case <-ctx.Done():
		log.Printf("GetWorkerConfig timeout after %v", time.Since(startTime))
		s.writeJSONError(w, r, "Request timeout", http.StatusRequestTimeout)
	}
}

// handleUpdateWorkerConfig обновляет конфигурацию воркеров
func (s *Server) handleUpdateWorkerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.workerConfigManager == nil {
		s.writeJSONError(w, r, "Worker config manager not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		Action string                 `json:"action"` // update_provider, update_model, set_default_provider, set_default_model, set_max_workers
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	var err error
	var response map[string]interface{}

	switch req.Action {
	case "update_provider":
		var providerConfig workers.ProviderConfig
		if err = mapToStruct(req.Data, &providerConfig); err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid provider config: %v", err), http.StatusBadRequest)
			return
		}
		providerName := req.Data["name"].(string)
		err = s.workerConfigManager.UpdateProvider(providerName, &providerConfig)

		// Обновляем клиент провайдера, если это Hugging Face
		if providerName == "huggingface" && err == nil {
			baseURL := providerConfig.BaseURL
			if baseURL == "" {
				baseURL = "https://api-inference.huggingface.co"
			}
			s.huggingfaceClient = ai.NewHuggingFaceClient(providerConfig.APIKey, baseURL)
			log.Printf("Hugging Face client updated with new API key (key length: %d)", len(providerConfig.APIKey))

			// Обновляем адаптер в ProviderOrchestrator, если он существует
			if s.providerOrchestrator != nil && providerConfig.APIKey != "" {
				// Создаем HuggingFaceClient через ai пакет
				huggingFaceClient := ai.NewHuggingFaceClient(providerConfig.APIKey, providerConfig.BaseURL)
				huggingfaceAdapter := ai.NewHuggingFaceProviderAdapter(huggingFaceClient)
				huggingfacePriority := providerConfig.Priority
				if huggingfacePriority == 0 {
					huggingfacePriority = 3 // Дефолтный приоритет
				}
				s.providerOrchestrator.RegisterProvider("huggingface", "Hugging Face", huggingfaceAdapter, providerConfig.Enabled, huggingfacePriority)
				log.Printf("Hugging Face provider updated in orchestrator (enabled: %v, priority: %d)", providerConfig.Enabled, huggingfacePriority)
			}
		}

		response = map[string]interface{}{"message": "Provider updated successfully"}

	case "update_model":
		var modelConfig workers.ModelConfig
		if err = mapToStruct(req.Data, &modelConfig); err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid model config: %v", err), http.StatusBadRequest)
			return
		}
		providerName := req.Data["provider"].(string)
		modelName := req.Data["name"].(string)
		err = s.workerConfigManager.UpdateModel(providerName, modelName, &modelConfig)
		response = map[string]interface{}{"message": "Model updated successfully"}

	case "set_default_provider":
		providerName := req.Data["provider"].(string)
		err = s.workerConfigManager.SetDefaultProvider(providerName)
		response = map[string]interface{}{"message": "Default provider updated successfully"}

	case "set_default_model":
		providerName := req.Data["provider"].(string)
		modelName := req.Data["model"].(string)
		err = s.workerConfigManager.SetDefaultModel(providerName, modelName)
		response = map[string]interface{}{"message": "Default model updated successfully"}

	case "set_max_workers":
		maxWorkers := int(req.Data["max_workers"].(float64))
		err = s.workerConfigManager.SetGlobalMaxWorkers(maxWorkers)
		response = map[string]interface{}{"message": "Global max workers updated successfully"}
	case "set_global_max_workers":
		maxWorkers := int(req.Data["max_workers"].(float64))
		err = s.workerConfigManager.SetGlobalMaxWorkers(maxWorkers)
		response = map[string]interface{}{"message": "Global max workers updated successfully"}

	default:
		s.writeJSONError(w, r, "Unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetAvailableProviders возвращает список доступных провайдеров
func (s *Server) handleGetAvailableProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.workerConfigManager == nil {
		s.writeJSONError(w, r, "Worker config manager not initialized", http.StatusInternalServerError)
		return
	}

	config := s.workerConfigManager.GetConfig()
	providers := config["providers"].(map[string]interface{})

	// Формируем список провайдеров с их моделями
	providersList := make([]map[string]interface{}, 0)
	for name, providerData := range providers {
		// Преобразуем interface{} в ProviderConfig
		var provider workers.ProviderConfig
		if providerMap, ok := providerData.(map[string]interface{}); ok {
			if err := mapToStruct(providerMap, &provider); err != nil {
				continue
			}
		} else if p, ok := providerData.(workers.ProviderConfig); ok {
			provider = p
		} else {
			continue
		}

		providerMap := map[string]interface{}{
			"name":        name,
			"enabled":     provider.Enabled,
			"priority":    provider.Priority,
			"max_workers": provider.MaxWorkers,
			"rate_limit":  provider.RateLimit,
			"models":      provider.Models,
		}
		providersList = append(providersList, providerMap)
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"providers":        providersList,
		"default_provider": config["default_provider"],
		"default_model":    config["default_model"],
	}, http.StatusOK)
}

// handleCheckArliaiConnection проверяет статус подключения к Arliai API
func (s *Server) handleCheckArliaiConnection(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	log.Printf("[%s] GET /api/workers/arliai/status", traceID)

	// Проверяем кеш
	if cached, ok := s.arliaiCache.GetStatus(); ok {
		cacheAge := s.arliaiCache.GetStatusAge()
		log.Printf("[%s] Returning cached status (age: %v)", traceID, cacheAge)

		response := APIResponse{
			Success:   true,
			Data:      cached,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
			Metadata: map[string]interface{}{
				"cached":      true,
				"cache_age_s": cacheAge.Seconds(),
			},
		}

		w.Header().Set("X-Request-ID", traceID)
		w.Header().Set("X-Cache", "HIT")
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
	}

	// Проверяем WorkerConfigManager для локальной информации
	var localStatus map[string]interface{}
	if s.workerConfigManager != nil {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err == nil && provider.Name == "arliai" {
			apiKey := provider.APIKey
			if apiKey == "" {
				apiKey = os.Getenv("ARLIAI_API_KEY")
			}

			model, modelErr := s.workerConfigManager.GetActiveModel(provider.Name)
			modelName := ""
			if modelErr == nil {
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
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	statusResp, err := s.arliaiClient.CheckConnection(ctx, traceID)
	if err != nil {
		// Если API недоступен, возвращаем локальный статус
		log.Printf("[%s] Arliai API check failed: %v, using local status", traceID, err)

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

		response := APIResponse{
			Success:   true,
			Data:      responseData,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
			Metadata: map[string]interface{}{
				"cached":    false,
				"api_error": err.Error(),
			},
		}

		// Кешируем результат даже при ошибке
		s.arliaiCache.SetStatus(response.Data)

		w.Header().Set("X-Request-ID", traceID)
		w.Header().Set("X-Cache", "MISS")
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
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

	response := APIResponse{
		Success:   true,
		Data:      responseData,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"cached": false,
		},
	}

	// Кешируем успешный результат
	s.arliaiCache.SetStatus(responseData)

	log.Printf("[%s] Status check completed (duration: %v, connected: %v)", traceID, time.Since(startTime), connected)

	w.Header().Set("X-Request-ID", traceID)
	w.Header().Set("X-Cache", "MISS")
	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleCheckOpenRouterConnection проверяет статус подключения к OpenRouter API
func (s *Server) handleCheckOpenRouterConnection(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	log.Printf("[%s] GET /api/workers/openrouter/status", traceID)

	// Проверяем WorkerConfigManager для локальной информации
	var localStatus map[string]interface{}
	var apiKey string
	if s.workerConfigManager != nil {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err == nil && provider.Name == "openrouter" {
			apiKey = provider.APIKey
			if apiKey == "" {
				apiKey = os.Getenv("OPENROUTER_API_KEY")
			}

			model, modelErr := s.workerConfigManager.GetActiveModel(provider.Name)
			modelName := ""
			if modelErr == nil {
				modelName = model.Name
			}

			localStatus = map[string]interface{}{
				"provider":    provider.Name,
				"has_api_key": apiKey != "",
				"model":       modelName,
				"enabled":     provider.Enabled,
			}
		} else {
			// Если openrouter не активен, проверяем, есть ли он в конфигурации
			config := s.workerConfigManager.GetConfig()
			if providers, ok := config["providers"].(map[string]interface{}); ok {
				if openrouterProvider, ok := providers["openrouter"].(map[string]interface{}); ok {
					hasKey := false
					if hasKeyVal, ok := openrouterProvider["has_api_key"].(bool); ok {
						hasKey = hasKeyVal
					}
					enabled := false
					if enabledVal, ok := openrouterProvider["enabled"].(bool); ok {
						enabled = enabledVal
					}
					localStatus = map[string]interface{}{
						"provider":    "openrouter",
						"has_api_key": hasKey,
						"enabled":     enabled,
					}
					// Пытаемся получить ключ из переменной окружения
					apiKey = os.Getenv("OPENROUTER_API_KEY")
					localStatus["has_api_key"] = apiKey != "" || hasKey
				}
			}
		}
	} else {
		// Если workerConfigManager не доступен, пробуем получить ключ из переменной окружения
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		localStatus = map[string]interface{}{
			"provider":    "openrouter",
			"has_api_key": apiKey != "",
			"enabled":     false,
		}
	}

	// Пытаемся проверить подключение через OpenRouter API
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Используем GetModels для проверки подключения
	// Создаем новый клиент для проверки
	serverOpenRouterClient := NewOpenRouterClient(apiKey)
	models, err := serverOpenRouterClient.GetModels(ctx, traceID)
	if err != nil {
		// Если API недоступен, возвращаем локальный статус
		log.Printf("[%s] OpenRouter API check failed: %v, using local status", traceID, err)

		connected := false
		if localStatus != nil {
			if hasKey, ok := localStatus["has_api_key"].(bool); ok {
				if enabled, ok2 := localStatus["enabled"].(bool); ok2 {
					connected = hasKey && enabled
				} else {
					connected = hasKey
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
			if model, ok := localStatus["model"].(string); ok && model != "" {
				responseData["model"] = model
			}
			responseData["enabled"] = localStatus["enabled"]
		}

		response := APIResponse{
			Success:   true,
			Data:      responseData,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
			Metadata: map[string]interface{}{
				"cached":    false,
				"api_error": err.Error(),
			},
		}

		w.Header().Set("X-Request-ID", traceID)
		w.Header().Set("X-Cache", "MISS")
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
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

	// Объединяем с локальной информацией
	if localStatus != nil {
		responseData["provider"] = localStatus["provider"]
		responseData["enabled"] = localStatus["enabled"]
		if model, ok := localStatus["model"].(string); ok && model != "" {
			responseData["model"] = model
		}
	}

	response := APIResponse{
		Success:   true,
		Data:      responseData,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"cached": false,
		},
	}

	log.Printf("[%s] OpenRouter status check completed (duration: %v, connected: %v, models: %d)", traceID, time.Since(startTime), connected, len(models))

	w.Header().Set("X-Request-ID", traceID)
	w.Header().Set("X-Cache", "MISS")
	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleCheckHuggingFaceConnection проверяет статус подключения к Hugging Face API
func (s *Server) handleCheckHuggingFaceConnection(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	log.Printf("[%s] GET /api/workers/huggingface/status", traceID)

	// Проверяем WorkerConfigManager для локальной информации
	var localStatus map[string]interface{}
	var apiKey string
	if s.workerConfigManager != nil {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err == nil && provider.Name == "huggingface" {
			apiKey = provider.APIKey
			if apiKey == "" {
				apiKey = os.Getenv("HUGGINGFACE_API_KEY")
			}

			model, modelErr := s.workerConfigManager.GetActiveModel(provider.Name)
			modelName := ""
			if modelErr == nil {
				modelName = model.Name
			}

			localStatus = map[string]interface{}{
				"provider":    provider.Name,
				"has_api_key": apiKey != "",
				"model":       modelName,
				"enabled":     provider.Enabled,
			}
		} else {
			// Если huggingface не активен, проверяем, есть ли он в конфигурации
			config := s.workerConfigManager.GetConfig()
			if providers, ok := config["providers"].(map[string]interface{}); ok {
				if hfProvider, ok := providers["huggingface"].(map[string]interface{}); ok {
					hasKey := false
					if hasKeyVal, ok := hfProvider["has_api_key"].(bool); ok {
						hasKey = hasKeyVal
					}
					enabled := false
					if enabledVal, ok := hfProvider["enabled"].(bool); ok {
						enabled = enabledVal
					}
					localStatus = map[string]interface{}{
						"provider":    "huggingface",
						"has_api_key": hasKey,
						"enabled":     enabled,
					}
					// Пытаемся получить ключ из переменной окружения
					apiKey = os.Getenv("HUGGINGFACE_API_KEY")
					localStatus["has_api_key"] = apiKey != "" || hasKey
				}
			}
		}
	} else {
		// Если workerConfigManager не доступен, пробуем получить ключ из переменной окружения
		apiKey = os.Getenv("HUGGINGFACE_API_KEY")
		localStatus = map[string]interface{}{
			"provider":    "huggingface",
			"has_api_key": apiKey != "",
			"enabled":     false,
		}
	}

	// Пытаемся проверить подключение через Hugging Face API
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Используем GetModels для проверки подключения
	// Создаем новый клиент для проверки
	baseURL := "https://api-inference.huggingface.co"
	if s.workerConfigManager != nil {
		// Пытаемся получить baseURL из конфигурации
		config := s.workerConfigManager.GetConfig()
		if providers, ok := config["providers"].(map[string]interface{}); ok {
			if hfProvider, ok := providers["huggingface"].(map[string]interface{}); ok {
				if baseURLVal, ok := hfProvider["base_url"].(string); ok && baseURLVal != "" {
					baseURL = baseURLVal
				}
			}
		}
	}
	serverHuggingFaceClient := NewHuggingFaceClient(apiKey, baseURL)

	statusResp, err := serverHuggingFaceClient.CheckConnection(ctx, traceID)
	if err != nil {
		// Если API недоступен, возвращаем локальный статус
		log.Printf("[%s] Hugging Face API check failed: %v, using local status", traceID, err)

		connected := false
		if localStatus != nil {
			if hasKey, ok := localStatus["has_api_key"].(bool); ok {
				if enabled, ok2 := localStatus["enabled"].(bool); ok2 {
					connected = hasKey && enabled
				} else {
					connected = hasKey
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
			if model, ok := localStatus["model"].(string); ok && model != "" {
				responseData["model"] = model
			}
			responseData["enabled"] = localStatus["enabled"]
		}

		response := APIResponse{
			Success:   true,
			Data:      responseData,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
			Metadata: map[string]interface{}{
				"cached":    false,
				"api_error": err.Error(),
			},
		}

		w.Header().Set("X-Request-ID", traceID)
		w.Header().Set("X-Cache", "MISS")
		s.writeJSONResponse(w, r, response, http.StatusOK)
		return
	}

	// Успешная проверка через API
	connected := statusResp["connected"].(bool)

	responseData := map[string]interface{}{
		"connected":        connected,
		"status":           statusResp["status"],
		"api_available":    true,
		"last_check":       statusResp["timestamp"],
		"models_count":     statusResp["models_count"],
		"response_time_ms": time.Since(startTime).Milliseconds(),
	}

	// Объединяем с локальной информацией
	if localStatus != nil {
		responseData["provider"] = localStatus["provider"]
		responseData["enabled"] = localStatus["enabled"]
		if model, ok := localStatus["model"].(string); ok && model != "" {
			responseData["model"] = model
		}
	}

	response := APIResponse{
		Success:   true,
		Data:      responseData,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"cached": false,
		},
	}

	log.Printf("[%s] Hugging Face status check completed (duration: %v, connected: %v, models: %v)", traceID, time.Since(startTime), connected, statusResp["models_count"])

	w.Header().Set("X-Request-ID", traceID)
	w.Header().Set("X-Cache", "MISS")
	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetModels возвращает список доступных моделей для активного провайдера
func (s *Server) handleGetModels(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	log.Printf("[%s] GET /api/workers/models", traceID)

	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	refresh := query.Get("refresh") == "1" || query.Get("refresh") == "true"
	opts := workerModelsOptions{
		FilterStatus:   query.Get("status"),
		FilterEnabled:  query.Get("enabled"),
		SearchQuery:    query.Get("search"),
		ProviderFilter: query.Get("provider"),
		Refresh:        refresh,
	}

	data, cached, apiErr := s.buildWorkerModelsData(r.Context(), traceID, opts)
	if apiErr != nil {
		errorResponse := APIResponse{
			Success: false,
			Error: &APIError{
				Code:      apiErr.Code,
				Message:   apiErr.Message,
				TraceID:   traceID,
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		}
		w.Header().Set("X-Request-ID", traceID)
		s.writeJSONResponse(w, r, errorResponse, apiErr.Status)
		return
	}

	response := APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"cached": cached,
		},
	}

	if cached {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	w.Header().Set("X-Request-ID", traceID)
	s.writeJSONResponse(w, r, response, http.StatusOK)
}

type workerModelsOptions struct {
	ProviderFilter string
	FilterStatus   string
	FilterEnabled  string
	SearchQuery    string
	Refresh        bool // Принудительное обновление кеша
}

type workerAPIError struct {
	Status  int
	Code    string
	Message string
}

func (e *workerAPIError) Error() string {
	return e.Message
}

func (s *Server) buildWorkerModelsData(ctx context.Context, traceID string, opts workerModelsOptions) (map[string]interface{}, bool, *workerAPIError) {
	// Если запрошено принудительное обновление, очищаем кеш
	if opts.Refresh {
		log.Printf("[%s] Refresh requested, clearing cache", traceID)
		s.arliaiCache.SetModels(nil)
	} else if cached, ok := s.arliaiCache.GetModels(); ok {
		if models, ok := cached.(map[string]interface{}); ok {
			return models, true, nil
		}
		log.Printf("[%s] Worker models cache contains unexpected type, resetting cache", traceID)
		s.arliaiCache.SetModels(nil)
	}

	if s.workerConfigManager == nil {
		return nil, false, &workerAPIError{
			Status:  http.StatusServiceUnavailable,
			Code:    "SERVICE_UNAVAILABLE",
			Message: "Worker config manager not initialized",
		}
	}

	provider, err := s.resolveWorkerProvider(opts.ProviderFilter)
	if err != nil {
		if apiErr, ok := err.(*workerAPIError); ok {
			return nil, false, apiErr
		}
		return nil, false, &workerAPIError{
			Status:  http.StatusInternalServerError,
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		return nil, false, &workerAPIError{
			Status:  http.StatusRequestTimeout,
			Code:    "REQUEST_TIMEOUT",
			Message: "Request timeout",
		}
	default:
	}

	log.Printf("[%s] Fetching models for provider: %s (refresh: %v)", traceID, provider.Name, opts.Refresh)
	apiModels, apiErr := s.fetchProviderModels(timeoutCtx, traceID, provider)
	
	if apiErr != nil {
		log.Printf("[%s] Error fetching models from API: %v", traceID, apiErr)
	} else {
		log.Printf("[%s] Successfully fetched %d models from API", traceID, len(apiModels))
	}
	
	localModels := s.buildLocalModels(provider)
	log.Printf("[%s] Local models count: %d", traceID, len(localModels))

	finalModels, mergeErr := s.mergeModels(provider, localModels, apiModels, apiErr == nil)
	if mergeErr != nil {
		log.Printf("[%s] Failed to merge models: %v", traceID, mergeErr)
	}

	filteredModels := filterWorkerModels(finalModels, opts.FilterStatus, opts.FilterEnabled, opts.SearchQuery)
	sortWorkerModels(filteredModels)

	defaultModel := ""
	config := s.workerConfigManager.GetConfig()
	if dm, ok := config["default_model"].(string); ok {
		defaultModel = dm
	}
	markDefaultModel(filteredModels, defaultModel)

	log.Printf("[%s] Models summary: total_before_filter=%d, filtered=%d, provider=%s, api_available=%v",
		traceID, len(finalModels), len(filteredModels), provider.Name, apiErr == nil)

	if len(finalModels) <= 2 && apiErr == nil {
		log.Printf("[%s] WARNING: Only %d models retrieved from API. Expected more models.", traceID, len(finalModels))
	}
	
	if len(finalModels) == 0 && apiErr == nil {
		log.Printf("[%s] WARNING: No models found. This might indicate an issue with API key or API response.", traceID)
	}

	responseData := map[string]interface{}{
		"models":              filteredModels,
		"provider":            provider.Name,
		"default_model":       defaultModel,
		"total":               len(filteredModels),
		"total_before_filter": len(finalModels),
		"api_available":       apiErr == nil,
		"api_models_count":    len(apiModels),
		"local_models_count":  len(localModels),
		"filters": map[string]interface{}{
			"status":  opts.FilterStatus,
			"enabled": opts.FilterEnabled,
			"search":  opts.SearchQuery,
		},
	}

	if apiErr != nil {
		responseData["api_error"] = apiErr.Error()
		// Добавляем более детальную информацию об ошибке
		if strings.Contains(apiErr.Error(), "API ключ") || strings.Contains(apiErr.Error(), "API key") {
			responseData["error_type"] = "missing_api_key"
			responseData["error_message"] = fmt.Sprintf("API ключ для провайдера '%s' не установлен", provider.Name)
		} else if strings.Contains(apiErr.Error(), "timeout") || strings.Contains(apiErr.Error(), "context deadline") {
			responseData["error_type"] = "timeout"
			responseData["error_message"] = "Превышено время ожидания ответа от API"
		} else if strings.Contains(apiErr.Error(), "401") || strings.Contains(apiErr.Error(), "unauthorized") {
			responseData["error_type"] = "unauthorized"
			responseData["error_message"] = "Неверный API ключ или отсутствует авторизация"
		} else if strings.Contains(apiErr.Error(), "403") || strings.Contains(apiErr.Error(), "forbidden") {
			responseData["error_type"] = "forbidden"
			responseData["error_message"] = "Доступ запрещен. Проверьте права доступа API ключа"
		} else {
			responseData["error_type"] = "unknown"
			responseData["error_message"] = apiErr.Error()
		}
	}

	s.arliaiCache.SetModels(responseData)
	log.Printf("[%s] Models fetch completed (count: %d)", traceID, len(finalModels))

	return responseData, false, nil
}

func (s *Server) resolveWorkerProvider(providerFilter string) (*workers.ProviderConfig, error) {
	if providerFilter == "" {
		provider, err := s.workerConfigManager.GetActiveProvider()
		if err != nil {
			return nil, &workerAPIError{
				Status:  http.StatusBadRequest,
				Code:    "NO_ACTIVE_PROVIDER",
				Message: fmt.Sprintf("No active provider: %v", err),
			}
		}
		return provider, nil
	}

	config := s.workerConfigManager.GetConfig()
	if providers, ok := config["providers"].(map[string]interface{}); ok {
		if providerData, ok := providers[providerFilter].(map[string]interface{}); ok {
			return &workers.ProviderConfig{
				Name:     getString(providerData, "name"),
				BaseURL:  getString(providerData, "base_url"),
				Enabled:  getBool(providerData, "enabled"),
				Priority: getInt(providerData, "priority"),
			}, nil
		}
	}

	if s.serviceDB != nil {
		if providersFromDB, err := s.serviceDB.GetProviders(); err == nil {
			for _, p := range providersFromDB {
				// Используем Type или Name для сравнения
				if p.Type == providerFilter || p.Name == providerFilter {
					// Извлекаем APIKey и BaseURL из config JSON
					var apiKey, baseURL string
					if p.Config != "" {
						var configMap map[string]interface{}
						if err := json.Unmarshal([]byte(p.Config), &configMap); err == nil {
							if ak, ok := configMap["api_key"].(string); ok {
								apiKey = ak
							}
							if bu, ok := configMap["base_url"].(string); ok {
								baseURL = bu
							}
						}
					}
					return &workers.ProviderConfig{
						Name:     p.Name,
						APIKey:   apiKey,
						BaseURL:  baseURL,
						Enabled:  p.IsActive,
						Priority: 0, // Priority больше не хранится в новой структуре
					}, nil
				}
			}
		}
	}

	return nil, &workerAPIError{
		Status:  http.StatusBadRequest,
		Code:    "PROVIDER_NOT_FOUND",
		Message: fmt.Sprintf("Provider '%s' not found", providerFilter),
	}
}

func (s *Server) fetchProviderModels(ctx context.Context, traceID string, provider *workers.ProviderConfig) ([]ArliaiModel, error) {
	apiKey := provider.APIKey
	
	// Если API ключ не задан в provider, пытаемся получить из WorkerConfigManager
	if apiKey == "" && s.workerConfigManager != nil {
		// Получаем провайдера из конфигурации, чтобы убедиться, что у нас актуальный API ключ
		if configProvider, err := s.workerConfigManager.GetActiveProvider(); err == nil {
			if configProvider.Name == provider.Name && configProvider.APIKey != "" {
				apiKey = configProvider.APIKey
				log.Printf("[%s] Using API key from WorkerConfigManager for provider %s", traceID, provider.Name)
			}
		}
	}
	
	// Fallback на переменные окружения, если ключ все еще не найден
	if apiKey == "" {
		switch provider.Name {
		case "openrouter":
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		case "arliai":
			apiKey = os.Getenv("ARLIAI_API_KEY")
		case "huggingface":
			apiKey = os.Getenv("HUGGINGFACE_API_KEY")
		case "edenai":
			apiKey = os.Getenv("EDENAI_API_KEY")
		}
		if apiKey != "" {
			log.Printf("[%s] Using API key from environment variable for provider %s", traceID, provider.Name)
		}
	}

	// Проверяем наличие API ключа перед попыткой загрузки
	if apiKey == "" {
		return nil, fmt.Errorf("API ключ для провайдера '%s' не установлен. Установите API ключ в разделе 'Воркеры' или через переменную окружения %s_API_KEY", provider.Name, strings.ToUpper(provider.Name))
	}

	switch provider.Name {
	case "openrouter":
		client := NewOpenRouterClient(apiKey)
		openrouterModels, err := client.GetModels(ctx, traceID)
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки моделей OpenRouter: %w", err)
		}
		apiModels := make([]ArliaiModel, 0, len(openrouterModels))
		for _, orModel := range openrouterModels {
			model := ArliaiModel{
				ID:          orModel.ID,
				Name:        orModel.Name,
				Description: orModel.Description,
				Status:      "active",
				MaxTokens:   orModel.Context,
				Speed:       "medium",
				Quality:     "high",
			}
			if model.Name == "" {
				model.Name = model.ID
			}
			apiModels = append(apiModels, model)
		}
		return apiModels, nil
	case "huggingface":
		baseURL := provider.BaseURL
		if baseURL == "" {
			baseURL = "https://api-inference.huggingface.co"
		}
		client := NewHuggingFaceClient(apiKey, baseURL)
		models, err := client.GetModels(ctx, traceID)
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки моделей Hugging Face: %w", err)
		}
		return models, nil
	case "edenai":
		baseURL := provider.BaseURL
		if baseURL == "" {
			baseURL = "https://api.edenai.run/v2"
		}
		client := NewEdenAIClient(apiKey, baseURL)
		models, err := client.GetModels(ctx, traceID)
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки моделей EdenAI: %w", err)
		}
		return models, nil
	default:
		if s.arliaiClient == nil {
			return nil, fmt.Errorf("клиент Arliai не инициализирован")
		}
		aiModels, err := s.arliaiClient.GetModels(ctx, traceID)
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки моделей Arliai: %w", err)
		}
		apiModels := make([]ArliaiModel, 0, len(aiModels))
		for _, m := range aiModels {
			apiModels = append(apiModels, ArliaiModel{
				ID:          m.ID,
				Name:        m.Name,
				Speed:       m.Speed,
				Quality:     m.Quality,
				Description: m.Description,
				Status:      m.Status,
				MaxTokens:   m.MaxTokens,
				Tags:        m.Tags,
			})
		}
		return apiModels, nil
	}
}

func (s *Server) buildLocalModels(provider *workers.ProviderConfig) []map[string]interface{} {
	localModels := make([]map[string]interface{}, 0)
	for _, model := range provider.Models {
		if !model.Enabled {
			continue
		}
		localModels = append(localModels, map[string]interface{}{
			"id":          model.Name,
			"name":        model.Name,
			"provider":    model.Provider,
			"enabled":     model.Enabled,
			"priority":    model.Priority,
			"max_tokens":  model.MaxTokens,
			"temperature": model.Temperature,
			"speed":       model.Speed,
			"quality":     model.Quality,
			"status":      "active",
		})
	}
	return localModels
}

func (s *Server) mergeModels(provider *workers.ProviderConfig, localModels []map[string]interface{}, apiModels []ArliaiModel, hasAPI bool) ([]map[string]interface{}, error) {
	if !hasAPI || len(apiModels) == 0 {
		return localModels, nil
	}

	modelMap := make(map[string]map[string]interface{})
	for _, apiModel := range apiModels {
		modelMap[apiModel.ID] = map[string]interface{}{
			"id":          apiModel.ID,
			"name":        apiModel.Name,
			"provider":    provider.Name,
			"speed":       apiModel.Speed,
			"quality":     apiModel.Quality,
			"description": apiModel.Description,
			"status":      apiModel.Status,
			"max_tokens":  apiModel.MaxTokens,
			"tags":        apiModel.Tags,
		}
	}

	for _, localModel := range localModels {
		modelID := getString(localModel, "id")
		if modelID == "" {
			modelID = getString(localModel, "name")
		}
		if modelID == "" {
			continue
		}
		if existing, ok := modelMap[modelID]; ok {
			for k, v := range localModel {
				existing[k] = v
			}
		} else {
			modelMap[modelID] = localModel
		}
	}

	finalModels := make([]map[string]interface{}, 0, len(modelMap))
	for _, model := range modelMap {
		finalModels = append(finalModels, model)
	}

	updatedModels := make([]workers.ModelConfig, 0, len(finalModels))
	for _, modelData := range finalModels {
		modelConfig := workers.ModelConfig{
			Name:         getString(modelData, "name"),
			Provider:     provider.Name,
			Enabled:      getBool(modelData, "enabled"),
			Priority:     getInt(modelData, "priority"),
			MaxTokens:    getInt(modelData, "max_tokens"),
			Temperature:  getFloat64(modelData, "temperature"),
			Speed:        getString(modelData, "speed"),
			Quality:      getString(modelData, "quality"),
			CostPerToken: getFloat64(modelData, "cost_per_token"),
		}
		if modelConfig.Name == "" {
			modelConfig.Name = getString(modelData, "id")
		}
		if modelConfig.Temperature == 0 {
			modelConfig.Temperature = 0.3
		}
		if modelConfig.Speed == "" {
			modelConfig.Speed = "medium"
		}
		if modelConfig.Quality == "" {
			modelConfig.Quality = "high"
		}
		updatedModels = append(updatedModels, modelConfig)
	}

	if len(updatedModels) > 0 {
		updatedProvider := *provider
		updatedProvider.Models = updatedModels
		if err := s.workerConfigManager.UpdateProvider(provider.Name, &updatedProvider); err != nil {
			return finalModels, err
		}
	}

	return finalModels, nil
}

func filterWorkerModels(models []map[string]interface{}, filterStatus, filterEnabled, searchQuery string) []map[string]interface{} {
	filtered := make([]map[string]interface{}, 0, len(models))
	for _, model := range models {
		if filterStatus != "" && filterStatus != "all" {
			if status, ok := model["status"].(string); !ok || status != filterStatus {
				continue
			}
		}
		if filterEnabled != "" && filterEnabled != "all" {
			enabled := model["enabled"] == true
			if filterEnabled == "true" && !enabled {
				continue
			}
			if filterEnabled == "false" && enabled {
				continue
			}
		}
		if searchQuery != "" {
			modelName := getString(model, "name")
			if modelName == "" {
				modelName = getString(model, "id")
			}
			if !strings.Contains(strings.ToLower(modelName), strings.ToLower(searchQuery)) {
				continue
			}
		}
		filtered = append(filtered, model)
	}
	return filtered
}

func sortWorkerModels(models []map[string]interface{}) {
	sort.Slice(models, func(i, j int) bool {
		priI, okI := models[i]["priority"].(int)
		priJ, okJ := models[j]["priority"].(int)
		if !okI || !okJ || priI == priJ {
			nameI := getString(models[i], "name")
			if nameI == "" {
				nameI = getString(models[i], "id")
			}
			nameJ := getString(models[j], "name")
			if nameJ == "" {
				nameJ = getString(models[j], "id")
			}
			return nameI < nameJ
		}
		return priI < priJ
	})
}

func markDefaultModel(models []map[string]interface{}, defaultModel string) {
	for i := range models {
		modelName := getString(models[i], "name")
		if modelName == "" {
			modelName = getString(models[i], "id")
		}
		models[i]["is_default"] = modelName == defaultModel
	}
}

// mapToStruct преобразует map в структуру
func mapToStruct(m map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
