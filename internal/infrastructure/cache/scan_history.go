package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"httpserver/internal/domain/models"

	_ "github.com/mattn/go-sqlite3"
)

// ScanHistoryEntry запись в истории сканирований
type ScanHistoryEntry struct {
	ID           int64                 `json:"id"`
	ScanTime     time.Time             `json:"scan_time"`
	Summary      *models.SystemSummary `json:"summary"`
	ScanDuration string                `json:"scan_duration"`
	Success      bool                  `json:"success"`
	Error        string                `json:"error,omitempty"`
}

// ScanHistoryManager управляет историей сканирований
type ScanHistoryManager struct {
	dbPath string
	db     *sql.DB
	mu     sync.RWMutex
}

// NewScanHistoryManager создает новый менеджер истории сканирований
func NewScanHistoryManager(dbPath string) (*ScanHistoryManager, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть БД истории: %w", err)
	}

	manager := &ScanHistoryManager{
		dbPath: dbPath,
		db:     db,
	}

	// Создаем таблицу для истории
	if err := manager.initTable(); err != nil {
		db.Close()
		return nil, fmt.Errorf("не удалось инициализировать таблицу истории: %w", err)
	}

	return manager, nil
}

// initTable создает таблицу для хранения истории сканирований
func (m *ScanHistoryManager) initTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS scan_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scan_time TEXT NOT NULL,
		summary_json TEXT NOT NULL,
		scan_duration TEXT,
		success INTEGER NOT NULL DEFAULT 1,
		error TEXT,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_scan_history_scan_time ON scan_history(scan_time);
	`
	_, err := m.db.Exec(query)
	return err
}

// SaveScan сохраняет результат сканирования в историю
func (m *ScanHistoryManager) SaveScan(ctx context.Context, summary *models.SystemSummary, scanDuration string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	summaryJSON, jsonErr := json.Marshal(summary)
	if jsonErr != nil {
		return fmt.Errorf("не удалось сериализовать сводку: %w", jsonErr)
	}

	success := 1
	errorMsg := ""
	if err != nil {
		success = 0
		errorMsg = err.Error()
	}

	query := `
		INSERT INTO scan_history (scan_time, summary_json, scan_duration, success, error)
		VALUES (?, ?, ?, ?, ?)
	`
	_, dbErr := m.db.ExecContext(ctx, query,
		time.Now().Format(time.RFC3339),
		string(summaryJSON),
		scanDuration,
		success,
		errorMsg,
	)

	return dbErr
}

// GetHistory получает историю сканирований с лимитом
func (m *ScanHistoryManager) GetHistory(ctx context.Context, limit int) ([]ScanHistoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 {
		limit = 50 // По умолчанию
	}
	if limit > 1000 {
		limit = 1000 // Максимум
	}

	query := `
		SELECT id, scan_time, summary_json, scan_duration, success, error
		FROM scan_history
		ORDER BY scan_time DESC
		LIMIT ?
	`

	rows, err := m.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить историю: %w", err)
	}
	defer rows.Close()

	var entries []ScanHistoryEntry
	for rows.Next() {
		var entry ScanHistoryEntry
		var scanTimeStr string
		var summaryJSON string
		var scanDuration sql.NullString
		var success int
		var errorMsg sql.NullString

		if err := rows.Scan(&entry.ID, &scanTimeStr, &summaryJSON, &scanDuration, &success, &errorMsg); err != nil {
			continue
		}

		entry.ScanTime, _ = time.Parse(time.RFC3339, scanTimeStr)
		entry.Success = success == 1
		if scanDuration.Valid {
			entry.ScanDuration = scanDuration.String
		}
		if errorMsg.Valid {
			entry.Error = errorMsg.String
		}

		// Десериализуем сводку
		if err := json.Unmarshal([]byte(summaryJSON), &entry.Summary); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetLastScan получает последнее сканирование
func (m *ScanHistoryManager) GetLastScan(ctx context.Context) (*ScanHistoryEntry, error) {
	history, err := m.GetHistory(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, fmt.Errorf("история сканирований пуста")
	}
	return &history[0], nil
}

// CompareScans сравнивает два сканирования
func CompareScans(old, new *models.SystemSummary) map[string]interface{} {
	diff := make(map[string]interface{})

	diff["total_databases_change"] = new.TotalDatabases - old.TotalDatabases
	diff["total_uploads_change"] = new.TotalUploads - old.TotalUploads
	diff["completed_uploads_change"] = new.CompletedUploads - old.CompletedUploads
	diff["failed_uploads_change"] = new.FailedUploads - old.FailedUploads
	diff["total_nomenclature_change"] = new.TotalNomenclature - old.TotalNomenclature
	diff["total_counterparties_change"] = new.TotalCounterparties - old.TotalCounterparties

	// Находим новые загрузки
	newUploads := make([]models.UploadSummary, 0)
	oldUploadMap := make(map[string]bool)
	for _, upload := range old.UploadDetails {
		oldUploadMap[upload.UploadUUID] = true
	}
	for _, upload := range new.UploadDetails {
		if !oldUploadMap[upload.UploadUUID] {
			newUploads = append(newUploads, upload)
		}
	}
	diff["new_uploads"] = newUploads
	diff["new_uploads_count"] = len(newUploads)

	return diff
}

// Close закрывает соединение с БД
func (m *ScanHistoryManager) Close() error {
	return m.db.Close()
}
