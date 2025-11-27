/**
 * Утилита для упрощения создания API routes с улучшенной обработкой ошибок и логированием
 * 
 * Предоставляет готовые обертки для стандартных паттернов API routes
 */

import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from './api-config'
import { logger, createApiContext, withLogging } from './logger'
import { handleBackendResponse, handleFetchError, validateRequired, ValidationError } from './error-handler'
import type { LogContext } from './logger'

export interface ApiRouteOptions {
  /**
   * Разрешить возврат дефолтных данных при 404
   */
  allow404?: boolean
  /**
   * Дефолтные данные для возврата при 404 (если allow404 = true)
   */
  defaultData?: unknown
  /**
   * Сообщение об ошибке по умолчанию
   */
  errorMessage?: string
  /**
   * Таймаут запроса в миллисекундах
   */
  timeout?: number
  /**
   * Дополнительные заголовки для запроса
   */
  headers?: Record<string, string>
}

/**
 * Создает GET handler для простого проксирования запроса к бэкенду
 */
export function createGetHandler(
  route: string,
  backendEndpoint: string | ((request: NextRequest, params?: Record<string, string>) => string),
  options: ApiRouteOptions = {}
) {
  return async function GET(
    request: NextRequest,
    { params }: { params?: Promise<Record<string, string>> } = {}
  ) {
    const resolvedParams = params ? await params : {}
    const context = createApiContext(route, 'GET', resolvedParams)
    const startTime = Date.now()

    return withLogging(
      `GET ${route}`,
      async () => {
        const backendUrl = getBackendUrl()
        const endpoint = typeof backendEndpoint === 'function'
          ? backendEndpoint(request, resolvedParams)
          : `${backendUrl}${backendEndpoint}`

        logger.logRequest('GET', route, context)

        const controller = new AbortController()
        const timeoutId = options.timeout
          ? setTimeout(() => controller.abort(), options.timeout)
          : null

        try {
          const response = await fetch(endpoint, {
            method: 'GET',
            cache: 'no-store',
            signal: controller.signal,
            headers: {
              'Accept': 'application/json',
              ...options.headers,
            },
          })

          if (timeoutId) clearTimeout(timeoutId)

          const duration = Date.now() - startTime
          logger.logResponse('GET', route, response.status, duration, context)

          return handleBackendResponse(
            response,
            endpoint,
            context,
            {
              allow404: options.allow404,
              defaultData: options.defaultData,
              errorMessage: options.errorMessage || `Failed to fetch from ${route}`,
            }
          )
        } catch (error) {
          if (timeoutId) clearTimeout(timeoutId)
          const duration = Date.now() - startTime
          return handleFetchError(error, endpoint, { ...context, duration })
        }
      },
      context
    )
  }
}

/**
 * Создает POST handler для проксирования запроса к бэкенду
 */
export function createPostHandler(
  route: string,
  backendEndpoint: string | ((request: NextRequest, params?: Record<string, string>) => string),
  options: ApiRouteOptions = {}
) {
  return async function POST(
    request: NextRequest,
    { params }: { params?: Promise<Record<string, string>> } = {}
  ) {
    const resolvedParams = params ? await params : {}
    const context = createApiContext(route, 'POST', resolvedParams)
    const startTime = Date.now()

    return withLogging(
      `POST ${route}`,
      async () => {
        let body: unknown = {}
        try {
          body = await request.json()
        } catch {
          // Body может быть пустым
        }

        const backendUrl = getBackendUrl()
        const endpoint = typeof backendEndpoint === 'function'
          ? backendEndpoint(request, resolvedParams)
          : `${backendUrl}${backendEndpoint}`

        logger.logRequest('POST', route, { ...context, hasBody: !!body })

        const controller = new AbortController()
        const timeoutId = options.timeout
          ? setTimeout(() => controller.abort(), options.timeout)
          : null

        try {
          const response = await fetch(endpoint, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              ...options.headers,
            },
            body: JSON.stringify(body),
            cache: 'no-store',
            signal: controller.signal,
          })

          if (timeoutId) clearTimeout(timeoutId)

          const duration = Date.now() - startTime
          logger.logResponse('POST', route, response.status, duration, context)

          return handleBackendResponse(
            response,
            endpoint,
            context,
            {
              allow404: options.allow404,
              defaultData: options.defaultData,
              errorMessage: options.errorMessage || `Failed to post to ${route}`,
            }
          )
        } catch (error) {
          if (timeoutId) clearTimeout(timeoutId)
          const duration = Date.now() - startTime
          return handleFetchError(error, endpoint, { ...context, duration })
        }
      },
      context
    )
  }
}

/**
 * Создает PUT handler
 */
export function createPutHandler(
  route: string,
  backendEndpoint: string | ((request: NextRequest, params?: Record<string, string>) => string),
  options: ApiRouteOptions = {}
) {
  return async function PUT(
    request: NextRequest,
    { params }: { params?: Promise<Record<string, string>> } = {}
  ) {
    const resolvedParams = params ? await params : {}
    const context = createApiContext(route, 'PUT', resolvedParams)
    const startTime = Date.now()

    return withLogging(
      `PUT ${route}`,
      async () => {
        const body = await request.json().catch(() => ({}))
        const backendUrl = getBackendUrl()
        const endpoint = typeof backendEndpoint === 'function'
          ? backendEndpoint(request, resolvedParams)
          : `${backendUrl}${backendEndpoint}`

        logger.logRequest('PUT', route, context)

        try {
          const response = await fetch(endpoint, {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              ...options.headers,
            },
            body: JSON.stringify(body),
            cache: 'no-store',
          })

          const duration = Date.now() - startTime
          logger.logResponse('PUT', route, response.status, duration, context)

          return handleBackendResponse(
            response,
            endpoint,
            context,
            {
              errorMessage: options.errorMessage || `Failed to update ${route}`,
            }
          )
        } catch (error) {
          const duration = Date.now() - startTime
          return handleFetchError(error, endpoint, { ...context, duration })
        }
      },
      context
    )
  }
}

/**
 * Создает DELETE handler
 */
export function createDeleteHandler(
  route: string,
  backendEndpoint: string | ((request: NextRequest, params?: Record<string, string>) => string),
  options: ApiRouteOptions = {}
) {
  return async function DELETE(
    request: NextRequest,
    { params }: { params?: Promise<Record<string, string>> } = {}
  ) {
    const resolvedParams = params ? await params : {}
    const context = createApiContext(route, 'DELETE', resolvedParams)
    const startTime = Date.now()

    return withLogging(
      `DELETE ${route}`,
      async () => {
        const backendUrl = getBackendUrl()
        const endpoint = typeof backendEndpoint === 'function'
          ? backendEndpoint(request, resolvedParams)
          : `${backendUrl}${backendEndpoint}`

        logger.logRequest('DELETE', route, context)

        try {
          const response = await fetch(endpoint, {
            method: 'DELETE',
            headers: {
              'Content-Type': 'application/json',
              ...options.headers,
            },
            cache: 'no-store',
          })

          const duration = Date.now() - startTime
          logger.logResponse('DELETE', route, response.status, duration, context)

          return handleBackendResponse(
            response,
            endpoint,
            context,
            {
              errorMessage: options.errorMessage || `Failed to delete ${route}`,
            }
          )
        } catch (error) {
          const duration = Date.now() - startTime
          return handleFetchError(error, endpoint, { ...context, duration })
        }
      },
      context
    )
  }
}

/**
 * Валидирует query параметры
 */
export function validateQueryParams(
  request: NextRequest,
  requiredParams: string[],
  context?: LogContext
): Record<string, string> {
  const searchParams = request.nextUrl.searchParams
  const params: Record<string, string> = {}

  for (const param of requiredParams) {
    const value = searchParams.get(param)
    validateRequired(value, param, context)
    params[param] = value!
  }

  return params
}

/**
 * Валидирует path параметры
 */
export async function validatePathParams(
  params: Promise<Record<string, string>> | Record<string, string>,
  requiredParams: string[],
  context?: LogContext
): Promise<Record<string, string>> {
  const resolvedParams = params instanceof Promise ? await params : params
  const validated: Record<string, string> = {}

  for (const param of requiredParams) {
    const value = resolvedParams[param]
    validateRequired(value, param, context)
    validated[param] = value
  }

  return validated
}

