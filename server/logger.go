package server

import (
	"context"
	"errors"
	"fmt"
	"httpserver/server/middleware"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var (
	// Logger глобальный структурированный логгер
	Logger *slog.Logger
)

func init() {
	// Инициализируем структурированный логгер в формате JSON
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		AddSource: true, // Добавляем информацию об источнике (файл, строка)
	}

	// Используем JSON handler для структурированного логирования
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// LogRequest логирует информацию о входящем HTTP запросе
func LogRequest(r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	Logger.Info("Request received",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"request_id", reqID,
	)
}

// LogError логирует ошибку с контекстом из запроса
func LogError(ctx context.Context, err error, msg string, attrs ...any) {
	reqID := middleware.GetRequestID(ctx)
	
	attrs = append(attrs, "error", err, "request_id", reqID)
	
	Logger.Error(msg, attrs...)
}

// LogErrorf логирует ошибку с форматированным сообщением
func LogErrorf(ctx context.Context, err error, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	LogError(ctx, err, msg)
}

// LogWarn логирует предупреждение
func LogWarn(ctx context.Context, msg string, attrs ...any) {
	reqID := middleware.GetRequestID(ctx)
	attrs = append(attrs, "request_id", reqID)
	Logger.Warn(msg, attrs...)
}

// LogInfo логирует информационное сообщение
func LogInfo(ctx context.Context, msg string, attrs ...any) {
	reqID := middleware.GetRequestID(ctx)
	attrs = append(attrs, "request_id", reqID)
	Logger.Info(msg, attrs...)
}

// LogDebug логирует отладочное сообщение
func LogDebug(ctx context.Context, msg string, attrs ...any) {
	reqID := middleware.GetRequestID(ctx)
	attrs = append(attrs, "request_id", reqID)
	Logger.Debug(msg, attrs...)
}

// LogHTTPError логирует HTTP ошибку с полным контекстом
func LogHTTPError(r *http.Request, err error, statusCode int) {
	reqID := middleware.GetRequestID(r.Context())
	
	var appErr *AppError
	errorMsg := err.Error()
	if errors.As(err, &appErr) {
		Logger.Error("HTTP error",
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", statusCode,
			"error", appErr.Err,
			"user_message", appErr.Message,
			"context", appErr.Context,
			"request_id", reqID,
		)
	} else {
		Logger.Error("HTTP error",
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", statusCode,
			"error", err,
			"error_message", errorMsg,
			"request_id", reqID,
		)
	}
}

// LogDuration логирует продолжительность выполнения операции
func LogDuration(ctx context.Context, operation string, duration time.Duration, attrs ...any) {
	reqID := middleware.GetRequestID(ctx)
	attrs = append(attrs, "request_id", reqID, "duration_ms", duration.Milliseconds())
	Logger.Info(operation+" completed", attrs...)
}

// --- Специализированные функции логирования для нормализации ---

// LogNormalizationStart логирует начало процесса нормализации
func LogNormalizationStart(clientID, projectID int, databasesCount int, normType string) {
	Logger.Info("Normalization started",
		"normalization_type", normType,
		"client_id", clientID,
		"project_id", projectID,
		"databases_count", databasesCount,
	)
}

// LogNormalizationProgress логирует прогресс нормализации
func LogNormalizationProgress(clientID, projectID int, processed, total int, databaseName string) {
	Logger.Info("Normalization progress",
		"client_id", clientID,
		"project_id", projectID,
		"processed", processed,
		"total", total,
		"database", databaseName,
		"progress_percent", float64(processed)/float64(total)*100,
	)
}

// LogNormalizationComplete логирует завершение нормализации
func LogNormalizationComplete(clientID, projectID int, processed, success, errors int, duration time.Duration) {
	Logger.Info("Normalization completed",
		"client_id", clientID,
		"project_id", projectID,
		"processed", processed,
		"success", success,
		"errors", errors,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogNormalizationStopped логирует остановку нормализации
func LogNormalizationStopped(clientID, projectID int, reason string, processed int) {
	Logger.Info("Normalization stopped",
		"client_id", clientID,
		"project_id", projectID,
		"reason", reason,
		"processed_before_stop", processed,
	)
}

// LogNormalizationError логирует ошибку нормализации
func LogNormalizationError(clientID, projectID int, err error, context string) {
	Logger.Error("Normalization error",
		"client_id", clientID,
		"project_id", projectID,
		"error", err,
		"context", context,
	)
}

// LogNormalizationPanic логирует панику в процессе нормализации
func LogNormalizationPanic(projectID int, recovered interface{}, stack string) {
	Logger.Error("Normalization panic",
		"project_id", projectID,
		"recovered", recovered,
		"stack_trace", stack,
	)
}

