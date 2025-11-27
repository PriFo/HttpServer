package server

import (
	"testing"
)

func TestNewDaDataClient(t *testing.T) {
	apiKey := "test-api-key"
	secretKey := "test-secret-key"
	baseURL := "https://suggestions.dadata.ru/suggestions/api/4_1/rs"

	client := NewDaDataClient(apiKey, secretKey, baseURL)

	if client == nil {
		t.Fatal("NewDaDataClient returned nil")
	}

	if client.apiKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, client.apiKey)
	}

	if client.secretKey != secretKey {
		t.Errorf("Expected secret key %s, got %s", secretKey, client.secretKey)
	}

	if client.baseURL != baseURL {
		t.Errorf("Expected base URL %s, got %s", baseURL, client.baseURL)
	}

	if client.httpClient == nil {
		t.Error("HTTP client is nil")
	}
}

func TestNewDaDataClient_DefaultBaseURL(t *testing.T) {
	apiKey := "test-api-key"
	client := NewDaDataClient(apiKey, "", "")

	if client == nil {
		t.Fatal("NewDaDataClient returned nil")
	}

	expectedBaseURL := "https://suggestions.dadata.ru/suggestions/api/4_1/rs"
	if client.baseURL != expectedBaseURL {
		t.Errorf("Expected default base URL %s, got %s", expectedBaseURL, client.baseURL)
	}
}

func TestDaDataClient_SuggestParty_NoAPIKey(t *testing.T) {
	client := NewDaDataClient("", "", "")
	
	_, err := client.SuggestParty("ООО Ромашка", "")
	if err == nil {
		t.Error("Expected error when API key is not set")
	}

	if err.Error() != "DaData API key is not set" {
		t.Errorf("Expected error message about missing API key, got: %v", err)
	}
}

// Интеграционные тесты требуют реального API ключа
// func TestDaDataClient_SuggestParty_Integration(t *testing.T) {
// 	apiKey := os.Getenv("DADATA_API_KEY")
// 	if apiKey == "" {
// 		t.Skip("DADATA_API_KEY not set, skipping integration test")
// 	}
//
// 	client := NewDaDataClient(apiKey, "", "")
// 	result, err := client.SuggestParty("ООО Ромашка", "")
// 	if err != nil {
// 		t.Fatalf("SuggestParty failed: %v", err)
// 	}
//
// 	if result == nil {
// 		t.Fatal("SuggestParty returned nil result")
// 	}
//
// 	if result.FullName == "" && result.ShortName == "" {
// 		t.Error("Result should have at least one name")
// 	}
// }

