# Руководство по использованию Swagger

Это руководство описывает процесс работы со Swagger документацией в проекте.

## Быстрый старт

### 1. Генерация документации

После добавления или изменения Swagger аннотаций в handlers:

```bash
make swagger
```

Или вручную:
```bash
swag init -g main.go -o ./docs
```

### 2. Доступ к Swagger UI

После запуска сервера, Swagger UI будет доступен по адресу:

```
http://localhost:9999/swagger/index.html
```

## Добавление новых эндпоинтов в документацию

### Базовые аннотации

Каждый handler должен иметь следующие аннотации:

```go
// @Summary Краткое описание эндпоинта (одна строка)
// @Description Детальное описание эндпоинта (может быть многострочным)
// @Tags тег-группы (группирует эндпоинты в Swagger UI)
// @Accept json
// @Produce json
// @Router /api/path [method]
```

### Примеры аннотаций

#### GET эндпоинт с query параметрами

```go
// @Summary Получить список баз данных
// @Description Возвращает список всех доступных баз данных в системе
// @Tags databases
// @Accept json
// @Produce json
// @Param q query string true "Поисковый запрос"
// @Param limit query int false "Лимит результатов" default(10)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} DatabaseListResponse "Список баз данных"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/list [get]
func (h *DatabaseHandler) HandleDatabasesListGin(c *gin.Context) {
    // ...
}
```

#### POST эндпоинт с body

```go
// @Summary Запустить нормализацию данных
// @Description Запускает процесс нормализации данных для указанной базы данных
// @Tags normalization
// @Accept json
// @Produce json
// @Param request body NormalizationStartRequest true "Параметры запуска нормализации"
// @Success 200 {object} map[string]interface{} "Успешный запуск"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/normalize/start [post]
func (h *NormalizationHandler) HandleNormalizeStartGin(c *gin.Context) {
    // ...
}
```

#### GET эндпоинт с path параметрами

```go
// @Summary Получить информацию о базе данных
// @Description Возвращает детальную информацию о базе данных по ID
// @Tags databases
// @Accept json
// @Produce json
// @Param id path int true "ID базы данных"
// @Success 200 {object} DatabaseInfoResponse "Информация о базе данных"
// @Failure 404 {object} ErrorResponse "База данных не найдена"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/{id} [get]
func (h *DatabaseHandler) HandleGetDatabaseGin(c *gin.Context) {
    // ...
}
```

#### DELETE эндпоинт

```go
// @Summary Удалить базу данных
// @Description Удаляет базу данных по ID
// @Tags databases
// @Accept json
// @Produce json
// @Param id path int true "ID базы данных"
// @Success 200 {object} map[string]interface{} "Успешное удаление"
// @Failure 404 {object} ErrorResponse "База данных не найдена"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/{id} [delete]
func (h *DatabaseHandler) HandleDeleteDatabaseGin(c *gin.Context) {
    // ...
}
```

## Определение моделей

Для правильной документации нужно определить структуры моделей:

### Пример определения моделей

```go
// DatabaseListResponse структура ответа для списка баз данных
type DatabaseListResponse struct {
    Databases []interface{} `json:"databases"`
    Total     int           `json:"total"`
}

// NormalizationStartRequest структура запроса для запуска нормализации
type NormalizationStartRequest struct {
    DatabaseID int                    `json:"database_id"`
    ClientID   int                    `json:"client_id,omitempty"`
    ProjectID  int                    `json:"project_id,omitempty"`
    Options    map[string]interface{} `json:"options,omitempty"`
}

// ErrorResponse структура ошибки
type ErrorResponse struct {
    Error   bool   `json:"error"`
    Message string `json:"message"`
}
```

### Аннотации для моделей

Можно добавить описания к полям структур:

```go
// DatabaseInfo структура информации о базе данных
// @Description Информация о базе данных
type DatabaseInfo struct {
    ID          int    `json:"id" example:"1"`          // ID базы данных
    Name        string `json:"name" example:"test.db"`  // Имя базы данных
    Path        string `json:"path" example:"/path/to/test.db"` // Путь к файлу
    Size        int64  `json:"size" example:"1024000"`  // Размер в байтах
    CreatedAt   string `json:"created_at" example:"2025-01-20T10:00:00Z"` // Дата создания
    UpdatedAt   string `json:"updated_at" example:"2025-01-20T10:00:00Z"` // Дата обновления
}
```

## Группировка эндпоинтов (Tags)

Используйте теги для группировки эндпоинтов в Swagger UI:

- `databases` - работа с базами данных
- `normalization` - нормализация данных
- `classification` - классификация данных
- `quality` - качество данных
- `monitoring` - мониторинг
- `reports` - отчеты
- `upload` - загрузка данных
- `counterparties` - работа с контрагентами

## Множественные статусы ответа

Можно указать несколько возможных статусов:

```go
// @Success 200 {object} SuccessResponse "Успешный ответ"
// @Success 201 {object} CreatedResponse "Создан новый ресурс"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 401 {object} ErrorResponse "Не авторизован"
// @Failure 403 {object} ErrorResponse "Доступ запрещен"
// @Failure 404 {object} ErrorResponse "Не найдено"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
```

## Заголовки запросов

Для работы с заголовками:

```go
// @Param Authorization header string true "Bearer token"
// @Param X-Request-ID header string false "Request ID"
```

## Запуск тестов через Swagger UI

1. Откройте Swagger UI: `http://localhost:9999/swagger/index.html`
2. Найдите нужный эндпоинт
3. Нажмите "Try it out"
4. Заполните параметры
5. Нажмите "Execute"
6. Посмотрите ответ

## Типичные проблемы

### Документация не обновляется

1. Убедитесь, что запустили `make swagger`
2. Проверьте, что аннотации находятся непосредственно перед функцией
3. Убедитесь, что функция экспортирована (начинается с заглавной буквы)

### Модели не отображаются

1. Убедитесь, что структуры определены в том же пакете или импортированы
2. Проверьте, что структуры экспортированы
3. Убедитесь, что используются правильные JSON теги

### Параметры не распознаются

1. Проверьте синтаксис аннотаций `@Param`
2. Убедитесь, что указан правильный тип параметра (query, path, body, header)
3. Проверьте, что параметр действительно используется в функции

## Дополнительные ресурсы

- [Swagger/OpenAPI спецификация](https://swagger.io/specification/)
- [swaggo документация](https://github.com/swaggo/swag)
- [Gin документация](https://gin-gonic.com/docs/)

