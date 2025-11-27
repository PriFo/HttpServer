package algorithms

import (
	"fmt"
)

// EvaluationMetrics метрики оценки эффективности алгоритмов поиска дублей
type EvaluationMetrics struct {
	TruePositives  int     // Правильно найденные дубли (TP)
	FalsePositives int     // Ложные срабатывания (FP)
	FalseNegatives int     // Пропущенные дубли (FN)
	TrueNegatives  int     // Правильно определенные уникальные записи (TN)
}

// NewEvaluationMetrics создает новые метрики оценки
func NewEvaluationMetrics() *EvaluationMetrics {
	return &EvaluationMetrics{}
}

// AddResult добавляет результат классификации
func (em *EvaluationMetrics) AddResult(predictedDuplicate, actualDuplicate bool) {
	if predictedDuplicate && actualDuplicate {
		em.TruePositives++
	} else if predictedDuplicate && !actualDuplicate {
		em.FalsePositives++
	} else if !predictedDuplicate && actualDuplicate {
		em.FalseNegatives++
	} else {
		em.TrueNegatives++
	}
}

// Precision вычисляет точность (Precision)
// Precision = TP / (TP + FP)
// Показывает долю корректно найденных дублей среди всех найденных
func (em *EvaluationMetrics) Precision() float64 {
	total := em.TruePositives + em.FalsePositives
	if total == 0 {
		return 0.0
	}
	return float64(em.TruePositives) / float64(total)
}

// Recall вычисляет полноту (Recall)
// Recall = TP / (TP + FN)
// Показывает долю найденных дублей среди всех существующих
func (em *EvaluationMetrics) Recall() float64 {
	total := em.TruePositives + em.FalseNegatives
	if total == 0 {
		return 0.0
	}
	return float64(em.TruePositives) / float64(total)
}

// F1Score вычисляет F-меру (F1-score)
// F1 = 2 * (Precision * Recall) / (Precision + Recall)
// Гармоническое среднее точности и полноты
func (em *EvaluationMetrics) F1Score() float64 {
	precision := em.Precision()
	recall := em.Recall()

	if precision+recall == 0 {
		return 0.0
	}

	return 2 * (precision * recall) / (precision + recall)
}

// Accuracy вычисляет точность классификации
// Accuracy = (TP + TN) / (TP + TN + FP + FN)
func (em *EvaluationMetrics) Accuracy() float64 {
	total := em.Total()
	if total == 0 {
		return 0.0
	}
	return float64(em.TruePositives+em.TrueNegatives) / float64(total)
}

// FalsePositiveRate вычисляет частоту ложных срабатываний (FPR)
// FPR = FP / (FP + TN)
// Ошибки первого рода - не должна превышать 10%
func (em *EvaluationMetrics) FalsePositiveRate() float64 {
	total := em.FalsePositives + em.TrueNegatives
	if total == 0 {
		return 0.0
	}
	return float64(em.FalsePositives) / float64(total)
}

// FalseNegativeRate вычисляет частоту пропусков (FNR)
// FNR = FN / (FN + TP)
// Ошибки второго рода - не должна превышать 5%
func (em *EvaluationMetrics) FalseNegativeRate() float64 {
	total := em.FalseNegatives + em.TruePositives
	if total == 0 {
		return 0.0
	}
	return float64(em.FalseNegatives) / float64(total)
}

// Total возвращает общее количество проверенных записей
func (em *EvaluationMetrics) Total() int {
	return em.TruePositives + em.FalsePositives + em.FalseNegatives + em.TrueNegatives
}

// Reset сбрасывает все метрики
func (em *EvaluationMetrics) Reset() {
	em.TruePositives = 0
	em.FalsePositives = 0
	em.FalseNegatives = 0
	em.TrueNegatives = 0
}

// String возвращает строковое представление метрик
func (em *EvaluationMetrics) String() string {
	return fmt.Sprintf(
		"Precision: %.4f, Recall: %.4f, F1: %.4f, Accuracy: %.4f, FPR: %.4f, FNR: %.4f",
		em.Precision(),
		em.Recall(),
		em.F1Score(),
		em.Accuracy(),
		em.FalsePositiveRate(),
		em.FalseNegativeRate(),
	)
}

// DetailedReport возвращает детальный отчет о метриках
func (em *EvaluationMetrics) DetailedReport() string {
	total := em.Total()
	if total == 0 {
		return "No data collected"
	}

	return fmt.Sprintf(`Evaluation Metrics Report:
Total samples: %d
True Positives (TP): %d (%.2f%%)
False Positives (FP): %d (%.2f%%)
False Negatives (FN): %d (%.2f%%)
True Negatives (TN): %d (%.2f%%)

Precision: %.4f (доля корректно найденных дублей среди всех найденных)
Recall: %.4f (доля найденных дублей среди всех существующих)
F1-Score: %.4f (гармоническое среднее точности и полноты)
Accuracy: %.4f (общая точность классификации)

False Positive Rate: %.4f (ошибки первого рода, должно быть < 0.10)
False Negative Rate: %.4f (ошибки второго рода, должно быть < 0.05)`,
		total,
		em.TruePositives, float64(em.TruePositives)/float64(total)*100,
		em.FalsePositives, float64(em.FalsePositives)/float64(total)*100,
		em.FalseNegatives, float64(em.FalseNegatives)/float64(total)*100,
		em.TrueNegatives, float64(em.TrueNegatives)/float64(total)*100,
		em.Precision(),
		em.Recall(),
		em.F1Score(),
		em.Accuracy(),
		em.FalsePositiveRate(),
		em.FalseNegativeRate(),
	)
}

// IsAcceptable проверяет, соответствуют ли метрики требованиям
// FPR < 10% и FNR < 5%
func (em *EvaluationMetrics) IsAcceptable() bool {
	return em.FalsePositiveRate() < 0.10 && em.FalseNegativeRate() < 0.05
}

// GetRecommendations возвращает рекомендации по улучшению на основе метрик
func (em *EvaluationMetrics) GetRecommendations() []string {
	var recommendations []string

	fpr := em.FalsePositiveRate()
	fnr := em.FalseNegativeRate()
	precision := em.Precision()
	recall := em.Recall()

	if fpr >= 0.10 {
		recommendations = append(recommendations,
			fmt.Sprintf("Высокий уровень ложных срабатываний (%.2f%%). Рекомендуется повысить порог схожести.", fpr*100))
	}

	if fnr >= 0.05 {
		recommendations = append(recommendations,
			fmt.Sprintf("Высокий уровень пропусков (%.2f%%). Рекомендуется снизить порог схожести или использовать дополнительные алгоритмы.", fnr*100))
	}

	if precision < 0.85 {
		recommendations = append(recommendations,
			fmt.Sprintf("Низкая точность (%.2f%%). Много ложных срабатываний. Рекомендуется улучшить алгоритмы фильтрации.", precision*100))
	}

	if recall < 0.90 {
		recommendations = append(recommendations,
			fmt.Sprintf("Низкая полнота (%.2f%%). Много пропущенных дублей. Рекомендуется использовать более чувствительные алгоритмы.", recall*100))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Метрики соответствуют требованиям. Система работает эффективно.")
	}

	return recommendations
}

// CompareMetrics сравнивает две метрики и возвращает разницу
func CompareMetrics(metrics1, metrics2 *EvaluationMetrics) string {
	_ = &EvaluationMetrics{
		TruePositives:  metrics1.TruePositives - metrics2.TruePositives,
		FalsePositives: metrics1.FalsePositives - metrics2.FalsePositives,
		FalseNegatives: metrics1.FalseNegatives - metrics2.FalseNegatives,
		TrueNegatives:  metrics1.TrueNegatives - metrics2.TrueNegatives,
	}

	return fmt.Sprintf(`Comparison:
Precision: %.4f vs %.4f (diff: %.4f)
Recall: %.4f vs %.4f (diff: %.4f)
F1-Score: %.4f vs %.4f (diff: %.4f)
FPR: %.4f vs %.4f (diff: %.4f)
FNR: %.4f vs %.4f (diff: %.4f)`,
		metrics1.Precision(), metrics2.Precision(), metrics1.Precision()-metrics2.Precision(),
		metrics1.Recall(), metrics2.Recall(), metrics1.Recall()-metrics2.Recall(),
		metrics1.F1Score(), metrics2.F1Score(), metrics1.F1Score()-metrics2.F1Score(),
		metrics1.FalsePositiveRate(), metrics2.FalsePositiveRate(), metrics1.FalsePositiveRate()-metrics2.FalsePositiveRate(),
		metrics1.FalseNegativeRate(), metrics2.FalseNegativeRate(), metrics1.FalseNegativeRate()-metrics2.FalseNegativeRate(),
	)
}

