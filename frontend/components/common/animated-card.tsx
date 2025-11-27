'use client'

'use client'

import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { getAnimationVariants } from '@/providers/animation-provider'
import { cn } from '@/lib/utils'
import { useContext } from 'react'
import { AnimationContext } from '@/providers/animation-provider'

interface AnimatedCardProps {
  children: React.ReactNode
  title?: string
  description?: string
  delay?: number
  variant?: 'default' | 'hover' | 'scale'
  className?: string
  style?: React.CSSProperties
}

/**
 * Анимированная карточка с поддержкой AnimationProvider
 * 
 * @example
 * ```tsx
 * <AnimatedCard title="Заголовок" description="Описание">
 *   Содержимое карточки
 * </AnimatedCard>
 * ```
 */
export function AnimatedCard({
  children,
  title,
  description,
  delay = 0,
  variant = 'default',
  className,
  style,
}: AnimatedCardProps) {
  // Используем контекст с fallback для случаев, когда он не доступен
  const animationContext = useContext(AnimationContext)
  const config = animationContext?.getAnimationConfig() || { duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }
  const variants = getAnimationVariants(config)

  const hoverProps = variant === 'hover' ? {
    whileHover: { scale: 1.02, y: -4 },
    whileTap: { scale: 0.98 },
  } : {}

  const scaleProps = variant === 'scale' ? {
    initial: { opacity: 0, scale: 0.95 },
    animate: { opacity: 1, scale: 1 },
    exit: { opacity: 0, scale: 0.95 },
  } : variants

  const transition = {
    duration: config.duration,
    ease: config.ease as [number, number, number, number],
    delay: delay,
  }

  return (
    <motion.div
      initial="hidden"
      animate="visible"
      exit="exit"
      variants={scaleProps}
      transition={transition}
      {...hoverProps}
      className={cn("h-full", className)}
      style={style}
    >
      <Card className="h-full">
        {(title || description) && (
          <CardHeader>
            {title && <CardTitle>{title}</CardTitle>}
            {description && <CardDescription>{description}</CardDescription>}
          </CardHeader>
        )}
        <CardContent>
          {children}
        </CardContent>
      </Card>
    </motion.div>
  )
}

