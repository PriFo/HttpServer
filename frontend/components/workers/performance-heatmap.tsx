'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { TrendingUp, Zap, Activity } from 'lucide-react'
import { cn } from '@/lib/utils'

interface BenchmarkResult {
  model: string
  priority: number
  speed: number
  avg_response_time_ms: number
  success_count: number
  error_count: number
  total_requests: number
  success_rate: number
  status: string
}

interface PerformanceHeatmapProps {
  results: BenchmarkResult[]
  metric?: 'speed' | 'success_rate' | 'response_time'
}

// Функция для получения цвета на основе значения
function getColor(value: number, min: number, max: number, metric: string): string {
  if (max === min) return 'bg-gray-300'
  
  const normalized = (value - min) / (max - min)
  
  if (metric === 'response_time') {
    // Для времени отклика: меньше = лучше (зеленый), больше = хуже (красный)
    if (normalized < 0.2) return 'bg-green-500'
    if (normalized < 0.4) return 'bg-green-400'
    if (normalized < 0.6) return 'bg-yellow-400'
    if (normalized < 0.8) return 'bg-orange-400'
    return 'bg-red-500'
  } else {
    // Для скорости и успешности: больше = лучше (зеленый), меньше = хуже (красный)
    if (normalized >= 0.8) return 'bg-green-500'
    if (normalized >= 0.6) return 'bg-green-400'
    if (normalized >= 0.4) return 'bg-yellow-400'
    if (normalized >= 0.2) return 'bg-orange-400'
    return 'bg-red-500'
  }
}

// Функция для вычисления размера ячейки на основе количества успешных запросов
function getSize(successCount: number, maxSuccessCount: number): string {
  if (maxSuccessCount === 0) return 'w-8 h-8'
  
  const ratio = successCount / maxSuccessCount
  if (ratio >= 0.8) return 'w-12 h-12'
  if (ratio >= 0.6) return 'w-10 h-10'
  if (ratio >= 0.4) return 'w-8 h-8'
  if (ratio >= 0.2) return 'w-6 h-6'
  return 'w-4 h-4'
}

export function PerformanceHeatmap({ results, metric = 'speed' }: PerformanceHeatmapProps) {
  const heatmapData = useMemo(() => {
    if (!results || results.length === 0) return []

    // Группируем модели по провайдерам (если есть в названии)
    const grouped = results.reduce((acc, result) => {
      const provider = result.model.includes('/') 
        ? result.model.split('/')[0] 
        : result.model.includes('-') 
          ? result.model.split('-')[0]
          : 'Other'
      
      if (!acc[provider]) {
        acc[provider] = []
      }
      acc[provider].push(result)
      return acc
    }, {} as Record<string, BenchmarkResult[]>)

    return Object.entries(grouped).map(([provider, models]) => ({
      provider,
      models: models.sort((a, b) => {
        if (metric === 'speed') return (b.speed || 0) - (a.speed || 0)
        if (metric === 'success_rate') return (b.success_rate || 0) - (a.success_rate || 0)
        return (a.avg_response_time_ms || 0) - (b.avg_response_time_ms || 0)
      }),
    }))
  }, [results, metric])

  const metricValues = useMemo(() => {
    if (!results || results.length === 0) return { min: 0, max: 0 }
    
    const values = results.map(r => {
      if (metric === 'speed') return r.speed || 0
      if (metric === 'success_rate') return r.success_rate || 0
      return r.avg_response_time_ms || 0
    })
    
    return {
      min: Math.min(...values),
      max: Math.max(...values),
    }
  }, [results, metric])

  const maxSuccessCount = useMemo(() => {
    if (!results || results.length === 0) return 0
    return Math.max(...results.map(r => r.success_count || 0))
  }, [results])

  if (heatmapData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Тепловая карта производительности
          </CardTitle>
          <CardDescription>
            Недостаточно данных для отображения
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const metricLabels = {
    speed: 'Скорость (req/s)',
    success_rate: 'Успешность (%)',
    response_time: 'Время отклика (мс)',
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Тепловая карта производительности
            </CardTitle>
            <CardDescription>
              Цвет: {metricLabels[metric]}, Размер: количество успешных запросов
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Badge variant="outline" className="text-xs">
              {results.length} моделей
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          {heatmapData.map(({ provider, models }) => (
            <div key={provider} className="space-y-2">
              <h4 className="text-sm font-semibold text-muted-foreground">{provider}</h4>
              <div className="flex flex-wrap gap-2">
                {models.map((result) => {
                  const value = metric === 'speed' 
                    ? result.speed || 0
                    : metric === 'success_rate'
                      ? result.success_rate || 0
                      : result.avg_response_time_ms || 0
                  
                  const color = getColor(value, metricValues.min, metricValues.max, metric)
                  const size = getSize(result.success_count || 0, maxSuccessCount)
                  
                  return (
                    <TooltipProvider key={result.model}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div
                            className={cn(
                              'rounded-md flex items-center justify-center cursor-pointer transition-all hover:scale-110 hover:shadow-lg',
                              color,
                              size,
                              'text-white text-xs font-medium'
                            )}
                            style={{ minWidth: size, minHeight: size }}
                          >
                            {result.model.length > 10 
                              ? result.model.substring(0, 7) + '...'
                              : result.model.substring(0, 10)}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="max-w-xs">
                          <div className="space-y-1">
                            <p className="font-semibold">{result.model}</p>
                            <div className="text-xs space-y-0.5">
                              <p>Скорость: {result.speed?.toFixed(2) || 0} req/s</p>
                              <p>Успешность: {result.success_rate?.toFixed(1) || 0}%</p>
                              <p>Время отклика: {(result.avg_response_time_ms / 1000)?.toFixed(2) || 0}s</p>
                              <p>Успешных: {result.success_count || 0}</p>
                              <p>Ошибок: {result.error_count || 0}</p>
                              <p>Приоритет: {result.priority || '-'}</p>
                            </div>
                          </div>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  )
                })}
              </div>
            </div>
          ))}
        </div>

        {/* Легенда */}
        <div className="mt-6 pt-4 border-t space-y-3">
          <div className="flex items-center gap-4 text-xs">
            <span className="font-semibold">Цвет ({metricLabels[metric]}):</span>
            <div className="flex items-center gap-1">
              <div className="w-4 h-4 bg-red-500 rounded" />
              <span>Низкий</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-4 h-4 bg-yellow-400 rounded" />
              <span>Средний</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="w-4 h-4 bg-green-500 rounded" />
              <span>Высокий</span>
            </div>
          </div>
          <div className="flex items-center gap-4 text-xs">
            <span className="font-semibold">Размер (успешные запросы):</span>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-gray-400 rounded" />
              <span>Мало</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 bg-gray-400 rounded" />
              <span>Средне</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-12 h-12 bg-gray-400 rounded" />
              <span>Много</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

