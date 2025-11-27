package algorithms

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ExportFormat формат экспорта
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatTSV  ExportFormat = "tsv"
)

// SimilarityExporter экспортирует результаты анализа
type SimilarityExporter struct {
	results *SimilarityAnalysisResult
}

// NewSimilarityExporter создает новый экспортер
func NewSimilarityExporter(results *SimilarityAnalysisResult) *SimilarityExporter {
	return &SimilarityExporter{
		results: results,
	}
}

// Export экспортирует результаты в указанный формат
func (se *SimilarityExporter) Export(filepath string, format ExportFormat) error {
	switch format {
	case ExportFormatJSON:
		return se.exportJSON(filepath)
	case ExportFormatCSV:
		return se.exportCSV(filepath)
	case ExportFormatTSV:
		return se.exportTSV(filepath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportJSON экспортирует в JSON
func (se *SimilarityExporter) exportJSON(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(se.results)
}

// exportCSV экспортирует в CSV
func (se *SimilarityExporter) exportCSV(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Заголовки
	headers := []string{
		"String1", "String2", "Similarity", "IsDuplicate", "Confidence",
		"JaroWinkler", "LCS", "Phonetic", "Ngram", "Jaccard",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Данные
	for _, pair := range se.results.Pairs {
		record := []string{
			pair.S1,
			pair.S2,
			fmt.Sprintf("%.4f", pair.Similarity),
			fmt.Sprintf("%v", pair.IsDuplicate),
			fmt.Sprintf("%.4f", pair.Confidence),
			fmt.Sprintf("%.4f", pair.Breakdown.JaroWinkler),
			fmt.Sprintf("%.4f", pair.Breakdown.LCS),
			fmt.Sprintf("%.4f", pair.Breakdown.Phonetic),
			fmt.Sprintf("%.4f", pair.Breakdown.Ngram),
			fmt.Sprintf("%.4f", pair.Breakdown.Jaccard),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// exportTSV экспортирует в TSV (табуляция)
func (se *SimilarityExporter) exportTSV(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Заголовки
	headers := "String1\tString2\tSimilarity\tIsDuplicate\tConfidence\tJaroWinkler\tLCS\tPhonetic\tNgram\tJaccard\n"
	if _, err := file.WriteString(headers); err != nil {
		return err
	}

	// Данные
	for _, pair := range se.results.Pairs {
		line := fmt.Sprintf("%s\t%s\t%.4f\t%v\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\n",
			pair.S1, pair.S2, pair.Similarity, pair.IsDuplicate, pair.Confidence,
			pair.Breakdown.JaroWinkler, pair.Breakdown.LCS, pair.Breakdown.Phonetic,
			pair.Breakdown.Ngram, pair.Breakdown.Jaccard)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// ExportReport экспортирует отчет с метриками
func (se *SimilarityExporter) ExportReport(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	report := fmt.Sprintf(`# Отчет по анализу схожести
Дата: %s

## Статистика

- Всего пар: %d
- Дубликаты: %d (%.1f%%)
- Не дубликаты: %d (%.1f%%)
- Средняя схожесть: %.4f
- Минимальная схожесть: %.4f
- Максимальная схожесть: %.4f
- Медианная схожесть: %.4f

## Рекомендации

`,
		time.Now().Format("2006-01-02 15:04:05"),
		se.results.Statistics.TotalPairs,
		se.results.Statistics.DuplicatePairs,
		float64(se.results.Statistics.DuplicatePairs)/float64(se.results.Statistics.TotalPairs)*100,
		se.results.Statistics.NonDuplicatePairs,
		float64(se.results.Statistics.NonDuplicatePairs)/float64(se.results.Statistics.TotalPairs)*100,
		se.results.Statistics.AverageSimilarity,
		se.results.Statistics.MinSimilarity,
		se.results.Statistics.MaxSimilarity,
		se.results.Statistics.MedianSimilarity,
	)

	for i, rec := range se.results.Recommendations {
		report += fmt.Sprintf("%d. %s\n", i+1, rec)
	}

	report += "\n## Детали пар\n\n"
	report += "| String1 | String2 | Similarity | IsDuplicate | Confidence |\n"
	report += "|---------|---------|------------|-------------|------------|\n"

	for _, pair := range se.results.Pairs {
		report += fmt.Sprintf("| %s | %s | %.4f | %v | %.4f |\n",
			pair.S1, pair.S2, pair.Similarity, pair.IsDuplicate, pair.Confidence)
	}

	_, err = file.WriteString(report)
	return err
}

// ImportTrainingPairs импортирует обучающие пары из файла
func ImportTrainingPairs(filepath string, format ExportFormat) ([]SimilarityTestPair, error) {
	switch format {
	case ExportFormatJSON:
		return importJSON(filepath)
	case ExportFormatCSV:
		return importCSV(filepath)
	default:
		return nil, fmt.Errorf("unsupported import format: %s", format)
	}
}

// importJSON импортирует из JSON
func importJSON(filepath string) ([]SimilarityTestPair, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var pairs []SimilarityTestPair
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&pairs); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return pairs, nil
}

// importCSV импортирует из CSV
func importCSV(filepath string) ([]SimilarityTestPair, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must have at least header and one data row")
	}

	pairs := make([]SimilarityTestPair, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 3 {
			continue
		}

		isDuplicate := false
		if len(record) >= 4 {
			isDuplicate = record[3] == "true" || record[3] == "1"
		}

		pairs = append(pairs, SimilarityTestPair{
			S1:          record[0],
			S2:          record[1],
			IsDuplicate: isDuplicate,
		})
	}

	return pairs, nil
}

