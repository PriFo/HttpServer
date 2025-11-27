package services

import (
	"context"

	"httpserver/database"
)

// ClientServiceInterface интерфейс для сервиса работы с клиентами, проектами и базами данных.
// Используется для улучшения тестируемости и возможности замены реализации.
type ClientServiceInterface interface {
	// GetAllClients возвращает список всех клиентов со статистикой.
	GetAllClients(ctx context.Context) ([]*database.Client, error)

	// GetClient возвращает клиента по ID.
	GetClient(ctx context.Context, clientID int) (*database.Client, error)

	// CreateClient создает нового клиента.
	CreateClient(ctx context.Context, name, legalName, description, contactEmail, contactPhone, taxID string) (*database.Client, error)

	// UpdateClient обновляет клиента.
	UpdateClient(ctx context.Context, clientID int, name, legalName, description, contactEmail, contactPhone, taxID, status string) (*database.Client, error)

	// DeleteClient удаляет клиента.
	DeleteClient(ctx context.Context, clientID int) error

	// GetClientProjects возвращает проекты клиента.
	GetClientProjects(ctx context.Context, clientID int) ([]*database.ClientProject, error)

	// GetClientProject возвращает проект клиента.
	GetClientProject(ctx context.Context, clientID, projectID int) (*database.ClientProject, error)

	// CreateClientProject создает новый проект для клиента.
	CreateClientProject(ctx context.Context, clientID int, name, projectType, description, sourceSystem string, targetQualityScore float64) (*database.ClientProject, error)

	// UpdateClientProject обновляет проект клиента.
	UpdateClientProject(ctx context.Context, clientID, projectID int, name, projectType, description, sourceSystem, status string, targetQualityScore float64) (*database.ClientProject, error)

	// DeleteClientProject удаляет проект клиента.
	DeleteClientProject(ctx context.Context, clientID, projectID int) error

	// GetClientDatabases возвращает базы данных клиента.
	GetClientDatabases(ctx context.Context, clientID int) ([]*database.ProjectDatabase, error)

	// GetProjectDatabases возвращает базы данных проекта.
	GetProjectDatabases(ctx context.Context, clientID, projectID int) ([]*database.ProjectDatabase, error)

	// GetProjectDatabase возвращает базу данных проекта.
	GetProjectDatabase(ctx context.Context, clientID, projectID, dbID int) (*database.ProjectDatabase, error)

	// CreateProjectDatabase создает новую базу данных для проекта.
	CreateProjectDatabase(ctx context.Context, clientID, projectID int, name, dbPath, description string, fileSize int64) (*database.ProjectDatabase, error)

	// UpdateProjectDatabase обновляет базу данных проекта.
	UpdateProjectDatabase(ctx context.Context, clientID, projectID, dbID int, name, dbPath, description string, isActive bool) (*database.ProjectDatabase, error)

	// DeleteProjectDatabase удаляет базу данных проекта.
	DeleteProjectDatabase(ctx context.Context, clientID, projectID, dbID int) error
}

