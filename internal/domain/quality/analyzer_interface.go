package quality

import (
	"context"
)

// QualityAnalyzerInterface интерфейс для анализатора качества данных
// Абстрагирует работу с quality.QualityAnalyzer
type QualityAnalyzerInterface interface {
	// AnalyzeUpload запускает анализ качества для выгрузки
	AnalyzeUpload(ctx context.Context, uploadID int) error
	
	// GetQualityStats получает статистику качества для базы данных
	GetQualityStats(ctx context.Context, databasePath string) (interface{}, error)
}

