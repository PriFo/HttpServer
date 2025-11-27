package algorithms

import (
	"fmt"
	"math"
	"sync"
)

// SimilarityLearner обучается на размеченных данных для оптимизации весов
type SimilarityLearner struct {
	trainingPairs []SimilarityTestPair
	weights       *SimilarityWeights
	mu            sync.RWMutex
}

// NewSimilarityLearner создает новый обучатель
func NewSimilarityLearner() *SimilarityLearner {
	return &SimilarityLearner{
		trainingPairs: make([]SimilarityTestPair, 0),
		weights:       DefaultSimilarityWeights(),
	}
}

// AddTrainingPair добавляет обучающую пару
func (sl *SimilarityLearner) AddTrainingPair(pair SimilarityTestPair) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.trainingPairs = append(sl.trainingPairs, pair)
}

// AddTrainingPairs добавляет множество обучающих пар
func (sl *SimilarityLearner) AddTrainingPairs(pairs []SimilarityTestPair) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.trainingPairs = append(sl.trainingPairs, pairs...)
}

// GetTrainingPairsCount возвращает количество обучающих пар
func (sl *SimilarityLearner) GetTrainingPairsCount() int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return len(sl.trainingPairs)
}

// OptimizeWeights оптимизирует веса на основе обучающих данных
// Использует градиентный спуск для минимизации ошибки
func (sl *SimilarityLearner) OptimizeWeights(iterations int, learningRate float64) (*SimilarityWeights, error) {
	sl.mu.RLock()
	pairs := make([]SimilarityTestPair, len(sl.trainingPairs))
	copy(pairs, sl.trainingPairs)
	sl.mu.RUnlock()

	if len(pairs) == 0 {
		return nil, NewSimilarityError(ErrCodeEmptyData, "no training pairs available", nil)
	}

	if err := ValidateTestPairs(pairs); err != nil {
		return nil, err
	}

	if iterations <= 0 {
		return nil, NewSimilarityError(ErrCodeInvalidInput, "iterations must be positive", nil).
			WithDetail("iterations", iterations)
	}

	if learningRate <= 0 {
		return nil, NewSimilarityError(ErrCodeInvalidInput, "learning rate must be positive", nil).
			WithDetail("learning_rate", learningRate)
	}

	// Начальные веса
	weights := &SimilarityWeights{
		JaroWinkler: 0.3,
		LCS:         0.2,
		Phonetic:    0.2,
		Ngram:       0.2,
		Jaccard:     0.1,
	}

	// Градиентный спуск
	for iter := 0; iter < iterations; iter++ {
		gradient := sl.computeGradient(pairs, weights)
		
		// Обновляем веса
		weights.JaroWinkler = math.Max(0, math.Min(1, weights.JaroWinkler-learningRate*gradient.JaroWinkler))
		weights.LCS = math.Max(0, math.Min(1, weights.LCS-learningRate*gradient.LCS))
		weights.Phonetic = math.Max(0, math.Min(1, weights.Phonetic-learningRate*gradient.Phonetic))
		weights.Ngram = math.Max(0, math.Min(1, weights.Ngram-learningRate*gradient.Ngram))
		weights.Jaccard = math.Max(0, math.Min(1, weights.Jaccard-learningRate*gradient.Jaccard))
		
		// Нормализуем веса
		weights.NormalizeWeights()
	}

	sl.mu.Lock()
	sl.weights = weights
	sl.mu.Unlock()

	return weights, nil
}

// computeGradient вычисляет градиент функции ошибки
func (sl *SimilarityLearner) computeGradient(pairs []SimilarityTestPair, weights *SimilarityWeights) *SimilarityWeights {
	gradient := &SimilarityWeights{}
	threshold := 0.75

	for _, pair := range pairs {
		similarity := HybridSimilarityAdvanced(pair.S1, pair.S2, weights)
		predicted := similarity >= threshold
		actual := pair.IsDuplicate

		// Вычисляем ошибку
		error := 0.0
		if predicted != actual {
			if actual {
				error = 1.0 - similarity // False negative
			} else {
				error = similarity // False positive
			}
		}

		// Вычисляем градиенты для каждого алгоритма
		if error > 0 {
			// Jaro-Winkler
			jw := JaroWinklerSimilarityAdvanced(pair.S1, pair.S2)
			gradient.JaroWinkler += error * jw

			// LCS
			lcs := LCSSimilarityAdvanced(pair.S1, pair.S2)
			gradient.LCS += error * lcs

			// Phonetic
			phoneticMatcher := NewPhoneticMatcher()
			phonetic := phoneticMatcher.Similarity(pair.S1, pair.S2)
			gradient.Phonetic += error * phonetic

			// Ngram
			ngram := NgramSimilarityAdvanced(pair.S1, pair.S2, 2)
			gradient.Ngram += error * ngram

			// Jaccard
			metrics := NewSimilarityMetrics()
			jaccard := metrics.JaccardIndex(pair.S1, pair.S2)
			gradient.Jaccard += error * jaccard
		}
	}

	// Нормализуем градиент
	count := float64(len(pairs))
	if count > 0 {
		gradient.JaroWinkler /= count
		gradient.LCS /= count
		gradient.Phonetic /= count
		gradient.Ngram /= count
		gradient.Jaccard /= count
	}

	return gradient
}

// EvaluateCurrentWeights оценивает текущие веса на тестовых данных
func (sl *SimilarityLearner) EvaluateCurrentWeights(testPairs []SimilarityTestPair, threshold float64) *EvaluationMetrics {
	sl.mu.RLock()
	weights := sl.weights
	sl.mu.RUnlock()

	algorithm := func(s1, s2 string) float64 {
		return HybridSimilarityAdvanced(s1, s2, weights)
	}

	return EvaluateAlgorithm(testPairs, threshold, algorithm)
}

// GetOptimalThreshold находит оптимальный порог для заданных весов
func (sl *SimilarityLearner) GetOptimalThreshold(testPairs []SimilarityTestPair, weights *SimilarityWeights) (float64, *EvaluationMetrics) {
	bestThreshold := 0.5
	bestF1 := 0.0
	var bestMetrics *EvaluationMetrics

	// Перебираем пороги от 0.5 до 0.95 с шагом 0.05
	for threshold := 0.5; threshold <= 0.95; threshold += 0.05 {
		algorithm := func(s1, s2 string) float64 {
			return HybridSimilarityAdvanced(s1, s2, weights)
		}
		metrics := EvaluateAlgorithm(testPairs, threshold, algorithm)
		f1 := metrics.F1Score()

		if f1 > bestF1 {
			bestF1 = f1
			bestThreshold = threshold
			bestMetrics = metrics
		}
	}

	return bestThreshold, bestMetrics
}

// CrossValidate выполняет кросс-валидацию на обучающих данных
func (sl *SimilarityLearner) CrossValidate(folds int) ([]*EvaluationMetrics, error) {
	sl.mu.RLock()
	pairs := make([]SimilarityTestPair, len(sl.trainingPairs))
	copy(pairs, sl.trainingPairs)
	sl.mu.RUnlock()

	if len(pairs) < folds {
		return nil, NewSimilarityError(ErrCodeInvalidInput,
			fmt.Sprintf("not enough training pairs for %d folds", folds), nil).
			WithDetail("pairs_count", len(pairs)).
			WithDetail("folds", folds)
	}

	if err := ValidateTestPairs(pairs); err != nil {
		return nil, err
	}

	foldSize := len(pairs) / folds
	results := make([]*EvaluationMetrics, folds)

	for i := 0; i < folds; i++ {
		// Разделяем на обучающую и тестовую выборки
		testStart := i * foldSize
		testEnd := testStart + foldSize
		if i == folds-1 {
			testEnd = len(pairs)
		}

		testPairs := pairs[testStart:testEnd]
		trainPairs := make([]SimilarityTestPair, 0)
		trainPairs = append(trainPairs, pairs[:testStart]...)
		trainPairs = append(trainPairs, pairs[testEnd:]...)

		// Обучаем на обучающей выборке
		tempLearner := NewSimilarityLearner()
		tempLearner.AddTrainingPairs(trainPairs)
		weights, err := tempLearner.OptimizeWeights(50, 0.01)
		if err != nil {
			return nil, err
		}

		// Оцениваем на тестовой выборке
		threshold := 0.75
		algorithm := func(s1, s2 string) float64 {
			return HybridSimilarityAdvanced(s1, s2, weights)
		}
		results[i] = EvaluateAlgorithm(testPairs, threshold, algorithm)
	}

	return results, nil
}

// GetAverageMetrics вычисляет средние метрики из списка
func GetAverageMetrics(metricsList []*EvaluationMetrics) *EvaluationMetrics {
	if len(metricsList) == 0 {
		return NewEvaluationMetrics()
	}

	avg := NewEvaluationMetrics()
	for _, m := range metricsList {
		avg.TruePositives += m.TruePositives
		avg.FalsePositives += m.FalsePositives
		avg.FalseNegatives += m.FalseNegatives
		avg.TrueNegatives += m.TrueNegatives
	}

	// Делим на количество фолдов
	count := len(metricsList)
	avg.TruePositives /= count
	avg.FalsePositives /= count
	avg.FalseNegatives /= count
	avg.TrueNegatives /= count

	return avg
}

// Reset очищает обучающие данные
func (sl *SimilarityLearner) Reset() {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.trainingPairs = make([]SimilarityTestPair, 0)
	sl.weights = DefaultSimilarityWeights()
}

// GetWeights возвращает текущие веса
func (sl *SimilarityLearner) GetWeights() *SimilarityWeights {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	
	// Возвращаем копию
	return &SimilarityWeights{
		JaroWinkler: sl.weights.JaroWinkler,
		LCS:         sl.weights.LCS,
		Phonetic:    sl.weights.Phonetic,
		Ngram:       sl.weights.Ngram,
		Jaccard:     sl.weights.Jaccard,
	}
}

