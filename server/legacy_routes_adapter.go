package server

import (
	"log"

	"httpserver/internal/api/routes"

	"github.com/gin-gonic/gin"
)

// newLegacyRouteAdapter создает binder для регистрации legacy-маршрутов в Gin.
func newLegacyRouteAdapter(s *Server) routes.LegacyRouteBinder {
	if s == nil {
		return nil
	}
	return &legacyRouteAdapter{server: s}
}

type legacyRouteAdapter struct {
	server *Server
}

func (a *legacyRouteAdapter) RegisterLegacy(group *gin.RouterGroup) {
	if group == nil {
		log.Printf("[RegisterLegacy] ⚠ group is nil")
		return
	}

	log.Printf("[RegisterLegacy] Регистрация legacy routes в группе: %s", group.BasePath())

	a.registerBenchmarks(group.Group("/benchmarks"))
	a.registerSimilarity(group.Group("/similarity"))
	a.registerDatabases(group.Group("/databases"))
	a.registerExport(group.Group("/export"))
	a.registerClassification(group.Group("/classification"))
	a.registerModels(group.Group("/models"))
	a.registerPatterns(group.Group("/patterns"))
	a.registerVersions(group.Group("/versions"))
	a.registerOkpd2(group.Group("/okpd2"))
	a.registerKpved(group.Group("/kpved"))
	a.registerGisp(group.Group("/gisp"))
	a.registerPipeline(group.Group("/pipeline"))
	a.registerReclassification(group.Group("/reclassification"))
	a.registerDuplicateDetection(group.Group("/duplicate-detection"))
	a.registerKpvedReclassify(group.Group("/kpved-reclassify"))

	log.Printf("[RegisterLegacy] ✓ Все legacy routes зарегистрированы")
}

func (a *legacyRouteAdapter) registerBenchmarks(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Метод всегда существует, так как он определен в server/benchmarks.go или server/handlers/legacy/benchmarks_legacy.go
	group.POST("/manufacturers/import", httpHandlerToGin(a.server.handleImportManufacturers))
}

func (a *legacyRouteAdapter) registerSimilarity(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Основные операции сравнения
	group.POST("/compare", httpHandlerToGin(a.server.handleSimilarityCompare))
	group.POST("/batch", httpHandlerToGin(a.server.handleSimilarityBatch))
	group.GET("/weights", httpHandlerToGin(a.server.handleSimilarityWeights))
	group.POST("/weights", httpHandlerToGin(a.server.handleSimilarityWeights))
	group.POST("/evaluate", httpHandlerToGin(a.server.handleSimilarityEvaluate))
	group.GET("/stats", httpHandlerToGin(a.server.handleSimilarityStats))
	group.POST("/cache/clear", httpHandlerToGin(a.server.handleSimilarityClearCache))

	// Обучение и оптимизация
	group.POST("/learn", httpHandlerToGin(a.server.handleSimilarityLearn))
	group.POST("/optimal-threshold", httpHandlerToGin(a.server.handleSimilarityOptimalThreshold))
	group.POST("/cross-validate", httpHandlerToGin(a.server.handleSimilarityCrossValidate))

	// Анализ и поиск
	group.POST("/analyze", httpHandlerToGin(a.server.handleSimilarityAnalyze))
	group.POST("/find-similar", httpHandlerToGin(a.server.handleSimilarityFindSimilar))
	group.POST("/compare-weights", httpHandlerToGin(a.server.handleSimilarityCompareWeights))
	group.POST("/breakdown", httpHandlerToGin(a.server.handleSimilarityBreakdown))

	// Экспорт и импорт
	group.POST("/export", httpHandlerToGin(a.server.handleSimilarityExport))
	group.POST("/import", httpHandlerToGin(a.server.handleSimilarityImport))

	// Производительность
	group.GET("/performance", httpHandlerToGin(a.server.handleSimilarityPerformance))
	group.POST("/performance/reset", httpHandlerToGin(a.server.handleSimilarityPerformanceReset))
}

func (a *legacyRouteAdapter) registerDatabases(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		log.Printf("[registerDatabases] ⚠ Пропуск регистрации: group=%v, adapter=%v, server=%v", group != nil, a != nil, a != nil && a.server != nil)
		return
	}

	log.Printf("[registerDatabases] Регистрация routes в группе: %s", group.BasePath())

	// GET /pending - УБРАНО, так как регистрируется в новом DatabaseHandler
	// group.GET("/pending", httpHandlerToGin(a.server.handlePendingDatabases))
	group.POST("/pending/cleanup", httpHandlerToGin(a.server.handleCleanupPendingDatabases))

	pendingGroup := group.Group("/pending")
	{
		pendingGroup.GET("/:id", httpHandlerToGin(a.server.handlePendingDatabaseRoutes))
		pendingGroup.DELETE("/:id", httpHandlerToGin(a.server.handlePendingDatabaseRoutes))
		pendingGroup.POST("/:id/index", httpHandlerToGin(a.server.handlePendingDatabaseRoutes))
		pendingGroup.POST("/:id/bind", httpHandlerToGin(a.server.handlePendingDatabaseRoutes))
	}

	log.Printf("[registerDatabases] ✓ Routes зарегистрированы: POST /pending/cleanup, GET /pending/:id, etc.")
}

func (a *legacyRouteAdapter) registerExport(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	group.GET("/data", httpHandlerToGin(a.server.handleExportData))
	group.POST("/data", httpHandlerToGin(a.server.handleExportData))
	group.GET("/report", httpHandlerToGin(a.server.handleExportReport))
	group.GET("/statistics", httpHandlerToGin(a.server.handleExportStatistics))
	group.GET("/stages/progress", httpHandlerToGin(a.server.handleStagesProgress))
	group.GET("/progress", httpHandlerToGin(a.server.handleStagesProgress))
}

func (a *legacyRouteAdapter) registerPatterns(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Pattern detection endpoints
	group.POST("/detect", httpHandlerToGin(a.server.handlePatternDetect))
	group.POST("/suggest", httpHandlerToGin(a.server.handlePatternSuggest))
	group.POST("/test-batch", httpHandlerToGin(a.server.handlePatternTestBatch))
}

func (a *legacyRouteAdapter) registerOkpd2(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// OKPD2 endpoints
	group.GET("/hierarchy", httpHandlerToGin(a.server.handleOkpd2Hierarchy))
	group.GET("/search", httpHandlerToGin(a.server.handleOkpd2Search))
	group.GET("/stats", httpHandlerToGin(a.server.handleOkpd2Stats))
	group.POST("/load", httpHandlerToGin(a.server.handleOkpd2LoadFromFile))
	group.POST("/clear", httpHandlerToGin(a.server.handleOkpd2Clear))
}

func (a *legacyRouteAdapter) registerClassification(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Classifiers endpoints - УБРАНО, так как регистрируется в новом ClassificationHandler
	// group.GET("/classifiers", httpHandlerToGin(a.server.handleGetClassifiers))
	// group.GET("/classifiers/by-project-type", httpHandlerToGin(a.server.handleGetClassifiersByProjectType))

	// Classification management endpoints
	group.POST("/reset", httpHandlerToGin(a.server.handleResetClassification))
	group.POST("/reset-all", httpHandlerToGin(a.server.handleResetAllClassification))
}

func (a *legacyRouteAdapter) registerModels(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Models benchmark endpoint
	group.GET("/benchmark", httpHandlerToGin(a.server.handleModelsBenchmark))
	group.POST("/benchmark", httpHandlerToGin(a.server.handleModelsBenchmark))
}

func (a *legacyRouteAdapter) registerGisp(group *gin.RouterGroup) {
	if group == nil {
		return
	}

	// GISP nomenclatures endpoints
	// Методы определены в server/handlers/legacy/gisp_nomenclatures_legacy.go
	group.POST("/nomenclatures/import", httpHandlerToGin(a.server.handleImportGISPNomenclatures))
	group.GET("/nomenclatures", httpHandlerToGin(a.server.handleGetGISPNomenclatures))
	group.GET("/nomenclatures/:id", httpHandlerToGin(a.server.handleGetGISPNomenclatureDetail))
	group.GET("/reference-books", httpHandlerToGin(a.server.handleGetGISPReferenceBooks))
	group.GET("/reference-books/search", httpHandlerToGin(a.server.handleSearchGISPReferenceBook))
	group.GET("/statistics", httpHandlerToGin(a.server.handleGetGISPStatistics))
}

func (a *legacyRouteAdapter) registerPipeline(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Pipeline endpoints
	group.GET("/stats", httpHandlerToGin(a.server.handlePipelineStats))
	group.GET("/stages/:stage/details", httpHandlerToGin(a.server.handleStageDetails))
	group.GET("/export", httpHandlerToGin(a.server.handleExport))
}

func (a *legacyRouteAdapter) registerReclassification(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Reclassification endpoints
	group.POST("/start", httpHandlerToGin(a.server.handleReclassificationStart))
	group.GET("/events", httpHandlerToGin(a.server.handleReclassificationEvents))
	group.GET("/status", httpHandlerToGin(a.server.handleReclassificationStatus))
	group.POST("/stop", httpHandlerToGin(a.server.handleReclassificationStop))
}

func (a *legacyRouteAdapter) registerDuplicateDetection(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Duplicate detection endpoints
	group.POST("/detect", httpHandlerToGin(a.server.handleDuplicateDetection))
	group.GET("/status", httpHandlerToGin(a.server.handleDuplicateDetectionStatus))
}

func (a *legacyRouteAdapter) registerKpvedReclassify(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// KPVED reclassification endpoints
	group.POST("/hierarchical", httpHandlerToGin(a.server.handleKpvedReclassifyHierarchical))
	// handleClassificationError - это внутренняя функция, не HTTP handler
}

func (a *legacyRouteAdapter) registerKpved(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// KPVED endpoints
	group.GET("/hierarchy", httpHandlerToGin(a.server.handleKpvedHierarchy))
	group.GET("/search", httpHandlerToGin(a.server.handleKpvedSearch))
	group.GET("/stats", httpHandlerToGin(a.server.handleKpvedStats))
	group.POST("/load", httpHandlerToGin(a.server.handleKpvedLoad))
	group.POST("/load-from-file", httpHandlerToGin(a.server.handleKpvedLoadFromFile))

	// KPVED Workers endpoints
	group.GET("/workers/status", httpHandlerToGin(a.server.handleKpvedWorkersStatus))
	group.POST("/workers/stop", httpHandlerToGin(a.server.handleKpvedWorkersStop))
	group.POST("/workers/resume", httpHandlerToGin(a.server.handleKpvedWorkersResume))
}

func (a *legacyRouteAdapter) registerVersions(group *gin.RouterGroup) {
	if group == nil || a == nil || a.server == nil {
		return
	}

	// Versions/Normalization endpoints
	group.POST("/start", httpHandlerToGin(a.server.handleStartNormalization))
	group.POST("/apply-patterns", httpHandlerToGin(a.server.handleApplyPatterns))
	group.POST("/apply-ai", httpHandlerToGin(a.server.handleApplyAI))
	group.GET("/session-history", httpHandlerToGin(a.server.handleGetSessionHistory))
	group.POST("/revert-stage", httpHandlerToGin(a.server.handleRevertStage))
}
