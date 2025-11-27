package server

import (
	"testing"
	"time"
)

// TestSnapshotComparisonResponse проверяет структуру ответа сравнения снимков
func TestSnapshotComparisonResponse(t *testing.T) {
	dbID := 1
	response := &SnapshotComparisonResponse{
		SnapshotID: 1,
		Iterations: []IterationComparison{
			{
				UploadID:        1,
				IterationNumber: 1,
				IterationLabel:  "test-iteration",
				StartedAt:       time.Now(),
				TotalItems:      100,
				TotalCatalogs:   5,
				DatabaseID:      &dbID,
			},
		},
		TotalItems: map[int]int{1: 100},
	}
	
	if response.SnapshotID <= 0 {
		t.Error("SnapshotComparisonResponse.SnapshotID should be positive")
	}
	
	if len(response.Iterations) == 0 {
		t.Error("SnapshotComparisonResponse.Iterations should not be empty")
	}
	
	if len(response.TotalItems) == 0 {
		t.Error("SnapshotComparisonResponse.TotalItems should not be empty")
	}
}

// TestIterationComparison проверяет структуру сравнения итерации
func TestIterationComparison(t *testing.T) {
	dbID := 1
	iteration := IterationComparison{
		UploadID:        1,
		IterationNumber: 1,
		IterationLabel:  "test-iteration",
		StartedAt:       time.Now(),
		TotalItems:      100,
		TotalCatalogs:   5,
		DatabaseID:      &dbID,
	}
	
	if iteration.UploadID <= 0 {
		t.Error("IterationComparison.UploadID should be positive")
	}
	
	if iteration.IterationNumber <= 0 {
		t.Error("IterationComparison.IterationNumber should be positive")
	}
	
	if iteration.TotalItems < 0 {
		t.Error("IterationComparison.TotalItems should be non-negative")
	}
	
	if iteration.TotalCatalogs < 0 {
		t.Error("IterationComparison.TotalCatalogs should be non-negative")
	}
}

// TestSnapshotMetricsResponse проверяет структуру ответа метрик снимка
func TestSnapshotMetricsResponse(t *testing.T) {
	response := &SnapshotMetricsResponse{
		SnapshotID:    1,
		QualityScores: map[int]float64{1: 0.95, 2: 0.85},
		OverallTrend:  "improving",
		Improvements:  []QualityImprovement{},
	}
	
	if response.SnapshotID <= 0 {
		t.Error("SnapshotMetricsResponse.SnapshotID should be positive")
	}
	
	// Проверяем, что quality scores в допустимом диапазоне
	for uploadID, score := range response.QualityScores {
		if score < 0 || score > 1 {
			t.Errorf("Quality score for upload %d = %f, should be between 0 and 1", uploadID, score)
		}
	}
	
	// Проверяем валидность тренда
	validTrends := map[string]bool{"improving": true, "stable": true, "degrading": true}
	if !validTrends[response.OverallTrend] {
		t.Errorf("OverallTrend = %s, should be one of: improving, stable, degrading", response.OverallTrend)
	}
}

// TestSnapshotMetricsResponse_Empty проверяет пустой ответ метрик
func TestSnapshotMetricsResponse_Empty(t *testing.T) {
	response := &SnapshotMetricsResponse{
		SnapshotID:    1,
		QualityScores: make(map[int]float64),
		OverallTrend:  "stable",
		Improvements:  []QualityImprovement{},
	}
	
	if len(response.QualityScores) != 0 {
		t.Error("QualityScores should be empty")
	}
	
	if response.OverallTrend == "" {
		t.Error("OverallTrend should not be empty")
	}
	
	if len(response.Improvements) != 0 {
		t.Error("Improvements should be empty")
	}
}

// TestSnapshotMetricsResponse_QualityScores проверяет валидность quality scores
func TestSnapshotMetricsResponse_QualityScores(t *testing.T) {
	tests := []struct {
		name  string
		scores map[int]float64
		valid  bool
	}{
		{
			name:  "valid scores",
			scores: map[int]float64{1: 0.5, 2: 0.8, 3: 1.0},
			valid:  true,
		},
		{
			name:  "score below zero",
			scores: map[int]float64{1: -0.1},
			valid:  false,
		},
		{
			name:  "score above one",
			scores: map[int]float64{1: 1.5},
			valid:  false,
		},
		{
			name:  "empty scores",
			scores: make(map[int]float64),
			valid:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &SnapshotMetricsResponse{
				SnapshotID:    1,
				QualityScores: tt.scores,
			}
			
			valid := true
			for _, score := range response.QualityScores {
				if score < 0 || score > 1 {
					valid = false
					break
				}
			}
			
			if valid != tt.valid {
				t.Errorf("Quality scores validation = %v, want %v", valid, tt.valid)
			}
		})
	}
}

// TestIterationComparison_Sorting проверяет сортировку итераций
func TestIterationComparison_Sorting(t *testing.T) {
	iterations := []IterationComparison{
		{IterationNumber: 3, UploadID: 3, StartedAt: time.Now()},
		{IterationNumber: 1, UploadID: 1, StartedAt: time.Now()},
		{IterationNumber: 2, UploadID: 2, StartedAt: time.Now()},
	}
	
	// Проверяем, что итерации можно отсортировать по номеру
	for i := 0; i < len(iterations)-1; i++ {
		for j := i + 1; j < len(iterations); j++ {
			if iterations[i].IterationNumber > iterations[j].IterationNumber {
				// Меняем местами для сортировки
				iterations[i], iterations[j] = iterations[j], iterations[i]
			}
		}
	}
	
	// Проверяем, что итерации отсортированы
	for i := 0; i < len(iterations)-1; i++ {
		if iterations[i].IterationNumber > iterations[i+1].IterationNumber {
			t.Errorf("Iterations not sorted: %d > %d", 
				iterations[i].IterationNumber, iterations[i+1].IterationNumber)
		}
	}
}

