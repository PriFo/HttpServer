package services

import (
	"testing"

	"httpserver/normalization/algorithms"
)

// TestNewSimilarityService проверяет создание нового сервиса схожести
func TestNewSimilarityService(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)
	if service == nil {
		t.Error("NewSimilarityService() should not return nil")
	}
}

// TestNewSimilarityService_NilCache проверяет создание сервиса без кеша
func TestNewSimilarityService_NilCache(t *testing.T) {
	service := NewSimilarityService(nil)
	if service == nil {
		t.Error("NewSimilarityService() should not return nil even with nil cache")
	}
}

// TestSimilarityService_Compare проверяет сравнение двух строк
func TestSimilarityService_Compare(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)

	result, err := service.Compare("test string", "test string", nil)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if results, ok := result["results"].(map[string]interface{}); ok {
		if hybrid, ok := results["hybrid"].(float64); ok {
			if hybrid < 0 || hybrid > 1 {
				t.Errorf("Expected hybrid similarity between 0 and 1, got %f", hybrid)
			}
		}
	}
}

// TestSimilarityService_Compare_DifferentStrings проверяет сравнение разных строк
func TestSimilarityService_Compare_DifferentStrings(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)

	result, err := service.Compare("test string", "different string", nil)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestSimilarityService_Compare_WithWeights проверяет сравнение с весами
func TestSimilarityService_Compare_WithWeights(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)

	weights := &algorithms.SimilarityWeights{
		JaroWinkler: 0.3,
		LCS:         0.2,
		Ngram:       0.3,
		Phonetic:    0.2,
	}

	result, err := service.Compare("test string", "test string", weights)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestSimilarityService_Compare_EmptyStrings проверяет обработку пустых строк
func TestSimilarityService_Compare_EmptyStrings(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)

	// Оба пустых должна вернуть ошибку
	if _, err := service.Compare("", "", nil); err == nil {
		t.Error("Expected error when both strings are empty")
	}

	// Один пустой, другой нет — допустимый кейс
	if result, err := service.Compare("", "test", nil); err != nil {
		t.Fatalf("Compare() should allow single empty string, got error: %v", err)
	} else if result == nil {
		t.Error("Expected non-nil result for single empty string comparison")
	}
}

// TestSimilarityService_BatchCompare проверяет пакетное сравнение
func TestSimilarityService_BatchCompare(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)

	pairs := []algorithms.SimilarityPair{
		{S1: "test1", S2: "test1"},
		{S1: "test2", S2: "test2"},
	}

	results, count, err := service.BatchCompare(pairs, nil)
	if err != nil {
		t.Fatalf("BatchCompare() error = %v", err)
	}

	if len(results) != len(pairs) {
		t.Errorf("Expected %d results, got %d", len(pairs), len(results))
	}

	if count != len(pairs) {
		t.Errorf("Expected count %d, got %d", len(pairs), count)
	}
}

// TestSimilarityService_BatchCompare_Empty проверяет обработку пустого массива
func TestSimilarityService_BatchCompare_Empty(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)

	_, _, err := service.BatchCompare([]algorithms.SimilarityPair{}, nil)
	if err == nil {
		t.Error("Expected error for empty pairs array")
	}
}

// TestSimilarityService_BatchCompare_TooMany проверяет обработку слишком большого массива
func TestSimilarityService_BatchCompare_TooMany(t *testing.T) {
	cache := algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	service := NewSimilarityService(cache)

	pairs := make([]algorithms.SimilarityPair, 1001)
	for i := range pairs {
		pairs[i] = algorithms.SimilarityPair{S1: "test1", S2: "test2"}
	}

	_, _, err := service.BatchCompare(pairs, nil)
	if err == nil {
		t.Error("Expected error for too many pairs")
	}
}
