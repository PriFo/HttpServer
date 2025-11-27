/**
 * Централизованный API-клиент с обработкой ошибок
 * Использует AppError для структурированных ошибок
 */

import { getApiUrl } from './api-config'
import { AppError, createNetworkError, logError } from './errors'

export type ApiClientOptions = RequestInit & {
  /**
   * Если true, ошибки не будут автоматически обрабатываться через ErrorContext
   * Полезно, когда нужно обработать ошибку вручную
   */
  skipErrorHandler?: boolean
  /**
   * Кастомный обработчик ошибок
   */
  onError?: (error: AppError) => void
  /**
   * Количество попыток повтора при ошибке (по умолчанию 0)
   * Работает для сетевых ошибок, 5xx ошибок и 429 (Too Many Requests)
   * Для 429 использует заголовок Retry-After, если он доступен
   */
  retries?: number
  /**
   * Задержка между попытками в миллисекундах (по умолчанию 1000)
   */
  retryDelay?: number
  /**
   * Кастомный таймаут запроса в миллисекундах.
   * Значение <= 0 отключает автоматический таймаут (fetch будет ждать бесконечно).
   * По умолчанию 7000 (7 секунд), чтобы соответствовать предыдущему поведению.
   */
  timeoutMs?: number
}

/**
 * Парсит ответ об ошибке от бэкенда
 * Бэкенд возвращает ошибки в формате {"error": "message"}
 */
async function parseErrorResponse(response: Response): Promise<{ message: string; technical?: string }> {
  let errorMessage = `Ошибка ${response.status}: ${response.statusText || 'Неизвестная ошибка'}`
  let technicalDetails = ''

  try {
    const contentType = response.headers.get('content-type')
    
    if (contentType && contentType.includes('application/json')) {
      const errorData = await response.json()
      errorMessage = errorData.error || errorData.message || errorMessage
      technicalDetails = JSON.stringify(errorData)
    } else {
      const errorText = await response.text()
      if (errorText) {
        try {
          const errorJson = JSON.parse(errorText)
          errorMessage = errorJson.error || errorJson.message || errorText
          technicalDetails = errorText
        } catch {
          errorMessage = errorText || errorMessage
          technicalDetails = errorText
        }
      }
    }
  } catch (err) {
    logError(err, { context: 'parseErrorResponse', status: response.status })
  }

  return { message: errorMessage, technical: technicalDetails }
}

/**
 * Централизованный API-клиент
 * 
 * @param url - URL для запроса (может быть относительным или абсолютным)
 * @param options - Опции запроса (RequestInit + дополнительные опции)
 * @returns Promise<Response> - Ответ от сервера
 * @throws AppError - Структурированная ошибка приложения
 * 
 * @example
 * ```ts
 * try {
 *   const response = await apiClient('/api/users')
 *   const data = await response.json()
 * } catch (error) {
 *   // error - это AppError
 *   handleError(error)
 * }
 * ```
 */
export async function apiClient(url: string, options: ApiClientOptions = {}): Promise<Response> {
  const {
    skipErrorHandler,
    onError,
    retries = 0,
    retryDelay = 1000,
    timeoutMs = 7000,
    signal: externalSignal,
    ...fetchOptions
  } = options
  
  let lastError: AppError | null = null
  
  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      // Преобразуем относительный путь в полный URL
      // Пути, начинающиеся с /api/, являются Next.js API routes
      const fullUrl = url.startsWith('http://') || url.startsWith('https://') 
        ? url 
        : url.startsWith('/api/')
        ? url  // Next.js API route - используем как относительный путь
        : getApiUrl(url)  // Прямой запрос к backend
      
      // Настраиваем контроллер для таймаута (можно отключить, установив timeoutMs <= 0)
      let controller: AbortController | null = null
      let timeoutId: ReturnType<typeof setTimeout> | null = null
      let finalSignal: AbortSignal | undefined = externalSignal || undefined

      if (timeoutMs > 0) {
        controller = new AbortController()
        timeoutId = setTimeout(() => controller?.abort(), timeoutMs)

        if (externalSignal) {
          if (externalSignal.aborted) {
            controller.abort()
          } else {
            externalSignal.addEventListener('abort', () => controller?.abort(), { once: true })
          }
        }

        finalSignal = controller.signal
      }

      try {
        const response = await fetch(fullUrl, {
          ...fetchOptions,
          signal: finalSignal,
          headers: {
            'Content-Type': 'application/json',
            ...fetchOptions.headers,
          },
        })

        if (timeoutId) {
          clearTimeout(timeoutId)
        }

        if (!response.ok) {
          const { message, technical } = await parseErrorResponse(response)
          const error = createNetworkError(message, response.status, technical)
          
          // Обработка 429 (Too Many Requests) с использованием Retry-After заголовка
          if (response.status === 429) {
            const retryAfter = response.headers.get('Retry-After')
            const delay = retryAfter 
              ? parseInt(retryAfter, 10) * 1000 
              : retryDelay * (attempt + 1) * 2 // Увеличиваем задержку для 429
            
            if (attempt < retries) {
              lastError = error
              await new Promise(resolve => setTimeout(resolve, delay))
              continue
            }
          }
          
          // Повторяем только для сетевых ошибок и 5xx ошибок
          const shouldRetry = attempt < retries && (
            response.status >= 500 || 
            response.status === 0 || // Сетевые ошибки
            response.status === 408 // Timeout
          )
          
          if (shouldRetry) {
            lastError = error
            await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)))
            continue
          }
          
          // Вызываем кастомный обработчик, если есть
          if (onError) {
            onError(error)
          }
          
          throw error
        }

        return response
      } catch (fetchError) {
        if (timeoutId) {
          clearTimeout(timeoutId)
        }
        
        // Обработка ошибок сети
        if (fetchError instanceof AppError) {
          // Повторяем только для сетевых ошибок
          const shouldRetry = attempt < retries && (
            fetchError.statusCode === 0 || // Сетевые ошибки
            fetchError.statusCode === undefined
          )
          
          if (shouldRetry) {
            lastError = fetchError
            await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)))
            continue
          }
          
          throw fetchError
        }
        
        if (fetchError instanceof Error) {
          if (fetchError.name === 'AbortError') {
            const timeoutMessage = timeoutMs > 0
              ? `Превышено время ожидания ответа от сервера (${timeoutMs / 1000} секунд). Сервер может быть перегружен или недоступен.`
              : 'Запрос был прерван'

            const error = createNetworkError(
              timeoutMessage,
              0,
              timeoutMs > 0
                ? `Request timeout after ${timeoutMs}ms`
                : 'Request aborted'
            )
            
            // Повторяем для таймаутов
            if (attempt < retries) {
              lastError = error
              await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)))
              continue
            }
            
            if (onError) {
              onError(error)
            }
            throw error
          }
          
          if (fetchError.message.includes('fetch failed') || 
              fetchError.message.includes('Failed to fetch') ||
              fetchError.message.includes('NetworkError') ||
              fetchError.message.includes('ECONNREFUSED') ||
              fetchError.message.includes('ERR_CONNECTION_REFUSED')) {
            const error = createNetworkError(
              'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999. Используйте скрипт start-backend-exe.bat для запуска.',
              0,
              fetchError.message
            )
            
            // Повторяем для сетевых ошибок
            if (attempt < retries) {
              lastError = error
              await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)))
              continue
            }
            
            if (onError) {
              onError(error)
            }
            throw error
          }
          
          const error = createNetworkError(
            fetchError.message,
            0,
            fetchError.stack
          )
          if (onError) {
            onError(error)
          }
          throw error
        }
        
        const error = createNetworkError(
          'Неизвестная ошибка при выполнении запроса',
          0,
          String(fetchError)
        )
        if (onError) {
          onError(error)
        }
        throw error
      }
    } catch (outerError) {
      // Если произошла ошибка при подготовке запроса, пробрасываем её дальше
      throw outerError
    }
  }
  
  if (lastError) {
    if (onError) {
      onError(lastError)
    }
    throw lastError
  }
  
  // Это не должно произойти, но на всякий случай
  throw createNetworkError(
    'Неизвестная ошибка при выполнении запроса',
    0,
    'All retry attempts exhausted'
  )
}

/**
 * Удобная функция для выполнения GET запроса и парсинга JSON
 */
export async function apiGet<T>(url: string, options?: ApiClientOptions): Promise<T> {
  const response = await apiClient(url, { ...options, method: 'GET' })
  return response.json() as Promise<T>
}

/**
 * Удобная функция для выполнения POST запроса и парсинга JSON
 */
export async function apiPost<T>(url: string, data?: unknown, options?: ApiClientOptions): Promise<T> {
  const response = await apiClient(url, {
    ...options,
    method: 'POST',
    body: data ? JSON.stringify(data) : undefined,
  })
  return response.json() as Promise<T>
}

/**
 * Удобная функция для выполнения PUT запроса и парсинга JSON
 */
export async function apiPut<T>(url: string, data?: unknown, options?: ApiClientOptions): Promise<T> {
  const response = await apiClient(url, {
    ...options,
    method: 'PUT',
    body: data ? JSON.stringify(data) : undefined,
  })
  return response.json() as Promise<T>
}

/**
 * Удобная функция для выполнения DELETE запроса и парсинга JSON
 */
export async function apiDelete<T>(url: string, options?: ApiClientOptions): Promise<T> {
  const response = await apiClient(url, { ...options, method: 'DELETE' })
  return response.json() as Promise<T>
}

/**
 * Устаревшая функция для обратной совместимости
 * @deprecated Используйте useApiClient() или apiGet/apiPost напрямую
 */
export async function apiClientJson<T>(
  url: string,
  options?: RequestInit & { skipErrorHandler?: boolean; onError?: (error: AppError) => void }
): Promise<T> {
  const { skipErrorHandler, onError, ...fetchOptions } = options || {}
  const response = await apiClient(url, {
    ...fetchOptions,
    skipErrorHandler,
    onError,
  })
  return response.json() as Promise<T>
}
