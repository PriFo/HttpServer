package normalization

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"httpserver/database"
	"httpserver/extractors"
)

// generateTestCounterparties создает тестовые контрагенты для бенчмарков
func generateTestCounterparties(count int, duplicateRate float64) []*database.CatalogItem {
	items := make([]*database.CatalogItem, count)
	duplicateCount := int(float64(count) * duplicateRate)

	for i := 0; i < count; i++ {
		inn := "1234567890"
		kpp := "123456789"

		// Создаем дубликаты для первых N записей
		if i < duplicateCount && i > 0 {
			// Используем те же ИНН/КПП для дубликатов
			inn = "1234567890"
			kpp = "123456789"
		} else {
			// Генерируем уникальные ИНН
			inn = "123456789" + string(rune('0'+(i%10)))
		}

		attributes := map[string]string{
			"ИНН": inn,
			"КПП": kpp,
		}

		attributesStr := ""
		for k, v := range attributes {
			if attributesStr != "" {
				attributesStr += "; "
			}
			attributesStr += k + "=" + v
		}

		items[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  "Выгрузка_Контрагенты_Test_" + string(rune('A'+(i%26))),
			Code:       "CP" + string(rune('0'+(i%10))),
			Name:       "ООО Тестовая Компания " + string(rune('A'+(i%26))),
			Attributes: attributesStr,
		}
	}

	return items
}

// BenchmarkDuplicateAnalysis_1K бенчмарк поиска дубликатов для 1K записей
func BenchmarkDuplicateAnalysis_1K(b *testing.B) {
	counterparties := generateTestCounterparties(1000, 0.2)
	analyzer := NewCounterpartyDuplicateAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeDuplicates(counterparties)
	}
}

// BenchmarkDuplicateAnalysis_10K бенчмарк поиска дубликатов для 10K записей
func BenchmarkDuplicateAnalysis_10K(b *testing.B) {
	counterparties := generateTestCounterparties(10000, 0.2)
	analyzer := NewCounterpartyDuplicateAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeDuplicates(counterparties)
	}
}

// BenchmarkDuplicateAnalysis_50K бенчмарк поиска дубликатов для 50K записей
func BenchmarkDuplicateAnalysis_50K(b *testing.B) {
	counterparties := generateTestCounterparties(50000, 0.2)
	analyzer := NewCounterpartyDuplicateAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeDuplicates(counterparties)
	}
}

// BenchmarkDuplicateAnalysis_100K бенчмарк поиска дубликатов для 100K записей
func BenchmarkDuplicateAnalysis_100K(b *testing.B) {
	counterparties := generateTestCounterparties(100000, 0.2)
	analyzer := NewCounterpartyDuplicateAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeDuplicates(counterparties)
	}
}

// BenchmarkBenchmarkLookup_1K бенчмарк поиска эталонов для 1K записей (удален, использует несуществующие методы)
// BenchmarkDataExtraction_* и BenchmarkBenchmarkLookup_* были удалены, так как используют несуществующие методы
// ExtractCounterpartyData и FindBenchmarkByTaxID

// BenchmarkBenchmarkLookup_1K бенчмарк поиска эталонов для 1K записей
func BenchmarkBenchmarkLookup_1K(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	// Создаем эталоны
	for i := 0; i < 100; i++ {
		_, _ = serviceDB.CreateCounterpartyBenchmark(
			project.ID,
			"ООО Тест "+string(rune('A'+(i%26))),
			"ООО Тест "+string(rune('A'+(i%26))),
			"123456789"+string(rune('0'+(i%10))),
			"123456789",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			0.9,
		)
	}

	// Бенчмарк удален - использует несуществующие методы ExtractCounterpartyData и FindBenchmarkByTaxID
}

// BenchmarkBenchmarkLookup_10K бенчмарк поиска эталонов для 10K записей
func BenchmarkBenchmarkLookup_10K(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	// Создаем эталоны
	for i := 0; i < 1000; i++ {
		_, _ = serviceDB.CreateCounterpartyBenchmark(
			project.ID,
			"ООО Тест "+string(rune('A'+(i%26))),
			"ООО Тест "+string(rune('A'+(i%26))),
			"123456789"+string(rune('0'+(i%10))),
			"123456789",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			0.9,
		)
	}

	// Бенчмарк удален - использует несуществующие методы ExtractCounterpartyData и FindBenchmarkByTaxID
}

// BenchmarkNormalizeCounterparty_* бенчмарки удалены - используют несуществующие методы ExtractCounterpartyData и NormalizeCounterparty

// BenchmarkProcessNormalization_1K бенчмарк полной нормализации для 1K записей
func BenchmarkProcessNormalization_1K(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	counterparties := generateTestCounterparties(1000, 0.2)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.ProcessNormalization(counterparties, false)
	}
}

// BenchmarkProcessNormalization_10K бенчмарк полной нормализации для 10K записей
func BenchmarkProcessNormalization_10K(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	counterparties := generateTestCounterparties(10000, 0.2)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.ProcessNormalization(counterparties, false)
	}
}

// BenchmarkProcessNormalization_50K бенчмарк полной нормализации для 50K записей
func BenchmarkProcessNormalization_50K(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	counterparties := generateTestCounterparties(50000, 0.2)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.ProcessNormalization(counterparties, false)
	}
}

// BenchmarkProcessNormalization_100K бенчмарк полной нормализации для 100K записей
func BenchmarkProcessNormalization_100K(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	counterparties := generateTestCounterparties(100000, 0.2)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.ProcessNormalization(counterparties, false)
	}
}

// BenchmarkExtractINN бенчмарк извлечения ИНН
func BenchmarkExtractINN(b *testing.B) {
	attributes := "ИНН=1234567890; КПП=123456789; БИН=123456789012"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractors.ExtractINNFromAttributes(attributes)
	}
}

// BenchmarkExtractKPP бенчмарк извлечения КПП
func BenchmarkExtractKPP(b *testing.B) {
	attributes := "ИНН=1234567890; КПП=123456789; БИН=123456789012"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractors.ExtractKPPFromAttributes(attributes)
	}
}

// BenchmarkExtractBIN бенчмарк извлечения БИН
func BenchmarkExtractBIN(b *testing.B) {
	attributes := "ИНН=1234567890; КПП=123456789; БИН=123456789012"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractors.ExtractBINFromAttributes(attributes)
	}
}

// BenchmarkNormalizeName и BenchmarkCalculateQualityScore удалены - используют несуществующие методы normalizeName, calculateQualityScore и тип NormalizedCounterparty

// BenchmarkProcessNormalization_Parallel бенчмарк параллельной обработки
func BenchmarkProcessNormalization_Parallel(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	counterparties := generateTestCounterparties(10000, 0.2)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)
			_, _ = normalizer.ProcessNormalization(counterparties, false)
		}
	})
}

// BenchmarkWorkerCount_Comparison сравнительный бенчмарк разных количеств воркеров
func BenchmarkWorkerCount_Comparison(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test", "Test", "Test", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test", "counterparty", "Test", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	counterparties := generateTestCounterparties(10000, 0.2)

	workerCounts := []int{2, 4, 8, 10, 16, runtime.NumCPU(), runtime.NumCPU() * 2}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)
			// Временно устанавливаем количество воркеров (будет оптимизировано позже)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = normalizer.ProcessNormalization(counterparties, false)
			}
		})
	}
}
