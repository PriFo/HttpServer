package normalization

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/extractors"
)

// BenchmarkFinder интерфейс для поиска эталонов
type BenchmarkFinder interface {
	FindBestMatch(name string, entityType string) (normalizedName string, found bool, err error)
}

// CounterpartyNormalizer нормализатор контрагентов
type CounterpartyNormalizer struct {
	serviceDB       *database.ServiceDB
	clientID        int
	projectID       int
	eventChannel    chan<- string
	ctx             context.Context  // Контекст для управления отменой операций
	nameNormalizer  AINameNormalizer // Интерфейс для нормализации имен с использованием AI
	benchmarkFinder BenchmarkFinder  // Интерфейс для поиска эталонов
	logger          *slog.Logger     // Структурированный логгер
	beforeProcessHook func(*database.CatalogItem) // Используется в тестах для синхронизации остановки
}

// CounterpartyNormalizationResult результат нормализации контрагентов
type CounterpartyNormalizationResult struct {
	ClientID          int      `json:"client_id"`
	ProjectID         int      `json:"project_id"`
	BenchmarkMatches  int      `json:"benchmark_matches"`
	EnrichedCount     int      `json:"enriched_count"`
	DuplicateGroups   int      `json:"duplicate_groups"`
	TotalProcessed    int      `json:"total_processed"`
	CreatedBenchmarks int      `json:"created_benchmarks"`
	Errors            []string `json:"errors"`
}

// ErrMsgNormalizationStopped сообщение об остановке нормализации
const ErrMsgNormalizationStopped = "Нормализация остановлена пользователем"

// NewCounterpartyNormalizer создает новый нормализатор контрагентов
// Если ctx равен nil, создается контекст, который никогда не отменяется (context.Background())
func NewCounterpartyNormalizer(
	serviceDB *database.ServiceDB,
	clientID int,
	projectID int,
	eventChannel chan<- string,
	ctx context.Context,
	nameNormalizer AINameNormalizer,
	benchmarkFinder BenchmarkFinder,
) *CounterpartyNormalizer {
	if ctx == nil {
		ctx = context.Background()
	}
	logger := slog.Default().With("component", "counterparty_normalizer", "client_id", clientID, "project_id", projectID)
	return &CounterpartyNormalizer{
		serviceDB:       serviceDB,
		clientID:        clientID,
		projectID:       projectID,
		eventChannel:    eventChannel,
		ctx:             ctx,
		nameNormalizer:  nameNormalizer,
		benchmarkFinder: benchmarkFinder,
		logger:          logger,
	}
}

// IsStopped проверяет, была ли операция отменена через контекст
func (cn *CounterpartyNormalizer) IsStopped() bool {
	select {
	case <-cn.ctx.Done():
		return true
	default:
		return false
	}
}

// ProcessNormalization обрабатывает нормализацию контрагентов
// Проверяет контекст для возможности отмены операции
func (cn *CounterpartyNormalizer) ProcessNormalization(counterparties []*database.CatalogItem, skipNormalized bool) (*CounterpartyNormalizationResult, error) {
	// Проверяем, не отменен ли контекст
	select {
	case <-cn.ctx.Done():
		return &CounterpartyNormalizationResult{
			ClientID:          cn.clientID,
			ProjectID:         cn.projectID,
			BenchmarkMatches:  0,
			EnrichedCount:     0,
			DuplicateGroups:   0,
			TotalProcessed:    0,
			CreatedBenchmarks: 0,
			Errors:            []string{ErrMsgNormalizationStopped},
		}, nil
	default:
		// Продолжаем выполнение
	}

	// Инициализируем результат
	result := &CounterpartyNormalizationResult{
		ClientID:          cn.clientID,
		ProjectID:         cn.projectID,
		BenchmarkMatches:  0,
		EnrichedCount:     0,
		DuplicateGroups:   0,
		TotalProcessed:    0,
		CreatedBenchmarks: 0,
		Errors:            []string{},
	}

	if len(counterparties) == 0 {
		cn.logger.Info("No counterparties to process")
		return result, nil
	}

	startTime := time.Now()
	cn.logger.Info("Starting counterparty normalization",
		"total", len(counterparties),
		"skip_normalized", skipNormalized)

	// Получаем список уже нормализованных контрагентов, если нужно пропускать
	normalizedMap := make(map[string]bool)
	if skipNormalized {
		checkStart := time.Now()
		sourceRefs := make([]string, 0, len(counterparties))
		for _, cp := range counterparties {
			if cp.Reference != "" {
				sourceRefs = append(sourceRefs, cp.Reference)
			}
		}
		if len(sourceRefs) > 0 {
			var err error
			normalizedMap, err = cn.serviceDB.GetNormalizedCounterpartiesBySourceReferences(cn.projectID, sourceRefs)
			if err != nil {
				cn.logger.Warn("Failed to get normalized counterparties",
					"error", err.Error(),
					"duration_ms", time.Since(checkStart).Milliseconds())
			} else {
			cn.logger.Info("Retrieved normalized counterparties map",
				"count", len(normalizedMap),
				"total_checked", len(sourceRefs),
				"duration_ms", time.Since(checkStart).Milliseconds())
			}
		}
	}

	// Обрабатываем контрагентов
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Ограничиваем параллелизм
	stopReported := false
	reportStop := func() {
		mu.Lock()
		defer mu.Unlock()
		if stopReported {
			return
		}
		result.Errors = append(result.Errors, ErrMsgNormalizationStopped)
		stopReported = true
	}

	for i, cp := range counterparties {
		// Проверяем контекст перед каждой итерацией
		select {
		case <-cn.ctx.Done():
			reportStop()
			cn.logger.Info("Normalization stopped by context",
				"processed", result.TotalProcessed,
				"total", len(counterparties))
			return result, nil
		default:
		}

		// Пропускаем уже нормализованные, если нужно
		if skipNormalized && normalizedMap[cp.Reference] {
			cn.logger.Debug("Skipping already normalized counterparty",
				"reference", cp.Reference,
				"counterparty_id", cp.ID)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Занимаем слот

		go func(index int, counterparty *database.CatalogItem) {
			defer func() {
				// Обработка паники в горутине для предотвращения краша всего процесса
				if rec := recover(); rec != nil {
					cn.logger.Error("Panic in counterparty processing goroutine",
						"counterparty_id", counterparty.ID,
						"recovered", rec)
					mu.Lock()
					result.Errors = append(result.Errors, fmt.Sprintf("panic processing counterparty %d: %v", counterparty.ID, rec))
					mu.Unlock()
				}
				wg.Done()
				<-semaphore // Освобождаем слот
			}()

			// Повторная проверка контекста уже внутри горутины
			select {
			case <-cn.ctx.Done():
				reportStop()
				return
			default:
			}

			if cn.beforeProcessHook != nil {
				cn.beforeProcessHook(counterparty)
			}

			// Проверяем контекст после пользовательского хука
			select {
			case <-cn.ctx.Done():
				reportStop()
				return
			default:
			}

			// Обрабатываем контрагента с измерением времени
			processStart := time.Now()
			if err := cn.processCounterparty(counterparty, result, &mu); err != nil {
				processDuration := time.Since(processStart)
				mu.Lock()
				result.Errors = append(result.Errors, fmt.Sprintf("counterparty %d: %v", counterparty.ID, err))
				mu.Unlock()
				cn.logger.Warn("Failed to process counterparty",
					"counterparty_id", counterparty.ID,
					"error", err.Error(),
					"duration_ms", processDuration.Milliseconds())
			} else {
				processDuration := time.Since(processStart)
				mu.Lock()
				result.TotalProcessed++
				mu.Unlock()
				// Логируем медленные операции (>1 секунды)
				if processDuration > time.Second {
					cn.logger.Debug("Slow counterparty processing",
						"counterparty_id", counterparty.ID,
						"duration_ms", processDuration.Milliseconds())
				}
			}

			// Отправляем событие о прогрессе каждые 10 контрагентов
			// Используем неблокирующую отправку с таймаутом для предотвращения блокировки
			if (index+1)%10 == 0 {
				mu.Lock()
				processed := result.TotalProcessed
				mu.Unlock()
				
				// Неблокирующая отправка события с таймаутом
				eventMsg := fmt.Sprintf("Обработано контрагентов: %d из %d", processed, len(counterparties))
				select {
				case cn.eventChannel <- eventMsg:
					// Событие успешно отправлено
				case <-time.After(100 * time.Millisecond):
					// Таймаут - канал переполнен, логируем предупреждение
					cn.logger.Debug("Event channel timeout, skipping progress event",
						"processed", processed,
						"total", len(counterparties))
				case <-cn.ctx.Done():
					// Контекст отменен, прекращаем отправку событий
					return
				}
			}
		}(i, cp)
	}

	wg.Wait()

	// Если в процессе ожидания произошла отмена контекста, фиксируем это и выходим
	select {
	case <-cn.ctx.Done():
		reportStop()
		cn.logger.Info("Normalization stopped while waiting for workers",
			"processed", result.TotalProcessed,
			"total", len(counterparties))
		return result, nil
	default:
	}

	// После нормализации всех контрагентов выполняем автоматический мэппинг и объединение дубликатов
	cn.logger.Info("Starting automatic counterparty mapping after normalization")
	mapper := NewCounterpartyMapper(cn.serviceDB)
	if err := mapper.MapAllCounterpartiesForProject(cn.projectID); err != nil {
		cn.logger.Warn("Failed to auto-map counterparties after normalization", "error", err)
		result.Errors = append(result.Errors, fmt.Sprintf("auto-mapping failed: %v", err))
	} else {
		cn.logger.Info("Successfully completed automatic counterparty mapping")
	}

	totalDuration := time.Since(startTime)
	cn.logger.Info("Counterparty normalization completed",
		"total_processed", result.TotalProcessed,
		"benchmark_matches", result.BenchmarkMatches,
		"enriched_count", result.EnrichedCount,
		"duplicate_groups", result.DuplicateGroups,
		"created_benchmarks", result.CreatedBenchmarks,
		"errors_count", len(result.Errors),
		"total_duration_ms", totalDuration.Milliseconds(),
		"avg_time_per_item_ms", func() int64 {
			if result.TotalProcessed > 0 {
				return totalDuration.Milliseconds() / int64(result.TotalProcessed)
			}
			return 0
		}(),
		"throughput_items_per_sec", func() float64 {
			if totalDuration.Seconds() > 0 {
				return float64(result.TotalProcessed) / totalDuration.Seconds()
			}
			return 0
		}())

	return result, nil
}

// processCounterparty обрабатывает одного контрагента
func (cn *CounterpartyNormalizer) processCounterparty(
	cp *database.CatalogItem,
	result *CounterpartyNormalizationResult,
	mu *sync.Mutex,
) error {
	// Валидация входных данных
	if cp == nil {
		return fmt.Errorf("counterparty is nil")
	}
	if cp.Name == "" {
		cn.logger.Debug("Skipping counterparty with empty name", "counterparty_id", cp.ID)
		return nil // Пропускаем контрагентов без имени
	}
	if len(cp.Name) > 1000 {
		cn.logger.Warn("Counterparty name too long, truncating",
			"counterparty_id", cp.ID,
			"original_length", len(cp.Name))
		cp.Name = cp.Name[:1000] // Обрезаем слишком длинные имена
	}

	// Извлекаем ИНН и БИН из атрибутов
	inn := ""
	bin := ""
	kpp := ""

	if cp.Attributes != "" {
		if extractedINN, err := extractors.ExtractINNFromAttributes(cp.Attributes); err == nil {
			inn = extractedINN
		}
		if extractedBIN, err := extractors.ExtractBINFromAttributes(cp.Attributes); err == nil {
			bin = extractedBIN
		}
		if extractedKPP, err := extractors.ExtractKPPFromAttributes(cp.Attributes); err == nil {
			kpp = extractedKPP
		}
	}

	// Если не нашли в атрибутах, пробуем в других полях
	if inn == "" && bin == "" {
		if cp.Code != "" {
			if extractedINN, err := extractors.ExtractINNFromAttributes(cp.Code); err == nil {
				inn = extractedINN
			} else if extractedBIN, err := extractors.ExtractBINFromAttributes(cp.Code); err == nil {
				bin = extractedBIN
			}
		}
		if inn == "" && bin == "" && cp.Reference != "" {
			if extractedINN, err := extractors.ExtractINNFromAttributes(cp.Reference); err == nil {
				inn = extractedINN
			} else if extractedBIN, err := extractors.ExtractBINFromAttributes(cp.Reference); err == nil {
				bin = extractedBIN
			}
		}
	}

	// Сначала проверяем эталоны перед AI-нормализацией
	normalizedName := cp.Name
	benchmarkFound := false

	if cn.benchmarkFinder != nil {
		normalized, found, err := cn.benchmarkFinder.FindBestMatch(cp.Name, "counterparty")
		if err == nil && found {
			normalizedName = normalized
			benchmarkFound = true
			cn.logger.Info("Found benchmark for counterparty by name",
				"counterparty_id", cp.ID,
				"original_name", cp.Name,
				"normalized_name", normalizedName)
		} else if err != nil {
			cn.logger.Warn("Error searching for benchmark",
				"counterparty_id", cp.ID,
				"error", err.Error())
		}
	}

	// Если эталон не найден, используем AI-нормализацию
	if !benchmarkFound && cn.nameNormalizer != nil {
		normalizedCtx, cancel := context.WithTimeout(cn.ctx, 30*time.Second)
		defer cancel()

		var err error
		if inn != "" || bin != "" {
			// Используем NormalizeCounterparty для лучшей точности
			normalizedName, err = cn.nameNormalizer.NormalizeCounterparty(normalizedCtx, cp.Name, inn, bin)
		} else {
			// Используем только NormalizeName
			normalizedName, err = cn.nameNormalizer.NormalizeName(normalizedCtx, cp.Name)
		}

		if err != nil {
			cn.logger.Warn("Failed to normalize name via AI, using original",
				"counterparty_id", cp.ID,
				"error", err.Error())
			// Используем исходное имя при ошибке
			normalizedName = cp.Name
		}
	}

	// Извлекаем дополнительные данные из атрибутов
	legalAddress := ""
	postalAddress := ""
	contactPhone := ""
	contactEmail := ""
	contactPerson := ""
	legalForm := ""
	bankName := ""
	bankAccount := ""
	correspondentAccount := ""
	bik := ""

	if cp.Attributes != "" {
		if addr, err := extractors.ExtractAddressFromAttributes(cp.Attributes); err == nil {
			legalAddress = addr
			postalAddress = addr
		}
		if phone, err := extractors.ExtractContactPhoneFromAttributes(cp.Attributes); err == nil {
			contactPhone = phone
		}
		if email, err := extractors.ExtractContactEmailFromAttributes(cp.Attributes); err == nil {
			contactEmail = email
		}
		if person, err := extractors.ExtractContactPersonFromAttributes(cp.Attributes); err == nil {
			contactPerson = person
		}
		if form, err := extractors.ExtractLegalFormFromAttributes(cp.Attributes); err == nil {
			legalForm = form
		}
		if bank, err := extractors.ExtractBankNameFromAttributes(cp.Attributes); err == nil {
			bankName = bank
		}
		if account, err := extractors.ExtractBankAccountFromAttributes(cp.Attributes); err == nil {
			bankAccount = account
		}
		if corrAccount, err := extractors.ExtractCorrespondentAccountFromAttributes(cp.Attributes); err == nil {
			correspondentAccount = corrAccount
		}
		if bikCode, err := extractors.ExtractBIKFromAttributes(cp.Attributes); err == nil {
			bik = bikCode
		}
	}

	// Проверяем на эталоны (benchmarks) по ИНН/БИН
	benchmarkID := 0
	if inn != "" || bin != "" {
		// Ищем эталон по ИНН или БИН
		// Приоритет: сначала по ИНН, затем по БИН
		taxID := inn
		if taxID == "" {
			taxID = bin
		}
		
		if taxID != "" {
			benchmarkStart := time.Now()
			benchmark, err := cn.serviceDB.FindBenchmarkByTaxID(cn.projectID, taxID)
			benchmarkDuration := time.Since(benchmarkStart)
			
			if err != nil {
				cn.logger.Warn("Error searching for benchmark by tax ID",
					"counterparty_id", cp.ID,
					"tax_id", taxID,
					"error", err.Error(),
					"duration_ms", benchmarkDuration.Milliseconds())
			} else if benchmark != nil {
				benchmarkID = benchmark.ID
				// Используем нормализованное имя из эталона, если оно лучше
				if benchmark.NormalizedName != "" && normalizedName != benchmark.NormalizedName {
					cn.logger.Info("Found benchmark by tax ID, using normalized name from benchmark",
						"counterparty_id", cp.ID,
						"tax_id", taxID,
						"benchmark_id", benchmarkID,
						"benchmark_normalized_name", benchmark.NormalizedName,
						"duration_ms", benchmarkDuration.Milliseconds())
					normalizedName = benchmark.NormalizedName
					benchmarkFound = true
				} else {
					cn.logger.Debug("Found benchmark by tax ID but name unchanged",
						"counterparty_id", cp.ID,
						"tax_id", taxID,
						"benchmark_id", benchmarkID,
						"duration_ms", benchmarkDuration.Milliseconds())
				}
			}
		}
	}

	// Сохраняем нормализованного контрагента
	enrichmentApplied := false
	sourceEnrichment := ""
	if legalAddress != "" || contactPhone != "" || contactEmail != "" {
		enrichmentApplied = true
		sourceEnrichment = "attributes"
		mu.Lock()
		result.EnrichedCount++
		mu.Unlock()
	}

	qualityScore := 0.8 // Базовая оценка качества
	// Если найден эталон (по имени или по taxID), увеличиваем счетчик и качество
	hasBenchmark := benchmarkID > 0 || benchmarkFound
	if hasBenchmark {
		qualityScore = 1.0 // Эталон имеет максимальную оценку
		mu.Lock()
		// Увеличиваем счетчик только один раз, если эталон найден любым способом
		result.BenchmarkMatches++
		mu.Unlock()
	}

	// Проверяем контекст перед сохранением в БД
	select {
	case <-cn.ctx.Done():
		return fmt.Errorf("normalization stopped before saving counterparty")
	default:
	}

	saveStart := time.Now()
	
	// Используем retry логику для критических операций сохранения
	saveFunc := func() error {
		return cn.serviceDB.SaveNormalizedCounterparty(
			cn.projectID,
			cp.Reference,
			cp.Name,
			normalizedName,
			inn,
			kpp,
			bin,
			legalAddress,
			postalAddress,
			contactPhone,
			contactEmail,
			contactPerson,
			legalForm,
			bankName,
			bankAccount,
			correspondentAccount,
			bik,
			benchmarkID,
			qualityScore,
			enrichmentApplied,
			sourceEnrichment,
			cp.CatalogName,
			"", // subcategory
		)
	}
	
	// Retry конфигурация для операций с БД
	retryConfig := DefaultRetryConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.InitialDelay = 100 * time.Millisecond
	retryConfig.MaxDelay = 1 * time.Second
	
	err := Retry(saveFunc, retryConfig)
	saveDuration := time.Since(saveStart)

	if err != nil {
		cn.logger.Warn("Failed to save normalized counterparty after retries",
			"counterparty_id", cp.ID,
			"error", err.Error(),
			"duration_ms", saveDuration.Milliseconds())
		return fmt.Errorf("failed to save normalized counterparty: %w", err)
	}

	// Логируем медленные операции сохранения (>500ms)
	if saveDuration > 500*time.Millisecond {
		cn.logger.Debug("Slow counterparty save operation",
			"counterparty_id", cp.ID,
			"duration_ms", saveDuration.Milliseconds())
	}

	return nil
}
