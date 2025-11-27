package main

import (
	"fmt"
	"os"

	"httpserver/internal/config"
)

func main() {
	fmt.Println("=== Проверка конфигурации ===")
	fmt.Println("")

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Конфигурация успешно загружена")
	fmt.Println("")

	// Выводим основные настройки
	fmt.Println("Основные настройки:")
	fmt.Printf("  Порт: %s\n", cfg.Port)
	fmt.Printf("  Основная БД: %s\n", cfg.DatabasePath)
	fmt.Printf("  Нормализованная БД: %s\n", cfg.NormalizedDatabasePath)
	fmt.Printf("  Сервисная БД: %s\n", cfg.ServiceDatabasePath)
	fmt.Println("")

	// Выводим настройки connection pooling
	fmt.Println("Connection Pooling:")
	fmt.Printf("  Max Open Connections: %d\n", cfg.MaxOpenConns)
	fmt.Printf("  Max Idle Connections: %d\n", cfg.MaxIdleConns)
	fmt.Printf("  Connection Max Lifetime: %v\n", cfg.ConnMaxLifetime)
	fmt.Println("")

	// Выводим AI настройки
	fmt.Println("AI Configuration:")
	if cfg.ArliaiAPIKey != "" {
		fmt.Printf("  Arliai API Key: [установлен]\n")
	} else {
		fmt.Printf("  Arliai API Key: [не установлен]\n")
	}
	fmt.Printf("  Arliai Model: %s\n", cfg.ArliaiModel)
	fmt.Printf("  AI Timeout: %v\n", cfg.AITimeout)
	fmt.Println("")

	// Выводим настройки мульти-провайдера
	fmt.Println("Multi-Provider:")
	fmt.Printf("  Enabled: %v\n", cfg.MultiProviderEnabled)
	fmt.Printf("  Aggregation Strategy: %s\n", cfg.AggregationStrategy)
	fmt.Println("")

	// Выводим настройки обогащения
	if cfg.Enrichment != nil {
		fmt.Println("Enrichment:")
		fmt.Printf("  Enabled: %v\n", cfg.Enrichment.Enabled)
		fmt.Printf("  Auto Enrich: %v\n", cfg.Enrichment.AutoEnrich)
		fmt.Printf("  Min Quality Score: %.2f\n", cfg.Enrichment.MinQualityScore)
		fmt.Printf("  Services: %d\n", len(cfg.Enrichment.Services))
		if cfg.Enrichment.Cache != nil {
			fmt.Printf("  Cache Enabled: %v\n", cfg.Enrichment.Cache.Enabled)
			fmt.Printf("  Cache TTL: %v\n", cfg.Enrichment.Cache.TTL)
		}
		fmt.Println("")
	}

	// Проверяем валидацию
	if err := cfg.Validate(); err != nil {
		fmt.Printf("⚠️  Предупреждения валидации: %v\n", err)
		fmt.Println("")
	} else {
		fmt.Println("✅ Валидация пройдена успешно")
		fmt.Println("")
	}

	fmt.Println("=== Проверка завершена ===")
}

