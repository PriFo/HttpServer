'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { AlertCircle, AlertTriangle, Info, XCircle, CheckCircle } from 'lucide-react'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { Pagination } from '@/components/ui/pagination'
import { FilterBar, type FilterConfig } from '@/components/common/filter-bar'

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
    color: 'bg-red-500 text-white',
    borderColor: 'border-red-500',
  },
  error: {
    label: 'Ошибка',
    icon: AlertCircle,
    color: 'bg-orange-500 text-white',
    borderColor: 'border-orange-500',
  },
  warning: {
    label: 'Предупреждение',
    icon: AlertTriangle,
    color: 'bg-yellow-500 text-white',
    borderColor: 'border-yellow-500',
  },
  info: {
    label: 'Информация',
    icon: Info,
    color: 'bg-blue-500 text-white',
    borderColor: 'border-blue-500',
  }
}

const categoryConfig = {
  completeness: 'Полнота данных',
  accuracy: 'Точность',
  consistency: 'Согласованность',
  format: 'Формат'
}

export function QualityViolationsTab({ database }: { database: string }) {
  const [violations, setViolations] = useState<Violation[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [resolvingId, setResolvingId] = useState<number | null>(null)
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

  const fetchViolations = useCallback(async () => {
    if (!database) return

    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams({
        database,
        limit: itemsPerPage.toString(),
        offset: ((currentPage - 1) * itemsPerPage).toString()
      })

      if (filters.severity !== 'all') {
        params.append('severity', filters.severity)
      }

      if (filters.category !== 'all') {
        params.append('category', filters.category)
      }

      const response = await fetch(
        `/api/quality/violations?${params.toString()}`
      )

      if (!response.ok) {
        throw new Error('Failed to fetch violations')
      }

      const data: ViolationsResponse = await response.json()

      // Filter out resolved if needed and apply search
      let filteredViolations = data.violations || []
      if (!filters.showResolved) {
        filteredViolations = filteredViolations.filter(v => !v.resolved)
      }
      if (filters.search) {
        filteredViolations = filteredViolations.filter(v =>
          v.rule_name.toLowerCase().includes(filters.search.toLowerCase()) ||
          v.message.toLowerCase().includes(filters.search.toLowerCase())
        )
      }

      setViolations(filteredViolations)
      setTotal(data.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [database, filters, currentPage])

  useEffect(() => {
    if (database) {
      fetchViolations()
    }
  }, [database, fetchViolations])

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
            fetchViolations()
          }
        })
        .catch(() => {
          // Игнорируем ошибки проверки статуса
        })
    }, 5000)

    return () => clearInterval(interval)
  }, [database, fetchViolations])

  const handleResolveViolation = async (violationId: number) => {
    setResolvingId(violationId)

    try {
      const response = await fetch(
        `/api/quality/violations/${violationId}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            resolved_by: 'User'
          })
        }
      )

      if (!response.ok) {
        throw new Error('Failed to resolve violation')
      }

      await fetchViolations()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resolve violation')
    } finally {
      setResolvingId(null)
    }
  }

  const getSeverityBadge = (severity: string) => {
    const config = severityConfig[severity as keyof typeof severityConfig]
    if (!config) return null

    const Icon = config.icon

    return (
      <Badge className={config.color}>
        <Icon className="w-3 h-3 mr-1" />
        {config.label}
      </Badge>
    )
  }

  const getCategoryBadge = (category: string) => {
    const label = categoryConfig[category as keyof typeof categoryConfig] || category
    return (
      <Badge variant="outline">
        {label}
      </Badge>
    )
  }

  const totalPages = Math.ceil(total / itemsPerPage)

  if (!database) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="Выберите базу данных"
        description="Для просмотра нарушений необходимо выбрать базу данных"
      />
    )
  }

  if (loading && violations.length === 0) {
    return <LoadingState message="Загрузка нарушений..." size="lg" fullScreen />
  }

  if (error && violations.length === 0) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>{error}</AlertDescription>
      </Alert>
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
      {!loading && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Найдено нарушений: <span className="font-bold text-foreground">{total}</span>
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

      {/* Violations List */}
      {loading && violations.length === 0 ? (
        <LoadingState message="Загрузка нарушений..." size="lg" fullScreen />
      ) : violations.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            <EmptyState
              icon={CheckCircle}
              title="Нарушений не найдено"
              description={
                filters.showResolved
                  ? 'В базе данных нет нарушений качества'
                  : 'Все нарушения были решены'
              }
            />
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {violations.map((violation) => {
            const severityConf = severityConfig[violation.severity as keyof typeof severityConfig]

            return (
              <Card
                key={violation.id}
                className={`border-l-4 ${severityConf?.borderColor || 'border-gray-500'} ${
                  violation.resolved ? 'opacity-60' : ''
                }`}
              >
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="space-y-2 flex-1">
                      <div className="flex items-center gap-2">
                        {getSeverityBadge(violation.severity)}
                        {getCategoryBadge(violation.category)}
                        {violation.resolved && (
                          <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                            <CheckCircle className="w-3 h-3 mr-1" />
                            Решено
                          </Badge>
                        )}
                      </div>
                      <CardTitle className="text-lg">
                        {violation.rule_name}
                      </CardTitle>
                      <CardDescription>
                        ID записи: #{violation.normalized_item_id}
                        {violation.field_name && ` • Поле: ${violation.field_name}`}
                      </CardDescription>
                    </div>
                    {!violation.resolved && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleResolveViolation(violation.id)}
                        disabled={resolvingId === violation.id}
                      >
                        {resolvingId === violation.id ? 'Решение...' : 'Решить'}
                      </Button>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <h4 className="text-sm font-medium mb-1">Сообщение:</h4>
                    <p className="text-sm text-muted-foreground">{violation.message}</p>
                  </div>

                  {violation.current_value && (
                    <div>
                      <h4 className="text-sm font-medium mb-1">Текущее значение:</h4>
                      <code className="text-sm bg-muted px-2 py-1 rounded">
                        {violation.current_value}
                      </code>
                    </div>
                  )}

                  {violation.recommendation && (
                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                      <h4 className="text-sm font-medium text-blue-900 mb-1">
                        Рекомендация:
                      </h4>
                      <p className="text-sm text-blue-700">{violation.recommendation}</p>
                    </div>
                  )}

                  {violation.resolved && (
                    <div className="text-xs text-muted-foreground pt-2 border-t">
                      Решено {violation.resolved_by}
                      {violation.resolved_at && ` • ${new Date(violation.resolved_at).toLocaleString('ru-RU')}`}
                    </div>
                  )}

                  <div className="text-xs text-muted-foreground">
                    Создано: {new Date(violation.created_at).toLocaleString('ru-RU')}
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
