package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/normalization"
	apperrors "httpserver/server/errors"
)

// ClientService сервис для работы с клиентами, проектами и базами данных.
// Предоставляет методы для управления клиентами, их проектами и базами данных.
//
// Все методы принимают context.Context для поддержки отмены и таймаутов.
// Методы валидируют входные параметры и возвращают понятные ошибки.
//
// Пример использования:
//
//	service, err := NewClientService(serviceDB, db, normalizedDB)
//	if err != nil {
//	    return err
//	}
//	clients, err := service.GetAllClients(ctx)
//	if err != nil {
//	    return err
//	}
type ClientService struct {
	serviceDB    *database.ServiceDB
	db           *database.DB
	normalizedDB *database.DB
	logger       *slog.Logger
}

// convertToIntSafe конвертирует значение в int, возвращая 0 при ошибке
func convertToIntSafe(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case uint64:
		return int(v)
	default:
		return 0
	}
}

// NewClientService создает новый сервис для работы с клиентами.
// Принимает подключения к базам данных (serviceDB обязателен, db и normalizedDB опциональны).
// Возвращает ошибку, если serviceDB равен nil.
func NewClientService(
	serviceDB *database.ServiceDB,
	db *database.DB,
	normalizedDB *database.DB,
) (*ClientService, error) {
	if serviceDB == nil {
		return nil, apperrors.NewInternalError("serviceDB не может быть nil", nil)
	}
	return &ClientService{
		serviceDB:    serviceDB,
		db:           db,
		normalizedDB: normalizedDB,
		logger:       slog.Default(),
	}, nil
}

// NewClientServiceWithLogger создает новый сервис с возможностью указать логгер (для тестирования).
func NewClientServiceWithLogger(
	serviceDB *database.ServiceDB,
	db *database.DB,
	normalizedDB *database.DB,
	logger *slog.Logger,
) (*ClientService, error) {
	if serviceDB == nil {
		return nil, apperrors.NewInternalError("serviceDB не может быть nil", nil)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ClientService{
		serviceDB:    serviceDB,
		db:           db,
		normalizedDB: normalizedDB,
		logger:       logger,
	}, nil
}

// GetAllClients возвращает список всех клиентов со статистикой.
//
// Использует batch-запрос GetClientsByIDs для оптимизации производительности
// (исправлена N+1 проблема).
//
// Возвращает список клиентов или ошибку при неудаче.
func (s *ClientService) GetAllClients(ctx context.Context) ([]*database.Client, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	s.logger.Info("Getting all clients")

	// GetClientsWithStats возвращает []map[string]interface{}, нужно преобразовать
	clientsWithStats, err := s.serviceDB.GetClientsWithStats()
	if err != nil {
		s.logger.Error("Failed to get clients with stats", "error", err)
		return nil, apperrors.NewInternalError("не удалось получить клиентов со статистикой", err)
	}

	s.logger.Info("Retrieved clients with stats", "count", len(clientsWithStats))

	statsByID := make(map[int]map[string]interface{}, len(clientsWithStats))
	// Собираем все ID клиентов для batch-запроса (исправление N+1 проблемы)
	clientIDs := make([]int, 0, len(clientsWithStats))
	for _, stat := range clientsWithStats {
		clientID, ok := stat["id"].(int)
		if ok && clientID > 0 {
			clientIDs = append(clientIDs, clientID)
			statsByID[clientID] = stat
		}
	}

	if len(clientIDs) == 0 {
		s.logger.Info("No valid client IDs found", "stats_count", len(clientsWithStats))
		return []*database.Client{}, nil
	}

	s.logger.Info("Getting clients by IDs", "count", len(clientIDs))

	// Получаем все клиенты одним batch-запросом (исправление N+1 проблемы)
	clients, err := s.serviceDB.GetClientsByIDs(clientIDs)
	if err != nil {
		s.logger.Error("Failed to get clients by IDs", "error", err, "count", len(clientIDs))
		return nil, apperrors.NewInternalError("не удалось получить клиентов по ID", err)
	}

	// Создаем map для быстрого поиска клиентов по ID
	clientMap := make(map[int]*database.Client, len(clients))
	for _, client := range clients {
		if stat, found := statsByID[client.ID]; found {
			if name, ok := stat["name"].(string); ok && client.Name == "" {
				client.Name = name
			}
			if legalName, ok := stat["legal_name"].(string); ok && client.LegalName == "" {
				client.LegalName = legalName
			}
			if description, ok := stat["description"].(string); ok && client.Description == "" {
				client.Description = description
			}
			if country, ok := stat["country"].(string); ok && client.Country == "" {
				client.Country = country
			}
			if status, ok := stat["status"].(string); ok && client.Status == "" {
				client.Status = status
			}
			client.ProjectCount = convertToIntSafe(stat["project_count"])
			client.BenchmarkCount = convertToIntSafe(stat["benchmark_count"])
			client.LastActivity = parseLastActivity(stat["last_activity"])
		}
		clientMap[client.ID] = client
	}

	// Сохраняем порядок из clientsWithStats
	orderedClients := make([]*database.Client, 0, len(clientsWithStats))
	for _, clientStatMap := range clientsWithStats {
		clientID, ok := clientStatMap["id"].(int)
		if ok && clientID > 0 {
			if client, found := clientMap[clientID]; found {
				orderedClients = append(orderedClients, client)
			}
		}
	}

	s.logger.Info("Successfully retrieved clients", "count", len(orderedClients))
	return orderedClients, nil
}

// GetClient возвращает клиента по ID.
// clientID - ID клиента для получения.
// Возвращает клиента или ошибку при неудаче (например, клиент не найден).
func (s *ClientService) GetClient(ctx context.Context, clientID int) (*database.Client, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	s.logger.Info("Getting client", "client_id", clientID)

	client, err := s.serviceDB.GetClient(clientID)
	if err != nil {
		s.logger.Error("Failed to get client", "client_id", clientID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("клиент не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить клиента", err)
	}

	return client, nil
}

// CreateClient создает нового клиента.
// name - обязательное поле, имя клиента.
// legalName, description, contactEmail, contactPhone, taxID, country - опциональные поля.
// Возвращает созданного клиента или ошибку при неудаче.
func (s *ClientService) CreateClient(ctx context.Context, name, legalName, description, contactEmail, contactPhone, taxID, country string) (*database.Client, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if name == "" {
		return nil, apperrors.NewValidationError("имя обязательно", nil)
	}

	s.logger.Info("Creating client", "name", name)

	client, err := s.serviceDB.CreateClient(name, legalName, description, contactEmail, contactPhone, taxID, country, "system")
	if err != nil {
		s.logger.Error("Failed to create client", "name", name, "error", err)
		
		// Проверяем, является ли это ошибкой уникальности имени
		if strings.Contains(err.Error(), "UNIQUE constraint failed") && strings.Contains(err.Error(), "clients.name") {
			return nil, apperrors.NewValidationError("клиент с таким именем уже существует", err)
		}
		
		return nil, apperrors.NewInternalError("не удалось создать клиента", err)
	}

	s.logger.Info("Successfully created client", "client_id", client.ID, "name", name)
	return client, nil
}

// UpdateClient обновляет клиента.
// clientID - ID клиента для обновления.
// name, legalName, description, contactEmail, contactPhone, taxID, country, status - поля для обновления.
// Возвращает обновленного клиента или ошибку при неудаче.
func (s *ClientService) UpdateClient(ctx context.Context, clientID int, name, legalName, description, contactEmail, contactPhone, taxID, country, status string) (*database.Client, error) {
	// Используем UpdateClientFields для обратной совместимости
	updates := &database.Client{
		Name:         name,
		LegalName:    legalName,
		Description:  description,
		ContactEmail: contactEmail,
		ContactPhone: contactPhone,
		TaxID:        taxID,
		Country:      country,
		Status:       status,
	}
	return s.UpdateClientFields(ctx, clientID, updates)
}

// UpdateClientFields обновляет поля клиента (поддерживает все новые поля)
// clientID - ID клиента для обновления.
// updates - структура Client с полями для обновления (обновляются только непустые поля).
// Возвращает обновленного клиента или ошибку при неудаче.
func (s *ClientService) UpdateClientFields(ctx context.Context, clientID int, updates *database.Client) (*database.Client, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if updates == nil {
		return nil, apperrors.NewValidationError("updates не может быть nil", nil)
	}

	s.logger.Info("Updating client fields", "client_id", clientID)

	if err := s.serviceDB.UpdateClientFields(clientID, updates); err != nil {
		s.logger.Error("Failed to update client fields", "client_id", clientID, "error", err)
		return nil, apperrors.NewInternalError("не удалось обновить клиента", err)
	}

	client, err := s.serviceDB.GetClient(clientID)
	if err != nil {
		s.logger.Error("Failed to get updated client", "client_id", clientID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("обновленный клиент не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить обновленного клиента", err)
	}

	s.logger.Info("Successfully updated client", "client_id", clientID)
	return client, nil
}

// DeleteClient удаляет клиента.
// clientID - ID клиента для удаления.
// Возвращает ошибку при неудаче.
func (s *ClientService) DeleteClient(ctx context.Context, clientID int) error {
	if ctx == nil {
		return apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	s.logger.Info("Deleting client", "client_id", clientID)

	if err := s.serviceDB.DeleteClient(clientID); err != nil {
		s.logger.Error("Failed to delete client", "client_id", clientID, "error", err)
		return apperrors.NewInternalError("не удалось удалить клиента", err)
	}

	s.logger.Info("Successfully deleted client", "client_id", clientID)
	return nil
}

func parseLastActivity(value interface{}) *time.Time {
	switch v := value.(type) {
	case nil:
		return nil
	case *time.Time:
		if v == nil || v.IsZero() {
			return nil
		}
		copy := v.UTC()
		return &copy
	case time.Time:
		if v.IsZero() {
			return nil
		}
		copy := v.UTC()
		return &copy
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			ts = ts.UTC()
			return &ts
		}
	case []byte:
		if len(v) == 0 {
			return nil
		}
		if ts, err := time.Parse(time.RFC3339, string(v)); err == nil {
			ts = ts.UTC()
			return &ts
		}
	}
	return nil
}

// GetClientProjects возвращает проекты клиента.
// clientID - ID клиента.
// Возвращает список проектов клиента или ошибку при неудаче.
func (s *ClientService) GetClientProjects(ctx context.Context, clientID int) ([]*database.ClientProject, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	s.logger.Info("Getting client projects", "client_id", clientID)

	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		s.logger.Error("Failed to get client projects", "client_id", clientID, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить проекты клиента", err)
	}

	s.logger.Info("Successfully retrieved client projects", "client_id", clientID, "count", len(projects))
	return projects, nil
}

// GetClientProject возвращает проект клиента.
// clientID - ID клиента.
// projectID - ID проекта.
// Возвращает проект или ошибку при неудаче.
func (s *ClientService) GetClientProject(ctx context.Context, clientID, projectID int) (*database.ClientProject, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return nil, apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	s.logger.Info("Getting client project", "client_id", clientID, "project_id", projectID)

	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.logger.Error("Failed to get client project", "client_id", clientID, "project_id", projectID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("проект клиента не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить проект клиента", err)
	}

	return project, nil
}

// CreateClientProject создает новый проект для клиента.
// clientID - ID клиента.
// name - обязательное поле, имя проекта.
// projectType, description, sourceSystem - опциональные поля.
// targetQualityScore - целевой балл качества.
// Возвращает созданный проект или ошибку при неудаче.
func (s *ClientService) CreateClientProject(ctx context.Context, clientID int, name, projectType, description, sourceSystem string, targetQualityScore float64) (*database.ClientProject, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if name == "" {
		return nil, apperrors.NewValidationError("имя обязательно", nil)
	}

	s.logger.Info("Creating client project", "client_id", clientID, "name", name)

	project, err := s.serviceDB.CreateClientProject(clientID, name, projectType, description, sourceSystem, targetQualityScore)
	if err != nil {
		s.logger.Error("Failed to create client project", "client_id", clientID, "name", name, "error", err)
		return nil, apperrors.NewInternalError("не удалось создать проект клиента", err)
	}

	s.logger.Info("Successfully created client project", "client_id", clientID, "project_id", project.ID, "name", name)
	return project, nil
}

// UpdateClientProject обновляет проект клиента.
// clientID - ID клиента.
// projectID - ID проекта.
// name, projectType, description, sourceSystem, status - поля для обновления.
// targetQualityScore - целевой балл качества.
// Возвращает обновленный проект или ошибку при неудаче.
func (s *ClientService) UpdateClientProject(ctx context.Context, clientID, projectID int, name, projectType, description, sourceSystem, status string, targetQualityScore float64) (*database.ClientProject, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return nil, apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	s.logger.Info("Updating client project", "client_id", clientID, "project_id", projectID)

	if err := s.serviceDB.UpdateClientProject(projectID, name, projectType, description, sourceSystem, status, targetQualityScore); err != nil {
		s.logger.Error("Failed to update client project", "client_id", clientID, "project_id", projectID, "error", err)
		return nil, apperrors.NewInternalError("не удалось обновить проект клиента", err)
	}

	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.logger.Error("Failed to get updated client project", "client_id", clientID, "project_id", projectID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("обновленный проект клиента не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить обновленный проект клиента", err)
	}

	s.logger.Info("Successfully updated client project", "client_id", clientID, "project_id", projectID)
	return project, nil
}

// DeleteClientProject удаляет проект клиента.
// clientID - ID клиента.
// projectID - ID проекта.
// Возвращает ошибку при неудаче.
func (s *ClientService) DeleteClientProject(ctx context.Context, clientID, projectID int) error {
	if ctx == nil {
		return apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	s.logger.Info("Deleting client project", "client_id", clientID, "project_id", projectID)

	if err := s.serviceDB.DeleteClientProject(projectID); err != nil {
		s.logger.Error("Failed to delete client project", "client_id", clientID, "project_id", projectID, "error", err)
		return apperrors.NewInternalError("не удалось удалить проект клиента", err)
	}

	s.logger.Info("Successfully deleted client project", "client_id", clientID, "project_id", projectID)
	return nil
}

// GetClientDatabases возвращает базы данных клиента.
// clientID - ID клиента.
//
// ПРИМЕЧАНИЕ: Есть потенциальная N+1 проблема - метод получает проекты клиента,
// затем для каждого делает отдельный запрос GetProjectDatabases.
// Для оптимизации рекомендуется добавить метод GetDatabasesByClientID в database.ServiceDB.
//
// Возвращает список баз данных клиента или ошибку при неудаче.
func (s *ClientService) GetClientDatabases(ctx context.Context, clientID int) ([]*database.ProjectDatabase, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	s.logger.Info("Getting client databases", "client_id", clientID)

	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		s.logger.Error("Failed to get client projects", "client_id", clientID, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить проекты клиента", err)
	}

	var allDatabases []*database.ProjectDatabase
	var errorCount int
	for _, project := range projects {
		databases, err := s.serviceDB.GetProjectDatabases(project.ID, false)
		if err != nil {
			errorCount++
			s.logger.Warn("Failed to get project databases", "client_id", clientID, "project_id", project.ID, "error", err)
			continue // Пропускаем проекты с ошибками
		}
		allDatabases = append(allDatabases, databases...)
	}

	if errorCount > 0 {
		s.logger.Warn("Some projects failed to get databases", "client_id", clientID, "errors_count", errorCount, "total_projects", len(projects))
	}

	s.logger.Info("Successfully retrieved client databases", "client_id", clientID, "count", len(allDatabases))
	return allDatabases, nil
}

// GetProjectDatabases возвращает базы данных проекта.
// clientID - ID клиента (используется для валидации).
// projectID - ID проекта.
// Возвращает список баз данных проекта или ошибку при неудаче.
func (s *ClientService) GetProjectDatabases(ctx context.Context, clientID, projectID int) ([]*database.ProjectDatabase, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return nil, apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	s.logger.Info("Getting project databases", "client_id", clientID, "project_id", projectID)

	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		s.logger.Error("Failed to get project databases", "client_id", clientID, "project_id", projectID, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить базы данных проекта", err)
	}

	s.logger.Info("Successfully retrieved project databases", "client_id", clientID, "project_id", projectID, "count", len(databases))
	return databases, nil
}

// GetServiceDB возвращает указатель на serviceDB для прямого доступа (используется в handlers)
func (s *ClientService) GetServiceDB() *database.ServiceDB {
	return s.serviceDB
}

// GetProjectDatabase возвращает базу данных проекта.
// clientID - ID клиента (используется для валидации).
// projectID - ID проекта (используется для валидации).
// dbID - ID базы данных.
// Возвращает базу данных или ошибку при неудаче.
func (s *ClientService) GetProjectDatabase(ctx context.Context, clientID, projectID, dbID int) (*database.ProjectDatabase, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return nil, apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	if dbID <= 0 {
		return nil, apperrors.NewValidationError("dbID должен быть положительным числом", nil)
	}

	s.logger.Info("Getting project database", "client_id", clientID, "project_id", projectID, "db_id", dbID)

	database, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		s.logger.Error("Failed to get project database", "client_id", clientID, "project_id", projectID, "db_id", dbID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("база данных проекта не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить базу данных проекта", err)
	}

	return database, nil
}

// CreateProjectDatabase создает новую базу данных для проекта.
// clientID - ID клиента (используется для валидации).
// projectID - ID проекта.
// name - обязательное поле, имя базы данных.
// dbPath, description - опциональные поля.
// fileSize - размер файла базы данных.
// Возвращает созданную базу данных или ошибку при неудаче.
func (s *ClientService) CreateProjectDatabase(ctx context.Context, clientID, projectID int, name, dbPath, description string, fileSize int64) (*database.ProjectDatabase, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return nil, apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	if name == "" {
		return nil, apperrors.NewValidationError("имя обязательно", nil)
	}

	s.logger.Info("Creating project database", "client_id", clientID, "project_id", projectID, "name", name)

	database, err := s.serviceDB.CreateProjectDatabase(projectID, name, dbPath, description, fileSize)
	if err != nil {
		s.logger.Error("Failed to create project database", "client_id", clientID, "project_id", projectID, "name", name, "error", err)
		return nil, apperrors.NewInternalError("не удалось создать базу данных проекта", err)
	}

	// Запускаем автоматический мэппинг контрагентов для новой базы данных
	// Выполняем в фоне, чтобы не блокировать создание базы данных
	go func() {
		mapper := normalization.NewCounterpartyMapper(s.serviceDB)
		if err := mapper.MapCounterpartiesFromDatabase(projectID, database.ID); err != nil {
			s.logger.Warn("Failed to auto-map counterparties for new database", "database_id", database.ID, "error", err)
		} else {
			s.logger.Info("Successfully auto-mapped counterparties for new database", "database_id", database.ID)
		}
	}()

	s.logger.Info("Successfully created project database", "client_id", clientID, "project_id", projectID, "db_id", database.ID, "name", name)
	return database, nil
}

// UpdateProjectDatabase обновляет базу данных проекта.
// clientID - ID клиента (используется для валидации).
// projectID - ID проекта (используется для валидации).
// dbID - ID базы данных.
// name, dbPath, description - поля для обновления.
// isActive - активна ли база данных.
// Возвращает обновленную базу данных или ошибку при неудаче.
func (s *ClientService) UpdateProjectDatabase(ctx context.Context, clientID, projectID, dbID int, name, dbPath, description string, isActive bool) (*database.ProjectDatabase, error) {
	if ctx == nil {
		return nil, apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return nil, apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return nil, apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return nil, apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	if dbID <= 0 {
		return nil, apperrors.NewValidationError("dbID должен быть положительным числом", nil)
	}

	s.logger.Info("Updating project database", "client_id", clientID, "project_id", projectID, "db_id", dbID)

	if err := s.serviceDB.UpdateProjectDatabase(dbID, name, dbPath, description, isActive); err != nil {
		s.logger.Error("Failed to update project database", "client_id", clientID, "project_id", projectID, "db_id", dbID, "error", err)
		return nil, apperrors.NewInternalError("не удалось обновить базу данных проекта", err)
	}

	database, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		s.logger.Error("Failed to get updated project database", "client_id", clientID, "project_id", projectID, "db_id", dbID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("обновленная база данных проекта не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить обновленную базу данных проекта", err)
	}

	s.logger.Info("Successfully updated project database", "client_id", clientID, "project_id", projectID, "db_id", dbID)
	return database, nil
}

// DeleteProjectDatabase удаляет базу данных проекта.
// clientID - ID клиента (используется для валидации).
// projectID - ID проекта (используется для валидации).
// dbID - ID базы данных.
// Возвращает ошибку при неудаче.
func (s *ClientService) DeleteProjectDatabase(ctx context.Context, clientID, projectID, dbID int) error {
	if ctx == nil {
		return apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
	}

	if s.serviceDB == nil {
		return apperrors.NewInternalError("сервисная база данных недоступна", nil)
	}

	if clientID <= 0 {
		return apperrors.NewValidationError("clientID должен быть положительным числом", nil)
	}

	if projectID <= 0 {
		return apperrors.NewValidationError("projectID должен быть положительным числом", nil)
	}

	if dbID <= 0 {
		return apperrors.NewValidationError("dbID должен быть положительным числом", nil)
	}

	s.logger.Info("Deleting project database", "client_id", clientID, "project_id", projectID, "db_id", dbID)

	if err := s.serviceDB.DeleteProjectDatabase(dbID); err != nil {
		s.logger.Error("Failed to delete project database", "client_id", clientID, "project_id", projectID, "db_id", dbID, "error", err)
		return apperrors.NewInternalError("не удалось удалить базу данных проекта", err)
	}

	s.logger.Info("Successfully deleted project database", "client_id", clientID, "project_id", projectID, "db_id", dbID)
	return nil
}
