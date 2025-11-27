package server

import (
	"testing"
)

// MockProviderClient для тестирования
type MockProviderClient struct {
	name      string
	enabled   bool
	shouldErr bool
	result    string
}

func (m *MockProviderClient) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	if m.shouldErr {
		return "", &MockError{message: "mock error"}
	}
	return m.result, nil
}

func (m *MockProviderClient) GetProviderName() string {
	return m.name
}

func (m *MockProviderClient) IsEnabled() bool {
	return m.enabled
}

type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

func TestNewCounterpartyProviderRouter(t *testing.T) {
	dadataAdapter := &MockProviderClient{name: "DaData", enabled: true}
	adataAdapter := &MockProviderClient{name: "Adata", enabled: true}

	router := NewCounterpartyProviderRouter(dadataAdapter, adataAdapter)

	if router == nil {
		t.Fatal("NewCounterpartyProviderRouter returned nil")
	}

	if router.dadataAdapter != dadataAdapter {
		t.Error("DaData adapter not set correctly")
	}

	if router.adataAdapter != adataAdapter {
		t.Error("Adata adapter not set correctly")
	}
}

func TestCounterpartyProviderRouter_RouteByTaxID(t *testing.T) {
	dadataAdapter := &MockProviderClient{name: "DaData", enabled: true}
	adataAdapter := &MockProviderClient{name: "Adata", enabled: true}

	router := NewCounterpartyProviderRouter(dadataAdapter, adataAdapter)

	tests := []struct {
		name     string
		inn      string
		bin      string
		expected string
	}{
		{
			name:     "Russian INN 10 digits",
			inn:      "1234567890",
			bin:      "",
			expected: "DaData",
		},
		{
			name:     "Russian INN 12 digits",
			inn:      "123456789012",
			bin:      "",
			expected: "DaData",
		},
		{
			name:     "Kazakh BIN 12 digits",
			inn:      "",
			bin:      "123456789012",
			expected: "Adata",
		},
		{
			name:     "BIN has priority over INN",
			inn:      "1234567890",
			bin:      "123456789012",
			expected: "Adata",
		},
		{
			name:     "No tax ID",
			inn:      "",
			bin:      "",
			expected: "",
		},
		{
			name:     "Invalid INN length",
			inn:      "12345",
			bin:      "",
			expected: "",
		},
		{
			name:     "Invalid BIN length",
			inn:      "",
			bin:      "12345",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := router.RouteByTaxID(tt.inn, tt.bin)
			if tt.expected == "" {
				if provider != nil {
					t.Errorf("Expected nil provider, got %s", provider.GetProviderName())
				}
			} else {
				if provider == nil {
					t.Errorf("Expected provider %s, got nil", tt.expected)
				} else if provider.GetProviderName() != tt.expected {
					t.Errorf("Expected provider %s, got %s", tt.expected, provider.GetProviderName())
				}
			}
		})
	}
}

func TestCounterpartyProviderRouter_RouteByTaxID_DisabledProviders(t *testing.T) {
	dadataAdapter := &MockProviderClient{name: "DaData", enabled: false}
	adataAdapter := &MockProviderClient{name: "Adata", enabled: false}

	router := NewCounterpartyProviderRouter(dadataAdapter, adataAdapter)

	// Даже если провайдеры отключены, роутер должен вернуть nil
	provider := router.RouteByTaxID("1234567890", "")
	if provider != nil {
		t.Error("Expected nil when providers are disabled")
	}
}

func TestCounterpartyProviderRouter_StandardizeCounterparty(t *testing.T) {
	dadataAdapter := &MockProviderClient{
		name:      "DaData",
		enabled:   true,
		shouldErr: false,
		result:    "ООО 'Ромашка'",
	}
	adataAdapter := &MockProviderClient{
		name:      "Adata",
		enabled:   true,
		shouldErr: false,
		result:    "ТОО 'Тест'",
	}

	router := NewCounterpartyProviderRouter(dadataAdapter, adataAdapter)

	tests := []struct {
		name     string
		company  string
		inn      string
		bin      string
		expected string
		shouldErr bool
	}{
		{
			name:      "Russian company with INN",
			company:   "ООО Ромашка",
			inn:       "1234567890",
			bin:       "",
			expected:  "ООО 'Ромашка'",
			shouldErr: false,
		},
		{
			name:      "Kazakh company with BIN",
			company:   "ТОО Тест",
			inn:       "",
			bin:       "123456789012",
			expected:  "ТОО 'Тест'",
			shouldErr: false,
		},
		{
			name:      "No tax ID",
			company:   "ООО Ромашка",
			inn:       "",
			bin:       "",
			expected:  "",
			shouldErr: true,
		},
		{
			name:      "Empty company name",
			company:   "",
			inn:       "1234567890",
			bin:       "",
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := router.StandardizeCounterparty(tt.company, tt.inn, tt.bin)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestDetectCountryByTaxID(t *testing.T) {
	tests := []struct {
		name     string
		inn      string
		bin      string
		expected string
	}{
		{
			name:     "Russian INN 10 digits",
			inn:      "1234567890",
			bin:      "",
			expected: "RU",
		},
		{
			name:     "Russian INN 12 digits",
			inn:      "123456789012",
			bin:      "",
			expected: "RU",
		},
		{
			name:     "Kazakh BIN 12 digits",
			inn:      "",
			bin:      "123456789012",
			expected: "KZ",
		},
		{
			name:     "BIN has priority",
			inn:      "1234567890",
			bin:      "123456789012",
			expected: "KZ",
		},
		{
			name:     "No tax ID",
			inn:      "",
			bin:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectCountryByTaxID(tt.inn, tt.bin)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsRussianINN(t *testing.T) {
	tests := []struct {
		name     string
		inn      string
		expected bool
	}{
		{"Valid 10 digits", "1234567890", true},
		{"Valid 12 digits", "123456789012", true},
		{"Invalid 9 digits", "123456789", false},
		{"Invalid 11 digits", "12345678901", false},
		{"Invalid 13 digits", "1234567890123", false},
		{"With spaces", "123 456 789 0", true},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRussianINN(tt.inn)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsKazakhBIN(t *testing.T) {
	tests := []struct {
		name     string
		bin      string
		expected bool
	}{
		{"Valid 12 digits", "123456789012", true},
		{"Invalid 11 digits", "12345678901", false},
		{"Invalid 13 digits", "1234567890123", false},
		{"With spaces", "123 456 789 012", true},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsKazakhBIN(tt.bin)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

