package repositories

import (
	"context"
	"time"
)

// UploadRepository интерфейс для работы с выгрузками данных
type UploadRepository interface {
	// Основные операции CRUD
	Create(ctx context.Context, upload *Upload) error
	GetByID(ctx context.Context, id string) (*Upload, error)
	GetByUUID(ctx context.Context, uuid string) (*Upload, error)
	Update(ctx context.Context, upload *Upload) error
	Delete(ctx context.Context, id string) error

	// Поиск и фильтрация
	List(ctx context.Context, filter UploadFilter) ([]Upload, int64, error)
	GetByDatabaseID(ctx context.Context, databaseID string) ([]Upload, error)
	GetByStatus(ctx context.Context, status string) ([]Upload, error)
	GetByDateRange(ctx context.Context, start, end time.Time) ([]Upload, error)

	// Статистика
	GetStatistics(ctx context.Context, databaseID string) (*UploadStatistics, error)
	GetRecentUploads(ctx context.Context, limit int) ([]Upload, error)

	// Массовые операции
	BatchCreate(ctx context.Context, uploads []Upload) error
	BatchUpdateStatus(ctx context.Context, ids []string, status string) error
}

// NormalizationRepository интерфейс для работы с процессами нормализации
type NormalizationRepository interface {
	// Основные операции
	Create(ctx context.Context, normalization *NormalizationProcess) error
	GetByID(ctx context.Context, id string) (*NormalizationProcess, error)
	GetByUploadID(ctx context.Context, uploadID string) (*NormalizationProcess, error)
	Update(ctx context.Context, normalization *NormalizationProcess) error
	Delete(ctx context.Context, id string) error

	// Управление процессом
	StartProcess(ctx context.Context, uploadID string) (*NormalizationProcess, error)
	CompleteProcess(ctx context.Context, processID string, result *NormalizationResult) error
	GetActiveProcesses(ctx context.Context) ([]NormalizationProcess, error)

	// Прогресс и логирование
	UpdateProgress(ctx context.Context, processID string, progress float64, processed int, total int) error
	AddLog(ctx context.Context, processID string, logEntry NormalizationLog) error
	GetLogs(ctx context.Context, processID string) ([]NormalizationLog, error)

	// Статистика
	GetStatistics(ctx context.Context) (*NormalizationStatistics, error)
	GetProcessHistory(ctx context.Context, uploadID string) ([]NormalizationProcess, error)
}

// QualityRepository интерфейс для работы с качеством данных
type QualityRepository interface {
	// Основные операции
	Create(ctx context.Context, quality *QualityReport) error
	GetByID(ctx context.Context, id string) (*QualityReport, error)
	GetByUploadID(ctx context.Context, uploadID string) (*QualityReport, error)
	Update(ctx context.Context, quality *QualityReport) error
	Delete(ctx context.Context, id string) error

	// Анализ качества
	AnalyzeUpload(ctx context.Context, uploadID string) (*QualityReport, error)
	GetQualityTrends(ctx context.Context, databaseID string, period time.Duration) ([]QualityTrend, error)
	GetQualityIssues(ctx context.Context, filter QualityIssueFilter) ([]QualityIssue, int64, error)

	// Метрики
	GetMetrics(ctx context.Context, entityID string) (*EntityMetrics, error)
	UpdateMetrics(ctx context.Context, entityID string, metrics *EntityMetrics) error
	GetOverallQualityScore(ctx context.Context, databaseID string) (float64, error)
}

// ClassificationRepository интерфейс для работы с классификацией
type ClassificationRepository interface {
	// Основные операции
	Create(ctx context.Context, classification *Classification) error
	GetByID(ctx context.Context, id string) (*Classification, error)
	GetByEntityID(ctx context.Context, entityID string) (*Classification, error)
	Update(ctx context.Context, classification *Classification) error
	Delete(ctx context.Context, id string) error

	// Классификация
	ClassifyEntity(ctx context.Context, entityID string, category string) (*Classification, error)
	GetClassificationHistory(ctx context.Context, entityID string) ([]Classification, error)
	GetEntitiesByCategory(ctx context.Context, category string) ([]string, error)

	// Статистика
	GetClassificationAccuracy(ctx context.Context, category string) (float64, error)
	GetCategoryDistribution(ctx context.Context, databaseID string) (map[string]int, error)
	GetClassificationStats(ctx context.Context) (*ClassificationStatistics, error)
}

// CounterpartyRepository интерфейс для работы с контрагентами
type CounterpartyRepository interface {
	// Основные операции
	Create(ctx context.Context, counterparty *Counterparty) error
	GetByID(ctx context.Context, id string) (*Counterparty, error)
	Update(ctx context.Context, counterparty *Counterparty) error
	Delete(ctx context.Context, id string) error

	// Поиск и фильтрация
	Search(ctx context.Context, query string, filters CounterpartyFilter) ([]Counterparty, int64, error)
	GetByTaxID(ctx context.Context, taxID string) (*Counterparty, error)
	GetByDatabaseID(ctx context.Context, databaseID string) ([]Counterparty, error)

	// Нормализация и дубликаты
	FindDuplicates(ctx context.Context, counterparty *Counterparty) ([]Counterparty, error)
	MergeDuplicates(ctx context.Context, ids []string) (*Counterparty, error)
	Normalize(ctx context.Context, counterparty *Counterparty) (*Counterparty, error)

	// Статистика
	GetStatistics(ctx context.Context, databaseID string) (*CounterpartyStatistics, error)
	GetRecentActivity(ctx context.Context, limit int) ([]CounterpartyActivity, error)
}

// ClientRepository интерфейс для работы с клиентами
type ClientRepository interface {
	// Основные операции
	Create(ctx context.Context, client *Client) error
	GetByID(ctx context.Context, id string) (*Client, error)
	Update(ctx context.Context, client *Client) error
	Delete(ctx context.Context, id string) error

	// Поиск и фильтрация
	List(ctx context.Context, filter ClientFilter) ([]Client, int64, error)
	Search(ctx context.Context, query string) ([]Client, error)
	GetByContactEmail(ctx context.Context, email string) (*Client, error)
	GetByTaxID(ctx context.Context, taxID string) (*Client, error)

	// Проекты и эталоны
	GetProjects(ctx context.Context, clientID string) ([]ClientProject, error)
	CreateProject(ctx context.Context, project *ClientProject) error
	GetBenchmarks(ctx context.Context, clientID string) ([]ClientBenchmark, error)
	CreateBenchmark(ctx context.Context, benchmark *ClientBenchmark) error
}

// ProjectRepository интерфейс для работы с проектами
type ProjectRepository interface {
	// Основные операции
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id string) error

	// Поиск и фильтрация
	List(ctx context.Context, filter ProjectFilter) ([]Project, int64, error)
	GetByClientID(ctx context.Context, clientID string) ([]Project, error)
	Search(ctx context.Context, query string) ([]Project, error)
}

// DatabaseRepository интерфейс для работы с базами данных
type DatabaseRepository interface {
	// Основные операции
	Create(ctx context.Context, db *Database) error
	GetByID(ctx context.Context, id string) (*Database, error)
	Update(ctx context.Context, db *Database) error
	Delete(ctx context.Context, id string) error

	// Поиск и фильтрация
	List(ctx context.Context, filter DatabaseFilter) ([]Database, int64, error)
	GetByProjectID(ctx context.Context, projectID string) ([]Database, error)
	GetByPath(ctx context.Context, path string) (*Database, error)
	GetByConnectionString(ctx context.Context, connectionString string) (*Database, error)

	// Подключение и статус
	TestConnection(ctx context.Context, db *Database) error
	GetConnectionStatus(ctx context.Context, id string) (string, error)

	// Информация о базе
	GetSchema(ctx context.Context, id string) (*DatabaseSchema, error)
	GetTables(ctx context.Context, id string) ([]string, error)
	GetColumns(ctx context.Context, tableID string) ([]Column, error)

	// Бэкап и восстановление
	CreateBackup(ctx context.Context, dbID string) (*Backup, error)
	RestoreBackup(ctx context.Context, backupID string) error
	GetBackups(ctx context.Context, dbID string) ([]Backup, error)
}

// SnapshotRepository интерфейс для работы с срезами данных
type SnapshotRepository interface {
	// Основные операции
	Create(ctx context.Context, snapshot *Snapshot) error
	GetByID(ctx context.Context, id string) (*Snapshot, error)
	Update(ctx context.Context, snapshot *Snapshot) error
	Delete(ctx context.Context, id string) error

	// Управление срезами
	CreateFromUploads(ctx context.Context, name string, description string, uploadIDs []string) (*Snapshot, error)
	GetByProjectID(ctx context.Context, projectID int) ([]Snapshot, error)
	GetByClientID(ctx context.Context, clientID int) ([]Snapshot, error)

	// Сравнение и анализ
	CompareSnapshots(ctx context.Context, snapshotID1, snapshotID2 int) (*SnapshotComparison, error)
	GetSnapshotEvolution(ctx context.Context, snapshotID int) (*SnapshotEvolution, error)

	// Метрики
	GetSnapshotMetrics(ctx context.Context, snapshotID int) (*SnapshotMetrics, error)
}
