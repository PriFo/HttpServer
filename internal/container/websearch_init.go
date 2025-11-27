package container

import (
	"log"
	"time"

	"httpserver/internal/infrastructure/persistence"
	"httpserver/websearch"

	"golang.org/x/time/rate"
)

// InitWebSearch инициализирует систему веб-поиска
// Теперь поддерживает MultiProviderClient с несколькими провайдерами
func (c *Container) InitWebSearch() error {
	if c.Config.WebSearch == nil || !c.Config.WebSearch.Enabled {
		log.Println("Web search is disabled in config")
		return nil
	}

	if c.ServiceDB == nil {
		log.Println("Warning: ServiceDB is not initialized, using simple web search client")
		// Создаем кэш для простого клиента
		cacheConfig := &websearch.CacheConfig{
			Enabled:         c.Config.WebSearch.CacheEnabled,
			TTL:             c.Config.WebSearch.CacheTTL,
			CleanupInterval: c.Config.WebSearch.CacheTTL / 4,
			MaxSize:         1000,
		}
		cache := websearch.NewCache(cacheConfig)
		return c.initSimpleWebSearch(cache)
	}

	// Создаем репозиторий
	webSearchRepo := persistence.NewWebSearchRepository(c.ServiceDB)

	// Создаем загрузчик конфигурации
	configLoader := websearch.NewConfigLoader(webSearchRepo)
	c.WebSearchConfigLoader = configLoader

	// Создаем кэш
	cacheConfig := &websearch.CacheConfig{
		Enabled:         c.Config.WebSearch.CacheEnabled,
		TTL:             c.Config.WebSearch.CacheTTL,
		CleanupInterval: c.Config.WebSearch.CacheTTL / 4,
		MaxSize:         1000,
	}
	cache := websearch.NewCache(cacheConfig)
	c.WebSearchCache = cache

	// Загружаем провайдеры из БД
	providerConfigs, err := configLoader.LoadEnabledProviders()
	if err != nil {
		log.Printf("Warning: failed to load providers from DB: %v, using simple client", err)
		return c.initSimpleWebSearchWithCache(cache)
	}

	// Создаем фабрику провайдеров
	factory := websearch.NewProviderFactory(c.Config.WebSearch.Timeout)
	c.WebSearchFactory = factory

	// Создаем провайдеры из конфигураций
	providers, err := factory.CreateProviders(providerConfigs)
	if err != nil {
		log.Printf("Warning: failed to create providers: %v, using simple client", err)
		return c.initSimpleWebSearchWithCache(cache)
	}

	// Если провайдеров нет, используем простой клиент
	if len(providers) == 0 {
		log.Println("No providers configured, using simple DuckDuckGo client")
		return c.initSimpleWebSearchWithCache(cache)
	}

	// Создаем ReliabilityManager для отслеживания статистики
	var reliabilityManager websearch.ReliabilityManagerInterface
	if webSearchRepo != nil {
		// Пытаемся создать полный ReliabilityManager с БД
		rm, err := websearch.NewReliabilityManager(webSearchRepo)
		if err == nil {
			reliabilityManager = rm
		} else {
			log.Printf("Warning: failed to create reliability manager: %v, using stub", err)
			reliabilityManager = websearch.NewStubReliabilityManager()
		}
	} else {
		reliabilityManager = websearch.NewStubReliabilityManager()
	}
	c.WebSearchReliabilityManager = reliabilityManager

	// Создаем роутер провайдеров с конфигурацией
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

// initSimpleWebSearchWithCache инициализирует упрощенную версию веб-поиска без БД с переданным кэшем
func (c *Container) initSimpleWebSearchWithCache(cache *websearch.Cache) error {
	c.WebSearchCache = cache

	// Создаем простой клиент
	rateLimit := rate.Every(time.Duration(1000/c.Config.WebSearch.RateLimitPerSec) * time.Millisecond)
	clientConfig := websearch.ClientConfig{
		BaseURL:   c.Config.WebSearch.BaseURL,
		Timeout:   c.Config.WebSearch.Timeout,
		RateLimit: rateLimit,
		Cache:     cache,
	}
	client := websearch.NewClient(clientConfig)
	c.WebSearchClient = client

	return nil
}
