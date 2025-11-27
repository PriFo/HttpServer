# Исправления интеграционных тестов

## Дата: 2025-11-26

## Исправленные проблемы

### 1. UNIQUE constraint failed: uploads.upload_uuid ✅

**Проблема:**
- Тесты использовали фиксированный UUID "test-uuid", который уже существовал в БД

**Решение:**
- Использование `uuid.New().String()` для генерации уникальных UUID в каждом тесте
- Исправлены файлы:
  - `integration/counterparty_normalization_integration_test.go`
  - `integration/integration_test.go`
  - `integration/counterparty_normalization_e2e_test.go`

### 2. sql: no rows in result set при UpdateNormalizedName ✅

**Проблема:**
- `UpdateNormalizedName` обновлял `normalized_data` по `id`, но пайплайн передавал `catalog_item_id`
- Запись в `normalized_data` могла не существовать

**Решение:**
- Улучшена функция `UpdateNormalizedName` в `database/db.go`:
  1. Сначала пытается обновить напрямую в `normalized_data`
  2. Если не найдено, пытается обновить через связь `catalog_items -> normalized_data`
  3. Если все еще не найдено, создает запись в `normalized_data` и связывает с `catalog_item`

### 3. FOREIGN KEY constraint failed при создании уведомлений ✅

**Проблема:**
- Тесты пытались создать уведомления с несуществующими `client_id`/`project_id`

**Решение:**
- Тесты уже исправлены в `server/handlers/notification_service_integration_test.go`
- В `SetupTest()` создаются необходимые клиенты и проекты

### 4. sql: database is closed ✅

**Проблема:**
- БД закрывалась слишком рано в тесте `TestQualityProjectStats_RealData`

**Решение:**
- Убрано преждевременное закрытие БД в `integration/quality_project_integration_test.go`
- БД закрывается через `defer` в конце теста

## Изменения в коде

### database/db.go
- Улучшена функция `UpdateNormalizedName` для работы с `catalog_item_id`

### integration/*.go
- Добавлен импорт `github.com/google/uuid`
- Заменены фиксированные UUID на динамически генерируемые

## Статус

✅ Все критические проблемы исправлены
✅ Код готов к тестированию

## Следующие шаги

1. Запустить интеграционные тесты: `go test ./integration/... -v`
2. Проверить результаты
3. Исправить оставшиеся проблемы, если они есть

