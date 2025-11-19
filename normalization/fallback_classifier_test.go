package normalization

import (
	"database/sql"
	"testing"
)

// Mock implementation of KpvedDB for testing
type MockKpvedDB struct{}

func (m *MockKpvedDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockKpvedDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

func TestFallbackClassifier_tryParentCode(t *testing.T) {
	// Создаем mock дерево с тестовыми данными
	tree := NewKpvedTree()
	tree.NodeMap["27.32"] = &KpvedNode{
		Code: "27.32",
		Name: "Кабели электрические прочие",
	}
	tree.NodeMap["27.32.11"] = &KpvedNode{
		Code: "27.32.11",
		Name: "Кабели волоконно-оптические",
	}

	fc := &FallbackClassifier{
		tree:              tree,
		keywordClassifier: NewKeywordClassifier(),
		productDetector:   NewProductServiceDetector(),
	}

	tests := []struct {
		name           string
		code           string
		normalizedName string
		wantCode       string
		wantNil        bool
	}{
		{
			name:           "Parent code from 3-level",
			code:           "27.32.11",
			normalizedName: "Кабель специальный",
			wantCode:       "27.32",
			wantNil:        false,
		},
		{
			name:           "No parent for 2-level",
			code:           "27.32",
			normalizedName: "Кабель",
			wantCode:       "",
			wantNil:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.tryParentCode(tt.code, tt.normalizedName)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if result.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", result.Code, tt.wantCode)
			}

			if result.Method != "parent_code" {
				t.Errorf("Method = %s, want parent_code", result.Method)
			}

			if result.Confidence != 0.55 {
				t.Errorf("Confidence = %.2f, want 0.55", result.Confidence)
			}
		})
	}
}

func TestFallbackClassifier_getParentCode(t *testing.T) {
	fc := &FallbackClassifier{}

	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "3-level to 2-level",
			code: "27.32.11",
			want: "27.32",
		},
		{
			name: "4-level to 3-level",
			code: "27.32.11.123",
			want: "27.32.11",
		},
		{
			name: "2-level has no parent",
			code: "27.32",
			want: "",
		},
		{
			name: "Invalid code",
			code: "27",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fc.getParentCode(tt.code)
			if got != tt.want {
				t.Errorf("getParentCode(%s) = %s, want %s", tt.code, got, tt.want)
			}
		})
	}
}

func TestFallbackClassifier_shouldRequireManualReview(t *testing.T) {
	fc := &FallbackClassifier{}

	tests := []struct {
		name           string
		confidence     float64
		method         string
		normalizedName string
		want           bool
	}{
		{
			name:           "Low confidence",
			confidence:     0.4,
			method:         "parent_code",
			normalizedName: "Болт М12",
			want:           true,
		},
		{
			name:           "Category default always requires review",
			confidence:     0.6,
			method:         "category_default",
			normalizedName: "Изделие",
			want:           true,
		},
		{
			name:           "Very short name",
			confidence:     0.6,
			method:         "keyword_simple",
			normalizedName: "ABC",
			want:           true,
		},
		{
			name:           "Digits only",
			confidence:     0.6,
			method:         "similar_names",
			normalizedName: "123-456",
			want:           true,
		},
		{
			name:           "Normal case",
			confidence:     0.6,
			method:         "parent_code",
			normalizedName: "Болт М12 оцинкованный",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fc.shouldRequireManualReview(tt.confidence, tt.method, tt.normalizedName)
			if got != tt.want {
				t.Errorf("shouldRequireManualReview() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFallbackClassifier_tryCategoryDefault(t *testing.T) {
	// Создаем mock дерево
	tree := NewKpvedTree()
	tree.NodeMap["32.99.5"] = &KpvedNode{
		Code: "32.99.5",
		Name: "Прочие готовые изделия, не включенные в другие группировки",
	}
	tree.NodeMap["96.09.1"] = &KpvedNode{
		Code: "96.09.1",
		Name: "Услуги индивидуальные прочие, не включенные в другие группировки",
	}

	fc := &FallbackClassifier{
		tree:            tree,
		productDetector: NewProductServiceDetector(),
	}

	tests := []struct {
		name           string
		normalizedName string
		category       string
		wantCode       string
	}{
		{
			name:           "Product default",
			normalizedName: "изделие неизвестное",
			category:       "unknown",
			wantCode:       "32.99.5",
		},
		{
			name:           "Service default",
			normalizedName: "услуга консультации",
			category:       "services",
			wantCode:       "96.09.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.tryCategoryDefault(tt.normalizedName, tt.category)

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if result.Code != tt.wantCode {
				t.Errorf("Code = %s, want %s", result.Code, tt.wantCode)
			}

			if result.Method != "category_default" {
				t.Errorf("Method = %s, want category_default", result.Method)
			}

			if !result.ManualReviewRequired {
				t.Error("ManualReviewRequired should be true for category_default")
			}

			if result.Confidence != 0.35 {
				t.Errorf("Confidence = %.2f, want 0.35", result.Confidence)
			}
		})
	}
}

func TestFallbackClassifier_GetStatistics(t *testing.T) {
	fc := &FallbackClassifier{}

	results := []*FallbackResult{
		{
			Code:                 "27.32",
			Confidence:           0.55,
			Method:               "parent_code",
			ManualReviewRequired: false,
		},
		{
			Code:                 "25.94.11",
			Confidence:           0.45,
			Method:               "keyword_simple",
			ManualReviewRequired: false,
		},
		{
			Code:                 "32.99.5",
			Confidence:           0.35,
			Method:               "category_default",
			ManualReviewRequired: true,
		},
		{
			Code:                 "27.32",
			Confidence:           0.55,
			Method:               "parent_code",
			ManualReviewRequired: true,
		},
	}

	stats := fc.GetStatistics(results)

	// Проверяем общее количество
	if stats["total"] != 4 {
		t.Errorf("Total = %v, want 4", stats["total"])
	}

	// Проверяем manual review
	if stats["manual_review"] != 2 {
		t.Errorf("Manual review = %v, want 2", stats["manual_review"])
	}

	// Проверяем среднюю уверенность
	avgConf := stats["avg_confidence"].(float64)
	expectedAvg := (0.55 + 0.45 + 0.35 + 0.55) / 4.0
	tolerance := 0.001
	if avgConf < expectedAvg-tolerance || avgConf > expectedAvg+tolerance {
		t.Errorf("Avg confidence = %.3f, want %.3f", avgConf, expectedAvg)
	}

	// Проверяем распределение по методам
	byMethod := stats["by_method"].(map[string]int)
	if byMethod["parent_code"] != 2 {
		t.Errorf("parent_code count = %d, want 2", byMethod["parent_code"])
	}
	if byMethod["keyword_simple"] != 1 {
		t.Errorf("keyword_simple count = %d, want 1", byMethod["keyword_simple"])
	}
	if byMethod["category_default"] != 1 {
		t.Errorf("category_default count = %d, want 1", byMethod["category_default"])
	}
}

func TestFallbackClassifier_GetStatistics_Empty(t *testing.T) {
	fc := &FallbackClassifier{}

	stats := fc.GetStatistics([]*FallbackResult{})

	if stats["total"] != 0 {
		t.Errorf("Total = %v, want 0", stats["total"])
	}

	if stats["manual_review"] != 0 {
		t.Errorf("Manual review = %v, want 0", stats["manual_review"])
	}

	if stats["avg_confidence"] != 0.0 {
		t.Errorf("Avg confidence = %v, want 0.0", stats["avg_confidence"])
	}
}

// Бенчмарк для проверки производительности
func BenchmarkFallbackClassifier_tryParentCode(b *testing.B) {
	tree := NewKpvedTree()
	tree.NodeMap["27.32"] = &KpvedNode{
		Code: "27.32",
		Name: "Кабели электрические прочие",
	}

	fc := &FallbackClassifier{
		tree:              tree,
		keywordClassifier: NewKeywordClassifier(),
		productDetector:   NewProductServiceDetector(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fc.tryParentCode("27.32.11", "Кабель специальный")
	}
}
