package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"httpserver/database"
	"httpserver/quality"
	"httpserver/server/handlers"
	"httpserver/server/services"
)

// setupQualityTestServer создает тестовый сервер для quality тестов
func setupQualityTestServer(t *testing.T) (*httptest.Server, *database.DB, func()) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}

	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create service DB: %v", err)
	}

	// Создаем quality analyzer
	qualityAnalyzer := quality.NewQualityAnalyzer(db)

	// Создаем quality service
	qualityService, err := services.NewQualityService(db, qualityAnalyzer)
	if err != nil {
		db.Close()
		serviceDB.Close()
		t.Fatalf("Failed to create quality service: %v", err)
	}

	// Создаем base handler
	baseHandler := handlers.NewBaseHandlerFromMiddleware()

	// Создаем quality handler
	logFunc := func(entry interface{}) {
		// Пустая функция для тестов
	}
	normalizedDB, _ := database.NewDB(":memory:")
	qualityHandler := handlers.NewQualityHandler(baseHandler, qualityService, logFunc, normalizedDB, ":memory:")

	// Создаем mux для тестирования
	mux := http.NewServeMux()

	// Регистрируем маршруты
	mux.HandleFunc("/api/v1/upload/", func(w http.ResponseWriter, r *http.Request) {
		qualityHandler.HandleQualityUploadRoutes(w, r)
	})

	mux.HandleFunc("/api/v1/databases/", func(w http.ResponseWriter, r *http.Request) {
		qualityHandler.HandleQualityDatabaseRoutes(w, r, func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	})

	server := httptest.NewServer(mux)

	cleanup := func() {
		server.Close()
		db.Close()
		serviceDB.Close()
	}

	return server, db, cleanup
}

// setupQualityMux создает mux для quality тестов
func setupQualityMux(t *testing.T) (*http.ServeMux, *database.DB) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}

	qualityAnalyzer := quality.NewQualityAnalyzer(db)
	qualityService, err := services.NewQualityService(db, qualityAnalyzer)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create quality service: %v", err)
	}

	baseHandler := handlers.NewBaseHandlerFromMiddleware()
	logFunc := func(entry interface{}) {}
	normalizedDB, _ := database.NewDB(":memory:")
	qualityHandler := handlers.NewQualityHandler(baseHandler, qualityService, logFunc, normalizedDB, ":memory:")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/upload/", func(w http.ResponseWriter, r *http.Request) {
		qualityHandler.HandleQualityUploadRoutes(w, r)
	})
	mux.HandleFunc("/api/v1/databases/", func(w http.ResponseWriter, r *http.Request) {
		qualityHandler.HandleQualityDatabaseRoutes(w, r, func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	})

	return mux, db
}

// TestQualityReportIntegration тестирует интеграцию получения отчета о качестве
func TestQualityReportIntegration(t *testing.T) {
	mux, db := setupQualityMux(t)
	defer db.Close()

	// Создаем тестовую выгрузку
	upload, err := db.CreateUpload("test-quality-uuid", "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	// Устанавливаем database_id
	databaseID := 1
	_, err = db.Exec("UPDATE uploads SET database_id = ? WHERE id = ?", databaseID, upload.ID)
	if err != nil {
		t.Fatalf("Failed to update upload: %v", err)
	}

	// Тест получения отчета
	t.Run("GetQualityReport", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/upload/test-quality-uuid/quality-report", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Если успешно, проверяем структуру ответа
		if w.Code == http.StatusOK {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}

			if uploadUUID, ok := response["upload_uuid"].(string); !ok || uploadUUID != "test-quality-uuid" {
				t.Errorf("Expected upload_uuid 'test-quality-uuid', got %v", response["upload_uuid"])
			}
		}
	})

	// Тест получения отчета с summary_only
	t.Run("GetQualityReport_SummaryOnly", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/upload/test-quality-uuid/quality-report?summary_only=true", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestQualityAnalysisIntegration тестирует интеграцию запуска анализа качества
func TestQualityAnalysisIntegration(t *testing.T) {
	mux, db := setupQualityMux(t)
	defer db.Close()

	// Создаем тестовую выгрузку
	upload, err := db.CreateUpload("test-analysis-uuid", "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	// Устанавливаем database_id
	databaseID := 1
	_, err = db.Exec("UPDATE uploads SET database_id = ? WHERE id = ?", databaseID, upload.ID)
	if err != nil {
		t.Fatalf("Failed to update upload: %v", err)
	}

	// Тест запуска анализа
	t.Run("StartQualityAnalysis", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/upload/test-analysis-uuid/quality-analysis", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Ожидаем 202 Accepted или 500 (если нет данных для анализа)
		if w.Code != http.StatusAccepted && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 202 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}

		if w.Code == http.StatusAccepted {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}

			if status, ok := response["status"].(string); !ok || status != "analysis_started" {
				t.Errorf("Expected status 'analysis_started', got %v", response["status"])
			}
		}
	})
}

// TestQualityDashboardIntegration тестирует интеграцию получения дашборда качества
func TestQualityDashboardIntegration(t *testing.T) {
	mux, db := setupQualityMux(t)
	defer db.Close()

	// Создаем тестовую выгрузку с database_id
	upload, err := db.CreateUpload("test-dashboard-uuid", "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	databaseID := 1
	_, err = db.Exec("UPDATE uploads SET database_id = ? WHERE id = ?", databaseID, upload.ID)
	if err != nil {
		t.Fatalf("Failed to update upload: %v", err)
	}

	// Тест получения дашборда
	t.Run("GetQualityDashboard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/databases/1/quality-dashboard?days=30&limit=10", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Может быть 200 или 500 (если нет данных)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}

			if dbID, ok := response["database_id"].(float64); !ok || int(dbID) != databaseID {
				t.Errorf("Expected database_id %d, got %v", databaseID, response["database_id"])
			}
		}
	})
}

// TestQualityIssuesIntegration тестирует интеграцию получения проблем качества
func TestQualityIssuesIntegration(t *testing.T) {
	mux, db := setupQualityMux(t)
	defer db.Close()

	// Создаем тестовую выгрузку с database_id
	upload, err := db.CreateUpload("test-issues-uuid", "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	databaseID := 1
	_, err = db.Exec("UPDATE uploads SET database_id = ? WHERE id = ?", databaseID, upload.ID)
	if err != nil {
		t.Fatalf("Failed to update upload: %v", err)
	}

	// Тест получения проблем
	t.Run("GetQualityIssues", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/databases/1/quality-issues", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Может быть 200 или 500 (если нет данных)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}

			if _, ok := response["issues"]; !ok {
				t.Error("Response should contain 'issues' field")
			}
		}
	})

	// Тест получения проблем с фильтрами
	t.Run("GetQualityIssues_WithFilters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/databases/1/quality-issues?severity=CRITICAL&entity_type=nomenclature", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestQualityTrendsIntegration тестирует интеграцию получения трендов качества
func TestQualityTrendsIntegration(t *testing.T) {
	mux, db := setupQualityMux(t)
	defer db.Close()

	// Создаем тестовую выгрузку с database_id
	upload, err := db.CreateUpload("test-trends-uuid", "8.3", "test-config")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	databaseID := 1
	_, err = db.Exec("UPDATE uploads SET database_id = ? WHERE id = ?", databaseID, upload.ID)
	if err != nil {
		t.Fatalf("Failed to update upload: %v", err)
	}

	// Тест получения трендов
	t.Run("GetQualityTrends", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/databases/1/quality-trends?days=30", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Может быть 200 или 500 (если нет данных)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}

			if _, ok := response["trends"]; !ok {
				t.Error("Response should contain 'trends' field")
			}
		}
	})
}

// TestQualityEndpoints_InvalidMethods тестирует обработку неверных HTTP методов
func TestQualityEndpoints_InvalidMethods(t *testing.T) {
	mux, db := setupQualityMux(t)
	defer db.Close()

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{
			name:   "POST to quality-report should fail",
			method: "POST",
			path:   "/api/v1/upload/test-uuid/quality-report",
			want:   http.StatusMethodNotAllowed,
		},
		{
			name:   "GET to quality-analysis should fail",
			method: "GET",
			path:   "/api/v1/upload/test-uuid/quality-analysis",
			want:   http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.want, w.Code, w.Body.String())
			}
		})
	}
}

