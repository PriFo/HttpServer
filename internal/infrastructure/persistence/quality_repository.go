package persistence

import (
	"context"
	"fmt"
	"time"

	"httpserver/database"
	"httpserver/internal/domain/repositories"
)

// qualityRepository реализация репозитория для качества данных
// Адаптер между domain интерфейсом и infrastructure (database.DB)
type qualityRepository struct {
	db *database.DB
}

// NewQualityRepository создает новый репозиторий качества
func NewQualityRepository(db *database.DB) repositories.QualityRepository {
	return &qualityRepository{
		db: db,
	}
}

// Create создает новый отчет о качестве
func (r *qualityRepository) Create(ctx context.Context, quality *repositories.QualityReport) error {
	// TODO: Реализовать создание отчета о качестве
	return fmt.Errorf("not implemented yet")
}

// GetByID возвращает отчет о качестве по ID
func (r *qualityRepository) GetByID(ctx context.Context, id string) (*repositories.QualityReport, error) {
	// TODO: Реализовать получение отчета по ID
	return nil, fmt.Errorf("not implemented yet")
}

// GetByUploadID возвращает отчет о качестве по UploadID
func (r *qualityRepository) GetByUploadID(ctx context.Context, uploadID string) (*repositories.QualityReport, error) {
	// TODO: Интегрировать с существующим GetQualityReport
	// Пока возвращаем заглушку
	return &repositories.QualityReport{
		ID:          fmt.Sprintf("report_%s", uploadID),
		UploadID:    uploadID,
		OverallScore: 0.0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// Update обновляет отчет о качестве
func (r *qualityRepository) Update(ctx context.Context, quality *repositories.QualityReport) error {
	// TODO: Реализовать обновление отчета
	return fmt.Errorf("not implemented yet")
}

// Delete удаляет отчет о качестве
func (r *qualityRepository) Delete(ctx context.Context, id string) error {
	// TODO: Реализовать удаление отчета
	return fmt.Errorf("not implemented yet")
}

// AnalyzeUpload запускает анализ качества для выгрузки
func (r *qualityRepository) AnalyzeUpload(ctx context.Context, uploadID string) (*repositories.QualityReport, error) {
	// TODO: Интегрировать с QualityAnalyzer
	return nil, fmt.Errorf("not implemented yet")
}

// GetQualityTrends возвращает тренды качества
func (r *qualityRepository) GetQualityTrends(ctx context.Context, databaseID string, period time.Duration) ([]repositories.QualityTrend, error) {
	// Преобразуем databaseID в int
	var dbID int
	_, err := fmt.Sscanf(databaseID, "%d", &dbID)
	if err != nil {
		return nil, fmt.Errorf("invalid database ID: %w", err)
	}

	// Получаем количество дней из периода
	days := int(period.Hours() / 24)
	if days <= 0 {
		days = 7 // По умолчанию 7 дней
	}

	// Используем существующий метод database.DB
	trends, err := r.db.GetQualityTrends(dbID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality trends: %w", err)
	}

	// Преобразуем в domain модели
	result := make([]repositories.QualityTrend, 0, len(trends))
	for _, trend := range trends {
		result = append(result, r.toDomainTrend(&trend))
	}

	return result, nil
}

// GetQualityIssues возвращает проблемы качества с фильтрацией
func (r *qualityRepository) GetQualityIssues(ctx context.Context, filter repositories.QualityIssueFilter) ([]repositories.QualityIssue, int64, error) {
	// TODO: Реализовать получение проблем с фильтрацией
	return []repositories.QualityIssue{}, 0, nil
}

// GetMetrics возвращает метрики качества для сущности
func (r *qualityRepository) GetMetrics(ctx context.Context, entityID string) (*repositories.EntityMetrics, error) {
	// TODO: Реализовать получение метрик
	return &repositories.EntityMetrics{}, nil
}

// UpdateMetrics обновляет метрики качества для сущности
func (r *qualityRepository) UpdateMetrics(ctx context.Context, entityID string, metrics *repositories.EntityMetrics) error {
	// TODO: Реализовать обновление метрик
	return fmt.Errorf("not implemented yet")
}

// GetOverallQualityScore возвращает общую оценку качества для базы данных
func (r *qualityRepository) GetOverallQualityScore(ctx context.Context, databaseID string) (float64, error) {
	// TODO: Реализовать получение общей оценки качества
	return 0.0, nil
}

// Вспомогательные методы

// toDomainTrend преобразует database.QualityTrend в repositories.QualityTrend
func (r *qualityRepository) toDomainTrend(t *database.QualityTrend) repositories.QualityTrend {
	// database.QualityTrend имеет поля: MeasurementDate, OverallScore, IssuesCount
	// Metrics нужно получить отдельно или создать из полей Score
	metrics := []repositories.QualityMetric{}

	// Создаем метрики из полей QualityTrend
	if t.CompletenessScore != nil {
		metrics = append(metrics, repositories.QualityMetric{
			Name:   "completeness",
			Value:  *t.CompletenessScore,
			Target: 90.0,
			Status: getMetricStatus(*t.CompletenessScore, 90.0),
		})
	}
	if t.ConsistencyScore != nil {
		metrics = append(metrics, repositories.QualityMetric{
			Name:   "consistency",
			Value:  *t.ConsistencyScore,
			Target: 90.0,
			Status: getMetricStatus(*t.ConsistencyScore, 90.0),
		})
	}
	if t.UniquenessScore != nil {
		metrics = append(metrics, repositories.QualityMetric{
			Name:   "uniqueness",
			Value:  *t.UniquenessScore,
			Target: 90.0,
			Status: getMetricStatus(*t.UniquenessScore, 90.0),
		})
	}
	if t.ValidityScore != nil {
		metrics = append(metrics, repositories.QualityMetric{
			Name:   "validity",
			Value:  *t.ValidityScore,
			Target: 90.0,
			Status: getMetricStatus(*t.ValidityScore, 90.0),
		})
	}

	return repositories.QualityTrend{
		Date:        t.MeasurementDate,
		Score:       t.OverallScore,
		Metrics:     metrics,
		IssuesCount: t.IssuesCount,
	}
}

// getMetricStatus определяет статус метрики на основе значения и целевого значения
func getMetricStatus(value, target float64) string {
	if value >= target {
		return "good"
	} else if value >= target*0.7 {
		return "warning"
	}
	return "critical"
}

