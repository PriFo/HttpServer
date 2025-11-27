/**
 * Утилиты для экспорта отчетов
 */

import type { NormalizationReport, DataQualityReport } from '@/types/reports'

/**
 * Экспорт отчета по нормализации в CSV
 */
export function exportNormalizationReportToCSV(report: NormalizationReport): void {
  const rows: string[][] = []
  
  // Метаданные
  rows.push(['Метаданные отчета'])
  rows.push(['Дата генерации', report.metadata.generated_at])
  rows.push(['Версия отчета', report.metadata.report_version])
  rows.push(['Всего баз данных', report.metadata.total_databases.toString()])
  rows.push(['Всего проектов', report.metadata.total_projects.toString()])
  rows.push([])
  
  // Общая статистика
  rows.push(['Общая статистика'])
  rows.push(['Обработано баз данных', report.overall_stats.total_databases_processed.toString()])
  rows.push(['Всего контрагентов', report.overall_stats.total_counterparties.toString()])
  rows.push(['Всего номенклатуры', report.overall_stats.total_nomenclature.toString()])
  rows.push(['Групп дубликатов', report.overall_stats.total_duplicate_groups.toString()])
  rows.push(['Ошибок', report.overall_stats.total_errors.toString()])
  rows.push(['Средний балл качества', report.overall_stats.average_quality_score.toFixed(2)])
  rows.push([])
  
  // Анализ контрагентов
  rows.push(['Анализ контрагентов'])
  rows.push(['Записей до', report.counterparty_analysis.total_records_before.toString()])
  rows.push(['Записей после', report.counterparty_analysis.total_records_after.toString()])
  rows.push(['Процент сокращения', `${report.counterparty_analysis.reduction_percentage.toFixed(2)}%`])
  rows.push(['Групп дубликатов', report.counterparty_analysis.duplicate_groups_found.toString()])
  rows.push(['Ошибок валидации', report.counterparty_analysis.validation_errors.toString()])
  rows.push(['Ошибок нормализации', report.counterparty_analysis.normalization_errors.toString()])
  rows.push(['Средний балл качества', report.counterparty_analysis.average_quality_score.toFixed(2)])
  rows.push([])
  
  // Статистика обогащения
  rows.push(['Статистика обогащения контрагентов'])
  rows.push(['Всего обогащено', report.counterparty_analysis.enrichment_stats.total_enriched.toString()])
  rows.push(['Процент обогащения', `${report.counterparty_analysis.enrichment_stats.enrichment_rate.toFixed(2)}%`])
  rows.push(['Совпадений с эталоном', report.counterparty_analysis.enrichment_stats.benchmark_matches.toString()])
  rows.push(['Внешнее обогащение', report.counterparty_analysis.enrichment_stats.external_enrichment.toString()])
  rows.push([])
  
  // Топ нормализованных имен контрагентов
  if (report.counterparty_analysis.top_normalized_names.length > 0) {
    rows.push(['Топ нормализованных имен контрагентов'])
    rows.push(['Название', 'Частота', 'Процент'])
    report.counterparty_analysis.top_normalized_names.forEach(item => {
      rows.push([item.name, item.frequency.toString(), `${item.percentage.toFixed(2)}%`])
    })
    rows.push([])
  }
  
  // Анализ номенклатуры
  rows.push(['Анализ номенклатуры'])
  rows.push(['Записей до', report.nomenclature_analysis.total_records_before.toString()])
  rows.push(['Записей после', report.nomenclature_analysis.total_records_after.toString()])
  rows.push(['Процент сокращения', `${report.nomenclature_analysis.reduction_percentage.toFixed(2)}%`])
  rows.push(['Групп дубликатов', report.nomenclature_analysis.duplicate_groups_found.toString()])
  rows.push(['Ошибок валидации', report.nomenclature_analysis.validation_errors.toString()])
  rows.push(['Ошибок нормализации', report.nomenclature_analysis.normalization_errors.toString()])
  rows.push(['Средний балл качества', report.nomenclature_analysis.average_quality_score.toFixed(2)])
  rows.push([])
  
  // Топ нормализованных имен номенклатуры
  if (report.nomenclature_analysis.top_normalized_names.length > 0) {
    rows.push(['Топ нормализованных имен номенклатуры'])
    rows.push(['Название', 'Частота', 'Процент'])
    report.nomenclature_analysis.top_normalized_names.forEach(item => {
      rows.push([item.name, item.frequency.toString(), `${item.percentage.toFixed(2)}%`])
    })
    rows.push([])
  }
  
  // Производительность провайдеров
  if (report.provider_performance.length > 0) {
    rows.push(['Производительность провайдеров'])
    rows.push(['Провайдер', 'Всего запросов', 'Успешных', 'Неудачных', 'Средняя задержка (мс)', 'Запросов/сек'])
    report.provider_performance.forEach(provider => {
      rows.push([
        provider.name,
        provider.total_requests.toString(),
        provider.successful_requests.toString(),
        provider.failed_requests.toString(),
        provider.average_latency_ms.toFixed(0),
        provider.requests_per_second.toFixed(2),
      ])
    })
    rows.push([])
  }
  
  // Разбивка по базам данных
  if (report.database_breakdown.length > 0) {
    rows.push(['Разбивка по базам данных'])
    rows.push(['ID БД', 'Название', 'Путь', 'Контрагенты', 'Номенклатура', 'Группы дубликатов', 'Ошибки', 'Балл качества'])
    report.database_breakdown.forEach(db => {
      rows.push([
        db.database_id.toString(),
        db.database_name,
        db.file_path,
        db.counterparties.toString(),
        db.nomenclature.toString(),
        db.duplicate_groups.toString(),
        db.errors.toString(),
        db.quality_score.toFixed(2),
      ])
    })
    rows.push([])
  }
  
  // Рекомендации
  if (report.recommendations.length > 0) {
    rows.push(['Рекомендации'])
    report.recommendations.forEach((rec, idx) => {
      rows.push([`${idx + 1}. ${rec}`])
    })
  }
  
  const csvContent = rows.map(row => row.map(cell => `"${cell}"`).join(',')).join('\n')
  const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', `normalization-report-${new Date().toISOString().split('T')[0]}.csv`)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

/**
 * Экспорт отчета о качестве данных в CSV
 */
export function exportDataQualityReportToCSV(report: DataQualityReport): void {
  const rows: string[][] = []
  
  // Метаданные
  rows.push(['Метаданные отчета'])
  rows.push(['Дата генерации', report.metadata.generated_at])
  rows.push(['Версия отчета', report.metadata.report_version])
  rows.push(['Всего баз данных', report.metadata.total_databases.toString()])
  rows.push(['Всего проектов', report.metadata.total_projects.toString()])
  rows.push([])
  
  // Общий балл
  rows.push(['Общий балл качества'])
  rows.push(['Балл', report.overall_score.score.toFixed(2)])
  rows.push(['Полнота', report.overall_score.completeness.toFixed(2)])
  rows.push(['Уникальность', report.overall_score.uniqueness.toFixed(2)])
  rows.push(['Согласованность', report.overall_score.consistency.toFixed(2)])
  rows.push(['Качество данных', report.overall_score.data_quality])
  rows.push([])
  
  // Статистика контрагентов
  rows.push(['Статистика контрагентов'])
  rows.push(['Всего записей', report.counterparty_stats.total_records.toString()])
  rows.push(['Балл полноты', report.counterparty_stats.completeness_score.toFixed(2)])
  rows.push(['Балл уникальности', ((report.counterparty_stats as any).uniqueness_score || 0).toFixed(2)])
  rows.push(['Балл согласованности', ((report.counterparty_stats as any).consistency_score || 0).toFixed(2)])
  rows.push(['Дубликатов', ((report.counterparty_stats as any).duplicates || 0).toString()])
  rows.push(['Несоответствий', ((report.counterparty_stats as any).violations || 0).toString()])
  rows.push([])
  
  // Статистика номенклатуры
  rows.push(['Статистика номенклатуры'])
  rows.push(['Всего записей', report.nomenclature_stats.total_records.toString()])
  rows.push(['Балл полноты', report.nomenclature_stats.completeness_score.toFixed(2)])
  rows.push(['Балл уникальности', ((report.nomenclature_stats as any).uniqueness_score || 0).toFixed(2)])
  rows.push(['Балл согласованности', ((report.nomenclature_stats as any).consistency_score || 0).toFixed(2)])
  rows.push(['Дубликатов', ((report.nomenclature_stats as any).duplicates || 0).toString()])
  rows.push(['Несоответствий', ((report.nomenclature_stats as any).violations || 0).toString()])
  rows.push([])
  
  // Топ несоответствий контрагентов
  const topViolations = ((report.counterparty_stats as any).top_violations || []) as Array<{ type: string; count: number; percentage: number }>
  if (topViolations.length > 0) {
    rows.push(['Топ несоответствий контрагентов'])
    rows.push(['Тип', 'Количество', 'Процент'])
    topViolations.forEach(violation => {
      rows.push([violation.type, violation.count.toString(), `${violation.percentage.toFixed(2)}%`])
    })
    rows.push([])
  }
  
  // Топ несоответствий номенклатуры
  const topNomenclatureViolations = ((report.nomenclature_stats as any).top_violations || []) as Array<{ type: string; count: number; percentage: number }>
  if (topNomenclatureViolations.length > 0) {
    rows.push(['Топ несоответствий номенклатуры'])
    rows.push(['Тип', 'Количество', 'Процент'])
    topNomenclatureViolations.forEach(violation => {
      rows.push([violation.type, violation.count.toString(), `${violation.percentage.toFixed(2)}%`])
    })
    rows.push([])
  }
  
  // Разбивка по базам данных
  if (report.database_breakdown.length > 0) {
    rows.push(['Разбивка по базам данных'])
    rows.push(['ID БД', 'Название', 'Балл качества', 'Полнота', 'Уникальность', 'Согласованность'])
    report.database_breakdown.forEach(db => {
      rows.push([
        db.database_id.toString(),
        db.database_name,
        db.quality_score.toFixed(2),
        ((db as any).completeness || 0).toFixed(2),
        ((db as any).uniqueness || 0).toFixed(2),
        ((db as any).consistency || 0).toFixed(2),
      ])
    })
    rows.push([])
  }
  
  // Рекомендации
  if (report.recommendations.length > 0) {
    rows.push(['Рекомендации'])
    report.recommendations.forEach((rec, idx) => {
      rows.push([`${idx + 1}. ${rec}`])
    })
  }
  
  const csvContent = rows.map(row => row.map(cell => `"${cell}"`).join(',')).join('\n')
  const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', `data-quality-report-${new Date().toISOString().split('T')[0]}.csv`)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

/**
 * Экспорт отчета в JSON
 */
export function exportReportToJSON<T extends NormalizationReport | DataQualityReport>(report: T, type: 'normalization' | 'data-quality'): void {
  const jsonContent = JSON.stringify(report, null, 2)
  const blob = new Blob([jsonContent], { type: 'application/json' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', `${type}-report-${new Date().toISOString().split('T')[0]}.json`)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

