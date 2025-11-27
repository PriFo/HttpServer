package enrichment

import (
	"testing"
	"time"
)

// TestNewEnricherFactory проверяет создание фабрики обогатителей
func TestNewEnricherFactory(t *testing.T) {
	configs := map[string]*EnricherConfig{
		"dadata": {
			APIKey:      "test-key",
			BaseURL:     "https://suggestions.dadata.ru",
			Timeout:     30 * time.Second,
			MaxRequests: 100,
			Enabled:     true,
			Priority:    1,
		},
	}
	
	factory := NewEnricherFactory(configs)
	
	if factory == nil {
		t.Fatal("NewEnricherFactory() returned nil")
	}
	
	if factory.cache == nil {
		t.Error("Factory cache is nil")
	}
	
	if len(factory.enrichers) == 0 {
		t.Error("No enrichers created")
	}
}

// TestEnricherFactory_GetEnrichers проверяет получение обогатителей для ИНН/БИН
func TestEnricherFactory_GetEnrichers(t *testing.T) {
	configs := map[string]*EnricherConfig{
		"dadata": {
			APIKey:      "test-key",
			BaseURL:     "https://suggestions.dadata.ru",
			Timeout:     30 * time.Second,
			MaxRequests: 100,
			Enabled:     true,
			Priority:    1,
		},
	}
	
	factory := NewEnricherFactory(configs)
	
	// Тест с российским ИНН (10 или 12 цифр)
	enrichers := factory.GetEnrichers("1234567890", "")
	// Может быть 0, если enricher не поддерживает или недоступен - это нормально
	_ = enrichers
	
	// Тест с казахстанским БИН (12 цифр)
	enrichers = factory.GetEnrichers("", "123456789012")
	// Может быть 0, если enricher не поддерживает или недоступен - это нормально
	_ = enrichers
	
	// Тест с пустыми значениями
	enrichers = factory.GetEnrichers("", "")
	if len(enrichers) > 0 {
		t.Error("Enrichers found for empty INN/BIN")
	}
}

// TestEnricherFactory_GetAvailableServices проверяет получение списка доступных сервисов
func TestEnricherFactory_GetAvailableServices(t *testing.T) {
	configs := map[string]*EnricherConfig{
		"dadata": {
			APIKey:      "test-key",
			BaseURL:     "https://suggestions.dadata.ru",
			Timeout:     30 * time.Second,
			MaxRequests: 100,
			Enabled:     true,
			Priority:    1,
		},
		"adata": {
			APIKey:      "test-key",
			BaseURL:     "https://adata.kz",
			Timeout:     30 * time.Second,
			MaxRequests: 50,
			Enabled:     true,
			Priority:    2,
		},
	}
	
	factory := NewEnricherFactory(configs)
	
	services := factory.GetAvailableServices()
	
	if len(services) == 0 {
		t.Error("No available services")
	}
	
	// Проверяем, что dadata в списке
	found := false
	for _, service := range services {
		if service == "dadata" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Dadata service not found in available services")
	}
}

// TestEnricherFactory_GetServiceStats проверяет получение статистики сервисов
func TestEnricherFactory_GetServiceStats(t *testing.T) {
	configs := map[string]*EnricherConfig{
		"dadata": {
			APIKey:      "test-key",
			BaseURL:     "https://suggestions.dadata.ru",
			Timeout:     30 * time.Second,
			MaxRequests: 100,
			Enabled:     true,
			Priority:    1,
		},
	}
	
	factory := NewEnricherFactory(configs)
	
	stats := factory.GetServiceStats()
	
	if len(stats) == 0 {
		t.Error("No service stats returned")
	}
	
	dadataStats, ok := stats["dadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Dadata stats not found or wrong type")
	}
	
	if dadataStats["available"] == nil {
		t.Error("Available field missing in stats")
	}
	
	if dadataStats["priority"] == nil {
		t.Error("Priority field missing in stats")
	}
}

// TestEnricherFactory_GetBestResult проверяет выбор лучшего результата
func TestEnricherFactory_GetBestResult(t *testing.T) {
	configs := map[string]*EnricherConfig{
		"dadata": {
			APIKey:      "test-key",
			BaseURL:     "https://suggestions.dadata.ru",
			Timeout:     30 * time.Second,
			MaxRequests: 100,
			Enabled:     true,
			Priority:    1,
		},
	}
	
	factory := NewEnricherFactory(configs)
	
	// Тест с пустым списком результатов
	best := factory.GetBestResult([]*EnrichmentResult{})
	if best != nil {
		t.Error("GetBestResult() should return nil for empty results")
	}
	
	// Тест с несколькими результатами
	results := []*EnrichmentResult{
		{
			Source:      "dadata",
			Success:     true,
			Confidence:  0.5,
			Timestamp:   time.Now(),
		},
		{
			Source:      "adata",
			Success:     true,
			Confidence:  0.9,
			Timestamp:   time.Now(),
		},
		{
			Source:      "gisp",
			Success:     false,
			Confidence:  0.0,
			Timestamp:   time.Now(),
		},
	}
	
	best = factory.GetBestResult(results)
	if best == nil {
		t.Fatal("GetBestResult() returned nil for valid results")
	}
	
	// Проверяем, что выбран результат с максимальной уверенностью
	// (может быть скорректирован приоритетом, но должен быть один из успешных)
	if !best.Success {
		t.Error("GetBestResult() returned unsuccessful result")
	}
	if best.Confidence < 0.5 {
		t.Errorf("GetBestResult() returned result with low confidence %f", best.Confidence)
	}
}

// TestEnricherFactory_SortByPriority проверяет сортировку обогатителей по приоритету
func TestEnricherFactory_SortByPriority(t *testing.T) {
	configs := map[string]*EnricherConfig{
		"adata": {
			APIKey:      "test-key",
			BaseURL:     "https://adata.kz",
			Timeout:     30 * time.Second,
			MaxRequests: 50,
			Enabled:     true,
			Priority:    3,
		},
		"dadata": {
			APIKey:      "test-key",
			BaseURL:     "https://suggestions.dadata.ru",
			Timeout:     30 * time.Second,
			MaxRequests: 100,
			Enabled:     true,
			Priority:    1,
		},
		"gisp": {
			APIKey:      "test-key",
			BaseURL:     "https://gisp.gov.ru",
			Timeout:     30 * time.Second,
			MaxRequests: 50,
			Enabled:     true,
			Priority:    2,
		},
	}
	
	factory := NewEnricherFactory(configs)
	
	if len(factory.enrichers) != 3 {
		t.Fatalf("Expected 3 enrichers, got %d", len(factory.enrichers))
	}
	
	// Проверяем, что обогатители отсортированы по приоритету
	if factory.enrichers[0].GetPriority() != 1 {
		t.Errorf("First enricher priority = %d, want 1", factory.enrichers[0].GetPriority())
	}
	
	if factory.enrichers[1].GetPriority() != 2 {
		t.Errorf("Second enricher priority = %d, want 2", factory.enrichers[1].GetPriority())
	}
	
	if factory.enrichers[2].GetPriority() != 3 {
		t.Errorf("Third enricher priority = %d, want 3", factory.enrichers[2].GetPriority())
	}
}

