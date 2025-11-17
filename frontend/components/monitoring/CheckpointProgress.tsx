'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Save, CheckCircle, Clock, Play } from 'lucide-react'

interface CheckpointStatus {
  enabled: boolean
  active: boolean
  processed_count: number
  total_count: number
  progress_percent: number
  last_checkpoint_time?: string
  current_batch_id?: string
}

interface CheckpointProgressProps {
  data: CheckpointStatus
}

export function CheckpointProgress({ data }: CheckpointProgressProps) {
  const formatNumber = (num: number) => {
    if (num >= 1000000) {
      return `${(num / 1000000).toFixed(2)}M`
    } else if (num >= 1000) {
      return `${(num / 1000).toFixed(2)}K`
    }
    return num.toString()
  }

  const getProgressColor = (percent: number) => {
    if (percent >= 100) return 'text-green-500'
    if (percent >= 75) return 'text-blue-500'
    if (percent >= 50) return 'text-yellow-500'
    return 'text-gray-500'
  }

  if (!data.enabled) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Save className="h-5 w-5" />
            Checkpoint System
          </CardTitle>
          <CardDescription>Контрольные точки при длительной обработке</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-4">
            <Badge variant="secondary">Отключен</Badge>
            <p className="text-sm text-muted-foreground mt-2">
              Система checkpoint не активирована
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={data.active ? 'border-blue-500' : ''}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Save className="h-5 w-5" />
            Checkpoint System
          </div>
          <Badge variant={data.active ? 'default' : 'secondary'} className="flex items-center gap-1">
            {data.active ? (
              <>
                <Play className="h-3 w-3" />
                Активна обработка
              </>
            ) : (
              <>
                <CheckCircle className="h-3 w-3" />
                Ожидание
              </>
            )}
          </Badge>
        </CardTitle>
        <CardDescription>
          {data.active
            ? 'Выполняется нормализация с сохранением прогресса'
            : 'Контрольные точки для восстановления после сбоев'}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {data.active ? (
          <>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium">Прогресс обработки</span>
                <span className={`font-bold ${getProgressColor(data.progress_percent)}`}>
                  {data.progress_percent.toFixed(1)}%
                </span>
              </div>
              <Progress value={data.progress_percent} className="h-3" />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1">
                <p className="text-sm font-medium text-muted-foreground">Обработано</p>
                <p className="text-2xl font-bold">{formatNumber(data.processed_count)}</p>
              </div>
              <div className="space-y-1">
                <p className="text-sm font-medium text-muted-foreground">Всего</p>
                <p className="text-2xl font-bold">{formatNumber(data.total_count)}</p>
              </div>
            </div>

            {data.current_batch_id && (
              <div className="pt-2 border-t">
                <p className="text-xs text-muted-foreground">
                  Текущий батч: <code className="font-mono">{data.current_batch_id}</code>
                </p>
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-4 space-y-2">
            <CheckCircle className="h-12 w-12 mx-auto text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Нет активных процессов нормализации
            </p>
            {data.last_checkpoint_time && (
              <p className="text-xs text-muted-foreground">
                Последний checkpoint: {new Date(data.last_checkpoint_time).toLocaleString('ru-RU')}
              </p>
            )}
          </div>
        )}

        {data.last_checkpoint_time && data.active && (
          <div className="pt-2 border-t">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Clock className="h-3 w-3" />
              <span>
                Последнее сохранение: {new Date(data.last_checkpoint_time).toLocaleString('ru-RU')}
              </span>
            </div>
          </div>
        )}

        <div className="pt-2">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Save className="h-4 w-4" />
            <span>Автоматическое сохранение прогресса каждые 100 записей</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
