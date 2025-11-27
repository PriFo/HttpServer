package upload

import (
	"context"
	"fmt"
	"time"

	"httpserver/internal/domain/repositories"

	"github.com/google/uuid"
)

// service реализация domain service для upload
type service struct {
	uploadRepo           repositories.UploadRepository
	databaseInfoService  DatabaseInfoService
}

// NewService создает новый domain service для upload
func NewService(uploadRepo repositories.UploadRepository, databaseInfoService DatabaseInfoService) Service {
	return &service{
		uploadRepo:          uploadRepo,
		databaseInfoService: databaseInfoService,
	}
}

// ProcessHandshake обрабатывает handshake запрос от 1С
func (s *service) ProcessHandshake(ctx context.Context, req HandshakeRequest) (*HandshakeResult, error) {
	// Валидация запроса
	if req.Version1C == "" {
		return nil, ErrInvalidVersion
	}
	if req.ConfigName == "" {
		return nil, ErrInvalidConfigName
	}

	// Генерируем UUID для новой выгрузки
	uploadUUID := uuid.New().String()

	// Создаем новую выгрузку
	upload := &repositories.Upload{
		UUID:          uploadUUID,
		Version1C:     req.Version1C,
		ConfigName:    req.ConfigName,
		ConfigVersion: req.ConfigVersion,
		ComputerName:  req.ComputerName,
		UserName:      req.UserName,
		Status:        "started",
		StartedAt:     time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Определяем database_id и получаем информацию о базе данных
	var databaseID *int = req.DatabaseID
	var identifiedBy string = "database_id"
	var clientName, projectName, databaseName string

	if s.databaseInfoService != nil {
		// Пытаемся определить database_id через DatabaseInfoService
		resolvedID, idBy, _, err := s.databaseInfoService.ResolveDatabaseID(
			ctx,
			req.DatabaseID,
			req.ComputerName,
			req.UserName,
			req.ConfigName,
			req.Version1C,
			req.ConfigVersion,
		)
		if err == nil && resolvedID != nil {
			databaseID = resolvedID
			identifiedBy = idBy
			upload.DatabaseID = fmt.Sprintf("%d", *databaseID)

			// Получаем информацию о клиенте, проекте и базе данных
			clientName, projectName, databaseName, _ = s.databaseInfoService.GetDatabaseInfo(ctx, *databaseID)
		} else if req.DatabaseID != nil {
			// Если resolve не сработал, но database_id был указан напрямую
			upload.DatabaseID = fmt.Sprintf("%d", *req.DatabaseID)
			clientName, projectName, databaseName, _ = s.databaseInfoService.GetDatabaseInfo(ctx, *req.DatabaseID)
		}
	} else if req.DatabaseID != nil {
		// Если DatabaseInfoService не доступен, просто используем указанный database_id
		upload.DatabaseID = fmt.Sprintf("%d", *req.DatabaseID)
	}

	result := &HandshakeResult{
		UploadUUID:   uploadUUID,
		DatabaseID:   databaseID,
		ClientName:   clientName,
		ProjectName:  projectName,
		DatabaseName: databaseName,
		IdentifiedBy: identifiedBy,
	}

	// Сохраняем выгрузку
	if err := s.uploadRepo.Create(ctx, upload); err != nil {
		return nil, fmt.Errorf("failed to create upload: %w", err)
	}

	return result, nil
}

// ProcessMetadata обрабатывает метаданные выгрузки
func (s *service) ProcessMetadata(ctx context.Context, uploadUUID string, metadata MetadataRequest) error {
	// Получаем выгрузку
	upload, err := s.uploadRepo.GetByUUID(ctx, uploadUUID)
	if err != nil {
		return fmt.Errorf("failed to get upload: %w", err)
	}

	// Обновляем метаданные (если поддерживается полем Metadata)
	// TODO: Преобразовать metadata в JSON строку
	// upload.Metadata = serializeMetadata(metadata.Metadata)

	// Обновляем выгрузку
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return fmt.Errorf("failed to update upload metadata: %w", err)
	}

	return nil
}

// ProcessConstant обрабатывает константу выгрузки
func (s *service) ProcessConstant(ctx context.Context, uploadUUID string, constant ConstantRequest) error {
	// Получаем выгрузку
	upload, err := s.uploadRepo.GetByUUID(ctx, uploadUUID)
	if err != nil {
		return fmt.Errorf("failed to get upload: %w", err)
	}

	// Увеличиваем счетчик констант
	upload.TotalConstants++

	// TODO: Сохранить константу в отдельную таблицу через репозиторий констант
	// Пока просто обновляем счетчик
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return fmt.Errorf("failed to update upload: %w", err)
	}

	return nil
}

// ProcessCatalogMeta обрабатывает метаданные каталога
func (s *service) ProcessCatalogMeta(ctx context.Context, uploadUUID string, catalog CatalogMetaRequest) error {
	// Получаем выгрузку
	upload, err := s.uploadRepo.GetByUUID(ctx, uploadUUID)
	if err != nil {
		return fmt.Errorf("failed to get upload: %w", err)
	}

	// Увеличиваем счетчик каталогов
	upload.TotalCatalogs++

	// TODO: Сохранить метаданные каталога в отдельную таблицу
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return fmt.Errorf("failed to update upload: %w", err)
	}

	return nil
}

// ProcessCatalogItem обрабатывает элемент каталога
func (s *service) ProcessCatalogItem(ctx context.Context, uploadUUID string, item CatalogItemRequest) error {
	// Получаем выгрузку
	upload, err := s.uploadRepo.GetByUUID(ctx, uploadUUID)
	if err != nil {
		return fmt.Errorf("failed to get upload: %w", err)
	}

	// Увеличиваем счетчик элементов
	upload.TotalItems++

	// TODO: Сохранить элемент каталога в отдельную таблицу
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return fmt.Errorf("failed to update upload: %w", err)
	}

	return nil
}

// ProcessCatalogItems обрабатывает пакет элементов каталога
func (s *service) ProcessCatalogItems(ctx context.Context, uploadUUID string, items []CatalogItemRequest) error {
	// Получаем выгрузку
	upload, err := s.uploadRepo.GetByUUID(ctx, uploadUUID)
	if err != nil {
		return fmt.Errorf("failed to get upload: %w", err)
	}

	// Увеличиваем счетчик элементов
	upload.TotalItems += len(items)

	// TODO: Сохранить элементы каталога пакетом
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return fmt.Errorf("failed to update upload: %w", err)
	}

	return nil
}

// ProcessNomenclatureBatch обрабатывает пакет номенклатуры
func (s *service) ProcessNomenclatureBatch(ctx context.Context, uploadUUID string, batch NomenclatureBatchRequest) error {
	// Обрабатываем как пакет элементов каталога
	return s.ProcessCatalogItems(ctx, uploadUUID, batch.Items)
}

// CompleteUpload завершает выгрузку
func (s *service) CompleteUpload(ctx context.Context, uploadUUID string) (*Upload, error) {
	// Получаем выгрузку
	upload, err := s.uploadRepo.GetByUUID(ctx, uploadUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload: %w", err)
	}

	// Проверяем статус
	if upload.Status == "completed" {
		return nil, ErrUploadCompleted
	}

	// Обновляем статус
	now := time.Now()
	upload.Status = "completed"
	upload.CompletedAt = &now
	upload.UpdatedAt = now

	// Обновляем в репозитории
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return nil, fmt.Errorf("failed to complete upload: %w", err)
	}

	// Преобразуем в domain Upload
	result := s.toDomainUpload(upload)

	return result, nil
}

// GetUpload возвращает выгрузку по UUID
func (s *service) GetUpload(ctx context.Context, uploadUUID string) (*Upload, error) {
	upload, err := s.uploadRepo.GetByUUID(ctx, uploadUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload: %w", err)
	}

	return s.toDomainUpload(upload), nil
}

// ListUploads возвращает список выгрузок с фильтрацией
func (s *service) ListUploads(ctx context.Context, filter repositories.UploadFilter) ([]*Upload, int64, error) {
	uploads, total, err := s.uploadRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list uploads: %w", err)
	}

	result := make([]*Upload, 0, len(uploads))
	for _, u := range uploads {
		result = append(result, s.toDomainUpload(&u))
	}

	return result, total, nil
}

// toDomainUpload преобразует repositories.Upload в domain Upload
func (s *service) toDomainUpload(u *repositories.Upload) *Upload {
	uploadID, _ := parseInt(u.ID)
	upload := &Upload{
		ID:             uploadID,
		UUID:           u.UUID,
		Version1C:      u.Version1C,
		ConfigName:     u.ConfigName,
		ConfigVersion:  u.ConfigVersion,
		ComputerName:   u.ComputerName,
		UserName:       u.UserName,
		Status:         u.Status,
		StartedAt:      u.StartedAt.Format(time.RFC3339),
		CompletedAt:    nil,
		TotalConstants: u.TotalConstants,
		TotalCatalogs:  u.TotalCatalogs,
		TotalItems:     u.TotalItems,
		ProcessedCount: u.ProcessedCount,
		ErrorCount:     u.ErrorCount,
		ErrorMessage:   u.ErrorMessage,
		Metadata:       u.Metadata,
		CreatedAt:      u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      u.UpdatedAt.Format(time.RFC3339),
	}

	if u.DatabaseID != "" {
		dbID, _ := parseInt(u.DatabaseID)
		upload.DatabaseID = &dbID
	}

	if u.CompletedAt != nil {
		completedStr := u.CompletedAt.Format(time.RFC3339)
		upload.CompletedAt = &completedStr
	}

	return upload
}

// parseInt вспомогательная функция для парсинга строки в int
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
