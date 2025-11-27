/**
 * Утилиты для оптимизации анимаций
 * Интегрировано с AnimationProvider для единого управления
 */

import { getAnimationVariants } from '@/providers/animation-provider'
import type { Variants } from 'framer-motion'

// Оптимизированные варианты анимаций для лучшей производительности
// Эти варианты будут автоматически адаптироваться к настройкам AnimationProvider
export const animationVariants = {
  fadeIn: {
    initial: { opacity: 0 },
    animate: { opacity: 1 },
    exit: { opacity: 0 },
    transition: { duration: 0.3, ease: 'easeOut' },
  },
  slideUp: {
    initial: { opacity: 0, y: 20 },
    animate: { opacity: 1, y: 0 },
    exit: { opacity: 0, y: -20 },
    transition: { duration: 0.3, ease: 'easeOut' },
  },
  scaleIn: {
    initial: { opacity: 0, scale: 0.95 },
    animate: { opacity: 1, scale: 1 },
    exit: { opacity: 0, scale: 0.95 },
    transition: { duration: 0.2, ease: 'easeOut' },
  },
  stagger: {
    container: {
      initial: 'hidden',
      animate: 'visible',
      variants: {
        visible: {
          transition: {
            staggerChildren: 0.1,
          },
        },
      },
    },
    item: {
      hidden: { opacity: 0, y: 20 },
      visible: { 
        opacity: 1, 
        y: 0,
        transition: { duration: 0.3, ease: 'easeOut' },
      },
    },
  },
} as const

/**
 * Получить варианты анимации с учетом настроек AnimationProvider
 * Используйте эту функцию для анимаций, которые должны соответствовать глобальным настройкам
 */
export function getProviderAnimationVariants(): Variants {
  // По умолчанию возвращаем базовые варианты
  // В компонентах можно использовать useAnimationContext для получения актуальных настроек
  return getAnimationVariants({ duration: 0.3, ease: [0.25, 0.1, 0.25, 1] })
}

// Проверка поддержки prefers-reduced-motion для доступности
export const prefersReducedMotion = () => {
  if (typeof window === 'undefined') return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

// Упрощенные анимации для пользователей с prefers-reduced-motion
export const getAnimationProps = (variant: keyof typeof animationVariants) => {
  if (prefersReducedMotion()) {
    // Минимальные анимации для доступности
    return {
      initial: { opacity: 0 },
      animate: { opacity: 1 },
      exit: { opacity: 0 },
      transition: { duration: 0.1 },
    }
  }
  return animationVariants[variant]
}
