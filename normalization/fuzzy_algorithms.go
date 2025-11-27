package normalization

import (
	"strings"
	"unicode"
)

// FuzzyAlgorithms предоставляет различные алгоритмы нечеткого поиска для НСИ
type FuzzyAlgorithms struct{}

// NewFuzzyAlgorithms создает новый экземпляр алгоритмов нечеткого поиска
func NewFuzzyAlgorithms() *FuzzyAlgorithms {
	return &FuzzyAlgorithms{}
}

// NGramSimilarity вычисляет схожесть на основе N-грамм
// n - размер граммы (2 для bigram, 3 для trigram)
func (fa *FuzzyAlgorithms) NGramSimilarity(s1, s2 string, n int) float64 {
	if s1 == s2 {
		return 1.0
	}

	grams1 := fa.generateNGrams(s1, n)
	grams2 := fa.generateNGrams(s2, n)

	if len(grams1) == 0 && len(grams2) == 0 {
		return 1.0
	}
	if len(grams1) == 0 || len(grams2) == 0 {
		return 0.0
	}

	// Вычисляем Jaccard индекс для множеств N-грамм
	return fa.jaccardIndex(grams1, grams2)
}

// BigramSimilarity вычисляет схожесть на основе биграмм
func (fa *FuzzyAlgorithms) BigramSimilarity(s1, s2 string) float64 {
	return fa.NGramSimilarity(s1, s2, 2)
}

// TrigramSimilarity вычисляет схожесть на основе триграмм
func (fa *FuzzyAlgorithms) TrigramSimilarity(s1, s2 string) float64 {
	return fa.NGramSimilarity(s1, s2, 3)
}

// generateNGrams генерирует N-граммы из строки
func (fa *FuzzyAlgorithms) generateNGrams(text string, n int) map[string]int {
	text = strings.ToLower(strings.TrimSpace(text))
	grams := make(map[string]int)

	runes := []rune(text)
	if len(runes) < n {
		// Если строка короче n, возвращаем саму строку как грамму
		if len(runes) > 0 {
			grams[string(runes)] = 1
		}
		return grams
	}

	for i := 0; i <= len(runes)-n; i++ {
		gram := string(runes[i : i+n])
		grams[gram]++
	}

	return grams
}

// JaccardIndex вычисляет индекс Жаккара для двух множеств токенов
func (fa *FuzzyAlgorithms) JaccardIndex(s1, s2 string) float64 {
	tokens1 := fa.tokenize(s1)
	tokens2 := fa.tokenize(s2)

	return fa.jaccardIndex(tokens1, tokens2)
}

// jaccardIndex вычисляет индекс Жаккара для двух множеств
func (fa *FuzzyAlgorithms) jaccardIndex(set1, set2 map[string]int) float64 {
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0
	}

	// Подсчитываем пересечение
	intersection := 0
	for key := range set1 {
		if _, exists := set2[key]; exists {
			intersection++
		}
	}

	// Подсчитываем объединение
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// tokenize разбивает строку на токены
func (fa *FuzzyAlgorithms) tokenize(text string) map[string]int {
	text = strings.ToLower(strings.TrimSpace(text))
	tokens := make(map[string]int)

	// Разбиваем по пробелам и знакам препинания
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	for _, word := range words {
		if len(word) > 0 {
			tokens[word]++
		}
	}

	return tokens
}

// Soundex вычисляет Soundex код для русского текста
// Адаптированная версия Soundex для кириллицы
func (fa *FuzzyAlgorithms) Soundex(text string) string {
	if text == "" {
		return "0000"
	}

	text = strings.ToUpper(strings.TrimSpace(text))
	if len(text) == 0 {
		return "0000"
	}

	// Таблица кодирования для русского языка
	codeMap := map[rune]int{
		'Б': 1, 'П': 1, 'Ф': 1, 'В': 1,
		'Г': 2, 'К': 2, 'Х': 2,
		'Д': 3, 'Т': 3,
		'Л': 4,
		'М': 5, 'Н': 5,
		'Р': 6,
		'З': 7, 'С': 7, 'Ц': 7, 'Ж': 7, 'Ш': 7, 'Щ': 7, 'Ч': 7,
	}

	// Первая буква
	result := []rune{[]rune(text)[0]}
	lastCode := 0

	runes := []rune(text)
	for i := 1; i < len(runes); i++ {
		r := runes[i]
		code, exists := codeMap[r]
		if !exists {
			// Пропускаем гласные и другие символы
			continue
		}

		// Не добавляем повторяющиеся коды
		if code != lastCode && code != 0 {
			result = append(result, rune('0'+code))
			lastCode = code
		}

		// Ограничиваем длину до 4 символов
		if len(result) >= 4 {
			break
		}
	}

	// Дополняем нулями до 4 символов
	for len(result) < 4 {
		result = append(result, '0')
	}

	return string(result[:4])
}

// SoundexSimilarity вычисляет схожесть на основе Soundex кодов
func (fa *FuzzyAlgorithms) SoundexSimilarity(s1, s2 string) float64 {
	code1 := fa.Soundex(s1)
	code2 := fa.Soundex(s2)

	if code1 == code2 {
		return 1.0
	}

	// Вычисляем расстояние Хэмминга
	distance := 0
	for i := 0; i < len(code1) && i < len(code2); i++ {
		if code1[i] != code2[i] {
			distance++
		}
	}

	return 1.0 - float64(distance)/4.0
}

// Metaphone вычисляет Metaphone код для русского текста
// Улучшенная версия Soundex
func (fa *FuzzyAlgorithms) Metaphone(text string) string {
	if text == "" {
		return ""
	}

	text = strings.ToUpper(strings.TrimSpace(text))
	runes := []rune(text)

	if len(runes) == 0 {
		return ""
	}

	var result []rune
	i := 0

	// Обрабатываем первый символ
	if i < len(runes) {
		first := runes[i]
		// Пропускаем гласные в начале
		if !fa.isVowel(first) {
			result = append(result, first)
		}
		i++
	}

	// Обрабатываем остальные символы
	for i < len(runes) {
		r := runes[i]
		prev := rune(0)
		next := rune(0)

		if i > 0 {
			prev = runes[i-1]
		}
		if i < len(runes)-1 {
			next = runes[i+1]
		}

		// Пропускаем гласные (кроме первой)
		if fa.isVowel(r) {
			i++
			continue
		}

		// Правила преобразования для русского языка
		switch r {
		case 'Б', 'П':
			if prev != 'М' && prev != 'Н' {
				result = append(result, 'П')
			}
		case 'В', 'Ф':
			result = append(result, 'Ф')
		case 'Г', 'К', 'Х':
			result = append(result, 'К')
		case 'Д', 'Т':
			if next != 'Ц' && next != 'Ч' {
				result = append(result, 'Т')
			}
		case 'Ж', 'Ш', 'Щ':
			result = append(result, 'Ш')
		case 'З', 'С', 'Ц':
			result = append(result, 'С')
		case 'Ч':
			result = append(result, 'Ч')
		case 'Л':
			result = append(result, 'Л')
		case 'М':
			result = append(result, 'М')
		case 'Н':
			result = append(result, 'Н')
		case 'Р':
			result = append(result, 'Р')
		default:
			// Оставляем другие символы как есть
			if unicode.IsLetter(r) {
				result = append(result, r)
			}
		}

		i++
	}

	// Ограничиваем длину до 4 символов
	if len(result) > 4 {
		result = result[:4]
	}

	return string(result)
}

// isVowel проверяет, является ли символ гласной
func (fa *FuzzyAlgorithms) isVowel(r rune) bool {
	vowels := "АЕЁИОУЫЭЮЯ"
	return strings.ContainsRune(vowels, unicode.ToUpper(r))
}

// MetaphoneSimilarity вычисляет схожесть на основе Metaphone кодов
func (fa *FuzzyAlgorithms) MetaphoneSimilarity(s1, s2 string) float64 {
	code1 := fa.Metaphone(s1)
	code2 := fa.Metaphone(s2)

	if code1 == code2 {
		return 1.0
	}
	if len(code1) == 0 || len(code2) == 0 {
		return 0.0
	}

	// Вычисляем расстояние Левенштейна между кодами
	distance := levenshteinDistance(code1, code2)
	maxLen := max(len(code1), len(code2))

	return 1.0 - float64(distance)/float64(maxLen)
}

// DamerauLevenshteinDistance вычисляет расстояние Дамерау-Левенштейна
// Учитывает транспозиции (перестановки соседних символов)
func (fa *FuzzyAlgorithms) DamerauLevenshteinDistance(s1, s2 string) int {
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
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)

			// Учитываем транспозицию
			if i > 1 && j > 1 && r1[i-1] == r2[j-2] && r1[i-2] == r2[j-1] {
				matrix[i][j] = min3(matrix[i][j], matrix[i-2][j-2]+cost, matrix[i][j])
			}
		}
	}

	return matrix[len1][len2]
}

// DamerauLevenshteinSimilarity вычисляет схожесть на основе расстояния Дамерау-Левенштейна
func (fa *FuzzyAlgorithms) DamerauLevenshteinSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	distance := fa.DamerauLevenshteinDistance(s1, s2)
	maxLen := max(len([]rune(s1)), len([]rune(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// WeightedLevenshteinDistance вычисляет взвешенное расстояние Левенштейна
// weights - веса для операций: [deletion, insertion, substitution]
func (fa *FuzzyAlgorithms) WeightedLevenshteinDistance(s1, s2 string, weights [3]float64) float64 {
	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return float64(len2) * weights[1] // insertion
	}
	if len2 == 0 {
		return float64(len1) * weights[0] // deletion
	}

	// Создаем матрицу
	matrix := make([][]float64, len1+1)
	for i := range matrix {
		matrix[i] = make([]float64, len2+1)
	}

	// Инициализация
	for i := 0; i <= len1; i++ {
		matrix[i][0] = float64(i) * weights[0] // deletion
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = float64(j) * weights[1] // insertion
	}

	// Заполнение матрицы
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := weights[2] // substitution
			if r1[i-1] == r2[j-1] {
				cost = 0
			}

			matrix[i][j] = minFloat(
				matrix[i-1][j]+weights[0], // deletion
				matrix[i][j-1]+weights[1], // insertion
				matrix[i-1][j-1]+cost,     // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// WeightedLevenshteinSimilarity вычисляет схожесть на основе взвешенного расстояния Левенштейна
func (fa *FuzzyAlgorithms) WeightedLevenshteinSimilarity(s1, s2 string, weights [3]float64) float64 {
	if s1 == s2 {
		return 1.0
	}

	distance := fa.WeightedLevenshteinDistance(s1, s2, weights)
	maxLen := float64(maxInt(len([]rune(s1)), len([]rune(s2))))

	if maxLen == 0 {
		return 1.0
	}

	// Нормализуем расстояние
	maxWeight := weights[0]
	if weights[1] > maxWeight {
		maxWeight = weights[1]
	}
	if weights[2] > maxWeight {
		maxWeight = weights[2]
	}
	normalizedDistance := distance / (maxLen * maxWeight)

	return 1.0 - normalizedDistance
}

// CombinedSimilarity вычисляет комбинированную схожесть используя несколько алгоритмов
func (fa *FuzzyAlgorithms) CombinedSimilarity(s1, s2 string, weights SimilarityWeights) float64 {
	var similarities []float64
	var totalWeight float64

	// Levenshtein
	if weights.Levenshtein > 0 {
		lev := 1.0 - float64(levenshteinDistance(s1, s2))/float64(max(len([]rune(s1)), len([]rune(s2))))
		similarities = append(similarities, lev*weights.Levenshtein)
		totalWeight += weights.Levenshtein
	}

	// Damerau-Levenshtein
	if weights.DamerauLevenshtein > 0 {
		dl := fa.DamerauLevenshteinSimilarity(s1, s2)
		similarities = append(similarities, dl*weights.DamerauLevenshtein)
		totalWeight += weights.DamerauLevenshtein
	}

	// Bigram
	if weights.Bigram > 0 {
		bigram := fa.BigramSimilarity(s1, s2)
		similarities = append(similarities, bigram*weights.Bigram)
		totalWeight += weights.Bigram
	}

	// Trigram
	if weights.Trigram > 0 {
		trigram := fa.TrigramSimilarity(s1, s2)
		similarities = append(similarities, trigram*weights.Trigram)
		totalWeight += weights.Trigram
	}

	// Jaccard
	if weights.Jaccard > 0 {
		jaccard := fa.JaccardIndex(s1, s2)
		similarities = append(similarities, jaccard*weights.Jaccard)
		totalWeight += weights.Jaccard
	}

	// Soundex
	if weights.Soundex > 0 {
		soundex := fa.SoundexSimilarity(s1, s2)
		similarities = append(similarities, soundex*weights.Soundex)
		totalWeight += weights.Soundex
	}

	// Metaphone
	if weights.Metaphone > 0 {
		metaphone := fa.MetaphoneSimilarity(s1, s2)
		similarities = append(similarities, metaphone*weights.Metaphone)
		totalWeight += weights.Metaphone
	}

	if totalWeight == 0 {
		return 0.0
	}

	// Вычисляем взвешенное среднее
	var sum float64
	for _, sim := range similarities {
		sum += sim
	}

	return sum / totalWeight
}

// SimilarityWeights веса для различных алгоритмов схожести
type SimilarityWeights struct {
	Levenshtein        float64 // Вес для алгоритма Левенштейна
	DamerauLevenshtein float64 // Вес для алгоритма Дамерау-Левенштейна
	Bigram             float64 // Вес для биграмм
	Trigram            float64 // Вес для триграмм
	Jaccard            float64 // Вес для индекса Жаккара
	Soundex            float64 // Вес для Soundex
	Metaphone          float64 // Вес для Metaphone
}

// DefaultSimilarityWeights возвращает веса по умолчанию
func DefaultSimilarityWeights() SimilarityWeights {
	return SimilarityWeights{
		Levenshtein:        0.3,
		DamerauLevenshtein: 0.2,
		Bigram:             0.2,
		Trigram:            0.1,
		Jaccard:            0.1,
		Soundex:            0.05,
		Metaphone:          0.05,
	}
}

// Вспомогательные функции

func minFloat(a, b, c float64) float64 {
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

func min3(a, b, c int) int {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
