package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/quality"
	"httpserver/server/handlers"
	"httpserver/server/services"
)

// TestQualityProjectStats_RealData тестирует агрегацию статистики проекта на реальных данных
func TestQualityProjectStats_RealData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Создаем временную директорию для тестовых БД
	tempDir, err := os.MkdirTemp("", "quality_integration_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Создаем несколько тестовых БД с реалистичными данными
	projectDatabases := make([]*database.ProjectDatabase, 0, 5)

	for i := 0; i < 5; i++ {
		dbPath := filepath.Join(tempDir, fmt.Sprintf("project_db_%d.db", i))
		db, err := database.NewDB(dbPath)
		if err != nil {
			t.Fatalf("Failed to create test DB %d: %v", i, err)
		}

		// Создаем таблицы
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS normalized_data (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				code TEXT,
				normalized_name TEXT,
				processing_level TEXT,
				quality_score REAL,
				ai_confidence REAL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			db.Close()
			t.Fatalf("Failed to create table in DB %d: %v", i, err)
		}

		// Вставляем реалистичные данные с разным качеством
		// БД 0: много базовых элементов
		// БД 1: смешанные элементы
		// БД 2: много AI улучшенных
		// БД 3: много эталонов
		// БД 4: пустая БД

		if i < 4 {
			switch i {
			case 0: // Много базовых
				for j := 0; j < 100; j++ {
					_, err = db.Exec(`
						INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score, ai_confidence)
						VALUES (?, ?, 'basic', 0.5, 0.0)
					`, fmt.Sprintf("CODE%03d", j), fmt.Sprintf("Item %d", j))
					if err != nil {
						t.Logf("Warning: failed to insert data in DB %d: %v", i, err)
					}
				}
			case 1: // Смешанные
				for j := 0; j < 50; j++ {
					level := "basic"
					quality := 0.5
					if j%3 == 0 {
						level = "ai_enhanced"
						quality = 0.85
					} else if j%5 == 0 {
						level = "benchmark"
						quality = 0.95
					}
					_, err = db.Exec(`
						INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score, ai_confidence)
						VALUES (?, ?, ?, ?, ?)
					`, fmt.Sprintf("CODE%03d", j), fmt.Sprintf("Item %d", j), level, quality, 0.0)
					if err != nil {
						t.Logf("Warning: failed to insert data in DB %d: %v", i, err)
					}
				}
			case 2: // Много AI улучшенных
				for j := 0; j < 75; j++ {
					_, err = db.Exec(`
						INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score, ai_confidence)
						VALUES (?, ?, 'ai_enhanced', 0.0, 0.9)
					`, fmt.Sprintf("CODE%03d", j), fmt.Sprintf("Item %d", j))
					if err != nil {
						t.Logf("Warning: failed to insert data in DB %d: %v", i, err)
					}
				}
			case 3: // Много эталонов
				for j := 0; j < 60; j++ {
					_, err = db.Exec(`
						INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score, ai_confidence)
						VALUES (?, ?, 'benchmark', 0.95, 0.0)
					`, fmt.Sprintf("CODE%03d", j), fmt.Sprintf("Item %d", j))
					if err != nil {
						t.Logf("Warning: failed to insert data in DB %d: %v", i, err)
					}
				}
			}
		}

		db.Close()

		projectDatabases = append(projectDatabases, &database.ProjectDatabase{
			ID:       i + 1,
			Name:     fmt.Sprintf("Project Database %d", i+1),
			FilePath: dbPath,
			IsActive: true,
		})
	}

	// Создаем тестовую БД для handler
	testDBPath := filepath.Join(tempDir, "test.db")
	testDB, err := database.NewDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer testDB.Close()

	// Создаем QualityService
	qualityAnalyzer := quality.NewQualityAnalyzer(testDB)
	qualityService, err := services.NewQualityService(testDB, qualityAnalyzer)
	if err != nil {
		t.Fatalf("Failed to create quality service: %v", err)
	}

	// Создаем handler
	baseHandler := handlers.NewBaseHandlerFromMiddleware()
	handler := handlers.NewQualityHandler(baseHandler, qualityService, func(entry interface{}) {
		// Тестовый логгер
	}, testDB, testDBPath)

	// Тестируем агрегацию
	start := time.Now()

	// Используем рефлексию или публичный метод для тестирования aggregateProjectStats
	// Для интеграционного теста создаем запрос к API
	handler.SetGetProjectDatabases(func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error) {
		return projectDatabases, nil
	})

	req := httptest.NewRequest("GET", "/api/quality/stats?project=1:1", nil)
	w := httptest.NewRecorder()

	handler.HandleQualityStats(w, req, testDB, testDBPath)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleQualityStats failed with status %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	duration := time.Since(start)

	resultMap := result

	// Проверяем результаты
	totalItems, ok := resultMap["total_items"].(float64)
	if !ok {
		t.Error("total_items is not a number")
	} else {
		expectedTotal := 100 + 50 + 75 + 60 // Сумма элементов из всех БД
		if int(totalItems) != expectedTotal {
			t.Errorf("Expected %d total_items, got %d", expectedTotal, int(totalItems))
		}
	}

	databases, ok := resultMap["databases"].([]interface{})
	if !ok {
		t.Fatal("databases is not an array")
	}

	if len(databases) != 4 { // 4 БД с данными (5-я пустая)
		t.Errorf("Expected 4 databases with data, got %d", len(databases))
	}

	// Проверяем, что параллельная обработка работает быстро
	if duration > 10*time.Second {
		t.Errorf("Aggregation took too long: %v", duration)
	}

	t.Logf("Successfully aggregated stats for %d databases in %v", len(databases), duration)
	t.Logf("Total items: %d", int(totalItems))
}

// TestQualityProjectStats_APIEndpoint тестирует API endpoint с реальными данными
func TestQualityProjectStats_APIEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "quality_api_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Создаем тестовую БД
	testDBPath := filepath.Join(tempDir, "test.db")
	testDB, err := database.NewDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer testDB.Close()

	// Создаем таблицы
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS normalized_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT,
			normalized_name TEXT,
			processing_level TEXT,
			quality_score REAL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Создаем БД для проекта
	db1Path := filepath.Join(tempDir, "project_db1.db")
	db1, _ := database.NewDB(db1Path)
	db1.Exec(`
		CREATE TABLE IF NOT EXISTS normalized_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT,
			normalized_name TEXT,
			processing_level TEXT,
			quality_score REAL
		)
	`)
	db1.Exec(`INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score) VALUES ('001', 'Item 1', 'basic', 0.5)`)
	// Не закрываем db1 здесь, так как он используется позже через FilePath
	// defer db1.Close() будет вызван в конце теста, если нужно

	// Создаем QualityService и Handler
	qualityAnalyzer := quality.NewQualityAnalyzer(testDB)
	qualityService, _ := services.NewQualityService(testDB, qualityAnalyzer)

	baseHandler := handlers.NewBaseHandlerFromMiddleware()
	handler := handlers.NewQualityHandler(baseHandler, qualityService, func(entry interface{}) {}, testDB, testDBPath)

	// Устанавливаем функцию получения баз проекта
	handler.SetGetProjectDatabases(func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error) {
		if projectID == 1 {
			return []*database.ProjectDatabase{
				{
					ID:       1,
					Name:     "Test Project DB",
					FilePath: db1Path,
					IsActive: true,
				},
			}, nil
		}
		return []*database.ProjectDatabase{}, nil
	})

	// Создаем HTTP запрос
	req := httptest.NewRequest("GET", "/api/quality/stats?project=1:1", nil)
	w := httptest.NewRecorder()

	// Вызываем handler
	handler.HandleQualityStats(w, req, testDB, testDBPath)

	// Проверяем ответ
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	// Парсим ответ
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Проверяем структуру ответа
	if _, ok := response["total_items"]; !ok {
		t.Error("Response missing 'total_items' field")
	}

	if _, ok := response["databases"]; !ok {
		t.Error("Response missing 'databases' field")
	}

	databases, ok := response["databases"].([]interface{})
	if !ok {
		t.Error("'databases' field is not an array")
	} else if len(databases) != 1 {
		t.Errorf("Expected 1 database, got %d", len(databases))
	}

	t.Logf("API endpoint test passed. Response: %+v", response)
}
