/**
 * Серверные утилиты для единообразной обработки fetch запросов в Next.js API routes
 * 
 * Эти утилиты предназначены для использования в серверных компонентах и API routes,
 * где нет доступа к браузерным API, но есть доступ к fetch API Node.js.
 */

import { QUALITY_TIMEOUTS, QUALITY_ERROR_MESSAGES } from './quality-constants'

export interface ServerFetchOptions extends RequestInit {
  timeout?: number
}

export interface ServerFetchError extends Error {
  status?: number
  isTimeout: boolean
  isNetworkError: boolean
}

/**
 * Выполняет fetch запрос с таймаутом на сервере (для Next.js API routes)
 * 
 * @param url - URL для запроса
 * @param options - Опции fetch, включая timeout (по умолчанию 10 секунд)
 * @returns Promise с данными ответа
 * @throws ServerFetchError с понятным сообщением
 */
export async function fetchWithTimeoutServer(
  url: string,
  options: ServerFetchOptions = {}
): Promise<Response> {
  const { timeout = QUALITY_TIMEOUTS.STANDARD, ...fetchOptions } = options

  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeout)

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      signal: controller.signal,
    })

    clearTimeout(timeoutId)

    if (!response.ok) {
      let errorData: { error?: string }
      try {
        errorData = await response.json()
      } catch {
        errorData = { error: await response.text() }
      }

      const error: ServerFetchError = new Error(
        errorData.error || `Backend responded with status ${response.status}`
      ) as ServerFetchError
      error.status = response.status
      error.isTimeout = false
      error.isNetworkError = false
      throw error
    }

    return response
  } catch (error: unknown) {
    clearTimeout(timeoutId)

    const err = error as { name?: string; message?: string; status?: number };
    const fetchError: ServerFetchError = error as ServerFetchError
    fetchError.isTimeout = err.name === 'AbortError'
    fetchError.isNetworkError =
      err.message?.includes('fetch failed') === true ||
      err.message?.includes('Failed to fetch') === true ||
      err.message?.includes('NetworkError') === true ||
      err.message?.includes('ECONNREFUSED') === true ||
      err.message?.includes('ERR_CONNECTION_REFUSED') === true

    if (fetchError.isTimeout) {
      fetchError.message = QUALITY_ERROR_MESSAGES.TIMEOUT
      fetchError.status = 504
    } else if (fetchError.isNetworkError) {
      fetchError.message = QUALITY_ERROR_MESSAGES.SERVER_UNAVAILABLE
      fetchError.status = 503
    } else {
      fetchError.message = fetchError.message || err.message || QUALITY_ERROR_MESSAGES.UNKNOWN_ERROR
      fetchError.status = fetchError.status || err.status || 500
    }
    throw fetchError
  }
}

/**
 * Выполняет fetch запрос и парсит JSON ответ на сервере.
 * 
 * @param url - URL для запроса
 * @param options - Опции fetch, включая timeout (по умолчанию 10 секунд)
 * @returns Promise с JSON данными
 * @throws ServerFetchError
 */
export async function fetchJsonServer<T = unknown>(
  url: string,
  options: ServerFetchOptions = {}
): Promise<T> {
  const response = await fetchWithTimeoutServer(url, options)
  return response.json()
}

/**
 * Получает пользовательское сообщение об ошибке из объекта ServerFetchError.
 * @param error - Объект ошибки
 * @param fallback - Сообщение по умолчанию, если сообщение об ошибке отсутствует
 * @returns Пользовательское сообщение об ошибке
 */
export function getServerErrorMessage(error: unknown, fallback = 'Произошла ошибка'): string {
  if (error && typeof error === 'object' && 'message' in error) {
    return (error as ServerFetchError).message || fallback
  }
  return fallback
}

/**
 * Получает HTTP статус код из объекта ошибки.
 * @param error - Объект ошибки
 * @param fallback - Статус код по умолчанию, если статус отсутствует
 * @returns HTTP статус код
 */
export function getServerErrorStatus(error: unknown, fallback = 500): number {
  if (error && typeof error === 'object' && 'status' in error) {
    const status = (error as { status?: number }).status
    return status || fallback
  }
  return fallback
}

/**
 * Проверяет, является ли ошибка таймаутом
 */
export function isTimeoutError(error: unknown): boolean {
  if (error && typeof error === 'object' && 'isTimeout' in error) {
    return (error as ServerFetchError).isTimeout === true
  }
  if (error instanceof Error) {
    return error.name === 'AbortError'
  }
  return false
}

/**
 * Проверяет, является ли ошибка сетевой ошибкой
 */
export function isNetworkError(error: unknown): boolean {
  if (error && typeof error === 'object' && 'isNetworkError' in error) {
    return (error as ServerFetchError).isNetworkError === true
  }
  if (error instanceof Error) {
    return error.message.includes('fetch failed') ||
           error.message.includes('Failed to fetch') ||
           error.message.includes('NetworkError') ||
           error.message.includes('ECONNREFUSED') ||
           error.message.includes('ERR_CONNECTION_REFUSED')
  }
  return false
}


