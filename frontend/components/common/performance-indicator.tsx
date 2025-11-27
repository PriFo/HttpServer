/**
 * Индикатор производительности с использованием токенов дизайна
 * 
 * Визуализирует метрики производительности с цветовой индикацией
 */

'use client'

import { motion } from 'framer-motion'
import { Badge } from '@/components/ui/badge'
import { tokens } from '@/styles/tokens'
import { cn } from '@/lib/utils'
import { CheckCircle2, AlertTriangle, XCircle, TrendingUp } from 'lucide-react'

interface PerformanceIndicatorProps {
  label: string
  value: number
  unit: string
  thresholds: {
    good: number
    needsImprovement: number
  }
  higherIsBetter?: boolean
  className?: string
}

/**
 * Индикатор производительности с цветовой индикацией
 * 
 * @example
 * ```tsx
 * <PerformanceIndicator
 *   label="LCP"
 *   value={2.1}
 *   unit="s"
 *   thresholds={{ good: 2.5, needsImprovement: 4.0 }}
 *   higherIsBetter={false}
 * />
 * ```
 */
export function PerformanceIndicator({
  label,
  value,
  unit,
  thresholds,
  higherIsBetter = false,
  className,
}: PerformanceIndicatorProps) {
  // Определяем статус производительности
  const getStatus = () => {
    if (higherIsBetter) {
      if (value >= thresholds.good) return 'good'
      if (value >= thresholds.needsImprovement) return 'needs-improvement'
      return 'poor'
    } else {
      if (value <= thresholds.good) return 'good'
      if (value <= thresholds.needsImprovement) return 'needs-improvement'
      return 'poor'
    }
  }

  const status = getStatus()

  const statusConfig = {
    good: {
      color: tokens.color.success[500],
      bgColor: tokens.color.success[50],
      icon: CheckCircle2,
      text: 'Отлично',
    },
    'needs-improvement': {
      color: tokens.color.warning[500],
      bgColor: tokens.color.warning[50],
      icon: AlertTriangle,
      text: 'Требует внимания',
    },
    poor: {
      color: tokens.color.error[500],
      bgColor: tokens.color.error[50],
      icon: XCircle,
      text: 'Плохо',
    },
  }

  const config = statusConfig[status]
  const Icon = config.icon

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.3 }}
      className={cn(
        'flex items-center gap-3 p-4 border rounded-lg transition-colors',
        className
      )}
      style={{
        borderColor: config.color,
        backgroundColor: config.bgColor,
        borderRadius: tokens.borderRadius.md,
        padding: tokens.spacing.md,
      }}
    >
      <Icon 
        className="h-8 w-8"
        style={{ color: config.color }}
      />
      <div className="flex-1">
        <div className="text-sm text-muted-foreground mb-1">{label}</div>
        <div className="text-2xl font-bold" style={{ color: config.color }}>
          {value}{unit}
        </div>
        <Badge 
          variant="outline"
          className="mt-2"
          style={{
            borderColor: config.color,
            color: config.color,
          }}
        >
          {config.text}
        </Badge>
      </div>
    </motion.div>
  )
}

/**
 * Группа индикаторов производительности
 */
interface PerformanceIndicatorsProps {
  items: Array<{
    label: string
    value: number
    unit: string
    thresholds: {
      good: number
      needsImprovement: number
    }
    higherIsBetter?: boolean
  }>
  className?: string
}

export function PerformanceIndicators({
  items,
  className,
}: PerformanceIndicatorsProps) {
  return (
    <div className={cn('grid grid-cols-1 md:grid-cols-3 gap-4', className)}>
      {items.map((item, index) => (
        <motion.div
          key={item.label}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: index * 0.1 }}
        >
          <PerformanceIndicator {...item} />
        </motion.div>
      ))}
    </div>
  )
}

