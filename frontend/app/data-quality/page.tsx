'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DynamicBarChart,
  DynamicBar,
  DynamicPieChart,
  DynamicPie,
  DynamicCell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from '@/lib/recharts-dynamic'
import {
  Download,
  FileText,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Database,
  TrendingUp,
  TrendingDown,
  Play,
  BarChart3,
} from 'lucide-react'
import jsPDF from 'jspdf'
import html2canvas from 'html2canvas'
import { toast } from 'sonner'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { FadeIn } from '@/components/animations/fade-in'
import { useApiClient } from '@/hooks/useApiClient'

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

export default function DataQualityPage() {
  const router = useRouter()
  const { post } = useApiClient()
  const [loading, setLoading] = useState(false)
  const [reportData, setReportData] = useState<DataQualityReport | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [startingNormalization, setStartingNormalization] = useState(false)

  const generateReport = async () => {
    setLoading(true)
    setError(null)
    setReportData(null)

    try {
      const data = await post<DataQualityReport>('/api/reports/generate-data-quality-report', {}, { skipErrorHandler: true })
      setReportData(data)
      toast.success('Отчет о качестве данных успешно сгенерирован')
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      setError('Не удалось сгенерировать отчет')
    } finally {
      setLoading(false)
    }
  }

  const downloadPDF = async () => {
    if (!reportData) return

    setLoading(true)
    try {
      const element = document.getElementById('report-content')
      if (!element) {
        throw new Error('Report content not found')
      }

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
      const fileName = `data-quality-report-${new Date().toISOString().split('T')[0]}.pdf`
      pdf.save(fileName)
      toast.success('PDF отчет успешно скачан')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate PDF')
      toast.error('Ошибка генерации PDF')
    } finally {
      setLoading(false)
    }
  }

  const startNormalization = async () => {
    if (!reportData || reportData.database_breakdown.length === 0) {
      toast.error('Нет данных для нормализации')
      return
    }

    // Получаем первый проект из отчета
    const firstDB = reportData.database_breakdown.find(db => db.status === 'success')
    if (!firstDB) {
      toast.error('Нет доступных баз данных для нормализации')
      return
    }

    setStartingNormalization(true)
    try {
      // Получаем client_id из данных отчета
      const clientId = firstDB.client_id || 1

      // Запускаем нормализацию для проекта
      await post(
        `/api/clients/${clientId}/projects/${firstDB.project_id}/normalization/start`,
        {
          all_active: true,
          use_kpved: false,
          use_okpd2: false,
        },
        { skipErrorHandler: true }
      )

      toast.success('Нормализация запущена')
      // Перенаправляем на страницу мониторинга
      router.push('/monitoring')
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
    } finally {
      setStartingNormalization(false)
    }
  }

  const getQualityColor = (quality: string) => {
    switch (quality) {
      case 'excellent':
        return 'bg-green-500'
      case 'good':
        return 'bg-blue-500'
      case 'fair':
        return 'bg-yellow-500'
      case 'poor':
        return 'bg-red-500'
      default:
        return 'bg-gray-500'
    }
  }

  const getQualityLabel = (quality: string) => {
    switch (quality) {
      case 'excellent':
        return 'Отличное'
      case 'good':
        return 'Хорошее'
      case 'fair':
        return 'Удовлетворительное'
      case 'poor':
        return 'Плохое'
      default:
        return 'Неизвестно'
    }
  }

  // Данные для графиков
  const databaseChartData = reportData?.database_breakdown
    .filter(db => db.status === 'success')
    .map(db => ({
      name: db.database_name.length > 20 ? db.database_name.substring(0, 20) + '...' : db.database_name,
      completeness: db.completeness_score,
      duplicates: db.potential_duplicate_rate,
    })) || []

  const qualityDistribution = reportData
    ? [
        { name: 'Полнота', value: reportData.overall_score.completeness },
        { name: 'Уникальность', value: reportData.overall_score.uniqueness },
        { name: 'Консистентность', value: reportData.overall_score.consistency },
      ]
    : []

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b']

  return (
    <div className="container mx-auto py-8 px-4">
      <Breadcrumb
        items={[
            { label: 'Главная', href: '/' },
            { label: 'Анализ качества данных', href: '/data-quality' },
          ]}
      />

      <FadeIn>
        <div className="mb-8">
          <h1 className="text-3xl font-bold mb-2">Анализ качества данных</h1>
          <p className="text-muted-foreground">
            Оценка качества данных перед нормализацией. Анализ полноты, дубликатов и несоответствий.
          </p>
        </div>

        {error && (
          <Alert variant="destructive" className="mb-6">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {!reportData && (
          <Card>
            <CardHeader>
              <CardTitle>Генерация отчета о качестве данных</CardTitle>
              <CardDescription>
                Нажмите кнопку ниже, чтобы проанализировать качество данных во всех базах данных проекта.
                Анализ займет несколько секунд.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button
                onClick={generateReport}
                disabled={loading}
                size="lg"
                className="w-full"
              >
                {loading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Анализ данных...
                  </>
                ) : (
                  <>
                    <BarChart3 className="mr-2 h-4 w-4" />
                    Проанализировать качество данных
                  </>
                )}
              </Button>
            </CardContent>
          </Card>
        )}

        {reportData && (
          <div id="report-content">
            {/* Общая оценка */}
            <Card className="mb-6">
              <CardHeader>
                <CardTitle>Общая оценка качества данных</CardTitle>
                <CardDescription>
                  Отчет сгенерирован: {new Date(reportData.metadata.generated_at).toLocaleString('ru-RU')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                  <Card>
                    <CardContent className="pt-6">
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="text-sm text-muted-foreground">Общая оценка</p>
                          <p className="text-2xl font-bold">{reportData.overall_score.score.toFixed(1)}%</p>
                        </div>
                        <Badge className={getQualityColor(reportData.overall_score.data_quality)}>
                          {getQualityLabel(reportData.overall_score.data_quality)}
                        </Badge>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pt-6">
                      <div>
                        <p className="text-sm text-muted-foreground">Полнота данных</p>
                        <p className="text-2xl font-bold">{reportData.overall_score.completeness.toFixed(1)}%</p>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pt-6">
                      <div>
                        <p className="text-sm text-muted-foreground">Уникальность</p>
                        <p className="text-2xl font-bold">{reportData.overall_score.uniqueness.toFixed(1)}%</p>
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pt-6">
                      <div>
                        <p className="text-sm text-muted-foreground">Консистентность</p>
                        <p className="text-2xl font-bold">{reportData.overall_score.consistency.toFixed(1)}%</p>
                      </div>
                    </CardContent>
                  </Card>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg">Статистика по контрагентам</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span>Всего записей:</span>
                          <span className="font-semibold">{reportData.counterparty_stats.total_records}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Полнота данных:</span>
                          <span className="font-semibold">
                            {reportData.counterparty_stats.completeness_score.toFixed(1)}%
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Потенциальные дубликаты:</span>
                          <span className="font-semibold">
                            {reportData.counterparty_stats.potential_duplicate_rate.toFixed(1)}%
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Записей с именем:</span>
                          <span className="font-semibold">{reportData.counterparty_stats.records_with_name}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Записей с ИНН:</span>
                          <span className="font-semibold">{reportData.counterparty_stats.records_with_inn}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Записей с БИН:</span>
                          <span className="font-semibold">{reportData.counterparty_stats.records_with_bin}</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg">Несоответствия</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        <div className="flex justify-between">
                          <span>Без имени:</span>
                          <span className="font-semibold text-yellow-600">
                            {reportData.counterparty_stats.records_without_name}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Без ИНН/БИН:</span>
                          <span className="font-semibold text-yellow-600">
                            {reportData.counterparty_stats.records_without_tax_id}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Некорректный ИНН:</span>
                          <span className="font-semibold text-red-600">
                            {reportData.counterparty_stats.invalid_inn_format}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Некорректный БИН:</span>
                          <span className="font-semibold text-red-600">
                            {reportData.counterparty_stats.invalid_bin_format}
                          </span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </CardContent>
            </Card>

            {/* Секция номенклатуры */}
            {reportData.nomenclature_stats.total_records > 0 && (
              <Card className="mb-6">
                <CardHeader>
                  <CardTitle>Статистика по номенклатуре</CardTitle>
                  <CardDescription>
                    Анализ качества данных товарного каталога
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                    <Card>
                      <CardHeader>
                        <CardTitle className="text-lg">Основные метрики</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2">
                          <div className="flex justify-between">
                            <span>Всего записей:</span>
                            <span className="font-semibold">{reportData.nomenclature_stats.total_records}</span>
                          </div>
                          <div className="flex justify-between">
                            <span>Полнота данных:</span>
                            <span className="font-semibold">
                              {reportData.nomenclature_stats.completeness_score.toFixed(1)}%
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span>Потенциальные дубликаты:</span>
                            <span className="font-semibold">
                              {reportData.nomenclature_stats.potential_duplicate_rate.toFixed(1)}%
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span>Записей с названием:</span>
                            <span className="font-semibold">{reportData.nomenclature_stats.records_with_name}</span>
                          </div>
                          <div className="flex justify-between">
                            <span>Записей с артикулом:</span>
                            <span className="font-semibold">{reportData.nomenclature_stats.records_with_article}</span>
                          </div>
                          <div className="flex justify-between">
                            <span>Записей с SKU:</span>
                            <span className="font-semibold">{reportData.nomenclature_stats.records_with_sku}</span>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardHeader>
                        <CardTitle className="text-lg">Заполненность атрибутов</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2">
                          <div className="flex justify-between">
                            <span>Бренд:</span>
                            <span className="font-semibold">
                              {reportData.nomenclature_stats.attribute_completeness?.brand_percent?.toFixed(1) || '0.0'}%
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span>Производитель:</span>
                            <span className="font-semibold">
                              {reportData.nomenclature_stats.attribute_completeness?.manufacturer_percent?.toFixed(1) || '0.0'}%
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span>Страна происхождения:</span>
                            <span className="font-semibold">
                              {reportData.nomenclature_stats.attribute_completeness?.country_percent?.toFixed(1) || '0.0'}%
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span>Без артикула:</span>
                            <span className="font-semibold text-yellow-600">
                              {reportData.nomenclature_stats.records_without_article}
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span>Без названия:</span>
                            <span className="font-semibold text-yellow-600">
                              {reportData.nomenclature_stats.records_without_name}
                            </span>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  {/* Графики для номенклатуры */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                    {reportData.nomenclature_stats.unit_of_measure_stats?.top_variations?.length > 0 && (
                      <Card>
                        <CardHeader>
                          <CardTitle>Топ-10 единиц измерения</CardTitle>
                        </CardHeader>
                        <CardContent>
                          <ResponsiveContainer width="100%" height={300}>
                            <DynamicBarChart
                              data={reportData.nomenclature_stats.unit_of_measure_stats.top_variations}
                            >
                              <CartesianGrid strokeDasharray="3 3" />
                              <XAxis dataKey="unit" angle={-45} textAnchor="end" height={100} />
                              <YAxis />
                              <Tooltip />
                              <Legend />
                              <DynamicBar dataKey="count" fill="#10b981" name="Количество" />
                            </DynamicBarChart>
                          </ResponsiveContainer>
                        </CardContent>
                      </Card>
                    )}

                    {reportData.nomenclature_stats.unit_of_measure_stats?.top_variations?.length > 0 && (
                      <Card>
                        <CardHeader>
                          <CardTitle>Распределение единиц измерения</CardTitle>
                        </CardHeader>
                        <CardContent>
                          <ResponsiveContainer width="100%" height={300}>
                            <DynamicPieChart>
                              <DynamicPie
                                data={reportData.nomenclature_stats.unit_of_measure_stats?.top_variations || []}
                                cx="50%"
                                cy="50%"
                                labelLine={false}
                                label={({ percent }) => `${((percent || 0) * 100).toFixed(0)}%`}
                                outerRadius={80}
                                fill="#8884d8"
                                dataKey="count"
                              >
                                {(reportData.nomenclature_stats.unit_of_measure_stats?.top_variations || []).map(
                                  (entry, index) => (
                                    <DynamicCell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                                  )
                                )}
                              </DynamicPie>
                              <Tooltip />
                            </DynamicPieChart>
                          </ResponsiveContainer>
                        </CardContent>
                      </Card>
                    )}
                  </div>

                  {/* График топ-10 брендов */}
                  {reportData.nomenclature_stats.attribute_completeness?.top_brands?.length > 0 && (
                    <Card className="mb-6">
                      <CardHeader>
                        <CardTitle>Топ-10 брендов</CardTitle>
                        <CardDescription>
                          Наиболее часто встречающиеся бренды в номенклатуре
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <ResponsiveContainer width="100%" height={300}>
                          <DynamicBarChart
                            data={reportData.nomenclature_stats.attribute_completeness.top_brands}
                          >
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="unit" angle={-45} textAnchor="end" height={100} />
                            <YAxis />
                            <Tooltip />
                            <Legend />
                            <DynamicBar dataKey="count" fill="#8b5cf6" name="Количество" />
                          </DynamicBarChart>
                        </ResponsiveContainer>
                      </CardContent>
                    </Card>
                  )}

                  {/* Несоответствия номенклатуры */}
                  {reportData.nomenclature_stats.top_inconsistencies?.length > 0 && (
                    <Card className="mb-6">
                      <CardHeader>
                        <CardTitle>Несоответствия номенклатуры</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2">
                          {(reportData.nomenclature_stats.top_inconsistencies || []).map((inc, index) => (
                            <div key={index} className="flex items-start gap-2 p-3 border rounded-lg">
                              <AlertCircle className="h-4 w-4 mt-1 text-yellow-500" />
                              <div className="flex-1">
                                <div className="font-semibold">{inc.description}</div>
                                <div className="text-sm text-muted-foreground">
                                  Количество: {inc.count} | Пример: {inc.example}
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  )}

                  {/* Неконсистентные единицы измерения */}
                  {reportData.nomenclature_stats.unit_of_measure_stats?.inconsistencies?.length > 0 && (
                    <Card className="mb-6">
                      <CardHeader>
                        <CardTitle>Неконсистентные единицы измерения</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2">
                          {(reportData.nomenclature_stats.unit_of_measure_stats?.inconsistencies || []).map(
                            (inc, index) => (
                              <div key={index} className="flex items-start gap-2 p-3 border rounded-lg">
                                <AlertCircle className="h-4 w-4 mt-1 text-orange-500" />
                                <span>{inc}</span>
                              </div>
                            )
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  )}
                </CardContent>
              </Card>
            )}

            {/* Графики */}
            {databaseChartData.length > 0 && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                <Card>
                  <CardHeader>
                    <CardTitle>Полнота данных по БД</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <ResponsiveContainer width="100%" height={300}>
                      <DynamicBarChart data={databaseChartData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                        <YAxis />
                        <Tooltip />
                        <Legend />
                        <DynamicBar dataKey="completeness" fill="#3b82f6" name="Полнота (%)" />
                      </DynamicBarChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>Дубликаты по БД</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <ResponsiveContainer width="100%" height={300}>
                      <DynamicBarChart data={databaseChartData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                        <YAxis />
                        <Tooltip />
                        <Legend />
                        <DynamicBar dataKey="duplicates" fill="#ef4444" name="Дубликаты (%)" />
                      </DynamicBarChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Таблица по БД */}
            {reportData.database_breakdown.length > 0 && (
              <Card className="mb-6">
                <CardHeader>
                  <CardTitle>Детальная статистика по базам данных</CardTitle>
                  <CardDescription>
                    Всего баз данных: {reportData.metadata.total_databases}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>База данных</TableHead>
                          <TableHead>Проект</TableHead>
                          <TableHead>Контрагенты</TableHead>
                          <TableHead>Номенклатура</TableHead>
                          <TableHead>Полнота</TableHead>
                          <TableHead>Дубликаты</TableHead>
                          <TableHead>Статус</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {reportData.database_breakdown.map((db) => (
                          <TableRow key={db.database_id}>
                            <TableCell className="font-medium">{db.database_name}</TableCell>
                            <TableCell>{db.project_name}</TableCell>
                            <TableCell>{db.counterparties}</TableCell>
                            <TableCell>{db.nomenclature}</TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                <span>{db.completeness_score.toFixed(1)}%</span>
                                <Progress value={db.completeness_score} className="w-20" />
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                <span>{db.potential_duplicate_rate.toFixed(1)}%</span>
                                <Progress value={db.potential_duplicate_rate} className="w-20" />
                              </div>
                            </TableCell>
                            <TableCell>
                              {db.status === 'success' ? (
                                <Badge className="bg-green-500">
                                  <CheckCircle2 className="h-3 w-3 mr-1" />
                                  Успешно
                                </Badge>
                              ) : (
                                <Badge variant="destructive">
                                  <AlertCircle className="h-3 w-3 mr-1" />
                                  Ошибка
                                </Badge>
                              )}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Рекомендации */}
            {reportData.recommendations.length > 0 && (
              <Card className="mb-6">
                <CardHeader>
                  <CardTitle>Рекомендации</CardTitle>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2">
                    {reportData.recommendations.map((rec, index) => (
                      <li key={index} className="flex items-start gap-2">
                        <AlertCircle className="h-4 w-4 mt-1 text-blue-500" />
                        <span>{rec}</span>
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            )}

            {/* Кнопки действий */}
            <div className="flex gap-4 mb-6">
              <Button onClick={downloadPDF} disabled={loading} variant="outline">
                <Download className="mr-2 h-4 w-4" />
                Скачать PDF-отчет
              </Button>
              <Button
                onClick={startNormalization}
                disabled={startingNormalization || reportData.database_breakdown.length === 0}
                size="lg"
                className="flex-1"
              >
                {startingNormalization ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Запуск нормализации...
                  </>
                ) : (
                  <>
                    <Play className="mr-2 h-4 w-4" />
                    Начать нормализацию
                  </>
                )}
              </Button>
            </div>
          </div>
        )}
      </FadeIn>
    </div>
  )
}

