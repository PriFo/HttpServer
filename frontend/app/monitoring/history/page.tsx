'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { MetricsChart } from '@/components/monitoring/MetricsChart'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { BarChart3, ArrowLeft, Activity, Download, FileSpreadsheet, FileCode, FileJson, TrendingUp, TrendingDown, Minus, RefreshCw } from 'lucide-react'
import Link from 'next/link'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { motion } from 'framer-motion'
import { FadeIn } from '@/components/animations/fade-in'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { toast } from 'sonner'
import { useMemo } from 'react'

interface MetricDataPoint {
  timestamp: string
  value: number
}

interface HistoryData {
  ai_success_rate: MetricDataPoint[]
  cache_hit_rate: MetricDataPoint[]
  throughput: MetricDataPoint[]
  uptime: MetricDataPoint[]
  batch_queue_size: MetricDataPoint[]
  circuit_breaker_failures: MetricDataPoint[]
}

type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d'

export default function MonitoringHistoryPage() {
  const router = useRouter()
  const [historyData, setHistoryData] = useState<HistoryData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [timeRange, setTimeRange] = useState<TimeRange>('24h')

  const fetchHistory = async (range: TimeRange) => {
    setLoading(true)
    try {
      // Calculate time range
      const now = new Date()
      const from = new Date()

      switch (range) {
        case '1h':
          from.setHours(now.getHours() - 1)
          break
        case '6h':
          from.setHours(now.getHours() - 6)
          break
        case '24h':
          from.setHours(now.getHours() - 24)
          break
        case '7d':
          from.setDate(now.getDate() - 7)
          break
        case '30d':
          from.setDate(now.getDate() - 30)
          break
      }

      const params = new URLSearchParams({
        from: from.toISOString(),
        to: now.toISOString(),
        limit: '1000',
      })

      const response = await fetch(`/api/monitoring/history?${params}`)
      if (response.ok) {
        const responseData = await response.json()
        
        // Backend returns { count, snapshots }, so we need to extract snapshots
        const data = responseData.snapshots || responseData || []

        // Transform data into chart format
        const transformed: HistoryData = {
          ai_success_rate: [],
          cache_hit_rate: [],
          throughput: [],
          uptime: [],
          batch_queue_size: [],
          circuit_breaker_failures: [],
        }

        // Ensure data is an array
        const snapshots = Array.isArray(data) ? data : []
        snapshots.forEach((item: any) => {
          const timestamp = item.timestamp

          if (item.ai_success_rate !== null && item.ai_success_rate !== undefined) {
            transformed.ai_success_rate.push({
              timestamp,
              value: item.ai_success_rate * 100, // Convert to percentage
            })
          }

          if (item.cache_hit_rate !== null && item.cache_hit_rate !== undefined) {
            transformed.cache_hit_rate.push({
              timestamp,
              value: item.cache_hit_rate * 100,
            })
          }

          if (item.throughput !== null && item.throughput !== undefined) {
            transformed.throughput.push({
              timestamp,
              value: item.throughput,
            })
          }

          if (item.uptime_seconds !== null && item.uptime_seconds !== undefined) {
            transformed.uptime.push({
              timestamp,
              value: item.uptime_seconds / 3600, // Convert to hours
            })
          }

          if (item.batch_queue_size !== null && item.batch_queue_size !== undefined) {
            transformed.batch_queue_size.push({
              timestamp,
              value: item.batch_queue_size,
            })
          }

          // Parse metric_data JSON if available
          if (item.metric_data) {
            try {
              const metricData = JSON.parse(item.metric_data)
              if (metricData.circuit_breaker?.failure_count !== undefined) {
                transformed.circuit_breaker_failures.push({
                  timestamp,
                  value: metricData.circuit_breaker.failure_count,
                })
              }
            } catch (e) {
              // Ignore JSON parse errors
            }
          }
        })

        setHistoryData(transformed)
        setError(null)
      } else {
        setError('Failed to fetch history')
      }
    } catch (err) {
      console.error('Error fetching history:', err)
      setError('Failed to connect to backend')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchHistory(timeRange)
  }, [timeRange])

  const getTimeRangeLabel = (range: TimeRange) => {
    switch (range) {
      case '1h':
        return 'Последний час'
      case '6h':
        return 'Последние 6 часов'
      case '24h':
        return 'Последние 24 часа'
      case '7d':
        return 'Последние 7 дней'
      case '30d':
        return 'Последние 30 дней'
    }
  }

  // Calculate statistics for metrics
  const calculateStats = (data: MetricDataPoint[]) => {
    if (data.length === 0) return { min: 0, max: 0, avg: 0, trend: 'stable' as const }
    
    const values = data.map(d => d.value)
    const min = Math.min(...values)
    const max = Math.max(...values)
    const avg = values.reduce((a, b) => a + b, 0) / values.length
    
    // Calculate trend (comparing first half vs second half)
    const mid = Math.floor(values.length / 2)
    const firstHalf = values.slice(0, mid).reduce((a, b) => a + b, 0) / mid
    const secondHalf = values.slice(mid).reduce((a, b) => a + b, 0) / (values.length - mid)
    const trend = secondHalf > firstHalf * 1.05 ? 'up' : secondHalf < firstHalf * 0.95 ? 'down' : 'stable'
    
    return { min, max, avg, trend }
  }

  const stats = useMemo(() => {
    if (!historyData) return null
    return {
      ai_success_rate: calculateStats(historyData.ai_success_rate),
      cache_hit_rate: calculateStats(historyData.cache_hit_rate),
      throughput: calculateStats(historyData.throughput),
      uptime: calculateStats(historyData.uptime),
    }
  }, [historyData])

  // Export functions
  const exportToCSV = () => {
    if (!historyData) return
    
    const headers = ['Timestamp', 'AI Success Rate (%)', 'Cache Hit Rate (%)', 'Throughput (r/s)', 'Uptime (h)']
    const rows = []
    
    // Get max length to align all arrays
    const maxLength = Math.max(
      historyData.ai_success_rate.length,
      historyData.cache_hit_rate.length,
      historyData.throughput.length,
      historyData.uptime.length
    )
    
    for (let i = 0; i < maxLength; i++) {
      const timestamp = historyData.ai_success_rate[i]?.timestamp || 
                       historyData.cache_hit_rate[i]?.timestamp || 
                       historyData.throughput[i]?.timestamp || 
                       historyData.uptime[i]?.timestamp || ''
      const ai = historyData.ai_success_rate[i]?.value.toFixed(2) || ''
      const cache = historyData.cache_hit_rate[i]?.value.toFixed(2) || ''
      const throughput = historyData.throughput[i]?.value.toFixed(2) || ''
      const uptime = historyData.uptime[i]?.value.toFixed(2) || ''
      
      rows.push([timestamp, ai, cache, throughput, uptime].join(','))
    }
    
    const csv = [headers.join(','), ...rows].join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `monitoring-history-${timeRange}-${new Date().toISOString().split('T')[0]}.csv`
    link.click()
    
    toast.success('Данные экспортированы в CSV')
  }

  const exportToJSON = () => {
    if (!historyData) return
    
    const json = JSON.stringify(historyData, null, 2)
    const blob = new Blob([json], { type: 'application/json' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `monitoring-history-${timeRange}-${new Date().toISOString().split('T')[0]}.json`
    link.click()
    
    toast.success('Данные экспортированы в JSON')
  }

  if (loading) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <LoadingState message="Загрузка исторических данных..." size="lg" fullScreen />
      </div>
    )
  }

  if (error || !historyData) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <ErrorState
          title="Ошибка загрузки истории"
          message={error || 'Исторические данные недоступны'}
          action={{
            label: 'Повторить',
            onClick: () => fetchHistory(timeRange),
          }}
          variant="destructive"
        />
      </div>
    )
  }

  const breadcrumbItems = [
    { label: 'Мониторинг', href: '/monitoring', icon: Activity },
    { label: 'История', href: '/monitoring/history', icon: BarChart3 },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="flex items-center justify-between"
        >
          <div className="space-y-1">
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <BarChart3 className="h-8 w-8 text-primary" />
              Историческая аналитика
            </h1>
            <p className="text-muted-foreground">
              Графики метрик производительности за выбранный период
            </p>
          </div>

        <div className="flex items-center gap-4 flex-wrap">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Период:</span>
            <Select value={timeRange} onValueChange={(value) => setTimeRange(value as TimeRange)}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Выберите период" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1h">Последний час</SelectItem>
                <SelectItem value="6h">Последние 6 часов</SelectItem>
                <SelectItem value="24h">Последние 24 часа</SelectItem>
                <SelectItem value="7d">Последние 7 дней</SelectItem>
                <SelectItem value="30d">Последние 30 дней</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Badge variant="outline">
            {getTimeRangeLabel(timeRange)}
          </Badge>
          <Button
            variant="outline"
            size="sm"
            onClick={() => fetchHistory(timeRange)}
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Обновить
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <Download className="h-4 w-4 mr-2" />
                Экспорт
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Экспорт данных</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={exportToCSV}>
                <FileSpreadsheet className="h-4 w-4 mr-2" />
                CSV
              </DropdownMenuItem>
              <DropdownMenuItem onClick={exportToJSON}>
                <FileJson className="h-4 w-4 mr-2" />
                JSON
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </motion.div>
      </FadeIn>

      {/* Statistics Summary */}
      {stats && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
        >
          <Card>
            <CardHeader>
              <CardTitle>Статистика за период</CardTitle>
              <CardDescription>Сводка по ключевым метрикам</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                {stats.ai_success_rate && (
                  <div className="space-y-2 p-4 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-muted-foreground">AI Success Rate</span>
                      {stats.ai_success_rate.trend === 'up' && <TrendingUp className="h-4 w-4 text-green-500" />}
                      {stats.ai_success_rate.trend === 'down' && <TrendingDown className="h-4 w-4 text-red-500" />}
                      {stats.ai_success_rate.trend === 'stable' && <Minus className="h-4 w-4 text-gray-500" />}
                    </div>
                    <div className="space-y-1">
                      <div className="flex justify-between text-sm">
                        <span>Среднее:</span>
                        <span className="font-bold">{stats.ai_success_rate.avg.toFixed(2)}%</span>
                      </div>
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span>Мин: {stats.ai_success_rate.min.toFixed(2)}%</span>
                        <span>Макс: {stats.ai_success_rate.max.toFixed(2)}%</span>
                      </div>
                    </div>
                  </div>
                )}
                {stats.cache_hit_rate && (
                  <div className="space-y-2 p-4 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-muted-foreground">Cache Hit Rate</span>
                      {stats.cache_hit_rate.trend === 'up' && <TrendingUp className="h-4 w-4 text-green-500" />}
                      {stats.cache_hit_rate.trend === 'down' && <TrendingDown className="h-4 w-4 text-red-500" />}
                      {stats.cache_hit_rate.trend === 'stable' && <Minus className="h-4 w-4 text-gray-500" />}
                    </div>
                    <div className="space-y-1">
                      <div className="flex justify-between text-sm">
                        <span>Среднее:</span>
                        <span className="font-bold">{stats.cache_hit_rate.avg.toFixed(2)}%</span>
                      </div>
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span>Мин: {stats.cache_hit_rate.min.toFixed(2)}%</span>
                        <span>Макс: {stats.cache_hit_rate.max.toFixed(2)}%</span>
                      </div>
                    </div>
                  </div>
                )}
                {stats.throughput && (
                  <div className="space-y-2 p-4 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-muted-foreground">Throughput</span>
                      {stats.throughput.trend === 'up' && <TrendingUp className="h-4 w-4 text-green-500" />}
                      {stats.throughput.trend === 'down' && <TrendingDown className="h-4 w-4 text-red-500" />}
                      {stats.throughput.trend === 'stable' && <Minus className="h-4 w-4 text-gray-500" />}
                    </div>
                    <div className="space-y-1">
                      <div className="flex justify-between text-sm">
                        <span>Среднее:</span>
                        <span className="font-bold">{stats.throughput.avg.toFixed(2)} r/s</span>
                      </div>
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span>Мин: {stats.throughput.min.toFixed(2)}</span>
                        <span>Макс: {stats.throughput.max.toFixed(2)}</span>
                      </div>
                    </div>
                  </div>
                )}
                {stats.uptime && (
                  <div className="space-y-2 p-4 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-muted-foreground">Uptime</span>
                      {stats.uptime.trend === 'up' && <TrendingUp className="h-4 w-4 text-green-500" />}
                      {stats.uptime.trend === 'down' && <TrendingDown className="h-4 w-4 text-red-500" />}
                      {stats.uptime.trend === 'stable' && <Minus className="h-4 w-4 text-gray-500" />}
                    </div>
                    <div className="space-y-1">
                      <div className="flex justify-between text-sm">
                        <span>Среднее:</span>
                        <span className="font-bold">{stats.uptime.avg.toFixed(2)} ч</span>
                      </div>
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span>Мин: {stats.uptime.min.toFixed(2)}</span>
                        <span>Макс: {stats.uptime.max.toFixed(2)}</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Performance Metrics */}
      <div className="space-y-4">
        <div>
          <h2 className="text-2xl font-bold">Производительность системы</h2>
          <p className="text-muted-foreground">Ключевые метрики эффективности</p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <MetricsChart
            title="AI Success Rate"
            description="Процент успешных AI запросов"
            data={historyData.ai_success_rate}
            unit="%"
            color="green"
          />

          <MetricsChart
            title="Cache Hit Rate"
            description="Процент попаданий в кеш"
            data={historyData.cache_hit_rate}
            unit="%"
            color="blue"
          />

          <MetricsChart
            title="Throughput"
            description="Пропускная способность (записей/сек)"
            data={historyData.throughput}
            unit=" r/s"
            color="purple"
          />

          <MetricsChart
            title="Uptime"
            description="Время работы системы"
            data={historyData.uptime}
            unit=" ч"
            color="blue"
          />
        </div>
      </div>

      {/* Optimization Components */}
      {(historyData.batch_queue_size.length > 0 || historyData.circuit_breaker_failures.length > 0) && (
        <div className="space-y-4">
          <div>
            <h2 className="text-2xl font-bold">Компоненты оптимизации</h2>
            <p className="text-muted-foreground">Метрики продвинутых механизмов</p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {historyData.batch_queue_size.length > 0 && (
              <MetricsChart
                title="Batch Processor Queue"
                description="Размер очереди батчевой обработки"
                data={historyData.batch_queue_size}
                unit=" items"
                color="yellow"
              />
            )}

            {historyData.circuit_breaker_failures.length > 0 && (
              <MetricsChart
                title="Circuit Breaker Failures"
                description="Счетчик ошибок Circuit Breaker"
                data={historyData.circuit_breaker_failures}
                unit=""
                color="red"
              />
            )}
          </div>
        </div>
      )}

      {/* Info Card */}
      <Card>
        <CardHeader>
          <CardTitle>О метриках</CardTitle>
          <CardDescription>Информация о сборе и хранении данных</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>• Метрики собираются автоматически каждые 60 секунд</p>
          <p>• Данные хранятся в течение 7 дней (настраивается)</p>
          <p>• Графики показывают тренды и помогают выявить проблемы</p>
          <p>• Для детального анализа используйте меньший временной диапазон</p>
        </CardContent>
      </Card>
    </div>
  )
}
