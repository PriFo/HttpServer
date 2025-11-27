# Комплексная система нормализации наименований НСИ

## Обзор

Реализована комплексная система нормализации наименований с конфигурируемым pipeline, поддерживающая все основные алгоритмические методы для автоматического обнаружения дублей в НСИ.

## Архитектура

### Модуль алгоритмов (`normalization/algorithms/`)

Содержит реализации всех алгоритмов нормализации:

1. **Soundex (RU)** - `soundex_ru.go`
   - Фонетическое кодирование для русского языка
   - Быстрое сравнение похожих по звучанию слов

2. **Metaphone (RU)** - `metaphone_ru.go`
   - Улучшенный фонетический алгоритм
   - Учитывает контекст и позицию звуков

3. **Индекс Жаккара** - `jaccard.go`
   - Сравнение множеств токенов или N-грамм
   - Поддержка взвешенного индекса Жаккара

4. **N-граммы** - интегрировано с `ngram.go`
   - Биграммы, триграммы
   - Настраиваемый размер N-грамм

5. **Дамерау-Левенштейн** - `damerau_levenshtein.go`
   - Расстояние с учетом транспозиций
   - Оптимизированная реализация

6. **Косинусная близость** - `cosine_similarity.go`
   - TF-IDF векторизация
   - Бинарные и частотные векторы
   - Поддержка N-грамм

7. **Токен-ориентированные методы** - `token_based.go`
   - Сравнение по общим словам
   - Взвешенное сравнение
   - Позиционное сравнение

### Конфигурируемый Pipeline (`normalization/pipeline_normalization/`)

1. **Конфигурация** - `config.go`
   - Настройка алгоритмов
   - Пороги срабатывания
   - Веса алгоритмов
   - Методы комбинирования

2. **Pipeline процессор** - `pipeline.go`
   - Последовательное и параллельное выполнение
   - Кэширование результатов
   - Обработка батчей

3. **Результаты и метрики** - `results.go`
   - Структуры результатов
   - Метрики качества (precision, recall, F-measure)
   - Детализация по алгоритмам

### Интеграция

1. **Algorithm Selector** - `algorithm_selector.go`
   - Фабрика алгоритмов
   - Предустановленные конфигурации
   - Рекомендации по выбору

2. **Интеграция с DuplicateAnalyzer**
   - Поддержка pipeline в анализаторе дубликатов
   - Метод `findDuplicatesWithPipeline`

3. **Интеграция с Normalizer**
   - Методы `SetNormalizationPipeline` и `GetNormalizationPipeline`

## Использование

### Базовый пример

```go
package main

import (
    "fmt"
    "log"
    
    "httpserver/normalization/pipeline_normalization"
)

func main() {
    // Создаем конфигурацию
    config := pipeline_normalization.NewDefaultConfig()
    
    // Создаем pipeline
    pipeline, err := pipeline_normalization.NewNormalizationPipeline(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Сравниваем строки
    result, err := pipeline.Normalize(
        "Кабель ВВГ 3x2.5мм",
        "Кабель ВВГ 3x2,5 мм",
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Схожесть: %.2f%%\n", result.Similarity.OverallSimilarity*100)
    fmt.Printf("Дубликат: %v\n", result.Similarity.IsDuplicate)
    fmt.Printf("Уверенность: %.2f%%\n", result.Similarity.Confidence*100)
    
    // Детализация по алгоритмам
    for alg, score := range result.Similarity.AlgorithmScores {
        fmt.Printf("  %s: %.2f%%\n", alg, score*100)
    }
}
```

### Использование с DuplicateAnalyzer

```go
package main

import (
    "httpserver/normalization"
    "httpserver/normalization/pipeline_normalization"
)

func main() {
    // Создаем pipeline
    config := pipeline_normalization.NewDefaultConfig()
    pipeline, _ := pipeline_normalization.NewNormalizationPipeline(config)
    
    // Создаем анализатор с pipeline
    analyzer := normalization.NewDuplicateAnalyzerWithPipeline(pipeline)
    
    // Анализируем дубликаты
    items := []normalization.DuplicateItem{
        {ID: 1, NormalizedName: "Кабель ВВГ 3x2.5"},
        {ID: 2, NormalizedName: "Кабель ВВГ 3x2,5"},
        {ID: 3, NormalizedName: "Провод МГТФ"},
    }
    
    groups := analyzer.AnalyzeDuplicates(items)
    
    for _, group := range groups {
        fmt.Printf("Группа: %s, Схожесть: %.2f, Элементов: %d\n",
            group.GroupID, group.SimilarityScore, len(group.Items))
    }
}
```

### Использование Algorithm Selector

```go
package main

import (
    "httpserver/normalization"
    "httpserver/normalization/pipeline_normalization"
)

func main() {
    selector := normalization.NewAlgorithmSelector()
    
    // Создаем pipeline с предустановленной конфигурацией
    pipeline, _ := selector.CreatePipeline("fast")
    
    // Или создаем с пользовательскими требованиями
    requirements := normalization.PipelineRequirements{
        Speed:           0.7,
        Accuracy:        0.3,
        MinSimilarity:   0.85,
        UsePhonetic:     true,
        UseNGrams:       true,
    }
    
    config := selector.RecommendConfiguration(requirements)
    pipeline, _ = pipeline_normalization.NewNormalizationPipeline(config)
}
```

## Предустановленные конфигурации

### Default (по умолчанию)
- **Назначение**: Сбалансированная конфигурация
- **Алгоритмы**: Дамерау-Левенштейн, Jaccard, N-граммы, Косинусная близость, Токен-ориентированные
- **Скорость**: Средняя
- **Точность**: Высокая

### Fast (быстрая)
- **Назначение**: Максимальная скорость
- **Алгоритмы**: Дамерау-Левенштейн, Jaccard
- **Скорость**: Высокая
- **Точность**: Средняя

### Precise (точная)
- **Назначение**: Максимальная точность
- **Алгоритмы**: Все доступные алгоритмы
- **Скорость**: Низкая
- **Точность**: Очень высокая

## Метрики качества

Система поддерживает вычисление метрик качества:

- **Precision** (Точность) - доля найденных дублей, которые действительно дубли
- **Recall** (Полнота) - доля всех дублей, которые были найдены
- **F-measure** (F-мера) - гармоническое среднее точности и полноты

```go
config.CalculateMetrics = true
config.PrecisionWeight = 0.5
config.RecallWeight = 0.5
```

## Производительность

### Оптимизации

1. **Кэширование** - результаты сравнения кэшируются
2. **Параллельное выполнение** - алгоритмы могут выполняться параллельно
3. **Батчевая обработка** - обработка множества пар за один вызов

### Рекомендации

- Для больших объемов данных используйте конфигурацию `Fast`
- Для критически важных данных используйте конфигурацию `Precise`
- Включите кэширование для повторяющихся сравнений
- Используйте параллельное выполнение на многоядерных системах

## Тестирование

Все алгоритмы покрыты unit-тестами:

```bash
go test ./normalization/algorithms/...
go test ./normalization/pipeline_normalization/...
```

## Документация

- [README алгоритмов](algorithms/README.md) - описание всех алгоритмов
- [README pipeline](pipeline_normalization/README.md) - использование pipeline
- [Примеры использования](pipeline_normalization/example_usage.go) - примеры кода

## Оценка эффективности

Система позволяет оценивать эффективность алгоритмов с помощью стандартных метрик:

- **Точность (Precision)** - должна быть высокой для минимизации ложных срабатываний
- **Полнота (Recall)** - должна быть высокой для минимизации пропущенных дублей
- **F-мера** - сбалансированная оценка

Ошибки первого рода (ложная тревога) не должны превышать 10%.
Ошибки второго рода (недостаточная бдительность) не должны превышать 5%.

