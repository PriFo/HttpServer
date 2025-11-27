package container

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	"httpserver/database"
	"httpserver/internal/config"
	"httpserver/normalization"
	"httpserver/server/handlers"
	"httpserver/server/monitoring"
	"httpserver/server/services"
)

// SimpleContainer упрощенная версия контейнера для постепенного рефакторинга
// Содержит только критически важные компоненты, остальные инициализируются по требованию
type SimpleContainer struct {
	mu sync.RWMutex

	// Конфигурация
	Config *config.Config

	// Базы данных
	DB            *database.DB
	NormalizedDB  *database.DB
	ServiceDB     *database.ServiceDB
	CurrentDBPath string
	NormalizedDBPath string

	// Основные сервисы (только критически важные)
	NormalizationService *services.NormalizationService
	UploadService        *services.UploadService
	ClientService        *services.ClientService
	DatabaseService      *services.DatabaseService

	// Основные обработчики
	UploadHandler    *handlers.UploadHandler
	DatabaseHandler  *handlers.DatabaseHandler

	// Инфраструктурные компоненты
	Normalizer       *normalization.Normalizer
	NormalizerEvents chan string

	// Мониторинг
	HealthChecker    *monitoring.HealthChecker
	MetricsCollector *monitoring.MetricsCollector

	// Контекст
	ctx    context.Context
	cancel context.CancelFunc

	initialized bool
}

// NewSimpleContainer создает упрощенный контейнер
func NewSimpleContainer(cfg *config.Config) (*SimpleContainer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SimpleContainer{
		Config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Initialize инициализирует упрощенный контейнер
func (c *SimpleContainer) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return fmt.Errorf("container already initialized")
	}

	// Шаг 1: Базы данных
	if err := c.initDatabases(); err != nil {
		return fmt.Errorf("failed to initialize databases: %w", err)
	}

	// Шаг 2: Инфраструктура
	if err := c.initInfrastructure(); err != nil {
		return fmt.Errorf("failed to initialize infrastructure: %w", err)
	}

	// Шаг 3: Базовые сервисы
	if err := c.initBasicServices(); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Шаг 4: Базовые обработчики
	if err := c.initBasicHandlers(); err != nil {
		return fmt.Errorf("failed to initialize handlers: %w", err)
	}

	// Шаг 5: Мониторинг
	if err := c.initMonitoring(); err != nil {
		return fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	c.initialized = true
	log.Println("SimpleContainer initialized successfully")
	return nil
}

func (c *SimpleContainer) initDatabases() error {
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

func (c *SimpleContainer) initInfrastructure() error {
	// Канал для событий нормализатора
	c.NormalizerEvents = make(chan string, c.Config.NormalizerEventsBufferSize)

	// Нормализатор
	c.Normalizer = normalization.NewNormalizer(c.DB, c.NormalizerEvents, nil)

	return nil
}

func (c *SimpleContainer) initBasicServices() error {
	// Benchmark service
	benchmarkService := services.NewBenchmarkService(nil, c.DB, c.ServiceDB)

	// Normalization service
	c.NormalizationService = services.NewNormalizationService(
		c.DB,
		c.ServiceDB,
		c.Normalizer,
		benchmarkService,
		c.NormalizerEvents,
	)

	// Upload service (упрощенная версия без dbInfoCache)
	c.UploadService = services.NewUploadService(
		c.DB,
		c.ServiceDB,
		nil, // dbInfoCache будет nil пока
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
	)

	// Client service (требует serviceDB, db, normalizedDB)
	clientService, err := services.NewClientService(
		c.ServiceDB,
		c.DB,
		c.NormalizedDB,
	)
	if err != nil {
		return fmt.Errorf("failed to create client service: %w", err)
	}
	c.ClientService = clientService

	// Database service
	c.DatabaseService = services.NewDatabaseService(
		c.ServiceDB,
		c.DB,
		c.NormalizedDB,
		c.CurrentDBPath,
		c.NormalizedDBPath,
		nil, // dbInfoCache будет nil пока
	)

	return nil
}

func (c *SimpleContainer) initBasicHandlers() error {
	baseHandler := handlers.NewBaseHandlerFromMiddleware()

	// Upload handler
	c.UploadHandler = handlers.NewUploadHandler(
		c.UploadService,
		baseHandler,
		func(entry interface{}) {
			// logFunc будет установлен позже
		},
	)

	// Database handler
	c.DatabaseHandler = handlers.NewDatabaseHandler(
		c.DatabaseService,
		baseHandler,
	)

	return nil
}

func (c *SimpleContainer) initMonitoring() error {
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
func (c *SimpleContainer) Shutdown(ctx context.Context) error {
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

	log.Println("SimpleContainer shut down successfully")
	return nil
}

// GetContext возвращает контекст контейнера
func (c *SimpleContainer) GetContext() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// IsInitialized проверяет, инициализирован ли контейнер
func (c *SimpleContainer) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

