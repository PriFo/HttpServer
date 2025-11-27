/**
 * Анимированная карточка с использованием токенов дизайна
 * 
 * Этот компонент является оберткой над AnimatedCard с предустановленными
 * стилями из дизайн-токенов
 */

'use client'

import { AnimatedCard } from '@/components/common/animated-card'
import { tokens } from '@/styles/tokens'
import { cn } from '@/lib/utils'

interface AnimatedCardWithTokensProps {
  children: React.ReactNode
  title?: string
  description?: string
  delay?: number
  variant?: 'default' | 'hover' | 'scale'
  className?: string
  padding?: keyof typeof tokens.spacing
  shadow?: keyof typeof tokens.shadows
}

/**
 * Анимированная карточка с применением токенов дизайна
 * 
 * @example
 * ```tsx
 * <AnimatedCardWithTokens
 *   title="Заголовок"
 *   padding="lg"
 *   shadow="md"
 * >
 *   Контент
 * </AnimatedCardWithTokens>
 * ```
 */
export function AnimatedCardWithTokens({
  children,
  title,
  description,
  delay = 0,
  variant = 'default',
  className,
  padding = 'md',
  shadow = 'md',
}: AnimatedCardWithTokensProps) {
  return (
    <AnimatedCard
      title={title}
      description={description}
      delay={delay}
      variant={variant}
      className={cn(
        className,
        'shadow-[var(--token-shadow-md)]'
      )}
      style={{
        padding: tokens.spacing[padding],
        boxShadow: tokens.shadows[shadow],
        borderRadius: tokens.borderRadius.lg,
      }}
    >
      {children}
    </AnimatedCard>
  )
}

