'use client'

import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { AlertCircle, RefreshCw, Trash2, TrendingUp, Activity, Server, FileX, AlertTriangle } from 'lucide-react'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { EmptyState } from '@/components/common/empty-state'
import { toast } from 'sonner'
import dynamic from 'next/dynamic'
import { fetchJson, getErrorMessage } from '@/lib/fetch-utils'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

// Динамическая загрузка графиков
const DynamicBarChart = dynamic(
  () => import('recharts').then((mod) => {
    const { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, Legend, CartesianGrid } = mod
    return ({ data, dataKey, nameKey }: { data: any[]; dataKey: string; nameKey: string }) => (
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey={nameKey} />
          <YAxis />
          <Tooltip />
          <Legend />
          <Bar dataKey={dataKey} fill="#3b82f6" />
        </BarChart>
      </ResponsiveContainer>
    )
  }),
  { ssr: false }
)

const DynamicPieChart = dynamic(
  () => import('recharts').then((mod) => {
    const { ResponsiveContainer, PieChart, Pie, Cell, Tooltip, Legend } = mod
    return ({ data }: { data: Array<{ name: string; value: number }> }) => {
      const COLORS = ['#3b82f6', '#ef4444', '#f59e0b', '#22c55e', '#8b5cf6', '#ec4899', '#14b8a6', '#6366f1']
      return (
        <ResponsiveContainer width="100%" height={300}>
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              labelLine={false}
              label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(1)}%`}
              outerRadius={80}
              fill="#8884d8"
              dataKey="value"
            >
              {data.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
              ))}
            </Pie>
            <Tooltip />
            <Legend />
          </PieChart>
        </ResponsiveContainer>
      )
    }
  }),
  { ssr: false }
)

interface ErrorMetrics {
  total_errors: number
  errors_by_type: Record<string, number>
  errors_by_code: Record<string, number>
  errors_by_endpoint: Record<string, number>
}

interface ErrorEntry {
  timestamp: string
  type: string
  code: number
  message: string
  endpoint: string
  request_id: string
  user_message: string
}

interface LastErrorsResponse {
  errors: ErrorEntry[]
  count: number
  limit: number
}

export default function ErrorsPage() {
  const [metrics, setMetrics] = useState<ErrorMetrics | null>(null)
  const [lastErrors, setLastErrors] = useState<ErrorEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState('overview')

  useEffect(() => {
    fetchMetrics()
    fetchLastErrors()
    const interval = setInterval(() => {
      fetchMetrics()
      fetchLastErrors()
    }, 5000) // Обновляем каждые 5 секунд
    return () => clearInterval(interval)
  }, [])

  async function fetchMetrics() {
    try {
      setError(null)
      const data = await fetchJson<ErrorMetrics>('/api/errors/metrics', {
        timeout: QUALITY_TIMEOUTS.FAST,
        cache: 'no-store',
      })
      setMetrics(data)
      setLoading(false)
    } catch (err) {
      const errorMessage = getErrorMessage(err, 'Произошла ошибка')
      setError(errorMessage)
      setLoading(false)
    }
  }

  async function fetchLastErrors() {
    try {
      const data = await fetchJson<LastErrorsResponse>('/api/errors/last?limit=50', {
        timeout: QUALITY_TIMEOUTS.FAST,
        cache: 'no-store',
      })
      setLastErrors(data.errors || [])
    } catch (err) {
      console.error('Error fetching last errors:', err)
    }
  }

  async function handleReset() {
    try {
      await fetchJson('/api/errors/reset', {
        method: 'POST',
        timeout: QUALITY_TIMEOUTS.STANDARD,
      })
      toast.success('Метрики ошибок сброшены')
      fetchMetrics()
    } catch (err) {
      const errorMessage = getErrorMessage(err, 'Произошла ошибка')
      toast.error(errorMessage)
    }
  }

  if (loading && !metrics) {
    return <LoadingState message="Загрузка метрик ошибок..." />
  }

  if (error && !metrics) {
    return <ErrorState message={error} action={{ label: 'Повторить', onClick: fetchMetrics }} />
  }

  if (!metrics) {
    return <EmptyState title="Нет данных о метриках ошибок" />
  }

  // Преобразуем данные для графиков
  const errorsByTypeData = Object.entries(metrics.errors_by_type || {}).map(([name, value]) => ({
    name,
    value: Number(value),
  }))

  const errorsByCodeData = Object.entries(metrics.errors_by_code || {}).map(([name, value]) => ({
    name: `HTTP ${name}`,
    value: Number(value),
  }))

  const errorsByEndpointData = Object.entries(metrics.errors_by_endpoint || {})
    .sort((a, b) => Number(b[1]) - Number(a[1]))
    .slice(0, 10)
    .map(([name, value]) => ({
      name: name.length > 30 ? name.substring(0, 30) + '...' : name,
      value: Number(value),
    }))


  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Метрики ошибок</h1>
          <p className="text-muted-foreground mt-2">
            Мониторинг и анализ ошибок в системе
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={fetchMetrics} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Обновить
          </Button>
          <Button onClick={handleReset} variant="destructive" size="sm">
            <Trash2 className="h-4 w-4 mr-2" />
            Сбросить
          </Button>
        </div>
      </div>

      {/* Общая статистика */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Всего ошибок</CardTitle>
            <AlertCircle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics.total_errors.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              Всего зарегистрировано
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Последние ошибки</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{lastErrors.length}</div>
            <p className="text-xs text-muted-foreground">
              В списке последних
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Типов ошибок</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{Object.keys(metrics.errors_by_type || {}).length}</div>
            <p className="text-xs text-muted-foreground">
              Уникальных типов
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Эндпоинтов</CardTitle>
            <FileX className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{Object.keys(metrics.errors_by_endpoint || {}).length}</div>
            <p className="text-xs text-muted-foreground">
              С ошибками
            </p>
          </CardContent>
        </Card>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="by-type">По типу</TabsTrigger>
          <TabsTrigger value="by-code">По коду</TabsTrigger>
          <TabsTrigger value="by-endpoint">По эндпоинту</TabsTrigger>
          <TabsTrigger value="last-errors">Последние ошибки</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle>Ошибки по типу</CardTitle>
                <CardDescription>Распределение ошибок по типам</CardDescription>
              </CardHeader>
              <CardContent>
                {errorsByTypeData.length > 0 ? (
                  <DynamicPieChart data={errorsByTypeData} />
                ) : (
                  <EmptyState title="Нет данных" />
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Ошибки по HTTP коду</CardTitle>
                <CardDescription>Распределение ошибок по статус кодам</CardDescription>
              </CardHeader>
              <CardContent>
                {errorsByCodeData.length > 0 ? (
                  <DynamicBarChart data={errorsByCodeData} dataKey="value" nameKey="name" />
                ) : (
                  <EmptyState title="Нет данных" />
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="by-type" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Ошибки по типу</CardTitle>
              <CardDescription>Детальное распределение ошибок по типам</CardDescription>
            </CardHeader>
            <CardContent>
              {errorsByTypeData.length > 0 ? (
                <div className="space-y-2">
                  {errorsByTypeData
                    .sort((a, b) => b.value - a.value)
                    .map((item) => (
                      <div key={item.name} className="flex items-center justify-between p-2 border rounded">
                        <span className="font-medium">{item.name}</span>
                        <span className="text-lg font-bold">{item.value.toLocaleString()}</span>
                      </div>
                    ))}
                </div>
              ) : (
                <EmptyState title="Нет данных" />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="by-code" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Ошибки по HTTP коду</CardTitle>
              <CardDescription>Распределение ошибок по статус кодам</CardDescription>
            </CardHeader>
            <CardContent>
              {errorsByCodeData.length > 0 ? (
                <div className="space-y-2">
                  {errorsByCodeData
                    .sort((a, b) => b.value - a.value)
                    .map((item) => (
                      <div key={item.name} className="flex items-center justify-between p-2 border rounded">
                        <span className="font-medium">{item.name}</span>
                        <span className="text-lg font-bold">{item.value.toLocaleString()}</span>
                      </div>
                    ))}
                </div>
              ) : (
                <EmptyState title="Нет данных" />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="by-endpoint" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Ошибки по эндпоинту</CardTitle>
              <CardDescription>Топ-10 эндпоинтов с наибольшим количеством ошибок</CardDescription>
            </CardHeader>
            <CardContent>
              {errorsByEndpointData.length > 0 ? (
                <>
                  <DynamicBarChart data={errorsByEndpointData} dataKey="value" nameKey="name" />
                  <div className="mt-4 space-y-2">
                    {errorsByEndpointData.map((item) => (
                      <div key={item.name} className="flex items-center justify-between p-2 border rounded">
                        <span className="font-medium text-sm">{item.name}</span>
                        <span className="text-lg font-bold">{item.value.toLocaleString()}</span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <EmptyState title="Нет данных" />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="last-errors" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Последние ошибки</CardTitle>
              <CardDescription>Последние 50 ошибок с деталями</CardDescription>
            </CardHeader>
            <CardContent>
              {lastErrors && lastErrors.length > 0 ? (
                <div className="space-y-2 max-h-[600px] overflow-y-auto">
                  {lastErrors.map((err, index) => (
                    <motion.div
                      key={index}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: index * 0.05 }}
                      className="p-4 border rounded-lg space-y-2"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <AlertTriangle className={`h-4 w-4 ${
                            err.code >= 500 ? 'text-red-500' :
                            err.code >= 400 ? 'text-yellow-500' :
                            'text-blue-500'
                          }`} />
                          <span className="font-medium">{err.type}</span>
                          <span className="text-sm text-muted-foreground">HTTP {err.code}</span>
                        </div>
                        <span className="text-xs text-muted-foreground">
                          {new Date(err.timestamp).toLocaleString('ru-RU')}
                        </span>
                      </div>
                      <div className="text-sm">
                        <div className="font-medium">{err.user_message}</div>
                        <div className="text-muted-foreground mt-1">
                          <div>Эндпоинт: {err.endpoint}</div>
                          {err.request_id && (
                            <div>Request ID: {err.request_id}</div>
                          )}
                        </div>
                      </div>
                    </motion.div>
                  ))}
                </div>
              ) : (
                <EmptyState title="Нет последних ошибок" />
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

