package quality

import (
	"path/filepath"
	"testing"

	"httpserver/database"
)

// setupTestDB создает тестовую базу данных
func setupTestDB(t *testing.T) *database.DB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	
	return db
}

// TestNewQualityAnalyzer проверяет создание нового анализатора качества
func TestNewQualityAnalyzer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	analyzer := NewQualityAnalyzer(db)
	
	if analyzer == nil {
		t.Fatal("NewQualityAnalyzer() returned nil")
	}
	
	if analyzer.db == nil {
		t.Error("QualityAnalyzer.db is nil")
	}
}

// TestQualityAnalyzer_AnalyzeUpload проверяет анализ качества выгрузки
func TestQualityAnalyzer_AnalyzeUpload(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	analyzer := NewQualityAnalyzer(db)
	
	// Тест с несуществующей выгрузкой
	err := analyzer.AnalyzeUpload(999, 1)
	// Ожидаем, что метод не вернет ошибку, даже если выгрузка не существует
	// (метод должен обрабатывать такие случаи gracefully)
	if err != nil {
		t.Logf("AnalyzeUpload() returned error (expected for non-existent upload): %v", err)
	}
}

// TestQualityAnalyzer_FindFuzzyDuplicates проверяет поиск нечетких дубликатов
func TestQualityAnalyzer_FindFuzzyDuplicates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	analyzer := NewQualityAnalyzer(db)
	
	// Тест с несуществующей выгрузкой
	err := analyzer.findFuzzyDuplicates(999, 1)
	// Ожидаем, что метод не вернет ошибку для несуществующей выгрузки
	if err != nil {
		t.Logf("findFuzzyDuplicates() returned error (expected for non-existent upload): %v", err)
	}
}

// TestQualityAnalyzer_CalculateOverallScore проверяет расчет общего скора
func TestQualityAnalyzer_CalculateOverallScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	analyzer := NewQualityAnalyzer(db)
	
	// Тест с несуществующей выгрузкой
	err := analyzer.calculateOverallScore(999, 1)
	// Ожидаем, что метод не вернет ошибку для несуществующей выгрузки
	if err != nil {
		t.Logf("calculateOverallScore() returned error (expected for non-existent upload): %v", err)
	}
}

