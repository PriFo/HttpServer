# Быстрый старт: Веб-поиск DuckDuckGo

## Настройка

### 1. Переменные окружения

```bash
# Включить веб-поиск
export WEB_SEARCH_ENABLED=true

# Таймаут запросов (по умолчанию 5 секунд)
export WEB_SEARCH_TIMEOUT=10s

# Время жизни кэша (по умолчанию 24 часа)
export WEB_SEARCH_CACHE_TTL=24h

# Включить кэширование
export WEB_SEARCH_CACHE_ENABLED=true

# Лимит запросов в секунду (по умолчанию 1)
export WEB_SEARCH_RATE_LIMIT_PER_SEC=1

# Базовый URL API (по умолчанию DuckDuckGo)
export WEB_SEARCH_BASE_URL=https://api.duckduckgo.com
```

### 2. Запуск сервера

Сервер автоматически инициализирует модуль веб-поиска, если `WEB_SEARCH_ENABLED=true`.

## Использование API

### 1. Простой поиск

```bash
GET /api/validation/web-search?query=ГОСТ 5264-80
```

**Ответ:**
```json
{
  "success": true,
  "found": true,
  "results": [
    {
      "title": "ГОСТ Р 5264-80 Ручная дуговая сварка...",
      "url": "https://docs.cntd.ru/document/...",
      "snippet": "...",
      "relevance": 0.5
    }
  ],
  "query": "ГОСТ 5264-80",
  "source": "duckduckgo-html"
}
```

### 2. Валидация товара

```bash
POST /api/validation/websearch
Content-Type: application/json

{
  "name": "ГОСТ 5264-80",
  "code": "5264-80",
  "type": "existence"
}
```

**Ответ:**
```json
{
  "success": true,
  "validation": {
    "status": "success",
    "found": true,
    "score": 0.8,
    "message": "Найдено 92 релевантных результатов",
    "provider": "duckduckgo-html",
    "results": [...]
  }
}
```

### 3. Батч-валидация

```bash
POST /api/validation/web-search/batch
Content-Type: application/json

{
  "items": [
    {
      "id": "1",
      "name": "ГОСТ 5264-80",
      "code": "5264-80"
    },
    {
      "id": "2",
      "name": "ГОСТ 50.04.03-2018",
      "code": "50.04.03-2018"
    }
  ]
}
```

## Примеры использования в коде

### Использование клиента напрямую

```go
import (
    "context"
    "time"
    "httpserver/websearch"
    "golang.org/x/time/rate"
)

// Создание клиента
cacheConfig := &websearch.CacheConfig{
    Enabled:         true,
    TTL:             24 * time.Hour,
    CleanupInterval: 6 * time.Hour,
    MaxSize:         1000,
}
cache := websearch.NewCache(cacheConfig)

clientConfig := websearch.ClientConfig{
    BaseURL:   "https://api.duckduckgo.com",
    Timeout:   10 * time.Second,
    RateLimit: rate.Every(time.Second),
    Cache:     cache,
}
client := websearch.NewClient(clientConfig)

// Выполнение поиска
ctx := context.Background()
result, err := client.Search(ctx, "ГОСТ 5264-80")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Найдено результатов: %d\n", len(result.Results))
```

### Использование валидаторов

```go
// Создание валидаторов
existenceValidator := websearch.NewProductExistenceValidator(client)
accuracyValidator := websearch.NewProductAccuracyValidator(client)

// Проверка существования
validation, err := existenceValidator.Validate(ctx, "ГОСТ 5264-80")
if err != nil {
    log.Fatal(err)
}

if validation.Found {
    fmt.Printf("Товар найден! Оценка: %.2f\n", validation.Score)
}

// Проверка точности
accuracy, err := accuracyValidator.Validate(ctx, "ГОСТ 5264-80", "Ручная дуговая сварка")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Точность данных: %.2f\n", accuracy.Score)
```

## Интеграция с ValidationEngine

```go
import "httpserver/normalization"

// Создание валидаторов
existenceValidator := websearch.NewProductExistenceValidator(client)
accuracyValidator := websearch.NewProductAccuracyValidator(client)

// Загрузка конфигурации правил
rulesConfig, err := normalization.LoadWebSearchRulesConfig(db)
if err != nil {
    log.Fatal(err)
}

// Добавление правил в ValidationEngine
validationEngine.SetWebSearchValidators(
    existenceValidator,
    accuracyValidator,
    rulesConfig,
)
```

## Мониторинг

### Статистика кэша

```bash
GET /api/admin/websearch/stats
```

**Ответ:**
```json
{
  "success": true,
  "providers": {
    "duckduckgo": {
      "enabled": true,
      "type": "duckduckgo"
    }
  },
  "cache": {
    "hits": 15,
    "misses": 85,
    "size": 100
  },
  "timestamp": "2025-11-23T18:00:00Z"
}
```

## Обработка ошибок

Модуль обрабатывает следующие ошибки:
- Таймауты запросов
- Превышение rate limit
- Ошибки парсинга HTML
- Ошибки сети

Все ошибки логируются и возвращаются в структурированном виде.

## Производительность

- **Instant Answer API:** ~100 мс
- **HTML-поиск:** ~1-2 сек
- **Кэш hit rate:** ~15-20%
- **Параллельные запросы:** ограничены rate limiter

## Ограничения

1. Rate limit: 1 запрос/секунду по умолчанию
2. Таймаут: 10 секунд по умолчанию
3. Размер батча: максимум 10 элементов
4. Размер кэша: 1000 записей по умолчанию

## Поддержка

При возникновении проблем:
1. Проверьте логи сервера
2. Проверьте переменные окружения
3. Проверьте доступность DuckDuckGo API
4. Проверьте статистику кэша через API

