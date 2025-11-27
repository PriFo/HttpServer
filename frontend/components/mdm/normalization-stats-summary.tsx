'use client'

import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { TrendingUp, TrendingDown, Minus, Activity, CheckCircle2, AlertTriangle } from 'lucide-react'
import { useNormalizationMetrics } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { SkeletonLoader } from '@/components/common/skeleton-loader'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'

interface NormalizationStatsSummaryProps {
  clientId?: string
  projectId?: string
  isProcessRunning?: boolean
}

export const NormalizationStatsSummary: React.FC<NormalizationStatsSummaryProps> = ({
  clientId,
  projectId,
  isProcessRunning,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const processRunning = isProcessRunning ?? identifiers.isProcessRunning

  const { data: stats, loading, error, refetch } = useNormalizationMetrics(
    effectiveClientId || '',
    effectiveProjectId || '',
    processRunning
  )

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Сводная статистика</CardTitle>
          <CardDescription>Выберите проект для просмотра метрик нормализации</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (loading && !stats) {
    return <SkeletonLoader variant="stats" count={4} />
  }

  if (error) {
    return (
      <ErrorState
        title="Ошибка загрузки статистики"
        message={error}
        action={{
          label: 'Повторить',
          onClick: refetch,
        }}
      />
    )
  }

  if (!stats) {
    return null
  }

  const getTrendIcon = (trend: number) => {
    if (trend > 0) return <TrendingUp className="h-4 w-4 text-green-600" />
    if (trend < 0) return <TrendingDown className="h-4 w-4 text-red-600" />
    return <Minus className="h-4 w-4 text-muted-foreground" />
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <Activity className="h-4 w-4" />
            Всего групп
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-1">
            <div className="text-2xl font-bold">{stats.totalGroups || 0}</div>
            {stats.groupsTrend !== undefined && (
              <div className="flex items-center gap-1 text-xs text-muted-foreground">
                {getTrendIcon(stats.groupsTrend)}
                <span>{stats.groupsTrend > 0 ? '+' : ''}{Math.round(stats.groupsTrend * 100)}%</span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4" />
            Качество данных
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="text-2xl font-bold">
              {stats.avgQuality ? `${Math.round(stats.avgQuality * 100)}%` : '—'}
            </div>
            {stats.avgQuality && (
              <Progress value={stats.avgQuality * 100} className="h-2" />
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <Activity className="h-4 w-4" />
            Обработано записей
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-1">
            <div className="text-2xl font-bold">{stats.totalProcessed || 0}</div>
            {stats.processedTrend !== undefined && (
              <div className="flex items-center gap-1 text-xs text-muted-foreground">
                {getTrendIcon(stats.processedTrend)}
                <span>+{stats.processedTrend}</span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            Проблемы
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-1">
            <div className="text-2xl font-bold text-red-600">{stats.issuesCount || 0}</div>
            <p className="text-xs text-muted-foreground">
              {stats.issuesCount === 0 ? 'Проблем не найдено' : 'Требуют внимания'}
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

