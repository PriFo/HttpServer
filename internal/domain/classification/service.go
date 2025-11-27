package classification

import (
	"context"
	"time"
)

// Service интерфейс бизнес-логики для работы с классификацией
// Определяет операции на уровне предметной области
type Service interface {
	// Классификация сущностей
	ClassifyEntity(ctx context.Context, entityID string, entityType string, category string) (*Classification, error)
	BatchClassify(ctx context.Context, entityIDs []string, entityType string, category string) (*BatchClassificationResult, error)
	
	// Получение результатов классификации
	GetClassification(ctx context.Context, classificationID string) (*Classification, error)
	GetClassificationByEntity(ctx context.Context, entityID string) (*Classification, error)
	GetClassificationHistory(ctx context.Context, entityID string) ([]*Classification, error)
	
	// Обновление классификации
	UpdateClassification(ctx context.Context, classificationID string, category string, subcategory string, confidence float64) (*Classification, error)
	DeleteClassification(ctx context.Context, classificationID string) error
	
	// Статистика и аналитика
	GetClassificationStatistics(ctx context.Context) (*ClassificationStatistics, error)
	GetClassificationAccuracy(ctx context.Context, category string) (float64, error)
	GetClassificationsByCategory(ctx context.Context, category string, limit, offset int) ([]*Classification, int64, error)
	
	// Иерархическая классификация
	ClassifyHierarchical(ctx context.Context, entityID string, entityType string, category string) (*Classification, error)
}

// Classification представляет результат классификации
type Classification struct {
	ID          string
	EntityID    string
	EntityType  string
	Category    string
	Subcategory string
	Confidence  float64
	Rule        string
	Source      string // "manual", "ai", "rule", "hierarchical"
	ProcessedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BatchClassificationResult результат пакетной классификации
type BatchClassificationResult struct {
	Total        int
	Successful   int
	Failed       int
	Results      []*Classification
	Errors       []ClassificationError
	ProcessingTime time.Duration
}

// ClassificationError ошибка классификации
type ClassificationError struct {
	EntityID string
	Error    string
	Severity string // "low", "medium", "high", "critical"
}

// ClassificationStatistics статистика классификации
type ClassificationStatistics struct {
	TotalClassifications      int64
	SuccessfulClassifications int64
	FailedClassifications     int64
	AverageConfidence         float64
	AccuracyByCategory        map[string]float64
}

