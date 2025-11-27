package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/ для организации,
// но остается в пакете server для доступа к методам Server

import (
	"net/http"
)

// handleImportManufacturers обрабатывает импорт производителей из файла
// Использует BenchmarkHandler
func (s *Server) handleImportManufacturers(w http.ResponseWriter, r *http.Request) {
	if s.benchmarkHandler != nil {
		s.benchmarkHandler.HandleImportManufacturers(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Benchmark handler not initialized", http.StatusServiceUnavailable)
}

