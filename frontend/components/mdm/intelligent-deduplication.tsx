'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Users, Merge, X, Search, CheckCircle2, AlertCircle, Eye } from 'lucide-react'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { DuplicateClusterView } from './duplicate-cluster-view'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import { logger } from '@/lib/logger'
import { handleError } from '@/lib/error-handler'
import { fetchDuplicateClustersApi } from '@/lib/mdm/api'
import type { DuplicateClustersResponse } from '@/types/normalization'
import { formatPercent } from '@/utils/normalization-helpers'

interface IntelligentDeduplicationProps {
  clientId?: string
  projectId?: string
}

export const IntelligentDeduplication: React.FC<IntelligentDeduplicationProps> = ({
  clientId,
  projectId,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const [similarityThreshold, setSimilarityThreshold] = useState([0.85])
  const [maxClusterSize, setMaxClusterSize] = useState(10)
  const [expandedClusters, setExpandedClusters] = useState<Set<string>>(new Set())
  const [selectedCluster, setSelectedCluster] = useState<any | null>(null)
  const [showClusterDetail, setShowClusterDetail] = useState(false)

  const { data: duplicatesData, loading, error, refetch } = useProjectState<DuplicateClustersResponse>(
    (cid, pid, signal) =>
      fetchDuplicateClustersApi(
        cid,
        pid,
        { similarityThreshold: similarityThreshold[0], maxClusterSize },
        signal
      ),
    effectiveClientId || '',
    effectiveProjectId || '',
    [similarityThreshold[0], maxClusterSize],
    {
      enabled: !!effectiveClientId && !!effectiveProjectId,
      refetchInterval: null,
    }
  )

  const duplicateClusters = duplicatesData?.clusters || []

  const handleSeparate = async (clusterId: string) => {
    if (!effectiveClientId || !effectiveProjectId) return
    try {
      const response = await fetch(
        `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/duplicates`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            action: 'separate',
            clusterId,
          }),
        }
      )
      if (response.ok) {
        // Обновляем данные после разделения
        await refetch()
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'IntelligentDeduplication',
          action: 'separate',
          clusterId,
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось разделить кластер',
      })
    }
  }

  const handleMerge = async (clusterId: string, selectedAttributes: Record<string, any>) => {
    if (!effectiveClientId || !effectiveProjectId) return
    try {
      const response = await fetch(
        `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/duplicates`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            action: 'merge',
            clusterId,
            selectedAttributes,
          }),
        }
      )
      if (response.ok) {
        // Обновляем данные после слияния
        await refetch()
        setShowClusterDetail(false)
        setSelectedCluster(null)
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'IntelligentDeduplication',
          action: 'merge',
          clusterId,
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось объединить кластер',
      })
    }
  }

  const handleExclude = async (clusterId: string, itemId: string) => {
    if (!effectiveClientId || !effectiveProjectId) return
    // Исключение записи из группы
    try {
      const response = await fetch(
        `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/duplicates`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            action: 'exclude',
            clusterId,
            itemId,
          }),
        }
      )
      if (response.ok) {
        // Обновляем данные после исключения
        await refetch()
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'IntelligentDeduplication',
          action: 'exclude',
          clusterId,
          itemId,
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось исключить элемент',
      })
    }
  }

  const handleViewCluster = (cluster: any) => {
    setSelectedCluster(cluster)
    setShowClusterDetail(true)
  }

  const toggleCluster = (clusterId: string) => {
    setExpandedClusters(prev => {
      const newSet = new Set(prev)
      if (newSet.has(clusterId)) {
        newSet.delete(clusterId)
      } else {
        newSet.add(clusterId)
      }
      return newSet
    })
  }

  if (loading && duplicateClusters.length === 0) {
    return <LoadingState message="Загрузка кластеров дубликатов..." />
  }

  if (error) {
    return (
      <ErrorState 
        title="Ошибка загрузки дубликатов" 
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
      <Card>
        <CardHeader>
          <CardTitle>Интеллектуальная дедупликация</CardTitle>
          <CardDescription>
            Настройка алгоритма и управление дубликатами
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="threshold">
                Порог схожести (%)
              </Label>
              <Input
                id="threshold"
                type="number"
                min="0"
                max="100"
                step="5"
                value={Math.round(similarityThreshold[0] * 100)}
                onChange={(e) => {
                  const value = Math.max(0, Math.min(100, parseInt(e.target.value) || 0))
                  setSimilarityThreshold([value / 100])
                }}
              />
              <p className="text-xs text-muted-foreground">
                Текущее значение: {Math.round(similarityThreshold[0] * 100)}%
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="clusterSize">Максимальный размер кластера</Label>
              <Input
                id="clusterSize"
                type="number"
                min="2"
                max="50"
                value={maxClusterSize}
                onChange={(e) => setMaxClusterSize(parseInt(e.target.value) || 10)}
              />
              <p className="text-xs text-muted-foreground">
                Группы больше этого размера будут разделены
              </p>
            </div>
          </div>

          <div className="flex gap-2">
            <Button 
              onClick={() => {
                // Обновляем данные для поиска дубликатов
                refetch()
              }}
              disabled={loading}
            >
              <Search className="h-4 w-4 mr-2" />
              Найти дубликаты
            </Button>
            {duplicateClusters.length > 0 && (
              <Button variant="outline">
                <Merge className="h-4 w-4 mr-2" />
                Объединить все ({duplicateClusters.length})
              </Button>
            )}
          </div>

          {loading ? (
            <div className="text-center py-8 text-muted-foreground">
              <div className="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full mx-auto mb-4" />
              <p>Поиск дубликатов...</p>
            </div>
          ) : duplicateClusters.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Users className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="mb-2">Дубликаты не найдены</p>
              <p className="text-xs">Нажмите "Найти дубликаты" для начала поиска</p>
            </div>
          ) : (
            <div className="space-y-2">
              <div className="flex items-center justify-between mb-4">
                <p className="text-sm font-medium">
                  Найдено кластеров: {duplicateClusters.length}
                </p>
                <Badge variant="secondary">
                  Всего записей: {duplicateClusters.reduce((sum: number, c: any) => sum + (c.items?.length || 0), 0)}
                </Badge>
              </div>
              {duplicateClusters.map((cluster: any) => {
                const isExpanded = expandedClusters.has(cluster.id)
                return (
                  <Collapsible key={cluster.id} open={isExpanded} onOpenChange={() => toggleCluster(cluster.id)}>
                    <Card>
                      <CollapsibleTrigger asChild>
                        <CardContent className="p-4 cursor-pointer hover:bg-muted/50 transition-colors">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-3">
                              <div className="flex items-center gap-2">
                                <Users className="h-5 w-5 text-muted-foreground" />
                                <div>
                                  <p className="font-medium">{cluster.name || `Кластер ${cluster.id}`}</p>
                                  <p className="text-sm text-muted-foreground">
                                    {cluster.items?.length || 0} записей
                                    {cluster.similarity && ` • Схожесть: ${Math.round(cluster.similarity * 100)}%`}
                                  </p>
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              <Badge variant={cluster.similarity && cluster.similarity >= similarityThreshold[0] ? 'default' : 'secondary'}>
                                {cluster.similarity && cluster.similarity >= similarityThreshold[0] ? (
                                  <CheckCircle2 className="h-3 w-3 mr-1" />
                                ) : (
                                  <AlertCircle className="h-3 w-3 mr-1" />
                                )}
                                {cluster.similarity && cluster.similarity >= similarityThreshold[0] ? 'Высокая' : 'Средняя'}
                              </Badge>
                            </div>
                          </div>
                        </CardContent>
                      </CollapsibleTrigger>
                      <CollapsibleContent>
                        <CardContent className="pt-0 pb-4">
                          <div className="space-y-3 mt-2">
                            {cluster.items?.map((item: any, idx: number) => (
                              <div key={idx} className="p-3 border rounded-lg bg-muted/30">
                                <div className="flex items-center justify-between">
                                  <div>
                                    <p className="font-medium text-sm">{item.name || item.source_name}</p>
                                    {item.code && (
                                      <p className="text-xs text-muted-foreground">Код: {item.code}</p>
                                    )}
                                  </div>
                                  {item.similarity && (
                                    <Badge variant="outline" className="text-xs">
                                      {formatPercent(item.similarity, 0)}
                                    </Badge>
                                  )}
                                </div>
                              </div>
                            ))}
                            <div className="flex gap-2 pt-2">
                              <Button 
                                size="sm" 
                                variant="outline"
                                onClick={() => handleViewCluster(cluster)}
                                className="flex-1"
                              >
                                <Eye className="h-4 w-4 mr-2" />
                                Детали
                              </Button>
                              <Button 
                                size="sm" 
                                onClick={() => handleMerge(cluster.id, {})}
                                className="flex-1"
                              >
                                <Merge className="h-4 w-4 mr-2" />
                                Объединить
                              </Button>
                              <Button 
                                size="sm" 
                                variant="outline"
                                onClick={() => handleSeparate(cluster.id)}
                              >
                                <X className="h-4 w-4" />
                              </Button>
                            </div>
                          </div>
                        </CardContent>
                      </CollapsibleContent>
                    </Card>
                  </Collapsible>
                )
              })}
            </div>
          )}

          {/* Диалог детального просмотра кластера */}
          {selectedCluster && (
            <Dialog open={showClusterDetail} onOpenChange={setShowClusterDetail}>
              <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                  <DialogTitle>Детальный просмотр группы дубликатов</DialogTitle>
                  <DialogDescription>
                    Выберите атрибуты для слияния и просмотрите результат
                  </DialogDescription>
                </DialogHeader>
                <DuplicateClusterView
                  cluster={selectedCluster}
                  onMerge={handleMerge}
                  onSeparate={handleSeparate}
                  onExclude={handleExclude}
                />
              </DialogContent>
            </Dialog>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

