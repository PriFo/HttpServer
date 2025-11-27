package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"httpserver/database"
	apperrors "httpserver/server/errors"
	"httpserver/server/types"
	"httpserver/server/utils"
)

// UploadService сервис для работы с выгрузками данных из 1С
type UploadService struct {
	db              *database.DB
	serviceDB       *database.ServiceDB
	dbInfoCache     interface{} // *server.DatabaseInfoCache
	logFunc         func(entry interface{}) // server.LogEntry, но без прямого импорта для избежания циклических зависимостей
}

// NewUploadService создает новый сервис для работы с выгрузками
func NewUploadService(
	db *database.DB,
	serviceDB *database.ServiceDB,
	dbInfoCache interface{},
	logFunc func(entry interface{}), // server.LogEntry, но без прямого импорта
) *UploadService {
	return &UploadService{
		db:          db,
		serviceDB:   serviceDB,
		dbInfoCache: dbInfoCache,
		logFunc:     logFunc,
	}
}

// HandshakeResult результат выполнения handshake
type HandshakeResult struct {
	UploadUUID   string
	DatabaseID   *int
	ClientName   string
	ProjectName  string
	DatabaseName string
	IdentifiedBy string
	Upload       *database.Upload
}

// ProcessHandshake обрабатывает handshake запрос
func (s *UploadService) ProcessHandshake(req types.HandshakeRequest) (*HandshakeResult, error) {
	// Валидация обязательных полей
	if err := utils.ValidateHandshakeRequest(req.Version1C, req.ConfigName); err != nil {
		return nil, apperrors.NewValidationError("ошибка валидации запроса", err)
	}

	// Создаем новую выгрузку
	uploadUUID := uuid.New().String()

	// Определяем database_id
	databaseID, identifiedBy, similarUpload, err := utils.ResolveDatabaseID(
		req.DatabaseID,
		req.ComputerName,
		req.UserName,
		req.ConfigName,
		req.Version1C,
		req.ConfigVersion,
		s.db,
	)
	if err != nil {
		s.logFunc(types.LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("Error resolving database ID: %v", err),
			UploadUUID: uploadUUID,
			Endpoint:   "/handshake",
		})
	}

	// Получаем информацию о базе данных, проекте и клиенте
	var clientName, projectName, databaseName string
	if databaseID != nil {
		clientName, projectName, databaseName, err = utils.GetDatabaseInfo(s.serviceDB, *databaseID, s.dbInfoCache)
		if err != nil {
			s.logFunc(types.LogEntry{
				Timestamp: time.Now(),
				Level:     "WARN",
				Message:   fmt.Sprintf("Failed to get database info: %v", err),
				UploadUUID: uploadUUID,
				Endpoint:   "/handshake",
			})
		}
	}

	// Определяем parent_upload_id если указан ParentUploadID
	var parentUploadID *int
	if req.ParentUploadID != "" {
		parentUpload, err := s.db.GetUploadByUUID(req.ParentUploadID)
		if err == nil {
			parentUploadID = &parentUpload.ID
		}
	}

	// Нормализуем номер итерации
	iterationNumber := utils.NormalizeIterationNumber(req.IterationNumber)

	// Создаем выгрузку
	upload, err := s.db.CreateUploadWithDatabase(
		uploadUUID, req.Version1C, req.ConfigName, databaseID,
		req.ComputerName, req.UserName, req.ConfigVersion,
		iterationNumber, req.IterationLabel, req.ProgrammerName, req.UploadPurpose, parentUploadID,
	)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось создать выгрузку", err)
	}

	// Обновляем кэшированные значения client_id и project_id
	if databaseID != nil {
		var clientID, projectID int

		// Если идентификация была по похожей выгрузке, используем её значения
		if identifiedBy != "" && similarUpload != nil {
			if similarUpload.ClientID != nil {
				clientID = *similarUpload.ClientID
			}
			if similarUpload.ProjectID != nil {
				projectID = *similarUpload.ProjectID
			}
		}

		// Если не получили из похожей выгрузки, получаем из serviceDB
		if clientID == 0 || projectID == 0 {
			clientID, projectID, err = utils.GetClientProjectIDs(s.serviceDB, *databaseID, s.dbInfoCache)
			if err != nil {
				s.logFunc(types.LogEntry{
					Timestamp:  time.Now(),
					Level:      "WARN",
					Message:    fmt.Sprintf("Failed to get client/project IDs: %v", err),
					UploadUUID: uploadUUID,
					Endpoint:   "/handshake",
				})
			}
		}

		// Обновляем upload с кэшированными значениями
		if clientID > 0 && projectID > 0 {
			err = s.db.UpdateUploadClientProject(upload.ID, clientID, projectID)
			if err != nil {
				s.logFunc(types.LogEntry{
					Timestamp:  time.Now(),
					Level:      "WARNING",
					Message:    fmt.Sprintf("Failed to update cached client_id and project_id: %v", err),
					UploadUUID: uploadUUID,
					Endpoint:   "/handshake",
				})
			}
		}
	}

	return &HandshakeResult{
		UploadUUID:   uploadUUID,
		DatabaseID:   databaseID,
		ClientName:   clientName,
		ProjectName:  projectName,
		DatabaseName: databaseName,
		IdentifiedBy: identifiedBy,
		Upload:       upload,
	}, nil
}

// ProcessMetadata обрабатывает метаинформацию
func (s *UploadService) ProcessMetadata(uploadUUID string) error {
	// Проверяем существование выгрузки
	_, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return apperrors.NewInternalError("не удалось получить выгрузку", err)
	}
	return nil
}

// ProcessConstant обрабатывает константу
func (s *UploadService) ProcessConstant(uploadUUID, name, synonym, constType, valueContent string) error {
	// Получаем выгрузку
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	// Добавляем константу
	if err := s.db.AddConstant(upload.ID, name, synonym, constType, valueContent); err != nil {
		return apperrors.NewInternalError("не удалось добавить константу", err)
	}

	return nil
}

// ProcessCatalogMeta обрабатывает метаданные справочника
func (s *UploadService) ProcessCatalogMeta(uploadUUID, name, synonym string) (*database.Catalog, error) {
	// Получаем выгрузку
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	// Добавляем справочник
	catalog, err := s.db.AddCatalog(upload.ID, name, synonym)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось добавить каталог", err)
	}

	return catalog, nil
}

// ProcessCatalogItem обрабатывает элемент справочника
func (s *UploadService) ProcessCatalogItem(uploadUUID, catalogName, reference, code, name, attributes, tableParts string) error {
	// Получаем выгрузку
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	// Находим справочник по имени
	var catalogID int
	err = s.db.QueryRow("SELECT id FROM catalogs WHERE upload_id = ? AND name = ?", upload.ID, catalogName).Scan(&catalogID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.NewNotFoundError("каталог не найден", err)
		}
		return apperrors.NewInternalError("не удалось получить каталог", err)
	}

	// Добавляем элемент справочника
	if err := s.db.AddCatalogItem(catalogID, reference, code, name, attributes, tableParts); err != nil {
		return apperrors.NewInternalError("не удалось добавить элемент каталога", err)
	}

	return nil
}

// ProcessCatalogItemsBatch обрабатывает пакетную загрузку элементов справочника
func (s *UploadService) ProcessCatalogItemsBatch(uploadUUID, catalogName string, items []types.CatalogItem) (processedCount, failedCount int, err error) {
	// Получаем выгрузку
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return 0, 0, apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	// Находим справочник по имени
	var catalogID int
	err = s.db.QueryRow("SELECT id FROM catalogs WHERE upload_id = ? AND name = ?", upload.ID, catalogName).Scan(&catalogID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, apperrors.NewNotFoundError("каталог не найден", err)
		}
		return 0, 0, apperrors.NewInternalError("не удалось получить каталог", err)
	}

	// Обрабатываем каждый элемент пакета
	processedCount = 0
	failedCount = 0

	for _, item := range items {
		if err := s.db.AddCatalogItem(catalogID, item.Reference, item.Code, item.Name, item.Attributes, item.TableParts); err != nil {
			failedCount++
			s.logFunc(types.LogEntry{
				Timestamp:  time.Now(),
				Level:      "ERROR",
				Message:    fmt.Sprintf("Failed to add catalog item '%s': %v", item.Name, err),
				UploadUUID: uploadUUID,
				Endpoint:   "/catalog/items",
			})
		} else {
			processedCount++
		}
	}

	return processedCount, failedCount, nil
}

// ProcessNomenclatureBatch обрабатывает пакетную загрузку номенклатуры
func (s *UploadService) ProcessNomenclatureBatch(uploadUUID string, items []types.NomenclatureItem) (int, error) {
	// Получаем выгрузку
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return 0, apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	// Преобразуем элементы в формат для базы данных
	nomenclatureItems := make([]database.NomenclatureItem, 0, len(items))
	for _, item := range items {
		nomenclatureItems = append(nomenclatureItems, database.NomenclatureItem{
			NomenclatureReference:   item.NomenclatureReference,
			NomenclatureCode:        item.NomenclatureCode,
			NomenclatureName:        item.NomenclatureName,
			CharacteristicReference: item.CharacteristicReference,
			CharacteristicName:      item.CharacteristicName,
			AttributesXML:           item.Attributes,
			TablePartsXML:           item.TableParts,
		})
	}

	// Добавляем пакет элементов номенклатуры
	if err := s.db.AddNomenclatureItemsBatch(upload.ID, nomenclatureItems); err != nil {
		return 0, apperrors.NewInternalError("не удалось добавить элементы номенклатуры", err)
	}

	return len(items), nil
}

// ProcessComplete обрабатывает завершение выгрузки
func (s *UploadService) ProcessComplete(uploadUUID string) error {
	_, err := s.ProcessCompleteWithUpload(uploadUUID)
	return err
}

// ProcessCompleteWithUpload обрабатывает завершение выгрузки и возвращает upload
func (s *UploadService) ProcessCompleteWithUpload(uploadUUID string) (*database.Upload, error) {
	// Получаем выгрузку
	upload, err := s.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	// Завершаем выгрузку
	if err := s.db.CompleteUpload(upload.ID); err != nil {
		return nil, apperrors.NewInternalError("не удалось завершить выгрузку", err)
	}

	return upload, nil
}

// GetAllUploads получает список всех выгрузок
func (s *UploadService) GetAllUploads() ([]*database.Upload, error) {
	return s.db.GetAllUploads()
}

// GetUploadByUUID получает выгрузку по UUID
func (s *UploadService) GetUploadByUUID(uuid string) (*database.Upload, error) {
	return s.db.GetUploadByUUID(uuid)
}

// GetUploadDetails получает детальную информацию о выгрузке
func (s *UploadService) GetUploadDetails(uuid string) (*database.Upload, []*database.Catalog, []*database.Constant, error) {
	return s.db.GetUploadDetails(uuid)
}

// GetClientProjectIDs получает clientID и projectID для databaseID
func (s *UploadService) GetClientProjectIDs(databaseID int) (clientID, projectID int, err error) {
	return utils.GetClientProjectIDs(s.serviceDB, databaseID, s.dbInfoCache)
}
