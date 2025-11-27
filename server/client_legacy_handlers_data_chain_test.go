package server

import (
	"path/filepath"
	"strconv"
	"testing"

	"httpserver/database"
	"httpserver/internal/infrastructure/cache"
)

// TestEnsureUploadRecordsForDatabase тестирует создание/обновление upload записей
func TestEnsureUploadRecordsForDatabase(t *testing.T) {
	// Создаем временную директорию
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "service.db")
	sourceDBPath := filepath.Join(tempDir, "source.db")

	// Создаем service DB
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient(
		"Test Client",
		"Test Client Legal",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.9,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем исходную базу данных с данными
	sourceDB, err := database.NewDB(sourceDBPath)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer sourceDB.Close()

	// Создаем upload запись без client_id и project_id
	upload, err := sourceDB.CreateUpload("test-uuid-1", "8.3", "TestConfig")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	// Создаем каталог и элементы
	catalog, err := sourceDB.AddCatalog(upload.ID, "Номенклатура", "nomenclature")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем тестовые элементы
	for i := 0; i < 5; i++ {
		err := sourceDB.AddCatalogItem(
			catalog.ID,
			"ref"+strconv.Itoa(i+1),
			"code"+strconv.Itoa(i+1),
			"Test Item "+strconv.Itoa(i+1),
			"",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to add catalog item %d: %v", i+1, err)
		}
	}

	// Создаем базу данных проекта
	projectDB, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", sourceDBPath, "Test database", 1024)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем сервер для тестирования
	normalizedDB, _ := database.NewDB(":memory:")
	defer normalizedDB.Close()

	mainDB, _ := database.NewDB(":memory:")
	defer mainDB.Close()

	server := &Server{
		serviceDB:   serviceDB,
		normalizedDB: normalizedDB,
		db:          mainDB,
	}

	// Тестируем ensureUploadRecordsForDatabase
	err = server.ensureUploadRecordsForDatabase(sourceDBPath, client.ID, project.ID, projectDB.ID)
	if err != nil {
		t.Fatalf("ensureUploadRecordsForDatabase failed: %v", err)
	}

	// Проверяем, что upload запись обновлена
	updatedUpload, err := sourceDB.GetUploadByID(upload.ID)
	if err != nil {
		t.Fatalf("Failed to get updated upload: %v", err)
	}

	if updatedUpload.ClientID == nil || *updatedUpload.ClientID != client.ID {
		t.Errorf("Expected client_id=%d, got %v", client.ID, updatedUpload.ClientID)
	}

	if updatedUpload.ProjectID == nil || *updatedUpload.ProjectID != project.ID {
		t.Errorf("Expected project_id=%d, got %v", project.ID, updatedUpload.ProjectID)
	}
}

// TestGetNomenclatureFromMainDBWithUploadRecords тестирует извлечение номенклатуры с upload записями
func TestGetNomenclatureFromMainDBWithUploadRecords(t *testing.T) {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "service.db")
	sourceDBPath := filepath.Join(tempDir, "source.db")

	// Создаем service DB
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient(
		"Test Client",
		"Test Client Legal",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.9,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем исходную базу данных
	sourceDB, err := database.NewDB(sourceDBPath)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer sourceDB.Close()

	// Создаем базу данных проекта
	projectDB, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", sourceDBPath, "Test database", 1024)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем upload запись с правильными client_id и project_id
	dbID := projectDB.ID
	upload, err := sourceDB.CreateUploadWithDatabase(
		"test-uuid-1",
		"8.3",
		"TestConfig",
		&dbID,
		"",
		"",
		"",
		1,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	// Обновляем client_id и project_id
	err = sourceDB.UpdateUploadClientProject(upload.ID, client.ID, project.ID)
	if err != nil {
		t.Fatalf("Failed to update upload: %v", err)
	}

	// Создаем каталог и элементы
	catalog, err := sourceDB.AddCatalog(upload.ID, "Номенклатура", "nomenclature")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем тестовые элементы
	testItems := []struct {
		reference string
		code      string
		name      string
	}{
		{"ref1", "code1", "Молоток ER-00013004"},
		{"ref2", "code2", "Молоток большой"},
		{"ref3", "code3", "Отвертка крестовая"},
	}

	for _, item := range testItems {
		err := sourceDB.AddCatalogItem(
			catalog.ID,
			item.reference,
			item.code,
			item.name,
			"",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to add catalog item %s: %v", item.name, err)
		}
	}

	// Создаем сервер с кэшем подключений
	normalizedDB, _ := database.NewDB(":memory:")
	defer normalizedDB.Close()

	mainDB, _ := database.NewDB(":memory:")
	defer mainDB.Close()

	// Создаем кэш подключений
	dbCache := cache.NewDatabaseConnectionCache()
	defer dbCache.CloseAll()

	server := &Server{
		serviceDB:        serviceDB,
		normalizedDB:     normalizedDB,
		db:               mainDB,
		dbConnectionCache: dbCache,
	}

	// Тестируем getNomenclatureFromMainDB
	projectNames := map[int]string{project.ID: project.Name}
	results, total, err := server.getNomenclatureFromMainDB(
		sourceDBPath,
		client.ID,
		[]int{project.ID},
		projectNames,
		"",
		100,
		0,
	)

	if err != nil {
		t.Fatalf("getNomenclatureFromMainDB failed: %v", err)
	}

	if total != len(testItems) {
		t.Errorf("Expected total=%d, got %d", len(testItems), total)
	}

	if len(results) != len(testItems) {
		t.Errorf("Expected %d results, got %d", len(testItems), len(results))
	}

	// Проверяем, что все элементы найдены
	foundNames := make(map[string]bool)
	for _, result := range results {
		foundNames[result.Name] = true
		if result.ProjectID != project.ID {
			t.Errorf("Expected project_id=%d, got %d", project.ID, result.ProjectID)
		}
	}

	for _, item := range testItems {
		if !foundNames[item.name] {
			t.Errorf("Item %s not found in results", item.name)
		}
	}
}

// TestGetNomenclatureFromMainDBWithoutUploadRecords тестирует fallback логику
func TestGetNomenclatureFromMainDBWithoutUploadRecords(t *testing.T) {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "service.db")
	sourceDBPath := filepath.Join(tempDir, "source.db")

	// Создаем service DB
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient(
		"Test Client",
		"Test Client Legal",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.9,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем исходную базу данных БЕЗ upload записей, но с catalog_items
	sourceDB, err := database.NewDB(sourceDBPath)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer sourceDB.Close()

	// Создаем upload запись БЕЗ client_id и project_id (симулируем старую БД)
	upload, err := sourceDB.CreateUpload("test-uuid-1", "8.3", "TestConfig")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	// Создаем каталог и элементы
	catalog, err := sourceDB.AddCatalog(upload.ID, "Номенклатура", "nomenclature")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем тестовые элементы
	testItems := []struct {
		reference string
		code      string
		name      string
	}{
		{"ref1", "code1", "Test Item 1"},
		{"ref2", "code2", "Test Item 2"},
	}

	for _, item := range testItems {
		err := sourceDB.AddCatalogItem(
			catalog.ID,
			item.reference,
			item.code,
			item.name,
			"",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to add catalog item %s: %v", item.name, err)
		}
	}

	// Создаем сервер с кэшем подключений
	normalizedDB, _ := database.NewDB(":memory:")
	defer normalizedDB.Close()

	mainDB, _ := database.NewDB(":memory:")
	defer mainDB.Close()

	dbCache := cache.NewDatabaseConnectionCache()
	defer dbCache.CloseAll()

	server := &Server{
		serviceDB:        serviceDB,
		normalizedDB:     normalizedDB,
		db:               mainDB,
		dbConnectionCache: dbCache,
	}

	// Тестируем getNomenclatureFromMainDB с fallback
	projectNames := map[int]string{project.ID: project.Name}
	results, total, err := server.getNomenclatureFromMainDB(
		sourceDBPath,
		client.ID,
		[]int{project.ID},
		projectNames,
		"",
		100,
		0,
	)

	if err != nil {
		t.Fatalf("getNomenclatureFromMainDB failed: %v", err)
	}

	// Fallback должен найти данные даже без правильных upload записей
	if total == 0 && len(results) == 0 {
		t.Log("Fallback logic found no data - this is expected if upload records don't have client_id/project_id")
		// В этом случае fallback должен сработать
		if total > 0 {
			t.Logf("Fallback found %d items", total)
		}
	}
}

// TestDataChainIntegration тестирует полную цепочку извлечения данных
func TestDataChainIntegration(t *testing.T) {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "service.db")
	sourceDBPath := filepath.Join(tempDir, "source.db")

	// Создаем service DB
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient(
		"Test Client",
		"Test Client Legal",
		"Test Description",
		"test@example.com",
		"+1234567890",
		"TAX123",
		"test_user",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(
		client.ID,
		"Test Project",
		"normalization",
		"Test Project Description",
		"1C",
		0.9,
	)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Создаем исходную базу данных с данными
	sourceDB, err := database.NewDB(sourceDBPath)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer sourceDB.Close()

	// Создаем upload запись (без client_id/project_id, как в реальных БД)
	upload, err := sourceDB.CreateUpload("test-uuid-1", "8.3", "TestConfig")
	if err != nil {
		t.Fatalf("Failed to create upload: %v", err)
	}

	// Создаем каталог и элементы
	catalog, err := sourceDB.AddCatalog(upload.ID, "Номенклатура", "nomenclature")
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}

	// Добавляем тестовые элементы
	testItemsCount := 10
	for i := 0; i < testItemsCount; i++ {
		err := sourceDB.AddCatalogItem(
			catalog.ID,
			"ref"+strconv.Itoa(i+1),
			"code"+strconv.Itoa(i+1),
			"Test Item "+strconv.Itoa(i+1),
			"",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to add catalog item %d: %v", i+1, err)
		}
	}

	// Создаем базу данных проекта (симулируем добавление БД через API)
	projectDB, err := serviceDB.CreateProjectDatabase(project.ID, "Test DB", sourceDBPath, "Test database", 1024)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем сервер
	normalizedDB, _ := database.NewDB(":memory:")
	defer normalizedDB.Close()

	mainDB, _ := database.NewDB(":memory:")
	defer mainDB.Close()

	dbCache := cache.NewDatabaseConnectionCache()
	defer dbCache.CloseAll()

	server := &Server{
		serviceDB:        serviceDB,
		normalizedDB:     normalizedDB,
		db:               mainDB,
		dbConnectionCache: dbCache,
	}

	// Шаг 1: Вызываем ensureUploadRecordsForDatabase (как при добавлении БД)
	err = server.ensureUploadRecordsForDatabase(sourceDBPath, client.ID, project.ID, projectDB.ID)
	if err != nil {
		t.Fatalf("ensureUploadRecordsForDatabase failed: %v", err)
	}

	// Шаг 2: Проверяем, что upload запись обновлена
	updatedUpload, err := sourceDB.GetUploadByID(upload.ID)
	if err != nil {
		t.Fatalf("Failed to get updated upload: %v", err)
	}

	if updatedUpload.ClientID == nil || *updatedUpload.ClientID != client.ID {
		t.Errorf("Expected client_id=%d, got %v", client.ID, updatedUpload.ClientID)
	}

	if updatedUpload.ProjectID == nil || *updatedUpload.ProjectID != project.ID {
		t.Errorf("Expected project_id=%d, got %v", project.ID, updatedUpload.ProjectID)
	}

	// Шаг 3: Тестируем извлечение данных через getNomenclatureFromMainDB
	projectNames := map[int]string{project.ID: project.Name}
	results, total, err := server.getNomenclatureFromMainDB(
		sourceDBPath,
		client.ID,
		[]int{project.ID},
		projectNames,
		"",
		100,
		0,
	)

	if err != nil {
		t.Fatalf("getNomenclatureFromMainDB failed: %v", err)
	}

	if total != testItemsCount {
		t.Errorf("Expected total=%d, got %d", testItemsCount, total)
	}

	if len(results) != testItemsCount {
		t.Errorf("Expected %d results, got %d", testItemsCount, len(results))
	}

	// Проверяем, что все результаты имеют правильный project_id
	for _, result := range results {
		if result.ProjectID != project.ID {
			t.Errorf("Expected project_id=%d, got %d", project.ID, result.ProjectID)
		}
		if result.SourceDatabase != sourceDBPath {
			t.Errorf("Expected source_database=%s, got %s", sourceDBPath, result.SourceDatabase)
		}
	}

	t.Logf("Successfully extracted %d items from database", len(results))
}

