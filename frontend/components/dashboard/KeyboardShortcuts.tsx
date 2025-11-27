'use client'

import { useEffect } from 'react'
import { useDashboardStore } from '@/stores/dashboard-store'

type TabType = 'overview' | 'monitoring' | 'processes' | 'quality' | 'clients'

const shortcuts: Record<string, TabType> = {
  '1': 'overview',
  '2': 'monitoring',
  '3': 'processes',
  '4': 'quality',
  '5': 'clients',
}

export function KeyboardShortcuts() {
  const { setActiveTab } = useDashboardStore()

  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      // Игнорируем, если пользователь вводит текст
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement ||
        (e.target as HTMLElement).isContentEditable
      ) {
        return
      }

      // Проверяем Alt/Ctrl для предотвращения конфликтов с браузерными горячими клавишами
      if (e.altKey || e.ctrlKey || e.metaKey) {
        return
      }

      const tab = shortcuts[e.key]
      if (tab) {
        e.preventDefault()
        setActiveTab(tab)
        
        // Визуальная обратная связь - можно добавить toast или анимацию
        // toast.info(`Переключено на: ${tab}`, { duration: 1000 })
      }
    }

    window.addEventListener('keydown', handleKeyPress)
    return () => window.removeEventListener('keydown', handleKeyPress)
  }, [setActiveTab])

  return null
}

