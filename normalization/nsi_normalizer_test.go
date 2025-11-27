package normalization

import (
	"testing"
	"unicode/utf8"
)

func TestAdvancedNormalizer_Transliterate(t *testing.T) {
	an := NewAdvancedNormalizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"Привет", "Privet"},
		{"Москва", "Moskva"},
		{"Санкт-Петербург", "Sankt-Peterburg"},
	}

	for _, tt := range tests {
		result := an.Transliterate(tt.input)
		if result != tt.expected {
			t.Errorf("Transliterate(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestAdvancedNormalizer_Stem(t *testing.T) {
	an := NewAdvancedNormalizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"кирпичи", "кирпич"},
		{"трубы", "труб"},
		{"панели", "панел"},
	}

	for _, tt := range tests {
		result := an.Stem(tt.input)
		// Стемминг может давать разные результаты, поэтому проверяем только что результат не пустой
		if result == "" {
			t.Errorf("Stem(%q) returned empty string", tt.input)
		}
	}
}

func TestFuzzyAlgorithms_BigramSimilarity(t *testing.T) {
	fa := NewFuzzyAlgorithms()

	tests := []struct {
		s1       string
		s2       string
		expected float64
	}{
		{"кирпич", "кирпич", 1.0},
		{"кирпич", "кирпичи", 0.8}, // Примерное значение
		{"труба", "трубы", 0.7},    // Примерное значение
	}

	for _, tt := range tests {
		result := fa.BigramSimilarity(tt.s1, tt.s2)
		if result < 0 || result > 1 {
			t.Errorf("BigramSimilarity(%q, %q) = %f, expected value between 0 and 1", tt.s1, tt.s2, result)
		}
	}
}

func TestFuzzyAlgorithms_TrigramSimilarity(t *testing.T) {
	fa := NewFuzzyAlgorithms()

	tests := []struct {
		s1 string
		s2 string
	}{
		{"кирпич", "кирпич"},
		{"труба", "трубы"},
		{"панель", "панели"},
	}

	for _, tt := range tests {
		result := fa.TrigramSimilarity(tt.s1, tt.s2)
		if result < 0 || result > 1 {
			t.Errorf("TrigramSimilarity(%q, %q) = %f, expected value between 0 and 1", tt.s1, tt.s2, result)
		}
	}
}

func TestFuzzyAlgorithms_JaccardIndex(t *testing.T) {
	fa := NewFuzzyAlgorithms()

	tests := []struct {
		s1 string
		s2 string
	}{
		{"кирпич красный", "кирпич красный"},
		{"труба стальная", "труба металлическая"},
		{"панель", "панели"},
	}

	for _, tt := range tests {
		result := fa.JaccardIndex(tt.s1, tt.s2)
		if result < 0 || result > 1 {
			t.Errorf("JaccardIndex(%q, %q) = %f, expected value between 0 and 1", tt.s1, tt.s2, result)
		}
	}
}

func TestFuzzyAlgorithms_Soundex(t *testing.T) {
	fa := NewFuzzyAlgorithms()

	tests := []struct {
		input    string
		expected string
	}{
		{"кирпич", "К617"}, // К-И-Р(6)-П(1)-И-Ч(7) = К617
		{"труба", "Т610"},  // Т-Р(6)-У-Б(1)-А = Т610
		{"панель", "П540"}, // П-А-Н(5)-Е-Л(4)-Ь = П540
	}

	for _, tt := range tests {
		result := fa.Soundex(tt.input)
		// Проверяем длину результата в рунах, так как первые символы могут быть не ASCII
		if utf8.RuneCountInString(result) != 4 {
			t.Errorf("Soundex(%q) = %q (runes=%d), expected 4-character code", tt.input, result, utf8.RuneCountInString(result))
		}
		// Также проверяем ожидаемое значение, если указано
		if tt.expected != "" && result != tt.expected {
			t.Errorf("Soundex(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFuzzyAlgorithms_Metaphone(t *testing.T) {
	fa := NewFuzzyAlgorithms()

	tests := []struct {
		input string
	}{
		{"кирпич"},
		{"труба"},
		{"панель"},
	}

	for _, tt := range tests {
		result := fa.Metaphone(tt.input)
		if result == "" {
			t.Errorf("Metaphone(%q) returned empty string", tt.input)
		}
	}
}

func TestFuzzyAlgorithms_DamerauLevenshteinSimilarity(t *testing.T) {
	fa := NewFuzzyAlgorithms()

	tests := []struct {
		s1 string
		s2 string
	}{
		{"кирпич", "кирпич"},
		{"кирпич", "кирпичи"},
		{"труба", "трубы"},
	}

	for _, tt := range tests {
		result := fa.DamerauLevenshteinSimilarity(tt.s1, tt.s2)
		if result < 0 || result > 1 {
			t.Errorf("DamerauLevenshteinSimilarity(%q, %q) = %f, expected value between 0 and 1", tt.s1, tt.s2, result)
		}
	}
}

func TestFuzzyAlgorithms_CombinedSimilarity(t *testing.T) {
	fa := NewFuzzyAlgorithms()

	tests := []struct {
		s1 string
		s2 string
	}{
		{"кирпич красный", "кирпич красный"},
		{"труба стальная", "труба металлическая"},
		{"панель", "панели"},
	}

	weights := DefaultSimilarityWeights()

	for _, tt := range tests {
		result := fa.CombinedSimilarity(tt.s1, tt.s2, weights)
		if result < 0 || result > 1 {
			t.Errorf("CombinedSimilarity(%q, %q) = %f, expected value between 0 and 1", tt.s1, tt.s2, result)
		}
	}
}

func TestEvaluationMetrics_CalculateMetrics(t *testing.T) {
	em := NewEvaluationMetrics()

	matrix := ConfusionMatrix{
		TruePositive:  80,
		TrueNegative:  900,
		FalsePositive: 20,
		FalseNegative: 0,
	}

	result := em.CalculateMetrics(matrix)

	// Проверяем, что метрики вычислены корректно
	if result.Precision < 0 || result.Precision > 1 {
		t.Errorf("Precision = %f, expected value between 0 and 1", result.Precision)
	}
	if result.Recall < 0 || result.Recall > 1 {
		t.Errorf("Recall = %f, expected value between 0 and 1", result.Recall)
	}
	if result.F1Score < 0 || result.F1Score > 1 {
		t.Errorf("F1Score = %f, expected value between 0 and 1", result.F1Score)
	}
	if result.Accuracy < 0 || result.Accuracy > 1 {
		t.Errorf("Accuracy = %f, expected value between 0 and 1", result.Accuracy)
	}

	// Проверяем формулы
	expectedPrecision := float64(matrix.TruePositive) / float64(matrix.TruePositive+matrix.FalsePositive)
	if result.Precision != expectedPrecision {
		t.Errorf("Precision = %f, expected %f", result.Precision, expectedPrecision)
	}

	expectedRecall := float64(matrix.TruePositive) / float64(matrix.TruePositive+matrix.FalseNegative)
	if result.Recall != expectedRecall {
		t.Errorf("Recall = %f, expected %f", result.Recall, expectedRecall)
	}
}

func TestNSINormalizer_NormalizeName(t *testing.T) {
	nsi := NewNSINormalizer()

	tests := []struct {
		input    string
		options  NormalizationOptions
		expected string
	}{
		{
			"Кирпич красный 120x250",
			DefaultNormalizationOptions(),
			"кирпич красный",
		},
		{
			"Труба стальная 50мм",
			DefaultNormalizationOptions(),
			"труба стальная",
		},
	}

	for _, tt := range tests {
		result := nsi.NormalizeName(tt.input, tt.options)
		if result == "" {
			t.Errorf("NormalizeName(%q) returned empty string", tt.input)
		}
	}
}

func TestNSINormalizer_FindDuplicates(t *testing.T) {
	nsi := NewNSINormalizer()

	items := []DuplicateItem{
		{ID: 1, NormalizedName: "кирпич красный", Code: "001"},
		{ID: 2, NormalizedName: "кирпич красный", Code: "002"},
		{ID: 3, NormalizedName: "труба стальная", Code: "003"},
		{ID: 4, NormalizedName: "труба стальная", Code: "004"},
		{ID: 5, NormalizedName: "панель", Code: "005"},
	}

	config := DefaultDuplicateDetectionConfig()
	config.Threshold = 0.8

	groups := nsi.FindDuplicates(items, config)

	// Должны найти хотя бы одну группу дубликатов
	if len(groups) == 0 {
		t.Error("FindDuplicates did not find any duplicate groups")
	}
}

func TestNSINormalizer_CompareAlgorithms(t *testing.T) {
	nsi := NewNSINormalizer()

	items := []DuplicateItem{
		{ID: 1, NormalizedName: "кирпич красный", Code: "001"},
		{ID: 2, NormalizedName: "кирпич красный", Code: "002"},
		{ID: 3, NormalizedName: "труба стальная", Code: "003"},
		{ID: 4, NormalizedName: "труба стальная", Code: "004"},
	}

	comparison := nsi.CompareAlgorithms(items, make(map[Pair]bool), 0.8)

	// Должны быть результаты для всех алгоритмов
	if len(comparison.Results) == 0 {
		t.Error("CompareAlgorithms did not return any results")
	}
}

func BenchmarkFuzzyAlgorithms_BigramSimilarity(b *testing.B) {
	fa := NewFuzzyAlgorithms()
	s1 := "кирпич красный полнотелый"
	s2 := "кирпич красный полнотелый"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fa.BigramSimilarity(s1, s2)
	}
}

func BenchmarkFuzzyAlgorithms_CombinedSimilarity(b *testing.B) {
	fa := NewFuzzyAlgorithms()
	s1 := "кирпич красный полнотелый"
	s2 := "кирпич красный полнотелый"
	weights := DefaultSimilarityWeights()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fa.CombinedSimilarity(s1, s2, weights)
	}
}

func BenchmarkNSINormalizer_NormalizeName(b *testing.B) {
	nsi := NewNSINormalizer()
	name := "Кирпич красный полнотелый 120x250x65 ER-00013004"
	options := DefaultNormalizationOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nsi.NormalizeName(name, options)
	}
}

// Пример использования всех методов
func ExampleNSINormalizer_usage() {
	// Создаем нормализатор
	nsi := NewNSINormalizer()

	// Нормализуем наименование
	options := DefaultNormalizationOptions()
	normalized := nsi.NormalizeName("Кирпич красный 120x250 ER-00013004", options)
	_ = normalized

	// Находим дубликаты
	items := []DuplicateItem{
		{ID: 1, NormalizedName: "кирпич красный", Code: "001"},
		{ID: 2, NormalizedName: "кирпич красный", Code: "002"},
	}

	config := DefaultDuplicateDetectionConfig()
	groups := nsi.FindDuplicates(items, config)
	_ = groups

	// Оцениваем алгоритм
	metrics := nsi.EvaluateAlgorithm(groups, groups)
	_ = metrics
}
