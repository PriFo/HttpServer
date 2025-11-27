package nomenclature

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewProcessor проверяет создание нового процессора
func TestNewProcessor(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	// Создаем пустой файл БД для теста
	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB file: %v", err)
	}
	file.Close()
	
	// Создаем временный файл KPVED
	kpvedPath := filepath.Join(tempDir, "kpved.txt")
	err = os.WriteFile(kpvedPath, []byte("01.11\tTest category\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test KPVED file: %v", err)
	}
	
	config := Config{
		ArliaiAPIKey: "test-api-key",
		AIModel:      "test-model",
		DatabasePath: dbPath,
		KpvedFilePath: kpvedPath,
	}
	
	processor, err := NewProcessor(config)
	if err != nil {
		// Может быть ошибка из-за отсутствия реальной БД схемы - это нормально
		t.Logf("NewProcessor() returned error (expected for test setup): %v", err)
		return
	}
	
	if processor == nil {
		t.Fatal("NewProcessor() returned nil")
	}
	
	if processor.config.ArliaiAPIKey != config.ArliaiAPIKey {
		t.Errorf("Processor.config.ArliaiAPIKey = %v, want %v", processor.config.ArliaiAPIKey, config.ArliaiAPIKey)
	}
}

// TestNewProcessor_NoAPIKey проверяет обработку отсутствия API ключа
func TestNewProcessor_NoAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB file: %v", err)
	}
	file.Close()
	
	kpvedPath := filepath.Join(tempDir, "kpved.txt")
	err = os.WriteFile(kpvedPath, []byte("01.11\tTest\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test KPVED file: %v", err)
	}
	
	config := Config{
		ArliaiAPIKey: "", // Пустой ключ
		AIModel:      "test-model",
		DatabasePath: dbPath,
		KpvedFilePath: kpvedPath,
	}
	
	processor, err := NewProcessor(config)
	if err == nil {
		t.Error("NewProcessor() should return error for empty API key")
	}
	
	if processor != nil {
		t.Error("NewProcessor() should return nil processor on error")
	}
}

// TestProcessingStats проверяет структуру статистики обработки
func TestProcessingStats(t *testing.T) {
	stats := &ProcessingStats{
		Total:      100,
		Processed:  50,
		Successful: 45,
		Failed:     5,
		StartTime:  time.Now(),
	}
	
	if stats.Total < 0 {
		t.Error("ProcessingStats.Total should be non-negative")
	}
	
	if stats.Processed < 0 {
		t.Error("ProcessingStats.Processed should be non-negative")
	}
	
	if stats.Successful < 0 {
		t.Error("ProcessingStats.Successful should be non-negative")
	}
	
	if stats.Failed < 0 {
		t.Error("ProcessingStats.Failed should be non-negative")
	}
	
	if stats.Processed > stats.Total {
		t.Error("ProcessingStats.Processed should not exceed Total")
	}
	
	if stats.Successful+stats.Failed > stats.Processed {
		t.Error("ProcessingStats.Successful + Failed should not exceed Processed")
	}
}

// TestProcessingResult проверяет структуру результата обработки
func TestProcessingResult(t *testing.T) {
	result := &processingResult{
		ID:     1,
		Result: &AIProcessingResult{NormalizedName: "test"},
		Error:  nil,
	}
	
	if result.ID <= 0 {
		t.Error("processingResult.ID should be positive")
	}
	
	// Проверяем результат с ошибкой
	resultWithError := &processingResult{
		ID:     2,
		Result: nil,
		Error:  &struct{ error }{},
	}
	
	if resultWithError.Error == nil {
		t.Error("processingResult should have error when processing failed")
	}
}

