# API для автоматического обнаружения дублей

## Обзор

API предоставляет эндпоинты для автоматического обнаружения дублей в базе данных с поддержкой асинхронной обработки и мониторинга прогресса.

## Эндпоинты

### 1. Запуск обнаружения дублей

**POST** `/api/duplicates/detect`

Запускает асинхронное обнаружение дублей в базе данных для указанного проекта.

#### Запрос

```json
{
  "project_id": 1,
  "threshold": 0.75,
  "batch_size": 100,
  "use_advanced": true,
  "weights": {
    "jaro_winkler": 0.3,
    "lcs": 0.2,
    "phonetic": 0.2,
    "ngram": 0.2,
    "jaccard": 0.1
  },
  "max_items": 1000
}
```

#### Параметры

- `project_id` (обязательный) - ID проекта для анализа
- `threshold` (опционально, по умолчанию 0.75) - Порог схожести (0-1)
- `batch_size` (опционально, по умолчанию 100) - Размер батча для обработки
- `use_advanced` (опционально, по умолчанию true) - Использовать продвинутые алгоритмы
- `weights` (опционально) - Веса алгоритмов
- `max_items` (опционально) - Максимальное количество записей для обработки (для тестирования)

#### Ответ

```json
{
  "task_id": "task_1705752345_1",
  "status": "started",
  "message": "Duplicate detection started"
}
```

### 2. Получение статуса задачи

**GET** `/api/duplicates/detect/{task_id}`

Получает текущий статус задачи обнаружения дублей.

#### Ответ

```json
{
  "task_id": "task_1705752345_1",
  "status": "running",
  "progress": 45,
  "total_items": 1000,
  "processed": 450,
  "found_groups": 23,
  "started_at": "2024-01-20T15:30:45Z",
  "completed_at": null
}
```

#### Статусы

- `running` - Задача выполняется
- `completed` - Задача завершена
- `failed` - Задача завершилась с ошибкой

### 3. Список всех задач

**GET** `/api/duplicates/detect`

Получает список всех задач обнаружения дублей.

#### Ответ

```json
{
  "tasks": [
    {
      "task_id": "task_1705752345_1",
      "status": "completed",
      "progress": 100,
      "total_items": 1000,
      "processed": 1000,
      "found_groups": 45,
      "started_at": "2024-01-20T15:30:45Z",
      "completed_at": "2024-01-20T15:32:10Z"
    }
  ],
  "count": 1
}
```

## Примеры использования

### Запуск обнаружения дублей

```bash
curl -X POST http://localhost:9999/api/duplicates/detect \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": 1,
    "threshold": 0.80,
    "batch_size": 200,
    "use_advanced": true
  }'
```

### Проверка статуса

```bash
curl http://localhost:9999/api/duplicates/detect/task_1705752345_1
```

### Получение списка задач

```bash
curl http://localhost:9999/api/duplicates/detect
```

### JavaScript (fetch)

```javascript
// Запуск обнаружения
const startResponse = await fetch('http://localhost:9999/api/duplicates/detect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    project_id: 1,
    threshold: 0.75,
    batch_size: 100,
    use_advanced: true
  })
});

const { task_id } = await startResponse.json();

// Мониторинг прогресса
const checkStatus = async () => {
  const response = await fetch(`http://localhost:9999/api/duplicates/detect/${task_id}`);
  const status = await response.json();
  
  console.log(`Progress: ${status.progress}%`);
  console.log(`Found groups: ${status.found_groups}`);
  
  if (status.status === 'running') {
    setTimeout(checkStatus, 2000); // Проверяем каждые 2 секунды
  } else if (status.status === 'completed') {
    console.log('Detection completed!');
  }
};

checkStatus();
```

## Рекомендации

### Производительность

1. **Batch Size**: Используйте batch_size 100-500 для оптимальной производительности
2. **Max Items**: Для тестирования используйте max_items, чтобы ограничить объем данных
3. **Advanced Methods**: Включите use_advanced для более точных результатов

### Пороги

- **0.90-1.0**: Очень строгий (только точные дубликаты)
- **0.75-0.90**: Рекомендуемый диапазон
- **0.60-0.75**: Более мягкий (может включать похожие записи)
- **< 0.60**: Очень мягкий (много ложных срабатываний)

### Мониторинг

- Проверяйте статус каждые 2-5 секунд
- Для больших объемов данных (10000+ записей) процесс может занять несколько минут
- Используйте batch_size для баланса между скоростью и использованием памяти

## Обработка ошибок

### Ошибки запуска

```json
{
  "error": "project_id is required"
}
```

### Ошибки выполнения

```json
{
  "task_id": "task_1705752345_1",
  "status": "failed",
  "error": "Normalized database not available",
  "progress": 0
}
```

## Интеграция с другими API

После завершения обнаружения дублей можно:

1. **Экспортировать результаты** через `/api/similarity/export`
2. **Оценить качество** через `/api/similarity/evaluate`
3. **Обучить алгоритм** на найденных дублях через `/api/similarity/learn`

## См. также

- [SIMILARITY_API.md](./SIMILARITY_API.md) - API для работы с алгоритмами схожести
- [SIMILARITY_ANALYSIS_GUIDE.md](./SIMILARITY_ANALYSIS_GUIDE.md) - Руководство по анализу
- [SIMILARITY_LEARNING_GUIDE.md](./SIMILARITY_LEARNING_GUIDE.md) - Руководство по обучению

