package websearch

import (
	"time"
)

// StubReliabilityManager простая реализация ReliabilityManagerInterface без БД
// Используется когда репозиторий недоступен
type StubReliabilityManager struct {
	stats map[string]*ProviderStats
}

// NewStubReliabilityManager создает stub менеджер надежности
func NewStubReliabilityManager() ReliabilityManagerInterface {
	return &StubReliabilityManager{
		stats: make(map[string]*ProviderStats),
	}
}

// RecordSuccess записывает успешный запрос
func (s *StubReliabilityManager) RecordSuccess(providerName string, responseTime time.Duration) error {
	return s.RecordSuccessWithTime(providerName, responseTime)
}

// RecordSuccessWithTime записывает успешный запрос с временем ответа
func (s *StubReliabilityManager) RecordSuccessWithTime(providerName string, responseTime time.Duration) error {
	stats := s.getOrCreateStats(providerName)
	stats.RequestsTotal++
	stats.RequestsSuccess++
	stats.AvgResponseTimeMs = (stats.AvgResponseTimeMs*int64(stats.RequestsSuccess-1) + responseTime.Milliseconds()) / int64(stats.RequestsSuccess)
	now := time.Now()
	stats.LastSuccess = &now
	stats.UpdatedAt = now
	if stats.RequestsTotal > 0 {
		stats.FailureRate = float64(stats.RequestsFailed) / float64(stats.RequestsTotal)
	}
	return nil
}

// RecordFailure записывает неуспешный запрос без ошибки
func (s *StubReliabilityManager) RecordFailure(providerName string) error {
	return s.RecordFailureWithError(providerName, nil)
}

// RecordFailureWithError записывает неуспешный запрос с ошибкой
func (s *StubReliabilityManager) RecordFailureWithError(providerName string, err error) error {
	stats := s.getOrCreateStats(providerName)
	stats.RequestsTotal++
	stats.RequestsFailed++
	now := time.Now()
	stats.LastFailure = &now
	stats.UpdatedAt = now
	if err != nil {
		stats.LastError = err.Error()
	}
	if stats.RequestsTotal > 0 {
		stats.FailureRate = float64(stats.RequestsFailed) / float64(stats.RequestsTotal)
	}
	return nil
}

// GetStats возвращает статистику провайдера
func (s *StubReliabilityManager) GetStats(providerName string) *ProviderStats {
	return s.getOrCreateStats(providerName)
}

// GetAllStats возвращает статистику всех провайдеров
func (s *StubReliabilityManager) GetAllStats() map[string]*ProviderStats {
	result := make(map[string]*ProviderStats)
	for k, v := range s.stats {
		// Создаем копию, чтобы избежать изменений извне
		statsCopy := *v
		result[k] = &statsCopy
	}
	return result
}

// GetWeight возвращает вес провайдера для выбора
func (s *StubReliabilityManager) GetWeight(providerName string, basePriority int) float64 {
	stats := s.getOrCreateStats(providerName)
	if stats.RequestsTotal == 0 {
		return float64(basePriority)
	}
	// Вес = базовый приоритет * (1 - failure rate)
	return float64(basePriority) * (1.0 - stats.FailureRate)
}

// getOrCreateStats получает или создает статистику провайдера
func (s *StubReliabilityManager) getOrCreateStats(providerName string) *ProviderStats {
	if stats, exists := s.stats[providerName]; exists {
		return stats
	}
	stats := &ProviderStats{
		ProviderName:      providerName,
		RequestsTotal:    0,
		RequestsSuccess:   0,
		RequestsFailed:    0,
		FailureRate:       0.0,
		AvgResponseTimeMs: 0,
		UpdatedAt:         time.Now(),
	}
	s.stats[providerName] = stats
	return stats
}

