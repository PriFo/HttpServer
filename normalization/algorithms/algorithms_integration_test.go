package algorithms

import (
	"testing"
)

// Базовые тесты для проверки работоспособности алгоритмов
func TestLevenshteinSimilarity(t *testing.T) {
	tests := []struct {
		s1, s2   string
		expected float64
	}{
		{"кот", "кот", 1.0},
		{"кот", "котенок", 0.5},
		{"", "", 1.0},
		{"abc", "def", 0.0},
	}

	for _, tt := range tests {
		result := LevenshteinSimilarity(tt.s1, tt.s2)
		if result < tt.expected-0.1 || result > tt.expected+0.1 {
			t.Errorf("LevenshteinSimilarity(%q, %q) = %f, expected ~%f", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestJaccardIndex(t *testing.T) {
	tests := []struct {
		s1, s2   string
		expected float64
	}{
		{"кот собака", "кот собака", 1.0},
		{"кот", "собака", 0.0},
		{"кот собака", "кот", 0.5},
	}

	for _, tt := range tests {
		result := JaccardIndexSimilarity(tt.s1, tt.s2)
		if result < tt.expected-0.1 || result > tt.expected+0.1 {
			t.Errorf("JaccardIndexSimilarity(%q, %q) = %f, expected ~%f", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestRussianSoundex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"молоток", "М000"},
		{"", ""},
	}

	for _, tt := range tests {
		result := RussianSoundex(tt.input)
		if tt.expected != "" && len(result) != 4 {
			t.Errorf("RussianSoundex(%q) = %q, expected length 4", tt.input, result)
		}
		if tt.input == "" && result != "" {
			t.Errorf("RussianSoundex(%q) = %q, expected empty", tt.input, result)
		}
	}
}

func TestNGramSimilarity(t *testing.T) {
	s1 := "молоток"
	s2 := "молоток"
	result := NGramSimilarity(s1, s2, 2)
	if result < 0.8 {
		t.Errorf("NGramSimilarity(%q, %q, 2) = %f, expected >= 0.8", s1, s2, result)
	}
}

func TestPhoneticSimilarity(t *testing.T) {
	s1 := "молоток"
	s2 := "молотак" // опечатка
	
	// Тестируем разные методы
	result1 := PhoneticSimilarity(s1, s2, "soundex")
	if result1 < 0.0 || result1 > 1.0 {
		t.Errorf("PhoneticSimilarity(%q, %q, soundex) = %f, expected 0.0-1.0", s1, s2, result1)
	}
	
	result2 := PhoneticSimilarity(s1, s2, "metaphone")
	if result2 < 0.0 || result2 > 1.0 {
		t.Errorf("PhoneticSimilarity(%q, %q, metaphone) = %f, expected 0.0-1.0", s1, s2, result2)
	}
	
	result3 := PhoneticSimilarity(s1, s2, "phonetic")
	if result3 < 0.0 || result3 > 1.0 {
		t.Errorf("PhoneticSimilarity(%q, %q, phonetic) = %f, expected 0.0-1.0", s1, s2, result3)
	}
}

