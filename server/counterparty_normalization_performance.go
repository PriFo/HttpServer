package server

import (
	"net/http"

	"httpserver/normalization"
)

// handleCounterpartyNormalizationStopCheckPerformance получает метрики производительности проверок остановки
// GET /api/counterparty/normalization/stop-check/performance
func (s *Server) handleCounterpartyNormalizationStopCheckPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := normalization.GetStopCheckStats()

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleCounterpartyNormalizationStopCheckPerformanceReset сбрасывает метрики производительности проверок остановки
// POST /api/counterparty/normalization/stop-check/performance/reset
func (s *Server) handleCounterpartyNormalizationStopCheckPerformanceReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	normalization.ResetStopCheckMetrics()

	s.writeJSONResponse(w, r, map[string]interface{}{
		"message": "Stop check performance metrics reset successfully",
	}, http.StatusOK)
}

