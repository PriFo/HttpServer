package utils

import (
	"fmt"
	"strconv"

	"httpserver/database"
)

// ValidateHandshakeRequest валидирует обязательные поля handshake запроса
func ValidateHandshakeRequest(version1C, configName string) error {
	if version1C == "" {
		return fmt.Errorf("version_1c is required")
	}
	if configName == "" {
		return fmt.Errorf("config_name is required")
	}
	return nil
}

// ResolveDatabaseID определяет database_id из запроса
// Возвращает databaseID, identifiedBy и similarUpload
func ResolveDatabaseID(
	dbIDStr string,
	computerName, userName, configName, version1C, configVersion string,
	db *database.DB,
) (*int, string, *database.Upload, error) {
	if dbIDStr != "" {
		// Приоритет 1: Прямой database_id из запроса
		dbID, err := ValidateIntPathParam(dbIDStr, "database_id")
		if err == nil {
			return &dbID, "direct_database_id", nil, nil
		}
		return nil, "", nil, err
	}

	// Приоритет 2: Автоматический поиск по косвенным параметрам
	similarUpload, err := db.FindSimilarUpload(
		computerName,
		userName,
		configName,
		version1C,
		configVersion,
	)

	if err == nil && similarUpload != nil && similarUpload.DatabaseID != nil {
		identifiedBy := fmt.Sprintf("similar_upload_%d", similarUpload.ID)
		return similarUpload.DatabaseID, identifiedBy, similarUpload, nil
	}

	return nil, "none", nil, nil
}

// ValidateIntPathParam валидирует целочисленный параметр из path
func ValidateIntPathParam(paramStr, paramName string) (int, error) {
	if paramStr == "" {
		return 0, fmt.Errorf("%s is required", paramName)
	}

	value, err := strconv.Atoi(paramStr)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got: %s", paramName, paramStr)
	}

	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive, got: %d", paramName, value)
	}

	return value, nil
}

// GetDatabaseInfo получает информацию о базе данных, проекте и клиенте
// Использует кэш для оптимизации производительности
func GetDatabaseInfo(serviceDB *database.ServiceDB, databaseID int, cache interface{}) (clientName, projectName, databaseName string, err error) {
	if serviceDB == nil {
		return "", "", "", fmt.Errorf("serviceDB is nil")
	}

	// Используем кэш, если он доступен
	var dbInfo *database.ProjectDatabase
	var project *database.ClientProject
	var client *database.Client

	if cache != nil {
		// Приведение типа к *DatabaseInfoCache
		if dbCache, ok := cache.(interface {
			GetProjectDatabase(*database.ServiceDB, int) (*database.ProjectDatabase, error)
			GetClientProject(*database.ServiceDB, int) (*database.ClientProject, error)
			GetClient(*database.ServiceDB, int) (*database.Client, error)
		}); ok {
			dbInfo, err = dbCache.GetProjectDatabase(serviceDB, databaseID)
			if err != nil {
				return "", "", "", fmt.Errorf("failed to get project database: %w", err)
			}
			if dbInfo == nil {
				return "", "", "", fmt.Errorf("project database not found")
			}

			project, err = dbCache.GetClientProject(serviceDB, dbInfo.ClientProjectID)
			if err != nil {
				return "", "", dbInfo.Name, fmt.Errorf("failed to get client project: %w", err)
			}
			if project == nil {
				return "", "", dbInfo.Name, fmt.Errorf("client project not found")
			}

			client, err = dbCache.GetClient(serviceDB, project.ClientID)
			if err != nil {
				return "", project.Name, dbInfo.Name, fmt.Errorf("failed to get client: %w", err)
			}
			if client == nil {
				return "", project.Name, dbInfo.Name, fmt.Errorf("client not found")
			}

			return client.Name, project.Name, dbInfo.Name, nil
		}
	}

	// Fallback к прямому обращению к БД, если кэш недоступен
	dbInfo, err = serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get project database: %w", err)
	}
	if dbInfo == nil {
		return "", "", "", fmt.Errorf("project database not found")
	}

	databaseName = dbInfo.Name

	project, err = serviceDB.GetClientProject(dbInfo.ClientProjectID)
	if err != nil {
		return "", "", databaseName, fmt.Errorf("failed to get client project: %w", err)
	}
	if project == nil {
		return "", "", databaseName, fmt.Errorf("client project not found")
	}

	projectName = project.Name

	client, err = serviceDB.GetClient(project.ClientID)
	if err != nil {
		return "", projectName, databaseName, fmt.Errorf("failed to get client: %w", err)
	}
	if client == nil {
		return "", projectName, databaseName, fmt.Errorf("client not found")
	}

	clientName = client.Name
	return clientName, projectName, databaseName, nil
}

// GetClientProjectIDs получает client_id и project_id для database_id
// Использует кэш для оптимизации производительности
func GetClientProjectIDs(serviceDB *database.ServiceDB, databaseID int, cache interface{}) (clientID, projectID int, err error) {
	if serviceDB == nil {
		return 0, 0, fmt.Errorf("serviceDB is nil")
	}

	// Используем кэш, если он доступен
	if cache != nil {
		if dbCache, ok := cache.(interface {
			GetProjectDatabase(*database.ServiceDB, int) (*database.ProjectDatabase, error)
			GetClientProject(*database.ServiceDB, int) (*database.ClientProject, error)
		}); ok {
			dbInfo, err := dbCache.GetProjectDatabase(serviceDB, databaseID)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to get project database: %w", err)
			}
			if dbInfo == nil {
				return 0, 0, fmt.Errorf("project database not found")
			}

			project, err := dbCache.GetClientProject(serviceDB, dbInfo.ClientProjectID)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to get client project: %w", err)
			}
			if project == nil {
				return 0, 0, fmt.Errorf("client project not found")
			}

			return project.ClientID, project.ID, nil
		}
	}

	// Fallback к прямому обращению к БД
	dbInfo, err := serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get project database: %w", err)
	}
	if dbInfo == nil {
		return 0, 0, fmt.Errorf("project database not found")
	}

	project, err := serviceDB.GetClientProject(dbInfo.ClientProjectID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get client project: %w", err)
	}
	if project == nil {
		return 0, 0, fmt.Errorf("client project not found")
	}

	return project.ClientID, project.ID, nil
}

// NormalizeIterationNumber нормализует номер итерации
func NormalizeIterationNumber(iterationNumber int) int {
	if iterationNumber <= 0 {
		return 1
	}
	return iterationNumber
}

