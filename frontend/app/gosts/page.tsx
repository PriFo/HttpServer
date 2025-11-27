'use client'

import { useState, useEffect, useCallback, useMemo, Suspense } from 'react'
import { useSearchParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { GostList } from '@/components/gosts/gost-list'
import { GostFilters } from '@/components/gosts/gost-filters'
import { GostImportDialog } from '@/components/gosts/gost-import-dialog'
import { GostStatisticsChart } from '@/components/gosts/gost-statistics-chart'
import { StatCard } from '@/components/common/stat-card'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { 
  FileText, 
  Upload, 
  BarChart3, 
  Search,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  Loader2,
  Download,
  RefreshCw,
  Grid3x3,
  List,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Hash,
  X,
  Star,
  Database,
  Filter,
  Clock
} from 'lucide-react'
import { motion } from 'framer-motion'
import { FadeIn } from '@/components/animations/fade-in'
import Link from 'next/link'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import { getFavorites, getHistory, clearHistory, type FavoriteGost, type HistoryGost } from '@/lib/gost-favorites'
import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent } from '@/components/ui/collapsible'

interface Gost {
  id: number
  gost_number: string
  title: string
  adoption_date?: string
  effective_date?: string
  status?: string
  source_type?: string
  source_url?: string
  description?: string
  keywords?: string
}

interface Statistics {
  total_gosts?: number
  by_status?: Record<string, number>
  by_source_type?: Record<string, number>
  total_documents?: number
  total_sources?: number
}

const ITEMS_PER_PAGE = 20

function GostsPageContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  
  const [gosts, setGosts] = useState<Gost[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState(searchParams.get('search') || '')
  const [statusFilter, setStatusFilter] = useState(searchParams.get('status') || '')
  const [sourceTypeFilter, setSourceTypeFilter] = useState(searchParams.get('source_type') || '')
  const [sourceTypes, setSourceTypes] = useState<string[]>([])
  const [currentPage, setCurrentPage] = useState(parseInt(searchParams.get('page') || '1', 10))
  const [total, setTotal] = useState(0)
  const [statistics, setStatistics] = useState<Statistics | null>(null)
  const [statisticsLoading, setStatisticsLoading] = useState(false)
  const [showImportDialog, setShowImportDialog] = useState(false)
  const [sortField, setSortField] = useState<'gost_number' | 'adoption_date' | 'effective_date' | 'status'>(
    (searchParams.get('sort') as 'gost_number' | 'adoption_date' | 'effective_date' | 'status' | null) || 'gost_number'
  )
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>(
    (searchParams.get('order') as 'asc' | 'desc' | null) || 'asc'
  )
  const [viewMode, setViewMode] = useState<'grid' | 'list'>(
    (searchParams.get('view') as 'grid' | 'list' | null) || 'grid'
  )
  const [exporting, setExporting] = useState(false)
  const [adoptionFrom, setAdoptionFrom] = useState(searchParams.get('adoption_from') || '')
  const [adoptionTo, setAdoptionTo] = useState(searchParams.get('adoption_to') || '')
  const [effectiveFrom, setEffectiveFrom] = useState(searchParams.get('effective_from') || '')
  const [effectiveTo, setEffectiveTo] = useState(searchParams.get('effective_to') || '')
  const [quickSearchNumber, setQuickSearchNumber] = useState('')
  const [showQuickSearch, setShowQuickSearch] = useState(false)
  const [showFavorites, setShowFavorites] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const [historyItems, setHistoryItems] = useState<HistoryGost[]>([])
  const [isFromCache, setIsFromCache] = useState(false)
  const [filtersCollapsed, setFiltersCollapsed] = useState(false)

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }
    setFiltersCollapsed(window.innerWidth < 640)
  }, [])

  const refreshHistory = useCallback(() => {
    const history = getHistory()
    setHistoryItems(Array.isArray(history) ? history : [])
  }, [])

  useEffect(() => {
    refreshHistory()
    if (typeof window === 'undefined') {
      return
    }
    const handleFocus = () => refreshHistory()
    window.addEventListener('focus', handleFocus)
    return () => {
      window.removeEventListener('focus', handleFocus)
    }
  }, [refreshHistory])

  // Обновление URL при изменении фильтров
  useEffect(() => {
    const params = new URLSearchParams()
    if (searchQuery) params.set('search', searchQuery)
    if (statusFilter) params.set('status', statusFilter)
    if (sourceTypeFilter) params.set('source_type', sourceTypeFilter)
    if (currentPage > 1) params.set('page', currentPage.toString())
    if (sortField !== 'gost_number') params.set('sort', sortField)
    if (sortOrder !== 'asc') params.set('order', sortOrder)
    if (viewMode !== 'grid') params.set('view', viewMode)
    if (adoptionFrom) params.set('adoption_from', adoptionFrom)
    if (adoptionTo) params.set('adoption_to', adoptionTo)
    if (effectiveFrom) params.set('effective_from', effectiveFrom)
    if (effectiveTo) params.set('effective_to', effectiveTo)

    const newUrl = params.toString() ? `?${params.toString()}` : '/gosts'
    router.replace(newUrl, { scroll: false })
  }, [
    searchQuery,
    statusFilter,
    sourceTypeFilter,
    currentPage,
    sortField,
    sortOrder,
    viewMode,
    adoptionFrom,
    adoptionTo,
    effectiveFrom,
    effectiveTo,
    router,
  ])

  // Кэш для запросов ГОСТов (2 минуты)
  const CACHE_KEY = useMemo(() => {
    return [
      'gosts',
      currentPage,
      searchQuery,
      statusFilter,
      sourceTypeFilter,
      sortField,
      sortOrder,
      adoptionFrom,
      adoptionTo,
      effectiveFrom,
      effectiveTo,
    ].join('_')
  }, [
    currentPage,
    searchQuery,
    statusFilter,
    sourceTypeFilter,
    sortField,
    sortOrder,
    adoptionFrom,
    adoptionTo,
    effectiveFrom,
    effectiveTo,
  ])
  const CACHE_DURATION = 2 * 60 * 1000 // 2 минуты

  const fetchGosts = useCallback(async (forceRefresh = false) => {
    // Проверяем кэш
    if (!forceRefresh) {
      try {
        const cached = sessionStorage.getItem(CACHE_KEY)
        if (cached) {
          const { data, timestamp } = JSON.parse(cached)
          const age = Date.now() - timestamp
          if (age < CACHE_DURATION) {
            setGosts(data.gosts || [])
            setTotal(data.total || 0)
            setLoading(false)
            setError(null)
            setIsFromCache(true)
            // Обновляем в фоне, если кэш старше половины времени жизни
            if (age > CACHE_DURATION / 2) {
              fetchGosts(true)
            }
            return
          }
        }
      } catch (err) {
        console.warn('Failed to read cache:', err)
      }
    }

    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams()
      params.append('limit', ITEMS_PER_PAGE.toString())
      params.append('offset', ((currentPage - 1) * ITEMS_PER_PAGE).toString())
      
      if (searchQuery.trim()) {
        params.append('search', searchQuery.trim())
      }
      if (statusFilter) {
        params.append('status', statusFilter)
      }
      if (sourceTypeFilter) {
        params.append('source_type', sourceTypeFilter)
      }
      if (adoptionFrom) {
        params.append('adoption_from', adoptionFrom)
      }
      if (adoptionTo) {
        params.append('adoption_to', adoptionTo)
      }
      if (effectiveFrom) {
        params.append('effective_from', effectiveFrom)
      }
      if (effectiveTo) {
        params.append('effective_to', effectiveTo)
      }

      const response = await fetch(`/api/gosts?${params.toString()}`)
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Ошибка загрузки ГОСТов' }))
        throw new Error(errorData.error || 'Ошибка загрузки ГОСТов')
      }

      const data = await response.json()
      
      // Логируем структуру ответа для отладки
      console.log('GOSTs API Response:', {
        total: data.total,
        limit: data.limit,
        offset: data.offset,
        gostsCount: data.gosts?.length || 0,
        structure: {
          hasGosts: Array.isArray(data.gosts),
          hasTotal: typeof data.total === 'number',
          firstGost: data.gosts?.[0] ? Object.keys(data.gosts[0]) : null,
        }
      })
      
      let gostsList = data.gosts || []
      
      // Сортировка на клиенте (создаем новый массив для избежания мутаций)
      const sortedGosts = [...gostsList].sort((a: Gost, b: Gost) => {
        let comparison = 0
        switch (sortField) {
          case 'gost_number':
            comparison = (a.gost_number || '').localeCompare(b.gost_number || '')
            break
          case 'adoption_date':
            const dateA = a.adoption_date ? new Date(a.adoption_date).getTime() : 0
            const dateB = b.adoption_date ? new Date(b.adoption_date).getTime() : 0
            comparison = dateA - dateB
            break
          case 'effective_date':
            const effDateA = a.effective_date ? new Date(a.effective_date).getTime() : 0
            const effDateB = b.effective_date ? new Date(b.effective_date).getTime() : 0
            comparison = effDateA - effDateB
            break
          case 'status':
            comparison = (a.status || '').localeCompare(b.status || '')
            break
        }
        return sortOrder === 'asc' ? comparison : -comparison
      })
      
      setGosts(sortedGosts)
      setTotal(data.total || 0)
      setIsFromCache(false)
      
      // Сохраняем в кэш
      try {
        sessionStorage.setItem(CACHE_KEY, JSON.stringify({
          data: { gosts: sortedGosts, total: data.total || 0 },
          timestamp: Date.now()
        }))
      } catch (err) {
        console.warn('Failed to save cache:', err)
      }
    } catch (err) {
      // Если есть кэш, показываем его при ошибке
      try {
        const cached = sessionStorage.getItem(CACHE_KEY)
        if (cached) {
          const { data } = JSON.parse(cached)
          setGosts(data.gosts || [])
          setTotal(data.total || 0)
          setError((err instanceof Error ? err.message : 'Ошибка загрузки ГОСТов') + '. Показаны кэшированные данные.')
        } else {
          setError(err instanceof Error ? err.message : 'Ошибка загрузки ГОСТов')
          setGosts([])
        }
      } catch (cacheErr) {
        setError(err instanceof Error ? err.message : 'Ошибка загрузки ГОСТов')
        setGosts([])
      }
    } finally {
      setLoading(false)
    }
  }, [
    currentPage,
    searchQuery,
    statusFilter,
    sourceTypeFilter,
    sortField,
    sortOrder,
    adoptionFrom,
    adoptionTo,
    effectiveFrom,
    effectiveTo,
    CACHE_KEY,
  ])

  // Кэш для статистики (5 минут)
  const STATS_CACHE_KEY = 'gosts_statistics'
  const STATS_CACHE_DURATION = 5 * 60 * 1000 // 5 минут

  const fetchStatistics = useCallback(async (forceRefresh = false) => {
    // Проверяем кэш
    if (!forceRefresh) {
      try {
        const cached = localStorage.getItem(STATS_CACHE_KEY)
        if (cached) {
          const { data, timestamp } = JSON.parse(cached)
          const age = Date.now() - timestamp
          if (age < STATS_CACHE_DURATION) {
            setStatistics(data)
            // Извлекаем список типов источников из статистики
            if (data?.by_source_type && typeof data.by_source_type === 'object') {
              setSourceTypes(Object.keys(data.by_source_type))
            }
            // Обновляем в фоне, если кэш старше половины времени жизни
            if (age > STATS_CACHE_DURATION / 2) {
              fetchStatistics(true)
            }
            return
          }
        }
      } catch (err) {
        console.warn('Failed to read statistics cache:', err)
      }
    }

    setStatisticsLoading(true)
    try {
      const response = await fetch('/api/gosts/statistics')
      if (response.ok) {
        const data = await response.json()
        
        // Логируем статистику для отладки
        console.log('GOSTs Statistics:', {
          total_gosts: data.total_gosts,
          total_sources: data.total_sources,
          total_documents: data.total_documents,
          by_status: data.by_status,
          by_source_type: data.by_source_type ? Object.keys(data.by_source_type).length : 0,
          sourceTypes: data.by_source_type ? Object.keys(data.by_source_type) : [],
        })
        
        setStatistics(data)
        
        // Извлекаем список типов источников из статистики
        if (data.by_source_type) {
          setSourceTypes(Object.keys(data.by_source_type))
        }
        
        // Сохраняем в кэш
        try {
          localStorage.setItem(STATS_CACHE_KEY, JSON.stringify({
            data,
            timestamp: Date.now()
          }))
        } catch (err) {
          console.warn('Failed to save statistics cache:', err)
        }
      }
    } catch (err) {
      console.error('Error fetching statistics:', err)
      // Если есть кэш, показываем его при ошибке
      try {
        const cached = localStorage.getItem(STATS_CACHE_KEY)
        if (cached) {
          const { data } = JSON.parse(cached)
          setStatistics(data)
          if (data.by_source_type) {
            setSourceTypes(Object.keys(data.by_source_type))
          }
        }
      } catch (cacheErr) {
        // Игнорируем ошибки кэша
      }
    } finally {
      setStatisticsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchStatistics()
  }, [fetchStatistics])

  useEffect(() => {
    fetchGosts()
  }, [fetchGosts])

  // Debounce для поиска
  useEffect(() => {
    const timer = setTimeout(() => {
      setCurrentPage(1)
      fetchGosts()
    }, 500)

    return () => clearTimeout(timer)
  }, [searchQuery, statusFilter, sourceTypeFilter, adoptionFrom, adoptionTo, effectiveFrom, effectiveTo, fetchGosts])

  const handleClearFilters = () => {
    setSearchQuery('')
    setStatusFilter('')
    setSourceTypeFilter('')
    setAdoptionFrom('')
    setAdoptionTo('')
    setEffectiveFrom('')
    setEffectiveTo('')
    setCurrentPage(1)
  }

  const handleClearHistory = () => {
    clearHistory()
    setHistoryItems([])
    toast.info('История очищена')
  }

  const formatViewedAt = useCallback((dateStr: string) => {
    if (!dateStr) return ''
    try {
      const date = new Date(dateStr)
      return new Intl.DateTimeFormat('ru-RU', {
        day: '2-digit',
        month: 'short',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }).format(date)
    } catch {
      return dateStr
    }
  }, [])

  const hasActiveFilters = !!(
    searchQuery ||
    statusFilter ||
    sourceTypeFilter ||
    adoptionFrom ||
    adoptionTo ||
    effectiveFrom ||
    effectiveTo
  )
  const activeFiltersCount = useMemo(() => {
    let count = 0
    if (searchQuery) count += 1
    if (statusFilter) count += 1
    if (sourceTypeFilter) count += 1
    if (adoptionFrom) count += 1
    if (adoptionTo) count += 1
    if (effectiveFrom) count += 1
    if (effectiveTo) count += 1
    return count
  }, [searchQuery, statusFilter, sourceTypeFilter, adoptionFrom, adoptionTo, effectiveFrom, effectiveTo])

  const totalPages = Math.ceil(total / ITEMS_PER_PAGE)

  const handleExport = async () => {
    if (exporting) return
    
    setExporting(true)
    try {
      // Формируем параметры запроса с учетом текущих фильтров
      const params = new URLSearchParams()
      if (searchQuery) params.set('search', searchQuery)
      if (statusFilter) params.set('status', statusFilter)
      if (sourceTypeFilter) params.set('source_type', sourceTypeFilter)
      if (adoptionFrom) params.set('adoption_from', adoptionFrom)
      if (adoptionTo) params.set('adoption_to', adoptionTo)
      if (effectiveFrom) params.set('effective_from', effectiveFrom)
      if (effectiveTo) params.set('effective_to', effectiveTo)

      // Показываем уведомление о начале экспорта
      const exportToast = toast.loading('Экспорт данных...', {
        description: 'Подготовка файла для скачивания',
      })

      // Запрашиваем экспорт с сервера
      const response = await fetch(`/api/gosts/export?${params.toString()}`)
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Ошибка экспорта' }))
        throw new Error(errorData.error || 'Не удалось экспортировать данные')
      }

      // Получаем имя файла из заголовка Content-Disposition или генерируем
      const contentDisposition = response.headers.get('Content-Disposition')
      let filename = `gosts_export_${new Date().toISOString().split('T')[0]}.csv`
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/)
        if (filenameMatch && filenameMatch[1]) {
          filename = filenameMatch[1].replace(/['"]/g, '')
        }
      }

      // Создаем blob из ответа
      const blob = await response.blob()
      
      // Используем File System Access API для выбора места сохранения (если доступен)
      // Иначе используем стандартное скачивание
      if ('showSaveFilePicker' in window) {
        try {
          const fileHandle = await (window as any).showSaveFilePicker({
            suggestedName: filename,
            types: [{
              description: 'CSV файлы',
              accept: { 'text/csv': ['.csv'] },
            }],
          })
          
          const writable = await fileHandle.createWritable()
          await writable.write(blob)
          await writable.close()
          
          toast.dismiss(exportToast)
          toast.success('Экспорт завершен', {
            description: `Файл сохранен: ${filename} (${total.toLocaleString('ru-RU')} записей)`,
            duration: 5000,
          })
        } catch (saveErr: any) {
          // Пользователь отменил выбор файла или произошла ошибка
          if (saveErr.name !== 'AbortError') {
            // Если не отмена, используем стандартное скачивание
            const url = window.URL.createObjectURL(blob)
            const link = document.createElement('a')
            link.href = url
            link.download = filename
            document.body.appendChild(link)
            link.click()
            document.body.removeChild(link)
            window.URL.revokeObjectURL(url)
            
            toast.dismiss(exportToast)
            toast.success('Экспорт завершен', {
              description: `Файл ${filename} скачан`,
              duration: 5000,
            })
          } else {
            toast.dismiss(exportToast)
            toast.info('Экспорт отменен')
          }
        }
      } else {
        // Стандартное скачивание для браузеров без поддержки File System Access API
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = filename
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)
        
        toast.dismiss(exportToast)
        toast.success('Экспорт завершен', {
          description: `Файл ${filename} скачан (${total.toLocaleString('ru-RU')} записей)`,
          duration: 5000,
        })
      }
    } catch (err) {
      toast.error('Ошибка экспорта', {
        description: err instanceof Error ? err.message : 'Не удалось экспортировать данные',
        duration: 5000,
      })
    } finally {
      setExporting(false)
    }
  }

  const handleRefresh = useCallback(() => {
    // Очищаем кэш и загружаем заново
    try {
      sessionStorage.removeItem(CACHE_KEY)
      localStorage.removeItem(STATS_CACHE_KEY)
    } catch (err) {
      console.warn('Failed to clear cache:', err)
    }
    fetchGosts(true)
    fetchStatistics(true)
  }, [fetchGosts, fetchStatistics, CACHE_KEY])

  const handleSortChange = (field: typeof sortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortOrder('asc')
    }
  }

  const handleQuickSearch = async () => {
    if (!quickSearchNumber.trim()) return

    setLoading(true)
    setError(null)

    try {
      const encodedNumber = encodeURIComponent(quickSearchNumber.trim())
      const response = await fetch(`/api/gosts/number/${encodedNumber}`)
      
      if (!response.ok) {
        if (response.status === 404) {
          setError(`ГОСТ с номером "${quickSearchNumber}" не найден`)
        } else {
          const errorData = await response.json().catch(() => ({ error: 'Ошибка поиска' }))
          throw new Error(errorData.error || 'Ошибка поиска')
        }
        return
      }

      const data = await response.json()
      toast.success('ГОСТ найден', {
        description: `Переход к ГОСТу ${data.gost_number}`,
        duration: 2000,
      })
      // Перенаправляем на страницу детального просмотра
      window.location.href = `/gosts/${data.id}`
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Ошибка поиска'
      setError(errorMessage)
      toast.error('ГОСТ не найден', {
        description: errorMessage,
        duration: 4000,
      })
    } finally {
      setLoading(false)
    }
  }

  const breadcrumbItems = [
    { label: 'ГОСТы', href: '/gosts', icon: FileText },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 sm:gap-3 mb-2">
              <motion.h1 
                className="text-2xl sm:text-3xl font-bold flex items-center gap-2"
                initial={{ opacity: 0, y: -20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
              >
                <div className="p-1.5 sm:p-2 rounded-lg bg-blue-100 dark:bg-blue-900/50">
                  <FileText className="w-5 h-5 sm:w-6 sm:h-6 text-blue-600 dark:text-blue-400" />
                </div>
                <span className="truncate">ГОСТы</span>
              </motion.h1>
            </div>
            <motion.p 
              className="text-sm sm:text-base text-muted-foreground mt-1 sm:mt-2"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.1 }}
            >
              Просмотр и поиск ГОСТов из 50 источников Росстандарта
            </motion.p>
          </div>
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.3, delay: 0.2 }}
            className="flex flex-col sm:flex-row gap-2 w-full sm:w-auto"
          >
            {/* Quick Search by Number */}
            <div className="flex items-center gap-2">
              <div className="relative">
                <Hash className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Поиск по номеру..."
                  value={quickSearchNumber}
                  onChange={(e) => setQuickSearchNumber(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleQuickSearch()
                    }
                  }}
                  className="pl-9 pr-9 w-[200px] sm:w-[250px]"
                />
                {quickSearchNumber && (
                  <button
                    onClick={() => setQuickSearchNumber('')}
                    className="absolute right-2 top-1/2 transform -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    aria-label="Очистить"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleQuickSearch}
                disabled={!quickSearchNumber.trim() || loading}
              >
                <Search className="h-4 w-4" />
              </Button>
            </div>
            
            <div className="flex gap-2">
              <Button 
                variant="outline" 
                onClick={handleRefresh}
                disabled={loading}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                Обновить
              </Button>
              <div className="relative group">
                <Button 
                  variant="outline" 
                  onClick={handleExport}
                  disabled={loading || exporting || total === 0}
                  title={total > 0 ? `Экспортировать ${total.toLocaleString('ru-RU')} ГОСТ${total !== 1 ? 'ов' : ''}` : 'Нет данных для экспорта'}
                >
                  {exporting ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Экспорт...
                    </>
                  ) : (
                    <>
                      <Download className="h-4 w-4 mr-2" />
                      Экспорт CSV
                      {total > 0 && (
                        <span className="ml-2 px-1.5 py-0.5 text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded-full">
                          {total.toLocaleString('ru-RU')}
                        </span>
                      )}
                    </>
                  )}
                </Button>
              </div>
              <Button onClick={() => setShowImportDialog(true)}>
                <Upload className="h-4 w-4 mr-2" />
                Импорт ГОСТов
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowFavorites(!showFavorites)}
                className={showFavorites ? 'bg-yellow-50 dark:bg-yellow-950' : ''}
              >
                <Star className={`h-4 w-4 mr-2 ${showFavorites ? 'fill-current text-yellow-500' : ''}`} />
                Избранное
                {getFavorites().length > 0 && (
                  <span className="ml-2 px-1.5 py-0.5 text-xs bg-yellow-500 text-white rounded-full">
                    {getFavorites().length}
                  </span>
                )}
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowHistory(!showHistory)}
                className={showHistory ? 'bg-blue-50 dark:bg-blue-950/50' : ''}
              >
                <Clock className={`h-4 w-4 mr-2 ${showHistory ? 'text-blue-600 dark:text-blue-300' : ''}`} />
                История
                {(historyItems || []).length > 0 && (
                  <span className="ml-2 px-1.5 py-0.5 text-xs bg-blue-600 text-white rounded-full">
                    {(historyItems || []).length}
                  </span>
                )}
              </Button>
              <Link href="/gosts/debug">
                <Button variant="outline" title="Отладка и проверка данных">
                  <Database className="h-4 w-4 mr-2" />
                  Отладка
                </Button>
              </Link>
            </div>
          </motion.div>
        </div>
      </FadeIn>

      {/* Favorites Section */}
      {showFavorites && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Star className="h-5 w-5 text-yellow-500 fill-current" />
              Избранные ГОСТы
            </CardTitle>
            <CardDescription>
              Ваши сохраненные ГОСТы
            </CardDescription>
          </CardHeader>
          <CardContent>
            {getFavorites().length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <Star className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Нет избранных ГОСТов</p>
                <p className="text-sm mt-2">Добавьте ГОСТы в избранное, нажав на звездочку</p>
              </div>
            ) : (
              <div className="space-y-2">
                {getFavorites().map((fav: FavoriteGost) => (
                  <div
                    key={fav.id}
                    className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted transition-colors"
                  >
                    <div className="flex-1 min-w-0">
                      <Link href={`/gosts/${fav.id}`} className="block">
                        <p className="font-mono font-medium">{fav.gost_number}</p>
                        <p className="text-sm text-muted-foreground truncate">{fav.title}</p>
                      </Link>
                    </div>
                    <div className="flex items-center gap-2">
                      <Link href={`/gosts/${fav.id}`}>
                        <Button variant="outline" size="sm">
                          Открыть
                        </Button>
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* History Section */}
      {showHistory && (
        <Card>
          <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Clock className="h-5 w-5 text-blue-600 dark:text-blue-300" />
                Недавние просмотры
              </CardTitle>
              <CardDescription>Последние ГОСТы, которые вы открывали</CardDescription>
            </div>
            {(historyItems || []).length > 0 && (
              <Button variant="ghost" size="sm" onClick={handleClearHistory}>
                Очистить
              </Button>
            )}
          </CardHeader>
          <CardContent>
            {(historyItems || []).length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>История просмотров пуста</p>
                <p className="text-sm mt-2">Открывайте ГОСТы, чтобы быстро возвращаться к ним позже</p>
              </div>
            ) : (
              <div className="space-y-2">
                {(historyItems || []).slice(0, 8).map((item) => (
                  <div
                    key={`${item.id}-${item.viewedAt}`}
                    className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted transition-colors"
                  >
                    <div className="flex-1 min-w-0">
                      <Link href={`/gosts/${item.id}`} className="block">
                        <p className="font-mono font-medium truncate">{item.gost_number}</p>
                        <p className="text-sm text-muted-foreground truncate">{item.title}</p>
                      </Link>
                      <p className="text-xs text-muted-foreground mt-1">{formatViewedAt(item.viewedAt)}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <Link href={`/gosts/${item.id}`}>
                        <Button variant="outline" size="sm">
                          Открыть
                        </Button>
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Statistics */}
      {statistics && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-semibold">Статистика базы данных</h2>
            <Link href="/gosts/debug">
              <Button variant="ghost" size="sm">
                <Database className="h-4 w-4 mr-2" />
                Подробная информация
              </Button>
            </Link>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {statistics.total_gosts !== undefined && (
              <StatCard
                title="Всего ГОСТов"
                value={statistics.total_gosts}
                icon={FileText}
                formatValue={(val) => val.toLocaleString('ru-RU')}
                description="Загружено в базу данных"
              />
            )}
            {statistics.total_sources !== undefined && (
              <StatCard
                title="Источников"
                value={statistics.total_sources}
                icon={BarChart3}
                variant="primary"
                formatValue={(val) => val.toLocaleString('ru-RU')}
              />
            )}
            {statistics.total_documents !== undefined && (
              <StatCard
                title="Документов"
                value={statistics.total_documents}
                icon={FileText}
                formatValue={(val) => val.toLocaleString('ru-RU')}
              />
            )}
            {statistics.by_status && (
              <StatCard
                title="Действующих"
                value={statistics.by_status['действующий'] || statistics.by_status['действует'] || 0}
                icon={BarChart3}
                formatValue={(val) => val.toLocaleString('ru-RU')}
              />
            )}
          </div>

          {/* Charts */}
          {statistics && (statistics.by_status || statistics.by_source_type) && (
            <GostStatisticsChart statistics={statistics} />
          )}
        </div>
      )}

      {/* Filters */}
      <Card>
        <CardHeader className="space-y-3">
          <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-3">
            <div>
              <CardTitle>Поиск и фильтры</CardTitle>
              <CardDescription>
                Найдите ГОСТ по номеру, названию или ключевым словам
              </CardDescription>
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              {hasActiveFilters && (
                <Badge variant="outline" className="text-xs">
                  Фильтров: {activeFiltersCount}
                </Badge>
              )}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setFiltersCollapsed((prev) => !prev)}
                className="flex items-center gap-2"
              >
                <Filter className="h-3.5 w-3.5" />
                {filtersCollapsed ? 'Показать фильтры' : 'Скрыть фильтры'}
                <ChevronDown
                  className={`h-4 w-4 transition-transform ${filtersCollapsed ? '' : 'rotate-180'}`}
                />
              </Button>
            </div>
          </div>
          {filtersCollapsed && hasActiveFilters && (
            <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
              {searchQuery && (
                <Badge variant="secondary" className="text-xs">
                  Поиск: {searchQuery}
                </Badge>
              )}
              {statusFilter && (
                <Badge variant="secondary" className="text-xs">
                  Статус: {statusFilter}
                </Badge>
              )}
              {sourceTypeFilter && (
                <Badge variant="secondary" className="text-xs">
                  Источник: {sourceTypeFilter}
                </Badge>
              )}
            {adoptionFrom && (
              <Badge variant="secondary" className="text-xs">
                Принятия с: {adoptionFrom}
              </Badge>
            )}
            {adoptionTo && (
              <Badge variant="secondary" className="text-xs">
                Принятия до: {adoptionTo}
              </Badge>
            )}
            {effectiveFrom && (
              <Badge variant="secondary" className="text-xs">
                Вступления с: {effectiveFrom}
              </Badge>
            )}
            {effectiveTo && (
              <Badge variant="secondary" className="text-xs">
                Вступления до: {effectiveTo}
              </Badge>
            )}
            </div>
          )}
        </CardHeader>
        <Collapsible open={!filtersCollapsed}>
          <CollapsibleContent>
            <CardContent>
              <GostFilters
                searchQuery={searchQuery}
                onSearchChange={setSearchQuery}
                statusFilter={statusFilter}
                onStatusFilterChange={setStatusFilter}
                sourceTypeFilter={sourceTypeFilter}
                onSourceTypeFilterChange={setSourceTypeFilter}
                sourceTypes={sourceTypes}
                onClearFilters={handleClearFilters}
                hasActiveFilters={hasActiveFilters}
              adoptionFrom={adoptionFrom}
              adoptionTo={adoptionTo}
              onAdoptionFromChange={setAdoptionFrom}
              onAdoptionToChange={setAdoptionTo}
              effectiveFrom={effectiveFrom}
              effectiveTo={effectiveTo}
              onEffectiveFromChange={setEffectiveFrom}
              onEffectiveToChange={setEffectiveTo}
              />
            </CardContent>
          </CollapsibleContent>
        </Collapsible>
      </Card>

      {/* Results */}
      <div className="space-y-4">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 sm:gap-4">
          <div className="flex flex-col gap-1">
            <p className="text-sm text-muted-foreground">
              {total > 0 ? (
                <>
                  Найдено: <span className="font-semibold text-foreground">{total.toLocaleString('ru-RU')}</span> ГОСТ{total !== 1 ? 'ов' : ''}
                  {hasActiveFilters && (
                    <span className="ml-2 text-xs">
                      (с учетом фильтров)
                    </span>
                  )}
                </>
              ) : (
                <span>ГОСТы не найдены</span>
              )}
            </p>
            {total > 0 && (
              <p className="text-xs text-muted-foreground">
                Показано {((currentPage - 1) * ITEMS_PER_PAGE + 1).toLocaleString('ru-RU')} - {Math.min(currentPage * ITEMS_PER_PAGE, total).toLocaleString('ru-RU')} из {total.toLocaleString('ru-RU')}
                {totalPages > 1 && (
                  <span className="ml-2">(страница {currentPage} из {totalPages})</span>
                )}
              </p>
            )}
          </div>
          
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 sm:gap-2 w-full sm:w-auto">
            <Select
              value={sortField || 'gost_number'}
              onValueChange={(value) => handleSortChange(value as typeof sortField)}
            >
              <SelectTrigger className="w-full sm:w-[200px]">
                <SelectValue placeholder="Сортировка" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="gost_number">
                  <div className="flex items-center gap-2">
                    <span>По номеру</span>
                    {sortField === 'gost_number' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
                <SelectItem value="adoption_date">
                  <div className="flex items-center gap-2">
                    <span>По дате принятия</span>
                    {sortField === 'adoption_date' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
                <SelectItem value="effective_date">
                  <div className="flex items-center gap-2">
                    <span>По дате вступления</span>
                    {sortField === 'effective_date' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
                <SelectItem value="status">
                  <div className="flex items-center gap-2">
                    <span>По статусу</span>
                    {sortField === 'status' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
            
            <div className="flex items-center gap-1 border rounded-md p-1">
              <Button
                variant={viewMode === 'grid' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('grid')}
                className="h-8 px-3"
                aria-label="Сетка"
              >
                <Grid3x3 className="h-4 w-4" />
              </Button>
              <Button
                variant={viewMode === 'list' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('list')}
                className="h-8 px-3"
                aria-label="Список"
              >
                <List className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        <GostList gosts={gosts} loading={loading} error={error} viewMode={viewMode} />

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex flex-col sm:flex-row items-center justify-center gap-2 sm:gap-2">
            <div className="flex items-center gap-1 sm:gap-2 w-full sm:w-auto justify-center">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                disabled={currentPage === 1 || loading}
                className="flex-1 sm:flex-initial"
              >
                <ChevronLeft className="h-4 w-4 sm:mr-1" />
                <span className="hidden sm:inline">Назад</span>
              </Button>
              <div className="flex items-center gap-1 overflow-x-auto max-w-full sm:max-w-none">
                {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                  let pageNum: number
                  if (totalPages <= 5) {
                    pageNum = i + 1
                  } else if (currentPage <= 3) {
                    pageNum = i + 1
                  } else if (currentPage >= totalPages - 2) {
                    pageNum = totalPages - 4 + i
                  } else {
                    pageNum = currentPage - 2 + i
                  }
                  return (
                    <Button
                      key={pageNum}
                      variant={currentPage === pageNum ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setCurrentPage(pageNum)}
                      disabled={loading}
                      className="min-w-[2.5rem]"
                    >
                      {pageNum}
                    </Button>
                  )
                })}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                disabled={currentPage === totalPages || loading}
                className="flex-1 sm:flex-initial"
              >
                <span className="hidden sm:inline">Вперед</span>
                <ChevronRight className="h-4 w-4 sm:ml-1" />
              </Button>
            </div>
            <div className="text-xs sm:text-sm text-muted-foreground text-center sm:text-left">
              Страница {currentPage} из {totalPages}
            </div>
          </div>
        )}
      </div>

      {/* Import Dialog */}
      <GostImportDialog
        open={showImportDialog}
        onOpenChange={setShowImportDialog}
        onImportSuccess={() => {
          fetchGosts()
          fetchStatistics()
        }}
      />
    </div>
  )
}

export default function GostsPage() {
  return (
    <Suspense fallback={<div className="flex items-center justify-center min-h-screen"><div className="text-lg">Загрузка...</div></div>}>
      <GostsPageContent />
    </Suspense>
  )
}
