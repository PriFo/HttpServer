'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import dynamic from 'next/dynamic'
import { Skeleton } from '@/components/ui/skeleton'
import { Radar, RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, ResponsiveContainer, Tooltip, Legend } from 'recharts'
import { TrendingUp, Zap, Shield, Award, DollarSign } from 'lucide-react'

// Динамическая загрузка для уменьшения размера бандла
const DynamicRadarChart = dynamic(
  () => import('recharts').then((mod) => mod.RadarChart),
  { ssr: false, loading: () => <Skeleton className="h-[400px] w-full" /> }
)

const DynamicRadar = dynamic(
  () => import('recharts').then((mod) => mod.Radar),
  { ssr: false }
)

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
  avg_confidence?: number
  coefficient_of_variation?: number
  avg_retries?: number
}

interface BenchmarkRadarChartProps {
  results: BenchmarkResult[]
  selectedModels?: string[]
  maxModels?: number
}

// Нормализация значений для радиальной диаграммы (0-100)
function normalizeValue(value: number, min: number, max: number): number {
  if (max === min) return 50
  return Math.max(0, Math.min(100, ((value - min) / (max - min)) * 100))
}

// Вычисление метрик для модели
function calculateMetrics(result: BenchmarkResult, allResults: BenchmarkResult[]) {
  // Speed: нормализуем скорость (req/s)
  const speeds = allResults.map(r => r.speed || 0)
  const speedScore = normalizeValue(result.speed || 0, Math.min(...speeds), Math.max(...speeds))

  // Reliability: успешность запросов
  const reliabilityScore = result.success_rate || 0

  // Quality: используем avg_confidence если доступно, иначе обратная зависимость от ошибок
  let qualityScore = 0
  if (result.avg_confidence !== undefined && result.avg_confidence !== null) {
    // Используем уверенность напрямую (0-1 -> 0-100)
    qualityScore = result.avg_confidence * 100
  } else {
    // Fallback: обратная зависимость от ошибок
    const errorRates = allResults.map(r => (r.error_count / (r.total_requests || 1)) * 100)
    const maxErrorRate = Math.max(...errorRates, 1)
    const errorRate = (result.error_count / (result.total_requests || 1)) * 100
    qualityScore = 100 - normalizeValue(errorRate, 0, maxErrorRate)
  }

  // Cost: обратная зависимость от приоритета (меньше приоритет = выше стоимость в контексте доступности)
  const priorities = allResults.map(r => r.priority || 999)
  const minPriority = Math.min(...priorities)
  const maxPriority = Math.max(...priorities)
  const costScore = normalizeValue(result.priority || 999, minPriority, maxPriority)

  // Stability: используем coefficient_of_variation если доступно, иначе на основе success_rate
  let stabilityScore = 0
  if (result.coefficient_of_variation !== undefined && result.coefficient_of_variation !== null) {
    // Коэффициент вариации: меньше = лучше, инвертируем для визуализации
    const coeffVars = allResults
      .map(r => r.coefficient_of_variation)
      .filter((v): v is number => v !== undefined && v !== null)
    if (coeffVars.length > 0) {
      const maxCoeffVar = Math.max(...coeffVars, 0.01)
      // Инвертируем: меньше вариация = выше стабильность
      stabilityScore = 100 - normalizeValue(result.coefficient_of_variation, 0, maxCoeffVar) * 100
    } else {
      // Fallback: на основе success_rate
      stabilityScore = result.success_rate || 0
    }
  } else {
    // Fallback: на основе success_rate
    stabilityScore = result.success_rate || 0
  }

  return {
    model: result.model,
    Speed: Math.round(speedScore),
    Reliability: Math.round(reliabilityScore),
    Quality: Math.round(qualityScore),
    Cost: Math.round(costScore),
    Stability: Math.round(stabilityScore),
    success_rate: result.success_rate,
    speed: result.speed,
    confidence: result.avg_confidence,
    coeff_variation: result.coefficient_of_variation,
  }
}

export function BenchmarkRadarChart({ results, selectedModels, maxModels = 5 }: BenchmarkRadarChartProps) {
  const { chartData, modelNames } = useMemo(() => {
    if (!results || results.length === 0) return { chartData: [], modelNames: [] }

    // Фильтруем результаты по выбранным моделям или берем топ по success_rate
    let filteredResults = results
    
    if (selectedModels && selectedModels.length > 0) {
      filteredResults = results.filter(r => selectedModels.includes(r.model))
    } else {
      // Берем топ модели по success_rate и speed
      filteredResults = [...results]
        .sort((a, b) => {
          const scoreA = (a.success_rate || 0) * 0.7 + (a.speed || 0) * 0.3
          const scoreB = (b.success_rate || 0) * 0.7 + (b.speed || 0) * 0.3
          return scoreB - scoreA
        })
        .slice(0, maxModels)
    }

    const metrics = filteredResults.map(result => calculateMetrics(result, results))
    const names = metrics.map(m => m.model)
    
    // Преобразуем в формат для Recharts: массив объектов с метриками
    const categories = ['Speed', 'Reliability', 'Quality', 'Cost', 'Stability']
    const data = categories.map(category => {
      const dataPoint: Record<string, string | number> = { category }
      metrics.forEach(metric => {
        dataPoint[metric.model] = metric[category as keyof typeof metric] as number
        // Сохраняем дополнительные данные для tooltip
        if (category === 'Speed') {
          dataPoint[`${metric.model}_success_rate`] = metric.success_rate
          dataPoint[`${metric.model}_speed`] = metric.speed
          if (metric.confidence !== undefined) {
            dataPoint[`${metric.model}_confidence`] = metric.confidence
          }
          if (metric.coeff_variation !== undefined) {
            dataPoint[`${metric.model}_coeff_variation`] = metric.coeff_variation
          }
        }
      })
      return dataPoint
    })
    
    return { chartData: data, modelNames: names }
  }, [results, selectedModels, maxModels])

  if (chartData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            Радарная диаграмма производительности
          </CardTitle>
          <CardDescription>
            Недостаточно данных для отображения
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const colors = [
    '#3b82f6', // blue
    '#22c55e', // green
    '#ef4444', // red
    '#f59e0b', // amber
    '#8b5cf6', // purple
    '#ec4899', // pink
    '#06b6d4', // cyan
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5" />
          Радарная диаграмма производительности
        </CardTitle>
        <CardDescription>
          Сравнение метрик моделей: Скорость, Надежность, Качество, Стоимость, Стабильность
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="h-[400px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <RadarChart data={chartData}>
              <PolarGrid stroke="#e5e7eb" />
              <PolarAngleAxis 
                dataKey="category" 
                tick={{ fill: '#6b7280', fontSize: 12 }}
              />
              <PolarRadiusAxis 
                angle={90} 
                domain={[0, 100]}
                tick={{ fill: '#9ca3af', fontSize: 10 }}
              />
              <Tooltip
                content={({ active, payload }) => {
                  if (!active || !payload || payload.length === 0) return null
                  
                  const data = payload[0].payload
                  return (
                    <div className="bg-background border rounded-lg shadow-lg p-3 space-y-1">
                      <p className="font-semibold text-sm">{data.model}</p>
                      {payload.map((entry, index) => (
                        <p key={index} className="text-xs" style={{ color: entry.color }}>
                          {entry.name}: {entry.value}%
                        </p>
                      ))}
                      <div className="pt-2 mt-2 border-t text-xs text-muted-foreground">
                        {payload.length > 0 && (
                          <>
                            {data[`${payload[0].name}_success_rate`] !== undefined && (
                              <p>Успешность: {data[`${payload[0].name}_success_rate`]?.toFixed(1)}%</p>
                            )}
                            {data[`${payload[0].name}_speed`] !== undefined && (
                              <p>Скорость: {data[`${payload[0].name}_speed`]?.toFixed(2)} req/s</p>
                            )}
                            {data[`${payload[0].name}_confidence`] !== undefined && data[`${payload[0].name}_confidence`] !== null && (
                              <p>Уверенность: {((data[`${payload[0].name}_confidence`] as number) * 100).toFixed(1)}%</p>
                            )}
                            {data[`${payload[0].name}_coeff_variation`] !== undefined && data[`${payload[0].name}_coeff_variation`] !== null && (
                              <p>Коэф. вариации: {(data[`${payload[0].name}_coeff_variation`] as number).toFixed(3)}</p>
                            )}
                          </>
                        )}
                      </div>
                    </div>
                  )
                }}
              />
              <Legend 
                wrapperStyle={{ paddingTop: '20px' }}
                iconType="circle"
              />
              {modelNames.map((modelName, index) => (
                <Radar
                  key={modelName}
                  name={modelName}
                  dataKey={modelName}
                  stroke={colors[index % colors.length]}
                  fill={colors[index % colors.length]}
                  fillOpacity={0.3}
                  strokeWidth={2}
                  dot={{ r: 4 }}
                />
              ))}
            </RadarChart>
          </ResponsiveContainer>
        </div>
        
        {/* Легенда с иконками */}
        <div className="mt-4 grid grid-cols-2 md:grid-cols-5 gap-2 text-xs">
          <div className="flex items-center gap-1 text-blue-600">
            <Zap className="h-3 w-3" />
            <span>Скорость</span>
          </div>
          <div className="flex items-center gap-1 text-green-600">
            <Shield className="h-3 w-3" />
            <span>Надежность</span>
          </div>
          <div className="flex items-center gap-1 text-purple-600">
            <Award className="h-3 w-3" />
            <span>Качество</span>
          </div>
          <div className="flex items-center gap-1 text-amber-600">
            <DollarSign className="h-3 w-3" />
            <span>Стоимость</span>
          </div>
          <div className="flex items-center gap-1 text-cyan-600">
            <TrendingUp className="h-3 w-3" />
            <span>Стабильность</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

