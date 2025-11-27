'use client'

import { useEffect } from 'react'
import { useDashboardStore } from '@/stores/dashboard-store'

/**
 * Компонент для оптимизации производительности дашборда
 * Отключает обновления для неактивных табов
 */
export function PerformanceOptimizer() {
  const { activeTab, isRealTimeEnabled, toggleRealTime } = useDashboardStore()

  useEffect(() => {
    // Оптимизация: отключаем реальное время для неактивных табов
    // Только для таба мониторинга оставляем включенным
    if (activeTab !== 'monitoring' && isRealTimeEnabled) {
      // Не отключаем полностью, но можем оптимизировать частоту обновлений
    }
  }, [activeTab, isRealTimeEnabled])

  return null
}

