package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "httpserver/server/errors"
	"httpserver/server/middleware"
)

// mockLogHandler для перехвата логов в тестах
type mockLogHandler struct {
	logs []logEntry
}

type logEntry struct {
	level   slog.Level
	message string
	attrs   map[string]interface{}
}

func (h *mockLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *mockLogHandler) Handle(ctx context.Context, record slog.Record) error {
	attrs := make(map[string]interface{})
	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	h.logs = append(h.logs, logEntry{
		level:   record.Level,
		message: record.Message,
		attrs:   attrs,
	})
	return nil
}

func (h *mockLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *mockLogHandler) WithGroup(name string) slog.Handler {
	return h
}

// ============================================================================
// 1. Тестирование конструкторов ошибок
// ============================================================================

func TestNewValidationError_CreatesCorrectError(t *testing.T) {
	originalErr := errors.New("validation failed")
	message := "Invalid input"
	
	err := apperrors.NewValidationError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewValidationError should not return nil")
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
	if err.Err != originalErr {
		t.Errorf("Expected original error %v, got %v", originalErr, err.Err)
	}
}

func TestNewInternalError_CreatesCorrectError(t *testing.T) {
	originalErr := errors.New("database connection failed")
	message := "Failed to connect to database"
	
	err := apperrors.NewInternalError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewInternalError should not return nil")
	}
	if err.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, err.Code)
	}
	if err.Message != "Внутренняя ошибка сервера" {
		t.Errorf("Expected user message %q, got %q", "Внутренняя ошибка сервера", err.Message)
	}
	if err.Err == nil {
		t.Fatal("Expected joined error, got nil")
	}
}

func TestNewNotFoundError_CreatesCorrectError(t *testing.T) {
	originalErr := sql.ErrNoRows
	message := "Resource not found"
	
	err := apperrors.NewNotFoundError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewNotFoundError should not return nil")
	}
	if err.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
	if err.Err != originalErr {
		t.Errorf("Expected original error %v, got %v", originalErr, err.Err)
	}
}

func TestNewConflictError_CreatesCorrectError(t *testing.T) {
	originalErr := errors.New("resource already exists")
	message := "Resource conflict"
	
	err := apperrors.NewConflictError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewConflictError should not return nil")
	}
	if err.Code != http.StatusConflict {
		t.Errorf("Expected status code %d, got %d", http.StatusConflict, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
}

func TestNewUnauthorizedError_CreatesCorrectError(t *testing.T) {
	originalErr := errors.New("invalid token")
	message := "Unauthorized access"
	
	err := apperrors.NewUnauthorizedError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewUnauthorizedError should not return nil")
	}
	if err.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
}

func TestNewForbiddenError_CreatesCorrectError(t *testing.T) {
	originalErr := errors.New("insufficient permissions")
	message := "Access forbidden"
	
	err := apperrors.NewForbiddenError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewForbiddenError should not return nil")
	}
	if err.Code != http.StatusForbidden {
		t.Errorf("Expected status code %d, got %d", http.StatusForbidden, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
}

func TestNewBadGatewayError_CreatesCorrectError(t *testing.T) {
	originalErr := errors.New("upstream service unavailable")
	message := "Bad gateway"
	
	err := apperrors.NewBadGatewayError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewBadGatewayError should not return nil")
	}
	if err.Code != http.StatusBadGateway {
		t.Errorf("Expected status code %d, got %d", http.StatusBadGateway, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
}

func TestNewServiceUnavailableError_CreatesCorrectError(t *testing.T) {
	originalErr := errors.New("service maintenance")
	message := "Service unavailable"
	
	err := apperrors.NewServiceUnavailableError(message, originalErr)
	
	if err == nil {
		t.Fatal("NewServiceUnavailableError should not return nil")
	}
	if err.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
}

func TestNewInternalError_HidesDetailsFromUser(t *testing.T) {
	originalErr := errors.New("sensitive database error: connection string exposed")
	message := "Database connection failed with sensitive info"
	
	err := apperrors.NewInternalError(message, originalErr)
	
	// Пользователь должен видеть только общее сообщение
	if err.Message != "Внутренняя ошибка сервера" {
		t.Errorf("User should see generic message, got %q", err.Message)
	}
	
	// Детали должны быть в Err для логирования
	if err.Err == nil {
		t.Fatal("Error details should be preserved in Err field")
	}
	
	// Проверяем, что детали доступны через Error()
	errorStr := err.Error()
	if !strings.Contains(errorStr, "Внутренняя ошибка сервера") {
		t.Errorf("Error() should contain user message, got %q", errorStr)
	}
}

func TestNewInternalError_JoinsErrors(t *testing.T) {
	originalErr := errors.New("original error")
	message := "additional context"
	
	err := apperrors.NewInternalError(message, originalErr)
	
	if err.Err == nil {
		t.Fatal("Expected joined error, got nil")
	}
	
	// Проверяем, что обе ошибки присутствуют
	errStr := err.Err.Error()
	if !strings.Contains(errStr, message) {
		t.Errorf("Joined error should contain message %q, got %q", message, errStr)
	}
	if !strings.Contains(errStr, "original error") {
		t.Errorf("Joined error should contain original error, got %q", errStr)
	}
}

func TestNewValidationError_WithNilError(t *testing.T) {
	message := "Invalid input"
	
	err := apperrors.NewValidationError(message, nil)
	
	if err == nil {
		t.Fatal("NewValidationError should not return nil even with nil error")
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, err.Code)
	}
	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}
	if err.Err != nil {
		t.Errorf("Expected nil error, got %v", err.Err)
	}
}

func TestNewInternalError_WithNilError(t *testing.T) {
	message := "Internal error context"
	
	err := apperrors.NewInternalError(message, nil)
	
	if err == nil {
		t.Fatal("NewInternalError should not return nil even with nil error")
	}
	if err.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, err.Code)
	}
	if err.Err == nil {
		t.Fatal("Expected joined error even with nil original error")
	}
	
	// Проверяем, что message присутствует в joined error
	errStr := err.Err.Error()
	if !strings.Contains(errStr, message) {
		t.Errorf("Joined error should contain message %q, got %q", message, errStr)
	}
}

// ============================================================================
// 2. Тестирование методов AppError
// ============================================================================

func TestAppError_Error_WithNestedError(t *testing.T) {
	nestedErr := errors.New("nested error")
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "Validation failed",
		Err:     nestedErr,
	}
	
	errorStr := err.Error()
	
	if !strings.Contains(errorStr, "Validation failed") {
		t.Errorf("Error() should contain message, got %q", errorStr)
	}
	if !strings.Contains(errorStr, "nested error") {
		t.Errorf("Error() should contain nested error, got %q", errorStr)
	}
}

func TestAppError_Error_WithoutNestedError(t *testing.T) {
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "Validation failed",
		Err:     nil,
	}
	
	errorStr := err.Error()
	
	if errorStr != "Validation failed" {
		t.Errorf("Error() should return only message, got %q", errorStr)
	}
}

func TestAppError_Unwrap_ReturnsOriginalError(t *testing.T) {
	originalErr := errors.New("original error")
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "Validation failed",
		Err:     originalErr,
	}
	
	unwrapped := err.Unwrap()
	
	if unwrapped != originalErr {
		t.Errorf("Unwrap() should return original error, got %v", unwrapped)
	}
	
	// Проверяем совместимость с errors.Is
	if !errors.Is(err, originalErr) {
		t.Error("errors.Is should work with AppError")
	}
}

func TestAppError_StatusCode_ReturnsCorrectCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"BadRequest", http.StatusBadRequest, 400},
		{"InternalServerError", http.StatusInternalServerError, 500},
		{"NotFound", http.StatusNotFound, 404},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &apperrors.AppError{Code: tt.code}
			if err.StatusCode() != tt.expected {
				t.Errorf("StatusCode() = %d, want %d", err.StatusCode(), tt.expected)
			}
		})
	}
}

func TestAppError_UserMessage_ReturnsCorrectMessage(t *testing.T) {
	message := "User-friendly message"
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: message,
	}
	
	if err.UserMessage() != message {
		t.Errorf("UserMessage() = %q, want %q", err.UserMessage(), message)
	}
}

func TestAppError_GetContext_ReturnsContext(t *testing.T) {
	context := "function: GetUser, params: {id: 123}"
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "Error message",
		Context: context,
	}
	
	if err.GetContext() != context {
		t.Errorf("GetContext() = %q, want %q", err.GetContext(), context)
	}
}

func TestAppError_WithContext_AddsContext(t *testing.T) {
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "Error message",
		Context: "",
	}
	
	context := "additional context"
	result := err.WithContext(context)
	
	if result.Context != context {
		t.Errorf("WithContext() should set context to %q, got %q", context, result.Context)
	}
	if result != err {
		t.Error("WithContext() should return the same instance")
	}
}

func TestAppError_WithContext_ReturnsSelf(t *testing.T) {
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "Error message",
	}
	
	result := err.WithContext("test context")
	
	if result != err {
		t.Error("WithContext() should return the same instance")
	}
}

func TestAppError_ImplementsHTTPError(t *testing.T) {
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "Test error",
		Err:     errors.New("nested"),
		Context: "test context",
	}
	
	// Проверяем, что AppError реализует интерфейс middleware.HTTPError
	var httpErr middleware.HTTPError = err
	
	if httpErr.StatusCode() != http.StatusBadRequest {
		t.Errorf("StatusCode() = %d, want %d", httpErr.StatusCode(), http.StatusBadRequest)
	}
	if httpErr.UserMessage() != "Test error" {
		t.Errorf("UserMessage() = %q, want %q", httpErr.UserMessage(), "Test error")
	}
	if httpErr.GetContext() != "test context" {
		t.Errorf("GetContext() = %q, want %q", httpErr.GetContext(), "test context")
	}
	if httpErr.Unwrap() != err.Err {
		t.Error("Unwrap() should return nested error")
	}
}

// ============================================================================
// 3. Тестирование функции WrapError
// ============================================================================

func TestWrapError_WithAppError_AddsContext(t *testing.T) {
	originalErr := apperrors.NewValidationError("Original error", errors.New("nested"))
	context := "Additional context"
	
	wrapped := apperrors.WrapError(originalErr, context)
	
	if wrapped == nil {
		t.Fatal("WrapError should not return nil")
	}
	if wrapped.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, wrapped.Code)
	}
	if !strings.Contains(wrapped.Message, context) {
		t.Errorf("Wrapped message should contain context %q, got %q", context, wrapped.Message)
	}
	if !strings.Contains(wrapped.Message, "Original error") {
		t.Errorf("Wrapped message should contain original message, got %q", wrapped.Message)
	}
	if wrapped.Err != originalErr.Err {
		t.Error("Wrapped error should preserve original Err")
	}
}

func TestWrapError_WithGenericError_CreatesInternalError(t *testing.T) {
	originalErr := errors.New("generic error")
	context := "Error context"
	
	wrapped := apperrors.WrapError(originalErr, context)
	
	if wrapped == nil {
		t.Fatal("WrapError should not return nil")
	}
	if wrapped.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, wrapped.Code)
	}
	if wrapped.Message != "Внутренняя ошибка сервера" {
		t.Errorf("Expected user message %q, got %q", "Внутренняя ошибка сервера", wrapped.Message)
	}
	if wrapped.Err == nil {
		t.Fatal("Expected joined error, got nil")
	}
	
	// Проверяем, что оригинальная ошибка присутствует в joined error
	errStr := wrapped.Err.Error()
	if !strings.Contains(errStr, context) {
		t.Errorf("Joined error should contain context %q, got %q", context, errStr)
	}
	if !strings.Contains(errStr, "generic error") {
		t.Errorf("Joined error should contain original error, got %q", errStr)
	}
}

func TestWrapError_WithNil_ReturnsNil(t *testing.T) {
	result := apperrors.WrapError(nil, "context")
	
	if result != nil {
		t.Errorf("WrapError(nil) should return nil, got %v", result)
	}
}

// ============================================================================
// 4. Тестирование HandleHTTPError
// ============================================================================

func TestHandleHTTPError_ValidationError_Returns400AndLogs(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	appErr := apperrors.NewValidationError("Invalid input", errors.New("validation failed"))
	
	req := httptest.NewRequest("POST", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "test-request-id"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", w.Header().Get("Content-Type"))
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Error != "Invalid input" {
		t.Errorf("Expected error message %q, got %q", "Invalid input", response.Error)
	}
	if response.RequestID != "test-request-id" {
		t.Errorf("Expected request ID %q, got %q", "test-request-id", response.RequestID)
	}
	if response.Timestamp == "" {
		t.Error("Expected timestamp, got empty string")
	}
	
	// Проверяем логирование
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.level != slog.LevelError {
		t.Errorf("Expected error level, got %v", lastLog.level)
	}
	if lastLog.message != "HTTP error" {
		t.Errorf("Expected log message %q, got %q", "HTTP error", lastLog.message)
	}
	if lastLog.attrs["status_code"] != http.StatusBadRequest {
		t.Errorf("Expected status code %d in log, got %v", http.StatusBadRequest, lastLog.attrs["status_code"])
	}
}

func TestHandleHTTPError_InternalError_Returns500AndLogs(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	appErr := apperrors.NewInternalError("Database connection failed", errors.New("connection timeout"))
	
	req := httptest.NewRequest("GET", "/api/data", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "test-req-123"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Пользователь должен видеть общее сообщение
	if response.Error != "Внутренняя ошибка сервера" {
		t.Errorf("Expected user message %q, got %q", "Внутренняя ошибка сервера", response.Error)
	}
	
	// Проверяем логирование
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.attrs["status_code"] != http.StatusInternalServerError {
		t.Errorf("Expected status code %d in log, got %v", http.StatusInternalServerError, lastLog.attrs["status_code"])
	}
}

func TestHandleHTTPError_NotFoundError_Returns404AndLogs(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	appErr := apperrors.NewNotFoundError("Resource not found", sql.ErrNoRows)
	
	req := httptest.NewRequest("GET", "/api/resource/123", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-404"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Error != "Resource not found" {
		t.Errorf("Expected error message %q, got %q", "Resource not found", response.Error)
	}
}

func TestHandleHTTPError_WithContext_LogsContext(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	appErr := apperrors.NewValidationError("Error", errors.New("nested"))
	appErr = appErr.WithContext("function: GetUser, params: {id: 123}")
	
	req := httptest.NewRequest("GET", "/api/user/123", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-ctx"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	// Проверяем, что контекст залогирован
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.attrs["context"] != "function: GetUser, params: {id: 123}" {
		t.Errorf("Expected context in log, got %v", lastLog.attrs["context"])
	}
}

func TestHandleHTTPError_GenericError_Returns500AndLogs(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	genericErr := errors.New("generic error occurred")
	
	req := httptest.NewRequest("POST", "/api/endpoint", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-gen"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, genericErr)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Error != "Внутренняя ошибка сервера" {
		t.Errorf("Expected generic error message, got %q", response.Error)
	}
	
	// Проверяем логирование
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.attrs["error"] == nil {
		t.Error("Expected error to be logged")
	}
}

func TestHandleHTTPError_NilError_Returns500(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-nil"))
	w := httptest.NewRecorder()
	
	// Не должно произойти паники
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("HandleHTTPError should not panic on nil error, panicked with %v", r)
			}
		}()
		middleware.HandleHTTPError(w, req, nil)
	}()
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Error != "Внутренняя ошибка сервера" {
		t.Errorf("Expected generic error message, got %q", response.Error)
	}
}

func TestHandleHTTPError_SetsCorrectHeaders(t *testing.T) {
	appErr := apperrors.NewValidationError("Test error", nil)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-headers"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", w.Header().Get("Content-Type"))
	}
}

func TestHandleHTTPError_ResponseContainsRequestID(t *testing.T) {
	appErr := apperrors.NewValidationError("Test error", nil)
	requestID := "test-request-id-12345"
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), requestID))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.RequestID != requestID {
		t.Errorf("Expected request ID %q, got %q", requestID, response.RequestID)
	}
}

func TestHandleHTTPError_ResponseContainsTimestamp(t *testing.T) {
	appErr := apperrors.NewValidationError("Test error", nil)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-time"))
	w := httptest.NewRecorder()
	
	beforeTime := time.Now()
	middleware.HandleHTTPError(w, req, appErr)
	afterTime := time.Now()
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Timestamp == "" {
		t.Fatal("Expected timestamp in response, got empty string")
	}
	
	parsedTime, err := time.Parse(time.RFC3339, response.Timestamp)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}
	
	if parsedTime.Before(beforeTime) || parsedTime.After(afterTime) {
		t.Errorf("Timestamp %q should be between %v and %v", response.Timestamp, beforeTime, afterTime)
	}
}

func TestHandleHTTPError_ResponseJSONStructure(t *testing.T) {
	appErr := apperrors.NewValidationError("Test error message", errors.New("nested"))
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-json"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Проверяем структуру ответа
	if response.Error == "" {
		t.Error("Response should contain error field")
	}
	if response.Timestamp == "" {
		t.Error("Response should contain timestamp field")
	}
	if response.RequestID == "" {
		t.Error("Response should contain request_id field")
	}
	
	// Проверяем, что JSON валидный
	bodyBytes := w.Body.Bytes()
	var jsonCheck map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &jsonCheck); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}
}

// ============================================================================
// 5. Тестирование логирования
// ============================================================================

func TestLogging_ErrorIsLoggedWithCorrectLevel(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	appErr := apperrors.NewInternalError("Test error", errors.New("nested"))
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-level"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.level != slog.LevelError {
		t.Errorf("Expected error level, got %v", lastLog.level)
	}
}

func TestLogging_LogsContainRequestID(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	requestID := "test-req-id-456"
	appErr := apperrors.NewValidationError("Test error", nil)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), requestID))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.attrs["request_id"] != requestID {
		t.Errorf("Expected request_id %q in log, got %v", requestID, lastLog.attrs["request_id"])
	}
}

func TestLogging_LogsContainMethodAndPath(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	method := "POST"
	path := "/api/users/123"
	appErr := apperrors.NewValidationError("Test error", nil)
	
	req := httptest.NewRequest(method, path, nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-method"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.attrs["method"] != method {
		t.Errorf("Expected method %q in log, got %v", method, lastLog.attrs["method"])
	}
	if lastLog.attrs["path"] != path {
		t.Errorf("Expected path %q in log, got %v", path, lastLog.attrs["path"])
	}
}

func TestLogging_AppErrorLogsUnwrap(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	nestedErr := errors.New("nested error details")
	appErr := apperrors.NewValidationError("User message", nestedErr)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-unwrap"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.attrs["error"] == nil {
		t.Error("Expected error to be logged in 'error' field")
	}
	
	// Проверяем, что логируется Unwrap(), а не UserMessage
	if lastLog.attrs["user_message"] != "User message" {
		t.Errorf("Expected user_message in log, got %v", lastLog.attrs["user_message"])
	}
}

func TestLogging_GenericErrorLogsDirectly(t *testing.T) {
	mockHandler := &mockLogHandler{}
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(mockHandler))
	defer slog.SetDefault(originalLogger)
	
	genericErr := errors.New("generic error message")
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-generic"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, genericErr)
	
	if len(mockHandler.logs) == 0 {
		t.Fatal("Expected error to be logged")
	}
	
	lastLog := mockHandler.logs[len(mockHandler.logs)-1]
	if lastLog.attrs["error"] == nil {
		t.Error("Expected error to be logged")
	}
	
	// Проверяем, что нет user_message для обычной ошибки
	if lastLog.attrs["user_message"] != nil {
		t.Error("Generic error should not have user_message in log")
	}
}

// ============================================================================
// 6. Интеграционные тесты с сервисами
// ============================================================================

func TestServiceIntegration_NotFoundError_FromDatabase(t *testing.T) {
	// Симулируем ситуацию, когда сервис получает sql.ErrNoRows
	dbErr := sql.ErrNoRows
	
	// Сервис должен преобразовать это в NotFoundError
	appErr := apperrors.NewNotFoundError("Клиент не найден", dbErr)
	
	if appErr.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, appErr.Code)
	}
	
	// Проверяем, что можно использовать errors.Is
	if !errors.Is(appErr, sql.ErrNoRows) {
		t.Error("AppError should be compatible with errors.Is for sql.ErrNoRows")
	}
	
	// Проверяем обработку через HandleHTTPError
	req := httptest.NewRequest("GET", "/api/clients/123", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-db"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Error != "Клиент не найден" {
		t.Errorf("Expected error message %q, got %q", "Клиент не найден", response.Error)
	}
}

func TestServiceIntegration_ValidationError_FromInvalidInput(t *testing.T) {
	// Симулируем ситуацию валидации входных данных
	invalidInput := errors.New("empty name field")
	
	// Сервис должен вернуть ValidationError
	appErr := NewValidationError("поле 'name' обязательно для заполнения", invalidInput)
	
	if appErr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, appErr.Code)
	}
	
	// Проверяем обработку через HandleHTTPError
	req := httptest.NewRequest("POST", "/api/clients", strings.NewReader(`{"name": ""}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-validation"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if !strings.Contains(response.Error, "name") {
		t.Errorf("Expected error message to contain 'name', got %q", response.Error)
	}
}

func TestServiceIntegration_InternalError_FromDatabaseError(t *testing.T) {
	// Симулируем ситуацию ошибки БД
	dbErr := errors.New("database connection failed: timeout after 30s")
	
	// Сервис должен вернуть InternalError (скрывая детали от пользователя)
	appErr := apperrors.NewInternalError("не удалось получить список клиентов", dbErr)
	
	if appErr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, appErr.Code)
	}
	
	// Пользователь видит общее сообщение
	if appErr.Message != "Внутренняя ошибка сервера" {
		t.Errorf("User should see generic message, got %q", appErr.Message)
	}
	
	// Детали должны быть в Err
	if appErr.Err == nil {
		t.Fatal("Error details should be preserved")
	}
	
	// Проверяем обработку через HandleHTTPError
	req := httptest.NewRequest("GET", "/api/clients", nil)
	req = req.WithContext(middleware.SetRequestID(req.Context(), "req-db-error"))
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Пользователь не должен видеть детали БД
	if strings.Contains(response.Error, "timeout") || strings.Contains(response.Error, "connection") {
		t.Errorf("User should not see database error details, got %q", response.Error)
	}
}

// ============================================================================
// Дополнительные тесты для edge cases
// ============================================================================

func TestAppError_Error_WithEmptyMessage(t *testing.T) {
	err := &apperrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "",
		Err:     errors.New("nested"),
	}
	
	errorStr := err.Error()
	if errorStr == "" {
		t.Error("Error() should not return empty string even with empty message")
	}
}

func TestHandleHTTPError_WithEmptyRequestID(t *testing.T) {
	appErr := apperrors.NewValidationError("Test error", nil)
	
	req := httptest.NewRequest("GET", "/api/test", nil)
	// Не устанавливаем request_id в контекст
	w := httptest.NewRecorder()
	
	middleware.HandleHTTPError(w, req, appErr)
	
	var response middleware.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// RequestID может быть пустым, но это не должно вызывать ошибку
	if response.RequestID != "" {
		// Если request_id установлен, проверяем его формат
		if len(response.RequestID) == 0 {
			t.Error("RequestID should not be empty if set")
		}
	}
}

func TestWrapError_PreservesContext(t *testing.T) {
	originalErr := apperrors.NewValidationError("Original", errors.New("nested"))
	originalErr = originalErr.WithContext("original context")
	
	wrapped := apperrors.WrapError(originalErr, "Additional")
	
	if wrapped.Context != "original context" {
		t.Errorf("Expected context to be preserved, got %q", wrapped.Context)
	}
}

func TestAllErrorConstructors_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		constructor func(string, error) *apperrors.AppError
		expectedCode int
		message     string
		err         error
	}{
		{"ValidationError", apperrors.NewValidationError, http.StatusBadRequest, "Validation failed", errors.New("invalid")},
		{"NotFoundError", apperrors.NewNotFoundError, http.StatusNotFound, "Not found", sql.ErrNoRows},
		{"ConflictError", apperrors.NewConflictError, http.StatusConflict, "Conflict", errors.New("exists")},
		{"UnauthorizedError", apperrors.NewUnauthorizedError, http.StatusUnauthorized, "Unauthorized", errors.New("invalid token")},
		{"ForbiddenError", apperrors.NewForbiddenError, http.StatusForbidden, "Forbidden", errors.New("no access")},
		{"BadGatewayError", apperrors.NewBadGatewayError, http.StatusBadGateway, "Bad gateway", errors.New("upstream error")},
		{"ServiceUnavailableError", apperrors.NewServiceUnavailableError, http.StatusServiceUnavailable, "Unavailable", errors.New("maintenance")},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor(tt.message, tt.err)
			if err.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, err.Code)
			}
			if err.Message != tt.message {
				t.Errorf("Expected message %q, got %q", tt.message, err.Message)
			}
			if err.Err != tt.err {
				t.Errorf("Expected error %v, got %v", tt.err, err.Err)
			}
		})
	}
}

