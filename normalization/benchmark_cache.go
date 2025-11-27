package normalization

import (
	"sync"
	"time"

	"httpserver/database"
)

const (
	// BenchmarkCacheTTL время жизни кэша эталонов
	BenchmarkCacheTTL = 5 * time.Minute
	// BenchmarkCacheMaxSize максимальный размер кэша
	BenchmarkCacheMaxSize = 1000
)

// BenchmarkCacheEntry запись в кэше эталонов
type BenchmarkCacheEntry struct {
	Benchmark *database.ClientBenchmark
	ExpiresAt time.Time
}

// BenchmarkCache кэш эталонов для уменьшения запросов к БД
type BenchmarkCache struct {
	cache map[string]*BenchmarkCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewBenchmarkCache создает новый кэш эталонов
func NewBenchmarkCache() *BenchmarkCache {
	return &BenchmarkCache{
		cache: make(map[string]*BenchmarkCacheEntry),
		ttl:   BenchmarkCacheTTL,
	}
}

// Get получает эталон из кэша по taxID
func (bc *BenchmarkCache) Get(taxID string) *database.ClientBenchmark {
	if taxID == "" {
		return nil
	}

	bc.mu.RLock()
	defer bc.mu.RUnlock()

	entry, exists := bc.cache[taxID]
	if !exists {
		return nil
	}

	// Проверяем срок действия
	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	return entry.Benchmark
}

// Set сохраняет эталон в кэш
func (bc *BenchmarkCache) Set(taxID string, benchmark *database.ClientBenchmark) {
	if taxID == "" {
		return
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Очищаем просроченные записи, если кэш слишком большой
	if len(bc.cache) >= BenchmarkCacheMaxSize {
		bc.cleanExpired()
	}

	bc.cache[taxID] = &BenchmarkCacheEntry{
		Benchmark: benchmark,
		ExpiresAt: time.Now().Add(bc.ttl),
	}
}

// cleanExpired удаляет просроченные записи из кэша
func (bc *BenchmarkCache) cleanExpired() {
	now := time.Now()
	for key, entry := range bc.cache {
		if now.After(entry.ExpiresAt) {
			delete(bc.cache, key)
		}
	}
}

// Clear очищает весь кэш
func (bc *BenchmarkCache) Clear() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.cache = make(map[string]*BenchmarkCacheEntry)
}

// Size возвращает текущий размер кэша
func (bc *BenchmarkCache) Size() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.cache)
}
