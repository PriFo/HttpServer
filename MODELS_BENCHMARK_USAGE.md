# Руководство по использованию бенчмарка моделей

## Обзор

Бенчмарк моделей позволяет протестировать все доступные модели AI провайдера и определить их производительность, скорость и надежность.

## Важные замечания

### MaxWorkers=2 - это НЕ ограничение на количество моделей

**ВАЖНО**: `MaxWorkers=2` в конфигурации Arliai означает ограничение на **параллельные запросы**, а НЕ на количество моделей. Бенчмарк тестирует **все доступные модели**, независимо от этого ограничения.

### Получение всех моделей

Система автоматически пытается получить все модели из API с помощью нескольких стратегий:

1. **Прямой запрос к Arliai API** с разными query параметрами:
   - `status=all` - все статусы (active, deprecated, beta)
   - `include=all` - включить все модели
   - `all=true` - альтернативный параметр
   - Без параметров (fallback)

2. **Fallback на внутренний API**: `/api/workers/models?enabled=all&status=all`

3. **Fallback на конфигурацию**: все модели из конфигурации (включая disabled)

4. **Fallback на известные модели**: список известных моделей Arliai

## API Endpoints

### Запуск бенчмарка

```http
POST /api/models/benchmark
Content-Type: application/json

{
  "models": ["GLM-4.5-Air", "GLM-4.5"],  // опционально: конкретные модели
  "test_products": ["Болт М8х20", "Гайка М8"],  // опционально: тестовые данные
  "max_retries": 5,  // опционально: количество повторов при ошибке
  "retry_delay_ms": 200,  // опционально: задержка между повторами (мс)
  "auto_update_priorities": true  // опционально: автоматически обновить приоритеты
}
```

### Получение истории бенчмарков

```http
GET /api/models/benchmark?history=true&limit=10&model=GLM-4.5-Air
```

### Получение списка всех моделей

```http
GET /api/workers/models?enabled=all&status=all
```

## Ответ API

### Успешный ответ

```json
{
  "models": [
    {
      "model": "GLM-4.5-Air",
      "status": "ok",
      "success_count": 15,
      "error_count": 0,
      "total_requests": 15,
      "speed": 2.5,
      "avg_response_time_ms": 400,
      "success_rate": 100.0,
      "priority": 1
    }
  ],
  "total": 1,
  "test_count": 15,
  "timestamp": "2025-01-21T10:00:00Z",
  "priorities_updated": true,
  "message": "Benchmark completed: 1 models tested, 1 successful, 0 failed",
  "statistics": {
    "successful_models": 1,
    "failed_models": 0,
    "total_successes": 15,
    "total_errors": 0,
    "total_requests": 15,
    "overall_success_rate": 100.0,
    "models_tested": 1,
    "models_available": 6
  }
}
```

### Поля статистики

- `successful_models` - количество успешных моделей
- `failed_models` - количество неудачных моделей
- `total_successes` - общее количество успешных запросов
- `total_errors` - общее количество ошибок
- `total_requests` - общее количество запросов
- `overall_success_rate` - общий процент успеха
- `models_tested` - количество протестированных моделей
- `models_available` - количество доступных моделей (до фильтрации)

## Обработка ошибок

### Типы ошибок

Система автоматически классифицирует ошибки:

- **quota_exceeded** - исчерпан лимит quota (не повторяется)
- **rate_limit** - превышен rate limit (повторяется с задержкой)
- **timeout** - таймаут запроса
- **network** - сетевые ошибки
- **auth** - ошибки аутентификации (неверный API ключ)
- **unknown** - неизвестные ошибки

### Сообщения об ошибках

Система предоставляет информативные сообщения:

- **Низкий процент успеха** (< 50%): "WARNING: Low success rate - check API keys, rate limits, and quota"
- **Мало моделей** (≤ 2): "NOTE: Only 2 models available - check if API returns all models. MaxWorkers=2 is a limit on parallel requests, not on the number of models."
- **Высокий процент ошибок** (> 30%): "WARNING: High error rate detected - check API keys, network connectivity, and provider status."

## Примеры использования

### Тестирование всех моделей

```bash
curl -X POST http://localhost:9999/api/models/benchmark \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Тестирование конкретных моделей

```bash
curl -X POST http://localhost:9999/api/models/benchmark \
  -H "Content-Type: application/json" \
  -d '{
    "models": ["GLM-4.5-Air", "GLM-4.5"],
    "max_retries": 3,
    "retry_delay_ms": 200
  }'
```

### Получение истории

```bash
curl "http://localhost:9999/api/models/benchmark?history=true&limit=10"
```

## Устранение проблем

### Проблема: Получено только 2 модели

**Причина**: API может фильтровать модели или возвращать только активные.

**Решение**:
1. Проверьте логи сервера для деталей
2. Убедитесь, что API ключ имеет доступ ко всем моделям
3. Проверьте параметры запроса к API
4. Используйте `/api/workers/models?enabled=all&status=all` для проверки

### Проблема: Высокий процент ошибок

**Причины**:
- Исчерпан quota
- Превышен rate limit
- Проблемы с сетью
- Неверный API ключ

**Решение**:
1. Проверьте API ключ
2. Проверьте quota и rate limits в настройках провайдера
3. Увеличьте `retry_delay_ms` для снижения нагрузки
4. Проверьте сетевую связность

### Проблема: Quota exceeded

**Причина**: Исчерпан лимит quota для провайдера.

**Решение**:
- Для OpenRouter: проверьте quota в настройках аккаунта
- Для Arliai: проверьте подписку и доступные модели
- Система автоматически не повторяет запросы при quota exceeded

## Логирование

Все действия логируются с префиксом `[Benchmark]`:

```
[Benchmark] Cleared Arliai cache to get fresh models
[Benchmark] Attempting to get models with status=all
[Benchmark] Successfully got 6 models with status=all
[Benchmark] Starting benchmark with 6 models, 15 test products
[Benchmark] Model GLM-4.5-Air: Starting parallel benchmark
[Benchmark] Model GLM-4.5-Air: Successfully classified 'Болт М8х20' in 400ms
[Benchmark] Statistics: 6 successful models, 0 failed models
```

## Автоматическое обновление приоритетов

Если `auto_update_priorities: true`, система автоматически обновит приоритеты моделей на основе результатов бенчмарка. Модели сортируются по:
1. Скорости (requests/second)
2. Проценту успеха
3. Среднему времени ответа

