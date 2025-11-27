package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProviderMetricsResponse структура ответа метрик провайдера
type ProviderMetricsResponse struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	ActiveChannels     int       `json:"active_channels"`
	CurrentRequests    int       `json:"current_requests"`
	TotalRequests      int64     `json:"total_requests"`
	SuccessfulRequests int64     `json:"successful_requests"`
	FailedRequests     int64     `json:"failed_requests"`
	AverageLatencyMs   float64   `json:"average_latency_ms"`
	LastRequestTime    string    `json:"last_request_time"`
	Status             string    `json:"status"`
	RequestsPerSecond  float64   `json:"requests_per_second"`
}

// SystemStatsResponse структура ответа системной статистики
type SystemStatsResponse struct {
	TotalProviders          int     `json:"total_providers"`
	ActiveProviders         int     `json:"active_providers"`
	TotalRequests           int64   `json:"total_requests"`
	TotalSuccessful         int64   `json:"total_successful"`
	TotalFailed             int64   `json:"total_failed"`
	SystemRequestsPerSecond float64 `json:"system_requests_per_second"`
	Timestamp               string  `json:"timestamp"`
}

// MonitoringDataResponse структура ответа данных мониторинга
type MonitoringDataResponse struct {
	Providers []ProviderMetricsResponse `json:"providers"`
	System    SystemStatsResponse       `json:"system"`
}

// HandleGetProvidersGin обработчик получения статуса провайдеров для Gin
// @Summary Получить статус AI-провайдеров
// @Description Возвращает текущий статус и метрики всех AI-провайдеров
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} MonitoringDataResponse "Данные мониторинга провайдеров"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/monitoring/providers [get]
func (h *MonitoringHandler) HandleGetProvidersGin(c *gin.Context) {
	// Здесь должна быть логика получения данных мониторинга
	// Пример структуры ответа
	response := MonitoringDataResponse{
		Providers: []ProviderMetricsResponse{},
		System: SystemStatsResponse{
			TotalProviders: 0,
			ActiveProviders: 0,
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}

	SendJSONResponse(c, http.StatusOK, response)
}

// HandleGetProviderMetricsGin обработчик получения метрик провайдера для Gin
// @Summary Получить метрики провайдера
// @Description Возвращает детальные метрики конкретного AI-провайдера
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "ID провайдера"
// @Success 200 {object} ProviderMetricsResponse "Метрики провайдера"
// @Failure 404 {object} ErrorResponse "Провайдер не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/monitoring/providers/{id} [get]
func (h *MonitoringHandler) HandleGetProviderMetricsGin(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		SendJSONError(c, http.StatusBadRequest, "Provider ID is required")
		return
	}

	// Здесь должна быть логика получения метрик провайдера
	response := ProviderMetricsResponse{
		ID:    providerID,
		Name:  "Provider Name",
		Status: "active",
	}

	SendJSONResponse(c, http.StatusOK, response)
}

// HandleStartProviderGin обработчик запуска провайдера для Gin
// @Summary Запустить провайдер
// @Description Запускает указанный AI-провайдер
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "ID провайдера"
// @Success 200 {object} map[string]interface{} "Успешный запуск"
// @Failure 404 {object} ErrorResponse "Провайдер не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/monitoring/providers/{id}/start [post]
func (h *MonitoringHandler) HandleStartProviderGin(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		SendJSONError(c, http.StatusBadRequest, "Provider ID is required")
		return
	}

	result := map[string]interface{}{
		"message":    fmt.Sprintf("Provider %s started", providerID),
		"provider_id": providerID,
	}

	SendJSONResponse(c, http.StatusOK, result)
}

// HandleStopProviderGin обработчик остановки провайдера для Gin
// @Summary Остановить провайдер
// @Description Останавливает указанный AI-провайдер
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "ID провайдера"
// @Success 200 {object} map[string]interface{} "Успешная остановка"
// @Failure 404 {object} ErrorResponse "Провайдер не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/monitoring/providers/{id}/stop [post]
func (h *MonitoringHandler) HandleStopProviderGin(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		SendJSONError(c, http.StatusBadRequest, "Provider ID is required")
		return
	}

	result := map[string]interface{}{
		"message":    fmt.Sprintf("Provider %s stopped", providerID),
		"provider_id": providerID,
	}

	SendJSONResponse(c, http.StatusOK, result)
}

