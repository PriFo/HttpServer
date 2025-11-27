package algorithms

import "testing"

// Тесты для Soundex
func TestSoundexRU_Encode(t *testing.T) {
	soundex := NewSoundexRU()
	
	tests := []struct {
		input    string
		expected string
	}{
		{"Москва", "М821"}, // М-С(8)-К(2)-В(1) = М821 (первые 3 согласные)
		{"Петербург", "П361"}, // П-Т(3)-Р(6)-Б(1) = П361 (Р повторяется, пропускается)
		{"", ""},
	}
	
	for _, tt := range tests {
		result := soundex.Encode(tt.input)
		// Нормализуем строки для сравнения (проблема с кодировкой в выводе)
		if result != tt.expected {
			// Проверяем длину и первые символы
			if len([]rune(result)) != len([]rune(tt.expected)) {
				t.Errorf("Soundex.Encode(%q) = %q (len=%d), want %q (len=%d)", 
					tt.input, result, len([]rune(result)), tt.expected, len([]rune(tt.expected)))
			} else {
				// Сравниваем посимвольно
				resultRunes := []rune(result)
				expectedRunes := []rune(tt.expected)
				for i := 0; i < len(resultRunes) && i < len(expectedRunes); i++ {
					if resultRunes[i] != expectedRunes[i] {
						t.Errorf("Soundex.Encode(%q) = %q, want %q (diff at pos %d: %q vs %q)", 
							tt.input, result, tt.expected, i, string(resultRunes[i]), string(expectedRunes[i]))
						break
					}
				}
			}
		}
	}
}

func TestSoundexRU_Similarity(t *testing.T) {
	soundex := NewSoundexRU()
	
	similarity := soundex.Similarity("Москва", "Москва")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}
}

// Тесты для Metaphone
func TestMetaphoneRU_Encode(t *testing.T) {
	metaphone := NewMetaphoneRU()
	
	result := metaphone.Encode("Москва")
	if result == "" {
		t.Error("Metaphone.Encode should return non-empty string")
	}
}

// Тесты для N-грамм
func TestNGramGenerator_Generate(t *testing.T) {
	gen := NewNGramGenerator(2)
	
	ngrams := gen.Generate("тест")
	if len(ngrams) == 0 {
		t.Error("NGramGenerator.Generate should return non-empty map")
	}
}

func TestNGramSimilarity_Similarity(t *testing.T) {
	gen := NewNGramGenerator(2)
	
	similarity := gen.Similarity("тест", "тест")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}
}

// Тесты для Jaccard
func TestJaccardIndex_Similarity(t *testing.T) {
	jaccard := NewJaccardIndex()
	
	similarity := jaccard.Similarity("тест один", "тест один")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}
}

// Тесты для Damerau-Levenshtein
func TestDamerauLevenshtein_Distance(t *testing.T) {
	dl := NewDamerauLevenshtein()
	
	distance := dl.Distance("тест", "тест")
	if distance != 0 {
		t.Errorf("Expected distance 0 for identical strings, got %d", distance)
	}
}

func TestDamerauLevenshtein_Similarity(t *testing.T) {
	dl := NewDamerauLevenshtein()
	
	similarity := dl.Similarity("тест", "тест")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}
}

// Тесты для Cosine Similarity
func TestCosineSimilarity_Similarity(t *testing.T) {
	cosine := NewCosineSimilarity()
	
	similarity := cosine.Similarity("тест один", "тест один")
	if similarity < 0.9 {
		t.Errorf("Expected high similarity for identical strings, got %f", similarity)
	}
}

// Тесты для Token Based
func TestTokenBasedSimilarity_Similarity(t *testing.T) {
	token := NewTokenBasedSimilarity()
	
	similarity := token.Similarity("тест один", "тест один")
	if similarity < 0.9 {
		t.Errorf("Expected high similarity for identical strings, got %f", similarity)
	}
}

// Тесты для Hamming Similarity
func TestHammingSimilarity_Identical(t *testing.T) {
	similarity := HammingSimilarity("тест", "тест")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}
}

func TestHammingSimilarity_Different(t *testing.T) {
	similarity := HammingSimilarity("тест", "тест")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}
	
	// Тест с одной заменой
	similarity = HammingSimilarity("тест", "теср")
	expected := 0.75 // 3 из 4 символов совпадают
	if similarity != expected {
		t.Errorf("Expected similarity %f for strings with 1 difference, got %f", expected, similarity)
	}
}

func TestHammingSimilarity_DifferentLength(t *testing.T) {
	// Для строк разной длины должен использоваться Levenshtein
	similarity := HammingSimilarity("тест", "тест1")
	if similarity < 0.8 {
		t.Errorf("Expected similarity >= 0.8 for strings with different length, got %f", similarity)
	}
}

func TestHammingSimilarity_Empty(t *testing.T) {
	similarity := HammingSimilarity("", "")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for empty strings, got %f", similarity)
	}
}

func TestHammingSimilarity_Unicode(t *testing.T) {
	// Тест с Unicode символами
	similarity := HammingSimilarity("кабель", "кабель")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical Unicode strings, got %f", similarity)
	}
	
	similarity = HammingSimilarity("кабель", "кабелр")
	expected := 5.0 / 6.0 // 5 из 6 символов совпадают
	if similarity != expected {
		t.Errorf("Expected similarity %f for Unicode strings with 1 difference, got %f", expected, similarity)
	}
}