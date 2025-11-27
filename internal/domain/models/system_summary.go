package models

import (
	"time"
)

// SystemSummary содержит сводную информацию по всей системе
type SystemSummary struct {
	TotalDatabases      int             `json:"total_databases"`
	TotalUploads        int64           `json:"total_uploads"`
	CompletedUploads    int64           `json:"completed_uploads"`
	FailedUploads       int64           `json:"failed_uploads"`
	InProgressUploads   int64           `json:"in_progress_uploads"`
	LastActivity        time.Time       `json:"last_activity"`
	TotalNomenclature   int64           `json:"total_nomenclature"`
	TotalCounterparties int64           `json:"total_counterparties"`
	UploadDetails       []UploadSummary `json:"upload_details"`
	// Метрики производительности (опционально)
	ScanDuration       *string     `json:"scan_duration,omitempty"`     // Длительность сканирования в формате "1.23s"
	DatabasesProcessed int         `json:"databases_processed"`         // Количество обработанных БД
	DatabasesSkipped   int         `json:"databases_skipped,omitempty"` // Количество пропущенных БД (не найдены или ошибки)
	Alerts             []ScanAlert `json:"alerts,omitempty"`            // Алерты о проблемах при сканировании
}

// UploadSummary содержит детальную информацию по каждой загрузке
type UploadSummary struct {
	ID                string     `json:"id"`
	UploadUUID        string     `json:"upload_uuid"`
	Name              string     `json:"name"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	NomenclatureCount int64      `json:"nomenclature_count"`
	CounterpartyCount int64      `json:"counterparty_count"`
	DatabaseFile      string     `json:"database_file"`
	DatabaseID        *int       `json:"database_id,omitempty"`
	ClientID          *int       `json:"client_id,omitempty"`
	ProjectID         *int       `json:"project_id,omitempty"`
	DatabaseSize      *int64     `json:"database_size,omitempty"` // Размер файла БД в байтах
}

// ScanAlertType тип алерта
type ScanAlertType string

const (
	AlertTypeManySkippedDBs    ScanAlertType = "many_skipped_dbs"
	AlertTypeSlowScan          ScanAlertType = "slow_scan"
	AlertTypeHighErrorRate     ScanAlertType = "high_error_rate"
	AlertTypeDatabaseNotFound  ScanAlertType = "database_not_found"
	AlertTypeLargeDatabaseSize ScanAlertType = "large_database_size"
)

// ScanAlert алерт о проблеме при сканировании
type ScanAlert struct {
	Type      ScanAlertType          `json:"type"`
	Severity  string                 `json:"severity"` // "warning", "error", "critical"
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}
