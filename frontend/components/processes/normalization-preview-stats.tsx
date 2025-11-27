'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Loader2, Database, Package, Building2, Copy, AlertCircle, RefreshCw, CheckCircle2, Download, Search, ArrowUpDown, ArrowUp, ArrowDown, RotateCw, Activity, ChevronLeft, ChevronRight, Info, Clipboard, Clock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { toast } from 'sonner'
import { NormalizationAnalyticsCharts } from './normalization-analytics-charts'
import { DataCompletenessAnalytics } from './data-completeness-analytics'
import { PreviewStatsResponse, DatabasePreviewStats, NormalizationType } from '@/types/normalization'
import { logger } from '@/lib/logger'
import { handleErrorWithDetails as handleError } from '@/lib/error-handler'

interface NormalizationPreviewStatsProps {
  clientId: number
  projectId: number
  normalizationType?: NormalizationType
  onReady?: (ready: boolean) => void
  onDatabasesSelected?: (selectedDatabaseIds: number[]) => void
  selectedDatabaseIds?: number[]
  onRefresh?: () => void
  onExport?: (format: 'csv' | 'json') => void
  onStatsUpdate?: (stats: { nomenclatureCount: number; counterpartyCount: number; totalRecords: number }) => void
  onFullStatsUpdate?: (stats: PreviewStatsResponse) => void
}

export function NormalizationPreviewStats({ 
  clientId, 
  projectId,
  normalizationType = 'both',
  onReady,
  onDatabasesSelected,
  selectedDatabaseIds = [],
  onRefresh,
  onExport,
  onStatsUpdate,
  onFullStatsUpdate
}: NormalizationPreviewStatsProps) {
  const [stats, setStats] = useState<PreviewStatsResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [loadingProgress, setLoadingProgress] = useState<number | null>(null)
  const [retryCount, setRetryCount] = useState(0)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortKey, setSortKey] = useState<'name' | 'nomenclature' | 'counterparty' | 'total' | 'size'>('name')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [statusFilter, setStatusFilter] = useState<'all' | 'ready' | 'problematic'>('all')
  const [sizeFilter, setSizeFilter] = useState<'all' | 'small' | 'medium' | 'large'>('all')
  const [recordsFilter, setRecordsFilter] = useState<'all' | 'small' | 'medium' | 'large'>('all')
  const [isFromCache, setIsFromCache] = useState(false)
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [autoRefreshInterval, setAutoRefreshInterval] = useState(30000) // 30 секунд по умолчанию
  const [currentPage, setCurrentPage] = useState(1)
  const [itemsPerPage, setItemsPerPage] = useState(10)
  const [localSelectedIds, setLocalSelectedIds] = useState<number[]>(selectedDatabaseIds)
  const [showSelectionMode, setShowSelectionMode] = useState(false)
  
  // Ключ для сохранения выбранных БД в localStorage
  const SELECTION_STORAGE_KEY = `normalization_selected_dbs_${clientId}_${projectId}`
  
  // Синхронизация выбранных ID с пропсами и загрузка из localStorage
  useEffect(() => {
    if (selectedDatabaseIds.length > 0) {
      setLocalSelectedIds(selectedDatabaseIds)
    } else {
      // Загружаем из localStorage, если нет пропсов
      try {
        const saved = localStorage.getItem(SELECTION_STORAGE_KEY)
        if (saved) {
          const parsed = JSON.parse(saved)
          if (Array.isArray(parsed) && parsed.length > 0) {
            setLocalSelectedIds(parsed)
            if (onDatabasesSelected) {
              onDatabasesSelected(parsed)
            }
          }
        }
      } catch (err) {
        logger.warn('Failed to load selected databases from localStorage', {
          clientId,
          projectId,
          storageKey: SELECTION_STORAGE_KEY,
          error: err instanceof Error ? err.message : String(err)
        })
      }
    }
  }, [selectedDatabaseIds, SELECTION_STORAGE_KEY, onDatabasesSelected])
  
  // Сохранение выбранных БД в localStorage
  useEffect(() => {
    if (localSelectedIds.length > 0) {
      try {
        localStorage.setItem(SELECTION_STORAGE_KEY, JSON.stringify(localSelectedIds))
      } catch (err) {
        logger.warn('Failed to save selected databases to localStorage', {
          component: 'NormalizationPreviewStats',
          clientId,
          projectId,
          storageKey: SELECTION_STORAGE_KEY,
          selectedCount: localSelectedIds.length,
          error: err instanceof Error ? err.message : String(err)
        })
      }
    } else {
      // Удаляем из localStorage, если выбор пуст
      try {
        localStorage.removeItem(SELECTION_STORAGE_KEY)
      } catch (err) {
        logger.warn('Failed to remove selected databases from localStorage', {
          component: 'NormalizationPreviewStats',
          clientId,
          projectId,
          storageKey: SELECTION_STORAGE_KEY,
          error: err instanceof Error ? err.message : String(err)
        })
      }
    }
  }, [localSelectedIds, SELECTION_STORAGE_KEY])
  
  // Кэш для статистики (5 минут) - включает normalizationType для разных фильтров
  const CACHE_KEY = `normalization_preview_stats_${clientId}_${projectId}_${normalizationType}`
  const CACHE_DURATION = 5 * 60 * 1000 // 5 минут

  const fetchStats = async (forceRefresh = false) => {
    // Проверяем кэш
    if (!forceRefresh) {
      try {
        const cached = localStorage.getItem(CACHE_KEY)
        if (cached) {
          const { data, timestamp } = JSON.parse(cached)
        const age = Date.now() - timestamp
        if (age < CACHE_DURATION) {
          // Валидация кэшированных данных
          if (data && typeof data === 'object' && 
              typeof data.total_databases === 'number' && 
              typeof data.total_records === 'number' &&
              Array.isArray(data.databases)) {
            setStats(data)
            setLastUpdated(new Date(timestamp))
            setIsFromCache(true)
            setIsLoading(false)
            
            logger.debug('Using cached stats', {
              component: 'NormalizationPreviewStats',
              clientId,
              projectId,
              normalizationType,
              cacheAge: `${Math.round(age / 1000)}s`,
              totalDatabases: data.total_databases
            })
            
            // Передаем данные статистики наверх
            if (onStatsUpdate) {
              onStatsUpdate({
                nomenclatureCount: typeof data.total_nomenclature === 'number' ? data.total_nomenclature : 0,
                counterpartyCount: typeof data.total_counterparties === 'number' ? data.total_counterparties : 0,
                totalRecords: typeof data.total_records === 'number' ? data.total_records : 0
              })
            }
            
            // Уведомляем о готовности системы
            if (onReady && data.total_databases > 0) {
              onReady(true)
            }
            return
          } else {
            logger.warn('Invalid cached data structure, clearing cache', {
              component: 'NormalizationPreviewStats',
              clientId,
              projectId,
              normalizationType
            })
            // Удаляем невалидный кэш
            try {
              localStorage.removeItem(CACHE_KEY)
            } catch (err) {
              // Игнорируем ошибку удаления
            }
          }
        } else {
          logger.debug('Cache expired, fetching fresh data', {
            component: 'NormalizationPreviewStats',
            clientId,
            projectId,
            normalizationType,
            cacheAge: `${Math.round(age / 1000)}s`,
            cacheDuration: `${CACHE_DURATION / 1000}s`
          })
        }
        }
      } catch (err) {
        // Игнорируем ошибки кэша, но логируем для отладки
        logger.warn('Failed to read cache', {
          component: 'NormalizationPreviewStats',
          clientId,
          projectId,
          normalizationType,
          error: err instanceof Error ? err.message : String(err)
        })
      }
    }

    setIsLoading(true)
    setLoadingProgress(0)
    // Очищаем ошибку только при принудительном обновлении, чтобы пользователь видел прогресс
    if (forceRefresh) {
      setError(null)
      setRetryCount(0)
    }
    
    try {
      // Симуляция прогресса загрузки
      const progressInterval = setInterval(() => {
        setLoadingProgress(prev => {
          if (prev === null) return 10
          if (prev >= 90) return prev
          return prev + 10
        })
      }, 200)

      // Формируем URL с параметром normalizationType
      const url = new URL(`/api/clients/${clientId}/projects/${projectId}/normalization/preview-stats`, window.location.origin)
      url.searchParams.set('normalization_type', normalizationType)

      // Добавляем retry логику с экспоненциальной задержкой
      let lastError: Error | null = null
      const maxRetries = 3
      let attempt = 0
      let data: PreviewStatsResponse | null = null
      
      while (attempt <= maxRetries) {
        try {
          const response = await fetch(url.toString(), {
            cache: 'no-store',
            signal: AbortSignal.timeout(30000), // 30 секунд таймаут
          })

          clearInterval(progressInterval)
          setLoadingProgress(90)

          if (!response.ok) {
            // Для ошибок 4xx не повторяем запрос
            if (response.status >= 400 && response.status < 500) {
              const errorData = await response.json().catch(() => ({ 
                error: `Ошибка клиента: ${response.status}` 
              }))
              throw new Error(errorData.error || `Ошибка ${response.status}: ${response.statusText}`)
            }
            
            // Для ошибок 5xx повторяем запрос
            if (response.status >= 500) {
              const errorData = await response.json().catch(() => ({ 
                error: `Ошибка сервера: ${response.status}` 
              }))
              lastError = new Error(errorData.error || `Ошибка сервера ${response.status}`)
              
              if (attempt < maxRetries) {
                attempt++
                const delay = Math.min(1000 * Math.pow(2, attempt - 1), 10000) // Экспоненциальная задержка, макс 10 сек
                await new Promise(resolve => setTimeout(resolve, delay))
                continue
              }
              throw lastError
            }
          }

          data = await response.json()
          
          // Валидация полученных данных
          if (!data || typeof data !== 'object') {
            throw new Error('Некорректный формат данных ответа')
          }
          
          // Проверка обязательных полей
          if (typeof data.total_databases !== 'number' || 
              typeof data.total_records !== 'number' ||
              !Array.isArray(data.databases)) {
            throw new Error('Некорректная структура данных: отсутствуют обязательные поля')
          }
          
          // Данные валидны, продолжаем обработку
          clearInterval(progressInterval)
          setLoadingProgress(90)
          
          logger.info('Preview stats fetched successfully', { 
            component: 'NormalizationPreviewStats',
            clientId, 
            projectId, 
            normalizationType,
            totalDatabases: data.total_databases,
            totalRecords: data.total_records,
            totalNomenclature: data.total_nomenclature,
            totalCounterparties: data.total_counterparties,
            estimatedDuplicates: data.estimated_duplicates,
            attempt: attempt + 1
          })

          // Данные успешно получены, выходим из цикла
          clearInterval(progressInterval)
          setLoadingProgress(100)
          
          // Обрабатываем успешно полученные данные
          const now = new Date()
          setStats(data)
          setLastUpdated(now)
          setIsFromCache(false)
          // Очищаем ошибку при успешном обновлении
          setError(null)
          
          // Передаем данные статистики наверх
          if (onStatsUpdate) {
            onStatsUpdate({
              nomenclatureCount: data.total_nomenclature || 0,
              counterpartyCount: data.total_counterparties || 0,
              totalRecords: data.total_records || 0
            })
          }
          
          // Передаем полный объект статистики
          if (onFullStatsUpdate) {
            onFullStatsUpdate(data)
          }
          
          // Сохраняем в кэш
          try {
            localStorage.setItem(CACHE_KEY, JSON.stringify({
              data,
              timestamp: now.getTime()
            }))
            logger.debug('Cache saved successfully', { 
              clientId, 
              projectId, 
              normalizationType,
              cacheKey: CACHE_KEY,
              dataSize: JSON.stringify(data).length
            })
          } catch (err) {
            // Игнорируем ошибки сохранения в кэш, но логируем
            logger.warn('Failed to save cache', { 
              clientId, 
              projectId, 
              normalizationType,
              error: err instanceof Error ? err.message : String(err)
            })
          }
          
          // Уведомляем о готовности системы
          if (onReady && data.total_databases > 0) {
            onReady(true)
          }
          
          // Успешно завершили, выходим из функции
          return
        } catch (fetchErr) {
          // Обработка ошибок сети и таймаутов
          lastError = fetchErr instanceof Error ? fetchErr : new Error(String(fetchErr))
          
          // Для ошибок сети и таймаутов повторяем запрос
          if (lastError.name === 'TimeoutError' || lastError.name === 'AbortError' || 
              lastError.message.includes('fetch') || lastError.message.includes('network')) {
            if (attempt < maxRetries) {
              attempt++
              const delay = Math.min(1000 * Math.pow(2, attempt - 1), 10000) // Экспоненциальная задержка, макс 10 сек
              setLoadingProgress(50) // Показываем прогресс retry
              logger.debug('Retrying fetch after error', {
                component: 'NormalizationPreviewStats',
                attempt,
                maxRetries,
                delay,
                error: lastError.message
              })
              await new Promise(resolve => setTimeout(resolve, delay))
              continue
            }
          }
          
          // Для других ошибок не повторяем - выходим из цикла
          break
        }
      }
      
      // Если все попытки исчерпаны или произошла не-retryable ошибка
      clearInterval(progressInterval)
      
      if (lastError) {
        throw lastError
      }
      
      throw new Error('Не удалось получить данные после нескольких попыток')
    } catch (err) {
      // Используем централизованную обработку ошибок
      const urlString = typeof window !== 'undefined' 
        ? `/api/clients/${clientId}/projects/${projectId}/normalization/preview-stats?normalization_type=${normalizationType}`
        : 'unknown'
      
      const errorDetails = handleError(
        err instanceof Error ? err : new Error(String(err)),
        'NormalizationPreviewStats',
        'fetchStats',
        { 
          clientId, 
          projectId, 
          normalizationType,
          forceRefresh,
          cacheKey: CACHE_KEY,
          url: urlString
        }
      )

      // Устанавливаем понятное сообщение об ошибке для пользователя
      setError(errorDetails.message)
      setLoadingProgress(null)

      // Если ошибка позволяет повторную попытку и есть кэш, показываем предупреждение
      if (errorDetails.isRetryable && stats) {
        logger.info('Using cached data due to error', { 
          component: 'NormalizationPreviewStats',
          error: errorDetails.message,
          hasCache: !!stats 
        })
      }

      // Показываем toast уведомление для критических ошибок
      if (errorDetails.statusCode && errorDetails.statusCode >= 500) {
        toast.error('Ошибка сервера', {
          description: errorDetails.message,
          duration: 5000,
        })
      } else if (!errorDetails.isRetryable) {
        toast.error('Ошибка загрузки', {
          description: errorDetails.message,
          duration: 4000,
        })
      }
    } finally {
      setIsLoading(false)
      setLoadingProgress(null)
    }
  }

  // Функция для повторной попытки загрузки
  const handleRetry = () => {
    if (retryCount < 3) {
      setRetryCount(prev => prev + 1)
      fetchStats(true)
    } else {
      setError('Превышено максимальное количество попыток. Проверьте подключение и попробуйте позже.')
    }
  }

  // Загрузка сохраненных фильтров из localStorage
  useEffect(() => {
    if (clientId && projectId) {
      try {
        const savedFilters = localStorage.getItem(`normalization_filters_${clientId}_${projectId}`)
        if (savedFilters) {
          const filters = JSON.parse(savedFilters)
          if (filters.statusFilter) setStatusFilter(filters.statusFilter)
          if (filters.sizeFilter) setSizeFilter(filters.sizeFilter)
          if (filters.recordsFilter) setRecordsFilter(filters.recordsFilter)
          if (filters.sortKey) setSortKey(filters.sortKey)
          if (filters.sortDirection) setSortDirection(filters.sortDirection)
          if (filters.itemsPerPage) setItemsPerPage(filters.itemsPerPage)
        }
      } catch (err) {
        logger.warn('Failed to load filters from localStorage', {
          component: 'NormalizationPreviewStats',
          clientId,
          projectId,
          error: err instanceof Error ? err.message : String(err)
        })
      }
      fetchStats()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clientId, projectId, normalizationType])

  // Сохранение фильтров в localStorage
  useEffect(() => {
    if (clientId && projectId) {
      try {
        localStorage.setItem(`normalization_filters_${clientId}_${projectId}`, JSON.stringify({
          statusFilter,
          sizeFilter,
          recordsFilter,
          sortKey,
          sortDirection,
          itemsPerPage,
        }))
      } catch (err) {
        logger.warn('Failed to save filters to localStorage', {
          component: 'NormalizationPreviewStats',
          clientId,
          projectId,
          filters: { statusFilter, sizeFilter, recordsFilter, sortKey, sortDirection, itemsPerPage },
          error: err instanceof Error ? err.message : String(err)
        })
      }
    }
  }, [clientId, projectId, statusFilter, sizeFilter, recordsFilter, sortKey, sortDirection, itemsPerPage])

  // Автоматическое обновление статистики
  useEffect(() => {
    if (!autoRefresh || !clientId || !projectId) return

    const interval = setInterval(() => {
      fetchStats(false) // Используем кэш, если доступен
    }, autoRefreshInterval)

    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoRefresh, autoRefreshInterval, clientId, projectId])

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 Б'
    const k = 1024
    const sizes = ['Б', 'КБ', 'МБ', 'ГБ']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  const formatDuration = (seconds: number): string => {
    if (seconds < 60) {
      return `${Math.ceil(seconds)} сек`
    } else if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60)
      const secs = Math.ceil(seconds % 60)
      return `${minutes} мин ${secs} сек`
    } else {
      const hours = Math.floor(seconds / 3600)
      const minutes = Math.floor((seconds % 3600) / 60)
      return `${hours} ч ${minutes} мин`
    }
  }

  const estimateProcessingTime = (totalRecords: number): string => {
    // Средняя скорость обработки: ~50-100 записей в секунду (консервативная оценка)
    // Для больших объемов может быть медленнее из-за дубликатов и AI обработки
    const baseSpeed = 50 // записей в секунду
    const estimatedSeconds = totalRecords / baseSpeed
    
    // Учитываем дополнительные факторы
    const duplicateFactor = stats?.estimated_duplicates ? 1.2 : 1.0 // Дубликаты требуют дополнительной обработки
    const aiFactor = 1.1 // AI обработка добавляет немного времени
    
    const finalSeconds = estimatedSeconds * duplicateFactor * aiFactor
    
    return formatDuration(finalSeconds)
  }

  // Функция для копирования информации о БД
  const handleCopyDatabaseInfo = async (db: DatabasePreviewStats) => {
    const info = [
      `База данных: ${db.database_name}`,
      `Путь: ${db.file_path}`,
      `Номенклатура: ${db.nomenclature_count.toLocaleString()}`,
      `Контрагенты: ${db.counterparty_count.toLocaleString()}`,
      `Всего записей: ${db.total_records.toLocaleString()}`,
      `Размер: ${formatBytes(db.database_size)}`,
      `Статус: ${db.is_valid === true && db.is_accessible === true && !db.error ? 'Готова' : 'Проблема'}`,
      db.error ? `Ошибка: ${db.error}` : '',
    ].filter(Boolean).join('\n')

    try {
      await navigator.clipboard.writeText(info)
      toast.success('Информация скопирована', {
        description: `Данные о БД "${db.database_name}" скопированы в буфер обмена`,
        duration: 2000,
      })
    } catch (err) {
      toast.error('Ошибка копирования', {
        description: 'Не удалось скопировать информацию',
        duration: 3000,
      })
    }
  }

  // Экспорт выбранных БД
  const handleExportSelected = (format: 'csv' | 'json') => {
    if (!stats || localSelectedIds.length === 0) {
      toast.error('Нет выбранных БД', {
        description: 'Выберите базы данных для экспорта',
        duration: 2000,
      })
      return
    }

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5)
    const selectedDBs = stats.databases.filter(db => localSelectedIds.includes(db.database_id))
    const filename = `normalization_selected_dbs_${clientId}_${projectId}_${timestamp}.${format}`

    if (format === 'csv') {
      const headers = ['ID БД', 'Название БД', 'Путь', 'Номенклатура', 'Контрагенты', 'Всего записей', 'Размер', 'Статус', 'Ошибка']
      const rows = selectedDBs.map(db => [
        db.database_id.toString(),
        db.database_name,
        db.file_path,
        db.nomenclature_count.toString(),
        db.counterparty_count.toString(),
        db.total_records.toString(),
        formatBytes(db.database_size),
        db.is_valid === true && db.is_accessible === true && !db.error ? 'Готова' : 'Проблема',
        db.error || '',
      ])
      const csvContent = '\uFEFF' + [headers, ...rows].map(row => row.map(cell => `"${cell}"`).join(',')).join('\n')
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = filename
      link.click()
      toast.success('Экспорт завершен', {
        description: `Выбранные БД экспортированы в CSV`,
        duration: 2000,
      })
    } else {
      const exportData = {
        export_date: new Date().toISOString(),
        client_id: clientId,
        project_id: projectId,
        selected_count: localSelectedIds.length,
        summary: {
          total_nomenclature: selectedStats?.totalNomenclature || 0,
          total_counterparties: selectedStats?.totalCounterparties || 0,
          total_records: selectedStats?.totalRecords || 0,
          total_size: selectedStats?.totalSize || 0,
          ready_count: selectedStats?.readyCount || 0,
        },
        databases: selectedDBs
      }
      const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = filename
      link.click()
      toast.success('Экспорт завершен', {
        description: `Выбранные БД экспортированы в JSON`,
        duration: 2000,
      })
    }
  }

  // Копирование списка выбранных БД
  const handleCopySelectedList = async () => {
    if (!stats || localSelectedIds.length === 0) {
      toast.error('Нет выбранных БД', {
        description: 'Выберите базы данных для копирования',
        duration: 2000,
      })
      return
    }

    const selectedDBs = stats.databases.filter(db => localSelectedIds.includes(db.database_id))
    const list = selectedDBs.map((db, index) => 
      `${index + 1}. ${db.database_name} (${db.total_records.toLocaleString()} записей, ${formatBytes(db.database_size)})`
    ).join('\n')

    const fullText = `Выбранные базы данных (${localSelectedIds.length}):\n\n${list}\n\n` +
      `Итого: ${selectedStats?.totalRecords.toLocaleString() || 0} записей, ` +
      `${formatBytes(selectedStats?.totalSize || 0)}`

    try {
      await navigator.clipboard.writeText(fullText)
      toast.success('Список скопирован', {
        description: `Список из ${localSelectedIds.length} выбранных БД скопирован в буфер обмена`,
        duration: 2000,
      })
    } catch (err) {
      logger.error('Failed to copy selected databases list', {
        component: 'NormalizationPreviewStats',
        clientId,
        projectId,
        selectedCount: localSelectedIds.length
      }, err instanceof Error ? err : undefined)
      
      toast.error('Ошибка копирования', {
        description: 'Не удалось скопировать список',
        duration: 3000,
      })
    }
  }

  const handleRefresh = useCallback(async () => {
    // Очищаем кэш перед обновлением
    try {
      localStorage.removeItem(CACHE_KEY)
    } catch (err) {
      logger.warn('Failed to clear cache before refresh', { 
        component: 'NormalizationPreviewStats',
        clientId, 
        projectId, 
        normalizationType,
        error: err instanceof Error ? err.message : String(err)
      })
    }
    
    setIsLoading(true)
    setError(null)
    setRetryCount(0)
    
    try {
      // Формируем URL с параметром normalizationType
      const url = new URL(`/api/clients/${clientId}/projects/${projectId}/normalization/preview-stats`, window.location.origin)
      url.searchParams.set('normalization_type', normalizationType)

      logger.debug('Refreshing stats', { 
        component: 'NormalizationPreviewStats',
        clientId, 
        projectId, 
        normalizationType,
        url: url.toString()
      })

      // Вызываем fetchStats напрямую
      const response = await fetch(url.toString(), {
        cache: 'no-store',
        signal: AbortSignal.timeout(30000),
      })

      if (!response.ok) {
        let errorData: { error?: string; message?: string } = {}
        try {
          errorData = await response.json()
        } catch (parseErr) {
          logger.warn('Failed to parse error response during refresh', { 
            component: 'NormalizationPreviewStats',
            status: response.status,
            statusText: response.statusText 
          })
          errorData = { error: `HTTP ${response.status}: ${response.statusText}` }
        }
        
        const fetchError = {
          status: response.status,
          statusText: response.statusText,
          error: errorData.error || errorData.message || `Ошибка ${response.status}`,
        }
        
        throw fetchError
      }

      const data = await response.json()
      setStats(data)
      const now = new Date()
      setLastUpdated(now)
      setIsFromCache(false)
      setError(null)
      
      // Передаем данные статистики наверх
      if (onStatsUpdate) {
        onStatsUpdate({
          nomenclatureCount: data.total_nomenclature || 0,
          counterpartyCount: data.total_counterparties || 0,
          totalRecords: data.total_records || 0
        })
      }
      
      // Сохраняем в кэш
      try {
        localStorage.setItem(CACHE_KEY, JSON.stringify({
          data,
          timestamp: now.getTime()
        }))
        logger.debug('Cache saved after refresh', { 
          component: 'NormalizationPreviewStats',
          clientId, 
          projectId, 
          normalizationType 
        })
      } catch (err) {
        logger.warn('Failed to save cache after refresh', { 
          component: 'NormalizationPreviewStats',
          clientId, 
          projectId, 
          normalizationType,
          error: err instanceof Error ? err.message : String(err)
        })
      }
      
      // Уведомляем о готовности системы
      if (onReady && data.total_databases > 0) {
        onReady(true)
      }
      
      logger.info('Stats refreshed successfully', { 
        component: 'NormalizationPreviewStats',
        clientId, 
        projectId, 
        normalizationType,
        databasesCount: data.total_databases,
        totalRecords: data.total_records
      })
      
      toast.success('Данные обновлены', {
        description: 'Статистика успешно обновлена',
        duration: 2000,
      })
    } catch (err) {
      const errorDetails = handleError(
        err,
        'NormalizationPreviewStats',
        'handleRefresh',
        { clientId, projectId, normalizationType }
      )
      
      setError(errorDetails.message)
      toast.error('Ошибка при обновлении', {
        description: errorDetails.message,
        duration: 3000,
      })
    } finally {
      setIsLoading(false)
      // Вызываем callback если он есть
      if (onRefresh) {
        onRefresh()
      }
    }
  }, [clientId, projectId, normalizationType, onReady, onRefresh])

  // Экспортируем handleRefresh и handleExport для использования извне
  useEffect(() => {
    // Сохраняем функцию обновления в window для доступа из родительского компонента
    if (clientId && projectId) {
      const refreshKey = `refreshStats_${clientId}_${projectId}` as keyof Window
      const exportKey = `exportStats_${clientId}_${projectId}` as keyof Window
      
      ;(window as unknown as Record<string, unknown>)[refreshKey] = handleRefresh
      ;(window as unknown as Record<string, unknown>)[exportKey] = (format: 'csv' | 'json') => {
        if (stats) {
          handleExport(format, false)
        } else {
          toast.error('Нет данных для экспорта', {
            description: 'Загрузите статистику перед экспортом',
            duration: 3000,
          })
        }
      }
      
      return () => {
        delete (window as unknown as Record<string, unknown>)[refreshKey]
        delete (window as unknown as Record<string, unknown>)[exportKey]
      }
    }
  }, [handleRefresh, clientId, projectId, stats])

  const handleExport = (format: 'csv' | 'json', exportFiltered = false) => {
    if (!stats) {
      logger.warn('Export attempted without stats', { 
        component: 'NormalizationPreviewStats',
        clientId, 
        projectId, 
        normalizationType,
        format,
        exportFiltered
      })
      
      toast.error('Нет данных для экспорта', {
        description: 'Загрузите статистику перед экспортом',
        duration: 3000,
      })
      return
    }

    logger.info('Starting export', { 
      component: 'NormalizationPreviewStats',
      clientId, 
      projectId, 
      normalizationType,
      format,
      exportFiltered,
      databasesCount: stats.databases.length
    })

    try {
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5)
      const dataToExport = exportFiltered ? filteredAndSortedDatabases : stats.databases
      const filename = `normalization_preview_stats_${clientId}_${projectId}_${timestamp}${exportFiltered ? '_filtered' : ''}.${format}`

      if (format === 'csv') {
        // CSV экспорт
        const headers = ['ID БД', 'Название БД', 'Путь', 'Номенклатура', 'Контрагенты', 'Всего записей', 'Размер', 'Статус', 'Ошибка']
        const rows = dataToExport.map(db => [
          db.database_id.toString(),
          db.database_name,
          db.file_path,
          db.nomenclature_count.toString(),
          db.counterparty_count.toString(),
          db.total_records.toString(),
          formatBytes(db.database_size),
          db.is_valid && db.is_accessible ? 'Готова' : 'Проблема',
          db.error || ''
        ])

        const csvContent = [
          headers.join(','),
          ...rows.map(row => row.map(cell => `"${cell.replace(/"/g, '""')}"`).join(','))
        ].join('\n')

        const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
        const link = document.createElement('a')
        link.href = URL.createObjectURL(blob)
        link.download = filename
        link.click()
        URL.revokeObjectURL(link.href)
      } else {
        // JSON экспорт
        const exportData = {
          export_date: new Date().toISOString(),
          client_id: clientId,
          project_id: projectId,
          filters_applied: exportFiltered ? {
            search_query: searchQuery,
            status_filter: statusFilter,
            sort_key: sortKey,
            sort_direction: sortDirection
          } : null,
          summary: {
            total_databases: stats.total_databases,
            accessible_databases: stats.accessible_databases,
            valid_databases: stats.valid_databases,
            total_nomenclature: stats.total_nomenclature,
            total_counterparties: stats.total_counterparties,
            total_records: stats.total_records,
            estimated_duplicates: stats.estimated_duplicates,
            duplicate_groups: stats.duplicate_groups,
            exported_count: dataToExport.length
          },
          databases: dataToExport
        }

        const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
        const link = document.createElement('a')
        link.href = URL.createObjectURL(blob)
        link.download = filename
        link.click()
        URL.revokeObjectURL(link.href)
      }
      
      // Вызываем callback если он есть
      if (onExport) {
        onExport(format)
      }
      
      logger.info('Export completed successfully', { 
        component: 'NormalizationPreviewStats',
        clientId, 
        projectId, 
        normalizationType,
        format,
        filename,
        exportFiltered,
        exportedCount: dataToExport.length
      })
      
      toast.success('Экспорт выполнен', {
        description: `Файл ${filename} успешно скачан`,
        duration: 3000,
      })
    } catch (error) {
      const errorDetails = handleError(
        error,
        'NormalizationPreviewStats',
        'handleExport',
        { 
          clientId, 
          projectId, 
          normalizationType,
          format,
          exportFiltered,
          databasesCount: stats.databases.length
        }
      )
      
      toast.error('Ошибка при экспорте', {
        description: errorDetails.message,
        duration: 5000,
      })
    }
  }

  // Обработчики выбора БД
  const handleToggleDatabase = (databaseId: number) => {
    setLocalSelectedIds(prev => {
      const newIds = prev.includes(databaseId)
        ? prev.filter(id => id !== databaseId)
        : [...prev, databaseId]
      
      // Уведомляем родительский компонент
      if (onDatabasesSelected) {
        onDatabasesSelected(newIds)
      }
      
      return newIds
    })
  }

  const handleSelectAll = () => {
    const readyDBs = filteredAndSortedDatabases
      .filter(db => db.is_valid === true && db.is_accessible === true && !db.error)
      .map(db => db.database_id)
    
    setLocalSelectedIds(readyDBs)
    if (onDatabasesSelected) {
      onDatabasesSelected(readyDBs)
    }
  }

  const handleDeselectAll = () => {
    setLocalSelectedIds([])
    if (onDatabasesSelected) {
      onDatabasesSelected([])
    }
  }

  // Быстрые действия для выбора БД по размеру
  const handleSelectBySize = (sizeCategory: 'small' | 'medium' | 'large') => {
    const dbsBySize = filteredAndSortedDatabases
      .filter(db => {
        const category = getDatabaseSizeCategory(db.database_size)
        return category === sizeCategory && 
               db.is_valid === true && 
               db.is_accessible === true && 
               !db.error
      })
      .map(db => db.database_id)
    
    const newIds = [...new Set([...localSelectedIds, ...dbsBySize])]
    setLocalSelectedIds(newIds)
    if (onDatabasesSelected) {
      onDatabasesSelected(newIds)
    }
  }

  // Проверка наличия проблемных БД среди выбранных
  const selectedProblematicDBs = stats?.databases?.filter(db => 
    localSelectedIds.includes(db.database_id) && 
    (db.is_valid === false || db.is_accessible === false || !!db.error)
  ) || []

  const handleSort = (key: typeof sortKey) => {
    if (sortKey === key) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortDirection('asc')
    }
  }

  // Функция для определения размера БД
  const getDatabaseSizeCategory = (size: number): 'small' | 'medium' | 'large' => {
    // Маленькие: до 10 МБ
    // Средние: 10 МБ - 100 МБ
    // Большие: свыше 100 МБ
    if (size < 10 * 1024 * 1024) return 'small'
    if (size < 100 * 1024 * 1024) return 'medium'
    return 'large'
  }

  // Функция для определения категории по количеству записей
  const getRecordsCategory = (records: number): 'small' | 'medium' | 'large' => {
    // Маленькие: до 10,000 записей
    // Средние: 10,000 - 100,000 записей
    // Большие: свыше 100,000 записей
    if (records < 10000) return 'small'
    if (records < 100000) return 'medium'
    return 'large'
  }

  // Подсчитываем количество БД по статусам и размерам (мемоизируем)
  const dbCounts = stats ? {
    total: stats.databases.length,
    ready: stats.databases.filter(db => db.is_valid === true && db.is_accessible === true && !db.error).length,
    problematic: stats.databases.filter(db => db.is_valid === false || db.is_accessible === false || !!db.error).length,
    bySize: {
      small: stats.databases.filter(db => getDatabaseSizeCategory(db.database_size) === 'small').length,
      medium: stats.databases.filter(db => getDatabaseSizeCategory(db.database_size) === 'medium').length,
      large: stats.databases.filter(db => getDatabaseSizeCategory(db.database_size) === 'large').length,
    },
    byRecords: {
      small: stats.databases.filter(db => getRecordsCategory(db.total_records) === 'small').length,
      medium: stats.databases.filter(db => getRecordsCategory(db.total_records) === 'medium').length,
      large: stats.databases.filter(db => getRecordsCategory(db.total_records) === 'large').length,
    }
  } : { total: 0, ready: 0, problematic: 0, bySize: { small: 0, medium: 0, large: 0 }, byRecords: { small: 0, medium: 0, large: 0 } }

  // Статистика по выбранным БД
  const selectedStats = stats && localSelectedIds.length > 0 ? {
    count: localSelectedIds.length,
    databases: stats.databases.filter(db => localSelectedIds.includes(db.database_id)),
    totalNomenclature: stats.databases
      .filter(db => localSelectedIds.includes(db.database_id))
      .reduce((sum, db) => sum + db.nomenclature_count, 0),
    totalCounterparties: stats.databases
      .filter(db => localSelectedIds.includes(db.database_id))
      .reduce((sum, db) => sum + db.counterparty_count, 0),
    totalRecords: stats.databases
      .filter(db => localSelectedIds.includes(db.database_id))
      .reduce((sum, db) => sum + db.total_records, 0),
    totalSize: stats.databases
      .filter(db => localSelectedIds.includes(db.database_id))
      .reduce((sum, db) => sum + db.database_size, 0),
    readyCount: stats.databases.filter(db => 
      localSelectedIds.includes(db.database_id) && 
      db.is_valid === true && 
      db.is_accessible === true && 
      !db.error
    ).length,
  } : null

  const filteredAndSortedDatabases = stats?.databases
    ?.filter(db => {
      // Фильтр по поисковому запросу
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase()
        const matchesSearch = (
          db.database_name.toLowerCase().includes(query) ||
          db.file_path.toLowerCase().includes(query) ||
          (db.error && db.error.toLowerCase().includes(query))
        )
        if (!matchesSearch) return false
      }
      
      // Фильтр по статусу
      if (statusFilter === 'ready') {
        if (!(db.is_valid === true && db.is_accessible === true && !db.error)) return false
      } else if (statusFilter === 'problematic') {
        if (!(db.is_valid === false || db.is_accessible === false || !!db.error)) return false
      }
      
      // Фильтр по размеру
      if (sizeFilter !== 'all') {
        const sizeCategory = getDatabaseSizeCategory(db.database_size)
        if (sizeCategory !== sizeFilter) return false
      }
      
      // Фильтр по количеству записей
      if (recordsFilter !== 'all') {
        const recordsCategory = getRecordsCategory(db.total_records)
        if (recordsCategory !== recordsFilter) return false
      }
      
      return true
    })
    .sort((a, b) => {
      let aValue: string | number
      let bValue: string | number

      switch (sortKey) {
        case 'name':
          aValue = a.database_name.toLowerCase()
          bValue = b.database_name.toLowerCase()
          break
        case 'nomenclature':
          aValue = a.nomenclature_count
          bValue = b.nomenclature_count
          break
        case 'counterparty':
          aValue = a.counterparty_count
          bValue = b.counterparty_count
          break
        case 'total':
          aValue = a.total_records
          bValue = b.total_records
          break
        case 'size':
          aValue = a.database_size
          bValue = b.database_size
          break
        default:
          return 0
      }

      if (typeof aValue === 'string' && typeof bValue === 'string') {
        const comparison = aValue.localeCompare(bValue, 'ru-RU', { numeric: true })
        return sortDirection === 'asc' ? comparison : -comparison
      } else {
        const comparison = (aValue as number) - (bValue as number)
        return sortDirection === 'asc' ? comparison : -comparison
      }
    }) || []

  // Пагинация
  const totalPages = Math.ceil((filteredAndSortedDatabases.length || 0) / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const paginatedDatabases = filteredAndSortedDatabases.slice(startIndex, endIndex)

  // Сброс страницы при изменении фильтров
  useEffect(() => {
    setCurrentPage(1)
  }, [searchQuery, statusFilter, sizeFilter, recordsFilter, sortKey, sortDirection])

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Статистика запускаемой нормализации</CardTitle>
          <CardDescription>Сбор данных...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-8 space-y-4">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
            <div className="text-center space-y-2 w-full max-w-md">
              <p className="text-sm font-medium">Сбор статистики по базам данных...</p>
              <p className="text-xs text-muted-foreground">
                Проверка доступности и подсчет записей
              </p>
              <Progress value={loadingProgress || undefined} className="w-full" />
              {loadingProgress !== null && (
                <p className="text-xs text-muted-foreground">
                  {loadingProgress}%
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error && !stats) {
    // Показываем ошибку только если нет данных для отображения
    return (
      <Card>
        <CardHeader>
          <CardTitle>Статистика запускаемой нормализации</CardTitle>
          <CardDescription>Не удалось загрузить статистику</CardDescription>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive" className="mb-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              <div className="space-y-2">
                <p>{error}</p>
                {retryCount < 3 && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleRetry}
                    className="mt-2"
                  >
                    <RefreshCw className="h-3 w-3 mr-2" />
                    Повторить попытку ({retryCount}/3)
                  </Button>
                )}
              </div>
            </AlertDescription>
          </Alert>
          <div className="flex gap-2">
            <Button
              variant="default"
              size="sm"
              onClick={() => {
                setError(null)
                fetchStats(true)
              }}
              disabled={isLoading}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              Попробовать снова
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setError(null)
                fetchStats(false)
              }}
              disabled={isLoading}
            >
              Загрузить из кэша
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!stats) {
    return null
  }

  return (
    <Card className="backdrop-blur-sm bg-card/95 border-border/50 shadow-lg">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              Статистика запускаемой нормализации
            </CardTitle>
            <CardDescription>
              <div className="flex items-center justify-between">
                <span>
                  Предварительная статистика по базам данных проекта перед запуском нормализации
                </span>
                {stats && (
                  <div className="flex items-center gap-3 text-xs">
                    {stats.valid_databases !== undefined && (
                      <span className="text-muted-foreground">
                        {stats.valid_databases}/{stats.total_databases} готовы
                      </span>
                    )}
                    {lastUpdated && (
                      <span className="text-muted-foreground flex items-center gap-1">
                        {isFromCache && (
                          <Badge variant="outline" className="text-xs px-1 py-0 h-4">
                            Кэш
                          </Badge>
                        )}
                        обновлено: {lastUpdated.toLocaleTimeString('ru-RU')}
                      </span>
                    )}
                  </div>
                )}
              </div>
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!stats || isLoading}
                  title="Экспорт статистики"
                >
                  <Download className="h-4 w-4 mr-2" />
                  Экспорт
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleExport('csv', false)}>
                  Экспорт всех в CSV
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleExport('json', false)}>
                  Экспорт всех в JSON
                </DropdownMenuItem>
                {(searchQuery || statusFilter !== 'all' || sizeFilter !== 'all' || recordsFilter !== 'all' || sortKey !== 'name' || sortDirection !== 'asc') && (
                  <>
                    <DropdownMenuItem onClick={() => handleExport('csv', true)}>
                      Экспорт отфильтрованных в CSV
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleExport('json', true)}>
                      Экспорт отфильтрованных в JSON
                    </DropdownMenuItem>
                  </>
                )}
                {localSelectedIds.length > 0 && (
                  <>
                    <DropdownMenuItem onClick={() => handleExportSelected('csv')}>
                      Экспорт выбранных в CSV ({localSelectedIds.length})
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleExportSelected('json')}>
                      Экспорт выбранных в JSON ({localSelectedIds.length})
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={handleCopySelectedList}>
                      <Clipboard className="h-3 w-3 mr-2" />
                      Копировать список выбранных
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => fetchStats(false)}
                    disabled={isLoading}
                    title="Обновить из кэша (если доступен)"
                  >
                    <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  <p className="text-xs">Обновить статистику (использует кэш если доступен)</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => {
                      // Очищаем кэш перед принудительным обновлением
                      try {
                        localStorage.removeItem(CACHE_KEY)
                        setIsFromCache(false)
                      } catch (err) {
                        logger.warn('Failed to clear cache', {
                          component: 'NormalizationPreviewStats',
                          clientId,
                          projectId,
                          normalizationType,
                          cacheKey: CACHE_KEY,
                          error: err instanceof Error ? err.message : String(err)
                        })
                      }
                      fetchStats(true)
                    }}
                    disabled={isLoading}
                    className="flex items-center gap-2"
                    title="Принудительное обновление (игнорирует кэш)"
                  >
                    <RotateCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
                    {isLoading ? 'Обновление...' : 'Обновить данные'}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  <p className="text-xs">
                    Принудительное обновление статистики
                    <br />
                    Очищает кэш и загружает свежие данные с сервера
                  </p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
            {/* Переключатель автоматического обновления */}
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant={autoRefresh ? "default" : "outline"}
                    size="icon"
                    onClick={() => setAutoRefresh(!autoRefresh)}
                    title={autoRefresh ? "Отключить автоматическое обновление" : "Включить автоматическое обновление"}
                  >
                    <Activity className={`h-4 w-4 ${autoRefresh ? 'animate-pulse' : ''}`} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  <p className="text-xs">
                    {autoRefresh 
                      ? `Автообновление включено (каждые ${autoRefreshInterval / 1000} сек)`
                      : 'Включить автоматическое обновление статистики'
                    }
                  </p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Показываем ошибку, если есть, но не скрываем данные */}
        {error && stats && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription className="flex items-center justify-between">
              <span>{error}</span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setError(null)
                  fetchStats(true)
                }}
                disabled={isLoading}
                className="ml-4"
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                Обновить
              </Button>
            </AlertDescription>
          </Alert>
        )}
        
        {/* Информация о готовности */}
        {(() => {
          const accessibleCount = stats.accessible_databases ?? stats.total_databases
          const validCount = stats.valid_databases ?? stats.total_databases
          const hasIssues = accessibleCount < stats.total_databases || validCount < stats.total_databases
          
          if (hasIssues) {
            return (
              <Alert className="border-yellow-200 bg-yellow-50 dark:bg-yellow-950 dark:border-yellow-800">
                <AlertCircle className="h-4 w-4 text-yellow-600 dark:text-yellow-400" />
                <AlertDescription>
                  <div className="space-y-2">
                    <p className="font-medium text-yellow-900 dark:text-yellow-100">
                      ⚠️ Обнаружены проблемы с некоторыми базами данных
                    </p>
                    <div className="text-sm text-yellow-800 dark:text-yellow-200 space-y-1">
                      <p>
                        Из {stats.total_databases} активных баз данных:
                      </p>
                      <ul className="list-disc list-inside ml-2 space-y-0.5">
                        <li><strong>{accessibleCount}</strong> доступны для обработки</li>
                        <li><strong>{validCount}</strong> валидны и готовы к нормализации</li>
                        {accessibleCount < stats.total_databases && (
                          <li className="text-red-600 dark:text-red-400">
                            {stats.total_databases - accessibleCount} баз данных недоступны
                          </li>
                        )}
                      </ul>
                      <p className="mt-2">
                        Нормализация будет запущена только для доступных и валидных баз данных.
                      </p>
                    </div>
                  </div>
                </AlertDescription>
              </Alert>
            )
          }
          
          return (
            <Alert className="border-green-200 bg-green-50 dark:bg-green-950 dark:border-green-800">
              <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
              <AlertDescription>
                <div className="space-y-2">
                  <p className="font-medium text-green-900 dark:text-green-100">
                    ✅ Система готова принять записи из всех {stats.total_databases} активных баз данных проекта
                  </p>
                  <div className="text-sm text-green-800 dark:text-green-200 space-y-1">
                    <p>
                      При запуске нормализации будут обработаны <strong>все активные базы данных</strong> проекта параллельно.
                    </p>
                    <p>
                      Всего найдено <strong>{stats.total_records.toLocaleString()}</strong> записей:
                    </p>
                    <ul className="list-disc list-inside ml-2 space-y-0.5">
                      <li><strong>{stats.total_nomenclature.toLocaleString()}</strong> записей номенклатуры</li>
                      <li><strong>{stats.total_counterparties.toLocaleString()}</strong> записей контрагентов</li>
                      {stats.estimated_duplicates > 0 && (
                        <li>~<strong>{stats.estimated_duplicates.toLocaleString()}</strong> возможных дубликатов</li>
                      )}
                      {stats.total_records > 0 && (
                        <li className="mt-2 pt-2 border-t border-green-200 dark:border-green-800">
                          <strong>Оценка времени обработки:</strong> ~{estimateProcessingTime(stats.total_records)}
                        </li>
                      )}
                    </ul>
                  </div>
                </div>
              </AlertDescription>
            </Alert>
          )
        })()}

        {/* Панель быстрой аналитики (KPI) */}
        <div className="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-4 gap-3 md:gap-4">
          <Card className="bg-linear-to-br from-blue-50/50 to-blue-100/50 dark:from-blue-950/20 dark:to-blue-900/20 border-blue-200/50 dark:border-blue-800/50">
            <CardContent className="pt-6">
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground flex items-center gap-1.5 uppercase tracking-wide">
                  <Database className="h-3.5 w-3.5" />
                  Баз данных
                </div>
                <div className="text-2xl md:text-3xl font-bold text-blue-600">
                  {stats.total_databases.toLocaleString()}
                </div>
                {stats.valid_databases !== undefined && (
                  <div className="text-xs text-muted-foreground">
                    {stats.valid_databases} готовы
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
          <Card className="bg-linear-to-br from-purple-50/50 to-purple-100/50 dark:from-purple-950/20 dark:to-purple-900/20 border-purple-200/50 dark:border-purple-800/50">
            <CardContent className="pt-6">
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground flex items-center gap-1.5 uppercase tracking-wide">
                  <Package className="h-3.5 w-3.5" />
                  Номенклатура
                </div>
                <div className="text-2xl md:text-3xl font-bold text-purple-600">
                  {stats.total_nomenclature.toLocaleString()}
                </div>
                {stats.total_records > 0 && (
                  <div className="text-xs text-muted-foreground">
                    {((stats.total_nomenclature / stats.total_records) * 100).toFixed(1)}% от общего объема
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
          <Card className="bg-linear-to-br from-green-50/50 to-green-100/50 dark:from-green-950/20 dark:to-green-900/20 border-green-200/50 dark:border-green-800/50">
            <CardContent className="pt-6">
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground flex items-center gap-1.5 uppercase tracking-wide">
                  <Building2 className="h-3.5 w-3.5" />
                  Контрагенты
                </div>
                <div className="text-2xl md:text-3xl font-bold text-green-600">
                  {stats.total_counterparties.toLocaleString()}
                </div>
                {stats.total_records > 0 && (
                  <div className="text-xs text-muted-foreground">
                    {((stats.total_counterparties / stats.total_records) * 100).toFixed(1)}% от общего объема
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
          <Card className="bg-linear-to-br from-orange-50/50 to-orange-100/50 dark:from-orange-950/20 dark:to-orange-900/20 border-orange-200/50 dark:border-orange-800/50">
            <CardContent className="pt-6">
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground flex items-center gap-1.5 uppercase tracking-wide">
                  <Clock className="h-3.5 w-3.5" />
                  Время обработки
                </div>
                {stats.total_records > 0 ? (
                  <>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="text-xl md:text-2xl font-bold text-orange-600 cursor-help">
                            ~{estimateProcessingTime(stats.total_records)}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p className="text-xs">
                            Оценка основана на средней скорости обработки ~50 записей/сек
                            <br />
                            Учитываются дубликаты и AI обработка
                            <br />
                            Всего записей: {stats.total_records.toLocaleString()}
                          </p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                    <div className="text-xs text-muted-foreground">
                      {stats.total_records.toLocaleString()} записей
                    </div>
                  </>
                ) : (
                  <div className="text-xl md:text-2xl font-bold text-muted-foreground">—</div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Аналитика заполнения справочников */}
        <DataCompletenessAnalytics
          completeness={stats.completeness_metrics}
          normalizationType={normalizationType}
          isLoading={isLoading}
        />

        {/* Детальная предварительная аналитика (графики) */}
        <NormalizationAnalyticsCharts stats={stats} isLoading={isLoading} normalizationType={normalizationType} />

        {/* Статистика по выбранным БД */}
        {selectedStats && selectedStats.count > 0 && (
          <Card className="backdrop-blur-sm bg-linear-to-br from-primary/10 via-primary/5 to-background border-primary/30 shadow-lg">
            <CardHeader className="pb-3">
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <CheckCircle2 className="h-5 w-5 text-primary" />
                Статистика по выбранным БД
                <Badge variant="default" className="ml-2">
                  {selectedStats.count} {selectedStats.count === 1 ? 'БД' : selectedStats.count < 5 ? 'БД' : 'БД'}
                </Badge>
              </CardTitle>
              <CardDescription className="text-xs">
                Сводная информация по выбранным базам данных для обработки
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
                <div className="space-y-1 p-3 bg-background/50 rounded-lg border border-green-200/50 dark:border-green-800/50">
                  <div className="text-xs text-muted-foreground uppercase tracking-wide flex items-center gap-1">
                    <CheckCircle2 className="h-3 w-3 text-green-600" />
                    Готовых БД
                  </div>
                  <div className="text-2xl font-bold text-green-600">
                    {selectedStats.readyCount} / {selectedStats.count}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {selectedStats.count > 0 ? ((selectedStats.readyCount / selectedStats.count) * 100).toFixed(0) : 0}% готовы
                  </div>
                </div>
                <div className="space-y-1 p-3 bg-background/50 rounded-lg border border-blue-200/50 dark:border-blue-800/50">
                  <div className="text-xs text-muted-foreground uppercase tracking-wide flex items-center gap-1">
                    <Package className="h-3 w-3 text-blue-600" />
                    Номенклатура
                  </div>
                  <div className="text-2xl font-bold text-blue-600">
                    {selectedStats.totalNomenclature.toLocaleString()}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {selectedStats.totalRecords > 0 ? ((selectedStats.totalNomenclature / selectedStats.totalRecords) * 100).toFixed(1) : 0}% от общего
                  </div>
                </div>
                <div className="space-y-1 p-3 bg-background/50 rounded-lg border border-green-200/50 dark:border-green-800/50">
                  <div className="text-xs text-muted-foreground uppercase tracking-wide flex items-center gap-1">
                    <Building2 className="h-3 w-3 text-green-600" />
                    Контрагенты
                  </div>
                  <div className="text-2xl font-bold text-green-600">
                    {selectedStats.totalCounterparties.toLocaleString()}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {selectedStats.totalRecords > 0 ? ((selectedStats.totalCounterparties / selectedStats.totalRecords) * 100).toFixed(1) : 0}% от общего
                  </div>
                </div>
                <div className="space-y-1 p-3 bg-background/50 rounded-lg border border-orange-200/50 dark:border-orange-800/50">
                  <div className="text-xs text-muted-foreground uppercase tracking-wide flex items-center gap-1">
                    <Copy className="h-3 w-3 text-orange-600" />
                    Всего записей
                  </div>
                  <div className="text-2xl font-bold text-orange-600">
                    {selectedStats.totalRecords.toLocaleString()}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {selectedStats.totalRecords > 0 && (
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        ~{estimateProcessingTime(selectedStats.totalRecords)}
                      </span>
                    )}
                  </div>
                </div>
              </div>
              <div className="pt-4 border-t border-primary/20 space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground flex items-center gap-1">
                    <Database className="h-3 w-3" />
                    Общий размер:
                  </span>
                  <span className="font-semibold">{formatBytes(selectedStats.totalSize)}</span>
                </div>
                {selectedStats.totalRecords > 0 && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      Оценка времени обработки:
                    </span>
                    <span className="font-semibold text-orange-600">
                      ~{estimateProcessingTime(selectedStats.totalRecords)}
                    </span>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Дополнительная информация о дубликатах */}
        {stats.estimated_duplicates > 0 && stats.total_records > 0 && (
          <div className="text-xs text-muted-foreground space-y-1 pt-2 border-t">
            <div className="flex items-center justify-between">
              <span>Потенциальные дубликаты:</span>
              <span className="font-medium text-orange-600">
                {((stats.estimated_duplicates / stats.total_records) * 100).toFixed(1)}% от общего количества
              </span>
            </div>
          </div>
        )}

        {/* Сводка по проблемным БД */}
        {(() => {
          const problematicDBs = stats.databases?.filter(db => 
            db.is_valid === false || db.is_accessible === false || db.error
          ) || []
          
          if (problematicDBs.length > 0) {
            return (
              <Alert variant="destructive" className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <div className="space-y-1">
                    <p className="font-medium">
                      Обнаружены проблемы с {problematicDBs.length} {problematicDBs.length === 1 ? 'базой данных' : 'базами данных'}:
                    </p>
                    <ul className="list-disc list-inside ml-2 text-sm space-y-0.5">
                      {problematicDBs.map(db => (
                        <li key={db.database_id}>
                          <strong>{db.database_name}</strong>: {db.error || 'Недоступна или невалидна'}
                        </li>
                      ))}
                    </ul>
                    <p className="text-sm mt-2">
                      Эти базы данных будут пропущены при запуске нормализации.
                    </p>
                  </div>
                </AlertDescription>
              </Alert>
            )
          }
          return null
        })()}

        {/* Детальная статистика по базам данных */}
        {stats.databases && stats.databases.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-medium">
                Статистика по базам данных:
                {stats.valid_databases !== undefined && (
                  <span className="ml-2 text-xs text-muted-foreground font-normal">
                    ({stats.valid_databases} из {stats.total_databases} готовы к обработке)
                  </span>
                )}
              </h3>
            </div>

            {/* Режим выбора БД и фильтры */}
            {onDatabasesSelected && (
              <div className="space-y-2 mb-2">
                <div className="flex items-center gap-2 flex-wrap">
                  <Button
                    variant={showSelectionMode ? "default" : "outline"}
                    size="sm"
                    onClick={() => setShowSelectionMode(!showSelectionMode)}
                  >
                    {showSelectionMode ? 'Завершить выбор' : 'Выбрать БД'}
                  </Button>
                  {showSelectionMode && (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleSelectAll}
                      >
                        Выбрать все готовые
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleDeselectAll}
                      >
                        Снять выбор
                      </Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="outline" size="sm">
                            Выбрать по размеру
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => handleSelectBySize('small')}>
                            <Database className="h-3 w-3 mr-2 text-blue-500" />
                            Маленькие (&lt;10 МБ) ({dbCounts.bySize.small})
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleSelectBySize('medium')}>
                            <Database className="h-3 w-3 mr-2 text-yellow-500" />
                            Средние (10-100 МБ) ({dbCounts.bySize.medium})
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleSelectBySize('large')}>
                            <Database className="h-3 w-3 mr-2 text-orange-500" />
                            Большие (&gt;100 МБ) ({dbCounts.bySize.large})
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                      {localSelectedIds.length > 0 && (
                        <Badge variant="default" className="ml-2">
                          Выбрано: {localSelectedIds.length}
                        </Badge>
                      )}
                    </>
                  )}
                </div>
                {/* Предупреждение о проблемных БД */}
                {selectedProblematicDBs.length > 0 && (
                  <Alert variant="destructive" className="py-2">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription className="text-sm">
                      В выбранных БД обнаружены проблемы: {selectedProblematicDBs.length} {selectedProblematicDBs.length === 1 ? 'база данных' : 'баз данных'} имеет ошибки или недоступна. 
                      Рекомендуется снять выбор с проблемных БД перед запуском нормализации.
                    </AlertDescription>
                  </Alert>
                )}
              </div>
            )}
            
            {/* Поиск и фильтры */}
            <div className="flex items-center gap-2">
              {stats.databases.length > 3 && (
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Поиск по названию или пути..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-9"
                  />
                </div>
              )}
              <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as typeof statusFilter)}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="Фильтр по статусу" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">
                    Все БД ({dbCounts.total})
                  </SelectItem>
                  <SelectItem value="ready">
                    <span className="flex items-center gap-2">
                      <CheckCircle2 className="h-3 w-3 text-green-600" />
                      Готовы ({dbCounts.ready})
                    </span>
                  </SelectItem>
                  <SelectItem value="problematic">
                    <span className="flex items-center gap-2">
                      <AlertCircle className="h-3 w-3 text-destructive" />
                      Проблемные ({dbCounts.problematic})
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
              <Select value={sizeFilter} onValueChange={(value) => setSizeFilter(value as typeof sizeFilter)}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="Фильтр по размеру" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">
                    Все размеры
                  </SelectItem>
                  <SelectItem value="small">
                    <span className="flex items-center gap-2">
                      <Database className="h-3 w-3 text-blue-500" />
                      Маленькие (&lt;10 МБ) ({dbCounts.bySize.small})
                    </span>
                  </SelectItem>
                  <SelectItem value="medium">
                    <span className="flex items-center gap-2">
                      <Database className="h-3 w-3 text-yellow-500" />
                      Средние (10-100 МБ) ({dbCounts.bySize.medium})
                    </span>
                  </SelectItem>
                  <SelectItem value="large">
                    <span className="flex items-center gap-2">
                      <Database className="h-3 w-3 text-orange-500" />
                      Большие (&gt;100 МБ) ({dbCounts.bySize.large})
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
              <Select value={recordsFilter} onValueChange={(value) => setRecordsFilter(value as typeof recordsFilter)}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="Фильтр по записям" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">
                    Все объемы
                  </SelectItem>
                  <SelectItem value="small">
                    <span className="flex items-center gap-2">
                      <Package className="h-3 w-3 text-blue-500" />
                      Маленькие (&lt;10K) ({dbCounts.byRecords.small})
                    </span>
                  </SelectItem>
                  <SelectItem value="medium">
                    <span className="flex items-center gap-2">
                      <Package className="h-3 w-3 text-yellow-500" />
                      Средние (10K-100K) ({dbCounts.byRecords.medium})
                    </span>
                  </SelectItem>
                  <SelectItem value="large">
                    <span className="flex items-center gap-2">
                      <Package className="h-3 w-3 text-orange-500" />
                      Большие (&gt;100K) ({dbCounts.byRecords.large})
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="border rounded-lg overflow-hidden max-h-[600px] overflow-y-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    {showSelectionMode && onDatabasesSelected && (
                      <TableHead className="w-12">
                        <Checkbox
                          checked={paginatedDatabases.length > 0 && paginatedDatabases.every(db => 
                            localSelectedIds.includes(db.database_id) && 
                            db.is_valid === true && 
                            db.is_accessible === true && 
                            !db.error
                          )}
                          onCheckedChange={(checked) => {
                            if (checked) {
                              const readyDBs = paginatedDatabases
                                .filter(db => db.is_valid === true && db.is_accessible === true && !db.error)
                                .map(db => db.database_id)
                              const newIds = [...new Set([...localSelectedIds, ...readyDBs])]
                              setLocalSelectedIds(newIds)
                              if (onDatabasesSelected) {
                                onDatabasesSelected(newIds)
                              }
                            } else {
                              const pageIds = paginatedDatabases.map(db => db.database_id)
                              const newIds = localSelectedIds.filter(id => !pageIds.includes(id))
                              setLocalSelectedIds(newIds)
                              if (onDatabasesSelected) {
                                onDatabasesSelected(newIds)
                              }
                            }
                          }}
                        />
                      </TableHead>
                    )}
                    <TableHead>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 -ml-2"
                        onClick={() => handleSort('name')}
                      >
                        База данных
                        {sortKey === 'name' ? (
                          sortDirection === 'asc' ? (
                            <ArrowUp className="h-3 w-3 ml-1" />
                          ) : (
                            <ArrowDown className="h-3 w-3 ml-1" />
                          )
                        ) : (
                          <ArrowUpDown className="h-3 w-3 ml-1 opacity-50" />
                        )}
                      </Button>
                    </TableHead>
                    <TableHead className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 -mr-2"
                        onClick={() => handleSort('nomenclature')}
                      >
                        Номенклатура
                        {sortKey === 'nomenclature' ? (
                          sortDirection === 'asc' ? (
                            <ArrowUp className="h-3 w-3 ml-1" />
                          ) : (
                            <ArrowDown className="h-3 w-3 ml-1" />
                          )
                        ) : (
                          <ArrowUpDown className="h-3 w-3 ml-1 opacity-50" />
                        )}
                      </Button>
                    </TableHead>
                    <TableHead className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 -mr-2"
                        onClick={() => handleSort('counterparty')}
                      >
                        Контрагенты
                        {sortKey === 'counterparty' ? (
                          sortDirection === 'asc' ? (
                            <ArrowUp className="h-3 w-3 ml-1" />
                          ) : (
                            <ArrowDown className="h-3 w-3 ml-1" />
                          )
                        ) : (
                          <ArrowUpDown className="h-3 w-3 ml-1 opacity-50" />
                        )}
                      </Button>
                    </TableHead>
                    <TableHead className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 -mr-2"
                        onClick={() => handleSort('total')}
                      >
                        Всего
                        {sortKey === 'total' ? (
                          sortDirection === 'asc' ? (
                            <ArrowUp className="h-3 w-3 ml-1" />
                          ) : (
                            <ArrowDown className="h-3 w-3 ml-1" />
                          )
                        ) : (
                          <ArrowUpDown className="h-3 w-3 ml-1 opacity-50" />
                        )}
                      </Button>
                    </TableHead>
                    <TableHead className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 px-2 -mr-2"
                        onClick={() => handleSort('size')}
                      >
                        Размер
                        {sortKey === 'size' ? (
                          sortDirection === 'asc' ? (
                            <ArrowUp className="h-3 w-3 ml-1" />
                          ) : (
                            <ArrowDown className="h-3 w-3 ml-1" />
                          )
                        ) : (
                          <ArrowUpDown className="h-3 w-3 ml-1 opacity-50" />
                        )}
                      </Button>
                    </TableHead>
                    <TableHead className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Clock className="h-3 w-3" />
                        <span>Время</span>
                      </div>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedDatabases.map((db) => {
                    const isSelected = localSelectedIds.includes(db.database_id)
                    const canSelect = db.is_valid === true && db.is_accessible === true && !db.error
                    
                    return (
                    <TableRow 
                      key={db.database_id}
                      className={showSelectionMode && isSelected ? 'bg-muted/50' : ''}
                    >
                      {showSelectionMode && onDatabasesSelected && (
                        <TableCell>
                          <Checkbox
                            checked={isSelected}
                            disabled={!canSelect}
                            onCheckedChange={() => handleToggleDatabase(db.database_id)}
                          />
                        </TableCell>
                      )}
                      <TableCell>
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <div className="font-medium cursor-help">{db.database_name}</div>
                                </TooltipTrigger>
                                <TooltipContent className="max-w-md">
                                  <div className="space-y-2 text-sm">
                                    <div><strong>База данных:</strong> {db.database_name}</div>
                                    <div><strong>Путь:</strong> <span className="text-xs break-all">{db.file_path}</span></div>
                                    <div><strong>Номенклатура:</strong> {db.nomenclature_count.toLocaleString()}</div>
                                    <div><strong>Контрагенты:</strong> {db.counterparty_count.toLocaleString()}</div>
                                    <div><strong>Всего записей:</strong> {db.total_records.toLocaleString()}</div>
                                    <div><strong>Размер:</strong> {formatBytes(db.database_size)}</div>
                                    <div><strong>Статус:</strong> {
                                      db.is_valid === true && db.is_accessible === true && !db.error 
                                        ? 'Готова к обработке' 
                                        : db.error 
                                          ? `Ошибка: ${db.error}` 
                                          : 'Недоступна или невалидна'
                                    }</div>
                                    {db.total_records > 0 && (
                                      <div><strong>Оценка времени обработки:</strong> ~{estimateProcessingTime(db.total_records)}</div>
                                    )}
                                  </div>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-6 w-6 p-0"
                                    onClick={() => handleCopyDatabaseInfo(db)}
                                  >
                                    <Clipboard className="h-3 w-3" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>
                                  <p>Копировать информацию о БД</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                            {db.is_valid !== false && db.is_accessible !== false && (
                              <Badge variant="outline" className="text-xs text-green-600 border-green-300">
                                <CheckCircle2 className="h-3 w-3 mr-1" />
                                Готова
                              </Badge>
                            )}
                            {(db.is_valid === false || db.is_accessible === false || db.error) && (
                              <Badge variant="outline" className="text-xs text-destructive border-destructive">
                                <AlertCircle className="h-3 w-3 mr-1" />
                                Проблема
                              </Badge>
                            )}
                            {db.total_records > 50000 && (
                              <Badge variant="secondary" className="text-xs">
                                Большая БД
                              </Badge>
                            )}
                          </div>
                          {db.file_path && (
                            <div className="text-xs text-muted-foreground font-mono truncate max-w-[300px]" title={db.file_path}>
                              {db.file_path.split(/[/\\]/).pop() || db.file_path}
                            </div>
                          )}
                          {db.error && (
                            <div className="text-xs text-destructive flex items-center gap-1">
                              <AlertCircle className="h-3 w-3" />
                              {db.error}
                            </div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        {db.error ? (
                          <span className="text-muted-foreground">—</span>
                        ) : (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="cursor-help">
                                  {db.nomenclature_count.toLocaleString()}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="text-xs">
                                  {db.total_records > 0 
                                    ? `${((db.nomenclature_count / db.total_records) * 100).toFixed(1)}% от всех записей`
                                    : 'Номенклатура'}
                                </p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        {db.error ? (
                          <span className="text-muted-foreground">—</span>
                        ) : (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="cursor-help">
                                  {db.counterparty_count.toLocaleString()}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="text-xs">
                                  {db.total_records > 0 
                                    ? `${((db.counterparty_count / db.total_records) * 100).toFixed(1)}% от всех записей`
                                    : 'Контрагенты'}
                                </p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        {db.error ? (
                          <span className="text-muted-foreground">—</span>
                        ) : (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Badge variant="outline" className="cursor-help">
                                  {db.total_records.toLocaleString()}
                                </Badge>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="text-xs">
                                  Оценка времени: ~{estimateProcessingTime(db.total_records)}
                                </p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </TableCell>
                      <TableCell className="text-right text-xs text-muted-foreground">
                        {db.error ? (
                          <span className="text-muted-foreground">—</span>
                        ) : (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="cursor-help">
                                  {formatBytes(db.database_size)}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="text-xs">
                                  {db.database_size > 0 && db.total_records > 0 && (
                                    <>~{formatBytes(Math.floor(db.database_size / db.total_records))} на запись</>
                                  )}
                                </p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </TableCell>
                      <TableCell className="text-right text-xs">
                        {db.error ? (
                          <span className="text-muted-foreground">—</span>
                        ) : db.total_records > 0 ? (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <div className="flex items-center justify-end gap-1 text-orange-600 font-medium cursor-help">
                                  <Clock className="h-3 w-3" />
                                  <span>~{estimateProcessingTime(db.total_records)}</span>
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="text-xs">
                                  Оценка времени обработки для {db.total_records.toLocaleString()} записей
                                  <br />
                                  Основано на скорости ~50 записей/сек
                                  <br />
                                  Учитываются дубликаты и AI обработка
                                </p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                    </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
            {/* Пагинация и информация */}
            <div className="flex items-center justify-between flex-wrap gap-2">
              <div className="text-xs text-muted-foreground flex items-center gap-2 flex-wrap">
                <span>
                  Показано {startIndex + 1}-{Math.min(endIndex, filteredAndSortedDatabases.length)} из {filteredAndSortedDatabases.length} баз данных
                  {filteredAndSortedDatabases.length !== stats.databases.length && ` (из ${stats.databases.length} всего)`}
                </span>
                {(searchQuery || statusFilter !== 'all' || sortKey !== 'name' || sortDirection !== 'asc') && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={() => {
                      setSearchQuery('')
                      setStatusFilter('all')
                      setSizeFilter('all')
                      setRecordsFilter('all')
                      setSortKey('name')
                      setSortDirection('asc')
                    }}
                  >
                    Сбросить фильтры
                  </Button>
                )}
              </div>
              
              {/* Элементы управления пагинацией */}
              {totalPages > 1 && (
                <div className="flex items-center gap-2">
                  <Select value={itemsPerPage.toString()} onValueChange={(value) => {
                    setItemsPerPage(Number(value))
                    setCurrentPage(1)
                  }}>
                    <SelectTrigger className="w-[100px] h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="10">10</SelectItem>
                      <SelectItem value="25">25</SelectItem>
                      <SelectItem value="50">50</SelectItem>
                      <SelectItem value="100">100</SelectItem>
                    </SelectContent>
                  </Select>
                  
                  <div className="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                      disabled={currentPage === 1}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Button>
                    
                    <div className="flex items-center gap-1 px-2">
                      <span className="text-sm text-muted-foreground">
                        Страница {currentPage} из {totalPages}
                      </span>
                    </div>
                    
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                      disabled={currentPage === totalPages}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

