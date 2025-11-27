package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"httpserver/server/services"
)

// ReportHandler обработчик для генерации отчетов
type ReportHandler struct {
	*BaseHandler
	reportService *services.ReportService
	logFunc       func(entry interface{}) // server.LogEntry, но без прямого импорта
	// Функции генерации отчетов от Server
	generateNormalizationReport func() (interface{}, error)
	generateDataQualityReport    func(*int) (interface{}, error)
	generateQualityReport        func(string) (interface{}, error)
}

// NewReportHandler создает новый обработчик отчетов
func NewReportHandler(
	baseHandler *BaseHandler,
	reportService *services.ReportService,
	logFunc func(entry interface{}), // server.LogEntry, но без прямого импорта
	generateNormalizationReport func() (interface{}, error),
	generateDataQualityReport func(*int) (interface{}, error),
	generateQualityReport func(string) (interface{}, error),
) *ReportHandler {
	return &ReportHandler{
		BaseHandler:                baseHandler,
		reportService:              reportService,
		logFunc:                    logFunc,
		generateNormalizationReport: generateNormalizationReport,
		generateDataQualityReport:  generateDataQualityReport,
		generateQualityReport:      generateQualityReport,
	}
}

// HandleGenerateNormalizationReport обрабатывает запрос генерации отчета о нормализации
func (h *ReportHandler) HandleGenerateNormalizationReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	report, err := h.reportService.GenerateNormalizationReport(h.generateNormalizationReport)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error generating normalization report: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, report, http.StatusOK)
}

// HandleGenerateDataQualityReport обрабатывает запрос генерации отчета о качестве данных
func (h *ReportHandler) HandleGenerateDataQualityReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим тело запроса (опционально - project_id)
	type Request struct {
		ProjectID *int `json:"project_id,omitempty"`
	}

	var req Request
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "WARN",
				Message:   fmt.Sprintf("Error decoding request body: %v, using defaults", err),
				Endpoint:  r.URL.Path,
			})
			// Продолжаем с пустым запросом (все проекты)
		}
	}

	report, err := h.reportService.GenerateDataQualityReport(req.ProjectID, h.generateDataQualityReport)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error generating data quality report: %v", err),
			Endpoint:  r.URL.Path,
		})
		if err.Error() == "invalid project_id: must be positive integer" {
			h.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to generate data quality report: %v", err), http.StatusInternalServerError)
		}
		return
	}

	h.WriteJSONResponse(w, r, report, http.StatusOK)
}

// HandleGetQualityReport обрабатывает запрос получения отчета о качестве нормализации
func (h *ReportHandler) HandleGetQualityReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметр database из query
	databasePath := r.URL.Query().Get("database")
	if databasePath == "" {
		h.WriteJSONError(w, r, "database parameter is required", http.StatusBadRequest)
		return
	}

	report, err := h.reportService.GenerateQualityReport(databasePath, h.generateQualityReport)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error generating quality report: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to generate quality report: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, report, http.StatusOK)
}

