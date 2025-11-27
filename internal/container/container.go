package container

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/internal/config"
	"httpserver/internal/infrastructure/persistence"
	"httpserver/normalization"
	"httpserver/normalization/algorithms"
	"httpserver/server/handlers"
	"httpserver/server/monitoring"
	"httpserver/server/services"

	"golang.org/x/time/rate"

	"httpserver/websearch"
)

// Container контейнер зависимостей для enterprise-архитектуры
// Управляет жизненным циклом всех компонентов приложения
type Container struct {
	mu sync.RWMutex

	// Конфигурация
	Config *config.Config

	// Базы данных
	DB               *database.DB
	NormalizedDB     *database.DB
	ServiceDB        *database.ServiceDB
	CurrentDBPath    string
	NormalizedDBPath string

	// Сервисы (бизнес-логика)
	BenchmarkService          *services.BenchmarkService
	NormalizationService      *services.NormalizationService
	CounterpartyService       *services.CounterpartyService
	UploadService             *services.UploadService
	ClientService             *services.ClientService
	DatabaseService           *services.DatabaseService
	QualityService            *services.QualityService
	ClassificationService     *services.ClassificationService
	SimilarityService         *services.SimilarityService
	MonitoringService         *services.MonitoringService
	ReportService             *services.ReportService
	SnapshotService           *services.SnapshotService
	WorkerService             *services.WorkerService
	NotificationService       *services.NotificationService
	NomenclatureService       *services.NomenclatureService
	GISPService               *services.GISPService
	GostService               *services.GostService
	Processing1CService       *services.Processing1CService
	DuplicateDetectionService *services.DuplicateDetectionService
	PatternDetectionService   *services.PatternDetectionService

	// Обработчики (HTTP handlers)
	UploadHandler *handlers.UploadHandler
	// Новая архитектура Upload Domain (Clean Architecture)
	UploadHandlerV2     interface{} // *upload.Handler из internal/api/handlers/upload
	UploadUseCase       interface{} // *uploadapp.UseCase
	UploadDomainService interface{} // *uploaddomain.Service

	// Новая архитектура Normalization Domain (Clean Architecture)
	NormalizationHandlerV2     interface{} // *normalization.Handler из internal/api/handlers/normalization
	NormalizationUseCase       interface{} // *normalizationapp.UseCase
	NormalizationDomainService interface{} // *normalizationdomain.Service

	// Новая архитектура Quality Domain (Clean Architecture)
	QualityHandlerV2     interface{} // *quality.Handler из internal/api/handlers/quality
	QualityUseCase       interface{} // *qualityapp.UseCase
	QualityDomainService interface{} // *qualitydomain.Service

	// Новая архитектура Classification Domain (Clean Architecture)
	ClassificationHandlerV2     interface{} // *classification.Handler из internal/api/handlers/classification
	ClassificationUseCase       interface{} // *classificationapp.UseCase
	ClassificationDomainService interface{} // *classificationdomain.Service

	// Новая архитектура Client Domain (Clean Architecture)
	ClientHandlerV2     interface{} // *client.Handler из internal/api/handlers/client
	ClientUseCase       interface{} // *clientapp.UseCase
	ClientDomainService interface{} // *clientdomain.Service

	// Новая архитектура Project Domain (Clean Architecture)
	ProjectHandlerV2     interface{} // *project.Handler из internal/api/handlers/project
	ProjectUseCase       interface{} // *projectapp.UseCase
	ProjectDomainService interface{} // *projectdomain.Service

	// Новая архитектура Database Domain (Clean Architecture)
	DatabaseHandlerV2     interface{} // *database.Handler из internal/api/handlers/database
	DatabaseUseCase       interface{} // *databaseapp.UseCase
	DatabaseDomainService interface{} // *databasedomain.Service

	NormalizationHandler      *handlers.NormalizationHandler
	CounterpartyHandler       *handlers.CounterpartyHandler
	ClientHandler             *handlers.ClientHandler
	DatabaseHandler           *handlers.DatabaseHandler
	QualityHandler            *handlers.QualityHandler
	ClassificationHandler     *handlers.ClassificationHandler
	SimilarityHandler         *handlers.SimilarityHandler
	MonitoringHandler         *handlers.MonitoringHandler
	ReportHandler             *handlers.ReportHandler
	SnapshotHandler           *handlers.SnapshotHandler
	WorkerHandler             *handlers.WorkerHandler
	NotificationHandler       *handlers.NotificationHandler
	NomenclatureHandler       *handlers.NomenclatureHandler
	GISPHandler               *handlers.GISPHandler
	GostHandler               *handlers.GostHandler
	Processing1CHandler       *handlers.Processing1CHandler
	DuplicateDetectionHandler *handlers.DuplicateDetectionHandler
	PatternDetectionHandler   *handlers.PatternDetectionHandler
	BenchmarkHandler          *handlers.BenchmarkHandler
	ErrorMetricsHandler       *handlers.ErrorMetricsHandler

	// Веб-поиск handlers
	WebSearchHandler           *handlers.WebSearchHandler
	WebSearchAdminHandler      *handlers.WebSearchAdminHandler
	WebSearchValidationHandler *handlers.WebSearchValidationHandler

	// Инфраструктурные компоненты
	Normalizer             *normalization.Normalizer
	NormalizerEvents       chan string
	QualityAnalyzer        interface{} // *quality.QualityAnalyzer
	MonitoringManager      interface{} // *MonitoringManager
	ProviderOrchestrator   interface{} // *ProviderOrchestrator
	MultiProviderClient    interface{} // *MultiProviderClient
	EnrichmentFactory      interface{} // *enrichment.EnricherFactory
	HierarchicalClassifier *normalization.HierarchicalClassifier
	SimilarityCache        *algorithms.OptimizedHybridSimilarity

	// Веб-поиск компоненты
	WebSearchClient             interface{}            // *websearch.MultiProviderClient или *websearch.Client
	WebSearchConfigLoader       interface{}            // *websearch.ConfigLoader
	WebSearchFactory            interface{}            // *websearch.ProviderFactory
	WebSearchReliabilityManager interface{}            // *websearch.ReliabilityManager
	WebSearchRouter             interface{}            // *websearch.ProviderRouter
	WebSearchCache              interface{}            // *websearch.Cache
	WebSearchExistenceValidator interface{}            // *websearch.ProductExistenceValidator
	WebSearchAccuracyValidator  interface{}            // *websearch.ProductAccuracyValidator
	WebSearchRulesConfig        map[string]interface{} // Конфигурация правил валидации

	// ValidationEngine для валидации с использованием веб-поиска
	ValidationEngine *normalization.ValidationEngine

	// Мониторинг
	HealthChecker    *monitoring.HealthChecker
	MetricsCollector *monitoring.MetricsCollector

	// Кэши и менеджеры
	DBInfoCache                 interface{} // *DatabaseInfoCache
	SystemSummaryCache          interface{} // *SystemSummaryCache
	ScanHistoryManager          interface{} // *ScanHistoryManager
	DatabaseModificationTracker interface{} // *DatabaseModificationTracker
	DatabaseConnectionCache     interface{} // *DatabaseConnectionCache

	// Контекст для управления жизненным циклом
	ctx    context.Context
	cancel context.CancelFunc

	// Флаги инициализации
	initialized bool
}

// NewContainer создает новый контейнер зависимостей
func NewContainer(cfg *config.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	container := &Container{
		Config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	return container, nil
}

// Initialize инициализирует все зависимости контейнера
// Выполняется в правильном порядке для избежания циклических зависимостей
func (c *Container) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return fmt.Errorf("container already initialized")
	}

	// Шаг 1: Инициализация баз данных
	if err := c.initDatabases(); err != nil {
		return fmt.Errorf("failed to initialize databases: %w", err)
	}

	// Шаг 2: Инициализация инфраструктурных компонентов
	if err := c.initInfrastructure(); err != nil {
		return fmt.Errorf("failed to initialize infrastructure: %w", err)
	}

	// Шаг 3: Инициализация сервисов
	if err := c.initServices(); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Шаг 3.5: Инициализация новой архитектуры доменов (Clean Architecture)
	if err := c.initUploadComponents(); err != nil {
		return fmt.Errorf("failed to initialize upload components: %w", err)
	}

	if err := c.initNormalizationComponents(); err != nil {
		return fmt.Errorf("failed to initialize normalization components: %w", err)
	}

	if err := c.initQualityComponents(); err != nil {
		return fmt.Errorf("failed to initialize quality components: %w", err)
	}

	if err := c.initClassificationComponents(); err != nil {
		return fmt.Errorf("failed to initialize classification components: %w", err)
	}

	if err := c.initClientComponents(); err != nil {
		return fmt.Errorf("failed to initialize client components: %w", err)
	}

	if err := c.initProjectComponents(); err != nil {
		return fmt.Errorf("failed to initialize project components: %w", err)
	}

	if err := c.initDatabaseComponents(); err != nil {
		return fmt.Errorf("failed to initialize database components: %w", err)
	}

	// Шаг 4: Инициализация обработчиков
	if err := c.initHandlers(); err != nil {
		return fmt.Errorf("failed to initialize handlers: %w", err)
	}

	// Шаг 5: Инициализация мониторинга
	if err := c.initMonitoring(); err != nil {
		return fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	c.initialized = true
	log.Println("Container initialized successfully")
	return nil
}

// initDatabases инициализирует базы данных
func (c *Container) initDatabases() error {
	// Основная БД
	db, err := database.NewDB(c.Config.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to create main database: %w", err)
	}
	c.DB = db

	// Нормализованная БД
	normalizedDB, err := database.NewDB(c.Config.NormalizedDatabasePath)
	if err != nil {
		return fmt.Errorf("failed to create normalized database: %w", err)
	}
	c.NormalizedDB = normalizedDB

	// Сервисная БД
	serviceDB, err := database.NewServiceDB(c.Config.ServiceDatabasePath)
	if err != nil {
		return fmt.Errorf("failed to create service database: %w", err)
	}
	c.ServiceDB = serviceDB

	c.CurrentDBPath = c.Config.DatabasePath
	c.NormalizedDBPath = c.Config.NormalizedDatabasePath

	return nil
}

// initInfrastructure инициализирует инфраструктурные компоненты
func (c *Container) initInfrastructure() error {
	// Канал для событий нормализатора
	c.NormalizerEvents = make(chan string, c.Config.NormalizerEventsBufferSize)

	// Нормализатор
	normalizer := normalization.NewNormalizer(c.DB, c.NormalizerEvents, nil)
	c.Normalizer = normalizer

	// Кэш similarity (используем дефолтные веса)
	weights := algorithms.DefaultSimilarityWeights()
	c.SimilarityCache = algorithms.NewOptimizedHybridSimilarity(weights, 1000)

	// Иерархический классификатор (пока nil, будет инициализирован позже при необходимости)
	c.HierarchicalClassifier = nil

	// Инициализация веб-поиска
	if err := c.initWebSearch(); err != nil {
		log.Printf("Warning: failed to initialize web search: %v", err)
		// Не блокируем инициализацию, если веб-поиск не настроен
	}

	// Настройка ValidationEngine с websearch validators (если включен веб-поиск)
	if err := c.setupValidationWithWebSearch(); err != nil {
		log.Printf("Warning: failed to setup validation with web search: %v", err)
		// Не блокируем инициализацию при ошибке
	}

	// Устанавливаем ValidationEngine в Normalizer (если он был создан)
	if c.ValidationEngine != nil {
		c.Normalizer.SetValidationEngine(c.ValidationEngine)
		log.Println("ValidationEngine установлен в Normalizer")
	}

	return nil
}

// setupValidationWithWebSearch настраивает ValidationEngine с веб-поиском валидаторами
func (c *Container) setupValidationWithWebSearch() error {
	if c.WebSearchClient == nil {
		return nil // Веб-поиск не включен
	}

	if c.ServiceDB == nil {
		return nil // БД недоступна
	}

	// Создаем ValidationEngine
	validationEngine := normalization.NewValidationEngine()
	c.ValidationEngine = validationEngine

	// Получаем клиент (может быть как Client, так и MultiProviderClient)
	var searchClient websearch.SearchClientInterface
	if client, ok := c.WebSearchClient.(websearch.SearchClientInterface); ok {
		searchClient = client
	} else if simpleClient, ok := c.WebSearchClient.(*websearch.Client); ok {
		searchClient = simpleClient
	} else if multiClient, ok := c.WebSearchClient.(*websearch.MultiProviderClient); ok {
		searchClient = multiClient
	} else {
		return nil // Клиент не поддерживается
	}

	// Создаем валидаторы
	existenceValidator := websearch.NewProductExistenceValidator(searchClient)
	accuracyValidator := websearch.NewProductAccuracyValidator(searchClient)

	// Загружаем конфигурацию правил из БД
	rulesConfig, err := normalization.LoadWebSearchRulesConfig(c.ServiceDB)
	if err != nil {
		log.Printf("Warning: failed to load websearch rules config: %v", err)
		rulesConfig = make(map[string]interface{})
	}

	// Настраиваем валидаторы в ValidationEngine
	validationEngine.SetWebSearchValidators(existenceValidator, accuracyValidator, rulesConfig)

	// Сохраняем валидаторы в контейнере для возможного использования
	c.WebSearchExistenceValidator = existenceValidator
	c.WebSearchAccuracyValidator = accuracyValidator
	c.WebSearchRulesConfig = rulesConfig

	log.Printf("ValidationEngine configured with websearch validators (rules enabled: existence=%v, accuracy=%v)",
		normalization.IsRuleEnabled(rulesConfig, "product_name", "existence"),
		normalization.IsRuleEnabled(rulesConfig, "product_code", "accuracy"))

	return nil
}

// initWebSearch инициализирует компоненты веб-поиска
// Поддерживает как простой Client, так и MultiProviderClient с несколькими провайдерами
func (c *Container) initWebSearch() error {
	if c.Config.WebSearch == nil || !c.Config.WebSearch.Enabled {
		log.Println("Web search is disabled in config")
		return nil
	}

	// Создаем кэш
	cacheConfig := &websearch.CacheConfig{
		Enabled:         c.Config.WebSearch.CacheEnabled,
		TTL:             c.Config.WebSearch.CacheTTL,
		CleanupInterval: c.Config.WebSearch.CacheTTL / 4,
		MaxSize:         1000,
	}
	cache := websearch.NewCache(cacheConfig)
	c.WebSearchCache = cache

	// Если ServiceDB доступна, пытаемся использовать MultiProviderClient
	if c.ServiceDB != nil {
		return c.initMultiProviderWebSearch(cache)
	}

	// Иначе используем простой клиент (DuckDuckGo)
	return c.initSimpleWebSearch(cache)
}

// initMultiProviderWebSearch инициализирует MultiProviderClient с несколькими провайдерами
func (c *Container) initMultiProviderWebSearch(cache *websearch.Cache) error {
	// Создаем репозиторий
	webSearchRepo := persistence.NewWebSearchRepository(c.ServiceDB)

	// Создаем загрузчик конфигурации
	configLoader := websearch.NewConfigLoader(webSearchRepo)
	c.WebSearchConfigLoader = configLoader

	// Загружаем провайдеры из БД
	providerConfigs, err := configLoader.LoadEnabledProviders()
	if err != nil {
		log.Printf("Warning: failed to load providers from DB: %v, using simple client", err)
		return c.initSimpleWebSearch(cache)
	}

	// Создаем фабрику провайдеров
	factory := websearch.NewProviderFactory(c.Config.WebSearch.Timeout)
	c.WebSearchFactory = factory

	// Создаем провайдеры из конфигураций
	providers, err := factory.CreateProviders(providerConfigs)
	if err != nil {
		log.Printf("Warning: failed to create providers: %v, using simple client", err)
		return c.initSimpleWebSearch(cache)
	}

	// Если провайдеров нет, используем простой клиент
	if len(providers) == 0 {
		log.Println("No providers configured, using simple DuckDuckGo client")
		return c.initSimpleWebSearch(cache)
	}

	// Создаем ReliabilityManager для отслеживания статистики
	// Временно используем stub реализацию
	var reliabilityManager websearch.ReliabilityManagerInterface
	// TODO: Восстановить после доработки reliability.go
	// reliabilityManager, err := websearch.NewReliabilityManager(webSearchRepo)
	// if err != nil {
	// 	log.Printf("Warning: failed to create reliability manager: %v", err)
	// 	reliabilityManager = websearch.NewStubReliabilityManager()
	// }
	reliabilityManager = websearch.NewStubReliabilityManager()
	c.WebSearchReliabilityManager = reliabilityManager

	// Создаем роутер провайдеров
	routerConfig := websearch.RouterConfig{
		Strategy: websearch.StrategyRoundRobin,
	}
	router := websearch.NewProviderRouter(providers, reliabilityManager, routerConfig)
	c.WebSearchRouter = router

	// Создаем MultiProviderClient
	multiClientConfig := websearch.MultiProviderClientConfig{
		Providers: providers,
		Router:    router,
		Cache:     cache,
		Timeout:   c.Config.WebSearch.Timeout,
	}
	multiClient := websearch.NewMultiProviderClient(multiClientConfig)
	c.WebSearchClient = multiClient

	log.Printf("Web search initialized with MultiProviderClient (%d providers)", len(providers))
	return nil
}

// initSimpleWebSearch инициализирует упрощенную версию веб-поиска без БД
func (c *Container) initSimpleWebSearch(cache *websearch.Cache) error {
	// Создаем простой клиент (DuckDuckGo)
	rateLimit := rate.Every(time.Duration(1000/c.Config.WebSearch.RateLimitPerSec) * time.Millisecond)
	clientConfig := websearch.ClientConfig{
		BaseURL:   c.Config.WebSearch.BaseURL,
		Timeout:   c.Config.WebSearch.Timeout,
		RateLimit: rateLimit,
		Cache:     cache,
	}
	client := websearch.NewClient(clientConfig)
	c.WebSearchClient = client

	log.Println("Web search initialized with DuckDuckGo client")
	return nil
}

// initServices инициализирует бизнес-сервисы
func (c *Container) initServices() error {

	// Benchmark service (используется другими сервисами)
	c.BenchmarkService = services.NewBenchmarkService(nil, c.DB, c.ServiceDB)

	// Normalization service
	c.NormalizationService = services.NewNormalizationService(
		c.DB,
		c.ServiceDB,
		c.Normalizer,
		c.BenchmarkService,
		c.NormalizerEvents,
	)

	// Counterparty service
	c.CounterpartyService = services.NewCounterpartyService(
		c.ServiceDB,
		c.NormalizerEvents,
		c.BenchmarkService,
	)

	// Upload service (правильный порядок: db, serviceDB, dbInfoCache, logFunc)
	c.UploadService = services.NewUploadService(
		c.DB,
		c.ServiceDB,
		c.DBInfoCache,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
	)

	// Client service (требует serviceDB, db, normalizedDB)
	var err error
	c.ClientService, err = services.NewClientService(
		c.ServiceDB,
		c.DB,
		c.NormalizedDB,
	)
	if err != nil {
		return fmt.Errorf("failed to create client service: %w", err)
	}

	// Database service
	c.DatabaseService = services.NewDatabaseService(
		c.ServiceDB,
		c.DB,
		c.NormalizedDB,
		c.CurrentDBPath,
		c.NormalizedDBPath,
		c.DBInfoCache,
	)

	// Quality service (требует db и qualityAnalyzer, пока используем nil для analyzer)
	// TODO: Инициализировать QualityAnalyzer перед созданием сервиса
	c.QualityService, err = services.NewQualityService(
		c.DB,
		nil, // qualityAnalyzer будет инициализирован позже
	)
	if err != nil {
		return fmt.Errorf("failed to create quality service: %w", err)
	}

	// Classification service (требует db, normalizedDB, serviceDB, getModelFromConfig, getAPIKeyFromConfig)
	// В этом контейнере WorkerConfigManager не доступен, используем только переменную окружения
	c.ClassificationService = services.NewClassificationService(
		c.DB,
		c.NormalizedDB,
		c.ServiceDB,
		func() string {
			// TODO: Получать модель из конфигурации
			return ""
		},
		nil, // getAPIKeyFromConfig - не доступен в этом контейнере
	)

	// Similarity service (требует только similarityCache)
	c.SimilarityService = services.NewSimilarityService(
		c.SimilarityCache,
	)

	// Monitoring service (требует db, normalizer, startTime)
	// TODO: Исправить несоответствие интерфейса NormalizerInterface
	// Пока используем nil для normalizer, так как требуется адаптер
	c.MonitoringService = services.NewMonitoringService(
		c.DB,
		nil, // TODO: Создать адаптер для normalization.Normalizer -> services.NormalizerInterface
		time.Now(),
	)

	// Report service (требует db, normalizedDB, serviceDB)
	c.ReportService = services.NewReportService(
		c.DB,
		c.NormalizedDB,
		c.ServiceDB,
	)

	// Snapshot service
	c.SnapshotService = services.NewSnapshotService(c.DB)

	// Worker service (требует workerConfigManager интерфейс)
	// TODO: Инициализировать workerConfigManager позже
	c.WorkerService = nil // Будет инициализирован после создания workerConfigManager

	// Notification service (требует только serviceDB)
	if c.ServiceDB == nil {
		return fmt.Errorf("ServiceDB is required for NotificationService but is nil")
	}
	c.NotificationService = services.NewNotificationService(
		c.ServiceDB,
	)

	// Nomenclature service (требует workerConfigManager с GetNomenclatureConfig)
	// TODO: Инициализировать после создания workerConfigManager
	c.NomenclatureService = nil // Будет инициализирован позже, когда будет workerConfigManager

	// GISP service
	c.GISPService = services.NewGISPService(c.ServiceDB)

	// Gost service
	gostsDB, err := database.NewGostsDB("gosts.db")
	if err != nil {
		log.Printf("Warning: failed to initialize GOSTs database: %v", err)
	} else if gostsDB != nil {
		c.GostService = services.NewGostService(gostsDB)
	}

	// Processing1C service
	c.Processing1CService = services.NewProcessing1CService()

	// Duplicate detection service
	c.DuplicateDetectionService = services.NewDuplicateDetectionService()

	// Pattern detection service (требует функцию получения API ключа)
	c.PatternDetectionService = services.NewPatternDetectionService(
		func() string {
			// TODO: Получать API ключ из конфигурации
			return os.Getenv("ARLIAI_API_KEY")
		},
	)

	return nil
}

// initHandlers инициализирует HTTP обработчики
func (c *Container) initHandlers() error {
	baseHandler := handlers.NewBaseHandlerFromMiddleware()

	// Upload handler
	c.UploadHandler = handlers.NewUploadHandler(
		c.UploadService,
		baseHandler,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
	)

	// Normalization handler (требует только <-chan string для чтения)
	normalizerEventsReadOnly := (<-chan string)(c.NormalizerEvents)
	c.NormalizationHandler = handlers.NewNormalizationHandler(
		c.NormalizationService,
		baseHandler,
		normalizerEventsReadOnly,
	)

	// Counterparty handler (правильный порядок: baseHandler, counterpartyService, logFunc)
	c.CounterpartyHandler = handlers.NewCounterpartyHandler(
		baseHandler,
		c.CounterpartyService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
	)
	c.CounterpartyHandler.SetExportManager(handlers.NewDefaultCounterpartyExportManager())

	// Client handler (требует только clientService и baseHandler)
	c.ClientHandler = handlers.NewClientHandler(
		c.ClientService,
		baseHandler,
	)

	// Database handler
	c.DatabaseHandler = handlers.NewDatabaseHandler(
		c.DatabaseService,
		baseHandler,
	)

	// Quality handler (требует baseHandler, qualityService, logFunc, normalizedDB, currentNormalizedDBPath)
	c.QualityHandler = handlers.NewQualityHandler(
		baseHandler,
		c.QualityService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
		c.NormalizedDB,
		c.NormalizedDBPath,
	)

	// Classification handler (правильный порядок: baseHandler, classificationService, logFunc)
	c.ClassificationHandler = handlers.NewClassificationHandler(
		baseHandler,
		c.ClassificationService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
	)

	// Similarity handler (правильный порядок: baseHandler, similarityService, logFunc)
	c.SimilarityHandler = handlers.NewSimilarityHandler(
		baseHandler,
		c.SimilarityService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
	)

	// Monitoring handler (требует много функций для получения метрик)
	c.MonitoringHandler = handlers.NewMonitoringHandler(
		baseHandler,
		c.MonitoringService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
		func() map[string]interface{} {
			return map[string]interface{}{"state": "closed"}
		},
		func() map[string]interface{} {
			return map[string]interface{}{}
		},
		func() map[string]interface{} {
			return map[string]interface{}{}
		},
		func() *database.PerformanceMetricsSnapshot {
			return nil // TODO: Реализовать позже
		},
		func() handlers.MonitoringData {
			return handlers.MonitoringData{}
		},
	)

	// Report handler (правильный порядок и требуется много функций генерации)
	c.ReportHandler = handlers.NewReportHandler(
		baseHandler,
		c.ReportService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
		func() (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
		func(*int) (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
		func(string) (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
	)

	// WebSearch handlers (используем уже созданные компоненты из initInfrastructure)
	if c.Config.WebSearch != nil && c.Config.WebSearch.Enabled {
		// Получаем клиент из контейнера (уже создан в initInfrastructure)
		// Может быть как *websearch.Client, так и *websearch.MultiProviderClient
		var searchClient websearch.SearchClientInterface

		if client, ok := c.WebSearchClient.(websearch.SearchClientInterface); ok {
			searchClient = client
		} else if simpleClient, ok := c.WebSearchClient.(*websearch.Client); ok {
			searchClient = simpleClient
		} else if multiClient, ok := c.WebSearchClient.(*websearch.MultiProviderClient); ok {
			searchClient = multiClient
		} else {
			// Fallback: создаем простой клиент, если не был создан ранее
			cacheConfig := &websearch.CacheConfig{
				Enabled:         c.Config.WebSearch.CacheEnabled,
				TTL:             c.Config.WebSearch.CacheTTL,
				CleanupInterval: c.Config.WebSearch.CacheTTL / 4,
				MaxSize:         1000,
			}
			searchCache := websearch.NewCache(cacheConfig)
			rateLimit := rate.Every(time.Duration(1000/c.Config.WebSearch.RateLimitPerSec) * time.Millisecond)
			clientConfig := websearch.ClientConfig{
				BaseURL:   c.Config.WebSearch.BaseURL,
				Timeout:   c.Config.WebSearch.Timeout,
				RateLimit: rateLimit,
				Cache:     searchCache,
			}
			simpleClient := websearch.NewClient(clientConfig)
			searchClient = simpleClient
			c.WebSearchClient = simpleClient
			c.WebSearchCache = searchCache
		}

		// Создаем handlers
		c.WebSearchHandler = handlers.NewWebSearchHandler(baseHandler, searchClient)
		c.WebSearchValidationHandler = handlers.NewWebSearchValidationHandler(baseHandler, searchClient)
		c.WebSearchAdminHandler = handlers.NewWebSearchAdminHandler(baseHandler, searchClient)
	}

	// Snapshot handler
	c.SnapshotHandler = handlers.NewSnapshotHandler(
		baseHandler,
		c.SnapshotService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
		c.ServiceDB,
		nil, nil, nil, nil, nil,
	)

	// Worker handler (требует много функций для работы с провайдерами)
	// TODO: Инициализировать после создания всех провайдеров
	c.WorkerHandler = handlers.NewWorkerHandler(
		baseHandler,
		c.WorkerService,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
		func(ctx context.Context, traceID string) (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
		func(ctx context.Context, traceID string, apiKey string) (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
		func(ctx context.Context, traceID string, apiKey string, baseURL string) (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
		func(ctx context.Context, traceID string, providerFilter string, filterStatus string, filterEnabled string, searchQuery string) (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
		func() (string, []string, []interface{}) {
			return "", []string{}, []interface{}{}
		},
		func(strategy string) error {
			return fmt.Errorf("not implemented")
		},
		func() (interface{}, error) {
			return nil, fmt.Errorf("not implemented")
		},
		func(apiKey string, baseURL string) error {
			return fmt.Errorf("not implemented")
		},
		func(providerName string, adapter interface{}, enabled bool, priority int) error {
			return fmt.Errorf("not implemented")
		},
	)

	// Notification handler (требует только notificationService и baseHandler)
	c.NotificationHandler = handlers.NewNotificationHandler(
		c.NotificationService,
		baseHandler,
	)

	// Nomenclature handler (требует только nomenclatureService и baseHandler)
	if c.NomenclatureService != nil {
		c.NomenclatureHandler = handlers.NewNomenclatureHandler(
			c.NomenclatureService,
			baseHandler,
		)
	}

	// GISP handler
	c.GISPHandler = handlers.NewGISPHandler(
		c.GISPService,
		baseHandler,
	)

	// Gost handler
	if c.GostService != nil {
		c.GostHandler = handlers.NewGostHandler(c.GostService)
	}

	// Processing1C handler
	c.Processing1CHandler = handlers.NewProcessing1CHandler(
		c.Processing1CService,
		baseHandler,
	)

	// Duplicate detection handler
	c.DuplicateDetectionHandler = handlers.NewDuplicateDetectionHandler(
		c.DuplicateDetectionService,
		baseHandler,
	)

	// Pattern detection handler (требует функцию для получения уникальных значений)
	c.PatternDetectionHandler = handlers.NewPatternDetectionHandler(
		c.PatternDetectionService,
		baseHandler,
		func(limit int, table string, column string) ([]string, error) {
			// TODO: Реализовать получение уникальных значений из БД
			return []string{}, nil
		},
	)

	// Benchmark handler
	c.BenchmarkHandler = handlers.NewBenchmarkHandler(
		c.BenchmarkService,
		baseHandler,
	)

	// Error metrics handler
	c.ErrorMetricsHandler = handlers.NewErrorMetricsHandler(baseHandler)

	return nil
}

// initMonitoring инициализирует компоненты мониторинга
func (c *Container) initMonitoring() error {
	version := "1.0.0"

	var mainDBConn *sql.DB
	if c.DB != nil {
		mainDBConn = c.DB.GetDB()
	}

	c.HealthChecker = monitoring.NewHealthChecker(version, mainDBConn, c.ServiceDB)
	c.MetricsCollector = monitoring.NewMetricsCollector()

	return nil
}

// Shutdown корректно завершает работу контейнера
func (c *Container) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil
	}

	c.cancel()

	// Закрываем базы данных
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			log.Printf("Error closing main database: %v", err)
		}
	}

	if c.NormalizedDB != nil {
		if err := c.NormalizedDB.Close(); err != nil {
			log.Printf("Error closing normalized database: %v", err)
		}
	}

	if c.ServiceDB != nil {
		if err := c.ServiceDB.Close(); err != nil {
			log.Printf("Error closing service database: %v", err)
		}
	}

	// Закрываем каналы
	if c.NormalizerEvents != nil {
		close(c.NormalizerEvents)
	}

	log.Println("Container shut down successfully")
	return nil
}

// GetContext возвращает контекст контейнера
func (c *Container) GetContext() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// IsInitialized проверяет, инициализирован ли контейнер
func (c *Container) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}
