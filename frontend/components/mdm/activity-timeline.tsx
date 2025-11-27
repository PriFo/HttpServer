'use client'

import React, { useMemo, useState, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Clock, User, CheckCircle2, XCircle, AlertCircle, Info, RefreshCw } from 'lucide-react'
import { formatDate } from '@/lib/locale'
import { useActivityEvents } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import type { ActivityEvent } from '@/types/normalization'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'

interface ActivityTimelineProps {
  clientId?: string
  projectId?: string
  maxEvents?: number
}

export const ActivityTimeline: React.FC<ActivityTimelineProps> = ({
  clientId,
  projectId,
  maxEvents = 50,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId

  const [typeFilter, setTypeFilter] = useState<'all' | 'success' | 'error' | 'warning' | 'info'>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [interval, setInterval] = useState<'15s' | '30s' | '60s'>('30s')

  const intervalMs = useMemo(() => {
    switch (interval) {
      case '15s':
        return 15000
      case '60s':
        return 60000
      default:
        return 30000
    }
  }, [interval])

  // Используем специализированный хук для активности
  const { data, loading, error, refetch } = useActivityEvents(
    effectiveClientId || '',
    effectiveProjectId || '',
    maxEvents,
    { refetchInterval: autoRefresh ? intervalMs : null, enabled: !!effectiveClientId && !!effectiveProjectId }
  )

  const events: ActivityEvent[] = data?.events || []

  const filteredEvents = useMemo(() => {
    return events.filter((event: ActivityEvent) => {
      const matchesType = typeFilter === 'all' || event.type === typeFilter
      const query = searchQuery.trim().toLowerCase()
      const matchesSearch =
        !query ||
        event.description.toLowerCase().includes(query) ||
        event.user.toLowerCase().includes(query) ||
        event.action.toLowerCase().includes(query)
      return matchesType && matchesSearch
    })
  }, [events, typeFilter, searchQuery])

  const handleManualRefresh = useCallback(() => {
    refetch()
  }, [refetch])

  const getEventIcon = (type: string) => {
    switch (type) {
      case 'success':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'warning':
        return <AlertCircle className="h-4 w-4 text-yellow-600" />
      default:
        return <Info className="h-4 w-4 text-blue-600" />
    }
  }

  const getEventColor = (type: string) => {
    switch (type) {
      case 'success':
        return 'border-green-500'
      case 'error':
        return 'border-red-500'
      case 'warning':
        return 'border-yellow-500'
      default:
        return 'border-blue-500'
    }
  }

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Clock className="h-5 w-5" />
            Лента активности
          </CardTitle>
          <CardDescription>Выберите проект для просмотра истории действий</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (loading && events.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Clock className="h-5 w-5" />
            Лента активности
          </CardTitle>
          <CardDescription>
            Последние события и изменения в системе нормализации
          </CardDescription>
        </CardHeader>
        <CardContent>
          <LoadingState message="Загрузка активности..." />
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Clock className="h-5 w-5" />
            Лента активности
          </CardTitle>
          <CardDescription>
            Последние события и изменения в системе нормализации
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ErrorState 
            title="Ошибка загрузки активности" 
            message={error} 
            action={{
              label: 'Повторить',
              onClick: refetch,
            }}
          />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div>
            <CardTitle className="text-base flex items-center gap-2">
              <Clock className="h-5 w-5" />
              Лента активности
              <Badge variant="outline">{filteredEvents.length}</Badge>
            </CardTitle>
            <CardDescription>
              Последние события и изменения в системе нормализации
            </CardDescription>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <Switch
                checked={autoRefresh}
                onCheckedChange={setAutoRefresh}
                id="activity-auto-refresh"
              />
              <Label htmlFor="activity-auto-refresh" className="text-sm">
                Автообновление
              </Label>
            </div>
            <Select
              value={interval}
              onValueChange={(value: '15s' | '30s' | '60s') => setInterval(value)}
              disabled={!autoRefresh}
            >
              <SelectTrigger className="w-[110px]">
                <SelectValue placeholder="Интервал" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="15s">15 c</SelectItem>
                <SelectItem value="30s">30 c</SelectItem>
                <SelectItem value="60s">60 c</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" size="sm" onClick={handleManualRefresh} disabled={loading}>
              <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
              Обновить
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
          <Input
            placeholder="Поиск по описанию или пользователю..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          <Select value={typeFilter} onValueChange={(value: any) => setTypeFilter(value)}>
            <SelectTrigger>
              <SelectValue placeholder="Тип события" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Все типы</SelectItem>
              <SelectItem value="success">Успех</SelectItem>
              <SelectItem value="info">Информация</SelectItem>
              <SelectItem value="warning">Предупреждения</SelectItem>
              <SelectItem value="error">Ошибки</SelectItem>
            </SelectContent>
          </Select>
          <div className="text-sm text-muted-foreground flex items-center">
            Показано {filteredEvents.length} из {events.length}
          </div>
        </div>
        {filteredEvents.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            Нет событий активности
          </div>
        ) : (
          <ScrollArea className="h-[400px]">
            <div className="relative">
              {/* Вертикальная линия */}
              <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-border" />

              <div className="space-y-4">
                {filteredEvents.map((event) => (
                  <div key={event.id} className="relative flex items-start gap-4">
                    {/* Иконка события */}
                    <div className={`relative z-10 flex items-center justify-center w-8 h-8 rounded-full bg-background border-2 ${getEventColor(event.type)}`}>
                      {getEventIcon(event.type)}
                    </div>

                    {/* Содержимое события */}
                    <div className="flex-1 min-w-0 pt-1">
                      <div className="flex items-start justify-between gap-2 mb-1">
                        <div className="flex-1">
                          <p className="text-sm font-medium">{event.description}</p>
                          <div className="flex items-center gap-2 mt-1">
                            <div className="flex items-center gap-1 text-xs text-muted-foreground">
                              <User className="h-3 w-3" />
                              <span>{event.user}</span>
                            </div>
                            <span className="text-xs text-muted-foreground">•</span>
                            <div className="flex items-center gap-1 text-xs text-muted-foreground">
                              <Clock className="h-3 w-3" />
                              <span>{formatDate(event.timestamp)}</span>
                            </div>
                          </div>
                        </div>
                        <Badge variant="outline" className="text-xs">
                          {event.action}
                        </Badge>
                      </div>

                      {event.details && event.details.length > 0 && (
                        <div className="mt-2 p-2 bg-muted rounded text-xs">
                          <details>
                            <summary className="cursor-pointer text-muted-foreground">
                              Детали изменений ({event.details.length})
                            </summary>
                            <div className="mt-2 space-y-1">
                              {event.details.map((change: any, idx: number) => (
                                <div key={idx} className="text-xs">
                                  <span className="font-medium">{change.field}:</span>{' '}
                                  <span className="line-through text-muted-foreground">
                                    {String(change.old_value || '—')}
                                  </span>
                                  {' → '}
                                  <span className="font-medium">
                                    {String(change.new_value || '—')}
                                  </span>
                                </div>
                              ))}
                            </div>
                          </details>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  )
}

