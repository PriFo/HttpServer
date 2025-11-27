package algorithms

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SimilarityMetrics предоставляет различные метрики схожести строк
type SimilarityMetrics struct{}

// NewSimilarityMetrics создает новый экземпляр метрик схожести
func NewSimilarityMetrics() *SimilarityMetrics {
	return &SimilarityMetrics{}
}

// JaccardIndex вычисляет индекс Жаккара (коэффициент Танимото)
// Возвращает значение от 0.0 до 1.0
func (sm *SimilarityMetrics) JaccardIndex(text1, text2 string) float64 {
	runeCount1 := utf8.RuneCountInString(text1)
	runeCount2 := utf8.RuneCountInString(text2)

	if runeCount1 == 0 && runeCount2 == 0 {
		return 1.0
	}
	if runeCount1 == 0 || runeCount2 == 0 {
		return 0.0
	}

	// Для коротких строк лучше работают символьные N-граммы (биграммы)
	if runeCount1 <= 20 && runeCount2 <= 20 {
		return CharacterNGramSimilarity(text1, text2, 2)
	}

	set1 := sm.stringToSet(text1)
	set2 := sm.stringToSet(text2)

	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0
	}

	// Вычисляем пересечение
	intersection := 0
	for token := range set1 {
		if set2[token] {
			intersection++
		}
	}

	// Вычисляем объединение
	union := len(set1)
	for token := range set2 {
		if !set1[token] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// CosineSimilarity вычисляет косинусное сходство
// Использует TF (term frequency) для создания векторов
func (sm *SimilarityMetrics) CosineSimilarity(text1, text2 string) float64 {
	vec1 := sm.textToVector(text1)
	vec2 := sm.textToVector(text2)

	if len(vec1) == 0 && len(vec2) == 0 {
		return 1.0
	}
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0.0
	}

	// Вычисляем скалярное произведение
	dotProduct := 0.0
	for token, freq1 := range vec1 {
		if freq2, exists := vec2[token]; exists {
			dotProduct += freq1 * freq2
		}
	}

	// Вычисляем нормы векторов
	norm1 := 0.0
	for _, freq := range vec1 {
		norm1 += freq * freq
	}
	norm1 = math.Sqrt(norm1)

	norm2 := 0.0
	for _, freq := range vec2 {
		norm2 += freq * freq
	}
	norm2 = math.Sqrt(norm2)

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (norm1 * norm2)
}

// DamerauLevenshteinDistance вычисляет расстояние Дамерау-Левенштейна
// Учитывает транспозиции (перестановки соседних символов)
func (sm *SimilarityMetrics) DamerauLevenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Создаем матрицу
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Инициализация
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Заполнение матрицы
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}

			// Стандартные операции: удаление, вставка, замена
			matrix[i][j] = min3Similarity(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)

			// Транспозиция (перестановка соседних символов)
			if i > 1 && j > 1 && r1[i-1] == r2[j-2] && r1[i-2] == r2[j-1] {
				matrix[i][j] = minSimilarity(
					matrix[i][j],
					matrix[i-2][j-2]+1, // transposition
				)
			}
		}
	}

	return matrix[len1][len2]
}

// DamerauLevenshteinSimilarity вычисляет схожесть на основе расстояния Дамерау-Левенштейна
func (sm *SimilarityMetrics) DamerauLevenshteinSimilarity(s1, s2 string) float64 {
	distance := sm.DamerauLevenshteinDistance(s1, s2)
	maxLen := len([]rune(s1))
	if len([]rune(s2)) > maxLen {
		maxLen = len([]rune(s2))
	}

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// LevenshteinDistance вычисляет стандартное расстояние Левенштейна
func (sm *SimilarityMetrics) LevenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Оптимизированный алгоритм с одним массивом
	column := make([]int, len1+1)
	for i := 1; i <= len1; i++ {
		column[i] = i
	}

	for x := 1; x <= len2; x++ {
		column[0] = x
		lastDiag := x - 1
		for y := 1; y <= len1; y++ {
			oldDiag := column[y]
			cost := 0
			if r1[y-1] != r2[x-1] {
				cost = 1
			}
			column[y] = min3Similarity(column[y]+1, column[y-1]+1, lastDiag+cost)
			lastDiag = oldDiag
		}
	}

	return column[len1]
}

// LevenshteinSimilarity вычисляет схожесть на основе расстояния Левенштейна
func (sm *SimilarityMetrics) LevenshteinSimilarity(s1, s2 string) float64 {
	distance := sm.LevenshteinDistance(s1, s2)
	maxLen := len([]rune(s1))
	if len([]rune(s2)) > maxLen {
		maxLen = len([]rune(s2))
	}

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// CombinedSimilarity вычисляет комбинированную метрику схожести
// Использует несколько алгоритмов и возвращает среднее значение
func (sm *SimilarityMetrics) CombinedSimilarity(text1, text2 string) float64 {
	// Нормализуем тексты
	norm1 := strings.ToLower(strings.TrimSpace(text1))
	norm2 := strings.ToLower(strings.TrimSpace(text2))

	if norm1 == norm2 {
		return 1.0
	}

	// Вычисляем различные метрики
	jaccard := sm.JaccardIndex(norm1, norm2)
	cosine := sm.CosineSimilarity(norm1, norm2)
	levenshtein := sm.LevenshteinSimilarity(norm1, norm2)
	damerau := sm.DamerauLevenshteinSimilarity(norm1, norm2)

	// Взвешенное среднее (можно настроить веса)
	weights := map[string]float64{
		"jaccard":     0.2,
		"cosine":      0.3,
		"levenshtein": 0.3,
		"damerau":     0.2,
	}

	combined := jaccard*weights["jaccard"] +
		cosine*weights["cosine"] +
		levenshtein*weights["levenshtein"] +
		damerau*weights["damerau"]

	return combined
}

// stringToSet преобразует строку в множество токенов (слов)
func (sm *SimilarityMetrics) stringToSet(text string) map[string]bool {
	text = strings.ToLower(strings.TrimSpace(text))
	tokens := strings.Fields(text)

	set := make(map[string]bool)
	for _, token := range tokens {
		// Удаляем знаки препинания
		cleaned := strings.TrimFunc(token, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if cleaned != "" {
			set[cleaned] = true
		}
	}

	return set
}

// textToVector преобразует текст в вектор частот токенов (TF)
func (sm *SimilarityMetrics) textToVector(text string) map[string]float64 {
	text = strings.ToLower(strings.TrimSpace(text))
	tokens := strings.Fields(text)

	// Подсчитываем частоты
	freq := make(map[string]int)
	total := 0
	for _, token := range tokens {
		cleaned := strings.TrimFunc(token, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if cleaned != "" {
			freq[cleaned]++
			total++
		}
	}

	// Нормализуем частоты (TF)
	vector := make(map[string]float64)
	if total > 0 {
		for token, count := range freq {
			vector[token] = float64(count) / float64(total)
		}
	}

	return vector
}

// minSimilarity возвращает минимальное из двух чисел
func minSimilarity(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// min3Similarity возвращает минимальное из трех чисел
func min3Similarity(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
