package main

import (
	"fmt"
	"log"

	"httpserver/normalization/algorithms"
)

func main() {
	fmt.Println("=== Пример использования системы схожести ===")
	fmt.Println()

	// Пример 1: Простое сравнение двух строк
	fmt.Println("1. Простое сравнение:")
	s1 := "ООО Рога и Копыта"
	s2 := "Рога и Копыта ООО"
	
	weights := algorithms.DefaultSimilarityWeights()
	similarity := algorithms.HybridSimilarityAdvanced(s1, s2, weights)
	fmt.Printf("   '%s' vs '%s': %.4f\n\n", s1, s2, similarity)

	// Пример 2: Детальный анализ
	fmt.Println("2. Детальный анализ:")
	pairs := []algorithms.SimilarityPair{
		{"ООО Рога и Копыта", "Рога и Копыта ООО"},
		{"Кабель ВВГнг 3x2.5", "Кабель ВВГ 3x2.5"},
		{"ООО Рога и Копыта", "ООО Другая Компания"},
	}

	analyzer := algorithms.NewSimilarityAnalyzer(weights)
	result := analyzer.AnalyzePairs(pairs, 0.75)

	fmt.Printf("   Всего пар: %d\n", result.Statistics.TotalPairs)
	fmt.Printf("   Дубликаты: %d\n", result.Statistics.DuplicatePairs)
	fmt.Printf("   Средняя схожесть: %.4f\n", result.Statistics.AverageSimilarity)
	fmt.Println("   Рекомендации:")
	for i, rec := range result.Recommendations {
		fmt.Printf("     %d. %s\n", i+1, rec)
	}
	fmt.Println()

	// Пример 3: Обучение на данных
	fmt.Println("3. Обучение на размеченных данных:")
	trainingPairs := []algorithms.SimilarityTestPair{
		{"ООО Рога и Копыта", "ООО Рога и Копыта", true},
		{"ООО Рога и Копыта", "Рога и Копыта ООО", true},
		{"ООО Рога и Копыта", "ООО Другая Компания", false},
		{"Кабель ВВГнг", "Кабель ВВГ", true},
		{"Кабель ВВГнг", "Провод ПВС", false},
	}

	learner := algorithms.NewSimilarityLearner()
	learner.AddTrainingPairs(trainingPairs)

	optimizedWeights, err := learner.OptimizeWeights(50, 0.01)
	if err != nil {
		log.Printf("Ошибка оптимизации: %v", err)
	} else {
		fmt.Printf("   Оптимизированные веса:\n")
		fmt.Printf("     Jaro-Winkler: %.2f\n", optimizedWeights.JaroWinkler)
		fmt.Printf("     LCS: %.2f\n", optimizedWeights.LCS)
		fmt.Printf("     Phonetic: %.2f\n", optimizedWeights.Phonetic)
		fmt.Printf("     N-gram: %.2f\n", optimizedWeights.Ngram)
		fmt.Printf("     Jaccard: %.2f\n", optimizedWeights.Jaccard)
	}
	fmt.Println()

	// Пример 4: Поиск оптимального порога
	fmt.Println("4. Поиск оптимального порога:")
	testPairs := []algorithms.SimilarityTestPair{
		{"ООО Рога и Копыта", "ООО Рога и Копыта", true},
		{"ООО Рога и Копыта", "Рога и Копыта ООО", true},
		{"ООО Рога и Копыта", "ООО Другая Компания", false},
	}

	threshold, metrics := learner.GetOptimalThreshold(testPairs, optimizedWeights)
	fmt.Printf("   Оптимальный порог: %.2f\n", threshold)
	fmt.Printf("   Precision: %.4f\n", metrics.Precision())
	fmt.Printf("   Recall: %.4f\n", metrics.Recall())
	fmt.Printf("   F1-score: %.4f\n", metrics.F1Score())
	fmt.Println()

	// Пример 5: Экспорт результатов
	fmt.Println("5. Экспорт результатов:")
	exporter := algorithms.NewSimilarityExporter(result)
	
	// Экспорт в JSON
	if err := exporter.Export("example_export.json", algorithms.ExportFormatJSON); err != nil {
		log.Printf("Ошибка экспорта JSON: %v", err)
	} else {
		fmt.Println("   ✓ Экспортировано в example_export.json")
	}

	// Экспорт отчета
	if err := exporter.ExportReport("example_report.md"); err != nil {
		log.Printf("Ошибка экспорта отчета: %v", err)
	} else {
		fmt.Println("   ✓ Экспортирован отчет в example_report.md")
	}

	fmt.Println()
	fmt.Println("=== Пример завершен ===")
}

