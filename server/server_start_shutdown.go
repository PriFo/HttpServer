package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"httpserver/internal/api/routes"
	"httpserver/server/handlers"
	"httpserver/server/middleware"
)

// Start запускает HTTP сервер
func (s *Server) Start() error {
	handler, err := s.ensureHTTPHandler()
	if err != nil {
		return err
	}

	// Создаем http.Server
	// WriteTimeout увеличен для SSE потоков, которые могут отправлять данные реже
	// Heartbeat отправляется каждые 30 секунд, поэтому WriteTimeout должен быть больше
	addr := fmt.Sprintf(":%s", s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 5 * time.Minute,   // Поддерживаем длинные стримы экспорта
		IdleTimeout:  120 * time.Second, // Увеличено для долгих SSE соединений
	}

	log.Printf("Сервер запускается на порту %s", s.config.Port)

	// Проверяем, что httpServer создан корректно
	if s.httpServer == nil {
		log.Fatalf("✗ КРИТИЧЕСКАЯ ОШИБКА: httpServer равен nil, сервер не может быть запущен")
	}
	if s.httpServer.Addr == "" {
		log.Fatalf("✗ КРИТИЧЕСКАЯ ОШИБКА: httpServer.Addr пуст, сервер не может быть запущен")
	}
	if s.httpServer.Handler == nil {
		log.Fatalf("✗ КРИТИЧЕСКАЯ ОШИБКА: httpServer.Handler равен nil, сервер не может быть запущен")
	}

	// Запускаем фоновые задачи
	go s.startSessionTimeoutChecker()

	// Проверяем и загружаем КПВЭД при необходимости
	s.ensureKpvedLoaded()

	// Логируем перед запуском сервера
	log.Printf("Starting HTTP server on %s...", s.httpServer.Addr)
	log.Printf("API доступно по адресу: http://localhost%s", s.httpServer.Addr)

	// Запускаем сервер
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("✗ КРИТИЧЕСКАЯ ОШИБКА: Не удалось запустить HTTP сервер на %s: %v", s.httpServer.Addr, err)
	}

	return nil
}

func (s *Server) ensureHTTPHandler() (http.Handler, error) {
	s.handlerOnce.Do(func() {
		log.Printf("[ensureHTTPHandler] Начало создания HTTP handler")
		handler, err := s.buildHTTPHandler()
		if err != nil {
			log.Printf("[ensureHTTPHandler] ✗ ОШИБКА при создании HTTP handler: %v", err)
			s.handlerInitErr = err
			return
		}
		s.httpHandler = handler
		log.Printf("[ensureHTTPHandler] ✓ HTTP handler успешно создан")
	})

	if s.handlerInitErr != nil {
		log.Printf("[ensureHTTPHandler] Возврат ошибки: %v", s.handlerInitErr)
		return nil, s.handlerInitErr
	}

	if s.httpHandler == nil {
		log.Printf("[ensureHTTPHandler] ✗ КРИТИЧЕСКАЯ ОШИБКА: httpHandler равен nil после создания")
		return nil, fmt.Errorf("httpHandler is nil")
	}

	return s.httpHandler, nil
}

func (s *Server) buildHTTPHandler() (http.Handler, error) {
	log.Printf("[buildHTTPHandler] Начало создания HTTP handler")

	// Инициализируем handlers если еще не инициализированы
	s.initHandlers()
	log.Printf("[buildHTTPHandler] Handlers инициализированы")

	// Устанавливаем режим Gin: release для продакшена, debug для разработки
	// Можно переопределить через переменную окружения GIN_MODE
	if ginMode := os.Getenv("GIN_MODE"); ginMode == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Создаем gin router
	router := gin.New()
	log.Printf("[buildHTTPHandler] Gin router создан")

	// Применяем middleware
	router.Use(middleware.GinRequestIDMiddleware())
	router.Use(middleware.GinCORSMiddleware())
	router.Use(middleware.GinGzipMiddleware())
	router.Use(middleware.GinLoggerMiddleware())
	router.Use(gin.Recovery())
	log.Printf("[buildHTTPHandler] Middleware применены")

	// Регистрируем Swagger
	handlers.RegisterSwaggerRoutes(router)
	log.Printf("[buildHTTPHandler] Swagger routes зарегистрированы")

	// Регистрируем Gin handlers
	s.registerGinHandlers(router)
	log.Printf("[buildHTTPHandler] Gin handlers зарегистрированы")

	// Создаем http.ServeMux для legacy handlers
	mux := http.NewServeMux()

	// Регистрируем legacy routes через legacy routes adapter
	// Убираем зависимость от cleanContainer, так как legacy routes должны работать всегда
	log.Printf("[buildHTTPHandler] Регистрация legacy routes...")
	legacyAdapter := newLegacyRouteAdapter(s)
	if legacyAdapter != nil {
		routes.RegisterLegacyRoutes(router, legacyAdapter)
		log.Printf("[buildHTTPHandler] ✓ Legacy routes adapter создан и зарегистрирован")
	} else {
		log.Printf("[buildHTTPHandler] ⚠ Legacy routes adapter is nil")
	}

	// Регистрируем legacy handlers в mux
	s.registerLegacyHandlers(mux)

	// Используем NoRoute для передачи в mux
	// Добавляем Deprecation заголовок для всех legacy routes
	router.NoRoute(func(c *gin.Context) {
		// Добавляем заголовок Deprecation для legacy routes
		// Это сигнализирует клиентам о необходимости миграции на новые API
		c.Writer.Header().Set("Deprecation", "true")
		c.Writer.Header().Set("Link", "</api/normalization>; rel=\"successor-version\"")
		mux.ServeHTTP(c.Writer, c.Request)
	})

	return router, nil
}

// ServeHTTP реализует http.Handler для тестов и вспомогательных утилит
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, err := s.ensureHTTPHandler()
	if err != nil {
		http.Error(w, "server is not initialized", http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, r)
}

// Shutdown останавливает HTTP сервер gracefully
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	log.Println("Initiating graceful shutdown...")

	// Останавливаем нормализацию, если она запущена
	s.stopAllNormalization()

	// Останавливаем фоновые задачи
	close(s.shutdownChan)

	// Останавливаем сервер
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("ошибка остановки сервера: %w", err)
	}

	log.Println("Graceful shutdown completed")
	return nil
}

// stopAllNormalization останавливает все процессы нормализации
// Вызывается при graceful shutdown для предотвращения утечки горутин
func (s *Server) stopAllNormalization() {
	log.Println("Stopping all normalization processes...")

	// Останавливаем основную нормализацию
	s.normalizerMutex.Lock()
	wasRunning := s.normalizerRunning
	s.normalizerRunning = false
	if s.normalizerCancel != nil {
		s.normalizerCancel()
		s.normalizerCancel = nil
	}
	s.normalizerMutex.Unlock()

	if wasRunning {
		log.Println("Main normalization process stopped")
	}

	// Останавливаем нормализацию контрагентов через сервис
	if s.counterpartyService != nil {
		if s.counterpartyService.Stop() {
			log.Println("Counterparty normalization process stopped")
		}
	}

	// Останавливаем сервис нормализации
	if s.normalizationService != nil {
		if s.normalizationService.Stop() {
			log.Println("NormalizationService stopped")
		}
	}

	log.Println("All normalization processes stopped")
}

// registerGinHandlers регистрирует все Gin handlers
func (s *Server) registerGinHandlers(router *gin.Engine) {
	// Health check endpoint - простой эндпоинт без зависимостей
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "http-server",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	api := router.Group("/api")

	// Databases API
	if s.databaseHandler != nil {
		databasesAPI := api.Group("/databases")
		{
			databasesAPI.GET("/list", s.databaseHandler.HandleDatabasesListGin)
			databasesAPI.GET("/find", s.databaseHandler.HandleFindDatabaseGin)
			databasesAPI.GET("/pending", s.databaseHandler.HandlePendingDatabasesGin)
		}

		databaseAPI := api.Group("/database")
		{
			databaseAPI.GET("/info", s.databaseHandler.HandleDatabaseInfoGin)
		}
	}

	// Quality API
	if s.qualityHandler != nil {
		qualityAPI := api.Group("/quality")
		{
			qualityAPI.GET("/report", s.qualityHandler.HandleQualityReportGin)
			qualityAPI.GET("/score/:database_id", s.qualityHandler.HandleQualityScoreGin)
			qualityAPI.GET("/cache/stats", s.qualityHandler.HandleQualityCacheStatsGin)
			qualityAPI.POST("/cache/invalidate", s.qualityHandler.HandleQualityCacheInvalidateGin)
			qualityAPI.DELETE("/cache/invalidate", s.qualityHandler.HandleQualityCacheInvalidateGin)
			// Добавляем роут для /api/quality/stats
			qualityAPI.GET("/stats", func(c *gin.Context) {
				// Используем текущую БД из сервера
				currentDB := s.db
				if currentDB == nil {
					currentDB = s.normalizedDB
				}
				s.qualityHandler.HandleQualityStats(c.Writer, c.Request, currentDB, s.currentNormalizedDBPath)
			})
		}
	}

	// Dashboard API
	if s.dashboardHandler != nil {
		dashboardAPI := api.Group("/dashboard")
		{
			dashboardAPI.GET("/stats", httpHandlerToGin(s.dashboardHandler.HandleGetStats))
			dashboardAPI.GET("/overview", httpHandlerToGin(s.dashboardHandler.HandleDashboardOverview))
			dashboardAPI.GET("/normalization-status", httpHandlerToGin(s.dashboardHandler.HandleGetNormalizationStatus))
		}

		// Качество на дашборде использует общий обработчик
		qualityDashboardAPI := api.Group("/quality")
		{
			qualityDashboardAPI.GET("/metrics", httpHandlerToGin(s.dashboardHandler.HandleGetQualityMetrics))
		}
	}

	// Monitoring API
	if s.monitoringHandler != nil {
		monitoringAPI := api.Group("/monitoring")
		{
			monitoringAPI.GET("/metrics", httpHandlerToGin(s.monitoringHandler.HandleMonitoringMetrics))
			monitoringAPI.GET("/providers", httpHandlerToGin(s.monitoringHandler.HandleMonitoringProviders))
			// Используем стандартные HTTP handlers через адаптер
			monitoringAPI.Any("/providers/stream", httpHandlerToGin(s.monitoringHandler.HandleMonitoringProvidersStream))
		}
	}

	// Logs API
	if s.logsHandler != nil {
		logsAPI := api.Group("/logs")
		{
			logsAPI.POST("/client-error", httpHandlerToGin(s.logsHandler.HandleClientError))
		}
	}

	// Classification API
	if s.classificationHandler != nil {
		classificationAPI := api.Group("/classification")
		{
			// Используем стандартные HTTP handlers через адаптер
			classificationAPI.GET("/classifiers", httpHandlerToGin(s.classificationHandler.HandleGetClassifiers))
		}

	}

	// Normalization API
	if s.normalizationHandler != nil {
		normalizationAPI := api.Group("/normalization")
		{
			normalizationAPI.GET("/pipeline/stats", httpHandlerToGin(s.normalizationHandler.HandlePipelineStats))
			normalizationAPI.GET("/pipeline/stage-details", httpHandlerToGin(s.normalizationHandler.HandleStageDetails))
			normalizationAPI.POST("/start", httpHandlerToGin(s.normalizationHandler.HandleStartVersionedNormalization))
			normalizationAPI.POST("/stop", httpHandlerToGin(s.normalizationHandler.HandleNormalizationStop))
			normalizationAPI.POST("/apply-patterns", httpHandlerToGin(s.normalizationHandler.HandleApplyPatterns))
			normalizationAPI.POST("/apply-ai", httpHandlerToGin(s.normalizationHandler.HandleApplyAI))
			normalizationAPI.POST("/apply-categorization", httpHandlerToGin(s.normalizationHandler.HandleApplyCategorization))
			normalizationAPI.GET("/history", httpHandlerToGin(s.normalizationHandler.HandleGetSessionHistory))
			normalizationAPI.POST("/revert", httpHandlerToGin(s.normalizationHandler.HandleRevertStage))
			normalizationAPI.GET("/events", httpHandlerToGin(s.normalizationHandler.HandleNormalizationEvents))
			normalizationAPI.GET("/status", httpHandlerToGin(s.normalizationHandler.HandleNormalizationStatus))
			normalizationAPI.GET("/stats", httpHandlerToGin(s.normalizationHandler.HandleNormalizationStats))
			normalizationAPI.GET("/groups", httpHandlerToGin(s.normalizationHandler.HandleNormalizationGroups))
			normalizationAPI.GET("/group-items", httpHandlerToGin(s.normalizationHandler.HandleNormalizationGroupItems))
			normalizationAPI.GET("/item-attributes/:id", httpHandlerToGin(s.normalizationHandler.HandleNormalizationItemAttributes))
			normalizationAPI.GET("/export-group", httpHandlerToGin(s.normalizationHandler.HandleNormalizationExportGroup))
			normalizationAPI.GET("/export", httpHandlerToGin(s.normalizationHandler.HandleExport))
			normalizationAPI.GET("/config", httpHandlerToGin(s.normalizationHandler.HandleNormalizationConfig))
			normalizationAPI.PUT("/config", httpHandlerToGin(s.normalizationHandler.HandleNormalizationConfig))
			normalizationAPI.POST("/config", httpHandlerToGin(s.normalizationHandler.HandleNormalizationConfig))
			normalizationAPI.GET("/databases", httpHandlerToGin(s.normalizationHandler.HandleNormalizationDatabases))
			normalizationAPI.GET("/tables", httpHandlerToGin(s.normalizationHandler.HandleNormalizationTables))
			normalizationAPI.GET("/columns", httpHandlerToGin(s.normalizationHandler.HandleNormalizationColumns))
			normalizationAPI.GET("/benchmark-dataset", httpHandlerToGin(s.normalizationHandler.HandleBenchmarkDataset))
			normalizationAPI.DELETE("/data/all", httpHandlerToGin(s.normalizationHandler.HandleDeleteAllNormalizedData))
			normalizationAPI.DELETE("/data/project", httpHandlerToGin(s.normalizationHandler.HandleDeleteNormalizedDataByProject))
			normalizationAPI.DELETE("/data/session", httpHandlerToGin(s.normalizationHandler.HandleDeleteNormalizedDataBySession))
		}
	}

	// Config API
	if s.configHandler != nil {
		configAPI := api.Group("/config")
		{
			configAPI.GET("", httpHandlerToGin(s.configHandler.HandleGetConfigSafe))
			configAPI.PUT("", httpHandlerToGin(s.configHandler.HandleUpdateConfig))
			configAPI.POST("", httpHandlerToGin(s.configHandler.HandleUpdateConfig))
			configAPI.GET("/full", httpHandlerToGin(s.configHandler.HandleGetConfig))
			configAPI.GET("/history", httpHandlerToGin(s.configHandler.HandleGetConfigHistory))
		}
		log.Printf("[Routes] Config API routes registered: GET /api/config, PUT /api/config, POST /api/config, GET /api/config/full, GET /api/config/history")
	} else {
		log.Printf("⚠ WARNING: configHandler is nil, Config API routes will not be registered")
	}

	// Clients API
	log.Printf("[registerGinHandlers] Проверка clientHandler: %v", s.clientHandler != nil)
	if s.clientHandler != nil {
		log.Printf("[registerGinHandlers] Регистрация Clients API routes")
		clientsAPI := api.Group("/clients")
		{
			// POST /api/clients
			clientsAPI.POST("", httpHandlerToGin(s.clientHandler.CreateClient))
			// GET /api/clients (список всех клиентов)
			clientsAPI.GET("", httpHandlerToGin(s.clientHandler.GetClients))

			// GET /api/clients/:clientId
			clientsAPI.GET("/:clientId", func(c *gin.Context) {
				log.Printf("[Gin Route] GET /api/clients/:clientId matched, path: %s, clientId param: %s", c.Request.URL.Path, c.Param("clientId"))
				clientIDWrapper(s.clientHandler.GetClient)(c)
			})
			// PUT /api/clients/:clientId (обновление клиента)
			clientsAPI.PUT("/:clientId", clientIDWrapper(s.clientHandler.UpdateClient))
			// DELETE /api/clients/:clientId
			clientsAPI.DELETE("/:clientId", clientIDWrapper(s.clientHandler.DeleteClient))

			// GET /api/clients/:clientId/statistics - статистика клиента
			clientsAPI.GET("/:clientId/statistics", clientIDWrapper(s.clientHandler.GetClientStatistics))
			// GET /api/clients/:clientId/nomenclature - номенклатура клиента
			clientsAPI.GET("/:clientId/nomenclature", clientIDWrapper(s.clientHandler.GetClientNomenclature))
			// GET /api/clients/:clientId/databases - базы данных клиента
			clientsAPI.GET("/:clientId/databases", clientIDWrapper(s.clientHandler.GetClientDatabases))

			// Documents для клиента
			clientDocumentsAPI := clientsAPI.Group("/:clientId/documents")
			{
				// GET /api/clients/:clientId/documents
				clientDocumentsAPI.GET("", clientIDWrapper(s.clientHandler.HandleGetClientDocuments))
				// POST /api/clients/:clientId/documents
				clientDocumentsAPI.POST("", clientIDWrapper(s.clientHandler.HandleUploadClientDocument))
				// GET /api/clients/:clientId/documents/:docId
				clientDocumentsAPI.GET("/:docId", func(c *gin.Context) {
					clientID, err := strconv.Atoi(c.Param("clientId"))
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
						return
					}
					docID, err := strconv.Atoi(c.Param("docId"))
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
						return
					}
					s.clientHandler.HandleDownloadClientDocument(c.Writer, c.Request, clientID, docID)
				})
				// DELETE /api/clients/:clientId/documents/:docId
				clientDocumentsAPI.DELETE("/:docId", func(c *gin.Context) {
					clientID, err := strconv.Atoi(c.Param("clientId"))
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
						return
					}
					docID, err := strconv.Atoi(c.Param("docId"))
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
						return
					}
					s.clientHandler.HandleDeleteClientDocument(c.Writer, c.Request, clientID, docID)
				})
			}

			// Projects для клиента
			clientProjectsAPI := clientsAPI.Group("/:clientId/projects")
			{
				// POST /api/clients/:clientId/projects
				clientProjectsAPI.POST("", clientIDWrapper(s.clientHandler.CreateClientProject))
				// GET /api/clients/:clientId/projects (список проектов клиента)
				clientProjectsAPI.GET("", clientIDWrapper(s.clientHandler.GetClientProjects))

				// GET /api/clients/:clientId/projects/:projectId
				clientProjectsAPI.GET("/:projectId", clientProjectIDWrapper(s.clientHandler.GetClientProject))
				// PUT /api/clients/:clientId/projects/:projectId
				clientProjectsAPI.PUT("/:projectId", clientProjectIDWrapper(s.clientHandler.UpdateClientProject))
				// DELETE /api/clients/:clientId/projects/:projectId
				clientProjectsAPI.DELETE("/:projectId", clientProjectIDWrapper(s.clientHandler.DeleteClientProject))

				// Databases для проекта
				projectDatabasesAPI := clientProjectsAPI.Group("/:projectId/databases")
				{
					// POST /api/clients/:clientId/projects/:projectId/databases
					// Обрабатывает как JSON запросы, так и multipart/form-data (загрузка файлов)
					projectDatabasesAPI.POST("", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
						// Проверяем Content-Type для определения типа запроса
						contentType := r.Header.Get("Content-Type")
						if strings.HasPrefix(contentType, "multipart/form-data") {
							// Загрузка файла - используем legacy handler
							s.handleUploadProjectDatabase(w, r, clientID, projectID)
						} else {
							// Обычный JSON запрос - используем новый handler
							s.clientHandler.CreateProjectDatabase(w, r, clientID, projectID)
						}
					}))
					// GET /api/clients/:clientId/projects/:projectId/databases (список баз данных проекта)
					projectDatabasesAPI.GET("", clientProjectIDWrapper(s.clientHandler.GetProjectDatabases))

					// GET /api/clients/:clientId/projects/:projectId/databases/:databaseId
					projectDatabasesAPI.GET("/:databaseId", clientProjectDatabaseIDWrapper(s.clientHandler.GetProjectDatabase))
					// PUT /api/clients/:clientId/projects/:projectId/databases/:databaseId
					projectDatabasesAPI.PUT("/:databaseId", clientProjectDatabaseIDWrapper(s.clientHandler.UpdateProjectDatabase))
					// DELETE /api/clients/:clientId/projects/:projectId/databases/:databaseId
					projectDatabasesAPI.DELETE("/:databaseId", clientProjectDatabaseIDWrapper(s.clientHandler.DeleteProjectDatabase))

					// Tables для базы данных
					// GET /api/clients/:clientId/projects/:projectId/databases/:databaseId/tables
					projectDatabasesAPI.GET("/:databaseId/tables", clientProjectDatabaseIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
						s.handleGetProjectDatabaseTables(w, r, clientID, projectID, dbID)
					}))
					// GET /api/clients/:clientId/projects/:projectId/databases/:databaseId/tables/:tableName
					projectDatabasesAPI.GET("/:databaseId/tables/:tableName", func(c *gin.Context) {
						clientIDStr := c.Param("clientId")
						projectIDStr := c.Param("projectId")
						dbIDStr := c.Param("databaseId")
						tableName := c.Param("tableName")

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

						s.handleGetProjectDatabaseTableData(c.Writer, c.Request, clientID, projectID, dbID, tableName)
					})
				}

				// Benchmarks для проекта
				projectBenchmarksAPI := clientProjectsAPI.Group("/:projectId/benchmarks")
				{
					// GET /api/clients/:clientId/projects/:projectId/benchmarks
					projectBenchmarksAPI.GET("", clientProjectIDWrapper(s.clientHandler.GetProjectBenchmarks))
					// POST /api/clients/:clientId/projects/:projectId/benchmarks
					projectBenchmarksAPI.POST("", clientProjectIDWrapper(s.clientHandler.CreateProjectBenchmark))
				}

				// Diagnostics для проекта
				if s.diagnosticsHandler != nil {
					projectDiagnosticsAPI := clientProjectsAPI.Group("/:projectId/diagnostics")
					{
						// GET /api/clients/:clientId/projects/:projectId/diagnostics/databases
						projectDiagnosticsAPI.GET("/databases", func(c *gin.Context) {
							s.diagnosticsHandler.HandleCheckDatabases(c)
						})
						// GET /api/clients/:clientId/projects/:projectId/diagnostics/uploads
						projectDiagnosticsAPI.GET("/uploads", func(c *gin.Context) {
							s.diagnosticsHandler.HandleCheckUploads(c)
						})
						// POST /api/clients/:clientId/projects/:projectId/diagnostics/uploads/fix
						projectDiagnosticsAPI.POST("/uploads/fix", func(c *gin.Context) {
							s.diagnosticsHandler.HandleFixUploads(c)
						})
						// GET /api/clients/:clientId/projects/:projectId/diagnostics/extraction
						projectDiagnosticsAPI.GET("/extraction", func(c *gin.Context) {
							s.diagnosticsHandler.HandleCheckExtraction(c)
						})
						// GET /api/clients/:clientId/projects/:projectId/diagnostics/normalization
						projectDiagnosticsAPI.GET("/normalization", func(c *gin.Context) {
							s.diagnosticsHandler.HandleCheckNormalization(c)
						})
					}
					log.Printf("[Routes] ✓ Diagnostics API routes registered")
				}

				// Nomenclature для проекта
				projectNomenclatureAPI := clientProjectsAPI.Group("/:projectId/nomenclature")
				{
					// GET /api/clients/:clientId/projects/:projectId/nomenclature
					// Используем GetClientNomenclature с передачей projectId через query параметр
					projectNomenclatureAPI.GET("", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
						// Добавляем project_id в query параметры
						q := r.URL.Query()
						q.Set("project_id", fmt.Sprintf("%d", projectID))
						r.URL.RawQuery = q.Encode()
						s.clientHandler.GetClientNomenclature(w, r, clientID)
					}))
				}

				// Normalization для проекта
				if s.normalizationHandler != nil {
					projectNormalizationAPI := clientProjectsAPI.Group("/:projectId/normalization")
					{
						// POST /api/clients/:clientId/projects/:projectId/normalization/start
						projectNormalizationAPI.POST("/start", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
							// Добавляем параметры в контекст для использования в handler
							ctx := context.WithValue(r.Context(), "clientId", clientID)
							ctx = context.WithValue(ctx, "projectId", projectID)
							r = r.WithContext(ctx)
							s.normalizationHandler.HandleStartClientProjectNormalization(w, r)
						}))
						// POST /api/clients/:clientId/projects/:projectId/normalization/stop
						projectNormalizationAPI.POST("/stop", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
							// Добавляем параметры в контекст для использования в handler
							ctx := context.WithValue(r.Context(), "clientId", clientID)
							ctx = context.WithValue(ctx, "projectId", projectID)
							r = r.WithContext(ctx)
							s.normalizationHandler.HandleNormalizationStop(w, r)
						}))
						// GET /api/clients/:clientId/projects/:projectId/normalization/status
						projectNormalizationAPI.GET("/status", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
							// Добавляем параметры в контекст для использования в handler
							ctx := context.WithValue(r.Context(), "clientId", clientID)
							ctx = context.WithValue(ctx, "projectId", projectID)
							r = r.WithContext(ctx)
							s.normalizationHandler.HandleGetClientProjectNormalizationStatus(w, r)
						}))
						// GET /api/clients/:clientId/projects/:projectId/normalization/preview-stats
						projectNormalizationAPI.GET("/preview-stats", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
							// Добавляем параметры в контекст для использования в handler
							ctx := context.WithValue(r.Context(), "clientId", clientID)
							ctx = context.WithValue(ctx, "projectId", projectID)
							r = r.WithContext(ctx)
							s.normalizationHandler.HandleGetClientProjectNormalizationPreviewStats(w, r)
						}))
						// GET /api/clients/:clientId/projects/:projectId/normalization/events (SSE)
						projectNormalizationAPI.GET("/events", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
							// Добавляем параметры в контекст для использования в handler
							ctx := context.WithValue(r.Context(), "clientId", clientID)
							ctx = context.WithValue(ctx, "projectId", projectID)
							r = r.WithContext(ctx)
							s.normalizationHandler.HandleNormalizationEvents(w, r)
						}))
						// GET /api/clients/:clientId/projects/:projectId/normalization/groups
						projectNormalizationAPI.GET("/groups", clientProjectIDWrapper(func(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
							// Добавляем параметры в контекст для использования в handler
							ctx := context.WithValue(r.Context(), "clientId", clientID)
							ctx = context.WithValue(ctx, "projectId", projectID)
							r = r.WithContext(ctx)
							s.normalizationHandler.HandleNormalizationGroups(w, r)
						}))
					}
				}
			}
		}
		log.Printf("[Routes] ✓ Clients API routes registered: POST /api/clients, GET /api/clients, GET /api/clients/:clientId, etc.")
	} else {
		log.Printf("⚠ WARNING: clientHandler is nil, Clients API routes will not be registered")
	}

	// Notifications API
	if s.notificationHandler != nil {
		notificationsAPI := api.Group("/notifications")
		{
			notificationsAPI.POST("", httpHandlerToGin(s.notificationHandler.HandleAddNotification))
			notificationsAPI.GET("", httpHandlerToGin(s.notificationHandler.HandleGetNotifications))
			notificationsAPI.POST("/:id/read", httpHandlerToGin(s.notificationHandler.HandleMarkAsRead))
			notificationsAPI.POST("/read-all", httpHandlerToGin(s.notificationHandler.HandleMarkAllAsRead))
			notificationsAPI.GET("/unread-count", httpHandlerToGin(s.notificationHandler.HandleGetUnreadCount))
			notificationsAPI.DELETE("/:id", httpHandlerToGin(s.notificationHandler.HandleDeleteNotification))
		}
		log.Printf("[Routes] ✓ Notifications API routes registered: POST /api/notifications, GET /api/notifications, etc.")
	} else {
		log.Printf("⚠ WARNING: notificationHandler is nil, Notifications API routes will not be registered")
	}

	// Workers API (legacy handlers via adapter)
	if s.workerHandler != nil {
		workersAPI := api.Group("/workers")
		{
			workersAPI.GET("/config", httpHandlerToGin(s.workerHandler.HandleGetWorkerConfig))
			workersAPI.POST("/config/update", httpHandlerToGin(s.workerHandler.HandleUpdateWorkerConfig))
			workersAPI.GET("/providers", httpHandlerToGin(s.workerHandler.HandleGetAvailableProviders))
			workersAPI.GET("/models", httpHandlerToGin(s.workerHandler.HandleGetModels))
			workersAPI.GET("/arliai/status", httpHandlerToGin(s.workerHandler.HandleCheckArliaiConnection))
			workersAPI.GET("/openrouter/status", httpHandlerToGin(s.workerHandler.HandleCheckOpenRouterConnection))
			workersAPI.GET("/huggingface/status", httpHandlerToGin(s.workerHandler.HandleCheckHuggingFaceConnection))
			workersAPI.GET("/orchestrator/strategy", httpHandlerToGin(s.workerHandler.HandleOrchestratorStrategy))
			workersAPI.POST("/orchestrator/strategy", httpHandlerToGin(s.workerHandler.HandleOrchestratorStrategy))
			workersAPI.GET("/orchestrator/stats", httpHandlerToGin(s.workerHandler.HandleOrchestratorStats))
		}
	}
	if s.workerTraceHandler != nil {
		api.GET("/workers/trace", httpHandlerToGin(s.workerTraceHandler.HandleWorkerTraceStream))
	}

	// Counterparties API
	log.Printf("[DEBUG] counterpartyHandler status: %v (nil=%t)", s.counterpartyHandler, s.counterpartyHandler == nil)
	if s.counterpartyHandler != nil {
		counterpartiesAPI := api.Group("/counterparties")
		{
			// GET /api/counterparties/all - получение всех контрагентов клиента
			counterpartiesAPI.GET("/all", httpHandlerToGin(s.counterpartyHandler.HandleGetAllCounterparties))
			// GET /api/counterparties/all/export - экспорт контрагентов
			counterpartiesAPI.GET("/all/export", httpHandlerToGin(s.counterpartyHandler.HandleExportAllCounterparties))
		}
		log.Printf("[Routes] ✓ Counterparties API routes registered: GET /api/counterparties/all, GET /api/counterparties/all/export")
	} else {
		log.Printf("⚠ WARNING: counterpartyHandler is nil, Counterparties API routes will not be registered")
	}

	// Reports API
	if s.reportHandler != nil {
		reportsAPI := api.Group("/reports")
		{
			reportsAPI.POST("/generate-normalization-report", httpHandlerToGin(s.reportHandler.HandleGenerateNormalizationReport))
			reportsAPI.POST("/generate-data-quality-report", httpHandlerToGin(s.reportHandler.HandleGenerateDataQualityReport))
		}
	}

	// GOST API
	if s.gostHandler != nil {
		gostsAPI := api.Group("/gosts")
		{
			// GET /api/gosts - список ГОСТов
			gostsAPI.GET("", s.gostHandler.HandleGetGosts)
			// GET /api/gosts/:id - детали ГОСТа
			gostsAPI.GET("/:id", s.gostHandler.HandleGetGostDetail)
			// GET /api/gosts/number/:number - получение ГОСТа по номеру
			gostsAPI.GET("/number/:number", s.gostHandler.HandleGetGostByNumber)
			// GET /api/gosts/search - поиск ГОСТов
			gostsAPI.GET("/search", s.gostHandler.HandleSearchGosts)
			// POST /api/gosts/import - импорт ГОСТов
			gostsAPI.POST("/import", s.gostHandler.HandleImportGosts)
			// GET /api/gosts/statistics - статистика ГОСТов
			gostsAPI.GET("/statistics", s.gostHandler.HandleGetStatistics)
			// GET /api/gosts/export - экспорт ГОСТов в CSV
			gostsAPI.GET("/export", s.gostHandler.HandleExportGosts)
			// POST /api/gosts/:id/document - загрузка документа ГОСТа
			gostsAPI.POST("/:id/document", s.gostHandler.HandleUploadDocument)
			// GET /api/gosts/:id/document - получение документа ГОСТа
			gostsAPI.GET("/:id/document", s.gostHandler.HandleGetDocument)
		}
		log.Printf("[Routes] ✓ GOST API routes registered")
	} else {
		log.Printf("⚠ WARNING: gostHandler is nil, GOST API routes will not be registered")
	}

	// System API
	if s.systemHandler != nil {
		systemAPI := api.Group("/system")
		{
			// GET /api/system/stats - статистика системы
			systemAPI.GET("/stats", httpHandlerToGin(s.systemHandler.HandleStats))
			// GET /api/system/health - health check системы
			systemAPI.GET("/health", httpHandlerToGin(s.systemHandler.HandleHealth))
			// GET /api/system/performance-metrics - метрики производительности
			systemAPI.GET("/performance-metrics", httpHandlerToGin(s.systemHandler.HandlePerformanceMetrics))
		}
		log.Printf("[Routes] ✓ System API routes registered")
	}

	// System Summary API
	if s.systemSummaryHandler != nil {
		systemSummaryAPI := api.Group("/system/summary")
		{
			// GET /api/system/summary/cache/stats - статистика кэша
			systemSummaryAPI.GET("/cache/stats", httpHandlerToGin(s.systemSummaryHandler.HandleSystemSummaryCacheStats))
			// POST /api/system/summary/cache/invalidate - инвалидация кэша
			systemSummaryAPI.POST("/cache/invalidate", httpHandlerToGin(s.systemSummaryHandler.HandleSystemSummaryCacheInvalidate))
		}
		log.Printf("[Routes] ✓ System Summary API routes registered")
	}

	// Error Metrics API
	if s.errorMetricsHandler != nil {
		errorsAPI := api.Group("/errors")
		{
			// GET /api/errors/metrics - метрики ошибок
			errorsAPI.GET("/metrics", httpHandlerToGin(s.errorMetricsHandler.GetErrorMetrics))
			// GET /api/errors/last - последние ошибки
			errorsAPI.GET("/last", httpHandlerToGin(s.errorMetricsHandler.GetLastErrors))
			// POST /api/errors/reset - сброс метрик ошибок
			errorsAPI.POST("/reset", httpHandlerToGin(s.errorMetricsHandler.ResetErrorMetrics))
			// GET /api/errors/by-endpoint - ошибки по эндпоинтам
			errorsAPI.GET("/by-endpoint", httpHandlerToGin(s.errorMetricsHandler.GetErrorsByEndpoint))
			// GET /api/errors/by-type - ошибки по типу
			errorsAPI.GET("/by-type", httpHandlerToGin(s.errorMetricsHandler.GetErrorsByType))
			// GET /api/errors/by-code - ошибки по HTTP коду
			errorsAPI.GET("/by-code", httpHandlerToGin(s.errorMetricsHandler.GetErrorsByCode))
		}
		log.Printf("[Routes] ✓ Error Metrics API routes registered")
	}
}

// registerLegacyHandlers регистрирует legacy handlers в mux
func (s *Server) registerLegacyHandlers(mux *http.ServeMux) {
	// Клиенты
	mux.HandleFunc("/api/clients", s.handleClients)
	mux.HandleFunc("/api/clients/", s.handleClientRoutes)

	// Экспорт
	s.RegisterExportRoutes(mux)

	// Config routes
	if s.configHandler != nil {
		mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				s.configHandler.HandleGetConfigSafe(w, r)
			} else if r.Method == http.MethodPut || r.Method == http.MethodPost {
				s.configHandler.HandleUpdateConfig(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})
		mux.HandleFunc("/api/config/full", s.configHandler.HandleGetConfig)
		mux.HandleFunc("/api/config/history", s.configHandler.HandleGetConfigHistory)
	}

	// Workers fallback routes
	if s.workerHandler != nil {
		mux.HandleFunc("/api/workers/config", s.workerHandler.HandleGetWorkerConfig)
		mux.HandleFunc("/api/workers/config/update", s.workerHandler.HandleUpdateWorkerConfig)
		mux.HandleFunc("/api/workers/providers", s.workerHandler.HandleGetAvailableProviders)
		mux.HandleFunc("/api/workers/models", s.workerHandler.HandleGetModels)
		mux.HandleFunc("/api/workers/arliai/status", s.workerHandler.HandleCheckArliaiConnection)
		mux.HandleFunc("/api/workers/openrouter/status", s.workerHandler.HandleCheckOpenRouterConnection)
		mux.HandleFunc("/api/workers/huggingface/status", s.workerHandler.HandleCheckHuggingFaceConnection)
		mux.HandleFunc("/api/workers/orchestrator/strategy", s.workerHandler.HandleOrchestratorStrategy)
		mux.HandleFunc("/api/workers/orchestrator/stats", s.workerHandler.HandleOrchestratorStats)
	}
	if s.workerTraceHandler != nil {
		mux.HandleFunc("/api/internal/worker-trace/stream", s.workerTraceHandler.HandleWorkerTraceStream)
	}

	// Reports fallback routes
	if s.reportHandler != nil {
		mux.HandleFunc("/api/reports/generate-normalization-report", s.reportHandler.HandleGenerateNormalizationReport)
		mux.HandleFunc("/api/reports/generate-data-quality-report", s.reportHandler.HandleGenerateDataQualityReport)
	}

	// Dashboard fallback (для совместимости при отключенном Gin)
	if s.dashboardHandler != nil {
		mux.HandleFunc("/api/dashboard/stats", s.dashboardHandler.HandleGetStats)
		mux.HandleFunc("/api/dashboard/overview", s.dashboardHandler.HandleDashboardOverview)
		mux.HandleFunc("/api/dashboard/normalization-status", s.dashboardHandler.HandleGetNormalizationStatus)
		mux.HandleFunc("/api/quality/metrics", s.dashboardHandler.HandleGetQualityMetrics)
	}

	// Quality fallback routes
	if s.qualityHandler != nil {
		// Добавляем fallback для /api/quality/stats
		mux.HandleFunc("/api/quality/stats", func(w http.ResponseWriter, r *http.Request) {
			currentDB := s.db
			if currentDB == nil {
				currentDB = s.normalizedDB
			}
			s.qualityHandler.HandleQualityStats(w, r, currentDB, s.currentNormalizedDBPath)
		})
	}

	// Counterparties fallback routes
	if s.counterpartyHandler != nil {
		mux.HandleFunc("/api/counterparties/all", s.counterpartyHandler.HandleGetAllCounterparties)
		mux.HandleFunc("/api/counterparties/all/export", s.counterpartyHandler.HandleExportAllCounterparties)
	}

	// Monitoring fallback (для SSE и REST метрик)
	if s.monitoringHandler != nil {
		mux.HandleFunc("/api/monitoring/metrics", s.monitoringHandler.HandleMonitoringMetrics)
		mux.HandleFunc("/api/monitoring/providers/stream", s.monitoringHandler.HandleMonitoringProvidersStream)
		mux.HandleFunc("/api/monitoring/providers", s.monitoringHandler.HandleMonitoringProviders)
		mux.HandleFunc("/api/monitoring/events", s.monitoringHandler.HandleMonitoringEvents)
		mux.HandleFunc("/api/monitoring/history", s.monitoringHandler.HandleMonitoringHistory)
		mux.HandleFunc("/api/monitoring/cache", s.monitoringHandler.HandleMonitoringCache)
		mux.HandleFunc("/api/monitoring/ai", s.monitoringHandler.HandleMonitoringAI)
	}

	// Logs fallback
	if s.logsHandler != nil {
		mux.HandleFunc("/api/logs/client-error", s.logsHandler.HandleClientError)
	}

	// Добавьте другие legacy handlers по мере необходимости
}
