package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"httpserver/server/middleware"
)

// GinHandlerFunc адаптирует http.HandlerFunc в gin.HandlerFunc
func GinHandlerFunc(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c.Writer, c.Request)
	}
}

// GinHandler адаптирует http.Handler в gin.HandlerFunc
func GinHandler(handler http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

// SendJSONResponse отправляет JSON ответ через Gin context
func SendJSONResponse(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// SendJSONError отправляет JSON ошибку через Gin context и логирует её
func SendJSONError(c *gin.Context, statusCode int, message string) {
	reqID := middleware.GetRequestIDFromGin(c)
	
	// Логируем ошибку
	slog.Error("Gin HTTP error",
		"error", message,
		"status_code", statusCode,
		"request_id", reqID,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)
	
	c.JSON(statusCode, gin.H{
		"error":   true,
		"message": message,
	})
}

// ValidateMethod проверяет HTTP метод и возвращает ошибку если не совпадает
func ValidateMethod(c *gin.Context, allowedMethod string) bool {
	if c.Request.Method != allowedMethod {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":   true,
			"message": "Method not allowed",
		})
		return false
	}
	return true
}

