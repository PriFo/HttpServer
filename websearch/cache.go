package websearch

import (
	"sync"
	"time"
)

// CacheConfig конфигурация кэша
type CacheConfig struct {
	Enabled         bool          `json:"enabled"`
	TTL             time.Duration `json:"ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxSize         int           `json:"max_size"`
}

// CacheEntry запись в кэше
type CacheEntry struct {
	Result      *SearchResult
	Expiration  time.Time
	AccessCount int64
}

// Cache кэш для результатов веб-поиска
type Cache struct {
	config *CacheConfig
	data   map[string]*CacheEntry
	mutex  sync.RWMutex
	stats  *CacheStats
}

// CacheStats статистика кэша
type CacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Size   int   `json:"size"`
}

// NewCache создает новый кэш
func NewCache(config *CacheConfig) *Cache {
	cache := &Cache{
		config: config,
		data:   make(map[string]*CacheEntry),
		stats:  &CacheStats{},
	}

	// Запускаем очистку устаревших записей
	if config.Enabled && config.CleanupInterval > 0 {
		go cache.startCleanup()
	}

	return cache
}

// Get возвращает результат из кэша
func (c *Cache) Get(key string) (*SearchResult, bool) {
	if !c.config.Enabled {
		c.mutex.Lock()
		c.stats.Misses++
		c.mutex.Unlock()
		return nil, false
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Проверяем TTL
	if time.Now().After(entry.Expiration) {
		c.stats.Misses++
		return nil, false
	}

	// Увеличиваем счетчик обращений
	entry.AccessCount++
	c.stats.Hits++
	return entry.Result, true
}

// Set сохраняет результат в кэш
func (c *Cache) Set(key string, result *SearchResult) {
	if !c.config.Enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Проверяем максимальный размер кэша
	if c.config.MaxSize > 0 && len(c.data) >= c.config.MaxSize {
		c.evictLRU()
	}

	c.data[key] = &CacheEntry{
		Result:     result,
		Expiration: time.Now().Add(c.config.TTL),
		AccessCount: 1,
	}

	c.stats.Size = len(c.data)
}

// Remove удаляет запись из кэша
func (c *Cache) Remove(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
	c.stats.Size = len(c.data)
}

// Clear очищает весь кэш
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*CacheEntry)
	c.stats.Size = 0
	c.stats.Hits = 0
	c.stats.Misses = 0
}

// GetStats возвращает статистику кэша
func (c *Cache) GetStats() *CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := *c.stats // Копируем
	stats.Size = len(c.data)
	return &stats
}

// evictLRU удаляет наименее используемую запись
func (c *Cache) evictLRU() {
	if len(c.data) == 0 {
		return
	}

	var lruKey string
	var lruCount int64 = -1

	for key, entry := range c.data {
		if lruCount == -1 || entry.AccessCount < lruCount {
			lruKey = key
			lruCount = entry.AccessCount
		}
	}

	if lruKey != "" {
		delete(c.data, lruKey)
	}
}

// startCleanup запускает периодическую очистку устаревших записей
func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup удаляет устаревшие записи
func (c *Cache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.Expiration) {
			delete(c.data, key)
		}
	}

	c.stats.Size = len(c.data)
}

