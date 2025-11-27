'use client'

import { useEffect } from 'react'
import confetti from 'canvas-confetti'

interface ConfettiEffectProps {
  trigger: boolean
  type?: 'success' | 'milestone' | 'celebration'
}

export function ConfettiEffect({ trigger, type = 'success' }: ConfettiEffectProps) {
  useEffect(() => {
    if (!trigger) return

    const duration = 3000
    const end = Date.now() + duration

    const config: confetti.Options = {
      particleCount: 100,
      startVelocity: 30,
      spread: 360,
      origin: { x: 0.5, y: 0.5 },
      colors: ['#3b82f6', '#10b981', '#8b5cf6', '#f59e0b', '#ef4444'],
    }

    if (type === 'milestone') {
      config.particleCount = 200
      config.startVelocity = 50
      config.colors = ['#3b82f6', '#10b981', '#8b5cf6', '#f59e0b', '#ef4444', '#ec4899', '#06b6d4']
    } else if (type === 'celebration') {
      config.particleCount = 150
      config.startVelocity = 40
      config.colors = ['#3b82f6', '#10b981', '#8b5cf6', '#f59e0b']
    }

    // Первый взрыв в центре
    confetti({
      ...config,
      origin: { x: 0.5, y: 0.5 },
    })

    // Дополнительные взрывы для milestone
    if (type === 'milestone') {
      setTimeout(() => {
        confetti({
          ...config,
          origin: { x: 0.2, y: 0.3 },
        })
      }, 300)
      setTimeout(() => {
        confetti({
          ...config,
          origin: { x: 0.8, y: 0.3 },
        })
      }, 600)
    }

    const interval = setInterval(() => {
      if (Date.now() > end) {
        clearInterval(interval)
        return
      }

      confetti({
        ...config,
        origin: {
          x: Math.random(),
          y: Math.random() * 0.5,
        },
      })
    }, 200)

    // Cleanup
    return () => clearInterval(interval)
  }, [trigger, type])

  return null
}

