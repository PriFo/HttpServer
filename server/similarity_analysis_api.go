package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/similarity/ для организации,
// но остается в пакете server для доступа к методам Server
// TODO:legacy-migration revisit dependencies after handler extraction

import (
	"encoding/json"
	"net/http"

	"httpserver/normalization/algorithms"
)

// handleSimilarityAnalyze анализирует множество пар строк с детальной разбивкой
// POST /api/similarity/analyze
func (s *Server) handleSimilarityAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Pairs     []algorithms.SimilarityPair `json:"pairs"`
		Threshold float64                     `json:"threshold"`
		Weights   *algorithms.SimilarityWeights `json:"weights,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Pairs) == 0 {
		s.writeJSONError(w, r, "pairs array is required", http.StatusBadRequest)
		return
	}

	if len(req.Pairs) > 500 {
		s.writeJSONError(w, r, "maximum 500 pairs allowed per request", http.StatusBadRequest)
		return
	}

	if req.Threshold <= 0 || req.Threshold > 1 {
		req.Threshold = 0.75 // По умолчанию
	}

	if req.Weights == nil {
		req.Weights = algorithms.DefaultSimilarityWeights()
	}

	// Анализируем пары
	analyzer := algorithms.NewSimilarityAnalyzer(req.Weights)
	result := analyzer.AnalyzePairs(req.Pairs, req.Threshold)

	s.writeJSONResponse(w, r, result, http.StatusOK)
}

// handleSimilarityFindSimilar находит похожие пары
// POST /api/similarity/find-similar
func (s *Server) handleSimilarityFindSimilar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Pairs     []algorithms.SimilarityPair `json:"pairs"`
		Threshold float64                     `json:"threshold"`
		Weights   *algorithms.SimilarityWeights `json:"weights,omitempty"`
		Limit     int                         `json:"limit,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Pairs) == 0 {
		s.writeJSONError(w, r, "pairs array is required", http.StatusBadRequest)
		return
	}

	if req.Threshold <= 0 || req.Threshold > 1 {
		req.Threshold = 0.75
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}

	if req.Weights == nil {
		req.Weights = algorithms.DefaultSimilarityWeights()
	}

	// Находим похожие пары
	analyzer := algorithms.NewSimilarityAnalyzer(req.Weights)
	similar := analyzer.FindSimilarPairs(req.Pairs, req.Threshold)

	// Ограничиваем количество результатов
	if len(similar) > req.Limit {
		similar = similar[:req.Limit]
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"similar_pairs": similar,
		"count":          len(similar),
		"threshold":     req.Threshold,
	}, http.StatusOK)
}

// handleSimilarityCompareWeights сравнивает эффективность разных наборов весов
// POST /api/similarity/compare-weights
func (s *Server) handleSimilarityCompareWeights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TestPairs  []algorithms.SimilarityTestPair `json:"test_pairs"`
		Weights    []*algorithms.SimilarityWeights `json:"weights"`
		Threshold  float64                         `json:"threshold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.TestPairs) == 0 {
		s.writeJSONError(w, r, "test_pairs array is required", http.StatusBadRequest)
		return
	}

	if len(req.Weights) == 0 {
		s.writeJSONError(w, r, "weights array is required", http.StatusBadRequest)
		return
	}

	if req.Threshold <= 0 || req.Threshold > 1 {
		req.Threshold = 0.75
	}

	// Сравниваем веса
	analyzer := algorithms.NewSimilarityAnalyzer(nil)
	results := analyzer.CompareWeights(req.TestPairs, req.Weights, req.Threshold)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"comparisons": results,
		"best_weights": results[0].Weights, // Первый - лучший по F1-score
		"best_f1_score": results[0].F1Score,
	}, http.StatusOK)
}

// handleSimilarityBreakdown получает разбивку схожести по алгоритмам
// POST /api/similarity/breakdown
func (s *Server) handleSimilarityBreakdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		String1 string                        `json:"string1"`
		String2 string                        `json:"string2"`
		Weights *algorithms.SimilarityWeights `json:"weights,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.String1 == "" || req.String2 == "" {
		s.writeJSONError(w, r, "string1 and string2 are required", http.StatusBadRequest)
		return
	}

	if req.Weights == nil {
		req.Weights = algorithms.DefaultSimilarityWeights()
	}

	// Вычисляем разбивку
	analyzer := algorithms.NewSimilarityAnalyzer(req.Weights)
	breakdown := analyzer.ComputeBreakdown(req.String1, req.String2)

	// Вычисляем общую схожесть
	hybrid := algorithms.HybridSimilarityAdvanced(req.String1, req.String2, req.Weights)

	// Вычисляем вклад каждого алгоритма
	contribution := map[string]float64{
		"jaro_winkler": breakdown.JaroWinkler * req.Weights.JaroWinkler,
		"lcs":          breakdown.LCS * req.Weights.LCS,
		"phonetic":     breakdown.Phonetic * req.Weights.Phonetic,
		"ngram":        breakdown.Ngram * req.Weights.Ngram,
		"jaccard":      breakdown.Jaccard * req.Weights.Jaccard,
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"string1":      req.String1,
		"string2":      req.String2,
		"hybrid":       hybrid,
		"breakdown":    breakdown,
		"contribution": contribution,
		"weights":      req.Weights,
	}, http.StatusOK)
}

