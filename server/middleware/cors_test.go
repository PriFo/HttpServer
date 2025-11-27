package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORS проверяет добавление CORS заголовков
func TestCORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := CORS(handler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	middleware.ServeHTTP(w, req)
	
	// Проверяем CORS заголовки
	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Cache-Control",
	}
	
	for header, expectedValue := range headers {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Header %s = %v, want %v", header, actualValue, expectedValue)
		}
	}
}

// TestCORS_OPTIONS проверяет обработку preflight запросов
func TestCORS_OPTIONS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := CORS(handler)
	
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	
	middleware.ServeHTTP(w, req)
	
	// OPTIONS запрос должен вернуть 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("OPTIONS request status = %d, want %d", w.Code, http.StatusOK)
	}
	
	// Проверяем, что CORS заголовки установлены
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Access-Control-Allow-Origin header should be set for OPTIONS request")
	}
}

// TestCORS_RegularRequest проверяет, что обычные запросы проходят через middleware
func TestCORS_RegularRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("test response"))
	})
	
	middleware := CORS(handler)
	
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	
	middleware.ServeHTTP(w, req)
	
	// Проверяем, что запрос прошел через handler
	if w.Code != http.StatusCreated {
		t.Errorf("Response status = %d, want %d", w.Code, http.StatusCreated)
	}
	
	// Проверяем, что CORS заголовки установлены
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Access-Control-Allow-Origin header should be set")
	}
	
	// Проверяем тело ответа
	if w.Body.String() != "test response" {
		t.Errorf("Response body = %s, want 'test response'", w.Body.String())
	}
}

// TestCORS_AllMethods проверяет поддержку всех методов
func TestCORS_AllMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := CORS(handler)
	
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()
			
			middleware.ServeHTTP(w, req)
			
			// Проверяем, что CORS заголовки установлены для всех методов
			if w.Header().Get("Access-Control-Allow-Origin") == "" {
				t.Errorf("Access-Control-Allow-Origin header should be set for %s request", method)
			}
		})
	}
}

