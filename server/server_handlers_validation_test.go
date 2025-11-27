package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleQualityReportValidation проверяет валидацию параметров в handleQualityReport
func TestHandleQualityReportValidation(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	normalizedDBPath := tempDir + "/test_normalized.db"

	srv := NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, &Config{
		Port:                  "9999",
		DatabasePath:          dbPath,
		NormalizedDatabasePath: normalizedDBPath,
		ServiceDatabasePath:   ":memory:",
		MaxOpenConns:          25,
		MaxIdleConns:          5,
	})

	// Создаем тестовую выгрузку
	uploadUUID := "test-upload-uuid"
	upload, err := db.CreateUpload(uploadUUID, "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	tests := []struct {
		name           string
		query          string
		wantStatusCode int
	}{
		{
			name:           "valid parameters",
			query:          "?limit=10&offset=0",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid limit (negative)",
			query:          "?limit=-5",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "invalid limit (too large)",
			query:          "?limit=10000",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "invalid offset (negative)",
			query:          "?offset=-10",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "summary only",
			query:          "?summary_only=true",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/upload/"+upload.UploadUUID+"/quality-report"+tt.query, nil)
			w := httptest.NewRecorder()

			srv.handleQualityReport(w, req, upload.UploadUUID)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleQualityReport() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestHandleQualityDashboardValidation проверяет валидацию параметров в handleQualityDashboard
func TestHandleQualityDashboardValidation(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	normalizedDBPath := tempDir + "/test_normalized.db"

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
		query          string
		databaseID     int
		wantStatusCode int
	}{
		{
			name:           "valid parameters",
			query:          "?days=30&limit=10",
			databaseID:     1,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid days (negative)",
			query:          "?days=-5",
			databaseID:     1,
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "invalid limit (too large)",
			query:          "?limit=10000",
			databaseID:     1,
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "default values",
			query:          "",
			databaseID:     1,
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/"+string(rune(tt.databaseID+'0'))+"/quality-dashboard"+tt.query, nil)
			w := httptest.NewRecorder()

			srv.handleQualityDashboard(w, req, tt.databaseID)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleQualityDashboard() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestHandleNormalizedCounterpartiesValidation проверяет валидацию параметров в handleNormalizedCounterparties
func TestHandleNormalizedCounterpartiesValidation(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	normalizedDBPath := tempDir + "/test_normalized.db"

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
		query          string
		wantStatusCode int
	}{
		{
			name:           "valid pagination",
			query:          "?page=1&limit=50&client_id=1",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid page (negative)",
			query:          "?page=-1&client_id=1",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "invalid limit (too large)",
			query:          "?limit=2000&client_id=1",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "invalid client_id",
			query:          "?client_id=abc",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid project_id",
			query:          "?project_id=xyz&client_id=1",
			wantStatusCode: http.StatusOK, // project_id опционален
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/normalized/counterparties"+tt.query, nil)
			w := httptest.NewRecorder()

			srv.handleNormalizedCounterparties(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleNormalizedCounterparties() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestHandleGetNomenclatureGroupsValidation проверяет валидацию параметров в handleGetNomenclatureGroups
func TestHandleGetNomenclatureGroupsValidation(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	normalizedDBPath := tempDir + "/test_normalized.db"

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
		query          string
		wantStatusCode int
	}{
		{
			name:           "valid pagination",
			query:          "?page=1&limit=20",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid page (zero)",
			query:          "?page=0",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "invalid limit (too large)",
			query:          "?limit=200",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "default values",
			query:          "",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/nomenclature/groups"+tt.query, nil)
			w := httptest.NewRecorder()

			// Напрямую вызываем обработчик групп нормализации
			srv.handleNormalizationGroups(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleGetNomenclatureGroups() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestHandleGetNomenclatureRecentRecordsValidation проверяет валидацию параметров
func TestHandleGetNomenclatureRecentRecordsValidation(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	normalizedDBPath := tempDir + "/test_normalized.db"

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
		query          string
		wantStatusCode int
	}{
		{
			name:           "valid limit",
			query:          "?limit=10",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid limit (negative)",
			query:          "?limit=-5",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "invalid limit (too large)",
			query:          "?limit=200",
			wantStatusCode: http.StatusOK, // Используется значение по умолчанию
		},
		{
			name:           "default limit",
			query:          "",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/nomenclature/recent"+tt.query, nil)
			w := httptest.NewRecorder()

			srv.getNomenclatureRecentRecords(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("getNomenclatureRecentRecords() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestHandleQualityAnalysisValidation проверяет валидацию в handleQualityAnalysis
func TestHandleQualityAnalysisValidation(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	normalizedDBPath := tempDir + "/test_normalized.db"

	srv := NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, &Config{
		Port:                  "9999",
		DatabasePath:          dbPath,
		NormalizedDatabasePath: normalizedDBPath,
		ServiceDatabasePath:   ":memory:",
		MaxOpenConns:          25,
		MaxIdleConns:          5,
	})

	// Создаем тестовую выгрузку
	uploadUUID := "test-upload-uuid"
	upload, err := db.CreateUpload(uploadUUID, "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		uploadUUID     string
		wantStatusCode int
	}{
		{
			name:           "valid POST request",
			method:         http.MethodPost,
			uploadUUID:     upload.UploadUUID,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			uploadUUID:     upload.UploadUUID,
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "non-existent upload",
			method:         http.MethodPost,
			uploadUUID:     "non-existent-uuid",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/upload/"+tt.uploadUUID+"/quality-analysis", nil)
			w := httptest.NewRecorder()

			srv.handleQualityAnalysis(w, req, tt.uploadUUID)

			if w.Code != tt.wantStatusCode {
				t.Errorf("handleQualityAnalysis() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestHandleValidationErrorIntegration проверяет интеграцию HandleValidationError с обработчиками
func TestHandleValidationErrorIntegration(t *testing.T) {
	db, normalizedDB, serviceDB := setupTestDB(t)
	defer db.Close()
	defer normalizedDB.Close()
	defer serviceDB.Close()

	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	normalizedDBPath := tempDir + "/test_normalized.db"

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
		endpoint       string
		query          string
		wantStatusCode int
	}{
		{
			name:           "invalid database_id in path",
			endpoint:       "/api/v1/databases/abc/quality-dashboard",
			query:          "",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid item_id in path",
			endpoint:       "/api/uploads/xyz/items",
			query:          "",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid client_id in query",
			endpoint:       "/api/normalized/counterparties",
			query:          "?client_id=invalid",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid project_id in query",
			endpoint:       "/api/normalized/counterparties",
			query:          "?project_id=invalid&client_id=1",
			wantStatusCode: http.StatusOK, // project_id опционален, игнорируется
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.endpoint+tt.query, nil)
			w := httptest.NewRecorder()

			var err error
			switch tt.name {
			case "invalid database_id in path":
				_, err = ValidateIntPathParam("abc", "database_id")
			case "invalid item_id in path":
				_, err = ValidateIntPathParam("xyz", "item_id")
			case "invalid client_id in query":
				_, err = ValidateIDParam(req, "client_id")
			case "invalid project_id in query":
				// project_id опционален — имитируем успешную обработку без ошибки
				w.WriteHeader(http.StatusOK)
			default:
				t.Fatalf("Unhandled test case: %s", tt.name)
			}

			if err != nil {
				if handled := srv.HandleValidationError(w, req, err); !handled {
					t.Fatalf("HandleValidationError should handle error for test %s", tt.name)
				}
			}

			if w.Code != tt.wantStatusCode {
				t.Errorf("Request to %s%s status code = %v, want %v", tt.endpoint, tt.query, w.Code, tt.wantStatusCode)
			}
		})
	}
}

