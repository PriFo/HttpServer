package database

import (
	"context"
	"fmt"
	"time"

	"httpserver/internal/domain/repositories"
)

// service реализация domain service для database
type service struct {
	databaseRepo repositories.DatabaseRepository
	projectRepo  repositories.ProjectRepository
}

// NewService создает новый domain service для database
func NewService(
	databaseRepo repositories.DatabaseRepository,
	projectRepo repositories.ProjectRepository,
) Service {
	return &service{
		databaseRepo: databaseRepo,
		projectRepo:  projectRepo,
	}
}

// CreateDatabase создает новую базу данных
func (s *service) CreateDatabase(ctx context.Context, req CreateDatabaseRequest) (*Database, error) {
	if req.Name == "" {
		return nil, ErrDatabaseNameRequired
	}

	if req.FilePath == "" {
		return nil, ErrDatabasePathRequired
	}

	if req.ProjectID == "" {
		return nil, ErrInvalidProjectID
	}

	// Проверяем существование проекта
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if project == nil {
		return nil, ErrProjectNotFound
	}

	// Проверяем, не существует ли уже база данных с таким путем
	existing, err := s.databaseRepo.GetByPath(ctx, req.FilePath)
	if err == nil && existing != nil {
		return nil, ErrDatabaseExists
	}

	now := time.Now()
	status := "active"
	if !req.IsActive {
		status = "inactive"
	}

	db := &repositories.Database{
		Name:             req.Name,
		Path:             req.FilePath,
		ConnectionString: req.FilePath,
		Type:             "sqlite", // По умолчанию SQLite
		Status:           status,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.databaseRepo.Create(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	clientIDStr := fmt.Sprintf("%d", project.ClientID)
	return s.toDomainDatabase(db, clientIDStr, req.ProjectID), nil
}

// GetDatabase возвращает базу данных по ID
func (s *service) GetDatabase(ctx context.Context, databaseID string) (*Database, error) {
	if databaseID == "" {
		return nil, ErrInvalidDatabaseID
	}

	db, err := s.databaseRepo.GetByID(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	if db == nil {
		return nil, ErrDatabaseNotFound
	}

	// Получаем projectID из базы данных (нужно будет добавить это поле в модель)
	// Пока используем пустые значения
	return s.toDomainDatabase(db, "", ""), nil
}

// UpdateDatabase обновляет базу данных
func (s *service) UpdateDatabase(ctx context.Context, databaseID string, req UpdateDatabaseRequest) (*Database, error) {
	if databaseID == "" {
		return nil, ErrInvalidDatabaseID
	}

	db, err := s.databaseRepo.GetByID(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	if db == nil {
		return nil, ErrDatabaseNotFound
	}

	// Обновляем только переданные поля
	if req.Name != "" {
		db.Name = req.Name
	}
	if req.FilePath != "" {
		db.Path = req.FilePath
		db.ConnectionString = req.FilePath
	}
	if req.Description != "" {
		// TODO: Добавить Description в repositories.Database модель
	}
	if req.IsActive != nil {
		if *req.IsActive {
			db.Status = "active"
		} else {
			db.Status = "inactive"
		}
	}

	db.UpdatedAt = time.Now()

	if err := s.databaseRepo.Update(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to update database: %w", err)
	}

	return s.toDomainDatabase(db, "", ""), nil
}

// DeleteDatabase удаляет базу данных
func (s *service) DeleteDatabase(ctx context.Context, databaseID string) error {
	if databaseID == "" {
		return ErrInvalidDatabaseID
	}

	db, err := s.databaseRepo.GetByID(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	if db == nil {
		return ErrDatabaseNotFound
	}

	if err := s.databaseRepo.Delete(ctx, databaseID); err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	return nil
}

// ListDatabases возвращает список баз данных с фильтрацией
func (s *service) ListDatabases(ctx context.Context, filter repositories.DatabaseFilter) ([]*Database, int64, error) {
	databases, total, err := s.databaseRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list databases: %w", err)
	}

	domainDatabases := make([]*Database, len(databases))
	for i, d := range databases {
		domainDatabases[i] = s.toDomainDatabase(&d, "", "")
	}

	return domainDatabases, total, nil
}

// GetDatabasesByProject возвращает базы данных проекта
func (s *service) GetDatabasesByProject(ctx context.Context, projectID string) ([]*Database, error) {
	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	databases, err := s.databaseRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	domainDatabases := make([]*Database, len(databases))
	for i, d := range databases {
		domainDatabases[i] = s.toDomainDatabase(&d, "", projectID)
	}

	return domainDatabases, nil
}

// GetDatabasesByClient возвращает базы данных клиента
func (s *service) GetDatabasesByClient(ctx context.Context, clientID string) ([]*Database, error) {
	// TODO: Реализовать получение баз данных по clientID
	// Пока возвращаем пустой список
	return []*Database{}, nil
}

// TestConnection проверяет подключение к базе данных
func (s *service) TestConnection(ctx context.Context, databaseID string) error {
	if databaseID == "" {
		return ErrInvalidDatabaseID
	}

	db, err := s.databaseRepo.GetByID(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	if db == nil {
		return ErrDatabaseNotFound
	}

	if err := s.databaseRepo.TestConnection(ctx, db); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// GetConnectionStatus возвращает статус подключения
func (s *service) GetConnectionStatus(ctx context.Context, databaseID string) (string, error) {
	if databaseID == "" {
		return "", ErrInvalidDatabaseID
	}

	status, err := s.databaseRepo.GetConnectionStatus(ctx, databaseID)
	if err != nil {
		return "", fmt.Errorf("failed to get connection status: %w", err)
	}

	return status, nil
}

// GetDatabaseStatistics возвращает статистику базы данных
func (s *service) GetDatabaseStatistics(ctx context.Context, databaseID string) (*DatabaseStatistics, error) {
	if databaseID == "" {
		return nil, ErrInvalidDatabaseID
	}

	status, err := s.databaseRepo.GetConnectionStatus(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection status: %w", err)
	}

	stats := &DatabaseStatistics{
		TotalUploads:     0,
		TotalRecords:     0,
		ConnectionStatus: status,
	}

	// TODO: Получить статистику из других репозиториев (uploads, quality)

	return stats, nil
}

// toDomainDatabase преобразует repository Database в domain Database
func (s *service) toDomainDatabase(d *repositories.Database, clientID, projectID string) *Database {
	isActive := d.Status == "active"
	
	var lastUsedAt string
	if d.LastConnected != nil {
		lastUsedAt = d.LastConnected.Format(time.RFC3339)
	}

	return &Database{
		ID:          d.ID,
		ClientID:    clientID,
		ProjectID:   projectID,
		Name:        d.Name,
		FilePath:    d.Path,
		Description: "", // TODO: Добавить Description в repositories.Database
		IsActive:    isActive,
		LastUsedAt:  lastUsedAt,
		CreatedAt:   d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
	}
}

