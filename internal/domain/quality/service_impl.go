package quality

import (
	"context"
	"fmt"
	"time"

	"httpserver/internal/domain/repositories"
)

// service реализация domain service для quality
type service struct {
	qualityRepo     repositories.QualityRepository
	qualityAnalyzer QualityAnalyzerInterface
}

// NewService создает новый domain service для quality
func NewService(qualityRepo repositories.QualityRepository, qualityAnalyzer QualityAnalyzerInterface) Service {
	return &service{
		qualityRepo:     qualityRepo,
		qualityAnalyzer: qualityAnalyzer,
	}
}

// AnalyzeQuality запускает анализ качества для выгрузки
func (s *service) AnalyzeQuality(ctx context.Context, uploadID string) error {
	if uploadID == "" {
		return ErrInvalidUploadID
	}

	// Преобразуем uploadID в int для анализатора
	var uploadIDInt int
	if _, err := fmt.Sscanf(uploadID, "%d", &uploadIDInt); err != nil {
		return fmt.Errorf("invalid upload ID format: %w", err)
	}

	// Запускаем анализ через QualityAnalyzer
	if s.qualityAnalyzer != nil {
		if err := s.qualityAnalyzer.AnalyzeUpload(ctx, uploadIDInt); err != nil {
			return fmt.Errorf("failed to analyze quality: %w", err)
		}
	}

	// Создаем отчет через репозиторий
	now := time.Now()
	report := &repositories.QualityReport{
		ID:          fmt.Sprintf("report_%d", time.Now().Unix()),
		UploadID:    uploadID,
		AnalyzedAt:  &now,
		OverallScore: 0.0, // Будет обновлен после анализа
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.qualityRepo.Create(ctx, report); err != nil {
		return fmt.Errorf("failed to create quality report: %w", err)
	}

	return nil
}

// GetQualityReport возвращает отчет о качестве
func (s *service) GetQualityReport(ctx context.Context, uploadID string, summaryOnly bool, limit, offset int) (*QualityReport, error) {
	report, err := s.qualityRepo.GetByUploadID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality report: %w", err)
	}

	// TODO: Получить Issues и Metrics через репозиторий
	return s.toDomainReport(report), nil
}

// GetQualityScore возвращает оценку качества для сущности
func (s *service) GetQualityScore(ctx context.Context, entityID string) (float64, error) {
	metrics, err := s.qualityRepo.GetMetrics(ctx, entityID)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get metrics: %w", err)
	}

	if metrics == nil {
		return 0.0, nil
	}

	return metrics.OverallScore, nil
}

// GetQualityIssues возвращает проблемы качества с фильтрацией
func (s *service) GetQualityIssues(ctx context.Context, filter QualityIssueFilter) (*QualityIssueList, error) {
	repoFilter := s.toRepoFilter(filter)

	issues, total, err := s.qualityRepo.GetQualityIssues(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality issues: %w", err)
	}

	domainIssues := make([]*QualityIssue, 0, len(issues))
	for _, issue := range issues {
		domainIssues = append(domainIssues, s.toDomainIssue(&issue))
	}

	return &QualityIssueList{
		Issues: domainIssues,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

// GetQualityDashboard возвращает дашборд качества
func (s *service) GetQualityDashboard(ctx context.Context, databaseID int, days int, limit int) (*QualityDashboard, error) {
	if databaseID <= 0 {
		return nil, ErrInvalidDatabaseID
	}

	// Получаем тренды
	trends, err := s.qualityRepo.GetQualityTrends(ctx, fmt.Sprintf("%d", databaseID), time.Duration(days)*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality trends: %w", err)
	}

	domainTrends := make([]*QualityTrend, 0, len(trends))
	for _, trend := range trends {
		domainTrends = append(domainTrends, s.toDomainTrend(&trend))
	}

	// TODO: Получить TopIssues и MetricsByEntity
	dashboard := &QualityDashboard{
		DatabaseID:      databaseID,
		CurrentScore:    0.0, // TODO: Рассчитать текущий балл
		Trends:          domainTrends,
		TopIssues:       []*QualityIssue{},
		MetricsByEntity: make(map[string]*EntityMetrics),
	}

	return dashboard, nil
}

// GetQualityTrends возвращает тренды качества
func (s *service) GetQualityTrends(ctx context.Context, databaseID int, period time.Duration) ([]*QualityTrend, error) {
	trends, err := s.qualityRepo.GetQualityTrends(ctx, fmt.Sprintf("%d", databaseID), period)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality trends: %w", err)
	}

	result := make([]*QualityTrend, 0, len(trends))
	for _, trend := range trends {
		result = append(result, s.toDomainTrend(&trend))
	}

	return result, nil
}

// GetQualityMetrics возвращает метрики качества для сущности
func (s *service) GetQualityMetrics(ctx context.Context, entityID string) (*EntityMetrics, error) {
	metrics, err := s.qualityRepo.GetMetrics(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if metrics == nil {
		return &EntityMetrics{}, nil
	}

	return s.toDomainMetrics(metrics), nil
}

// SuggestImprovements предлагает улучшения качества
func (s *service) SuggestImprovements(ctx context.Context, entityID string) ([]*QualityImprovement, error) {
	// TODO: Реализовать предложения по улучшению
	return []*QualityImprovement{}, nil
}

// GetQualityRecommendations возвращает рекомендации по улучшению качества
func (s *service) GetQualityRecommendations(ctx context.Context, databaseID int) ([]*QualityRecommendation, error) {
	// TODO: Реализовать рекомендации
	return []*QualityRecommendation{}, nil
}

// GetQualityStatistics возвращает статистику качества
func (s *service) GetQualityStatistics(ctx context.Context, databaseID int) (*QualityStatistics, error) {
	// TODO: Реализовать через репозиторий
	return &QualityStatistics{}, nil
}

// GetQualityDistribution возвращает распределение качества
func (s *service) GetQualityDistribution(ctx context.Context) (*QualityDistribution, error) {
	// TODO: Реализовать распределение качества
	return &QualityDistribution{}, nil
}

// Вспомогательные методы преобразования

func (s *service) toDomainReport(r *repositories.QualityReport) *QualityReport {
	return &QualityReport{
		ID:           r.ID,
		UploadID:     r.UploadID,
		DatabaseID:   r.DatabaseID,
		AnalyzedAt:   r.AnalyzedAt,
		OverallScore: r.OverallScore,
		Metrics:      []*QualityMetric{}, // TODO: Преобразовать
		Issues:       []*QualityIssue{},  // TODO: Преобразовать
		Summary:      &QualitySummary{},  // TODO: Преобразовать
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (s *service) toDomainIssue(i *repositories.QualityIssue) *QualityIssue {
	return &QualityIssue{
		ID:          i.ID,
		EntityID:    i.EntityID,
		EntityType:  i.EntityType,
		Field:       i.Field,
		Severity:    i.Severity,
		Description: i.Description,
		Suggestion:  i.Suggestion,
		Rule:        i.Rule,
		CreatedAt:   i.CreatedAt,
		ResolvedAt:  i.ResolvedAt,
	}
}

func (s *service) toDomainTrend(t *repositories.QualityTrend) *QualityTrend {
	metrics := make([]*QualityMetric, 0, len(t.Metrics))
	for _, m := range t.Metrics {
		metrics = append(metrics, &QualityMetric{
			Name:        m.Name,
			Value:       m.Value,
			Target:      m.Target,
			Status:      m.Status,
			Description: m.Description,
		})
	}

	return &QualityTrend{
		Date:        t.Date,
		Score:       t.Score,
		Metrics:     metrics,
		IssuesCount: t.IssuesCount,
	}
}

func (s *service) toDomainMetrics(m *repositories.EntityMetrics) *EntityMetrics {
	return &EntityMetrics{
		Completeness: m.Completeness,
		Consistency:  m.Consistency,
		Uniqueness:   m.Uniqueness,
		Validity:     m.Validity,
		OverallScore: m.OverallScore,
	}
}

func (s *service) toRepoFilter(f QualityIssueFilter) repositories.QualityIssueFilter {
	return repositories.QualityIssueFilter{
		Severity:   f.Severity,
		EntityType: f.EntityType,
		EntityID:   f.EntityID,
		Field:      f.Field,
		Resolved:   f.Resolved,
		DateFrom:   f.DateFrom,
		DateTo:     f.DateTo,
		Limit:      f.Limit,
		Offset:     f.Offset,
	}
}

