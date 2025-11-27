package routes

import (
	"net/http"

	"httpserver/internal/api/handlers/classification"
)

// ClassificationHandlers содержит handlers для классификации
type ClassificationHandlers struct {
	// Новый handler из internal/api/handlers/classification
	NewHandler *classification.Handler
	// Старый handler из server/handlers (через интерфейс)
	OldHandler interface {
		HandleKpvedHierarchy(http.ResponseWriter, *http.Request)
		HandleKpvedSearch(http.ResponseWriter, *http.Request)
		HandleKpvedStats(http.ResponseWriter, *http.Request)
		HandleKpvedLoad(http.ResponseWriter, *http.Request)
		HandleKpvedLoadFromFile(http.ResponseWriter, *http.Request)
		HandleKpvedClassifyTest(http.ResponseWriter, *http.Request)
		HandleKpvedClassifyHierarchical(http.ResponseWriter, *http.Request)
		HandleResetClassification(http.ResponseWriter, *http.Request)
		HandleMarkIncorrect(http.ResponseWriter, *http.Request)
		HandleMarkCorrect(http.ResponseWriter, *http.Request)
		HandleKpvedReclassify(http.ResponseWriter, *http.Request)
		HandleKpvedReclassifyHierarchical(http.ResponseWriter, *http.Request)
		HandleKpvedCurrentTasks(http.ResponseWriter, *http.Request)
		HandleResetAllClassification(http.ResponseWriter, *http.Request)
		HandleResetByCode(http.ResponseWriter, *http.Request)
		HandleResetLowConfidence(http.ResponseWriter, *http.Request)
		HandleKpvedWorkersStatus(http.ResponseWriter, *http.Request)
		HandleKpvedWorkersStop(http.ResponseWriter, *http.Request)
		HandleKpvedWorkersResume(http.ResponseWriter, *http.Request)
		HandleKpvedStatsGeneral(http.ResponseWriter, *http.Request)
		HandleKpvedStatsByCategory(http.ResponseWriter, *http.Request)
		HandleKpvedStatsIncorrect(http.ResponseWriter, *http.Request)
		HandleModelsBenchmark(http.ResponseWriter, *http.Request)
		HandleOkpd2Hierarchy(http.ResponseWriter, *http.Request)
		HandleOkpd2Search(http.ResponseWriter, *http.Request)
		HandleOkpd2Stats(http.ResponseWriter, *http.Request)
		HandleOkpd2LoadFromFile(http.ResponseWriter, *http.Request)
		HandleOkpd2Clear(http.ResponseWriter, *http.Request)
		HandleClassifyItem(http.ResponseWriter, *http.Request)
		HandleClassifyItemDirect(http.ResponseWriter, *http.Request)
		HandleGetStrategies(http.ResponseWriter, *http.Request)
		HandleConfigureStrategy(http.ResponseWriter, *http.Request)
		HandleGetClientStrategies(http.ResponseWriter, *http.Request)
		HandleCreateOrUpdateClientStrategy(http.ResponseWriter, *http.Request)
		HandleGetAvailableStrategies(http.ResponseWriter, *http.Request)
		HandleGetClassifiers(http.ResponseWriter, *http.Request)
		HandleGetClassifiersByProjectType(http.ResponseWriter, *http.Request)
		HandleClassificationOptimizationStats(http.ResponseWriter, *http.Request)
	}
	// Legacy handlers для fallback
	HandleKpvedHierarchy            http.HandlerFunc
	HandleKpvedSearch               http.HandlerFunc
	HandleKpvedStats                http.HandlerFunc
	HandleKpvedLoad                 http.HandlerFunc
	HandleKpvedLoadFromFile         http.HandlerFunc
	HandleKpvedClassifyTest         http.HandlerFunc
	HandleKpvedClassifyHierarchical http.HandlerFunc
	HandleResetClassification       http.HandlerFunc
	HandleMarkIncorrect             http.HandlerFunc
	HandleMarkCorrect               http.HandlerFunc
	HandleKpvedReclassify           http.HandlerFunc
	HandleKpvedReclassifyHierarchical http.HandlerFunc
	HandleKpvedCurrentTasks         http.HandlerFunc
	HandleResetAllClassification    http.HandlerFunc
	HandleResetByCode               http.HandlerFunc
	HandleResetLowConfidence        http.HandlerFunc
	HandleKpvedWorkersStatus        http.HandlerFunc
	HandleKpvedWorkersStop          http.HandlerFunc
	HandleKpvedWorkersResume        http.HandlerFunc
	HandleKpvedStatsGeneral         http.HandlerFunc
	HandleKpvedStatsByCategory      http.HandlerFunc
	HandleKpvedStatsIncorrect       http.HandlerFunc
	HandleModelsBenchmark           http.HandlerFunc
	HandleOkpd2Hierarchy            http.HandlerFunc
	HandleOkpd2Search               http.HandlerFunc
	HandleOkpd2Stats                http.HandlerFunc
	HandleOkpd2LoadFromFile         http.HandlerFunc
	HandleOkpd2Clear                http.HandlerFunc
	HandleClassifyItem              http.HandlerFunc
	HandleClassifyItemDirect        http.HandlerFunc
	HandleGetStrategies             http.HandlerFunc
	HandleConfigureStrategy         http.HandlerFunc
	HandleGetClientStrategies       http.HandlerFunc
	HandleCreateOrUpdateClientStrategy http.HandlerFunc
	HandleGetAvailableStrategies    http.HandlerFunc
	HandleGetClassifiers            http.HandlerFunc
	HandleGetClassifiersByProjectType http.HandlerFunc
	HandleClassificationOptimizationStats http.HandlerFunc
}

// RegisterClassificationRoutes регистрирует маршруты для classification
func RegisterClassificationRoutes(mux *http.ServeMux, h *ClassificationHandlers) {
	// Используем новый handler если доступен
	if h.NewHandler != nil {
		mux.HandleFunc("/api/classification/classify", h.NewHandler.HandleClassifyEntity)
		mux.HandleFunc("/api/classification/entity/", h.NewHandler.HandleGetClassificationByEntity)
		mux.HandleFunc("/api/classification/history", h.NewHandler.HandleGetClassificationHistory)
		mux.HandleFunc("/api/classification/stats", h.NewHandler.HandleGetClassificationStatistics)
		mux.HandleFunc("/api/classification/hierarchical", h.NewHandler.HandleClassifyHierarchical)
		mux.HandleFunc("/api/classification/", func(w http.ResponseWriter, r *http.Request) {
			h.NewHandler.HandleGetClassification(w, r)
		})
	}

	// Используем старый handler если доступен
	if h.OldHandler != nil {
		// КПВЭД endpoints
		mux.HandleFunc("/api/kpved/hierarchy", h.OldHandler.HandleKpvedHierarchy)
		mux.HandleFunc("/api/kpved/search", h.OldHandler.HandleKpvedSearch)
		mux.HandleFunc("/api/kpved/stats", h.OldHandler.HandleKpvedStats)
		mux.HandleFunc("/api/kpved/load", h.OldHandler.HandleKpvedLoad)
		mux.HandleFunc("/api/kpved/load-from-file", h.OldHandler.HandleKpvedLoadFromFile)
		mux.HandleFunc("/api/kpved/classify-test", h.OldHandler.HandleKpvedClassifyTest)
		mux.HandleFunc("/api/kpved/classify-hierarchical", h.OldHandler.HandleKpvedClassifyHierarchical)
		mux.HandleFunc("/api/kpved/reset", h.OldHandler.HandleResetClassification)
		mux.HandleFunc("/api/kpved/mark-incorrect", h.OldHandler.HandleMarkIncorrect)
		mux.HandleFunc("/api/kpved/mark-correct", h.OldHandler.HandleMarkCorrect)
		mux.HandleFunc("/api/kpved/reclassify", h.OldHandler.HandleKpvedReclassify)
		mux.HandleFunc("/api/kpved/reclassify-hierarchical", h.OldHandler.HandleKpvedReclassifyHierarchical)
		mux.HandleFunc("/api/kpved/current-tasks", h.OldHandler.HandleKpvedCurrentTasks)
		mux.HandleFunc("/api/kpved/reset-all", h.OldHandler.HandleResetAllClassification)
		mux.HandleFunc("/api/kpved/reset-by-code", h.OldHandler.HandleResetByCode)
		mux.HandleFunc("/api/kpved/reset-low-confidence", h.OldHandler.HandleResetLowConfidence)
		mux.HandleFunc("/api/kpved/workers/status", h.OldHandler.HandleKpvedWorkersStatus)
		mux.HandleFunc("/api/kpved/workers/stop", h.OldHandler.HandleKpvedWorkersStop)
		mux.HandleFunc("/api/kpved/workers/resume", h.OldHandler.HandleKpvedWorkersResume)
		mux.HandleFunc("/api/kpved/workers/start", h.OldHandler.HandleKpvedWorkersResume)
		mux.HandleFunc("/api/kpved/stats/classification", h.OldHandler.HandleKpvedStatsGeneral)
		mux.HandleFunc("/api/kpved/stats/by-category", h.OldHandler.HandleKpvedStatsByCategory)
		mux.HandleFunc("/api/kpved/stats/incorrect", h.OldHandler.HandleKpvedStatsIncorrect)
		mux.HandleFunc("/api/models/benchmark", h.OldHandler.HandleModelsBenchmark)
		// ОКПД2 endpoints
		mux.HandleFunc("/api/okpd2/hierarchy", h.OldHandler.HandleOkpd2Hierarchy)
		mux.HandleFunc("/api/okpd2/search", h.OldHandler.HandleOkpd2Search)
		mux.HandleFunc("/api/okpd2/stats", h.OldHandler.HandleOkpd2Stats)
		mux.HandleFunc("/api/okpd2/load-from-file", h.OldHandler.HandleOkpd2LoadFromFile)
		mux.HandleFunc("/api/okpd2/clear", h.OldHandler.HandleOkpd2Clear)
		// Общие classification endpoints
		mux.HandleFunc("/api/classification/classify", h.OldHandler.HandleClassifyItem)
		mux.HandleFunc("/api/classification/classify-item", h.OldHandler.HandleClassifyItemDirect)
		mux.HandleFunc("/api/classification/strategies", h.OldHandler.HandleGetStrategies)
		mux.HandleFunc("/api/classification/strategies/configure", h.OldHandler.HandleConfigureStrategy)
		mux.HandleFunc("/api/classification/strategies/client", h.OldHandler.HandleGetClientStrategies)
		mux.HandleFunc("/api/classification/strategies/create", h.OldHandler.HandleCreateOrUpdateClientStrategy)
		mux.HandleFunc("/api/classification/available", h.OldHandler.HandleGetAvailableStrategies)
		mux.HandleFunc("/api/classification/classifiers", h.OldHandler.HandleGetClassifiers)
		mux.HandleFunc("/api/classification/classifiers/by-project-type", h.OldHandler.HandleGetClassifiersByProjectType)
		mux.HandleFunc("/api/classification/optimization-stats", h.OldHandler.HandleClassificationOptimizationStats)
	} else {
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
}

