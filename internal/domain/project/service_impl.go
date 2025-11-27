package project

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"httpserver/internal/domain/repositories"
)

// service реализация domain service для project
type service struct {
	projectRepo  repositories.ProjectRepository
	clientRepo   repositories.ClientRepository
	databaseRepo repositories.DatabaseRepository
}

// NewService создает новый domain service для project
func NewService(
	projectRepo repositories.ProjectRepository,
	clientRepo repositories.ClientRepository,
	databaseRepo repositories.DatabaseRepository,
) Service {
	return &service{
		projectRepo:  projectRepo,
		clientRepo:   clientRepo,
		databaseRepo: databaseRepo,
	}
}

// CreateProject создает новый проект
func (s *service) CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error) {
	if req.Name == "" {
		return nil, ErrProjectNameRequired
	}

	if req.ClientID == "" {
		return nil, ErrInvalidClientID
	}

	// Проверяем существование клиента
	client, err := s.clientRepo.GetByID(ctx, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if client == nil {
		return nil, ErrClientNotFound
	}

	// Преобразуем clientID из string в int
	clientIDInt, err := strconv.Atoi(req.ClientID)
	if err != nil {
		return nil, ErrInvalidClientID
	}

	now := time.Now()
	project := &repositories.Project{
		ClientID:     clientIDInt,
		Name:         req.Name,
		Description:  req.Description,
		ProjectType:  req.Type,
		Status:       req.Status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if req.Status == "" {
		project.Status = "active"
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return s.toDomainProject(project), nil
}

// GetProject возвращает проект по ID
func (s *service) GetProject(ctx context.Context, projectID string) (*Project, error) {
	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if project == nil {
		return nil, ErrProjectNotFound
	}

	return s.toDomainProject(project), nil
}

// UpdateProject обновляет проект
func (s *service) UpdateProject(ctx context.Context, projectID string, req UpdateProjectRequest) (*Project, error) {
	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if project == nil {
		return nil, ErrProjectNotFound
	}

	// Обновляем только переданные поля
	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Description != "" {
		project.Description = req.Description
	}
	if req.Type != "" {
		project.ProjectType = req.Type
	}
	if req.Status != "" {
		project.Status = req.Status
	}

	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return s.toDomainProject(project), nil
}

// DeleteProject удаляет проект
func (s *service) DeleteProject(ctx context.Context, projectID string) error {
	if projectID == "" {
		return ErrInvalidProjectID
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if project == nil {
		return ErrProjectNotFound
	}

	if err := s.projectRepo.Delete(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// ListProjects возвращает список проектов с фильтрацией
func (s *service) ListProjects(ctx context.Context, filter repositories.ProjectFilter) ([]*Project, int64, error) {
	projects, total, err := s.projectRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	domainProjects := make([]*Project, len(projects))
	for i, p := range projects {
		domainProjects[i] = s.toDomainProject(&p)
	}

	return domainProjects, total, nil
}

// GetProjectDatabases возвращает базы данных проекта
func (s *service) GetProjectDatabases(ctx context.Context, projectID string) ([]*repositories.Database, error) {
	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	databases, err := s.databaseRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	result := make([]*repositories.Database, len(databases))
	for i := range databases {
		result[i] = &databases[i]
	}

	return result, nil
}

// GetProjectStatistics возвращает статистику проекта
func (s *service) GetProjectStatistics(ctx context.Context, projectID string) (*ProjectStatistics, error) {
	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	// Получаем базы данных проекта
	databases, err := s.databaseRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	stats := &ProjectStatistics{
		TotalDatabases:  int64(len(databases)),
		TotalUploads:     0,
		ActiveDatabases:  0,
	}

	// Подсчитываем активные базы данных
	for _, db := range databases {
		if db.Status == "active" {
			stats.ActiveDatabases++
		}
	}

	// TODO: Получить статистику из других репозиториев (uploads, quality)

	return stats, nil
}

// toDomainProject преобразует repository Project в domain Project
func (s *service) toDomainProject(p *repositories.Project) *Project {
	return &Project{
		ID:          fmt.Sprintf("%d", p.ID),
		ClientID:    fmt.Sprintf("%d", p.ClientID),
		Name:        p.Name,
		Description: p.Description,
		Type:        p.ProjectType,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}

