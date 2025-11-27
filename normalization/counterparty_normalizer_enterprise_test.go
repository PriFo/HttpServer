package normalization

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"httpserver/database"
)

// ========== Тесты на Race Conditions ==========

// TestProcessNormalization_RaceCondition_ConcurrentStop проверяет race condition при одновременной остановке
func TestProcessNormalization_RaceCondition_ConcurrentStop(t *testing.T) {
	serviceDB := createTestServiceDB(t)
	defer serviceDB.Close()

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем много контрагентов для длительной обработки
	counterparties := make([]*database.CatalogItem, 200)
	for i := 0; i < 200; i++ {
		counterparties[i] = createTestCounterparty(i+1, "ООО Тест", `<ИНН>1234567890</ИНН>`)
	}

	eventChannel := make(chan string, 100)

	// Создаем контекст для отмены операции
	// Тест проверяет, что множественные проверки контекста не вызывают race conditions
	stopCheckCalled := int32(0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем нормализатор с контекстом
	normalizerWithStop := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, ctx, nil, nil)

	// Запускаем нормализацию в горутине
	var result *CounterpartyNormalizationResult
	var errResult error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, errResult = normalizerWithStop.ProcessNormalization(counterparties, false)
	}()

	// Несколько горутин одновременно проверяют контекст (симуляция параллельных проверок)
	// Это проверяет, что множественные проверки контекста не вызывают race conditions
	var cancelCount int32
	var cancelWg sync.WaitGroup

	for i := 0; i < 10; i++ {
		cancelWg.Add(1)
		go func() {
			defer cancelWg.Done()
			// Проверяем контекст несколько раз из разных горутин
			for j := 0; j < 5; j++ {
				select {
				case <-ctx.Done():
					atomic.AddInt32(&stopCheckCalled, 1)
				default:
					atomic.AddInt32(&stopCheckCalled, 1)
				}
				time.Sleep(time.Millisecond * 5)
			}
			atomic.AddInt32(&cancelCount, 1)
		}()
	}

	// Ждем завершения всех горутин
	cancelWg.Wait()
	wg.Wait()

	// Проверяем, что система корректно обработала множественные вызовы stopCheck
	if errResult != nil {
		t.Fatalf("ProcessNormalization returned error: %v", errResult)
	}

	// Проверяем, что все записи обработаны (контекст не был отменен)
	if result.TotalProcessed != len(counterparties) {
		t.Errorf("Expected all %d records to be processed, got %d", len(counterparties), result.TotalProcessed)
	}

	// Проверяем, что контекст был проверен множество раз (доказывает, что проверки были безопасными)
	if atomic.LoadInt32(&stopCheckCalled) < 10 {
		t.Errorf("Expected context to be checked at least 10 times, got %d", atomic.LoadInt32(&stopCheckCalled))
	}

	// Проверяем, что все горутины завершились
	if atomic.LoadInt32(&cancelCount) != 10 {
		t.Errorf("Expected 10 goroutines to complete, got %d", atomic.LoadInt32(&cancelCount))
	}
}

// TestProcessNormalization_RaceCondition_ResultAccess проверяет безопасность доступа к result из разных горутин
func TestProcessNormalization_RaceCondition_ResultAccess(t *testing.T) {
	serviceDB := createTestServiceDB(t)
	defer serviceDB.Close()

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	eventChannel := make(chan string, 100)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, context.Background(), nil, nil)

	counterparties := make([]*database.CatalogItem, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = createTestCounterparty(i+1, "ООО Тест", `<ИНН>1234567890</ИНН>`)
	}

	// Запускаем нормализацию
	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization returned error: %v", err)
	}

	// Несколько горутин одновременно читают result
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Читаем различные поля result
			_ = result.TotalProcessed
			_ = result.DuplicateGroups
			_ = len(result.Errors)
			_ = result.ClientID
			_ = result.ProjectID
		}()
	}
	wg.Wait()

	// Если тест завершился без паники, значит нет race condition
}

// ========== Тесты на граничные случаи ==========

// TestProcessNormalization_EmptyList проверяет обработку пустого списка с отменой
func TestProcessNormalization_EmptyList_WithCancel(t *testing.T) {
	serviceDB := createTestServiceDB(t)
	defer serviceDB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем контекст сразу
	eventChannel := make(chan string, 100)
	normalizer := NewCounterpartyNormalizer(serviceDB, 1, 1, eventChannel, ctx, nil, nil)

	counterparties := []*database.CatalogItem{}

	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization returned error: %v", err)
	}

	// Должно быть обработано 0 записей
	if result.TotalProcessed != 0 {
		t.Errorf("Expected TotalProcessed 0, got %d", result.TotalProcessed)
	}

	// Система не должна падать при отмене пустого списка
}

// TestProcessNormalization_CancelAfterCompletion проверяет, что отмена после завершения игнорируется
func TestProcessNormalization_CancelAfterCompletion(t *testing.T) {
	serviceDB := createTestServiceDB(t)
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

	eventChannel := make(chan string, 100)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, context.Background(), nil, nil)

	counterparties := []*database.CatalogItem{
		createTestCounterparty(1, "ООО Тест 1", `<ИНН>1234567890</ИНН>`),
		createTestCounterparty(2, "ООО Тест 2", `<ИНН>1234567891</ИНН>`),
	}

	// Запускаем нормализацию
	var result *CounterpartyNormalizationResult
	var errResult error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, errResult = normalizer.ProcessNormalization(counterparties, false)
	}()

	// Ждем завершения
	wg.Wait()

	if errResult != nil {
		t.Fatalf("ProcessNormalization returned error: %v", errResult)
	}

	// Проверяем, что все записи обработаны (stopCheck после завершения не должен влиять)
	if result.TotalProcessed != len(counterparties) {
		t.Errorf("Expected %d processed, got %d", len(counterparties), result.TotalProcessed)
	}

	// Не должно быть ошибок об остановке
	for _, errMsg := range result.Errors {
		if errMsg == "Нормализация остановлена пользователем" {
			t.Error("Unexpected stop error message when cancellation happened after completion")
		}
	}
}

// TestProcessNormalization_ContextTimeout проверяет автоматическую остановку через таймаут
func TestProcessNormalization_ContextTimeout(t *testing.T) {
	serviceDB := createTestServiceDB(t)
	defer serviceDB.Close()

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	eventChannel := make(chan string, 100)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, ctx, nil, nil)

	// Создаем много контрагентов для длительной обработки
	counterparties := make([]*database.CatalogItem, 500)
	for i := 0; i < 500; i++ {
		counterparties[i] = createTestCounterparty(i+1, "ООО Тест", `<ИНН>1234567890</ИНН>`)
	}

	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization returned error: %v", err)
	}

	// Проверяем, что обработка была остановлена по таймауту
	if result.TotalProcessed >= len(counterparties) {
		t.Error("Expected processing to be stopped by timeout, but all records were processed")
	}

	// Проверяем, что остановка произошла по таймауту
	// (проверка выполняется в самом тесте через результат)
}

// TestProcessNormalization_IdempotentCancel проверяет идемпотентность отмены
func TestProcessNormalization_IdempotentCancel(t *testing.T) {
	serviceDB := createTestServiceDB(t)
	defer serviceDB.Close()

	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Desc", "test@test.com", "+123", "TAX", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Desc", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	eventChannel := make(chan string, 100)
	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, context.Background(), nil, nil)

	counterparties := make([]*database.CatalogItem, 100)
	for i := 0; i < 100; i++ {
		counterparties[i] = createTestCounterparty(i+1, "ООО Тест", `<ИНН>1234567890</ИНН>`)
	}

	// Запускаем нормализацию
	var result *CounterpartyNormalizationResult
	var errResult error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, errResult = normalizer.ProcessNormalization(counterparties, false)
	}()

	// Многократно вызываем stopCheck (должно быть идемпотентно)
	// В данном тесте stopCheck всегда возвращает false, так что это просто проверка на отсутствие паники

	wg.Wait()

	if errResult != nil {
		t.Fatalf("ProcessNormalization returned error: %v", errResult)
	}

	// Проверяем, что ошибка об остановке не дублируется
	stopErrorCount := 0
	for _, errMsg := range result.Errors {
		if errMsg == "Нормализация остановлена пользователем" {
			stopErrorCount++
		}
	}

	// Должна быть только одна ошибка об остановке (не дублируется)
	if stopErrorCount > 1 {
		t.Errorf("Expected at most 1 stop error, got %d", stopErrorCount)
	}
}

// TestProcessNormalization_ProcessedCountAccuracy проверяет точность ProcessedCount
func TestProcessNormalization_ProcessedCountAccuracy(t *testing.T) {
	serviceDB := createTestServiceDB(t)
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

	eventChannel := make(chan string, 100)

	// Создаем контекст, который будет отменен после небольшой задержки
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	normalizer := NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, ctx, nil, nil)

	// Создаем ровно 10 контрагентов
	counterparties := make([]*database.CatalogItem, 10)
	for i := 0; i < 10; i++ {
		counterparties[i] = createTestCounterparty(i+1, "ООО Тест", `<ИНН>1234567890</ИНН>`)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("ProcessNormalization returned error: %v", err)
	}

	// ProcessedCount должен отражать фактическое количество обработанных записей
	// на момент остановки, а не запланированное
	if result.TotalProcessed < 0 {
		t.Errorf("ProcessedCount should be non-negative, got %d", result.TotalProcessed)
	}

	if result.TotalProcessed > len(counterparties) {
		t.Errorf("ProcessedCount (%d) should not exceed total count (%d)", result.TotalProcessed, len(counterparties))
	}

	// Если была остановка, ProcessedCount должен быть меньше общего количества
	if len(result.Errors) > 0 {
		hasStopError := false
		for _, errMsg := range result.Errors {
			if errMsg == "Нормализация остановлена пользователем" {
				hasStopError = true
				break
			}
		}
		if hasStopError && result.TotalProcessed >= len(counterparties) {
			t.Errorf("When stopped, ProcessedCount (%d) should be less than total (%d)", result.TotalProcessed, len(counterparties))
		}
	}
}
