'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts'
import { AlertCircle } from 'lucide-react'

interface ErrorBreakdown {
  quota_exceeded: number
  rate_limit: number
  timeout: number
  network: number
  auth: number
  other: number
}

interface BenchmarkErrorBreakdownProps {
  errorBreakdown?: ErrorBreakdown
  totalErrors: number
  modelName: string
}

const ERROR_COLORS = {
  quota_exceeded: '#ef4444', // red
  rate_limit: '#f59e0b',    // amber
  timeout: '#eab308',        // yellow
  network: '#3b82f6',        // blue
  auth: '#8b5cf6',           // purple
  other: '#6b7280',          // gray
}

const ERROR_LABELS = {
  quota_exceeded: 'Quota Exceeded',
  rate_limit: 'Rate Limit',
  timeout: 'Timeout',
  network: 'Network',
  auth: 'Auth',
  other: 'Other',
}

export function BenchmarkErrorBreakdown({ errorBreakdown, totalErrors, modelName }: BenchmarkErrorBreakdownProps) {
  const chartData = useMemo(() => {
    if (!errorBreakdown || totalErrors === 0) return []

    const data = Object.entries(errorBreakdown)
      .filter(([_, count]) => count > 0)
      .map(([type, count]) => ({
        name: ERROR_LABELS[type as keyof typeof ERROR_LABELS],
        value: count,
        percentage: ((count / totalErrors) * 100).toFixed(1),
        color: ERROR_COLORS[type as keyof typeof ERROR_COLORS],
      }))
      .sort((a, b) => b.value - a.value)

    return data
  }, [errorBreakdown, totalErrors])

  if (!errorBreakdown || totalErrors === 0 || chartData.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-sm">
            <AlertCircle className="h-4 w-4" />
            Детальная статистика ошибок
          </CardTitle>
          <CardDescription>
            Нет ошибок для отображения
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <AlertCircle className="h-4 w-4" />
          Детальная статистика ошибок
        </CardTitle>
        <CardDescription>
          Распределение {totalErrors} ошибок по типам для модели {modelName}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Круговая диаграмма */}
          <div className="h-[250px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={chartData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, percent }) => `${name}: ${percent ? (percent * 100).toFixed(1) : '0'}%`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {chartData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip
                  content={({ active, payload }) => {
                    if (!active || !payload || payload.length === 0) return null
                    const data = payload[0]
                    return (
                      <div className="bg-background border rounded-lg shadow-lg p-3">
                        <p className="font-semibold text-sm">{data.name}</p>
                        <p className="text-xs text-muted-foreground">
                          Количество: {data.value}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          Процент: {data.payload.percentage}%
                        </p>
                      </div>
                    )
                  }}
                />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </div>

          {/* Легенда с деталями */}
          <div className="space-y-2">
            <h4 className="text-sm font-semibold">Детализация ошибок:</h4>
            <div className="space-y-2">
              {chartData.map((item, index) => (
                <div key={index} className="flex items-center justify-between p-2 border rounded-lg">
                  <div className="flex items-center gap-2">
                    <div 
                      className="w-4 h-4 rounded" 
                      style={{ backgroundColor: item.color }}
                    />
                    <span className="text-sm">{item.name}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">
                      {item.value}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {item.percentage}%
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

