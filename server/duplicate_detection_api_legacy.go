package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен из server/server_duplicate_detection_api.go для организации,
// но остается в пакете server для доступа к методам Server
// TODO:legacy-migration revisit dependencies after handler extraction

import (
	"net/http"
	"strings"
)

// handleDuplicateDetection запускает обнаружение дублей в базе данных
// POST /api/duplicates/detect
// Использует DuplicateDetectionHandler
func (s *Server) handleDuplicateDetection(w http.ResponseWriter, r *http.Request) {
	if s.duplicateDetectionHandler != nil {
		s.duplicateDetectionHandler.HandleStartDetection(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Duplicate detection handler not initialized", http.StatusServiceUnavailable)
}

// runDuplicateDetection удален - логика перенесена в DuplicateDetectionService

// handleDuplicateDetectionStatus получает статус задачи или список задач
// GET /api/duplicates/detect - список всех задач
// GET /api/duplicates/detect/{task_id} - статус конкретной задачи
// Использует DuplicateDetectionHandler
func (s *Server) handleDuplicateDetectionStatus(w http.ResponseWriter, r *http.Request) {
	if s.duplicateDetectionHandler != nil {
		// Проверяем, есть ли task_id в пути
		path := r.URL.Path
		if strings.Contains(path, "/api/duplicates/detect/") && path != "/api/duplicates/detect" {
			// Есть task_id - используем HandleGetStatus
			s.duplicateDetectionHandler.HandleGetStatus(w, r)
		} else {
			// Нет task_id - возвращаем список задач (пока не реализовано в handler)
			s.writeJSONError(w, r, "List of tasks not yet implemented in handler", http.StatusNotImplemented)
		}
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Duplicate detection handler not initialized", http.StatusServiceUnavailable)
}


