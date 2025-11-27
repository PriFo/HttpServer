package enrichment

import (
	"testing"
	"time"
)

func TestNewEnrichmentCache(t *testing.T) {
	tests := []struct {
		name   string
		config *CacheConfig
		want   bool
	}{
		{
			name: "enabled cache",
			config: &CacheConfig{
				Enabled:         true,
				TTL:             5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
			},
			want: true,
		},
		{
			name: "disabled cache",
			config: &CacheConfig{
				Enabled:         false,
				TTL:             5 * time.Minute,
				CleanupInterval: 1 * time.Minute,
			},
			want: false,
		},
		{
			name: "zero cleanup interval",
			config: &CacheConfig{
				Enabled:         true,
				TTL:             5 * time.Minute,
				CleanupInterval: 0,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewEnrichmentCache(tt.config)
			if cache == nil {
				t.Fatal("NewEnrichmentCache returned nil")
			}
			if cache.config.Enabled != tt.want {
				t.Errorf("cache.config.Enabled = %v, want %v", cache.config.Enabled, tt.want)
			}
		})
	}
}

func TestEnrichmentCache_Get_Set(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	result := &EnrichmentResult{
		Source:    "test",
		Timestamp: time.Now(),
		Success:   true,
		INN:       "1234567890",
	}

	// Test Set and Get
	key := "test_key"
	cache.Set(key, result)

	got, found := cache.Get(key)
	if !found {
		t.Fatal("Get returned found = false, want true")
	}
	if got.INN != result.INN {
		t.Errorf("Get().INN = %v, want %v", got.INN, result.INN)
	}
	if got.Source != result.Source {
		t.Errorf("Get().Source = %v, want %v", got.Source, result.Source)
	}
}

func TestEnrichmentCache_Get_NotFound(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	_, found := cache.Get("non_existent_key")
	if found {
		t.Error("Get returned found = true, want false")
	}
}

func TestEnrichmentCache_Get_Expired(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             100 * time.Millisecond,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	result := &EnrichmentResult{
		Source:    "test",
		Timestamp: time.Now(),
		Success:   true,
	}

	key := "expired_key"
	cache.Set(key, result)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, found := cache.Get(key)
	if found {
		t.Error("Get returned found = true for expired entry, want false")
	}
}

func TestEnrichmentCache_Get_Disabled(t *testing.T) {
	config := &CacheConfig{
		Enabled:         false,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	result := &EnrichmentResult{
		Source:    "test",
		Timestamp: time.Now(),
		Success:   true,
	}

	key := "test_key"
	cache.Set(key, result)

	_, found := cache.Get(key)
	if found {
		t.Error("Get returned found = true for disabled cache, want false")
	}
}

func TestEnrichmentCache_Remove(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	result := &EnrichmentResult{
		Source:    "test",
		Timestamp: time.Now(),
		Success:   true,
	}

	key := "test_key"
	cache.Set(key, result)

	// Verify it's in cache
	_, found := cache.Get(key)
	if !found {
		t.Fatal("Get returned found = false before Remove, want true")
	}

	// Remove it
	cache.Remove(key)

	// Verify it's gone
	_, found = cache.Get(key)
	if found {
		t.Error("Get returned found = true after Remove, want false")
	}
}

func TestEnrichmentCache_Clear(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	// Add multiple entries
	for i := 0; i < 5; i++ {
		result := &EnrichmentResult{
			Source:    "test",
			Timestamp: time.Now(),
			Success:   true,
		}
		cache.Set(string(rune('a'+i)), result)
	}

	// Verify entries exist
	stats := cache.GetStats()
	if stats.Size != 5 {
		t.Errorf("GetStats().Size = %d, want 5", stats.Size)
	}

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	stats = cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("GetStats().Size = %d after Clear, want 0", stats.Size)
	}
	if stats.Hits != 0 {
		t.Errorf("GetStats().Hits = %d after Clear, want 0", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("GetStats().Misses = %d after Clear, want 0", stats.Misses)
	}
}

func TestEnrichmentCache_GetStats(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	// Initial stats
	stats := cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("GetStats().Size = %d, want 0", stats.Size)
	}
	if stats.Hits != 0 {
		t.Errorf("GetStats().Hits = %d, want 0", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("GetStats().Misses = %d, want 0", stats.Misses)
	}

	// Add entry
	result := &EnrichmentResult{
		Source:    "test",
		Timestamp: time.Now(),
		Success:   true,
	}
	cache.Set("key1", result)

	// Get entry (hit)
	cache.Get("key1")

	// Get non-existent entry (miss)
	cache.Get("non_existent")

	// Check stats
	stats = cache.GetStats()
	if stats.Size != 1 {
		t.Errorf("GetStats().Size = %d, want 1", stats.Size)
	}
	if stats.Hits != 1 {
		t.Errorf("GetStats().Hits = %d, want 1", stats.Hits)
	}
	if stats.Misses < 1 {
		t.Errorf("GetStats().Misses = %d, want at least 1", stats.Misses)
	}
}

func TestEnrichmentCache_ConcurrentAccess(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			result := &EnrichmentResult{
				Source:    "test",
				Timestamp: time.Now(),
				Success:   true,
				INN:       string(rune('0' + idx)),
			}
			cache.Set(string(rune('a'+idx)), result)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(idx int) {
			cache.Get(string(rune('a' + idx)))
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all entries are present
	stats := cache.GetStats()
	if stats.Size != 10 {
		t.Errorf("GetStats().Size = %d, want 10", stats.Size)
	}
}

func TestEnrichmentCache_Set_Disabled(t *testing.T) {
	config := &CacheConfig{
		Enabled:         false,
		TTL:             5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewEnrichmentCache(config)

	result := &EnrichmentResult{
		Source:    "test",
		Timestamp: time.Now(),
		Success:   true,
	}

	key := "test_key"
	cache.Set(key, result)

	// Verify entry is not stored
	stats := cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("GetStats().Size = %d for disabled cache, want 0", stats.Size)
	}
}

func TestEnrichmentCache_Cleanup(t *testing.T) {
	config := &CacheConfig{
		Enabled:         true,
		TTL:             100 * time.Millisecond,
		CleanupInterval: 200 * time.Millisecond,
	}
	cache := NewEnrichmentCache(config)

	// Add entries
	for i := 0; i < 3; i++ {
		result := &EnrichmentResult{
			Source:    "test",
			Timestamp: time.Now(),
			Success:   true,
		}
		cache.Set(string(rune('a'+i)), result)
	}

	// Verify entries exist
	stats := cache.GetStats()
	if stats.Size != 3 {
		t.Errorf("GetStats().Size = %d, want 3", stats.Size)
	}

	// Wait for expiration and cleanup
	time.Sleep(250 * time.Millisecond)

	// Verify entries are cleaned up
	stats = cache.GetStats()
	if stats.Size != 0 {
		t.Errorf("GetStats().Size = %d after cleanup, want 0", stats.Size)
	}
}

