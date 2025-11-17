'use client'

import { useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { RefreshCw, Play, Square, Clock, Zap, CheckCircle2, XCircle, AlertCircle, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface ProcessStatus {
  isRunning: boolean
  progress: number
  processed: number
  total: number
  success?: number
  errors?: number
  skipped?: number
  currentStep: string
  logs: string[]
  rate?: number
  startTime?: string
  elapsedTime?: string
}

interface ProcessMonitorProps {
  title: string
  statusEndpoint: string
  startEndpoint?: string
  stopEndpoint?: string
  eventsEndpoint?: string
  onStart?: () => void
  onStop?: () => void
}

export function ProcessMonitor({
  title,
  statusEndpoint,
  startEndpoint,
  stopEndpoint,
  eventsEndpoint,
  onStart,
  onStop
}: ProcessMonitorProps) {
  const [status, setStatus] = useState<ProcessStatus>({
    isRunning: false,
    progress: 0,
    processed: 0,
    total: 0,
    currentStep: 'Не запущено',
    logs: []
  })
  const [loading, setLoading] = useState(false)
  const [estimatedTimeRemaining, setEstimatedTimeRemaining] = useState<string | null>(null)

  const fetchStatus = async () => {
    try {
      const response = await fetch(statusEndpoint)
      if (response.ok) {
        const data = await response.json()
        setStatus(prev => ({
          ...prev,
          ...data,
          // Сохраняем логи, если они не пришли в ответе
          logs: data.logs || prev.logs || []
        }))

        // Расчет оставшегося времени
        if (data.isRunning && data.processed > 0 && data.total > 0 && data.rate && data.rate > 0) {
          const remaining = data.total - data.processed
          const estimatedSeconds = Math.round(remaining / data.rate)
          
          if (estimatedSeconds < 60) {
            setEstimatedTimeRemaining(`~${estimatedSeconds} сек`)
          } else if (estimatedSeconds < 3600) {
            const minutes = Math.round(estimatedSeconds / 60)
            setEstimatedTimeRemaining(`~${minutes} мин`)
          } else {
            const hours = (estimatedSeconds / 3600).toFixed(1)
            setEstimatedTimeRemaining(`~${hours} час`)
          }
        } else if (data.isRunning && data.processed > 0 && data.total > 0) {
          // Если нет rate, но есть прогресс, пытаемся оценить по времени
          setEstimatedTimeRemaining(null)
        } else {
          setEstimatedTimeRemaining(null)
        }
      } else {
        // Если статус недоступен, но процесс был запущен - показываем предупреждение
        setStatus(prev => ({
          ...prev,
          currentStep: prev.isRunning ? 'Ошибка получения статуса' : prev.currentStep
        }))
      }
    } catch (error) {
      console.error('Error fetching status:', error)
      // Не сбрасываем статус при ошибке сети, чтобы не потерять информацию
    }
  }

  useEffect(() => {
    fetchStatus()
    const interval = setInterval(() => {
      fetchStatus()
    }, 2000) // Обновление каждые 2 секунды
    return () => clearInterval(interval)
  }, [statusEndpoint, fetchStatus])

  // Подключение к SSE потоку, если процесс запущен
  useEffect(() => {
    if (status.isRunning && eventsEndpoint) {
      const eventSource = new EventSource(eventsEndpoint)
      
      eventSource.onmessage = (event) => {
        try {
          // Пытаемся распарсить как JSON, если не получается - используем как строку
          let message = event.data
          try {
            const data = JSON.parse(event.data)
            if (data.type === 'log' && data.message) {
              message = data.message
            } else if (data.message) {
              message = data.message
            }
          } catch {
            // Не JSON, используем как есть
          }
          
          setStatus(prev => ({
            ...prev,
            logs: [...prev.logs.slice(-99), message], // Храним последние 100 логов
            currentStep: message
          }))
          
          // Обновляем прогресс из сообщений
          if (message.includes('Обработано')) {
            const match = message.match(/Обработано[:\s]+(\d+)[/\s]+(\d+)/)
            if (match) {
              const processed = parseInt(match[1])
              const total = parseInt(match[2])
              setStatus(prev => ({
                ...prev,
                processed,
                total,
                progress: total > 0 ? (processed / total) * 100 : 0
              }))
            }
          }
        } catch (error) {
          console.error('Error parsing SSE event:', error)
        }
      }

      eventSource.onerror = () => {
        eventSource.close()
      }

      return () => {
        eventSource.close()
      }
    }
  }, [status.isRunning, eventsEndpoint])

  const handleStart = async () => {
    if (!startEndpoint || !onStart) return
    
    setLoading(true)
    try {
      await onStart()
      setTimeout(() => fetchStatus(), 1000)
    } catch (error) {
      console.error('Error starting process:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleStop = async () => {
    if (!stopEndpoint || !onStop) return
    
    setLoading(true)
    try {
      await onStop()
      setTimeout(() => fetchStatus(), 1000)
    } catch (error) {
      console.error('Error stopping process:', error)
    } finally {
      setLoading(false)
    }
  }

  const progressPercent = status.total > 0 
    ? Math.min((status.processed / status.total) * 100, 100) 
    : 0

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              {status.isRunning ? (
                <RefreshCw className="h-5 w-5 animate-spin text-green-500" />
              ) : (
                <Square className="h-5 w-5 text-gray-400" />
              )}
              {title}
            </CardTitle>
            <CardDescription>
              {status.isRunning ? 'Процесс выполняется' : 'Процесс не запущен'}
            </CardDescription>
          </div>
          <Badge variant={status.isRunning ? 'default' : 'secondary'}>
            {status.isRunning ? 'Выполняется' : 'Остановлено'}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Прогресс */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium">Прогресс</span>
            <span className="text-sm text-muted-foreground">
              {status.processed} / {status.total} ({progressPercent.toFixed(1)}%)
            </span>
          </div>
          <Progress value={progressPercent} className="h-2" />
        </div>

        {/* Статистика */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {status.success !== undefined && (
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              <div>
                <div className="text-xs text-muted-foreground">Успешно</div>
                <div className="text-lg font-semibold">{status.success}</div>
              </div>
            </div>
          )}
          {status.errors !== undefined && (
            <div className="flex items-center gap-2">
              <XCircle className="h-4 w-4 text-red-500" />
              <div>
                <div className="text-xs text-muted-foreground">Ошибок</div>
                <div className="text-lg font-semibold">{status.errors}</div>
              </div>
            </div>
          )}
          {status.rate !== undefined && status.rate > 0 && (
            <div className="flex items-center gap-2">
              <Zap className="h-4 w-4 text-yellow-500" />
              <div>
                <div className="text-xs text-muted-foreground">Скорость</div>
                <div className="text-lg font-semibold">{status.rate.toFixed(2)}/сек</div>
              </div>
            </div>
          )}
          {estimatedTimeRemaining && (
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-blue-500" />
              <div>
                <div className="text-xs text-muted-foreground">Осталось</div>
                <div className="text-lg font-semibold">{estimatedTimeRemaining}</div>
              </div>
            </div>
          )}
        </div>

        {/* Текущий шаг */}
        {status.currentStep && status.currentStep !== 'Не запущено' && (
          <Alert className={
            status.currentStep.includes('Ошибка') ? 'border-red-500 bg-red-50 dark:bg-red-950' :
            status.currentStep.includes('завершена') || status.currentStep.includes('завершен') ? 'border-green-500 bg-green-50 dark:bg-green-950' :
            ''
          }>
            <AlertCircle className={`h-4 w-4 ${
              status.currentStep.includes('Ошибка') ? 'text-red-500' :
              status.currentStep.includes('завершена') || status.currentStep.includes('завершен') ? 'text-green-500' :
              ''
            }`} />
            <AlertDescription className={
              status.currentStep.includes('Ошибка') ? 'text-red-700 dark:text-red-300' :
              status.currentStep.includes('завершена') || status.currentStep.includes('завершен') ? 'text-green-700 dark:text-green-300' :
              ''
            }>
              {status.currentStep}
            </AlertDescription>
          </Alert>
        )}

        {/* Прошедшее время */}
        {status.startTime && (
          <div className="text-sm text-muted-foreground">
            <Clock className="h-4 w-4 inline mr-1" />
            Начато: {new Date(status.startTime as string).toLocaleTimeString('ru-RU')}
            {status.elapsedTime && ` • Прошло: ${status.elapsedTime}`}
          </div>
        )}

        {/* Кнопки управления */}
        {(startEndpoint || stopEndpoint) && (
          <div className="flex gap-2">
            {startEndpoint && !status.isRunning && (
              <Button 
                onClick={handleStart} 
                disabled={loading}
                className="flex-1"
              >
                <Play className="h-4 w-4 mr-2" />
                Запустить
              </Button>
            )}
            {stopEndpoint && status.isRunning && (
              <Button 
                onClick={handleStop} 
                disabled={loading}
                variant="destructive"
                className="flex-1"
              >
                <Square className="h-4 w-4 mr-2" />
                Остановить
              </Button>
            )}
          </div>
        )}

        {/* Последние логи */}
        {status.logs.length > 0 && (
          <div className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <div className="text-sm font-medium">Последние логи ({status.logs.length})</div>
              {status.logs.length > 10 && (
                <div className="text-xs text-muted-foreground">
                  Показано последние 10 из {status.logs.length}
                </div>
              )}
            </div>
            <div className="bg-muted rounded-md p-3 max-h-40 overflow-y-auto">
              <div className="space-y-1 text-xs font-mono">
                {status.logs.slice(-10).map((log, index) => (
                  <div 
                    key={index} 
                    className={`text-muted-foreground ${
                      log.includes('Ошибка') || log.includes('❌') ? 'text-red-500' :
                      log.includes('✅') || log.includes('завершена') ? 'text-green-500' :
                      log.includes('⚠') ? 'text-yellow-500' : ''
                    }`}
                  >
                    {log}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Индикатор завершения */}
        {!status.isRunning && status.progress >= 100 && (
          <Alert className="mt-4 border-green-500 bg-green-50 dark:bg-green-950">
            <CheckCircle2 className="h-4 w-4 text-green-500" />
            <AlertDescription className="text-green-700 dark:text-green-300">
              Процесс успешно завершен! Все записи обработаны.
            </AlertDescription>
          </Alert>
        )}

        {/* Индикатор ошибок */}
        {status.errors !== undefined && status.errors > 0 && (
          <Alert className="mt-4 border-red-500 bg-red-50 dark:bg-red-950">
            <XCircle className="h-4 w-4 text-red-500" />
            <AlertDescription className="text-red-700 dark:text-red-300">
              Обнаружено {status.errors} ошибок при обработке. Проверьте логи для деталей.
            </AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  )
}

