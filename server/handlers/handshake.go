package handlers

import (
	"database/sql"
	"errors"
	"fmt"

	"httpserver/database"
	apperrors "httpserver/server/errors"
)

// HandshakeDatabaseInfo информация о базе данных для handshake
type HandshakeDatabaseInfo struct {
	DatabaseID   *int
	ClientName   string
	ProjectName  string
	DatabaseName string
	IdentifiedBy string
}

// ValidateHandshakeRequest валидирует запрос handshake
func ValidateHandshakeRequest(version1C, configName string) error {
	if version1C == "" {
		return apperrors.NewValidationError("version_1c обязателен", nil)
	}
	if configName == "" {
		return apperrors.NewValidationError("config_name обязателен", nil)
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

// GetDatabaseInfo получает информацию о базе данных, проекте и клиенте
// Использует кэш для оптимизации производительности
func GetDatabaseInfo(serviceDB *database.ServiceDB, databaseID int, cache interface{}) (clientName, projectName, databaseName string, err error) {
	if serviceDB == nil {
		return "", "", "", apperrors.NewInternalError("serviceDB не инициализирован", nil)
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
				if errors.Is(err, sql.ErrNoRows) {
					return "", "", "", apperrors.NewNotFoundError("база данных проекта не найдена", err)
				}
				return "", "", "", apperrors.NewInternalError("не удалось получить базу данных проекта", err)
			}
			if dbInfo == nil {
				return "", "", "", apperrors.NewNotFoundError("база данных проекта не найдена", nil)
			}

			project, err = dbCache.GetClientProject(serviceDB, dbInfo.ClientProjectID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return "", "", dbInfo.Name, apperrors.NewNotFoundError("проект клиента не найден", err)
				}
				return "", "", dbInfo.Name, apperrors.NewInternalError("не удалось получить проект клиента", err)
			}
			if project == nil {
				return "", "", dbInfo.Name, apperrors.NewNotFoundError("проект клиента не найден", nil)
			}

			client, err = dbCache.GetClient(serviceDB, project.ClientID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return "", project.Name, dbInfo.Name, apperrors.NewNotFoundError("клиент не найден", err)
				}
				return "", project.Name, dbInfo.Name, apperrors.NewInternalError("не удалось получить клиента", err)
			}
			if client == nil {
				return "", project.Name, dbInfo.Name, apperrors.NewNotFoundError("клиент не найден", nil)
			}

			return client.Name, project.Name, dbInfo.Name, nil
		}
	}

	// Fallback к прямому обращению к БД, если кэш недоступен
	dbInfo, err = serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", "", apperrors.NewNotFoundError("база данных проекта не найдена", err)
		}
		return "", "", "", apperrors.NewInternalError("не удалось получить базу данных проекта", err)
	}
	if dbInfo == nil {
		return "", "", "", apperrors.NewNotFoundError("база данных проекта не найдена", nil)
	}

	databaseName = dbInfo.Name

	project, err = serviceDB.GetClientProject(dbInfo.ClientProjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", databaseName, apperrors.NewNotFoundError("проект клиента не найден", err)
		}
		return "", "", databaseName, apperrors.NewInternalError("не удалось получить проект клиента", err)
	}
	if project == nil {
		return "", "", databaseName, apperrors.NewNotFoundError("проект клиента не найден", nil)
	}

	projectName = project.Name

	client, err = serviceDB.GetClient(project.ClientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", projectName, databaseName, apperrors.NewNotFoundError("клиент не найден", err)
		}
		return "", projectName, databaseName, apperrors.NewInternalError("не удалось получить клиента", err)
	}
	if client == nil {
		return "", projectName, databaseName, apperrors.NewNotFoundError("клиент не найден", nil)
	}

	clientName = client.Name
	return clientName, projectName, databaseName, nil
}

// GetClientProjectIDs получает client_id и project_id для database_id
// Использует кэш для оптимизации производительности
func GetClientProjectIDs(serviceDB *database.ServiceDB, databaseID int, cache interface{}) (clientID, projectID int, err error) {
	if serviceDB == nil {
		return 0, 0, apperrors.NewInternalError("serviceDB не инициализирован", nil)
	}

	// Используем кэш, если он доступен
	if cache != nil {
		if dbCache, ok := cache.(interface {
			GetProjectDatabase(*database.ServiceDB, int) (*database.ProjectDatabase, error)
			GetClientProject(*database.ServiceDB, int) (*database.ClientProject, error)
		}); ok {
			dbInfo, err := dbCache.GetProjectDatabase(serviceDB, databaseID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return 0, 0, apperrors.NewNotFoundError("база данных проекта не найдена", err)
				}
				return 0, 0, apperrors.NewInternalError("не удалось получить базу данных проекта", err)
			}
			if dbInfo == nil {
				return 0, 0, apperrors.NewNotFoundError("база данных проекта не найдена", nil)
			}

			project, err := dbCache.GetClientProject(serviceDB, dbInfo.ClientProjectID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return 0, 0, apperrors.NewNotFoundError("проект клиента не найден", err)
				}
				return 0, 0, apperrors.NewInternalError("не удалось получить проект клиента", err)
			}
			if project == nil {
				return 0, 0, apperrors.NewNotFoundError("проект клиента не найден", nil)
			}

			return project.ClientID, project.ID, nil
		}
	}

	// Fallback к прямому обращению к БД
	dbInfo, err := serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, apperrors.NewNotFoundError("база данных проекта не найдена", err)
		}
		return 0, 0, apperrors.NewInternalError("не удалось получить базу данных проекта", err)
	}
	if dbInfo == nil {
		return 0, 0, apperrors.NewNotFoundError("база данных проекта не найдена", nil)
	}

	project, err := serviceDB.GetClientProject(dbInfo.ClientProjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, apperrors.NewNotFoundError("проект клиента не найден", err)
		}
		return 0, 0, apperrors.NewInternalError("не удалось получить проект клиента", err)
	}
	if project == nil {
		return 0, 0, apperrors.NewNotFoundError("проект клиента не найден", nil)
	}

	return project.ClientID, project.ID, nil
}

// NormalizeIterationNumber нормализует номер итерации (минимум 1)
func NormalizeIterationNumber(iterationNumber int) int {
	if iterationNumber <= 0 {
		return 1
	}
	return iterationNumber
}

