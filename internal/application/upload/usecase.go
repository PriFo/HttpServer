package upload

import (
	"context"
	"fmt"

	"httpserver/internal/domain/repositories"
	"httpserver/internal/domain/upload"
)

// UseCase представляет use case для работы с выгрузками
// Координирует выполнение бизнес-логики между domain и infrastructure слоями
type UseCase struct {
	uploadRepo    repositories.UploadRepository
	uploadService upload.Service
}

// NewUseCase создает новый use case для выгрузок
func NewUseCase(
	uploadRepo repositories.UploadRepository,
	uploadService upload.Service,
) *UseCase {
	return &UseCase{
		uploadRepo:    uploadRepo,
		uploadService: uploadService,
	}
}

// ProcessHandshake обрабатывает handshake запрос
func (uc *UseCase) ProcessHandshake(ctx context.Context, req upload.HandshakeRequest) (*upload.HandshakeResult, error) {
	// Делегируем обработку domain service
	result, err := uc.uploadService.ProcessHandshake(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process handshake: %w", err)
	}
	return result, nil
}

// ProcessMetadata обрабатывает метаданные
func (uc *UseCase) ProcessMetadata(ctx context.Context, uploadUUID string, metadata upload.MetadataRequest) error {
	if err := uc.uploadService.ProcessMetadata(ctx, uploadUUID, metadata); err != nil {
		return fmt.Errorf("failed to process metadata: %w", err)
	}
	return nil
}

// ProcessConstant обрабатывает константу
func (uc *UseCase) ProcessConstant(ctx context.Context, uploadUUID string, constant upload.ConstantRequest) error {
	if err := uc.uploadService.ProcessConstant(ctx, uploadUUID, constant); err != nil {
		return fmt.Errorf("failed to process constant: %w", err)
	}
	return nil
}

// ProcessCatalogMeta обрабатывает метаданные каталога
func (uc *UseCase) ProcessCatalogMeta(ctx context.Context, uploadUUID string, catalog upload.CatalogMetaRequest) error {
	if err := uc.uploadService.ProcessCatalogMeta(ctx, uploadUUID, catalog); err != nil {
		return fmt.Errorf("failed to process catalog metadata: %w", err)
	}
	return nil
}

// ProcessCatalogItem обрабатывает элемент каталога
func (uc *UseCase) ProcessCatalogItem(ctx context.Context, uploadUUID string, item upload.CatalogItemRequest) error {
	if err := uc.uploadService.ProcessCatalogItem(ctx, uploadUUID, item); err != nil {
		return fmt.Errorf("failed to process catalog item: %w", err)
	}
	return nil
}

// ProcessCatalogItems обрабатывает пакет элементов каталога
func (uc *UseCase) ProcessCatalogItems(ctx context.Context, uploadUUID string, items []upload.CatalogItemRequest) error {
	if err := uc.uploadService.ProcessCatalogItems(ctx, uploadUUID, items); err != nil {
		return fmt.Errorf("failed to process catalog items: %w", err)
	}
	return nil
}

// ProcessNomenclatureBatch обрабатывает пакет номенклатуры
func (uc *UseCase) ProcessNomenclatureBatch(ctx context.Context, uploadUUID string, batch upload.NomenclatureBatchRequest) error {
	if err := uc.uploadService.ProcessNomenclatureBatch(ctx, uploadUUID, batch); err != nil {
		return fmt.Errorf("failed to process nomenclature batch: %w", err)
	}
	return nil
}

// CompleteUpload завершает выгрузку
func (uc *UseCase) CompleteUpload(ctx context.Context, uploadUUID string) (*upload.Upload, error) {
	result, err := uc.uploadService.CompleteUpload(ctx, uploadUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to complete upload: %w", err)
	}
	return result, nil
}

// GetUpload возвращает выгрузку по UUID
func (uc *UseCase) GetUpload(ctx context.Context, uploadUUID string) (*upload.Upload, error) {
	result, err := uc.uploadService.GetUpload(ctx, uploadUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload: %w", err)
	}
	return result, nil
}

// ListUploads возвращает список выгрузок с фильтрацией
func (uc *UseCase) ListUploads(ctx context.Context, filter repositories.UploadFilter) ([]*upload.Upload, int64, error) {
	uploads, total, err := uc.uploadService.ListUploads(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list uploads: %w", err)
	}
	return uploads, total, nil
}

