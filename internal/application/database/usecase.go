package database

import (
	"context"

	databasedomain "httpserver/internal/domain/database"
	"httpserver/internal/domain/repositories"
)

// UseCase представляет use case для работы с базами данных
// Координирует выполнение бизнес-логики между domain и infrastructure слоями
type UseCase struct {
	databaseRepo   repositories.DatabaseRepository
	projectRepo    repositories.ProjectRepository
	databaseService databasedomain.Service
}

// NewUseCase создает новый use case для баз данных
func NewUseCase(
	databaseRepo repositories.DatabaseRepository,
	projectRepo repositories.ProjectRepository,
	databaseService databasedomain.Service,
) *UseCase {
	return &UseCase{
		databaseRepo:   databaseRepo,
		projectRepo:    projectRepo,
		databaseService: databaseService,
	}
}

// CreateDatabaseRequest запрос на создание базы данных (алиас для удобства)
type CreateDatabaseRequest = databasedomain.CreateDatabaseRequest

// UpdateDatabaseRequest запрос на обновление базы данных (алиас для удобства)
type UpdateDatabaseRequest = databasedomain.UpdateDatabaseRequest

// CreateDatabase создает новую базу данных
func (uc *UseCase) CreateDatabase(ctx context.Context, req CreateDatabaseRequest) (*databasedomain.Database, error) {
	return uc.databaseService.CreateDatabase(ctx, req)
}

// GetDatabase возвращает базу данных по ID
func (uc *UseCase) GetDatabase(ctx context.Context, databaseID string) (*databasedomain.Database, error) {
	return uc.databaseService.GetDatabase(ctx, databaseID)
}

// UpdateDatabase обновляет базу данных
func (uc *UseCase) UpdateDatabase(ctx context.Context, databaseID string, req UpdateDatabaseRequest) (*databasedomain.Database, error) {
	return uc.databaseService.UpdateDatabase(ctx, databaseID, req)
}

// DeleteDatabase удаляет базу данных
func (uc *UseCase) DeleteDatabase(ctx context.Context, databaseID string) error {
	return uc.databaseService.DeleteDatabase(ctx, databaseID)
}

// ListDatabases возвращает список баз данных
func (uc *UseCase) ListDatabases(ctx context.Context, filter repositories.DatabaseFilter) ([]*databasedomain.Database, int64, error) {
	return uc.databaseService.ListDatabases(ctx, filter)
}

// GetDatabasesByProject возвращает базы данных проекта
func (uc *UseCase) GetDatabasesByProject(ctx context.Context, projectID string) ([]*databasedomain.Database, error) {
	return uc.databaseService.GetDatabasesByProject(ctx, projectID)
}

// GetDatabasesByClient возвращает базы данных клиента
func (uc *UseCase) GetDatabasesByClient(ctx context.Context, clientID string) ([]*databasedomain.Database, error) {
	return uc.databaseService.GetDatabasesByClient(ctx, clientID)
}

// TestConnection проверяет подключение к базе данных
func (uc *UseCase) TestConnection(ctx context.Context, databaseID string) error {
	return uc.databaseService.TestConnection(ctx, databaseID)
}

// GetConnectionStatus возвращает статус подключения
func (uc *UseCase) GetConnectionStatus(ctx context.Context, databaseID string) (string, error) {
	return uc.databaseService.GetConnectionStatus(ctx, databaseID)
}

// GetDatabaseStatistics возвращает статистику базы данных
func (uc *UseCase) GetDatabaseStatistics(ctx context.Context, databaseID string) (*databasedomain.DatabaseStatistics, error) {
	return uc.databaseService.GetDatabaseStatistics(ctx, databaseID)
}

