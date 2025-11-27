package server

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/enrichment"
	"httpserver/internal/config"
	"httpserver/internal/container"
	"httpserver/internal/infrastructure/ai"
	"httpserver/internal/infrastructure/cache"
	inframonitoring "httpserver/internal/infrastructure/monitoring"
	"httpserver/internal/infrastructure/workers"
	"httpserver/nomenclature"
	"httpserver/normalization"
	"httpserver/normalization/algorithms"
	"httpserver/quality"
	"httpserver/server/handlers"
	servermonitoring "httpserver/server/monitoring"
	"httpserver/server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Алиасы для обратной совместимости
type Config = config.Config
type EnrichmentConfig = config.EnrichmentConfig

var LoadConfig = config.LoadConfig
var LoadEnrichmentConfig = config.LoadEnrichmentConfig
var SaveConfig = config.SaveConfig
var SaveConfigWithHistory = config.SaveConfigWithHistory

// LogEntry уже определен в server/models.go, не дублируем здесь

// Server HTTP сервер для приема данных из 1С
type Server struct {
	db                      *database.DB
	normalizedDB            *database.DB
	serviceDB               *database.ServiceDB
	currentDBPath           string
	currentNormalizedDBPath string
	config                  *Config
	httpServer              *http.Server
	httpHandler             http.Handler
	logChan                 chan LogEntry
	nomenclatureProcessor   *nomenclature.NomenclatureProcessor
	processorMutex          sync.RWMutex
	normalizer              *normalization.Normalizer
	normalizerEvents        chan string
	// АРХИТЕКТУРНАЯ ЗАМЕТКА: normalizerRunning дублируется в NormalizationService и CounterpartyService.
	// TODO: Централизовать управление состоянием через services.NormalizationStateManager интерфейс.
	// См. services/normalization_service.go для деталей рефакторинга.
	normalizerRunning       bool
	normalizerMutex         sync.RWMutex
	normalizerStartTime     time.Time
	normalizerProcessed     int
	normalizerSuccess       int
	normalizerErrors        int
	// Context для управления жизненным циклом нормализации
	normalizerCtx        context.Context
	normalizerCancel     context.CancelFunc
	dbMutex              sync.RWMutex
	shutdownChan         chan struct{}
	startTime            time.Time
	qualityAnalyzer      *quality.QualityAnalyzer
	workerConfigManager  *workers.WorkerConfigManager
	arliaiClient         *ai.ArliaiClient
	arliaiCache          *cache.ArliaiCache
	openrouterClient     *ai.OpenRouterClient
	huggingfaceClient    *ai.HuggingFaceClient
	multiProviderClient  *MultiProviderClient                  // Мульти-провайдерный клиент для нормализации имен контрагентов
	similarityCache      *algorithms.OptimizedHybridSimilarity // Глобальный кэш для similarity
	similarityCacheMutex sync.RWMutex
	// Статус анализа качества
	qualityAnalysisRunning bool
	qualityAnalysisMutex   sync.RWMutex
	qualityAnalysisStatus  QualityAnalysisStatus
	// KPVED классификация
	hierarchicalClassifier *normalization.HierarchicalClassifier
	kpvedClassifierMutex   sync.RWMutex
	// Кэшированное дерево КПВЭД для переиспользования (избегает множественных запросов к БД)
	kpvedTree      *normalization.KpvedTree
	kpvedTreeMutex sync.RWMutex
	// Отслеживание текущих задач КПВЭД классификации
	kpvedCurrentTasks      map[int]*classificationTask // workerID -> текущая задача
	kpvedCurrentTasksMutex sync.RWMutex
	// Флаг остановки воркеров КПВЭД классификации
	kpvedWorkersStopped   bool
	kpvedWorkersStopMutex sync.RWMutex
	// Обогащение контрагентов
	enrichmentFactory *enrichment.EnricherFactory
	// Мониторинг провайдеров
	monitoringManager    *inframonitoring.Manager
	providerOrchestrator *ai.ProviderOrchestrator // Оркестратор для мульти-провайдерной нормализации
	// Кэш для информации о БД, проектах и клиентах
	dbInfoCache *cache.DatabaseInfoCache
	// Кэш для результатов сканирования системы
	systemSummaryCache *cache.SystemSummaryCache
	// Менеджер истории сканирований
	scanHistoryManager *cache.ScanHistoryManager
	// Трекер изменений БД для инкрементального сканирования
	dbModificationTracker *cache.DatabaseModificationTracker
	// Кэш для подключений к базам данных (оптимизация открытия БД в циклах)
	dbConnectionCache *cache.DatabaseConnectionCache
	// Сервисы
	normalizationService  *services.NormalizationService
	counterpartyService   *services.CounterpartyService
	uploadService         *services.UploadService
	clientService         *services.ClientService
	databaseService       *services.DatabaseService
	qualityService        *services.QualityService
	classificationService *services.ClassificationService
	similarityService     *services.SimilarityService
	monitoringService     *services.MonitoringService
	reportService         *services.ReportService
	snapshotService       *services.SnapshotService
	workerService         *services.WorkerService
	notificationService   *services.NotificationService
	dashboardService      *services.DashboardService
	// Handlers
	uploadHandler         *handlers.UploadHandler
	clientHandler         *handlers.ClientHandler
	normalizationHandler  *handlers.NormalizationHandler
	qualityHandler        *handlers.QualityHandler
	classificationHandler *handlers.ClassificationHandler
	counterpartyHandler   *handlers.CounterpartyHandler
	similarityHandler     *handlers.SimilarityHandler
	databaseHandler       *handlers.DatabaseHandler
	nomenclatureHandler   *handlers.NomenclatureHandler
	dashboardHandler      *handlers.DashboardHandler
	// dashboardLegacyHandler        *handlers.DashboardLegacyHandler // TODO: восстановить если нужен
	gispHandler                   *handlers.GISPHandler
	gostHandler                   *handlers.GostHandler
	benchmarkHandler              *handlers.BenchmarkHandler
	diagnosticsHandler            *handlers.DiagnosticsHandler
	processing1CHandler           *handlers.Processing1CHandler
	duplicateDetectionHandler     *handlers.DuplicateDetectionHandler
	patternDetectionHandler       *handlers.PatternDetectionHandler
	reclassificationHandler       *handlers.ReclassificationHandler
	normalizationBenchmarkHandler *handlers.NormalizationBenchmarkHandler
	monitoringHandler             *handlers.MonitoringHandler
	workerTraceHandler            *handlers.WorkerTraceHandler
	reportHandler                 *handlers.ReportHandler
	logsHandler                   *handlers.LogsHandler
	snapshotHandler               *handlers.SnapshotHandler
	workerHandler                 *handlers.WorkerHandler
	notificationHandler           *handlers.NotificationHandler
	configHandler                 *handlers.ConfigHandler
	errorMetricsHandler           *handlers.ErrorMetricsHandler
	systemHandler                 *handlers.SystemHandler
	systemSummaryHandler          *handlers.SystemSummaryHandler
	uploadLegacyHandler           *handlers.UploadLegacyHandler
	// Мониторинг
	healthChecker    *servermonitoring.HealthChecker
	metricsCollector *servermonitoring.MetricsCollector
	// Новая архитектура Upload Domain (Clean Architecture)
	uploadHandlerV2 interface{} // *upload.Handler из internal/api/handlers/upload

	// DI контейнер - содержит все зависимости
	// container хранит старый контейнер server.Container для обратной совместимости
	container interface{} // *Container (из server)
	// cleanContainer хранит контейнер новой архитектуры (internal/container)
	cleanContainer *container.Container

	handlerOnce    sync.Once
	handlerInitErr error
}

// QualityAnalysisStatus статус анализа качества
type QualityAnalysisStatus struct {
	IsRunning        bool    `json:"is_running"`
	Progress         float64 `json:"progress"`
	Processed        int     `json:"processed"`
	Total            int     `json:"total"`
	CurrentStep      string  `json:"current_step"`
	DuplicatesFound  int     `json:"duplicates_found"`
	ViolationsFound  int     `json:"violations_found"`
	SuggestionsFound int     `json:"suggestions_found"`
	Error            string  `json:"error,omitempty"`
}

// NewServer создает новый сервер (устаревший метод, используйте NewServerWithConfig)

// shouldStopNormalization проверяет, нужно ли остановить нормализацию
// Thread-safe метод для проверки флага normalizerRunning
func (s *Server) shouldStopNormalization() bool {
	s.normalizerMutex.RLock()
	defer s.normalizerMutex.RUnlock()
	return !s.normalizerRunning
}

// createStopCheckFunction создает функцию проверки остановки для передачи в нормализаторы
// Эта функция используется для устранения дублирования кода проверки остановки
func (s *Server) createStopCheckFunction() func() bool {
	return func() bool {
		return s.shouldStopNormalization()
	}
}

// startSessionTimeoutChecker запускает фоновую задачу для проверки зависших сессий
func (s *Server) startSessionTimeoutChecker() {
	ticker := time.NewTicker(1 * time.Minute) // Проверяем каждую минуту
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, err := s.serviceDB.CheckAndMarkTimeoutSessions()
			if err != nil {
				s.logErrorf("Error checking timeout sessions: %v", err)
			} else if count > 0 {
				log.Printf("Marked %d sessions as timeout", count)
			}
		case <-s.shutdownChan:
			return
		}
	}
}

// getOrCreateKpvedTree получает или создает кэшированное дерево КПВЭД
// Это позволяет переиспользовать дерево для множественных операций, избегая повторных запросов к БД
func (s *Server) getOrCreateKpvedTree() *normalization.KpvedTree {
	// Сначала проверяем кэш (read lock для быстрого доступа)
	s.kpvedTreeMutex.RLock()
	if s.kpvedTree != nil {
		tree := s.kpvedTree
		s.kpvedTreeMutex.RUnlock()
		return tree
	}
	s.kpvedTreeMutex.RUnlock()

	// Дерево не найдено, создаем новое (write lock)
	s.kpvedTreeMutex.Lock()
	defer s.kpvedTreeMutex.Unlock()

	// Двойная проверка (double-checked locking pattern)
	if s.kpvedTree != nil {
		return s.kpvedTree
	}

	// Создаем новое дерево
	if s.serviceDB == nil {
		log.Printf("[KpvedTree] ERROR: ServiceDB is nil, cannot build KPVED tree")
		return nil
	}

	log.Printf("[KpvedTree] Building KPVED tree from database...")
	tree := normalization.NewKpvedTree()
	if err := tree.BuildFromDatabase(s.serviceDB); err != nil {
		log.Printf("[KpvedTree] ERROR: Failed to build KPVED tree: %v", err)
		return nil
	}

	nodeCount := len(tree.NodeMap)
	if nodeCount == 0 {
		log.Printf("[KpvedTree] ERROR: KPVED tree is empty!")
		return nil
	}

	sectionCount := len(tree.Root.Children)
	log.Printf("[KpvedTree] KPVED tree built successfully: %d nodes, %d sections (cached for reuse)", nodeCount, sectionCount)

	// Кэшируем дерево
	s.kpvedTree = tree
	return tree
}

// invalidateKpvedTreeCache инвалидирует кэш дерева КПВЭД
// Вызывается при изменении данных классификатора в БД
func (s *Server) invalidateKpvedTreeCache() {
	s.kpvedTreeMutex.Lock()
	defer s.kpvedTreeMutex.Unlock()
	s.kpvedTree = nil
	log.Printf("[KpvedTree] Cache invalidated")
}

// initDefaultProjectTypeClassifiers инициализирует дефолтные привязки классификаторов к типам проектов
func (s *Server) initDefaultProjectTypeClassifiers() {
	if s.serviceDB == nil {
		log.Printf("Warning: ServiceDB not initialized, skipping default project type classifiers initialization")
		return
	}

	// Получаем все существующие привязки
	existing, err := s.serviceDB.GetAllProjectTypeClassifiers()
	if err != nil {
		log.Printf("Warning: Failed to get existing project type classifiers: %v", err)
		return
	}

	// Если уже есть привязки, не инициализируем
	if len(existing) > 0 {
		log.Printf("Project type classifiers already initialized, skipping")
		return
	}

	// Получаем все классификаторы через serviceDB
	// Используем прямой SQL запрос, так как GetCategoryClassifiersByFilter находится в database.DB
	query := `SELECT id, name FROM category_classifiers WHERE is_active = TRUE`
	rows, err := s.serviceDB.Query(query)
	if err != nil {
		log.Printf("Warning: Failed to get classifiers: %v", err)
		return
	}
	defer rows.Close()

	classifierMap := make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Printf("Warning: Failed to scan classifier: %v", err)
			continue
		}
		classifierMap[name] = id
	}

	// Привязываем классификаторы к типу nomenclature_counterparties
	projectType := "nomenclature_counterparties"
	bindings := []struct {
		name        string
		description string
		isDefault   bool
	}{
		{"Adata.kz", "Классификатор адресов Adata.kz для нормализации адресов контрагентов", true},
		{"DaData.ru", "Классификатор адресов DaData.ru для нормализации адресов контрагентов", true},
		{"КПВЭД", "Классификатор видов экономической деятельности (КПВЭД) для классификации номенклатуры", true},
	}

	// Создаем недостающие классификаторы
	for _, binding := range bindings {
		if _, exists := classifierMap[binding.name]; !exists {
			// Создаем классификатор через прямой SQL запрос
			insertQuery := `INSERT INTO category_classifiers (name, description, max_depth, tree_structure, is_active) 
				VALUES (?, ?, ?, ?, ?)`
			result, err := s.serviceDB.Exec(insertQuery, binding.name, binding.description, 6, "{}", true)
			if err != nil {
				log.Printf("Warning: Failed to create classifier '%s': %v", binding.name, err)
				continue
			}
			id, err := result.LastInsertId()
			if err != nil {
				log.Printf("Warning: Failed to get classifier ID for '%s': %v", binding.name, err)
				continue
			}
			classifierMap[binding.name] = int(id)
			log.Printf("Created missing classifier: %s (ID: %d)", binding.name, id)
		}
	}

	created := 0
	for _, binding := range bindings {
		if classifierID, exists := classifierMap[binding.name]; exists {
			_, err := s.serviceDB.CreateProjectTypeClassifier(projectType, classifierID, binding.isDefault)
			if err != nil {
				log.Printf("Warning: Failed to create project type classifier binding for %s: %v", binding.name, err)
			} else {
				created++
				log.Printf("Created project type classifier binding: %s -> %s", projectType, binding.name)
			}
		} else {
			log.Printf("Warning: Classifier '%s' not found, skipping binding", binding.name)
		}
	}

	if created > 0 {
		log.Printf("Initialized %d default project type classifier bindings for %s", created, projectType)
	} else if len(classifierMap) == 0 {
		log.Printf("Info: No classifiers found in database. Classifiers need to be created first before binding to project types.")
		log.Printf("Info: To create classifiers, use the classification API or load them from external sources.")
	} else {
		missing := []string{}
		for _, binding := range bindings {
			if _, exists := classifierMap[binding.name]; !exists {
				missing = append(missing, binding.name)
			}
		}
		if len(missing) > 0 {
			log.Printf("Info: Some classifiers not found for binding: %v. They need to be created first.", missing)
		}
	}
}

// httpHandlerToGin адаптирует http.HandlerFunc в gin.HandlerFunc и прокидывает path-параметры в http.Request context.
func httpHandlerToGin(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request

		// Прокидываем все path-параметры Gin в контекст стандартного http.Request,
		// чтобы legacy handlers могли получать их через r.Context().Value(...)
		if len(c.Params) > 0 {
			ctx := req.Context()
			for _, param := range c.Params {
				ctx = context.WithValue(ctx, param.Key, param.Value)
			}
			req = req.WithContext(ctx)
		}

		handler(c.Writer, req)
	}
}

// clientIDWrapper создает обертку для методов, принимающих clientID
func clientIDWrapper(handler func(http.ResponseWriter, *http.Request, int)) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIDStr := c.Param("clientId")
		if clientIDStr == "" {
			clientIDStr = c.Param("id")
		}
		if clientIDStr == "" {
			log.Printf("[clientIDWrapper] ERROR: clientId and id params are both empty for path: %s", c.Request.URL.Path)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
			return
		}
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			log.Printf("[clientIDWrapper] ERROR: Invalid client ID '%s': %v", clientIDStr, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}
		log.Printf("[clientIDWrapper] Extracted clientID: %d from path: %s", clientID, c.Request.URL.Path)
		handler(c.Writer, c.Request, clientID)
	}
}

// clientProjectIDWrapper создает обертку для методов, принимающих clientID и projectID
func clientProjectIDWrapper(handler func(http.ResponseWriter, *http.Request, int, int)) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIDStr := c.Param("clientId")
		projectIDStr := c.Param("projectId")
		if projectIDStr == "" {
			projectIDStr = c.Param("id")
		}
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
			return
		}
		handler(c.Writer, c.Request, clientID, projectID)
	}
}

// clientProjectDatabaseIDWrapper создает обертку для методов, принимающих clientID, projectID и dbID
func clientProjectDatabaseIDWrapper(handler func(http.ResponseWriter, *http.Request, int, int, int)) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIDStr := c.Param("clientId")
		projectIDStr := c.Param("projectId")
		dbIDStr := c.Param("databaseId")
		if dbIDStr == "" {
			dbIDStr = c.Param("id")
		}
		clientID, err := strconv.Atoi(clientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
			return
		}
		dbID, err := strconv.Atoi(dbIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid database ID"})
			return
		}
		handler(c.Writer, c.Request, clientID, projectID, dbID)
	}
}

// handleVerifyUploadNormalized, handleNormalizedHandshake, handleNormalizedMetadata,
// handleNormalizedConstant, handleNormalizedCatalogMeta, handleNormalizedCatalogItem,
// handleNormalizedComplete определены в upload_normalized_handlers.go

// startNomenclatureProcessing запускает обработку номенклатуры
func (s *Server) startNomenclatureProcessing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorResponse(w, "Failed to read request body", err)
		return
	}

	var req HandshakeRequest
	if err := xml.Unmarshal(body, &req); err != nil {
		s.writeErrorResponse(w, "Failed to parse XML", err)
		return
	}

	// Создаем новую выгрузку в нормализованной БД
	uploadUUID := uuid.New().String()
	_, err = s.normalizedDB.CreateUpload(uploadUUID, req.Version1C, req.ConfigName)
	if err != nil {
		s.writeErrorResponse(w, "Failed to create normalized upload", err)
		return
	}

	s.log(LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    fmt.Sprintf("Normalized handshake successful for upload %s", uploadUUID),
		UploadUUID: uploadUUID,
		Endpoint:   "/api/normalized/upload/handshake",
	})

	response := HandshakeResponse{
		Success:    true,
		UploadUUID: uploadUUID,
		Message:    "Normalized handshake successful",
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	s.writeXMLResponse(w, response)
}

// startNomenclatureProcessing запускает обработку номенклатуры
