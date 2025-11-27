package handlers

import (
	"net/http"
	"time"

	"httpserver/websearch"
)

// WebSearchAdminHandler обработчик для административных операций с веб-поиском
// Поддерживает как *websearch.Client, так и *websearch.MultiProviderClient
type WebSearchAdminHandler struct {
	*BaseHandler
	client websearch.SearchClientInterface
}

// NewWebSearchAdminHandler создает новый административный обработчик
// Принимает как *websearch.Client, так и *websearch.MultiProviderClient
func NewWebSearchAdminHandler(baseHandler *BaseHandler, client websearch.SearchClientInterface) *WebSearchAdminHandler {
	return &WebSearchAdminHandler{
		BaseHandler: baseHandler,
		client:      client,
	}
}

// HandleListProviders возвращает список всех провайдеров
// GET /api/admin/websearch/providers
// Поддерживает как простой Client, так и MultiProviderClient
func (h *WebSearchAdminHandler) HandleListProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var providers []map[string]interface{}

	// Проверяем, является ли клиент MultiProviderClient
	if multiClient, ok := h.client.(*websearch.MultiProviderClient); ok {
		// Получаем статистику провайдеров
		stats := multiClient.GetStats()
		for name, stat := range stats {
			providers = append(providers, map[string]interface{}{
				"name":     name,
				"type":     name, // Используем имя как тип
				"enabled":  true,
				"stats":    stat,
			})
		}
	} else {
		// Простой клиент - возвращаем только информацию о DuckDuckGo
		providers = []map[string]interface{}{
			{
				"name":     "duckduckgo",
				"type":     "duckduckgo",
				"enabled":  h.client != nil,
				"base_url": "https://api.duckduckgo.com",
			},
		}
	}

	response := map[string]interface{}{
		"success":  true,
		"providers": providers,
		"count":    len(providers),
	}

	h.BaseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleCreateProvider создает или обновляет провайдер
// POST /api/admin/websearch/providers
// Упрощенная версия - только для информации
func (h *WebSearchAdminHandler) HandleCreateProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.Error(w, "Multi-provider support is temporarily disabled. Only DuckDuckGo is available.", http.StatusNotImplemented)
}

// HandleUpdateProvider обновляет конкретный провайдер
// PUT /api/admin/websearch/providers/{name}
// Упрощенная версия - только для информации
func (h *WebSearchAdminHandler) HandleUpdateProvider(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.Error(w, "Multi-provider support is temporarily disabled. Only DuckDuckGo is available.", http.StatusNotImplemented)
}

// HandleDeleteProvider удаляет провайдер (soft delete)
// DELETE /api/admin/websearch/providers/{name}
// Упрощенная версия - только для информации
func (h *WebSearchAdminHandler) HandleDeleteProvider(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.Error(w, "Multi-provider support is temporarily disabled. Only DuckDuckGo is available.", http.StatusNotImplemented)
}

// HandleReloadProviders перезагружает конфигурацию провайдеров
// POST /api/admin/websearch/providers/reload
// Упрощенная версия - только для информации
func (h *WebSearchAdminHandler) HandleReloadProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "DuckDuckGo provider is active",
		"count":   1,
	}

	h.BaseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleGetStats возвращает статистику по провайдерам
// GET /api/admin/websearch/stats
// Поддерживает как простой Client, так и MultiProviderClient
func (h *WebSearchAdminHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.client == nil {
		http.Error(w, "Web search client is not configured", http.StatusServiceUnavailable)
		return
	}

	var providerStats map[string]interface{}
	var cacheStats map[string]interface{}

	// Проверяем, является ли клиент MultiProviderClient
	if multiClient, ok := h.client.(*websearch.MultiProviderClient); ok {
		// Получаем статистику провайдеров
		providerStats = multiClient.GetStats()
		
		// Получаем статистику кэша
		cacheStats = multiClient.GetCacheStats()
		
		// Добавляем общую информацию
		cacheStats["active_providers"] = multiClient.GetActiveProvidersCount()
	} else {
		// Простой клиент - возвращаем базовую информацию
		providerStats = map[string]interface{}{
			"duckduckgo": map[string]interface{}{
				"enabled": true,
				"type":    "duckduckgo",
			},
		}
		
		cacheStats = map[string]interface{}{
			"enabled": true,
			"type":    "memory",
		}
	}

	response := map[string]interface{}{
		"success":   true,
		"providers": providerStats,
		"cache":     cacheStats,
		"timestamp": time.Now(),
	}

	h.BaseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

