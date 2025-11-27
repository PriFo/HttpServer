package types

import (
	"httpserver/internal/domain/models"
)

// Алиасы для обратной совместимости
// Все типы теперь в internal/domain/models
type (
	HandshakeRequest              = models.HandshakeRequest
	HandshakeResponse             = models.HandshakeResponse
	MetadataRequest               = models.MetadataRequest
	MetadataResponse              = models.MetadataResponse
	ConstantValue                 = models.ConstantValue
	ConstantRequest               = models.ConstantRequest
	ConstantResponse              = models.ConstantResponse
	CatalogMetaRequest            = models.CatalogMetaRequest
	CatalogMetaResponse           = models.CatalogMetaResponse
	CatalogItemRequest            = models.CatalogItemRequest
	CatalogItemResponse           = models.CatalogItemResponse
	CatalogItem                   = models.CatalogItem
	CatalogItemsRequest           = models.CatalogItemsRequest
	CatalogItemsResponse          = models.CatalogItemsResponse
	NomenclatureItem              = models.NomenclatureItem
	NomenclatureBatchRequest      = models.NomenclatureBatchRequest
	NomenclatureBatchResponse     = models.NomenclatureBatchResponse
	CompleteRequest               = models.CompleteRequest
	CompleteResponse              = models.CompleteResponse
	LogEntry                      = models.LogEntry
	ServerStats                   = models.ServerStats
	CurrentUploadInfo             = models.CurrentUploadInfo
	Client                        = models.Client
	ClientProject                 = models.ClientProject
	ClientDetailResponse          = models.ClientDetailResponse
	ClientStatistics              = models.ClientStatistics
	ClientDocument                = models.ClientDocument
	NormalizationStatus           = models.NormalizationStatus
	QualityReport                 = models.QualityReport
	QualitySummary                = models.QualitySummary
	QualityDashboard              = models.QualityDashboard
	EntityMetrics                 = models.EntityMetrics
	ActivityLog                   = models.ActivityLog
	DashboardOverviewResponse     = models.DashboardOverviewResponse
)


