package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/websearch"
	"golang.org/x/time/rate"
)

func main() {
	fmt.Println("=== Тестирование поиска ГОСТов о сварке через DuckDuckGo ===")
	fmt.Println()

	// Подключаемся к базе данных ГОСТов
	gostsDB, err := database.NewGostsDB("./gosts.db")
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer gostsDB.Close()

	// Ищем ГОСТы о сварке в базе
	fmt.Println("1. Поиск ГОСТов о сварке в базе данных...")
	query := "свар"
	
	// Используем прямой SQL-запрос для избежания проблем с NULL
	db := gostsDB.GetDB()
	searchQuery := `
		SELECT id, gost_number, title, adoption_date, effective_date, status,
		       source_type, source_id, source_url, description, keywords,
		       COALESCE(created_at, updated_at) as created_at, updated_at
		FROM gosts
		WHERE gost_number LIKE ? OR title LIKE ? OR keywords LIKE ?
		ORDER BY gost_number
		LIMIT ? OFFSET ?
	`
	
	searchPattern := "%" + query + "%"
	rows, err := db.Query(searchQuery, searchPattern, searchPattern, searchPattern, 10, 0)
	if err != nil {
		log.Fatalf("Ошибка поиска ГОСТов: %v", err)
	}
	defer rows.Close()

	var gosts []*database.Gost
	for rows.Next() {
		gost := &database.Gost{}
		var adoptionDate, effectiveDate sql.NullTime
		var sourceID sql.NullInt64
		var createdAtStr sql.NullString
		var updatedAtStr sql.NullString

		err := rows.Scan(
			&gost.ID, &gost.GostNumber, &gost.Title,
			&adoptionDate, &effectiveDate,
			&gost.Status, &gost.SourceType, &sourceID,
			&gost.SourceURL, &gost.Description, &gost.Keywords,
			&createdAtStr, &updatedAtStr,
		)
		if err != nil {
			log.Printf("Ошибка сканирования: %v", err)
			continue
		}

		// Парсим даты из строк
		if createdAtStr.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr.String); err == nil {
				gost.CreatedAt = t
			} else if t, err := time.Parse(time.RFC3339, createdAtStr.String); err == nil {
				gost.CreatedAt = t
			}
		}
		if updatedAtStr.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", updatedAtStr.String); err == nil {
				gost.UpdatedAt = t
			} else if t, err := time.Parse(time.RFC3339, updatedAtStr.String); err == nil {
				gost.UpdatedAt = t
			}
		}
		if !createdAtStr.Valid || gost.CreatedAt.IsZero() {
			gost.CreatedAt = gost.UpdatedAt
		}

		if adoptionDate.Valid {
			gost.AdoptionDate = &adoptionDate.Time
		}
		if effectiveDate.Valid {
			gost.EffectiveDate = &effectiveDate.Time
		}
		if sourceID.Valid {
			id := int(sourceID.Int64)
			gost.SourceID = &id
		}

		gosts = append(gosts, gost)
	}
	
	total := len(gosts) // Упрощенная версия, можно сделать отдельный COUNT запрос

	if total == 0 {
		log.Fatal("ГОСТы о сварке не найдены в базе данных.")
	}

	fmt.Printf("   Найдено ГОСТов о сварке: %d\n", total)
	fmt.Printf("   Показано: %d\n\n", len(gosts))

	// Выводим найденные ГОСТы
	fmt.Println("Найденные ГОСТы о сварке:")
	for i, gost := range gosts {
		fmt.Printf("  %d. %s - %s\n", i+1, gost.GostNumber, gost.Title)
	}
	fmt.Println()

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
		Timeout:   15 * time.Second,
		RateLimit: rateLimit,
		Cache:     searchCache,
	}
	searchClient := websearch.NewClient(clientConfig)

	// Создаем валидаторы
	existenceValidator := websearch.NewProductExistenceValidator(searchClient)
	accuracyValidator := websearch.NewProductAccuracyValidator(searchClient)

	// Тестируем поиск для нескольких ГОСТов о сварке
	fmt.Println("2. Тестирование поиска через DuckDuckGo HTML-поиск...")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))

	// Берем первые 3 ГОСТа для тестирования
	testGosts := gosts
	if len(gosts) > 3 {
		testGosts = gosts[:3]
	}

	for i, gost := range testGosts {
		fmt.Printf("\n[Тест %d/%d] ГОСТ: %s\n", i+1, len(testGosts), gost.GostNumber)
		fmt.Printf("Название: %s\n", gost.Title)
		fmt.Println(strings.Repeat("-", 80))

		// Формируем поисковый запрос
		query := fmt.Sprintf("%s %s", gost.GostNumber, gost.Title)
		if len(query) > 200 {
			query = query[:200] + "..."
		}

		fmt.Printf("Поисковый запрос: %s\n\n", query)

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// Тест 1: HTML-поиск
		fmt.Println("Тест 1: HTML-поиск через DuckDuckGo")
		start := time.Now()
		result, err := searchClient.SearchHTML(ctx, query)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("  ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("  ✅ Успешно (время: %v)\n", elapsed)
			fmt.Printf("  Найдено результатов: %d\n", len(result.Results))
			fmt.Printf("  Найдено: %v\n", result.Found)
			fmt.Printf("  Уверенность: %.2f\n", result.Confidence)
			fmt.Printf("  Источник: %s\n", result.Source)

			if len(result.Results) > 0 {
				fmt.Println("  Первые 5 результатов:")
				for j, item := range result.Results {
					if j >= 5 {
						break
					}
					fmt.Printf("    %d. %s\n", j+1, item.Title)
					if len(item.Snippet) > 150 {
						fmt.Printf("       %s...\n", item.Snippet[:150])
					} else {
						fmt.Printf("       %s\n", item.Snippet)
					}
					fmt.Printf("       URL: %s\n", item.URL)
					fmt.Printf("       Релевантность: %.2f\n", item.Relevance)
					fmt.Println()
				}
			}
		}

		// Небольшая задержка между запросами
		time.Sleep(3 * time.Second)

		// Тест 2: Проверка существования
		fmt.Println()
		fmt.Println("Тест 2: Проверка существования через валидатор")
		start = time.Now()
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
			if len(validation.Results) > 0 {
				fmt.Printf("  Найдено результатов валидации: %d\n", len(validation.Results))
			}
		}

		// Небольшая задержка между запросами
		time.Sleep(3 * time.Second)

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
		if i < len(testGosts)-1 {
			time.Sleep(4 * time.Second)
		}
	}

	// Выводим статистику кэша
	fmt.Println()
	fmt.Println("3. Статистика кэша:")
	cacheStats := searchCache.GetStats()
	statsJSON, _ := json.MarshalIndent(cacheStats, "  ", "  ")
	fmt.Println(string(statsJSON))

	// Тест 4: Поиск по общему запросу "ГОСТ сварка"
	fmt.Println()
	fmt.Println("4. Тест общего поиска: 'ГОСТ сварка'")
	fmt.Println(strings.Repeat("=", 80))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	generalQuery := "ГОСТ сварка"
	fmt.Printf("Поисковый запрос: %s\n\n", generalQuery)

	start := time.Now()
	generalResult, err := searchClient.SearchHTML(ctx, generalQuery)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
	} else {
		fmt.Printf("✅ Успешно (время: %v)\n", elapsed)
		fmt.Printf("Найдено результатов: %d\n", len(generalResult.Results))
		fmt.Printf("Найдено: %v\n", generalResult.Found)
		fmt.Printf("Уверенность: %.2f\n", generalResult.Confidence)
		fmt.Printf("Источник: %s\n", generalResult.Source)

		if len(generalResult.Results) > 0 {
			fmt.Println()
			fmt.Println("Первые 10 результатов:")
			for j, item := range generalResult.Results {
				if j >= 10 {
					break
				}
				fmt.Printf("  %d. %s\n", j+1, item.Title)
				fmt.Printf("     URL: %s\n", item.URL)
				fmt.Printf("     Релевантность: %.2f\n", item.Relevance)
				fmt.Println()
			}
		}
	}

	fmt.Println()
	fmt.Println("=== Тестирование завершено ===")
}

