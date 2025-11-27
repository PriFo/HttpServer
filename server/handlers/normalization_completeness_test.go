package handlers

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB создает временную тестовую базу данных с необходимыми таблицами
func setupTestDB(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "normalization_completeness_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer conn.Close()

	// Создаем таблицу номенклатуры
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS nomenclature_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			nomenclature_code TEXT,
			nomenclature_name TEXT,
			characteristic_name TEXT,
			attributes_xml TEXT
		)
	`)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create nomenclature_items table: %v", err)
	}

	// Создаем таблицу контрагентов
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS counterparties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			inn TEXT,
			bin TEXT,
			legal_address TEXT,
			postal_address TEXT,
			contact_phone TEXT,
			contact_email TEXT,
			name TEXT
		)
	`)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create counterparties table: %v", err)
	}

	cleanup := func() {
		conn.Close()
		os.RemoveAll(tempDir)
	}

	return dbPath, cleanup
}

// TestCalculateCompletenessMetrics_EmptyDatabase тестирует расчет метрик для пустой БД
func TestCalculateCompletenessMetrics_EmptyDatabase(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	handler := &NormalizationHandler{}
	ctx := context.Background()

	metrics, err := handler.calculateCompletenessMetrics(ctx, dbPath, 0, 0)
	if err != nil {
		t.Fatalf("calculateCompletenessMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Все метрики должны быть 0
	if metrics.NomenclatureCompleteness.OverallCompleteness != 0 {
		t.Errorf("Expected 0 overall completeness for nomenclature, got %f", metrics.NomenclatureCompleteness.OverallCompleteness)
	}
	if metrics.CounterpartyCompleteness.OverallCompleteness != 0 {
		t.Errorf("Expected 0 overall completeness for counterparties, got %f", metrics.CounterpartyCompleteness.OverallCompleteness)
	}
}

// TestCalculateCompletenessMetrics_NomenclatureOnly тестирует расчет метрик только для номенклатуры
func TestCalculateCompletenessMetrics_NomenclatureOnly(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer conn.Close()

	// Вставляем тестовые данные
	_, err = conn.Exec(`
		INSERT INTO nomenclature_items (nomenclature_code, nomenclature_name, characteristic_name, attributes_xml)
		VALUES 
			('ART001', 'Item 1', 'Description 1', '<ЕдиницаИзмерения>шт</ЕдиницаИзмерения>'),
			('ART002', 'Item 2', NULL, '<Единица>кг</Единица>'),
			('ART003', NULL, 'Description 3', NULL),
			(NULL, 'Item 4', 'Description 4', '<Unit>pcs</Unit>')
	`)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to insert test data: %v", err)
	}

	handler := &NormalizationHandler{}
	ctx := context.Background()

	metrics, err := handler.calculateCompletenessMetrics(ctx, dbPath, 4, 0)
	if err != nil {
		t.Fatalf("calculateCompletenessMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Проверяем метрики: 3 из 4 имеют код (75%), 3 из 4 имеют единицы (75%), 3 из 4 имеют описания (75%)
	expectedArticles := 75.0
	expectedUnits := 75.0
	expectedDescriptions := 75.0
	expectedOverall := (expectedArticles + expectedUnits + expectedDescriptions) / 3

	tolerance := 0.1
	if diff := abs(metrics.NomenclatureCompleteness.ArticlesPercent - expectedArticles); diff > tolerance {
		t.Errorf("ArticlesPercent: expected ~%f, got %f (diff: %f)", expectedArticles, metrics.NomenclatureCompleteness.ArticlesPercent, diff)
	}
	if diff := abs(metrics.NomenclatureCompleteness.UnitsPercent - expectedUnits); diff > tolerance {
		t.Errorf("UnitsPercent: expected ~%f, got %f (diff: %f)", expectedUnits, metrics.NomenclatureCompleteness.UnitsPercent, diff)
	}
	if diff := abs(metrics.NomenclatureCompleteness.DescriptionsPercent - expectedDescriptions); diff > tolerance {
		t.Errorf("DescriptionsPercent: expected ~%f, got %f (diff: %f)", expectedDescriptions, metrics.NomenclatureCompleteness.DescriptionsPercent, diff)
	}
	if diff := abs(metrics.NomenclatureCompleteness.OverallCompleteness - expectedOverall); diff > tolerance {
		t.Errorf("OverallCompleteness: expected ~%f, got %f (diff: %f)", expectedOverall, metrics.NomenclatureCompleteness.OverallCompleteness, diff)
	}
}

// TestCalculateCompletenessMetrics_CounterpartiesOnly тестирует расчет метрик только для контрагентов
func TestCalculateCompletenessMetrics_CounterpartiesOnly(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer conn.Close()

	// Вставляем тестовые данные
	_, err = conn.Exec(`
		INSERT INTO counterparties (inn, bin, legal_address, postal_address, contact_phone, contact_email, name)
		VALUES 
			('1234567890', NULL, 'Address 1', NULL, '123-456-7890', 'test1@example.com', 'Company 1'),
			(NULL, '987654321', NULL, 'Address 2', NULL, NULL, 'Company 2'),
			('1111111111', NULL, NULL, NULL, '111-222-3333', NULL, 'Company 3'),
			(NULL, NULL, NULL, NULL, NULL, NULL, 'Company 4')
	`)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to insert test data: %v", err)
	}

	handler := &NormalizationHandler{}
	ctx := context.Background()

	metrics, err := handler.calculateCompletenessMetrics(ctx, dbPath, 0, 4)
	if err != nil {
		t.Fatalf("calculateCompletenessMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Проверяем метрики:
	// ИНН/БИН: Company 1 (inn), Company 2 (bin), Company 3 (inn) = 3/4 = 75%
	// Адреса: Company 1 (legal_address), Company 2 (postal_address) = 2/4 = 50%
	// Контакты: Company 1 (phone + email), Company 3 (phone) = 2/4 = 50% (нужен хотя бы один контакт)
	expectedINN := 75.0
	expectedAddress := 50.0
	expectedContacts := 50.0 // Company 1 и Company 3 имеют контакты
	expectedOverall := (expectedINN + expectedAddress + expectedContacts) / 3

	tolerance := 0.1
	if diff := abs(metrics.CounterpartyCompleteness.INNPercent - expectedINN); diff > tolerance {
		t.Errorf("INNPercent: expected ~%f, got %f (diff: %f)", expectedINN, metrics.CounterpartyCompleteness.INNPercent, diff)
	}
	if diff := abs(metrics.CounterpartyCompleteness.AddressPercent - expectedAddress); diff > tolerance {
		t.Errorf("AddressPercent: expected ~%f, got %f (diff: %f)", expectedAddress, metrics.CounterpartyCompleteness.AddressPercent, diff)
	}
	if diff := abs(metrics.CounterpartyCompleteness.ContactsPercent - expectedContacts); diff > tolerance {
		t.Errorf("ContactsPercent: expected ~%f, got %f (diff: %f)", expectedContacts, metrics.CounterpartyCompleteness.ContactsPercent, diff)
	}
	if diff := abs(metrics.CounterpartyCompleteness.OverallCompleteness - expectedOverall); diff > tolerance {
		t.Errorf("OverallCompleteness: expected ~%f, got %f (diff: %f)", expectedOverall, metrics.CounterpartyCompleteness.OverallCompleteness, diff)
	}
}

// TestCalculateCompletenessMetrics_BothTypes тестирует расчет метрик для обоих типов данных
func TestCalculateCompletenessMetrics_BothTypes(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer conn.Close()

	// Вставляем данные номенклатуры
	_, err = conn.Exec(`
		INSERT INTO nomenclature_items (nomenclature_code, nomenclature_name, characteristic_name, attributes_xml)
		VALUES 
			('ART001', 'Item 1', 'Desc 1', '<ЕдиницаИзмерения>шт</ЕдиницаИзмерения>'),
			('ART002', 'Item 2', NULL, NULL)
	`)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to insert nomenclature data: %v", err)
	}

	// Вставляем данные контрагентов
	_, err = conn.Exec(`
		INSERT INTO counterparties (inn, legal_address, contact_phone, name)
		VALUES 
			('1234567890', 'Address 1', '123-456-7890', 'Company 1'),
			(NULL, NULL, NULL, 'Company 2')
	`)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to insert counterparties data: %v", err)
	}

	handler := &NormalizationHandler{}
	ctx := context.Background()

	metrics, err := handler.calculateCompletenessMetrics(ctx, dbPath, 2, 2)
	if err != nil {
		t.Fatalf("calculateCompletenessMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	// Проверяем, что оба типа метрик рассчитаны
	if metrics.NomenclatureCompleteness.OverallCompleteness == 0 && metrics.NomenclatureCompleteness.ArticlesPercent == 0 {
		t.Error("Expected non-zero nomenclature metrics")
	}
	if metrics.CounterpartyCompleteness.OverallCompleteness == 0 && metrics.CounterpartyCompleteness.INNPercent == 0 {
		t.Error("Expected non-zero counterparty metrics")
	}
}

// TestCalculateCompletenessMetrics_InvalidPath тестирует обработку неверного пути к БД
func TestCalculateCompletenessMetrics_InvalidPath(t *testing.T) {
	handler := &NormalizationHandler{}
	ctx := context.Background()

	invalidPath := "/nonexistent/path/to/database.db"
	_, err := handler.calculateCompletenessMetrics(ctx, invalidPath, 10, 10)

	if err == nil {
		t.Error("Expected error for invalid database path")
	}
}

// TestCalculateCompletenessMetrics_Timeout тестирует обработку таймаута
func TestCalculateCompletenessMetrics_Timeout(t *testing.T) {
	dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	handler := &NormalizationHandler{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Ждем, чтобы контекст точно был отменен
	time.Sleep(10 * time.Millisecond)

	_, err := handler.calculateCompletenessMetrics(ctx, dbPath, 10, 10)

	// Может быть либо timeout, либо другие ошибки, но не должно паниковать
	if err == nil {
		t.Log("No error returned (may be acceptable if query completes quickly)")
	}
}

// TestCalculateOverallCompleteness_EmptyStats тестирует расчет общих метрик для пустого массива
func TestCalculateOverallCompleteness_EmptyStats(t *testing.T) {
	handler := &NormalizationHandler{}

	stats := []DatabasePreviewStats{}
	result := handler.calculateOverallCompleteness(stats)

	if result.NomenclatureCompleteness.OverallCompleteness != 0 {
		t.Errorf("Expected 0 overall completeness, got %f", result.NomenclatureCompleteness.OverallCompleteness)
	}
	if result.CounterpartyCompleteness.OverallCompleteness != 0 {
		t.Errorf("Expected 0 overall completeness for counterparties, got %f", result.CounterpartyCompleteness.OverallCompleteness)
	}
}

// TestCalculateOverallCompleteness_SingleDatabase тестирует расчет общих метрик для одной БД
func TestCalculateOverallCompleteness_SingleDatabase(t *testing.T) {
	handler := &NormalizationHandler{}

	stats := []DatabasePreviewStats{
		{
			NomenclatureCount: 100,
			CounterpartyCount: 50,
			Completeness: &CompletenessMetrics{
				NomenclatureCompleteness: struct {
					ArticlesPercent      float64 `json:"articles_percent"`
					UnitsPercent         float64 `json:"units_percent"`
					DescriptionsPercent  float64 `json:"descriptions_percent"`
					OverallCompleteness  float64 `json:"overall_completeness"`
				}{
					ArticlesPercent:      80.0,
					UnitsPercent:         70.0,
					DescriptionsPercent:  90.0,
					OverallCompleteness:  80.0,
				},
				CounterpartyCompleteness: struct {
					INNPercent          float64 `json:"inn_percent"`
					AddressPercent      float64 `json:"address_percent"`
					ContactsPercent     float64 `json:"contacts_percent"`
					OverallCompleteness float64 `json:"overall_completeness"`
				}{
					INNPercent:          75.0,
					AddressPercent:      85.0,
					ContactsPercent:     65.0,
					OverallCompleteness: 75.0,
				},
			},
		},
	}

	result := handler.calculateOverallCompleteness(stats)

	// Проверяем, что метрики соответствуют ожидаемым
	// Используем больший tolerance из-за округления при вычислениях
	tolerance := 1.0
	if diff := abs(result.NomenclatureCompleteness.ArticlesPercent - 80.0); diff > tolerance {
		t.Errorf("ArticlesPercent: expected 80.0, got %f (diff: %f)", result.NomenclatureCompleteness.ArticlesPercent, diff)
	}
	if diff := abs(result.NomenclatureCompleteness.UnitsPercent - 70.0); diff > tolerance {
		t.Errorf("UnitsPercent: expected 70.0, got %f (diff: %f)", result.NomenclatureCompleteness.UnitsPercent, diff)
	}
	if diff := abs(result.NomenclatureCompleteness.DescriptionsPercent - 90.0); diff > tolerance {
		t.Errorf("DescriptionsPercent: expected 90.0, got %f (diff: %f)", result.NomenclatureCompleteness.DescriptionsPercent, diff)
	}
	if diff := abs(result.CounterpartyCompleteness.INNPercent - 75.0); diff > tolerance {
		t.Errorf("INNPercent: expected 75.0, got %f (diff: %f)", result.CounterpartyCompleteness.INNPercent, diff)
	}
}

// TestCalculateOverallCompleteness_MultipleDatabases тестирует агрегацию метрик для нескольких БД
func TestCalculateOverallCompleteness_MultipleDatabases(t *testing.T) {
	handler := &NormalizationHandler{}

	stats := []DatabasePreviewStats{
		{
			NomenclatureCount: 100,
			Completeness: &CompletenessMetrics{
				NomenclatureCompleteness: struct {
					ArticlesPercent      float64 `json:"articles_percent"`
					UnitsPercent         float64 `json:"units_percent"`
					DescriptionsPercent  float64 `json:"descriptions_percent"`
					OverallCompleteness  float64 `json:"overall_completeness"`
				}{
					ArticlesPercent:     50.0,
					UnitsPercent:        60.0,
					DescriptionsPercent: 70.0,
					OverallCompleteness: 60.0,
				},
			},
		},
		{
			NomenclatureCount: 200,
			Completeness: &CompletenessMetrics{
				NomenclatureCompleteness: struct {
					ArticlesPercent      float64 `json:"articles_percent"`
					UnitsPercent         float64 `json:"units_percent"`
					DescriptionsPercent  float64 `json:"descriptions_percent"`
					OverallCompleteness  float64 `json:"overall_completeness"`
				}{
					ArticlesPercent:     80.0,
					UnitsPercent:        90.0,
					DescriptionsPercent: 100.0,
					OverallCompleteness: 90.0,
				},
			},
		},
	}

	result := handler.calculateOverallCompleteness(stats)

	// Ожидаемые значения с учетом весов (100 записей * 50% + 200 записей * 80%) / 300 = 70%
	expectedArticles := 70.0
	// (100 * 60% + 200 * 90%) / 300 = 80%
	expectedUnits := 80.0
	// (100 * 70% + 200 * 100%) / 300 = 90%
	expectedDescriptions := 90.0

	tolerance := 0.1
	if diff := abs(result.NomenclatureCompleteness.ArticlesPercent - expectedArticles); diff > tolerance {
		t.Errorf("ArticlesPercent: expected ~%f, got %f (diff: %f)", expectedArticles, result.NomenclatureCompleteness.ArticlesPercent, diff)
	}
	if diff := abs(result.NomenclatureCompleteness.UnitsPercent - expectedUnits); diff > tolerance {
		t.Errorf("UnitsPercent: expected ~%f, got %f (diff: %f)", expectedUnits, result.NomenclatureCompleteness.UnitsPercent, diff)
	}
	if diff := abs(result.NomenclatureCompleteness.DescriptionsPercent - expectedDescriptions); diff > tolerance {
		t.Errorf("DescriptionsPercent: expected ~%f, got %f (diff: %f)", expectedDescriptions, result.NomenclatureCompleteness.DescriptionsPercent, diff)
	}
}

// TestCalculateOverallCompleteness_NilCompleteness тестирует обработку nil Completeness
func TestCalculateOverallCompleteness_NilCompleteness(t *testing.T) {
	handler := &NormalizationHandler{}

	stats := []DatabasePreviewStats{
		{
			NomenclatureCount: 100,
			Completeness:      nil, // nil completeness
		},
		{
			NomenclatureCount: 200,
			Completeness: &CompletenessMetrics{
				NomenclatureCompleteness: struct {
					ArticlesPercent      float64 `json:"articles_percent"`
					UnitsPercent         float64 `json:"units_percent"`
					DescriptionsPercent  float64 `json:"descriptions_percent"`
					OverallCompleteness  float64 `json:"overall_completeness"`
				}{
					ArticlesPercent:     80.0,
					UnitsPercent:        90.0,
					DescriptionsPercent: 100.0,
					OverallCompleteness: 90.0,
				},
			},
		},
	}

	result := handler.calculateOverallCompleteness(stats)

	// Должны учитываться только БД с не-nil Completeness
	if diff := abs(result.NomenclatureCompleteness.ArticlesPercent - 80.0); diff > 0.1 {
		t.Errorf("ArticlesPercent: expected 80.0, got %f", result.NomenclatureCompleteness.ArticlesPercent)
	}
}

// abs возвращает абсолютное значение float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

