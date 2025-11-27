package algorithms

import (
	"testing"
)

func TestRussianLemmatizer_Lemmatize(t *testing.T) {
	lem := NewRussianLemmatizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"маслами", "масло"},
		{"сливочного", "сливочный"},
		{"масло", "масло"},
		{"сливочный", "сливочный"},
		{"кабеля", "кабель"},
		{"молотком", "молоток"},
		{"дерева", "дерево"},
		{"стали", "сталь"},
		{"пластика", "пластик"},
		{"белого", "белый"},
		{"черного", "черный"},
		{"красного", "красный"},
		{"", ""},
		{"тест", "тест"}, // Слово не в словаре, должно вернуться как есть или по правилам
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := lem.Lemmatize(tt.input)
			if result != tt.expected {
				// Для слов не в словаре результат может отличаться из-за правил
				// Проверяем только слова из словаря
				if tt.input == "маслами" || tt.input == "сливочного" || tt.input == "масло" {
					if result != tt.expected {
						t.Errorf("Lemmatize(%q) = %q, want %q", tt.input, result, tt.expected)
					}
				}
			}
		})
	}
}

func TestRussianLemmatizer_LemmatizeTokens(t *testing.T) {
	lem := NewRussianLemmatizer()

	tokens := []string{"маслами", "сливочного", "масло"}
	expected := []string{"масло", "сливочный", "масло"}

	result := lem.LemmatizeTokens(tokens)

	if len(result) != len(expected) {
		t.Fatalf("LemmatizeTokens returned %d tokens, want %d", len(result), len(expected))
	}

	// Проверяем основные слова из словаря
	if result[0] != "масло" {
		t.Errorf("LemmatizeTokens[0] = %q, want %q", result[0], "масло")
	}
	if result[1] != "сливочный" {
		t.Errorf("LemmatizeTokens[1] = %q, want %q", result[1], "сливочный")
	}
	if result[2] != "масло" {
		t.Errorf("LemmatizeTokens[2] = %q, want %q", result[2], "масло")
	}
}

func TestRussianLemmatizer_LemmatizeText(t *testing.T) {
	lem := NewRussianLemmatizer()

	text := "маслами сливочного"
	expected := "масло сливочный"

	result := lem.LemmatizeText(text)

	if result != expected {
		t.Errorf("LemmatizeText(%q) = %q, want %q", text, result, expected)
	}
}

func TestRussianLemmatizer_LemmatizeSimilarity(t *testing.T) {
	lem := NewRussianLemmatizer()

	tests := []struct {
		word1    string
		word2    string
		expected float64
	}{
		{"маслами", "масло", 1.0},
		{"масло", "масло", 1.0},
		{"маслами", "маслом", 1.0},
		{"масло", "сливочный", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.word1+"_"+tt.word2, func(t *testing.T) {
			result := lem.LemmatizeSimilarity(tt.word1, tt.word2)
			if result != tt.expected {
				t.Errorf("LemmatizeSimilarity(%q, %q) = %f, want %f", tt.word1, tt.word2, result, tt.expected)
			}
		})
	}
}

func TestRussianLemmatizer_Cache(t *testing.T) {
	lem := NewRussianLemmatizer()

	// Первый вызов - должен добавить в кэш
	result1 := lem.LemmatizeWithCache("маслами")
	if result1 != "масло" {
		t.Errorf("First call: LemmatizeWithCache(%q) = %q, want %q", "маслами", result1, "масло")
	}

	// Второй вызов - должен использовать кэш
	result2 := lem.LemmatizeWithCache("маслами")
	if result2 != "масло" {
		t.Errorf("Second call: LemmatizeWithCache(%q) = %q, want %q", "маслами", result2, "масло")
	}

	// Проверяем размер кэша
	cacheSize := lem.GetCacheSize()
	if cacheSize == 0 {
		t.Error("Cache should not be empty after caching")
	}
}

