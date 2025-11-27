package performance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"httpserver/database"
	ai "httpserver/internal/infrastructure/ai"
	"httpserver/normalization"
	"httpserver/server"
)

// setupBenchmarkDB создает тестовую БД с указанным количеством записей
func setupBenchmarkDB(recordCount int) (*database.ServiceDB, int, int, error) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		return nil, 0, 0, err
	}

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		serviceDB.Close()
		return nil, 0, 0, err
	}

	// Создаем тестового клиента и проект
	client, err := serviceDB.CreateClient("Benchmark Client", "Benchmark Legal", "Desc", "benchmark@test.com", "+123", "TAX", "user")
	if err != nil {
		serviceDB.Close()
		return nil, 0, 0, err
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Benchmark Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		serviceDB.Close()
		return nil, 0, 0, err
	}

	return serviceDB, client.ID, project.ID, nil
}

// BenchmarkNormalization_1KRecords бенчмарк для 1K записей
func BenchmarkNormalization_1KRecords(b *testing.B) {
	serviceDB, clientID, projectID, err := setupBenchmarkDB(1000)
	if err != nil {
		b.Fatalf("Failed to setup benchmark DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем тестовые данные
	counterparties := make([]*database.CatalogItem, 1000)
	for i := 0; i < 1000; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  "ref" + string(rune('0'+i%10)),
			Code:       "code" + string(rune('0'+i%10)),
			Name:       "ООО Тест " + string(rune('A'+i%26)),
			Attributes: `<ИНН>123456789` + string(rune('0'+i%10)) + `</ИНН>`,
		}
	}

	eventChannel := make(chan string, 10000)
	ctx := context.Background()
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, clientID, projectID, eventChannel, ctx, nil, nil)

	b.ResetTimer()                    // Начинаем замер здесь
	b.ReportMetric(1000.0, "records") // Сообщаем, сколько записей обработано

	for i := 0; i < b.N; i++ {
		_, err := normalizer.ProcessNormalization(counterparties, false)
		if err != nil {
			b.Errorf("ProcessNormalization failed: %v", err)
		}
	}
}

// BenchmarkNormalization_10KRecords бенчмарк для 10K записей
func BenchmarkNormalization_10KRecords(b *testing.B) {
	serviceDB, clientID, projectID, err := setupBenchmarkDB(10000)
	if err != nil {
		b.Fatalf("Failed to setup benchmark DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем тестовые данные
	counterparties := make([]*database.CatalogItem, 10000)
	for i := 0; i < 10000; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  "ref" + string(rune('0'+i%10)),
			Code:       "code" + string(rune('0'+i%10)),
			Name:       "ООО Тест " + string(rune('A'+i%26)),
			Attributes: `<ИНН>123456789` + string(rune('0'+i%10)) + `</ИНН>`,
		}
	}

	eventChannel := make(chan string, 100000)
	ctx := context.Background()
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, clientID, projectID, eventChannel, ctx, nil, nil)

	b.ResetTimer()
	b.ReportMetric(10000.0, "records")

	// Запускаем только один раз для больших объемов
	if b.N > 1 {
		b.N = 1
	}

	start := time.Now()
	_, err = normalizer.ProcessNormalization(counterparties, false)
	duration := time.Since(start)

	if err != nil {
		b.Errorf("ProcessNormalization failed: %v", err)
	}

	b.ReportMetric(float64(10000)/duration.Seconds(), "records/sec")
}

// BenchmarkNormalization_50KRecords бенчмарк для 50K записей
func BenchmarkNormalization_50KRecords(b *testing.B) {
	serviceDB, clientID, projectID, err := setupBenchmarkDB(50000)
	if err != nil {
		b.Fatalf("Failed to setup benchmark DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем тестовые данные
	counterparties := make([]*database.CatalogItem, 50000)
	for i := 0; i < 50000; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  "ref" + string(rune('0'+i%10)),
			Code:       "code" + string(rune('0'+i%10)),
			Name:       "ООО Тест " + string(rune('A'+i%26)),
			Attributes: `<ИНН>123456789` + string(rune('0'+i%10)) + `</ИНН>`,
		}
	}

	eventChannel := make(chan string, 500000)
	ctx := context.Background()
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, clientID, projectID, eventChannel, ctx, nil, nil)

	b.ResetTimer()
	b.ReportMetric(50000.0, "records")

	// Запускаем только один раз для больших объемов
	if b.N > 1 {
		b.N = 1
	}

	start := time.Now()
	_, err = normalizer.ProcessNormalization(counterparties, false)
	duration := time.Since(start)

	if err != nil {
		b.Errorf("ProcessNormalization failed: %v", err)
	}

	b.ReportMetric(float64(50000)/duration.Seconds(), "records/sec")
}

// BenchmarkNormalization_100KRecords бенчмарк для 100K записей
func BenchmarkNormalization_100KRecords(b *testing.B) {
	serviceDB, clientID, projectID, err := setupBenchmarkDB(100000)
	if err != nil {
		b.Fatalf("Failed to setup benchmark DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем тестовые данные
	counterparties := make([]*database.CatalogItem, 100000)
	for i := 0; i < 100000; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  fmt.Sprintf("ref%d", i+1),
			Code:       fmt.Sprintf("code%d", i+1),
			Name:       fmt.Sprintf("ООО Тест %d", i+1),
			Attributes: fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10),
		}
	}

	eventChannel := make(chan string, 1000000)
	ctx := context.Background()
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, clientID, projectID, eventChannel, ctx, nil, nil)

	b.ResetTimer()
	b.ReportMetric(100000.0, "records")

	// Запускаем только один раз для больших объемов
	if b.N > 1 {
		b.N = 1
	}

	start := time.Now()
	_, err = normalizer.ProcessNormalization(counterparties, false)
	duration := time.Since(start)

	if err != nil {
		b.Errorf("ProcessNormalization failed: %v", err)
	}

	b.ReportMetric(float64(100000)/duration.Seconds(), "records/sec")
}

// BenchmarkNormalization_Parallel бенчмарк параллельной обработки
func BenchmarkNormalization_Parallel(b *testing.B) {
	serviceDB, clientID, projectID, err := setupBenchmarkDB(10000)
	if err != nil {
		b.Fatalf("Failed to setup benchmark DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем тестовые данные
	counterparties := make([]*database.CatalogItem, 10000)
	for i := 0; i < 10000; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  fmt.Sprintf("ref%d", i+1),
			Code:       fmt.Sprintf("code%d", i+1),
			Name:       fmt.Sprintf("ООО Тест %d", i+1),
			Attributes: fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10),
		}
	}

	b.ResetTimer()
	b.ReportMetric(10000.0, "records")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			eventChannel := make(chan string, 10000)
			ctx := context.Background()
			normalizer := normalization.NewCounterpartyNormalizer(serviceDB, clientID, projectID, eventChannel, ctx, nil, nil)

			// Обрабатываем часть данных параллельно
			chunkSize := len(counterparties) / 4
			chunk := counterparties[:chunkSize]

			_, err := normalizer.ProcessNormalization(chunk, false)
			if err != nil {
				b.Errorf("ProcessNormalization failed: %v", err)
			}
		}
	})
}

// BenchmarkMultiProviderClient_NormalizeName бенчмарк для мульти-провайдерного клиента
func BenchmarkMultiProviderClient_NormalizeName(b *testing.B) {
	providers := []*database.Provider{
		{
			ID:       1,
			Name:     "Provider 1",
			Type:     "provider1",
			IsActive: true,
			Config:   `{"channels":2}`,
		},
		{
			ID:       2,
			Name:     "Provider 2",
			Type:     "provider2",
			IsActive: true,
			Config:   `{"channels":1}`,
		},
	}

	// Создаем моки провайдеров
	clients := map[string]ai.ProviderClient{
		"provider1": &mockBenchmarkProvider{name: "Provider 1", enabled: true, response: "ООО Тест"},
		"provider2": &mockBenchmarkProvider{name: "Provider 2", enabled: true, response: "ООО Тест"},
	}

	mpc := server.NewMultiProviderClient(providers, clients, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mpc.NormalizeName(ctx, "ооо тест")
		if err != nil {
			b.Errorf("NormalizeName failed: %v", err)
		}
	}
}

// mockBenchmarkProvider мок провайдера для бенчмарков
type mockBenchmarkProvider struct {
	name     string
	enabled  bool
	response string
}

func (m *mockBenchmarkProvider) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	return m.response, nil
}

func (m *mockBenchmarkProvider) GetProviderName() string {
	return m.name
}

func (m *mockBenchmarkProvider) IsEnabled() bool {
	return m.enabled
}
