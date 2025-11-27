package normalization

import (
	"context"
	"testing"

	"httpserver/database"
)

// BenchmarkProcessNormalization_Small бенчмарк обработки 10 контрагентов
func BenchmarkProcessNormalization_Small(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)

	counterparties := make([]*database.CatalogItem, 10)
	for i := 0; i < 10; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  "ref_test_" + string(rune('0'+i)),
			Code:       "code_test_" + string(rune('0'+i)),
			Name:       "ООО Тест " + string(rune('0'+i)),
			Attributes: `<ИНН>123456789` + string(rune('0'+i)) + `</ИНН>`,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.ProcessNormalization(counterparties, false)
	}
}

// BenchmarkProcessNormalization_Medium бенчмарк обработки 100 контрагентов
func BenchmarkProcessNormalization_Medium(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)

	counterparties := make([]*database.CatalogItem, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  "ref_test_" + string(rune('0'+(i%10))),
			Code:       "code_test_" + string(rune('0'+(i%10))),
			Name:       "ООО Тест " + string(rune('0'+(i%10))),
			Attributes: `<ИНН>123456789` + string(rune('0'+(i%10))) + `</ИНН>`,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.ProcessNormalization(counterparties, false)
	}
}

// BenchmarkProcessNormalization_Large бенчмарк обработки 1000 контрагентов
func BenchmarkProcessNormalization_Large(b *testing.B) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		b.Fatalf("Failed to create ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		b.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		b.Fatalf("Failed to create project: %v", err)
	}

	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, context.Background(), nil, nil)

	counterparties := make([]*database.CatalogItem, 1000)
	for i := 0; i < 1000; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  "ref_test_" + string(rune('0'+(i%10))),
			Code:       "code_test_" + string(rune('0'+(i%10))),
			Name:       "ООО Тест " + string(rune('0'+(i%10))),
			Attributes: `<ИНН>123456789` + string(rune('0'+(i%10))) + `</ИНН>`,
		}
	}

	b.ResetTimer()
	b.StopTimer()  // Останавливаем таймер для подготовки данных
	b.StartTimer() // Запускаем таймер для бенчмарка
	_, _ = normalizer.ProcessNormalization(counterparties, false)
}
