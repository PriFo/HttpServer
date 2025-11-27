'use client'

import React, { useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Button } from '@/components/ui/button'
import { CheckCircle2, AlertTriangle, Info, TrendingUp, TrendingDown, Minus } from 'lucide-react'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import { fetchQualityMetricsApi } from '@/lib/mdm/api'
import type { QualityMetricsResponse } from '@/lib/mdm/api'
import type { QualityDimension, QualityMetric, QualityMetrics } from '@/types/normalization'
import { formatPercent, formatNumber } from '@/utils/normalization-helpers'

const DEFAULT_METRICS: QualityMetrics = {
  completeness: { score: 0.85, issues: 15, trend: 0.02 },
  accuracy: { score: 0.92, issues: 8, trend: -0.01 },
  consistency: { score: 0.78, issues: 22, trend: 0.05 },
  timeliness: { score: 0.95, issues: 5, trend: 0 },
}

interface DataQualityWorkspaceProps {
  clientId?: string
  projectId?: string
}

export const DataQualityWorkspace: React.FC<DataQualityWorkspaceProps> = ({
  clientId,
  projectId,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const [activeDimension, setActiveDimension] = useState<QualityDimension>('completeness')
  
  const { data: qualityMetrics, loading, error, refetch } = useProjectState<QualityMetricsResponse>(
    fetchQualityMetricsApi,
    effectiveClientId || '',
    effectiveProjectId || '',
    [],
    {
      enabled: !!effectiveClientId && !!effectiveProjectId,
    }
  )

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Качество данных</CardTitle>
          <CardDescription>Выберите проект для просмотра метрик качества</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const metrics = useMemo<QualityMetrics>(
    () => ({
      completeness: qualityMetrics?.completeness ?? DEFAULT_METRICS.completeness,
      accuracy: qualityMetrics?.accuracy ?? DEFAULT_METRICS.accuracy,
      consistency: qualityMetrics?.consistency ?? DEFAULT_METRICS.consistency,
      timeliness: qualityMetrics?.timeliness ?? DEFAULT_METRICS.timeliness,
    }),
    [qualityMetrics]
  )

  const metricEntries = useMemo(
    () => Object.entries(metrics) as [QualityDimension, QualityMetric][],
    [metrics]
  )

  const getTrendIcon = (trend: number) => {
    if (trend > 0) return <TrendingUp className="h-4 w-4 text-green-600" />
    if (trend < 0) return <TrendingDown className="h-4 w-4 text-red-600" />
    return <Minus className="h-4 w-4 text-muted-foreground" />
  }

  const getScoreColor = (score: number) => {
    if (score >= 0.9) return 'text-green-600'
    if (score >= 0.7) return 'text-yellow-600'
    return 'text-red-600'
  }

  const getScoreVariant = (score: number): 'default' | 'secondary' | 'destructive' => {
    if (score >= 0.9) return 'default'
    if (score >= 0.7) return 'secondary'
    return 'destructive'
  }

  const currentMetric = metrics[activeDimension]

  if (loading && !qualityMetrics) {
    return <LoadingState message="Загрузка метрик качества данных..." />
  }

  if (error) {
    return (
      <ErrorState 
        title="Ошибка загрузки метрик качества" 
        message={error} 
        action={{
          label: 'Повторить',
          onClick: refetch,
        }}
      />
    )
  }

  return (
    <div className="space-y-4">
      {/* Общая статистика качества */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {metricEntries.map(([key, metric]) => (
          <Card key={key} className={key === activeDimension ? 'border-primary' : ''}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">
                {key === 'completeness' && 'Полнота'}
                {key === 'accuracy' && 'Точность'}
                {key === 'consistency' && 'Согласованность'}
                {key === 'timeliness' && 'Актуальность'}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className={`text-2xl font-bold ${getScoreColor(metric.score)}`}>
                    {formatPercent(metric.score, 0)}
                  </span>
                  {getTrendIcon(metric.trend)}
                </div>
                <Progress value={metric.score * 100} className="h-2" />
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>Проблем: {formatNumber(metric.issues)}</span>
                  {metric.trend !== 0 && (
                    <span className={metric.trend > 0 ? 'text-green-600' : 'text-red-600'}>
                      {metric.trend > 0 ? '+' : ''}{formatPercent(metric.trend, 0)}
                    </span>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Детальный анализ */}
      <Card>
        <CardHeader>
          <CardTitle>Детальный анализ качества</CardTitle>
          <CardDescription>Метрики качества нормализованных данных</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs value={activeDimension} onValueChange={(value) => setActiveDimension(value as QualityDimension)}>
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="completeness">Полнота</TabsTrigger>
              <TabsTrigger value="accuracy">Точность</TabsTrigger>
              <TabsTrigger value="consistency">Согласованность</TabsTrigger>
              <TabsTrigger value="timeliness">Актуальность</TabsTrigger>
            </TabsList>

            <TabsContent value={activeDimension} className="mt-4">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold text-lg">
                      {activeDimension === 'completeness' && 'Полнота данных'}
                      {activeDimension === 'accuracy' && 'Точность данных'}
                      {activeDimension === 'consistency' && 'Согласованность данных'}
                      {activeDimension === 'timeliness' && 'Актуальность данных'}
                    </h3>
                    <p className="text-sm text-muted-foreground mt-1">
                      Оценка качества по измерению {activeDimension}
                    </p>
                  </div>
                  <Badge variant={getScoreVariant(currentMetric.score)} className="text-base px-3 py-1">
                    {Math.round(currentMetric.score * 100)}%
                  </Badge>
                </div>

                <div className="p-4 border rounded-lg space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Info className="h-4 w-4 text-muted-foreground" />
                      <span className="text-sm font-medium">Найдено проблем</span>
                    </div>
                    <Badge variant={currentMetric.issues > 10 ? 'destructive' : 'secondary'}>
                      {currentMetric.issues}
                    </Badge>
                  </div>

                  <Progress value={currentMetric.score * 100} className="h-3" />

                  <div className="flex items-center gap-2 text-sm">
                    {getTrendIcon(currentMetric.trend)}
                    <span className="text-muted-foreground">
                      {currentMetric.trend > 0 && 'Улучшение за период'}
                      {currentMetric.trend < 0 && 'Ухудшение за период'}
                      {currentMetric.trend === 0 && 'Без изменений'}
                    </span>
                  </div>

                  <div className="pt-2 border-t">
                    <Button variant="outline" size="sm" className="w-full">
                      Просмотреть детали проблем
                    </Button>
                  </div>
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  )
}

