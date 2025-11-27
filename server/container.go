package server

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"httpserver/database"
	"httpserver/enrichment"
	serverconfig "httpserver/internal/config"
	"httpserver/internal/infrastructure/ai"
	"httpserver/internal/infrastructure/cache"
	"httpserver/internal/infrastructure/monitoring"
	"httpserver/internal/infrastructure/workers"
	"httpserver/nomenclature"
	"httpserver/normalization"
	"httpserver/normalization/algorithms"
	"httpserver/quality"
	"httpserver/server/handlers"
	"httpserver/server/middleware"
	servermonitoring "httpserver/server/monitoring"
	"httpserver/server/services"
)

// Container контейнер зависимостей для сервера
// Инкапсулирует всю логику инициализации зависимостей
type Container struct {
	// Конфигурация
	Config *serverconfig.Config

	// Базы данных
	DB               *database.DB
	NormalizedDB     *database.DB
	ServiceDB        *database.ServiceDB
	BenchmarksDB     *database.BenchmarksDB
	GostsDB          *database.GostsDB
	DBPath           string
	NormalizedDBPath string

	// AI клиенты
	ArliaiClient      *ai.ArliaiClient
	OpenRouterClient  *ai.OpenRouterClient
	HuggingFaceClient *ai.HuggingFaceClient
	EdenAIClient      *ai.EdenAIClient

	// Кэши
	ArliaiCache             *cache.ArliaiCache
	DatabaseInfoCache       *cache.DatabaseInfoCache
	SystemSummaryCache      *cache.SystemSummaryCache
	DatabaseConnectionCache *cache.DatabaseConnectionCache
	SimilarityCache         *algorithms.OptimizedHybridSimilarity

	// Нормализация
	Normalizer       *normalization.Normalizer
	NormalizerEvents chan string
	QualityAnalyzer  *quality.QualityAnalyzer

	// Менеджеры
	WorkerConfigManager         interface{}
	MonitoringManager           interface{}
	ProviderOrchestrator        interface{}
	ScanHistoryManager          interface{}
	DatabaseModificationTracker interface{}

	// Обогащение
	EnrichmentFactory *enrichment.EnricherFactory

	// Классификация
	HierarchicalClassifier *normalization.HierarchicalClassifier

	// Сервисы
	NormalizationService          *services.NormalizationService
	CounterpartyService           *services.CounterpartyService
	UploadService                 *services.UploadService
	ClientService                 *services.ClientService
	DatabaseService               *services.DatabaseService
	QualityService                *services.QualityService
	ClassificationService         *services.ClassificationService
	SimilarityService             *services.SimilarityService
	MonitoringService             *services.MonitoringService
	ReportService                 *services.ReportService
	SnapshotService               *services.SnapshotService
	WorkerService                 *services.WorkerService
	NotificationService           *services.NotificationService
	NomenclatureService           *services.NomenclatureService
	DashboardService              *services.DashboardService
	GISPService                   *services.GISPService
	GostService                   *services.GostService
	Processing1CService           *services.Processing1CService
	DuplicateDetectionService     *services.DuplicateDetectionService
	PatternDetectionService       *services.PatternDetectionService
	ReclassificationService       *services.ReclassificationService
	NormalizationBenchmarkService *services.NormalizationBenchmarkService
	BenchmarkService              *services.BenchmarkService

	// Handlers
	BaseHandler                   *handlers.BaseHandler
	UploadHandler                 *handlers.UploadHandler
	ClientHandler                 *handlers.ClientHandler
	NormalizationHandler          *handlers.NormalizationHandler
	QualityHandler                *handlers.QualityHandler
	ClassificationHandler         *handlers.ClassificationHandler
	CounterpartyHandler           *handlers.CounterpartyHandler
	SimilarityHandler             *handlers.SimilarityHandler
	DatabaseHandler               *handlers.DatabaseHandler
	NomenclatureHandler           *handlers.NomenclatureHandler
	DashboardHandler              *handlers.DashboardHandler
	GISPHandler                   *handlers.GISPHandler
	GostHandler                   *handlers.GostHandler
	BenchmarkHandler              *handlers.BenchmarkHandler
	Processing1CHandler           *handlers.Processing1CHandler
	DuplicateDetectionHandler     *handlers.DuplicateDetectionHandler
	PatternDetectionHandler       *handlers.PatternDetectionHandler
	ReclassificationHandler       *handlers.ReclassificationHandler
	NormalizationBenchmarkHandler *handlers.NormalizationBenchmarkHandler
	MonitoringHandler             *handlers.MonitoringHandler
	ReportHandler                 *handlers.ReportHandler
	SnapshotHandler               *handlers.SnapshotHandler
	WorkerHandler                 *handlers.WorkerHandler
	WorkerTraceHandler            *handlers.WorkerTraceHandler
	NotificationHandler           *handlers.NotificationHandler
	ErrorMetricsHandler           *handlers.ErrorMetricsHandler
	SystemHandler                 *handlers.SystemHandler
	SystemSummaryHandler          *handlers.SystemSummaryHandler
	LogsHandler                   *handlers.LogsHandler

	// Мониторинг
	HealthChecker    *servermonitoring.HealthChecker
	MetricsCollector *servermonitoring.MetricsCollector
}

// NewContainer создает новый контейнер зависимостей
// Принимает уже инициализированные базы данных и конфигурацию
func NewContainer(
	db *database.DB,
	normalizedDB *database.DB,
	serviceDB *database.ServiceDB,
	dbPath, normalizedDBPath string,
	cfg *serverconfig.Config,
) (*Container, error) {
	container := &Container{
		Config:           cfg,
		DB:               db,
		NormalizedDB:     normalizedDB,
		ServiceDB:        serviceDB,
		DBPath:           dbPath,
		NormalizedDBPath: normalizedDBPath,
	}

	// Инициализируем компоненты в правильном порядке
	log.Printf("Инициализация кэшей...")
	if err := container.InitCaches(); err != nil {
		log.Printf("✗ Ошибка инициализации кэшей: %v", err)
		return nil, fmt.Errorf("failed to init caches: %w", err)
	}
	log.Printf("✓ Кэши инициализированы")

	log.Printf("Инициализация AI клиентов...")
	if err := container.InitAIClients(); err != nil {
		log.Printf("✗ Ошибка инициализации AI клиентов: %v", err)
		return nil, fmt.Errorf("failed to init AI clients: %w", err)
	}
	log.Printf("✓ AI клиенты инициализированы")

	log.Printf("Инициализация компонентов нормализации...")
	if err := container.InitNormalization(); err != nil {
		log.Printf("✗ Ошибка инициализации нормализации: %v", err)
		return nil, fmt.Errorf("failed to init normalization: %w", err)
	}
	log.Printf("✓ Компоненты нормализации инициализированы")

	log.Printf("Инициализация менеджеров...")
	if err := container.InitManagers(); err != nil {
		log.Printf("✗ Ошибка инициализации менеджеров: %v", err)
		return nil, fmt.Errorf("failed to init managers: %w", err)
	}
	log.Printf("✓ Менеджеры инициализированы")

	log.Printf("Инициализация сервисов...")
	if err := container.InitServices(); err != nil {
		log.Printf("✗ Ошибка инициализации сервисов: %v", err)
		return nil, fmt.Errorf("failed to init services: %w", err)
	}
	log.Printf("✓ Сервисы инициализированы")

	log.Printf("Инициализация обработчиков...")
	if err := container.InitHandlers(); err != nil {
		log.Printf("✗ Ошибка инициализации обработчиков: %v", err)
		return nil, fmt.Errorf("failed to init handlers: %w", err)
	}
	log.Printf("✓ Обработчики инициализированы")

	log.Printf("Инициализация мониторинга...")
	if err := container.InitMonitoring(); err != nil {
		log.Printf("✗ Ошибка инициализации мониторинга: %v", err)
		return nil, fmt.Errorf("failed to init monitoring: %w", err)
	}
	log.Printf("✓ Мониторинг инициализирован")

	log.Printf("✓ Все компоненты контейнера успешно инициализированы")
	return container, nil
}

// InitCaches инициализирует все кэши
func (c *Container) InitCaches() error {
	c.ArliaiCache = cache.NewArliaiCache()
	c.DatabaseInfoCache = cache.NewDatabaseInfoCache()
	c.SystemSummaryCache = cache.NewSystemSummaryCache(2 * time.Minute)
	c.DatabaseConnectionCache = cache.NewDatabaseConnectionCache()
	c.SimilarityCache = algorithms.NewOptimizedHybridSimilarity(nil, 10000)
	return nil
}

// InitAIClients инициализирует AI клиенты
func (c *Container) InitAIClients() error {
	c.ArliaiClient = ai.NewArliaiClient()

	openrouterAPIKey := os.Getenv("OPENROUTER_API_KEY")
	c.OpenRouterClient = ai.NewOpenRouterClient(openrouterAPIKey)

	huggingfaceAPIKey := os.Getenv("HUGGINGFACE_API_KEY")
	c.HuggingFaceClient = ai.NewHuggingFaceClient(huggingfaceAPIKey, "https://api-inference.huggingface.co")

	edenaiAPIKey := os.Getenv("EDENAI_API_KEY")
	edenaiBaseURL := os.Getenv("EDENAI_BASE_URL")
	if edenaiBaseURL == "" {
		edenaiBaseURL = "https://api.edenai.run/v2"
	}
	c.EdenAIClient = ai.NewEdenAIClient(edenaiAPIKey, edenaiBaseURL)

	return nil
}

// InitNormalization инициализирует компоненты нормализации
func (c *Container) InitNormalization() error {
	// Создаем канал событий для нормализатора
	c.NormalizerEvents = make(chan string, c.Config.NormalizerEventsBufferSize)

	// Инициализируем AI конфигурацию для нормализатора
	aiConfig := &normalization.AIConfig{
		Enabled:        true,
		MinConfidence:  0.7,
		RateLimitDelay: 100 * time.Millisecond,
		MaxRetries:     3,
	}

	// Создаем функцию получения API ключа из конфигурации воркеров
	// Используем замыкание на контейнер, чтобы получить доступ к WorkerConfigManager после его инициализации
	getAPIKey := func() string {
		if c.WorkerConfigManager != nil {
			if manager, ok := c.WorkerConfigManager.(*workers.WorkerConfigManager); ok {
				apiKey, _, err := manager.GetModelAndAPIKey()
				if err == nil && apiKey != "" {
					return apiKey
				}
			}
		}
		// Fallback на переменную окружения
		return os.Getenv("ARLIAI_API_KEY")
	}

	// Создаем нормализатор с функцией получения API ключа
	c.Normalizer = normalization.NewNormalizerWithStopCheck(c.DB, c.NormalizerEvents, aiConfig, nil, getAPIKey)

	// Создаем анализатор качества
	c.QualityAnalyzer = quality.NewQualityAnalyzer(c.DB)
	// No manager imports need to be updated as they are correctly referenced

	return nil
}

// InitManagers инициализирует менеджеры
func (c *Container) InitManagers() error {
	// Создаем менеджер конфигурации воркеров
	workerConfigManager := workers.NewWorkerConfigManager(c.ServiceDB)
	c.WorkerConfigManager = workerConfigManager

	// Создаем менеджер мониторинга провайдеров
	monitoringManager := monitoring.NewManager()
	monitoringManager.StartHistoryProcessor()
	c.MonitoringManager = monitoringManager

	// Создаем оркестратор провайдеров
	orchestratorTimeout := 30 * time.Second
	if c.Config.AITimeout > 0 {
		orchestratorTimeout = c.Config.AITimeout
	}
	providerOrchestrator := ai.NewProviderOrchestrator(orchestratorTimeout, monitoringManager)
	if c.Config.AggregationStrategy != "" {
		providerOrchestrator.SetStrategy(ai.AggregationStrategy(c.Config.AggregationStrategy))
	}
	c.ProviderOrchestrator = providerOrchestrator

	// Создаем менеджер истории сканирований
	historyDBPath := filepath.Join("data", "scan_history.db")
	scanHistoryManager, err := cache.NewScanHistoryManager(historyDBPath)
	if err != nil {
		log.Printf("Предупреждение: не удалось создать менеджер истории сканирований: %v", err)
		scanHistoryManager = nil
	}
	c.ScanHistoryManager = scanHistoryManager

	// Создаем трекер изменений БД
	c.DatabaseModificationTracker = cache.NewDatabaseModificationTracker()

	// Инициализируем фабрику обогатителей
	if c.Config.Enrichment != nil && c.Config.Enrichment.Enabled {
		c.EnrichmentFactory = enrichment.NewEnricherFactory(c.Config.Enrichment.Services)
		log.Printf("Enrichment factory initialized with %d services", len(c.Config.Enrichment.Services))
	}

	// Инициализируем KPVED hierarchical classifier
	apiKey := os.Getenv("ARLIAI_API_KEY")
	model := os.Getenv("ARLIAI_MODEL")
	if model == "" {
		model = "GLM-4.5-Air"
	}
	if apiKey != "" && c.ServiceDB != nil {
		aiClient := nomenclature.NewAIClient(apiKey, model)
		var err error
		c.HierarchicalClassifier, err = normalization.NewHierarchicalClassifier(c.ServiceDB, aiClient)
		if err != nil {
			log.Printf("Warning: Failed to initialize KPVED classifier: %v", err)
			c.HierarchicalClassifier = nil
		} else {
			log.Printf("KPVED hierarchical classifier initialized successfully")
		}
	}

	return nil
}

// InitServices инициализирует все сервисы
func (c *Container) InitServices() error {
	// Создаем БД эталонов
	benchmarksDBPath := filepath.Join("data", "benchmarks.db")
	log.Printf("  Создание БД эталонов: %s", benchmarksDBPath)
	
	// Проверяем/создаем папку data перед созданием БД
	dataDir := filepath.Dir(benchmarksDBPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("✗ Не удалось создать папку %s: %v", dataDir, err)
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	log.Printf("  Папка %s проверена/создана", dataDir)
	
	benchmarksDB, err := database.NewBenchmarksDB(benchmarksDBPath)
	if err != nil {
		log.Printf("✗ Ошибка создания БД эталонов: %v", err)
		log.Printf("  Путь: %s", benchmarksDBPath)
		return fmt.Errorf("failed to create benchmarks database: %w", err)
	}
	c.BenchmarksDB = benchmarksDB
	log.Printf("  ✓ БД эталонов создана: %s", benchmarksDBPath)

	// Создаем benchmark service
	log.Printf("  Создание BenchmarkService...")
	c.BenchmarkService = services.NewBenchmarkService(benchmarksDB, c.DB, c.ServiceDB)
	log.Printf("  ✓ BenchmarkService создан")

	// Устанавливаем BenchmarkFinder в normalizer
	if c.BenchmarkService != nil {
		benchmarkFinderAdapter := &services.BenchmarkFinderAdapter{BenchmarkService: c.BenchmarkService}
		c.Normalizer.SetBenchmarkFinder(benchmarkFinderAdapter)
	}

	// Создаем сервисы
	log.Printf("  Создание NormalizationService...")
	c.NormalizationService = services.NewNormalizationService(
		c.DB, c.ServiceDB, c.Normalizer, c.BenchmarkService, c.NormalizerEvents,
	)
	log.Printf("  ✓ NormalizationService создан")

	log.Printf("  Создание CounterpartyService...")
	c.CounterpartyService = services.NewCounterpartyService(
		c.ServiceDB, c.NormalizerEvents, c.BenchmarkService,
	)
	log.Printf("  ✓ CounterpartyService создан")

	// UploadService будет создан в InitHandlers, так как требует logFunc

	// ClientService
	log.Printf("  Создание ClientService...")
	clientService, err := services.NewClientService(c.ServiceDB, c.DB, c.NormalizedDB)
	if err != nil {
		log.Printf("✗ Ошибка создания ClientService: %v", err)
		log.Printf("  ServiceDB: %v", c.ServiceDB != nil)
		log.Printf("  DB: %v", c.DB != nil)
		log.Printf("  NormalizedDB: %v", c.NormalizedDB != nil)
		return fmt.Errorf("failed to create client service: %w", err)
	}
	c.ClientService = clientService
	log.Printf("  ✓ ClientService создан")

	// DatabaseService
	log.Printf("  Создание DatabaseService...")
	c.DatabaseService = services.NewDatabaseService(
		c.ServiceDB, c.DB, c.NormalizedDB, c.DBPath, c.NormalizedDBPath, c.DatabaseInfoCache,
	)
	log.Printf("  ✓ DatabaseService создан")

	// QualityService
	log.Printf("  Создание QualityService...")
	qualityService, err := services.NewQualityService(c.DB, c.QualityAnalyzer)
	if err != nil {
		log.Printf("✗ Ошибка создания QualityService: %v", err)
		log.Printf("  DB: %v", c.DB != nil)
		log.Printf("  QualityAnalyzer: %v", c.QualityAnalyzer != nil)
		return fmt.Errorf("failed to create quality service: %w", err)
	}
	c.QualityService = qualityService
	log.Printf("  ✓ QualityService создан")

	// ClassificationService
	log.Printf("  Создание ClassificationService...")
	// Создаем функцию получения API ключа из конфигурации воркеров
	getAPIKeyForClassification := func() string {
		if c.WorkerConfigManager != nil {
			if manager, ok := c.WorkerConfigManager.(*workers.WorkerConfigManager); ok {
				apiKey, _, err := manager.GetModelAndAPIKey()
				if err == nil && apiKey != "" {
					return apiKey
				}
			}
		}
		// Fallback на переменную окружения
		return os.Getenv("ARLIAI_API_KEY")
	}
	c.ClassificationService = services.NewClassificationService(
		c.DB, c.NormalizedDB, c.ServiceDB, func() string {
			return "GLM-4.5-Air" // Дефолтная модель, будет обновлена в Start()
		},
		getAPIKeyForClassification,
	)
	log.Printf("  ✓ ClassificationService создан")

	// SimilarityService
	log.Printf("  Создание SimilarityService...")
	c.SimilarityService = services.NewSimilarityService(c.SimilarityCache)
	log.Printf("  ✓ SimilarityService создан")

	// NotificationService
	log.Printf("  Создание NotificationService...")
	if c.ServiceDB == nil {
		return fmt.Errorf("ServiceDB is required for NotificationService but is nil")
	}
	c.NotificationService = services.NewNotificationService(c.ServiceDB)
	log.Printf("  ✓ NotificationService создан")

	// MonitoringService (требует normalizer adapter, будет создан в InitHandlers)
	// ReportService
	log.Printf("  Создание ReportService...")
	c.ReportService = services.NewReportService(c.DB, c.NormalizedDB, c.ServiceDB)
	log.Printf("  ✓ ReportService создан")

	// SnapshotService
	log.Printf("  Создание SnapshotService...")
	c.SnapshotService = services.NewSnapshotService(c.DB)
	log.Printf("  ✓ SnapshotService создан")

	// NomenclatureService (требует workerConfigManager и processor getter/setter, будет создан в InitHandlers)
	// DashboardService (требует функции для получения метрик, будет создан в InitHandlers)
	// GISPService
	log.Printf("  Создание GISPService...")
	c.GISPService = services.NewGISPService(c.ServiceDB)
	log.Printf("  ✓ GISPService создан")

	// GostService (требует gostsDB, будет создан в InitHandlers)
	// Processing1CService
	log.Printf("  Создание Processing1CService...")
	c.Processing1CService = services.NewProcessing1CService()
	log.Printf("  ✓ Processing1CService создан")

	// DuplicateDetectionService
	log.Printf("  Создание DuplicateDetectionService...")
	c.DuplicateDetectionService = services.NewDuplicateDetectionService()
	log.Printf("  ✓ DuplicateDetectionService создан")

	// PatternDetectionService (требует getArliaiAPIKey, будет создан в InitHandlers)
	// ReclassificationService
	log.Printf("  Создание ReclassificationService...")
	c.ReclassificationService = services.NewReclassificationService()
	log.Printf("  ✓ ReclassificationService создан")

	// NormalizationBenchmarkService
	log.Printf("  Создание NormalizationBenchmarkService...")
	c.NormalizationBenchmarkService = services.NewNormalizationBenchmarkService()
	log.Printf("  ✓ NormalizationBenchmarkService создан")

	// GostsDB (опционально, может не инициализироваться)
	log.Printf("  Инициализация GOSTs базы данных...")
	gostsDB, err := database.NewGostsDB("gosts.db")
	if err != nil {
		log.Printf("  ⚠ Предупреждение: не удалось инициализировать GOSTs базу данных: %v", err)
		log.Printf("  GOSTs функциональность будет недоступна")
	} else {
		c.GostsDB = gostsDB
		c.GostService = services.NewGostService(gostsDB)
		log.Printf("  ✓ GOSTs база данных и сервис инициализированы")
	}

	// WorkerService, MonitoringService, NomenclatureService, DashboardService, PatternDetectionService
	// требуют менеджеров или функций, которые будут установлены в InitHandlers или Start()

	return nil
}

// InitHandlers инициализирует все handlers
func (c *Container) InitHandlers() error {
	log.Printf("  Создание базового handler...")
	// Создаем базовый handler
	c.BaseHandler = handlers.NewBaseHandlerFromMiddleware()
	log.Printf("  ✓ Базовый handler создан")

	// Создаем handlers (многие требуют logFunc, который будет установлен в Start())
	// Пока создаем с заглушками

	// CounterpartyHandler
	log.Printf("  Создание CounterpartyHandler...")
	c.CounterpartyHandler = handlers.NewCounterpartyHandler(
		c.BaseHandler, c.CounterpartyService, func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
	)
	c.CounterpartyHandler.SetExportManager(handlers.NewDefaultCounterpartyExportManager())
	log.Printf("  ✓ CounterpartyHandler создан")

	// ClientHandler
	log.Printf("  Создание ClientHandler...")
	c.ClientHandler = handlers.NewClientHandler(c.ClientService, c.BaseHandler)
	log.Printf("  ✓ ClientHandler создан")

	// NormalizationHandler
	c.NormalizationHandler = handlers.NewNormalizationHandler(
		c.NormalizationService, c.BaseHandler, c.NormalizerEvents,
	)
	c.NormalizationHandler.SetDatabase(c.DB, c.DBPath, c.NormalizedDB, c.NormalizedDBPath)

	// QualityHandler
	c.QualityHandler = handlers.NewQualityHandler(
		c.BaseHandler, c.QualityService, func(entry interface{}) {
			// logFunc будет установлен в Start()
		}, c.NormalizedDB, c.NormalizedDBPath,
	)

	// ClassificationHandler
	c.ClassificationHandler = handlers.NewClassificationHandler(
		c.BaseHandler, c.ClassificationService, func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
	)

	// SimilarityHandler
	c.SimilarityHandler = handlers.NewSimilarityHandler(
		c.BaseHandler, c.SimilarityService, func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
	)

	// DatabaseHandler
	c.DatabaseHandler = handlers.NewDatabaseHandler(c.DatabaseService, c.BaseHandler)

	// ErrorMetricsHandler
	c.ErrorMetricsHandler = handlers.NewErrorMetricsHandler(c.BaseHandler)

	// LogsHandler
	c.LogsHandler = handlers.NewLogsHandler(c.BaseHandler)

	// ReportHandler
	c.ReportHandler = handlers.NewReportHandler(
		c.BaseHandler,
		c.ReportService,
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

	// SnapshotHandler (требует дополнительные функции, будет обновлен в Start())
	c.SnapshotHandler = handlers.NewSnapshotHandler(
		c.BaseHandler,
		c.SnapshotService,
		func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
		c.ServiceDB,
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

	// GISPHandler
	c.GISPHandler = handlers.NewGISPHandler(c.GISPService, c.BaseHandler)

	// Processing1CHandler
	c.Processing1CHandler = handlers.NewProcessing1CHandler(c.Processing1CService, c.BaseHandler)

	// DuplicateDetectionHandler
	c.DuplicateDetectionHandler = handlers.NewDuplicateDetectionHandler(c.DuplicateDetectionService, c.BaseHandler)

	// ReclassificationHandler
	c.ReclassificationHandler = handlers.NewReclassificationHandler(c.ReclassificationService, c.BaseHandler)

	// NormalizationBenchmarkHandler
	c.NormalizationBenchmarkHandler = handlers.NewNormalizationBenchmarkHandler(c.NormalizationBenchmarkService, c.BaseHandler)

	// GostHandler (если GostService создан)
	if c.GostService != nil {
		c.GostHandler = handlers.NewGostHandler(c.GostService)
	}

	// BenchmarkHandler
	c.BenchmarkHandler = handlers.NewBenchmarkHandler(c.BenchmarkService, c.BaseHandler)

	// NotificationHandler
	c.NotificationHandler = handlers.NewNotificationHandler(c.NotificationService, c.BaseHandler)

	// Остальные handlers требуют дополнительных сервисов или функций:
	// - UploadHandler (требует logFunc, будет создан в Start())
	// - MonitoringHandler (требует normalizerAdapter и функции, будет создан в Start())
	// - DashboardHandler (требует DashboardService с функциями, будет создан в Start())
	// - NomenclatureHandler (требует NomenclatureService с processor getter/setter, будет создан в Start())
	// - WorkerHandler (требует WorkerService и множество функций, будет создан в Start())
	// - WorkerTraceHandler (требует logFunc, будет создан в Start())
	// - PatternDetectionHandler (требует PatternDetectionService с getArliaiAPIKey, будет создан в Start())

	return nil
}

// InitMonitoring инициализирует компоненты мониторинга
func (c *Container) InitMonitoring() error {
	// Инициализируем метрики ошибок
	middleware.InitErrorMetrics()

	// Создаем health checker
	version := "1.0.0"
	var mainDBConn *sql.DB
	if c.DB != nil {
		mainDBConn = c.DB.GetDB()
	}
	c.HealthChecker = servermonitoring.NewHealthChecker(version, mainDBConn, c.ServiceDB)
	c.MetricsCollector = servermonitoring.NewMetricsCollector()

	// Создаем SystemHandler
	c.SystemHandler = handlers.NewSystemHandler(
		c.BaseHandler,
		c.DB,
		c.HealthChecker,
		c.MetricsCollector,
	)

	// Создаем SystemSummaryHandler
	c.SystemSummaryHandler = handlers.NewSystemSummaryHandler(
		c.BaseHandler,
		c.SystemSummaryCache,
		c.ScanHistoryManager,
		c.Config,
		c.DBPath,
	)

	return nil
}
