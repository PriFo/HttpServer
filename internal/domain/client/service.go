package client

import (
	"context"

	"httpserver/internal/domain/repositories"
)

// Service интерфейс бизнес-логики для работы с клиентами и проектами
// Определяет операции на уровне предметной области
type Service interface {
	// Клиенты
	CreateClient(ctx context.Context, req CreateClientRequest) (*Client, error)
	GetClient(ctx context.Context, clientID string) (*Client, error)
	UpdateClient(ctx context.Context, clientID string, req UpdateClientRequest) (*Client, error)
	DeleteClient(ctx context.Context, clientID string) error
	ListClients(ctx context.Context, filter repositories.ClientFilter) ([]*Client, int64, error)
	GetClientStatistics(ctx context.Context, clientID string) (*ClientStatistics, error)

	// Проекты
	CreateProject(ctx context.Context, clientID string, req CreateProjectRequest) (*Project, error)
	GetProject(ctx context.Context, clientID string, projectID string) (*Project, error)
	UpdateProject(ctx context.Context, clientID string, projectID string, req UpdateProjectRequest) (*Project, error)
	DeleteProject(ctx context.Context, clientID string, projectID string) error
	ListProjects(ctx context.Context, clientID string, filter repositories.ProjectFilter) ([]*Project, int64, error)
	GetProjectStatistics(ctx context.Context, clientID string, projectID string) (*ProjectStatistics, error)

	// Базы данных проектов
	GetProjectDatabases(ctx context.Context, clientID string, projectID string) ([]*Database, error)
	GetClientDatabases(ctx context.Context, clientID string) ([]*Database, error)
}

// Client представляет клиента в доменной модели
type Client struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	LegalName    string `json:"legal_name,omitempty"`
	Description  string `json:"description,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	TaxID        string `json:"tax_id,omitempty"`
	Country      string `json:"country,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Project представляет проект клиента в доменной модели
type Project struct {
	ID          string `json:"id"`
	ClientID    string `json:"client_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Database представляет базу данных проекта в доменной модели
type Database struct {
	ID              string `json:"id"`
	ClientID        string `json:"client_id"`
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	FilePath        string `json:"file_path"`
	Description     string `json:"description,omitempty"`
	IsActive        bool   `json:"is_active"`
	FileSize        int64  `json:"file_size,omitempty"`
	LastUsedAt      string `json:"last_used_at,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// CreateClientRequest запрос на создание клиента
type CreateClientRequest struct {
	Name         string `json:"name"`
	LegalName    string `json:"legal_name,omitempty"`
	Description  string `json:"description,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	TaxID        string `json:"tax_id,omitempty"`
	Country      string `json:"country,omitempty"`
	Status       string `json:"status,omitempty"`
}

// UpdateClientRequest запрос на обновление клиента
type UpdateClientRequest struct {
	Name         string `json:"name,omitempty"`
	LegalName    string `json:"legal_name,omitempty"`
	Description  string `json:"description,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
	TaxID        string `json:"tax_id,omitempty"`
	Country      string `json:"country,omitempty"`
	Status       string `json:"status,omitempty"`
}

// CreateProjectRequest запрос на создание проекта
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
}

// UpdateProjectRequest запрос на обновление проекта
type UpdateProjectRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Status      string `json:"status,omitempty"`
}

// ClientStatistics статистика клиента
type ClientStatistics struct {
	TotalProjects    int64   `json:"total_projects"`
	TotalDatabases   int64   `json:"total_databases"`
	TotalUploads     int64   `json:"total_uploads"`
	ActiveProjects   int64   `json:"active_projects"`
	ActiveDatabases  int64   `json:"active_databases"`
	LastUploadAt     string  `json:"last_upload_at,omitempty"`
	AverageQuality   float64 `json:"average_quality,omitempty"`
}

// ProjectStatistics статистика проекта
type ProjectStatistics struct {
	TotalDatabases   int64   `json:"total_databases"`
	TotalUploads       int64   `json:"total_uploads"`
	ActiveDatabases   int64   `json:"active_databases"`
	LastUploadAt      string  `json:"last_upload_at,omitempty"`
	AverageQuality    float64 `json:"average_quality,omitempty"`
}

