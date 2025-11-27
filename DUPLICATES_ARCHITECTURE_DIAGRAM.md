# Архитектура системы обнаружения дублей в НСИ

## Дата: 2025-01-20

---

## 🏗️ Общая архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                        API Layer (REST)                          │
├─────────────────────────────────────────────────────────────────┤
│  /api/quality/duplicates          /api/counterparties/duplicates│
│  GET, POST, PUT, DELETE           GET, POST (merge)             │
└────────────────────┬──────────────────────┬─────────────────────┘
                     │                      │
                     ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Business Logic Layer                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────┐      ┌──────────────────────────┐    │
│  │   NSINormalizer      │      │ CounterpartyDuplicate   │    │
│  │  (Unified Interface)  │      │      Analyzer            │    │
│  └──────────┬───────────┘      └──────────┬───────────────┘    │
│             │                             │                     │
│             ▼                             ▼                     │
│  ┌──────────────────────────────────────────────────────┐      │
│  │           DuplicateAnalyzer                          │      │
│  │  - Exact Matching                                    │      │
│  │  - Semantic Duplicates                              │      │
│  │  - Phonetic Duplicates                              │      │
│  │  - Word-based Duplicates                            │      │
│  └──────────┬───────────────────────────────────────────┘      │
│             │                                                   │
│             ▼                                                   │
│  ┌──────────────────────────────────────────────────────┐      │
│  │         FuzzyAlgorithms                              │      │
│  │  - Levenshtein Distance                             │      │
│  │  - Damerau-Levenshtein                              │      │
│  │  - N-grams (Bigram, Trigram)                        │      │
│  │  - Jaccard Index                                    │      │
│  │  - Jaro-Winkler                                     │      │
│  │  - LCS Similarity                                   │      │
│  └──────────┬───────────────────────────────────────────┘      │
│             │                                                   │
│             ▼                                                   │
│  ┌──────────────────────────────────────────────────────┐      │
│  │         NameNormalizer                               │      │
│  │  - Text Preprocessing                               │      │
│  │  - Attribute Extraction                             │      │
│  │  - Contextual Tokenization                          │      │
│  └──────────┬───────────────────────────────────────────┘      │
│             │                                                   │
│             ▼                                                   │
│  ┌──────────────────────────────────────────────────────┐      │
│  │      Phonetic Algorithms                             │      │
│  │  - Soundex (Russian)                                 │      │
│  │  - Metaphone (Russian)                               │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                  │
│  ┌──────────────────────────────────────────────────────┐      │
│  │         AINormalizer                                 │      │
│  │  - LLM Integration (Arliai API)                     │      │
│  │  - Batch Processing                                  │      │
│  │  - Caching                                           │      │
│  └───────────────────────────────────────────────────────┘      │
│                                                                  │
│  ┌──────────────────────────────────────────────────────┐      │
│  │      EvaluationMetrics                               │      │
│  │  - Precision, Recall, F1                              │      │
│  │  - ROC Curve, AUC                                    │      │
│  │  - Optimal Threshold                                 │      │
│  └───────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Data Layer                                  │
├─────────────────────────────────────────────────────────────────┤
│  - Database (SQLite)                                            │
│  - Duplicate Groups Storage                                     │
│  - Cache (in-memory)                                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Процесс обнаружения дублей

```
┌─────────────────────────────────────────────────────────────────┐
│                    Входные данные                                │
│              (Список записей НСИ)                                │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │   Предобработка (Normalize)   │
        │  - Lowercase                  │
        │  - Trim spaces                │
        │  - Remove punctuation         │
        │  - Unicode normalization      │
        └───────────────┬───────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │   Лингвистическая обработка   │
        │  - Stemming (Snowball)        │
        │  - Stop-word removal         │
        │  - Tokenization              │
        └───────────────┬───────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │   Поиск дублей (Multi-stage)  │
        │                               │
        │  1. Exact Matching           │
        │     └─> По коду               │
        │     └─> По имени              │
        │                               │
        │  2. Fuzzy Matching           │
        │     ├─> Levenshtein          │
        │     ├─> N-grams              │
        │     ├─> Jaccard              │
        │     └─> Combined Similarity  │
        │                               │
        │  3. Phonetic Matching        │
        │     ├─> Soundex              │
        │     └─> Metaphone            │
        │                               │
        │  4. Semantic Matching        │
        │     └─> Cosine Similarity    │
        │                               │
        │  5. Word-based Grouping      │
        │     └─> Common words         │
        └───────────────┬───────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │   Группировка дубликатов      │
        │  - Merge overlapping groups  │
        │  - Filter by confidence      │
        └───────────────┬───────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │   Выбор мастер-записи         │
        │  - По полноте информации     │
        │  - По качеству               │
        │  - По уровню обработки       │
        └───────────────┬───────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │   Оценка качества             │
        │  - Precision, Recall, F1     │
        │  - ROC Curve                 │
        │  - Optimal Threshold         │
        └───────────────┬───────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │      Результат               │
        │  (Группы дубликатов)         │
        └───────────────────────────────┘
```

---

## 🧩 Компоненты системы

### 1. NSINormalizer (Главный интерфейс)

```
NSINormalizer
├── AdvancedNormalizer
│   └── Normalize() - базовая нормализация
├── FuzzyAlgorithms
│   ├── LevenshteinSimilarity()
│   ├── DamerauLevenshteinSimilarity()
│   ├── BigramSimilarity()
│   ├── TrigramSimilarity()
│   ├── JaccardIndex()
│   ├── JaroWinklerSimilarity()
│   └── CombinedSimilarity()
├── NameNormalizer
│   ├── NormalizeName()
│   ├── ExtractAttributes()
│   └── ExtractAttributesContextual()
├── DuplicateAnalyzer
│   ├── AnalyzeDuplicates()
│   ├── mergeOverlappingGroups()
│   └── selectMasterRecord()
├── EvaluationMetrics
│   ├── CalculateMetrics()
│   ├── EvaluateAlgorithm()
│   ├── CalculateROC()
│   └── CalculateOptimalThreshold()
└── AINormalizer
    ├── NormalizeWithAI()
    └── BatchNormalize()
```

---

### 2. Алгоритмы нечеткого поиска

```
FuzzyAlgorithms
│
├── Расстояния
│   ├── Levenshtein Distance
│   │   └── O(n*m) сложность
│   └── Damerau-Levenshtein Distance
│       └── O(n*m) сложность, учитывает транспозиции
│
├── N-граммы
│   ├── Bigram Similarity
│   │   └── Формула Сёренсена
│   └── Trigram Similarity
│       └── Более точное сравнение
│
├── Индексы
│   ├── Jaccard Index
│   │   └── Intersection / Union
│   └── Jaro-Winkler
│       └── Учитывает префиксы
│
└── Комбинированные
    └── Combined Similarity
        └── Взвешенная комбинация алгоритмов
```

---

### 3. Фонетические алгоритмы

```
Phonetic Algorithms
│
├── Soundex (Russian)
│   ├── Кодирование по звучанию
│   └── Для похожих по произношению слов
│
└── Metaphone (Russian)
    ├── Улучшенная версия Soundex
    └── Более точное кодирование
```

---

### 4. Метрики оценки

```
EvaluationMetrics
│
├── Базовые метрики
│   ├── Precision = TP / (TP + FP)
│   ├── Recall = TP / (TP + FN)
│   ├── F1 = 2 * (P * R) / (P + R)
│   └── Accuracy = (TP + TN) / Total
│
├── Дополнительные метрики
│   ├── Specificity = TN / (TN + FP)
│   ├── FPR = FP / (FP + TN)
│   └── FNR = FN / (FN + TP)
│
├── ROC анализ
│   ├── ROC Curve
│   └── AUC (Area Under Curve)
│
└── Оптимизация
    └── Optimal Threshold
        └── Максимизация F1-меры
```

---

## 🔌 API Endpoints

```
API Endpoints
│
├── Quality Duplicates
│   ├── GET /api/quality/duplicates
│   │   └── Получить список групп дубликатов
│   ├── POST /api/quality/duplicates/{groupId}/merge
│   │   └── Объединить группу дубликатов
│   ├── PUT /api/quality/duplicates/{groupId}
│   │   └── Обновить группу
│   └── DELETE /api/quality/duplicates/{groupId}
│       └── Удалить группу
│
└── Counterparty Duplicates
    ├── GET /api/counterparties/duplicates
    │   └── Получить дубли контрагентов
    └── POST /api/counterparties/duplicates/{groupId}/merge
        └── Объединить дубли контрагентов
```

---

## 📊 Поток данных

```
┌─────────────┐
│   Request   │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  API Handler    │
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│ NSINormalizer   │
│ FindDuplicates()│
└──────┬──────────┘
       │
       ├──► Exact Matching
       ├──► Fuzzy Matching
       ├──► Phonetic Matching
       ├──► Semantic Matching
       └──► Word-based Grouping
       │
       ▼
┌─────────────────┐
│  Group Merging  │
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│ Master Selection│
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│   Response      │
│ (JSON Groups)   │
└─────────────────┘
```

---

## 🎯 Ключевые особенности архитектуры

### ✅ Модульность
- Каждый компонент независим
- Легко тестировать отдельно
- Простое расширение

### ✅ Производительность
- Кэширование результатов
- Оптимизированные алгоритмы
- Batch processing для AI

### ✅ Расширяемость
- Легко добавить новые алгоритмы
- Плагинная архитектура
- Универсальный интерфейс

### ✅ Качество
- Метрики оценки встроены
- Валидация результатов
- Оптимизация порогов

---

## 📈 Масштабируемость

```
Малые объемы (< 1K записей)
    └─> In-memory обработка
    └─> Все алгоритмы

Средние объемы (1K - 100K записей)
    └─> Кэширование
    └─> Batch processing
    └─> Оптимизированные алгоритмы

Большие объемы (> 100K записей)
    └─> Префиксная фильтрация (TODO)
    └─> Индексирование
    └─> Параллельная обработка
```

---

## 🔮 Будущие улучшения

```
Текущая архитектура
    │
    ├─► Фаза 1: Лемматизация
    │   └─> Полная лемматизация вместо стемминга
    │
    ├─► Фаза 2: ML интеграция
    │   ├─> BERT для семантики
    │   └─> Random Forest для классификации
    │
    └─► Фаза 3: Глубокое обучение
        ├─> Seq2Seq нормализация
        └─> Gradient Boosting
```

---

**Дата создания**: 2025-01-20  
**Версия**: 1.0  
**Статус**: ✅ Актуально

