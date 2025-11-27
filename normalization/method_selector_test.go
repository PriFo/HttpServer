package normalization

import (
	"testing"
)

func TestMethodSelector_SelectMethod(t *testing.T) {
	matcher := NewUniversalMatcher(true)
	selector := NewMethodSelector(matcher)

	testCases := []struct {
		s1, s2 string
		desc   string
	}{
		{"молоток", "молотак", "Короткие строки с опечаткой"},
		{"кабель медный ВВГ", "кабель медный ВВГ", "Длинные строки"},
		{"провод", "провода", "Разные формы слова"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			method, methods, err := selector.SelectMethod(tc.s1, tc.s2)
			if err != nil {
				t.Errorf("SelectMethod failed: %v", err)
				return
			}
			if method == "" {
				t.Error("Expected non-empty method name")
			}
			if len(methods) == 0 {
				t.Error("Expected at least one alternative method")
			}
		})
	}
}

func TestMethodSelector_RecommendHybridMethod(t *testing.T) {
	matcher := NewUniversalMatcher(true)
	selector := NewMethodSelector(matcher)

	testCases := []struct {
		s1, s2 string
		desc   string
	}{
		{"молоток", "молотак", "Короткие строки"},
		{"кабель медный", "медный кабель", "Порядок слов"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			methods, weights, err := selector.RecommendHybridMethod(tc.s1, tc.s2)
			if err != nil {
				t.Errorf("RecommendHybridMethod failed: %v", err)
				return
			}
			if len(methods) == 0 {
				t.Error("Expected at least one method")
			}
			if len(weights) != len(methods) {
				t.Errorf("Expected %d weights, got %d", len(methods), len(weights))
			}

			// Проверяем, что веса в диапазоне [0, 1]
			totalWeight := 0.0
			for i, weight := range weights {
				if weight < 0.0 || weight > 1.0 {
					t.Errorf("Weight %d (%s) = %.3f, ожидалось [0, 1]",
						i, methods[i], weight)
				}
				totalWeight += weight
			}

			// Сумма весов должна быть близка к 1.0
			if totalWeight < 0.9 || totalWeight > 1.1 {
				t.Errorf("Total weight = %.3f, ожидалось близко к 1.0", totalWeight)
			}
		})
	}
}
