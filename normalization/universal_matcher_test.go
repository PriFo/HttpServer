package normalization

import (
	"testing"
)

func TestUniversalMatcher_Similarity(t *testing.T) {
	matcher := NewUniversalMatcher(true)

	testCases := []struct {
		s1, s2   string
		method   string
		expected float64
	}{
		{"молоток", "молоток", "levenshtein", 1.0},
		{"молоток", "молотак", "levenshtein", 0.7},
		{"кабель", "кабель", "jaccard", 1.0},
		{"провод медный", "медный провод", "jaccard", 0.7},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.s1+"_"+tc.s2, func(t *testing.T) {
			similarity, err := matcher.Similarity(tc.s1, tc.s2, tc.method)
			if err != nil {
				t.Errorf("Similarity failed: %v", err)
				return
			}
			if similarity < tc.expected {
				t.Errorf("Similarity(%q, %q, %q) = %.3f, ожидалось >= %.3f",
					tc.s1, tc.s2, tc.method, similarity, tc.expected)
			}
		})
	}
}

func TestUniversalMatcher_SimilarityMultiple(t *testing.T) {
	matcher := NewUniversalMatcher(true)

	s1, s2 := "молоток", "молотак"
	methods := []string{"levenshtein", "jaccard", "damerau_levenshtein"}

	results, err := matcher.SimilarityMultiple(s1, s2, methods)
	if err != nil {
		t.Fatalf("SimilarityMultiple failed: %v", err)
	}

	if len(results) != len(methods) {
		t.Errorf("Expected %d results, got %d", len(methods), len(results))
	}

	for _, method := range methods {
		if _, ok := results[method]; !ok {
			t.Errorf("Method %s not found in results", method)
		}
	}
}

func TestUniversalMatcher_HybridSimilarity(t *testing.T) {
	matcher := NewUniversalMatcher(true)

	s1, s2 := "молоток", "молотак"
	methods := []string{"levenshtein", "jaccard"}
	weights := []float64{0.6, 0.4}

	similarity, err := matcher.HybridSimilarity(s1, s2, methods, weights)
	if err != nil {
		t.Fatalf("HybridSimilarity failed: %v", err)
	}

	if similarity < 0.0 || similarity > 1.0 {
		t.Errorf("HybridSimilarity returned invalid value: %.3f", similarity)
	}
}

func TestUniversalMatcher_EnsembleSimilarity(t *testing.T) {
	matcher := NewUniversalMatcher(true)

	s1, s2 := "молоток", "молотак"
	methods := []string{"levenshtein", "jaccard"}

	similarity, err := matcher.EnsembleSimilarity(s1, s2, methods, "average")
	if err != nil {
		t.Fatalf("EnsembleSimilarity failed: %v", err)
	}

	if similarity < 0.0 || similarity > 1.0 {
		t.Errorf("EnsembleSimilarity returned invalid value: %.3f", similarity)
	}
}

func TestUniversalMatcher_GetAvailableMethods(t *testing.T) {
	matcher := NewUniversalMatcher(true)

	methods := matcher.GetAvailableMethods()
	if len(methods) == 0 {
		t.Error("Expected at least one method")
	}
}

func TestUniversalMatcher_Cache(t *testing.T) {
	matcher := NewUniversalMatcher(true)

	s1, s2 := "тест", "тест"
	method := "levenshtein"

	// Первый вызов
	sim1, err1 := matcher.Similarity(s1, s2, method)
	if err1 != nil {
		t.Fatalf("First call failed: %v", err1)
	}

	// Второй вызов (должен использовать кэш)
	sim2, err2 := matcher.Similarity(s1, s2, method)
	if err2 != nil {
		t.Fatalf("Second call failed: %v", err2)
	}

	if sim1 != sim2 {
		t.Errorf("Cached result differs: %.3f != %.3f", sim1, sim2)
	}

	// Очистка кэша
	matcher.ClearCache()
}

func TestUniversalMatcher_IsMatch(t *testing.T) {
	matcher := NewUniversalMatcher(true)

	testCases := []struct {
		s1, s2   string
		method   string
		expected bool
	}{
		{"молоток", "молоток", "levenshtein", true},
		{"молоток", "кабель", "levenshtein", false},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.s1+"_"+tc.s2, func(t *testing.T) {
			isMatch, err := matcher.IsMatch(tc.s1, tc.s2, tc.method)
			if err != nil {
				t.Errorf("IsMatch failed: %v", err)
				return
			}
			if isMatch != tc.expected {
				t.Errorf("IsMatch(%q, %q, %q) = %v, ожидалось %v",
					tc.s1, tc.s2, tc.method, isMatch, tc.expected)
			}
		})
	}
}
