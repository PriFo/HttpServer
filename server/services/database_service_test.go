package services

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"httpserver/database"
	apperrors "httpserver/server/errors"
)

// setupTestDatabaseService создает тестовый DatabaseService
func setupTestDatabaseService(t *testing.T) (*DatabaseService, *database.ServiceDB) {
	tempDir := t.TempDir()
	serviceDBPath := filepath.Join(tempDir, "service_test.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create test ServiceDB: %v", err)
	}

	service := NewDatabaseService(
		serviceDB,
		nil, // db
		nil, // normalizedDB
		"",  // currentDBPath
		"",  // currentNormalizedDBPath
		nil, // dbInfoCache
	)

	return service, serviceDB
}

// TestDatabaseService_CleanupPendingDatabases проверяет очистку старых pending databases
func TestDatabaseService_CleanupPendingDatabases(t *testing.T) {
	service, serviceDB := setupTestDatabaseService(t)
	defer serviceDB.Close()

	// Создаем несколько тестовых pending databases
	oldDate := time.Now().AddDate(0, 0, -10) // 10 дней назад
	newDate := time.Now().AddDate(0, 0, -1)  // 1 день назад

	// Создаем старую базу данных (должна быть удалена)
	oldDB, err := serviceDB.CreatePendingDatabase("/old/path.db", "old.db", 1000)
	if err != nil {
		t.Fatalf("Failed to create old pending database: %v", err)
	}

	// Обновляем дату обнаружения на старую и убеждаемся, что статус 'pending'
	// и client_id/project_id NULL (условия для удаления в CleanupOldPendingDatabases)
	_, err = serviceDB.Exec(
		"UPDATE pending_databases SET detected_at = ?, indexing_status = 'pending', client_id = NULL, project_id = NULL WHERE id = ?",
		oldDate, oldDB.ID,
	)
	if err != nil {
		t.Fatalf("Failed to update old database date: %v", err)
	}

	// Создаем новую базу данных (не должна быть удалена)
	newDB, err := serviceDB.CreatePendingDatabase("/new/path.db", "new.db", 2000)
	if err != nil {
		t.Fatalf("Failed to create new pending database: %v", err)
	}

	// Обновляем дату обнаружения на новую
	_, err = serviceDB.Exec(
		"UPDATE pending_databases SET detected_at = ? WHERE id = ?",
		newDate, newDB.ID,
	)
	if err != nil {
		t.Fatalf("Failed to update new database date: %v", err)
	}

	// Очищаем базы данных старше 5 дней
	deleted, err := service.CleanupPendingDatabases(5)
	if err != nil {
		t.Fatalf("CleanupPendingDatabases() error = %v", err)
	}

	// Должна быть удалена только старая база данных
	if deleted != 1 {
		t.Errorf("Expected 1 deleted database, got %d", deleted)
	}

	// Проверяем, что старая база данных удалена
	deletedDB, err := serviceDB.GetPendingDatabase(oldDB.ID)
	if err == nil && deletedDB != nil {
		t.Error("Old database should be deleted")
	}

	// Проверяем, что новая база данных осталась
	databases, err := serviceDB.GetPendingDatabases("")
	if err != nil {
		t.Fatalf("Failed to get pending databases: %v", err)
	}
	if len(databases) != 1 {
		t.Errorf("Expected 1 remaining database, got %d", len(databases))
	}
}

// TestDatabaseService_CleanupPendingDatabases_InvalidDays проверяет обработку невалидного количества дней
func TestDatabaseService_CleanupPendingDatabases_InvalidDays(t *testing.T) {
	service, serviceDB := setupTestDatabaseService(t)
	defer serviceDB.Close()

	// Тест с отрицательным количеством дней
	_, err := service.CleanupPendingDatabases(-1)
	if err == nil {
		t.Error("Expected error for negative days")
	}
	
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("Expected AppError, got %T", err)
	} else if appErr.StatusCode() != 400 {
		t.Errorf("Expected status code 400, got %d", appErr.StatusCode())
	}
}

// TestDatabaseService_CleanupPendingDatabases_NilServiceDB проверяет обработку nil serviceDB
func TestDatabaseService_CleanupPendingDatabases_NilServiceDB(t *testing.T) {
	service := NewDatabaseService(
		nil, // serviceDB
		nil, // db
		nil, // normalizedDB
		"",  // currentDBPath
		"",  // currentNormalizedDBPath
		nil, // dbInfoCache
	)

	_, err := service.CleanupPendingDatabases(7)
	if err == nil {
		t.Error("Expected error when serviceDB is nil")
	}
	
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("Expected AppError, got %T", err)
	} else if appErr.StatusCode() != 500 {
		t.Errorf("Expected status code 500, got %d", appErr.StatusCode())
	}
}
