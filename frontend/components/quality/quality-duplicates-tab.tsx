'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { normalizePercentage } from '@/lib/locale'
import { Badge } from '@/components/ui/badge'
import { AlertCircle, CheckCircle, GitMerge, Star, TrendingUp, Copy, ArrowRight, Loader2, BookmarkPlus } from 'lucide-react'
import { CreateBenchmarkDialog } from './CreateBenchmarkDialog'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { Pagination } from '@/components/ui/pagination'
import { FilterBar, type FilterConfig } from '@/components/common/filter-bar'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion"
import { Skeleton } from '@/components/ui/skeleton'
import { fetchJson, getErrorMessage } from '@/lib/fetch-utils'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { useProjectState } from '@/hooks/useProjectState'

interface DuplicateItem {
  id: number
  normalized_name: string
  code: string
  category: string
  kpved_code: string
  quality_score: number
  processing_level: string
  merged_count: number
}

interface DuplicateGroup {
  id: number
  group_hash?: string
  duplicate_type?: string
  detection_method?: string
  similarity_score: number
  suggested_master_id: number
  item_count: number
  merged: boolean
  merged_at: string | null
  created_at: string
  items: DuplicateItem[]
}

interface DuplicatesResponse {
  groups: DuplicateGroup[]
  total: number
  limit: number
  offset: number
}

const methodConfig: Record<string, { label: string; color: string; icon: string }> = {
  exact: {
    label: 'Точное совпадение',
    color: 'bg-red-500 text-white',
    icon: '🔴'
  },
  exact_code: {
    label: 'Точное совпадение кода',
    color: 'bg-red-500 text-white',
    icon: '🔴'
  },
  exact_name: {
    label: 'Точное совпадение имени',
    color: 'bg-orange-500 text-white',
    icon: '🟠'
  },
  semantic: {
    label: 'Семантическое сходство',
    color: 'bg-blue-500 text-white',
    icon: '🔵'
  },
  phonetic: {
    label: 'Фонетическое сходство',
    color: 'bg-purple-500 text-white',
    icon: '🟣'
  },
  word_based: {
    label: 'По общим словам',
    color: 'bg-green-500 text-white',
    icon: '🟢'
  },
  mixed: {
    label: 'Смешанный тип',
    color: 'bg-yellow-500 text-white',
    icon: '🟡'
  }
}

interface AnalysisStatus {
  is_running: boolean
  progress: number
  current_step: string
  duplicates_found: number
}

export function QualityDuplicatesTab({ database, project }: { database: string; project?: string }) {
  const [actionError, setActionError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [mergingId, setMergingId] = useState<number | null>(null)
  const [analysisStatus, setAnalysisStatus] = useState<AnalysisStatus | null>(null)
  const [createBenchmarkOpen, setCreateBenchmarkOpen] = useState(false)
  const [selectedGroupForBenchmark, setSelectedGroupForBenchmark] = useState<DuplicateGroup | null>(null)
  const itemsPerPage = 10

  const [filters, setFilters] = useState({ showMerged: false })

  const filterConfigs: FilterConfig[] = [
    {
      type: 'checkbox',
      key: 'showMerged',
      label: 'Показать объединенные',
    },
  ]

  const stateClientKey = database ? `quality-db:${database}` : project ? `quality-project:${project}` : 'quality'
  const stateProjectKey = `${filters.showMerged ? 'merged' : 'unmerged'}:${currentPage}`

  const {
    data: duplicatesData,
    loading,
    error,
    refetch: refetchDuplicates,
  } = useProjectState<DuplicatesResponse>(
    async (_cid, _pid, signal) => {
      if (!database && !project) {
        return { groups: [], total: 0, limit: itemsPerPage, offset: 0 }
      }

      const params = new URLSearchParams({
        limit: itemsPerPage.toString(),
        offset: ((currentPage - 1) * itemsPerPage).toString(),
      })

      if (database) params.append('database', database)
      if (project) params.append('project', project)
      if (!filters.showMerged) params.append('unmerged', 'true')

      const response = await fetch(`/api/quality/duplicates?${params.toString()}`, {
        cache: 'no-store',
        signal,
      })
      if (!response.ok) {
        // Для 404 возвращаем пустые данные вместо ошибки
        if (response.status === 404) {
          return {
            groups: [],
            total: 0,
            limit: itemsPerPage,
            offset: (currentPage - 1) * itemsPerPage,
          }
        }
        const payload = await response.json().catch(() => ({}))
        throw new Error(payload?.error || 'Не удалось загрузить дубликаты')
      }
      return response.json()
    },
    stateClientKey,
    stateProjectKey,
    [database, project, filters.showMerged, currentPage],
    {
      enabled: Boolean(database || project),
      keepPreviousData: true,
    }
  )

  const groups = duplicatesData?.groups || []
  const total = duplicatesData?.total || 0
  const combinedError = error || actionError
  const isInitialLoading = loading && !duplicatesData

  if (!database && !project) {
    return (
      <Card>
        <CardContent className="py-12">
          <EmptyState
            icon={GitMerge}
            title="Выберите источник данных"
            description="Чтобы просмотреть дубликаты, выберите базу данных или проект в параметрах качества."
          />
        </CardContent>
      </Card>
    )
  }

  useEffect(() => {
    setActionError(null)
  }, [database, project, filters.showMerged, currentPage])

  // Fetch analysis status
  const fetchAnalysisStatus = useCallback(async () => {
    try {
      const data = await fetchJson<AnalysisStatus>(
        '/api/quality/analyze/status',
        {
          timeout: QUALITY_TIMEOUTS.FAST,
          cache: 'no-store',
        }
      )
      setAnalysisStatus(data)
    } catch {
      // Ignore errors, status is optional
    }
  }, [])

  useEffect(() => {
    if (database || project) {
      fetchAnalysisStatus()
    }
  }, [database, project, fetchAnalysisStatus])

  // Auto-refresh during analysis
  useEffect(() => {
    if (!database && !project) return
    
    if (analysisStatus?.is_running && (analysisStatus.current_step === 'duplicates' || analysisStatus.current_step === 'violations' || analysisStatus.current_step === 'suggestions')) {
      const interval = setInterval(() => {
        refetchDuplicates()
        fetchAnalysisStatus()
      }, 2000) // Refresh every 2 seconds during analysis
      return () => clearInterval(interval)
    } else if (analysisStatus?.is_running) {
      // Refresh less frequently during other steps
      const interval = setInterval(() => {
        fetchAnalysisStatus()
      }, 5000)
      return () => clearInterval(interval)
    }
  }, [database, project, analysisStatus?.is_running, analysisStatus?.current_step, refetchDuplicates, fetchAnalysisStatus])

  const handleMergeGroup = async (groupId: number) => {
    setMergingId(groupId)

    try {
      await fetchJson(
        `/api/quality/duplicates/${groupId}/merge`,
        {
          method: 'POST',
          timeout: QUALITY_TIMEOUTS.LONG,
          headers: {
            'Content-Type': 'application/json'
          }
        }
      )

      await refetchDuplicates()
    } catch (err) {
      const errorMessage = getErrorMessage(err, 'Не удалось объединить группу дубликатов')
      console.error(errorMessage)
      throw err
    } finally {
      setMergingId(null)
    }
  }

  const getMethodBadge = (method: string) => {
    const config = methodConfig[method as keyof typeof methodConfig]
    if (!config) return <Badge variant="secondary">{method}</Badge>

    return (
      <Badge className={config.color}>
        <span className="mr-1">{config.icon}</span>
        {config.label}
      </Badge>
    )
  }

  const getSimilarityBadge = (score: number) => {
    const safeScore = isNaN(score) || score === null || score === undefined ? 0 : score
    const percentage = Math.round(Math.max(0, Math.min(100, safeScore * 100)))
    let color = 'bg-gray-500'

    if (percentage >= 95) color = 'bg-red-500'
    else if (percentage >= 90) color = 'bg-orange-500'
    else if (percentage >= 85) color = 'bg-yellow-500'
    else color = 'bg-blue-500'

    return (
      <Badge className={`${color} text-white`}>
        {percentage}% схожесть
      </Badge>
    )
  }

  const getQualityBadge = (score: number) => {
    const safeScore = isNaN(score) || score === null || score === undefined ? 0 : score
    const percentage = Math.round(normalizePercentage(safeScore))
    let variant: 'default' | 'destructive' | 'outline' = 'outline'

    if (percentage >= 90) variant = 'default'
    else if (percentage < 70) variant = 'destructive'

    return (
      <Badge variant={variant}>
        {percentage}% качество
      </Badge>
    )
  }

  const getProcessingLevelBadge = (level: string) => {
    const labels: Record<string, string> = {
      basic: 'База',
      ai_enhanced: 'AI',
      benchmark: 'Эталон'
    }

    const colors: Record<string, string> = {
      basic: 'bg-gray-500',
      ai_enhanced: 'bg-blue-500',
      benchmark: 'bg-green-500'
    }

    return (
      <Badge className={`${colors[level] || 'bg-gray-500'} text-white text-xs`}>
        {labels[level] || level}
      </Badge>
    )
  }

  const totalPages = itemsPerPage > 0 ? Math.ceil(Math.max(0, total) / itemsPerPage) : 1

  if (!database) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="Выберите базу данных"
        description="Для просмотра дубликатов необходимо выбрать базу данных"
      />
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
            onChange={(values) => {
                setFilters(values as { showMerged: boolean })
                setCurrentPage(1)
            }}
            onReset={() => {
              setFilters({ showMerged: false })
              setCurrentPage(1)
            }}
          />
        </CardContent>
      </Card>

      {/* Summary */}
      {!isInitialLoading && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <p className="text-sm text-muted-foreground">
                  Найдено групп: <span className="font-bold text-foreground">{total}</span>
                </p>
                {analysisStatus?.is_running && analysisStatus.current_step === 'duplicates' && (
                  <Badge variant="outline" className="text-xs">
                    <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                    Анализ выполняется...
                  </Badge>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Error Alert */}
      {/* Error State */}
      {combinedError && !loading && (
        <ErrorState
          title="Ошибка загрузки дубликатов"
          message={combinedError}
          action={{
            label: 'Повторить',
            onClick: () => refetchDuplicates(),
          }}
          variant="destructive"
          className="mt-4"
        />
      )}

      {/* Duplicates List */}
      {isInitialLoading ? (
        <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
                <Card key={i}>
                    <CardHeader>
                        <Skeleton className="h-6 w-1/3" />
                        <Skeleton className="h-4 w-1/4 mt-2" />
                    </CardHeader>
                    <CardContent>
                        <Skeleton className="h-24 w-full" />
                    </CardContent>
                </Card>
            ))}
        </div>
      ) : groups.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            {analysisStatus?.is_running && analysisStatus.current_step === 'duplicates' ? (
              <div className="flex flex-col items-center justify-center py-8 space-y-4">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
                <div className="text-center space-y-2">
                  <p className="font-medium">Выполняется поиск дубликатов</p>
                  <p className="text-sm text-muted-foreground">
                    Прогресс: {(isNaN(analysisStatus.progress) ? 0 : analysisStatus.progress).toFixed(1)}%
                  </p>
                  {analysisStatus.duplicates_found > 0 && (
                    <p className="text-sm text-muted-foreground">
                      Найдено групп дубликатов: {analysisStatus.duplicates_found}
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <EmptyState
                icon={CheckCircle}
                title="Дубликатов не найдено"
                description={
                  filters.showMerged
                    ? 'В базе данных нет дубликатов. Все записи уникальны.'
                    : 'Все дубликаты были объединены или не найдены. Если анализ еще не выполнялся, запустите анализ качества для поиска дубликатов.'
                }
              />
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {groups.map((group, groupIndex) => {
            const masterItem = group.items?.find(item => item.id === group.suggested_master_id)

            return (
              <Accordion type="single" collapsible key={group.id ? `group-${group.id}-${groupIndex}` : `group-${groupIndex}`} className="bg-card border rounded-lg shadow-sm">
                <AccordionItem value={`item-${group.id}`} className="border-0">
                    <div className={`flex flex-col md:flex-row md:items-center justify-between p-4 rounded-t-lg ${group.merged ? 'bg-muted/30' : 'bg-orange-50/30 border-l-4 border-l-orange-500'}`}>
                        <div className="flex-1 space-y-2">
                            <div className="flex items-center gap-2 flex-wrap">
                                {getMethodBadge(group.duplicate_type || group.detection_method || 'unknown')}
                                {getSimilarityBadge(group.similarity_score)}
                                <Badge variant="outline" className="bg-background">
                                    <Copy className="w-3 h-3 mr-1" />
                                    {group.item_count} записей
                                </Badge>
                                {group.merged && (
                                    <Badge className="bg-green-500 text-white">
                                        <CheckCircle className="w-3 h-3 mr-1" />
                                        Объединено
                                    </Badge>
                                )}
                            </div>
                            <div className="flex items-center justify-between md:justify-start gap-4">
                                <div>
                                    <h3 className="font-semibold text-lg leading-none">Группа #{group.id}</h3>
                                    <p className="text-xs text-muted-foreground mt-1">
                                        Создано: {new Date(group.created_at).toLocaleString('ru-RU')}
                                        {group.merged_at && ` • Объединено: ${new Date(group.merged_at).toLocaleString('ru-RU')}`}
                                    </p>
                                </div>
                            </div>
                        </div>
                        
                        <div className="flex items-center gap-4 mt-4 md:mt-0">
                             {!group.merged && (
                                <>
                                    <Button
                                        size="sm"
                                        variant="outline"
                                        onClick={(e) => {
                                            e.stopPropagation()
                                            setSelectedGroupForBenchmark(group)
                                            setCreateBenchmarkOpen(true)
                                        }}
                                        className="bg-blue-50 hover:bg-blue-100 text-blue-700 border-blue-200"
                                    >
                                        <BookmarkPlus className="w-4 h-4 mr-2" />
                                        Создать эталон
                                    </Button>
                                    <Button
                                        size="sm"
                                        onClick={(e) => {
                                            e.stopPropagation() // Prevent accordion toggle
                                            handleMergeGroup(group.id)
                                        }}
                                        disabled={mergingId === group.id}
                                        className="bg-green-600 hover:bg-green-700 text-white shadow-sm"
                                    >
                                        {mergingId === group.id ? (
                                            <>
                                                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                                                Объединение...
                                            </>
                                        ) : (
                                            <>
                                                <GitMerge className="w-4 h-4 mr-2" />
                                                Объединить
                                            </>
                                        )}
                                    </Button>
                                </>
                            )}
                            <AccordionTrigger className="p-0 hover:no-underline py-2 px-4" />
                        </div>
                    </div>

                  <AccordionContent>
                    <div className="p-4 space-y-6 border-t">
                      {/* Master Record */}
                      {masterItem && (
                        <div className="bg-yellow-50/50 dark:bg-yellow-900/10 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
                          <div className="flex items-center gap-2 mb-3 text-yellow-700 dark:text-yellow-500">
                            <Star className="w-5 h-5 fill-current" />
                            <h4 className="font-semibold">Рекомендуемая мастер-запись</h4>
                          </div>
                          
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div className="space-y-1">
                                <span className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">Название</span>
                                <p className="font-medium text-lg">{masterItem.normalized_name}</p>
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-1">
                                    <span className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">Код</span>
                                    <p><code className="bg-background px-2 py-1 rounded border">{masterItem.code || 'N/A'}</code></p>
                                </div>
                                <div className="space-y-1">
                                    <span className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">Категория</span>
                                    <p>{masterItem.category}</p>
                                </div>
                            </div>
                            <div className="md:col-span-2 flex items-center gap-4 pt-2 border-t border-yellow-200/50 dark:border-yellow-800/50">
                                <div className="flex items-center gap-2">
                                    <span className="text-sm text-muted-foreground">Качество:</span>
                                    {getQualityBadge(masterItem.quality_score)}
                                </div>
                                <div className="flex items-center gap-2">
                                    <span className="text-sm text-muted-foreground">Уровень:</span>
                                    {getProcessingLevelBadge(masterItem.processing_level)}
                                </div>
                                {masterItem.merged_count > 0 && (
                                    <Badge variant="outline" className="ml-auto">
                                        <TrendingUp className="w-3 h-3 mr-1" />
                                        {masterItem.merged_count} объединений
                                    </Badge>
                                )}
                            </div>
                          </div>
                        </div>
                      )}

                      {/* Items Table */}
                      <div>
                        <h4 className="font-semibold mb-3 text-sm text-muted-foreground flex items-center gap-2">
                            <ArrowRight className="w-4 h-4" />
                            Состав группы ({group.items?.length || 0})
                        </h4>
                        <div className="border rounded-md overflow-hidden">
                            <Table>
                                <TableHeader>
                                    <TableRow className="bg-muted/50">
                                        <TableHead className="w-[50px]">ID</TableHead>
                                        <TableHead>Название</TableHead>
                                        <TableHead>Код</TableHead>
                                        <TableHead>Категория</TableHead>
                                        <TableHead>Качество</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {group.items?.map((item, itemIndex) => (
                                        <TableRow key={item.id ? `item-${item.id}-${itemIndex}` : `item-${itemIndex}`} className={item.id === group.suggested_master_id ? 'bg-yellow-50/30 dark:bg-yellow-900/10' : ''}>
                                            <TableCell className="font-mono text-xs text-muted-foreground">
                                                {item.id}
                                                {item.id === group.suggested_master_id && (
                                                    <Star className="w-3 h-3 text-yellow-500 fill-yellow-500 inline-block ml-1" />
                                                )}
                                            </TableCell>
                                            <TableCell className="font-medium">
                                                {item.normalized_name}
                                            </TableCell>
                                            <TableCell>
                                                <code className="bg-muted px-1.5 py-0.5 rounded text-xs">{item.code || '-'}</code>
                                            </TableCell>
                                            <TableCell className="text-sm text-muted-foreground">
                                                {item.category}
                                            </TableCell>
                                            <TableCell>
                                                {getQualityBadge(item.quality_score)}
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </div>
                      </div>
                    </div>
                  </AccordionContent>
                </AccordionItem>
              </Accordion>
            )
          })}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="py-4">
                <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPageChange={setCurrentPage}
                itemsPerPage={itemsPerPage}
                totalItems={total}
                />
            </div>
          )}
        </div>
      )}

      {/* Create Benchmark Dialog */}
      {selectedGroupForBenchmark && (
        <CreateBenchmarkDialog
          isOpen={createBenchmarkOpen}
          onClose={() => {
            setCreateBenchmarkOpen(false)
            setSelectedGroupForBenchmark(null)
          }}
          uploadId={selectedGroupForBenchmark.id.toString()}
          duplicateItems={(selectedGroupForBenchmark.items || []).map(item => ({
            id: item.id.toString(),
            name: item.normalized_name,
            code: item.code,
            category: item.category
          }))}
          onSuccess={() => {
            refetchDuplicates()
          }}
        />
      )}
    </div>
  )
}
