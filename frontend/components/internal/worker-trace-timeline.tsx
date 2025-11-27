'use client'

import { useMemo, useContext } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { motion, AnimatePresence } from 'framer-motion'
import { Clock, CheckCircle2, AlertCircle, Info } from 'lucide-react'
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

interface WorkerTraceTimelineProps {
  steps: WorkerTraceStep[]
}

const getLevelIcon = (level: string) => {
  switch (level) {
    case 'ERROR':
      return <AlertCircle className="h-4 w-4 text-red-500" />
    case 'WARNING':
      return <AlertCircle className="h-4 w-4 text-yellow-500" />
    case 'INFO':
      return <Info className="h-4 w-4 text-blue-500" />
    default:
      return <CheckCircle2 className="h-4 w-4 text-gray-500" />
  }
}

const getLevelColor = (level: string) => {
  switch (level) {
    case 'ERROR':
      return 'border-red-500 bg-red-50 dark:bg-red-950'
    case 'WARNING':
      return 'border-yellow-500 bg-yellow-50 dark:bg-yellow-950'
    case 'INFO':
      return 'border-blue-500 bg-blue-50 dark:bg-blue-950'
    default:
      return 'border-gray-500 bg-gray-50 dark:bg-gray-950'
  }
}

export function WorkerTraceTimeline({ steps }: WorkerTraceTimelineProps) {
  const animationContext = useContext(AnimationContext)
  const config = animationContext?.getAnimationConfig() || { duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }
  const variants = getAnimationVariants(config)

  const sortedSteps = useMemo(() => {
    return [...steps].sort((a, b) => a.start_time - b.start_time)
  }, [steps])

  if (sortedSteps.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Временная шкала</CardTitle>
          <CardDescription>Хронология выполнения шагов</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-[200px] text-muted-foreground">
            Нет данных для отображения
          </div>
        </CardContent>
      </Card>
    )
  }

  const totalDuration = sortedSteps.length > 0
    ? (sortedSteps[sortedSteps.length - 1].end_time || sortedSteps[sortedSteps.length - 1].start_time) - sortedSteps[0].start_time
    : 0

  return (
    <Card>
      <CardHeader>
        <CardTitle>Временная шкала</CardTitle>
        <CardDescription>
          Хронология выполнения шагов • Общая длительность: {totalDuration}ms
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="relative">
          {/* Вертикальная линия */}
          <div className="absolute left-6 top-0 bottom-0 w-0.5 bg-border" />

          <motion.div
            initial="hidden"
            animate="visible"
            variants={{
              visible: {
                transition: {
                  staggerChildren: 0.05,
                },
              },
            }}
            className="space-y-6"
          >
            {sortedSteps.map((step, index) => {
              const isLast = index === sortedSteps.length - 1
              const prevStep = index > 0 ? sortedSteps[index - 1] : null
              const gap = prevStep
                ? step.start_time - (prevStep.end_time || prevStep.start_time)
                : 0

              return (
                <motion.div
                  key={step.id}
                  initial="hidden"
                  animate="visible"
                  variants={variants}
                  className="relative flex items-start gap-4"
                  style={{
                    padding: tokens.spacing.md,
                    borderRadius: tokens.borderRadius.md,
                  }}
                  whileHover={{ scale: 1.01, x: 4 }}
                  transition={{ duration: 0.2 }}
                >
                  {/* Иконка на линии */}
                  <div className={`relative z-10 flex items-center justify-center w-12 h-12 rounded-full border-2 ${getLevelColor(step.level)}`}>
                    {getLevelIcon(step.level)}
                  </div>

                  {/* Контент */}
                  <div className="flex-1 pt-1">
                    <div className="flex items-center gap-2 mb-1">
                      <Badge variant="outline" className="text-xs">
                        {step.step}
                      </Badge>
                      <Badge variant="outline" className="text-xs">
                        {step.level}
                      </Badge>
                      {step.duration && (
                        <span className="text-xs text-muted-foreground flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          {step.duration}ms
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mb-2">{step.message}</p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(step.start_time).toLocaleString('ru-RU', {
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit',
                        fractionalSecondDigits: 3,
                      })}
                    </p>
                    {gap > 0 && (
                      <div className="mt-2 text-xs text-muted-foreground italic">
                        Пауза: {gap}ms
                      </div>
                    )}
                  </div>
                </motion.div>
              )
            })}
          </motion.div>
        </div>
      </CardContent>
    </Card>
  )
}

