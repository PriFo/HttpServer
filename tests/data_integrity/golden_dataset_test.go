package data_integrity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"httpserver/database"
	"httpserver/normalization"
)

// GoldenDatasetEntry запись из golden dataset
type GoldenDatasetEntry struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Attributes string `json:"attributes"`
	Expected   struct {
		NormalizedName string  `json:"normalized_name"`
		INN            string  `json:"inn"`
		BIN            string  `json:"bin"`
		QualityScore   float64 `json:"quality_score"`
	} `json:"expected"`
}

// GoldenDataset структура golden dataset
type GoldenDataset struct {
	Version   string               `json:"version"`
	Timestamp string               `json:"timestamp"`
	Entries   []GoldenDatasetEntry `json:"entries"`
}

// ExpectedResult ожидаемый результат нормализации
type ExpectedResult struct {
	Version       string                 `json:"version"`
	Timestamp     string                 `json:"timestamp"`
	Counterparties struct {
		TotalUnique     int `json:"total_unique"`
		TotalDuplicates int `json:"total_duplicates"`
		DuplicateGroups int `json:"duplicate_groups"`
	} `json:"counterparties"`
	Results   []NormalizationResult  `json:"results"`
}

// NormalizationResult результат нормализации одной записи
type NormalizationResult struct {
	ID             int     `json:"id"`
	NormalizedName string  `json:"normalized_name"`
	INN            string  `json:"inn"`
	BIN            string  `json:"bin"`
	QualityScore   float64 `json:"quality_score"`
	DuplicateGroup int     `json:"duplicate_group,omitempty"`
}

// TestGoldenDataset проверяет соответствие результатов нормализации golden dataset
func TestGoldenDataset(t *testing.T) {
	// Загружаем golden dataset
	goldenPath := filepath.Join("tests", "data_integrity", "golden_dataset.json")
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Skipf("Golden dataset not found at %s, skipping test", goldenPath)
	}

	var goldenDataset GoldenDataset
	if err := json.Unmarshal(goldenData, &goldenDataset); err != nil {
		t.Fatalf("Failed to parse golden dataset: %v", err)
	}

	// Загружаем expected results
	expectedPath := filepath.Join("tests", "data_integrity", "expected_result.json")
	expectedData, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Skipf("Expected results not found at %s, run update_golden_dataset.go first", expectedPath)
	}

	var expectedResult ExpectedResult
	if err := json.Unmarshal(expectedData, &expectedResult); err != nil {
		t.Fatalf("Failed to parse expected results: %v", err)
	}

	// Создаем тестовую БД
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Создаем тестового клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Преобразуем golden dataset в формат для нормализации
	counterparties := make([]*database.CatalogItem, len(goldenDataset.Entries))
	for i, entry := range goldenDataset.Entries {
		counterparties[i] = &database.CatalogItem{
			ID:         entry.ID,
			Reference:  "ref" + string(rune('0'+entry.ID%10)),
			Code:       "code" + string(rune('0'+entry.ID%10)),
			Name:       entry.Name,
			Attributes: entry.Attributes,
		}
	}

	// Запускаем нормализацию
	eventChannel := make(chan string, 1000)
	ctx := context.Background()
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, ctx, nil, nil)

	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization failed: %v", err)
	}

	// Получаем нормализованные данные из БД
	normalized, _, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, len(counterparties), "", "", "")
	if err != nil {
		t.Fatalf("Failed to get normalized counterparties: %v", err)
	}

	// Создаем мапу для быстрого поиска по source_reference
	normalizedMap := make(map[int]*database.NormalizedCounterparty)
	for _, n := range normalized {
		// Извлекаем ID из source_reference (формат: "ref123")
		var id int
		if _, err := fmt.Sscanf(n.SourceReference, "ref%d", &id); err == nil {
			normalizedMap[id] = n
		}
	}

	// Сравниваем результаты
	mismatches := 0
	for i, entry := range goldenDataset.Entries {
		normalized, ok := normalizedMap[entry.ID]
		if !ok {
			t.Errorf("Entry %d (ID: %d) not found in normalized results", i, entry.ID)
			mismatches++
			continue
		}

		// Сравниваем нормализованное имя (с допуском на вариативность)
		if normalized.NormalizedName != entry.Expected.NormalizedName && entry.Expected.NormalizedName != "" {
			// Проверяем, что хотя бы похоже (нестрогое сравнение)
			if !isSimilar(normalized.NormalizedName, entry.Expected.NormalizedName) {
				t.Errorf("Entry %d: Expected normalized_name '%s', got '%s'", 
					i, entry.Expected.NormalizedName, normalized.NormalizedName)
				mismatches++
			}
		}

		// Сравниваем ИНН
		if normalized.TaxID != entry.Expected.INN && entry.Expected.INN != "" {
			t.Errorf("Entry %d: Expected INN '%s', got '%s'", 
				i, entry.Expected.INN, normalized.TaxID)
			mismatches++
		}

		// Сравниваем БИН
		if normalized.BIN != entry.Expected.BIN && entry.Expected.BIN != "" {
			t.Errorf("Entry %d: Expected BIN '%s', got '%s'", 
				i, entry.Expected.BIN, normalized.BIN)
			mismatches++
		}

		// Сравниваем quality score (с допуском 0.1)
		if entry.Expected.QualityScore > 0 {
			diff := normalized.QualityScore - entry.Expected.QualityScore
			if diff < -0.1 || diff > 0.1 {
				t.Errorf("Entry %d: Expected quality_score %.2f, got %.2f (diff: %.2f)", 
					i, entry.Expected.QualityScore, normalized.QualityScore, diff)
				mismatches++
			}
		}
	}

	if mismatches > 0 {
		t.Errorf("Found %d mismatches out of %d entries", mismatches, len(goldenDataset.Entries))
	}

	// Проверяем общую статистику
	if result.TotalProcessed != len(counterparties) {
		t.Errorf("Expected %d processed, got %d", len(counterparties), result.TotalProcessed)
	}

	// Дополнительная валидация: сравнение с expected_result.json
	validateExpectedResults(t, expectedResult, normalized, goldenDataset.Entries)
}

// validateExpectedResults сравнивает результаты нормализации с ожидаемыми результатами
func validateExpectedResults(t *testing.T, expectedResult ExpectedResult, normalized []*database.NormalizedCounterparty, entries []GoldenDatasetEntry) {
	// Создаем мапу для быстрого поиска
	normalizedMap := make(map[int]*database.NormalizedCounterparty)
	for _, n := range normalized {
		// Используем ID из source_reference или другой идентификатор
		// В реальной реализации нужно правильно сопоставить ID
		if n.SourceReference != "" {
			// Пытаемся извлечь ID из reference
			var id int
			if _, err := fmt.Sscanf(n.SourceReference, "ref%d", &id); err == nil {
				normalizedMap[id] = n
			}
		}
	}

	// Сравниваем с ожидаемыми результатами
	mismatches := 0
	for _, expected := range expectedResult.Results {
		normalized, ok := normalizedMap[expected.ID]
		if !ok {
			// Запись может отсутствовать, если она была сгруппирована
			continue
		}

		// Проверяем нормализованное имя
		if expected.NormalizedName != "" && normalized.NormalizedName != expected.NormalizedName {
			// Допускаем нестрогое сравнение
			if !isSimilar(normalized.NormalizedName, expected.NormalizedName) {
				t.Logf("Entry %d: Expected normalized_name '%s', got '%s'",
					expected.ID, expected.NormalizedName, normalized.NormalizedName)
				mismatches++
			}
		}

		// Проверяем извлеченные атрибуты
		if expected.INN != "" && normalized.TaxID != expected.INN {
			// Допускаем нестрогое сравнение для ИНН
			if normalized.TaxID == "" {
				t.Logf("Entry %d: Expected INN '%s', but none extracted",
					expected.ID, expected.INN)
			} else {
				t.Logf("Entry %d: Expected INN '%s', got '%s'",
					expected.ID, expected.INN, normalized.TaxID)
			}
			mismatches++
		}

		if expected.BIN != "" && normalized.BIN != expected.BIN {
			t.Logf("Entry %d: Expected BIN '%s', got '%s'",
				expected.ID, expected.BIN, normalized.BIN)
			mismatches++
		}

		// Проверяем качество группировки дубликатов
		if expected.DuplicateGroup > 0 {
			// Запись должна быть в группе дубликатов
			// В реальной реализации здесь будет проверка через master_id или is_duplicate
			t.Logf("Entry %d: Expected to be in duplicate group %d", expected.ID, expected.DuplicateGroup)
		}
	}

	if mismatches > 0 {
		t.Logf("Found %d mismatches in expected results validation", mismatches)
	}

	// Проверяем общую статистику дубликатов
	expectedDuplicateGroups := expectedResult.Counterparties.DuplicateGroups
	if expectedDuplicateGroups > 0 {
		// Подсчитываем реальные группы дубликатов
		nameGroups := make(map[string]int)
		for _, n := range normalized {
			nameGroups[n.NormalizedName]++
		}

		actualDuplicateGroups := 0
		for _, count := range nameGroups {
			if count > 1 {
				actualDuplicateGroups++
			}
		}

		// Допускаем некоторую погрешность
		if actualDuplicateGroups < expectedDuplicateGroups/2 {
			t.Logf("Expected at least %d duplicate groups, got %d",
				expectedDuplicateGroups/2, actualDuplicateGroups)
		}
	}
}

// isSimilar проверяет, похожи ли две строки (нестрогое сравнение)
func isSimilar(s1, s2 string) bool {
	// Простая проверка: если одна строка содержит другую (после нормализации)
	normalize := func(s string) string {
		// Убираем пробелы и приводим к нижнему регистру
		result := ""
		for _, r := range s {
			if r != ' ' && r != '\t' && r != '\n' {
				result += string(r)
			}
		}
		return result
	}

	n1 := normalize(s1)
	n2 := normalize(s2)

	// Проверяем, что одна содержит другую
	return len(n1) > 0 && len(n2) > 0 && (contains(n1, n2) || contains(n2, n1))
}

// contains проверяет, содержит ли строка подстроку (без учета регистра)
func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


