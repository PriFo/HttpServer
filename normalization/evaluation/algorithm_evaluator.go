package evaluation

import (
	"fmt"
	"time"
)

// LabeledPair размеченная пара записей для оценки
type LabeledPair struct {
	Item1      string
	Item2      string
	IsDuplicate bool // true если это дубликат
}

// SimilarityFunction функция вычисления сходства
type SimilarityFunction func(s1, s2 string) float64

// AlgorithmEvaluator оценивает алгоритмы на размеченных данных
type AlgorithmEvaluator struct {
	labeledPairs []LabeledPair
	threshold    float64
}

// NewAlgorithmEvaluator создает новый оценщик алгоритмов
func NewAlgorithmEvaluator(labeledPairs []LabeledPair, threshold float64) *AlgorithmEvaluator {
	return &AlgorithmEvaluator{
		labeledPairs: labeledPairs,
		threshold:    threshold,
	}
}

// Evaluate оценивает алгоритм на размеченных данных
func (ae *AlgorithmEvaluator) Evaluate(algorithmName string, similarityFunc SimilarityFunction) EvaluationResult {
	startTime := time.Now()
	
	cm := ConfusionMatrix{}
	
	for _, pair := range ae.labeledPairs {
		similarity := similarityFunc(pair.Item1, pair.Item2)
		predicted := similarity >= ae.threshold
		
		if pair.IsDuplicate && predicted {
			cm.TruePositives++
		} else if !pair.IsDuplicate && !predicted {
			cm.TrueNegatives++
		} else if !pair.IsDuplicate && predicted {
			cm.FalsePositives++
		} else if pair.IsDuplicate && !predicted {
			cm.FalseNegatives++
		}
	}
	
	elapsed := time.Since(startTime)
	totalTime := elapsed.Seconds()
	itemsPerSecond := float64(len(ae.labeledPairs)) / totalTime
	
	metrics := CalculateMetrics(cm)
	
	return EvaluationResult{
		AlgorithmName:  algorithmName,
		Metrics:        metrics,
		TotalTime:      totalTime,
		ItemsPerSecond: itemsPerSecond,
	}
}

// EvaluateWithAdaptiveThreshold оценивает алгоритм с адаптивным порогом
func (ae *AlgorithmEvaluator) EvaluateWithAdaptiveThreshold(
	algorithmName string,
	similarityFunc SimilarityFunction,
	thresholdFunc func(s1, s2 string) float64,
) EvaluationResult {
	startTime := time.Now()
	
	cm := ConfusionMatrix{}
	
	for _, pair := range ae.labeledPairs {
		similarity := similarityFunc(pair.Item1, pair.Item2)
		threshold := thresholdFunc(pair.Item1, pair.Item2)
		predicted := similarity >= threshold
		
		if pair.IsDuplicate && predicted {
			cm.TruePositives++
		} else if !pair.IsDuplicate && !predicted {
			cm.TrueNegatives++
		} else if !pair.IsDuplicate && predicted {
			cm.FalsePositives++
		} else if pair.IsDuplicate && !predicted {
			cm.FalseNegatives++
		}
	}
	
	elapsed := time.Since(startTime)
	totalTime := elapsed.Seconds()
	itemsPerSecond := float64(len(ae.labeledPairs)) / totalTime
	
	metrics := CalculateMetrics(cm)
	
	return EvaluationResult{
		AlgorithmName:  algorithmName,
		Metrics:        metrics,
		TotalTime:      totalTime,
		ItemsPerSecond: itemsPerSecond,
	}
}

// FindOptimalThreshold находит оптимальный порог для алгоритма
func (ae *AlgorithmEvaluator) FindOptimalThreshold(
	algorithmName string,
	similarityFunc SimilarityFunction,
) (float64, EvaluationResult) {
	bestThreshold := 0.0
	bestResult := EvaluationResult{}
	bestF1 := 0.0
	
	// Перебираем пороги от 0.5 до 1.0 с шагом 0.01
	for threshold := 0.5; threshold <= 1.0; threshold += 0.01 {
		evaluator := NewAlgorithmEvaluator(ae.labeledPairs, threshold)
		result := evaluator.Evaluate(algorithmName, similarityFunc)
		
		if result.Metrics.F1Score > bestF1 {
			bestF1 = result.Metrics.F1Score
			bestThreshold = threshold
			bestResult = result
		}
	}
	
	return bestThreshold, bestResult
}

// GenerateReport генерирует текстовый отчет об оценке
func GenerateReport(results []EvaluationResult) string {
	report := "=== ОТЧЕТ ОБ ОЦЕНКЕ АЛГОРИТМОВ ===\n\n"
	
	for i, result := range results {
		report += fmt.Sprintf("Алгоритм %d: %s\n", i+1, result.AlgorithmName)
		report += fmt.Sprintf("  Precision:     %.4f\n", result.Metrics.Precision)
		report += fmt.Sprintf("  Recall:        %.4f\n", result.Metrics.Recall)
		report += fmt.Sprintf("  F1-Score:      %.4f\n", result.Metrics.F1Score)
		report += fmt.Sprintf("  F2-Score:      %.4f\n", result.Metrics.F2Score)
		report += fmt.Sprintf("  Accuracy:      %.4f\n", result.Metrics.Accuracy)
		report += fmt.Sprintf("  False Positive Rate: %.4f\n", result.Metrics.FalsePositiveRate)
		report += fmt.Sprintf("  False Negative Rate: %.4f\n", result.Metrics.FalseNegativeRate)
		report += fmt.Sprintf("  Время выполнения: %.4f сек\n", result.TotalTime)
		report += fmt.Sprintf("  Производительность: %.2f пар/сек\n", result.ItemsPerSecond)
		report += fmt.Sprintf("  TP: %d, TN: %d, FP: %d, FN: %d\n\n",
			result.Metrics.ConfusionMatrix.TruePositives,
			result.Metrics.ConfusionMatrix.TrueNegatives,
			result.Metrics.ConfusionMatrix.FalsePositives,
			result.Metrics.ConfusionMatrix.FalseNegatives)
	}
	
	// Находим лучший алгоритм
	best := BestAlgorithm(results)
	if best != nil {
		report += fmt.Sprintf("Лучший алгоритм: %s (F1=%.4f)\n", best.AlgorithmName, best.Metrics.F1Score)
	}
	
	return report
}

// GenerateHTMLReport генерирует HTML отчет
func GenerateHTMLReport(results []EvaluationResult) string {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Отчет об оценке алгоритмов</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        tr:nth-child(even) { background-color: #f2f2f2; }
        .best { background-color: #90EE90 !important; font-weight: bold; }
    </style>
</head>
<body>
    <h1>Отчет об оценке алгоритмов нормализации НСИ</h1>
    <table>
        <tr>
            <th>Алгоритм</th>
            <th>Precision</th>
            <th>Recall</th>
            <th>F1-Score</th>
            <th>F2-Score</th>
            <th>Accuracy</th>
            <th>FPR</th>
            <th>FNR</th>
            <th>Время (сек)</th>
            <th>Производительность</th>
        </tr>`

	best := BestAlgorithm(results)
	
	for _, result := range results {
		isBest := best != nil && result.AlgorithmName == best.AlgorithmName
		rowClass := ""
		if isBest {
			rowClass = " class='best'"
		}
		
		html += fmt.Sprintf(`
        <tr%s>
            <td>%s</td>
            <td>%.4f</td>
            <td>%.4f</td>
            <td>%.4f</td>
            <td>%.4f</td>
            <td>%.4f</td>
            <td>%.4f</td>
            <td>%.4f</td>
            <td>%.4f</td>
            <td>%.2f</td>
        </tr>`,
			rowClass,
			result.AlgorithmName,
			result.Metrics.Precision,
			result.Metrics.Recall,
			result.Metrics.F1Score,
			result.Metrics.F2Score,
			result.Metrics.Accuracy,
			result.Metrics.FalsePositiveRate,
			result.Metrics.FalseNegativeRate,
			result.TotalTime,
			result.ItemsPerSecond)
	}
	
	html += `
    </table>
</body>
</html>`
	
	return html
}

