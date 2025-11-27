'use client'

import { useEffect } from 'react'
import { DashboardHeader } from '@/components/dashboard/DashboardHeader'
import { DashboardSidebar } from '@/components/dashboard/DashboardSidebar'
import { MainContentArea } from '@/components/dashboard/MainContentArea'
import { PerformanceOptimizer } from '@/components/dashboard/PerformanceOptimizer'
import { KeyboardShortcuts } from '@/components/dashboard/KeyboardShortcuts'
import { NotificationToast } from '@/components/dashboard/NotificationToast'
import { useRealTimeData } from '@/hooks/useRealTimeData'
import { useDashboardStore } from '@/stores/dashboard-store'

export default function DashboardPage() {
  const { setLoading, addNotification } = useDashboardStore()
  
  // Подключаемся к реальному времени
  useRealTimeData()

  useEffect(() => {
    // Скрываем стандартный Header и Footer на главной странице
    const mainHeader = document.getElementById('main-header')
    const mainFooter = document.getElementById('main-footer')
    
    if (mainHeader) mainHeader.style.display = 'none'
    if (mainFooter) mainFooter.style.display = 'none'

    // Инициализация страницы
    setLoading(false)
    
    // Приветственное уведомление
    addNotification({
      type: 'info',
      title: 'Добро пожаловать!',
      message: 'Добро пожаловать в миссионный центр системы нормализации данных',
    })

    // Cleanup при размонтировании
    return () => {
      if (mainHeader) mainHeader.style.display = ''
      if (mainFooter) mainFooter.style.display = ''
    }
  }, [setLoading, addNotification])

  return (
    <div className="min-h-screen bg-background">
      <PerformanceOptimizer />
      <KeyboardShortcuts />
      <NotificationToast />
      <DashboardHeader />
      <DashboardSidebar />
      <MainContentArea />
    </div>
  )
}
