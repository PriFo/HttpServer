package algorithms

import (
	"fmt"
)

// ExampleUsage демонстрирует использование всех алгоритмов нормализации
func ExampleUsage() {
	fmt.Println("=== Примеры использования алгоритмов нормализации НСИ ===")
	fmt.Println()

	// 1. N-граммы
	fmt.Println("1. Сравнение N-грамм:")
	ngramGen := NewNGramGenerator(2)
	similarity := ngramGen.Similarity("молоток строительный", "молотак строительный")
	fmt.Printf("   Схожесть 'молоток' и 'молотак': %.2f\n\n", similarity)

	// 2. Фонетические алгоритмы
	fmt.Println("2. Фонетические алгоритмы:")
	phoneticMatcher := NewPhoneticMatcher()
	soundexCode := phoneticMatcher.EncodeSoundex("молоток")
	metaphoneCode := phoneticMatcher.EncodeMetaphone("молоток")
	phoneticSim := phoneticMatcher.Similarity("молоток", "молотак")
	fmt.Printf("   Soundex код 'молоток': %s\n", soundexCode)
	fmt.Printf("   Metaphone код 'молоток': %s\n", metaphoneCode)
	fmt.Printf("   Фонетическая схожесть 'молоток' и 'молотак': %.2f\n\n", phoneticSim)

	// 3. Метрики схожести
	fmt.Println("3. Метрики схожести:")
	metrics := NewSimilarityMetrics()
	jaccard := metrics.JaccardIndex("молоток строительный", "молоток")
	cosine := metrics.CosineSimilarity("молоток", "молотак")
	levenshtein := metrics.LevenshteinSimilarity("молоток", "молотак")
	combined := metrics.CombinedSimilarity("молоток", "молотак")
	fmt.Printf("   Индекс Жаккара: %.2f\n", jaccard)
	fmt.Printf("   Косинусное сходство: %.2f\n", cosine)
	fmt.Printf("   Левенштейна: %.2f\n", levenshtein)
	fmt.Printf("   Комбинированная: %.2f\n\n", combined)

	// 4. Система правил
	fmt.Println("4. Система правил сопоставления:")
	engine := NewRuleEngine()
	ruleSet := CreateNomenclatureRuleSet("test_nomenclature")
	engine.RegisterRuleSet(ruleSet)

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

	similarity, reason, isDuplicate := engine.MatchRecords(record1, record2, "test_nomenclature")
	fmt.Printf("   Схожесть: %.2f\n", similarity)
	fmt.Printf("   Причина: %s\n", reason)
	fmt.Printf("   Является дублем: %v\n\n", isDuplicate)

	// 5. Метрики оценки эффективности
	fmt.Println("5. Метрики оценки эффективности:")
	evalMetrics := NewEvaluationMetrics()

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

	fmt.Printf("   Precision: %.4f\n", evalMetrics.Precision())
	fmt.Printf("   Recall: %.4f\n", evalMetrics.Recall())
	fmt.Printf("   F1-Score: %.4f\n", evalMetrics.F1Score())
	fmt.Printf("   False Positive Rate: %.4f (требование: < 0.10)\n", evalMetrics.FalsePositiveRate())
	fmt.Printf("   False Negative Rate: %.4f (требование: < 0.05)\n", evalMetrics.FalseNegativeRate())
	fmt.Printf("   Соответствует требованиям: %v\n", evalMetrics.IsAcceptable())

	recommendations := evalMetrics.GetRecommendations()
	if len(recommendations) > 0 {
		fmt.Println("\n   Рекомендации:")
		for _, rec := range recommendations {
			fmt.Printf("   - %s\n", rec)
		}
	}
}

