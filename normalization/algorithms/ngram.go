package algorithms

import (
	"strings"
	"unicode"
)

// NGramGenerator генерирует N-граммы из текста
type NGramGenerator struct {
	n int // размер граммы (2 для биграмм, 3 для триграмм)
}

// NewNGramGenerator создает новый генератор N-грамм
func NewNGramGenerator(n int) *NGramGenerator {
	if n < 1 {
		n = 2 // по умолчанию биграммы
	}
	return &NGramGenerator{n: n}
}

// Generate создает N-граммы из текста
// Добавляет padding символы в начале и конце для лучшего сравнения
func (ng *NGramGenerator) Generate(text string) []string {
	// Нормализуем текст: приводим к нижнему регистру и удаляем лишние пробелы
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return []string{}
	}

	// Добавляем padding символы
	padded := strings.Repeat("_", ng.n-1) + normalized + strings.Repeat("_", ng.n-1)

	var ngrams []string
	runes := []rune(padded)

	// Генерируем N-граммы
	for i := 0; i <= len(runes)-ng.n; i++ {
		ngram := string(runes[i : i+ng.n])
		// Пропускаем граммы, состоящие только из padding
		if strings.Trim(ngram, "_") != "" {
			ngrams = append(ngrams, ngram)
		}
	}

	return ngrams
}

// GenerateWordNGrams создает N-граммы на уровне слов
// Разбивает текст на слова и генерирует N-граммы из последовательностей слов
func (ng *NGramGenerator) GenerateWordNGrams(text string) []string {
	// Разбиваем на слова
	words := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(words) == 0 {
		return []string{}
	}

	// Фильтруем стоп-слова (можно расширить список)
	stopWords := map[string]bool{
		"и": true, "в": true, "на": true, "с": true, "по": true,
		"для": true, "от": true, "до": true, "из": true, "к": true,
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"of": true, "to": true, "in": true, "on": true, "at": true,
	}

	filteredWords := make([]string, 0, len(words))
	for _, word := range words {
		// Удаляем знаки препинания
		cleaned := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if cleaned != "" && !stopWords[cleaned] {
			filteredWords = append(filteredWords, cleaned)
		}
	}

	if len(filteredWords) == 0 {
		return []string{}
	}

	// Генерируем N-граммы из слов
	var ngrams []string
	for i := 0; i <= len(filteredWords)-ng.n; i++ {
		ngram := strings.Join(filteredWords[i:i+ng.n], " ")
		ngrams = append(ngrams, ngram)
	}

	return ngrams
}

// Similarity вычисляет схожесть двух текстов на основе N-грамм
// Возвращает значение от 0.0 до 1.0 (1.0 = идентичные)
func (ng *NGramGenerator) Similarity(text1, text2 string) float64 {
	ngrams1 := ng.Generate(text1)
	ngrams2 := ng.Generate(text2)

	if len(ngrams1) == 0 && len(ngrams2) == 0 {
		return 1.0
	}
	if len(ngrams1) == 0 || len(ngrams2) == 0 {
		return 0.0
	}

	// Создаем множества N-грамм
	set1 := make(map[string]int)
	set2 := make(map[string]int)

	for _, ngram := range ngrams1 {
		set1[ngram]++
	}
	for _, ngram := range ngrams2 {
		set2[ngram]++
	}

	// Вычисляем пересечение и объединение
	intersection := 0
	union := len(set1)

	// Считаем пересечение (берем минимум частот)
	for ngram, count1 := range set1 {
		if count2, exists := set2[ngram]; exists {
			if count1 < count2 {
				intersection += count1
			} else {
				intersection += count2
			}
		}
	}

	// Добавляем уникальные N-граммы из второго множества
	for ngram := range set2 {
		if _, exists := set1[ngram]; !exists {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	// Коэффициент Жаккара
	return float64(intersection) / float64(union)
}

// WordSimilarity вычисляет схожесть на основе N-грамм слов
func (ng *NGramGenerator) WordSimilarity(text1, text2 string) float64 {
	ngrams1 := ng.GenerateWordNGrams(text1)
	ngrams2 := ng.GenerateWordNGrams(text2)

	if len(ngrams1) == 0 && len(ngrams2) == 0 {
		return 1.0
	}
	if len(ngrams1) == 0 || len(ngrams2) == 0 {
		return 0.0
	}

	// Создаем множества
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, ngram := range ngrams1 {
		set1[ngram] = true
	}
	for _, ngram := range ngrams2 {
		set2[ngram] = true
	}

	// Вычисляем пересечение
	intersection := 0
	for ngram := range set1 {
		if set2[ngram] {
			intersection++
		}
	}

	// Вычисляем объединение
	union := len(set1)
	for ngram := range set2 {
		if !set1[ngram] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// FindSimilar находит похожие тексты из списка на основе N-грамм
// Возвращает индексы текстов с схожестью >= threshold
func (ng *NGramGenerator) FindSimilar(target string, candidates []string, threshold float64) []int {
	var similar []int
	targetNGrams := ng.Generate(target)

	if len(targetNGrams) == 0 {
		return similar
	}

	for i, candidate := range candidates {
		similarity := ng.Similarity(target, candidate)
		if similarity >= threshold {
			similar = append(similar, i)
		}
	}

	return similar
}

