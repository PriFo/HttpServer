# Руководство по тестированию API конфигурации

## Подготовка

1. Убедитесь, что сервер запущен на `http://localhost:9999`
2. Убедитесь, что установлены необходимые инструменты:
   - PowerShell (для Windows)
   - curl (опционально, для ручного тестирования)
   - jq (опционально, для форматирования JSON)

## Запуск автоматических тестов

### Windows (PowerShell)
```powershell
.\test_config_api.ps1
```

### Linux/Mac (Bash)
```bash
./test_config_api.sh
```

## Ручное тестирование

### 1. Тест GET /api/config (безопасная версия)

**Запрос:**
```bash
curl http://localhost:9999/api/config
```

**Ожидаемый результат:**
- Статус: 200 OK
- В ответе НЕ должно быть поля `arliai_api_key`
- Должно быть поле `has_arliai_api_key` (boolean)
- Должно быть поле `log_level` (string)
- API ключи в `enrichment.services.*.api_key` должны быть пустыми

**Проверка:**
```bash
curl http://localhost:9999/api/config | jq '.log_level'
curl http://localhost:9999/api/config | jq 'has("arliai_api_key")'  # должно быть false
curl http://localhost:9999/api/config | jq '.has_arliai_api_key'
```

### 2. Тест GET /api/config/full (полная версия)

**Запрос:**
```bash
curl http://localhost:9999/api/config/full
```

**Ожидаемый результат:**
- Статус: 200 OK
- В ответе должно быть поле `arliai_api_key` (если установлен)
- Должны быть API ключи в `enrichment.services.*.api_key` (если установлены)
- Должно быть поле `log_level`

**Проверка:**
```bash
curl http://localhost:9999/api/config/full | jq '.log_level'
curl http://localhost:9999/api/config/full | jq 'has("arliai_api_key")'  # должно быть true
```

### 3. Тест PUT /api/config (обновление log_level)

**Шаг 1: Получить текущую конфигурацию**
```bash
curl http://localhost:9999/api/config/full > current_config.json
```

**Шаг 2: Изменить log_level**
```bash
# Используя jq
cat current_config.json | jq '.log_level = "DEBUG"' > updated_config.json
```

**Шаг 3: Отправить обновление**
```bash
curl -X PUT http://localhost:9999/api/config?reason=QA%20test%20update \
  -H "Content-Type: application/json" \
  -d @updated_config.json
```

**Ожидаемый результат:**
- Статус: 200 OK
- В ответе должно быть обновленное значение `log_level: "DEBUG"`

**Проверка:**
```bash
curl http://localhost:9999/api/config/full | jq '.log_level'  # должно быть "DEBUG"
```

**Шаг 4: Проверить логи сервера**

Найдите в логах сервера записи с префиксом `[Config]`:
```
[Config] Configuration changes detected: log_level: INFO -> DEBUG
[Config] Configuration updated successfully
```

### 4. Тест GET /api/config/history

**Запрос:**
```bash
curl http://localhost:9999/api/config/history
```

**Ожидаемый результат:**
- Статус: 200 OK
- Структура ответа:
  ```json
  {
    "current_version": 2,
    "history": [
      {
        "version": 2,
        "config_json": "...",
        "changed_by": "127.0.0.1",
        "change_reason": "QA test update",
        "created_at": "2024-01-01T12:00:00Z"
      }
    ],
    "count": 1
  }
  ```

**Проверка:**
```bash
curl http://localhost:9999/api/config/history | jq '.current_version'
curl http://localhost:9999/api/config/history | jq '.history[0].version'
curl http://localhost:9999/api/config/history | jq '.history[0].changed_by'
curl http://localhost:9999/api/config/history | jq '.history[0].change_reason'  # должно быть "QA test update"
curl http://localhost:9999/api/config/history | jq '.history[0].created_at'
```

### 5. Тест негативных сценариев

#### 5.1 Невалидный JSON
```bash
curl -X PUT http://localhost:9999/api/config \
  -H "Content-Type: application/json" \
  -d '{"log_level":}'
```

**Ожидаемый результат:**
- Статус: 400 Bad Request
- Сообщение об ошибке парсинга JSON

#### 5.2 Невалидное значение log_level
```bash
curl -X PUT http://localhost:9999/api/config \
  -H "Content-Type: application/json" \
  -d '{"log_level": "INVALID", "port": "9999", ...}'
```

**Ожидаемый результат:**
- Статус: 400 Bad Request
- Сообщение валидации: "invalid log level: INVALID (valid: DEBUG, INFO, WARN, ERROR)"

#### 5.3 Некорректный параметр limit
```bash
curl http://localhost:9999/api/config/history?limit=-1
```

**Ожидаемый результат:**
- Статус: 200 OK (или 400, в зависимости от реализации)
- Если 200, то limit должен быть приведен к 1 или 10 по умолчанию

#### 5.4 Слишком большое значение limit
```bash
curl http://localhost:9999/api/config/history?limit=1000
```

**Ожидаемый результат:**
- Статус: 200 OK
- Количество записей должно быть ограничено максимумом (100)

## Формат отчета

После выполнения тестов будет создан файл `config_api_test_report.md` со следующей структурой:

```markdown
# Отчет о тестировании API конфигурации

Дата: 2024-01-01 12:00:00

## Тест: GET /api/config (безопасная версия)

**Запрос:**
- Метод: GET
- URL: http://localhost:9999/api/config

**Ответ:**
- Статус: 200
- Body:
```json
{
  "log_level": "INFO",
  "has_arliai_api_key": true,
  ...
}
```

**Результат:** ✅ PASS

---
```

## Проверка логов сервера

После обновления конфигурации проверьте логи сервера на наличие записей:

1. Записи с префиксом `[Config]`
2. Формат записи об изменении: `log_level: <старое> -> <новое>`
3. Запись об успешном обновлении: `Configuration updated successfully`

Пример:
```
[Config] Configuration changes detected: log_level: INFO -> DEBUG
[Config] Configuration updated successfully
```

## Дополнительные проверки

### Проверка версионирования
Выполните несколько последовательных обновлений и убедитесь, что версия увеличивается:
```bash
# Обновление 1
curl -X PUT http://localhost:9999/api/config?reason=Test1 -H "Content-Type: application/json" -d @config1.json
curl http://localhost:9999/api/config/history | jq '.current_version'  # должно быть 2

# Обновление 2
curl -X PUT http://localhost:9999/api/config?reason=Test2 -H "Content-Type: application/json" -d @config2.json
curl http://localhost:9999/api/config/history | jq '.current_version'  # должно быть 3
```

### Проверка неизмененных полей
Обновите только `log_level`, оставив остальные поля без изменений:
```bash
# Получить текущую конфигурацию
curl http://localhost:9999/api/config/full > before.json

# Изменить только log_level
cat before.json | jq '.log_level = "DEBUG"' > after.json

# Отправить обновление
curl -X PUT http://localhost:9999/api/config?reason=Test -H "Content-Type: application/json" -d @after.json

# Проверить, что остальные поля не изменились
curl http://localhost:9999/api/config/full > result.json
diff <(jq -S . before.json) <(jq -S . result.json)  # должны отличаться только log_level
```

## Устранение проблем

### Сервер не отвечает
- Убедитесь, что сервер запущен: `netstat -an | findstr :9999` (Windows) или `netstat -an | grep 9999` (Linux)
- Проверьте логи сервера на наличие ошибок

### Ошибка 500 Internal Server Error
- Проверьте логи сервера
- Убедитесь, что база данных доступна
- Проверьте, что все обязательные поля конфигурации заполнены

### Ошибка 400 Bad Request
- Проверьте формат JSON
- Убедитесь, что все обязательные поля присутствуют
- Проверьте валидность значений (особенно log_level)

## Контакты

При возникновении проблем обратитесь к разработчикам или создайте issue в системе отслеживания ошибок.

