package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"httpserver/database"
)

// TestValidateIntParam проверяет валидацию целочисленных параметров из query string
func TestValidateIntParam(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		paramName    string
		defaultValue int
		min          int
		max          int
		wantValue    int
		wantErr      bool
	}{
		{
			name:         "valid param",
			query:        "?page=5",
			paramName:    "page",
			defaultValue: 1,
			min:          1,
			max:          100,
			wantValue:    5,
			wantErr:      false,
		},
		{
			name:         "missing param uses default",
			query:        "",
			paramName:    "page",
			defaultValue: 1,
			min:          1,
			max:          100,
			wantValue:    1,
			wantErr:      false,
		},
		{
			name:         "invalid param",
			query:        "?page=abc",
			paramName:    "page",
			defaultValue: 1,
			min:          1,
			max:          100,
			wantValue:    0,
			wantErr:      true,
		},
		{
			name:         "param below min",
			query:        "?page=0",
			paramName:    "page",
			defaultValue: 1,
			min:          1,
			max:          100,
			wantValue:    0,
			wantErr:      true,
		},
		{
			name:         "param above max",
			query:        "?page=200",
			paramName:    "page",
			defaultValue: 1,
			min:          1,
			max:          100,
			wantValue:    0,
			wantErr:      true,
		},
		{
			name:         "no min/max constraints",
			query:        "?page=-5",
			paramName:    "page",
			defaultValue: 1,
			min:          0,
			max:          0,
			wantValue:    -5,
			wantErr:      false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/test"+tt.query, nil)
			value, err := ValidateIntParam(req, tt.paramName, tt.defaultValue, tt.min, tt.max)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIntParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && value != tt.wantValue {
				t.Errorf("ValidateIntParam() = %v, want %v", value, tt.wantValue)
			}
			
			if tt.wantErr {
				validationErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
				} else if validationErr.Field != tt.paramName {
					t.Errorf("ValidationError.Field = %v, want %v", validationErr.Field, tt.paramName)
				}
			}
		})
	}
}

// TestValidateIntPathParam проверяет валидацию целочисленных параметров из path
func TestValidateIntPathParam(t *testing.T) {
	tests := []struct {
		name      string
		paramStr  string
		paramName string
		wantValue int
		wantErr   bool
	}{
		{
			name:      "valid param",
			paramStr:  "123",
			paramName: "id",
			wantValue: 123,
			wantErr:   false,
		},
		{
			name:      "empty param",
			paramStr:  "",
			paramName: "id",
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid param",
			paramStr:  "abc",
			paramName: "id",
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "zero value",
			paramStr:  "0",
			paramName: "id",
			wantValue: 0,
			wantErr:   true,
		},
		{
			name:      "negative value",
			paramStr:  "-5",
			paramName: "id",
			wantValue: 0,
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := ValidateIntPathParam(tt.paramStr, tt.paramName)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIntPathParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && value != tt.wantValue {
				t.Errorf("ValidateIntPathParam() = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

// TestValidateStringParam проверяет валидацию строковых параметров
func TestValidateStringParam(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		paramName string
		required  bool
		maxLength int
		wantErr   bool
	}{
		{
			name:      "valid string",
			value:     "test",
			paramName: "name",
			required:  true,
			maxLength: 100,
			wantErr:   false,
		},
		{
			name:      "empty required",
			value:     "",
			paramName: "name",
			required:  true,
			maxLength: 100,
			wantErr:   true,
		},
		{
			name:      "empty not required",
			value:     "",
			paramName: "name",
			required:  false,
			maxLength: 100,
			wantErr:   false,
		},
		{
			name:      "whitespace only required",
			value:     "   ",
			paramName: "name",
			required:  true,
			maxLength: 100,
			wantErr:   true,
		},
		{
			name:      "exceeds max length",
			value:     "a very long string that exceeds the maximum length",
			paramName: "name",
			required:  true,
			maxLength: 10,
			wantErr:   true,
		},
		{
			name:      "no max length constraint",
			value:     "any length string",
			paramName: "name",
			required:  true,
			maxLength: 0,
			wantErr:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringParam(tt.value, tt.paramName, tt.required, tt.maxLength)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStringParam() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				validationErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
				} else if validationErr.Field != tt.paramName {
					t.Errorf("ValidationError.Field = %v, want %v", validationErr.Field, tt.paramName)
				}
			}
		})
	}
}

// TestValidateSearchQuery проверяет валидацию поискового запроса
func TestValidateSearchQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		maxLength int
		wantErr   bool
	}{
		{
			name:      "valid query",
			query:     "test search",
			maxLength: 100,
			wantErr:   false,
		},
		{
			name:      "empty query",
			query:     "",
			maxLength: 100,
			wantErr:   false,
		},
		{
			name:      "exceeds max length",
			query:     "a very long search query that exceeds the maximum length",
			maxLength: 10,
			wantErr:   true,
		},
		{
			name:      "no max length constraint",
			query:     "any length query",
			maxLength: 0,
			wantErr:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSearchQuery(tt.query, tt.maxLength)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSearchQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidationError проверяет структуру ValidationError
func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}
	
	expected := "validation error: test_field - test message"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
}

// setupTestDB создает тестовые базы данных
func setupTestDB(t *testing.T) (*database.DB, *database.DB, *database.ServiceDB) {
	tempDir := t.TempDir()
	
	dbPath := filepath.Join(tempDir, "test.db")
	normalizedDBPath := filepath.Join(tempDir, "test_normalized.db")
	serviceDBPath := filepath.Join(tempDir, "test_service.db")
	
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	
	normalizedDB, err := database.NewDB(normalizedDBPath)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create test normalized DB: %v", err)
	}
	
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		db.Close()
		normalizedDB.Close()
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	
	return db, normalizedDB, serviceDB
}

// TestHandleValidationError проверяет обработку ошибок валидации
func TestHandleValidationError(t *testing.T) {
	// Создаем тестовый сервер
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
	
	tests := []struct {
		name           string
		err            error
		wantHandled    bool
		wantStatusCode int
	}{
		{
			name:           "validation error",
			err:            &ValidationError{Field: "test", Message: "test error"},
			wantHandled:    true,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "non-validation error",
			err:            &struct{ error }{},
			wantHandled:    false,
			wantStatusCode: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)
			handled := srv.HandleValidationError(w, r, tt.err)
			
			if handled != tt.wantHandled {
				t.Errorf("HandleValidationError() = %v, want %v", handled, tt.wantHandled)
			}
			
			if tt.wantHandled && w.Code != tt.wantStatusCode {
				t.Errorf("HandleValidationError() status code = %v, want %v", w.Code, tt.wantStatusCode)
			}
		})
	}
}

