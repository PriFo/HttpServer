package server

import (
	"testing"
	"time"
)

func TestNewSystemSummaryCache(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{
			name: "valid TTL",
			ttl:  5 * time.Minute,
			want: 5 * time.Minute,
		},
		{
			name: "zero TTL should use default",
			ttl:  0,
			want: 2 * time.Minute,
		},
		{
			name: "negative TTL should use default",
			ttl:  -1 * time.Minute,
			want: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewSystemSummaryCache(tt.ttl)
			if cache == nil {
				t.Fatal("NewSystemSummaryCache returned nil")
			}
			if cache.ttl != tt.want {
				t.Errorf("cache.ttl = %v, want %v", cache.ttl, tt.want)
			}
		})
	}
}

func TestSystemSummaryCache_Get_Set(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	summary := &SystemSummary{
		TotalDatabases:    10,
		TotalUploads:      25,
		CompletedUploads:  20,
		FailedUploads:     3,
		InProgressUploads: 2,
		TotalNomenclature: 1000,
		TotalCounterparties: 500,
	}

	// Test Set
	cache.Set(summary)

	// Test Get
	got, found := cache.Get()
	if !found {
		t.Fatal("Get returned found = false, want true")
	}
	if got.TotalDatabases != summary.TotalDatabases {
		t.Errorf("Get().TotalDatabases = %d, want %d", got.TotalDatabases, summary.TotalDatabases)
	}
	if got.TotalUploads != summary.TotalUploads {
		t.Errorf("Get().TotalUploads = %d, want %d", got.TotalUploads, summary.TotalUploads)
	}
}

func TestSystemSummaryCache_Get_Empty(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	// Test Get on empty cache
	_, found := cache.Get()
	if found {
		t.Error("Get returned found = true for empty cache, want false")
	}

	// Check stats
	stats := cache.GetStats()
	if stats.Misses != 1 {
		t.Errorf("GetStats().Misses = %d, want 1", stats.Misses)
	}
}

func TestSystemSummaryCache_Get_Expired(t *testing.T) {
	cache := NewSystemSummaryCache(100 * time.Millisecond)

	summary := &SystemSummary{
		TotalDatabases: 5,
		TotalUploads:    10,
	}

	cache.Set(summary)

	// Get should succeed immediately
	_, found := cache.Get()
	if !found {
		t.Error("Get returned found = false immediately after Set, want true")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get should fail after expiration
	_, found = cache.Get()
	if found {
		t.Error("Get returned found = true after expiration, want false")
	}

	// Check stats
	stats := cache.GetStats()
	if stats.Misses < 1 {
		t.Errorf("GetStats().Misses = %d, want at least 1", stats.Misses)
	}
}

func TestSystemSummaryCache_Invalidate(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	summary := &SystemSummary{
		TotalDatabases: 5,
		TotalUploads:   10,
	}

	cache.Set(summary)

	// Verify it's in cache
	_, found := cache.Get()
	if !found {
		t.Fatal("Get returned found = false before Invalidate, want true")
	}

	// Invalidate
	cache.Invalidate()

	// Verify it's marked as stale
	isStale := cache.IsStale()
	if !isStale {
		t.Error("IsStale returned false after Invalidate, want true")
	}

	// Get should return false (but data is still there for fallback)
	_, found = cache.Get()
	if found {
		t.Error("Get returned found = true after Invalidate, want false")
	}

	// Check stats
	stats := cache.GetStats()
	if !stats.IsStale {
		t.Error("GetStats().IsStale = false, want true")
	}
	if !stats.HasData {
		t.Error("GetStats().HasData = false, want true (data should remain for fallback)")
	}
}

func TestSystemSummaryCache_Clear(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	summary := &SystemSummary{
		TotalDatabases: 5,
		TotalUploads:   10,
	}

	cache.Set(summary)

	// Verify it's in cache
	_, found := cache.Get()
	if !found {
		t.Fatal("Get returned found = false before Clear, want true")
	}

	// Clear
	cache.Clear()

	// Verify it's gone
	_, found = cache.Get()
	if found {
		t.Error("Get returned found = true after Clear, want false")
	}

	// Check stats
	stats := cache.GetStats()
	if stats.HasData {
		t.Error("GetStats().HasData = true after Clear, want false")
	}
	if !stats.IsStale {
		t.Error("GetStats().IsStale = false after Clear, want true")
	}
}

func TestSystemSummaryCache_IsStale(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	// Empty cache should be stale
	if !cache.IsStale() {
		t.Error("IsStale returned false for empty cache, want true")
	}

	summary := &SystemSummary{
		TotalDatabases: 5,
		TotalUploads:   10,
	}

	cache.Set(summary)

	// Fresh cache should not be stale
	if cache.IsStale() {
		t.Error("IsStale returned true for fresh cache, want false")
	}

	// Invalidate should make it stale
	cache.Invalidate()
	if !cache.IsStale() {
		t.Error("IsStale returned false after Invalidate, want true")
	}
}

func TestSystemSummaryCache_GetStats(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	// Test stats on empty cache
	stats := cache.GetStats()
	if stats.Hits != 0 {
		t.Errorf("GetStats().Hits = %d, want 0", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("GetStats().Misses = %d, want 0", stats.Misses)
	}
	if stats.HitRate != 0.0 {
		t.Errorf("GetStats().HitRate = %f, want 0.0", stats.HitRate)
	}
	if stats.HasData {
		t.Error("GetStats().HasData = true for empty cache, want false")
	}
	if !stats.IsStale {
		t.Error("GetStats().IsStale = false for empty cache, want true")
	}

	// Add data
	summary := &SystemSummary{
		TotalDatabases: 5,
		TotalUploads:   10,
	}
	cache.Set(summary)

	// Test miss
	_, found := cache.Get()
	if !found {
		t.Fatal("Get should succeed after Set")
	}

	// Test hit
	_, found = cache.Get()
	if !found {
		t.Fatal("Get should succeed on second call (cache hit)")
	}

	// Check stats
	stats = cache.GetStats()
	if stats.Hits < 1 {
		t.Errorf("GetStats().Hits = %d, want at least 1", stats.Hits)
	}
	if stats.Misses < 1 {
		t.Errorf("GetStats().Misses = %d, want at least 1", stats.Misses)
	}
	if stats.HitRate < 0.0 || stats.HitRate > 1.0 {
		t.Errorf("GetStats().HitRate = %f, want between 0.0 and 1.0", stats.HitRate)
	}
	if !stats.HasData {
		t.Error("GetStats().HasData = false, want true")
	}
	if stats.IsStale {
		t.Error("GetStats().IsStale = true, want false")
	}
}

func TestSystemSummaryCache_GetStats_HitRate(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	summary := &SystemSummary{
		TotalDatabases: 5,
		TotalUploads:   10,
	}
	cache.Set(summary)

	// Make 3 hits
	for i := 0; i < 3; i++ {
		cache.Get()
	}

	// Make 2 misses (by invalidating)
	cache.Invalidate()
	cache.Get()
	cache.Invalidate()
	cache.Get()

	stats := cache.GetStats()
	expectedHitRate := 3.0 / 5.0 // 3 hits out of 5 total requests
	if stats.HitRate != expectedHitRate {
		t.Errorf("GetStats().HitRate = %f, want %f", stats.HitRate, expectedHitRate)
	}
}

func TestSystemSummaryCache_ConcurrentAccess(t *testing.T) {
	cache := NewSystemSummaryCache(5 * time.Minute)

	summary := &SystemSummary{
		TotalDatabases: 5,
		TotalUploads:   10,
	}

	// Test concurrent Set
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			cache.Set(summary)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent Get
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = cache.Get()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should still work
	_, found := cache.Get()
	if !found {
		t.Error("Get failed after concurrent access")
	}
}

