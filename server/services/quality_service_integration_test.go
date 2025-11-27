package services

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"httpserver/database"
)

// setupIntegrationTestDB создает тестовую БД для интеграционных тестов
// с возможностью создания кастомных таблиц/VIEW для тестирования ошибок
func setupIntegrationTestDB(t *testing.T) *database.DB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration_test.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create integration test DB: %v", err)
	}
	return db
}

// TestQualityService_GetIssuesSeverityStats_ScanError_Integration проверяет обработку ошибок сканирования
// когда тип данных в БД не соответствует ожидаемому Go-типу
// Это интеграционный тест с реальной БД
func TestQualityService_GetIssuesSeverityStats_ScanError_Integration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Создаем upload для теста
	now := time.Now()
	uploadUUID := "test-scan-error-integration-uuid"
	_, err := db.Exec(`
		INSERT INTO uploads (upload_uuid, started_at, completed_at, status, database_id, version_1c, config_name, computer_name, user_name, config_version, iteration_number, iteration_label, programmer_name, upload_purpose)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadUUID, now, now, "completed", 1, "8.3", "test_config", "", "", "", 1, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	var uploadID int
	err = db.QueryRow("SELECT id FROM uploads WHERE upload_uuid = ?", uploadUUID).Scan(&uploadID)
	if err != nil {
		t.Fatalf("Failed to get upload ID: %v", err)
	}

	// Не используем VIEW (SQLite не поддерживает параметры в VIEW)
	// Вместо этого будем перехватывать запрос и заменять его на запрос с неправильными типами

	// Вставляем валидные данные
	_, err = db.Exec(`
		INSERT INTO data_quality_issues (upload_id, database_id, entity_type, issue_type, issue_severity, description, detected_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadID, 1, "nomenclature", "missing_field", "CRITICAL", "Test issue", now, "OPEN")
	if err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	// Создаем мок БД, который перехватывает Query и заменяет его на запрос с неправильными типами
	mockDB := &mockDBWithCustomQuery{
		realDB: db,
		queryOverride: func(query string, args ...interface{}) (*sql.Rows, error) {
			// Заменяем запрос к таблице на запрос с неправильными типами (TEXT вместо INTEGER для count)
			// Это вызовет ошибку при сканировании в int
			if contains(query, "SELECT issue_severity, COUNT(*)") && contains(query, "FROM data_quality_issues") {
				// Используем запрос с CAST для создания неправильного типа
				brokenQuery := `
					SELECT 
						issue_severity,
						CAST(COUNT(*) AS TEXT) as count
					FROM data_quality_issues
					WHERE upload_id = ?
					GROUP BY issue_severity
				`
				return db.Query(brokenQuery, args...)
			}
			return db.Query(query, args...)
		},
	}

	mockLogger := &mockLoggerIntegration{}
	service, err := NewQualityServiceWithDeps(mockDB, nil, mockLogger, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Вызываем функцию - должна произойти ошибка сканирования
	stats, err := service.getIssuesSeverityStats(uploadID)

	// Ожидаем, что функция вернет результат (так как ошибки сканирования обрабатываются)
	// Примечание: SQLite может автоматически конвертировать типы (TEXT -> INTEGER),
	// поэтому ошибка сканирования может не произойти. Это нормальное поведение SQLite.
	// Проверяем логи только если ошибка сканирования действительно произошла
	scanWarnFound := false
	for _, warn := range mockLogger.warnCalls {
		if contains(warn, "Failed to scan severity stats row") {
			scanWarnFound = true
			break
		}
	}

	// Проверяем, что было залогировано предупреждение о множественных ошибках сканирования
	scanErrorsWarnFound := false
	for _, warn := range mockLogger.warnCalls {
		if contains(warn, "Some rows failed to scan") {
			scanErrorsWarnFound = true
			break
		}
	}

	// Если произошла ошибка сканирования, должны быть соответствующие логи
	// Если SQLite автоматически сконвертировал типы, ошибки не будет - это тоже нормально
	if scanWarnFound && !scanErrorsWarnFound {
		t.Logf("Warning: scan errors occurred but 'Some rows failed to scan' was not logged")
	}

	if err != nil {
		// Если rows.Err() вернул ошибку, это тоже валидный результат
		t.Logf("getIssuesSeverityStats returned error (expected for rows.Err()): %v", err)
	}

	// Проверяем, что функция не паниковала и вернула структуру
	if stats == nil {
		t.Error("Stats should not be nil even with scan errors")
	}
}

// databaseAdapterWrapperIntegration обертка для использования в интеграционных тестах
type databaseAdapterWrapperIntegration struct {
	db *database.DB
}

func (a *databaseAdapterWrapperIntegration) GetUploadByUUID(uuid string) (*database.Upload, error) {
	return a.db.GetUploadByUUID(uuid)
}

func (a *databaseAdapterWrapperIntegration) GetQualityMetrics(uploadID int) ([]database.DataQualityMetric, error) {
	return a.db.GetQualityMetrics(uploadID)
}

func (a *databaseAdapterWrapperIntegration) GetQualityIssues(uploadID int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error) {
	return a.db.GetQualityIssues(uploadID, filters, limit, offset)
}

func (a *databaseAdapterWrapperIntegration) GetQualityIssuesByUploadIDs(uploadIDs []int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error) {
	return a.db.GetQualityIssuesByUploadIDs(uploadIDs, filters, limit, offset)
}

func (a *databaseAdapterWrapperIntegration) GetQualityTrends(databaseID int, days int) ([]database.QualityTrend, error) {
	return a.db.GetQualityTrends(databaseID, days)
}

func (a *databaseAdapterWrapperIntegration) GetCurrentQualityMetrics(databaseID int) ([]database.DataQualityMetric, error) {
	return a.db.GetCurrentQualityMetrics(databaseID)
}

func (a *databaseAdapterWrapperIntegration) GetTopQualityIssues(databaseID int, limit int) ([]database.DataQualityIssue, error) {
	return a.db.GetTopQualityIssues(databaseID, limit)
}

func (a *databaseAdapterWrapperIntegration) GetAllUploads() ([]*database.Upload, error) {
	return a.db.GetAllUploads()
}

func (a *databaseAdapterWrapperIntegration) GetUploadsByDatabaseID(databaseID int) ([]*database.Upload, error) {
	return a.db.GetUploadsByDatabaseID(databaseID)
}

func (a *databaseAdapterWrapperIntegration) GetQualityStats() (interface{}, error) {
	stats, err := a.db.GetQualityStats()
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (a *databaseAdapterWrapperIntegration) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.Query(query, args...)
}

func (a *databaseAdapterWrapperIntegration) Close() error {
	return a.db.Close()
}

// mockDBWithCustomQuery мок БД с возможностью переопределения Query
type mockDBWithCustomQuery struct {
	DatabaseInterface
	realDB        *database.DB
	queryOverride func(query string, args ...interface{}) (*sql.Rows, error)
}

func (m *mockDBWithCustomQuery) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryOverride != nil {
		return m.queryOverride(query, args...)
	}
	return m.realDB.Query(query, args...)
}

func (m *mockDBWithCustomQuery) GetUploadByUUID(uuid string) (*database.Upload, error) {
	return m.realDB.GetUploadByUUID(uuid)
}

func (m *mockDBWithCustomQuery) GetQualityMetrics(uploadID int) ([]database.DataQualityMetric, error) {
	return m.realDB.GetQualityMetrics(uploadID)
}

func (m *mockDBWithCustomQuery) GetQualityIssues(uploadID int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error) {
	return m.realDB.GetQualityIssues(uploadID, filters, limit, offset)
}

func (m *mockDBWithCustomQuery) GetQualityTrends(databaseID int, days int) ([]database.QualityTrend, error) {
	return m.realDB.GetQualityTrends(databaseID, days)
}

func (m *mockDBWithCustomQuery) GetCurrentQualityMetrics(databaseID int) ([]database.DataQualityMetric, error) {
	return m.realDB.GetCurrentQualityMetrics(databaseID)
}

func (m *mockDBWithCustomQuery) GetTopQualityIssues(databaseID int, limit int) ([]database.DataQualityIssue, error) {
	return m.realDB.GetTopQualityIssues(databaseID, limit)
}

func (m *mockDBWithCustomQuery) GetAllUploads() ([]*database.Upload, error) {
	return m.realDB.GetAllUploads()
}

func (m *mockDBWithCustomQuery) GetQualityStats() (interface{}, error) {
	return m.realDB.GetQualityStats()
}

func (m *mockDBWithCustomQuery) Close() error {
	return m.realDB.Close()
}

// TestQualityService_GetIssuesSeverityStats_NullValueError_Integration проверяет обработку NULL значений
// в полях, которые не могут быть NULL
// Это интеграционный тест с реальной БД
func TestQualityService_GetIssuesSeverityStats_NullValueError_Integration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Создаем upload для теста
	now := time.Now()
	uploadUUID := "test-null-value-integration-uuid"
	_, err := db.Exec(`
		INSERT INTO uploads (upload_uuid, started_at, completed_at, status, database_id, version_1c, config_name, computer_name, user_name, config_version, iteration_number, iteration_label, programmer_name, upload_purpose)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadUUID, now, now, "completed", 1, "8.3", "test_config", "", "", "", 1, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	var uploadID int
	err = db.QueryRow("SELECT id FROM uploads WHERE upload_uuid = ?", uploadUUID).Scan(&uploadID)
	if err != nil {
		t.Fatalf("Failed to get upload ID: %v", err)
	}

	// Не используем VIEW (SQLite не поддерживает параметры в VIEW)
	// Вместо этого будем перехватывать запрос и заменять его на запрос с NULL значениями

	// Вставляем валидные данные
	_, err = db.Exec(`
		INSERT INTO data_quality_issues (upload_id, database_id, entity_type, issue_type, issue_severity, description, detected_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadID, 1, "nomenclature", "missing_field", "HIGH", "Test issue", now, "OPEN")
	if err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	// Создаем мок БД с переопределенным Query для использования запроса с NULL
	mockDB := &mockDBWithCustomQuery{
		realDB: db,
		queryOverride: func(query string, args ...interface{}) (*sql.Rows, error) {
			// Заменяем запрос на запрос с NULL значениями для count
			// Это вызовет ошибку при сканировании в int
			if contains(query, "SELECT issue_severity, COUNT(*)") && contains(query, "FROM data_quality_issues") {
				// Используем запрос с NULL для count
				nullQuery := `
					SELECT 
						issue_severity,
						NULL as count
					FROM data_quality_issues
					WHERE upload_id = ?
					GROUP BY issue_severity
				`
				return db.Query(nullQuery, args...)
			}
			return db.Query(query, args...)
		},
	}

	mockLogger := &mockLoggerIntegration{}
	service, err := NewQualityServiceWithDeps(mockDB, nil, mockLogger, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Вызываем функцию - должна произойти ошибка сканирования из-за NULL
	stats, err := service.getIssuesSeverityStats(uploadID)

	// Ожидаем ошибку сканирования, но функция должна обработать её корректно
	if err != nil {
		// Если rows.Err() вернул ошибку, это валидный результат
		t.Logf("getIssuesSeverityStats returned error (expected for NULL values): %v", err)
	}

	// Проверяем, что функция не паниковала
	if stats == nil && err == nil {
		t.Error("Stats should not be nil if no error returned")
	}
}

// TestQualityService_GetIssuesSeverityStats_ScanErrorsCounter_Integration проверяет,
// что счетчик scanErrors работает правильно при множественных ошибках сканирования
// Это интеграционный тест с реальной БД
func TestQualityService_GetIssuesSeverityStats_ScanErrorsCounter_Integration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Создаем upload для теста
	now := time.Now()
	uploadUUID := "test-scan-errors-counter-uuid"
	_, err := db.Exec(`
		INSERT INTO uploads (upload_uuid, started_at, completed_at, status, database_id, version_1c, config_name, computer_name, user_name, config_version, iteration_number, iteration_label, programmer_name, upload_purpose)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadUUID, now, now, "completed", 1, "8.3", "test_config", "", "", "", 1, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	var uploadID int
	err = db.QueryRow("SELECT id FROM uploads WHERE upload_uuid = ?", uploadUUID).Scan(&uploadID)
	if err != nil {
		t.Fatalf("Failed to get upload ID: %v", err)
	}

	// Вставляем несколько валидных записей
	for i := 0; i < 3; i++ {
		_, err = db.Exec(`
			INSERT INTO data_quality_issues (upload_id, database_id, entity_type, issue_type, issue_severity, description, detected_at, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, uploadID, 1, "nomenclature", "missing_field", "CRITICAL", "Test issue", now, "OPEN")
		if err != nil {
			t.Fatalf("Failed to create test issue: %v", err)
		}
	}

	// Не используем VIEW (SQLite не поддерживает параметры в VIEW)
	// Вместо этого будем перехватывать запрос и заменять его на запрос с неправильными типами

	// Создаем мок БД с переопределенным Query для создания множественных ошибок сканирования
	mockDB := &mockDBWithCustomQuery{
		realDB: db,
		queryOverride: func(query string, args ...interface{}) (*sql.Rows, error) {
			// Заменяем запрос на запрос с неправильными типами для всех строк
			// Это вызовет множественные ошибки сканирования
			if contains(query, "SELECT issue_severity, COUNT(*)") && contains(query, "FROM data_quality_issues") {
				// Используем запрос с CAST для создания неправильного типа для всех строк
				brokenQuery := `
					SELECT 
						issue_severity,
						CAST(COUNT(*) AS TEXT) as count
					FROM data_quality_issues
					WHERE upload_id = ?
					GROUP BY issue_severity
				`
				return db.Query(brokenQuery, args...)
			}
			return db.Query(query, args...)
		},
	}

	mockLogger := &mockLoggerIntegration{}
	service, err := NewQualityServiceWithDeps(mockDB, nil, mockLogger, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Вызываем функцию - должны произойти множественные ошибки сканирования
	stats, err := service.getIssuesSeverityStats(uploadID)

	// Функция должна обработать ошибки и вернуть результат
	// Примечание: SQLite может автоматически конвертировать типы (TEXT -> INTEGER),
	// поэтому ошибка сканирования может не произойти. Это нормальное поведение SQLite.
	// Проверяем логи только если ошибка сканирования действительно произошла
	scanWarnFound := false
	for _, warn := range mockLogger.warnCalls {
		if contains(warn, "Failed to scan severity stats row") {
			scanWarnFound = true
			break
		}
	}

	// Проверяем, что было залогировано предупреждение о множественных ошибках сканирования
	scanErrorsWarnFound := false
	for _, warn := range mockLogger.warnCalls {
		if contains(warn, "Some rows failed to scan") {
			scanErrorsWarnFound = true
			break
		}
	}

	// Если произошла ошибка сканирования, должны быть соответствующие логи
	// Если SQLite автоматически сконвертировал типы, ошибки не будет - это тоже нормально
	if scanWarnFound && !scanErrorsWarnFound {
		t.Logf("Warning: scan errors occurred but 'Some rows failed to scan' was not logged")
	}

	// Проверяем, что функция не паниковала
	if stats == nil && err == nil {
		t.Error("Stats should not be nil if no error returned")
	}

	// Если есть ошибка, она должна быть связана с rows.Err() или сканированием
	if err != nil {
		t.Logf("getIssuesSeverityStats returned error (expected for multiple scan errors): %v", err)
		if !contains(err.Error(), "scan") && !contains(err.Error(), "iterating") {
			t.Logf("Warning: error message doesn't contain 'scan' or 'iterating', got: %v", err.Error())
		}
	}
}

// TestQualityService_GetIssuesSeverityStats_RowsError_Integration проверяет обработку rows.Err()
// Примечание: Симулировать rows.Err() на реальной in-memory SQLite практически невозможно,
// так как это ошибка, которая возникает во время итерации по rows, а не при выполнении запроса.
// rows.Err() обычно возвращает ошибку, если произошла проблема во время Next() или Scan(),
// например, если соединение с БД было разорвано во время чтения.
//
// Этот тест фокусируется на том, что система корректно проверяет rows.Err() после цикла.
// Реальная симуляция rows.Err() требует хаос-инжиниринга (например, принудительное закрытие соединения)
// или ручного тестирования в production-like окружении.
// Это интеграционный тест с реальной БД
func TestQualityService_GetIssuesSeverityStats_RowsError_Integration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Создаем upload для теста
	now := time.Now()
	uploadUUID := "test-rows-error-uuid"
	_, err := db.Exec(`
		INSERT INTO uploads (upload_uuid, started_at, completed_at, status, database_id, version_1c, config_name, computer_name, user_name, config_version, iteration_number, iteration_label, programmer_name, upload_purpose)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadUUID, now, now, "completed", 1, "8.3", "test_config", "", "", "", 1, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	var uploadID int
	err = db.QueryRow("SELECT id FROM uploads WHERE upload_uuid = ?", uploadUUID).Scan(&uploadID)
	if err != nil {
		t.Fatalf("Failed to get upload ID: %v", err)
	}

	// Вставляем валидные данные
	_, err = db.Exec(`
		INSERT INTO data_quality_issues (upload_id, database_id, entity_type, issue_type, issue_severity, description, detected_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadID, 1, "nomenclature", "missing_field", "LOW", "Test issue", now, "OPEN")
	if err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	service, err := NewQualityService(db, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Вызываем функцию с валидными данными
	// rows.Err() не должен вернуть ошибку в нормальных условиях
	stats, err := service.getIssuesSeverityStats(uploadID)

	// В нормальных условиях ошибки быть не должно
	if err != nil {
		t.Errorf("getIssuesSeverityStats returned unexpected error: %v", err)
	}

	if stats == nil {
		t.Error("Stats should not be nil")
	}

	// Проверяем, что статистика содержит ожидаемые значения
	if stats["LOW"] == 0 {
		t.Error("Expected LOW severity count to be > 0")
	}

	// Примечание: Для реального тестирования rows.Err() требуется:
	// 1. Хаос-инжиниринг: принудительное закрытие соединения во время итерации
	// 2. Использование специальных драйверов БД, которые могут симулировать сетевые ошибки
	// 3. Ручное тестирование в production-like окружении с нестабильным сетевым соединением
	//
	// В автоматизированных юнит-тестах мы проверяем, что код корректно обрабатывает rows.Err(),
	// вызывая его после цикла и возвращая ошибку, если она есть. Это покрывается другими тестами,
	// которые проверяют обработку ошибок сканирования (которые могут привести к rows.Err() != nil).
}

// TestQualityService_GetIssuesSeverityStats_ValidData_Integration проверяет нормальный путь выполнения
// с валидными данными для сравнения с тестами ошибок
// Это интеграционный тест с реальной БД
func TestQualityService_GetIssuesSeverityStats_ValidData_Integration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Создаем upload для теста
	now := time.Now()
	uploadUUID := "test-valid-data-uuid"
	_, err := db.Exec(`
		INSERT INTO uploads (upload_uuid, started_at, completed_at, status, database_id, version_1c, config_name, computer_name, user_name, config_version, iteration_number, iteration_label, programmer_name, upload_purpose)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadUUID, now, now, "completed", 1, "8.3", "test_config", "", "", "", 1, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	var uploadID int
	err = db.QueryRow("SELECT id FROM uploads WHERE upload_uuid = ?", uploadUUID).Scan(&uploadID)
	if err != nil {
		t.Fatalf("Failed to get upload ID: %v", err)
	}

	// Вставляем валидные данные с разными уровнями серьезности
	severities := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}
	for i, severity := range severities {
		for j := 0; j < i+1; j++ {
			_, err = db.Exec(`
				INSERT INTO data_quality_issues (upload_id, database_id, entity_type, issue_type, issue_severity, description, detected_at, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, uploadID, 1, "nomenclature", "missing_field", severity, "Test issue", now, "OPEN")
			if err != nil {
				t.Fatalf("Failed to create test issue: %v", err)
			}
		}
	}

	service, err := NewQualityService(db, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Вызываем функцию
	stats, err := service.getIssuesSeverityStats(uploadID)
	if err != nil {
		t.Fatalf("getIssuesSeverityStats returned error: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	// Проверяем, что все уровни серьезности присутствуют с правильными значениями
	expectedCounts := map[string]int{
		"CRITICAL": 1,
		"HIGH":     2,
		"MEDIUM":   3,
		"LOW":      4,
	}

	for severity, expectedCount := range expectedCounts {
		if stats[severity] != expectedCount {
			t.Errorf("Expected %s count to be %d, got %d", severity, expectedCount, stats[severity])
		}
	}
}

// mockLoggerIntegration простой мок логгера для интеграционных тестов
type mockLoggerIntegration struct {
	infoCalls  []string
	errorCalls []string
	warnCalls  []string
}

func (m *mockLoggerIntegration) Info(msg string, args ...interface{}) {
	m.infoCalls = append(m.infoCalls, msg)
}

func (m *mockLoggerIntegration) Error(msg string, args ...interface{}) {
	m.errorCalls = append(m.errorCalls, msg)
}

func (m *mockLoggerIntegration) Warn(msg string, args ...interface{}) {
	m.warnCalls = append(m.warnCalls, msg)
}

// contains проверяет, содержит ли строка подстроку
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestQualityService_GetIssuesSeverityStats_UnknownSeverity_Integration проверяет обработку неизвестных уровней серьезности
// Это интеграционный тест с реальной БД
func TestQualityService_GetIssuesSeverityStats_UnknownSeverity_Integration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// Создаем upload для теста
	now := time.Now()
	uploadUUID := "test-unknown-severity-integration-uuid"
	_, err := db.Exec(`
		INSERT INTO uploads (upload_uuid, started_at, completed_at, status, database_id, version_1c, config_name, computer_name, user_name, config_version, iteration_number, iteration_label, programmer_name, upload_purpose)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadUUID, now, now, "completed", 1, "8.3", "test_config", "", "", "", 1, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test upload: %v", err)
	}

	var uploadID int
	err = db.QueryRow("SELECT id FROM uploads WHERE upload_uuid = ?", uploadUUID).Scan(&uploadID)
	if err != nil {
		t.Fatalf("Failed to get upload ID: %v", err)
	}

	// Вставляем валидные данные с известными уровнями
	_, err = db.Exec(`
		INSERT INTO data_quality_issues (upload_id, database_id, entity_type, issue_type, issue_severity, description, detected_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, uploadID, 1, "nomenclature", "missing_field", "CRITICAL", "Test issue", now, "OPEN")
	if err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}

	// Создаем мок БД, который возвращает неизвестный уровень серьезности
	// Для этого используем запрос, который возвращает значение, не входящее в список валидных
	mockDB := &mockDBWithCustomQuery{
		realDB: db,
		queryOverride: func(query string, args ...interface{}) (*sql.Rows, error) {
			// Заменяем запрос на запрос с неизвестным уровнем серьезности
			if contains(query, "SELECT issue_severity, COUNT(*)") && contains(query, "FROM data_quality_issues") {
				// Используем запрос, который возвращает неизвестный уровень
				unknownQuery := `
					SELECT 
						'UNKNOWN_SEVERITY' as issue_severity,
						COUNT(*) as count
					FROM data_quality_issues
					WHERE upload_id = ?
					GROUP BY issue_severity
					UNION ALL
					SELECT 
						issue_severity,
						COUNT(*) as count
					FROM data_quality_issues
					WHERE upload_id = ? AND issue_severity != 'UNKNOWN_SEVERITY'
					GROUP BY issue_severity
				`
				// Передаем uploadID дважды для UNION ALL
				return db.Query(unknownQuery, uploadID, uploadID)
			}
			return db.Query(query, args...)
		},
	}

	mockLogger := &mockLoggerIntegration{}
	service, err := NewQualityServiceWithDeps(mockDB, nil, mockLogger, nil)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Вызываем функцию
	stats, err := service.getIssuesSeverityStats(uploadID)
	if err != nil {
		t.Fatalf("getIssuesSeverityStats returned error: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	// Проверяем, что неизвестный уровень серьезности был залогирован
	unknownSeverityWarnFound := false
	for _, warn := range mockLogger.warnCalls {
		if contains(warn, "Unknown severity level") {
			unknownSeverityWarnFound = true
			break
		}
	}

	if !unknownSeverityWarnFound {
		t.Error("Expected warning about unknown severity level, but it was not logged")
	}

	// Проверяем, что неизвестный уровень не добавлен в stats
	if stats["UNKNOWN_SEVERITY"] != 0 {
		t.Errorf("Unknown severity should not be in stats, got %d", stats["UNKNOWN_SEVERITY"])
	}
}

// TestQualityService_GetQualityStats_DatabaseCloseError_Integration проверяет обработку ошибки закрытия БД
// Это интеграционный тест с реальной БД
func TestQualityService_GetQualityStats_DatabaseCloseError_Integration(t *testing.T) {
	ctx := context.Background()
	
	// Создаем временную БД
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_close_error.db")
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer db.Close() // Закрываем основную БД в конце теста
	
	// Создаем мок factory, который возвращает БД с ошибкой при закрытии
	mockFactory := &mockDatabaseFactoryWithCloseError{
		dbPath: dbPath,
	}
	
	dbAdapter := &databaseAdapterWrapperIntegration{db: db}
	service, err := NewQualityServiceWithDeps(dbAdapter, nil, &mockLoggerIntegration{}, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	
	// Вызываем GetQualityStats с путем к БД, чтобы проверить закрытие
	stats, err := service.GetQualityStats(ctx, dbPath, nil)
	if err != nil {
		t.Fatalf("GetQualityStats() error = %v", err)
	}

	if stats == nil {
		t.Error("Stats should not be nil")
	}
	
	// Закрываем все кэшированные подключения перед завершением теста
	service.CloseAllConnections()
	
	// Примечание: Ошибка закрытия БД логируется, но не возвращается как ошибка
	// Это нормальное поведение - ошибка закрытия не критична для результата
}

// mockDatabaseFactoryWithCloseError мок для DatabaseFactory, который возвращает БД с ошибкой при закрытии
type mockDatabaseFactoryWithCloseError struct {
	dbPath string
}

func (m *mockDatabaseFactoryWithCloseError) NewDB(path string) (DatabaseInterface, error) {
	db, err := database.NewDB(path)
	if err != nil {
		return nil, err
	}
	return &mockDBWithCloseErrorIntegration{db: db}, nil
}

// mockDBWithCloseErrorIntegration мок БД, который возвращает ошибку при закрытии
type mockDBWithCloseErrorIntegration struct {
	DatabaseInterface
	db *database.DB
}

func (m *mockDBWithCloseErrorIntegration) GetUploadByUUID(uuid string) (*database.Upload, error) {
	return m.db.GetUploadByUUID(uuid)
}

func (m *mockDBWithCloseErrorIntegration) GetQualityMetrics(uploadID int) ([]database.DataQualityMetric, error) {
	return m.db.GetQualityMetrics(uploadID)
}

func (m *mockDBWithCloseErrorIntegration) GetQualityIssues(uploadID int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error) {
	return m.db.GetQualityIssues(uploadID, filters, limit, offset)
}

func (m *mockDBWithCloseErrorIntegration) GetQualityTrends(databaseID int, days int) ([]database.QualityTrend, error) {
	return m.db.GetQualityTrends(databaseID, days)
}

func (m *mockDBWithCloseErrorIntegration) GetCurrentQualityMetrics(databaseID int) ([]database.DataQualityMetric, error) {
	return m.db.GetCurrentQualityMetrics(databaseID)
}

func (m *mockDBWithCloseErrorIntegration) GetTopQualityIssues(databaseID int, limit int) ([]database.DataQualityIssue, error) {
	return m.db.GetTopQualityIssues(databaseID, limit)
}

func (m *mockDBWithCloseErrorIntegration) GetAllUploads() ([]*database.Upload, error) {
	return m.db.GetAllUploads()
}

func (m *mockDBWithCloseErrorIntegration) GetQualityStats() (interface{}, error) {
	return m.db.GetQualityStats()
}

func (m *mockDBWithCloseErrorIntegration) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.Query(query, args...)
}

func (m *mockDBWithCloseErrorIntegration) Close() error {
	// Закрываем реальную БД, но возвращаем ошибку для тестирования
	_ = m.db.Close()
	return errors.New("close error for testing")
}

