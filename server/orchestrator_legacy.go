package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"httpserver/internal/infrastructure/ai"
	"httpserver/internal/infrastructure/monitoring"
)

// handleOrchestratorStrategy управляет стратегией агрегации оркестратора
func (s *Server) handleOrchestratorStrategy(w http.ResponseWriter, r *http.Request) {
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	log.Printf("[%s] %s /api/workers/orchestrator/strategy", traceID, r.Method)

	if s.providerOrchestrator == nil {
		s.writeJSONError(w, r, "Provider orchestrator not initialized", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Возвращаем текущую стратегию и доступные стратегии
		activeProviders := s.providerOrchestrator.GetActiveProviders()
		providersInfo := make([]map[string]interface{}, 0, len(activeProviders))
		for _, p := range activeProviders {
			providersInfo = append(providersInfo, map[string]interface{}{
				"id":       p.ID,
				"name":     p.Name,
				"priority": p.Priority,
				"enabled":  p.Enabled,
			})
		}

		response := map[string]interface{}{
			"current_strategy": s.providerOrchestrator.GetStrategy(),
			"available_strategies": []string{
				string(ai.FirstSuccess),
				string(ai.MajorityVote),
				string(ai.AllResults),
				string(ai.HighestConfidence),
			},
			"active_providers": providersInfo,
			"total_providers":  len(activeProviders),
		}

		s.writeJSONResponse(w, r, response, http.StatusOK)

	case http.MethodPost, http.MethodPut:
		// Устанавливаем новую стратегию
		var req struct {
			Strategy string `json:"strategy"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// Валидация стратегии
		validStrategies := map[string]bool{
			string(ai.FirstSuccess):      true,
			string(ai.MajorityVote):      true,
			string(ai.AllResults):        true,
			string(ai.HighestConfidence): true,
		}

		if !validStrategies[req.Strategy] {
			s.writeJSONError(w, r, fmt.Sprintf("Invalid strategy: %s. Available: first_success, majority_vote, all_results, highest_confidence", req.Strategy), http.StatusBadRequest)
			return
		}

		oldStrategy := s.providerOrchestrator.GetStrategy()
		s.providerOrchestrator.SetStrategy(ai.AggregationStrategy(req.Strategy))

		log.Printf("[%s] Aggregation strategy changed from %s to %s", traceID, oldStrategy, req.Strategy)

		response := map[string]interface{}{
			"success":           true,
			"previous_strategy": oldStrategy,
			"current_strategy":  req.Strategy,
			"message":           fmt.Sprintf("Strategy changed from %s to %s", oldStrategy, req.Strategy),
		}

		s.writeJSONResponse(w, r, response, http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOrchestratorStats возвращает статистику работы оркестратора
func (s *Server) handleOrchestratorStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.providerOrchestrator == nil {
		s.writeJSONError(w, r, "Provider orchestrator not initialized", http.StatusServiceUnavailable)
		return
	}

	// Получаем статистику от мониторинга
	stats := make(map[string]interface{})
	if s.monitoringManager != nil {
		activeProviders := s.providerOrchestrator.GetActiveProviders()
		providersStats := make([]map[string]interface{}, 0, len(activeProviders))

		// Получаем все метрики из мониторинга
		allMetrics := s.monitoringManager.GetAllMetrics()
		metricsMap := make(map[string]*monitoring.ProviderMetrics)
		for i := range allMetrics.Providers {
			metricsMap[allMetrics.Providers[i].ID] = &allMetrics.Providers[i]
		}

		for _, p := range activeProviders {
			providerStats := map[string]interface{}{
				"id":       p.ID,
				"name":     p.Name,
				"priority": p.Priority,
				"enabled":  p.Enabled,
			}

			// Получаем метрики провайдера, если они есть
			if metrics, exists := metricsMap[p.ID]; exists {
				providerStats["total_requests"] = metrics.TotalRequests
				providerStats["successful"] = metrics.SuccessfulRequests
				providerStats["failed"] = metrics.FailedRequests
				providerStats["current_requests"] = metrics.CurrentRequests
				providerStats["status"] = metrics.Status
				providerStats["rps"] = metrics.RequestsPerSecond

				// Рассчитываем success rate
				if metrics.TotalRequests > 0 {
					providerStats["success_rate"] = float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests)
				} else {
					providerStats["success_rate"] = 0.0
				}

				// Используем среднюю задержку из метрик
				providerStats["avg_latency_ms"] = metrics.AverageLatencyMs
			} else {
				// Если метрик нет, устанавливаем нулевые значения
				providerStats["total_requests"] = 0
				providerStats["successful"] = 0
				providerStats["failed"] = 0
				providerStats["current_requests"] = 0
				providerStats["status"] = "unknown"
				providerStats["rps"] = 0.0
				providerStats["success_rate"] = 0.0
				providerStats["avg_latency_ms"] = 0.0
			}

			providersStats = append(providersStats, providerStats)
		}

		stats["providers"] = providersStats
		stats["total_providers"] = len(activeProviders)
		stats["system"] = map[string]interface{}{
			"total_requests":   allMetrics.System.TotalRequests,
			"total_successful": allMetrics.System.TotalSuccessful,
			"total_failed":     allMetrics.System.TotalFailed,
			"system_rps":       allMetrics.System.SystemRequestsPerSecond,
		}
	}

	stats["strategy"] = s.providerOrchestrator.GetStrategy()
	stats["timeout"] = s.providerOrchestrator.GetTimeout().String()

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

