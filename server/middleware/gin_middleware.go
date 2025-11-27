package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GinRequestIDMiddleware добавляет уникальный request ID к каждому запросу в Gin
func GinRequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Генерируем или получаем request ID из заголовка
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// Добавляем request ID в контекст Gin
		c.Set("request_id", reqID)

		// Добавляем request ID в контекст HTTP
		ctx := SetRequestID(c.Request.Context(), reqID)
		c.Request = c.Request.WithContext(ctx)

		// Добавляем request ID в заголовок ответа
		c.Header("X-Request-ID", reqID)

		c.Next()
	}
}

// GetRequestIDFromGin извлекает request ID из Gin context
func GetRequestIDFromGin(c *gin.Context) string {
	if c == nil {
		return ""
	}

	reqID, exists := c.Get("request_id")
	if !exists {
		return ""
	}

	if id, ok := reqID.(string); ok {
		return id
	}

	return ""
}

// GinCORSMiddleware добавляет CORS заголовки в Gin
func GinCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// GinGzipMiddleware включает сжатие ответов
func GinGzipMiddleware() gin.HandlerFunc {
	return gzip.Gzip(gzip.BestSpeed)
}

// GinLoggerMiddleware логирует запросы в Gin (улучшенная версия gin.Logger())
func GinLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Latency
		latency := time.Since(start)

		// Client IP
		clientIP := c.ClientIP()

		// Method
		method := c.Request.Method

		// Status code
		statusCode := c.Writer.Status()

		// Request ID
		reqID := GetRequestIDFromGin(c)

		// Body size
		bodySize := c.Writer.Size()

		if raw != "" {
			path = path + "?" + raw
		}

		// Логирование в формате Gin
		timestamp := time.Now().Format("2006/01/02 - 15:04:05")
		logLine := fmt.Sprintf(
			"[%s] %s [%s] %s %d %s %d",
			timestamp,
			clientIP,
			method,
			path,
			statusCode,
			latency,
			bodySize,
		)
		if reqID != "" {
			logLine += " [RequestID: " + reqID + "]"
		}
		if err := c.Errors.Last(); err != nil {
			logLine += " [Error: " + err.Error() + "]"
		}
		gin.DefaultWriter.Write([]byte(logLine + "\n"))

		// Дополнительное логирование с request ID
		if reqID != "" {
			gin.DefaultWriter.Write([]byte(
				"[GIN] Request ID: " + reqID + "\n",
			))
		}
	}
}

// GinRecoveryMiddleware обрабатывает паники в Gin
func GinRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				reqID := GetRequestIDFromGin(c)
				stackTrace := debug.Stack()

				// Логируем панику через slog
				slog.Error("[GIN] Panic recovered",
					"panic", err,
					"stack", string(stackTrace),
					"request_id", reqID,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
				)

				// Также пишем в gin.DefaultErrorWriter для совместимости
				gin.DefaultErrorWriter.Write([]byte(
					"[GIN] Panic recovered: " + toString(err) + "\n",
				))
				gin.DefaultErrorWriter.Write(stackTrace)

				if reqID != "" {
					gin.DefaultErrorWriter.Write([]byte(
						"[GIN] Request ID: " + reqID + "\n",
					))
				}

				// Отправляем JSON ошибку
				c.JSON(500, gin.H{
					"error":      true,
					"message":    "Internal server error",
					"request_id": reqID,
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// toString преобразует значение в строку
func toString(v interface{}) string {
	if str, ok := v.(string); ok {
		return str
	}
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return ""
}
