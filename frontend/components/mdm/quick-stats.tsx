'use client'

import React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { TrendingUp, TrendingDown, Package, Users, CheckCircle2, AlertTriangle } from 'lucide-react'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import { fetchNormalizationStats } from '@/lib/mdm/api'
import { formatNumber } from '@/utils/normalization-helpers'

interface QuickStatsProps {
  clientId?: string
  projectId?: string
}

const mapStats = (data: any) => ({
  totalGroups: data?.totalGroups ?? data?.groupsCount ?? 0,
  totalItems: data?.totalProcessed ?? data?.processedRecords ?? 0,
  normalizedCount: data?.normalizedCount ?? data?.totalGroups ?? 0,
  issuesCount: data?.issuesCount ?? data?.issues ?? 0,
  groupsTrend: data?.groupsTrend ?? 0,
  itemsTrend: data?.processedTrend ?? 0,
})

export const QuickStats: React.FC<QuickStatsProps> = ({
  clientId,
  projectId,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId

  const { data: stats, loading, error } = useProjectState(
    (cid, pid, signal) =>
      fetchNormalizationStats(cid, pid, undefined, signal).then(mapStats),
    effectiveClientId || '',
    effectiveProjectId || '',
    [],
    {
      refetchInterval: 30000, // Обновляем каждые 30 секунд
      enabled: !!effectiveClientId && !!effectiveProjectId,
    }
  )

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-muted-foreground text-sm">
          Выберите проект, чтобы увидеть сводные метрики
        </CardContent>
      </Card>
    )
  }

  if (loading && !stats) {
    return <LoadingState message="Загрузка статистики..." />
  }

  if (error) {
    return <ErrorState message={error} />
  }

  const displayStats = stats || {
    totalGroups: 0,
    totalItems: 0,
    normalizedCount: 0,
    issuesCount: 0,
    groupsTrend: 0,
    itemsTrend: 0,
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
      <Card className="hover:shadow-md transition-shadow">
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-2">
            <Package className="h-5 w-5 text-blue-600" />
            {displayStats.groupsTrend !== 0 && (
              displayStats.groupsTrend > 0 ? (
                <TrendingUp className="h-4 w-4 text-green-600" />
              ) : (
                <TrendingDown className="h-4 w-4 text-red-600" />
              )
            )}
          </div>
          <div className="text-2xl font-bold">{formatNumber(displayStats.totalGroups)}</div>
          <div className="text-xs text-muted-foreground">Групп нормализации</div>
        </CardContent>
      </Card>

      <Card className="hover:shadow-md transition-shadow">
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-2">
            <Users className="h-5 w-5 text-green-600" />
            {displayStats.itemsTrend !== 0 && (
              displayStats.itemsTrend > 0 ? (
                <TrendingUp className="h-4 w-4 text-green-600" />
              ) : (
                <TrendingDown className="h-4 w-4 text-red-600" />
              )
            )}
          </div>
          <div className="text-2xl font-bold">{formatNumber(displayStats.totalItems)}</div>
          <div className="text-xs text-muted-foreground">Обработано записей</div>
        </CardContent>
      </Card>

      <Card className="hover:shadow-md transition-shadow">
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-2">
            <CheckCircle2 className="h-5 w-5 text-green-600" />
          </div>
          <div className="text-2xl font-bold">{formatNumber(displayStats.normalizedCount)}</div>
          <div className="text-xs text-muted-foreground">Нормализовано</div>
        </CardContent>
      </Card>

      <Card className="hover:shadow-md transition-shadow">
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-2">
            <AlertTriangle className="h-5 w-5 text-red-600" />
          </div>
          <div className="text-2xl font-bold text-red-600">{formatNumber(displayStats.issuesCount)}</div>
          <div className="text-xs text-muted-foreground">Проблем</div>
        </CardContent>
      </Card>
    </div>
  )
}

