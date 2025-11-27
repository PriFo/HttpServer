package algorithms

import (
	"sync"
)

// OptimizedHybridSimilarity кэшированная версия гибридного алгоритма схожести
// Использует кэш для ускорения повторных вычислений
type OptimizedHybridSimilarity struct {
	cache    map[string]float64
	cacheMu  sync.RWMutex
	weights  *SimilarityWeights
	maxCache int // Максимальный размер кэша
}

// NewOptimizedHybridSimilarity создает новый оптимизированный гибридный матчер
func NewOptimizedHybridSimilarity(weights *SimilarityWeights, maxCache int) *OptimizedHybridSimilarity {
	if weights == nil {
		weights = DefaultSimilarityWeights()
	}
	if maxCache <= 0 {
		maxCache = 10000 // По умолчанию 10000 записей
	}

	return &OptimizedHybridSimilarity{
		cache:    make(map[string]float64),
		weights:  weights,
		maxCache: maxCache,
	}
}

// Similarity вычисляет схожесть с кэшированием
func (ohs *OptimizedHybridSimilarity) Similarity(s1, s2 string) float64 {
	// Создаем ключ кэша (нормализуем порядок строк для симметричности)
	cacheKey := ohs.getCacheKey(s1, s2)

	// Проверяем кэш
	ohs.cacheMu.RLock()
	if cached, ok := ohs.cache[cacheKey]; ok {
		ohs.cacheMu.RUnlock()
		return cached
	}
	ohs.cacheMu.RUnlock()

	// Вычисляем схожесть
	similarity := HybridSimilarityAdvanced(s1, s2, ohs.weights)

	// Сохраняем в кэш (с проверкой размера)
	ohs.cacheMu.Lock()
	if len(ohs.cache) >= ohs.maxCache {
		// Очищаем часть кэша (удаляем 20% старых записей)
		ohs.clearCache(ohs.maxCache / 5)
	}
	ohs.cache[cacheKey] = similarity
	ohs.cacheMu.Unlock()

	return similarity
}

// getCacheKey создает ключ кэша (симметричный)
func (ohs *OptimizedHybridSimilarity) getCacheKey(s1, s2 string) string {
	// Сортируем строки для симметричности ключа
	if s1 > s2 {
		s1, s2 = s2, s1
	}
	return s1 + "|" + s2
}

// clearCache очищает часть кэша
func (ohs *OptimizedHybridSimilarity) clearCache(count int) {
	removed := 0
	for key := range ohs.cache {
		if removed >= count {
			break
		}
		delete(ohs.cache, key)
		removed++
	}
}

// ClearCache очищает весь кэш
func (ohs *OptimizedHybridSimilarity) ClearCache() {
	ohs.cacheMu.Lock()
	ohs.cache = make(map[string]float64)
	ohs.cacheMu.Unlock()
}

// GetCacheSize возвращает текущий размер кэша
func (ohs *OptimizedHybridSimilarity) GetCacheSize() int {
	ohs.cacheMu.RLock()
	defer ohs.cacheMu.RUnlock()
	return len(ohs.cache)
}

// SetWeights устанавливает новые веса
func (ohs *OptimizedHybridSimilarity) SetWeights(weights *SimilarityWeights) {
	ohs.cacheMu.Lock()
	ohs.weights = weights
	// Очищаем кэш при изменении весов
	ohs.cache = make(map[string]float64)
	ohs.cacheMu.Unlock()
}

// BatchSimilarity вычисляет схожесть для множества пар одновременно
// Оптимизировано для пакетной обработки
func (ohs *OptimizedHybridSimilarity) BatchSimilarity(pairs []SimilarityPair) []float64 {
	results := make([]float64, len(pairs))
	
	// Обрабатываем параллельно
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Ограничиваем параллелизм
	
	for i, pair := range pairs {
		wg.Add(1)
		go func(idx int, p SimilarityPair) {
			defer wg.Done()
			semaphore <- struct{}{} // Захватываем слот
			results[idx] = ohs.Similarity(p.S1, p.S2)
			<-semaphore // Освобождаем слот
		}(i, pair)
	}
	
	wg.Wait()
	return results
}

// SimilarityPair пара строк для сравнения
type SimilarityPair struct {
	S1 string
	S2 string
}

// PerformanceStats статистика производительности
type PerformanceStats struct {
	CacheHits   int64
	CacheMisses int64
	TotalCalls  int64
}

// GetStats возвращает статистику производительности
func (ohs *OptimizedHybridSimilarity) GetStats() PerformanceStats {
	// В реальной реализации нужно добавить счетчики
	return PerformanceStats{
		CacheHits:   0,
		CacheMisses: 0,
		TotalCalls:  0,
	}
}

