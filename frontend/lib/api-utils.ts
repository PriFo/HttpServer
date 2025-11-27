import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from './api-config'

/**
 * Утилиты для работы с API routes
 */

export interface BackendRequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  body?: any
  headers?: Record<string, string>
  cache?: RequestCache
  signal?: AbortSignal
  timeout?: number
}

/**
 * Выполняет запрос к бэкенду с обработкой ошибок
 */
export async function fetchFromBackend(
  endpoint: string,
  options: BackendRequestOptions = {}
): Promise<Response> {
  const {
    method = 'GET',
    body,
    headers = {},
    cache = 'no-store',
    signal,
    timeout = 10000,
  } = options

  const backendUrl = getBackendUrl()
  const url = endpoint.startsWith('http') ? endpoint : `${backendUrl}${endpoint}`

  // Создаем контроллер для таймаута
  const controller = new AbortController()
  const timeoutId = timeout > 0 ? setTimeout(() => controller.abort(), timeout) : null

  // Если передан внешний signal, создаем композитный сигнал
  let finalSignal: AbortSignal = controller.signal
  if (signal) {
    // Если внешний сигнал уже отменен, отменяем и наш контроллер
    if (signal.aborted) {
      controller.abort()
    } else {
      // Слушаем отмену внешнего сигнала
      signal.addEventListener('abort', () => controller.abort())
    }
    // Используем внешний сигнал в fetch
    finalSignal = signal
  }

  try {
    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...headers,
      },
      body: body ? JSON.stringify(body) : undefined,
      cache,
      signal: finalSignal,
    })

    if (timeoutId) clearTimeout(timeoutId)
    return response
  } catch (error: any) {
    if (timeoutId) clearTimeout(timeoutId)
    
    if (error.name === 'AbortError') {
      throw new Error('Request timeout')
    }
    throw error
  }
}

/**
 * Обрабатывает ответ от бэкенда и возвращает NextResponse
 */
export async function handleBackendResponse(
  response: Response,
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

    return NextResponse.json(errorData, { status: response.status })
  }

  const data = await response.json().catch(() => null)
  return NextResponse.json(data)
}

/**
 * Универсальный обработчик для API routes
 */
export async function handleApiRoute(
  request: NextRequest,
  endpoint: string,
  options: BackendRequestOptions & {
    allow404?: boolean
    defaultData?: any
    errorMessage?: string
  } = {}
): Promise<NextResponse> {
  try {
    const { allow404, defaultData, errorMessage, ...fetchOptions } = options

    // Определяем метод из запроса
    const method = (request.method as any) || 'GET'
    
    // Получаем body для POST/PUT/PATCH
    let body: any = undefined
    if (['POST', 'PUT', 'PATCH'].includes(method)) {
      try {
        body = await request.json()
      } catch {
        // Body может быть пустым
      }
    }

    const response = await fetchFromBackend(endpoint, {
      ...fetchOptions,
      method,
      body,
    })

    return handleBackendResponse(response, {
      allow404,
      defaultData,
      errorMessage,
    })
  } catch (error: any) {
    console.error(`Error in API route ${endpoint}:`, error?.message || error)
    return NextResponse.json(
      {
        error: 'Internal server error',
        details: process.env.NODE_ENV === 'development' ? error?.message : undefined,
      },
      { status: 500 }
    )
  }
}

/**
 * Создает NextResponse с ошибкой
 */
export function createErrorResponse(
  message: string,
  status: number = 500,
  details?: string
): NextResponse {
  return NextResponse.json(
    {
      error: message,
      details: process.env.NODE_ENV === 'development' ? details : undefined,
    },
    { status }
  )
}

/**
 * Создает NextResponse с успешным ответом
 */
export function createSuccessResponse(data: any, status: number = 200): NextResponse {
  return NextResponse.json(data, { status })
}

/**
 * Клиентская функция для выполнения API запросов
 * Используется в клиентских компонентах
 */
export async function apiRequest<T = any>(
  endpoint: string,
  options: {
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
    body?: string
    headers?: Record<string, string>
    signal?: AbortSignal
  } = {}
): Promise<T> {
  const { method = 'GET', body, headers = {} } = options

  const url = endpoint.startsWith('http') ? endpoint : endpoint

  try {
    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        ...headers,
      },
      body,
      cache: 'no-store',
      signal: options.signal,
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Request failed' }))
      throw new Error(errorData.error || errorData.message || `HTTP ${response.status}`)
    }

    const data = await response.json().catch(() => null)
    return data as T
  } catch (error: any) {
    if (error.name === 'AbortError') {
      throw new Error('Request timeout')
    }
    throw error
  }
}

/**
 * Форматирует ошибку в понятное сообщение для пользователя
 */
export function formatError(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === 'string') {
    return error
  }
  if (error && typeof error === 'object' && 'message' in error) {
    return String(error.message)
  }
  return 'Произошла неизвестная ошибка'
}

/**
 * Обрабатывает ошибку API запроса
 */
export async function handleApiError(response: Response): Promise<string> {
  try {
    const errorData = await response.json()
    return errorData.error || errorData.message || `HTTP ${response.status}`
  } catch {
    return `HTTP ${response.status}: ${response.statusText}`
  }
}