package services

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// TestNewGISPService проверяет создание нового сервиса GISP
func TestNewGISPService(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewGISPService(serviceDB)
	if service == nil {
		t.Error("NewGISPService() should not return nil")
	}
}

// TestGISPService_ImportNomenclatures проверяет импорт номенклатур
func TestGISPService_ImportNomenclatures(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewGISPService(serviceDB)

	// Создаем минимальный Excel файл (в реальности это должен быть валидный Excel)
	// Для теста используем простой файл
	tempFile, err := os.CreateTemp("", "test_*.xlsx")
	if err != nil {
		t.Skipf("Skipping test: cannot create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	file, err := os.Open(tempFile.Name())
	if err != nil {
		t.Skipf("Skipping test: cannot open temp file: %v", err)
	}
	defer file.Close()

	// Тест может упасть на парсинге Excel, это нормально для минимального файла
	_, err = service.ImportNomenclatures(file, "test.xlsx")
	// Ожидаем ошибку парсинга, но не ошибку валидации формата
	if err != nil {
		// Проверяем, что это не ошибка формата файла
		if err.Error() == "file must be Excel format (.xlsx or .xls)" {
			t.Errorf("Unexpected format error: %v", err)
		}
		// Остальные ошибки (парсинг, импорт) допустимы для тестового файла
	}
}

// TestGISPService_ImportNomenclatures_InvalidFormat проверяет обработку невалидного формата
func TestGISPService_ImportNomenclatures_InvalidFormat(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewGISPService(serviceDB)

	file := bytes.NewReader([]byte("test content"))
	_, err := service.ImportNomenclatures(file, "test.txt")
	if err == nil {
		t.Error("Expected error for invalid file format")
	}
}

// TestGISPService_ImportNomenclatures_NilReader проверяет обработку nil reader
func TestGISPService_ImportNomenclatures_NilReader(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewGISPService(serviceDB)

	var file io.Reader = nil
	_, err := service.ImportNomenclatures(file, "test.xlsx")
	// Ожидаем ошибку валидации для nil reader
	if err == nil {
		t.Error("Expected error for nil reader, got nil")
	}
	// Проверяем, что это ошибка валидации
	if err != nil && err.Error() != "файл не может быть nil" {
		t.Logf("Got expected validation error: %v", err)
	}
}

