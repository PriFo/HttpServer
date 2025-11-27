/**
 * Хук для получения метрик приложения
 * 
 * Используется на странице /internal/metrics для получения
 * метрик производительности, размера бандла и Core Web Vitals
 */

'use client'

import { useState, useEffect } from 'react'

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

interface UseMetricsOptions {
  autoRefresh?: boolean
  refreshInterval?: number
}

/**
 * Хук для получения метрик приложения
 * 
 * @param options - Опции для настройки автообновления
 * @returns Объект с метриками, состоянием загрузки и ошибками
 * 
 * @example
 * ```tsx
 * const { metrics, loading, error, refresh } = useMetrics({
 *   autoRefresh: true,
 *   refreshInterval: 5000,
 * })
 * ```
 */
export function useMetrics(options: UseMetricsOptions = {}) {
  const { autoRefresh = false, refreshInterval = 5000 } = options

  const [metrics, setMetrics] = useState<MetricsData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchMetrics = async () => {
    try {
      setLoading(true)
      setError(null)

      // Получаем метрики из API
      const [bundleRes, renderRes, apiRes, vitalsRes] = await Promise.all([
        fetch('/api/monitoring/metrics').catch(() => null),
        fetch('/api/dashboard/stats').catch(() => null),
        fetch('/api/monitoring/metrics').catch(() => null),
        fetch('/api/monitoring/metrics').catch(() => null),
      ])

      // В реальном приложении здесь будут реальные данные
      // Сейчас используем мокированные данные
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
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMetrics()

    if (autoRefresh) {
      const interval = setInterval(fetchMetrics, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [autoRefresh, refreshInterval])

  return {
    metrics,
    loading,
    error,
    refresh: fetchMetrics,
  }
}

