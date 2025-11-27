'use client'

import { useEffect } from 'react'
import { useError } from '@/contexts/ErrorContext'
import { createUnknownError, logError } from '@/lib/errors'

/**
 * Компонент для установки глобальных обработчиков ошибок
 * Обрабатывает необработанные промисы и глобальные JS ошибки
 * 
 * Должен быть размещен внутри ErrorProvider
 */
export function GlobalErrorHandlers() {
  const { handleError } = useError()

  useEffect(() => {
    // Обработчик необработанных промисов
    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      // Предотвращаем вывод ошибки в консоль браузера по умолчанию
      event.preventDefault()

      const error = createUnknownError(event.reason)
      
      // Логируем для разработчика
      logError(error, {
        type: 'unhandledRejection',
        reason: event.reason,
        url: window.location.href,
      })

      // Показываем пользователю дружественное сообщение
      handleError(error, 'Произошла ошибка при выполнении операции')
    }

    // Обработчик глобальных JS ошибок
    const handleErrorEvent = (event: ErrorEvent) => {
      // Предотвращаем вывод ошибки в консоль браузера по умолчанию
      event.preventDefault()

      const error = createUnknownError(event.error || event.message)
      
      // Логируем для разработчика
      logError(error, {
        type: 'globalError',
        message: event.message,
        filename: event.filename,
        lineno: event.lineno,
        colno: event.colno,
        url: window.location.href,
      })

      // Показываем пользователю дружественное сообщение
      handleError(error, 'Произошла непредвиденная ошибка')
    }

    // Устанавливаем обработчики
    window.addEventListener('unhandledrejection', handleUnhandledRejection)
    window.addEventListener('error', handleErrorEvent)

    // Очистка при размонтировании
    return () => {
      window.removeEventListener('unhandledrejection', handleUnhandledRejection)
      window.removeEventListener('error', handleErrorEvent)
    }
  }, [handleError])

  // Этот компонент не рендерит ничего
  return null
}
