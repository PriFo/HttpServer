'use client'

import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Activity, Clock, Zap, TrendingUp, Database, CheckCircle2, AlertCircle } from 'lucide-react'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import type { NormalizationStatus } from '@/types/normalization'
import { fetchNormalizationStatusApi } from '@/lib/mdm/api'
import { formatNumber, formatPercent, formatDuration, calculateProgress, calculateRemainingTime } from '@/utils/normalization-helpers'

interface ProcessMonitoringWorkspaceProps {
  clientId?: string
  projectId?: string
  timeRange?: 'realtime' | '24h' | '7d' | '30d'
}

export const ProcessMonitoringWorkspace: React.FC<ProcessMonitoringWorkspaceProps> = ({
  clientId,
  projectId,
  timeRange = 'realtime',
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId

  const { data: metrics, loading, error, refetch } = useProjectState<NormalizationStatus>(
    (cid, pid, signal) => fetchNormalizationStatusApi(cid, pid, signal),
    effectiveClientId || '',
    effectiveProjectId || '',
    [],
    {
      refetchInterval: 5000, // Обновляем каждые 5 секунд
      enabled: !!effectiveClientId && !!effectiveProjectId,
    }
  )

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Мониторинг процессов</CardTitle>
          <CardDescription>Выберите проект, чтобы увидеть статус нормализации</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (loading && !metrics) {
    return <LoadingState message="Загрузка метрик мониторинга..." />
  }

  if (error) {
    return (
      <ErrorState 
        title="Ошибка загрузки метрик" 
        message={error} 
        action={{
          label: 'Повторить',
          onClick: refetch,
        }}
      />
    )
  }

  if (!metrics) {
    return null
  }

  const progress = calculateProgress(metrics?.processed || 0, metrics?.total || 0)
  const speed = metrics?.speed || 0
  const remaining = (metrics?.total || 0) - (metrics?.processed || 0)
  const estimatedTime = calculateRemainingTime(remaining, speed)

  return (
    <div className="space-y-4">
      {/* Основные метрики */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Обработано
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="flex items-baseline gap-2">
                <span className="text-2xl font-bold">{formatNumber(metrics?.processed || 0)}</span>
                <span className="text-sm text-muted-foreground">из {formatNumber(metrics?.total || 0)}</span>
              </div>
              {metrics?.total && (
                <Progress value={progress} className="h-2" />
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Zap className="h-4 w-4" />
              Скорость
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-1">
              <span className="text-2xl font-bold">{formatNumber(speed, 1)}</span>
              <span className="text-sm text-muted-foreground block">записей/сек</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <TrendingUp className="h-4 w-4" />
              Прогресс
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-1">
              <span className="text-2xl font-bold">
                {formatPercent(progress / 100, 0)}
              </span>
              {estimatedTime > 0 && (
                <span className="text-sm text-muted-foreground block">
                  ~{formatDuration(estimatedTime)} осталось
                </span>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Database className="h-4 w-4" />
              Статус
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Badge 
              variant={metrics?.isRunning ? 'default' : 'secondary'}
              className={metrics?.isRunning ? 'animate-pulse' : ''}
            >
              {metrics?.isRunning ? (
                <>
                  <Activity className="h-3 w-3 mr-1 animate-pulse" />
                  В работе
                </>
              ) : (
                <>
                  <CheckCircle2 className="h-3 w-3 mr-1" />
                  Остановлено
                </>
              )}
            </Badge>
          </CardContent>
        </Card>
      </div>

      {/* Детальная информация */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Текущий этап</CardTitle>
            <CardDescription>{metrics?.currentStep || 'Не запущено'}</CardDescription>
          </CardHeader>
          <CardContent>
            {metrics?.isRunning && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Clock className="h-4 w-4" />
                <span>Обработка в процессе...</span>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Статистика</CardTitle>
            <CardDescription>Дополнительные метрики</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Групп создано:</span>
                <span className="font-medium">{metrics?.groupsCreated || 0}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Дубликатов найдено:</span>
                <span className="font-medium">{metrics?.duplicatesFound || 0}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Ошибок:</span>
                <span className={`font-medium ${(metrics?.errors || 0) > 0 ? 'text-destructive' : ''}`}>
                  {metrics?.errors || 0}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

