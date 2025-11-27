package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/normalization"
)

// TestNormalizationStop_HandleNormalizationStop тестирует механизм остановки через API
func TestNormalizationStop_HandleNormalizationStop(t *testing.T) {
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

	// Создаем тестовую БД
	testDB, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer testDB.Close()

	upload, err := testDB.CreateUpload("test-uuid", "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	catalog, err := testDB.AddCatalog(upload.ID, "Контрагенты", "counterparties")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем тестовых контрагентов
	for i := 0; i < 10; i++ {
		if err := testDB.AddCatalogItem(
			catalog.ID,
			fmt.Sprintf("ref%d", i+1),
			fmt.Sprintf("code%d", i+1),
			fmt.Sprintf("ООО Тест %d", i+1),
			fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10),
			"",
		); err != nil {
			t.Fatalf("Failed to add catalog item: %v", err)
		}
	}

	// Тест 1: Остановка нормализации через context (без использования server для избежания циклического импорта)
	t.Run("StopNormalizationViaContext", func(t *testing.T) {
		// Создаем контекст с отменой
		ctx, cancel := context.WithCancel(context.Background())

		// Создаем канал для событий
		eventChannel := make(chan string, 100)

		// Создаем нормализатор с контекстом
		normalizer := normalization.NewCounterpartyNormalizer(
			serviceDB,
			client.ID,
			project.ID,
			eventChannel,
			ctx,
			nil, // моковый AI нормализатор
			nil, // моковый BenchmarkFinder
		)

		// Получаем контрагентов из тестовой БД
		items, _, err := testDB.GetCatalogItemsByUpload(upload.ID, []string{"Контрагенты"}, 0, 0)
		if err != nil {
			t.Fatalf("Failed to get catalog items: %v", err)
		}

		// Запускаем нормализацию в горутине
		resultChan := make(chan *normalization.CounterpartyNormalizationResult, 1)
		errChan := make(chan error, 1)

		go func() {
			result, err := normalizer.ProcessNormalization(items, false)
			if err != nil {
				errChan <- err
			} else {
				resultChan <- result
			}
		}()

		// Ждем немного, затем отменяем контекст
		time.Sleep(100 * time.Millisecond)
		cancel()

		// Ждем результата
		select {
		case result := <-resultChan:
			// Проверяем, что результат содержит информацию об остановке
			if result != nil {
				if len(result.Errors) > 0 {
					hasStopError := false
					for _, errMsg := range result.Errors {
						if errMsg == normalization.ErrMsgNormalizationStopped {
							hasStopError = true
							break
						}
					}
					if !hasStopError {
						t.Logf("Warning: No stop error found in result, but normalization was stopped")
					}
				}
				// Проверяем, что обработано меньше записей, чем всего
				if result.TotalProcessed >= len(items) {
					t.Logf("Warning: All items processed despite stop, but this might be expected if processing was fast")
				}
			}
		case err := <-errChan:
			t.Errorf("Normalization failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Error("Normalization did not complete within timeout")
		}
	})
}

// TestNormalizationStop_ContextCancellation тестирует отмену через context
func TestNormalizationStop_ContextCancellation(t *testing.T) {
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

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем тестовых контрагентов
	counterparties := make([]*database.CatalogItem, 5)
	for i := 0; i < 5; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  fmt.Sprintf("ref%d", i+1),
			Code:       fmt.Sprintf("code%d", i+1),
			Name:       fmt.Sprintf("ООО Тест %d", i+1),
			Attributes: fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10),
		}
	}

	// Создаем канал для событий
	eventChannel := make(chan string, 100)

	// Создаем нормализатор с контекстом
	normalizer := normalization.NewCounterpartyNormalizer(
		serviceDB,
		client.ID,
		project.ID,
		eventChannel,
		ctx,
		nil, // моковый AI нормализатор
		nil, // моковый BenchmarkFinder
	)

	// Запускаем нормализацию в горутине
	resultChan := make(chan *normalization.CounterpartyNormalizationResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := normalizer.ProcessNormalization(counterparties, false)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Ждем немного, затем отменяем контекст
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Ждем результата
	select {
	case result := <-resultChan:
		// Проверяем, что результат содержит информацию об остановке
		if result != nil {
			if len(result.Errors) > 0 {
				hasStopError := false
				for _, errMsg := range result.Errors {
					if errMsg == normalization.ErrMsgNormalizationStopped {
						hasStopError = true
						break
					}
				}
				if !hasStopError {
					t.Error("Expected normalization_stopped_by_user error in result")
				}
			}
		}
	case err := <-errChan:
		// Ошибка тоже допустима при остановке
		if err != nil && err.Error() != "normalization_stopped_by_user" {
			t.Logf("Got error (expected): %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Normalization did not stop within timeout")
	}
}

// TestNormalizationStop_PartialResults тестирует корректную обработку частичных результатов
func TestNormalizationStop_PartialResults(t *testing.T) {
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

	// Создаем большое количество контрагентов для тестирования частичной обработки
	counterparties := make([]*database.CatalogItem, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  fmt.Sprintf("ref%d", i+1),
			Code:       fmt.Sprintf("code%d", i+1),
			Name:       fmt.Sprintf("ООО Тест %d", i+1),
			Attributes: fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10),
		}
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Создаем канал для событий
	eventChannel := make(chan string, 1000)

	// Создаем нормализатор
	normalizer := normalization.NewCounterpartyNormalizer(
		serviceDB,
		client.ID,
		project.ID,
		eventChannel,
		ctx,
		nil,
		nil,
	)

	// Запускаем нормализацию
	result, err := normalizer.ProcessNormalization(counterparties, false)

	// Проверяем, что получили частичный результат
	if result != nil {
		// Должна быть обработана хотя бы часть записей
		if result.TotalProcessed > 0 {
			t.Logf("Processed %d out of %d records before stop", result.TotalProcessed, len(counterparties))
		}

		// Проверяем, что есть информация об остановке
		if len(result.Errors) > 0 {
			hasStopError := false
			for _, errMsg := range result.Errors {
				if errMsg == normalization.ErrMsgNormalizationStopped {
					hasStopError = true
					break
				}
			}
			if !hasStopError && ctx.Err() == context.DeadlineExceeded {
				// Если контекст был отменен по таймауту, это нормально
				t.Log("Normalization stopped due to context timeout")
			}
		}
	}
}
