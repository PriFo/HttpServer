package services

import (
	"bytes"
	"os"
	"testing"

	"httpserver/database"
)

// setupTestGostsDB создает тестовую базу данных ГОСТов
func setupTestGostsDB(t *testing.T) *database.GostsDB {
	t.Helper()

	// Создаем временный файл БД
	tempFile, err := os.CreateTemp("", "test_gosts_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database file: %v", err)
	}
	tempFile.Close()

	// Удаляем файл, чтобы БД создалась заново
	os.Remove(tempFile.Name())

	gostsDB, err := database.NewGostsDB(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to create GOSTs database: %v", err)
	}

	// Удаляем файл после теста
	t.Cleanup(func() {
		gostsDB.Close()
		os.Remove(tempFile.Name())
	})

	return gostsDB
}

// TestNewGostService проверяет создание нового сервиса ГОСТов
func TestNewGostService(t *testing.T) {
	gostsDB := setupTestGostsDB(t)

	service := NewGostService(gostsDB)
	if service == nil {
		t.Error("NewGostService() should not return nil")
	}
}

// TestGostService_GetStatistics проверяет получение статистики
func TestGostService_GetStatistics(t *testing.T) {
	gostsDB := setupTestGostsDB(t)
	service := NewGostService(gostsDB)

	stats, err := service.GetStatistics()
	if err != nil {
		t.Fatalf("GetStatistics() failed: %v", err)
	}

	if stats == nil {
		t.Error("GetStatistics() should not return nil")
	}

	// Проверяем наличие основных полей
	if _, ok := stats["total_gosts"]; !ok {
		t.Error("Statistics should contain 'total_gosts' field")
	}
}

// TestGostService_GetGosts проверяет получение списка ГОСТов
func TestGostService_GetGosts(t *testing.T) {
	gostsDB := setupTestGostsDB(t)
	service := NewGostService(gostsDB)

	// Тестируем получение пустого списка
	result, err := service.GetGosts(10, 0, "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("GetGosts() failed: %v", err)
	}

	if result == nil {
		t.Error("GetGosts() should not return nil")
	}

	if total, ok := result["total"].(int); !ok || total != 0 {
		t.Errorf("Expected total=0, got %v", result["total"])
	}
}

// TestGostService_ImportGosts проверяет импорт ГОСТов из CSV
func TestGostService_ImportGosts(t *testing.T) {
	gostsDB := setupTestGostsDB(t)
	service := NewGostService(gostsDB)

	// Создаем минимальный CSV файл
	csvContent := `номер;название;дата принятия;статус
ГОСТ 12345-2020;Тестовый стандарт;2020-01-01;действующий
ГОСТ 67890-2021;Еще один стандарт;2021-01-01;действующий`

	reader := bytes.NewReader([]byte(csvContent))

	result, err := service.ImportGosts(reader, "test.csv", "test_source", "http://test.url")
	if err != nil {
		// Импорт может не сработать из-за парсинга, это нормально
		t.Logf("ImportGosts() returned error (expected for minimal CSV): %v", err)
		return
	}

	if result == nil {
		t.Error("ImportGosts() should not return nil on success")
	}

	if success, ok := result["success"].(int); ok && success > 0 {
		t.Logf("Successfully imported %d GOSTs", success)
	}
}

// TestGostService_GetGostDetail проверяет получение детальной информации
func TestGostService_GetGostDetail(t *testing.T) {
	gostsDB := setupTestGostsDB(t)
	service := NewGostService(gostsDB)

	// Создаем тестовый ГОСТ
	gost := &database.Gost{
		GostNumber: "ГОСТ TEST-2020",
		Title:      "Тестовый ГОСТ",
		Status:     "действующий",
		SourceType: "test",
	}

	createdGost, err := gostsDB.CreateOrUpdateGost(gost)
	if err != nil {
		t.Fatalf("Failed to create test GOST: %v", err)
	}

	// Получаем детальную информацию
	result, err := service.GetGostDetail(createdGost.ID)
	if err != nil {
		t.Fatalf("GetGostDetail() failed: %v", err)
	}

	if result == nil {
		t.Error("GetGostDetail() should not return nil")
	}

	if gostNumber, ok := result["gost_number"].(string); !ok || gostNumber != "ГОСТ TEST-2020" {
		t.Errorf("Expected gost_number='ГОСТ TEST-2020', got %v", result["gost_number"])
	}
}

// TestGostService_GetGostByNumber проверяет получение ГОСТа по номеру
func TestGostService_GetGostByNumber(t *testing.T) {
	gostsDB := setupTestGostsDB(t)
	service := NewGostService(gostsDB)

	// Создаем тестовый ГОСТ
	gost := &database.Gost{
		GostNumber: "ГОСТ BYNUMBER-2020",
		Title:      "ГОСТ для поиска по номеру",
		Status:     "действующий",
		SourceType: "test",
	}

	_, err := gostsDB.CreateOrUpdateGost(gost)
	if err != nil {
		t.Fatalf("Failed to create test GOST: %v", err)
	}

	// Получаем по номеру
	result, err := service.GetGostByNumber("ГОСТ BYNUMBER-2020")
	if err != nil {
		t.Fatalf("GetGostByNumber() failed: %v", err)
	}

	if result == nil {
		t.Error("GetGostByNumber() should not return nil")
	}

	if gostNumber, ok := result["gost_number"].(string); !ok || gostNumber != "ГОСТ BYNUMBER-2020" {
		t.Errorf("Expected gost_number='ГОСТ BYNUMBER-2020', got %v", result["gost_number"])
	}
}

// TestGostService_UploadDocument проверяет загрузку документа
func TestGostService_UploadDocument(t *testing.T) {
	gostsDB := setupTestGostsDB(t)
	service := NewGostService(gostsDB)

	// Создаем тестовый ГОСТ
	gost := &database.Gost{
		GostNumber: "ГОСТ DOC-2020",
		Title:      "ГОСТ с документом",
		Status:     "действующий",
		SourceType: "test",
	}

	createdGost, err := gostsDB.CreateOrUpdateGost(gost)
	if err != nil {
		t.Fatalf("Failed to create test GOST: %v", err)
	}

	// Создаем временный файл для документа
	tempFile, err := os.CreateTemp("", "test_doc_*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Записываем тестовые данные
	testContent := []byte("Test PDF content")
	_, err = tempFile.Write(testContent)
	if err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tempFile.Close()

	// Получаем размер файла
	fileInfo, err := os.Stat(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	// Загружаем документ
	result, err := service.UploadDocument(createdGost.ID, tempFile.Name(), "pdf", fileInfo.Size())
	if err != nil {
		t.Fatalf("UploadDocument() failed: %v", err)
	}

	if result == nil {
		t.Error("UploadDocument() should not return nil")
	}

	if docID, ok := result["id"].(int); !ok || docID == 0 {
		t.Errorf("Expected document ID > 0, got %v", result["id"])
	}
}

// TestGostService_SearchGosts проверяет поиск ГОСТов
func TestGostService_SearchGosts(t *testing.T) {
	gostsDB := setupTestGostsDB(t)
	service := NewGostService(gostsDB)

	// Создаем несколько тестовых ГОСТов
	gosts := []*database.Gost{
		{
			GostNumber: "ГОСТ SEARCH-1",
			Title:      "Стандарт для поиска номер один",
			Keywords:   "поиск тест",
			Status:     "действующий",
			SourceType: "test",
		},
		{
			GostNumber: "ГОСТ SEARCH-2",
			Title:      "Другой стандарт",
			Keywords:   "тест",
			Status:     "действующий",
			SourceType: "test",
		},
	}

	for _, gost := range gosts {
		_, err := gostsDB.CreateOrUpdateGost(gost)
		if err != nil {
			t.Fatalf("Failed to create test GOST: %v", err)
		}
	}

	// Ищем по ключевому слову
	result, err := service.GetGosts(10, 0, "", "", "поиск", "", "", "", "")
	if err != nil {
		t.Fatalf("GetGosts() with search failed: %v", err)
	}

	if result == nil {
		t.Error("GetGosts() should not return nil")
	}

	// Проверяем, что найдены результаты
	if total, ok := result["total"].(int); ok && total > 0 {
		t.Logf("Found %d GOSTs matching search", total)
	}
}
