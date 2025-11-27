'use client'

import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Clock, User, RotateCcw, Eye } from 'lucide-react'
import { formatDate } from '@/lib/locale'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import { fetchChangeHistoryApi, type HistoryResponse, type HistoryEntry } from '@/lib/mdm/api'

interface ChangeHistoryProps {
  clientId?: string
  projectId?: string
  entityType?: 'group' | 'item' | 'attribute' | 'rule'
  entityId?: string
}

export const ChangeHistory: React.FC<ChangeHistoryProps> = ({
  clientId,
  projectId,
  entityType,
  entityId,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId

  const { data, loading, error, refetch } = useProjectState<HistoryResponse>(
    (cid, pid, signal) =>
      fetchChangeHistoryApi(
        cid,
        pid,
        { entityType, entityId },
        signal
      ),
    effectiveClientId || '',
    effectiveProjectId || '',
    [entityType, entityId],
    {
      enabled: !!effectiveClientId && !!effectiveProjectId,
      refetchInterval: null, // История не требует автообновления
    }
  )

  const history = data?.history || []

  const getActionBadge = (action: string) => {
    const variants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
      create: 'default',
      update: 'secondary',
      delete: 'destructive',
      merge: 'default',
      separate: 'outline',
    }

    const labels: Record<string, string> = {
      create: 'Создано',
      update: 'Обновлено',
      delete: 'Удалено',
      merge: 'Объединено',
      separate: 'Разделено',
    }

    return (
      <Badge variant={variants[action] || 'outline'}>
        {labels[action] || action}
      </Badge>
    )
  }

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>История изменений</CardTitle>
          <CardDescription>Выберите проект для просмотра истории</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (loading && history.length === 0) {
    return <LoadingState message="Загрузка истории изменений..." />
  }

  if (error) {
    return (
      <ErrorState 
        title="Ошибка загрузки истории" 
        message={error} 
        action={{
          label: 'Повторить',
          onClick: refetch,
        }}
      />
    )
  }

  if (history.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>История изменений</CardTitle>
          <CardDescription>Журнал всех изменений в нормализации</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            История изменений пуста
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>История изменений</CardTitle>
        <CardDescription>
          Журнал всех изменений в нормализации
          {entityType && ` (${entityType})`}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[400px]">
          <div className="space-y-4">
            {history.map((entry: any) => (
              <div key={entry.id} className="border rounded-lg p-4 space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {getActionBadge(entry.action)}
                    <span className="text-sm font-medium">{entry.entity_type}</span>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3" />
                    {formatDate(entry.timestamp)}
                  </div>
                </div>

                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <User className="h-3 w-3" />
                  <span>{entry.user || 'Система'}</span>
                </div>

                {entry.description && (
                  <p className="text-sm">{entry.description}</p>
                )}

                {entry.changes && entry.changes.length > 0 && (
                  <div className="mt-2 pt-2 border-t space-y-1">
                    {entry.changes.map((change: any, idx: number) => (
                      <div key={idx} className="text-xs">
                        <span className="font-medium">{change.field}:</span>{' '}
                        <span className="text-muted-foreground line-through">
                          {String(change.old_value || '—')}
                        </span>
                        {' → '}
                        <span className="font-medium">{String(change.new_value || '—')}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  )
}

