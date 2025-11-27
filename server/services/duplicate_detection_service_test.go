package services

import (
	"testing"

	"httpserver/normalization/algorithms"
)

// TestNewDuplicateDetectionService проверяет создание нового сервиса обнаружения дубликатов
func TestNewDuplicateDetectionService(t *testing.T) {
	service := NewDuplicateDetectionService()
	if service == nil {
		t.Error("NewDuplicateDetectionService() should not return nil")
	}
}

// TestDuplicateDetectionService_StartDetection проверяет запуск обнаружения дубликатов
func TestDuplicateDetectionService_StartDetection(t *testing.T) {
	service := NewDuplicateDetectionService()

	taskID, err := service.StartDetection(1, 0.75, 100, false, nil, 1000)
	if err != nil {
		t.Fatalf("StartDetection() error = %v", err)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}
}

// TestDuplicateDetectionService_StartDetection_InvalidProjectID проверяет обработку невалидного projectID
func TestDuplicateDetectionService_StartDetection_InvalidProjectID(t *testing.T) {
	service := NewDuplicateDetectionService()

	_, err := service.StartDetection(0, 0.75, 100, false, nil, 1000)
	if err == nil {
		t.Error("Expected error for invalid projectID")
	}
}

// TestDuplicateDetectionService_StartDetection_InvalidThreshold проверяет обработку невалидного threshold
func TestDuplicateDetectionService_StartDetection_InvalidThreshold(t *testing.T) {
	service := NewDuplicateDetectionService()

	// Threshold > 1 должен быть скорректирован
	taskID, err := service.StartDetection(1, 1.5, 100, false, nil, 1000)
	if err != nil {
		t.Fatalf("StartDetection() should handle invalid threshold, got error: %v", err)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}
}

// TestDuplicateDetectionService_StartDetection_InvalidBatchSize проверяет обработку невалидного batchSize
func TestDuplicateDetectionService_StartDetection_InvalidBatchSize(t *testing.T) {
	service := NewDuplicateDetectionService()

	// batchSize <= 0 должен быть скорректирован
	taskID, err := service.StartDetection(1, 0.75, 0, false, nil, 1000)
	if err != nil {
		t.Fatalf("StartDetection() should handle invalid batchSize, got error: %v", err)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}
}

// TestDuplicateDetectionService_StartDetection_WithWeights проверяет запуск с весами
func TestDuplicateDetectionService_StartDetection_WithWeights(t *testing.T) {
	service := NewDuplicateDetectionService()

	weights := &algorithms.SimilarityWeights{
		JaroWinkler: 0.3,
		LCS:         0.2,
		Ngram:       0.3,
		Phonetic:    0.2,
	}

	taskID, err := service.StartDetection(1, 0.75, 100, false, weights, 1000)
	if err != nil {
		t.Fatalf("StartDetection() error = %v", err)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}
}

// TestDuplicateDetectionService_GetTaskStatus проверяет получение статуса задачи
func TestDuplicateDetectionService_GetTaskStatus(t *testing.T) {
	service := NewDuplicateDetectionService()

	taskID, err := service.StartDetection(1, 0.75, 100, false, nil, 1000)
	if err != nil {
		t.Fatalf("StartDetection() error = %v", err)
	}

	task, err := service.GetTaskStatus(taskID)
	if err != nil {
		t.Fatalf("GetTaskStatus() error = %v", err)
	}

	if task == nil {
		t.Error("Expected non-nil task")
	}

	if task.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", task.Status)
	}
}

// TestDuplicateDetectionService_GetTaskStatus_NotFound проверяет обработку несуществующей задачи
func TestDuplicateDetectionService_GetTaskStatus_NotFound(t *testing.T) {
	service := NewDuplicateDetectionService()

	_, err := service.GetTaskStatus("non-existent-task")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}


