'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { BarChart3, Package, Users, TrendingUp, AlertCircle, RefreshCw } from 'lucide-react'
import { StatCard } from '@/components/common/stat-card'
import { ErrorState } from '@/components/common/error-state'

interface NormalizationStats {
  total_processed?: number
  total_groups?: number
  total_merged?: number
  categories?: Record<string, number>
  success_rate?: number
  error_rate?: number
}

interface NormalizationStatsProps {
  type: 'nomenclature' | 'counterparties'
  clientId?: number | null
  projectId?: number | null
}

export function NormalizationStats({ type, clientId, projectId }: NormalizationStatsProps) {
  const [stats, setStats] = useState<NormalizationStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [isFromCache, setIsFromCache] = useState(false)
  
  // Кэш для статистики (5 минут)
  const CACHE_KEY = `normalization_stats_${type}_${clientId || 'all'}_${projectId || 'all'}`
  const CACHE_DURATION = 5 * 60 * 1000 // 5 минут

  const fetchStats = useCallback(async (forceRefresh = false) => {
    // Проверяем кэш
    if (!forceRefresh) {
      try {
        const cached = localStorage.getItem(CACHE_KEY)
        if (cached) {
          const { data, timestamp } = JSON.parse(cached)
          const age = Date.now() - timestamp
          if (age < CACHE_DURATION) {
            setStats(data)
            setLastUpdated(new Date(timestamp))
            setIsFromCache(true)
            setLoading(false)
            setError(null)
            // Загружаем свежие данные в фоне
            if (age > CACHE_DURATION / 2) {
              // Если кэш старше половины времени жизни, обновляем в фоне
              fetchStats(true)
            }
            return
          }
        }
      } catch (err) {
        // Игнорируем ошибки чтения кэша
        console.warn('Failed to read cache:', err)
      }
    }

    if (forceRefresh) {
      setError(null)
    }
    
    setLoading(true)
    setIsFromCache(false)

    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут

      let endpoint = type === 'nomenclature' 
        ? '/api/normalization/stats'
        : '/api/counterparties/normalized/stats'

      // Если указаны clientId и projectId, используем специфичный endpoint
      if (clientId && projectId) {
        if (type === 'nomenclature') {
          endpoint = `/api/clients/${clientId}/projects/${projectId}/normalization/stats`
        } else {
          endpoint = `/api/counterparties/normalized/stats?client_id=${clientId}&project_id=${projectId}`
        }
      }

      const response = await fetch(endpoint, {
        cache: 'no-store',
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        // Если ошибка, но есть кэш, показываем его
        if (!forceRefresh && stats) {
          setIsFromCache(true)
          setError('Не удалось обновить статистику. Показаны кэшированные данные.')
          setLoading(false)
          return
        }
        
        let errorMessage = 'Не удалось загрузить статистику'
        if (response.status === 404) {
          errorMessage = 'Статистика не найдена'
        } else if (response.status === 503 || response.status === 504) {
          errorMessage = 'Сервер временно недоступен. Проверьте подключение к backend серверу на порту 9999'
        } else if (response.status >= 500) {
          errorMessage = `Ошибка сервера: ${response.status}`
        }
        throw new Error(errorMessage)
      }

      const data = await response.json()
      setStats(data)
      const now = new Date()
      setLastUpdated(now)
      setIsFromCache(false)
      setError(null)
      
      // Сохраняем в кэш
      try {
        localStorage.setItem(CACHE_KEY, JSON.stringify({
          data,
          timestamp: now.getTime()
        }))
      } catch (err) {
        // Игнорируем ошибки сохранения в кэш
        console.warn('Failed to save cache:', err)
      }
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        // Если есть кэш, показываем его
        if (!forceRefresh && stats) {
          setIsFromCache(true)
          setError(err.message + '. Показаны кэшированные данные.')
        } else {
          setError(err.message)
        }
      }
    } finally {
      setLoading(false)
    }
  }, [type, clientId, projectId, CACHE_KEY, CACHE_DURATION, stats])

  useEffect(() => {
    fetchStats()
    
    // Обновляем статистику каждые 30 секунд
    const interval = setInterval(() => fetchStats(), 30000)
    return () => clearInterval(interval)
  }, [fetchStats])

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {[...Array(4)].map((_, i) => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
    )
  }

  if (error && !stats) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Статистика нормализации</CardTitle>
          <CardDescription>
            {type === 'nomenclature' ? 'Номенклатура' : 'Контрагенты'}
          </CardDescription>
        </CardHeader>
        <CardContent className="pt-6">
          <ErrorState
            message={error}
            action={{
              label: "Повторить",
              onClick: () => fetchStats(true)
            }}
          />
        </CardContent>
      </Card>
    )
  }

  const totalProcessed = stats?.total_processed || 0
  const totalGroups = stats?.total_groups || 0
  const totalMerged = stats?.total_merged || 0
  const successRate = stats?.success_rate || 0
  const errorRate = stats?.error_rate || 0

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold">
            Статистика нормализации {type === 'nomenclature' ? 'номенклатуры' : 'контрагентов'}
          </h3>
          {lastUpdated && (
            <p className="text-sm text-muted-foreground">
              {isFromCache && 'Кэшированные данные. '}
              Обновлено: {lastUpdated.toLocaleTimeString('ru-RU')}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {error && (
            <p className="text-sm text-yellow-600 dark:text-yellow-400">{error}</p>
          )}
          <Button
            onClick={() => fetchStats(true)}
            variant="outline"
            size="sm"
            disabled={loading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Обновить
          </Button>
        </div>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Обработано записей"
          value={totalProcessed.toLocaleString('ru-RU')}
          icon={type === 'nomenclature' ? Package : Users}
          description="Всего обработано"
          trend={undefined}
        />
        
        <StatCard
          title="Групп создано"
          value={totalGroups.toLocaleString('ru-RU')}
          icon={BarChart3}
          description="Нормализованных групп"
          trend={undefined}
        />
        
        <StatCard
          title="Объединено"
          value={totalMerged.toLocaleString('ru-RU')}
          icon={TrendingUp}
          description="Дубликатов объединено"
          trend={undefined}
        />
        
        <StatCard
          title="Успешность"
          value={`${successRate.toFixed(1)}%`}
          icon={TrendingUp}
          description={`Ошибок: ${errorRate.toFixed(1)}%`}
          trend={successRate >= 95 ? { value: successRate, label: 'отлично', isPositive: true } : successRate >= 80 ? { value: successRate, label: 'хорошо', isPositive: true } : { value: errorRate, label: 'требует внимания', isPositive: false }}
        />
      </div>
    </div>
  )
}

