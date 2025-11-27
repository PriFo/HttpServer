package services

import (
	"errors"
	"testing"
)

// TestNewPatternDetectionService проверяет создание нового сервиса обнаружения паттернов
func TestNewPatternDetectionService(t *testing.T) {
	getAPIKey := func() string { return "test-key" }
	service := NewPatternDetectionService(getAPIKey)
	if service == nil {
		t.Error("NewPatternDetectionService() should not return nil")
	}
}

// TestPatternDetectionService_DetectPatterns_Success проверяет успешное обнаружение паттернов
func TestPatternDetectionService_DetectPatterns_Success(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	result, err := service.DetectPatterns("Test Name")
	if err != nil {
		t.Fatalf("DetectPatterns() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if result["original_name"] != "Test Name" {
		t.Errorf("Expected original_name 'Test Name', got %v", result["original_name"])
	}
}

// TestPatternDetectionService_DetectPatterns_EmptyName проверяет обработку пустого имени
func TestPatternDetectionService_DetectPatterns_EmptyName(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	_, err := service.DetectPatterns("")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

// TestPatternDetectionService_SuggestPatternCorrection_Success проверяет успешное предложение исправления
func TestPatternDetectionService_SuggestPatternCorrection_Success(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	result, err := service.SuggestPatternCorrection("Test Name", false)
	if err != nil {
		t.Fatalf("SuggestPatternCorrection() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if result["original_name"] != "Test Name" {
		t.Errorf("Expected original_name 'Test Name', got %v", result["original_name"])
	}
}

// TestPatternDetectionService_SuggestPatternCorrection_EmptyName проверяет обработку пустого имени
func TestPatternDetectionService_SuggestPatternCorrection_EmptyName(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	_, err := service.SuggestPatternCorrection("", false)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

// TestPatternDetectionService_TestPatternsBatch_Success проверяет успешное тестирование паттернов на выборке
func TestPatternDetectionService_TestPatternsBatch_Success(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	getNames := func(limit int, table, column string) ([]string, error) {
		return []string{"Name 1", "Name 2", "Name 3"}, nil
	}

	result, err := service.TestPatternsBatch(10, false, "test_table", "test_column", getNames)
	if err != nil {
		t.Fatalf("TestPatternsBatch() error = %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if total, ok := result["total_analyzed"].(int); !ok || total != 3 {
		t.Errorf("Expected total_analyzed 3, got %v", result["total_analyzed"])
	}
}

// TestPatternDetectionService_TestPatternsBatch_GetNamesError проверяет обработку ошибки при получении имен
func TestPatternDetectionService_TestPatternsBatch_GetNamesError(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	getNames := func(limit int, table, column string) ([]string, error) {
		return nil, errors.New("failed to get names")
	}

	_, err := service.TestPatternsBatch(10, false, "test_table", "test_column", getNames)
	if err == nil {
		t.Error("Expected error when getNames fails")
	}
}

// TestPatternDetectionService_TestPatternsBatch_DefaultValues проверяет использование значений по умолчанию
func TestPatternDetectionService_TestPatternsBatch_DefaultValues(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	getNames := func(limit int, table, column string) ([]string, error) {
		// Проверяем, что используются значения по умолчанию
		if table != "catalog_items" {
			t.Errorf("Expected default table 'catalog_items', got '%s'", table)
		}
		if column != "name" {
			t.Errorf("Expected default column 'name', got '%s'", column)
		}
		if limit != 50 {
			t.Errorf("Expected default limit 50, got %d", limit)
		}
		return []string{}, nil
	}

	_, err := service.TestPatternsBatch(0, false, "", "", getNames)
	if err != nil {
		t.Fatalf("TestPatternsBatch() error = %v", err)
	}
}

// TestPatternDetectionService_TestPatternsBatch_LimitBounds проверяет ограничения лимита
func TestPatternDetectionService_TestPatternsBatch_LimitBounds(t *testing.T) {
	getAPIKey := func() string { return "" }
	service := NewPatternDetectionService(getAPIKey)

	getNames := func(limit int, table, column string) ([]string, error) {
		// Лимит должен быть ограничен максимумом 500
		if limit > 500 {
			t.Errorf("Expected limit <= 500, got %d", limit)
		}
		return []string{}, nil
	}

	_, err := service.TestPatternsBatch(1000, false, "test_table", "test_column", getNames)
	if err != nil {
		t.Fatalf("TestPatternsBatch() error = %v", err)
	}
}

