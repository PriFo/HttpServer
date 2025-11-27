package server

import (
	"testing"
)

func TestNewAdataClient(t *testing.T) {
	apiToken := "test-api-token"
	baseURL := "https://api.adata.kz"

	client := NewAdataClient(apiToken, baseURL)

	if client == nil {
		t.Fatal("NewAdataClient returned nil")
	}

	if client.apiToken != apiToken {
		t.Errorf("Expected API token %s, got %s", apiToken, client.apiToken)
	}

	if client.baseURL != baseURL {
		t.Errorf("Expected base URL %s, got %s", baseURL, client.baseURL)
	}

	if client.httpClient == nil {
		t.Error("HTTP client is nil")
	}
}

func TestNewAdataClient_DefaultBaseURL(t *testing.T) {
	apiToken := "test-api-token"
	client := NewAdataClient(apiToken, "")

	if client == nil {
		t.Fatal("NewAdataClient returned nil")
	}

	expectedBaseURL := "https://api.adata.kz"
	if client.baseURL != expectedBaseURL {
		t.Errorf("Expected default base URL %s, got %s", expectedBaseURL, client.baseURL)
	}
}

func TestAdataClient_FindCompany_NoAPIToken(t *testing.T) {
	client := NewAdataClient("", "")
	
	_, err := client.FindCompany("ТОО Тест", "")
	if err == nil {
		t.Error("Expected error when API token is not set")
	}

	if err.Error() != "Adata API token is not set" {
		t.Errorf("Expected error message about missing API token, got: %v", err)
	}
}

func TestExtractBINFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid BIN with spaces",
			input:    "123 456 789 012",
			expected: "123456789012",
		},
		{
			name:     "Valid BIN without spaces",
			input:    "123456789012",
			expected: "123456789012",
		},
		{
			name:     "BIN with dashes",
			input:    "123-456-789-012",
			expected: "123456789012",
		},
		{
			name:     "Invalid length",
			input:    "12345678901",
			expected: "",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "BIN with text",
			input:    "БИН: 123456789012",
			expected: "123456789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBINFromString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Интеграционные тесты требуют реального API токена
// func TestAdataClient_FindCompany_Integration(t *testing.T) {
// 	apiToken := os.Getenv("ADATA_API_KEY")
// 	if apiToken == "" {
// 		t.Skip("ADATA_API_KEY not set, skipping integration test")
// 	}
//
// 	client := NewAdataClient(apiToken, "")
// 	result, err := client.FindCompany("ТОО Тест", "123456789012")
// 	if err != nil {
// 		t.Fatalf("FindCompany failed: %v", err)
// 	}
//
// 	if result == nil {
// 		t.Fatal("FindCompany returned nil result")
// 	}
// }

