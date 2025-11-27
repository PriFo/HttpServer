# Руководство по миграции на Gin Framework

Это руководство описывает процесс миграции handlers с `http.HandlerFunc` на `gin.HandlerFunc` с добавлением Swagger аннотаций.

## Текущее состояние

Проект частично мигрирован на Gin:
- Основной роутер использует `gin.Engine`
- Старые handlers работают через адаптер (`NoRoute`)
- Swagger UI доступен по адресу `/swagger/index.html`

## Процесс миграции handler

### Шаг 1: Создание Gin версии handler

Для каждого handler создайте Gin-версию:

**Было (http.HandlerFunc):**
```go
func (h *DatabaseHandler) HandleDatabasesList(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
        return
    }
    
    databases, err := h.databaseService.ListDatabases()
    if err != nil {
        h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("error: %v", err), http.StatusInternalServerError)
        return
    }
    
    h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
        "databases": databases,
        "total": len(databases),
    }, http.StatusOK)
}
```

**Стало (gin.HandlerFunc):**
```go
// @Summary Получить список всех баз данных
// @Description Возвращает список всех доступных баз данных в системе
// @Tags databases
// @Accept json
// @Produce json
// @Success 200 {object} DatabaseListResponse "Список баз данных"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/list [get]
func (h *DatabaseHandler) HandleDatabasesListGin(c *gin.Context) {
    databases, err := h.databaseService.ListDatabases()
    if err != nil {
        JSONError(c, http.StatusInternalServerError, fmt.Sprintf("error: %v", err))
        return
    }
    
    JSONResponse(c, http.StatusOK, DatabaseListResponse{
        Databases: databases,
        Total: len(databases),
    })
}
```

### Шаг 2: Добавление Swagger аннотаций

Каждый handler должен иметь следующие аннотации:

```go
// @Summary Краткое описание эндпоинта
// @Description Детальное описание эндпоинта
// @Tags тег-группы
// @Accept json
// @Produce json
// @Param параметр query/path/body тип required "Описание"
// @Success 200 {object} ResponseModel "Успешный ответ"
// @Failure 400 {object} ErrorResponse "Ошибка запроса"
// @Router /api/path [method]
```

**Примеры аннотаций:**

**Query параметры:**
```go
// @Param q query string true "Поисковый запрос"
```

**Path параметры:**
```go
// @Param id path int true "ID базы данных"
```

**Body параметры:**
```go
// @Param request body CreateDatabaseRequest true "Данные для создания"
```

**Множественные статусы:**
```go
// @Success 200 {object} SuccessResponse "Успешно"
// @Success 201 {object} CreatedResponse "Создано"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 404 {object} ErrorResponse "Не найдено"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
```

### Шаг 3: Использование helper функций

Используйте helper функции из `gin_helpers.go`:

- `JSONResponse(c, statusCode, data)` - отправка JSON ответа
- `JSONError(c, statusCode, message)` - отправка JSON ошибки
- `ValidateMethod(c, method)` - проверка HTTP метода
- `GinHandlerFunc(http.HandlerFunc)` - адаптация старого handler

### Шаг 4: Определение моделей ответов

Определите структуры для Swagger документации:

```go
// DatabaseListResponse структура ответа для списка баз данных
type DatabaseListResponse struct {
    Databases []interface{} `json:"databases"`
    Total     int           `json:"total"`
}

// ErrorResponse структура ошибки
type ErrorResponse struct {
    Error   bool   `json:"error"`
    Message string `json:"message"`
}
```

### Шаг 5: Регистрация маршрута в Gin

После создания Gin-версии handler, зарегистрируйте маршрут в `server/server.go`:

**В методе `setupRouter()` добавьте:**

```go
// Регистрируем Gin handlers (ПЕРЕД NoRoute)
router.GET("/api/databases/list", h.databaseHandler.HandleDatabasesListGin)
router.GET("/api/database/info", h.databaseHandler.HandleDatabaseInfoGin)
router.GET("/api/databases/find", h.databaseHandler.HandleFindDatabaseGin)
```

### Шаг 6: Удаление старого handler

После проверки работоспособности Gin-версии:
1. Удалите старый `http.HandlerFunc` handler
2. Удалите регистрацию в `http.ServeMux`
3. Переименуйте Gin-версию (уберите суффикс `Gin`)

## Примеры миграции

См. файлы:
- `server/handlers/databases_gin.go` - пример миграции DatabaseHandler
- `server/handlers/gin_helpers.go` - helper функции

## Проверка миграции

После миграции handler:

1. **Генерация Swagger документации:**
   ```bash
   make swagger
   ```

2. **Проверка Swagger UI:**
   - Откройте `http://localhost:9999/swagger/index.html`
   - Найдите ваш эндпоинт
   - Проверьте, что все параметры и ответы отображаются корректно

3. **Тестирование:**
   - Протестируйте эндпоинт через Swagger UI
   - Убедитесь, что все параметры работают
   - Проверьте валидацию и обработку ошибок

## Приоритет миграции

Рекомендуемый порядок миграции handlers:

1. **Базовые handlers** (databases, health, stats)
2. **CRUD операции** (upload, normalization)
3. **Сложные handlers** (classification, quality, monitoring)
4. **Специальные handlers** (SSE, file upload)

## Примечания

- Старые handlers продолжают работать через адаптер (`NoRoute`)
- Миграция может происходить постепенно
- Все Gin handlers регистрируются ПЕРЕД `NoRoute`
- Gin handlers имеют приоритет над старыми handlers

