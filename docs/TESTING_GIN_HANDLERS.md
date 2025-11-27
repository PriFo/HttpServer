# Руководство по тестированию Gin Handlers

Это руководство описывает процесс тестирования handlers, мигрированных на Gin Framework.

## Структура тестов

### Базовый тест

```go
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

// TestHandlerExample тестирует пример handler
func TestHandlerExample(t *testing.T) {
	// Подготовка
	router := setupGinTestRouter()
	
	// Создаем handler с моками сервисов (если нужно)
	handler := &DatabaseHandler{
		databaseService: mockService, // mock или реальный сервис
		baseHandler:     NewBaseHandlerFromMiddleware(),
	}
	
	// Регистрируем маршрут
	router.GET("/api/test", handler.HandleExample)
	
	// Выполнение
	req, _ := http.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Проверка
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
```

## Тестирование с использованием gin.CreateTestContext

Для более детального тестирования можно использовать `gin.CreateTestContext`:

```go
func TestHandlerWithContext(t *testing.T) {
	// Создаем тестовый контекст
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// Настраиваем request
	c.Request, _ = http.NewRequest("GET", "/api/test", nil)
	
	// Вызываем handler
	handler := &DatabaseHandler{...}
	handler.HandleExample(c)
	
	// Проверка
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
```

## Тестирование query параметров

```go
func TestHandlerWithQueryParams(t *testing.T) {
	router := setupGinTestRouter()
	router.GET("/api/databases/find", handler.HandleFindDatabaseGin)
	
	// Тест с параметрами
	req, _ := http.NewRequest("GET", "/api/databases/find?q=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Проверка
	assert.Equal(t, http.StatusOK, w.Code)
}
```

## Тестирование path параметров

```go
func TestHandlerWithPathParams(t *testing.T) {
	router := setupGinTestRouter()
	router.GET("/api/databases/:id", handler.HandleGetDatabaseGin)
	
	req, _ := http.NewRequest("GET", "/api/databases/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Проверка
	assert.Equal(t, http.StatusOK, w.Code)
}
```

## Тестирование body параметров

```go
func TestHandlerWithBody(t *testing.T) {
	router := setupGinTestRouter()
	router.POST("/api/normalize/start", handler.HandleNormalizeStartGin)
	
	// Создаем JSON body
	body := `{"database_id": 1}`
	req, _ := http.NewRequest("POST", "/api/normalize/start", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Проверка
	assert.Equal(t, http.StatusOK, w.Code)
}
```

## Тестирование middleware

Для тестирования middleware создайте отдельные тесты:

```go
func TestGinRequestIDMiddleware(t *testing.T) {
	router := setupGinTestRouter()
	router.Use(GinRequestIDMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		reqID := GetRequestIDFromGin(c)
		if reqID == "" {
			t.Error("Request ID should not be empty")
		}
		c.JSON(200, gin.H{"request_id": reqID})
	})
	
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}
```

## Примеры

См. файл `server/handlers/databases_gin_test.go` для примеров тестирования Gin handlers.

## Запуск тестов

```bash
# Все тесты
go test ./...

# Конкретный пакет
go test ./server/handlers

# С покрытием
go test -cover ./server/handlers

# Вербозный вывод
go test -v ./server/handlers
```

## Примечания

- Используйте `gin.SetMode(gin.TestMode)` для тестового режима
- Избегайте циклических зависимостей при тестировании
- Для моков сервисов используйте интерфейсы
- Тестируйте как успешные, так и ошибочные сценарии

