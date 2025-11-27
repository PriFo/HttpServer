package server

import (
	"httpserver/server/middleware"
	"log/slog"
	"net/http"
	"time"
)

// SecurityHeadersMiddleware добавляет заголовки безопасности
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Заголовки безопасности
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// CORS заголовки (настраиваются в зависимости от окружения)
		origin := r.Header.Get("Origin")
		if origin != "" {
			// В продакшене здесь должна быть проверка разрешенных origins
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}
		
		// Обработка preflight запросов
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware удален - используйте middleware.RequestIDMiddleware

// LoggingMiddleware логирует входящие запросы
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		requestID := middleware.GetRequestID(r.Context())
		
		// Пропускаем логирование для health checks и служебных эндпоинтов
		skipLogging := false
		skipPaths := []string{"/health", "/favicon.ico", "/metrics"}
		for _, path := range skipPaths {
			if r.URL.Path == path {
				skipLogging = true
				break
			}
		}
		
		if !skipLogging {
			// Логируем входящий запрос с структурированным форматом
			slog.Info("Request received",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
		}
		
		// Обертка для ResponseWriter для отслеживания статуса
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(startTime)
		
		if !skipLogging {
			// Логируем завершение запроса с деталями
			attrs := []interface{}{
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
			}
			
			if wrapped.statusCode >= 500 {
				slog.Error("Request completed with error", attrs...)
			} else if wrapped.statusCode >= 400 {
				slog.Warn("Request completed with client error", attrs...)
			} else {
				slog.Info("Request completed successfully", attrs...)
			}
		}
	})
}

// formatDuration удалена - используем duration.Milliseconds() напрямую

// responseWriter обертка для ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

