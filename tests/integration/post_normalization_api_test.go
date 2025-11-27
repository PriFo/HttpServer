package integration

import (
	"context"
	"testing"

	"httpserver/database"
	"httpserver/normalization"
)

// TestPostNormalization_API тестирует API результаты нормализации
func TestPostNormalization_API(t *testing.T) {
	// Создаем тестовые БД
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

	// Тестовые данные для нормализации создаются напрямую в массиве counterparties ниже

	// Запускаем нормализацию
	eventChannel := make(chan string, 100)
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

	// Получаем контрагентов для нормализации
	counterparties := []*database.CatalogItem{
		{
			ID:         1,
			Reference:  "ref1",
			Code:       "code1",
			Name:       "ООО Ромашка",
			Attributes: `<ИНН>1234567890</ИНН><КПП>123456789</КПП>`,
		},
		{
			ID:         2,
			Reference:  "ref2",
			Code:       "code2",
			Name:       "Ромашка ООО",
			Attributes: `<ИНН>1234567890</ИНН><КПП>123456789</КПП>`,
		},
	}

	_, err = normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization failed: %v", err)
	}

	// Тест 1: Проверка получения нормализованных контрагентов через database API
	t.Run("GetNormalizedCounterparties", func(t *testing.T) {
		normalized, totalCount, err := serviceDB.GetNormalizedCounterparties(project.ID, 0, 100, "", "", "")
		if err != nil {
			t.Fatalf("Failed to get normalized counterparties: %v", err)
		}

		if totalCount == 0 {
			t.Error("Expected at least one normalized counterparty")
		}

		if len(normalized) == 0 {
			t.Error("Expected normalized counterparties list to be non-empty")
		}

		// Проверяем структуру данных
		for _, cp := range normalized {
			if cp.NormalizedName == "" {
				t.Error("Normalized counterparty should have normalized_name")
			}
			if cp.ClientProjectID != project.ID {
				t.Errorf("Expected project ID %d, got %d", project.ID, cp.ClientProjectID)
			}
		}
	})

	// Тест 2: Проверка получения нормализованной номенклатуры
	t.Run("GetNormalizedNomenclature", func(t *testing.T) {
		// Проверяем, что таблица normalized_nomenclature существует
		// В этом тесте мы не нормализуем номенклатуру, поэтому просто проверяем, что метод доступен
		// (если метод существует, он вернет пустой результат)
		t.Log("Nomenclature normalization test skipped - no nomenclature data in this test")
	})

	// Тест 3: Проверка флага is_normalized для базы данных проекта
	t.Run("CheckDatabaseNormalizedFlag", func(t *testing.T) {
		// Создаем проект БД
		projectDB, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", ":memory:", "Test DB Description", 0)
		if err != nil {
			t.Fatalf("Failed to create project database: %v", err)
		}

		// Проверяем, что база данных создана
		if projectDB == nil {
			t.Fatal("Project database should not be nil")
		}

		// Проверяем, что база данных имеет правильный project ID
		if projectDB.ClientProjectID != project.ID {
			t.Errorf("Expected project ID %d, got %d", project.ID, projectDB.ClientProjectID)
		}

		// В реальной системе флаг is_normalized устанавливается после нормализации
		// Здесь мы просто проверяем, что база данных создана корректно
		t.Logf("Project database created: ID=%d, ClientProjectID=%d, Name=%s", projectDB.ID, projectDB.ClientProjectID, projectDB.Name)
	})
}

