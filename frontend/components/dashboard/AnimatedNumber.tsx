'use client'

import { useEffect, useState } from 'react'
import { motion, animate } from 'framer-motion'

interface AnimatedNumberProps {
  value: number
  duration?: number
  decimals?: number
  className?: string
  prefix?: string
  suffix?: string
}

export function AnimatedNumber({
  value,
  duration = 0.8,
  decimals = 0,
  className,
  prefix = '',
  suffix = '',
}: AnimatedNumberProps) {
  // Безопасная нормализация значения
  const safeValue = typeof value === 'number' && !isNaN(value) && isFinite(value) ? value : 0
  const safeDecimals = typeof decimals === 'number' && decimals >= 0 ? Math.floor(decimals) : 0
  
  const [displayValue, setDisplayValue] = useState(safeValue)

  useEffect(() => {
    // Используем начальное значение для плавного перехода
    const startValue = typeof displayValue === 'number' && !isNaN(displayValue) ? displayValue : 0
    const targetValue = safeValue
    
    // Проверяем, что значения валидны перед анимацией
    if (isNaN(startValue) || isNaN(targetValue) || !isFinite(startValue) || !isFinite(targetValue)) {
      setDisplayValue(targetValue)
      return
    }
    
    const controls = animate(startValue, targetValue, {
      duration,
      ease: [0.4, 0, 0.2, 1], // Custom easing для более естественного движения
      onUpdate: (latest) => {
        if (typeof latest === 'number' && !isNaN(latest) && isFinite(latest)) {
          setDisplayValue(latest)
        }
      },
    })

    return () => controls.stop()
  }, [safeValue, duration])

  const safeDisplayValue = typeof displayValue === 'number' && !isNaN(displayValue) && isFinite(displayValue) 
    ? displayValue 
    : 0
  const formatted = safeDisplayValue.toFixed(safeDecimals).replace(/\B(?=(\d{3})+(?!\d))/g, ' ')

  return (
    <motion.span 
      className={className}
      key={value} // Пересоздаем при изменении значения для анимации
      initial={{ scale: 1.1 }}
      animate={{ scale: 1 }}
      transition={{ duration: 0.2 }}
    >
      {prefix}
      {formatted}
      {suffix}
    </motion.span>
  )
}

