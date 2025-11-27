package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"httpserver/database"
)

// setupTestScanDir создает тестовую директорию для сканирования
func setupTestScanDir(t *testing.T) (string, func()) {
	tempDir := t.TempDir()
	
	// Создаем поддиректории
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2")
	os.MkdirAll(subDir1, 0755)
	os.MkdirAll(subDir2, 0755)
	
	// Создаем тестовые файлы
	testFiles := []struct {
		path    string
		content []byte
	}{
		{filepath.Join(tempDir, "Выгрузка_Номенклатура_test1.db"), []byte("test1")},
		{filepath.Join(tempDir, "Выгрузка_Контрагенты_test2.db"), []byte("test2")},
		{filepath.Join(subDir1, "Выгрузка_Номенклатура_test3.db"), []byte("test3")},
		{filepath.Join(tempDir, "other_file.db"), []byte("other")}, // Не соответствует паттерну
		{filepath.Join(tempDir, "not_a_db.txt"), []byte("text")},    // Не .db файл
	}
	
	for _, tf := range testFiles {
		if err := os.WriteFile(tf.path, tf.content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.path, err)
		}
	}
	
	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	
	return tempDir, cleanup
}

// TestScanForDatabaseFiles проверяет сканирование файлов БД
func TestScanForDatabaseFiles(t *testing.T) {
	scanDir, cleanup := setupTestScanDir(t)
	defer cleanup()
	
	// Создаем тестовую сервисную БД
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "test_service.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer serviceDB.Close()
	
	foundFiles, err := ScanForDatabaseFiles([]string{scanDir}, serviceDB)
	if err != nil {
		t.Fatalf("ScanForDatabaseFiles() failed: %v", err)
	}
	
	// Должно найти 3 файла (2 в корне + 1 в поддиректории)
	// Файл other_file.db не соответствует паттерну, not_a_db.txt не .db
	expectedCount := 3
	if len(foundFiles) < expectedCount {
		t.Errorf("ScanForDatabaseFiles() found %d files, expected at least %d", len(foundFiles), expectedCount)
	}
	
	// Проверяем, что найденные файлы соответствуют паттернам
	for _, file := range foundFiles {
		fileName := filepath.Base(file)
		if !strings.HasPrefix(fileName, "Выгрузка_Номенклатура_") &&
			!strings.HasPrefix(fileName, "Выгрузка_Контрагенты_") {
			t.Errorf("Found file %s does not match expected patterns", fileName)
		}
		
		if !strings.HasSuffix(strings.ToLower(file), ".db") {
			t.Errorf("Found file %s is not a .db file", file)
		}
	}
}

// TestScanForDatabaseFiles_EmptyDir проверяет сканирование пустой директории
func TestScanForDatabaseFiles_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	
	serviceDBPath := filepath.Join(tempDir, "test_service.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer serviceDB.Close()
	
	foundFiles, err := ScanForDatabaseFiles([]string{tempDir}, serviceDB)
	if err != nil {
		t.Fatalf("ScanForDatabaseFiles() failed: %v", err)
	}
	
	if len(foundFiles) != 0 {
		t.Errorf("ScanForDatabaseFiles() found %d files in empty directory, expected 0", len(foundFiles))
	}
}

// TestScanForDatabaseFiles_NonExistentPath проверяет обработку несуществующего пути
func TestScanForDatabaseFiles_NonExistentPath(t *testing.T) {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "test_service.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer serviceDB.Close()
	
	nonExistentPath := filepath.Join(tempDir, "nonexistent")
	foundFiles, err := ScanForDatabaseFiles([]string{nonExistentPath}, serviceDB)
	
	// Не должно быть ошибки, просто пустой результат
	if err != nil {
		t.Errorf("ScanForDatabaseFiles() should not return error for non-existent path: %v", err)
	}
	
	if len(foundFiles) != 0 {
		t.Errorf("ScanForDatabaseFiles() found %d files for non-existent path, expected 0", len(foundFiles))
	}
}

// TestScanForDatabaseFiles_MultiplePaths проверяет сканирование нескольких путей
func TestScanForDatabaseFiles_MultiplePaths(t *testing.T) {
	scanDir1, cleanup1 := setupTestScanDir(t)
	defer cleanup1()
	
	scanDir2, cleanup2 := setupTestScanDir(t)
	defer cleanup2()
	
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "test_service.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer serviceDB.Close()
	
	foundFiles, err := ScanForDatabaseFiles([]string{scanDir1, scanDir2}, serviceDB)
	if err != nil {
		t.Fatalf("ScanForDatabaseFiles() failed: %v", err)
	}
	
	// Должно найти файлы из обеих директорий
	if len(foundFiles) < 6 { // Минимум 3 файла из каждой директории
		t.Errorf("ScanForDatabaseFiles() found %d files, expected at least 6", len(foundFiles))
	}
}

// TestMoveDatabaseToUploads проверяет перемещение файла в uploads
func TestMoveDatabaseToUploads(t *testing.T) {
	tempDir := t.TempDir()
	
	// Создаем исходный файл
	sourceFile := filepath.Join(tempDir, "test.db")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Создаем директорию uploads
	uploadsDir := filepath.Join(tempDir, "uploads")
	
	newPath, err := MoveDatabaseToUploads(sourceFile, uploadsDir)
	if err != nil {
		t.Fatalf("MoveDatabaseToUploads() failed: %v", err)
	}
	
	// Проверяем, что файл перемещен
	if _, err := os.Stat(newPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Errorf("File was not moved to %s", newPath)
		} else {
			t.Errorf("Error checking file %s: %v", newPath, err)
		}
	}
	
	// Проверяем, что исходный файл больше не существует
	if _, err := os.Stat(sourceFile); err == nil {
		t.Errorf("Source file %s still exists after move", sourceFile)
	}
	
	// Проверяем содержимое
	content, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}
	
	if string(content) != "test content" {
		t.Errorf("Moved file content = %s, want 'test content'", string(content))
	}
}

// TestMoveDatabaseToUploads_AlreadyInUploads проверяет случай, когда файл уже в uploads
func TestMoveDatabaseToUploads_AlreadyInUploads(t *testing.T) {
	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)
	
	// Создаем файл уже в uploads
	sourceFile := filepath.Join(uploadsDir, "test.db")
	if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	newPath, err := MoveDatabaseToUploads(sourceFile, uploadsDir)
	if err != nil {
		t.Fatalf("MoveDatabaseToUploads() failed: %v", err)
	}
	
	// Должен вернуть тот же путь
	if newPath != sourceFile {
		t.Errorf("MoveDatabaseToUploads() = %s, want %s", newPath, sourceFile)
	}
}

// TestMoveDatabaseToUploads_FileExists проверяет случай, когда файл уже существует в uploads
func TestMoveDatabaseToUploads_FileExists(t *testing.T) {
	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)
	
	// Создаем файл в uploads
	existingFile := filepath.Join(uploadsDir, "test.db")
	if err := os.WriteFile(existingFile, []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}
	
	// Создаем исходный файл
	sourceFile := filepath.Join(tempDir, "test.db")
	if err := os.WriteFile(sourceFile, []byte("new"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	newPath, err := MoveDatabaseToUploads(sourceFile, uploadsDir)
	if err != nil {
		t.Fatalf("MoveDatabaseToUploads() failed: %v", err)
	}
	
	// Должен вернуть путь к существующему файлу
	if newPath != existingFile {
		t.Errorf("MoveDatabaseToUploads() = %s, want %s", newPath, existingFile)
	}
	
	// Проверяем, что содержимое существующего файла не изменилось
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("Failed to read existing file: %v", err)
	}
	
	if string(content) != "existing" {
		t.Errorf("Existing file content = %s, want 'existing'", string(content))
	}
}

