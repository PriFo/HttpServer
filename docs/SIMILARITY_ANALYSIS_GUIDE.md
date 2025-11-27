# Руководство по анализу схожести

## Обзор

Система анализа схожести предоставляет инструменты для детального анализа результатов сравнения строк, включая разбивку по алгоритмам, статистику и рекомендации.

## Новые API эндпоинты

### 1. Детальный анализ пар

**POST** `/api/similarity/analyze`

Анализирует множество пар строк с детальной разбивкой по алгоритмам и статистикой.

#### Запрос

```json
{
  "pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "Рога и Копыта ООО"
    },
    {
      "s1": "Кабель ВВГнг",
      "s2": "Кабель ВВГ"
    }
  ],
  "threshold": 0.75,
  "weights": {
    "jaro_winkler": 0.3,
    "lcs": 0.2,
    "phonetic": 0.2,
    "ngram": 0.2,
    "jaccard": 0.1
  }
}
```

#### Ответ

```json
{
  "pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "Рога и Копыта ООО",
      "similarity": 0.87,
      "is_duplicate": true,
      "confidence": 0.76,
      "breakdown": {
        "jaro_winkler": 0.92,
        "lcs": 0.85,
        "phonetic": 0.90,
        "ngram": 0.78,
        "jaccard": 0.75
      }
    }
  ],
  "statistics": {
    "total_pairs": 2,
    "duplicate_pairs": 2,
    "non_duplicate_pairs": 0,
    "average_similarity": 0.89,
    "min_similarity": 0.87,
    "max_similarity": 0.91,
    "median_similarity": 0.89
  },
  "recommendations": [
    "Статистика выглядит нормально. Система работает эффективно."
  ]
}
```

### 2. Поиск похожих пар

**POST** `/api/similarity/find-similar`

Находит пары с схожестью выше порога, отсортированные по убыванию схожести.

#### Запрос

```json
{
  "pairs": [
    {"s1": "ООО Рога и Копыта", "s2": "Рога и Копыта ООО"},
    {"s1": "Кабель ВВГнг", "s2": "Провод ПВС"}
  ],
  "threshold": 0.75,
  "limit": 10
}
```

#### Ответ

```json
{
  "similar_pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "Рога и Копыта ООО",
      "similarity": 0.87,
      "is_duplicate": true,
      "confidence": 0.76,
      "breakdown": {...}
    }
  ],
  "count": 1,
  "threshold": 0.75
}
```

### 3. Сравнение весов

**POST** `/api/similarity/compare-weights`

Сравнивает эффективность разных наборов весов на тестовых данных.

#### Запрос

```json
{
  "test_pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Рога и Копыта",
      "is_duplicate": true
    }
  ],
  "weights": [
    {
      "jaro_winkler": 0.3,
      "lcs": 0.2,
      "phonetic": 0.2,
      "ngram": 0.2,
      "jaccard": 0.1
    },
    {
      "jaro_winkler": 0.5,
      "lcs": 0.3,
      "phonetic": 0.2,
      "ngram": 0.0,
      "jaccard": 0.0
    }
  ],
  "threshold": 0.75
}
```

#### Ответ

```json
{
  "comparisons": [
    {
      "weights": {...},
      "metrics": {
        "precision": 0.95,
        "recall": 0.90,
        "f1_score": 0.925
      },
      "f1_score": 0.925,
      "precision": 0.95,
      "recall": 0.90
    }
  ],
  "best_weights": {...},
  "best_f1_score": 0.925
}
```

### 4. Разбивка по алгоритмам

**POST** `/api/similarity/breakdown`

Получает детальную разбивку схожести по каждому алгоритму.

#### Запрос

```json
{
  "string1": "ООО Рога и Копыта",
  "string2": "Рога и Копыта ООО"
}
```

#### Ответ

```json
{
  "string1": "ООО Рога и Копыта",
  "string2": "Рога и Копыта ООО",
  "hybrid": 0.87,
  "breakdown": {
    "jaro_winkler": 0.92,
    "lcs": 0.85,
    "phonetic": 0.90,
    "ngram": 0.78,
    "jaccard": 0.75
  },
  "contribution": {
    "jaro_winkler": 0.276,
    "lcs": 0.17,
    "phonetic": 0.18,
    "ngram": 0.156,
    "jaccard": 0.075
  },
  "weights": {...}
}
```

### 5. Метрики производительности

**GET** `/api/similarity/performance`

Получает метрики производительности системы.

#### Ответ

```json
{
  "total_requests": 1000,
  "total_time_ms": 5000,
  "average_time_ms": 5.0,
  "cache_hits": 750,
  "cache_misses": 250,
  "cache_hit_rate": 75.0,
  "batch_requests": 50,
  "total_pairs": 5000,
  "average_pairs_per_batch": 100,
  "cache_size": 500
}
```

**POST** `/api/similarity/performance/reset`

Сбрасывает метрики производительности.

## Использование

### Пример: Анализ данных контрагентов

```javascript
const pairs = [
  { s1: "ООО Рога и Копыта", s2: "Рога и Копыта ООО" },
  { s1: "ООО Рога и Копыта", s2: "ООО Другая Компания" },
];

const response = await fetch('http://localhost:9999/api/similarity/analyze', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    pairs: pairs,
    threshold: 0.75
  })
});

const result = await response.json();
console.log('Statistics:', result.statistics);
console.log('Recommendations:', result.recommendations);
```

### Пример: Поиск похожих записей

```javascript
const response = await fetch('http://localhost:9999/api/similarity/find-similar', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    pairs: allPairs,
    threshold: 0.80,
    limit: 20
  })
});

const result = await response.json();
// result.similar_pairs содержит топ-20 самых похожих пар
```

### Пример: Сравнение весов

```javascript
const weightsToCompare = [
  { jaro_winkler: 0.3, lcs: 0.2, phonetic: 0.2, ngram: 0.2, jaccard: 0.1 },
  { jaro_winkler: 0.5, lcs: 0.3, phonetic: 0.2, ngram: 0.0, jaccard: 0.0 },
];

const response = await fetch('http://localhost:9999/api/similarity/compare-weights', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    test_pairs: testData,
    weights: weightsToCompare,
    threshold: 0.75
  })
});

const result = await response.json();
console.log('Best weights:', result.best_weights);
console.log('Best F1-score:', result.best_f1_score);
```

## Интерпретация результатов

### Разбивка по алгоритмам

- **Jaro-Winkler высокий**: Много опечаток и перестановок в данных
- **LCS высокий**: Много общих подпоследовательностей
- **Phonetic высокий**: Много похожих по звучанию слов
- **N-gram высокий**: Много частичных совпадений
- **Jaccard высокий**: Много общих токенов

### Уверенность (Confidence)

- **> 0.8**: Высокая уверенность в результате
- **0.5-0.8**: Средняя уверенность
- **< 0.5**: Низкая уверенность, требуется проверка

### Рекомендации

Система автоматически генерирует рекомендации на основе статистики:
- Предложения по настройке порога
- Рекомендации по балансу данных
- Предупреждения о проблемах с данными

## См. также

- [SIMILARITY_API.md](./SIMILARITY_API.md) - Полная документация API
- [SIMILARITY_LEARNING_GUIDE.md](./SIMILARITY_LEARNING_GUIDE.md) - Руководство по обучению

