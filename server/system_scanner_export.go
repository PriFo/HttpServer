package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ExportSystemSummaryToCSV экспортирует SystemSummary в CSV формат
func ExportSystemSummaryToCSV(w http.ResponseWriter, summary *SystemSummary) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	timestamp := time.Now().Format("20060102_150405")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=system_summary_%s.csv", timestamp))

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Записываем заголовки
	headers := []string{
		"ID", "Upload UUID", "Name", "Status", "Created At", "Completed At",
		"Nomenclature Count", "Counterparty Count", "Database File",
		"Database ID", "Client ID", "Project ID",
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Записываем данные
	for _, upload := range summary.UploadDetails {
		completedAt := ""
		if upload.CompletedAt != nil {
			completedAt = upload.CompletedAt.Format(time.RFC3339)
		}

		databaseID := ""
		if upload.DatabaseID != nil {
			databaseID = strconv.Itoa(*upload.DatabaseID)
		}

		clientID := ""
		if upload.ClientID != nil {
			clientID = strconv.Itoa(*upload.ClientID)
		}

		projectID := ""
		if upload.ProjectID != nil {
			projectID = strconv.Itoa(*upload.ProjectID)
		}

		row := []string{
			upload.ID,
			upload.UploadUUID,
			upload.Name,
			upload.Status,
			upload.CreatedAt.Format(time.RFC3339),
			completedAt,
			strconv.FormatInt(upload.NomenclatureCount, 10),
			strconv.FormatInt(upload.CounterpartyCount, 10),
			upload.DatabaseFile,
			databaseID,
			clientID,
			projectID,
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	// Записываем сводную статистику
	csvWriter.Write([]string{}) // Пустая строка
	csvWriter.Write([]string{"Summary Statistics"})
	csvWriter.Write([]string{"Total Databases", strconv.Itoa(summary.TotalDatabases)})
	csvWriter.Write([]string{"Total Uploads", strconv.FormatInt(summary.TotalUploads, 10)})
	csvWriter.Write([]string{"Completed Uploads", strconv.FormatInt(summary.CompletedUploads, 10)})
	csvWriter.Write([]string{"Failed Uploads", strconv.FormatInt(summary.FailedUploads, 10)})
	csvWriter.Write([]string{"In Progress Uploads", strconv.FormatInt(summary.InProgressUploads, 10)})
	csvWriter.Write([]string{"Total Nomenclature", strconv.FormatInt(summary.TotalNomenclature, 10)})
	csvWriter.Write([]string{"Total Counterparties", strconv.FormatInt(summary.TotalCounterparties, 10)})
	csvWriter.Write([]string{"Last Activity", summary.LastActivity.Format(time.RFC3339)})
	if summary.ScanDuration != nil {
		csvWriter.Write([]string{"Scan Duration", *summary.ScanDuration})
	}
	csvWriter.Write([]string{"Databases Processed", strconv.Itoa(summary.DatabasesProcessed)})
	if summary.DatabasesSkipped > 0 {
		csvWriter.Write([]string{"Databases Skipped", strconv.Itoa(summary.DatabasesSkipped)})
	}

	return nil
}

// ExportSystemSummaryToJSON экспортирует SystemSummary в JSON формат (с форматированием)
func ExportSystemSummaryToJSON(w http.ResponseWriter, summary *SystemSummary) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	timestamp := time.Now().Format("20060102_150405")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=system_summary_%s.json", timestamp))

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

