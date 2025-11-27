package performance

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/normalization"
)

// TestMemoryUsage_Normalization тестирует потребление памяти во время нормализации
func TestMemoryUsage_Normalization(t *testing.T) {
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

	// Создаем 50K записей для тестирования
	recordCount := 50000
	counterparties := make([]*database.CatalogItem, recordCount)
	for i := 0; i < recordCount; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  fmt.Sprintf("ref%d", i+1),
			Code:       fmt.Sprintf("code%d", i+1),
			Name:       fmt.Sprintf("ООО Тест %d", i+1),
			Attributes: fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10),
		}
	}

	// Измеряем память до нормализации
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Запускаем нормализацию
	eventChannel := make(chan string, 100000)
	ctx := context.Background()
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, ctx, nil, nil)

	start := time.Now()
	result, err := normalizer.ProcessNormalization(counterparties, false)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ProcessNormalization failed: %v", err)
	}

	// Измеряем память после нормализации
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Вычисляем использованную память
	memUsed := memAfter.Alloc - memBefore.Alloc
	memUsedMB := float64(memUsed) / 1024 / 1024

	t.Logf("Memory usage: %.2f MB", memUsedMB)
	t.Logf("Processing time: %v", duration)
	t.Logf("Records processed: %d", result.TotalProcessed)
	t.Logf("Records per second: %.2f", float64(result.TotalProcessed)/duration.Seconds())

	// Проверяем, что память не превышает разумные пределы (< 2GB для 50K записей)
	maxMemoryMB := 2048.0 // 2GB
	if memUsedMB > maxMemoryMB {
		t.Errorf("Memory usage %.2f MB exceeds limit of %.2f MB", memUsedMB, maxMemoryMB)
	}

	// Проверяем на утечки памяти - память должна быть освобождена после GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var memAfterGC runtime.MemStats
	runtime.ReadMemStats(&memAfterGC)

	memAfterGCMB := float64(memAfterGC.Alloc) / 1024 / 1024
	t.Logf("Memory after GC: %.2f MB", memAfterGCMB)

	// Память должна уменьшиться после GC
	if memAfterGCMB > memUsedMB*1.5 {
		t.Logf("Warning: Memory may not be fully released after GC (%.2f MB vs %.2f MB)", memAfterGCMB, memUsedMB)
	}
}

// TestMemoryUsage_NoLeaks проверяет отсутствие утечек памяти при множественных запусках
func TestMemoryUsage_NoLeaks(t *testing.T) {
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

	// Создаем тестовые данные
	recordCount := 1000
	counterparties := make([]*database.CatalogItem, recordCount)
	for i := 0; i < recordCount; i++ {
		counterparties[i] = &database.CatalogItem{
			ID:         i + 1,
			Reference:  fmt.Sprintf("ref%d", i+1),
			Code:       fmt.Sprintf("code%d", i+1),
			Name:       fmt.Sprintf("ООО Тест %d", i+1),
			Attributes: fmt.Sprintf(`<ИНН>123456789%d</ИНН>`, i%10),
		}
	}

	// Измеряем память до циклов
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Запускаем нормализацию несколько раз
	iterations := 10
	for i := 0; i < iterations; i++ {
		eventChannel := make(chan string, 1000)
		ctx := context.Background()
		normalizer := normalization.NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, eventChannel, ctx, nil, nil)

		_, err := normalizer.ProcessNormalization(counterparties, false)
		if err != nil {
			t.Fatalf("ProcessNormalization failed on iteration %d: %v", i+1, err)
		}

		// Принудительный GC после каждой итерации
		runtime.GC()
	}

	// Измеряем память после циклов
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Вычисляем изменение памяти (может быть отрицательным, если память освободилась)
	memDiff := int64(memAfter.Alloc) - int64(memBefore.Alloc)
	memUsedMB := float64(memDiff) / 1024 / 1024
	t.Logf("Memory increase after %d iterations: %.2f MB", iterations, memUsedMB)
	t.Logf("Memory before: %.2f MB, Memory after: %.2f MB", 
		float64(memBefore.Alloc)/1024/1024, float64(memAfter.Alloc)/1024/1024)

	// Память не должна значительно увеличиваться (допускаем до 100MB для 10 итераций)
	// Если память уменьшилась, это нормально
	maxIncreaseMB := 100.0
	if memUsedMB > maxIncreaseMB {
		t.Errorf("Memory leak detected: memory increased by %.2f MB (limit: %.2f MB)", memUsedMB, maxIncreaseMB)
	} else if memUsedMB < 0 {
		t.Logf("Memory decreased by %.2f MB (good sign - no leaks)", -memUsedMB)
	}
}

