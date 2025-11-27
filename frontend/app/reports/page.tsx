'use client'

import { useState, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Download, FileText, Loader2, AlertCircle, BarChart3, CheckCircle2 } from 'lucide-react'
import jsPDF from 'jspdf'
import html2canvas from 'html2canvas'
import { useError } from '@/contexts/ErrorContext'
import { apiClientJson } from '@/lib/api-client'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { NormalizationReport, DataQualityReport } from '@/types/reports'
import { toast } from 'sonner'
import { Skeleton } from '@/components/ui/skeleton'
import { 
  exportNormalizationReportToCSV, 
  exportDataQualityReportToCSV, 
  exportReportToJSON 
} from '@/lib/export-reports'
import { ExportButton, type ExportOption } from '@/components/common/export-button'

// Удаляем локальные интерфейсы, используем из types/reports
/* 
interface NormalizationReport {
  metadata: {
    generated_at: string
    report_version: string
    total_databases: number
    total_projects: number
  }
  overall_stats: {
    total_databases_processed: number
    total_counterparties: number
    total_nomenclature: number
    total_duplicate_groups: number
    total_errors: number
    average_quality_score: number
  }
  counterparty_analysis: {
    total_records_before: number
    total_records_after: number
    reduction_percentage: number
    duplicate_groups_found: number
    top_normalized_names: Array<{ name: string; frequency: number; percentage: number }>
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
  nomenclature_analysis: {
    total_records_before: number
    total_records_after: number
    reduction_percentage: number
    duplicate_groups_found: number
    top_normalized_names: Array<{ name: string; frequency: number; percentage: number }>
    validation_errors: number
    normalization_errors: number
    average_quality_score: number
  }
  provider_performance: Array<{
    id: string
    name: string
    total_requests: number
    successful_requests: number
    failed_requests: number
    average_latency_ms: number
    requests_per_second: number
  }>
  database_breakdown: Array<{
    database_id: number
    database_name: string
    file_path: string
    counterparties: number
    nomenclature: number
    duplicate_groups: number
    errors: number
    quality_score: number
    last_processed?: string
  }>
  recommendations: string[]
}
*/

/* 
interface DataQualityReport {
  metadata: {
    generated_at: string
    report_version: string
    total_databases: number
    total_projects: number
  }
  overall_score: {
    score: number
    completeness: number
    uniqueness: number
    consistency: number
    data_quality: string
  }
  counterparty_stats: {
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
  nomenclature_stats: {
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
  database_breakdown: Array<{
    database_id: number
    database_name: string
    file_path: string
    project_id: number
    project_name: string
    client_id: number
    counterparties: number
    nomenclature: number
    completeness_score: number
    potential_duplicate_rate: number
    inconsistencies_count: number
    status: string
    error_message?: string
  }>
  recommendations: string[]
}
*/

type ReportType = 'normalization' | 'data-quality'

export default function ReportsPage() {
  const { handleError } = useError()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [normalizationReport, setNormalizationReport] = useState<NormalizationReport | null>(null)
  const [dataQualityReport, setDataQualityReport] = useState<DataQualityReport | null>(null)
  const [activeTab, setActiveTab] = useState<ReportType>('normalization')

  const generateNormalizationReport = useCallback(async () => {
    setLoading(true)
    setError(null)
    setNormalizationReport(null)

    try {
      const data = await apiClientJson<NormalizationReport>('/api/reports/generate-normalization-report', {
        method: 'POST',
      })
      
      // Валидация ответа
      if (!data || !data.metadata) {
        throw new Error('Получен некорректный ответ от сервера')
      }
      
      setNormalizationReport(data)
      setActiveTab('normalization')
      toast.success('Отчет по нормализации сгенерирован', {
        description: `Обработано ${data.metadata.total_databases} баз данных из ${data.metadata.total_projects} проектов`,
      })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Не удалось сгенерировать отчет по нормализации'
      handleError(err, errorMessage)
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }, [handleError])

  const generateDataQualityReport = useCallback(async () => {
    setLoading(true)
    setError(null)
    setDataQualityReport(null)

    try {
      const data = await apiClientJson<DataQualityReport>('/api/reports/generate-data-quality-report', {
        method: 'POST',
      })
      
      // Валидация ответа
      if (!data || !data.metadata) {
        throw new Error('Получен некорректный ответ от сервера')
      }
      
      setDataQualityReport(data)
      setActiveTab('data-quality')
      toast.success('Отчет о качестве данных сгенерирован', {
        description: `Общий балл качества: ${data.overall_score.score.toFixed(1)}% (${data.overall_score.data_quality === 'excellent' ? 'Отлично' : data.overall_score.data_quality === 'good' ? 'Хорошо' : data.overall_score.data_quality === 'fair' ? 'Удовлетворительно' : 'Плохо'})`,
      })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Не удалось сгенерировать отчет о качестве данных'
      handleError(err, errorMessage)
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }, [handleError])

  const downloadPDF = useCallback(async (reportType: ReportType) => {
    // Валидация наличия отчета
    if (reportType === 'normalization' && !normalizationReport) {
      setError('Сначала сгенерируйте отчет по нормализации')
      handleError(new Error('Report not generated'), 'Сначала сгенерируйте отчет по нормализации')
      return
    }
    
    if (reportType === 'data-quality' && !dataQualityReport) {
      setError('Сначала сгенерируйте отчет о качестве данных')
      handleError(new Error('Report not generated'), 'Сначала сгенерируйте отчет о качестве данных')
      return
    }
    
    const reportId = reportType === 'normalization' ? 'normalization-report-content' : 'data-quality-report-content'
    const element = document.getElementById(reportId)
    if (!element) {
      const error = new Error('Report content not found')
      setError('Не удалось найти содержимое отчета для экспорта')
      handleError(error, 'Не удалось найти содержимое отчета для экспорта')
      return
    }

    setLoading(true)
    try {
      // Конвертируем в canvas
      const canvas = await html2canvas(element, {
        scale: 2,
        useCORS: true,
        logging: false,
      })

      const imgData = canvas.toDataURL('image/png')
      const pdf = new jsPDF('p', 'mm', 'a4')
      const imgWidth = 210
      const pageHeight = 295
      const imgHeight = (canvas.height * imgWidth) / canvas.width
      let heightLeft = imgHeight
      let position = 0

      // Добавляем первую страницу
      pdf.addImage(imgData, 'PNG', 0, position, imgWidth, imgHeight)
      heightLeft -= pageHeight

      // Добавляем остальные страницы
      while (heightLeft >= 0) {
        position = heightLeft - imgHeight
        pdf.addPage()
        pdf.addImage(imgData, 'PNG', 0, position, imgWidth, imgHeight)
        heightLeft -= pageHeight
      }

      // Сохраняем PDF
      const fileName = `${reportType === 'normalization' ? 'normalization' : 'data-quality'}-report-${new Date().toISOString().split('T')[0]}.pdf`
      pdf.save(fileName)
      toast.success('PDF отчет скачан', {
        description: `Файл ${fileName} успешно сохранен`,
      })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Не удалось создать PDF'
      setError(errorMessage)
      handleError(err, errorMessage)
    } finally {
      setLoading(false)
    }
  }, [normalizationReport, dataQualityReport, handleError])

  return (
    <div className="container mx-auto py-8 px-4">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Отчеты</h1>
        <p className="text-muted-foreground">
          Генерация комплексных PDF-отчетов по нормализации и качеству данных
        </p>
      </div>

      {loading && !normalizationReport && !dataQualityReport && (
        <div className="space-y-6 mb-6">
          <Skeleton className="h-10 w-full" />
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
          </div>
        </div>
      )}

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as ReportType)} className="mb-6">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="normalization">Отчет по нормализации</TabsTrigger>
          <TabsTrigger value="data-quality">Отчет о качестве данных</TabsTrigger>
        </TabsList>

        <TabsContent value="normalization" className="space-y-6">
          <div className="flex gap-4">
            <Button
              onClick={generateNormalizationReport}
              disabled={loading}
              size="lg"
              aria-label="Сгенерировать отчет по нормализации"
              aria-busy={loading && !normalizationReport}
            >
              {loading && !normalizationReport ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Генерация отчета...
                </>
              ) : (
                <>
                  <FileText className="mr-2 h-4 w-4" />
                  Сгенерировать отчет по нормализации
                </>
              )}
            </Button>

            {normalizationReport && (
              <ExportButton
                options={[
                  {
                    label: 'Скачать PDF',
                    icon: <FileText className="mr-2 h-4 w-4" />,
                    onClick: () => downloadPDF('normalization'),
                    disabled: loading,
                  },
                  {
                    label: 'Экспорт в CSV',
                    onClick: () => {
                      exportNormalizationReportToCSV(normalizationReport)
                    },
                    disabled: loading,
                  },
                  {
                    label: 'Экспорт в JSON',
                    onClick: () => {
                      exportReportToJSON(normalizationReport, 'normalization')
                    },
                    disabled: loading,
                  },
                ]}
                disabled={loading}
                loading={loading}
                label="Экспорт отчета"
                aria-label="Экспорт отчета по нормализации"
              />
            )}
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {normalizationReport && (
            <div id="normalization-report-content" className="bg-white p-8 space-y-8" style={{ minHeight: '1000px' }}>
          {/* Заголовок отчета */}
          <div className="text-center border-b pb-6 mb-6">
            <h1 className="text-4xl font-bold mb-2">Отчет по нормализации данных</h1>
            <p className="text-lg text-gray-600">
              Сгенерирован: {new Date(normalizationReport.metadata.generated_at).toLocaleString('ru-RU')}
            </p>
            <p className="text-sm text-gray-500">
              Версия отчета: {normalizationReport.metadata.report_version}
            </p>
          </div>

          {/* Executive Summary */}
          <section>
            <h2 className="text-2xl font-bold mb-4">Сводка</h2>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium">Проектов</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{normalizationReport.metadata.total_projects}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium">Баз данных</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{normalizationReport.metadata.total_databases}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium">Контрагентов</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{normalizationReport.overall_stats.total_counterparties.toLocaleString()}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium">Номенклатуры</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{normalizationReport.overall_stats.total_nomenclature.toLocaleString()}</div>
                </CardContent>
              </Card>
            </div>
          </section>

          {/* Overall Statistics */}
          <section>
            <h2 className="text-2xl font-bold mb-4">Общая статистика</h2>
            <div className="grid grid-cols-2 gap-4">
              <Card>
                <CardHeader>
                  <CardTitle>Обработка данных</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span>Групп дубликатов:</span>
                      <span className="font-bold">{normalizationReport.overall_stats.total_duplicate_groups}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Средний quality score:</span>
                      <span className="font-bold">{(normalizationReport.overall_stats.average_quality_score * 100).toFixed(1)}%</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Ошибок:</span>
                      <span className="font-bold">{normalizationReport.overall_stats.total_errors}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle>Качество данных</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span>Контрагенты (quality):</span>
                      <span className="font-bold">{(normalizationReport.counterparty_analysis.average_quality_score * 100).toFixed(1)}%</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Номенклатура (quality):</span>
                      <span className="font-bold">{(normalizationReport.nomenclature_analysis.average_quality_score * 100).toFixed(1)}%</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </section>

          {/* Counterparty Analysis */}
          <section>
            <h2 className="text-2xl font-bold mb-4">Анализ контрагентов</h2>
            <div className="grid grid-cols-2 gap-4 mb-4">
              <Card>
                <CardHeader>
                  <CardTitle>Статистика</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span>Записей до:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.total_records_before.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Записей после:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.total_records_after.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Групп дубликатов:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.duplicate_groups_found}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Снижение:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.reduction_percentage.toFixed(1)}%</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle>Обогащение</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span>Обогащено:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.enrichment_stats.total_enriched}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Процент обогащения:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.enrichment_stats.enrichment_rate.toFixed(1)}%</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Совпадений с эталонами:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.enrichment_stats.benchmark_matches}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Внешнее обогащение:</span>
                      <span className="font-bold">{normalizationReport.counterparty_analysis.enrichment_stats.external_enrichment}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {normalizationReport.counterparty_analysis.top_normalized_names.length > 0 && (
              <Card className="mb-4">
                <CardHeader>
                  <CardTitle>Топ-10 нормализованных имен</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {normalizationReport.counterparty_analysis.top_normalized_names.map((item, idx) => (
                      <div key={idx} className="flex justify-between items-center">
                        <span className="text-sm">{item.name}</span>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">{item.frequency}</span>
                          <span className="text-xs text-gray-500">({item.percentage.toFixed(1)}%)</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </section>

          {/* Nomenclature Analysis */}
          <section>
            <h2 className="text-2xl font-bold mb-4">Анализ номенклатуры</h2>
            <div className="grid grid-cols-2 gap-4 mb-4">
              <Card>
                <CardHeader>
                  <CardTitle>Статистика</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span>Записей до:</span>
                      <span className="font-bold">{normalizationReport.nomenclature_analysis.total_records_before.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Записей после:</span>
                      <span className="font-bold">{normalizationReport.nomenclature_analysis.total_records_after.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Групп дубликатов:</span>
                      <span className="font-bold">{normalizationReport.nomenclature_analysis.duplicate_groups_found}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Снижение:</span>
                      <span className="font-bold">{normalizationReport.nomenclature_analysis.reduction_percentage.toFixed(1)}%</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {normalizationReport.nomenclature_analysis.top_normalized_names.length > 0 && (
              <Card className="mb-4">
                <CardHeader>
                  <CardTitle>Топ-10 нормализованных имен</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {normalizationReport.nomenclature_analysis.top_normalized_names.map((item, idx) => (
                      <div key={idx} className="flex justify-between items-center">
                        <span className="text-sm">{item.name}</span>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">{item.frequency}</span>
                          <span className="text-xs text-gray-500">({item.percentage.toFixed(1)}%)</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </section>

          {/* Provider Performance */}
          {normalizationReport.provider_performance && normalizationReport.provider_performance.length > 0 && (
            <section>
              <h2 className="text-2xl font-bold mb-4">Производительность провайдеров</h2>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                {normalizationReport.provider_performance.map((provider) => (
                  <Card key={provider.id}>
                    <CardHeader>
                      <CardTitle>{provider.name}</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span>Всего запросов:</span>
                          <span className="font-bold">{provider.total_requests.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Успешных:</span>
                          <span className="font-bold text-green-600">{provider.successful_requests.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Ошибок:</span>
                          <span className="font-bold text-red-600">{provider.failed_requests.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Средняя задержка:</span>
                          <span className="font-bold">{provider.average_latency_ms.toFixed(0)} мс</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Запросов/сек:</span>
                          <span className="font-bold">{provider.requests_per_second.toFixed(2)}</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </section>
          )}

          {/* Database Breakdown */}
          {normalizationReport.database_breakdown && normalizationReport.database_breakdown.length > 0 && (
            <section>
              <h2 className="text-2xl font-bold mb-4">Разбивка по базам данных</h2>
              <div className="overflow-x-auto">
                <table className="w-full border-collapse border border-gray-300">
                  <thead>
                    <tr className="bg-gray-100">
                      <th className="border border-gray-300 p-2 text-left">База данных</th>
                      <th className="border border-gray-300 p-2 text-center">Контрагенты</th>
                      <th className="border border-gray-300 p-2 text-center">Номенклатура</th>
                      <th className="border border-gray-300 p-2 text-center">Дубликаты</th>
                      <th className="border border-gray-300 p-2 text-center">Quality Score</th>
                    </tr>
                  </thead>
                  <tbody>
                    {normalizationReport.database_breakdown.map((db) => (
                      <tr key={db.database_id}>
                        <td className="border border-gray-300 p-2">{db.database_name}</td>
                        <td className="border border-gray-300 p-2 text-center">{db.counterparties.toLocaleString()}</td>
                        <td className="border border-gray-300 p-2 text-center">{db.nomenclature.toLocaleString()}</td>
                        <td className="border border-gray-300 p-2 text-center">{db.duplicate_groups}</td>
                        <td className="border border-gray-300 p-2 text-center">{(db.quality_score * 100).toFixed(1)}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}

          {/* Recommendations */}
          {normalizationReport.recommendations && normalizationReport.recommendations.length > 0 && (
            <section>
              <h2 className="text-2xl font-bold mb-4">Рекомендации</h2>
              <Card>
                <CardContent className="pt-6">
                  <ul className="space-y-2">
                    {normalizationReport.recommendations.map((rec, idx) => (
                      <li key={idx} className="flex items-start gap-2">
                        <span className="text-blue-600 mt-1">•</span>
                        <span>{rec}</span>
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            </section>
          )}
            </div>
          )}
        </TabsContent>

        <TabsContent value="data-quality" className="space-y-6">
          <div className="flex gap-4">
            <Button
              onClick={generateDataQualityReport}
              disabled={loading}
              size="lg"
              aria-label="Сгенерировать отчет о качестве данных"
              aria-busy={loading && !dataQualityReport}
            >
              {loading && !dataQualityReport ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Генерация отчета...
                </>
              ) : (
                <>
                  <BarChart3 className="mr-2 h-4 w-4" />
                  Сгенерировать отчет о качестве данных
                </>
              )}
            </Button>

            {dataQualityReport && (
              <ExportButton
                options={[
                  {
                    label: 'Скачать PDF',
                    icon: <FileText className="mr-2 h-4 w-4" />,
                    onClick: () => downloadPDF('data-quality'),
                    disabled: loading,
                  },
                  {
                    label: 'Экспорт в CSV',
                    onClick: () => {
                      exportDataQualityReportToCSV(dataQualityReport)
                    },
                    disabled: loading,
                  },
                  {
                    label: 'Экспорт в JSON',
                    onClick: () => {
                      exportReportToJSON(dataQualityReport, 'data-quality')
                    },
                    disabled: loading,
                  },
                ]}
                disabled={loading}
                loading={loading}
                label="Экспорт отчета"
                aria-label="Экспорт отчета о качестве данных"
              />
            )}
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {dataQualityReport && (
            <div id="data-quality-report-content" className="bg-white p-8 space-y-8" style={{ minHeight: '1000px' }}>
              {/* Заголовок отчета */}
              <div className="text-center border-b pb-6 mb-6">
                <h1 className="text-4xl font-bold mb-2">Отчет о качестве данных</h1>
                <p className="text-lg text-gray-600">
                  Сгенерирован: {new Date(dataQualityReport.metadata.generated_at).toLocaleString('ru-RU')}
                </p>
                <p className="text-sm text-gray-500">
                  Версия отчета: {dataQualityReport.metadata.report_version}
                </p>
              </div>

              {/* Общая оценка качества */}
              <section>
                <h2 className="text-2xl font-bold mb-4">Общая оценка качества</h2>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium">Общий балл</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{dataQualityReport.overall_score.score.toFixed(1)}%</div>
                      <p className="text-xs text-muted-foreground mt-1">
                        {dataQualityReport.overall_score.data_quality === 'excellent' ? 'Отлично' :
                         dataQualityReport.overall_score.data_quality === 'good' ? 'Хорошо' :
                         dataQualityReport.overall_score.data_quality === 'fair' ? 'Удовлетворительно' : 'Плохо'}
                      </p>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium">Полнота</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{dataQualityReport.overall_score.completeness.toFixed(1)}%</div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium">Уникальность</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{dataQualityReport.overall_score.uniqueness.toFixed(1)}%</div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm font-medium">Консистентность</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{dataQualityReport.overall_score.consistency.toFixed(1)}%</div>
                    </CardContent>
                  </Card>
                </div>
              </section>

              {/* Статистика по контрагентам */}
              <section>
                <h2 className="text-2xl font-bold mb-4">Статистика по контрагентам</h2>
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Основные метрики</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span>Всего записей:</span>
                          <span className="font-bold">{dataQualityReport.counterparty_stats.total_records.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Полнота данных:</span>
                          <span className="font-bold">{dataQualityReport.counterparty_stats.completeness_score.toFixed(1)}%</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Потенциальные дубликаты:</span>
                          <span className="font-bold">{dataQualityReport.counterparty_stats.potential_duplicate_rate.toFixed(1)}%</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle>Заполненность полей</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span>С названием:</span>
                          <span className="font-bold text-green-600">{dataQualityReport.counterparty_stats.records_with_name.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Без названия:</span>
                          <span className="font-bold text-red-600">{dataQualityReport.counterparty_stats.records_without_name.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>С ИНН:</span>
                          <span className="font-bold text-green-600">{dataQualityReport.counterparty_stats.records_with_inn.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>С БИН:</span>
                          <span className="font-bold text-green-600">{dataQualityReport.counterparty_stats.records_with_bin.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Без ИНН/БИН:</span>
                          <span className="font-bold text-red-600">{dataQualityReport.counterparty_stats.records_without_tax_id.toLocaleString()}</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>

                {dataQualityReport.counterparty_stats.top_inconsistencies.length > 0 && (
                  <Card className="mb-4">
                    <CardHeader>
                      <CardTitle>Топ несоответствий</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {dataQualityReport.counterparty_stats.top_inconsistencies.map((item, idx) => (
                          <div key={idx} className="flex justify-between items-start">
                            <div className="flex-1">
                              <span className="text-sm font-medium">{item.type}</span>
                              <p className="text-xs text-muted-foreground">{item.description}</p>
                              <p className="text-xs text-muted-foreground mt-1">Пример: {item.example}</p>
                            </div>
                            <span className="text-sm font-bold ml-4">{item.count}</span>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                )}
              </section>

              {/* Статистика по номенклатуре */}
              <section>
                <h2 className="text-2xl font-bold mb-4">Статистика по номенклатуре</h2>
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Основные метрики</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span>Всего записей:</span>
                          <span className="font-bold">{dataQualityReport.nomenclature_stats.total_records.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Полнота данных:</span>
                          <span className="font-bold">{dataQualityReport.nomenclature_stats.completeness_score.toFixed(1)}%</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Потенциальные дубликаты:</span>
                          <span className="font-bold">{dataQualityReport.nomenclature_stats.potential_duplicate_rate.toFixed(1)}%</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle>Заполненность полей</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span>С названием:</span>
                          <span className="font-bold text-green-600">{dataQualityReport.nomenclature_stats.records_with_name.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Без названия:</span>
                          <span className="font-bold text-red-600">{dataQualityReport.nomenclature_stats.records_without_name.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>С артикулом:</span>
                          <span className="font-bold text-green-600">{dataQualityReport.nomenclature_stats.records_with_article.toLocaleString()}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>С SKU:</span>
                          <span className="font-bold text-green-600">{dataQualityReport.nomenclature_stats.records_with_sku.toLocaleString()}</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>

                {dataQualityReport.nomenclature_stats.top_inconsistencies.length > 0 && (
                  <Card className="mb-4">
                    <CardHeader>
                      <CardTitle>Топ несоответствий</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {dataQualityReport.nomenclature_stats.top_inconsistencies.map((item, idx) => (
                          <div key={idx} className="flex justify-between items-start">
                            <div className="flex-1">
                              <span className="text-sm font-medium">{item.type}</span>
                              <p className="text-xs text-muted-foreground">{item.description}</p>
                              <p className="text-xs text-muted-foreground mt-1">Пример: {item.example}</p>
                            </div>
                            <span className="text-sm font-bold ml-4">{item.count}</span>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                )}
              </section>

              {/* Разбивка по базам данных */}
              {dataQualityReport.database_breakdown && dataQualityReport.database_breakdown.length > 0 && (
                <section>
                  <h2 className="text-2xl font-bold mb-4">Разбивка по базам данных</h2>
                  <div className="overflow-x-auto">
                    <table className="w-full border-collapse border border-gray-300">
                      <thead>
                        <tr className="bg-gray-100">
                          <th className="border border-gray-300 p-2 text-left">База данных</th>
                          <th className="border border-gray-300 p-2 text-center">Контрагенты</th>
                          <th className="border border-gray-300 p-2 text-center">Номенклатура</th>
                          <th className="border border-gray-300 p-2 text-center">Полнота</th>
                          <th className="border border-gray-300 p-2 text-center">Дубликаты</th>
                          <th className="border border-gray-300 p-2 text-center">Статус</th>
                        </tr>
                      </thead>
                      <tbody>
                        {dataQualityReport.database_breakdown.map((db) => (
                          <tr key={db.database_id}>
                            <td className="border border-gray-300 p-2">{db.database_name}</td>
                            <td className="border border-gray-300 p-2 text-center">{db.counterparties.toLocaleString()}</td>
                            <td className="border border-gray-300 p-2 text-center">{db.nomenclature.toLocaleString()}</td>
                            <td className="border border-gray-300 p-2 text-center">{((db as any).completeness_score || 0).toFixed(1)}%</td>
                            <td className="border border-gray-300 p-2 text-center">{((db as any).potential_duplicate_rate || 0).toFixed(1)}%</td>
                            <td className="border border-gray-300 p-2 text-center">
                              {((db as any).status || '') === 'success' ? (
                                <CheckCircle2 className="h-4 w-4 text-green-600 mx-auto" />
                              ) : (
                                <AlertCircle className="h-4 w-4 text-red-600 mx-auto" />
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </section>
              )}

              {/* Рекомендации */}
              {dataQualityReport.recommendations && dataQualityReport.recommendations.length > 0 && (
                <section>
                  <h2 className="text-2xl font-bold mb-4">Рекомендации</h2>
                  <Card>
                    <CardContent className="pt-6">
                      <ul className="space-y-2">
                        {dataQualityReport.recommendations.map((rec, idx) => (
                          <li key={idx} className="flex items-start gap-2">
                            <span className="text-blue-600 mt-1">•</span>
                            <span>{rec}</span>
                          </li>
                        ))}
                      </ul>
                    </CardContent>
                  </Card>
                </section>
              )}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}

