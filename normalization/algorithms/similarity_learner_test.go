package algorithms

import (
	"testing"
)

func TestSimilarityLearner(t *testing.T) {
	learner := NewSimilarityLearner()

	// Добавляем обучающие пары
	trainingPairs := []SimilarityTestPair{
		{"ООО Рога и Копыта", "ООО Рога и Копыта", true},
		{"ООО Рога и Копыта", "Рога и Копыта ООО", true},
		{"ООО Рога и Копыта", "ООО Другая Компания", false},
		{"Кабель ВВГнг 3x2.5", "Кабель ВВГ 3x2.5", true},
		{"Кабель ВВГнг 3x2.5", "Провод ПВС 3x2.5", false},
	}

	learner.AddTrainingPairs(trainingPairs)

	if learner.GetTrainingPairsCount() != len(trainingPairs) {
		t.Errorf("Expected %d training pairs, got %d", len(trainingPairs), learner.GetTrainingPairsCount())
	}

	// Тестируем оптимизацию весов
	weights, err := learner.OptimizeWeights(10, 0.01)
	if err != nil {
		t.Fatalf("Failed to optimize weights: %v", err)
	}

	if weights == nil {
		t.Fatal("Optimized weights should not be nil")
	}

	// Проверяем, что веса нормализованы
	total := weights.JaroWinkler + weights.LCS + weights.Phonetic + weights.Ngram + weights.Jaccard
	if total < 0.99 || total > 1.01 {
		t.Errorf("Weights should sum to ~1.0, got %f", total)
	}
}

func TestSimilarityLearnerOptimalThreshold(t *testing.T) {
	learner := NewSimilarityLearner()

	testPairs := []SimilarityTestPair{
		{"ООО Рога и Копыта", "ООО Рога и Копыта", true},
		{"ООО Рога и Копыта", "Рога и Копыта ООО", true},
		{"ООО Рога и Копыта", "ООО Другая Компания", false},
		{"Кабель ВВГнг", "Кабель ВВГ", true},
		{"Кабель ВВГнг", "Провод ПВС", false},
	}

	weights := DefaultSimilarityWeights()
	threshold, metrics := learner.GetOptimalThreshold(testPairs, weights)

	if threshold < 0.5 || threshold > 0.95 {
		t.Errorf("Optimal threshold should be between 0.5 and 0.95, got %f", threshold)
	}

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Проверяем, что метрики валидны
	if metrics.Precision() < 0 || metrics.Precision() > 1 {
		t.Errorf("Precision should be between 0 and 1, got %f", metrics.Precision())
	}
}

func TestSimilarityLearnerCrossValidate(t *testing.T) {
	learner := NewSimilarityLearner()

	// Создаем достаточно пар для кросс-валидации
	trainingPairs := make([]SimilarityTestPair, 0)
	for i := 0; i < 20; i++ {
		if i%2 == 0 {
			trainingPairs = append(trainingPairs, SimilarityTestPair{
				S1:          "Тест " + string(rune('A'+i)),
				S2:          "Тест " + string(rune('A'+i)),
				IsDuplicate: true,
			})
		} else {
			trainingPairs = append(trainingPairs, SimilarityTestPair{
				S1:          "Тест " + string(rune('A'+i)),
				S2:          "Другой " + string(rune('A'+i)),
				IsDuplicate: false,
			})
		}
	}

	learner.AddTrainingPairs(trainingPairs)

	results, err := learner.CrossValidate(5)
	if err != nil {
		t.Fatalf("Cross-validation failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 fold results, got %d", len(results))
	}

	// Проверяем средние метрики
	avgMetrics := GetAverageMetrics(results)
	if avgMetrics == nil {
		t.Fatal("Average metrics should not be nil")
	}
}

func TestGetAverageMetrics(t *testing.T) {
	metrics1 := &EvaluationMetrics{
		TruePositives:  10,
		FalsePositives: 2,
		FalseNegatives: 3,
		TrueNegatives:  85,
	}

	metrics2 := &EvaluationMetrics{
		TruePositives:  12,
		FalsePositives: 1,
		FalseNegatives: 2,
		TrueNegatives:  85,
	}

	avg := GetAverageMetrics([]*EvaluationMetrics{metrics1, metrics2})

	if avg.TruePositives != 11 {
		t.Errorf("Expected average TP = 11, got %d", avg.TruePositives)
	}

	if avg.FalsePositives != 1 {
		t.Errorf("Expected average FP = 1, got %d", avg.FalsePositives)
	}
}

func TestSimilarityLearnerReset(t *testing.T) {
	learner := NewSimilarityLearner()

	learner.AddTrainingPair(SimilarityTestPair{"test1", "test1", true})
	if learner.GetTrainingPairsCount() != 1 {
		t.Error("Should have 1 training pair")
	}

	learner.Reset()
	if learner.GetTrainingPairsCount() != 0 {
		t.Error("Should have 0 training pairs after reset")
	}
}

