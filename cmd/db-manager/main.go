package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"httpserver/database"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "list":
		handleList()
	case "delete":
		handleDelete()
	case "backup":
		handleBackup()
	case "cleanup":
		handleCleanup()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Database Manager - CLI utility for managing database files")
	fmt.Println()
	fmt.Println("Usage: db-manager <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list                    List all database files")
	fmt.Println("  delete <path>           Delete a database file")
	fmt.Println("  backup [--output=path]  Create a backup of all databases")
	fmt.Println("  cleanup                 Delete unused databases")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  db-manager list")
	fmt.Println("  db-manager delete data/uploads/test.db")
	fmt.Println("  db-manager backup --output=backup.zip")
	fmt.Println("  db-manager cleanup")
}

func handleList() {
	serviceDBPath := "data/service.db"
	if _, err := os.Stat(serviceDBPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			serviceDBPath = "service.db"
		}
	}

	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		log.Printf("Warning: Could not open service database: %v", err)
		serviceDB = nil
	} else {
		defer serviceDB.Close()
	}

	// Защищенные файлы
	protectedFiles := map[string]bool{
		"service.db":         true,
		"1c_data.db":         true,
		"data.db":            true,
		"normalized_data.db": true,
	}

	scanPaths := []string{
		".",
		"data",
		"data/uploads",
		"/app",
		"/app/data",
		"/app/data/uploads",
	}

	fileMap := make(map[string]bool)
	var allFiles []map[string]interface{}

	for _, scanPath := range scanPaths {
		if _, err := os.Stat(scanPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			log.Printf("Error checking path %s: %v, skipping", scanPath, err)
			continue
		}

		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(path), ".db") {
				return nil
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil
			}

			if fileMap[absPath] {
				return nil
			}
			fileMap[absPath] = true

			fileName := filepath.Base(absPath)
			isProtected := protectedFiles[fileName]

			fileType := "other"
			if isProtected {
				if fileName == "service.db" {
					fileType = "service"
				} else {
					fileType = "main"
				}
			} else if strings.Contains(absPath, "uploads") || strings.Contains(absPath, "data/uploads") {
				fileType = "uploaded"
			} else if strings.Contains(absPath, "data") {
				fileType = "main"
			}

			fileInfo := map[string]interface{}{
				"path":        absPath,
				"name":        fileName,
				"size":        info.Size(),
				"modified_at": info.ModTime().Format(time.RFC3339),
				"type":        fileType,
				"protected":   isProtected,
			}

			// Проверяем, связан ли файл с проектом
			if serviceDB != nil {
				_, projectID, err := serviceDB.FindClientAndProjectByDatabasePath(absPath)
				if err == nil && projectID > 0 {
					fileInfo["linked_to_project"] = true
					fileInfo["project_id"] = projectID

					projectDB, err := serviceDB.GetProjectDatabaseByPath(projectID, absPath)
					if err == nil && projectDB != nil {
						fileInfo["database_id"] = projectDB.ID
					}
				} else {
					fileInfo["linked_to_project"] = false
				}
			}

			allFiles = append(allFiles, fileInfo)
			return nil
		})

		if err != nil {
			log.Printf("Error scanning path %s: %v", scanPath, err)
		}
	}

	// Выводим результаты
	fmt.Printf("Found %d database files:\n\n", len(allFiles))
	for _, file := range allFiles {
		protected := ""
		if file["protected"].(bool) {
			protected = " [PROTECTED]"
		}
		linked := ""
		if linkedToProject, ok := file["linked_to_project"].(bool); ok && linkedToProject {
			linked = " [LINKED]"
		}
		fmt.Printf("%s%s%s\n", file["path"], protected, linked)
		fmt.Printf("  Type: %s, Size: %d bytes, Modified: %s\n",
			file["type"], file["size"], file["modified_at"])
		if projectID, ok := file["project_id"]; ok {
			fmt.Printf("  Project ID: %d\n", projectID)
		}
		fmt.Println()
	}
}

func handleDelete() {
	if len(os.Args) < 3 {
		fmt.Println("Error: path is required")
		fmt.Println("Usage: db-manager delete <path>")
		os.Exit(1)
	}

	path := os.Args[2]

	// Защищенные файлы
	protectedFiles := map[string]bool{
		"service.db":         true,
		"1c_data.db":         true,
		"data.db":            true,
		"normalized_data.db": true,
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Invalid path: %v", err)
	}

	fileName := filepath.Base(absPath)
	if protectedFiles[fileName] {
		log.Fatalf("File %s is protected and cannot be deleted", fileName)
	}

	// Проверяем существование файла
	if _, err := os.Stat(absPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Fatalf("File does not exist: %s", absPath)
		}
		log.Fatalf("Error checking file %s: %v", absPath, err)
	}

	// Удаляем записи из project_databases, если они существуют
	serviceDBPath := "data/service.db"
	if _, err := os.Stat(serviceDBPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			serviceDBPath = "service.db"
		}
	}

	if serviceDB, err := database.NewServiceDB(serviceDBPath); err == nil {
		defer serviceDB.Close()

		_, projectID, err := serviceDB.FindClientAndProjectByDatabasePath(absPath)
		if err == nil && projectID > 0 {
			projectDB, err := serviceDB.GetProjectDatabaseByPath(projectID, absPath)
			if err == nil && projectDB != nil {
				if err := serviceDB.DeleteProjectDatabase(projectDB.ID); err != nil {
					log.Printf("Warning: Failed to delete database record: %v", err)
				} else {
					fmt.Printf("Deleted database record (ID: %d)\n", projectDB.ID)
				}
			}
		}
	}

	// Удаляем физический файл
	if err := os.Remove(absPath); err != nil {
		log.Fatalf("Failed to delete file: %v", err)
	}

	fmt.Printf("Successfully deleted: %s\n", absPath)
}

func handleBackup() {
	outputFlag := flag.NewFlagSet("backup", flag.ExitOnError)
	outputPath := outputFlag.String("output", "", "Output path for backup file")
	outputFlag.Parse(os.Args[2:])

	// Определяем путь к бэкапу
	backupDir := "data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Fatalf("Failed to create backup directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("backup_%s.zip", timestamp)
	if *outputPath != "" {
		backupFileName = *outputPath
		if !strings.HasSuffix(backupFileName, ".zip") {
			backupFileName += ".zip"
		}
	}

	backupPath := filepath.Join(backupDir, backupFileName)

	// Создаем ZIP архив
	zipFile, err := os.Create(backupPath)
	if err != nil {
		log.Fatalf("Failed to create backup file: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Собираем файлы для бэкапа
	scanPaths := []string{
		".",
		"data",
		"data/uploads",
	}

	fileMap := make(map[string]bool)
	addedFiles := 0
	totalSize := int64(0)

	protectedFiles := map[string]bool{
		"service.db": true,
	}

	for _, scanPath := range scanPaths {
		if _, err := os.Stat(scanPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			log.Printf("Error checking path %s: %v, skipping", scanPath, err)
			continue
		}

		err := filepath.Walk(scanPath, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(filePath), ".db") {
				return nil
			}

			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return nil
			}

			if fileMap[absPath] {
				return nil
			}
			fileMap[absPath] = true

			fileName := filepath.Base(absPath)
			if protectedFiles[fileName] {
				// Пропускаем service.db по умолчанию
				return nil
			}

			// Определяем путь в архиве
			var archivePath string
			if strings.Contains(filePath, "uploads") {
				archivePath = filepath.Join("uploads", fileName)
			} else {
				archivePath = filepath.Join("main", fileName)
			}

			// Открываем файл
			sourceFile, err := os.Open(filePath)
			if err != nil {
				log.Printf("Failed to open file %s: %v", filePath, err)
				return nil
			}
			defer sourceFile.Close()

			// Создаем запись в архиве
			archiveFile, err := zipWriter.Create(archivePath)
			if err != nil {
				log.Printf("Failed to create archive entry for %s: %v", filePath, err)
				return nil
			}

			// Копируем содержимое
			if _, err := io.Copy(archiveFile, sourceFile); err != nil {
				log.Printf("Failed to copy file %s to archive: %v", filePath, err)
				return nil
			}

			addedFiles++
			totalSize += info.Size()
			return nil
		})

		if err != nil {
			log.Printf("Error scanning path %s: %v", scanPath, err)
		}
	}

	fmt.Printf("Backup created successfully: %s\n", backupPath)
	fmt.Printf("Files: %d, Total size: %d bytes\n", addedFiles, totalSize)
}

func handleCleanup() {
	serviceDBPath := "data/service.db"
	if _, err := os.Stat(serviceDBPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			serviceDBPath = "service.db"
		}
	}

	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		log.Fatalf("Failed to open service database: %v", err)
	}
	defer serviceDB.Close()

	// Получаем все базы данных из project_databases
	// (здесь нужен метод для получения всех баз данных, но для упрощения
	// мы просто сканируем файлы и проверяем, есть ли они в БД)

	scanPaths := []string{
		"data/uploads",
	}

	fileMap := make(map[string]bool)
	deletedCount := 0

	for _, scanPath := range scanPaths {
		if _, err := os.Stat(scanPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			log.Printf("Error checking path %s: %v, skipping", scanPath, err)
			continue
		}

		err := filepath.Walk(scanPath, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(filePath), ".db") {
				return nil
			}

			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return nil
			}

			if fileMap[absPath] {
				return nil
			}
			fileMap[absPath] = true

			// Проверяем, есть ли файл в project_databases
			_, projectID, err := serviceDB.FindClientAndProjectByDatabasePath(absPath)
			if err != nil || projectID == 0 {
				// Файл не связан с проектом - можно удалить
				fmt.Printf("Deleting unused database: %s\n", absPath)
				if err := os.Remove(absPath); err != nil {
					log.Printf("Failed to delete %s: %v", absPath, err)
				} else {
					deletedCount++
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error scanning path %s: %v", scanPath, err)
		}
	}

	fmt.Printf("\nCleanup completed. Deleted %d unused database files.\n", deletedCount)
}

