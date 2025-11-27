package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл физически перемещен из server/server_pipeline.go для организации,
// но остается в пакете server для доступа к методам Server
// TODO:legacy-migration revisit dependencies after handler extraction

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// handlePipelineStats returns pipeline stage progress statistics
// Использует NormalizationHandler
func (s *Server) handlePipelineStats(w http.ResponseWriter, r *http.Request) {
	if s.normalizationHandler != nil {
		s.normalizationHandler.HandlePipelineStats(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Normalization handler not initialized", http.StatusServiceUnavailable)
}

// handleStageDetails returns detailed information about a specific pipeline stage
// Использует NormalizationHandler
func (s *Server) handleStageDetails(w http.ResponseWriter, r *http.Request) {
	if s.normalizationHandler != nil {
		s.normalizationHandler.HandleStageDetails(w, r)
		return
	}
	// Fallback если handler не инициализирован
	s.writeJSONError(w, r, "Normalization handler not initialized", http.StatusServiceUnavailable)
}

// handleExport exports pipeline data in requested format
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	columnsParam := r.URL.Query().Get("columns")
	var columns []string
	if columnsParam != "" {
		columns = strings.Split(columnsParam, ",")
		// Очищаем пробелы
		for i := range columns {
			columns[i] = strings.TrimSpace(columns[i])
		}
	} else {
		// Дефолтные колонки
		columns = []string{
			"id", "source_reference", "source_name", "code", "normalized_name",
			"normalized_reference", "category", "merged_count", "ai_confidence",
			"processing_level", "kpved_code", "kpved_name", "kpved_confidence", "created_at",
		}
	}

	// Валидация колонок
	allowedColumns := map[string]bool{
		"id": true, "source_reference": true, "source_name": true, "code": true,
		"normalized_name": true, "normalized_reference": true, "category": true,
		"merged_count": true, "ai_confidence": true, "processing_level": true,
		"kpved_code": true, "kpved_name": true, "kpved_confidence": true,
		"created_at": true, "stage05_completed": true, "stage1_completed": true,
		"stage2_completed": true, "stage25_completed": true, "stage3_completed": true,
		"stage35_completed": true, "stage4_completed": true, "stage5_completed": true,
		"stage6_completed": true, "stage65_completed": true, "stage7_ai_processed": true,
		"stage8_completed": true, "stage9_completed": true, "stage10_exported": true,
		"stage11_kpved_completed": true, "stage12_okpd2_completed": true,
	}

	validColumns := []string{}
	for _, col := range columns {
		if allowedColumns[col] {
			validColumns = append(validColumns, col)
		}
	}

	if len(validColumns) == 0 {
		validColumns = []string{"id", "normalized_name", "category", "merged_count"}
	}

	// Формируем WHERE условие из фильтров
	whereConditions := []string{}
	args := []interface{}{}

	// Фильтры из query параметров
	if category := r.URL.Query().Get("category"); category != "" {
		whereConditions = append(whereConditions, "category = ?")
		args = append(args, category)
	}

	if kpvedCode := r.URL.Query().Get("kpved_code"); kpvedCode != "" {
		whereConditions = append(whereConditions, "kpved_code = ?")
		args = append(args, kpvedCode)
	}

	if processingLevel := r.URL.Query().Get("processing_level"); processingLevel != "" {
		whereConditions = append(whereConditions, "processing_level = ?")
		args = append(args, processingLevel)
	}

	whereClause := "1=1"
	if len(whereConditions) > 0 {
		whereClause = strings.Join(whereConditions, " AND ")
	}

	// Формируем SELECT запрос
	selectColumns := strings.Join(validColumns, ", ")
	query := "SELECT " + selectColumns + " FROM normalized_data WHERE " + whereClause + " ORDER BY created_at DESC"

	rows, err := s.normalizedDB.Query(query, args...)
	if err != nil {
		log.Printf("Export query error: %v", err)
		s.writeJSONError(w, r, "Failed to export data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Получаем имена колонок из запроса
	columnNames := validColumns

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=normalized_data_export.csv")
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		// Записываем заголовки
		if err := csvWriter.Write(columnNames); err != nil {
			log.Printf("Export CSV header error: %v", err)
			return
		}

		// Записываем данные
		values := make([]interface{}, len(columnNames))
		valuePtrs := make([]interface{}, len(columnNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		for rows.Next() {
			if err := rows.Scan(valuePtrs...); err != nil {
				log.Printf("Export CSV scan error: %v", err)
				continue
			}

			row := make([]string, len(columnNames))
			for i, val := range values {
				if val == nil {
					row[i] = ""
				} else {
					row[i] = strings.TrimSpace(fmt.Sprintf("%v", val))
				}
			}

			if err := csvWriter.Write(row); err != nil {
				log.Printf("Export CSV write error: %v", err)
				continue
			}
		}

	case "json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=normalized_data_export.json")

		values := make([]interface{}, len(columnNames))
		valuePtrs := make([]interface{}, len(columnNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		data := []map[string]interface{}{}
		for rows.Next() {
			if err := rows.Scan(valuePtrs...); err != nil {
				log.Printf("Export JSON scan error: %v", err)
				continue
			}

			item := make(map[string]interface{})
			for i, val := range values {
				if val != nil {
					item[columnNames[i]] = val
				} else {
					item[columnNames[i]] = nil
				}
			}
			data = append(data, item)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"columns": columnNames,
			"count":   len(data),
			"data":    data,
		})

	default:
		s.writeJSONError(w, r, "Unsupported export format. Use 'csv' or 'json'", http.StatusBadRequest)
	}
}
