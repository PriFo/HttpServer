package ai

import (
	"sync"
	"time"
)

// MetricsCollector собирает метрики для провайдеров и операций нормализации
// В будущем может быть расширен для интеграции с Prometheus
type MetricsCollector struct {
	mu sync.RWMutex

	// Метрики запросов к провайдерам
	providerRequestsTotal map[string]int64
	providerErrorsTotal   map[string]int64
	providerDurationTotal map[string]time.Duration
	providerRequestCount  map[string]int64 // Количество запросов для расчета среднего

	// Метрики нормализации
	normalizationRequestsTotal int64
	normalizationDurationTotal time.Duration
	normalizationRequestCount  int64
}

// NewMetricsCollector создает новый сборщик метрик
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		providerRequestsTotal: make(map[string]int64),
		providerErrorsTotal:   make(map[string]int64),
		providerDurationTotal: make(map[string]time.Duration),
		providerRequestCount:  make(map[string]int64),
	}
}

// IncrementProviderRequest инкрементирует счетчик запросов к провайдеру
func (mc *MetricsCollector) IncrementProviderRequest(providerID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.providerRequestsTotal[providerID]++
	mc.providerRequestCount[providerID]++
}

// IncrementProviderError инкрементирует счетчик ошибок провайдера
func (mc *MetricsCollector) IncrementProviderError(providerID string, errorType string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	key := providerID + ":" + errorType
	mc.providerErrorsTotal[key]++
}

// RecordProviderDuration записывает длительность запроса к провайдеру
func (mc *MetricsCollector) RecordProviderDuration(providerID string, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.providerDurationTotal[providerID] += duration
}

// IncrementNormalizationRequest инкрементирует счетчик запросов нормализации
func (mc *MetricsCollector) IncrementNormalizationRequest() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.normalizationRequestsTotal++
	mc.normalizationRequestCount++
}

// RecordNormalizationDuration записывает длительность операции нормализации
func (mc *MetricsCollector) RecordNormalizationDuration(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.normalizationDurationTotal += duration
}

// GetProviderMetrics возвращает метрики для провайдера
func (mc *MetricsCollector) GetProviderMetrics(providerID string) map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	requests := mc.providerRequestsTotal[providerID]
	count := mc.providerRequestCount[providerID]
	duration := mc.providerDurationTotal[providerID]

	avgDuration := time.Duration(0)
	if count > 0 {
		avgDuration = duration / time.Duration(count)
	}

	errors := int64(0)
	for key, value := range mc.providerErrorsTotal {
		if len(key) > len(providerID) && key[:len(providerID)] == providerID {
			errors += value
		}
	}

	return map[string]interface{}{
		"requests_total":    requests,
		"errors_total":      errors,
		"duration_total_ms": duration.Milliseconds(),
		"duration_avg_ms":   avgDuration.Milliseconds(),
		"request_count":     count,
	}
}

// GetAllMetrics возвращает все метрики
func (mc *MetricsCollector) GetAllMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	providers := make(map[string]interface{})
	for providerID := range mc.providerRequestsTotal {
		providers[providerID] = mc.GetProviderMetrics(providerID)
	}

	avgNormalizationDuration := time.Duration(0)
	if mc.normalizationRequestCount > 0 {
		avgNormalizationDuration = mc.normalizationDurationTotal / time.Duration(mc.normalizationRequestCount)
	}

	return map[string]interface{}{
		"providers": map[string]interface{}{
			"metrics": providers,
		},
		"normalization": map[string]interface{}{
			"requests_total":    mc.normalizationRequestsTotal,
			"duration_total_ms": mc.normalizationDurationTotal.Milliseconds(),
			"duration_avg_ms":   avgNormalizationDuration.Milliseconds(),
			"request_count":     mc.normalizationRequestCount,
		},
	}
}
