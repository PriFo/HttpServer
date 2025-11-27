package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"httpserver/internal/infrastructure/workers"
	"httpserver/server/services"
)

// WorkerHandler обработчик для работы с конфигурацией воркеров и провайдеров
type WorkerHandler struct {
	*BaseHandler
	workerService *services.WorkerService
	logFunc       func(entry interface{}) // server.LogEntry, но без прямого импорта
	// Функции от Server для проверки подключений
	checkArliaiConnection      func(ctx context.Context, traceID string) (interface{}, error)
	checkOpenRouterConnection  func(ctx context.Context, traceID string, apiKey string) (interface{}, error)
	checkHuggingFaceConnection func(ctx context.Context, traceID string, apiKey string, baseURL string) (interface{}, error)
	getModelsFunc              func(ctx context.Context, traceID string, providerFilter string, filterStatus string, filterEnabled string, searchQuery string) (interface{}, error)
	// Функции для оркестратора
	getOrchestratorStrategy func() (string, []string, []interface{})
	setOrchestratorStrategy func(strategy string) error
	getOrchestratorStats    func() (interface{}, error)
	// Функции для обновления клиентов
	updateHuggingFaceClient    func(apiKey string, baseURL string) error
	updateProviderOrchestrator func(providerName string, adapter interface{}, enabled bool, priority int) error
}

// NewWorkerHandler создает новый обработчик воркеров
func NewWorkerHandler(
	baseHandler *BaseHandler,
	workerService *services.WorkerService,
	logFunc func(entry interface{}), // server.LogEntry, но без прямого импорта
	checkArliaiConnection func(ctx context.Context, traceID string) (interface{}, error),
	checkOpenRouterConnection func(ctx context.Context, traceID string, apiKey string) (interface{}, error),
	checkHuggingFaceConnection func(ctx context.Context, traceID string, apiKey string, baseURL string) (interface{}, error),
	getModelsFunc func(ctx context.Context, traceID string, providerFilter string, filterStatus string, filterEnabled string, searchQuery string) (interface{}, error),
	getOrchestratorStrategy func() (string, []string, []interface{}),
	setOrchestratorStrategy func(strategy string) error,
	getOrchestratorStats func() (interface{}, error),
	updateHuggingFaceClient func(apiKey string, baseURL string) error,
	updateProviderOrchestrator func(providerName string, adapter interface{}, enabled bool, priority int) error,
) *WorkerHandler {
	return &WorkerHandler{
		BaseHandler:                baseHandler,
		workerService:              workerService,
		logFunc:                    logFunc,
		checkArliaiConnection:      checkArliaiConnection,
		checkOpenRouterConnection:  checkOpenRouterConnection,
		checkHuggingFaceConnection: checkHuggingFaceConnection,
		getModelsFunc:              getModelsFunc,
		getOrchestratorStrategy:    getOrchestratorStrategy,
		setOrchestratorStrategy:    setOrchestratorStrategy,
		getOrchestratorStats:       getOrchestratorStats,
		updateHuggingFaceClient:    updateHuggingFaceClient,
		updateProviderOrchestrator: updateProviderOrchestrator,
	}
}

// HandleGetWorkerConfig обрабатывает запрос получения конфигурации воркеров
func (h *WorkerHandler) HandleGetWorkerConfig(w http.ResponseWriter, r *http.Request) {
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
		h.WriteJSONError(w, r, "Request timeout", http.StatusRequestTimeout)
		return
	default:
	}

	config, err := h.workerService.GetConfig()
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting worker config: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("GetWorkerConfig completed in %v", time.Since(startTime)),
		Endpoint:  r.URL.Path,
	})

	h.WriteJSONResponse(w, r, config, http.StatusOK)
}

// HandleUpdateWorkerConfig обрабатывает запрос обновления конфигурации воркеров
func (h *WorkerHandler) HandleUpdateWorkerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action string                 `json:"action"`
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	var err error
	var response map[string]interface{}

	switch req.Action {
	case "update_provider":
		var providerConfig workers.ProviderConfig
		if err := mapToStruct(req.Data, &providerConfig); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid provider data: %v", err), http.StatusBadRequest)
			return
		}
		if providerConfig.Name == "" {
			if name, ok := req.Data["name"].(string); ok {
				providerConfig.Name = name
			}
		}
		if providerConfig.Name == "" {
			h.WriteJSONError(w, r, "Provider name is required", http.StatusBadRequest)
			return
		}

		err = h.workerService.UpdateProvider(providerConfig.Name, &providerConfig)
		if err == nil {
			// Обновляем клиент провайдера, если это Hugging Face
			if providerConfig.Name == "huggingface" && h.updateHuggingFaceClient != nil {
				baseURL := ""
				if baseURLVal, ok := req.Data["base_url"].(string); ok {
					baseURL = baseURLVal
				}
				if baseURL == "" {
					baseURL = "https://api-inference.huggingface.co"
				}
				apiKey := ""
				if apiKeyVal, ok := req.Data["api_key"].(string); ok {
					apiKey = apiKeyVal
				}
				if err := h.updateHuggingFaceClient(apiKey, baseURL); err != nil {
					h.logFunc(LogEntry{
						Timestamp: time.Now(),
						Level:     "ERROR",
						Message:   fmt.Sprintf("Error updating Hugging Face client: %v", err),
						Endpoint:  r.URL.Path,
					})
				}

				// Обновляем адаптер в ProviderOrchestrator
				if h.updateProviderOrchestrator != nil {
					enabled := true
					if enabledVal, ok := req.Data["enabled"].(bool); ok {
						enabled = enabledVal
					}
					priority := 3
					if priorityVal, ok := req.Data["priority"].(float64); ok {
						priority = int(priorityVal)
					}
					// Передаем nil для adapter, так как он будет создан внутри Server
					h.updateProviderOrchestrator("huggingface", nil, enabled, priority)
				}
			}
		}
		response = map[string]interface{}{"message": "Provider updated successfully"}

	case "update_model":
		var modelConfig workers.ModelConfig
		if err := mapToStruct(req.Data, &modelConfig); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid model data: %v", err), http.StatusBadRequest)
			return
		}
		if modelConfig.Provider == "" {
			if provider, ok := req.Data["provider"].(string); ok {
				modelConfig.Provider = provider
			}
		}
		if modelConfig.Name == "" {
			if name, ok := req.Data["name"].(string); ok {
				modelConfig.Name = name
			}
		}
		if modelConfig.Provider == "" || modelConfig.Name == "" {
			h.WriteJSONError(w, r, "Model provider and name are required", http.StatusBadRequest)
			return
		}

		err = h.workerService.UpdateModel(modelConfig.Provider, modelConfig.Name, &modelConfig)
		response = map[string]interface{}{"message": "Model updated successfully"}

	case "set_default_provider":
		providerName := req.Data["provider"].(string)
		err = h.workerService.SetDefaultProvider(providerName)
		response = map[string]interface{}{"message": "Default provider updated successfully"}

	case "set_default_model":
		providerName := req.Data["provider"].(string)
		modelName := req.Data["model"].(string)
		err = h.workerService.SetDefaultModel(providerName, modelName)
		response = map[string]interface{}{"message": "Default model updated successfully"}

	case "set_max_workers", "set_global_max_workers":
		maxValue, ok := req.Data["max_workers"]
		if !ok {
			h.WriteJSONError(w, r, "max_workers is required", http.StatusBadRequest)
			return
		}
		maxWorkers := intFromInterface(maxValue)
		if maxWorkers <= 0 {
			h.WriteJSONError(w, r, "max_workers must be positive", http.StatusBadRequest)
			return
		}
		err = h.workerService.SetGlobalMaxWorkers(maxWorkers)
		response = map[string]interface{}{"message": "Global max workers updated successfully"}

	default:
		h.WriteJSONError(w, r, "Unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error updating worker config: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleGetAvailableProviders обрабатывает запрос получения списка провайдеров
func (h *WorkerHandler) HandleGetAvailableProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config, err := h.workerService.GetConfig()
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting providers: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	providers := config["providers"].(map[string]interface{})

	// Формируем список провайдеров с их моделями
	providersList := make([]map[string]interface{}, 0)
	for name, providerData := range providers {
		if providerMap, ok := providerData.(map[string]interface{}); ok {
			providerMap["name"] = name
			providersList = append(providersList, providerMap)
		}
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"providers":        providersList,
		"default_provider": config["default_provider"],
		"default_model":    config["default_model"],
	}, http.StatusOK)
}

// HandleCheckArliaiConnection обрабатывает запрос проверки подключения к Arliai
func (h *WorkerHandler) HandleCheckArliaiConnection(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.checkArliaiConnection(ctx, traceID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error checking Arliai connection: %v", err),
			Endpoint:  r.URL.Path,
		})
	}

	w.Header().Set("X-Request-ID", traceID)
	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleCheckOpenRouterConnection обрабатывает запрос проверки подключения к OpenRouter
func (h *WorkerHandler) HandleCheckOpenRouterConnection(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.checkOpenRouterConnection(ctx, traceID, apiKey)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error checking OpenRouter connection: %v", err),
			Endpoint:  r.URL.Path,
		})
	}

	w.Header().Set("X-Request-ID", traceID)
	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleCheckHuggingFaceConnection обрабатывает запрос проверки подключения к Hugging Face
func (h *WorkerHandler) HandleCheckHuggingFaceConnection(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	apiKey := os.Getenv("HUGGINGFACE_API_KEY")
	baseURL := "https://api-inference.huggingface.co"
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.checkHuggingFaceConnection(ctx, traceID, apiKey, baseURL)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error checking Hugging Face connection: %v", err),
			Endpoint:  r.URL.Path,
		})
	}

	w.Header().Set("X-Request-ID", traceID)
	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetModels обрабатывает запрос получения списка моделей
func (h *WorkerHandler) HandleGetModels(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	filterStatus := query.Get("status")
	filterEnabled := query.Get("enabled")
	searchQuery := query.Get("search")
	providerFilter := query.Get("provider")
	refresh := query.Get("refresh") == "1" || query.Get("refresh") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	
	// Передаем refresh через контекст
	if refresh {
		ctx = context.WithValue(ctx, "refresh", true)
	}

	result, err := h.getModelsFunc(ctx, traceID, providerFilter, filterStatus, filterEnabled, searchQuery)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting models: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Request-ID", traceID)
	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleOrchestratorStrategy обрабатывает запросы управления стратегией оркестратора
func (h *WorkerHandler) HandleOrchestratorStrategy(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	switch r.Method {
	case http.MethodGet:
		currentStrategy, availableStrategies, activeProviders := h.getOrchestratorStrategy()
		response := map[string]interface{}{
			"current_strategy":     currentStrategy,
			"available_strategies": availableStrategies,
			"active_providers":     activeProviders,
			"total_providers":      len(activeProviders),
		}
		h.WriteJSONResponse(w, r, response, http.StatusOK)

	case http.MethodPost, http.MethodPut:
		var req struct {
			Strategy string `json:"strategy"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		oldStrategy, _, _ := h.getOrchestratorStrategy()
		if err := h.setOrchestratorStrategy(req.Strategy); err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error setting orchestrator strategy: %v", err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"success":           true,
			"previous_strategy": oldStrategy,
			"current_strategy":  req.Strategy,
			"message":           fmt.Sprintf("Strategy changed from %s to %s", oldStrategy, req.Strategy),
		}
		h.WriteJSONResponse(w, r, response, http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleOrchestratorStats обрабатывает запрос получения статистики оркестратора
func (h *WorkerHandler) HandleOrchestratorStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.getOrchestratorStats()
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting orchestrator stats: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

func mapToStruct(src map[string]interface{}, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func intFromInterface(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	default:
		return 0
	}
}
