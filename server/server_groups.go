package server

import (
	"context"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/enrichment"
	"httpserver/nomenclature"
	"httpserver/normalization"
	"httpserver/normalization/algorithms"
	"httpserver/quality"
)

// ServerGroups группирует связанные поля Server для лучшей организации
type ServerGroups struct {
	// Базы данных
	Databases *DatabaseGroup
	
	// Нормализация
	Normalization *NormalizationGroup
	
	// Классификация
	Classification *ClassificationGroup
	
	// Качество данных
	Quality *QualityGroup
	
	// AI клиенты
	AIClients *AIClientsGroup
}

// DatabaseGroup группа полей, связанных с базами данных
type DatabaseGroup struct {
	DB                      *database.DB
	NormalizedDB            *database.DB
	ServiceDB               *database.ServiceDB
	CurrentDBPath           string
	CurrentNormalizedDBPath string
	DBMutex                 sync.RWMutex
}

// NormalizationGroup группа полей, связанных с нормализацией
type NormalizationGroup struct {
	Normalizer          *normalization.Normalizer
	NormalizerEvents    chan string
	NormalizerRunning   bool
	NormalizerMutex     sync.RWMutex
	NormalizerStartTime time.Time
	NormalizerProcessed int
	NormalizerSuccess   int
	NormalizerErrors    int
	NormalizerCtx       context.Context
	NormalizerCancel    context.CancelFunc
}

// ClassificationGroup группа полей, связанных с классификацией
type ClassificationGroup struct {
	HierarchicalClassifier *normalization.HierarchicalClassifier
	KpvedClassifierMutex  sync.RWMutex
	KpvedCurrentTasks     map[int]*ClassificationTask
	KpvedCurrentTasksMutex sync.RWMutex
	KpvedWorkersStopped   bool
	KpvedWorkersStopMutex sync.RWMutex
}

// QualityGroup группа полей, связанных с качеством данных
type QualityGroup struct {
	QualityAnalyzer        *quality.QualityAnalyzer
	QualityAnalysisRunning bool
	QualityAnalysisMutex   sync.RWMutex
	QualityAnalysisStatus  QualityAnalysisStatus
}

// AIClientsGroup группа полей, связанных с AI клиентами
type AIClientsGroup struct {
	ArliaiClient     *ArliaiClient
	ArliaiCache      *ArliaiCache
	OpenRouterClient *OpenRouterClient
	HuggingFaceClient *HuggingFaceClient
	EnrichmentFactory *enrichment.EnricherFactory
}

// ProcessorGroup группа полей, связанных с обработкой
type ProcessorGroup struct {
	NomenclatureProcessor *nomenclature.NomenclatureProcessor
	ProcessorMutex        sync.RWMutex
}

// CacheGroup группа полей, связанных с кэшированием
type CacheGroup struct {
	SimilarityCache      *algorithms.OptimizedHybridSimilarity
	SimilarityCacheMutex sync.RWMutex
}

