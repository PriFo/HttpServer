package services

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewProcessing1CService проверяет создание нового сервиса обработки 1С
func TestNewProcessing1CService(t *testing.T) {
	service := NewProcessing1CService()
	if service == nil {
		t.Error("NewProcessing1CService() should not return nil")
	}
}

// TestProcessing1CService_GenerateProcessingXML проверяет генерацию XML обработки
func TestProcessing1CService_GenerateProcessingXML(t *testing.T) {
	service := NewProcessing1CService()

	// Создаем необходимые директории и файлы для теста
	moduleDir := filepath.Join(service.workDir, "1c_processing", "Module")
	err := os.MkdirAll(moduleDir, 0755)
	if err != nil {
		t.Skipf("Skipping test: cannot create module directory: %v", err)
	}
	defer os.RemoveAll(filepath.Join(service.workDir, "1c_processing"))

	moduleFile := filepath.Join(moduleDir, "Module.bsl")
	err = os.WriteFile(moduleFile, []byte("// Test module code"), 0644)
	if err != nil {
		t.Skipf("Skipping test: cannot create module file: %v", err)
	}

	xml, err := service.GenerateProcessingXML()
	if err != nil {
		t.Fatalf("GenerateProcessingXML() error = %v", err)
	}

	if xml == "" {
		t.Error("Expected non-empty XML")
	}
}

// TestProcessing1CService_GenerateProcessingXML_NoModule проверяет обработку отсутствия модуля
func TestProcessing1CService_GenerateProcessingXML_NoModule(t *testing.T) {
	service := NewProcessing1CService()

	// Удаляем директорию модуля, если она существует
	moduleDir := filepath.Join(service.workDir, "1c_processing")
	os.RemoveAll(moduleDir)

	_, err := service.GenerateProcessingXML()
	if err == nil {
		t.Error("Expected error when module file does not exist")
	}
}

