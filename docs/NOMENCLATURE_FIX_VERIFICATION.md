# Проверка исправления фильтрации номенклатур

## Как проверить, что исправление работает

### 1. Автоматическая проверка (тест)

Запустите тест:
```bash
go test -v ./server/handlers -run TestGetClientNomenclature_ShowsAllNomenclatures
```

Ожидаемый результат:
```
Successfully retrieved 7 items:
  - Normalized: true
  - From main.db: true
  - From mixed.db: true
  ✓ Fix verified: databases are no longer skipped due to normalized_data table
--- PASS: TestGetClientNomenclature_ShowsAllNomenclatures
```

### 2. Проверка с помощью утилиты

#### Шаг 1: Компиляция утилиты
```bash
go build -o tools/check_nomenclature_count.exe tools/check_nomenclature_count.go
```

#### Шаг 2: Проверка базы данных
```bash
# Для клиента
.\tools\check_nomenclature_count.exe -db "путь\к\базе.db" -client 1 -details

# Для конкретного проекта
.\tools\check_nomenclature_count.exe -db "путь\к\базе.db" -client 1 -project 1 -details
```

#### Шаг 3: Анализ результатов

**До исправления:**
```
Table Status:
  normalized_data:     true (count: 1)
  catalog_items:        true (count: 3)
  nomenclature_items:   false (count: 0)

Summary:
  Normalized items:     1
  Main DB items:       3
  TOTAL:                4

⚠️  IMPORTANT:
  This database contains BOTH normalized_data AND source tables.
  After the fix, ALL items should be shown:
    - 1 normalized items (from normalized_data table)
    - 3 source items (from catalog_items/nomenclature_items)
    - Total: 4 items
```

**После исправления:**
В интерфейсе должны показываться все 4 номенклатуры (1 нормализованная + 3 исходные).

### 3. Проверка через API

#### Запрос
```bash
GET /api/clients/{clientId}/nomenclature?limit=100
```

#### Ожидаемый ответ
```json
{
  "items": [
    {
      "id": 1,
      "code": "NORM001",
      "name": "Нормализованный товар",
      "normalized_name": "Нормализованный товар",
      "source_type": "normalized",
      ...
    },
    {
      "id": 2,
      "code": "CODE001",
      "name": "Товар 1",
      "normalized_name": "Товар 1",
      "source_type": "main",
      "source_database": "path/to/main.db",
      ...
    },
    {
      "id": 3,
      "code": "CODE002",
      "name": "Товар 2",
      "normalized_name": "Товар 2",
      "source_type": "main",
      "source_database": "path/to/main.db",
      ...
    },
    {
      "id": 4,
      "code": "CODE003",
      "name": "Товар 3",
      "normalized_name": "Товар 3",
      "source_type": "main",
      "source_database": "path/to/main.db",
      ...
    }
  ],
  "total": 4,
  "page": 1,
  "limit": 100
}
```

### 4. Проверка в интерфейсе

1. Откройте страницу клиента
2. Перейдите на вкладку "Номенклатура"
3. Убедитесь, что отображаются все номенклатуры:
   - Нормализованные (source_type: "normalized")
   - Исходные из баз проектов (source_type: "main")

### 5. Проверка для баз с normalized_data

Если у вас есть база данных, которая содержит **и** таблицу `normalized_data`, **и** исходные таблицы (`catalog_items` или `nomenclature_items`):

**До исправления:**
- Такая база полностью пропускалась
- Показывались только данные из централизованной `normalized_data.db`

**После исправления:**
- База обрабатывается
- Показываются данные из исходных таблиц этой базы
- Показываются данные из `normalized_data.db`
- Все данные объединяются с дедупликацией

## Критерии успешной проверки

✅ Тест проходит успешно  
✅ Утилита показывает номенклатуры из всех источников  
✅ API возвращает все номенклатуры  
✅ В интерфейсе отображаются все номенклатуры  
✅ Базы с `normalized_data` не пропускаются  

## Известные ограничения

1. **Дедупликация**: Если в разных базах есть номенклатуры с одинаковым кодом и названием, они будут дедуплицированы. Приоритет отдается нормализованным данным.

2. **Производительность**: При большом количестве баз данных запрос может быть медленным. Рекомендуется использовать пагинацию.

3. **Память**: Все результаты сначала собираются в память, затем применяется дедупликация и пагинация. Для очень больших объемов данных может потребоваться оптимизация.

