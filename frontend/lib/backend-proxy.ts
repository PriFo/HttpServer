/**
 * Единая утилита для проксирования запросов к Go backend
 * Стандартизирует обработку ошибок, таймауты и логирование
 */

import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from './api-config'
import { extractTokenFromHeader, getUserFromRequest } from './jwt'

export interface ProxyOptions {
  timeout?: number // Таймаут в миллисекундах (по умолчанию 7000)
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  body?: any
  headers?: Record<string, string>
  onError?: (error: Error, response?: Response) => NextResponse
  returnEmptyOnError?: boolean // Возвращать пустой массив/объект при ошибке вместо ошибки
  requireAuth?: boolean // Требовать JWT токен (по умолчанию false)
  forwardAuth?: boolean // Передавать Authorization header на backend (по умолчанию true)
}

/**
 * Проксирует запрос к Go backend с единой обработкой ошибок
 */
export async function proxyToBackend(
  endpoint: string,
  request: NextRequest,
  options: ProxyOptions = {}
): Promise<NextResponse> {
  const {
    timeout = 7000,
    method = 'GET',
    body,
    headers = {},
    onError,
    returnEmptyOnError = false,
    requireAuth = false,
    forwardAuth = true,
  } = options

  // Проверка аутентификации, если требуется
  if (requireAuth) {
    const user = getUserFromRequest(request)
    if (!user) {
      return NextResponse.json(
        { error: 'Unauthorized. JWT token required.' },
        { status: 401 }
      )
    }
  }

  const backendUrl = getBackendUrl()
  const fullUrl = `${backendUrl}${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`

  // Добавляем query параметры из запроса
  const url = new URL(fullUrl)
  const searchParams = new URL(request.url).searchParams
  searchParams.forEach((value, key) => {
    url.searchParams.set(key, value)
  })

  // Подготавливаем заголовки для backend
  const backendHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...headers,
  }

  // Передаем Authorization header на backend, если требуется
  if (forwardAuth) {
    const authHeader = request.headers.get('authorization')
    if (authHeader) {
      backendHeaders['Authorization'] = authHeader
    }

    // Также передаем информацию о пользователе в заголовках (для Go backend)
    const user = getUserFromRequest(request)
    if (user) {
      backendHeaders['X-User-Id'] = user.userId
      if (user.email) {
        backendHeaders['X-User-Email'] = user.email
      }
      if (user.roles && user.roles.length > 0) {
        backendHeaders['X-User-Roles'] = user.roles.join(',')
      }
    }
  }

  try {
    // Создаем контроллер для таймаута
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeout)

    try {
      const fetchOptions: RequestInit = {
        method,
        headers: backendHeaders,
        cache: 'no-store',
        signal: controller.signal,
      }

      if (body && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
        fetchOptions.body = typeof body === 'string' ? body : JSON.stringify(body)
      }

      const response = await fetch(url.toString(), fetchOptions)
      clearTimeout(timeoutId)

      // Обработка успешного ответа
      if (response.ok) {
        const data = await response.json()
        return NextResponse.json(data)
      }

      // Обработка ошибок от backend
      if (onError) {
        return onError(new Error(`HTTP ${response.status}`), response)
      }

      // Стандартная обработка ошибок
      let errorMessage = `HTTP ${response.status}: ${response.statusText || 'Unknown error'}`

      try {
        const contentType = response.headers.get('content-type')
        if (contentType?.includes('application/json')) {
          const errorData = await response.json()
          errorMessage = errorData.error || errorData.message || errorMessage
        } else {
          const errorText = await response.text()
          if (errorText) {
            try {
              const errorJson = JSON.parse(errorText)
              errorMessage = errorJson.error || errorJson.message || errorText
            } catch {
              errorMessage = errorText || errorMessage
            }
          }
        }
      } catch (parseError) {
        console.error('Error parsing error response:', parseError)
      }

      // Возвращаем пустой результат при ошибке, если указано
      if (returnEmptyOnError) {
        console.warn(`Backend error (${response.status}) for ${endpoint}, returning empty result`)
        return NextResponse.json(Array.isArray(response.body) ? [] : {})
      }

      return NextResponse.json(
        { error: errorMessage },
        { status: response.status }
      )
    } catch (fetchError) {
      clearTimeout(timeoutId)

      // Обработка сетевых ошибок
      if (fetchError instanceof Error) {
        if (fetchError.name === 'AbortError') {
          const timeoutError = new Error(
            `Превышено время ожидания ответа от сервера (${timeout / 1000} секунд). Сервер может быть перегружен или недоступен.`
          )
          if (onError) {
            return onError(timeoutError)
          }
          return NextResponse.json(
            { error: timeoutError.message },
            { status: 504 }
          )
        }

        if (
          fetchError.message.includes('fetch failed') ||
          fetchError.message.includes('Failed to fetch') ||
          fetchError.message.includes('NetworkError') ||
          fetchError.message.includes('ECONNREFUSED') ||
          fetchError.message.includes('ERR_CONNECTION_REFUSED')
        ) {
          const connectionError = new Error(
            'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.'
          )
          console.error(`Backend connection failed for ${endpoint}:`, fetchError.message)

          if (onError) {
            return onError(connectionError)
          }

          if (returnEmptyOnError) {
            return NextResponse.json([])
          }

          return NextResponse.json(
            { error: connectionError.message },
            { status: 503 }
          )
        }

        if (onError) {
          return onError(fetchError)
        }

        throw fetchError
      }

      throw fetchError
    }
  } catch (error) {
    console.error(`Error in backend proxy for ${endpoint}:`, error)

    const errorMessage = error instanceof Error
      ? error.message
      : 'Internal server error'

    if (onError) {
      return onError(error instanceof Error ? error : new Error(errorMessage))
    }

    if (returnEmptyOnError) {
      return NextResponse.json([])
    }

    return NextResponse.json(
      { error: errorMessage },
      { status: 500 }
    )
  }
}

/**
 * Упрощенная функция для GET запросов
 */
export async function proxyGet(
  endpoint: string,
  request: NextRequest,
  options?: Omit<ProxyOptions, 'method'>
): Promise<NextResponse> {
  return proxyToBackend(endpoint, request, { ...options, method: 'GET' })
}

/**
 * Упрощенная функция для POST запросов
 */
export async function proxyPost(
  endpoint: string,
  request: NextRequest,
  body?: any,
  options?: Omit<ProxyOptions, 'method' | 'body'>
): Promise<NextResponse> {
  return proxyToBackend(endpoint, request, {
    ...options,
    method: 'POST',
    body,
  })
}

/**
 * Создает обработчик для Next.js API Route, который проксирует запросы на Go backend
 * 
 * Это функция-фабрика, которая инкапсулирует всю логику проксирования,
 * что позволяет избежать дублирования кода в route.ts файлах.
 * 
 * @param backendPath - Путь на Go backend (например, '/api/clients')
 * @param options - Опции для проксирования
 * @returns Объект с методами GET, POST, PUT, DELETE, PATCH
 * 
 * @example
 * ```ts
 * // app/api/clients/route.ts
 * import { createProxyHandler } from '@/lib/backend-proxy'
 * 
 * const handler = createProxyHandler('/api/clients', {
 *   requireAuth: true, // Требовать JWT токен
 *   returnEmptyOnError: true, // Возвращать [] при ошибке
 * })
 * 
 * export const GET = handler.GET
 * export const POST = handler.POST
 * ```
 */
export function createProxyHandler(
  backendPath: string,
  options: ProxyOptions = {}
) {
  return {
    GET: async (request: NextRequest) => {
      return proxyToBackend(backendPath, request, { ...options, method: 'GET' })
    },
    POST: async (request: NextRequest) => {
      const body = await request.json().catch(() => null)
      return proxyToBackend(backendPath, request, { ...options, method: 'POST', body })
    },
    PUT: async (request: NextRequest) => {
      const body = await request.json().catch(() => null)
      return proxyToBackend(backendPath, request, { ...options, method: 'PUT', body })
    },
    PATCH: async (request: NextRequest) => {
      const body = await request.json().catch(() => null)
      return proxyToBackend(backendPath, request, { ...options, method: 'PATCH', body })
    },
    DELETE: async (request: NextRequest) => {
      return proxyToBackend(backendPath, request, { ...options, method: 'DELETE' })
    },
  }
}

