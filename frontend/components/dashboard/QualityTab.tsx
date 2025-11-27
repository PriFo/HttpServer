'use client'

import { useEffect, useState, useCallback } from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { CheckCircle2, AlertTriangle, TrendingUp, BarChart3, Link as LinkIcon } from 'lucide-react'
import Link from 'next/link'
import { useDashboardStore } from '@/stores/dashboard-store'
import { apiClientJson } from '@/lib/api-client'
import { QualityDistributionChart } from '@/components/quality/QualityDistributionChart'
import { cn } from '@/lib/utils'

export function QualityTab() {
  const {
    systemStats,
    setSystemStats,
    setLoading,
    backendFallback,
    setBackendFallback,
  } = useDashboardStore()
  const [qualityStats, setQualityStats] = useState<ProjectQualityStats | null>(null)

  const loadQualityData = useCallback(async () => {
    try {
      setLoading(true)
      const fallbackReasons: string[] = []
      
      const [metricsData, statsData] = await Promise.allSettled([
        apiClientJson<QualityMetricsResponse>('/api/quality/metrics', { skipErrorHandler: true }),
        apiClientJson<ProjectQualityStats>('/api/quality/stats', { skipErrorHandler: true }),
      ])

      if (metricsData.status === 'fulfilled') {
        const metrics = metricsData.value
        if (metrics?.isFallback) {
          fallbackReasons.push(normalizeFallbackReason(metrics.fallbackReason))
        }
        setSystemStats({
          qualityMetrics: {
            overallQuality: metrics?.overallQuality ?? 0,
            highConfidence: metrics?.highConfidence ?? 0,
            mediumConfidence: metrics?.mediumConfidence ?? 0,
            lowConfidence: metrics?.lowConfidence ?? 0,
            totalRecords: metrics?.totalRecords,
          },
        })
      } else if (metricsData.status === 'rejected') {
        const error = metricsData.reason
        if (error && typeof error === 'object' && 'message' in error) {
          fallbackReasons.push(normalizeFallbackReason((error as Error).message))
        }
      }

      if (statsData.status === 'fulfilled') {
        setQualityStats(statsData.value)
      } else if (statsData.status === 'rejected') {
        const error = statsData.reason
        if (error && typeof error === 'object' && 'message' in error) {
          fallbackReasons.push(normalizeFallbackReason((error as Error).message))
        }
      }

      if (fallbackReasons.length > 0) {
        setBackendFallback({
          isActive: true,
          reasons: Array.from(new Set([...(backendFallback?.reasons || []), ...fallbackReasons])),
          timestamp: new Date().toISOString(),
        })
      }
    } catch (error) {
      try {
        const errorMessage = error instanceof Error ? error.message : String(error)
        setBackendFallback({
          isActive: true,
          reasons: Array.from(new Set([...(backendFallback?.reasons || []), normalizeFallbackReason(errorMessage)])),
          timestamp: new Date().toISOString(),
        })
      } catch {
        // ignore
      }
    } finally {
      setLoading(false)
    }
  }, [backendFallback?.reasons, setBackendFallback, setLoading, setSystemStats])

  useEffect(() => {
    loadQualityData()
    const interval = setInterval(loadQualityData, 30000)
    return () => clearInterval(interval)
  }, [loadQualityData])

  const normalizeFallbackReason = (reason?: string) => {
    if (!reason || reason.trim().length === 0) {
      return 'Данные качества недоступны. Проверьте состояние backend сервиса.'
    }
    const lower = reason.toLowerCase()
    if (lower.includes('body is unusable')) {
      return 'Эндпоинт /api/quality/metrics вернул пустой ответ.'
    }
    if (lower.includes('fetch failed') || lower.includes('failed to fetch')) {
      return 'Не удалось подключиться к backend серверу.'
    }
    return reason
  }

  interface QualityMetricsResponse {
    isFallback?: boolean
    fallbackReason?: string
    overallQuality?: number
    highConfidence?: number
    mediumConfidence?: number
    lowConfidence?: number
    totalRecords?: number
  }

  interface ProjectQualityStats {
    total_items: number
    average_quality: number
    benchmark_count: number
    benchmark_percentage: number
    by_level?: Record<string, {
      count: number
      avg_quality: number
      percentage: number
    }>
    isFallback?: boolean
    fallbackReason?: string
    last_activity?: string
  }

  const qualityMetrics = systemStats?.qualityMetrics as QualityMetricsResponse | undefined

  const safeNumber = (value?: number): number => {
    if (value === null || value === undefined || Number.isNaN(value)) {
      return 0
    }
    return value
  }

  // Нормализуем значения, чтобы они были в диапазоне 0-1
  const overallQuality = Math.max(0, Math.min(1, safeNumber(qualityMetrics?.overallQuality)))
  const highConfidence = Math.max(0, Math.min(1, safeNumber(qualityMetrics?.highConfidence)))
  const mediumConfidence = Math.max(0, Math.min(1, safeNumber(qualityMetrics?.mediumConfidence)))
  const lowConfidence = Math.max(0, Math.min(1, safeNumber(qualityMetrics?.lowConfidence)))
  const totalRecords = safeNumber(qualityMetrics?.totalRecords)

  const getQualityColor = (quality: number) => {
    if (quality >= 0.9) return 'text-green-600'
    if (quality >= 0.7) return 'text-yellow-600'
    return 'text-red-600'
  }


  return (
    <div className="container mx-auto p-6 space-y-6">
      <div>
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <CheckCircle2 className="h-6 w-6" />
          Качество данных
        </h2>
        <p className="text-muted-foreground mt-1">
          Анализ и мониторинг качества нормализованных данных
        </p>
      </div>

      {/* Quality Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
        >
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <TrendingUp className="h-4 w-4" />
                Общее качество
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className={cn("text-3xl font-bold mb-2", getQualityColor(overallQuality))}>
                {(overallQuality * 100).toFixed(1)}%
              </div>
              <Progress 
                value={overallQuality * 100} 
                className="h-2"
              />
            </CardContent>
          </Card>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Высокая уверенность</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-green-600 mb-2">
                {(highConfidence * 100).toFixed(1)}%
              </div>
              <Progress value={highConfidence * 100} className="h-2" />
            </CardContent>
          </Card>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Средняя уверенность</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-yellow-600 mb-2">
                {(mediumConfidence * 100).toFixed(1)}%
              </div>
              <Progress value={mediumConfidence * 100} className="h-2" />
            </CardContent>
          </Card>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Низкая уверенность</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold text-red-600 mb-2">
                {(lowConfidence * 100).toFixed(1)}%
              </div>
              <Progress value={lowConfidence * 100} className="h-2" />
            </CardContent>
          </Card>
        </motion.div>
      </div>

      {/* Quality Chart */}
      {totalRecords > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
        >
          <Card>
            <CardHeader>
              <CardTitle>Распределение уверенности классификации</CardTitle>
              <CardDescription>
                Какой процент записей имеет различный уровень уверенности
              </CardDescription>
            </CardHeader>
            <CardContent>
              <QualityDistributionChart
                data={[
                  {
                    range: '0.9-1.0',
                    count: Math.round(highConfidence * totalRecords),
                    percentage: highConfidence * 100,
                  },
                  {
                    range: '0.7-0.9',
                    count: Math.round(mediumConfidence * totalRecords),
                    percentage: mediumConfidence * 100,
                  },
                  {
                    range: '0.0-0.7',
                    count: Math.round(lowConfidence * totalRecords),
                    percentage: lowConfidence * 100,
                  },
                ]}
                totalRecords={totalRecords}
                viewType="pie"
              />
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Top Issues Summary */}
      {qualityStats && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.5 }}
          className="grid grid-cols-1 md:grid-cols-3 gap-4"
        >
          {/* Duplicates, violations, suggestions removed - not in ProjectQualityStats type */}
        </motion.div>
      )}

      {/* Quick Links */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.6 }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Быстрые ссылки</CardTitle>
            <CardDescription>
              Переход к детальному анализу качества данных
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Button variant="outline" asChild>
                <Link href="/quality/violations">
                  <AlertTriangle className="h-4 w-4 mr-2" />
                  Нарушения качества
                </Link>
              </Button>
              <Button variant="outline" asChild>
                <Link href="/quality/duplicates">
                  <BarChart3 className="h-4 w-4 mr-2" />
                  Дубликаты
                </Link>
              </Button>
              <Button variant="outline" asChild>
                <Link href="/quality/suggestions">
                  <LinkIcon className="h-4 w-4 mr-2" />
                  Предложения
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}

