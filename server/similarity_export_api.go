package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен в server/handlers/legacy/similarity/ для организации,
// но остается в пакете server для доступа к методам Server
// TODO:legacy-migration revisit dependencies after handler extraction

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"httpserver/normalization/algorithms"
)

// handleSimilarityExport экспортирует результаты анализа
// POST /api/similarity/export
func (s *Server) handleSimilarityExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Pairs   []algorithms.SimilarityPair `json:"pairs"`
		Format  string                      `json:"format"` // "json", "csv", "tsv", "report"
		Threshold float64                   `json:"threshold"`
		Weights *algorithms.SimilarityWeights `json:"weights,omitempty"`
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

	if req.Weights == nil {
		req.Weights = algorithms.DefaultSimilarityWeights()
	}

	// Анализируем пары
	analyzer := algorithms.NewSimilarityAnalyzer(req.Weights)
	result := analyzer.AnalyzePairs(req.Pairs, req.Threshold)

	// Определяем формат
	format := algorithms.ExportFormat(req.Format)
	if format == "" {
		format = algorithms.ExportFormatJSON
	}

	// Создаем временный файл
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("similarity_export_%s", timestamp)
	
	var ext string
	switch format {
	case algorithms.ExportFormatCSV:
		ext = ".csv"
	case algorithms.ExportFormatTSV:
		ext = ".tsv"
	case "report":
		ext = ".md"
	default:
		ext = ".json"
		format = algorithms.ExportFormatJSON
	}

	exportPath := filepath.Join("exports", filename+ext)

	// Экспортируем
	exporter := algorithms.NewSimilarityExporter(result)
	var err error
	if req.Format == "report" {
		err = exporter.ExportReport(exportPath)
	} else {
		err = exporter.Export(exportPath, format)
	}

	if err != nil {
		s.writeJSONError(w, r, "Failed to export: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"filepath": exportPath,
		"format":   req.Format,
		"count":    len(req.Pairs),
		"message":  "Export completed successfully",
	}, http.StatusOK)
}

// handleSimilarityImport импортирует обучающие пары
// POST /api/similarity/import
func (s *Server) handleSimilarityImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		s.writeJSONError(w, r, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.writeJSONError(w, r, "File is required: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Определяем формат по расширению
	ext := filepath.Ext(header.Filename)
	var format algorithms.ExportFormat
	switch ext {
	case ".json":
		format = algorithms.ExportFormatJSON
	case ".csv":
		format = algorithms.ExportFormatCSV
	default:
		s.writeJSONError(w, r, "Unsupported file format. Use .json or .csv", http.StatusBadRequest)
		return
	}

	// Сохраняем временный файл
	tempPath := filepath.Join("tmp", fmt.Sprintf("import_%d%s", time.Now().Unix(), ext))
	tempFile, err := os.Create(tempPath)
	if err != nil {
		s.writeJSONError(w, r, "Failed to create temp file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempPath)

	if _, err := io.Copy(tempFile, file); err != nil {
		s.writeJSONError(w, r, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tempFile.Close()

	// Импортируем
	pairs, err := algorithms.ImportTrainingPairs(tempPath, format)
	if err != nil {
		s.writeJSONError(w, r, "Failed to import: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"pairs_imported": len(pairs),
		"pairs":         pairs,
		"message":       "Import completed successfully",
	}, http.StatusOK)
}

