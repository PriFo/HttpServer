'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { StatCard } from "@/components/common/stat-card"
import { Badge } from "@/components/ui/badge"
import { 
  Target, 
  Package, 
  Users, 
  Database, 
  Activity,
  AlertCircle,
  RefreshCw,
  Link2,
  Zap,
  Loader2,
  Settings
} from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { ErrorState } from "@/components/common/error-state"
import { formatDate, formatNumber } from "@/lib/locale"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"

interface ProjectStatistics {
  project_id: number
  project_name: string
  project_type: string
  status: string
  total_nomenclature: number
  total_counterparties: number
  total_databases: number
  avg_quality_score: number
  last_updated: string
  duplicate_groups?: number
  duplicates_count?: number
  multi_database_count?: number
  configs?: Record<string, number>
  quality_stats?: ProjectQualityStats
}

interface UnlinkedDatabase {
  id: number
  name: string
  path: string
  size?: number
  config_name?: string
  display_name?: string
}

interface ClientStatistics {
  total_projects: number
  total_benchmarks: number
  active_sessions: number
  avg_quality_score: number
  total_nomenclature: number
  total_counterparties: number
  total_databases: number
  projects: ProjectStatistics[]
  unlinked_databases?: UnlinkedDatabase[]
  unlinked_databases_count?: number
  duplicate_summary?: {
    total_groups?: number
    total_records?: number
    multi_database_counterparties?: number
  }
}

interface QualityLevelStat {
  count: number
  avg_quality: number
  percentage: number
}

interface ProjectQualityStats {
  total_items: number
  by_level?: Record<string, QualityLevelStat>
  average_quality: number
  benchmark_count: number
  benchmark_percentage: number
  databases?: Array<{
    database_id: number
    database_name: string
    database_path: string
    last_activity?: string
    last_upload_at?: string
    last_used_at?: string
    stats?: {
      total_items?: number
      average_quality?: number
      benchmark_count?: number
    }
  }>
  databases_count?: number
  databases_processed?: number
  last_activity?: string
}

const QUALITY_LEVEL_LABELS: Record<string, string> = {
  basic: 'Базовый',
  ai_enhanced: 'AI',
  benchmark: 'Эталон'
}

interface StatisticsTabProps {
  clientId: string
}

export function StatisticsTab({ clientId }: StatisticsTabProps) {
  const [statistics, setStatistics] = useState<ClientStatistics | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastFetchTime, setLastFetchTime] = useState<number | null>(null)
  const [linkDialogOpen, setLinkDialogOpen] = useState(false)
  const [databaseToLink, setDatabaseToLink] = useState<UnlinkedDatabase | null>(null)
  const [selectedLinkProjectId, setSelectedLinkProjectId] = useState<number | null>(null)
  const [isLinking, setIsLinking] = useState(false)
  const [isAutoLinking, setIsAutoLinking] = useState(false)
  const [backendStatus, setBackendStatus] = useState<'unknown' | 'ok' | 'unreachable'>('unknown')
  const backendErrorToastAt = useRef<number>(0)
  const [projectQuality, setProjectQuality] = useState<Record<number, { data?: ProjectQualityStats; loading: boolean; error?: string }>>({})

  const markBackendHealthy = useCallback(() => {
    setBackendStatus((prev) => (prev === 'ok' ? prev : 'ok'))
  }, [])

  const notifyBackendUnavailable = useCallback((message: string, forceToast = false) => {
    setBackendStatus((prev) => (prev === 'unreachable' ? prev : 'unreachable'))
    const now = Date.now()
    const throttleWindow = forceToast ? 5000 : 60000
    if (now - backendErrorToastAt.current > throttleWindow) {
      toast.error(message)
      backendErrorToastAt.current = now
    }
  }, [])

  const isBackendConnectionError = useCallback((message: string) => {
    const normalized = message.toLowerCase()
    return (
      normalized.includes('backend') ||
      normalized.includes('9999') ||
      normalized.includes('failed to fetch') ||
      normalized.includes('networkerror')
    )
  }, [])

  const fetchStatistics = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000)

      const response = await fetch(`/api/clients/${clientId}/statistics`, {
        cache: 'no-store',
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        // Обработка различных статусов ошибок
        if (response.status === 404) {
          throw new Error('Клиент не найден')
        } else if (response.status === 503 || response.status === 504) {
          throw new Error('Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.')
        } else if (response.status >= 500) {
          throw new Error('Ошибка сервера при загрузке статистики')
        } else {
          const errorData = await response.json().catch(() => ({ error: 'Не удалось загрузить статистику' }))
          throw new Error(errorData.error || 'Не удалось загрузить статистику')
        }
      }

      const data = await response.json()
      setStatistics(data)
      setLastFetchTime(Date.now())
      markBackendHealthy()
    } catch (error) {
      console.error('Failed to fetch statistics:', error)
      
      let errorMessage = 'Не удалось загрузить статистику'
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          errorMessage = 'Превышено время ожидания ответа от сервера'
        } else if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
          errorMessage = 'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.'
        } else {
          errorMessage = error.message
        }
      }
      
      setError(errorMessage)
      if (isBackendConnectionError(errorMessage)) {
        notifyBackendUnavailable(errorMessage, true)
      }
    } finally {
      setIsLoading(false)
    }
  }, [clientId, markBackendHealthy, isBackendConnectionError, notifyBackendUnavailable])

  useEffect(() => {
    fetchStatistics()
  }, [clientId, fetchStatistics])

  const handleBackendRetry = useCallback(() => {
    setBackendStatus('unknown')
    backendErrorToastAt.current = 0
    fetchStatistics()
  }, [fetchStatistics])

  const fetchProjectQualityStats = useCallback(async (projectId: number) => {
    setProjectQuality((prev) => ({
      ...prev,
      [projectId]: { ...prev[projectId], loading: true, error: undefined }
    }))

    try {
      const response = await fetch(`/api/quality/stats?project=${clientId}:${projectId}`, {
        cache: 'no-store'
      })

      if (!response.ok) {
        const errorBody = await response.json().catch(() => null)
        throw new Error(errorBody?.error || 'Не удалось загрузить метрики качества')
      }

      const data = await response.json()
      setProjectQuality((prev) => ({
        ...prev,
        [projectId]: { data, loading: false, error: undefined }
      }))
      markBackendHealthy()
    } catch (fetchError) {
      const errorMessage = fetchError instanceof Error ? fetchError.message : 'Не удалось загрузить метрики качества'
      setProjectQuality((prev) => ({
        ...prev,
        [projectId]: { ...prev[projectId], loading: false, error: errorMessage }
      }))
      if (isBackendConnectionError(errorMessage)) {
        notifyBackendUnavailable(errorMessage)
      }
    }
  }, [clientId, markBackendHealthy, isBackendConnectionError, notifyBackendUnavailable])

  const handleOpenLinkDialog = (db: UnlinkedDatabase) => {
    setDatabaseToLink(db)
    setSelectedLinkProjectId(null)
    setLinkDialogOpen(true)
  }

  const handleCloseLinkDialog = () => {
    setLinkDialogOpen(false)
    setDatabaseToLink(null)
    setSelectedLinkProjectId(null)
  }

  const handleConfirmLink = async () => {
    if (!databaseToLink || !selectedLinkProjectId) {
      toast.error('Выберите проект для привязки')
      return
    }

    setIsLinking(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/databases/${databaseToLink.id}/link`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ project_id: selectedLinkProjectId }),
      })

      if (!response.ok) {
        const error = await response.json().catch(() => ({}))
        throw new Error(error.error || 'Не удалось привязать базу данных')
      }

      const projectName = statistics?.projects?.find(p => p.project_id === selectedLinkProjectId)?.project_name
      toast.success('База данных привязана', {
        description: projectName 
          ? `База "${databaseToLink.name}" привязана к проекту "${projectName}"`
          : `База "${databaseToLink.name}" успешно привязана`,
        duration: 4000,
      })

      markBackendHealthy()
      await fetchStatistics()
      handleCloseLinkDialog()
    } catch (err) {
      console.error('Failed to link database:', err)
      const description = err instanceof Error ? err.message : 'Не удалось привязать базу данных'
      if (isBackendConnectionError(description)) {
        notifyBackendUnavailable(description, true)
      } else {
        toast.error('Ошибка привязки', {
          description,
          duration: 5000,
        })
      }
    } finally {
      setIsLinking(false)
    }
  }

  const handleAutoLink = async () => {
    setIsAutoLinking(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/databases/auto-link`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        const error = await response.json().catch(() => ({}))
        throw new Error(error.error || 'Не удалось выполнить автоматическую привязку')
      }

      const result = await response.json()
      const linked = result.linked_count || 0
      const total = result.total_databases ?? (linked + (result.unlinked_count || 0))

      if (linked > 0) {
        toast.success('Авто-привязка завершена', {
          description: `Привязано баз: ${linked} из ${total}.`,
          duration: 5000,
        })
      } else {
        toast.info('Нет баз для привязки', {
          description: 'Непривязанных баз данных не найдено.',
          duration: 4000,
        })
      }

      if (Array.isArray(result.errors) && result.errors.length > 0) {
        toast.warning('Часть баз не привязана', {
          description: `Ошибок: ${result.errors.length}. Подробности в консоли.`,
          duration: 5000,
        })
        console.error('Ошибки авто-привязки:', result.errors)
      }

      markBackendHealthy()
      await fetchStatistics()
    } catch (err) {
      console.error('Failed to auto-link databases:', err)
      const description = err instanceof Error ? err.message : 'Не удалось выполнить автоматическую привязку'
      if (isBackendConnectionError(description)) {
        notifyBackendUnavailable(description, true)
      } else {
        toast.error('Ошибка автоматической привязки', {
          description,
          duration: 5000,
        })
      }
    } finally {
      setIsAutoLinking(false)
    }
  }

  if (isLoading) {
    return <LoadingState message="Загрузка статистики..." />
  }

  if (error && !statistics) {
    return (
      <div className="space-y-4">
        <ErrorState
          title="Ошибка загрузки"
          message={error}
          action={{
            label: 'Повторить',
            onClick: handleBackendRetry,
          }}
          variant="destructive"
        />
        {error.includes('подключиться к backend') && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              <div className="space-y-2">
                <p>Для загрузки статистики необходимо запустить backend сервер.</p>
                <p className="text-sm text-muted-foreground">
                  Используйте скрипт <code className="bg-muted px-1 rounded">start-backend-exe.bat</code> для запуска сервера.
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleBackendRetry}
                  className="mt-2"
                >
                  <RefreshCw className="h-3 w-3 mr-1" />
                  Повторить попытку
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {backendStatus === 'unreachable' && (
        <Alert variant="destructive">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Backend недоступен</AlertTitle>
          </div>
          <AlertDescription className="mt-2 space-y-2">
            <p>Не удаётся подключиться к API (порт 9999). Повторные запросы приостановлены до восстановления соединения.</p>
            <Button variant="outline" size="sm" onClick={handleBackendRetry} className="flex items-center gap-2">
              <RefreshCw className="h-3 w-3" />
              Повторить попытку
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Индикатор ошибки, если есть кэшированные данные */}
      {error && statistics && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            <div className="flex items-center justify-between">
              <span>{error}</span>
              <Button
                variant="outline"
                size="sm"
                onClick={handleBackendRetry}
              >
                <RefreshCw className="h-3 w-3 mr-1" />
                Обновить
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      {/* Индикатор времени последнего обновления */}
      {lastFetchTime && (
        <div className="text-xs text-muted-foreground text-right">
          Обновлено: {new Date(lastFetchTime).toLocaleTimeString('ru-RU')}
        </div>
      )}

      {/* Общая статистика */}
      {statistics && (
        <div>
          <h3 className="text-lg font-semibold mb-4">Общая статистика</h3>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <StatCard
              title="Номенклатура"
              value={statistics.total_nomenclature || 0}
              description="всего записей"
              icon={Package}
              variant="primary"
            />
            <StatCard
              title="Контрагенты"
              value={statistics.total_counterparties || 0}
              description="всего записей"
              icon={Users}
              variant="success"
            />
            <StatCard
              title="Базы данных"
              value={statistics.total_databases || 0}
              description="всего БД"
              icon={Database}
              variant="default"
            />
            <StatCard
              title="Качество"
              value={`${Math.round((statistics.avg_quality_score || 0) * 100)}%`}
              description="среднее качество"
              variant={(statistics.avg_quality_score || 0) >= 0.9 ? 'success' : (statistics.avg_quality_score || 0) >= 0.7 ? 'warning' : 'danger'}
              progress={(statistics.avg_quality_score || 0) * 100}
            />
          </div>
          {statistics.duplicate_summary && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
            <StatCard
              title="Группы дубликатов"
              value={formatNumber(statistics.duplicate_summary.total_groups || 0)}
              description={`${formatNumber(statistics.duplicate_summary.total_records || 0)} записей`}
              icon={AlertCircle}
              variant={(statistics.duplicate_summary.total_groups || 0) > 0 ? 'warning' : 'success'}
            />
            <StatCard
              title="Многобазовые контрагенты"
              value={formatNumber(statistics.duplicate_summary.multi_database_counterparties || 0)}
              description="связаны с несколькими БД"
              icon={Database}
              variant={(statistics.duplicate_summary.multi_database_counterparties || 0) > 0 ? 'primary' : 'default'}
            />
          </div>
        )}
        </div>
      )}

      {/* Статистика по проектам */}
      {statistics && (
        <div>
          <h3 className="text-lg font-semibold mb-4">Статистика по проектам</h3>
          {statistics.projects && statistics.projects.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {statistics.projects.map((project, index) => {
              const cardKey = project.project_id ?? `${project.project_name || 'project'}-${index}`
              const qualityEntry = project.project_id ? projectQuality[project.project_id] : undefined
              return (
                <Card key={cardKey}>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center justify-between">
                      <span className="truncate">{project.project_name}</span>
                      <Badge variant={project.status === 'active' ? 'default' : 'secondary'}>
                        {project.status}
                      </Badge>
                    </CardTitle>
                    <CardDescription>
                      {project.project_type === 'nomenclature' ? 'Номенклатура' :
                       project.project_type === 'counterparties' ? 'Контрагенты' :
                       project.project_type === 'nomenclature_counterparties' ? 'Номенклатура + Контрагенты' :
                       project.project_type || 'Неизвестный тип'}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                  <div className="space-y-3">
                    <div className="space-y-3">
                      <div className="grid grid-cols-2 gap-3 text-sm">
                        <div className="space-y-1">
                          <div className="flex items-center gap-1.5 text-muted-foreground">
                            <Package className="h-3.5 w-3.5" />
                            <span>Номенклатура:</span>
                          </div>
                          <div className="font-semibold text-lg">{formatNumber(project.total_nomenclature || 0)}</div>
                        </div>
                        <div className="space-y-1">
                          <div className="flex items-center gap-1.5 text-muted-foreground">
                            <Users className="h-3.5 w-3.5" />
                            <span>Контрагенты:</span>
                          </div>
                          <div className="font-semibold text-lg">{formatNumber(project.total_counterparties || 0)}</div>
                        </div>
                        <div className="space-y-1">
                          <div className="flex items-center gap-1.5 text-muted-foreground">
                            <Database className="h-3.5 w-3.5" />
                            <span>Базы данных:</span>
                          </div>
                          <div className="font-semibold text-lg">{formatNumber(project.total_databases || 0)}</div>
                        </div>
                        <div className="space-y-1">
                          <div className="flex items-center gap-1.5 text-muted-foreground">
                            <Target className="h-3.5 w-3.5" />
                            <span>Качество:</span>
                          </div>
                          <div className="font-semibold text-lg">
                            {Math.round((project.avg_quality_score || 0) * 100)}%
                          </div>
                        </div>
                      </div>
                      {/* Прогресс-бар качества */}
                      <div className="space-y-1">
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-muted-foreground">Оценка качества</span>
                          <span className="font-medium">
                            {Math.round((project.avg_quality_score || 0) * 100)}%
                          </span>
                        </div>
                        <Progress 
                          value={(project.avg_quality_score || 0) * 100} 
                          className="h-2"
                        />
                        <div className="flex items-center justify-between text-xs text-muted-foreground">
                          <span>0%</span>
                          <span>100%</span>
                        </div>
                      </div>
                    </div>
                    {project.configs && Object.keys(project.configs).length > 0 && (
                      <div className="text-xs text-muted-foreground pt-2 border-t">
                        <div className="font-medium mb-1">Конфигурации 1С:</div>
                        <div className="flex flex-wrap gap-1">
                          {Object.entries(project.configs).map(([config, count]) => (
                            <Badge key={config} variant="outline" className="text-xs">
                              {config} ({count})
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                    {(project.duplicate_groups || project.multi_database_count) && (
                      <div className="text-xs text-muted-foreground pt-2 border-t space-y-1">
                        {project.duplicate_groups !== undefined && (
                          <div className="flex items-center justify-between">
                            <span>Группы дубликатов</span>
                            <span className="font-medium text-orange-600">
                              {formatNumber(project.duplicate_groups || 0)}
                            </span>
                          </div>
                        )}
                        {project.duplicates_count !== undefined && (
                          <div className="flex items-center justify-between">
                            <span>Записей-дубликатов</span>
                            <span className="font-medium">
                              {formatNumber(project.duplicates_count || 0)}
                            </span>
                          </div>
                        )}
                        {project.multi_database_count !== undefined && (
                          <div className="flex items-center justify-between">
                            <span>Из нескольких БД</span>
                            <span className="font-medium">
                              {formatNumber(project.multi_database_count || 0)}
                            </span>
                          </div>
                        )}
                      </div>
                    )}
                    {project.last_updated && (
                      <div className="text-xs text-muted-foreground pt-2 border-t">
                        Обновлено: {formatDate(project.last_updated)}
                      </div>
                    )}
                    <div className="text-xs text-muted-foreground pt-3 border-t space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="flex items-center gap-1 font-medium text-sm">
                          <Activity className="h-3.5 w-3.5" />
                          Метрики качества
                        </span>
                        {project.project_id ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => fetchProjectQualityStats(project.project_id)}
                            disabled={qualityEntry?.loading}
                          >
                            {qualityEntry?.loading ? (
                              <Loader2 className="h-3 w-3 animate-spin" />
                            ) : qualityEntry?.data ? 'Обновить' : 'Загрузить'}
                          </Button>
                        ) : null}
                      </div>
                      {qualityEntry?.error && (
                        <Alert variant="destructive">
                          <AlertDescription className="text-xs">
                            {qualityEntry.error}
                          </AlertDescription>
                        </Alert>
                      )}
                      {qualityEntry?.loading && (
                        <div className="h-16 rounded-md bg-muted/50 animate-pulse" />
                      )}
                      {qualityEntry?.data && (
                        <div className="space-y-2">
                          <div className="grid grid-cols-2 gap-3">
                            <div>
                              <span className="text-muted-foreground block">Записей</span>
                              <span className="font-semibold">
                                {formatNumber(qualityEntry.data.total_items || 0)}
                              </span>
                            </div>
                            <div>
                              <span className="text-muted-foreground block">Среднее качество</span>
                              <span className="font-semibold">
                                {Math.round((qualityEntry.data.average_quality || 0) * 100)}%
                              </span>
                            </div>
                            <div>
                              <span className="text-muted-foreground block">Эталонов</span>
                              <span className="font-semibold">
                                {formatNumber(qualityEntry.data.benchmark_count || 0)}
                              </span>
                            </div>
                            <div>
                              <span className="text-muted-foreground block">БД обработано</span>
                              <span className="font-semibold">
                                {qualityEntry.data.databases_processed ?? qualityEntry.data.databases_count ?? 0}
                              </span>
                            </div>
                          </div>
                          {qualityEntry.data.last_activity && (
                            <div className="text-[11px] text-muted-foreground">
                              Последняя активность: {formatDate(qualityEntry.data.last_activity)}
                            </div>
                          )}
                          {qualityEntry.data.by_level && (
                            <div className="flex flex-wrap gap-1">
                              {Object.entries(qualityEntry.data.by_level).map(([level, data]) => (
                                <Badge key={level} variant="outline" className="text-[11px]">
                                  {QUALITY_LEVEL_LABELS[level] || level}: {formatNumber(data.count || 0)}
                                </Badge>
                              ))}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        ) : (
          <Card>
            <CardContent className="py-8 text-center text-muted-foreground">
              Нет данных по проектам
            </CardContent>
          </Card>
        )}
        </div>
      )}

      {/* Непривязанные базы данных */}
      {statistics && statistics.unlinked_databases && statistics.unlinked_databases.length > 0 && (() => {
        // Группируем по конфигурациям
        const groups: Record<string, typeof statistics.unlinked_databases> = {}
        const noConfig: typeof statistics.unlinked_databases = []
        
        statistics.unlinked_databases.forEach(db => {
          const configName = db.display_name || db.config_name || 'Без конфигурации'
          if (configName === 'Без конфигурации') {
            noConfig.push(db)
          } else {
            if (!groups[configName]) {
              groups[configName] = []
            }
            groups[configName].push(db)
          }
        })
        
        const groupedByConfig = { groups, noConfig }
        const totalSize = statistics.unlinked_databases.reduce((sum, db) => sum + (db.size || 0), 0)

        return (
          <div>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">
                Непривязанные базы данных
              </h3>
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant="outline" className="bg-orange-50 dark:bg-orange-950">
                  {statistics.unlinked_databases_count || statistics.unlinked_databases.length} баз
                </Badge>
                {totalSize > 0 && (
                  <Badge variant="outline" className="bg-blue-50 dark:bg-blue-950">
                    {(totalSize / 1024 / 1024).toFixed(2)} MB
                  </Badge>
                )}
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handleAutoLink}
                  disabled={isAutoLinking}
                  className="ml-auto"
                >
                  {isAutoLinking ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Zap className="mr-2 h-4 w-4" />
                  )}
                  Авто-привязка
                </Button>
              </div>
            </div>
            <Alert variant="default" className="mb-4 border-orange-200 dark:border-orange-900 bg-orange-50/50 dark:bg-orange-950/20">
              <AlertCircle className="h-4 w-4 text-orange-600 dark:text-orange-400" />
              <AlertDescription className="text-sm">
                Эти базы данных не привязаны к проектам. Используйте автоматическую привязку во вкладке &quot;Базы данных&quot; для их привязки.
              </AlertDescription>
            </Alert>

            {/* Группировка по конфигурациям */}
            {Object.keys(groupedByConfig.groups).length > 0 && (
              <div className="space-y-4 mb-4">
                {Object.entries(groupedByConfig.groups).map(([configName, dbs]) => {
                  const configSize = dbs.reduce((sum, db) => sum + (db.size || 0), 0)
                  return (
                    <Card key={configName} className="border-blue-200 dark:border-blue-900">
                      <CardHeader>
                        <CardTitle className="text-base flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Settings className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                            <span>{configName}</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Badge variant="outline">{dbs.length} баз</Badge>
                            {configSize > 0 && (
                              <Badge variant="outline" className="text-xs">
                                {(configSize / 1024 / 1024).toFixed(2)} MB
                              </Badge>
                            )}
                          </div>
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                          {dbs.map((db, dbIndex) => (
                            <Card key={db.id ?? `db-${dbIndex}-${db.name ?? 'unknown'}`} className="border-orange-200 dark:border-orange-900">
                              <CardHeader className="pb-2">
                                <CardTitle className="text-sm flex items-center gap-2">
                                  <Database className="h-3.5 w-3.5 text-orange-600 dark:text-orange-400" />
                                  <span className="truncate">{db.name}</span>
                                </CardTitle>
                              </CardHeader>
                              <CardContent className="pt-0">
                                <div className="space-y-1.5 text-xs">
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <div className="cursor-help">
                                        <span className="text-muted-foreground">Путь:</span>
                                        <div className="font-mono text-[10px] truncate mt-0.5">{db.path}</div>
                                      </div>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      <p className="font-mono text-xs max-w-xs break-all">{db.path}</p>
                                    </TooltipContent>
                                  </Tooltip>
                                  {db.size && (
                                    <div>
                                      <span className="text-muted-foreground">Размер:</span>
                                      <div className="font-medium text-xs">
                                        {(db.size / 1024 / 1024).toFixed(2)} MB
                                      </div>
                                    </div>
                                  )}
                                </div>
                              </CardContent>
                            </Card>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  )
                })}
              </div>
            )}

            {/* Базы без конфигурации */}
            {groupedByConfig.noConfig.length > 0 && (
              <Card className="border-orange-200 dark:border-orange-900">
                <CardHeader>
                  <CardTitle className="text-base flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <AlertCircle className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                      <span>Без конфигурации</span>
                    </div>
                    <Badge variant="outline">{groupedByConfig.noConfig.length} баз</Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                    {groupedByConfig.noConfig.map((db, dbIndex) => (
                      <Card key={db.id ?? `no-config-db-${dbIndex}-${db.name ?? 'unknown'}`} className="border-orange-200 dark:border-orange-900">
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm flex items-center gap-2">
                            <Database className="h-3.5 w-3.5 text-orange-600 dark:text-orange-400" />
                            <span className="truncate">{db.name}</span>
                          </CardTitle>
                        </CardHeader>
                        <CardContent className="pt-0">
                          <div className="space-y-1.5 text-xs">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <div className="cursor-help">
                                  <span className="text-muted-foreground">Путь:</span>
                                  <div className="font-mono text-[10px] truncate mt-0.5">{db.path}</div>
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="font-mono text-xs max-w-xs break-all">{db.path}</p>
                              </TooltipContent>
                            </Tooltip>
                            {db.size && (
                              <div>
                                <span className="text-muted-foreground">Размер:</span>
                                <div className="font-medium text-xs">
                                  {(db.size / 1024 / 1024).toFixed(2)} MB
                                </div>
                              </div>
                            )}
                          </div>
                          <Button
                            variant="outline"
                            size="sm"
                            className="w-full mt-3"
                            onClick={() => handleOpenLinkDialog(db)}
                          >
                            <Link2 className="h-3.5 w-3.5 mr-1" />
                            Привязать
                          </Button>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )
      })()}
      <Dialog open={linkDialogOpen} onOpenChange={(open) => open ? setLinkDialogOpen(true) : handleCloseLinkDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Привязать базу данных</DialogTitle>
            {databaseToLink && (
              <DialogDescription>
                Выберите проект, к которому нужно привязать базу <span className="font-medium">&laquo;{databaseToLink.name}&raquo;</span>
              </DialogDescription>
            )}
          </DialogHeader>

          <div className="space-y-3">
            <div>
              <Label htmlFor="project-select">Проект</Label>
              <Select
                value={selectedLinkProjectId ? selectedLinkProjectId.toString() : ""}
                onValueChange={(value) => setSelectedLinkProjectId(Number(value))}
              >
                <SelectTrigger id="project-select">
                  <SelectValue placeholder="Выберите проект" />
                </SelectTrigger>
                <SelectContent>
                  {(statistics?.projects || []).map((project) => (
                    <SelectItem key={project.project_id} value={project.project_id.toString()}>
                      {project.project_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {databaseToLink && (
              <div className="text-xs text-muted-foreground">
                <div>Путь: <span className="font-mono break-all">{databaseToLink.path}</span></div>
                {databaseToLink.display_name && (
                  <div>Конфигурация: <span className="font-medium">{databaseToLink.display_name}</span></div>
                )}
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={handleCloseLinkDialog}>
              Отмена
            </Button>
            <Button
              onClick={handleConfirmLink}
              disabled={!selectedLinkProjectId || isLinking}
            >
              {isLinking ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Link2 className="mr-2 h-4 w-4" />
              )}
              Привязать
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

