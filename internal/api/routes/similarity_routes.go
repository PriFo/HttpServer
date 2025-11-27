package routes

import (
	"net/http"
)

// SimilarityHandlers содержит handlers для работы с алгоритмами схожести
type SimilarityHandlers struct {
	SimilarityHandler interface {
		HandleSimilarityCompare(http.ResponseWriter, *http.Request)
		HandleSimilarityBatch(http.ResponseWriter, *http.Request)
		HandleSimilarityWeights(http.ResponseWriter, *http.Request)
		HandleSimilarityEvaluate(http.ResponseWriter, *http.Request)
		HandleSimilarityStats(http.ResponseWriter, *http.Request)
		HandleSimilarityClearCache(http.ResponseWriter, *http.Request)
		HandleSimilarityLearn(http.ResponseWriter, *http.Request)
		HandleSimilarityOptimalThreshold(http.ResponseWriter, *http.Request)
		HandleSimilarityCrossValidate(http.ResponseWriter, *http.Request)
		HandleSimilarityPerformance(http.ResponseWriter, *http.Request)
		HandleSimilarityPerformanceReset(http.ResponseWriter, *http.Request)
		HandleSimilarityAnalyze(http.ResponseWriter, *http.Request)
		HandleSimilarityFindSimilar(http.ResponseWriter, *http.Request)
		HandleSimilarityCompareWeights(http.ResponseWriter, *http.Request)
		HandleSimilarityBreakdown(http.ResponseWriter, *http.Request)
		HandleSimilarityExport(http.ResponseWriter, *http.Request)
		HandleSimilarityImport(http.ResponseWriter, *http.Request)
	}
	// Legacy handlers для fallback
	HandleSimilarityCompare          http.HandlerFunc
	HandleSimilarityBatch            http.HandlerFunc
	HandleSimilarityWeights          http.HandlerFunc
	HandleSimilarityEvaluate         http.HandlerFunc
	HandleSimilarityStats            http.HandlerFunc
	HandleSimilarityClearCache       http.HandlerFunc
	HandleSimilarityLearn            http.HandlerFunc
	HandleSimilarityOptimalThreshold http.HandlerFunc
	HandleSimilarityCrossValidate    http.HandlerFunc
	HandleSimilarityPerformance      http.HandlerFunc
	HandleSimilarityPerformanceReset http.HandlerFunc
	HandleSimilarityAnalyze          http.HandlerFunc
	HandleSimilarityFindSimilar      http.HandlerFunc
	HandleSimilarityCompareWeights   http.HandlerFunc
	HandleSimilarityBreakdown         http.HandlerFunc
	HandleSimilarityExport           http.HandlerFunc
	HandleSimilarityImport           http.HandlerFunc
}

// RegisterSimilarityRoutes регистрирует маршруты для работы с алгоритмами схожести
func RegisterSimilarityRoutes(mux *http.ServeMux, h *SimilarityHandlers) {
	if h.SimilarityHandler != nil {
		mux.HandleFunc("/api/similarity/compare", h.SimilarityHandler.HandleSimilarityCompare)
		mux.HandleFunc("/api/similarity/batch", h.SimilarityHandler.HandleSimilarityBatch)
		mux.HandleFunc("/api/similarity/weights", h.SimilarityHandler.HandleSimilarityWeights)
		mux.HandleFunc("/api/similarity/evaluate", h.SimilarityHandler.HandleSimilarityEvaluate)
		mux.HandleFunc("/api/similarity/stats", h.SimilarityHandler.HandleSimilarityStats)
		mux.HandleFunc("/api/similarity/cache/clear", h.SimilarityHandler.HandleSimilarityClearCache)
		mux.HandleFunc("/api/similarity/learn", h.SimilarityHandler.HandleSimilarityLearn)
		mux.HandleFunc("/api/similarity/optimal-threshold", h.SimilarityHandler.HandleSimilarityOptimalThreshold)
		mux.HandleFunc("/api/similarity/cross-validate", h.SimilarityHandler.HandleSimilarityCrossValidate)
		mux.HandleFunc("/api/similarity/performance", h.SimilarityHandler.HandleSimilarityPerformance)
		mux.HandleFunc("/api/similarity/performance/reset", h.SimilarityHandler.HandleSimilarityPerformanceReset)
		mux.HandleFunc("/api/similarity/analyze", h.SimilarityHandler.HandleSimilarityAnalyze)
		mux.HandleFunc("/api/similarity/find-similar", h.SimilarityHandler.HandleSimilarityFindSimilar)
		mux.HandleFunc("/api/similarity/compare-weights", h.SimilarityHandler.HandleSimilarityCompareWeights)
		mux.HandleFunc("/api/similarity/breakdown", h.SimilarityHandler.HandleSimilarityBreakdown)
		mux.HandleFunc("/api/similarity/export", h.SimilarityHandler.HandleSimilarityExport)
		mux.HandleFunc("/api/similarity/import", h.SimilarityHandler.HandleSimilarityImport)
	} else {
		// Fallback к старым handlers
		if h.HandleSimilarityCompare != nil {
			mux.HandleFunc("/api/similarity/compare", h.HandleSimilarityCompare)
		}
		if h.HandleSimilarityBatch != nil {
			mux.HandleFunc("/api/similarity/batch", h.HandleSimilarityBatch)
		}
		if h.HandleSimilarityWeights != nil {
			mux.HandleFunc("/api/similarity/weights", h.HandleSimilarityWeights)
		}
		if h.HandleSimilarityEvaluate != nil {
			mux.HandleFunc("/api/similarity/evaluate", h.HandleSimilarityEvaluate)
		}
		if h.HandleSimilarityStats != nil {
			mux.HandleFunc("/api/similarity/stats", h.HandleSimilarityStats)
		}
		if h.HandleSimilarityClearCache != nil {
			mux.HandleFunc("/api/similarity/cache/clear", h.HandleSimilarityClearCache)
		}
		if h.HandleSimilarityLearn != nil {
			mux.HandleFunc("/api/similarity/learn", h.HandleSimilarityLearn)
		}
		if h.HandleSimilarityOptimalThreshold != nil {
			mux.HandleFunc("/api/similarity/optimal-threshold", h.HandleSimilarityOptimalThreshold)
		}
		if h.HandleSimilarityCrossValidate != nil {
			mux.HandleFunc("/api/similarity/cross-validate", h.HandleSimilarityCrossValidate)
		}
		if h.HandleSimilarityPerformance != nil {
			mux.HandleFunc("/api/similarity/performance", h.HandleSimilarityPerformance)
		}
		if h.HandleSimilarityPerformanceReset != nil {
			mux.HandleFunc("/api/similarity/performance/reset", h.HandleSimilarityPerformanceReset)
		}
		if h.HandleSimilarityAnalyze != nil {
			mux.HandleFunc("/api/similarity/analyze", h.HandleSimilarityAnalyze)
		}
		if h.HandleSimilarityFindSimilar != nil {
			mux.HandleFunc("/api/similarity/find-similar", h.HandleSimilarityFindSimilar)
		}
		if h.HandleSimilarityCompareWeights != nil {
			mux.HandleFunc("/api/similarity/compare-weights", h.HandleSimilarityCompareWeights)
		}
		if h.HandleSimilarityBreakdown != nil {
			mux.HandleFunc("/api/similarity/breakdown", h.HandleSimilarityBreakdown)
		}
		if h.HandleSimilarityExport != nil {
			mux.HandleFunc("/api/similarity/export", h.HandleSimilarityExport)
		}
		if h.HandleSimilarityImport != nil {
			mux.HandleFunc("/api/similarity/import", h.HandleSimilarityImport)
		}
	}
}
