package enrichment

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewDadataEnricher(t *testing.T) {
	tests := []struct {
		name   string
		config *EnricherConfig
		want   string
	}{
		{
			name: "default timeout",
			config: &EnricherConfig{
				APIKey:      "test-key",
				BaseURL:     "https://dadata.ru",
				Timeout:     0,
				MaxRequests: 0,
			},
			want: "dadata",
		},
		{
			name: "custom timeout",
			config: &EnricherConfig{
				APIKey:      "test-key",
				BaseURL:     "https://dadata.ru",
				Timeout:     60 * time.Second,
				MaxRequests: 200,
			},
			want: "dadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enricher := NewDadataEnricher(tt.config)
			if enricher == nil {
				t.Fatal("NewDadataEnricher returned nil")
			}
			if enricher.GetName() != tt.want {
				t.Errorf("GetName() = %v, want %v", enricher.GetName(), tt.want)
			}
			if tt.config.Timeout == 0 && enricher.config.Timeout != 30*time.Second {
				t.Errorf("Default timeout = %v, want 30s", enricher.config.Timeout)
			}
			if tt.config.MaxRequests == 0 && enricher.config.MaxRequests != 100 {
				t.Errorf("Default MaxRequests = %v, want 100", enricher.config.MaxRequests)
			}
		})
	}
}

func TestDadataEnricher_GetName(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://dadata.ru",
	}
	enricher := NewDadataEnricher(config)
	if enricher.GetName() != "dadata" {
		t.Errorf("GetName() = %v, want dadata", enricher.GetName())
	}
}

func TestDadataEnricher_GetPriority(t *testing.T) {
	config := &EnricherConfig{
		APIKey:   "test-key",
		BaseURL:  "https://dadata.ru",
		Priority: 1,
	}
	enricher := NewDadataEnricher(config)
	if enricher.GetPriority() != 1 {
		t.Errorf("GetPriority() = %v, want 1", enricher.GetPriority())
	}
}

func TestDadataEnricher_IsAvailable(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://dadata.ru",
		Enabled: true,
	}
	enricher := NewDadataEnricher(config)
	if !enricher.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	config.Enabled = false
	enricher2 := NewDadataEnricher(config)
	if enricher2.IsAvailable() {
		t.Error("IsAvailable() = true, want false")
	}
}

func TestDadataEnricher_Supports(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://dadata.ru",
	}
	enricher := NewDadataEnricher(config)

	tests := []struct {
		name string
		inn  string
		bin  string
		want bool
	}{
		{
			name: "valid INN 10 digits",
			inn:  "7707083893",
			bin:  "",
			want: true,
		},
		{
			name: "valid INN 12 digits",
			inn:  "500100732259",
			bin:  "",
			want: true,
		},
		{
			name: "invalid INN",
			inn:  "123",
			bin:  "",
			want: false,
		},
		{
			name: "empty both",
			inn:  "",
			bin:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enricher.Supports(tt.inn, tt.bin)
			if got != tt.want {
				t.Errorf("Supports(%q, %q) = %v, want %v", tt.inn, tt.bin, got, tt.want)
			}
		})
	}
}

func TestDadataEnricher_Enrich_NoQuery(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://dadata.ru",
	}
	enricher := NewDadataEnricher(config)

	result, err := enricher.Enrich("", "")
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if result.Success {
		t.Error("Enrich() Success = true, want false")
	}
	if result.Error == "" {
		t.Error("Enrich() Error is empty, want error message")
	}
}

func TestDadataEnricher_Enrich_HTTPError(t *testing.T) {
	// Создаем тестовый сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}
	enricher := NewDadataEnricher(config)

	result, err := enricher.Enrich("7707083893", "")
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if result.Success {
		t.Error("Enrich() Success = true, want false")
	}
	if result.Error == "" {
		t.Error("Enrich() Error is empty, want error message")
	}
}

func TestNewAdataEnricher(t *testing.T) {
	tests := []struct {
		name   string
		config *EnricherConfig
	}{
		{
			name: "default timeout",
			config: &EnricherConfig{
				APIKey:  "test-key",
				BaseURL: "https://adata.kz",
				Timeout: 0,
			},
		},
		{
			name: "custom timeout",
			config: &EnricherConfig{
				APIKey:  "test-key",
				BaseURL: "https://adata.kz",
				Timeout: 60 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enricher := NewAdataEnricher(tt.config)
			if enricher == nil {
				t.Fatal("NewAdataEnricher returned nil")
			}
			if enricher.GetName() != "adata" {
				t.Errorf("GetName() = %v, want adata", enricher.GetName())
			}
			if tt.config.Timeout == 0 && enricher.config.Timeout != 30*time.Second {
				t.Errorf("Default timeout = %v, want 30s", enricher.config.Timeout)
			}
		})
	}
}

func TestAdataEnricher_GetName(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://adata.kz",
	}
	enricher := NewAdataEnricher(config)
	if enricher.GetName() != "adata" {
		t.Errorf("GetName() = %v, want adata", enricher.GetName())
	}
}

func TestAdataEnricher_GetPriority(t *testing.T) {
	config := &EnricherConfig{
		APIKey:   "test-key",
		BaseURL: "https://adata.kz",
		Priority: 2,
	}
	enricher := NewAdataEnricher(config)
	if enricher.GetPriority() != 2 {
		t.Errorf("GetPriority() = %v, want 2", enricher.GetPriority())
	}
}

func TestAdataEnricher_IsAvailable(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://adata.kz",
		Enabled: true,
	}
	enricher := NewAdataEnricher(config)
	if !enricher.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	config.Enabled = false
	enricher2 := NewAdataEnricher(config)
	if enricher2.IsAvailable() {
		t.Error("IsAvailable() = true, want false")
	}
}

func TestAdataEnricher_Supports(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://adata.kz",
	}
	enricher := NewAdataEnricher(config)

	tests := []struct {
		name string
		inn  string
		bin  string
		want bool
	}{
		{
			name: "valid BIN 12 digits",
			inn:  "",
			bin:  "123456789012",
			want: true,
		},
		{
			name: "invalid BIN",
			inn:  "",
			bin:  "123",
			want: false,
		},
		{
			name: "empty both",
			inn:  "",
			bin:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enricher.Supports(tt.inn, tt.bin)
			if got != tt.want {
				t.Errorf("Supports(%q, %q) = %v, want %v", tt.inn, tt.bin, got, tt.want)
			}
		})
	}
}

func TestAdataEnricher_Enrich_NoQuery(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://adata.kz",
	}
	enricher := NewAdataEnricher(config)

	result, err := enricher.Enrich("", "")
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if result.Success {
		t.Error("Enrich() Success = true, want false")
	}
	if result.Error == "" {
		t.Error("Enrich() Error is empty, want error message")
	}
}

func TestNewGispEnricher(t *testing.T) {
	tests := []struct {
		name   string
		config *EnricherConfig
	}{
		{
			name: "default timeout",
			config: &EnricherConfig{
				APIKey:  "test-key",
				BaseURL: "https://gisp.gov.ru",
				Timeout: 0,
			},
		},
		{
			name: "custom timeout",
			config: &EnricherConfig{
				APIKey:  "test-key",
				BaseURL: "https://gisp.gov.ru",
				Timeout: 60 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enricher := NewGispEnricher(tt.config)
			if enricher == nil {
				t.Fatal("NewGispEnricher returned nil")
			}
			if enricher.GetName() != "gisp" {
				t.Errorf("GetName() = %v, want gisp", enricher.GetName())
			}
			if tt.config.Timeout == 0 && enricher.config.Timeout != 30*time.Second {
				t.Errorf("Default timeout = %v, want 30s", enricher.config.Timeout)
			}
		})
	}
}

func TestGispEnricher_GetName(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://gisp.gov.ru",
	}
	enricher := NewGispEnricher(config)
	if enricher.GetName() != "gisp" {
		t.Errorf("GetName() = %v, want gisp", enricher.GetName())
	}
}

func TestGispEnricher_GetPriority(t *testing.T) {
	config := &EnricherConfig{
		APIKey:   "test-key",
		BaseURL:  "https://gisp.gov.ru",
		Priority: 3,
	}
	enricher := NewGispEnricher(config)
	if enricher.GetPriority() != 3 {
		t.Errorf("GetPriority() = %v, want 3", enricher.GetPriority())
	}
}

func TestGispEnricher_IsAvailable(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://gisp.gov.ru",
		Enabled: true,
	}
	enricher := NewGispEnricher(config)
	if !enricher.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}

	config.Enabled = false
	enricher2 := NewGispEnricher(config)
	if enricher2.IsAvailable() {
		t.Error("IsAvailable() = true, want false")
	}
}

func TestGispEnricher_Supports(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://gisp.gov.ru",
	}
	enricher := NewGispEnricher(config)

	tests := []struct {
		name string
		inn  string
		bin  string
		want bool
	}{
		{
			name: "valid INN 10 digits",
			inn:  "7707083893",
			bin:  "",
			want: true,
		},
		{
			name: "valid INN 12 digits",
			inn:  "500100732259",
			bin:  "",
			want: true,
		},
		{
			name: "invalid INN",
			inn:  "123",
			bin:  "",
			want: false,
		},
		{
			name: "empty both",
			inn:  "",
			bin:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enricher.Supports(tt.inn, tt.bin)
			if got != tt.want {
				t.Errorf("Supports(%q, %q) = %v, want %v", tt.inn, tt.bin, got, tt.want)
			}
		})
	}
}

func TestGispEnricher_Enrich_NoQuery(t *testing.T) {
	config := &EnricherConfig{
		APIKey:  "test-key",
		BaseURL: "https://gisp.gov.ru",
	}
	enricher := NewGispEnricher(config)

	result, err := enricher.Enrich("", "")
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if result.Success {
		t.Error("Enrich() Success = true, want false")
	}
	if result.Error == "" {
		t.Error("Enrich() Error is empty, want error message")
	}
}

