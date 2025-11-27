'use client'

import { useState, useEffect, useCallback, useContext } from 'react'
import { AnimationContext } from '@/providers/animation-provider'

/**
 * Хук для плавной анимации изменений состояния
 * 
 * @template T - Тип состояния
 * @param initialState - Начальное состояние
 * @returns Объект с состоянием, функцией для его изменения и флагом видимости
 * 
 * @example
 * ```tsx
 * const { state, setAnimatedState, isVisible } = useAnimatedState('initial')
 * 
 * // Изменение состояния с анимацией
 * setAnimatedState('new state')
 * 
 * // Отображение только когда элемент видим
 * {isVisible && <Component data={state} />}
 * ```
 */
export function useAnimatedState<T>(initialState: T) {
  const [state, setState] = useState<T>(initialState)
  const [isVisible, setIsVisible] = useState(true)
  const animationContext = useContext(AnimationContext)
  const config = animationContext?.getAnimationConfig() || { duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }

  const setAnimatedState = useCallback((newState: T) => {
    // Если анимации отключены, сразу обновляем состояние
    if (config.duration === 0) {
      setState(newState)
      setIsVisible(true)
      return
    }

    // Сначала скрываем
    setIsVisible(false)
    
    // Затем обновляем состояние после завершения анимации выхода
    setTimeout(() => {
      setState(newState)
      // Показываем с новым состоянием после небольшой задержки
      setTimeout(() => {
        setIsVisible(true)
      }, 10)
    }, config.duration * 500) // Половина длительности для плавности
  }, [config.duration])

  return { 
    state, 
    setAnimatedState, 
    isVisible,
    animationDuration: config.duration,
  }
}

/**
 * Хук для анимации списка элементов при изменении
 * 
 * @template T - Тип элементов списка
 * @param initialItems - Начальный список элементов
 * @returns Объект с элементами списка, функцией для их изменения и флагом видимости
 * 
 * @example
 * ```tsx
 * const { items, setItems, isVisible } = useAnimatedList(['item1', 'item2'])
 * 
 * // Изменение списка с анимацией
 * setItems(['item3', 'item4'])
 * 
 * // Использование с AnimatePresence
 * <AnimatePresence>
 *   {isVisible && items.map(item => <Item key={item} item={item} />)}
 * </AnimatePresence>
 * ```
 */
export function useAnimatedList<T>(initialItems: T[]) {
  const [items, setItemsState] = useState<T[]>(initialItems)
  const [isVisible, setIsVisible] = useState(true)
  const animationContext = useContext(AnimationContext)
  const config = animationContext?.getAnimationConfig() || { duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }

  const setItems = useCallback((newItems: T[]) => {
    // Если анимации отключены, сразу обновляем список
    if (config.duration === 0) {
      setItemsState(newItems)
      setIsVisible(true)
      return
    }

    // Сначала скрываем
    setIsVisible(false)
    
    // Затем обновляем список после завершения анимации выхода
    setTimeout(() => {
      setItemsState(newItems)
      // Показываем с новым списком после небольшой задержки
      setTimeout(() => {
        setIsVisible(true)
      }, 10)
    }, config.duration * 500)
  }, [config.duration])

  return { 
    items, 
    setItems, 
    isVisible,
    animationDuration: config.duration,
  }
}

