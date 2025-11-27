package cache

import (
	"context"
	"os"
	"sync"
	"time"

	"httpserver/internal/domain/models"
)

// DatabaseModificationTracker отслеживает изменения файлов БД
type DatabaseModificationTracker struct {
	mu           sync.RWMutex
	lastModified map[string]time.Time // путь к БД -> время последней модификации
}

// NewDatabaseModificationTracker создает новый трекер изменений
func NewDatabaseModificationTracker() *DatabaseModificationTracker {
	return &DatabaseModificationTracker{
		lastModified: make(map[string]time.Time),
	}
}

// GetLastModified возвращает время последней модификации файла БД
func (t *DatabaseModificationTracker) GetLastModified(dbPath string) (time.Time, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	lastMod, exists := t.lastModified[dbPath]
	return lastMod, exists
}

// UpdateLastModified обновляет время последней модификации
func (t *DatabaseModificationTracker) UpdateLastModified(dbPath string, modTime time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastModified[dbPath] = modTime
}

// ShouldScan проверяет, нужно ли сканировать БД (изменилась ли она)
func (t *DatabaseModificationTracker) ShouldScan(dbPath string) bool {
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		return true // Если файл не найден, сканируем (чтобы обнаружить проблему)
	}

	currentModTime := fileInfo.ModTime()
	lastMod, exists := t.lastModified[dbPath]

	if !exists {
		// Первое сканирование - сохраняем время и сканируем
		t.UpdateLastModified(dbPath, currentModTime)
		return true
	}

	// Сканируем только если файл изменился
	if currentModTime.After(lastMod) {
		t.UpdateLastModified(dbPath, currentModTime)
		return true
	}

	return false
}

// ScanAndSummarizeAllDatabases выполняет полное сканирование всех баз данных
// и возвращает сводную статистику по системе
func ScanAndSummarizeAllDatabases(ctx context.Context, serviceDBPath, mainDBPath string) (*models.SystemSummary, error) {
	// TODO: Implement actual database scanning logic
	// For now, returning a placeholder implementation
	return &models.SystemSummary{
		TotalDatabases:      0,
		TotalUploads:        0,
		CompletedUploads:    0,
		FailedUploads:       0,
		InProgressUploads:   0,
		TotalNomenclature:   0,
		TotalCounterparties: 0,
		UploadDetails:       []models.UploadSummary{},
	}, nil
}

// ScanAndSummarizeAllDatabasesIncremental выполняет инкрементальное сканирование
// Сканирует только те БД, которые изменились с последнего сканирования
func ScanAndSummarizeAllDatabasesIncremental(ctx context.Context, serviceDBPath, mainDBPath string, tracker *DatabaseModificationTracker) (*models.SystemSummary, error) {
	// Получаем полную сводку
	fullSummary, err := ScanAndSummarizeAllDatabases(ctx, serviceDBPath, mainDBPath)
	if err != nil {
		return nil, err
	}

	// Фильтруем только измененные БД
	if tracker != nil {
		filteredDetails := make([]models.UploadSummary, 0)
		for _, upload := range fullSummary.UploadDetails {
			if upload.DatabaseFile != "" {
				if tracker.ShouldScan(upload.DatabaseFile) {
					filteredDetails = append(filteredDetails, upload)
				}
			} else {
				// Если путь к БД не указан, включаем в результат
				filteredDetails = append(filteredDetails, upload)
			}
		}

		// Обновляем сводку с отфильтрованными данными
		fullSummary.UploadDetails = filteredDetails
		fullSummary.TotalUploads = int64(len(filteredDetails))

		// Пересчитываем статистику
		fullSummary.CompletedUploads = 0
		fullSummary.FailedUploads = 0
		fullSummary.InProgressUploads = 0
		fullSummary.TotalNomenclature = 0
		fullSummary.TotalCounterparties = 0

		for _, upload := range filteredDetails {
			switch upload.Status {
			case "completed":
				fullSummary.CompletedUploads++
			case "failed":
				fullSummary.FailedUploads++
			case "in_progress":
				fullSummary.InProgressUploads++
			}
			fullSummary.TotalNomenclature += upload.NomenclatureCount
			fullSummary.TotalCounterparties += upload.CounterpartyCount
		}
	}

	return fullSummary, nil
}
