package services

import (
	"testing"
	"time"

	"httpserver/normalization"
)

// mockNormalizer мок для нормализатора
type mockNormalizer struct {
	aiNormalizer *mockAINormalizer
}

func (m *mockNormalizer) GetAINormalizer() AINormalizerInterface {
	return m.aiNormalizer
}

// mockAINormalizer мок для AI нормализатора
type mockAINormalizer struct {
	statsCollector *normalization.StatsCollector
	cacheStats     normalization.CacheStats
}

func (m *mockAINormalizer) GetStatsCollector() *normalization.StatsCollector {
	return m.statsCollector
}

func (m *mockAINormalizer) GetCacheStats() normalization.CacheStats {
	return m.cacheStats
}

// TestNewMonitoringService проверяет создание нового сервиса мониторинга
func TestNewMonitoringService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	normalizer := &mockNormalizer{}
	startTime := time.Now()

	service := NewMonitoringService(db, normalizer, startTime)
	if service == nil {
		t.Error("NewMonitoringService() should not return nil")
	}
}

// TestMonitoringService_GetMetrics проверяет получение метрик
func TestMonitoringService_GetMetrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	statsCollector := normalization.NewStatsCollector()
	aiNormalizer := &mockAINormalizer{
		statsCollector: statsCollector,
		cacheStats:     normalization.CacheStats{},
	}
	normalizer := &mockNormalizer{
		aiNormalizer: aiNormalizer,
	}
	startTime := time.Now()

	service := NewMonitoringService(db, normalizer, startTime)

	getCircuitBreakerState := func() map[string]interface{} {
		return map[string]interface{}{"state": "closed"}
	}

	getBatchProcessorStats := func() map[string]interface{} {
		return map[string]interface{}{"processed": 100}
	}

	getCheckpointStatus := func() map[string]interface{} {
		return map[string]interface{}{"status": "ok"}
	}

	metrics, err := service.GetMetrics(getCircuitBreakerState, getBatchProcessorStats, getCheckpointStatus)
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}
}

// TestMonitoringService_GetMetrics_NilNormalizer проверяет получение метрик без нормализатора
func TestMonitoringService_GetMetrics_NilNormalizer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	startTime := time.Now()

	service := NewMonitoringService(db, nil, startTime)

	getCircuitBreakerState := func() map[string]interface{} {
		return map[string]interface{}{"state": "closed"}
	}

	getBatchProcessorStats := func() map[string]interface{} {
		return map[string]interface{}{"processed": 100}
	}

	getCheckpointStatus := func() map[string]interface{} {
		return map[string]interface{}{"status": "ok"}
	}

	metrics, err := service.GetMetrics(getCircuitBreakerState, getBatchProcessorStats, getCheckpointStatus)
	if err != nil {
		t.Fatalf("GetMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Error("Expected non-nil metrics even without normalizer")
	}
}

