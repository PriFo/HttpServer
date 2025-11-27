package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл содержит вспомогательные методы Server, извлеченные из server.go
// для сокращения размера server.go

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"time"

	"httpserver/database"
	"httpserver/server/middleware"
)

func (s *Server) log(entry LogEntry) {
	select {
	case s.logChan <- entry:
	default:
		// Если канал полон, пропускаем запись
	}

	// Форматируем уровень логирования с эмодзи для лучшей читаемости
	levelIcon := ""
	switch entry.Level {
	case "ERROR":
		levelIcon = "✗"
	case "WARN":
		levelIcon = "⚠"
	case "INFO":
		levelIcon = "ℹ"
	case "DEBUG":
		levelIcon = "🔍"
	default:
		levelIcon = "•"
	}

	log.Printf("%s [%s] %s: %s", levelIcon, entry.Level, entry.Timestamp.Format("15:04:05"), entry.Message)
}

// logError логирует ошибку с уровнем ERROR
func (s *Server) logError(message string, endpoint string) {
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   message,
		Endpoint:  endpoint,
	})
}

// logErrorf логирует ошибку с форматированием
func (s *Server) logErrorf(format string, args ...interface{}) {
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   fmt.Sprintf(format, args...),
	})
}

// logWarn логирует предупреждение
func (s *Server) logWarn(message string, endpoint string) {
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "WARN",
		Message:   message,
		Endpoint:  endpoint,
	})
}

// logWarnf логирует предупреждение с форматированием
func (s *Server) logWarnf(format string, args ...interface{}) {
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "WARN",
		Message:   fmt.Sprintf(format, args...),
	})
}

// logInfo логирует информационное сообщение
func (s *Server) logInfo(message string, endpoint string) {
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   message,
		Endpoint:  endpoint,
	})
}

// logInfof логирует информационное сообщение с форматированием
func (s *Server) logInfof(format string, args ...interface{}) {
	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf(format, args...),
	})
}

// writeXMLResponse записывает XML ответ
func (s *Server) writeXMLResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	xmlData, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		s.writeErrorResponse(w, "Failed to marshal XML", err)
		return
	}

	w.Write([]byte(xml.Header))
	w.Write(xmlData)
}

// writeErrorResponse записывает ошибку в XML формате
func (s *Server) writeErrorResponse(w http.ResponseWriter, message string, err error) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	response := ErrorResponse{
		Success:   false,
		Error:     err.Error(),
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	xmlData, _ := xml.MarshalIndent(response, "", "  ")
	w.Write([]byte(xml.Header))
	w.Write(xmlData)
}

// handleStats обрабатывает запрос статистики

// System handlers перемещены в server/system_legacy_handlers.go

// TODO: Реализовать сравнение по ID (требует добавления метода GetScanByID)
// Функция была перемещена в system_legacy_handlers.go

// handleDatabaseV1Routes обрабатывает маршруты /api/v1/databases/{id}

// Database V1 routes handler перемещен в server/database_legacy_handlers.go
func (s *Server) handleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	middleware.HandleHTTPError(w, r, err)
}

// Upload и Normalized handlers перемещены в server/upload_normalized_handlers.go

// startNomenclatureProcessing запускает обработку номенклатуры
func (s *Server) getNomenclatureDBStats(db *database.DB) (DBStatsResponse, error) {
	var stats DBStatsResponse

	// Общее количество записей
	row := db.QueryRow("SELECT COUNT(*) FROM catalog_items")
	err := row.Scan(&stats.Total)
	if err != nil {
		return stats, fmt.Errorf("failed to get total count: %w", err)
	}

	// Количество обработанных
	row = db.QueryRow("SELECT COUNT(*) FROM catalog_items WHERE processing_status = 'completed'")
	err = row.Scan(&stats.Completed)
	if err != nil {
		return stats, fmt.Errorf("failed to get completed count: %w", err)
	}

	// Количество с ошибками
	row = db.QueryRow("SELECT COUNT(*) FROM catalog_items WHERE processing_status = 'error'")
	err = row.Scan(&stats.Errors)
	if err != nil {
		return stats, fmt.Errorf("failed to get error count: %w", err)
	}

	// Количество ожидающих обработки
	row = db.QueryRow("SELECT COUNT(*) FROM catalog_items WHERE processing_status IS NULL OR processing_status = 'pending'")
	err = row.Scan(&stats.Pending)
	if err != nil {
		return stats, fmt.Errorf("failed to get pending count: %w", err)
	}

	return stats, nil
}

// handleClients обрабатывает запросы к /api/clients

// Client handlers перемещены в server/client_legacy_handlers.go
// handleQualityUploadRoutes обрабатывает маршруты качества для выгрузок

// Quality handlers перемещены в server/quality_legacy_handlers.go
// Counterparties и прочие handlers перемещены в server/counterparties_handlers.go
