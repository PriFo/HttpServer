package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"httpserver/server/middleware"
)

// TestSecurityHeadersMiddleware проверяет добавление заголовков безопасности
func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := SecurityHeadersMiddleware(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	mw.ServeHTTP(w, req)
	
	// Проверяем заголовки безопасности
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":         "DENY",
		"X-XSS-Protection":         "1; mode=block",
		"Referrer-Policy":          "strict-origin-when-cross-origin",
	}
	
	for header, expectedValue := range headers {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Header %s = %v, want %v", header, actualValue, expectedValue)
		}
	}
}

// TestSecurityHeadersMiddleware_CORS проверяет CORS заголовки
func TestSecurityHeadersMiddleware_CORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := SecurityHeadersMiddleware(handler)
	
	tests := []struct {
		name           string
		origin         string
		wantCORSHeader bool
	}{
		{
			name:           "with origin",
			origin:         "https://example.com",
			wantCORSHeader: true,
		},
		{
			name:           "without origin",
			origin:         "",
			wantCORSHeader: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			
			mw.ServeHTTP(w, req)
			
			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			if (corsHeader != "") != tt.wantCORSHeader {
				t.Errorf("CORS header present = %v, want %v", corsHeader != "", tt.wantCORSHeader)
			}
			
			if tt.wantCORSHeader && corsHeader != tt.origin {
				t.Errorf("Access-Control-Allow-Origin = %v, want %v", corsHeader, tt.origin)
			}
		})
	}
}

// TestSecurityHeadersMiddleware_OPTIONS проверяет обработку preflight запросов
func TestSecurityHeadersMiddleware_OPTIONS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := SecurityHeadersMiddleware(handler)
	
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	
	mw.ServeHTTP(w, req)
	
	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS request status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

// TestRequestIDMiddleware проверяет добавление request ID
func TestRequestIDMiddleware(t *testing.T) {
	var capturedRequestID string
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusOK)
	})
	
	mw := middleware.RequestIDMiddleware(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	mw.ServeHTTP(w, req)
	
	// Проверяем, что request ID был добавлен
	if capturedRequestID == "" {
		t.Error("Request ID was not set in request header")
	}
	
	// Проверяем, что request ID был добавлен в response header
	responseID := w.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Error("Request ID was not set in response header")
	}
	
	if capturedRequestID != responseID {
		t.Errorf("Request ID mismatch: request=%v, response=%v", capturedRequestID, responseID)
	}
}

// TestRequestIDMiddleware_ExistingID проверяет использование существующего request ID
func TestRequestIDMiddleware_ExistingID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := middleware.RequestIDMiddleware(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	w := httptest.NewRecorder()
	
	mw.ServeHTTP(w, req)
	
	responseID := w.Header().Get("X-Request-ID")
	if responseID != "existing-id" {
		t.Errorf("Request ID = %v, want existing-id", responseID)
	}
}

// TestLoggingMiddleware проверяет логирование запросов
func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := LoggingMiddleware(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	mw.ServeHTTP(w, req)
	
	// Проверяем, что запрос был обработан
	if w.Code != http.StatusOK {
		t.Errorf("Request status = %v, want %v", w.Code, http.StatusOK)
	}
}

// TestLoggingMiddleware_SkipPaths проверяет пропуск логирования для служебных путей
func TestLoggingMiddleware_SkipPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := LoggingMiddleware(handler)
	
	skipPaths := []string{"/health", "/favicon.ico", "/metrics"}
	
	for _, path := range skipPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			
			mw.ServeHTTP(w, req)
			
			// Проверяем, что запрос был обработан (логирование пропущено, но обработка выполнена)
			if w.Code != http.StatusOK {
				t.Errorf("Request status = %v, want %v", w.Code, http.StatusOK)
			}
		})
	}
}

// TestLoggingMiddleware_StatusCodes проверяет логирование для разных статус кодов
func TestLoggingMiddleware_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
		},
		{
			name:       "client error",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})
			
			mw := LoggingMiddleware(handler)
			
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			
			mw.ServeHTTP(w, req)
			
			if w.Code != tt.statusCode {
				t.Errorf("Request status = %v, want %v", w.Code, tt.statusCode)
			}
		})
	}
}

// TestFormatDuration проверяет форматирование длительности
// Примечание: formatDuration была удалена, тест оставлен для документации
// Если функция будет восстановлена, тест можно будет использовать
func TestFormatDuration(t *testing.T) {
	t.Skip("formatDuration function was removed, test skipped")
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
		t.Errorf("Initial status code = %v, want %v", rw.statusCode, http.StatusOK)
	}
	
	// Устанавливаем новый статус
	newStatus := http.StatusNotFound
	rw.WriteHeader(newStatus)
	
	if rw.statusCode != newStatus {
		t.Errorf("Status code after WriteHeader = %v, want %v", rw.statusCode, newStatus)
	}
	
	if w.Code != newStatus {
		t.Errorf("Underlying ResponseWriter status = %v, want %v", w.Code, newStatus)
	}
}

