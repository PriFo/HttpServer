// Типы для отчетов

export interface DataQualityReportMetadata {
  generated_at: string
  report_version: string
  total_databases: number
  total_projects: number
}

export interface OverallQualityScore {
  score: number // 0-100
  completeness: number // % полноты данных
  uniqueness: number // % уникальности (100 - % дубликатов)
  consistency: number // % консистентности
  data_quality: 'excellent' | 'good' | 'fair' | 'poor'
}

export interface CounterpartyQualityStats {
  total_records: number
  completeness_score: number
  potential_duplicate_rate: number
  records_with_name: number
  records_with_inn: number
  records_with_bin: number
  records_without_name: number
  records_without_tax_id: number
  invalid_inn_format: number
  invalid_bin_format: number
  top_inconsistencies: Array<{
    type: string
    count: number
    example: string
    description: string
  }>
  name_length_stats: {
    min: number
    max: number
    average: number
  }
}

export interface NomenclatureQualityStats {
  total_records: number
  completeness_score: number
  potential_duplicate_rate: number
  records_with_name: number
  records_without_name: number
  records_with_article: number
  records_without_article: number
  records_with_sku: number
  records_without_sku: number
  top_inconsistencies: Array<{
    type: string
    count: number
    example: string
    description: string
  }>
  name_length_stats: {
    min: number
    max: number
    average: number
  }
  unit_of_measure_stats: {
    unique_count: number
    top_variations: Array<{
      unit: string
      count: number
    }>
    inconsistencies: string[]
  }
  attribute_completeness: {
    brand_percent: number
    manufacturer_percent: number
    country_percent: number
    records_with_brand: number
    records_with_manufacturer: number
    records_with_country: number
    top_brands: Array<{
      unit: string
      count: number
    }>
  }
}

export interface DatabaseQualityStats {
  database_id: number
  database_name: string
  file_path: string
  counterparties: number
  nomenclature: number
  duplicate_groups: number
  quality_score: number
  last_processed?: string
}

export interface DataQualityReport {
  metadata: DataQualityReportMetadata
  overall_score: OverallQualityScore
  counterparty_stats: CounterpartyQualityStats
  nomenclature_stats: NomenclatureQualityStats
  database_breakdown: DatabaseQualityStats[]
  recommendations: string[]
}

export interface NormalizationReportMetadata {
  generated_at: string
  report_version: string
  total_databases: number
  total_projects: number
}

export interface NormalizationReportOverallStats {
  total_databases_processed: number
  total_counterparties: number
  total_nomenclature: number
  total_duplicate_groups: number
  total_errors: number
  average_quality_score: number
}

export interface NormalizationReportCounterpartyAnalysis {
  total_records_before: number
  total_records_after: number
  reduction_percentage: number
  duplicate_groups_found: number
  top_normalized_names: Array<{
    name: string
    frequency: number
    percentage: number
  }>
  validation_errors: number
  normalization_errors: number
  average_quality_score: number
  enrichment_stats: {
    total_enriched: number
    enrichment_rate: number
    benchmark_matches: number
    external_enrichment: number
  }
}

export interface NormalizationReportNomenclatureAnalysis {
  total_records_before: number
  total_records_after: number
  reduction_percentage: number
  duplicate_groups_found: number
  top_normalized_names: Array<{
    name: string
    frequency: number
    percentage: number
  }>
  validation_errors: number
  normalization_errors: number
  average_quality_score: number
}

export interface ProviderPerformance {
  id: string
  name: string
  total_requests: number
  successful_requests: number
  failed_requests: number
  average_latency_ms: number
  requests_per_second: number
}

export interface NormalizationReportDatabaseBreakdown {
  database_id: number
  database_name: string
  file_path: string
  counterparties: number
  nomenclature: number
  duplicate_groups: number
  errors: number
  quality_score: number
  last_processed?: string
}

export interface NormalizationReport {
  metadata: NormalizationReportMetadata
  overall_stats: NormalizationReportOverallStats
  counterparty_analysis: NormalizationReportCounterpartyAnalysis
  nomenclature_analysis: NormalizationReportNomenclatureAnalysis
  provider_performance: ProviderPerformance[]
  database_breakdown: NormalizationReportDatabaseBreakdown[]
  recommendations: string[]
}

