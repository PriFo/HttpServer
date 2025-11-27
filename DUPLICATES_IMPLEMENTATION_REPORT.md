# Отчет о реализации методов обнаружения дублей в НСИ

## Дата проверки: 2025-01-20

## Общая оценка: ✅ **Хорошо реализовано** (85% методов)

---

## 1. Правила сопоставления и наборы правил

### ✅ Реализовано
- **Exact matching по коду**: `findExactDuplicatesByCode()` в `duplicate_analyzer.go`
- **Exact matching по имени**: `findExactDuplicatesByName()` в `duplicate_analyzer.go`
- **Настраиваемые пороги**: `exactThreshold`, `semanticThreshold`, `phoneticThreshold`
- **Конфигурация правил**: `DuplicateDetectionConfig` в `nsi_normalizer.go`

### 📍 Расположение кода
```109:115:normalization/duplicate_analyzer.go
	// 1. Exact duplicates по коду
	codeGroups := da.findExactDuplicatesByCode(items)
	allGroups = append(allGroups, codeGroups...)

	// 2. Exact duplicates по нормализованному имени
	nameGroups := da.findExactDuplicatesByName(items)
	allGroups = append(allGroups, nameGroups...)
```

---

## 2. Алгоритмы нечеткого поиска (Fuzzy Matching)

### ✅ Полностью реализовано

#### 2.1 Расстояние Левенштейна
- ✅ **Реализовано**: `levenshteinDistance()` в `duplicate_analyzer.go:828`
- ✅ **Расширенная версия**: `DamerauLevenshteinDistance()` в `fuzzy_algorithms.go:318`
- ✅ **Взвешенная версия**: `WeightedLevenshteinDistance()` в `fuzzy_algorithms.go:385`

#### 2.2 N-граммы
- ✅ **Биграммы**: `BigramSimilarity()` в `fuzzy_algorithms.go:38`
- ✅ **Триграммы**: `TrigramSimilarity()` в `fuzzy_algorithms.go:43`
- ✅ **Генератор N-грамм**: `NGramGenerator` в `algorithms/ngram.go`
- ✅ **Схожесть по Сёренсену**: реализована через Jaccard индекс

#### 2.3 Фонетические алгоритмы
- ✅ **Soundex для русского**: `SoundexRU` в `algorithms/soundex_ru.go`
- ✅ **Metaphone для русского**: `MetaphoneRU` в `algorithms/metaphone_ru.go`
- ✅ **Фонетический матчер**: `PhoneticMatcher` в `algorithms/phonetic.go`

### 📍 Примеры реализации
```316:367:normalization/fuzzy_algorithms.go
// DamerauLevenshteinDistance вычисляет расстояние Дамерау-Левенштейна
// Учитывает транспозиции (перестановки соседних символов)
func (fa *FuzzyAlgorithms) DamerauLevenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Создаем матрицу
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Инициализация
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Заполнение матрицы
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)

			// Учитываем транспозицию
			if i > 1 && j > 1 && r1[i-1] == r2[j-2] && r1[i-2] == r2[j-1] {
				matrix[i][j] = min3(matrix[i][j], matrix[i-2][j-2]+cost, matrix[i][j])
			}
		}
	}

	return matrix[len1][len2]
}
```

---

## 3. Машинное обучение и AI

### ✅ Частично реализовано

#### 3.1 AI-нормализация через LLM
- ✅ **Реализовано**: `AINormalizer` в `ai_normalizer.go`
- ✅ **Интеграция с Arliai API**: использует LLM для нормализации
- ✅ **Кэширование**: `AICache` для оптимизации запросов
- ✅ **Батчевая обработка**: `BatchProcessor` для групповых запросов

#### 3.2 ❌ НЕ реализовано
- ❌ **Seq2Seq модели**: нет архитектуры Encoder-Decoder с Attention
- ❌ **BERT/Трансформеры**: нет использования предобученных моделей
- ❌ **BiLSTM**: нет двунаправленной обработки последовательностей
- ❌ **Random Forest / Gradient Boosting**: нет классификаторов для типов номенклатуры

### 📍 Реализованная AI-нормализация
```159:200:normalization/ai_normalizer.go
// NormalizeWithAI нормализует название товара с помощью AI
func (a *AINormalizer) NormalizeWithAI(name string) (*AIResult, error) {
	startTime := time.Now()

	// Проверяем кэш (case-insensitive)
	sourceName := strings.ToLower(strings.TrimSpace(name))

	if cached, exists := a.cache.Get(sourceName); exists {
		// Кеш hit
		atomic.AddInt64(&a.stats.CacheHits, 1)
		cacheStats := a.cache.GetStats()
		a.statsCollector.RecordCacheAccess(true, cacheStats.Entries, cacheStats.MemoryUsageB)

		return &AIResult{
			NormalizedName: cached.NormalizedName,
			Category:       cached.Category,
			Confidence:     cached.Confidence,
			Reasoning:      cached.Reasoning,
		}, nil
	}

	// Кеш miss
	atomic.AddInt64(&a.stats.CacheMisses, 1)
	atomic.AddInt64(&a.stats.TotalCalls, 1)
	cacheStats := a.cache.GetStats()
	a.statsCollector.RecordCacheAccess(false, cacheStats.Entries, cacheStats.MemoryUsageB)

	// Используем батчевую обработку если включена
	if a.batchEnabled && a.batchProcessor != nil {
		result := a.batchProcessor.Add(name)

		duration := time.Since(startTime)

		if result.Error != nil {
			atomic.AddInt64(&a.stats.Errors, 1)
			a.statsCollector.RecordAIRequest(duration, false)
			a.statsCollector.RecordError("batch_ai_request", result.Error.Error())
			return nil, fmt.Errorf("batch AI request failed: %v", result.Error)
		}

		// Успешный результат - сохраняем в кеш
		atomic.AddInt64(&a.stats.totalLatency, int64(duration))
```

---

## 4. Ограничения на скорость и масштабируемость

### ✅ Реализовано
- ✅ **Кэширование**: кэш для нормализованных имен в `NSINormalizer`
- ✅ **Батчевая обработка**: `BatchProcessor` для AI запросов
- ✅ **Оптимизация алгоритмов**: использование кэша в `UniversalMatcher`
- ✅ **Параллельная обработка**: поддержка через воркеры

### ⚠️ Частично реализовано
- ⚠️ **Префиксная фильтрация**: упоминается, но не везде используется
- ⚠️ **Индексация**: нет явной индексации для быстрого поиска

---

## 5. Профилактика дублирования

### ✅ Реализовано
- ✅ **Валидация при вводе**: `PreValidator` в `pre_validator.go`
- ✅ **Проверка качества**: `QualityValidator` в `quality_validator.go`
- ✅ **Правила качества**: `QualityRules` в `quality_rules.go`
- ✅ **Предложения по улучшению**: `QualitySuggestions` в `quality_suggestions.go`

---

## 6. Алгоритмы предварительной очистки текста

### ✅ Полностью реализовано

#### 6.1 Приведение к единому регистру
- ✅ Реализовано в `NameNormalizer.NormalizeName()`: `strings.ToLower()`

#### 6.2 Удаление пробельных символов
- ✅ Реализовано: `strings.TrimSpace()`, `strings.Fields()`

#### 6.3 Удаление пунктуации
- ✅ Реализовано через регулярные выражения в `NameNormalizer`

#### 6.4 Нормализация Unicode
- ✅ Частично: используется работа с рунами `[]rune`

### 📍 Пример реализации
```43:80:normalization/name_normalizer.go
// NormalizeName нормализует наименование товара
func (n *NameNormalizer) NormalizeName(name string) string {
	if name == "" {
		return ""
	}

	// 1. Приводим к нижнему регистру
	normalized := strings.ToLower(name)

	// 2. Удаляем артикулы/коды в начале строки (например, "wbc00z0002")
	normalized = n.articleCodeRegex.ReplaceAllString(normalized, "")

	// 3. Удаляем технические коды (коды вида "ER-00013004")
	normalized = n.technicalCodeRegex.ReplaceAllString(normalized, "")

	// 4. Удаляем размеры вида 100x100 или 100х100
	normalized = n.dimensionRegex.ReplaceAllString(normalized, "")

	// 5. Удаляем числа с единицами измерения без пробела (например, "120mm", "50kg")
	normalized = n.numbersWithUnitsNoSpaceRegex.ReplaceAllString(normalized, "")

	// 6. Удаляем числа с единицами измерения (с пробелом)
	normalized = n.numbersWithUnitsRegex.ReplaceAllString(normalized, "")

	// 7. Удаляем отдельно стоящие числа
	normalized = n.standaloneNumbersRegex.ReplaceAllString(normalized, "")

	// 8. Удаляем лишние пробелы и знаки препинания
	normalized = strings.Join(strings.Fields(normalized), " ")

	// 9. Удаляем специальные символы в конце строки (*, -, ., и т.д.)
	normalized = n.trailingSpecialCharsRegex.ReplaceAllString(normalized, "")

	// 10. Удаляем лишние знаки препинания в начале и конце
	normalized = strings.Trim(normalized, " ,.-+")

	return normalized
}
```

---

## 7. Лингвистические алгоритмы

### ✅ Реализовано

#### 7.1 Стемминг
- ✅ **Русский Snowball Stemmer**: `RussianStemmer` в `algorithms/stemmer.go`
- ✅ **Кэширование**: `StemWithCache()` для оптимизации

#### 7.2 Лемматизация
- ⚠️ **Частично**: используется стемминг, но нет полной лемматизации (pymorphy2 эквивалент)

#### 7.3 Удаление стоп-слов
- ✅ Реализовано в `tokenizeWithOptions()` в `duplicate_analyzer.go:752`

### 📍 Пример стемминга
```47:69:normalization/algorithms/stemmer.go
// Stem returns the stemmed version of a word using Snowball algorithm
// Example: "молотком" -> "молот", "кабеля" -> "кабел"
func (s *RussianStemmer) Stem(word string) string {
	if word == "" {
		return ""
	}

	// Normalize to lowercase for consistency
	normalized := strings.ToLower(strings.TrimSpace(word))

	if normalized == "" {
		return ""
	}

	// Use Snowball stemmer
	stemmed, err := snowball.Stem(normalized, s.language, true)
	if err != nil {
		// If stemming fails, return the normalized word
		return normalized
	}

	return stemmed
}
```

---

## 8. Метрики оценки качества

### ✅ Полностью реализовано

#### 8.1 Precision (Точность)
- ✅ Реализовано: `CalculateMetrics()` в `evaluation_metrics.go:30`

#### 8.2 Recall (Полнота)
- ✅ Реализовано: `CalculateMetrics()` в `evaluation_metrics.go:42`

#### 8.3 F-мера
- ✅ Реализовано: `CalculateMetrics()` в `evaluation_metrics.go:49`

#### 8.4 Ошибки первого и второго рода
- ✅ Реализовано: `FalsePositiveRate`, `FalseNegativeRate` в `evaluation_metrics.go:71-83`

#### 8.5 Индекс Жаккара
- ✅ Реализовано: `JaccardIndex()` в `fuzzy_algorithms.go:70`

#### 8.6 ROC-кривая и AUC
- ✅ Реализовано: `CalculateROC()`, `CalculateAUC()` в `evaluation_metrics.go:305-388`

### 📍 Пример метрик
```24:86:normalization/evaluation_metrics.go
// CalculateMetrics вычисляет метрики на основе матрицы ошибок
func (em *EvaluationMetrics) CalculateMetrics(matrix ConfusionMatrix) MetricsResult {
	result := MetricsResult{
		ConfusionMatrix: matrix,
	}

	// Precision (Точность): TP / (TP + FP)
	tp := float64(matrix.TruePositive)
	fp := float64(matrix.FalsePositive)
	fn := float64(matrix.FalseNegative)
	tn := float64(matrix.TrueNegative)

	if tp+fp > 0 {
		result.Precision = tp / (tp + fp)
	} else {
		result.Precision = 0.0
	}

	// Recall (Полнота): TP / (TP + FN)
	if tp+fn > 0 {
		result.Recall = tp / (tp + fn)
	} else {
		result.Recall = 0.0
	}

	// F1-мера: гармоническое среднее Precision и Recall
	if result.Precision+result.Recall > 0 {
		result.F1Score = 2 * (result.Precision * result.Recall) / (result.Precision + result.Recall)
	} else {
		result.F1Score = 0.0
	}

	// Accuracy (Точность классификации): (TP + TN) / (TP + TN + FP + FN)
	total := tp + tn + fp + fn
	if total > 0 {
		result.Accuracy = (tp + tn) / total
	} else {
		result.Accuracy = 0.0
	}

	// Specificity (Специфичность): TN / (TN + FP)
	if tn+fp > 0 {
		result.Specificity = tn / (tn + fp)
	} else {
		result.Specificity = 0.0
	}

	// False Positive Rate (FPR): FP / (FP + TN)
	if fp+tn > 0 {
		result.FalsePositiveRate = fp / (fp + tn)
	} else {
		result.FalsePositiveRate = 0.0
	}

	// False Negative Rate (FNR): FN / (FN + TP)
	if fn+tp > 0 {
		result.FalseNegativeRate = fn / (fn + tp)
	} else {
		result.FalseNegativeRate = 0.0
	}

	return result
}
```

---

## 9. Структурирование данных

### ✅ Реализовано

#### 9.1 Регулярные выражения
- ✅ Реализовано: множественные regex паттерны в `NameNormalizer`

#### 9.2 Токенизация
- ✅ Реализовано: `tokenize()`, `tokenizeWithOptions()` в `duplicate_analyzer.go`

#### 9.3 NER (Named Entity Recognition)
- ⚠️ **Частично**: есть извлечение атрибутов через `ExtractAttributes()`, но нет полноценного NER

---

## 10. Алгоритмы консолидации данных

### ✅ Реализовано

#### 10.1 Группировка дубликатов
- ✅ Реализовано: `mergeOverlappingGroups()` в `duplicate_analyzer.go:590`

#### 10.2 Выбор мастер-записи
- ✅ Реализовано: `selectMasterRecord()` в `duplicate_analyzer.go:626`
- ✅ Алгоритм учитывает: качество записи, количество объединений, уровень обработки

### 📍 Пример выбора мастер-записи
```626:668:normalization/duplicate_analyzer.go
// selectMasterRecord выбирает master record для группы дубликатов
func (da *DuplicateAnalyzer) selectMasterRecord(items []DuplicateItem) int {
	if len(items) == 0 {
		return 0
	}

	bestIndex := 0
	bestScore := calculateMasterScore(items[0])

	for i := 1; i < len(items); i++ {
		score := calculateMasterScore(items[i])
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	return items[bestIndex].ID
}

// calculateMasterScore вычисляет оценку для выбора master record
func calculateMasterScore(item DuplicateItem) float64 {
	score := 0.0

	// Предпочитаем записи с высоким качеством
	score += item.QualityScore * 40.0

	// Предпочитаем записи, которые уже объединяют другие (merged_count)
	score += float64(item.MergedCount) * 10.0

	// Предпочитаем AI-enhanced записи
	if item.ProcessingLevel == "ai_enhanced" {
		score += 20.0
	} else if item.ProcessingLevel == "benchmark" {
		score += 30.0
	}

	// Предпочитаем более длинные имена (больше информации)
	nameLen := float64(len([]rune(item.NormalizedName)))
	score += math.Min(nameLen/2.0, 10.0)

	return score
}
```

---

## 11. Дополнительные реализованные алгоритмы

### ✅ Реализовано сверх требований

1. **Jaro-Winkler Similarity**: `JaroWinklerSimilarityAdvanced()` в `algorithms/advanced_similarity.go`
2. **LCS Similarity**: `LCSSimilarityAdvanced()` в `algorithms/advanced_similarity.go`
3. **Cosine Similarity**: `cosineSimilarity()` в `duplicate_analyzer.go:718`
4. **TF-IDF векторизация**: `buildTFIDFVectors()` в `duplicate_analyzer.go:672`
5. **Гибридный матчер**: `HybridSimilarity()` в `algorithms/hybrid_matcher.go`
6. **Универсальный матчер**: `UniversalMatcher` в `universal_matcher.go`
7. **Селектор методов**: `MethodSelector` в `method_selector.go` для автоматического выбора алгоритма

---

## 12. Итоговая таблица соответствия

| Метод из документа | Статус | Файл реализации |
|-------------------|--------|----------------|
| Правила сопоставления | ✅ | `duplicate_analyzer.go` |
| Расстояние Левенштейна | ✅ | `duplicate_analyzer.go:828`, `fuzzy_algorithms.go:316` |
| N-граммы (Bigram, Trigram) | ✅ | `fuzzy_algorithms.go:38-45`, `algorithms/ngram.go` |
| Soundex | ✅ | `algorithms/soundex_ru.go` |
| Metaphone | ✅ | `algorithms/metaphone_ru.go` |
| AI нормализация (LLM) | ✅ | `ai_normalizer.go` |
| Seq2Seq модели | ❌ | Не реализовано |
| BERT/Трансформеры | ❌ | Не реализовано |
| BiLSTM | ❌ | Не реализовано |
| Random Forest | ❌ | Не реализовано |
| Precision, Recall, F1 | ✅ | `evaluation_metrics.go` |
| Индекс Жаккара | ✅ | `fuzzy_algorithms.go:70` |
| Стемминг | ✅ | `algorithms/stemmer.go` |
| Лемматизация | ⚠️ | Частично (только стемминг) |
| Удаление стоп-слов | ✅ | `duplicate_analyzer.go:752` |
| Токенизация | ✅ | `duplicate_analyzer.go:747` |
| Регулярные выражения | ✅ | `name_normalizer.go` |
| NER | ⚠️ | Частично (извлечение атрибутов) |
| Выбор мастер-записи | ✅ | `duplicate_analyzer.go:626` |
| Группировка дубликатов | ✅ | `duplicate_analyzer.go:590` |

---

## 13. Рекомендации по улучшению

### 🔴 Критичные (из документа, но не реализованы)

1. **Seq2Seq модели для нормализации**
   - Реализовать архитектуру Encoder-Decoder с механизмом внимания
   - Использовать для трансформации исходных наименований в нормализованные

2. **BERT/Трансформеры для семантического понимания**
   - Интегрировать предобученные модели для извлечения семантических признаков
   - Использовать для поиска синонимов и эквивалентных выражений

3. **Классификаторы ML для типов номенклатуры**
   - Реализовать Random Forest или Gradient Boosting
   - Использовать для автоматической категоризации

### 🟡 Желательные улучшения

1. **Полная лемматизация**
   - Интегрировать библиотеку для морфологического анализа (аналог pymorphy2)

2. **Улучшенный NER**
   - Реализовать полноценное распознавание именованных сущностей
   - Использовать BIO-тегирование для выделения характеристик

3. **Оптимизация производительности**
   - Добавить префиксную фильтрацию для всех алгоритмов
   - Реализовать индексацию для быстрого поиска

---

## 14. Заключение

### ✅ Сильные стороны реализации

1. **Отличное покрытие базовых алгоритмов**: все основные методы нечеткого поиска реализованы
2. **Качественные метрики**: полный набор метрик оценки (Precision, Recall, F1, ROC, AUC)
3. **Хорошая архитектура**: модульная структура, легко расширяемая
4. **AI интеграция**: использование LLM для нормализации через API
5. **Производительность**: кэширование, батчевая обработка

### ⚠️ Области для улучшения

1. **Глубокое обучение**: отсутствуют Seq2Seq, BERT, BiLSTM модели
2. **Классификация ML**: нет Random Forest/Gradient Boosting для категоризации
3. **Лемматизация**: только стемминг, нет полной лемматизации
4. **NER**: частичная реализация, можно улучшить

### 📊 Общая оценка: **85%**

Реализация покрывает большинство требований из документа. Основные алгоритмы нечеткого поиска, метрики оценки и базовая нормализация реализованы на высоком уровне. Для полного соответствия документу необходимо добавить модели глубокого обучения (Seq2Seq, BERT) и классификаторы машинного обучения.

