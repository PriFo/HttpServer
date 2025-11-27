//go:build tool_clear_normalized_data
// +build tool_clear_normalized_data

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"httpserver/database"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     🗑️  ОЧИСТКА РЕЗУЛЬТАТОВ НОРМАЛИЗАЦИИ                ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Путь к базе данных нормализованных данных
	dbPath := "data/normalized_data.db"
	var projectID *int
	var sessionID *int

	// Парсим аргументы командной строки
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--db", "-d":
			if i+1 < len(os.Args) {
				dbPath = os.Args[i+1]
				i++
			}
		case "--project-id", "-p":
			if i+1 < len(os.Args) {
				var id int
				if _, err := fmt.Sscanf(os.Args[i+1], "%d", &id); err == nil {
					projectID = &id
					i++
				}
			}
		case "--session-id", "-s":
			if i+1 < len(os.Args) {
				var id int
				if _, err := fmt.Sscanf(os.Args[i+1], "%d", &id); err == nil {
					sessionID = &id
					i++
				}
			}
		case "--help", "-h":
			fmt.Println("Использование: clear_normalized_data [опции]")
			fmt.Println()
			fmt.Println("Опции:")
			fmt.Println("  --db, -d PATH          Путь к базе данных (по умолчанию: data/normalized_data.db)")
			fmt.Println("  --project-id, -p ID    Удалить данные только для указанного проекта")
			fmt.Println("  --session-id, -s ID    Удалить данные только для указанной сессии")
			fmt.Println("  --help, -h             Показать эту справку")
			fmt.Println()
			fmt.Println("Примеры:")
			fmt.Println("  clear_normalized_data")
			fmt.Println("  clear_normalized_data --db custom/path/normalized_data.db")
			fmt.Println("  clear_normalized_data --project-id 1")
			fmt.Println("  clear_normalized_data --session-id 42")
			os.Exit(0)
		default:
			if i == 1 && !strings.HasPrefix(arg, "-") {
				// Первый аргумент без префикса - это путь к БД (для обратной совместимости)
				dbPath = arg
			}
		}
	}

	fmt.Printf("📁 База данных: %s\n", dbPath)
	if projectID != nil {
		fmt.Printf("📊 Проект ID: %d\n", *projectID)
	}
	if sessionID != nil {
		fmt.Printf("🔄 Сессия ID: %d\n", *sessionID)
	}
	fmt.Println()

	// Проверяем существование файла
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("❌ Файл базы данных не найден: %s", dbPath)
	}

	// Подключаемся к базе данных
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Подключено к базе данных")
	fmt.Println()

	// Получаем статистику до удаления
	var countBefore int64
	var attributesCount int64
	var groupsCount int64

	// Формируем WHERE условие в зависимости от параметров
	var whereClause string
	var queryArgs []interface{}
	if projectID != nil {
		whereClause = "WHERE project_id = ?"
		queryArgs = []interface{}{*projectID}
	} else if sessionID != nil {
		whereClause = "WHERE normalization_session_id = ?"
		queryArgs = []interface{}{*sessionID}
	}

	countQuery := "SELECT COUNT(*) FROM normalized_data"
	if whereClause != "" {
		countQuery += " " + whereClause
	}

	err = db.QueryRow(countQuery, queryArgs...).Scan(&countBefore)
	if err != nil {
		log.Printf("⚠️  Ошибка подсчета записей: %v", err)
		countBefore = 0
	}

	// Подсчет атрибутов (только если удаляем все, иначе сложно)
	if projectID == nil && sessionID == nil {
		err = db.QueryRow("SELECT COUNT(*) FROM normalized_item_attributes").Scan(&attributesCount)
		if err != nil {
			log.Printf("⚠️  Ошибка подсчета атрибутов: %v", err)
			attributesCount = 0
		}
	}

	groupsQuery := `
		SELECT COUNT(DISTINCT normalized_name || '|' || category) 
		FROM normalized_data`
	if whereClause != "" {
		groupsQuery += " " + whereClause
	}

	err = db.QueryRow(groupsQuery, queryArgs...).Scan(&groupsCount)
	if err != nil {
		log.Printf("⚠️  Ошибка подсчета групп: %v", err)
		groupsCount = 0
	}

	// Выводим статистику
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📊 ТЕКУЩАЯ СТАТИСТИКА:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("   • Нормализованных записей: %d\n", countBefore)
	fmt.Printf("   • Групп товаров: %d\n", groupsCount)
	fmt.Printf("   • Атрибутов товаров: %d\n", attributesCount)
	fmt.Println()

	if countBefore == 0 {
		fmt.Println("✅ База данных уже пуста. Нечего удалять.")
		return
	}

	// Предупреждение
	fmt.Println("⚠️  ВНИМАНИЕ!")
	if projectID != nil {
		fmt.Printf("   Это действие удалит все результаты нормализации для проекта ID: %d\n", *projectID)
	} else if sessionID != nil {
		fmt.Printf("   Это действие удалит все результаты нормализации для сессии ID: %d\n", *sessionID)
	} else {
		fmt.Println("   Это действие удалит ВСЕ результаты нормализации:")
	}
	fmt.Println("   • Нормализованные записи")
	fmt.Println("   • Атрибуты товаров (удалятся автоматически)")
	fmt.Println()
	fmt.Println("   Это действие НЕОБРАТИМО!")
	fmt.Println()

	// Запрашиваем подтверждение
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("   Введите 'DELETE' для подтверждения: ")
	confirmation, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("❌ Ошибка чтения подтверждения: %v", err)
	}

	confirmation = strings.TrimSpace(confirmation)
	if confirmation != "DELETE" {
		fmt.Println()
		fmt.Println("❌ Операция отменена. Подтверждение не получено.")
		return
	}

	fmt.Println()
	fmt.Println("🔄 Удаление данных...")

	// Выполняем удаление в зависимости от параметров
	var rowsAffected int64
	if projectID != nil {
		rowsAffected, err = db.DeleteNormalizedDataByProjectID(*projectID)
		if err != nil {
			log.Fatalf("❌ Ошибка удаления данных проекта: %v", err)
		}
	} else if sessionID != nil {
		rowsAffected, err = db.DeleteNormalizedDataBySessionID(*sessionID)
		if err != nil {
			log.Fatalf("❌ Ошибка удаления данных сессии: %v", err)
		}
	} else {
		rowsAffected, err = db.DeleteAllNormalizedData()
		if err != nil {
			log.Fatalf("❌ Ошибка удаления данных: %v", err)
		}
	}

	// Проверяем результат
	var countAfter int64
	checkQuery := "SELECT COUNT(*) FROM normalized_data"
	if whereClause != "" {
		checkQuery += " " + whereClause
	}
	err = db.QueryRow(checkQuery, queryArgs...).Scan(&countAfter)
	if err != nil {
		log.Printf("⚠️  Ошибка проверки результата: %v", err)
		countAfter = 0
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("✅ РЕЗУЛЬТАТ:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("   • Удалено записей: %d\n", rowsAffected)
	fmt.Printf("   • Осталось записей: %d\n", countAfter)
	fmt.Println()

	if countAfter == 0 {
		fmt.Println("✅ База данных успешно очищена!")
	} else {
		fmt.Printf("⚠️  В базе данных осталось %d записей. Возможно, произошла ошибка.\n", countAfter)
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     ✅ ОПЕРАЦИЯ ЗАВЕРШЕНА                                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

