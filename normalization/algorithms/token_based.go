package algorithms

import (
	"math"
	"strings"
)

// TokenBasedSimilarity вычисляет схожесть на основе общих токенов
// Поддерживает различные методы взвешивания токенов
type TokenBasedSimilarity struct {
	useStopWords    bool
	stopWords       map[string]bool
	useWeighted     bool
	usePositional   bool
	minTokenLength  int
}

// NewTokenBasedSimilarity создает новый вычислитель токен-ориентированной схожести
func NewTokenBasedSimilarity() *TokenBasedSimilarity {
	return &TokenBasedSimilarity{
		useStopWords:   false,
		stopWords:      getDefaultStopWords(),
		useWeighted:    false,
		usePositional:  false,
		minTokenLength: 2,
	}
}

// NewTokenBasedSimilarityWeighted создает вычислитель с взвешиванием токенов
func NewTokenBasedSimilarityWeighted() *TokenBasedSimilarity {
	return &TokenBasedSimilarity{
		useStopWords:   true,
		stopWords:      getDefaultStopWords(),
		useWeighted:    true,
		usePositional:  false,
		minTokenLength: 2,
	}
}

// Similarity вычисляет схожесть двух текстов на основе общих токенов
func (tb *TokenBasedSimilarity) Similarity(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}

	tokens1 := tb.tokenize(text1)
	tokens2 := tb.tokenize(text2)

	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	if tb.useWeighted {
		return tb.computeWeightedSimilarity(tokens1, tokens2)
	}

	if tb.usePositional {
		return tb.computePositionalSimilarity(tokens1, tokens2)
	}

	return tb.computeBasicSimilarity(tokens1, tokens2)
}

// computeBasicSimilarity вычисляет базовую схожесть по общим токенам
func (tb *TokenBasedSimilarity) computeBasicSimilarity(tokens1, tokens2 []string) float64 {
	set1 := make(map[string]bool)
	for _, token := range tokens1 {
		set1[token] = true
	}

	commonCount := 0
	for _, token := range tokens2 {
		if set1[token] {
			commonCount++
		}
	}

	// Используем индекс Жаккара
	union := len(set1)
	for _, token := range tokens2 {
		if !set1[token] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(commonCount) / float64(union)
}

// computeWeightedSimilarity вычисляет взвешенную схожесть
func (tb *TokenBasedSimilarity) computeWeightedSimilarity(tokens1, tokens2 []string) float64 {
	// Подсчитываем частоты токенов
	freq1 := make(map[string]int)
	freq2 := make(map[string]int)

	for _, token := range tokens1 {
		freq1[token]++
	}
	for _, token := range tokens2 {
		freq2[token]++
	}

	// Вычисляем веса токенов (обратная частота документа)
	totalTokens1 := len(tokens1)
	totalTokens2 := len(tokens2)

	weights1 := make(map[string]float64)
	weights2 := make(map[string]float64)

	for token, count := range freq1 {
		// TF * IDF (упрощенная версия)
		tf := float64(count) / float64(totalTokens1)
		idf := math.Log(2.0 / (1.0 + float64(freq2[token])))
		weights1[token] = tf * idf
	}

	for token, count := range freq2 {
		tf := float64(count) / float64(totalTokens2)
		idf := math.Log(2.0 / (1.0 + float64(freq1[token])))
		weights2[token] = tf * idf
	}

	// Вычисляем взвешенное пересечение и объединение
	weightedIntersection := 0.0
	weightedUnion := 0.0

	allTokens := make(map[string]bool)
	for token := range weights1 {
		allTokens[token] = true
	}
	for token := range weights2 {
		allTokens[token] = true
	}

	for token := range allTokens {
		weight1 := weights1[token]
		weight2 := weights2[token]

		// Пересечение: минимум весов
		weightedIntersection += math.Min(weight1, weight2)
		// Объединение: максимум весов
		weightedUnion += math.Max(weight1, weight2)
	}

	if weightedUnion == 0 {
		return 0.0
	}

	return weightedIntersection / weightedUnion
}

// computePositionalSimilarity вычисляет схожесть с учетом позиций токенов
func (tb *TokenBasedSimilarity) computePositionalSimilarity(tokens1, tokens2 []string) float64 {
	// Создаем индексы позиций токенов
	pos1 := make(map[string][]int)
	pos2 := make(map[string][]int)

	for i, token := range tokens1 {
		pos1[token] = append(pos1[token], i)
	}
	for i, token := range tokens2 {
		pos2[token] = append(pos2[token], i)
	}

	// Находим общие токены
	commonTokens := make(map[string]bool)
	for token := range pos1 {
		if _, exists := pos2[token]; exists {
			commonTokens[token] = true
		}
	}

	if len(commonTokens) == 0 {
		return 0.0
	}

	// Вычисляем схожесть позиций для общих токенов
	totalSimilarity := 0.0
	tokenCount := 0

	for token := range commonTokens {
		positions1 := pos1[token]
		positions2 := pos2[token]

		// Вычисляем минимальное расстояние между позициями
		minDist := math.MaxFloat64
		for _, p1 := range positions1 {
			for _, p2 := range positions2 {
				dist := math.Abs(float64(p1) - float64(p2))
				if dist < minDist {
					minDist = dist
				}
			}
		}

		// Нормализуем расстояние (чем ближе позиции, тем выше схожесть)
		maxLen := float64(len(tokens1))
		if float64(len(tokens2)) > maxLen {
			maxLen = float64(len(tokens2))
		}
		positionSimilarity := 1.0 - (minDist / maxLen)
		if positionSimilarity < 0 {
			positionSimilarity = 0
		}

		totalSimilarity += positionSimilarity
		tokenCount++
	}

	if tokenCount == 0 {
		return 0.0
	}

	return totalSimilarity / float64(tokenCount)
}

// tokenize разбивает текст на токены
func (tb *TokenBasedSimilarity) tokenize(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return []string{}
	}

	// Разбиваем на слова
	words := strings.Fields(text)
	tokens := make([]string, 0, len(words))

	for _, word := range words {
		// Удаляем знаки препинания
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		
		// Пропускаем короткие слова
		if len(word) < tb.minTokenLength {
			continue
		}

		// Пропускаем стоп-слова если включена фильтрация
		if tb.useStopWords && tb.stopWords[word] {
			continue
		}

		tokens = append(tokens, word)
	}

	return tokens
}

// GetCommonTokens возвращает общие токены между двумя текстами
func (tb *TokenBasedSimilarity) GetCommonTokens(text1, text2 string) []string {
	tokens1 := tb.tokenize(text1)
	tokens2 := tb.tokenize(text2)

	set1 := make(map[string]bool)
	for _, token := range tokens1 {
		set1[token] = true
	}

	common := make([]string, 0)
	seen := make(map[string]bool)
	for _, token := range tokens2 {
		if set1[token] && !seen[token] {
			common = append(common, token)
			seen[token] = true
		}
	}

	return common
}

// GetUniqueTokens возвращает уникальные токены для каждого текста
func (tb *TokenBasedSimilarity) GetUniqueTokens(text1, text2 string) ([]string, []string) {
	tokens1 := tb.tokenize(text1)
	tokens2 := tb.tokenize(text2)

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, token := range tokens1 {
		set1[token] = true
	}
	for _, token := range tokens2 {
		set2[token] = true
	}

	unique1 := make([]string, 0)
	unique2 := make([]string, 0)

	for token := range set1 {
		if !set2[token] {
			unique1 = append(unique1, token)
		}
	}

	for token := range set2 {
		if !set1[token] {
			unique2 = append(unique2, token)
		}
	}

	return unique1, unique2
}

// getDefaultStopWords возвращает список стоп-слов для русского языка
func getDefaultStopWords() map[string]bool {
	stopWords := []string{
		"и", "в", "на", "с", "для", "по", "из", "к", "от", "о",
		"а", "но", "или", "то", "что", "как", "так", "это",
		"он", "она", "оно", "они", "мы", "вы", "я", "ты",
		"быть", "был", "была", "было", "были",
		"не", "нет", "ни", "да", "же", "ли", "бы",
	}

	result := make(map[string]bool)
	for _, word := range stopWords {
		result[word] = true
	}

	return result
}

