'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { LogsPanel } from '@/components/normalization/logs-panel'
import { Play, Square, RefreshCw, Clock, Zap } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface ReclassificationProcessTabProps {
  database: string
}

interface ReclassificationStatus {
  isRunning: boolean
  progress: number
  processed: number
  total: number
  success?: number
  errors?: number
  skipped?: number
  currentStep: string
  logs: string[]
  startTime?: string
  elapsedTime?: string
  rate?: number
}

export function ReclassificationProcessTab({ database }: ReclassificationProcessTabProps) {
  const [status, setStatus] = useState<ReclassificationStatus>({
    isRunning: false,
    progress: 0,
    processed: 0,
    total: 0,
    currentStep: 'Не запущено',
    logs: [],
  })
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchStatus = useCallback(async () => {
    try {
      const response = await fetch('/api/reclassification/status', {
        cache: 'no-store',
      })
      
      if (!response.ok) {
        throw new Error('Не удалось получить статус')
      }
      
      const data = await response.json()
      setStatus(data)
      setError(null)
    } catch (err) {
      console.error('Error fetching reclassification status:', err)
      if (!status.isRunning) {
        setError('Не удалось подключиться к серверу')
      }
    }
  }, [status.isRunning])

  useEffect(() => {
    // Первоначальная загрузка
    fetchStatus()

    // Автообновление статуса каждые 2 секунды, если процесс запущен
    const interval = setInterval(() => {
      if (status.isRunning) {
        fetchStatus()
      }
    }, 2000)

    return () => clearInterval(interval)
  }, [status.isRunning, fetchStatus])

  const handleStart = async () => {
    setIsLoading(true)
    setError(null)
    
    try {
      const response = await fetch('/api/reclassification/start', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          classifier_id: 1, // КПВЭД по умолчанию
          strategy_id: 'top_priority',
          limit: 0, // Без лимита
        }),
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Неизвестная ошибка' }))
        throw new Error(errorData.error || 'Не удалось запустить переклассификацию')
      }

      // Обновляем статус после запуска
      setTimeout(() => {
        fetchStatus()
      }, 500)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка запуска переклассификации')
    } finally {
      setIsLoading(false)
    }
  }

  const handleStop = async () => {
    setIsLoading(true)
    setError(null)
    
    try {
      const response = await fetch('/api/reclassification/stop', {
        method: 'POST',
      })

      if (!response.ok) {
        throw new Error('Не удалось остановить переклассификацию')
      }

      // Обновляем статус после остановки
      setTimeout(() => {
        fetchStatus()
      }, 500)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка остановки переклассификации')
    } finally {
      setIsLoading(false)
    }
  }

  const progressPercent = status.total > 0 
    ? Math.min(100, (status.processed / status.total) * 100)
    : status.progress

  return (
    <div className="space-y-6">
      {/* Статус и управление */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Процесс переклассификации</CardTitle>
              <CardDescription>
                {database ? `База данных: ${database}` : 'Управление процессом переклассификации данных'}
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={status.isRunning ? 'default' : 'secondary'}>
                {status.isRunning ? 'Выполняется' : 'Остановлено'}
              </Badge>
              <Button
                variant="outline"
                size="icon"
                onClick={fetchStatus}
                disabled={isLoading}
              >
                <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {/* Текущий шаг */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Текущий шаг:</span>
              <span className="font-medium">{status.currentStep}</span>
            </div>
          </div>

          {/* Прогресс */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Прогресс:</span>
              <span className="font-medium">
                {status.processed.toLocaleString()} / {status.total.toLocaleString()} 
                {status.total > 0 && ` (${progressPercent.toFixed(1)}%)`}
              </span>
            </div>
            <Progress value={progressPercent} className="h-2" />
          </div>

          {/* Статистика */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {status.success !== undefined && (
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground">Успешно</div>
                <div className="text-2xl font-bold text-green-600">
                  {status.success.toLocaleString()}
                </div>
              </div>
            )}
            {status.errors !== undefined && (
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground">Ошибок</div>
                <div className="text-2xl font-bold text-red-600">
                  {status.errors.toLocaleString()}
                </div>
              </div>
            )}
            {status.skipped !== undefined && (
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground">Пропущено</div>
                <div className="text-2xl font-bold text-yellow-600">
                  {status.skipped.toLocaleString()}
                </div>
              </div>
            )}
            {status.rate && status.rate > 0 && (
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground flex items-center gap-1">
                  <Zap className="h-3 w-3" />
                  Скорость
                </div>
                <div className="text-2xl font-bold">
                  {status.rate.toFixed(1)}/сек
                </div>
              </div>
            )}
            {status.elapsedTime && (
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  Время
                </div>
                <div className="text-2xl font-bold">
                  {status.elapsedTime}
                </div>
              </div>
            )}
          </div>

          {/* Кнопки управления */}
          <div className="flex items-center gap-2 pt-4 border-t">
            {!status.isRunning ? (
              <Button
                onClick={handleStart}
                disabled={isLoading}
                className="flex items-center gap-2"
              >
                <Play className="h-4 w-4" />
                Запустить переклассификацию
              </Button>
            ) : (
              <Button
                onClick={handleStop}
                disabled={isLoading}
                variant="destructive"
                className="flex items-center gap-2"
              >
                <Square className="h-4 w-4" />
                Остановить
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Логи */}
      {status.logs && status.logs.length > 0 && (
        <LogsPanel
          logs={status.logs}
          title="Логи переклассификации"
          description="Детальная информация о процессе переклассификации"
        />
      )}
    </div>
  )
}
