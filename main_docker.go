//go:build docker
// +build docker

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"httpserver/database"
	"httpserver/internal/config"
	"httpserver/server"
)

func main() {
	log.Println("Запуск 1C HTTP Server (Docker режим без GUI)...")
	
	// Загружаем конфигурацию
	config, err := server.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}
	
	// Определяем путь к основной БД
	// Используем 1c_data.db если существует, иначе data.db
	dbPath := config.DatabasePath
	if _, err := os.Stat("1c_data.db"); err == nil {
		dbPath = "1c_data.db"
		log.Printf("Используется существующая база данных: %s", dbPath)
	}
	
	// Создаем конфигурацию для БД
	dbConfig := database.DBConfig{
		MaxOpenConns:    config.MaxOpenConns,
		MaxIdleConns:    config.MaxIdleConns,
		ConnMaxLifetime: config.ConnMaxLifetime,
	}
	
	// Создаем базу данных
	db, err := database.NewDBWithConfig(dbPath, dbConfig)
	if err != nil {
		log.Fatalf("Ошибка создания базы данных: %v", err)
	}
	defer db.Close()
	
	// Создаем базу данных для нормализованных данных
	normalizedDBPath := config.NormalizedDatabasePath
	normalizedDB, err := database.NewDBWithConfig(normalizedDBPath, dbConfig)
	if err != nil {
		log.Fatalf("Ошибка создания нормализованной базы данных: %v", err)
	}
	defer normalizedDB.Close()
	log.Printf("Используется нормализованная база данных: %s", normalizedDBPath)
	
	// Создаем сервисную базу данных для системной информации
	serviceDBPath := config.ServiceDatabasePath
	serviceDB, err := database.NewServiceDBWithConfig(serviceDBPath, dbConfig)
	if err != nil {
		log.Fatalf("Ошибка создания сервисной базы данных: %v", err)
	}
	defer serviceDB.Close()
	log.Printf("Используется сервисная база данных: %s", serviceDBPath)
	
	// Перезагружаем конфигурацию из сервисной БД (если есть)
	config, err = server.LoadConfig(serviceDB)
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации из БД: %v", err)
	}

	// Если конфигурации нет в БД, сохраняем текущую из env
	configJSON, _ := serviceDB.GetAppConfig()
	if configJSON == "" {
		log.Printf("Config not found in DB, saving current config from environment")
		if err := server.SaveConfig(config, serviceDB); err != nil {
			log.Printf("Warning: failed to save config to DB: %v", err)
		} else {
			log.Printf("Config saved to service database")
		}
	}
	
	// Создаем сервер с обеими БД и сервисной БД
	srv := server.NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, config)
	
	// Запускаем сервер в отдельной горутине
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Fatalf("✗ КРИТИЧЕСКАЯ ОШИБКА: Паника при запуске сервера: %v", r)
			}
		}()
		if err := srv.Start(); err != nil {
			log.Fatalf("✗ КРИТИЧЕСКАЯ ОШИБКА: Ошибка запуска сервера: %v", err)
		}
	}()
	
	// Статистика больше не выводится в консоль каждые 5 секунд
	// Используйте API /api/database/info для получения статистики при необходимости
	
	// Обработка сигналов для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Println("═══════════════════════════════════════════════════════")
		log.Println("⏹  Получен сигнал завершения, останавливаю сервер...")
		
		// Graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("✗ Ошибка при остановке сервера: %v", err)
		} else {
			log.Println("✓ Сервер успешно остановлен")
		}
		
		cancel()
		os.Exit(0)
	}()
	
	log.Println("═══════════════════════════════════════════════════════")
	log.Printf("✓ Сервер успешно запущен на порту %s", config.Port)
	log.Printf("✓ API доступно: http://localhost:%s", config.Port)
	log.Printf("✓ База данных: %s", dbPath)
	log.Printf("✓ Нормализованная БД: %s", normalizedDBPath)
	log.Printf("✓ Сервисная БД: %s", serviceDBPath)
	log.Println("✓ Режим: Docker контейнер (без GUI)")
	log.Println("  Для остановки нажмите Ctrl+C")
	log.Println("═══════════════════════════════════════════════════════")
	
	// Блокируем выполнение
	<-ctx.Done()
}

