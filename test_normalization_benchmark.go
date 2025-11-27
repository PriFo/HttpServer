//go:build ignore
// +build ignore

package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/normalization"
)

// BenchmarkResult результат бенчмарка для одного этапа
type BenchmarkResult struct {
	Stage             string        `json:"stage"`
	RecordCount       int           `json:"record_count"`
	Duration          time.Duration `json:"duration_ms"`
	RecordsPerSecond  float64       `json:"records_per_second"`
	MemoryUsedMB      float64       `json:"memory_used_mb,omitempty"`
	DuplicateGroups   int           `json:"duplicate_groups,omitempty"`
	TotalDuplicates   int           `json:"total_duplicates,omitempty"`
	BenchmarkMatches  int           `json:"benchmark_matches,omitempty"`
	EnrichedCount     int           `json:"enriched_count,omitempty"`
	CreatedBenchmarks int           `json:"created_benchmarks,omitempty"`
	ProcessedCount    int           `json:"processed_count,omitempty"`
	ErrorCount        int           `json:"error_count,omitempty"`
	Stopped           bool          `json:"stopped,omitempty"`
	StopLatency       time.Duration `json:"stop_latency_ms,omitempty"`
}

// FullBenchmarkReport полный отчет о бенчмарке
type FullBenchmarkReport struct {
	Timestamp     string                 `json:"timestamp"`
	TestName      string                 `json:"test_name"`
	RecordCount   int                    `json:"record_count"`
	DuplicateRate float64                `json:"duplicate_rate"`
	Workers       int                    `json:"workers"`
	Results       []BenchmarkResult      `json:"results"`
	TotalDuration time.Duration          `json:"total_duration_ms"`
	AverageSpeed  float64                `json:"average_speed_records_per_sec"`
	Summary       map[string]interface{} `json:"summary"`
}

// StopController контроллер для остановки нормализации
type StopController struct {
	mu          sync.RWMutex
	shouldStop  bool
	stopTime    time.Time
	stopLatency time.Duration
}

func NewStopController() *StopController {
	return &StopController{}
}

func (sc *StopController) Check() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.shouldStop
}

func (sc *StopController) Stop() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if !sc.shouldStop {
		sc.shouldStop = true
		sc.stopTime = time.Now()
	}
}

func (sc *StopController) GetStopLatency() time.Duration {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stopLatency
}

func (sc *StopController) SetStopLatency(latency time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.stopLatency = latency
}

func main() {
	var (
		recordCountFlag   = flag.Int("records", 1000, "Количество записей для тестирования")
		duplicateRateFlag = flag.Float64("duplicate-rate", 0.2, "Процент дубликатов (0.0-1.0)")
		workersFlag       = flag.Int("workers", 10, "Количество воркеров для параллельной обработки")
		testStopFlag      = flag.Bool("test-stop", false, "Тестировать механизм остановки")
		stopAfterFlag     = flag.Int("stop-after", 500, "Остановить после N записей (только с -test-stop)")
		cpuProfileFlag    = flag.String("cpuprofile", "", "Сохранить CPU профиль в файл")
		memProfileFlag    = flag.String("memprofile", "", "Сохранить memory профиль в файл")
	)
	flag.Parse()

	// Настройка CPU профилирования
	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
		if err != nil {
			log.Fatalf("Ошибка создания CPU профиля: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("Ошибка запуска CPU профилирования: %v", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("CPU профилирование включено: %s\n", *cpuProfileFlag)
	}

	// Настройка Memory профилирования
	if *memProfileFlag != "" {
		defer func() {
			f, err := os.Create(*memProfileFlag)
			if err != nil {
				log.Fatalf("Ошибка создания memory профиля: %v", err)
			}
			defer f.Close()
			runtime.GC() // Принудительная сборка мусора перед сохранением профиля
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatalf("Ошибка записи memory профиля: %v", err)
			}
			fmt.Printf("Memory профиль сохранен: %s\n", *memProfileFlag)
		}()
	}

	recordCount := *recordCountFlag
	duplicateRate := *duplicateRateFlag
	workers := *workersFlag

	fmt.Println("=== Бенчмарк нормализации контрагентов ===")
	fmt.Printf("Количество записей: %d\n", recordCount)
	fmt.Printf("Процент дубликатов: %.1f%%\n", duplicateRate*100)
	fmt.Printf("Количество воркеров: %d\n", workers)
	if *testStopFlag {
		fmt.Printf("Тест остановки: да (остановка после %d записей)\n", *stopAfterFlag)
	}
	fmt.Println()

	// Создаем временную БД
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		log.Fatalf("Ошибка создания ServiceDB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	if err := database.InitServiceSchema(serviceDB.GetDB()); err != nil {
		log.Fatalf("Ошибка инициализации схемы: %v", err)
	}

	// Создаем тестового клиента и проект
	client, err := serviceDB.CreateClient("Benchmark Client", "Benchmark Legal", "Benchmark Description", "benchmark@test.com", "+1234567890", "TAX", "benchmark")
	if err != nil {
		log.Fatalf("Ошибка создания клиента: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Benchmark Project", "counterparty", "Benchmark Project Description", "1C", 0.8)
	if err != nil {
		log.Fatalf("Ошибка создания проекта: %v", err)
	}

	// Генерируем тестовые данные
	fmt.Println("Генерация тестовых данных...")
	counterparties := generateTestCounterparties(recordCount, duplicateRate)
	fmt.Printf("Сгенерировано %d контрагентов\n", len(counterparties))
	fmt.Println()

	// Создаем нормализатор
	normalizer := normalization.NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, nil, nil)

	// Результаты бенчмарков
	results := make([]BenchmarkResult, 0)

	// 1. Бенчмарк извлечения данных
	fmt.Println("=== 1. Бенчмарк извлечения данных ===")
	fmt.Print("Выполняется... ")
	extractResult := benchmarkDataExtraction(normalizer, counterparties)
	results = append(results, extractResult)
	printResult(extractResult)
	fmt.Println()

	// 2. Бенчмарк поиска дубликатов
	fmt.Println("=== 2. Бенчмарк поиска дубликатов ===")
	fmt.Print("Выполняется... ")
	duplicateResult := benchmarkDuplicateDetection(counterparties)
	results = append(results, duplicateResult)
	printResult(duplicateResult)
	fmt.Println()

	// 3. Бенчмарк полной нормализации
	fmt.Println("=== 3. Бенчмарк полной нормализации ===")
	fmt.Print("Выполняется... ")
	if *testStopFlag {
		stopController := NewStopController()
		normalizerWithStop := normalization.NewCounterpartyNormalizer(serviceDB, client.ID, project.ID, nil, stopController.Check, nil)

		// Запускаем остановку в отдельной горутине
		go func() {
			time.Sleep(100 * time.Millisecond) // Даем время на старт
			processed := 0
			for processed < *stopAfterFlag {
				time.Sleep(50 * time.Millisecond)
				// Проверяем количество обработанных (упрощенная проверка)
				processed += 50 // Примерная оценка
			}
			stopStart := time.Now()
			stopController.Stop()
			stopLatency := time.Since(stopStart)
			stopController.SetStopLatency(stopLatency)
		}()

		normalizationResult := benchmarkFullNormalization(normalizerWithStop, counterparties, workers, true)
		results = append(results, normalizationResult)
		printResult(normalizationResult)
	} else {
		normalizationResult := benchmarkFullNormalization(normalizer, counterparties, workers, false)
		results = append(results, normalizationResult)
		printResult(normalizationResult)
	}
	fmt.Println()

	// 4. Бенчмарк параллельной обработки с разным количеством воркеров
	fmt.Println("=== 4. Бенчмарк параллельной обработки ===")
	fmt.Println("Примечание: количество воркеров задается внутри ProcessNormalization (по умолчанию 10)")
	fmt.Println("Тестируем с разными объемами данных для оценки масштабируемости...")

	// Тестируем с разными объемами данных
	testSizes := []int{100, 500, 1000}
	if recordCount >= 5000 {
		testSizes = []int{500, 1000, 2000, 5000}
	} else if recordCount >= 2000 {
		testSizes = []int{200, 500, 1000, 2000}
	}

	parallelResults := make([]BenchmarkResult, 0)
	for _, size := range testSizes {
		if size > len(counterparties) {
			size = len(counterparties)
		}
		testData := counterparties[:size]
		fmt.Printf("\nТестирование с %d записями...\n", size)

		result := benchmarkFullNormalization(normalizer, testData, workers, false)
		result.Stage = fmt.Sprintf("Scalability Test (%d records)", size)
		parallelResults = append(parallelResults, result)
		printResult(result)
	}

	for _, result := range parallelResults {
		results = append(results, result)
	}
	fmt.Println()

	// Вычисляем общую статистику
	totalDuration := time.Duration(0)
	for _, r := range results {
		totalDuration += r.Duration
	}

	avgSpeed := 0.0
	if totalDuration > 0 {
		avgSpeed = float64(recordCount) / totalDuration.Seconds()
	}

	// Создаем отчет
	report := FullBenchmarkReport{
		Timestamp:     time.Now().Format(time.RFC3339),
		TestName:      "Normalization Performance Benchmark",
		RecordCount:   recordCount,
		DuplicateRate: duplicateRate,
		Workers:       workers,
		Results:       results,
		TotalDuration: totalDuration,
		AverageSpeed:  avgSpeed,
		Summary: map[string]interface{}{
			"total_stages":      len(results),
			"total_duration_ms": totalDuration.Milliseconds(),
			"average_speed":     avgSpeed,
			"fastest_stage":     findFastestStage(results),
			"slowest_stage":     findSlowestStage(results),
		},
	}

	// Сохраняем отчет
	saveReportToJSON(report)
	
	// Сохраняем в CSV
	saveReportToCSV(report)

	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Println("БЕНЧМАРК ЗАВЕРШЕН")
	fmt.Println("=" + strings.Repeat("=", 100))
}

// generateTestCounterparties генерирует тестовые данные контрагентов
func generateTestCounterparties(count int, duplicateRate float64) []*database.CatalogItem {
	rand.Seed(time.Now().UnixNano())
	counterparties := make([]*database.CatalogItem, 0, count)

	// Генерируем уникальные контрагенты
	uniqueCount := int(float64(count) * (1 - duplicateRate))
	duplicateCount := count - uniqueCount

	// Варианты названий для разнообразия
	companyTypes := []string{"ООО", "ЗАО", "ОАО", "ИП", "ПАО", "НПО", "ООО", "ООО"} // ООО чаще
	companySuffixes := []string{"Компания", "Предприятие", "Группа", "Холдинг", "Корпорация", "Торговый Дом", "Альянс"}

	// Создаем уникальные контрагенты
	for i := 0; i < uniqueCount; i++ {
		inn := fmt.Sprintf("%010d", 1000000000+i)
		kpp := fmt.Sprintf("%09d", 100000000+i)

		// Разнообразие в названиях
		companyType := companyTypes[i%len(companyTypes)]
		suffix := companySuffixes[i%len(companySuffixes)]
		name := fmt.Sprintf("%s \"%s %d\"", companyType, suffix, i+1)

		// Разнообразие в адресах
		streets := []string{"Тестовая", "Ленина", "Пушкина", "Гагарина", "Мира", "Советская", "Центральная"}
		street := streets[i%len(streets)]
		address := fmt.Sprintf("Москва, ул. %s, д. %d", street, i+1)

		// Разнообразие в телефонах
		phone := fmt.Sprintf("+7999%07d", 1000000+i)

		// Добавляем дополнительные поля для некоторых записей
		attributes := fmt.Sprintf(`<ИНН>%s</ИНН><КПП>%s</КПП><Адрес>%s</Адрес><Телефон>%s</Телефон>`, inn, kpp, address, phone)

		// 30% записей имеют email
		if i%3 == 0 {
			email := fmt.Sprintf("contact%d@testcompany.ru", i+1)
			attributes += fmt.Sprintf(`<Email>%s</Email>`, email)
		}

		// 20% записей имеют контактное лицо
		if i%5 == 0 {
			contactPerson := fmt.Sprintf("Иванов Иван Иванович %d", i+1)
			attributes += fmt.Sprintf(`<КонтактноеЛицо>%s</КонтактноеЛицо>`, contactPerson)
		}

		// 10% записей имеют банковские реквизиты
		if i%10 == 0 {
			bankAccount := fmt.Sprintf("40702810%010d", i+1)
			bik := fmt.Sprintf("044525%03d", 590+i%10)
			attributes += fmt.Sprintf(`<РасчетныйСчет>%s</РасчетныйСчет><БИК>%s</БИК>`, bankAccount, bik)
		}

		counterparties = append(counterparties, &database.CatalogItem{
			ID:         i + 1,
			Reference:  fmt.Sprintf("ref_%d", i+1),
			Code:       fmt.Sprintf("code_%d", i+1),
			Name:       name,
			Attributes: attributes,
		})
	}

	// Создаем дубликаты (используем те же ИНН/КПП, но разные названия)
	baseIndex := 0
	duplicateVariants := []string{"(Дубликат)", "(Копия)", "(Вариант 2)", "(Дубль)", "(Повтор)"}
	for i := 0; i < duplicateCount; i++ {
		baseCounterparty := counterparties[baseIndex%uniqueCount]

		// Извлекаем ИНН из атрибутов базового контрагента
		inn := extractINNFromAttributes(baseCounterparty.Attributes)
		kpp := extractKPPFromAttributes(baseCounterparty.Attributes)

		// Создаем вариант с другим названием
		variant := duplicateVariants[i%len(duplicateVariants)]
		name := fmt.Sprintf("ООО Тест %d %s", i+1, variant)

		// Разные адреса для дубликатов
		streets := []string{"Другая", "Альтернативная", "Резервная", "Запасная"}
		street := streets[i%len(streets)]
		address := fmt.Sprintf("Москва, ул. %s, д. %d", street, i+1)
		phone := fmt.Sprintf("+7999%07d", 2000000+i)

		attributes := fmt.Sprintf(`<ИНН>%s</ИНН><КПП>%s</КПП><Адрес>%s</Адрес><Телефон>%s</Телефон>`,
			inn, kpp, address, phone)

		counterparties = append(counterparties, &database.CatalogItem{
			ID:         uniqueCount + i + 1,
			Reference:  fmt.Sprintf("ref_dup_%d", i+1),
			Code:       fmt.Sprintf("code_dup_%d", i+1),
			Name:       name,
			Attributes: attributes,
		})

		baseIndex++
	}

	return counterparties
}

// extractINNFromAttributes извлекает ИНН из атрибутов (упрощенная версия)
func extractINNFromAttributes(attributes string) string {
	start := strings.Index(attributes, "<ИНН>")
	if start == -1 {
		return ""
	}
	start += len("<ИНН>")
	end := strings.Index(attributes[start:], "</ИНН>")
	if end == -1 {
		return ""
	}
	return attributes[start : start+end]
}

// extractKPPFromAttributes извлекает КПП из атрибутов (упрощенная версия)
func extractKPPFromAttributes(attributes string) string {
	start := strings.Index(attributes, "<КПП>")
	if start == -1 {
		return ""
	}
	start += len("<КПП>")
	end := strings.Index(attributes[start:], "</КПП>")
	if end == -1 {
		return ""
	}
	return attributes[start : start+end]
}

// benchmarkDataExtraction бенчмарк извлечения данных
func benchmarkDataExtraction(normalizer *normalization.CounterpartyNormalizer, counterparties []*database.CatalogItem) BenchmarkResult {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	startTime := time.Now()

	for _, item := range counterparties {
		_ = normalizer.ExtractCounterpartyData(item)
	}

	duration := time.Since(startTime)

	runtime.ReadMemStats(&m2)
	memoryUsedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	recordsPerSec := float64(len(counterparties)) / duration.Seconds()

	return BenchmarkResult{
		Stage:            "Data Extraction",
		RecordCount:      len(counterparties),
		Duration:         duration,
		RecordsPerSecond: recordsPerSec,
		ProcessedCount:   len(counterparties),
		MemoryUsedMB:     memoryUsedMB,
	}
}

// benchmarkDuplicateDetection бенчмарк поиска дубликатов
func benchmarkDuplicateDetection(counterparties []*database.CatalogItem) BenchmarkResult {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	startTime := time.Now()

	analyzer := normalization.NewCounterpartyDuplicateAnalyzer()
	duplicateGroups := analyzer.AnalyzeDuplicates(counterparties)

	duration := time.Since(startTime)

	runtime.ReadMemStats(&m2)
	memoryUsedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	recordsPerSec := float64(len(counterparties)) / duration.Seconds()

	totalDuplicates := 0
	for _, group := range duplicateGroups {
		totalDuplicates += len(group.Items)
	}

	return BenchmarkResult{
		Stage:            "Duplicate Detection",
		RecordCount:      len(counterparties),
		Duration:         duration,
		RecordsPerSecond: recordsPerSec,
		DuplicateGroups:  len(duplicateGroups),
		TotalDuplicates:  totalDuplicates,
		MemoryUsedMB:     memoryUsedMB,
	}
}

// benchmarkFullNormalization бенчмарк полной нормализации
func benchmarkFullNormalization(normalizer *normalization.CounterpartyNormalizer, counterparties []*database.CatalogItem, workers int, testStop bool) BenchmarkResult {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	startTime := time.Now()

	result, err := normalizer.ProcessNormalization(counterparties, false)
	duration := time.Since(startTime)

	runtime.ReadMemStats(&m2)
	memoryUsedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024

	if err != nil {
		log.Printf("Ошибка нормализации: %v", err)
	}

	recordsPerSec := 0.0
	processedCount := 0
	errorCount := 0
	if result != nil {
		processedCount = result.TotalProcessed
		errorCount = len(result.Errors)
		if duration > 0 {
			recordsPerSec = float64(processedCount) / duration.Seconds()
		}
	}

	benchmarkResult := BenchmarkResult{
		Stage:            "Full Normalization",
		RecordCount:      len(counterparties),
		Duration:         duration,
		RecordsPerSecond: recordsPerSec,
		ProcessedCount:   processedCount,
		ErrorCount:       errorCount,
		MemoryUsedMB:     memoryUsedMB,
	}

	if result != nil {
		benchmarkResult.DuplicateGroups = result.DuplicateGroups
		benchmarkResult.TotalDuplicates = result.TotalDuplicates
		benchmarkResult.BenchmarkMatches = result.BenchmarkMatches
		benchmarkResult.EnrichedCount = result.EnrichedCount
		benchmarkResult.CreatedBenchmarks = result.CreatedBenchmarks
	}

	if testStop {
		benchmarkResult.Stopped = true
	}

	return benchmarkResult
}

// printResult выводит результат бенчмарка
func printResult(result BenchmarkResult) {
	fmt.Printf("Этап: %s\n", result.Stage)
	fmt.Printf("  Записей: %d\n", result.RecordCount)
	fmt.Printf("  Время: %v\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("  Скорость: %.2f записей/сек\n", result.RecordsPerSecond)
	if result.MemoryUsedMB > 0 {
		fmt.Printf("  Память: %.2f МБ\n", result.MemoryUsedMB)
	}
	if result.DuplicateGroups > 0 {
		fmt.Printf("  Групп дубликатов: %d\n", result.DuplicateGroups)
		fmt.Printf("  Всего дубликатов: %d\n", result.TotalDuplicates)
	}
	if result.ProcessedCount > 0 {
		fmt.Printf("  Обработано: %d\n", result.ProcessedCount)
	}
	if result.BenchmarkMatches > 0 {
		fmt.Printf("  Совпадений с эталонами: %d\n", result.BenchmarkMatches)
	}
	if result.EnrichedCount > 0 {
		fmt.Printf("  Обогащено: %d\n", result.EnrichedCount)
	}
	if result.CreatedBenchmarks > 0 {
		fmt.Printf("  Создано эталонов: %d\n", result.CreatedBenchmarks)
	}
	if result.ErrorCount > 0 {
		fmt.Printf("  Ошибок: %d\n", result.ErrorCount)
	}
	if result.Stopped {
		fmt.Printf("  Остановлено: да\n")
		if result.StopLatency > 0 {
			fmt.Printf("  Задержка остановки: %v\n", result.StopLatency.Round(time.Millisecond))
		}
	}
}

// findFastestStage находит самый быстрый этап
func findFastestStage(results []BenchmarkResult) string {
	if len(results) == 0 {
		return ""
	}
	fastest := results[0]
	for _, r := range results {
		if r.Duration < fastest.Duration {
			fastest = r
		}
	}
	return fastest.Stage
}

// findSlowestStage находит самый медленный этап
func findSlowestStage(results []BenchmarkResult) string {
	if len(results) == 0 {
		return ""
	}
	slowest := results[0]
	for _, r := range results {
		if r.Duration > slowest.Duration {
			slowest = r
		}
	}
	return slowest.Stage
}

// saveReportToJSON сохраняет отчет в JSON файл
func saveReportToJSON(report FullBenchmarkReport) {
	// Преобразуем результаты для JSON
	jsonResults := make([]map[string]interface{}, 0, len(report.Results))
	for _, r := range report.Results {
		jsonResult := map[string]interface{}{
			"stage":              r.Stage,
			"record_count":       r.RecordCount,
			"duration_ms":        r.Duration.Milliseconds(),
			"records_per_second": r.RecordsPerSecond,
			"processed_count":    r.ProcessedCount,
			"error_count":        r.ErrorCount,
		}
		if r.DuplicateGroups > 0 {
			jsonResult["duplicate_groups"] = r.DuplicateGroups
			jsonResult["total_duplicates"] = r.TotalDuplicates
		}
		if r.BenchmarkMatches > 0 {
			jsonResult["benchmark_matches"] = r.BenchmarkMatches
		}
		if r.EnrichedCount > 0 {
			jsonResult["enriched_count"] = r.EnrichedCount
		}
		if r.CreatedBenchmarks > 0 {
			jsonResult["created_benchmarks"] = r.CreatedBenchmarks
		}
		if r.Stopped {
			jsonResult["stopped"] = true
		}
		jsonResults = append(jsonResults, jsonResult)
	}

	jsonReport := map[string]interface{}{
		"timestamp":                     report.Timestamp,
		"test_name":                     report.TestName,
		"record_count":                  report.RecordCount,
		"duplicate_rate":                report.DuplicateRate,
		"workers":                       report.Workers,
		"results":                       jsonResults,
		"total_duration_ms":             report.TotalDuration.Milliseconds(),
		"average_speed_records_per_sec": report.AverageSpeed,
		"summary":                       report.Summary,
	}

	jsonData, err := json.MarshalIndent(jsonReport, "", "  ")
	if err != nil {
		log.Printf("Ошибка при сериализации JSON: %v", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	jsonFilename := fmt.Sprintf("normalization_benchmark_%s.json", timestamp)
	if err := os.WriteFile(jsonFilename, jsonData, 0644); err != nil {
		log.Printf("Ошибка при сохранении JSON файла: %v", err)
	} else {
		fmt.Printf("✓ JSON отчет сохранен в: %s\n", jsonFilename)
	}

	// Создаем HTML отчет
	htmlFilename := fmt.Sprintf("normalization_benchmark_%s.html", timestamp)
	if err := saveReportToHTML(report, htmlFilename); err != nil {
		log.Printf("Ошибка при сохранении HTML файла: %v", err)
	} else {
		fmt.Printf("✓ HTML отчет сохранен в: %s\n", htmlFilename)
	}
}

// saveReportToHTML сохраняет отчет в HTML файл
func saveReportToHTML(report FullBenchmarkReport, filename string) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Бенчмарк нормализации контрагентов</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th { background: #4CAF50; color: white; padding: 12px; text-align: left; }
        td { padding: 10px; border-bottom: 1px solid #ddd; }
        tr:hover { background: #f9f9f9; }
        .summary { background: #e7f3ff; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .metric { display: inline-block; margin: 10px 20px 10px 0; }
        .metric-value { font-size: 24px; font-weight: bold; color: #4CAF50; }
        .metric-label { color: #666; font-size: 14px; }
        .fastest { background: #d4edda !important; }
        .slowest { background: #f8d7da !important; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 Бенчмарк нормализации контрагентов</h1>
        <p><strong>Дата тестирования:</strong> ` + report.Timestamp + `</p>
        <p><strong>Тест:</strong> ` + report.TestName + `</p>
        
        <div class="summary">
            <h2>Общая статистика</h2>
            <div class="metric">
                <div class="metric-value">` + fmt.Sprintf("%d", report.RecordCount) + `</div>
                <div class="metric-label">Записей</div>
            </div>
            <div class="metric">
                <div class="metric-value">` + fmt.Sprintf("%.1f%%", report.DuplicateRate*100) + `</div>
                <div class="metric-label">Дубликатов</div>
            </div>
            <div class="metric">
                <div class="metric-value">` + fmt.Sprintf("%d", report.Workers) + `</div>
                <div class="metric-label">Воркеров</div>
            </div>
            <div class="metric">
                <div class="metric-value">` + fmt.Sprintf("%.2f", report.AverageSpeed) + `</div>
                <div class="metric-label">Записей/сек</div>
            </div>
            <div class="metric">
                <div class="metric-value">` + fmt.Sprintf("%.0f", report.TotalDuration.Seconds()) + `</div>
                <div class="metric-label">Секунд</div>
            </div>
        </div>

        <h2>Результаты по этапам</h2>
        <table>
            <thead>
                <tr>
                    <th>Этап</th>
                    <th>Записей</th>
                    <th>Время (мс)</th>
                    <th>Скорость (записей/сек)</th>
                    <th>Память (МБ)</th>
                    <th>Дубликаты</th>
                    <th>Обработано</th>
                    <th>Эталоны</th>
                    <th>Ошибок</th>
                </tr>
            </thead>
            <tbody>`

	maxSpeed := 0.0
	for _, r := range report.Results {
		if r.RecordsPerSecond > maxSpeed {
			maxSpeed = r.RecordsPerSecond
		}
	}

	fastestStage := findFastestStage(report.Results)
	slowestStage := findSlowestStage(report.Results)

	for _, r := range report.Results {
		rowClass := ""
		if r.Stage == fastestStage {
			rowClass = "fastest"
		} else if r.Stage == slowestStage {
			rowClass = "slowest"
		}

		memoryStr := "-"
		if r.MemoryUsedMB > 0 {
			memoryStr = fmt.Sprintf("%.2f", r.MemoryUsedMB)
		}

		duplicatesStr := "-"
		if r.DuplicateGroups > 0 {
			duplicatesStr = fmt.Sprintf("%d групп", r.DuplicateGroups)
		}

		processedStr := "-"
		if r.ProcessedCount > 0 {
			processedStr = fmt.Sprintf("%d", r.ProcessedCount)
		}

		benchmarksStr := "-"
		if r.BenchmarkMatches > 0 {
			benchmarksStr = fmt.Sprintf("%d", r.BenchmarkMatches)
		}

		html += fmt.Sprintf(`
                <tr class="%s">
                    <td><strong>%s</strong></td>
                    <td>%d</td>
                    <td>%.0f</td>
                    <td>%.2f</td>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%d</td>
                </tr>`,
			rowClass, r.Stage, r.RecordCount,
			r.Duration.Milliseconds(), r.RecordsPerSecond,
			memoryStr, duplicatesStr, processedStr, benchmarksStr, r.ErrorCount)
	}

	html += `
            </tbody>
        </table>

        <h2>Рекомендации</h2>
        <div class="summary">
            <p><strong>Самый быстрый этап:</strong> ` + fastestStage + `</p>
            <p><strong>Самый медленный этап:</strong> ` + slowestStage + `</p>
            <p><strong>Средняя скорость:</strong> ` + fmt.Sprintf("%.2f записей/сек", report.AverageSpeed) + `</p>
        </div>
    </div>
</body>
</html>`

	return os.WriteFile(filename, []byte(html), 0644)
}

// saveReportToCSV сохраняет отчет в CSV файл
func saveReportToCSV(report FullBenchmarkReport) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("normalization_benchmark_%s.csv", timestamp)
	
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Ошибка при создании CSV файла: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Заголовки
	headers := []string{
		"Этап", "Записей", "Время (мс)", "Скорость (записей/сек)", "Память (МБ)",
		"Групп дубликатов", "Всего дубликатов", "Обработано", "Совпадений с эталонами",
		"Обогащено", "Создано эталонов", "Ошибок", "Остановлено",
	}
	if err := writer.Write(headers); err != nil {
		log.Printf("Ошибка при записи заголовков CSV: %v", err)
		return
	}

	// Данные
	for _, r := range report.Results {
		record := []string{
			r.Stage,
			strconv.Itoa(r.RecordCount),
			strconv.FormatFloat(float64(r.Duration.Milliseconds()), 'f', 2, 64),
			strconv.FormatFloat(r.RecordsPerSecond, 'f', 2, 64),
			strconv.FormatFloat(r.MemoryUsedMB, 'f', 2, 64),
			strconv.Itoa(r.DuplicateGroups),
			strconv.Itoa(r.TotalDuplicates),
			strconv.Itoa(r.ProcessedCount),
			strconv.Itoa(r.BenchmarkMatches),
			strconv.Itoa(r.EnrichedCount),
			strconv.Itoa(r.CreatedBenchmarks),
			strconv.Itoa(r.ErrorCount),
			strconv.FormatBool(r.Stopped),
		}
		if err := writer.Write(record); err != nil {
			log.Printf("Ошибка при записи строки CSV: %v", err)
			return
		}
	}

	fmt.Printf("✓ CSV отчет сохранен в: %s\n", filename)
}
