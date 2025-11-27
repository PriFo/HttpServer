package client

import (
	"context"

	clientdomain "httpserver/internal/domain/client"
	"httpserver/internal/domain/repositories"
)

// UseCase представляет use case для работы с клиентами и проектами
// Координирует выполнение бизнес-логики между domain и infrastructure слоями
type UseCase struct {
	clientRepo      repositories.ClientRepository
	databaseRepo    repositories.DatabaseRepository
	clientService   clientdomain.Service
}

// NewUseCase создает новый use case для клиентов
func NewUseCase(
	clientRepo repositories.ClientRepository,
	databaseRepo repositories.DatabaseRepository,
	clientService clientdomain.Service,
) *UseCase {
	return &UseCase{
		clientRepo:    clientRepo,
		databaseRepo:  databaseRepo,
		clientService: clientService,
	}
}

// CreateClientRequest запрос на создание клиента (алиас для удобства)
type CreateClientRequest = clientdomain.CreateClientRequest

// UpdateClientRequest запрос на обновление клиента (алиас для удобства)
type UpdateClientRequest = clientdomain.UpdateClientRequest

// CreateProjectRequest запрос на создание проекта (алиас для удобства)
type CreateProjectRequest = clientdomain.CreateProjectRequest

// UpdateProjectRequest запрос на обновление проекта (алиас для удобства)
type UpdateProjectRequest = clientdomain.UpdateProjectRequest

// CreateClient создает нового клиента
func (uc *UseCase) CreateClient(ctx context.Context, req CreateClientRequest) (*clientdomain.Client, error) {
	return uc.clientService.CreateClient(ctx, req)
}

// GetClient возвращает клиента по ID
func (uc *UseCase) GetClient(ctx context.Context, clientID string) (*clientdomain.Client, error) {
	return uc.clientService.GetClient(ctx, clientID)
}

// UpdateClient обновляет клиента
func (uc *UseCase) UpdateClient(ctx context.Context, clientID string, req UpdateClientRequest) (*clientdomain.Client, error) {
	return uc.clientService.UpdateClient(ctx, clientID, req)
}

// DeleteClient удаляет клиента
func (uc *UseCase) DeleteClient(ctx context.Context, clientID string) error {
	return uc.clientService.DeleteClient(ctx, clientID)
}

// ListClients возвращает список клиентов
func (uc *UseCase) ListClients(ctx context.Context, filter repositories.ClientFilter) ([]*clientdomain.Client, int64, error) {
	return uc.clientService.ListClients(ctx, filter)
}

// GetClientStatistics возвращает статистику клиента
func (uc *UseCase) GetClientStatistics(ctx context.Context, clientID string) (*clientdomain.ClientStatistics, error) {
	return uc.clientService.GetClientStatistics(ctx, clientID)
}

// CreateProject создает новый проект для клиента
func (uc *UseCase) CreateProject(ctx context.Context, clientID string, req CreateProjectRequest) (*clientdomain.Project, error) {
	return uc.clientService.CreateProject(ctx, clientID, req)
}

// GetProject возвращает проект по ID
func (uc *UseCase) GetProject(ctx context.Context, clientID string, projectID string) (*clientdomain.Project, error) {
	return uc.clientService.GetProject(ctx, clientID, projectID)
}

// UpdateProject обновляет проект
func (uc *UseCase) UpdateProject(ctx context.Context, clientID string, projectID string, req UpdateProjectRequest) (*clientdomain.Project, error) {
	return uc.clientService.UpdateProject(ctx, clientID, projectID, req)
}

// DeleteProject удаляет проект
func (uc *UseCase) DeleteProject(ctx context.Context, clientID string, projectID string) error {
	return uc.clientService.DeleteProject(ctx, clientID, projectID)
}

// ListProjects возвращает список проектов клиента
func (uc *UseCase) ListProjects(ctx context.Context, clientID string, filter repositories.ProjectFilter) ([]*clientdomain.Project, int64, error) {
	return uc.clientService.ListProjects(ctx, clientID, filter)
}

// GetProjectStatistics возвращает статистику проекта
func (uc *UseCase) GetProjectStatistics(ctx context.Context, clientID string, projectID string) (*clientdomain.ProjectStatistics, error) {
	return uc.clientService.GetProjectStatistics(ctx, clientID, projectID)
}

// GetProjectDatabases возвращает базы данных проекта
func (uc *UseCase) GetProjectDatabases(ctx context.Context, clientID string, projectID string) ([]*clientdomain.Database, error) {
	return uc.clientService.GetProjectDatabases(ctx, clientID, projectID)
}

// GetClientDatabases возвращает все базы данных клиента
func (uc *UseCase) GetClientDatabases(ctx context.Context, clientID string) ([]*clientdomain.Database, error) {
	return uc.clientService.GetClientDatabases(ctx, clientID)
}

