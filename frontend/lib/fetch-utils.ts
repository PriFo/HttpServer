/**
 * Утилиты для единообразной обработки fetch запросов с таймаутами и обработкой ошибок
 */

import { QUALITY_TIMEOUTS, QUALITY_RETRY_CONFIG, QUALITY_ERROR_MESSAGES } from './quality-constants'

export interface FetchOptions extends RequestInit {
  timeout?: number
  retryCount?: number
  retryDelay?: number
}

export interface FetchError {
  message: string
  status?: number
  isTimeout: boolean
  isNetworkError: boolean
  retryCount?: number
}

/**
 * Выполняет fetch запрос с таймаутом и единообразной обработкой ошибок
 * 
 * @param url - URL для запроса
 * @param options - Опции fetch, включая timeout (по умолчанию 10 секунд)
 * @returns Promise с данными ответа
 * @throws FetchError с понятным сообщением
 * 
 * @example
 * ```ts
 * try {
 *   const data = await fetchWithTimeout('/api/quality/stats?database=test', {
 *     timeout: 10000,
 *     cache: 'no-store'
 *   })
 *   const json = await data.json()
 * } catch (error) {
 *   if (error.isTimeout) {
 *     console.error('Таймаут запроса')
 *   } else if (error.isNetworkError) {
 *     console.error('Ошибка сети')
 *   }
 * }
 * ```
 */
export async function fetchWithTimeout(
  url: string,
  options: FetchOptions = {}
): Promise<Response> {
  const {
    timeout = QUALITY_TIMEOUTS.STANDARD,
    retryCount = QUALITY_RETRY_CONFIG.MAX_RETRIES,
    retryDelay = QUALITY_RETRY_CONFIG.RETRY_DELAY,
    signal: externalSignal,
    ...fetchOptions
  } = options

  let lastError: FetchError | null = null
  let attempt = 0

  while (attempt <= retryCount) {
    if (externalSignal?.aborted) {
      throw externalSignal.reason instanceof Error
        ? externalSignal.reason
        : new DOMException('Aborted', 'AbortError')
    }

    const controller = new AbortController()
    let didTimeout = false
    const timeoutId = setTimeout(() => {
      didTimeout = true
      controller.abort()
    }, timeout)

    const abortWithExternal = () => controller.abort()
    if (externalSignal) {
      externalSignal.addEventListener('abort', abortWithExternal, { once: true })
    }

    try {
      const response = await fetch(url, {
        ...fetchOptions,
        signal: controller.signal,
      })

      clearTimeout(timeoutId)
      if (externalSignal) {
        externalSignal.removeEventListener('abort', abortWithExternal)
      }

      return response
    } catch (error) {
      clearTimeout(timeoutId)
      if (externalSignal) {
        externalSignal.removeEventListener('abort', abortWithExternal)
      }

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          if (!didTimeout && externalSignal?.aborted) {
            throw error
          }
          lastError = {
            message: QUALITY_ERROR_MESSAGES.TIMEOUT,
            isTimeout: true,
            isNetworkError: false,
            retryCount: attempt,
          } as FetchError
        } else if (
          error.message.includes('fetch failed') ||
          error.message.includes('Failed to fetch') ||
          error.message.includes('NetworkError') ||
          error.message.includes('ECONNREFUSED') ||
          error.message.includes('ERR_CONNECTION_REFUSED')
        ) {
          lastError = {
            message: QUALITY_ERROR_MESSAGES.NETWORK_ERROR,
            isTimeout: false,
            isNetworkError: true,
            retryCount: attempt,
          } as FetchError

          if (attempt < retryCount) {
            attempt++
            await new Promise(resolve => setTimeout(resolve, retryDelay))
            continue
          }
        } else {
          lastError = {
            message: error.message || QUALITY_ERROR_MESSAGES.CONNECTION_ERROR,
            isTimeout: false,
            isNetworkError: false,
            retryCount: attempt,
          } as FetchError
        }
      } else {
        lastError = {
          message: QUALITY_ERROR_MESSAGES.UNKNOWN_ERROR,
          isTimeout: false,
          isNetworkError: false,
          retryCount: attempt,
        } as FetchError
      }

      if (!lastError.isNetworkError || attempt >= retryCount) {
        throw lastError
      }

      attempt++
      await new Promise(resolve => setTimeout(resolve, retryDelay))
    }
  }

  throw lastError || ({
    message: QUALITY_ERROR_MESSAGES.UNKNOWN_ERROR,
    isTimeout: false,
    isNetworkError: false,
    retryCount: attempt,
  } as FetchError)
}

/**
 * Выполняет fetch запрос и парсит JSON ответ с обработкой ошибок
 * 
 * @param url - URL для запроса
 * @param options - Опции fetch
 * @returns Promise с распарсенными данными
 * @throws FetchError с понятным сообщением
 * 
 * @example
 * ```ts
 * try {
 *   const data = await fetchJson('/api/quality/stats?database=test')
 *   console.log(data)
 * } catch (error) {
 *   console.error(error.message)
 * }
 * ```
 */
export async function fetchJson<T = any>(
  url: string,
  options: FetchOptions = {}
): Promise<T> {
  const response = await fetchWithTimeout(url, options)

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({ error: 'Failed to fetch data' }))
    let errorMessage = errorData.error || 'Не удалось загрузить данные'

    // Улучшаем сообщения об ошибках на основе статуса
    if (response.status === 503) {
      errorMessage = QUALITY_ERROR_MESSAGES.SERVER_UNAVAILABLE
    } else if (response.status === 504) {
      errorMessage = `${QUALITY_ERROR_MESSAGES.TIMEOUT}. Попробуйте позже.`
    } else if (response.status === 404) {
      errorMessage = QUALITY_ERROR_MESSAGES.NOT_FOUND
    } else if (response.status >= 500) {
      errorMessage = QUALITY_ERROR_MESSAGES.SERVER_ERROR
    }

    throw {
      message: errorMessage,
      status: response.status,
      isTimeout: false,
      isNetworkError: false,
    } as FetchError
  }

  return response.json()
}

/**
 * Получает понятное сообщение об ошибке для отображения пользователю
 * 
 * @param error - Ошибка (FetchError, Error или строка)
 * @param defaultMessage - Сообщение по умолчанию
 * @returns Понятное сообщение на русском языке
 */
export function getErrorMessage(
  error: unknown,
  defaultMessage: string = 'Произошла ошибка'
): string {
  if (typeof error === 'string') {
    return error
  }

  if (error && typeof error === 'object' && 'message' in error) {
    return (error as { message: string }).message
  }

  if (error instanceof Error) {
    return error.message || defaultMessage
  }

  return defaultMessage
}

/**
 * Проверяет, является ли ошибка таймаутом
 */
export function isTimeoutError(error: unknown): boolean {
  if (!error || typeof error !== 'object') {
    return false
  }
  return 'isTimeout' in error && (error as FetchError).isTimeout === true
}

/**
 * Проверяет, является ли ошибка сетевой ошибкой
 */
export function isNetworkError(error: unknown): boolean {
  if (!error || typeof error !== 'object') {
    return false
  }
  return 'isNetworkError' in error && (error as FetchError).isNetworkError === true
}

