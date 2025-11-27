# Экспорт всех контрагентов клиента

## Статус: ✅ ЗАВЕРШЕНО

## Обзор

Добавлен endpoint для экспорта всех контрагентов клиента в CSV или JSON форматах с поддержкой всех фильтров и параметров сортировки.

## Endpoint

```
GET /api/counterparties/all/export
```

## Параметры запроса

Все параметры из основного endpoint `/api/counterparties/all` поддерживаются, плюс:

| Параметр | Тип | Обязательный | Описание |
|----------|-----|--------------|----------|
| `client_id` | integer | Да | ID клиента |
| `project_id` | integer | Нет | ID проекта (для фильтрации) |
| `search` | string | Нет | Поисковый запрос |
| `source` | string | Нет | Фильтр: "database", "normalized" или пусто |
| `sort_by` | string | Нет | Поле сортировки: "name", "quality", "source", "id" |
| `order` | string | Нет | Порядок: "asc", "desc" |
| `format` | string | Нет | Формат экспорта: "csv" или "json" (по умолчанию: определяется из Accept header или JSON) |

## Определение формата

Формат экспорта определяется в следующем порядке:
1. Параметр `format` в query string (если указан)
2. HTTP заголовок `Accept` (если содержит "text/csv" или "application/csv" → CSV, иначе JSON)
3. По умолчанию: JSON

## Примеры использования

### Экспорт в CSV

```bash
# Через параметр format
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&format=csv" \
  -o counterparties.csv

# Через Accept header
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1" \
  -H "Accept: text/csv" \
  -o counterparties.csv
```

### Экспорт в JSON

```bash
# Через параметр format
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&format=json" \
  -o counterparties.json

# По умолчанию (без указания формата)
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1" \
  -o counterparties.json
```

### Экспорт с фильтрами

```bash
# Только нормализованные контрагенты
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&source=normalized&format=csv" \
  -o normalized.csv

# Конкретный проект
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&project_id=1&format=csv" \
  -o project1.csv

# С поиском
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&search=ООО&format=csv" \
  -o search_results.csv

# С сортировкой по качеству
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&sort_by=quality&order=desc&format=csv" \
  -o sorted_by_quality.csv
```

## Формат CSV

CSV файл содержит следующие колонки (в порядке):

1. **ID** - Уникальный идентификатор контрагента
2. **Name** - Наименование контрагента
3. **Source** - Источник данных ("database" или "normalized")
4. **Project ID** - ID проекта
5. **Project Name** - Название проекта
6. **Database ID** - ID базы данных (если источник = "database")
7. **Database Name** - Название базы данных (если источник = "database")
8. **Tax ID (INN)** - ИНН
9. **KPP** - КПП
10. **BIN** - БИН
11. **Legal Address** - Юридический адрес
12. **Postal Address** - Почтовый адрес
13. **Contact Phone** - Контактный телефон
14. **Contact Email** - Контактный email
15. **Contact Person** - Контактное лицо
16. **Quality Score** - Оценка качества (если доступна)
17. **Reference** - Референс (если источник = "database")
18. **Code** - Код (если источник = "database")
19. **Normalized Name** - Нормализованное наименование (если источник = "normalized")
20. **Source Name** - Исходное наименование (если источник = "normalized")
21. **Source Reference** - Исходный референс (если источник = "normalized")

### Пример CSV файла

```csv
ID,Name,Source,Project ID,Project Name,Database ID,Database Name,Tax ID (INN),KPP,BIN,Legal Address,Postal Address,Contact Phone,Contact Email,Contact Person,Quality Score,Reference,Code,Normalized Name,Source Name,Source Reference
1,ООО Пример,database,1,Проект 1,1,База данных 1,1234567890,123456789,,г. Москва, ул. Примерная, д. 1,г. Москва, ул. Примерная, д. 1,+7 (495) 123-45-67,info@example.com,Иванов И.И.,,xxx-xxx-xxx,001,ООО Пример,ООО ПРИМЕР,xxx-xxx-xxx
2,ООО Пример,normalized,1,Проект 1,,,,1234567890,123456789,,г. Москва, ул. Примерная, д. 1,г. Москва, ул. Примерная, д. 1,+7 (495) 123-45-67,info@example.com,Иванов И.И.,0.95,,,ООО Пример,ООО ПРИМЕР,xxx-xxx-xxx
```

## Формат JSON

JSON файл содержит полную структуру данных:

```json
{
  "client_id": 1,
  "project_id": null,
  "export_date": "2024-01-15T10:30:00Z",
  "format_version": "1.0",
  "total": 150,
  "stats": {
    "total_from_database": 100,
    "total_normalized": 50,
    "total_with_quality": 45,
    "average_quality": 0.87,
    "databases_processed": 5,
    "projects_processed": 3,
    "processing_time_ms": 1250
  },
  "counterparties": [
    {
      "id": 1,
      "name": "ООО Пример",
      "source": "database",
      "project_id": 1,
      "project_name": "Проект 1",
      "database_id": 1,
      "database_name": "База данных 1",
      "tax_id": "1234567890",
      "kpp": "123456789",
      "bin": "",
      "legal_address": "г. Москва, ул. Примерная, д. 1",
      "postal_address": "г. Москва, ул. Примерная, д. 1",
      "contact_phone": "+7 (495) 123-45-67",
      "contact_email": "info@example.com",
      "contact_person": "Иванов И.И.",
      "quality_score": null,
      "reference": "xxx-xxx-xxx",
      "code": "001"
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
      "quality_score": 0.95
    }
  ],
  "projects": [
    {
      "id": 1,
      "name": "Проект 1"
    }
  ]
}
```

## Именование файлов

Файлы экспорта автоматически именуются с timestamp:

- CSV: `counterparties_client_{client_id}_{timestamp}.csv`
- JSON: `counterparties_client_{client_id}_{timestamp}.json`

Если указан `project_id`:
- CSV: `counterparties_client_{client_id}_project_{project_id}_{timestamp}.csv`
- JSON: `counterparties_client_{client_id}_project_{project_id}_{timestamp}.json`

Пример: `counterparties_client_1_20240115_103000.csv`

## Особенности

1. **Без пагинации**: Экспорт возвращает ВСЕ контрагенты, соответствующие фильтрам (без ограничения по limit)

2. **Поддержка всех фильтров**: Все фильтры из основного endpoint работают:
   - Поиск (`search`)
   - Фильтр по источнику (`source`)
   - Фильтр по проекту (`project_id`)
   - Сортировка (`sort_by`, `order`)

3. **Валидация**: Все параметры валидируются так же, как в основном endpoint

4. **Логирование**: Все запросы на экспорт логируются с полной информацией

5. **Обработка ошибок**: Детальные сообщения об ошибках при проблемах с экспортом

## Ограничения

- Максимальное количество записей для экспорта: 1,000,000 (для предотвращения перегрузки)
- Для очень больших объемов данных рекомендуется использовать фильтры

## Использование в браузере

Экспорт можно использовать напрямую в браузере:

```html
<a href="/api/counterparties/all/export?client_id=1&format=csv" download>
  Скачать CSV
</a>

<a href="/api/counterparties/all/export?client_id=1&format=json" download>
  Скачать JSON
</a>
```

## Интеграция с другими системами

Экспорт можно использовать для интеграции с другими системами:

```bash
# Автоматический экспорт в CSV для импорта в Excel
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&format=csv" \
  -o /path/to/export/counterparties.csv

# Экспорт в JSON для обработки скриптами
curl -X GET "http://localhost:9999/api/counterparties/all/export?client_id=1&format=json" \
  | jq '.counterparties[] | select(.source == "normalized")'
```

## Проверка

✅ Код компилируется успешно  
✅ Линтер не выявил ошибок  
✅ Поддержка CSV и JSON форматов  
✅ Автоматическое определение формата из Accept header  
✅ Поддержка всех фильтров и параметров сортировки  
✅ Именованные файлы с timestamp  
✅ Детальное логирование  
✅ Обработка ошибок  

## Готово к использованию

Endpoint для экспорта полностью реализован и готов к использованию в продакшене.

