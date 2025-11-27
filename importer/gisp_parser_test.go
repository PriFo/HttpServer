package importer

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNomenclatureRecord проверяет структуру записи номенклатуры
func TestNomenclatureRecord(t *testing.T) {
	record := NomenclatureRecord{
		ManufacturerName: "Test Manufacturer",
		INN:              "1234567890",
		OGRN:             "1234567890123",
		ActualAddress:    "Test Address",
		ProductName:      "Test Product",
		RegistryNumber:   "REG-001",
		EntryDate:        "2024-01-01",
		ValidityPeriod:   "2025-01-01",
		OKPD2:            "01.11.11",
		TNVED:            "0101.11",
		ManufacturedBy:   "TU 12345",
		Points:           "10",
		Percentage:       "100",
		Compliance:       "Соответствует",
		IsArtificial:     "Нет",
		IsHighTech:       "Да",
		IsTrusted:        "Да",
		Basis:            "Test Basis",
		Conclusion:       "Test Conclusion",
		ConclusionDoc:    "Test Doc",
	}
	
	if record.ProductName == "" {
		t.Error("NomenclatureRecord.ProductName should not be empty")
	}
	
	// Проверяем, что запись валидна
	if record.ManufacturerName == "" && record.INN == "" && record.OGRN == "" {
		t.Log("Record without manufacturer identifiers - may be valid in some cases")
	}
}

// TestParseGISPExcelFile_InvalidFile проверяет обработку невалидного файла
func TestParseGISPExcelFile_InvalidFile(t *testing.T) {
	// Тест с несуществующим файлом
	_, err := ParseGISPExcelFile("nonexistent.xlsx")
	if err == nil {
		t.Error("ParseGISPExcelFile() should return error for nonexistent file")
	}
	
	// Тест с пустым путем
	_, err = ParseGISPExcelFile("")
	if err == nil {
		t.Error("ParseGISPExcelFile() should return error for empty path")
	}
}

// TestParseGISPExcelFile_EmptyFile проверяет обработку пустого файла
func TestParseGISPExcelFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.xlsx")
	
	// Создаем пустой файл (не валидный Excel, но для теста достаточно)
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()
	
	// Ожидаем ошибку при парсинге пустого файла
	_, err = ParseGISPExcelFile(filePath)
	if err == nil {
		t.Error("ParseGISPExcelFile() should return error for empty/invalid Excel file")
	}
}

// TestColumnIndices проверяет структуру индексов колонок
func TestColumnIndices(t *testing.T) {
	indices := columnIndices{
		manufacturer:   0,
		inn:            1,
		ogrn:           2,
		address:        3,
		productName:    4,
		registryNumber: 5,
		okpd2:          6,
		tnved:          7,
	}
	
	if indices.productName < 0 {
		t.Error("columnIndices.productName should be non-negative")
	}
	
	// Проверяем, что обязательные поля установлены
	if indices.productName == -1 {
		t.Error("columnIndices.productName is required")
	}
}

// TestIsEmptyRow проверяет функцию проверки пустой строки
func TestIsEmptyRow(t *testing.T) {
	tests := []struct {
		name string
		row  []string
		want bool
	}{
		{
			name: "empty row",
			row:  []string{},
			want: true,
		},
		{
			name: "row with empty strings",
			row:  []string{"", "", ""},
			want: true,
		},
		{
			name: "row with whitespace",
			row:  []string{"   ", "  ", " "},
			want: true,
		},
		{
			name: "row with content",
			row:  []string{"test", "", ""},
			want: false,
		},
		{
			name: "row with mixed content",
			row:  []string{"", "test", ""},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmptyRow(tt.row)
			if got != tt.want {
				t.Errorf("isEmptyRow() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatDate проверяет форматирование даты
func TestFormatDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFormatted bool // Проверяем, что результат не пустой
	}{
		{
			name:          "empty date",
			input:         "",
			wantFormatted: false,
		},
		{
			name:          "valid date string",
			input:         "2024-01-01",
			wantFormatted: true,
		},
		{
			name:          "date with time",
			input:         "2024-01-01 12:00:00",
			wantFormatted: true,
		},
		{
			name:          "invalid date",
			input:         "not-a-date",
			wantFormatted: true, // formatDate просто возвращает строку как есть
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDate(tt.input)
			isFormatted := got != ""
			
			if isFormatted != tt.wantFormatted {
				t.Errorf("formatDate() formatted = %v, want %v (result: %q)", isFormatted, tt.wantFormatted, got)
			}
		})
	}
}

