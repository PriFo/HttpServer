package persistence

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"httpserver/database"
	"httpserver/internal/domain/repositories"
)

// projectRepository реализация репозитория для проектов
// Адаптер между domain интерфейсом и infrastructure (database.ServiceDB)
type projectRepository struct {
	serviceDB *database.ServiceDB
}

// NewProjectRepository создает новый репозиторий проектов
func NewProjectRepository(serviceDB *database.ServiceDB) repositories.ProjectRepository {
	return &projectRepository{
		serviceDB: serviceDB,
	}
}

// Create создает новый проект
func (r *projectRepository) Create(ctx context.Context, project *repositories.Project) error {
	createdProject, err := r.serviceDB.CreateClientProject(
		project.ClientID,
		project.Name,
		project.ProjectType,
		project.Description,
		project.SourceSystem,
		project.TargetQualityScore,
	)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Обновляем поля в domain модели
	project.ID = createdProject.ID
	project.ClientID = createdProject.ClientID
	project.CreatedAt = createdProject.CreatedAt
	project.UpdatedAt = createdProject.UpdatedAt

	return nil
}

// GetByID возвращает проект по ID
func (r *projectRepository) GetByID(ctx context.Context, id string) (*repositories.Project, error) {
	projectID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Получаем проект через ServiceDB
	// ServiceDB не имеет метода GetProject, поэтому получаем через GetClientProjects всех клиентов
	// Нужно найти проект по ID среди всех проектов
	// Для этого получаем всех клиентов и их проекты
	allClients, err := r.serviceDB.GetAllClients()
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}

	for _, client := range allClients {
		projects, err := r.serviceDB.GetClientProjects(client.ID)
		if err != nil {
			continue
		}
		for _, p := range projects {
			if p.ID == projectID {
				return r.toDomainProject(p), nil
			}
		}
	}

	return nil, nil
}

// Update обновляет проект
func (r *projectRepository) Update(ctx context.Context, project *repositories.Project) error {
	projectID := project.ID

	err := r.serviceDB.UpdateClientProject(
		projectID,
		project.Name,
		project.ProjectType,
		project.Description,
		project.SourceSystem,
		project.Status,
		project.TargetQualityScore,
	)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// Delete удаляет проект
func (r *projectRepository) Delete(ctx context.Context, id string) error {
	projectID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	err = r.serviceDB.DeleteClientProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// List возвращает список проектов с фильтрацией
func (r *projectRepository) List(ctx context.Context, filter repositories.ProjectFilter) ([]repositories.Project, int64, error) {
	// Получаем все проекты через всех клиентов
	allClients, err := r.serviceDB.GetAllClients()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get clients: %w", err)
	}

	var allProjects []*database.ClientProject
	for _, client := range allClients {
		projects, err := r.serviceDB.GetClientProjects(client.ID)
		if err != nil {
			continue
		}
		allProjects = append(allProjects, projects...)
	}

	// Применяем фильтры
	var filtered []*database.ClientProject
	for _, p := range allProjects {
		if filter.Name != "" && !strings.Contains(strings.ToLower(p.Name), strings.ToLower(filter.Name)) {
			continue
		}
		if filter.Type != "" && p.ProjectType != filter.Type {
			continue
		}
		if len(filter.Status) > 0 {
			found := false
			for _, status := range filter.Status {
				if p.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if filter.ClientID != "" {
			clientIDInt, err := strconv.Atoi(filter.ClientID)
			if err == nil && p.ClientID != clientIDInt {
				continue
			}
		}
		filtered = append(filtered, p)
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

	var paginated []*database.ClientProject
	if start < len(filtered) {
		paginated = filtered[start:end]
	}

	// Преобразуем в domain модели
	domainProjects := make([]repositories.Project, len(paginated))
	for i, p := range paginated {
		domainProjects[i] = *r.toDomainProject(p)
	}

	return domainProjects, total, nil
}

// GetByClientID возвращает проекты клиента
func (r *projectRepository) GetByClientID(ctx context.Context, clientID string) ([]repositories.Project, error) {
	clientIDInt, err := strconv.Atoi(clientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client ID: %w", err)
	}

	dbProjects, err := r.serviceDB.GetClientProjects(clientIDInt)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	projects := make([]repositories.Project, len(dbProjects))
	for i, p := range dbProjects {
		projects[i] = *r.toDomainProject(p)
	}

	return projects, nil
}

// Search ищет проекты по запросу
func (r *projectRepository) Search(ctx context.Context, query string) ([]repositories.Project, error) {
	// Получаем все проекты через всех клиентов
	allClients, err := r.serviceDB.GetAllClients()
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}

	var allProjects []*database.ClientProject
	for _, client := range allClients {
		projects, err := r.serviceDB.GetClientProjects(client.ID)
		if err != nil {
			continue
		}
		allProjects = append(allProjects, projects...)
	}

	var results []repositories.Project
	queryLower := strings.ToLower(query)
	for _, p := range allProjects {
		if strings.Contains(strings.ToLower(p.Name), queryLower) ||
			strings.Contains(strings.ToLower(p.Description), queryLower) {
			results = append(results, *r.toDomainProject(p))
		}
	}

	return results, nil
}

// toDomainProject преобразует database.ClientProject в repositories.Project
func (r *projectRepository) toDomainProject(p *database.ClientProject) *repositories.Project {
	return &repositories.Project{
		ID:                 p.ID,
		ClientID:           p.ClientID,
		Name:               p.Name,
		ProjectType:        p.ProjectType,
		Description:        p.Description,
		SourceSystem:       p.SourceSystem,
		Status:             p.Status,
		TargetQualityScore: p.TargetQualityScore,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

