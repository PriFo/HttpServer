package services

import (
	"context"
	"fmt"

	"httpserver/database"
	"httpserver/internal/domain/upload"
	"httpserver/server/utils"
)

// databaseInfoAdapter адаптер для использования существующих функций utils в domain layer
type databaseInfoAdapter struct {
	db          *database.DB
	serviceDB   *database.ServiceDB
	dbInfoCache interface{}
}

// NewDatabaseInfoAdapter создает новый адаптер для работы с информацией о базах данных
func NewDatabaseInfoAdapter(
	db *database.DB,
	serviceDB *database.ServiceDB,
	dbInfoCache interface{},
) upload.DatabaseInfoService {
	return &databaseInfoAdapter{
		db:          db,
		serviceDB:   serviceDB,
		dbInfoCache: dbInfoCache,
	}
}

// ResolveDatabaseID определяет database_id на основе параметров запроса
func (a *databaseInfoAdapter) ResolveDatabaseID(
	ctx context.Context,
	databaseID *int,
	computerName, userName, configName, version1C, configVersion string,
) (*int, string, interface{}, error) {
	// Преобразуем *int в string для совместимости с utils.ResolveDatabaseID
	var dbIDStr string
	if databaseID != nil {
		dbIDStr = fmt.Sprintf("%d", *databaseID)
	}

	// Вызываем существующую функцию из utils
	resolvedID, identifiedBy, similarUpload, err := utils.ResolveDatabaseID(
		dbIDStr,
		computerName,
		userName,
		configName,
		version1C,
		configVersion,
		a.db,
	)

	return resolvedID, identifiedBy, similarUpload, err
}

// GetDatabaseInfo получает информацию о клиенте, проекте и базе данных
func (a *databaseInfoAdapter) GetDatabaseInfo(
	ctx context.Context,
	databaseID int,
) (clientName, projectName, databaseName string, err error) {
	return utils.GetDatabaseInfo(a.serviceDB, databaseID, a.dbInfoCache)
}

// GetClientProjectIDs получает client_id и project_id для database_id
func (a *databaseInfoAdapter) GetClientProjectIDs(
	ctx context.Context,
	databaseID int,
) (clientID, projectID int, err error) {
	return utils.GetClientProjectIDs(a.serviceDB, databaseID, a.dbInfoCache)
}

