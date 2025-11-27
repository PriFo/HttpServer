package services

import (
	"log"
	"time"

	"httpserver/database"
	"httpserver/normalization"
)

// MonitoringService сервис для работы с мониторингом
type MonitoringService struct {
	db         *database.DB
	normalizer NormalizerInterface
	startTime  time.Time
}

// NormalizerInterface интерфейс для доступа к нормализатору
type NormalizerInterface interface {
	GetAINormalizer() AINormalizerInterface
}

// AINormalizerInterface интерфейс для доступа к AI нормализатору
type AINormalizerInterface interface {
	GetStatsCollector() *normalization.StatsCollector
	GetCacheStats() normalization.CacheStats
}

// NewMonitoringService создает новый сервис мониторинга
func NewMonitoringService(db *database.DB, normalizer NormalizerInterface, startTime time.Time) *MonitoringService {
	return &MonitoringService{
		db:         db,
		normalizer: normalizer,
		startTime:  startTime,
	}
}

// GetMetrics возвращает общие метрики мониторинга
func (ms *MonitoringService) GetMetrics(getCircuitBreakerState func() map[string]interface{}, getBatchProcessorStats func() map[string]interface{}, getCheckpointStatus func() map[string]interface{}) (map[string]interface{}, error) {
	// Получаем реальную статистику от нормализатора
	var statsCollector *normalization.StatsCollector
	var cacheStats normalization.CacheStats
	hasCacheStats := false

	if ms.normalizer != nil && ms.normalizer.GetAINormalizer() != nil {
		statsCollector = ms.normalizer.GetAINormalizer().GetStatsCollector()
		cacheStats = ms.normalizer.GetAINormalizer().GetCacheStats()
		hasCacheStats = true
	}

	// Получаем статистику качества из БД
	qualityStatsMap, err := ms.db.GetQualityStats()
	if err != nil {
		log.Printf("Error getting quality stats: %v", err)
		qualityStatsMap = make(map[string]interface{})
	}

	// Извлекаем значения из карты БД
	dbTotalNormalized := int64(0)
	dbBasicNormalized := int64(0)
	dbAIEnhanced := int64(0)
	dbBenchmarkQuality := int64(0)
	dbAverageQualityScore := 0.0

	if total, ok := qualityStatsMap["total_items"].(int); ok {
		dbTotalNormalized = int64(total)
	}
	if avg, ok := qualityStatsMap["average_quality"].(float64); ok {
		dbAverageQualityScore = avg
	}
	if benchmark, ok := qualityStatsMap["benchmark_count"].(int); ok {
		dbBenchmarkQuality = int64(benchmark)
	}
	if byLevel, ok := qualityStatsMap["by_level"].(map[string]map[string]interface{}); ok {
		if basicStats, ok := byLevel["basic"]; ok {
			if count, ok := basicStats["count"].(int); ok {
				dbBasicNormalized = int64(count)
			}
		}
		if aiStats, ok := byLevel["ai_enhanced"]; ok {
			if count, ok := aiStats["count"].(int); ok {
				dbAIEnhanced = int64(count)
			}
		}
	}

	// Используем метрики из StatsCollector как основной источник, БД как fallback
	totalNormalized := dbTotalNormalized
	basicNormalized := dbBasicNormalized
	aiEnhanced := dbAIEnhanced
	benchmarkQuality := dbBenchmarkQuality
	averageQualityScore := dbAverageQualityScore

	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		if perfMetrics.TotalNormalized > 0 {
			totalNormalized = perfMetrics.TotalNormalized
			basicNormalized = perfMetrics.BasicNormalized
			aiEnhanced = perfMetrics.AIEnhanced
			benchmarkQuality = perfMetrics.BenchmarkQuality
			if perfMetrics.AverageQualityScore > 0 {
				averageQualityScore = perfMetrics.AverageQualityScore
			} else if dbAverageQualityScore > 0 {
				averageQualityScore = dbAverageQualityScore
			}
		}
	}

	// Рассчитываем uptime
	uptime := time.Since(ms.startTime).Seconds()

	// Рассчитываем throughput
	throughput := 0.0
	if uptime > 0 && totalNormalized > 0 {
		throughput = float64(totalNormalized) / uptime
	}

	// Формируем ответ
	summary := map[string]interface{}{
		"uptime_seconds":              uptime,
		"throughput_items_per_second": throughput,
		"ai": map[string]interface{}{
			"total_requests":     0,
			"successful":         0,
			"failed":             0,
			"success_rate":       0.0,
			"average_latency_ms": 0.0,
		},
		"cache": map[string]interface{}{
			"hits":            0,
			"misses":          0,
			"hit_rate":        0.0,
			"size":            0,
			"memory_usage_kb": 0.0,
		},
		"quality": map[string]interface{}{
			"total_normalized":      totalNormalized,
			"basic":                 basicNormalized,
			"ai_enhanced":           aiEnhanced,
			"benchmark":             benchmarkQuality,
			"average_quality_score": averageQualityScore,
		},
	}

	// Добавляем реальные AI метрики если доступны
	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		successRate := 0.0
		if perfMetrics.TotalAIRequests > 0 {
			successRate = float64(perfMetrics.SuccessfulAIRequest) / float64(perfMetrics.TotalAIRequests)
		}
		avgLatencyMs := float64(perfMetrics.AverageAILatency.Milliseconds())

		summary["ai"] = map[string]interface{}{
			"total_requests":     perfMetrics.TotalAIRequests,
			"successful":         perfMetrics.SuccessfulAIRequest,
			"failed":             perfMetrics.FailedAIRequests,
			"success_rate":       successRate,
			"average_latency_ms": avgLatencyMs,
		}
	}

	// Добавляем реальные cache метрики если доступны
	if hasCacheStats {
		summary["cache"] = map[string]interface{}{
			"hits":            cacheStats.Hits,
			"misses":          cacheStats.Misses,
			"hit_rate":        cacheStats.HitRate,
			"size":            cacheStats.Entries,
			"memory_usage_kb": float64(cacheStats.MemoryUsageB) / 1024.0,
		}
	}

	// Добавляем метрики Circuit Breaker
	if getCircuitBreakerState != nil {
		summary["circuit_breaker"] = getCircuitBreakerState()
	}

	// Добавляем метрики Batch Processor
	if getBatchProcessorStats != nil {
		summary["batch_processor"] = getBatchProcessorStats()
	}

	// Добавляем статус Checkpoint
	if getCheckpointStatus != nil {
		summary["checkpoint"] = getCheckpointStatus()
	}

	return summary, nil
}

// GetCacheStats возвращает статистику кеша
func (ms *MonitoringService) GetCacheStats() (map[string]interface{}, error) {
	var cacheStats normalization.CacheStats
	if ms.normalizer != nil && ms.normalizer.GetAINormalizer() != nil {
		cacheStats = ms.normalizer.GetAINormalizer().GetCacheStats()
	}

	return map[string]interface{}{
		"hits":            cacheStats.Hits,
		"misses":          cacheStats.Misses,
		"hit_rate_pct":    cacheStats.HitRate * 100.0,
		"size":            cacheStats.Entries,
		"memory_usage_kb": float64(cacheStats.MemoryUsageB) / 1024.0,
	}, nil
}

// GetAIStats возвращает статистику AI обработки
func (ms *MonitoringService) GetAIStats() (map[string]interface{}, error) {
	var statsCollector *normalization.StatsCollector
	var cacheStats normalization.CacheStats

	if ms.normalizer != nil && ms.normalizer.GetAINormalizer() != nil {
		statsCollector = ms.normalizer.GetAINormalizer().GetStatsCollector()
		cacheStats = ms.normalizer.GetAINormalizer().GetCacheStats()
	}

	totalCalls := int64(0)
	errors := int64(0)
	avgLatencyMs := 0.0

	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		totalCalls = perfMetrics.TotalAIRequests
		errors = perfMetrics.FailedAIRequests
		avgLatencyMs = float64(perfMetrics.AverageAILatency.Milliseconds())
	}

	cacheHitRate := 0.0
	if cacheStats.Hits+cacheStats.Misses > 0 {
		cacheHitRate = float64(cacheStats.Hits) / float64(cacheStats.Hits+cacheStats.Misses) * 100.0
	}

	return map[string]interface{}{
		"total_calls":    totalCalls,
		"cache_hits":     cacheStats.Hits,
		"cache_misses":   cacheStats.Misses,
		"errors":         errors,
		"avg_latency_ms": avgLatencyMs,
		"cache_hit_rate": cacheHitRate,
	}, nil
}

// GetMetricsHistory возвращает историю метрик за указанный период
func (ms *MonitoringService) GetMetricsHistory(fromTime, toTime *time.Time, metricType string, limit int) ([]database.PerformanceMetricsSnapshot, error) {
	return ms.db.GetMetricsHistory(fromTime, toTime, metricType, limit)
}

// CollectMetricsSnapshot собирает текущий снимок метрик производительности
func (ms *MonitoringService) CollectMetricsSnapshot(
	getCircuitBreakerState func() map[string]interface{},
	getBatchProcessorStats func() map[string]interface{},
	getCheckpointStatus func() map[string]interface{},
) (*database.PerformanceMetricsSnapshot, error) {
	// Рассчитываем uptime
	uptime := time.Since(ms.startTime).Seconds()

	// Получаем метрики от компонентов
	var cbState map[string]interface{}
	var batchStats map[string]interface{}
	var checkpointStatus map[string]interface{}

	if getCircuitBreakerState != nil {
		cbState = getCircuitBreakerState()
	}
	if getBatchProcessorStats != nil {
		batchStats = getBatchProcessorStats()
	}
	if getCheckpointStatus != nil {
		checkpointStatus = getCheckpointStatus()
	}

	// Собираем AI и cache метрики
	aiSuccessRate := 0.0
	cacheHitRate := 0.0
	throughput := 0.0

	if ms.normalizer != nil && ms.normalizer.GetAINormalizer() != nil {
		statsCollector := ms.normalizer.GetAINormalizer().GetStatsCollector()
		if statsCollector != nil {
			perfMetrics := statsCollector.GetMetrics()
			if perfMetrics.TotalAIRequests > 0 {
				aiSuccessRate = float64(perfMetrics.SuccessfulAIRequest) / float64(perfMetrics.TotalAIRequests)
			}
			if perfMetrics.TotalNormalized > 0 && uptime > 0 {
				throughput = float64(perfMetrics.TotalNormalized) / uptime
			}
		}

		cacheStats := ms.normalizer.GetAINormalizer().GetCacheStats()
		cacheHitRate = cacheStats.HitRate
	}

	// Получаем checkpoint progress
	checkpointProgress := 0.0
	if checkpointStatus != nil {
		if progress, ok := checkpointStatus["progress_percent"].(float64); ok {
			checkpointProgress = progress
		}
	}

	// Формируем детальные метрики
	detailedMetrics := map[string]interface{}{
		"uptime_seconds":        int(uptime),
		"throughput":            throughput,
		"ai_success_rate":       aiSuccessRate,
		"cache_hit_rate":        cacheHitRate,
		"checkpoint_progress":   checkpointProgress,
	}

	// Добавляем batch queue size если доступен
	var batchQueueSize int
	if batchStats != nil {
		if queueSize, ok := batchStats["queue_size"].(int); ok {
			batchQueueSize = queueSize
			detailedMetrics["batch_queue_size"] = queueSize
		}
	}

	// Добавляем circuit breaker state
	circuitBreakerState := ""
	if cbState != nil {
		if state, ok := cbState["state"].(string); ok {
			circuitBreakerState = state
			detailedMetrics["circuit_breaker_state"] = state
		}
	}

	// Создаем snapshot
	snapshot := &database.PerformanceMetricsSnapshot{
		Timestamp:           time.Now(),
		UptimeSeconds:       int(uptime),
		Throughput:          throughput,
		AISuccessRate:       aiSuccessRate,
		CacheHitRate:        cacheHitRate,
		BatchQueueSize:      batchQueueSize,
		CircuitBreakerState: circuitBreakerState,
		CheckpointProgress:  checkpointProgress,
	}

	return snapshot, nil
}

