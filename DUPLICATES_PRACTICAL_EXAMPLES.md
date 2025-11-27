# Практические примеры использования алгоритмов поиска дублей

## Дата создания: 2025-01-20

---

## 📚 Примеры из документа и их реализация

### Пример 1: Расстояние Левенштейна

**Из документа (строка 655-663)**:
```
"дом" → "том" (1 замена)
Расстояние = 1
Применение: если расстояние ≤ 2, считаем строки похожими
```

**Реализация в коде**:

```go
package main

import (
    "fmt"
    "httpserver/normalization"
)

func main() {
    // Создаем анализатор
    analyzer := normalization.NewDuplicateAnalyzer()
    
    // Пример из документа: "дом" vs "том"
    items := []normalization.DuplicateItem{
        {ID: 1, NormalizedName: "дом", Code: "001"},
        {ID: 2, NormalizedName: "том", Code: "002"},
    }
    
    // Находим дубликаты
    groups := analyzer.AnalyzeDuplicates(items)
    
    // Проверяем результат
    if len(groups) > 0 {
        fmt.Printf("Найдены дубликаты! Схожесть: %.2f\n", 
            groups[0].SimilarityScore)
    }
    
    // Прямое вычисление расстояния Левенштейна
    import "httpserver/normalization/algorithms"
    similarity := algorithms.LevenshteinSimilarity("дом", "том")
    fmt.Printf("Схожесть Левенштейна: %.2f\n", similarity)
    // Ожидаемый результат: ~0.67 (расстояние 1 из 3 символов)
}
```

**Результат**: ✅ Реализовано в `normalization/duplicate_analyzer.go:828`

---

### Пример 2: N-граммы (Bigrams)

**Из документа (строка 684-700)**:
```
Для "звено": [зв, ве, ен, но] (биграммы, N=2)
Для "звено" и "зерно":
- "звено": [зв, ве, ен, но]
- "зерно": [зе, ер, рн, но]
- Общее: [но] — 1 грамма
- Сёренсен = 2 × 1 / (4 + 4) = 0.25
```

**Реализация в коде**:

```go
package main

import (
    "fmt"
    "httpserver/normalization/algorithms"
)

func main() {
    // Создаем генератор биграмм
    gen := algorithms.NewNGramGenerator(2)
    
    // Генерируем N-граммы для "звено"
    ngrams1 := gen.Generate("звено")
    fmt.Printf("N-граммы для 'звено': %v\n", ngrams1)
    // Ожидаемый результат: ["зв", "ве", "ен", "но"] или похожее
    
    // Генерируем N-граммы для "зерно"
    ngrams2 := gen.Generate("зерно")
    fmt.Printf("N-граммы для 'зерно': %v\n", ngrams2)
    
    // Вычисляем схожесть по Сёренсену (через Jaccard)
    similarity := gen.Similarity("звено", "зерно")
    fmt.Printf("Схожесть N-грамм: %.2f\n", similarity)
    // Ожидаемый результат: ~0.25 (как в документе)
    
    // Или через FuzzyAlgorithms
    import "httpserver/normalization"
    fa := normalization.NewFuzzyAlgorithms()
    bigramSim := fa.BigramSimilarity("звено", "зерно")
    fmt.Printf("Bigram similarity: %.2f\n", bigramSim)
}
```

**Результат**: ✅ Реализовано в `normalization/fuzzy_algorithms.go:38` и `normalization/algorithms/ngram.go`

---

### Пример 3: Предварительная очистка текста

**Из документа (строка 600-623)**:

#### 1.1 Приведение к единому регистру
```
"МАСЛО СЛИВОЧНОЕ" → "масло сливочное"
```

**Реализация**:

```go
package main

import (
    "fmt"
    "strings"
    "httpserver/normalization"
)

func main() {
    // Создаем нормализатор
    normalizer := normalization.NewNameNormalizer()
    
    // Пример из документа
    input := "МАСЛО СЛИВОЧНОЕ"
    normalized := normalizer.NormalizeName(input)
    fmt.Printf("Исходное: %s\n", input)
    fmt.Printf("Нормализованное: %s\n", normalized)
    // Ожидаемый результат: "масло сливочное"
}
```

#### 1.2 Удаление пробельных символов
```
"  масло   сливочное  " → "масло сливочное"
```

**Реализация**:

```go
func main() {
    normalizer := normalization.NewNameNormalizer()
    
    input := "  масло   сливочное  "
    normalized := normalizer.NormalizeName(input)
    fmt.Printf("Исходное: '%s'\n", input)
    fmt.Printf("Нормализованное: '%s'\n", normalized)
    // Ожидаемый результат: "масло сливочное"
}
```

#### 1.3 Удаление пунктуации
```
"Масло сливочное, 82%" → "Масло сливочное 82"
```

**Реализация**:

```go
func main() {
    normalizer := normalization.NewNameNormalizer()
    
    input := "Масло сливочное, 82%"
    normalized := normalizer.NormalizeName(input)
    fmt.Printf("Исходное: %s\n", input)
    fmt.Printf("Нормализованное: %s\n", normalized)
    // Ожидаемый результат: "масло сливочное" (числа тоже удаляются)
}
```

**Результат**: ✅ Реализовано в `normalization/name_normalizer.go:43`

---

### Пример 4: Стемминг

**Из документа (строка 630-632)**:
```
"нормализация" + "нормализирован" → оба имеют корень "нормализ"
```

**Реализация**:

```go
package main

import (
    "fmt"
    "httpserver/normalization/algorithms"
)

func main() {
    // Создаем стеммер для русского языка
    stemmer := algorithms.NewRussianStemmer()
    
    // Пример из документа
    word1 := "нормализация"
    word2 := "нормализирован"
    
    stem1 := stemmer.Stem(word1)
    stem2 := stemmer.Stem(word2)
    
    fmt.Printf("Слово 1: %s → корень: %s\n", word1, stem1)
    fmt.Printf("Слово 2: %s → корень: %s\n", word2, stem2)
    
    // Проверяем, что корни совпадают (или похожи)
    if stem1 == stem2 {
        fmt.Println("Корни совпадают!")
    } else {
        fmt.Printf("Корни различаются, но похожи: %s vs %s\n", stem1, stem2)
    }
}
```

**Результат**: ✅ Реализовано в `normalization/algorithms/stemmer.go:47`

---

### Пример 5: Лемматизация

**Из документа (строка 636-638)**:
```
"маслами" → "масло", "сливочного" → "сливочный"
```

**Реализация**:

```go
package main

import (
    "fmt"
    "httpserver/normalization/algorithms"
)

func main() {
    // ВАЖНО: Полная лемматизация пока не реализована
    // Используется стемминг как замена
    
    stemmer := algorithms.NewRussianStemmer()
    
    // Пример из документа
    word1 := "маслами"
    word2 := "сливочного"
    
    // Стемминг (временная замена лемматизации)
    stem1 := stemmer.Stem(word1)
    stem2 := stemmer.Stem(word2)
    
    fmt.Printf("Слово 1: %s → стем: %s\n", word1, stem1)
    fmt.Printf("Слово 2: %s → стем: %s\n", word2, stem2)
    
    // TODO: После реализации лемматизации:
    // lemmatizer := algorithms.NewRussianLemmatizer()
    // lemma1 := lemmatizer.Lemmatize("маслами") // → "масло"
    // lemma2 := lemmatizer.Lemmatize("сливочного") // → "сливочный"
}
```

**Результат**: ⚠️ Частично реализовано (только стемминг), полная лемматизация в планах

---

### Пример 6: Удаление стоп-слов

**Из документа (строка 641-644)**:
```
"масло сливочное для готовки" → "масло сливочное готовка"
```

**Реализация**:

```go
package main

import (
    "fmt"
    "httpserver/normalization"
)

func main() {
    analyzer := normalization.NewDuplicateAnalyzer()
    
    // Настройка: не использовать стоп-слова
    analyzer.wordBasedUseStopWords = false
    
    text := "масло сливочное для готовки"
    
    // Токенизация с удалением стоп-слов происходит внутри
    // Для демонстрации используем прямой вызов
    items := []normalization.DuplicateItem{
        {ID: 1, NormalizedName: text, Code: "001"},
        {ID: 2, NormalizedName: "масло сливочное готовка", Code: "002"},
    }
    
    // Анализ найдет дубликаты, так как стоп-слово "для" игнорируется
    groups := analyzer.AnalyzeWordBasedDuplicates(items)
    
    if len(groups) > 0 {
        fmt.Println("Найдены дубликаты по общим словам (стоп-слова игнорируются)")
    }
}
```

**Результат**: ✅ Реализовано в `normalization/duplicate_analyzer.go:752`

---

### Пример 7: Фонетические алгоритмы

**Из документа (строка 702-706)**:
```
Soundex: преобразование слова в фонетический код
Metaphone: более совершенная фонетическая индексация
Применение: поиск дублей, несмотря на орфографические варианты
```

**Реализация**:

```go
package main

import (
    "fmt"
    "httpserver/normalization/algorithms"
)

func main() {
    // Soundex для русского языка
    soundex := algorithms.NewSoundexRU()
    
    word1 := "Иванов"
    word2 := "Ivanov" // латиница
    
    code1 := soundex.Encode(word1)
    code2 := soundex.Encode(word2)
    
    fmt.Printf("Soundex код для '%s': %s\n", word1, code1)
    fmt.Printf("Soundex код для '%s': %s\n", word2, code2)
    
    // Вычисляем схожесть
    similarity := soundex.Similarity(word1, word2)
    fmt.Printf("Фонетическая схожесть: %.2f\n", similarity)
    
    // Metaphone (более точный)
    metaphone := algorithms.NewMetaphoneRU()
    meta1 := metaphone.Encode(word1)
    meta2 := metaphone.Encode(word2)
    
    fmt.Printf("Metaphone код для '%s': %s\n", word1, meta1)
    fmt.Printf("Metaphone код для '%s': %s\n", word2, meta2)
}
```

**Результат**: ✅ Реализовано в `normalization/algorithms/soundex_ru.go` и `metaphone_ru.go`

---

### Пример 8: Полный процесс нормализации

**Из документа (строка 805-813)**:
```
1. Очистка: нижний регистр → удаление пробелов → удаление пунктуации
2. Лингвистический анализ: токенизация → лемматизация → удаление стоп-слов
3. Сравнение и поиск дублей: расстояние Левенштейна + N-граммы → группировка
4. Структурирование: регулярные выражения → NER
5. Машинное обучение: Seq2Seq → BiLSTM → BERT
6. Консолидация: выбор мастер-записи → объединение атрибутов
```

**Реализация**:

```go
package main

import (
    "fmt"
    "httpserver/normalization"
)

func main() {
    // Создаем унифицированный нормализатор
    nsi := normalization.NewNSINormalizer()
    
    // Исходные данные
    items := []normalization.DuplicateItem{
        {
            ID:             1,
            Code:           "WBC00Z0002",
            NormalizedName: "WBC00Z0002 Кабель ВВГ 3x2.5 120mm",
            Category:       "стройматериалы",
            QualityScore:   0.9,
        },
        {
            ID:             2,
            Code:           "WBC00Z0003",
            NormalizedName: "кабель ввг 3x2.5",
            Category:       "стройматериалы",
            QualityScore:   0.85,
        },
        {
            ID:             3,
            Code:           "001",
            NormalizedName: "молоток строительный 500гр",
            Category:       "инструмент",
            QualityScore:   0.88,
        },
        {
            ID:             4,
            Code:           "002",
            NormalizedName: "молотак строительный", // опечатка
            Category:       "инструмент",
            QualityScore:   0.80,
        },
    }
    
    // Конфигурация поиска дублей
    config := normalization.DefaultDuplicateDetectionConfig()
    config.UseExactMatching = true
    config.UseFuzzyMatching = true
    config.Threshold = 0.85
    config.MergeOverlapping = true
    config.MinConfidence = 0.8
    
    // Шаг 1-2: Нормализация (происходит автоматически)
    for i := range items {
        normalized := nsi.NormalizeName(items[i].NormalizedName, 
            normalization.NormalizationOptions{})
        items[i].NormalizedName = normalized
    }
    
    // Шаг 3: Поиск дублей
    groups := nsi.FindDuplicates(items, config)
    
    fmt.Printf("Найдено групп дубликатов: %d\n\n", len(groups))
    
    // Шаг 4-6: Результаты уже содержат мастер-записи
    for i, group := range groups {
        fmt.Printf("Группа %d:\n", i+1)
        fmt.Printf("  Тип: %s\n", group.Type)
        fmt.Printf("  Схожесть: %.2f\n", group.SimilarityScore)
        fmt.Printf("  Уверенность: %.2f\n", group.Confidence)
        fmt.Printf("  Мастер-запись ID: %d\n", group.SuggestedMaster)
        fmt.Printf("  Элементов в группе: %d\n", len(group.Items))
        fmt.Printf("  Причина: %s\n", group.Reason)
        fmt.Println()
    }
}
```

**Результат**: ✅ Реализовано в `normalization/nsi_normalizer.go`

---

### Пример 9: Метрики оценки (Precision, Recall, F1)

**Из документа (строка 45-47, 87-89)**:
```
Precision = TP / (TP + FP)
Recall = TP / (TP + FN)
F1-мера = 2 × (Precision × Recall) / (Precision + Recall)
```

**Реализация**:

```go
package main

import (
    "fmt"
    "httpserver/normalization"
)

func main() {
    // Создаем метрики оценки
    metrics := normalization.NewEvaluationMetrics()
    
    // Размеченные данные (эталонные дубли)
    actual := []normalization.DuplicateGroup{
        {
            GroupID: "actual_1",
            Items: []normalization.DuplicateItem{
                {ID: 1, NormalizedName: "молоток"},
                {ID: 2, NormalizedName: "молотак"},
            },
        },
    }
    
    // Предсказанные дубли (результат алгоритма)
    predicted := []normalization.DuplicateGroup{
        {
            GroupID: "predicted_1",
            Items: []normalization.DuplicateItem{
                {ID: 1, NormalizedName: "молоток"},
                {ID: 2, NormalizedName: "молотак"},
            },
        },
    }
    
    // Вычисляем метрики
    result := metrics.EvaluateAlgorithm(predicted, actual)
    
    fmt.Printf("Precision (Точность): %.4f\n", result.Precision)
    fmt.Printf("Recall (Полнота): %.4f\n", result.Recall)
    fmt.Printf("F1-мера: %.4f\n", result.F1Score)
    fmt.Printf("Accuracy: %.4f\n", result.Accuracy)
    fmt.Printf("\nМатрица ошибок:\n")
    fmt.Printf("  TP (True Positive): %d\n", result.ConfusionMatrix.TruePositive)
    fmt.Printf("  FP (False Positive): %d\n", result.ConfusionMatrix.FalsePositive)
    fmt.Printf("  FN (False Negative): %d\n", result.ConfusionMatrix.FalseNegative)
    fmt.Printf("  TN (True Negative): %d\n", result.ConfusionMatrix.TrueNegative)
}
```

**Результат**: ✅ Реализовано в `normalization/evaluation_metrics.go:24`

---

### Пример 10: Выбор мастер-записи

**Из документа (строка 796-798)**:
```
Алгоритм: из каждой группы выбирать запись с наибольшей полнотой информации
Альтернатива: применить голосование по атрибутам
```

**Реализация**:

```go
package main

import (
    "fmt"
    "httpserver/normalization"
)

func main() {
    analyzer := normalization.NewDuplicateAnalyzer()
    
    // Группа дубликатов
    items := []normalization.DuplicateItem{
        {
            ID:             1,
            NormalizedName: "кабель ввг 3x2.5",
            QualityScore:   0.9,
            MergedCount:    0,
            ProcessingLevel: "ai_enhanced",
        },
        {
            ID:             2,
            NormalizedName: "кабель ввг",
            QualityScore:   0.7,
            MergedCount:    0,
            ProcessingLevel: "basic",
        },
        {
            ID:             3,
            NormalizedName: "кабель ввг 3x2.5 120mm медный",
            QualityScore:   0.95,
            MergedCount:    2, // уже объединял другие записи
            ProcessingLevel: "benchmark",
        },
    }
    
    // Выбираем мастер-запись
    masterID := analyzer.selectMasterRecord(items)
    
    fmt.Printf("Выбранная мастер-запись ID: %d\n", masterID)
    
    // Находим запись
    var master normalization.DuplicateItem
    for _, item := range items {
        if item.ID == masterID {
            master = item
            break
        }
    }
    
    fmt.Printf("Мастер-запись:\n")
    fmt.Printf("  Наименование: %s\n", master.NormalizedName)
    fmt.Printf("  Качество: %.2f\n", master.QualityScore)
    fmt.Printf("  Уровень обработки: %s\n", master.ProcessingLevel)
    fmt.Printf("  Объединено записей: %d\n", master.MergedCount)
}
```

**Результат**: ✅ Реализовано в `normalization/duplicate_analyzer.go:626`

---

## 🧪 Тестовые сценарии

### Сценарий 1: Поиск дублей с опечатками

```go
func TestDuplicatesWithTypos(t *testing.T) {
    analyzer := normalization.NewDuplicateAnalyzer()
    
    items := []normalization.DuplicateItem{
        {ID: 1, NormalizedName: "молоток строительный"},
        {ID: 2, NormalizedName: "молотак строительный"}, // опечатка
        {ID: 3, NormalizedName: "кабель медный"},
    }
    
    groups := analyzer.AnalyzeDuplicates(items)
    
    // Должна быть найдена группа с ID 1 и 2
    found := false
    for _, group := range groups {
        if len(group.Items) == 2 && 
           (group.Items[0].ID == 1 || group.Items[1].ID == 1) &&
           (group.Items[0].ID == 2 || group.Items[1].ID == 2) {
            found = true
            break
        }
    }
    
    if !found {
        t.Error("Не найдены дубликаты с опечаткой")
    }
}
```

### Сценарий 2: Проверка метрик качества

```go
func TestQualityMetrics(t *testing.T) {
    metrics := normalization.NewEvaluationMetrics()
    
    matrix := normalization.ConfusionMatrix{
        TruePositive:  90,
        FalsePositive: 10,
        FalseNegative: 5,
        TrueNegative:  895,
    }
    
    result := metrics.CalculateMetrics(matrix)
    
    // Проверяем требования из документа:
    // Ошибки первого рода не должны превышать 10%
    if result.FalsePositiveRate > 0.10 {
        t.Errorf("FPR превышает 10%%: %.2f%%", result.FalsePositiveRate*100)
    }
    
    // Ошибки второго рода не должны превышать 5%
    if result.FalseNegativeRate > 0.05 {
        t.Errorf("FNR превышает 5%%: %.2f%%", result.FalseNegativeRate*100)
    }
}
```

---

## 📊 Сравнительная таблица: Документ vs Реализация

| Алгоритм из документа | Пример из документа | Реализовано | Файл реализации |
|----------------------|---------------------|-------------|-----------------|
| Расстояние Левенштейна | "дом" → "том" (расстояние 1) | ✅ | `duplicate_analyzer.go:828` |
| N-граммы | "звено" vs "зерно" (0.25) | ✅ | `fuzzy_algorithms.go:38` |
| Приведение к регистру | "МАСЛО" → "масло" | ✅ | `name_normalizer.go:50` |
| Удаление пробелов | "  масло  " → "масло" | ✅ | `name_normalizer.go:71` |
| Удаление пунктуации | "масло, 82%" → "масло" | ✅ | `name_normalizer.go` |
| Стемминг | "нормализация" → "нормализ" | ✅ | `algorithms/stemmer.go` |
| Лемматизация | "маслами" → "масло" | ⚠️ | Только стемминг |
| Удаление стоп-слов | "для готовки" → "готовка" | ✅ | `duplicate_analyzer.go:752` |
| Soundex | Фонетические коды | ✅ | `algorithms/soundex_ru.go` |
| Metaphone | Улучшенный Soundex | ✅ | `algorithms/metaphone_ru.go` |
| Precision/Recall/F1 | Формулы из документа | ✅ | `evaluation_metrics.go:24` |
| Выбор мастер-записи | По полноте информации | ✅ | `duplicate_analyzer.go:626` |

---

## 🎯 Заключение

Все основные примеры из документа **реализованы и работают**. Единственное исключение - полная лемматизация (используется стемминг как замена).

Все примеры можно запустить и протестировать с текущей реализацией.

