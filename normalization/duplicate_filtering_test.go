package normalization

import (
	"testing"

	"httpserver/database"
)

// TestFilterDuplicatesFromBatch_NoDuplicates проверяет, что батч без дубликатов возвращается без изменений
func TestFilterDuplicatesFromBatch_NoDuplicates(t *testing.T) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	events := make(chan string, 10)
	normalizer := NewNormalizer(db, events, nil)

	batch := []*database.NormalizedItem{
		{
			SourceReference:     "ref1",
			SourceName:          "Item 1",
			Code:                "code1",
			NormalizedName:      "item 1",
			NormalizedReference: "item 1",
			Category:            "category1",
			MergedCount:         1,
		},
		{
			SourceReference:     "ref2",
			SourceName:          "Item 2",
			Code:                "code2",
			NormalizedName:      "item 2",
			NormalizedReference: "item 2",
			Category:            "category2",
			MergedCount:         1,
		},
	}

	filtered, err := normalizer.filterDuplicatesFromBatch(batch)
	if err != nil {
		t.Fatalf("Failed to filter duplicates: %v", err)
	}

	if len(filtered) != len(batch) {
		t.Errorf("Expected %d items, got %d", len(batch), len(filtered))
	}
}

// TestFilterDuplicatesFromBatch_WithDuplicates проверяет фильтрацию дубликатов по normalized_name
func TestFilterDuplicatesFromBatch_WithDuplicates(t *testing.T) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	// Создаем существующую запись в БД
	existingItem := &database.NormalizedItem{
		SourceReference:     "existing_ref",
		SourceName:          "Existing Item",
		Code:                "existing_code",
		NormalizedName:      "existing item",
		NormalizedReference: "existing item",
		Category:            "category1",
		MergedCount:         1,
	}

	// Вставляем существующую запись напрямую в БД
	_, err = db.InsertNormalizedItemsWithAttributesBatch(
		[]*database.NormalizedItem{existingItem},
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to insert existing item: %v", err)
	}

	events := make(chan string, 10)
	normalizer := NewNormalizer(db, events, nil)

	// Создаем батч с дубликатом
	batch := []*database.NormalizedItem{
		{
			SourceReference:     "ref1",
			SourceName:          "Item 1",
			Code:                "code1",
			NormalizedName:      "existing item", // Дубликат по normalized_name
			NormalizedReference: "existing item",
			Category:            "category1",
			MergedCount:         1,
		},
		{
			SourceReference:     "ref2",
			SourceName:          "Item 2",
			Code:                "code2",
			NormalizedName:      "new item",
			NormalizedReference: "new item",
			Category:            "category2",
			MergedCount:         1,
		},
	}

	filtered, err := normalizer.filterDuplicatesFromBatch(batch)
	if err != nil {
		t.Fatalf("Failed to filter duplicates: %v", err)
	}

	// Должен остаться только один элемент (без дубликата)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 item after filtering, got %d", len(filtered))
	}

	if filtered[0].NormalizedName != "new item" {
		t.Errorf("Expected 'new item', got '%s'", filtered[0].NormalizedName)
	}
}

// TestFilterDuplicatesFromBatch_ByCode проверяет фильтрацию дубликатов по code
// Проверка по code работает только для элементов, которые уже были найдены по normalized_name
func TestFilterDuplicatesFromBatch_ByCode(t *testing.T) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	// Создаем существующую запись в БД с code и normalized_name
	existingItem := &database.NormalizedItem{
		SourceReference:     "existing_ref",
		SourceName:          "Existing Item",
		Code:                "EXISTING_CODE",
		NormalizedName:      "existing item",
		NormalizedReference: "existing item",
		Category:            "category1",
		MergedCount:         1,
	}

	// Вставляем существующую запись напрямую в БД
	_, err = db.InsertNormalizedItemsWithAttributesBatch(
		[]*database.NormalizedItem{existingItem},
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to insert existing item: %v", err)
	}

	events := make(chan string, 10)
	normalizer := NewNormalizer(db, events, nil)

	// Создаем батч с дубликатом по normalized_name и code
	// Элемент с таким же normalized_name и code должен быть отфильтрован
	batch := []*database.NormalizedItem{
		{
			SourceReference:     "ref1",
			SourceName:          "Item 1",
			Code:                "EXISTING_CODE", // Дубликат по code и normalized_name
			NormalizedName:      "existing item", // Дубликат по normalized_name
			NormalizedReference: "existing item",
			Category:            "category1",
			MergedCount:         1,
		},
		{
			SourceReference:     "ref2",
			SourceName:          "Item 2",
			Code:                "NEW_CODE",
			NormalizedName:      "new item",
			NormalizedReference: "new item",
			Category:            "category2",
			MergedCount:         1,
		},
	}

	filtered, err := normalizer.filterDuplicatesFromBatch(batch)
	if err != nil {
		t.Fatalf("Failed to filter duplicates: %v", err)
	}

	// Должен остаться только один элемент (без дубликата)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 item after filtering, got %d", len(filtered))
	}

	if filtered[0].Code != "NEW_CODE" {
		t.Errorf("Expected 'NEW_CODE', got '%s'", filtered[0].Code)
	}
}

// TestFilterDuplicatesFromBatch_EmptyBatch проверяет обработку пустого батча
func TestFilterDuplicatesFromBatch_EmptyBatch(t *testing.T) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	events := make(chan string, 10)
	normalizer := NewNormalizer(db, events, nil)

	batch := []*database.NormalizedItem{}

	filtered, err := normalizer.filterDuplicatesFromBatch(batch)
	if err != nil {
		t.Fatalf("Failed to filter duplicates: %v", err)
	}

	if len(filtered) != 0 {
		t.Errorf("Expected 0 items, got %d", len(filtered))
	}
}

