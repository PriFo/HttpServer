package websearch

import (
	"context"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewClient(t *testing.T) {
	config := ClientConfig{
		Timeout:   5 * time.Second,
		RateLimit: rate.Every(time.Second),
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.baseURL == "" {
		t.Error("baseURL should not be empty")
	}

	if client.timeout != config.Timeout {
		t.Errorf("timeout mismatch: expected %v, got %v", config.Timeout, client.timeout)
	}
}

func TestSanitizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal query",
			input:    "test product",
			expected: "test product",
		},
		{
			name:     "query with extra spaces",
			input:    "  test   product  ",
			expected: "test   product",
		},
		{
			name:     "very long query",
			input:    string(make([]byte, 300)),
			expected: string(make([]byte, 200)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeQuery(tt.input)
			if len(result) > 200 {
				t.Errorf("sanitizeQuery returned query longer than 200 chars: %d", len(result))
			}
		})
	}
}

func TestGenerateCacheKey(t *testing.T) {
	key1 := generateCacheKey("test query")
	key2 := generateCacheKey("test query")
	key3 := generateCacheKey("different query")

	if key1 != key2 {
		t.Error("Same query should generate same cache key")
	}

	if key1 == key3 {
		t.Error("Different queries should generate different cache keys")
	}

	if len(key1) != 64 {
		t.Errorf("Cache key should be 64 chars (SHA256 hex), got %d", len(key1))
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short text",
			input:    "Short text",
			expected: "Short text",
		},
		{
			name:     "long text",
			input:    string(make([]byte, 150)),
			expected: string(make([]byte, 100)) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.input)
			if len(result) > 103 { // 100 + "..."
				t.Errorf("extractTitle returned title longer than expected: %d", len(result))
			}
		})
	}
}

// Интеграционный тест (требует интернет-соединения)
func TestClientSearch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := ClientConfig{
		BaseURL:   "https://api.duckduckgo.com",
		Timeout:   5 * time.Second,
		RateLimit: rate.Every(time.Second),
		Cache:     NewCache(&CacheConfig{Enabled: false}),
	}

	client := NewClient(config)
	ctx := context.Background()

	result, err := client.Search(ctx, "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result == nil {
		t.Fatal("Search returned nil result")
	}

	if result.Query != "test" {
		t.Errorf("Query mismatch: expected 'test', got '%s'", result.Query)
	}

	// Источник может быть 'duckduckgo' или 'duckduckgo-html' в зависимости от типа поиска
	if result.Source != "duckduckgo" && result.Source != "duckduckgo-html" {
		t.Errorf("Source mismatch: expected 'duckduckgo' or 'duckduckgo-html', got '%s'", result.Source)
	}
}

