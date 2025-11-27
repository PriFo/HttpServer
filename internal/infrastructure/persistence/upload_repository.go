package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"httpserver/database"
	"httpserver/internal/domain/repositories"
)

// uploadRepository реализация репозитория для выгрузок
// Адаптер между domain интерфейсом и infrastructure (database.DB)
type uploadRepository struct {
	db *database.DB
}

// NewUploadRepository создает новый репозиторий выгрузок
func NewUploadRepository(db *database.DB) repositories.UploadRepository {
	return &uploadRepository{
		db: db,
	}
}

// Create создает новую выгрузку
func (r *uploadRepository) Create(ctx context.Context, upload *repositories.Upload) error {
	var databaseID *int
	if upload.DatabaseID != "" {
		dbID, err := parseInt(upload.DatabaseID)
		if err == nil {
			databaseID = &dbID
		}
	}

	var parentUploadID *int
	// TODO: Добавить поддержку ParentUploadID если нужно

	// Используем существующий метод database.DB
	createdUpload, err := r.db.CreateUploadWithDatabase(
		upload.UUID,
		upload.Version1C,
		upload.ConfigName,
		databaseID,
		upload.ComputerName,
		upload.UserName,
		upload.ConfigVersion,
		1, // iterationNumber
		"", // iterationLabel
		"", // programmerName
		"", // uploadPurpose
		parentUploadID,
	)
	if err != nil {
		return fmt.Errorf("failed to create upload: %w", err)
	}

	// Обновляем ID созданной выгрузки
	upload.ID = fmt.Sprintf("%d", createdUpload.ID)

	return nil
}

// GetByID возвращает выгрузку по ID
func (r *uploadRepository) GetByID(ctx context.Context, id string) (*repositories.Upload, error) {
	uploadID, err := parseInt(id)
	if err != nil {
		return nil, fmt.Errorf("invalid upload ID: %w", err)
	}

	dbUpload, err := r.db.GetUploadByID(uploadID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("upload not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get upload: %w", err)
	}

	return r.toDomainUpload(dbUpload), nil
}

// GetByUUID возвращает выгрузку по UUID
func (r *uploadRepository) GetByUUID(ctx context.Context, uuid string) (*repositories.Upload, error) {
	dbUpload, err := r.db.GetUploadByUUID(uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("upload not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get upload: %w", err)
	}

	return r.toDomainUpload(dbUpload), nil
}

// Update обновляет выгрузку
func (r *uploadRepository) Update(ctx context.Context, upload *repositories.Upload) error {
	uploadID, err := parseInt(upload.ID)
	if err != nil {
		return fmt.Errorf("invalid upload ID: %w", err)
	}

	// Используем прямой SQL запрос для обновления
	query := `
		UPDATE uploads 
		SET status = ?, 
		    completed_at = ?,
		    total_constants = ?,
		    total_catalogs = ?,
		    total_items = ?,
		    error_count = ?,
		    error_message = ?,
		    metadata = ?
		WHERE id = ?
	`

	var completedAt interface{}
	if upload.CompletedAt != nil {
		completedAt = *upload.CompletedAt
	}

	_, err = r.db.GetDB().ExecContext(ctx, query,
		upload.Status,
		completedAt,
		upload.TotalConstants,
		upload.TotalCatalogs,
		upload.TotalItems,
		upload.ErrorCount,
		upload.ErrorMessage,
		upload.Metadata,
		uploadID,
	)

	if err != nil {
		return fmt.Errorf("failed to update upload: %w", err)
	}

	return nil
}

// Delete удаляет выгрузку
func (r *uploadRepository) Delete(ctx context.Context, id string) error {
	uploadID, err := parseInt(id)
	if err != nil {
		return fmt.Errorf("invalid upload ID: %w", err)
	}

	// Используем прямой SQL запрос для удаления
	query := `DELETE FROM uploads WHERE id = ?`
	_, err = r.db.ExecContext(ctx, query, uploadID)
	if err != nil {
		return fmt.Errorf("failed to delete upload: %w", err)
	}

	return nil
}

// List возвращает список выгрузок с фильтрацией
func (r *uploadRepository) List(ctx context.Context, filter repositories.UploadFilter) ([]repositories.Upload, int64, error) {
	// Получаем все выгрузки
	dbUploads, err := r.db.GetAllUploads()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get uploads: %w", err)
	}

	// Преобразуем в domain модели
	uploads := make([]repositories.Upload, 0, len(dbUploads))
	for _, dbUpload := range dbUploads {
		uploads = append(uploads, *r.toDomainUpload(dbUpload))
	}

	// Применяем фильтры
	filteredUploads := r.applyFilters(uploads, filter)

	// Применяем пагинацию
	total := int64(len(filteredUploads))
	if filter.Limit > 0 {
		offset := filter.Offset
		if offset < 0 {
			offset = 0
		}
		limit := filter.Limit
		if offset >= len(filteredUploads) {
			return []repositories.Upload{}, total, nil
		}
		end := offset + limit
		if end > len(filteredUploads) {
			end = len(filteredUploads)
		}
		filteredUploads = filteredUploads[offset:end]
	}

	return filteredUploads, total, nil
}

// GetByDatabaseID возвращает выгрузки по ID базы данных
func (r *uploadRepository) GetByDatabaseID(ctx context.Context, databaseID string) ([]repositories.Upload, error) {
	dbID, err := parseInt(databaseID)
	if err != nil {
		return nil, fmt.Errorf("invalid database ID: %w", err)
	}

	dbUploads, err := r.db.GetUploadsByDatabaseID(dbID)
	if err != nil {
		return nil, fmt.Errorf("failed to get uploads by database ID: %w", err)
	}

	uploads := make([]repositories.Upload, 0, len(dbUploads))
	for _, dbUpload := range dbUploads {
		uploads = append(uploads, *r.toDomainUpload(dbUpload))
	}

	return uploads, nil
}

// GetByStatus возвращает выгрузки по статусу
func (r *uploadRepository) GetByStatus(ctx context.Context, status string) ([]repositories.Upload, error) {
	filter := repositories.UploadFilter{
		Status: []string{status},
	}

	uploads, _, err := r.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return uploads, nil
}

// GetByDateRange возвращает выгрузки за период
func (r *uploadRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]repositories.Upload, error) {
	filter := repositories.UploadFilter{
		DateFrom: &start,
		DateTo:   &end,
	}

	uploads, _, err := r.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return uploads, nil
}

// GetStatistics возвращает статистику выгрузок
func (r *uploadRepository) GetStatistics(ctx context.Context, databaseID string) (*repositories.UploadStatistics, error) {
	// TODO: Реализовать подсчет статистики
	return &repositories.UploadStatistics{}, nil
}

// GetRecentUploads возвращает последние выгрузки
func (r *uploadRepository) GetRecentUploads(ctx context.Context, limit int) ([]repositories.Upload, error) {
	filter := repositories.UploadFilter{
		Limit:  limit,
		Offset: 0,
	}

	uploads, _, err := r.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return uploads, nil
}

// BatchCreate создает несколько выгрузок
func (r *uploadRepository) BatchCreate(ctx context.Context, uploads []repositories.Upload) error {
	for _, upload := range uploads {
		if err := r.Create(ctx, &upload); err != nil {
			return fmt.Errorf("failed to create upload %s: %w", upload.UUID, err)
		}
	}
	return nil
}

// BatchUpdateStatus обновляет статус нескольких выгрузок
func (r *uploadRepository) BatchUpdateStatus(ctx context.Context, ids []string, status string) error {
	for _, id := range ids {
		upload, err := r.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get upload %s: %w", id, err)
		}
		upload.Status = status
		if err := r.Update(ctx, upload); err != nil {
			return fmt.Errorf("failed to update upload %s: %w", id, err)
		}
	}
	return nil
}

// Вспомогательные методы

// toDomainUpload преобразует database.Upload в repositories.Upload
func (r *uploadRepository) toDomainUpload(dbUpload *database.Upload) *repositories.Upload {
	upload := &repositories.Upload{
		ID:            fmt.Sprintf("%d", dbUpload.ID),
		UUID:          dbUpload.UploadUUID,
		Version1C:     dbUpload.Version1C,
		ConfigName:    dbUpload.ConfigName,
		ConfigVersion: dbUpload.ConfigVersion,
		ComputerName:  dbUpload.ComputerName,
		UserName:      dbUpload.UserName,
		Status:        dbUpload.Status,
		StartedAt:     dbUpload.StartedAt, // time.Time напрямую
		CompletedAt:   dbUpload.CompletedAt, // *time.Time напрямую
		TotalConstants: dbUpload.TotalConstants,
		TotalCatalogs:  dbUpload.TotalCatalogs,
		TotalItems:     dbUpload.TotalItems,
		ProcessedCount: 0, // TODO: Добавить это поле в database.Upload если нужно
		ErrorCount:     0, // TODO: Добавить это поле в database.Upload если нужно
		ErrorMessage:   "", // TODO: Добавить это поле в database.Upload если нужно
		Metadata:       "", // TODO: Добавить это поле в database.Upload если нужно
		CreatedAt:      dbUpload.StartedAt, // Используем StartedAt как CreatedAt
		UpdatedAt:      dbUpload.StartedAt, // TODO: Добавить UpdatedAt в database.Upload
	}

	if dbUpload.DatabaseID != nil {
		upload.DatabaseID = fmt.Sprintf("%d", *dbUpload.DatabaseID)
	}

	return upload
}

// applyFilters применяет фильтры к списку выгрузок
func (r *uploadRepository) applyFilters(uploads []repositories.Upload, filter repositories.UploadFilter) []repositories.Upload {
	filtered := make([]repositories.Upload, 0)

	for _, upload := range uploads {
		// Фильтр по статусу
		if len(filter.Status) > 0 {
			match := false
			for _, status := range filter.Status {
				if upload.Status == status {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// Фильтр по database_id
		if filter.DatabaseID != "" && upload.DatabaseID != filter.DatabaseID {
			continue
		}

		// Фильтр по дате
		startTime := upload.StartedAt // Already time.Time
		if filter.DateFrom != nil {
			if startTime.Before(*filter.DateFrom) {
				continue
			}
		}
		if filter.DateTo != nil {
			if startTime.After(*filter.DateTo) {
				continue
			}
		}

		// Фильтр по версии 1С
		if filter.Version1C != "" && upload.Version1C != filter.Version1C {
			continue
		}

		// Фильтр по имени конфигурации
		if filter.ConfigName != "" && upload.ConfigName != filter.ConfigName {
			continue
		}

		filtered = append(filtered, upload)
	}

	return filtered
}

// Вспомогательные функции для преобразования типов

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func parseTimePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t := parseTime(*s)
	return &t
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatTime(*t)
	return &s
}

