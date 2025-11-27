package services

import (
	"context"

	"httpserver/database"
	"httpserver/server/types"
)

// QualityServiceInterface интерфейс для сервиса качества данных
// Используется для улучшения тестируемости и возможности замены реализации
type QualityServiceInterface interface {
	// GetQualityStats получает статистику качества для базы данных.
	// Если databasePath не пустой, открывает новое подключение к указанной базе данных.
	// Если databasePath пустой, использует currentDB (который не должен быть nil).
	// currentDB должен реализовывать DatabaseInterface.
	GetQualityStats(ctx context.Context, databasePath string, currentDB DatabaseInterface) (interface{}, error)

	// GetQualityReport получает отчет о качестве для выгрузки.
	// uploadUUID - UUID выгрузки для которой нужно получить отчет.
	// summaryOnly - если true, возвращает только сводку без детальных проблем.
	// limit и offset используются для пагинации проблем качества.
	GetQualityReport(ctx context.Context, uploadUUID string, summaryOnly bool, limit, offset int) (*types.QualityReport, error)

	// AnalyzeQuality запускает анализ качества для выгрузки.
	// uploadUUID - UUID выгрузки для которой нужно запустить анализ.
	AnalyzeQuality(ctx context.Context, uploadUUID string) error

	// GetQualityDashboard получает дашборд качества для базы данных.
	// databaseID - ID базы данных.
	// days - количество дней для расчета трендов.
	// limit - максимальное количество топ проблем для возврата.
	GetQualityDashboard(ctx context.Context, databaseID int, days int, limit int) (*types.QualityDashboard, error)

	// GetQualityIssues получает проблемы качества для базы данных.
	// databaseID - ID базы данных.
	// filters - карта фильтров для проблем (entity_type, severity, status и т.д.).
	GetQualityIssues(ctx context.Context, databaseID int, filters map[string]interface{}) ([]database.DataQualityIssue, error)

	// GetQualityTrends получает тренды качества для базы данных.
	// databaseID - ID базы данных.
	// days - количество дней для расчета трендов.
	GetQualityTrends(ctx context.Context, databaseID int, days int) ([]database.QualityTrend, error)
}


