package algorithms

import "testing"

// Тесты для TextNormalizer
func TestTextNormalizer_Normalize(t *testing.T) {
	normalizer := NewTextNormalizer(false)
	
	result := normalizer.Normalize("  Тест   Один  ")
	if result != "тест один" {
		t.Errorf("Expected normalized text 'тест один', got %q", result)
	}
}

func TestTextNormalizer_NormalizeWithStopWords(t *testing.T) {
	normalizer := NewTextNormalizer(true)
	
	result := normalizer.Normalize("тест и один")
	// После удаления стоп-слов "и" должно быть удалено
	if result == "" {
		t.Error("Normalized text should not be empty")
	}
}

// Тесты для normalizeUnicode
func TestNormalizeUnicode_Empty(t *testing.T) {
	result := normalizeUnicode("")
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestNormalizeUnicode_NoDiacritics(t *testing.T) {
	result := normalizeUnicode("тест")
	if result != "тест" {
		t.Errorf("Expected 'тест', got %q", result)
	}
}

func TestNormalizeUnicode_WithDiacritics(t *testing.T) {
	// Тест с комбинирующими диакритическими знаками
	// é может быть представлен как e + ́ (U+0301)
	text := "cafe\u0301" // café с комбинирующим знаком
	result := normalizeUnicode(text)
	
	// Должен удалить комбинирующий знак
	if len(result) != 4 { // "cafe"
		t.Errorf("Expected length 4 after removing diacritics, got %d", len(result))
	}
	
	// Проверяем, что результат содержит только "cafe"
	if result != "cafe" {
		t.Errorf("Expected 'cafe' after removing diacritics, got %q", result)
	}
}

func TestNormalizeUnicode_CombiningMarks(t *testing.T) {
	// Тест с различными комбинирующими знаками
	// U+0300-U+036F - Combining Diacritical Marks
	text := "a\u0300b\u0301c\u0302" // a с grave, b с acute, c с circumflex
	result := normalizeUnicode(text)
	
	// Должен удалить все комбинирующие знаки
	if result != "abc" {
		t.Errorf("Expected 'abc' after removing diacritics, got %q", result)
	}
}

func TestNormalizeUnicode_RussianText(t *testing.T) {
	// Тест с русским текстом (не должно быть изменений)
	result := normalizeUnicode("кабель стальной")
	if result != "кабель стальной" {
		t.Errorf("Expected 'кабель стальной', got %q", result)
	}
}

func TestNormalizeUnicode_MixedText(t *testing.T) {
	// Тест со смешанным текстом
	text := "cafe\u0301 кабель"
	result := normalizeUnicode(text)
	
	// Должен удалить диакритики из латиницы, но сохранить кириллицу
	if len(result) < 7 { // минимум "cafe кабель"
		t.Errorf("Expected normalized text with at least 7 characters, got %d", len(result))
	}
}

// Тесты для normalizeQuotes
func TestNormalizeQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"«тест»", "\"тест\""},
		{"\u201Eтест\u201C", "\"тест\""}, // „тест" (German quotes)
		{"'тест'", "'тест'"},
		{"тест", "тест"},
	}
	
	for _, tt := range tests {
		result := normalizeQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Тесты для normalizeHyphens
func TestNormalizeHyphens(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"тест—один", "тест-один"},
		{"тест–два", "тест-два"},
		{"тест−три", "тест-три"},
		{"тест-четыре", "тест-четыре"},
	}
	
	for _, tt := range tests {
		result := normalizeHyphens(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeHyphens(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Тесты для RemovePunctuation
func TestRemovePunctuation(t *testing.T) {
	result := RemovePunctuation("тест, один!")
	if result != "тест один" {
		t.Errorf("Expected 'тест один', got %q", result)
	}
}

// Тесты для RemoveNumbers
func TestRemoveNumbers(t *testing.T) {
	result := RemoveNumbers("тест123один456")
	if result != "тестодин" {
		t.Errorf("Expected 'тестодин', got %q", result)
	}
}

// Тесты для NormalizeWhitespace
func TestNormalizeWhitespace(t *testing.T) {
	result := NormalizeWhitespace("тест\tодин\nдва\rтри")
	if result != "тест один два три" {
		t.Errorf("Expected 'тест один два три', got %q", result)
	}
}

