/**
 * Централизованная система обработки ошибок
 * Интегрирует логирование и обработку ошибок
 */

import { NextResponse } from 'next/server'
import { logger } from './logger'
import { 
  AppError, 
  BackendConnectionError, 
  BackendResponseError, 
  NotFoundError 
} from './errors'

// Re-export для обратной совместимости
export { AppError, BackendConnectionError, BackendResponseError, NotFoundError }

export interface LogContext {
  [key: string]: unknown
}

export interface ErrorHandlerOptions {
  logError?: boolean
  showToast?: boolean
  context?: LogContext
  fallbackMessage?: string
}

/**
 * Обрабатывает ошибку с логированием и уведомлением пользователя
 */
export function handleError(
  error: unknown,
  options: ErrorHandlerOptions = {}
): string {
  const {
    logError = true,
    showToast = true,
    context = {},
    fallbackMessage = 'Произошла ошибка',
  } = options

  let userMessage = fallbackMessage
  let errorToLog: Error | null = null

  // Преобразуем ошибку в Error объект
  if (error instanceof AppError) {
    userMessage = error.message || fallbackMessage
    errorToLog = error

    if (logError) {
      logger.error('Application error', {
        ...context,
        code: error.code,
        statusCode: error.statusCode,
        technicalDetails: error.technicalDetails,
      }, error)
    }
  } else if (error instanceof Error) {
    userMessage = getUserFriendlyMessage(error)
    errorToLog = error

    if (logError) {
      logger.error('Error occurred', context, error)
    }
  } else if (typeof error === 'string') {
    userMessage = error
    if (logError) {
      logger.error('String error', { ...context, error })
    }
  } else {
    if (logError) {
      logger.error('Unknown error', { ...context, error: String(error) })
    }
  }

  // Показываем toast уведомление
  if (showToast && typeof window !== 'undefined') {
    import('sonner').then(({ toast }) => {
      toast.error(userMessage)
    }).catch(() => {
      // Если sonner недоступен, просто логируем
      console.error(userMessage)
    })
  }

  return userMessage
}

/**
 * Анализирует ошибку и возвращает детали для обработки
 */
export interface ErrorDetails {
  message: string
  code?: string
  statusCode?: number
  isRetryable?: boolean
  retryAfter?: number
  context?: Record<string, any>
}

export function analyzeError(error: unknown, context?: Record<string, any>): ErrorDetails {
  // Если это уже наш AppError, просто возвращаем его детали
  if (error instanceof AppError) {
    return {
      message: error.message,
      code: error.code,
      statusCode: error.statusCode,
      isRetryable: error.statusCode ? error.statusCode >= 500 : true,
      context: { ...context },
    }
  }

  // Обработка стандартных ошибок Error
  if (error instanceof Error) {
    // Ошибки сети и таймауты
    if (error.name === 'TimeoutError' || error.message.includes('timeout')) {
      return {
        message: 'Превышено время ожидания ответа от сервера. Попробуйте обновить позже.',
        code: 'TIMEOUT',
        isRetryable: true,
        retryAfter: 5, // секунд
        context,
      }
    }

    if (error.name === 'AbortError' || error.message.includes('aborted')) {
      return {
        message: 'Запрос был прерван. Попробуйте еще раз.',
        code: 'ABORTED',
        isRetryable: true,
        context,
      }
    }

    if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
      return {
        message: 'Ошибка сети. Проверьте подключение к интернету и повторите попытку.',
        code: 'NETWORK_ERROR',
        isRetryable: true,
        retryAfter: 3,
        context,
      }
    }

    // JSON parsing ошибки
    if (error instanceof SyntaxError && error.message.includes('JSON')) {
      return {
        message: 'Ошибка обработки данных с сервера. Попробуйте обновить страницу.',
        code: 'PARSE_ERROR',
        statusCode: 500,
        isRetryable: true,
        context,
      }
    }

    // Общая ошибка
    return {
      message: error.message || 'Произошла неизвестная ошибка',
      code: 'UNKNOWN_ERROR',
      context: { originalError: error.message, ...context },
    }
  }

  // Обработка ошибок из fetch (Response с ошибкой)
  if (error && typeof error === 'object' && 'status' in error) {
    const fetchError = error as { status: number; statusText?: string; error?: string; message?: string }
    const statusCode = fetchError.status

    let message = 'Произошла ошибка при запросе к серверу'
    let code = 'HTTP_ERROR'
    let isRetryable = false

    switch (statusCode) {
      case 400:
        message = fetchError.error || fetchError.message || 'Некорректный запрос. Проверьте введенные данные.'
        code = 'BAD_REQUEST'
        break
      case 401:
        message = 'Требуется авторизация. Пожалуйста, войдите в систему.'
        code = 'UNAUTHORIZED'
        break
      case 403:
        message = 'Доступ запрещен. У вас нет прав для выполнения этого действия.'
        code = 'FORBIDDEN'
        break
      case 404:
        message = 'Запрашиваемый ресурс не найден.'
        code = 'NOT_FOUND'
        isRetryable = false
        break
      case 429:
        message = 'Слишком много запросов. Пожалуйста, подождите немного и попробуйте снова.'
        code = 'RATE_LIMIT'
        isRetryable = true
        break
      case 500:
        message = 'Внутренняя ошибка сервера. Пожалуйста, попробуйте позже или обратитесь в поддержку.'
        code = 'SERVER_ERROR'
        isRetryable = true
        break
      case 502:
      case 503:
        message = 'Сервер временно недоступен. Попробуйте обновить позже.'
        code = 'SERVICE_UNAVAILABLE'
        isRetryable = true
        break
      case 504:
        message = 'Превышено время ожидания ответа от сервера. Попробуйте позже.'
        code = 'GATEWAY_TIMEOUT'
        isRetryable = true
        break
      default:
        message = fetchError.error || fetchError.message || `Ошибка ${statusCode}: ${fetchError.statusText || 'Неизвестная ошибка'}`
    }

    return {
      message,
      code,
      statusCode,
      isRetryable,
      context: { statusText: fetchError.statusText, ...context },
    }
  }

  // Обработка строковых ошибок
  if (typeof error === 'string') {
    return {
      message: error,
      code: 'STRING_ERROR',
      context,
    }
  }

  // Неизвестная ошибка
  logger.error('Unknown error type', context, error instanceof Error ? error : undefined)
  return {
    message: 'Произошла неизвестная ошибка. Пожалуйста, попробуйте обновить страницу.',
    code: 'UNKNOWN',
    isRetryable: true,
    context: { errorType: typeof error, error, ...context },
  }
}

/**
 * Обрабатывает ошибку с логированием (новая сигнатура для совместимости)
 */
export function handleErrorWithDetails(
  error: unknown,
  component: string,
  operation: string,
  context?: Record<string, any>
): ErrorDetails {
  const fullContext = { component, operation, ...context }
  const errorDetails = analyzeError(error, fullContext)
  
  logger.error(
    `Error in ${component}.${operation}: ${errorDetails.message}`,
    { ...errorDetails.context, ...fullContext, component },
    error instanceof Error ? error : undefined
  )

  return errorDetails
}

/**
 * Получает понятное сообщение для пользователя из ошибки
 */
function getUserFriendlyMessage(error: Error): string {
  const message = error.message.toLowerCase()

  // Сетевые ошибки
  if (
    message.includes('fetch failed') ||
    message.includes('failed to fetch') ||
    message.includes('networkerror') ||
    message.includes('econnrefused') ||
    message.includes('connection refused')
  ) {
    return 'Не удалось подключиться к серверу. Проверьте подключение к интернету.'
  }

  // Таймауты
  if (
    message.includes('timeout') ||
    message.includes('aborted') ||
    message.includes('превышено время')
  ) {
    return 'Превышено время ожидания ответа. Попробуйте еще раз.'
  }

  // HTTP ошибки
  if (message.includes('404')) {
    return 'Запрашиваемый ресурс не найден.'
  }

  if (message.includes('403')) {
    return 'Доступ запрещен.'
  }

  if (message.includes('401')) {
    return 'Требуется авторизация.'
  }

  if (message.includes('500') || message.includes('502') || message.includes('503')) {
    return 'Ошибка сервера. Попробуйте позже.'
  }

  // Возвращаем оригинальное сообщение, если оно понятное
  if (error.message && error.message.length < 200) {
    return error.message
  }

  return 'Произошла ошибка. Попробуйте обновить страницу.'
}

/**
 * Обертка для async функций с автоматической обработкой ошибок
 */
export function withErrorHandling<T extends (...args: any[]) => Promise<any>>(
  fn: T,
  options: ErrorHandlerOptions = {}
): T {
  return (async (...args: Parameters<T>) => {
    try {
      return await fn(...args)
    } catch (error) {
      const userMessage = handleError(error, {
        ...options,
        context: {
          ...options.context,
          function: fn.name,
          args: args.length,
        },
      })
      
      // Пробрасываем ошибку дальше, если нужно
      throw error
    }
  }) as T
}

/**
 * Создает безопасный обработчик для React компонентов
 */
export function createErrorHandler(
  componentName: string,
  options: ErrorHandlerOptions = {}
) {
  return (error: unknown, errorInfo?: React.ErrorInfo) => {
    handleError(error, {
      ...options,
      context: {
        ...options.context,
        component: componentName,
        errorInfo: errorInfo ? {
          componentStack: errorInfo.componentStack,
        } : undefined,
      },
    })
  }
}

// ============================================================================
// API Route Error Handling Functions
// ============================================================================

/**
 * Класс ошибки валидации
 */
export class ValidationError extends Error {
  constructor(
    message: string,
    public readonly paramName?: string,
    public readonly context?: LogContext
  ) {
    super(message)
    this.name = 'ValidationError'
  }
}

/**
 * Валидирует обязательный параметр
 */
export function validateRequired(
  value: unknown,
  paramName: string,
  context?: LogContext
): asserts value is string {
  if (value === null || value === undefined || value === '') {
    const error = new ValidationError(
      `Required parameter '${paramName}' is missing or empty`,
      paramName,
      context
    )
    logger.error(`Validation error: ${error.message}`, context || {})
    throw error
  }
}

/**
 * Обрабатывает ответ от бэкенда для API routes с логированием
 */
export async function handleBackendResponse(
  response: Response,
  endpoint: string,
  context: LogContext,
  options: {
    allow404?: boolean
    defaultData?: any
    errorMessage?: string
  } = {}
): Promise<NextResponse> {
  const { allow404 = false, defaultData = null, errorMessage = 'Backend request failed' } = options

  if (!response.ok) {
    // Для 404 возвращаем дефолтные данные, если разрешено
    if (response.status === 404 && allow404) {
      return NextResponse.json(defaultData)
    }

    const errorText = await response.text().catch(() => 'Unknown error')
    let errorData: any = { error: errorMessage, status: response.status }

    try {
      errorData = JSON.parse(errorText)
    } catch {
      errorData.message = errorText
    }

    logger.error(`Backend request failed: ${endpoint}`, {
      ...context,
      status: response.status,
      error: errorData,
    })

    return NextResponse.json(errorData, { status: response.status })
  }

  const data = await response.json().catch(() => null)
  return NextResponse.json(data)
}

/**
 * Обрабатывает ошибки fetch запросов для API routes
 */
export function handleFetchError(
  error: unknown,
  endpoint: string,
  context: LogContext
): NextResponse {
  let status = 500
  let errorMessage = 'Internal server error'
  let errorDetails: any = {}

  if (error instanceof ValidationError) {
    status = 400
    errorMessage = error.message
    errorDetails = {
      param: error.paramName,
      type: 'validation_error',
    }
  } else if (error instanceof Error) {
    if (error.name === 'AbortError' || error.message.includes('timeout')) {
      status = 504
      errorMessage = 'Request timeout'
    } else if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
      status = 503
      errorMessage = 'Service unavailable'
    } else {
      errorMessage = error.message
    }
    errorDetails = {
      type: error.name,
      message: error.message,
    }
  } else if (typeof error === 'string') {
    errorMessage = error
  }

  logger.error(`Fetch error: ${endpoint}`, {
    ...context,
    status,
    error: errorDetails,
  }, error instanceof Error ? error : undefined)

  return NextResponse.json(
    {
      error: errorMessage,
      details: process.env.NODE_ENV === 'development' ? errorDetails : undefined,
    },
    { status }
  )
}
