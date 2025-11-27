package client

import (
	"context"
	"fmt"
	"time"

	"httpserver/internal/domain/repositories"
)

// service реализация domain service для client
type service struct {
	clientRepo   repositories.ClientRepository
	databaseRepo repositories.DatabaseRepository
}

// NewService создает новый domain service для client
func NewService(
	clientRepo repositories.ClientRepository,
	databaseRepo repositories.DatabaseRepository,
) Service {
	return &service{
		clientRepo:   clientRepo,
		databaseRepo: databaseRepo,
	}
}

// CreateClient создает нового клиента
func (s *service) CreateClient(ctx context.Context, req CreateClientRequest) (*Client, error) {
	if req.Name == "" {
		return nil, ErrClientNameRequired
	}

	// Проверяем, не существует ли уже клиент с таким именем
	existing, err := s.clientRepo.Search(ctx, req.Name)
	if err == nil && len(existing) > 0 {
		return nil, ErrClientExists
	}

	now := time.Now()
	client := &repositories.Client{
		Name:         req.Name,
		LegalName:    req.LegalName,
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		TaxID:        req.TaxID,
		Country:      req.Country,
		Status:       req.Status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if req.Status == "" {
		client.Status = "active"
	}

	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return s.toDomainClient(client), nil
}

// GetClient возвращает клиента по ID
func (s *service) GetClient(ctx context.Context, clientID string) (*Client, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	// Repository.GetByID принимает string, но Client.ID это int
	// Преобразуем string в int для поиска
	var clientIDInt int
	if _, err := fmt.Sscanf(clientID, "%d", &clientIDInt); err != nil {
		return nil, ErrInvalidClientID
	}

	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if client == nil {
		return nil, ErrClientNotFound
	}

	return s.toDomainClient(client), nil
}

// UpdateClient обновляет клиента
func (s *service) UpdateClient(ctx context.Context, clientID string, req UpdateClientRequest) (*Client, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if client == nil {
		return nil, ErrClientNotFound
	}

	// Обновляем только переданные поля
	if req.Name != "" {
		client.Name = req.Name
	}
	if req.LegalName != "" {
		client.LegalName = req.LegalName
	}
	if req.Description != "" {
		client.Description = req.Description
	}
	if req.ContactEmail != "" {
		client.ContactEmail = req.ContactEmail
	}
	if req.ContactPhone != "" {
		client.ContactPhone = req.ContactPhone
	}
	if req.TaxID != "" {
		client.TaxID = req.TaxID
	}
	if req.Country != "" {
		client.Country = req.Country
	}
	if req.Status != "" {
		client.Status = req.Status
	}

	client.UpdatedAt = time.Now()

	if err := s.clientRepo.Update(ctx, client); err != nil {
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	return s.toDomainClient(client), nil
}

// DeleteClient удаляет клиента
func (s *service) DeleteClient(ctx context.Context, clientID string) error {
	if clientID == "" {
		return ErrInvalidClientID
	}

	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if client == nil {
		return ErrClientNotFound
	}

	if err := s.clientRepo.Delete(ctx, clientID); err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	return nil
}

// ListClients возвращает список клиентов с фильтрацией
func (s *service) ListClients(ctx context.Context, filter repositories.ClientFilter) ([]*Client, int64, error) {
	clients, total, err := s.clientRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list clients: %w", err)
	}

	domainClients := make([]*Client, len(clients))
	for i, c := range clients {
		domainClients[i] = s.toDomainClient(&c)
	}

	return domainClients, total, nil
}

// GetClientStatistics возвращает статистику клиента
func (s *service) GetClientStatistics(ctx context.Context, clientID string) (*ClientStatistics, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if client == nil {
		return nil, ErrClientNotFound
	}

	// Получаем проекты клиента
	projects, err := s.clientRepo.GetProjects(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	// TODO: Получить статистику из других репозиториев (uploads, databases)
	stats := &ClientStatistics{
		TotalProjects:  int64(len(projects)),
		TotalDatabases: 0,
		TotalUploads:   0,
		ActiveProjects: 0,
		ActiveDatabases: 0,
	}

	// Подсчитываем активные проекты
	for _, p := range projects {
		if p.Status == "active" {
			stats.ActiveProjects++
		}
	}

	return stats, nil
}

// CreateProject создает новый проект для клиента
func (s *service) CreateProject(ctx context.Context, clientID string, req CreateProjectRequest) (*Project, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	if req.Name == "" {
		return nil, ErrProjectNameRequired
	}

	// Проверяем существование клиента
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if client == nil {
		return nil, ErrClientNotFound
	}

	// Преобразуем clientID из string в int
	var clientIDInt int
	if _, err := fmt.Sscanf(clientID, "%d", &clientIDInt); err != nil {
		return nil, ErrInvalidClientID
	}

	now := time.Now()
	project := &repositories.ClientProject{
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

	if err := s.clientRepo.CreateProject(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return s.toDomainProject(project), nil
}

// GetProject возвращает проект по ID
func (s *service) GetProject(ctx context.Context, clientID string, projectID string) (*Project, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	projects, err := s.clientRepo.GetProjects(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	for _, p := range projects {
		if fmt.Sprintf("%d", p.ID) == projectID {
			return s.toDomainProject(&p), nil
		}
	}

	return nil, ErrProjectNotFound
}

// UpdateProject обновляет проект
func (s *service) UpdateProject(ctx context.Context, clientID string, projectID string, req UpdateProjectRequest) (*Project, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	projects, err := s.clientRepo.GetProjects(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	var project *repositories.ClientProject
	for i := range projects {
		if fmt.Sprintf("%d", projects[i].ID) == projectID {
			project = &projects[i]
			break
		}
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

	// TODO: Добавить метод UpdateProject в ClientRepository
	// if err := s.clientRepo.UpdateProject(ctx, project); err != nil {
	// 	return nil, fmt.Errorf("failed to update project: %w", err)
	// }

	return s.toDomainProject(project), nil
}

// DeleteProject удаляет проект
func (s *service) DeleteProject(ctx context.Context, clientID string, projectID string) error {
	if clientID == "" {
		return ErrInvalidClientID
	}

	if projectID == "" {
		return ErrInvalidProjectID
	}

	// TODO: Добавить метод DeleteProject в ClientRepository
	return fmt.Errorf("delete project not implemented yet")
}

// ListProjects возвращает список проектов клиента
func (s *service) ListProjects(ctx context.Context, clientID string, filter repositories.ProjectFilter) ([]*Project, int64, error) {
	if clientID == "" {
		return nil, 0, ErrInvalidClientID
	}

	projects, err := s.clientRepo.GetProjects(ctx, clientID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get projects: %w", err)
	}

	domainProjects := make([]*Project, len(projects))
	for i, p := range projects {
		domainProjects[i] = s.toDomainProject(&p)
	}

	return domainProjects, int64(len(domainProjects)), nil
}

// GetProjectStatistics возвращает статистику проекта
func (s *service) GetProjectStatistics(ctx context.Context, clientID string, projectID string) (*ProjectStatistics, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	// TODO: Получить статистику из других репозиториев
	stats := &ProjectStatistics{
		TotalDatabases: 0,
		TotalUploads:   0,
		ActiveDatabases: 0,
	}

	return stats, nil
}

// GetProjectDatabases возвращает базы данных проекта
func (s *service) GetProjectDatabases(ctx context.Context, clientID string, projectID string) ([]*Database, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	if projectID == "" {
		return nil, ErrInvalidProjectID
	}

	// TODO: Получить базы данных проекта через DatabaseRepository
	return []*Database{}, nil
}

// GetClientDatabases возвращает все базы данных клиента
func (s *service) GetClientDatabases(ctx context.Context, clientID string) ([]*Database, error) {
	if clientID == "" {
		return nil, ErrInvalidClientID
	}

	// TODO: Получить базы данных клиента через DatabaseRepository
	return []*Database{}, nil
}

// toDomainClient преобразует repository Client в domain Client
func (s *service) toDomainClient(c *repositories.Client) *Client {
	return &Client{
		ID:           fmt.Sprintf("%d", c.ID),
		Name:         c.Name,
		LegalName:    c.LegalName,
		Description:  c.Description,
		ContactEmail: c.ContactEmail,
		ContactPhone: c.ContactPhone,
		TaxID:        c.TaxID,
		Country:      c.Country,
		Status:       c.Status,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}
}

// toDomainProject преобразует repository ClientProject в domain Project
func (s *service) toDomainProject(p *repositories.ClientProject) *Project {
	return &Project{
		ID:          fmt.Sprintf("%d", p.ID),
		ClientID:    fmt.Sprintf("%d", p.ClientID),
		Name:        p.Name,
		Description: p.Description,
		Type:        p.ProjectType, // Используем ProjectType из repository
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}

