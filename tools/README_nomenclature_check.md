# Утилита проверки количества номенклатур

## Описание

Утилита `check_nomenclature_count.go` позволяет проверить, сколько номенклатур находится в базе данных и из каких источников они берутся.

## Использование

### Компиляция

```bash
cd tools
go build -o check_nomenclature_count check_nomenclature_count.go
```

### Запуск

```bash
# Проверка всех номенклатур клиента
./check_nomenclature_count -db /path/to/database.db -client 1

# Проверка номенклатур конкретного проекта
./check_nomenclature_count -db /path/to/database.db -client 1 -project 1

# С детальной информацией (первые 10 записей из каждой таблицы)
./check_nomenclature_count -db /path/to/database.db -client 1 -details
```

### Параметры

- `-db` (обязательный) - путь к файлу базы данных
- `-client` (обязательный) - ID клиента
- `-project` (опциональный) - ID проекта для фильтрации
- `-details` (опциональный) - показать детальную информацию о первых 10 записях

## Что проверяет утилита

1. **Таблица `normalized_data`** - нормализованные номенклатуры
2. **Таблица `catalog_items`** - исходные номенклатуры из каталогов
3. **Таблица `nomenclature_items`** - исходные номенклатуры из справочника номенклатуры

## Пример вывода

```
Checking nomenclature in database: /path/to/database.db
Client ID: 1
================================================================================

Table Status:
  normalized_data:     true (count: 1)
  catalog_items:        true (count: 3)
  nomenclature_items:   false (count: 0)

Summary:
  Normalized items:     1
  Main DB items:       3
    - catalog_items:    3
    - nomenclature_items: 0
  TOTAL:                4

⚠️  IMPORTANT:
  This database contains BOTH normalized_data AND source tables.
  After the fix, ALL items should be shown:
    - 1 normalized items (from normalized_data table)
    - 3 source items (from catalog_items/nomenclature_items)
    - Total: 4 items
```

## Важно

После исправления багфикса, базы данных, которые содержат **и** таблицу `normalized_data`, **и** исходные таблицы (`catalog_items` или `nomenclature_items`), должны показывать **все** номенклатуры из обоих источников.

Раньше такие базы пропускались полностью, и показывались только нормализованные номенклатуры из централизованной базы `normalized_data.db`.

