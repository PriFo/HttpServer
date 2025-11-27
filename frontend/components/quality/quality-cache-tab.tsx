'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { Progress } from '@/components/ui/progress'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  RefreshCw,
  Trash2,
  Activity,
  Server,
  Database,
  Search,
  ClipboardList,
  Download,
  Clipboard,
  FileJson,
  Info,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'

interface ProjectQualityCacheEntry {
  key: string
  project_id?: number
  cached_at: string
  last_access?: string
  hit_count: number
  age_seconds: number
  expires_in_seconds: number
  is_expired: boolean
}

interface ProjectQualityCacheStats {
  total_entries: number
  valid_entries: number
  expired_entries: number
  ttl_seconds: number
  total_hits: number
  total_misses: number
  hit_rate: number
  entries: ProjectQualityCacheEntry[]
}

interface QualityCacheApiResponse {
  enabled: boolean
  message?: string
  stats?: ProjectQualityCacheStats
}

type ProjectQualityCacheEntrySort = 'age' | 'hit' | 'expires' | 'project'

const EXPIRING_THRESHOLD_SECONDS = 90
const CACHE_SETTINGS_KEY = 'quality_cache_settings'
const AUTO_REFRESH_OPTIONS = [15, 30, 60, 120, 300]

const formatDateTime = (value?: string) => {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleString('ru-RU')
  } catch {
    return value
  }
}

const formatSeconds = (seconds: number) => {
  if (seconds <= 0) return '0 сек'
  const mins = Math.floor(seconds / 60)
  const hrs = Math.floor(mins / 60)
  const remMins = mins % 60
  const remSecs = Math.floor(seconds % 60)
  if (hrs > 0) return `${hrs} ч ${remMins} мин`
  if (mins > 0) return `${mins} мин ${remSecs} сек`
  return `${remSecs} сек`
}

const formatRelativeTime = (value?: string) => {
  if (!value) return '—'
  try {
    return formatDistanceToNow(new Date(value), { addSuffix: true, locale: ru })
  } catch {
    return value
  }
}

export function QualityCacheTab() {
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [search, setSearch] = useState('')
  const [onlyActive, setOnlyActive] = useState(false)
  const [showExpiringSoon, setShowExpiringSoon] = useState(false)
  const [sortBy, setSortBy] = useState<ProjectQualityCacheEntrySort>('age')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc')
  const [lastUpdatedLabel, setLastUpdatedLabel] = useState<string>('—')
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [autoRefreshInterval, setAutoRefreshInterval] = useState(30)
  const [exportingFormat, setExportingFormat] = useState<'csv' | 'json' | null>(null)
  const [isCopyingSummary, setIsCopyingSummary] = useState(false)
  const [selectedEntry, setSelectedEntry] = useState<ProjectQualityCacheEntry | null>(null)
  const [isCopyingEntry, setIsCopyingEntry] = useState(false)
  const [isCopyingKeys, setIsCopyingKeys] = useState(false)
  const [settingsLoaded, setSettingsLoaded] = useState(false)
  const [nextRefreshIn, setNextRefreshIn] = useState<number | null>(null)

  const {
    data: cacheData,
    loading,
    error,
    refetch,
    lastUpdated,
  } = useProjectState<QualityCacheApiResponse | null>(
    async (_cid, _pid, signal) => {
      const response = await fetch('/api/quality/cache/stats', {
        cache: 'no-store',
        signal,
      })
      
      // Для 404 возвращаем пустые данные вместо ошибки
      if (response.status === 404) {
        return {
          enabled: false,
          message: 'Кэш качества не настроен',
          stats: undefined,
        } as QualityCacheApiResponse
      }
      
      const payload: QualityCacheApiResponse | { error?: string } = await response.json().catch(() => ({}))
      if (!response.ok) {
        throw new Error((payload as { error?: string }).error || 'Не удалось загрузить статистику кэша')
      }
      return payload as QualityCacheApiResponse
    },
    'quality',
    'cache',
    [],
    {
      refetchInterval: autoRefresh ? autoRefreshInterval * 1000 : null,
      keepPreviousData: true,
    }
  )

  const cacheEnabled = cacheData?.enabled !== false
  const cacheMessage = cacheData?.message ?? (cacheEnabled ? null : 'Кэш качества отключён на сервере')
  const stats = cacheData?.stats ?? null
  const isInitialLoading = loading && !stats
  useEffect(() => {
    if (lastUpdated) {
      setLastUpdatedLabel(new Date(lastUpdated).toLocaleString('ru-RU'))
    } else {
      setLastUpdatedLabel('—')
    }
  }, [lastUpdated])

  useEffect(() => {
    try {
      const savedRaw = typeof window !== 'undefined' ? localStorage.getItem(CACHE_SETTINGS_KEY) : null
      if (!savedRaw) {
        setSettingsLoaded(true)
        return
      }
      const saved = JSON.parse(savedRaw)
      if (typeof saved.search === 'string') setSearch(saved.search)
      if (typeof saved.onlyActive === 'boolean') setOnlyActive(saved.onlyActive)
      if (typeof saved.showExpiringSoon === 'boolean') setShowExpiringSoon(saved.showExpiringSoon)
      if (saved.sortBy) setSortBy(saved.sortBy)
      if (saved.sortDirection) setSortDirection(saved.sortDirection)
      if (typeof saved.autoRefresh === 'boolean') setAutoRefresh(saved.autoRefresh)
      if (typeof saved.autoRefreshInterval === 'number') setAutoRefreshInterval(saved.autoRefreshInterval)
    } catch (error) {
      console.warn('Failed to load cache settings', error)
    } finally {
      setSettingsLoaded(true)
    }
  }, [])

  useEffect(() => {
    if (!settingsLoaded) return
    try {
      const payload = {
        search,
        onlyActive,
        showExpiringSoon,
        sortBy,
        sortDirection,
        autoRefresh,
      }
      localStorage.setItem(CACHE_SETTINGS_KEY, JSON.stringify({
        ...payload,
        autoRefreshInterval,
      }))
    } catch (error) {
      console.warn('Failed to persist cache settings', error)
    }
  }, [search, onlyActive, showExpiringSoon, sortBy, sortDirection, autoRefresh, autoRefreshInterval, settingsLoaded])

  useEffect(() => {
    if (!autoRefresh) {
      setNextRefreshIn(null)
      return
    }
    setNextRefreshIn(autoRefreshInterval)

    const timer = setInterval(() => {
      setNextRefreshIn((prev) => {
        if (prev === null) {
          return autoRefreshInterval
        }
        if (prev <= 1) {
          refetch()
          return autoRefreshInterval
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [autoRefresh, autoRefreshInterval, refetch])

  useEffect(() => {
    if (error) {
      toast.error('Ошибка загрузки кэша', { description: error })
    }
  }, [error])

  const handleManualRefresh = useCallback(async () => {
    setIsRefreshing(true)
    try {
      await refetch()
      if (autoRefresh) {
        setNextRefreshIn(autoRefreshInterval)
      }
    } finally {
      setIsRefreshing(false)
    }
  }, [refetch, autoRefresh, autoRefreshInterval])

  const invalidateCache = async (projectId?: number) => {
    try {
      const url = projectId
        ? `/api/quality/cache/invalidate?project_id=${projectId}`
        : '/api/quality/cache/invalidate'
      const response = await fetch(url, {
        method: 'POST',
        cache: 'no-store',
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        throw new Error(data.error || 'Не удалось инвалидировать кэш')
      }
      toast.success(projectId ? `Кэш проекта ${projectId} сброшен` : 'Кэш качества очищен')
      await refetch()
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Не удалось выполнить операцию'
      toast.error('Ошибка инвалидации', { description: message })
    }
  }

  const handleExport = async (format: 'csv' | 'json') => {
    if (!processedEntries.length) {
      toast.error('Нет данных для экспорта', {
        description: 'Сначала загрузите статистику и примените нужные фильтры.',
      })
      return
    }

    try {
      setExportingFormat(format)
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5)

      if (format === 'csv') {
        const headers = [
          'project_id',
          'key',
          'cached_at',
          'last_access',
          'hit_count',
          'age_seconds',
          'expires_in_seconds',
          'status',
        ]
        const rows = processedEntries.map((entry) => [
          entry.project_id ?? '',
          entry.key,
          entry.cached_at,
          entry.last_access ?? '',
          entry.hit_count,
          entry.age_seconds,
          entry.expires_in_seconds,
          entry.is_expired ? 'expired' : 'active',
        ])
        const csvContent = '\uFEFF' + [headers, ...rows].map((row) => row.map((cell) => `"${cell}"`).join(',')).join('\n')
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
        const link = document.createElement('a')
        link.href = URL.createObjectURL(blob)
        link.download = `quality_cache_entries_${timestamp}.csv`
        link.click()
        toast.success('Экспорт завершён', {
          description: `Вы выгрузили ${processedEntries.length} записей`,
        })
        return
      }

      const payload = {
        generated_at: new Date().toISOString(),
        cache_enabled: cacheEnabled,
        totals: stats && {
          total_entries: stats.total_entries,
          valid_entries: stats.valid_entries,
          expired_entries: stats.expired_entries,
          hit_rate: hitRatePercent,
          ttl_seconds: stats.ttl_seconds,
        },
        filters: {
          search: search || null,
          only_active: onlyActive,
          show_expiring: showExpiringSoon,
          sort_by: sortBy,
          sort_direction: sortDirection,
        },
        entries: processedEntries,
      }

      const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = `quality_cache_entries_${timestamp}.json`
      link.click()
      toast.success('JSON экспортирован', {
        description: `Сохранено ${processedEntries.length} записей`,
      })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Не удалось выполнить экспорт'
      toast.error('Ошибка экспорта', { description: message })
    } finally {
      setExportingFormat(null)
    }
  }

  const handleCopySummary = async () => {
    if (!stats) {
      toast.error('Нет данных для копирования', {
        description: 'Статистика ещё не загружена',
      })
      return
    }

    try {
      setIsCopyingSummary(true)
      const summaryLines = [
        `Состояние кэша: ${cacheEnabled ? 'активен' : 'отключён'}`,
        `Всего записей: ${stats.total_entries}`,
        `Активных: ${stats.valid_entries}`,
        `Истёкших: ${stats.expired_entries}`,
        `Hit rate: ${hitRatePercent}% (${stats.total_hits} hits / ${stats.total_misses} misses)`,
        `TTL: ${formatSeconds(stats.ttl_seconds)}`,
        `Автообновление: ${autoRefresh ? `${autoRefreshInterval} секунд` : 'выключено'}`,
        autoRefresh ? `Следующее обновление через: ${nextRefreshIn ?? autoRefreshInterval} сек` : null,
        `Риск истечения (<= ${formatSeconds(EXPIRING_THRESHOLD_SECONDS)}): ${expiringSoonCount}`,
        `Последнее обновление: ${lastUpdatedLabel}`,
        `Применённые фильтры: ${
          [
            onlyActive && 'только активные',
            showExpiringSoon && 'скоро истекают',
            search && `поиск="${search}"`,
          ]
            .filter(Boolean)
            .join(', ') || 'нет'
        }`,
        `Сортировка: ${sortBy} (${sortDirection === 'asc' ? 'по возрастанию' : 'по убыванию'})`,
        `Видимых записей: ${processedEntries.length}`,
      ]

      if (topHitEntries.length > 0) {
        summaryLines.push('', 'Топ записей:')
        topHitEntries.forEach((entry, idx) => {
          summaryLines.push(
            `${idx + 1}. project=${entry.project_id ?? '—'} key=${entry.key} hits=${entry.hit_count} ttl=${formatSeconds(entry.expires_in_seconds)}`
          )
        })
      }

      const filteredSummaryLines = summaryLines.filter(Boolean) as string[]
      await navigator.clipboard.writeText(filteredSummaryLines.join('\n'))
      toast.success('Сводка скопирована', {
        description: 'Данные по кэшу доступны в буфере обмена',
      })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Не удалось скопировать текст'
      toast.error('Ошибка копирования', { description: message })
    } finally {
      setIsCopyingSummary(false)
    }
  }

  const handleCopyEntryJson = async (entry: ProjectQualityCacheEntry | null) => {
    if (!entry) return
    try {
      setIsCopyingEntry(true)
      await navigator.clipboard.writeText(JSON.stringify(entry, null, 2))
      toast.success('JSON записи скопирован')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Не удалось скопировать JSON'
      toast.error('Ошибка копирования', { description: message })
    } finally {
      setIsCopyingEntry(false)
    }
  }

  const handleCopyVisibleKeys = async () => {
    if (!processedEntries.length) {
      toast.error('Нет записей для копирования', {
        description: 'Измените фильтры или обновите данные',
      })
      return
    }

    try {
      setIsCopyingKeys(true)
      const lines = processedEntries.map((entry) => {
        const parts = [
          `project:${entry.project_id ?? '—'}`,
          `key:${entry.key}`,
          `status:${entry.is_expired ? 'expired' : 'active'}`,
          `ttl:${entry.is_expired ? '0' : entry.expires_in_seconds}s`,
        ]
        return parts.join(' | ')
      })
      await navigator.clipboard.writeText(lines.join('\n'))
      toast.success('Список скопирован', {
        description: `Скопировано ${processedEntries.length} записей`,
      })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Не удалось скопировать список'
      toast.error('Ошибка копирования', { description: message })
    } finally {
      setIsCopyingKeys(false)
    }
  }

  const processedEntries = useMemo(() => {
    if (!stats?.entries) return []
    const query = search.trim().toLowerCase()
    const filtered = stats.entries.filter((entry) => {
      if (onlyActive && entry.is_expired) return false
      if (showExpiringSoon && (entry.is_expired || entry.expires_in_seconds > EXPIRING_THRESHOLD_SECONDS)) {
        return false
      }
      if (!query) return true
      const text = `${entry.key} ${entry.project_id ?? ''}`
      return text.toLowerCase().includes(query)
    })

    const sorted = [...filtered].sort((a, b) => {
      const direction = sortDirection === 'asc' ? 1 : -1
      const getValue = (entry: ProjectQualityCacheEntry) => {
        switch (sortBy) {
          case 'hit':
            return entry.hit_count
          case 'expires':
            return entry.expires_in_seconds
          case 'project':
            return entry.project_id ?? 0
          case 'age':
          default:
            return entry.age_seconds
        }
      }

      const diff = getValue(a) - getValue(b)
      if (diff === 0) {
        return a.key.localeCompare(b.key) * direction
      }
      return diff * direction
    })

    return sorted
  }, [stats, search, onlyActive, showExpiringSoon, sortBy, sortDirection])

  const hitRatePercent = stats ? Math.round((stats.hit_rate || 0) * 100) : 0
  const expiringSoonCount = stats?.entries
    ? stats.entries.filter((entry) => !entry.is_expired && entry.expires_in_seconds <= EXPIRING_THRESHOLD_SECONDS).length
    : 0
  const topHitEntries = useMemo(() => {
    if (!stats?.entries) return []
    return [...stats.entries]
      .sort((a, b) => b.hit_count - a.hit_count || a.key.localeCompare(b.key))
      .slice(0, 5)
  }, [stats])
  const isExporting = exportingFormat !== null
  const isExportingCsv = exportingFormat === 'csv'
  const isExportingJson = exportingFormat === 'json'
  const nextRefreshLabel = autoRefresh ? nextRefreshIn ?? autoRefreshInterval : null
  const noEntriesMessage = (() => {
    if (!stats) return 'Данные ещё не загружены'
    if (showExpiringSoon) return 'Нет записей, которые истекут в ближайшее время'
    if (onlyActive) return 'Нет активных записей кэша'
    if (search.trim()) return 'По запросу ничего не найдено'
    return 'Нет записей кэша'
  })()

  if (isInitialLoading) {
    return <LoadingState message="Загрузка статистики кэша качества..." />
  }

  if (error && !stats) {
    return (
      <ErrorState
        title="Ошибка загрузки кэша"
        message={error}
        action={{
          label: 'Повторить',
          onClick: handleManualRefresh,
        }}
        variant="destructive"
        className="mt-4"
      />
    )
  }

  return (
    <>
      <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="text-lg font-semibold">Кэш статистики качества проектов</h2>
          <p className="text-sm text-muted-foreground">
            Мониторинг состояния кэша агрегированных показателей качества и быстрая инвалидация.
          </p>
          <p className="text-xs text-muted-foreground mt-1">
            Последнее обновление: {lastUpdatedLabel}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-2">
            <Button variant={autoRefresh ? 'default' : 'outline'} size="sm" onClick={() => setAutoRefresh(!autoRefresh)}>
              <Activity className={`h-4 w-4 mr-2 ${autoRefresh ? 'animate-pulse' : ''}`} />
              Автообновление
            </Button>
            <Select
              value={String(autoRefreshInterval)}
              onValueChange={(value) => setAutoRefreshInterval(Number(value))}
              disabled={!autoRefresh}
            >
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder="Интервал" />
              </SelectTrigger>
              <SelectContent>
                {AUTO_REFRESH_OPTIONS.map((option) => (
                  <SelectItem key={option} value={String(option)}>
                    {option} сек
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {autoRefresh && (
            <span className="text-xs text-muted-foreground">
              Следующее обновление через:{' '}
              <span className="font-medium text-primary">{nextRefreshLabel} сек</span>
            </span>
          )}
          <Button variant="outline" size="sm" onClick={handleCopySummary} disabled={!stats || isCopyingSummary}>
            <Clipboard className="h-4 w-4 mr-2" />
            Скопировать сводку
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleCopyVisibleKeys}
            disabled={processedEntries.length === 0 || isCopyingKeys}
          >
            <ClipboardList className={`h-4 w-4 mr-2 ${isCopyingKeys ? 'animate-spin' : ''}`} />
            Скопировать ключи
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport('csv')}
            disabled={processedEntries.length === 0 || isExporting}
          >
            <Download className={`h-4 w-4 mr-2 ${isExportingCsv ? 'animate-spin' : ''}`} />
            Экспорт CSV
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleExport('json')}
            disabled={processedEntries.length === 0 || isExporting}
          >
            <FileJson className={`h-4 w-4 mr-2 ${isExportingJson ? 'animate-spin' : ''}`} />
            Экспорт JSON
          </Button>
          <Button variant="outline" size="sm" onClick={() => invalidateCache()} disabled={isRefreshing || !cacheEnabled}>
            <Trash2 className="h-4 w-4 mr-2" />
            Очистить кэш
          </Button>
          <Button variant="default" size="sm" onClick={handleManualRefresh} disabled={isRefreshing}>
            <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? 'animate-spin' : ''}`} />
            Обновить
          </Button>
        </div>
      </div>

      {!cacheEnabled && (
        <Alert variant="destructive">
          <AlertDescription>
            {cacheMessage || 'Кэш качества отключён на сервере. Отображаются последние доступные данные (если они были).'}
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Состояние кэша</CardDescription>
            <CardTitle className="text-2xl flex items-center gap-2">
              <span>{cacheEnabled ? 'Активен' : 'Отключён'}</span>
              <Badge variant={cacheEnabled ? 'secondary' : 'destructive'}>
                {cacheEnabled ? 'OK' : 'OFF'}
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground space-y-1">
            <div>TTL: {stats ? formatSeconds(stats.ttl_seconds) : '—'}</div>
            <div>Обновлено: {lastUpdatedLabel}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Записей в кэше</CardDescription>
            <CardTitle className="text-2xl">{stats?.total_entries ?? 0}</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground space-y-1">
            <div>Активных: {stats?.valid_entries ?? 0}</div>
            <div>Истёкших: {stats?.expired_entries ?? 0}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Hit Rate</CardDescription>
            <CardTitle className="text-2xl">{hitRatePercent}%</CardTitle>
          </CardHeader>
          <CardContent>
            <Progress value={hitRatePercent} />
            <p className="text-xs text-muted-foreground mt-2">
              {(stats?.total_hits ?? 0)} hits / {(stats?.total_misses ?? 0)} misses
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Риск истечения</CardDescription>
            <CardTitle className="text-2xl">{expiringSoonCount}</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground space-y-1">
            <div>Порог: {formatSeconds(EXPIRING_THRESHOLD_SECONDS)}</div>
            <div>Истекших: {stats?.expired_entries ?? 0}</div>
          </CardContent>
        </Card>
      </div>

      {topHitEntries.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle>Самые востребованные записи</CardTitle>
            <CardDescription>Топ-5 проектов по количеству обращений за TTL</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {topHitEntries.map((entry) => (
              <div key={entry.key} className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-xs">
                    {entry.project_id ?? '—'}
                  </Badge>
                  <span className="font-mono text-xs text-muted-foreground">{entry.key}</span>
                </div>
                <div className="text-muted-foreground">
                  {entry.hit_count} запросов · остаётся {formatSeconds(entry.expires_in_seconds)}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="pb-3 gap-2">
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle>Записи кэша</CardTitle>
              <CardDescription>Список всех проектов, статистика которых закэширована.</CardDescription>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <div className="relative">
                <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Поиск по ключу или project_id..."
                  className="pl-9 w-[220px]"
                />
              </div>
              <Button
                variant={onlyActive ? 'default' : 'outline'}
                size="sm"
                onClick={() => setOnlyActive(!onlyActive)}
              >
                <Server className="h-4 w-4 mr-2" />
                {onlyActive ? 'Только активные' : 'Все записи'}
              </Button>
              <Button
                variant={showExpiringSoon ? 'default' : 'outline'}
                size="sm"
                onClick={() => setShowExpiringSoon(!showExpiringSoon)}
              >
                {showExpiringSoon ? 'До конца TTL' : 'Скоро истекают'}
              </Button>
              <Select value={sortBy} onValueChange={(value) => setSortBy(value as ProjectQualityCacheEntrySort)}>
                <SelectTrigger className="w-[160px]">
                  <SelectValue placeholder="Сортировка" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="age">По возрасту</SelectItem>
                  <SelectItem value="hit">По hit count</SelectItem>
                  <SelectItem value="expires">По TTL</SelectItem>
                  <SelectItem value="project">По project_id</SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')}
              >
                {sortDirection === 'asc' ? 'По возрастанию' : 'По убыванию'}
              </Button>
              <Badge variant="outline" className="text-xs">
                {processedEntries.length} записей
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && stats && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Проект</TableHead>
                  <TableHead>Ключ</TableHead>
                  <TableHead>Сохранено</TableHead>
                  <TableHead>Последний доступ</TableHead>
                  <TableHead className="text-right">Hit count</TableHead>
                  <TableHead className="text-right">Возраст</TableHead>
                  <TableHead className="text-right">До истечения</TableHead>
                  <TableHead className="text-right">Действия</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {processedEntries.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center text-sm text-muted-foreground py-8">
                      {noEntriesMessage}
                    </TableCell>
                  </TableRow>
                )}
                {processedEntries.map((entry) => {
                  const percent = stats ? ((stats.ttl_seconds - entry.expires_in_seconds) / stats.ttl_seconds) * 100 : 0
                  return (
                    <TableRow key={entry.key} className={entry.is_expired ? 'bg-destructive/5' : undefined}>
                      <TableCell>
                        {entry.project_id ? (
                          <Badge variant="outline" className="text-xs">
                            <Database className="h-3 w-3 mr-1" />
                            {entry.project_id}
                          </Badge>
                        ) : (
                          <span className="text-muted-foreground text-xs">—</span>
                        )}
                      </TableCell>
                      <TableCell className="font-mono text-xs">{entry.key}</TableCell>
                      <TableCell className="text-sm">{formatDateTime(entry.cached_at)}</TableCell>
                      <TableCell className="text-sm">{formatDateTime(entry.last_access)}</TableCell>
                      <TableCell className="text-right text-sm">{entry.hit_count}</TableCell>
                      <TableCell className="text-right text-sm">{formatSeconds(entry.age_seconds)}</TableCell>
                      <TableCell className="text-right text-sm">
                        {entry.is_expired ? (
                          <Badge variant="destructive" className="text-xs">Истекло</Badge>
                        ) : (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <div>
                                  <Progress value={percent} className="h-2 mb-1" />
                                  <div className="text-xs">{formatSeconds(entry.expires_in_seconds)}</div>
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>Обновите статистику проекта, если требуется актуальные данные</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="icon"
                                  onClick={() => invalidateCache(entry.project_id)}
                                  disabled={!entry.project_id}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>Инвалидировать только этот проект</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="icon"
                                  onClick={() => {
                                    if (!entry.project_id) {
                                      navigator.clipboard.writeText(entry.key)
                                      toast.success('Ключ скопирован')
                                      return
                                    }
                                    navigator.clipboard.writeText(String(entry.project_id))
                                    toast.success('ID проекта скопирован')
                                  }}
                                >
                                  <ClipboardList className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>Скопировать ключ/ID проекта</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="icon"
                                  onClick={() => setSelectedEntry(entry)}
                                >
                                  <Info className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>Подробнее о записи</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
      </div>
      <Dialog
        open={!!selectedEntry}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedEntry(null)
          }
        }}
      >
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Детали записи кэша</DialogTitle>
            <DialogDescription>
              Проект {selectedEntry?.project_id ?? '—'} · ключ {selectedEntry?.key}
            </DialogDescription>
          </DialogHeader>
          {selectedEntry && (
            <div className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
                <div>
                  <p className="text-muted-foreground">Сохранено</p>
                  <p className="font-medium">{formatDateTime(selectedEntry.cached_at)}</p>
                  <p className="text-xs text-muted-foreground">{formatRelativeTime(selectedEntry.cached_at)}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Последний доступ</p>
                  <p className="font-medium">{formatDateTime(selectedEntry.last_access)}</p>
                  <p className="text-xs text-muted-foreground">{formatRelativeTime(selectedEntry.last_access)}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Hit count</p>
                  <p className="font-medium">{selectedEntry.hit_count}</p>
                </div>
                <div>
                  <p className="text-muted-foreground">Оставшееся время</p>
                  {selectedEntry.is_expired ? (
                    <p className="font-medium text-destructive">Истекло</p>
                  ) : (
                    <p className="font-medium">{formatSeconds(selectedEntry.expires_in_seconds)}</p>
                  )}
                  {!selectedEntry.is_expired && (
                    <p className="text-xs text-muted-foreground">
                      Возраст: {formatSeconds(selectedEntry.age_seconds)}
                    </p>
                  )}
                </div>
              </div>

              <div className="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleCopyEntryJson(selectedEntry)}
                  disabled={isCopyingEntry}
                >
                  <Clipboard className={`h-4 w-4 mr-2 ${isCopyingEntry ? 'animate-spin' : ''}`} />
                  Скопировать JSON
                </Button>
                {selectedEntry.project_id && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      navigator.clipboard.writeText(String(selectedEntry.project_id))
                      toast.success('ID проекта скопирован')
                    }}
                  >
                    <ClipboardList className="h-4 w-4 mr-2" />
                    Скопировать project_id
                  </Button>
                )}
              </div>

              <ScrollArea className="max-h-72 rounded-md border bg-muted/40 p-3">
                <pre className="text-xs font-mono whitespace-pre-wrap">
                  {JSON.stringify(selectedEntry, null, 2)}
                </pre>
              </ScrollArea>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}


