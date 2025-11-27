package normalization

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"httpserver/database"
	"httpserver/nomenclature"
)

// setupTestKpvedDB создает тестовую БД с данными КПВЭД
func setupTestKpvedDB(t *testing.T) (*database.ServiceDB, func()) {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}

	// Инициализируем схему
	err = database.InitServiceSchema(serviceDB.GetDB())
	if err != nil {
		serviceDB.Close()
		t.Fatalf("Failed to init schema: %v", err)
	}

	// Создаем таблицу kpved_classifier
	createTable := `
		CREATE TABLE IF NOT EXISTS kpved_classifier (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_code TEXT,
			level INTEGER NOT NULL
		)
	`
	_, err = serviceDB.Exec(createTable)
	if err != nil {
		serviceDB.Close()
		t.Fatalf("Failed to create kpved_classifier table: %v", err)
	}

	// Добавляем тестовые данные
	_, err = serviceDB.Exec(`
		INSERT INTO kpved_classifier (code, name, parent_code, level) VALUES
		('A', 'Сельское, лесное и рыбное хозяйство', NULL, 1),
		('01', 'Растениеводство и животноводство, охота и предоставление соответствующих услуг в этих областях', 'A', 2),
		('01.1', 'Выращивание однолетних культур', '01', 3),
		('01.11', 'Выращивание зерновых культур', '01.1', 4),
		('01.11.1', 'Выращивание пшеницы', '01.11', 5),
		('25', 'Производство готовых металлических изделий', NULL, 1),
		('25.94', 'Изделия крепежные, изделия с резьбой нарезанной', '25', 2),
		('25.94.11', 'Изделия с резьбой нарезанной из металлов черных', '25.94', 3)
	`)
	if err != nil {
		serviceDB.Close()
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return serviceDB, func() { serviceDB.Close() }
}

// TestNewHierarchicalClassifierWithTree_Basic проверяет базовое создание классификатора с деревом
func TestNewHierarchicalClassifierWithTree_Basic(t *testing.T) {
	db, cleanup := setupTestKpvedDB(t)
	defer cleanup()

	// Создаем дерево
	tree := NewKpvedTree()
	if err := tree.BuildFromDatabase(db); err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	if len(tree.NodeMap) == 0 {
		t.Fatal("Expected tree to have nodes, got empty tree")
	}

	// Создаем AI клиент
	aiClient := nomenclature.NewAIClient("test-key", "test-model")

	// Создаем классификатор с готовым деревом
	classifier := NewHierarchicalClassifierWithTree(tree, db, aiClient)
	if classifier == nil {
		t.Fatal("Expected classifier to be created, got nil")
	}

	// Проверяем, что классификатор инициализирован
	if classifier.tree == nil {
		t.Fatal("Expected classifier to have tree, got nil")
	}

	if classifier.tree != tree {
		t.Fatal("Expected classifier to use the provided tree instance")
	}

	if classifier.aiClient == nil {
		t.Fatal("Expected classifier to have AI client, got nil")
	}

	if classifier.promptBuilder == nil {
		t.Fatal("Expected classifier to have prompt builder, got nil")
	}

	if classifier.cache == nil {
		t.Fatal("Expected classifier to have cache, got nil")
	}
}

// TestNewHierarchicalClassifierWithTree_Reuse проверяет переиспользование дерева
func TestNewHierarchicalClassifierWithTree_Reuse(t *testing.T) {
	db, cleanup := setupTestKpvedDB(t)
	defer cleanup()

	// Создаем одно дерево
	tree := NewKpvedTree()
	if err := tree.BuildFromDatabase(db); err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	nodeCount := len(tree.NodeMap)

	// Создаем несколько классификаторов с одним деревом
	classifiers := make([]*HierarchicalClassifier, 10)
	for i := 0; i < 10; i++ {
		aiClient := nomenclature.NewAIClient("test-key", "test-model")
		classifiers[i] = NewHierarchicalClassifierWithTree(tree, db, aiClient)
		if classifiers[i] == nil {
			t.Fatalf("Expected classifier %d to be created, got nil", i)
		}

		// Проверяем, что все используют одно дерево
		if classifiers[i].tree != tree {
			t.Fatalf("Classifier %d uses different tree instance", i)
		}

		// Проверяем, что дерево не изменилось
		if len(tree.NodeMap) != nodeCount {
			t.Fatalf("Tree node count changed from %d to %d", nodeCount, len(tree.NodeMap))
		}
	}
}

// TestNewHierarchicalClassifierWithTree_EmptyTree проверяет обработку пустого дерева
func TestNewHierarchicalClassifierWithTree_EmptyTree(t *testing.T) {
	db, cleanup := setupTestKpvedDB(t)
	defer cleanup()

	// Создаем пустое дерево
	tree := NewKpvedTree()

	aiClient := nomenclature.NewAIClient("test-key", "test-model")

	// Создание классификатора с пустым деревом должно работать
	// (но классификация может не работать корректно)
	classifier := NewHierarchicalClassifierWithTree(tree, db, aiClient)
	if classifier == nil {
		t.Fatal("Expected classifier to be created even with empty tree, got nil")
	}

	if classifier.tree == nil {
		t.Fatal("Expected classifier to have tree (even if empty), got nil")
	}

	if len(classifier.tree.NodeMap) != 0 {
		t.Fatal("Expected empty tree, got tree with nodes")
	}
}

// TestNewHierarchicalClassifier_Comparison проверяет, что оба конструктора создают рабочие классификаторы
func TestNewHierarchicalClassifier_Comparison(t *testing.T) {
	db, cleanup := setupTestKpvedDB(t)
	defer cleanup()

	aiClient := nomenclature.NewAIClient("test-key", "test-model")

	// Создаем классификатор обычным способом
	classifier1, err := NewHierarchicalClassifier(db, aiClient)
	if err != nil {
		t.Fatalf("Failed to create classifier: %v", err)
	}

	// Создаем дерево вручную
	tree := NewKpvedTree()
	if err := tree.BuildFromDatabase(db); err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Создаем классификатор с готовым деревом
	classifier2 := NewHierarchicalClassifierWithTree(tree, db, aiClient)

	// Оба классификатора должны иметь одинаковое количество узлов в дереве
	if len(classifier1.tree.NodeMap) != len(classifier2.tree.NodeMap) {
		t.Errorf("Expected same node count, got %d vs %d",
			len(classifier1.tree.NodeMap), len(classifier2.tree.NodeMap))
	}

	// Оба должны иметь одинаковое количество секций
	if len(classifier1.tree.Root.Children) != len(classifier2.tree.Root.Children) {
		t.Errorf("Expected same section count, got %d vs %d",
			len(classifier1.tree.Root.Children), len(classifier2.tree.Root.Children))
	}
}

// TestNewHierarchicalClassifierWithTree_NilDB проверяет обработку nil БД
func TestNewHierarchicalClassifierWithTree_NilDB(t *testing.T) {
	tree := NewKpvedTree()
	aiClient := nomenclature.NewAIClient("test-key", "test-model")

	// Создание классификатора с nil БД должно работать
	// (БД используется только для некоторых операций, не для инициализации)
	classifier := NewHierarchicalClassifierWithTree(tree, nil, aiClient)
	if classifier == nil {
		t.Fatal("Expected classifier to be created even with nil DB, got nil")
	}

	if classifier.db != nil {
		t.Fatal("Expected classifier to have nil DB, got non-nil")
	}
}

// TestNewHierarchicalClassifierWithTree_NilAIClient проверяет обработку nil AI клиента
func TestNewHierarchicalClassifierWithTree_NilAIClient(t *testing.T) {
	db, cleanup := setupTestKpvedDB(t)
	defer cleanup()

	tree := NewKpvedTree()
	if err := tree.BuildFromDatabase(db); err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Создание классификатора с nil AI клиентом должно работать
	// (но AI классификация не будет работать)
	classifier := NewHierarchicalClassifierWithTree(tree, db, nil)
	if classifier == nil {
		t.Fatal("Expected classifier to be created even with nil AI client, got nil")
	}

	if classifier.aiClient != nil {
		t.Fatal("Expected classifier to have nil AI client, got non-nil")
	}
}

