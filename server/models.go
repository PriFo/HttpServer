package server

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
	ErrorResponse                 = models.ErrorResponse
	LogEntry                      = models.LogEntry
	ServerStats                   = models.ServerStats
	CurrentUploadInfo             = models.CurrentUploadInfo
	UploadListItem                = models.UploadListItem
	CatalogInfo                   = models.CatalogInfo
	UploadDetails                 = models.UploadDetails
	DataItem                      = models.DataItem
	DataResponse                  = models.DataResponse
	VerifyRequest                 = models.VerifyRequest
	VerifyResponse                = models.VerifyResponse
	NomenclatureProcessingResponse = models.NomenclatureProcessingResponse
	DBStatsResponse               = models.DBStatsResponse
	ProcessingStatsResponse       = models.ProcessingStatsResponse
	NomenclatureStatusResponse    = models.NomenclatureStatusResponse
	RecentRecord                  = models.RecentRecord
	RecentRecordsResponse         = models.RecentRecordsResponse
	PendingRecord                 = models.PendingRecord
	PendingRecordsResponse        = models.PendingRecordsResponse
	Client                        = models.Client
	ClientProject                 = models.ClientProject
	ClientBenchmark               = models.ClientBenchmark
	ClientListResponse            = models.ClientListResponse
	ClientListItem                = models.ClientListItem
	ClientDetailResponse          = models.ClientDetailResponse
	ClientStatistics              = models.ClientStatistics
	ClientDocument                = models.ClientDocument
	ClientProjectResponse         = models.ClientProjectResponse
	ProjectStatistics             = models.ProjectStatistics
	ClientBenchmarkResponse       = models.ClientBenchmarkResponse
	QualityReport                 = models.QualityReport
	QualitySummary                = models.QualitySummary
	QualityDashboard              = models.QualityDashboard
	EntityMetrics                 = models.EntityMetrics
	Recommendation                = models.Recommendation
	SnapshotRequest               = models.SnapshotRequest
	SnapshotUploadRequest         = models.SnapshotUploadRequest
	SnapshotResponse              = models.SnapshotResponse
	AutoSnapshotRequest           = models.AutoSnapshotRequest
	SnapshotNormalizationRequest  = models.SnapshotNormalizationRequest
	SnapshotNormalizationResult   = models.SnapshotNormalizationResult
	UploadNormalizationResult     = models.UploadNormalizationResult
	NormalizationChanges          = models.NormalizationChanges
	SnapshotListResponse          = models.SnapshotListResponse
	SnapshotComparisonResponse    = models.SnapshotComparisonResponse
	IterationComparison           = models.IterationComparison
	ItemChange                    = models.ItemChange
	SnapshotMetricsResponse       = models.SnapshotMetricsResponse
	QualityImprovement            = models.QualityImprovement
	SnapshotEvolutionResponse     = models.SnapshotEvolutionResponse
	NomenclatureEvolution         = models.NomenclatureEvolution
	NomenclatureHistoryItem       = models.NomenclatureHistoryItem
	NormalizationStatus           = models.NormalizationStatus
	ActivityLog                   = models.ActivityLog
	DashboardOverviewResponse     = models.DashboardOverviewResponse
)
