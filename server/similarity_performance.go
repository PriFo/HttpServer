package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/similarity/ для организации,
// но остается в пакете server для доступа к методам Server
// TODO:legacy-migration revisit dependencies after handler extraction

import (
	"net/http"
	"sync"
	"time"
)

// PerformanceMetrics метрики производительности для алгоритмов схожести
type PerformanceMetrics struct {
	TotalRequests     int64         `json:"total_requests"`
	TotalTime         time.Duration `json:"total_time_ms"`
	AverageTime       float64       `json:"average_time_ms"`
	CacheHits         int64         `json:"cache_hits"`
	CacheMisses       int64         `json:"cache_misses"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	BatchRequests     int64         `json:"batch_requests"`
	TotalPairs        int64         `json:"total_pairs"`
	mu                sync.RWMutex
}

// similarityPerformance глобальные метрики производительности
var similarityPerformance = &PerformanceMetrics{}

// recordRequest записывает метрику запроса
func (pm *PerformanceMetrics) recordRequest(duration time.Duration, cacheHit bool, pairCount int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.TotalRequests++
	pm.TotalTime += duration
	if cacheHit {
		pm.CacheHits++
	} else {
		pm.CacheMisses++
	}
	if pairCount > 1 {
		pm.BatchRequests++
		pm.TotalPairs += int64(pairCount)
	}
}

// GetStats возвращает текущие метрики
func (pm *PerformanceMetrics) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	avgTime := 0.0
	if pm.TotalRequests > 0 {
		avgTime = float64(pm.TotalTime.Milliseconds()) / float64(pm.TotalRequests)
	}

	hitRate := 0.0
	totalCacheRequests := pm.CacheHits + pm.CacheMisses
	if totalCacheRequests > 0 {
		hitRate = float64(pm.CacheHits) / float64(totalCacheRequests) * 100
	}

	return map[string]interface{}{
		"total_requests":    pm.TotalRequests,
		"total_time_ms":     pm.TotalTime.Milliseconds(),
		"average_time_ms":   avgTime,
		"cache_hits":        pm.CacheHits,
		"cache_misses":      pm.CacheMisses,
		"cache_hit_rate":    hitRate,
		"batch_requests":    pm.BatchRequests,
		"total_pairs":       pm.TotalPairs,
		"average_pairs_per_batch": func() float64 {
			if pm.BatchRequests > 0 {
				return float64(pm.TotalPairs) / float64(pm.BatchRequests)
			}
			return 0
		}(),
	}
}

// Reset сбрасывает метрики
func (pm *PerformanceMetrics) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.TotalRequests = 0
	pm.TotalTime = 0
	pm.CacheHits = 0
	pm.CacheMisses = 0
	pm.BatchRequests = 0
	pm.TotalPairs = 0
}

// handleSimilarityPerformance получает метрики производительности
// GET /api/similarity/performance
func (s *Server) handleSimilarityPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := similarityPerformance.GetStats()

	// Добавляем информацию о кэше
	if s.similarityCache != nil {
		s.similarityCacheMutex.RLock()
		stats["cache_size"] = s.similarityCache.GetCacheSize()
		s.similarityCacheMutex.RUnlock()
	} else {
		stats["cache_size"] = 0
	}

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleSimilarityPerformanceReset сбрасывает метрики производительности
// POST /api/similarity/performance/reset
func (s *Server) handleSimilarityPerformanceReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	similarityPerformance.Reset()

	s.writeJSONResponse(w, r, map[string]interface{}{
		"message": "Performance metrics reset successfully",
	}, http.StatusOK)
}

// measureSimilarity измеряет время выполнения и записывает метрики
func (s *Server) measureSimilarity(fn func() float64, cacheHit bool, pairCount int) float64 {
	start := time.Now()
	result := fn()
	duration := time.Since(start)
	similarityPerformance.recordRequest(duration, cacheHit, pairCount)
	return result
}

