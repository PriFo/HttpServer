package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"httpserver/database"

	"github.com/google/uuid"
)

func main() {
	var serviceDBPath string
	var projectID int
	var clientID int
	var fix bool

	flag.StringVar(&serviceDBPath, "service-db", "data/service.db", "Путь к service.db")
	flag.IntVar(&projectID, "project", 0, "ID проекта (0 = все проекты)")
	flag.IntVar(&clientID, "client", 0, "ID клиента (0 = все клиенты)")
	flag.BoolVar(&fix, "fix", false, "Исправить upload записи (по умолчанию только проверка)")
	flag.Parse()

	fmt.Println("Проверка и исправление upload записей для существующих баз данных")
	fmt.Println("=" + string(make([]byte, 80)) + "=")

	// Открываем service DB
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		log.Fatalf("Не удалось открыть service DB: %v", err)
	}
	defer serviceDB.Close()

	// Получаем проекты для проверки
	var projects []*database.ClientProject
	if projectID > 0 {
		project, err := serviceDB.GetClientProject(projectID)
		if err != nil {
			log.Fatalf("Не удалось получить проект %d: %v", projectID, err)
		}
		if project == nil {
			log.Fatalf("Проект %d не найден", projectID)
		}
		projects = []*database.ClientProject{project}
	} else if clientID > 0 {
		clientProjects, err := serviceDB.GetClientProjects(clientID)
		if err != nil {
			log.Fatalf("Не удалось получить проекты клиента %d: %v", clientID, err)
		}
		projects = clientProjects
	} else {
		// Получаем все проекты
		clients, err := serviceDB.GetAllClients()
		if err != nil {
			log.Fatalf("Не удалось получить клиентов: %v", err)
		}

		for _, client := range clients {
			clientProjects, err := serviceDB.GetClientProjects(client.ID)
			if err != nil {
				log.Printf("Ошибка получения проектов клиента %d: %v", client.ID, err)
				continue
			}
			projects = append(projects, clientProjects...)
		}
	}

	fmt.Printf("\nНайдено проектов для проверки: %d\n\n", len(projects))


	// Функция для исправления upload записей (копия логики из ensureUploadRecordsForDatabase)
	fixUploadRecords := func(dbPath string, clientID, projectID, databaseID int) error {
		// Открываем исходную базу данных
		sourceDB, err := database.NewDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open source database %s: %w", dbPath, err)
		}
		defer sourceDB.Close()

		// Получаем все существующие upload записи
		uploads, err := sourceDB.GetAllUploads()
		if err != nil {
			log.Printf("Note: Could not get uploads from %s (table may not exist): %v", dbPath, err)
			uploads = []*database.Upload{}
		}

		// Проверяем, есть ли upload записи с правильными client_id и project_id
		needsUpdate := false
		needsCreate := false

		if len(uploads) == 0 {
			needsCreate = true
		} else {
			// Проверяем, есть ли хотя бы одна запись с правильными client_id и project_id
			hasCorrectUpload := false
			for _, upload := range uploads {
				if upload.ClientID != nil && *upload.ClientID == clientID &&
					upload.ProjectID != nil && *upload.ProjectID == projectID {
					hasCorrectUpload = true
					break
				}
			}

			if !hasCorrectUpload {
				needsUpdate = true
				// Если все upload записи не имеют правильных client_id/project_id, создаем новую
				allMissingIDs := true
				for _, upload := range uploads {
					if upload.ClientID != nil || upload.ProjectID != nil {
						allMissingIDs = false
						break
					}
				}
				if allMissingIDs {
					needsCreate = true
				}
			}
		}

		// Обновляем существующие upload записи
		if needsUpdate {
			for _, upload := range uploads {
				// Обновляем только если client_id или project_id отсутствуют или неверны
				shouldUpdate := false
				if upload.ClientID == nil || *upload.ClientID != clientID {
					shouldUpdate = true
				}
				if upload.ProjectID == nil || *upload.ProjectID != projectID {
					shouldUpdate = true
				}

				if shouldUpdate {
					err := sourceDB.UpdateUploadClientProject(upload.ID, clientID, projectID)
					if err != nil {
						log.Printf("Warning: Failed to update upload %d in %s: %v", upload.ID, dbPath, err)
					} else {
						log.Printf("Updated upload %d in %s with client_id=%d, project_id=%d", upload.ID, dbPath, clientID, projectID)
					}
				}
			}
		}

		// Создаем новую upload запись, если нужно
		if needsCreate {
			uploadUUID := uuid.New().String()
			dbID := databaseID

			// Пытаемся определить версию 1С и имя конфигурации из метаданных или имени файла
			version1C := "8.3"
			configName := "Unknown"

			// Парсим имя файла для получения информации
			fileName := filepath.Base(dbPath)
			fileInfo := database.ParseDatabaseFileInfo(fileName)
			if fileInfo.ConfigName != "" && fileInfo.ConfigName != "Unknown" {
				configName = fileInfo.ConfigName
			}

			upload, err := sourceDB.CreateUploadWithDatabase(
				uploadUUID,
				version1C,
				configName,
				&dbID,
				"", // computerName
				"", // userName
				"", // configVersion
				1,  // iterationNumber
				"", // iterationLabel
				"", // programmerName
				"", // uploadPurpose
				nil, // parentUploadID
			)
			if err != nil {
				return fmt.Errorf("failed to create upload in %s: %w", dbPath, err)
			}

			// Обновляем client_id и project_id
			err = sourceDB.UpdateUploadClientProject(upload.ID, clientID, projectID)
			if err != nil {
				log.Printf("Warning: Failed to update new upload %d with client_id/project_id: %v", upload.ID, err)
			} else {
				log.Printf("Created and updated upload %d in %s with client_id=%d, project_id=%d", upload.ID, dbPath, clientID, projectID)
			}
		}

		return nil
	}

	totalDatabases := 0
	fixedDatabases := 0
	skippedDatabases := 0
	errorDatabases := 0

	// Проверяем каждый проект
	for _, project := range projects {
		fmt.Printf("Проект: %s (ID: %d, Клиент: %d)\n", project.Name, project.ID, project.ClientID)
		fmt.Println("-" + string(make([]byte, 60)) + "-")

		// Получаем базы данных проекта
		databases, err := serviceDB.GetProjectDatabases(project.ID, false)
		if err != nil {
			fmt.Printf("  ❌ Ошибка получения БД: %v\n\n", err)
			continue
		}

		if len(databases) == 0 {
			fmt.Printf("  ℹ️  Нет баз данных\n\n")
			continue
		}

		fmt.Printf("  Найдено баз данных: %d\n", len(databases))

		// Проверяем каждую базу данных
		for _, db := range databases {
			totalDatabases++
			fmt.Printf("\n  БД: %s (ID: %d)\n", db.Name, db.ID)
			fmt.Printf("    Путь: %s\n", db.FilePath)

			// Проверяем существование файла (пробуем разные варианты путей)
			dbPath := db.FilePath
			if !filepath.IsAbs(dbPath) {
				// Пробуем разные варианты путей
				possiblePaths := []string{
					dbPath,                                    // Как есть
					filepath.Join("data", dbPath),            // data/uploads/...
					filepath.Join(".", dbPath),               // ./uploads/...
					filepath.Join("data", "uploads", filepath.Base(dbPath)), // data/uploads/имя_файла.db
				}
				
				found := false
				for _, path := range possiblePaths {
					if _, err := os.Stat(path); err == nil {
						dbPath = path
						found = true
						break
					}
				}
				
				if !found {
					fmt.Printf("    ❌ Файл не существует: %s\n", db.FilePath)
					skippedDatabases++
					continue
				}
			} else {
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					fmt.Printf("    ❌ Файл не существует: %s\n", dbPath)
					skippedDatabases++
					continue
				}
			}

			// Открываем исходную БД для проверки
			sourceDB, err := database.NewDB(dbPath)
			if err != nil {
				fmt.Printf("    ❌ Не удалось открыть БД: %v\n", err)
				errorDatabases++
				continue
			}

			// Проверяем upload записи
			uploads, err := sourceDB.GetAllUploads()
			if err != nil {
				fmt.Printf("    ⚠️  Не удалось получить upload записи: %v\n", err)
				uploads = []*database.Upload{}
			}

			fmt.Printf("    Upload записей: %d\n", len(uploads))

			// Проверяем, есть ли правильные upload записи
			hasCorrectUpload := false
			for _, upload := range uploads {
				if upload.ClientID != nil && *upload.ClientID == project.ClientID &&
					upload.ProjectID != nil && *upload.ProjectID == project.ID {
					hasCorrectUpload = true
					break
				}
			}

			if hasCorrectUpload {
				fmt.Printf("    ✅ Есть правильные upload записи\n")
				skippedDatabases++
				sourceDB.Close()
				continue
			}

			// Проверяем наличие данных
			var catalogItemsCount int
			var nomenclatureItemsCount int

			if len(uploads) > 0 {
				for _, upload := range uploads {
					items, _, err := sourceDB.GetCatalogItemsByUpload(upload.ID, nil, 0, 0)
					if err == nil {
						catalogItemsCount += len(items)
					}

					var count int
					sourceDB.QueryRow(`SELECT COUNT(*) FROM nomenclature_items WHERE upload_id = ?`, upload.ID).Scan(&count)
					nomenclatureItemsCount += count
				}
			} else {
				// Проверяем напрямую
				sourceDB.QueryRow(`SELECT COUNT(*) FROM catalog_items`).Scan(&catalogItemsCount)
				sourceDB.QueryRow(`SELECT COUNT(*) FROM nomenclature_items`).Scan(&nomenclatureItemsCount)
			}

			fmt.Printf("    Catalog items: %d\n", catalogItemsCount)
			fmt.Printf("    Nomenclature items: %d\n", nomenclatureItemsCount)

			if catalogItemsCount == 0 && nomenclatureItemsCount == 0 {
				fmt.Printf("    ⚠️  Нет данных в БД\n")
				skippedDatabases++
				sourceDB.Close()
				continue
			}

			// Исправляем, если нужно
			if fix {
				fmt.Printf("    🔧 Исправление upload записей...\n")
				err := fixUploadRecords(dbPath, project.ClientID, project.ID, db.ID)
				if err != nil {
					fmt.Printf("    ❌ Ошибка исправления: %v\n", err)
					errorDatabases++
				} else {
					fmt.Printf("    ✅ Upload записи исправлены\n")
					fixedDatabases++
				}
			} else {
				fmt.Printf("    ⚠️  Требуется исправление (используйте -fix)\n")
				skippedDatabases++
			}

			sourceDB.Close()
		}

		fmt.Println()
	}

	// Итоговая статистика
	fmt.Println("=" + string(make([]byte, 80)) + "=")
	fmt.Println("Итоговая статистика:")
	fmt.Printf("  Всего баз данных: %d\n", totalDatabases)
	fmt.Printf("  Исправлено: %d\n", fixedDatabases)
	fmt.Printf("  Пропущено (уже правильно): %d\n", skippedDatabases)
	fmt.Printf("  Ошибок: %d\n", errorDatabases)

	if !fix && (totalDatabases - skippedDatabases - errorDatabases) > 0 {
		fmt.Println("\n💡 Для исправления запустите с флагом -fix")
	}
}

