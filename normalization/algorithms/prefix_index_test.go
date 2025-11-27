package algorithms

import (
	"testing"
)

func TestPrefixIndex_Add(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	pi.Add(0, "масло")
	pi.Add(1, "масло сливочное")
	pi.Add(2, "кабель")
	
	stats := pi.GetStats()
	if stats.TotalItems != 3 {
		t.Errorf("Expected 3 items, got %d", stats.TotalItems)
	}
	if stats.TotalPrefixes == 0 {
		t.Error("Expected at least 1 prefix")
	}
}

func TestPrefixIndex_GetCandidatesExact(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	texts := []string{
		"масло сливочное",
		"масло подсолнечное",
		"кабель медный",
		"кабель алюминиевый",
		"шкаф деревянный",
	}
	
	pi.AddBatch(texts)
	
	// Ищем кандидатов для "масло сливочное"
	candidates := pi.GetCandidatesExact(0, "масло сливочное")
	
	// Должен найти "масло подсолнечное" (индекс 1)
	found := false
	for _, idx := range candidates {
		if idx == 1 {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected to find candidate with index 1")
	}
	
	// Не должен найти "кабель" или "шкаф"
	for _, idx := range candidates {
		if idx == 2 || idx == 3 || idx == 4 {
			t.Errorf("Unexpected candidate found: %d", idx)
		}
	}
}

func TestPrefixIndex_GetCandidates(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	texts := []string{
		"масло",
		"масла", // Похожий префикс
		"кабель",
	}
	
	pi.AddBatch(texts)
	
	// Ищем кандидатов для "масло"
	candidates := pi.GetCandidates(0, "масло")
	
	// Должен найти "масла" (индекс 1) из-за похожего префикса
	if len(candidates) == 0 {
		t.Error("Expected to find at least one candidate")
	}
}

func TestPrefixIndex_Remove(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	pi.Add(0, "масло")
	pi.Add(1, "масло сливочное")
	
	stats := pi.GetStats()
	if stats.TotalItems != 2 {
		t.Errorf("Expected 2 items before removal, got %d", stats.TotalItems)
	}
	
	pi.Remove(0)
	
	stats = pi.GetStats()
	if stats.TotalItems != 1 {
		t.Errorf("Expected 1 item after removal, got %d", stats.TotalItems)
	}
}

func TestPrefixIndex_Update(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	pi.Add(0, "масло")
	
	candidates := pi.GetCandidatesExact(0, "масло")
	if len(candidates) > 0 {
		t.Error("Should not find candidates for itself")
	}
	
	pi.Update(0, "масло", "кабель")
	
	// Теперь префикс должен быть другим
	prefixes := pi.GetPrefixes(0)
	if len(prefixes) == 0 {
		t.Error("Expected prefixes after update")
	}
}

func TestPrefixIndex_FilterByPrefix(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	texts := []string{
		"масло сливочное",
		"масло подсолнечное",
		"кабель медный",
		"кабель алюминиевый",
	}
	
	pi.AddBatch(texts)
	
	// Фильтруем все индексы по префиксу "мас"
	allIndices := []int{0, 1, 2, 3}
	filtered := pi.FilterByPrefix("мас", allIndices)
	
	// Должны остаться только индексы 0 и 1
	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered items, got %d", len(filtered))
	}
	
	found0, found1 := false, false
	for _, idx := range filtered {
		if idx == 0 {
			found0 = true
		}
		if idx == 1 {
			found1 = true
		}
	}
	
	if !found0 || !found1 {
		t.Error("Expected to find indices 0 and 1 in filtered results")
	}
}

func TestPrefixIndex_PrefixSimilarity(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	tests := []struct {
		p1       string
		p2       string
		min      float64
		max      float64
	}{
		{"мас", "мас", 0.99, 1.01}, // Полное совпадение
		{"мас", "мал", 0.5, 0.8},   // 2 из 3 совпадают (примерно 0.67)
		{"мас", "каб", 0.0, 0.5},  // Разные префиксы (может быть небольшое совпадение из-за кодировки)
		{"", "", 0.99, 1.01},
		{"мас", "", 0.0, 0.1},
	}
	
	for _, tt := range tests {
		result := pi.prefixSimilarity(tt.p1, tt.p2)
		if result < tt.min || result > tt.max {
			t.Errorf("prefixSimilarity(%q, %q) = %f, want between %f and %f", tt.p1, tt.p2, result, tt.min, tt.max)
		}
	}
}

func TestPrefixIndex_Clear(t *testing.T) {
	pi := NewPrefixIndex(3, 3)
	
	pi.Add(0, "масло")
	pi.Add(1, "кабель")
	
	stats := pi.GetStats()
	if stats.TotalItems != 2 {
		t.Errorf("Expected 2 items before clear, got %d", stats.TotalItems)
	}
	
	pi.Clear()
	
	stats = pi.GetStats()
	if stats.TotalItems != 0 {
		t.Errorf("Expected 0 items after clear, got %d", stats.TotalItems)
	}
}

