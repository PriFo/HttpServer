//go:build no_gui
// +build no_gui

// @title HTTP Server API
// @version 1.0
// @description API для системы нормализации данных из 1С. Мульти-провайдерная нормализация, AI-классификация, управление качеством данных.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name Internal Use Only
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:9999
// @BasePath /api
// @schemes http https

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"httpserver/database"
	"httpserver/server"
)

func main() {
	log.Println("Запуск 1C HTTP Server (без GUI)...")
	log.Printf("Версия: no_gui build")
	log.Printf("Рабочая директория: %s", getWorkingDir())

	// Загружаем конфигурацию
	log.Println("[1/8] Загрузка конфигурации...")
	config, err := server.LoadConfig()
	if err != nil {
		log.Printf("✗ Ошибка загрузки конфигурации: %v", err)
		log.Fatalf("Не удалось загрузить конфигурацию из переменных окружения")
	}
	log.Printf("✓ Конфигурация загружена. Порт: %s", config.Port)

	// Определяем путь к основной БД
	// Используем 1c_data.db если существует, иначе data.db
	dbPath := config.DatabasePath
	if _, err := os.Stat("1c_data.db"); err == nil {
		dbPath = "1c_data.db"
		log.Printf("Используется существующая база данных: %s", dbPath)
	}
	log.Printf("Путь к основной БД: %s", dbPath)
	log.Printf("Путь к нормализованной БД: %s", config.NormalizedDatabasePath)
	log.Printf("Путь к сервисной БД: %s", config.ServiceDatabasePath)

	// Создаем конфигурацию для БД
	dbConfig := database.DBConfig{
		MaxOpenConns:    config.MaxOpenConns,
		MaxIdleConns:    config.MaxIdleConns,
		ConnMaxLifetime: config.ConnMaxLifetime,
	}

	// Создаем базу данных
	log.Println("[2/8] Инициализация основной базы данных...")
	db, err := database.NewDBWithConfig(dbPath, dbConfig)
	if err != nil {
		log.Printf("✗ Ошибка создания базы данных: %v", err)
		log.Fatalf("Не удалось инициализировать основную базу данных по пути: %s", dbPath)
	}
	defer db.Close()
	log.Printf("✓ Основная БД инициализирована: %s", dbPath)

	// Создаем базу данных для нормализованных данных
	log.Println("[3/8] Инициализация нормализованной базы данных...")
	normalizedDBPath := config.NormalizedDatabasePath
	normalizedDB, err := database.NewDBWithConfig(normalizedDBPath, dbConfig)
	if err != nil {
		log.Printf("✗ Ошибка создания базы нормализованных данных: %v", err)
		log.Fatalf("Не удалось инициализировать нормализованную базу данных по пути: %s", normalizedDBPath)
	}
	defer normalizedDB.Close()
	log.Printf("✓ Нормализованная БД инициализирована: %s", normalizedDBPath)

	// Создаем сервисную базу данных для системной информации
	log.Println("[4/8] Инициализация сервисной базы данных...")
	serviceDBPath := config.ServiceDatabasePath
	serviceDB, err := database.NewServiceDBWithConfig(serviceDBPath, dbConfig)
	if err != nil {
		log.Printf("✗ Ошибка создания сервисной базы данных: %v", err)
		log.Fatalf("Не удалось инициализировать сервисную базу данных по пути: %s", serviceDBPath)
	}
	defer serviceDB.Close()
	log.Printf("✓ Сервисная БД инициализирована: %s", serviceDBPath)

	// Перезагружаем конфигурацию из сервисной БД (если есть)
	log.Println("[5/8] Загрузка конфигурации из БД...")
	config, err = server.LoadConfig(serviceDB)
	if err != nil {
		log.Printf("✗ Ошибка загрузки конфигурации из БД: %v", err)
		log.Printf("⚠ Используется конфигурация из переменных окружения")
		// Не делаем fatal - используем конфигурацию из env
	} else {
		log.Printf("✓ Конфигурация загружена из БД")
	}

	// Если конфигурации нет в БД, сохраняем текущую из env
	configJSON, _ := serviceDB.GetAppConfig()
	if configJSON == "" {
		log.Printf("[6/8] Сохранение конфигурации в БД...")
		if err := server.SaveConfig(config, serviceDB); err != nil {
			log.Printf("⚠ Предупреждение: не удалось сохранить конфигурацию в БД: %v", err)
			log.Printf("  Сервер продолжит работу с конфигурацией из переменных окружения")
		} else {
			log.Printf("✓ Конфигурация сохранена в сервисную БД")
		}
	}

	// Создаем сервер
	log.Println("[7/8] Создание сервера и инициализация компонентов...")
	srv := server.NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, config)
	log.Printf("✓ Сервер создан")

	// Канал для отслеживания ошибок запуска
	startErrorChan := make(chan error, 1)
	serverStarted := make(chan bool, 1)

	// Запускаем сервер в горутине
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("✗ КРИТИЧЕСКАЯ ОШИБКА: Паника при запуске сервера: %v", r)
				startErrorChan <- fmt.Errorf("panic: %v", r)
				time.Sleep(2 * time.Second)
				log.Fatalf("Паника при запуске сервера: %v", r)
			}
		}()
		log.Printf("Запуск HTTP сервера на порту %s...", config.Port)
		if err := srv.Start(); err != nil {
			// Детальное логирование ошибки перед fatal
			log.Printf("✗ КРИТИЧЕСКАЯ ОШИБКА: Не удалось запустить HTTP сервер")
			log.Printf("  Порт: %s", config.Port)
			log.Printf("  Ошибка: %v", err)
			log.Printf("  Тип ошибки: %T", err)
			startErrorChan <- err
			// Даем время на вывод логов перед завершением
			time.Sleep(2 * time.Second)
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ждем запуска сервера или ошибки
	log.Println("[8/8] Ожидание запуска сервера...")
	select {
	case err := <-startErrorChan:
		log.Printf("✗ Сервер не запустился: %v", err)
		time.Sleep(2 * time.Second) // Даем время на вывод логов
		os.Exit(1)
	case <-time.After(3 * time.Second):
		// Проверяем, действительно ли сервер запустился
		// Если за 3 секунды нет ошибки, считаем что запустился
		serverStarted <- true
	}

	log.Printf("✓ Сервер запущен на порту %s", config.Port)
	log.Println("API доступно по адресу: http://localhost:9999")
	log.Println("Health check: http://localhost:9999/health")
	log.Println("Для остановки нажмите Ctrl+C")

	// Ожидаем сигнал завершения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Получен сигнал завершения, останавливаем сервер...")

	// Останавливаем сервер с таймаутом
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при остановке сервера: %v", err)
	} else {
		log.Println("Сервер успешно остановлен")
	}
}

// getWorkingDir возвращает рабочую директорию или путь к исполняемому файлу
func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		// Если не удалось получить рабочую директорию, используем путь к exe
		if exePath, err := os.Executable(); err == nil {
			return filepath.Dir(exePath)
		}
		return "unknown"
	}
	return wd
}

