package normalization

import (
	"testing"

	"httpserver/database"
	"httpserver/nomenclature"
)

// TestHierarchicalClassifier_KeywordIntegration проверяет интеграцию KeywordClassifier
func TestHierarchicalClassifier_KeywordIntegration(t *testing.T) {
	// Создаем тестовую БД в памяти
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer serviceDB.Close()

	// Инициализируем схему
	err = database.InitServiceSchema(serviceDB.GetDB())
	if err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Создаем таблицу kpved_classifier, если её нет
	createTable := `
		CREATE TABLE IF NOT EXISTS kpved_classifier (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			parent_code TEXT,
			level INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err = serviceDB.Exec(createTable)
	if err != nil {
		t.Fatalf("Failed to create kpved_classifier table: %v", err)
	}

	// Добавляем минимальные данные КПВЭД для теста
	_, err = serviceDB.Exec(`
		INSERT INTO kpved_classifier (code, name, parent_code, level) VALUES
		('A', 'Раздел A', NULL, 1),
		('25', 'Производство готовых металлических изделий', 'A', 2),
		('25.94', 'Изделия крепежные, изделия с резьбой нарезанной', '25', 3),
		('25.94.11', 'Изделия с резьбой нарезанной из металлов черных, не включенные в другие группировки', '25.94', 4)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Создаем AI клиент (можно использовать mock или реальный)
	aiClient := nomenclature.NewAIClient("test-key", "test-model")

	// Создаем классификатор
	classifier, err := NewHierarchicalClassifier(serviceDB, aiClient)
	if err != nil {
		t.Fatalf("Failed to create classifier: %v", err)
	}

	// Проверяем, что keywordClassifier инициализирован
	if classifier.keywordClassifier == nil {
		t.Error("keywordClassifier is nil")
	}

	// Проверяем, что baseWordCache инициализирован
	if classifier.baseWordCache == nil {
		t.Error("baseWordCache is nil")
	}

	// Проверяем, что KeywordClassifier содержит паттерны
	patterns := classifier.keywordClassifier.GetPatterns()
	if len(patterns) == 0 {
		t.Error("KeywordClassifier has no patterns")
	}

	// Проверяем наличие конкретных паттернов
	if _, exists := patterns["болт"]; !exists {
		t.Error("Pattern 'болт' not found")
	}
	if _, exists := patterns["сверло"]; !exists {
		t.Error("Pattern 'сверло' not found")
	}
}

// TestHierarchicalClassifier_KeywordCache проверяет работу кэша корневых слов
func TestHierarchicalClassifier_KeywordCache(t *testing.T) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer serviceDB.Close()

	err = database.InitServiceSchema(serviceDB.GetDB())
	if err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Создаем таблицу kpved_classifier, если её нет
	createTable := `
		CREATE TABLE IF NOT EXISTS kpved_classifier (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			parent_code TEXT,
			level INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err = serviceDB.Exec(createTable)
	if err != nil {
		t.Fatalf("Failed to create kpved_classifier table: %v", err)
	}

	// Добавляем минимальные данные
	_, err = serviceDB.Exec(`
		INSERT INTO kpved_classifier (code, name, parent_code, level) VALUES
		('A', 'Раздел A', NULL, 1),
		('25', 'Производство готовых металлических изделий', 'A', 2),
		('25.94', 'Изделия крепежные, изделия с резьбой нарезанной', '25', 3),
		('25.94.11', 'Изделия с резьбой нарезанной из металлов черных, не включенные в другие группировки', '25.94', 4)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	aiClient := nomenclature.NewAIClient("test-key", "test-model")
	classifier, err := NewHierarchicalClassifier(serviceDB, aiClient)
	if err != nil {
		t.Fatalf("Failed to create classifier: %v", err)
	}

	// Проверяем извлечение корневого слова
	rootWord := classifier.keywordClassifier.extractRootWord("болт м10")
	if rootWord != "болт" {
		t.Errorf("Expected root word 'болт', got '%s'", rootWord)
	}

	// Проверяем классификацию по ключевому слову
	result, found := classifier.keywordClassifier.ClassifyByKeyword("болт м10", "инструмент")
	if !found {
		t.Error("Expected keyword classification to succeed")
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.FinalCode != "25.94.11" {
		t.Errorf("Expected code '25.94.11', got '%s'", result.FinalCode)
	}
}
