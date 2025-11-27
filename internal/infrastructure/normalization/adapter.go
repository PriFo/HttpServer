package normalization

import (
	"httpserver/normalization"
	"httpserver/server/services"
)

// Adapter адаптер для Normalizer, чтобы соответствовать интерфейсу MonitoringService
type Adapter struct {
	Normalizer *normalization.Normalizer
}

// GetAINormalizer получает AI нормализатор
func (a *Adapter) GetAINormalizer() services.AINormalizerInterface {
	if a.Normalizer == nil {
		return nil
	}
	aiNormalizer := a.Normalizer.GetAINormalizer()
	if aiNormalizer == nil {
		return nil
	}
	return &AINormalizerAdapter{AINormalizer: aiNormalizer}
}

// AINormalizerAdapter адаптер для AINormalizer
type AINormalizerAdapter struct {
	AINormalizer *normalization.AINormalizer
}

// GetStatsCollector получает сборщик статистики
func (a *AINormalizerAdapter) GetStatsCollector() *normalization.StatsCollector {
	if a.AINormalizer == nil {
		return nil
	}
	return a.AINormalizer.GetStatsCollector()
}

// GetCacheStats получает статистику кеша
func (a *AINormalizerAdapter) GetCacheStats() normalization.CacheStats {
	if a.AINormalizer == nil {
		return normalization.CacheStats{}
	}
	return a.AINormalizer.GetCacheStats()
}
