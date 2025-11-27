'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { RefreshCw, Play, Square, Clock, Zap, CheckCircle2, XCircle, AlertCircle, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'

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
  // Поля для номенклатуры (КПВЭД)
  kpvedClassified?: number
  kpvedTotal?: number
  kpvedProgress?: number
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
  const [endpointNotFound, setEndpointNotFound] = useState(false) // Флаг для 404 ошибок
  const [backendStatus, setBackendStatus] = useState<'unknown' | 'ok' | 'unreachable'>('unknown')
  const backendErrorToastAt = useRef<number>(0)

  const markBackendHealthy = useCallback(() => {
    setBackendStatus(prev => (prev === 'ok' ? prev : 'ok'))
  }, [])

  const notifyBackendUnavailable = useCallback((message: string) => {
    setBackendStatus(prev => (prev === 'unreachable' ? prev : 'unreachable'))
    const now = Date.now()
    if (now - backendErrorToastAt.current > 60000) {
      toast.error(message)
      backendErrorToastAt.current = now
    }
  }, [])

  const isBackendConnectionError = useCallback((message: string) => {
    const normalized = message.toLowerCase()
    return (
      normalized.includes('backend') ||
      normalized.includes('9999') ||
      normalized.includes('failed to fetch') ||
      normalized.includes('networkerror') ||
      normalized.includes('econnrefused')
    )
  }, [])

  const fetchStatus = useCallback(async () => {
    // Если эндпоинт не найден (404), не делаем запросы
    if (endpointNotFound) {
      return
    }

    try {
      const response = await fetch(statusEndpoint)
      if (response.ok) {
        // Сбрасываем флаг 404, если запрос успешен
        setEndpointNotFound(false)
        markBackendHealthy()
        
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
        // Обработка различных HTTP статусов
        let errorMessage = 'Ошибка получения статуса'
        
        // Если 404, устанавливаем флаг и прекращаем дальнейшие запросы
        if (response.status === 404) {
          setEndpointNotFound(true)
          errorMessage = 'Эндпоинт не найден. Проверьте конфигурацию API.'
          setStatus(prev => ({
            ...prev,
            currentStep: errorMessage,
            isRunning: false
          }))
          return // Прекращаем дальнейшие запросы
        }
        
        if (response.status === 503) {
          errorMessage = 'Backend сервер недоступен. Убедитесь, что сервер запущен на порту 9999.'
        } else if (response.status === 504) {
          errorMessage = 'Превышено время ожидания ответа от сервера'
        } else if (response.status >= 500) {
          errorMessage = 'Ошибка сервера при получении статуса'
        }
        
        // Пытаемся получить детали ошибки из ответа
        try {
          const errorData = await response.json()
          if (errorData.error) {
            errorMessage = errorData.error
          }
        } catch {
          // Игнорируем ошибку парсинга JSON
        }
        
        setStatus(prev => ({
          ...prev,
          currentStep: prev.isRunning ? errorMessage : prev.currentStep,
          isRunning: false // Сбрасываем флаг запуска при ошибке
        }))

        if (response.status === 503 || response.status === 504 || response.status === 502 || isBackendConnectionError(errorMessage)) {
          notifyBackendUnavailable(errorMessage)
        }
      }
    } catch (error) {
      // Логируем ошибку только если она содержит полезную информацию
      if (error instanceof Error && error.message) {
        console.error('Error fetching status:', error.message)
      } else if (error && typeof error === 'object' && Object.keys(error).length > 0) {
        console.error('Error fetching status:', error)
      }
      // Обработка сетевых ошибок
      const errorMessage = error instanceof Error && error.message && (
        error.message.includes('fetch failed') ||
        error.message.includes('Failed to fetch') ||
        error.message.includes('NetworkError') ||
        error.message.includes('ECONNREFUSED')
      ) ? 'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.' : 'Ошибка сети при получении статуса'
      
      setStatus(prev => ({
        ...prev,
        currentStep: errorMessage,
        isRunning: false
      }))

      if (isBackendConnectionError(errorMessage)) {
        notifyBackendUnavailable(errorMessage)
      }
    }
  }, [statusEndpoint, endpointNotFound, markBackendHealthy, notifyBackendUnavailable, isBackendConnectionError])

  const handleBackendRetry = useCallback(() => {
    setBackendStatus('unknown')
    backendErrorToastAt.current = 0
    setEndpointNotFound(false)
    fetchStatus()
  }, [fetchStatus])

  useEffect(() => {
    // Сбрасываем флаг 404 при изменении endpoint
    setEndpointNotFound(false)
  }, [statusEndpoint])

  useEffect(() => {
    // Если эндпоинт не найден, не делаем запросы
    if (endpointNotFound) {
      return
    }
    
    fetchStatus()
    
    let interval: NodeJS.Timeout | null = null
    
    const setupInterval = () => {
      if (interval) {
        clearInterval(interval)
      }
      
      // Если эндпоинт не найден, не создаем интервал
      if (endpointNotFound) {
        return
      }
      
      // Оптимизация частоты запросов:
      // - Если процесс запущен - обновляем каждые 2 секунды
      // - Если процесс не запущен - обновляем каждые 15 секунд (увеличено с 10 для снижения нагрузки)
      // - Если была ошибка подключения - обновляем каждые 30 секунд (редкие проверки)
      const hasConnectionError = backendStatus === 'unreachable' || status.currentStep?.includes('подключиться') || status.currentStep?.includes('Ошибка сети')
      const intervalTime = status.isRunning 
        ? 2000 
        : hasConnectionError 
          ? 30000 
          : 15000
      
      interval = setInterval(() => {
        // Проверяем перед каждым запросом
        if (!endpointNotFound) {
          fetchStatus()
        } else {
          if (interval) {
            clearInterval(interval)
          }
        }
      }, intervalTime)
    }
    
    setupInterval()
    
    return () => {
      if (interval) {
        clearInterval(interval)
      }
    }
  }, [fetchStatus, status.isRunning, endpointNotFound, backendStatus, status.currentStep])

  // Подключение к SSE потоку, если процесс запущен
  useEffect(() => {
    if (status.isRunning && eventsEndpoint && backendStatus !== 'unreachable') {
      let reconnectAttempts = 0
      let reconnectTimeout: NodeJS.Timeout | null = null
      
      const connect = () => {
        try {
          const eventSource = new EventSource(eventsEndpoint)
          
          eventSource.onopen = () => {
            reconnectAttempts = 0
            markBackendHealthy()
          }
          
          eventSource.onmessage = (event) => {
        try {
          // Пытаемся распарсить как JSON, если не получается - используем как строку
          let message = event.data
          let structuredData: any = null
          
          try {
            const data = JSON.parse(event.data)
            
            // Обрабатываем структурированные события нормализации (контрагенты и номенклатура)
            if (data.type && (data.data || data.type === 'log') && (data.type === 'progress' || data.type === 'start' || 
                data.type === 'completed' || data.type === 'duplicates_found' || 
                data.type === 'processing_started' || data.type === 'database_start' ||
                data.type === 'database_stopped' || data.type === 'database_completed' ||
                data.type === 'kpved_classification' || data.type === 'kpved_progress')) {
              structuredData = data
              
              // Обновляем статус из структурированных данных
              if (data.type === 'progress' && data.data) {
                const progressData = data.data
                setStatus(prev => ({
                  ...prev,
                  processed: progressData.processed || prev.processed,
                  total: progressData.total || prev.total,
                  progress: progressData.progress_percent || prev.progress,
                  logs: [...prev.logs.slice(-99), `Обработано ${progressData.processed || 0} из ${progressData.total || 0} (${(progressData.progress_percent || 0).toFixed(1)}%)`],
                  currentStep: `Обработка: ${progressData.processed || 0}/${progressData.total || 0}`
                }))
                return // Не добавляем в логи, так как уже обновили статус
              }
              
              if (data.type === 'completed' && data.data) {
                const completedData = data.data
                message = `Нормализация завершена: обработано ${completedData.total_processed || 0}, эталонов: ${completedData.benchmark_matches || 0}, дозаполнено: ${completedData.enriched_count || 0}`
              }
              
              if (data.type === 'start' && data.data) {
                message = `Начало нормализации: ${data.data.total_counterparties || 0} контрагентов`
              }
              
              if (data.type === 'database_start' && (data.data || data.database_name)) {
                const dbName = data.database_name || data.data?.database_name || 'неизвестно'
                const count = data.counterparties_count || data.data?.counterparties_count || 0
                message = `Начало обработки БД: ${dbName} (${count} контрагентов)`
              }
              
              if (data.type === 'database_completed' && (data.data || data.database_name)) {
                const dbName = data.database_name || data.data?.database_name || 'неизвестно'
                const processed = data.processed || data.data?.processed || 0
                const total = data.total || data.data?.total || 0
                const benchmarkMatches = data.benchmark_matches || data.data?.benchmark_matches || 0
                const enrichedCount = data.enriched_count || data.data?.enriched_count || 0
                message = `БД ${dbName} обработана: ${processed}/${total} (эталонов: ${benchmarkMatches}, дозаполнено: ${enrichedCount})`
              }
              
              if (data.type === 'database_stopped' && (data.data || data.database_name)) {
                const dbName = data.database_name || data.data?.database_name || 'неизвестно'
                const processed = data.processed || data.data?.processed || 0
                const total = data.total || data.data?.total || 0
                const progressPercent = data.progress_percent || data.data?.progress_percent || 0
                message = `БД ${dbName} остановлена: обработано ${processed}/${total} (${progressPercent.toFixed(1)}%)`
              }
              
              if (data.type === 'duplicates_found' && data.data) {
                const dupData = data.data
                message = `Найдено групп дублей: ${dupData.total_groups || 0}, всего дубликатов: ${dupData.total_duplicates || 0}`
              }
              
              if (data.type === 'processing_started' && data.data) {
                const procData = data.data
                message = `Начало обработки: ${procData.total_counterparties || 0} контрагентов, ${procData.workers || 0} воркеров`
              }
              
              if (data.type === 'normalization_stopped' && data.data) {
                const stopData = data.data
                message = `Нормализация остановлена: обработано ${stopData.processed || 0}/${stopData.total || 0} (${(stopData.progress_percent || 0).toFixed(1)}%)`
              }
              
              // События для номенклатуры (КПВЭД классификация)
              if (data.type === 'kpved_classification' && data.data) {
                const kpvedData = data.data
                message = `КПВЭД классификация: ${kpvedData.classified || 0}/${kpvedData.total || 0} групп (${(kpvedData.progress_percent || 0).toFixed(1)}%)`
                // Обновляем статус с информацией о КПВЭД
                setStatus(prev => ({
                  ...prev,
                  kpvedClassified: kpvedData.classified || prev.kpvedClassified,
                  kpvedTotal: kpvedData.total || prev.kpvedTotal,
                  kpvedProgress: kpvedData.progress_percent || prev.kpvedProgress,
                }))
                return
              }
              
              if (data.type === 'kpved_progress' && data.data) {
                const kpvedData = data.data
                message = `КПВЭД: ${kpvedData.classified || 0}/${kpvedData.total || 0} (${(kpvedData.progress_percent || 0).toFixed(1)}%)`
                setStatus(prev => ({
                  ...prev,
                  kpvedClassified: kpvedData.classified || prev.kpvedClassified,
                  kpvedTotal: kpvedData.total || prev.kpvedTotal,
                  kpvedProgress: kpvedData.progress_percent || prev.kpvedProgress,
                }))
                return
              }
            } else if (data.type === 'log' && data.message) {
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
          
          // Обновляем прогресс из сообщений (fallback для старых форматов)
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

          eventSource.onerror = (err) => {
            // Проверяем состояние соединения
            if (eventSource.readyState === EventSource.CLOSED) {
              // Соединение закрыто - пытаемся переподключиться
              eventSource.close()
              
              // Переподключение с экспоненциальной задержкой
              reconnectAttempts += 1
              const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000)
              
              if (reconnectAttempts <= 5 && status.isRunning) {
                console.log(`[SSE] Reconnecting in ${delay}ms (attempt ${reconnectAttempts})`)
                reconnectTimeout = setTimeout(() => {
                  connect()
                }, delay)
              } else {
                console.error('[SSE] Connection failed after multiple attempts')
                notifyBackendUnavailable('Поток событий недоступен. Проверьте backend сервер (порт 9999).')
              }
            } else {
              // Другие ошибки
              if (eventSource.readyState !== EventSource.CONNECTING) {
                console.error('[SSE] Connection error:', err)
              }
            }
          }
        } catch (err) {
          console.error('[SSE] Error creating EventSource:', err)
        }
      }
      
      // Начальное подключение
      connect()
      
      return () => {
        if (reconnectTimeout) {
          clearTimeout(reconnectTimeout)
        }
      }
    }
  }, [status.isRunning, eventsEndpoint, backendStatus, markBackendHealthy, notifyBackendUnavailable])

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
        {backendStatus === 'unreachable' && (
          <Alert variant="destructive">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              <AlertTitle>Backend недоступен</AlertTitle>
            </div>
            <AlertDescription className="mt-2 space-y-2">
              <p>Не удаётся получить статус процесса (порт 9999). Повторные запросы временно замедлены.</p>
              <Button variant="outline" size="sm" onClick={handleBackendRetry} className="flex items-center gap-2">
                <RefreshCw className="h-3 w-3" />
                Повторить попытку
              </Button>
            </AlertDescription>
          </Alert>
        )}
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
            status.currentStep.includes('Ошибка') || status.currentStep.includes('подключиться') ? 'border-red-500 bg-red-50 dark:bg-red-950' :
            status.currentStep.includes('завершена') || status.currentStep.includes('завершен') ? 'border-green-500 bg-green-50 dark:bg-green-950' :
            ''
          }>
            <AlertCircle className={`h-4 w-4 ${
              status.currentStep.includes('Ошибка') || status.currentStep.includes('подключиться') ? 'text-red-500' :
              status.currentStep.includes('завершена') || status.currentStep.includes('завершен') ? 'text-green-500' :
              ''
            }`} />
            <AlertDescription className={
              status.currentStep.includes('Ошибка') || status.currentStep.includes('подключиться') ? 'text-red-700 dark:text-red-300' :
              status.currentStep.includes('завершена') || status.currentStep.includes('завершен') ? 'text-green-700 dark:text-green-300' :
              ''
            }>
              <div className="flex items-center justify-between">
                <span>{status.currentStep}</span>
                {(status.currentStep.includes('Ошибка') || status.currentStep.includes('подключиться')) && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setEndpointNotFound(false)
                      fetchStatus()
                    }}
                    className="ml-4"
                  >
                    <RefreshCw className="h-3 w-3 mr-1" />
                    Повторить
                  </Button>
                )}
              </div>
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

