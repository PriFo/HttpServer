package project

import (
	"context"

	"httpserver/internal/domain/repositories"
)

// Service интерфейс бизнес-логики для работы с проектами
// Определяет операции на уровне предметной области
type Service interface {
	// Основные операции
	CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error)
	GetProject(ctx context.Context, projectID string) (*Project, error)
	UpdateProject(ctx context.Context, projectID string, req UpdateProjectRequest) (*Project, error)
	DeleteProject(ctx context.Context, projectID string) error
	ListProjects(ctx context.Context, filter repositories.ProjectFilter) ([]*Project, int64, error)

	// Связанные операции
	GetProjectDatabases(ctx context.Context, projectID string) ([]*repositories.Database, error)
	GetProjectStatistics(ctx context.Context, projectID string) (*ProjectStatistics, error)
}

// Project представляет проект в доменной модели
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

// CreateProjectRequest запрос на создание проекта
type CreateProjectRequest struct {
	ClientID    string `json:"client_id"`
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

// ProjectStatistics статистика проекта
type ProjectStatistics struct {
	TotalDatabases   int64   `json:"total_databases"`
	TotalUploads     int64   `json:"total_uploads"`
	ActiveDatabases  int64   `json:"active_databases"`
	LastUploadAt     string  `json:"last_upload_at,omitempty"`
	AverageQuality   float64 `json:"average_quality,omitempty"`
}

