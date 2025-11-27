package project

import (
	"context"

	projectdomain "httpserver/internal/domain/project"
	"httpserver/internal/domain/repositories"
)

// UseCase представляет use case для работы с проектами
// Координирует выполнение бизнес-логики между domain и infrastructure слоями
type UseCase struct {
	projectRepo   repositories.ProjectRepository
	clientRepo    repositories.ClientRepository
	databaseRepo  repositories.DatabaseRepository
	projectService projectdomain.Service
}

// NewUseCase создает новый use case для проектов
func NewUseCase(
	projectRepo repositories.ProjectRepository,
	clientRepo repositories.ClientRepository,
	databaseRepo repositories.DatabaseRepository,
	projectService projectdomain.Service,
) *UseCase {
	return &UseCase{
		projectRepo:    projectRepo,
		clientRepo:     clientRepo,
		databaseRepo:   databaseRepo,
		projectService: projectService,
	}
}

// CreateProjectRequest запрос на создание проекта (алиас для удобства)
type CreateProjectRequest = projectdomain.CreateProjectRequest

// UpdateProjectRequest запрос на обновление проекта (алиас для удобства)
type UpdateProjectRequest = projectdomain.UpdateProjectRequest

// CreateProject создает новый проект
func (uc *UseCase) CreateProject(ctx context.Context, req CreateProjectRequest) (*projectdomain.Project, error) {
	return uc.projectService.CreateProject(ctx, req)
}

// GetProject возвращает проект по ID
func (uc *UseCase) GetProject(ctx context.Context, projectID string) (*projectdomain.Project, error) {
	return uc.projectService.GetProject(ctx, projectID)
}

// UpdateProject обновляет проект
func (uc *UseCase) UpdateProject(ctx context.Context, projectID string, req UpdateProjectRequest) (*projectdomain.Project, error) {
	return uc.projectService.UpdateProject(ctx, projectID, req)
}

// DeleteProject удаляет проект
func (uc *UseCase) DeleteProject(ctx context.Context, projectID string) error {
	return uc.projectService.DeleteProject(ctx, projectID)
}

// ListProjects возвращает список проектов
func (uc *UseCase) ListProjects(ctx context.Context, filter repositories.ProjectFilter) ([]*projectdomain.Project, int64, error) {
	return uc.projectService.ListProjects(ctx, filter)
}

// GetProjectDatabases возвращает базы данных проекта
func (uc *UseCase) GetProjectDatabases(ctx context.Context, projectID string) ([]*repositories.Database, error) {
	return uc.projectService.GetProjectDatabases(ctx, projectID)
}

// GetProjectStatistics возвращает статистику проекта
func (uc *UseCase) GetProjectStatistics(ctx context.Context, projectID string) (*projectdomain.ProjectStatistics, error) {
	return uc.projectService.GetProjectStatistics(ctx, projectID)
}

