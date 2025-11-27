# Проверка функциональности просмотра контрагентов по клиенту

**Дата**: 2025-01-21  
**Статус**: ✅ **РЕАЛИЗОВАНО**

---

## ✅ Реализованная функциональность

### 1. API Endpoints

#### `/api/counterparties/normalized`
- **Метод**: GET
- **Параметры**:
  - `client_id` (обязательный) - ID клиента
  - `project_id` (опциональный) - ID проекта для фильтрации
  - `page` - номер страницы (по умолчанию 1)
  - `limit` - количество записей на странице (по умолчанию 100, максимум 1000)
  - `offset` - смещение (альтернатива page)
  - `search` - поисковый запрос (по имени, ИНН, БИН)
  - `enrichment` - фильтр по источнику обогащения
  - `subcategory` - фильтр по подкатегории

**Пример запроса**:
```
GET /api/counterparties/normalized?client_id=1&page=1&limit=20&search=ООО
```

**Ответ**:
```json
{
  "counterparties": [...],
  "projects": [...],
  "total": 100,
  "offset": 0,
  "limit": 20,
  "page": 1
}
```

#### `/api/counterparties/all`
- **Метод**: GET
- **Параметры**:
  - `client_id` (обязательный) - ID клиента
  - `project_id` (опциональный) - ID проекта для фильтрации
  - `offset` - смещение (по умолчанию 0)
  - `limit` - количество записей (по умолчанию 100, максимум 1000)
  - `search` - поисковый запрос
  - `source` - фильтр по источнику: "database", "normalized" или пусто (все)
  - `sort_by` - поле сортировки: "name", "quality", "source", "id"
  - `order` - порядок сортировки: "asc", "desc"
  - `min_quality` - минимальное качество (0.0 - 1.0)
  - `max_quality` - максимальное качество (0.0 - 1.0)

**Пример запроса**:
```
GET /api/counterparties/all?client_id=1&limit=50&source=normalized&sort_by=quality&order=desc
```

**Ответ**:
```json
{
  "counterparties": [...],
  "projects": [...],
  "total": 150,
  "offset": 0,
  "limit": 50,
  "stats": {
    "total_from_database": 50,
    "total_normalized": 100,
    "total_with_quality": 80,
    "average_quality": 0.85,
    "databases_processed": 3,
    "projects_processed": 2,
    "processing_time_ms": 150
  }
}
```

#### `/api/counterparties/all/export`
- **Метод**: GET
- **Параметры**: те же, что и для `/api/counterparties/all`
- **Дополнительно**: `format` - формат экспорта: "csv" или "json"
- **Возвращает**: файл для скачивания

---

### 2. Функции базы данных

#### `GetNormalizedCounterpartiesByClient`
```go
func (db *ServiceDB) GetNormalizedCounterpartiesByClient(
    clientID int, 
    projectID *int, 
    offset, limit int, 
    search, enrichment, subcategory string
) ([]*NormalizedCounterparty, []*ClientProject, int, error)
```

**Функциональность**:
- Получает все проекты клиента (или конкретный проект)
- Извлекает нормализованных контрагентов из всех проектов
- Поддерживает поиск, фильтрацию и пагинацию
- Возвращает список контрагентов, проектов и общее количество

#### `GetAllCounterpartiesByClient`
```go
func (db *ServiceDB) GetAllCounterpartiesByClient(
    clientID int, 
    projectID *int, 
    offset, limit int, 
    search, source, sortBy, order string, 
    minQuality, maxQuality *float64
) (*GetAllCounterpartiesByClientResult, error)
```

**Функциональность**:
- Получает контрагентов из двух источников:
  1. Исходные базы данных (source="database")
  2. Нормализованные записи (source="normalized")
- Объединяет данные из обоих источников
- Применяет фильтры по качеству
- Сортирует результаты
- Применяет пагинацию
- Возвращает статистику обработки

**Особенности**:
- Параллельная обработка баз данных (до 5 одновременных подключений)
- Автоматическое извлечение данных из атрибутов (ИНН, КПП, БИН, адреса, контакты)
- Поддержка фильтрации по качеству данных
- Гибкая сортировка

---

### 3. Фронтенд компоненты

#### `CounterpartiesTab`
**Расположение**: `frontend/app/clients/[clientId]/components/counterparties-tab.tsx`

**Функциональность**:
- Отображение списка контрагентов клиента
- Поиск по имени, ИНН, БИН
- Фильтрация по проекту
- Пагинация
- Сортировка
- Отображение детальной информации

**Используемые API**:
- `/api/counterparties/normalized?client_id={clientId}&project_id={projectId}&page={page}&limit={limit}&search={search}`

---

## 📊 Структура данных

### UnifiedCounterparty
```go
type UnifiedCounterparty struct {
    ID              int     `json:"id"`
    Name            string  `json:"name"`
    Source          string  `json:"source"` // "database" или "normalized"
    ProjectID       int     `json:"project_id"`
    ProjectName     string  `json:"project_name"`
    DatabaseID      *int    `json:"database_id,omitempty"`
    DatabaseName    string  `json:"database_name,omitempty"`
    NormalizedName  string  `json:"normalized_name,omitempty"`
    SourceName      string  `json:"source_name,omitempty"`
    SourceReference string  `json:"source_reference,omitempty"`
    TaxID           string  `json:"tax_id,omitempty"`
    KPP             string  `json:"kpp,omitempty"`
    BIN             string  `json:"bin,omitempty"`
    LegalAddress    string  `json:"legal_address,omitempty"`
    PostalAddress   string  `json:"postal_address,omitempty"`
    ContactPhone    string  `json:"contact_phone,omitempty"`
    ContactEmail    string  `json:"contact_email,omitempty"`
    ContactPerson   string  `json:"contact_person,omitempty"`
    QualityScore    *float64 `json:"quality_score,omitempty"`
    Reference       string  `json:"reference,omitempty"`
    Code            string  `json:"code,omitempty"`
    Attributes      map[string]interface{} `json:"attributes,omitempty"`
}
```

### GetAllCounterpartiesByClientResult
```go
type GetAllCounterpartiesByClientResult struct {
    Counterparties []*UnifiedCounterparty
    Projects       []*ClientProject
    TotalCount     int
    Stats          *CounterpartiesStats
}
```

### CounterpartiesStats
```go
type CounterpartiesStats struct {
    TotalFromDatabase  int     `json:"total_from_database"`
    TotalNormalized    int     `json:"total_normalized"`
    TotalWithQuality   int     `json:"total_with_quality"`
    AverageQuality     float64 `json:"average_quality,omitempty"`
    DatabasesProcessed int     `json:"databases_processed,omitempty"`
    ProjectsProcessed  int     `json:"projects_processed,omitempty"`
    ProcessingTimeMs   int64   `json:"processing_time_ms,omitempty"`
}
```

---

## ✅ Проверка функциональности

### Реализованные возможности

1. ✅ **Просмотр контрагентов по клиенту**
   - Получение всех контрагентов клиента из всех проектов
   - Фильтрация по конкретному проекту

2. ✅ **Поиск контрагентов**
   - По имени
   - По ИНН
   - По БИН
   - По нормализованному имени

3. ✅ **Фильтрация**
   - По источнику (database/normalized)
   - По качеству данных (min_quality, max_quality)
   - По проекту
   - По источнику обогащения
   - По подкатегории

4. ✅ **Сортировка**
   - По имени
   - По качеству
   - По источнику
   - По ID
   - По умолчанию (качество -> имя -> ID)

5. ✅ **Пагинация**
   - Поддержка offset/limit
   - Поддержка page/limit
   - Максимум 1000 записей за запрос

6. ✅ **Экспорт**
   - CSV формат
   - JSON формат
   - Экспорт всех контрагентов клиента

7. ✅ **Статистика**
   - Общее количество из баз данных
   - Общее количество нормализованных
   - Количество с оценкой качества
   - Среднее качество
   - Время обработки

---

## 🔍 Примеры использования

### 1. Получить все контрагенты клиента
```bash
curl "http://localhost:8080/api/counterparties/all?client_id=1&limit=100"
```

### 2. Получить контрагенты с фильтрацией по качеству
```bash
curl "http://localhost:8080/api/counterparties/all?client_id=1&min_quality=0.8&sort_by=quality&order=desc"
```

### 3. Поиск контрагентов
```bash
curl "http://localhost:8080/api/counterparties/all?client_id=1&search=ООО&limit=50"
```

### 4. Получить только нормализованные контрагенты
```bash
curl "http://localhost:8080/api/counterparties/all?client_id=1&source=normalized"
```

### 5. Экспорт в CSV
```bash
curl "http://localhost:8080/api/counterparties/all/export?client_id=1&format=csv" -o counterparties.csv
```

---

## ✅ Итоговое заключение

**Функциональность просмотра контрагентов по клиенту полностью реализована и работает!**

### Реализовано:
- ✅ API endpoints для получения контрагентов
- ✅ Функции базы данных для извлечения данных
- ✅ Фронтенд компоненты для отображения
- ✅ Поиск и фильтрация
- ✅ Сортировка и пагинация
- ✅ Экспорт данных
- ✅ Статистика

### Доступные endpoints:
1. `/api/counterparties/normalized` - нормализованные контрагенты
2. `/api/counterparties/all` - все контрагенты (из баз и нормализованных)
3. `/api/counterparties/all/export` - экспорт контрагентов

**Статус**: ✅ **ГОТОВО К ИСПОЛЬЗОВАНИЮ**

---

**Дата проверки**: 2025-01-21

