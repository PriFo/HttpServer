package database

import (
	"context"

	"httpserver/internal/domain/repositories"
)

// Service интерфейс бизнес-логики для работы с базами данных
// Определяет операции на уровне предметной области
type Service interface {
	// Основные операции
	CreateDatabase(ctx context.Context, req CreateDatabaseRequest) (*Database, error)
	GetDatabase(ctx context.Context, databaseID string) (*Database, error)
	UpdateDatabase(ctx context.Context, databaseID string, req UpdateDatabaseRequest) (*Database, error)
	DeleteDatabase(ctx context.Context, databaseID string) error
	ListDatabases(ctx context.Context, filter repositories.DatabaseFilter) ([]*Database, int64, error)

	// Связанные операции
	GetDatabasesByProject(ctx context.Context, projectID string) ([]*Database, error)
	GetDatabasesByClient(ctx context.Context, clientID string) ([]*Database, error)
	TestConnection(ctx context.Context, databaseID string) error
	GetConnectionStatus(ctx context.Context, databaseID string) (string, error)
	GetDatabaseStatistics(ctx context.Context, databaseID string) (*DatabaseStatistics, error)
}

// Database представляет базу данных в доменной модели
type Database struct {
	ID              string `json:"id"`
	ClientID        string `json:"client_id,omitempty"`
	ProjectID       string `json:"project_id,omitempty"`
	Name            string `json:"name"`
	FilePath        string `json:"file_path"`
	Description     string `json:"description,omitempty"`
	IsActive        bool   `json:"is_active"`
	FileSize        int64  `json:"file_size,omitempty"`
	LastUsedAt      string `json:"last_used_at,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// CreateDatabaseRequest запрос на создание базы данных
type CreateDatabaseRequest struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	FilePath    string `json:"file_path"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active,omitempty"`
}

// UpdateDatabaseRequest запрос на обновление базы данных
type UpdateDatabaseRequest struct {
	Name        string `json:"name,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Description string `json:"description,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// DatabaseStatistics статистика базы данных
type DatabaseStatistics struct {
	TotalUploads     int64   `json:"total_uploads"`
	TotalRecords     int64   `json:"total_records"`
	LastUploadAt     string  `json:"last_upload_at,omitempty"`
	AverageQuality   float64 `json:"average_quality,omitempty"`
	ConnectionStatus string  `json:"connection_status"`
}

