package algorithms

// Функции используются напрямую из того же пакета

// HybridMatcher комбинирует несколько методов для повышения точности
type HybridMatcher struct {
	methods   []SimilarityMethod
	weights   []float64
	threshold float64
}

// SimilarityMethod представляет метод вычисления сходства
type SimilarityMethod struct {
	Name      string
	Compute   func(s1, s2 string) float64
	Weight    float64
	Threshold float64
}

// NewHybridMatcher создает новый гибридный матчер
func NewHybridMatcher(methods []SimilarityMethod, weights []float64, threshold float64) *HybridMatcher {
	// Нормализуем веса
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	normalizedWeights := make([]float64, len(weights))
	if totalWeight > 0 {
		for i, w := range weights {
			normalizedWeights[i] = w / totalWeight
		}
	}

	return &HybridMatcher{
		methods:   methods,
		weights:   normalizedWeights,
		threshold: threshold,
	}
}

// Similarity вычисляет взвешенное сходство используя все методы
func (hm *HybridMatcher) Similarity(s1, s2 string) float64 {
	if len(hm.methods) == 0 {
		return 0.0
	}

	totalSimilarity := 0.0
	for i, method := range hm.methods {
		weight := 1.0
		if i < len(hm.weights) {
			weight = hm.weights[i]
		}
		similarity := method.Compute(s1, s2)
		totalSimilarity += similarity * weight
	}

	return totalSimilarity
}

// IsMatch проверяет, являются ли строки совпадением
func (hm *HybridMatcher) IsMatch(s1, s2 string) bool {
	return hm.Similarity(s1, s2) >= hm.threshold
}

// EnsembleMatcher использует ансамбль методов с голосованием
type EnsembleMatcher struct {
	methods   []SimilarityMethod
	threshold float64
	voting    VotingStrategy
}

// VotingStrategy стратегия голосования
type VotingStrategy string

const (
	VotingMajority VotingStrategy = "majority" // Большинство голосов
	VotingAverage  VotingStrategy = "average"  // Среднее значение
	VotingMax      VotingStrategy = "max"      // Максимальное значение
	VotingMin      VotingStrategy = "min"      // Минимальное значение
)

// NewEnsembleMatcher создает новый ансамблевый матчер
func NewEnsembleMatcher(methods []SimilarityMethod, threshold float64, voting VotingStrategy) *EnsembleMatcher {
	return &EnsembleMatcher{
		methods:   methods,
		threshold: threshold,
		voting:    voting,
	}
}

// Similarity вычисляет сходство используя ансамбль методов
func (em *EnsembleMatcher) Similarity(s1, s2 string) float64 {
	if len(em.methods) == 0 {
		return 0.0
	}

	similarities := make([]float64, 0, len(em.methods))
	for _, method := range em.methods {
		sim := method.Compute(s1, s2)
		similarities = append(similarities, sim)
	}

	switch em.voting {
	case VotingAverage:
		sum := 0.0
		for _, sim := range similarities {
			sum += sim
		}
		return sum / float64(len(similarities))

	case VotingMax:
		maxSim := 0.0
		for _, sim := range similarities {
			if sim > maxSim {
				maxSim = sim
			}
		}
		return maxSim

	case VotingMin:
		minSim := 1.0
		for _, sim := range similarities {
			if sim < minSim {
				minSim = sim
			}
		}
		return minSim

	case VotingMajority:
		// Большинство методов должны дать сходство >= threshold
		votes := 0
		for _, sim := range similarities {
			if sim >= em.threshold {
				votes++
			}
		}
		majority := len(em.methods) / 2
		if votes > majority {
			// Возвращаем среднее значение положительных голосов
			sum := 0.0
			count := 0
			for _, sim := range similarities {
				if sim >= em.threshold {
					sum += sim
					count++
				}
			}
			if count > 0 {
				return sum / float64(count)
			}
		}
		return 0.0

	default:
		// По умолчанию используем среднее
		sum := 0.0
		for _, sim := range similarities {
			sum += sim
		}
		return sum / float64(len(similarities))
	}
}

// IsMatch проверяет, являются ли строки совпадением
func (em *EnsembleMatcher) IsMatch(s1, s2 string) bool {
	return em.Similarity(s1, s2) >= em.threshold
}

// AdaptiveThresholdMatcher использует адаптивные пороги в зависимости от длины строк
type AdaptiveThresholdMatcher struct {
	baseMethod SimilarityMethod
	minLength  int
	maxLength  int
	thresholds map[int]float64 // длина -> порог
}

// NewAdaptiveThresholdMatcher создает новый матчер с адаптивными порогами
func NewAdaptiveThresholdMatcher(method SimilarityMethod, minLength, maxLength int) *AdaptiveThresholdMatcher {
	atm := &AdaptiveThresholdMatcher{
		baseMethod: method,
		minLength:  minLength,
		maxLength:  maxLength,
		thresholds: make(map[int]float64),
	}

	// Устанавливаем адаптивные пороги
	// Для коротких строк нужен более высокий порог
	// Для длинных строк можно использовать более низкий порог
	for length := minLength; length <= maxLength; length++ {
		if length <= 5 {
			atm.thresholds[length] = 0.95
		} else if length <= 10 {
			atm.thresholds[length] = 0.90
		} else if length <= 20 {
			atm.thresholds[length] = 0.85
		} else {
			atm.thresholds[length] = 0.80
		}
	}

	return atm
}

// SetThreshold устанавливает порог для определенной длины
func (atm *AdaptiveThresholdMatcher) SetThreshold(length int, threshold float64) {
	atm.thresholds[length] = threshold
}

// GetThreshold получает порог для длины строки
func (atm *AdaptiveThresholdMatcher) GetThreshold(length int) float64 {
	if threshold, exists := atm.thresholds[length]; exists {
		return threshold
	}

	// Интерполяция для промежуточных значений
	if length <= atm.minLength {
		return atm.thresholds[atm.minLength]
	}
	if length >= atm.maxLength {
		return atm.thresholds[atm.maxLength]
	}

	// Линейная интерполяция
	prevLength := atm.minLength
	nextLength := atm.maxLength
	for l := range atm.thresholds {
		if l < length && l > prevLength {
			prevLength = l
		}
		if l > length && l < nextLength {
			nextLength = l
		}
	}

	prevThreshold := atm.thresholds[prevLength]
	nextThreshold := atm.thresholds[nextLength]

	// Линейная интерполяция
	ratio := float64(length-prevLength) / float64(nextLength-prevLength)
	return prevThreshold + (nextThreshold-prevThreshold)*ratio
}

// Similarity вычисляет сходство
func (atm *AdaptiveThresholdMatcher) Similarity(s1, s2 string) float64 {
	return atm.baseMethod.Compute(s1, s2)
}

// IsMatch проверяет совпадение с адаптивным порогом
func (atm *AdaptiveThresholdMatcher) IsMatch(s1, s2 string) bool {
	avgLength := (len([]rune(s1)) + len([]rune(s2))) / 2
	threshold := atm.GetThreshold(avgLength)
	similarity := atm.Similarity(s1, s2)
	return similarity >= threshold
}

// ConfidenceScore вычисляет оценку уверенности для совпадения
type ConfidenceScore struct {
	Similarity float64
	Confidence float64
	Method     string
	Reason     string
}

// ComputeConfidence вычисляет оценку уверенности на основе сходства и контекста
func ComputeConfidence(similarity float64, method string, context map[string]interface{}) ConfidenceScore {
	confidence := similarity

	// Корректируем уверенность на основе контекста
	if length1, ok := context["length1"].(int); ok {
		if length2, ok := context["length2"].(int); ok {
			// Для очень коротких строк требуем более высокую уверенность
			avgLength := (length1 + length2) / 2
			if avgLength < 5 {
				confidence *= 0.9
			} else if avgLength > 50 {
				// Для длинных строк немного снижаем требования
				confidence *= 1.05
				if confidence > 1.0 {
					confidence = 1.0
				}
			}
		}
	}

	// Корректируем на основе метода
	switch method {
	case "exact", "jaccard":
		// Эти методы более надежны
		confidence *= 1.0
	case "levenshtein", "damerau_levenshtein":
		// Эти методы могут давать ложные срабатывания
		confidence *= 0.95
	case "phonetic":
		// Фонетические методы менее точны
		confidence *= 0.90
	default:
		confidence *= 0.95
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	reason := "Base similarity"
	if confidence < similarity {
		reason = "Adjusted down based on context"
	} else if confidence > similarity {
		reason = "Adjusted up based on context"
	}

	return ConfidenceScore{
		Similarity: similarity,
		Confidence: confidence,
		Method:     method,
		Reason:     reason,
	}
}

// WeightedSimilarity вычисляет взвешенное сходство с учетом нескольких факторов
func WeightedSimilarity(s1, s2 string, methods []SimilarityMethod, weights []float64) float64 {
	if len(methods) == 0 {
		return 0.0
	}

	// Нормализуем веса
	totalWeight := 0.0
	normalizedWeights := make([]float64, len(methods))
	
	for i := range methods {
		weight := 1.0
		if i < len(weights) {
			weight = weights[i]
		}
		normalizedWeights[i] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	// Вычисляем взвешенное среднее
	weightedSum := 0.0
	for i, method := range methods {
		similarity := method.Compute(s1, s2)
		weight := normalizedWeights[i] / totalWeight
		weightedSum += similarity * weight
	}

	return weightedSum
}

// GetDefaultMethods возвращает набор методов по умолчанию
func GetDefaultMethods() []SimilarityMethod {
	return []SimilarityMethod{
		{
			Name: "levenshtein",
			Compute: func(s1, s2 string) float64 {
				return LevenshteinSimilarity(s1, s2)
			},
			Weight:    0.25,
			Threshold: 0.85,
		},
		{
			Name: "damerau_levenshtein",
			Compute: func(s1, s2 string) float64 {
				return DamerauLevenshteinSimilarity(s1, s2)
			},
			Weight:    0.25,
			Threshold: 0.85,
		},
		{
			Name: "jaro_winkler",
			Compute: func(s1, s2 string) float64 {
				return JaroWinklerSimilarity(s1, s2)
			},
			Weight:    0.15,
			Threshold: 0.80,
		},
		{
			Name: "jaccard",
			Compute: func(s1, s2 string) float64 {
				return JaccardIndexSimilarity(s1, s2)
			},
			Weight:    0.15,
			Threshold: 0.75,
		},
		{
			Name: "ngram",
			Compute: func(s1, s2 string) float64 {
				return CombinedNGramSimilarity(s1, s2, nil)
			},
			Weight:    0.10,
			Threshold: 0.70,
		},
		{
			Name: "phonetic",
			Compute: func(s1, s2 string) float64 {
				pm := NewPhoneticMatcher()
				return pm.Similarity(s1, s2)
			},
			Weight:    0.10,
			Threshold: 0.75,
		},
		{
			Name: "hamming",
			Compute: func(s1, s2 string) float64 {
				return HammingSimilarity(s1, s2)
			},
			Weight:    0.05,
			Threshold: 0.80,
		},
	}
}

// GetDefaultWeights возвращает веса по умолчанию для методов
func GetDefaultWeights() []float64 {
	// Веса для: levenshtein, jaro_winkler, jaccard, ngram, phonetic, hamming
	return []float64{0.25, 0.2, 0.2, 0.15, 0.15, 0.05}
}

// min возвращает минимальное значение
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

