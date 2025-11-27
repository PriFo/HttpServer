package routes

import (
	"net/http"

	"httpserver/server/handlers"
)

// ClassificationLegacyHandlers содержит legacy handlers для классификации (КПВЭД, ОКПД2)
type ClassificationLegacyHandlers struct {
	// Handler из server/handlers
	Handler *handlers.ClassificationHandler
	// Legacy handlers для fallback
	HandleKpvedHierarchy              http.HandlerFunc
	HandleKpvedSearch                 http.HandlerFunc
	HandleKpvedStats                  http.HandlerFunc
	HandleKpvedLoad                   http.HandlerFunc
	HandleKpvedLoadFromFile            http.HandlerFunc
	HandleKpvedClassifyTest            http.HandlerFunc
	HandleKpvedClassifyHierarchical    http.HandlerFunc
	HandleResetClassification         http.HandlerFunc
	HandleMarkIncorrect               http.HandlerFunc
	HandleMarkCorrect                 http.HandlerFunc
	HandleKpvedReclassify             http.HandlerFunc
	HandleKpvedReclassifyHierarchical http.HandlerFunc
	HandleKpvedCurrentTasks           http.HandlerFunc
	HandleResetAllClassification      http.HandlerFunc
	HandleResetByCode                 http.HandlerFunc
	HandleResetLowConfidence          http.HandlerFunc
	HandleKpvedWorkersStatus           http.HandlerFunc
	HandleKpvedWorkersStop             http.HandlerFunc
	HandleKpvedWorkersResume           http.HandlerFunc
	HandleKpvedStatsGeneral           http.HandlerFunc
	HandleKpvedStatsByCategory        http.HandlerFunc
	HandleKpvedStatsIncorrect         http.HandlerFunc
	HandleModelsBenchmark             http.HandlerFunc
	HandleOkpd2Hierarchy              http.HandlerFunc
	HandleOkpd2Search                 http.HandlerFunc
	HandleOkpd2Stats                  http.HandlerFunc
	HandleOkpd2LoadFromFile           http.HandlerFunc
	HandleOkpd2Clear                  http.HandlerFunc
	// Новые маршруты для классификации
	HandleClassifyItem                http.HandlerFunc
	HandleClassifyItemDirect          http.HandlerFunc
	HandleGetStrategies               http.HandlerFunc
	HandleConfigureStrategy           http.HandlerFunc
	HandleGetClientStrategies         http.HandlerFunc
	HandleCreateOrUpdateClientStrategy http.HandlerFunc
	HandleGetAvailableStrategies      http.HandlerFunc
	HandleGetClassifiers              http.HandlerFunc
	HandleGetClassifiersByProjectType http.HandlerFunc
	HandleClassificationOptimizationStats http.HandlerFunc
}

// RegisterClassificationLegacyRoutes регистрирует legacy маршруты для классификации (КПВЭД, ОКПД2)
func RegisterClassificationLegacyRoutes(mux *http.ServeMux, h *ClassificationLegacyHandlers) {
	// Используем handler если доступен
	if h.Handler != nil {
		// КПВЭД endpoints
		mux.HandleFunc("/api/kpved/hierarchy", h.Handler.HandleKpvedHierarchy)
		mux.HandleFunc("/api/kpved/search", h.Handler.HandleKpvedSearch)
		mux.HandleFunc("/api/kpved/stats", h.Handler.HandleKpvedStats)
		mux.HandleFunc("/api/kpved/load", h.Handler.HandleKpvedLoad)
		mux.HandleFunc("/api/kpved/load-from-file", h.Handler.HandleKpvedLoadFromFile)
		mux.HandleFunc("/api/kpved/classify-test", h.Handler.HandleKpvedClassifyTest)
		mux.HandleFunc("/api/kpved/classify-hierarchical", h.Handler.HandleKpvedClassifyHierarchical)
		mux.HandleFunc("/api/kpved/reset", h.Handler.HandleResetClassification)
		mux.HandleFunc("/api/kpved/mark-incorrect", h.Handler.HandleMarkIncorrect)
		mux.HandleFunc("/api/kpved/mark-correct", h.Handler.HandleMarkCorrect)
		mux.HandleFunc("/api/kpved/reclassify", h.Handler.HandleKpvedReclassify)
		mux.HandleFunc("/api/kpved/reclassify-hierarchical", h.Handler.HandleKpvedReclassifyHierarchical)
		mux.HandleFunc("/api/kpved/current-tasks", h.Handler.HandleKpvedCurrentTasks)
		mux.HandleFunc("/api/kpved/reset-all", h.Handler.HandleResetAllClassification)
		mux.HandleFunc("/api/kpved/reset-by-code", h.Handler.HandleResetByCode)
		mux.HandleFunc("/api/kpved/reset-low-confidence", h.Handler.HandleResetLowConfidence)
		mux.HandleFunc("/api/kpved/workers/status", h.Handler.HandleKpvedWorkersStatus)
		mux.HandleFunc("/api/kpved/workers/stop", h.Handler.HandleKpvedWorkersStop)
		mux.HandleFunc("/api/kpved/workers/resume", h.Handler.HandleKpvedWorkersResume)
		mux.HandleFunc("/api/kpved/workers/start", h.Handler.HandleKpvedWorkersResume)
		mux.HandleFunc("/api/kpved/stats/classification", h.Handler.HandleKpvedStatsGeneral)
		mux.HandleFunc("/api/kpved/stats/by-category", h.Handler.HandleKpvedStatsByCategory)
		mux.HandleFunc("/api/kpved/stats/incorrect", h.Handler.HandleKpvedStatsIncorrect)
		mux.HandleFunc("/api/models/benchmark", h.Handler.HandleModelsBenchmark)
		// ОКПД2 endpoints
		mux.HandleFunc("/api/okpd2/hierarchy", h.Handler.HandleOkpd2Hierarchy)
		mux.HandleFunc("/api/okpd2/search", h.Handler.HandleOkpd2Search)
		mux.HandleFunc("/api/okpd2/stats", h.Handler.HandleOkpd2Stats)
		mux.HandleFunc("/api/okpd2/load-from-file", h.Handler.HandleOkpd2LoadFromFile)
		mux.HandleFunc("/api/okpd2/clear", h.Handler.HandleOkpd2Clear)
		// Новые маршруты для классификации
		mux.HandleFunc("/api/classification/classify", h.Handler.HandleClassifyItem)
		mux.HandleFunc("/api/classification/classify-item", h.Handler.HandleClassifyItemDirect)
		mux.HandleFunc("/api/classification/strategies", h.Handler.HandleGetStrategies)
		mux.HandleFunc("/api/classification/strategies/configure", h.Handler.HandleConfigureStrategy)
		mux.HandleFunc("/api/classification/strategies/client", h.Handler.HandleGetClientStrategies)
		mux.HandleFunc("/api/classification/strategies/create", h.Handler.HandleCreateOrUpdateClientStrategy)
		mux.HandleFunc("/api/classification/available", h.Handler.HandleGetAvailableStrategies)
		mux.HandleFunc("/api/classification/classifiers", h.Handler.HandleGetClassifiers)
		mux.HandleFunc("/api/classification/classifiers/by-project-type", h.Handler.HandleGetClassifiersByProjectType)
		mux.HandleFunc("/api/classification/optimization-stats", h.Handler.HandleClassificationOptimizationStats)
		return
	}

	// Fallback к legacy handlers
	if h.HandleKpvedHierarchy != nil {
		mux.HandleFunc("/api/kpved/hierarchy", h.HandleKpvedHierarchy)
	}
	if h.HandleKpvedSearch != nil {
		mux.HandleFunc("/api/kpved/search", h.HandleKpvedSearch)
	}
	if h.HandleKpvedStats != nil {
		mux.HandleFunc("/api/kpved/stats", h.HandleKpvedStats)
	}
	if h.HandleKpvedLoad != nil {
		mux.HandleFunc("/api/kpved/load", h.HandleKpvedLoad)
	}
	if h.HandleKpvedLoadFromFile != nil {
		mux.HandleFunc("/api/kpved/load-from-file", h.HandleKpvedLoadFromFile)
	}
	if h.HandleKpvedClassifyTest != nil {
		mux.HandleFunc("/api/kpved/classify-test", h.HandleKpvedClassifyTest)
	}
	if h.HandleKpvedClassifyHierarchical != nil {
		mux.HandleFunc("/api/kpved/classify-hierarchical", h.HandleKpvedClassifyHierarchical)
	}
	if h.HandleResetClassification != nil {
		mux.HandleFunc("/api/kpved/reset", h.HandleResetClassification)
	}
	if h.HandleMarkIncorrect != nil {
		mux.HandleFunc("/api/kpved/mark-incorrect", h.HandleMarkIncorrect)
	}
	if h.HandleMarkCorrect != nil {
		mux.HandleFunc("/api/kpved/mark-correct", h.HandleMarkCorrect)
	}
	if h.HandleKpvedReclassify != nil {
		mux.HandleFunc("/api/kpved/reclassify", h.HandleKpvedReclassify)
	}
	if h.HandleKpvedReclassifyHierarchical != nil {
		mux.HandleFunc("/api/kpved/reclassify-hierarchical", h.HandleKpvedReclassifyHierarchical)
	}
	if h.HandleKpvedCurrentTasks != nil {
		mux.HandleFunc("/api/kpved/current-tasks", h.HandleKpvedCurrentTasks)
	}
	if h.HandleResetAllClassification != nil {
		mux.HandleFunc("/api/kpved/reset-all", h.HandleResetAllClassification)
	}
	if h.HandleResetByCode != nil {
		mux.HandleFunc("/api/kpved/reset-by-code", h.HandleResetByCode)
	}
	if h.HandleResetLowConfidence != nil {
		mux.HandleFunc("/api/kpved/reset-low-confidence", h.HandleResetLowConfidence)
	}
	if h.HandleKpvedWorkersStatus != nil {
		mux.HandleFunc("/api/kpved/workers/status", h.HandleKpvedWorkersStatus)
	}
	if h.HandleKpvedWorkersStop != nil {
		mux.HandleFunc("/api/kpved/workers/stop", h.HandleKpvedWorkersStop)
	}
	if h.HandleKpvedWorkersResume != nil {
		mux.HandleFunc("/api/kpved/workers/resume", h.HandleKpvedWorkersResume)
		mux.HandleFunc("/api/kpved/workers/start", h.HandleKpvedWorkersResume)
	}
	if h.HandleKpvedStatsGeneral != nil {
		mux.HandleFunc("/api/kpved/stats/classification", h.HandleKpvedStatsGeneral)
	}
	if h.HandleKpvedStatsByCategory != nil {
		mux.HandleFunc("/api/kpved/stats/by-category", h.HandleKpvedStatsByCategory)
	}
	if h.HandleKpvedStatsIncorrect != nil {
		mux.HandleFunc("/api/kpved/stats/incorrect", h.HandleKpvedStatsIncorrect)
	}
	if h.HandleModelsBenchmark != nil {
		mux.HandleFunc("/api/models/benchmark", h.HandleModelsBenchmark)
	}
	// ОКПД2 endpoints
	if h.HandleOkpd2Hierarchy != nil {
		mux.HandleFunc("/api/okpd2/hierarchy", h.HandleOkpd2Hierarchy)
	}
	if h.HandleOkpd2Search != nil {
		mux.HandleFunc("/api/okpd2/search", h.HandleOkpd2Search)
	}
	if h.HandleOkpd2Stats != nil {
		mux.HandleFunc("/api/okpd2/stats", h.HandleOkpd2Stats)
	}
	if h.HandleOkpd2LoadFromFile != nil {
		mux.HandleFunc("/api/okpd2/load-from-file", h.HandleOkpd2LoadFromFile)
	}
	if h.HandleOkpd2Clear != nil {
		mux.HandleFunc("/api/okpd2/clear", h.HandleOkpd2Clear)
	}
	// Новые маршруты для классификации
	if h.HandleClassifyItem != nil {
		mux.HandleFunc("/api/classification/classify", h.HandleClassifyItem)
	}
	if h.HandleClassifyItemDirect != nil {
		mux.HandleFunc("/api/classification/classify-item", h.HandleClassifyItemDirect)
	}
	if h.HandleGetStrategies != nil {
		mux.HandleFunc("/api/classification/strategies", h.HandleGetStrategies)
	}
	if h.HandleConfigureStrategy != nil {
		mux.HandleFunc("/api/classification/strategies/configure", h.HandleConfigureStrategy)
	}
	if h.HandleGetClientStrategies != nil {
		mux.HandleFunc("/api/classification/strategies/client", h.HandleGetClientStrategies)
	}
	if h.HandleCreateOrUpdateClientStrategy != nil {
		mux.HandleFunc("/api/classification/strategies/create", h.HandleCreateOrUpdateClientStrategy)
	}
	if h.HandleGetAvailableStrategies != nil {
		mux.HandleFunc("/api/classification/available", h.HandleGetAvailableStrategies)
	}
	if h.HandleGetClassifiers != nil {
		mux.HandleFunc("/api/classification/classifiers", h.HandleGetClassifiers)
	}
	if h.HandleGetClassifiersByProjectType != nil {
		mux.HandleFunc("/api/classification/classifiers/by-project-type", h.HandleGetClassifiersByProjectType)
	}
	if h.HandleClassificationOptimizationStats != nil {
		mux.HandleFunc("/api/classification/optimization-stats", h.HandleClassificationOptimizationStats)
	}
}

