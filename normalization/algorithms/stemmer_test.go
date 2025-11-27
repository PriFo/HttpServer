package algorithms

import (
	"testing"
)

func TestRussianStemmer_Stem(t *testing.T) {
	stemmer := NewRussianStemmer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "молоток - Snowball keeps it as is",
			input:    "молоток",
			expected: "молоток",
		},
		{
			name:     "молотка - Snowball removes 'а'",
			input:    "молотка",
			expected: "молотк",
		},
		{
			name:     "молотком - Snowball removes 'ом'",
			input:    "молотком",
			expected: "молотк",
		},
		{
			name:     "кабель variants",
			input:    "кабель",
			expected: "кабел",
		},
		{
			name:     "кабеля should stem to кабел",
			input:    "кабеля",
			expected: "кабел",
		},
		{
			name:     "кабелем should stem to кабел",
			input:    "кабелем",
			expected: "кабел",
		},
		{
			name:     "нормализация variants",
			input:    "нормализация",
			expected: "нормализац",
		},
		{
			name:     "нормализацию should stem similarly",
			input:    "нормализацию",
			expected: "нормализац",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "uppercase should be normalized",
			input:    "МОЛОТОК",
			expected: "молоток",
		},
		{
			name:     "mixed case",
			input:    "МоЛоТоК",
			expected: "молоток",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stemmer.Stem(tt.input)
			if result != tt.expected {
				t.Errorf("Stem(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRussianStemmer_StemTokens(t *testing.T) {
	stemmer := NewRussianStemmer()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "multiple молоток variants",
			input:    []string{"молоток", "молотка", "молотком"},
			expected: []string{"молоток", "молотк", "молотк"},
		},
		{
			name:     "mixed words",
			input:    []string{"красный", "молоток", "синий", "кабель"},
			expected: []string{"красн", "молоток", "син", "кабел"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single word",
			input:    []string{"нормализация"},
			expected: []string{"нормализац"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stemmer.StemTokens(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("StemTokens() returned %d items, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("StemTokens()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRussianStemmer_StemText(t *testing.T) {
	stemmer := NewRussianStemmer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple sentence",
			input:    "красный молоток и синий кабель",
			expected: "красн молоток и син кабел",
		},
		{
			name:     "with multiple spaces",
			input:    "красный  молоток   и синий",
			expected: "красн молоток и син",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word",
			input:    "нормализация",
			expected: "нормализац",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stemmer.StemText(tt.input)
			if result != tt.expected {
				t.Errorf("StemText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRussianStemmer_StemWithCache(t *testing.T) {
	stemmer := NewRussianStemmer()

	// First call should stem and cache
	result1 := stemmer.StemWithCache("молоток")
	if result1 != "молоток" {
		t.Errorf("First call: StemWithCache(%q) = %q, want %q", "молоток", result1, "молоток")
	}

	// Check cache size
	if stemmer.GetCacheSize() != 1 {
		t.Errorf("Cache size = %d, want 1", stemmer.GetCacheSize())
	}

	// Second call should use cache (same result)
	result2 := stemmer.StemWithCache("молоток")
	if result2 != "молоток" {
		t.Errorf("Second call: StemWithCache(%q) = %q, want %q", "молоток", result2, "молоток")
	}

	// Cache size should still be 1
	if stemmer.GetCacheSize() != 1 {
		t.Errorf("Cache size after second call = %d, want 1", stemmer.GetCacheSize())
	}

	// Clear cache
	stemmer.ClearCache()
	if stemmer.GetCacheSize() != 0 {
		t.Errorf("Cache size after clear = %d, want 0", stemmer.GetCacheSize())
	}
}

func TestRussianStemmer_StemSimilarity(t *testing.T) {
	stemmer := NewRussianStemmer()

	tests := []struct {
		name     string
		word1    string
		word2    string
		expected float64
	}{
		{
			name:     "same stem (молотка/молотком have same stem)",
			word1:    "молотка",
			word2:    "молотком",
			expected: 1.0,
		},
		{
			name:     "same stem (кабель variants)",
			word1:    "кабель",
			word2:    "кабеля",
			expected: 1.0,
		},
		{
			name:     "different stems",
			word1:    "молоток",
			word2:    "кабель",
			expected: 0.0,
		},
		{
			name:     "both empty",
			word1:    "",
			word2:    "",
			expected: 1.0,
		},
		{
			name:     "one empty",
			word1:    "молоток",
			word2:    "",
			expected: 0.0,
		},
		{
			name:     "identical words",
			word1:    "молоток",
			word2:    "молоток",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stemmer.StemSimilarity(tt.word1, tt.word2)
			if result != tt.expected {
				t.Errorf("StemSimilarity(%q, %q) = %f, want %f", tt.word1, tt.word2, result, tt.expected)
			}
		})
	}
}

func TestRussianStemmer_GetCommonStem(t *testing.T) {
	stemmer := NewRussianStemmer()

	tests := []struct {
		name     string
		words    []string
		expected string
	}{
		{
			name:     "молотка/молотком have common stem",
			words:    []string{"молотка", "молотком"},
			expected: "молотк",
		},
		{
			name:     "кабель variants have common stem",
			words:    []string{"кабель", "кабеля", "кабелем"},
			expected: "кабел",
		},
		{
			name:     "different words - no common stem",
			words:    []string{"молоток", "кабель"},
			expected: "",
		},
		{
			name:     "single word",
			words:    []string{"молоток"},
			expected: "молоток",
		},
		{
			name:     "empty slice",
			words:    []string{},
			expected: "",
		},
		{
			name:     "mixed - no common stem",
			words:    []string{"молоток", "молотка", "кабель"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stemmer.GetCommonStem(tt.words)
			if result != tt.expected {
				t.Errorf("GetCommonStem(%v) = %q, want %q", tt.words, result, tt.expected)
			}
		})
	}
}

func TestRussianStemmer_BatchStem(t *testing.T) {
	stemmer := NewRussianStemmer()

	texts := []string{
		"красный молоток",
		"синий кабель",
		"зеленый провод",
	}

	expected := []string{
		"красн молоток",
		"син кабел",
		"зелен провод",
	}

	results := stemmer.BatchStem(texts, 2)

	if len(results) != len(expected) {
		t.Errorf("BatchStem returned %d results, want %d", len(results), len(expected))
		return
	}

	for i := range results {
		if results[i] != expected[i] {
			t.Errorf("BatchStem()[%d] = %q, want %q", i, results[i], expected[i])
		}
	}
}

func TestRussianStemmer_WithoutCache(t *testing.T) {
	stemmer := NewRussianStemmerWithoutCache()

	result := stemmer.StemWithCache("молоток")
	if result != "молоток" {
		t.Errorf("StemWithCache(%q) = %q, want %q", "молоток", result, "молоток")
	}

	// Cache should not be used
	if stemmer.GetCacheSize() != 0 {
		t.Errorf("Cache size = %d, want 0 (cache disabled)", stemmer.GetCacheSize())
	}
}

// Benchmark tests
func BenchmarkRussianStemmer_Stem(b *testing.B) {
	stemmer := NewRussianStemmer()
	word := "нормализация"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stemmer.Stem(word)
	}
}

func BenchmarkRussianStemmer_StemWithCache(b *testing.B) {
	stemmer := NewRussianStemmer()
	word := "нормализация"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stemmer.StemWithCache(word)
	}
}

func BenchmarkRussianStemmer_StemTokens(b *testing.B) {
	stemmer := NewRussianStemmer()
	tokens := []string{"красный", "молоток", "синий", "кабель", "зеленый", "провод"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stemmer.StemTokens(tokens)
	}
}

func BenchmarkRussianStemmer_StemText(b *testing.B) {
	stemmer := NewRussianStemmer()
	text := "красный молоток и синий кабель для электромонтажных работ"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stemmer.StemText(text)
	}
}
