# Примеры Swagger аннотаций

Этот документ содержит примеры Swagger аннотаций для различных типов эндпоинтов.

## Базовые аннотации

### Минимальный набор

```go
// @Summary Краткое описание
// @Tags databases
// @Router /api/databases [get]
```

### Полный набор

```go
// @Summary Полное описание эндпоинта
// @Description Детальное описание функциональности
// @Tags databases
// @Accept json
// @Produce json
// @Success 200 {object} ResponseModel
// @Failure 400 {object} ErrorResponse
// @Router /api/databases [get]
```

## GET эндпоинты

### Без параметров

```go
// @Summary Получить список баз данных
// @Description Возвращает список всех доступных баз данных
// @Tags databases
// @Accept json
// @Produce json
// @Success 200 {object} DatabaseListResponse "Список баз данных"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/list [get]
func (h *Handler) HandleList(c *gin.Context) {
    // ...
}
```

### С query параметрами

```go
// @Summary Найти базы данных
// @Description Ищет базы данных по запросу
// @Tags databases
// @Accept json
// @Produce json
// @Param q query string true "Поисковый запрос"
// @Param limit query int false "Лимит результатов" default(10)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} DatabaseListResponse "Найденные базы данных"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Router /api/databases/find [get]
func (h *Handler) HandleFind(c *gin.Context) {
    // ...
}
```

### С path параметрами

```go
// @Summary Получить информацию о базе данных
// @Description Возвращает детальную информацию о базе данных по ID
// @Tags databases
// @Accept json
// @Produce json
// @Param id path int true "ID базы данных"
// @Success 200 {object} DatabaseInfoResponse "Информация о базе данных"
// @Failure 404 {object} ErrorResponse "База данных не найдена"
// @Router /api/databases/{id} [get]
func (h *Handler) HandleGet(c *gin.Context) {
    // ...
}
```

### Комбинация path и query параметров

```go
// @Summary Получить элементы базы данных
// @Description Возвращает элементы базы данных с пагинацией
// @Tags databases
// @Accept json
// @Produce json
// @Param id path int true "ID базы данных"
// @Param page query int false "Номер страницы" default(1)
// @Param page_size query int false "Размер страницы" default(20)
// @Success 200 {object} DatabaseItemsResponse "Элементы базы данных"
// @Router /api/databases/{id}/items [get]
func (h *Handler) HandleGetItems(c *gin.Context) {
    // ...
}
```

## POST эндпоинты

### С JSON body

```go
// @Summary Создать новую базу данных
// @Description Создает новую базу данных с указанными параметрами
// @Tags databases
// @Accept json
// @Produce json
// @Param request body CreateDatabaseRequest true "Данные для создания базы данных"
// @Success 201 {object} DatabaseResponse "Созданная база данных"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases [post]
func (h *Handler) HandleCreate(c *gin.Context) {
    // ...
}
```

### С path параметрами и body

```go
// @Summary Обновить базу данных
// @Description Обновляет информацию о базе данных
// @Tags databases
// @Accept json
// @Produce json
// @Param id path int true "ID базы данных"
// @Param request body UpdateDatabaseRequest true "Данные для обновления"
// @Success 200 {object} DatabaseResponse "Обновленная база данных"
// @Failure 404 {object} ErrorResponse "База данных не найдена"
// @Router /api/databases/{id} [put]
func (h *Handler) HandleUpdate(c *gin.Context) {
    // ...
}
```

## DELETE эндпоинты

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
func (h *Handler) HandleDelete(c *gin.Context) {
    // ...
}
```

## PATCH эндпоинты

```go
// @Summary Частично обновить базу данных
// @Description Обновляет отдельные поля базы данных
// @Tags databases
// @Accept json
// @Produce json
// @Param id path int true "ID базы данных"
// @Param request body PatchDatabaseRequest true "Данные для частичного обновления"
// @Success 200 {object} DatabaseResponse "Обновленная база данных"
// @Router /api/databases/{id} [patch]
func (h *Handler) HandlePatch(c *gin.Context) {
    // ...
}
```

## Множественные статусы

```go
// @Summary Запустить процесс
// @Description Запускает процесс обработки данных
// @Tags processes
// @Accept json
// @Produce json
// @Param request body StartProcessRequest true "Параметры запуска"
// @Success 200 {object} ProcessResponse "Процесс запущен"
// @Success 202 {object} ProcessResponse "Процесс принят в обработку"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Failure 409 {object} ErrorResponse "Процесс уже запущен"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/processes/start [post]
func (h *Handler) HandleStart(c *gin.Context) {
    // ...
}
```

## Заголовки запросов

```go
// @Summary Получить данные с авторизацией
// @Description Возвращает данные, требующие авторизации
// @Tags data
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param X-Request-ID header string false "Request ID"
// @Success 200 {object} DataResponse "Данные"
// @Failure 401 {object} ErrorResponse "Не авторизован"
// @Router /api/data [get]
func (h *Handler) HandleGetData(c *gin.Context) {
    // ...
}
```

## Файлы и загрузка

```go
// @Summary Загрузить файл
// @Description Загружает файл на сервер
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл для загрузки"
// @Param description formData string false "Описание файла"
// @Success 200 {object} UploadResponse "Файл загружен"
// @Router /api/upload [post]
func (h *Handler) HandleUpload(c *gin.Context) {
    // ...
}
```

## Server-Sent Events (SSE)

```go
// @Summary Получить события в реальном времени
// @Description Устанавливает SSE соединение для получения событий
// @Tags events
// @Accept json
// @Produce text/event-stream
// @Param process_id query string true "ID процесса"
// @Success 200 "SSE соединение установлено"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Router /api/events/stream [get]
func (h *Handler) HandleStream(c *gin.Context) {
    // ...
}
```

## Аннотации моделей

### Описание структуры

```go
// DatabaseInfo структура информации о базе данных
// @Description Информация о базе данных в системе
type DatabaseInfo struct {
    ID          int    `json:"id" example:"1"`                           // ID базы данных
    Name        string `json:"name" example:"test.db"`                   // Имя базы данных
    Path        string `json:"path" example:"/path/to/test.db"`          // Путь к файлу
    Size        int64  `json:"size" example:"1024000"`                   // Размер в байтах
    CreatedAt   string `json:"created_at" example:"2025-01-20T10:00:00Z"` // Дата создания
    UpdatedAt   string `json:"updated_at" example:"2025-01-20T10:00:00Z"` // Дата обновления
}
```

### Вложенные структуры

```go
// NestedResponse структура с вложенными данными
type NestedResponse struct {
    MainData MainData     `json:"main_data"`
    Metadata []Metadata   `json:"metadata"`
}

// MainData основные данные
type MainData struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

// Metadata метаданные
type Metadata struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}
```

## Группировка эндпоинтов (Tags)

Используйте теги для логической группировки:

- `databases` - работа с базами данных
- `normalization` - нормализация данных
- `classification` - классификация данных
- `quality` - качество данных
- `monitoring` - мониторинг системы
- `reports` - отчеты
- `upload` - загрузка данных
- `counterparties` - работа с контрагентами
- `clients` - управление клиентами
- `projects` - управление проектами

## Примеры из проекта

### Databases

```go
// @Summary Получить список всех баз данных
// @Description Возвращает список всех доступных баз данных в системе
// @Tags databases
// @Accept json
// @Produce json
// @Success 200 {object} DatabaseListResponse "Список баз данных"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/databases/list [get]
```

### Normalization

```go
// @Summary Запустить нормализацию данных
// @Description Запускает процесс нормализации данных для указанной базы данных
// @Tags normalization
// @Accept json
// @Produce json
// @Param request body NormalizationStartRequest true "Параметры запуска нормализации"
// @Success 200 {object} map[string]interface{} "Успешный запуск нормализации"
// @Failure 400 {object} ErrorResponse "Неверный запрос"
// @Router /api/normalize/start [post]
```

### Monitoring

```go
// @Summary Получить статус AI-провайдеров
// @Description Возвращает текущий статус и метрики всех AI-провайдеров
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} MonitoringDataResponse "Данные мониторинга провайдеров"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/monitoring/providers [get]
```

## См. также

- [Swagger Usage Guide](./SWAGGER_USAGE.md) - Руководство по использованию Swagger
- [Migration to Gin Guide](./MIGRATION_TO_GIN.md) - Руководство по миграции на Gin

