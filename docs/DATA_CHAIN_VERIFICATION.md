# Проверка цепочки извлечения данных

## Проблема
Базы данных добавлены в проект, но записи номенклатуры и контрагентов не отображаются на фронтенде.

## Реализованные исправления

### 1. Автоматическое создание/обновление upload записей

**Файл**: `server/client_legacy_handlers.go`

**Функция**: `ensureUploadRecordsForDatabase`

**Что делает**:
- Открывает исходную базу данных проекта
- Проверяет наличие upload записей
- Создает новую upload запись, если их нет
- Обновляет существующие upload записи с правильными `client_id` и `project_id`

**Когда вызывается**:
- Автоматически при добавлении базы данных через `handleCreateProjectDatabase`
- Синхронно, чтобы данные были доступны сразу

### 2. Улучшенная логика извлечения номенклатуры

**Файл**: `server/client_legacy_handlers.go`

**Функция**: `getNomenclatureFromMainDB`

**Улучшения**:
- Добавлена fallback логика для случаев, когда upload записей нет
- Если upload записей нет, но есть данные в `catalog_items`, они извлекаются напрямую
- Это позволяет показывать данные даже если upload записи еще не созданы

## Цепочка данных

### Шаг 1: Исходные базы данных (файлы .db)
- Файлы баз данных должны существовать и быть доступны
- Могут содержать таблицы:
  - `catalog_items` - элементы справочников
  - `nomenclature_items` - элементы номенклатуры
  - Исходные таблицы 1С (требуют извлечения)

### Шаг 2: Upload записи
- **КРИТИЧНО**: Должны существовать upload записи с правильными `client_id` и `project_id`
- Без них `getNomenclatureFromMainDB` не сможет найти данные
- **Решение**: Автоматическое создание/обновление при добавлении БД

### Шаг 3: Извлечение в catalog_items/nomenclature_items
- Данные должны быть в таблицах `catalog_items` или `nomenclature_items`
- Связаны с upload записями через `catalogs.upload_id`

### Шаг 4: Нормализованная база данных
- Данные могут быть в таблице `normalized_data` (после нормализации)
- Должны иметь правильный `project_id`

### Шаг 5: API Endpoints
- `GET /api/clients/{clientId}/projects/{projectId}/nomenclature`
- `GET /api/counterparties/normalized?project_id={projectId}`
- Правильно объединяют данные из разных источников

### Шаг 6: Фронтенд
- Запрашивает данные через API
- Отображает номенклатуру и контрагентов

## Тестирование

### Unit тесты
Создан файл `server/client_legacy_handlers_data_chain_test.go` с тестами:

1. **TestEnsureUploadRecordsForDatabase** - проверяет создание/обновление upload записей
2. **TestGetNomenclatureFromMainDBWithUploadRecords** - проверяет извлечение с upload записями
3. **TestGetNomenclatureFromMainDBWithoutUploadRecords** - проверяет fallback логику
4. **TestDataChainIntegration** - полная интеграция цепочки

### Запуск тестов
```bash
go test -v ./server -run TestEnsureUploadRecordsForDatabase
go test -v ./server -run TestGetNomenclatureFromMainDBWithUploadRecords
go test -v ./server -run TestDataChainIntegration
```

### Инструмент проверки
Создан инструмент `tools/check_data_chain/main.go` для проверки реальных баз данных:

```bash
go build -o tools/check_data_chain/check_data_chain.exe tools/check_data_chain/main.go
./tools/check_data_chain/check_data_chain.exe -db <путь_к_бд> -project <project_id> -client <client_id>
```

**Что проверяет**:
- Наличие таблиц (uploads, catalogs, catalog_items, nomenclature_items)
- Наличие и корректность upload записей
- Количество catalog_items и nomenclature_items
- Итоговую статистику

## Проверка в реальных условиях

### 1. Проверка существующих баз данных

Для каждой базы данных проекта нужно проверить:

```bash
# Пример для базы данных проекта
./tools/check_data_chain/check_data_chain.exe \
  -db "data/uploads/Выгрузка_Номенклатура_ERPWE_ukpf_Unknown_2025_11_25_01_11_05_20251126_035703.db" \
  -project 8 \
  -client 1
```

### 2. Проверка через API

После добавления базы данных проверьте:

```bash
# Проверка номенклатуры
curl "http://localhost:9999/api/clients/1/projects/8/nomenclature?page=1&limit=20"

# Проверка контрагентов
curl "http://localhost:9999/api/counterparties/normalized?project_id=8&limit=20"
```

### 3. Проверка логов

После добавления базы данных в логах должно быть:

```
Database created successfully: database_id=X, project_id=Y, name="..."
Updated upload N in <путь> with client_id=X, project_id=Y
```

или

```
Created and updated upload N in <путь> with client_id=X, project_id=Y
```

## Возможные проблемы и решения

### Проблема 1: Нет upload записей
**Симптомы**: Данные не отображаются, хотя они есть в БД
**Решение**: Функция `ensureUploadRecordsForDatabase` должна создавать их автоматически

### Проблема 2: Upload записи без client_id/project_id
**Симптомы**: Данные не отображаются для конкретного проекта
**Решение**: Функция `ensureUploadRecordsForDatabase` обновляет существующие записи

### Проблема 3: Данные не извлечены в catalog_items
**Симптомы**: Таблица catalog_items пуста, хотя исходные таблицы 1С содержат данные
**Решение**: Требуется запуск процесса импорта данных из исходных таблиц 1С

### Проблема 4: Неправильный project_id в normalized_data
**Симптомы**: Данные не отображаются после нормализации
**Решение**: Проверить логику заполнения project_id при нормализации

## Следующие шаги

1. ✅ Автоматическое создание upload записей - реализовано
2. ✅ Fallback логика для извлечения данных - реализовано
3. ⏳ Автоматическое извлечение данных из исходных таблиц 1С - требует дополнительной реализации
4. ⏳ Автоматический маппинг контрагентов - уже реализован в новом API, можно добавить в старый

