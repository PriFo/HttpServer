package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"httpserver/server/services"
)

const (
	// Максимальная длина заголовка уведомления
	MaxNotificationTitleLength = 500
	// Максимальная длина сообщения уведомления
	MaxNotificationMessageLength = 5000
	// Максимальный размер metadata JSON в байтах
	MaxNotificationMetadataSize = 10000
)

// NotificationHandler обработчик для работы с уведомлениями
type NotificationHandler struct {
	notificationService *services.NotificationService
	baseHandler         *BaseHandler
	logFunc             func(entry interface{}) // Опциональная функция логирования
}

// NewNotificationHandler создает новый обработчик уведомлений
func NewNotificationHandler(
	notificationService *services.NotificationService,
	baseHandler *BaseHandler,
) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		baseHandler:         baseHandler,
		logFunc:             nil,
	}
}

// NewNotificationHandlerWithLogging создает новый обработчик уведомлений с поддержкой логирования
func NewNotificationHandlerWithLogging(
	notificationService *services.NotificationService,
	baseHandler *BaseHandler,
	logFunc func(entry interface{}),
) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		baseHandler:         baseHandler,
		logFunc:             logFunc,
	}
}

// SetLogFunc устанавливает функцию логирования
func (h *NotificationHandler) SetLogFunc(logFunc func(entry interface{})) {
	h.logFunc = logFunc
}

// HandleAddNotification обрабатывает POST запросы к /api/notifications для создания нового уведомления
func (h *NotificationHandler) HandleAddNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		Type      string                 `json:"type"`
		Title     string                 `json:"title"`
		Message   string                 `json:"message"`
		ClientID  *int                   `json:"client_id,omitempty"`
		ProjectID *int                   `json:"project_id,omitempty"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	title := strings.TrimSpace(req.Title)
	if title == "" {
		h.baseHandler.WriteJSONError(w, r, "title is required", http.StatusBadRequest)
		return
	}
	if len(title) > MaxNotificationTitleLength {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("title exceeds maximum length of %d characters", MaxNotificationTitleLength), http.StatusBadRequest)
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		h.baseHandler.WriteJSONError(w, r, "message is required", http.StatusBadRequest)
		return
	}
	if len(message) > MaxNotificationMessageLength {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("message exceeds maximum length of %d characters", MaxNotificationMessageLength), http.StatusBadRequest)
		return
	}

	// Валидация размера metadata
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid metadata format: %v", err), http.StatusBadRequest)
			return
		}
		if len(metadataJSON) > MaxNotificationMetadataSize {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("metadata exceeds maximum size of %d bytes", MaxNotificationMetadataSize), http.StatusBadRequest)
			return
		}
	}

	// Валидация типа уведомления
	notificationType := services.NotificationType(req.Type)
	if notificationType != services.NotificationTypeInfo &&
		notificationType != services.NotificationTypeSuccess &&
		notificationType != services.NotificationTypeWarning &&
		notificationType != services.NotificationTypeError {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("invalid notification type: %s. Allowed types: info, success, warning, error", req.Type), http.StatusBadRequest)
		return
	}

	// Создаем уведомление через сервис (используем уже валидированные и обрезанные значения)
	createdNotification, err := h.notificationService.AddNotification(
		r.Context(),
		notificationType,
		title,
		message,
		req.ClientID,
		req.ProjectID,
		req.Metadata,
	)

	if err != nil {
		if h.logFunc != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to create notification: %v", err),
				Endpoint:  r.URL.Path,
			})
		}
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to create notification: %v", err), http.StatusInternalServerError)
		return
	}

	if createdNotification == nil {
		if h.logFunc != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   "Failed to create notification: returned nil",
				Endpoint:  r.URL.Path,
			})
		}
		h.baseHandler.WriteJSONError(w, r, "Failed to create notification", http.StatusInternalServerError)
		return
	}

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Notification created: ID=%d, Type=%s, Title=%s", createdNotification.ID, createdNotification.Type, title),
			Endpoint:  r.URL.Path,
		})
	}

	h.baseHandler.WriteJSONResponse(w, r, createdNotification, http.StatusCreated)
}

// HandleGetNotifications обрабатывает запросы к /api/notifications
func (h *NotificationHandler) HandleGetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Парсим параметры запроса с валидацией
	limit, err := ValidateIntParam(r, "limit", 50, 1, 1000)
	if err != nil {
		// Если ошибка валидации, используем значение по умолчанию
		limit = 50
	}

	offset, err := ValidateIntParam(r, "offset", 0, 0, 0)
	if err != nil {
		// Если ошибка валидации, используем значение по умолчанию
		offset = 0
	}

	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	var clientID, projectID *int
	if clientIDStr := r.URL.Query().Get("client_id"); clientIDStr != "" {
		if parsedID, err := strconv.Atoi(clientIDStr); err == nil {
			clientID = &parsedID
		}
	}
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		if parsedID, err := strconv.Atoi(projectIDStr); err == nil {
			projectID = &parsedID
		}
	}

	notifications, err := h.notificationService.GetNotifications(r.Context(), limit, offset, unreadOnly, clientID, projectID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get notifications: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем общее количество уведомлений для метаданных пагинации
	totalCount, err := h.notificationService.GetNotificationsCount(r.Context(), unreadOnly, clientID, projectID)
	if err != nil {
		// Если не удалось получить общее количество, используем количество возвращенных уведомлений
		totalCount = len(notifications)
	}

	response := map[string]interface{}{
		"notifications": notifications,
		"count":         len(notifications),
		"pagination": map[string]interface{}{
			"limit":       limit,
			"offset":      offset,
			"total_count": totalCount,
			"returned":    len(notifications),
			"has_more":    offset+len(notifications) < totalCount,
		},
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleGetUnreadCount обрабатывает запросы к /api/notifications/unread-count
func (h *NotificationHandler) HandleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	var clientID, projectID *int
	if clientIDStr := r.URL.Query().Get("client_id"); clientIDStr != "" {
		if parsedID, err := strconv.Atoi(clientIDStr); err == nil && parsedID > 0 {
			clientID = &parsedID
		}
	}
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		if parsedID, err := strconv.Atoi(projectIDStr); err == nil && parsedID > 0 {
			projectID = &parsedID
		}
	}

	count, err := h.notificationService.GetUnreadCount(r.Context(), clientID, projectID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get unread count: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"count": count,
	}, http.StatusOK)
}

// HandleMarkAsRead обрабатывает запросы к /api/notifications/{id}/read
func (h *NotificationHandler) HandleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Извлекаем ID из пути
	path := r.URL.Path
	// Предполагаем формат /api/notifications/{id}/read
	notificationIDStr := ""
	if strings.HasPrefix(path, "/api/notifications/") && strings.HasSuffix(path, "/read") {
		// Удаляем префикс и суффикс
		notificationIDStr = path[len("/api/notifications/") : len(path)-len("/read")]
	}

	if notificationIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "Notification ID is required", http.StatusBadRequest)
		return
	}

	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil || notificationID <= 0 {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid notification ID: %s (must be a positive integer)", notificationIDStr), http.StatusBadRequest)
		return
	}

	if err := h.notificationService.MarkAsRead(r.Context(), notificationID); err != nil {
		// Проверяем, является ли ошибка "not found"
		if strings.Contains(err.Error(), "not found") {
			if h.logFunc != nil {
				h.logFunc(LogEntry{
					Timestamp: time.Now(),
					Level:     "WARN",
					Message:   fmt.Sprintf("Attempt to mark non-existent notification as read: ID=%d", notificationID),
					Endpoint:  r.URL.Path,
				})
			}
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Notification not found: %v", err), http.StatusNotFound)
		} else {
			if h.logFunc != nil {
				h.logFunc(LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Failed to mark notification as read: ID=%d, Error=%v", notificationID, err),
					Endpoint:  r.URL.Path,
				})
			}
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to mark notification as read: %v", err), http.StatusInternalServerError)
		}
		return
	}

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Notification marked as read: ID=%d", notificationID),
			Endpoint:  r.URL.Path,
		})
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
	}, http.StatusOK)
}

// HandleMarkAllAsRead обрабатывает запросы к /api/notifications/read-all
func (h *NotificationHandler) HandleMarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var clientID, projectID *int
	if clientIDStr := r.URL.Query().Get("client_id"); clientIDStr != "" {
		if parsedID, err := strconv.Atoi(clientIDStr); err == nil && parsedID > 0 {
			clientID = &parsedID
		}
	}
	if projectIDStr := r.URL.Query().Get("project_id"); projectIDStr != "" {
		if parsedID, err := strconv.Atoi(projectIDStr); err == nil && parsedID > 0 {
			projectID = &parsedID
		}
	}

	if err := h.notificationService.MarkAllAsRead(r.Context(), clientID, projectID); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to mark all notifications as read: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
	}, http.StatusOK)
}

// HandleDeleteNotification обрабатывает запросы к /api/notifications/{id}
func (h *NotificationHandler) HandleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	// Извлекаем ID из пути
	path := r.URL.Path
	notificationIDStr := ""
	if len(path) > len("/api/notifications/") {
		notificationIDStr = path[len("/api/notifications/"):]
	}

	if notificationIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "Notification ID is required", http.StatusBadRequest)
		return
	}

	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil || notificationID <= 0 {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid notification ID: %s (must be a positive integer)", notificationIDStr), http.StatusBadRequest)
		return
	}

	if err := h.notificationService.DeleteNotification(r.Context(), notificationID); err != nil {
		// Проверяем, является ли ошибка "not found"
		if strings.Contains(err.Error(), "not found") {
			if h.logFunc != nil {
				h.logFunc(LogEntry{
					Timestamp: time.Now(),
					Level:     "WARN",
					Message:   fmt.Sprintf("Attempt to delete non-existent notification: ID=%d", notificationID),
					Endpoint:  r.URL.Path,
				})
			}
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Notification not found: %v", err), http.StatusNotFound)
		} else {
			if h.logFunc != nil {
				h.logFunc(LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Failed to delete notification: ID=%d, Error=%v", notificationID, err),
					Endpoint:  r.URL.Path,
				})
			}
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to delete notification: %v", err), http.StatusInternalServerError)
		}
		return
	}

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Notification deleted: ID=%d", notificationID),
			Endpoint:  r.URL.Path,
		})
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
	}, http.StatusOK)
}

// HandleNotificationRoutes обрабатывает все запросы к /api/notifications
func (h *NotificationHandler) HandleNotificationRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Удаляем префикс /api/notifications
	if len(path) >= len("/api/notifications") {
		path = path[len("/api/notifications"):]
	}

	if path == "" || path == "/" {
		// GET /api/notifications или POST /api/notifications
		if r.Method == http.MethodPost {
			h.HandleAddNotification(w, r)
		} else {
			h.HandleGetNotifications(w, r)
		}
		return
	}

	// Удаляем ведущий слеш
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	parts := strings.Split(path, "/")

	if len(parts) > 0 && parts[0] != "" {
		switch parts[0] {
		case "unread-count":
			// GET /api/notifications/unread-count
			h.HandleGetUnreadCount(w, r)
			return
		case "read-all":
			// POST /api/notifications/read-all
			h.HandleMarkAllAsRead(w, r)
			return
		default:
			// Проверяем, является ли это ID с суффиксом /read
			if len(parts) == 2 && parts[1] == "read" {
				// POST /api/notifications/{id}/read
				// Обновляем путь для HandleMarkAsRead
				r.URL.Path = "/api/notifications/" + parts[0] + "/read"
				h.HandleMarkAsRead(w, r)
				return
			} else if len(parts) == 1 {
				// DELETE /api/notifications/{id}
				// Обновляем путь для HandleDeleteNotification
				r.URL.Path = "/api/notifications/" + parts[0]
				h.HandleDeleteNotification(w, r)
				return
			}
		}
	}

	// Если ничего не подошло, возвращаем список уведомлений
	h.HandleGetNotifications(w, r)
}

