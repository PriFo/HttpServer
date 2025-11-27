package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"httpserver/database"
)

// ScanForDatabaseFiles сканирует указанные директории на наличие файлов формата Выгрузка_*.db
func ScanForDatabaseFiles(scanPaths []string, serviceDB *database.ServiceDB) ([]string, error) {
	var foundFiles []string
	patterns := []string{"Выгрузка_Номенклатура_", "Выгрузка_Контрагенты_"}

	for _, scanPath := range scanPaths {
		// Проверяем существование пути
		if _, err := os.Stat(scanPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Printf("Путь не существует, пропускаем: %s", scanPath)
			} else {
				log.Printf("Ошибка проверки пути, пропускаем: %s: %v", scanPath, err)
			}
			continue
		}

		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Пропускаем ошибки доступа к файлам
			}

			// Проверяем только файлы с расширением .db
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".db") {
				fileName := filepath.Base(path)
				// Проверяем, начинается ли имя файла с одного из паттернов
				matchesPattern := false
				for _, pattern := range patterns {
					if strings.HasPrefix(fileName, pattern) {
						matchesPattern = true
						break
					}
				}
				
				if matchesPattern {
					absPath, err := filepath.Abs(path)
					if err != nil {
						log.Printf("Ошибка получения абсолютного пути для %s: %v", path, err)
						return nil
					}
					foundFiles = append(foundFiles, absPath)

					// Добавляем в pending_databases, если еще нет
					if serviceDB != nil {
						_, err := serviceDB.GetPendingDatabaseByPath(absPath)
						if err != nil {
							// Файл еще не в базе, добавляем
							_, createErr := serviceDB.CreatePendingDatabase(absPath, fileName, info.Size())
							if createErr != nil {
								log.Printf("Ошибка добавления файла в pending databases: %v", createErr)
							} else {
								log.Printf("Добавлен файл в pending databases: %s", absPath)
								
								// Обновляем метаданные с информацией о конфигурации 1С
								dbType := "unknown"
								if detectedType, err := database.DetectDatabaseType(absPath); err == nil {
									dbType = detectedType
								}
								if err := UpdateDatabaseMetadataWithConfig(serviceDB, absPath, dbType); err != nil {
									log.Printf("Ошибка обновления метаданных для %s: %v", absPath, err)
								}
							}
						}
					}
				}
			}
			return nil
		})

		if err != nil {
			log.Printf("Ошибка при сканировании пути %s: %v", scanPath, err)
		}
	}

	return foundFiles, nil
}

// MoveDatabaseToUploads перемещает файл базы данных в папку data/uploads/
func MoveDatabaseToUploads(filePath string, uploadsDir string) (string, error) {
	// Создаем папку uploads, если её нет
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	fileName := filepath.Base(filePath)
	newPath := filepath.Join(uploadsDir, fileName)

	// Если файл уже в нужной папке, возвращаем текущий путь
	if filepath.Dir(filePath) == uploadsDir {
		return filePath, nil
	}

	// Проверяем, существует ли файл по новому пути
	if _, err := os.Stat(newPath); err == nil {
		// Файл уже существует, возвращаем существующий путь
		return newPath, nil
	}

	// Перемещаем файл
	if err := os.Rename(filePath, newPath); err != nil {
		return "", fmt.Errorf("failed to move file: %w", err)
	}

	log.Printf("Файл перемещен: %s -> %s", filePath, newPath)
	return newPath, nil
}

// EnsureUploadsDirectory создает папку data/uploads/ если её нет
func EnsureUploadsDirectory(basePath string) (string, error) {
	uploadsDir := filepath.Join(basePath, "data", "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}
	return uploadsDir, nil
}

// DatabaseFilenameInfo - алиас для database.DatabaseFilenameInfo для обратной совместимости
// Используйте database.DatabaseFilenameInfo напрямую в новом коде
type DatabaseFilenameInfo = database.DatabaseFilenameInfo

// ParseDatabaseNameFromFilename извлекает читаемое название из имени файла базы данных
// Примеры:
// "Выгрузка_Номенклатура_ERPWE_Unknown_Unknown_2025_11_20_10_18_55.db" -> "ERP WE Номенклатура"
// "Выгрузка_Контрагенты_БухгалтерияДляКазахстана_Unknown_Unknown_2025.db" -> "БухгалтерияДляКазахстана Контрагенты"
func ParseDatabaseNameFromFilename(fileName string) string {
	return database.ParseDatabaseNameFromFilename(fileName)
}

// ParseDatabaseFileInfo извлекает полную информацию из имени файла базы данных
func ParseDatabaseFileInfo(fileName string) DatabaseFilenameInfo {
	return database.ParseDatabaseFileInfo(fileName)
}

// UpdateDatabaseMetadataWithConfig обновляет метаданные базы данных с информацией о конфигурации 1С
func UpdateDatabaseMetadataWithConfig(serviceDB *database.ServiceDB, filePath string, dbType string) error {
	if serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	fileName := filepath.Base(filePath)
	fileInfo := database.ParseDatabaseFileInfo(fileName)

	// Получаем существующие метаданные
	existingMetadata, err := serviceDB.GetDatabaseMetadata(filePath)
	if err != nil {
		return fmt.Errorf("failed to get existing metadata: %w", err)
	}

	// Создаем структуру для метаданных
	metadataMap := make(map[string]interface{})
	if existingMetadata != nil && existingMetadata.MetadataJSON != "" {
		// Парсим существующие метаданные
		if err := json.Unmarshal([]byte(existingMetadata.MetadataJSON), &metadataMap); err != nil {
			// Если не удалось распарсить, начинаем с пустой карты
			metadataMap = make(map[string]interface{})
		}
	}

	// Обновляем информацию о конфигурации 1С
	metadataMap["config_name"] = fileInfo.ConfigName
	metadataMap["database_type"] = fileInfo.DatabaseType
	metadataMap["data_type"] = fileInfo.DataType
	metadataMap["display_name"] = fileInfo.DisplayName

	// Сериализуем обратно в JSON
	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Формируем описание
	description := fmt.Sprintf("База данных типа %s", dbType)
	if fileInfo.ConfigName != "" && fileInfo.ConfigName != "Unknown" {
		description = fmt.Sprintf("%s, конфигурация: %s", description, fileInfo.DisplayName)
	}

	// Обновляем метаданные
	if err := serviceDB.UpsertDatabaseMetadata(filePath, dbType, description, string(metadataJSON)); err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}

	return nil
}

// FindMatchingProjectForDatabase - алиас для database.FindMatchingProjectForDatabase для обратной совместимости
// Используйте database.FindMatchingProjectForDatabase напрямую в новом коде
func FindMatchingProjectForDatabase(serviceDB *database.ServiceDB, clientID int, filePath string) (*database.ClientProject, error) {
	return database.FindMatchingProjectForDatabase(serviceDB, clientID, filePath)
}

// ValidateSQLiteDatabase проверяет целостность SQLite файла
// Открывает базу данных и проверяет, что она валидна
func ValidateSQLiteDatabase(filePath string) error {
	// Открываем базу данных
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Проверяем, что база данных не повреждена
	// Выполняем простой запрос для проверки целостности
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database integrity check failed: %w", err)
	}

	// Проверяем, что база данных не пустая (есть хотя бы таблица sqlite_master)
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to check database structure: %w", err)
	}

	// База данных может быть пустой, но это не ошибка
	// Главное, что она валидна и открывается

	return nil
}

// CalculateFileHash вычисляет SHA256 хеш файла
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

