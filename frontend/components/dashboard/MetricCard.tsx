'use client'

import { motion } from 'framer-motion'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { LucideIcon } from 'lucide-react'
import { AnimatedNumber } from './AnimatedNumber'
import { cn } from '@/lib/utils'

interface MetricCardProps {
  title: string
  value: number | string
  previousValue?: number
  icon: LucideIcon
  color?: 'blue' | 'green' | 'yellow' | 'red' | 'purple'
  format?: (value: number) => string
  suffix?: string
  trend?: {
    value: number
    isPositive: boolean
  }
}

const colorClasses = {
  blue: {
    bg: 'bg-blue-50 dark:bg-blue-950',
    text: 'text-blue-600 dark:text-blue-400',
    border: 'border-blue-200 dark:border-blue-800',
  },
  green: {
    bg: 'bg-green-50 dark:bg-green-950',
    text: 'text-green-600 dark:text-green-400',
    border: 'border-green-200 dark:border-green-800',
  },
  yellow: {
    bg: 'bg-yellow-50 dark:bg-yellow-950',
    text: 'text-yellow-600 dark:text-yellow-400',
    border: 'border-yellow-200 dark:border-yellow-800',
  },
  red: {
    bg: 'bg-red-50 dark:bg-red-950',
    text: 'text-red-600 dark:text-red-400',
    border: 'border-red-200 dark:border-red-800',
  },
  purple: {
    bg: 'bg-purple-50 dark:bg-purple-950',
    text: 'text-purple-600 dark:text-purple-400',
    border: 'border-purple-200 dark:border-purple-800',
  },
}

export function MetricCard({
  title,
  value,
  previousValue,
  icon: Icon,
  color = 'blue',
  format,
  suffix = '',
  trend,
}: MetricCardProps) {
  const colors = colorClasses[color]
  
  // Безопасная нормализация значения
  const safeValue = typeof value === 'number' 
    ? (isNaN(value) || !isFinite(value) ? 0 : value)
    : value || '0'
  
  const formattedValue = typeof safeValue === 'number'
    ? (format ? format(safeValue) : safeValue.toLocaleString('ru-RU'))
    : String(safeValue)

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      whileHover={{ 
        scale: 1.03,
        y: -4,
        transition: { duration: 0.2 }
      }}
      transition={{ 
        duration: 0.3,
        type: "spring",
        stiffness: 200,
        damping: 20
      }}
    >
      <Card className={cn('relative overflow-hidden', colors.bg, colors.border)}>
        <div className={cn('absolute top-0 right-0 w-32 h-32 rounded-full blur-3xl opacity-20', colors.text)} />
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium">{title}</CardTitle>
            <div className={cn('p-2 rounded-lg', colors.bg)}>
              <Icon className={cn('h-4 w-4', colors.text)} />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <motion.div
            key={safeValue}
            initial={{ scale: 1.2, opacity: 0.5 }}
            animate={{ scale: 1, opacity: 1 }}
            className={cn('text-3xl font-bold', colors.text)}
          >
            {typeof safeValue === 'number' ? (
              format ? (
                <span className={colors.text}>
                  {format(safeValue)}
                  {suffix}
                </span>
              ) : (
                <AnimatedNumber
                  value={safeValue}
                  duration={0.8}
                  decimals={0}
                  className={colors.text}
                  suffix={suffix}
                />
              )
            ) : (
              <>
                {formattedValue}
                {suffix}
              </>
            )}
          </motion.div>
          {trend && typeof trend.value === 'number' && !isNaN(trend.value) && (
            <div className="flex items-center gap-1 text-xs mt-2">
              <span className={cn(
                'font-semibold',
                trend.isPositive ? 'text-green-600' : 'text-red-600'
              )}>
                {trend.isPositive ? '↑' : '↓'} {Math.abs(trend.value).toFixed(1)}%
              </span>
              <span className="text-muted-foreground">за период</span>
            </div>
          )}
        </CardContent>
      </Card>
    </motion.div>
  )
}

