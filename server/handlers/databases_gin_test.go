package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupGinTestRouter создает тестовый Gin роутер
func setupGinTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// TestGinHelpers тестирует helper функции для Gin
func TestGinHelpers(t *testing.T) {
	t.Run("ValidateMethod allows correct method", func(t *testing.T) {
		router := setupGinTestRouter()
		called := false

		router.GET("/test", func(c *gin.Context) {
			if ValidateMethod(c, "GET") {
				called = true
				c.JSON(200, gin.H{"success": true})
			}
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !called {
			t.Error("Handler should be called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ValidateMethod rejects incorrect method", func(t *testing.T) {
		router := setupGinTestRouter()

		router.GET("/test", func(c *gin.Context) {
			if !ValidateMethod(c, "POST") {
				// Method not allowed
				return
			}
			c.JSON(200, gin.H{"success": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("SendJSONResponse sends correct response", func(t *testing.T) {
		router := setupGinTestRouter()

		router.GET("/test", func(c *gin.Context) {
			SendJSONResponse(c, http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() == "" {
			t.Error("Response body should not be empty")
		}
	})

	t.Run("SendJSONError sends correct error", func(t *testing.T) {
		router := setupGinTestRouter()

		router.GET("/test", func(c *gin.Context) {
			SendJSONError(c, http.StatusBadRequest, "Test error")
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
		if w.Body.String() == "" {
			t.Error("Response body should not be empty")
		}
	})
}

