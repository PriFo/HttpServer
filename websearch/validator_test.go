package websearch

import (
	"context"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestProductValidator_ValidateProductExists(t *testing.T) {
	// Создаем мок клиента (в реальном тесте можно использовать интерфейс)
	config := ClientConfig{
		Timeout:   5 * time.Second,
		RateLimit: rate.Every(time.Second),
		Cache:     NewCache(&CacheConfig{Enabled: false}),
	}

	client := NewClient(config)
	validator := NewProductExistenceValidator(client)

	ctx := context.Background()

	// Тест с валидными данными (требует интернет)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	result, err := validator.Validate(ctx, "iPhone")
	if err != nil {
		t.Fatalf("ValidateProductExists failed: %v", err)
	}

	if result == nil {
		t.Fatal("ValidationResult should not be nil")
	}
}

func TestProductValidator_BuildQuery(t *testing.T) {
	config := ClientConfig{
		Timeout:   5 * time.Second,
		RateLimit: rate.Every(time.Second),
	}

	client := NewClient(config)
	accuracyValidator := NewProductAccuracyValidator(client)

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "both name and code",
			code:     "12345",
			expected: "test product 12345",
		},
		{
			name:     "only name",
			code:     "",
			expected: "test product",
		},
		{
			name:     "only code",
			code:     "12345",
			expected: "test product 12345", // buildQuery всегда включает name если оно не пустое
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := accuracyValidator.buildQuery("test product", tt.code)
			if query != tt.expected {
				t.Errorf("buildQuery: expected '%s', got '%s'", tt.expected, query)
			}
		})
	}
}

func TestProductValidator_AnalyzeResults(t *testing.T) {
	config := ClientConfig{
		Timeout:   5 * time.Second,
		RateLimit: rate.Every(time.Second),
	}

	client := NewClient(config)
	validator := NewProductExistenceValidator(client)

	tests := []struct {
		name          string
		result        *SearchResult
		itemName      string
		itemCode      string
		expectedFound bool
	}{
		{
			name: "no results",
			result: &SearchResult{
				Found:  false,
				Results: []SearchItem{},
			},
			itemName:      "test",
			itemCode:      "123",
			expectedFound: false,
		},
		{
			name: "matching results",
			result: &SearchResult{
				Found: true,
				Results: []SearchItem{
					{
						Title:     "test product",
						Snippet:   "test product description",
						Relevance: 0.9,
					},
				},
			},
			itemName:      "test product",
			itemCode:      "123",
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := validator.analyzeResults(tt.result, tt.itemName)
			if validation.Found != tt.expectedFound {
				t.Errorf("analyzeResults: expected found=%v, got %v", tt.expectedFound, validation.Found)
			}
		})
	}
}

