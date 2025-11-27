// Централизованные error messages для consistency

export const ERROR_MESSAGES = {
  // Network errors
  NETWORK_ERROR: 'Не удалось выполнить запрос. Проверьте подключение к сети.',
  NETWORK_TIMEOUT: 'Время ожидания истекло. Проверьте подключение к сети и попробуйте позже.',
  SERVER_ERROR: 'Ошибка сервера. Попробуйте позже.',

  // Data loading errors
  LOAD_GROUPS_ERROR: 'Не удалось загрузить группы. Попробуйте еще раз.',
  LOAD_DETAILS_ERROR: 'Не удалось загрузить детали группы. Попробуйте еще раз.',
  LOAD_STATS_ERROR: 'Не удалось загрузить статистику.',
  LOAD_KPVED_ERROR: 'Не удалось загрузить данные КПВЭД. Проверьте подключение к сети.',

  // Navigation errors
  NAVIGATION_ERROR: 'Не удалось перейти к детальной странице. Попробуйте еще раз.',
  URL_TOO_LONG: 'URL слишком длинный, возможны проблемы в некоторых браузерах.',

  // Export errors
  EXPORT_ERROR: 'Не удалось экспортировать данные. Проверьте подключение к сети и попробуйте позже.',

  // Search errors
  SEARCH_ERROR: 'Не удалось выполнить поиск. Попробуйте еще раз.',

  // Generic
  UNKNOWN_ERROR: 'Произошла неизвестная ошибка. Попробуйте еще раз.',
  TRY_AGAIN: 'Попробуйте еще раз позже.',
} as const

export type ErrorMessageKey = keyof typeof ERROR_MESSAGES

export function getErrorMessage(key: ErrorMessageKey, customMessage?: string): string {
  return customMessage || ERROR_MESSAGES[key]
}

export function handleApiError(error: unknown, fallbackKey: ErrorMessageKey = 'UNKNOWN_ERROR'): string {
  if (error instanceof Error) {
    // Check for specific error types
    if (error.message.includes('NetworkError') || error.message.includes('Failed to fetch')) {
      return ERROR_MESSAGES.NETWORK_ERROR
    }
    if (error.message.includes('timeout')) {
      return ERROR_MESSAGES.NETWORK_TIMEOUT
    }
    if (error.message.includes('500') || error.message.includes('502') || error.message.includes('503')) {
      return ERROR_MESSAGES.SERVER_ERROR
    }
  }

  return ERROR_MESSAGES[fallbackKey]
}

// ============================================================================
// Advanced Error Handling for API Routes
// ============================================================================

import { NextResponse } from 'next/server'

/**
 * Custom application error class for client-side
 * Основной класс для всех ошибок приложения
 */
export class AppError extends Error {
  constructor(
    message: string, // Сообщение для пользователя
    public readonly technicalDetails?: string, // Детали для логов
    public readonly statusCode?: number, // HTTP статус, если применимо
    public readonly code?: string
  ) {
    super(message)
    this.name = 'AppError'
    // Сохраняем stack trace
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, this.constructor)
    }
  }
}

/**
 * Фабрики для создания типичных ошибок
 */
export const createNetworkError = (message: string, statusCode?: number, technical?: string) =>
  new AppError(message, technical, statusCode, 'NETWORK_ERROR')

export const createValidationError = (message: string, technical?: string) =>
  new AppError(message, technical, 400, 'VALIDATION_ERROR')

export const createUnknownError = (error: unknown): AppError => {
  if (error instanceof AppError) return error
  if (error instanceof Error) {
    return new AppError(
      'Произошла непредвиденная ошибка',
      error.message,
      undefined,
      'UNKNOWN_ERROR'
    )
  }
  return new AppError('Произошла непредвиденная ошибка', String(error), undefined, 'UNKNOWN_ERROR')
}

/**
 * Утилита для структурированного логирования ошибок
 */
export function logError(error: unknown, context?: Record<string, unknown>): void {
  const timestamp = new Date().toISOString()
  const errorInfo: Record<string, unknown> = {
    timestamp,
    error: error instanceof Error ? {
      name: error.name,
      message: error.message,
      stack: error.stack,
    } : String(error),
  }

  if (error instanceof AppError) {
    errorInfo.userMessage = error.message
    errorInfo.technicalDetails = error.technicalDetails
    errorInfo.statusCode = error.statusCode
    errorInfo.code = error.code
  }

  if (context) {
    errorInfo.context = context
  }

  console.error('[Error Handler]', errorInfo)
  
  // Здесь можно добавить интеграцию с Sentry в будущем
  // if (typeof window !== 'undefined' && window.Sentry) {
  //   window.Sentry.captureException(error, { contexts: { custom: context } })
  // }
}

export class ValidationError extends AppError {
  constructor(message: string, details?: unknown) {
    super(message, typeof details === 'string' ? details : JSON.stringify(details), 400, 'VALIDATION_ERROR')
    this.name = 'ValidationError'
  }
}

export class UnauthorizedError extends AppError {
  constructor(message: string = 'Необходима авторизация') {
    super(message, undefined, 401, 'UNAUTHORIZED')
    this.name = 'UnauthorizedError'
  }
}

export class BackendError extends AppError {
  constructor(message: string, statusCode: number = 502, details?: unknown) {
    super(
      message,
      typeof details === 'string' ? details : JSON.stringify(details),
      statusCode,
      'BACKEND_ERROR'
    )
    this.name = 'BackendError'
  }
}

export class BackendConnectionError extends AppError {
  constructor(message: string = 'Не удалось подключиться к бэкенду', details?: unknown) {
    super(
      message,
      typeof details === 'string' ? details : JSON.stringify(details),
      503,
      'BACKEND_CONNECTION_ERROR'
    )
    this.name = 'BackendConnectionError'
  }
}

export class BackendResponseError extends AppError {
  constructor(message: string, statusCode: number = 502, details?: unknown) {
    super(
      message,
      typeof details === 'string' ? details : JSON.stringify(details),
      statusCode,
      'BACKEND_RESPONSE_ERROR'
    )
    this.name = 'BackendResponseError'
  }
}

export class NotFoundError extends AppError {
  constructor(resource: string, details?: unknown) {
    super(
      `${resource} не найден`,
      typeof details === 'string' ? details : JSON.stringify(details),
      404,
      'NOT_FOUND'
    )
    this.name = 'NotFoundError'
  }
}

/**
 * Error response structure
 */
interface ErrorResponse {
  error: string
  code?: string
  details?: unknown
  timestamp?: string
  path?: string
}

/**
 * Creates a standardized error response for API routes
 */
export function createErrorResponse(
  error: Error | AppError | unknown,
  options?: {
    includeStack?: boolean
    path?: string
  }
): NextResponse<ErrorResponse> {
  const isDevelopment = process.env.NODE_ENV === 'development'
  const includeStack = options?.includeStack ?? isDevelopment

  // Handle AppError instances
  if (error instanceof AppError) {
    const response: ErrorResponse = {
      error: error.message,
      code: error.code,
      timestamp: new Date().toISOString(),
    }

    if (error.technicalDetails) {
      response.details = error.technicalDetails
    }

    if (options?.path) {
      response.path = options.path
    }

    if (includeStack && error.stack) {
      response.details = {
        ...(typeof response.details === 'object' && response.details !== null ? response.details : {}),
        stack: error.stack,
      }
    }

    return NextResponse.json(response, { status: error.statusCode || 500 })
  }

  // Handle standard Error instances
  if (error instanceof Error) {
    const response: ErrorResponse = {
      error: isDevelopment ? error.message : 'Internal server error',
      code: 'INTERNAL_ERROR',
      timestamp: new Date().toISOString(),
    }

    if (options?.path) {
      response.path = options.path
    }

    if (includeStack && error.stack) {
      response.details = { stack: error.stack }
    }

    return NextResponse.json(response, { status: 500 })
  }

  // Handle unknown errors
  const response: ErrorResponse = {
    error: 'An unexpected error occurred',
    code: 'UNKNOWN_ERROR',
    timestamp: new Date().toISOString(),
  }

  if (options?.path) {
    response.path = options.path
  }

  if (isDevelopment && error) {
    response.details = error
  }

  return NextResponse.json(response, { status: 500 })
}

/**
 * Wraps an async route handler with error handling
 */
export function withErrorHandler<T extends (...args: any[]) => Promise<NextResponse>>(
  handler: T
): T {
  return (async (...args: Parameters<T>) => {
    try {
      return await handler(...args)
    } catch (error) {
      console.error('API Route Error:', error)

      const request = args[0]
      const path = request?.url ? new URL(request.url).pathname : undefined

      return createErrorResponse(error, { path })
    }
  }) as T
}
