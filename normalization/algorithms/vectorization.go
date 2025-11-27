package algorithms

import (
	"math"
	"strings"
	"unicode"
)

// TFIDFVectorizer создает TF-IDF векторы для корпуса текстов
type TFIDFVectorizer struct {
	vocabulary map[string]int
	idf        map[string]float64
	docCount   int
}

// NewTFIDFVectorizer создает новый TF-IDF векторизатор
func NewTFIDFVectorizer() *TFIDFVectorizer {
	return &TFIDFVectorizer{
		vocabulary: make(map[string]int),
		idf:        make(map[string]float64),
	}
}

// Fit обучает векторизатор на корпусе документов
func (tf *TFIDFVectorizer) Fit(corpus []string) {
	tf.docCount = len(corpus)
	docFreq := make(map[string]int)

	// Подсчитываем частоту документов для каждого термина
	for _, doc := range corpus {
		tokens := tokenizeDocument(doc)
		uniqueTokens := make(map[string]bool)

		for _, token := range tokens {
			if !uniqueTokens[token] {
				docFreq[token]++
				uniqueTokens[token] = true
			}
		}
	}

	// Вычисляем IDF
	for term, freq := range docFreq {
		tf.idf[term] = math.Log(float64(tf.docCount) / float64(freq))
		tf.vocabulary[term] = len(tf.vocabulary)
	}
}

// Transform преобразует документ в TF-IDF вектор
func (tf *TFIDFVectorizer) Transform(doc string) map[string]float64 {
	tokens := tokenizeDocument(doc)
	termFreq := make(map[string]int)

	// Подсчитываем частоту терминов
	for _, token := range tokens {
		termFreq[token]++
	}

	// Вычисляем TF-IDF
	vector := make(map[string]float64)
	tokenCount := float64(len(tokens))

	for term, freq := range termFreq {
		if idf, exists := tf.idf[term]; exists {
			tf := float64(freq) / tokenCount
			vector[term] = tf * idf
		}
	}

	return vector
}

// FitTransform обучает и преобразует корпус
func (tf *TFIDFVectorizer) FitTransform(corpus []string) []map[string]float64 {
	tf.Fit(corpus)
	vectors := make([]map[string]float64, len(corpus))

	for i, doc := range corpus {
		vectors[i] = tf.Transform(doc)
	}

	return vectors
}

// tokenizeDocument разбивает документ на токены
func tokenizeDocument(doc string) []string {
	doc = strings.ToLower(strings.TrimSpace(doc))
	
	// Удаляем знаки пунктуации
	var builder strings.Builder
	for _, r := range doc {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			builder.WriteRune(r)
		}
	}
	doc = builder.String()

	// Разбиваем на слова
	words := strings.Fields(doc)
	tokens := make([]string, 0, len(words))

	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) >= 2 { // Игнорируем слишком короткие слова
			tokens = append(tokens, word)
		}
	}

	return tokens
}

// generateCharacterNGrams генерирует символьные N-граммы
func generateCharacterNGrams(text string, n int) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	runes := []rune(text)

	if len(runes) < n {
		return []string{}
	}

	ngrams := make([]string, 0, len(runes)-n+1)
	for i := 0; i <= len(runes)-n; i++ {
		ngram := string(runes[i : i+n])
		ngrams = append(ngrams, ngram)
	}

	return ngrams
}

// CharacterNGramVectorizer создает векторы на основе символьных N-грамм
type CharacterNGramVectorizer struct {
	n          int
	vocabulary map[string]int
}

// NewCharacterNGramVectorizer создает новый векторизатор символьных N-грамм
func NewCharacterNGramVectorizer(n int) *CharacterNGramVectorizer {
	if n < 1 {
		n = 3 // По умолчанию триграммы
	}
	return &CharacterNGramVectorizer{
		n:          n,
		vocabulary: make(map[string]int),
	}
}

// Fit обучает векторизатор на корпусе
func (cng *CharacterNGramVectorizer) Fit(corpus []string) {
	ngramSet := make(map[string]bool)

	for _, doc := range corpus {
		ngrams := generateCharacterNGrams(doc, cng.n)
		for _, ngram := range ngrams {
			ngramSet[ngram] = true
		}
	}

	// Создаем словарь
	index := 0
	for ngram := range ngramSet {
		cng.vocabulary[ngram] = index
		index++
	}
}

// Transform преобразует документ в вектор символьных N-грамм
func (cng *CharacterNGramVectorizer) Transform(doc string) map[string]float64 {
	ngrams := generateCharacterNGrams(doc, cng.n)
	vector := make(map[string]float64)

	// Подсчитываем частоту N-грамм
	ngramFreq := make(map[string]int)
	for _, ngram := range ngrams {
		ngramFreq[ngram]++
	}

	// Нормализуем частоты
	total := float64(len(ngrams))
	for ngram, freq := range ngramFreq {
		if _, exists := cng.vocabulary[ngram]; exists {
			vector[ngram] = float64(freq) / total
		}
	}

	return vector
}

// BagOfWords создает мешок слов (BoW) для документа
type BagOfWords struct {
	vocabulary map[string]int
}

// NewBagOfWords создает новый BoW векторизатор
func NewBagOfWords() *BagOfWords {
	return &BagOfWords{
		vocabulary: make(map[string]int),
	}
}

// Fit обучает BoW на корпусе
func (bow *BagOfWords) Fit(corpus []string) {
	wordSet := make(map[string]bool)

	for _, doc := range corpus {
		tokens := tokenizeDocument(doc)
		for _, token := range tokens {
			wordSet[token] = true
		}
	}

	// Создаем словарь
	index := 0
	for word := range wordSet {
		bow.vocabulary[word] = index
		index++
	}
}

// Transform преобразует документ в BoW вектор
func (bow *BagOfWords) Transform(doc string) map[string]float64 {
	tokens := tokenizeDocument(doc)
	vector := make(map[string]float64)

	// Подсчитываем частоту слов
	wordFreq := make(map[string]int)
	for _, token := range tokens {
		if _, exists := bow.vocabulary[token]; exists {
			wordFreq[token]++
		}
	}

	// Нормализуем частоты
	total := float64(len(tokens))
	for word, freq := range wordFreq {
		vector[word] = float64(freq) / total
	}

	return vector
}

// CosineSimilarityVectors вычисляет косинусное сходство между двумя векторами
func CosineSimilarityVectors(vec1, vec2 map[string]float64) float64 {
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0.0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	// Вычисляем скалярное произведение и нормы
	for term, val1 := range vec1 {
		if val2, exists := vec2[term]; exists {
			dotProduct += val1 * val2
		}
		norm1 += val1 * val1
	}

	for _, val2 := range vec2 {
		norm2 += val2 * val2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// EuclideanDistance вычисляет евклидово расстояние между векторами
func EuclideanDistance(vec1, vec2 map[string]float64) float64 {
	if len(vec1) == 0 && len(vec2) == 0 {
		return 0.0
	}

	sumSquares := 0.0
	allTerms := make(map[string]bool)

	for term := range vec1 {
		allTerms[term] = true
	}
	for term := range vec2 {
		allTerms[term] = true
	}

	for term := range allTerms {
		val1 := vec1[term]
		val2 := vec2[term]
		diff := val1 - val2
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares)
}

// NormalizeVector нормализует вектор (L2 норма)
func NormalizeVector(vec map[string]float64) map[string]float64 {
	norm := 0.0
	for _, val := range vec {
		norm += val * val
	}

	if norm == 0 {
		return vec
	}

	norm = math.Sqrt(norm)
	normalized := make(map[string]float64)
	for term, val := range vec {
		normalized[term] = val / norm
	}

	return normalized
}

