'use client'

import { useEffect, useRef } from 'react'
import { useDashboardStore } from '@/stores/dashboard-store'
import { toast } from 'sonner'

/**
 * Компонент для автоматического отображения уведомлений из store через toast
 */
export function NotificationToast() {
  const { notifications } = useDashboardStore()
  const processedIdsRef = useRef<Set<string>>(new Set())

  useEffect(() => {
    // Безопасная проверка массива уведомлений
    if (!Array.isArray(notifications)) {
      return
    }

    const unreadNotifications = notifications.filter(
      n => n && typeof n === 'object' && !n.read && n.id && !processedIdsRef.current.has(n.id)
    )

    unreadNotifications.forEach(notification => {
      try {
        if (!notification || !notification.id) {
          return
        }

        processedIdsRef.current.add(notification.id)

        const toastOptions = {
          description: notification.message || '',
          duration: notification.type === 'error' ? 7000 : 5000,
        }

        const title = notification.title || 'Уведомление'

        switch (notification.type) {
          case 'success':
            toast.success(title, toastOptions)
            break
          case 'error':
            toast.error(title, toastOptions)
            break
          case 'warning':
            toast.warning(title, toastOptions)
            break
          default:
            toast.info(title, toastOptions)
        }

        // Помечаем как прочитанное
        try {
          useDashboardStore.getState().markNotificationAsRead(notification.id)
        } catch {
          // Игнорируем ошибки пометки как прочитанное
        }
      } catch {
        // Игнорируем ошибки обработки уведомления
      }
    })
  }, [notifications])

  return null
}

