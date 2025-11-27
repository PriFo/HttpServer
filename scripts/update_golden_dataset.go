//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"httpserver/database"
	"httpserver/normalization"
)

// GoldenDatasetEntry запись из golden dataset
type GoldenDatasetEntry struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Attributes string `json:"attributes"`
}

// GoldenDataset структура golden dataset
type GoldenDataset struct {
	Version   string               `json:"version"`
	Timestamp string               `json:"timestamp"`
	Entries   []GoldenDatasetEntry `json:"entries"`
}

// ExpectedResult ожидаемый результат нормализации
type ExpectedResult struct {
	Version   string                 `json:"version"`
	Timestamp string                 `json:"timestamp"`
	Results    []NormalizationResult `json:"results"`
}

// NormalizationResult результат нормализации одной записи
type NormalizationResult struct {
	ID             int     `json:"id"`
	NormalizedName string  `json:"normalized_name"`
	INN            string  `json:"inn"`
	BIN            string  `json:"bin"`
	QualityScore   float64 `json:"quality_score"`
}

func main() {
	// Загружаем golden dataset
	goldenPath := filepath.Join("tests", "data_integrity", "golden_dataset.json")
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		log.Fatalf("Failed to read golden dataset: %v", err)
	}

	var goldenDataset GoldenDataset
	if err := json.Unmarshal(goldenData, &goldenDataset); err != nil {
		log.Fatalf("Failed to parse golden dataset: %v", err)
	}

	// Создаем тестовую БД
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		log.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	// Создаем тестового клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}

	// Преобразуем golden dataset в формат для нормализации
	counterparties := make([]*database.CatalogItem, len(goldenDataset.Entries))
	for i, entry := range goldenDataset.Entries {
		counterparties[i] = &database.CatalogItem{
			ID:         entry.ID,
			Reference:  "ref" + fmt.Sprintf("%d", entry.ID),
			Code:       "code" + fmt.Sprintf("%d", entry.ID),
			Name:       entry.Name,
			Attributes: entry.Attributes,
		}
	}

	// Запускаем нормализацию с детерминированными моками
	// В реальном сценарии здесь должны быть замоканные AI провайдеры
	eventChannel := make(chan string, 1000)
	stopCheck := func() bool { return false }
	
	// Используем nil для nameNormalizer, так как мы хотим использовать только эталоны
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, stopCheck, nil)

	fmt.Println("Running normalization on golden dataset...")
	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		log.Fatalf("ProcessNormalization failed: %v", err)
	}

	fmt.Printf("Processed %d records\n", result.TotalProcessed)

	// Получаем нормализованные данные из БД
	normalized, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, len(counterparties))
	if err != nil {
		log.Fatalf("Failed to get normalized counterparties: %v", err)
	}

	// Создаем мапу для быстрого поиска
	normalizedMap := make(map[int]*database.NormalizedCounterparty)
	for _, n := range normalized {
		normalizedMap[n.SourceReferenceID] = n
	}

	// Формируем expected results
	results := make([]NormalizationResult, len(goldenDataset.Entries))
	for i, entry := range goldenDataset.Entries {
		normalized, ok := normalizedMap[entry.ID]
		if !ok {
			log.Printf("Warning: Entry %d not found in normalized results", entry.ID)
			results[i] = NormalizationResult{
				ID:             entry.ID,
				NormalizedName: entry.Name, // Используем исходное имя
				INN:            "",
				BIN:            "",
				QualityScore:   0.5,
			}
			continue
		}

		results[i] = NormalizationResult{
			ID:             entry.ID,
			NormalizedName: normalized.NormalizedName,
			INN:            normalized.INN,
			BIN:            normalized.BIN,
			QualityScore:   normalized.QualityScore,
		}
	}

	expectedResult := ExpectedResult{
		Version:   "1.0.0",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Results:   results,
	}

	// Сохраняем expected results
	expectedPath := filepath.Join("tests", "data_integrity", "expected_result.json")
	data, err := json.MarshalIndent(expectedResult, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal expected results: %v", err)
	}

	if err := os.WriteFile(expectedPath, data, 0644); err != nil {
		log.Fatalf("Failed to write expected results: %v", err)
	}

	fmt.Printf("Expected results saved to %s\n", expectedPath)
}


