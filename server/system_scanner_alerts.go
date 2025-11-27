package server

import (
	"fmt"
	"time"

	"httpserver/internal/domain/models"
)

// CheckScanAlerts проверяет результаты сканирования и генерирует алерты
func CheckScanAlerts(summary *models.SystemSummary) []models.ScanAlert {
	var alerts []models.ScanAlert

	// Проверка на большое количество пропущенных БД
	if summary.TotalDatabases > 0 {
		skipRate := float64(summary.DatabasesSkipped) / float64(summary.TotalDatabases)
		if skipRate > 0.3 { // Более 30% пропущено
			severity := "warning"
			if skipRate > 0.5 {
				severity = "error"
			}
			if skipRate > 0.7 {
				severity = "critical"
			}
			alerts = append(alerts, models.ScanAlert{
				Type:     models.AlertTypeManySkippedDBs,
				Severity: severity,
				Message:  fmt.Sprintf("Высокий процент пропущенных БД: %.1f%% (%d из %d)", skipRate*100, summary.DatabasesSkipped, summary.TotalDatabases),
				Details: map[string]interface{}{
					"skipped": summary.DatabasesSkipped,
					"total":   summary.TotalDatabases,
					"rate":    skipRate,
				},
				Timestamp: time.Now(),
			})
		}
	}

	// Проверка на медленное сканирование
	if summary.ScanDuration != nil {
		duration, err := time.ParseDuration(*summary.ScanDuration)
		if err == nil {
			if duration > 2*time.Minute {
				alerts = append(alerts, models.ScanAlert{
					Type:     models.AlertTypeSlowScan,
					Severity: "warning",
					Message:  fmt.Sprintf("Сканирование заняло много времени: %s", duration.Round(time.Second)),
					Details: map[string]interface{}{
						"duration": duration.String(),
					},
					Timestamp: time.Now(),
				})
			}
			if duration > 5*time.Minute {
				alerts = append(alerts, models.ScanAlert{
					Type:     models.AlertTypeSlowScan,
					Severity: "error",
					Message:  fmt.Sprintf("Очень медленное сканирование: %s", duration.Round(time.Second)),
					Details: map[string]interface{}{
						"duration": duration.String(),
					},
					Timestamp: time.Now(),
				})
			}
		}
	}

	// Проверка на высокий процент ошибок
	if summary.TotalUploads > 0 {
		errorRate := float64(summary.FailedUploads) / float64(summary.TotalUploads)
		if errorRate > 0.2 { // Более 20% ошибок
			severity := "warning"
			if errorRate > 0.4 {
				severity = "error"
			}
			alerts = append(alerts, models.ScanAlert{
				Type:     models.AlertTypeHighErrorRate,
				Severity: severity,
				Message:  fmt.Sprintf("Высокий процент проваленных загрузок: %.1f%% (%d из %d)", errorRate*100, summary.FailedUploads, summary.TotalUploads),
				Details: map[string]interface{}{
					"failed": summary.FailedUploads,
					"total":  summary.TotalUploads,
					"rate":   errorRate,
				},
				Timestamp: time.Now(),
			})
		}
	}

	// Проверка на очень большие БД
	for _, upload := range summary.UploadDetails {
		if upload.DatabaseSize != nil {
			sizeMB := float64(*upload.DatabaseSize) / (1024 * 1024)
			if sizeMB > 1000 { // Более 1GB
				alerts = append(alerts, models.ScanAlert{
					Type:     models.AlertTypeLargeDatabaseSize,
					Severity: "warning",
					Message:  fmt.Sprintf("Большая БД для загрузки %s: %.2f MB", upload.Name, sizeMB),
					Details: map[string]interface{}{
						"upload_name": upload.Name,
						"size_bytes":  *upload.DatabaseSize,
						"size_mb":     sizeMB,
					},
					Timestamp: time.Now(),
				})
			}
		}
	}

	return alerts
}
