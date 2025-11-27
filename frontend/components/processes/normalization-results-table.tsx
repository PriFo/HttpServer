'use client'

import { useState, useEffect, useCallback, useRef, memo, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Pagination } from '@/components/ui/pagination'
import { ChevronDown, ChevronUp, RefreshCw, ArrowUpDown, ArrowUp, ArrowDown, Users, Layers, Activity } from 'lucide-react'
import { ErrorState } from '@/components/common/error-state'
import { AttributesDisplay } from './attributes-display'
import { useProjectSearchParams } from '@/hooks/useProjectSearchParams'
import { motion, AnimatePresence } from 'framer-motion'
import type { NormalizationType } from '@/types/normalization'
import { logger } from '@/lib/logger'
import { handleErrorWithDetails as handleError } from '@/lib/error-handler'

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

interface GroupItem {
  id: number
  source_reference: string
  source_name: string
  code: string
  attributes?: ItemAttribute[]
}

interface ItemAttribute {
  id: number
  attribute_type: string
  attribute_name: string
  attribute_value: string
  unit?: string
  original_text?: string
  confidence?: number
}

interface NormalizationResultsTableProps {
  isRunning: boolean
  database?: string
  project?: string // Формат: "clientId:projectId"
  projectType?: string | null // Тип проекта: 'counterparty', 'nomenclature', 'nomenclature_counterparties'
  normalizationType?: NormalizationType
  showOnlyWhenRunning?: boolean
}

// ============================================================================
// Memoized Components for Performance Optimization
// ============================================================================

/**
 * Memoized attribute card component
 * Prevents re-renders when parent updates but attribute data hasn't changed
 */
const AttributeCard = memo<{ attr: ItemAttribute }>(({ attr }) => (
  <div className="bg-muted/50 border rounded p-2 text-xs">
    <div className="flex items-center justify-between mb-1">
      <span className="font-medium text-muted-foreground uppercase">
        {attr.attribute_name || attr.attribute_type}
      </span>
      {attr.confidence !== undefined && attr.confidence < 1.0 && (
        <span className="text-muted-foreground">
          {(attr.confidence * 100).toFixed(0)}%
        </span>
      )}
    </div>
    <div className="flex items-baseline gap-1">
      <span className="font-semibold">{attr.attribute_value}</span>
      {attr.unit && <span className="text-muted-foreground">{attr.unit}</span>}
    </div>
    {attr.original_text && (
      <div className="text-muted-foreground mt-1 text-xs">
        Из: "{attr.original_text}"
      </div>
    )}
    <div className="mt-1">
      <Badge variant="outline" className="text-xs">
        {attr.attribute_type}
      </Badge>
    </div>
  </div>
))
AttributeCard.displayName = 'AttributeCard'

/**
 * Memoized group item component
 * Prevents re-renders when parent updates but item data hasn't changed
 * Now uses AttributesDisplay for better attribute visualization
 */
const GroupItemCard = memo<{ item: GroupItem; showAttributes?: boolean }>(({ item, showAttributes = true }) => (
  <div className="bg-background border rounded-lg p-3 space-y-3">
    <div className="flex items-start justify-between">
      <div className="flex-1">
        <div className="font-medium text-sm">{item.source_name}</div>
        <div className="text-xs text-muted-foreground mt-1">
          Код: {item.code} | Reference: {item.source_reference}
        </div>
      </div>
    </div>

    {showAttributes && (
      <div className="mt-2 pt-2 border-t">
        <div className="text-sm font-semibold text-foreground mb-3">
          Извлеченные реквизиты:
        </div>
        {item.attributes && item.attributes.length > 0 ? (
          <AttributesDisplay 
            attributes={item.attributes} 
            loading={false}
            compact={false}
          />
        ) : (
          <div className="p-4 text-center text-sm text-muted-foreground italic">
            Реквизиты не извлечены для данного элемента
          </div>
        )}
      </div>
    )}
  </div>
))
GroupItemCard.displayName = 'GroupItemCard'

/**
 * Memoized group row component
 * Only re-renders when group data, expansion state, or items change
 */
interface GroupRowProps {
  group: Group
  isExpanded: boolean
  items: GroupItem[]
  isLoadingGroup: boolean
  attributeCount: number
  onToggleExpansion: () => void
  projectType?: string | null
}

const GroupRow = memo<GroupRowProps>(({
  group,
  isExpanded,
  items,
  isLoadingGroup,
  attributeCount,
  onToggleExpansion,
  projectType,
}) => {
  const groupKey = `${group.normalized_name}|${group.category}`

  return (
    <>
      <TableRow key={groupKey}>
        <TableCell>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={onToggleExpansion}
          >
            {isExpanded ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </Button>
        </TableCell>
        <TableCell>
          <div className="font-medium max-w-[400px] truncate" title={group.normalized_name}>
            {group.normalized_name}
          </div>
        </TableCell>
        <TableCell>
          <Badge variant="outline">{group.category}</Badge>
        </TableCell>
        {projectType !== 'counterparty' && (
          <TableCell>
            {group.kpved_code ? (
              <div className="space-y-1">
                <div className="font-medium text-sm">{group.kpved_code}</div>
                {group.kpved_name && (
                  <div className="text-xs text-muted-foreground max-w-[200px] truncate" title={group.kpved_name}>
                    {group.kpved_name}
                  </div>
                )}
                {group.kpved_confidence !== undefined && (
                  <div className="text-xs text-muted-foreground">
                    Уверенность: {(group.kpved_confidence * 100).toFixed(1)}%
                  </div>
                )}
              </div>
            ) : (
              <span className="text-muted-foreground text-sm">—</span>
            )}
          </TableCell>
        )}
        <TableCell className="text-center">
          <Badge variant="secondary">{group.merged_count}</Badge>
        </TableCell>
        <TableCell className="text-center">
          {isExpanded ? (
            <Badge variant={attributeCount > 0 ? 'default' : 'secondary'}>
              {attributeCount}
            </Badge>
          ) : (
            <span className="text-muted-foreground">—</span>
          )}
        </TableCell>
      </TableRow>

      {isExpanded && (
        <TableRow>
          <TableCell colSpan={projectType === 'counterparty' ? 5 : 6} className="bg-muted/30 p-0">
            <div className="p-4 space-y-4">
              {isLoadingGroup ? (
                <div className="flex items-center justify-center py-4">
                  <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />
                  <span className="ml-2 text-sm text-muted-foreground">Загрузка...</span>
                </div>
              ) : items.length === 0 ? (
                <div className="text-center py-4 text-muted-foreground text-sm">
                  Нет исходных записей
                </div>
              ) : (
                <div className="space-y-2">
                  <h4 className="text-sm font-medium">Исходные записи ({items.length}):</h4>
                  <div className="space-y-2">
                    {items.map((item) => (
                      <GroupItemCard key={item.id} item={item} showAttributes={true} />
                    ))}
                  </div>
                </div>
              )}
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  )
})
GroupRow.displayName = 'GroupRow'

export function NormalizationResultsTable({ 
  isRunning, 
  database, 
  project, 
  projectType,
  normalizationType = 'both',
  showOnlyWhenRunning = false
}: NormalizationResultsTableProps) {
  const [groups, setGroups] = useState<Group[]>([])
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set())
  const [groupItems, setGroupItems] = useState<Map<string, GroupItem[]>>(new Map())
  const [loadingItems, setLoadingItems] = useState<Set<string>>(new Set())
  const loadingRef = useRef<Set<string>>(new Set())
  const [isLoading, setIsLoading] = useState(false)
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [total, setTotal] = useState(0)
  const [limit, setLimit] = useState(10) // Показываем первые 10 групп на странице процессов
  const [error, setError] = useState<string | null>(null)

  // Добавляем состояние для clientId, projectId и dbId
  const [clientId, setClientId] = useState<number | null>(null)
  const [projectId, setProjectId] = useState<number | null>(null)
  const [dbId, setDbId] = useState<number | null>(null)

  // URL параметры (используем только если есть clientId и projectId)
  const urlParams = useProjectSearchParams(
    clientId ? String(clientId) : null,
    projectId ? String(projectId) : null,
    { resetOnProjectChange: true }
  )

  // Получаем параметры из URL или используем локальное состояние
  const urlPage = urlParams.getParam('page', '1')
  const urlFilter = urlParams.getParam('filter', '')
  const urlSortKey = urlParams.getParam('sortKey', '')
  const urlSortDirection = urlParams.getParam('sortDirection', '') as 'asc' | 'desc' | ''

  // Состояние для сортировки (синхронизируем с URL)
  const [sortKey, setSortKey] = useState<string | null>(urlSortKey || null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc' | null>(
    urlSortDirection === 'asc' || urlSortDirection === 'desc' ? urlSortDirection : null
  )
  
  // Синхронизация URL параметров с состоянием при загрузке
  useEffect(() => {
    if (urlPage && urlPage !== '1') {
      const pageNum = parseInt(urlPage, 10)
      if (!isNaN(pageNum) && pageNum !== currentPage && pageNum > 0) {
        setCurrentPage(pageNum)
      }
    } else if (!urlPage && currentPage !== 1) {
      setCurrentPage(1)
    }
  }, [urlPage, currentPage])

  useEffect(() => {
    if (urlSortKey && urlSortKey !== sortKey) {
      setSortKey(urlSortKey)
    } else if (!urlSortKey && sortKey) {
      setSortKey(null)
    }
  }, [urlSortKey, sortKey])

  useEffect(() => {
    if (urlSortDirection && (urlSortDirection === 'asc' || urlSortDirection === 'desc')) {
      if (urlSortDirection !== sortDirection) {
        setSortDirection(urlSortDirection)
      }
    } else if (!urlSortDirection && sortDirection) {
      setSortDirection(null)
    }
  }, [urlSortDirection, sortDirection])

  // Обновление URL при изменении пагинации
  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page)
    if (clientId && projectId) {
      urlParams.setParam('page', page.toString())
    }
  }, [clientId, projectId, urlParams])

  // Обновление URL при изменении сортировки
  const handleSortChange = useCallback((key: string | null, direction: 'asc' | 'desc' | null) => {
    setSortKey(key)
    setSortDirection(direction)
    if (clientId && projectId) {
      if (key && direction) {
        urlParams.setParams({
          sortKey: key,
          sortDirection: direction,
          page: '1', // Сбрасываем на первую страницу при изменении сортировки
        })
      } else {
        urlParams.setParams({
          sortKey: null,
          sortDirection: null,
        })
      }
    }
  }, [clientId, projectId, urlParams])

  // useRef для отслеживания текущего проекта и предотвращения race conditions
  const currentProjectRef = useRef<string | null>(null)
  const abortControllerRef = useRef<AbortController | null>(null)

  // Очистка состояния при смене проекта
  useEffect(() => {
    const projectKey = project ? project : database ? `db:${database}` : null
    const clientProjectKey = clientId && projectId ? `${clientId}:${projectId}` : null
    const effectiveKey = projectKey || clientProjectKey || 'none'

    // Если проект изменился, сбрасываем все состояние
    if (currentProjectRef.current !== effectiveKey) {
      // Отменяем предыдущие запросы
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }

      // Сбрасываем состояние
      setGroups([])
      setExpandedGroups(new Set())
      setGroupItems(new Map())
      setLoadingItems(new Set())
      loadingRef.current.clear()
      setCurrentPage(1)
      setTotalPages(1)
      setTotal(0)
      setError(null)
      setIsLoading(false)
      setSortKey(null)
      setSortDirection(null)

      // Сбрасываем URL параметры при смене проекта (если есть clientId и projectId)
      if (clientId && projectId) {
        urlParams.resetParams()
      }

      // Обновляем ref
      currentProjectRef.current = effectiveKey
    }
  }, [project, database, clientId, projectId, dbId])

  // Повторная загрузка при смене проекта
  useEffect(() => {
    if ((project && projectId) || (clientId && projectId)) {
      // Небольшая задержка, чтобы дать время для очистки состояния
      const timeoutId = setTimeout(() => {
        fetchGroups()
      }, 100)
      return () => clearTimeout(timeoutId)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project, projectId, clientId])

  // Парсим проект из пропса или находим клиента и проект по базе данных
  useEffect(() => {
    if (project) {
      const parts = project.split(':')
      if (parts.length === 2) {
        const clientIdNum = parseInt(parts[0], 10)
        const projectIdNum = parseInt(parts[1], 10)
        if (!isNaN(clientIdNum) && !isNaN(projectIdNum)) {
          setClientId(clientIdNum)
          setProjectId(projectIdNum)
          // Для проектов без конкретной базы данных dbId остается null
          setDbId(null)
        } else {
          setClientId(null)
          setProjectId(null)
          setDbId(null)
        }
      } else {
        setClientId(null)
        setProjectId(null)
        setDbId(null)
      }
    } else if (database) {
      // Отменяем предыдущий запрос, если он существует
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }

      const controller = new AbortController()
      abortControllerRef.current = controller
      const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут
      
      fetch(`/api/databases/find-project?file_path=${encodeURIComponent(database)}`, {
        signal: controller.signal,
        cache: 'no-store',
      })
        .then(res => {
          clearTimeout(timeoutId)
          if (!res.ok) {
            // Если 404 - база данных не найдена в проектах, это нормально
            if (res.status === 404) {
              logger.debug('Database not found in any project', { component: 'NormalizationResultsTable', database })
              return null
            }
            // Для других ошибок логируем
            if (res.status === 500) {
              logger.warn('Server error when finding project', {
                component: 'NormalizationResultsTable',
                database,
                status: res.status,
                statusText: res.statusText
              })
            } else {
              logger.warn('Failed to find project', {
                component: 'NormalizationResultsTable',
                database,
                status: res.status,
                statusText: res.statusText
              })
            }
            return null
          }
          return res.json()
        })
        .then(data => {
          if (data && data.client_id && data.project_id) {
            setClientId(data.client_id)
            setProjectId(data.project_id)
            setDbId(data.db_id || null)
          } else {
            setClientId(null)
            setProjectId(null)
            setDbId(null)
          }
        })
        .catch(err => {
          clearTimeout(timeoutId)
          if (err.name === 'AbortError') {
            logger.warn('Find project request timed out', { component: 'NormalizationResultsTable', database })
          } else {
            logger.error('Failed to find project', { component: 'NormalizationResultsTable', database }, err instanceof Error ? err : undefined)
          }
          setClientId(null)
          setProjectId(null)
          setDbId(null)
        })
      
      return () => {
        clearTimeout(timeoutId)
        controller.abort()
      }
    } else {
      setClientId(null)
      setProjectId(null)
      setDbId(null)
    }
  }, [database, project]) // Добавляем project в зависимости

  const fetchGroups = useCallback(async () => {
    // Проверяем, что проект не изменился во время запроса
    const projectKey = project ? project : database ? `db:${database}` : null
    const clientProjectKey = clientId && projectId ? `${clientId}:${projectId}` : null
    const effectiveKey = projectKey || clientProjectKey || 'none'

    if (currentProjectRef.current !== effectiveKey) {
      // Проект изменился, не выполняем запрос
      return
    }

    setIsLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({
        page: currentPage.toString(),
        limit: limit.toString(),
        include_ai: 'true',
      })

      // Определяем тип проекта и используем соответствующий API
      const isCounterpartyProject = projectType === 'counterparty' || projectType === 'nomenclature_counterparties'
      
      let apiUrl = '/api/normalization/groups'
      let useClientProjectEndpoint = false
      
      // Проверяем наличие обязательных параметров для client/project endpoint
      if (clientId && projectId && dbId) {
        // Используем client/project endpoint только если есть dbId
        useClientProjectEndpoint = true
        apiUrl = `/api/clients/${clientId}/projects/${projectId}/normalization/groups`
        params.append('db_id', dbId.toString())
      } else if (database) {
        // Fallback: используем старый API с параметром database
        params.append('database', database)
      } else if (clientId && projectId && !dbId) {
        // Если есть clientId/projectId, но нет dbId, не используем client/project endpoint
        // Используем fallback endpoint без db_id
        logger.warn('db_id is missing for client/project endpoint, using fallback endpoint', {
          component: 'NormalizationResultsTable',
          clientId,
          projectId
        })
        useClientProjectEndpoint = false
        // Не добавляем db_id в параметры, используем общий endpoint
      } else {
        // Если нет ни clientId/projectId, ни database, используем общий endpoint
        useClientProjectEndpoint = false
      }

      const response = await fetch(`${apiUrl}?${params}`, {
        cache: 'no-store',
        signal: AbortSignal.timeout(10000), // 10 секунд таймаут
      })
      
      if (!response.ok) {
        const errorText = await response.text().catch(() => 'Unknown error')
        let errorData: any = {}
        try {
          errorData = JSON.parse(errorText)
        } catch {
          // Если не JSON, используем текст как есть
        }
        
        // Если это 404 или 400, и мы использовали client/project endpoint, пробуем fallback
        if ((response.status === 404 || response.status === 400) && useClientProjectEndpoint) {
          logger.warn('Client-specific groups endpoint failed, trying fallback endpoint', {
            component: 'NormalizationResultsTable',
            clientId,
            projectId
          })
          try {
            const fallbackParams = new URLSearchParams({
              page: currentPage.toString(),
              limit: limit.toString(),
              include_ai: 'true',
            })
            if (database) {
              fallbackParams.append('database', database)
            } else if (dbId && clientId && projectId) {
              // Если dbId есть, но endpoint не сработал, попробуем без него
              logger.warn('Trying fallback without db_id parameter', {
                component: 'NormalizationResultsTable',
                clientId,
                projectId
              })
            }
            const fallbackResponse = await fetch(`/api/normalization/groups?${fallbackParams}`, {
              cache: 'no-store',
              signal: AbortSignal.timeout(10000),
            })
            if (fallbackResponse.ok) {
              const fallbackData = await fallbackResponse.json()
              setGroups(fallbackData.groups || [])
              setTotalPages(fallbackData.totalPages || 1)
              setTotal(fallbackData.total || 0)
              return
            }
          } catch {
            // Игнорируем ошибки fallback
          }
        }
        
        throw new Error(errorData.error || errorText || 'Не удалось загрузить группы')
      }

      const data = await response.json()
      
      // Проверяем, что проект не изменился во время запроса
      if (currentProjectRef.current !== effectiveKey) {
        return
      }

      setGroups(data.groups || [])
      setTotalPages(data.totalPages || 1)
      setTotal(data.total || 0)
      setError(null)
    } catch (err) {
      const errorDetails = handleError(
        err,
        'NormalizationResultsTable',
        'fetchGroups',
        { clientId, projectId, database, project, currentPage, limit }
      )
      logger.error('Error fetching groups', {
        component: 'NormalizationResultsTable',
        clientId,
        projectId,
        database,
        project,
        currentPage,
        limit,
        errorMessage: errorDetails.message
      }, err instanceof Error ? err : undefined)
      if (err instanceof Error && err.name !== 'AbortError') {
        // Проверяем, что проект не изменился во время ошибки
        if (currentProjectRef.current === effectiveKey) {
          setError(err.message)
        }
      }
    } finally {
      // Проверяем, что проект не изменился перед обновлением состояния загрузки
      if (currentProjectRef.current === effectiveKey) {
        setIsLoading(false)
      }
    }
  }, [currentPage, limit, database, clientId, projectId, dbId, projectType, project]) // Добавляем project в зависимости

  const fetchGroupItems = useCallback(async (normalizedName: string, category: string) => {
    // Проверяем, что проект не изменился
    const projectKey = project ? project : database ? `db:${database}` : null
    const clientProjectKey = clientId && projectId ? `${clientId}:${projectId}` : null
    const effectiveKey = projectKey || clientProjectKey || 'none'

    if (currentProjectRef.current !== effectiveKey) {
      // Проект изменился, не загружаем элементы
      return
    }

    const groupKey = `${normalizedName}|${category}`
    
    // Проверяем, не загружены ли уже элементы или не загружается ли уже
    let shouldLoad = false
    setGroupItems(prev => {
      // Если уже загружены или загружается, выходим
      if (prev.has(groupKey) || loadingRef.current.has(groupKey)) {
        return prev
      }
      shouldLoad = true
      return prev
    })

    // Если не нужно загружать или уже загружается, выходим
    if (!shouldLoad || loadingRef.current.has(groupKey)) {
      return
    }

    // Отмечаем как загружающийся
    loadingRef.current.add(groupKey)
    setLoadingItems(prev => new Set(prev).add(groupKey))
    
    try {
      const params = new URLSearchParams({
        normalized_name: normalizedName,
        category: category,
        include_ai: 'true',
        include_attributes: 'false',
      })

      const response = await fetch(`/api/normalization/group-items?${params}`, {
        cache: 'no-store',
        headers: {
          'Cache-Control': 'no-cache',
        },
      })
      
      if (!response.ok) {
        const errorText = await response.text().catch(() => 'Unknown error')
        let errorMessage = 'Не удалось загрузить элементы группы'
        if (response.status === 404) {
          errorMessage = 'Элементы группы не найдены'
        } else if (response.status >= 500) {
          errorMessage = 'Ошибка сервера. Попробуйте позже'
        }
        throw new Error(errorMessage)
      }

      const data = await response.json()
      setError(null)
      const items: GroupItem[] = data.items || []
      
      // Проверяем, что проект не изменился перед загрузкой атрибутов
      if (currentProjectRef.current !== effectiveKey) {
        return // Проект изменился, не загружаем атрибуты
      }
      
      // Загружаем атрибуты для каждого элемента
      const itemsWithAttributes = await Promise.all(
        items.map(async (item) => {
          // Проверяем, что проект не изменился во время загрузки атрибутов
          if (currentProjectRef.current !== effectiveKey) {
            return item // Проект изменился, возвращаем элемент без атрибутов
          }

          try {
            const attrResponse = await fetch(`/api/normalization/item-attributes/${item.id}`, {
              cache: 'no-store',
              headers: {
                'Cache-Control': 'no-cache',
              },
            })
            if (attrResponse.ok) {
              const attrData = await attrResponse.json()
              return { ...item, attributes: attrData.attributes || [] }
            }
          } catch (error) {
            // Игнорируем ошибки загрузки атрибутов - это не критично
            logger.debug('Could not fetch attributes for item', {
              component: 'NormalizationResultsTable',
              itemId: item.id,
              error: error instanceof Error ? error.message : String(error)
            })
          }
          return item
        })
      )

      // Проверяем, что проект не изменился перед сохранением данных
      if (currentProjectRef.current === effectiveKey) {
        setGroupItems(prev => {
          // Проверяем, не загружены ли уже элементы (на случай параллельных запросов)
          if (prev.has(groupKey)) {
            return prev
          }
          return new Map(prev).set(groupKey, itemsWithAttributes)
        })
      }
    } catch (error) {
      const errorDetails = handleError(
        error,
        'NormalizationResultsTable',
        'fetchGroupItems',
        { normalizedName, category }
      )
      logger.error('Error fetching group items', {
        component: 'NormalizationResultsTable',
        normalizedName,
        category,
        errorMessage: errorDetails.message
      }, error instanceof Error ? error : undefined)
    } finally {
      loadingRef.current.delete(groupKey)
      setLoadingItems(prev => {
        const next = new Set(prev)
        next.delete(groupKey)
        return next
      })
    }
  }, [project, database, clientId, projectId])

  // Загрузка данных и автообновление при работе процесса
  useEffect(() => {
    // Начальная загрузка
    fetchGroups()

    // Если процесс работает, запускаем polling
    if (isRunning) {
      const interval = setInterval(fetchGroups, 3000)
      return () => clearInterval(interval)
    }
  }, [fetchGroups, isRunning]) // fetchGroups уже включает все зависимости

  const toggleGroupExpansion = useCallback((normalizedName: string, category: string) => {
    const groupKey = `${normalizedName}|${category}`
    const newExpanded = new Set(expandedGroups)

    if (newExpanded.has(groupKey)) {
      newExpanded.delete(groupKey)
    } else {
      newExpanded.add(groupKey)
      // Загружаем элементы группы при раскрытии
      fetchGroupItems(normalizedName, category)
    }

    setExpandedGroups(newExpanded)
  }, [expandedGroups, fetchGroupItems])

  const getAttributeCount = useCallback((groupKey: string): number => {
    const items = groupItems.get(groupKey) || []
    return items.reduce((count, item) => count + (item.attributes?.length || 0), 0)
  }, [groupItems])

  // Функция для обработки сортировки
  const handleSort = (key: string) => {
    let newDirection: 'asc' | 'desc' | null = 'asc'
    if (sortKey === key) {
      if (sortDirection === 'asc') {
        newDirection = 'desc'
      } else if (sortDirection === 'desc') {
        newDirection = null
      }
    }

    // Используем handleSortChange для синхронизации с URL
    handleSortChange(newDirection ? key : null, newDirection)
  }

  // Сортировка данных
  const sortedGroups = useMemo(() => {
    if (!sortKey || !sortDirection) return groups

    return [...groups].sort((a, b) => {
      let aValue: any
      let bValue: any

      switch (sortKey) {
        case 'normalized_name':
          aValue = a.normalized_name
          bValue = b.normalized_name
          break
        case 'category':
          aValue = a.category
          bValue = b.category
          break
        case 'kpved_code':
          aValue = a.kpved_code || ''
          bValue = b.kpved_code || ''
          break
        case 'merged_count':
          aValue = a.merged_count
          bValue = b.merged_count
          break
        case 'kpved_confidence':
          aValue = a.kpved_confidence ?? 0
          bValue = b.kpved_confidence ?? 0
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
  }, [groups, sortKey, sortDirection])

  const isCounterpartyView = projectType === 'counterparty' || projectType === 'nomenclature_counterparties'

  const mobileGroups = useMemo(() => {
    if (!isCounterpartyView || sortedGroups.length === 0) {
      return []
    }
    const active = sortedGroups.filter(
      (group) =>
        group.processing_level &&
        group.processing_level.toLowerCase() !== 'completed' &&
        group.processing_level.toLowerCase() !== 'готово'
    )
    const source = isRunning && active.length > 0 ? active : sortedGroups
    return source.slice(0, Math.min(5, source.length))
  }, [isCounterpartyView, sortedGroups, isRunning])

  // Функция для получения иконки сортировки
  const getSortIcon = (key: string) => {
    if (sortKey !== key) {
      return <ArrowUpDown className="ml-2 h-3 w-3 opacity-50" />
    }
    if (sortDirection === 'asc') {
      return <ArrowUp className="ml-2 h-3 w-3" />
    }
    if (sortDirection === 'desc') {
      return <ArrowDown className="ml-2 h-3 w-3" />
    }
    return <ArrowUpDown className="ml-2 h-3 w-3 opacity-50" />
  }

  const handlePageSizeChange = (newLimit: string) => {
    setLimit(Number(newLimit))
    // Используем handlePageChange для синхронизации с URL
    handlePageChange(1)
  }

  // Условное отображение: скрывать таблицу до запуска процесса, если указано
  if (showOnlyWhenRunning && !isRunning && groups.length === 0) {
    return null
  }

  if (isLoading && groups.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Результаты нормализации</CardTitle>
          <CardDescription>Загрузка данных...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error && groups.length === 0 && !isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Результаты нормализации</CardTitle>
          <CardDescription>
            Ошибка загрузки данных
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ErrorState
            message={error}
            action={{
              label: "Повторить",
              onClick: () => fetchGroups()
            }}
          />
        </CardContent>
      </Card>
    )
  }

  if (groups.length === 0 && !isLoading && !error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Результаты нормализации</CardTitle>
          <CardDescription>
            {isRunning ? 'Обработка данных...' : 'Нет данных для отображения'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            {isRunning 
              ? 'Дождитесь завершения нормализации для просмотра результатов'
              : 'Запустите процесс нормализации для просмотра результатов'}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <Card className="backdrop-blur-sm bg-card/95 border-border/50 shadow-lg">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Результаты нормализации</CardTitle>
            <CardDescription>
              Показано {sortedGroups.length} из {total.toLocaleString()} групп
              {error && (
                <span className="text-yellow-600 dark:text-yellow-400 ml-2">
                  ({error})
                </span>
              )}
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {error && (
              <Button
                onClick={() => fetchGroups()}
                variant="outline"
                size="sm"
                disabled={isLoading}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                Обновить
              </Button>
            )}
            {isRunning && (
              <Badge variant="default" className="animate-pulse">
                Обновление...
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isCounterpartyView && (
          <div className="md:hidden mb-6 space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">
                {isRunning ? 'В обработке группы контрагентов' : 'Последние обработанные группы'}
              </p>
              {isRunning ? (
                <Badge variant="outline" className="text-xs">
                  В работе
                </Badge>
              ) : (
                <span className="text-xs text-muted-foreground">обновлено автоматически</span>
              )}
            </div>
            {mobileGroups.length === 0 ? (
              <div className="rounded-lg border p-3 text-xs text-muted-foreground">
                Нет данных для отображения. Запустите нормализацию, чтобы увидеть активные группы.
              </div>
            ) : (
              <div className="space-y-3">
                {mobileGroups.map((group, index) => {
                  const badgeLabel = group.processing_level || 'Очередь'
                  return (
                    <div
                      key={`${group.normalized_name}-${group.category}-${index}`}
                      className="rounded-lg border bg-muted/40 p-3 space-y-3"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="min-w-0">
                          <p className="text-sm font-semibold truncate">
                            {group.normalized_name || 'Без названия'}
                          </p>
                          <p className="text-xs text-muted-foreground truncate flex items-center gap-1">
                            <Layers className="h-3 w-3" />
                            {group.category || 'Категория не указана'}
                          </p>
                        </div>
                        <Badge variant="secondary" className="text-[10px] uppercase tracking-wide">
                          {badgeLabel}
                        </Badge>
                      </div>
                      <div className="grid grid-cols-2 gap-3 text-xs text-muted-foreground">
                        <div className="flex items-center gap-2">
                          <Users className="h-3.5 w-3.5 text-primary" />
                          <span className="font-medium text-foreground">
                            {group.merged_count?.toLocaleString('ru-RU') ?? 0}
                          </span>
                          <span className="text-[11px]">контрагентов</span>
                        </div>
                        <div className="flex items-center gap-2 justify-end">
                          <Activity className="h-3.5 w-3.5 text-primary" />
                          {group.kpved_code ? (
                            <span className="text-[11px] text-foreground truncate max-w-[80px]">
                              {group.kpved_code}
                            </span>
                          ) : (
                            <span className="text-[11px]">без КПВЭД</span>
                          )}
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )}
        <ScrollArea className="h-[500px]">
          <Table>
            <TableHeader className="sticky top-0 bg-background z-10">
              <TableRow>
                <TableHead className="w-[50px]"></TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    className="h-8 px-2 hover:bg-transparent"
                    onClick={() => handleSort('normalized_name')}
                  >
                    Нормализованное имя
                    {getSortIcon('normalized_name')}
                  </Button>
                </TableHead>
                <TableHead>
                  <Button
                    variant="ghost"
                    className="h-8 px-2 hover:bg-transparent"
                    onClick={() => handleSort('category')}
                  >
                    Категория
                    {getSortIcon('category')}
                  </Button>
                </TableHead>
                {projectType !== 'counterparty' && (
                  <TableHead>
                    <Button
                      variant="ghost"
                      className="h-8 px-2 hover:bg-transparent"
                      onClick={() => handleSort('kpved_code')}
                    >
                      КПВЭД
                      {getSortIcon('kpved_code')}
                    </Button>
                  </TableHead>
                )}
                <TableHead className="text-center">
                  <Button
                    variant="ghost"
                    className="h-8 px-2 hover:bg-transparent"
                    onClick={() => handleSort('merged_count')}
                  >
                    Объединено
                    {getSortIcon('merged_count')}
                  </Button>
                </TableHead>
                <TableHead className="text-center">Атрибутов</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedGroups.map((group, index) => {
                const groupKey = `${group.normalized_name}|${group.category}`
                // Используем groupKey с индексом для React key, чтобы гарантировать уникальность
                // groupKey остается для внутренней логики (expandedGroups, groupItems и т.д.)
                const reactKey = `${groupKey}-${index}`
                return (
                  <GroupRow
                    key={reactKey}
                    group={group}
                    isExpanded={expandedGroups.has(groupKey)}
                    items={groupItems.get(groupKey) || []}
                    isLoadingGroup={loadingItems.has(groupKey)}
                    attributeCount={getAttributeCount(groupKey)}
                    onToggleExpansion={() => toggleGroupExpansion(group.normalized_name, group.category)}
                    projectType={projectType}
                  />
                )
              })}
            </TableBody>
          </Table>
        </ScrollArea>
        <div className="mt-4 flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span>Показывать:</span>
            <Select value={limit.toString()} onValueChange={handlePageSizeChange}>
              <SelectTrigger className="w-[100px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10">10</SelectItem>
                <SelectItem value="20">20</SelectItem>
                <SelectItem value="50">50</SelectItem>
                <SelectItem value="100">100</SelectItem>
              </SelectContent>
            </Select>
            <span>на странице</span>
          </div>
          {totalPages > 1 && (
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={handlePageChange}
              itemsPerPage={limit}
              totalItems={total}
            />
          )}
        </div>
      </CardContent>
    </Card>
    </motion.div>
  )
}
