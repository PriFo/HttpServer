# Примеры использования алгоритмов нормализации НСИ

## Полный пример: Поиск дублей в справочнике номенклатуры

```go
package main

import (
	"fmt"
	"httpserver/normalization"
	"httpserver/normalization/algorithms"
)

func main() {
	// 1. Создаем анализатор дубликатов
	analyzer := normalization.NewDuplicateAnalyzer()
	
	// 2. Включаем продвинутые методы (по умолчанию включены)
	analyzer.EnableAdvancedMethods(true)
	
	// 3. Подготавливаем данные для анализа
	items := []normalization.DuplicateItem{
		{
			ID:             1,
			Code:           "001",
			NormalizedName: "молоток строительный",
			Category:       "инструмент",
		},
		{
			ID:             2,
			Code:           "002",
			NormalizedName: "молотак строительный", // опечатка
			Category:       "инструмент",
		},
		{
			ID:             3,
			Code:           "003",
			NormalizedName: "кабель медный",
			Category:       "стройматериалы",
		},
		{
			ID:             4,
			Code:           "004",
			NormalizedName: "кабель медный ввгнг",
			Category:       "стройматериалы",
		},
	}
	
	// 4. Анализируем дубликаты
	groups := analyzer.AnalyzeDuplicates(items)
	
	// 5. Выводим результаты
	fmt.Printf("Найдено групп дубликатов: %d\n\n", len(groups))
	for i, group := range groups {
		fmt.Printf("Группа %d:\n", i+1)
		fmt.Printf("  Тип: %s\n", group.Type)
		fmt.Printf("  Схожесть: %.2f\n", group.SimilarityScore)
		fmt.Printf("  Уверенность: %.2f\n", group.Confidence)
		fmt.Printf("  Причина: %s\n", group.Reason)
		fmt.Printf("  Элементы:\n")
		for _, item := range group.Items {
			fmt.Printf("    - [%s] %s (%s)\n", item.Code, item.NormalizedName, item.Category)
		}
		fmt.Println()
	}
}
```

## Пример: Использование отдельных алгоритмов

### N-граммы

```go
// Создаем генератор биграмм
ngramGen := algorithms.NewNGramGenerator(2)

// Вычисляем схожесть
similarity := ngramGen.Similarity("молоток", "молотак")
fmt.Printf("Схожесть N-грамм: %.2f\n", similarity) // ~0.7

// Находим похожие тексты
candidates := []string{
	"молоток строительный",
	"молотак строительный",
	"кабель медный",
}
similar := ngramGen.FindSimilar("молоток", candidates, 0.6)
fmt.Printf("Похожие тексты: %v\n", similar) // [0, 1]
```

### Фонетические алгоритмы

```go
// Создаем фонетический матчер
phoneticMatcher := algorithms.NewPhoneticMatcher()

// Soundex код
soundexCode := phoneticMatcher.EncodeSoundex("молоток")
fmt.Printf("Soundex: %s\n", soundexCode) // М000

// Metaphone код
metaphoneCode := phoneticMatcher.EncodeMetaphone("молоток")
fmt.Printf("Metaphone: %s\n", metaphoneCode)

// Фонетическая схожесть
similarity := phoneticMatcher.Similarity("молоток", "молотак")
fmt.Printf("Фонетическая схожесть: %.2f\n", similarity) // ~0.8
```

### Метрики схожести

```go
// Создаем метрики схожести
metrics := algorithms.NewSimilarityMetrics()

text1 := "молоток строительный"
text2 := "молотак строительный"

// Индекс Жаккара
jaccard := metrics.JaccardIndex(text1, text2)
fmt.Printf("Jaccard: %.2f\n", jaccard)

// Косинусное сходство
cosine := metrics.CosineSimilarity(text1, text2)
fmt.Printf("Cosine: %.2f\n", cosine)

// Расстояние Левенштейна
levenshtein := metrics.LevenshteinSimilarity(text1, text2)
fmt.Printf("Levenshtein: %.2f\n", levenshtein)

// Расстояние Дамерау-Левенштейна (с учетом транспозиций)
damerau := metrics.DamerauLevenshteinSimilarity(text1, text2)
fmt.Printf("Damerau-Levenshtein: %.2f\n", damerau)

// Комбинированная метрика (рекомендуется)
combined := metrics.CombinedSimilarity(text1, text2)
fmt.Printf("Combined: %.2f\n", combined)
```

### Система правил

```go
// Создаем движок правил
engine := algorithms.NewRuleEngine()

// Создаем набор правил для номенклатуры
ruleSet := algorithms.CreateNomenclatureRuleSet("nomenclature_001")
engine.RegisterRuleSet(ruleSet)

// Проверяем записи
record1 := map[string]string{
	"code":            "001",
	"name":            "молоток строительный",
	"normalized_name": "молоток строительный",
}

record2 := map[string]string{
	"code":            "001",
	"name":            "молоток строительный",
	"normalized_name": "молоток строительный",
}

similarity, reason, isDuplicate := engine.MatchRecords(record1, record2, "nomenclature_001")
if isDuplicate {
	fmt.Printf("Дубликат найден! Схожесть: %.2f, Причина: %s\n", similarity, reason)
}
```

### Метрики оценки эффективности

```go
// Создаем метрики оценки
evalMetrics := algorithms.NewEvaluationMetrics()

// Симулируем результаты классификации
// TP=90, FP=5, FN=3, TN=100
for i := 0; i < 90; i++ {
	evalMetrics.AddResult(true, true) // TP
}
for i := 0; i < 5; i++ {
	evalMetrics.AddResult(true, false) // FP
}
for i := 0; i < 3; i++ {
	evalMetrics.AddResult(false, true) // FN
}
for i := 0; i < 100; i++ {
	evalMetrics.AddResult(false, false) // TN
}

// Получаем метрики
fmt.Printf("Precision: %.4f\n", evalMetrics.Precision())
fmt.Printf("Recall: %.4f\n", evalMetrics.Recall())
fmt.Printf("F1-Score: %.4f\n", evalMetrics.F1Score())
fmt.Printf("False Positive Rate: %.4f (требование: < 0.10)\n", evalMetrics.FalsePositiveRate())
fmt.Printf("False Negative Rate: %.4f (требование: < 0.05)\n", evalMetrics.FalseNegativeRate())

// Проверяем соответствие требованиям
if evalMetrics.IsAcceptable() {
	fmt.Println("✓ Метрики соответствуют требованиям")
} else {
	fmt.Println("✗ Метрики не соответствуют требованиям")
	recommendations := evalMetrics.GetRecommendations()
	for _, rec := range recommendations {
		fmt.Printf("  - %s\n", rec)
	}
}

// Детальный отчет
fmt.Println("\n" + evalMetrics.DetailedReport())
```

## Пример: Комбинированное использование

```go
// Комплексный анализ дубликатов с использованием всех алгоритмов
func comprehensiveDuplicateAnalysis(items []normalization.DuplicateItem) {
	// 1. Создаем анализатор с продвинутыми методами
	analyzer := normalization.NewDuplicateAnalyzer()
	analyzer.EnableAdvancedMethods(true)
	
	// 2. Анализируем дубликаты
	groups := analyzer.AnalyzeDuplicates(items)
	
	// 3. Оцениваем качество результатов
	evalMetrics := algorithms.NewEvaluationMetrics()
	
	// Симулируем оценку (в реальности это делается на размеченных данных)
	for _, group := range groups {
		// Предполагаем, что группы с высокой уверенностью - правильные
		if group.Confidence > 0.9 {
			evalMetrics.AddResult(true, true) // TP
		} else if group.Confidence > 0.7 {
			// Средняя уверенность - требует проверки
			evalMetrics.AddResult(true, true) // TP (предположительно)
		}
	}
	
	// 4. Выводим результаты
	fmt.Printf("Найдено групп: %d\n", len(groups))
	fmt.Printf("Метрики качества:\n")
	fmt.Printf("  Precision: %.2f%%\n", evalMetrics.Precision()*100)
	fmt.Printf("  Recall: %.2f%%\n", evalMetrics.Recall()*100)
	fmt.Printf("  F1-Score: %.2f%%\n", evalMetrics.F1Score()*100)
}
```

## Пример: Настройка правил для конкретного справочника

```go
// Создаем специализированные правила для справочника контрагентов
func createCounterpartyRules() *algorithms.RuleSet {
	return &algorithms.RuleSet{
		ID:          "counterparty_rules",
		Name:        "Правила для контрагентов",
		ReferenceID: "counterparties",
		Priority:    3,
		Enabled:     true,
		Rules: []algorithms.MatchingRule{
			{
				ID:          "exact_inn",
				Name:        "Точное совпадение ИНН",
				Description: "Точное совпадение по полю ИНН",
				Fields:      []string{"inn"},
				Algorithm:   "exact",
				Threshold:   1.0,
				Weight:      1.0,
				Enabled:     true,
			},
			{
				ID:          "fuzzy_name",
				Name:        "Нечеткое совпадение названия",
				Description: "Нечеткое совпадение по полю name",
				Fields:      []string{"name"},
				Algorithm:   "fuzzy",
				Threshold:   0.90,
				Weight:      0.8,
				Enabled:     true,
			},
			{
				ID:          "phonetic_name",
				Name:        "Фонетическое совпадение",
				Description: "Фонетическое сравнение для обнаружения опечаток",
				Fields:      []string{"name"},
				Algorithm:   "phonetic",
				Threshold:   0.85,
				Weight:      0.6,
				Enabled:     true,
			},
		},
	}
}

// Использование
func main() {
	engine := algorithms.NewRuleEngine()
	ruleSet := createCounterpartyRules()
	engine.RegisterRuleSet(ruleSet)
	
	// Проверяем контрагентов
	record1 := map[string]string{
		"inn":  "1234567890",
		"name": "ООО Рога и Копыта",
	}
	
	record2 := map[string]string{
		"inn":  "1234567890",
		"name": "ООО Рога и Копыта",
	}
	
	similarity, reason, isDuplicate := engine.MatchRecords(record1, record2, "counterparties")
	fmt.Printf("Дубликат: %v, Схожесть: %.2f, Причина: %s\n", isDuplicate, similarity, reason)
}
```

## Рекомендации по использованию

1. **Для точного поиска дублей**: Используйте `exact` алгоритм с порогом 1.0
2. **Для поиска опечаток**: Используйте `phonetic` или `damerau_levenshtein`
3. **Для вариаций написания**: Используйте `ngram` или `jaccard`
4. **Для общего случая**: Используйте `combined` метрику или `HybridSimilarity`
5. **Для оценки качества**: Всегда используйте `EvaluationMetrics` на размеченных данных

## Производительность

- **N-граммы**: Быстрые, O(n*m), подходят для больших объемов
- **Левенштейна**: Средняя скорость, O(n*m), точные результаты
- **Фонетические**: Очень быстрые, O(n), хороши для опечаток
- **Комбинированные**: Медленнее, но точнее, используйте с кешированием

