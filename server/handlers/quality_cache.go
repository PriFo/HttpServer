package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProjectQualityStatsCache кэш для статистики качества проектов
type ProjectQualityStatsCache struct {
	mu          sync.RWMutex
	cache       map[string]*cachedProjectStats
	ttl         time.Duration
	totalHits   uint64
	totalMisses uint64
}

type cachedProjectStats struct {
	stats      interface{}
	cachedAt   time.Time
	lastAccess time.Time
	hitCount   int
}

// ProjectQualityCacheEntry описывает запись кэша для мониторинга
type ProjectQualityCacheEntry struct {
	Key              string     `json:"key"`
	ProjectID        int        `json:"project_id,omitempty"`
	CachedAt         time.Time  `json:"cached_at"`
	LastAccess       *time.Time `json:"last_access,omitempty"`
	HitCount         int        `json:"hit_count"`
	AgeSeconds       int        `json:"age_seconds"`
	ExpiresInSeconds int        `json:"expires_in_seconds"`
	IsExpired        bool       `json:"is_expired"`
}

// ProjectQualityCacheStats описывает агрегированную статистику кэша
type ProjectQualityCacheStats struct {
	TotalEntries   int                        `json:"total_entries"`
	ValidEntries   int                        `json:"valid_entries"`
	ExpiredEntries int                        `json:"expired_entries"`
	TTLSeconds     int                        `json:"ttl_seconds"`
	TotalHits      uint64                     `json:"total_hits"`
	TotalMisses    uint64                     `json:"total_misses"`
	HitRate        float64                    `json:"hit_rate"`
	Entries        []ProjectQualityCacheEntry `json:"entries"`
}

// NewProjectQualityStatsCache создает новый кэш для статистики проектов
func NewProjectQualityStatsCache(ttl time.Duration) *ProjectQualityStatsCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute // По умолчанию 5 минут
	}
	cache := &ProjectQualityStatsCache{
		cache: make(map[string]*cachedProjectStats),
		ttl:   ttl,
	}

	// Запускаем фоновую очистку устаревших записей
	go cache.startCleanup()

	return cache
}

// startCleanup запускает фоновую задачу для очистки устаревших записей
func (c *ProjectQualityStatsCache) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute) // Проверяем каждую минуту
	defer ticker.Stop()

	for range ticker.C {
		c.CleanupExpired()
	}
}

// Get получает статистику из кэша
func (c *ProjectQualityStatsCache) Get(projectKey string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.cache[projectKey]
	if !exists {
		c.totalMisses++
		return nil, false
	}

	// Проверяем, не истек ли срок действия
	if time.Since(entry.cachedAt) > c.ttl {
		delete(c.cache, projectKey)
		c.totalMisses++
		return nil, false
	}

	entry.hitCount++
	entry.lastAccess = time.Now()
	c.totalHits++

	return entry.stats, true
}

// Set сохраняет статистику в кэш
func (c *ProjectQualityStatsCache) Set(projectKey string, stats interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[projectKey] = &cachedProjectStats{
		stats:    stats,
		cachedAt: time.Now(),
	}
}

// Invalidate удаляет запись из кэша
func (c *ProjectQualityStatsCache) Invalidate(projectKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, projectKey)
}

// InvalidateProject удаляет все записи для проекта
func (c *ProjectQualityStatsCache) InvalidateProject(projectID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := formatProjectKey(projectID)
	delete(c.cache, key)
}

// Clear очищает весь кэш
func (c *ProjectQualityStatsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cachedProjectStats)
	c.totalHits = 0
	c.totalMisses = 0
}

// CleanupExpired удаляет все устаревшие записи
func (c *ProjectQualityStatsCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.Sub(entry.cachedAt) > c.ttl {
			delete(c.cache, key)
		}
	}
}

// Size возвращает количество записей в кэше
func (c *ProjectQualityStatsCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// GetStats возвращает статистику кэша (для мониторинга)
func (c *ProjectQualityStatsCache) GetStats() ProjectQualityCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	stats := ProjectQualityCacheStats{
		TotalEntries: len(c.cache),
		TTLSeconds:   int(c.ttl.Seconds()),
		TotalHits:    c.totalHits,
		TotalMisses:  c.totalMisses,
	}

	totalRequests := c.totalHits + c.totalMisses
	if totalRequests > 0 {
		stats.HitRate = float64(c.totalHits) / float64(totalRequests)
	}

	entries := make([]ProjectQualityCacheEntry, 0, len(c.cache))
	for key, entry := range c.cache {
		age := now.Sub(entry.cachedAt)
		expiresIn := int(c.ttl.Seconds() - age.Seconds())
		if expiresIn < 0 {
			expiresIn = 0
		}

		cacheEntry := ProjectQualityCacheEntry{
			Key:              key,
			CachedAt:         entry.cachedAt,
			HitCount:         entry.hitCount,
			AgeSeconds:       int(age.Seconds()),
			ExpiresInSeconds: expiresIn,
			IsExpired:        age > c.ttl,
		}

		if projectID := parseProjectIDFromKey(key); projectID > 0 {
			cacheEntry.ProjectID = projectID
		}

		if !entry.lastAccess.IsZero() {
			lastAccess := entry.lastAccess
			cacheEntry.LastAccess = &lastAccess
		}

		if cacheEntry.IsExpired {
			stats.ExpiredEntries++
		} else {
			stats.ValidEntries++
		}

		entries = append(entries, cacheEntry)
	}

	stats.Entries = entries
	return stats
}

// formatProjectKey форматирует ключ для проекта
func formatProjectKey(projectID int) string {
	return fmt.Sprintf("project:%d", projectID)
}

// parseProjectIDFromKey извлекает projectID из ключа
func parseProjectIDFromKey(key string) int {
	if !strings.HasPrefix(key, "project:") {
		return 0
	}
	projectIDStr := strings.TrimPrefix(key, "project:")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return 0
	}
	return projectID
}
