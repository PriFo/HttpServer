'use client'

import { useContext } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { motion } from 'framer-motion'
import { Clock, FileText, AlertCircle, Info, CheckCircle2 } from 'lucide-react'
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
  metadata?: Record<string, any>
}

interface WorkerTraceDetailsProps {
  step: WorkerTraceStep | null
  onClose: () => void
}

const getLevelIcon = (level: string) => {
  switch (level) {
    case 'ERROR':
      return <AlertCircle className="h-5 w-5 text-red-500" />
    case 'WARNING':
      return <AlertCircle className="h-5 w-5 text-yellow-500" />
    case 'INFO':
      return <Info className="h-5 w-5 text-blue-500" />
    default:
      return <CheckCircle2 className="h-5 w-5 text-gray-500" />
  }
}

const getLevelColor = (level: string) => {
  switch (level) {
    case 'ERROR':
      return 'text-red-600 border-red-500'
    case 'WARNING':
      return 'text-yellow-600 border-yellow-500'
    case 'INFO':
      return 'text-blue-600 border-blue-500'
    default:
      return 'text-gray-600 border-gray-500'
  }
}

export function WorkerTraceDetails({ step, onClose }: WorkerTraceDetailsProps) {
  const animationContext = useContext(AnimationContext)
  const config = animationContext?.getAnimationConfig() || { duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }
  const variants = getAnimationVariants(config)

  if (!step) return null

  return (
    <Dialog open={!!step} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <motion.div
          initial="hidden"
          animate="visible"
          variants={variants}
        >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {getLevelIcon(step.level)}
            <span>Детали шага: {step.step}</span>
          </DialogTitle>
          <DialogDescription>
            Подробная информация о выполнении шага
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Основная информация */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground mb-1">ID шага</p>
              <p className="font-mono text-sm">{step.id}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">Trace ID</p>
              <p className="font-mono text-sm">{step.trace_id}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">Уровень</p>
              <Badge variant="outline" className={getLevelColor(step.level)}>
                {step.level}
              </Badge>
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">Длительность</p>
              <p className="text-sm font-medium">
                {step.duration ? `${step.duration}ms` : 'N/A'}
              </p>
            </div>
          </div>

          {/* Временные метки */}
          <div className="border-t pt-4">
            <p className="text-sm font-medium mb-2 flex items-center gap-2">
              <Clock className="h-4 w-4" />
              Временные метки
            </p>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground mb-1">Начало</p>
                <p className="font-mono">
                  {new Date(step.start_time).toLocaleString('ru-RU', {
                    dateStyle: 'short',
                    timeStyle: 'medium',
                  })}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {new Date(step.start_time).toISOString()}
                </p>
              </div>
              {step.end_time && (
                <div>
                  <p className="text-muted-foreground mb-1">Завершение</p>
                  <p className="font-mono">
                    {new Date(step.end_time).toLocaleString('ru-RU', {
                      dateStyle: 'short',
                      timeStyle: 'medium',
                    })}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {new Date(step.end_time).toISOString()}
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Сообщение */}
          <div className="border-t pt-4">
            <p className="text-sm font-medium mb-2 flex items-center gap-2">
              <FileText className="h-4 w-4" />
              Сообщение
            </p>
            <p className="text-sm text-muted-foreground bg-muted p-3 rounded-md">
              {step.message}
            </p>
          </div>

          {/* Метаданные */}
          {step.metadata && Object.keys(step.metadata).length > 0 && (
            <div className="border-t pt-4">
              <p className="text-sm font-medium mb-2">Метаданные</p>
              <div className="bg-muted p-3 rounded-md">
                <pre className="text-xs overflow-x-auto">
                  {JSON.stringify(step.metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>
        </motion.div>
      </DialogContent>
    </Dialog>
  )
}

