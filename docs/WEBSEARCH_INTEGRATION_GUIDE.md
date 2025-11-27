# Руководство по интеграции веб-поиска с валидацией

## Обзор

Веб-поиск может быть интегрирован с `ValidationEngine` для автоматической валидации товаров при нормализации данных.

## Автоматическая интеграция

### Вариант 1: Интеграция через контейнер (рекомендуется)

При инициализации контейнера веб-поиск автоматически настраивается, если включен в конфигурации. Для использования валидаторов в ValidationEngine:

```go
// В internal/container/container.go после инициализации веб-поиска:

func (c *Container) setupValidationWithWebSearch() error {
    if c.WebSearchClient == nil {
        return nil // Веб-поиск не включен
    }

    // Создаем валидаторы
    client, ok := c.WebSearchClient.(*websearch.Client)
    if !ok {
        return nil
    }

    existenceValidator := websearch.NewProductExistenceValidator(client)
    accuracyValidator := websearch.NewProductAccuracyValidator(client)

    // Загружаем конфигурацию правил из БД
    rulesConfig, err := normalization.LoadWebSearchRulesConfig(c.ServiceDB)
    if err != nil {
        log.Printf("Warning: failed to load websearch rules config: %v", err)
        rulesConfig = make(map[string]interface{})
    }

    // Сохраняем валидаторы в контейнере для использования
    c.WebSearchExistenceValidator = existenceValidator
    c.WebSearchAccuracyValidator = accuracyValidator
    c.WebSearchRulesConfig = rulesConfig

    return nil
}
```

### Вариант 2: Ручная интеграция

При создании `ValidationEngine` вручную:

```go
import (
    "httpserver/normalization"
    "httpserver/websearch"
    "httpserver/database"
)

// Создаем ValidationEngine
validationEngine := normalization.NewValidationEngine()

// Создаем валидаторы веб-поиска
searchClient := websearch.NewClient(websearch.ClientConfig{
    BaseURL:   "https://api.duckduckgo.com",
    Timeout:   5 * time.Second,
    RateLimit: rate.Every(time.Second),
})

existenceValidator := websearch.NewProductExistenceValidator(searchClient)
accuracyValidator := websearch.NewProductAccuracyValidator(searchClient)

// Загружаем конфигурацию правил
rulesConfig, err := normalization.LoadWebSearchRulesConfig(serviceDB)
if err != nil {
    // Используем пустую конфигурацию
    rulesConfig = make(map[string]interface{})
}

// Устанавливаем валидаторы
validationEngine.SetWebSearchValidators(
    existenceValidator,
    accuracyValidator,
    rulesConfig,
)
```

## Конфигурация правил

Правила веб-поиска настраиваются в таблице `normalization_config` в поле `websearch_rules` (JSON):

```json
{
  "product_name": {
    "validator": "existence",
    "enabled": true
  },
  "product_code": {
    "validator": "accuracy",
    "enabled": true
  }
}
```

### Пример SQL для настройки:

```sql
UPDATE normalization_config 
SET websearch_rules = '{
  "product_name": {
    "validator": "existence",
    "enabled": true
  },
  "product_code": {
    "validator": "accuracy",
    "enabled": true
  }
}'
WHERE id = 1;
```

## Использование

После настройки, правила веб-поиска будут автоматически применяться при валидации:

```go
// Валидация элемента
isValid := validationEngine.ValidateItem(catalogItem)

// Получение отчета с результатами веб-поиска
report := validationEngine.GenerateReport()

// Получение ошибок и предупреждений
errors := validationEngine.GetErrors()
warnings := validationEngine.GetWarnings()
```

## Правила валидации

### `web_search_product_exists`

- **Severity:** Medium
- **Описание:** Проверяет существование товара в интернете по названию
- **Действие:** Добавляет предупреждение, если товар не найден
- **Не блокирует:** Ошибки веб-поиска не блокируют валидацию

### `web_search_data_accuracy`

- **Severity:** Low  
- **Описание:** Проверяет точность данных (название + код) через веб-поиск
- **Действие:** Добавляет предупреждение при низкой точности (score < 0.5)
- **Не блокирует:** Ошибки веб-поиска не блокируют валидацию

## API Endpoints

Веб-поиск также доступен через REST API:

### Проверка существования товара

```bash
POST /api/validation/websearch
{
  "name": "iPhone 15",
  "code": "A2847",
  "type": "existence"
}
```

### Проверка точности данных

```bash
POST /api/validation/websearch
{
  "name": "iPhone 15",
  "code": "A2847",
  "type": "accuracy"
}
```

### Прямой поиск

```bash
GET /api/validation/websearch/search?query=iPhone+15
```

## Ограничения

1. **Rate Limiting:** По умолчанию 1 запрос/сек (настраивается)
2. **Таймаут:** 5-10 секунд на запрос
3. **Кэширование:** Результаты кэшируются на 24 часа
4. **Неблокирующие:** Ошибки веб-поиска не прерывают валидацию

## Отладка

Для включения логирования:

```go
// Включить детальное логирование веб-поиска
os.Setenv("WEB_SEARCH_DEBUG", "true")
```

## Пример полной интеграции

```go
package main

import (
    "log"
    "time"

    "golang.org/x/time/rate"
    "httpserver/internal/container"
    "httpserver/normalization"
    "httpserver/websearch"
)

func setupValidationWithWebSearch(c *container.Container) error {
    if c.WebSearchClient == nil {
        log.Println("Web search is not enabled")
        return nil
    }

    // Получаем клиент
    client, ok := c.WebSearchClient.(*websearch.Client)
    if !ok {
        return nil
    }

    // Создаем валидаторы
    existenceValidator := websearch.NewProductExistenceValidator(client)
    accuracyValidator := websearch.NewProductAccuracyValidator(client)

    // Загружаем конфигурацию
    rulesConfig, err := normalization.LoadWebSearchRulesConfig(c.ServiceDB)
    if err != nil {
        log.Printf("Warning: failed to load rules: %v", err)
        rulesConfig = make(map[string]interface{})
    }

    // Сохраняем в контейнере
    c.WebSearchExistenceValidator = existenceValidator
    c.WebSearchAccuracyValidator = accuracyValidator

    log.Println("Web search validators configured successfully")
    return nil
}
```

## Следующие шаги

1. Добавьте вызов `setupValidationWithWebSearch()` после инициализации контейнера
2. Настройте правила в БД через SQL или API
3. Протестируйте валидацию с реальными данными

