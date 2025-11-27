package normalization

import (
	"context"
	"testing"
	"time"

	"httpserver/database"
)

// TestCounterpartyNormalizer_ProcessNormalization_StopCheck_InterruptsCorrectly проверяет корректную остановку после N вызовов
func TestCounterpartyNormalizer_ProcessNormalization_StopCheck_InterruptsCorrectly(t *testing.T) {
	serviceDB := createTestServiceDBForStop(t)
	defer serviceDB.Close()

	// Инициализируем схему БД
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Отменяем контекст через небольшую задержку, чтобы симулировать остановку
	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()

	eventChannel := make(chan string, 1000)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, ctx, nil, nil)

	// Создаем достаточно контрагентов для длительной обработки
	counterparties := make([]*database.CatalogItem, 0, 300)
	for i := 0; i < 300; i++ {
		counterparties = append(counterparties, createTestCounterpartyForStop(i+1,
			"ООО Тест "+string(rune('A'+i%26)),
			`<ИНН>123456789`+string(rune('0'+i%10))+`</ИНН>`))
	}

	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization returned error: %v", err)
	}

	// Проверяем, что ProcessedCount меньше общего числа записей (нормализация была прервана)
	if result.TotalProcessed >= len(counterparties) {
		t.Errorf("Expected ProcessedCount (%d) to be less than total (%d) due to stop",
			result.TotalProcessed, len(counterparties))
	}

	// Проверяем, что была остановка
	found := false
	for _, errMsg := range result.Errors {
		if errMsg == ErrMsgNormalizationStopped {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error message 'Нормализация остановлена пользователем' not found in errors")
	}

	// Проверяем, что обработка была остановлена (контекст был проверен внутри ProcessNormalization)
	// Проверка контекста происходит внутри ProcessNormalization
	// Важно, что обработка была остановлена

	// Проверяем, что было обработано некоторое количество записей (не 0, но и не все)
	if result.TotalProcessed == 0 {
		t.Error("Expected some records to be processed before stop, got 0")
	}
}

// createTestCounterparty создает тестового контрагента
func createTestCounterpartyForStop(id int, name, attributes string) *database.CatalogItem {
	return &database.CatalogItem{
		ID:         id,
		Reference:  "ref" + string(rune('0'+id)),
		Code:       "code" + string(rune('0'+id)),
		Name:       name,
		Attributes: attributes,
	}
}

// createTestServiceDB создает тестовую БД для сервиса
func createTestServiceDBForStop(t *testing.T) *database.ServiceDB {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	return serviceDB
}
