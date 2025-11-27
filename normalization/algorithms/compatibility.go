package algorithms

// Файл совместимости: функции-обертки для единообразного доступа к методам

// LevenshteinSimilarity вычисляет сходство на основе расстояния Левенштейна
func LevenshteinSimilarity(s1, s2 string) float64 {
	sm := NewSimilarityMetrics()
	return sm.LevenshteinSimilarity(s1, s2)
}

// DamerauLevenshteinSimilarity вычисляет сходство на основе расстояния Дамерау-Левенштейна
func DamerauLevenshteinSimilarity(s1, s2 string) float64 {
	sm := NewSimilarityMetrics()
	return sm.DamerauLevenshteinSimilarity(s1, s2)
}

// JaroSimilarity вычисляет сходство Jaro
// Использует полную реализацию JaroSimilarityAdvanced
func JaroSimilarity(s1, s2 string) float64 {
	return JaroSimilarityAdvanced(s1, s2)
}

// JaroWinklerSimilarity вычисляет сходство Jaro-Winkler
// Использует полную реализацию JaroWinklerSimilarityAdvanced
func JaroWinklerSimilarity(s1, s2 string) float64 {
	return JaroWinklerSimilarityAdvanced(s1, s2)
}

// JaccardIndexSimilarity вычисляет индекс Жаккара
func JaccardIndexSimilarity(s1, s2 string) float64 {
	sm := NewSimilarityMetrics()
	return sm.JaccardIndex(s1, s2)
}

// DiceCoefficient вычисляет коэффициент Dice (Sørensen-Dice)
// Dice = 2 * |A ∩ B| / (|A| + |B|)
func DiceCoefficient(s1, s2 string) float64 {
	jaccard := JaccardIndexSimilarity(s1, s2)
	// Преобразуем Jaccard в Dice: Dice = 2*J / (1+J)
	if jaccard == 0 {
		return 0.0
	}
	return 2.0 * jaccard / (1.0 + jaccard)
}

// LCSSimilarity вычисляет сходство на основе LCS
// Использует приближение через Jaccard для токенов
func LCSSimilarity(s1, s2 string) float64 {
	// LCS обычно близок к Jaccard для текстов
	return JaccardIndexSimilarity(s1, s2)
}

// HammingSimilarity вычисляет сходство на основе расстояния Хэмминга
// Работает только для строк одинаковой длины
func HammingSimilarity(s1, s2 string) float64 {
	r1, r2 := []rune(s1), []rune(s2)
	if len(r1) != len(r2) {
		// Для строк разной длины используем Levenshtein
		return LevenshteinSimilarity(s1, s2)
	}
	
	distance := 0
	for i := 0; i < len(r1); i++ {
		if r1[i] != r2[i] {
			distance++
		}
	}
	
	if len(r1) == 0 {
		return 1.0
	}
	
	return 1.0 - float64(distance)/float64(len(r1))
}

// CharacterNGramSimilarity вычисляет сходство на основе символьных N-грамм
func CharacterNGramSimilarity(s1, s2 string, n int) float64 {
	ngGen := NewNGramGenerator(n)
	ngrams1 := ngGen.Generate(s1)
	ngrams2 := ngGen.Generate(s2)
	
	set1 := make(map[string]bool)
	for _, ngram := range ngrams1 {
		set1[ngram] = true
	}
	
	set2 := make(map[string]bool)
	for _, ngram := range ngrams2 {
		set2[ngram] = true
	}
	
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}
	
	intersection := 0
	for ngram := range set1 {
		if set2[ngram] {
			intersection++
		}
	}
	
	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}
	
	return float64(intersection) / float64(union)
}

// CombinedNGramSimilarity вычисляет комбинированное сходство N-грамм
func CombinedNGramSimilarity(s1, s2 string, weights map[string]float64) float64 {
	if weights == nil {
		weights = map[string]float64{
			"char_bigram":  0.3,
			"char_trigram": 0.3,
			"word_bigram":  0.2,
			"word_trigram": 0.2,
		}
	}
	
	totalWeight := 0.0
	weightedSum := 0.0
	
	if w, ok := weights["char_bigram"]; ok && w > 0 {
		sim := CharacterNGramSimilarity(s1, s2, 2)
		weightedSum += sim * w
		totalWeight += w
	}
	
	if w, ok := weights["char_trigram"]; ok && w > 0 {
		sim := CharacterNGramSimilarity(s1, s2, 3)
		weightedSum += sim * w
		totalWeight += w
	}
	
	if totalWeight == 0 {
		return 0.0
	}
	
	return weightedSum / totalWeight
}

// NGramSimilarity вычисляет сходство на основе N-грамм
func NGramSimilarity(s1, s2 string, n int) float64 {
	return CharacterNGramSimilarity(s1, s2, n)
}

// WordNGramSimilarity вычисляет сходство на основе словесных N-грамм
func WordNGramSimilarity(s1, s2 string, n int) float64 {
	ngGen := NewNGramGenerator(n)
	ngrams1 := ngGen.GenerateWordNGrams(s1)
	ngrams2 := ngGen.GenerateWordNGrams(s2)
	
	set1 := make(map[string]bool)
	for _, ngram := range ngrams1 {
		set1[ngram] = true
	}
	
	set2 := make(map[string]bool)
	for _, ngram := range ngrams2 {
		set2[ngram] = true
	}
	
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}
	
	intersection := 0
	for ngram := range set1 {
		if set2[ngram] {
			intersection++
		}
	}
	
	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}
	
	return float64(intersection) / float64(union)
}

// Вспомогательные функции
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

