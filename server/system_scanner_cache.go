package server

import (
	"sync"
	"time"
)

// SystemSummaryCache кэш для результатов сканирования системы
type SystemSummaryCache struct {
	mu       sync.RWMutex
	summary  *SystemSummary
	expiry   time.Time
	ttl      time.Duration
	isDirty  bool // Флаг, указывающий, что данные устарели
	hits     int64 // Количество попаданий в кеш
	misses   int64 // Количество промахов кеша
}

// NewSystemSummaryCache создает новый кэш для системной сводки
func NewSystemSummaryCache(ttl time.Duration) *SystemSummaryCache {
	if ttl <= 0 {
		ttl = 2 * time.Minute // TTL по умолчанию: 2 минуты
	}
	return &SystemSummaryCache{
		ttl: ttl,
	}
}

// Get возвращает кэшированную сводку, если она актуальна
func (c *SystemSummaryCache) Get() (*SystemSummary, bool) {
	c.mu.RLock()
	hasData := c.summary != nil
	isDirty := c.isDirty
	expired := !c.expiry.IsZero() && time.Now().After(c.expiry)
	c.mu.RUnlock()

	// Если данных нет или они устарели, обновляем статистику и возвращаем false
	if !hasData || isDirty || expired {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	// Данные актуальны - обновляем статистику и возвращаем
	c.mu.Lock()
	c.hits++
	c.mu.Unlock()

	c.mu.RLock()
	result := c.summary
	c.mu.RUnlock()

	return result, true
}

// Set сохраняет сводку в кэш
func (c *SystemSummaryCache) Set(summary *SystemSummary) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.summary = summary
	c.expiry = time.Now().Add(c.ttl)
	c.isDirty = false
}

// Invalidate инвалидирует кэш (помечает как устаревший)
func (c *SystemSummaryCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isDirty = true
	// Не удаляем данные полностью, чтобы при следующем запросе
	// они использовались как fallback, если новое сканирование не удалось
}

// Clear полностью очищает кэш
func (c *SystemSummaryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.summary = nil
	c.expiry = time.Time{}
	c.isDirty = false
}

// IsStale проверяет, устарели ли данные
func (c *SystemSummaryCache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.summary == nil || c.isDirty || time.Now().After(c.expiry)
}

// CacheStats статистика кеша
type CacheStats struct {
	Hits          int64     `json:"hits"`
	Misses        int64     `json:"misses"`
	HitRate       float64   `json:"hit_rate"`       // Процент попаданий (0-1)
	HasData       bool      `json:"has_data"`       // Есть ли данные в кеше
	IsStale       bool      `json:"is_stale"`       // Устарели ли данные
	Expiry        time.Time `json:"expiry"`         // Время истечения кеша
	TimeToExpiry  string    `json:"time_to_expiry"` // Оставшееся время до истечения
}

// GetStats возвращает статистику кеша
func (c *SystemSummaryCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	isStale := c.summary == nil || c.isDirty || time.Now().After(c.expiry)
	timeToExpiry := ""
	if !c.expiry.IsZero() && !isStale {
		remaining := time.Until(c.expiry)
		if remaining > 0 {
			timeToExpiry = remaining.Round(time.Second).String()
		}
	}

	return CacheStats{
		Hits:         c.hits,
		Misses:       c.misses,
		HitRate:      hitRate,
		HasData:      c.summary != nil,
		IsStale:      isStale,
		Expiry:       c.expiry,
		TimeToExpiry: timeToExpiry,
	}
}

