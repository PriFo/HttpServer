# Интеграция расширенных алгоритмов сходства

## Добавленные алгоритмы

В файл `advanced_similarity.go` добавлены три новых алгоритма вычисления сходства строк:

1. **JaroSimilarityAdvanced** - алгоритм Jaro для вычисления сходства строк
2. **JaroWinklerSimilarityAdvanced** - улучшенная версия Jaro с учетом общего префикса
3. **LCSSimilarityAdvanced** - сходство на основе наибольшей общей подпоследовательности (LCS)

## Интеграция в Pipeline

Алгоритмы интегрированы в систему нормализации через pipeline:

### Новые типы алгоритмов

В `config.go` добавлены новые типы:
- `AlgorithmJaro` - алгоритм Jaro
- `AlgorithmJaroWinkler` - алгоритм Jaro-Winkler  
- `AlgorithmLCS` - алгоритм на основе LCS

### Executors

Созданы executors для каждого алгоритма:
- `JaroExecutor` - выполняет `JaroSimilarityAdvanced`
- `JaroWinklerExecutor` - выполняет `JaroWinklerSimilarityAdvanced`
- `LCSExecutor` - выполняет `LCSSimilarityAdvanced`

## Использование

### В конфигурации pipeline

```go
config := &NormalizationPipelineConfig{
    Algorithms: []AlgorithmConfig{
        {
            Type:      AlgorithmJaro,
            Enabled:   true,
            Weight:    0.2,
            Threshold: 0.85,
            Params:    make(map[string]interface{}),
        },
        {
            Type:      AlgorithmJaroWinkler,
            Enabled:   true,
            Weight:    0.2,
            Threshold: 0.85,
            Params:    make(map[string]interface{}),
        },
        {
            Type:      AlgorithmLCS,
            Enabled:   true,
            Weight:    0.15,
            Threshold: 0.80,
            Params:    make(map[string]interface{}),
        },
    },
    // ... другие настройки
}
```

### Прямое использование

```go
import "httpserver/normalization/algorithms"

// Jaro
similarity := algorithms.JaroSimilarityAdvanced("кабель", "кабел")

// Jaro-Winkler
similarity := algorithms.JaroWinklerSimilarityAdvanced("молоток", "молот")

// LCS
similarity := algorithms.LCSSimilarityAdvanced("кабель медный", "кабель")
```

## Тесты

Создан файл `advanced_similarity_test.go` с тестами для всех трех алгоритмов:
- Unit тесты для каждого алгоритма
- Тесты граничных случаев (пустые строки, идентичные строки и т.д.)
- Benchmark тесты для оценки производительности

## Особенности алгоритмов

### Jaro
- Учитывает совпадения символов в пределах окна
- Учитывает транспозиции
- Хорошо работает для коротких строк

### Jaro-Winkler
- Расширение Jaro с учетом общего префикса
- Дает бонус строкам с общим началом
- Эффективен для имен и названий

### LCS
- Основан на наибольшей общей подпоследовательности
- Устойчив к перестановкам символов
- Хорошо работает для длинных строк

## Производительность

Все алгоритмы оптимизированы для работы с Unicode (используют `[]rune` вместо `[]byte`) и обрабатывают регистр и пробелы автоматически.

