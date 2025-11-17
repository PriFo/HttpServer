'use client'

import { ReactNode } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { AlertCircle, RefreshCw, X } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface ErrorStateProps {
  title?: string
  message: string
  icon?: ReactNode
  action?: {
    label: string
    onClick: () => void
  }
  variant?: 'default' | 'destructive' | 'warning'
  className?: string
  fullScreen?: boolean
  dismissible?: boolean
  onDismiss?: () => void
}

export function ErrorState({
  title,
  message,
  icon,
  action,
  variant = 'destructive',
  className,
  fullScreen = false,
  dismissible = false,
  onDismiss,
}: ErrorStateProps) {
  // Alert component only supports 'default' and 'destructive'
  const alertVariant = variant === 'warning' ? 'default' : variant
  
  const content = (
    <Alert 
      variant={alertVariant} 
      className={cn(
        dismissible && 'pr-10',
        variant === 'warning' && 'border-yellow-500 bg-yellow-50 dark:bg-yellow-950',
        className
      )}
    >
      {icon || <AlertCircle className="h-4 w-4" />}
      <AlertTitle className={cn(variant === 'warning' && 'text-yellow-800 dark:text-yellow-200')}>
        {title || 'Произошла ошибка'}
      </AlertTitle>
      <AlertDescription className={cn(
        'flex items-center justify-between',
        variant === 'warning' && 'text-yellow-800 dark:text-yellow-200'
      )}>
        <span className="flex-1">{message}</span>
        {action && (
          <Button
            variant="outline"
            size="sm"
            onClick={action.onClick}
            className={cn(
              'ml-4',
              variant === 'warning' && 'border-yellow-600 text-yellow-800 hover:bg-yellow-100 dark:text-yellow-200 dark:hover:bg-yellow-900'
            )}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            {action.label}
          </Button>
        )}
      </AlertDescription>
      {dismissible && onDismiss && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          className={cn(
            'absolute right-2 top-2 h-6 w-6 p-0',
            variant === 'warning' && 'text-yellow-800 hover:text-yellow-900 dark:text-yellow-200 dark:hover:text-yellow-100'
          )}
        >
          <X className="h-3 w-3" />
        </Button>
      )}
    </Alert>
  )

  if (fullScreen) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="w-full max-w-2xl relative">{content}</div>
      </div>
    )
  }

  return <div className="relative">{content}</div>
}

