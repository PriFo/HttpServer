# Улучшения валидации и обработки ошибок для API получения всех контрагентов

## Статус: ✅ ЗАВЕРШЕНО

## Обзор

Добавлена улучшенная валидация параметров запроса и обработка ошибок для endpoint `/api/counterparties/all`.

## Реализованные улучшения

### 1. Валидация обязательных параметров ✅

**client_id:**
- Обязательный параметр
- Должен быть положительным целым числом
- Используется `ValidateIntParam` для валидации

**Примеры ошибок:**
```json
{
  "error": "Invalid client_id: validation error: client_id - must be a valid integer, got: abc"
}
```

### 2. Валидация опциональных параметров ✅

**project_id:**
- Опциональный параметр
- Если указан, должен быть положительным целым числом
- Валидируется через `ValidateIntParam`

**offset:**
- Опциональный параметр (по умолчанию: 0)
- Должен быть неотрицательным целым числом
- Автоматически устанавливается в 0, если указано отрицательное значение

**limit:**
- Опциональный параметр (по умолчанию: 100)
- Должен быть в диапазоне от 1 до 1000
- Валидируется через `ValidateIntParam` с ограничениями

**Примеры ошибок:**
```json
{
  "error": "Invalid limit: validation error: limit - must be at most 1000, got: 2000"
}
```

### 3. Валидация строковых параметров ✅

**search:**
- Опциональный параметр
- Максимальная длина: 500 символов
- Автоматически обрезается пробелами
- Валидируется через `ValidateSearchQuery`

**source:**
- Опциональный параметр
- Допустимые значения: "database", "normalized" или пустая строка (все)
- Регистронезависимая валидация
- Автоматическое приведение к нижнему регистру

**Примеры ошибок:**
```json
{
  "error": "Invalid source parameter. Must be 'database', 'normalized', or empty"
}
```

**sort_by:**
- Опциональный параметр
- Допустимые значения: "name", "quality", "source", "id" или пустая строка (по умолчанию)
- Регистронезависимая валидация
- Автоматическое приведение к нижнему регистру

**Примеры ошибок:**
```json
{
  "error": "Invalid sort_by parameter. Must be 'name', 'quality', 'source', 'id', or empty"
}
```

**order:**
- Опциональный параметр
- Допустимые значения: "asc", "desc" или пустая строка (по умолчанию)
- Регистронезависимая валидация
- Автоматическое приведение к нижнему регистру

**Примеры ошибок:**
```json
{
  "error": "Invalid order parameter. Must be 'asc', 'desc', or empty"
}
```

### 4. Улучшенное логирование ✅

**Логирование запросов:**
- Все входящие запросы логируются с полными параметрами
- Уровень: INFO
- Включает все параметры запроса для отладки

**Пример лога:**
```
[INFO] GetAllCounterparties request - client_id: 1, project_id: <nil>, offset: 0, limit: 100, search: "", source: "", sort_by: "", order: ""
```

**Логирование ошибок:**
- Все ошибки логируются с детальной информацией
- Уровень: ERROR
- Включает контекст запроса (client_id, параметры)

**Пример лога ошибки:**
```
[ERROR] Failed to get counterparties for client_id 1: database connection failed
```

**Логирование успешных ответов:**
- Успешные запросы логируются с метриками
- Уровень: INFO
- Включает статистику (total, returned, processing_time)

**Пример лога успеха:**
```
[INFO] GetAllCounterparties success - client_id: 1, total: 150, returned: 100, processing_time: 1250ms
```

### 5. Обработка ошибок ✅

**Использование стандартных функций валидации:**
- `ValidateIntParam` - для целочисленных параметров
- `ValidateSearchQuery` - для поисковых запросов
- `HandleValidationError` - для единообразной обработки ошибок валидации

**Структурированные сообщения об ошибках:**
- Все ошибки валидации возвращают понятные сообщения
- HTTP статус коды соответствуют типу ошибки:
  - `400 Bad Request` - ошибки валидации
  - `405 Method Not Allowed` - неподдерживаемый HTTP метод
  - `500 Internal Server Error` - внутренние ошибки сервера

## Примеры использования

### Валидный запрос
```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&limit=50&sort_by=quality&order=desc"
```

### Ошибка валидации client_id
```bash
curl "http://localhost:9999/api/counterparties/all?client_id=abc"
# Ответ: {"error": "Invalid client_id: validation error: client_id - must be a valid integer, got: abc"}
```

### Ошибка валидации limit
```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&limit=2000"
# Ответ: {"error": "Invalid limit: validation error: limit - must be at most 1000, got: 2000"}
```

### Ошибка валидации source
```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&source=invalid"
# Ответ: {"error": "Invalid source parameter. Must be 'database', 'normalized', or empty"}
```

### Ошибка валидации sort_by
```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&sort_by=invalid"
# Ответ: {"error": "Invalid sort_by parameter. Must be 'name', 'quality', 'source', 'id', or empty"}
```

### Ошибка валидации order
```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&order=invalid"
# Ответ: {"error": "Invalid order parameter. Must be 'asc', 'desc', or empty"}
```

## Технические детали

### Измененные файлы

1. **`server/server.go`**
   - Улучшен handler `handleGetAllCounterparties`
   - Добавлена валидация всех параметров
   - Добавлено логирование запросов и ответов
   - Улучшена обработка ошибок

### Используемые функции валидации

- `ValidateIntParam` - валидация целочисленных параметров с ограничениями
- `ValidateSearchQuery` - валидация поисковых запросов
- `HandleValidationError` - единообразная обработка ошибок валидации

### Преимущества

1. **Безопасность**: Предотвращение некорректных запросов
2. **Отладка**: Детальное логирование всех запросов и ошибок
3. **Пользовательский опыт**: Понятные сообщения об ошибках
4. **Производительность**: Раннее обнаружение ошибок до выполнения запроса к БД
5. **Консистентность**: Единообразная обработка ошибок во всем API

## Проверка

✅ Код компилируется успешно  
✅ Линтер не выявил ошибок  
✅ Валидация всех параметров работает корректно  
✅ Логирование запросов и ошибок работает  
✅ Сообщения об ошибках понятны и информативны  

## Готово к использованию

Улучшения валидации и обработки ошибок полностью реализованы и готовы к использованию в продакшене.

**Ключевые улучшения:**
- ✅ Валидация всех параметров запроса
- ✅ Детальное логирование запросов и ответов
- ✅ Понятные сообщения об ошибках
- ✅ Единообразная обработка ошибок
- ✅ Раннее обнаружение ошибок

