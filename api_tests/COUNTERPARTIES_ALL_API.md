# API: Получение всех контрагентов клиента

## Описание

Endpoint для получения всех контрагентов клиента из всех баз данных по всем проектам, включая как исходные (ненормализованные) контрагенты из баз данных, так и нормализованные из таблицы `normalized_counterparties`.

## Endpoint

```
GET /api/counterparties/all
```

## Параметры запроса

| Параметр | Тип | Обязательный | Описание |
|----------|-----|--------------|----------|
| `client_id` | integer | Да | ID клиента |
| `project_id` | integer | Нет | ID проекта (для фильтрации по конкретному проекту) |
| `search` | string | Нет | Поисковый запрос (поиск по имени, ИНН, БИН) |
| `source` | string | Нет | Фильтр по источнику: `"database"` (только из баз), `"normalized"` (только нормализованные) или пусто (все) |
| `sort_by` | string | Нет | Поле для сортировки: `"name"`, `"quality"`, `"source"`, `"id"` или пусто (по умолчанию: качество -> имя -> ID) |
| `order` | string | Нет | Порядок сортировки: `"asc"`, `"desc"` или пусто (по умолчанию: asc) |
| `offset` | integer | Нет | Смещение для пагинации (по умолчанию: 0) |
| `limit` | integer | Нет | Количество записей (по умолчанию: 100, максимум: 1000) |
| `min_quality` | float | Нет | Минимальная оценка качества (0.0-1.0) |
| `max_quality` | float | Нет | Максимальная оценка качества (0.0-1.0) |

## Примеры запросов

### 1. Получение всех контрагентов клиента

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1"
```

### 2. С пагинацией

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&offset=0&limit=10"
```

### 3. Фильтр по источнику - только из баз данных

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&source=database"
```

### 4. Фильтр по источнику - только нормализованные

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&source=normalized"
```

### 5. Поиск по имени

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&search=ООО"
```

### 6. Фильтр по проекту

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&project_id=1"
```

### 7. Фильтр по качеству

```bash
# Контрагенты с качеством >= 0.8
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&min_quality=0.8"

# Контрагенты с качеством от 0.5 до 0.9
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&min_quality=0.5&max_quality=0.9"
```

## Формат ответа

```json
{
  "counterparties": [
    {
      "id": 1,
      "name": "ООО Пример",
      "source": "database",
      "project_id": 1,
      "project_name": "Проект 1",
      "database_id": 1,
      "database_name": "База данных 1",
      "reference": "xxx-xxx-xxx",
      "code": "001",
      "tax_id": "1234567890",
      "kpp": "123456789",
      "bin": "",
      "legal_address": "г. Москва, ул. Примерная, д. 1",
      "postal_address": "г. Москва, ул. Примерная, д. 1",
      "contact_phone": "+7 (495) 123-45-67",
      "contact_email": "info@example.com",
      "contact_person": "Иванов Иван Иванович",
      "quality_score": null
    },
    {
      "id": 2,
      "name": "ООО Пример",
      "source": "normalized",
      "project_id": 1,
      "project_name": "Проект 1",
      "normalized_name": "ООО Пример",
      "source_name": "ООО ПРИМЕР",
      "source_reference": "xxx-xxx-xxx",
      "tax_id": "1234567890",
      "kpp": "123456789",
      "bin": "",
      "legal_address": "г. Москва, ул. Примерная, д. 1",
      "postal_address": "г. Москва, ул. Примерная, д. 1",
      "contact_phone": "+7 (495) 123-45-67",
      "contact_email": "info@example.com",
      "contact_person": "Иванов Иван Иванович",
      "quality_score": 0.95
    }
  ],
  "projects": [
    {
      "id": 1,
      "name": "Проект 1"
    }
  ],
  "total": 2,
  "offset": 0,
  "limit": 100,
  "stats": {
    "total_from_database": 1,
    "total_normalized": 1,
    "total_with_quality": 1,
    "average_quality": 0.95,
    "databases_processed": 1,
    "projects_processed": 1,
    "processing_time_ms": 125
  }
}
```

## Экспорт данных

Для экспорта всех контрагентов в CSV или JSON формате используйте endpoint:

```
GET /api/counterparties/all/export
```

### Параметры экспорта

Все параметры из основного endpoint поддерживаются, плюс:

| Параметр | Тип | Обязательный | Описание |
|----------|-----|--------------|----------|
| `format` | string | Нет | Формат экспорта: `"csv"` или `"json"` (по умолчанию: определяется из Accept header или JSON) |

### Примеры экспорта

**Экспорт в CSV:**
```bash
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&format=csv" -o counterparties.csv
```

**Экспорт в JSON:**
```bash
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&format=json" -o counterparties.json
```

**Экспорт с фильтрами:**
```bash
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&source=normalized&format=csv" -o normalized.csv
```

**Использование Accept header:**
```bash
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1" \
  -H "Accept: text/csv" \
  -o counterparties.csv
```

Подробнее см. [COUNTERPARTIES_ALL_API_EXPORT.md](../COUNTERPARTIES_ALL_API_EXPORT.md)

## Поля ответа

### UnifiedCounterparty

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | integer | ID контрагента |
| `name` | string | Название контрагента |
| `source` | string | Источник данных: `"database"` или `"normalized"` |
| `project_id` | integer | ID проекта |
| `project_name` | string | Название проекта |
| `database_id` | integer (optional) | ID базы данных (только для source="database") |
| `database_name` | string (optional) | Название базы данных (только для source="database") |
| `reference` | string (optional) | Ссылка на запись (только для source="database") |
| `code` | string (optional) | Код записи (только для source="database") |
| `normalized_name` | string (optional) | Нормализованное название (только для source="normalized") |
| `source_name` | string (optional) | Исходное название (только для source="normalized") |
| `source_reference` | string (optional) | Исходная ссылка (только для source="normalized") |
| `tax_id` | string | ИНН |
| `kpp` | string | КПП |
| `bin` | string | БИН |
| `legal_address` | string | Юридический адрес |
| `postal_address` | string | Почтовый адрес |
| `contact_phone` | string | Телефон |
| `contact_email` | string | Email |
| `contact_person` | string | Контактное лицо |
| `quality_score` | float (optional) | Оценка качества данных (только для source="normalized") |

## Коды ответов

| Код | Описание |
|-----|----------|
| 200 | Успешный запрос |
| 400 | Ошибка валидации (например, отсутствует `client_id`) |
| 500 | Внутренняя ошибка сервера |

## Особенности

1. **Объединение данных**: Endpoint объединяет контрагентов из двух источников:
   - Исходные базы данных (через `GetCatalogItemsByUpload`)
   - Нормализованные записи (из таблицы `normalized_counterparties`)

2. **Извлечение данных**: Для контрагентов из исходных баз данных данные извлекаются из XML-атрибутов с помощью функций из пакета `extractors`.

3. **Сортировка**: Объединенный список контрагентов сортируется:
   - Сначала по наличию `quality_score` (нормализованные записи выше)
   - Затем по значению `quality_score` (выше качество - выше в списке)
   - Затем по имени (без учета регистра, алфавитный порядок)
   - В конце по ID (для стабильности сортировки)

4. **Пагинация**: Применяется после объединения и сортировки всех контрагентов, поэтому общее количество (`total`) включает все записи из обоих источников.

5. **Фильтрация по источнику**: Параметр `source` позволяет получить только контрагентов из определенного источника:
   - `source=database` - только из исходных баз данных
   - `source=normalized` - только нормализованные записи
   - Без параметра - все контрагенты

6. **Поиск**: Параметр `search` ищет по полям: `name`, `tax_id`, `bin`, `normalized_name`, `source_name`.

7. **Фильтр по качеству**: Параметры `min_quality` и `max_quality` позволяют фильтровать контрагентов по оценке качества:
   - `min_quality` - минимальная оценка качества (0.0-1.0)
   - `max_quality` - максимальная оценка качества (0.0-1.0)
   - Записи без оценки качества (`quality_score = null`) исключаются, если указан `min_quality`
   - Можно использовать оба параметра одновременно для диапазона

7. **Сортировка**: Параметры `sort_by` и `order` позволяют настроить сортировку результатов:
   - `sort_by=name` - сортировка по имени
   - `sort_by=quality` - сортировка по качеству (quality_score)
   - `sort_by=source` - сортировка по источнику
   - `sort_by=id` - сортировка по ID
   - Без параметра - сортировка по умолчанию (качество -> имя -> ID)
   - `order=asc` - по возрастанию
   - `order=desc` - по убыванию

8. **Статистика**: В ответе включена статистика по источникам данных:
   - `total_from_database` - количество контрагентов из исходных баз данных
   - `total_normalized` - количество нормализованных контрагентов
   - `total_with_quality` - количество контрагентов с оценкой качества
   - `average_quality` - средняя оценка качества (если есть записи с качеством)
   - `databases_processed` - количество успешно обработанных баз данных (заполняется автоматически)
   - `projects_processed` - количество обработанных проектов (заполняется автоматически)
   - `processing_time_ms` - время обработки запроса в миллисекундах

9. **Оптимизация производительности**: 
   - Параллельная обработка баз данных (до 5 одновременных подключений)
   - Использование goroutines для ускорения работы
   - Безопасная синхронизация при параллельном доступе
   - Метрики производительности в ответе для мониторинга

## Тестирование

Для тестирования используйте:

1. **PowerShell скрипт**: `.\test_counterparties_all_api.ps1 [client_id] [base_url]`
2. **HTML интерфейс**: Откройте `api_tests/test_counterparties_all_api.html` в браузере

## Примеры использования

### Получение всех контрагентов клиента с пагинацией

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&offset=0&limit=50"
```

### Поиск контрагентов по ИНН

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&search=1234567890"
```

### Получение только нормализованных контрагентов конкретного проекта

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&project_id=1&source=normalized"
```

### Сортировка по качеству (по убыванию)

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&sort_by=quality&order=desc"
```

### Сортировка по имени (по возрастанию)

```bash
curl -X GET "http://localhost:9999/api/counterparties/all?client_id=1&sort_by=name&order=asc"
```

