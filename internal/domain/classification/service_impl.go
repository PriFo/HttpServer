package classification

import (
	"context"
	"fmt"
	"time"

	"httpserver/internal/domain/repositories"
)

// service реализация domain service для classification
type service struct {
	classificationRepo repositories.ClassificationRepository
	// TODO: Добавить зависимости для AI классификатора и иерархического классификатора
}

// NewService создает новый domain service для classification
func NewService(classificationRepo repositories.ClassificationRepository) Service {
	return &service{
		classificationRepo: classificationRepo,
	}
}

// ClassifyEntity классифицирует сущность
func (s *service) ClassifyEntity(ctx context.Context, entityID string, entityType string, category string) (*Classification, error) {
	if entityID == "" {
		return nil, ErrInvalidEntityID
	}
	if category == "" {
		return nil, ErrInvalidCategory
	}

	// Проверяем, есть ли уже классификация
	existing, err := s.classificationRepo.GetByEntityID(ctx, entityID)
	if err == nil && existing != nil {
		// Обновляем существующую классификацию
		existing.Category = category
		existing.UpdatedAt = time.Now()
		if err := s.classificationRepo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update classification: %w", err)
		}
		return s.toDomainClassification(existing), nil
	}

	// Создаем новую классификацию
	classification := &repositories.Classification{
		ID:          fmt.Sprintf("cls_%d", time.Now().UnixNano()),
		EntityID:    entityID,
		EntityType:  entityType,
		Category:    category,
		Subcategory: "",
		Confidence:  1.0, // По умолчанию 100% уверенность для ручной классификации
		Rule:        "",
		Source:      "manual",
		ProcessedAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.classificationRepo.Create(ctx, classification); err != nil {
		return nil, fmt.Errorf("failed to create classification: %w", err)
	}

	return s.toDomainClassification(classification), nil
}

// BatchClassify выполняет пакетную классификацию
func (s *service) BatchClassify(ctx context.Context, entityIDs []string, entityType string, category string) (*BatchClassificationResult, error) {
	startTime := time.Now()
	result := &BatchClassificationResult{
		Total:   len(entityIDs),
		Results: make([]*Classification, 0, len(entityIDs)),
		Errors:  make([]ClassificationError, 0),
	}

	for _, entityID := range entityIDs {
		classification, err := s.ClassifyEntity(ctx, entityID, entityType, category)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, ClassificationError{
				EntityID: entityID,
				Error:    err.Error(),
				Severity: "medium",
			})
			continue
		}

		result.Successful++
		result.Results = append(result.Results, classification)
	}

	result.ProcessingTime = time.Since(startTime)
	return result, nil
}

// GetClassification возвращает классификацию по ID
func (s *service) GetClassification(ctx context.Context, classificationID string) (*Classification, error) {
	classification, err := s.classificationRepo.GetByID(ctx, classificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification: %w", err)
	}

	if classification == nil {
		return nil, ErrClassificationNotFound
	}

	return s.toDomainClassification(classification), nil
}

// GetClassificationByEntity возвращает классификацию по ID сущности
func (s *service) GetClassificationByEntity(ctx context.Context, entityID string) (*Classification, error) {
	classification, err := s.classificationRepo.GetByEntityID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification: %w", err)
	}

	if classification == nil {
		return nil, ErrClassificationNotFound
	}

	return s.toDomainClassification(classification), nil
}

// GetClassificationHistory возвращает историю классификаций для сущности
func (s *service) GetClassificationHistory(ctx context.Context, entityID string) ([]*Classification, error) {
	history, err := s.classificationRepo.GetClassificationHistory(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification history: %w", err)
	}

	result := make([]*Classification, 0, len(history))
	for _, h := range history {
		result = append(result, s.toDomainClassification(&h))
	}

	return result, nil
}

// UpdateClassification обновляет классификацию
func (s *service) UpdateClassification(ctx context.Context, classificationID string, category string, subcategory string, confidence float64) (*Classification, error) {
	if confidence < 0 || confidence > 1 {
		return nil, ErrInvalidConfidence
	}

	classification, err := s.classificationRepo.GetByID(ctx, classificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification: %w", err)
	}

	if classification == nil {
		return nil, ErrClassificationNotFound
	}

	classification.Category = category
	classification.Subcategory = subcategory
	classification.Confidence = confidence
	classification.UpdatedAt = time.Now()

	if err := s.classificationRepo.Update(ctx, classification); err != nil {
		return nil, fmt.Errorf("failed to update classification: %w", err)
	}

	return s.toDomainClassification(classification), nil
}

// DeleteClassification удаляет классификацию
func (s *service) DeleteClassification(ctx context.Context, classificationID string) error {
	if err := s.classificationRepo.Delete(ctx, classificationID); err != nil {
		return fmt.Errorf("failed to delete classification: %w", err)
	}
	return nil
}

// GetClassificationStatistics возвращает статистику классификации
func (s *service) GetClassificationStatistics(ctx context.Context) (*ClassificationStatistics, error) {
	stats, err := s.classificationRepo.GetClassificationStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification statistics: %w", err)
	}

	if stats == nil {
		return &ClassificationStatistics{}, nil
	}

	return &ClassificationStatistics{
		TotalClassifications:      stats.TotalClassifications,
		SuccessfulClassifications: stats.SuccessfulClassifications,
		FailedClassifications:     stats.FailedClassifications,
		AverageConfidence:         stats.AverageConfidence,
		AccuracyByCategory:        stats.AccuracyByCategory,
	}, nil
}

// GetClassificationAccuracy возвращает точность классификации для категории
func (s *service) GetClassificationAccuracy(ctx context.Context, category string) (float64, error) {
	accuracy, err := s.classificationRepo.GetClassificationAccuracy(ctx, category)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get classification accuracy: %w", err)
	}
	return accuracy, nil
}

// GetClassificationsByCategory возвращает классификации по категории с пагинацией
func (s *service) GetClassificationsByCategory(ctx context.Context, category string, limit, offset int) ([]*Classification, int64, error) {
	// TODO: Добавить метод GetByCategory в репозиторий
	return []*Classification{}, 0, fmt.Errorf("not implemented yet")
}

// ClassifyHierarchical выполняет иерархическую классификацию
func (s *service) ClassifyHierarchical(ctx context.Context, entityID string, entityType string, category string) (*Classification, error) {
	// TODO: Интегрировать с HierarchicalClassifier
	return s.ClassifyEntity(ctx, entityID, entityType, category)
}

// toDomainClassification преобразует repositories.Classification в domain Classification
func (s *service) toDomainClassification(c *repositories.Classification) *Classification {
	return &Classification{
		ID:          c.ID,
		EntityID:    c.EntityID,
		EntityType:  c.EntityType,
		Category:    c.Category,
		Subcategory: c.Subcategory,
		Confidence:  c.Confidence,
		Rule:        c.Rule,
		Source:      c.Source,
		ProcessedAt: c.ProcessedAt,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

