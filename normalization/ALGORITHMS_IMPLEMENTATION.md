# Реализация всех методик нормализации наименований НСИ

## Обзор

Реализован полный набор алгоритмических методов для нормализации и сравнения наименований нормативно-справочной информации (НСИ). Система включает универсальный матчер, метод селектор, модуль оценки и интеграцию с существующей системой.

## Структура модулей

### 1. `normalization/algorithms/` - Алгоритмические методы

#### Метрики сходства (`similarity_metrics.go`)
- **LevenshteinSimilarity** - Расстояние Левенштейна (редакционное расстояние)
- **DamerauLevenshteinSimilarity** - Расстояние Дамерау-Левенштейна (с учетом транспозиций)
- **JaroSimilarity** - Сходство Jaro
- **JaroWinklerSimilarity** - Сходство Jaro-Winkler (учитывает префиксы)
- **JaccardIndex** - Индекс Жаккара (для множеств токенов)
- **DiceCoefficient** - Коэффициент Dice (Sørensen-Dice)
- **LCSSimilarity** - Сходство на основе наибольшей общей подпоследовательности
- **HammingSimilarity** - Расстояние Хэмминга (для строк одинаковой длины)

#### N-граммы (`ngrams.go`)
- **NGramSimilarity** - Сходство на основе N-грамм
- **CharacterNGramSimilarity** - Символьные N-граммы
- **WordNGramSimilarity** - Словесные N-граммы
- **CombinedNGramSimilarity** - Комбинированное сходство

#### Фонетические алгоритмы (`phonetic.go`)
- **Soundex** - Soundex для русского языка (структура)
- **Metaphone** - Metaphone для русского языка (структура)
- **PhoneticMatcher** - Комбинированный фонетический матчер
- **RussianSoundex()** - Функция-обертка для обратной совместимости
- **RussianMetaphone()** - Функция-обертка для обратной совместимости
- **PhoneticSimilarity()** - Фонетическое сходство с выбором метода

#### Векторные методы (`vectorization.go`)
- **TFIDFVectorizer** - TF-IDF векторизация
- **CharacterNGramVectorizer** - Векторизация символьных N-грамм
- **BagOfWords** - Мешок слов (BoW)
- **CosineSimilarityVectors** - Косинусное сходство векторов
- **EuclideanDistance** - Евклидово расстояние
- **NormalizeVector** - Нормализация векторов (L2 норма)

#### Комбинированные методы (`hybrid_matcher.go`)
- **HybridMatcher** - Гибридный матчер с взвешенным сходством
- **EnsembleMatcher** - Ансамблевый матчер с различными стратегиями голосования
- **AdaptiveThresholdMatcher** - Матчер с адаптивными порогами
- **WeightedSimilarity** - Взвешенное сходство нескольких методов
- **GetDefaultMethods()** - Набор методов по умолчанию

#### Нормализация текста (`text_normalizer.go`)
- **TextNormalizer** - Полная нормализация текста
- **Transliterate** - Транслитерация кириллица ↔ латиница
- **RemovePunctuation** - Удаление знаков пунктуации
- **NormalizeWhitespace** - Нормализация пробельных символов

### 2. `normalization/evaluation/` - Модуль оценки

#### Метрики (`metrics.go`)
- **ConfusionMatrix** - Матрица ошибок
- **Precision** - Точность (TP / (TP + FP))
- **Recall** - Полнота (TP / (TP + FN))
- **F1Score** - F1-мера (гармоническое среднее)
- **FBetaScore** - F-бета мера (настраиваемый beta)
- **Accuracy** - Точность классификации
- **Specificity** - Специфичность
- **FalsePositiveRate** - Частота ложных срабатываний
- **FalseNegativeRate** - Частота пропусков

#### Оценщик алгоритмов (`algorithm_evaluator.go`)
- **AlgorithmEvaluator** - Оценка алгоритмов на размеченных данных
- **Evaluate** - Оценка с фиксированным порогом
- **EvaluateWithAdaptiveThreshold** - Оценка с адаптивным порогом
- **FindOptimalThreshold** - Поиск оптимального порога
- **GenerateReport** - Генерация текстового отчета
- **GenerateHTMLReport** - Генерация HTML отчета

### 3. Универсальный матчер (`universal_matcher.go`)

**UniversalMatcher** - Единый интерфейс для всех методов нормализации:
- Регистрация методов
- Вычисление сходства одним или несколькими методами
- Гибридное и ансамблевое сходство
- Кэширование результатов
- Список доступных методов

### 4. Метод селектор (`method_selector.go`)

**MethodSelector** - Автоматический выбор оптимального метода:
- Анализ характеристик данных
- Рекомендации на основе характеристик
- Гибридные методы с весами
- Рекомендации для наборов данных

### 5. Интеграция с DuplicateAnalyzer

Обновлен `DuplicateAnalyzer` для использования новых методов:
- Поддержка универсального матчера
- Использование продвинутых методов в семантическом и фонетическом поиске
- Автоматический выбор методов через селектор

## Примеры использования

### Базовое использование

```go
import "httpserver/normalization/algorithms"

// Простое сходство
similarity := algorithms.LevenshteinSimilarity("молоток", "молотак")

// N-граммы
ngramSim := algorithms.NGramSimilarity("кабель", "кабел", 2)

// Фонетическое сходство
phoneticSim := algorithms.PhoneticSimilarity("молоток", "молотак", "soundex")
```

### Использование универсального матчера

```go
import "httpserver/normalization"

// Создание матчера
matcher := normalization.NewUniversalMatcher(true) // с кэшем

// Вычисление сходства одним методом
similarity, err := matcher.Similarity("строка1", "строка2", "levenshtein")

// Вычисление сходства несколькими методами
results, err := matcher.SimilarityMultiple("строка1", "строка2", 
    []string{"levenshtein", "jaccard", "jaro_winkler"})

// Гибридное сходство
hybridSim, err := matcher.HybridSimilarity("строка1", "строка2",
    []string{"levenshtein", "jaccard"},
    []float64{0.6, 0.4})
```

### Использование метод селектора

```go
selector := normalization.NewMethodSelector(matcher)

// Автоматический выбор метода
method, methods, err := selector.SelectMethod("строка1", "строка2")

// Рекомендация гибридного метода
methods, weights, err := selector.RecommendHybridMethod("строка1", "строка2")
```

### Использование модуля оценки

```go
import "httpserver/normalization/evaluation"

// Создание размеченных данных
pairs := []evaluation.LabeledPair{
    {Item1: "молоток", Item2: "молотак", IsDuplicate: true},
    {Item1: "кабель", Item2: "провод", IsDuplicate: false},
}

// Оценка алгоритма
evaluator := evaluation.NewAlgorithmEvaluator(pairs, 0.85)
result := evaluator.Evaluate("levenshtein", 
    func(s1, s2 string) float64 {
        return algorithms.LevenshteinSimilarity(s1, s2)
    })

// Генерация отчета
report := evaluation.GenerateReport([]evaluation.EvaluationResult{result})
```

## Рекомендации по выбору метода

### По длине текста
- **Короткие тексты (< 10 символов)**: Jaro-Winkler, Levenshtein
- **Средние тексты (10-50 символов)**: Levenshtein, Jaccard
- **Длинные тексты (> 50 символов)**: Jaccard, LCS, N-граммы

### По типу вариаций
- **Опечатки**: Damerau-Levenshtein, Jaro-Winkler
- **Фонетическое сходство**: Soundex, Metaphone
- **Вариации порядка слов**: Jaccard, Dice
- **Вариации символов**: Character N-граммы
- **Семантическое сходство**: TF-IDF, косинусное сходство

### Комбинированные подходы
- **Ensemble** - для максимальной точности
- **Hybrid** - для баланса точности и производительности
- **Adaptive Threshold** - для разных длин строк

## Производительность

Все алгоритмы оптимизированы для работы с большими объемами данных:
- Кэширование результатов
- Оптимизированные структуры данных
- Параллельная обработка (где применимо)
- Батчевая обработка

## Метрики оценки

Система поддерживает полный набор метрик для оценки качества:
- **Precision** - доля корректных дублей среди найденных
- **Recall** - доля найденных дублей среди всех существующих
- **F1-Score** - гармоническое среднее точности и полноты
- **F2-Score** - F-мера с большим весом для полноты
- **Accuracy** - общая точность классификации
- **False Positive Rate** - частота ложных срабатываний
- **False Negative Rate** - частота пропусков

## Интеграция

Все методы интегрированы в существующую систему:
- `DuplicateAnalyzer` использует новые методы
- Поддержка обратной совместимости
- Автоматический выбор оптимальных методов
- Расширяемая архитектура

## Тестирование

Созданы базовые unit-тесты для проверки работоспособности:
- Тесты метрик сходства
- Тесты N-грамм
- Тесты фонетических алгоритмов
- Интеграционные тесты

## Документация

- `normalization/algorithms/README.md` - Описание алгоритмов
- `normalization/ALGORITHMS_IMPLEMENTATION.md` - Этот файл
- Примеры использования в коде

## Статус реализации

✅ Все запланированные компоненты реализованы:
- ✅ Метрики сходства
- ✅ N-граммы
- ✅ Фонетические алгоритмы
- ✅ Векторные методы
- ✅ Комбинированные методы
- ✅ Модуль оценки
- ✅ Универсальный матчер
- ✅ Метод селектор
- ✅ Интеграция
- ✅ Тесты
- ✅ Документация

Система готова к использованию!

