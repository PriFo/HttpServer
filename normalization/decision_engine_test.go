package normalization

import (
	"testing"
)

func TestDecisionEngine_collectCandidates(t *testing.T) {
	// Создаем mock дерево
	tree := NewKpvedTree()
	tree.NodeMap["25.94.11"] = &KpvedNode{Code: "25.94.11", Name: "Болты"}
	tree.NodeMap["27.32.11"] = &KpvedNode{Code: "27.32.11", Name: "Кабели"}

	de := &DecisionEngine{tree: tree}

	stage6 := &HierarchicalResult{
		FinalCode:       "25.94.11",
		FinalName:       "Болты",
		FinalConfidence: 0.95,
	}

	stage7 := &HierarchicalResult{
		FinalCode:       "27.32.11",
		FinalName:       "Кабели",
		FinalConfidence: 0.85,
	}

	stage8 := &FallbackResult{
		Code:       "32.99.5",
		Name:       "Прочие изделия",
		Confidence: 0.40,
	}

	candidates := de.collectCandidates(stage6, stage7, stage8)

	if len(candidates) != 3 {
		t.Errorf("collectCandidates returned %d candidates, want 3", len(candidates))
	}

	// Проверяем что все кандидаты присутствуют
	sources := make(map[string]bool)
	for _, c := range candidates {
		sources[c.Source] = true
	}

	if !sources["stage6"] || !sources["stage7"] || !sources["stage8"] {
		t.Error("Not all stages are present in candidates")
	}
}

func TestDecisionEngine_collectCandidates_EmptyResults(t *testing.T) {
	de := &DecisionEngine{tree: NewKpvedTree()}

	candidates := de.collectCandidates(nil, nil, nil)

	if len(candidates) != 0 {
		t.Errorf("collectCandidates with nil results returned %d candidates, want 0", len(candidates))
	}
}

func TestDecisionEngine_getSourcePriority(t *testing.T) {
	de := &DecisionEngine{}

	tests := []struct {
		name       string
		source     string
		confidence float64
		wantHigher string // Какой источник должен иметь более высокий приоритет
	}{
		{
			name:       "Stage7 high confidence beats Stage6 low confidence",
			source:     "stage7",
			confidence: 0.85,
			wantHigher: "stage7",
		},
		{
			name:       "Stage6 very high confidence beats Stage7 moderate",
			source:     "stage6",
			confidence: 0.95,
			wantHigher: "stage6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority7 := de.getSourcePriority("stage7", 0.75)
			priority6 := de.getSourcePriority("stage6", 0.95)

			if tt.wantHigher == "stage6" {
				if priority6 <= priority7 {
					t.Errorf("Stage6 priority (%.2f) should be higher than Stage7 (%.2f)", priority6, priority7)
				}
			}
		})
	}
}

func TestDecisionEngine_checkTypeCompatibility(t *testing.T) {
	tree := NewKpvedTree()
	tree.NodeMap["25.94.11"] = &KpvedNode{Code: "25.94.11", Name: "Болты"}          // Товар
	tree.NodeMap["71.20.1"] = &KpvedNode{Code: "71.20.1", Name: "Услуги испытаний"} // Услуга

	de := &DecisionEngine{tree: tree}

	tests := []struct {
		name         string
		code         string
		expectedType string
		want         bool
	}{
		{
			name:         "Product code matches product type",
			code:         "25.94.11",
			expectedType: "product",
			want:         true,
		},
		{
			name:         "Service code matches service type",
			code:         "71.20.1",
			expectedType: "service",
			want:         true,
		},
		{
			name:         "Product code doesn't match service type",
			code:         "25.94.11",
			expectedType: "service",
			want:         false,
		},
		{
			name:         "Unknown type always compatible",
			code:         "25.94.11",
			expectedType: "unknown",
			want:         true,
		},
		{
			name:         "Empty type always compatible",
			code:         "25.94.11",
			expectedType: "",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := de.checkTypeCompatibility(tt.code, tt.expectedType)
			if got != tt.want {
				t.Errorf("checkTypeCompatibility(%s, %s) = %v, want %v", tt.code, tt.expectedType, got, tt.want)
			}
		})
	}
}

func TestDecisionEngine_findAlternativeByType(t *testing.T) {
	tree := NewKpvedTree()
	tree.NodeMap["25.94.11"] = &KpvedNode{Code: "25.94.11", Name: "Болты"}          // Товар (25)
	tree.NodeMap["71.20.1"] = &KpvedNode{Code: "71.20.1", Name: "Услуги испытаний"} // Услуга (71)

	de := &DecisionEngine{tree: tree}

	candidates := []CandidateResult{
		{Source: "stage6", Code: "71.20.1", Confidence: 0.85},  // Услуга
		{Source: "stage7", Code: "25.94.11", Confidence: 0.75}, // Товар
	}

	// Ищем товар
	alternative := de.findAlternativeByType(candidates, "product")
	if alternative == nil {
		t.Fatal("findAlternativeByType returned nil, expected product alternative")
	}
	if alternative.Code != "25.94.11" {
		t.Errorf("findAlternativeByType returned %s, want 25.94.11", alternative.Code)
	}

	// Ищем услугу
	alternative = de.findAlternativeByType(candidates, "service")
	if alternative == nil {
		t.Fatal("findAlternativeByType returned nil, expected service alternative")
	}
	if alternative.Code != "71.20.1" {
		t.Errorf("findAlternativeByType returned %s, want 71.20.1", alternative.Code)
	}
}

func TestDecisionEngine_generateDecisionReason(t *testing.T) {
	de := &DecisionEngine{}

	tests := []struct {
		name             string
		best             CandidateResult
		validationResult CodeValidationResult
		expectedContains string
	}{
		{
			name: "Stage7 high confidence",
			best: CandidateResult{
				Source:     "stage7",
				Confidence: 0.85,
			},
			validationResult: CodeValidationResult{
				ValidationReason: "valid",
			},
			expectedContains: "stage7_high_confidence",
		},
		{
			name: "Stage6 keyword high confidence",
			best: CandidateResult{
				Source:     "stage6",
				Confidence: 0.95,
			},
			validationResult: CodeValidationResult{
				ValidationReason: "valid",
			},
			expectedContains: "stage6_keyword_high_confidence",
		},
		{
			name: "Fallback used",
			best: CandidateResult{
				Source:     "stage8",
				Confidence: 0.45,
			},
			validationResult: CodeValidationResult{
				ValidationReason: "valid",
			},
			expectedContains: "fallback_used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := de.generateDecisionReason(tt.best, tt.validationResult)
			if reason != tt.expectedContains {
				t.Errorf("generateDecisionReason() = %s, want to contain %s", reason, tt.expectedContains)
			}
		})
	}
}

func TestDecisionEngine_GetStatistics(t *testing.T) {
	de := &DecisionEngine{}

	decisions := []*FinalDecision{
		{
			Code:             "25.94.11",
			Confidence:       0.95,
			Method:           "stage6",
			ValidationPassed: true,
			DecisionReason:   "stage6_keyword_high_confidence",
		},
		{
			Code:             "27.32.11",
			Confidence:       0.85,
			Method:           "stage7",
			ValidationPassed: true,
			DecisionReason:   "stage7_high_confidence",
		},
		{
			Code:             "",
			Confidence:       0.0,
			Method:           "manual",
			ValidationPassed: false,
			DecisionReason:   "no_valid_classification",
		},
		{
			Code:             "32.99.5",
			Confidence:       0.40,
			Method:           "stage8",
			ValidationPassed: false,
			DecisionReason:   "fallback_used",
		},
	}

	stats := de.GetStatistics(decisions)

	// Проверяем total
	if stats["total"] != 4 {
		t.Errorf("Total = %v, want 4", stats["total"])
	}

	// Проверяем validation_passed
	if stats["validation_passed"] != 2 {
		t.Errorf("Validation passed = %v, want 2", stats["validation_passed"])
	}

	// Проверяем manual_review
	if stats["manual_review"] != 1 {
		t.Errorf("Manual review = %v, want 1", stats["manual_review"])
	}

	// Проверяем среднюю уверенность
	avgConf := stats["avg_confidence"].(float64)
	expectedAvg := (0.95 + 0.85 + 0.0 + 0.40) / 4.0
	tolerance := 0.001
	if avgConf < expectedAvg-tolerance || avgConf > expectedAvg+tolerance {
		t.Errorf("Avg confidence = %.3f, want %.3f", avgConf, expectedAvg)
	}

	// Проверяем распределение по методам
	byMethod := stats["by_method"].(map[string]int)
	if byMethod["stage6"] != 1 {
		t.Errorf("stage6 count = %d, want 1", byMethod["stage6"])
	}
	if byMethod["stage7"] != 1 {
		t.Errorf("stage7 count = %d, want 1", byMethod["stage7"])
	}
	if byMethod["stage8"] != 1 {
		t.Errorf("stage8 count = %d, want 1", byMethod["stage8"])
	}
	if byMethod["manual"] != 1 {
		t.Errorf("manual count = %d, want 1", byMethod["manual"])
	}
}

func TestDecisionEngine_GetStatistics_Empty(t *testing.T) {
	de := &DecisionEngine{}

	stats := de.GetStatistics([]*FinalDecision{})

	if stats["total"] != 0 {
		t.Errorf("Total = %v, want 0", stats["total"])
	}

	if stats["validation_passed"] != 0 {
		t.Errorf("Validation passed = %v, want 0", stats["validation_passed"])
	}

	if stats["manual_review"] != 0 {
		t.Errorf("Manual review = %v, want 0", stats["manual_review"])
	}

	if stats["avg_confidence"] != 0.0 {
		t.Errorf("Avg confidence = %v, want 0.0", stats["avg_confidence"])
	}
}

// Бенчмарк для проверки производительности
func BenchmarkDecisionEngine_Decide(b *testing.B) {
	tree := NewKpvedTree()
	tree.NodeMap["25.94.11"] = &KpvedNode{Code: "25.94.11", Name: "Болты"}

	de := &DecisionEngine{tree: tree}

	stage6 := &HierarchicalResult{
		FinalCode:       "25.94.11",
		FinalName:       "Болты",
		FinalConfidence: 0.95,
	}

	stage7 := &HierarchicalResult{
		FinalCode:       "25.94.11",
		FinalName:       "Болты",
		FinalConfidence: 0.85,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		de.collectCandidates(stage6, stage7, nil)
	}
}
