'use client'

import React from 'react'
import { ErrorState } from '@/components/common/error-state'
import { logError } from '@/lib/errors'

interface ErrorBoundaryProps {
  children: React.ReactNode
  fallback?: React.ComponentType<{ error: Error; resetError: () => void }>
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

/**
 * Error Boundary компонент для обработки ошибок рендеринга
 * 
 * @example
 * ```tsx
 * <ErrorBoundary>
 *   <YourComponent />
 * </ErrorBoundary>
 * ```
 */
export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Логируем ошибку для разработчика
    logError(error, {
      componentStack: errorInfo.componentStack,
      errorBoundary: true,
    })

    // Вызываем кастомный обработчик, если есть
    if (this.props.onError) {
      this.props.onError(error, errorInfo)
    }

    // Здесь можно добавить интеграцию с Sentry в будущем
    // if (typeof window !== 'undefined' && window.Sentry) {
    //   window.Sentry.captureException(error, {
    //     contexts: {
    //       react: {
    //         componentStack: errorInfo.componentStack,
    //       },
    //     },
    //   })
    // }
  }

  resetError = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError && this.state.error) {
      const FallbackComponent = this.props.fallback || DefaultErrorFallback
      return (
        <FallbackComponent
          error={this.state.error}
          resetError={this.resetError}
        />
      )
    }

    return this.props.children
  }
}

/**
 * Компонент по умолчанию для отображения ошибки
 */
const DefaultErrorFallback: React.FC<{ error: Error; resetError: () => void }> = ({ error, resetError }) => (
  <ErrorState
    title="Что-то пошло не так!"
    message="Произошла непредвиденная ошибка. Попробуйте перезагрузить страницу."
    fullScreen
    action={{
      label: 'Попробовать снова',
      onClick: resetError,
    }}
  />
)
