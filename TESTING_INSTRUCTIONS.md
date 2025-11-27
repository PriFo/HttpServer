# Инструкции по тестированию API конфигурации

## Быстрый старт

### 1. Запустите сервер

В отдельном окне PowerShell:

```powershell
cd E:\HttpServer
$env:ARLIAI_API_KEY='597dbe7e-16ca-4803-ab17-5fa084909f37'
$env:SERVER_PORT='9999'
$env:LOG_LEVEL='INFO'
go run -tags no_gui main_no_gui.go
```

Или используйте скомпилированный файл:

```powershell
cd E:\HttpServer
$env:ARLIAI_API_KEY='597dbe7e-16ca-4803-ab17-5fa084909f37'
$env:SERVER_PORT='9999'
.\test_build.exe
```

### 2. Запустите быстрый тест

В другом окне PowerShell:

```powershell
cd E:\HttpServer
.\quick_test_config.ps1
```

### 3. Или запустите полный тест с отчетом

```powershell
cd E:\HttpServer
.\test_config_api.ps1
```

Отчет будет сохранен в `config_api_test_report.md`

## Что проверяется

1. **GET /api/config** - безопасная версия без секретов
   - ✅ Поле `log_level` присутствует
   - ✅ Поле `arliai_api_key` отсутствует
   - ✅ Поле `has_arliai_api_key` присутствует

2. **GET /api/config/full** - полная версия с секретами
   - ✅ Поле `log_level` присутствует
   - ✅ Все поля конфигурации доступны

3. **PUT /api/config** - обновление конфигурации
   - ✅ Обновление `log_level` работает
   - ✅ Параметр `reason` сохраняется в истории

4. **GET /api/config/history** - история изменений
   - ✅ Структура ответа корректна
   - ✅ Метаданные (`changed_by`, `change_reason`, `created_at`) сохраняются

5. **Негативные сценарии**
   - ✅ Невалидный `log_level` отклоняется с ошибкой 400

## Проверка логов

После обновления конфигурации проверьте логи сервера на наличие:

```
[Config] Configuration changes detected: log_level: INFO -> DEBUG
[Config] Configuration updated successfully
```

## Устранение проблем

### Сервер не запускается
- Проверьте, что порт 9999 свободен: `netstat -an | findstr :9999`
- Проверьте наличие базы данных `service.db`

### Ошибка 500
- Проверьте логи сервера
- Убедитесь, что все обязательные поля конфигурации заполнены

### Ошибка 400 при обновлении
- Проверьте формат JSON
- Убедитесь, что `log_level` имеет одно из значений: DEBUG, INFO, WARN, ERROR

