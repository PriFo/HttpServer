'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { Play, Square, ExternalLink, RefreshCw, Loader2 } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { toast } from 'sonner'

interface NormalizationStatus {
  isRunning: boolean
  progress: number
  processed: number
  total: number
  currentStep: string
  errors?: number
  success?: number
}

interface NormalizationProcessCardProps {
  title: string
  description: string
  statusEndpoint: string
  startEndpoint: string
  stopEndpoint: string
  detailPagePath: string
  icon?: React.ReactNode
  clientId?: number | null
  projectId?: number | null
}

export function NormalizationProcessCard({
  title,
  description,
  statusEndpoint,
  startEndpoint,
  stopEndpoint,
  detailPagePath,
  icon,
  clientId,
  projectId,
}: NormalizationProcessCardProps) {
  const router = useRouter()
  const [status, setStatus] = useState<NormalizationStatus>({
    isRunning: false,
    progress: 0,
    processed: 0,
    total: 0,
    currentStep: 'Процесс не запущен',
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isStarting, setIsStarting] = useState(false)
  const [isStopping, setIsStopping] = useState(false)
  const [wasRunning, setWasRunning] = useState(false)

  // Функция для получения статуса
  const fetchStatus = async () => {
    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 7000)

      const response = await fetch(statusEndpoint, {
        cache: 'no-store',
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        // Если 404 и есть clientId/projectId, это нормально - процесс еще не запущен
        if (response.status === 404 && clientId && projectId) {
          setStatus({
            isRunning: false,
            progress: 0,
            processed: 0,
            total: 0,
            currentStep: 'Процесс не запущен',
            errors: 0,
            success: 0,
          })
          setError(null)
          return
        }
        throw new Error('Не удалось получить статус')
      }

      const data = await response.json()
      const newIsRunning = data.isRunning || data.is_running || false
      const prevIsRunning = status.isRunning
      
      setStatus({
        isRunning: newIsRunning,
        progress: data.progress || 0,
        processed: data.processed || 0,
        total: data.total || 0,
        currentStep: data.currentStep || data.current_step || 'Процесс не запущен',
        errors: data.errors || 0,
        success: data.success || 0,
      })
      setError(null)

      // Показываем уведомление при завершении процесса
      if (wasRunning && !newIsRunning && status.processed > 0) {
        const totalProcessed = data.processed || status.processed
        const totalSuccess = data.success || status.success || 0
        const totalErrors = data.errors || status.errors || 0
        
        toast.success('Процесс завершен', {
          description: `Обработано: ${totalProcessed} записей. Успешно: ${totalSuccess}, Ошибок: ${totalErrors}`,
          duration: 5000,
        })
        setWasRunning(false)
      }

      // Обновляем флаг выполнения
      if (newIsRunning && !wasRunning) {
        setWasRunning(true)
      } else if (!newIsRunning) {
        setWasRunning(false)
      }
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        // Логируем ошибку только если она содержит полезную информацию
        if (err.message) {
          console.error('Error fetching status:', err.message)
        } else if (err && typeof err === 'object' && Object.keys(err).length > 0) {
          console.error('Error fetching status:', err)
        }
        // Не показываем ошибку в toast при обычном обновлении статуса
        // setError(err.message)
      }
    } finally {
      setLoading(false)
    }
  }

  // Оптимизация: обновляем чаще только когда процесс запущен
  useEffect(() => {
    fetchStatus()

    let interval: NodeJS.Timeout | null = null
    
    const setupInterval = () => {
      if (interval) {
        clearInterval(interval)
      }
      
      // Оптимизация частоты запросов:
      // - Если процесс запущен - обновляем каждые 2 секунды
      // - Если процесс не запущен - обновляем каждые 15 секунд
      const intervalTime = status.isRunning ? 2000 : 15000
      
      interval = setInterval(() => {
        fetchStatus()
      }, intervalTime)
    }
    
    setupInterval()

    return () => {
      if (interval) {
        clearInterval(interval)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusEndpoint, status.isRunning])

  // Функция для запуска процесса
  const handleStart = async () => {
    setIsStarting(true)
    setError(null)

    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 7000)

      const response = await fetch(startEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Не удалось запустить процесс' }))
        const errorMessage = errorData.error || 'Не удалось запустить процесс'
        
        // Если требуется client_id и project_id, и они не переданы, показываем сообщение
        const errorLower = errorMessage.toLowerCase()
        if (
          (errorLower.includes('client_id') || 
          errorLower.includes('project_id') || 
          errorLower.includes('invalid route') ||
          errorLower.includes('required')) &&
          (!clientId || !projectId)
        ) {
          toast.error('Требуется выбор проекта', {
            description: 'Для запуска процесса необходимо выбрать клиента и проект выше',
          })
          return
        }
        
        throw new Error(errorMessage)
      }

      // Показываем уведомление об успешном запуске
      toast.success('Процесс запущен', {
        description: `${title} успешно запущен`,
      })

      // Обновляем статус после запуска
      setTimeout(() => {
        fetchStatus()
      }, 500)
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        console.error('Error starting process:', err)
        setError(err.message)
        toast.error('Ошибка запуска процесса', {
          description: err.message,
        })
      }
    } finally {
      setIsStarting(false)
    }
  }

  // Функция для остановки процесса
  const handleStop = async () => {
    setIsStopping(true)
    setError(null)

    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 7000)

      const response = await fetch(stopEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Не удалось остановить процесс' }))
        throw new Error(errorData.error || 'Не удалось остановить процесс')
      }

      // Показываем уведомление об успешной остановке
      toast.success('Процесс остановлен', {
        description: `${title} успешно остановлен`,
      })

      // Обновляем статус после остановки
      setTimeout(() => {
        fetchStatus()
      }, 500)
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        console.error('Error stopping process:', err)
        setError(err.message)
        toast.error('Ошибка остановки процесса', {
          description: err.message,
        })
      }
    } finally {
      setIsStopping(false)
    }
  }

  // Определяем статус для отображения
  const getStatusLabel = () => {
    if (status.isRunning) {
      return 'Выполняется'
    }
    if (status.processed > 0) {
      return 'Остановлено'
    }
    return 'Ожидание'
  }

  const getStatusVariant = () => {
    if (status.isRunning) {
      return 'default'
    }
    if (status.processed > 0) {
      return 'secondary'
    }
    return 'outline'
  }

  // Рассчитываем процент прогресса
  const progressPercent = status.total > 0 
    ? (status.processed / status.total) * 100 
    : status.progress

  return (
    <Card className="w-full">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {icon && <div className="text-primary">{icon}</div>}
            <div>
              <CardTitle className="text-xl">{title}</CardTitle>
              <CardDescription className="mt-1">{description}</CardDescription>
            </div>
          </div>
          <Badge variant={getStatusVariant() as any}>
            {getStatusLabel()}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {!clientId || !projectId ? (
          <Alert className="border-yellow-200 bg-yellow-50 dark:bg-yellow-950 dark:border-yellow-800">
            <AlertDescription className="text-yellow-800 dark:text-yellow-200">
              Для запуска процесса выберите клиента и проект на странице выше.
            </AlertDescription>
          </Alert>
        ) : null}

        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Текущий этап:</span>
            <span className="font-medium">{status.currentStep}</span>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Прогресс:</span>
              <span className="font-medium">
                {status.processed} / {status.total} ({progressPercent.toFixed(1)}%)
              </span>
            </div>
            <Progress value={progressPercent} className="h-2" />
          </div>

          {status.success !== undefined && status.errors !== undefined && (
            <div className="flex items-center gap-4 text-sm">
              <div className="flex items-center gap-1">
                <span className="text-muted-foreground">Успешно:</span>
                <span className="font-medium text-green-600">{status.success}</span>
              </div>
              <div className="flex items-center gap-1">
                <span className="text-muted-foreground">Ошибок:</span>
                <span className="font-medium text-red-600">{status.errors}</span>
              </div>
            </div>
          )}
        </div>
      </CardContent>

      <CardFooter className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => router.push(detailPagePath)}
            className="flex items-center gap-2"
          >
            <ExternalLink className="h-4 w-4" />
            Детали
          </Button>
          
          <Button
            variant="ghost"
            size="sm"
            onClick={fetchStatus}
            disabled={loading}
            className="flex items-center gap-2"
            title="Обновить статус"
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
          </Button>
        </div>

        <div className="flex items-center gap-2">
          {status.isRunning ? (
            <Button
              variant="destructive"
              size="sm"
              onClick={handleStop}
              disabled={isStopping}
              className="flex items-center gap-2"
            >
              {isStopping ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Остановка...
                </>
              ) : (
                <>
                  <Square className="h-4 w-4" />
                  Остановить
                </>
              )}
            </Button>
          ) : (
            <Button
              variant="default"
              size="sm"
              onClick={handleStart}
              disabled={isStarting}
              className="flex items-center gap-2"
            >
              {isStarting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Запуск...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4" />
                  Запустить
                </>
              )}
            </Button>
          )}
        </div>
      </CardFooter>
    </Card>
  )
}

