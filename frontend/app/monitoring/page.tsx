'use client'

import { useState, useEffect, useMemo, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { 
  Activity, 
  Server, 
  Zap, 
  AlertCircle, 
  CheckCircle2,
  Clock,
  TrendingUp,
  Power,
  PowerOff,
  Download,
  ShieldCheck,
  ShieldAlert,
  AlertTriangle,
  RefreshCw,
} from 'lucide-react'
import dynamic from 'next/dynamic'
import { Skeleton } from '@/components/ui/skeleton'

// Динамическая загрузка компонентов Recharts
const RechartsBarChart = dynamic(
  () => import('recharts').then((mod) => mod.BarChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
)
const RechartsBar = dynamic(
  () => import('recharts').then((mod) => mod.Bar),
  { ssr: false }
)
const RechartsLineChart = dynamic(
  () => import('recharts').then((mod) => mod.LineChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
)
const RechartsLine = dynamic(
  () => import('recharts').then((mod) => mod.Line),
  { ssr: false }
)
const RechartsPieChart = dynamic(
  () => import('recharts').then((mod) => mod.PieChart),
  { ssr: false, loading: () => <Skeleton className="h-[150px] w-full" /> }
)
const RechartsPie = dynamic(
  () => import('recharts').then((mod) => mod.Pie),
  { ssr: false }
)
const RechartsCell = dynamic(
  () => import('recharts').then((mod) => mod.Cell),
  { ssr: false }
)
// Легкие компоненты можно импортировать напрямую
import { XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { Loader2 } from 'lucide-react'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { FadeIn } from '@/components/animations/fade-in'
import { motion } from 'framer-motion'
import { getBackendUrl } from '@/lib/api-config'
import { useError } from '@/contexts/ErrorContext'
import { Button } from '@/components/ui/button'
import { apiClientJson } from '@/lib/api-client'
import { formatTime } from '@/lib/locale'
import type { MonitoringData } from '@/types/monitoring'
import { toast } from 'sonner'
import { exportMonitoringToCSV, exportMonitoringToJSON, exportProviderMetricsToCSV } from '@/lib/export-monitoring'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { StatCard } from '@/components/common/stat-card'

const STATUS_COLORS = {
  active: 'bg-green-500',
  idle: 'bg-gray-400',
  error: 'bg-red-500',
}

const PROVIDER_COLORS: Record<string, string> = {
  openrouter: '#3b82f6',
  huggingface: '#10b981',
  arliai: '#8b5cf6',
  edenai: '#f59e0b',
  dadata: '#ef4444',
  'adata.kz': '#ec4899',
}

interface RequestHistoryPoint {
  timestamp: number
  requests_per_minute: number
  provider_id: string
}

interface RouteHealthSummary {
  checkedAt: string
  backendUrl: string
  total: number
  healthy: number
  degraded: number
  failed: number
  hasCriticalFailure: boolean
}

interface RouteHealthResult {
  path: string
  label: string
  method?: string
  critical?: boolean
  url: string
  ok: boolean
  status: number | null
  statusText?: string
  durationMs: number
  error?: string
  isTimeout?: boolean
}

export default function MonitoringPage() {
  const { handleError } = useError()
  const [data, setData] = useState<MonitoringData | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [togglingProvider, setTogglingProvider] = useState<string | null>(null)
  const [workersStatus, setWorkersStatus] = useState<{ is_running: boolean; stopped: boolean } | null>(null)
  const [togglingWorkers, setTogglingWorkers] = useState(false)
  const [requestHistory, setRequestHistory] = useState<Map<string, RequestHistoryPoint[]>>(new Map())
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'idle' | 'error'>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [routeHealthSummary, setRouteHealthSummary] = useState<RouteHealthSummary | null>(null)
  const [routeHealthResults, setRouteHealthResults] = useState<RouteHealthResult[]>([])
  const [routeHealthLoading, setRouteHealthLoading] = useState(false)
  const [routeHealthError, setRouteHealthError] = useState<string | null>(null)

  useEffect(() => {
    // Используем прямой URL к бэкенду для SSE
    const backendUrl = getBackendUrl()
    let reconnectAttempts = 0
    let reconnectTimeout: NodeJS.Timeout | null = null
    
    const connect = () => {
      try {
        // Используем Next.js API route для проксирования SSE
        const eventSource = new EventSource('/api/monitoring/providers/stream')
        
        eventSource.onopen = () => {
          setConnected(true)
          setError(null)
          reconnectAttempts = 0
        }
        
        eventSource.onmessage = (event) => {
          try {
            const parsed = JSON.parse(event.data)
            
            // Обрабатываем ошибки от сервера
            if (parsed.error) {
              console.error('[SSE] Error from server:', parsed.error)
              setError(parsed.error)
              return
            }
            
            if (parsed.type === 'connected') {
              return
            }
            const monitoringData = parsed as MonitoringData
            setData(monitoringData)
        
            // Обновляем историю запросов
            const now = Date.now()
            setRequestHistory(prev => {
              const newHistory = new Map(prev)
              
              monitoringData.providers.forEach(provider => {
                const history = newHistory.get(provider.id) || []
                const requestsPerMinute = provider.requests_per_second * 60
                
                // Добавляем новую точку
                const newPoint: RequestHistoryPoint = {
                  timestamp: now,
                  requests_per_minute: requestsPerMinute,
                  provider_id: provider.id,
                }
                
                // Удаляем точки старше 60 секунд
                const oneMinuteAgo = now - 60000
                const filteredHistory = history.filter(point => point.timestamp > oneMinuteAgo)
                filteredHistory.push(newPoint)
                
                newHistory.set(provider.id, filteredHistory)
              })
              
              return newHistory
            })
          } catch (err) {
            console.error('[SSE] Error parsing message:', err)
            handleError(err, 'Не удалось обработать данные мониторинга')
          }
        }
        
        eventSource.onerror = (err) => {
          // Проверяем состояние соединения
          if (eventSource.readyState === EventSource.CLOSED) {
            // Соединение закрыто - пытаемся переподключиться
            setConnected(false)
            eventSource.close()
            
            // Переподключение с экспоненциальной задержкой
            reconnectAttempts += 1
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000)
            
            if (reconnectAttempts <= 5) {
              console.log(`[SSE] Reconnecting in ${delay}ms (attempt ${reconnectAttempts})`)
              setError(`Соединение потеряно. Переподключение через ${(delay / 1000).toFixed(0)}с...`)
              reconnectTimeout = setTimeout(() => {
                connect()
              }, delay)
            } else {
              setError('Не удалось подключиться после нескольких попыток')
              console.error('[SSE] Connection failed after multiple attempts')
            }
          } else if (eventSource.readyState === EventSource.CONNECTING) {
            // Соединение устанавливается - это нормально, не логируем
            return
          } else {
            // Другие ошибки - логируем только если это не нормальное закрытие
            const errorMessage = err instanceof Error ? err.message : String(err)
            if (!errorMessage.includes('terminated') && !errorMessage.includes('Stream error')) {
              console.error('[SSE] Connection error:', err)
              setError('Ошибка подключения к серверу мониторинга')
            }
          }
        }
      } catch (err) {
        console.error('[SSE] Error creating EventSource:', err)
        setError('Не удалось установить соединение')
        handleError(err, 'Ошибка создания SSE соединения')
      }
    }
    
    // Начальное подключение
    connect()
    
    return () => {
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout)
      }
    }
  }, [handleError])

  // Загружаем статус воркеров KPVED
  useEffect(() => {
    const fetchWorkersStatus = async () => {
      try {
        const response = await fetch('/api/kpved/workers/status', {
          cache: 'no-store',
        })
        if (response.ok) {
          const status = await response.json()
          setWorkersStatus(status)
        }
      } catch (err) {
        console.error('Failed to fetch workers status:', err)
      }
    }

    fetchWorkersStatus()
    // Оптимизация: увеличиваем интервал до 10 секунд для снижения нагрузки
    const interval = setInterval(fetchWorkersStatus, 10000) // Обновляем каждые 10 секунд
    return () => clearInterval(interval)
  }, [])

  const fetchRouteHealth = useCallback(async () => {
    setRouteHealthLoading(true)
    try {
      const response = await fetch('/api/system/routes-health', { cache: 'no-store' })
      if (!response.ok) {
        throw new Error(`Не удалось проверить маршруты (HTTP ${response.status})`)
      }
      const payload = await response.json()
      setRouteHealthSummary(payload.summary)
      setRouteHealthResults(payload.results || [])
      setRouteHealthError(null)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Не удалось проверить маршруты'
      setRouteHealthError(message)
    } finally {
      setRouteHealthLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchRouteHealth()
    const interval = setInterval(fetchRouteHealth, 60000)
    return () => clearInterval(interval)
  }, [fetchRouteHealth])

  // Мемоизируем данные для графиков
  const chartData = useMemo(() => {
    if (!data) return []
    return data.providers.map(provider => ({
      name: provider.name,
      requests_per_second: provider.requests_per_second,
      average_latency_ms: provider.average_latency_ms,
      successful_requests: provider.successful_requests,
      failed_requests: provider.failed_requests,
    }))
  }, [data])

  // Фильтруем провайдеров
  const filteredProviders = useMemo(() => {
    if (!data) return []
    
    let filtered = data.providers
    
    // Фильтр по статусу
    if (statusFilter !== 'all') {
      filtered = filtered.filter(p => p.status === statusFilter)
    }
    
    // Фильтр по поисковому запросу
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(p => 
        p.name.toLowerCase().includes(query) ||
        p.id.toLowerCase().includes(query)
      )
    }
    
    return filtered
  }, [data, statusFilter, searchQuery])

  // Мемоизируем историю запросов для графиков
  const historyChartData = useMemo(() => {
    if (!data || requestHistory.size === 0) return []
    
    const allData: Array<{ timestamp: number; [key: string]: number | string }> = []
    const providerIds = Array.from(requestHistory.keys())
    
    // Собираем все уникальные временные метки
    const timestamps = new Set<number>()
    requestHistory.forEach(history => {
      history.forEach(point => timestamps.add(point.timestamp))
    })
    
    // Создаем точки данных для каждого timestamp
    Array.from(timestamps).sort().forEach(timestamp => {
      const point: { timestamp: number; [key: string]: number | string } = { timestamp }
      providerIds.forEach(providerId => {
        const history = requestHistory.get(providerId) || []
        const historyPoint = history.find(p => p.timestamp === timestamp)
        point[providerId] = historyPoint ? historyPoint.requests_per_minute : 0
      })
      allData.push(point)
    })
    
    return allData
  }, [data, requestHistory])

  const toggleWorkers = useCallback(async (stop: boolean) => {
    setTogglingWorkers(true)
    
    // Оптимистичное обновление UI
    if (workersStatus) {
      setWorkersStatus(prev => prev ? { ...prev, stopped: stop, is_running: !stop } : null)
    }
    
    try {
      const endpoint = stop ? '/api/kpved/workers/stop' : '/api/kpved/workers/resume'
      const response = await apiClientJson(endpoint, {
        method: 'POST',
      })
      
      // Обновляем данные с сервера для синхронизации
      setTimeout(async () => {
        try {
          const statusResponse = await fetch('/api/kpved/workers/status', { cache: 'no-store' })
          if (statusResponse.ok) {
            const status = await statusResponse.json()
            setWorkersStatus(status)
          }
        } catch (err) {
          console.error('Failed to refresh workers status:', err)
        }
      }, 500)
      
      // Показываем успешное уведомление
      toast.success(
        stop ? 'Воркеры KPVED остановлены' : 'Воркеры KPVED запущены',
        {
          description: stop 
            ? 'Обработка задач классификации КПВЭД приостановлена'
            : 'Обработка задач классификации КПВЭД возобновлена',
        }
      )
    } catch (err) {
      // Откатываем оптимистичное обновление при ошибке
      if (workersStatus) {
        setWorkersStatus(prev => prev ? { ...prev, stopped: !stop, is_running: stop } : null)
      }
      handleError(err, `Не удалось ${stop ? 'остановить' : 'запустить'} воркеры`)
    } finally {
      setTogglingWorkers(false)
    }
  }, [workersStatus, handleError])

  const toggleProvider = useCallback(async (providerId: string, enabled: boolean) => {
    setTogglingProvider(providerId)
    
    // Оптимистичное обновление UI
    if (data) {
      const newData = { ...data }
      const providerIndex = newData.providers.findIndex(p => p.id === providerId)
      if (providerIndex !== -1) {
        newData.providers[providerIndex] = {
          ...newData.providers[providerIndex],
          status: enabled ? 'idle' : 'active',
        }
        newData.system.active_providers = enabled 
          ? Math.max(0, newData.system.active_providers - 1)
          : Math.min(newData.system.total_providers, newData.system.active_providers + 1)
        setData(newData)
      }
    }
    
    try {
      await apiClientJson('/api/workers/config/update', {
        method: 'POST',
        body: JSON.stringify({
          action: 'update_provider',
          data: {
            name: providerId,
            enabled: !enabled,
          },
        }),
      })
      // Обновляем данные с сервера для синхронизации
      setTimeout(() => {
        const backendUrl = getBackendUrl()
        fetch(`${backendUrl}/api/monitoring/providers`)
          .then(res => res.json())
          .then(serverData => setData(serverData))
          .catch(err => handleError(err, 'Не удалось обновить данные'))
      }, 500)
      
      // Показываем успешное уведомление
      const provider = data?.providers.find(p => p.id === providerId)
      const providerName = provider?.name || providerId
      toast.success(
        enabled ? `${providerName} остановлен` : `${providerName} запущен`,
        {
          description: enabled 
            ? 'Провайдер временно отключен'
            : 'Провайдер активирован и готов к работе',
        }
      )
    } catch (err) {
      // Откатываем оптимистичное обновление при ошибке
      if (data) {
        const rollbackData = { ...data }
        const providerIndex = rollbackData.providers.findIndex(p => p.id === providerId)
        if (providerIndex !== -1) {
          rollbackData.providers[providerIndex] = {
            ...rollbackData.providers[providerIndex],
            status: enabled ? 'active' : 'idle',
          }
          rollbackData.system.active_providers = enabled 
            ? Math.min(rollbackData.system.total_providers, rollbackData.system.active_providers + 1)
            : Math.max(0, rollbackData.system.active_providers - 1)
          setData(rollbackData)
        }
      }
      handleError(err, `Не удалось ${enabled ? 'остановить' : 'запустить'} провайдер`)
    } finally {
      setTogglingProvider(null)
    }
  }, [data, handleError])

  const breadcrumbItems = [
    { label: 'Мониторинг', href: '/monitoring', icon: Activity },
  ]

  if (!data) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
        <div className="mb-4">
          <Breadcrumb items={breadcrumbItems} />
        </div>
        
        {/* Skeleton для заголовка */}
        <div className="mb-8">
          <Skeleton className="h-9 w-64 mb-2" />
          <Skeleton className="h-5 w-96" />
        </div>

        {/* Skeleton для статистики */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-4 rounded" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-20 mb-2" />
                <Skeleton className="h-4 w-32" />
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Skeleton для графиков */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {[1, 2].map((i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-6 w-48 mb-2" />
                <Skeleton className="h-4 w-64" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-[300px] w-full" />
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Skeleton для карточек провайдеров */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map((i) => (
            <Card key={i}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <Skeleton className="h-6 w-32" />
                  <Skeleton className="h-6 w-20" />
                </div>
                <Skeleton className="h-4 w-48 mt-2" />
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  {[1, 2, 3, 4].map((j) => (
                    <div key={j}>
                      <Skeleton className="h-4 w-20 mb-2" />
                      <Skeleton className="h-8 w-16" />
                    </div>
                  ))}
                </div>
                <Skeleton className="h-2 w-full" />
                <Skeleton className="h-[150px] w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="container-wide mx-auto px-4 py-8">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        {/* Заголовок */}
        <div className="mb-8">
          <motion.h1
            className="text-3xl font-bold mb-2 flex items-center gap-2"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <Activity className="h-8 w-8 text-primary" aria-hidden="true" />
            Мониторинг AI-провайдеров
          </motion.h1>
          <motion.p
            className="text-muted-foreground"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            Отслеживание использования провайдеров в реальном времени
          </motion.p>
          <div className="flex items-center gap-4 mt-4">
            <div className="flex items-center gap-2">
              <div 
                className={`h-3 w-3 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'} ${connected ? 'animate-pulse' : ''}`}
                role="status"
                aria-live="polite"
                aria-label={connected ? 'Подключено к серверу мониторинга' : 'Отключено от сервера мониторинга'}
              />
              <span className="text-sm text-muted-foreground" aria-hidden="true">
                {connected ? 'Подключено' : 'Отключено'}
              </span>
              <span className="sr-only">
                {connected ? 'Подключено к серверу мониторинга' : 'Отключено от сервера мониторинга'}
              </span>
            </div>
            {data && (
              <span className="text-xs text-muted-foreground">
                Обновлено: {formatTime(data.system.timestamp)}
              </span>
            )}
          </div>
          
          {/* Кнопка экспорта */}
          {data && (
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.3, delay: 0.2 }}
              className="mt-4"
            >
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm">
                    <Download className="mr-2 h-4 w-4" />
                    Экспорт данных
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => {
                    exportMonitoringToCSV(data)
                    toast.success('Данные экспортированы в CSV')
                  }}>
                    Экспорт в CSV
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => {
                    exportMonitoringToJSON(data)
                    toast.success('Данные экспортированы в JSON')
                  }}>
                    Экспорт в JSON
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </motion.div>
          )}
        </div>

        {error && (
          <Alert variant="destructive" className="mb-6">
            <AlertCircle className="h-4 w-4" aria-hidden="true" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <Card className="mb-6 border-primary/20 shadow-sm">
          <CardHeader className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>Проверка ключевых маршрутов</CardTitle>
              <CardDescription>
                Автоматический пинг основных backend эндпоинтов. Помогает заметить 404/500 сразу.
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              {routeHealthSummary?.checkedAt && (
                <span className="text-xs text-muted-foreground">
                  Обновлено {new Date(routeHealthSummary.checkedAt).toLocaleTimeString('ru-RU')}
                </span>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={fetchRouteHealth}
                disabled={routeHealthLoading}
              >
                <RefreshCw className={`mr-2 h-4 w-4 ${routeHealthLoading ? 'animate-spin' : ''}`} />
                Проверить
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {routeHealthError && (
              <Alert variant="destructive">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertTitle>Не удалось обновить статус маршрутов</AlertTitle>
                </div>
                <AlertDescription>{routeHealthError}</AlertDescription>
              </Alert>
            )}

            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatCard
                title="Всего маршрутов"
                value={routeHealthSummary?.total ?? (routeHealthResults.length > 0 ? routeHealthResults.length : '--')}
                icon={Server}
              />
              <StatCard
                title="Работают"
                value={routeHealthSummary?.healthy ?? routeHealthResults.filter((r) => r.ok).length}
                icon={ShieldCheck}
                variant="success"
              />
              <StatCard
                title="Деградированы"
                value={
                  routeHealthSummary?.degraded ??
                  routeHealthResults.filter((r) => !r.ok && !r.critical).length
                }
                icon={ShieldAlert}
                variant="warning"
              />
              <StatCard
                title="Критические"
                value={
                  routeHealthSummary?.failed ??
                  routeHealthResults.filter((r) => r.critical && !r.ok).length
                }
                icon={AlertTriangle}
                variant={
                  (routeHealthSummary?.failed ?? 0) > 0 ||
                  routeHealthResults.some((r) => r.critical && !r.ok)
                    ? 'danger'
                    : 'default'
                }
              />
            </div>

            <div className="space-y-2">
              {routeHealthLoading && routeHealthResults.length === 0 && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Проверяем маршруты...
                </div>
              )}

              {routeHealthResults.map((result) => {
                const statusLabel = result.ok
                  ? 'OK'
                  : result.isTimeout
                    ? 'Timeout'
                    : result.status
                      ? result.status
                      : 'Нет ответа'
                const statusClass = result.ok
                  ? 'bg-emerald-100 text-emerald-800 border border-emerald-200'
                  : result.critical
                    ? 'bg-red-100 text-red-800 border border-red-200'
                    : 'bg-amber-100 text-amber-800 border border-amber-200'

                return (
                  <div
                    key={`${result.method ?? 'GET'}-${result.path}`}
                    className="rounded-lg border bg-muted/10 p-3"
                  >
                    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                      <div>
                        <p className="text-sm font-medium">{result.label}</p>
                        <p className="text-xs text-muted-foreground">
                          {(result.method ?? 'GET').toUpperCase()} · {result.path}
                        </p>
                      </div>
                      <div className="flex flex-wrap items-center gap-2 text-sm">
                        <span className={`inline-flex items-center rounded-full px-2 py-1 text-xs ${statusClass}`}>
                          {statusLabel}
                          {result.statusText && result.ok && ` · ${result.statusText}`}
                        </span>
                        <span className="text-muted-foreground">
                          {result.durationMs.toLocaleString('ru-RU')} мс
                        </span>
                      </div>
                    </div>
                    {!result.ok && (
                      <p className="mt-2 text-xs text-red-600 dark:text-red-400">
                        {result.error || 'Маршрут недоступен'}
                      </p>
                    )}
                  </div>
                )
              })}

              {!routeHealthLoading && routeHealthResults.length === 0 && (
                <p className="text-sm text-muted-foreground">
                  Нет данных проверки. Нажмите «Проверить», чтобы обновить статус.
                </p>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Управление воркерами KPVED */}
        {workersStatus !== null && (
          <Card className="mb-6">
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Управление воркерами KPVED</span>
                <Badge variant={workersStatus.is_running && !workersStatus.stopped ? 'default' : 'secondary'}>
                  {workersStatus.is_running && !workersStatus.stopped ? 'Работают' : 'Остановлены'}
                </Badge>
              </CardTitle>
              <CardDescription>
                Управление воркерами классификации КПВЭД
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                <Button
                  variant={workersStatus.stopped ? 'default' : 'destructive'}
                  size="sm"
                  onClick={() => toggleWorkers(!workersStatus.stopped)}
                  disabled={togglingWorkers}
                  aria-label={workersStatus.stopped ? 'Запустить воркеры KPVED' : 'Остановить воркеры KPVED'}
                  aria-busy={togglingWorkers}
                  title={workersStatus.stopped ? 'Запустить воркеры KPVED' : 'Остановить воркеры KPVED'}
                >
                  {togglingWorkers ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      {workersStatus.stopped ? 'Запуск...' : 'Остановка...'}
                    </>
                  ) : workersStatus.stopped ? (
                    <>
                      <Power className="mr-2 h-4 w-4" />
                      Запустить воркеры
                    </>
                  ) : (
                    <>
                      <PowerOff className="mr-2 h-4 w-4" />
                      Остановить воркеры
                    </>
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Фильтры */}
        {data && (
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Фильтры</CardTitle>
              <CardDescription>Фильтрация провайдеров по статусу и названию</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col sm:flex-row gap-4">
                <div className="flex-1">
                  <Input
                    placeholder="Поиск по названию провайдера..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full"
                  />
                </div>
                <div className="w-full sm:w-48">
                  <Select value={statusFilter} onValueChange={(value: 'all' | 'active' | 'idle' | 'error') => setStatusFilter(value)}>
                    <SelectTrigger>
                      <SelectValue placeholder="Все статусы" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">Все статусы</SelectItem>
                      <SelectItem value="active">Активные</SelectItem>
                      <SelectItem value="idle">Ожидание</SelectItem>
                      <SelectItem value="error">Ошибки</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                {(statusFilter !== 'all' || searchQuery.trim()) && (
                  <Button
                    variant="outline"
                    onClick={() => {
                      setStatusFilter('all')
                      setSearchQuery('')
                    }}
                  >
                    Сбросить
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Общая статистика */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Всего запросов</CardTitle>
              <Server className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{data.system.total_requests.toLocaleString()}</div>
              <p className="text-xs text-muted-foreground">
                {data.system.total_successful.toLocaleString()} успешных
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Запросов/сек</CardTitle>
              <Zap className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {data.system.system_requests_per_second.toFixed(2)}
              </div>
              <p className="text-xs text-muted-foreground">
                Системная нагрузка
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Активных провайдеров</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {data.system.active_providers} / {data.system.total_providers}
              </div>
              <p className="text-xs text-muted-foreground">
                В работе
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Успешность</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {data.system.total_requests > 0
                  ? ((data.system.total_successful / data.system.total_requests) * 100).toFixed(1)
                  : 0}%
              </div>
              <p className="text-xs text-muted-foreground">
                {data.system.total_failed} ошибок
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Графики сравнения провайдеров */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* График нагрузки на провайдеры */}
          <Card>
            <CardHeader>
              <CardTitle>Нагрузка на провайдеры</CardTitle>
              <CardDescription>Запросов в секунду по каждому провайдеру</CardDescription>
            </CardHeader>
            <CardContent>
              <div role="img" aria-label="График нагрузки на провайдеров в запросах в секунду">
              <ResponsiveContainer width="100%" height={300}>
                <RechartsBarChart data={data.providers}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip 
                    formatter={(value: number) => [value.toFixed(2), 'Запросов/сек']}
                  />
                  <Legend />
                  <RechartsBar 
                    dataKey="requests_per_second" 
                    fill="#3b82f6" 
                    name="Запросов/сек"
                    radius={[8, 8, 0, 0]}
                  />
                </RechartsBarChart>
              </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>

          {/* График запросов в минуту (история) */}
          <Card>
            <CardHeader>
              <CardTitle>Запросов в минуту (история)</CardTitle>
              <CardDescription>Динамика запросов за последние 60 секунд</CardDescription>
            </CardHeader>
            <CardContent>
              {historyChartData.length > 0 ? (
                <div role="img" aria-label="График истории запросов в минуту за последние 60 секунд">
                <ResponsiveContainer width="100%" height={300}>
                  <RechartsLineChart data={historyChartData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis 
                        dataKey="timestamp" 
                        type="number"
                        scale="time"
                        domain={['dataMin', 'dataMax']}
                        tickFormatter={(value) => {
                          const date = new Date(value)
                          const secondsAgo = Math.floor((Date.now() - value) / 1000)
                          return secondsAgo <= 0 ? 'сейчас' : `${secondsAgo}с`
                        }}
                      />
                      <YAxis label={{ value: 'Запросов/мин', angle: -90, position: 'insideLeft' }} />
                      <Tooltip 
                        formatter={(value: number) => [value.toFixed(2), 'Запросов/мин']}
                        labelFormatter={(value) => {
                          const secondsAgo = Math.floor((Date.now() - value) / 1000)
                          return secondsAgo <= 0 ? 'Сейчас' : `${secondsAgo} секунд назад`
                        }}
                      />
                      <Legend />
                      {data.providers.map((provider, idx) => {
                        const color = PROVIDER_COLORS[provider.id.toLowerCase()] || `hsl(${idx * 60}, 70%, 50%)`
                        return (
                          <RechartsLine
                            key={provider.id}
                            type="monotone"
                            dataKey={provider.id}
                            name={provider.name}
                            stroke={color}
                            strokeWidth={2}
                            dot={false}
                            isAnimationActive={false}
                          />
                        )
                      })}
                    </RechartsLineChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                  Нет данных для отображения. Данные появятся после начала обработки запросов.
                </div>
              )}
            </CardContent>
          </Card>

          {/* График средней задержки */}
          <Card>
            <CardHeader>
              <CardTitle>Средняя задержка</CardTitle>
              <CardDescription>Время ответа провайдеров в миллисекундах</CardDescription>
            </CardHeader>
            <CardContent>
              {filteredProviders.length > 0 ? (
                <ResponsiveContainer width="100%" height={300}>
                  <RechartsBarChart data={filteredProviders}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="name" />
                    <YAxis />
                    <Tooltip 
                      formatter={(value: number) => [value.toFixed(0) + ' мс', 'Задержка']}
                    />
                    <Legend />
                    <RechartsBar 
                      dataKey="average_latency_ms" 
                      fill="#8b5cf6" 
                      name="Задержка (мс)"
                      radius={[8, 8, 0, 0]}
                    />
                  </RechartsBarChart>
                </ResponsiveContainer>
              ) : (
                <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                  Нет данных для отображения
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Карточки провайдеров */}
        {filteredProviders.length === 0 && data.providers.length > 0 ? (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <p className="text-muted-foreground">Нет провайдеров, соответствующих фильтрам</p>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-4"
                  onClick={() => {
                    setStatusFilter('all')
                    setSearchQuery('')
                  }}
                >
                  Сбросить фильтры
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredProviders.map((provider) => (
            <Card key={provider.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg">{provider.name}</CardTitle>
                  <div className="flex items-center gap-2">
                    <div 
                      className={`h-3 w-3 rounded-full ${STATUS_COLORS[provider.status]}`}
                      aria-label={`Статус: ${provider.status === 'active' ? 'Активен' : provider.status === 'idle' ? 'Ожидание' : 'Ошибка'}`}
                    />
                    <Badge variant={provider.status === 'active' ? 'default' : 'secondary'}>
                      {provider.status === 'active' ? 'Активен' : provider.status === 'idle' ? 'Ожидание' : 'Ошибка'}
                    </Badge>
                  </div>
                </div>
                <CardDescription>
                  {provider.active_channels} канал(ов) • {provider.current_requests} активных запросов
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Кнопки управления */}
                <div className="flex justify-end gap-2">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="outline" size="sm">
                        <Download className="h-3 w-3" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => {
                        exportProviderMetricsToCSV(provider)
                        toast.success(`Метрики ${provider.name} экспортированы`)
                      }}>
                        Экспорт метрик в CSV
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <Button
                    variant={provider.status === 'active' ? 'destructive' : 'default'}
                    size="sm"
                    onClick={() => toggleProvider(provider.id, provider.status === 'active')}
                    disabled={togglingProvider === provider.id}
                  >
                    {togglingProvider === provider.id ? (
                      <>
                        <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                        {provider.status === 'active' ? 'Остановка...' : 'Запуск...'}
                      </>
                    ) : provider.status === 'active' ? (
                      <>
                        <PowerOff className="mr-2 h-3 w-3" />
                        Остановить
                      </>
                    ) : (
                      <>
                        <Power className="mr-2 h-3 w-3" />
                        Запустить
                      </>
                    )}
                  </Button>
                </div>

                {/* Статистика */}
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Всего запросов</p>
                    <p className="text-2xl font-bold">{provider.total_requests.toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Запросов/сек</p>
                    <p className="text-2xl font-bold">{provider.requests_per_second.toFixed(2)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Успешных</p>
                    <p className="text-2xl font-bold text-green-600">
                      {provider.successful_requests.toLocaleString()}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Ошибок</p>
                    <p className="text-2xl font-bold text-red-600">
                      {provider.failed_requests.toLocaleString()}
                    </p>
                  </div>
                </div>

                {/* Средняя задержка */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-sm text-muted-foreground flex items-center gap-1">
                      <Clock className="h-3 w-3" aria-hidden="true" />
                      Средняя задержка
                    </p>
                    <p className="text-sm font-medium">
                      {provider.average_latency_ms > 0 
                        ? provider.average_latency_ms.toFixed(0) 
                        : '—'} мс
                    </p>
                  </div>
                  <Progress 
                    value={provider.average_latency_ms > 0 
                      ? Math.min((provider.average_latency_ms / 5000) * 100, 100)
                      : 0} 
                    className="h-2"
                    aria-label={`Средняя задержка: ${provider.average_latency_ms > 0 ? provider.average_latency_ms.toFixed(0) : 'N/A'} мс`}
                  />
                  {provider.average_latency_ms > 0 && (
                    <p className="text-xs text-muted-foreground mt-1">
                      {provider.average_latency_ms < 500 ? 'Отлично' :
                       provider.average_latency_ms < 1000 ? 'Хорошо' :
                       provider.average_latency_ms < 2000 ? 'Нормально' : 'Медленно'}
                    </p>
                  )}
                </div>

                {/* График успешности */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-sm text-muted-foreground">Успешность запросов</p>
                    <p className="text-sm font-medium">
                      {provider.total_requests > 0
                        ? ((provider.successful_requests / provider.total_requests) * 100).toFixed(1)
                        : 0}%
                    </p>
                  </div>
                  {provider.total_requests > 0 ? (
                    <ResponsiveContainer width="100%" height={150}>
                      <RechartsPieChart>
                        <RechartsPie
                          data={[
                            { name: 'Успешные', value: provider.successful_requests },
                            { name: 'Ошибки', value: provider.failed_requests },
                          ]}
                          cx="50%"
                          cy="50%"
                          labelLine={false}
                          label={({ name, percent }) => 
                            (percent || 0) > 0.05 ? `${((percent || 0) * 100).toFixed(0)}%` : ''
                          }
                          outerRadius={50}
                          fill="#8884d8"
                          dataKey="value"
                        >
                          <RechartsCell fill="#10b981" />
                          <RechartsCell fill="#ef4444" />
                        </RechartsPie>
                        <Tooltip 
                          formatter={(value: number, name: string) => [
                            value.toLocaleString(),
                            name
                          ]}
                        />
                      </RechartsPieChart>
                    </ResponsiveContainer>
                  ) : (
                    <div className="h-[150px] flex items-center justify-center text-muted-foreground text-sm">
                      Нет данных
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
            ))}
          </div>
        )}
        
        {filteredProviders.length === 0 && data.providers.length === 0 && (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <AlertCircle className="h-8 w-8 text-muted-foreground mx-auto mb-4" />
                <p className="text-muted-foreground">Нет зарегистрированных провайдеров</p>
              </div>
            </CardContent>
          </Card>
        )}
      </FadeIn>
    </div>
  )
}
