package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"httpserver/database"
)

// NotificationType тип уведомления
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

// Notification представляет уведомление
type Notification struct {
	ID        int             `json:"id"`
	Type      NotificationType `json:"type"`
	Title     string          `json:"title"`
	Message   string          `json:"message"`
	Timestamp time.Time       `json:"timestamp"`
	Read      bool            `json:"read"`
	ClientID  *int            `json:"client_id,omitempty"`
	ProjectID *int            `json:"project_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationService сервис для управления уведомлениями
// Все операции выполняются через ServiceDB, fallback на память удален.
// ServiceDB обязателен и не может быть nil.
type NotificationService struct {
	serviceDB *database.ServiceDB // Обязательное поле, не может быть nil
	mu        sync.RWMutex        // Мьютекс для защиты кеша в памяти
	notifications []Notification  // Кеш уведомлений в памяти (только для оптимизации, не основной источник данных)
	maxNotifications int          // Максимальное количество уведомлений в кеше
}

// NewNotificationService создает новый сервис уведомлений
// serviceDB обязателен и не может быть nil
func NewNotificationService(serviceDB *database.ServiceDB) *NotificationService {
	if serviceDB == nil {
		panic("ServiceDB is required for NotificationService and cannot be nil")
	}
	return &NotificationService{
		serviceDB:        serviceDB,
		notifications:    make([]Notification, 0),
		maxNotifications: 1000, // Максимальное количество уведомлений в памяти (для кеширования)
	}
}

// AddNotification добавляет новое уведомление и возвращает созданное уведомление
// ServiceDB обязателен, поэтому всегда используем БД
func (ns *NotificationService) AddNotification(ctx context.Context, notificationType NotificationType, title, message string, clientID, projectID *int, metadata map[string]interface{}) (*Notification, error) {
	// Сохраняем в БД (serviceDB гарантированно не nil)
	dbID, err := ns.serviceDB.SaveNotification(
		string(notificationType),
		title,
		message,
		clientID,
		projectID,
		metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save notification to database: %w", err)
	}

	// Создаем объект уведомления с данными, которые мы уже знаем
	// Timestamp устанавливается в БД как CURRENT_TIMESTAMP, используем текущее время
	notification := Notification{
		ID:        dbID,
		Type:      notificationType,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(), // БД установит точное время, но для кеша используем текущее
		Read:      false,
		ClientID:  clientID,
		ProjectID: projectID,
		Metadata:  metadata,
	}

	// Добавляем в память для кеширования
	ns.mu.Lock()
	ns.notifications = append([]Notification{notification}, ns.notifications...)
	if len(ns.notifications) > ns.maxNotifications {
		ns.notifications = ns.notifications[:ns.maxNotifications]
	}
	ns.mu.Unlock()

	return &notification, nil
}

// GetNotifications возвращает список уведомлений
// ServiceDB обязателен, поэтому всегда используем БД
func (ns *NotificationService) GetNotifications(ctx context.Context, limit, offset int, unreadOnly bool, clientID, projectID *int) ([]Notification, error) {
	// Получаем уведомления из БД (serviceDB гарантированно не nil)
	dbNotifications, err := ns.serviceDB.GetNotificationsFromDB(limit, offset, unreadOnly, clientID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications from database: %w", err)
	}

	// Конвертируем из БД в структуру Notification
	notifications := make([]Notification, 0, len(dbNotifications))
	for _, dbNotif := range dbNotifications {
		var notifID int
		if id, ok := dbNotif["id"].(int); ok {
			notifID = id
		} else if id, ok := dbNotif["id"].(float64); ok {
			notifID = int(id)
		}

		var readVal bool
		if read, ok := dbNotif["read"].(bool); ok {
			readVal = read
		} else if read, ok := dbNotif["read"].(int); ok {
			readVal = read != 0
		} else if read, ok := dbNotif["read"].(float64); ok {
			readVal = read != 0
		}

		// Безопасное извлечение timestamp
		var timestamp time.Time
		if ts, ok := dbNotif["timestamp"].(time.Time); ok {
			timestamp = ts
		} else if tsStr, ok := dbNotif["timestamp"].(string); ok {
			// Пытаемся распарсить строку, если timestamp пришел как строка
			if parsed, err := time.Parse(time.RFC3339, tsStr); err == nil {
				timestamp = parsed
			} else if parsed, err := time.Parse("2006-01-02 15:04:05", tsStr); err == nil {
				timestamp = parsed
			} else {
				timestamp = time.Now() // Fallback на текущее время
			}
		} else {
			timestamp = time.Now() // Fallback на текущее время
		}

		// Безопасное извлечение строковых полей
		var notifType, title, message string
		if t, ok := dbNotif["type"].(string); ok {
			notifType = t
		}
		if t, ok := dbNotif["title"].(string); ok {
			title = t
		}
		if m, ok := dbNotif["message"].(string); ok {
			message = m
		}

		notification := Notification{
			ID:        notifID,
			Type:      NotificationType(notifType),
			Title:     title,
			Message:   message,
			Timestamp: timestamp,
			Read:      readVal,
		}

		if clientIDVal, ok := dbNotif["client_id"].(*int); ok && clientIDVal != nil {
			notification.ClientID = clientIDVal
		} else if clientIDVal, ok := dbNotif["client_id"].(int); ok {
			notification.ClientID = &clientIDVal
		} else if clientIDVal, ok := dbNotif["client_id"].(float64); ok {
			val := int(clientIDVal)
			notification.ClientID = &val
		}

		if projectIDVal, ok := dbNotif["project_id"].(*int); ok && projectIDVal != nil {
			notification.ProjectID = projectIDVal
		} else if projectIDVal, ok := dbNotif["project_id"].(int); ok {
			notification.ProjectID = &projectIDVal
		} else if projectIDVal, ok := dbNotif["project_id"].(float64); ok {
			val := int(projectIDVal)
			notification.ProjectID = &val
		}

		if metadata, ok := dbNotif["metadata"].(map[string]interface{}); ok {
			notification.Metadata = metadata
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// GetNotificationsCount возвращает общее количество уведомлений с учетом фильтров
// ServiceDB обязателен, поэтому всегда используем БД
func (ns *NotificationService) GetNotificationsCount(ctx context.Context, unreadOnly bool, clientID, projectID *int) (int, error) {
	// Получаем из БД (serviceDB гарантированно не nil)
	count, err := ns.serviceDB.GetNotificationsCount(unreadOnly, clientID, projectID)
	if err != nil {
		return 0, fmt.Errorf("failed to get notifications count from database: %w", err)
	}
	return count, nil
}

// MarkAsRead помечает уведомление как прочитанное
// ServiceDB обязателен, поэтому всегда используем БД
func (ns *NotificationService) MarkAsRead(ctx context.Context, notificationID int) error {
	// Обновляем в БД (serviceDB гарантированно не nil)
	err := ns.serviceDB.MarkNotificationAsRead(notificationID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read in database: %w", err)
	}

	// Также обновляем в памяти для консистентности кеша
	ns.mu.Lock()
	for i := range ns.notifications {
		if ns.notifications[i].ID == notificationID {
			ns.notifications[i].Read = true
			ns.mu.Unlock()
			return nil
		}
	}
	ns.mu.Unlock()

	// Уведомление может отсутствовать в кеше, но это нормально - оно есть в БД
	return nil
}

// MarkAllAsRead помечает все уведомления как прочитанные
// ServiceDB обязателен, поэтому всегда используем БД
func (ns *NotificationService) MarkAllAsRead(ctx context.Context, clientID, projectID *int) error {
	// Обновляем в БД (serviceDB гарантированно не nil)
	err := ns.serviceDB.MarkAllNotificationsAsRead(clientID, projectID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read in database: %w", err)
	}

	// Также обновляем в памяти для консистентности кеша
	ns.mu.Lock()
	for i := range ns.notifications {
		if clientID != nil && (ns.notifications[i].ClientID == nil || *ns.notifications[i].ClientID != *clientID) {
			continue
		}
		if projectID != nil && (ns.notifications[i].ProjectID == nil || *ns.notifications[i].ProjectID != *projectID) {
			continue
		}
		ns.notifications[i].Read = true
	}
	ns.mu.Unlock()

	return nil
}

// GetUnreadCount возвращает количество непрочитанных уведомлений
// ServiceDB обязателен, поэтому всегда используем БД
func (ns *NotificationService) GetUnreadCount(ctx context.Context, clientID, projectID *int) (int, error) {
	// Получаем из БД (serviceDB гарантированно не nil)
	count, err := ns.serviceDB.GetUnreadNotificationsCount(clientID, projectID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread notifications count from database: %w", err)
	}
	return count, nil
}

// DeleteNotification удаляет уведомление
// ServiceDB обязателен, поэтому всегда используем БД
func (ns *NotificationService) DeleteNotification(ctx context.Context, notificationID int) error {
	// Удаляем из БД (serviceDB гарантированно не nil)
	err := ns.serviceDB.DeleteNotification(notificationID)
	if err != nil {
		return fmt.Errorf("failed to delete notification from database: %w", err)
	}

	// Также удаляем из памяти для консистентности кеша
	ns.mu.Lock()
	for i, n := range ns.notifications {
		if n.ID == notificationID {
			ns.notifications = append(ns.notifications[:i], ns.notifications[i+1:]...)
			ns.mu.Unlock()
			return nil
		}
	}
	ns.mu.Unlock()

	// Уведомление может отсутствовать в кеше, но это нормально - оно удалено из БД
	return nil
}

// ClearCache очищает кеш уведомлений в памяти
// Полезно для тестирования или при необходимости сбросить кеш
func (ns *NotificationService) ClearCache() {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.notifications = make([]Notification, 0)
}

// GetCacheSize возвращает текущий размер кеша уведомлений
func (ns *NotificationService) GetCacheSize() int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return len(ns.notifications)
}

// SetMaxCacheSize устанавливает максимальный размер кеша уведомлений
// Если новый размер меньше текущего размера кеша, кеш будет обрезан
func (ns *NotificationService) SetMaxCacheSize(maxSize int) {
	if maxSize < 1 {
		maxSize = 100 // Минимальный размер кеша
	}
	if maxSize > 10000 {
		maxSize = 10000 // Максимальный размер кеша для защиты от переполнения памяти
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.maxNotifications = maxSize
	// Обрезаем кеш, если он больше нового максимума
	if len(ns.notifications) > maxSize {
		ns.notifications = ns.notifications[:maxSize]
	}
}

// GetMaxCacheSize возвращает максимальный размер кеша уведомлений
func (ns *NotificationService) GetMaxCacheSize() int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.maxNotifications
}

