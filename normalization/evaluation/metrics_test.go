package evaluation

import (
	"math"
	"testing"
)

// TestConfusionMatrix проверяет структуру матрицы ошибок
func TestConfusionMatrix(t *testing.T) {
	cm := ConfusionMatrix{
		TruePositives:  10,
		TrueNegatives:  20,
		FalsePositives: 5,
		FalseNegatives: 3,
	}
	
	if cm.TruePositives < 0 {
		t.Error("TruePositives should be non-negative")
	}
	
	if cm.TrueNegatives < 0 {
		t.Error("TrueNegatives should be non-negative")
	}
	
	if cm.FalsePositives < 0 {
		t.Error("FalsePositives should be non-negative")
	}
	
	if cm.FalseNegatives < 0 {
		t.Error("FalseNegatives should be non-negative")
	}
	
	total := cm.TruePositives + cm.TrueNegatives + cm.FalsePositives + cm.FalseNegatives
	if total != 38 {
		t.Errorf("Total = %d, want 38", total)
	}
}

// TestCalculateMetrics проверяет расчет метрик
func TestCalculateMetrics(t *testing.T) {
	tests := []struct {
		name           string
		cm             ConfusionMatrix
		wantPrecision  float64
		wantRecall     float64
		wantF1Score    float64
		wantAccuracy   float64
	}{
		{
			name: "perfect classification",
			cm: ConfusionMatrix{
				TruePositives:  10,
				TrueNegatives:  10,
				FalsePositives: 0,
				FalseNegatives: 0,
			},
			wantPrecision: 1.0,
			wantRecall:    1.0,
			wantF1Score:   1.0,
			wantAccuracy:  1.0,
		},
		{
			name: "all false positives",
			cm: ConfusionMatrix{
				TruePositives:  0,
				TrueNegatives:  0,
				FalsePositives: 10,
				FalseNegatives: 0,
			},
			wantPrecision: 0.0,
			wantRecall:    0.0,
			wantF1Score:   0.0,
			wantAccuracy:  0.0,
		},
		{
			name: "mixed results",
			cm: ConfusionMatrix{
				TruePositives:  8,
				TrueNegatives:  12,
				FalsePositives: 2,
				FalseNegatives: 3,
			},
			wantPrecision: 0.8,  // 8 / (8 + 2)
			wantRecall:    0.727, // 8 / (8 + 3) ≈ 0.727
			wantAccuracy:  0.8,  // (8 + 12) / 25
		},
		{
			name: "no positives",
			cm: ConfusionMatrix{
				TruePositives:  0,
				TrueNegatives:  20,
				FalsePositives: 0,
				FalseNegatives: 5,
			},
			wantPrecision: 0.0, // Division by zero handled
			wantRecall:    0.0,
			wantF1Score:   0.0,
			wantAccuracy:  0.8, // 20 / 25
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := CalculateMetrics(tt.cm)
			
			// Metrics - это структура, не указатель, проверяем валидность
			if metrics.Precision < 0 {
				t.Error("CalculateMetrics() should return valid metrics")
			}
			
			// Проверяем точность с небольшой погрешностью
			if !floatEquals(metrics.Precision, tt.wantPrecision) {
				t.Errorf("Precision = %f, want %f", metrics.Precision, tt.wantPrecision)
			}
			
			if tt.wantRecall > 0 && !floatEquals(metrics.Recall, tt.wantRecall) {
				// Для recall допускаем погрешность
				if math.Abs(metrics.Recall-tt.wantRecall) > 0.01 {
					t.Errorf("Recall = %f, want %f", metrics.Recall, tt.wantRecall)
				}
			}
			
			if tt.wantF1Score > 0 && !floatEquals(metrics.F1Score, tt.wantF1Score) {
				if math.Abs(metrics.F1Score-tt.wantF1Score) > 0.01 {
					t.Errorf("F1Score = %f, want %f", metrics.F1Score, tt.wantF1Score)
				}
			}
			
			if !floatEquals(metrics.Accuracy, tt.wantAccuracy) {
				t.Errorf("Accuracy = %f, want %f", metrics.Accuracy, tt.wantAccuracy)
			}
			
			// Проверяем, что все метрики в допустимом диапазоне
			if metrics.Precision < 0 || metrics.Precision > 1 {
				t.Errorf("Precision = %f, should be between 0 and 1", metrics.Precision)
			}
			
			if metrics.Recall < 0 || metrics.Recall > 1 {
				t.Errorf("Recall = %f, should be between 0 and 1", metrics.Recall)
			}
			
			if metrics.F1Score < 0 || metrics.F1Score > 1 {
				t.Errorf("F1Score = %f, should be between 0 and 1", metrics.F1Score)
			}
			
			if metrics.Accuracy < 0 || metrics.Accuracy > 1 {
				t.Errorf("Accuracy = %f, should be between 0 and 1", metrics.Accuracy)
			}
		})
	}
}

// floatEquals проверяет равенство float64 с погрешностью
func floatEquals(a, b float64) bool {
	const epsilon = 0.0001
	return math.Abs(a-b) < epsilon
}

// TestEvaluationResult проверяет структуру результата оценки
func TestEvaluationResult(t *testing.T) {
	result := EvaluationResult{
		AlgorithmName:  "test_algorithm",
		Metrics:        Metrics{Precision: 0.9, Recall: 0.8, F1Score: 0.85, Accuracy: 0.88},
		TotalTime:      1.5,
		ItemsPerSecond: 100.0,
	}
	
	if result.AlgorithmName == "" {
		t.Error("EvaluationResult.AlgorithmName should not be empty")
	}
	
	// Metrics - это структура, проверяем валидность значений
	if result.Metrics.Precision < 0 {
		t.Error("EvaluationResult.Metrics should have valid values")
	}
	
	if result.TotalTime < 0 {
		t.Error("EvaluationResult.TotalTime should be non-negative")
	}
	
	if result.ItemsPerSecond < 0 {
		t.Error("EvaluationResult.ItemsPerSecond should be non-negative")
	}
}

// TestCalculateMetrics_EdgeCases проверяет граничные случаи
func TestCalculateMetrics_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		cm   ConfusionMatrix
	}{
		{
			name: "all zeros",
			cm: ConfusionMatrix{
				TruePositives:  0,
				TrueNegatives:  0,
				FalsePositives: 0,
				FalseNegatives: 0,
			},
		},
		{
			name: "only true positives",
			cm: ConfusionMatrix{
				TruePositives:  10,
				TrueNegatives:  0,
				FalsePositives: 0,
				FalseNegatives: 0,
			},
		},
		{
			name: "only true negatives",
			cm: ConfusionMatrix{
				TruePositives:  0,
				TrueNegatives:  10,
				FalsePositives: 0,
				FalseNegatives: 0,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := CalculateMetrics(tt.cm)
			
			// Metrics - это структура, не указатель, проверяем валидность
			if metrics.Precision < 0 {
				t.Error("CalculateMetrics() should return valid metrics")
			}
			
			// Проверяем, что метрики валидны (не NaN, не Inf)
			if math.IsNaN(metrics.Precision) || math.IsInf(metrics.Precision, 0) {
				t.Error("Precision should not be NaN or Inf")
			}
			
			if math.IsNaN(metrics.Recall) || math.IsInf(metrics.Recall, 0) {
				t.Error("Recall should not be NaN or Inf")
			}
			
			if math.IsNaN(metrics.F1Score) || math.IsInf(metrics.F1Score, 0) {
				t.Error("F1Score should not be NaN or Inf")
			}
			
			if math.IsNaN(metrics.Accuracy) || math.IsInf(metrics.Accuracy, 0) {
				t.Error("Accuracy should not be NaN or Inf")
			}
		})
	}
}

