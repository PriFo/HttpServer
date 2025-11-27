package nomenclature

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewKpvedProcessor проверяет создание нового процессора КПВЭД
func TestNewKpvedProcessor(t *testing.T) {
	processor := NewKpvedProcessor()
	
	if processor == nil {
		t.Fatal("NewKpvedProcessor() returned nil")
	}
	
	if processor.codes == nil {
		t.Error("KpvedProcessor.codes is nil")
	}
	
	if processor.loaded {
		t.Error("KpvedProcessor should not be loaded initially")
	}
}

// TestKpvedProcessor_LoadKpved проверяет загрузку данных КПВЭД
func TestKpvedProcessor_LoadKpved(t *testing.T) {
	processor := NewKpvedProcessor()
	
	// Создаем временный файл с тестовыми данными КПВЭД
	tempDir := t.TempDir()
	kpvedPath := filepath.Join(tempDir, "kpved.txt")
	
	testData := `01.11.1	Продукция растениеводства
01.12.10	Зерновые культуры
10.51.11	Молочная продукция
`
	
	if err := os.WriteFile(kpvedPath, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test KPVED file: %v", err)
	}
	
	err := processor.LoadKpved(kpvedPath)
	if err != nil {
		t.Fatalf("LoadKpved() failed: %v", err)
	}
	
	if !processor.loaded {
		t.Error("KpvedProcessor should be loaded after LoadKpved()")
	}
	
	if processor.data == "" {
		t.Error("KpvedProcessor.data should not be empty after loading")
	}
	
	// Проверяем, что коды извлечены
	if len(processor.codes) == 0 {
		t.Error("KpvedProcessor.codes should contain extracted codes")
	}
}

// TestKpvedProcessor_LoadKpved_NonExistentFile проверяет обработку несуществующего файла
func TestKpvedProcessor_LoadKpved_NonExistentFile(t *testing.T) {
	processor := NewKpvedProcessor()
	
	err := processor.LoadKpved("nonexistent.txt")
	if err == nil {
		t.Error("LoadKpved() should return error for non-existent file")
	}
}

// TestKpvedProcessor_LoadKpved_EmptyFile проверяет обработку пустого файла
func TestKpvedProcessor_LoadKpved_EmptyFile(t *testing.T) {
	processor := NewKpvedProcessor()
	
	tempDir := t.TempDir()
	kpvedPath := filepath.Join(tempDir, "empty.txt")
	
	if err := os.WriteFile(kpvedPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}
	
	err := processor.LoadKpved(kpvedPath)
	if err != nil {
		t.Fatalf("LoadKpved() should not fail for empty file: %v", err)
	}
	
	if !processor.loaded {
		t.Error("KpvedProcessor should be loaded even for empty file")
	}
}

// TestKpvedProcessor_CodeExists проверяет проверку существования кода
func TestKpvedProcessor_CodeExists(t *testing.T) {
	processor := NewKpvedProcessor()
	
	tempDir := t.TempDir()
	kpvedPath := filepath.Join(tempDir, "kpved.txt")
	
	testData := `01.11.1	Продукция растениеводства
01.12.10	Зерновые культуры
`
	
	if err := os.WriteFile(kpvedPath, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	if err := processor.LoadKpved(kpvedPath); err != nil {
		t.Fatalf("LoadKpved() failed: %v", err)
	}
	
	// Проверяем существующий код
	if !processor.CodeExists("01.11.1") {
		t.Error("CodeExists() should return true for existing code")
	}
	
	// Проверяем несуществующий код
	if processor.CodeExists("99.99.99") {
		t.Error("CodeExists() should return false for non-existent code")
	}
	
	// Проверяем пустой код
	if processor.CodeExists("") {
		t.Error("CodeExists() should return false for empty code")
	}
}

// TestKpvedProcessor_GetData проверяет получение данных
func TestKpvedProcessor_GetData(t *testing.T) {
	processor := NewKpvedProcessor()
	
	// До загрузки данных должно быть пусто
	data := processor.GetData()
	if data != "" {
		t.Errorf("GetData() = %q, want empty string before loading", data)
	}
	
	// Загружаем данные
	tempDir := t.TempDir()
	kpvedPath := filepath.Join(tempDir, "kpved.txt")
	testData := "01.11.1	Test\n"
	
	if err := os.WriteFile(kpvedPath, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	if err := processor.LoadKpved(kpvedPath); err != nil {
		t.Fatalf("LoadKpved() failed: %v", err)
	}
	
	// После загрузки данные должны быть доступны
	data = processor.GetData()
	if data == "" {
		t.Error("GetData() should return data after loading")
	}
	
	if !strings.Contains(data, "01.11.1") {
		t.Error("GetData() should contain loaded data")
	}
}

// TestKpvedProcessor_LoadKpved_Twice проверяет повторную загрузку
func TestKpvedProcessor_LoadKpved_Twice(t *testing.T) {
	processor := NewKpvedProcessor()
	
	tempDir := t.TempDir()
	kpvedPath1 := filepath.Join(tempDir, "kpved1.txt")
	kpvedPath2 := filepath.Join(tempDir, "kpved2.txt")
	
	testData1 := "01.11.1	First\n"
	testData2 := "02.22.2	Second\n"
	
	if err := os.WriteFile(kpvedPath1, []byte(testData1), 0644); err != nil {
		t.Fatalf("Failed to create first file: %v", err)
	}
	
	if err := os.WriteFile(kpvedPath2, []byte(testData2), 0644); err != nil {
		t.Fatalf("Failed to create second file: %v", err)
	}
	
	// Первая загрузка
	if err := processor.LoadKpved(kpvedPath1); err != nil {
		t.Fatalf("First LoadKpved() failed: %v", err)
	}
	
	firstData := processor.GetData()
	
	// Вторая загрузка (должна быть проигнорирована)
	if err := processor.LoadKpved(kpvedPath2); err != nil {
		t.Fatalf("Second LoadKpved() failed: %v", err)
	}
	
	secondData := processor.GetData()
	
	// Данные должны остаться от первой загрузки
	if secondData != firstData {
		t.Error("LoadKpved() should not reload data if already loaded")
	}
}

// TestKpvedProcessor_ExtractKpvedCodes проверяет извлечение кодов
func TestKpvedProcessor_ExtractKpvedCodes(t *testing.T) {
	processor := NewKpvedProcessor()
	
	testCases := []struct {
		name     string
		line     string
		wantCode string
		wantFound bool
	}{
		{
			name:      "valid code",
			line:      "01.11.1	Description",
			wantCode:  "01.11.1",
			wantFound: true,
		},
		{
			name:      "code with spaces",
			line:      "  01.12.10  Description",
			wantCode:  "01.12.10",
			wantFound: true,
		},
		{
			name:      "no code",
			line:      "Just text without code",
			wantCode:  "",
			wantFound: false,
		},
		{
			name:      "empty line",
			line:      "",
			wantCode:  "",
			wantFound: false,
		},
		{
			name:      "short code",
			line:      "01	Description",
			wantCode:  "01",
			wantFound: true,
		},
	}
	
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			processor.extractKpvedCodes(tt.line)
			
			if tt.wantFound {
				if !processor.CodeExists(tt.wantCode) {
					t.Errorf("Code %s should be extracted from line: %s", tt.wantCode, tt.line)
				}
			} else {
				// Проверяем, что код не был добавлен (если он не пустой)
				if tt.wantCode != "" && processor.CodeExists(tt.wantCode) {
					t.Errorf("Code %s should not be extracted from line: %s", tt.wantCode, tt.line)
				}
			}
		})
	}
}

