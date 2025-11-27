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
	"httpserver/internal/config"
	"httpserver/internal/infrastructure/ai"
	"httpserver/internal/infrastructure/cache"
	"httpserver/normalization"
	"httpserver/normalization/algorithms"
	"httpserver/quality"
	"httpserver/server/handlers"
	"httpserver/server/middleware"
	"httpserver/server/monitoring"
	"httpserver/server/services"
	"httpserver/websearch"
	"golang.org/x/time/rate"
)

// Container контейнер зависимостей для сервера
// Инкапсулирует всю логику инициализации зависимостей
type Container struct {
	// Конфигурация
	Config *config.Config

	// Базы данных
	DB            *database.DB
	NormalizedDB  *database.DB
	ServiceDB     *database.ServiceDB
	BenchmarksDB  *database.BenchmarksDB
	GostsDB       *database.GostsDB
	DBPath        string
	NormalizedDBPath string

	// AI клиенты
	ArliaiClient      *ai.ArliaiClient
	OpenRouterClient  *ai.OpenRouterClient
	HuggingFaceClient *ai.HuggingFaceClient
	EdenAIClient      *ai.EdenAIClient

	// Кэши
	ArliaiCache          *cache.ArliaiCache
	DatabaseInfoCache    *cache.DatabaseInfoCache
	SystemSummaryCache   *cache.SystemSummaryCache
	DatabaseConnectionCache *cache.DatabaseConnectionCache
	SimilarityCache      *algorithms.OptimizedHybridSimilarity

	// Нормализация
	Normalizer        *normalization.Normalizer
	NormalizerEvents  chan string
	QualityAnalyzer   *quality.QualityAnalyzer

	// Менеджеры
	WorkerConfigManager   interface{} // *WorkerConfigManager (из server пакета)
	MonitoringManager     interface{} // *MonitoringManager (из server пакета)
	ProviderOrchestrator   interface{} // *ProviderOrchestrator (из server пакета)
	ScanHistoryManager     interface{} // *ScanHistoryManager (из server пакета)
	DatabaseModificationTracker interface{} // *DatabaseModificationTracker (из server пакета)

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
	BaseHandler                    *handlers.BaseHandler
	UploadHandler                  *handlers.UploadHandler
	ClientHandler                  *handlers.ClientHandler
	NormalizationHandler           *handlers.NormalizationHandler
	QualityHandler                 *handlers.QualityHandler
	ClassificationHandler          *handlers.ClassificationHandler
	CounterpartyHandler            *handlers.CounterpartyHandler
	SimilarityHandler              *handlers.SimilarityHandler
	DatabaseHandler                *handlers.DatabaseHandler
	NomenclatureHandler            *handlers.NomenclatureHandler
	DashboardHandler               *handlers.DashboardHandler
	GISPHandler                    *handlers.GISPHandler
	GostHandler                    *handlers.GostHandler
	BenchmarkHandler               *handlers.BenchmarkHandler
	Processing1CHandler           *handlers.Processing1CHandler
	DuplicateDetectionHandler     *handlers.DuplicateDetectionHandler
	PatternDetectionHandler        *handlers.PatternDetectionHandler
	ReclassificationHandler        *handlers.ReclassificationHandler
	NormalizationBenchmarkHandler  *handlers.NormalizationBenchmarkHandler
	MonitoringHandler              *handlers.MonitoringHandler
	ReportHandler                  *handlers.ReportHandler
	SnapshotHandler                *handlers.SnapshotHandler
	WorkerHandler                  *handlers.WorkerHandler
	WorkerTraceHandler             *handlers.WorkerTraceHandler
	NotificationHandler            *handlers.NotificationHandler
	ErrorMetricsHandler            *handlers.ErrorMetricsHandler
	WebSearchHandler                *handlers.WebSearchHandler

	// Мониторинг
	HealthChecker    *monitoring.HealthChecker
	MetricsCollector *monitoring.MetricsCollector
}

// NewContainer создает новый контейнер зависимостей
// Принимает уже инициализированные базы данных и конфигурацию
func NewContainer(
	db *database.DB,
	normalizedDB *database.DB,
	serviceDB *database.ServiceDB,
	dbPath, normalizedDBPath string,
	cfg *config.Config,
) (*Container, error) {
	container := &Container{
		Config:            cfg,
		DB:                db,
		NormalizedDB:      normalizedDB,
		ServiceDB:         serviceDB,
		DBPath:            dbPath,
		NormalizedDBPath:  normalizedDBPath,
	}

	// Инициализируем компоненты в правильном порядке
	if err := container.InitCaches(); err != nil {
		return nil, fmt.Errorf("failed to init caches: %w", err)
	}

	if err := container.InitAIClients(); err != nil {
		return nil, fmt.Errorf("failed to init AI clients: %w", err)
	}

	if err := container.InitNormalization(); err != nil {
		return nil, fmt.Errorf("failed to init normalization: %w", err)
	}

	if err := container.InitManagers(); err != nil {
		return nil, fmt.Errorf("failed to init managers: %w", err)
	}

	if err := container.InitServices(); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	if err := container.InitHandlers(); err != nil {
		return nil, fmt.Errorf("failed to init handlers: %w", err)
	}

	if err := container.InitMonitoring(); err != nil {
		return nil, fmt.Errorf("failed to init monitoring: %w", err)
	}

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

	// Создаем нормализатор
	c.Normalizer = normalization.NewNormalizer(c.DB, c.NormalizerEvents, aiConfig)

	// Создаем анализатор качества
	c.QualityAnalyzer = quality.NewQualityAnalyzer(c.DB)

	return nil
}

// InitManagers инициализирует менеджеры
// ВАЖНО: Этот метод должен быть реализован в server пакете, так как использует типы из server
// Пока оставляем заглушку
func (c *Container) InitManagers() error {
	// TODO: Реализовать инициализацию менеджеров
	// Это требует доступа к типам из server пакета
	// Возможно, нужно будет создать интерфейсы или переместить менеджеры в отдельный пакет
	return nil
}

// InitServices инициализирует все сервисы
func (c *Container) InitServices() error {
	// Создаем БД эталонов
	benchmarksDBPath := filepath.Join("data", "benchmarks.db")
	benchmarksDB, err := database.NewBenchmarksDB(benchmarksDBPath)
	if err != nil {
		return fmt.Errorf("failed to create benchmarks database: %w", err)
	}
	c.BenchmarksDB = benchmarksDB

	// Создаем benchmark service
	c.BenchmarkService = services.NewBenchmarkService(benchmarksDB, c.DB, c.ServiceDB)

	// Устанавливаем BenchmarkFinder в normalizer
	if c.BenchmarkService != nil {
		benchmarkFinderAdapter := &services.BenchmarkFinderAdapter{BenchmarkService: c.BenchmarkService}
		c.Normalizer.SetBenchmarkFinder(benchmarkFinderAdapter)
	}

	// Создаем сервисы
	c.NormalizationService = services.NewNormalizationService(
		c.DB, c.ServiceDB, c.Normalizer, c.BenchmarkService, c.NormalizerEvents,
	)

	c.CounterpartyService = services.NewCounterpartyService(
		c.ServiceDB, c.NormalizerEvents, c.BenchmarkService,
	)

	// UploadService будет создан в InitHandlers, так как требует logFunc

	// ClientService
	clientService, err := services.NewClientService(c.ServiceDB, c.DB, c.NormalizedDB)
	if err != nil {
		return fmt.Errorf("failed to create client service: %w", err)
	}
	c.ClientService = clientService

	// DatabaseService
	c.DatabaseService = services.NewDatabaseService(
		c.ServiceDB, c.DB, c.NormalizedDB, c.DBPath, c.NormalizedDBPath, c.DatabaseInfoCache,
	)

	// QualityService
	qualityService, err := services.NewQualityService(c.DB, c.QualityAnalyzer)
	if err != nil {
		return fmt.Errorf("failed to create quality service: %w", err)
	}
	c.QualityService = qualityService

	// ClassificationService
	var getAPIKey func() string
	if c.WorkerConfigManager != nil {
		// WorkerConfigManager имеет тип interface{}, нужно проверить наличие метода GetModelAndAPIKey
		type APIKeyGetter interface {
			GetModelAndAPIKey() (string, string, error)
		}
		if manager, ok := c.WorkerConfigManager.(APIKeyGetter); ok {
			getAPIKey = func() string {
				apiKey, _, err := manager.GetModelAndAPIKey()
				if err == nil && apiKey != "" {
					return apiKey
				}
				return os.Getenv("ARLIAI_API_KEY")
			}
		}
	}
	c.ClassificationService = services.NewClassificationService(
		c.DB, c.NormalizedDB, c.ServiceDB, func() string {
			return "GLM-4.5-Air" // Дефолтная модель, будет обновлена в Start()
		},
		getAPIKey,
	)

	// SimilarityService
	c.SimilarityService = services.NewSimilarityService(c.SimilarityCache)

	// NotificationService
	if c.ServiceDB == nil {
		return fmt.Errorf("ServiceDB is required for NotificationService but is nil")
	}
	c.NotificationService = services.NewNotificationService(c.ServiceDB)

	// MonitoringService (требует normalizer adapter, будет создан в InitHandlers)
	// ReportService
	c.ReportService = services.NewReportService(c.DB, c.NormalizedDB, c.ServiceDB)

	// SnapshotService
	c.SnapshotService = services.NewSnapshotService(c.DB)

	// NomenclatureService (требует workerConfigManager и processor getter/setter, будет создан в InitHandlers)
	// DashboardService (требует функции для получения метрик, будет создан в InitHandlers)
	// GISPService
	c.GISPService = services.NewGISPService(c.ServiceDB)

	// GostService (требует gostsDB, будет создан в InitHandlers)
	// Processing1CService
	c.Processing1CService = services.NewProcessing1CService()

	// DuplicateDetectionService
	c.DuplicateDetectionService = services.NewDuplicateDetectionService()

	// PatternDetectionService (требует getArliaiAPIKey, будет создан в InitHandlers)
	// ReclassificationService
	c.ReclassificationService = services.NewReclassificationService()

	// NormalizationBenchmarkService
	c.NormalizationBenchmarkService = services.NewNormalizationBenchmarkService()

	// GostsDB (опционально, может не инициализироваться)
	gostsDB, err := database.NewGostsDB("gosts.db")
	if err != nil {
		log.Printf("Warning: failed to initialize GOSTs database: %v", err)
	} else {
		c.GostsDB = gostsDB
		c.GostService = services.NewGostService(gostsDB)
	}

	// WorkerService, MonitoringService, NomenclatureService, DashboardService, PatternDetectionService
	// требуют менеджеров или функций, которые будут установлены в InitHandlers или Start()

	return nil
}

// InitHandlers инициализирует все handlers
func (c *Container) InitHandlers() error {
	// Создаем базовый handler
	c.BaseHandler = handlers.NewBaseHandlerFromMiddleware()

	// Создаем handlers (многие требуют logFunc, который будет установлен в Start())
	// Пока создаем с заглушками

	// CounterpartyHandler
	c.CounterpartyHandler = handlers.NewCounterpartyHandler(
		c.BaseHandler, c.CounterpartyService, func(entry interface{}) {
			// logFunc будет установлен в Start()
		},
	)
	c.CounterpartyHandler.SetExportManager(handlers.NewDefaultCounterpartyExportManager())

	// ClientHandler
	c.ClientHandler = handlers.NewClientHandler(c.ClientService, c.BaseHandler)

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

	// WebSearchHandler
	if c.Config.WebSearch != nil && c.Config.WebSearch.Enabled {
		// Создаем кэш для веб-поиска
		cacheConfig := &websearch.CacheConfig{
			Enabled:         c.Config.WebSearch.CacheEnabled,
			TTL:             c.Config.WebSearch.CacheTTL,
			CleanupInterval: c.Config.WebSearch.CacheTTL / 4,
			MaxSize:         1000, // По умолчанию
		}
		searchCache := websearch.NewCache(cacheConfig)

		// Создаем клиент веб-поиска
		// Преобразуем RateLimitPerSec в rate.Limit
		rateLimit := rate.Every(time.Duration(1000/c.Config.WebSearch.RateLimitPerSec) * time.Millisecond)
		clientConfig := websearch.ClientConfig{
			BaseURL:    c.Config.WebSearch.BaseURL,
			Timeout:    c.Config.WebSearch.Timeout,
			RateLimit:  rateLimit,
			Cache:      searchCache,
		}
		searchClient := websearch.NewClient(clientConfig)

		// Создаем handler
		c.WebSearchHandler = handlers.NewWebSearchHandler(c.BaseHandler, searchClient)
	}

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
	c.HealthChecker = monitoring.NewHealthChecker(version, mainDBConn, c.ServiceDB)
	c.MetricsCollector = monitoring.NewMetricsCollector()

	return nil
}

