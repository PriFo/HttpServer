package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"httpserver/database"
	"httpserver/websearch"
)

func main() {
	fmt.Println("=== Тестирование DuckDuckGo поиска для ГОСТов ===")
	fmt.Println()

	// Подключаемся к базе данных ГОСТов
	gostsDB, err := database.NewGostsDB("./gosts.db")
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer gostsDB.Close()

	// Получаем несколько ГОСТов из базы
	fmt.Println("1. Получение ГОСТов из базы данных...")
	gosts, total, err := gostsDB.ListGosts(5, 0, "", "", "", "", "", "")
	if err != nil {
		log.Fatalf("Ошибка получения ГОСТов: %v", err)
	}

	if total == 0 {
		log.Fatal("База данных ГОСТов пуста. Сначала импортируйте ГОСТы.")
	}

	fmt.Printf("   Всего ГОСТов в базе: %d\n", total)
	fmt.Printf("   Получено для тестирования: %d\n\n", len(gosts))

	// Создаем кэш для веб-поиска
	cacheConfig := &websearch.CacheConfig{
		Enabled:         true,
		TTL:             24 * time.Hour,
		CleanupInterval: 6 * time.Hour,
		MaxSize:         1000,
	}
	searchCache := websearch.NewCache(cacheConfig)

	// Создаем клиент DuckDuckGo
	rateLimit := rate.Every(time.Second) // 1 запрос в секунду
	clientConfig := websearch.ClientConfig{
		BaseURL:   "https://api.duckduckgo.com",
		Timeout:   10 * time.Second,
		RateLimit: rateLimit,
		Cache:     searchCache,
	}
	searchClient := websearch.NewClient(clientConfig)

	// Создаем валидаторы
	existenceValidator := websearch.NewProductExistenceValidator(searchClient)
	accuracyValidator := websearch.NewProductAccuracyValidator(searchClient)

	// Тестируем поиск для каждого ГОСТа
	fmt.Println("2. Тестирование поиска через DuckDuckGo API...")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))

	for i, gost := range gosts {
		fmt.Printf("\n[Тест %d/%d] ГОСТ: %s\n", i+1, len(gosts), gost.GostNumber)
		fmt.Printf("Название: %s\n", gost.Title)
		fmt.Println(strings.Repeat("-", 80))

		// Формируем несколько вариантов поисковых запросов
		queries := []string{
			gost.GostNumber, // Только номер
			fmt.Sprintf("%s ГОСТ", gost.GostNumber), // Номер + ГОСТ
			"GOST " + gost.GostNumber,               // Английский вариант
		}

		fmt.Printf("Поисковые запросы:\n")
		for i, q := range queries {
			fmt.Printf("  %d. %s\n", i+1, q)
		}
		fmt.Println()

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Тест 1: Простой поиск (пробуем разные варианты запросов)
		fmt.Println("Тест 1: Простой поиск")
		var result *websearch.SearchResult
		var err error
		var elapsed time.Duration

		// Пробуем разные варианты запросов, пока не найдем результаты
		for _, testQuery := range queries {
			fmt.Printf("  Пробуем запрос: %s\n", testQuery)
			start := time.Now()
			result, err = searchClient.Search(ctx, testQuery)
			elapsed = time.Since(start)

			if err == nil && result.Found && len(result.Results) > 0 {
				fmt.Printf("  ✅ Найдены результаты для запроса: %s\n", testQuery)
				break
			}
			time.Sleep(1 * time.Second) // Небольшая задержка между попытками
		}

		if err != nil {
			fmt.Printf("  ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("  ✅ Успешно (время: %v)\n", elapsed)
			fmt.Printf("  Найдено результатов: %d\n", len(result.Results))
			fmt.Printf("  Найдено: %v\n", result.Found)
			fmt.Printf("  Уверенность: %.2f\n", result.Confidence)
			fmt.Printf("  Источник: %s\n", result.Source)

			if len(result.Results) > 0 {
				fmt.Println("  Первые результаты:")
				for j, item := range result.Results {
					if j >= 3 {
						break
					}
					fmt.Printf("    %d. %s\n", j+1, item.Title)
					if len(item.Snippet) > 100 {
						fmt.Printf("       %s...\n", item.Snippet[:100])
					} else {
						fmt.Printf("       %s\n", item.Snippet)
					}
					fmt.Printf("       URL: %s\n", item.URL)
					fmt.Printf("       Релевантность: %.2f\n", item.Relevance)
				}
			}
		}

		// Небольшая задержка между запросами
		time.Sleep(2 * time.Second)

		// Тест 2: Проверка существования
		fmt.Println()
		fmt.Println("Тест 2: Проверка существования товара")
		start := time.Now()
		validation, err := existenceValidator.Validate(ctx, gost.GostNumber)
		elapsed = time.Since(start)

		if err != nil {
			fmt.Printf("  ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("  ✅ Успешно (время: %v)\n", elapsed)
			fmt.Printf("  Статус: %s\n", validation.Status)
			fmt.Printf("  Найдено: %v\n", validation.Found)
			fmt.Printf("  Оценка: %.2f\n", validation.Score)
			fmt.Printf("  Сообщение: %s\n", validation.Message)
			if validation.Provider != "" {
				fmt.Printf("  Провайдер: %s\n", validation.Provider)
			}
		}

		// Небольшая задержка между запросами
		time.Sleep(2 * time.Second)

		// Тест 3: Проверка точности данных
		fmt.Println()
		fmt.Println("Тест 3: Проверка точности данных")
		start = time.Now()
		accuracyValidation, err := accuracyValidator.Validate(ctx, gost.GostNumber, gost.Title)
		elapsed = time.Since(start)

		if err != nil {
			fmt.Printf("  ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("  ✅ Успешно (время: %v)\n", elapsed)
			fmt.Printf("  Статус: %s\n", accuracyValidation.Status)
			fmt.Printf("  Найдено: %v\n", accuracyValidation.Found)
			fmt.Printf("  Оценка точности: %.2f\n", accuracyValidation.Score)
			fmt.Printf("  Сообщение: %s\n", accuracyValidation.Message)
			if len(accuracyValidation.Details) > 0 {
				fmt.Printf("  Детали:\n")
				for key, value := range accuracyValidation.Details {
					fmt.Printf("    %s: %v\n", key, value)
				}
			}
		}

		fmt.Println(strings.Repeat("=", 80))

		// Задержка между ГОСТами
		if i < len(gosts)-1 {
			time.Sleep(3 * time.Second)
		}
	}

	// Выводим статистику кэша
	fmt.Println()
	fmt.Println("3. Статистика кэша:")
	cacheStats := searchCache.GetStats()
	statsJSON, _ := json.MarshalIndent(cacheStats, "  ", "  ")
	fmt.Println(string(statsJSON))

	fmt.Println()
	fmt.Println("=== Тестирование завершено ===")
}
