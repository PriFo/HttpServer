package services

import (
	"httpserver/database"
	apperrors "httpserver/server/errors"
)

// ReportService сервис для генерации отчетов
type ReportService struct {
	db         *database.DB
	normalizedDB *database.DB
	serviceDB  *database.ServiceDB
}

// NewReportService создает новый сервис отчетов
func NewReportService(db, normalizedDB *database.DB, serviceDB *database.ServiceDB) *ReportService {
	return &ReportService{
		db:          db,
		normalizedDB: normalizedDB,
		serviceDB:   serviceDB,
	}
}

// GenerateNormalizationReport генерирует отчет о нормализации
// Принимает функцию генерации отчета от Server
func (rs *ReportService) GenerateNormalizationReport(generateReport func() (interface{}, error)) (interface{}, error) {
	return generateReport()
}

// GenerateDataQualityReport генерирует отчет о качестве данных
// Принимает функцию генерации отчета от Server
func (rs *ReportService) GenerateDataQualityReport(projectID *int, generateReport func(*int) (interface{}, error)) (interface{}, error) {
	// Валидация project_id, если указан
	if projectID != nil && *projectID <= 0 {
		return nil, apperrors.NewValidationError("project_id должен быть положительным числом", nil)
	}

	return generateReport(projectID)
}

// GenerateQualityReport генерирует отчет о качестве нормализации
// Принимает функцию генерации отчета от Server
func (rs *ReportService) GenerateQualityReport(databasePath string, generateReport func(string) (interface{}, error)) (interface{}, error) {
	if databasePath == "" {
		return nil, apperrors.NewValidationError("путь к базе данных обязателен", nil)
	}

	return generateReport(databasePath)
}

