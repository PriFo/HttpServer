'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { AlertCircle, AlertTriangle, Info, XCircle, CheckCircle, Loader2, Calendar } from 'lucide-react'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { Pagination } from '@/components/ui/pagination'
import { FilterBar, type FilterConfig } from '@/components/common/filter-bar'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchJson, getErrorMessage } from '@/lib/fetch-utils'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { useProjectState } from '@/hooks/useProjectState'

interface Violation {
  id: number
  normalized_item_id: number
  rule_name: string
  category: string
  severity: string
  message: string
  recommendation: string
  field_name: string
  current_value: string
  resolved: boolean
  resolved_at: string | null
  resolved_by: string | null
  created_at: string
}

interface ViolationsResponse {
  violations: Violation[]
  total: number
  limit: number
  offset: number
}

const severityConfig = {
  critical: {
    label: 'Критический',
    icon: XCircle,
    color: 'bg-red-500 hover:bg-red-600 text-white',
    borderColor: 'border-red-500',
    textColor: 'text-red-600',
  },
  error: {
    label: 'Ошибка',
    icon: AlertCircle,
    color: 'bg-orange-500 hover:bg-orange-600 text-white',
    borderColor: 'border-orange-500',
    textColor: 'text-orange-600',
  },
  warning: {
    label: 'Предупреждение',
    icon: AlertTriangle,
    color: 'bg-yellow-500 hover:bg-yellow-600 text-white',
    borderColor: 'border-yellow-500',
    textColor: 'text-yellow-600',
  },
  info: {
    label: 'Информация',
    icon: Info,
    color: 'bg-blue-500 hover:bg-blue-600 text-white',
    borderColor: 'border-blue-500',
    textColor: 'text-blue-600',
  }
}

const categoryConfig: Record<string, string> = {
  completeness: 'Полнота данных',
  accuracy: 'Точность',
  consistency: 'Согласованность',
  format: 'Формат'
}

interface AnalysisStatus {
  is_running: boolean
  progress: number
  current_step: string
  violations_found: number
}

export function QualityViolationsTab({ database, project }: { database: string; project?: string }) {
  const [actionError, setActionError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [resolvingId, setResolvingId] = useState<number | null>(null)
  const [analysisStatus, setAnalysisStatus] = useState<AnalysisStatus | null>(null)
  const itemsPerPage = 20

  const [filters, setFilters] = useState({
    severity: 'all',
    category: 'all',
    showResolved: false,
    search: '',
  })

  const filterConfigs: FilterConfig[] = [
    {
      type: 'search',
      key: 'search',
      label: 'Поиск',
      placeholder: 'Поиск по правилу...',
    },
    {
      type: 'select',
      key: 'severity',
      label: 'Серьезность',
      options: [
        { value: 'all', label: 'Все' },
        { value: 'critical', label: 'Критический' },
        { value: 'error', label: 'Ошибка' },
        { value: 'warning', label: 'Предупреждение' },
        { value: 'info', label: 'Информация' },
      ],
    },
    {
      type: 'select',
      key: 'category',
      label: 'Категория',
      options: [
        { value: 'all', label: 'Все' },
        { value: 'completeness', label: 'Полнота данных' },
        { value: 'accuracy', label: 'Точность' },
        { value: 'consistency', label: 'Согласованность' },
        { value: 'format', label: 'Формат' },
      ],
    },
    {
      type: 'checkbox',
      key: 'showResolved',
      label: 'Показать решенные',
    },
  ]

  const clientKey = database ? `quality-db:${database}` : project ? `quality-project:${project}` : 'quality'
  const projectKey = JSON.stringify({
    page: currentPage,
    severity: filters.severity,
    category: filters.category,
    showResolved: filters.showResolved,
    search: filters.search,
  })

  const {
    data: violationsData,
    loading,
    error,
    refetch: refetchViolations,
  } = useProjectState<ViolationsResponse>(
    async (_cid, _pid, signal) => {
      if (!database && !project) {
        return { violations: [], total: 0, limit: itemsPerPage, offset: 0 }
      }

      const params = new URLSearchParams({
        limit: itemsPerPage.toString(),
        offset: ((currentPage - 1) * itemsPerPage).toString(),
      })

      if (database) params.append('database', database)
      if (project) params.append('project', project)
      if (filters.severity !== 'all') params.append('severity', filters.severity)
      if (filters.category !== 'all') params.append('category', filters.category)
      params.append('show_resolved', filters.showResolved ? 'true' : 'false')
      if (filters.search.trim()) params.append('search', filters.search.trim())

      const response = await fetch(`/api/quality/violations?${params.toString()}`, {
        cache: 'no-store',
        signal,
        headers: { 'Cache-Control': 'no-cache' },
      })

      if (!response.ok) {
        const payload = await response.json().catch(() => ({}))
        throw new Error(payload?.error || 'Не удалось загрузить нарушения')
      }

      return response.json()
    },
    clientKey,
    projectKey,
    [
      database,
      project,
      filters.severity,
      filters.category,
      filters.showResolved,
      filters.search,
      currentPage,
    ],
    {
      enabled: Boolean(database || project),
      keepPreviousData: true,
    }
  )

  const violations = violationsData?.violations || []
  const total = violationsData?.total || 0
  const combinedError = error || actionError
  const isInitialLoading = loading && !violationsData
  const hasSource = Boolean(database || project)

  useEffect(() => {
    setActionError(null)
  }, [
    database,
    project,
    filters.severity,
    filters.category,
    filters.showResolved,
    filters.search,
    currentPage,
  ])

  // Fetch analysis status
  const fetchAnalysisStatus = useCallback(async () => {
    try {
      const data = await fetchJson<AnalysisStatus>(
        '/api/quality/analyze/status',
        {
          timeout: QUALITY_TIMEOUTS.FAST,
          cache: 'no-store',
        }
      )
      setAnalysisStatus(data)
    } catch (err) {
      // Ignore errors, status is optional
    }
  }, [])

  useEffect(() => {
    if (hasSource) {
      fetchAnalysisStatus()
    }
  }, [hasSource, fetchAnalysisStatus])

  // Auto-refresh during analysis
  useEffect(() => {
    if (!hasSource) return
    
    if (analysisStatus?.is_running && (analysisStatus.current_step === 'violations' || analysisStatus.current_step === 'suggestions')) {
      const interval = setInterval(() => {
        refetchViolations()
        fetchAnalysisStatus()
      }, 2000) // Refresh every 2 seconds during analysis
      return () => clearInterval(interval)
    } else if (analysisStatus?.is_running) {
      // Refresh less frequently during other steps
      const interval = setInterval(() => {
        fetchAnalysisStatus()
      }, 5000)
      return () => clearInterval(interval)
    }
  }, [hasSource, analysisStatus?.is_running, analysisStatus?.current_step, refetchViolations, fetchAnalysisStatus])

  const handleResolveViolation = async (violationId: number) => {
    setResolvingId(violationId)

    try {
      await fetchJson(
        `/api/quality/violations/${violationId}/resolve`,
        {
          method: 'POST',
          timeout: QUALITY_TIMEOUTS.LONG,
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            resolved_by: 'User'
          })
        }
      )

      await refetchViolations()
    } catch (err) {
      const errorMessage = getErrorMessage(err, 'Не удалось решить нарушение')
      setActionError(errorMessage)
    } finally {
      setResolvingId(null)
    }
  }

  const getSeverityBadge = (severity: string) => {
    const config = severityConfig[severity as keyof typeof severityConfig]
    if (!config) return <Badge variant="outline">{severity}</Badge>

    const Icon = config.icon

    return (
      <Badge className={`${config.color} border-transparent shadow-sm`}>
        <Icon className="w-3 h-3 mr-1" />
        {config.label}
      </Badge>
    )
  }

  const getCategoryBadge = (category: string) => {
    const label = categoryConfig[category] || category
    return (
      <Badge variant="secondary" className="bg-muted text-muted-foreground hover:bg-muted/80">
        {label}
      </Badge>
    )
  }

  const totalPages = itemsPerPage > 0 ? Math.ceil(Math.max(0, total) / itemsPerPage) : 1

  if (!hasSource) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="Выберите источник данных"
        description="Для просмотра нарушений выберите базу данных или проект."
      />
    )
  }

  return (
    <div className="space-y-6">
      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle>Фильтры</CardTitle>
        </CardHeader>
        <CardContent>
          <FilterBar
            filters={filterConfigs}
            values={filters}
            onChange={(newFilters) => {
              setFilters(newFilters as { severity: string; category: string; showResolved: boolean; search: string })
              setCurrentPage(1)
            }}
            onReset={() => {
              setFilters({ severity: 'all', category: 'all', showResolved: false, search: '' })
              setCurrentPage(1)
            }}
          />
        </CardContent>
      </Card>

      {/* Summary */}
      {!isInitialLoading && (
        <div className="flex items-center justify-between px-1">
            <div className="flex items-center gap-2">
              <p className="text-sm text-muted-foreground">
                Найдено нарушений: <span className="font-medium text-foreground">{total}</span>
              </p>
              {analysisStatus?.is_running && analysisStatus.current_step === 'violations' && (
                <Badge variant="outline" className="text-xs">
                  <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                  Анализ выполняется...
                </Badge>
              )}
              {loading && !isInitialLoading && (
                <Badge variant="outline" className="text-xs">
                  <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                  Обновление данных...
                </Badge>
              )}
            </div>
            {totalPages > 1 && (
                <p className="text-sm text-muted-foreground">
                Страница {currentPage} из {totalPages}
                </p>
            )}
        </div>
      )}

      {/* Error Alert */}
      {combinedError && !isInitialLoading && (
        <ErrorState
          title="Ошибка загрузки нарушений"
          message={combinedError}
          action={{
            label: 'Повторить',
            onClick: () => refetchViolations(),
          }}
          variant="destructive"
          className="mt-4"
        />
      )}

      {/* Violations List */}
      {isInitialLoading ? (
        <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
                <Card key={i}>
                    <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
                        <div className="space-y-2">
                            <Skeleton className="h-5 w-[200px]" />
                            <Skeleton className="h-4 w-[300px]" />
                        </div>
                        <Skeleton className="h-8 w-[100px]" />
                    </CardHeader>
                    <CardContent>
                        <Skeleton className="h-16 w-full" />
                    </CardContent>
                </Card>
            ))}
        </div>
      ) : violations.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            {analysisStatus?.is_running && (analysisStatus.current_step === 'violations' || analysisStatus.current_step === 'suggestions') ? (
              <div className="flex flex-col items-center justify-center py-8 space-y-4">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
                <div className="text-center space-y-2">
                  <p className="font-medium">Выполняется анализ нарушений</p>
                  <p className="text-sm text-muted-foreground">
                    Прогресс: {(isNaN(analysisStatus.progress) ? 0 : analysisStatus.progress).toFixed(1)}%
                  </p>
                  {analysisStatus.violations_found > 0 && (
                    <p className="text-sm text-muted-foreground">
                      Найдено нарушений: {analysisStatus.violations_found}
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <EmptyState
                icon={CheckCircle}
                title="Нарушений не найдено"
                description={
                  filters.showResolved
                    ? 'В базе данных нет нарушений качества. Все данные соответствуют правилам качества.'
                    : 'Все нарушения были решены или не найдены по заданным фильтрам. Если анализ еще не выполнялся, запустите анализ качества для выявления нарушений.'
                }
              />
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {violations.map((violation) => {
            const severityConf = severityConfig[violation.severity as keyof typeof severityConfig]

            return (
              <Card
                key={violation.id}
                className={`transition-all hover:shadow-md ${severityConf?.borderColor ? `border-l-4 ${severityConf.borderColor}` : ''} ${
                  violation.resolved ? 'opacity-70 bg-muted/30' : ''
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
                    <div className="space-y-1.5 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        {getSeverityBadge(violation.severity)}
                        {getCategoryBadge(violation.category)}
                        {violation.resolved && (
                          <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                            <CheckCircle className="w-3 h-3 mr-1" />
                            Решено
                          </Badge>
                        )}
                        <span className="text-xs text-muted-foreground ml-auto md:ml-0 flex items-center gap-1">
                            <Calendar className="w-3 h-3" />
                            {new Date(violation.created_at).toLocaleDateString('ru-RU')}
                        </span>
                      </div>
                      <CardTitle className="text-lg font-semibold leading-tight">
                        {violation.rule_name}
                      </CardTitle>
                      <CardDescription className="flex items-center gap-2 text-xs font-mono">
                        ID: {violation.normalized_item_id}
                        {violation.field_name && (
                            <>
                                <span className="text-muted-foreground/50">•</span>
                                <span>Поле: {violation.field_name}</span>
                            </>
                        )}
                      </CardDescription>
                    </div>
                    {!violation.resolved && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleResolveViolation(violation.id)}
                        disabled={resolvingId === violation.id}
                        className="shrink-0"
                      >
                        {resolvingId === violation.id ? (
                            <>
                                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                                Решение...
                            </>
                        ) : (
                            <>
                                <CheckCircle className="mr-2 h-3 w-3" />
                                Решить
                            </>
                        )}
                      </Button>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4 pt-0">
                  <div className="grid md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                        <div>
                            <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-1">Проблема</h4>
                            <p className="text-sm">{violation.message}</p>
                        </div>
                        {violation.current_value && (
                            <div>
                                <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-1">Текущее значение</h4>
                                <code className="text-sm bg-muted px-2 py-1 rounded border inline-block max-w-full overflow-hidden text-ellipsis">
                                    {violation.current_value}
                                </code>
                            </div>
                        )}
                    </div>

                    {violation.recommendation && (
                        <div className="bg-blue-50/50 dark:bg-blue-900/10 border border-blue-100 dark:border-blue-800 rounded-lg p-3 h-fit">
                            <h4 className="text-xs font-medium uppercase tracking-wider text-blue-700 dark:text-blue-400 mb-1 flex items-center gap-1">
                                <Info className="w-3 h-3" />
                                Рекомендация
                            </h4>
                            <p className="text-sm text-blue-900 dark:text-blue-300">{violation.recommendation}</p>
                        </div>
                    )}
                  </div>

                  {violation.resolved && (
                    <div className="text-xs text-muted-foreground pt-3 border-t mt-2 flex items-center gap-2">
                      <CheckCircle className="w-3 h-3 text-green-600" />
                      Решено пользователем {violation.resolved_by || 'System'}
                      {violation.resolved_at && ` • ${new Date(violation.resolved_at).toLocaleString('ru-RU')}`}
                    </div>
                  )}
                </CardContent>
              </Card>
            )
          })}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="py-4 flex justify-center">
                <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPageChange={setCurrentPage}
                itemsPerPage={itemsPerPage}
                totalItems={total}
                />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
