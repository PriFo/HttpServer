package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"httpserver/database"
	"httpserver/internal/domain/repositories"
)

// clientRepository реализация репозитория для клиентов
// Адаптер между domain интерфейсом и infrastructure (database.ServiceDB)
type clientRepository struct {
	serviceDB *database.ServiceDB
}

// NewClientRepository создает новый репозиторий клиентов
func NewClientRepository(serviceDB *database.ServiceDB) repositories.ClientRepository {
	return &clientRepository{
		serviceDB: serviceDB,
	}
}

// Create создает нового клиента
func (r *clientRepository) Create(ctx context.Context, client *repositories.Client) error {
	createdClient, err := r.serviceDB.CreateClient(
		client.Name,
		client.LegalName,
		client.Description,
		client.ContactEmail,
		client.ContactPhone,
		client.TaxID,
		client.Country,
		"system", // createdBy
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Обновляем ID и временные метки в domain модели
	client.ID = createdClient.ID
	client.CreatedAt = createdClient.CreatedAt
	client.UpdatedAt = createdClient.UpdatedAt

	return nil
}

// GetByID возвращает клиента по ID
func (r *clientRepository) GetByID(ctx context.Context, id string) (*repositories.Client, error) {
	clientID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid client ID: %w", err)
	}

	dbClient, err := r.serviceDB.GetClient(clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if dbClient == nil {
		return nil, nil
	}

	return r.toDomainClient(dbClient), nil
}

// Update обновляет клиента
func (r *clientRepository) Update(ctx context.Context, client *repositories.Client) error {
	err := r.serviceDB.UpdateClient(
		client.ID,
		client.Name,
		client.LegalName,
		client.Description,
		client.ContactEmail,
		client.ContactPhone,
		client.TaxID,
		client.Country,
		client.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}

	return nil
}

// Delete удаляет клиента
func (r *clientRepository) Delete(ctx context.Context, id string) error {
	clientID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid client ID: %w", err)
	}

	err = r.serviceDB.DeleteClient(clientID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	return nil
}

// List возвращает список клиентов с фильтрацией
func (r *clientRepository) List(ctx context.Context, filter repositories.ClientFilter) ([]repositories.Client, int64, error) {
	// Получаем всех клиентов
	dbClients, err := r.serviceDB.GetAllClients()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get clients: %w", err)
	}

	// Применяем фильтры
	var filtered []*database.Client
	for _, c := range dbClients {
		if filter.Name != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(filter.Name)) {
			continue
		}
		if len(filter.Status) > 0 {
			found := false
			for _, status := range filter.Status {
				if c.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if filter.TaxID != "" && c.TaxID != filter.TaxID {
			continue
		}
		if filter.Email != "" && c.ContactEmail != filter.Email {
			continue
		}
		filtered = append(filtered, c)
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

	var paginated []*database.Client
	if start < len(filtered) {
		paginated = filtered[start:end]
	}

	// Преобразуем в domain модели
	domainClients := make([]repositories.Client, len(paginated))
	for i, c := range paginated {
		domainClients[i] = *r.toDomainClient(c)
	}

	return domainClients, total, nil
}

// Search ищет клиентов по запросу
func (r *clientRepository) Search(ctx context.Context, query string) ([]repositories.Client, error) {
	// Получаем всех клиентов и фильтруем по query
	dbClients, err := r.serviceDB.GetAllClients()
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}

	var results []repositories.Client
	queryLower := strings.ToLower(query)
	for _, c := range dbClients {
		if strings.Contains(strings.ToLower(c.Name), queryLower) ||
			strings.Contains(strings.ToLower(c.LegalName), queryLower) ||
			strings.Contains(strings.ToLower(c.Description), queryLower) ||
			strings.Contains(c.TaxID, query) {
			results = append(results, *r.toDomainClient(c))
		}
	}

	return results, nil
}

// GetByContactEmail возвращает клиента по email
func (r *clientRepository) GetByContactEmail(ctx context.Context, email string) (*repositories.Client, error) {
	// Получаем всех клиентов и ищем по email
	dbClients, err := r.serviceDB.GetAllClients()
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}

	for _, c := range dbClients {
		if c.ContactEmail == email {
			return r.toDomainClient(c), nil
		}
	}

	return nil, nil
}

// GetByTaxID возвращает клиента по TaxID
func (r *clientRepository) GetByTaxID(ctx context.Context, taxID string) (*repositories.Client, error) {
	// Получаем всех клиентов и ищем по TaxID
	dbClients, err := r.serviceDB.GetAllClients()
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}

	for _, c := range dbClients {
		if c.TaxID == taxID {
			return r.toDomainClient(c), nil
		}
	}

	return nil, nil
}

// GetProjects возвращает проекты клиента
func (r *clientRepository) GetProjects(ctx context.Context, clientID string) ([]repositories.ClientProject, error) {
	clientIDInt, err := strconv.Atoi(clientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client ID: %w", err)
	}

	dbProjects, err := r.serviceDB.GetClientProjects(clientIDInt)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	projects := make([]repositories.ClientProject, len(dbProjects))
	for i, p := range dbProjects {
		projects[i] = *r.toDomainProject(p)
	}

	return projects, nil
}

// CreateProject создает новый проект
func (r *clientRepository) CreateProject(ctx context.Context, project *repositories.ClientProject) error {
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

// GetBenchmarks возвращает эталоны клиента
func (r *clientRepository) GetBenchmarks(ctx context.Context, clientID string) ([]repositories.ClientBenchmark, error) {
	// TODO: Реализовать получение эталонов клиента
	// Пока возвращаем пустой список
	return []repositories.ClientBenchmark{}, nil
}

// CreateBenchmark создает новый эталон
func (r *clientRepository) CreateBenchmark(ctx context.Context, benchmark *repositories.ClientBenchmark) error {
	// TODO: Реализовать создание эталона
	return fmt.Errorf("not implemented yet")
}

// toDomainClient преобразует database.Client в repositories.Client
func (r *clientRepository) toDomainClient(c *database.Client) *repositories.Client {
	return &repositories.Client{
		ID:           c.ID,
		Name:         c.Name,
		LegalName:    c.LegalName,
		Description:  c.Description,
		ContactEmail: c.ContactEmail,
		ContactPhone: c.ContactPhone,
		TaxID:        c.TaxID,
		Country:      c.Country,
		Status:       c.Status,
		CreatedBy:    c.CreatedBy,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// toDomainProject преобразует database.ClientProject в repositories.ClientProject
func (r *clientRepository) toDomainProject(p *database.ClientProject) *repositories.ClientProject {
	return &repositories.ClientProject{
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

