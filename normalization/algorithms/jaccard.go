package algorithms

import (
	"strings"
)

// JaccardIndex вычисляет индекс Жаккара для сравнения множеств
// Индекс Жаккара = |A ∩ B| / |A ∪ B|
// Значение от 0.0 (нет общих элементов) до 1.0 (полное совпадение)
type JaccardIndex struct {
	useNGrams bool
	nGramSize int
}

// NewJaccardIndex создает новый вычислитель индекса Жаккара
func NewJaccardIndex() *JaccardIndex {
	return &JaccardIndex{
		useNGrams: false,
		nGramSize: 2,
	}
}

// NewJaccardIndexWithNGrams создает вычислитель с использованием N-грамм
func NewJaccardIndexWithNGrams(nGramSize int) *JaccardIndex {
	if nGramSize < 1 {
		nGramSize = 2
	}
	return &JaccardIndex{
		useNGrams: true,
		nGramSize: nGramSize,
	}
}

// Similarity вычисляет индекс Жаккара для двух строк
func (j *JaccardIndex) Similarity(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}

	var set1, set2 map[string]bool

	if j.useNGrams {
		// Используем N-граммы
		ngGen := NewNGramGenerator(j.nGramSize)
		ngrams1 := ngGen.Generate(text1)
		ngrams2 := ngGen.Generate(text2)

		set1 = make(map[string]bool)
		set2 = make(map[string]bool)

		for _, ngram := range ngrams1 {
			set1[ngram] = true
		}
		for _, ngram := range ngrams2 {
			set2[ngram] = true
		}
	} else {
		// Используем токены (слова)
		set1 = j.tokenizeToSet(text1)
		set2 = j.tokenizeToSet(text2)
	}

	return j.computeJaccard(set1, set2)
}

// computeJaccard вычисляет индекс Жаккара для двух множеств
func (j *JaccardIndex) computeJaccard(set1, set2 map[string]bool) float64 {
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0
	}

	// Вычисляем пересечение
	intersection := 0
	for elem := range set1 {
		if set2[elem] {
			intersection++
		}
	}

	// Вычисляем объединение
	union := len(set1)
	for elem := range set2 {
		if !set1[elem] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	// Индекс Жаккара = пересечение / объединение
	return float64(intersection) / float64(union)
}

// tokenizeToSet разбивает текст на токены и возвращает множество
func (j *JaccardIndex) tokenizeToSet(text string) map[string]bool {
	// Нормализуем: приводим к нижнему регистру
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return make(map[string]bool)
	}

	// Разбиваем на слова
	words := strings.Fields(text)
	set := make(map[string]bool)

	for _, word := range words {
		// Удаляем знаки препинания
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if word != "" {
			set[word] = true
		}
	}

	return set
}

// SimilaritySets вычисляет индекс Жаккара для двух множеств напрямую
func (j *JaccardIndex) SimilaritySets(set1, set2 map[string]bool) float64 {
	return j.computeJaccard(set1, set2)
}

// SimilarityWeighted вычисляет взвешенный индекс Жаккара
// Учитывает частоту появления элементов
func (j *JaccardIndex) SimilarityWeighted(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}

	var map1, map2 map[string]int

	if j.useNGrams {
		// Используем N-граммы с частотами
		ngGen := NewNGramGenerator(j.nGramSize)
		ngrams1 := ngGen.Generate(text1)
		ngrams2 := ngGen.Generate(text2)
		// Преобразуем []string в map[string]int с частотами
		map1 = make(map[string]int)
		map2 = make(map[string]int)
		for _, ngram := range ngrams1 {
			map1[ngram]++
		}
		for _, ngram := range ngrams2 {
			map2[ngram]++
		}
	} else {
		// Используем токены с частотами
		map1 = j.tokenizeToMap(text1)
		map2 = j.tokenizeToMap(text2)
	}

	return j.computeWeightedJaccard(map1, map2)
}

// computeWeightedJaccard вычисляет взвешенный индекс Жаккара
func (j *JaccardIndex) computeWeightedJaccard(map1, map2 map[string]int) float64 {
	if len(map1) == 0 && len(map2) == 0 {
		return 1.0
	}
	if len(map1) == 0 || len(map2) == 0 {
		return 0.0
	}

	// Вычисляем пересечение (минимум частот)
	intersection := 0
	// Вычисляем объединение (максимум частот)
	union := 0

	// Проходим по элементам первого множества
	for elem, count1 := range map1 {
		count2 := map2[elem]
		// Пересечение: минимум частот
		intersection += min(count1, count2)
		// Объединение: максимум частот
		union += max(count1, count2)
		// Удаляем из второго, чтобы не считать дважды
		delete(map2, elem)
	}

	// Добавляем оставшиеся элементы из второго множества
	for _, count2 := range map2 {
		union += count2
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// tokenizeToMap разбивает текст на токены и возвращает map с частотами
func (j *JaccardIndex) tokenizeToMap(text string) map[string]int {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return make(map[string]int)
	}

	words := strings.Fields(text)
	result := make(map[string]int)

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if word != "" {
			result[word]++
		}
	}

	return result
}

// GetCommonElements возвращает общие элементы между двумя текстами
func (j *JaccardIndex) GetCommonElements(text1, text2 string) []string {
	var set1, set2 map[string]bool

	if j.useNGrams {
		ngGen := NewNGramGenerator(j.nGramSize)
		ngrams1 := ngGen.Generate(text1)
		ngrams2 := ngGen.Generate(text2)

		set1 = make(map[string]bool)
		set2 = make(map[string]bool)

		for _, ngram := range ngrams1 {
			set1[ngram] = true
		}
		for _, ngram := range ngrams2 {
			set2[ngram] = true
		}
	} else {
		set1 = j.tokenizeToSet(text1)
		set2 = j.tokenizeToSet(text2)
	}

	common := make([]string, 0)
	for elem := range set1 {
		if set2[elem] {
			common = append(common, elem)
		}
	}

	return common
}

