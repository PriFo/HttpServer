'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Download, FileText, Database, TrendingUp, AlertTriangle, CheckCircle2, BarChart3, Loader2, FileSpreadsheet, FileJson, FileCode, ArrowRight, Calendar } from 'lucide-react'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { normalizePercentage } from '@/lib/locale'
import { exportToExcel, exportToPDF, exportToCSV, exportToJSON, exportToWord } from './export-utils'
import { Skeleton } from '@/components/ui/skeleton'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { useProjectState } from '@/hooks/useProjectState'

interface QualityStats {
  total_items: number
  by_level: {
    [key: string]: {
      count: number
      avg_quality: number
      percentage: number
    }
  }
  average_quality: number
  benchmark_count: number
  benchmark_percentage: number
}

interface QualityReportData {
  generated_at: string
  database: string
  quality_score: number
  summary: {
    total_records: number
    high_quality_records: number
    medium_quality_records: number
    low_quality_records: number
    unique_groups: number
    avg_confidence: number
    success_rate: number
    issues_count: number
    critical_issues: number
  }
  distribution: {
    quality_levels: Array<{
      name: string
      count: number
      percentage: number
    }>
    completed: number
    in_progress: number
    requires_review: number
    failed: number
  }
  detailed: {
    duplicates: Array<any>
    violations: Array<any>
    completeness: Array<any>
    consistency: Array<any>
    format: Array<any>
  }
  recommendations: Array<any>
}

interface QualityReportTabProps {
  database: string
  project?: string
  stats: QualityStats | null
}

export function QualityReportTab({ database, project: _project, stats }: QualityReportTabProps) {
  const hasSource = Boolean(database)

  const {
    data: reportData,
    loading,
    error,
    refetch: refetchReport,
  } = useProjectState<QualityReportData | null>(
    async (_cid, _pid, signal) => {
      if (!database) {
        return null
      }

      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), QUALITY_TIMEOUTS.STANDARD)

      try {
        const response = await fetch(`/api/quality/report?database=${encodeURIComponent(database)}`, {
          cache: 'no-store',
          signal: signal || controller.signal,
          headers: {
            'Cache-Control': 'no-cache',
          },
        })

        if (!response.ok) {
          // Для 404 возвращаем null вместо ошибки (отчёт не найден)
          if (response.status === 404) {
            return null
          }
          const payload = await response.json().catch(() => ({}))
          throw new Error(payload?.error || 'Не удалось загрузить отчёт')
        }

        return response.json()
      } finally {
        clearTimeout(timeoutId)
      }
    },
    hasSource ? `quality-report:${database}` : 'quality-report',
    hasSource ? database : 'none',
    [database],
    {
      enabled: hasSource,
      keepPreviousData: true,
    }
  )

  const handleExportPDF = () => {
    if (!reportData) return
    const dbName = database.split('/').pop()?.replace('.db', '') || 'database'
    exportToPDF(reportData, dbName)
  }

  const handleExportExcel = () => {
    if (!reportData) return
    const dbName = database.split('/').pop()?.replace('.db', '') || 'database'
    exportToExcel(reportData, dbName)
  }

  const handleExportCSV = () => {
    if (!reportData) return
    const dbName = database.split('/').pop()?.replace('.db', '') || 'database'
    exportToCSV(reportData, dbName)
  }

  const handleExportJSON = () => {
    if (!reportData) return
    const dbName = database.split('/').pop()?.replace('.db', '') || 'database'
    exportToJSON(reportData, dbName)
  }

  const handleExportWord = async () => {
    if (!reportData) return
    const dbName = database.split('/').pop()?.replace('.db', '') || 'database'
    await exportToWord(reportData, dbName)
  }

  if (!hasSource) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <Database className="h-12 w-12 text-muted-foreground mb-4 opacity-20" />
          <h3 className="text-lg font-medium mb-2">Выберите базу данных</h3>
          <p className="text-muted-foreground text-center">
            Пожалуйста, выберите базу данных для генерации отчёта качества
          </p>
        </CardContent>
      </Card>
    )
  }

  if (loading && !reportData) {
    return <QualityReportSkeleton />
  }

  if (error && !reportData) {
    return (
      <ErrorState
        title="Ошибка загрузки отчёта"
        message={error}
        action={{
          label: 'Повторить',
          onClick: () => refetchReport(),
        }}
        variant="destructive"
        className="mt-4"
      />
    )
  }

  if (!reportData) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <BarChart3 className="h-12 w-12 text-muted-foreground mb-4 opacity-20" />
          <h3 className="text-lg font-medium mb-2">Отчёт не сгенерирован</h3>
          <p className="text-muted-foreground text-center mb-4">
            Запустите анализ качества для генерации детального отчёта
          </p>
          <Button onClick={() => refetchReport()}>Сгенерировать отчет</Button>
        </CardContent>
      </Card>
    )
  }

  const dbName = database.split('/').pop()?.replace('.db', '') || 'database'

  return (
    <div className="space-y-6">
      {/* Заголовок и действия */}
      <Card>
        <CardHeader>
          <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
            <div>
              <CardTitle className="text-2xl">Отчёт оценки качества</CardTitle>
              <CardDescription className="mt-1 flex items-center gap-2">
                <Database className="h-3 w-3" /> {dbName}
                <span className="text-muted-foreground/30">|</span>
                <Calendar className="h-3 w-3" /> {new Date(reportData.generated_at).toLocaleDateString('ru-RU')}
              </CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" onClick={handleExportPDF} size="sm" className="h-9">
                <FileText className="h-4 w-4 mr-2 text-red-500" />
                PDF
              </Button>
              <Button variant="outline" onClick={handleExportExcel} size="sm" className="h-9">
                <FileSpreadsheet className="h-4 w-4 mr-2 text-green-500" />
                Excel
              </Button>
              <Button variant="outline" onClick={handleExportCSV} size="sm" className="h-9">
                <FileCode className="h-4 w-4 mr-2 text-blue-500" />
                CSV
              </Button>
              <Button variant="outline" onClick={handleExportJSON} size="sm" className="h-9">
                <FileJson className="h-4 w-4 mr-2 text-yellow-500" />
                JSON
              </Button>
              <Button variant="outline" onClick={handleExportWord} size="sm" className="h-9">
                <FileText className="h-4 w-4 mr-2 text-blue-700" />
                Word
              </Button>
            </div>
          </div>
        </CardHeader>
      </Card>

      <Tabs defaultValue="summary" className="space-y-4">
        <TabsList className="grid w-full grid-cols-2 md:w-auto md:inline-flex md:grid-cols-none">
          <TabsTrigger value="summary">Сводка</TabsTrigger>
          <TabsTrigger value="detailed">Детальный отчёт</TabsTrigger>
          <TabsTrigger value="metrics">Метрики</TabsTrigger>
          <TabsTrigger value="recommendations">Рекомендации</TabsTrigger>
        </TabsList>

        <TabsContent value="summary" className="space-y-4 focus-visible:outline-none focus-visible:ring-0">
          <SummaryView reportData={reportData} />
        </TabsContent>

        <TabsContent value="detailed" className="focus-visible:outline-none focus-visible:ring-0">
          <DetailedReportView reportData={reportData} />
        </TabsContent>

        <TabsContent value="metrics" className="focus-visible:outline-none focus-visible:ring-0">
          <MetricsView reportData={reportData} stats={stats} />
        </TabsContent>

        <TabsContent value="recommendations" className="focus-visible:outline-none focus-visible:ring-0">
          <RecommendationsView reportData={reportData} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// Компонент сводки
function SummaryView({ reportData }: { reportData: QualityReportData }) {
  const { summary, quality_score, distribution } = reportData

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {/* Основные метрики */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium flex items-center text-muted-foreground">
            <TrendingUp className="mr-2 h-4 w-4 text-primary" />
            Общее качество
          </CardTitle>
        </CardHeader>
        <CardContent>
          {(() => {
            const safeScore = isNaN(quality_score) || quality_score === null || quality_score === undefined ? 0 : quality_score
            const normalizedScore = normalizePercentage(safeScore)
            return (
              <>
                <div className="text-2xl font-bold">{normalizedScore.toFixed(1)}%</div>
                <Progress value={normalizedScore} className="h-2 mt-2" />
              </>
            )
          })()}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">Записей обработано</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{summary.total_records.toLocaleString('ru-RU')}</div>
          <div className="text-xs text-muted-foreground mt-1">
            {summary.high_quality_records.toLocaleString('ru-RU')} высокого качества
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium flex items-center text-muted-foreground">
            <AlertTriangle className="mr-2 h-4 w-4 text-yellow-600" />
            Проблемы
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-yellow-600">{summary.issues_count}</div>
          <div className="text-xs text-muted-foreground mt-1">
            {summary.critical_issues} критических
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium flex items-center text-muted-foreground">
            <CheckCircle2 className="mr-2 h-4 w-4 text-green-600" />
            Успешно
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-green-600">{(isNaN(summary.success_rate) ? 0 : summary.success_rate).toFixed(1)}%</div>
          <div className="text-xs text-muted-foreground mt-1">
            Успешных операций
          </div>
        </CardContent>
      </Card>

      {/* Распределение качества */}
      <Card className="md:col-span-2">
        <CardHeader>
          <CardTitle>Распределение по уровням качества</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {distribution.quality_levels.map((level, index) => (
              <div key={`quality-level-${level.name}-${index}`} className="space-y-1">
                <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center space-x-2">
                        <Badge variant={
                            level.name === 'Высокое' ? 'default' : 
                            level.name === 'Среднее' ? 'secondary' : 'destructive'
                        } className="w-20 justify-center">
                            {level.name}
                        </Badge>
                        <span className="text-muted-foreground">
                            {level.count.toLocaleString('ru-RU')} зап.
                        </span>
                    </div>
                    <span className="font-medium">{level.percentage.toFixed(1)}%</span>
                </div>
                <Progress 
                    value={level.percentage} 
                    className="h-1.5"
                    // Customizing progress color based on level could be done via CSS variables or conditional classes if supported
                 />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Статус обработки */}
      <Card className="md:col-span-2">
        <CardHeader>
          <CardTitle>Статус обработки</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            <div className="text-center p-2 bg-green-50 dark:bg-green-900/10 rounded-lg">
              <div className="text-2xl font-bold text-green-600 dark:text-green-400">
                {distribution.completed.toLocaleString('ru-RU')}
              </div>
              <div className="text-xs font-medium text-green-800 dark:text-green-300 mt-1">Завершено</div>
            </div>
            <div className="text-center p-2 bg-blue-50 dark:bg-blue-900/10 rounded-lg">
              <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                {distribution.in_progress.toLocaleString('ru-RU')}
              </div>
              <div className="text-xs font-medium text-blue-800 dark:text-blue-300 mt-1">В процессе</div>
            </div>
            <div className="text-center p-2 bg-yellow-50 dark:bg-yellow-900/10 rounded-lg">
              <div className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
                {distribution.requires_review.toLocaleString('ru-RU')}
              </div>
              <div className="text-xs font-medium text-yellow-800 dark:text-yellow-300 mt-1">Требует проверки</div>
            </div>
            <div className="text-center p-2 bg-red-50 dark:bg-red-900/10 rounded-lg">
              <div className="text-2xl font-bold text-red-600 dark:text-red-400">
                {distribution.failed.toLocaleString('ru-RU')}
              </div>
              <div className="text-xs font-medium text-red-800 dark:text-red-300 mt-1">Ошибки</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// Компонент детального отчёта
function DetailedReportView({ reportData }: { reportData: QualityReportData }) {
  const { detailed } = reportData

  return (
    <Card>
      <CardHeader>
        <CardTitle>Детальный анализ качества</CardTitle>
        <CardDescription>
          Подробная информация по всем аспектам качества данных
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="duplicates" className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="duplicates">Дубликаты</TabsTrigger>
            <TabsTrigger value="violations">Нарушения</TabsTrigger>
            <TabsTrigger value="completeness">Полнота</TabsTrigger>
            <TabsTrigger value="consistency">Согласованность</TabsTrigger>
          </TabsList>

          <TabsContent value="duplicates" className="mt-6">
            {detailed.duplicates && detailed.duplicates.length > 0 ? (
              <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                    <TableRow>
                        <TableHead>Группа дубликатов</TableHead>
                        <TableHead>Тип</TableHead>
                        <TableHead>Количество</TableHead>
                        <TableHead>Схожесть</TableHead>
                        <TableHead>Уверенность</TableHead>
                        <TableHead>Статус</TableHead>
                    </TableRow>
                    </TableHeader>
                    <TableBody>
                    {detailed.duplicates.slice(0, 10).map((duplicate: any, index: number) => (
                        <TableRow key={duplicate.id ? `duplicate-${duplicate.id}-${index}` : `duplicate-${duplicate.group_id}-${index}`}>
                        <TableCell className="font-medium">
                            {duplicate.group_name || duplicate.normalized_name || 'Группа ' + (duplicate.id || duplicate.group_id)}
                        </TableCell>
                        <TableCell>
                            <Badge variant="outline">
                            {duplicate.duplicate_type_name || duplicate.duplicate_type || 'Неизвестно'}
                            </Badge>
                        </TableCell>
                        <TableCell>
                            <Badge variant="secondary">{duplicate.count || duplicate.item_count || 0} зап.</Badge>
                        </TableCell>
                        <TableCell>
                            <div className="flex items-center space-x-2">
                            <Progress value={normalizePercentage(isNaN(duplicate.similarity_score) ? 0 : (duplicate.similarity_score || 0))} className="h-1.5 w-16" />
                            <span className="text-xs text-muted-foreground">{normalizePercentage(isNaN(duplicate.similarity_score) ? 0 : (duplicate.similarity_score || 0)).toFixed(0)}%</span>
                            </div>
                        </TableCell>
                        <TableCell>
                            <div className="flex items-center space-x-2">
                            <Progress value={normalizePercentage(isNaN(duplicate.confidence) ? 0 : (duplicate.confidence || 0))} className="h-1.5 w-16" />
                            <span className="text-xs text-muted-foreground">{normalizePercentage(isNaN(duplicate.confidence) ? 0 : (duplicate.confidence || 0)).toFixed(0)}%</span>
                            </div>
                        </TableCell>
                        <TableCell>
                            <Badge variant={
                            duplicate.status === 'resolved' || duplicate.merged ? 'default' : 
                            duplicate.status === 'in_review' ? 'secondary' : 'destructive'
                            }>
                            {duplicate.status === 'resolved' || duplicate.merged ? 'Объединено' : 
                            duplicate.status === 'in_review' ? 'На проверке' : 'Требует проверки'}
                            </Badge>
                        </TableCell>
                        </TableRow>
                    ))}
                    </TableBody>
                </Table>
              </div>
            ) : (
                <EmptyState 
                    title="Дубликатов не найдено" 
                    description="В отчете нет информации о дубликатах"
                    icon={CheckCircle2}
                />
            )}
          </TabsContent>

          <TabsContent value="violations" className="mt-6">
            {detailed.violations && detailed.violations.length > 0 ? (
              <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                    <TableRow>
                        <TableHead>Тип нарушения</TableHead>
                        <TableHead>Описание</TableHead>
                        <TableHead>Количество</TableHead>
                        <TableHead>Серьёзность</TableHead>
                    </TableRow>
                    </TableHeader>
                    <TableBody>
                    {detailed.violations.slice(0, 10).map((violation: any, index: number) => (
                        <TableRow key={violation.id || `violation-${index}`}>
                        <TableCell className="font-medium">{violation.type || violation.rule_name || 'Неизвестно'}</TableCell>
                        <TableCell>{violation.description || violation.message || ''}</TableCell>
                        <TableCell>
                            <Badge variant="outline">{violation.count || 1}</Badge>
                        </TableCell>
                        <TableCell>
                            <Badge variant={
                            violation.severity === 'high' || violation.severity === 'critical' ? 'destructive' : 
                            violation.severity === 'medium' ? 'secondary' : 'default'
                            }>
                            {violation.severity === 'critical' ? 'Критический' :
                            violation.severity === 'high' ? 'Высокий' :
                            violation.severity === 'medium' ? 'Средний' : 'Низкий'}
                            </Badge>
                        </TableCell>
                        </TableRow>
                    ))}
                    </TableBody>
                </Table>
              </div>
            ) : (
               <EmptyState 
                    title="Нарушений не найдено" 
                    description="В отчете нет информации о нарушениях"
                    icon={CheckCircle2}
                />
            )}
          </TabsContent>

          <TabsContent value="completeness" className="mt-6">
            {detailed.completeness && detailed.completeness.length > 0 ? (
              <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                    <TableRow>
                        <TableHead>Тип</TableHead>
                        <TableHead>Поле</TableHead>
                        <TableHead>Текущее значение</TableHead>
                        <TableHead>Предлагаемое значение</TableHead>
                        <TableHead>Приоритет</TableHead>
                    </TableRow>
                    </TableHeader>
                    <TableBody>
                    {detailed.completeness.slice(0, 10).map((item: any, index: number) => (
                        <TableRow key={item.id ? `completeness-${item.id}-${index}` : `completeness-${index}`}>
                        <TableCell className="font-medium">{item.type || 'Неизвестно'}</TableCell>
                        <TableCell>{item.field || item.field_name || ''}</TableCell>
                        <TableCell className="font-mono text-xs">{item.current_value || ''}</TableCell>
                        <TableCell className="font-mono text-xs">{item.suggested_value || ''}</TableCell>
                        <TableCell>
                            <Badge variant={
                            item.priority === 'high' ? 'destructive' : 
                            item.priority === 'medium' ? 'secondary' : 'default'
                            }>
                            {item.priority === 'high' ? 'Высокий' :
                            item.priority === 'medium' ? 'Средний' : 'Низкий'}
                            </Badge>
                        </TableCell>
                        </TableRow>
                    ))}
                    </TableBody>
                </Table>
              </div>
            ) : (
                <EmptyState 
                    title="Предложений не найдено" 
                    description="В отчете нет информации о полноте данных"
                    icon={CheckCircle2}
                />
            )}
          </TabsContent>

          <TabsContent value="consistency" className="mt-6">
            {detailed.consistency && detailed.consistency.length > 0 ? (
              <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                    <TableRow>
                        <TableHead>Тип несоответствия</TableHead>
                        <TableHead>Описание</TableHead>
                        <TableHead>Количество</TableHead>
                    </TableRow>
                    </TableHeader>
                    <TableBody>
                    {detailed.consistency.slice(0, 10).map((item: any, index: number) => (
                        <TableRow key={item.id ? `consistency-${item.id}-${index}` : `consistency-${index}`}>
                        <TableCell className="font-medium">{item.type || 'Неизвестно'}</TableCell>
                        <TableCell>{item.description || ''}</TableCell>
                        <TableCell>
                            <Badge variant="outline">{item.count || 1}</Badge>
                        </TableCell>
                        </TableRow>
                    ))}
                    </TableBody>
                </Table>
              </div>
            ) : (
                <EmptyState 
                    title="Несоответствий не найдено" 
                    description="В отчете нет информации о несоответствиях"
                    icon={CheckCircle2}
                />
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

// Компонент метрик
function MetricsView({ reportData, stats }: { reportData: QualityReportData, stats: any }) {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Дополнительные метрики</CardTitle>
          <CardDescription>
            Детальная статистика по качеству данных
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center p-4 border rounded-lg bg-card hover:bg-accent/5 transition-colors">
              <div className="text-3xl font-bold text-primary">{reportData.summary.unique_groups}</div>
              <div className="text-sm text-muted-foreground mt-1">Уникальных групп</div>
            </div>
            <div className="text-center p-4 border rounded-lg bg-card hover:bg-accent/5 transition-colors">
              <div className="text-3xl font-bold text-primary">{normalizePercentage(isNaN(reportData.summary.avg_confidence) ? 0 : reportData.summary.avg_confidence).toFixed(1)}%</div>
              <div className="text-sm text-muted-foreground mt-1">Средняя уверенность</div>
            </div>
            <div className="text-center p-4 border rounded-lg bg-card hover:bg-accent/5 transition-colors">
              <div className="text-3xl font-bold text-green-600">{reportData.summary.high_quality_records}</div>
              <div className="text-sm text-muted-foreground mt-1">Высокое качество</div>
            </div>
            <div className="text-center p-4 border rounded-lg bg-card hover:bg-accent/5 transition-colors">
              <div className="text-3xl font-bold text-yellow-600">{reportData.summary.medium_quality_records}</div>
              <div className="text-sm text-muted-foreground mt-1">Среднее качество</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// Компонент рекомендаций
function RecommendationsView({ reportData }: { reportData: QualityReportData }) {
  const { recommendations } = reportData

  return (
    <Card>
      <CardHeader>
        <CardTitle>Рекомендации по улучшению</CardTitle>
        <CardDescription>
          Предложения по повышению качества данных
        </CardDescription>
      </CardHeader>
      <CardContent>
        {recommendations && recommendations.length > 0 ? (
          <div className="grid gap-4">
            {recommendations.map((rec: any, index: number) => (
              <div key={index} className="p-4 border rounded-lg hover:shadow-sm transition-shadow">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-medium flex items-center gap-2">
                     <div className={`w-2 h-2 rounded-full ${rec.priority === 'high' ? 'bg-red-500' : 'bg-blue-500'}`} />
                     {rec.title || rec.type || `Рекомендация ${index + 1}`}
                  </h4>
                  <Badge variant={rec.priority === 'high' ? 'destructive' : 'secondary'}>
                    {rec.priority === 'high' ? 'Высокий' : 'Средний'}
                  </Badge>
                </div>
                <p className="text-sm text-muted-foreground mb-3">{rec.description || rec.message || ''}</p>
                {rec.action && (
                  <div className="text-sm bg-muted/50 p-2 rounded flex items-start gap-2">
                    <ArrowRight className="w-4 h-4 mt-0.5 text-primary" />
                    <span className="font-medium">Действие:</span> 
                    <span>{rec.action}</span>
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
            <EmptyState 
                title="Рекомендации отсутствуют" 
                description="Система не нашла явных рекомендаций по улучшению качества данных."
                icon={CheckCircle2}
            />
        )}
      </CardContent>
    </Card>
  )
}

function QualityReportSkeleton() {
    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <Skeleton className="h-8 w-1/3 mb-2" />
                    <Skeleton className="h-4 w-1/4" />
                </CardHeader>
            </Card>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                {[...Array(4)].map((_, i) => (
                    <Card key={i}>
                        <CardContent className="pt-6">
                            <Skeleton className="h-4 w-1/2 mb-2" />
                            <Skeleton className="h-8 w-1/3" />
                        </CardContent>
                    </Card>
                ))}
            </div>
            <Card>
                <CardContent className="h-[300px] flex items-center justify-center">
                    <Skeleton className="h-full w-full" />
                </CardContent>
            </Card>
        </div>
    )
}
