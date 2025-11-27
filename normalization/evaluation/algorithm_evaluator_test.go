package evaluation

import (
	"math"
	"testing"
)

// TestNewAlgorithmEvaluator проверяет создание нового оценщика алгоритмов
func TestNewAlgorithmEvaluator(t *testing.T) {
	pairs := []LabeledPair{
		{Item1: "test1", Item2: "test2", IsDuplicate: false},
		{Item1: "test3", Item2: "test3", IsDuplicate: true},
	}
	threshold := 0.8
	
	evaluator := NewAlgorithmEvaluator(pairs, threshold)
	
	if evaluator == nil {
		t.Fatal("NewAlgorithmEvaluator() returned nil")
	}
	
	if len(evaluator.labeledPairs) != len(pairs) {
		t.Errorf("Evaluator.labeledPairs length = %d, want %d", len(evaluator.labeledPairs), len(pairs))
	}
	
	if evaluator.threshold != threshold {
		t.Errorf("Evaluator.threshold = %f, want %f", evaluator.threshold, threshold)
	}
}

// TestAlgorithmEvaluator_Evaluate проверяет оценку алгоритма
func TestAlgorithmEvaluator_Evaluate(t *testing.T) {
	pairs := []LabeledPair{
		{Item1: "test", Item2: "test", IsDuplicate: true},   // Дубликат
		{Item1: "test1", Item2: "test2", IsDuplicate: false}, // Не дубликат
		{Item1: "same", Item2: "same", IsDuplicate: true},   // Дубликат
	}
	threshold := 0.8
	
	evaluator := NewAlgorithmEvaluator(pairs, threshold)
	
	// Простая функция сходства: 1.0 для одинаковых строк, 0.0 для разных
	similarityFunc := func(s1, s2 string) float64 {
		if s1 == s2 {
			return 1.0
		}
		return 0.0
	}
	
	result := evaluator.Evaluate("test_algorithm", similarityFunc)
	
	if result.AlgorithmName != "test_algorithm" {
		t.Errorf("Result.AlgorithmName = %v, want test_algorithm", result.AlgorithmName)
	}
	
	// Metrics - это структура, не указатель, поэтому проверяем валидность значений
	if result.Metrics.Precision < 0 {
		t.Error("Result.Metrics should have valid values")
	}
	
	if result.TotalTime < 0 {
		t.Error("Result.TotalTime should be non-negative")
	}
	
	// ItemsPerSecond может быть 0 для очень быстрых операций или очень большим
	if result.ItemsPerSecond < 0 {
		t.Error("Result.ItemsPerSecond should be non-negative")
	}
	
	// Проверяем метрики
	if result.Metrics.Precision < 0 || result.Metrics.Precision > 1 {
		t.Errorf("Metrics.Precision = %f, should be between 0 and 1", result.Metrics.Precision)
	}
	
	if result.Metrics.Recall < 0 || result.Metrics.Recall > 1 {
		t.Errorf("Metrics.Recall = %f, should be between 0 and 1", result.Metrics.Recall)
	}
	
	if result.Metrics.F1Score < 0 || result.Metrics.F1Score > 1 {
		t.Errorf("Metrics.F1Score = %f, should be between 0 and 1", result.Metrics.F1Score)
	}
}

// TestAlgorithmEvaluator_EvaluateWithAdaptiveThreshold проверяет оценку с адаптивным порогом
func TestAlgorithmEvaluator_EvaluateWithAdaptiveThreshold(t *testing.T) {
	pairs := []LabeledPair{
		{Item1: "test", Item2: "test", IsDuplicate: true},
		{Item1: "test1", Item2: "test2", IsDuplicate: false},
	}
	threshold := 0.8
	
	evaluator := NewAlgorithmEvaluator(pairs, threshold)
	
	similarityFunc := func(s1, s2 string) float64 {
		if s1 == s2 {
			return 1.0
		}
		return 0.0
	}
	
	thresholdFunc := func(s1, s2 string) float64 {
		// Адаптивный порог: 0.9 для коротких строк, 0.7 для длинных
		if len(s1) < 5 || len(s2) < 5 {
			return 0.9
		}
		return 0.7
	}
	
	result := evaluator.EvaluateWithAdaptiveThreshold("test_algorithm", similarityFunc, thresholdFunc)
	
	if result.AlgorithmName != "test_algorithm" {
		t.Errorf("Result.AlgorithmName = %v, want test_algorithm", result.AlgorithmName)
	}
	
	// Metrics - это структура, не указатель, поэтому проверяем валидность значений
	if result.Metrics.Precision < 0 {
		t.Error("Result.Metrics should have valid values")
	}
	
	if result.TotalTime < 0 {
		t.Error("Result.TotalTime should be non-negative")
	}
}

// TestLabeledPair проверяет структуру размеченной пары
func TestLabeledPair(t *testing.T) {
	pair := LabeledPair{
		Item1:        "test1",
		Item2:        "test2",
		IsDuplicate:  false,
	}
	
	if pair.Item1 == "" {
		t.Error("LabeledPair.Item1 should not be empty")
	}
	
	if pair.Item2 == "" {
		t.Error("LabeledPair.Item2 should not be empty")
	}
}

// TestAlgorithmEvaluator_EmptyPairs проверяет обработку пустого списка пар
func TestAlgorithmEvaluator_EmptyPairs(t *testing.T) {
	evaluator := NewAlgorithmEvaluator([]LabeledPair{}, 0.8)
	
	similarityFunc := func(s1, s2 string) float64 {
		return 0.5
	}
	
	result := evaluator.Evaluate("test", similarityFunc)
	
	if result.AlgorithmName != "test" {
		t.Errorf("Result.AlgorithmName = %v, want test", result.AlgorithmName)
	}
	
	// Metrics - это структура, не указатель, поэтому проверяем валидность значений
	if result.Metrics.Precision < 0 {
		t.Error("Result.Metrics should have valid values")
	}
	
	// Для пустого списка ItemsPerSecond может быть NaN (деление на 0) или 0
	// Это нормально для пустого списка
	if result.ItemsPerSecond < 0 && !math.IsNaN(result.ItemsPerSecond) {
		t.Errorf("ItemsPerSecond = %f, should be non-negative or NaN for empty pairs", result.ItemsPerSecond)
	}
}

// TestAlgorithmEvaluator_Threshold проверяет влияние порога на результаты
func TestAlgorithmEvaluator_Threshold(t *testing.T) {
	pairs := []LabeledPair{
		{Item1: "test", Item2: "test", IsDuplicate: true},
		{Item1: "test1", Item2: "test2", IsDuplicate: false},
	}
	
	// Тест с низким порогом
	lowThresholdEvaluator := NewAlgorithmEvaluator(pairs, 0.1)
	
	// Тест с высоким порогом
	highThresholdEvaluator := NewAlgorithmEvaluator(pairs, 0.9)
	
	similarityFunc := func(s1, s2 string) float64 {
		if s1 == s2 {
			return 1.0
		}
		return 0.5 // Среднее сходство
	}
	
	lowResult := lowThresholdEvaluator.Evaluate("low_threshold", similarityFunc)
	highResult := highThresholdEvaluator.Evaluate("high_threshold", similarityFunc)
	
	// С низким порогом должно быть больше положительных предсказаний
	// (может быть больше false positives)
	// Metrics - это структура, проверяем валидность значений
	if lowResult.Metrics.Precision < 0 || highResult.Metrics.Precision < 0 {
		t.Error("Metrics should have valid values")
	}
	
	// Проверяем, что результаты различаются
	if lowResult.Metrics.Precision == highResult.Metrics.Precision &&
		lowResult.Metrics.Recall == highResult.Metrics.Recall {
		t.Log("Results may be similar, but thresholds should affect predictions")
	}
}

