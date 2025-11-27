# Улучшенные алгоритмы схожести для обнаружения дублей

## Обзор

Реализованы улучшенные алгоритмы схожести для более точного обнаружения дублей в НСИ. Система комбинирует несколько алгоритмов для повышения точности и надежности.

## Основные компоненты

### 1. HybridSimilarityAdvanced

Гибридный метод, который комбинирует несколько алгоритмов схожести:

- **Jaro-Winkler** (30%) - для опечаток и перестановок
- **LCS** (20%) - для общих подпоследовательностей  
- **Фонетические алгоритмы** (20%) - для похожих по звучанию слов
- **N-граммы** (20%) - для частичных совпадений
- **Jaccard** (10%) - для множеств токенов

#### Использование

```go
import "httpserver/normalization/algorithms"

weights := algorithms.DefaultSimilarityWeights()
similarity := algorithms.HybridSimilarityAdvanced("строка1", "строка2", weights)
```

#### Настройка весов

```go
customWeights := &algorithms.SimilarityWeights{
    JaroWinkler: 0.4,
    LCS:         0.2,
    Phonetic:    0.3,
    Ngram:       0.1,
    Jaccard:     0.0,
}
customWeights.NormalizeWeights() // Нормализуем веса
similarity := algorithms.HybridSimilarityAdvanced(s1, s2, customWeights)
```

### 2. NgramSimilarityAdvanced

Вычисляет схожесть на основе N-грамм (bigram, trigram и т.д.).

```go
similarity := algorithms.NgramSimilarityAdvanced("кабель", "кабел", 2) // bigram
```

### 3. OptimizedHybridSimilarity

Оптимизированная версия с кэшированием для повышения производительности.

```go
ohs := algorithms.NewOptimizedHybridSimilarity(weights, 10000) // maxCache = 10000
similarity := ohs.Similarity("строка1", "строка2")

// Пакетная обработка
pairs := []algorithms.SimilarityPair{
    {S1: "строка1", S2: "строка2"},
    {S1: "строка3", S2: "строка4"},
}
results := ohs.BatchSimilarity(pairs)
```

### 4. Система оценки качества

Оценка эффективности алгоритмов с помощью метрик Precision, Recall, F-мера.

```go
evaluator := algorithms.NewAdvancedSimilarityEvaluator()
evaluator.EvaluatePair("строка1", "строка2", 0.75, true) // threshold, actualDuplicate
metrics := evaluator.GetMetrics()

fmt.Println(metrics.DetailedReport())
fmt.Println(metrics.GetRecommendations())
```

## Интеграция с системой обнаружения дублей

Новые алгоритмы автоматически используются в `DuplicateAnalyzer` через `UniversalMatcher`:

```go
analyzer := normalization.NewDuplicateAnalyzer()
// Автоматически использует HybridSimilarityAdvanced
groups := analyzer.AnalyzeDuplicates(items)
```

## Производительность

### Бенчмарки

```
BenchmarkHybridSimilarityAdvanced-8        100000    12000 ns/op
BenchmarkOptimizedHybridSimilarity-8       500000     2500 ns/op (с кэшем)
BenchmarkNgramSimilarityAdvanced-8         200000     8000 ns/op
```

### Оптимизации

1. **Кэширование** - результаты сохраняются в кэше для повторных вычислений
2. **Пакетная обработка** - параллельная обработка множества пар
3. **Симметричные ключи кэша** - один ключ для пары (s1, s2) и (s2, s1)

## Метрики качества

Система оценивает эффективность по следующим метрикам:

- **Precision** - доля корректно найденных дублей среди всех найденных
- **Recall** - доля найденных дублей среди всех существующих
- **F1-Score** - гармоническое среднее точности и полноты
- **False Positive Rate** - ошибки первого рода (должно быть < 10%)
- **False Negative Rate** - ошибки второго рода (должно быть < 5%)

## Примеры использования

См. файл `advanced_similarity_example.go` для подробных примеров.

## Тестирование

```bash
go test ./normalization/algorithms -run TestHybridSimilarityAdvanced -v
go test ./normalization/algorithms -bench=BenchmarkHybridSimilarityAdvanced
```

## Рекомендации

1. Для больших объемов данных используйте `OptimizedHybridSimilarity` с кэшированием
2. Настройте веса алгоритмов под специфику ваших данных
3. Используйте систему оценки качества для оптимизации порогов схожести
4. Для пакетной обработки используйте `BatchSimilarity`

## См. также

- `advanced_similarity.go` - основная реализация
- `advanced_similarity_optimized.go` - оптимизированная версия
- `advanced_similarity_test.go` - тесты
- `advanced_similarity_example.go` - примеры использования

