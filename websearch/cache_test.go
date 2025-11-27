package websearch

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             1 * time.Hour,
		CleanupInterval: 10 * time.Minute,
		MaxSize:         100,
	}

	cache := NewCache(config)

	if cache == nil {
		t.Fatal("NewCache returned nil")
	}

	if cache.config != config {
		t.Error("config not set correctly")
	}
}

func TestCache_GetSet(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		TTL:     1 * time.Hour,
	}

	cache := NewCache(config)

	result := &SearchResult{
		Query:     "test",
		Found:     true,
		Source:    "test",
		Timestamp: time.Now(),
	}

	// Set
	cache.Set("test-key", result)

	// Get
	cached, found := cache.Get("test-key")
	if !found {
		t.Fatal("Cache entry not found")
	}

	if cached.Query != result.Query {
		t.Errorf("Query mismatch: expected '%s', got '%s'", result.Query, cached.Query)
	}
}

func TestCache_Expiration(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		TTL:     100 * time.Millisecond,
	}

	cache := NewCache(config)

	result := &SearchResult{
		Query:     "test",
		Found:     true,
		Source:    "test",
		Timestamp: time.Now(),
	}

	cache.Set("test-key", result)

	// Should be found immediately
	_, found := cache.Get("test-key")
	if !found {
		t.Error("Cache entry should be found immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get("test-key")
	if found {
		t.Error("Cache entry should be expired")
	}
}

func TestCache_Disabled(t *testing.T) {
	config := &CacheConfig{
		Enabled: false,
		TTL:     1 * time.Hour,
	}

	cache := NewCache(config)

	result := &SearchResult{
		Query:     "test",
		Found:     true,
		Source:    "test",
		Timestamp: time.Now(),
	}

	// Set should not store when disabled
	cache.Set("test-key", result)

	// Get should not find when disabled
	_, found := cache.Get("test-key")
	if found {
		t.Error("Cache should not work when disabled")
	}
}

func TestCache_Stats(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		TTL:     1 * time.Hour,
	}

	cache := NewCache(config)

	result := &SearchResult{
		Query:     "test",
		Found:     true,
		Source:    "test",
		Timestamp: time.Now(),
	}

	// Initial stats
	stats := cache.GetStats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("Initial stats should be zero: hits=%d, misses=%d", stats.Hits, stats.Misses)
	}

	// Set and get
	cache.Set("test-key", result)
	cache.Get("test-key")

	// Check stats
	stats = cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
}

func TestCache_Clear(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		TTL:     1 * time.Hour,
	}

	cache := NewCache(config)

	result := &SearchResult{
		Query:     "test",
		Found:     true,
		Source:    "test",
		Timestamp: time.Now(),
	}

	cache.Set("test-key", result)
	cache.Clear()

	stats := cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("Cache should be empty after Clear, got size %d", stats.Size)
	}
}

