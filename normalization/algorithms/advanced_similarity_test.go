package algorithms

import (
	"testing"
)

func TestHybridSimilarityAdvanced(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
		min      float64
		max      float64
	}{
		{
			name:     "identical strings",
			s1:       "ООО Рога и Копыта",
			s2:       "ООО Рога и Копыта",
			expected: 1.0,
			min:      0.99,
			max:      1.0,
		},
		{
			name:     "reordered words",
			s1:       "ООО Рога и Копыта",
			s2:       "Рога и Копыта ООО",
			expected: 0.85,
			min:      0.75,
			max:      0.95,
		},
		{
			name:     "similar strings",
			s1:       "Кабель ВВГнг 3x2.5",
			s2:       "Кабель ВВГ 3x2.5",
			expected: 0.80,
			min:      0.70,
			max:      0.90,
		},
		{
			name:     "different strings",
			s1:       "Кабель ВВГнг 3x2.5",
			s2:       "Провод ПВС 3x2.5",
			expected: 0.50,
			min:      0.40,
			max:      0.60,
		},
		{
			name:     "phonetic similarity",
			s1:       "Иванов Иван Иванович",
			s2:       "Иванов И.И.",
			expected: 0.70,
			min:      0.60,
			max:      0.80,
		},
	}

	weights := DefaultSimilarityWeights()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HybridSimilarityAdvanced(tt.s1, tt.s2, weights)

			if result < tt.min || result > tt.max {
				t.Errorf("HybridSimilarityAdvanced(%q, %q) = %f, expected between %f and %f",
					tt.s1, tt.s2, result, tt.min, tt.max)
			}
		})
	}
}

func TestNgramSimilarityAdvanced(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		n        int
		expected float64
		min      float64
		max      float64
	}{
		{
			name:     "identical strings bigram",
			s1:       "тест",
			s2:       "тест",
			n:        2,
			expected: 1.0,
			min:      0.99,
			max:      1.0,
		},
		{
			name:     "similar strings bigram",
			s1:       "кабель",
			s2:       "кабел",
			n:        2,
			expected: 0.80,
			min:      0.70,
			max:      0.90,
		},
		{
			name:     "different strings bigram",
			s1:       "кабель",
			s2:       "провод",
			n:        2,
			expected: 0.20,
			min:      0.0,
			max:      0.40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NgramSimilarityAdvanced(tt.s1, tt.s2, tt.n)

			if result < tt.min || result > tt.max {
				t.Errorf("NgramSimilarityAdvanced(%q, %q, %d) = %f, expected between %f and %f",
					tt.s1, tt.s2, tt.n, result, tt.min, tt.max)
			}
		})
	}
}

func TestOptimizedHybridSimilarity(t *testing.T) {
	weights := DefaultSimilarityWeights()
	ohs := NewOptimizedHybridSimilarity(weights, 1000)

	s1 := "ООО Рога и Копыта"
	s2 := "Рога и Копыта ООО"

	// Первый вызов (кэш miss)
	result1 := ohs.Similarity(s1, s2)

	// Второй вызов (кэш hit)
	result2 := ohs.Similarity(s1, s2)

	if result1 != result2 {
		t.Errorf("Cached result differs: first=%f, second=%f", result1, result2)
	}

	// Проверяем размер кэша
	cacheSize := ohs.GetCacheSize()
	if cacheSize == 0 {
		t.Error("Cache should not be empty")
	}

	// Очищаем кэш
	ohs.ClearCache()
	cacheSize = ohs.GetCacheSize()
	if cacheSize != 0 {
		t.Errorf("Cache should be empty after ClearCache, got %d", cacheSize)
	}
}

func TestSimilarityWeights(t *testing.T) {
	// Тест 1: веса уже нормализованы (сумма = 1.0)
	weights1 := &SimilarityWeights{
		JaroWinkler: 0.5,
		LCS:         0.3,
		Phonetic:    0.2,
		Ngram:       0.0,
		Jaccard:     0.0,
	}
	weights1.NormalizeWeights()
	total1 := weights1.JaroWinkler + weights1.LCS + weights1.Phonetic + weights1.Ngram + weights1.Jaccard
	if total1 < 0.99 || total1 > 1.01 {
		t.Errorf("Normalized weights should sum to ~1.0, got %f", total1)
	}

	// Тест 2: веса не нормализованы (сумма != 1.0)
	weights2 := &SimilarityWeights{
		JaroWinkler: 1.0,
		LCS:         0.5,
		Phonetic:    0.3,
		Ngram:       0.0,
		Jaccard:     0.0,
	}
	originalJaroWinkler := weights2.JaroWinkler
	weights2.NormalizeWeights()
	total2 := weights2.JaroWinkler + weights2.LCS + weights2.Phonetic + weights2.Ngram + weights2.Jaccard
	if total2 < 0.99 || total2 > 1.01 {
		t.Errorf("Normalized weights should sum to ~1.0, got %f", total2)
	}
	// Проверяем, что веса изменились после нормализации
	if weights2.JaroWinkler == originalJaroWinkler {
		t.Error("Weights should be normalized when sum != 1.0")
	}
}

func BenchmarkHybridSimilarityAdvanced(b *testing.B) {
	weights := DefaultSimilarityWeights()
	s1 := "ООО Рога и Копыта"
	s2 := "Рога и Копыта ООО"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HybridSimilarityAdvanced(s1, s2, weights)
	}
}

func BenchmarkOptimizedHybridSimilarity(b *testing.B) {
	weights := DefaultSimilarityWeights()
	ohs := NewOptimizedHybridSimilarity(weights, 10000)
	s1 := "ООО Рога и Копыта"
	s2 := "Рога и Копыта ООО"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ohs.Similarity(s1, s2)
	}
}

func BenchmarkNgramSimilarityAdvanced(b *testing.B) {
	s1 := "ООО Рога и Копыта"
	s2 := "Рога и Копыта ООО"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NgramSimilarityAdvanced(s1, s2, 2)
	}
}
