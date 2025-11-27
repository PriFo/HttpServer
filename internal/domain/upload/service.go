package upload

import (
	"context"

	"httpserver/internal/domain/repositories"
)

// Service интерфейс бизнес-логики для работы с выгрузками
// Определяет операции на уровне предметной области
type Service interface {
	// ProcessHandshake обрабатывает handshake запрос от 1С
	// Создает новую выгрузку и определяет связанную базу данных
	ProcessHandshake(ctx context.Context, req HandshakeRequest) (*HandshakeResult, error)

	// ProcessMetadata обрабатывает метаданные выгрузки
	ProcessMetadata(ctx context.Context, uploadUUID string, metadata MetadataRequest) error

	// ProcessConstant обрабатывает константу выгрузки
	ProcessConstant(ctx context.Context, uploadUUID string, constant ConstantRequest) error

	// ProcessCatalogMeta обрабатывает метаданные каталога
	ProcessCatalogMeta(ctx context.Context, uploadUUID string, catalog CatalogMetaRequest) error

	// ProcessCatalogItem обрабатывает элемент каталога
	ProcessCatalogItem(ctx context.Context, uploadUUID string, item CatalogItemRequest) error

	// ProcessCatalogItems обрабатывает пакет элементов каталога
	ProcessCatalogItems(ctx context.Context, uploadUUID string, items []CatalogItemRequest) error

	// ProcessNomenclatureBatch обрабатывает пакет номенклатуры
	ProcessNomenclatureBatch(ctx context.Context, uploadUUID string, batch NomenclatureBatchRequest) error

	// CompleteUpload завершает выгрузку
	CompleteUpload(ctx context.Context, uploadUUID string) (*Upload, error)

	// GetUpload возвращает выгрузку по UUID
	GetUpload(ctx context.Context, uploadUUID string) (*Upload, error)

	// ListUploads возвращает список выгрузок с фильтрацией
	ListUploads(ctx context.Context, filter repositories.UploadFilter) ([]*Upload, int64, error)
}

// HandshakeRequest запрос handshake
type HandshakeRequest struct {
	Version1C       string `json:"version_1c"`
	ConfigName      string `json:"config_name"`
	ConfigVersion   string `json:"config_version,omitempty"`
	ComputerName    string `json:"computer_name,omitempty"`
	UserName        string `json:"user_name,omitempty"`
	DatabaseID      *int   `json:"database_id,omitempty"`
	ParentUploadID  string `json:"parent_upload_id,omitempty"`
	IterationNumber string `json:"iteration_number,omitempty"`
}

// HandshakeResult результат handshake
type HandshakeResult struct {
	UploadUUID   string `json:"upload_uuid"`
	DatabaseID   *int   `json:"database_id,omitempty"`
	ClientName   string `json:"client_name,omitempty"`
	ProjectName  string `json:"project_name,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`
	IdentifiedBy string `json:"identified_by,omitempty"`
}

// MetadataRequest запрос метаданных
type MetadataRequest struct {
	Metadata map[string]interface{} `json:"metadata"`
}

// ConstantRequest запрос константы
type ConstantRequest struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"type,omitempty"`
}

// CatalogMetaRequest запрос метаданных каталога
type CatalogMetaRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CatalogItemRequest запрос элемента каталога
type CatalogItemRequest struct {
	Code        string                 `json:"code"`
	Name        string                 `json:"name"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	ParentCode  string                 `json:"parent_code,omitempty"`
}

// NomenclatureBatchRequest запрос пакета номенклатуры
type NomenclatureBatchRequest struct {
	Items []CatalogItemRequest `json:"items"`
}

// Upload представляет выгрузку данных из 1С
// Aggregate Root для домена Upload
type Upload struct {
	ID             int
	UUID           string
	DatabaseID     *int
	ParentID       *int
	Version1C      string
	ConfigName     string
	ConfigVersion  string
	ComputerName   string
	UserName       string
	IterationNumber int
	ClientName     string
	ProjectName    string
	DatabaseName   string
	IdentifiedBy   string
	Status         string
	StartedAt      string // time.Time в формате RFC3339
	CompletedAt    *string
	TotalConstants int
	TotalCatalogs  int
	TotalItems     int
	ProcessedCount int
	ErrorCount     int
	ErrorMessage   string
	Metadata       string
	CreatedAt      string // time.Time в формате RFC3339
	UpdatedAt      string // time.Time в формате RFC3339
}

