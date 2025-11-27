package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл содержит конструкторы Server (NewServer, NewServerWithConfig), извлеченные из server.go
// для сокращения размера server.go

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"httpserver/database"
	"httpserver/internal/infrastructure/ai"
	"httpserver/internal/infrastructure/cache"
	inframonitoring "httpserver/internal/infrastructure/monitoring"
	infranormalization "httpserver/internal/infrastructure/normalization"
	"httpserver/internal/infrastructure/workers"
	"httpserver/nomenclature"
	"httpserver/server/handlers"
	"httpserver/server/services"
)

func NewServer(db *database.DB, normalizedDB *database.DB, serviceDB *database.ServiceDB, dbPath, normalizedDBPath, port string) *Server {
	cfg := &Config{
		Port:                       port,
		DatabasePath:               dbPath,
		NormalizedDatabasePath:     normalizedDBPath,
		ServiceDatabasePath:        "service.db",
		LogBufferSize:              100,
		NormalizerEventsBufferSize: 100,
	}
	return NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, cfg)
}

// NewServerWithConfig создает новый сервер с конфигурацией
func NewServerWithConfig(db *database.DB, normalizedDB *database.DB, serviceDB *database.ServiceDB, dbPath, normalizedDBPath string, config *Config) *Server {
	log.Printf("Создание DI контейнера...")
	// Создаем DI контейнер для управления зависимостями
	container, err := NewContainer(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, config)
	if err != nil {
		log.Printf("✗ КРИТИЧЕСКАЯ ОШИБКА: Не удалось создать контейнер зависимостей")
		log.Printf("  Детали ошибки: %v", err)
		log.Printf("  Тип ошибки: %T", err)
		log.Printf("  Проверьте логи выше для деталей инициализации компонентов")
		// Даем время на вывод логов перед завершением
		time.Sleep(2 * time.Second)
		log.Fatalf("Failed to create container: %v", err)
	}
	log.Printf("✓ DI контейнер успешно создан")

	// Получаем зависимости из контейнера
	normalizerEvents := container.NormalizerEvents
	normalizer := container.Normalizer
	qualityAnalyzer := container.QualityAnalyzer
	arliaiClient := container.ArliaiClient
	arliaiCache := container.ArliaiCache
	openrouterClient := container.OpenRouterClient
	huggingfaceClient := container.HuggingFaceClient
	similarityCache := container.SimilarityCache

	// Получаем менеджеры из контейнера (с type assertion)
	workerConfigManager, _ := container.WorkerConfigManager.(*workers.WorkerConfigManager)
	monitoringManager, _ := container.MonitoringManager.(*inframonitoring.Manager)
	providerOrchestrator, _ := container.ProviderOrchestrator.(*ai.ProviderOrchestrator)
	scanHistoryManager, _ := container.ScanHistoryManager.(*cache.ScanHistoryManager)
	dbModificationTracker, _ := container.DatabaseModificationTracker.(*cache.DatabaseModificationTracker)
	enrichmentFactory := container.EnrichmentFactory
	hierarchicalClassifier := container.HierarchicalClassifier

	// Получаем кэши из контейнера
	dbInfoCache := container.DatabaseInfoCache
	systemSummaryCache := container.SystemSummaryCache
	dbConnectionCache := container.DatabaseConnectionCache

	log.Printf("Provider orchestrator initialized with strategy: %s, timeout: %v, multi-provider enabled: %v",
		providerOrchestrator.GetStrategy(), config.AITimeout, config.MultiProviderEnabled)

	// Регистрируем провайдеры в мониторинге и оркестраторе
	// Arliai: 2 канала (по умолчанию из конфигурации)
	// ВАЖНО: MaxWorkers=2 означает ограничение на ПАРАЛЛЕЛЬНЫЕ ЗАПРОСЫ, а НЕ на количество моделей
	// Бенчмарк должен тестировать ВСЕ доступные модели, а не только 2
	// Получаем API ключ из workerConfigManager или переменной окружения
	arliaiAPIKey := os.Getenv("ARLIAI_API_KEY")
	if workerConfigManager != nil {
		if apiKey, _, err := workerConfigManager.GetModelAndAPIKey(); err == nil && apiKey != "" {
			arliaiAPIKey = apiKey
		}
	}
	if arliaiClient != nil && arliaiAPIKey != "" {
		monitoringManager.RegisterProvider("arliai", "Arliai", 2)
		// Создаем AI клиент для Arliai
		model := os.Getenv("ARLIAI_MODEL")
		if model == "" {
			model = "GLM-4.5-Air"
		}
		arliaiAIClient := nomenclature.NewAIClient(arliaiAPIKey, model)
		arliaiAdapter := ai.NewArliaiProviderAdapter(arliaiAIClient)
		// Получаем приоритет из конфигурации
		arliaiPriority := 1
		if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "arliai" {
			arliaiPriority = provider.Priority
		}
		providerOrchestrator.RegisterProvider("arliai", "Arliai", arliaiAdapter, true, arliaiPriority)
	}
	// OpenRouter: 1 канал
	openrouterAPIKey := os.Getenv("OPENROUTER_API_KEY")
	if workerConfigManager != nil {
		if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "openrouter" {
			if provider.APIKey != "" {
				openrouterAPIKey = provider.APIKey
			}
		}
	}
	if openrouterClient != nil && openrouterAPIKey != "" {
		monitoringManager.RegisterProvider("openrouter", "OpenRouter", 1)
		// Создаем OpenRouterClient
		serverOpenRouterClient := ai.NewOpenRouterClient(openrouterAPIKey)
		openrouterAdapter := ai.NewOpenRouterProviderAdapter(serverOpenRouterClient)
		// Получаем приоритет из конфигурации
		openrouterPriority := 2
		if workerConfigManager != nil {
			if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "openrouter" {
				openrouterPriority = provider.Priority
			}
		}
		providerOrchestrator.RegisterProvider("openrouter", "OpenRouter", openrouterAdapter, true, openrouterPriority)
	}
	// Hugging Face: 1 канал
	huggingfaceAPIKey := os.Getenv("HUGGINGFACE_API_KEY")
	if workerConfigManager != nil {
		if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "huggingface" {
			if provider.APIKey != "" {
				huggingfaceAPIKey = provider.APIKey
			}
		}
	}
	if huggingfaceAPIKey != "" {
		monitoringManager.RegisterProvider("huggingface", "Hugging Face", 1)
		// Создаем HuggingFaceClient из API ключа
		baseURL := "https://api-inference.huggingface.co"
		if workerConfigManager != nil {
			if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "huggingface" && provider.BaseURL != "" {
				baseURL = provider.BaseURL
			}
		}
		serverHuggingFaceClient := ai.NewHuggingFaceClient(huggingfaceAPIKey, baseURL)
		huggingfaceAdapter := ai.NewHuggingFaceProviderAdapter(serverHuggingFaceClient)
		// Получаем приоритет из конфигурации
		huggingfacePriority := 3
		if workerConfigManager != nil {
			if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "huggingface" {
				huggingfacePriority = provider.Priority
			}
		}
		providerOrchestrator.RegisterProvider("huggingface", "Hugging Face", huggingfaceAdapter, true, huggingfacePriority)
	}
	// Eden AI: 1 канал
	edenaiAPIKey := os.Getenv("EDENAI_API_KEY")
	if workerConfigManager != nil {
		if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "edenai" {
			if provider.APIKey != "" {
				edenaiAPIKey = provider.APIKey
			}
		}
	}
	if edenaiAPIKey != "" {
		monitoringManager.RegisterProvider("edenai", "Eden AI", 1)
		// Создаем server.EdenAIClient из API ключа
		edenaiBaseURL := "https://api.edenai.run/v2"
		if workerConfigManager != nil {
			if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "edenai" && provider.BaseURL != "" {
				edenaiBaseURL = provider.BaseURL
			}
		}
		serverEdenAIClient := ai.NewEdenAIClient(edenaiAPIKey, edenaiBaseURL)
		edenaiAdapter := ai.NewEdenAIProviderAdapter(serverEdenAIClient)
		// Получаем приоритет из конфигурации
		edenaiPriority := 4
		if workerConfigManager != nil {
			if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "edenai" {
				edenaiPriority = provider.Priority
			}
		}
		providerOrchestrator.RegisterProvider("edenai", "Eden AI", edenaiAdapter, true, edenaiPriority)
	}

	// Создаем мульти-провайдерный клиент для нормализации имен контрагентов
	var multiProviderClient *MultiProviderClient
	if serviceDB != nil {
		// Получаем провайдеров из БД
		providersFromDB, err := serviceDB.GetActiveProviders()
		if err != nil {
			log.Printf("Warning: Failed to get providers from DB: %v. Multi-provider client will not be initialized.", err)
		} else {
			// Создаем мапу клиентов
			clients := make(map[string]ai.ProviderClient)
			// Получаем API ключи из конфигурации или переменных окружения
			arliaiAPIKeyForMulti := os.Getenv("ARLIAI_API_KEY")
			if workerConfigManager != nil {
				if apiKey, _, err := workerConfigManager.GetModelAndAPIKey(); err == nil && apiKey != "" {
					arliaiAPIKeyForMulti = apiKey
				}
			}
			if arliaiAPIKeyForMulti != "" {
				model := os.Getenv("ARLIAI_MODEL")
				if model == "" {
					model = "GLM-4.5-Air"
				}
				arliaiAIClient := nomenclature.NewAIClient(arliaiAPIKeyForMulti, model)
				clients["arliai"] = ai.NewArliaiProviderAdapter(arliaiAIClient)
			}
			openrouterAPIKeyForMulti := os.Getenv("OPENROUTER_API_KEY")
			if workerConfigManager != nil {
				if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "openrouter" && provider.APIKey != "" {
					openrouterAPIKeyForMulti = provider.APIKey
				}
			}
			if openrouterAPIKeyForMulti != "" {
				// Используем ai.NewOpenRouterClient для правильного типа
				openRouterClientForMulti := ai.NewOpenRouterClient(openrouterAPIKeyForMulti)
				clients["openrouter"] = ai.NewOpenRouterProviderAdapter(openRouterClientForMulti)
			}
			huggingfaceAPIKeyForMulti := os.Getenv("HUGGINGFACE_API_KEY")
			if workerConfigManager != nil {
				if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "huggingface" && provider.APIKey != "" {
					huggingfaceAPIKeyForMulti = provider.APIKey
				}
			}
			if huggingfaceAPIKeyForMulti != "" {
				baseURLForMulti := "https://api-inference.huggingface.co"
				if workerConfigManager != nil {
					if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "huggingface" && provider.BaseURL != "" {
						baseURLForMulti = provider.BaseURL
					}
				}
				// Используем ai.NewHuggingFaceClient для правильного типа
				huggingFaceClientForMulti := ai.NewHuggingFaceClient(huggingfaceAPIKeyForMulti, baseURLForMulti)
				clients["huggingface"] = ai.NewHuggingFaceProviderAdapter(huggingFaceClientForMulti)
			}
			edenaiAPIKeyForMulti := os.Getenv("EDENAI_API_KEY")
			if workerConfigManager != nil {
				if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "edenai" && provider.APIKey != "" {
					edenaiAPIKeyForMulti = provider.APIKey
				}
			}
			if edenaiAPIKeyForMulti != "" {
				// Создаем server.EdenAIClient из API ключа
				edenaiBaseURLForMulti := os.Getenv("EDENAI_BASE_URL")
				if edenaiBaseURLForMulti == "" {
					edenaiBaseURLForMulti = "https://api.edenai.run/v2"
				}
				if workerConfigManager != nil {
					if provider, err := workerConfigManager.GetActiveProvider(); err == nil && provider.Name == "edenai" && provider.BaseURL != "" {
						edenaiBaseURLForMulti = provider.BaseURL
					}
				}
				// Используем ai.NewEdenAIClient для правильного типа
				edenAIClientForMulti := ai.NewEdenAIClient(edenaiAPIKeyForMulti, edenaiBaseURLForMulti)
				clients["edenai"] = ai.NewEdenAIProviderAdapter(edenAIClientForMulti)
			}

			// Создаем роутер для контрагентов (DaData/Adata)
			var counterpartyRouter *CounterpartyProviderRouter
			dadataAdapter, hasDadata := clients["dadata"]
			adataAdapter, hasAdata := clients["adata"]
			if hasDadata || hasAdata {
				counterpartyRouter = NewCounterpartyProviderRouter(dadataAdapter, adataAdapter)
			}

			// Создаем мульти-провайдерный клиент
			multiProviderClient = NewMultiProviderClient(providersFromDB, clients, counterpartyRouter)
			log.Printf("Multi-provider client initialized with %d active providers, %d total channels",
				multiProviderClient.GetActiveProvidersCount(), multiProviderClient.GetTotalChannels())
		}
	}

	// Кэши и менеджеры уже получены из контейнера выше
	// dbInfoCache, systemSummaryCache, dbConnectionCache, scanHistoryManager, dbModificationTracker

	// Создаем БД эталонов (нужно для benchmarkService)
	benchmarksDBPath := filepath.Join("data", "benchmarks.db")
	benchmarksDB, err := database.NewBenchmarksDB(benchmarksDBPath)
	if err != nil {
		log.Fatalf("Failed to create benchmarks database: %v", err)
	}

	// Создаем benchmark service (нужен для counterpartyService и normalizer)
	benchmarkService := services.NewBenchmarkService(benchmarksDB, db, serviceDB)

	// Устанавливаем BenchmarkFinder в normalizer для проверки эталонов перед AI
	if benchmarkService != nil {
		benchmarkFinderAdapter := &services.BenchmarkFinderAdapter{BenchmarkService: benchmarkService}
		normalizer.SetBenchmarkFinder(benchmarkFinderAdapter)
	}

	// Создаем сервисы
	normalizationService := services.NewNormalizationService(db, serviceDB, normalizer, benchmarkService, normalizerEvents)
	counterpartyService := services.NewCounterpartyService(serviceDB, normalizerEvents, benchmarkService)

	// Создаем базовый handler
	baseHandler := handlers.NewBaseHandlerFromMiddleware()

	// Создаем counterparty handler
	counterpartyExportManager := handlers.NewDefaultCounterpartyExportManager()
	counterpartyHandler := handlers.NewCounterpartyHandler(baseHandler, counterpartyService, func(entry interface{}) {
		// logFunc будет установлен в Start()
	})
	counterpartyHandler.SetExportManager(counterpartyExportManager)
	// Устанавливаем enrichmentFactory для массового обогащения
	if counterpartyHandler != nil && enrichmentFactory != nil {
		counterpartyHandler.SetEnrichmentFactory(enrichmentFactory)
	}

	// Создаем upload service и handler
	// logFunc будет установлен в Start() через замыкание на s.log
	var uploadLogFunc func(entry LogEntry)
	uploadService := services.NewUploadService(db, serviceDB, dbInfoCache, func(entry interface{}) {
		if uploadLogFunc != nil {
			// Преобразуем interface{} в LogEntry
			if logEntry, ok := entry.(LogEntry); ok {
				uploadLogFunc(logEntry)
			}
		}
	})
	// Создаем notification service (будет использован для всех обработчиков)
	// serviceDB обязателен для NotificationService
	if serviceDB == nil {
		log.Fatalf("✗ КРИТИЧЕСКАЯ ОШИБКА: serviceDB равен nil, NotificationService не может быть создан")
	}
	notificationService := services.NewNotificationService(serviceDB)

	uploadHandler := handlers.NewUploadHandlerWithNotifications(
		uploadService,
		notificationService,
		baseHandler,
		func(entry interface{}) {
			if uploadLogFunc != nil {
				// Преобразуем interface{} в LogEntry
				if logEntry, ok := entry.(LogEntry); ok {
					uploadLogFunc(logEntry)
				}
			}
		},
	)

	// Создаем client service и handler
	clientService, err := services.NewClientService(serviceDB, db, normalizedDB)
	if err != nil {
		log.Fatalf("Failed to create client service: %v", err)
	}
	clientHandler := handlers.NewClientHandler(clientService, baseHandler)
	// Функции будут установлены после создания Server, так как они требуют доступ к методам Server

	// Создаем normalization handler (будет обновлен в Start() с функцией запуска)
	normalizationHandler := handlers.NewNormalizationHandler(normalizationService, baseHandler, normalizerEvents)
	// Устанавливаем доступ к базам данных
	normalizationHandler.SetDatabase(db, dbPath, normalizedDB, normalizedDBPath)

	// Создаем quality service и handler
	qualityService, err := services.NewQualityService(db, qualityAnalyzer)
	if err != nil {
		log.Fatalf("Failed to create quality service: %v", err)
	}
	qualityHandler := handlers.NewQualityHandler(baseHandler, qualityService, func(entry interface{}) {
		// logFunc будет установлен в Start()
		// Преобразуем interface{} в LogEntry при необходимости
	}, normalizedDB, normalizedDBPath)

	// Создаем classification service и handler
	var getAPIKey func() string
	if workerConfigManager != nil {
		getAPIKey = func() string {
			apiKey, _, err := workerConfigManager.GetModelAndAPIKey()
			if err == nil && apiKey != "" {
				return apiKey
			}
			return os.Getenv("ARLIAI_API_KEY")
		}
	}
	classificationService := services.NewClassificationService(db, normalizedDB, serviceDB, func() string {
		// getModelFromConfig будет установлен в Start()
		return "GLM-4.5-Air" // Дефолтная модель
	}, getAPIKey)
	classificationHandler := handlers.NewClassificationHandler(baseHandler, classificationService, func(entry interface{}) {
		// logFunc будет установлен в Start()
	})

	// Создаем similarity service и handler
	similarityService := services.NewSimilarityService(similarityCache)
	similarityHandler := handlers.NewSimilarityHandler(baseHandler, similarityService, func(entry interface{}) {
		// logFunc будет установлен в Start()
	})

	// Создаем monitoring service и handler
	// Создаем адаптер для Normalizer, чтобы соответствовать интерфейсу
	normalizerAdapter := &infranormalization.Adapter{Normalizer: normalizer}
	monitoringService := services.NewMonitoringService(db, normalizerAdapter, time.Now())
	monitoringHandler := handlers.NewMonitoringHandler(
		baseHandler,
		monitoringService,
		func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
		func() map[string]interface{} {
			// getCircuitBreakerState будет установлен в Start()
			return map[string]interface{}{"state": "closed"}
		},
		func() map[string]interface{} {
			// getBatchProcessorStats будет установлен в Start()
			return map[string]interface{}{}
		},
		func() map[string]interface{} {
			// getCheckpointStatus будет установлен в Start()
			return map[string]interface{}{}
		},
		func() *database.PerformanceMetricsSnapshot {
			// collectMetricsSnapshot будет установлен в Start()
			return nil
		},
		func() handlers.MonitoringData {
			// getMonitoringMetrics - функция для получения метрик провайдеров
			// Будет установлена в Start() для доступа к monitoringManager
			return handlers.MonitoringData{
				Providers: []handlers.ProviderMetrics{},
				System:    handlers.SystemStats{},
			}
		},
	)

	// Создаем report service и handler
	reportService := services.NewReportService(db, normalizedDB, serviceDB)
	reportHandler := handlers.NewReportHandler(
		baseHandler,
		reportService,
		func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
		func() (interface{}, error) {
			// generateNormalizationReport будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
		func(projectID *int) (interface{}, error) {
			// generateDataQualityReport будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
		func(databasePath string) (interface{}, error) {
			// generateQualityReport будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
	)

	// Создаем snapshot service и handler
	snapshotService := services.NewSnapshotService(db)
	snapshotHandler := handlers.NewSnapshotHandler(
		baseHandler,
		snapshotService,
		func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
		serviceDB,
		func(snapshotID int, req interface{}) (interface{}, error) {
			// normalizeSnapshotFunc будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
		func(snapshotID int) (interface{}, error) {
			// compareSnapshotIterations будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
		func(snapshotID int) (interface{}, error) {
			// calculateSnapshotMetrics будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
		func(snapshotID int) (interface{}, error) {
			// getSnapshotEvolution будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
		func(projectID int, uploadsPerDatabase int, name, description string) (*database.DataSnapshot, error) {
			// createAutoSnapshotFunc будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
	)

	// Создаем database service и handler
	databaseService := services.NewDatabaseService(
		serviceDB,
		db,
		normalizedDB,
		dbPath,
		normalizedDBPath,
		dbInfoCache,
	)
	databaseHandler := handlers.NewDatabaseHandler(
		databaseService,
		baseHandler,
	)

	// Создаем error metrics handler
	errorMetricsHandler := handlers.NewErrorMetricsHandler(baseHandler)

	// Получаем системные handlers из контейнера
	systemHandler := container.SystemHandler
	systemSummaryHandler := container.SystemSummaryHandler
	healthChecker := container.HealthChecker
	metricsCollector := container.MetricsCollector

	// Создаем legacy upload handler
	uploadLegacyHandler := handlers.NewUploadLegacyHandler(
		db,
		serviceDB,
		dbInfoCache,
		qualityAnalyzer,
		func(entry LogEntry) {
			// logFunc будет установлен в Start()
		},
	)

	// Используем сервисы и handlers из контейнера
	// NomenclatureHandler будет создан в Start() с правильными функциями
	var nomenclatureHandler *handlers.NomenclatureHandler
	// DashboardHandler будет создан в Start() с правильными функциями
	var dashboardHandler *handlers.DashboardHandler
	notificationHandler := container.NotificationHandler

	// Временно создаем dashboardHandler (будет пересоздан в Start() с правильными функциями)
	// Используем сервисы из контейнера
	dashboardHandler = handlers.NewDashboardHandlerWithServices(
		container.DashboardService,
		clientService,
		normalizationService,
		qualityService,
		baseHandler,
		func() handlers.MonitoringData {
			// getMonitoringMetrics - преобразуем MonitoringData из server пакета в handlers.MonitoringData
			if monitoringManager == nil {
				return handlers.MonitoringData{
					Providers: []handlers.ProviderMetrics{},
					System:    handlers.SystemStats{},
				}
			}
			serverData := monitoringManager.GetAllMetrics()
			// Преобразуем провайдеры
			providers := make([]handlers.ProviderMetrics, len(serverData.Providers))
			for i, p := range serverData.Providers {
				lastRequestTimeStr := ""
				if !p.LastRequestTime.IsZero() {
					lastRequestTimeStr = p.LastRequestTime.Format(time.RFC3339)
				}
				providers[i] = handlers.ProviderMetrics{
					ID:                 p.ID,
					Name:               p.Name,
					ActiveChannels:     p.ActiveChannels,
					CurrentRequests:    p.CurrentRequests,
					TotalRequests:      p.TotalRequests,
					SuccessfulRequests: p.SuccessfulRequests,
					FailedRequests:     p.FailedRequests,
					AverageLatencyMs:   p.AverageLatencyMs,
					LastRequestTime:    lastRequestTimeStr,
					Status:             p.Status,
					RequestsPerSecond:  p.RequestsPerSecond,
				}
			}
			// Преобразуем системную статистику
			timestampStr := ""
			if !serverData.System.Timestamp.IsZero() {
				timestampStr = serverData.System.Timestamp.Format(time.RFC3339)
			}
			return handlers.MonitoringData{
				Providers: providers,
				System: handlers.SystemStats{
					TotalProviders:          serverData.System.TotalProviders,
					ActiveProviders:         serverData.System.ActiveProviders,
					TotalRequests:           serverData.System.TotalRequests,
					TotalSuccessful:         serverData.System.TotalSuccessful,
					TotalFailed:             serverData.System.TotalFailed,
					SystemRequestsPerSecond: serverData.System.SystemRequestsPerSecond,
					Timestamp:               timestampStr,
				},
			}
		},
	)

	// Создаем GISP service и handler
	gispService := services.NewGISPService(serviceDB)
	gispHandler := handlers.NewGISPHandler(
		gispService,
		baseHandler,
	)

	// Создаем GOSTs database, service and handler
	gostsDB, err := database.NewGostsDB("gosts.db")
	if err != nil {
		// Логируем ошибку, но не прерываем создание сервера
		// База ГОСТов может быть создана позже
		log.Printf("Warning: failed to initialize GOSTs database: %v", err)
	}
	var gostService *services.GostService
	var gostHandler *handlers.GostHandler
	if gostsDB != nil {
		gostService = services.NewGostService(gostsDB)
		gostHandler = handlers.NewGostHandler(gostService)
	}

	// Создаем benchmark handler (benchmarkService уже создан выше)
	benchmarkHandler := handlers.NewBenchmarkHandler(
		benchmarkService,
		baseHandler,
	)

	// Создаем processing1c service и handler
	processing1CService := services.NewProcessing1CService()
	processing1CHandler := handlers.NewProcessing1CHandler(
		processing1CService,
		baseHandler,
	)

	// Создаем duplicate detection service и handler
	duplicateDetectionService := services.NewDuplicateDetectionService()
	duplicateDetectionHandler := handlers.NewDuplicateDetectionHandler(
		duplicateDetectionService,
		baseHandler,
	)

	// Создаем pattern detection service и handler
	patternDetectionService := services.NewPatternDetectionService(func() string {
		// getArliaiAPIKey будет установлен в Start()
		return ""
	})
	patternDetectionHandler := handlers.NewPatternDetectionHandler(
		patternDetectionService,
		baseHandler,
		func(limit int, table, column string) ([]string, error) {
			// getNamesFunc будет установлен в Start()
			return nil, fmt.Errorf("not implemented")
		},
	)

	// Создаем reclassification service и handler
	reclassificationService := services.NewReclassificationService()
	reclassificationHandler := handlers.NewReclassificationHandler(
		reclassificationService,
		baseHandler,
	)

	// Создаем normalization benchmark service и handler
	normalizationBenchmarkService := services.NewNormalizationBenchmarkService()
	normalizationBenchmarkHandler := handlers.NewNormalizationBenchmarkHandler(
		normalizationBenchmarkService,
		baseHandler,
	)

	// Создаем worker trace handler
	workerTraceHandler := handlers.NewWorkerTraceHandler(
		baseHandler,
		func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
	)

	// Создаем diagnostics handler (будет инициализирован после создания Server)
	var diagnosticsHandler *handlers.DiagnosticsHandler

	// Создаем config handler
	var configHandler *handlers.ConfigHandler
	if serviceDB == nil {
		log.Printf("✗ КРИТИЧЕСКАЯ ОШИБКА: serviceDB равен nil, configHandler не может быть создан")
		log.Fatalf("Failed to create configHandler: serviceDB is nil")
	}
	configHandler = handlers.NewConfigHandler(serviceDB)
	if configHandler == nil {
		log.Printf("✗ КРИТИЧЕСКАЯ ОШИБКА: NewConfigHandler вернул nil")
		log.Fatalf("Failed to create configHandler: NewConfigHandler returned nil")
	}
	log.Printf("✓ ConfigHandler успешно создан")

	srv := &Server{
		db:                            db,
		normalizedDB:                  normalizedDB,
		serviceDB:                     serviceDB,
		currentDBPath:                 dbPath,
		currentNormalizedDBPath:       normalizedDBPath,
		config:                        config,
		httpServer:                    nil,
		logChan:                       make(chan LogEntry, config.LogBufferSize),
		nomenclatureProcessor:         nil,
		normalizer:                    normalizer,
		normalizerEvents:              normalizerEvents,
		normalizerRunning:             false,
		shutdownChan:                  make(chan struct{}),
		startTime:                     time.Now(),
		qualityAnalyzer:               qualityAnalyzer,
		workerConfigManager:           workerConfigManager,
		arliaiClient:                  arliaiClient,
		arliaiCache:                   arliaiCache,
		openrouterClient:              openrouterClient,
		huggingfaceClient:             huggingfaceClient,
		multiProviderClient:           multiProviderClient,
		similarityCache:               similarityCache,
		hierarchicalClassifier:        hierarchicalClassifier,
		kpvedCurrentTasks:             make(map[int]*classificationTask),
		kpvedWorkersStopped:           false,
		enrichmentFactory:             enrichmentFactory,
		monitoringManager:             monitoringManager,
		providerOrchestrator:          providerOrchestrator,
		dbInfoCache:                   dbInfoCache,
		systemSummaryCache:            systemSummaryCache,
		scanHistoryManager:            scanHistoryManager,
		dbModificationTracker:         dbModificationTracker,
		dbConnectionCache:             dbConnectionCache,
		normalizationService:          normalizationService,
		counterpartyService:           counterpartyService,
		uploadService:                 uploadService,
		uploadHandler:                 uploadHandler,
		clientService:                 clientService,
		clientHandler:                 clientHandler,
		databaseService:               databaseService,
		normalizationHandler:          normalizationHandler,
		qualityService:                qualityService,
		qualityHandler:                qualityHandler,
		classificationService:         classificationService,
		classificationHandler:         classificationHandler,
		counterpartyHandler:           counterpartyHandler,
		similarityService:             similarityService,
		similarityHandler:             similarityHandler,
		databaseHandler:               databaseHandler,
		nomenclatureHandler:           nomenclatureHandler,
		dashboardHandler:              dashboardHandler,
		gispHandler:                   gispHandler,
		gostHandler:                   gostHandler,
		benchmarkHandler:              benchmarkHandler,
		processing1CHandler:           processing1CHandler,
		duplicateDetectionHandler:     duplicateDetectionHandler,
		patternDetectionHandler:       patternDetectionHandler,
		reclassificationHandler:       reclassificationHandler,
		normalizationBenchmarkHandler: normalizationBenchmarkHandler,
		diagnosticsHandler:            diagnosticsHandler, // Будет инициализирован после создания Server
		monitoringService:             monitoringService,
		dashboardService:              container.DashboardService,
		monitoringHandler:             monitoringHandler,
		reportService:                 reportService,
		reportHandler:                 reportHandler,
		snapshotService:               snapshotService,
		snapshotHandler:               snapshotHandler,
		workerTraceHandler:            workerTraceHandler,
		notificationService:           notificationService,
		notificationHandler:           notificationHandler,
		configHandler:                 configHandler,
		errorMetricsHandler:           errorMetricsHandler,
		systemHandler:                 systemHandler,
		systemSummaryHandler:          systemSummaryHandler,
		uploadLegacyHandler:           uploadLegacyHandler,
		logsHandler:                   container.LogsHandler,
		healthChecker:                 healthChecker,
		metricsCollector:              metricsCollector,
		container:                     container,
	}

	// Инициализируем diagnostics handler после создания Server (требует Server в качестве параметра)
	srv.diagnosticsHandler = handlers.NewDiagnosticsHandler(srv)

	// Валидация критических зависимостей перед возвратом
	if err := srv.validateCriticalDependencies(); err != nil {
		log.Fatalf("Failed to validate critical dependencies: %v", err)
	}

	return srv
}

// validateCriticalDependencies проверяет, что все критические зависимости инициализированы
func (s *Server) validateCriticalDependencies() error {
	var missing []string

	if s.serviceDB == nil {
		missing = append(missing, "serviceDB")
	}
	if s.config == nil {
		missing = append(missing, "config")
	}
	if s.configHandler == nil {
		missing = append(missing, "configHandler")
	}
	if s.normalizationHandler == nil {
		missing = append(missing, "normalizationHandler")
	}
	if s.databaseHandler == nil {
		missing = append(missing, "databaseHandler")
	}

	if len(missing) > 0 {
		return fmt.Errorf("critical dependencies are nil: %v", missing)
	}

	log.Printf("✓ Все критические зависимости валидированы успешно")
	return nil
}
