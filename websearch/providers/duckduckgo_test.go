package providers

import (
	"context"
	"testing"
	"time"

	"httpserver/websearch/types"
)

func TestNewDuckDuckGoProvider(t *testing.T) {
	provider := NewDuckDuckGoProvider(5*time.Second, time.Second)

	if provider == nil {
		t.Fatal("NewDuckDuckGoProvider returned nil")
	}

	if provider.GetName() != "duckduckgo" {
		t.Errorf("Expected name 'duckduckgo', got '%s'", provider.GetName())
	}

	if !provider.IsAvailable() {
		t.Error("DuckDuckGo provider should be available by default")
	}
}

func TestDuckDuckGoProvider_ValidateCredentials(t *testing.T) {
	provider := NewDuckDuckGoProvider(5*time.Second, time.Second)
	ctx := context.Background()

	err := provider.ValidateCredentials(ctx)
	if err != nil {
		t.Errorf("ValidateCredentials should not return error for DuckDuckGo: %v", err)
	}
}

func TestDuckDuckGoProvider_TransformResults(t *testing.T) {
	provider := NewDuckDuckGoProvider(5*time.Second, time.Second)

	tests := []struct {
		name     string
		response *DuckDuckGoResponse
		query    string
		check    func(*testing.T, *types.SearchResult)
	}{
		{
			name: "empty response",
			response: &DuckDuckGoResponse{},
			query: "test",
			check: func(t *testing.T, result *types.SearchResult) {
				if result.Found {
					t.Error("Empty response should not be found")
				}
				if len(result.Results) != 0 {
					t.Errorf("Expected 0 results, got %d", len(result.Results))
				}
			},
		},
		{
			name: "response with abstract",
			response: &DuckDuckGoResponse{
				Heading:      "Test Heading",
				AbstractText: "Test abstract text",
				AbstractURL:  "https://example.com",
			},
			query: "test",
			check: func(t *testing.T, result *types.SearchResult) {
				if !result.Found {
					t.Error("Response with abstract should be found")
				}
				if len(result.Results) == 0 {
					t.Error("Expected at least one result")
				}
				if result.Confidence < 0.7 {
					t.Errorf("Expected confidence >= 0.7, got %f", result.Confidence)
				}
			},
		},
		{
			name: "response with related topics",
			response: &DuckDuckGoResponse{
				RelatedTopics: []struct {
					FirstURL string `json:"FirstURL"`
					Icon     struct {
						Height string `json:"Height"`
						URL    string `json:"URL"`
						Width  string `json:"Width"`
					} `json:"Icon"`
					Result string `json:"Result"`
					Text   string `json:"Text"`
				}{
					{
						FirstURL: "https://example.com/topic1",
						Text:     "Topic 1",
					},
				},
			},
			query: "test",
			check: func(t *testing.T, result *types.SearchResult) {
				if !result.Found {
					t.Error("Response with related topics should be found")
				}
				if len(result.Results) == 0 {
					t.Error("Expected at least one result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.transformResults(tt.query, tt.response)
			if result == nil {
				t.Fatal("transformResults returned nil")
			}
			if result.Query != tt.query {
				t.Errorf("Query mismatch: expected '%s', got '%s'", tt.query, result.Query)
			}
			if result.Source != "duckduckgo" {
				t.Errorf("Source mismatch: expected 'duckduckgo', got '%s'", result.Source)
			}
			tt.check(t, result)
		})
	}
}

func TestDuckDuckGoProvider_GetRateLimit(t *testing.T) {
	rateLimit := 2 * time.Second
	provider := NewDuckDuckGoProvider(5*time.Second, rateLimit)

	if provider.GetRateLimit() != rateLimit {
		t.Errorf("Expected rate limit %v, got %v", rateLimit, provider.GetRateLimit())
	}
}

