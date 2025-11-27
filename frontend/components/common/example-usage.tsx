/**
 * Примеры использования новых компонентов
 * 
 * Этот файл содержит примеры использования:
 * - AnimationProvider
 * - useAnimatedState
 * - AnimatedCard
 * - AnimatedList
 * - Design Tokens
 */

'use client'

import { useState } from 'react'
import { AnimatedCard } from '@/components/common/animated-card'
import { AnimatedList } from '@/components/common/animated-list'
import { useAnimatedState, useAnimatedList } from '@/hooks/use-animated-state'
import { useAnimationContext } from '@/providers/animation-provider'
import { tokens } from '@/styles/tokens'
import { Button } from '@/components/ui/button'
import { motion } from 'framer-motion'

/**
 * Пример 1: Использование AnimatedCard
 */
export function ExampleAnimatedCard() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <AnimatedCard 
        title="Заголовок 1" 
        description="Описание карточки"
        variant="hover"
        delay={0}
      >
        <p>Содержимое карточки 1</p>
      </AnimatedCard>
      
      <AnimatedCard 
        title="Заголовок 2" 
        description="Описание карточки"
        variant="scale"
        delay={0.1}
      >
        <p>Содержимое карточки 2</p>
      </AnimatedCard>
      
      <AnimatedCard 
        title="Заголовок 3" 
        description="Описание карточки"
        variant="default"
        delay={0.2}
      >
        <p>Содержимое карточки 3</p>
      </AnimatedCard>
    </div>
  )
}

/**
 * Пример 2: Использование AnimatedList
 */
export function ExampleAnimatedList() {
  const items = [
    { id: '1', name: 'Элемент 1' },
    { id: '2', name: 'Элемент 2' },
    { id: '3', name: 'Элемент 3' },
  ]

  return (
    <AnimatedList
      items={items}
      keyExtractor={(item) => item.id}
      renderItem={(item, index) => (
        <div className="p-4 border rounded-lg">
          {index + 1}. {item.name}
        </div>
      )}
      staggerDelay={0.1}
      emptyMessage="Список пуст"
    />
  )
}

/**
 * Пример 3: Использование useAnimatedState
 */
export function ExampleAnimatedState() {
  const { state, setAnimatedState, isVisible } = useAnimatedState('Начальное состояние')
  const [counter, setCounter] = useState(0)

  const handleChange = () => {
    setCounter(c => c + 1)
    setAnimatedState(`Состояние ${counter + 1}`)
  }

  return (
    <div className="space-y-4">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: isVisible ? 1 : 0 }}
        transition={{ duration: 0.3 }}
      >
        {state}
      </motion.div>
      <Button onClick={handleChange}>
        Изменить состояние
      </Button>
    </div>
  )
}

/**
 * Пример 4: Использование useAnimatedList
 */
export function ExampleAnimatedListHook() {
  const { items, setItems, isVisible } = useAnimatedList(['item1', 'item2'])

  const addItem = () => {
    setItems([...items, `item${items.length + 1}`])
  }

  return (
    <div className="space-y-4">
      <AnimatePresence>
        {isVisible && items.map((item, index) => (
          <motion.div
            key={item}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ delay: index * 0.1 }}
          >
            {item}
          </motion.div>
        ))}
      </AnimatePresence>
      <Button onClick={addItem}>
        Добавить элемент
      </Button>
    </div>
  )
}

/**
 * Пример 5: Использование токенов дизайна
 */
export function ExampleDesignTokens() {
  return (
    <div 
      className="p-6 rounded-lg"
      style={{
        padding: tokens.spacing.lg,
        backgroundColor: tokens.color.primary[50],
        color: tokens.color.primary[900],
        borderRadius: tokens.borderRadius.lg,
        boxShadow: tokens.shadows.md,
        gap: tokens.spacing.md,
      }}
    >
      <h2 style={{ fontSize: tokens.typography.fontSize['2xl'] }}>
        Заголовок с токенами
      </h2>
      <p style={{ fontSize: tokens.typography.fontSize.base }}>
        Текст с использованием токенов дизайна
      </p>
    </div>
  )
}

/**
 * Пример 6: Управление анимациями через AnimationProvider
 */
export function ExampleAnimationControl() {
  const { variant, setVariant, animations } = useAnimationContext()

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4">
        <span>Вариант анимации: {variant}</span>
        <select
          value={variant}
          onChange={(e) => setVariant(e.target.value as any)}
          className="px-3 py-1 border rounded-md"
        >
          <option value="gentle">Плавный</option>
          <option value="fast">Быстрый</option>
          <option value="none">Без анимаций</option>
        </select>
      </div>
      <div className="text-sm text-muted-foreground">
        Длительность: {animations[variant].duration}s
      </div>
    </div>
  )
}

import { AnimatePresence } from 'framer-motion'

