# Отчет о совместимости типов данных и средствах анализа

## 1. Совместимость типов данных

### Структура DatabasePreviewStats

**Бэкенд (Go):**
```go
type DatabasePreviewStats struct {
    DatabaseID        int                 `json:"database_id"`
    DatabaseName      string              `json:"database_name"`
    FilePath          string              `json:"file_path"`
    NomenclatureCount int64               `json:"nomenclature_count"`
    CounterpartyCount int64               `json:"counterparty_count"`
    TotalRecords      int64               `json:"total_records"`
    DatabaseSize      int64               `json:"database_size"`
    Error             string              `json:"error,omitempty"`
    IsAccessible      bool                `json:"is_accessible"`
    IsValid           bool                `json:"is_valid"`
    Completeness      *CompletenessMetrics `json:"completeness,omitempty"`
}
```

**Фронтенд (TypeScript):**
```typescript
export interface DatabasePreviewStats {
  database_id: number
  database_name: string
  file_path: string
  nomenclature_count: number
  counterparty_count: number
  total_records: number
  database_size: number
  error?: string
  is_accessible?: boolean
  is_valid?: boolean
  completeness?: CompletenessMetrics
}
```

✅ **Совместимость:** Полная совместимость. Все поля совпадают по названиям и типам.

### Структура PreviewStatsResponse

**Бэкенд (Go):**
```go
response := map[string]interface{}{
    "total_databases":      len(activeDatabases),
    "accessible_databases": accessibleCount,
    "valid_databases":      validCount,
    "total_nomenclature":   totalNomenclature,
    "total_counterparties": totalCounterparties,
    "total_records":        totalRecords,
    "estimated_duplicates": estimatedDuplicates,
    "duplicate_groups":     duplicateGroups,
    "completeness_metrics": overallCompleteness,
    "databases":            stats,
}
```

**Фронтенд (TypeScript):**
```typescript
export interface PreviewStatsResponse {
  total_databases: number
  accessible_databases?: number
  valid_databases?: number
  total_nomenclature: number
  total_counterparties: number
  total_records: number
  estimated_duplicates: number
  duplicate_groups?: number
  completeness_metrics?: CompletenessMetrics
  databases: DatabasePreviewStats[]
}
```

✅ **Совместимость:** Полная совместимость. Все поля совпадают.

### Структура CompletenessMetrics

**Бэкенд (Go):**
```go
type CompletenessMetrics struct {
    NomenclatureCompleteness struct {
        ArticlesPercent      float64 `json:"articles_percent"`
        UnitsPercent         float64 `json:"units_percent"`
        DescriptionsPercent  float64 `json:"descriptions_percent"`
        OverallCompleteness  float64 `json:"overall_completeness"`
    } `json:"nomenclature_completeness,omitempty"`
    CounterpartyCompleteness struct {
        INNPercent          float64 `json:"inn_percent"`
        AddressPercent      float64 `json:"address_percent"`
        ContactsPercent     float64 `json:"contacts_percent"`
        OverallCompleteness float64 `json:"overall_completeness"`
    } `json:"counterparty_completeness,omitempty"`
}
```

**Фронтенд (TypeScript):**
```typescript
export interface CompletenessMetrics {
  nomenclature_completeness?: {
    articles_percent: number
    units_percent: number
    descriptions_percent: number
    overall_completeness: number
  }
  counterparty_completeness?: {
    inn_percent: number
    address_percent: number
    contacts_percent: number
    overall_completeness: number
  }
}
```

✅ **Совместимость:** Полная совместимость. Все поля совпадают.

## 2. Поддержка параметра normalizationType

### Текущее состояние

❌ **Проблема:** Бэкенд не принимает параметр `normalizationType` в endpoint `/api/clients/{clientId}/projects/{projectId}/normalization/preview-stats`.

**Фронтенд передает:**
- `normalizationType` в компоненте `NormalizationPreviewStats`
- Но не передает его в API запрос

**Бэкенд ожидает:**
- Только `clientId` и `projectId` в пути
- Не обрабатывает query параметры для фильтрации по типу нормализации

### Рекомендации

1. **Добавить поддержку query параметра `normalization_type` в бэкенд:**
   - Значения: `nomenclature`, `counterparties`, `both`
   - Фильтровать статистику по типу данных

2. **Обновить фронтенд для передачи параметра:**
   - Добавить `normalization_type` в query string запроса
   - Фильтровать результаты на фронтенде, если бэкенд не поддерживает

## 3. Средства анализа данных, реализованные на бэкенде

### 3.1. Подсчет записей (`countDatabaseRecords`)

**Функциональность:**
- Подсчет номенклатуры из таблицы `nomenclature_items`
- Подсчет контрагентов из таблицы `counterparties`
- Поддержка альтернативных таблиц (`catalog_items` с определением типа по каталогу)
- Определение размера файла БД

**Метрики:**
- `nomenclature_count` - количество записей номенклатуры
- `counterparty_count` - количество записей контрагентов
- `total_records` - общее количество записей
- `database_size` - размер файла БД в байтах

### 3.2. Метрики заполненности (`calculateCompletenessMetrics`)

**Для номенклатуры:**
- `articles_percent` - процент записей с артикулами (`nomenclature_code`)
- `units_percent` - процент записей с единицами измерения (из `attributes_xml`)
- `descriptions_percent` - процент записей с описаниями (`characteristic_name`)
- `overall_completeness` - среднее значение всех метрик

**Для контрагентов:**
- `inn_percent` - процент записей с ИНН/БИН (`inn`, `bin` или из `attributes_xml`)
- `address_percent` - процент записей с адресами (`legal_address`, `postal_address` или из `attributes_xml`)
- `contacts_percent` - процент записей с контактами (`contact_phone`, `contact_email` или из `attributes_xml`)
- `overall_completeness` - среднее значение всех метрик

**Агрегация:**
- `calculateOverallCompleteness` - рассчитывает общие метрики по всем БД проекта

### 3.3. Подсчет дубликатов (`countQuickDuplicates`)

**Функциональность:**
- Быстрый подсчет потенциальных дубликатов по именам
- Работает только для БД с < 100,000 записей (для производительности)
- Для больших БД использует оценку (~5% от общего количества)

**Метрики:**
- `estimated_duplicates` - количество потенциальных дубликатов
- `duplicate_groups` - количество групп дубликатов

### 3.4. Проверка доступности БД

**Функциональность:**
- Проверка существования файла БД (`os.Stat`)
- Проверка подключения к БД (`conn.PingContext`)
- Валидация структуры БД (проверка наличия таблиц)

**Метрики:**
- `is_accessible` - доступность файла БД
- `is_valid` - валидность структуры БД
- `error` - сообщение об ошибке, если есть

### 3.5. Агрегация статистики

**Функциональность:**
- Подсчет общего количества БД
- Подсчет доступных БД
- Подсчет валидных БД
- Агрегация метрик по всем БД проекта

## 4. Рекомендации по улучшению

### 4.1. Добавить поддержку normalizationType

**Бэкенд:**
```go
// В HandleGetClientProjectNormalizationPreviewStats
normalizationType := r.URL.Query().Get("normalization_type")
if normalizationType == "" {
    normalizationType = "both" // По умолчанию
}

// Фильтровать статистику по типу
if normalizationType == "nomenclature" {
    // Показывать только номенклатуру
} else if normalizationType == "counterparties" {
    // Показывать только контрагентов
}
```

**Фронтенд:**
```typescript
const response = await fetch(
  `/api/clients/${clientId}/projects/${projectId}/normalization/preview-stats?normalization_type=${normalizationType}`,
  { cache: 'no-store' }
)
```

### 4.2. Улучшить анализ дубликатов

- Добавить более точный алгоритм поиска дубликатов
- Поддержка семантического поиска дубликатов
- Классификация типов дубликатов (точные, семантические)

### 4.3. Добавить метрики качества данных

- Процент записей с ошибками
- Процент записей, требующих ручной проверки
- Распределение записей по категориям качества

### 4.4. Добавить временные метрики

- Оценка времени обработки на основе объема данных
- Прогноз скорости обработки
- Распределение времени по БД

## 5. Выводы

✅ **Совместимость типов:** Полная совместимость между фронтендом и бэкендом
⚠️ **Параметр normalizationType:** Не поддерживается, требуется доработка
✅ **Средства анализа:** Реализованы базовые метрики заполненности, подсчет записей и дубликатов
📊 **Потенциал:** Есть возможность расширения анализа данных

