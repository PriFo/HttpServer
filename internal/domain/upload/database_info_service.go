package upload

import (
	"context"
)

// DatabaseInfoService интерфейс для получения информации о базе данных
// Абстракция для работы с serviceDB и кэшем информации о базах данных
type DatabaseInfoService interface {
	// ResolveDatabaseID определяет database_id на основе параметров запроса
	ResolveDatabaseID(
		ctx context.Context,
		databaseID *int,
		computerName, userName, configName, version1C, configVersion string,
	) (*int, string, interface{}, error) // databaseID, identifiedBy, similarUpload, error

	// GetDatabaseInfo получает информацию о клиенте, проекте и базе данных
	GetDatabaseInfo(ctx context.Context, databaseID int) (clientName, projectName, databaseName string, err error)

	// GetClientProjectIDs получает client_id и project_id для database_id
	GetClientProjectIDs(ctx context.Context, databaseID int) (clientID, projectID int, err error)
}

