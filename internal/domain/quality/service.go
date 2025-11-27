package quality

import (
	"context"
	"time"
)

// Service интерфейс бизнес-логики для работы с качеством данных
// Определяет операции на уровне предметной области
type Service interface {
	// Анализ качества
	AnalyzeQuality(ctx context.Context, uploadID string) error
	GetQualityReport(ctx context.Context, uploadID string, summaryOnly bool, limit, offset int) (*QualityReport, error)
	GetQualityScore(ctx context.Context, entityID string) (float64, error)
	GetQualityIssues(ctx context.Context, filter QualityIssueFilter) (*QualityIssueList, error)

	// Dashboard и аналитика
	GetQualityDashboard(ctx context.Context, databaseID int, days int, limit int) (*QualityDashboard, error)
	GetQualityTrends(ctx context.Context, databaseID int, period time.Duration) ([]*QualityTrend, error)
	GetQualityMetrics(ctx context.Context, entityID string) (*EntityMetrics, error)

	// Улучшение качества
	SuggestImprovements(ctx context.Context, entityID string) ([]*QualityImprovement, error)
	GetQualityRecommendations(ctx context.Context, databaseID int) ([]*QualityRecommendation, error)

	// Статистика
	GetQualityStatistics(ctx context.Context, databaseID int) (*QualityStatistics, error)
	GetQualityDistribution(ctx context.Context) (*QualityDistribution, error)
}

// QualityReport отчет о качестве данных
type QualityReport struct {
	ID           string
	UploadID     string
	DatabaseID   int
	AnalyzedAt   *time.Time
	OverallScore float64
	Metrics      []*QualityMetric
	Issues       []*QualityIssue
	Summary      *QualitySummary
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// QualityMetric метрика качества
type QualityMetric struct {
	Name        string
	Value       float64
	Target      float64
	Status      string // "good", "warning", "critical"
	Description string
}

// QualityIssue проблема качества данных
type QualityIssue struct {
	ID          string
	EntityID    string
	EntityType  string
	Field       string
	Severity    string // "low", "medium", "high", "critical"
	Description string
	Suggestion  string
	Rule        string
	CreatedAt   time.Time
	ResolvedAt  *time.Time
}

// QualitySummary сводка по качеству
type QualitySummary struct {
	TotalIssues       int
	CriticalIssues    int
	HighIssues        int
	MediumIssues      int
	LowIssues         int
	MetricsByCategory map[string]float64
}

// QualityIssueFilter фильтр для поиска проблем качества
type QualityIssueFilter struct {
	Severity   []string
	EntityType string
	EntityID   string
	Field      string
	Resolved   *bool
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int
	Offset     int
}

// QualityIssueList список проблем качества с пагинацией
type QualityIssueList struct {
	Issues []*QualityIssue
	Total  int64
	Limit  int
	Offset int
}

// QualityTrend тренд качества
type QualityTrend struct {
	Date        time.Time
	Score       float64
	Metrics     []*QualityMetric
	IssuesCount int
}

// EntityMetrics метрики по сущности
type EntityMetrics struct {
	Completeness float64
	Consistency  float64
	Uniqueness   float64
	Validity     float64
	OverallScore float64
}

// QualityDashboard дашборд качества данных
type QualityDashboard struct {
	DatabaseID      int
	CurrentScore    float64
	Trends          []*QualityTrend
	TopIssues       []*QualityIssue
	MetricsByEntity map[string]*EntityMetrics
}

// QualityImprovement предложение по улучшению качества
type QualityImprovement struct {
	EntityID    string
	EntityType  string
	Field       string
	Issue       string
	Suggestion  string
	Priority    string // "low", "medium", "high"
	ImpactScore float64
}

// QualityRecommendation рекомендация по улучшению качества
type QualityRecommendation struct {
	ID          string
	DatabaseID  int
	Category    string
	Description string
	Priority    string
	Action      string
	ImpactScore float64
}

// QualityStatistics статистика качества
type QualityStatistics struct {
	TotalReports       int64
	AverageScore       float64
	TotalIssues        int64
	IssuesBySeverity   map[string]int64
	MetricsByCategory  map[string]float64
}

// QualityDistribution распределение качества
type QualityDistribution struct {
	Excellent int64 // >= 90
	Good      int64 // 70-89
	Fair      int64 // 50-69
	Poor      int64 // < 50
}

