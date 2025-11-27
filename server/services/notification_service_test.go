package services

import (
	"context"
	"fmt"
	"testing"
)

// TestNewNotificationService проверяет создание нового сервиса уведомлений
func TestNewNotificationService(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	if service == nil {
		t.Error("NewNotificationService() should not return nil")
	}
}

// TestNotificationService_AddNotification проверяет добавление уведомления
func TestNotificationService_AddNotification(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	notification, err := service.AddNotification(ctx, NotificationTypeInfo, "Test Title", "Test Message", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}
	if notification == nil {
		t.Error("AddNotification() should return notification")
	}
}

// TestNotificationService_AddNotification_WithIDs проверяет добавление уведомления с ID клиента и проекта
func TestNotificationService_AddNotification_WithIDs(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	clientID := 1
	projectID := 1
	_, err := service.AddNotification(ctx, NotificationTypeSuccess, "Test Title", "Test Message", &clientID, &projectID, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}
}

// TestNotificationService_GetNotifications проверяет получение уведомлений
func TestNotificationService_GetNotifications(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Добавляем уведомление
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test Title", "Test Message", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	// Получаем уведомления
	notifications, err := service.GetNotifications(ctx, 10, 0, false, nil, nil)
	if err != nil {
		t.Fatalf("GetNotifications() error = %v", err)
	}

	if len(notifications) == 0 {
		t.Error("Expected at least one notification")
	}
}

// TestNotificationService_GetNotifications_UnreadOnly проверяет получение только непрочитанных уведомлений
func TestNotificationService_GetNotifications_UnreadOnly(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Добавляем уведомление
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test Title", "Test Message", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	// Получаем только непрочитанные
	notifications, err := service.GetNotifications(ctx, 10, 0, true, nil, nil)
	if err != nil {
		t.Fatalf("GetNotifications() error = %v", err)
	}

	if len(notifications) == 0 {
		t.Error("Expected at least one unread notification")
	}
}

// TestNotificationService_MarkAsRead проверяет пометку уведомления как прочитанного
func TestNotificationService_MarkAsRead(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Добавляем уведомление
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test Title", "Test Message", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	// Получаем уведомления
	notifications, err := service.GetNotifications(ctx, 10, 0, false, nil, nil)
	if err != nil {
		t.Fatalf("GetNotifications() error = %v", err)
	}

	if len(notifications) == 0 {
		t.Fatal("Expected at least one notification")
	}

	// Помечаем как прочитанное
	err = service.MarkAsRead(ctx, notifications[0].ID)
	if err != nil {
		t.Fatalf("MarkAsRead() error = %v", err)
	}
}

// TestNotificationService_MarkAsRead_NotFound проверяет обработку несуществующего уведомления
func TestNotificationService_MarkAsRead_NotFound(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	err := service.MarkAsRead(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent notification")
	}
}

// TestNotificationService_MarkAllAsRead проверяет пометку всех уведомлений как прочитанных
func TestNotificationService_MarkAllAsRead(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Добавляем несколько уведомлений
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test 1", "Message 1", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	_, err = service.AddNotification(ctx, NotificationTypeWarning, "Test 2", "Message 2", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	// Помечаем все как прочитанные
	err = service.MarkAllAsRead(ctx, nil, nil)
	if err != nil {
		t.Fatalf("MarkAllAsRead() error = %v", err)
	}

	// Проверяем, что непрочитанных нет
	count, err := service.GetUnreadCount(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetUnreadCount() error = %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 unread notifications, got %d", count)
	}
}

// TestNotificationService_GetUnreadCount проверяет получение количества непрочитанных уведомлений
func TestNotificationService_GetUnreadCount(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Добавляем уведомление
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test Title", "Test Message", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	count, err := service.GetUnreadCount(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetUnreadCount() error = %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one unread notification")
	}
}

// TestNotificationService_DeleteNotification проверяет удаление уведомления
func TestNotificationService_DeleteNotification(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Добавляем уведомление
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test Title", "Test Message", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	// Получаем уведомления
	notifications, err := service.GetNotifications(ctx, 10, 0, false, nil, nil)
	if err != nil {
		t.Fatalf("GetNotifications() error = %v", err)
	}

	if len(notifications) == 0 {
		t.Fatal("Expected at least one notification")
	}

	// Удаляем уведомление
	err = service.DeleteNotification(ctx, notifications[0].ID)
	if err != nil {
		t.Fatalf("DeleteNotification() error = %v", err)
	}
}

// TestNotificationService_DeleteNotification_NotFound проверяет обработку несуществующего уведомления
func TestNotificationService_DeleteNotification_NotFound(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	err := service.DeleteNotification(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent notification")
	}
}

// TestNotificationService_ClearCache проверяет очистку кеша
func TestNotificationService_ClearCache(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Добавляем несколько уведомлений
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test 1", "Message 1", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	_, err = service.AddNotification(ctx, NotificationTypeWarning, "Test 2", "Message 2", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	// Проверяем, что кеш не пустой
	if service.GetCacheSize() == 0 {
		t.Error("Expected cache to have notifications")
	}

	// Очищаем кеш
	service.ClearCache()

	// Проверяем, что кеш пустой
	if service.GetCacheSize() != 0 {
		t.Errorf("Expected cache to be empty after ClearCache, got size %d", service.GetCacheSize())
	}

	// Проверяем, что данные все еще в БД
	notifications, err := service.GetNotifications(ctx, 10, 0, false, nil, nil)
	if err != nil {
		t.Fatalf("GetNotifications() error = %v", err)
	}

	if len(notifications) == 0 {
		t.Error("Expected notifications to still exist in database after cache clear")
	}
}

// TestNotificationService_GetCacheSize проверяет получение размера кеша
func TestNotificationService_GetCacheSize(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Изначально кеш пустой
	if service.GetCacheSize() != 0 {
		t.Errorf("Expected initial cache size to be 0, got %d", service.GetCacheSize())
	}

	// Добавляем уведомление
	_, err := service.AddNotification(ctx, NotificationTypeInfo, "Test", "Message", nil, nil, nil)
	if err != nil {
		t.Fatalf("AddNotification() error = %v", err)
	}

	// Проверяем, что размер кеша увеличился
	if service.GetCacheSize() != 1 {
		t.Errorf("Expected cache size to be 1, got %d", service.GetCacheSize())
	}
}

// TestNotificationService_SetMaxCacheSize проверяет установку максимального размера кеша
func TestNotificationService_SetMaxCacheSize(t *testing.T) {
	serviceDB := setupTestServiceDB(t)
	defer serviceDB.Close()

	service := NewNotificationService(serviceDB)
	ctx := context.Background()

	// Проверяем значение по умолчанию
	if service.GetMaxCacheSize() != 1000 {
		t.Errorf("Expected default max cache size to be 1000, got %d", service.GetMaxCacheSize())
	}

	// Устанавливаем новый размер
	service.SetMaxCacheSize(500)
	if service.GetMaxCacheSize() != 500 {
		t.Errorf("Expected max cache size to be 500, got %d", service.GetMaxCacheSize())
	}

	// Создаем больше уведомлений, чем новый максимум
	for i := 1; i <= 600; i++ {
		_, err := service.AddNotification(ctx, NotificationTypeInfo, fmt.Sprintf("Test %d", i), "Message", nil, nil, nil)
		if err != nil {
			t.Fatalf("AddNotification() error = %v", err)
		}
	}

	// Проверяем, что кеш обрезан до максимума
	if service.GetCacheSize() > 500 {
		t.Errorf("Expected cache size to be at most 500, got %d", service.GetCacheSize())
	}

	// Тест с минимальным значением
	service.SetMaxCacheSize(0)
	if service.GetMaxCacheSize() < 100 {
		t.Errorf("Expected max cache size to be at least 100 (minimum), got %d", service.GetMaxCacheSize())
	}

	// Тест с очень большим значением
	service.SetMaxCacheSize(50000)
	if service.GetMaxCacheSize() > 10000 {
		t.Errorf("Expected max cache size to be at most 10000 (maximum), got %d", service.GetMaxCacheSize())
	}
}


