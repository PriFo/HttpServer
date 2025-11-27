'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { BarChart3, PieChart } from 'lucide-react'

interface GostStatisticsChartProps {
  statistics: {
    by_status?: Record<string, number>
    by_source_type?: Record<string, number>
    total_gosts?: number
  }
}

export function GostStatisticsChart({ statistics }: GostStatisticsChartProps) {
  // Безопасная инициализация с проверкой типа
  const statusData = (statistics?.by_status && typeof statistics.by_status === 'object' && !Array.isArray(statistics.by_status))
    ? statistics.by_status
    : {}
  const sourceTypeData = (statistics?.by_source_type && typeof statistics.by_source_type === 'object' && !Array.isArray(statistics.by_source_type))
    ? statistics.by_source_type
    : {}
  const total = statistics?.total_gosts || 0

  // Получаем топ-10 источников
  const topSources = Object.entries(sourceTypeData)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 10)

  // Получаем все статусы
  const statusEntries = Object.entries(statusData)

  const getStatusColor = (status: string) => {
    const statusLower = status.toLowerCase()
    if (statusLower.includes('действующий') || statusLower.includes('действует')) {
      return 'bg-green-500'
    }
    if (statusLower.includes('отменен') || statusLower.includes('отменён')) {
      return 'bg-red-500'
    }
    if (statusLower.includes('заменен') || statusLower.includes('заменён')) {
      return 'bg-yellow-500'
    }
    return 'bg-gray-500'
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      {/* Распределение по статусам */}
      {statusEntries.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <PieChart className="h-5 w-5" />
              Распределение по статусам
            </CardTitle>
            <CardDescription>
              Количество ГОСТов по статусам
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {statusEntries.map(([status, count]) => {
                const percentage = total > 0 ? (count / total) * 100 : 0
                return (
                  <div key={status} className="space-y-1">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">{status}</span>
                      <span className="text-muted-foreground">
                        {count.toLocaleString('ru-RU')} ({percentage.toFixed(1)}%)
                      </span>
                    </div>
                    <div className="w-full bg-muted rounded-full h-2">
                      <div
                        className={`${getStatusColor(status)} h-2 rounded-full transition-all duration-500`}
                        style={{ width: `${percentage}%` }}
                      />
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Топ источников */}
      {topSources.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5" />
              Топ источников
            </CardTitle>
            <CardDescription>
              Топ-10 источников по количеству ГОСТов
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {topSources.map(([sourceType, count], index) => {
                const maxCount = topSources[0]?.[1] || 1
                const percentage = (count / maxCount) * 100
                return (
                  <div key={sourceType} className="space-y-1">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium truncate flex-1 mr-2">
                        {index + 1}. {sourceType}
                      </span>
                      <span className="text-muted-foreground shrink-0">
                        {count.toLocaleString('ru-RU')}
                      </span>
                    </div>
                    <div className="w-full bg-muted rounded-full h-2">
                      <div
                        className="bg-blue-500 h-2 rounded-full transition-all duration-500"
                        style={{ width: `${percentage}%` }}
                      />
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

