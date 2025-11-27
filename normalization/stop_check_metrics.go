package normalization

import (
	"sync"
	"sync/atomic"
	"time"
)

// StopCheckMetrics метрики производительности для проверок остановки
type StopCheckMetrics struct {
	TotalChecks       int64         `json:"total_checks"`          // Общее количество проверок
	TotalCheckTime    time.Duration `json:"total_check_time_ms"`   // Общее время на проверки
	AverageCheckTime  time.Duration `json:"average_check_time_ms"` // Среднее время проверки
	MaxCheckTime      time.Duration `json:"max_check_time_ms"`     // Максимальное время проверки
	MinCheckTime      time.Duration `json:"min_check_time_ms"`     // Минимальное время проверки
	StopDetectedCount int64         `json:"stop_detected_count"`   // Количество обнаруженных остановок
	ChecksBeforeStop  int64         `json:"checks_before_stop"`    // Количество проверок до остановки
	mu                sync.RWMutex
}

// stopCheckMetrics глобальные метрики проверок остановки
var stopCheckMetrics = &StopCheckMetrics{
	MinCheckTime: time.Hour, // Инициализируем большим значением
}

// recordStopCheck записывает метрику проверки остановки
func recordStopCheck(duration time.Duration, stopDetected bool) {
	stopCheckMetrics.mu.Lock()
	defer stopCheckMetrics.mu.Unlock()

	stopCheckMetrics.TotalChecks++
	stopCheckMetrics.TotalCheckTime += duration

	if duration > stopCheckMetrics.MaxCheckTime {
		stopCheckMetrics.MaxCheckTime = duration
	}
	if duration < stopCheckMetrics.MinCheckTime {
		stopCheckMetrics.MinCheckTime = duration
	}

	if stopCheckMetrics.TotalChecks > 0 {
		stopCheckMetrics.AverageCheckTime = stopCheckMetrics.TotalCheckTime / time.Duration(stopCheckMetrics.TotalChecks)
	}

	if stopDetected {
		atomic.AddInt64(&stopCheckMetrics.StopDetectedCount, 1)
		atomic.StoreInt64(&stopCheckMetrics.ChecksBeforeStop, stopCheckMetrics.TotalChecks)
	}
}

// GetStopCheckMetrics возвращает текущие метрики проверок остановки
func GetStopCheckMetrics() *StopCheckMetrics {
	stopCheckMetrics.mu.RLock()
	defer stopCheckMetrics.mu.RUnlock()

	// Создаем копию для безопасного возврата
	return &StopCheckMetrics{
		TotalChecks:       stopCheckMetrics.TotalChecks,
		TotalCheckTime:    stopCheckMetrics.TotalCheckTime,
		AverageCheckTime:  stopCheckMetrics.AverageCheckTime,
		MaxCheckTime:      stopCheckMetrics.MaxCheckTime,
		MinCheckTime:      stopCheckMetrics.MinCheckTime,
		StopDetectedCount: atomic.LoadInt64(&stopCheckMetrics.StopDetectedCount),
		ChecksBeforeStop:  atomic.LoadInt64(&stopCheckMetrics.ChecksBeforeStop),
	}
}

// ResetStopCheckMetrics сбрасывает метрики проверок остановки
func ResetStopCheckMetrics() {
	stopCheckMetrics.mu.Lock()
	defer stopCheckMetrics.mu.Unlock()

	stopCheckMetrics.TotalChecks = 0
	stopCheckMetrics.TotalCheckTime = 0
	stopCheckMetrics.AverageCheckTime = 0
	stopCheckMetrics.MaxCheckTime = 0
	stopCheckMetrics.MinCheckTime = time.Hour
	atomic.StoreInt64(&stopCheckMetrics.StopDetectedCount, 0)
	atomic.StoreInt64(&stopCheckMetrics.ChecksBeforeStop, 0)
}

// GetStopCheckStats возвращает статистику проверок остановки в виде map
func GetStopCheckStats() map[string]interface{} {
	metrics := GetStopCheckMetrics()

	stats := map[string]interface{}{
		"total_checks":          metrics.TotalChecks,
		"total_check_time_ms":   metrics.TotalCheckTime.Milliseconds(),
		"average_check_time_ms": metrics.AverageCheckTime.Milliseconds(),
		"max_check_time_ms":     metrics.MaxCheckTime.Milliseconds(),
		"min_check_time_ms":     metrics.MinCheckTime.Milliseconds(),
		"stop_detected_count":   metrics.StopDetectedCount,
		"checks_before_stop":    metrics.ChecksBeforeStop,
	}

	if metrics.TotalChecks > 0 {
		stats["stop_detection_rate"] = float64(metrics.StopDetectedCount) / float64(metrics.TotalChecks) * 100
	} else {
		stats["stop_detection_rate"] = 0.0
	}

	return stats
}
