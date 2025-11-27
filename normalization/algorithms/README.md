# Алгоритмы нормализации наименований НСИ

Этот пакет содержит реализацию различных алгоритмов для нормализации и сравнения наименований в нормативно-справочной информации (НСИ).

## Доступные алгоритмы

### 1. Soundex (RU)
Фонетический алгоритм для русского языка. Кодирует слова в код, где похожие по звучанию слова получают одинаковый код.

```go
soundex := algorithms.NewSoundexRU()
code := soundex.Encode("Москва")
similarity := soundex.Similarity("Москва", "Москва")
```

### 2. Metaphone (RU)
Улучшенный фонетический алгоритм для русского языка. Более точный чем Soundex.

```go
metaphone := algorithms.NewMetaphoneRU()
code := metaphone.Encode("Петербург")
similarity := metaphone.Similarity("Петербург", "Питербург")
```

### 3. Индекс Жаккара
Сравнение множеств токенов или N-грамм. Возвращает значение от 0.0 до 1.0.

```go
jaccard := algorithms.NewJaccardIndex()
similarity := jaccard.Similarity("текст один", "текст два")

// С N-граммами
jaccardNGrams := algorithms.NewJaccardIndexWithNGrams(2)
similarity := jaccardNGrams.Similarity("текст", "текст")
```

### 4. N-граммы
Сравнение на основе N-грамм (биграммы, триграммы и т.д.).

```go
gen := algorithms.NewNGramGenerator(2) // биграммы
ngrams := gen.Generate("текст")
similarity := gen.Similarity("текст1", "текст2")
```

### 5. Расстояние Дамерау-Левенштейна
Улучшенная версия расстояния Левенштейна с учетом транспозиций.

```go
dl := algorithms.NewDamerauLevenshtein()
distance := dl.Distance("текст1", "текст2")
similarity := dl.Similarity("текст1", "текст2")
```

### 6. Косинусная близость
Сравнение на основе TF-IDF векторов или частотных векторов.

```go
cosine := algorithms.NewCosineSimilarity()
similarity := cosine.Similarity("текст1", "текст2")

// С N-граммами
similarity := cosine.SimilarityWithNGrams("текст1", "текст2", 2)
```

### 7. Токен-ориентированные методы
Сравнение на основе общих токенов с возможностью взвешивания.

```go
token := algorithms.NewTokenBasedSimilarity()
similarity := token.Similarity("текст один", "текст два")

// С взвешиванием
tokenWeighted := algorithms.NewTokenBasedSimilarityWeighted()
similarity := tokenWeighted.Similarity("текст1", "текст2")
```

## Использование через Pipeline

Рекомендуется использовать алгоритмы через конфигурируемый pipeline:

```go
import "httpserver/normalization/pipeline_normalization"

config := pipeline_normalization.NewDefaultConfig()
pipeline, _ := pipeline_normalization.NewNormalizationPipeline(config)
result, _ := pipeline.Normalize("текст1", "текст2")
```

## Производительность

- **Soundex/Metaphone**: O(n) - очень быстрые
- **Jaccard**: O(n+m) - быстрые
- **N-граммы**: O(n*m) - средние
- **Дамерау-Левенштейн**: O(n*m) - средние
- **Косинусная близость**: O(n+m) - быстрые
- **Токен-ориентированные**: O(n+m) - быстрые

## Рекомендации по выбору алгоритмов

- **Для опечаток**: Soundex, Metaphone, Дамерау-Левенштейн
- **Для семантического сравнения**: Косинусная близость, Токен-ориентированные
- **Для точного сравнения**: N-граммы, Индекс Жаккара
- **Для комбинированного анализа**: Используйте pipeline с несколькими алгоритмами
