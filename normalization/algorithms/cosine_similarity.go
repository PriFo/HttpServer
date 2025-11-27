package algorithms

import (
	"math"
	"strings"
)

// CosineSimilarity вычисляет косинусную близость между двумя текстами
// Использует различные методы векторизации: TF-IDF, бинарные векторы, частотные векторы
type CosineSimilarity struct {
	useTFIDF      bool
	useBinary     bool
	useFrequency  bool
	normalize     bool
}

// NewCosineSimilarity создает новый вычислитель косинусной близости
func NewCosineSimilarity() *CosineSimilarity {
	return &CosineSimilarity{
		useTFIDF:     true,
		useBinary:   false,
		useFrequency: false,
		normalize:   true,
	}
}

// NewCosineSimilarityBinary создает вычислитель с бинарными векторами
func NewCosineSimilarityBinary() *CosineSimilarity {
	return &CosineSimilarity{
		useTFIDF:     false,
		useBinary:    true,
		useFrequency: false,
		normalize:   true,
	}
}

// NewCosineSimilarityFrequency создает вычислитель с частотными векторами
func NewCosineSimilarityFrequency() *CosineSimilarity {
	return &CosineSimilarity{
		useTFIDF:     false,
		useBinary:    false,
		useFrequency: true,
		normalize:   true,
	}
}

// Similarity вычисляет косинусную близость между двумя текстами
func (cs *CosineSimilarity) Similarity(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}
	// Для идентичных строк возвращаем 1.0
	if text1 == text2 {
		return 1.0
	}

	var vec1, vec2 map[string]float64

	if cs.useTFIDF {
		// Используем TF-IDF векторы
		corpus := []string{text1, text2}
		vectors := cs.buildTFIDFVectors(corpus)
		vec1 = vectors[0]
		vec2 = vectors[1]
	} else if cs.useBinary {
		// Используем бинарные векторы (наличие/отсутствие токена)
		vec1 = cs.buildBinaryVector(text1)
		vec2 = cs.buildBinaryVector(text2)
	} else if cs.useFrequency {
		// Используем частотные векторы (TF)
		vec1 = cs.buildFrequencyVector(text1)
		vec2 = cs.buildFrequencyVector(text2)
	} else {
		// По умолчанию TF-IDF
		corpus := []string{text1, text2}
		vectors := cs.buildTFIDFVectors(corpus)
		vec1 = vectors[0]
		vec2 = vectors[1]
	}

	return cs.computeCosineSimilarity(vec1, vec2)
}

// computeCosineSimilarity вычисляет косинусную близость между двумя векторами
func (cs *CosineSimilarity) computeCosineSimilarity(vec1, vec2 map[string]float64) float64 {
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0.0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	// Вычисляем скалярное произведение и нормы
	for term, val1 := range vec1 {
		val2 := vec2[term]
		dotProduct += val1 * val2
		norm1 += val1 * val1
	}

	for term, val2 := range vec2 {
		if _, exists := vec1[term]; !exists {
			norm2 += val2 * val2
		}
	}

	norm1 = math.Sqrt(norm1)
	norm2 = math.Sqrt(norm2)

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	similarity := dotProduct / (norm1 * norm2)
	
	// Нормализуем в диапазон [0, 1]
	if cs.normalize && similarity < 0 {
		similarity = 0.0
	}

	return similarity
}

// buildTFIDFVectors строит TF-IDF векторы для корпуса текстов
func (cs *CosineSimilarity) buildTFIDFVectors(corpus []string) []map[string]float64 {
	// Подсчет частоты терминов в документах (IDF)
	docFreq := make(map[string]int)
	tokenizedDocs := make([][]string, len(corpus))

	for i, doc := range corpus {
		tokens := cs.tokenize(doc)
		tokenizedDocs[i] = tokens

		uniqueTokens := make(map[string]bool)
		for _, token := range tokens {
			uniqueTokens[token] = true
		}

		for token := range uniqueTokens {
			docFreq[token]++
		}
	}

	// Вычисление TF-IDF
	vectors := make([]map[string]float64, len(corpus))
	numDocs := float64(len(corpus))

	for i, tokens := range tokenizedDocs {
		vector := make(map[string]float64)
		termFreq := make(map[string]int)

		// TF (Term Frequency)
		for _, token := range tokens {
			termFreq[token]++
		}

		// TF-IDF
		for term, freq := range termFreq {
			tf := float64(freq) / float64(len(tokens))
			// Избегаем деления на ноль и log(1) = 0 для идентичных документов
			// Используем log((numDocs + 1) / (docFreq[term] + 1)) для сглаживания
			idf := math.Log((numDocs + 1) / (float64(docFreq[term]) + 1))
			vector[term] = tf * idf
		}

		vectors[i] = vector
	}

	return vectors
}

// buildBinaryVector строит бинарный вектор (наличие/отсутствие токена)
func (cs *CosineSimilarity) buildBinaryVector(text string) map[string]float64 {
	tokens := cs.tokenize(text)
	vector := make(map[string]float64)

	for _, token := range tokens {
		vector[token] = 1.0
	}

	return vector
}

// buildFrequencyVector строит частотный вектор (TF)
func (cs *CosineSimilarity) buildFrequencyVector(text string) map[string]float64 {
	tokens := cs.tokenize(text)
	vector := make(map[string]float64)
	totalTokens := len(tokens)

	if totalTokens == 0 {
		return vector
	}

	termFreq := make(map[string]int)
	for _, token := range tokens {
		termFreq[token]++
	}

	// Нормализуем частоты
	for term, freq := range termFreq {
		vector[term] = float64(freq) / float64(totalTokens)
	}

	return vector
}

// tokenize разбивает текст на токены
func (cs *CosineSimilarity) tokenize(text string) []string {
	// Приводим к нижнему регистру
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return []string{}
	}

	// Удаляем знаки пунктуации и разбиваем на слова
	words := strings.Fields(text)
	tokens := make([]string, 0, len(words))

	for _, word := range words {
		// Удаляем знаки препинания
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) >= 2 { // Пропускаем слишком короткие слова
			tokens = append(tokens, word)
		}
	}

	return tokens
}

// SimilarityWithNGrams вычисляет косинусную близость используя N-граммы вместо токенов
func (cs *CosineSimilarity) SimilarityWithNGrams(text1, text2 string, nGramSize int) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}

	ngGen := NewNGramGenerator(nGramSize)
	ngrams1 := ngGen.Generate(text1)
	ngrams2 := ngGen.Generate(text2)

	// Преобразуем в векторы с частотами
	vec1 := make(map[string]float64)
	vec2 := make(map[string]float64)

	for _, ngram := range ngrams1 {
		vec1[ngram]++
	}
	for _, ngram := range ngrams2 {
		vec2[ngram]++
	}

	// Нормализуем векторы
	vec1 = cs.normalizeVector(vec1)
	vec2 = cs.normalizeVector(vec2)

	return cs.computeCosineSimilarity(vec1, vec2)
}

// normalizeVector нормализует вектор (L2 норма)
func (cs *CosineSimilarity) normalizeVector(vec map[string]float64) map[string]float64 {
	norm := 0.0
	for _, val := range vec {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return vec
	}

	normalized := make(map[string]float64)
	for term, val := range vec {
		normalized[term] = val / norm
	}

	return normalized
}

// GetCommonTerms возвращает общие термины между двумя текстами
func (cs *CosineSimilarity) GetCommonTerms(text1, text2 string) []string {
	tokens1 := cs.tokenize(text1)
	tokens2 := cs.tokenize(text2)

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

