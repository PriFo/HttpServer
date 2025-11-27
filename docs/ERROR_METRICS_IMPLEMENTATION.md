# Реализация метрик ошибок

## Обзор

Реализована система сбора и мониторинга метрик ошибок для отслеживания типов ошибок, их частоты и распределения по эндпоинтам.

## Компоненты

### 1. ErrorMetricsCollector (`server/errors/metrics.go`)

Сборщик метрик ошибок, который собирает:
- Общее количество ошибок
- Ошибки по типу (ValidationError, InternalError, NotFoundError и т.д.)
- Ошибки по HTTP статус коду (400, 404, 500 и т.д.)
- Ошибки по эндпоинту
- Временные метрики (последний час, по минутам)
- Последние N ошибок с деталями

### 2. Интеграция в HandleHTTPError (`server/middleware/errors.go`)

Метрики автоматически собираются при обработке ошибок через `HandleHTTPError`:
- Определяется тип ошибки (AppError или обычная error)
- Записывается в метрики с информацией об эндпоинте и request_id
- Сохраняется в последние ошибки для детального анализа

### 3. ErrorMetricsHandler (`server/handlers/error_metrics_handler.go`)

HTTP обработчик для получения метрик ошибок:
- `GET /api/errors/metrics` - все метрики
- `GET /api/errors/by-type` - ошибки по типу
- `GET /api/errors/by-code` - ошибки по HTTP коду
- `GET /api/errors/by-endpoint` - ошибки по эндпоинту
- `GET /api/errors/last?limit=50` - последние ошибки
- `POST /api/errors/reset` - сброс метрик

## Использование

### Инициализация

Метрики автоматически инициализируются при создании сервера в `NewServerWithConfig`:
```go
middleware.InitErrorMetrics()
```

### Автоматический сбор

Метрики собираются автоматически при обработке ошибок через `HandleHTTPError`:
```go
middleware.HandleHTTPError(w, r, err)
```

### Получение метрик

```bash
# Все метрики
curl http://localhost:9999/api/errors/metrics

# Ошибки по типу
curl http://localhost:9999/api/errors/by-type

# Ошибки по HTTP коду
curl http://localhost:9999/api/errors/by-code

# Ошибки по эндпоинту
curl http://localhost:9999/api/errors/by-endpoint

# Последние 50 ошибок
curl http://localhost:9999/api/errors/last?limit=50
```

## Структура данных

### GetMetrics() возвращает:
```json
{
  "total_errors": 1234,
  "errors_by_type": {
    "ValidationError": 100,
    "InternalError": 500,
    "NotFoundError": 200,
    ...
  },
  "errors_by_code": {
    400: 100,
    404: 200,
    500: 500,
    ...
  },
  "errors_by_endpoint": {
    "/api/databases/list": 50,
    "/api/quality/report": 30,
    ...
  },
  "time_buckets": [
    {
      "time": "2024-01-01T12:00:00Z",
      "count": 10,
      "by_type": {...},
      "by_code": {...}
    },
    ...
  ],
  "last_errors": [
    {
      "timestamp": "2024-01-01T12:00:00Z",
      "type": "InternalError",
      "code": 500,
      "message": "...",
      "endpoint": "/api/...",
      "request_id": "...",
      "user_message": "..."
    },
    ...
  ],
  "uptime_seconds": 3600,
  "errors_per_minute": 2.5
}
```

## Frontend дашборд

✅ Реализован в `frontend/app/errors/page.tsx`:
- Визуализация метрик через графики (BarChart, PieChart)
- Таблицы с детальной информацией
- Автоматическое обновление каждые 5 секунд
- Фильтрация по типам, кодам и эндпоинтам
- Просмотр последних ошибок с деталями

## Следующие шаги (опционально)

1. ✅ Создать frontend дашборд для визуализации метрик - **ЗАВЕРШЕНО**
2. Добавить экспорт метрик в Prometheus/Grafana
3. Добавить алерты при превышении порогов ошибок
4. Добавить фильтрацию и поиск по последним ошибкам
5. Добавить экспорт метрик в CSV/JSON

