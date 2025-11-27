/**
 * Тесты для Error Boundary
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ErrorBoundary } from '../error-boundary'
import { logger } from '@/lib/logger'

// Компонент, который выбрасывает ошибку
const ThrowError = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) {
    throw new Error('Test error')
  }
  return <div>No error</div>
}

describe('ErrorBoundary', () => {
  beforeEach(() => {
    // Подавляем ошибки в консоли во время тестов
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.spyOn(logger, 'error').mockImplementation(() => {})
  })

  it('should render children when there is no error', () => {
    render(
      <ErrorBoundary>
        <div>Test content</div>
      </ErrorBoundary>
    )

    expect(screen.getByText('Test content')).toBeInTheDocument()
  })

  it('should catch and display error', () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText(/Произошла ошибка/i)).toBeInTheDocument()
  })

  it('should log error when caught', () => {
    const logSpy = vi.spyOn(logger, 'error')
    
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(logSpy).toHaveBeenCalled()
  })

  it('should call custom onError handler', () => {
    const onError = vi.fn()
    
    render(
      <ErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(onError).toHaveBeenCalled()
  })

  it('should reset error boundary', () => {
    const { rerender } = render(
      <ErrorBoundary resetKeys={[1]}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText(/Произошла ошибка/i)).toBeInTheDocument()

    // Изменяем resetKeys, что должно сбросить ошибку
    rerender(
      <ErrorBoundary resetKeys={[2]}>
        <ThrowError shouldThrow={false} />
      </ErrorBoundary>
    )

    expect(screen.queryByText(/Произошла ошибка/i)).not.toBeInTheDocument()
  })

  it('should render custom fallback', () => {
    const fallback = <div>Custom fallback</div>
    
    render(
      <ErrorBoundary fallback={fallback}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    expect(screen.getByText('Custom fallback')).toBeInTheDocument()
  })

  it('should show stack trace in development', () => {
    const originalEnv = process.env.NODE_ENV
    process.env.NODE_ENV = 'development'

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    )

    // В dev режиме должен быть stack trace
    const details = screen.queryByText(/Stack trace/i)
    expect(details).toBeInTheDocument()

    process.env.NODE_ENV = originalEnv
  })
})

