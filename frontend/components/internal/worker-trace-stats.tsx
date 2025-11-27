'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Clock, CheckCircle2, AlertTriangle, Info, TrendingUp, Timer } from 'lucide-react'
import { tokens } from '@/styles/tokens'

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

interface WorkerTraceStatsProps {
  steps: WorkerTraceStep[]
}

export function WorkerTraceStats({ steps }: WorkerTraceStatsProps) {
  const stats = useMemo(() => {
    if (steps.length === 0) {
      return {
        totalSteps: 0,
        totalDuration: 0,
        avgDuration: 0,
        levelCounts: { INFO: 0, WARNING: 0, ERROR: 0 },
        longestStep: null as WorkerTraceStep | null,
        shortestStep: null as WorkerTraceStep | null,
        startTime: undefined as number | undefined,
        endTime: undefined as number | undefined,
      }
    }

    const levelCounts = {
      INFO: 0,
      WARNING: 0,
      ERROR: 0,
    }

    let totalDuration = 0
    let longestStep: WorkerTraceStep | null = null
    let shortestStep: WorkerTraceStep | null = null
    let maxDuration = 0
    let minDuration = Infinity

    steps.forEach((step) => {
      levelCounts[step.level]++
      
      const duration = step.duration || (step.end_time ? step.end_time - step.start_time : 0)
      totalDuration += duration

      if (duration > maxDuration) {
        maxDuration = duration
        longestStep = step
      }
      if (duration < minDuration && duration > 0) {
        minDuration = duration
        shortestStep = step
      }
    })

    const startTime = Math.min(...steps.map((s) => s.start_time))
    const endTime = Math.max(...steps.map((s) => s.end_time || s.start_time))
    const totalExecutionTime = endTime - startTime

    return {
      totalSteps: steps.length,
      totalDuration: totalExecutionTime,
      avgDuration: totalDuration / steps.length,
      levelCounts,
      longestStep,
      shortestStep,
      startTime,
      endTime,
    }
  }, [steps])

  if (steps.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Статистика</CardTitle>
          <CardDescription>Статистика выполнения шагов</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-[200px] text-muted-foreground">
            Нет данных для отображения
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5" />
          Статистика выполнения
        </CardTitle>
        <CardDescription>
          Анализ шагов выполнения воркера
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Общая статистика */}
          <div className="p-4 border rounded-lg">
            <div className="text-sm text-muted-foreground mb-1">Всего шагов</div>
            <div className="text-2xl font-bold">{stats.totalSteps}</div>
          </div>

          <div className="p-4 border rounded-lg">
            <div className="text-sm text-muted-foreground mb-1 flex items-center gap-1">
              <Timer className="h-4 w-4" />
              Общее время
            </div>
            <div className="text-2xl font-bold">{stats.totalDuration.toFixed(0)}ms</div>
          </div>

          <div className="p-4 border rounded-lg">
            <div className="text-sm text-muted-foreground mb-1 flex items-center gap-1">
              <Clock className="h-4 w-4" />
              Среднее время
            </div>
            <div className="text-2xl font-bold">{stats.avgDuration.toFixed(1)}ms</div>
          </div>

          <div className="p-4 border rounded-lg">
            <div className="text-sm text-muted-foreground mb-1">Успешных</div>
            <div className="text-2xl font-bold text-green-600">
              {stats.levelCounts.INFO}
            </div>
          </div>

          {/* Уровни */}
          <div className="p-4 border rounded-lg">
            <div className="text-sm text-muted-foreground mb-2">Уровни логов</div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Info className="h-4 w-4 text-blue-500" />
                  <span className="text-sm">INFO</span>
                </div>
                <Badge variant="outline">{stats.levelCounts.INFO}</Badge>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-yellow-500" />
                  <span className="text-sm">WARNING</span>
                </div>
                <Badge variant="outline">{stats.levelCounts.WARNING}</Badge>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-red-500" />
                  <span className="text-sm">ERROR</span>
                </div>
                <Badge variant="outline">{stats.levelCounts.ERROR}</Badge>
              </div>
            </div>
          </div>

          {/* Самый долгий шаг */}
          {stats.longestStep && (
            <div className="p-4 border rounded-lg">
              <div className="text-sm text-muted-foreground mb-1">Самый долгий шаг</div>
              <div className="text-sm font-medium mb-1">{stats.longestStep.step}</div>
              <div className="text-xs text-muted-foreground">
                {stats.longestStep.duration}ms
              </div>
            </div>
          )}

          {/* Самый короткий шаг */}
          {stats.shortestStep && (
            <div className="p-4 border rounded-lg">
              <div className="text-sm text-muted-foreground mb-1">Самый короткий шаг</div>
              <div className="text-sm font-medium mb-1">{stats.shortestStep.step}</div>
              <div className="text-xs text-muted-foreground">
                {stats.shortestStep.duration}ms
              </div>
            </div>
          )}

          {/* Временной диапазон */}
          <div className="p-4 border rounded-lg">
            <div className="text-sm text-muted-foreground mb-1">Временной диапазон</div>
            <div className="text-xs text-muted-foreground mb-1">
              Начало: {stats.startTime ? new Date(stats.startTime).toLocaleTimeString('ru-RU') : 'N/A'}
            </div>
            <div className="text-xs text-muted-foreground">
              Завершение: {stats.endTime ? new Date(stats.endTime).toLocaleTimeString('ru-RU') : 'N/A'}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

