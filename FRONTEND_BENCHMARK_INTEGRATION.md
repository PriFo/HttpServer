# Интеграция фронтенда с бенчмарками нормализации

## Обзор

Полная интеграция фронтенда для работы с бенчмарками нормализации контрагентов. Позволяет загружать, просматривать и анализировать результаты производительности системы нормализации.

## Компоненты

### 1. Frontend API Route
**Файл:** `frontend/app/api/normalization/benchmark/route.ts`

Проксирует запросы к бэкенду:
- `GET /api/normalization/benchmark?list=true` - получение списка бенчмарков
- `GET /api/normalization/benchmark?id={id}` - получение конкретного бенчмарка
- `POST /api/normalization/benchmark` - загрузка JSON файла с результатами

### 2. Frontend Page
**Файл:** `frontend/app/normalization/benchmark/page.tsx`

Основная страница для работы с бенчмарками:
- Загрузка JSON файлов
- Просмотр списка сохраненных бенчмарков
- Визуализация результатов (таблицы, графики)
- Экспорт результатов

### 3. Backend Handlers
**Файл:** `server/normalization_benchmark_handlers.go`

Обработчики на бэкенде:
- `handleNormalizationBenchmarkUpload` - загрузка и сохранение бенчмарка
- `handleNormalizationBenchmarkList` - получение списка бенчмарков
- `handleNormalizationBenchmarkGet` - получение конкретного бенчмарка

## API Endpoints

### POST /api/normalization/benchmark/upload
Загружает результаты бенчмарка.

**Request Body (JSON):**
```json
{
  "timestamp": "2025-01-20T12:00:00Z",
  "test_name": "Normalization Performance Benchmark",
  "record_count": 1000,
  "duplicate_rate": 0.2,
  "workers": 10,
  "results": [...],
  "total_duration_ms": 6700,
  "average_speed_records_per_sec": 149.25,
  "summary": {...}
}
```

**Response:**
```json
{
  "success": true,
  "message": "Benchmark uploaded successfully",
  "id": "20250120_120000",
  "file": "normalization_benchmark_20250120_120000.json",
  "path": "./benchmarks/normalization_benchmark_20250120_120000.json"
}
```

### GET /api/normalization/benchmark/list
Возвращает список всех сохраненных бенчмарков.

**Response:**
```json
{
  "benchmarks": [
    {
      "id": "2025-01-20T12:00:00Z",
      "filename": "normalization_benchmark_20250120_120000.json",
      "timestamp": "2025-01-20T12:00:00Z",
      "test_name": "Normalization Performance Benchmark",
      "record_count": 1000,
      "duplicate_rate": 0.2,
      "workers": 10,
      "average_speed": 149.25,
      "total_duration_ms": 6700,
      "file_size": 12345,
      "created_at": "2025-01-20T12:00:00Z"
    }
  ],
  "total": 1
}
```

### GET /api/normalization/benchmark/{id}
Возвращает конкретный бенчмарк по ID (timestamp).

**Response:**
Полный объект `NormalizationBenchmarkReport`

## Использование

### Загрузка бенчмарка через фронтенд

1. Откройте страницу `/normalization/benchmark`
2. Нажмите "Загрузить файл" или перетащите JSON файл
3. Выберите файл с результатами бенчмарка (созданный `test_normalization_benchmark.go`)
4. Результаты автоматически отобразятся

### Просмотр сохраненных бенчмарков

1. На странице `/normalization/benchmark` в правой колонке отображается список сохраненных бенчмарков
2. Кликните на бенчмарк для загрузки
3. Используйте кнопку обновления для обновления списка

### Анализ результатов

Страница предоставляет три вкладки:

1. **Результаты** - таблица с метриками по каждому этапу
2. **Графики** - визуализация:
   - Время выполнения по этапам
   - Скорость обработки
   - Использование памяти
   - Распределение времени
3. **Сводка** - детальная информация и экспорт

### Экспорт результатов

1. Откройте вкладку "Сводка"
2. Нажмите кнопку "Экспорт JSON"
3. Файл будет скачан с результатами

## Навигация

Ссылка на страницу бенчмарка добавлена в главное меню:
- **Процессы** → **Бенчмарк нормализации**

## Структура данных

### BenchmarkReport
```typescript
interface BenchmarkReport {
  timestamp: string
  test_name: string
  record_count: number
  duplicate_rate: number
  workers: number
  results: BenchmarkResult[]
  total_duration_ms: number
  average_speed_records_per_sec: number
  summary: Record<string, any>
}
```

### BenchmarkResult
```typescript
interface BenchmarkResult {
  stage: string
  record_count: number
  duration_ms: number
  records_per_second: number
  memory_used_mb?: number
  duplicate_groups?: number
  total_duplicates?: number
  processed_count?: number
  benchmark_matches?: number
  enriched_count?: number
  created_benchmarks?: number
  error_count?: number
  stopped?: boolean
}
```

## Валидация

Backend выполняет валидацию:
- Проверка наличия результатов
- Проверка корректности значений (record_count > 0, duplicate_rate 0-1, workers > 0)
- Проверка структуры каждого результата
- Проверка доступности директории для сохранения

## Хранение

Бенчмарки сохраняются в директории `./benchmarks/` с именами:
```
normalization_benchmark_YYYYMMDD_HHMMSS.json
```

## Обработка ошибок

- Валидация файлов на фронтенде (только JSON)
- Валидация структуры данных на бэкенде
- Обработка ошибок сети
- Fallback при недоступности бэкенда
- Информативные сообщения об ошибках

## Примеры использования

### Загрузка через curl

```bash
curl -X POST http://localhost:8080/api/normalization/benchmark/upload \
  -H "Content-Type: application/json" \
  -d @test_benchmark_example.json
```

### Получение списка

```bash
curl http://localhost:8080/api/normalization/benchmark/list
```

### Получение конкретного бенчмарка

```bash
curl http://localhost:8080/api/normalization/benchmark/2025-01-20T12:00:00Z
```

## Интеграция с тестами

Бенчмарки создаются утилитой `test_normalization_benchmark.go`:

```bash
go run test_normalization_benchmark.go -records 5000 -workers 20
```

Результаты сохраняются в JSON файл, который можно загрузить через фронтенд.

## Дальнейшие улучшения

- [ ] Сравнение нескольких бенчмарков
- [ ] Фильтрация и сортировка списка бенчмарков
- [ ] Удаление бенчмарков
- [ ] Экспорт в другие форматы (CSV, PDF)
- [ ] Автоматическое обновление списка
- [ ] Поиск по бенчмаркам
- [ ] Теги и категории бенчмарков

