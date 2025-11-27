# План универсального обнаружения контрагентов в БД

## Проблема
Каждая выгрузка из 1С имеет разную структуру:
- Разные названия таблиц (`Контрагенты`, `Клиенты`, `counterparties`, `clients`)
- Разные названия колонок (`Наименование`, `name`, `full_name`)
- Разные схемы данных (с каталогами или без)

## Решение: Умный детектор контрагентов

### 1. Автоматическое обнаружение структуры

```sql
-- Поиск таблиц с контрагентами
SELECT name FROM sqlite_master 
WHERE type='table' 
AND (
  LOWER(name) LIKE '%контрагент%' 
  OR LOWER(name) LIKE '%counterpart%'
  OR LOWER(name) LIKE '%клиент%'
  OR name IN ('catalog_items', 'documents')
)
```

### 2. Анализ схемы таблицы

```sql
-- Получение колонок таблицы
PRAGMA table_info(table_name);

-- Поиск ключевых полей:
-- - ИНН/БИН: inn, ИНН, БИН, bin, tax_id
-- - Наименование: name, Наименование, full_name, title
-- - ОГРН: ogrn, ОГРН
-- - КПП: kpp, КПП
```

### 3. Кэширование метаданных БД

Создать таблицу `database_metadata`:
```sql
CREATE TABLE IF NOT EXISTS database_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    database_id INTEGER NOT NULL,
    table_name TEXT NOT NULL,
    entity_type TEXT NOT NULL, -- 'counterparty', 'nomenclature', 'document'
    column_mappings TEXT NOT NULL, -- JSON: {"name": "Наименование", "inn": "ИНН"}
    detection_confidence REAL, -- 0.0-1.0
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(database_id, table_name, entity_type)
);
```

### 4. Универсальный запрос

```go
type CounterpartyDetectionResult struct {
    TableName        string
    NameColumn       string
    INNColumn        string
    OGRNColumn       string
    KPPColumn        string
    Confidence       float64
}

func DetectCounterpartyStructure(db *sql.DB) (*CounterpartyDetectionResult, error) {
    // 1. Найти подходящие таблицы
    // 2. Проанализировать колонки
    // 3. Вычислить confidence score
    // 4. Вернуть лучшее совпадение
}
```

### 5. Динамический SQL-запрос

```go
func GetCounterparties(dbPath string, metadata *CounterpartyDetectionResult) ([]Counterparty, error) {
    query := fmt.Sprintf(`
        SELECT 
            %s as name,
            %s as inn,
            %s as ogrn,
            %s as kpp
        FROM %s
        WHERE %s IS NOT NULL AND %s != ''
    `,
        metadata.NameColumn,
        metadata.INNColumn,
        metadata.OGRNColumn,
        metadata.KPPColumn,
        metadata.TableName,
        metadata.NameColumn,
        metadata.NameColumn,
    )
    // ...
}
```

## Реализация

### database/counterparty_detector.go
```go
package database

type CounterpartyDetector struct {
    db        *sql.DB
    serviceDB *ServiceDB
}

func NewCounterpartyDetector(db *sql.DB, serviceDB *ServiceDB) *CounterpartyDetector

func (d *CounterpartyDetector) DetectStructure(databaseID int, dbPath string) error

func (d *CounterpartyDetector) GetCounterparties(databaseID int) ([]map[string]interface{}, error)

func (d *CounterpartyDetector) GetCachedMetadata(databaseID int) (*DatabaseMetadata, error)
```

### Алгоритм определения confidence:
1. Название таблицы содержит "контрагент" → +0.4
2. Найдена колонка "ИНН"/"inn" → +0.3
3. Найдена колонка "Наименование"/"name" → +0.2
4. Найдена колонка "ОГРН" → +0.1
5. Total confidence > 0.7 → используем эту таблицу

## Использование

```go
// В API handler
func (h *CounterpartyHandler) HandleGetAllCounterparties(w http.ResponseWriter, r *http.Request) {
    clientID := getClientID(r)
    
    // Получаем все БД клиента
    databases, err := h.serviceDB.GetClientDatabases(clientID)
    
    allCounterparties := []Counterparty{}
    
    for _, db := range databases {
        // Проверяем кэш метаданных
        metadata, err := h.detector.GetCachedMetadata(db.ID)
        if err != nil {
            // Первый раз - обнаруживаем структуру
            metadata, err = h.detector.DetectStructure(db.ID, db.FilePath)
            if err != nil {
                continue
            }
        }
        
        // Получаем контрагентов используя обнаруженную структуру
        counterparties, err := h.detector.GetCounterparties(db.ID)
        if err != nil {
            continue
        }
        
        allCounterparties = append(allCounterparties, counterparties...)
    }
    
    // Дедупликация по ИНН
    unique := deduplicateByINN(allCounterparties)
    
    writeJSON(w, unique)
}
```

## Преимущества

1. ✅ Работает с любой структурой БД
2. ✅ Кэширует метаданные для быстродействия
3. ✅ Автоматически адаптируется к новым выгрузкам
4. ✅ Не требует ручной настройки
5. ✅ Поддерживает множественные источники данных

## Следующие шаги

1. Создать `database/counterparty_detector.go`
2. Добавить миграцию для `database_metadata`
3. Обновить `HandleGetAllCounterparties`
4. Добавить UI для просмотра обнаруженных структур
5. Реализовать ручное переопределение (если автоопределение ошиблось)

