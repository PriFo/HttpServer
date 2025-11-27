package normalization

import (
	"context"
	"time"
)

// Service интерфейс бизнес-логики для работы с нормализацией
// Определяет операции на уровне предметной области
type Service interface {
	// Управление процессами нормализации
	StartProcess(ctx context.Context, uploadID string) (*NormalizationProcess, error)
	GetProcessStatus(ctx context.Context, processID string) (*NormalizationProcess, error)
	StopProcess(ctx context.Context, processID string) error
	GetActiveProcesses(ctx context.Context) ([]*NormalizationProcess, error)

	// Нормализация данных
	NormalizeName(ctx context.Context, name string, entityType string) (string, error)
	NormalizeEntity(ctx context.Context, entityID string, entityType string) (*NormalizedEntity, error)
	BatchNormalize(ctx context.Context, entityIDs []string, entityType string) (*BatchNormalizationResult, error)

	// Версионированная нормализация
	StartVersionedNormalization(ctx context.Context, itemID int, originalName string) (*NormalizationSession, error)
	ApplyPatterns(ctx context.Context, sessionID int) (*NormalizationSession, error)
	ApplyAI(ctx context.Context, sessionID int, useChat bool) (*NormalizationSession, error)
	GetSessionHistory(ctx context.Context, sessionID int) ([]*NormalizationStage, error)

	// Статистика и аналитика
	GetStatistics(ctx context.Context) (*NormalizationStatistics, error)
	GetProcessHistory(ctx context.Context, uploadID string) ([]*NormalizationProcess, error)
}

// NormalizationProcess представляет процесс нормализации
type NormalizationProcess struct {
	ID          string
	UploadID    string
	Status      string // "pending", "running", "completed", "failed", "cancelled"
	Progress    float64
	Processed   int
	Total       int
	StartedAt   time.Time
	CompletedAt *time.Time
	Error       string
	Config      string // JSON
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NormalizedEntity результат нормализации сущности
type NormalizedEntity struct {
	EntityID      string
	EntityType    string
	OriginalName  string
	NormalizedName string
	Confidence    float64
	Method        string // "benchmark", "ai", "pattern"
	QualityScore  float64
}

// BatchNormalizationResult результат пакетной нормализации
type BatchNormalizationResult struct {
	Total        int
	Successful   int
	Failed       int
	Results      []*NormalizedEntity
	Errors       []NormalizationError
	ProcessingTime time.Duration
}

// NormalizationError ошибка нормализации
type NormalizationError struct {
	RecordID string
	Field    string
	Error    string
	Severity string // "low", "medium", "high", "critical"
}

// NormalizationSession сессия версионированной нормализации
type NormalizationSession struct {
	SessionID    int
	ItemID       int
	OriginalName string
	CurrentName  string
	StageCount   int
}

// NormalizationStage этап нормализации
type NormalizationStage struct {
	StageID     int
	SessionID   int
	Name        string
	Method      string
	Timestamp   time.Time
}

// NormalizationStatistics статистика нормализации
type NormalizationStatistics struct {
	TotalProcesses      int64
	SuccessfulProcesses int64
	FailedProcesses     int64
	AverageDuration     time.Duration
	TotalProcessed      int64
	AverageProgress     float64
}

