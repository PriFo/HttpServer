package evaluation

// ConfusionMatrix представляет матрицу ошибок для бинарной классификации
type ConfusionMatrix struct {
	TruePositives  int // TP - правильно определенные дубликаты
	TrueNegatives  int // TN - правильно определенные уникальные записи
	FalsePositives int // FP - ложные срабатывания (уникальные, но определены как дубликаты)
	FalseNegatives int // FN - пропущенные дубликаты
}

// Precision вычисляет точность (Precision)
// Precision = TP / (TP + FP)
func (cm *ConfusionMatrix) Precision() float64 {
	total := cm.TruePositives + cm.FalsePositives
	if total == 0 {
		return 0.0
	}
	return float64(cm.TruePositives) / float64(total)
}

// Recall вычисляет полноту (Recall)
// Recall = TP / (TP + FN)
func (cm *ConfusionMatrix) Recall() float64 {
	total := cm.TruePositives + cm.FalseNegatives
	if total == 0 {
		return 0.0
	}
	return float64(cm.TruePositives) / float64(total)
}

// F1Score вычисляет F1-меру (гармоническое среднее точности и полноты)
// F1 = 2 * (Precision * Recall) / (Precision + Recall)
func (cm *ConfusionMatrix) F1Score() float64 {
	precision := cm.Precision()
	recall := cm.Recall()
	
	if precision+recall == 0 {
		return 0.0
	}
	
	return 2.0 * (precision * recall) / (precision + recall)
}

// FBetaScore вычисляет F-бета меру
// F-beta = (1 + beta^2) * (Precision * Recall) / (beta^2 * Precision + Recall)
func (cm *ConfusionMatrix) FBetaScore(beta float64) float64 {
	precision := cm.Precision()
	recall := cm.Recall()
	
	if precision+recall == 0 {
		return 0.0
	}
	
	betaSquared := beta * beta
	return (1.0 + betaSquared) * (precision * recall) / (betaSquared*precision + recall)
}

// Accuracy вычисляет точность классификации
// Accuracy = (TP + TN) / (TP + TN + FP + FN)
func (cm *ConfusionMatrix) Accuracy() float64 {
	total := cm.TruePositives + cm.TrueNegatives + cm.FalsePositives + cm.FalseNegatives
	if total == 0 {
		return 0.0
	}
	return float64(cm.TruePositives+cm.TrueNegatives) / float64(total)
}

// Specificity вычисляет специфичность (True Negative Rate)
// Specificity = TN / (TN + FP)
func (cm *ConfusionMatrix) Specificity() float64 {
	total := cm.TrueNegatives + cm.FalsePositives
	if total == 0 {
		return 0.0
	}
	return float64(cm.TrueNegatives) / float64(total)
}

// FalsePositiveRate вычисляет частоту ложных срабатываний
// FPR = FP / (FP + TN)
func (cm *ConfusionMatrix) FalsePositiveRate() float64 {
	total := cm.FalsePositives + cm.TrueNegatives
	if total == 0 {
		return 0.0
	}
	return float64(cm.FalsePositives) / float64(total)
}

// FalseNegativeRate вычисляет частоту пропусков
// FNR = FN / (FN + TP)
func (cm *ConfusionMatrix) FalseNegativeRate() float64 {
	total := cm.FalseNegatives + cm.TruePositives
	if total == 0 {
		return 0.0
	}
	return float64(cm.FalseNegatives) / float64(total)
}

// Total возвращает общее количество примеров
func (cm *ConfusionMatrix) Total() int {
	return cm.TruePositives + cm.TrueNegatives + cm.FalsePositives + cm.FalseNegatives
}

// Metrics содержит все метрики оценки алгоритма
type Metrics struct {
	ConfusionMatrix ConfusionMatrix
	Precision       float64
	Recall          float64
	F1Score         float64
	F2Score         float64 // F-beta с beta=2 (больше веса для Recall)
	Accuracy        float64
	Specificity     float64
	FalsePositiveRate float64
	FalseNegativeRate float64
}

// CalculateMetrics вычисляет все метрики на основе матрицы ошибок
func CalculateMetrics(cm ConfusionMatrix) Metrics {
	return Metrics{
		ConfusionMatrix:   cm,
		Precision:         cm.Precision(),
		Recall:            cm.Recall(),
		F1Score:           cm.F1Score(),
		F2Score:           cm.FBetaScore(2.0),
		Accuracy:          cm.Accuracy(),
		Specificity:       cm.Specificity(),
		FalsePositiveRate: cm.FalsePositiveRate(),
		FalseNegativeRate: cm.FalseNegativeRate(),
	}
}

// EvaluationResult результат оценки алгоритма
type EvaluationResult struct {
	AlgorithmName string
	Metrics       Metrics
	TotalTime     float64 // Время выполнения в секундах
	ItemsPerSecond float64 // Производительность
}

// CompareResults сравнивает результаты нескольких алгоритмов
func CompareResults(results []EvaluationResult) []EvaluationResult {
	// Сортируем по F1-мере (по убыванию)
	sorted := make([]EvaluationResult, len(results))
	copy(sorted, results)
	
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Metrics.F1Score < sorted[j].Metrics.F1Score {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

// BestAlgorithm возвращает лучший алгоритм по F1-мере
func BestAlgorithm(results []EvaluationResult) *EvaluationResult {
	if len(results) == 0 {
		return nil
	}
	
	best := &results[0]
	for i := 1; i < len(results); i++ {
		if results[i].Metrics.F1Score > best.Metrics.F1Score {
			best = &results[i]
		}
	}
	
	return best
}

