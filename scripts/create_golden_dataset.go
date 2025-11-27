//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"httpserver/database"
)

// GoldenEntry запись из golden dataset
type GoldenEntry struct {
	ID         int    `json:"id"`
	EntityType string `json:"entity_type"` // "counterparty" или "nomenclature"
	Reference  string `json:"reference"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Attributes string `json:"attributes"`
	Expected   struct {
		NormalizedName string            `json:"normalized_name"`
		INN            string            `json:"inn,omitempty"`
		BIN            string            `json:"bin,omitempty"`
		KPP            string            `json:"kpp,omitempty"`
		Category       string            `json:"category,omitempty"`
		QualityScore   float64           `json:"quality_score"`
		DuplicateGroup int               `json:"duplicate_group,omitempty"` // ID группы дубликатов
		IsMaster       bool              `json:"is_master,omitempty"`       // Является ли мастер-записью
		ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
	} `json:"expected"`
}

// GoldenDataset структура golden dataset
type GoldenDataset struct {
	Version   string        `json:"version"`
	Timestamp string        `json:"timestamp"`
	Entries   []GoldenEntry `json:"entries"`
}

// ExpectedResult ожидаемый результат нормализации
type ExpectedResult struct {
	Version        string `json:"version"`
	Timestamp      string `json:"timestamp"`
	Counterparties struct {
		TotalUnique     int `json:"total_unique"`
		TotalDuplicates int `json:"total_duplicates"`
		DuplicateGroups int `json:"duplicate_groups"`
	} `json:"counterparties"`
	Nomenclature struct {
		TotalUnique     int `json:"total_unique"`
		TotalDuplicates int `json:"total_duplicates"`
		DuplicateGroups int `json:"duplicate_groups"`
	} `json:"nomenclature"`
	Results []ResultEntry `json:"results"`
}

// ResultEntry результат нормализации одной записи
type ResultEntry struct {
	ID             int     `json:"id"`
	EntityType     string  `json:"entity_type"`
	NormalizedName string  `json:"normalized_name"`
	INN            string  `json:"inn,omitempty"`
	BIN            string  `json:"bin,omitempty"`
	KPP            string  `json:"kpp,omitempty"`
	Category       string  `json:"category,omitempty"`
	QualityScore   float64 `json:"quality_score"`
	DuplicateGroup int     `json:"duplicate_group,omitempty"`
	IsMaster       bool    `json:"is_master,omitempty"`
}

func main() {
	// Создаем директорию для тестовых данных
	dataDir := filepath.Join("tests", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	fmt.Println("Generating golden dataset...")

	// Генерируем тестовые данные
	entries := generateTestEntries()

	goldenDataset := GoldenDataset{
		Version:   "1.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Entries:   entries,
	}

	// Сохраняем golden dataset
	goldenPath := filepath.Join(dataDir, "golden_dataset.json")
	goldenData, err := json.MarshalIndent(goldenDataset, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal golden dataset: %v", err)
	}

	if err := os.WriteFile(goldenPath, goldenData, 0644); err != nil {
		log.Fatalf("Failed to write golden dataset: %v", err)
	}

	fmt.Printf("Generated golden dataset with %d entries in %s\n", len(entries), goldenPath)

	// Генерируем expected results
	expectedResult := generateExpectedResults(entries)

	expectedPath := filepath.Join(dataDir, "expected_result.json")
	expectedData, err := json.MarshalIndent(expectedResult, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal expected results: %v", err)
	}

	if err := os.WriteFile(expectedPath, expectedData, 0644); err != nil {
		log.Fatalf("Failed to write expected results: %v", err)
	}

	fmt.Printf("Generated expected results in %s\n", expectedPath)

	// Также создаем SQLite БД с golden dataset для тестирования
	fmt.Println("\nGenerating SQLite database with golden dataset...")
	generateSQLiteDB(dataDir, entries)
}

// generateTestEntries генерирует тестовые записи с различными типами проблем
func generateTestEntries() []GoldenEntry {
	entries := []GoldenEntry{}
	id := 1

	// === КОНТРАГЕНТЫ ===

	// Группа 1: Явные дубликаты "Ромашка"
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "ООО Ромашка",
		Attributes: `<ИНН>1234567890</ИНН><КПП>123456789</КПП><Адрес>г. Москва, ул. Тестовая, д. 1</Адрес>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ООО Ромашка",
			INN:            "1234567890",
			KPP:            "123456789",
			QualityScore:   0.95,
			DuplicateGroup: 1,
			IsMaster:       true,
			ExtractedAttrs: map[string]string{
				"legal_address": "г. Москва, ул. Тестовая, д. 1",
			},
		},
	})
	id++

	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "Ромашка ООО",
		Attributes: `<ИНН>1234567890</ИНН><КПП>123456789</КПП>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ООО Ромашка",
			INN:            "1234567890",
			KPP:            "123456789",
			QualityScore:   0.90,
			DuplicateGroup: 1,
			IsMaster:       false,
		},
	})
	id++

	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "ИП Ромашка",
		Attributes: `<ИНН>123456789012</ИНН>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ИП Ромашка",
			INN:            "123456789012",
			QualityScore:   0.85,
			DuplicateGroup: 0, // Разная организационно-правовая форма
			IsMaster:       true,
		},
	})
	id++

	// Группа 2: Дубликаты с ошибками
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "OOO Ромашка", // Ошибка: латинские буквы
		Attributes: `<ИНН>1234567890</ИНН>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ООО Ромашка",
			INN:            "1234567890",
			QualityScore:   0.80,
			DuplicateGroup: 1,
			IsMaster:       false,
		},
	})
	id++

	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "ИНН 1234567890", // Ошибка: ИНН в названии
		Attributes: `<ИНН>1234567890</ИНН>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ООО Ромашка", // Должно быть исправлено
			INN:            "1234567890",
			QualityScore:   0.75,
			DuplicateGroup: 1,
			IsMaster:       false,
		},
	})
	id++

	// Группа 3: Разные форматы
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "Ромашка", // Без ООО
		Attributes: `<ИНН>1234567890</ИНН>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "Ромашка",
			INN:            "1234567890",
			QualityScore:   0.70,
			DuplicateGroup: 0,
			IsMaster:       true,
		},
	})
	id++

	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "ООО РОМАШКА", // Верхний регистр
		Attributes: `<ИНН>1234567890</ИНН>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ООО Ромашка",
			INN:            "1234567890",
			QualityScore:   0.90,
			DuplicateGroup: 1,
			IsMaster:       false,
		},
	})
	id++

	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       " Ромашка ", // С пробелами
		Attributes: `<ИНН>1234567890</ИНН>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "Ромашка",
			INN:            "1234567890",
			QualityScore:   0.85,
			DuplicateGroup: 0,
			IsMaster:       true,
		},
	})
	id++

	// Группа 4: Грязные данные
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "ООО Ромашка",      // Неполный ИНН
		Attributes: `<ИНН>12345</ИНН>`, // Неполный ИНН
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ООО Ромашка",
			INN:            "12345",
			QualityScore:   0.60,
			DuplicateGroup: 0,
			IsMaster:       true,
		},
	})
	id++

	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "counterparty",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "code" + strconv.Itoa(id),
		Name:       "ООО Ромашка", // Опечатка
		Attributes: `<ИНН>9876543210</ИНН>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "ООО Ромашка",
			INN:            "9876543210",
			QualityScore:   0.90,
			DuplicateGroup: 2, // Новая группа (другой ИНН)
			IsMaster:       true,
		},
	})
	id++

	// === НОМЕНКЛАТУРА ===

	// Группа 1: Дубликаты товаров
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "nomenclature",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "АРТ: 123",
		Name:       "Винт Винтажный",
		Attributes: `<Артикул>123</Артикул><Категория>Крепеж</Категория>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "Винт Винтажный",
			Category:       "Крепеж",
			QualityScore:   0.95,
			DuplicateGroup: 3,
			IsMaster:       true,
			ExtractedAttrs: map[string]string{
				"артикул": "123",
			},
		},
	})
	id++

	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "nomenclature",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "АРТ: 321",
		Name:       "Винт винтажный", // Разный регистр
		Attributes: `<Артикул>321</Артикул><Категория>Крепеж</Категория>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "Винт Винтажный",
			Category:       "Крепеж",
			QualityScore:   0.90,
			DuplicateGroup: 3,
			IsMaster:       false,
			ExtractedAttrs: map[string]string{
				"артикул": "321",
			},
		},
	})
	id++

	// Группа 2: Товары без артикулов
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "nomenclature",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "",
		Name:       "Гайка М8",
		Attributes: `<Категория>Крепеж</Категория>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "Гайка М8",
			Category:       "Крепеж",
			QualityScore:   0.80,
			DuplicateGroup: 0,
			IsMaster:       true,
		},
	})
	id++

	// Группа 3: Некорректная категоризация
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "nomenclature",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "АРТ: 456",
		Name:       "Болт М10",
		Attributes: `<Артикул>456</Артикул><Категория>Инструмент</Категория>`, // Неправильная категория
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "Болт М10",
			Category:       "Крепеж", // Должна быть исправлена
			QualityScore:   0.75,
			DuplicateGroup: 0,
			IsMaster:       true,
			ExtractedAttrs: map[string]string{
				"артикул": "456",
			},
		},
	})
	id++

	// Группа 4: Опечатки в названиях
	entries = append(entries, GoldenEntry{
		ID:         id,
		EntityType: "nomenclature",
		Reference:  "ref" + strconv.Itoa(id),
		Code:       "АРТ: 789",
		Name:       "Шайба М8", // Опечатка
		Attributes: `<Артикул>789</Артикул><Категория>Крепеж</Категория>`,
		Expected: struct {
			NormalizedName string            `json:"normalized_name"`
			INN            string            `json:"inn,omitempty"`
			BIN            string            `json:"bin,omitempty"`
			KPP            string            `json:"kpp,omitempty"`
			Category       string            `json:"category,omitempty"`
			QualityScore   float64           `json:"quality_score"`
			DuplicateGroup int               `json:"duplicate_group,omitempty"`
			IsMaster       bool              `json:"is_master,omitempty"`
			ExtractedAttrs map[string]string `json:"extracted_attrs,omitempty"`
		}{
			NormalizedName: "Шайба М8",
			Category:       "Крепеж",
			QualityScore:   0.85,
			DuplicateGroup: 0,
			IsMaster:       true,
			ExtractedAttrs: map[string]string{
				"артикул": "789",
			},
		},
	})
	id++

	return entries
}

// generateExpectedResults генерирует ожидаемые результаты нормализации
func generateExpectedResults(entries []GoldenEntry) ExpectedResult {
	results := []ResultEntry{}

	counterparties := struct {
		TotalUnique     int `json:"total_unique"`
		TotalDuplicates int `json:"total_duplicates"`
		DuplicateGroups int `json:"duplicate_groups"`
	}{
		TotalUnique:     0,
		TotalDuplicates: 0,
		DuplicateGroups: 0,
	}

	nomenclature := struct {
		TotalUnique     int `json:"total_unique"`
		TotalDuplicates int `json:"total_duplicates"`
		DuplicateGroups int `json:"duplicate_groups"`
	}{
		TotalUnique:     0,
		TotalDuplicates: 0,
		DuplicateGroups: 0,
	}

	// Подсчитываем группы дубликатов
	duplicateGroups := make(map[int]int) // groupID -> count
	counterpartyGroups := make(map[int]bool)
	nomenclatureGroups := make(map[int]bool)

	for _, entry := range entries {
		result := ResultEntry{
			ID:             entry.ID,
			EntityType:     entry.EntityType,
			NormalizedName: entry.Expected.NormalizedName,
			INN:            entry.Expected.INN,
			BIN:            entry.Expected.BIN,
			KPP:            entry.Expected.KPP,
			Category:       entry.Expected.Category,
			QualityScore:   entry.Expected.QualityScore,
			DuplicateGroup: entry.Expected.DuplicateGroup,
			IsMaster:       entry.Expected.IsMaster,
		}
		results = append(results, result)

		if entry.Expected.DuplicateGroup > 0 {
			duplicateGroups[entry.Expected.DuplicateGroup]++
			if entry.EntityType == "counterparty" {
				counterpartyGroups[entry.Expected.DuplicateGroup] = true
			} else {
				nomenclatureGroups[entry.Expected.DuplicateGroup] = true
			}
		} else {
			if entry.EntityType == "counterparty" {
				counterparties.TotalUnique++
			} else {
				nomenclature.TotalUnique++
			}
		}
	}

	// Подсчитываем дубликаты
	for groupID, count := range duplicateGroups {
		if count > 1 {
			if counterpartyGroups[groupID] {
				counterparties.TotalDuplicates += count
				counterparties.DuplicateGroups++
			} else if nomenclatureGroups[groupID] {
				nomenclature.TotalDuplicates += count
				nomenclature.DuplicateGroups++
			}
		}
	}

	return ExpectedResult{
		Version:        "1.0",
		Timestamp:      time.Now().Format(time.RFC3339),
		Counterparties: counterparties,
		Nomenclature:   nomenclature,
		Results:        results,
	}
}

// generateSQLiteDB создает SQLite БД с golden dataset
func generateSQLiteDB(dataDir string, entries []GoldenEntry) {
	dbPath := filepath.Join(dataDir, "golden_dataset.db")

	// Удаляем существующую БД
	os.Remove(dbPath)

	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Создаем upload
	upload, err := db.CreateUpload("golden-dataset-uuid", "8.3", "golden-config")
	if err != nil {
		log.Fatalf("Failed to create upload: %v", err)
	}

	// Разделяем на контрагентов и номенклатуру
	counterparties := []GoldenEntry{}
	nomenclature := []GoldenEntry{}

	for _, entry := range entries {
		if entry.EntityType == "counterparty" {
			counterparties = append(counterparties, entry)
		} else {
			nomenclature = append(nomenclature, entry)
		}
	}

	// Создаем каталог для контрагентов
	if len(counterparties) > 0 {
		catalog, err := db.AddCatalog(upload.ID, "Контрагенты", "counterparties")
		if err != nil {
			log.Fatalf("Failed to create catalog: %v", err)
		}

		for _, entry := range counterparties {
			if err := db.AddCatalogItem(
				catalog.ID,
				entry.Reference,
				entry.Code,
				entry.Name,
				entry.Attributes,
				"",
			); err != nil {
				log.Fatalf("Failed to add catalog item %d: %v", entry.ID, err)
			}
		}
	}

	// Добавляем номенклатуру
	if len(nomenclature) > 0 {
		for _, entry := range nomenclature {
			// Используем nomenclature_items таблицу
			attrs := entry.Attributes
			if !strings.Contains(attrs, "<Артикул>") && entry.Code != "" {
				attrs = fmt.Sprintf(`<Артикул>%s</Артикул>%s`, entry.Code, attrs)
			}

			if err := db.AddNomenclatureItem(
				upload.ID,
				entry.Reference,
				entry.Code,
				entry.Name,
				"",
				"",
				attrs,
				"",
			); err != nil {
				log.Fatalf("Failed to add nomenclature item %d: %v", entry.ID, err)
			}
		}
	}

	fmt.Printf("Generated SQLite database with %d counterparties and %d nomenclature items in %s\n",
		len(counterparties), len(nomenclature), dbPath)
}
