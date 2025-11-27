'use client'

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { motion, AnimatePresence, type Variants } from 'framer-motion'
import { prefersReducedMotion } from '@/lib/animations'

type AnimationVariant = 'gentle' | 'fast' | 'none'

interface AnimationConfig {
  duration: number
  ease: number[]
}

interface AnimationContextType {
  variant: AnimationVariant
  setVariant: (variant: AnimationVariant) => void
  animations: {
    gentle: AnimationConfig
    fast: AnimationConfig
    none: AnimationConfig
  }
  getAnimationConfig: () => AnimationConfig
}

export const animations: Record<AnimationVariant, AnimationConfig> = {
  gentle: { duration: 0.4, ease: [0.25, 0.1, 0.25, 1] },
  fast: { duration: 0.2, ease: [0.4, 0, 1, 1] },
  none: { duration: 0, ease: [0, 0, 0, 0] },
}

export const AnimationContext = createContext<AnimationContextType | undefined>(undefined)

export function useAnimationContext() {
  const context = useContext(AnimationContext)
  if (!context) {
    throw new Error('useAnimationContext must be used within AnimationProvider')
  }
  return context
}

interface AnimationProviderProps {
  children: ReactNode
  defaultVariant?: AnimationVariant
}

export function AnimationProvider({ 
  children, 
  defaultVariant = 'gentle' 
}: AnimationProviderProps) {
  const [variant, setVariant] = useState<AnimationVariant>(defaultVariant)
  const [reducedMotion, setReducedMotion] = useState(false)

  useEffect(() => {
    // Проверяем prefers-reduced-motion при монтировании
    setReducedMotion(prefersReducedMotion())

    // Слушаем изменения в медиа-запросе
    if (typeof window !== 'undefined') {
      const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
      const handleChange = (e: MediaQueryListEvent) => {
        setReducedMotion(e.matches)
        if (e.matches && variant !== 'none') {
          setVariant('none')
        }
      }

      // Современные браузеры
      if (mediaQuery.addEventListener) {
        mediaQuery.addEventListener('change', handleChange)
        return () => mediaQuery.removeEventListener('change', handleChange)
      } 
      // Старые браузеры
      else if (mediaQuery.addListener) {
        mediaQuery.addListener(handleChange)
        return () => mediaQuery.removeListener(handleChange)
      }
    }
  }, [variant])

  // Автоматически переключаемся на 'none' если пользователь предпочитает уменьшенные анимации
  useEffect(() => {
    if (reducedMotion && variant !== 'none') {
      setVariant('none')
    }
  }, [reducedMotion])

  const getAnimationConfig = (): AnimationConfig => {
    if (reducedMotion) {
      return animations.none
    }
    return animations[variant]
  }

  const value: AnimationContextType = {
    variant: reducedMotion ? 'none' : variant,
    setVariant: (newVariant) => {
      if (!reducedMotion) {
        setVariant(newVariant)
      }
    },
    animations,
    getAnimationConfig,
  }

  return (
    <AnimationContext.Provider value={value}>
      <motion.div
        initial={false}
        animate="visible"
        variants={{
          visible: { opacity: 1 },
        }}
        style={{
          // Применяем CSS переменные для удобства использования в CSS
          ['--animation-duration' as string]: `${getAnimationConfig().duration}s`,
        }}
      >
        {children}
      </motion.div>
    </AnimationContext.Provider>
  )
}

// Вспомогательная функция для получения вариантов анимации
export function getAnimationVariants(config: AnimationConfig): Variants {
  return {
    hidden: {
      opacity: 0,
      y: 20,
    },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: config.duration,
        ease: config.ease as [number, number, number, number],
      },
    },
    exit: {
      opacity: 0,
      y: -20,
      transition: {
        duration: config.duration * 0.5, // Быстрее при выходе
        ease: config.ease as [number, number, number, number],
      },
    },
  }
}

// Экспорт для использования в компонентах
export { AnimatePresence, motion }

