package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	apperrors "httpserver/server/errors"
)

// Глобальный сборщик метрик ошибок
var globalErrorMetrics *apperrors.ErrorMetricsCollector

// InitErrorMetrics инициализирует глобальный сборщик метрик ошибок
func InitErrorMetrics() {
	globalErrorMetrics = apperrors.NewErrorMetricsCollector()
}

// GetErrorMetrics возвращает глобальный сборщик метрик ошибок
func GetErrorMetrics() *apperrors.ErrorMetricsCollector {
	if globalErrorMetrics == nil {
		globalErrorMetrics = apperrors.NewErrorMetricsCollector()
	}
	return globalErrorMetrics
}

// HTTPError интерфейс для ошибок с HTTP статусом и сообщением
// Используется для избежания циклических зависимостей
type HTTPError interface {
	error
	StatusCode() int
	UserMessage() string
	GetContext() string
	Unwrap() error
}

// ErrorResponse структура ответа об ошибке
type ErrorResponse struct {
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteJSONError записывает JSON ошибку и логирует её
func WriteJSONError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	reqID := GetRequestID(r.Context())
	
	// Логируем ошибку
	if r != nil {
		slog.Error("HTTP error",
			"error", message,
			"status_code", statusCode,
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
		)
	} else {
		slog.Error("HTTP error",
			"error", message,
			"status_code", statusCode,
			"request_id", reqID,
		)
	}
	
	WriteJSONErrorWithRequestID(w, message, statusCode, reqID)
}

// WriteJSONErrorWithRequestID записывает JSON ошибку с request ID
func WriteJSONErrorWithRequestID(w http.ResponseWriter, message string, statusCode int, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := ErrorResponse{
		Error:     message,
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: requestID,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Используем slog для логирования ошибки кодирования
		slog.Error("Error encoding JSON error response", "error", err)
	}
}

// HandleHTTPError обрабатывает ошибку и возвращает JSON ответ
// Поддерживает HTTPError интерфейс для правильной обработки статус кодов и сообщений
func HandleHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	reqID := GetRequestID(r.Context())
	endpoint := r.URL.Path
	
	var statusCode int
	var message string
	var appErr *apperrors.AppError
	
	// Проверяем, реализует ли ошибка интерфейс HTTPError
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		statusCode = httpErr.StatusCode()
		message = httpErr.UserMessage()
		
		// Пытаемся получить AppError для метрик
		if errors.As(err, &appErr) {
			// Записываем в метрики
			metrics := GetErrorMetrics()
			metrics.RecordError(appErr, endpoint, reqID)
		}
		
		// Логируем с полным контекстом HTTPError
		slog.Error("HTTP error",
			"error", httpErr.Unwrap(),
			"user_message", httpErr.UserMessage(),
			"context", httpErr.GetContext(),
			"status_code", statusCode,
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
		)
	} else {
		// Обычная ошибка - используем дефолтные значения
		statusCode = http.StatusInternalServerError
		message = "Внутренняя ошибка сервера"
		
		// Создаем AppError для метрик
		appErr = apperrors.NewInternalError("внутренняя ошибка сервера", err)
		metrics := GetErrorMetrics()
		metrics.RecordError(appErr, endpoint, reqID)
		
		// Логируем ошибку
		slog.Error("HTTP error",
			"error", err,
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := ErrorResponse{
		Error:     message,
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: reqID,
	}
	
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		slog.Error("Error encoding JSON error response", "error", encodeErr)
	}
}

// WriteJSONResponse записывает JSON ответ
func WriteJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Error encoding JSON response", 
			"error", err,
			"request_id", GetRequestID(r.Context()),
		)
		WriteJSONError(w, r, "Internal server error", http.StatusInternalServerError)
	}
}

// RecoverMiddleware обрабатывает паники с детальным логированием
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stackTrace := debug.Stack()
				reqID := GetRequestID(r.Context())
				
				// Логируем панику с полным контекстом
				slog.Error("Panic recovered",
					"panic", err,
					"stack_trace", string(stackTrace),
					"method", r.Method,
					"path", r.URL.Path,
					"request_id", reqID,
				)
				
				// В production не отправляем stack trace клиенту
				WriteJSONError(w, r, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ErrorHandlerMiddleware обрабатывает ошибки в цепочке middleware
func ErrorHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Создаем кастомный ResponseWriter для перехвата статуса
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(rw, r)
		
		// Если статус код указывает на ошибку, логируем
		if rw.statusCode >= 400 {
			reqID := GetRequestID(r.Context())
			slog.Warn("HTTP error",
				"status_code", rw.statusCode,
				"method", r.Method,
				"path", r.URL.Path,
				"request_id", reqID,
			)
		}
	})
}

// responseWriter обертка для ResponseWriter для перехвата статуса
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

