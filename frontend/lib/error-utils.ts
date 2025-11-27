/**
 * Утилиты для упрощения миграции на новую систему обработки ошибок
 * и вспомогательные функции для работы с ошибками
 */

import { AppError, createUnknownError } from './errors'

/**
 * Обертка для безопасного выполнения асинхронных операций
 * Автоматически обрабатывает ошибки через ErrorContext
 * 
 * @example
 * ```ts
 * const result = await safeExecute(
 *   () => apiClient('/api/data'),
 *   'Не удалось загрузить данные'
 * )
 * ```
 */
export async function safeExecute<T>(
  fn: () => Promise<T>,
  fallbackMessage?: string
): Promise<T | null> {
  try {
    return await fn()
  } catch (error) {
    const appError = createUnknownError(error)
    console.error('Safe execute error:', {
      message: appError.message,
      technical: appError.technicalDetails,
      fallback: fallbackMessage,
    })
    return null
  }
}

/**
 * Преобразует любую ошибку в AppError
 * Полезно для миграции старого кода
 */
export function toAppError(error: unknown, userMessage?: string): AppError {
  if (error instanceof AppError) {
    return error
  }
  
  if (error instanceof Error) {
    return new AppError(
      userMessage || error.message || 'Произошла ошибка',
      error.message,
      undefined,
      'UNKNOWN_ERROR'
    )
  }
  
  return new AppError(
    userMessage || 'Произошла неизвестная ошибка',
    String(error),
    undefined,
    'UNKNOWN_ERROR'
  )
}

/**
 * Проверяет, является ли ошибка сетевой ошибкой
 */
export function isNetworkError(error: unknown): boolean {
  if (error instanceof AppError) {
    return error.code === 'NETWORK_ERROR' || error.statusCode === 0
  }
  
  if (error instanceof Error) {
    return (
      error.message.includes('fetch failed') ||
      error.message.includes('Failed to fetch') ||
      error.message.includes('NetworkError') ||
      error.message.includes('ECONNREFUSED') ||
      error.message.includes('ERR_CONNECTION_REFUSED') ||
      error.name === 'AbortError'
    )
  }
  
  return false
}

/**
 * Проверяет, является ли ошибка ошибкой таймаута
 */
export function isTimeoutError(error: unknown): boolean {
  if (error instanceof AppError) {
    return error.message.includes('время ожидания') || error.message.includes('timeout')
  }
  
  if (error instanceof Error) {
    return error.name === 'AbortError' || error.message.includes('timeout')
  }
  
  return false
}

/**
 * Получает пользовательское сообщение из ошибки
 */
export function getUserMessage(error: unknown, fallback = 'Произошла ошибка'): string {
  if (error instanceof AppError) {
    return error.message || fallback
  }
  
  if (error instanceof Error) {
    return error.message || fallback
  }
  
  if (typeof error === 'string') {
    return error
  }
  
  return fallback
}

/**
 * Получает технические детали из ошибки
 */
export function getTechnicalDetails(error: unknown): string | undefined {
  if (error instanceof AppError) {
    return error.technicalDetails
  }
  
  if (error instanceof Error) {
    return error.stack
  }
  
  return String(error)
}

