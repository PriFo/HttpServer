'use client'

import { motion, AnimatePresence } from 'framer-motion'
import { getAnimationVariants } from '@/providers/animation-provider'
import { cn } from '@/lib/utils'
import { useContext } from 'react'
import { AnimationContext } from '@/providers/animation-provider'

interface AnimatedListProps {
  items: any[]
  renderItem: (item: any, index: number) => React.ReactNode
  keyExtractor: (item: any, index: number) => string | number
  className?: string
  itemClassName?: string
  staggerDelay?: number
  emptyMessage?: string
}

/**
 * Анимированный список с поддержкой AnimationProvider
 * 
 * @example
 * ```tsx
 * <AnimatedList
 *   items={items}
 *   keyExtractor={(item) => item.id}
 *   renderItem={(item, index) => <Item key={item.id} item={item} />}
 *   emptyMessage="Список пуст"
 * />
 * ```
 */
export function AnimatedList({
  items,
  renderItem,
  keyExtractor,
  className,
  itemClassName,
  staggerDelay = 0.05,
  emptyMessage,
}: AnimatedListProps) {
  // Используем контекст с fallback для случаев, когда он не доступен
  const animationContext = useContext(AnimationContext)
  const config = animationContext?.getAnimationConfig() || { duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }
  const variants = getAnimationVariants(config)

  if (items.length === 0 && emptyMessage) {
    return (
      <div className={cn("text-center py-8 text-muted-foreground", className)}>
        {emptyMessage}
      </div>
    )
  }

  return (
    <motion.div
      initial="hidden"
      animate="visible"
      variants={{
        visible: {
          transition: {
            staggerChildren: staggerDelay,
          },
        },
      }}
      className={className}
    >
      <AnimatePresence mode="popLayout">
        {items.map((item, index) => (
          <motion.div
            key={keyExtractor(item, index)}
            initial="hidden"
            animate="visible"
            exit="exit"
            variants={variants}
            className={itemClassName}
          >
            {renderItem(item, index)}
          </motion.div>
        ))}
      </AnimatePresence>
    </motion.div>
  )
}

