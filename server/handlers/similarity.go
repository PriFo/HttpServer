package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"httpserver/normalization/algorithms"
	"httpserver/server/services"
)

// SimilarityHandler обработчик для алгоритмов схожести
type SimilarityHandler struct {
	*BaseHandler
	similarityService *services.SimilarityService
	logFunc           func(entry interface{}) // server.LogEntry, но без прямого импорта
}

// NewSimilarityHandler создает новый обработчик схожести
func NewSimilarityHandler(baseHandler *BaseHandler, similarityService *services.SimilarityService, logFunc func(entry interface{})) *SimilarityHandler {
	return &SimilarityHandler{
		BaseHandler:       baseHandler,
		similarityService: similarityService,
		logFunc:           logFunc,
	}
}

// HandleSimilarityCompare обрабатывает запрос сравнения двух строк
func (h *SimilarityHandler) HandleSimilarityCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		String1 string                        `json:"string1"`
		String2 string                        `json:"string2"`
		Weights *algorithms.SimilarityWeights `json:"weights,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	result, err := h.similarityService.Compare(req.String1, req.String2, req.Weights)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error comparing strings: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleSimilarityBatch обрабатывает запрос пакетного сравнения
func (h *SimilarityHandler) HandleSimilarityBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Pairs   []algorithms.SimilarityPair   `json:"pairs"`
		Weights *algorithms.SimilarityWeights `json:"weights,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	results, cacheSize, err := h.similarityService.BatchCompare(req.Pairs, req.Weights)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error in batch comparison: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"results":    results,
		"count":      len(results),
		"cache_size": cacheSize,
	}, http.StatusOK)
}

// HandleSimilarityWeights обрабатывает запросы управления весами
func (h *SimilarityHandler) HandleSimilarityWeights(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Возвращаем веса по умолчанию
		weights := h.similarityService.GetDefaultWeights()
		h.WriteJSONResponse(w, r, map[string]interface{}{
			"weights": weights,
			"description": map[string]string{
				"jaro_winkler": "Для опечаток и перестановок (рекомендуется: 0.2-0.4)",
				"lcs":          "Для общих подпоследовательностей (рекомендуется: 0.1-0.3)",
				"phonetic":     "Для похожих по звучанию слов (рекомендуется: 0.1-0.3)",
				"ngram":        "Для частичных совпадений (рекомендуется: 0.1-0.3)",
				"jaccard":      "Для множеств токенов (рекомендуется: 0.1-0.2)",
			},
		}, http.StatusOK)

	case http.MethodPost:
		var req struct {
			Weights *algorithms.SimilarityWeights `json:"weights"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if err := h.similarityService.SetWeights(req.Weights); err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error setting weights: %v", err),
				Endpoint:  r.URL.Path,
			})
			h.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		h.WriteJSONResponse(w, r, map[string]interface{}{
			"success": true,
			"message": "Weights updated successfully",
			"weights": req.Weights,
		}, http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSimilarityStats обрабатывает запрос статистики кэша
func (h *SimilarityHandler) HandleSimilarityStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.similarityService.GetCacheStats()
	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleSimilarityClearCache обрабатывает запрос очистки кэша
func (h *SimilarityHandler) HandleSimilarityClearCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cacheSize := h.similarityService.ClearCache()
	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":    true,
		"message":    "Cache cleared successfully",
		"cache_size": cacheSize,
	}, http.StatusOK)
}

// HandleSimilarityEvaluate обрабатывает запросы к /api/similarity/evaluate
func (h *SimilarityHandler) HandleSimilarityEvaluate(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать оценку алгоритмов схожести
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityLearn обрабатывает запросы к /api/similarity/learn
func (h *SimilarityHandler) HandleSimilarityLearn(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать обучение алгоритмов схожести
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityOptimalThreshold обрабатывает запросы к /api/similarity/optimal-threshold
func (h *SimilarityHandler) HandleSimilarityOptimalThreshold(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать поиск оптимального порога
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityCrossValidate обрабатывает запросы к /api/similarity/cross-validate
func (h *SimilarityHandler) HandleSimilarityCrossValidate(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать кросс-валидацию
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityPerformance обрабатывает запросы к /api/similarity/performance
func (h *SimilarityHandler) HandleSimilarityPerformance(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать получение метрик производительности
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityPerformanceReset обрабатывает запросы к /api/similarity/performance/reset
func (h *SimilarityHandler) HandleSimilarityPerformanceReset(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать сброс метрик производительности
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityAnalyze обрабатывает запросы к /api/similarity/analyze
func (h *SimilarityHandler) HandleSimilarityAnalyze(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать анализ схожести
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityFindSimilar обрабатывает запросы к /api/similarity/find-similar
func (h *SimilarityHandler) HandleSimilarityFindSimilar(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать поиск похожих строк
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityCompareWeights обрабатывает запросы к /api/similarity/compare-weights
func (h *SimilarityHandler) HandleSimilarityCompareWeights(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать сравнение весов
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityBreakdown обрабатывает запросы к /api/similarity/breakdown
func (h *SimilarityHandler) HandleSimilarityBreakdown(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать разбивку схожести
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityExport обрабатывает запросы к /api/similarity/export
func (h *SimilarityHandler) HandleSimilarityExport(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать экспорт конфигурации
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}

// HandleSimilarityImport обрабатывает запросы к /api/similarity/import
func (h *SimilarityHandler) HandleSimilarityImport(w http.ResponseWriter, r *http.Request) {
	// TODO: Реализовать импорт конфигурации
	h.WriteJSONError(w, r, "Not implemented yet", http.StatusNotImplemented)
}