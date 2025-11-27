/**
 * Тесты для системы обработки ошибок
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { handleError, withErrorHandling, createErrorHandler } from '../error-handler'
import { AppError } from '../errors'
import { logger } from '../logger'

// Мокаем sonner
vi.mock('sonner', () => ({
  toast: {
    error: vi.fn(),
  },
}))

describe('Error Handler', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('handleError', () => {
    it('should handle AppError', () => {
      const error = new AppError('User message', 'Technical details', 500, 'TEST_ERROR')
      const message = handleError(error, { logError: false, showToast: false })
      
      expect(message).toBe('User message')
    })

    it('should handle standard Error', () => {
      const error = new Error('Test error')
      const message = handleError(error, { logError: false, showToast: false })
      
      expect(message).toBeTruthy()
    })

    it('should handle string errors', () => {
      const message = handleError('String error', { logError: false, showToast: false })
      expect(message).toBe('String error')
    })

    it('should handle unknown errors', () => {
      const message = handleError({ some: 'object' }, { 
        logError: false, 
        showToast: false,
        fallbackMessage: 'Custom fallback'
      })
      expect(message).toBe('Custom fallback')
    })

    it('should log errors when logError is true', () => {
      const logSpy = vi.spyOn(logger, 'error').mockImplementation(() => {})
      const error = new Error('Test error')
      
      handleError(error, { logError: true, showToast: false })
      
      expect(logSpy).toHaveBeenCalled()
      logSpy.mockRestore()
    })

    it('should provide user-friendly messages for network errors', () => {
      const networkError = new Error('Failed to fetch')
      const message = handleError(networkError, { logError: false, showToast: false })
      
      expect(message).toContain('подключиться')
    })

    it('should provide user-friendly messages for timeout errors', () => {
      const timeoutError = new Error('timeout')
      const message = handleError(timeoutError, { logError: false, showToast: false })
      
      expect(message).toContain('время ожидания')
    })

    it('should provide user-friendly messages for HTTP errors', () => {
      const http404Error = new Error('404 Not Found')
      const message = handleError(http404Error, { logError: false, showToast: false })
      
      expect(message).toContain('не найден')
    })
  })

  describe('withErrorHandling', () => {
    it('should wrap async function and handle errors', async () => {
      const errorFn = vi.fn().mockRejectedValue(new Error('Test error'))
      const wrappedFn = withErrorHandling(errorFn, { logError: false, showToast: false })
      
      await expect(wrappedFn()).rejects.toThrow()
      expect(errorFn).toHaveBeenCalled()
    })

    it('should pass through successful results', async () => {
      const successFn = vi.fn().mockResolvedValue('success')
      const wrappedFn = withErrorHandling(successFn, { logError: false, showToast: false })
      
      const result = await wrappedFn()
      expect(result).toBe('success')
    })

    it('should include function context in logs', async () => {
      const logSpy = vi.spyOn(logger, 'error').mockImplementation(() => {})
      const errorFn = vi.fn().mockRejectedValue(new Error('Test error'))
      const wrappedFn = withErrorHandling(errorFn, { logError: true, showToast: false })
      
      await expect(wrappedFn('arg1', 'arg2')).rejects.toThrow()
      
      expect(logSpy).toHaveBeenCalled()
      const callArgs = logSpy.mock.calls[0]
      expect(callArgs[1]).toHaveProperty('function')
      expect(callArgs[1]).toHaveProperty('args')
      
      logSpy.mockRestore()
    })
  })

  describe('createErrorHandler', () => {
    it('should create error handler for component', () => {
      const handler = createErrorHandler('TestComponent', { logError: false, showToast: false })
      const error = new Error('Component error')
      
      const message = handler(error)
      expect(message).toBeTruthy()
    })

    it('should include component name in context', () => {
      const logSpy = vi.spyOn(logger, 'error').mockImplementation(() => {})
      const handler = createErrorHandler('TestComponent', { logError: true, showToast: false })
      const error = new Error('Component error')
      
      handler(error, { componentStack: 'Stack trace' })
      
      expect(logSpy).toHaveBeenCalled()
      const callArgs = logSpy.mock.calls[0]
      expect(callArgs[1]).toHaveProperty('component', 'TestComponent')
      
      logSpy.mockRestore()
    })
  })
})

