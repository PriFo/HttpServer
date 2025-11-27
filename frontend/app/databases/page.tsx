'use client'

import { useState, useEffect, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Database, HardDrive, Calendar, CheckCircle2, AlertCircle, RefreshCw, BarChart3, Activity, TrendingUp, FileText, Layers, ArrowUpDown, Search, X, Folder, FolderOpen, List, Scan } from 'lucide-react'
import { toast } from 'sonner'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { StatCard } from '@/components/common/stat-card'
import { DatabasesPageSkeleton } from '@/components/common/database-skeleton'
import { Skeleton } from '@/components/ui/skeleton'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useRouter } from 'next/navigation'
import { DatabaseTypeBadge } from '@/components/database-type-badge'
import { DatabaseAnalyticsDialog } from '@/components/database-analytics-dialog'
import { formatDateTime, formatNumber } from '@/lib/locale'
import { FadeIn } from '@/components/animations/fade-in'
import { StaggerContainer, StaggerItem } from '@/components/animations/stagger-container'
import { motion, AnimatePresence } from 'framer-motion'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useApiClient } from '@/hooks/useApiClient'

interface DatabaseInfo {
  name: string
  path: string
  size: number
  modified_at: string
  is_current?: boolean
  type?: string
  table_count?: number
  total_rows?: number
  stats?: {
    total_uploads?: number
    uploads_count?: number
    total_catalogs?: number
    catalogs_count?: number
    total_items?: number
    items_count?: number
    last_upload_date?: string
    avg_items_per_upload?: number
  }
}

interface CurrentDatabaseInfo extends DatabaseInfo {
  status: string
  upload_stats?: {
    total_uploads?: number
    uploads_count?: number
    total_catalogs?: number
    catalogs_count?: number
    total_items?: number
    items_count?: number
    last_upload_date?: string
    avg_items_per_upload?: number
  }
}

export default function DatabasesPage() {
  const router = useRouter()
  const { get, post } = useApiClient()
  const [currentDB, setCurrentDB] = useState<CurrentDatabaseInfo | null>(null)
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [switching, setSwitching] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedDB, setSelectedDB] = useState<string | null>(null)
  const [showConfirmDialog, setShowConfirmDialog] = useState(false)
  const [showAnalyticsDialog, setShowAnalyticsDialog] = useState(false)
  const [analyticsDB, setAnalyticsDB] = useState<{ name: string; path: string } | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortBy, setSortBy] = useState<'name' | 'size' | 'date' | 'rows'>('name')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')
  const [filterType, setFilterType] = useState<string>('all')
  const [groupByDirectory, setGroupByDirectory] = useState(false)
  const [expandedDirectories, setExpandedDirectories] = useState<Set<string>>(new Set())
  const [scanning, setScanning] = useState(false)
  const [aggregatedStats, setAggregatedStats] = useState<{
    total_uploads?: number
    uploads_count?: number
    total_catalogs?: number
    catalogs_count?: number
    total_items?: number
    items_count?: number
    databases_with_uploads?: number
    last_upload_date?: string
    avg_items_per_upload?: number
  } | null>(null)

  const fetchData = async () => {
    setLoading(true)
    setError(null)

    try {
      // Fetch current database info
      try {
        const infoData = await get<CurrentDatabaseInfo>('/api/database/info', { skipErrorHandler: true })
        // Устанавливаем currentDB только если есть имя или путь
        if (infoData && (infoData.name || infoData.path)) {
          setCurrentDB(infoData)
        } else {
          // Если данных нет или они пустые, не устанавливаем currentDB
          setCurrentDB(null)
        }
      } catch (infoError) {
        // Если ошибка при получении информации о БД, просто не устанавливаем currentDB
        console.warn('Could not fetch current database info:', infoError)
        setCurrentDB(null)
      }

      // Fetch list of databases
      const listData = await get<{ databases: DatabaseInfo[], aggregated_stats?: {
        total_uploads?: number
        uploads_count?: number
        total_catalogs?: number
        catalogs_count?: number
        total_items?: number
        items_count?: number
        databases_with_uploads?: number
        last_upload_date?: string
        avg_items_per_upload?: number
      } }>('/api/databases/list', { skipErrorHandler: true })
      setDatabases(listData.databases || [])
      
      // Сохраняем агрегированную статистику, если она доступна
      if (listData.aggregated_stats) {
        setAggregatedStats(listData.aggregated_stats)
      } else {
        setAggregatedStats(null)
      }
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      setError(err instanceof Error ? err.message : 'Unknown error occurred')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleSwitchDatabase = async () => {
    if (!selectedDB) return

    setSwitching(true)
    setError(null)

    try {
      await post('/api/database/switch', { path: selectedDB }, { skipErrorHandler: true })

      // Refresh data after successful switch
      await fetchData()
      setShowConfirmDialog(false)
      setSelectedDB(null)

      toast.success('База данных переключена', {
        description: `Активная база данных: ${selectedDB.split(/[/\\]/).pop()}`,
      })

      // Redirect to home page
      router.push('/')
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      const errorMessage = err instanceof Error ? err.message : 'Failed to switch database'
      setError(errorMessage)
    } finally {
      setSwitching(false)
    }
  }

  const handleScanDatabases = async () => {
    setScanning(true)
    setError(null)

    try {
      const data = await post<{ found_files?: number }>('/api/databases/scan', { paths: ['.', 'data/uploads', 'data'] }, { skipErrorHandler: true })
      const foundCount = data.found_files || 0

      if (foundCount > 0) {
        toast.success('Сканирование завершено', {
          description: `Найдено ${foundCount} ${foundCount === 1 ? 'файл' : foundCount < 5 ? 'файла' : 'файлов'}`,
        })
      } else {
        toast.info('Сканирование завершено', {
          description: 'Новых файлов не найдено',
        })
      }

      // Обновляем список баз данных
      await fetchData()
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      const errorMessage = err instanceof Error ? err.message : 'Failed to scan databases'
      setError(errorMessage)
    } finally {
      setScanning(false)
    }
  }

  const formatFileSize = (bytes: number | null | undefined) => {
    if (bytes == null || bytes === 0) return '0 B'
    if (isNaN(bytes) || bytes < 0) return 'Неизвестно'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  // Используем formatDateTime из lib/locale для единообразия
  const formatDate = (dateString: string) => {
    if (!dateString) return 'Неизвестно'
    return formatDateTime(dateString, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  // Функция для получения директории из пути
  const getDirectory = (path: string): string => {
    // Убираем имя файла, оставляем только директорию
    const lastSlash = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
    if (lastSlash === -1) return '.'
    return path.substring(0, lastSlash) || '.'
  }

  // Функция для форматирования пути (показываем только директорию, если она не текущая)
  const formatPath = (path: string): string => {
    const dir = getDirectory(path)
    if (dir === '.' || dir === '') return 'Текущая директория'
    return dir
  }

  // Фильтрация и сортировка баз данных
  const filteredAndSortedDatabases = useMemo(() => {
    if (!databases || databases.length === 0) {
      return []
    }
    
    return databases
      .filter((db) => {
        if (!db) return false
        // Фильтр по поисковому запросу
        if (searchQuery) {
          const query = searchQuery.toLowerCase()
          if (!db.name?.toLowerCase().includes(query) && !db.path?.toLowerCase().includes(query)) {
            return false
          }
        }
        // Фильтр по типу
        if (filterType !== 'all' && db.type !== filterType) {
          return false
        }
        return true
      })
      .sort((a, b) => {
        let comparison = 0
        
        switch (sortBy) {
          case 'name':
            comparison = (a.name || '').localeCompare(b.name || '', 'ru')
            break
          case 'size':
            comparison = (a.size || 0) - (b.size || 0)
            break
          case 'date':
            comparison = new Date(a.modified_at || 0).getTime() - new Date(b.modified_at || 0).getTime()
            break
          case 'rows':
            comparison = (a.total_rows || 0) - (b.total_rows || 0)
            break
        }
        
        return sortOrder === 'asc' ? comparison : -comparison
      })
  }, [databases, searchQuery, filterType, sortBy, sortOrder])

  // Группировка по директориям
  const { groupedByDirectory, sortedDirectories } = useMemo(() => {
    if (!filteredAndSortedDatabases || filteredAndSortedDatabases.length === 0) {
      return { groupedByDirectory: {} as Record<string, DatabaseInfo[]>, sortedDirectories: [] as string[] }
    }

    const getDir = (path: string): string => {
      if (!path) return '.'
      const lastSlash = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
      if (lastSlash === -1) return '.'
      return path.substring(0, lastSlash) || '.'
    }

    const grouped = filteredAndSortedDatabases.reduce((acc, db) => {
      if (!db || !db.path) return acc
      const dir = getDir(db.path)
      if (!acc[dir]) {
        acc[dir] = []
      }
      acc[dir].push(db)
      return acc
    }, {} as Record<string, DatabaseInfo[]>)

    const sorted = Object.keys(grouped).sort((a, b) => {
      if (a === '.' || a === '') return -1
      if (b === '.' || b === '') return 1
      return a.localeCompare(b, 'ru')
    })

    return { groupedByDirectory: grouped, sortedDirectories: sorted }
  }, [filteredAndSortedDatabases])

  // Статистика по директориям
  const directoryStats = useMemo(() => {
    if (!filteredAndSortedDatabases || filteredAndSortedDatabases.length === 0) {
      return {} as Record<string, { count: number; totalSize: number; totalRows: number }>
    }

    const getDir = (path: string): string => {
      if (!path) return '.'
      const lastSlash = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
      if (lastSlash === -1) return '.'
      return path.substring(0, lastSlash) || '.'
    }

    const stats: Record<string, { count: number; totalSize: number; totalRows: number }> = {}
    filteredAndSortedDatabases.forEach(db => {
      if (!db || !db.path) return
      const dir = getDir(db.path)
      if (!stats[dir]) {
        stats[dir] = { count: 0, totalSize: 0, totalRows: 0 }
      }
      stats[dir].count++
      stats[dir].totalSize += (db.size || 0)
      stats[dir].totalRows += (db.total_rows || 0)
    })
    return stats
  }, [filteredAndSortedDatabases])

  // Автоматически раскрываем директории при группировке
  useEffect(() => {
    if (groupByDirectory && sortedDirectories && sortedDirectories.length > 0) {
      setExpandedDirectories(new Set(sortedDirectories))
    }
  }, [groupByDirectory, sortedDirectories])

  const toggleDirectory = (dir: string) => {
    const newExpanded = new Set(expandedDirectories)
    if (newExpanded.has(dir)) {
      newExpanded.delete(dir)
    } else {
      newExpanded.add(dir)
    }
    setExpandedDirectories(newExpanded)
  }

  // Функция для отображения элемента базы данных
  const renderDatabaseItem = (db: DatabaseInfo, isCurrent: boolean, index: number) => {
    return (
      <StaggerItem key={db.path}>
        <motion.div
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, x: 20 }}
          transition={{ duration: 0.3, delay: index * 0.05 }}
          whileHover={{ scale: 1.01 }}
          className={`flex items-center justify-between p-4 border rounded-lg transition-all ${
            isCurrent 
              ? 'bg-primary/10 border-primary shadow-md shadow-primary/10' 
              : 'bg-card hover:bg-muted/50 hover:border-muted-foreground/20'
          }`}
        >
          <div className="flex items-center gap-4 flex-1 min-w-0">
            <div className={`p-2 rounded-lg transition-colors ${
              isCurrent 
                ? 'bg-primary/20 text-primary' 
                : 'bg-muted text-muted-foreground'
            }`}>
              <Database className="h-5 w-5" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-2 flex-wrap">
                <p className="font-semibold text-base">{db.name}</p>
                {isCurrent && (
                  <Badge variant="default" className="text-xs gap-1">
                    <CheckCircle2 className="h-3 w-3" />
                    Текущая
                  </Badge>
                )}
                {db.type && <DatabaseTypeBadge type={db.type} />}
              </div>
              <div className="flex items-center gap-4 text-sm text-muted-foreground flex-wrap">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="flex items-center gap-1.5 hover:text-foreground transition-colors cursor-help">
                      <HardDrive className="h-3.5 w-3.5" />
                      {formatFileSize(db.size)}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Размер файла базы данных</p>
                  </TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="flex items-center gap-1.5 hover:text-foreground transition-colors cursor-help">
                      <Calendar className="h-3.5 w-3.5" />
                      {formatDate(db.modified_at)}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Дата последнего изменения</p>
                  </TooltipContent>
                </Tooltip>
                {!groupByDirectory && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex items-center gap-1.5 hover:text-foreground transition-colors cursor-help text-xs">
                        <Folder className="h-3.5 w-3.5" />
                        {formatPath(db.path)}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className="font-mono text-xs">{db.path}</p>
                    </TooltipContent>
                  </Tooltip>
                )}
                {db.table_count !== undefined && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex items-center gap-1.5 hover:text-foreground transition-colors cursor-help">
                        <Layers className="h-3.5 w-3.5" />
                        {db.table_count} таблиц
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Количество таблиц в базе данных</p>
                    </TooltipContent>
                  </Tooltip>
                )}
                {db.total_rows !== undefined && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex items-center gap-1.5 hover:text-foreground transition-colors cursor-help">
                        <Activity className="h-3.5 w-3.5" />
                        {formatNumber(db.total_rows)} записей
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Общее количество записей</p>
                    </TooltipContent>
                  </Tooltip>
                )}
                {db.stats && (db.stats.total_uploads !== undefined || db.stats.uploads_count !== undefined) && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="flex items-center gap-1.5 hover:text-foreground transition-colors cursor-help">
                        <FileText className="h-3.5 w-3.5" />
                        {db.stats.total_uploads ?? db.stats.uploads_count ?? 0} выгрузок
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <div className="space-y-1">
                        <p>Выгрузок: {db.stats.total_uploads ?? db.stats.uploads_count ?? 0}</p>
                        {db.stats.total_catalogs !== undefined && (
                          <p>Справочников: {db.stats.total_catalogs ?? db.stats.catalogs_count ?? 0}</p>
                        )}
                        {db.stats.total_items !== undefined && (
                          <p>Записей: {formatNumber(db.stats.total_items ?? db.stats.items_count ?? 0)}</p>
                        )}
                        {db.stats.last_upload_date && (
                          <p>Последняя выгрузка: {formatDateTime(new Date(db.stats.last_upload_date))}</p>
                        )}
                        {db.stats.avg_items_per_upload !== undefined && (
                          <p>Среднее записей/выгрузка: {formatNumber(Math.round(db.stats.avg_items_per_upload))}</p>
                        )}
                      </div>
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2 flex-shrink-0">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setAnalyticsDB({ name: db.name, path: db.path })
                    setShowAnalyticsDialog(true)
                  }}
                  className="gap-2"
                >
                  <BarChart3 className="h-4 w-4" />
                  <span className="hidden sm:inline">Аналитика</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Просмотр аналитики базы данных</p>
              </TooltipContent>
            </Tooltip>
            {!isCurrent && (
              <Button
                variant="default"
                size="sm"
                onClick={() => {
                  setSelectedDB(db.path)
                  setShowConfirmDialog(true)
                }}
                disabled={switching}
                className="gap-2"
              >
                {switching ? (
                  <>
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    <span className="hidden sm:inline">Переключение...</span>
                  </>
                ) : (
                  <>
                    <Database className="h-4 w-4" />
                    <span className="hidden sm:inline">Переключить</span>
                  </>
                )}
              </Button>
            )}
          </div>
        </motion.div>
      </StaggerItem>
    )
  }

  const breadcrumbItems = [
    { label: 'Базы данных', href: '/databases', icon: Database },
  ]

  if (loading) {
    return (
      <div className="container-wide mx-auto px-4 py-6 sm:py-8">
        <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
        <div className="mb-4">
          <Breadcrumb items={breadcrumbItems} />
        </div>
        <DatabasesPageSkeleton />
      </div>
    )
  }

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      {/* Header */}
      <FadeIn>
        <div className="mb-8">
          <motion.h1 
            className="text-3xl font-bold mb-2"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            Управление базами данных
          </motion.h1>
          <motion.p 
            className="text-muted-foreground"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            Просмотр и переключение между базами данных 1С
          </motion.p>
        </div>
      </FadeIn>

      {error && (
        <ErrorState
          message={error}
          action={{
            label: 'Повторить',
            onClick: fetchData,
          }}
          variant="destructive"
          className="mb-6"
        />
      )}

      {/* Current Database Info */}
      <AnimatePresence mode="wait">
        {currentDB && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
          >
            <Card className="mb-8 border-2 border-primary/20 bg-gradient-to-br from-primary/5 to-background relative overflow-hidden group">
              {/* Декоративный градиент */}
              <div className="absolute top-0 right-0 w-64 h-64 rounded-full bg-primary/10 blur-3xl group-hover:bg-primary/20 transition-colors" />
              
              <CardHeader className="relative z-10">
                <div className="flex items-center justify-between flex-wrap gap-4">
                  <div>
                    <CardTitle className="flex items-center gap-2 text-xl">
                      <div className="p-2 rounded-lg bg-primary/10 group-hover:bg-primary/20 transition-colors">
                        <Database className="h-5 w-5 text-primary" />
                      </div>
                      Текущая база данных
                    </CardTitle>
                    <CardDescription className="mt-1">Активная база данных для работы</CardDescription>
                  </div>
                  <Badge variant="outline" className="gap-2 border-green-500/50 bg-green-50 dark:bg-green-950/30">
                    <CheckCircle2 className="h-3 w-3 text-green-500 animate-pulse" />
                    <span className="text-green-700 dark:text-green-400 font-medium">Подключено</span>
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="relative z-10">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  <div className="space-y-4">
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                        <FileText className="h-4 w-4" />
                        Имя файла
                      </p>
                      <p className="text-lg font-semibold">{currentDB.name || 'Неизвестно'}</p>
                    </div>
                    <Separator />
                    <div className="space-y-1">
                      <p className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                        <HardDrive className="h-4 w-4" />
                        Путь
                      </p>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <p className="text-sm font-mono bg-muted px-3 py-2 rounded-md border hover:bg-muted/80 transition-colors cursor-help truncate">
                            {currentDB.path || 'Не указан'}
                          </p>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-md">
                          <p className="font-mono text-xs">{currentDB.path}</p>
                        </TooltipContent>
                      </Tooltip>
                    </div>
                    <Separator />
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-1">
                        <p className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                          <Activity className="h-4 w-4" />
                          Размер
                        </p>
                        <p className="text-base font-semibold">{formatFileSize(currentDB.size ?? 0)}</p>
                      </div>
                      <div className="space-y-1">
                        <p className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                          <Calendar className="h-4 w-4" />
                          Изменено
                        </p>
                        <p className="text-sm">{formatDate(currentDB.modified_at)}</p>
                      </div>
                    </div>
                  </div>

                  {(currentDB.stats || currentDB.upload_stats) && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-4 flex items-center gap-2">
                        <TrendingUp className="h-4 w-4" />
                        Статистика
                      </p>
                      <StaggerContainer className="grid grid-cols-1 gap-3">
                        <StaggerItem>
                          <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
                            <StatCard
                              title="Выгрузок"
                              value={
                                currentDB.stats?.total_uploads ?? 
                                currentDB.stats?.uploads_count ?? 
                                currentDB.upload_stats?.total_uploads ?? 
                                currentDB.upload_stats?.uploads_count ?? 
                                0
                              }
                              variant="default"
                              icon={Layers}
                              className="p-3"
                            />
                          </motion.div>
                        </StaggerItem>
                        <StaggerItem>
                          <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
                            <StatCard
                              title="Справочников"
                              value={
                                currentDB.stats?.total_catalogs ?? 
                                currentDB.stats?.catalogs_count ?? 
                                currentDB.upload_stats?.total_catalogs ?? 
                                currentDB.upload_stats?.catalogs_count ?? 
                                0
                              }
                              variant="default"
                              icon={FileText}
                              className="p-3"
                            />
                          </motion.div>
                        </StaggerItem>
                        <StaggerItem>
                          <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
                            <StatCard
                              title="Записей"
                              value={
                                currentDB.stats?.total_items ?? 
                                currentDB.stats?.items_count ?? 
                                currentDB.upload_stats?.total_items ?? 
                                currentDB.upload_stats?.items_count ?? 
                                0
                              }
                              variant="primary"
                              icon={Database}
                              className="p-3"
                              formatValue={(val) => formatNumber(val)}
                            />
                          </motion.div>
                        </StaggerItem>
                        {(currentDB.stats?.last_upload_date || currentDB.upload_stats?.last_upload_date || 
                          currentDB.stats?.avg_items_per_upload !== undefined || currentDB.upload_stats?.avg_items_per_upload !== undefined) && (
                          <>
                            {(currentDB.stats?.last_upload_date || currentDB.upload_stats?.last_upload_date) && (
                              <StaggerItem>
                                <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
                                  <StatCard
                                    title="Последняя выгрузка"
                                    value={formatDateTime(new Date(currentDB.stats?.last_upload_date || currentDB.upload_stats?.last_upload_date || ''))}
                                    variant="default"
                                    icon={Calendar}
                                    className="p-3"
                                  />
                                </motion.div>
                              </StaggerItem>
                            )}
                            {(currentDB.stats?.avg_items_per_upload !== undefined || currentDB.upload_stats?.avg_items_per_upload !== undefined) && (
                              <StaggerItem>
                                <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
                                  <StatCard
                                    title="Среднее записей/выгрузка"
                                    value={formatNumber(Math.round(currentDB.stats?.avg_items_per_upload ?? currentDB.upload_stats?.avg_items_per_upload ?? 0))}
                                    variant="default"
                                    icon={TrendingUp}
                                    className="p-3"
                                  />
                                </motion.div>
                              </StaggerItem>
                            )}
                          </>
                        )}
                      </StaggerContainer>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Overall Statistics */}
      {databases && databases.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.2 }}
        >
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                Общая статистика
              </CardTitle>
              <CardDescription>Сводка по всем базам данных</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard
                  title="Всего баз данных"
                  value={databases.length}
                  description={`${filteredAndSortedDatabases.length} отображается`}
                  icon={Database}
                  variant="default"
                />
                <StatCard
                  title="Общий размер"
                  value={formatFileSize(databases.reduce((sum, db) => sum + (db.size || 0), 0))}
                  description={`${databases.length} файлов`}
                  icon={HardDrive}
                  variant="primary"
                />
                <StatCard
                  title="Всего записей"
                  value={formatNumber(databases.reduce((sum, db) => sum + (db.total_rows || 0), 0))}
                  description="Во всех базах"
                  icon={Activity}
                  variant="default"
                />
                <StatCard
                  title="Всего таблиц"
                  value={databases.reduce((sum, db) => sum + (db.table_count || 0), 0)}
                  description="Во всех базах"
                  icon={Layers}
                  variant="default"
                />
              </div>
              
              {/* Статистика из выгрузок */}
              {(aggregatedStats || databases.some(db => db.stats && (db.stats.total_uploads !== undefined || db.stats.uploads_count !== undefined))) && (
                <div className="mt-6 pt-6 border-t">
                  <h3 className="text-sm font-semibold mb-4 flex items-center gap-2">
                    <FileText className="h-4 w-4" />
                    Статистика выгрузок
                  </h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                    <StatCard
                      title="Всего выгрузок"
                      value={formatNumber(
                        aggregatedStats?.total_uploads ?? aggregatedStats?.uploads_count ?? 
                        databases.reduce((sum, db) => 
                          sum + (db.stats?.total_uploads ?? db.stats?.uploads_count ?? 0), 0
                        )
                      )}
                      description={aggregatedStats?.databases_with_uploads ? 
                        `Из ${aggregatedStats.databases_with_uploads} баз данных` : 
                        "Из всех баз данных"}
                      icon={FileText}
                      variant="default"
                    />
                    <StatCard
                      title="Всего справочников"
                      value={formatNumber(
                        aggregatedStats?.total_catalogs ?? aggregatedStats?.catalogs_count ?? 
                        databases.reduce((sum, db) => 
                          sum + (db.stats?.total_catalogs ?? db.stats?.catalogs_count ?? 0), 0
                        )
                      )}
                      description="Из всех выгрузок"
                      icon={Layers}
                      variant="default"
                    />
                    <StatCard
                      title="Всего записей в выгрузках"
                      value={formatNumber(
                        aggregatedStats?.total_items ?? aggregatedStats?.items_count ?? 
                        databases.reduce((sum, db) => 
                          sum + (db.stats?.total_items ?? db.stats?.items_count ?? 0), 0
                        )
                      )}
                      description="Из всех выгрузок"
                      icon={Activity}
                      variant="default"
                    />
                  </div>
                  {(aggregatedStats?.last_upload_date || aggregatedStats?.avg_items_per_upload !== undefined) && (
                    <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
                      {aggregatedStats.last_upload_date && (
                        <StatCard
                          title="Последняя выгрузка"
                          value={formatDateTime(new Date(aggregatedStats.last_upload_date))}
                          description="Среди всех баз данных"
                          icon={Calendar}
                          variant="default"
                        />
                      )}
                      {aggregatedStats.avg_items_per_upload !== undefined && (
                        <StatCard
                          title="Среднее записей/выгрузка"
                          value={formatNumber(Math.round(aggregatedStats.avg_items_per_upload))}
                          description="По всем базам данных"
                          icon={TrendingUp}
                          variant="default"
                        />
                      )}
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Available Databases */}
      <FadeIn>
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between flex-wrap gap-4">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <div className="p-2 rounded-lg bg-muted">
                    <HardDrive className="h-5 w-5" />
                  </div>
                  Доступные базы данных
                </CardTitle>
                <CardDescription>
                  Список всех баз данных в текущей директории ({filteredAndSortedDatabases?.length || 0} из {databases?.length || 0})
                </CardDescription>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleScanDatabases}
                  disabled={scanning || loading}
                  className="gap-2"
                >
                  <Scan className={`h-4 w-4 ${scanning ? 'animate-pulse' : ''}`} />
                  {scanning ? 'Сканирование...' : 'Сканировать'}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={fetchData}
                  disabled={loading || scanning}
                  className="gap-2"
                >
                  <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                  Обновить
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {/* Фильтры и сортировка */}
            <div className="mb-6 space-y-4">
              <div className="flex flex-col sm:flex-row gap-4">
                {/* Поиск */}
                <div className="flex-1 relative">
                  <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Поиск по имени или пути..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-9"
                  />
                </div>
                
                {/* Фильтр по типу */}
                <Select value={filterType} onValueChange={setFilterType}>
                  <SelectTrigger className="w-full sm:w-[180px]">
                    <SelectValue placeholder="Тип БД" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Все типы</SelectItem>
                    <SelectItem value="uploads">Выгрузки</SelectItem>
                    <SelectItem value="service">Сервисная</SelectItem>
                    <SelectItem value="combined">Комбинированная</SelectItem>
                    <SelectItem value="benchmarks">Эталоны</SelectItem>
                  </SelectContent>
                </Select>

                {/* Сортировка */}
                <Select value={sortBy} onValueChange={(value) => setSortBy(value as typeof sortBy)}>
                  <SelectTrigger className="w-full sm:w-[180px]">
                    <SelectValue placeholder="Сортировка" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="name">По имени</SelectItem>
                    <SelectItem value="size">По размеру</SelectItem>
                    <SelectItem value="date">По дате</SelectItem>
                    <SelectItem value="rows">По записям</SelectItem>
                  </SelectContent>
                </Select>

                {/* Порядок сортировки */}
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
                  title={sortOrder === 'asc' ? 'По возрастанию' : 'По убыванию'}
                >
                  <ArrowUpDown className="h-4 w-4" />
                </Button>

                {/* Переключение группировки */}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={groupByDirectory ? "default" : "outline"}
                      size="icon"
                      onClick={() => setGroupByDirectory(!groupByDirectory)}
                      title={groupByDirectory ? "Показать плоский список" : "Группировать по директориям"}
                    >
                      {groupByDirectory ? <List className="h-4 w-4" /> : <Folder className="h-4 w-4" />}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{groupByDirectory ? "Показать плоский список" : "Группировать по директориям"}</p>
                  </TooltipContent>
                </Tooltip>
              </div>

              {/* Активные фильтры */}
              {(searchQuery || filterType !== 'all') && (
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-sm text-muted-foreground">Активные фильтры:</span>
                  {searchQuery && (
                    <Badge variant="secondary" className="gap-1">
                      Поиск: {searchQuery}
                      <button
                        onClick={() => setSearchQuery('')}
                        className="ml-1 hover:text-destructive"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  )}
                  {filterType !== 'all' && (
                    <Badge variant="secondary" className="gap-1">
                      Тип: {filterType === 'uploads' ? 'Выгрузки' : filterType === 'service' ? 'Сервисная' : filterType === 'combined' ? 'Комбинированная' : filterType === 'benchmarks' ? 'Эталоны' : filterType}
                      <button
                        onClick={() => setFilterType('all')}
                        className="ml-1 hover:text-destructive"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setSearchQuery('')
                      setFilterType('all')
                      setSortBy('name')
                      setSortOrder('asc')
                    }}
                    className="h-6 text-xs"
                  >
                    Сбросить все
                  </Button>
                </div>
              )}
            </div>
            <AnimatePresence mode="wait">
              {filteredAndSortedDatabases.length === 0 ? (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                >
                  <EmptyState
                    icon={Database}
                    title="Не найдено доступных баз данных"
                    description="В текущей директории нет доступных баз данных"
                  />
                </motion.div>
              ) : groupByDirectory ? (
                // Группировка по директориям
                <StaggerContainer className="space-y-4">
                  {sortedDirectories.map((dir, dirIndex) => {
                    const dirDatabases = groupedByDirectory[dir]
                    const isExpanded = expandedDirectories.has(dir)
                    const dirDisplayName = dir === '.' || dir === '' ? 'Текущая директория' : dir

                    return (
                      <StaggerItem key={dir}>
                        <Card className="overflow-hidden">
                          <CardHeader 
                            className="cursor-pointer hover:bg-muted/50 transition-colors"
                            onClick={() => toggleDirectory(dir)}
                          >
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-2 flex-wrap">
                                {isExpanded ? (
                                  <FolderOpen className="h-5 w-5 text-primary" />
                                ) : (
                                  <Folder className="h-5 w-5 text-muted-foreground" />
                                )}
                                <CardTitle className="text-base">{dirDisplayName}</CardTitle>
                                <Badge variant="secondary">{dirDatabases.length}</Badge>
                                {directoryStats[dir] && (
                                  <>
                                    <Badge variant="outline" className="text-xs">
                                      {formatFileSize(directoryStats[dir].totalSize)}
                                    </Badge>
                                    {directoryStats[dir].totalRows > 0 && (
                                      <Badge variant="outline" className="text-xs">
                                        {formatNumber(directoryStats[dir].totalRows)} записей
                                      </Badge>
                                    )}
                                  </>
                                )}
                              </div>
                            </div>
                          </CardHeader>
                          {isExpanded && (
                            <CardContent className="pt-0">
                              <div className="space-y-2">
                                {dirDatabases.map((db, index) => {
                                  const isCurrent = db.path === currentDB?.path
                                  return renderDatabaseItem(db, isCurrent, index)
                                })}
                              </div>
                            </CardContent>
                          )}
                        </Card>
                      </StaggerItem>
                    )
                  })}
                </StaggerContainer>
              ) : (
                // Плоский список
                <StaggerContainer className="space-y-3">
                  {filteredAndSortedDatabases.map((db, index) => {
                    const isCurrent = db.path === currentDB?.path
                    return renderDatabaseItem(db, isCurrent, index)
                  })}
                </StaggerContainer>
              )}
            </AnimatePresence>
          </CardContent>
        </Card>
      </FadeIn>

      {/* Confirmation Dialog */}
      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Подтвердите переключение базы данных</AlertDialogTitle>
            <AlertDialogDescription>
              Вы уверены, что хотите переключиться на базу данных{' '}
              <span className="font-semibold">{selectedDB}</span>?
              <br />
              <br />
              Это действие закроет текущее подключение и переключит все операции на выбранную базу данных.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={switching}>Отмена</AlertDialogCancel>
            <AlertDialogAction onClick={handleSwitchDatabase} disabled={switching}>
              {switching ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Переключение...
                </>
              ) : (
                'Переключить'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Analytics Dialog */}
      {analyticsDB && (
        <DatabaseAnalyticsDialog
          open={showAnalyticsDialog}
          onOpenChange={setShowAnalyticsDialog}
          databaseName={analyticsDB.name}
          databasePath={analyticsDB.path}
        />
      )}
    </div>
  )
}

