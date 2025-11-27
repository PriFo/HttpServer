'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { 
  Package, 
  Search,
  Eye,
  ChevronLeft,
  ChevronRight,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Download,
  FileSpreadsheet,
  FileCode,
  FileJson
} from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useMemo } from 'react'
import { LoadingState } from "@/components/common/loading-state"
import { ErrorState } from "@/components/common/error-state"
import { NomenclatureDetailDialog } from "./nomenclature-detail-dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

interface NomenclatureItem {
  id: number
  code: string
  name: string
  normalized_name: string
  category: string
  quality_score: number
  status: string
  merged_count: number
  kpved_code?: string
  kpved_name?: string
  source_database?: string
  source_type?: string
  project_id?: number
  project_name?: string
}

interface NomenclatureTabProps {
  clientId: string
  projects: Array<{
    id: number
    name: string
    project_type: string
    status: string
  }>
}

export function NomenclatureTab({ clientId, projects }: NomenclatureTabProps) {
  const [items, setItems] = useState<NomenclatureItem[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState("")
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [totalItems, setTotalItems] = useState(0)
  const [selectedItem, setSelectedItem] = useState<NomenclatureItem | null>(null)
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc' | null>(null)
  const [isExporting, setIsExporting] = useState(false)
  const [sourceTypeFilter, setSourceTypeFilter] = useState<string>("all")
  const itemsPerPage = 20

  // useRef для отслеживания текущего клиента и предотвращения race conditions
  const currentClientRef = useRef<string | null>(null)
  const abortControllerRef = useRef<AbortController | null>(null)

  // Сброс состояния при изменении clientId
  useEffect(() => {
    const clientIdStr = String(clientId)
    
    // Если клиент изменился, сбрасываем все состояние
    if (currentClientRef.current !== clientIdStr && currentClientRef.current !== null) {
      // Отменяем предыдущие запросы
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      
      // Сбрасываем состояние
      setItems([])
      setSelectedProjectId(null)
      setSearchQuery("")
      setDebouncedSearchQuery("")
      setCurrentPage(1)
      setTotalPages(1)
      setTotalItems(0)
      setSelectedItem(null)
      setSortKey(null)
      setSortDirection(null)
      setError(null)
      setIsLoading(true)
      setSourceTypeFilter("all")
      
      // Обновляем ref
      currentClientRef.current = clientIdStr
    } else if (currentClientRef.current === null) {
      // Первая загрузка
      currentClientRef.current = clientIdStr
    }
  }, [clientId])

  // Загрузка данных при изменении проекта
  useEffect(() => {
    if (selectedProjectId && clientId) {
      setCurrentPage(1)
      fetchNomenclature()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProjectId, clientId])

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

  const fetchNomenclature = useCallback(async () => {
    // Валидация параметров перед запросом
    if (!clientId) {
      console.warn('fetchNomenclature: clientId is required')
      setIsLoading(false)
      return
    }

    // Валидация selectedProjectId - если выбран проект, он должен быть валидным числом
    if (selectedProjectId !== null && (isNaN(Number(selectedProjectId)) || selectedProjectId <= 0)) {
      console.warn('fetchNomenclature: selectedProjectId must be a valid positive number')
      setIsLoading(false)
      setError('Неверный идентификатор проекта')
      return
    }

    // Проверяем, что клиент не изменился
    const clientIdStr = String(clientId)
    if (currentClientRef.current !== clientIdStr) {
      // Клиент изменился, не выполняем запрос
      return
    }

    // Отменяем предыдущий запрос
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    // Создаем новый AbortController
    const controller = new AbortController()
    abortControllerRef.current = controller

    setIsLoading(true)
    setError(null)
    try {
      let url = `/api/clients/${clientId}/nomenclature?page=${currentPage}&limit=${itemsPerPage}`
      if (selectedProjectId) {
        url = `/api/clients/${clientId}/projects/${selectedProjectId}/nomenclature?page=${currentPage}&limit=${itemsPerPage}`
      }
      if (debouncedSearchQuery) {
        url += `&search=${encodeURIComponent(debouncedSearchQuery)}`
      }
      
      const response = await fetch(url, { signal: controller.signal })
      if (!response.ok) {
        // Проверяем, что клиент не изменился
        if (currentClientRef.current !== clientIdStr) {
          return
        }

        // Если endpoint не существует, пробуем альтернативный способ
        if (response.status === 404) {
          // Если выбран проект, но endpoint не найден - это нормально, просто пустой список
          if (selectedProjectId) {
            console.info(`Project ${selectedProjectId} not found or has no nomenclature data`)
            setItems([])
            setTotalItems(0)
            setTotalPages(1)
            return
          }
          
          // Используем normalized/uploads как fallback только если проект не выбран
          const uploadsResponse = await fetch(`/api/normalized/uploads`, { signal: controller.signal })
          if (uploadsResponse.ok) {
            const uploadsData = await uploadsResponse.json()
            // Преобразуем данные выгрузок в формат номенклатуры
            const transformedItems: NomenclatureItem[] = []
            if (uploadsData.uploads && Array.isArray(uploadsData.uploads)) {
              uploadsData.uploads.forEach((upload: any, index: number) => {
                transformedItems.push({
                  id: index + 1,
                  code: upload.UploadUUID?.substring(0, 8) || `UP${index}`,
                  name: upload.ConfigName || `Выгрузка ${index + 1}`,
                  normalized_name: upload.ConfigName || `Выгрузка ${index + 1}`,
                  category: 'Выгрузка',
                  quality_score: 0.8,
                  status: upload.Status || 'completed',
                  merged_count: upload.TotalItems || 0,
                })
              })
            }
            setItems(transformedItems)
            setTotalItems(transformedItems.length)
            setTotalPages(Math.ceil(transformedItems.length / itemsPerPage))
            return
          }
        }
        
        // Для других ошибок выбрасываем исключение
        const errorText = await response.text().catch(() => '')
        throw new Error(`Failed to fetch nomenclature: ${response.status} ${response.statusText}${errorText ? ` - ${errorText}` : ''}`)
      }
      const data = await response.json()
      
      // Обработка разных форматов ответа
      const itemsList = data.items || data.nomenclature || data.data || (Array.isArray(data) ? data : [])
      const total = data.total || data.count || itemsList.length
      
      // Проверяем, что клиент не изменился во время запроса
      if (currentClientRef.current !== clientIdStr) {
        return // Клиент изменился, не обновляем состояние
      }

      setItems(itemsList)
      setTotalItems(total)
      setTotalPages(Math.ceil(total / itemsPerPage))
    } catch (error: any) {
      // Игнорируем ошибки отмены запроса
      if (error.name === 'AbortError') {
        return
      }
      
      // Проверяем, что клиент не изменился
      if (currentClientRef.current !== clientIdStr) {
        return
      }
      
      console.error('Failed to fetch nomenclature:', error)
      setError(error instanceof Error ? error.message : 'Не удалось загрузить номенклатуру')
    } finally {
      // Проверяем, что клиент не изменился
      if (currentClientRef.current === clientIdStr) {
        setIsLoading(false)
      }
    }
  }, [clientId, selectedProjectId, currentPage, debouncedSearchQuery, itemsPerPage])

  useEffect(() => {
    fetchNomenclature()
  }, [fetchNomenclature])

  const handleSearch = (value: string) => {
    setSearchQuery(value)
    setCurrentPage(1)
  }

  const getQualityBadgeVariant = (score: number) => {
    if (score >= 0.9) return 'default'
    if (score >= 0.7) return 'secondary'
    return 'destructive'
  }

  const getQualityLabel = (score: number) => {
    if (score >= 0.9) return 'Высокое'
    if (score >= 0.7) return 'Среднее'
    return 'Низкое'
  }

  const handleSort = (key: string) => {
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
  }

  const getSortIcon = (key: string) => {
    if (sortKey !== key) {
      return <ArrowUpDown className="h-4 w-4 ml-1 opacity-50" />
    }
    if (sortDirection === 'asc') {
      return <ArrowUp className="h-4 w-4 ml-1" />
    }
    return <ArrowDown className="h-4 w-4 ml-1" />
  }

  // Фильтрация и сортировка данных
  const filteredAndSortedItems = useMemo(() => {
    // Фильтрация по типу источника
    let filtered = items
    if (sourceTypeFilter !== "all") {
      filtered = items.filter(item => item.source_type === sourceTypeFilter)
    }

    // Сортировка
    if (!sortKey || !sortDirection) return filtered

    return [...filtered].sort((a, b) => {
      let aValue: any
      let bValue: any

      switch (sortKey) {
        case 'code':
          aValue = a.code || ''
          bValue = b.code || ''
          break
        case 'name':
          aValue = a.name || ''
          bValue = b.name || ''
          break
        case 'normalized_name':
          aValue = a.normalized_name || ''
          bValue = b.normalized_name || ''
          break
        case 'category':
          aValue = a.category || ''
          bValue = b.category || ''
          break
        case 'quality_score':
          aValue = a.quality_score || 0
          bValue = b.quality_score || 0
          break
        case 'merged_count':
          aValue = a.merged_count || 0
          bValue = b.merged_count || 0
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
  }, [items, sortKey, sortDirection, sourceTypeFilter])

  // Используем filteredAndSortedItems вместо sortedItems
  const sortedItems = filteredAndSortedItems

  // Пересчитываем пагинацию на основе отфильтрованных данных
  const filteredTotalItems = filteredAndSortedItems.length
  const filteredTotalPages = Math.ceil(filteredTotalItems / itemsPerPage)
  
  // Сбрасываем страницу, если текущая страница больше доступных страниц после фильтрации
  useEffect(() => {
    if (currentPage > filteredTotalPages && filteredTotalPages > 0) {
      setCurrentPage(1)
    }
  }, [filteredTotalPages, currentPage])
  
  // Применяем пагинацию к отфильтрованным данным
  const paginatedItems = useMemo(() => {
    const start = (currentPage - 1) * itemsPerPage
    const end = start + itemsPerPage
    return filteredAndSortedItems.slice(start, end)
  }, [filteredAndSortedItems, currentPage, itemsPerPage])

  // Статистика по источникам
  const sourceStats = useMemo(() => {
    const stats = {
      all: items.length,
      normalized: items.filter(item => item.source_type === 'normalized').length,
      main: items.filter(item => item.source_type === 'main').length,
    }
    return stats
  }, [items])

  if (isLoading && items.length === 0) {
    return <LoadingState message="Загрузка номенклатуры..." />
  }

  // Показываем индикатор загрузки поверх данных при обновлении
  const isRefreshing = isLoading && items.length > 0

  return (
    <div className="space-y-4">
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
              }}
            >
              <SelectTrigger className="w-full md:w-[300px]">
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
              value={sourceTypeFilter}
              onValueChange={(value) => {
                setSourceTypeFilter(value)
                setCurrentPage(1)
              }}
            >
              <SelectTrigger className="w-full md:w-[200px]">
                <SelectValue placeholder="Тип источника" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  Все источники {sourceStats.all > 0 && `(${sourceStats.all})`}
                </SelectItem>
                <SelectItem value="normalized">
                  Нормализованная база {sourceStats.normalized > 0 && `(${sourceStats.normalized})`}
                </SelectItem>
                <SelectItem value="main">
                  Основная база {sourceStats.main > 0 && `(${sourceStats.main})`}
                </SelectItem>
              </SelectContent>
            </Select>
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Поиск по названию или коду..."
                value={searchQuery}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Таблица номенклатуры */}
      {error && items.length === 0 ? (
        <ErrorState
          title="Ошибка загрузки"
          message={error}
          action={{
            label: 'Повторить',
            onClick: fetchNomenclature,
          }}
          variant="destructive"
        />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="h-5 w-5" />
              Номенклатура
              {filteredTotalItems > 0 && (
                <Badge variant="outline">{filteredTotalItems}</Badge>
              )}
              {sourceTypeFilter === "all" && sourceStats.all > 0 && (
                <div className="flex gap-1 ml-2 text-xs text-muted-foreground">
                  <span>(норм: {sourceStats.normalized}, осн: {sourceStats.main})</span>
                </div>
              )}
            </CardTitle>
            <CardDescription>
              Список нормализованной номенклатуры
            </CardDescription>
          </CardHeader>
          <CardContent>
            {items.length === 0 ? (
              <div className="py-8 text-center text-muted-foreground">
                Номенклатура не найдена
              </div>
            ) : (
              <>
                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>
                          <button
                            onClick={() => handleSort('code')}
                            className="flex items-center hover:text-foreground"
                          >
                            Код
                            {getSortIcon('code')}
                          </button>
                        </TableHead>
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
                            onClick={() => handleSort('category')}
                            className="flex items-center hover:text-foreground"
                          >
                            Категория
                            {getSortIcon('category')}
                          </button>
                        </TableHead>
                        <TableHead>КПВЭД</TableHead>
                        <TableHead>
                          <button
                            onClick={() => handleSort('quality_score')}
                            className="flex items-center hover:text-foreground"
                          >
                            Качество
                            {getSortIcon('quality_score')}
                          </button>
                        </TableHead>
                        <TableHead>
                          <button
                            onClick={() => handleSort('merged_count')}
                            className="flex items-center hover:text-foreground"
                          >
                            Объединено
                            {getSortIcon('merged_count')}
                          </button>
                        </TableHead>
                        <TableHead>Источник</TableHead>
                        <TableHead className="text-right">Действия</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {paginatedItems.map((item) => (
                        <TableRow key={item.id}>
                          <TableCell className="font-mono text-sm">{item.code}</TableCell>
                          <TableCell className="max-w-[200px] truncate" title={item.name}>
                            {item.name}
                          </TableCell>
                          <TableCell className="max-w-[200px] truncate" title={item.normalized_name}>
                            {item.normalized_name}
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline">{item.category}</Badge>
                          </TableCell>
                          <TableCell>
                            {item.kpved_code ? (
                              <div className="text-sm">
                                <div className="font-mono">{item.kpved_code}</div>
                                {item.kpved_name && (
                                  <div className="text-xs text-muted-foreground truncate max-w-[150px]" title={item.kpved_name}>
                                    {item.kpved_name}
                                  </div>
                                )}
                              </div>
                            ) : (
                              <span className="text-muted-foreground text-sm">—</span>
                            )}
                          </TableCell>
                          <TableCell>
                            <Badge variant={getQualityBadgeVariant(item.quality_score)}>
                              {getQualityLabel(item.quality_score)} ({Math.round(item.quality_score * 100)}%)
                            </Badge>
                          </TableCell>
                          <TableCell>
                            {item.merged_count > 1 && (
                              <Badge variant="secondary">{item.merged_count}</Badge>
                            )}
                          </TableCell>
                          <TableCell>
                            {item.source_type ? (
                              <div className="flex flex-col gap-1">
                                <Badge variant={item.source_type === 'normalized' ? 'default' : 'outline'} className="text-xs">
                                  {item.source_type === 'normalized' ? 'Нормализованная' : 'Основная'}
                                </Badge>
                                {item.source_database && (
                                  <span className="text-xs text-muted-foreground truncate max-w-[150px]" title={item.source_database}>
                                    {item.source_database.split(/[/\\]/).pop() || item.source_database}
                                  </span>
                                )}
                                {item.project_name && (
                                  <span className="text-xs text-muted-foreground truncate max-w-[150px]" title={item.project_name}>
                                    {item.project_name}
                                  </span>
                                )}
                              </div>
                            ) : (
                              <span className="text-muted-foreground text-sm">—</span>
                            )}
                          </TableCell>
                          <TableCell className="text-right">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => setSelectedItem(item)}
                            >
                              <Eye className="h-4 w-4" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {/* Пагинация */}
                {filteredTotalPages > 1 && (
                  <div className="flex items-center justify-between mt-4">
                    <div className="text-sm text-muted-foreground">
                      Показано {(currentPage - 1) * itemsPerPage + 1} - {Math.min(currentPage * itemsPerPage, filteredTotalItems)} из {filteredTotalItems}
                      {sourceTypeFilter !== "all" && filteredTotalItems !== totalItems && (
                        <span className="ml-2">(всего: {totalItems})</span>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                        disabled={currentPage === 1}
                      >
                        <ChevronLeft className="h-4 w-4" />
                        Назад
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage(p => Math.min(filteredTotalPages, p + 1))}
                        disabled={currentPage === filteredTotalPages}
                      >
                        Вперед
                        <ChevronRight className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      )}

      {selectedItem && (
        <NomenclatureDetailDialog
          item={selectedItem}
          open={!!selectedItem}
          onOpenChange={(open) => !open && setSelectedItem(null)}
        />
      )}
    </div>
  )
}

