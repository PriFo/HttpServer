package services

import (
	"errors"
	"testing"
)

// TestNewReportService проверяет создание нового сервиса отчетов
func TestNewReportService(t *testing.T) {
	service := NewReportService(nil, nil, nil)
	if service == nil {
		t.Error("NewReportService() should not return nil")
	}
}

// TestReportService_GenerateNormalizationReport_Success проверяет успешную генерацию отчета о нормализации
func TestReportService_GenerateNormalizationReport_Success(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	generateReport := func() (interface{}, error) {
		return map[string]interface{}{
			"status": "success",
			"data":   "report data",
		}, nil
	}

	result, err := service.GenerateNormalizationReport(generateReport)
	if err != nil {
		t.Fatalf("GenerateNormalizationReport() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestReportService_GenerateNormalizationReport_Error проверяет обработку ошибки при генерации отчета
func TestReportService_GenerateNormalizationReport_Error(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	generateReport := func() (interface{}, error) {
		return nil, errors.New("report generation failed")
	}

	_, err := service.GenerateNormalizationReport(generateReport)
	if err == nil {
		t.Error("Expected error when report generation fails")
	}
}

// TestReportService_GenerateDataQualityReport_Success проверяет успешную генерацию отчета о качестве данных
func TestReportService_GenerateDataQualityReport_Success(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	projectID := 1
	generateReport := func(id *int) (interface{}, error) {
		return map[string]interface{}{
			"project_id": id,
			"status":     "success",
		}, nil
	}

	result, err := service.GenerateDataQualityReport(&projectID, generateReport)
	if err != nil {
		t.Fatalf("GenerateDataQualityReport() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestReportService_GenerateDataQualityReport_NilProjectID проверяет обработку nil projectID
func TestReportService_GenerateDataQualityReport_NilProjectID(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	generateReport := func(id *int) (interface{}, error) {
		return map[string]interface{}{
			"project_id": id,
			"status":     "success",
		}, nil
	}

	result, err := service.GenerateDataQualityReport(nil, generateReport)
	if err != nil {
		t.Fatalf("GenerateDataQualityReport() should accept nil projectID, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestReportService_GenerateDataQualityReport_InvalidProjectID проверяет обработку невалидного projectID
func TestReportService_GenerateDataQualityReport_InvalidProjectID(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	projectID := 0
	generateReport := func(id *int) (interface{}, error) {
		return nil, nil
	}

	_, err := service.GenerateDataQualityReport(&projectID, generateReport)
	if err == nil {
		t.Error("Expected error for invalid projectID (<= 0)")
	}
}

// TestReportService_GenerateDataQualityReport_NegativeProjectID проверяет обработку отрицательного projectID
func TestReportService_GenerateDataQualityReport_NegativeProjectID(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	projectID := -1
	generateReport := func(id *int) (interface{}, error) {
		return nil, nil
	}

	_, err := service.GenerateDataQualityReport(&projectID, generateReport)
	if err == nil {
		t.Error("Expected error for negative projectID")
	}
}

// TestReportService_GenerateQualityReport_Success проверяет успешную генерацию отчета о качестве нормализации
func TestReportService_GenerateQualityReport_Success(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	databasePath := "/path/to/database.db"
	generateReport := func(path string) (interface{}, error) {
		return map[string]interface{}{
			"database_path": path,
			"status":        "success",
		}, nil
	}

	result, err := service.GenerateQualityReport(databasePath, generateReport)
	if err != nil {
		t.Fatalf("GenerateQualityReport() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// TestReportService_GenerateQualityReport_EmptyPath проверяет обработку пустого пути к базе данных
func TestReportService_GenerateQualityReport_EmptyPath(t *testing.T) {
	service := NewReportService(nil, nil, nil)

	generateReport := func(path string) (interface{}, error) {
		return nil, nil
	}

	_, err := service.GenerateQualityReport("", generateReport)
	if err == nil {
		t.Error("Expected error for empty database path")
	}
}

