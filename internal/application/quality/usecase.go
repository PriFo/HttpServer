package quality

import (
	"context"
	"fmt"
	"time"

	qualitydomain "httpserver/internal/domain/quality"
	"httpserver/internal/domain/repositories"
)

// UseCase представляет use case для работы с качеством данных
// Координирует выполнение бизнес-логики между domain и infrastructure слоями
type UseCase struct {
	qualityRepo    repositories.QualityRepository
	qualityService qualitydomain.Service
}

// NewUseCase создает новый use case для качества
func NewUseCase(
	qualityRepo repositories.QualityRepository,
	qualityService qualitydomain.Service,
) *UseCase {
	return &UseCase{
		qualityRepo:    qualityRepo,
		qualityService: qualityService,
	}
}

// AnalyzeQuality запускает анализ качества для выгрузки
func (uc *UseCase) AnalyzeQuality(ctx context.Context, uploadID string) error {
	if err := uc.qualityService.AnalyzeQuality(ctx, uploadID); err != nil {
		return fmt.Errorf("failed to analyze quality: %w", err)
	}
	return nil
}

// GetQualityReport возвращает отчет о качестве
func (uc *UseCase) GetQualityReport(ctx context.Context, uploadID string, summaryOnly bool, limit, offset int) (*qualitydomain.QualityReport, error) {
	result, err := uc.qualityService.GetQualityReport(ctx, uploadID, summaryOnly, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality report: %w", err)
	}
	return result, nil
}

// GetQualityScore возвращает оценку качества для сущности
func (uc *UseCase) GetQualityScore(ctx context.Context, entityID string) (float64, error) {
	result, err := uc.qualityService.GetQualityScore(ctx, entityID)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get quality score: %w", err)
	}
	return result, nil
}

// GetQualityIssues возвращает проблемы качества с фильтрацией
func (uc *UseCase) GetQualityIssues(ctx context.Context, filter qualitydomain.QualityIssueFilter) (*qualitydomain.QualityIssueList, error) {
	result, err := uc.qualityService.GetQualityIssues(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality issues: %w", err)
	}
	return result, nil
}

// GetQualityDashboard возвращает дашборд качества
func (uc *UseCase) GetQualityDashboard(ctx context.Context, databaseID int, days int, limit int) (*qualitydomain.QualityDashboard, error) {
	result, err := uc.qualityService.GetQualityDashboard(ctx, databaseID, days, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality dashboard: %w", err)
	}
	return result, nil
}

// GetQualityTrends возвращает тренды качества
func (uc *UseCase) GetQualityTrends(ctx context.Context, databaseID int, period time.Duration) ([]*qualitydomain.QualityTrend, error) {
	result, err := uc.qualityService.GetQualityTrends(ctx, databaseID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality trends: %w", err)
	}
	return result, nil
}

// GetQualityMetrics возвращает метрики качества для сущности
func (uc *UseCase) GetQualityMetrics(ctx context.Context, entityID string) (*qualitydomain.EntityMetrics, error) {
	result, err := uc.qualityService.GetQualityMetrics(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality metrics: %w", err)
	}
	return result, nil
}

// GetQualityStatistics возвращает статистику качества
func (uc *UseCase) GetQualityStatistics(ctx context.Context, databaseID int) (*qualitydomain.QualityStatistics, error) {
	result, err := uc.qualityService.GetQualityStatistics(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality statistics: %w", err)
	}
	return result, nil
}

