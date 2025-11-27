'use client'

import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Play, Square, Download, Filter } from 'lucide-react'
import { getBackendUrl } from '@/lib/api-config'
import { toast } from 'sonner'
import { useAnimationContext } from '@/providers/animation-provider'
import { WorkerTraceGantt } from '@/components/internal/worker-trace-gantt'
import { WorkerTraceTimeline } from '@/components/internal/worker-trace-timeline'
import { WorkerTraceDetails } from '@/components/internal/worker-trace-details'
import { WorkerTraceStats } from '@/components/internal/worker-trace-stats'
import { TraceIDHistory, useTraceHistory } from '@/components/internal/trace-id-history'

interface WorkerTraceStep {
  id: string
  trace_id: string
  step: string
  start_time: number
  end_time?: number
  duration?: number
  level: 'INFO' | 'WARNING' | 'ERROR'
  message: string
  metadata?: Record<string, unknown>
}

export default function WorkerTracePage() {
  const [traceId, setTraceId] = useState('')
  const [steps, setSteps] = useState<WorkerTraceStep[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [levelFilter, setLevelFilter] = useState<'all' | 'INFO' | 'WARNING' | 'ERROR'>('all')
  const [selectedStep, setSelectedStep] = useState<WorkerTraceStep | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const { getAnimationConfig } = useAnimationContext()
  const animationConfig = getAnimationConfig()
  const { addToHistory } = useTraceHistory()

  const connect = () => {
    if (!traceId.trim()) {
      toast.error('Введите trace_id')
      return
    }

    disconnect()

    setIsLoading(true)
    const backendUrl = getBackendUrl()
    const url = `${backendUrl}/api/internal/worker-trace/stream?trace_id=${encodeURIComponent(traceId)}`
    
    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onopen = () => {
      setIsConnected(true)
      setIsLoading(false)
      setSteps([])
      toast.success('Подключено к потоку трассировки')
    }

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        
        if (data.type === 'connected') {
          setIsConnected(true)
          setIsLoading(false)
          return
        }

        if (data.type === 'finished') {
          setIsConnected(false)
          toast.info('Трассировка завершена')
          return
        }

        // Добавляем шаг (нормализуем данные с бэкенда)
        if (data.id && data.step) {
          const normalizedStep: WorkerTraceStep = {
            id: data.id || data.ID,
            trace_id: data.trace_id || data.TraceID || data.trace_id,
            step: data.step || data.Step,
            start_time: data.start_time || data.StartTime,
            end_time: data.end_time || data.EndTime,
            duration: data.duration || data.Duration,
            level: (data.level || data.Level || 'INFO') as 'INFO' | 'WARNING' | 'ERROR',
            message: data.message || data.Message || '',
            metadata: data.metadata || data.Metadata,
          }
          setSteps((prev) => [...prev, normalizedStep])
        }
      } catch (err) {
        console.error('Error parsing SSE data:', err)
      }
    }

    eventSource.onerror = (error) => {
      console.error('SSE error:', error)
      setIsConnected(false)
      setIsLoading(false)
      toast.error('Ошибка подключения к потоку трассировки')
      eventSource.close()
    }
  }

  const disconnect = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setIsConnected(false)
  }

  useEffect(() => {
    return () => {
      disconnect()
    }
  }, [])

  const filteredSteps = steps.filter((step) => 
    levelFilter === 'all' || step.level === levelFilter
  )

  const exportToJSON = () => {
    const dataStr = JSON.stringify(steps, null, 2)
    const dataBlob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(dataBlob)
    const link = document.createElement('a')
    link.href = url
    link.download = `worker-trace-${traceId}-${Date.now()}.json`
    link.click()
    URL.revokeObjectURL(url)
  }

  const exportToCSV = () => {
    const headers = ['ID', 'Step', 'Level', 'Message', 'Start Time', 'Duration (ms)']
    const rows = steps.map((step) => [
      step.id,
      step.step,
      step.level,
      step.message,
      new Date(step.start_time).toISOString(),
      step.duration || '',
    ])
    
    const csv = [
      headers.join(','),
      ...rows.map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(',')),
    ].join('\n')

    const dataBlob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(dataBlob)
    const link = document.createElement('a')
    link.href = url
    link.download = `worker-trace-${traceId}-${Date.now()}.csv`
    link.click()
    URL.revokeObjectURL(url)
  }

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'ERROR':
        return 'bg-red-500'
      case 'WARNING':
        return 'bg-yellow-500'
      case 'INFO':
        return 'bg-blue-500'
      default:
        return 'bg-gray-500'
    }
  }

  return (
    <div className="space-y-6">
      {/* История trace_id */}
      <TraceIDHistory onSelectTraceId={setTraceId} />

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: animationConfig.duration, ease: animationConfig.ease as [number, number, number, number] }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Трассировка воркеров</CardTitle>
            <CardDescription>
              Отслеживание выполнения шагов воркеров в реальном времени по trace_id
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-4 mb-4">
              <div className="flex-1">
                <Input
                  placeholder="Введите trace_id"
                  value={traceId}
                  onChange={(e) => setTraceId(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !isConnected) {
                      connect()
                    }
                  }}
                  disabled={isConnected}
                />
              </div>
              {!isConnected ? (
                <Button onClick={connect} disabled={isLoading || !traceId.trim()}>
                  <Play className="h-4 w-4 mr-2" />
                  {isLoading ? 'Подключение...' : 'Подключиться'}
                </Button>
              ) : (
                <Button onClick={disconnect} variant="destructive">
                  <Square className="h-4 w-4 mr-2" />
                  Отключиться
                </Button>
              )}
            </div>

            {steps.length > 0 && (
              <div className="flex gap-2 mb-4">
                <Button variant="outline" size="sm" onClick={exportToJSON}>
                  <Download className="h-4 w-4 mr-2" />
                  Экспорт JSON
                </Button>
                <Button variant="outline" size="sm" onClick={exportToCSV}>
                  <Download className="h-4 w-4 mr-2" />
                  Экспорт CSV
                </Button>
                <div className="flex items-center gap-2 ml-auto">
                  <Filter className="h-4 w-4 text-muted-foreground" />
                  <select
                    value={levelFilter}
                    onChange={(e) => setLevelFilter(e.target.value as any)}
                    className="px-3 py-1 border rounded-md text-sm"
                  >
                    <option value="all">Все уровни</option>
                    <option value="INFO">INFO</option>
                    <option value="WARNING">WARNING</option>
                    <option value="ERROR">ERROR</option>
                  </select>
                </div>
              </div>
            )}

            {isConnected && (
              <div className="flex items-center gap-2 text-sm text-green-600 mb-4">
                <div className="h-2 w-2 bg-green-500 rounded-full animate-pulse" />
                Подключено • Получено шагов: {steps.length}
              </div>
            )}
          </CardContent>
        </Card>
      </motion.div>

      {/* Статистика */}
      {filteredSteps.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: animationConfig.duration, delay: 0.12, ease: animationConfig.ease as [number, number, number, number] }}
        >
          <WorkerTraceStats steps={filteredSteps} />
        </motion.div>
      )}

      {/* Gantt-диаграмма */}
      {filteredSteps.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: animationConfig.duration, delay: 0.15, ease: animationConfig.ease as [number, number, number, number] }}
        >
          <WorkerTraceGantt steps={filteredSteps} />
        </motion.div>
      )}

      {/* Список шагов */}
      {filteredSteps.length > 0 ? (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: animationConfig.duration, delay: 0.1, ease: animationConfig.ease as [number, number, number, number] }}
        >
          <Card>
            <CardHeader>
              <CardTitle>Шаги выполнения ({filteredSteps.length})</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {filteredSteps.map((step, index) => (
                  <motion.div
                    key={step.id}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ 
                      duration: animationConfig.duration * 0.5, 
                      delay: index * 0.05,
                      ease: animationConfig.ease as [number, number, number, number] 
                    }}
                    className="p-4 border rounded-lg hover:bg-accent/50 transition-colors cursor-pointer"
                    onClick={() => setSelectedStep(step)}
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          <Badge 
                            variant="outline" 
                            className={getLevelColor(step.level)}
                          >
                            {step.level}
                          </Badge>
                          <span className="font-medium">{step.step}</span>
                          {step.duration && (
                            <span className="text-sm text-muted-foreground">
                              {step.duration}ms
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">{step.message}</p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {new Date(step.start_time).toLocaleString('ru-RU')}
                        </p>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            </CardContent>
          </Card>
        </motion.div>
      ) : isConnected ? (
        <Card>
          <CardContent className="py-8">
            <div className="text-center text-muted-foreground">
              <Skeleton className="h-4 w-48 mx-auto mb-2" />
              <p>Ожидание данных...</p>
            </div>
          </CardContent>
        </Card>
      ) : null}

      {/* Временная шкала */}
      {filteredSteps.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: animationConfig.duration, delay: 0.2, ease: animationConfig.ease as [number, number, number, number] }}
        >
          <WorkerTraceTimeline steps={filteredSteps} />
        </motion.div>
      )}

      {/* Детали шага */}
      <WorkerTraceDetails step={selectedStep} onClose={() => setSelectedStep(null)} />
    </div>
  )
}
