# Быстрый старт: Использование алгоритмов нормализации НСИ

## Простейшие примеры

### 1. Базовое сходство строк

```go
import "httpserver/normalization/algorithms"

// Расстояние Левенштейна
similarity := algorithms.LevenshteinSimilarity("молоток", "молотак")
// Результат: ~0.71

// Индекс Жаккара
jaccard := algorithms.JaccardIndex("кабель медный", "кабель")
// Результат: ~0.5

// Jaro-Winkler (хорош для коротких строк)
jaro := algorithms.JaroWinklerSimilarity("кот", "котенок")
// Результат: ~0.78
```

### 2. N-граммы

```go
// Символьные биграммы
bigramSim := algorithms.CharacterNGramSimilarity("молоток", "молотак", 2)

// Словесные триграммы
trigramSim := algorithms.WordNGramSimilarity("кабель медный", "кабель", 3)

// Комбинированные N-граммы
combined := algorithms.CombinedNGramSimilarity("строка1", "строка2", nil)
```

### 3. Фонетическое сходство

```go
// Soundex
soundex := algorithms.NewSoundex()
code := soundex.Encode("молоток")
similarity := soundex.Similarity("молоток", "молотак")

// Или через функцию-обертку
similarity := algorithms.PhoneticSimilarity("молоток", "молотак", "soundex")
```

### 4. Универсальный матчер

```go
import "httpserver/normalization"

// Создание матчера
matcher := normalization.NewUniversalMatcher(true)

// Один метод
sim, _ := matcher.Similarity("строка1", "строка2", "levenshtein")

// Несколько методов
results, _ := matcher.SimilarityMultiple("строка1", "строка2", 
    []string{"levenshtein", "jaccard", "jaro_winkler"})

// Гибридное сходство
hybrid, _ := matcher.HybridSimilarity("строка1", "строка2",
    []string{"levenshtein", "jaccard"},
    []float64{0.6, 0.4})
```

### 5. Автоматический выбор метода

```go
selector := normalization.NewMethodSelector(matcher)

// Автоматический выбор
method, methods, _ := selector.SelectMethod("строка1", "строка2")

// Рекомендация гибридного метода
methods, weights, _ := selector.RecommendHybridMethod("строка1", "строка2")
```

### 6. Оценка алгоритмов

```go
import "httpserver/normalization/evaluation"

// Создание размеченных данных
pairs := []evaluation.LabeledPair{
    {Item1: "молоток", Item2: "молотак", IsDuplicate: true},
    {Item1: "кабель", Item2: "провод", IsDuplicate: false},
}

// Оценка
evaluator := evaluation.NewAlgorithmEvaluator(pairs, 0.85)
result := evaluator.Evaluate("levenshtein", 
    algorithms.LevenshteinSimilarity)

// Отчет
report := evaluation.GenerateHTMLReport([]evaluation.EvaluationResult{result})
```

## Рекомендации по выбору метода

| Ситуация | Рекомендуемый метод |
|----------|---------------------|
| Короткие строки (< 10 символов) | Jaro-Winkler |
| Длинные строки (> 50 символов) | Jaccard, LCS |
| Ожидаются опечатки | Damerau-Levenshtein |
| Фонетическое сходство | Soundex, Metaphone |
| Вариации порядка слов | Jaccard, Dice |
| Максимальная точность | Ensemble методы |

## Полный список доступных методов

### Метрики сходства
- `levenshtein` - Расстояние Левенштейна
- `damerau_levenshtein` - Дамерау-Левенштейн
- `jaro` - Jaro
- `jaro_winkler` - Jaro-Winkler
- `jaccard` - Индекс Жаккара
- `dice` - Коэффициент Dice
- `lcs` - Наибольшая общая подпоследовательность
- `hamming` - Расстояние Хэмминга

### N-граммы
- `char_bigram` - Символьные биграммы
- `char_trigram` - Символьные триграммы
- `word_bigram` - Словесные биграммы
- `word_trigram` - Словесные триграммы
- `ngram` - Комбинированные N-граммы

### Фонетические
- `soundex` - Soundex
- `metaphone` - Metaphone
- `phonetic` - Комбинированный фонетический

## Примеры использования в DuplicateAnalyzer

```go
analyzer := normalization.NewDuplicateAnalyzer()

// Включить продвинутые методы
analyzer.EnableAdvancedMethods(true)

// Анализ дубликатов
groups := analyzer.AnalyzeDuplicates(items)
```

## Дополнительная информация

- Полная документация: `normalization/ALGORITHMS_IMPLEMENTATION.md`
- Описание алгоритмов: `normalization/algorithms/README.md`
- Статус реализации: `normalization/IMPLEMENTATION_COMPLETE.md`

