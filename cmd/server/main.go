//go:build !no_gui
// +build !no_gui

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
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"httpserver/database"
	"httpserver/gui"
	"httpserver/internal/config"
	"httpserver/server"
)

func main() {
	log.Println("═══════════════════════════════════════════════════════")
	log.Println("🚀 Запуск 1C HTTP Server...")

	// Создаем папку data/uploads если её нет
	if _, err := server.EnsureUploadsDirectory("."); err != nil {
		log.Printf("Предупреждение: не удалось создать папку data/uploads: %v", err)
	}

	// Загружаем базовую конфигурацию из env (только для путей к БД)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Определяем путь к основной БД
	// Используем 1c_data.db если существует, иначе data.db
	dbPath := cfg.DatabasePath
	if _, err := os.Stat("1c_data.db"); err == nil {
		dbPath = "1c_data.db"
		log.Printf("Используется существующая база данных: %s", dbPath)
	}

	// Создаем конфигурацию для БД
	dbConfig := database.DBConfig{
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}

	// Создаем базу данных
	db, err := database.NewDBWithConfig(dbPath, dbConfig)
	if err != nil {
		log.Fatalf("Ошибка создания базы данных: %v", err)
	}
	defer db.Close()

	// Создаем базу данных для нормализованных данных
	normalizedDBPath := cfg.NormalizedDatabasePath
	normalizedDB, err := database.NewDBWithConfig(normalizedDBPath, dbConfig)
	if err != nil {
		log.Fatalf("Ошибка создания нормализованной базы данных: %v", err)
	}
	defer normalizedDB.Close()
	log.Printf("Используется нормализованная база данных: %s", normalizedDBPath)

	// Создаем сервисную базу данных для системной информации
	serviceDBPath := cfg.ServiceDatabasePath
	serviceDB, err := database.NewServiceDBWithConfig(serviceDBPath, dbConfig)
	if err != nil {
		log.Fatalf("Ошибка создания сервисной базы данных: %v", err)
	}
	defer serviceDB.Close()
	log.Printf("Используется сервисная база данных: %s", serviceDBPath)

	// Перезагружаем конфигурацию из сервисной БД (если есть)
	cfg, err = config.LoadConfig(serviceDB)
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации из БД: %v", err)
	}

	// Если конфигурации нет в БД, сохраняем текущую из env
	configJSON, _ := serviceDB.GetAppConfig()
	if configJSON == "" {
		log.Printf("Config not found in DB, saving current config from environment")
		if err := server.SaveConfig(cfg, serviceDB); err != nil {
			log.Printf("Warning: failed to save config to DB: %v", err)
		} else {
			log.Printf("Config saved to service database")
		}
	}

	// Создаем сервер с обеими БД и сервисной БД
	srv := server.NewServerWithConfig(db, normalizedDB, serviceDB, dbPath, normalizedDBPath, cfg)

	// Проверяем, нужно ли запускать GUI (по умолчанию в контейнере без GUI)
	useGUI := os.Getenv("USE_GUI") == "true"

	var window *gui.Window
	if useGUI {
		// Создаем GUI окно только если явно указано
		window = gui.NewWindow(srv.GetLogChannel())
	}

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

	// Фоновое сохранение метрик производительности в БД
	go func() {
		// Ждем 10 секунд перед первым сохранением (чтобы сервер успел инициализироваться)
		time.Sleep(10 * time.Second)

		// Периодически очищаем старые метрики (раз в день)
		cleanupTicker := time.NewTicker(24 * time.Hour)
		defer cleanupTicker.Stop()

		// Сохраняем метрики каждые 60 секунд
		saveTicker := time.NewTicker(60 * time.Second)
		defer saveTicker.Stop()

		for {
			select {
			case <-saveTicker.C:
				// Собираем текущие метрики (без логирования - это фоновый процесс)
				snapshot := srv.CollectMetricsSnapshot()
				if snapshot != nil {
					// Сохраняем в БД
					if err := db.SaveMetrics(snapshot); err != nil {
						log.Printf("⚠ [Метрики] Ошибка сохранения: %v", err)
					}
				}

			case <-cleanupTicker.C:
				// Очищаем метрики старше 7 дней
				if err := db.CleanOldMetrics(7); err != nil {
					log.Printf("⚠ [Метрики] Ошибка очистки старых данных: %v", err)
				} else {
					log.Printf("✓ [Метрики] Очистка завершена (retention: 7 дней)")
				}
			}
		}
	}()

	// Обновляем статистику каждые 5 секунд (только для GUI)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Обновляем статистику только для GUI, без спама в консоль
				if useGUI && window != nil {
					stats, err := db.GetStats()
					if err != nil {
						log.Printf("⚠ [GUI] Ошибка получения статистики: %v", err)
						continue
					}
					
					serverStats := server.ServerStats{
						IsRunning:    true,
						TotalStats:   stats,
						LastActivity: time.Now(),
					}
					// Обновляем статистику в GUI
					window.UpdateStatsFromMain(serverStats)
				}
			}
		}
	}()

	// Обработка сигналов для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("═══════════════════════════════════════════════════════")
		log.Println("⏹  Получен сигнал завершения, останавливаю сервер...")
		if useGUI && window != nil {
			window.SetStatus("Сервер останавливается...")
		}

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
	log.Printf("✓ Сервер успешно запущен на порту %s", cfg.Port)
	log.Printf("✓ API доступно: http://localhost:%s", cfg.Port)
	log.Printf("✓ База данных: %s", dbPath)
	log.Printf("✓ Нормализованная БД: %s", normalizedDBPath)
	log.Printf("✓ Сервисная БД: %s", serviceDBPath)
	
	if useGUI && window != nil {
		log.Println("✓ GUI интерфейс включен")
		log.Println("═══════════════════════════════════════════════════════")
		// Показываем GUI и блокируем выполнение
		window.ShowAndRun()
	} else {
		log.Println("✓ Режим без GUI (консольный режим)")
		log.Println("  Для остановки нажмите Ctrl+C")
		log.Println("═══════════════════════════════════════════════════════")
		// Блокируем выполнение
		<-ctx.Done()
	}
}
