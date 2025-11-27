package algorithms

import (
	"fmt"
	"log"
	"strings"
)

// ExampleHybridSimilarity демонстрирует использование гибридного метода схожести
func ExampleHybridSimilarity() {
	// Примеры пар строк для сравнения
	testPairs := []struct {
		s1, s2   string
		expected bool // ожидается ли дубликат
	}{
		{"ООО Рога и Копыта", "ООО Рога и Копыта", true},
		{"ООО Рога и Копыта", "Рога и Копыта ООО", true},
		{"ООО Рога и Копыта", "ООО Рога и Копыта Лтд", false},
		{"Кабель ВВГнг 3x2.5", "Кабель ВВГнг 3x2.5", true},
		{"Кабель ВВГнг 3x2.5", "Кабель ВВГ 3x2.5", true},
		{"Кабель ВВГнг 3x2.5", "Провод ПВС 3x2.5", false},
		{"Иванов Иван Иванович", "Иванов И.И.", true},
		{"Иванов Иван Иванович", "Петров Петр Петрович", false},
	}

	// Используем гибридный метод с весами по умолчанию
	weights := DefaultSimilarityWeights()
	evaluator := NewAdvancedSimilarityEvaluator()
	threshold := 0.75

	fmt.Println("=== Тестирование гибридного метода схожести ===")
	fmt.Println()

	for i, pair := range testPairs {
		similarity := HybridSimilarityAdvanced(pair.s1, pair.s2, weights)
		isDuplicate := similarity >= threshold

		// Оцениваем результат
		evaluator.EvaluatePair(pair.s1, pair.s2, threshold, pair.expected)

		status := "✓"
		if isDuplicate != pair.expected {
			status = "✗"
		}

		fmt.Printf("%d. %s\n", i+1, status)
		fmt.Printf("   Строка 1: %s\n", pair.s1)
		fmt.Printf("   Строка 2: %s\n", pair.s2)
		fmt.Printf("   Схожесть: %.4f\n", similarity)
		fmt.Printf("   Дубликат: %v (ожидалось: %v)\n\n", isDuplicate, pair.expected)
	}

	// Выводим метрики
	metrics := evaluator.GetMetrics()
	fmt.Println("=== Метрики оценки ===")
	fmt.Println(metrics.DetailedReport())
	fmt.Println("\n=== Рекомендации ===")
	recommendations := metrics.GetRecommendations()
	for _, rec := range recommendations {
		fmt.Printf("- %s\n", rec)
	}
}

// ExampleCustomWeights демонстрирует использование пользовательских весов
func ExampleCustomWeights() {
	s1 := "ООО Рога и Копыта"
	s2 := "Рога и Копыта ООО"

	// Веса по умолчанию
	defaultWeights := DefaultSimilarityWeights()
	similarity1 := HybridSimilarityAdvanced(s1, s2, defaultWeights)

	// Пользовательские веса (больше внимания фонетике)
	customWeights := &SimilarityWeights{
		JaroWinkler: 0.2,
		LCS:         0.1,
		Phonetic:    0.5, // Увеличиваем вес фонетики
		Ngram:       0.1,
		Jaccard:     0.1,
	}
	customWeights.NormalizeWeights() // Нормализуем веса
	similarity2 := HybridSimilarityAdvanced(s1, s2, customWeights)

	fmt.Printf("Строка 1: %s\n", s1)
	fmt.Printf("Строка 2: %s\n", s2)
	fmt.Printf("Схожесть (веса по умолчанию): %.4f\n", similarity1)
	fmt.Printf("Схожесть (пользовательские веса): %.4f\n", similarity2)
}

// ExampleAlgorithmEvaluation демонстрирует оценку различных алгоритмов
func ExampleAlgorithmEvaluation() {
	// Тестовые пары с известными результатами
	testPairs := []SimilarityTestPair{
		{"ООО Рога и Копыта", "ООО Рога и Копыта", true},
		{"ООО Рога и Копыта", "Рога и Копыта ООО", true},
		{"ООО Рога и Копыта", "ООО Рога и Копыта Лтд", false},
		{"Кабель ВВГнг 3x2.5", "Кабель ВВГ 3x2.5", true},
		{"Кабель ВВГнг 3x2.5", "Провод ПВС 3x2.5", false},
	}

	threshold := 0.75

	fmt.Println("=== Сравнение алгоритмов ===")
	fmt.Println()

	// 1. Jaro-Winkler
	jwMetrics := EvaluateAlgorithm(testPairs, threshold, JaroWinklerSimilarityAdvanced)
	fmt.Println("Jaro-Winkler:")
	fmt.Printf("  Precision: %.4f, Recall: %.4f, F1: %.4f\n\n",
		jwMetrics.Precision(), jwMetrics.Recall(), jwMetrics.F1Score())

	// 2. LCS
	lcsMetrics := EvaluateAlgorithm(testPairs, threshold, LCSSimilarityAdvanced)
	fmt.Println("LCS:")
	fmt.Printf("  Precision: %.4f, Recall: %.4f, F1: %.4f\n\n",
		lcsMetrics.Precision(), lcsMetrics.Recall(), lcsMetrics.F1Score())

	// 3. Гибридный метод
	hybridFunc := func(s1, s2 string) float64 {
		return HybridSimilarityAdvanced(s1, s2, DefaultSimilarityWeights())
	}
	hybridMetrics := EvaluateAlgorithm(testPairs, threshold, hybridFunc)
	fmt.Println("Гибридный метод:")
	fmt.Printf("  Precision: %.4f, Recall: %.4f, F1: %.4f\n\n",
		hybridMetrics.Precision(), hybridMetrics.Recall(), hybridMetrics.F1Score())

	// Сравнение метрик
	fmt.Println("=== Сравнение метрик ===")
	fmt.Println(CompareMetrics(hybridMetrics, jwMetrics))
}

// RunExamples запускает все примеры
func RunExamples() {
	log.Println("Запуск примеров использования advanced similarity...")

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Пример 1: Гибридный метод схожести")
	fmt.Println(strings.Repeat("=", 60))
	ExampleHybridSimilarity()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Пример 2: Пользовательские веса")
	fmt.Println(strings.Repeat("=", 60))
	ExampleCustomWeights()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Пример 3: Оценка алгоритмов")
	fmt.Println(strings.Repeat("=", 60))
	ExampleAlgorithmEvaluation()
}

