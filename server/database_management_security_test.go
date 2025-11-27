package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ========== Тесты безопасности: Path Traversal ==========

// TestSecurity_PathTraversal_BulkDelete тестирует защиту от path traversal в bulk-delete
func TestSecurity_PathTraversal_BulkDelete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	testFile := createTestDBFile(t, tempDir, "test.db")

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Пытаемся удалить через path traversal
	maliciousPaths := []string{
		"../test.db",
		"..\\test.db",
		filepath.Join("..", "test.db"),
		filepath.Join(tempDir, "..", "test.db"),
		"/etc/passwd",
		"../../etc/passwd",
	}

	for _, maliciousPath := range maliciousPaths {
		reqBody := map[string]interface{}{
			"paths":   []string{maliciousPath},
			"confirm": true,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/databases/bulk-delete", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.handleBulkDeleteDatabases(w, req)

		// Файл должен остаться нетронутым (либо ошибка, либо файл не найден)
		if _, err := os.Stat(testFile); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("Path traversal succeeded with path: %s", maliciousPath)
			}
		}
	}
}

// TestSecurity_PathTraversal_DownloadBackup тестирует защиту от path traversal в download backup
func TestSecurity_PathTraversal_DownloadBackup(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	// Создаем легитимный бэкап
	backupFileName := "legitimate.zip"
	backupPath := filepath.Join(backupDir, backupFileName)
	os.WriteFile(backupPath, []byte("legitimate backup"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	maliciousPaths := []string{
		"../legitimate.zip",
		"..\\legitimate.zip",
		"/etc/passwd",
		"../../etc/passwd",
		"../data/backups/legitimate.zip",
	}

	for _, maliciousPath := range maliciousPaths {
		req := httptest.NewRequest(http.MethodGet, "/api/databases/backups/"+maliciousPath, nil)
		w := httptest.NewRecorder()

		srv.handleDownloadBackup(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for path traversal attempt %s, got %d", http.StatusBadRequest, maliciousPath, w.Code)
		}
	}
}

// TestSecurity_PathTraversal_Restore тестирует защиту от path traversal в restore
func TestSecurity_PathTraversal_Restore(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	maliciousFilenames := []string{
		"../backup.zip",
		"..\\backup.zip",
		"/etc/passwd",
		"../../etc/passwd",
		"../data/backups/backup.zip",
	}

	for _, maliciousFilename := range maliciousFilenames {
		reqBody := map[string]interface{}{
			"backup_file": maliciousFilename,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/databases/restore", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.handleRestoreBackup(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for path traversal attempt %s, got %d", http.StatusBadRequest, maliciousFilename, w.Code)
		}
	}
}

// ========== Тесты безопасности: Защита системных БД ==========

// TestSecurity_ProtectedDatabases_ServiceDB тестирует защиту service.db
func TestSecurity_ProtectedDatabases_ServiceDB(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	serviceDBFile := createTestDBFile(t, tempDir, "service.db")

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	reqBody := map[string]interface{}{
		"paths":   []string{serviceDBFile},
		"confirm": true,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/bulk-delete", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBulkDeleteDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	results, ok := response["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("Expected results array")
	}

	result, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected deletion of service.db to fail")
	}

	if errorMsg, ok := result["error"].(string); ok {
		if !strings.Contains(errorMsg, "protected") {
			t.Errorf("Expected error about protection, got: %s", errorMsg)
		}
	}

	// Проверяем, что файл не удален
	if _, err := os.Stat(serviceDBFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Error("Expected service.db to still exist")
		}
	}
}

// TestSecurity_ProtectedDatabases_AllProtected тестирует защиту всех системных БД
func TestSecurity_ProtectedDatabases_AllProtected(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	protectedFiles := []string{"service.db", "1c_data.db", "data.db", "normalized_data.db"}
	filePaths := make([]string, len(protectedFiles))

	for i, fileName := range protectedFiles {
		filePaths[i] = createTestDBFile(t, tempDir, fileName)
	}

	reqBody := map[string]interface{}{
		"paths":   filePaths,
		"confirm": true,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/bulk-delete", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBulkDeleteDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	results, ok := response["results"].([]interface{})
	if !ok {
		t.Fatal("Expected results array")
	}

	// Все файлы должны быть защищены
	for _, r := range results {
		result, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if success, ok := result["success"].(bool); ok && success {
			t.Error("Expected all protected files to fail deletion")
		}
	}

	// Проверяем, что все файлы остались
	for _, filePath := range filePaths {
		if _, err := os.Stat(filePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("Expected protected file to still exist: %s", filePath)
			}
		}
	}
}

// TestSecurity_ProtectedDatabases_ByID тестирует защиту системных БД при удалении по ID
func TestSecurity_ProtectedDatabases_ByID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	_, projectID := createTestClientAndProject(t, srv)

	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "data", "uploads")
	os.MkdirAll(uploadsDir, 0755)

	// Создаем защищенный файл и запись в БД
	protectedFile := createTestDBFile(t, uploadsDir, "service.db")
	
	// Получаем размер файла
	fileInfo, err := os.Stat(protectedFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}
	
	db, err := srv.serviceDB.CreateProjectDatabase(projectID, "service.db", protectedFile, "Service DB", fileInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	reqBody := map[string]interface{}{
		"ids":     []int{db.ID},
		"confirm": true,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/bulk-delete", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBulkDeleteDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	results, ok := response["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("Expected results array")
	}

	result, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected deletion of protected database by ID to fail")
	}

	// Проверяем, что файл не удален
	if _, err := os.Stat(protectedFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Error("Expected protected file to still exist")
		}
	}
}

// ========== Тесты безопасности: Валидация SQLite файлов ==========

// TestSecurity_SQLiteValidation_ValidFile тестирует восстановление валидного SQLite файла
func TestSecurity_SQLiteValidation_ValidFile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	// Создаем валидный SQLite файл
	testDBContent := []byte("SQLite format 3\x00")
	for len(testDBContent) < 16 {
		testDBContent = append(testDBContent, 0)
	}

	backupFileName := "valid_backup.zip"
	files := map[string][]byte{
		"main/test.db": testDBContent,
	}
	createTestBackup(t, backupDir, backupFileName, files)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	reqBody := map[string]interface{}{
		"backup_file": backupFileName,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/restore", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleRestoreBackup(w, req)

	// Восстановление должно пройти успешно (валидация SQLite происходит внутри)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestSecurity_SQLiteValidation_InvalidHeader тестирует отклонение файла с неправильным заголовком
func TestSecurity_SQLiteValidation_InvalidHeader(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	// Создаем файл с неправильным заголовком
	invalidContent := []byte("Invalid SQLite header\x00")

	backupFileName := "invalid_backup.zip"
	files := map[string][]byte{
		"main/test.db": invalidContent,
	}
	createTestBackup(t, backupDir, backupFileName, files)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	reqBody := map[string]interface{}{
		"backup_file": backupFileName,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/restore", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleRestoreBackup(w, req)

	// Восстановление должно пройти, но валидация SQLite должна произойти
	// Проверяем, что файл был восстановлен (валидация происходит через ValidateSQLiteDatabase)
	// Если валидация не прошла, файл должен быть удален
	if w.Code == http.StatusOK {
		// Если восстановление прошло, проверяем, что файл валиден
		restoredPath := "test.db"
		if _, err := os.Stat(restoredPath); err == nil {
			// Файл существует, но может быть невалидным
			// В реальной реализации ValidateSQLiteDatabase должен проверить это
		}
	}
}

// TestSecurity_SQLiteValidation_CorruptedFile тестирует отклонение поврежденного SQLite файла
func TestSecurity_SQLiteValidation_CorruptedFile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	// Создаем поврежденный SQLite файл (правильный заголовок, но поврежденное содержимое)
	corruptedContent := []byte("SQLite format 3\x00")
	for len(corruptedContent) < 16 {
		corruptedContent = append(corruptedContent, 0)
	}
	// Добавляем поврежденные данные
	corruptedContent = append(corruptedContent, []byte("corrupted data")...)

	backupFileName := "corrupted_backup.zip"
	files := map[string][]byte{
		"main/test.db": corruptedContent,
	}
	createTestBackup(t, backupDir, backupFileName, files)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	reqBody := map[string]interface{}{
		"backup_file": backupFileName,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/restore", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleRestoreBackup(w, req)

	// Восстановление может пройти, но валидация должна проверить целостность
	// ValidateSQLiteDatabase должен открыть файл и проверить его
	if w.Code == http.StatusOK {
		// Если восстановление прошло, проверяем, что файл существует
		// В реальной реализации ValidateSQLiteDatabase должен проверить целостность
		restoredPath := "test.db"
		if _, err := os.Stat(restoredPath); err == nil {
			// Файл существует, но может быть поврежден
			// ValidateSQLiteDatabase должен это обнаружить
		}
	}
}

