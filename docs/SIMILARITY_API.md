# API для работы с алгоритмами схожести

## Обзор

API предоставляет эндпоинты для работы с улучшенными алгоритмами схожести, включая гибридный метод, настройку весов и оценку качества.

## Эндпоинты

### Обучение и оптимизация

### 7. Обучение на размеченных данных

**POST** `/api/similarity/learn`

Обучает алгоритм на размеченных данных для оптимизации весов.

#### Запрос

```json
{
  "training_pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Рога и Копыта",
      "is_duplicate": true
    },
    {
      "s1": "ООО Рога и Копыта",
      "s2": "Рога и Копыта ООО",
      "is_duplicate": true
    },
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Другая Компания",
      "is_duplicate": false
    }
  ],
  "iterations": 100,
  "learning_rate": 0.01
}
```

#### Ответ

```json
{
  "weights": {
    "jaro_winkler": 0.32,
    "lcs": 0.18,
    "phonetic": 0.22,
    "ngram": 0.19,
    "jaccard": 0.09
  },
  "metrics": {
    "precision": 0.95,
    "recall": 0.90,
    "f1_score": 0.925,
    "accuracy": 0.93,
    "false_positive_rate": 0.05,
    "false_negative_rate": 0.10
  },
  "training_pairs_count": 3,
  "iterations": 100,
  "learning_rate": 0.01
}
```

### 8. Поиск оптимального порога

**POST** `/api/similarity/optimal-threshold`

Находит оптимальный порог схожести для заданных весов.

#### Запрос

```json
{
  "test_pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Рога и Копыта",
      "is_duplicate": true
    },
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Другая Компания",
      "is_duplicate": false
    }
  ],
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
  "optimal_threshold": 0.75,
  "metrics": {
    "precision": 0.95,
    "recall": 0.90,
    "f1_score": 0.925,
    "accuracy": 0.93,
    "false_positive_rate": 0.05,
    "false_negative_rate": 0.10
  },
  "weights": {
    "jaro_winkler": 0.3,
    "lcs": 0.2,
    "phonetic": 0.2,
    "ngram": 0.2,
    "jaccard": 0.1
  }
}
```

### 9. Кросс-валидация

**POST** `/api/similarity/cross-validate`

Выполняет кросс-валидацию на обучающих данных.

#### Запрос

```json
{
  "training_pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Рога и Копыта",
      "is_duplicate": true
    }
  ],
  "folds": 5
}
```

#### Ответ

```json
{
  "folds": 5,
  "average_metrics": {
    "precision": 0.94,
    "recall": 0.89,
    "f1_score": 0.915,
    "accuracy": 0.92,
    "false_positive_rate": 0.06,
    "false_negative_rate": 0.11
  },
  "fold_results": [
    {
      "fold": 1,
      "metrics": {
        "precision": 0.95,
        "recall": 0.90,
        "f1_score": 0.925,
        "accuracy": 0.93,
        "false_positive_rate": 0.05,
        "false_negative_rate": 0.10
      }
    }
  ],
  "training_pairs_count": 20
}
```

## Эндпоинты

### 1. Сравнение двух строк

**POST** `/api/similarity/compare`

Сравнивает две строки используя различные алгоритмы схожести.

#### Запрос

```json
{
  "string1": "ООО Рога и Копыта",
  "string2": "Рога и Копыта ООО",
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
  "string1": "ООО Рога и Копыта",
  "string2": "Рога и Копыта ООО",
  "results": {
    "hybrid": 0.87,
    "jaro_winkler": 0.92,
    "lcs": 0.85,
    "ngram_bigram": 0.78,
    "ngram_trigram": 0.82,
    "phonetic": 0.90,
    "phonetic_soundex": true,
    "phonetic_metaphone": false,
    "jaccard": 0.75,
    "weights": {
      "jaro_winkler": 0.3,
      "lcs": 0.2,
      "phonetic": 0.2,
      "ngram": 0.2,
      "jaccard": 0.1
    }
  }
}
```

### 2. Пакетное сравнение

**POST** `/api/similarity/batch`

Сравнивает множество пар строк одновременно с оптимизацией производительности.

#### Запрос

```json
{
  "pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "Рога и Копыта ООО"
    },
    {
      "s1": "Кабель ВВГнг 3x2.5",
      "s2": "Кабель ВВГ 3x2.5"
    }
  ],
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
  "results": [
    {
      "string1": "ООО Рога и Копыта",
      "string2": "Рога и Копыта ООО",
      "similarity": 0.87
    },
    {
      "string1": "Кабель ВВГнг 3x2.5",
      "string2": "Кабель ВВГ 3x2.5",
      "similarity": 0.92
    }
  ],
  "count": 2,
  "cache_size": 2
}
```

**Ограничения:**
- Максимум 1000 пар за один запрос

### 3. Управление весами

**GET** `/api/similarity/weights`

Получает веса по умолчанию и их описание.

#### Ответ

```json
{
  "weights": {
    "jaro_winkler": 0.3,
    "lcs": 0.2,
    "phonetic": 0.2,
    "ngram": 0.2,
    "jaccard": 0.1
  },
  "description": {
    "jaro_winkler": "Для опечаток и перестановок (рекомендуется: 0.2-0.4)",
    "lcs": "Для общих подпоследовательностей (рекомендуется: 0.1-0.3)",
    "phonetic": "Для похожих по звучанию слов (рекомендуется: 0.1-0.3)",
    "ngram": "Для частичных совпадений (рекомендуется: 0.1-0.3)",
    "jaccard": "Для множеств токенов (рекомендуется: 0.1-0.2)"
  }
}
```

**POST** `/api/similarity/weights`

Устанавливает пользовательские веса. Веса автоматически нормализуются.

#### Запрос

```json
{
  "weights": {
    "jaro_winkler": 0.5,
    "lcs": 0.3,
    "phonetic": 0.2,
    "ngram": 0.0,
    "jaccard": 0.0
  }
}
```

#### Ответ

```json
{
  "weights": {
    "jaro_winkler": 0.5,
    "lcs": 0.3,
    "phonetic": 0.2,
    "ngram": 0.0,
    "jaccard": 0.0
  },
  "normalized": true,
  "message": "Weights updated successfully"
}
```

### 4. Оценка качества алгоритма

**POST** `/api/similarity/evaluate`

Оценивает эффективность алгоритма на тестовых данных с известными результатами.

#### Запрос

```json
{
  "test_pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Рога и Копыта",
      "is_duplicate": true
    },
    {
      "s1": "ООО Рога и Копыта",
      "s2": "Рога и Копыта ООО",
      "is_duplicate": true
    },
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Другая Компания",
      "is_duplicate": false
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
  "metrics": {
    "precision": 0.95,
    "recall": 0.90,
    "f1_score": 0.925,
    "accuracy": 0.93,
    "false_positive_rate": 0.05,
    "false_negative_rate": 0.10
  },
  "counts": {
    "true_positives": 19,
    "false_positives": 1,
    "false_negatives": 2,
    "true_negatives": 78,
    "total": 100
  },
  "threshold": 0.75,
  "weights": {
    "jaro_winkler": 0.3,
    "lcs": 0.2,
    "phonetic": 0.2,
    "ngram": 0.2,
    "jaccard": 0.1
  },
  "acceptable": true,
  "recommendations": [
    "Метрики соответствуют требованиям. Система работает эффективно."
  ]
}
```

### 5. Статистика

**GET** `/api/similarity/stats`

Получает информацию о доступных алгоритмах и оптимизациях.

#### Ответ

```json
{
  "algorithms": [
    "hybrid_advanced",
    "jaro_winkler",
    "lcs",
    "ngram",
    "phonetic",
    "jaccard"
  ],
  "cache_size": 0,
  "optimizations": {
    "caching": true,
    "batch_processing": true,
    "parallel": true,
    "symmetric_keys": true
  },
  "default_weights": {
    "jaro_winkler": 0.3,
    "lcs": 0.2,
    "phonetic": 0.2,
    "ngram": 0.2,
    "jaccard": 0.1
  }
}
```

### 6. Очистка кэша

**POST** `/api/similarity/cache/clear`

Очищает кэш результатов вычислений.

#### Ответ

```json
{
  "message": "Cache cleared successfully"
}
```

## Примеры использования

### cURL

```bash
# Сравнение двух строк
curl -X POST http://localhost:9999/api/similarity/compare \
  -H "Content-Type: application/json" \
  -d '{
    "string1": "ООО Рога и Копыта",
    "string2": "Рога и Копыта ООО"
  }'

# Пакетное сравнение
curl -X POST http://localhost:9999/api/similarity/batch \
  -H "Content-Type: application/json" \
  -d '{
    "pairs": [
      {"s1": "Кабель ВВГнг", "s2": "Кабель ВВГ"},
      {"s1": "Провод ПВС", "s2": "Кабель ПВС"}
    ]
  }'

# Оценка качества
curl -X POST http://localhost:9999/api/similarity/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "test_pairs": [
      {"s1": "тест1", "s2": "тест1", "is_duplicate": true},
      {"s1": "тест1", "s2": "тест2", "is_duplicate": false}
    ],
    "threshold": 0.75
  }'
```

### JavaScript (fetch)

```javascript
// Сравнение строк
const response = await fetch('http://localhost:9999/api/similarity/compare', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    string1: 'ООО Рога и Копыта',
    string2: 'Рога и Копыта ООО'
  })
});
const data = await response.json();
console.log('Similarity:', data.results.hybrid);

// Пакетное сравнение
const batchResponse = await fetch('http://localhost:9999/api/similarity/batch', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    pairs: [
      { s1: 'Кабель ВВГнг', s2: 'Кабель ВВГ' },
      { s1: 'Провод ПВС', s2: 'Кабель ПВС' }
    ]
  })
});
const batchData = await batchResponse.json();
console.log('Results:', batchData.results);
```

## Рекомендации

1. **Для больших объемов данных** используйте `/api/similarity/batch` вместо множественных вызовов `/api/similarity/compare`
2. **Настройте веса** под специфику ваших данных через `/api/similarity/weights`
3. **Оцените качество** алгоритма на тестовых данных перед использованием в продакшене
4. **Используйте кэширование** - результаты автоматически кэшируются для повторных запросов
5. **Мониторьте метрики** - регулярно проверяйте Precision, Recall и F1-score

## Ограничения

- Максимум 1000 пар за один запрос в `/api/similarity/batch`
- Порог схожести должен быть между 0 и 1
- Все веса должны быть между 0 и 1

## Коды ошибок

- `400 Bad Request` - неверный формат запроса или отсутствуют обязательные поля
- `405 Method Not Allowed` - неверный HTTP метод
- `500 Internal Server Error` - внутренняя ошибка сервера

