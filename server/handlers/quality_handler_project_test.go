package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/quality"
	"httpserver/server/services"
)

// TestAggregateProjectStats_EmptyProject тестирует агрегацию для пустого проекта
func TestAggregateProjectStats_EmptyProject(t *testing.T) {
	handler, cleanup := setupTestQualityHandler(t)
	defer cleanup()

	ctx := context.Background()
	result, err := handler.aggregateProjectStats(ctx, []*database.ProjectDatabase{}, handler.normalizedDB)

	if err != nil {
		t.Fatalf("aggregateProjectStats failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// Проверяем, что все значения равны 0
	if totalItems, ok := resultMap["total_items"].(float64); ok {
		if int(totalItems) != 0 {
			t.Errorf("Expected 0 total_items, got %d", int(totalItems))
		}
	}

	databases, ok := resultMap["databases"].([]interface{})
	if !ok {
		t.Fatal("databases is not an array")
	}

	if len(databases) != 0 {
		t.Errorf("Expected 0 databases, got %d", len(databases))
	}
}

// setupTestQualityHandler создает тестовый QualityHandler
func setupTestQualityHandler(t *testing.T) (*QualityHandler, func()) {
	// Создаем временную директорию для тестовых БД
	tempDir, err := os.MkdirTemp("", "quality_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	// Создаем тестовую БД
	testDBPath := filepath.Join(tempDir, "test.db")
	testDB, err := database.NewDB(testDBPath)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create test DB: %v", err)
	}

	// Создаем таблицы для тестирования
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS normalized_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT,
			normalized_name TEXT,
			processing_level TEXT,
			quality_score REAL,
			ai_confidence REAL
		)
	`)
	if err != nil {
		testDB.Close()
		cleanup()
		t.Fatalf("Failed to create table: %v", err)
	}

	// Вставляем тестовые данные
	_, err = testDB.Exec(`
		INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score, ai_confidence)
		VALUES 
			('001', 'Test Item 1', 'basic', 0.5, 0.0),
			('002', 'Test Item 2', 'ai_enhanced', 0.0, 0.85),
			('003', 'Test Item 3', 'benchmark', 0.95, 0.0)
	`)
	if err != nil {
		testDB.Close()
		cleanup()
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Создаем QualityService
	qualityAnalyzer := quality.NewQualityAnalyzer(testDB)
	qualityService, err := services.NewQualityService(testDB, qualityAnalyzer)
	if err != nil {
		testDB.Close()
		cleanup()
		t.Fatalf("Failed to create quality service: %v", err)
	}

	// Создаем BaseHandler
	baseHandler := &BaseHandler{}

	// Создаем QualityHandler
	handler := &QualityHandler{
		BaseHandler:     baseHandler,
		qualityService:  qualityService,
		normalizedDB:     testDB,
		currentNormalizedDBPath: testDBPath,
		logFunc: func(entry interface{}) {
			// Тестовый логгер
		},
	}

	return handler, cleanup
}

// TestAggregateProjectStats_SingleDatabase тестирует агрегацию для одной БД
func TestAggregateProjectStats_SingleDatabase(t *testing.T) {
	handler, cleanup := setupTestQualityHandler(t)
	defer cleanup()

	// Создаем временную БД для проекта
	tempDir, _ := os.MkdirTemp("", "project_test_*")
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "db1.db")
	db, _ := database.NewDB(dbPath)
	db.Exec(`
		CREATE TABLE IF NOT EXISTS normalized_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT,
			normalized_name TEXT,
			processing_level TEXT,
			quality_score REAL
		)
	`)
	db.Exec(`INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score) VALUES ('001', 'Item 1', 'basic', 0.5)`)
	db.Close()

	projectDatabases := []*database.ProjectDatabase{
		{
			ID:       1,
			Name:     "Test DB 1",
			FilePath: dbPath,
			IsActive: true,
		},
	}

	ctx := context.Background()
	result, err := handler.aggregateProjectStats(ctx, projectDatabases, handler.normalizedDB)

	if err != nil {
		t.Fatalf("aggregateProjectStats failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// Проверяем наличие полей
	if _, ok := resultMap["total_items"]; !ok {
		t.Error("Response missing 'total_items' field")
	}

	if _, ok := resultMap["databases"]; !ok {
		t.Error("Response missing 'databases' field")
	}

	// Проверяем, что databases - это массив
	// Может быть []interface{} или []map[string]interface{}
	databasesRaw, ok := resultMap["databases"]
	if !ok {
		t.Error("Response missing 'databases' field")
		return
	}

	// Пробуем разные типы
	var databases []interface{}
	switch v := databasesRaw.(type) {
	case []interface{}:
		databases = v
	case []map[string]interface{}:
		// Преобразуем []map[string]interface{} в []interface{}
		databases = make([]interface{}, len(v))
		for i, m := range v {
			databases[i] = m
		}
	default:
		// Пробуем через JSON marshal/unmarshal
		jsonData, err := json.Marshal(v)
		if err != nil {
			t.Errorf("'databases' field is not an array (type: %T): %v", v, v)
			return
		}
		if err := json.Unmarshal(jsonData, &databases); err != nil {
			t.Errorf("Failed to unmarshal databases array: %v", err)
			return
		}
	}

	if len(databases) != 1 {
		t.Errorf("Expected 1 database, got %d", len(databases))
	}
}

// TestAggregateProjectStats_ErrorHandling тестирует обработку ошибок
func TestAggregateProjectStats_ErrorHandling(t *testing.T) {
	handler, cleanup := setupTestQualityHandler(t)
	defer cleanup()

	// Создаем БД с валидными и невалидными путями
	projectDatabases := []*database.ProjectDatabase{
		{
			ID:       1,
			Name:     "Valid DB",
			FilePath: handler.currentNormalizedDBPath, // Используем существующую БД
			IsActive: true,
		},
		{
			ID:       2,
			Name:     "Invalid DB",
			FilePath: "/nonexistent/path/to/db.db",
			IsActive: true,
		},
	}

	ctx := context.Background()
	result, err := handler.aggregateProjectStats(ctx, projectDatabases, handler.normalizedDB)

	if err != nil {
		t.Fatalf("aggregateProjectStats should handle errors gracefully, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// Должны обработаться хотя бы валидные БД
	databasesRaw, ok := resultMap["databases"]
	if !ok {
		t.Fatal("Response missing 'databases' field")
	}

	// Преобразуем в массив
	var databases []interface{}
	switch v := databasesRaw.(type) {
	case []interface{}:
		databases = v
	case []map[string]interface{}:
		databases = make([]interface{}, len(v))
		for i, m := range v {
			databases[i] = m
		}
	default:
		jsonData, _ := json.Marshal(v)
		json.Unmarshal(jsonData, &databases)
	}

	// Должна быть обработана хотя бы 1 валидная БД
	if len(databases) < 1 {
		t.Errorf("Expected at least 1 valid database, got %d", len(databases))
	}
}

// TestAggregateProjectStats_ParallelProcessing тестирует параллельную обработку
func TestAggregateProjectStats_ParallelProcessing(t *testing.T) {
	handler, cleanup := setupTestQualityHandler(t)
	defer cleanup()

	// Создаем несколько тестовых БД
	tempDir, _ := os.MkdirTemp("", "parallel_test_*")
	defer os.RemoveAll(tempDir)

	projectDatabases := make([]*database.ProjectDatabase, 0, 5)
	for i := 0; i < 5; i++ {
		dbPath := filepath.Join(tempDir, fmt.Sprintf("db%d.db", i))
		db, _ := database.NewDB(dbPath)
		db.Exec(`
			CREATE TABLE IF NOT EXISTS normalized_data (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				code TEXT,
				normalized_name TEXT,
				processing_level TEXT,
				quality_score REAL
			)
		`)
		db.Exec(`INSERT INTO normalized_data (code, normalized_name, processing_level, quality_score) VALUES (?, 'Item', 'basic', 0.5)`, fmt.Sprintf("00%d", i))
		db.Close()

		projectDatabases = append(projectDatabases, &database.ProjectDatabase{
			ID:       i + 1,
			Name:     fmt.Sprintf("Test DB %d", i+1),
			FilePath: dbPath,
			IsActive: true,
		})
	}

	ctx := context.Background()
	start := time.Now()
	result, err := handler.aggregateProjectStats(ctx, projectDatabases, handler.normalizedDB)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("aggregateProjectStats failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// Проверяем, что все БД обработаны
	databasesRaw, ok := resultMap["databases"]
	if !ok {
		t.Fatal("Response missing 'databases' field")
	}

	// Преобразуем в массив
	var databases []interface{}
	switch v := databasesRaw.(type) {
	case []interface{}:
		databases = v
	case []map[string]interface{}:
		databases = make([]interface{}, len(v))
		for i, m := range v {
			databases[i] = m
		}
	default:
		jsonData, _ := json.Marshal(v)
		json.Unmarshal(jsonData, &databases)
	}

	if len(databases) != 5 {
		t.Errorf("Expected 5 databases, got %d", len(databases))
	}

	// Проверяем, что параллельная обработка работает (не слишком долго)
	if duration > 10*time.Second {
		t.Errorf("Parallel processing took too long: %v", duration)
	}

	t.Logf("Processed %d databases in %v", len(databases), duration)
}

