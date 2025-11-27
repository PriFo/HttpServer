# Регистрация Gin Handlers

Этот документ описывает процесс регистрации Gin handlers в роутере.

## Текущая структура

В `server/server.go` метод `setupRouter()` создает Gin роутер и регистрирует handlers в следующем порядке:

1. **Gin Middleware** - RequestID, CORS, Logger, Recovery
2. **Swagger UI** - `/swagger/*any`
3. **Gin Handlers** - новые мигрированные handlers (через `registerGinHandlers()`)
4. **Старые Handlers** - через `NoRoute` и `http.ServeMux`

## Регистрация Gin Handlers

Функция `registerGinHandlers()` регистрирует все мигрированные Gin handlers:

```go
func (s *Server) registerGinHandlers(router *gin.Engine) {
    api := router.Group("/api")
    
    // Databases API
    if s.databaseHandler != nil {
        databasesAPI := api.Group("/databases")
        {
            databasesAPI.GET("/list", s.databaseHandler.HandleDatabasesListGin)
            databasesAPI.GET("/find", s.databaseHandler.HandleFindDatabaseGin)
        }
        
        databaseAPI := api.Group("/database")
        {
            databaseAPI.GET("/info", s.databaseHandler.HandleDatabaseInfoGin)
        }
    }
    
    // Monitoring, Quality, Classification...
}
```

## Приоритет маршрутов

Gin handlers, зарегистрированные через `registerGinHandlers()`, имеют **приоритет** перед старыми handlers через `NoRoute`. Это означает:

- Если маршрут зарегистрирован в Gin, он обрабатывается Gin handler
- Если маршрут не найден в Gin, запрос передается в старый `http.ServeMux` через `NoRoute`

## Добавление новых Gin Handlers

### Шаг 1: Создать Gin handler

Создайте файл `server/handlers/yourhandler_gin.go`:

```go
package handlers

import (
    "github.com/gin-gonic/gin"
)

// @Summary Описание эндпоинта
// @Tags yourtag
// @Router /api/your-endpoint [get]
func (h *YourHandler) HandleYourEndpointGin(c *gin.Context) {
    // Логика обработчика
}
```

### Шаг 2: Зарегистрировать в registerGinHandlers()

Добавьте регистрацию в `server/server.go`:

```go
func (s *Server) registerGinHandlers(router *gin.Engine) {
    api := router.Group("/api")
    
    // Your API
    if s.yourHandler != nil {
        yourAPI := api.Group("/your")
        {
            yourAPI.GET("/endpoint", s.yourHandler.HandleYourEndpointGin)
        }
    }
    
    // ... другие handlers
}
```

## Группировка маршрутов

Используйте группы Gin для лучшей организации:

```go
databasesAPI := api.Group("/databases")
{
    databasesAPI.GET("/list", handler.HandleList)
    databasesAPI.GET("/:id", handler.HandleGet)
    databasesAPI.POST("/", handler.HandleCreate)
    databasesAPI.PUT("/:id", handler.HandleUpdate)
    databasesAPI.DELETE("/:id", handler.HandleDelete)
}
```

## Middleware для групп

Можно добавить middleware для конкретных групп:

```go
authAPI := api.Group("/auth")
authAPI.Use(AuthMiddleware()) // Применяется ко всем маршрутам в группе
{
    authAPI.POST("/login", handler.HandleLogin)
    authAPI.POST("/logout", handler.HandleLogout)
}
```

## Примеры зарегистрированных handlers

### Databases
- `GET /api/databases/list` - список баз данных
- `GET /api/databases/find` - поиск баз данных
- `GET /api/database/info` - информация о базе данных

### Monitoring
- `GET /api/monitoring/providers` - статус провайдеров
- `GET /api/monitoring/providers/:id` - метрики провайдера
- `POST /api/monitoring/providers/:id/start` - запуск провайдера
- `POST /api/monitoring/providers/:id/stop` - остановка провайдера

### Quality
- `GET /api/quality/report` - отчет о качестве
- `GET /api/quality/score/:database_id` - оценка качества

### Classification
- `POST /api/classification/classify` - классификация элемента
- `GET /api/classification/stats` - статистика классификации

## Порядок миграции

1. Создайте Gin handler с полными Swagger аннотациями
2. Добавьте регистрацию в `registerGinHandlers()`
3. Протестируйте новый handler
4. Убедитесь, что старый handler все еще работает (через NoRoute)
5. После проверки можно удалить старый handler

## Замечания

- **Приоритет**: Gin handlers всегда имеют приоритет перед старыми handlers
- **Обратная совместимость**: Старые handlers продолжают работать через NoRoute
- **Постепенная миграция**: Можно мигрировать handlers постепенно, один за другим
- **Тестирование**: Всегда тестируйте новые handlers перед удалением старых

## См. также

- [Migration to Gin Guide](./MIGRATION_TO_GIN.md) - подробное руководство по миграции
- [Swagger Annotations Examples](./SWAGGER_ANNOTATIONS_EXAMPLES.md) - примеры аннотаций
- [Examples Index](./EXAMPLES_INDEX.md) - индекс примеров кода

