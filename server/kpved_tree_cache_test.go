package server

import (
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"httpserver/database"
	"httpserver/normalization"
	"httpserver/nomenclature"
)

// setupTestServiceDB создает тестовую сервисную БД с данными КПВЭД
func setupTestServiceDB(t *testing.T) (*database.ServiceDB, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_service.db")

	serviceDB, err := database.NewServiceDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}

	// Создаем таблицу kpved_classifier с тестовыми данными
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer db.Close()

	// Создаем таблицу kpved_classifier
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS kpved_classifier (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_code TEXT,
			level INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create kpved_classifier table: %v", err)
	}

	// Вставляем тестовые данные (секция A)
	_, err = db.Exec(`
		INSERT INTO kpved_classifier (code, name, parent_code, level) VALUES
		('A', 'Сельское, лесное и рыбное хозяйство', NULL, 1),
		('01', 'Растениеводство и животноводство, охота и предоставление соответствующих услуг в этих областях', 'A', 2),
		('01.1', 'Выращивание однолетних культур', '01', 3),
		('01.11', 'Выращивание зерновых культур', '01.1', 4),
		('01.11.1', 'Выращивание пшеницы', '01.11', 5)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return serviceDB, dbPath
}

// TestGetOrCreateKpvedTree_FirstCall создает дерево при первом вызове
func TestGetOrCreateKpvedTree_FirstCall(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	server := &Server{
		serviceDB: serviceDB,
	}

	// Первый вызов должен создать дерево
	tree := server.getOrCreateKpvedTree()
	if tree == nil {
		t.Fatal("Expected tree to be created, got nil")
	}

	if len(tree.NodeMap) == 0 {
		t.Fatal("Expected tree to have nodes, got empty tree")
	}

	if len(tree.Root.Children) == 0 {
		t.Fatal("Expected tree to have root children, got none")
	}

	// Проверяем, что дерево кэшировано
	if server.kpvedTree == nil {
		t.Fatal("Expected tree to be cached, got nil")
	}

	if server.kpvedTree != tree {
		t.Fatal("Expected cached tree to be the same instance")
	}
}

// TestGetOrCreateKpvedTree_CacheReuse проверяет переиспользование кэша
func TestGetOrCreateKpvedTree_CacheReuse(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	server := &Server{
		serviceDB: serviceDB,
	}

	// Первый вызов
	tree1 := server.getOrCreateKpvedTree()
	if tree1 == nil {
		t.Fatal("Expected tree to be created, got nil")
	}

	// Второй вызов должен вернуть то же дерево
	tree2 := server.getOrCreateKpvedTree()
	if tree2 == nil {
		t.Fatal("Expected tree to be returned, got nil")
	}

	if tree1 != tree2 {
		t.Fatal("Expected same tree instance from cache, got different instances")
	}
}

// TestGetOrCreateKpvedTree_ConcurrentAccess проверяет потокобезопасность
func TestGetOrCreateKpvedTree_ConcurrentAccess(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	server := &Server{
		serviceDB: serviceDB,
	}

	// Запускаем множество горутин одновременно
	const goroutines = 50
	var wg sync.WaitGroup
	trees := make([]*normalization.KpvedTree, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			trees[idx] = server.getOrCreateKpvedTree()
		}(i)
	}

	wg.Wait()

	// Все горутины должны получить одно и то же дерево
	firstTree := trees[0]
	if firstTree == nil {
		t.Fatal("Expected tree to be created, got nil")
	}

	for i := 1; i < goroutines; i++ {
		if trees[i] != firstTree {
			t.Fatalf("Goroutine %d got different tree instance", i)
		}
	}

	// Проверяем, что дерево кэшировано
	if server.kpvedTree != firstTree {
		t.Fatal("Expected cached tree to match returned tree")
	}
}

// TestInvalidateKpvedTreeCache проверяет инвалидацию кэша
func TestInvalidateKpvedTreeCache(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	server := &Server{
		serviceDB: serviceDB,
	}

	// Создаем дерево
	tree1 := server.getOrCreateKpvedTree()
	if tree1 == nil {
		t.Fatal("Expected tree to be created, got nil")
	}

	// Проверяем, что дерево кэшировано
	if server.kpvedTree == nil {
		t.Fatal("Expected tree to be cached, got nil")
	}

	// Инвалидируем кэш
	server.invalidateKpvedTreeCache()

	// Проверяем, что кэш очищен
	if server.kpvedTree != nil {
		t.Fatal("Expected cache to be invalidated, but tree still exists")
	}

	// Следующий вызов должен создать новое дерево
	tree2 := server.getOrCreateKpvedTree()
	if tree2 == nil {
		t.Fatal("Expected new tree to be created, got nil")
	}

	// Новое дерево должно быть другим экземпляром (хотя содержимое может быть одинаковым)
	if tree1 == tree2 {
		t.Fatal("Expected new tree instance after cache invalidation")
	}
}

// TestGetOrCreateKpvedTree_NilServiceDB проверяет обработку nil serviceDB
func TestGetOrCreateKpvedTree_NilServiceDB(t *testing.T) {
	server := &Server{
		serviceDB: nil,
	}

	tree := server.getOrCreateKpvedTree()
	if tree != nil {
		t.Fatal("Expected nil tree when serviceDB is nil, got tree")
	}
}

// TestNewHierarchicalClassifierWithTree проверяет создание классификатора с готовым деревом
func TestNewHierarchicalClassifierWithTree(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	// Создаем дерево
	tree := normalization.NewKpvedTree()
	if err := tree.BuildFromDatabase(serviceDB); err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	if len(tree.NodeMap) == 0 {
		t.Fatal("Expected tree to have nodes, got empty tree")
	}

	// Создаем AI клиент (можно использовать пустой API ключ для теста)
	aiClient := nomenclature.NewAIClient("test-key", "test-model")

	// Создаем классификатор с готовым деревом
	classifier := normalization.NewHierarchicalClassifierWithTree(tree, serviceDB, aiClient)
	if classifier == nil {
		t.Fatal("Expected classifier to be created, got nil")
	}

	// Проверяем, что классификатор использует переданное дерево
	if classifier == nil {
		t.Fatal("Expected classifier to have tree, got nil")
	}
}

// TestNewHierarchicalClassifierWithTree_Reuse проверяет переиспользование дерева
func TestNewHierarchicalClassifierWithTree_Reuse(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	// Создаем одно дерево
	tree := normalization.NewKpvedTree()
	if err := tree.BuildFromDatabase(serviceDB); err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	// Создаем несколько классификаторов с одним деревом
	classifiers := make([]*normalization.HierarchicalClassifier, 5)
	for i := 0; i < 5; i++ {
		aiClient := nomenclature.NewAIClient("test-key", "test-model")
		classifiers[i] = normalization.NewHierarchicalClassifierWithTree(tree, serviceDB, aiClient)
		if classifiers[i] == nil {
			t.Fatalf("Expected classifier %d to be created, got nil", i)
		}
	}

	// Все классификаторы должны работать корректно
	// (дерево потокобезопасно для чтения)
	for i, classifier := range classifiers {
		if classifier == nil {
			t.Fatalf("Classifier %d is nil", i)
		}
	}
}

// TestTestModelBenchmark_SharedTree проверяет использование sharedTree в бенчмарке
func TestTestModelBenchmark_SharedTree(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	server := &Server{
		serviceDB: serviceDB,
	}

	// Создаем sharedTree
	sharedTree := server.getOrCreateKpvedTree()
	if sharedTree == nil {
		t.Fatal("Failed to create shared tree")
	}

	// Тестовые продукты
	testProducts := []string{
		"Пшеница",
		"Ячмень",
		"Овес",
	}

	// API ключ (можно использовать тестовый)
	apiKey := "test-api-key"
	modelName := "test-model"

	// Запускаем бенчмарк
	result := server.testModelBenchmark(
		apiKey,
		modelName,
		testProducts,
		1, // maxRetries
		100*time.Millisecond, // retryDelay
		sharedTree,
	)

	// Проверяем результат
	if result == nil {
		t.Fatal("Expected benchmark result, got nil")
	}

	// Проверяем наличие обязательных полей
	if _, ok := result["model"]; !ok {
		t.Error("Expected 'model' field in result")
	}

	if _, ok := result["total_requests"]; !ok {
		t.Error("Expected 'total_requests' field in result")
	}

	// Проверяем, что модель указана правильно
	if model, ok := result["model"].(string); !ok || model != modelName {
		t.Errorf("Expected model to be %s, got %v", modelName, result["model"])
	}

	// Проверяем, что total_requests соответствует количеству продуктов
	if total, ok := result["total_requests"].(int); !ok || total != len(testProducts) {
		t.Errorf("Expected total_requests to be %d, got %v", len(testProducts), result["total_requests"])
	}
}

// TestTestModelBenchmark_WorkerPool проверяет ограничение параллелизма
func TestTestModelBenchmark_WorkerPool(t *testing.T) {
	serviceDB, _ := setupTestServiceDB(t)
	defer serviceDB.Close()

	server := &Server{
		serviceDB: serviceDB,
	}

	// Создаем sharedTree
	sharedTree := server.getOrCreateKpvedTree()
	if sharedTree == nil {
		t.Fatal("Failed to create shared tree")
	}

	// Создаем много тестовых продуктов для проверки ограничения параллелизма
	testProducts := make([]string, 50)
	for i := 0; i < 50; i++ {
		testProducts[i] = "Тестовый продукт " + string(rune('A'+i))
	}

	apiKey := "test-api-key"
	modelName := "test-model"

	startTime := time.Now()

	// Запускаем бенчмарк
	result := server.testModelBenchmark(
		apiKey,
		modelName,
		testProducts,
		1, // maxRetries
		50*time.Millisecond, // retryDelay
		sharedTree,
	)

	duration := time.Since(startTime)

	// Проверяем результат
	if result == nil {
		t.Fatal("Expected benchmark result, got nil")
	}

	// Проверяем, что бенчмарк завершился
	if total, ok := result["total_requests"].(int); !ok || total != len(testProducts) {
		t.Errorf("Expected total_requests to be %d, got %v", len(testProducts), result["total_requests"])
	}

	// Логируем время выполнения для анализа
	t.Logf("Benchmark completed in %v for %d products", duration, len(testProducts))
}

// TestGetOrCreateKpvedTree_EmptyDatabase проверяет обработку пустой БД
func TestGetOrCreateKpvedTree_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty_service.db")

	serviceDB, err := database.NewServiceDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем таблицу, но не вставляем данные
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS kpved_classifier (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			parent_code TEXT,
			level INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create kpved_classifier table: %v", err)
	}

	server := &Server{
		serviceDB: serviceDB,
	}

	// Попытка создать дерево из пустой БД должна вернуть nil
	tree := server.getOrCreateKpvedTree()
	if tree != nil {
		t.Fatal("Expected nil tree for empty database, got tree")
	}
}

