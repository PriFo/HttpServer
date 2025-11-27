# Тестирование бенчмарков (Эталонов)

## Обзор

E2E тесты для функциональности работы с эталонами (benchmarks) - эталонными данными, используемыми для нормализации.

## Файл тестов

`benchmarks.spec.ts` - содержит тесты для:
- Получения списка эталонов
- Поиска эталонов
- Создания эталонов из загрузок
- Получения эталона по ID
- Фильтрации эталонов по типу
- Проверки UI для бенчмарков

## API эндпоинты

### Основные эндпоинты

- `GET /api/benchmarks` - список эталонов
  - Параметры: `type`, `active`, `limit`, `offset`
- `GET /api/benchmarks/:id` - получение эталона по ID
- `POST /api/benchmarks` - создание эталона
- `PUT /api/benchmarks/:id` - обновление эталона
- `DELETE /api/benchmarks/:id` - удаление эталона
- `GET /api/benchmarks/search` - поиск эталонов
  - Параметры: `q`, `type`
- `POST /api/benchmarks/from-upload` - создание из загрузки
- `POST /api/benchmarks/import-manufacturers` - импорт производителей

### Эндпоинты для бенчмарков нормализации

- `POST /api/normalization/benchmark/upload` - загрузка бенчмарка нормализации
- `GET /api/normalization/benchmark/list` - список бенчмарков нормализации
- `GET /api/normalization/benchmark/:id` - получение бенчмарка нормализации

## Утилиты API

Добавлены функции в `utils/api-testing.ts`:

- `listBenchmarks(entityType?, activeOnly?, limit?, offset?)` - список эталонов
- `getBenchmarkById(benchmarkId)` - получение эталона по ID
- `searchBenchmarks(query, entityType?)` - поиск эталонов
- `createBenchmarkFromUpload(uploadId, entityType)` - создание из загрузки

## Использование

```typescript
import { 
  listBenchmarks, 
  getBenchmarkById, 
  searchBenchmarks 
} from '../../utils/api-testing'

// Получить список эталонов
const benchmarks = await listBenchmarks('counterparty', true)

// Найти эталон
const results = await searchBenchmarks('тест', 'counterparty')

// Получить эталон по ID
const benchmark = await getBenchmarkById('benchmark-id')
```

## Запуск тестов

```bash
# Все тесты бенчмарков
npx playwright test tests/e2e/benchmarks.spec.ts

# С UI
npx playwright test tests/e2e/benchmarks.spec.ts --ui

# В видимом режиме
npx playwright test tests/e2e/benchmarks.spec.ts --headed
```

## Структура эталона

```typescript
interface Benchmark {
  id: string
  entity_type: string // 'counterparty' | 'nomenclature'
  name: string
  data: Record<string, any>
  source_upload_id: string
  source_client_id: number
  is_active: boolean
  created_at: string
  updated_at: string
}
```

## Примечания

- Тесты используют API напрямую, так как UI для бенчмарков может быть не реализован
- Некоторые тесты могут быть пропущены, если тестовая БД не загружена
- Тесты не падают при недоступности API, только логируют предупреждения

