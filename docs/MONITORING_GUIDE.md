# Руководство по мониторингу

**Версия:** 1.0  
**Дата:** 2025-11-23

---

## 📋 Содержание

1. [Обзор](#обзор)
2. [Health Checks](#health-checks)
3. [Метрики](#метрики)
4. [Логирование](#логирование)
5. [Алерты](#алерты)
6. [Дашборды](#дашборды)

---

## Обзор

Система мониторинга включает:
- **Health Checks** - проверка здоровья компонентов
- **Метрики** - сбор производительности и использования ресурсов
- **Логирование** - структурированные логи
- **Алерты** - уведомления о проблемах

---

## Health Checks

### Endpoints

#### `/health` - Полная проверка здоровья
```bash
curl http://localhost:9999/health
```

**Ответ:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-23T12:00:00Z",
  "uptime": "2h30m15s",
  "version": "1.0.0",
  "components": {
    "database": {
      "name": "database",
      "status": "healthy",
      "message": "Database is healthy",
      "timestamp": "2025-11-23T12:00:00Z",
      "latency": "5ms"
    },
    "service_database": {
      "name": "service_database",
      "status": "healthy",
      "message": "Service database is healthy",
      "timestamp": "2025-11-23T12:00:00Z",
      "latency": "3ms"
    }
  },
  "system": {
    "cpu_usage_percent": 25.5,
    "memory_usage_percent": 45.2,
    "goroutines": 42
  }
}
```

**Статусы:**
- `healthy` - все компоненты работают нормально
- `degraded` - некоторые компоненты работают с ограничениями
- `unhealthy` - критические компоненты недоступны

#### `/health/live` - Liveness probe (Kubernetes)
```bash
curl http://localhost:9999/health/live
```

**Ответ:** `OK` (200) или ошибка (503)

#### `/health/ready` - Readiness probe (Kubernetes)
```bash
curl http://localhost:9999/health/ready
```

**Ответ:** `Ready` (200) или `Not Ready` (503)

---

## Метрики

### Endpoints

#### `/api/monitoring/metrics` - Общие метрики
```bash
curl http://localhost:9999/api/monitoring/metrics
```

**Ответ:**
```json
{
  "http": {
    "requests_total": 12345,
    "requests_success": 12200,
    "requests_error": 145,
    "success_rate": 98.8,
    "avg_duration_ms": 125,
    "requests_per_second": 2.5
  },
  "database": {
    "queries_total": 45678,
    "avg_duration_ms": 15,
    "connections_active": 5,
    "connections_idle": 20
  },
  "system": {
    "uptime_seconds": 3600,
    "start_time": "2025-11-23T10:00:00Z"
  }
}
```

#### `/api/monitoring/providers` - Метрики провайдеров
```bash
curl http://localhost:9999/api/monitoring/providers
```

#### `/api/errors/metrics` - Метрики ошибок
```bash
curl http://localhost:9999/api/errors/metrics
```

**Ответ:**
```json
{
  "total_errors": 145,
  "errors_per_minute": 0.5,
  "errors_by_type": {
    "ValidationError": 50,
    "InternalError": 80,
    "NotFoundError": 15
  },
  "errors_by_code": {
    "400": 50,
    "500": 80,
    "404": 15
  },
  "errors_by_endpoint": {
    "/api/normalization/start": 30,
    "/api/quality/issues": 20
  }
}
```

---

## Логирование

### Формат логов

Логи структурированы и включают:
- **Timestamp** - время события
- **Level** - уровень (INFO, WARN, ERROR, DEBUG)
- **Message** - сообщение
- **Context** - дополнительный контекст (endpoint, request_id и т.д.)

### Уровни логирования

- **DEBUG** - детальная информация для отладки
- **INFO** - общая информация о работе системы
- **WARN** - предупреждения о потенциальных проблемах
- **ERROR** - ошибки, требующие внимания

### Настройка уровня логирования

Через переменную окружения:
```bash
LOG_LEVEL=debug  # debug, info, warn, error
```

### Примеры логов

```json
{
  "timestamp": "2025-11-23T12:00:00Z",
  "level": "INFO",
  "message": "Request processed",
  "endpoint": "/api/normalization/start",
  "request_id": "abc123",
  "duration_ms": 125
}
```

---

## Алерты

### Настройка алертов

Алерты настраиваются через конфигурацию мониторинга:

```yaml
alerts:
  - name: high_error_rate
    condition: error_rate > 1%
    severity: critical
    action: notify_team
    
  - name: high_response_time
    condition: avg_response_time > 1s
    severity: warning
    action: log
    
  - name: database_unavailable
    condition: database_status == "unhealthy"
    severity: critical
    action: notify_team_and_escalate
```

### Типы алертов

1. **Critical** - требует немедленного внимания
   - База данных недоступна
   - Error rate > 5%
   - Response time > 5s

2. **Warning** - требует мониторинга
   - Error rate > 1%
   - Response time > 1s
   - Memory usage > 80%

3. **Info** - информационные
   - Высокий трафик
   - Завершение длительных операций

---

## Дашборды

### Рекомендуемые метрики для дашборда

1. **Общие метрики**
   - Uptime
   - Requests per second
   - Success rate
   - Average response time

2. **Метрики по компонентам**
   - Database queries per second
   - Database connection pool usage
   - Cache hit rate

3. **Метрики ошибок**
   - Error rate
   - Errors by type
   - Errors by endpoint

4. **Системные метрики**
   - CPU usage
   - Memory usage
   - Goroutines count

### Интеграция с Prometheus

Для интеграции с Prometheus добавьте endpoint `/metrics`:

```go
// Пример экспорта метрик в формате Prometheus
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
    // Экспорт метрик в формате Prometheus
}
```

---

## Best Practices

1. **Мониторинг критических компонентов**
   - База данных
   - Внешние API
   - Критические сервисы

2. **Настройка алертов**
   - Не слишком много алертов (alert fatigue)
   - Настройка правильных порогов
   - Эскалация для критических проблем

3. **Логирование**
   - Структурированные логи
   - Не логировать чувствительные данные
   - Ротация логов

4. **Метрики**
   - Сбор только необходимых метрик
   - Агрегация для снижения нагрузки
   - Хранение метрик с правильным retention

---

## Troubleshooting

### Проблемы с health checks

```bash
# Проверьте логи
docker-compose logs backend | grep health

# Проверьте подключение к БД
sqlite3 data/service.db "SELECT 1;"
```

### Проблемы с метриками

```bash
# Проверьте endpoint метрик
curl http://localhost:9999/api/monitoring/metrics

# Проверьте логи
docker-compose logs backend | grep metrics
```

---

*Последнее обновление: 2025-11-23*


