'use client'

import React, { createContext, useContext, useCallback } from 'react'
import { toast } from 'sonner'
import { AppError, createUnknownError, logError } from '@/lib/errors'

type ErrorContextType = {
  handleError: (error: unknown, fallbackMessage?: string) => void
}

const ErrorContext = createContext<ErrorContextType | undefined>(undefined)

export const ErrorProvider = ({ children }: { children: React.ReactNode }) => {
  const handleError = useCallback((error: unknown, fallbackMessage = 'Произошла ошибка') => {
    const appError = createUnknownError(error)

    // Если у ошибки есть пользовательское сообщение, используем его
    // Иначе используем fallbackMessage
    const userMessage = appError.message || fallbackMessage

    // Показываем пользователю дружественное сообщение через toast
    toast.error(userMessage, {
      description: appError.technicalDetails && process.env.NODE_ENV === 'development' 
        ? appError.technicalDetails 
        : undefined,
      duration: 5000,
    })

    // Логируем технические детали в консоль для разработчика
    logError(appError, {
      userMessage,
      fallbackMessage,
      url: typeof window !== 'undefined' ? window.location.href : undefined,
    })
  }, [])

  return (
    <ErrorContext.Provider value={{ handleError }}>
      {children}
    </ErrorContext.Provider>
  )
}

export const useError = () => {
  const context = useContext(ErrorContext)
  if (!context) {
    throw new Error('useError must be used within an ErrorProvider')
  }
  return context
}
