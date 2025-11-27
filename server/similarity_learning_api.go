package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/similarity/ для организации,
// но остается в пакете server для доступа к методам Server

import (
	"encoding/json"
	"net/http"

	"httpserver/normalization/algorithms"
)

// handleSimilarityLearn обучает алгоритм на размеченных данных
// POST /api/similarity/learn
func (s *Server) handleSimilarityLearn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TrainingPairs []algorithms.SimilarityTestPair `json:"training_pairs"`
		Iterations    int                             `json:"iterations,omitempty"`
		LearningRate  float64                         `json:"learning_rate,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.TrainingPairs) == 0 {
		s.writeJSONError(w, r, "training_pairs array is required", http.StatusBadRequest)
		return
	}

	if req.Iterations <= 0 {
		req.Iterations = 100 // По умолчанию
	}

	if req.LearningRate <= 0 {
		req.LearningRate = 0.01 // По умолчанию
	}

	// Создаем обучатель
	learner := algorithms.NewSimilarityLearner()
	learner.AddTrainingPairs(req.TrainingPairs)

	// Обучаем
	weights, err := learner.OptimizeWeights(req.Iterations, req.LearningRate)
	if err != nil {
		s.writeJSONError(w, r, "Failed to optimize weights: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Оцениваем на обучающих данных
	threshold := 0.75
	metrics := learner.EvaluateCurrentWeights(req.TrainingPairs, threshold)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"weights": weights,
		"metrics": map[string]interface{}{
			"precision":            metrics.Precision(),
			"recall":               metrics.Recall(),
			"f1_score":             metrics.F1Score(),
			"accuracy":             metrics.Accuracy(),
			"false_positive_rate":  metrics.FalsePositiveRate(),
			"false_negative_rate":  metrics.FalseNegativeRate(),
		},
		"training_pairs_count": len(req.TrainingPairs),
		"iterations":            req.Iterations,
		"learning_rate":         req.LearningRate,
	}, http.StatusOK)
}

// handleSimilarityOptimalThreshold находит оптимальный порог
// POST /api/similarity/optimal-threshold
func (s *Server) handleSimilarityOptimalThreshold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TestPairs []algorithms.SimilarityTestPair `json:"test_pairs"`
		Weights   *algorithms.SimilarityWeights  `json:"weights,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.TestPairs) == 0 {
		s.writeJSONError(w, r, "test_pairs array is required", http.StatusBadRequest)
		return
	}

	if req.Weights == nil {
		req.Weights = algorithms.DefaultSimilarityWeights()
	}

	learner := algorithms.NewSimilarityLearner()
	threshold, metrics := learner.GetOptimalThreshold(req.TestPairs, req.Weights)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"optimal_threshold": threshold,
		"metrics": map[string]interface{}{
			"precision":            metrics.Precision(),
			"recall":               metrics.Recall(),
			"f1_score":             metrics.F1Score(),
			"accuracy":             metrics.Accuracy(),
			"false_positive_rate":  metrics.FalsePositiveRate(),
			"false_negative_rate":  metrics.FalseNegativeRate(),
		},
		"weights": req.Weights,
	}, http.StatusOK)
}

// handleSimilarityCrossValidate выполняет кросс-валидацию
// POST /api/similarity/cross-validate
func (s *Server) handleSimilarityCrossValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TrainingPairs []algorithms.SimilarityTestPair `json:"training_pairs"`
		Folds         int                             `json:"folds,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.TrainingPairs) == 0 {
		s.writeJSONError(w, r, "training_pairs array is required", http.StatusBadRequest)
		return
	}

	if req.Folds <= 0 {
		req.Folds = 5 // По умолчанию
	}

	if len(req.TrainingPairs) < req.Folds {
		s.writeJSONError(w, r, "not enough training pairs for specified number of folds", http.StatusBadRequest)
		return
	}

	learner := algorithms.NewSimilarityLearner()
	learner.AddTrainingPairs(req.TrainingPairs)

	results, err := learner.CrossValidate(req.Folds)
	if err != nil {
		s.writeJSONError(w, r, "Cross-validation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Вычисляем средние метрики
	avgMetrics := algorithms.GetAverageMetrics(results)

	// Формируем детальные результаты для каждого фолда
	foldResults := make([]map[string]interface{}, len(results))
	for i, m := range results {
		foldResults[i] = map[string]interface{}{
			"fold": i + 1,
			"metrics": map[string]interface{}{
				"precision":            m.Precision(),
				"recall":               m.Recall(),
				"f1_score":             m.F1Score(),
				"accuracy":             m.Accuracy(),
				"false_positive_rate":  m.FalsePositiveRate(),
				"false_negative_rate":  m.FalseNegativeRate(),
			},
		}
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"folds": req.Folds,
		"average_metrics": map[string]interface{}{
			"precision":            avgMetrics.Precision(),
			"recall":               avgMetrics.Recall(),
			"f1_score":             avgMetrics.F1Score(),
			"accuracy":             avgMetrics.Accuracy(),
			"false_positive_rate":  avgMetrics.FalsePositiveRate(),
			"false_negative_rate":  avgMetrics.FalseNegativeRate(),
		},
		"fold_results": foldResults,
		"training_pairs_count": len(req.TrainingPairs),
	}, http.StatusOK)
}

