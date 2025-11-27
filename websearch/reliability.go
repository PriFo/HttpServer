package websearch

import (
	"fmt"
	"time"

	"httpserver/internal/infrastructure/persistence"
)

// ReliabilityManager управляет статистикой надежности провайдеров
// Поддерживает работу с БД через репозиторий (опционально)
type ReliabilityManager struct {
	repo  *persistence.WebSearchRepository
	stats map[string]*ProviderStats
}

// NewReliabilityManager создает новый менеджер надежности
// Если репозиторий передан, загружает статистику из БД
func NewReliabilityManager(repo *persistence.WebSearchRepository) (*ReliabilityManager, error) {
	rm := &ReliabilityManager{
		repo:  repo,
		stats: make(map[string]*ProviderStats),
	}

	// Если репозиторий доступен, загружаем статистику из БД
	if repo != nil {
		_ = rm.loadStats() // Игнорируем ошибки при загрузке статистики
	}

	return rm, nil
}

// loadStats загружает статистику из базы данных
func (rm *ReliabilityManager) loadStats() error {
	if rm.repo == nil {
		return nil
	}

	// Получаем всех провайдеров из репозитория
	providers, err := rm.repo.GetAllProviders()
	if err != nil {
		// Игнорируем ошибку при первом запуске
		return nil
	}

	for _, provider := range providers {
		stats, err := rm.repo.GetProviderStats(provider.Name)
		if err != nil {
			continue
		}

		rm.stats[provider.Name] = &ProviderStats{
			ProviderName:      stats.ProviderName,
			RequestsTotal:     stats.RequestsTotal,
			RequestsSuccess:   stats.RequestsSuccess,
			RequestsFailed:    stats.RequestsFailed,
			FailureRate:       stats.FailureRate,
			AvgResponseTimeMs: stats.AvgResponseTimeMs,
			LastSuccess:       stats.LastSuccess,
			LastFailure:       stats.LastFailure,
			LastError:         stats.LastError,
			UpdatedAt:         stats.UpdatedAt,
		}
	}

	return nil
}

// RecordSuccess записывает успешный запрос
func (rm *ReliabilityManager) RecordSuccess(providerName string, responseTime time.Duration) error {
	return rm.RecordSuccessWithTime(providerName, responseTime)
}

// RecordSuccessWithTime записывает успешный запрос с временем ответа
func (rm *ReliabilityManager) RecordSuccessWithTime(providerName string, responseTime time.Duration) error {
	if rm == nil {
		return nil
	}

	stats := rm.getOrCreateStats(providerName)
	stats.RequestsTotal++
	stats.RequestsSuccess++
	now := time.Now()
	stats.LastSuccess = &now
	stats.UpdatedAt = now

	// Обновляем среднее время ответа
	if stats.RequestsSuccess > 0 {
		avgMs := int64(responseTime.Milliseconds())
		stats.AvgResponseTimeMs = (stats.AvgResponseTimeMs*(stats.RequestsSuccess-1) + avgMs) / stats.RequestsSuccess
	}

	// Обновляем процент ошибок
	if stats.RequestsTotal > 0 {
		stats.FailureRate = float64(stats.RequestsFailed) / float64(stats.RequestsTotal)
	}

	// Асинхронно обновляем в базе данных, если репозиторий доступен
	if rm.repo != nil {
		go rm.updateStatsInDB(stats)
	}

	return nil
}

// RecordFailure записывает неуспешный запрос
func (rm *ReliabilityManager) RecordFailure(providerName string) error {
	return rm.RecordFailureWithError(providerName, nil)
}

// RecordFailureWithError записывает неуспешный запрос с ошибкой
func (rm *ReliabilityManager) RecordFailureWithError(providerName string, err error) error {
	if rm == nil {
		return nil
	}

	stats := rm.getOrCreateStats(providerName)
	stats.RequestsTotal++
	stats.RequestsFailed++
	now := time.Now()
	stats.LastFailure = &now
	stats.UpdatedAt = now

	if err != nil {
		stats.LastError = err.Error()
	}

	// Обновляем процент ошибок
	if stats.RequestsTotal > 0 {
		stats.FailureRate = float64(stats.RequestsFailed) / float64(stats.RequestsTotal)
	}

	// Асинхронно обновляем в базе данных, если репозиторий доступен
	if rm.repo != nil {
		go rm.updateStatsInDB(stats)
	}

	return nil
}

// getOrCreateStats получает или создает статистику для провайдера
func (rm *ReliabilityManager) getOrCreateStats(providerName string) *ProviderStats {
	if stats, exists := rm.stats[providerName]; exists {
		return stats
	}

	stats := &ProviderStats{
		ProviderName: providerName,
		UpdatedAt:    time.Now(),
	}
	rm.stats[providerName] = stats
	return stats
}

// GetStats возвращает статистику провайдера
func (rm *ReliabilityManager) GetStats(providerName string) *ProviderStats {
	if rm == nil {
		return nil
	}
	return rm.getOrCreateStats(providerName)
}

// GetAllStats возвращает всю статистику
func (rm *ReliabilityManager) GetAllStats() map[string]*ProviderStats {
	if rm == nil {
		return make(map[string]*ProviderStats)
	}
	return rm.stats
}

// GetWeight вычисляет эффективный вес провайдера на основе его надежности
func (rm *ReliabilityManager) GetWeight(providerName string, basePriority int) float64 {
	stat := rm.getOrCreateStats(providerName)
	if stat.RequestsTotal == 0 {
		return float64(basePriority)
	}

	// Провайдеры с failure_rate >= 0.9 временно исключаются
	if stat.FailureRate >= 0.9 {
		return 0.0
	}

	// Вес = базовый_приоритет * (1 - failure_rate)
	weight := float64(basePriority) * (1.0 - stat.FailureRate)
	if weight < 0 {
		weight = 0
	}

	return weight
}

// updateStatsInDB обновляет статистику в базе данных
func (rm *ReliabilityManager) updateStatsInDB(stat *ProviderStats) {
	if rm.repo == nil {
		return
	}

	dbStats := &persistence.ProviderStats{
		ProviderName:      stat.ProviderName,
		RequestsTotal:     stat.RequestsTotal,
		RequestsSuccess:   stat.RequestsSuccess,
		RequestsFailed:    stat.RequestsFailed,
		FailureRate:       stat.FailureRate,
		AvgResponseTimeMs: stat.AvgResponseTimeMs,
		LastSuccess:       stat.LastSuccess,
		LastFailure:       stat.LastFailure,
		LastError:         stat.LastError,
		UpdatedAt:         stat.UpdatedAt,
	}

	if err := rm.repo.UpdateProviderStats(dbStats); err != nil {
		// Логируем ошибку, но не блокируем основной поток
		fmt.Printf("Failed to update stats for provider %s: %v\n", stat.ProviderName, err)
	}
}
