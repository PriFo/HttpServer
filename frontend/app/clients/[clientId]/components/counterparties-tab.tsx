'use client'

import { useState, useEffect, useCallback, useMemo, useRef, memo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { 
  Users, 
  Search,
  Eye,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Download,
  Loader2,
  FileSpreadsheet,
  FileCode,
  FileJson,
  RefreshCw,
  Settings,
  Zap,
  Info,
  AlertTriangle
} from "lucide-react"
import { Pagination } from "@/components/ui/pagination"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { LoadingState } from "@/components/common/loading-state"
import { ErrorState } from "@/components/common/error-state"
import { EmptyState } from "@/components/common/empty-state"
import { CounterpartyDetailDialog } from "./counterparty-detail-dialog"
import { CounterpartyDuplicatesDialog, DuplicateGroup } from "./counterparty-duplicates-dialog"
import { Progress } from "@/components/ui/progress"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { toast } from "sonner"

interface DatabaseSource {
  database_id: number
  database_name: string
  source_reference?: string
  source_name?: string
}

interface CounterpartyItem {
  id: number
  uniqueKey?: string // Уникальный ключ для React
  name: string
  normalized_name: string
  tax_id?: string
  kpp?: string
  bin?: string
  type?: string
  status: string
  quality_score?: number
  country?: string
  contact_email?: string
  contact_phone?: string
  contact_person?: string
  legal_address?: string
  postal_address?: string
  source_reference?: string
  source_name?: string
  project_name?: string
  database_name?: string
  source_databases?: DatabaseSource[] // Связанные базы данных
}

interface CounterpartyApiItem {
  id?: number | string
  database_id?: number | string
  project_id?: number | string
  code?: string
  name?: string
  normalized_name?: string
  source_name?: string
  reference?: string
  source?: string
  attributes?: Record<string, unknown> | string
  legal_address?: string
  postal_address?: string
  legal_country?: string
  postal_country?: string
  tax_id?: string
  bin?: string
  kpp?: string
  contact_email?: string
  contact_phone?: string
  contact_person?: string
  quality_score?: number
  source_reference?: string
  project_name?: string
  database_name?: string
  source_databases?: DatabaseSource[]
}

interface CounterpartiesTabProps {
  clientId: string
  projects: Array<{
    id: number
    name: string
    project_type: string
    status: string
  }>
}

type MappingStrategy = 'max_quality' | 'max_databases' | 'max_data' | string

interface MappingConfigState {
  auto_map_counterparties: boolean
  auto_merge_duplicates: boolean
  master_selection_strategy: MappingStrategy
}

interface MappingStatsState {
  total?: number
  total_count?: number
  normalized?: number
  processed?: number
  planned?: number
  with_inn?: number
  with_address?: number
  with_contacts?: number
  enriched?: number
  duplicate_groups?: number
  duplicates_count?: number
  multi_database_count?: number
  average_quality_score?: number
  status?: string
  is_running?: boolean
  [key: string]: number | string | boolean | undefined
}

interface MappingProgressState {
  message?: string
  processed?: number
  total?: number
}

interface MappingStatusResponse {
  stats?: MappingStatsState
  config?: Partial<MappingConfigState>
  auto_map_counterparties?: boolean
  auto_merge_duplicates?: boolean
  master_selection_strategy?: MappingStrategy
  progress?: MappingProgressState | string
  mapping_progress?: MappingProgressState | string
  is_running?: boolean
}

interface CounterpartiesResponse {
  counterparties?: Record<string, unknown>[]
  items?: Record<string, unknown>[]
  data?: Record<string, unknown>[]
  total?: number
  count?: number
}

// Мемоизированный компонент строки таблицы для оптимизации производительности
interface CounterpartyRowProps {
  item: CounterpartyItem
  onView: (item: CounterpartyItem) => void
}

const CounterpartyRow = memo<CounterpartyRowProps>(({ item, onView }) => {
  // Мемоизируем вычисление title для избежания лишних вычислений
  const cellTitle = [
    item.name || 'Без названия',
    item.tax_id && `ИНН/БИН: ${item.tax_id}`,
    item.country && `Страна: ${item.country}`,
    item.project_name && `Проект: ${item.project_name}`,
    item.database_name && `БД: ${item.database_name}`,
    item.contact_email && `Email: ${item.contact_email}`,
    item.contact_phone && `Телефон: ${item.contact_phone}`,
  ].filter(Boolean).join('\n')

  return (
    <TableRow>
      <TableCell 
        className="max-w-[200px] truncate font-medium" 
        title={cellTitle}
      >
        <div className="flex flex-col gap-0.5">
          <span className="truncate">
            {item.name || <span className="text-muted-foreground italic">Без названия</span>}
          </span>
          {item.quality_score !== undefined && item.quality_score !== null && item.quality_score > 0 && (
            <div className="flex items-center gap-1">
              <div className="h-1 flex-1 bg-muted rounded-full overflow-hidden">
                <div 
                  className="h-full bg-primary transition-all"
                  style={{ width: `${Math.round(item.quality_score * 100)}%` }}
                />
              </div>
              <span className="text-xs text-muted-foreground">
                {Math.round(item.quality_score * 100)}%
              </span>
            </div>
          )}
        </div>
      </TableCell>
      <TableCell className="max-w-[200px] truncate" title={item.normalized_name || item.name || ''}>
        {item.normalized_name ? (
          <div className="flex flex-col gap-1">
            <span className="truncate">{item.normalized_name}</span>
            {item.quality_score !== undefined && item.quality_score !== null && item.quality_score > 0 && (
              <div className="flex items-center gap-1">
                <div className="h-1 flex-1 bg-muted rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-primary transition-all"
                    style={{ width: `${Math.round(item.quality_score * 100)}%` }}
                  />
                </div>
                <span className="text-xs text-muted-foreground">
                  {Math.round(item.quality_score * 100)}%
                </span>
              </div>
            )}
          </div>
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell className="font-mono text-sm">
        {item.tax_id || item.bin ? (
          <div className="flex flex-col gap-0.5">
            <span title={`ИНН/БИН: ${item.tax_id || item.bin || ''}`} className="cursor-help">
              {item.tax_id || item.bin || '—'}
            </span>
            {item.kpp && (
              <span className="text-xs text-muted-foreground" title={`КПП: ${item.kpp}`}>
                КПП: {item.kpp}
              </span>
            )}
            {(item.contact_email || item.contact_phone) && (
              <div className="flex gap-1 text-xs text-muted-foreground mt-1">
                {item.contact_email && (
                  <span title={`Email: ${item.contact_email}`} className="truncate max-w-[100px]">
                    ✉️
                  </span>
                )}
                {item.contact_phone && (
                  <span title={`Телефон: ${item.contact_phone}`}>
                    📞
                  </span>
                )}
              </div>
            )}
            {(item.legal_address || item.postal_address) && (
              <span className="text-xs text-muted-foreground truncate max-w-[150px]" title={item.legal_address || item.postal_address || ''}>
                📍 {item.legal_address || item.postal_address}
              </span>
            )}
          </div>
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell>
        {item.type ? (
          <div className="flex flex-col gap-1">
            <Badge 
              variant={item.type === 'Нормализованный' ? 'default' : item.type === 'База данных' ? 'secondary' : 'outline'}
              className="text-xs w-fit"
              title={item.type === 'Нормализованный' ? 'Нормализованный контрагент' : 'Из базы данных'}
            >
              {item.type === 'Нормализованный' ? 'Нормализован' : item.type === 'База данных' ? 'База данных' : item.type}
            </Badge>
            {item.database_name && (
              <span className="text-xs text-muted-foreground truncate max-w-[150px]" title={item.database_name}>
                📁 {item.database_name.split(/[/\\]/).pop() || item.database_name}
              </span>
            )}
            {item.source_databases && item.source_databases.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-1">
                {item.source_databases.slice(0, 2).map((db, idx) => (
                  <span 
                    key={idx}
                    className="text-xs text-muted-foreground truncate max-w-[120px]" 
                    title={`База данных: ${db.database_name}${db.source_reference ? ` (${db.source_reference})` : ''}`}
                  >
                    📁 {db.database_name.split(/[/\\]/).pop() || db.database_name}
                  </span>
                ))}
                {item.source_databases.length > 2 && (
                  <span className="text-xs text-muted-foreground" title={`Всего баз: ${item.source_databases.length}`}>
                    (+{item.source_databases.length - 2})
                  </span>
                )}
              </div>
            )}
            {item.project_name && (
              <span className="text-xs text-muted-foreground truncate max-w-[150px]" title={item.project_name}>
                📋 {item.project_name}
              </span>
            )}
          </div>
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        )}
      </TableCell>
      <TableCell>
        {item.country ? (
          <div className="flex items-center gap-1">
            <span className="text-sm">{item.country}</span>
            {item.legal_address && (
              <span 
                className="text-xs text-muted-foreground truncate max-w-[100px]" 
                title={item.legal_address}
              >
                📍
              </span>
            )}
          </div>
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        )}
      </TableCell>
      <TableCell>
        <Badge variant={item.status === 'active' ? 'default' : 'secondary'}>
          {item.status}
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onView(item)}
        >
          <Eye className="h-4 w-4" />
        </Button>
      </TableCell>
    </TableRow>
  )
})

CounterpartyRow.displayName = 'CounterpartyRow'

const toStringSafe = (value: unknown): string => {
  if (typeof value === 'string') return value
  if (value === null || value === undefined) return ''
  return String(value)
}

export function CounterpartiesTab({ clientId, projects }: CounterpartiesTabProps) {
  const [items, setItems] = useState<CounterpartyItem[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null)
  const [selectedSource, setSelectedSource] = useState<string | null>(null) // null (все) | "database" | "normalized"
  const [selectedCountry, setSelectedCountry] = useState<string | null>(null)
  const [qualityFilter, setQualityFilter] = useState<string>("all") // "all" | "high" | "medium" | "low" | "no-quality"
  const [searchQuery, setSearchQuery] = useState("")
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState("")
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [totalItems, setTotalItems] = useState(0)
  const [selectedItem, setSelectedItem] = useState<CounterpartyItem | null>(null)
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc' | null>(null)
  const [isExporting, setIsExporting] = useState(false)
  const [loadAllData, setLoadAllData] = useState(false) // Флаг для загрузки всех данных
  const [limitWarningShown, setLimitWarningShown] = useState(false)
  const [loadingProgress, setLoadingProgress] = useState<number | null>(null) // Прогресс загрузки
  const [mappingStats, setMappingStats] = useState<MappingStatsState | null>(null) // Статистика мэппинга
  const [isMappingRunning, setIsMappingRunning] = useState(false) // Флаг выполнения мэппинга
  const [mappingConfig, setMappingConfig] = useState<MappingConfigState | null>(null) // Конфигурация мэппинга
  const [showConfigDialog, setShowConfigDialog] = useState(false) // Показать диалог конфигурации
  const [isSavingConfig, setIsSavingConfig] = useState(false) // Сохранение конфигурации
  const [mappingProgress, setMappingProgress] = useState<MappingProgressState | null>(null) // Прогресс мэппинга
  const [showDuplicatesDialog, setShowDuplicatesDialog] = useState(false)
  const [duplicateGroupsData, setDuplicateGroupsData] = useState<DuplicateGroup[] | null>(null)
  const [isDuplicatesLoading, setIsDuplicatesLoading] = useState(false)
  const [duplicatesError, setDuplicatesError] = useState<string | null>(null)
  const [backendStatus, setBackendStatus] = useState<'unknown' | 'ok' | 'unreachable'>('unknown')
const itemsPerPage = 20
const MAX_BACKEND_LIMIT = 100000
const HEAVY_THRESHOLD = 5000
  const backendErrorToastAt = useRef<number>(0)

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

  const normalizedRecords =
    mappingStats?.normalized ?? mappingStats?.processed ?? 0
  const totalRecords =
    mappingStats?.total ?? mappingStats?.total_count ?? mappingStats?.planned ?? 0
  const withInn = mappingStats?.with_inn ?? 0
  const withAddress = mappingStats?.with_address ?? 0
  const withContacts = mappingStats?.with_contacts ?? 0
  const enrichedRecords = mappingStats?.enriched ?? 0
  const duplicateGroupCount = mappingStats?.duplicate_groups ?? 0
  const duplicatesCount = mappingStats?.duplicates_count ?? 0
  const multiDatabaseCount = mappingStats?.multi_database_count ?? 0
  const averageQuality = mappingStats?.average_quality_score ?? null

  // Вычисляем наличие клиентских фильтров один раз
  const hasClientFilters = useMemo(() => {
    return selectedSource || selectedCountry || qualityFilter !== 'all'
  }, [selectedSource, selectedCountry, qualityFilter])

  useEffect(() => {
    if (!loadAllData && !hasClientFilters) {
      setLimitWarningShown(false)
    }
  }, [loadAllData, hasClientFilters])

  // Функция для получения статуса мэппинга
  const fetchMappingStatus = useCallback(async (silent = false) => {
    if (!selectedProjectId) return
    
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 10000)
    
    try {
      const response = await fetch(`/api/projects/${selectedProjectId}/counterparties/mapping-status`, {
        cache: 'no-store',
        signal: controller.signal,
      })
      
      if (!response.ok) {
        let errorMessage = 'Не удалось получить статус мэппинга'
        if (response.status === 404) {
          errorMessage = 'Статистика мэппинга недоступна для выбранного проекта'
        } else if (response.status === 503 || response.status === 504) {
          errorMessage = 'Backend сервер недоступен (порт 9999). Проверьте подключение.'
        } else if (response.status >= 500) {
          errorMessage = `Ошибка сервера: ${response.status}`
        }
        throw new Error(errorMessage)
      }
      
      const data: MappingStatusResponse = await response.json()
      const stats: MappingStatsState | null = data.stats ?? null
      setMappingStats(stats)
      const resolvedConfig: MappingConfigState = {
        auto_map_counterparties: data.auto_map_counterparties ?? data.config?.auto_map_counterparties ?? true,
        auto_merge_duplicates: data.auto_merge_duplicates ?? data.config?.auto_merge_duplicates ?? true,
        master_selection_strategy: data.master_selection_strategy || data.config?.master_selection_strategy || 'max_data',
      }
      setMappingConfig(resolvedConfig)
      
      const rawProgress = data.progress || data.mapping_progress || null
      const normalizedProgress: MappingProgressState | null = rawProgress
        ? (typeof rawProgress === 'string' ? { message: rawProgress } : rawProgress)
        : null
      setMappingProgress(normalizedProgress)
      
      const normalizedValue = stats?.normalized ?? stats?.processed ?? 0
      const total = stats?.total ?? stats?.total_count ?? stats?.planned ?? 0
      const statusValue = stats?.status
      const statusFlag = statusValue
        ? ['running', 'in_progress'].includes(String(statusValue).toLowerCase())
        : false
      let derivedIsRunning: boolean | null | undefined = data.is_running
      if (derivedIsRunning === undefined || derivedIsRunning === null) {
        derivedIsRunning = stats?.is_running
      }
      if (derivedIsRunning === undefined || derivedIsRunning === null) {
        derivedIsRunning = statusFlag
      }
      if (derivedIsRunning === undefined || derivedIsRunning === null) {
        derivedIsRunning = total > 0 && normalizedValue < total
      }
      setIsMappingRunning(Boolean(derivedIsRunning))
      markBackendHealthy()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Не удалось получить статус мэппинга'
      if (isBackendConnectionError(message)) {
        notifyBackendUnavailable(message, !silent)
      } else if (!silent) {
        toast.error(message)
      }
      // Не логируем сетевые ошибки подключения к бэкенду - они ожидаемы
      if (!isBackendConnectionError(message)) {
        console.error('Error fetching mapping status:', error)
      }
    } finally {
      clearTimeout(timeoutId)
    }
  }, [selectedProjectId, isBackendConnectionError, notifyBackendUnavailable, markBackendHealthy])

  // Загружаем статистику при изменении выбранного проекта
  useEffect(() => {
    if (selectedProjectId) {
      fetchMappingStatus()
    } else {
      setMappingStats(null)
      setMappingConfig(null)
    }
  }, [selectedProjectId, fetchMappingStatus])

  // Автоматическое обновление статистики при выполнении мэппинга
  useEffect(() => {
    if (!isMappingRunning || !selectedProjectId || backendStatus === 'unreachable') return

    const interval = setInterval(() => {
      fetchMappingStatus(true)
    }, 3000) // Обновляем каждые 3 секунды

    return () => clearInterval(interval)
  }, [isMappingRunning, selectedProjectId, fetchMappingStatus, backendStatus])

  const loadDuplicateGroups = useCallback(async () => {
    if (!selectedProjectId) return
    setIsDuplicatesLoading(true)
    setDuplicatesError(null)
    
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 20000)
    
    try {
      const response = await fetch(`/api/counterparties/duplicates?project_id=${selectedProjectId}`, {
        signal: controller.signal,
        cache: 'no-store',
      })
      if (!response.ok) {
        let errorMessage = 'Не удалось загрузить дубликаты'
        if (response.status === 404) {
          errorMessage = 'Дубликаты не найдены для выбранного проекта'
        } else if (response.status === 503 || response.status === 504) {
          errorMessage = 'Backend сервер недоступен (порт 9999). Проверьте подключение.'
        } else if (response.status >= 500) {
          errorMessage = `Ошибка сервера: ${response.status}`
        }
        throw new Error(errorMessage)
      }
      const data: { groups?: DuplicateGroup[] } = await response.json()
      setDuplicateGroupsData(data.groups || [])
      markBackendHealthy()
    } catch (error) {
      const message = error instanceof Error
        ? (error.name === 'AbortError' ? 'Превышено время ожидания ответа от сервера' : error.message)
        : 'Неизвестная ошибка'
      setDuplicatesError(message)
      // Не логируем сетевые ошибки подключения к бэкенду - они ожидаемы
      if (!isBackendConnectionError(message)) {
        console.error('Failed to fetch duplicates:', error)
      }
      if (isBackendConnectionError(message)) {
        notifyBackendUnavailable(message, false)
      }
    } finally {
      clearTimeout(timeoutId)
      setIsDuplicatesLoading(false)
    }
  }, [selectedProjectId, isBackendConnectionError, notifyBackendUnavailable, markBackendHealthy])

  useEffect(() => {
    setShowDuplicatesDialog(false)
    setDuplicateGroupsData(null)
    setDuplicatesError(null)
  }, [selectedProjectId])

  // Debounce для поиска - задержка 500мс
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery)
      // Сбрасываем на первую страницу при изменении поиска
      if (searchQuery !== debouncedSearchQuery) {
        setCurrentPage(1)
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [searchQuery, debouncedSearchQuery])

  const fetchCounterparties = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      // Используем endpoint /api/counterparties/all для получения всех контрагентов (из баз и нормализованных)
      // Фильтры по источнику, стране и качеству применяются на клиенте
      // Для больших объемов данных (тысячи записей) загружаем больше данных
      // hasClientFilters уже объявлена в useMemo на строке 103
      // Если запрошена загрузка всех данных (кнопка "Загрузить все"), запрашиваем максимум, который разрешает сервер
      // Если есть клиентские фильтры, также запрашиваем максимум для корректной фильтрации
      // Если нет фильтров и нет поиска, стараемся загрузить достаточно записей для первичной фильтрации (сервер ограничивает до 500)
      // При поиске используем серверную пагинацию для оптимизации
      // Максимальный лимит на стороне бэкенда составляет 500 записей
      // ВАЖНО: loadAllData должен быть явно установлен в true только при нажатии кнопки "Загрузить все"
      const forceLoadAll = loadAllData
      const baseLimit = debouncedSearchQuery
        ? itemsPerPage
        : (hasClientFilters ? 5000 : 1000)
      const desiredLimit = forceLoadAll ? MAX_BACKEND_LIMIT : baseLimit
      const limitForRequest = Math.min(desiredLimit, MAX_BACKEND_LIMIT)
      const limitWasClamped = desiredLimit > MAX_BACKEND_LIMIT && !forceLoadAll
      const heavyRequest = forceLoadAll || limitForRequest >= HEAVY_THRESHOLD

      if (limitWasClamped && !limitWarningShown) {
        toast.info(`Сервер возвращает не более ${MAX_BACKEND_LIMIT.toLocaleString('ru-RU')} записей за один запрос. Используйте фильтры или экспорт для полной выгрузки.`, {
          duration: 6000,
        })
        setLimitWarningShown(true)
      }

      const offsetForRequest = forceLoadAll ? 0 : (currentPage - 1) * itemsPerPage
      const requestTimeoutMs = heavyRequest ? 120000 : 30000
      
      let url = `/api/counterparties/all?client_id=${clientId}&offset=${offsetForRequest}&limit=${limitForRequest}`
      if (selectedProjectId) {
        url += `&project_id=${selectedProjectId}`
      }
      if (forceLoadAll) {
        url += `&load_all=1`
      }
      // Поиск применяется на сервере для оптимизации
      if (debouncedSearchQuery) {
        url += `&search=${encodeURIComponent(debouncedSearchQuery)}`
      }
      
      setLoadingProgress(heavyRequest ? 5 : 10)
      
      // Добавляем таймаут и обработку ошибок
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), requestTimeoutMs)
      let responseData: CounterpartiesResponse | null = null
      
      try {
        const response = await fetch(url, {
          cache: 'no-store',
          signal: controller.signal,
        })
        
        if (!response.ok) {
          // Обработка различных статусов ошибок
          if (response.status === 404) {
            // 404 - это не ошибка, просто данных нет
            setLoadingProgress(null)
            setItems([])
            setError(null)
            setTotalItems(0)
            setTotalPages(1)
            setCurrentPage(1)
            return
          } else if (response.status === 429) {
            setLoadingProgress(null)
            throw new Error('Очередь выгрузки занята. Повторите запрос через несколько секунд.')
          } else if (response.status === 503 || response.status === 504) {
            throw new Error('Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.')
          } else if (response.status >= 500) {
            throw new Error('Ошибка сервера при загрузке контрагентов')
          }
          
          const errorText = await response.text().catch(() => 'Unknown error')
          let errorMessage = 'Не удалось загрузить контрагенты'
          try {
            const errorData = JSON.parse(errorText)
            errorMessage = errorData.error || errorMessage
          } catch {
            // Если не удалось распарсить JSON, используем текст ошибки
            if (errorText && errorText !== 'Unknown error') {
              errorMessage = errorText
            }
          }
          setLoadingProgress(null)
          throw new Error(errorMessage)
        }
        
        setLoadingProgress(heavyRequest ? 35 : 40)
        responseData = await response.json()
        if (responseData && typeof responseData === 'object' && (responseData as any).limit_clamped && !limitWarningShown) {
          toast.info(`Получено ${MAX_BACKEND_LIMIT.toLocaleString('ru-RU')} записей (серверный лимит). Используйте фильтры или экспорт для полного набора.`, {
            duration: 6000,
          })
          setLimitWarningShown(true)
        }
        setLoadingProgress(heavyRequest ? 80 : 70)
      } finally {
        clearTimeout(timeoutId)
      }
      
      if (!responseData) {
        throw new Error('Пустой ответ от сервера контрагентов')
      }
      
      // Обработка ответа от /api/counterparties/all
      const itemsList = (responseData.counterparties || responseData.items || responseData.data || []) as CounterpartyApiItem[]
      const total = responseData.total || responseData.count || itemsList.length
      
      // Преобразуем UnifiedCounterparty в CounterpartyItem для отображения
      const transformedItems: CounterpartyItem[] = itemsList.map((item: CounterpartyApiItem, index: number) => {
        const numericId = typeof item.id === 'number' ? item.id : Number(item.id)
        const resolvedId = Number.isFinite(numericId) ? (numericId as number) : index + 1
        const rawSource = typeof item.source === 'string' ? item.source : ''
        const databaseKey = toStringSafe(item.database_id) || 'no-db'
        const projectKey = toStringSafe(item.project_id) || 'no-proj'

        // Функция для извлечения страны из различных источников
        const extractCountry = (): string => {
          // 1. Проверяем атрибуты
          if (item.attributes) {
            try {
              let attrs = item.attributes
              if (typeof attrs === 'string') {
                attrs = JSON.parse(attrs)
              }
              
              if (attrs && typeof attrs === 'object') {
                const attrsObj = attrs as Record<string, unknown>;
                const countryFromAttrs = String(attrsObj.country || attrsObj.Country || attrsObj.страна || attrsObj.Страна ||
                                       attrsObj.country_name || attrsObj.countryName || '')
                if (countryFromAttrs) return countryFromAttrs
              }
            } catch {
              // Игнорируем ошибки парсинга
            }
          }
          
          // 2. Извлекаем из названия (например, "27 AAT GmbH (Германия)")
          const nameFields = [item.name, item.normalized_name, item.source_name]
            .filter(Boolean)
            .map(n => String(n))
          for (const name of nameFields) {
            const countryMatch = name.match(/\(([^)]+)\)/g)
            if (countryMatch) {
              const countryInBrackets = countryMatch[countryMatch.length - 1].replace(/[()]/g, '').trim()
              const countryPatterns: { [key: string]: string } = {
                'россия': 'Россия', 'russia': 'Россия', 'ru': 'Россия',
                'казахстан': 'Казахстан', 'kazakhstan': 'Казахстан', 'kz': 'Казахстан',
                'беларусь': 'Беларусь', 'belarus': 'Беларусь', 'by': 'Беларусь',
                'германия': 'Германия', 'germany': 'Германия', 'de': 'Германия',
                'дания': 'Дания', 'denmark': 'Дания', 'dk': 'Дания',
                'азербайджан': 'Азербайджан', 'azerbaijan': 'Азербайджан', 'az': 'Азербайджан',
                'украина': 'Украина', 'ukraine': 'Украина', 'ua': 'Украина',
                'китай': 'Китай', 'china': 'Китай', 'cn': 'Китай',
                'турция': 'Турция', 'turkey': 'Турция', 'tr': 'Турция',
              }
              const lowerCountry = countryInBrackets.toLowerCase()
              if (countryPatterns[lowerCountry]) {
                return countryPatterns[lowerCountry]
              }
              // Если не найдено в паттернах, но есть текст в скобках, используем его
              if (countryInBrackets.length > 1 && countryInBrackets.length < 30) {
                return countryInBrackets
              }
            }
          }
          
          // 3. Извлекаем из адреса
          const addressFields = [item.legal_address, item.postal_address]
            .filter(Boolean)
            .map(a => String(a))
          for (const address of addressFields) {
            const addressLower = address.toLowerCase()
            const countryPatterns: { [key: string]: string } = {
              'россия': 'Россия', 'russia': 'Россия', 'russian federation': 'Россия',
              'казахстан': 'Казахстан', 'kazakhstan': 'Казахстан', 'republic of kazakhstan': 'Казахстан',
              'беларусь': 'Беларусь', 'belarus': 'Беларусь', 'republic of belarus': 'Беларусь',
              'германия': 'Германия', 'germany': 'Германия', 'deutschland': 'Германия',
              'дания': 'Дания', 'denmark': 'Дания', 'danmark': 'Дания',
              'азербайджан': 'Азербайджан', 'azerbaijan': 'Азербайджан',
              'украина': 'Украина', 'ukraine': 'Украина',
              'китай': 'Китай', 'china': 'Китай',
              'турция': 'Турция', 'turkey': 'Турция', 'türkiye': 'Турция',
            }
            
            for (const [pattern, countryName] of Object.entries(countryPatterns)) {
              if (addressLower.includes(pattern)) {
                return countryName
              }
            }
          }
          
          return ''
        }
        
        // Извлекаем ИНН/БИН из всех возможных источников
        const extractTaxId = (): string => {
          // Приоритет: tax_id > bin > извлечение из других полей
          const taxId = String(item.tax_id || '');
          if (taxId && taxId.trim() && taxId !== '<>') return taxId.trim()
          
          // Проверяем BIN (для Казахстана)
          const bin = String(item.bin || '');
          if (bin && bin.trim() && bin !== '<>') return bin.trim()
          
          // Пробуем извлечь из code или reference
          const codeFields = [item.code, item.reference, item.source_reference]
            .filter(Boolean)
            .map(c => String(c))
          for (const field of codeFields) {
            if (!field || field === '<>') continue
            // Простая проверка на ИНН (10 или 12 цифр) или БИН (12 цифр)
            const numbers = field.replace(/\D/g, '')
            if (numbers.length === 10 || numbers.length === 12) {
              return numbers
            }
          }
          
          return ''
        }
        
        // Преобразуем источник в читаемый формат
        const sourceDisplay = rawSource === 'database' ? 'База данных' : 
                             rawSource === 'normalized' ? 'Нормализованный' : 
                             rawSource || 'Неизвестно'
        
        // Очищаем название от страны в скобках для нормализованного названия
        const cleanName = (name: string): string => {
          if (!name) return ''
          // Удаляем страну в скобках в конце (например, "27 AAT GmbH (Германия)" -> "27 AAT GmbH")
          return name.replace(/\s*\([^)]+\)\s*$/, '').trim()
        }
        
        const nameStr = toStringSafe(item.name);
        const normalizedNameStr = toStringSafe(item.normalized_name);
        const sourceNameStr = toStringSafe(item.source_name);
        const referenceStr = toStringSafe(item.reference);
        
        const rawName = (nameStr && nameStr.trim() && nameStr !== '<>') 
          ? nameStr 
          : (normalizedNameStr && normalizedNameStr.trim() && normalizedNameStr !== '<>')
          ? normalizedNameStr
          : (sourceNameStr && sourceNameStr.trim() && sourceNameStr !== '<>')
          ? sourceNameStr
          : (referenceStr && referenceStr.trim() && referenceStr !== '<>')
          ? referenceStr
          : ''

        const rawNormalizedName = (normalizedNameStr && normalizedNameStr.trim() && normalizedNameStr !== '<>')
          ? normalizedNameStr
          : (nameStr && nameStr.trim() && nameStr !== '<>')
          ? nameStr
          : (referenceStr && referenceStr.trim() && referenceStr !== '<>')
          ? referenceStr
          : ''
        
        const sourceDatabases = Array.isArray(item.source_databases) ? item.source_databases : undefined

        return {
          id: resolvedId,
          // Создаем уникальный ключ: source + id + database_id + project_id + index для избежания дубликатов
          uniqueKey: `${rawSource || 'unknown'}-${resolvedId}-${databaseKey}-${projectKey}-${index}`,
          name: cleanName(rawName) || rawName,
          normalized_name: cleanName(rawNormalizedName) || rawNormalizedName,
          tax_id: extractTaxId(),
          type: sourceDisplay,
          status: 'active', // По умолчанию активный
          quality_score: typeof item.quality_score === 'number' ? item.quality_score : undefined,
          country: extractCountry(),
          contact_email: toStringSafe(item.contact_email),
          contact_phone: toStringSafe(item.contact_phone),
          contact_person: toStringSafe(item.contact_person),
          legal_address: toStringSafe(item.legal_address),
          postal_address: toStringSafe(item.postal_address),
          kpp: toStringSafe(item.kpp),
          bin: toStringSafe(item.bin),
          source_reference: toStringSafe(item.source_reference ?? item.reference),
          source_name: toStringSafe(item.source_name),
          project_name: toStringSafe(item.project_name),
          database_name: toStringSafe(item.database_name),
          source_databases: sourceDatabases,
        }
      })
      
      setItems(transformedItems)
      markBackendHealthy()
      // Для серверной пагинации (без клиентских фильтров) используем total от API
      // Для клиентской фильтрации total будет пересчитан в useMemo на основе filteredItems
      // hasClientFilters уже объявлена в useMemo на строке 99
      if (!hasClientFilters) {
        // Используем total от API для серверной пагинации
        setTotalItems(total)
        setTotalPages(Math.ceil(total / itemsPerPage))
      } else {
        // При клиентской фильтрации используем количество загруженных элементов
        // Если загружено меньше чем total, значит есть еще данные на сервере
        setTotalItems(transformedItems.length < total ? total : transformedItems.length)
      }
      // При клиентской фильтрации totalItems и totalPages будут пересчитаны в useMemo
      
      setLoadingProgress(95)
      // Небольшая задержка для плавного завершения
      await new Promise(resolve => setTimeout(resolve, 100))
      setLoadingProgress(100)
      setTimeout(() => setLoadingProgress(null), 500) // Скрываем прогресс через 500мс
      if (forceLoadAll) {
        toast.success(`Выгрузка ${Math.min(total, MAX_BACKEND_LIMIT).toLocaleString('ru-RU')} записей завершена`, {
          duration: 6000,
        })
      }
    } catch (error) {
      let errorMessage = 'Не удалось загрузить контрагентов'
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          const seconds = Math.round(requestTimeoutMs / 1000)
          errorMessage = `Превышено время ожидания ответа от сервера (${seconds} секунд). Попробуйте уменьшить объем данных или использовать фильтры.`
        } else if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
          errorMessage = 'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.'
        } else if (error.message.includes('подключиться к backend')) {
          errorMessage = error.message
        } else if (error.message.includes('Очередь выгрузки')) {
          errorMessage = error.message
        } else {
          errorMessage = error.message
        }
      }
      
      setError(errorMessage)
      // Не логируем сетевые ошибки подключения к бэкенду - они ожидаемы
      if (!isBackendConnectionError(errorMessage)) {
        console.error('Failed to fetch counterparties:', error)
      }
      if (isBackendConnectionError(errorMessage)) {
        notifyBackendUnavailable(errorMessage, true)
      }
      setLoadingProgress(null)
    } finally {
      setIsLoading(false)
    }
  }, [
    clientId,
    selectedProjectId,
    currentPage,
    debouncedSearchQuery,
    itemsPerPage,
    loadAllData,
    hasClientFilters,
    isBackendConnectionError,
    notifyBackendUnavailable,
    markBackendHealthy,
    limitWarningShown,
  ])

  useEffect(() => {
    fetchCounterparties()
  }, [fetchCounterparties])

  const handleBackendRetry = useCallback(() => {
    setBackendStatus('unknown')
    backendErrorToastAt.current = 0
    fetchCounterparties()
    fetchMappingStatus(true)
  }, [fetchCounterparties, fetchMappingStatus])

  const handleSearch = useCallback((value: string) => {
    setSearchQuery(value)
    setCurrentPage(1)
    setLoadAllData(false) // Сбрасываем флаг загрузки всех данных при поиске
  }, [])

  const handleQualityFilterChange = useCallback((value: string) => {
    setQualityFilter(value)
    setCurrentPage(1)
    setLoadAllData(false) // Сбрасываем флаг загрузки всех данных при изменении фильтра
  }, [])

  const handleViewItem = useCallback((item: CounterpartyItem) => {
    setSelectedItem(item)
  }, [])

  // Получаем уникальные страны из данных
  const availableCountries = useMemo(() => {
    const countries = new Set<string>()
    items.forEach(item => {
      const country = item.country?.trim()
      // Фильтруем пустые значения и null/undefined
      if (country && country.length > 0) {
        countries.add(country)
      }
    })
    return Array.from(countries).sort()
  }, [items])

  // Статистика по источникам
  const sourceStats = useMemo(() => {
    const stats = {
      all: items.length,
      database: items.filter(item => item.type === 'База данных' || item.type === 'database').length,
      normalized: items.filter(item => item.type === 'Нормализованный' || item.type === 'normalized').length,
    }
    return stats
  }, [items])

  // Фильтрация по источнику, стране и качеству
  const filteredItems = useMemo(() => {
    let filtered = items

    // Фильтр по источнику
    if (selectedSource) {
      if (selectedSource === 'database') {
        filtered = filtered.filter(item => item.type === 'База данных' || item.type === 'database')
      } else if (selectedSource === 'normalized') {
        filtered = filtered.filter(item => item.type === 'Нормализованный' || item.type === 'normalized')
      }
    }

    // Фильтр по стране
    if (selectedCountry) {
      filtered = filtered.filter(item => item.country === selectedCountry)
    }

    // Фильтр по качеству
    if (qualityFilter !== 'all') {
      filtered = filtered.filter(item => {
        if (qualityFilter === 'no-quality') {
          return item.quality_score === undefined || item.quality_score === null
        }
        const score = item.quality_score ?? 0
        switch (qualityFilter) {
          case 'high':
            return score >= 0.9
          case 'medium':
            return score >= 0.7 && score < 0.9
          case 'low':
            return score < 0.7
          default:
            return true
        }
      })
    }

    return filtered
  }, [items, selectedSource, selectedCountry, qualityFilter])

  // Пересчитываем totalItems и totalPages на основе отфильтрованных данных
  const filteredTotalItems = useMemo(() => filteredItems.length, [filteredItems])
  const filteredTotalPages = useMemo(() => {
    if (hasClientFilters) {
      return Math.ceil(filteredItems.length / itemsPerPage)
    }
    return totalPages
  }, [filteredItems, itemsPerPage, totalPages, hasClientFilters])
  
  // Обновляем состояние пагинации при изменении фильтров
  useEffect(() => {
    if (filteredTotalPages > 0 && currentPage > filteredTotalPages) {
      setCurrentPage(1)
    }
  }, [filteredTotalPages, currentPage])

  const handleExport = async (format: 'csv' | 'json' | 'xml' = 'json') => {
    setIsExporting(true)
    try {
      // Экспортируем отфильтрованные данные (те, что видны пользователю)
      // Используем filteredItems вместо sortedItems, чтобы экспортировать все отфильтрованные записи, а не только текущую страницу
      const dataToExport = filteredItems.map(item => ({
        id: item.id,
        name: String(item.name || ''),
        normalized_name: String(item.normalized_name || ''),
        tax_id: String(item.tax_id || ''),
        kpp: String(item.kpp || ''),
        bin: item.bin,
        type: item.type,
        status: item.status,
        quality_score: item.quality_score,
        country: item.country,
        contact_email: item.contact_email,
        contact_phone: item.contact_phone,
        contact_person: item.contact_person,
        legal_address: String(item.legal_address || ''),
        postal_address: String(item.postal_address || ''),
        project_name: String(item.project_name || ''),
        database_name: item.database_name,
        source_reference: item.source_reference,
        source_name: String(item.source_name || ''),
      }))

      if (format === 'json') {
        const blob = new Blob([JSON.stringify(dataToExport, null, 2)], { type: 'application/json' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `counterparties_${clientId}_${new Date().toISOString().split('T')[0]}.json`
        a.click()
        URL.revokeObjectURL(url)
      } else if (format === 'csv') {
        // Преобразуем в CSV
        const headers = ['ID', 'Название', 'Нормализованное название', 'ИНН', 'КПП', 'БИН', 'Источник', 'Статус', 'Качество', 'Страна', 'Email', 'Телефон', 'Контактное лицо', 'Юридический адрес', 'Почтовый адрес', 'Проект', 'База данных']
        const rows = dataToExport.map(item => [
          item.id,
          item.name || '',
          item.normalized_name || '',
          item.tax_id || '',
          item.kpp || '',
          item.bin || '',
          item.type || '',
          item.status || '',
          item.quality_score !== undefined && item.quality_score !== null ? Math.round(item.quality_score * 100) : '',
          item.country || '',
          item.contact_email || '',
          item.contact_phone || '',
          item.contact_person || '',
          String(item.legal_address || ''),
          String(item.postal_address || ''),
          String(item.project_name || ''),
          item.database_name || '',
        ])
        
        const csvContent = [
          headers.join(','),
          ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
        ].join('\n')
        
        const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `counterparties_${clientId}_${new Date().toISOString().split('T')[0]}.csv`
        a.click()
        URL.revokeObjectURL(url)
      } else if (format === 'xml') {
        // Простой XML экспорт
        const xmlContent = `<?xml version="1.0" encoding="UTF-8"?>
<counterparties>
${dataToExport.map(item => `  <counterparty>
    <id>${item.id}</id>
    <name>${escapeXml(item.name || '')}</name>
    <normalized_name>${escapeXml(item.normalized_name || '')}</normalized_name>
    <tax_id>${escapeXml(item.tax_id || '')}</tax_id>
    <kpp>${escapeXml(item.kpp || '')}</kpp>
    <bin>${escapeXml(item.bin || '')}</bin>
    <type>${escapeXml(item.type || '')}</type>
    <status>${escapeXml(item.status || '')}</status>
    <quality_score>${item.quality_score !== undefined && item.quality_score !== null ? item.quality_score : ''}</quality_score>
    <country>${escapeXml(item.country || '')}</country>
    <contact_email>${escapeXml(item.contact_email || '')}</contact_email>
    <contact_phone>${escapeXml(item.contact_phone || '')}</contact_phone>
    <contact_person>${escapeXml(item.contact_person || '')}</contact_person>
    <legal_address>${escapeXml(String(item.legal_address || ''))}</legal_address>
    <postal_address>${escapeXml(String(item.postal_address || ''))}</postal_address>
    <project_name>${escapeXml(String(item.project_name || ''))}</project_name>
    <database_name>${escapeXml(item.database_name || '')}</database_name>
  </counterparty>`).join('\n')}
</counterparties>`
        
        const blob = new Blob([xmlContent], { type: 'application/xml' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `counterparties_${clientId}_${new Date().toISOString().split('T')[0]}.xml`
        a.click()
        URL.revokeObjectURL(url)
      }
    } catch (error) {
      console.error('Failed to export counterparties:', error)
      setError(error instanceof Error ? error.message : 'Не удалось экспортировать контрагентов')
      toast.error(`Ошибка экспорта: ${error instanceof Error ? error.message : 'Неизвестная ошибка'}`)
    } finally {
      setIsExporting(false)
    }
  }

  const escapeXml = (str: string): string => {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&apos;')
  }

  const handleSort = useCallback((key: string) => {
    if (sortKey === key) {
      if (sortDirection === 'asc') {
        setSortDirection('desc')
      } else if (sortDirection === 'desc') {
        setSortKey(null)
        setSortDirection(null)
      } else {
        setSortDirection('asc')
      }
    } else {
      setSortKey(key)
      setSortDirection('asc')
    }
  }, [sortKey, sortDirection])

  const getSortIcon = (key: string) => {
    if (sortKey !== key) {
      return <ArrowUpDown className="h-4 w-4 ml-1 opacity-50" />
    }
    if (sortDirection === 'asc') {
      return <ArrowUp className="h-4 w-4 ml-1" />
    }
    return <ArrowDown className="h-4 w-4 ml-1" />
  }

  // Сортировка данных
  const sortedItems = useMemo(() => {
    if (!sortKey || !sortDirection) return filteredItems

    return [...filteredItems].sort((a, b) => {
      let aValue: string | number | undefined
      let bValue: string | number | undefined

      switch (sortKey) {
        case 'name':
          aValue = a.name || ''
          bValue = b.name || ''
          break
        case 'normalized_name':
          aValue = a.normalized_name || ''
          bValue = b.normalized_name || ''
          break
        case 'tax_id':
          aValue = a.tax_id || ''
          bValue = b.tax_id || ''
          break
        case 'type':
          aValue = a.type || ''
          bValue = b.type || ''
          break
        case 'country':
          aValue = a.country || ''
          bValue = b.country || ''
          break
        case 'status':
          aValue = a.status || ''
          bValue = b.status || ''
          break
        default:
          return 0
      }

      // Обработка null/undefined
      if (aValue == null && bValue == null) return 0
      if (aValue == null) return 1
      if (bValue == null) return -1

      // Сравнение значений
      let comparison = 0
      if (typeof aValue === 'string' && typeof bValue === 'string') {
        comparison = aValue.localeCompare(bValue, 'ru-RU', { numeric: true, sensitivity: 'base' })
      } else if (typeof aValue === 'number' && typeof bValue === 'number') {
        comparison = aValue - bValue
      } else {
        comparison = String(aValue).localeCompare(String(bValue), 'ru-RU', { numeric: true })
      }

      return sortDirection === 'asc' ? comparison : -comparison
    })
  }, [filteredItems, sortKey, sortDirection])

  if (isLoading && items.length === 0) {
    return <LoadingState message="Загрузка контрагентов..." />
  }

  // Показываем индикатор загрузки поверх данных при обновлении
  // const isRefreshing = isLoading && items.length > 0

  return (
    <div className="space-y-4">
      {backendStatus === 'unreachable' && (
        <Alert variant="destructive">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Backend недоступен</AlertTitle>
          </div>
          <AlertDescription className="mt-2 space-y-2">
            <p>Не удаётся подключиться к API (порт 9999). Повторные фоновые запросы временно приостановлены.</p>
            <Button
              variant="outline"
              size="sm"
              onClick={handleBackendRetry}
              className="flex items-center gap-2"
            >
              <RefreshCw className="h-4 w-4" />
              Повторить попытку
            </Button>
          </AlertDescription>
        </Alert>
      )}
      {/* Фильтры */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Фильтры</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col md:flex-row gap-4">
            <Select
              value={selectedProjectId?.toString() || "all"}
              onValueChange={(value) => {
                setSelectedProjectId(value === "all" ? null : parseInt(value))
                setCurrentPage(1)
                setLoadAllData(false) // Сбрасываем флаг загрузки всех данных при изменении проекта
              }}
            >
              <SelectTrigger className="w-full md:w-[200px]">
                <SelectValue placeholder="Выберите проект" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все проекты</SelectItem>
                {projects.map((project) => (
                  <SelectItem key={project.id} value={project.id.toString()}>
                    {project.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={selectedSource || "all"}
              onValueChange={(value) => {
                setSelectedSource(value === 'all' ? null : value)
                setCurrentPage(1)
                setLoadAllData(false) // Сбрасываем флаг загрузки всех данных при изменении источника
              }}
            >
              <SelectTrigger className="w-full md:w-[200px]">
                <SelectValue placeholder="Источник" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  Все источники {sourceStats.all > 0 && `(${sourceStats.all})`}
                </SelectItem>
                <SelectItem value="database">
                  База данных {sourceStats.database > 0 && `(${sourceStats.database})`}
                </SelectItem>
                <SelectItem value="normalized">
                  Нормализованные {sourceStats.normalized > 0 && `(${sourceStats.normalized})`}
                </SelectItem>
              </SelectContent>
            </Select>
            {availableCountries.length > 0 && (
              <Select
                value={selectedCountry || "all"}
                onValueChange={(value) => {
                  setSelectedCountry(value === 'all' ? null : value)
                  setCurrentPage(1)
                  setLoadAllData(false) // Сбрасываем флаг загрузки всех данных при изменении страны
                }}
              >
                <SelectTrigger className="w-full md:w-[180px]">
                  <SelectValue placeholder="Страна" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все страны</SelectItem>
                  {availableCountries.map((country) => (
                    <SelectItem key={country} value={country}>
                      {country}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <Select
              value={qualityFilter}
              onValueChange={handleQualityFilterChange}
            >
              <SelectTrigger className="w-full md:w-[180px]">
                <SelectValue placeholder="Качество" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все</SelectItem>
                <SelectItem value="high">Высокое (≥90%)</SelectItem>
                <SelectItem value="medium">Среднее (70-89%)</SelectItem>
                <SelectItem value="low">Низкое (&lt;70%)</SelectItem>
                <SelectItem value="no-quality">Без оценки</SelectItem>
              </SelectContent>
            </Select>
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Поиск по названию или ИНН/БИН..."
                value={searchQuery}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Управление мэппингом контрагентов */}
      {selectedProjectId && (
        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Settings className="h-4 w-4" />
              Автоматический мэппинг контрагентов
            </CardTitle>
            <CardDescription>
              Настройка и запуск автоматического объединения дубликатов контрагентов
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Статистика мэппинга */}
            {mappingStats && (
              <>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-3 bg-muted/50 rounded-lg">
                  <div className="flex flex-col gap-1">
                    <span className="text-xs text-muted-foreground">Всего контрагентов</span>
                    <span className="text-lg font-semibold">{totalRecords.toLocaleString('ru-RU')}</span>
                  </div>
                  <div className="flex flex-col gap-1">
                    <span className="text-xs text-muted-foreground">Нормализовано</span>
                    <span className="text-lg font-semibold">{normalizedRecords.toLocaleString('ru-RU')}</span>
                    {totalRecords > 0 && (
                      <span className="text-xs text-muted-foreground">
                        {Math.round((normalizedRecords || 0) / totalRecords * 100)}% от всего
                      </span>
                    )}
                  </div>
                  <div className="flex flex-col gap-1">
                    <span className="text-xs text-muted-foreground">С ИНН/БИН</span>
                    <span className="text-lg font-semibold">{withInn.toLocaleString('ru-RU')}</span>
                    {normalizedRecords > 0 && (
                      <span className="text-xs text-muted-foreground">
                        {Math.round((withInn / normalizedRecords) * 100)}% нормализованных
                      </span>
                    )}
                  </div>
                  <div className="flex flex-col gap-1">
                    <span className="text-xs text-muted-foreground">Среднее качество</span>
                    <span className="text-lg font-semibold">
                      {averageQuality !== null
                        ? `${Math.round(averageQuality * 100)}%`
                        : '—'}
                    </span>
                    {averageQuality !== null && (
                      <div className="flex gap-1 mt-1">
                        {averageQuality >= 0.9 ? (
                          <Badge variant="default" className="text-xs">Высокое</Badge>
                        ) : averageQuality >= 0.7 ? (
                          <Badge variant="secondary" className="text-xs">Среднее</Badge>
                        ) : (
                          <Badge variant="outline" className="text-xs">Низкое</Badge>
                        )}
                      </div>
                    )}
                  </div>
                </div>
                
                {/* Дополнительная статистика */}
                {(withAddress > 0 || withContacts > 0 || enrichedRecords > 0 || duplicateGroupCount > 0 || multiDatabaseCount > 0) && (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3 p-2 bg-muted/30 rounded-lg text-sm">
                    {withAddress > 0 && (
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs text-muted-foreground">С адресами</span>
                        <span className="font-medium">{withAddress.toLocaleString('ru-RU')}</span>
                      </div>
                    )}
                    {withContacts > 0 && (
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs text-muted-foreground">С контактами</span>
                        <span className="font-medium">{withContacts.toLocaleString('ru-RU')}</span>
                      </div>
                    )}
                    {enrichedRecords > 0 && (
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs text-muted-foreground">Обогащено</span>
                        <span className="font-medium">{enrichedRecords.toLocaleString('ru-RU')}</span>
                      </div>
                    )}
                    {duplicateGroupCount > 0 && (
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs text-muted-foreground">Групп дубликатов</span>
                        <span className="font-medium text-orange-600">
                          {duplicateGroupCount.toLocaleString('ru-RU')}
                          {duplicatesCount > 0 && (
                            <span className="text-xs ml-1">
                              ({duplicatesCount} записей)
                            </span>
                          )}
                        </span>
                      </div>
                    )}
                    {multiDatabaseCount > 0 && (
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs text-muted-foreground">Из нескольких БД</span>
                        <span className="font-medium">{multiDatabaseCount.toLocaleString('ru-RU')}</span>
                      </div>
                    )}
                  </div>
                )}
              </>
            )}

            {/* Конфигурация */}
            {mappingConfig && (
              <div className="flex flex-wrap gap-2 items-center justify-between p-3 bg-muted/30 rounded-lg">
                <div className="flex flex-wrap gap-2 items-center text-sm">
                  <Badge variant={mappingConfig.auto_map_counterparties ? "default" : "secondary"}>
                    {mappingConfig.auto_map_counterparties ? "Автомэппинг включен" : "Автомэппинг выключен"}
                  </Badge>
                  <Badge variant={mappingConfig.auto_merge_duplicates ? "default" : "secondary"}>
                    {mappingConfig.auto_merge_duplicates ? "Автообъединение включено" : "Автообъединение выключено"}
                  </Badge>
                  <Badge variant="outline">
                    Стратегия: {
                      mappingConfig.master_selection_strategy === "max_quality" ? "Максимальное качество" :
                      mappingConfig.master_selection_strategy === "max_databases" ? "Максимум баз данных" :
                      "Максимум данных"
                    }
                  </Badge>
                </div>
                <Button
                  onClick={() => setShowConfigDialog(true)}
                  variant="outline"
                  size="sm"
                >
                  <Settings className="h-4 w-4 mr-2" />
                  Настроить
                </Button>
              </div>
            )}

            {/* Индикатор прогресса мэппинга */}
            {mappingProgress && isMappingRunning && (
              <div className="p-3 bg-blue-50 dark:bg-blue-950 rounded-lg border border-blue-200 dark:border-blue-800">
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-2 flex-1">
                    <Loader2 className="h-4 w-4 animate-spin text-blue-600" />
                    <span className="text-sm font-medium text-blue-900 dark:text-blue-100">
                      {mappingProgress.message || 'Выполняется мэппинг контрагентов...'}
                    </span>
                  </div>
                  {mappingStats && totalRecords > 0 && (
                    <div className="text-xs text-blue-700 dark:text-blue-300">
                      Обработано: {normalizedRecords} / {totalRecords}
                    </div>
                  )}
                </div>
                <Progress 
                  value={mappingStats && totalRecords > 0 
                    ? Math.min(100, (normalizedRecords / totalRecords) * 100)
                    : 0
                  } 
                  className="h-2 mt-2" 
                />
              </div>
            )}

            {/* Кнопки управления */}
            <div className="flex gap-2">
              <Button
                onClick={async () => {
                  if (!selectedProjectId) return
                  setIsMappingRunning(true)
                  const controller = new AbortController()
                  const timeoutId = setTimeout(() => controller.abort(), 45000)
                  try {
                    const response = await fetch(`/api/projects/${selectedProjectId}/counterparties/auto-map`, {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/json' },
                      signal: controller.signal,
                    })
                    
                    clearTimeout(timeoutId)
                    
                    if (!response.ok) {
                      let errorMessage = 'Не удалось запустить мэппинг'
                      if (response.status === 503 || response.status === 504) {
                        errorMessage = 'Backend сервер недоступен (порт 9999). Проверьте подключение.'
                      } else if (response.status >= 500) {
                        errorMessage = `Ошибка сервера: ${response.status}`
                      } else {
                        const errorData = await response.json().catch(() => ({ error: errorMessage }))
                        errorMessage = errorData.error || errorMessage
                      }
                      throw new Error(errorMessage)
                    }
                    const data = await response.json().catch(() => ({}))
                    markBackendHealthy()
                    toast.success(data.message || 'Мэппинг контрагентов запущен')
                    // Обновляем статистику через 2 секунды
                    setTimeout(() => {
                      fetchMappingStatus(true)
                    }, 2000)
                  } catch (error) {
                    const errorMessage = error instanceof Error
                      ? (error.name === 'AbortError'
                        ? 'Превышено время ожидания ответа от сервера'
                        : error.message)
                      : 'Неизвестная ошибка при запуске мэппинга'
                    // Не логируем сетевые ошибки подключения к бэкенду - они ожидаемы
                    if (!isBackendConnectionError(errorMessage)) {
                      console.error('Failed to start mapping:', error)
                    }
                    if (isBackendConnectionError(errorMessage)) {
                      notifyBackendUnavailable(errorMessage, true)
                    } else {
                      toast.error(errorMessage)
                    }
                    setIsMappingRunning(false)
                    clearTimeout(timeoutId)
                  }
                }}
                disabled={isMappingRunning || !selectedProjectId}
                variant="default"
              >
                {isMappingRunning ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Запуск...
                  </>
                ) : (
                  <>
                    <Zap className="h-4 w-4 mr-2" />
                    Запустить мэппинг
                  </>
                )}
              </Button>
              <Button
                onClick={() => fetchMappingStatus()}
                variant="outline"
                size="sm"
              >
                <RefreshCw className="h-4 w-4 mr-2" />
                Обновить статистику
              </Button>
              {duplicateGroupCount > 0 && (
                <Button
                  onClick={() => {
                    if (!selectedProjectId) return
                    setShowDuplicatesDialog(true)
                    loadDuplicateGroups()
                  }}
                  variant="outline"
                  size="sm"
                  className="border-orange-300 text-orange-700 hover:bg-orange-50"
                >
                  <Info className="h-4 w-4 mr-2" />
                  Дубликаты ({duplicateGroupCount})
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Диалог настройки конфигурации мэппинга */}
      {selectedProjectId && mappingConfig && (
        <Dialog open={showConfigDialog} onOpenChange={setShowConfigDialog}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Настройка автоматического мэппинга</DialogTitle>
              <DialogDescription>
                Управляйте поведением объединения дубликатов и выбором эталонных контрагентов для проекта
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-6 py-4">
              <div className="flex items-center justify-between space-x-2">
                <div className="space-y-0.5">
                  <Label htmlFor="auto-map" className="text-base font-medium">
                    Автоматический мэппинг
                  </Label>
                  <p className="text-sm text-muted-foreground">
                    Автоматически запускать объединение контрагентов при добавлении баз или нормализации
                  </p>
                </div>
                <Switch
                  id="auto-map"
                  checked={mappingConfig.auto_map_counterparties}
                  onCheckedChange={(checked) => setMappingConfig({
                    ...mappingConfig,
                    auto_map_counterparties: checked,
                  })}
                  disabled={isSavingConfig}
                />
              </div>

              <div className="flex items-center justify-between space-x-2">
                <div className="space-y-0.5">
                  <Label htmlFor="auto-merge" className="text-base font-medium">
                    Автоматическое объединение дубликатов
                  </Label>
                  <p className="text-sm text-muted-foreground">
                    Сразу объединять найденные дубликаты без подтверждения
                  </p>
                </div>
                <Switch
                  id="auto-merge"
                  checked={mappingConfig.auto_merge_duplicates}
                  onCheckedChange={(checked) => setMappingConfig({
                    ...mappingConfig,
                    auto_merge_duplicates: checked,
                  })}
                  disabled={isSavingConfig}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="master-strategy" className="text-base font-medium">
                  Стратегия выбора эталона
                </Label>
                <Select
                  value={mappingConfig.master_selection_strategy || 'max_data'}
                  onValueChange={(value) => setMappingConfig({
                    ...mappingConfig,
                    master_selection_strategy: value,
                  })}
                  disabled={isSavingConfig}
                >
                  <SelectTrigger id="master-strategy">
                    <SelectValue placeholder="Выберите стратегию" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="max_data">
                      <div className="flex flex-col">
                        <span className="font-medium">Максимум данных</span>
                        <span className="text-xs text-muted-foreground">
                          Выбирать контрагента с наиболее полным набором реквизитов
                        </span>
                      </div>
                    </SelectItem>
                    <SelectItem value="max_quality">
                      <div className="flex flex-col">
                        <span className="font-medium">Максимальное качество</span>
                        <span className="text-xs text-muted-foreground">
                          Выбирать запись с наибольшим quality score
                        </span>
                      </div>
                    </SelectItem>
                    <SelectItem value="max_databases">
                      <div className="flex flex-col">
                        <span className="font-medium">Максимум баз данных</span>
                        <span className="text-xs text-muted-foreground">
                          Выбирать запись, привязанную к наибольшему количеству баз
                        </span>
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Определяет, какие данные сохраняются в эталонной записи при объединении
                </p>
              </div>
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => {
                  fetchMappingStatus()
                  setShowConfigDialog(false)
                }}
                disabled={isSavingConfig}
              >
                Отмена
              </Button>
              <Button
                onClick={async () => {
                  if (!selectedProjectId) return
                  setIsSavingConfig(true)
                  try {
                    const response = await fetch(`/api/projects/${selectedProjectId}/normalization-config`, {
                      method: 'PUT',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({
                        auto_map_counterparties: mappingConfig.auto_map_counterparties,
                        auto_merge_duplicates: mappingConfig.auto_merge_duplicates,
                        master_selection_strategy: mappingConfig.master_selection_strategy,
                      }),
                    })
                    if (!response.ok) {
                      throw new Error('Не удалось сохранить конфигурацию')
                    }
                    toast.success('Настройки мэппинга сохранены')
                    await fetchMappingStatus()
                    setShowConfigDialog(false)
                  } catch (error) {
                    const errorMessage = error instanceof Error ? error.message : 'Неизвестная ошибка'
                    // Не логируем сетевые ошибки подключения к бэкенду - они ожидаемы
                    const isNetworkErr = error instanceof Error && (
                      errorMessage.toLowerCase().includes('backend') ||
                      errorMessage.toLowerCase().includes('9999') ||
                      errorMessage.toLowerCase().includes('failed to fetch') ||
                      errorMessage.toLowerCase().includes('networkerror')
                    )
                    if (!isNetworkErr) {
                      console.error('Failed to save mapping config:', error)
                    }
                    toast.error(errorMessage)
                  } finally {
                    setIsSavingConfig(false)
                  }
                }}
                disabled={isSavingConfig}
              >
                {isSavingConfig ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Сохранение...
                  </>
                ) : (
                  'Сохранить'
                )}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      {/* Таблица контрагентов */}
      {error && items.length === 0 ? (
        <ErrorState
          title="Ошибка загрузки"
          message={error}
          action={{
            label: 'Повторить',
            onClick: fetchCounterparties,
          }}
          variant="destructive"
        />
      ) : (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Users className="h-5 w-5" />
                  Контрагенты
                  {filteredItems.length > 0 && (
                    <Badge variant="outline">{filteredItems.length.toLocaleString('ru-RU')}</Badge>
                  )}
                  {selectedSource === null && sourceStats.all > 0 && (
                    <div className="flex gap-1 ml-2 text-xs text-muted-foreground">
                      <span>(БД: {sourceStats.database}, норм: {sourceStats.normalized})</span>
                    </div>
                  )}
                  {hasClientFilters && (
                    <Badge variant="secondary" className="ml-2">
                      Фильтры активны
                    </Badge>
                  )}
                </CardTitle>
                {loadingProgress !== null && (
                  <div className="w-full mt-2 space-y-1">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <span>{loadAllData ? 'Большая выгрузка...' : 'Загрузка данных...'}</span>
                      <span className="font-medium">{Math.round(loadingProgress)}%</span>
                    </div>
                    <Progress value={loadingProgress} className="h-2" />
                    {loadAllData && loadingProgress < 100 && (
                      <div className="text-xs text-muted-foreground text-center">
                        Запрос поставлен в очередь. Максимум {MAX_BACKEND_LIMIT.toLocaleString('ru-RU')} записей. Это может занять до нескольких минут.
                      </div>
                    )}
                  </div>
                )}
                <CardDescription className="space-y-1">
                  <div>
                    Список всех контрагентов клиента (из баз данных и нормализованных)
                    {selectedSource && (
                      <span className="ml-2">
                        • Фильтр: {selectedSource === 'database' ? 'База данных' : 'Нормализованные'}
                      </span>
                    )}
                    {selectedCountry && (
                      <span className="ml-2">
                        • Страна: {selectedCountry}
                      </span>
                    )}
                    {qualityFilter !== 'all' && (
                      <span className="ml-2">
                        • Качество: {
                          qualityFilter === 'high' ? 'Высокое (≥90%)' :
                          qualityFilter === 'medium' ? 'Среднее (70-89%)' :
                          qualityFilter === 'low' ? 'Низкое (&lt;70%)' :
                          qualityFilter === 'no-quality' ? 'Без оценки' : qualityFilter
                        }
                      </span>
                    )}
                    {selectedProjectId && (
                      <span className="ml-2">
                        • Проект: {projects.find(p => p.id === selectedProjectId)?.name || selectedProjectId}
                      </span>
                    )}
                  </div>
                  {filteredItems.length > 0 && (() => {
                    const withQuality = filteredItems.filter(item => item.quality_score !== undefined && item.quality_score !== null)
                    if (withQuality.length === 0) return null
                    const avgQuality = withQuality.reduce((sum, item) => sum + (item.quality_score || 0), 0) / withQuality.length
                    const highQuality = withQuality.filter(item => item.quality_score! >= 0.9).length
                    const mediumQuality = withQuality.filter(item => item.quality_score! >= 0.7 && item.quality_score! < 0.9).length
                    const lowQuality = withQuality.filter(item => item.quality_score! < 0.7).length
                    return (
                      <div className="flex items-center gap-3 text-xs text-muted-foreground mt-1">
                        <span>Среднее качество: <strong>{Math.round(avgQuality * 100)}%</strong></span>
                        <span className="text-green-600">Высокое: {highQuality}</span>
                        <span className="text-yellow-600">Среднее: {mediumQuality}</span>
                        <span className="text-red-600">Низкое: {lowQuality}</span>
                      </div>
                    )
                  })()}
                </CardDescription>
              </div>
              {filteredItems.length > 0 && (
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <span>
                      {items.length !== filteredItems.length ? (
                        <>
                          Показано: {filteredItems.length} из {items.length.toLocaleString('ru-RU')} (отфильтровано)
                        </>
                      ) : (
                        <>
                          Всего: {items.length.toLocaleString('ru-RU')}
                          {totalItems > items.length && (
                            <span className="ml-2 text-xs">
                              (загружено {items.length.toLocaleString('ru-RU')} из {totalItems.toLocaleString('ru-RU')})
                            </span>
                          )}
                        </>
                      )}
                    </span>
                  </div>
                  {totalItems > items.length && !loadAllData && !hasClientFilters && !debouncedSearchQuery && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setLoadAllData(true)
                        setCurrentPage(1)
                        fetchCounterparties()
                      }}
                      disabled={isLoading}
                      title={`Будет загружено до ${MAX_BACKEND_LIMIT.toLocaleString('ru-RU')} записей с использованием очереди. Это может занять до 2 минут.`}
                    >
                      <RefreshCw className="mr-2 h-4 w-4" />
                      Загрузить максимум ({Math.min(totalItems, MAX_BACKEND_LIMIT).toLocaleString('ru-RU')}{totalItems > MAX_BACKEND_LIMIT ? ' из ' + totalItems.toLocaleString('ru-RU') : ''})
                    </Button>
                  )}
                  {loadAllData && items.length >= MAX_BACKEND_LIMIT && totalItems > items.length && (
                    <div className="text-xs text-muted-foreground">
                      Загружено максимум ({MAX_BACKEND_LIMIT.toLocaleString('ru-RU')}). Всего доступно: {totalItems.toLocaleString('ru-RU')}
                    </div>
                  )}
                  <div className="flex items-center gap-2">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="outline" size="sm" disabled={isExporting || filteredItems.length === 0}>
                          {isExporting ? (
                            <>
                              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                              Экспорт...
                            </>
                          ) : (
                            <>
                              <Download className="mr-2 h-4 w-4" />
                              Экспорт
                            </>
                          )}
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel>Экспорт данных</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onClick={() => handleExport('json')}>
                          <FileJson className="mr-2 h-4 w-4" />
                          JSON ({filteredItems.length} записей)
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleExport('csv')}>
                          <FileSpreadsheet className="mr-2 h-4 w-4" />
                          CSV ({filteredItems.length} записей)
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleExport('xml')}>
                          <FileCode className="mr-2 h-4 w-4" />
                          XML ({filteredItems.length} записей)
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {items.length === 0 ? (
              <EmptyState
                icon={Users}
                title="Контрагенты не найдены"
                  description={
                    debouncedSearchQuery || selectedProjectId || hasClientFilters
                      ? 'Попробуйте изменить фильтры поиска или очистить их'
                      : 'В проектах клиента пока нет контрагентов. Загрузите базу данных или запустите нормализацию.'
                  }
                  action={
                    debouncedSearchQuery || selectedProjectId || hasClientFilters
                      ? {
                          label: 'Очистить фильтры',
                          onClick: () => {
                            setSearchQuery('')
                            setSelectedProjectId(null)
                            setSelectedSource(null)
                            setSelectedCountry(null)
                            setQualityFilter('all')
                            setCurrentPage(1)
                          },
                        }
                      : undefined
                  }
              />
            ) : (
              <>
                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>
                          <button
                            onClick={() => handleSort('name')}
                            className="flex items-center hover:text-foreground"
                          >
                            Название
                            {getSortIcon('name')}
                          </button>
                        </TableHead>
                        <TableHead>
                          <button
                            onClick={() => handleSort('normalized_name')}
                            className="flex items-center hover:text-foreground"
                          >
                            Нормализованное название
                            {getSortIcon('normalized_name')}
                          </button>
                        </TableHead>
                        <TableHead>
                          <button
                            onClick={() => handleSort('tax_id')}
                            className="flex items-center hover:text-foreground"
                          >
                            ИНН/БИН
                            {getSortIcon('tax_id')}
                          </button>
                        </TableHead>
                        <TableHead>
                          <button
                            onClick={() => handleSort('type')}
                            className="flex items-center hover:text-foreground"
                          >
                            Источник
                            {getSortIcon('type')}
                          </button>
                        </TableHead>
                        <TableHead>
                          <button
                            onClick={() => handleSort('country')}
                            className="flex items-center hover:text-foreground"
                          >
                            Страна
                            {getSortIcon('country')}
                          </button>
                        </TableHead>
                        <TableHead>
                          <button
                            onClick={() => handleSort('status')}
                            className="flex items-center hover:text-foreground"
                          >
                            Статус
                            {getSortIcon('status')}
                          </button>
                        </TableHead>
                        <TableHead className="text-right">Действия</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {sortedItems.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage).map((item) => (
                        <CounterpartyRow 
                          key={item.uniqueKey || `${item.type}-${item.id}`}
                          item={item}
                          onView={handleViewItem}
                        />
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {/* Пагинация */}
                {filteredTotalPages > 1 && (
                  <Pagination
                    currentPage={currentPage}
                    totalPages={filteredTotalPages}
                    onPageChange={setCurrentPage}
                    itemsPerPage={itemsPerPage}
                    totalItems={filteredTotalItems}
                    className="mt-4"
                  />
                )}
              </>
            )}
          </CardContent>
        </Card>
      )}

      {selectedItem && (
        <CounterpartyDetailDialog
          item={selectedItem}
          open={!!selectedItem}
          onOpenChange={(open) => !open && setSelectedItem(null)}
        />
      )}

      <CounterpartyDuplicatesDialog
        open={showDuplicatesDialog}
        onOpenChange={(open) => {
          setShowDuplicatesDialog(open)
          if (open) {
            loadDuplicateGroups()
          }
        }}
        groups={duplicateGroupsData}
        isLoading={isDuplicatesLoading}
        error={duplicatesError}
        onRefresh={loadDuplicateGroups}
      />
    </div>
  )
}

