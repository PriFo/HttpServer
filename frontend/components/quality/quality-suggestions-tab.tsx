'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { ErrorState } from '@/components/common/error-state'
import { AlertCircle, CheckCircle, Lightbulb, Zap, ArrowRight, TrendingUp, Settings, RefreshCw, GitMerge, Eye } from 'lucide-react'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { Pagination } from '@/components/ui/pagination'
import { FilterBar, type FilterConfig } from '@/components/common/filter-bar'
import { Progress } from '@/components/ui/progress'

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

const typeConfig = {
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
    color: 'bg-red-500 text-white',
  },
  high: {
    label: 'Высокий',
    color: 'bg-orange-500 text-white',
  },
  medium: {
    label: 'Средний',
    color: 'bg-yellow-500 text-white',
  },
  low: {
    label: 'Низкий',
    color: 'bg-blue-500 text-white',
  }
}

export function QualitySuggestionsTab({ database }: { database: string }) {
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [applyingId, setApplyingId] = useState<number | null>(null)
  const itemsPerPage = 20

  const [filters, setFilters] = useState({
    priority: 'all',
    type: 'all',
    showApplied: false,
    autoApplyableOnly: false,
  })

  const filterConfigs: FilterConfig[] = [
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

  const fetchSuggestions = useCallback(async () => {
    if (!database) return

    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams({
        database,
        limit: itemsPerPage.toString(),
        offset: ((currentPage - 1) * itemsPerPage).toString()
      })

      if (filters.priority !== 'all') {
        params.append('priority', filters.priority)
      }

      if (!filters.showApplied) {
        params.append('applied', 'false')
      }

      if (filters.autoApplyableOnly) {
        params.append('auto_applyable', 'true')
      }

      const response = await fetch(
        `/api/quality/suggestions?${params.toString()}`
      )

      if (!response.ok) {
        throw new Error('Failed to fetch suggestions')
      }

      const data: SuggestionsResponse = await response.json()

      // Filter by type if needed
      let filteredSuggestions = data.suggestions || []
      if (filters.type !== 'all') {
        filteredSuggestions = filteredSuggestions.filter(s => s.type === filters.type)
      }

      setSuggestions(filteredSuggestions)
      setTotal(data.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [database, filters, currentPage])

  useEffect(() => {
    if (database) {
      fetchSuggestions()
    }
  }, [database, fetchSuggestions])

  // Автоматическое обновление каждые 5 секунд, если есть активный анализ
  useEffect(() => {
    if (!database) return

    const interval = setInterval(() => {
      // Проверяем статус анализа
      fetch('/api/quality/analyze/status')
        .then(res => res.json())
        .then(status => {
          // Если анализ завершен недавно (в последние 30 секунд), обновляем данные
          if (!status.is_running && status.current_step === 'completed') {
            fetchSuggestions()
          }
        })
        .catch(() => {
          // Игнорируем ошибки проверки статуса
        })
    }, 5000)

    return () => clearInterval(interval)
  }, [database, fetchSuggestions])

  const handleApplySuggestion = async (suggestionId: number) => {
    setApplyingId(suggestionId)

    try {
      const response = await fetch(
        `/api/quality/suggestions/${suggestionId}/apply`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        }
      )

      if (!response.ok) {
        throw new Error('Failed to apply suggestion')
      }

      await fetchSuggestions()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply suggestion')
    } finally {
      setApplyingId(null)
    }
  }

  const getTypeBadge = (type: string) => {
    const config = typeConfig[type as keyof typeof typeConfig]
    if (!config) return null

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
    if (!config) return null

    return (
      <Badge className={config.color}>
        {config.label}
      </Badge>
    )
  }

  const getConfidenceBadge = (confidence: number) => {
    const percentage = Math.round(confidence * 100)
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

  const totalPages = Math.ceil(total / itemsPerPage)

  if (!database) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="Выберите базу данных"
        description="Для просмотра предложений необходимо выбрать базу данных"
      />
    )
  }

  if (loading && suggestions.length === 0) {
    return <LoadingState message="Загрузка предложений..." size="lg" fullScreen />
  }

  if (error && suggestions.length === 0) {
    return (
      <ErrorState
        title="Ошибка загрузки"
        message={error}
        variant="destructive"
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
              setFilters(newFilters as { priority: string; type: string; showApplied: boolean; autoApplyableOnly: boolean })
              setCurrentPage(1)
            }}
            onReset={() => {
              setFilters({ priority: 'all', type: 'all', showApplied: false, autoApplyableOnly: false })
              setCurrentPage(1)
            }}
          />
        </CardContent>
      </Card>

      {/* Summary */}
      {!loading && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Найдено предложений: <span className="font-bold text-foreground">{total}</span>
              </p>
              <p className="text-sm text-muted-foreground">
                Страница {currentPage} из {totalPages}
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Error Alert */}
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Suggestions List */}
      {loading && suggestions.length === 0 ? (
        <LoadingState message="Загрузка предложений..." size="lg" fullScreen />
      ) : suggestions.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            <EmptyState
              icon={CheckCircle}
              title="Предложений не найдено"
              description={
                filters.showApplied
                  ? 'В базе данных нет предложений по улучшению'
                  : 'Все предложения были применены'
              }
            />
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {suggestions.map((suggestion) => {
            const typeConf = typeConfig[suggestion.type as keyof typeof typeConfig]

            return (
              <Card
                key={suggestion.id}
                className={`border-l-4 ${
                  suggestion.applied
                    ? 'border-green-500 opacity-60'
                    : suggestion.auto_applyable
                    ? 'border-blue-500'
                    : 'border-yellow-500'
                }`}
              >
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="space-y-2 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        {getPriorityBadge(suggestion.priority)}
                        {getTypeBadge(suggestion.type)}
                        {getConfidenceBadge(suggestion.confidence)}
                        {suggestion.auto_applyable && !suggestion.applied && (
                          <Badge className="bg-blue-500 text-white">
                            <Zap className="w-3 h-3 mr-1" />
                            Автоприменяемо
                          </Badge>
                        )}
                        {suggestion.applied && (
                          <Badge className="bg-green-500 text-white">
                            <CheckCircle className="w-3 h-3 mr-1" />
                            Применено
                          </Badge>
                        )}
                      </div>
                      <CardTitle className="text-lg">
                        {typeConf?.description || suggestion.type}
                      </CardTitle>
                      <CardDescription>
                        ID записи: #{suggestion.normalized_item_id} • Поле: {suggestion.field}
                      </CardDescription>
                    </div>
                    {!suggestion.applied && (
                      <Button
                        size="sm"
                        onClick={() => handleApplySuggestion(suggestion.id)}
                        disabled={applyingId === suggestion.id}
                        className="bg-green-600 hover:bg-green-700"
                      >
                        {applyingId === suggestion.id ? 'Применение...' : 'Применить'}
                      </Button>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  {/* Current vs Suggested Value */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-muted-foreground">
                        Текущее значение:
                      </h4>
                      <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                        <code className="text-sm text-red-900 break-all">
                          {suggestion.current_value || '<пусто>'}
                        </code>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                        <ArrowRight className="w-4 h-4" />
                        Предлагаемое значение:
                      </h4>
                      <div className="bg-green-50 border border-green-200 rounded-lg p-3">
                        <code className="text-sm text-green-900 break-all">
                          {suggestion.suggested_value}
                        </code>
                      </div>
                    </div>
                  </div>

                  {/* Confidence Bar */}
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">Уверенность:</span>
                      <span className="font-medium">
                        {Math.round(suggestion.confidence * 100)}%
                      </span>
                    </div>
                    <Progress value={suggestion.confidence * 100} className="h-2" />
                  </div>

                  {/* Reasoning */}
                  {suggestion.reasoning && (
                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                      <h4 className="text-sm font-medium text-blue-900 mb-1 flex items-center gap-2">
                        <TrendingUp className="w-4 h-4" />
                        Обоснование:
                      </h4>
                      <p className="text-sm text-blue-700">{suggestion.reasoning}</p>
                    </div>
                  )}

                  {/* Metadata */}
                  <div className="text-xs text-muted-foreground pt-2 border-t flex items-center justify-between">
                    <span>Создано: {new Date(suggestion.created_at).toLocaleString('ru-RU')}</span>
                    {suggestion.applied_at && (
                      <span>Применено: {new Date(suggestion.applied_at).toLocaleString('ru-RU')}</span>
                    )}
                  </div>
                </CardContent>
              </Card>
            )
          })}

          {/* Pagination */}
          {totalPages > 1 && (
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={setCurrentPage}
              itemsPerPage={itemsPerPage}
              totalItems={total}
            />
          )}
        </div>
      )}
    </div>
  )
}
