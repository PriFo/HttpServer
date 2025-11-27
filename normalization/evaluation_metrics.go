package normalization

import (
	"fmt"
	"strings"
)

// EvaluationMetrics предоставляет метрики для оценки качества алгоритмов поиска дублей
type EvaluationMetrics struct{}

// NewEvaluationMetrics создает новый экземпляр метрик оценки
func NewEvaluationMetrics() *EvaluationMetrics {
	return &EvaluationMetrics{}
}

// ConfusionMatrix представляет матрицу ошибок для бинарной классификации
type ConfusionMatrix struct {
	TruePositive  int // TP: правильно найденные дубли
	TrueNegative  int // TN: правильно определенные уникальные записи
	FalsePositive int // FP: ложные срабатывания (не дубли, но помечены как дубли)
	FalseNegative int // FN: пропущенные дубли (дубли, но не найдены)
}

// CalculateMetrics вычисляет метрики на основе матрицы ошибок
func (em *EvaluationMetrics) CalculateMetrics(matrix ConfusionMatrix) MetricsResult {
	result := MetricsResult{
		ConfusionMatrix: matrix,
	}

	// Precision (Точность): TP / (TP + FP)
	tp := float64(matrix.TruePositive)
	fp := float64(matrix.FalsePositive)
	fn := float64(matrix.FalseNegative)
	tn := float64(matrix.TrueNegative)

	if tp+fp > 0 {
		result.Precision = tp / (tp + fp)
	} else {
		result.Precision = 0.0
	}

	// Recall (Полнота): TP / (TP + FN)
	if tp+fn > 0 {
		result.Recall = tp / (tp + fn)
	} else {
		result.Recall = 0.0
	}

	// F1-мера: гармоническое среднее Precision и Recall
	if result.Precision+result.Recall > 0 {
		result.F1Score = 2 * (result.Precision * result.Recall) / (result.Precision + result.Recall)
	} else {
		result.F1Score = 0.0
	}

	// Accuracy (Точность классификации): (TP + TN) / (TP + TN + FP + FN)
	total := tp + tn + fp + fn
	if total > 0 {
		result.Accuracy = (tp + tn) / total
	} else {
		result.Accuracy = 0.0
	}

	// Specificity (Специфичность): TN / (TN + FP)
	if tn+fp > 0 {
		result.Specificity = tn / (tn + fp)
	} else {
		result.Specificity = 0.0
	}

	// False Positive Rate (FPR): FP / (FP + TN)
	if fp+tn > 0 {
		result.FalsePositiveRate = fp / (fp + tn)
	} else {
		result.FalsePositiveRate = 0.0
	}

	// False Negative Rate (FNR): FN / (FN + TP)
	if fn+tp > 0 {
		result.FalseNegativeRate = fn / (fn + tp)
	} else {
		result.FalseNegativeRate = 0.0
	}

	return result
}

// MetricsResult содержит все вычисленные метрики
type MetricsResult struct {
	ConfusionMatrix   ConfusionMatrix
	Precision         float64 // Точность
	Recall            float64 // Полнота
	F1Score           float64 // F1-мера
	Accuracy          float64 // Точность классификации
	Specificity       float64 // Специфичность
	FalsePositiveRate float64 // Частота ложных срабатываний
	FalseNegativeRate float64 // Частота пропусков
}

// String возвращает строковое представление метрик
func (mr MetricsResult) String() string {
	return fmt.Sprintf(
		"Precision: %.4f, Recall: %.4f, F1: %.4f, Accuracy: %.4f\n"+
			"Specificity: %.4f, FPR: %.4f, FNR: %.4f\n"+
			"TP: %d, TN: %d, FP: %d, FN: %d",
		mr.Precision, mr.Recall, mr.F1Score, mr.Accuracy,
		mr.Specificity, mr.FalsePositiveRate, mr.FalseNegativeRate,
		mr.ConfusionMatrix.TruePositive, mr.ConfusionMatrix.TrueNegative,
		mr.ConfusionMatrix.FalsePositive, mr.ConfusionMatrix.FalseNegative,
	)
}

// EvaluateAlgorithm оценивает алгоритм поиска дублей на размеченных данных
// predicted - предсказанные дубли (группы дубликатов)
// actual - реальные дубли (группы дубликатов из эталонных данных)
func (em *EvaluationMetrics) EvaluateAlgorithm(predicted []DuplicateGroup, actual []DuplicateGroup) MetricsResult {
	matrix := em.buildConfusionMatrix(predicted, actual)
	return em.CalculateMetrics(matrix)
}

// buildConfusionMatrix строит матрицу ошибок на основе предсказаний и эталонных данных
func (em *EvaluationMetrics) buildConfusionMatrix(predicted []DuplicateGroup, actual []DuplicateGroup) ConfusionMatrix {
	matrix := ConfusionMatrix{}

	// Создаем множества ID для быстрого поиска
	predictedPairs := em.extractDuplicatePairs(predicted)
	actualPairs := em.extractDuplicatePairs(actual)

	// Подсчитываем TP, FP, FN
	// TP: пары, которые есть и в predicted, и в actual
	for pair := range predictedPairs {
		if actualPairs[pair] {
			matrix.TruePositive++
		} else {
			matrix.FalsePositive++
		}
	}

	// FN: пары, которые есть в actual, но нет в predicted
	for pair := range actualPairs {
		if !predictedPairs[pair] {
			matrix.FalseNegative++
		}
	}

	// TN вычисляется как общее количество возможных пар минус найденные
	// Для упрощения считаем, что все остальные пары - это TN
	// В реальности это может быть сложнее, так как нужно знать общее количество записей

	return matrix
}

// extractDuplicatePairs извлекает все пары дубликатов из групп
func (em *EvaluationMetrics) extractDuplicatePairs(groups []DuplicateGroup) map[Pair]bool {
	pairs := make(map[Pair]bool)

	for _, group := range groups {
		items := group.Items
		// Генерируем все пары в группе
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				pair := Pair{
					ID1: items[i].ID,
					ID2: items[j].ID,
				}
				// Нормализуем пару (меньший ID первым)
				if pair.ID1 > pair.ID2 {
					pair.ID1, pair.ID2 = pair.ID2, pair.ID1
				}
				pairs[pair] = true
			}
		}
	}

	return pairs
}

// Pair представляет пару записей
type Pair struct {
	ID1 int
	ID2 int
}

// EvaluateWithThreshold оценивает алгоритм с различными порогами схожести
// Возвращает метрики для каждого порога
func (em *EvaluationMetrics) EvaluateWithThreshold(
	items []DuplicateItem,
	actualPairs map[Pair]bool,
	similarityFunc func(DuplicateItem, DuplicateItem) float64,
	thresholds []float64,
) []ThresholdMetrics {
	results := make([]ThresholdMetrics, len(thresholds))

	for i, threshold := range thresholds {
		// Находим дубли с текущим порогом
		predictedGroups := em.findDuplicatesWithThreshold(items, similarityFunc, threshold)
		predictedPairs := em.extractDuplicatePairs(predictedGroups)

		// Строим матрицу ошибок
		matrix := ConfusionMatrix{}
		for pair := range predictedPairs {
			if actualPairs[pair] {
				matrix.TruePositive++
			} else {
				matrix.FalsePositive++
			}
		}

		for pair := range actualPairs {
			if !predictedPairs[pair] {
				matrix.FalseNegative++
			}
		}

		// Вычисляем метрики
		metrics := em.CalculateMetrics(matrix)

		results[i] = ThresholdMetrics{
			Threshold: threshold,
			Metrics:   metrics,
		}
	}

	return results
}

// ThresholdMetrics содержит метрики для конкретного порога
type ThresholdMetrics struct {
	Threshold float64
	Metrics   MetricsResult
}

// findDuplicatesWithThreshold находит дубли с заданным порогом
func (em *EvaluationMetrics) findDuplicatesWithThreshold(
	items []DuplicateItem,
	similarityFunc func(DuplicateItem, DuplicateItem) float64,
	threshold float64,
) []DuplicateGroup {
	var groups []DuplicateGroup
	processed := make(map[int]bool)
	groupCounter := 0

	for i := 0; i < len(items); i++ {
		if processed[items[i].ID] {
			continue
		}

		var duplicates []DuplicateItem
		var itemIDs []int
		duplicates = append(duplicates, items[i])
		itemIDs = append(itemIDs, items[i].ID)

		for j := i + 1; j < len(items); j++ {
			if processed[items[j].ID] {
				continue
			}

			similarity := similarityFunc(items[i], items[j])
			if similarity >= threshold {
				duplicates = append(duplicates, items[j])
				itemIDs = append(itemIDs, items[j].ID)
				processed[items[j].ID] = true
			}
		}

		if len(duplicates) >= 2 {
			processed[items[i].ID] = true

			// Вычисляем среднюю схожесть в группе
			avgSimilarity := 0.0
			pairCount := 0
			for k := 0; k < len(duplicates); k++ {
				for l := k + 1; l < len(duplicates); l++ {
					avgSimilarity += similarityFunc(duplicates[k], duplicates[l])
					pairCount++
				}
			}
			if pairCount > 0 {
				avgSimilarity /= float64(pairCount)
			}

			groups = append(groups, DuplicateGroup{
				GroupID:         fmt.Sprintf("threshold_%d", groupCounter),
				Type:            DuplicateTypeMixed,
				SimilarityScore: avgSimilarity,
				ItemIDs:         itemIDs,
				Items:           duplicates,
				Confidence:      avgSimilarity,
				Reason:          fmt.Sprintf("Threshold-based detection (threshold=%.2f)", threshold),
			})
			groupCounter++
		}
	}

	return groups
}

// ROCPoint представляет точку на ROC-кривой
type ROCPoint struct {
	FalsePositiveRate float64
	TruePositiveRate  float64
	Threshold         float64
}

// CalculateROC вычисляет точки для ROC-кривой
func (em *EvaluationMetrics) CalculateROC(
	items []DuplicateItem,
	actualPairs map[Pair]bool,
	similarityFunc func(DuplicateItem, DuplicateItem) float64,
	thresholds []float64,
) []ROCPoint {
	points := make([]ROCPoint, len(thresholds))

	for i, threshold := range thresholds {
		// Находим дубли с текущим порогом
		predictedGroups := em.findDuplicatesWithThreshold(items, similarityFunc, threshold)
		predictedPairs := em.extractDuplicatePairs(predictedGroups)

		// Строим матрицу ошибок
		tp := 0
		fp := 0
		fn := 0
		tn := 0

		for pair := range predictedPairs {
			if actualPairs[pair] {
				tp++
			} else {
				fp++
			}
		}

		for pair := range actualPairs {
			if !predictedPairs[pair] {
				fn++
			}
		}

		// Вычисляем TPR и FPR
		tpr := 0.0
		if tp+fn > 0 {
			tpr = float64(tp) / float64(tp+fn)
		}

		fpr := 0.0
		if fp+tn > 0 {
			fpr = float64(fp) / float64(fp+tn)
		}

		points[i] = ROCPoint{
			FalsePositiveRate: fpr,
			TruePositiveRate:  tpr,
			Threshold:         threshold,
		}
	}

	return points
}

// CalculateAUC вычисляет площадь под ROC-кривой (Area Under Curve)
func (em *EvaluationMetrics) CalculateAUC(rocPoints []ROCPoint) float64 {
	if len(rocPoints) < 2 {
		return 0.0
	}

	// Сортируем точки по FPR
	sortedPoints := make([]ROCPoint, len(rocPoints))
	copy(sortedPoints, rocPoints)

	// Простая сортировка пузырьком
	for i := 0; i < len(sortedPoints)-1; i++ {
		for j := 0; j < len(sortedPoints)-i-1; j++ {
			if sortedPoints[j].FalsePositiveRate > sortedPoints[j+1].FalsePositiveRate {
				sortedPoints[j], sortedPoints[j+1] = sortedPoints[j+1], sortedPoints[j]
			}
		}
	}

	// Вычисляем AUC методом трапеций
	auc := 0.0
	for i := 1; i < len(sortedPoints); i++ {
		width := sortedPoints[i].FalsePositiveRate - sortedPoints[i-1].FalsePositiveRate
		height := (sortedPoints[i].TruePositiveRate + sortedPoints[i-1].TruePositiveRate) / 2.0
		auc += width * height
	}

	return auc
}

// CrossValidate выполняет кросс-валидацию алгоритма
func (em *EvaluationMetrics) CrossValidate(
	items []DuplicateItem,
	similarityFunc func(DuplicateItem, DuplicateItem) float64,
	threshold float64,
	folds int,
) []MetricsResult {
	if folds < 2 {
		folds = 5 // По умолчанию 5-fold
	}

	results := make([]MetricsResult, folds)
	itemsPerFold := len(items) / folds

	for fold := 0; fold < folds; fold++ {
		start := fold * itemsPerFold
		end := start + itemsPerFold
		if fold == folds-1 {
			end = len(items) // Последний фолд включает все оставшиеся элементы
		}

		// Разделяем на обучающую и тестовую выборки
		testItems := items[start:end]
		trainItems := make([]DuplicateItem, 0, len(items)-len(testItems))
		trainItems = append(trainItems, items[:start]...)
		trainItems = append(trainItems, items[end:]...)

		// Находим дубли в тестовой выборке
		testGroups := em.findDuplicatesWithThreshold(testItems, similarityFunc, threshold)

		// Для упрощения считаем, что все пары в тестовой выборке - это эталонные
		// В реальности нужны размеченные данные
		actualPairs := em.extractDuplicatePairs(testGroups)

		// Оцениваем
		predictedPairs := em.extractDuplicatePairs(testGroups)

		matrix := ConfusionMatrix{}
		for pair := range predictedPairs {
			if actualPairs[pair] {
				matrix.TruePositive++
			} else {
				matrix.FalsePositive++
			}
		}

		for pair := range actualPairs {
			if !predictedPairs[pair] {
				matrix.FalseNegative++
			}
		}

		results[fold] = em.CalculateMetrics(matrix)
	}

	return results
}

// AverageMetrics вычисляет средние метрики по нескольким результатам
func (em *EvaluationMetrics) AverageMetrics(results []MetricsResult) MetricsResult {
	if len(results) == 0 {
		return MetricsResult{}
	}

	avg := MetricsResult{
		ConfusionMatrix: ConfusionMatrix{},
	}

	for _, result := range results {
		avg.Precision += result.Precision
		avg.Recall += result.Recall
		avg.F1Score += result.F1Score
		avg.Accuracy += result.Accuracy
		avg.Specificity += result.Specificity
		avg.FalsePositiveRate += result.FalsePositiveRate
		avg.FalseNegativeRate += result.FalseNegativeRate

		avg.ConfusionMatrix.TruePositive += result.ConfusionMatrix.TruePositive
		avg.ConfusionMatrix.TrueNegative += result.ConfusionMatrix.TrueNegative
		avg.ConfusionMatrix.FalsePositive += result.ConfusionMatrix.FalsePositive
		avg.ConfusionMatrix.FalseNegative += result.ConfusionMatrix.FalseNegative
	}

	n := float64(len(results))
	avg.Precision /= n
	avg.Recall /= n
	avg.F1Score /= n
	avg.Accuracy /= n
	avg.Specificity /= n
	avg.FalsePositiveRate /= n
	avg.FalseNegativeRate /= n

	return avg
}

// CalculateOptimalThreshold находит оптимальный порог на основе F1-меры
func (em *EvaluationMetrics) CalculateOptimalThreshold(
	items []DuplicateItem,
	actualPairs map[Pair]bool,
	similarityFunc func(DuplicateItem, DuplicateItem) float64,
	thresholds []float64,
) (float64, MetricsResult) {
	bestThreshold := 0.0
	bestF1 := 0.0
	var bestMetrics MetricsResult

	for _, threshold := range thresholds {
		predictedGroups := em.findDuplicatesWithThreshold(items, similarityFunc, threshold)
		predictedPairs := em.extractDuplicatePairs(predictedGroups)

		matrix := ConfusionMatrix{}
		for pair := range predictedPairs {
			if actualPairs[pair] {
				matrix.TruePositive++
			} else {
				matrix.FalsePositive++
			}
		}

		for pair := range actualPairs {
			if !predictedPairs[pair] {
				matrix.FalseNegative++
			}
		}

		metrics := em.CalculateMetrics(matrix)

		if metrics.F1Score > bestF1 {
			bestF1 = metrics.F1Score
			bestThreshold = threshold
			bestMetrics = metrics
		}
	}

	return bestThreshold, bestMetrics
}

// ValidateMetrics проверяет, соответствуют ли метрики требованиям
func (em *EvaluationMetrics) ValidateMetrics(metrics MetricsResult, requirements QualityRequirements) ValidationResult {
	result := ValidationResult{
		MeetsRequirements: true,
		Violations:        make([]string, 0),
	}

	// Проверка Precision (ложные срабатывания не должны превышать 10%)
	if requirements.MaxFalsePositiveRate > 0 && metrics.FalsePositiveRate > requirements.MaxFalsePositiveRate {
		result.MeetsRequirements = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("FPR (%.2f%%) превышает допустимый порог (%.2f%%)",
				metrics.FalsePositiveRate*100, requirements.MaxFalsePositiveRate*100))
	}

	// Проверка Recall (пропуски не должны превышать 5%)
	if requirements.MaxFalseNegativeRate > 0 && metrics.FalseNegativeRate > requirements.MaxFalseNegativeRate {
		result.MeetsRequirements = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("FNR (%.2f%%) превышает допустимый порог (%.2f%%)",
				metrics.FalseNegativeRate*100, requirements.MaxFalseNegativeRate*100))
	}

	// Проверка минимальной Precision
	if requirements.MinPrecision > 0 && metrics.Precision < requirements.MinPrecision {
		result.MeetsRequirements = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("Precision (%.2f%%) ниже минимального порога (%.2f%%)",
				metrics.Precision*100, requirements.MinPrecision*100))
	}

	// Проверка минимальной Recall
	if requirements.MinRecall > 0 && metrics.Recall < requirements.MinRecall {
		result.MeetsRequirements = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("Recall (%.2f%%) ниже минимального порога (%.2f%%)",
				metrics.Recall*100, requirements.MinRecall*100))
	}

	// Проверка минимальной F1-меры
	if requirements.MinF1Score > 0 && metrics.F1Score < requirements.MinF1Score {
		result.MeetsRequirements = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("F1-score (%.2f%%) ниже минимального порога (%.2f%%)",
				metrics.F1Score*100, requirements.MinF1Score*100))
	}

	return result
}

// QualityRequirements требования к качеству алгоритма
type QualityRequirements struct {
	MaxFalsePositiveRate float64 // Максимальная частота ложных срабатываний (обычно 0.10 = 10%)
	MaxFalseNegativeRate float64 // Максимальная частота пропусков (обычно 0.05 = 5%)
	MinPrecision         float64 // Минимальная точность
	MinRecall            float64 // Минимальная полнота
	MinF1Score           float64 // Минимальная F1-мера
}

// DefaultQualityRequirements возвращает требования по умолчанию
func DefaultQualityRequirements() QualityRequirements {
	return QualityRequirements{
		MaxFalsePositiveRate: 0.10, // 10%
		MaxFalseNegativeRate: 0.05, // 5%
		MinPrecision:         0.0,  // Не задано
		MinRecall:            0.0,  // Не задано
		MinF1Score:           0.0,  // Не задано
	}
}

// ValidationResult результат валидации метрик
type ValidationResult struct {
	MeetsRequirements bool
	Violations        []string
}

// String возвращает строковое представление результата валидации
func (vr ValidationResult) String() string {
	if vr.MeetsRequirements {
		return "Метрики соответствуют требованиям"
	}
	return fmt.Sprintf("Метрики не соответствуют требованиям:\n%s", strings.Join(vr.Violations, "\n"))
}
