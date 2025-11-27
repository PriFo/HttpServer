package classification

import (
	"context"
	"fmt"

	classificationdomain "httpserver/internal/domain/classification"
	"httpserver/internal/domain/repositories"
)

// UseCase представляет use case для работы с классификацией
// Координирует выполнение бизнес-логики между domain и infrastructure слоями
type UseCase struct {
	classificationRepo    repositories.ClassificationRepository
	classificationService classificationdomain.Service
}

// NewUseCase создает новый use case для классификации
func NewUseCase(
	classificationRepo repositories.ClassificationRepository,
	classificationService classificationdomain.Service,
) *UseCase {
	return &UseCase{
		classificationRepo:    classificationRepo,
		classificationService: classificationService,
	}
}

// ClassifyEntity классифицирует сущность
func (uc *UseCase) ClassifyEntity(ctx context.Context, entityID string, entityType string, category string) (*classificationdomain.Classification, error) {
	result, err := uc.classificationService.ClassifyEntity(ctx, entityID, entityType, category)
	if err != nil {
		return nil, fmt.Errorf("failed to classify entity: %w", err)
	}
	return result, nil
}

// BatchClassify выполняет пакетную классификацию
func (uc *UseCase) BatchClassify(ctx context.Context, entityIDs []string, entityType string, category string) (*classificationdomain.BatchClassificationResult, error) {
	result, err := uc.classificationService.BatchClassify(ctx, entityIDs, entityType, category)
	if err != nil {
		return nil, fmt.Errorf("failed to batch classify: %w", err)
	}
	return result, nil
}

// GetClassification возвращает классификацию по ID
func (uc *UseCase) GetClassification(ctx context.Context, classificationID string) (*classificationdomain.Classification, error) {
	result, err := uc.classificationService.GetClassification(ctx, classificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification: %w", err)
	}
	return result, nil
}

// GetClassificationByEntity возвращает классификацию по ID сущности
func (uc *UseCase) GetClassificationByEntity(ctx context.Context, entityID string) (*classificationdomain.Classification, error) {
	result, err := uc.classificationService.GetClassificationByEntity(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification by entity: %w", err)
	}
	return result, nil
}

// GetClassificationHistory возвращает историю классификаций
func (uc *UseCase) GetClassificationHistory(ctx context.Context, entityID string) ([]*classificationdomain.Classification, error) {
	result, err := uc.classificationService.GetClassificationHistory(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification history: %w", err)
	}
	return result, nil
}

// UpdateClassification обновляет классификацию
func (uc *UseCase) UpdateClassification(ctx context.Context, classificationID string, category string, subcategory string, confidence float64) (*classificationdomain.Classification, error) {
	result, err := uc.classificationService.UpdateClassification(ctx, classificationID, category, subcategory, confidence)
	if err != nil {
		return nil, fmt.Errorf("failed to update classification: %w", err)
	}
	return result, nil
}

// DeleteClassification удаляет классификацию
func (uc *UseCase) DeleteClassification(ctx context.Context, classificationID string) error {
	if err := uc.classificationService.DeleteClassification(ctx, classificationID); err != nil {
		return fmt.Errorf("failed to delete classification: %w", err)
	}
	return nil
}

// GetClassificationStatistics возвращает статистику классификации
func (uc *UseCase) GetClassificationStatistics(ctx context.Context) (*classificationdomain.ClassificationStatistics, error) {
	result, err := uc.classificationService.GetClassificationStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification statistics: %w", err)
	}
	return result, nil
}

// GetClassificationAccuracy возвращает точность классификации
func (uc *UseCase) GetClassificationAccuracy(ctx context.Context, category string) (float64, error) {
	result, err := uc.classificationService.GetClassificationAccuracy(ctx, category)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get classification accuracy: %w", err)
	}
	return result, nil
}

// ClassifyHierarchical выполняет иерархическую классификацию
func (uc *UseCase) ClassifyHierarchical(ctx context.Context, entityID string, entityType string, category string) (*classificationdomain.Classification, error) {
	result, err := uc.classificationService.ClassifyHierarchical(ctx, entityID, entityType, category)
	if err != nil {
		return nil, fmt.Errorf("failed to hierarchical classify: %w", err)
	}
	return result, nil
}

