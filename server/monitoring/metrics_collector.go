package monitoring

import (
	"sync"
	"time"
)

// MetricsCollector собирает метрики производительности
type MetricsCollector struct {
	mu sync.RWMutex

	// HTTP метрики
	httpRequestsTotal    int64
	httpRequestsSuccess  int64
	httpRequestsError    int64
	httpRequestDuration  []time.Duration
	httpRequestDurationMu sync.RWMutex

	// Database метрики
	dbQueriesTotal       int64
	dbQueriesDuration    []time.Duration
	dbQueriesDurationMu  sync.RWMutex
	dbConnectionsActive  int64
	dbConnectionsIdle     int64

	// Системные метрики
	startTime            time.Time
	lastResetTime        time.Time
}

// NewMetricsCollector создает новый сборщик метрик
func NewMetricsCollector() *MetricsCollector {
	now := time.Now()
	return &MetricsCollector{
		startTime:     now,
		lastResetTime: now,
	}
}

// RecordHTTPRequest записывает HTTP запрос
func (mc *MetricsCollector) RecordHTTPRequest(success bool, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.httpRequestsTotal++
	if success {
		mc.httpRequestsSuccess++
	} else {
		mc.httpRequestsError++
	}

	mc.httpRequestDurationMu.Lock()
	mc.httpRequestDuration = append(mc.httpRequestDuration, duration)
	// Ограничиваем размер массива (храним последние 1000 записей)
	if len(mc.httpRequestDuration) > 1000 {
		mc.httpRequestDuration = mc.httpRequestDuration[len(mc.httpRequestDuration)-1000:]
	}
	mc.httpRequestDurationMu.Unlock()
}

// RecordDBQuery записывает запрос к БД
func (mc *MetricsCollector) RecordDBQuery(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.dbQueriesTotal++

	mc.dbQueriesDurationMu.Lock()
	mc.dbQueriesDuration = append(mc.dbQueriesDuration, duration)
	// Ограничиваем размер массива
	if len(mc.dbQueriesDuration) > 1000 {
		mc.dbQueriesDuration = mc.dbQueriesDuration[len(mc.dbQueriesDuration)-1000:]
	}
	mc.dbQueriesDurationMu.Unlock()
}

// SetDBConnections устанавливает количество подключений к БД
func (mc *MetricsCollector) SetDBConnections(active, idle int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.dbConnectionsActive = active
	mc.dbConnectionsIdle = idle
}

// GetMetrics возвращает текущие метрики
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Вычисляем среднее время ответа HTTP
	mc.httpRequestDurationMu.RLock()
	avgHTTPDuration := time.Duration(0)
	if len(mc.httpRequestDuration) > 0 {
		total := time.Duration(0)
		for _, d := range mc.httpRequestDuration {
			total += d
		}
		avgHTTPDuration = total / time.Duration(len(mc.httpRequestDuration))
	}
	mc.httpRequestDurationMu.RUnlock()

	// Вычисляем среднее время запросов к БД
	mc.dbQueriesDurationMu.RLock()
	avgDBDuration := time.Duration(0)
	if len(mc.dbQueriesDuration) > 0 {
		total := time.Duration(0)
		for _, d := range mc.dbQueriesDuration {
			total += d
		}
		avgDBDuration = total / time.Duration(len(mc.dbQueriesDuration))
	}
	mc.dbQueriesDurationMu.RUnlock()

	// Вычисляем success rate
	successRate := 0.0
	if mc.httpRequestsTotal > 0 {
		successRate = float64(mc.httpRequestsSuccess) / float64(mc.httpRequestsTotal) * 100
	}

	// Вычисляем requests per second
	uptime := time.Since(mc.startTime).Seconds()
	requestsPerSecond := 0.0
	if uptime > 0 {
		requestsPerSecond = float64(mc.httpRequestsTotal) / uptime
	}

	return map[string]interface{}{
		"http": map[string]interface{}{
			"requests_total":    mc.httpRequestsTotal,
			"requests_success":   mc.httpRequestsSuccess,
			"requests_error":    mc.httpRequestsError,
			"success_rate":      successRate,
			"avg_duration_ms":   avgHTTPDuration.Milliseconds(),
			"requests_per_second": requestsPerSecond,
		},
		"database": map[string]interface{}{
			"queries_total":     mc.dbQueriesTotal,
			"avg_duration_ms":  avgDBDuration.Milliseconds(),
			"connections_active": mc.dbConnectionsActive,
			"connections_idle":  mc.dbConnectionsIdle,
		},
		"system": map[string]interface{}{
			"uptime_seconds":    uptime,
			"start_time":        mc.startTime.Format(time.RFC3339),
		},
	}
}

// Reset сбрасывает метрики
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.httpRequestsTotal = 0
	mc.httpRequestsSuccess = 0
	mc.httpRequestsError = 0
	mc.dbQueriesTotal = 0
	mc.dbConnectionsActive = 0
	mc.dbConnectionsIdle = 0
	mc.lastResetTime = time.Now()

	mc.httpRequestDurationMu.Lock()
	mc.httpRequestDuration = nil
	mc.httpRequestDurationMu.Unlock()

	mc.dbQueriesDurationMu.Lock()
	mc.dbQueriesDuration = nil
	mc.dbQueriesDurationMu.Unlock()
}


