'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell } from 'recharts'
import { Clock, TrendingUp } from 'lucide-react'

interface PercentilesData {
  p50?: number
  p75?: number
  p90?: number
  p95?: number
  p99?: number
  min?: number
  max?: number
  avg?: number
}

interface BenchmarkPercentilesChartProps {
  percentiles: PercentilesData
  modelName: string
}

export function BenchmarkPercentilesChart({ percentiles, modelName }: BenchmarkPercentilesChartProps) {
  const chartData = useMemo(() => {
    const data: Array<{ name: string; value: number; color: string }> = []

    if (percentiles.p50 !== undefined) {
      data.push({ name: 'P50 (медиана)', value: percentiles.p50 / 1000, color: '#22c55e' })
    }
    if (percentiles.p75 !== undefined) {
      data.push({ name: 'P75', value: percentiles.p75 / 1000, color: '#3b82f6' })
    }
    if (percentiles.p90 !== undefined) {
      data.push({ name: 'P90', value: percentiles.p90 / 1000, color: '#f59e0b' })
    }
    if (percentiles.p95 !== undefined) {
      data.push({ name: 'P95', value: percentiles.p95 / 1000, color: '#ef4444' })
    }
    if (percentiles.p99 !== undefined) {
      data.push({ name: 'P99', value: percentiles.p99 / 1000, color: '#dc2626' })
    }

    return data
  }, [percentiles])

  if (chartData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-sm">
            <Clock className="h-4 w-4" />
            Распределение времени ответа
          </CardTitle>
          <CardDescription>
            Нет данных о перцентилях
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <Clock className="h-4 w-4" />
          Распределение времени ответа
        </CardTitle>
        <CardDescription>
          Перцентили времени ответа для модели {modelName}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="h-[300px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis type="number" label={{ value: 'Время (секунды)', position: 'insideBottom', offset: -5 }} />
              <YAxis dataKey="name" type="category" width={120} />
              <Tooltip
                formatter={(value: number) => [`${value.toFixed(3)}s`, 'Время ответа']}
                contentStyle={{ backgroundColor: 'hsl(var(--background))', border: '1px solid hsl(var(--border))' }}
              />
              <Legend />
              <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                {chartData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
        
        {/* Дополнительная информация */}
        <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
          {percentiles.min !== undefined && (
            <div className="flex items-center justify-between p-2 border rounded">
              <span className="text-muted-foreground">Мин:</span>
              <span className="font-medium">{(percentiles.min / 1000).toFixed(3)}s</span>
            </div>
          )}
          {percentiles.avg !== undefined && (
            <div className="flex items-center justify-between p-2 border rounded">
              <span className="text-muted-foreground">Среднее:</span>
              <span className="font-medium">{(percentiles.avg / 1000).toFixed(3)}s</span>
            </div>
          )}
          {percentiles.max !== undefined && (
            <div className="flex items-center justify-between p-2 border rounded">
              <span className="text-muted-foreground">Макс:</span>
              <span className="font-medium">{(percentiles.max / 1000).toFixed(3)}s</span>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

