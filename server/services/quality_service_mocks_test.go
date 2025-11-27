package services

import (
	"errors"
	"testing"

	"httpserver/database"
)

// TestMocks_InterfaceCompliance проверяет, что все моки правильно реализуют интерфейсы
func TestMocks_InterfaceCompliance(t *testing.T) {
	t.Run("mockDB implements DatabaseInterface", func(t *testing.T) {
		var _ DatabaseInterface = (*mockDB)(nil)
	})

	t.Run("mockQualityAnalyzer implements QualityAnalyzerInterface", func(t *testing.T) {
		var _ QualityAnalyzerInterface = (*mockQualityAnalyzer)(nil)
	})

	t.Run("mockLogger implements LoggerInterface", func(t *testing.T) {
		var _ LoggerInterface = (*mockLogger)(nil)
	})

	t.Run("mockDBWithQueryError implements DatabaseInterface", func(t *testing.T) {
		var _ DatabaseInterface = (*mockDBWithQueryError)(nil)
	})

	t.Run("mockDBWithCloseError implements DatabaseInterface", func(t *testing.T) {
		var _ DatabaseInterface = (*mockDBWithCloseError)(nil)
	})

	t.Run("mockDatabaseFactory implements DatabaseFactory", func(t *testing.T) {
		var _ DatabaseFactory = (*mockDatabaseFactory)(nil)
	})
}

// TestMocks_MockDB_AllMethods проверяет, что mockDB реализует все методы DatabaseInterface
func TestMocks_MockDB_AllMethods(t *testing.T) {
	mock := &mockDB{}

	// Проверяем, что все методы определены и возвращают ошибку "not implemented" по умолчанию
	t.Run("GetQualityStats", func(t *testing.T) {
		_, err := mock.GetQualityStats()
		if err == nil {
			t.Error("Expected error for unimplemented GetQualityStats")
		}
	})

	t.Run("GetUploadByUUID", func(t *testing.T) {
		_, err := mock.GetUploadByUUID("test")
		if err == nil {
			t.Error("Expected error for unimplemented GetUploadByUUID")
		}
	})

	t.Run("GetQualityMetrics", func(t *testing.T) {
		_, err := mock.GetQualityMetrics(1)
		if err == nil {
			t.Error("Expected error for unimplemented GetQualityMetrics")
		}
	})

	t.Run("GetQualityIssues", func(t *testing.T) {
		_, _, err := mock.GetQualityIssues(1, nil, 0, 0)
		if err == nil {
			t.Error("Expected error for unimplemented GetQualityIssues")
		}
	})

	t.Run("GetAllUploads", func(t *testing.T) {
		_, err := mock.GetAllUploads()
		if err == nil {
			t.Error("Expected error for unimplemented GetAllUploads")
		}
	})

	t.Run("GetQualityTrends", func(t *testing.T) {
		_, err := mock.GetQualityTrends(1, 30)
		if err == nil {
			t.Error("Expected error for unimplemented GetQualityTrends")
		}
	})

	t.Run("GetCurrentQualityMetrics", func(t *testing.T) {
		_, err := mock.GetCurrentQualityMetrics(1)
		if err == nil {
			t.Error("Expected error for unimplemented GetCurrentQualityMetrics")
		}
	})

	t.Run("GetTopQualityIssues", func(t *testing.T) {
		_, err := mock.GetTopQualityIssues(1, 10)
		if err == nil {
			t.Error("Expected error for unimplemented GetTopQualityIssues")
		}
	})

	t.Run("Query", func(t *testing.T) {
		_, err := mock.Query("SELECT 1", nil)
		if err == nil {
			t.Error("Expected error for unimplemented Query")
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := mock.Close()
		if err != nil {
			t.Errorf("Close should return nil, got: %v", err)
		}
	})
}

// TestMocks_MockDB_WithFunctions проверяет, что mockDB правильно использует функции
func TestMocks_MockDB_WithFunctions(t *testing.T) {
	t.Run("GetQualityStats with function", func(t *testing.T) {
		expectedStats := map[string]interface{}{"test": "value"}
		mock := &mockDB{
			getQualityStatsFunc: func() (interface{}, error) {
				return expectedStats, nil
			},
		}

		stats, err := mock.GetQualityStats()
		if err != nil {
			t.Fatalf("GetQualityStats() error = %v", err)
		}
		if stats == nil {
			t.Error("Stats should not be nil")
		}
	})

	t.Run("GetUploadByUUID with function", func(t *testing.T) {
		expectedUpload := &database.Upload{ID: 1, UploadUUID: "test-uuid"}
		mock := &mockDB{
			getUploadByUUIDFunc: func(uuid string) (*database.Upload, error) {
				if uuid != "test-uuid" {
					return nil, errors.New("unexpected UUID")
				}
				return expectedUpload, nil
			},
		}

		upload, err := mock.GetUploadByUUID("test-uuid")
		if err != nil {
			t.Fatalf("GetUploadByUUID() error = %v", err)
		}
		if upload == nil {
			t.Error("Upload should not be nil")
		}
		if upload.UploadUUID != "test-uuid" {
			t.Errorf("Expected UUID 'test-uuid', got '%s'", upload.UploadUUID)
		}
	})
}

// TestMocks_MockQualityAnalyzer проверяет mockQualityAnalyzer
func TestMocks_MockQualityAnalyzer(t *testing.T) {
	t.Run("AnalyzeUpload without function", func(t *testing.T) {
		mock := &mockQualityAnalyzer{}
		err := mock.AnalyzeUpload(1, 1)
		if err != nil {
			t.Errorf("AnalyzeUpload() should return nil when no function is set, got: %v", err)
		}
	})

	t.Run("AnalyzeUpload with function", func(t *testing.T) {
		expectedError := errors.New("test error")
		mock := &mockQualityAnalyzer{
			analyzeUploadFunc: func(uploadID int, databaseID int) error {
				if uploadID != 1 || databaseID != 2 {
					return errors.New("unexpected parameters")
				}
				return expectedError
			},
		}

		err := mock.AnalyzeUpload(1, 2)
		if err != expectedError {
			t.Errorf("Expected error %v, got %v", expectedError, err)
		}
	})
}

// TestMocks_MockLogger проверяет mockLogger
func TestMocks_MockLogger(t *testing.T) {
	mock := &mockLogger{}

	t.Run("Info", func(t *testing.T) {
		mock.Info("test info message")
		if len(mock.infoCalls) != 1 {
			t.Errorf("Expected 1 info call, got %d", len(mock.infoCalls))
		}
		if mock.infoCalls[0] != "test info message" {
			t.Errorf("Expected 'test info message', got '%s'", mock.infoCalls[0])
		}
	})

	t.Run("Error", func(t *testing.T) {
		mock.Error("test error message")
		if len(mock.errorCalls) != 1 {
			t.Errorf("Expected 1 error call, got %d", len(mock.errorCalls))
		}
		if mock.errorCalls[0] != "test error message" {
			t.Errorf("Expected 'test error message', got '%s'", mock.errorCalls[0])
		}
	})

	t.Run("Warn", func(t *testing.T) {
		mock.Warn("test warn message")
		if len(mock.warnCalls) != 1 {
			t.Errorf("Expected 1 warn call, got %d", len(mock.warnCalls))
		}
		if mock.warnCalls[0] != "test warn message" {
			t.Errorf("Expected 'test warn message', got '%s'", mock.warnCalls[0])
		}
	})
}

// TestMocks_MockDatabaseFactory проверяет mockDatabaseFactory
func TestMocks_MockDatabaseFactory(t *testing.T) {
	t.Run("NewDB without function", func(t *testing.T) {
		mock := &mockDatabaseFactory{}
		_, err := mock.NewDB("test.db")
		if err == nil {
			t.Error("Expected error for unimplemented NewDB")
		}
	})

	t.Run("NewDB with function", func(t *testing.T) {
		expectedDB := &mockDB{}
		mock := &mockDatabaseFactory{
			newDBFunc: func(path string) (DatabaseInterface, error) {
				if path != "test.db" {
					return nil, errors.New("unexpected path")
				}
				return expectedDB, nil
			},
		}

		db, err := mock.NewDB("test.db")
		if err != nil {
			t.Fatalf("NewDB() error = %v", err)
		}
		if db == nil {
			t.Error("DB should not be nil")
		}
		if db != expectedDB {
			t.Error("DB should be the expected mock")
		}
	})
}

// TestMocks_MockDBWithQueryError проверяет mockDBWithQueryError
func TestMocks_MockDBWithQueryError(t *testing.T) {
	// Создаем реальную БД для теста
	tempDir := t.TempDir()
	db, err := database.NewDB(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close()

	mock := &mockDBWithQueryError{
		realDB:     db,
		queryError: errors.New("query error"),
	}

	t.Run("Query returns error", func(t *testing.T) {
		_, err := mock.Query("SELECT issue_severity, COUNT(*) FROM data_quality_issues WHERE upload_id = ?", 1)
		if err == nil {
			t.Error("Expected query error")
		}
		if err.Error() != "query error" {
			t.Errorf("Expected 'query error', got '%s'", err.Error())
		}
	})

	t.Run("Other methods delegate to realDB", func(t *testing.T) {
		// Проверяем, что другие методы делегируются к realDB
		_, err := mock.GetAllUploads()
		// Может быть ошибка, но не должна быть "query error"
		if err != nil && err.Error() == "query error" {
			t.Error("GetAllUploads should not return query error")
		}
	})
}

// TestMocks_MockDBWithCloseError проверяет mockDBWithCloseError
func TestMocks_MockDBWithCloseError(t *testing.T) {
	mockDB := &mockDB{
		getQualityStatsFunc: func() (interface{}, error) {
			return map[string]interface{}{"test": "value"}, nil
		},
	}

	mock := &mockDBWithCloseError{
		db: mockDB,
	}

	t.Run("Close returns error", func(t *testing.T) {
		err := mock.Close()
		if err == nil {
			t.Error("Expected close error")
		}
		if err.Error() != "close error" {
			t.Errorf("Expected 'close error', got '%s'", err.Error())
		}
	})

	t.Run("GetQualityStats delegates to db", func(t *testing.T) {
		stats, err := mock.GetQualityStats()
		if err != nil {
			t.Fatalf("GetQualityStats() error = %v", err)
		}
		if stats == nil {
			t.Error("Stats should not be nil")
		}
	})
}

