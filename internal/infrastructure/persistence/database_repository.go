package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/internal/domain/repositories"
)

// databaseRepository реализация репозитория для баз данных
// Адаптер между domain интерфейсом и infrastructure (database.ServiceDB)
type databaseRepository struct {
	serviceDB *database.ServiceDB
}

// NewDatabaseRepository создает новый репозиторий баз данных
func NewDatabaseRepository(serviceDB *database.ServiceDB) repositories.DatabaseRepository {
	return &databaseRepository{
		serviceDB: serviceDB,
	}
}

// Create создает новую базу данных
func (r *databaseRepository) Create(ctx context.Context, db *repositories.Database) error {
	// TODO: Реализовать создание базы данных через ServiceDB
	// Пока возвращаем ошибку, так как метод CreateProjectDatabase требует projectID
	return fmt.Errorf("not implemented yet - use CreateProjectDatabase instead")
}

// GetByID возвращает базу данных по ID
func (r *databaseRepository) GetByID(ctx context.Context, id string) (*repositories.Database, error) {
	dbID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid database ID: %w", err)
	}

	projectDB, err := r.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	if projectDB == nil {
		return nil, nil
	}

	return r.toDomainDatabase(projectDB), nil
}

// Update обновляет базу данных
func (r *databaseRepository) Update(ctx context.Context, db *repositories.Database) error {
	dbID, err := strconv.Atoi(db.ID)
	if err != nil {
		return fmt.Errorf("invalid database ID: %w", err)
	}

	err = r.serviceDB.UpdateProjectDatabase(
		dbID,
		db.Name,
		db.Path,
		db.ConnectionString,
		db.Status == "active",
	)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	return nil
}

// Delete удаляет базу данных
func (r *databaseRepository) Delete(ctx context.Context, id string) error {
	dbID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid database ID: %w", err)
	}

	err = r.serviceDB.DeleteProjectDatabase(dbID)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	return nil
}

// List возвращает список баз данных с фильтрацией
func (r *databaseRepository) List(ctx context.Context, filter repositories.DatabaseFilter) ([]repositories.Database, int64, error) {
	// Получаем все базы данных через GetAllProjectDatabases
	dbDatabases, err := r.serviceDB.GetAllProjectDatabases()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get databases: %w", err)
	}

	// Применяем фильтры
	var filtered []*database.ProjectDatabase
	for _, d := range dbDatabases {
		if filter.Name != "" && !strings.Contains(strings.ToLower(d.Name), strings.ToLower(filter.Name)) {
			continue
		}
		if filter.Type != "" && d.Name != filter.Type {
			continue
		}
		if len(filter.Status) > 0 {
			found := false
			status := "active"
			if !d.IsActive {
				status = "inactive"
			}
			for _, s := range filter.Status {
				if status == s {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if filter.Path != "" && !strings.Contains(d.FilePath, filter.Path) {
			continue
		}
		filtered = append(filtered, d)
	}

	total := int64(len(filtered))

	// Применяем пагинацию
	start := filter.Offset
	end := start + filter.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	if start > len(filtered) {
		start = len(filtered)
	}

	var paginated []*database.ProjectDatabase
	if start < len(filtered) {
		paginated = filtered[start:end]
	}

	// Преобразуем в domain модели
	domainDatabases := make([]repositories.Database, len(paginated))
	for i, d := range paginated {
		domainDatabases[i] = *r.toDomainDatabase(d)
	}

	return domainDatabases, total, nil
}

// GetByPath возвращает базу данных по пути
func (r *databaseRepository) GetByPath(ctx context.Context, path string) (*repositories.Database, error) {
	// Получаем все базы данных и ищем по пути
	dbDatabases, err := r.serviceDB.GetAllProjectDatabases()
	if err != nil {
		return nil, fmt.Errorf("failed to get databases: %w", err)
	}

	for _, d := range dbDatabases {
		if d.FilePath == path {
			return r.toDomainDatabase(d), nil
		}
	}

	return nil, nil
}

// GetByConnectionString возвращает базу данных по строке подключения
func (r *databaseRepository) GetByConnectionString(ctx context.Context, connectionString string) (*repositories.Database, error) {
	// Получаем все базы данных и ищем по connection string
	dbDatabases, err := r.serviceDB.GetAllProjectDatabases()
	if err != nil {
		return nil, fmt.Errorf("failed to get databases: %w", err)
	}

	for _, d := range dbDatabases {
		if d.FilePath == connectionString {
			return r.toDomainDatabase(d), nil
		}
	}

	return nil, nil
}

// GetByProjectID возвращает базы данных проекта
func (r *databaseRepository) GetByProjectID(ctx context.Context, projectID string) ([]repositories.Database, error) {
	projectIDInt, err := strconv.Atoi(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	dbDatabases, err := r.serviceDB.GetProjectDatabases(projectIDInt, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	databases := make([]repositories.Database, len(dbDatabases))
	for i, d := range dbDatabases {
		databases[i] = *r.toDomainDatabase(d)
	}

	return databases, nil
}

// TestConnection проверяет подключение к базе данных
func (r *databaseRepository) TestConnection(ctx context.Context, db *repositories.Database) error {
	// TODO: Реализовать проверку подключения
	return fmt.Errorf("not implemented yet")
}

// GetConnectionStatus возвращает статус подключения
func (r *databaseRepository) GetConnectionStatus(ctx context.Context, id string) (string, error) {
	db, err := r.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if db == nil {
		return "", fmt.Errorf("database not found")
	}
	return db.Status, nil
}

// GetSchema возвращает схему базы данных
func (r *databaseRepository) GetSchema(ctx context.Context, id string) (*repositories.DatabaseSchema, error) {
	// TODO: Реализовать получение схемы
	return nil, fmt.Errorf("not implemented yet")
}

// GetTables возвращает список таблиц
func (r *databaseRepository) GetTables(ctx context.Context, id string) ([]string, error) {
	// TODO: Реализовать получение таблиц
	return nil, fmt.Errorf("not implemented yet")
}

// GetColumns возвращает колонки таблицы
func (r *databaseRepository) GetColumns(ctx context.Context, tableID string) ([]repositories.Column, error) {
	// TODO: Реализовать получение колонок
	return nil, fmt.Errorf("not implemented yet")
}

// CreateBackup создает бэкап базы данных
func (r *databaseRepository) CreateBackup(ctx context.Context, dbID string) (*repositories.Backup, error) {
	// TODO: Реализовать создание бэкапа
	return nil, fmt.Errorf("not implemented yet")
}

// RestoreBackup восстанавливает базу данных из бэкапа
func (r *databaseRepository) RestoreBackup(ctx context.Context, backupID string) error {
	// TODO: Реализовать восстановление из бэкапа
	return fmt.Errorf("not implemented yet")
}

// GetBackups возвращает список бэкапов
func (r *databaseRepository) GetBackups(ctx context.Context, dbID string) ([]repositories.Backup, error) {
	// TODO: Реализовать получение бэкапов
	return nil, fmt.Errorf("not implemented yet")
}

// toDomainDatabase преобразует database.ProjectDatabase в repositories.Database
func (r *databaseRepository) toDomainDatabase(d *database.ProjectDatabase) *repositories.Database {
	status := "active"
	if !d.IsActive {
		status = "inactive"
	}

	var lastConnected *time.Time
	if d.LastUsedAt != nil {
		lastConnected = d.LastUsedAt
	}

	return &repositories.Database{
		ID:               strconv.Itoa(d.ID),
		Name:             d.Name,
		Type:             "sqlite", // По умолчанию SQLite
		Path:             d.FilePath,
		ConnectionString: d.FilePath,
		Status:           status,
		LastConnected:    lastConnected,
		SchemaVersion:    "", // TODO: Добавить schema version если нужно
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

