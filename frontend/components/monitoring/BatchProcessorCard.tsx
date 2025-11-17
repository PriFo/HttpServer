'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Layers, TrendingDown, Package, Zap } from 'lucide-react'

interface BatchProcessorStats {
  enabled: boolean
  queue_size: number
  total_batches: number
  avg_items_per_batch: number
  api_calls_saved: number
  last_batch_time?: string
}

interface BatchProcessorCardProps {
  data: BatchProcessorStats
}

export function BatchProcessorCard({ data }: BatchProcessorCardProps) {
  const formatNumber = (num: number) => {
    if (num >= 1000000) {
      return `${(num / 1000000).toFixed(2)}M`
    } else if (num >= 1000) {
      return `${(num / 1000).toFixed(2)}K`
    }
    return num.toString()
  }

  const getQueueStatus = (queueSize: number): {
    variant: 'default' | 'destructive' | 'outline' | 'secondary'
    label: string
    className?: string
  } => {
    if (queueSize === 0) return { variant: 'default', label: 'Пусто', className: 'bg-green-500 hover:bg-green-600' }
    if (queueSize < 10) return { variant: 'secondary', label: 'Низкая' }
    if (queueSize < 50) return { variant: 'default', label: 'Средняя', className: 'bg-yellow-500 hover:bg-yellow-600' }
    return { variant: 'destructive', label: 'Высокая' }
  }

  if (!data.enabled) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Layers className="h-5 w-5" />
            Batch Processor
          </CardTitle>
          <CardDescription>Пакетная обработка AI запросов</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-4">
            <Badge variant="secondary">Отключен</Badge>
            <p className="text-sm text-muted-foreground mt-2">
              Пакетная обработка не активирована
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  const queueStatus = getQueueStatus(data.queue_size)
  const savingsPercent = data.total_batches > 0
    ? ((data.api_calls_saved / (data.api_calls_saved + data.total_batches)) * 100).toFixed(1)
    : '0.0'

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Layers className="h-5 w-5" />
            Batch Processor
          </div>
          <Badge variant="outline" className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-full bg-green-500"></div>
            Активен
          </Badge>
        </CardTitle>
        <CardDescription>Эффективная пакетная обработка AI запросов</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Очередь</p>
            <div className="flex items-baseline gap-2">
              <p className="text-2xl font-bold">{data.queue_size}</p>
              <Badge
                variant={queueStatus.variant}
                className={`text-xs ${queueStatus.className || ''}`}
              >
                {queueStatus.label}
              </Badge>
            </div>
          </div>

          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Всего батчей</p>
            <p className="text-2xl font-bold">{formatNumber(data.total_batches)}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 pt-2 border-t">
          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <Package className="h-4 w-4 text-muted-foreground" />
              <p className="text-sm font-medium text-muted-foreground">Ср. размер батча</p>
            </div>
            <p className="text-xl font-bold">{data.avg_items_per_batch.toFixed(1)}</p>
          </div>

          <div className="space-y-1">
            <div className="flex items-center gap-1">
              <TrendingDown className="h-4 w-4 text-green-500" />
              <p className="text-sm font-medium text-muted-foreground">Экономия API</p>
            </div>
            <p className="text-xl font-bold text-green-500">
              {formatNumber(data.api_calls_saved)}
            </p>
            <p className="text-xs text-muted-foreground">~{savingsPercent}%</p>
          </div>
        </div>

        {data.last_batch_time && (
          <div className="pt-2 border-t">
            <p className="text-xs text-muted-foreground">
              Последний батч: {new Date(data.last_batch_time).toLocaleString('ru-RU')}
            </p>
          </div>
        )}

        <div className="pt-2">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Zap className="h-4 w-4" />
            <span>
              Группирует до {data.avg_items_per_batch.toFixed(0)} запросов в один батч
            </span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
