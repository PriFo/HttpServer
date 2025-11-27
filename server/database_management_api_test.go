package server

import (
	"archive/zip"
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

// createTestDBFile создает валидный SQLite файл для тестирования
func createTestDBFile(t *testing.T, dir string, fileName string) string {
	testFileContent := []byte("SQLite format 3\x00")
	for len(testFileContent) < 16 {
		testFileContent = append(testFileContent, 0)
	}
	filePath := filepath.Join(dir, fileName)
	err := os.WriteFile(filePath, testFileContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// TestHandleDatabasesFiles_Success тестирует успешное получение списка файлов
func TestHandleDatabasesFiles_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	
	// Создаем тестовые файлы в разных директориях
	dataDir := filepath.Join(tempDir, "data")
	uploadsDir := filepath.Join(tempDir, "data", "uploads")
	os.MkdirAll(uploadsDir, 0755)
	
	createTestDBFile(t, tempDir, "main1.db")
	createTestDBFile(t, dataDir, "main2.db")
	createTestDBFile(t, uploadsDir, "upload1.db")
	createTestDBFile(t, tempDir, "service.db")
	
	// Меняем рабочую директорию на tempDir для теста
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/files", nil)
	w := httptest.NewRecorder()

	srv.handleDatabasesFiles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	if total, ok := response["total"].(float64); !ok || total < 4 {
		t.Errorf("Expected at least 4 files, got %v", response["total"])
	}

	files, ok := response["files"].([]interface{})
	if !ok {
		t.Fatal("Expected files array in response")
	}

	if len(files) < 4 {
		t.Errorf("Expected at least 4 files, got %d", len(files))
	}
}

// TestHandleDatabasesFiles_Grouping тестирует группировку по типам
func TestHandleDatabasesFiles_Grouping(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	
	dataDir := filepath.Join(tempDir, "data")
	uploadsDir := filepath.Join(tempDir, "data", "uploads")
	os.MkdirAll(uploadsDir, 0755)
	
	createTestDBFile(t, tempDir, "main1.db")
	createTestDBFile(t, dataDir, "main2.db")
	createTestDBFile(t, uploadsDir, "upload1.db")
	createTestDBFile(t, tempDir, "service.db")
	
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/files", nil)
	w := httptest.NewRecorder()

	srv.handleDatabasesFiles(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	grouped, ok := response["grouped"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected grouped map in response")
	}

	// Проверяем наличие всех типов
	expectedTypes := []string{"main", "service", "uploaded", "other"}
	for _, typ := range expectedTypes {
		if _, exists := grouped[typ]; !exists {
			t.Errorf("Expected type %s in grouped response", typ)
		}
	}

	summary, ok := response["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected summary map in response")
	}

	// Проверяем, что summary содержит правильные типы
	for _, typ := range expectedTypes {
		if _, exists := summary[typ]; !exists {
			t.Errorf("Expected type %s in summary", typ)
		}
	}
}

// TestHandleDatabasesFiles_Metadata тестирует метаданные файлов
func TestHandleDatabasesFiles_Metadata(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	createTestDBFile(t, tempDir, "test.db")
	
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/files", nil)
	w := httptest.NewRecorder()

	srv.handleDatabasesFiles(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	files, ok := response["files"].([]interface{})
	if !ok || len(files) == 0 {
		t.Fatal("Expected at least one file in response")
	}

	fileInfo, ok := files[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected file info to be a map")
	}

	// Проверяем наличие обязательных полей
	requiredFields := []string{"path", "name", "size", "modified_at", "type", "is_protected", "linked_to_project"}
	for _, field := range requiredFields {
		if _, exists := fileInfo[field]; !exists {
			t.Errorf("Expected field %s in file info", field)
		}
	}

	// Проверяем, что path соответствует созданному файлу
	if path, ok := fileInfo["path"].(string); ok {
		if !strings.Contains(path, "test.db") {
			t.Errorf("Expected path to contain test.db, got %s", path)
		}
	}

	// Проверяем размер файла
	if size, ok := fileInfo["size"].(float64); ok {
		if size < 16 {
			t.Errorf("Expected file size >= 16, got %v", size)
		}
	}
}

// TestHandleDatabasesFiles_LinkedToProject тестирует связь с проектами
func TestHandleDatabasesFiles_LinkedToProject(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	clientID, projectID := createTestClientAndProject(t, srv)

	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "data", "uploads")
	os.MkdirAll(uploadsDir, 0755)
	
	testFile := createTestDBFile(t, uploadsDir, "linked.db")
	
	// Получаем размер файла
	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}
	
	// Создаем запись в project_databases
	_, err = srv.serviceDB.CreateProjectDatabase(projectID, "linked.db", testFile, "Test DB", fileInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}
	
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/files", nil)
	w := httptest.NewRecorder()

	srv.handleDatabasesFiles(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	files, ok := response["files"].([]interface{})
	if !ok {
		t.Fatal("Expected files array in response")
	}

	// Ищем наш файл
	found := false
	for _, f := range files {
		fileInfo, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		
		if name, ok := fileInfo["name"].(string); ok && name == "linked.db" {
			found = true
			if linked, ok := fileInfo["linked_to_project"].(bool); !ok || !linked {
				t.Error("Expected linked_to_project=true for linked.db")
			}
			if pid, ok := fileInfo["project_id"].(float64); !ok || int(pid) != projectID {
				t.Errorf("Expected project_id=%d, got %v", projectID, pid)
			}
			if cid, ok := fileInfo["client_id"].(float64); !ok || int(cid) != clientID {
				t.Errorf("Expected client_id=%d, got %v", clientID, cid)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find linked.db in response")
	}
}

// TestHandleDatabasesFiles_ProtectedFiles тестирует фильтрацию защищенных файлов
func TestHandleDatabasesFiles_ProtectedFiles(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	
	protectedFiles := []string{"service.db", "1c_data.db", "data.db", "normalized_data.db"}
	for _, fileName := range protectedFiles {
		createTestDBFile(t, tempDir, fileName)
	}
	
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/files", nil)
	w := httptest.NewRecorder()

	srv.handleDatabasesFiles(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	files, ok := response["files"].([]interface{})
	if !ok {
		t.Fatal("Expected files array in response")
	}

	// Проверяем, что все защищенные файлы помечены как protected
	for _, fileName := range protectedFiles {
		found := false
		for _, f := range files {
			fileInfo, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			
			if name, ok := fileInfo["name"].(string); ok && name == fileName {
				found = true
				if protected, ok := fileInfo["is_protected"].(bool); !ok || !protected {
					t.Errorf("Expected is_protected=true for %s", fileName)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in response", fileName)
		}
	}
}

// TestHandleDatabasesFiles_NonexistentDirectories тестирует обработку несуществующих директорий
func TestHandleDatabasesFiles_NonexistentDirectories(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Используем временную директорию без создания поддиректорий
	tempDir := t.TempDir()
	
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/files", nil)
	w := httptest.NewRecorder()

	srv.handleDatabasesFiles(w, req)

	// Должен вернуть успешный ответ даже если директорий нет
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	// Должен вернуть пустой список или список с файлами из текущей директории
	if total, ok := response["total"].(float64); ok {
		if total < 0 {
			t.Errorf("Expected total >= 0, got %v", total)
		}
	}
}

// TestHandleDatabasesFiles_InvalidMethod тестирует обработку неправильного метода
func TestHandleDatabasesFiles_InvalidMethod(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/databases/files", nil)
	w := httptest.NewRecorder()

	srv.handleDatabasesFiles(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// ========== Тесты для POST /api/databases/bulk-delete ==========

// TestHandleBulkDeleteDatabases_SuccessByPath тестирует успешное удаление по путям
func TestHandleBulkDeleteDatabases_SuccessByPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Создаем тестовые файлы
	file1 := createTestDBFile(t, tempDir, "test1.db")
	file2 := createTestDBFile(t, tempDir, "test2.db")

	reqBody := map[string]interface{}{
		"paths":   []string{file1, file2},
		"confirm": true,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/bulk-delete", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBulkDeleteDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	// Проверяем, что файлы действительно удалены
	if _, err := os.Stat(file1); err == nil {
		t.Error("Expected file1 to be deleted")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected file1 to be deleted, but got error: %v", err)
	}
	if _, err := os.Stat(file2); err == nil {
		t.Error("Expected file2 to be deleted")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected file2 to be deleted, but got error: %v", err)
	}
}

// TestHandleBulkDeleteDatabases_SuccessByID тестирует успешное удаление по ID
func TestHandleBulkDeleteDatabases_SuccessByID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	_, projectID := createTestClientAndProject(t, srv)

	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "data", "uploads")
	os.MkdirAll(uploadsDir, 0755)
	
	testFile := createTestDBFile(t, uploadsDir, "test.db")
	
	// Получаем размер файла
	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}
	
	// Создаем запись в project_databases
	db, err := srv.serviceDB.CreateProjectDatabase(projectID, "test.db", testFile, "Test DB", fileInfo.Size())
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
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Проверяем, что файл удален
	if _, err := os.Stat(testFile); err == nil {
		t.Error("Expected file to be deleted")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected file to be deleted, but got error: %v", err)
	}

	// Проверяем, что запись удалена из БД
	_, err = srv.serviceDB.GetProjectDatabase(db.ID)
	if err == nil {
		t.Error("Expected database record to be deleted")
	}
}

// TestHandleBulkDeleteDatabases_ProtectedFiles тестирует защиту системных БД
func TestHandleBulkDeleteDatabases_ProtectedFiles(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	protectedFiles := []string{"service.db", "1c_data.db", "data.db", "normalized_data.db"}
	for _, fileName := range protectedFiles {
		createTestDBFile(t, tempDir, fileName)
	}

	reqBody := map[string]interface{}{
		"paths":   protectedFiles,
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
		t.Fatal("Expected results array in response")
	}

	// Проверяем, что все файлы защищены от удаления
	for _, r := range results {
		result, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if success, ok := result["success"].(bool); ok && success {
			t.Error("Expected all protected files to fail deletion")
		}
		if errorMsg, ok := result["error"].(string); ok {
			if !strings.Contains(errorMsg, "protected") {
				t.Errorf("Expected error message about protection, got: %s", errorMsg)
			}
		}
	}

	// Проверяем, что файлы не удалены
	for _, fileName := range protectedFiles {
		filePath := filepath.Join(tempDir, fileName)
		if _, err := os.Stat(filePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("Expected protected file %s to still exist", fileName)
			}
		}
	}
}

// TestHandleBulkDeleteDatabases_RequireConfirm тестирует требование confirm=true
func TestHandleBulkDeleteDatabases_RequireConfirm(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	testFile := createTestDBFile(t, tempDir, "test.db")

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	reqBody := map[string]interface{}{
		"paths":   []string{testFile},
		"confirm": false,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/bulk-delete", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBulkDeleteDatabases(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); ok {
		if !strings.Contains(errorMsg, "confirm") {
			t.Errorf("Expected error about confirm requirement, got: %s", errorMsg)
		}
	}

	// Файл не должен быть удален
	if _, err := os.Stat(testFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Error("Expected file to not be deleted without confirm=true")
		}
	}
}

// TestHandleBulkDeleteDatabases_NonexistentFile тестирует удаление несуществующего файла
func TestHandleBulkDeleteDatabases_NonexistentFile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := map[string]interface{}{
		"paths":   []string{"/nonexistent/path/file.db"},
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
		t.Fatal("Expected results array with at least one result")
	}

	result, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if success, ok := result["success"].(bool); ok && success {
		t.Error("Expected deletion to fail for nonexistent file")
	}

	if errorMsg, ok := result["error"].(string); ok {
		if !strings.Contains(errorMsg, "exist") {
			t.Errorf("Expected error about file not existing, got: %s", errorMsg)
		}
	}
}

// TestHandleBulkDeleteDatabases_PartialSuccess тестирует частичный успех/неудачу
func TestHandleBulkDeleteDatabases_PartialSuccess(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	file1 := createTestDBFile(t, tempDir, "test1.db")
	file2 := createTestDBFile(t, tempDir, "service.db") // защищенный файл
	nonexistent := filepath.Join(tempDir, "nonexistent.db")

	reqBody := map[string]interface{}{
		"paths":   []string{file1, file2, nonexistent},
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

	// success должен быть false, так как не все файлы удалены
	if success, ok := response["success"].(bool); ok && success {
		t.Error("Expected success=false for partial failure")
	}

	successCount, ok := response["success_count"].(float64)
	if !ok || successCount != 1 {
		t.Errorf("Expected success_count=1, got %v", successCount)
	}

	failedCount, ok := response["failed_count"].(float64)
	if !ok || failedCount != 2 {
		t.Errorf("Expected failed_count=2, got %v", failedCount)
	}

	// Проверяем, что file1 удален, а file2 остался
	if _, err := os.Stat(file1); err == nil {
		t.Error("Expected file1 to be deleted")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected file1 to be deleted, but got error: %v", err)
	}
	if _, err := os.Stat(file2); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Error("Expected protected file2 to not be deleted")
		}
	}
}

// ========== Тесты для POST /api/databases/backup ==========

// TestHandleBackupDatabases_ZIP тестирует создание ZIP архива
func TestHandleBackupDatabases_ZIP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Создаем тестовые файлы
	createTestDBFile(t, tempDir, "test1.db")
	dataDir := filepath.Join(tempDir, "data")
	os.MkdirAll(dataDir, 0755)
	createTestDBFile(t, dataDir, "test2.db")

	reqBody := map[string]interface{}{
		"include_main":    true,
		"include_uploads":  false,
		"include_service": false,
		"format":          "zip",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/backup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBackupDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	backup, ok := response["backup"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected backup info in response")
	}

	if backupFile, ok := backup["backup_file"].(string); ok {
		if !strings.HasSuffix(backupFile, ".zip") {
			t.Errorf("Expected backup file to be .zip, got %s", backupFile)
		}
		// Проверяем, что файл создан
		backupPath := filepath.Join("data/backups", backupFile)
		if _, err := os.Stat(backupPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("Expected backup file to exist: %s", backupPath)
			} else {
				t.Errorf("Error checking backup file %s: %v", backupPath, err)
			}
		}
	}
}

// TestHandleBackupDatabases_Copy тестирует создание копий файлов
func TestHandleBackupDatabases_Copy(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	createTestDBFile(t, tempDir, "test.db")

	reqBody := map[string]interface{}{
		"include_main":    true,
		"include_uploads":  false,
		"format":          "copy",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/backup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBackupDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	backup, ok := response["backup"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected backup info in response")
	}

	if copyDir, ok := backup["files_copy_dir"].(string); ok {
		if _, err := os.Stat(copyDir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("Expected copy directory to exist: %s", copyDir)
			} else {
				t.Errorf("Error checking copy directory %s: %v", copyDir, err)
			}
		}
	}
}

// TestHandleBackupDatabases_Both тестирует формат "both" (ZIP + копии)
func TestHandleBackupDatabases_Both(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	createTestDBFile(t, tempDir, "test.db")

	reqBody := map[string]interface{}{
		"include_main": true,
		"format":       "both",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/backup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBackupDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	backup, ok := response["backup"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected backup info in response")
	}

	// Проверяем наличие и ZIP, и копий
	if _, hasZip := backup["backup_file"]; !hasZip {
		t.Error("Expected backup_file in response for format 'both'")
	}
	if _, hasCopy := backup["files_copy_dir"]; !hasCopy {
		t.Error("Expected files_copy_dir in response for format 'both'")
	}
}

// TestHandleBackupDatabases_SelectedFiles тестирует выборочный бэкап
func TestHandleBackupDatabases_SelectedFiles(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	file1 := createTestDBFile(t, tempDir, "test1.db")
	file2 := createTestDBFile(t, tempDir, "test2.db")

	reqBody := map[string]interface{}{
		"selected_files": []string{file1, file2},
		"format":         "zip",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/backup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBackupDatabases(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	backup, ok := response["backup"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected backup info in response")
	}

	if filesCount, ok := backup["files_count"].(float64); ok {
		if filesCount != 2 {
			t.Errorf("Expected files_count=2, got %v", filesCount)
		}
	}
}

// TestHandleBackupDatabases_EmptyList тестирует обработку пустого списка файлов
func TestHandleBackupDatabases_EmptyList(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Не создаем файлы, чтобы список был пустым
	reqBody := map[string]interface{}{
		"include_main":    false,
		"include_uploads":  false,
		"include_service": false,
		"format":          "zip",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/backup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleBackupDatabases(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); ok {
		if !strings.Contains(errorMsg, "No files") {
			t.Errorf("Expected error about no files, got: %s", errorMsg)
		}
	}
}

// ========== Тесты для GET /api/databases/backups ==========

// TestHandleListBackups_Success тестирует получение списка бэкапов
func TestHandleListBackups_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	// Создаем тестовые файлы бэкапов
	testFiles := []string{"backup1.zip", "backup2.zip", "not_a_backup.txt"}
	for _, fileName := range testFiles {
		filePath := filepath.Join(backupDir, fileName)
		os.WriteFile(filePath, []byte("test content"), 0644)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/backups", nil)
	w := httptest.NewRecorder()

	srv.handleListBackups(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	backups, ok := response["backups"].([]interface{})
	if !ok {
		t.Fatal("Expected backups array in response")
	}

	// Должны быть только .zip файлы
	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(backups))
	}

	// Проверяем метаданные
	for _, b := range backups {
		backup, ok := b.(map[string]interface{})
		if !ok {
			t.Fatal("Expected backup to be a map")
		}

		requiredFields := []string{"filename", "size", "modified_at"}
		for _, field := range requiredFields {
			if _, exists := backup[field]; !exists {
				t.Errorf("Expected field %s in backup info", field)
			}
		}
	}
}

// TestHandleListBackups_EmptyDirectory тестирует обработку пустой директории
func TestHandleListBackups_EmptyDirectory(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/backups", nil)
	w := httptest.NewRecorder()

	srv.handleListBackups(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	backups, ok := response["backups"].([]interface{})
	if !ok {
		t.Fatal("Expected backups array in response")
	}

	if len(backups) != 0 {
		t.Errorf("Expected empty backups list, got %d", len(backups))
	}
}

// ========== Тесты для GET /api/databases/backups/{filename} ==========

// TestHandleDownloadBackup_Success тестирует успешное скачивание бэкапа
func TestHandleDownloadBackup_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	backupFileName := "test_backup.zip"
	backupPath := filepath.Join(backupDir, backupFileName)
	testContent := []byte("test backup content")
	os.WriteFile(backupPath, testContent, 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/backups/"+backupFileName, nil)
	w := httptest.NewRecorder()

	srv.handleDownloadBackup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Проверяем заголовки
	if contentType := w.Header().Get("Content-Type"); contentType != "application/zip" {
		t.Errorf("Expected Content-Type=application/zip, got %s", contentType)
	}

	if contentDisposition := w.Header().Get("Content-Disposition"); !strings.Contains(contentDisposition, backupFileName) {
		t.Errorf("Expected Content-Disposition to contain filename, got %s", contentDisposition)
	}

	// Проверяем содержимое
	if !bytes.Equal(w.Body.Bytes(), testContent) {
		t.Error("Expected response body to match backup file content")
	}
}

// TestHandleDownloadBackup_PathTraversal тестирует защиту от path traversal
func TestHandleDownloadBackup_PathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	maliciousPaths := []string{
		"../backup.zip",
		"..\\backup.zip",
		"/etc/passwd",
		"../../etc/passwd",
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

// TestHandleDownloadBackup_NotFound тестирует обработку несуществующего файла
func TestHandleDownloadBackup_NotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	req := httptest.NewRequest(http.MethodGet, "/api/databases/backups/nonexistent.zip", nil)
	w := httptest.NewRecorder()

	srv.handleDownloadBackup(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// ========== Тесты для POST /api/databases/restore ==========

// createTestBackup создает тестовый ZIP архив с базой данных
func createTestBackup(t *testing.T, backupDir string, backupFileName string, files map[string][]byte) string {
	backupPath := filepath.Join(backupDir, backupFileName)
	
	zipFile, err := os.Create(backupPath)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for archivePath, content := range files {
		archiveFile, err := zipWriter.Create(archivePath)
		if err != nil {
			t.Fatalf("Failed to create archive entry: %v", err)
		}
		if _, err := archiveFile.Write(content); err != nil {
			t.Fatalf("Failed to write to archive: %v", err)
		}
	}

	return backupPath
}

// TestHandleRestoreBackup_Success тестирует успешное восстановление из ZIP
func TestHandleRestoreBackup_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	// Создаем валидный SQLite файл для архива
	testDBContent := []byte("SQLite format 3\x00")
	for len(testDBContent) < 16 {
		testDBContent = append(testDBContent, 0)
	}

	backupFileName := "test_backup.zip"
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

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	// Проверяем, что файл восстановлен
	if _, err := os.Stat("test.db"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Error("Expected restored file to exist")
		} else {
			t.Errorf("Error checking restored file: %v", err)
		}
	}
}

// TestHandleRestoreBackup_PathTraversal тестирует защиту от path traversal
func TestHandleRestoreBackup_PathTraversal(t *testing.T) {
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

// TestHandleRestoreBackup_NotFound тестирует обработку несуществующего бэкапа
func TestHandleRestoreBackup_NotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	reqBody := map[string]interface{}{
		"backup_file": "nonexistent.zip",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/restore", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleRestoreBackup(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestHandleRestoreBackup_NonZIPFile тестирует обработку не-ZIP файлов
func TestHandleRestoreBackup_NonZIPFile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	// Создаем файл с неправильным расширением
	backupFileName := "backup.txt"
	backupPath := filepath.Join(backupDir, backupFileName)
	os.WriteFile(backupPath, []byte("not a zip"), 0644)

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

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); ok {
		if !strings.Contains(errorMsg, "ZIP") {
			t.Errorf("Expected error about ZIP format, got: %s", errorMsg)
		}
	}
}

// TestHandleRestoreBackup_UploadsDirectory тестирует восстановление в uploads директорию
func TestHandleRestoreBackup_UploadsDirectory(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "data", "backups")
	os.MkdirAll(backupDir, 0755)

	testDBContent := []byte("SQLite format 3\x00")
	for len(testDBContent) < 16 {
		testDBContent = append(testDBContent, 0)
	}

	backupFileName := "test_backup.zip"
	files := map[string][]byte{
		"uploads/test.db": testDBContent,
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

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Проверяем, что файл восстановлен в правильную директорию
	restoredPath := filepath.Join("data", "uploads", "test.db")
	if _, err := os.Stat(restoredPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Errorf("Expected restored file to exist at %s", restoredPath)
		} else {
			t.Errorf("Error checking restored file at %s: %v", restoredPath, err)
		}
	}
}

