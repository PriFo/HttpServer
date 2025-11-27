# Pipeline нормализации наименований НСИ

Конфигурируемый pipeline для нормализации и сравнения наименований с использованием различных алгоритмов.

## Быстрый старт

```go
import "httpserver/normalization/pipeline_normalization"

// Создаем pipeline с конфигурацией по умолчанию
config := pipeline_normalization.NewDefaultConfig()
pipeline, err := pipeline_normalization.NewNormalizationPipeline(config)
if err != nil {
    log.Fatal(err)
}

// Сравниваем две строки
result, err := pipeline.Normalize("Кабель ВВГ 3x2.5", "Кабель ВВГ 3x2,5")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Схожесть: %.2f%%\n", result.Similarity.OverallSimilarity*100)
fmt.Printf("Дубликат: %v\n", result.Similarity.IsDuplicate)
fmt.Printf("Уверенность: %.2f%%\n", result.Similarity.Confidence*100)
```

## Предустановленные конфигурации

### Default (по умолчанию)
Сбалансированная конфигурация для большинства случаев:
- Дамерау-Левенштейн (вес 0.3)
- Jaccard (вес 0.2)
- N-граммы (вес 0.2)
- Косинусная близость (вес 0.15)
- Токен-ориентированные (вес 0.15)

```go
config := pipeline_normalization.NewDefaultConfig()
```

### Fast (быстрая)
Оптимизирована для скорости, использует меньше алгоритмов:
- Дамерау-Левенштейн (вес 0.5)
- Jaccard (вес 0.5)

```go
config := pipeline_normalization.NewFastConfig()
```

### Precise (точная)
Максимальная точность, использует все алгоритмы:
- Soundex
- Metaphone
- Дамерау-Левенштейн
- Jaccard с N-граммами
- N-граммы
- Косинусная близость
- Токен-ориентированные

```go
config := pipeline_normalization.NewPreciseConfig()
```

## Пользовательская конфигурация

```go
config := &pipeline_normalization.NormalizationPipelineConfig{
    Algorithms: []pipeline_normalization.AlgorithmConfig{
        {
            Type:      pipeline_normalization.AlgorithmDamerauLevenshtein,
            Enabled:   true,
            Weight:    0.6,
            Threshold: 0.85,
            Params:    make(map[string]interface{}),
        },
        {
            Type:      pipeline_normalization.AlgorithmJaccard,
            Enabled:   true,
            Weight:    0.4,
            Threshold: 0.75,
            Params: map[string]interface{}{
                "use_ngrams": true,
                "n_gram_size": 2,
            },
        },
    },
    MinSimilarity:     0.85,
    CombineMethod:     "weighted", // "weighted", "max", "min", "average"
    ParallelExecution: true,
    CacheEnabled:      true,
}

pipeline, _ := pipeline_normalization.NewNormalizationPipeline(config)
```

## Обработка батча

```go
pairs := [][]string{
    {"Кабель ВВГ 3x2.5", "Кабель ВВГ 3x2,5"},
    {"Труба ПВХ 20мм", "Труба ПВХ 20 мм"},
    {"Провод МГТФ", "Провод МГТФ 0.5"},
}

batchResult, err := pipeline.BatchNormalize(pairs)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Обработано: %d пар\n", batchResult.TotalProcessed)
fmt.Printf("Найдено дубликатов: %d\n", batchResult.DuplicatesFound)
fmt.Printf("Средняя схожесть: %.2f%%\n", batchResult.AverageSimilarity*100)
```

## Методы комбинирования результатов

- **weighted** - взвешенное среднее (по умолчанию)
- **max** - максимальное значение
- **min** - минимальное значение
- **average** - простое среднее

## Метрики качества

При включении `CalculateMetrics: true` pipeline вычисляет метрики качества:

- **Precision** - точность (доля найденных дублей, которые действительно дубли)
- **Recall** - полнота (доля всех дублей, которые были найдены)
- **F-measure** - гармоническое среднее точности и полноты

## Кэширование

Pipeline поддерживает кэширование результатов для повышения производительности:

```go
config.CacheEnabled = true
pipeline, _ := pipeline_normalization.NewNormalizationPipeline(config)

// Первый вызов - вычисление
result1, _ := pipeline.Normalize("текст1", "текст2")

// Второй вызов - из кэша
result2, _ := pipeline.Normalize("текст1", "текст2")

// Очистка кэша
pipeline.ClearCache()
```

## Параллельное выполнение

Для ускорения обработки можно включить параллельное выполнение алгоритмов:

```go
config.ParallelExecution = true
```

## Интеграция с DuplicateAnalyzer

```go
import (
    "httpserver/normalization"
    "httpserver/normalization/pipeline_normalization"
)

config := pipeline_normalization.NewDefaultConfig()
pipeline, _ := pipeline_normalization.NewNormalizationPipeline(config)

analyzer := normalization.NewDuplicateAnalyzerWithPipeline(pipeline)
groups := analyzer.AnalyzeDuplicates(items)
```

## Интеграция с Normalizer

```go
normalizer := normalization.NewNormalizer(db, events, aiConfig)
normalizer.SetNormalizationPipeline(pipeline)
```

