package algorithms

import (
	"math"
	"strings"
)

// JaroSimilarityAdvanced вычисляет сходство Jaro между двумя строками
func JaroSimilarityAdvanced(s1, s2 string) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	// Определяем окно совпадений
	matchWindow := max(len1, len2)/2 - 1
	if matchWindow < 0 {
		matchWindow = 0
	}

	// Находим совпадения
	matches1 := make([]bool, len1)
	matches2 := make([]bool, len2)
	matches := 0

	for i := 0; i < len1; i++ {
		start := max(0, i-matchWindow)
		end := min(len2, i+matchWindow+1)

		for j := start; j < end; j++ {
			if matches2[j] || r1[i] != r2[j] {
				continue
			}
			matches1[i] = true
			matches2[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	// Находим транспозиции
	transpositions := 0
	k := 0
	for i := 0; i < len1; i++ {
		if !matches1[i] {
			continue
		}
		for !matches2[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	// Вычисляем сходство Jaro
	jaro := (float64(matches)/float64(len1) +
		float64(matches)/float64(len2) +
		(float64(matches)-float64(transpositions)/2.0)/float64(matches)) / 3.0

	return jaro
}

// JaroWinklerSimilarityAdvanced вычисляет сходство Jaro-Winkler
func JaroWinklerSimilarityAdvanced(s1, s2 string) float64 {
	jaro := JaroSimilarityAdvanced(s1, s2)

	if jaro < 0.7 {
		return jaro
	}

	// Находим длину общего префикса (максимум 4)
	prefixLen := 0
	maxPrefix := 4
	r1, r2 := []rune(strings.ToLower(s1)), []rune(strings.ToLower(s2))
	minLen := min(len(r1), len(r2))

	for i := 0; i < minLen && i < maxPrefix; i++ {
		if r1[i] == r2[i] {
			prefixLen++
		} else {
			break
		}
	}

	// Коэффициент масштабирования (обычно 0.1)
	p := 0.1
	winkler := jaro + float64(prefixLen)*p*(1.0-jaro)

	return math.Min(winkler, 1.0)
}

// LCSSimilarityAdvanced вычисляет сходство на основе LCS
func LCSSimilarityAdvanced(s1, s2 string) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	lcs := LongestCommonSubsequenceAdvanced(s1, s2)
	maxLen := max(len([]rune(s1)), len([]rune(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return float64(lcs) / float64(maxLen)
}

// LongestCommonSubsequenceAdvanced вычисляет длину наибольшей общей подпоследовательности
func LongestCommonSubsequenceAdvanced(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	if len1 == 0 || len2 == 0 {
		return 0
	}

	// Создаем матрицу
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Заполняем матрицу
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			if r1[i-1] == r2[j-1] {
				matrix[i][j] = matrix[i-1][j-1] + 1
			} else {
				matrix[i][j] = max(matrix[i-1][j], matrix[i][j-1])
			}
		}
	}

	return matrix[len1][len2]
}

// HybridSimilarityAdvanced вычисляет комбинированную схожесть используя несколько алгоритмов
// Комбинирует Jaro-Winkler, LCS, фонетические алгоритмы и N-граммы для более точного результата
func HybridSimilarityAdvanced(s1, s2 string, weights *SimilarityWeights) float64 {
	if weights == nil {
		weights = &SimilarityWeights{
			JaroWinkler: 0.3,
			LCS:         0.2,
			Phonetic:    0.2,
			Ngram:       0.2,
			Jaccard:     0.1,
		}
	}

	var similarity float64

	// 1. Jaro-Winkler (хорош для опечаток и перестановок)
	if weights.JaroWinkler > 0 {
		jw := JaroWinklerSimilarityAdvanced(s1, s2)
		similarity += jw * weights.JaroWinkler
	}

	// 2. LCS (хорош для общих подпоследовательностей)
	if weights.LCS > 0 {
		lcs := LCSSimilarityAdvanced(s1, s2)
		similarity += lcs * weights.LCS
	}

	// 3. Фонетические алгоритмы (для похожих по звучанию слов)
	if weights.Phonetic > 0 {
		phoneticMatcher := NewPhoneticMatcher()
		phonetic := phoneticMatcher.Similarity(s1, s2)
		similarity += phonetic * weights.Phonetic
	}

	// 4. N-граммы (для частичных совпадений)
	if weights.Ngram > 0 {
		ngram := NgramSimilarityAdvanced(s1, s2, 2) // bigram
		similarity += ngram * weights.Ngram
	}

	// 5. Jaccard (для множеств токенов)
	if weights.Jaccard > 0 {
		metrics := NewSimilarityMetrics()
		jaccard := metrics.JaccardIndex(s1, s2)
		similarity += jaccard * weights.Jaccard
	}

	return math.Min(similarity, 1.0)
}

// SimilarityWeights веса для различных алгоритмов схожести
type SimilarityWeights struct {
	JaroWinkler float64 // Вес для Jaro-Winkler (0.0 - 1.0)
	LCS         float64 // Вес для LCS (0.0 - 1.0)
	Phonetic    float64 // Вес для фонетических алгоритмов (0.0 - 1.0)
	Ngram       float64 // Вес для N-грамм (0.0 - 1.0)
	Jaccard     float64 // Вес для Jaccard (0.0 - 1.0)
}

// DefaultSimilarityWeights возвращает веса по умолчанию
func DefaultSimilarityWeights() *SimilarityWeights {
	return &SimilarityWeights{
		JaroWinkler: 0.3,
		LCS:         0.2,
		Phonetic:    0.2,
		Ngram:       0.2,
		Jaccard:     0.1,
	}
}

// NormalizeWeights нормализует веса так, чтобы их сумма была равна 1.0
func (sw *SimilarityWeights) NormalizeWeights() {
	total := sw.JaroWinkler + sw.LCS + sw.Phonetic + sw.Ngram + sw.Jaccard
	if total == 0 {
		return
	}
	sw.JaroWinkler /= total
	sw.LCS /= total
	sw.Phonetic /= total
	sw.Ngram /= total
	sw.Jaccard /= total
}

// NgramSimilarityAdvanced вычисляет схожесть на основе N-грамм
func NgramSimilarityAdvanced(s1, s2 string, n int) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	// Генерируем N-граммы
	ngrams1 := generateNgrams(s1, n)
	ngrams2 := generateNgrams(s2, n)

	if len(ngrams1) == 0 && len(ngrams2) == 0 {
		return 1.0
	}
	if len(ngrams1) == 0 || len(ngrams2) == 0 {
		return 0.0
	}

	// Вычисляем пересечение и объединение
	intersection := 0
	union := make(map[string]bool)

	for ngram := range ngrams1 {
		union[ngram] = true
		if ngrams2[ngram] {
			intersection++
		}
	}

	for ngram := range ngrams2 {
		union[ngram] = true
	}

	if len(union) == 0 {
		return 0.0
	}

	return float64(intersection) / float64(len(union))
}

// generateNgrams генерирует N-граммы из строки
func generateNgrams(text string, n int) map[string]bool {
	ngrams := make(map[string]bool)
	runes := []rune(text)

	if len(runes) < n {
		if len(runes) > 0 {
			ngrams[string(runes)] = true
		}
		return ngrams
	}

	for i := 0; i <= len(runes)-n; i++ {
		ngram := string(runes[i : i+n])
		ngrams[ngram] = true
	}

	return ngrams
}

// AdvancedSimilarityEvaluator оценивает эффективность алгоритмов схожести
type AdvancedSimilarityEvaluator struct {
	metrics *EvaluationMetrics
}

// NewAdvancedSimilarityEvaluator создает новый оценщик
func NewAdvancedSimilarityEvaluator() *AdvancedSimilarityEvaluator {
	return &AdvancedSimilarityEvaluator{
		metrics: NewEvaluationMetrics(),
	}
}

// EvaluatePair оценивает пару строк как дубликаты
// threshold - порог схожести (0.0 - 1.0)
// actualDuplicate - является ли пара реальным дубликатом
func (evaluator *AdvancedSimilarityEvaluator) EvaluatePair(s1, s2 string, threshold float64, actualDuplicate bool) {
	// Используем гибридный метод
	weights := DefaultSimilarityWeights()
	similarity := HybridSimilarityAdvanced(s1, s2, weights)

	// Предсказываем дубликат если схожесть выше порога
	predictedDuplicate := similarity >= threshold

	// Добавляем результат в метрики
	evaluator.metrics.AddResult(predictedDuplicate, actualDuplicate)
}

// GetMetrics возвращает метрики оценки
func (evaluator *AdvancedSimilarityEvaluator) GetMetrics() *EvaluationMetrics {
	return evaluator.metrics
}

// Reset сбрасывает метрики
func (evaluator *AdvancedSimilarityEvaluator) Reset() {
	evaluator.metrics.Reset()
}

// EvaluateAlgorithm оценивает эффективность алгоритма на наборе тестовых пар
func EvaluateAlgorithm(pairs []SimilarityTestPair, threshold float64, algorithm func(string, string) float64) *EvaluationMetrics {
	metrics := NewEvaluationMetrics()

	for _, pair := range pairs {
		similarity := algorithm(pair.S1, pair.S2)
		predictedDuplicate := similarity >= threshold
		metrics.AddResult(predictedDuplicate, pair.IsDuplicate)
	}

	return metrics
}

// SimilarityTestPair тестовая пара для оценки алгоритма
type SimilarityTestPair struct {
	S1          string `json:"s1"`
	S2          string `json:"s2"`
	IsDuplicate bool   `json:"is_duplicate"`
}
