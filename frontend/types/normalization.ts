'use client'

export interface PipelineStageMetric {
  records?: number
  time?: string
  confidence?: number
  issues?: number
}

export interface PipelineStage {
  id: string
  name: string
  description: string
  status: 'pending' | 'processing' | 'active' | 'completed' | 'error'
  icon?: string
  metrics?: PipelineStageMetric
}

export interface PipelineStatsData {
  total_records: number
  overall_progress: number
  stage_stats: Array<{
    stage_number: string
    stage_name: string
    completed: number
    total: number
    progress: number
    avg_confidence: number
    errors: number
    pending: number
    last_updated?: string
  }>
  quality_metrics: {
    avg_final_confidence: number
    manual_review_required: number
    classifier_success: number
    ai_success: number
    fallback_used: number
  }
  processing_duration?: string
  last_updated?: string
}

export interface ActivityEvent {
  id: string
  timestamp: string
  user: string
  action: string
  type: 'success' | 'error' | 'info' | 'warning'
  description: string
  details?: Array<{
    field: string
    old_value: unknown
    new_value: unknown
  }>
}

export interface ExportHistory {
  id: string
  timestamp: string
  format: string
  dataType: string
  recordCount: number
  status: 'completed' | 'failed' | 'processing'
  downloadUrl?: string
}
// Типы для нормализации

export interface NormalizationStatus {
  isRunning: boolean
  progress: number
  processed: number
  total: number
  success?: number
  errors?: number
  groupsCreated?: number // Количество созданных групп нормализации
  duplicatesFound?: number // Количество найденных дубликатов
  currentStep: string
  logs: string[]
  startTime?: string
  elapsedTime?: string
  rate?: number
  speed?: number // Скорость обработки (записей/сек)
  kpvedClassified?: number
  kpvedTotal?: number
  kpvedProgress?: number
  // Поля для нормализации контрагентов
  sessions?: Array<{
    id: number
    project_database_id: number
    database_name: string
    status: string
    created_at: string
    finished_at?: string
  }>
  databases?: Array<{
    id: number
    name: string
    file_path: string
    is_active: boolean
    last_session?: {
      id: number
      status: string
      created_at: string
      finished_at?: string | null
    }
  }>
  active_sessions_count?: number
  total_databases_count?: number
  // Информация о провайдерах, использованных для нормализации
  providers_used?: Array<{
    provider_id: string
    provider_name: string
    provider_type: 'dadata' | 'adata' | 'openrouter' | 'huggingface' | 'arliai' | 'edenai'
    requests_count: number
    success_rate: number
  }>
}

export type ProviderType = 'dadata' | 'adata' | 'openrouter' | 'huggingface' | 'arliai' | 'edenai'

export interface ProviderInfo {
  provider_id: string
  provider_name: string
  provider_type: ProviderType
  requests_count: number
  success_rate: number
}

export type QualityDimension = 'completeness' | 'accuracy' | 'consistency' | 'timeliness'

export interface QualityMetric {
  score: number
  issues: number
  trend: number
}

export type QualityMetrics = Record<QualityDimension, QualityMetric>

export interface DuplicateCluster {
  id: string
  items: Array<{
    id: string
    name: string
    code?: string
    category?: string
    similarity?: number
    [key: string]: any
  }>
  similarity_score?: number
  suggested_master?: string
  [key: string]: any
}

export interface DuplicateClustersResponse {
  clusters: DuplicateCluster[]
  total?: number
  [key: string]: any
}

export interface NormalizationMetrics {
  totalGroups?: number
  groupsTrend?: number
  avgQuality?: number
  totalProcessed?: number
  processedTrend?: number
  issuesCount?: number
  [key: string]: any
}

// Типы для метрик заполненности данных
export interface CompletenessMetrics {
  nomenclature_completeness?: {
    articles_percent: number
    units_percent: number
    descriptions_percent: number
    overall_completeness: number
  }
  counterparty_completeness?: {
    inn_percent: number
    address_percent: number
    contacts_percent: number
    overall_completeness: number
  }
}

// Типы для статистики предпросмотра нормализации
export interface DatabasePreviewStats {
  database_id: number
  database_name: string
  file_path: string
  nomenclature_count: number
  counterparty_count: number
  total_records: number
  database_size: number
  error?: string
  is_accessible?: boolean
  is_valid?: boolean
  completeness?: CompletenessMetrics
}

export interface PreviewStatsResponse {
  total_databases: number
  accessible_databases?: number
  valid_databases?: number
  total_nomenclature: number
  total_counterparties: number
  total_records: number
  estimated_duplicates: number
  duplicate_groups?: number
  completeness_metrics?: CompletenessMetrics
  databases: DatabasePreviewStats[]
}

// Тип для выбора типа нормализации
export type NormalizationType = 'nomenclature' | 'counterparties' | 'both'

