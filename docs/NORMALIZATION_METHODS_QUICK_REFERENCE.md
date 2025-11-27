# 🚀 Быстрый справочник методик нормализации НСИ

## 📌 Основные классы и методы

### 1. Базовая нормализация
```go
normalizer := normalization.NewNameNormalizer()
normalized := normalizer.NormalizeName("WBC00Z0002 Кабель ВВГ 3x2.5 120mm")
// Результат: "кабель ввг"
```

### 2. Извлечение атрибутов
```go
normalized, attrs := normalizer.ExtractAttributes("WBC00Z0002 Кабель ВВГ 3x2.5")
// Извлекает: артикулы, размеры, единицы измерения
```

### 3. Категоризация
```go
categorizer := normalization.NewCategorizer()
category := categorizer.Categorize("Кабель ВВГ 3x2.5")
// Результат: "Кабели и провода"
```

### 4. Поиск дубликатов
```go
analyzer := normalization.NewDuplicateAnalyzer()
groups := analyzer.AnalyzeDuplicates(items)
// Находит: exact, semantic, phonetic, word-based дубликаты
```

### 5. Fuzzy Matching
```go
fuzzyMatcher := quality.NewFuzzyMatcher(db, 0.85)
duplicates := fuzzyMatcher.FindDuplicateNames(uploadID, databaseID)
// Использует: Levenshtein distance, префиксная фильтрация
```

### 6. AI-нормализация
```go
aiNormalizer := normalization.NewAINormalizer(aiConfig)
result := aiNormalizer.NormalizeWithAI(name, category)
// Возвращает: улучшенное имя, категорию, уверенность, обоснование
```

## 📊 Алгоритмы

| Алгоритм | Класс | Метод | Порог |
|----------|-------|-------|-------|
| Levenshtein Distance | `FuzzyMatcher` | `levenshteinDistance()` | - |
| Cosine Similarity | `DuplicateAnalyzer` | `findSemanticDuplicates()` | 0.85 |
| Phonetic Hash | `DuplicateAnalyzer` | `phoneticHash()` | 0.90 |
| Word-based Grouping | `DuplicateAnalyzer` | `findWordBasedDuplicates()` | 1 слово |
| Exact Matching | `DuplicateAnalyzer` | `findExactDuplicatesByCode()` | 1.0 |

## 🔧 Регулярные выражения

```go
// Артикулы: ^[a-zа-я]{2,}\d+[a-zа-я]*\d+\s*
// Технические коды: \b[A-Z]{2}-\d+\b
// Размеры: \d+[xх]\d+
// Единицы измерения: \d+\.?\d*\s*(см|мм|м|л|кг|%|г|мг|шт|мл|...)
```

## 📈 Производительность

- **Обработка:** 500-1000 записей/сек
- **Поиск дубликатов:** несколько секунд для 100K записей
- **Fuzzy matching:** оптимизирован с префиксной фильтрацией

## 📚 Полная документация

См. `docs/NORMALIZATION_METHODS_COMPLETE.md` для детального описания всех методов.

