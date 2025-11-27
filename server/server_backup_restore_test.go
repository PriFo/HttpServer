package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestHandleRestoreBackup проверяет восстановление из бэкапа
func TestHandleRestoreBackup(t *testing.T) {
	// Создаем тестовые базы данных
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	normalizedDBPath := filepath.Join(tempDir, "test_normalized.db")

	srv := NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, &Config{
		Port:                  "9999",
		DatabasePath:          dbPath,
		NormalizedDatabasePath: normalizedDBPath,
		ServiceDatabasePath:   ":memory:",
		MaxOpenConns:          25,
		MaxIdleConns:          5,
	})

	// Создаем директорию для бэкапов (используем текущую рабочую директорию, как в коде)
	backupDir := "data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}
	defer func() {
		os.RemoveAll("data") // Очистка после теста
	}()

	// Создаем тестовый ZIP архив с базой данных
	backupFileName := "test_backup.zip"
	backupPath := filepath.Join(backupDir, backupFileName)

	zipFile, err := os.Create(backupPath)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Создаем тестовую базу данных в архиве
	testDBContent := []byte("SQLite format 3\x00" + string(make([]byte, 100)))
	archiveFile, err := zipWriter.Create("main/test.db")
	if err != nil {
		t.Fatalf("Failed to create archive entry: %v", err)
	}
	if _, err := archiveFile.Write(testDBContent); err != nil {
		t.Fatalf("Failed to write to archive: %v", err)
	}

	zipWriter.Close()
	zipFile.Close()

	tests := []struct {
		name           string
		method         string
		backupFile     string
		wantStatusCode int
		wantError      bool
	}{
		{
			name:           "valid restore",
			method:         http.MethodPost,
			backupFile:     backupFileName,
			wantStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			backupFile:     backupFileName,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantError:      true,
		},
		{
			name:           "missing backup file",
			method:         http.MethodPost,
			backupFile:     "",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
		{
			name:           "backup not found",
			method:         http.MethodPost,
			backupFile:     "nonexistent.zip",
			wantStatusCode: http.StatusNotFound,
			wantError:      true,
		},
		{
			name:           "invalid backup filename with path traversal",
			method:         http.MethodPost,
			backupFile:     "../backup.zip",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
		{
			name:           "non-zip file",
			method:         http.MethodPost,
			backupFile:     "backup.txt",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"backup_file": tt.backupFile,
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(tt.method, "/api/backups/restore", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.handleRestoreBackup(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleRestoreBackup() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantError {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if _, ok := response["error"]; !ok {
						t.Errorf("Expected error in response, got: %v", response)
					}
				}
			} else {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if success, ok := response["success"].(bool); !ok || !success {
						t.Errorf("Expected success=true, got: %v", response)
					}
				}
			}
		})
	}
}

// TestBuildSafeTableQuery проверяет безопасное построение SQL запросов
func TestBuildSafeTableQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		tableName string
		want      string
		wantEmpty bool
	}{
		{
			name:      "valid table name",
			query:     "SELECT * FROM %s",
			tableName: "catalog_items",
			want:      "SELECT * FROM catalog_items",
			wantEmpty: false,
		},
		{
			name:      "invalid table name with semicolon",
			query:     "SELECT * FROM %s",
			tableName: "catalog_items; DROP TABLE users;",
			want:      "",
			wantEmpty: true,
		},
		{
			name:      "invalid table name with quotes",
			query:     "SELECT * FROM %s",
			tableName: "catalog_items' OR '1'='1",
			want:      "",
			wantEmpty: true,
		},
		{
			name:      "invalid table name with spaces",
			query:     "SELECT * FROM %s",
			tableName: "catalog items",
			want:      "",
			wantEmpty: true,
		},
		{
			name:      "valid table name with underscore",
			query:     "SELECT COUNT(*) FROM %s",
			tableName: "normalized_data",
			want:      "SELECT COUNT(*) FROM normalized_data",
			wantEmpty: false,
		},
		{
			name:      "empty table name",
			query:     "SELECT * FROM %s",
			tableName: "",
			want:      "",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSafeTableQuery(tt.query, tt.tableName)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("buildSafeTableQuery() = %v, want empty string", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("buildSafeTableQuery() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

// TestHandleListBackups проверяет получение списка бэкапов
func TestHandleListBackups(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	normalizedDBPath := filepath.Join(tempDir, "test_normalized.db")

	srv := NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, &Config{
		Port:                  "9999",
		DatabasePath:          dbPath,
		NormalizedDatabasePath: normalizedDBPath,
		ServiceDatabasePath:   ":memory:",
		MaxOpenConns:          25,
		MaxIdleConns:          5,
	})

	// Создаем директорию для бэкапов
	backupDir := filepath.Join(tempDir, "data", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Создаем тестовые файлы бэкапов
	testFiles := []string{"backup1.zip", "backup2.zip", "not_a_backup.txt"}
	for _, fileName := range testFiles {
		filePath := filepath.Join(backupDir, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		file.Close()
	}

	tests := []struct {
		name           string
		method         string
		wantStatusCode int
		wantCount      int
	}{
		{
			name:           "valid request",
			method:         http.MethodGet,
			wantStatusCode: http.StatusOK,
			wantCount:      2, // Только .zip файлы
		},
		{
			name:           "invalid method",
			method:         http.MethodPost,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantCount:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/backups", nil)
			w := httptest.NewRecorder()

			srv.handleListBackups(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleListBackups() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if backups, ok := response["backups"].([]interface{}); ok {
					if len(backups) != tt.wantCount {
						t.Errorf("handleListBackups() backup count = %v, want %v", len(backups), tt.wantCount)
					}
				}
			}
		})
	}
}

// TestHandleBackupDatabases проверяет создание бэкапа
func TestHandleBackupDatabases(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	normalizedDBPath := filepath.Join(tempDir, "test_normalized.db")

	// Создаем тестовые файлы баз данных
	testDBFile, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB file: %v", err)
	}
	testDBFile.WriteString("SQLite format 3\x00")
	testDBFile.Close()

	srv := NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, &Config{
		Port:                  "9999",
		DatabasePath:          dbPath,
		NormalizedDatabasePath: normalizedDBPath,
		ServiceDatabasePath:   ":memory:",
		MaxOpenConns:          25,
		MaxIdleConns:          5,
	})

	tests := []struct {
		name           string
		method         string
		requestBody    map[string]interface{}
		wantStatusCode int
		wantError      bool
	}{
		{
			name:   "valid backup request",
			method: http.MethodPost,
			requestBody: map[string]interface{}{
				"include_main":    true,
				"include_uploads": false,
				"include_service": false,
				"format":          "zip",
			},
			wantStatusCode: http.StatusOK,
			wantError:      false,
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			requestBody:    nil,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantError:      true,
		},
		{
			name:   "empty request uses defaults",
			method: http.MethodPost,
			requestBody: map[string]interface{}{},
			wantStatusCode: http.StatusOK,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonBody []byte
			var err error
			if tt.requestBody != nil {
				jsonBody, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/backups/create", bytes.NewReader(jsonBody))
			if jsonBody != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			srv.handleBackupDatabases(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleBackupDatabases() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if !tt.wantError && tt.wantStatusCode == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if success, ok := response["success"].(bool); !ok || !success {
						t.Errorf("Expected success=true, got: %v", response)
					}
				}
			}
		})
	}
}

// TestIsValidTableName проверяет валидацию имен таблиц
func TestIsValidTableName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		want      bool
	}{
		{
			name:      "valid table name",
			tableName: "catalog_items",
			want:      true,
		},
		{
			name:      "valid table name with numbers",
			tableName: "table123",
			want:      true,
		},
		{
			name:      "invalid with semicolon",
			tableName: "table; DROP TABLE users;",
			want:      false,
		},
		{
			name:      "invalid with quotes",
			tableName: "table' OR '1'='1",
			want:      false,
		},
		{
			name:      "invalid with spaces",
			tableName: "table name",
			want:      false,
		},
		{
			name:      "invalid with dashes",
			tableName: "table-name",
			want:      false,
		},
		{
			name:      "empty string",
			tableName: "",
			want:      false,
		},
		{
			name:      "valid with underscores",
			tableName: "normalized_data",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTableName(tt.tableName)
			if result != tt.want {
				t.Errorf("isValidTableName(%q) = %v, want %v", tt.tableName, result, tt.want)
			}
		})
	}
}

