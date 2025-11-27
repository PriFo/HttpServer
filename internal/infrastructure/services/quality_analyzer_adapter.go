package services

import (
	"context"
	"fmt"

	qualitydomain "httpserver/internal/domain/quality"
	"httpserver/server/services"
)

// qualityAnalyzerAdapter адаптер для services.QualityService
type qualityAnalyzerAdapter struct {
	qualityService services.QualityServiceInterface
}

// NewQualityAnalyzerAdapter создает новый адаптер для анализатора качества
func NewQualityAnalyzerAdapter(qualityService services.QualityServiceInterface) qualitydomain.QualityAnalyzerInterface {
	return &qualityAnalyzerAdapter{
		qualityService: qualityService,
	}
}

// AnalyzeUpload запускает анализ качества для выгрузки
func (a *qualityAnalyzerAdapter) AnalyzeUpload(ctx context.Context, uploadID int) error {
	if a.qualityService == nil {
		return fmt.Errorf("quality service is not initialized")
	}

	// QualityService использует uploadUUID (string), нужно преобразовать
	uploadUUID := fmt.Sprintf("%d", uploadID)
	return a.qualityService.AnalyzeQuality(ctx, uploadUUID)
}

// GetQualityStats получает статистику качества для базы данных
func (a *qualityAnalyzerAdapter) GetQualityStats(ctx context.Context, databasePath string) (interface{}, error) {
	if a.qualityService == nil {
		return nil, fmt.Errorf("quality service is not initialized")
	}

	// QualityService.GetQualityStats требует currentDB, но мы можем передать nil
	// если databasePath указан
	return a.qualityService.GetQualityStats(ctx, databasePath, nil)
}

