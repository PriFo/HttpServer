package pipeline_normalization

// Пример использования pipeline нормализации
//
// Пример 1: Базовое использование с конфигурацией по умолчанию
//
//	config := NewDefaultConfig()
//	pipeline, err := NewNormalizationPipeline(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	result, err := pipeline.Normalize("Кабель ВВГ 3x2.5", "Кабель ВВГ 3x2,5")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Схожесть: %.2f\n", result.Similarity.OverallSimilarity)
//	fmt.Printf("Дубликат: %v\n", result.Similarity.IsDuplicate)
//
// Пример 2: Использование быстрой конфигурации
//
//	config := NewFastConfig()
//	pipeline, _ := NewNormalizationPipeline(config)
//	result, _ := pipeline.Normalize("текст1", "текст2")
//
// Пример 3: Использование точной конфигурации
//
//	config := NewPreciseConfig()
//	pipeline, _ := NewNormalizationPipeline(config)
//	result, _ := pipeline.Normalize("текст1", "текст2")
//
// Пример 4: Пользовательская конфигурация
//
//	config := &NormalizationPipelineConfig{
//		Algorithms: []AlgorithmConfig{
//			{
//				Type:      AlgorithmDamerauLevenshtein,
//				Enabled:   true,
//				Weight:    0.5,
//				Threshold: 0.85,
//				Params:    make(map[string]interface{}),
//			},
//			{
//				Type:      AlgorithmJaccard,
//				Enabled:   true,
//				Weight:    0.5,
//				Threshold: 0.75,
//				Params: map[string]interface{}{
//					"use_ngrams": false,
//				},
//			},
//		},
//		MinSimilarity:     0.85,
//		CombineMethod:     "weighted",
//		ParallelExecution: true,
//		CacheEnabled:      true,
//	}
//
//	pipeline, _ := NewNormalizationPipeline(config)
//	result, _ := pipeline.Normalize("текст1", "текст2")
//
// Пример 5: Обработка батча строк
//
//	pairs := [][]string{
//		{"текст1", "текст2"},
//		{"текст3", "текст4"},
//	}
//
//	batchResult, _ := pipeline.BatchNormalize(pairs)
//	fmt.Printf("Обработано: %d пар\n", batchResult.TotalProcessed)
//	fmt.Printf("Найдено дубликатов: %d\n", batchResult.DuplicatesFound)

