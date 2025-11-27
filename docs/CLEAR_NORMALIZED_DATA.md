# Очистка результатов нормализации

Документация по функционалу удаления результатов нормализации из базы данных.

## Обзор

Система предоставляет несколько способов удаления результатов нормализации:

1. **Удаление всех данных** - полная очистка базы данных
2. **Удаление по проекту** - удаление данных для конкретного проекта
3. **Удаление по сессии** - удаление данных для конкретной сессии нормализации

## API Endpoints

### DELETE /api/normalization/data/all

Удаляет все результаты нормализации из базы данных.

**Параметры:**
- `confirm` (query, обязательный) - должен быть `true` для подтверждения

**Пример запроса:**
```bash
curl -X DELETE "http://localhost:9999/api/normalization/data/all?confirm=true"
```

**Пример ответа:**
```json
{
  "success": true,
  "message": "Все результаты нормализации успешно удалены",
  "rows_affected": 13134,
  "count_before": 13134
}
```

**Коды ответа:**
- `200 OK` - успешное удаление
- `400 Bad Request` - отсутствует подтверждение
- `405 Method Not Allowed` - неверный HTTP метод
- `500 Internal Server Error` - ошибка при удалении

### DELETE /api/normalization/data/project

Удаляет результаты нормализации для указанного проекта.

**Параметры:**
- `project_id` (query, обязательный) - ID проекта
- `confirm` (query, обязательный) - должен быть `true` для подтверждения

**Пример запроса:**
```bash
curl -X DELETE "http://localhost:9999/api/normalization/data/project?project_id=1&confirm=true"
```

**Пример ответа:**
```json
{
  "success": true,
  "message": "Результаты нормализации для проекта 1 успешно удалены",
  "rows_affected": 5234,
  "count_before": 5234,
  "project_id": 1
}
```

**Коды ответа:**
- `200 OK` - успешное удаление (или нет данных для удаления)
- `400 Bad Request` - отсутствует подтверждение или неверный project_id
- `405 Method Not Allowed` - неверный HTTP метод
- `500 Internal Server Error` - ошибка при удалении

### DELETE /api/normalization/data/session

Удаляет результаты нормализации для указанной сессии.

**Параметры:**
- `session_id` (query, обязательный) - ID сессии нормализации
- `confirm` (query, обязательный) - должен быть `true` для подтверждения

**Пример запроса:**
```bash
curl -X DELETE "http://localhost:9999/api/normalization/data/session?session_id=42&confirm=true"
```

**Пример ответа:**
```json
{
  "success": true,
  "message": "Результаты нормализации для сессии 42 успешно удалены",
  "rows_affected": 1250,
  "count_before": 1250,
  "session_id": 42
}
```

**Коды ответа:**
- `200 OK` - успешное удаление (или нет данных для удаления)
- `400 Bad Request` - отсутствует подтверждение или неверный session_id
- `405 Method Not Allowed` - неверный HTTP метод
- `500 Internal Server Error` - ошибка при удалении

## Утилита командной строки

### Использование

```bash
go run -tags tool_clear_normalized_data tools/clear_normalized_data.go [опции]
```

### Опции

- `--db, -d PATH` - Путь к базе данных (по умолчанию: `data/normalized_data.db`)
- `--project-id, -p ID` - Удалить данные только для указанного проекта
- `--session-id, -s ID` - Удалить данные только для указанной сессии
- `--help, -h` - Показать справку

### Примеры

**Удаление всех данных:**
```bash
go run -tags tool_clear_normalized_data tools/clear_normalized_data.go
```

**Удаление с указанием пути к БД:**
```bash
go run -tags tool_clear_normalized_data tools/clear_normalized_data.go --db custom/path/normalized_data.db
```

**Удаление данных для проекта:**
```bash
go run -tags tool_clear_normalized_data tools/clear_normalized_data.go --project-id 1
```

**Удаление данных для сессии:**
```bash
go run -tags tool_clear_normalized_data tools/clear_normalized_data.go --session-id 42
```

### Процесс удаления

1. Утилита подключается к базе данных
2. Показывает текущую статистику:
   - Количество нормализованных записей
   - Количество групп товаров
   - Количество атрибутов товаров
3. Запрашивает подтверждение (нужно ввести `DELETE`)
4. Выполняет удаление
5. Показывает результат операции

## Программный API

### database.DB методы

#### DeleteAllNormalizedData()

Удаляет все записи из `normalized_data`. Атрибуты удаляются автоматически через CASCADE.

```go
rowsAffected, err := db.DeleteAllNormalizedData()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Удалено записей: %d\n", rowsAffected)
```

#### DeleteNormalizedDataByProjectID(projectID int)

Удаляет все записи для указанного проекта.

```go
rowsAffected, err := db.DeleteNormalizedDataByProjectID(1)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Удалено записей проекта: %d\n", rowsAffected)
```

#### DeleteNormalizedDataBySessionID(sessionID int)

Удаляет все записи для указанной сессии нормализации.

```go
rowsAffected, err := db.DeleteNormalizedDataBySessionID(42)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Удалено записей сессии: %d\n", rowsAffected)
```

## Сравнение методов удаления

| Метод | Когда использовать | Удаляет |
|-------|-------------------|---------|
| `DeleteAllNormalizedData()` | Полная очистка перед новой нормализацией | Все данные |
| `DeleteNormalizedDataByProjectID()` | Очистка данных конкретного проекта | Данные проекта |
| `DeleteNormalizedDataBySessionID()` | Удаление результатов конкретной сессии | Данные сессии |

## Рекомендации по использованию

1. **Полное удаление** - используйте перед запуском новой нормализации для всех проектов
2. **Удаление по проекту** - используйте для очистки данных конкретного проекта перед повторной нормализацией
3. **Удаление по сессии** - используйте для удаления результатов неудачной или тестовой сессии нормализации

## Безопасность

- Все операции выполняются в транзакциях для атомарности
- API требует явного подтверждения через параметр `confirm=true`
- CLI утилита требует ввода `DELETE` для подтверждения
- Все операции логируются

## Важные замечания

1. **Необратимость**: Удаление данных необратимо. Убедитесь, что у вас есть резервная копия перед удалением.

2. **Каскадное удаление**: При удалении записей из `normalized_data`, связанные записи из `normalized_item_attributes` удаляются автоматически благодаря `ON DELETE CASCADE`.

3. **Сессии нормализации**: Удаление данных не затрагивает записи в таблице `normalization_sessions` в `service.db`. Сессии остаются для истории.

4. **Производительность**: Удаление большого объема данных может занять некоторое время. Операция выполняется в транзакции, поэтому база данных будет заблокирована на время выполнения.

## Резервное копирование

Перед удалением рекомендуется создать резервную копию базы данных:

```bash
# SQLite
cp data/normalized_data.db data/normalized_data.db.backup

# Или через sqlite3
sqlite3 data/normalized_data.db ".backup 'data/normalized_data.db.backup'"
```

## Восстановление

Если у вас есть резервная копия, восстановление выполняется так:

```bash
cp data/normalized_data.db.backup data/normalized_data.db
```

