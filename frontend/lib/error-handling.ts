/**
 * Утилиты для улучшенной обработки ошибок
 */

export interface ErrorDetails {
  message: string
  code?: string
  statusCode?: number
  timestamp: Date
  context?: Record<string, any>
}

export class AppError extends Error {
  code?: string
  statusCode?: number
  context?: Record<string, any>
  userMessage?: string

  constructor(
    message: string,
    options?: {
      code?: string
      statusCode?: number
      context?: Record<string, any>
      userMessage?: string
    }
  ) {
    super(message)
    this.name = 'AppError'
    this.code = options?.code
    this.statusCode = options?.statusCode
    this.context = options?.context
    this.userMessage = options?.userMessage
  }
}

/**
 * Обрабатывает ошибки API и возвращает понятное сообщение для пользователя
 */
export function handleApiError(error: unknown): string {
  if (error instanceof AppError) {
    return error.userMessage || error.message
  }

  if (error instanceof Error) {
    // Обработка сетевых ошибок
    if (error.message.includes('fetch') || error.message.includes('network')) {
      return 'Ошибка подключения к серверу. Проверьте интернет-соединение.'
    }

    // Обработка таймаутов
    if (error.message.includes('timeout') || error.message.includes('aborted')) {
      return 'Превышено время ожидания ответа от сервера. Попробуйте еще раз.'
    }

    return error.message
  }

  if (typeof error === 'string') {
    return error
  }

  return 'Произошла неизвестная ошибка. Попробуйте обновить страницу.'
}

/**
 * Логирует ошибку с контекстом
 */
export function logError(error: unknown, context?: Record<string, any>): ErrorDetails {
  const errorDetails: ErrorDetails = {
    message: error instanceof Error ? error.message : String(error),
    timestamp: new Date(),
    context,
  }

  if (error instanceof AppError) {
    errorDetails.code = error.code
    errorDetails.statusCode = error.statusCode
    errorDetails.context = { ...errorDetails.context, ...error.context }
  }

  // Логируем в консоль для разработки
  if (process.env.NODE_ENV === 'development') {
    console.error('Error details:', errorDetails)
  }

  // Здесь можно добавить отправку ошибок в систему мониторинга (Sentry, LogRocket и т.д.)
  // if (typeof window !== 'undefined' && window.Sentry) {
  //   window.Sentry.captureException(error, { extra: context })
  // }

  return errorDetails
}

/**
 * Валидирует ответ API
 */
export function validateApiResponse<T>(
  response: Response,
  expectedStatus: number = 200
): Promise<T> {
  if (!response.ok) {
    return response
      .json()
      .then((data) => {
        throw new AppError(data.error || `HTTP ${response.status}`, {
          statusCode: response.status,
          userMessage: data.error || `Ошибка сервера: ${response.status}`,
        })
      })
      .catch(() => {
        throw new AppError(`HTTP ${response.status}`, {
          statusCode: response.status,
          userMessage: `Ошибка сервера: ${response.status}`,
        })
      })
  }

  if (response.status !== expectedStatus) {
    throw new AppError(`Unexpected status code: ${response.status}`, {
      statusCode: response.status,
    })
  }

  return response.json()
}

/**
 * Создает безопасный обработчик для async функций
 */
export function createSafeHandler<T extends (...args: any[]) => Promise<any>>(
  handler: T,
  errorHandler?: (error: unknown) => void
): T {
  return (async (...args: Parameters<T>) => {
    try {
      return await handler(...args)
    } catch (error) {
      const userMessage = handleApiError(error)
      logError(error, { handler: handler.name, args })
      
      if (errorHandler) {
        errorHandler(error)
      } else {
        // По умолчанию показываем toast
        if (typeof window !== 'undefined') {
          // Импортируем динамически, чтобы избежать проблем с SSR
          import('sonner').then(({ toast }) => {
            toast.error(userMessage)
          })
        }
      }
      
      throw error
    }
  }) as T
}

