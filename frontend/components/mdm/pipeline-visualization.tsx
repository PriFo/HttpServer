'use client'

import React, { useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { CheckCircle2, Circle, Loader2, Info } from 'lucide-react'
import { usePipelineStatus } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { getOverallStatus, getStatusText, getStatusColor, getStatusVariant, formatPercent, formatNumber, formatDuration } from '@/utils/normalization-helpers'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'

interface PipelineVisualizationProps {
  clientId: string
  projectId: string
  activeProcess?: string | null
}

interface PipelineStage {
  id: string
  name: string
  icon: string
  description: string
  metrics?: {
    records?: number
    duration?: number
    quality?: number
  }
}

const pipelineStages: PipelineStage[] = [
  { id: 'extraction', name: 'Извлечение данных', icon: '📥', description: 'Получение сырых данных из БД' },
  { id: 'cleaning', name: 'Очистка и предобработка', icon: '🧹', description: 'Нормализация форматов, удаление шума' },
  { id: 'normalization', name: 'Нормализация', icon: '🔧', description: 'Унификация наименований и атрибутов' },
  { id: 'deduplication', name: 'Дедупликация', icon: '🔍', description: 'Объединение дубликатов' },
  { id: 'classification', name: 'Классификация', icon: '🏷️', description: 'Сопоставление с классификаторами' },
  { id: 'enrichment', name: 'Обогащение', icon: '✨', description: 'Дополнение внешними данными' },
  { id: 'validation', name: 'Валидация', icon: '✅', description: 'Проверка качества данных' },
  { id: 'publication', name: 'Публикация', icon: '🚀', description: 'Экспорт нормализованных данных' },
]

export const PipelineVisualization: React.FC<PipelineVisualizationProps> = ({
  clientId,
  projectId,
  activeProcess,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const effectiveActiveProcess = useMemo(
    () => activeProcess ?? identifiers.normalizationStatus?.currentStep ?? null,
    [activeProcess, identifiers.normalizationStatus]
  )
  const [selectedStage, setSelectedStage] = useState<string | null>(null)
  
  // Используем специализированный хук для статуса пайплайна
  const { data: pipelineData, loading, error } = usePipelineStatus(
    effectiveClientId || '',
    effectiveProjectId || '',
    effectiveActiveProcess
  )

  const stageMetrics = pipelineData?.stages?.reduce((acc: Record<string, any>, stage: any) => {
    acc[stage.id] = stage.metrics
    return acc
  }, {}) || {}

  const getStageStatus = (stageId: string, index: number) => {
    // Используем данные из API если доступны
    const stageData = pipelineData?.stages?.find((s: any) => s.id === stageId)
    if (stageData?.status) {
      return stageData.status
    }
    
    // Fallback логика на основе activeProcess
    if (effectiveActiveProcess === stageId) return 'active'
    const activeIndex = pipelineStages.findIndex(s => s.id === effectiveActiveProcess)
    if (activeIndex === -1) return 'completed'
    return index < activeIndex ? 'completed' : 'pending'
  }

  // Получаем общий статус пайплайна
  const overallStatus = pipelineData?.stages 
    ? getOverallStatus(pipelineData.stages.map((s: any) => ({ status: s.status || 'pending' })))
    : 'pending'

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <Loader2 className="h-4 w-4 animate-spin text-primary" />
      case 'completed':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      default:
        return <Circle className="h-4 w-4 text-muted-foreground" />
    }
  }

  const getStatusBadge = (status: string) => {
    const statusText = getStatusText(status)
    const variant = getStatusVariant(status)
    const isActive = status === 'active' || status === 'processing'
    
    return (
      <Badge variant={variant} className={isActive ? 'animate-pulse' : ''}>
        {statusText}
      </Badge>
    )
  }

  if (loading && !pipelineData) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Пайплайн обработки данных</CardTitle>
          <CardDescription>Этапы нормализации данных с метриками</CardDescription>
        </CardHeader>
        <CardContent>
          <LoadingState message="Загрузка статуса пайплайна..." />
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Пайплайн обработки данных</CardTitle>
          <CardDescription>Этапы нормализации данных с метриками</CardDescription>
        </CardHeader>
        <CardContent>
          <ErrorState message={error} />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Пайплайн обработки данных</CardTitle>
        <CardDescription>Этапы нормализации данных с метриками</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {pipelineStages.map((stage, index) => {
            const status = getStageStatus(stage.id, index)
            const isSelected = selectedStage === stage.id
            const metrics = stageMetrics[stage.id]

            return (
              <div
                key={stage.id}
                className={`relative border rounded-lg p-4 transition-all ${
                  isSelected ? 'border-primary shadow-md' : 'border-border'
                } ${status === 'active' ? 'bg-primary/5' : ''}`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-start gap-3 flex-1">
                    <div className="flex items-center gap-2 min-w-[180px]">
                      <span className="text-2xl">{stage.icon}</span>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{stage.name}</span>
                          {getStatusIcon(status)}
                        </div>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {stage.description}
                        </p>
                      </div>
                    </div>

                    {/* Соединительная линия */}
                    {index < pipelineStages.length - 1 && (
                      <div className="absolute left-[90px] top-[60px] w-0.5 h-8 bg-border" />
                    )}

                    {/* Метрики этапа */}
                    {metrics && (
                      <div className="flex gap-4 text-sm text-muted-foreground">
                        {metrics.records && (
                          <div>
                            <span className="font-medium text-foreground">{metrics.records}</span> записей
                          </div>
                        )}
                        {metrics.duration && (
                          <div>
                            <span className="font-medium text-foreground">{metrics.duration}с</span>
                          </div>
                        )}
                        {metrics.quality && (
                          <div>
                            Качество: <span className="font-medium text-foreground">{Math.round(metrics.quality * 100)}%</span>
                          </div>
                        )}
                      </div>
                    )}
                  </div>

                  <div className="flex items-center gap-2">
                    {getStatusBadge(status)}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSelectedStage(isSelected ? null : stage.id)}
                    >
                      <Info className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                {/* Детальная информация */}
                {isSelected && (
                  <div className="mt-4 pt-4 border-t">
                    <div className="space-y-2 text-sm">
                      <p className="text-muted-foreground">{stage.description}</p>
                      {metrics ? (
                        <div className="grid grid-cols-3 gap-4">
                          <div>
                            <div className="text-xs text-muted-foreground">Обработано</div>
                            <div className="font-semibold">{formatNumber(metrics.records || 0)}</div>
                          </div>
                          <div>
                            <div className="text-xs text-muted-foreground">Время</div>
                            <div className="font-semibold">{formatDuration(metrics.duration || 0)}</div>
                          </div>
                          <div>
                            <div className="text-xs text-muted-foreground">Качество</div>
                            <div className="font-semibold">
                              {metrics.quality ? formatPercent(metrics.quality, 0) : '—'}
                            </div>
                          </div>
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground italic">
                          Метрики будут доступны после выполнения этапа
                        </p>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}

