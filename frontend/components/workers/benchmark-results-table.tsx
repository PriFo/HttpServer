'use client'

import { useState, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Zap,
  TrendingUp,
  TrendingDown,
  CheckCircle2,
  XCircle,
  AlertCircle,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Search,
  Filter,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  ChevronUp,
  BarChart3,
  Award,
  Clock,
  Target,
  Activity,
  RefreshCw,
  Info
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { BenchmarkErrorBreakdown } from './benchmark-error-breakdown'
import { BenchmarkPercentilesChart } from './benchmark-percentiles-chart'

interface BenchmarkResult {
  // Существующие поля
  model: string
  priority: number
  speed: number
  avg_response_time_ms: number
  median_response_time_ms?: number
  p95_response_time_ms?: number
  min_response_time_ms?: number
  max_response_time_ms?: number
  success_count: number
  error_count: number
  total_requests: number
  success_rate: number
  status: string
  
  // Новые поля для качества и стабильности
  avg_confidence?: number
  min_confidence?: number
  max_confidence?: number
  avg_ai_calls_count?: number
  avg_retries?: number
  coefficient_of_variation?: number
  p75_response_time_ms?: number
  p90_response_time_ms?: number
  p99_response_time_ms?: number
  throughput_items_per_sec?: number
  error_breakdown?: {
    quota_exceeded: number
    rate_limit: number
    timeout: number
    network: number
    auth: number
    other: number
  }
}

interface BenchmarkResultsTableProps {
  results: BenchmarkResult[]
  timestamp?: string
  onClose?: () => void
}

type SortField = 'model' | 'priority' | 'speed' | 'avg_response_time_ms' | 'success_rate' | 'success_count' | 'error_count' | 'avg_confidence' | 'coefficient_of_variation' | 'avg_retries'
type SortOrder = 'asc' | 'desc'

export function BenchmarkResultsTable({ results, timestamp, onClose }: BenchmarkResultsTableProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'success' | 'partial' | 'error'>('all')
  const [sortField, setSortField] = useState<SortField>('speed')
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(25)
  const [minSpeed, setMinSpeed] = useState<number | ''>('')
  const [minSuccessRate, setMinSuccessRate] = useState<number | ''>('')
  const [maxResponseTime, setMaxResponseTime] = useState<number | ''>('')
  const [minConfidence, setMinConfidence] = useState<number | ''>('')
  const [maxCoeffVariation, setMaxCoeffVariation] = useState<number | ''>('')
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false)
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set())

  // Фильтрация и сортировка
  const filteredAndSortedResults = useMemo(() => {
    let filtered = results.filter(result => {
      // Поиск по названию модели
      const matchesSearch = result.model.toLowerCase().includes(searchQuery.toLowerCase())
      
      // Фильтр по статусу
      let matchesStatus = true
      if (statusFilter === 'success') {
        matchesStatus = result.success_rate >= 80 && result.status.toLowerCase().includes('success')
      } else if (statusFilter === 'partial') {
        matchesStatus = result.success_rate >= 50 && result.success_rate < 80
      } else if (statusFilter === 'error') {
        matchesStatus = result.success_rate < 50 || result.status.toLowerCase().includes('error')
      }
      
      // Расширенные фильтры
      const matchesSpeed = minSpeed === '' || (result.speed || 0) >= minSpeed
      const matchesSuccessRate = minSuccessRate === '' || (result.success_rate || 0) >= minSuccessRate
      const matchesResponseTime = maxResponseTime === '' || (result.avg_response_time_ms || 0) <= maxResponseTime
      const matchesConfidence = minConfidence === '' || (result.avg_confidence || 0) >= minConfidence
      const matchesCoeffVariation = maxCoeffVariation === '' || (result.coefficient_of_variation || 999) <= maxCoeffVariation
      
      return matchesSearch && matchesStatus && matchesSpeed && matchesSuccessRate && matchesResponseTime && matchesConfidence && matchesCoeffVariation
    })

    // Сортировка
    filtered.sort((a, b) => {
      let aValue: number | string
      let bValue: number | string

      switch (sortField) {
        case 'model':
          aValue = a.model.toLowerCase()
          bValue = b.model.toLowerCase()
          break
        case 'priority':
          aValue = a.priority
          bValue = b.priority
          break
        case 'speed':
          aValue = a.speed || 0
          bValue = b.speed || 0
          break
        case 'avg_response_time_ms':
          aValue = a.avg_response_time_ms || 0
          bValue = b.avg_response_time_ms || 0
          break
        case 'success_rate':
          aValue = a.success_rate || 0
          bValue = b.success_rate || 0
          break
        case 'success_count':
          aValue = a.success_count || 0
          bValue = b.success_count || 0
          break
        case 'error_count':
          aValue = a.error_count || 0
          bValue = b.error_count || 0
          break
        default:
          return 0
      }

      if (typeof aValue === 'string' && typeof bValue === 'string') {
        return sortOrder === 'asc' 
          ? aValue.localeCompare(bValue)
          : bValue.localeCompare(aValue)
      }

      return sortOrder === 'asc' 
        ? (aValue as number) - (bValue as number)
        : (bValue as number) - (aValue as number)
    })

    return filtered
  }, [results, searchQuery, statusFilter, sortField, sortOrder])

  // Пагинация
  const totalPages = Math.ceil(filteredAndSortedResults.length / pageSize)
  const paginatedResults = useMemo(() => {
    const start = (currentPage - 1) * pageSize
    return filteredAndSortedResults.slice(start, start + pageSize)
  }, [filteredAndSortedResults, currentPage, pageSize])

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortOrder('desc')
    }
  }

  const getStatusBadge = (result: BenchmarkResult) => {
    if (result.success_rate >= 80) {
      return (
        <Badge variant="default" className="bg-green-500 hover:bg-green-600">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          SUCCESS
        </Badge>
      )
    } else if (result.success_rate >= 50) {
      return (
        <Badge variant="secondary" className="bg-yellow-500 hover:bg-yellow-600 text-white">
          <AlertCircle className="h-3 w-3 mr-1" />
          PARTIAL
        </Badge>
      )
    } else {
      return (
        <Badge variant="destructive">
          <XCircle className="h-3 w-3 mr-1" />
          ERROR
        </Badge>
      )
    }
  }

  const getSpeedBadge = (speed: number) => {
    if (speed >= 0.5) {
      return (
        <Badge variant="default" className="bg-green-500 hover:bg-green-600">
          <Zap className="h-3 w-3 mr-1" />
          Быстро
        </Badge>
      )
    } else if (speed >= 0.2) {
      return (
        <Badge variant="secondary" className="bg-yellow-500 hover:bg-yellow-600 text-white">
          <Clock className="h-3 w-3 mr-1" />
          Средне
        </Badge>
      )
    } else {
      return (
        <Badge variant="outline" className="bg-gray-500 hover:bg-gray-600 text-white">
          <TrendingDown className="h-3 w-3 mr-1" />
          Медленно
        </Badge>
      )
    }
  }

  const getConfidenceBadge = (confidence: number | undefined) => {
    if (confidence === undefined || confidence === null) return null
    const confidencePercent = confidence * 100
    let color = 'bg-red-500'
    if (confidencePercent >= 80) color = 'bg-green-500'
    else if (confidencePercent >= 50) color = 'bg-yellow-500'
    
    return (
      <div className="flex items-center gap-2 min-w-[100px]">
        <div className="flex-1 min-w-[60px]">
          <div className="h-2 bg-secondary rounded-full overflow-hidden">
            <div
              className={cn("h-full transition-all", color)}
              style={{ width: `${confidencePercent}%` }}
            />
          </div>
        </div>
        <span className="text-xs font-medium min-w-[45px]">
          {confidencePercent.toFixed(1)}%
        </span>
      </div>
    )
  }

  const getStabilityBadge = (coeffVar: number | undefined) => {
    if (coeffVar === undefined || coeffVar === null) return <span className="text-muted-foreground">-</span>
    
    let color = 'text-red-600 dark:text-red-400'
    let label = 'Нестабильно'
    if (coeffVar < 0.3) {
      color = 'text-green-600 dark:text-green-400'
      label = 'Стабильно'
    } else if (coeffVar < 0.6) {
      color = 'text-yellow-600 dark:text-yellow-400'
      label = 'Средне'
    }
    
    return (
      <div className="flex items-center gap-1">
        <Activity className={cn("h-3 w-3", color)} />
        <span className={cn("text-xs font-medium", color)}>
          {label}
        </span>
        <span className="text-xs text-muted-foreground">
          ({coeffVar.toFixed(2)})
        </span>
      </div>
    )
  }

  const toggleRowExpansion = (modelName: string) => {
    setExpandedRows(prev => {
      const newSet = new Set(prev)
      if (newSet.has(modelName)) {
        newSet.delete(modelName)
      } else {
        newSet.add(modelName)
      }
      return newSet
    })
  }

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) {
      return <ArrowUpDown className="h-3 w-3 ml-1 opacity-50" />
    }
    return sortOrder === 'asc' 
      ? <ArrowUp className="h-3 w-3 ml-1" />
      : <ArrowDown className="h-3 w-3 ml-1" />
  }

  // Статистика
  const stats = useMemo(() => {
    const total = filteredAndSortedResults.length
    const successful = filteredAndSortedResults.filter(r => r.success_rate >= 80).length
    const partial = filteredAndSortedResults.filter(r => r.success_rate >= 50 && r.success_rate < 80).length
    const errors = filteredAndSortedResults.filter(r => r.success_rate < 50).length
    const avgSpeed = filteredAndSortedResults.reduce((sum, r) => sum + (r.speed || 0), 0) / total || 0
    const avgSuccessRate = filteredAndSortedResults.reduce((sum, r) => sum + r.success_rate, 0) / total || 0

    return { total, successful, partial, errors, avgSpeed, avgSuccessRate }
  }, [filteredAndSortedResults])

  return (
    <Card className="w-full">
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Результаты бенчмарка
            <Badge variant="outline" className="ml-2">
              {stats.total} моделей
            </Badge>
          </CardTitle>
          {onClose && (
            <Button variant="ghost" size="sm" onClick={onClose}>
              <XCircle className="h-4 w-4" />
            </Button>
          )}
        </div>
        {timestamp && (
          <p className="text-xs text-muted-foreground mt-2">
            Обновлено: {new Date(timestamp).toLocaleString('ru-RU')}
          </p>
        )}
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Статистика */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="flex items-center gap-2 p-2 border rounded-lg">
            <CheckCircle2 className="h-4 w-4 text-green-500" />
            <div>
              <div className="text-xs text-muted-foreground">Успешных</div>
              <div className="font-semibold">{stats.successful}</div>
            </div>
          </div>
          <div className="flex items-center gap-2 p-2 border rounded-lg">
            <AlertCircle className="h-4 w-4 text-yellow-500" />
            <div>
              <div className="text-xs text-muted-foreground">Частичных</div>
              <div className="font-semibold">{stats.partial}</div>
            </div>
          </div>
          <div className="flex items-center gap-2 p-2 border rounded-lg">
            <XCircle className="h-4 w-4 text-red-500" />
            <div>
              <div className="text-xs text-muted-foreground">Ошибок</div>
              <div className="font-semibold">{stats.errors}</div>
            </div>
          </div>
          <div className="flex items-center gap-2 p-2 border rounded-lg">
            <Zap className="h-4 w-4 text-blue-500" />
            <div>
              <div className="text-xs text-muted-foreground">Средняя скорость</div>
              <div className="font-semibold">{stats.avgSpeed.toFixed(2)} req/s</div>
            </div>
          </div>
        </div>

        {/* Фильтры и поиск */}
        <div className="space-y-2">
          <div className="flex flex-col sm:flex-row gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Поиск по названию модели..."
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value)
                  setCurrentPage(1)
                }}
                className="pl-8"
              />
            </div>
            <Select
              value={statusFilter}
              onValueChange={(value: 'all' | 'success' | 'partial' | 'error') => {
                setStatusFilter(value)
                setCurrentPage(1)
              }}
            >
              <SelectTrigger className="w-full sm:w-[180px]">
                <Filter className="h-4 w-4 mr-2" />
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все статусы</SelectItem>
                <SelectItem value="success">Успешные (≥80%)</SelectItem>
                <SelectItem value="partial">Частичные (50-79%)</SelectItem>
                <SelectItem value="error">Ошибки (&lt;50%)</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={pageSize.toString()}
              onValueChange={(value) => {
                setPageSize(parseInt(value))
                setCurrentPage(1)
              }}
            >
              <SelectTrigger className="w-full sm:w-[120px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10">10 на странице</SelectItem>
                <SelectItem value="25">25 на странице</SelectItem>
                <SelectItem value="50">50 на странице</SelectItem>
                <SelectItem value="100">100 на странице</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAdvancedFilters(!showAdvancedFilters)}
              className="w-full sm:w-auto"
            >
              <Filter className="h-4 w-4 mr-2" />
              {showAdvancedFilters ? 'Скрыть' : 'Расширенные'} фильтры
            </Button>
          </div>
          
          {/* Расширенные фильтры */}
          {showAdvancedFilters && (
            <div className="p-4 border rounded-lg bg-muted/50 space-y-3">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Мин. скорость (req/s)</Label>
                  <Input
                    type="number"
                    step="0.1"
                    placeholder="0.0"
                    value={minSpeed}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMinSpeed(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Мин. успешность (%)</Label>
                  <Input
                    type="number"
                    step="1"
                    min="0"
                    max="100"
                    placeholder="0"
                    value={minSuccessRate}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMinSuccessRate(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Макс. время отклика (мс)</Label>
                  <Input
                    type="number"
                    step="100"
                    placeholder="10000"
                    value={maxResponseTime}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMaxResponseTime(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Мин. скорость (req/s)</Label>
                  <Input
                    type="number"
                    step="0.1"
                    placeholder="0.0"
                    value={minSpeed}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMinSpeed(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Мин. успешность (%)</Label>
                  <Input
                    type="number"
                    step="1"
                    min="0"
                    max="100"
                    placeholder="0"
                    value={minSuccessRate}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMinSuccessRate(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Макс. время отклика (мс)</Label>
                  <Input
                    type="number"
                    step="100"
                    placeholder="10000"
                    value={maxResponseTime}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMaxResponseTime(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Мин. уверенность (%)</Label>
                  <Input
                    type="number"
                    step="1"
                    min="0"
                    max="100"
                    placeholder="0"
                    value={minConfidence}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMinConfidence(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Макс. коэф. вариации</Label>
                  <Input
                    type="number"
                    step="0.1"
                    placeholder="1.0"
                    value={maxCoeffVariation}
                    onChange={(e) => {
                      const val = e.target.value === '' ? '' : parseFloat(e.target.value)
                      setMaxCoeffVariation(val as number | '')
                      setCurrentPage(1)
                    }}
                    className="h-8"
                  />
                </div>
              </div>
              {(minSpeed !== '' || minSuccessRate !== '' || maxResponseTime !== '' || minConfidence !== '' || maxCoeffVariation !== '') && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setMinSpeed('')
                    setMinSuccessRate('')
                    setMaxResponseTime('')
                    setMinConfidence('')
                    setMaxCoeffVariation('')
                    setCurrentPage(1)
                  }}
                  className="text-xs"
                >
                  Сбросить фильтры
                </Button>
              )}
            </div>
          )}
        </div>

        {/* Таблица */}
        <div className="border rounded-lg overflow-hidden">
          <div className="overflow-x-auto max-h-[600px] overflow-y-auto">
            <Table>
              <TableHeader className="sticky top-0 bg-background z-10 border-b">
                <TableRow>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('model')}
                  >
                    <div className="flex items-center">
                      Модель
                      <SortIcon field="model" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('priority')}
                  >
                    <div className="flex items-center">
                      Приоритет
                      <SortIcon field="priority" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('speed')}
                  >
                    <div className="flex items-center">
                      Скорость
                      <SortIcon field="speed" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('avg_response_time_ms')}
                  >
                    <div className="flex items-center">
                      Среднее время
                      <SortIcon field="avg_response_time_ms" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('success_count')}
                  >
                    <div className="flex items-center">
                      Успешно
                      <SortIcon field="success_count" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('error_count')}
                  >
                    <div className="flex items-center">
                      Ошибок
                      <SortIcon field="error_count" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('success_rate')}
                  >
                    <div className="flex items-center">
                      Успешность
                      <SortIcon field="success_rate" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('avg_confidence')}
                  >
                    <div className="flex items-center">
                      Уверенность
                      <SortIcon field="avg_confidence" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('coefficient_of_variation')}
                  >
                    <div className="flex items-center">
                      Стабильность
                      <SortIcon field="coefficient_of_variation" />
                    </div>
                  </TableHead>
                  <TableHead 
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleSort('avg_retries')}
                  >
                    <div className="flex items-center">
                      Retry
                      <SortIcon field="avg_retries" />
                    </div>
                  </TableHead>
                  <TableHead>Статус</TableHead>
                  <TableHead className="w-[50px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginatedResults.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={12} className="text-center py-8 text-muted-foreground">
                      Нет результатов, соответствующих фильтрам
                    </TableCell>
                  </TableRow>
                ) : (
                  paginatedResults.map((result, idx) => {
                    const isExpanded = expandedRows.has(result.model)
                    return (
                      <>
                        <TableRow key={`${result.model}-${idx}`} className="hover:bg-muted/50">
                          <TableCell className="font-medium">
                            <div className="flex items-center gap-2">
                              <span className="truncate max-w-[200px]" title={result.model}>
                                {result.model}
                              </span>
                              {getSpeedBadge(result.speed || 0)}
                            </div>
                          </TableCell>
                          <TableCell>{result.priority || '-'}</TableCell>
                          <TableCell>
                            <div className="flex items-center gap-1">
                              <Zap className="h-3 w-3 text-yellow-500" />
                              {result.speed ? result.speed.toFixed(2) : '-'} req/s
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="space-y-1">
                              {result.avg_response_time_ms 
                                ? `${(result.avg_response_time_ms / 1000).toFixed(2)}s`
                                : '-'}
                              {result.p95_response_time_ms && (
                                <div className="text-xs text-muted-foreground" title="P95 перцентиль">
                                  P95: {(result.p95_response_time_ms / 1000).toFixed(2)}s
                                </div>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-green-600 dark:text-green-400 font-medium">
                            {result.success_count || 0}
                          </TableCell>
                          <TableCell className="text-red-600 dark:text-red-400 font-medium">
                            {result.error_count || 0}
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <div className="flex-1 min-w-[60px]">
                                <div className="h-2 bg-secondary rounded-full overflow-hidden">
                                  <div
                                    className={cn(
                                      "h-full transition-all",
                                      result.success_rate >= 80 ? "bg-green-500" :
                                      result.success_rate >= 50 ? "bg-yellow-500" : "bg-red-500"
                                    )}
                                    style={{ width: `${result.success_rate}%` }}
                                  />
                                </div>
                              </div>
                              <span className="text-xs font-medium min-w-[45px]">
                                {result.success_rate ? `${result.success_rate.toFixed(1)}%` : '-'}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell>
                            {getConfidenceBadge(result.avg_confidence)}
                          </TableCell>
                          <TableCell>
                            {getStabilityBadge(result.coefficient_of_variation)}
                          </TableCell>
                          <TableCell>
                            {result.avg_retries !== undefined && result.avg_retries !== null ? (
                              <div className="flex items-center gap-1">
                                <RefreshCw className="h-3 w-3 text-muted-foreground" />
                                <span className="text-xs">{result.avg_retries.toFixed(2)}</span>
                              </div>
                            ) : (
                              <span className="text-muted-foreground">-</span>
                            )}
                          </TableCell>
                          <TableCell>
                            {getStatusBadge(result)}
                          </TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => toggleRowExpansion(result.model)}
                              className="h-8 w-8 p-0"
                            >
                              {isExpanded ? (
                                <ChevronUp className="h-4 w-4" />
                              ) : (
                                <ChevronDown className="h-4 w-4" />
                              )}
                            </Button>
                          </TableCell>
                        </TableRow>
                        {isExpanded && (
                          <TableRow key={`${result.model}-details-${idx}`}>
                            <TableCell colSpan={12} className="bg-muted/30 p-4">
                              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                {/* Метрики качества */}
                                <div className="space-y-2">
                                  <h4 className="font-semibold text-sm flex items-center gap-2">
                                    <Target className="h-4 w-4" />
                                    Качество классификации
                                  </h4>
                                  <div className="space-y-1 text-xs">
                                    <div className="flex justify-between">
                                      <span className="text-muted-foreground">Средняя уверенность:</span>
                                      <span className="font-medium">
                                        {result.avg_confidence !== undefined 
                                          ? `${(result.avg_confidence * 100).toFixed(1)}%` 
                                          : '-'}
                                      </span>
                                    </div>
                                    {result.min_confidence !== undefined && result.max_confidence !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">Диапазон:</span>
                                        <span className="font-medium">
                                          {(result.min_confidence * 100).toFixed(1)}% - {(result.max_confidence * 100).toFixed(1)}%
                                        </span>
                                      </div>
                                    )}
                                    <div className="flex justify-between">
                                      <span className="text-muted-foreground">AI вызовов/запрос:</span>
                                      <span className="font-medium">
                                        {result.avg_ai_calls_count !== undefined 
                                          ? result.avg_ai_calls_count.toFixed(2) 
                                          : '-'}
                                      </span>
                                    </div>
                                  </div>
                                </div>

                                {/* Перцентили времени ответа */}
                                <div className="space-y-2">
                                  <h4 className="font-semibold text-sm flex items-center gap-2">
                                    <Clock className="h-4 w-4" />
                                    Перцентили времени ответа
                                  </h4>
                                  <div className="space-y-1 text-xs">
                                    {result.median_response_time_ms !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">P50 (медиана):</span>
                                        <span className="font-medium">{(result.median_response_time_ms / 1000).toFixed(2)}s</span>
                                      </div>
                                    )}
                                    {result.p75_response_time_ms !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">P75:</span>
                                        <span className="font-medium">{(result.p75_response_time_ms / 1000).toFixed(2)}s</span>
                                      </div>
                                    )}
                                    {result.p90_response_time_ms !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">P90:</span>
                                        <span className="font-medium">{(result.p90_response_time_ms / 1000).toFixed(2)}s</span>
                                      </div>
                                    )}
                                    {result.p95_response_time_ms !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">P95:</span>
                                        <span className="font-medium">{(result.p95_response_time_ms / 1000).toFixed(2)}s</span>
                                      </div>
                                    )}
                                    {result.p99_response_time_ms !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">P99:</span>
                                        <span className="font-medium">{(result.p99_response_time_ms / 1000).toFixed(2)}s</span>
                                      </div>
                                    )}
                                  </div>
                                </div>

                                {/* Детальная статистика ошибок */}
                                {result.error_breakdown && result.error_count > 0 && (
                                  <div className="space-y-2">
                                    <h4 className="font-semibold text-sm flex items-center gap-2">
                                      <AlertCircle className="h-4 w-4" />
                                      Типы ошибок
                                    </h4>
                                    <div className="space-y-1 text-xs">
                                      {result.error_breakdown.quota_exceeded > 0 && (
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Quota exceeded:</span>
                                          <span className="font-medium text-red-600">{result.error_breakdown.quota_exceeded}</span>
                                        </div>
                                      )}
                                      {result.error_breakdown.rate_limit > 0 && (
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Rate limit:</span>
                                          <span className="font-medium text-orange-600">{result.error_breakdown.rate_limit}</span>
                                        </div>
                                      )}
                                      {result.error_breakdown.timeout > 0 && (
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Timeout:</span>
                                          <span className="font-medium text-yellow-600">{result.error_breakdown.timeout}</span>
                                        </div>
                                      )}
                                      {result.error_breakdown.network > 0 && (
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Network:</span>
                                          <span className="font-medium text-blue-600">{result.error_breakdown.network}</span>
                                        </div>
                                      )}
                                      {result.error_breakdown.auth > 0 && (
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Auth:</span>
                                          <span className="font-medium text-purple-600">{result.error_breakdown.auth}</span>
                                        </div>
                                      )}
                                      {result.error_breakdown.other > 0 && (
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Other:</span>
                                          <span className="font-medium text-gray-600">{result.error_breakdown.other}</span>
                                        </div>
                                      )}
                                    </div>
                                  </div>
                                )}

                                {/* Дополнительные метрики */}
                                <div className="space-y-2">
                                  <h4 className="font-semibold text-sm flex items-center gap-2">
                                    <Info className="h-4 w-4" />
                                    Дополнительно
                                  </h4>
                                  <div className="space-y-1 text-xs">
                                    {result.throughput_items_per_sec !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">Throughput:</span>
                                        <span className="font-medium">{result.throughput_items_per_sec.toFixed(2)} items/s</span>
                                      </div>
                                    )}
                                    {result.coefficient_of_variation !== undefined && (
                                      <div className="flex justify-between">
                                        <span className="text-muted-foreground">Коэф. вариации:</span>
                                        <span className="font-medium">{result.coefficient_of_variation.toFixed(3)}</span>
                                      </div>
                                    )}
                                    {result.min_response_time_ms !== undefined && result.max_response_time_ms !== undefined && (
                                      <>
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Мин. время:</span>
                                          <span className="font-medium">{(result.min_response_time_ms / 1000).toFixed(2)}s</span>
                                        </div>
                                        <div className="flex justify-between">
                                          <span className="text-muted-foreground">Макс. время:</span>
                                          <span className="font-medium">{(result.max_response_time_ms / 1000).toFixed(2)}s</span>
                                        </div>
                                      </>
                                    )}
                                  </div>
                                </div>
                              </div>
                              
                              {/* Визуализации */}
                              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mt-4">
                                {/* График перцентилей */}
                                {(result.median_response_time_ms !== undefined || result.p75_response_time_ms !== undefined || 
                                  result.p90_response_time_ms !== undefined || result.p95_response_time_ms !== undefined || 
                                  result.p99_response_time_ms !== undefined) && (
                                  <BenchmarkPercentilesChart
                                    percentiles={{
                                      p50: result.median_response_time_ms,
                                      p75: result.p75_response_time_ms,
                                      p90: result.p90_response_time_ms,
                                      p95: result.p95_response_time_ms,
                                      p99: result.p99_response_time_ms,
                                      min: result.min_response_time_ms,
                                      max: result.max_response_time_ms,
                                      avg: result.avg_response_time_ms,
                                    }}
                                    modelName={result.model}
                                  />
                                )}
                                
                                {/* Детальная статистика ошибок */}
                                {result.error_breakdown && result.error_count > 0 && (
                                  <BenchmarkErrorBreakdown
                                    errorBreakdown={result.error_breakdown}
                                    totalErrors={result.error_count}
                                    modelName={result.model}
                                  />
                                )}
                              </div>
                            </TableCell>
                          </TableRow>
                        )}
                      </>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </div>

        {/* Пагинация */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between">
            <div className="text-sm text-muted-foreground">
              Показано {((currentPage - 1) * pageSize) + 1} - {Math.min(currentPage * pageSize, filteredAndSortedResults.length)} из {filteredAndSortedResults.length}
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                disabled={currentPage === 1}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <div className="text-sm">
                Страница {currentPage} из {totalPages}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                disabled={currentPage === totalPages}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

