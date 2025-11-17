'use client'

import { ReactNode } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { LucideIcon, ArrowRight } from 'lucide-react'

export interface ActionCardProps {
  title: string
  description: string
  icon: LucideIcon | ReactNode
  action?: {
    label: string
    onClick: () => void
    variant?: 'default' | 'outline' | 'ghost'
  }
  onClick?: () => void
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'danger'
  borderColor?: string
  className?: string
  disabled?: boolean
}

export function ActionCard({
  title,
  description,
  icon: Icon,
  action,
  onClick,
  variant = 'default',
  borderColor,
  className,
  disabled = false,
}: ActionCardProps) {
  const handleClick = () => {
    if (disabled) return
    onClick?.()
  }

  const variantStyles = {
    default: 'hover:shadow-lg',
    primary: 'border-blue-500 hover:border-blue-600',
    success: 'border-green-500 hover:border-green-600',
    warning: 'border-yellow-500 hover:border-yellow-600',
    danger: 'border-red-500 hover:border-red-600',
  }

  const iconColors = {
    default: 'text-muted-foreground',
    primary: 'text-blue-500',
    success: 'text-green-500',
    warning: 'text-yellow-500',
    danger: 'text-red-500',
  }

  return (
    <Card
      className={cn(
        'border-l-4 transition-all cursor-pointer',
        borderColor || variantStyles[variant],
        disabled && 'opacity-50 cursor-not-allowed',
        !disabled && 'hover:shadow-lg',
        className
      )}
      onClick={handleClick}
    >
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className={cn('h-8 w-8', iconColors[variant])}>
            {typeof Icon === 'function' ? (
              <Icon className="h-8 w-8" />
            ) : (
              Icon
            )}
          </div>
          {!disabled && <ArrowRight className="h-5 w-5 text-muted-foreground" />}
        </div>
        <CardTitle className="mt-4">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      {action && (
        <CardContent>
          <Button
            variant={action.variant || 'outline'}
            className="w-full"
            onClick={(e) => {
              e.stopPropagation()
              action.onClick()
            }}
            disabled={disabled}
          >
            {action.label}
          </Button>
        </CardContent>
      )}
    </Card>
  )
}

