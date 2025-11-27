package repositories

import (
	"time"
)

// ============================================================================
// Upload Domain Models
// ============================================================================

// Upload представляет выгрузку данных из 1С
type Upload struct {
	ID             string     `json:"id"`
	UUID           string     `json:"uuid"`
	DatabaseID     string     `json:"database_id"`
	Version1C      string     `json:"version_1c"`
	ConfigName     string     `json:"config_name"`
	ConfigVersion  string     `json:"config_version,omitempty"`
	ComputerName   string     `json:"computer_name,omitempty"`
	UserName       string     `json:"user_name,omitempty"`
	Status         string     `json:"status"`
	StartedAt      time.Time  `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	TotalConstants int        `json:"total_constants"`
	TotalCatalogs  int        `json:"total_catalogs"`
	TotalItems     int        `json:"total_items"`
	ProcessedCount int        `json:"processed_count"`
	ErrorCount     int        `json:"error_count"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	Metadata       string     `json:"metadata,omitempty"` // JSON
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// UploadFilter фильтр для поиска выгрузок
type UploadFilter struct {
	Status         []string
	DatabaseID     string
	DateFrom       *time.Time
	DateTo         *time.Time
	Version1C      string
	ConfigName     string
	Limit          int
	Offset         int
	OrderBy        string
	OrderDirection string
}

// UploadStatistics статистика выгрузок
type UploadStatistics struct {
	TotalUploads      int64         `json:"total_uploads"`
	SuccessfulUploads int64         `json:"successful_uploads"`
	FailedUploads     int64         `json:"failed_uploads"`
	AverageDuration   time.Duration `json:"average_duration"`
	TotalItems        int64         `json:"total_items"`
	AverageItems      float64       `json:"average_items"`
}

// ============================================================================
// Normalization Domain Models
// ============================================================================

// NormalizationProcess представляет процесс нормализации
type NormalizationProcess struct {
	ID          string     `json:"id"`
	UploadID    string     `json:"upload_id"`
	Status      string     `json:"status"`
	Progress    float64    `json:"progress"`
	Processed   int        `json:"processed"`
	Total       int        `json:"total"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Config      string     `json:"config,omitempty"` // JSON
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NormalizationResult результат нормализации
type NormalizationResult struct {
	TotalProcessed  int                   `json:"total_processed"`
	TotalGroups     int                   `json:"total_groups"`
	Changes         *NormalizationChanges `json:"changes"`
	MasterReference map[string]string     `json:"master_reference"`
	QualityScore    float64               `json:"quality_score"`
	ProcessingTime  time.Duration         `json:"processing_time"`
	Errors          []NormalizationError  `json:"errors"`
}

// NormalizationChanges изменения после нормализации
type NormalizationChanges struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

// NormalizationError ошибка нормализации
type NormalizationError struct {
	RecordID string `json:"record_id"`
	Field    string `json:"field"`
	Error    string `json:"error"`
	Severity string `json:"severity"`
}

// NormalizationLog запись лога нормализации
type NormalizationLog struct {
	ID        int       `json:"id"`
	ProcessID string    `json:"process_id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"` // JSON
}

// NormalizationStatistics статистика нормализации
type NormalizationStatistics struct {
	TotalProcesses      int64         `json:"total_processes"`
	SuccessfulProcesses int64         `json:"successful_processes"`
	FailedProcesses     int64         `json:"failed_processes"`
	AverageDuration     time.Duration `json:"average_duration"`
	TotalProcessed      int64         `json:"total_processed"`
	AverageProgress     float64       `json:"average_progress"`
}

// ============================================================================
// Quality Domain Models
// ============================================================================

// QualityReport отчет о качестве данных
type QualityReport struct {
	ID           string          `json:"id"`
	UploadID     string          `json:"upload_id"`
	DatabaseID   int             `json:"database_id"`
	AnalyzedAt   *time.Time      `json:"analyzed_at,omitempty"`
	OverallScore float64         `json:"overall_score"`
	Metrics      []QualityMetric `json:"metrics"`
	Issues       []QualityIssue  `json:"issues"`
	Summary      QualitySummary  `json:"summary"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// QualityMetric метрика качества
type QualityMetric struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Target      float64 `json:"target"`
	Status      string  `json:"status"` // "good", "warning", "critical"
	Description string  `json:"description,omitempty"`
}

// QualityIssue проблема качества данных
type QualityIssue struct {
	ID          string     `json:"id"`
	EntityID    string     `json:"entity_id"`
	EntityType  string     `json:"entity_type"`
	Field       string     `json:"field"`
	Severity    string     `json:"severity"` // "low", "medium", "high", "critical"`
	Description string     `json:"description"`
	Suggestion  string     `json:"suggestion,omitempty"`
	Rule        string     `json:"rule"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// QualitySummary сводка по качеству
type QualitySummary struct {
	TotalIssues       int                `json:"total_issues"`
	CriticalIssues    int                `json:"critical_issues"`
	HighIssues        int                `json:"high_issues"`
	MediumIssues      int                `json:"medium_issues"`
	LowIssues         int                `json:"low_issues"`
	MetricsByCategory map[string]float64 `json:"metrics_by_category"`
}

// QualityIssueFilter фильтр для поиска проблем качества
type QualityIssueFilter struct {
	Severity   []string
	EntityType string
	EntityID   string
	Field      string
	Resolved   *bool
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int
	Offset     int
}

// QualityTrend тренд качества
type QualityTrend struct {
	Date        time.Time       `json:"date"`
	Score       float64         `json:"score"`
	Metrics     []QualityMetric `json:"metrics"`
	IssuesCount int             `json:"issues_count"`
}

// EntityMetrics метрики по сущности
type EntityMetrics struct {
	Completeness float64 `json:"completeness"`
	Consistency  float64 `json:"consistency"`
	Uniqueness   float64 `json:"uniqueness"`
	Validity     float64 `json:"validity"`
	OverallScore float64 `json:"overall_score"`
}

// ============================================================================
// Classification Domain Models
// ============================================================================

// Classification представляет результат классификации
type Classification struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	EntityType  string    `json:"entity_type"`
	Category    string    `json:"category"`
	Subcategory string    `json:"subcategory"`
	Confidence  float64   `json:"confidence"`
	Rule        string    `json:"rule"`
	Source      string    `json:"source"`
	ProcessedAt time.Time `json:"processed_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ClassificationStatistics статистика классификации
type ClassificationStatistics struct {
	TotalClassifications      int64              `json:"total_classifications"`
	SuccessfulClassifications int64              `json:"successful_classifications"`
	FailedClassifications     int64              `json:"failed_classifications"`
	AverageConfidence         float64            `json:"average_confidence"`
	AccuracyByCategory        map[string]float64 `json:"accuracy_by_category"`
}

// ============================================================================
// Counterparty Domain Models
// ============================================================================

// Counterparty представляет контрагента
type Counterparty struct {
	ID            string    `json:"id"`
	DatabaseID    string    `json:"database_id"`
	Name          string    `json:"name"`
	LegalName     string    `json:"legal_name"`
	TaxID         string    `json:"tax_id"`
	Address       string    `json:"address"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	Status        string    `json:"status"`
	IsNormalized  bool      `json:"is_normalized"`
	Normalization string    `json:"normalization,omitempty"` // JSON
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CounterpartyFilter фильтр для поиска контрагентов
type CounterpartyFilter struct {
	Name         string
	TaxID        string
	Status       []string
	IsNormalized *bool
	DatabaseID   string
	Limit        int
	Offset       int
}

// CounterpartyStatistics статистика контрагентов
type CounterpartyStatistics struct {
	TotalCounterparties      int64          `json:"total_counterparties"`
	NormalizedCounterparties int64          `json:"normalized_counterparties"`
	DuplicateCounterparties  int64          `json:"duplicate_counterparties"`
	AverageQualityScore      float64        `json:"average_quality_score"`
	TopCategories            map[string]int `json:"top_categories"`
}

// CounterpartyActivity активность контрагентов
type CounterpartyActivity struct {
	ID             string    `json:"id"`
	CounterpartyID string    `json:"counterparty_id"`
	Action         string    `json:"action"`
	Description    string    `json:"description"`
	Timestamp      time.Time `json:"timestamp"`
}

// ============================================================================
// Client Domain Models
// ============================================================================

// Client представляет клиента системы
type Client struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	LegalName    string    `json:"legal_name"`
	Description  string    `json:"description"`
	ContactEmail string    `json:"contact_email"`
	ContactPhone string    `json:"contact_phone"`
	TaxID        string    `json:"tax_id"`
	Country      string    `json:"country"`
	Status       string    `json:"status"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ClientProject представляет проект клиента
type ClientProject struct {
	ID                 int       `json:"id"`
	ClientID           int       `json:"client_id"`
	Name               string    `json:"name"`
	ProjectType        string    `json:"project_type"`
	Description        string    `json:"description"`
	SourceSystem       string    `json:"source_system"`
	Status             string    `json:"status"`
	TargetQualityScore float64   `json:"target_quality_score"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ClientBenchmark представляет эталонную запись клиента
type ClientBenchmark struct {
	ID              int        `json:"id"`
	ClientProjectID int        `json:"client_project_id"`
	OriginalName    string     `json:"original_name"`
	NormalizedName  string     `json:"normalized_name"`
	Category        string     `json:"category"`
	Subcategory     string     `json:"subcategory"`
	Attributes      string     `json:"attributes"` // JSON
	QualityScore    float64    `json:"quality_score"`
	IsApproved      bool       `json:"is_approved"`
	ApprovedBy      string     `json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	SourceDatabase  string     `json:"source_database"`
	UsageCount      int        `json:"usage_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ClientFilter фильтр для поиска клиентов
type ClientFilter struct {
	Name   string
	Status []string
	TaxID  string
	Email  string
	Limit  int
	Offset int
}

// Project представляет проект (отдельная модель для Project domain)
type Project struct {
	ID                 int       `json:"id"`
	ClientID           int       `json:"client_id"`
	Name               string    `json:"name"`
	ProjectType        string    `json:"project_type"`
	Description        string    `json:"description"`
	SourceSystem       string    `json:"source_system"`
	Status             string    `json:"status"`
	TargetQualityScore float64   `json:"target_quality_score"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ProjectFilter фильтр для поиска проектов
type ProjectFilter struct {
	Name     string
	Type     string
	Status   []string
	ClientID string
	Limit    int
	Offset   int
}

// ============================================================================
// Database Domain Models
// ============================================================================

// Database представляет подключение к базе данных
type Database struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Type             string     `json:"type"`
	Path             string     `json:"path"`
	ConnectionString string     `json:"connection_string"`
	Status           string     `json:"status"`
	LastConnected    *time.Time `json:"last_connected,omitempty"`
	SchemaVersion    string     `json:"schema_version"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// DatabaseFilter фильтр для поиска баз данных
type DatabaseFilter struct {
	Name   string
	Type   string
	Status []string
	Path   string
	Limit  int
	Offset int
}

// DatabaseSchema схема базы данных
type DatabaseSchema struct {
	Tables []Table `json:"tables"`
}

// Table представляет таблицу в базе данных
type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// Column представляет колонку в таблице
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Primary  bool   `json:"primary"`
}

// Backup представляет бэкап базы данных
type Backup struct {
	ID          string    `json:"id"`
	DatabaseID  string    `json:"database_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
}

// ============================================================================
// Snapshot Domain Models
// ============================================================================

// Snapshot представляет срез данных
type Snapshot struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SnapshotType string    `json:"snapshot_type"`
	ProjectID    *int      `json:"project_id,omitempty"`
	ClientID     *int      `json:"client_id,omitempty"`
	CreatedBy    *int      `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SnapshotComparison сравнение двух срезов
type SnapshotComparison struct {
	SnapshotID1 int                     `json:"snapshot_id1"`
	SnapshotID2 int                     `json:"snapshot_id2"`
	Changes     []SnapshotChange        `json:"changes"`
	Statistics  SnapshotComparisonStats `json:"statistics"`
}

// SnapshotChange изменение между срезами
type SnapshotChange struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	Field      string `json:"field"`
	OldValue   string `json:"old_value"`
	NewValue   string `json:"new_value"`
	ChangeType string `json:"change_type"` // "added", "removed", "modified"
}

// SnapshotComparisonStats статистика сравнения
type SnapshotComparisonStats struct {
	TotalEntities    int `json:"total_entities"`
	AddedEntities    int `json:"added_entities"`
	RemovedEntities  int `json:"removed_entities"`
	ModifiedEntities int `json:"modified_entities"`
}

// SnapshotEvolution эволюция среза
type SnapshotEvolution struct {
	SnapshotID int                       `json:"snapshot_id"`
	Entities   []SnapshotEvolutionEntity `json:"entities"`
}

// SnapshotEvolutionEntity эволюция сущности
type SnapshotEvolutionEntity struct {
	EntityID string                     `json:"entity_id"`
	Name     string                     `json:"name"`
	Status   string                     `json:"status"` // "new", "modified", "removed", "stable"
	History  []SnapshotEvolutionHistory `json:"history"`
}

// SnapshotEvolutionHistory история изменений
type SnapshotEvolutionHistory struct {
	SnapshotID int       `json:"snapshot_id"`
	Name       string    `json:"name"`
	ChangedAt  time.Time `json:"changed_at"`
	ChangeType string    `json:"change_type"`
}

// SnapshotMetrics метрики среза
type SnapshotMetrics struct {
	SnapshotID    int                          `json:"snapshot_id"`
	QualityScores map[int]float64              `json:"quality_scores"`
	Improvements  []SnapshotQualityImprovement `json:"improvements"`
	OverallTrend  string                       `json:"overall_trend"`
}

// SnapshotQualityImprovement улучшение качества в срезе
type SnapshotQualityImprovement struct {
	Metric      string  `json:"metric"`
	FromValue   float64 `json:"from_value"`
	ToValue     float64 `json:"to_value"`
	Improvement float64 `json:"improvement"`
}
