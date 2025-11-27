package algorithms

import (
	"errors"
	"os"
	"testing"
)

func TestSimilarityExporter(t *testing.T) {
	// Создаем тестовые данные
	pairs := []SimilarityPair{
		{"ООО Рога и Копыта", "Рога и Копыта ООО"},
		{"Кабель ВВГнг", "Кабель ВВГ"},
	}

	analyzer := NewSimilarityAnalyzer(nil)
	result := analyzer.AnalyzePairs(pairs, 0.75)

	exporter := NewSimilarityExporter(result)

	// Тестируем экспорт в JSON
	t.Run("ExportJSON", func(t *testing.T) {
		filepath := "test_export.json"
		defer os.Remove(filepath)

		if err := exporter.Export(filepath, ExportFormatJSON); err != nil {
			t.Fatalf("Failed to export JSON: %v", err)
		}

		// Проверяем, что файл создан
		if _, err := os.Stat(filepath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Fatal("JSON file was not created")
			}
			t.Fatalf("Error checking JSON file: %v", err)
		}
	})

	// Тестируем экспорт в CSV
	t.Run("ExportCSV", func(t *testing.T) {
		filepath := "test_export.csv"
		defer os.Remove(filepath)

		if err := exporter.Export(filepath, ExportFormatCSV); err != nil {
			t.Fatalf("Failed to export CSV: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Fatal("CSV file was not created")
			}
			t.Fatalf("Error checking CSV file: %v", err)
		}
	})

	// Тестируем экспорт в TSV
	t.Run("ExportTSV", func(t *testing.T) {
		filepath := "test_export.tsv"
		defer os.Remove(filepath)

		if err := exporter.Export(filepath, ExportFormatTSV); err != nil {
			t.Fatalf("Failed to export TSV: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Fatal("TSV file was not created")
			}
			t.Fatalf("Error checking TSV file: %v", err)
		}
	})

	// Тестируем экспорт отчета
	t.Run("ExportReport", func(t *testing.T) {
		filepath := "test_export.md"
		defer os.Remove(filepath)

		if err := exporter.ExportReport(filepath); err != nil {
			t.Fatalf("Failed to export report: %v", err)
		}

		if _, err := os.Stat(filepath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Fatal("Report file was not created")
			}
			t.Fatalf("Error checking report file: %v", err)
		}
	})
}

func TestImportTrainingPairs(t *testing.T) {
	// Создаем тестовый JSON файл
	jsonFile := "test_import.json"
	jsonContent := `[
		{"s1": "ООО Рога и Копыта", "s2": "ООО Рога и Копыта", "is_duplicate": true},
		{"s1": "ООО Рога и Копыта", "s2": "ООО Другая Компания", "is_duplicate": false}
	]`
	
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}
	defer os.Remove(jsonFile)

	// Тестируем импорт из JSON
	t.Run("ImportJSON", func(t *testing.T) {
		pairs, err := ImportTrainingPairs(jsonFile, ExportFormatJSON)
		if err != nil {
			t.Fatalf("Failed to import JSON: %v", err)
		}

		if len(pairs) != 2 {
			t.Errorf("Expected 2 pairs, got %d", len(pairs))
		}

		if pairs[0].S1 != "ООО Рога и Копыта" {
			t.Errorf("Expected first pair S1 to be 'ООО Рога и Копыта', got '%s'", pairs[0].S1)
		}

		if !pairs[0].IsDuplicate {
			t.Error("Expected first pair to be duplicate")
		}
	})

	// Создаем тестовый CSV файл
	csvFile := "test_import.csv"
	csvContent := `String1,String2,IsDuplicate
ООО Рога и Копыта,ООО Рога и Копыта,true
ООО Рога и Копыта,ООО Другая Компания,false
`
	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("Failed to create test CSV file: %v", err)
	}
	defer os.Remove(csvFile)

	// Тестируем импорт из CSV
	t.Run("ImportCSV", func(t *testing.T) {
		pairs, err := ImportTrainingPairs(csvFile, ExportFormatCSV)
		if err != nil {
			t.Fatalf("Failed to import CSV: %v", err)
		}

		if len(pairs) != 2 {
			t.Errorf("Expected 2 pairs, got %d", len(pairs))
		}
	})
}

