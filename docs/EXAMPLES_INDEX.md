# Индекс примеров кода

Эта страница содержит ссылки на все примеры кода в проекте.

## Примеры миграции на Gin

### Handlers

- [databases_gin.go](../server/handlers/databases_gin.go) - пример миграции DatabaseHandler с полными Swagger аннотациями
  - `HandleDatabasesListGin` - список баз данных
  - `HandleDatabaseInfoGin` - информация о базе данных
  - `HandleFindDatabaseGin` - поиск баз данных

- [monitoring_gin.go](../server/handlers/monitoring_gin.go) - пример миграции MonitoringHandler
  - `HandleGetProvidersGin` - статус AI-провайдеров
  - `HandleGetProviderMetricsGin` - метрики провайдера
  - `HandleStartProviderGin` - запуск провайдера
  - `HandleStopProviderGin` - остановка провайдера

- [quality_gin.go](../server/handlers/quality_gin.go) - пример миграции QualityHandler
  - `HandleQualityReportGin` - отчет о качестве данных
  - `HandleQualityScoreGin` - оценка качества

- [classification_gin.go](../server/handlers/classification_gin.go) - пример миграции ClassificationHandler
  - `HandleClassifyItemGin` - классификация элемента
  - `HandleClassificationStatsGin` - статистика классификации

### Helper функции

- [gin_helpers.go](../server/handlers/gin_helpers.go) - helper функции для работы с Gin
  - `SendJSONResponse()` - отправка JSON ответа
  - `SendJSONError()` - отправка JSON ошибки
  - `ValidateMethod()` - проверка HTTP метода
  - `GinHandlerFunc()` - адаптация старого handler

### Middleware

- [gin_middleware.go](../server/middleware/gin_middleware.go) - Gin middleware
  - `GinRequestIDMiddleware()` - добавление request ID
  - `GinCORSMiddleware()` - CORS заголовки
  - `GinLoggerMiddleware()` - логирование запросов
  - `GinRecoveryMiddleware()` - обработка паник

### Тесты

- [databases_gin_test.go](../server/handlers/databases_gin_test.go) - примеры тестов для Gin handlers
  - Тестирование helper функций
  - Тестирование валидации методов
  - Тестирование JSON ответов

## Примеры Swagger аннотаций

См. [SWAGGER_ANNOTATIONS_EXAMPLES.md](./SWAGGER_ANNOTATIONS_EXAMPLES.md) для подробных примеров аннотаций.

### Типы эндпоинтов

- GET с query параметрами
- GET с path параметрами
- POST с JSON body
- PUT/PATCH обновление
- DELETE удаление
- File upload
- Server-Sent Events (SSE)

## Примеры конфигурации

### Swagger

- [main.go](../main.go) - базовая Swagger конфигурация в main.go
- [swagger.go](../server/handlers/swagger.go) - обработчик Swagger UI

### Docker

- [docker-compose.yml](../docker-compose.yml) - конфигурация Docker Compose
- [Dockerfile](../Dockerfile) - Dockerfile для backend

## Использование примеров

1. **Изучите примеры** - начните с `databases_gin.go` для понимания структуры
2. **Используйте как шаблон** - копируйте структуру для новых handlers
3. **Адаптируйте под свои нужды** - изменяйте под конкретные требования
4. **Добавляйте Swagger аннотации** - следуйте примерам из `SWAGGER_ANNOTATIONS_EXAMPLES.md`

## См. также

- [Migration to Gin Guide](./MIGRATION_TO_GIN.md) - подробное руководство по миграции
- [Swagger Usage Guide](./SWAGGER_USAGE.md) - руководство по использованию Swagger
- [Testing Gin Handlers](./TESTING_GIN_HANDLERS.md) - руководство по тестированию

