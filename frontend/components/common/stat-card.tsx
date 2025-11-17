'use client'

import { ReactNode, isValidElement } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'
import { LucideIcon } from 'lucide-react'

export interface StatCardProps {
  title: string
  value: string | number
  description?: string
  icon?: LucideIcon | ReactNode
  trend?: {
    value: number
    label: string
    isPositive?: boolean
  }
  progress?: number
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'danger'
  className?: string
  onClick?: () => void
  formatValue?: (value: number | string) => string
}

export function StatCard({
  title,
  value,
  description,
  icon: Icon,
  trend,
  progress,
  variant = 'default',
  className,
  onClick,
  formatValue,
}: StatCardProps) {
  const formattedValue =
    typeof value === 'number'
      ? formatValue
        ? formatValue(value)
        : value.toLocaleString('ru-RU')
      : formatValue && typeof value === 'string' && !isNaN(Number(value))
        ? formatValue(Number(value))
        : value

  const variantStyles = {
    default: '',
    primary: 'border-blue-500 bg-blue-50 dark:bg-blue-950',
    success: 'border-green-500 bg-green-50 dark:bg-green-950',
    warning: 'border-yellow-500 bg-yellow-50 dark:bg-yellow-950',
    danger: 'border-red-500 bg-red-50 dark:bg-red-950',
  }

  const iconColors = {
    default: 'text-muted-foreground',
    primary: 'text-blue-600',
    success: 'text-green-600',
    warning: 'text-yellow-600',
    danger: 'text-red-600',
  }

  const valueColors = {
    default: '',
    primary: 'text-blue-600',
    success: 'text-green-600',
    warning: 'text-yellow-600',
    danger: 'text-red-600',
  }

  return (
    <Card
      className={cn(
        'transition-all hover:shadow-md',
        variantStyles[variant],
        onClick && 'cursor-pointer',
        className
      )}
      onClick={onClick}
    >
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
          {Icon && (
            <div className={cn('flex items-center justify-center', iconColors[variant])}>
              {isValidElement(Icon) ? (
                // Уже созданный React элемент
                Icon
              ) : typeof Icon === 'string' || typeof Icon === 'number' ? (
                // Примитивные типы
                <>{Icon}</>
              ) : typeof Icon === 'function' ? (
                // React компонент (например, LucideIcon из lucide-react)
                <Icon className="h-4 w-4" />
              ) : (
                // Остальные случаи (объекты, массивы и т.д.) - не рендерим напрямую
                null
              )}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div className={cn('text-2xl font-bold', valueColors[variant])}>
          {formattedValue}
        </div>
        {description && (
          <p className="text-xs text-muted-foreground mt-1">{description}</p>
        )}
        {progress !== undefined && (
          <Progress value={progress} className="mt-2" />
        )}
        {trend && (
          <div className="mt-2 flex items-center gap-1 text-xs">
            <span
              className={cn(
                'font-medium',
                trend.isPositive !== false
                  ? 'text-green-600'
                  : 'text-red-600'
              )}
            >
              {trend.isPositive !== false ? '↑' : '↓'} {trend.value}
            </span>
            <span className="text-muted-foreground">{trend.label}</span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

