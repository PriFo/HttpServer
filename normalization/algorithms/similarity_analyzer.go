package algorithms

import (
	"fmt"
	"sort"
)

// SimilarityAnalysisResult результат анализа схожести
type SimilarityAnalysisResult struct {
	Pairs           []SimilarityPairResult `json:"pairs"`
	Statistics      AnalysisStatistics      `json:"statistics"`
	Recommendations []string                `json:"recommendations"`
}

// SimilarityPairResult результат сравнения пары
type SimilarityPairResult struct {
	S1          string            `json:"s1"`
	S2          string            `json:"s2"`
	Similarity  float64           `json:"similarity"`
	IsDuplicate bool              `json:"is_duplicate"`
	Breakdown   AlgorithmBreakdown `json:"breakdown"`
	Confidence  float64           `json:"confidence"`
}

// AlgorithmBreakdown разбивка по алгоритмам
type AlgorithmBreakdown struct {
	JaroWinkler float64 `json:"jaro_winkler"`
	LCS         float64 `json:"lcs"`
	Phonetic    float64 `json:"phonetic"`
	Ngram       float64 `json:"ngram"`
	Jaccard     float64 `json:"jaccard"`
}

// AnalysisStatistics статистика анализа
type AnalysisStatistics struct {
	TotalPairs        int     `json:"total_pairs"`
	DuplicatePairs    int     `json:"duplicate_pairs"`
	NonDuplicatePairs int     `json:"non_duplicate_pairs"`
	AverageSimilarity float64 `json:"average_similarity"`
	MinSimilarity     float64 `json:"min_similarity"`
	MaxSimilarity     float64 `json:"max_similarity"`
	MedianSimilarity  float64 `json:"median_similarity"`
}

// SimilarityAnalyzer анализирует результаты схожести
type SimilarityAnalyzer struct {
	weights *SimilarityWeights
}

// NewSimilarityAnalyzer создает новый анализатор
func NewSimilarityAnalyzer(weights *SimilarityWeights) *SimilarityAnalyzer {
	if weights == nil {
		weights = DefaultSimilarityWeights()
	}
	return &SimilarityAnalyzer{
		weights: weights,
	}
}

// AnalyzePairs анализирует множество пар строк
func (sa *SimilarityAnalyzer) AnalyzePairs(pairs []SimilarityPair, threshold float64) *SimilarityAnalysisResult {
	// Валидация входных данных
	if err := ValidatePairs(pairs); err != nil {
		// Возвращаем пустой результат с ошибкой в рекомендациях
		return &SimilarityAnalysisResult{
			Pairs:           []SimilarityPairResult{},
			Statistics:      AnalysisStatistics{},
			Recommendations: []string{fmt.Sprintf("Error: %v", err)},
		}
	}

	if err := ValidateThreshold(threshold); err != nil {
		return &SimilarityAnalysisResult{
			Pairs:           []SimilarityPairResult{},
			Statistics:      AnalysisStatistics{},
			Recommendations: []string{fmt.Sprintf("Error: %v", err)},
		}
	}

	if err := ValidateWeights(sa.weights); err != nil {
		return &SimilarityAnalysisResult{
			Pairs:           []SimilarityPairResult{},
			Statistics:      AnalysisStatistics{},
			Recommendations: []string{fmt.Sprintf("Error: %v", err)},
		}
	}

	results := make([]SimilarityPairResult, len(pairs))
	similarities := make([]float64, len(pairs))

	for i, pair := range pairs {
		// Вычисляем общую схожесть
		similarity := HybridSimilarityAdvanced(pair.S1, pair.S2, sa.weights)

		// Вычисляем разбивку по алгоритмам
		breakdown := sa.ComputeBreakdown(pair.S1, pair.S2)

		// Определяем, является ли пара дубликатом
		isDuplicate := similarity >= threshold

		// Вычисляем уверенность (на основе близости к порогу)
		confidence := sa.computeConfidence(similarity, threshold)

		results[i] = SimilarityPairResult{
			S1:          pair.S1,
			S2:          pair.S2,
			Similarity:  similarity,
			IsDuplicate: isDuplicate,
			Breakdown:   breakdown,
			Confidence:  confidence,
		}

		similarities[i] = similarity
	}

	// Вычисляем статистику
	statistics := sa.computeStatistics(results, similarities)

	// Генерируем рекомендации
	recommendations := sa.generateRecommendations(statistics, threshold)

	return &SimilarityAnalysisResult{
		Pairs:           results,
		Statistics:      statistics,
		Recommendations: recommendations,
	}
}

// ComputeBreakdown вычисляет разбивку по алгоритмам (публичный метод)
func (sa *SimilarityAnalyzer) ComputeBreakdown(s1, s2 string) AlgorithmBreakdown {
	return AlgorithmBreakdown{
		JaroWinkler: JaroWinklerSimilarityAdvanced(s1, s2),
		LCS:         LCSSimilarityAdvanced(s1, s2),
		Phonetic: func() float64 {
			pm := NewPhoneticMatcher()
			return pm.Similarity(s1, s2)
		}(),
		Ngram: NgramSimilarityAdvanced(s1, s2, 2),
		Jaccard: func() float64 {
			metrics := NewSimilarityMetrics()
			return metrics.JaccardIndex(s1, s2)
		}(),
	}
}

// computeConfidence вычисляет уверенность в результате
func (sa *SimilarityAnalyzer) computeConfidence(similarity, threshold float64) float64 {
	// Уверенность выше, если схожесть далеко от порога
	distance := similarity - threshold
	if distance < 0 {
		distance = -distance
	}

	// Нормализуем до 0-1
	confidence := 1.0 - (distance / 0.5) // Максимальная уверенность при расстоянии > 0.5
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

// computeStatistics вычисляет статистику
func (sa *SimilarityAnalyzer) computeStatistics(results []SimilarityPairResult, similarities []float64) AnalysisStatistics {
	if len(results) == 0 {
		return AnalysisStatistics{}
	}

	// Подсчитываем дубликаты
	duplicateCount := 0
	for _, r := range results {
		if r.IsDuplicate {
			duplicateCount++
		}
	}

	// Вычисляем среднее
	sum := 0.0
	for _, s := range similarities {
		sum += s
	}
	avg := sum / float64(len(similarities))

	// Находим минимум и максимум
	min := similarities[0]
	max := similarities[0]
	for _, s := range similarities {
		if s < min {
			min = s
		}
		if s > max {
			max = s
		}
	}

	// Вычисляем медиану
	sorted := make([]float64, len(similarities))
	copy(sorted, similarities)
	sort.Float64s(sorted)
	median := 0.0
	if len(sorted) > 0 {
		if len(sorted)%2 == 0 {
			median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
		} else {
			median = sorted[len(sorted)/2]
		}
	}

	return AnalysisStatistics{
		TotalPairs:        len(results),
		DuplicatePairs:    duplicateCount,
		NonDuplicatePairs: len(results) - duplicateCount,
		AverageSimilarity: avg,
		MinSimilarity:     min,
		MaxSimilarity:     max,
		MedianSimilarity:  median,
	}
}

// generateRecommendations генерирует рекомендации на основе статистики
func (sa *SimilarityAnalyzer) generateRecommendations(stats AnalysisStatistics, threshold float64) []string {
	recommendations := make([]string, 0)

	// Анализ распределения дубликатов
	duplicateRate := float64(stats.DuplicatePairs) / float64(stats.TotalPairs)
	if duplicateRate > 0.5 {
		recommendations = append(recommendations,
			fmt.Sprintf("Высокий процент дубликатов (%.1f%%). Рекомендуется проверить качество данных.", duplicateRate*100))
	} else if duplicateRate < 0.1 {
		recommendations = append(recommendations,
			fmt.Sprintf("Низкий процент дубликатов (%.1f%%). Возможно, порог схожести слишком высок.", duplicateRate*100))
	}

	// Анализ среднего схожести
	if stats.AverageSimilarity < threshold-0.1 {
		recommendations = append(recommendations,
			fmt.Sprintf("Средняя схожесть (%.2f) значительно ниже порога (%.2f). Рассмотрите возможность снижения порога.", stats.AverageSimilarity, threshold))
	} else if stats.AverageSimilarity > threshold+0.1 {
		recommendations = append(recommendations,
			fmt.Sprintf("Средняя схожесть (%.2f) значительно выше порога (%.2f). Рассмотрите возможность повышения порога для уменьшения ложных срабатываний.", stats.AverageSimilarity, threshold))
	}

	// Анализ разброса
	spread := stats.MaxSimilarity - stats.MinSimilarity
	if spread < 0.3 {
		recommendations = append(recommendations,
			"Небольшой разброс схожести. Данные могут быть слишком однородными.")
	} else if spread > 0.8 {
		recommendations = append(recommendations,
			"Большой разброс схожести. Данные могут быть разнородными, что затрудняет обнаружение дублей.")
	}

	// Анализ медианы
	if stats.MedianSimilarity < threshold {
		recommendations = append(recommendations,
			"Медианная схожесть ниже порога. Большинство пар не являются дубликатами.")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Статистика выглядит нормально. Система работает эффективно.")
	}

	return recommendations
}

// FindSimilarPairs находит пары с схожестью выше порога
func (sa *SimilarityAnalyzer) FindSimilarPairs(pairs []SimilarityPair, threshold float64) []SimilarityPairResult {
	results := sa.AnalyzePairs(pairs, threshold)
	
	similar := make([]SimilarityPairResult, 0)
	for _, r := range results.Pairs {
		if r.IsDuplicate {
			similar = append(similar, r)
		}
	}

	// Сортируем по схожести (от большей к меньшей)
	sort.Slice(similar, func(i, j int) bool {
		return similar[i].Similarity > similar[j].Similarity
	})

	return similar
}

// CompareWeights сравнивает эффективность разных наборов весов
func (sa *SimilarityAnalyzer) CompareWeights(testPairs []SimilarityTestPair, weightsList []*SimilarityWeights, threshold float64) []WeightComparisonResult {
	results := make([]WeightComparisonResult, len(weightsList))

	for i, weights := range weightsList {
		algorithm := func(s1, s2 string) float64 {
			return HybridSimilarityAdvanced(s1, s2, weights)
		}
		metrics := EvaluateAlgorithm(testPairs, threshold, algorithm)

		results[i] = WeightComparisonResult{
			Weights:  weights,
			Metrics:  metrics,
			F1Score:  metrics.F1Score(),
			Precision: metrics.Precision(),
			Recall:    metrics.Recall(),
		}
	}

	// Сортируем по F1-score
	sort.Slice(results, func(i, j int) bool {
		return results[i].F1Score > results[j].F1Score
	})

	return results
}

// WeightComparisonResult результат сравнения весов
type WeightComparisonResult struct {
	Weights   *SimilarityWeights `json:"weights"`
	Metrics   *EvaluationMetrics  `json:"metrics"`
	F1Score   float64            `json:"f1_score"`
	Precision float64            `json:"precision"`
	Recall    float64            `json:"recall"`
}

