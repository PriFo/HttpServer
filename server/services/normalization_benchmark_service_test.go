package services

import (
	"os"
	"testing"
	"time"
)

// TestNewNormalizationBenchmarkService проверяет создание нового сервиса бенчмарков
func TestNewNormalizationBenchmarkService(t *testing.T) {
	service := NewNormalizationBenchmarkService()
	if service == nil {
		t.Error("NewNormalizationBenchmarkService() should not return nil")
	}
}

// TestNormalizationBenchmarkService_UploadBenchmark проверяет загрузку бенчмарка
func TestNormalizationBenchmarkService_UploadBenchmark(t *testing.T) {
	service := NewNormalizationBenchmarkService()

	report := NormalizationBenchmarkReport{
		Timestamp:  time.Now().Format(time.RFC3339),
		TotalItems: 100,
		Results: map[string]interface{}{
			"success": 95,
			"errors":  5,
		},
		Metrics: map[string]interface{}{
			"avg_time": 1.5,
		},
		Config: map[string]interface{}{
			"version": "1.0",
		},
		Environment: map[string]interface{}{
			"os": "linux",
		},
	}

	result, err := service.UploadBenchmark(report)
	if err != nil {
		t.Fatalf("UploadBenchmark() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Очищаем созданный файл
	if filePath, ok := result["file_path"].(string); ok {
		os.Remove(filePath)
	}
}

// TestNormalizationBenchmarkService_UploadBenchmark_EmptyTimestamp проверяет обработку пустого timestamp
func TestNormalizationBenchmarkService_UploadBenchmark_EmptyTimestamp(t *testing.T) {
	service := NewNormalizationBenchmarkService()

	report := NormalizationBenchmarkReport{
		Timestamp:   "", // Пустой timestamp должен быть заменен
		TotalItems:  50,
		Results:     map[string]interface{}{},
		Metrics:     map[string]interface{}{},
		Config:      map[string]interface{}{},
		Environment: map[string]interface{}{},
	}

	result, err := service.UploadBenchmark(report)
	if err != nil {
		t.Fatalf("UploadBenchmark() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Очищаем созданный файл
	if filePath, ok := result["file_path"].(string); ok {
		os.Remove(filePath)
	}
}

// TestNormalizationBenchmarkService_ListBenchmarks проверяет получение списка бенчмарков
func TestNormalizationBenchmarkService_ListBenchmarks(t *testing.T) {
	service := NewNormalizationBenchmarkService()

	// Создаем тестовый бенчмарк
	report := NormalizationBenchmarkReport{
		Timestamp:   time.Now().Format(time.RFC3339),
		TotalItems:  10,
		Results:     map[string]interface{}{},
		Metrics:     map[string]interface{}{},
		Config:      map[string]interface{}{},
		Environment: map[string]interface{}{},
	}

	uploadResult, err := service.UploadBenchmark(report)
	if err != nil {
		t.Fatalf("UploadBenchmark() error = %v", err)
	}

	// Получаем список бенчмарков
	list, err := service.ListBenchmarks()
	if err != nil {
		t.Fatalf("ListBenchmarks() error = %v", err)
	}

	if list == nil {
		t.Error("Expected non-nil list")
	}

	// Очищаем созданный файл
	if filePath, ok := uploadResult["file_path"].(string); ok {
		os.Remove(filePath)
	}
}

// TestNormalizationBenchmarkService_ListBenchmarks_EmptyDir проверяет получение списка из пустой директории
func TestNormalizationBenchmarkService_ListBenchmarks_EmptyDir(t *testing.T) {
	service := NewNormalizationBenchmarkService()

	// Удаляем директорию, если она существует
	os.RemoveAll(service.benchmarksDir)

	list, err := service.ListBenchmarks()
	if err != nil {
		t.Fatalf("ListBenchmarks() error = %v", err)
	}

	if list == nil {
		t.Error("Expected non-nil list even for empty directory")
	}
}
