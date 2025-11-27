'use client'

import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import dynamic from 'next/dynamic'
import { Skeleton } from '@/components/ui/skeleton'
import { Activity, Zap, Database, TrendingUp, Clock, HardDrive } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { PerformanceIndicators } from '@/components/common/performance-indicator'

// Динамическая загрузка компонентов Recharts для уменьшения размера бандла
const DynamicPieChart = dynamic(
  () => import('recharts').then((mod) => ({
    default: ({ data, colors }: { data: Array<{ name: string; value: number }>; colors: string[] }) => {
      const { ResponsiveContainer, PieChart, Pie, Cell, Tooltip } = mod
      return (
        <ResponsiveContainer width="100%" height={300}>
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              labelLine={false}
              label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(0)}%`}
              outerRadius={80}
              fill="#8884d8"
              dataKey="value"
            >
              {data.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={colors[index % colors.length]} />
              ))}
            </Pie>
            <Tooltip />
          </PieChart>
        </ResponsiveContainer>
      )
    },
  })),
  {
    ssr: false,
    loading: () => (
      <div className="flex items-center justify-center h-[300px]">
        <Skeleton className="h-full w-full" />
      </div>
    ),
  }
)

const DynamicBarChart = dynamic(
  () => import('recharts').then((mod) => ({
    default: ({ data }: { data: Array<{ route: string; ssr: number; csr: number }> }) => {
      const { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend } = mod
      return (
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="route" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="ssr" fill="#3b82f6" name="SSR" />
            <Bar dataKey="csr" fill="#22c55e" name="CSR" />
          </BarChart>
        </ResponsiveContainer>
      )
    },
  })),
  {
    ssr: false,
    loading: () => (
      <div className="flex items-center justify-center h-[300px]">
        <Skeleton className="h-full w-full" />
      </div>
    ),
  }
)

const DynamicAreaChart = dynamic(
  () => import('recharts').then((mod) => ({
    default: ({ data }: { data: Array<{ endpoint: string; count: number; avgDuration: number }> }) => {
      const { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend } = mod
      return (
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={data}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="endpoint" />
            <YAxis yAxisId="left" />
            <YAxis yAxisId="right" orientation="right" />
            <Tooltip />
            <Legend />
            <Area
              yAxisId="left"
              type="monotone"
              dataKey="count"
              fill="#3b82f6"
              fillOpacity={0.6}
              name="Количество запросов"
            />
            <Area
              yAxisId="right"
              type="monotone"
              dataKey="avgDuration"
              fill="#22c55e"
              fillOpacity={0.6}
              name="Среднее время (мс)"
            />
          </AreaChart>
        </ResponsiveContainer>
      )
    },
  })),
  {
    ssr: false,
    loading: () => (
      <div className="flex items-center justify-center h-[300px]">
        <Skeleton className="h-full w-full" />
      </div>
    ),
  }
)

interface MetricsData {
  bundleSize: {
    js: number
    css: number
    total: number
  }
  renderTimes: Array<{
    route: string
    ssr: number
    csr: number
  }>
  apiRequests: Array<{
    endpoint: string
    count: number
    avgDuration: number
  }>
  webVitals: {
    lcp: number
    fid: number
    cls: number
  }
}

const COLORS = ['#3b82f6', '#22c55e', '#ef4444', '#f59e0b', '#8b5cf6']

export default function MetricsPage() {
  const [metrics, setMetrics] = useState<MetricsData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchMetrics()
    
    // Обновляем метрики каждые 5 секунд
    const interval = setInterval(fetchMetrics, 5000)
    
    return () => clearInterval(interval)
  }, [])

  async function fetchMetrics() {
    try {
      setLoading(true)
      setError(null)
      
      // Получаем метрики из API
      const [bundleRes, renderRes, apiRes, vitalsRes] = await Promise.allSettled([
        fetch('/api/monitoring/metrics'),
        fetch('/api/dashboard/stats'),
        fetch('/api/monitoring/metrics'),
        fetch('/api/monitoring/metrics'),
      ])

      // В реальном приложении здесь будут реальные данные из API
      // Сейчас используем мокированные данные для демонстрации
      const mockMetrics: MetricsData = {
        bundleSize: {
          js: 245,
          css: 32,
          total: 277,
        },
        renderTimes: [
          { route: '/', ssr: 120, csr: 45 },
          { route: '/clients', ssr: 180, csr: 60 },
          { route: '/processes', ssr: 200, csr: 80 },
          { route: '/quality', ssr: 150, csr: 55 },
        ],
        apiRequests: [
          { endpoint: '/api/dashboard/stats', count: 120, avgDuration: 45 },
          { endpoint: '/api/monitoring/metrics', count: 80, avgDuration: 30 },
          { endpoint: '/api/clients', count: 60, avgDuration: 120 },
          { endpoint: '/api/quality/metrics', count: 40, avgDuration: 200 },
        ],
        webVitals: {
          lcp: 2.1,
          fid: 50,
          cls: 0.05,
        },
      }

      setMetrics(mockMetrics)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки метрик')
      console.error('Error fetching metrics:', err)
    } finally {
      setLoading(false)
    }
  }

  if (loading && !metrics) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-muted-foreground">Загрузка метрик...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-destructive">Ошибка: {error}</div>
      </div>
    )
  }

  if (!metrics) {
    return null
  }

  return (
    <div className="space-y-6">
      {/* Web Vitals */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Core Web Vitals
            </CardTitle>
            <CardDescription>
              Метрики производительности страниц
            </CardDescription>
          </CardHeader>
          <CardContent>
            <PerformanceIndicators
              items={[
                {
                  label: 'LCP',
                  value: metrics.webVitals.lcp,
                  unit: 's',
                  thresholds: { good: 2.5, needsImprovement: 4.0 },
                  higherIsBetter: false,
                },
                {
                  label: 'FID',
                  value: metrics.webVitals.fid,
                  unit: 'ms',
                  thresholds: { good: 100, needsImprovement: 300 },
                  higherIsBetter: false,
                },
                {
                  label: 'CLS',
                  value: metrics.webVitals.cls,
                  unit: '',
                  thresholds: { good: 0.1, needsImprovement: 0.25 },
                  higherIsBetter: false,
                },
              ]}
            />
          </CardContent>
        </Card>
      </motion.div>

      {/* Bundle Size */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.1 }}
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <HardDrive className="h-5 w-5" />
              Размер бандла
            </CardTitle>
            <CardDescription>
              Размер JavaScript и CSS файлов
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DynamicPieChart
              data={[
                { name: 'JavaScript', value: metrics.bundleSize.js },
                { name: 'CSS', value: metrics.bundleSize.css },
              ]}
              colors={COLORS}
            />
            <div className="mt-4 text-center">
              <div className="text-2xl font-bold">{metrics.bundleSize.total} KB</div>
              <div className="text-sm text-muted-foreground">Общий размер бандла</div>
            </div>
          </CardContent>
        </Card>
      </motion.div>

      {/* Render Times */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.2 }}
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-5 w-5" />
              Время рендеринга
            </CardTitle>
            <CardDescription>
              Время рендеринга серверных и клиентских компонентов (мс)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DynamicBarChart data={metrics.renderTimes} />
          </CardContent>
        </Card>
      </motion.div>

      {/* API Requests */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.3 }}
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              Запросы к API
            </CardTitle>
            <CardDescription>
              Количество и среднее время выполнения запросов
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DynamicAreaChart data={metrics.apiRequests} />
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}
