package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	servererrors "httpserver/server/errors"
)

// TestErrorResponse проверяет структуру ответа об ошибке
func TestErrorResponse(t *testing.T) {
	resp := ErrorResponse{
		Error:     "test error",
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: "test-request-id",
	}
	
	if resp.Error == "" {
		t.Error("ErrorResponse.Error should not be empty")
	}
	
	if resp.Timestamp == "" {
		t.Error("ErrorResponse.Timestamp should not be empty")
	}
}

// TestAppError проверяет структуру ошибки приложения
func TestAppError(t *testing.T) {
	err := &servererrors.AppError{
		Code:    http.StatusBadRequest,
		Message: "test error",
		Err:     nil,
	}
	
	if err.Code != http.StatusBadRequest {
		t.Errorf("AppError.Code = %d, want %d", err.Code, http.StatusBadRequest)
	}
	
	if err.Message == "" {
		t.Error("AppError.Message should not be empty")
	}
	
	errorStr := err.Error()
	if errorStr == "" {
		t.Error("AppError.Error() should not return empty string")
	}
}

// TestAppError_WithNestedError проверяет ошибку с вложенной ошибкой
func TestAppError_WithNestedError(t *testing.T) {
	nestedErr := &testError{msg: "nested error"}
	err := &servererrors.AppError{
		Code:    http.StatusInternalServerError,
		Message: "test error",
		Err:     nestedErr,
	}
	
	errorStr := err.Error()
	if !strings.Contains(errorStr, "test error") {
		t.Errorf("AppError.Error() = %s, should contain 'test error'", errorStr)
	}
}

// testError простая ошибка для тестирования
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestWriteJSONError проверяет запись JSON ошибки
func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	
	WriteJSONError(w, r, "test error", http.StatusBadRequest)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Response status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", w.Header().Get("Content-Type"))
	}
	
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Error != "test error" {
		t.Errorf("Response.Error = %s, want 'test error'", response.Error)
	}
	
	if response.Timestamp == "" {
		t.Error("Response.Timestamp should not be empty")
	}
}

// TestWriteJSONErrorWithRequestID проверяет запись JSON ошибки с request ID
func TestWriteJSONErrorWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	requestID := "test-request-id"
	
	WriteJSONErrorWithRequestID(w, "test error", http.StatusInternalServerError, requestID)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Response status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.RequestID != requestID {
		t.Errorf("Response.RequestID = %s, want %s", response.RequestID, requestID)
	}
}

// TestWriteJSONResponse проверяет запись JSON ответа
func TestWriteJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	data := map[string]string{"message": "success"}
	
	WriteJSONResponse(w, r, data, http.StatusOK)
	
	if w.Code != http.StatusOK {
		t.Errorf("Response status code = %d, want %d", w.Code, http.StatusOK)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", w.Header().Get("Content-Type"))
	}
	
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response["message"] != "success" {
		t.Errorf("Response.message = %s, want 'success'", response["message"])
	}
}

// TestRecoverMiddleware проверяет обработку паник
func TestRecoverMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	
	middleware := RecoverMiddleware(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	// Не должно произойти паники
	middleware.ServeHTTP(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Response status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.Error == "" {
		t.Error("Response should contain error message")
	}
}

// TestErrorHandlerMiddleware проверяет обработку ошибок
func TestErrorHandlerMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	
	middleware := ErrorHandlerMiddleware(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	middleware.ServeHTTP(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Response status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestResponseWriter проверяет обертку ResponseWriter
func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
	
	// Проверяем начальный статус
	if rw.statusCode != http.StatusOK {
		t.Errorf("Initial status code = %d, want %d", rw.statusCode, http.StatusOK)
	}
	
	// Устанавливаем новый статус
	newStatus := http.StatusCreated
	rw.WriteHeader(newStatus)
	
	if rw.statusCode != newStatus {
		t.Errorf("Status code after WriteHeader = %d, want %d", rw.statusCode, newStatus)
	}
	
	if w.Code != newStatus {
		t.Errorf("Underlying ResponseWriter status = %d, want %d", w.Code, newStatus)
	}
}

