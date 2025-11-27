'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ErrorState } from '@/components/common/error-state'
import { AlertCircle, CheckCircle, Lightbulb, Zap, ArrowRight, TrendingUp, Settings, RefreshCw, GitMerge, Eye, Loader2, Calendar } from 'lucide-react'
import { EmptyState } from '@/components/common/empty-state'
import { Pagination } from '@/components/ui/pagination'
import { FilterBar, type FilterConfig } from '@/components/common/filter-bar'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { fetchJson, getErrorMessage } from '@/lib/fetch-utils'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { useProjectState } from '@/hooks/useProjectState'

interface Suggestion {
  id: number
  normalized_item_id: number
  type: string
  priority: string
  field: string
  current_value: string
  suggested_value: string
  confidence: number
  reasoning: string
  auto_applyable: boolean
  applied: boolean
  applied_at: string | null
  created_at: string
}

interface SuggestionsResponse {
  suggestions: Suggestion[]
  total: number
  limit: number
  offset: number
}

const typeConfig: Record<string, { label: string; icon: any; color: string; description: string }> = {
  set_value: {
    label: 'Установить значение',
    icon: Settings,
    color: 'bg-blue-500 text-white',
    description: 'Установить новое значение поля'
  },
  correct_format: {
    label: 'Исправить формат',
    icon: RefreshCw,
    color: 'bg-purple-500 text-white',
    description: 'Исправить формат данных'
  },
  reprocess: {
    label: 'Повторная обработка',
    icon: RefreshCw,
    color: 'bg-orange-500 text-white',
    description: 'Повторно обработать запись'
  },
  merge: {
    label: 'Объединить',
    icon: GitMerge,
    color: 'bg-green-500 text-white',
    description: 'Объединить с другой записью'
  },
  review: {
    label: 'Требует проверки',
    icon: Eye,
    color: 'bg-yellow-500 text-white',
    description: 'Требуется ручная проверка'
  }
}

const priorityConfig = {
  critical: {
    label: 'Критический',
    color: 'bg-red-500 hover:bg-red-600 text-white',
  },
  high: {
    label: 'Высокий',
    color: 'bg-orange-500 hover:bg-orange-600 text-white',
  },
  medium: {
    label: 'Средний',
    color: 'bg-yellow-500 hover:bg-yellow-600 text-white',
  },
  low: {
    label: 'Низкий',
    color: 'bg-blue-500 hover:bg-blue-600 text-white',
  }
}

interface AnalysisStatus {
  is_running: boolean
  progress: number
  current_step: string
  suggestions_found: number
}

export function QualitySuggestionsTab({ database, project }: { database: string; project?: string }) {
  const [actionError, setActionError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [applyingId, setApplyingId] = useState<number | null>(null)
  const [analysisStatus, setAnalysisStatus] = useState<AnalysisStatus | null>(null)
  const itemsPerPage = 20

  const [filters, setFilters] = useState({
    priority: 'all',
    type: 'all',
    showApplied: false,
    autoApplyableOnly: false,
    search: '',
  })

  const filterConfigs: FilterConfig[] = [
    {
      type: 'search',
      key: 'search',
      label: 'Поиск',
      placeholder: 'Поиск по полю или значению...',
    },
    {
      type: 'select',
      key: 'priority',
      label: 'Приоритет',
      options: [
        { value: 'all', label: 'Все' },
        { value: 'critical', label: 'Критический' },
        { value: 'high', label: 'Высокий' },
        { value: 'medium', label: 'Средний' },
        { value: 'low', label: 'Низкий' },
      ],
    },
    {
      type: 'select',
      key: 'type',
      label: 'Тип',
      options: [
        { value: 'all', label: 'Все' },
        { value: 'set_value', label: 'Установить значение' },
        { value: 'correct_format', label: 'Исправить формат' },
        { value: 'reprocess', label: 'Повторная обработка' },
        { value: 'merge', label: 'Объединить' },
        { value: 'review', label: 'Требует проверки' },
      ],
    },
    {
      type: 'checkbox',
      key: 'showApplied',
      label: 'Показать примененные',
    },
    {
      type: 'checkbox',
      key: 'autoApplyableOnly',
      label: 'Только автоприменяемые',
    },
  ]

  const hasSource = Boolean(database || project)
  const clientKey = database ? `quality-db:${database}` : project ? `quality-project:${project}` : 'quality'
  const filtersKey = JSON.stringify({
    page: currentPage,
    priority: filters.priority,
    type: filters.type,
    showApplied: filters.showApplied,
    autoApplyableOnly: filters.autoApplyableOnly,
    search: filters.search,
  })

  const {
    data: suggestionsData,
    loading,
    error,
    refetch: refetchSuggestions,
  } = useProjectState<SuggestionsResponse>(
    async (_cid, _pid, signal) => {
      if (!hasSource) {
        return { suggestions: [], total: 0, limit: itemsPerPage, offset: 0 }
      }

      const params = new URLSearchParams({
        limit: itemsPerPage.toString(),
        offset: ((currentPage - 1) * itemsPerPage).toString(),
      })

      if (database) params.append('database', database)
      if (project) params.append('project', project)
      if (filters.priority !== 'all') params.append('priority', filters.priority)
      if (!filters.showApplied) params.append('applied', 'false')
      if (filters.autoApplyableOnly) params.append('auto_applyable', 'true')
      if (filters.type !== 'all') params.append('type', filters.type)
      if (filters.search.trim()) params.append('search', filters.search.trim())

      const response = await fetch(`/api/quality/suggestions?${params.toString()}`, {
        cache: 'no-store',
        signal,
        headers: { 'Cache-Control': 'no-cache' },
      })

      if (!response.ok) {
        const payload = await response.json().catch(() => ({}))
        throw new Error(payload?.error || 'Не удалось загрузить предложения')
      }

      return response.json()
    },
    clientKey,
    filtersKey,
    [
      database,
      project,
      filters.priority,
      filters.type,
      filters.showApplied,
      filters.autoApplyableOnly,
      filters.search,
      currentPage,
    ],
    {
      enabled: hasSource,
      keepPreviousData: true,
    }
  )

  const suggestions = suggestionsData?.suggestions || []
  const total = suggestionsData?.total || 0
  const combinedError = error || actionError
  const isInitialLoading = loading && !suggestionsData
  useEffect(() => {
    setActionError(null)
  }, [
    database,
    project,
    filters.priority,
    filters.type,
    filters.showApplied,
    filters.autoApplyableOnly,
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
    
    if (analysisStatus?.is_running && (analysisStatus.current_step === 'suggestions' || analysisStatus.current_step === 'violations')) {
      const interval = setInterval(() => {
        refetchSuggestions()
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
  }, [hasSource, analysisStatus?.is_running, analysisStatus?.current_step, refetchSuggestions, fetchAnalysisStatus])

  const handleApplySuggestion = async (suggestionId: number) => {
    setApplyingId(suggestionId)

    try {
      await fetchJson(
        `/api/quality/suggestions/${suggestionId}/apply`,
        {
          method: 'POST',
          timeout: QUALITY_TIMEOUTS.LONG,
          headers: {
            'Content-Type': 'application/json'
          }
        }
      )

      await refetchSuggestions()
    } catch (err) {
      const errorMessage = getErrorMessage(err, 'Не удалось применить предложение')
      setActionError(errorMessage)
    } finally {
      setApplyingId(null)
    }
  }

  const getTypeBadge = (type: string) => {
    const config = typeConfig[type as keyof typeof typeConfig]
    if (!config) return <Badge variant="secondary">{type}</Badge>

    const Icon = config.icon

    return (
      <Badge className={config.color}>
        <Icon className="w-3 h-3 mr-1" />
        {config.label}
      </Badge>
    )
  }

  const getPriorityBadge = (priority: string) => {
    const config = priorityConfig[priority as keyof typeof priorityConfig]
    if (!config) return <Badge variant="outline">{priority}</Badge>

    return (
      <Badge className={`${config.color} border-transparent`}>
        {config.label}
      </Badge>
    )
  }

  const getConfidenceBadge = (confidence: number) => {
    const safeConfidence = isNaN(confidence) || confidence === null || confidence === undefined ? 0 : confidence
    const percentage = Math.round(Math.max(0, Math.min(100, safeConfidence * 100)))
    let color = 'bg-gray-500'

    if (percentage >= 90) color = 'bg-green-500'
    else if (percentage >= 80) color = 'bg-blue-500'
    else if (percentage >= 70) color = 'bg-yellow-500'
    else color = 'bg-orange-500'

    return (
      <Badge className={`${color} text-white`}>
        {percentage}% уверенность
      </Badge>
    )
  }

  const totalPages = itemsPerPage > 0 ? Math.ceil(Math.max(0, total) / itemsPerPage) : 1

  if (!hasSource) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="Выберите источник данных"
        description="Для просмотра предложений выберите базу данных или проект."
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
              setFilters(newFilters as { priority: string; type: string; showApplied: boolean; autoApplyableOnly: boolean; search: string })
              setCurrentPage(1)
            }}
            onReset={() => {
              setFilters({ priority: 'all', type: 'all', showApplied: false, autoApplyableOnly: false, search: '' })
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
                Найдено предложений: <span className="font-medium text-foreground">{total}</span>
              </p>
              {analysisStatus?.is_running && analysisStatus.current_step === 'suggestions' && (
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
      {combinedError && !loading && (
        <ErrorState
          title="Ошибка загрузки предложений"
          message={combinedError}
          action={{
            label: 'Повторить',
            onClick: () => refetchSuggestions(),
          }}
          variant="destructive"
          className="mt-4"
        />
      )}

      {/* Suggestions List */}
      {isInitialLoading ? (
        <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
                <Card key={i}>
                    <CardHeader>
                        <Skeleton className="h-6 w-1/3" />
                        <Skeleton className="h-4 w-1/2 mt-2" />
                    </CardHeader>
                    <CardContent>
                        <Skeleton className="h-32 w-full" />
                    </CardContent>
                </Card>
            ))}
        </div>
      ) : suggestions.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            {analysisStatus?.is_running && analysisStatus.current_step === 'suggestions' ? (
              <div className="flex flex-col items-center justify-center py-8 space-y-4">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
                <div className="text-center space-y-2">
                  <p className="font-medium">Выполняется генерация предложений</p>
                  <p className="text-sm text-muted-foreground">
                    Прогресс: {(isNaN(analysisStatus.progress) ? 0 : analysisStatus.progress).toFixed(1)}%
                  </p>
                  {analysisStatus.suggestions_found > 0 && (
                    <p className="text-sm text-muted-foreground">
                      Найдено предложений: {analysisStatus.suggestions_found}
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <EmptyState
                icon={Lightbulb}
              title="Предложений не найдено"
              description={
                filters.showApplied
                  ? 'В базе данных нет предложений по улучшению. Все данные соответствуют стандартам качества.'
                  : 'Все предложения были применены или не найдены по заданным фильтрам. Если анализ еще не выполнялся, запустите анализ качества для генерации предложений.'
              }
            />
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {suggestions.map((suggestion) => {
            const typeConf = typeConfig[suggestion.type as keyof typeof typeConfig]

            return (
              <Card
                key={suggestion.id}
                className={`transition-all hover:shadow-md border-l-4 ${
                  suggestion.applied
                    ? 'border-green-500 opacity-70 bg-muted/30'
                    : suggestion.auto_applyable
                    ? 'border-blue-500'
                    : 'border-yellow-500'
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
                    <div className="space-y-2 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        {getPriorityBadge(suggestion.priority)}
                        {getTypeBadge(suggestion.type)}
                        {getConfidenceBadge(suggestion.confidence)}
                        {suggestion.auto_applyable && !suggestion.applied && (
                          <Badge className="bg-blue-500 text-white shadow-sm">
                            <Zap className="w-3 h-3 mr-1" />
                            Автоприменяемо
                          </Badge>
                        )}
                        {suggestion.applied && (
                          <Badge className="bg-green-50 text-green-700 border-green-200 shadow-sm">
                            <CheckCircle className="w-3 h-3 mr-1" />
                            Применено
                          </Badge>
                        )}
                        <span className="text-xs text-muted-foreground ml-auto md:ml-0 flex items-center gap-1">
                            <Calendar className="w-3 h-3" />
                            {new Date(suggestion.created_at).toLocaleDateString('ru-RU')}
                        </span>
                      </div>
                      <CardTitle className="text-lg font-semibold">
                        {typeConf?.description || suggestion.type}
                      </CardTitle>
                      <CardDescription className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-3 text-xs font-mono">
                        <span>ID: {suggestion.normalized_item_id}</span>
                        <span className="hidden sm:inline text-muted-foreground/50">•</span>
                        <span>Поле: {suggestion.field}</span>
                      </CardDescription>
                    </div>
                    {!suggestion.applied && (
                      <Button
                        size="sm"
                        onClick={() => handleApplySuggestion(suggestion.id)}
                        disabled={applyingId === suggestion.id}
                        className={`${suggestion.auto_applyable ? 'bg-blue-600 hover:bg-blue-700' : 'bg-green-600 hover:bg-green-700'} text-white shadow-sm shrink-0`}
                      >
                        {applyingId === suggestion.id ? (
                            <>
                                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                                Применение...
                            </>
                        ) : (
                            <>
                                <CheckCircle className="mr-2 h-3 w-3" />
                                Применить
                            </>
                        )}
                      </Button>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4 pt-0">
                  {/* Current vs Suggested Value */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-4 bg-muted/20 rounded-lg border">
                    <div className="space-y-2">
                      <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                        Текущее значение
                      </h4>
                      <div className="bg-background border border-border/50 rounded p-2 min-h-[2.5rem] flex items-center">
                        <code className="text-sm break-all text-foreground/80">
                          {suggestion.current_value || <span className="text-muted-foreground italic">&lt;пусто&gt;</span>}
                        </code>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground flex items-center gap-2">
                        Предлагаемое значение
                        <ArrowRight className="w-3 h-3 text-primary" />
                      </h4>
                      <div className="bg-green-50/50 dark:bg-green-900/10 border border-green-200 dark:border-green-800 rounded p-2 min-h-[2.5rem] flex items-center shadow-sm">
                        <code className="text-sm text-green-900 dark:text-green-300 font-semibold break-all">
                          {suggestion.suggested_value}
                        </code>
                      </div>
                    </div>
                  </div>

                  <div className="grid md:grid-cols-2 gap-6">
                    {/* Confidence Bar */}
                    <div className="space-y-2">
                        <div className="flex items-center justify-between text-xs">
                        <span className="text-muted-foreground font-medium">Уверенность модели</span>
                        <span className="font-bold">
                            {Math.round((isNaN(suggestion.confidence) ? 0 : suggestion.confidence) * 100)}%
                        </span>
                        </div>
                        <Progress value={Math.max(0, Math.min(100, (isNaN(suggestion.confidence) ? 0 : suggestion.confidence) * 100))} className="h-1.5" />
                    </div>

                    {/* Reasoning */}
                    {suggestion.reasoning && (
                        <div className="bg-blue-50/50 dark:bg-blue-900/10 border border-blue-100 dark:border-blue-800 rounded-lg p-3">
                        <h4 className="text-xs font-medium text-blue-700 dark:text-blue-400 mb-1 flex items-center gap-1 uppercase tracking-wider">
                            <TrendingUp className="w-3 h-3" />
                            Обоснование
                        </h4>
                        <p className="text-sm text-blue-900 dark:text-blue-300 leading-relaxed">{suggestion.reasoning}</p>
                        </div>
                    )}
                  </div>

                  {/* Metadata */}
                  {suggestion.applied && (
                    <div className="text-xs text-muted-foreground pt-3 border-t flex items-center gap-2">
                        <CheckCircle className="w-3 h-3 text-green-600" />
                        <span>Применено: {new Date(suggestion.applied_at!).toLocaleString('ru-RU')}</span>
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
