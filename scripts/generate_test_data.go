package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/brianvoe/gofakeit/v6"
	"httpserver/database"
)

// TestDataEntry запись тестовых данных
type TestDataEntry struct {
	ID         int    `json:"id"`
	Reference  string `json:"reference"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Attributes string `json:"attributes"`
}

// TestDataset набор тестовых данных
type TestDataset struct {
	Count   int             `json:"count"`
	Entries []TestDataEntry `json:"entries"`
}

func main() {
	// Инициализируем gofakeit
	gofakeit.Seed(0)

	// Размеры наборов данных
	sizes := []struct {
		name string
		size int
	}{
		{"1K", 1000},
		{"10K", 10000},
		{"50K", 50000},
	}

	// Создаем директорию для тестовых данных
	dataDir := filepath.Join("tests", "data_integrity")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	for _, size := range sizes {
		fmt.Printf("Generating %s records...\n", size.name)

		entries := make([]TestDataEntry, size.size)
		for i := 0; i < size.size; i++ {
			// Генерируем случайные данные
			companyName := generateCompanyName()
			inn := generateINN()
			bin := ""

			// Иногда добавляем БИН (для казахстанских компаний)
			if gofakeit.Bool() && i%10 == 0 {
				bin = generateBIN()
			}

			attributes := fmt.Sprintf("<ИНН>%s</ИНН>", inn)
			if bin != "" {
				attributes += fmt.Sprintf("<БИН>%s</БИН>", bin)
			}

			// Добавляем дополнительные атрибуты случайно
			if gofakeit.Bool() {
				attributes += fmt.Sprintf("<КПП>%s</КПП>", generateKPP())
			}
			if gofakeit.Bool() {
				attributes += fmt.Sprintf("<Адрес>%s</Адрес>", gofakeit.Address().Address)
			}

			entries[i] = TestDataEntry{
				ID:         i + 1,
				Reference:  "ref" + strconv.Itoa(i+1),
				Code:       "code" + strconv.Itoa(i+1),
				Name:       companyName,
				Attributes: attributes,
			}
		}

		dataset := TestDataset{
			Count:   size.size,
			Entries: entries,
		}

		// Сохраняем в JSON
		filename := filepath.Join(dataDir, fmt.Sprintf("test_data_%s.json", size.name))
		data, err := json.MarshalIndent(dataset, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal dataset: %v", err)
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			log.Fatalf("Failed to write file %s: %v", filename, err)
		}

		fmt.Printf("Generated %s records in %s\n", size.name, filename)
	}

	// Также создаем SQLite БД с тестовыми данными
	fmt.Println("\nGenerating SQLite database...")
	generateSQLiteDB(dataDir)
}

// generateCompanyName генерирует название компании
func generateCompanyName() string {
	legalForms := []string{"ООО", "ОАО", "ЗАО", "ИП", "ТОО", "АО"}
	legalForm := gofakeit.RandomString(legalForms)

	companyTypes := []string{"Ромашка", "Тест", "Алма", "Солнце", "Звезда", "Вектор", "Глобус", "Мир", "Триумф", "Лидер"}
	companyType := gofakeit.RandomString(companyTypes)

	// Иногда добавляем номер
	if gofakeit.Bool() {
		return fmt.Sprintf("%s %s %d", legalForm, companyType, gofakeit.Number(1, 100))
	}

	return fmt.Sprintf("%s %s", legalForm, companyType)
}

// generateINN генерирует российский ИНН (10 или 12 цифр)
func generateINN() string {
	if gofakeit.Bool() {
		// 10-значный ИНН
		return gofakeit.Numerify("##########")
	}
	// 12-значный ИНН
	return gofakeit.Numerify("############")
}

// generateBIN генерирует казахстанский БИН (12 цифр)
func generateBIN() string {
	return gofakeit.Numerify("############")
}

// generateKPP генерирует КПП (9 цифр)
func generateKPP() string {
	return gofakeit.Numerify("#########")
}

// generateSQLiteDB создает SQLite БД с тестовыми данными
func generateSQLiteDB(dataDir string) {
	dbPath := filepath.Join(dataDir, "test_data.db")

	// Удаляем существующую БД
	os.Remove(dbPath)

	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Создаем upload
	upload, err := db.CreateUpload("test-uuid", "8.3", "test-config")
	if err != nil {
		log.Fatalf("Failed to create upload: %v", err)
	}

	// Создаем каталог
	catalog, err := db.AddCatalog(upload.ID, "TestCatalog", "test_catalog")
	if err != nil {
		log.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем 1000 записей для быстрого тестирования
	for i := 0; i < 1000; i++ {
		companyName := generateCompanyName()
		inn := generateINN()
		attributes := fmt.Sprintf("<ИНН>%s</ИНН>", inn)

		if err := db.AddCatalogItem(catalog.ID,
			"ref"+strconv.Itoa(i+1),
			"code"+strconv.Itoa(i+1),
			companyName,
			attributes,
			""); err != nil {
			log.Fatalf("Failed to add catalog item %d: %v", i+1, err)
		}
	}

	fmt.Printf("Generated SQLite database with 1000 records in %s\n", dbPath)
}
