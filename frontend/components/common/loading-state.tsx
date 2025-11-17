'use client'

import * as React from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Card, CardContent } from '@/components/ui/card'

export interface LoadingStateProps {
  message?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
  fullScreen?: boolean
  variant?: 'default' | 'card' | 'minimal'
}

export function LoadingState({
  message = 'Загрузка...',
  size = 'md',
  className,
  fullScreen = false,
  variant = 'default',
}: LoadingStateProps) {
  const sizeClasses = {
    sm: 'h-4 w-4',
    md: 'h-8 w-8',
    lg: 'h-12 w-12',
  }

  const spinner = (
    <Loader2 className={cn('animate-spin text-primary', sizeClasses[size])} />
  )

  const text = message && (
    <p className="text-sm text-muted-foreground">{message}</p>
  )

  let content: React.ReactNode

  if (variant === 'card') {
    content = (
      <Card className={cn(className)}>
        <CardContent className="pt-6">
          <div className="flex flex-col items-center gap-2 py-8">
            {spinner}
            {text}
          </div>
        </CardContent>
      </Card>
    )
  } else if (variant === 'minimal') {
    content = spinner
  } else {
    content = (
      <div className={cn('flex flex-col items-center gap-2', className)}>
        {spinner}
        {text}
      </div>
    )
  }

  if (fullScreen) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        {content}
      </div>
    )
  }

  return content
}

