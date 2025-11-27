package websearch

import (
	"context"
	"errors"
	"testing"
	"time"

	"httpserver/websearch/types"
)

// mockProvider мок провайдера для тестирования
type mockProvider struct {
	name      string
	available bool
	shouldErr bool
	result    *types.SearchResult
}

func (m *mockProvider) GetName() string {
	return m.name
}

func (m *mockProvider) IsAvailable() bool {
	return m.available
}

func (m *mockProvider) ValidateCredentials(ctx context.Context) error {
	if !m.available {
		return errors.New("provider not available")
	}
	return nil
}

func (m *mockProvider) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	if m.shouldErr {
		return nil, errors.New("search error")
	}
	return m.result, nil
}

func (m *mockProvider) GetRateLimit() time.Duration {
	return time.Second
}

func TestNewProviderRouter(t *testing.T) {
	providers := map[string]types.SearchProviderInterface{
		"test": &mockProvider{name: "test", available: true},
	}

	reliabilityManager := NewStubReliabilityManager()
	config := RouterConfig{
		Strategy: StrategyRoundRobin,
	}

	router := NewProviderRouter(providers, reliabilityManager, config)

	if router == nil {
		t.Fatal("NewProviderRouter returned nil")
	}

	if router.config.Strategy != StrategyRoundRobin {
		t.Errorf("Expected strategy %s, got %s", StrategyRoundRobin, router.config.Strategy)
	}
}

func TestProviderRouter_SearchWithFallback(t *testing.T) {
	successResult := &types.SearchResult{
		Query:      "test",
		Found:      true,
		Results:    []types.SearchItem{},
		Confidence: 0.8,
		Source:     "test",
		Timestamp:  time.Now(),
	}

	tests := []struct {
		name            string
		providers       map[string]types.SearchProviderInterface
		maxAttempts     int
		expectedSuccess bool
		expectedError   bool
	}{
		{
			name: "successful search",
			providers: map[string]types.SearchProviderInterface{
				"success": &mockProvider{
					name:      "success",
					available: true,
					shouldErr: false,
					result:    successResult,
				},
			},
			maxAttempts:     1,
			expectedSuccess: true,
			expectedError:   false,
		},
		{
			name: "fallback to second provider",
			providers: map[string]types.SearchProviderInterface{
				"fail": &mockProvider{
					name:      "fail",
					available: true,
					shouldErr: true,
				},
				"success": &mockProvider{
					name:      "success",
					available: true,
					shouldErr: false,
					result:    successResult,
				},
			},
			maxAttempts:     2,
			expectedSuccess: true,
			expectedError:   false,
		},
		{
			name: "all providers fail",
			providers: map[string]types.SearchProviderInterface{
				"fail1": &mockProvider{
					name:      "fail1",
					available: true,
					shouldErr: true,
				},
				"fail2": &mockProvider{
					name:      "fail2",
					available: true,
					shouldErr: true,
				},
			},
			maxAttempts:     2,
			expectedSuccess: false,
			expectedError:   true,
		},
		{
			name: "no available providers",
			providers: map[string]types.SearchProviderInterface{
				"unavailable": &mockProvider{
					name:      "unavailable",
					available: false,
				},
			},
			maxAttempts:     1,
			expectedSuccess: false,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reliabilityManager := NewStubReliabilityManager()
			config := RouterConfig{
				Strategy: StrategyRoundRobin,
			}

			router := NewProviderRouter(tt.providers, reliabilityManager, config)
			ctx := context.Background()

			result, err := router.SearchWithFallback(ctx, "test", tt.maxAttempts)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if tt.expectedSuccess {
				if result == nil {
					t.Error("Expected result, got nil")
				} else if !result.Found {
					t.Error("Expected found=true, got false")
				}
			}
		})
	}
}

func TestProviderRouter_UpdateProviders(t *testing.T) {
	initialProviders := map[string]types.SearchProviderInterface{
		"provider1": &mockProvider{name: "provider1", available: true},
	}

	reliabilityManager := NewStubReliabilityManager()
	config := RouterConfig{
		Strategy: StrategyRoundRobin,
	}

	router := NewProviderRouter(initialProviders, reliabilityManager, config)

	// Обновляем провайдеров
	newProviders := map[string]types.SearchProviderInterface{
		"provider2": &mockProvider{name: "provider2", available: true},
		"provider3": &mockProvider{name: "provider3", available: true},
	}

	router.UpdateProviders(newProviders)

	updatedProviders := router.GetProviders()
	if len(updatedProviders) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(updatedProviders))
	}

	if _, exists := updatedProviders["provider2"]; !exists {
		t.Error("Provider2 should exist")
	}
	if _, exists := updatedProviders["provider3"]; !exists {
		t.Error("Provider3 should exist")
	}
}

func TestProviderRouter_SelectProviders(t *testing.T) {
	providers := []types.SearchProviderInterface{
		&mockProvider{name: "p1", available: true},
		&mockProvider{name: "p2", available: true},
		&mockProvider{name: "p3", available: true},
	}

	reliabilityManager := NewStubReliabilityManager()
	config := RouterConfig{
		Strategy: StrategyRoundRobin,
	}

	router := NewProviderRouter(nil, reliabilityManager, config)

	// Тест round-robin
	selected := router.selectProviders(providers, 2)
	if len(selected) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(selected))
	}

	// Тест weighted (должен fallback на round-robin без ReliabilityManager)
	router.config.Strategy = StrategyWeighted
	selected = router.selectProviders(providers, 2)
	if len(selected) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(selected))
	}

	// Тест random
	router.config.Strategy = StrategyRandom
	selected = router.selectProviders(providers, 2)
	if len(selected) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(selected))
	}
}
