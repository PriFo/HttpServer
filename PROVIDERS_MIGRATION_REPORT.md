# Отчет о миграции таблицы providers

**Дата:** 2025-11-25  
**Статус:** ✅ Завершено

## Выполненные задачи

### 1. ✅ Обновлена функция CreateProvidersTable

**Файл:** `database/provider_migrations.go`

Обновлена структура таблицы `providers` согласно требованиям:

**Новая структура:**
- `id INTEGER PRIMARY KEY AUTOINCREMENT` (было: `id TEXT PRIMARY KEY`)
- `name TEXT NOT NULL UNIQUE`
- `type TEXT NOT NULL` (новое поле)
- `config TEXT` (новое поле для JSON конфигурации)
- `is_active BOOLEAN NOT NULL DEFAULT 1` (было: `enabled BOOLEAN DEFAULT TRUE`)
- `created_at DATETIME DEFAULT CURRENT_TIMESTAMP` (было: `TIMESTAMP`)
- `updated_at DATETIME DEFAULT CURRENT_TIMESTAMP` (было: `TIMESTAMP`)

**Удалены поля:**
- `api_key TEXT` (перенесено в `config` JSON)
- `base_url TEXT` (перенесено в `config` JSON)
- `priority INTEGER` (больше не используется)
- `channels INTEGER` (перенесено в `config` JSON)

### 2. ✅ Создана функция миграции для обновления существующих таблиц

**Функция:** `migrateProvidersTable()`

Функция автоматически определяет структуру существующей таблицы и выполняет миграцию:

- Если таблица имеет старую структуру (`id TEXT`), создается новая таблица с правильной структурой, данные мигрируются, старая таблица удаляется
- Если таблица уже имеет `id INTEGER`, добавляются недостающие колонки (`type`, `config`, `is_active`)
- Значения из старых полей (`enabled`, `api_key`, `base_url`) мигрируются в новую структуру
- Создаются необходимые индексы

### 3. ✅ Обновлена структура Provider в Go коде

**Файл:** `database/provider_migrations.go`

```go
type Provider struct {
	ID        int      // INTEGER вместо TEXT
	Name      string
	Type      string   // Новое поле
	Config    string   // JSON с конфигурацией
	IsActive  bool     // Вместо Enabled
	CreatedAt string
	UpdatedAt string
}
```

### 4. ✅ Обновлены методы GetProviders и GetActiveProviders

Методы обновлены для работы с новой структурой:
- Используют `is_active` вместо `enabled`
- Извлекают данные из новых полей `type` и `config`
- Корректно обрабатывают NULL значения через `sql.NullString`

### 5. ✅ Обновлен код использования Provider в других модулях

**Файлы:**
- `server/multi_provider_client.go` - обновлен для работы с новой структурой
- `server/worker_config_legacy.go` - обновлен для извлечения данных из config JSON

**Изменения:**
- `p.Enabled` → `p.IsActive`
- `p.ID` (string) → `p.Type` (string) для идентификации провайдера
- `p.APIKey`, `p.BaseURL` → извлекаются из `p.Config` JSON
- `p.Channels`, `p.Priority` → извлекаются из `p.Config` JSON или используют значения по умолчанию

### 6. ✅ Проверена регистрация миграции

Миграция зарегистрирована в `database/schema.go`:
- Строка 805: `if err := CreateProvidersTable(db); err != nil`
- Строка 1380: `if err := CreateProvidersTable(db); err != nil`

`InitServiceSchema` вызывается в `database/service_db.go` на строке 156 при создании `ServiceDB`.

## SQL миграции

### UP миграция (создание таблицы)

```sql
CREATE TABLE IF NOT EXISTS providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    config TEXT,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_providers_type ON providers(type);
CREATE INDEX IF NOT EXISTS idx_providers_is_active ON providers(is_active);
CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);
```

### DOWN миграция (удаление таблицы)

```sql
DROP INDEX IF EXISTS idx_providers_type;
DROP INDEX IF EXISTS idx_providers_is_active;
DROP INDEX IF EXISTS idx_providers_name;
DROP TABLE IF EXISTS providers;
```

## Примеры использования

### Получение провайдеров

```go
providers, err := serviceDB.GetProviders()
if err != nil {
    log.Fatalf("Failed to get providers: %v", err)
}

for _, p := range providers {
    fmt.Printf("Provider: %s (Type: %s, Active: %v)\n", p.Name, p.Type, p.IsActive)
    
    // Парсим config JSON
    if p.Config != "" {
        var config map[string]interface{}
        json.Unmarshal([]byte(p.Config), &config)
        apiKey := config["api_key"].(string)
        baseURL := config["base_url"].(string)
    }
}
```

### Получение только активных провайдеров

```go
activeProviders, err := serviceDB.GetActiveProviders()
if err != nil {
    log.Fatalf("Failed to get active providers: %v", err)
}
```

## Обратная совместимость

Миграция обеспечивает обратную совместимость:

1. **Автоматическое определение структуры** - функция `migrateProvidersTable()` определяет, какая структура у существующей таблицы
2. **Миграция данных** - данные из старых полей автоматически переносятся в новую структуру
3. **Обновление кода** - все места использования `Provider` обновлены для работы с новой структурой

## Проверка

- ✅ Компиляция проекта проходит успешно
- ✅ Линтер не выявил ошибок
- ✅ Миграция зарегистрирована в `InitServiceSchema`
- ✅ Все методы обновлены для работы с новой структурой

## Следующие шаги

1. Протестировать миграцию на существующей базе данных
2. Убедиться, что все провайдеры корректно мигрированы
3. Проверить работу multi-provider client после миграции

## Примечания

- Поле `priority` больше не используется в новой структуре (можно добавить в `config` JSON при необходимости)
- Поле `channels` хранится в `config` JSON
- API ключи и URL провайдеров хранятся в `config` JSON для гибкости конфигурации

