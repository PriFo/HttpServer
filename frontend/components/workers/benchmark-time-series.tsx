'use client'

import { useState, useEffect, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { DynamicLineChart, DynamicLine, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from '@/lib/recharts-dynamic'
import { TrendingUp, Clock, RefreshCw, Zap, CheckCircle2, AlertCircle, Target, Activity } from 'lucide-react'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale/ru'

interface BenchmarkHistoryPoint {
  timestamp: string
  model_name: string
  speed: number
  success_rate: number
  avg_response_time_ms: number
  success_count: number
  error_count: number
  total_requests: number
  avg_confidence?: number
  coefficient_of_variation?: number
  avg_retries?: number
}

interface BenchmarkTimeSeriesProps {
  provider?: string
  selectedModels?: string[]
  timeRange?: '24h' | '7d' | '30d' | 'all'
}

export function BenchmarkTimeSeries({ provider, selectedModels, timeRange = '7d' }: BenchmarkTimeSeriesProps) {
  const [history, setHistory] = useState<BenchmarkHistoryPoint[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [metric, setMetric] = useState<'speed' | 'success_rate' | 'response_time' | 'confidence' | 'stability'>('speed')
  const [currentTimeRange, setCurrentTimeRange] = useState(timeRange)

  useEffect(() => {
    fetchHistory()
  }, [provider, currentTimeRange])

  const fetchHistory = async () => {
    setLoading(true)
    setError(null)
    
    try {
      const params = new URLSearchParams({
        history: 'true',
        limit: '500',
      })
      
      if (provider) {
        // Фильтруем по провайдеру, если указан
        // Пока просто получаем все, фильтрация будет на фронтенде
      }

      const response = await fetch(`/api/models/benchmark?${params}`)
      
      if (!response.ok) {
        throw new Error('Не удалось загрузить историю')
      }

      const data = await response.json()
      const historyData = data.history || []
      
      // Фильтруем по времени
      const now = new Date()
      const filtered = historyData.filter((point: BenchmarkHistoryPoint) => {
        const pointTime = new Date(point.timestamp)
        const diff = now.getTime() - pointTime.getTime()
        
        switch (currentTimeRange) {
          case '24h':
            return diff <= 24 * 60 * 60 * 1000
          case '7d':
            return diff <= 7 * 24 * 60 * 60 * 1000
          case '30d':
            return diff <= 30 * 24 * 60 * 60 * 1000
          default:
            return true
        }
      })
      
      setHistory(filtered)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки истории')
      console.error('[BenchmarkTimeSeries] Error fetching history:', err)
    } finally {
      setLoading(false)
    }
  }

  const chartData = useMemo(() => {
    if (!history || history.length === 0) return []

    // Группируем по времени и моделям
    const timeMap = new Map<string, Map<string, BenchmarkHistoryPoint>>()
    
    history.forEach(point => {
      const timeKey = new Date(point.timestamp).toISOString()
      if (!timeMap.has(timeKey)) {
        timeMap.set(timeKey, new Map())
      }
      timeMap.get(timeKey)!.set(point.model_name, point)
    })

    // Получаем уникальные модели
    const allModels = Array.from(new Set(history.map(p => p.model_name)))
    let modelsToShow = allModels
    
    if (selectedModels && selectedModels.length > 0) {
      modelsToShow = allModels.filter(m => selectedModels.includes(m))
    } else {
      // Берем топ-5 моделей по последним результатам
      const latestTime = Array.from(timeMap.keys()).sort().reverse()[0]
      if (latestTime) {
        const latestPoints = Array.from(timeMap.get(latestTime)!.values())
        modelsToShow = latestPoints
          .sort((a, b) => {
            const scoreA = (a.success_rate || 0) * 0.7 + (a.speed || 0) * 0.3
            const scoreB = (b.success_rate || 0) * 0.7 + (b.speed || 0) * 0.3
            return scoreB - scoreA
          })
          .slice(0, 5)
          .map(p => p.model_name)
      }
    }

    // Создаем данные для графика
    const sortedTimes = Array.from(timeMap.keys()).sort()
    
    return sortedTimes.map(timeKey => {
      const dataPoint: Record<string, string | number | null> = {
        timestamp: timeKey,
        time: format(new Date(timeKey), 'dd.MM HH:mm', { locale: ru }),
      }
      
      const pointsAtTime = timeMap.get(timeKey)!
      modelsToShow.forEach(modelName => {
        const point = pointsAtTime.get(modelName)
        if (point) {
          if (metric === 'speed') {
            dataPoint[modelName] = point.speed || 0
          } else if (metric === 'success_rate') {
            dataPoint[modelName] = point.success_rate || 0
          } else if (metric === 'response_time') {
            dataPoint[modelName] = (point.avg_response_time_ms || 0) / 1000 // в секундах
          } else if (metric === 'confidence') {
            dataPoint[modelName] = point.avg_confidence !== undefined 
              ? point.avg_confidence * 100 // конвертируем в проценты
              : null
          } else if (metric === 'stability') {
            // Коэффициент вариации (меньше = лучше)
            dataPoint[modelName] = point.coefficient_of_variation !== undefined
              ? point.coefficient_of_variation
              : null
          } else {
            dataPoint[modelName] = (point.avg_response_time_ms || 0) / 1000 // в секундах
          }
        } else {
          dataPoint[modelName] = null
        }
      })
      
      return dataPoint
    })
  }, [history, selectedModels, metric])

  const colors = [
    '#3b82f6', // blue
    '#22c55e', // green
    '#ef4444', // red
    '#f59e0b', // amber
    '#8b5cf6', // purple
    '#ec4899', // pink
    '#06b6d4', // cyan
    '#84cc16', // lime
  ]

  const metricLabels = {
    speed: 'Скорость (req/s)',
    success_rate: 'Успешность (%)',
    response_time: 'Время отклика (с)',
    confidence: 'Уверенность (%)',
    stability: 'Коэф. вариации',
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            Временные ряды производительности
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[400px] w-full" />
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            Временные ряды производительности
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center h-[400px] text-center space-y-4">
            <AlertCircle className="h-12 w-12 text-destructive" />
            <p className="text-muted-foreground">{error}</p>
            <Button onClick={fetchHistory} variant="outline">
              <RefreshCw className="h-4 w-4 mr-2" />
              Попробовать снова
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (chartData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            Временные ряды производительности
          </CardTitle>
          <CardDescription>
            Нет данных для отображения. Запустите бенчмарк для получения истории.
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  // Получаем уникальные модели из данных
  const modelsInData = useMemo(() => {
    if (chartData.length === 0) return []
    const firstPoint = chartData[0]
    return Object.keys(firstPoint).filter(key => key !== 'timestamp' && key !== 'time')
  }, [chartData])

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5" />
              Временные ряды производительности
            </CardTitle>
            <CardDescription>
              Изменение метрик моделей во времени
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Select value={currentTimeRange} onValueChange={(v) => setCurrentTimeRange(v as typeof currentTimeRange)}>
              <SelectTrigger className="w-[120px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="24h">24 часа</SelectItem>
                <SelectItem value="7d">7 дней</SelectItem>
                <SelectItem value="30d">30 дней</SelectItem>
                <SelectItem value="all">Все время</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" size="sm" onClick={fetchHistory}>
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Выбор метрики */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">Метрика:</span>
          <Select value={metric} onValueChange={(v) => setMetric(v as typeof metric)}>
            <SelectTrigger className="w-[200px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="speed">
                <div className="flex items-center gap-2">
                  <Zap className="h-4 w-4" />
                  Скорость
                </div>
              </SelectItem>
              <SelectItem value="success_rate">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4" />
                  Успешность
                </div>
              </SelectItem>
              <SelectItem value="response_time">
                <div className="flex items-center gap-2">
                  <Clock className="h-4 w-4" />
                  Время отклика
                </div>
              </SelectItem>
              <SelectItem value="confidence">
                <div className="flex items-center gap-2">
                  <Target className="h-4 w-4" />
                  Уверенность
                </div>
              </SelectItem>
              <SelectItem value="stability">
                <div className="flex items-center gap-2">
                  <Activity className="h-4 w-4" />
                  Стабильность
                </div>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* График */}
        <div className="h-[400px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <DynamicLineChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis 
                dataKey="time" 
                tick={{ fill: '#6b7280', fontSize: 12 }}
                angle={-45}
                textAnchor="end"
                height={80}
              />
              <YAxis 
                tick={{ fill: '#6b7280', fontSize: 12 }}
                label={{ 
                  value: metricLabels[metric], 
                  angle: -90, 
                  position: 'insideLeft',
                  style: { textAnchor: 'middle', fill: '#6b7280' }
                }}
              />
              <Tooltip
                content={({ active, payload, label }) => {
                  if (!active || !payload || payload.length === 0) return null
                  
                  return (
                    <div className="bg-background border rounded-lg shadow-lg p-3 space-y-1">
                      <p className="font-semibold text-sm mb-2">{label}</p>
                      {payload.map((entry, index) => (
                        <p key={index} className="text-xs" style={{ color: entry.color }}>
                          {entry.name}: {typeof entry.value === 'number' ? entry.value.toFixed(2) : entry.value} {metric === 'response_time' ? 'с' : metric === 'success_rate' ? '%' : 'req/s'}
                        </p>
                      ))}
                    </div>
                  )
                }}
              />
              <Legend 
                wrapperStyle={{ paddingTop: '20px' }}
                iconType="line"
              />
              {modelsInData.map((modelName, index) => (
                <DynamicLine
                  key={modelName}
                  type="monotone"
                  dataKey={modelName}
                  stroke={colors[index % colors.length]}
                  strokeWidth={2}
                  dot={{ r: 3 }}
                  name={modelName.length > 30 ? modelName.substring(0, 27) + '...' : modelName}
                  connectNulls
                />
              ))}
            </DynamicLineChart>
          </ResponsiveContainer>
        </div>

        {/* Статистика */}
        {chartData.length > 0 && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-4 border-t">
            <div className="text-center">
              <div className="text-2xl font-bold">{chartData.length}</div>
              <div className="text-xs text-muted-foreground">Точек данных</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{modelsInData.length}</div>
              <div className="text-xs text-muted-foreground">Моделей</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">
                {chartData[0]?.timestamp ? format(new Date(chartData[0].timestamp as string), 'dd.MM', { locale: ru }) : '-'}
              </div>
              <div className="text-xs text-muted-foreground">Начало</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">
                {chartData[chartData.length - 1]?.timestamp ? format(new Date(chartData[chartData.length - 1].timestamp as string), 'dd.MM', { locale: ru }) : '-'}
              </div>
              <div className="text-xs text-muted-foreground">Конец</div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

