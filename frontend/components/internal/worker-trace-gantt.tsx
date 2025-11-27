'use client'

import { useMemo, useContext } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { motion } from 'framer-motion'
import { DynamicBarChart, DynamicBar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell } from '@/lib/recharts-dynamic'
import { AnimationContext, getAnimationVariants } from '@/providers/animation-provider'
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

interface WorkerTraceGanttProps {
  steps: WorkerTraceStep[]
}

const LEVEL_COLORS = {
  INFO: tokens.color.info[500],
  WARNING: tokens.color.warning[500],
  ERROR: tokens.color.error[500],
}

// Выносим CustomTooltip из компонента
// Recharts передает сложный тип для Tooltip, используем any для совместимости
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const CustomTooltip = ({ active, payload }: { active?: boolean; payload?: readonly any[] }) => {
  if (active && payload && payload.length && payload[0]?.payload) {
    const data = payload[0].payload
    return (
      <div className="bg-background border rounded-lg p-3 shadow-lg">
        <p className="font-medium mb-2">{data.step || 'N/A'}</p>
        <div className="space-y-1 text-sm">
          <p>
            <span className="text-muted-foreground">Уровень:</span>{' '}
            <span className="font-medium">{data.level || 'N/A'}</span>
          </p>
          <p>
            <span className="text-muted-foreground">Длительность:</span>{' '}
            <span className="font-medium">{data.duration ? `${data.duration}ms` : 'N/A'}</span>
          </p>
          {data.absoluteStart && (
            <p>
              <span className="text-muted-foreground">Начало:</span>{' '}
              <span className="font-medium">
                {new Date(data.absoluteStart).toLocaleTimeString('ru-RU', {
                  hour: '2-digit',
                  minute: '2-digit',
                  second: '2-digit',
                  fractionalSecondDigits: 3,
                })}
              </span>
            </p>
          )}
          {data.message && (
            <p className="text-muted-foreground text-xs mt-2">{data.message}</p>
          )}
        </div>
      </div>
    )
  }
  return null
}

export function WorkerTraceGantt({ steps }: WorkerTraceGanttProps) {
  const chartData = useMemo(() => {
    if (steps.length === 0) return []

    // Находим минимальное время начала для нормализации
    const minStartTime = Math.min(...steps.map((s) => s.start_time))
    
    return steps.map((step, index) => {
      const startOffset = step.start_time - minStartTime
      const duration = step.duration || (step.end_time ? step.end_time - step.start_time : 100)
      
      return {
        name: step.step,
        step: step.step,
        start: startOffset,
        duration: duration,
        end: startOffset + duration,
        level: step.level,
        message: step.message,
        index: index,
        absoluteStart: step.start_time, // Сохраняем абсолютное время для tooltip
      }
    })
  }, [steps])

  if (steps.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Gantt-диаграмма</CardTitle>
          <CardDescription>Визуализация шагов выполнения</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-[300px] text-muted-foreground">
            Нет данных для отображения
          </div>
        </CardContent>
      </Card>
    )
  }

  const animationContext = useContext(AnimationContext)
  const config = animationContext?.getAnimationConfig() || { duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }
  const variants = getAnimationVariants(config)

  return (
    <motion.div
      initial="hidden"
      animate="visible"
      variants={variants}
    >
      <Card>
        <CardHeader>
          <CardTitle>Gantt-диаграмма</CardTitle>
          <CardDescription>
            Визуализация шагов выполнения воркера по времени
          </CardDescription>
        </CardHeader>
        <CardContent>
        <ResponsiveContainer width="100%" height={Math.max(300, steps.length * 50)}>
          <DynamicBarChart
            data={chartData}
            layout="vertical"
            margin={{ top: 20, right: 30, left: 150, bottom: 20 }}
          >
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis 
              type="number" 
              label={{ value: 'Время (мс)', position: 'insideBottom', offset: -5 }}
            />
            <YAxis 
              dataKey="name" 
              type="category" 
              width={140}
              tick={{ fontSize: 12 }}
            />
            <Tooltip content={(props) => <CustomTooltip {...props} />} />
            <Legend />
            <DynamicBar dataKey="duration" name="Длительность (мс)" radius={[0, 4, 4, 0]}>
              {chartData.map((entry, index) => (
                <Cell 
                  key={`cell-${index}`} 
                  fill={LEVEL_COLORS[entry.level as keyof typeof LEVEL_COLORS] || '#6b7280'} 
                />
              ))}
            </DynamicBar>
          </DynamicBarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
    </motion.div>
  )
}

