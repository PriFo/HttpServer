package examples

import (
	"fmt"
	"httpserver/normalization"
	"httpserver/normalization/algorithms"
)

// DemoPhase1Improvements демонстрирует улучшения Фазы 1
func DemoPhase1Improvements() {
	fmt.Println("=== Демонстрация улучшений Фазы 1 ===")

	// 1. Демонстрация лемматизации
	demoLemmatization()

	// 2. Демонстрация NER
	demoNER()

	// 3. Демонстрация префиксной фильтрации
	demoPrefixFiltering()

	// 4. Демонстрация интеграции
	demoIntegration()
}

// demoLemmatization демонстрирует работу лемматизации
func demoLemmatization() {
	fmt.Println("1. Лемматизация:")
	fmt.Println("---")

	lem := algorithms.NewRussianLemmatizer()

	examples := []string{
		"маслами",
		"сливочного",
		"кабеля",
		"молотком",
		"дерева",
	}

	for _, word := range examples {
		lemma := lem.Lemmatize(word)
		fmt.Printf("  %s → %s\n", word, lemma)
	}

	fmt.Println()
}

// demoNER демонстрирует работу NER
func demoNER() {
	fmt.Println("2. Named Entity Recognition (NER):")
	fmt.Println("---")

	ner := algorithms.NewRussianNER()

	text := "стальной белый кабель 100x200 5кг многожильный"
	entities := ner.ExtractEntities(text)

	fmt.Printf("  Текст: %s\n", text)
	fmt.Printf("  Найдено сущностей: %d\n", len(entities))
	for _, entity := range entities {
		fmt.Printf("    - %s: %s (уверенность: %.2f)\n", entity.Type, entity.Text, entity.Confidence)
	}

	// BIO-тегирование
	tokens := ner.TagWithBIO(text)
	fmt.Println("\n  BIO-тегирование:")
	for _, token := range tokens {
		if token.Tag != "O" {
			fmt.Printf("    %s [%s-%s]\n", token.Token, token.Tag, token.EntityType)
		}
	}

	fmt.Println()
}

// demoPrefixFiltering демонстрирует работу префиксной фильтрации
func demoPrefixFiltering() {
	fmt.Println("3. Префиксная фильтрация:")
	fmt.Println("---")

	index := algorithms.NewPrefixIndex(3, 3)

	texts := []string{
		"масло сливочное",
		"масло подсолнечное",
		"кабель медный",
		"кабель алюминиевый",
		"шкаф деревянный",
	}

	fmt.Println("  Добавление в индекс:")
	for i, text := range texts {
		index.Add(i, text)
		fmt.Printf("    [%d] %s\n", i, text)
	}

	stats := index.GetStats()
	fmt.Printf("\n  Статистика индекса:\n")
	fmt.Printf("    Всего элементов: %d\n", stats.TotalItems)
	fmt.Printf("    Всего префиксов: %d\n", stats.TotalPrefixes)
	fmt.Printf("    Среднее элементов на префикс: %.2f\n", stats.AvgItemsPerPrefix)

	fmt.Printf("\n  Поиск кандидатов для 'масло сливочное':\n")
	candidates := index.GetCandidates(0, "масло сливочное")
	fmt.Printf("    Найдено кандидатов: %d\n", len(candidates))
	for _, idx := range candidates {
		fmt.Printf("      - [%d] %s\n", idx, texts[idx])
	}

	fmt.Println()
}

// demoIntegration демонстрирует интеграцию всех улучшений
func demoIntegration() {
	fmt.Println("4. Интеграция всех улучшений:")
	fmt.Println("---")

	// Создаем анализатор дубликатов
	analyzer := normalization.NewDuplicateAnalyzer()

	// Тестовые данные
	items := []normalization.DuplicateItem{
		{
			ID:             1,
			Code:           "001",
			NormalizedName: "маслами сливочного",
			Category:       "продукты",
		},
		{
			ID:             2,
			Code:           "002",
			NormalizedName: "масло сливочное",
			Category:       "продукты",
		},
		{
			ID:             3,
			Code:           "003",
			NormalizedName: "кабеля медного",
			Category:       "электротехника",
		},
		{
			ID:             4,
			Code:           "004",
			NormalizedName: "кабель медный",
			Category:       "электротехника",
		},
	}

	fmt.Println("  Поиск дубликатов:")
	groups := analyzer.AnalyzeDuplicates(items)

	fmt.Printf("    Найдено групп: %d\n", len(groups))
	for i, group := range groups {
		fmt.Printf("\n    Группа %d:\n", i+1)
		fmt.Printf("      Тип: %s\n", group.Type)
		fmt.Printf("      Уверенность: %.2f\n", group.Confidence)
		fmt.Printf("      Элементы:\n")
		for _, item := range group.Items {
			fmt.Printf("        - [%d] %s\n", item.ID, item.NormalizedName)
		}
	}

	// Демонстрация NER в NameNormalizer
	fmt.Println("\n  Извлечение атрибутов с NER:")
	normalizer := normalization.NewNameNormalizer()
	
	testNames := []string{
		"стальной белый кабель 100x200",
		"многожильный медный кабель 2.5мм",
		"деревянный шкаф 150x200",
	}

	for _, name := range testNames {
		normalized, attrs := normalizer.ExtractAttributesWithNER(name)
		fmt.Printf("\n    Название: %s\n", name)
		fmt.Printf("    Нормализовано: %s\n", normalized)
		fmt.Printf("    Атрибуты (%d):\n", len(attrs))
		for _, attr := range attrs {
			fmt.Printf("      - %s: %s", attr.AttributeName, attr.AttributeValue)
			if attr.Unit != "" {
				fmt.Printf(" %s", attr.Unit)
			}
			fmt.Printf(" (уверенность: %.2f)\n", attr.Confidence)
		}
	}

	fmt.Println()
}

