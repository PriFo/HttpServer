'use client'

import { motion } from 'framer-motion'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { LucideIcon } from 'lucide-react'
import { AnimatedNumber } from './AnimatedNumber'
import { cn } from '@/lib/utils'

interface StatsWidgetProps {
  title: string
  value: number
  icon: LucideIcon
  color?: 'blue' | 'green' | 'yellow' | 'red' | 'purple'
  trend?: {
    value: number
    isPositive: boolean
  }
  suffix?: string
  decimals?: number
}

const colorClasses = {
  blue: 'text-blue-600 bg-blue-50 dark:bg-blue-950',
  green: 'text-green-600 bg-green-50 dark:bg-green-950',
  yellow: 'text-yellow-600 bg-yellow-50 dark:bg-yellow-950',
  red: 'text-red-600 bg-red-50 dark:bg-red-950',
  purple: 'text-purple-600 bg-purple-50 dark:bg-purple-950',
}

export function StatsWidget({
  title,
  value,
  icon: Icon,
  color = 'blue',
  trend,
  suffix = '',
  decimals = 0,
}: StatsWidgetProps) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      whileHover={{ scale: 1.02 }}
      transition={{ duration: 0.2 }}
    >
      <Card className="relative overflow-hidden">
        <div className={cn('absolute top-0 right-0 w-32 h-32 rounded-full blur-3xl opacity-20', colorClasses[color])} />
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium">{title}</CardTitle>
            <div className={cn('p-2 rounded-lg', colorClasses[color])}>
              <Icon className="h-4 w-4" />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className={cn('text-3xl font-bold mb-1', colorClasses[color].split(' ')[0])}>
            <AnimatedNumber value={value} duration={0.8} decimals={decimals} suffix={suffix} />
          </div>
          {trend && (
            <div className="flex items-center gap-1 text-xs mt-2">
              <span className={cn(
                'font-semibold',
                trend.isPositive ? 'text-green-600' : 'text-red-600'
              )}>
                {trend.isPositive ? '↑' : '↓'} {Math.abs(trend.value)}%
              </span>
              <span className="text-muted-foreground">за период</span>
            </div>
          )}
        </CardContent>
      </Card>
    </motion.div>
  )
}

