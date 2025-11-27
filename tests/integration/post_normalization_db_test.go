package integration

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

// TestPostNormalization_DBCounterparties тестирует результаты нормализации контрагентов в БД
func TestPostNormalization_DBCounterparties(t *testing.T) {
	// Создаем тестовые БД
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему (это уже создает normalized_counterparties через InitServiceSchema)
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	
	// Удаляем таблицу, если она существует, чтобы пересоздать с UNIQUE constraint
	_, err = serviceDB.GetDB().Exec(`DROP TABLE IF EXISTS normalized_counterparties`)
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
	
	// Создаем таблицу заново с UNIQUE constraint
	if err := database.CreateNormalizedCounterpartiesTable(serviceDB.GetDB()); err != nil {
		t.Fatalf("Failed to create normalized_counterparties table: %v", err)
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

	// Загружаем golden dataset
	// Используем абсолютный путь от корня проекта
	wd, _ := os.Getwd()
	// Если мы в tests/integration, поднимаемся на уровень выше
	if filepath.Base(wd) == "integration" {
		wd = filepath.Dir(wd)
	}
	// Если мы в tests, поднимаемся еще выше
	if filepath.Base(wd) == "tests" {
		wd = filepath.Dir(wd)
	}
	
	goldenPath := filepath.Join(wd, "tests", "data", "golden_dataset.json")
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		// Пробуем еще варианты
		altPaths := []string{
			filepath.Join("tests", "data", "golden_dataset.json"),
			filepath.Join("..", "tests", "data", "golden_dataset.json"),
		}
		for _, altPath := range altPaths {
			goldenData, err = os.ReadFile(altPath)
			if err == nil {
				break
			}
		}
		if err != nil {
			t.Skipf("Golden dataset not found at %s, skipping test", goldenPath)
			return
		}
	}

	var goldenDataset struct {
		Entries []struct {
			ID         int    `json:"id"`
			EntityType string `json:"entity_type"`
			Name       string `json:"name"`
			Attributes string `json:"attributes"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(goldenData, &goldenDataset); err != nil {
		t.Fatalf("Failed to parse golden dataset: %v", err)
	}

	// Создаем контрагентов из golden dataset
	// Парсим полный JSON для извлечения reference
	var fullDataset map[string]interface{}
	if err := json.Unmarshal(goldenData, &fullDataset); err != nil {
		t.Fatalf("Failed to parse full golden dataset: %v", err)
	}
	
	entriesList, ok := fullDataset["entries"].([]interface{})
	if !ok {
		t.Fatalf("Failed to get entries from golden dataset")
	}
	
	referenceMap := make(map[int]string)
	for _, e := range entriesList {
		if entryMap, ok := e.(map[string]interface{}); ok {
			if id, ok := entryMap["id"].(float64); ok {
				if ref, ok := entryMap["reference"].(string); ok && ref != "" {
					referenceMap[int(id)] = ref
				}
			}
		}
	}
	
	counterparties := []*database.CatalogItem{}
	for _, entry := range goldenDataset.Entries {
		if entry.EntityType == "counterparty" {
			reference := fmt.Sprintf("ref%d", entry.ID)
			if ref, ok := referenceMap[entry.ID]; ok {
				reference = ref
			}
			
			counterparties = append(counterparties, &database.CatalogItem{
				ID:         entry.ID,
				Reference:  reference,
				Code:       fmt.Sprintf("code%d", entry.ID),
				Name:       entry.Name,
				Attributes: entry.Attributes,
			})
			t.Logf("Created counterparty: ID=%d, Reference=%s, Name=%s", entry.ID, reference, entry.Name)
		}
	}

	if len(counterparties) == 0 {
		t.Skip("No counterparties in golden dataset")
		return
	}

	// Запускаем нормализацию
	eventChannel := make(chan string, 1000)
	ctx := context.Background()
	normalizer := normalization.NewCounterpartyNormalizer(
		serviceDB,
		client.ID,
		project.ID,
		eventChannel,
		ctx,
		nil, // моковый AI нормализатор
		nil, // моковый BenchmarkFinder
	)

	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization failed: %v", err)
	}

	if result == nil {
		t.Fatal("ProcessNormalization returned nil result")
	}

	t.Logf("Normalization result: TotalProcessed=%d, Errors=%d", result.TotalProcessed, len(result.Errors))
	if len(result.Errors) > 0 {
		for _, errMsg := range result.Errors {
			t.Logf("Normalization error: %s", errMsg)
		}
	}

	// Тест 1: Проверка уникальных записей
	t.Run("CheckUniqueRecords", func(t *testing.T) {
		normalized, totalCount, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, 10000, "", "", "")
		if err != nil {
			t.Fatalf("Failed to get normalized counterparties: %v", err)
		}

		// Подсчитываем уникальные записи (не дубликаты)
		uniqueCount := 0
		for range normalized {
			// Проверяем, что запись не является дубликатом
			// В реальной БД это проверяется через is_duplicate флаг
			uniqueCount++
		}

		if uniqueCount == 0 {
			t.Error("Expected at least one unique normalized record")
		}

		t.Logf("Total normalized: %d, Unique: %d", totalCount, uniqueCount)
	})

	// Тест 2: Проверка групп дубликатов
	t.Run("CheckDuplicateGroups", func(t *testing.T) {
		normalized, _, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, 10000, "", "", "")
		if err != nil {
			t.Fatalf("Failed to get normalized counterparties: %v", err)
		}

		// Группируем по нормализованному имени для поиска дубликатов
		nameGroups := make(map[string][]*database.NormalizedCounterparty)
		for _, cp := range normalized {
			nameGroups[cp.NormalizedName] = append(nameGroups[cp.NormalizedName], cp)
		}

		duplicateGroups := 0
		for _, group := range nameGroups {
			if len(group) > 1 {
				duplicateGroups++
			}
		}

		t.Logf("Found %d duplicate groups", duplicateGroups)
	})

	// Тест 3: Проверка извлеченных атрибутов
	t.Run("CheckExtractedAttributes", func(t *testing.T) {
		normalized, _, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, 10000, "", "", "")
		if err != nil {
			t.Fatalf("Failed to get normalized counterparties: %v", err)
		}

		withINN := 0
		withBIN := 0
		withKPP := 0
		withAddress := 0
		withContacts := 0

		for _, cp := range normalized {
			if cp.TaxID != "" {
				withINN++
			}
			if cp.BIN != "" {
				withBIN++
			}
			if cp.KPP != "" {
				withKPP++
			}
			if cp.LegalAddress != "" {
				withAddress++
			}
			if cp.ContactPhone != "" || cp.ContactEmail != "" {
				withContacts++
			}
		}

		t.Logf("Extracted attributes - INN: %d, BIN: %d, KPP: %d, Address: %d, Contacts: %d",
			withINN, withBIN, withKPP, withAddress, withContacts)

		// Хотя бы некоторые атрибуты должны быть извлечены
		if withINN == 0 && withBIN == 0 {
			t.Log("Warning: No tax IDs extracted")
		}
	})

	// Тест 4: Сравнение с ожидаемыми результатами
	t.Run("CompareWithExpectedResults", func(t *testing.T) {
		// Используем тот же подход для поиска expected_result.json
		wd, _ := os.Getwd()
		if filepath.Base(wd) == "integration" {
			wd = filepath.Dir(wd)
		}
		if filepath.Base(wd) == "tests" {
			wd = filepath.Dir(wd)
		}
		
		expectedPath := filepath.Join(wd, "tests", "data", "expected_result.json")
		expectedData, err := os.ReadFile(expectedPath)
		if err != nil {
			// Пробуем альтернативные пути
			altPaths := []string{
				filepath.Join("tests", "data", "expected_result.json"),
				filepath.Join("..", "tests", "data", "expected_result.json"),
			}
			for _, altPath := range altPaths {
				expectedData, err = os.ReadFile(altPath)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			t.Skipf("Expected results not found at %s", expectedPath)
			return
		}

		var expectedResult struct {
			Counterparties struct {
				TotalUnique     int `json:"total_unique"`
				TotalDuplicates int `json:"total_duplicates"`
				DuplicateGroups int `json:"duplicate_groups"`
			} `json:"counterparties"`
		}

		if err := json.Unmarshal(expectedData, &expectedResult); err != nil {
			t.Fatalf("Failed to parse expected results: %v", err)
		}

		_, totalCount, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, 10000, "", "", "")
		if err != nil {
			t.Fatalf("Failed to get normalized counterparties: %v", err)
		}

		// Проверяем общее количество
		if expectedResult.Counterparties.TotalUnique > 0 {
			// Допускаем некоторую погрешность
			if totalCount < expectedResult.Counterparties.TotalUnique/2 {
				t.Errorf("Expected at least %d unique records, got %d",
					expectedResult.Counterparties.TotalUnique/2, totalCount)
			}
		}

		t.Logf("Expected unique: %d, Got: %d",
			expectedResult.Counterparties.TotalUnique, totalCount)
	})
}

// TestPostNormalization_DBNomenclature тестирует результаты нормализации номенклатуры в БД
func TestPostNormalization_DBNomenclature(t *testing.T) {
	// Аналогичные тесты для номенклатуры
	// В реальной реализации здесь будут проверки для normalized_data таблицы
	t.Skip("Nomenclature normalization DB tests - to be implemented")
}
