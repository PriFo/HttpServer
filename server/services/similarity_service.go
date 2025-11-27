package services

import (
	"sync"

	"httpserver/normalization/algorithms"
	apperrors "httpserver/server/errors"
)

// SimilarityService сервис для работы с алгоритмами схожести
type SimilarityService struct {
	similarityCache      *algorithms.OptimizedHybridSimilarity
	similarityCacheMutex sync.RWMutex
}

// NewSimilarityService создает новый сервис схожести
func NewSimilarityService(similarityCache *algorithms.OptimizedHybridSimilarity) *SimilarityService {
	return &SimilarityService{
		similarityCache: similarityCache,
	}
}

// Compare сравнивает две строки используя различные алгоритмы
func (ss *SimilarityService) Compare(string1, string2 string, weights *algorithms.SimilarityWeights) (map[string]interface{}, error) {
	if err := algorithms.ValidatePair(string1, string2); err != nil {
		return nil, apperrors.NewValidationError("ошибка валидации пары", err)
	}

	// Используем веса по умолчанию, если не указаны
	if weights == nil {
		weights = algorithms.DefaultSimilarityWeights()
	}

	// Валидируем веса
	if err := algorithms.ValidateWeights(weights); err != nil {
		return nil, apperrors.NewValidationError("ошибка валидации весов", err)
	}

	// Используем глобальный кэш, если доступен
	var hybridSimilarity float64
	if ss.similarityCache != nil {
		ss.similarityCacheMutex.RLock()
		if weights != nil {
			ss.similarityCache.SetWeights(weights)
		}
		hybridSimilarity = ss.similarityCache.Similarity(string1, string2)
		ss.similarityCacheMutex.RUnlock()
	} else {
		hybridSimilarity = algorithms.HybridSimilarityAdvanced(string1, string2, weights)
	}

	// Вычисляем схожесть различными методами
	results := make(map[string]interface{})

	// Гибридный метод (уже вычислен выше)
	results["hybrid"] = hybridSimilarity

	// Отдельные алгоритмы
	results["jaro_winkler"] = algorithms.JaroWinklerSimilarityAdvanced(string1, string2)
	results["lcs"] = algorithms.LCSSimilarityAdvanced(string1, string2)
	results["ngram_bigram"] = algorithms.NgramSimilarityAdvanced(string1, string2, 2)
	results["ngram_trigram"] = algorithms.NgramSimilarityAdvanced(string1, string2, 3)

	// Фонетические алгоритмы
	phoneticMatcher := algorithms.NewPhoneticMatcher()
	results["phonetic"] = phoneticMatcher.Similarity(string1, string2)
	results["phonetic_soundex"] = phoneticMatcher.EncodeSoundex(string1) == phoneticMatcher.EncodeSoundex(string2)
	results["phonetic_metaphone"] = phoneticMatcher.EncodeMetaphone(string1) == phoneticMatcher.EncodeMetaphone(string2)

	// Jaccard
	metrics := algorithms.NewSimilarityMetrics()
	results["jaccard"] = metrics.JaccardIndex(string1, string2)

	// Детальная информация о весах
	results["weights"] = weights

	return map[string]interface{}{
		"string1": string1,
		"string2": string2,
		"results": results,
	}, nil
}

// BatchCompare сравнивает множество пар строк
func (ss *SimilarityService) BatchCompare(pairs []algorithms.SimilarityPair, weights *algorithms.SimilarityWeights) ([]map[string]interface{}, int, error) {
	if len(pairs) == 0 {
		return nil, 0, apperrors.NewValidationError("массив пар обязателен и не может быть пустым", nil)
	}

	if len(pairs) > 1000 {
		return nil, 0, apperrors.NewValidationError("максимум 1000 пар разрешено в запросе", nil)
	}

	// Используем веса по умолчанию, если не указаны
	if weights == nil {
		weights = algorithms.DefaultSimilarityWeights()
	}

	// Используем глобальный кэш для пакетной обработки
	var results []float64
	var cacheSize int
	ss.similarityCacheMutex.Lock()
	if ss.similarityCache != nil {
		if weights != nil {
			ss.similarityCache.SetWeights(weights)
		}
		results = ss.similarityCache.BatchSimilarity(pairs)
		cacheSize = ss.similarityCache.GetCacheSize()
		ss.similarityCacheMutex.Unlock()
	} else {
		ss.similarityCacheMutex.Unlock()
		// Создаем временный экземпляр для пакетной обработки
		ohs := algorithms.NewOptimizedHybridSimilarity(weights, 10000)
		results = ohs.BatchSimilarity(pairs)
		cacheSize = ohs.GetCacheSize()
	}

	// Формируем ответ
	response := make([]map[string]interface{}, len(pairs))
	for i, pair := range pairs {
		response[i] = map[string]interface{}{
			"string1":    pair.S1,
			"string2":    pair.S2,
			"similarity": results[i],
		}
	}

	return response, cacheSize, nil
}

// GetDefaultWeights возвращает веса по умолчанию
func (ss *SimilarityService) GetDefaultWeights() *algorithms.SimilarityWeights {
	return algorithms.DefaultSimilarityWeights()
}

// SetWeights устанавливает пользовательские веса
func (ss *SimilarityService) SetWeights(weights *algorithms.SimilarityWeights) error {
	if weights == nil {
		return apperrors.NewValidationError("веса обязательны", nil)
	}

	// Нормализуем веса
	weights.NormalizeWeights()

	// Валидация весов
	if weights.JaroWinkler < 0 || weights.JaroWinkler > 1 ||
		weights.LCS < 0 || weights.LCS > 1 ||
		weights.Phonetic < 0 || weights.Phonetic > 1 ||
		weights.Ngram < 0 || weights.Ngram > 1 ||
		weights.Jaccard < 0 || weights.Jaccard > 1 {
		return apperrors.NewValidationError("веса должны быть между 0 и 1", nil)
	}

	// Устанавливаем веса в кэш
	if ss.similarityCache != nil {
		ss.similarityCacheMutex.Lock()
		ss.similarityCache.SetWeights(weights)
		ss.similarityCacheMutex.Unlock()
	}

	return nil
}

// ClearCache очищает кэш схожести
func (ss *SimilarityService) ClearCache() int {
	if ss.similarityCache == nil {
		return 0
	}

	ss.similarityCacheMutex.Lock()
	defer ss.similarityCacheMutex.Unlock()

	cacheSize := ss.similarityCache.GetCacheSize()
	ss.similarityCache.ClearCache()
	return cacheSize
}

// GetCacheStats возвращает статистику кэша
func (ss *SimilarityService) GetCacheStats() map[string]interface{} {
	if ss.similarityCache == nil {
		return map[string]interface{}{
			"cache_size":    0,
			"cache_enabled": false,
		}
	}

	ss.similarityCacheMutex.RLock()
	defer ss.similarityCacheMutex.RUnlock()

	return map[string]interface{}{
		"cache_size":    ss.similarityCache.GetCacheSize(),
		"cache_enabled": true,
	}
}
