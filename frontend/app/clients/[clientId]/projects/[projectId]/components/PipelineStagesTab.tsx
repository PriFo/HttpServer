'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { RefreshCw } from 'lucide-react'
import { PipelineOverview } from '@/components/pipeline/PipelineOverview'
import { PipelineFunnelChart } from '@/components/pipeline/PipelineFunnelChart'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import type { PipelineStatsData } from '@/types/normalization'

interface PipelineStagesTabProps {
  clientId: string
  projectId: string
}

async function fetchPipelineStats(
  clientId: string,
  projectId: string,
  signal?: AbortSignal
): Promise<PipelineStatsData> {
  const response = await fetch(
    `/api/clients/${clientId}/projects/${projectId}/pipeline-stats`,
    { cache: 'no-store', signal }
  )

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new Error(errorData.error || `HTTP ${response.status}`)
  }

  return response.json()
}

export function PipelineStagesTab({ clientId, projectId }: PipelineStagesTabProps) {
  const [refreshing, setRefreshing] = useState(false)

  // Используем useProjectState для загрузки статистики пайплайна
  const { data: stats, loading, error, refetch } = useProjectState(
    fetchPipelineStats,
    clientId,
    projectId,
    [],
    {
      refetchInterval: 10000, // Автообновление каждые 10 секунд
      enabled: !!clientId && !!projectId,
    }
  )

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await refetch()
    } finally {
      setRefreshing(false)
    }
  }

  if (loading && !stats) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <Skeleton className="h-8 w-64 mb-2" />
            <Skeleton className="h-4 w-96" />
          </div>
          <Skeleton className="h-10 w-32" />
        </div>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
          {[...Array(15)].map((_, i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <ErrorState
        title="Ошибка загрузки статистики пайплайна"
        message={error}
        action={{
          label: 'Повторить',
          onClick: handleRefresh,
        }}
      />
    )
  }

  if (!stats || (stats.total_records === 0 && stats.stage_stats.length === 0)) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Этапы обработки</h2>
            <p className="text-muted-foreground">
              Статистика по этапам обработки данных проекта
            </p>
          </div>
          <Button
            variant="outline"
            size="icon"
            onClick={handleRefresh}
            disabled={refreshing || loading}
          >
            <RefreshCw className={`h-4 w-4 ${refreshing || loading ? 'animate-spin' : ''}`} />
          </Button>
        </div>
        <Alert>
          <AlertDescription>
            Нет данных для отображения. Запустите нормализацию для этого проекта, чтобы увидеть статистику по этапам обработки.
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Заголовок */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Этапы обработки</h2>
          <p className="text-muted-foreground">
            Прогресс по всем этапам нормализации данных • Обновлено: {stats.last_updated ? new Date(stats.last_updated).toLocaleTimeString() : 'N/A'}
          </p>
        </div>
        <Button
          variant="outline"
          size="icon"
          onClick={handleRefresh}
          disabled={refreshing || loading}
        >
          <RefreshCw className={`h-4 w-4 ${refreshing || loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Обзор этапов</TabsTrigger>
          <TabsTrigger value="funnel">Воронка обработки</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <PipelineOverview data={stats} />
        </TabsContent>

        <TabsContent value="funnel">
          <PipelineFunnelChart data={stats.stage_stats} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

