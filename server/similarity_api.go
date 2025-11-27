package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/similarity/ для организации,
// но остается в пакете server для доступа к методам Server

import (
	"net/http"
)

// handleSimilarityCompare сравнивает две строки используя различные алгоритмы
// POST /api/similarity/compare
// Использует SimilarityHandler
func (s *Server) handleSimilarityCompare(w http.ResponseWriter, r *http.Request) {
	if s.similarityHandler != nil {
		s.similarityHandler.HandleSimilarityCompare(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Similarity handler not initialized", http.StatusServiceUnavailable)
}

// handleSimilarityBatch сравнивает множество пар строк
// POST /api/similarity/batch
// Использует SimilarityHandler
func (s *Server) handleSimilarityBatch(w http.ResponseWriter, r *http.Request) {
	if s.similarityHandler != nil {
		s.similarityHandler.HandleSimilarityBatch(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Similarity handler not initialized", http.StatusServiceUnavailable)
}

// handleSimilarityWeights управление весами алгоритмов
// GET /api/similarity/weights - получить веса по умолчанию
// POST /api/similarity/weights - установить пользовательские веса
// Использует SimilarityHandler
func (s *Server) handleSimilarityWeights(w http.ResponseWriter, r *http.Request) {
	if s.similarityHandler != nil {
		s.similarityHandler.HandleSimilarityWeights(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Similarity handler not initialized", http.StatusServiceUnavailable)
}

// handleSimilarityEvaluate оценивает эффективность алгоритма на тестовых данных
// POST /api/similarity/evaluate
// Использует SimilarityHandler
func (s *Server) handleSimilarityEvaluate(w http.ResponseWriter, r *http.Request) {
	if s.similarityHandler != nil {
		s.similarityHandler.HandleSimilarityEvaluate(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Similarity handler not initialized", http.StatusServiceUnavailable)
}

// handleSimilarityStats получает статистику производительности
// GET /api/similarity/stats
// Использует SimilarityHandler
func (s *Server) handleSimilarityStats(w http.ResponseWriter, r *http.Request) {
	if s.similarityHandler != nil {
		s.similarityHandler.HandleSimilarityStats(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Similarity handler not initialized", http.StatusServiceUnavailable)
}

// handleSimilarityClearCache очищает кэш
// POST /api/similarity/cache/clear
// Использует SimilarityHandler
func (s *Server) handleSimilarityClearCache(w http.ResponseWriter, r *http.Request) {
	if s.similarityHandler != nil {
		s.similarityHandler.HandleSimilarityClearCache(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Similarity handler not initialized", http.StatusServiceUnavailable)
}

