# Поток данных номенклатуры

## Обзор

Документ описывает полный поток данных номенклатуры от frontend до backend и обратно.

## Архитектура

```
Frontend Component (nomenclature-tab.tsx)
    ↓
Frontend API Route (/api/clients/[clientId]/nomenclature/route.ts)
    ↓
Backend Handler (handleGetClientNomenclature / handleGetProjectNomenclature)
    ↓
Data Sources:
  - Нормализованная БД (normalized_data.db)
  - Основные БД проектов (project databases)
    ↓
Объединение и дедупликация (mergeNomenclatureResults)
    ↓
Пагинация
    ↓
JSON Response
    ↓
Frontend Component (отображение в таблице)
```

## 1. Frontend Component

**Файл**: `frontend/app/clients/[clientId]/components/nomenclature-tab.tsx`

### Запрос данных

```typescript
// Для клиента
GET /api/clients/${clientId}/nomenclature?page=${page}&limit=${limit}&search=${search}

// Для проекта
GET /api/clients/${clientId}/projects/${projectId}/nomenclature?page=${page}&limit=${limit}&search=${search}
```

### Ожидаемый формат ответа

```typescript
{
  items: NomenclatureItem[],
  total: number,
  page: number,
  limit: number
}
```

### Интерфейс NomenclatureItem

```typescript
interface NomenclatureItem {
  id: number
  code: string
  name: string
  normalized_name: string
  category: string
  quality_score: number
  status: string
  merged_count: number
  kpved_code?: string
  kpved_name?: string
  source_database?: string
  source_type?: string
  project_id?: number
  project_name?: string
}
```

## 2. Frontend API Routes

### Клиентская номенклатура

**Файл**: `frontend/app/api/clients/[clientId]/nomenclature/route.ts`

- Проксирует запрос к backend
- Передает параметры: `page`, `limit`, `search`
- Обрабатывает ошибки

### Номенклатура проекта

**Файл**: `frontend/app/api/clients/[clientId]/projects/[projectId]/nomenclature/route.ts`

- Аналогично клиентской номенклатуре
- Добавляет `projectId` в URL

## 3. Backend Handlers

**Файл**: `server/client_legacy_handlers.go`

### handleGetClientNomenclature

**Endpoint**: `GET /api/clients/{id}/nomenclature`

**Логика**:
1. Получает все проекты клиента
2. Собирает данные из:
   - Нормализованной БД (`getNomenclatureFromNormalizedDB`)
   - Основных БД всех проектов (`getNomenclatureFromMainDB`)
3. Объединяет результаты (`mergeNomenclatureResults`)
4. Применяет пагинацию
5. Возвращает JSON

### handleGetProjectNomenclature

**Endpoint**: `GET /api/clients/{id}/projects/{projectId}/nomenclature`

**Логика**:
1. Проверяет существование проекта
2. Собирает данные из:
   - Нормализованной БД для проекта
   - Основных БД проекта
3. Объединяет результаты
4. Применяет пагинацию
5. Возвращает JSON

## 4. Получение данных из БД

### getNomenclatureFromNormalizedDB

**Источник**: `normalized_data.db`

**Таблица**: `normalized_nomenclature`

**Поля**:
- `id`, `code`, `normalized_name`, `category`
- `quality_score`, `kpved_code`, `kpved_name`
- `source_reference`, `source_name`
- `ai_confidence`, `ai_reasoning`, `processing_level`
- `merged_count`, `project_id`

**SourceType**: `"normalized"`

### getNomenclatureFromMainDB

**Источник**: Базы данных проектов (SQLite файлы)

**Таблицы**:
- `nomenclature_items` (если есть)
- `counterparty_items` (если есть, преобразуется в номенклатуру)

**Поля**:
- `nomenclature_code`, `nomenclature_name`, `nomenclature_reference`
- `upload_id` (для связи с проектом)

**SourceType**: `"main"`

## 5. Объединение и дедупликация

**Функция**: `mergeNomenclatureResults`

**Алгоритм**:
1. Дедупликация по ключу: `code + "|" + normalized_name`
2. Приоритет нормализованной БД над основной
3. Если ключ пустой, используется: `id + "|" + sourceType + "|" + sourceDatabase`
4. Применяется пагинация после дедупликации

## 6. Формат ответа

```json
{
  "items": [
    {
      "id": 1,
      "code": "CODE001",
      "name": "Название",
      "normalized_name": "Нормализованное название",
      "category": "Категория",
      "quality_score": 0.95,
      "status": "active",
      "merged_count": 2,
      "source_database": "path/to/db.db",
      "source_type": "normalized",
      "project_id": 1,
      "project_name": "Название проекта",
      "kpved_code": "25.11.11",
      "kpved_name": "Название КПВЭД",
      "ai_confidence": 0.9,
      "ai_reasoning": "Обоснование",
      "processing_level": "advanced"
    }
  ],
  "total": 100,
  "page": 1,
  "limit": 20
}
```

## 7. Отображение во Frontend

### Компонент NomenclatureTab

**Функциональность**:
- Таблица с сортировкой
- Поиск (debounce 500ms)
- Фильтрация по типу источника
- Пагинация
- Детальный просмотр (диалог)

### Колонки таблицы

1. Код
2. Название
3. Нормализованное название
4. Категория
5. КПВЭД (код и название)
6. Качество (бейдж)
7. Объединено (количество)
8. Источник (тип и база данных)
9. Действия (просмотр деталей)

## Потенциальные проблемы

### 1. Производительность

- При больших объемах данных дедупликация может быть медленной
- Используется большой лимит (100000) для получения всех записей перед дедупликацией

**Рекомендации**:
- Добавить индексы на `code` и `normalized_name`
- Кэшировать результаты дедупликации
- Применять пагинацию на уровне БД с последующей дедупликацией только на границах страниц

### 2. Обработка ошибок

- Если нормализованная БД недоступна, продолжается работа с основными БД
- Если основная БД недоступна, она пропускается, работа продолжается

### 3. Формат данных

- Frontend ожидает поля: `items`, `total`, `page`, `limit`
- Backend возвращает именно этот формат
- ✅ Соответствие подтверждено

## Проверка потока

### ✅ Проверено

1. Frontend API routes корректно проксируют запросы
2. Backend handlers корректно обрабатывают запросы
3. Формат ответа соответствует ожиданиям frontend
4. Дедупликация работает корректно
5. Пагинация применяется правильно
6. Отображение данных в таблице работает

### ⚠️ Требует внимания

1. Производительность при больших объемах данных
2. Обработка ошибок (логирование, но не возврат ошибок пользователю)
3. Кэширование результатов

## Заключение

Поток данных номенклатуры реализован корректно. Все компоненты работают согласованно:
- Frontend правильно запрашивает данные
- Backend корректно обрабатывает запросы и возвращает данные
- Формат данных соответствует ожиданиям
- Отображение работает корректно

Основная область для улучшения - оптимизация производительности при работе с большими объемами данных.

