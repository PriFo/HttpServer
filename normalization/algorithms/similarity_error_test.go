package algorithms

import (
	"testing"
)

func TestSimilarityError(t *testing.T) {
	err := NewSimilarityError(ErrCodeInvalidInput, "test error", nil)
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}

	if err.Code != ErrCodeInvalidInput {
		t.Errorf("Expected code %s, got %s", ErrCodeInvalidInput, err.Code)
	}
}

func TestValidateWeights(t *testing.T) {
	t.Run("ValidWeights", func(t *testing.T) {
		weights := DefaultSimilarityWeights()
		if err := ValidateWeights(weights); err != nil {
			t.Errorf("Valid weights should not produce error: %v", err)
		}
	})

	t.Run("NilWeights", func(t *testing.T) {
		err := ValidateWeights(nil)
		if err == nil {
			t.Error("Nil weights should produce error")
		}
		if err != nil {
			// ValidateWeights возвращает ErrCodeInvalidWeights, а не ErrCodeInvalidInput
			if se, ok := err.(*SimilarityError); ok {
				if se.Code != ErrCodeInvalidWeights {
					t.Errorf("Expected ErrCodeInvalidWeights, got %s", se.Code)
				}
			} else {
				t.Error("Should be SimilarityError")
			}
		}
	})

	t.Run("NegativeWeights", func(t *testing.T) {
		weights := &SimilarityWeights{
			JaroWinkler: -0.1,
			LCS:         0.2,
			Phonetic:    0.2,
			Ngram:       0.2,
			Jaccard:     0.1,
		}
		if err := ValidateWeights(weights); err == nil {
			t.Error("Negative weights should produce error")
		}
	})
}

func TestValidateThreshold(t *testing.T) {
	t.Run("ValidThreshold", func(t *testing.T) {
		if err := ValidateThreshold(0.75); err != nil {
			t.Errorf("Valid threshold should not produce error: %v", err)
		}
	})

	t.Run("InvalidThreshold", func(t *testing.T) {
		if err := ValidateThreshold(1.5); err == nil {
			t.Error("Invalid threshold should produce error")
		}
		if err := ValidateThreshold(-0.1); err == nil {
			t.Error("Negative threshold should produce error")
		}
	})
}

func TestValidatePair(t *testing.T) {
	t.Run("ValidPair", func(t *testing.T) {
		if err := ValidatePair("test1", "test2"); err != nil {
			t.Errorf("Valid pair should not produce error: %v", err)
		}
	})

	t.Run("EmptyPair", func(t *testing.T) {
		if err := ValidatePair("", ""); err == nil {
			t.Error("Empty pair should produce error")
		}
	})
}

func TestValidatePairs(t *testing.T) {
	t.Run("ValidPairs", func(t *testing.T) {
		pairs := []SimilarityPair{
			{"test1", "test2"},
			{"test3", "test4"},
		}
		if err := ValidatePairs(pairs); err != nil {
			t.Errorf("Valid pairs should not produce error: %v", err)
		}
	})

	t.Run("EmptyPairs", func(t *testing.T) {
		pairs := []SimilarityPair{}
		if err := ValidatePairs(pairs); err == nil {
			t.Error("Empty pairs should produce error")
		}
	})
}

