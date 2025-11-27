'use client'

import { useState, useEffect, useCallback, useRef, Suspense } from 'react'
import { formatDateTime } from '@/lib/locale'
import Link from 'next/link'
import { useRouter, useSearchParams, usePathname } from 'next/navigation'
import { DatabaseSelector } from '@/components/database-selector'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { ErrorState } from "@/components/common/error-state"
import { EmptyState } from "@/components/common/empty-state"
import { FadeIn } from "@/components/animations/fade-in"
import { StaggerContainer, StaggerItem } from "@/components/animations/stagger-container"
import { motion } from "framer-motion"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { EyeOpenIcon } from "@radix-ui/react-icons"
import { ConfidenceBadge } from "@/components/results/confidence-badge"
import { ProcessingLevelBadge } from "@/components/results/processing-level-badge"
import { QuickViewModal } from "@/components/results/quick-view-modal"
import { KpvedBadge } from "@/components/results/kpved-badge"
import { KpvedHierarchySelector } from "@/components/results/kpved-hierarchy-selector"
import { TableSkeleton } from "@/components/results/table-skeleton"
import { handleApiError } from "@/lib/errors"
import { cache as ClientCache } from "@/lib/cache"
import { Pagination } from "@/components/ui/pagination"
import { DataTable, type Column } from "@/components/common/data-table"
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { BarChart3, Download, FileSpreadsheet, FileCode, FileJson, RefreshCw, Loader2, Zap, TrendingUp, Filter, Star, StarOff, Save, Bookmark, X } from "lucide-react"
import { exportGroupsToCSV, exportGroupsToJSON, exportGroupsToExcel } from "@/lib/export-results"
import { toast } from 'sonner'
import { useMemo } from 'react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

interface Group {
  normalized_name: string
  normalized_reference: string
  category: string
  merged_count: number
  avg_confidence?: number
  processing_level?: string
  kpved_code?: string
  kpved_name?: string
  kpved_confidence?: number
  last_normalized_at?: string
}

interface GroupDetails {
  normalized_name: string
  normalized_reference: string
  category: string
  merged_count: number
  items: Array<{
    id: number
    source_reference: string
    source_name: string
    code: string
  }>
}

interface Stats {
  totalGroups: number
  totalItems: number // Количество исправленных элементов
  totalItemsWithAttributes?: number // Количество элементов с извлеченными атрибутами
  categories: Record<string, number>
  mergedItems: number
  last_normalized_at?: string
}

function ResultsPageContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const pathname = usePathname()
  
  const [groups, setGroups] = useState<Group[]>([])
  const [stats, setStats] = useState<Stats | null>(null)
  const [quickViewGroup, setQuickViewGroup] = useState<Group | null>(null)
  const [isQuickViewOpen, setIsQuickViewOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [totalGroups, setTotalGroups] = useState(0)

  // Инициализация из URL параметров
  const [currentPage, setCurrentPage] = useState(() => {
    const page = searchParams.get('page')
    return page ? parseInt(page, 10) : 1
  })
  const [totalPages, setTotalPages] = useState(1)
  const [searchQuery, setSearchQuery] = useState(() => searchParams.get('search') || '')
  const [selectedCategory, setSelectedCategory] = useState<string>(() => searchParams.get('category') || '')
  const [selectedKpvedCode, setSelectedKpvedCode] = useState<string | null>(() => searchParams.get('kpved') || null)
  const [inputValue, setInputValue] = useState(() => searchParams.get('search') || '')
  const [selectedDatabase, setSelectedDatabase] = useState<string>(() => searchParams.get('database') || '')
  const [isExporting, setIsExporting] = useState(false)
  const [exportType, setExportType] = useState<'current' | 'all'>('current')
  const [favoriteGroups, setFavoriteGroups] = useState<Set<string>>(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('results_favorite_groups')
      return saved ? new Set(JSON.parse(saved)) : new Set()
    }
    return new Set()
  })
  
  // Инициализация minConfidence и pageSize из URL или localStorage
  const [minConfidence, setMinConfidence] = useState(() => {
    // Сначала проверяем URL параметры
    const urlMinConfidence = searchParams.get('minConfidence')
    if (urlMinConfidence) {
      const parsed = parseFloat(urlMinConfidence)
      if (!isNaN(parsed) && parsed >= 0 && parsed <= 1) {
        return parsed
      }
    }
    // Затем проверяем localStorage
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('results_min_confidence')
      return saved ? parseFloat(saved) : 0
    }
    return 0
  })
  const [pageSize, setPageSize] = useState(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('results_page_size')
      return saved ? parseInt(saved, 10) : 20
    }
    return 20
  })

  // Refs для горячих клавиш
  const searchInputRef = useRef<HTMLInputElement>(null)
  const exportButtonRef = useRef<HTMLButtonElement>(null)

  const limit = pageSize

  // Тип для пресета фильтров
  interface FilterPreset {
    id: string
    name: string
    description?: string
    minConfidence?: number
    maxConfidence?: number
    searchQuery?: string
    category?: string
    kpvedCode?: string | null
    database?: string
    icon?: string
    isCustom?: boolean
  }

  // Загрузка пользовательских пресетов из localStorage
  const [customPresets, setCustomPresets] = useState<FilterPreset[]>(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('results_custom_presets')
      return saved ? JSON.parse(saved) : []
    }
    return []
  })

  // Быстрые фильтры (presets)
  const filterPresets = useMemo(() => [
    {
      id: 'high-confidence',
      name: 'Высокая уверенность',
      description: 'Группы с уверенностью ≥ 80%',
      minConfidence: 0.8,
      icon: '🎯',
    },
    {
      id: 'medium-confidence',
      name: 'Средняя уверенность',
      description: 'Группы с уверенностью ≥ 50%',
      minConfidence: 0.5,
      icon: '📊',
    },
    {
      id: 'low-confidence',
      name: 'Низкая уверенность',
      description: 'Группы с уверенностью < 50%',
      minConfidence: 0,
      maxConfidence: 0.5,
      icon: '⚠️',
    },
    ...customPresets,
  ], [customPresets])

  // Статистика по отфильтрованным данным
  const filteredStats = useMemo(() => {
    if (!groups.length) return null
    
    const totalItems = groups.reduce((sum, group) => sum + group.merged_count, 0)
    const avgConfidence = groups.reduce((sum, group) => sum + (group.avg_confidence || 0), 0) / groups.length
    const withKpved = groups.filter(g => g.kpved_code).length
    const highConfidence = groups.filter(g => (g.avg_confidence || 0) >= 0.8).length
    
    return {
      totalItems,
      avgConfidence,
      withKpved,
      highConfidence,
      withKpvedPercent: (withKpved / groups.length) * 100,
      highConfidencePercent: (highConfidence / groups.length) * 100,
    }
  }, [groups])

  // Сохранение предпочтений пользователя
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('results_min_confidence', minConfidence.toString())
    }
  }, [minConfidence])

  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('results_page_size', pageSize.toString())
    }
  }, [pageSize])

  // Сохранение всех фильтров в localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const filters = {
        searchQuery,
        selectedCategory,
        selectedKpvedCode,
        selectedDatabase,
        minConfidence,
      }
      localStorage.setItem('results_last_filters', JSON.stringify(filters))
    }
  }, [searchQuery, selectedCategory, selectedKpvedCode, selectedDatabase, minConfidence])

  // Восстановление фильтров при загрузке (только если нет URL параметров)
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const hasUrlParams = searchParams.get('search') || searchParams.get('category') || 
                          searchParams.get('kpved') || searchParams.get('database') || 
                          searchParams.get('minConfidence')
      
      // Восстанавливаем только если нет URL параметров
      if (!hasUrlParams) {
        const saved = localStorage.getItem('results_last_filters')
        if (saved) {
          try {
            const filters = JSON.parse(saved)
            // Восстанавливаем только если значения не установлены из URL
            if (filters.searchQuery && !searchParams.get('search')) {
              setSearchQuery(filters.searchQuery)
              setInputValue(filters.searchQuery)
            }
            if (filters.selectedCategory && !searchParams.get('category')) {
              setSelectedCategory(filters.selectedCategory)
            }
            if (filters.selectedKpvedCode && !searchParams.get('kpved')) {
              setSelectedKpvedCode(filters.selectedKpvedCode)
            }
            if (filters.selectedDatabase && !searchParams.get('database')) {
              setSelectedDatabase(filters.selectedDatabase)
            }
            if (filters.minConfidence !== undefined && !searchParams.get('minConfidence')) {
              setMinConfidence(filters.minConfidence)
            }
          } catch (e) {
            console.error('Failed to restore filters:', e)
          }
        }
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []) // Только при монтировании, игнорируем предупреждения о зависимостях

  // Сохранение пользовательского пресета
  const saveCurrentFiltersAsPreset = useCallback(() => {
    const presetName = prompt('Введите название для сохранения фильтров:')
    if (!presetName || !presetName.trim()) return

    const newPreset: FilterPreset = {
      id: `custom-${Date.now()}`,
      name: presetName.trim(),
      description: `Сохраненные фильтры: ${searchQuery || 'без поиска'}, ${selectedCategory || 'все категории'}, уверенность ≥ ${(minConfidence * 100).toFixed(0)}%`,
      minConfidence,
      searchQuery: searchQuery || undefined,
      category: selectedCategory || undefined,
      kpvedCode: selectedKpvedCode || undefined,
      database: selectedDatabase || undefined,
      isCustom: true,
      icon: '💾',
    }

    const updated = [...customPresets, newPreset]
    setCustomPresets(updated)
    localStorage.setItem('results_custom_presets', JSON.stringify(updated))
    toast.success('Фильтры сохранены', {
      description: `Пресет "${presetName}" добавлен`,
    })
  }, [searchQuery, selectedCategory, selectedKpvedCode, selectedDatabase, minConfidence, customPresets])

  // Удаление пользовательского пресета
  const deleteCustomPreset = useCallback((presetId: string) => {
    const updated = customPresets.filter(p => p.id !== presetId)
    setCustomPresets(updated)
    localStorage.setItem('results_custom_presets', JSON.stringify(updated))
    toast.success('Пресет удален')
  }, [customPresets])

  // Функция для обновления URL
  const updateURL = useCallback((updates: {
    page?: number
    search?: string
    category?: string
    kpved?: string | null
    database?: string
    minConfidence?: number
  }) => {
    const params = new URLSearchParams(searchParams)
    
    if (updates.page !== undefined) {
      if (updates.page === 1) {
        params.delete('page')
      } else {
        params.set('page', updates.page.toString())
      }
    }
    
    if (updates.search !== undefined) {
      if (updates.search === '') {
        params.delete('search')
      } else {
        params.set('search', updates.search)
      }
    }
    
    if (updates.category !== undefined) {
      if (updates.category === '') {
        params.delete('category')
      } else {
        params.set('category', updates.category)
      }
    }
    
    if (updates.kpved !== undefined) {
      if (updates.kpved === null || updates.kpved === '') {
        params.delete('kpved')
      } else {
        params.set('kpved', updates.kpved)
      }
    }
    
    if (updates.database !== undefined) {
      if (updates.database === '') {
        params.delete('database')
      } else {
        params.set('database', updates.database)
      }
    }
    
    if (updates.minConfidence !== undefined) {
      if (updates.minConfidence === 0 || updates.minConfidence === undefined) {
        params.delete('minConfidence')
      } else {
        params.set('minConfidence', updates.minConfidence.toString())
      }
    }
    
    const newURL = params.toString() ? `${pathname}?${params.toString()}` : pathname
    router.replace(newURL, { scroll: false })
  }, [searchParams, pathname, router])

  // Применение быстрого фильтра (перемещено сюда, чтобы использовать updateURL)
  const applyPreset = useCallback((preset: FilterPreset) => {
    if (preset.minConfidence !== undefined) {
      setMinConfidence(preset.minConfidence)
    }
    if (preset.searchQuery !== undefined) {
      setSearchQuery(preset.searchQuery)
      setInputValue(preset.searchQuery)
    }
    if (preset.category !== undefined) {
      setSelectedCategory(preset.category)
    }
    if (preset.kpvedCode !== undefined) {
      setSelectedKpvedCode(preset.kpvedCode)
    }
    if (preset.database !== undefined) {
      setSelectedDatabase(preset.database)
    }
    setCurrentPage(1)
    updateURL({ 
      page: 1,
      search: preset.searchQuery || '',
      category: preset.category || '',
      kpved: preset.kpvedCode || null,
      database: preset.database || '',
      minConfidence: preset.minConfidence || 0,
    })
    toast.success(`Применен фильтр: ${preset.name}`, {
      description: preset.description,
    })
  }, [updateURL])

  const fetchStats = async () => {
    // Проверяем кеш сначала
    const cachedStats = ClientCache.get<Stats>('normalization_stats')
    if (cachedStats) {
      setStats(cachedStats)
      return
    }

    try {
      const response = await fetch('/api/normalization/stats')
      const data = await response.json()
      setStats(data)
      // Кешируем на 5 минут
      ClientCache.set('normalization_stats', data, 5 * 60 * 1000)
    } catch (error) {
      console.error('Error fetching stats:', error)
      // Устанавливаем пустую статистику при ошибке
      setStats({
        totalGroups: 0,
        totalItems: 0,
        totalItemsWithAttributes: 0,
        mergedItems: 0,
        categories: {},
      })
    }
  }

  const fetchGroups = useCallback(async (retryCount = 0) => {
    setIsLoading(true)
    setError(null)
    try {
      // Если используется фильтр по уверенности, загружаем больше данных для корректной фильтрации
      const effectiveLimit = minConfidence > 0 ? 1000 : limit
      const effectivePage = minConfidence > 0 ? 1 : currentPage
      
      const params = new URLSearchParams({
        page: effectivePage.toString(),
        limit: effectiveLimit.toString(),
        include_ai: 'true',
      })

      if (searchQuery) {
        params.append('search', searchQuery)
      }

      if (selectedCategory) {
        params.append('category', selectedCategory)
      }

      if (selectedKpvedCode) {
        params.append('kpved_code', selectedKpvedCode)
      }

      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 30000) // 30 секунд таймаут

      try {
        const response = await fetch(`/api/normalization/groups?${params}`, {
          signal: controller.signal,
        })

        clearTimeout(timeoutId)

        if (!response.ok) {
          // Если это ошибка сервера (5xx), пробуем повторить
          if (response.status >= 500 && retryCount < 2) {
            await new Promise(resolve => setTimeout(resolve, 1000 * (retryCount + 1))) // Экспоненциальная задержка
            return fetchGroups(retryCount + 1)
          }
          throw new Error(`Failed to fetch groups: ${response.status}`)
        }

        const data = await response.json()
        let filteredGroups = data.groups || []

        // Фильтрация по минимальной уверенности на фронтенде
        if (minConfidence > 0) {
          filteredGroups = filteredGroups.filter((group: Group) => {
            const confidence = group.avg_confidence || 0
            return confidence >= minConfidence
          })
          // Пересчитываем пагинацию для отфильтрованных данных
          const filteredTotal = filteredGroups.length
          const calculatedTotalPages = Math.ceil(filteredTotal / limit) || 1
          setTotalPages(calculatedTotalPages)
          setTotalGroups(filteredTotal)
          
          // Применяем пагинацию к отфильтрованным данным
          const startIndex = (currentPage - 1) * limit
          const endIndex = startIndex + limit
          filteredGroups = filteredGroups.slice(startIndex, endIndex)
          
          // Если текущая страница больше доступных страниц, переходим на первую
          if (currentPage > calculatedTotalPages && calculatedTotalPages > 0) {
            setCurrentPage(1)
            updateURL({ page: 1 })
          }
        } else {
          setTotalPages(data.totalPages || 1)
          setTotalGroups(data.total || 0)
        }

        setGroups(filteredGroups)
      } catch (fetchError) {
        clearTimeout(timeoutId)
        if (fetchError instanceof Error && fetchError.name === 'AbortError') {
          throw new Error('Время ожидания истекло. Попробуйте еще раз.')
        }
        throw fetchError
      }
    } catch (error) {
      console.error('Error fetching groups:', error)
      const errorMessage = handleApiError(error, 'LOAD_GROUPS_ERROR')
      setError(errorMessage)
      setGroups([])
      
      // Показываем toast только при последней попытке
      if (retryCount >= 2) {
        toast.error('Ошибка загрузки данных', {
          description: errorMessage,
          action: {
            label: 'Повторить',
            onClick: () => fetchGroups(0),
          },
        })
      }
    } finally {
      setIsLoading(false)
    }
  }, [currentPage, searchQuery, selectedCategory, selectedKpvedCode, minConfidence, pageSize, limit, updateURL])

  // Синхронизация состояния из URL при изменении параметров
  useEffect(() => {
    const page = searchParams.get('page')
    const search = searchParams.get('search')
    const category = searchParams.get('category')
    const kpved = searchParams.get('kpved')
    const database = searchParams.get('database')

    if (page) {
      const pageNum = parseInt(page, 10)
      if (pageNum !== currentPage && pageNum > 0) {
        setCurrentPage(pageNum)
      }
    } else if (currentPage !== 1) {
      setCurrentPage(1)
    }

    const newSearch = search || ''
    if (newSearch !== searchQuery) {
      setSearchQuery(newSearch)
      setInputValue(newSearch)
    }

    const newCategory = category || ''
    if (newCategory !== selectedCategory) {
      setSelectedCategory(newCategory)
    }

    const newKpved = kpved || null
    if (newKpved !== selectedKpvedCode) {
      setSelectedKpvedCode(newKpved)
    }

    const newDatabase = database || ''
    if (newDatabase !== selectedDatabase) {
      setSelectedDatabase(newDatabase)
    }
  }, [searchParams, currentPage, searchQuery, selectedCategory, selectedKpvedCode, selectedDatabase])

  // Загрузка статистики
  useEffect(() => {
    fetchStats()
  }, [])

  // Debounced search - автоматический поиск при вводе с задержкой
  useEffect(() => {
    const timer = setTimeout(() => {
      if (inputValue !== searchQuery) {
        setSearchQuery(inputValue)
        setCurrentPage(1)
        updateURL({ search: inputValue, page: 1 })
      }
    }, 500) // 500ms debounce delay

    return () => clearTimeout(timer)
  }, [inputValue, searchQuery, updateURL])

  // Загрузка групп при изменении фильтров или страницы
  useEffect(() => {
    fetchGroups()
  }, [fetchGroups])

  const handleRowClick = useCallback((group: Group) => {
    try {
      const encodedName = encodeURIComponent(group.normalized_name)
      const encodedCategory = encodeURIComponent(group.category)
      const url = `/results/groups/${encodedName}/${encodedCategory}`

      // Check URL length to prevent issues with very long URLs
      if (url.length > 2000) {
        console.warn('URL is too long, may cause issues in some browsers')
        toast.warning('Название группы слишком длинное для перехода')
        return
      }

      router.push(url)
    } catch (error) {
      console.error('Failed to navigate to group detail:', error)
      const errorMessage = error instanceof Error ? error.message : 'Не удалось перейти к детальной странице'
      setError(errorMessage)
      toast.error('Ошибка навигации', {
        description: errorMessage,
      })
    }
  }, [router])

  const handleQuickView = useCallback((group: Group, e: React.MouseEvent) => {
    e.stopPropagation()
    setQuickViewGroup(group)
    setIsQuickViewOpen(true)
  }, [])

  const handleSearch = useCallback(() => {
    setSearchQuery(inputValue)
    setCurrentPage(1)
    updateURL({ search: inputValue, page: 1 })
  }, [inputValue, updateURL])

  const handleCategoryChange = useCallback((value: string) => {
    const category = value === 'all' ? '' : value
    setSelectedCategory(category)
    setCurrentPage(1)
    updateURL({ category, page: 1 })
  }, [updateURL])

  const handleKpvedChange = useCallback((value: string | null) => {
    setSelectedKpvedCode(value)
    setCurrentPage(1)
    updateURL({ kpved: value, page: 1 })
  }, [updateURL])

  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page)
    updateURL({ page })
    // Прокрутка вверх при смене страницы
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }, [updateURL])

  const handleDatabaseChange = useCallback((db: string) => {
    setSelectedDatabase(db)
    setCurrentPage(1)
    updateURL({ database: db, page: 1 })
  }, [updateURL])

  const handleKeyPress = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }, [handleSearch])

  // Мемоизация списка категорий из статистики
  const categories = useMemo(() => {
    return stats?.categories ? Object.keys(stats.categories).sort() : []
  }, [stats?.categories])

  // Управление избранными группами
  const toggleFavorite = useCallback((group: Group, e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation()
    }
    const groupKey = `${group.normalized_name}|${group.category}`
    setFavoriteGroups(prev => {
      const newSet = new Set(prev)
      if (newSet.has(groupKey)) {
        newSet.delete(groupKey)
        toast.success('Группа удалена из избранного')
      } else {
        newSet.add(groupKey)
        toast.success('Группа добавлена в избранное')
      }
      
      // Сохраняем в localStorage
      if (typeof window !== 'undefined') {
        localStorage.setItem('results_favorite_groups', JSON.stringify(Array.from(newSet)))
      }
      
      return newSet
    })
  }, [])

  // Визуализация распределения уверенности
  const confidenceDistribution = useMemo(() => {
    if (!groups.length) return null
    
    const ranges = [
      { label: '0-20%', min: 0, max: 0.2, color: 'bg-red-500' },
      { label: '20-40%', min: 0.2, max: 0.4, color: 'bg-orange-500' },
      { label: '40-60%', min: 0.4, max: 0.6, color: 'bg-yellow-500' },
      { label: '60-80%', min: 0.6, max: 0.8, color: 'bg-blue-500' },
      { label: '80-100%', min: 0.8, max: 1.0, color: 'bg-green-500' },
    ]
    
    return ranges.map(range => {
      const count = groups.filter(g => {
        const conf = g.avg_confidence || 0
        return conf >= range.min && conf < range.max
      }).length
      return {
        ...range,
        count,
        percentage: (count / groups.length) * 100,
      }
    })
  }, [groups])

  // Мемоизация колонок таблицы
  const tableColumns = useMemo(() => [
    {
      key: 'normalized_name',
      header: 'Нормализованное название',
      accessor: (row: Group) => row.normalized_name,
      render: (row: Group) => {
        const groupKey = `${row.normalized_name}|${row.category}`
        const isFavorite = favoriteGroups.has(groupKey)
        return (
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 shrink-0"
              onClick={(e) => toggleFavorite(row, e)}
              title={isFavorite ? 'Удалить из избранного' : 'Добавить в избранное'}
              aria-label={isFavorite ? 'Удалить из избранного' : 'Добавить в избранное'}
            >
              {isFavorite ? (
                <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
              ) : (
                <StarOff className="h-4 w-4 text-muted-foreground hover:text-yellow-400" />
              )}
            </Button>
            <span className="font-medium">{row.normalized_name}</span>
          </div>
        )
      },
      sortable: true,
    },
    {
      key: 'normalized_reference',
      header: 'Нормализованный reference',
      accessor: (row: Group) => row.normalized_reference,
      render: (row: Group) => (
        <span className="text-sm text-muted-foreground">{row.normalized_reference}</span>
      ),
      sortable: true,
    },
    {
      key: 'category',
      header: 'Категория',
      accessor: (row: Group) => row.category,
      render: (row: Group) => <Badge variant="secondary">{row.category}</Badge>,
      sortable: true,
    },
    {
      key: 'kpved_code',
      header: 'КПВЭД',
      accessor: (row: Group) => row.kpved_code || '',
      render: (row: Group) => (
        <KpvedBadge
          code={row.kpved_code}
          name={row.kpved_name}
          confidence={row.kpved_confidence}
          showConfidence={true}
        />
      ),
      sortable: true,
    },
    {
      key: 'avg_confidence',
      header: 'AI Confidence',
      accessor: (row: Group) => row.avg_confidence || 0,
      render: (row: Group) => (
        <ConfidenceBadge confidence={row.avg_confidence} size="sm" showTooltip={false} />
      ),
      sortable: true,
    },
    {
      key: 'processing_level',
      header: 'Processing',
      accessor: (row: Group) => row.processing_level || '',
      render: (row: Group) => (
        <ProcessingLevelBadge level={row.processing_level} showTooltip={false} />
      ),
      sortable: true,
    },
    {
      key: 'merged_count',
      header: 'Элементов',
      accessor: (row: Group) => row.merged_count,
      render: (row: Group) => <span className="text-right">{row.merged_count}</span>,
      align: 'right' as const,
      sortable: true,
    },
    {
      key: 'actions',
      header: 'Действия',
      render: (row: Group) => {
        const groupKey = `${row.normalized_name}|${row.category}`
        const isFavorite = favoriteGroups.has(groupKey)
        return (
          <div className="text-right flex items-center justify-end gap-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={(e) => toggleFavorite(row, e)}
              title={isFavorite ? 'Удалить из избранного' : 'Добавить в избранное'}
              className={isFavorite ? 'text-yellow-500 hover:text-yellow-600' : ''}
            >
              {isFavorite ? (
                <Star className="h-4 w-4 fill-current" />
              ) : (
                <StarOff className="h-4 w-4" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={(e) => {
                e.stopPropagation()
                handleQuickView(row, e)
              }}
              title="Быстрый просмотр группы"
              aria-label={`Быстрый просмотр группы ${row.normalized_name}`}
            >
              <EyeOpenIcon className="h-4 w-4" aria-hidden="true" />
            </Button>
          </div>
        )
      },
      align: 'right' as const,
      sortable: false,
    },
  ], [handleQuickView, favoriteGroups, toggleFavorite])

  const breadcrumbItems = [
    { label: 'Результаты', href: '/results', icon: BarChart3 },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>
      {/* Заголовок */}
      <FadeIn>
        <div className="flex items-center justify-between">
          <div>
            <motion.h1 
              className="text-3xl font-bold"
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5 }}
            >
              Результаты нормализации
            </motion.h1>
            <motion.p 
              className="text-muted-foreground"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.1 }}
            >
              Просмотр нормализованных данных по группам
            </motion.p>
          </div>
          <div className="flex gap-2 items-center">
            <DatabaseSelector
              value={selectedDatabase}
              onChange={handleDatabaseChange}
              className="w-[250px]"
              placeholder="Выберите БД для обработки"
            />
            <Button asChild>
              <Link href={selectedDatabase ? `/processes?tab=normalization&database=${encodeURIComponent(selectedDatabase)}` : '/processes?tab=normalization'}>
                Запустить нормализацию
              </Link>
            </Button>
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button asChild variant="outline">
                <Link href="/normalization">
                  Назад к нормализации
                </Link>
              </Button>
            </motion.div>
          </div>
        </div>
      </FadeIn>

      {/* Статистика */}
      <StaggerContainer className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StaggerItem>
          <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Исправлено элементов</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(stats?.totalItems ?? 0).toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  элементов с разложенными атрибутами
                </p>
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>

        <StaggerItem>
          <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">С атрибутами</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(stats?.totalItemsWithAttributes ?? 0).toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  элементов с извлеченными размерами/брендами
                </p>
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>

        <StaggerItem>
          <motion.div whileHover={{ scale: 1.02 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Объединено</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(stats?.mergedItems ?? 0).toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  дубликатов найдено и объединено
                </p>
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>
      </StaggerContainer>

      {/* Проверка: база данных не была обработана */}
      {stats && stats.totalItems === 0 && stats.totalGroups === 0 && !isLoading && (
        <Card className="border-amber-200 bg-amber-50/50">
          <CardContent className="pt-6">
            <div className="flex items-start gap-4">
              <div className="rounded-full bg-amber-100 p-2">
                <RefreshCw className="h-5 w-5 text-amber-600 animate-spin" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-amber-900 mb-1">
                  База данных не была обработана
                </h3>
                <p className="text-sm text-amber-800">
                  По выбранной базе данных еще не было обработано элементов. 
                  Пожалуйста, запустите нормализацию и ожидайте завершения обработки.
                </p>
                <div className="mt-4">
                  <Button asChild>
                    <Link href={selectedDatabase ? `/processes?tab=normalization&database=${encodeURIComponent(selectedDatabase)}` : '/processes?tab=normalization'}>
                      Запустить нормализацию
                    </Link>
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Информация о последней нормализации */}
      {stats?.last_normalized_at && stats.totalItems > 0 && (
        <Card>
          <CardContent className="pt-6">
            <div className="text-sm text-muted-foreground">
              <span className="font-medium">Последняя нормализация: </span>
              <span>
                {formatDateTime(stats.last_normalized_at, {
                  day: '2-digit',
                  month: '2-digit',
                  year: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Фильтры */}
      <Card>
        <CardHeader>
          <CardTitle>Поиск и фильтрация</CardTitle>
          <CardDescription>
            Найдите группы по названию или отфильтруйте по категории
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4">
            <div className="flex-1 flex gap-2">
              <Input
                placeholder="Поиск по нормализованному названию..."
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                onKeyPress={handleKeyPress}
                aria-label="Поиск по нормализованному названию"
              />
              <Button onClick={handleSearch} aria-label="Выполнить поиск">Найти</Button>
            </div>
            <Select value={selectedCategory || 'all'} onValueChange={handleCategoryChange}>
              <SelectTrigger className="w-[200px]" aria-label="Фильтр по категориям">
                <SelectValue placeholder="Все категории" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все категории</SelectItem>
                {categories.map(category => (
                  <SelectItem key={category} value={category}>
                    {category} ({stats?.categories[category]})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <KpvedHierarchySelector
              value={selectedKpvedCode || undefined}
              onChange={handleKpvedChange}
              placeholder="Фильтр по КПВЭД..."
            />
            <div className="flex items-center gap-2">
              <label htmlFor="min-confidence" className="text-sm text-muted-foreground whitespace-nowrap">
                Мин. уверенность:
              </label>
              <Input
                id="min-confidence"
                type="number"
                min="0"
                max="1"
                step="0.1"
                value={minConfidence}
                onChange={(e) => {
                  const value = parseFloat(e.target.value) || 0
                  const clampedValue = Math.max(0, Math.min(1, value))
                  setMinConfidence(clampedValue)
                  setCurrentPage(1)
                }}
                className="w-20"
                aria-label="Минимальная уверенность AI (0-1)"
                title="Минимальная уверенность AI для фильтрации групп"
              />
            </div>
          </div>
          {(searchQuery || selectedCategory || selectedKpvedCode || minConfidence > 0) && (
            <div className="mt-4 flex items-center gap-2">
              <span className="text-sm text-muted-foreground">Активные фильтры:</span>
              {searchQuery && (
                <Badge variant="secondary">
                  Поиск: {searchQuery}
                  <button
                    className="ml-2 hover:text-destructive"
                    onClick={() => {
                      setSearchQuery('')
                      setInputValue('')
                      setCurrentPage(1)
                      updateURL({ search: '', page: 1 })
                    }}
                    aria-label="Удалить фильтр поиска"
                  >
                    ×
                  </button>
                </Badge>
              )}
              {selectedCategory && (
                <Badge variant="secondary">
                  Категория: {selectedCategory}
                  <button
                    className="ml-2 hover:text-destructive"
                    onClick={() => {
                      setSelectedCategory('')
                      setCurrentPage(1)
                      updateURL({ category: '', page: 1 })
                    }}
                    aria-label="Удалить фильтр категории"
                  >
                    ×
                  </button>
                </Badge>
              )}
              {selectedKpvedCode && (
                <Badge variant="secondary">
                  КПВЭД: {selectedKpvedCode}
                  <button
                    className="ml-2 hover:text-destructive"
                    onClick={() => {
                      setSelectedKpvedCode(null)
                      setCurrentPage(1)
                      updateURL({ kpved: null, page: 1 })
                    }}
                    aria-label="Удалить фильтр КПВЭД"
                  >
                    ×
                  </button>
                </Badge>
              )}
              {minConfidence > 0 && (
                <Badge variant="secondary">
                  Уверенность: ≥{(minConfidence * 100).toFixed(0)}%
                  <button
                    className="ml-2 hover:text-destructive"
                    onClick={() => {
                      setMinConfidence(0)
                      setCurrentPage(1)
                    }}
                    aria-label="Удалить фильтр уверенности"
                  >
                    ×
                  </button>
                </Badge>
              )}
            </div>
          )}

          {/* Быстрые фильтры и сохранение */}
          <div className="mt-6 pt-6 border-t">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-semibold flex items-center gap-2">
                  <Bookmark className="h-4 w-4" />
                  Быстрые фильтры
                </h3>
                <p className="text-xs text-muted-foreground mt-1">
                  Примените готовый набор фильтров или сохраните текущие
                </p>
              </div>
              {(searchQuery || selectedCategory || selectedKpvedCode || minConfidence > 0) && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={saveCurrentFiltersAsPreset}
                  className="gap-2"
                >
                  <Save className="h-4 w-4" />
                  Сохранить фильтры
                </Button>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              {filterPresets.map((preset) => (
                <div key={preset.id} className="relative group">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => applyPreset(preset)}
                    className="gap-2"
                    title={preset.description}
                  >
                    {preset.icon && <span>{preset.icon}</span>}
                    <span>{preset.name}</span>
                  </Button>
                  {preset.isCustom && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        if (confirm(`Удалить пресет "${preset.name}"?`)) {
                          deleteCustomPreset(preset.id)
                        }
                      }}
                      className="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-destructive text-destructive-foreground flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-destructive/90"
                      title="Удалить пресет"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Статистика по отфильтрованным данным */}
      {filteredStats && groups.length > 0 && (
        <Card className="border-primary/20 bg-primary/5">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5" />
              Статистика по отфильтрованным данным
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">Всего элементов</p>
                <p className="text-2xl font-bold">{filteredStats.totalItems.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Средняя уверенность</p>
                <p className="text-2xl font-bold">{(filteredStats.avgConfidence * 100).toFixed(1)}%</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">С КПВЭД</p>
                <p className="text-2xl font-bold">
                  {filteredStats.withKpved} ({filteredStats.withKpvedPercent.toFixed(0)}%)
                </p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Высокая уверенность</p>
                <p className="text-2xl font-bold">
                  {filteredStats.highConfidence} ({filteredStats.highConfidencePercent.toFixed(0)}%)
                </p>
              </div>
            </div>
            
            {/* Визуализация распределения уверенности */}
            {confidenceDistribution && (
              <div className="mt-4 pt-4 border-t">
                <p className="text-sm font-medium mb-3">Распределение уверенности</p>
                <div className="space-y-2">
                  {confidenceDistribution.map((range) => (
                    <div key={range.label} className="flex items-center gap-2">
                      <div className="w-20 text-xs text-muted-foreground">{range.label}</div>
                      <div className="flex-1 bg-muted rounded-full h-4 overflow-hidden">
                        <div
                          className={`${range.color} h-full transition-all duration-300`}
                          style={{ width: `${range.percentage}%` }}
                          title={`${range.count} групп (${range.percentage.toFixed(1)}%)`}
                        />
                      </div>
                      <div className="w-16 text-xs text-right text-muted-foreground">
                        {range.count} ({range.percentage.toFixed(0)}%)
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Таблица групп */}
      <Card>
        <CardHeader>
          <div className="flex justify-between items-start">
            <div>
              <CardTitle>Группы товаров</CardTitle>
              <CardDescription>
                Страница {currentPage} из {totalPages} • Всего групп: {totalGroups}
                {filteredStats && (
                  <span className="ml-2 text-xs">
                    • Показано: {groups.length} • Элементов: {filteredStats.totalItems.toLocaleString()}
                  </span>
                )}
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <label htmlFor="page-size" className="text-sm text-muted-foreground whitespace-nowrap">
                На странице:
              </label>
              <Select
                value={pageSize.toString()}
                onValueChange={(value) => {
                  const newSize = parseInt(value, 10)
                  setPageSize(newSize)
                  setCurrentPage(1)
                  updateURL({ page: 1 })
                }}
              >
                <SelectTrigger id="page-size" className="w-20" aria-label="Количество элементов на странице">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="10">10</SelectItem>
                  <SelectItem value="20">20</SelectItem>
                  <SelectItem value="50">50</SelectItem>
                  <SelectItem value="100">100</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {groups.length > 0 && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button 
                    ref={exportButtonRef}
                    variant="outline" 
                    size="sm"
                    disabled={isLoading || isExporting}
                    aria-label="Экспорт данных (Ctrl+E)"
                    title="Экспорт данных (Ctrl+E)"
                  >
                    {isExporting ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Экспорт...
                      </>
                    ) : (
                      <>
                        <Download className="h-4 w-4 mr-2" />
                        Экспорт
                      </>
                    )}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuLabel>Экспорт данных</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={async () => {
                      setIsExporting(true)
                      try {
                        const count = exportType === 'all' ? totalGroups : groups.length
                        await exportGroupsToCSV(
                          groups,
                          selectedDatabase,
                          exportType === 'all',
                          searchQuery,
                          selectedCategory,
                          selectedKpvedCode
                        )
                        toast.success('Экспорт выполнен успешно', {
                          description: `Экспортировано ${count} групп в CSV`,
                        })
                      } catch (error) {
                        console.error('Export error:', error)
                        const errorMessage = error instanceof Error ? error.message : 'Ошибка при экспорте в CSV'
                        setError(errorMessage)
                        toast.error('Ошибка экспорта', {
                          description: errorMessage,
                        })
                      } finally {
                        setIsExporting(false)
                      }
                    }}
                    disabled={isExporting}
                  >
                    <FileCode className="h-4 w-4 mr-2" />
                    CSV ({exportType === 'all' ? 'все' : 'текущая страница'})
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={async () => {
                      setIsExporting(true)
                      try {
                        const count = exportType === 'all' ? totalGroups : groups.length
                        await exportGroupsToJSON(
                          groups,
                          selectedDatabase,
                          exportType === 'all',
                          searchQuery,
                          selectedCategory,
                          selectedKpvedCode
                        )
                        toast.success('Экспорт выполнен успешно', {
                          description: `Экспортировано ${count} групп в JSON`,
                        })
                      } catch (error) {
                        console.error('Export error:', error)
                        const errorMessage = error instanceof Error ? error.message : 'Ошибка при экспорте в JSON'
                        setError(errorMessage)
                        toast.error('Ошибка экспорта', {
                          description: errorMessage,
                        })
                      } finally {
                        setIsExporting(false)
                      }
                    }}
                    disabled={isExporting}
                  >
                    <FileJson className="h-4 w-4 mr-2" />
                    JSON ({exportType === 'all' ? 'все' : 'текущая страница'})
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={async () => {
                      setIsExporting(true)
                      try {
                        const count = exportType === 'all' ? totalGroups : groups.length
                        await exportGroupsToExcel(
                          groups,
                          selectedDatabase,
                          exportType === 'all',
                          searchQuery,
                          selectedCategory,
                          selectedKpvedCode
                        )
                        toast.success('Экспорт выполнен успешно', {
                          description: `Экспортировано ${count} групп в Excel`,
                        })
                      } catch (error) {
                        console.error('Export error:', error)
                        const errorMessage = error instanceof Error ? error.message : 'Ошибка при экспорте в Excel'
                        setError(errorMessage)
                        toast.error('Ошибка экспорта', {
                          description: errorMessage,
                        })
                      } finally {
                        setIsExporting(false)
                      }
                    }}
                    disabled={isExporting}
                  >
                    <FileSpreadsheet className="h-4 w-4 mr-2" />
                    Excel ({exportType === 'all' ? 'все' : 'текущая страница'})
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => setExportType(exportType === 'all' ? 'current' : 'all')}
                  >
                    {exportType === 'all' ? '✓ Экспортировать все' : 'Экспортировать все'}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => setExportType(exportType === 'current' ? 'all' : 'current')}
                  >
                    {exportType === 'current' ? '✓ Текущая страница' : 'Текущая страница'}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {error && !isLoading ? (
            <ErrorState
              title="Ошибка загрузки данных"
              message={error}
              action={{
                label: 'Повторить',
                onClick: fetchGroups,
              }}
              variant="destructive"
            />
          ) : isLoading ? (
            <div role="status" aria-live="polite" aria-label="Загрузка данных">
              <TableSkeleton rows={10} columns={8} />
            </div>
          ) : stats && stats.totalItems === 0 && stats.totalGroups === 0 ? (
            <EmptyState
              icon={RefreshCw}
              title="База данных не была обработана"
              description="По выбранной базе данных еще не было обработано элементов. Запустите нормализацию и ожидайте завершения обработки."
            />
          ) : groups.length === 0 ? (
            <EmptyState
              title="Групп не найдено"
              description={searchQuery || selectedCategory || selectedKpvedCode 
                ? "Попробуйте изменить фильтры поиска" 
                : "Нет данных для отображения. Запустите нормализацию для получения результатов."}
            />
          ) : (
            <>
              <DataTable
                data={groups}
                columns={tableColumns}
                onRowClick={handleRowClick}
                keyExtractor={(row, index) => `${row.normalized_name}-${row.category}-${index}`}
                rowClassName={() => 'cursor-pointer hover:bg-muted/50 transition-colors'}
                emptyMessage="Группы не найдены"
              />

              {/* Пагинация */}
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPageChange={handlePageChange}
                itemsPerPage={limit}
                totalItems={totalGroups}
                className="mt-4"
              />
            </>
          )}
        </CardContent>
      </Card>

      {/* Модальное окно быстрого просмотра */}
      <QuickViewModal
        group={quickViewGroup}
        open={isQuickViewOpen}
        onOpenChange={setIsQuickViewOpen}
      />
    </div>
  )
}

export default function ResultsPage() {
  return (
    <Suspense fallback={
      <div className="container-wide mx-auto px-4 py-8 space-y-6">
        <div className="flex items-center justify-center py-12">
          <div className="text-center">
            <div className="h-8 w-8 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
            <p className="text-sm text-muted-foreground">Загрузка...</p>
          </div>
        </div>
      </div>
    }>
      <ResultsPageContent />
    </Suspense>
  )
}
